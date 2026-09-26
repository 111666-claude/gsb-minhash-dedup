// Package minhash 做 MinHash 近重复检测。
package minhash

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math/bits"
	"math/rand"
	"sort"
)

// prime 是通用哈希族使用的素数模数 2^61-1。
const prime = uint64(1)<<61 - 1

// hashItem 把元素确定性地映射到 [0, prime) 内。
func hashItem(item string) uint64 {
	hasher := fnv.New64a()
	hasher.Write([]byte(item))
	value := hasher.Sum64()
	value = (value & prime) + (value >> 61)
	if value >= prime {
		value -= prime
	}
	return value
}

// coefficient 用 splitmix64 从位置下标确定性地生成哈希系数，不依赖随机种子。
func coefficient(index int, salt uint64) uint64 {
	z := uint64(index)*0x9E3779B97F4A7C15 + salt
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	z = z ^ (z >> 31)
	return z % prime
}

// 系数盐值：两个不同的常数，保证 a、b 两列系数独立。
const (
	saltA = 0xA5A5A5A5A5A5A5A5
	saltB = 0x5A5A5A5A5A5A5A5A
)

// applyHash 计算 (a*x+b) mod (2^61-1)，用 128 位中间结果避免溢出。
func applyHash(a, b, x uint64) uint64 {
	hi, lo := bits.Mul64(a, x)
	var carry uint64
	lo, carry = bits.Add64(lo, b, 0)
	hi += carry
	// 2^64 ≡ 8 (mod 2^61-1)
	lo, carry = bits.Add64(lo, hi<<3, 0)
	reduced := (lo & prime) + (lo >> 61) + 8*carry
	for reduced >= prime {
		reduced -= prime
	}
	return reduced
}

// Signature 计算 k 个独立通用哈希函数下的最小值签名。
// 第 i 个位置用哈希族 h_i(x) = (a_i*x+b_i) mod (2^61-1) 取集合最小值，
// 是集合语义，与元素顺序无关；空集合的签名全部取最大值（合并的单位元）。
// 内存 O(k)，每个元素 O(k)。
func Signature(items []string, k int) []uint64 {
	signature := make([]uint64, k)
	for index := range signature {
		signature[index] = ^uint64(0)
	}
	if k <= 0 {
		return signature
	}
	coeffA := make([]uint64, k)
	coeffB := make([]uint64, k)
	for index := 0; index < k; index++ {
		coeffA[index] = coefficient(index, saltA)%(prime-1) + 1
		coeffB[index] = coefficient(index, saltB)
	}
	for _, item := range items {
		x := hashItem(item)
		for index := 0; index < k; index++ {
			if value := applyHash(coeffA[index], coeffB[index], x); value < signature[index] {
				signature[index] = value
			}
		}
	}
	return signature
}

// Similarity 返回两个签名相等位置的比例，取值 [0,1]。
func Similarity(left, right []uint64) float64 {
	length := len(left)
	if len(right) < length {
		length = len(right)
	}
	if length == 0 {
		return 0
	}
	equal := 0
	for index := 0; index < length; index++ {
		if left[index] == right[index] {
			equal++
		}
	}
	return float64(equal) / float64(length)
}

// Merge 逐位取最小值合并两个签名；满足交换律与结合律，
// 且 Signature(并集) 与 Merge(两个签名) 逐位一致。
func Merge(left, right []uint64) []uint64 {
	length := len(left)
	if len(right) < length {
		length = len(right)
	}
	out := make([]uint64, length)
	for index := 0; index < length; index++ {
		if right[index] < left[index] {
			out[index] = right[index]
		} else {
			out[index] = left[index]
		}
	}
	return out
}

