package minhash

import (
	"bufio"
	"math"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"testing"
)

func readSample(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
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
	return items
}

func toSet(items []string) map[string]bool {
	set := map[string]bool{}
	for _, item := range items {
		set[item] = true
	}
	return set
}

func TestSignatureLength(t *testing.T) {
	signature := Signature([]string{"a", "b", "c"}, 8)
	if len(signature) != 8 {
		t.Fatalf("签名长度不对：%d", len(signature))
	}
}

func TestSignatureStable(t *testing.T) {
	first := Signature([]string{"a", "b"}, 4)
	second := Signature([]string{"a", "b"}, 4)
	for index := range first {
		if first[index] != second[index] {
			t.Fatal("同一输入签名应该稳定")
		}
	}
}

func TestSignatureEmptyInput(t *testing.T) {
	signature := Signature(nil, 4)
	if len(signature) != 4 {
		t.Fatalf("空输入也要给出长度：%d", len(signature))
	}
}

func TestSignatureDistinctPositions(t *testing.T) {
	signature := Signature([]string{"a", "b", "c", "d", "e"}, 64)
	seen := map[uint64]int{}
	for _, value := range signature {
		seen[value]++
	}
	if len(seen) < 60 {
		t.Fatalf("k 个位置应使用不同哈希，几乎不该重复：%d 个不同值", len(seen))
	}
}

func TestSignatureOrderInsensitive(t *testing.T) {
	rng := rand.New(rand.NewSource(9))
	base := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	want := Signature(base, 64)
	for trial := 0; trial < 20; trial++ {
		shuffled := append([]string(nil), base...)
		rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		if got := Signature(shuffled, 64); !reflect.DeepEqual(want, got) {
			t.Fatal("集合语义下签名不应依赖元素顺序")
		}
	}
}

func TestSignatureDuplicateInsensitive(t *testing.T) {
	single := Signature([]string{"a", "b", "c"}, 32)
	double := Signature([]string{"a", "a", "b", "c", "c", "c"}, 32)
	if !reflect.DeepEqual(single, double) {
		t.Fatal("重复元素不应改变签名")
	}
}

func TestExactJaccardIdentical(t *testing.T) {
	set := map[string]bool{"a": true, "b": true}
	if got := ExactJaccard(set, set); got != 1 {
		t.Fatalf("完全相同的集合相似度应为 1：%v", got)
	}
}

func TestExactJaccardDisjoint(t *testing.T) {
	left := map[string]bool{"a": true}
	right := map[string]bool{"b": true}
	if got := ExactJaccard(left, right); got != 0 {
		t.Fatalf("不相交集合相似度应为 0：%v", got)
	}
}

func TestExactJaccardHalf(t *testing.T) {
	left := map[string]bool{"a": true, "b": true}
	right := map[string]bool{"b": true, "c": true}
	if got := ExactJaccard(left, right); got < 0.32 || got > 0.34 {
		t.Fatalf("交集 1、并集 3 应为 1/3：%v", got)
	}
}

func TestMergeLength(t *testing.T) {
	merged := Merge([]uint64{1, 2}, []uint64{3, 4})
	if len(merged) != 2 {
		t.Fatalf("合并后长度不对：%d", len(merged))
	}
}

func TestSimilaritySameSignature(t *testing.T) {
	signature := Signature([]string{"a", "b"}, 4)
	if got := Similarity(signature, signature); got != 1 {
		t.Fatalf("同一签名相似度应为 1：%v", got)
	}
}

func TestSamplePrecision(t *testing.T) {
	left := readSample(t, "sample/a.txt")
	right := readSample(t, "sample/b.txt")
	estimate := Similarity(Signature(left, 64), Signature(right, 64))
	exact := ExactJaccard(toSet(left), toSet(right))
	if math.Abs(exact-3.0/7.0) > 1e-9 {
		t.Fatalf("样例精确值应为 3/7：%v", exact)
	}
	if diff := math.Abs(estimate - exact); diff > 0.05 {
		t.Fatalf("k=64 时估计与精确差应 <= 0.05：estimate=%v exact=%v diff=%v", estimate, exact, diff)
	}
}

