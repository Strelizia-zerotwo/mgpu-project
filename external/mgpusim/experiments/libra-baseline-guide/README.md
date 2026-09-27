# 从当前 MGPUSim 到 LIBRA 配置的四 GPU 实验

本教程依据 2026-09-27 对虚拟机源码的只读检查。源码位于 `/home/only/projects/mgpu-project/external/mgpusim`，当时 Git HEAD 为 `7c09be3d28637f87b0f343bee844a54de159b932`，检查前工作区干净。本轮没有修改模拟器或应用源码，也没有启动新的应用模拟。新增的说明和结果读取工具放在 `experiments/libra-baseline-guide/`。

目标是四个模拟 GPU，使用 LIBRA 的表 III 配置与表 IV 中的 DNN 应用，输出每 GPU 的 L3 TLB 命中率。先前已完成的是固定数据位置的映射缺页基线；**四 GPU DNN + UVM + 三级 TLB 版本尚未实现，不能靠下面的入门命令直接得到目标实验。**

## 1. 先认识三个不同的修改位置

| 你想改变什么 | 在哪里改 | 是否重编译 |
|---|---|---|
| 选哪些 GPU、输入张量大小、是否输出统计 | 运行二进制时的命令行参数 | 否 |
| 已存在缓存的容量/路数、已存在模块的参数 | `amd/samples/runner/timingconfig/` 下的 Go 构建代码 | 是 |
| 增加 L3 TLB、改变共享范围、增加子项、让应用支持 UVM 和四 GPU | 新模块与应用代码、分配代码、连接代码 | 是；还需要功能和计时验证 |

`-gpus=1,2,3,4` 表示选择四个独立模拟 GPU，不会自动将一个单卡应用切成四份。`-use-unified-memory` 会调用应用的 `SetUnifiedMemory()`，是否真的采用统一内存还取决于该应用的实现。`-unified-gpus` 是另一种逻辑 GPU 模式，不是开启 UVM 的开关；之前的 demand/ideal 模式也不允许混用它。

## 2. 论文里有哪些 DNN 应用，当前能用哪些

LIBRA 中文 PDF 第 8 页的表 IV 列出了以下 DNN 相关应用。表格没有逐行标注所属套件，不能把所有神经网络模型都直接称为原版 DNNMark 已提供的应用。

| 论文缩写 | 论文所列每 GPU 内存占用 | 当前源码入口 | 本次静态检查结果 |
|---|---:|---|---|
| C2D | 23 MB | `amd/samples/conv2d/main.go` | 应用只支持单 GPU；UVM 标志未接入张量分配 |
| IM2COL | 20 MB | `amd/samples/im2col/main.go` | 应用只支持单 GPU；UVM 标志未接入张量分配 |
| LeNet | 6 MB | `amd/samples/lenet/main.go` | 有多 GPU 数据并行训练，但开启 UVM 会 panic |
| VGG | 55 MB | `amd/samples/vgg16/main.go` | 有多 GPU 数据并行训练，但开启 UVM 会 panic |
| BERT-T / BERT-M / BERT-ME / BERT-B | 68 / 136 / 272 / 544 MB | 在当前 `amd/` 下按 bert/gpt 名称查找未发现 | 需要额外适配，不代表库中任何地方都绝对不存在相关算子 |
| GPT2-M / GPT2 | 65 / 196 MB | 同上 | 需要额外适配 |

论文说使用默认输入集，但没有提供足以将这些 MB 数值唯一还原为张量维度/批量大小的配置文件。上游同名应用也不能自动视为论文的同规模、同划分实现。SC（简单卷积）也是表 IV 的一项，但不因名字里有卷积就自动归入 DNNMark。

具体阻碍可以自己查看：

```bash
cd /home/only/projects/mgpu-project/external/mgpusim
grep -n 'can only run on a single GPU' amd/benchmarks/dnn/layer_benchmarks/{conv2d,im2col}/benchmark.go
grep -n 'unified memory is not supported' amd/benchmarks/dnn/training_benchmarks/{lenet,vgg16}/benchmark.go
grep -n 'AllocateMemory\|FreeMemory' amd/benchmarks/dnn/gputensor/operator.go
```

