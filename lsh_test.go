package minhash

import (
	"reflect"
	"testing"
)

func signaturesOf(docs [][]string, k int) [][]uint64 {
	signatures := make([][]uint64, len(docs))
	for i, doc := range docs {
		signatures[i] = Signature(doc, k)
	}
	return signatures
}

func TestPickBandsProductAndPigeonhole(t *testing.T) {
	for _, k := range []int{1, 2, 4, 16, 64, 100, 256, 1024} {
		for _, threshold := range []float64{0.1, 0.5, 0.8, 0.9, 0.99, 1.0} {
			bands, rows := PickBands(k, threshold)
			if bands*rows != k {
				t.Fatalf("k=%d T=%v：bands*rows != k（%d*%d）", k, threshold, bands, rows)
			}
			if threshold > 0 && threshold < 1 && float64(bands) <= (1-threshold)*float64(k) {
				t.Fatalf("k=%d T=%v：鸽巢条件 bands > (1-T)*k 不成立（bands=%d）", k, threshold, bands)
			}
		}
	}
}

func TestCandidatesRecallComplete(t *testing.T) {
	docs := Generate(300, 3)
	for _, k := range []int{16, 64, 256} {
		for _, threshold := range []float64{0.5, 0.8, 0.9} {
			signatures := signaturesOf(docs, k)
			bands, rows := PickBands(k, threshold)
			candidates := Candidates(signatures, bands, rows)
			inCandidates := make(map[[2]int]bool, len(candidates))
			for _, pair := range candidates {
				inCandidates[pair] = true
			}
			for i := 0; i < len(signatures); i++ {
				for j := i + 1; j < len(signatures); j++ {
					if Similarity(signatures[i], signatures[j]) >= threshold && !inCandidates[[2]int{i, j}] {
						t.Fatalf("k=%d T=%v：相似度达标的对 (%d,%d) 未进入候选", k, threshold, i, j)
					}
				}
			}
		}
	}
}

func TestClusterMatchesReference(t *testing.T) {
	docs := Generate(200, 5)
	for _, k := range []int{16, 64} {
		for _, threshold := range []float64{0.5, 0.8, 0.9} {
			signatures := signaturesOf(docs, k)
			bands, rows := PickBands(k, threshold)
			got := Cluster(signatures, threshold, bands, rows)
			want := ClusterReference(signatures, threshold)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("k=%d T=%v：分带聚类与全量两两参考不一致", k, threshold)
			}
		}
	}
}

func TestClusterDeterministic(t *testing.T) {
	docs := Generate(100, 9)
	signatures := signaturesOf(docs, 64)
	bands, rows := PickBands(64, 0.8)
	first := Cluster(signatures, 0.8, bands, rows)
	second := Cluster(signatures, 0.8, bands, rows)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("同一输入重复聚类结果应相同")
	}
}

func TestCandidatesFarBelowQuadratic(t *testing.T) {
	const n = 2000
	docs := Generate(n, 1)
	signatures := signaturesOf(docs, 64)
	bands, rows := PickBands(64, 0.8)
	candidates := Candidates(signatures, bands, rows)
	if limit := n * n / 100; len(candidates) > limit {
		t.Fatalf("候选对数应远小于 N^2：candidates=%d limit=%d", len(candidates), limit)
	}
}

func TestClusterEdgeCases(t *testing.T) {
	if clusters := Cluster(nil, 0.8, 1, 1); len(clusters) != 0 {
		t.Fatalf("空输入应给出 0 个簇：%d", len(clusters))
	}
	docs := Generate(10, 2)
	signatures := signaturesOf(docs, 64)
	if clusters := Cluster(signatures, 0, 1, 64); len(clusters) != 1 {
		t.Fatalf("threshold=0 时所有文档应聚成一簇：%d", len(clusters))
	}
	if clusters := Cluster(signatures, 1, 1, 64); len(clusters) != len(signatures) {
		t.Fatalf("threshold=1 时不同文档应各自成簇：%d", len(clusters))
	}
}