func TestInvariants(t *testing.T) {
	for _, k := range []int{1, 2, 3, 63, 256, 1024} {
		items := []string{"x", "y", "z"}
		signature := Signature(items, k)
		if len(signature) != k {
			t.Fatalf("k=%d 签名长度不对：%d", k, len(signature))
		}
		if got := Similarity(signature, Signature(items, k)); got != 1 {
			t.Fatalf("k=%d 完全相同集合相似度应为 1：%v", k, got)
		}
	}
	left := Signature([]string{"a", "b", "c"}, 128)
	right := Signature([]string{"d", "e", "f"}, 128)
	if got := Similarity(left, right); got != 0 {
		t.Fatalf("无交集集合相似度应为 0：%v", got)
	}
	empty := Signature(nil, 16)
	if got := Similarity(empty, empty); got != 1 {
		t.Fatalf("两个空集合应相似度为 1：%v", got)
	}
	if got := Similarity(empty, left); got != 0 {
		t.Fatalf("空集合与非空集合相似度应为 0：%v", got)
	}
	if got := Similarity(nil, nil); got != 0 {
		t.Fatalf("空签名不应 panic，相似度为 0：%v", got)
	}
}

func TestEstimateVsExactRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	const k = 128
	totalDiff := 0.0
	for trial := 0; trial < 1000; trial++ {
		universe := 200
		leftSize := rng.Intn(60)
		rightSize := rng.Intn(60)
		leftSet, rightSet := map[string]bool{}, map[string]bool{}
		for len(leftSet) < leftSize {
			leftSet[string(rune('A'+rune(rng.Intn(universe))))] = true
		}
		for len(rightSet) < rightSize {
			rightSet[string(rune('A'+rune(rng.Intn(universe))))] = true
		}
		leftItems, rightItems := []string{}, []string{}
		for item := range leftSet {
			leftItems = append(leftItems, item)
		}
		for item := range rightSet {
			rightItems = append(rightItems, item)
		}
		estimate := Similarity(Signature(leftItems, k), Signature(rightItems, k))
		exact := ExactJaccard(leftSet, rightSet)
		diff := math.Abs(estimate - exact)
		if diff > 0.15 {
			t.Fatalf("第 %d 组差值过大：estimate=%v exact=%v", trial, estimate, exact)
		}
		totalDiff += diff
	}
	if mean := totalDiff / 1000; mean > 0.05 {
		t.Fatalf("平均差值应 <= 0.05：%v", mean)
	}
}

func TestMergeMatchesUnion(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	const k = 96
	for trial := 0; trial < 200; trial++ {
		makeSet := func(size int) []string {
			items := []string{}
			for index := 0; index < size; index++ {
				items = append(items, string(rune('a'+rune(rng.Intn(80)))))
			}
			return items
		}
		left, right, third := makeSet(rng.Intn(40)), makeSet(rng.Intn(40)), makeSet(rng.Intn(40))
		leftSig, rightSig, thirdSig := Signature(left, k), Signature(right, k), Signature(third, k)
		unionSig := Signature(append(append([]string(nil), left...), right...), k)
		merged := Merge(leftSig, rightSig)
		if !reflect.DeepEqual(unionSig, merged) {
			t.Fatalf("第 %d 组：Signature(并集) 必须等于 Merge(两个签名)", trial)
		}
		if !reflect.DeepEqual(merged, Merge(rightSig, leftSig)) {
			t.Fatalf("第 %d 组：Merge 必须满足交换律", trial)
		}
		assocLeft := Merge(Merge(leftSig, rightSig), thirdSig)
		assocRight := Merge(leftSig, Merge(rightSig, thirdSig))
		if !reflect.DeepEqual(assocLeft, assocRight) {
			t.Fatalf("第 %d 组：Merge 必须满足结合律", trial)
		}
		unionAll := Signature(append(append(append([]string(nil), left...), right...), third...), k)
		if !reflect.DeepEqual(assocLeft, unionAll) {
			t.Fatalf("第 %d 组：三方合并必须等于并集签名", trial)
		}
	}
}

