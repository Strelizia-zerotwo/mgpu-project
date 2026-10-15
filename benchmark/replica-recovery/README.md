# 失效副本低成本恢复：理想收益验证

目标：在 GPU 多卡 UVM 下，先回答"失效副本只补齐变化部分"的**理想上界**有多大，再决定是否值得设计真实机制。

## 设计空间（要测的四格）

| | 补整页 | 只补变化部分（delta） |
|---|---|---|
| **惰性**：下次访问缺页时修复 | INV（基线：失效后整页重新复制） | RETAIN-lazy |
| **主动**：在同步边界修复 | eager-full（≈ 整页预取） | RETAIN-eager；推给所有持有者时 ≈ GPS 按 epoch 合并 |

注意：RETAIN-lazy **不减少缺页次数**，只减少每次缺页搬运的字节。4KB 页的传输时间（约几十 ns）远小于缺页固定开销（µs 级），此时 delta 基本省不出时间。收益只可能来自两处：大页（64KB/2MB），或在边界主动修复从而消掉缺页。主动修复要付出给 C、D 这类"暂时不用或不再用"的持有者修复的浪费。

## 第一层：离线 trace 上界（已实现，不涉及时序）

插桩（`MGPU_MEMTRACE=<前缀>` 时启用）：

- `external/mgpusim/amd/timing/memtrace/memtrace.go`：输出 32 字节记录（R/W/S/C）和 `.meta` 事件（U 分配、K kernel、E 结束、H/D 拷贝）
- timing 模式的 hook：`timing/cu/vectormemoryunit.go`、`scalarunit.go`
- emulation 模式的 hook：`samples/runner/emusystem/emugpu/builder.go`（tracingAccessor，额外输出值真正改变的字节，记录类型 C）
- 驱动事件：`driver/driver.go`、`driver/api.go`

使用方法：

```bash
bash benchmark/replica-recovery/build.sh                 # 编译 apps.txt 里的应用和分析器
bash benchmark/replica-recovery/run.sh floydwarshall s -node=256 -iter=32
# 环境变量：PAGES=4096,65536,2097152  KEEP_TRACE=1（保留 trace.bin）  TIMEOUT=3600
benchmark/replica-recovery/bin/rranalyze -trace results/replica-recovery/floydwarshall/s/trace -changed
```

`-changed` 用"值真正改变"的字节作为脏集合，这是绝对下界；不加时用写掩码，对应现实中可行的硬件脏位。

分析器的模型假设：

- epoch 等于 kernel 边界（H2D 视为主机写）
- 某个 epoch 里有写的页收缩为单份 home，其他 GPU 在该 epoch 内远端访问
- 读取已失效的副本时修复

`summary.txt` 各项含义：

- refill：失效后又被重新读的次数；`never re-read`：失效后再也没读
- RETAIN / %INV：各粒度下补齐的字节，以及占整页重复制的比例
- UPD push / UPD->never：GPS 式推送的字节，以及推给后来再也没读的持有者的字节
- merged：一次修复合并了几个写 epoch；reuse distance：失效到重新读隔了几个 epoch
- stale retained：保留旧副本占用的页数

运行前需要修正的问题：

- 只带 gfx942 kernel 的应用要加 `-arch cdna3 -gpu mi300x`
- atax、bfs、spmv 没有用满 4 块 GPU，要剔除
- nw 本身验证失败
- 输入要大到足迹远大于页面：64KB 页需要几十 MB，2MB 页需要几百 MB
- `-unified-gpus` 按连续 WG 区间切分；这是建模选择，会影响共享模式

## 第二层：粗时间估计（不改模拟器）

```
T_INV        ≈ N_refill × (t_fault + P/BW)
T_lazy-delta ≈ N_refill × (t_fault + Δ/BW)
T_eager      ≈ Σ_边界 (推给被选持有者的 Δ)/BW  +  漏选导致的缺页
节省上限      =  (T_INV − T_x) / 基线 kernel 总时间（来自一次 timing 运行）
```

t_fault 至少扫两个点：模拟器默认值（host 400 + link 2×200 + install 100 周期，约 0.9 µs，很乐观）和真实 UVM 量级（数十 µs）。即使 Perfect（补齐成本为零）在多数应用上节省也小于约 10% 时，停止这个方向。

## 第三层：时序理想模型（需要改模拟器）

现状：`timing/faultvm` 明确写了"Data placement is fixed; no migration"，没有复制和迁移，翻译请求也不区分读写。因此时序实验要自己搭复制协议。推荐用"oracle 驱动的两遍法"，避免一开始就实现写保护缺页：

1. **第一遍**（emulation）：分析器为每个 kernel 输出 plan，包括每页的 home、合法本地副本集合，以及需要失效和修复的副本。
2. **第二遍**（timing）：
   - kernel 启动前，驱动或 CP 按 plan 执行：失效（PTE、TLB shootdown、该页在本地 L2 中的行全部作废）。理想情况下成本为零，可选择计入固定开销。
   - 缺页路径（faultvm host）：若该 GPU 在本 kernel 应持有副本，就把整页数据**功能上完整拷贝**以保证正确性，时序上 INV 计 P 字节、RETAIN 计 Δ 字节，经过带宽和排队模型；然后安装指向本地帧的映射。
   - eager 变体在 kernel 启动前完成修复，修复时间与上一个 kernel 尾部或启动开销重叠或串行，两种都报告。
   - 需要按 GPU 分开的页表视图（参考 `idealmapping` 的 `tables.View`）和本地副本帧分配。
3. **对比配置**：NoRep（固定放置加远端访问）、INV、RETAIN-lazy、eager-full、RETAIN-eager、GPS-like、Perfect（副本始终本地且零成本）。
4. **正确性与合理性检查**：所有配置都要 `-verify` 通过；RETAIN ≤ INV；BW→∞ 且 t_fault→0 时 INV 应趋近 Perfect。

oracle 在边界就知道本 kernel 会写哪些页，这是理想化假设，结果只能作为上界报告。
