# SC 原版对照与驱动停滞修复

项目：`/home/only/projects/mgpu-project`，WSL 发行版 `only`。

## 三个运行入口

在 WSL 终端执行：

```bash
cd /home/only/projects/mgpu-project

# 未修改的官方源码：用于复现旧问题，四 GPU 小输入已确认会停滞
bash "benchmark/original/sc.sh"

# 当前源码的 shared 路径：包含本次驱动修复，不启用 demand 或 L3
bash "benchmark/original/sc-fixed.sh"

# 当前源码的 demand-l3 路径：用于继续四 GPU UVM + L3 实验
bash "benchmark/l3 tlb/sc.sh"
```

三个入口默认都是四 GPU、GCN3 / R9 Nano、UVM、时序模拟、结果验证、62×62 输入、3×3 卷积核。每次自动编译。
原版已知会复现等待问题，若重新运行后停住，可用 Ctrl+C 停止该次运行。已保存诊断证据，无需为了查看结果再次等待。

输入增大示例：

```bash
bash "benchmark/original/sc-fixed.sh" -width=126 -height=126
bash "benchmark/l3 tlb/sc.sh" -width=126 -height=126
```

结果分别保存到：

```text
results/original/sc/时间_62x62_mask-3/          # 未修改的官方版本
results/original/sc-fixed/时间_62x62_mask-3/    # 当前源码 shared 模式
results/l3 tlb/sc/时间_62x62_mask-3/            # 当前源码 demand-l3 模式
```

`latest` 只指向成功完成的运行。原版失败结果没有有效的最终指标。

```bash
cat "results/original/sc-fixed/latest/status.txt"
cat "results/original/sc-fixed/latest/l2-hit-rate.txt"
cat "results/l3 tlb/sc/latest/status.txt"
cat "results/l3 tlb/sc/latest/l3-hit-rate.txt"
cat "results/l3 tlb/sc/latest/check.log"
```

`sc-fixed.sh` 是当前修改源码的 shared 模式，不能称作逐文件未修改的官方版本；原版入口 `sc.sh` 使用独立源码目录。
`sc-fixed.sh` 固定 shared；L3 入口固定 demand-l3，防止把无 L3 的结果混入 L3 实验。

## 原版来源

- 官方仓库：<https://github.com/sarchlab/mgpusim>
- 固定提交：`840e0b6440059d24f94141dda625351a36727ec9`，不是浮动的最新分支。
- 独立源码：`external/mgpusim-original/`，未应用修复。
- 压缩包 SHA256：`3018d4c9d135f79af16b9cb62267fca5f4b1082b11dd921a559fd7279a63c16f`。
- 下载地址和校验信息：本目录 `source.json`。
- 已将解压源码的 1,018 个文件逐一与归档比较，无更改或缺失。

独立原版源码与原始结果被 Git 忽略。今后在新机器恢复时，在项目根目录执行以下命令；目标目录必须不存在，防止覆盖已有文件：

```bash
curl -fL --retry 3 \
  https://codeload.github.com/sarchlab/mgpusim/tar.gz/840e0b6440059d24f94141dda625351a36727ec9 \
  -o /tmp/mgpusim-original-840e0b6.tar.gz
printf '%s\n' '3018d4c9d135f79af16b9cb62267fca5f4b1082b11dd921a559fd7279a63c16f  /tmp/mgpusim-original-840e0b6.tar.gz' | sha256sum -c -
mkdir external/mgpusim-original && \
  tar -xzf /tmp/mgpusim-original-840e0b6.tar.gz --strip-components=1 -C external/mgpusim-original
```

确认 SHA256 检查通过后再解压。当前 WSL 已有该目录，不需要重新下载。

## 根因与修改

旧 `amd/driver/memorycopy.go` 只在处理数据拷贝回复时检查“所有请求是否完成”。同一个拷贝命令还可以包含发往多个 GPU 的缓存刷新请求。

若数据拷贝先完成，而某个 GPU 的刷新回复最后到达，`processFlushReturn()` 只删除请求，不把命令出队、不清除 `IsRunning`。
于是出现：请求列表为空、没有待处理模拟事件，但命令队列一直标记为运行中。程序最终睡眠在 `DrainCommandQueue()`。

本次现场：GPU 1 卡在 H2D 拷贝命令，`Reqs=[]`、`IsRunning=true`，后续内核尚未启动；GPU 2–4 各自的 16 个工作组已经完成。
CU 中没有活跃 wavefront，CP 没有未完成工作。这是驱动完成逻辑遗漏，继续等待不会完成。

