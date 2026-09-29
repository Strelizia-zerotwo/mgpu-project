# 第四种模式：映射缺页基线 + 每 GPU 共享 L3 TLB

> 2026-09-29 路径更新：新的 L3 TLB 运行入口统一在项目根目录的 `benchmark/l3 tlb/`，结果在 `results/l3 tlb/<应用>/`。请优先使用该目录的 `README.md` 和 `VALIDATION.md`；下文的旧运行命令与环境状态作为历史说明保留。第四版现已接入 L3，但不代表本文中全部论文配置与 DNN 适配均已完成。

四种模式在同一份源码中通过一个互斥参数选择，无需复制四个仓库：

| 版本称呼 | 命令行参数 | 地址翻译路径 |
|---|---|---|
| 第一版：原有共享页表路径 | `-vm-mode=shared`（默认） | 原有 L1/L2 TLB → 共享 MMU |
| 第二版：映射缺页基线 | `-vm-mode=demand` | L1/L2 TLB → 每 GPU GMMU → 必要时 HostMMU |
| 第三版：本地映射全部预装 | `-vm-mode=ideal` | L1/L2 TLB → 每 GPU GMMU；有效分配的映射预先装入本地页表 |
| 第四版：第二版增加 L3 | `-vm-mode=demand-l3` | L1/L2 TLB → 每 GPU 共享 L3 TLB → GMMU → 必要时 HostMMU |

第二、三、四版共用已有的 GMMU/HostMMU 资源与延迟参数。第三版仍会查询 TLB 和进行本地页表遍历，并非所有 TLB 都命中。第二、四版模拟的是**固定数据位置的首次映射缺页**，不支持数据迁移、显存驱逐/换页和迁移后重新缺页。这里的“第一版”保留当前工作副本的旧执行路径，不代表自动还原为上游仓库的某个提交。

## 编译和选择模式

所有命令均在 Ubuntu 虚拟机执行：

```bash
export PATH=/usr/local/go/bin:$PATH
cd /home/only/projects/mgpu-project/external/mgpusim
go build -o amd/samples/fir/fir ./amd/samples/fir
```

编译一次后，通过参数切换模式。修改源码之后需要重新编译；只改参数不用重新编译。

运行第四版的四 GPU 小规模数值验证：

```bash
cd /home/only/projects/mgpu-project/external/mgpusim
mkdir -p "$HOME/mgpusim-runs"
mgpu_run=$(mktemp -d "$HOME/mgpusim-runs/demand-l3-XXXXXX")
./amd/samples/fir/fir \
  -timing -verify -disable-rtm \
  -gpus=1,2,3,4 -use-unified-memory -length=4096 \
  -vm-mode=demand-l3 -report-all \
  -metric-file-name="$mgpu_run/metrics"
printf '%s\n' "$mgpu_run"
python3 experiments/libra-baseline-guide/report_tlb.py \
  "$mgpu_run/metrics.sqlite3" --level L3 --gpus 1,2,3,4
python3 experiments/l3-tlb/check_metrics.py "$mgpu_run/metrics.sqlite3"
```

`-metric-file-name` 是输出前缀，生成 `.sqlite3` 文件。每次用新目录避免覆盖结果。`mgpu_run` 只在当前终端有效，换终端后需要设置为上一步打印的路径。

切换到第二版时，将 `-vm-mode=demand-l3` 换成 `-vm-mode=demand`；第三版换成 `ideal`；第一版换成 `shared`。其他参数保持相同，并分别保存结果。前三版没有 L3，不应对其使用 `--level L3`。旧的 `-ideal-local-page-table` 兼容开关仍存在，四种模式的对照实验请统一使用 `-vm-mode`，不要叠加旧开关。

## L3 配置含义

| 参数 | 默认值 | 含义 |
|---|---:|---|
| `-l3-tlb-entries` | 1024 | sector/tag 总项数 |
| `-l3-tlb-ways` | 8 | 每组 8 项，共 128 组 |
| `-l3-tlb-subentries` | 16 | 每项对应 16 个相邻虚拟页，各自独立有效 |
| `-l3-tlb-latency` | 40 | 1 GHz 下的内部查询周期数，即 40 ns，不含排队与连接延迟 |
| `-l3-tlb-width` | 4 | 每周期最多接收、完成查询、接收下级响应和发送响应各 4 个 |
| `-l3-tlb-mshrs` | 64 | 最多 64 个不同页的在途缺失 |
| `-l3-tlb-max-waiters` | 64 | 每个在途缺失最多合并 64 个请求，含最初的请求 |
| `-l3-tlb-max-inflight` | 256 | 组件内部最多 256 个已接收、尚未发送响应的请求 |

前四项采用所讨论配置表中的 L3 几何与延迟数值；吞吐宽度和队列容量是明确可调整的建模假设。Top 和 Bottom 各自还有宽度大小的有限收发缓冲，Control 缓冲为 1。资源不足时保留请求并施加反压，不丢弃请求。查询队列采用 FIFO，队头资源受阻时后续请求也等待。

