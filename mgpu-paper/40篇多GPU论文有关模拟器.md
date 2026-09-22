# 40 篇多 GPU 论文：模拟器使用核验


> 直接提出模拟器：2 篇。使用模拟器开展实验：24 篇。另有性能建模工具 MAD-Max 1 篇。


## 一、做模拟器：2 篇

| 论文 | 会议 | 模拟层次与贡献 |
| :--- | :--- | :--- |
| [TrioSim: A Lightweight Simulator for Large-Scale DNN Workloads on Multi-GPU Systems](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ISCA/2025/TrioSim A Lightweight Simulator for Large-Scale DNNWorkloads on Multi-GPU Systems.pdf>) | ISCA 2025 | 算子级性能模型、单 GPU trace 外推与轻量级网络模拟，预测多 GPU DNN 执行。 |
| [vTrain: A Simulation Framework for Evaluating Cost-effective and Compute-optimal Large Language Model Training](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/MICRO/2024/vTrain A Simulation Framework for Evaluating Cost-effective and Compute-optimal Large Language Model Training.pdf>) | MICRO 2024 | 基于 profiling 和任务图推进时间的训练模拟器，预测不同并行策略下的训练时间与成本。 |

这两篇面向分布式 DNN/LLM 执行性能建模，不属于 Accel-Sim、MGPUSim 那样的详细 GPU 微架构模拟器。

## 二、用模拟器做实验：24 篇

其中使用 GPU 执行模拟器的有 **12 篇**；其余为加速器、网络、分布式系统或集群级模拟。

### GPU 执行模拟：12 篇

| 论文 | 会议 | 使用的模拟工具 | 用途与边界 |
| :--- | :--- | :--- | :--- |
| [NetCrafter: Tailoring Network Traffic for Non-Uniform Bandwidth Multi-GPU Systems](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ISCA/2025/NetCrafter Tailoring Network Traffic for Non-UniformBandwidth Multi-GPU Systems.pdf>) | ISCA 2025 | MGPUSim（网络由 Akita 支持） | 模拟非均匀带宽多 GPU 系统、流量管理和页面访问。 |
| [Accelerating MoE with Dynamic In-Switch Computing on Multi-GPUs](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ISCA/2026/Accelerating MoE with Dynamic In-Switch Computing on Multi-GPUs.pdf>) | ISCA 2026 | 扩展 Accel-Sim + BookSim2 | 模拟 GH200 NVL32 风格系统和动态交换机内计算；正文报告另用真机校验。 |
| [RoCC: Harnessing Raster Operations Pipeline for Efficient Tensor Collective Communication](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ISCA/2026/RoCC_Harnessing_Raster_Operations_Pipeline_for_Efficient_Tensor_Collective_Communication.pdf>) | ISCA 2026 | MGPUSim + ASTRA-sim | MGPUSim 评估 ROP 辅助通信；ASTRA-sim 评估端到端 LLM 收益。 |
| [Coarse-Grained Duplication First, Fine-Grained Deduplication Later: Duplication-Centric Multi-GPU Memory Management](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ISCA/2026/Coarse-Grained_Duplication_First_Fine-Grained_Deduplication_Later_Duplication-Centric_Multi-GPU_Memory_Management.pdf>) | ISCA 2026 | MGPUSim | 评估多 GPU 页面复制、去重及相关硬件。 |
| [Reducing Page Faults via Invalidation-Based Mapping Propagation in Multi-GPU Systems](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ISCA/2026/Reducing_Page_Faults_via_Invalidation-Based_Mapping_Propagation_in_Multi-GPU_Systems.pdf>) | ISCA 2026 | MGPUSim | 扩展访问计数驱动页面迁移，评估映射传播、缺页与地址翻译。 |
| [LIBRA: A High-Accuracy, Cost-Aware, and Coordinated Multi-GPU Page Prefetcher](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ISCA/2026/LIBRA_A_High-Accuracy_Cost-Aware_and_Coordinated_Multi-GPU_Page_Prefetcher.pdf>) | ISCA 2026 | MGPUSim | 正文明确说明使用 MGPUSim 评估多 GPU 协同页面预取。 |
| [MoE-Hub: Taming Software Complexity for Seamless MoE Overlap with Hardware-Accelerated Communication on Multi-GPU Systems](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ISCA/2026/MoE-Hub Taming Software Complexity for Seamless MoE Overlap with Hardware-Accelerated Communication on Multi-GPU Systems.pdf>) | ISCA 2026 | 扩展 Accel-Sim + BookSim2 | 模拟 DGX-H800 风格多 GPU 系统、NVLink 网络及 MoE-Hub 硬件。 |
| [Supporting Secure Multi-GPU Computing with Dynamic and Batched Metadata Management](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/HPCA/2024/Supporting_Secure_Multi-GPU_Computing_with_Dynamic_and_Batched_Metadata_Management.pdf>) | HPCA 2024 | MGPUSim | 扩展模拟器支持安全设备间通信，评估密码流缓冲区和元数据管理。 |
| [GRIT: Enhancing Multi-GPU Performance with Fine-Grained Dynamic Page Placement](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/HPCA/2024/GRIT_Enhancing_Multi-GPU_Performance_with_Fine-Grained_Dynamic_Page_Placement.pdf>) | HPCA 2024 | MGPUSim | 评估多 GPU 页面迁移、访问计数和复制策略。 |
| [OASIS: Object-Aware Page Management for Multi-GPU Systems](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/HPCA/2025/OASIS Object-Aware Page Management for Multi-GPU Systems.pdf>) | HPCA 2025 | MGPUSim | 评估对象感知页面管理，包含 4、8、16 GPU 配置。 |
| [Towards Compute-Aware In-Switch Computing for LLMs Tensor-Parallelism on Multi-GPU Systems](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/HPCA/2026/Towards Compute-Aware In-Switch Computing for LLMs Tensor-Parallelism on Multi-GPU Systems.pdf>) | HPCA 2026 | 扩展 Accel-Sim + BookSim2 | 模拟 DGX-H100 风格系统，扩展 Hopper、NVLS multimem 与交换机内计算。 |
| [T3: Transparent Tracking & Triggering for Fine-grained Overlap of Compute & Collectives](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ASPLOS/2024/T3 Transparent Tracking & Triggering for Fine-grained Overlap of Compute & Collectives.pdf>) | ASPLOS 2024 | 扩展 Accel-Sim | 基于对称 GPU 执行与额外通信流量注入模拟多 GPU；不应自行加上 BookSim2。 |

