package minhash

import "testing"

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
