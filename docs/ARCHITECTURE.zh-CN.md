# dbx 架构走读

[English Version](ARCHITECTURE.md)

这份文档面向代码阅读，重点说明当前 `dbx` 仓库是怎样组织的、各个包负责什么、以及哪些内部行为在继续扩展时最值得注意。

## 总览

`dbx` 的核心思路很明确：REPL 和一次性 CLI 必须共享同一棵命令树、同一套校验规则、同一套模板解析逻辑，以及同一条执行流水线。

当前实现也正是按这个方向组织的：

- 进程入口尽量薄
- 命令注册集中管理
- 大部分用户行为收敛到 `internal/app`
- 先解析模板，再执行
- 传输细节下沉到 MySQL driver 层
- 用户状态统一落在 `~/.config/dbx/`

## 运行流程

### 1. 启动与模式选择

- [`cmd/dbx/main.go`](../cmd/dbx/main.go) 创建带信号感知的 context，然后把控制权交给 `app.NewCommandApp(...)`。
- [`internal/app/cli.go`](../internal/app/cli.go) 构建根 `cmd.App`，开启 REPL 模式，并注册全局 flags 与整棵命令树。
- 当前真正的 REPL 循环由 `pkg.gostartkit.com/cmd` 提供，而不是由本地 `internal/repl/` 中的代码实现。

### 2. 共享命令树

`internal/app` 是交互模式和非交互模式共同的编排层。

- `cli.go` 负责搭建共享命令树。
- `cli_*_commands.go` 负责注册命令分组和 CLI flag 面。
- `commands.go`、`row_commands.go`、`user_commands.go` 等文件承载 REPL 侧处理逻辑。
- `command_specs.go` 直接从同一棵命令树派生 help/usage 元数据，确保帮助文案、补全主题和可见命令路径保持一致。

这也是仓库里最关键的架构点：用户入口虽然不同，但最终都会汇聚到同一套应用逻辑。

### 3. 应用生命周期与会话状态

[`internal/app/app.go`](../internal/app/app.go) 持有长生命周期状态：

- prompt 实例
- 配置 store
- connector
- template service
- 当前 session
- 本地命令历史
- dry-run 开关
- 补全缓存

启动时，应用会加载历史和 session 状态。如果 `session.json` 指向上一次连接，`dbx` 会提示是否重连；重连后还会校验并恢复上次选中的数据库。

### 4. 配置与持久化

[`internal/config/store.go`](../internal/config/store.go) 负责 `~/.config/dbx/` 下的磁盘布局：

- connection 配置
- connection 级模板目录
- 全局模板
- `session.json`
- `history`
- `logs/audit.jsonl`

几个值得注意的实现细节：

- history 会持久化，并裁剪到最近 `1000` 条
- audit logging 是 best-effort
- `audit log` 默认读取最近 `50` 条
- connection 配置在 load/save 时都会校验
- `SessionFile` 仍兼容旧 JSON 字段名

[`internal/config/types.go`](../internal/config/types.go) 会补齐这些默认值：

- driver `mysql`
- 数据库端口 `3306`
- SSH 端口 `22`
- connect timeout `10s`
- query timeout `30s`

[`internal/config/diagnostics.go`](../internal/config/diagnostics.go) 负责严格的模式校验，并生成 `doctor` 使用的结构化诊断结果。

### 5. 模板与 operation 解析

`dbx` 不允许用户直接提交无限制 SQL。命令在执行前，必须先解析成模板或 operation。

模板相关职责主要在 [`internal/template/`](../internal/template/)：

- [`builtin.go`](../internal/template/builtin.go)：内置模板定义
- [`service.go`](../internal/template/service.go)：connection/global/builtin 三层模板加载与缓存
- [`render.go`](../internal/template/render.go)：变量渲染
- [`types.go`](../internal/template/types.go)：模板 schema、输入类型和 action 结构校验

查找优先级是：

```text
connection template
> global template
> builtin template
```

[`internal/app/operation_runtime.go`](../internal/app/operation_runtime.go) 会把模板暴露成 `exec` 使用的具名 operation。它内部合并了两类 provider：

- builtin operation
- 非 builtin 的已解析模板

也就是说，`exec <name>` 不是第二套执行系统，而是同一套模板执行机制的另一种入口。

### 6. 命令执行流水线

当前执行链路保持得很显式：

1. 解析 connection 和可选的 database 上下文
2. 解析模板或具名 operation
3. 合并 CLI/交互输入与模板默认值
4. 校验类型化输入
5. 生成执行计划和脱敏后的预览计划
6. 对变更型命令做确认
7. 真执行，或者返回 dry-run 结果

关键文件：

- [`internal/app/context_resolver.go`](../internal/app/context_resolver.go)
- [`internal/app/template_inputs.go`](../internal/app/template_inputs.go)
- [`internal/app/plan_support.go`](../internal/app/plan_support.go)
- [`internal/app/execution.go`](../internal/app/execution.go)

执行计划会携带这些信息：

- operation 名称
- 模板层级
- source
- transaction 标记
- 渲染后的 SQL action

如果计划要求事务，`execution.go` 会把所有 action 放进一个 SQL transaction 里执行，并记录最终是 committed 还是 rolled back。

### 7. 连接与传输栈

连接相关代码分层比较清楚：

