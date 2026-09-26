# minhash-dedup

用 MinHash 估计两批数据的 Jaccard 相似度，并按阈值做近重复聚类，只用 Go 标准库。

```
go test ./...
go run ./cmd/minhash-dedup --left sample/a.txt --right sample/b.txt --k 64
go run ./cmd/minhash-dedup --generate 2000 --threshold 0.8
```

## 口径（README 为准）

- **签名是 k 个独立哈希各自的最小值**：`Signature(items, k)` 的第 `i` 个位置用通用哈希族
  `h_i(x) = (a_i*x + b_i) mod p`（`p = 2^61-1`，`a_i`、`b_i` 由下标确定性推出），
  对所有元素取最小值。输出长度等于 `k`，对元素顺序不敏感（集合语义）。
- **相似度 = 签名相等位置的比例**：`Similarity` 返回 `相等位置数 / k`，取值在 `[0,1]`。
- **精度**：`k = 64` 时自带样例估计值与精确 Jaccard（3/7 ≈ 0.4286）的差 `<= 0.05`；`k` 越大越稳。
- **合并取最小值**：`Merge(left, right)` 逐位取 `min`，并且
  `Signature(左右并集) == Merge(左右签名)` 逐位成立；满足交换律与结合律。
- **分带筛选**：`PickBands(k, T)` 选取 `bands*rows == k` 且 `bands > (1-T)*k` 的参数。
  由鸽巢原理，估计相似度 `>= T` 的对至少有一条完全相同的带，必然进入候选（召回确定性完备），
  而候选对数远小于 `N^2`。
- **聚类**：`Cluster` 在候选对上做并查集，结果与 `ClusterReference`（全量两两比较）完全一致。
- 不变量：同一输入重复计算签名相同（哈希确定，不依赖 map 迭代顺序或随机种子）；
  两个集合完全相同时相似度为 1；无交集时为 0；空输入不 panic；`k` 在 1 到 1024 都可用。
- 规模：100 万个元素、`k = 256` 时内存 O(k)，单元素 O(k) 哈希；合并 O(k)；
  候选筛选与聚类总工作量 O(N*k + 候选数)，不是 O(N^2)。

## 命令行

```
--left/--right 文件   两两比较模式（默认），一行一个元素
--k N                 签名长度（哈希函数个数），默认 64
--threshold T         聚类阈值（0 到 1），设置后进入聚类模式
--generate N          生成 N 篇合成文档（确定性）用于聚类
--bands B --rows R    显式分带参数（B*R 必须等于 k；缺省按阈值自动选取）
文件...               聚类模式下作为额外文档读入
```

不带新参数时输出格式不变：

```
estimate=… exact=… diff=… k=…
```

聚类模式输出：

```
docs=… k=… bands=… rows=… threshold=… candidates=…
clusters=…
cluster 1 (size 5): gen-0000 gen-0001 …
```

## 目录

```
minhash.go             签名、相似度、合并、精确 Jaccard 参考
lsh.go                 分带参数、候选筛选、聚类（含全量两两参考实现）
generate.go            确定性合成数据生成
cmd/minhash-dedup      命令行入口
minhash_test.go        基础用例
minhash_fix_test.go    精度/合并/不变量用例（含 1000 组随机对照）
lsh_test.go            分带召回、聚类一致性、候选规模用例
sample/a.txt           5 个元素
sample/b.txt           5 个元素（与 a 交集 3）
```
