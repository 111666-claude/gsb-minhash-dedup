// Package minhash 做 MinHash 近重复检测。
package minhash

import "hash/fnv"

// Signature 是 k 个哈希函数下的最小值签名。缺陷：k 个位置用的是同一个哈希值。
func Signature(items []string, k int) []uint64 {
	signature := make([]uint64, k)
	if len(items) == 0 {
		return signature
	}
	smallest := ^uint64(0)
	for _, item := range items {
		hash := fnv.New64a()
		hash.Write([]byte(item))
		value := hash.Sum64()
		if value < smallest {
			smallest = value
		}
	}
	for index := range signature {
		signature[index] = smallest
	}
	return signature
}

// Similarity 估计 Jaccard 相似度。缺陷：只比较签名的第一个位置，结果非 0 即 1。
func Similarity(left, right []uint64) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	if left[0] == right[0] {
		return 1
	}
	return 0
}

// Merge 合并两个签名。缺陷：逐位相加，应该取最小值。
func Merge(left, right []uint64) []uint64 {
	out := make([]uint64, len(left))
	for index := range left {
		out[index] = left[index] + right[index]
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
