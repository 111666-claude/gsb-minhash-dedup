package minhash

import (
	"math"
	"math/rand"
	"testing"
)

var (
	sampleLeft  = []string{"alpha", "bravo", "charlie", "delta", "echo"}
	sampleRight = []string{"charlie", "delta", "echo", "foxtrot", "golf"}
)

func toSet(items []string) map[string]bool {
	set := map[string]bool{}
	for _, item := range items {
		set[item] = true
	}
	return set
}

func randomSets(rng *rand.Rand, count int) ([]string, []string) {
	shared := rng.Intn(60)
	leftOnly := rng.Intn(60)
	rightOnly := rng.Intn(60)
	left := make([]string, 0, shared+leftOnly)
	right := make([]string, 0, shared+rightOnly)
	for i := 0; i < shared; i++ {
		item := string(rune('A'+rng.Intn(26))) + string(rune('a'+rng.Intn(26))) + string(rune('0'+rng.Intn(10))) + string(rune('0'+rng.Intn(100)))
		left = append(left, item)
		right = append(right, item)
	}
	for i := 0; i < leftOnly; i++ {
		left = append(left, string(rune('L'+rune(rng.Intn(26))))+string(rune('0'+rng.Intn(1000))))
	}
	for i := 0; i < rightOnly; i++ {
		right = append(right, string(rune('R'+rune(rng.Intn(26))))+string(rune('0'+rng.Intn(1000))))
	}
	return left, right
}

func TestSampleAccuracyK64(t *testing.T) {
	estimate := Similarity(Signature(sampleLeft, 64), Signature(sampleRight, 64))
	exact := ExactJaccard(toSet(sampleLeft), toSet(sampleRight))
	if math.Abs(exact-3.0/7.0) > 1e-9 {
		t.Fatalf("样例精确值应为 3/7：%v", exact)
	}
	if diff := math.Abs(estimate - exact); diff > 0.05 {
		t.Fatalf("k=64 时估计与精确差应 <= 0.05：estimate=%v exact=%v diff=%v", estimate, exact, diff)
	}
}

func TestSignatureUsesDistinctPositions(t *testing.T) {
	signature := Signature([]string{"a", "b", "c", "d", "e"}, 64)
	distinct := map[uint64]bool{}
	for _, value := range signature {
		distinct[value] = true
	}
	if len(distinct) < 2 {
		t.Fatal("k 个位置不应全部填同一个最小值")
	}
}

func TestSignatureOrderInsensitive(t *testing.T) {
	forward := Signature([]string{"a", "b", "c", "d"}, 32)
	backward := Signature([]string{"d", "c", "b", "a"}, 32)
	for index := range forward {
		if forward[index] != backward[index] {
			t.Fatal("签名应对元素顺序不敏感（集合语义）")
		}
	}
}

func TestSignatureKRange(t *testing.T) {
	items := []string{"x", "y", "z"}
	for _, k := range []int{1, 2, 3, 5, 7, 31, 64, 255, 256, 511, 1000, 1024} {
		signature := Signature(items, k)
		if len(signature) != k {
			t.Fatalf("k=%d 时签名长度不对：%d", k, len(signature))
		}
		if got := Similarity(signature, signature); got != 1 {
			t.Fatalf("k=%d 时相同集合相似度应为 1：%v", k, got)
		}
	}
}

func TestDisjointSimilarityZero(t *testing.T) {
	left := Signature([]string{"a1", "a2", "a3"}, 64)
	right := Signature([]string{"b1", "b2", "b3"}, 64)
	if got := Similarity(left, right); got != 0 {
		t.Fatalf("无交集集合相似度应为 0：%v", got)
	}
}

func TestEmptyInputsNoPanic(t *testing.T) {
	empty := Signature(nil, 8)
	if len(empty) != 8 {
		t.Fatalf("空输入也要给出长度：%d", len(empty))
	}
	if got := Similarity(Signature(nil, 8), Signature(nil, 8)); got != 1 {
		t.Fatalf("两个空集合的签名应完全一致：%v", got)
	}
	if got := Similarity(nil, nil); got != 0 {
		t.Fatalf("空签名相似度应为 0：%v", got)
	}
	if merged := Merge(nil, nil); len(merged) != 0 {
		t.Fatalf("空签名合并应为空：%d", len(merged))
	}
}

func TestMergeMatchesUnion(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for round := 0; round < 1000; round++ {
		left, right := randomSets(rng, round)
		k := 1 + rng.Intn(128)
		merged := Merge(Signature(left, k), Signature(right, k))
		union := Signature(append(append([]string{}, left...), right...), k)
		if len(merged) != len(union) {
			t.Fatalf("round %d：合并长度不对", round)
		}
		for index := range merged {
			if merged[index] != union[index] {
				t.Fatalf("round %d 位置 %d：Merge(签名) != Signature(并集)", round, index)
			}
		}
	}
}

func TestMergeCommutativeAssociative(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	for round := 0; round < 200; round++ {
		a, b := randomSets(rng, round)
		c, _ := randomSets(rng, round+1)
		k := 1 + rng.Intn(64)
		sa, sb, sc := Signature(a, k), Signature(b, k), Signature(c, k)
		ab, ba := Merge(sa, sb), Merge(sb, sa)
		for index := range ab {
			if ab[index] != ba[index] {
				t.Fatal("Merge 应满足交换律")
			}
		}
		left := Merge(Merge(sa, sb), sc)
		right := Merge(sa, Merge(sb, sc))
		for index := range left {
			if left[index] != right[index] {
				t.Fatal("Merge 应满足结合律")
			}
		}
	}
}

func TestEstimateVsExactRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(13))
	const k = 512
	maxDiff := 0.0
	totalDiff := 0.0
	const rounds = 1000
	for round := 0; round < rounds; round++ {
		left, right := randomSets(rng, round)
		estimate := Similarity(Signature(left, k), Signature(right, k))
		exact := ExactJaccard(toSet(left), toSet(right))
		diff := math.Abs(estimate - exact)
		if diff > maxDiff {
			maxDiff = diff
		}
		totalDiff += diff
	}
	if mean := totalDiff / rounds; mean > 0.02 {
		t.Fatalf("k=%d 时平均误差应 <= 0.02：%v", k, mean)
	}
	if maxDiff > 0.1 {
		t.Fatalf("k=%d 时最大误差应 <= 0.1：%v", k, maxDiff)
	}
}
