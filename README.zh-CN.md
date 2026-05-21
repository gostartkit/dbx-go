# dbx

[English README](README.md)

`dbx` 是一个以 REPL 为核心、使用 Go 编写的 MySQL 运维 CLI。它把日常数据库操作收敛成有引导的模板化流程，REPL 与一次性 CLI 共享同一棵命令树，并且原生支持直连、SSH、SOCKS5、SOCKS5 -> SSH 四种链路，不依赖外部 SSH 命令。

`dbx` 当前不提供“让用户直接输入任意 SQL”的入口。相反，命令会收集类型化输入、校验标识符、解析可执行 operation spec、预览执行计划，然后再安全执行。

## 核心特性

- 以会话感知的 `dbx>` REPL 为主
- 交互模式与非交互模式共享命令树
- 规范命令面采用动词优先：`show`、`create`、`drop`、`use`、`exec`
- 原生支持 MySQL、SSH、SOCKS5、SOCKS5 -> SSH
- 模板解析优先级：连接级 > 全局 > 内置
- 支持 dry-run 预览、确认门槛、敏感信息脱敏、本地审计日志
- 依赖精简，项目结构保持经典 Go 布局

## 命令面

当前对外命令集合如下：

```text
connect [name]

show connections
show connection <name>
show databases
show tables
show table <name>
show columns <table>
show rows <table> [--limit n]
show users
show templates [query] [--tag value]
show context

create connection <name>
create database <name>
create user <name>

drop connection <name>
drop database <name>
drop user <name>

use database <name>

exec <operation> [--preview] [--verbose] [--validate]

doctor
audit log
help
exit
```

补充说明：

- `show table <name>` 输出该表的 `SHOW CREATE TABLE` DDL。
- `show rows <table>` 默认查看 `10` 行，最大限制为 `100`。
- `exec <operation>` 用来执行具名 operation。当前唯一 provider 是 `template`，但命令面保持 provider-neutral；当同一命令在同一层级上匹配到多个模板时，它也是显式选择 operation 的入口。

在非交互模式里，只需要在前面加上 `dbx`：

```bash
dbx show connections
dbx --connection prod show tables
dbx --connection prod exec create_database_with_user --preview
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

一个典型的引导式流程：

```text
dbx> create connection prod
dbx> connect prod
dbx(prod)> show databases
dbx(prod)> use database app_prod
dbx(prod/app_prod)> show tables
dbx(prod/app_prod)> show table users
dbx(prod/app_prod)> show rows users --limit 20
dbx(prod/app_prod)> create user analytics_ro
dbx(prod/app_prod)> audit log
```

## 非交互 CLI

创建一个直连连接：

```bash
dbx create connection dev \
  --mode direct \
  --host 127.0.0.1 \
  --port 3306 \
  --user root \
  --password-env MYSQL_DEV_PASSWORD \
  --yes
```

创建一个 `proxy-ssh` 连接并顺带测试：

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
  --ssh-private-key ~/.ssh/id_rsa \
  --test \
  --yes
```

查看保存的状态和当前上下文：

```bash
dbx show connections
dbx show connection prod
dbx --connection prod show databases
dbx --connection prod --database app_prod show tables
dbx --connection prod --database app_prod show table users
dbx --connection prod --database app_prod show columns users
dbx --connection prod --database app_prod show rows users --limit 20
dbx --connection prod show context --format json
```

执行引导式操作：

```bash
dbx --connection prod create database app_demo --yes
dbx --connection prod drop database app_demo --dry-run
dbx --connection prod --database app_prod create user analytics_ro \
  --grant readonly \
  --password-env ANALYTICS_RO_PASSWORD \
  --yes
dbx --connection prod --database app_prod drop user analytics_ro --yes
```

显式使用具名 operation：

