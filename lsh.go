package minhash

import "sort"

// PickBands 为签名长度 k 与阈值 threshold 选择分带参数，保证 bands*rows == k。
// 选取最大的 rows 使 bands > (1-threshold)*k：由鸽巢原理，估计相似度
// >= threshold 的签名对至多 (1-threshold)*k 个位置不同，不足以让 bands 条带
// 全部不同，因此至少有一条完全相同的带，必然成为候选（召回确定性完备）。
func PickBands(k int, threshold float64) (bands, rows int) {
	if k <= 0 {
		return 0, 0
	}
	if threshold >= 1 {
		return 1, k
	}
	rows = 1
	for r := 2; r <= k; r++ {
		if k%r != 0 {
			continue
		}
		if float64(r)*(1-threshold) < 1 {
			rows = r
		}
	}
	return k / rows, rows
}

// bandKey 把一条带的 rows 个值确定性地混成 64 位键。
func bandKey(band int, values []uint64) uint64 {
	key := splitmix64(uint64(band))
	for _, value := range values {
		key = splitmix64(key ^ value)
	}
	return key
}

// Candidates 用 LSH 分带筛选候选对，返回 (i, j)（i < j），按下标升序。
// 只比较共享同一条带的文档，工作量 O(N*k + 候选数)，不是 O(N^2)。
func Candidates(signatures [][]uint64, bands, rows int) [][2]int {
	if bands <= 0 || rows <= 0 {
		return nil
	}
	n := len(signatures)
	seen := make(map[uint64]struct{})
	pairs := [][2]int{}
	bucket := make(map[uint64][]int)
	for band := 0; band < bands; band++ {
		start := band * rows
		for id, signature := range signatures {
			if start+rows > len(signature) {
				continue
			}
			key := bandKey(band, signature[start:start+rows])
			bucket[key] = append(bucket[key], id)
		}
		for _, ids := range bucket {
			for x := 0; x < len(ids); x++ {
				for y := x + 1; y < len(ids); y++ {
					i, j := ids[x], ids[y]
					if i > j {
						i, j = j, i
					}
					pairKey := uint64(i)*uint64(n) + uint64(j)
					if _, dup := seen[pairKey]; !dup {
						seen[pairKey] = struct{}{}
						pairs = append(pairs, [2]int{i, j})
					}
				}
			}
		}
		for key := range bucket {
			delete(bucket, key)
		}
	}
	sort.Slice(pairs, func(a, b int) bool {
		if pairs[a][0] != pairs[b][0] {
			return pairs[a][0] < pairs[b][0]
		}
		return pairs[a][1] < pairs[b][1]
	})
	return pairs
}

// clusterFromPairs 在候选对上按阈值做并查集聚类，返回排序后的簇。
func clusterFromPairs(signatures [][]uint64, threshold float64, pairs [][2]int) [][]int {
	n := len(signatures)
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		for parent[x] != x {
			parent[x] = parent[parent[x]]
			x = parent[x]
		}
		return x
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			if ra > rb {
				ra, rb = rb, ra
			}
			parent[rb] = ra
		}
	}
	for _, pair := range pairs {
		if Similarity(signatures[pair[0]], signatures[pair[1]]) >= threshold {
			union(pair[0], pair[1])
		}
	}
	groups := make(map[int][]int)
	for i := 0; i < n; i++ {
		root := find(i)
		groups[root] = append(groups[root], i)
	}
	out := make([][]int, 0, len(groups))
	for _, members := range groups {
		out = append(out, members)
	}
	sort.Slice(out, func(a, b int) bool { return out[a][0] < out[b][0] })
	return out
}

// allPairs 全量两两组合（参考实现用）。
func allPairs(n int) [][2]int {
	pairs := make([][2]int, 0, n*(n-1)/2)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			pairs = append(pairs, [2]int{i, j})
		}
	}
	return pairs
}

// Cluster 用 LSH 分带候选把估计相似度 >= threshold 的文档聚成簇。
// 分带参数满足 PickBands 的鸽巢条件时，结果与 ClusterReference 完全一致。
// threshold <= 0 时所有文档互为候选（退化情形）。
func Cluster(signatures [][]uint64, threshold float64, bands, rows int) [][]int {
	if threshold <= 0 {
		return clusterFromPairs(signatures, threshold, allPairs(len(signatures)))
	}
	return clusterFromPairs(signatures, threshold, Candidates(signatures, bands, rows))
}

// ClusterReference 全量两两比较的参考实现，用于校验 Cluster 的同簇关系。
func ClusterReference(signatures [][]uint64, threshold float64) [][]int {
	return clusterFromPairs(signatures, threshold, allPairs(len(signatures)))
}
