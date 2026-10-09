# 多 GPU 论文阅读与模拟实验

本仓库保存多 GPU 相关论文、模拟器源码、实验入口及结果。
当前工作目录是 `/home/only/projects/mgpu-project`。

## 目录约定

```text
mgpu-project/
├── mgpu-paper/                 # 论文资料
├── docs/                       # 部署、阅读和实验说明
├── external/mgpusim/           # MGPUSim 与内置应用源码
├── external/triosim/           # TrioSim 源码
├── benchmark/
│   └── l3 tlb/                 # L3 TLB 的配置和统一运行入口
│       ├── fir.sh
│       ├── sc.sh
│       ├── run.sh
│       ├── run.py
│       └── config.json
├── results/
│   └── l3 tlb/                 # 原始输出与统计汇总，按应用分目录
│       ├── fir/                # 每轮目录及 latest 链接
│       └── sc/
├── experiments/triosim-baseline/ # 已有 TrioSim 实验入口
└── runs/triosim-baseline/        # 已有 TrioSim 结果
```

新建实验的运行脚本与配置放在 `benchmark/<实验类别>/`，输出放在
`results/<实验类别>/<应用>/`。目录名包含空格时，命令中必须加引号。
已有 TrioSim 文件保留原位；这里的 L3 TLB 入口采用新约定。
模拟器自带的应用源码继续放在 `external/mgpusim/amd/benchmarks/` 和 `amd/samples/`，由脚本调用。

## 四 GPU UVM + L3 TLB

```bash
cd /home/only/projects/mgpu-project
bash "benchmark/l3 tlb/fir.sh" -length=4096
bash "benchmark/l3 tlb/sc.sh" -width=62 -height=62
cat "results/l3 tlb/fir/latest/l3-hit-rate.txt"
cat "results/l3 tlb/sc/latest/l3-hit-rate.txt"
```

一条运行命令自动编译、模拟、验证计算结果并读取 L1/L2/L3 命中率。
当前实验默认 `-tlb-profile=libra-capacity`，按论文表对齐每个实例的容量和延迟；共享组织仍为 AMD 模型。`-tlb-profile=legacy` 可恢复原有 L1/L2 配置。见 [容量与范围说明](benchmark/l3%20tlb/PAPER-TLB.md)。
FIR 已通过 WSL 验证；SC 的驱动等待问题已修复，62×62、126×126 四 GPU UVM + L3 运行通过计算验证与 L3 计数检查。详见 [SC 原版对照与修复记录](benchmark/original/README.md)。
详情见 [L3 TLB 使用说明](benchmark/l3%20tlb/README.md)。
原始数据库和可执行文件默认不进入 Git；代码、配置与文档可以正常提交。

## SC 官方原版与修复后 shared 对照

```bash
# 固定官方源码，已确认四 GPU 小输入会复现旧驱动停滞
bash "benchmark/original/sc.sh"

# 当前源码的 shared 模式，包含驱动修复，不启用 demand / L3
bash "benchmark/original/sc-fixed.sh"
```

独立官方源码在 `external/mgpusim-original/`，官方提交、归档校验值和恢复方法见
[原版说明](benchmark/original/README.md)。两个入口分别输出到
`results/original/sc/` 与 `results/original/sc-fixed/`。
原版停滞日志已经保存，无需重复运行等待；正常的 L3 实验继续使用上面的 `benchmark/l3 tlb/sc.sh`。

## 历史记录

9.19
更新了仓库readme，文件路径，以及40篇多gpu论文
9.20
更新论文
Coarse-Grained_Duplication_First_Fine-Grained_Deduplication_Later_Duplication-Centric_Multi-GPU_Memory_Management
readnotes于documents文件夹
部署论文
TrioSim A Lightweight Simulator for Large-Scale DNNWorkloads on Multi-GPU Systems
仓库于external文件夹
部署论文
mgpusim
仓库于external文件夹
添加了一个简单的脚本
在experiments文件夹
