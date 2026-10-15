# Akita #529 与四 GPU 单级 TLB 验证

已核对 #529：当前为 OPEN，测试提交为 `fdeb737efed0ed6c59adcdffd53c87b5aa8d6158`。MGPUSim 官方 main 仍依赖 Akita `v5.0.0-beta.10`；本地测试基于 MGPUSim `a3bba0853776585f023956d41f4b723373e3cbee`。

## 测试范围

1. 原始 #529 源码：直接运行其 queueing、TLB 包测试及竞态检查，并执行完整 Go 构建，通过。
2. beta.10 回移：只将 TLB 的 `AcceptWithDelay(..., 1)` 改为 `Accept(...)`，保留其余 beta.10 API。保留额外等待的旧接口，不包含 #529 的 API 删除，因此这是核心行为回移测试。
3. 多 GPU：使用上游 MGPUSim 原有 Akita TLB 路径，将向量 L1 TLB 的流水线级数设为 1。scalar/instruction L1 和 L2 继续原有配置。四 GPU、UVM、详细计时模拟和结果验证。

不能将第2、3项描述成“MGPUSim 已直接兼容 #529 最新版”。最新 Akita 的目录及组件 API 与 beta.10 不同；这一轮没有迁移整个 MGPUSim。

研究版模拟器、akita-patched、原版参考目录，以及两个待提交 PR 的分支均未修改。这轮在新目录 `/home/only/projects/akita-529-validation-20261013/` 中工作。

## 你可以直接运行的命令

先重新运行真实 #529 的测试：

```bash
cd /home/only/projects/akita-529-validation-20261013/akita-529
/usr/local/go/bin/go test -race ./sim/queueing ./mem/vm/tlb -count=1
```

运行四 GPU UVM SC，向量 L1 为一级：

```bash
python3 "/home/only/projects/mgpu-project/benchmark/akita pr529/run.py" --app sc --variant fixed-1 --width 62 --height 62
```

运行四 GPU UVM FIR，向量 L1 为一级：

```bash
python3 "/home/only/projects/mgpu-project/benchmark/akita pr529/run.py" --app fir --variant fixed-1 --length 4096
```

脚本自动重新编译，执行 `-timing -verify -disable-rtm -gpus=1,2,3,4 -use-unified-memory -report-all`，然后检查结果。不能同时运行两个脚本实例，脚本带有并发保护。

如果想重复三种配置的对照：

```bash
python3 "/home/only/projects/mgpu-project/benchmark/akita pr529/run.py" --app all --variant all
```

| variant | Akita | 向量 L1 级数 |
|---|---|---|
| original-2 | 未修改的 beta.10 | 2 |
| fixed-2 | beta.10 + Accept 回移 | 2 |
| fixed-1 | beta.10 + Accept 回移 | 1 |

如果想调大 SC：

```bash
python3 "/home/only/projects/mgpu-project/benchmark/akita pr529/run.py" --app sc --variant fixed-1 --width 126 --height 126
```

本轮已经运行的是 SC62x62/mask3、FIR4096；上述126x126命令供你继续验证。

## 怎样判断验证成功

每个用例的 run.log 中必须有 Passed!，命令退出码必须为0。脚本还会从 SQLite 的 component_spec 和 mgpusim_metrics 中确认：

- 四张 GPU 各有64个向量 L1 TLB，实际保存的 latency 与选定级数一致；
- 四张 GPU 的 CommandProcessor 都有大于0的 kernel_time；
- 四张 GPU 都有大于0的向量 L1 hit/miss/mshr-hit 总数。

只有这些检查都通过才打印 PASS。每个输出目录保存 build.log、run.log、validation.json、command.json、akita-module.json、实际 builder 源码和 backport.patch。

输出目录统一为：

```text
/home/only/projects/mgpu-project/results/akita pr529/sc/<时间与规模>/<variant>/
/home/only/projects/mgpu-project/results/akita pr529/fir/<时间与规模>/<variant>/
```

## 本轮实际结果

| 用例 | original-2 | fixed-2 | fixed-1 |
|---|---|---|---|
| 四 GPU UVM SC62x62/mask3 | PASS | PASS | PASS |
| 四 GPU UVM FIR4096 | PASS | PASS | PASS |

以上6个结果都检查了四张GPU的实际配置与活动。固定单级后没有观察到停滞，计算结果正确。

其他验证：原始 #529 目标包竞态检查通过、完整Go构建通过；回移版queueing/TLB目标包竞态检查通过。回移依赖lint使用旧版兼容配置（只适配新工具新增的exhaustruct_v5、govet inline检查）；MGPUSim测试副本lint仅跳过与测试目的冲突的禁止本地replace检查。两者最终均为0 issues。这些临时lint适配不属于待提交PR。

本轮没有声称所有程序、规模均已验证，也没有将模拟结果视为真实GPU测量。