还有一个后续适配必须处理的问题：DNN 算子会释放临时缓冲，旧 demand/ideal 基线在开始翻译后禁止重映射和释放。直接把 `AllocateMemory` 换成 `AllocateUnifiedMemory` 并删除 panic 不能解决这一问题，必须保证释放/地址复用不会保留旧 TLB 映射。不能简单去掉保护检查。

## 3. 第一次由你操作：编译、看参数、跑一个小的现有 DNN

以下都是在 Ubuntu 终端执行。这个练习使用现有 IM2COL 的单 GPU 普通内存路径，只学习编译和结果读取；不是正式四 GPU UVM 实验。

```bash
cd /home/only/projects/mgpu-project/external/mgpusim
export PATH=/usr/local/go/bin:$PATH
go version
git status --short

mkdir -p "$HOME/mgpusim-runs"
mgpu_lab=$(mktemp -d "$HOME/mgpusim-runs/im2col-learn-XXXXXX")
printf '%s\n' "$mgpu_lab"
go build -o "$mgpu_lab/im2col" ./amd/samples/im2col
"$mgpu_lab/im2col" -help
```

`go build` 生成可执行文件；`-help` 只列出选项，不执行计算。`git status --short` 若没有输出，表示当时 Git 工作区没有可报告的改动。新结果目录每次名字不同，避免覆盖以前的结果。换终端后，需要将 `mgpu_lab` 重新设置成刚才输出的绝对路径。

编译成功后再执行：

```bash
"$mgpu_lab/im2col" \
  -timing -gpu=r9nano -arch=gcn3 -gpus=1 -vm-mode=shared \
  -N=1 -C=1 -H=8 -W=8 \
  -kernel-height=3 -kernel-width=3 \
  -verify -report-all -disable-rtm \
  -metric-file-name="$mgpu_lab/before"
```

N/C/H/W 分别表示批量、通道、高、宽。该应用的 `-verify` 会让 runner 调用算子的验证开关；应用末尾的 `Verify()` 本身为空，因此不要以是否恰好打印 FIR 的 `Passed!` 字样作为唯一依据。算子验证会增加 CPU/GPU 数据复制，验证运行与最终性能测量要分开。

结果写到 `before.sqlite3`。读取当前实际存在的 L2 TLB：

```bash
python3 experiments/libra-baseline-guide/report_tlb.py \
  "$mgpu_lab/before.sqlite3" --level L2 --gpus 1
```

这组命令已按当前源码参数核对，本轮尚未实际执行该 IM2COL 模拟。出现错误应先记录完整输出，不应换成四 GPU 命令跳过错误。

## 4. 第一次改硬件配置：只改一项容易观察的缓存参数

例如先练习把 L2 数据缓存从 16 路改为 8 路。用文本编辑器打开：

```bash
nano amd/samples/runner/timingconfig/r9nano/builder.go
```

在 `buildL2Caches()` 函数内，把：

```go
spec.WayAssociativity = 16
```

改为：

```go
spec.WayAssociativity = 8
```

只改这个函数中的这一项，然后保存。这里的 L2 是数据缓存，不是 L2 TLB。源码默认 L2 总容量已经是 2 MB；容量被分到多个 bank，并非每个 bank 都有 2 MB。

```bash
gofmt -w amd/samples/runner/timingconfig/r9nano/builder.go
git diff -- amd/samples/runner/timingconfig/r9nano/builder.go
go build -o "$mgpu_lab/im2col-after" ./amd/samples/im2col
```

再用相同输入参数运行 `im2col-after`，把输出前缀换成 `$mgpu_lab/after`。你会完成“改 Go 配置 -> 看差异 -> 重新编译 -> 同输入对比”的完整过程。小输入可能没有性能变化，这不代表配置没有生效。

不要一次更改全部参数，否则很难判断是哪一项引入了问题。修改后的路径仍是当前 AMD 模型；该练习不等于实现完整 LIBRA 架构。