### 加速器、网络与分布式系统模拟：10 篇

| 论文 | 会议 | 使用的模拟工具 | 用途与边界 |
| :--- | :--- | :--- | :--- |
| [Chimera: Communication Fusion for Hybrid Parallelism in Large Language Models](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ISCA/2025/Chimera Communication Fusion for Hybrid Parallelism in LargeLanguage Models.pdf>) | ISCA 2025 | SCALE-Sim v2 + BookSim2 | 评估计算、集合通信与通信融合；SCALE-Sim 模拟 TPU 式脉动阵列，不是 GPU 微架构模拟器。 |
| [TRACI: Network Acceleration of Input-Dynamic Communication for Large-Scale Deep Learning Recommendation Model](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ISCA/2025/TRACI Network Acceleration of Input-Dynamic Communicationfor Large-Scale Deep Learning Recommendation Model.pdf>) | ISCA 2025 | gem5 Garnet + ASTRA-sim | 扩展 Garnet 评估嵌入层聚合网络；结合 ASTRA-sim 估计其余计算及端到端收益。 |
| [DisDP: Disaggregating Compute, Network, and Storage for Model-Sharded Data-Parallel Training](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ISCA/2026/DisDP Disaggregating Compute, Network, and Storage for Model-Sharded Data-Parallel Training.pdf>) | ISCA 2026 | ASTRA-sim | 除原型实验外，用于大规模集群扩展性和成本性能分析。 |
| [Scalable Synthesis of Distributed LLM Workloads Through Symbolic Tensor Graphs](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ISCA/2026/Scalable Synthesis of Distributed LLM Workloads Through Symbolic Tensor Graphs.pdf>) | ISCA 2026 | ASTRA-sim、SimAI、SCALE-Sim；另接 Genie | STAGE 本身生成执行图；用下游模拟器验证内存、运行时间和可移植性。Genie 是 RDMA 流量仿真/重放工具，应与 GPU 模拟器区分。 |
| [TACOS: Topology-Aware Collective Algorithm Synthesizer for Distributed Machine Learning](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/MICRO/2024/TACOS Topology-Aware Collective Algorithm Synthesizer for Distributed Machine Learning.pdf>) | MICRO 2024 | ASTRA-sim + 自建拥塞感知网络后端 | TACOS 本身是集合通信算法合成器；使用模拟器评估合成算法及完整工作负载。 |
| [NetZIP: Algorithm/Hardware Co-design of In-network Lossless Compression for Distributed Large Model Training](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/MICRO/2025/NetZIP AlgorithmHardware Co-design of In-network LosslessCompression for Distributed Large Model Training.pdf>) | MICRO 2025 | SimAI | 结合 FPGA 原型与测量数据，评估 512 GPU 下的通信压缩收益。 |
| [Characterizing the Efficiency of Distributed Training: A Power, Performance, and Thermal Perspective](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/MICRO/2025/Characterizing the Efficiency of Distributed Training A Power, Performance, and Thermal Perspective.pdf>) | MICRO 2025 | ASTRA-sim | 主体为真机性能、功耗和温度分析；另结合实测数据外推到最多 8K GPU。 |
| [SCALE: Tackling Communication Bottlenecks in Confidential Distributed Machine Learning](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/HPCA/2026/SCALE_Tackling_Communication_Bottlenecks_in_Confidential_Distributed_Machine_Learning.pdf>) | HPCA 2026 | 修改后的 ASTRA-sim + Chakra traces | 结合真实执行轨迹和加密成本建模，评估机密计算下的集合通信。 |
| [MoC-System: Efficient Fault Tolerance for Sparse Mixture-of-Experts Model Training](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ASPLOS/2025/MoC-System Efficient Fault Tolerance for Sparse Mixture-of-Experts Model Training.pdf>) | ASPLOS 2025 | ASTRA-SIM | 除真实 MoE 训练实验外，在 §6.2.4 扩展 GPU 数量、模型规模和硬件配置。 |
| [Fine-grained and Non-intrusive LLM Training Monitoring via Microsecond-level Traffic Measurement](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ASPLOS/2026/Fine-grained and Non-intrusive LLM TrainingMonitoring via Microsecond-level Traffic Measurement.pdf>) | ASPLOS 2026 | SimAI | 除 SmartNIC 真机监控实验外，在 512 GPU 模拟环境验证并行策略识别。 |

