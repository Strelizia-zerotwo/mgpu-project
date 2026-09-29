# 四 GPU UVM + L3 TLB：统一运行入口

项目目录：`/home/only/projects/mgpu-project`。这里存放实验配置与运行脚本；模拟器和内置应用源码继续位于 `external/mgpusim/`。
脚本根据自身位置定位项目，可从任何工作目录调用，并正确处理 `l3 tlb` 中的空格。

2026-09-30 更新：已定位并修复官方旧驱动也存在的拷贝／刷新完成顺序缺陷。SC 的四 GPU UVM `62×62` 与 `126×126` 输入现已通过计算验证和 L3 计数检查，FIR 回归也通过。原版对照、根因、命令与证据见 [SC 修复记录](../original/README.md)。失败运行仍不会更新 `latest`。

## 直接运行

在 WSL 终端执行：

```bash
cd /home/only/projects/mgpu-project

# FIR：默认 length=4096
bash "benchmark/l3 tlb/fir.sh"

# SC：默认 width=62、height=62、mask-size=3
bash "benchmark/l3 tlb/sc.sh"
```

每条命令依次执行编译、四 GPU 时序模拟、计算正确性验证、L2/L3 统计读取和 L3 计数检查。
Go 优先使用 PATH 中的版本，PATH 中没有时自动使用 `/usr/local/go/bin/go`；Python 使用系统 `python3`，不需要安装 SQLite 图形软件。
每次执行 `go build -p 2` 利用 Go 缓存更新二进制，不要求手动重编译，也不会调用全仓库构建。

## 修改输入与 L3 参数

```bash
bash "benchmark/l3 tlb/fir.sh" -length=81920
bash "benchmark/l3 tlb/sc.sh" -width=126 -height=126
bash "benchmark/l3 tlb/sc.sh" -width=254 -height=254
bash "benchmark/l3 tlb/sc.sh" -width=126 -height=126 -l3-tlb-entries=512
```

命令行参数覆盖 `config.json`；也可以编辑配置文件来改变长期默认值。
SC 在 mask-size=3 时可先按 62、126、254、510 逐级同时增大宽高，确认小输入验证通过后再扩大。
没有保证输入增大后 L3 命中率一定提高。

其他已有多 GPU 与统一分配路径的应用入口：

```bash
bash "benchmark/l3 tlb/run.sh" mm -x=128 -y=128 -z=128
bash "benchmark/l3 tlb/run.sh" mt -width=256
bash "benchmark/l3 tlb/run.sh" bs -length=1024
bash "benchmark/l3 tlb/run.sh" km -points=1024 -features=32 -clusters=5 -max-iter=5
```

也接受完整应用名，如 `simpleconvolution`，结果仍归入 `sc/`。其他应用的入口存在不表示已经完成本轮运行验证。
这个入口专用于 `demand-l3`，必须开启 timing、UVM、verify、report-all 和 disable-rtm。
可以用 `-gpus=1,2` 改变 GPU 列表，读取与检查脚本会使用同一列表。
输出前缀由脚本管理，不接受手工传入 `-metric-file-name`。

## 结果位置

```text
/home/only/projects/mgpu-project/results/l3 tlb/fir/时间_length-4096/
/home/only/projects/mgpu-project/results/l3 tlb/sc/时间_62x62_mask-3/
```

查看最近一次成功运行的命中率：

```bash
cat "results/l3 tlb/fir/latest/l3-hit-rate.txt"
cat "results/l3 tlb/sc/latest/l3-hit-rate.txt"
cat "results/l3 tlb/sc/latest/l2-hit-rate.txt"
```

检查运行状态：

```bash
cat "results/l3 tlb/sc/latest/status.txt"
cat "results/l3 tlb/sc/latest/check.log"
```

脚本打印本次精确目录。若运行失败或被中断，去那个目录看 `status.txt`、`build.log`、`run.log`；`latest` 继续指向之前成功的结果，不会指向失败运行。

## 统计与模型范围

驻留映射命中率为 `hit / (hit + miss + mshr-hit)`；MSHR 合并不算已有映射命中。
这是到达对应 TLB 的整次运行请求统计，包括经过该路径的指令和数据翻译。
原有 L2 TLB 仍很大，小应用的 L3 为 0% 可以是正常的冷缺失行为。
本次目录统一没有更改 TLB 容量或模拟器硬件模型。

当前第四版模拟固定数据位置的首次映射缺页，不包含完整迁移、换页和再次缺页机制，也不等于完整 LIBRA 硬件配置。
原有组件说明与兼容工具保留在 `external/mgpusim/experiments/l3-tlb/` 和 `experiments/libra-baseline-guide/` 中；新的运行结果统一写到项目的 `results/l3 tlb/`。