- [`internal/connect/connect.go`](../internal/connect/connect.go)：带超时的 driver 分发
- [`internal/driver/mysql.go`](../internal/driver/mysql.go)：MySQL DSN 和打开连接
- [`internal/driver/mysql_transport.go`](../internal/driver/mysql_transport.go)：direct、SSH、proxy、proxy-ssh 四种拨号方式
- [`internal/driver/mysql_query.go`](../internal/driver/mysql_query.go)：查询辅助与结果整形

几个重要的传输行为：

- SSH 使用 `golang.org/x/crypto/ssh`
- SOCKS5 使用 `golang.org/x/net/proxy`
- SSH host key 通过 `known_hosts` 校验
- `DBX_KNOWN_HOSTS` 可以覆盖默认的 known_hosts 搜索路径
- 代理 URL 在面向用户输出前会先脱敏

`internal/driver/mysql_transport.go` 通过注册自定义 MySQL dialer 来支持这些链路，而不是 shell out 到外部工具，这也让传输层保持原生和可测试。

### 8. REPL 交互体验与补全

交互层本身非常轻量。

- [`internal/ui/prompt.go`](../internal/ui/prompt.go) 实现了 `Ask`、`Choose`、`Confirm`、`AskPassword`。
- 当 stdin 是终端时，密码输入会借助 `golang.org/x/term` 隐藏回显。
- history 和 prompt label 的管理主要在 `internal/app`。

但补全系统其实比表面更“重”一些：

- [`internal/app/completion*.go`](../internal/app/completion*.go) 负责构建补全结果
- [`internal/commandlang/`](../internal/commandlang/) 负责对当前输入做词法和语法分析
- 各类 completion provider 会把语法上下文、命令元数据和动态 resolver 数据合并起来

这意味着当前补全已经是“语法感知”的，而不只是简单前缀匹配。`commandlang` 的角色是给补全和上下文判断提供语法模型，并不是第二套命令执行引擎。

### 9. 诊断、审计与输出约定

安全相关能力目前分散但清晰：

- [`internal/app/doctor.go`](../internal/app/doctor.go)：静态配置与文件系统检查
- [`internal/app/audit.go`](../internal/app/audit.go)：best-effort JSONL 审计轨迹
- [`internal/util/error_codes.go`](../internal/util/error_codes.go)：稳定、脱敏的 JSON 错误封装

值得特别记住的行为：

- `doctor` 不会真的去拨数据库、代理或 SSH
- JSON 错误会映射到稳定 code，比如 `VALIDATION_FAILED`、`SSH_AUTH_FAILED`、`SQL_EXECUTION_FAILED`
- 预览、日志和 JSON 输出里都会排除秘密值

## 包职责地图

当前真正处于活跃执行路径中的包大致如下：

- [`cmd/dbx/`](../cmd/dbx)：进程入口
- [`internal/app/`](../internal/app)：命令树、REPL 处理、CLI 处理、执行编排、输出整形
- [`internal/commandlang/`](../internal/commandlang)：补全和 help 感知解析所需的词法/语法模型
- [`internal/config/`](../internal/config)：配置类型、store、诊断、audit/history/session 持久化
- [`internal/connect/`](../internal/connect)：带超时的 connector 分发
- [`internal/driver/`](../internal/driver)：MySQL 传输和查询辅助
- [`internal/template/`](../internal/template)：内置模板、分层解析、渲染、校验
- [`internal/ui/`](../internal/ui)：prompt 和补全相关 UI 类型
- [`internal/ui/editor/`](../internal/ui/editor)：buffer 和 completion edit 原语
- [`internal/util/`](../internal/util)：校验辅助、分层错误、路径辅助、JSON 错误码

另外有两个目录目前存在，但不在活跃执行路径里：

- `internal/repl/` 当前为空
- `internal/commandmeta/` 当前为空

## 当前实现观察

从现在的代码形态看，最值得记住的几点是：

- 共享命令树是这个仓库最强的架构决策。它让 REPL help、CLI help、校验和补全天然保持同步。
- 模板驱动执行贯彻得比较彻底。哪怕是 `create database` 这种内建动词，也不是在 handler 里手写 SQL，而是先变成 execution plan。
- 安全能力是分层散布的，而不是集中在一个“大安全文件”里：校验、脱敏、确认、静态诊断、审计记录各自靠近自己保护的行为。
- 有些用户命令会先归一化成内部模板命令名。例如 `show rows <table>` 实际解析的是 `peek rows` 模板，`show table <name>` 实际解析的是 `show create table` 模板。写模板时应以模板命令名为准，而不是总按最终展示给用户的字面命令写。
- 补全是代码库里相对更“高级”的一块。项目已经为了上下文感知补全引入了本地 lexer/parser 模型，但仍然没有引入 readline 风格的大框架。

## 测试覆盖形态

仓库在几个关键子系统上都已经有比较成型的自动化覆盖：

- `internal/app/*_test.go`：命令面、CLI/REPL 一致性、执行流程、补全、审计行为
- `internal/template/*_test.go`：模板加载、校验、渲染、优先级、性能
- `internal/config/*_test.go`：配置默认值、store 行为、持久化规则
- `internal/driver/*_test.go`：代理和传输行为
- `internal/commandlang/*_test.go`：lexer/parser/schema 行为

这和项目方向是匹配的：交互体验保持小而稳，但命令面和安全行为要有足够测试兜底。
