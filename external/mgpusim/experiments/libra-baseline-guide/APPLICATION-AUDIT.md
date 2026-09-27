# LIBRA 全部应用与当前 MGPUSim 的对应关系

核对日期：2026-09-28。目标源码：`/home/only/projects/mgpu-project/external/mgpusim`；HEAD：`7c09be3d28637f87b0f343bee844a54de159b932`。本次直接读取虚拟机源码。用户已将 R9 Nano 的 L2 数据缓存从 16 路改为 8 路；本次未修改该文件或任何模拟器源码，也没有运行新的模拟。

论文来源：LIBRA 中文译本，第 8 页表 III（配置）、表 IV（应用）及第 V 节；第 2 页图 1 和背景说明。表 IV 含 23 个应用，列出的内存占用是每 GPU 的足迹。

## TLB 层级的结论

当前 R9 Nano 与 MI300X 构建路径都是 L1 TLB -> L2 TLB -> MMU。检查了两个 builder 的 `buildL2TLB()`，其中下级 `TranslationProviderMapper` 直接指向 MMU 的 Top 端口；在 `amd/` 的 Go 源码中未找到名为 L3TLB/L3 TLB 的组件。

这说明当前这份代码没有接入 L3 TLB，不能泛化为所有 MGPUSim 分支都没有或框架不能支持 L3。此前新增 faultvm 改的是 MMU 及主机映射缺失路径，没有添加 L3。LIBRA 表 III 则明确包含三级 TLB，故论文模型与当前安装版本不同。

证据入口：

- `amd/samples/runner/timingconfig/r9nano/builder.go:666`：`buildL2TLB()`，下级指向 `b.mmu`。
- `amd/samples/runner/timingconfig/mi300x/builder.go:830`：同样的 L2 到 MMU 连接。
- `amd/samples/runner/timingconfig/fault.go`：此前新增的独立 MMU/HostMMU 模型。

## 全部 23 项

下表中的“有”表示源码中存在对应实现路径，不等于本次已经完成四 GPU 数值与计时验证。“统一分配”指确实调用 `AllocateUnifiedMemory`，不等于完整迁移、换页和一致性协议已经具备。

| 论文缩写 | 应用 | 每 GPU 足迹 | 当前 sample | 多 GPU 执行与统一内存状态 |
|---|---|---:|---|---|
| SC | 简单卷积 | 32 MB | simpleconvolution | 有工作范围划分、逐 GPU 队列提交；有统一分配路径 |
| C2D | 二维卷积 | 23 MB | conv2d | 显式拒绝多个 GPU；UVM 标志没有传入张量分配 |
| MM | 矩阵乘法 | 8 MB | matrixmultiplication | 有工作范围划分、逐 GPU 队列提交；有统一分配路径 |
| MT | 矩阵转置 | 16 MB | matrixtranspose | 有工作范围划分、逐 GPU 队列提交；有统一分配路径 |
| FIR | 有限脉冲响应 | 38 MB | fir | 有工作范围划分、逐 GPU 队列提交；有统一分配路径；此前仅双 GPU demand/ideal 实跑验证 |
| ST | 二维模板计算 | 8 MB | stencil2d | 接受 GPU 列表并创建队列，但实际 kernel 通过同一 context 提交到最后选中的 GPU；有统一分配路径 |
| IM2COL | 图像转列 | 20 MB | im2col | 显式拒绝多个 GPU；UVM 标志没有传入张量分配 |
| FFT | 快速傅里叶变换 | 12 MB | fft | 接受 GPU 列表并创建队列，但没有将计算分发到这些队列；有统一分配路径 |
| LeNet | LeNet 网络 | 6 MB | lenet | 有多 GPU 数据并行训练；SetUnifiedMemory 明确 panic |
| VGG | 16 层 VGG 网络 | 55 MB | vgg16 | 有多 GPU 数据并行训练；SetUnifiedMemory 明确 panic |
| BS | 双调排序 | 7 MB | bitonicsort | 有每轮排序工作划分、逐 GPU 队列提交；有统一分配路径 |
| BERT-T | BERT Tiny | 68 MB | 未找到 | 当前 amd 样例/基准目录未找到对应名称入口 |
| BERT-M | BERT Mini | 136 MB | 未找到 | 同上 |
| BERT-ME | BERT Medium | 272 MB | 未找到 | 同上 |
| BERT-B | BERT Base | 544 MB | 未找到 | 同上 |
| GPT2-M | GPT-2 Mini | 65 MB | 未找到 | 同上 |
| GPT2 | GPT-2 | 196 MB | 未找到 | 同上 |
| BFS | 广度优先搜索 | 8 MB | bfs | 显式拒绝多个 GPU；有统一分配路径 |
| PR | PageRank | 8 MB | pagerank | 接受 GPU 列表并创建队列，但没有将计算分发到这些队列；有统一分配路径 |
| MIS | 最大独立集 | 4 MB | 未找到 | 当前 amd 样例/基准目录未找到对应名称入口 |
| SSSP | 单源最短路径 | 14 MB | 未找到 | 同上 |
| SPMV | 稀疏矩阵向量乘法 | 14 MB | spmv | 接受 GPU 列表并创建队列，但没有将计算分发到这些队列；有统一分配路径 |
| KM | K-means | 33 MB | kmeans | 有数据范围划分、逐 GPU 队列提交；有统一分配路径 |

