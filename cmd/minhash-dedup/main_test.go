package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func runToString(args ...string) (string, string, int) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := Run(args, stdout, stderr)
	return stdout.String(), stderr.String(), code
}

func TestPairOutputFormat(t *testing.T) {
	stdout, _, code := runToString("--left", "../../sample/a.txt", "--right", "../../sample/b.txt", "--k", "64")
	if code != 0 {
		t.Fatalf("退出码应为 0：%d", code)
	}
	pattern := regexp.MustCompile(`^estimate=(\d\.\d{4}) exact=0\.4286 diff=(\d\.\d{4}) k=64\n$`)
	matches := pattern.FindStringSubmatch(stdout)
	if matches == nil {
		t.Fatalf("默认输出格式应与旧版一致：%q", stdout)
	}
	var estimate, diff float64
	fmt.Sscanf(matches[1], "%f", &estimate)
	fmt.Sscanf(matches[2], "%f", &diff)
	if diff > 0.05 {
		t.Fatalf("k=64 时 diff 必须 <= 0.05：estimate=%v diff=%v", estimate, diff)
	}
}

func TestPairRequiresBothFiles(t *testing.T) {
	if _, _, code := runToString("--left", "../../sample/a.txt"); code != 2 {
		t.Fatalf("缺少参数应返回 2")
	}
}

func TestClusterGenerate(t *testing.T) {
	stdout, _, code := runToString("--generate", "2000", "--threshold", "0.8")
	if code != 0 {
		t.Fatalf("退出码应为 0：%d", code)
	}
	var docs, k, bands, rows, candidates, clusters int
	var threshold float64
	firstLine := strings.SplitN(stdout, "\n", 2)[0]
	_, err := fmt.Sscanf(firstLine, "docs=%d k=%d threshold=%f bands=%d rows=%d candidates=%d clusters=%d",
		&docs, &k, &threshold, &bands, &rows, &candidates, &clusters)
	if err != nil {
		t.Fatalf("聚类汇总行格式不对：%q", firstLine)
	}
	if docs != 2000 || k != 64 || bands*rows != k {
		t.Fatalf("汇总行参数不对：%q", firstLine)
	}
	totalPairs := docs * (docs - 1) / 2
	if candidates*20 >= totalPairs {
		t.Fatalf("候选对数不许退化成两两比较：candidates=%d pairs=%d", candidates, totalPairs)
	}
	if !strings.Contains(stdout, "cluster 0 size=") {
		t.Fatalf("应输出每簇成员：%q", stdout[:200])
	}
}

func TestClusterDeterministic(t *testing.T) {
	first, _, _ := runToString("--generate", "300", "--threshold", "0.8")
	second, _, _ := runToString("--generate", "300", "--threshold", "0.8")
	if first != second {
		t.Fatal("同一输入重复执行结果必须相同")
	}
}

func TestClusterExplicitBands(t *testing.T) {
	stdout, _, code := runToString("--generate", "200", "--threshold", "0.8", "--bands", "16", "--rows", "4")
	if code != 0 || !strings.Contains(stdout, "bands=16 rows=4") {
		t.Fatalf("显式分带应生效：code=%d out=%q", code, stdout)
	}
	if _, _, code := runToString("--generate", "10", "--threshold", "0.8", "--bands", "16", "--rows", "5"); code != 2 {
		t.Fatal("B*R != k 应报错")
	}
}

func TestClusterFiles(t *testing.T) {
	stdout, _, code := runToString("--threshold", "0.5", "../../sample/a.txt", "../../sample/b.txt")
	if code != 0 {
		t.Fatalf("退出码应为 0：%d", code)
	}
	if !strings.Contains(stdout, "docs=2") || !strings.Contains(stdout, "clusters=") {
		t.Fatalf("文件聚类输出不对：%q", stdout)
	}
}

func TestInvalidK(t *testing.T) {
	if _, _, code := runToString("--left", "../../sample/a.txt", "--right", "../../sample/b.txt", "--k", "0"); code != 2 {
		t.Fatal("k=0 应报错")
	}
	if _, _, code := runToString("--left", "../../sample/a.txt", "--right", "../../sample/b.txt", "--k", "1025"); code != 2 {
		t.Fatal("k=1025 应报错")
	}
}