### 集群级自建模拟：2 篇

| 论文 | 会议 | 使用的模拟工具 | 用途与边界 |
| :--- | :--- | :--- | :--- |
| [DynamoLLM: Designing LLM Inference Clusters for Performance and Energy Efficiency](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/HPCA/2025/DynamoLLM Designing LLM Inference Clusters for Performance and Energy Efficiency.pdf>) | HPCA 2025 | 自建离散时间模拟器（未给独立名称） | 除真机实验外，以生产请求轨迹评估长时间、大规模场景的能耗。 |
| [Heet: Accelerating Elastic Training in Heterogeneous Deep Learning Clusters](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ASPLOS/2024/Heet Accelerating Elastic Training in HeterogeneousDeep Learning Clusters.pdf>) | ASPLOS 2024 | 基于实测吞吐量的集群调度模拟平台（未给独立名称） | 除 24 GPU 真机实验外，用 120 GPU 模拟平台评估弹性调度。 |

## 三、相邻的性能建模工具：1 篇

**[MAD-Max Beyond Single-Node: Enabling Large Machine Learning Model Acceleration on Distributed Systems](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ISCA/2024/MAD-Max Beyond Single-Node Enabling Large Machine Learning Model Acceleration on Distributed Systems.pdf>) · ISCA 2024**

MAD-Max 提出分布式机器学习性能建模框架，用于比较并行策略与软硬件设计。本文将其单列为性能模型，避免与 TrioSim、vTrain 的模拟框架混为一类。研究模拟与建模工具时，建议同时阅读 MAD-Max 和 STAGE。

## 四、未计入上述模拟器统计：13 篇

