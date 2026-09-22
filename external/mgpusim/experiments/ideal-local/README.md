# 每 GPU 本地页表映射全部可用：入门实验

> 新的映射缺页研究请使用 [demand/ideal 基线](../fault-baseline/README.md)。它提供真实请求、主机排队及相同硬件配置的理想对照；本文保留早期理想页表实验的说明。

## 先明确这个实验测什么

目标：**TLB 可以 miss，GPU 仍然执行有延迟的页表遍历，但合法分配的地址在 GPU 本地页表里始终有有效映射。**

新增参数 `-ideal-local-page-table` 开启这个理想化模型。它不是 NVIDIA GMMU 的硬件复刻，也不是完整 UVM 实现。

本项目的 MGPUSim v5 默认配置只有一个共享 MMU 和一张共享页表。内存分配时已经建立有效映射；找不到页面时会报错，而不是请求主机 UVM 处理。因此，共享模式与新增模式的时间差 **不能作为消除真实 UVM fault 的收益**。

| 内容 | 默认共享模式 | `-ideal-local-page-table` |
|---|---|---|
| MMU | 所有 GPU 共用 `MMU` | 每个 GPU 一个 `GPU[n].MMU` |
| 页表 | 驱动和 MMU 共用一张表 | 驱动权威表 + 每 GPU 独立副本 |
| 建立/更新/删除映射 | 操作共享表 | 零模拟延迟同步到所有 GPU 副本 |
| GPU 翻译时查哪张表 | 共享表 | 只查自己的副本，不查驱动表兜底 |
| TLB | 原来的 L1/L2 TLB | 保留相同配置 |
| 页表遍历配置 | 100 周期，1 GHz | 每个 MMU 都为 100 周期，1 GHz |
| 同时在途的页表遍历上限 | 共享 16 个 | 每 GPU 16 个 |
| 主机 fault 协议 | 未建模 | 未建模 |

**注意：独立 MMU 增加了整个系统的遍历并行度。** 时间差可能来自排队和并行度变化，不能全归因于映射命中率。两种模式都使用原有直接连接；这里没有补充真实 PCIe/NVLink 时延。

## 最简单的运行方式

以下均在 Linux 虚拟机执行：

```bash
cd /home/only/projects/mgpu-project/external/mgpusim
go build -o /tmp/mgpu-fir ./amd/samples/fir
mkdir -p ~/mgpusim-runs
mgpu_result=$(mktemp -d "$HOME/mgpusim-runs/my-ideal-XXXXXX")

/tmp/mgpu-fir -timing -verify -gpus=1,2 -use-unified-memory \
  -length=1024 -ideal-local-page-table -report-all \
  -metric-file-name="$mgpu_result/ideal"

printf '%s\n' "$mgpu_result"
```

`-gpus=1,2` 让 FIR 使用两块 GPU；`-use-unified-memory` 选择此版本的统一内存分配 API。当前分配器把统一内存页面放在 GPU 1；它不包含完整的按需迁移协议。GPU 2 可以通过已有远程访问路径读写 GPU 1 上的数据。**页表映射本地可用，不代表数据一定在本 GPU 显存里。**

`-ideal-local-page-table` 必须和 `-timing` 一起使用。启用该参数会自动输出 MMU 指标；`-report-all` 还输出 TLB、cache 等指标。单独 `-report-mmu` 可在共享模式下收集 MMU 指标。

不带 `-ideal-local-page-table` 就运行原来的共享 MMU 配置。

## 一条命令重跑双模式对照

```bash
cd /home/only/projects/mgpu-project/external/mgpusim
bash experiments/ideal-local/run.sh
```

脚本编译 FIR，依次执行共享模式和理想本地模式，并自动检查：

- 两次 FIR 计算结果均 `Passed!`。
- 共享模式有一个 MMU；理想模式有两个独立 MMU，且都完成过页表遍历。
- 两个 GPU 都有 TLB miss。
- 所有 MMU 的页表查询全部命中有效且非迁移中的映射。
- 页表遍历配置仍为 100 周期，实际平均翻译延迟没有被清零。

脚本在 `~/mgpusim-runs/ideal-local-XXXXXX/` 创建新目录，保存二进制、两份数据库、日志、检查结果、Git 版本、当前源码压缩包等。它不会覆盖以前的实验结果。

结果文件为 `shared.sqlite3` 和 `ideal.sqlite3`。可以用 DB Browser for SQLite 打开，执行：

