# 多 GPU论文

**2024—2026 · ISCA / MICRO / HPCA / ASPLOS · 40 篇**



[按会议查找论文](<论文索引.md>) · [建议阅读路线](#reading-routes)

## 研究方向导航

| 方向 | 核心问题 | 篇数 |
| :--- | :--- | ---: |
| [通信与互连](#communication) | 数据怎样传得更高效？ | 14 |
| [内存与地址管理](#memory) | 数据放在哪里、什么时候搬？ | 7 |
| [模拟与性能分析](#modeling) | 如何预测、测量和解释系统性能？ | 6 |
| [集群与运行管理](#systems) | 如何管理资源并稳定运行集群？ | 8 |
| [安全与机密计算](#security) | 安全保护的通信代价怎样降低？ | 2 |
| [其他应用与执行框架](#applications) | 其他计算任务怎样扩展到多 GPU？ | 3 |

<a id="reading-routes"></a>

## 阅读路线


| 阅读目标 | 建议顺序 |
| :--- | :--- |
| 理解跨 GPU 页面管理 | [GRIT](#paper-29) → [OASIS](#paper-31) → [CDFD](#paper-13) → [ShadowUpdate](#paper-14) → [LIBRA](#paper-15) |
| 理解通信与计算重叠 | [MSCCL++](#paper-63) → [T3](#paper-42) → [CAIS](#paper-35) → [DySHARP](#paper-11) / [MoE-Hub](#paper-17) |
| 做模拟器与性能分析 | [MAD-Max](#paper-2) → [vTrain](#paper-24) → [TrioSim](#paper-8) → [STAGE](#paper-22)；结合[效率实测](#paper-26)理解测量指标 |
| 理解互连拓扑与路径 | [TCCL](#paper-40) → [TACOS](#paper-23) → [NetCrafter](#paper-6) |
| 理解显存卸载 | [Aqua](#paper-59) → [SuperOffload](#paper-69) → [DisDP](#paper-21) |

---

<a id="communication"></a>

## 01 · 通信与互连

让 GPU 之间的数据传得更少、更快，并尽量与计算同时进行。

<a id="paper-4"></a>

### 01 · Chimera

**混合并行的通信融合**

> Chimera: Communication Fusion for Hybrid Parallelism in Large Language Models

**ISCA 2025** · [本地 PDF](<ISCA/2025/Chimera Communication Fusion for Hybrid Parallelism in LargeLanguage Models.pdf>) · [论文来源](https://doi.org/10.1145/3695053.3731025) · [GitHub](https://github.com/redbird-arch/isca2025-chimera-artifact)

针对混合并行切换过程中反复出现的集合通信，分析相邻通信操作中的冗余。通过算子重排和通信融合减少数据传输，降低多 GPU 执行大模型的通信开销。

**阅读重点：** 相邻集合通信中哪些数据搬运可以消除或合并？

<a id="paper-6"></a>

### 02 · NetCrafter

**非均匀带宽下的流量优化**

> NetCrafter: Tailoring Network Traffic for Non-Uniform Bandwidth Multi-GPU Systems

**ISCA 2025** · [本地 PDF](<ISCA/2025/NetCrafter Tailoring Network Traffic for Non-UniformBandwidth Multi-GPU Systems.pdf>) · [论文来源](https://doi.org/10.1145/3695053.3731040)

研究不同 GPU 组之间带宽不均匀造成的网络瓶颈。通过拼接未充分利用的数据包、减少不必要传输并调整传输优先级，提高有限互连带宽的有效利用率。

**阅读重点：** 拼接、裁剪和优先级调度分别减少哪一类网络开销？

<a id="paper-10"></a>

### 03 · TRACI

**推荐模型的动态通信**

> TRACI: Network Acceleration of Input-Dynamic Communication for Large-Scale Deep Learning Recommendation Model

**ISCA 2025** · [本地 PDF](<ISCA/2025/TRACI Network Acceleration of Input-Dynamic Communicationfor Large-Scale Deep Learning Recommendation Model.pdf>) · [论文来源](https://doi.org/10.1145/3695053.3731105)

针对推荐模型嵌入层的数据聚合，研究输入决定通信模式时产生的带宽瓶颈。利用新的网络事务和交换机设计识别数据复用，减少多 GPU 之间重复的数据传输。

**阅读重点：** 输入变化导致的通信模式如何在网络中利用数据复用？

<a id="paper-11"></a>

### 04 · DySHARP

**MoE 动态交换机内计算**

> Accelerating MoE with Dynamic In-Switch Computing on Multi-GPUs

**ISCA 2026** · [本地 PDF](<ISCA/2026/Accelerating MoE with Dynamic In-Switch Computing on Multi-GPUs.pdf>) · [论文来源](https://arxiv.org/abs/2605.05607)

提出 DySHARP，使交换机内计算支持 MoE 动态且不规则的专家通信。结合动态 multimem 寻址和 token 级计算通信融合，减少冗余传输并改善流水线重叠。

**阅读重点：** 不规则专家通信如何使用交换机内计算，并与 token 计算重叠？

<a id="paper-12"></a>

### 05 · RoCC

**复用 ROP 加速集合通信**

> RoCC: Harnessing Raster Operations Pipeline for Efficient Tensor Collective Communication

**ISCA 2026** · [本地 PDF](<ISCA/2026/RoCC_Harnessing_Raster_Operations_Pipeline_for_Efficient_Tensor_Collective_Communication.pdf>) · [论文来源](https://doi.org/10.1109/ISCA66397.2026.00094)

利用 GPU 中通常未被 AI 工作负载充分使用的光栅操作流水线 ROP，承接集合通信中的计算和消息处理。目标是让模型计算与通信细粒度重叠，降低二者争用常规计算资源的开销。

**阅读重点：** 哪些通信工作可以交给 ROP，如何减少与模型计算的资源竞争？

<a id="paper-17"></a>

### 06 · MoE-Hub

**MoE 通信的硬件辅助**

> MoE-Hub: Taming Software Complexity for Seamless MoE Overlap with Hardware-Accelerated Communication on Multi-GPU Systems

**ISCA 2026** · [本地 PDF](<ISCA/2026/MoE-Hub Taming Software Complexity for Seamless MoE Overlap with Hardware-Accelerated Communication on Multi-GPU Systems.pdf>) · [论文来源](https://doi.org/10.1109/ISCA66397.2026.00172)

针对 MoE 动态 token 到专家映射与 GPU 静态地址通信模型不匹配的问题，提出 MoE-Hub。将地址分配和通信控制交给轻量级硬件，降低软件协调复杂度并改善计算通信重叠。

**阅读重点：** 动态 token 地址与通信控制如何从软件转移到硬件？

<a id="paper-21"></a>

### 07 · DisDP

**计算、网络与存储解耦**

> DisDP: Disaggregating Compute, Network, and Storage for Model-Sharded Data-Parallel Training

**ISCA 2026** · [本地 PDF](<ISCA/2026/DisDP Disaggregating Compute, Network, and Storage for Model-Sharded Data-Parallel Training.pdf>) · [论文来源](https://doi.org/10.1109/isca66397.2026.00171)

针对模型分片数据并行中计算、集合通信和优化器状态管理相互干扰的问题，提出 DisDP 解耦架构。将通信与存储工作分别卸载到网络设备及参数服务，使 GPU 更集中地执行模型计算。

**阅读重点：** 通信和优化器状态管理移出 GPU 后，瓶颈转移到了哪里？

<a id="paper-23"></a>

### 08 · TACOS

**拓扑感知集合通信合成**

> TACOS: Topology-Aware Collective Algorithm Synthesizer for Distributed Machine Learning

**MICRO 2024** · [本地 PDF](<MICRO/2024/TACOS Topology-Aware Collective Algorithm Synthesizer for Distributed Machine Learning.pdf>) · [论文来源](https://doi.org/10.1109/micro61859.2024.00068) · [GitHub](https://github.com/astra-sim/tacos)

提出拓扑感知集合通信算法合成器 TACOS。根据网络连接、带宽及目标集合操作自动生成通信方案，适用于复杂或异构多 GPU 互连的算法评估。

**阅读重点：** 网络拓扑与带宽约束如何转化为具体的集合通信方案？

<a id="paper-25"></a>

### 09 · NetZIP

**网络内无损压缩**

> NetZIP: Algorithm/Hardware Co-design of In-network Lossless Compression for Distributed Large Model Training

**MICRO 2025** · [本地 PDF](<MICRO/2025/NetZIP AlgorithmHardware Co-design of In-network LosslessCompression for Distributed Large Model Training.pdf>) · [论文来源](https://doi.org/10.1145/3725843.3756079) · [GitHub · 实验代码](https://github.com/ece-fast-lab/MICRO-2025-NetZIP)

针对梯度和激活传输量较大、压缩本身又有额外延迟的问题，提出 NetZIP。通过数据变换与网卡上的轻量级无损压缩硬件协同，降低分布式训练的通信量与压缩开销。

**阅读重点：** 节省的通信时间能否覆盖数据变换与压缩引入的额外开销？

<a id="paper-27"></a>

### 10 · SkipReduce

**稀疏化 AllReduce 通信**

> SkipReduce: (Interconnection) Network Sparsity to Accelerate Distributed Machine Learning

**MICRO 2025** · [本地 PDF](<MICRO/2025/SkipReduce (Interconnection) Network Sparsity to AccelerateDistributed Machine Learning.pdf>) · [论文来源](https://doi.org/10.1145/3725843.3756092) · [GitHub · 通信实现](https://github.com/hanskasan/SkipReduce-2.0) · [评估脚本](https://github.com/hanskasan/SkipReduce_MICRO2025) · [Transformers 适配](https://github.com/hanskasan/transformers_SkipReduce)

提出 SkipReduce，通过有选择地跳过梯度分片来减少 AllReduce 的通信步骤。结合随机化和按层选择，尽量控制跳过梯度对训练准确率的影响，并在 NCCL 上实现。

**阅读重点：** 跳过哪些梯度分片，如何权衡通信收益与训练准确率？

<a id="paper-35"></a>

### 11 · CAIS

**计算感知交换机内计算**

> Towards Compute-Aware In-Switch Computing for LLMs Tensor-Parallelism on Multi-GPU Systems

**HPCA 2026** · [本地 PDF](<HPCA/2026/Towards Compute-Aware In-Switch Computing for LLMs Tensor-Parallelism on Multi-GPU Systems.pdf>) · [论文来源](https://doi.org/10.1109/hpca68181.2026.11408460)

提出计算感知交换机内计算框架 CAIS，使集合通信的数据流更贴合张量并行计算内核的访存需求。结合指令、微架构、线程块协调及图级优化，改善计算与通信的重叠。

**阅读重点：** 通信的数据到达顺序如何适配计算内核的消费顺序？

<a id="paper-40"></a>

### 12 · TCCL

**PCIe 集群通信路径搜索**

> TCCL: Discovering Better Communication Paths for PCIe GPU Clusters

**ASPLOS 2024** · [本地 PDF](<ASPLOS/2024/TCCL Discovering Better Communication Pathsfor PCIe GPU Clusters.pdf>) · [论文来源](https://doi.org/10.1145/3620666.3651362) · [GitHub](https://github.com/mcrl/tccl)

针对主要依赖 PCIe 互连的 GPU 集群，测量并发数据传输表现并搜索更好的通信路径。TCCL 将发现的路径用于集合通信运行时，以改善 AllReduce、AllGather 等操作的性能。

**阅读重点：** 并发传输和 PCIe 拓扑如何影响集合通信的路径选择？

<a id="paper-42"></a>

### 13 · T3

**硬件跟踪与通信触发**

> T3: Transparent Tracking & Triggering for Fine-grained Overlap of Compute & Collectives

**ASPLOS 2024** · [本地 PDF](<ASPLOS/2024/T3 Transparent Tracking & Triggering for Fine-grained Overlap of Compute & Collectives.pdf>) · [论文来源](https://doi.org/10.1145/3620665.3640410)

提出 T3，通过硬件跟踪与触发机制让生产数据的计算操作和后续集合通信细粒度重叠。结合通信计算卸载，减少软件融合的复杂度以及计算和通信之间的资源争用。

**阅读重点：** 生产者数据何时就绪，硬件如何据此提前启动细粒度通信？

<a id="paper-63"></a>

### 14 · MSCCL++

**GPU 通信抽象与编程**

> MSCCL++: Rethinking GPU Communication Abstractions for AI Inference

**ASPLOS 2026** · [本地 PDF](<ASPLOS/2026/MSCCL++ Rethinking GPU Communication Abstractions for AI Inference.pdf>) · [论文来源](https://doi.org/10.1145/3779212.3790188) · [GitHub](https://github.com/microsoft/mscclpp)

提出 MSCCL++，为 GPU 通信提供底层原语、上层 DSL 和集合通信实现。让开发者更方便地使用不同互连与硬件能力编写高效通信内核，同时处理同步和一致性细节。

**阅读重点：** 通信原语如何表达数据移动、同步与不同互连能力？

[返回方向导航](#研究方向导航)

---

<a id="memory"></a>

## 02 · 内存与地址管理

决定数据放在哪张 GPU、何时迁移或复制，以及如何维护地址映射。

<a id="paper-13"></a>

### 15 · CDFD

**先复制、后去重的显存管理**

> Coarse-Grained Duplication First, Fine-Grained Deduplication Later: Duplication-Centric Multi-GPU Memory Management

**ISCA 2026** · [本地 PDF](<ISCA/2026/Coarse-Grained_Duplication_First_Fine-Grained_Deduplication_Later_Duplication-Centric_Multi-GPU_Memory_Management.pdf>) · [论文来源](https://doi.org/10.1109/isca66397.2026.00109)

提出以页面复制为中心的多 GPU 统一内存管理机制 CDFD。先用粗粒度复制发挥 NVLink 带宽和空闲显存的优势，再通过细粒度去重减少不必要的远程更新。

**阅读重点：** 粗粒度复制节省什么开销，细粒度去重又处理什么代价？

<a id="paper-14"></a>

### 16 · ShadowUpdate

**迁移后的地址映射传播**

> Reducing Page Faults via Invalidation-Based Mapping Propagation in Multi-GPU Systems

**ISCA 2026** · [本地 PDF](<ISCA/2026/Reducing_Page_Faults_via_Invalidation-Based_Mapping_Propagation_in_Multi-GPU_Systems.pdf>) · [论文来源](https://doi.org/10.1109/isca66397.2026.00110)

分析页面迁移后只有目标 GPU 更新映射、其他 GPU 因无效映射再次缺页的问题。提出 ShadowUpdate，利用已有的失效广播同步传播新映射，并协调迁移中的地址翻译请求。

**阅读重点：** 一次页面迁移为什么会引发其他 GPU 缺页，映射如何同步？

<a id="paper-15"></a>

### 17 · LIBRA

**多 GPU 协同页面预取**

> LIBRA: A High-Accuracy, Cost-Aware, and Coordinated Multi-GPU Page Prefetcher

**ISCA 2026** · [本地 PDF](<ISCA/2026/LIBRA_A_High-Accuracy_Cost-Aware_and_Coordinated_Multi-GPU_Page_Prefetcher.pdf>) · [论文来源](https://doi.org/10.1109/isca66397.2026.00158)

提出多 GPU 页面预取器 LIBRA，根据访问步长预测后续页面需求。综合远程访问成本、迁移收益和各 GPU 的需求协调预取，减少错误预取与页面往返迁移。

**阅读重点：** 预取决策如何同时考虑预测准确率、迁移成本与跨 GPU 冲突？

<a id="paper-29"></a>

### 18 · GRIT

**细粒度动态页面放置**

> GRIT: Enhancing Multi-GPU Performance with Fine-Grained Dynamic Page Placement

**HPCA 2024** · [本地 PDF](<HPCA/2024/GRIT_Enhancing_Multi-GPU_Performance_with_Fine-Grained_Dynamic_Page_Placement.pdf>) · [论文来源](https://doi.org/10.1109/hpca57654.2024.00085)

指出多 GPU 统一内存中的最佳页面策略会随页面、应用和执行阶段变化。GRIT 在迁移、访问计数驱动迁移和复制等策略之间动态选择，降低非均匀内存访问开销。

**阅读重点：** 同一应用的不同页面为什么需要不同的迁移或复制策略？

<a id="paper-31"></a>

### 19 · OASIS

**对象感知页面管理**

> OASIS: Object-Aware Page Management for Multi-GPU Systems

**HPCA 2025** · [本地 PDF](<HPCA/2025/OASIS Object-Aware Page Management for Multi-GPU Systems.pdf>) · [论文来源](https://doi.org/10.1109/hpca61900.2025.00124)

从应用数据对象及其执行阶段出发管理多 GPU 页面。OASIS 动态识别对象访问模式，主动选择适合的迁移或复制策略，改善统一内存性能并降低管理复杂度。

**阅读重点：** 以数据对象为单位识别访问模式，相比逐页决策有什么作用？

<a id="paper-59"></a>

### 20 · Aqua

**推理状态的网络加速卸载**

> Aqua: Network-Accelerated Memory Offloading for LLMs in Scale-Up GPU Domains

**ASPLOS 2025** · [本地 PDF](<ASPLOS/2025/Aqua Network-Accelerated Memory Offloading for LLMs in Scale-Up GPU Domains.pdf>) · [论文来源](https://doi.org/10.1145/3676641.3715983) · [GitHub · 核心库](https://github.com/aquaml/aqua) · [项目其他仓库](https://github.com/aquaml)

研究请求突发和显存不足时，LLM 推理状态换入换出的成本。Aqua 利用多 GPU 互连加速内存卸载，使抢占式请求调度在改善响应性的同时保持较高吞吐量。

**阅读重点：** 抢占推理请求时，如何降低状态换入换出对吞吐量的影响？

<a id="paper-69"></a>

### 21 · SuperOffload

**Superchip 上的训练卸载**

> SuperOffload: Unleashing the Power of Large-Scale LLM Training on Superchips

**ASPLOS 2026** · [本地 PDF](<ASPLOS/2026/SuperOffload Unleashing the Power of Large-Scale LLM Training on Superchips.pdf>) · [论文来源](https://doi.org/10.1145/3760250.3762217) · [GitHub · DeepSpeed 实现](https://github.com/deepspeedai/DeepSpeed) · [SuperOffload 示例](https://github.com/deepspeedai/DeepSpeedExamples/tree/master/training/DeepSpeed-SuperOffload)

针对 Grace CPU 与 Hopper GPU 紧耦合的 Superchip，重新设计大模型训练的卸载执行方式。优化权重搬移、分桶、数据类型转换及优化器计算，并扩展到多 GPU 数据和序列并行。

**阅读重点：** 紧耦合 CPU–GPU 互连如何改变权重搬移和优化器计算安排？

[返回方向导航](#研究方向导航)

---

<a id="modeling"></a>

## 03 · 模拟与性能分析

预测和解释多 GPU 系统的行为，比较配置，并识别性能、功耗与温度瓶颈。

<a id="paper-2"></a>

### 22 · MAD-Max

**并行策略的性能建模**

> MAD-Max Beyond Single-Node: Enabling Large Machine Learning Model Acceleration on Distributed Systems

**ISCA 2024** · [本地 PDF](<ISCA/2024/MAD-Max Beyond Single-Node Enabling Large Machine Learning Model Acceleration on Distributed Systems.pdf>) · [论文来源](https://doi.org/10.1109/isca59077.2024.00064)

研究大模型在多 GPU 上训练和推理时的性能瓶颈，提出 MAD-Max 性能建模框架。它用于评估并行策略、通信开销和软硬件设计选择，帮助在大规模部署前比较配置。

**阅读重点：** 模型如何拆分计算时间、通信时间及二者的重叠？

<a id="paper-8"></a>

### 23 · TrioSim

**大规模多 GPU 轻量级模拟**

> TrioSim: A Lightweight Simulator for Large-Scale DNN Workloads on Multi-GPU Systems

**ISCA 2025** · [本地 PDF](<ISCA/2025/TrioSim A Lightweight Simulator for Large-Scale DNNWorkloads on Multi-GPU Systems.pdf>) · [论文来源](https://doi.org/10.1145/3695053.3731082) · [GitHub](https://github.com/sarchlab/triosim)

提出面向大规模 DNN 多 GPU 执行的轻量级模拟器 TrioSim。以算子级信息结合性能建模和模拟，降低输入准备及模拟成本，便于比较多 GPU 系统配置。

**阅读重点：** 模拟器需要哪些输入，又如何权衡速度、精度与建模粒度？

<a id="paper-18"></a>

### 24 · Lit Silicon

**温度不均衡的跨 GPU 影响**

> Lit Silicon: A Case Where Thermal Imbalance Couples Concurrent Execution in Multiple GPUs

**ISCA 2026** · [本地 PDF](<ISCA/2026/Lit Silicon A Case Where Thermal Imbalance Couples Concurrent Execution in Multiple GPUs.pdf>) · [论文来源](https://doi.org/10.1109/isca66397.2026.00169) · [GitHub](https://github.com/UnaryLab/lit_silicon_tuning_amd)

发现多 GPU 节点内的温度差异会使部分 GPU 成为慢节点，并通过计算通信重叠拖慢其他 GPU。对这种耦合效应进行测量和建模，提出检测与功率管理方案。

**阅读重点：** 单张 GPU 的热降频如何通过通信与同步影响其他 GPU？

<a id="paper-22"></a>

### 25 · STAGE

**分布式负载执行图生成**

> Scalable Synthesis of Distributed LLM Workloads Through Symbolic Tensor Graphs

**ISCA 2026** · [本地 PDF](<ISCA/2026/Scalable Synthesis of Distributed LLM Workloads Through Symbolic Tensor Graphs.pdf>) · [论文来源](https://arxiv.org/abs/2511.10480) · [GitHub](https://github.com/astra-sim/stage)

提出 STAGE，使用符号化张量图生成分布式大模型工作负载执行图。支持多种并行策略及大规模 GPU 配置，为模拟器输入生成、部署前性能评估和硬件设计提供支持。

**阅读重点：** 符号化张量图如何展开成不同并行配置下的模拟输入？

<a id="paper-24"></a>

### 26 · vTrain

**训练时间与成本模拟**

> vTrain: A Simulation Framework for Evaluating Cost-effective and Compute-optimal Large Language Model Training

**MICRO 2024** · [本地 PDF](<MICRO/2024/vTrain A Simulation Framework for Evaluating Cost-effective and Compute-optimal Large Language Model Training.pdf>) · [论文来源](https://doi.org/10.1109/micro61859.2024.00021) · [GitHub](https://github.com/VIA-Research/vTrain)

提出基于性能采样的训练模拟框架 vTrain。快速估计不同并行策略和 GPU 配置下的训练时间与成本，也可用于研究集群调度和固定算力预算下的模型选择。

**阅读重点：** 性能采样如何支持不同并行策略和 GPU 数量的训练时间预测？

<a id="paper-26"></a>

### 27 · 分布式训练效率实测

**性能、功耗与温度分析**

> Characterizing the Efficiency of Distributed Training: A Power, Performance, and Thermal Perspective

**MICRO 2025** · [本地 PDF](<MICRO/2025/Characterizing the Efficiency of Distributed Training A Power, Performance, and Thermal Perspective.pdf>) · [论文来源](https://doi.org/10.1145/3725843.3756111) · [GitHub · 实验代码](https://github.com/scai-tech/CharLLM-PPT)

系统测量分布式训练在不同 GPU、模型和并行策略下的性能、功耗与温度。分析张量、流水线、数据和专家并行及计算通信重叠之间的相互影响，为系统配置提供实测依据。

**阅读重点：** 改变并行策略或计算通信重叠时，性能和能耗是否同步改善？

[返回方向导航](#研究方向导航)

---

<a id="systems"></a>

## 04 · 集群与运行管理

围绕资源共享、平台部署、故障恢复、能效和监控，提高集群运行效率。

<a id="paper-32"></a>

### 28 · C4

**训练异常检测与通信优化**

> Enhancing Large-Scale AI Training Efficiency: The C4 Solution for Real-Time Anomaly Detection and Communication Optimization

**HPCA 2025** · [本地 PDF](<HPCA/2025/Enhancing Large-Scale AI Training Efficiency The C4 Solution for Real-Time Anomaly Detection and Communication Optimization.pdf>) · [论文来源](https://arxiv.org/abs/2406.04594)

提出 C4，利用分布式训练集合通信的规律性发现硬件异常并定位问题组件。结合通信流量规划减少网络竞争，提高大规模 GPU 训练集群的有效运行效率。

**阅读重点：** 集合通信的规律性如何同时用于异常定位和流量规划？

<a id="paper-34"></a>

### 29 · DynamoLLM

**推理集群能效管理**

> DynamoLLM: Designing LLM Inference Clusters for Performance and Energy Efficiency

**HPCA 2025** · [本地 PDF](<HPCA/2025/DynamoLLM Designing LLM Inference Clusters for Performance and Energy Efficiency.pdf>) · [论文来源](https://doi.org/10.1109/hpca61900.2025.00102)

研究 LLM 推理集群的时延目标与能耗之间的权衡。DynamoLLM 根据负载变化动态调整实例数量、模型并行配置和 GPU 频率，在满足服务要求的前提下降低能耗与成本。

**阅读重点：** 实例数量、模型并行度与频率如何共同满足时延要求？

<a id="paper-39"></a>

### 30 · eGPU

**生产集群弹性 GPU 共享**

> eGPU: Production-Scale Elastic Sharing over 10,000 GPUs

**HPCA 2026** · [本地 PDF](<HPCA/2026/eGPU_Production-Scale_Elastic_Sharing_Over_10000_GPUs.pdf>) · [论文来源](https://doi.org/10.1109/hpca68181.2026.11408556)

提出面向生产集群的弹性 GPU 共享框架 eGPU。支持运行时调整共享实例资源，同时支持实例间 NVLink/NCCL 通信，并与集群编排系统集成。

**阅读重点：** 共享实例动态调整资源时，如何支持跨 GPU 通信？

<a id="paper-44"></a>

### 31 · RAP

**预处理与训练共享 GPU**

> RAP: Resource-aware Automated GPU Sharing for Multi-GPU Recommendation Model Training and Input Preprocessing

**ASPLOS 2024** · [本地 PDF](<ASPLOS/2024/RAP Resource-aware Automated GPU Sharing forMulti-GPU Recommendation Model Training andInput Preprocessing.pdf>) · [论文来源](https://doi.org/10.1145/3620665.3640406) · [GitHub](https://github.com/Ash-Zheng/RAP-artifacts)

研究多 GPU 推荐模型训练时，如何利用尚未用满的 GPU 资源执行输入预处理。RAP 结合代价模型、内核融合和调度搜索，提高端到端训练吞吐量，减少额外 CPU 预处理资源需求。

**阅读重点：** 如何利用训练留下的资源空隙，又避免预处理拖慢训练？

<a id="paper-46"></a>

### 32 · Heet

**异构集群弹性训练**

> Heet: Accelerating Elastic Training in Heterogeneous Deep Learning Clusters

**ASPLOS 2024** · [本地 PDF](<ASPLOS/2024/Heet Accelerating Elastic Training in HeterogeneousDeep Learning Clusters.pdf>) · [论文来源](https://doi.org/10.1145/3620665.3640375)

研究异构 GPU 集群中弹性训练任务扩缩容的效率问题。Heet 估计不同资源配置的扩展收益，并通过任务匹配和资源调整协调扩展效率与集群调度效率。

**阅读重点：** 增加 GPU 的扩展收益如何影响任务匹配与资源调整？

<a id="paper-54"></a>

### 33 · Vela

**虚拟化训练与 GPUDirect RoCE**

> Vela: A Virtualized LLM Training System with GPU Direct RoCE

**ASPLOS 2025** · [本地 PDF](<ASPLOS/2025/Vela A Virtualized LLM Training System With GPUDirect RoCE.pdf>) · [论文来源](https://doi.org/10.1145/3676641.3716280) · [GitHub · 论文实验目录](https://github.com/IBM/HCIR/tree/main/vela_asplos2025)

介绍基于通用硬件、虚拟机和 RoCE 网络构建的大模型训练平台 Vela。重点是让虚拟化 GPU 与网卡之间支持高效数据传输，并总结大规模部署和运行经验。

**阅读重点：** 虚拟机中的 GPU 与网卡如何实现高效的数据传输？

<a id="paper-56"></a>

### 34 · MoC-System

**MoE 训练容错**

> MoC-System: Efficient Fault Tolerance for Sparse Mixture-of-Experts Model Training

**ASPLOS 2025** · [本地 PDF](<ASPLOS/2025/MoC-System Efficient Fault Tolerance for Sparse Mixture-of-Experts Model Training.pdf>) · [论文来源](https://doi.org/10.1145/3676641.3716006)

针对 MoE 专家参数多、检查点开销大的问题，提出 MoC-System。通过选择性专家检查点、结合并行策略的分片和异步持久化，降低分布式训练的容错开销。

**阅读重点：** 选择性专家检查点如何降低持久化成本，并支持故障恢复？

<a id="paper-70"></a>

### 35 · Pulse

**微秒级流量监控训练**

> Fine-grained and Non-intrusive LLM Training Monitoring via Microsecond-level Traffic Measurement

**ASPLOS 2026** · [本地 PDF](<ASPLOS/2026/Fine-grained and Non-intrusive LLM TrainingMonitoring via Microsecond-level Traffic Measurement.pdf>) · [论文来源](https://doi.org/10.1145/3779212.3790163)

提出 Pulse，通过网卡上的微秒级 RDMA 流量测量还原训练通信操作的行为。无需修改训练代码或通信库即可进行细粒度监控，帮助定位大规模 GPU 训练中的异常。

**阅读重点：** 网卡流量如何映射回训练通信行为，并帮助定位异常？

[返回方向导航](#研究方向导航)

---

<a id="security"></a>

## 05 · 安全与机密计算

在保护跨 GPU 数据的同时，降低加密、认证和安全元数据带来的额外开销。

<a id="paper-28"></a>

### 36 · 安全元数据管理

**安全通信的缓冲与批处理**

> Supporting Secure Multi-GPU Computing with Dynamic and Batched Metadata Management

**HPCA 2024** · [本地 PDF](<HPCA/2024/Supporting_Secure_Multi-GPU_Computing_with_Dynamic_and_Batched_Metadata_Management.pdf>) · [论文来源](https://doi.org/10.1109/hpca57654.2024.00025)

研究加密多 GPU 通信中一次性密码流缓冲区和安全元数据带来的额外开销。根据 GPU 通信模式动态分配缓冲区，并对元数据传输分批合并，提高安全通信效率。

**阅读重点：** 密码流缓冲区和元数据为什么成为瓶颈，动态分配与批处理如何配合？

<a id="paper-36"></a>

### 37 · SCALE

**机密训练的通信优化**

> SCALE: Tackling Communication Bottlenecks in Confidential Distributed Machine Learning

**HPCA 2026** · [本地 PDF](<HPCA/2026/SCALE_Tackling_Communication_Bottlenecks_in_Confidential_Distributed_Machine_Learning.pdf>) · [论文来源](https://doi.org/10.1109/hpca68181.2026.11408582)

研究机密计算环境中 GPU 间加密和认证对集合通信的额外影响。提出利用空闲 GPU 资源的协同加密与通信优化方案；安全模式部分结果采用基于硬件规格的建模估计。

**阅读重点：** 加密与认证如何改变集合通信成本；哪些结果来自测量，哪些来自建模？

[返回方向导航](#研究方向导航)

---

<a id="applications"></a>

## 06 · 其他应用与执行框架

考察零知识证明等计算任务如何扩展到多 GPU，以及通用分布式程序如何融合执行。

<a id="paper-45"></a>

### 38 · DistMSM

**多 GPU 多标量乘法**

> Accelerating Multi-Scalar Multiplication for Efficient Zero Knowledge Proofs with Multi-GPU Systems

**ASPLOS 2024** · [本地 PDF](<ASPLOS/2024/Accelerating Multi-Scalar Multiplication for EfficientZero Knowledge Proofs with Multi-GPU Systems.pdf>) · [论文来源](https://doi.org/10.1145/3620666.3651364)

提出用于零知识证明的多 GPU 多标量乘法方案 DistMSM。针对跨 GPU 扩展瓶颈改造算法，并优化椭圆曲线运算内核、寄存器压力和大整数计算。

**阅读重点：** 多标量乘法拆到多张 GPU 后，计算与数据交换如何组织？

<a id="paper-60"></a>

### 39 · UniNTT

**多 GPU 数论变换**

> Accelerating Number Theoretic Transform with Multi-GPU Systems for Efficient Zero Knowledge Proof

**ASPLOS 2025** · [本地 PDF](<ASPLOS/2025/Accelerating Number Theoretic Transform withMulti-GPU Systems for Efficient Zero Knowledge Proof.pdf>) · [论文来源](https://doi.org/10.1145/3669940.3707241)

提出多 GPU 数论变换算法 UniNTT，解决零知识证明中 NTT 置换访存导致的大量通信。通过递归分解在 warp、线程块、GPU 和多 GPU 层次统一组织计算与优化。

**阅读重点：** 递归分解如何在 warp、线程块与多 GPU 层次减少置换通信？

<a id="paper-61"></a>

### 40 · Diffuse

**分布式任务与内核融合**

> Composing Distributed Computations Through Task and Kernel Fusion

**ASPLOS 2025** · [本地 PDF](<ASPLOS/2025/Composing Distributed Computations Through Task and Kernel Fusion.pdf>) · [论文来源](https://doi.org/10.1145/3669940.3707216)

提出 Diffuse，在分布式任务运行时中同时融合任务及其内部计算内核。通过中间表示与即时编译发现跨函数、跨库的优化机会，加速通用多 GPU 数值计算应用。

**阅读重点：** 任务级融合与 kernel 融合分别消除哪些运行时开销？

[返回方向导航](#研究方向导航)

---

*概括侧重研究问题与主要机制；“阅读重点”是阅读提示。具体实验条件、性能数据与适用边界请以论文正文为准。*

*GitHub 链接依据论文正文、实验附录及项目页面核对，标注作者实现或对应实验仓库。未标注的条目不代表没有公开代码。链接核对日期：2026-09-20。*
