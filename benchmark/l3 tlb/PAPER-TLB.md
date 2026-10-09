# LIBRA TLB 容量配置：实现与使用

2026-09-30 已在 WSL 项目 `/home/only/projects/mgpu-project` 部署并运行验证。
操作前 Git HEAD：`57fb363bce19831a1ce61007ab3f1160b9bff896`。本轮改动尚未提交 Git。

## 本轮对齐的参数

| 层级 | 主项数 | 相联度 | 每主项子项数 | 查找延迟 | LRU 替换单位 |
|---|---:|---:|---:|---:|---|
| L1（每个现有实例） | 16 | 16 路 | 1 | 1 周期 | 页 |
| L2（每个 GPU 一份） | 128 | 8 路 | 16 | 10 周期 | 主项及其全部子项 |
| L3（每个 GPU 一份） | 1024 | 8 路 | 16 | 40 周期 | 主项及其全部子项 |

页大小仍为 4 KiB，时钟 1 GHz。每实例最大映射覆盖分别为 64 KiB、8 MiB、64 MiB；覆盖范围不是 TLB 本体的字节容量。
L2/L3 每个子项有独立有效映射，只填入实际请求的页，不推测邻页映射，不自动预取。
L1 使用相同可计时组件的单子项配置，避免原 Akita 单级 lookup pipeline 在 `Latency=1` 时的停滞问题；1/10/40 周期均有定时测试。

## 范围：容量对齐，AMD 共享组织

配置名特意使用 `libra-capacity`。它对齐上述容量、相联度、子项与查找延迟，没有实现论文的 NVIDIA TPC/GPC 共享拓扑。

当前每 GPU 保留 16 个 Shader Array、每 SA 4 个 CU，共 64 个 CU：

- 64 个向量 L1 TLB：每 CU 一份，每份 16 项。
- 16 个标量 L1 TLB、16 个指令 L1 TLB：各自每 SA 一份，每份 16 项。
- L2 仍是整个 GPU 共用一份，论文要求的是每 GPC 一份；两者不是同一共享范围。
- L3 每 GPU 共用一份。

因此不能把本轮结果直接称为完整 LIBRA/A100 硬件复现。108 SM、TPC/GPC 组织、DRAM 超额订阅、真实 NVLink/PCIe 带宽及完整迁移/再次缺页，不在本次容量调整内。
L1 全部实例合计 96 个；新配置会报告含零活动的所有实例，所以 `Components=96`，旧配置的普通 TLB 报告只包含产生流量的实例，数量不可直接横向比较。

执行资源是显式模型假设，不来自论文表：L1 保留各路径原来的宽度和 MSHR 数（向量/标量 32/64，指令 4/4），最大 256 个在途请求、每页 64 个等待者；L2 宽度 1024、64 个 MSHR、每页 64 个等待者、最多 4096 个在途请求。L3 沿用已有可调参数。这些资源均有完成与上限检查。

## 直接运行

项目级 L3 实验入口已默认选择新配置：

```bash
cd /home/only/projects/mgpu-project

# 默认小输入，先验证环境
bash "benchmark/l3 tlb/sc.sh"

# 较大的 SC；自行按需求调整输入
bash "benchmark/l3 tlb/sc.sh" -width=2560 -height=2560

# 同样输入，切回原有 L1/L2 配置做对照，仍保留 demand-l3
bash "benchmark/l3 tlb/sc.sh" -tlb-profile=legacy -width=2560 -height=2560

# 显式指定新配置
bash "benchmark/l3 tlb/sc.sh" -tlb-profile=libra-capacity -width=126 -height=126
```

每次自动重新编译。新输出目录附带配置名，避免混淆：

```text
results/l3 tlb/sc/时间_2560x2560_mask-3_libra-capacity/
results/l3 tlb/sc/时间_2560x2560_mask-3_legacy/
```

旧结果保留原位。`latest` 仍指向最近成功的运行，不限 profile；比较时优先使用脚本打印的精确目录，或核对 `run.json`。

