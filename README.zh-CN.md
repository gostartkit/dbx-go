# dbx

[English README](README.md)

`dbx` 是一个以 REPL 为核心的 MySQL 运维 CLI。它把常见数据库操作收敛成有引导的模板化流程，支持直连、SSH、SOCKS5 代理、SOCKS5 -> SSH 四种链路，也保留了 `run template` 和 `run sql` 这类明确的一次性入口，方便脚本和自动化使用。

## 核心特性

- 以交互式 `dbx>` 会话为主
- 规范命令面采用动词优先分组：`show`、`create`、`drop`、`run`、`use`
- REPL 与非交互 CLI 共享同一棵命令树
- 原生支持 MySQL、SSH、SOCKS5、SOCKS5 -> SSH
- 模板查找优先级：连接级 > 全局 > 内置
- 支持 dry-run、执行前确认、敏感信息脱敏、本地审计日志
- 依赖精简，项目结构保持经典 Go 布局

## 规范命令

新文档、脚本和示例都应优先使用下面这套规范写法：

```text
connect [name]

show connections
show connection <name>
show databases
show tables
show table <name>
show columns <table>
show rows <table> [--limit n]
show templates [query] [--tag value]
show context

create connection <name>
create database <name>
create user <name>

drop connection <name>
drop database <name>
drop user <name>

use database <name>

run template <name> [--preview] [--verbose] [--validate]
run sql <sql-or-file>

doctor
audit log
help
exit
```

交互模式与脚本模式的命令面一致；写脚本时只需要在前面加上 `dbx`：

```bash
dbx show connections
dbx --connection prod show tables
dbx --connection prod run template create_database_with_user --preview
```

## 快速开始

构建：

```bash
make build
```

启动 REPL：

```bash
go run ./cmd/dbx
```

一个典型交互流程：

```text
dbx> create connection prod
dbx> connect prod
dbx(prod)> show databases
dbx(prod)> use database app_prod
dbx(prod/app_prod)> show tables
dbx(prod/app_prod)> run template create_database_with_user --preview
dbx(prod/app_prod)> run sql @schema.sql
```

默认推荐路径仍然是引导式操作；`run sql` 是显式提供的直连 SQL 入口。

## 非交互示例

创建一个直连连接：

```bash
dbx create connection dev \
  --mode direct \
  --host 127.0.0.1 \
  --port 3306 \
  --user root \
  --password-env MYSQL_DEV_PASSWORD
```

创建一个 `proxy-ssh` 连接：

```bash
dbx create connection prod-proxy \
  --mode proxy-ssh \
  --host 10.0.1.20 \
  --port 3306 \
  --user root \
  --password-env MYSQL_PROD_PASSWORD \
  --proxy-url socks5://proxy_user:proxy_password@127.0.0.1:1080 \
  --ssh-host bastion.example.com \
  --ssh-port 22 \
  --ssh-user ubuntu \
  --ssh-private-key ~/.ssh/id_rsa
```

查看配置和上下文：

```bash
dbx show connections
dbx --connection prod show connection prod
dbx --connection prod show databases
dbx --connection prod --database app_prod show tables
dbx --connection prod --database app_prod show table users
dbx --connection prod --database app_prod show columns users
dbx --connection prod --database app_prod show rows users --limit 20
dbx --connection prod --database app_prod show context --format json
```

执行引导式工作流：

```bash
dbx --connection prod create database app_demo --yes
dbx --connection prod drop database app_demo --dry-run
dbx --connection prod --database app_prod create user analytics_ro --yes
dbx --connection prod show templates
dbx --connection prod show templates --tag tenant
dbx --connection prod run template create_database_with_user \
  --input database=greenhn_prod \
  --input user_host=% \
  --input password-env=GREENHN_PASSWORD \
  --preview
```

直接执行 SQL：

```bash
dbx --connection prod run sql "SELECT 1"
dbx --connection prod run sql @schema.sql
dbx --connection prod --database app_prod run sql migrations/bootstrap.sql
```

诊断与审计：

```bash
dbx --connection prod doctor
dbx audit log
dbx audit log --format json
```

## 连接模式

支持的链路：

```text
direct    -> db
ssh       -> ssh -> db
proxy     -> proxy -> db
proxy-ssh -> proxy -> ssh -> db
```

目前只支持 SOCKS5 代理。各模式的校验规则保持严格：

