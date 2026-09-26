// Command minhash-dedup 估计两批数据的相似度，或把多篇文档按阈值聚类。
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

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("minhash-dedup", flag.ContinueOnError)
	flags.SetOutput(stderr)
	leftPath := flags.String("left", "", "左边文件，一行一个元素")
	rightPath := flags.String("right", "", "右边文件")
	k := flags.Int("k", 64, "签名长度（哈希函数个数）")
	threshold := flags.Float64("threshold", -1, "聚类阈值 T（0 到 1）：相似度 >= T 的文档聚成一簇")
	bands := flags.Int("bands", 0, "LSH 分带数（需与 --rows 一起指定，bands*rows = k）")
	rows := flags.Int("rows", 0, "每条带的行数（需与 --bands 一起指定）")
	generate := flags.Int("generate", 0, "生成 N 篇合成文档并聚类")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *k < 1 {
		fmt.Fprintln(stderr, "k 必须 >= 1")
		return 2
	}
	if *threshold >= 0 || *generate > 0 || flags.NArg() > 0 {
		return runCluster(stdout, stderr, *k, *threshold, *bands, *rows, *generate, flags.Args())
	}
	return runPairwise(stdout, stderr, *leftPath, *rightPath, *k)
}

func runPairwise(stdout, stderr io.Writer, leftPath, rightPath string, k int) int {
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

func runCluster(stdout, stderr io.Writer, k int, threshold float64, bands, rows, generate int, files []string) int {
	if threshold < 0 {
		threshold = 0.8
	}
	if threshold > 1 {
		fmt.Fprintln(stderr, "threshold 必须在 0 到 1 之间")
		return 2
	}
	if (bands == 0) != (rows == 0) {
		fmt.Fprintln(stderr, "--bands 与 --rows 需要同时指定")
		return 2
	}
	if bands == 0 {
		bands, rows = minhash.PickBands(k, threshold)
	}
	if bands < 1 || rows < 1 || bands*rows != k {
		fmt.Fprintf(stderr, "需要 bands*rows = k（bands=%d rows=%d k=%d）\n", bands, rows, k)
		return 2
	}
	docs := [][]string{}
	names := []string{}
	for i, doc := range minhash.Generate(generate, 1) {
		docs = append(docs, doc)
		names = append(names, fmt.Sprintf("gen-%04d", i))
	}
	for _, path := range files {
		items, err := readItems(path)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		docs = append(docs, items)
		names = append(names, path)
	}
	if len(docs) == 0 {
		fmt.Fprintln(stderr, "用法：minhash-dedup --threshold 0.8 [--generate N | 文件...] [--k 64] [--bands B --rows R]")
		return 2
	}
	signatures := make([][]uint64, len(docs))
	for i, doc := range docs {
		signatures[i] = minhash.Signature(doc, k)
	}
	candidates := minhash.Candidates(signatures, bands, rows)
	clusters := minhash.Cluster(signatures, threshold, bands, rows)
	fmt.Fprintf(stdout, "docs=%d k=%d bands=%d rows=%d threshold=%.2f candidates=%d\n",
		len(docs), k, bands, rows, threshold, len(candidates))
	fmt.Fprintf(stdout, "clusters=%d\n", len(clusters))
	for i, members := range clusters {
		fmt.Fprintf(stdout, "cluster %d (size %d):", i+1, len(members))
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