```bash
dbx --connection prod show templates
dbx --connection prod show templates database --tag tenant
dbx --connection prod exec create_database_with_user --validate
dbx --connection prod exec create_database_with_user \
  --input database=greenhn_prod \
  --input user_host=% \
  --input password=super-secret \
  --preview
dbx --connection prod create database greenhn_prod \
  --template create_database_with_user \
  --input user_host=% \
  --input password=super-secret \
  --yes
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

目前只支持 SOCKS5 代理。各模式校验保持严格：

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

密码来源可以是内联、环境变量或运行时提示：

- 数据库密码：`password`、`password_env`、`password_prompt`
- SSH 认证：`ssh.private_key`、`ssh.password_env`、`ssh.password`

`doctor` 会对不安全的内联秘密值和缺失的密码环境变量给出告警，但不会自动替你改配置。

## 模板系统

模板是安全的运维工作流，不是通用脚本运行时。

查找优先级：

```text
连接级模板
> 全局模板
> 内置模板
```

模板目录：

```text
~/.config/dbx/templates/
~/.config/dbx/{connection}/templates/
```

当前支持的输入类型：

- `string`
- `secret`
- `select`
- `confirm`
- `identifier`
- `int`

模板相关事实：

- CLI 里通过 `--input key=value` 传入的 key 必须与模板输入名完全一致。
- `secret` 输入不会出现在预览、日志、审计记录或 JSON 输出中。
- 如果同一层级里有多个模板匹配到同一命令，REPL 会让你选一个，CLI 则需要显式传 `--template <name>` 或使用 `exec <name>`。

内置变量包括：

```text
{{database}}
{{connection.name}}
{{connection.host}}
{{connection.user}}
```

仓库里自带了一个示例模板：[examples/templates/create_database_with_user.json](examples/templates/create_database_with_user.json)。它匹配 `create database`，并额外引入 `user_host` 与 `password` 两个输入。

如果你想系统地从零学习模板设计、优先级、输入类型、连接级覆盖、歧义处理和安全约束，可以继续看 [TEMPLATE_STARTKIT.zh-CN.md](TEMPLATE_STARTKIT.zh-CN.md)。英文版见 [TEMPLATE_STARTKIT.md](TEMPLATE_STARTKIT.md)。

要在本地试用它，可以先复制到配置目录：

```bash
mkdir -p ~/.config/dbx/templates
cp examples/templates/create_database_with_user.json ~/.config/dbx/templates/
```

然后执行校验或预览：

```bash
dbx --connection prod exec create_database_with_user --validate
dbx --connection prod create database app_demo \
  --template create_database_with_user \
  --input user_host=% \
  --input password=super-secret \
  --preview
```

## 配置目录

所有用户状态都位于 `~/.config/dbx/`：

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

关键路径：

- 连接配置：`~/.config/dbx/{connection}/config.json`
- 连接级模板：`~/.config/dbx/{connection}/templates/`
- 全局模板：`~/.config/dbx/templates/`
- session 文件：`~/.config/dbx/session.json`
- history 文件：`~/.config/dbx/history`
- 审计日志：`~/.config/dbx/logs/audit.jsonl`

`session.json` 会持久化当前选中的连接和数据库。命令历史也会本地持久化，并裁剪为最近 `1000` 条。

## 诊断与安全

- `doctor` 是静态检查。它会检查配置结构、密码来源、代理 URL、SSH 认证方式、密钥文件权限和 `known_hosts`，但不会真的去拨通网络链路。
- `create connection ... --test` 会先保存配置，再做一次实时连通性检查；如果检查失败，会给出警告，但不会删除已保存配置。
- 变更型命令在 REPL 中需要确认，在非交互模式下需要传 `--yes`；如果该命令处于 `--dry-run` 或 `--preview` 状态，则会跳过真实执行。
- 密码、代理密码、模板秘密输入都会在用户可见输出和审计日志中脱敏。
- `audit log` 读取本地 JSONL 审计文件；即使追加审计日志失败，也不会让原始用户命令跟着失败。

## 架构

`dbx` 刻意保持小而清晰，按职责拆分：

- [cmd/dbx/main.go](cmd/dbx/main.go)：程序入口、信号感知退出、CLI 根命令
- [internal/app/](internal/app)：共享命令树、REPL 处理、一次性 CLI 流程、结果输出
- [internal/repl/](internal/repl)：最小化 REPL 循环
- [internal/config/](internal/config)：配置加载、session、history、audit log、超时默认值
- [internal/connect/](internal/connect)：带超时的连接辅助
- [internal/driver/](internal/driver)：MySQL、SSH、SOCKS5 传输实现
- [internal/template/](internal/template)：模板解析与渲染
- [internal/ui/](internal/ui)：轻量 prompt、history、补全
- [internal/util/](internal/util)：校验、路径、分层错误、脱敏辅助

## 项目布局

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

- Go `1.25+`
- 可以通过支持的链路之一访问 MySQL

常用命令：

```bash
make fmt
make test
make build
make check
```

发布打包脚本位于 [scripts/release.sh](scripts/release.sh)。安装辅助脚本 [scripts/install.sh](scripts/install.sh) 采用当前发布产物命名规则，但在真正发布前仍需要先把其中的 `REPO` 值配置成最终 GitHub 仓库地址。

## 贡献

开发与评审约束见 [CONTRIBUTING.zh-CN.md](CONTRIBUTING.zh-CN.md)。
