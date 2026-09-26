package minhash

import (
	"fmt"
	"math/rand"
)

// Generate 确定性地生成 n 篇文档：大部分 5 篇成一簇（簇内 Jaccard 约 0.88），
// 簇与簇之间基本不相交。同一 seed 重复生成结果完全相同。
func Generate(n int, seed int64) [][]string {
	rng := rand.New(rand.NewSource(seed))
	docs := make([][]string, 0, n)
	const clusterSize = 5
	const docSize = 200
	clusterID := 0
	fresh := 0
	for len(docs) < n {
		if len(docs)+clusterSize <= n {
			base := make([]string, docSize)
			for j := range base {
				base[j] = fmt.Sprintf("c%d-t%d", clusterID, j)
			}
			for member := 0; member < clusterSize && len(docs) < n; member++ {
				doc := make([]string, 0, docSize)
				for _, term := range base {
					if rng.Float64() < 0.03 {
						doc = append(doc, fmt.Sprintf("u%d", fresh))
						fresh++
					} else {
						doc = append(doc, term)
					}
				}
				docs = append(docs, doc)
			}
			clusterID++
		} else {
			doc := make([]string, docSize)
			for j := range doc {
				doc[j] = fmt.Sprintf("s%d-%d", clusterID, j)
			}
			clusterID++
			docs = append(docs, doc)
		}
	}
	return docs
}
