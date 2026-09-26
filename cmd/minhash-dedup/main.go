// Command minhash-dedup 估计两批数据的相似度。
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
	k := flags.Int("k", 64, "签名长度（哈希函数个数）")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *leftPath == "" || *rightPath == "" {
		fmt.Fprintln(stderr, "用法：minhash-dedup --left sample/a.txt --right sample/b.txt --k 64")
		return 2
	}
	read := func(path string) ([]string, error) {
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
	left, err := read(*leftPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	right, err := read(*rightPath)
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
	estimate := minhash.Similarity(minhash.Signature(left, *k), minhash.Signature(right, *k))
	exact := minhash.ExactJaccard(leftSet, rightSet)
	fmt.Fprintf(stdout, "estimate=%.4f exact=%.4f diff=%.4f k=%d\n", estimate, exact, math.Abs(estimate-exact), *k)
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