按当前代码状态计数：6 项有多 GPU 划分和统一分配路径；4 项接受列表但未实现独立 GPU 工作分发；3 项显式限制单 GPU；2 项有多 GPU 训练但拒绝 UVM；8 项未找到对应入口。总计 23 项。

表 IV 未逐项标注原套件归属。当前实现所在目录能够说明当前源码的组织，不能据此证明它与作者使用的套件版本、输入集或算法实现完全一致。

## 为什么不能只检查 GPU 列表

`stencil2d`、`fft`、`pagerank`、`spmv` 的 Run 中循环调用 SelectGPU，并为每个 GPU 创建队列。但后续计算使用的是 `LaunchKernel(b.context, ...)`，没有使用之前收集的多条队列，也没有在计算阶段重新选择各个独立 GPU。

Driver 的 `SelectGPU` 会覆盖 `context.currentGPUID`；`LaunchKernel` 又根据 context 新建队列。因此，使用普通 `-gpus=1,2,3,4` 路径时，主要计算提交到最后选中的 GPU。这是沿调用链得出的静态判断，未在本轮进行四 GPU 运行测量。`-unified-gpus` 的逻辑设备模式另有实现，不能作为本次独立四 GPU UVM 路径的替代品。

源码证据：

- `amd/driver/api.go:57`：SelectGPU 设置 currentGPUID；`:103`：CreateCommandQueue 从它取得 GPUID。
- `amd/driver/kernel.go:96`：LaunchKernel 新建并排空一个当前 context 的队列。
- `amd/benchmarks/shoc/stencil2d/stencil2d.go:205`：Run；`:299`：exec。
- `amd/benchmarks/shoc/fft/fft.go:129`：Run；`:165`：exec。
- `amd/benchmarks/heteromark/pagerank/pagerank.go:128`：Run；`:281`：exec。
- `amd/benchmarks/shoc/spmv/spmv.go:123`：Run；`:186`：exec。

## 论文为什么能做四 GPU 实验

LIBRA 第 V 节明确写明：基于 MGPUSim 模拟四 GPU，每 GPU 独立页表/GMMU；表 III 明确规定三级 TLB；应用来自四个套件，共 23 项。这是论文描述的实验环境，不能直接等同于任意公开 MGPUSim checkout 的默认模型与所有样例。

论文模型需要配套的硬件扩展和应用执行路径；已有多 GPU 应用可在该环境执行，尚无多 GPU/UVM 路径的同名应用则需要相应适配。当前没有取得作者所用的完整代码版本，因此不能说明每个应用究竟如何切分工作、哪些是沿用旧分支、哪些是新增移植，也不能保证当前 sample 的默认参数还原论文足迹。

一个单 GPU kernel 可以被复用在多 GPU 应用中。例如一个批量卷积有 128 张图片，host 可以将批量分成四份，为每 GPU 设置输入/输出区间并提交 kernel，再同步结果。跨 GPU 的权重/输入共享和 UVM 放置仍需显式设计。这个例子只是解释如何适配，不代表已确认 LIBRA 的实际划分方法。

## 对当前实验的建议

如果先研究多 GPU 映射缺页，优先从已有多 GPU 与统一分配路径的 MM、MT、FIR、BS、SC、KM 里选择。它们减少应用改造工作，但仍需由用户执行小规模数值校验并确认每 GPU 工作量，再扩大规模。

如果研究目标必须是逐 GPU L3 TLB 命中率，则任何应用都需要先接入 L3 模型；当前 L2 的统计只能命名为 L2。应用支持 UVM 与硬件模型包含 L3 是两个独立条件。

