# L3 TLB 实验结果

运行入口位于 `../../benchmark/l3 tlb/`。原始结果、可执行文件和汇总按应用存放：

```text
results/l3 tlb/
├── fir/
│   ├── 2026-09-29_23-10-00_length-4096/
│   └── latest -> 最近一次通过验证与统计检查的目录
├── sc/
├── mm/
├── mt/
├── bs/
└── km/
```

时间和目录名仅为示例。每轮新建目录，同一秒重复运行会追加序号。
失败运行保留日志，但不更新 `latest`。`latest` 是符号链接，不是结果副本。

每轮主要文件：`metrics.sqlite3`、`run.log`、`build.log`、`l2-hit-rate.txt`、
`l3-hit-rate.txt`、`check.log`、`status.txt`、`run.json`、`command.sh`、
`commit.txt`、`git-status.txt`、`working-tree.diff`、`staged.diff`、`go-version.txt`、
`config.json` 和本轮编译的可执行文件。

`run.json` 保存生效参数、完整命令、结果状态和二进制 SHA-256。
`command.sh` 是历史命令记录；直接执行会复用旧输出前缀。新实验请从 benchmark 入口运行。
Git diff 不包含未跟踪文件，且默认不会保存已修改二进制的内容；上述记录不等于完整源码备份。

应用结果目录已被这里的 `.gitignore` 忽略，避免把数据库和可执行文件意外提交。
需要版本管理的精选结果可以另存为本目录下的 Markdown/CSV 文件，或明确使用 `git add -f`。