修复位于当前 `external/mgpusim/amd/driver/memorycopy.go`：

- H2D、D2H、缓存刷新回复统一检查是否收齐全部请求。
- 全部完成后将命令出队并清除 `IsRunning`。
- D2H 的最后回复即使来自刷新，也会正确解码到主机目标缓冲区。
- 去掉刷新回复的一次重复 tracing finalize。

新增测试位于 `amd/driver/driver_test.go`：覆盖 H2D / D2H 两种方向、一个拷贝回复与两个刷新回复的全部 6 种顺序，共 12 个用例；检查不会提前完成、下一个命令可以继续、D2H 数据正确。

临时 CU / CP / 驱动状态诊断代码已移除。SC 应用源码、GPU 内核二进制、TLB 容量与时序参数未因本次修复而修改。
所有当前源码模式共用这个驱动修复；独立原版源码保留旧行为。

## 2026-09-30 实际验证

操作前项目 HEAD 为 `796c6ea`；Go 1.27.1。本轮未创建 Git 提交。

| 配置 | 输入 | 结果 |
|---|---|---|
| 独立官方版本，四 GPU UVM | SC 62×62 | 复现停滞，SIGQUIT 保存堆栈，失败结果保留 |
| 当前 shared，四 GPU UVM | SC 62×62 | Passed |
| 当前 demand，四 GPU UVM | SC 62×62 | Passed |
| 当前 ideal，四 GPU UVM | SC 62×62 | Passed |
| 当前 demand-l3，四 GPU UVM | SC 62×62 | Passed；L3 完成与容量检查通过 |
| 当前 demand-l3，四 GPU UVM | SC 126×126 | Passed；L3 完成与容量检查通过 |
| 当前 demand-l3，四 GPU UVM | FIR length=4096 | Passed；L3 完成与容量检查通过 |

SC 62×62 的 L3：GPU 1–3 各 7 miss，GPU 4 为 6 miss，各 GPU 的 hit / MSHR-hit 均为 0。
SC 126×126 的 L3：GPU 1–3 各 13 miss，GPU 4 为 12 miss，各 GPU 的 hit / MSHR-hit 均为 0。
这两个小输入的驻留命中率都是 0%；运行已正常结束，不能据此判断模拟器停滞。上层 L2 过滤重复翻译，L3 只收到这些冷缺失。

验证命令与结果：

- 用 Go overlay 临时让新测试编译官方旧 `memorycopy.go`，不修改任何源码文件：12 个回归用例中 8 失败、4 通过，失败情况全部为刷新最后回复。
- 修复后的 `go test -p 2 ./amd/driver`：通过，包含上述全部 12 个回归用例。
- `go test -race -p 2 ./amd/driver`：通过。
- 依照仓库要求安装 golangci-lint v2.13.2，生成被忽略的 mock 文件，执行 `golangci-lint run ./... --timeout=10m`：`0 issues.`。
- 原版归档逐文件校验：1,018 个文件全部一致。

测试所需 mock 可在模拟器目录用以下方式生成（本机已经生成）：

```bash
go install go.uber.org/mock/mockgen
export PATH="$(go env GOPATH)/bin:$PATH"
go generate ./amd/...
go test -race -p 2 ./amd/driver
```

关键证据位置（均相对项目根目录）：

```text
results/original/sc/2026-09-30_00-02-00_62x62_mask-3/run.log
results/original/sc/source/driver-stall-diagnostic.log
results/original/sc/source/regression-before.log
results/original/sc/source/regression-after.log
results/original/sc/source/regression-race.log
results/original/sc/source/lint.log
results/original/sc/source/upstream-integrity.txt
results/original/sc-fixed/2026-09-30_00-18-11_62x62_mask-3/
results/original/sc-fixed/validation-demand-62x62/
results/original/sc-fixed/validation-ideal-62x62/
results/l3 tlb/sc/2026-09-30_00-15-38_62x62_mask-3/
results/l3 tlb/sc/2026-09-30_00-16-38_126x126_mask-3/
results/l3 tlb/fir/2026-09-30_00-16-41_length-4096/
```

这验证了当前小规模 SC/FIR 的完成与计算正确性，不能推论全部应用或任意规模均无 bug；也不是完整 LIBRA 配置或真实 NVIDIA GPU 的验证。