## 5. 表 III 每项配置的实际入口

以下文件均相对仓库根目录。

| 目标 | 当前入口和状态 | 如何处理 |
|---|---|---|
| 4 GPU | `amd/samples/runner/flag.go` 的 `-gpus` | 应用支持后使用 `-gpus=1,2,3,4` |
| 1 GHz | `timingconfig/r9nano/builder.go` 的 `MakeBuilder()`，默认 `freq: 1 * timing.GHz` | 当前已经是 1 GHz；仍需检查新增组件使用同一时钟 |
| 108 SM | 同文件的 `numCUPerShaderArray`、`numShaderArray`；平台 builder 还有注册给 driver 的 CUCount | 当前为 16×4 个 AMD CU。改变乘积并不能实现 SM/TPC/GPC，且实际构建数量与 driver 的 CUCount 必须一致 |
| L1 D 64 KB、4 路 | `timingconfig/shaderarray/builder.go` 的 `buildL1VCaches()` | 当前每 CU 的 vector cache 默认 16 KB、4 路；可将默认 `l1vSize` 改成 `64 * mem.KB`，但这只改变容量，不改变共享关系，也不等于复制 NVIDIA 的 L1D |
| L1 I 32 KB、4 路 | 同文件的 `buildL1ICache()` | 数值已相同，目前按 Shader Array 共享；论文目标的共享范围仍需核对 |
| L2 2 MB、8 路 | `r9nano/builder.go` 的 `MakeBuilder()` 与 `buildL2Caches()` | 容量默认 2 MB；路数可按上一节修改 |
| TPC 共享 L1 TLB | `shaderarray/builder.go` 的 `buildL1VTLBs()` 和地址翻译器连接 | 当前每 CU 建一个 vector TLB，不能只改项数。需按两个执行核心连接一个目标 L1 TLB |
| GPC 共享 L2 + GPU 共享 L3 | `r9nano/builder.go` 的 `buildL2TLB()`、连接函数 | 当前单级 L2 直接接 MMU；必须增加层级和对应连接 |
| 每项 16 个子项 | Akita TLB 组件和目录实现 | 需要子项有效位、查找、填充、替换及失效逻辑，不是把容量直接乘以 16 |
| 每 GPU 8 个 PTW | `amd/samples/runner/faultflags.go` | `-vm-local-walkers=8` 已存在；是现有抽象查询器的并发上限 |
| 每级 100 周期 | 同文件、`amd/timing/faultvm/gpu.go` | `-vm-local-walk-cycles=100` 是整次查询 100 周期，不是每级 100 周期；需补充级数和逐级行为 |
| DRAM 为足迹的 70% | `timingconfig/builder.go`、`r9nano/builder.go`、driver 分配器与页面放置 | 需同时保持容量、物理地址布局、地址路由和 allocator 一致；不能只改其中一个大小 |
| 300 / 32 GB/s 网络 | `timingconfig/builder.go` 的 `createConnection()` | 当前使用 DirectConnection；没有可直接填写两个带宽的配置项。需按字节量和共享链路资源计时 |

两个重要细节：

- `buildL1VTLBs()` 留有针对旧 Akita beta.2 的注释，记录 `Latency=1` 曾触发流水线停滞；当前依赖已经是 beta.10。这是需重新验证的兼容性风险，不能直接把现在的 2 改成 1 然后宣称满足一周期时延，也不能凭旧注释断言 beta.10 必然有相同问题。
- 顶层 builder 的字段名不代表参数已经接到所有子模块。例如核心数量还出现在 driver 注册信息中。修改后要检查实际实例数量和连接，不能只检查配置文本。

## 6. 正式四 GPU UVM DNN 实验需要补的实现

建议先选 IM2COL，再逐步覆盖 C2D、LeNet、VGG，最后考虑当前没有入口的 BERT/GPT。它们都是论文中的应用，但应明确当前版本与论文版本的输入和实现差异。

