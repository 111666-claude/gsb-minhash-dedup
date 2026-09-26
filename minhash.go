// Package minhash 做 MinHash 近重复检测。
package minhash

import (
	"hash/fnv"
	"math/bits"
)

// prime 是通用哈希族使用的梅森素数 2^61-1。
const prime = uint64(1)<<61 - 1

// hashItem 用 FNV-1a 把元素确定性地映成 64 位整数。
func hashItem(item string) uint64 {
	hash := fnv.New64a()
	hash.Write([]byte(item))
	return hash.Sum64()
}

// splitmix64 确定性打散，用于从哈希函数下标推导种子。
func splitmix64(x uint64) uint64 {
	x += 0x9E3779B97F4A7C15
	x = (x ^ (x >> 30)) * 0xBF58476D1CE4E5B9
	x = (x ^ (x >> 27)) * 0x94D049BB133111EB
	return x ^ (x >> 31)
}

// hashParams 给出第 index 个通用哈希函数 h(x) = (a*x+b) mod p 的参数。
// 参数只由下标决定，完全确定，与运行次数无关。
func hashParams(index int) (a, b uint64) {
	a = splitmix64(uint64(2*index))%(prime-1) + 1
	b = splitmix64(uint64(2*index+1)) % prime
	return a, b
}

// applyHash 计算 (a*x+b) mod (2^61-1)，利用 2^61 ≡ 1 (mod p) 做折叠。
func applyHash(a, b, x uint64) uint64 {
	x = (x & prime) + (x >> 61)
	if x >= prime {
		x -= prime
	}
	hi, lo := bits.Mul64(a, x)
	sum := (lo & prime) + (lo >> 61) + (hi << 3) + b
	sum = (sum & prime) + (sum >> 61)
	if sum >= prime {
		sum -= prime
	}
	return sum
}

// Signature 计算 k 个独立通用哈希各自的最小值签名。
// 每个位置用不同种子的哈希函数；对元素顺序不敏感（集合语义）；
// 内存 O(k)，单元素 O(k)。空输入返回全 ^uint64(0)（最小值的单位元）。
func Signature(items []string, k int) []uint64 {
	signature := make([]uint64, k)
	for index := range signature {
		signature[index] = ^uint64(0)
	}
	if len(items) == 0 || k <= 0 {
		return signature
	}
	params := make([][2]uint64, k)
	for index := 0; index < k; index++ {
		a, b := hashParams(index)
		params[index] = [2]uint64{a, b}
	}
	for _, item := range items {
		x := hashItem(item)
		for index := 0; index < k; index++ {
			if value := applyHash(params[index][0], params[index][1], x); value < signature[index] {
				signature[index] = value
			}
		}
	}
	return signature
}

// Similarity 估计 Jaccard 相似度：签名相等位置的比例，取值 [0,1]。
func Similarity(left, right []uint64) float64 {
	n := len(left)
	if len(right) < n {
		n = len(right)
	}
	if n == 0 {
		return 0
	}
	equal := 0
	for index := 0; index < n; index++ {
		if left[index] == right[index] {
			equal++
		}
	}
	return float64(equal) / float64(n)
}

// Merge 逐位取最小值合并两个签名，等价于对并集重新计算签名。
// 满足交换律与结合律。
func Merge(left, right []uint64) []uint64 {
	n := len(left)
	if len(right) < n {
		n = len(right)
	}
	out := make([]uint64, n)
	for index := 0; index < n; index++ {
		if right[index] < left[index] {
			out[index] = right[index]
		} else {
			out[index] = left[index]
		}
	}
	return out
}

// ExactJaccard 精确相似度（参考实现）。
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
