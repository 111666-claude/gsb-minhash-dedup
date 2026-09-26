// Command minhash-dedup 估计两批数据的相似度，或按阈值做近重复聚类。
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"example.com/minhash"
)

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("minhash-dedup", flag.ContinueOnError)
	flags.SetOutput(stderr)
	leftPath := flags.String("left", "", "左边文件，一行一个元素")
	rightPath := flags.String("right", "", "右边文件")
	k := flags.Int("k", 64, "签名长度（哈希函数个数），1..1024")
	threshold := flags.Float64("threshold", -1, "聚类阈值 T（0 到 1），设置后进入聚类模式")
	bands := flags.Int("bands", 0, "LSH 分带数 B（与 --rows 一起给，B*R=k）")
	rows := flags.Int("rows", 0, "每个分带的行数 R（B*R=k）")
	generate := flags.Int("generate", 0, "生成 N 篇确定性合成文档用于聚类")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *k < 1 || *k > 1024 {
		fmt.Fprintln(stderr, "k 必须在 1 到 1024 之间")
		return 2
	}
	clusterMode := *threshold >= 0 || *generate > 0 || flags.NArg() > 0
	if !clusterMode {
		return runPair(*leftPath, *rightPath, *k, stdout, stderr)
	}
	return runCluster(*threshold, *generate, *k, *bands, *rows, flags.Args(), stdout, stderr)
}

// readItems 读取一个文档文件：一行一个元素，忽略空行与 # 注释。
func readItems(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	items := []string{}
	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		items = append(items, line)
	}
	return items, nil
}

// runPair 两文件比较模式：输出与旧版一致的 estimate/exact/diff/k。
func runPair(leftPath, rightPath string, k int, stdout, stderr io.Writer) int {
	if leftPath == "" || rightPath == "" {
		fmt.Fprintln(stderr, "用法：minhash-dedup --left sample/a.txt --right sample/b.txt --k 64")
		return 2
	}
	left, err := readItems(leftPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	right, err := readItems(rightPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	leftSet, rightSet := map[string]bool{}, map[string]bool{}
	for _, item := range left {
		leftSet[item] = true
	}
	for _, item := range right {
		rightSet[item] = true
	}
	estimate := minhash.Similarity(minhash.Signature(left, k), minhash.Signature(right, k))
	exact := minhash.ExactJaccard(leftSet, rightSet)
	fmt.Fprintf(stdout, "estimate=%.4f exact=%.4f diff=%.4f k=%d\n", estimate, exact, math.Abs(estimate-exact), k)
	return 0
}

// runCluster 聚类模式：LSH 分带筛候选对，再把估计相似度 >= T 的文档并查集聚簇。
func runCluster(threshold float64, generate, k, bands, rows int, files []string, stdout, stderr io.Writer) int {
	if threshold < 0 {
		threshold = 0.5
	}
	if threshold > 1 {
		fmt.Fprintln(stderr, "threshold 必须在 0 到 1 之间")
		return 2
	}
	if (bands == 0) != (rows == 0) {
		fmt.Fprintln(stderr, "--bands 与 --rows 要一起给")
		return 2
	}
	if bands == 0 {
		bands, rows = minhash.ChooseBands(k, threshold)
	} else if bands*rows != k {
		fmt.Fprintf(stderr, "bands*rows 必须等于 k：%d*%d != %d\n", bands, rows, k)
		return 2
	}
	names := []string{}
	docs := [][]string{}
	for _, path := range files {
		items, err := readItems(path)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		names = append(names, path)
		docs = append(docs, items)
	}
	for index, doc := range minhash.GenerateDocs(generate, 1) {
		names = append(names, fmt.Sprintf("gen-%04d", index))
		docs = append(docs, doc)
	}
	if len(docs) == 0 {
		fmt.Fprintln(stderr, "用法：minhash-dedup --threshold T [--generate N | 文件 ...] [--k 64] [--bands B --rows R]")
		return 2
	}
	signatures := make([][]uint64, len(docs))
	for index, doc := range docs {
		signatures[index] = minhash.Signature(doc, k)
	}
	var candidates [][2]int
	if threshold <= 0 {
		for i := 0; i < len(docs); i++ {
			for j := i + 1; j < len(docs); j++ {
				candidates = append(candidates, [2]int{i, j})
			}
		}
	} else {
		candidates = minhash.Candidates(signatures, bands, rows)
	}
	clusters := minhash.Cluster(signatures, threshold, candidates)
	fmt.Fprintf(stdout, "docs=%d k=%d threshold=%.4f bands=%d rows=%d candidates=%d clusters=%d\n",
		len(docs), k, threshold, bands, rows, len(candidates), len(clusters))
	for index, members := range clusters {
		fmt.Fprintf(stdout, "cluster %d size=%d:", index, len(members))
		for _, member := range members {
			fmt.Fprintf(stdout, " %s", names[member])
		}
		fmt.Fprintln(stdout)
	}
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
