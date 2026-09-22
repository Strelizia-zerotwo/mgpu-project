# 本次实际验证记录

日期：2026-09-22。执行环境：用户的 VirtualBox Ubuntu 虚拟机。

源码目录：`/home/only/projects/mgpu-project/external/mgpusim`。

起点为外层仓库提交 `3cfc5b895a6eb0fad305c51cb8158528fca47d1c`，Go 1.27.1，Akita v5.0.0-beta.10。

## 已执行并通过

- `go build ./...`：所有包编译通过。
- `go test -race ./amd/timing/idealmapping`：通过，包括并发检查。
- `go test ./amd/samples/runner/...`：通过；部分子包没有测试文件。
- `golangci-lint run ./amd/timing/idealmapping/... ./amd/samples/runner/... --timeout=10m`：0 issues。
- `git diff --check`：通过。
- `bash experiments/ideal-local/run.sh`：完整流程通过。
- 单 GPU、普通显存分配、FIR length=64、理想本地页表、仅 `-report-mmu`：计算校验与 MMU 结果检查通过。

全仓库 `golangci-lint run ./... --timeout=10m` 没有通过：原有 `amd/benchmarks/dnn/training/optimization/sgd_test.go` 引用未提供的 `MockOperator`、`MockLayer`、`MockTensor` 等类型，触发 typecheck 失败。此问题不在本次新增代码中；没有据此宣称全仓库测试或 lint 已通过。

## 双 GPU 实测结果

R9 Nano 默认配置，FIR length=1024，`-gpus=1,2 -use-unified-memory -timing -verify -report-all`。

| 配置 | 完成页表遍历 | 有效页表命中/查询 | 缺失/无效/迁移中 | 平均 MMU 翻译延迟 |
|---|---:|---:|---:|---:|
| 共享 MMU | 13 | 13/13 | 0/0/0 | 101 ns |
| 理想本地 GPU 1 MMU | 7 | 7/7 | 0/0/0 | 101 ns |
| 理想本地 GPU 2 MMU | 6 | 6/6 | 0/0/0 | 101 ns |

两种配置：GPU 1 各层 TLB miss 合计 16，GPU 2 合计 14；Driver 内核时间均为 **7.808 µs**。TLB miss 合计跨多个层级，不代表 16/14 个不同页面。

页表遍历配置保持 100 周期、1 GHz。101 ns 是从 MMU 接收请求到完成响应的实测值，包含事件调度边界。

最终复现目录：

```text
/home/only/mgpusim-runs/ideal-local-U2XTCf/
  shared.sqlite3
  ideal.sqlite3
  shared.log
  ideal.log
  check.txt
  fir
  source.tar.gz
  commit.txt
  go-version.txt
  working-tree.diff
  config.txt
```

这个结果证明本次 workload 中：每个 GPU 使用其独立本地页表，所有查询命中，TLB miss 和页表遍历时延仍存在。它没有测量真实 NVIDIA GMMU 或真实 UVM fault 的性能收益，也没有覆盖完整迁移/失效流程。

## 与修改前源码的回归比较

使用单独重建的修改前源码编译运行同样的双 GPU FIR。修改后的默认共享模式与修改前的 1869 条原有指标均在浮点容差内一致，计数要求严格相等；46 条浮点指标有微小差别。比较采用相对容差 `1e-10`；秒单位绝对容差 `1e-18`，其他浮点指标绝对容差 `1e-10`。新增加 12 条 MMU 指标。

原始回归结果与修改前完整源码备份在：

```text
/home/only/mgpusim-runs/ideal-local-20260922/
  before-source.tar.gz
  before.sqlite3
  before.log
  regression.txt
```

尚未验证 MI300X 运行、迁移中的 TLB 一致性、检查点恢复或多工作负载性能结论。
