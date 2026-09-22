# 多 GPU 映射缺页基线

这套基线用于研究：**当 GPU 本地缺少一个合法页面的映射时，怎样减少取得映射的开销。** 它在 MGPUSim 的实际 GPU kernel 执行中产生请求、排队、返回映射和恢复访问，不是给最终 kernel time 加一个估算值。

它是可修改的研究模型，不是 ShadowUpdate 复现，也没有校准成 NVIDIA 或 AMD 的真实 UVM 时延。

## 先运行

在 Linux 虚拟机执行：

```bash
cd /home/only/projects/mgpu-project/external/mgpusim
bash experiments/fault-baseline/run.sh
```

脚本编译 FIR，并执行三组长度为 16384 的双 GPU FIR：

1. `demand`：本地映射按需建立；主机 16 个服务槽、64 项等待队列。
2. `ideal`：本地映射预先可用；硬件参数与 demand 完全相同。
3. `congested`：demand 模式，主机缩减至 1 个服务槽、2 项等待队列，用来验证排队与反压。

每次会在 `~/mgpusim-runs/fault-baseline-XXXXXX/` 建立独立目录，保存三个 SQLite 数据库、日志、检查结果、实际命令、编译的二进制、Git 版本和完整源码快照。

检查程序验证计算结果、请求守恒、有限资源没有溢出、两个 GPU 都实际产生缺页、理想模式没有主机请求，以及拥塞配置确实有等待和反压。

## 自己运行与修改参数

```bash
cd /home/only/projects/mgpu-project/external/mgpusim
go build -o /tmp/mgpu-fir ./amd/samples/fir
mkdir -p ~/mgpusim-runs
mgpu_result=$(mktemp -d "$HOME/mgpusim-runs/my-fault-test-XXXXXX")

/tmp/mgpu-fir -timing -verify -gpus=1,2 -use-unified-memory \
  -length=16384 -report-all -vm-mode=demand \
  -metric-file-name="$mgpu_result/demand"

/tmp/mgpu-fir -timing -verify -gpus=1,2 -use-unified-memory \
  -length=16384 -report-all -vm-mode=ideal \
  -metric-file-name="$mgpu_result/ideal"
```

例如，将基线主机并行度减至 2、等待队列减至 8，在对应命令中加入：

```text
-vm-host-walkers=2 -vm-host-queue=8
```

所有新增参数的时间单位是 **1 GHz 时钟下的周期**，1 周期 = 1 ns。

| 参数 | 默认值 | 含义 |
|---|---:|---|
| `-vm-mode` | `shared` | `shared` 原有模式；`demand` 新缺页基线；`ideal` 匹配硬件的理想映射对照 |
| `-vm-local-walk-cycles` | 100 | GPU 一次本地查询的固定服务时间 |
| `-vm-local-walkers` | 8 | 每 GPU 可同时工作的本地遍历器 |
| `-vm-transaction-slots` | 64 | 每 GPU 在途页面翻译事务总上限，包含等待主机的事务 |
| `-vm-max-waiters` | 64 | 同一事务可合并的翻译请求数上限 |
| `-vm-host-walk-cycles` | 400 | 主机一次映射查询的服务时间 |
| `-vm-host-walkers` | 16 | 全系统共享的主机服务槽数量 |
| `-vm-host-queue` | 64 | 主机等待队列容量，不含服务中的请求及端口缓冲 |
| `-vm-link-cycles` | 200 | 每个方向的抽象消息传输延迟 |
| `-vm-install-cycles` | 100 | 收到映射后在 GPU 安装映射的延迟 |

这些值是**起始建模假设**。本地查询与主机查询目前都是固定服务时间，没有建模多级 PTE 的真实缓存/DRAM 流量。400 ns 不能当成已经测得的真实 UVM 服务时间。

默认一次缺页在本地失败之后的额外延迟约为：200 + 400 + 200 + 100 = 900 ns，再加排队、反压及事件阶段边界。实际测量以数据库为准。

`-vm-*` 时间/容量参数必须与 `-vm-mode=demand` 或 `ideal` 一起使用。`-vm-mode` 不与早先的 `-ideal-local-page-table` 同时使用；后者保留旧实验兼容性。新方法的基线应使用这里的 demand/ideal 两种模式。

## 映射怎样变化

```text
内存分配：
    主机权威表建立 VA -> PA
    demand：统一内存数据页在每个 GPU 的本地页表中都暂不安装
    ideal：同样的映射预先安装到所有 GPU 本地页表

GPU 执行：
    L1/L2 TLB miss
      -> 本地 GMMU 排队并遍历
      -> 本地有效映射存在：返回翻译，继续数据访问
      -> 本地映射不存在：发出主机映射请求
         -> 请求方向延迟 -> 主机入队 -> 等待空闲服务槽
         -> 查询主机权威表 -> 返回方向延迟
         -> 本 GPU 安装映射 -> 返回原请求的翻译结果
```

- 只对 `AllocateUnifiedMemory` 分配的页面采用按需映射。代码、内核参数和普通显存分配预先映射，避免把 kernel 装载等路径混成研究对象。
- 当前 MGPUSim 分配器将统一内存数据放在 GPU 1。GPU 2 获得映射后仍按原有路径远程访问数据；映射安装不移动数据。
- 同一 GPU 对同一 `(PID, 虚拟页)` 的在途请求合并，返回映射后答复所有等待者。
- 不同 GPU 各自建立映射。主机默认不合并跨 GPU 请求，这是一项明确的基线策略。
- 映射安装后保持有效且不驱逐，后续 TLB miss 可本地查询命中。因此本版研究首次映射缺页；它不会自行产生迁移后的重复缺页。
- 合法分配与本地映射缺失分开处理：如果主机权威表也找不到页面，立即报错，不会创建虚假物理地址。