```bash
cat "results/l3 tlb/sc/latest/l1-hit-rate.txt"
cat "results/l3 tlb/sc/latest/l2-hit-rate.txt"
cat "results/l3 tlb/sc/latest/l3-hit-rate.txt"
cat "results/l3 tlb/sc/latest/tlb-config-check.txt"
```

最后一个文件来自直接读取 SQLite 的 `component_spec` 和指标表，检查实际构建的结构、下一级端口、请求守恒、全部完成和资源上限，不是只打印命令行参数。
L3 容量参数仍能覆盖默认值，例如 `-l3-tlb-entries=512`；此时报告显示实际 512 项，不再称为论文默认 1024 项。

## 两类开关独立

`-tlb-profile=legacy|libra-capacity` 选择 L1/L2 容量和计时实现。
`-vm-mode=shared|demand|ideal|demand-l3` 选择地址翻译/缺页处理路径；仅 `demand-l3` 带 L3。

直接运行模拟器时，`-tlb-profile` 默认仍为 `legacy`。只有项目的 `benchmark/l3 tlb/config.json` 改为默认 `libra-capacity`，因此旧命令不会被悄悄切换。
新 profile 只支持 `-timing -gpu=r9nano`；其他 GPU 类型或功能模拟会明确拒绝。
原版对照 `benchmark/original/sc.sh` 继续使用独立、未修改的官方源码。`benchmark/original/sc-fixed.sh` 继续使用当前源码的 shared + legacy。

## 实际验证

- 三层连通测试：顺序访问 129 个不同 sector，各只访问一个子页，超过 L2 的 128 个主项；再次访问已被 L2 淘汰的第一页，命中 L3，页表遍历次数不增加。
- 再次访问第一页命中 L1；同一 sector 的未填充邻页仍产生真实缺失，证明没有把 L2 写成 2048 个普通表项或隐式预取。
- 单周期 L1、10 周期 L2、40 周期 L3 的命中计时验证。
- 新 profile 的四 GPU UVM SC `62×62`、`126×126`，FIR `length=4096`：全部 `Passed!`，保存参数与三层流量检查通过。
- 新 profile + shared/demand/ideal 的四 GPU UVM SC `62×62`：全部 `Passed!`。
- legacy + demand-l3 的 SC `62×62`：通过；与本次修改前保存的结果对比，5859 项指标一致。
- 相关组件和平台构建的 race 测试通过；全仓库 `golangci-lint run ./... --timeout=10m` 为 `0 issues.`。

小规模 SC/FIR 的 L3 命中率仍为 0%，它们不足以让当前 L2 淘汰有后续复用的映射。已通过受控三层测试确认 L3 命中路径有效。本轮没有重跑 2560×2560，不能声称该规模的新命中率已经测得。

验证数据：

```text
results/l3 tlb/sc/2026-09-30_23-14-47_62x62_mask-3_libra-capacity/
results/l3 tlb/sc/2026-09-30_23-17-17_126x126_mask-3_libra-capacity/
results/l3 tlb/fir/2026-09-30_23-14-52_length-4096_libra-capacity/
results/l3 tlb/sc/2026-09-30_23-15-48_62x62_mask-3_legacy/
results/l3 tlb/profile-validation/
```

## 主要代码位置

- `external/mgpusim/amd/samples/runner/timingconfig/tlbprofile/profile.go`：L1/L2 预设和资源参数。
- `external/mgpusim/amd/samples/runner/tlbflags.go`：配置开关及适用范围检查。
- `external/mgpusim/amd/samples/runner/timingconfig/r9nano/builder.go`：L2 构建与 profile 传递。
- `external/mgpusim/amd/samples/runner/timingconfig/shaderarray/builder.go`：三种 L1 路径。
- `external/mgpusim/amd/timing/sectortlb/hierarchy_test.go`：三层实际消息流、替换和命中计时测试。
- `benchmark/l3 tlb/check_tlb_profile.py`：运行后数据库结构与守恒检查。
