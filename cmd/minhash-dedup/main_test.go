package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestPairwiseOutputFormat(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := Run([]string{"--left", "../../sample/a.txt", "--right", "../../sample/b.txt", "--k", "64"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("退出码应为 0：%d（%s）", code, stderr.String())
	}
	var estimate, exact, diff float64
	var k int
	if _, err := fmt.Sscanf(stdout.String(), "estimate=%f exact=%f diff=%f k=%d", &estimate, &exact, &diff, &k); err != nil {
		t.Fatalf("输出格式变了：%q", stdout.String())
	}
	if exact < 0.4285 || exact > 0.4287 {
		t.Fatalf("精确值应为 0.4286：%v", exact)
	}
	if diff > 0.05 {
		t.Fatalf("k=64 时 diff 应 <= 0.05：%v", diff)
	}
	if k != 64 {
		t.Fatalf("k 应为 64：%d", k)
	}
}

func TestPairwiseRequiresBothSides(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := Run(nil, stdout, stderr); code != 2 {
		t.Fatalf("缺少参数应返回 2：%d", code)
	}
}

func TestClusterModeGenerate(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := Run([]string{"--generate", "50", "--threshold", "0.8"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("退出码应为 0：%d（%s）", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "docs=50") || !strings.Contains(out, "candidates=") {
		t.Fatalf("聚类模式应输出候选统计：%q", out)
	}
	if !strings.Contains(out, "clusters=") || !strings.Contains(out, "cluster 1 (size") {
		t.Fatalf("聚类模式应输出簇数与成员：%q", out)
	}
}

func TestClusterModeFiles(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := Run([]string{"--threshold", "0.4", "../../sample/a.txt", "../../sample/b.txt"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("退出码应为 0：%d（%s）", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "clusters=1") {
		t.Fatalf("样例两篇文档应聚成一簇：%q", stdout.String())
	}
}

func TestClusterModeBadBands(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := Run([]string{"--generate", "10", "--bands", "3", "--rows", "3"}, stdout, stderr); code != 2 {
		t.Fatalf("bands*rows != k 应返回 2：%d", code)
	}
	if code := Run([]string{"--generate", "10", "--bands", "8"}, stdout, stderr); code != 2 {
		t.Fatalf("只给 --bands 应返回 2：%d", code)
	}
	if code := Run([]string{"--generate", "10", "--threshold", "1.5"}, stdout, stderr); code != 2 {
		t.Fatalf("threshold 越界应返回 2：%d", code)
	}
}