替换以整项 sector 为单位执行 LRU。PID 是 tag 的一部分，不同进程不会混用映射。set = floor(VPN / 子项数) mod 组数。默认 4 KB 页时，16 个虚拟页属于同一 64 KB 对齐 sector，理论上最多缓存 1024 × 16 = 16384 个页映射。每次只填充 GMMU 实际返回的子项；不预取邻居页，不假定它们的物理地址连续。因此 16 个子项并不保证访问其中一页后其他 15 页立即命中。

示例：只修改第四版的 L3 延迟和项数，在原运行命令中追加：

```bash
-l3-tlb-latency=20 -l3-tlb-entries=512
```

这些 L3 参数只允许与 `demand-l3` 一起使用；其他模式显式传入会报错，防止参数被悄悄忽略。项数必须能被路数整除，子项数必须是正的 2 的幂，延迟、宽度和资源容量必须大于零。

## 结果怎么看

每张卡有一个 `GPU[1].L3TLB`、`GPU[2].L3TLB` 等组件；即使没有访问，也会输出零计数，读取工具会将无样本命中率显示为 `N/A`。

| 指标 | 含义 |
|---|---|
| `hit` | 查询完成时目标子项已经有效，直接返回映射 |
| `miss` | 新建一个缺失并向 GMMU 发送请求 |
| `mshr-hit` | 目标页仍在查询中，合并到已有缺失；不是有效映射命中 |
| `l3_requests` / `l3_completed` | L3 接收/已发送响应的请求数 |
| `l3_lower_responses` | 收到并验证通过的 GMMU 响应数 |
| `l3_fills` / `l3_sector_evictions` | 填回的子项数/替换的整项数 |
| `l3_outstanding_peak` / `l3_outstanding_end` | 内部未完成请求的峰值/结束剩余数 |
| `l3_mshr_peak` / `l3_mshr_end` | 不同页缺失槽的峰值/结束剩余数 |
| `l3_*_blocked_cycles` | 接收、查询或响应被有限资源阻塞的周期数 |
| `l3_translation_average` / `l3_translation_max` | 从 L3 接收请求到成功发送响应的模拟延迟，单位秒 |

这里使用的驻留映射命中率为：

```text
hit / (hit + miss + mshr-hit)
```

它是**到达 L3 的请求**的命中率，不是 GPU 全部访存的命中率。统计涵盖当前平台经由 L2 送来的翻译请求，包括指令与数据访问；默认统计窗口为整次运行，并未限制为某个 kernel 或稳态阶段。在没有 Reset 的完整运行中应满足：

```text
l3_requests = hit + miss + mshr-hit = l3_completed
miss = l3_lower_responses = GPU[n].MMU 的 vm_requests
l3_outstanding_end = l3_mshr_end = 0
```

### 小规模 FIR 的 L3 为 0% 是否错误？

不一定。目前旧平台的 L2 TLB 很大，重复访问通常在 L2 已经命中，L3 可能只接收到首次访问。这次只增加 L3，并未缩小 L2 或改变旧模式来人为制造 L3 命中。用小规模 FIR 主要验证四 GPU 连通、数值正确和计数守恒；组件测试另行验证首次缺失、再次访问准确命中以及 40 周期查询延迟。

这次改动不等于完整复现 LIBRA 的硬件表：TPC/GPC 共享关系、sector L2、108 SM、70% 显存容量和 NVLink/PCIe 带宽等仍需单独实现/校准。当前模型继续使用 MGPUSim 的 AMD 执行平台。

## 控制协议与代码位置

L3 Control 端口接入各 GPU 的 CommandProcessor；支持 Pause、Enable、Drain、Reset 和按 PID/虚拟页过滤的 Invalidate。Invalidate 必须在暂停且内部请求已经完成时执行，避免旧的在途响应再次装入已失效的映射。Drain 完成内部请求后暂停，尚在入口缓冲的请求留待 Enable。Reset 丢弃内部工作和缓存，晚到的旧响应不会填入缓存；统计保留累计值，并另计被丢弃请求和过期响应。Flush 不适用于没有脏数据的 TLB。

提供这些组件控制能力不代表系统已支持迁移 UVM；基线对运行中改变物理映射仍有固定位置约束。

- `amd/timing/sectortlb/`：缓存、时序、有限资源、控制协议与测试。
- `amd/samples/runner/l3flags.go`：L3 参数。
- `amd/samples/runner/l3report.go`：L3 指标。
- `amd/samples/runner/timingconfig/fault.go`：第四模式的平台连接。
- `amd/samples/runner/timingconfig/{r9nano,mi300x}/builder.go`：Control 端口连接。

组件验证命令：

```bash
go test -race ./amd/timing/sectortlb ./amd/timing/faultvm ./amd/timing/idealmapping
```