1. **应用分工。** 在 IM2COL 的应用层按批量或输出范围划分四 GPU 的工作，明确各 GPU 的输入/输出区间、kernel launch 和同步。共享输入时不要把复制出的四份独立分配称为跨 GPU 共享页面。
2. **统一内存接线。** 把应用的开关传到 GPUOperator，明确哪些张量走 `AllocateUnifiedMemory`，哪些元数据/临时缓冲保持显式分配，并统计两类分配。不能仅将布尔变量设为 true。
3. **内存生命周期。** 解决运行中临时张量释放与旧固定映射保护的冲突。可采用有明确占用统计的预分配复用方案，或实现正确的失效/释放协议；不能靠不释放所有缓冲来伪装论文的 70% 容量设置。
4. **翻译结构。** 实际加入 TPC/GPC/L3 共享结构和子项 TLB。论文第 2 页明确两 SM/TPC，108 SM 因而对应 54 个 TPC；GPC 数量未在已检查正文/表 III 中给出，采用值需单独标为模型假设。
5. **页面放置与网络。** 说明默认缺页处理是安装远端映射还是迁移到本地，确保与可用物理页和网络计时一致。用户研究的是新的缺页方法，不需要默认实现 LIBRA 预测器。
6. **统计和正确性。** 验证四 GPU 都执行非零工作，输出正确；每 GPU 的 L3 hit/miss/合并次数可信；容量未越界、请求不丢失、迁移/复用后无过时翻译。

因此，当前不能提供一条真实有效的 `im2col -gpus=1,2,3,4 -use-unified-memory ...` 命令来完成目标；这条命令在现有代码中会被应用的单 GPU 检查拒绝。

## 7. 每 GPU 的 L3 TLB 命中率怎样读

现有报告分别记录 `hit`、`miss`、`mshr-hit`。MSHR-hit 是合并到同页在途翻译，不表示 TLB 中已有可立即使用的映射。沿用三个互斥事件的口径时：

```text
真正驻留映射的命中率 = hit / (hit + miss + mshr-hit)
```

待 L3 模块接线并产生这些统计后，运行：

```bash
python3 experiments/libra-baseline-guide/report_tlb.py \
  /实际结果目录/metrics.sqlite3 --level L3 --gpus 1,2,3,4
```

脚本为每 GPU 分别输出原始计数和命中率。它不会把没有 L3 组件、没有采样或 GPU 没有请求的情况伪造为 0%/100%。计数缺失会报错；存在完整计数但总数为零时显示 N/A。

该脚本读取数据库中的整个统计区间。如果要排除热身/初始化阶段，应在模拟器中定义统计开始与结束位置，不能用最终 kernel time 的窗口替代 TLB 计数的窗口。

## 8. 关于 70% 内存容量

表 IV 的列头明确是“每 GPU 内存占用”。若暂按该值乘 0.7，C2D、IM2COL、LeNet、VGG 的参考容量分别为 16.1、14、4.2、38.5 MB；这只是由表格算出的目标值，不是对当前运行规模测量后的结果，最终还需按页面大小取整。

每 GPU 的访问足迹可能包含同一份共享数据，因此把四个足迹相加未必等于全系统去重后的已分配数据量。70% 不能单独证明全系统超额订阅率就是 1/0.7；远端 GPU 也可能持有可访问页面。论文第 11 页另设 125% 和 150% 全系统超额订阅实验，并明确超出的数据放在 CPU 内存。

当前基线不支持超额订阅换页。如果目标实验确实需要 CPU 后备页面或运行中重定位，需要补充相应机制；若只研究固定放置下取得远端映射的缺页，就必须明确其与论文迁移环境的差异。

## 9. 保存自己的每一步

每次先只改一个因素，编译后用独立前缀保存结果。记录 `git diff`、输入参数、硬件参数和数据库。在正式比较自己的新方法时，两边使用相同的应用划分、资源容量、输入和统计窗口。

本轮提供的是源码核对、操作教程和统计读取工具。尚未改出符合 LIBRA 表 III 的四 GPU DNN 模型，也没有新运行的四 GPU DNN 性能或命中率结果。
