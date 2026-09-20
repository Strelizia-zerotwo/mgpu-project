# 多 GPU 论文阅读与模拟实验


目前已在 `paper/` 中收集 40 篇相关论文，并整理了一份 Markdown 概述文档。其余目录暂为空，后续随阅读和实验进展逐步补充。

## 目录说明

```text
.
├── README.md       # 仓库介绍与目录说明
├── paper/          # 多 GPU 相关论文资料
├── docs/           # 模拟器部署、使用方法与问题记录
├── external/       # 本地拉取的模拟器及相关第三方代码
├── benchmarks/     # 自己编写或独立维护的 benchmark
├── experiments/    # 具体实验的说明、配置和运行脚本
├── runs/           # 每次运行的原始输出与运行记录
└── results/        # 整理后的实验数据、图表与结论
```

| 文件夹 | 用途 |
| --- | --- |
| `paper/` | 保存论文资料，后续逐步补充阅读笔记与分类整理。 |
| `docs/` | 保存模拟器及工具的说明文档，包括代码来源、使用版本、环境部署、编译运行方法和问题记录。模拟器相关文档可统一放在 `docs/simulators/` 下。 |
| `external/` | 存放本地拉取的模拟器、论文 artifact 和独立的第三方 benchmark 套件，保留各项目原有的目录结构。 |
| `benchmarks/` | 保存自己编写或需要独立维护的测试程序。模拟器自带的 benchmark 保留在其原仓库中，直接调用。 |
| `experiments/` | 按实验建立子目录，保存实验目的、输入参数、配置和 `run.sh` 等运行脚本。 |
| `runs/` | 按实验和运行编号保存原始输出，例如日志、指标数据库、实际运行命令与代码版本。同一实验的多次运行分别保存。 |
| `results/` | 保存从原始输出中整理出的指标汇总、图表和分析结论，并记录对应的实验与运行编号。 |

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