```sql
SELECT Location, What, Value, Unit
FROM mgpusim_metrics
WHERE What LIKE 'mmu_%'
ORDER BY Location, What;
```

查看 TLB miss：

```sql
SELECT Location, What, Value, Unit
FROM mgpusim_metrics
WHERE Location LIKE '%TLB%' AND What = 'miss'
ORDER BY Location;
```

## 新增指标怎么读

| 指标 | 含义 |
|---|---|
| `mmu_ideal_local` | 1 为理想本地页表；0 为共享页表 |
| `mmu_ptw_configured_cycles` | 配置的遍历延迟，当前为 100 周期 |
| `mmu_frequency_hz` | MMU 频率，当前为 1 GHz |
| `mmu_max_requests_in_flight` | 这个 MMU 允许同时在途的遍历数 |
| `mmu_page_size` | 页大小，R9 Nano 配置为 4096 字节 |
| `mmu_ptw_completed` | MMU 接收并完成的翻译请求数，不等于所有层级 TLB miss 之和 |
| `mmu_translation_average_latency` | MMU 取出请求到完成响应的平均模拟时间；包含遍历和响应端反压，不含取出请求之前的排队时间 |
| `mmu_pt_lookup_count` | MMU 实际调用页表 Find 的次数，不包含驱动的查询 |
| `mmu_pt_hit_count` | 找到有效、非迁移中映射的次数 |
| `mmu_pt_missing_count` | 没找到页表项的次数 |
| `mmu_pt_invalid_count` | 找到页表项，但 Valid=false 的次数 |
| `mmu_pt_migrating_count` | 找到有效页表项，但 IsMigrating=true 的次数 |

Find 的四类结果互斥，合计等于 lookup_count。Akita MMU 可能在等待响应端口时重新查询页表，所以 **lookup_count 可能高于 ptw_completed**；不要把它当成唯一页数。

100 周期是 MMU 的倒计时配置。实际请求从接收到响应还有事件调度边界和可能的反压，因此平均时间不必精确等于 100 ns。

## 实现和研究边界

- `amd/timing/idealmapping/table.go`：驱动权威表、每 GPU 独立副本、即时同步和查询统计。
- `amd/samples/runner/timingconfig/ideallocal.go`：创建每 GPU 的 MMU，并把该 GPU 的 L2 TLB 连接到相应 MMU。
- `amd/samples/runner/mmureport.go`：把 MMU 指标写入原有 SQLite 结果表。

映射可用的假设针对已经合法分配的地址。理想模式遇到缺失、无效或迁移中的映射会报错，不会凭空生成物理地址，也不会静默返回旧映射。

本扩展同步的是页表副本，**没有实现 TLB shootdown、实际数据迁移或迁移期间请求阻塞**。如果以后添加迁移，必须另行接入正确的 TLB 失效、数据复制完成和恢复协议。这里的 Update 测试只证明页表副本被更新，不能证明整个 GPU 的缓存翻译已经一致。

所有页表结构是软件数据结构，保留的是上游 MMU 的固定延迟遍历抽象，没有新增多级 PTE 的真实显存访存或 page-walk cache。零成本同步是理想化假设；当前测试也没有验证 checkpoint/restore。

若要研究“基线发生主机 UVM fault，而理想模式没有”的加速比，仍需要另建具有每 GPU 映射状态、主机 fault 处理、失效与迁移协议的基线，或选择有这些功能且可验证的研究分支。

## 验证代码

```bash
go test -race ./amd/timing/idealmapping
go test ./amd/samples/runner/...
bash experiments/ideal-local/run.sh
```

表级测试覆盖 4 KB/2 MB 页面、两 GPU 同步、重映射、删除、PID 隔离、无效/迁移中状态、禁止回退查询权威表，以及并发读写。

## Git 与回退提醒

此目录由外层 `/home/only/projects/mgpu-project` 仓库管理。本次检查发现原有 `.gitignore` 的 `fir` 规则也忽略了 `amd/samples/fir/` 源码目录，所以 Git 提交不是完整源码备份。

已额外重建并保存修改前完整源码：

```text
/home/only/mgpusim-runs/ideal-local-20260922/before-source.tar.gz
```

其中已跟踪文件来自本次起点提交 `3cfc5b895a6eb0fad305c51cb8158528fca47d1c`；缺少的 109 个原有文件由未改动的工作区补齐。该备份已单独编译并运行双 GPU FIR，通过校验。回退时建议先解压到新目录比较，避免覆盖后来新增的工作。
