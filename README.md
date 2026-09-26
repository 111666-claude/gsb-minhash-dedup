# minhash-dedup

用 MinHash 估计两批数据的 Jaccard 相似度，并按阈值做近重复聚类，只用 Go 标准库。

```
go test ./...
go run ./cmd/minhash-dedup --left sample/a.txt --right sample/b.txt --k 64
go run ./cmd/minhash-dedup --generate 2000 --threshold 0.8
```

## 口径（README 为准）

- **签名是 k 个独立哈希各自的最小值**：`Signature(items, k)` 的第 `i` 个位置用通用哈希族
  `h_i(x) = (a_i*x + b_i) mod (2^61-1)`（系数由位置下标确定性生成）取集合最小值，
  输出长度等于 `k`，是集合语义，与元素顺序无关。
- **相似度 = 签名相等位置的比例**：`Similarity` 返回 `相等位置数 / k`，取值在 `[0,1]`。
- **精度**：`k = 64` 时对自带样例 `estimate=0.4375 exact=0.4286 diff=0.0089`，差值 `<= 0.05`。
- **合并取最小值**：`Merge(left, right)` 逐位取 `min`，满足交换律与结合律，并且
  `Signature(左右并集) == Merge(左右签名)` 逐位成立。
- 不变量：同一输入重复计算签名相同；两个集合完全相同时相似度为 1；无交集时为 0；
  空输入不 panic；`k` 在 1..1024 范围内都可用，`k` 越大估计越稳。
- 规模：100 万个元素、`k = 256` 时内存 O(k)，单元素 O(k) 哈希；合并 O(k)。

## 近重复聚类与分带筛选

```
--threshold T    聚类阈值（0 到 1），相似度 >= T 的文档聚成一簇
--bands B        LSH 分带数（与 --rows 一起给，B*R = k）
--rows R         每个分带的行数
--generate N     生成 N 篇确定性合成文档（约十分之一为近重复簇）
```

- 不带 `--bands/--rows` 时自动选取满足 `B*R = k` 且 `B > k*(1-T)` 的最小分带数：
  不相等位置数少于分带数，至少一个分带完全相等，因此相似度 >= T 的对必然成为候选
  （召回完备），且同簇关系与全量两两比较完全一致。
- 候选对来自 LSH 分带，数量远小于 N^2：`--generate 2000 --threshold 0.8` 时
  候选约 9 千对（全量约 200 万对）。
- 文档来源：位置参数给出的文件（一行一个元素）与 `--generate` 的合成文档可以混用。

输出：

```
docs=2000 k=64 threshold=0.8000 bands=16 rows=4 candidates=9191 clusters=308
cluster 0 size=9: gen-0000 gen-0001 ...
```

## 两文件比较（默认模式，输出格式不变）

```
estimate=… exact=… diff=… k=…
```

## 目录

```
minhash.go                  签名、相似度、合并、分带候选、聚类
cmd/minhash-dedup           命令行入口
minhash_test.go             go test 用例（含精确参考对照）
sample/a.txt                5 个元素
sample/b.txt                5 个元素（与 a 交集 3）
```
