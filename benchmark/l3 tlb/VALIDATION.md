# 2026-09-29：WSL 目录统一与验证

> 2026-09-30 后续更新：下述 SC 停滞已定位为官方旧驱动也存在的拷贝／刷新完成顺序缺陷，并已修复。SC 62×62、126×126 的四 GPU UVM + L3 运行现已通过；详情见 [SC 修复记录](../original/README.md)。以下保留当时的诊断记录。

> 后续容量调整已完成：`libra-capacity` 的实际参数、小规模运行、三层命中验证及共享范围限制见 [容量配置说明](PAPER-TLB.md)。

环境：Windows 通过 `wsl.exe -d only -u only` 直接执行，发行版 `only` 为 WSL2，Linux 内核 `6.6.87.2-microsoft-standard-WSL2`。
源码：`/home/only/projects/mgpu-project/external/mgpusim`，Git HEAD `b543a92a74bb14fc08fac8d1aaa43eb6359f8042`，Go `1.27.1`。

本次新增项目级运行脚本、配置和目录说明，没有修改模拟器 Go 源码或硬件配置。
操作前已存在的 44 项外部代码/二进制工作区变化已保留；没有提交 Git。

## 通过的检查

- Bash 入口语法、Python 语法和 JSON 格式。
- 从 `/tmp` 调用入口时能正确定位项目。
- 拒绝覆盖脚本管理的输出前缀；拒绝将 L3 专用入口切成无 L3 的模式。
- 实际在含空格的目标路径内编译并运行四 GPU FIR，length=4096，UVM、demand-l3、verify。
- FIR 输出 `Passed!`，四张 GPU 各 7 个 L3 请求、0 个 hit、7 个 miss，命中率 0%；L3 请求完成、GMMU 对应计数和资源上限检查通过。
- 实际生成 L2/L3 报告、结果元数据、完整命令与二进制校验值，并创建 FIR 的 `latest` 相对符号链接。
- 失败 SC 验证保留 `FAILED` 状态和日志，没有创建虚假的 `latest` 成功链接。
- Git 空白检查和结果目录忽略规则检查。

FIR 成功结果：

```text
/home/only/projects/mgpu-project/results/l3 tlb/fir/2026-09-29_22-57-43_length-4096/
```

## SC 未通过的验证

SC 的 width=62、height=62、mask-size=3 在四 GPU 时序模拟中没有完成。
进程在实际计算约一秒后保持睡眠，Go 调用栈显示应用停在 `Driver.DrainCommandQueue()`，驱动等待新工作。
对本轮启动的停滞进程发送 SIGQUIT 保存了调用栈；没有取得有效最终指标。

```text
/home/only/projects/mgpu-project/results/l3 tlb/sc/2026-09-29_22-58-07_62x62_mask-3/run.log
```

随后使用同一个可执行文件、同样的小输入运行 `-vm-mode=shared` 对照，不含 L3 和 demand 扩展，25 秒诊断窗口内同样未完成并处于睡眠，调用栈同样显示等待命令队列。这说明问题不要求启用 L3 才出现，但具体根因尚未定位，不能直接归因于 WSL、原版模拟器或某个组件。

```text
/home/only/projects/mgpu-project/results/l3 tlb/sc/diagnostic-shared-62x62/
```

SC 入口已整理，不能宣称当前四 GPU SC 已验证可用。MM、MT、BS、KM 入口沿用已有应用参数，本轮未运行。

## 历史结果与范围

WSL 中没有找到旧的 `/home/only/mgpusim-runs` 结果目录；源码样例目录中的七份 SQLite 文件没有 L3 指标，不作为 L3 结果迁移。VirtualBox 中原有运行数据没有在本轮访问或移动。
新运行结果统一进入 `results/l3 tlb/<应用>/`。FIR 的原有 0% 行为保持不变；本次没有缩小 L2 TLB 或实现完整 LIBRA 配置。