func allPairs(n int) [][2]int {
	pairs := [][2]int{}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			pairs = append(pairs, [2]int{i, j})
		}
	}
	return pairs
}

func TestChooseBands(t *testing.T) {
	bands, rows := ChooseBands(64, 0.8)
	if bands != 16 || rows != 4 {
		t.Fatalf("k=64 T=0.8 应选 16x4：%dx%d", bands, rows)
	}
	for _, k := range []int{1, 2, 7, 64, 97, 256, 1024} {
		for _, threshold := range []float64{0.1, 0.5, 0.8, 0.99, 1.0} {
			b, r := ChooseBands(k, threshold)
			if b*r != k {
				t.Fatalf("k=%d T=%v：B*R 必须等于 k：%d*%d", k, threshold, b, r)
			}
		}
	}
}

func TestClusterMatchesAllPairs(t *testing.T) {
	docs := GenerateDocs(400, 3)
	const k = 64
	const threshold = 0.8
	signatures := make([][]uint64, len(docs))
	for index, doc := range docs {
		signatures[index] = Signature(doc, k)
	}
	bands, rows := ChooseBands(k, threshold)
	candidates := Candidates(signatures, bands, rows)
	banded := Cluster(signatures, threshold, candidates)
	reference := Cluster(signatures, threshold, allPairs(len(docs)))
	if !reflect.DeepEqual(banded, reference) {
		t.Fatal("分带聚类的同簇关系必须与全量两两比较完全一致")
	}
}

func TestCandidateRecallAndSparsity(t *testing.T) {
	docs := GenerateDocs(2000, 1)
	const k = 64
	const threshold = 0.8
	signatures := make([][]uint64, len(docs))
	for index, doc := range docs {
		signatures[index] = Signature(doc, k)
	}
	bands, rows := ChooseBands(k, threshold)
	candidates := Candidates(signatures, bands, rows)
	n := len(docs)
	totalPairs := n * (n - 1) / 2
	if len(candidates)*20 >= totalPairs {
		t.Fatalf("候选对数必须远小于 N^2：candidates=%d pairs=%d", len(candidates), totalPairs)
	}
	isCandidate := map[[2]int]bool{}
	for _, pair := range candidates {
		isCandidate[pair] = true
	}
	sets := make([]map[string]bool, n)
	for index, doc := range docs {
		sets[index] = toSet(doc)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if ExactJaccard(sets[i], sets[j]) >= threshold && !isCandidate[[2]int{i, j}] {
				t.Fatalf("真实相似度 >= T 的对必须进入候选：(%d, %d)", i, j)
			}
		}
	}
}

func TestDeterministic(t *testing.T) {
	docs := GenerateDocs(100, 5)
	again := GenerateDocs(100, 5)
	if !reflect.DeepEqual(docs, again) {
		t.Fatal("相同参数的生成结果必须一致")
	}
	first := Cluster(signaturesOf(docs, 32), 0.5, Candidates(signaturesOf(docs, 32), 8, 4))
	second := Cluster(signaturesOf(docs, 32), 0.5, Candidates(signaturesOf(docs, 32), 8, 4))
	if !reflect.DeepEqual(first, second) {
		t.Fatal("同一输入重复聚类结果必须一致")
	}
}

func signaturesOf(docs [][]string, k int) [][]uint64 {
	signatures := make([][]uint64, len(docs))
	for index, doc := range docs {
		signatures[index] = Signature(doc, k)
	}
	return signatures
}
