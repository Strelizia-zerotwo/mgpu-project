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

一条运行命令自动编译、模拟、验证计算结果并读取 L2/L3 命中率。
FIR 已通过本轮 WSL 验证；SC 小输入目前出现命令队列等待停滞，入口已整理但尚无有效 SC 结果。详见 [验证记录](benchmark/l3%20tlb/VALIDATION.md)。
详情见 [L3 TLB 使用说明](benchmark/l3%20tlb/README.md)。
原始数据库和可执行文件默认不进入 Git；代码、配置与文档可以正常提交。

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
