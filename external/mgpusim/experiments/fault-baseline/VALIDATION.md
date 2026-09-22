# 映射缺页基线：验证记录

验证日期：2026-09-23。以下数据来自 Linux 虚拟机中的实际执行，不是根据参数计算出的预测值。

## 代码与实验定位

- 源码：`/home/only/projects/mgpu-project/external/mgpusim`
- 外层 Git 仓库：`/home/only/projects/mgpu-project`
- 外层 HEAD：`3cfc5b895a6eb0fad305c51cb8158528fca47d1c`
- 修改未提交；HEAD 本身不包含这些改动，也不能独立标识本次实验版本。
- Go：1.27.1；Akita：`github.com/sarchlab/akita/v5 v5.0.0-beta.10`。
- 三组实验结果：`/home/only/mgpusim-runs/fault-baseline-XWYzye/`
- 模块检查和兼容性结果：`/home/only/mgpusim-runs/fault-baseline-20260923/`

这是供新缺页处理方法使用的研究基线，不是 ShadowUpdate 复现，也未校准真实 GPU 的 UVM 服务时延。使用方法及全部参数见同目录 `README.md`。

## 真实 FIR 对照结果

运行命令：

```bash
cd /home/only/projects/mgpu-project/external/mgpusim
bash experiments/fault-baseline/run.sh
```

工作负载为 FIR，`-length=16384`、`-gpus=1,2`、`-use-unified-memory`、`-timing`、`-verify`。三组都通过数值校验 `Passed!`。

本地遍历 100 周期，每 GPU 8 个遍历器、64 个事务槽，每事务最多合并 64 个请求；主机服务 400 周期；单向传输延迟 200 周期；安装映射 100 周期。模型时钟 1 GHz，每周期 1 ns。

| 模式 | 主机服务槽 / 等待队列 | kernel time（µs） | 本地映射缺页总数 | 主机内部队列平均等待（ns） | 队列占用峰值 | 服务槽占用峰值 | 队列满反压周期 |
|---|---|---:|---:|---:|---:|---:|---:|
| demand 基线 | 16 / 64 | 9.710 | 36 | 9.167 | 2 | 16 | 0 |
| ideal 理想映射 | 16 / 64 | 7.925 | 0 | 0 | 0 | 0 | 0 |
| congested 拥塞检查 | 1 / 2 | 19.789 | 36 | 765.389 | 2 | 1 | 13161 |

`demand` 与 `ideal` 的硬件结构和容量参数相同，区别在于统一内存映射是否预先安装。`congested` 改变主机资源，用于检查竞争机制，因此不应把它和 demand 的差值当成某种缺页优化方法的收益。

主机内部队列平均等待只计算请求进入内部队列之后的等待；入队前反压另记为 `vm_ingress_wait_average`。并发请求的延迟会重叠，不能用缺页总数乘以平均缺页延迟计算 kernel 时间。

同一目录中的 `commands.sh` 保存每一组的完整实际参数，`summary.json` 保存上表的数值，`check.txt` 保存守恒和容量检查结果。数据库为 `demand.sqlite3`、`ideal.sqlite3`、`congested.sqlite3`。

## 验证覆盖

| 检查 | 实际结果 |
|---|---|
| `go build ./...` | 通过 |
| `go test -race ./amd/timing/faultvm ./amd/timing/idealmapping` | 通过 |
| `go test ./amd/samples/runner/...` | 通过 |
| `golangci-lint run ./amd/timing/faultvm/... ./amd/timing/idealmapping/... ./amd/samples/runner/... --timeout=10m` | 0 issues |
| `git diff --check` | 通过 |
| 三组 FIR 数值校验 | 全部通过 |
| `experiments/fault-baseline/check.py` | 全部断言通过 |
| 旧 shared 模式改动前后对比 | 1,881 项指标一致；两次数值校验通过 |
| 原有 `-ideal-local-page-table` 模式 | 双 GPU FIR 数值校验通过 |
| 全仓库 `golangci-lint run ./... --timeout=10m` | 未通过：未修改的 DNN training 测试缺少 MockDataSource、MockLayer、MockLossFunction、MockAlg 等类型，typecheck 无法完成 |

未把全仓库 lint 写成通过，也没有为解决该问题修改无关的 DNN 测试。

模块测试包含：同 GPU 同页请求合并、返回的物理映射正确性、安装后热命中、不同 GPU 映射隔离、PID 隔离、普通分配预映射、有限队列与反压、服务槽数量对完成时间的影响、非法地址失败、过时映射拒绝、运行中重映射/释放拒绝。

真实 FIR 检查包括：两个 GPU 都仍有 TLB miss 和本地遍历；demand 两个 GPU 都实际缺页；ideal 没有主机请求；翻译请求/响应守恒；本地遍历数等于命中数加缺页数；主机接收/完成数与本地缺页数一致；队列/服务槽未超过容量；仿真结束无遗留事务；主机服务时间符合配置；拥塞情况下实际产生队列等待和反压。

兼容性对比使用 `-length=1024`、`-gpus=1,2`、统一内存、timing、verify 和 report-all。改动前后二进制的 Driver kernel time 都是 7.808 µs。count 指标严格相等；浮点指标比较使用相对容差 `1e-10`，绝对容差在 second 单位下为 `1e-18`，其他单位为 `1e-10`；NULL 只与 NULL 相等。

## 模型边界

- **模拟固定数据位置上的首次本地映射缺页。** 安装映射不迁移数据；当前统一内存分配在 GPU 1，GPU 2 通过既有数据路径访问。
- **不是理想 TLB。** TLB miss 和本地遍历都仍然存在；本地/主机遍历用固定服务时延建模，没有生成多级 PTE 的缓存与 DRAM 访问。
- **会发生资源竞争。** 主机服务槽、队列、端口缓冲及 GPU 在途事务均有限；链路延迟是抽象参数，没有建模真实 PCIe/NVLink 包流或带宽共享。
- **不是完整 UVM。** 没有迁移、复制、TLB shootdown、跨 GPU 失效广播、映射驱逐或迁移后的重复缺页。首次实际翻译后重映射或释放会明确报错，避免静默使用旧映射。
- **时延未做硬件校准。** 400 ns 的主机服务、200 ns 的单向链路等都是起始假设。本次速度差只能说明这个模型及参数下的行为。
- 目前验证了 FIR 和模块测试；未验证其他所有基准、检查点恢复或并行仿真引擎。

## 备份与可审查改动

修改前源码完整备份：

```text
/home/only/mgpusim-runs/fault-baseline-20260923/source-before.tar.gz
```

三组实际运行时源码快照、二进制和命令位于：

```text
/home/only/mgpusim-runs/fault-baseline-XWYzye/source.tar.gz
/home/only/mgpusim-runs/fault-baseline-XWYzye/fir
/home/only/mgpusim-runs/fault-baseline-XWYzye/commands.sh
```

本轮最终增量补丁、所有发生变化的文件及校验清单位于：

```text
/home/only/mgpusim-runs/fault-baseline-20260923/baseline.patch
/home/only/mgpusim-runs/fault-baseline-20260923/baseline-final-files.tar.gz
/home/only/mgpusim-runs/fault-baseline-20260923/final-files.sha256
```

补丁以本轮 `source-before.tar.gz` 为基础，包含新增文件；它不是相对干净上游 HEAD 的补丁，不应在已修改的当前目录重复应用。原来的理想映射扩展已包含在修改前备份中。最终说明文档在三组运行之后补充，数值以保留的运行快照、日志和数据库为准。
