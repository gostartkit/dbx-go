# 贡献指南

[English Version](CONTRIBUTING.md)

## 开发

- 提交前请运行 `make check`。它会检查格式、执行 `go vet`、跑测试并完成一次完整构建。
- 本地迭代时，常用流程是 `make fmt`、`make test` 和 `make build`。
- 保持项目以 REPL 为核心，不要在未经讨论的情况下扩展到文档范围之外的 MVP。
- 优先保持小文件、小函数、显式错误处理，并尽量先用标准库解决问题。

## 范围约束

- 引导式操作应始终是主要体验。直接 SQL 应保留为显式逃生舱，而不是默认路径。
- 不要在当前 `run sql` 入口之外继续扩展 raw SQL 工作流，除非先讨论。
- 不要引入大型 CLI 或交互框架。
- 不要加入 ORM、迁移系统、代理链、自动补全框架或 AI SQL 功能。

## 传输与依赖

- SSH 支持必须继续使用 `golang.org/x/crypto/ssh` 原生实现，不要 shell out 到 `ssh`。
- 代理支持必须继续限制为通过 `golang.org/x/net/proxy` 实现的 SOCKS5。
- 除非某个依赖已经在项目允许集合内，否则优先使用标准库。
- 避免引入 Cobra、Viper、PromptUI、Survey、readline、GORM、tablewriter 等框架。

## 评审要求

- 保持 `~/.config/dbx/` 下的配置布局不变。
- 不要记录或打印秘密信息。
- SSH 访问必须继续通过 Go SSH 库原生实现。
- 当命令行为变化时，文档和命令示例必须同步更新。
- 当用户可见的命令、工作流或示例变化时，`README.md` 和 `README.zh-CN.md` 需要一起更新。