- `direct` 不能带代理和 SSH 配置
- `ssh` 必须带 SSH 配置，且不能带代理配置
- `proxy` 必须带代理配置，且不能带 SSH 配置
- `proxy-ssh` 必须同时带代理和 SSH 配置

配置示例：

```json
{
  "version": 1,
  "name": "prod-proxy",
  "driver": "mysql",
  "mode": "proxy-ssh",
  "host": "10.0.1.20",
  "port": 3306,
  "user": "root",
  "password_env": "MYSQL_PROD_PASSWORD",
  "proxy": {
    "url": "socks5://proxy_user:proxy_password@127.0.0.1:1080"
  },
  "ssh": {
    "host": "bastion.example.com",
    "port": 22,
    "user": "ubuntu",
    "private_key": "~/.ssh/id_rsa"
  },
  "timeout": {
    "connect_seconds": 10,
    "query_seconds": 30
  }
}
```

## 模板系统

模板是安全的运维工作流，不是通用脚本语言。

查找优先级：

```text
连接级模板
> 全局模板
> 内置模板
```

目录位置：

```text
~/.config/dbx/templates/
~/.config/dbx/{connection}/templates/
```

当前模板输入支持 `string`、`secret`、`select`、`confirm`、`identifier`、`int` 等类型。`secret` 类型的值不会出现在预览、日志或 JSON 输出中。

示例模板文件：[`examples/templates/create_database_with_user.json`](examples/templates/create_database_with_user.json)

这个示例模板会：

- 创建数据库
- 创建同名 MySQL 用户
- 为该数据库授权

它当前使用的输入项包括：

- `database`
- `charset`
- `collation`
- `user_host`
- `password`

常用模板命令：

```bash
dbx --connection prod show templates
dbx --connection prod show templates --tag grant
dbx --connection prod run template create_database_with_user --validate
dbx --connection prod run template create_database_with_user --preview
dbx --connection prod run template create_database_with_user --verbose --yes
```

## 配置目录

所有用户状态都保存在 `~/.config/dbx/` 下：

```text
~/.config/dbx/
  history
  logs/
    audit.jsonl
  session.json
  templates/

  dev/
    config.json
    templates/

  prod/
    config.json
    templates/
```

连接配置文件路径是 `~/.config/dbx/{connection}/config.json`。

## 诊断与安全

- `doctor` 是静态检查，不会真的去拨通代理、SSH 或 MySQL。
- `create connection ... --test` 会在保存前做一次实时连通性验证；如果验证失败，配置仍会保存，但会给出警告，方便后续继续修复。
- 变更型命令在 REPL 中需要确认，在非交互模式下需要显式传 `--yes`；如果开启了 `--dry-run` 或 `--preview`，则不会真正执行。
- 密码、代理密码、模板里的秘密输入都会在输出和审计日志里被脱敏。

## 架构与目录

主要目录职责：

- [cmd/dbx/main.go](cmd/dbx/main.go)：程序入口、信号处理、CLI 根命令
- [internal/app/](internal/app)：共享命令树、REPL 处理、一次性 CLI 流程、结果输出
- [internal/repl/](internal/repl)：最小化 REPL 循环
- [internal/config/](internal/config)：配置、session、history、超时默认值
- [internal/connect/](internal/connect)：连接层超时应用
- [internal/driver/](internal/driver)：MySQL 与 SSH 传输实现
- [internal/template/](internal/template)：模板解析与渲染
- [internal/ui/](internal/ui)：轻量交互提示
- [internal/util/](internal/util)：校验、路径、分层错误、输出辅助

项目布局：

```text
dbx/
├── cmd/
│   └── dbx/
│       └── main.go
├── internal/
│   ├── app/
│   ├── config/
│   ├── connect/
│   ├── driver/
│   ├── repl/
│   ├── template/
│   ├── ui/
│   └── util/
├── examples/
├── AGENTS.md
├── CONTRIBUTING.md
├── LICENSE
├── Makefile
├── README.md
├── README.zh-CN.md
└── go.mod
```

## 开发

要求：

- Go 1.25+
- 能通过支持的链路之一访问 MySQL

常用命令：

```bash
make fmt
make test
make build
make check
```

发布脚本在 [scripts/release.sh](scripts/release.sh)，安装辅助脚本在 [scripts/install.sh](scripts/install.sh)。

贡献和评审约束见 [CONTRIBUTING.md](CONTRIBUTING.md)。