| 论文 | 会议 | 核对结果 |
| :--- | :--- | :--- |
| [Lit Silicon: A Case Where Thermal Imbalance Couples Concurrent Execution in Multiple GPUs](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ISCA/2026/Lit Silicon A Case Where Thermal Imbalance Couples Concurrent Execution in Multiple GPUs.pdf>) | ISCA 2026 | 多 GPU 真机测量及分析性性能/功率模型；未见使用系统模拟器进行实验。 |
| [SkipReduce: (Interconnection) Network Sparsity to Accelerate Distributed Machine Learning](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/MICRO/2025/SkipReduce (Interconnection) Network Sparsity to AccelerateDistributed Machine Learning.pdf>) | MICRO 2025 | 真实训练与修改后的 NCCL；文中模拟 top-k 行为是算法行为对照，不是系统模拟器。 |
| [Enhancing Large-Scale AI Training Efficiency: The C4 Solution for Real-Time Anomaly Detection and Communication Optimization](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/HPCA/2025/Enhancing Large-Scale AI Training Efficiency The C4 Solution for Real-Time Anomaly Detection and Communication Optimization.pdf>) | HPCA 2025 | 生产训练集群监测与通信优化；未见系统模拟器实验。 |
| [eGPU: Production-Scale Elastic Sharing over 10,000 GPUs](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/HPCA/2026/eGPU_Production-Scale_Elastic_Sharing_Over_10000_GPUs.pdf>) | HPCA 2026 | 生产 GPU 共享集群与实际部署；scientific simulations 指应用负载。 |
| [TCCL: Discovering Better Communication Paths for PCIe GPU Clusters](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ASPLOS/2024/TCCL Discovering Better Communication Pathsfor PCIe GPU Clusters.pdf>) | ASPLOS 2024 | PCIe GPU 集群真实通信测量与路径搜索；未见系统模拟器实验。 |
| [RAP: Resource-aware Automated GPU Sharing for Multi-GPU Recommendation Model Training and Input Preprocessing](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ASPLOS/2024/RAP Resource-aware Automated GPU Sharing forMulti-GPU Recommendation Model Training andInput Preprocessing.pdf>) | ASPLOS 2024 | 真实 GPU 上的预处理与训练；代价模型不单独算作模拟器。 |
| [Accelerating Multi-Scalar Multiplication for Efficient Zero Knowledge Proofs with Multi-GPU Systems](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ASPLOS/2024/Accelerating Multi-Scalar Multiplication for EfficientZero Knowledge Proofs with Multi-GPU Systems.pdf>) | ASPLOS 2024 | 真实多 GPU 零知识证明计算实验；未见系统模拟器实验。 |
| [Vela: A Virtualized LLM Training System with GPU Direct RoCE](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ASPLOS/2025/Vela A Virtualized LLM Training System With GPUDirect RoCE.pdf>) | ASPLOS 2025 | 真实虚拟机、GPU 和网络系统实验；正文提到的 HPC simulations 是应用负载。 |
| [Aqua: Network-Accelerated Memory Offloading for LLMs in Scale-Up GPU Domains](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ASPLOS/2025/Aqua Network-Accelerated Memory Offloading for LLMs in Scale-Up GPU Domains.pdf>) | ASPLOS 2025 | 真实多 GPU 推理实验；用请求生成器或虚拟服务模拟突发负载/扩容行为，不算系统模拟器。正文明确本工作直接从硬件测量。 |
| [Accelerating Number Theoretic Transform with Multi-GPU Systems for Efficient Zero Knowledge Proof](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ASPLOS/2025/Accelerating Number Theoretic Transform withMulti-GPU Systems for Efficient Zero Knowledge Proof.pdf>) | ASPLOS 2025 | 真实多 GPU 数论变换实验；未见系统模拟器实验。 |
| [Composing Distributed Computations Through Task and Kernel Fusion](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ASPLOS/2025/Composing Distributed Computations Through Task and Kernel Fusion.pdf>) | ASPLOS 2025 | 真实多 GPU 分布式执行实验；物理模拟是运行的应用，不是用于评估系统的模拟器。 |
| [MSCCL++: Rethinking GPU Communication Abstractions for AI Inference](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ASPLOS/2026/MSCCL++ Rethinking GPU Communication Abstractions for AI Inference.pdf>) | ASPLOS 2026 | 真实 GPU 通信与推理实验；未见系统模拟器实验。 |
| [SuperOffload: Unleashing the Power of Large-Scale LLM Training on Superchips](<C:/Users/only/Desktop/gpgpu项目/论文/mgpu-paper/ASPLOS/2026/SuperOffload Unleashing the Power of Large-Scale LLM Training on Superchips.pdf>) | ASPLOS 2026 | Superchip 真机训练实验；未见系统模拟器实验。 |

---