// ExactJaccard 精确相似度（参考用）。
func ExactJaccard(left, right map[string]bool) float64 {
	if len(left) == 0 && len(right) == 0 {
		return 1
	}
	inter := 0
	for item := range left {
		if right[item] {
			inter++
		}
	}
	union := len(left) + len(right) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// ChooseBands 选取满足 bands*rows == k 且 bands > k*(1-threshold) 的最小分带数。
// 不相等位置数少于分带数时至少有一个分带完全相等，因此估计相似度
// >= threshold 的对必然成为候选（召回完备）。没有合适约数时退化为 bands=k, rows=1。
func ChooseBands(k int, threshold float64) (bands, rows int) {
	for candidate := 1; candidate <= k; candidate++ {
		if k%candidate == 0 && float64(candidate) > float64(k)*(1-threshold) {
			return candidate, k / candidate
		}
	}
	return k, 1
}

// Candidates 用 LSH 分带从签名中筛候选相似对：同一分带内 R 个位置
// 全部相等的文档对进入候选。bands*rows 必须等于签名长度。
// 返回去重并排序后的文档下标对，结果确定。
func Candidates(signatures [][]uint64, bands, rows int) [][2]int {
	seen := map[[2]int]bool{}
	var buffer [8]byte
	for band := 0; band < bands; band++ {
		buckets := map[uint64][]int{}
		for doc, signature := range signatures {
			if len(signature) < (band+1)*rows {
				continue
			}
			hasher := fnv.New64a()
			for _, value := range signature[band*rows : (band+1)*rows] {
				binary.LittleEndian.PutUint64(buffer[:], value)
				hasher.Write(buffer[:])
			}
			key := hasher.Sum64()
			buckets[key] = append(buckets[key], doc)
		}
		for _, docs := range buckets {
			for i := 0; i < len(docs); i++ {
				for j := i + 1; j < len(docs); j++ {
					seen[[2]int{docs[i], docs[j]}] = true
				}
			}
		}
	}
	pairs := make([][2]int, 0, len(seen))
	for pair := range seen {
		pairs = append(pairs, pair)
	}
	sort.Slice(pairs, func(a, b int) bool {
		if pairs[a][0] != pairs[b][0] {
			return pairs[a][0] < pairs[b][0]
		}
		return pairs[a][1] < pairs[b][1]
	})
	return pairs
}

// Cluster 把估计相似度 >= threshold 的文档用并查集聚成簇，只考察
// candidates 里的对。返回的簇与簇内成员都按下标排序，结果确定。
func Cluster(signatures [][]uint64, threshold float64, candidates [][2]int) [][]int {
	parent := make([]int, len(signatures))
	for index := range parent {
		parent[index] = index
	}
	var find func(int) int
	find = func(x int) int {
		for parent[x] != x {
			parent[x] = parent[parent[x]]
			x = parent[x]
		}
		return x
	}
	for _, pair := range candidates {
		left, right := pair[0], pair[1]
		if Similarity(signatures[left], signatures[right]) < threshold {
			continue
		}
		rootLeft, rootRight := find(left), find(right)
		if rootLeft == rootRight {
			continue
		}
		if rootLeft < rootRight {
			parent[rootRight] = rootLeft
		} else {
			parent[rootLeft] = rootRight
		}
	}
	groups := map[int][]int{}
	for index := range signatures {
		root := find(index)
		groups[root] = append(groups[root], index)
	}
	clusters := make([][]int, 0, len(groups))
	for _, members := range groups {
		clusters = append(clusters, members)
	}
	sort.Slice(clusters, func(a, b int) bool { return clusters[a][0] < clusters[b][0] })
	return clusters
}

// GenerateDocs 生成确定性的合成文档集：约十分之一为近重复簇
// （簇内 Jaccard 约 0.9），其余为随机文档。相同 n 与 seed 生成相同结果。
func GenerateDocs(n, seed int) [][]string {
	rng := rand.New(rand.NewSource(int64(seed)))
	docs := make([][]string, 0, n)
	const baseSize = 40
	const universe = 10000000
	randomDoc := func(size int) []string {
		doc := make([]string, size)
		for index := range doc {
			doc[index] = fmt.Sprintf("e%07d", rng.Intn(universe))
		}
		return doc
	}
	clusterCount := n / 10
	for cluster := 0; cluster < clusterCount && len(docs) < n; cluster++ {
		base := randomDoc(baseSize)
		members := 8 + rng.Intn(5)
		for member := 0; member < members && len(docs) < n; member++ {
			doc := make([]string, 0, baseSize)
			for _, item := range base {
				if rng.Float64() < 0.95 {
					doc = append(doc, item)
				}
			}
			doc = append(doc, randomDoc(2)...)
			docs = append(docs, doc)
		}
	}
	for len(docs) < n {
		docs = append(docs, randomDoc(baseSize))
	}
	return docs
}