## 主机竞争模型

主机等待队列和服务槽是显式有限资源。队列满后停止接收；反压会逐级传回 GPU。每个 GPU 最多持有配置数量的在途事务，因此等待主机的事务过多时，也会阻塞新的地址翻译。

主机每周期最多从入口接收一个请求，入口/出口各有一个消息缓冲；等待队列之外另有配置数量的服务槽。响应出口被阻塞时，已完成查询的请求仍占用服务槽，直至响应送出。本地端口的输入/输出缓冲容量为 `vm-transaction-slots`，主机消息端口每方向一个缓冲。

消息通过现有 DirectConnection 搬运，`vm-link-cycles` 显式补充每方向的延迟。本版模拟主机服务竞争、队列满和端口反压，**不声称模拟了真实 PCIe/NVLink 的分包、带宽争用或交换机拓扑**。

## 结果怎么读

用 DB Browser for SQLite 打开 `demand.sqlite3`，执行：

```sql
SELECT Location, What, Value, Unit
FROM mgpusim_metrics
WHERE What LIKE 'vm_%'
ORDER BY Location, What;
```

| 位置与指标 | 意义 |
|---|---|
| GPU MMU：`vm_requests` / `vm_completed` | 收到/完成的上游翻译请求数 |
| GPU MMU：`vm_coalesced` | 合并到已有同页事务的请求数 |
| GPU MMU：`vm_local_walks` | 实际完成的本地页表查询数 |
| GPU MMU：`vm_local_hits` / `vm_local_faults` | 本地查询命中/缺失次数 |
| GPU MMU：`vm_fault_requests` / `vm_fault_resolved` | 发给主机/成功填入映射的次数 |
| GPU MMU：`vm_translation_average` / `vm_translation_max` | 从 MMU 取出上游请求到答复的平均/最大时间，含本地排队、查询和可能的主机路径 |
| GPU MMU：`vm_fault_average` / `vm_fault_max` | 从判定本地缺失到映射安装完成的时间，不含之前的本地遍历 |
| GPU MMU：`vm_local_queue_average` | 等待本地遍历器的平均时间 |
| GPU MMU：`vm_host_send_blocked_cycles` | 各缺页事务等待主机发送端口的周期数之和，不是互斥的总墙钟周期 |
| 主机：`vm_received` / `vm_completed` | 接收/答复的缺页请求数 |
| 主机：`vm_queue_wait_average` / `vm_queue_wait_max` | 进入内部队列后到取得服务槽的等待时间 |
| 主机：`vm_ingress_wait_average` | GPU 发出请求后到主机内部队列接收之间的时间，含端口传递及入队反压；不含此前显式的出站延迟 |
| 主机：`vm_service_average` | 主机查询的服务时间，不含排队和响应端口阻塞 |
| 主机：`vm_queue_peak` / `vm_active_peak` | 等待队列/服务槽占用峰值 |
| 主机：`vm_queue_full_cycles` | 有请求等待接收且内部队列已满的周期数 |

时间指标单位是 second。无样本时平均/最大值记录为 0，必须结合计数判断。每个请求的延迟可能重叠，不能把平均缺页延迟乘以缺页数直接当作 kernel 延迟。

TLB miss 仍从原有 `Location` 含 TLB、`What=miss` 的指标查看。TLB miss、本地页表缺失和主机请求是不同层次的计数。

## 给新方法留出的修改位置

| 文件 | 适合修改的策略 |
|---|---|
| `amd/timing/idealmapping/demand.go` | 初始映射策略、映射安装接口 |
| `amd/timing/faultvm/gpu.go` | 本地缺页后的请求合并、请求发起、返回映射的处理 |
| `amd/timing/faultvm/host.go` | 主机调度、服务策略；例如另行研究跨 GPU 合并 |
| `amd/timing/faultvm/types.go` | 配置、消息与统计结构 |
| `amd/samples/runner/timingconfig/fault.go` | 平台连接与 demand/ideal 模式选择 |
| `amd/samples/runner/faultreport.go` | 指标输出 |

基线/理想模式使用完全相同的本地 GMMU 结构和参数。新方案比较时也应保留同样的硬件资源预算，另外列出新增结构和开销。

## 本版的明确边界

这是固定数据放置的映射缺页模型。未实现访问计数迁移、页复制、CPU 软件 UVM 驱动、TLB shootdown、映射驱逐、GPU 之间的失效广播、迁移中请求挂起/恢复，也未验证检查点恢复或并行引擎。

为避免静默使用旧 TLB 映射，首次实际翻译发生后，驱动的重映射/释放操作会明确报错。新增页面分配仍可进行。将来若研究迁移或主动失效，需要补充完整一致性协议再放开这条限制。

最初的 `-ideal-local-page-table` 实验仍可运行，但它与默认共享 MMU 的比较会改变 MMU 数量。**新研究的基线应使用 `-vm-mode=demand`，匹配的理想对照使用 `-vm-mode=ideal`。**

## 开发验证

```bash
go test -race ./amd/timing/faultvm ./amd/timing/idealmapping
go test ./amd/samples/runner/...
go build ./...
bash experiments/fault-baseline/run.sh
```

模块测试覆盖同页请求合并、热映射命中、不同 GPU 映射隔离、PID 隔离、普通分配、有限主机队列、反压、服务并行度、非法地址、旧映射返回和运行中重映射保护。真实 FIR 运行验证完整 TLB/GMMU/主机/数据访问路径和计算结果。
