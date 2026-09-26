# minhash-dedup

用 MinHash 估计两批数据的 Jaccard 相似度，只用 Go 标准库。

```
go test ./...
go run ./cmd/minhash-dedup --left sample/a.txt --right sample/b.txt --k 64
```

## 口径（README 为准）

- **签名是 k 个独立哈希各自的最小值**：`Signature(items, k)` 要给出 `k` 个不同的位置，
  每个位置用不同的种子哈希（同一种子重复填 k 次不算）。现在 k 个位置填的是同一个最小值。
- **相似度 = 签名相等位置的比例**：`Similarity` 返回 `相等位置数 / k`，取值在 `[0,1]`。
  现在只看第一个位置，结果非 0 即 1：`--k 64` 对自带样例给 `estimate=1.0000`，而精确值是 `0.4286`。
- **精度**：`k = 64` 时估计值与精确 Jaccard 的差必须 `<= 0.05`（现在 `diff=0.5714`）。
- **合并取最小值**：`Merge(left, right)` 逐位取 `min`（不是相加），并且
  `Signature(左右并集) == Merge(左右签名)` 必须逐位成立。
- 不变量：同一输入重复计算签名相同；两个集合完全相同时相似度为 1；无交集时为 0；
  `k` 越大估计越稳（1..1024 范围内都可用）。
- 规模：100 万个元素、`k = 256` 时内存 O(k)，单元素 O(k) 哈希；合并 O(k)。

## 现在的行为

- `Signature` 只算一个 FNV 哈希，然后把同一个最小值填进 k 个位置：所有位置完全相同。
- `Similarity` 只比较 `left[0]` 与 `right[0]`：相等给 1、不等给 0，所以自带样例得到
  `estimate=1.0000 exact=0.4286 diff=0.5714`。
- `Merge` 逐位相加，合并后的签名没有意义。
- `k` 参数除了决定切片长度之外没有任何作用。

## 计划新增的能力

```
--k N          签名长度（哈希函数个数），默认 64
--exact        额外打印精确 Jaccard 与差值
```

## 输出

```
estimate=… exact=… diff=… k=…
```

## 目录

```
minhash.go             签名、相似度、合并
cmd/minhash-dedup      命令行入口
minhash_test.go        go test 用例
sample/a.txt           5 个元素
sample/b.txt           5 个元素（与 a 交集 3）
```
