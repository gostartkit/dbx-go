# dbx 模板系统 Startkit 教程

[English README](../README.md) | [中文 README](../README.zh-CN.md) | [English Tutorial](TEMPLATE_STARTKIT.md)

这份教程面向两类人：

- 想第一次给 `dbx` 写模板的人
- 已经会写一点 JSON，但想把模板写得更稳、更容易维护的人

目标不是只解释字段含义，而是帮你建立一套真正可落地的模板工作流：

1. 知道模板在 `dbx` 里是怎么被找到和执行的
2. 知道什么时候该写全局模板，什么时候该写连接级模板
3. 知道如何安全处理密码、授权 SQL、环境变量和 dry-run
4. 能从零写出一个可以在 REPL 和 CLI 中稳定工作的模板

## 1. 先建立正确心智模型

在 `dbx` 里，模板不是“脚本引擎”，而是“受约束的运维工作流定义”。

你可以把一个模板理解成：

- 一组元信息：名字、分类、标签、描述
- 一个匹配条件：这个模板服务哪个命令、哪个驱动
- 一组类型化输入：字符串、秘密值、选择项、确认项、整数等
- 一组 SQL 动作：由 `dbx` 渲染并执行

`dbx` 的核心思路是：

- 用户不要直接提供任意 SQL
- 用户提供的是参数和选择
- `dbx` 负责校验、渲染、预览、脱敏和执行

这意味着模板系统非常适合下面这类事情：

- 创建数据库
- 删除数据库
- 创建用户并授权
- 连接级的只读巡检动作
- 按环境差异覆盖默认行为

它不适合下面这些东西：

- 循环
- 条件分支
- 远程拉模板
- 插件系统
- 任意脚本执行

当前实现里，模板只支持一种动作类型：`sql`。

## 2. 模板放在哪里

模板有 3 层优先级：

```text
连接级模板
> 全局模板
> 内置模板
```

目录如下：

```text
~/.config/dbx/templates/
~/.config/dbx/{connection}/templates/
```

对应理解：

- 全局模板：给所有连接复用
- 连接级模板：只对某个连接生效，例如 `prod`
- 内置模板：代码里自带的兜底模板

解析规则是“先看高优先级层，再看低优先级层”：

- 如果连接级命中了，就不会继续看全局或内置
- 如果连接级没命中，再看全局
- 如果全局也没有，再落回内置

同名模板的覆盖规则也遵循这个优先级。比如：

- 全局有 `shared_workflow`
- `prod` 连接目录里也有 `shared_workflow`

那么 `prod` 上看到的 resolved 模板会是连接级版本。

## 3. 一个最小可用模板

先看最小例子：

```json
{
  "name": "drop_database_guarded",
  "match": {
    "command": "drop database",
    "driver": "mysql"
  },
  "actions": [
    {
      "type": "sql",
      "description": "Drop database `{{database}}`",
      "sql": "DROP DATABASE IF EXISTS `{{database}}`"
    }
  ]
}
```

这个模板已经可以工作，因为它满足了最小要求：

- 有 `name`
- 有 `match.command`
- 有 `actions`
- 每个 action 都是 `sql`

你把它放到：

```text
~/.config/dbx/templates/drop_database_guarded.json
```

然后运行：

```bash
dbx --connection prod drop database app_demo --template drop_database_guarded --dry-run
```

`dbx` 会：

1. 找到模板
2. 把 `database=app_demo` 注入模板变量
3. 渲染 SQL
4. 输出 preview / dry-run 结果

## 4. 模板 JSON 全字段说明

完整模板结构长这样：

```json
{
  "version": 1,
  "name": "create_database_with_user",
  "category": "database",
  "tags": ["grant", "tenant"],
  "description": "Create a database, create a same-name MySQL user, and grant privileges.",
  "transaction": true,
  "match": {
    "command": "create database",
    "driver": "mysql"
  },
  "inputs": [
    {
      "name": "database",
      "type": "string",
      "prompt": "Database name"
    }
  ],
  "actions": [
    {
      "type": "sql",
      "description": "Create database `{{database}}`",
      "sql": "CREATE DATABASE IF NOT EXISTS `{{database}}`"
    }
  ]
}
```

字段说明如下。

### 4.1 `version`

- 当前 schema 版本是 `1`
- 不写时会默认补成 `1`
- 写成别的值会报 `unsupported version`

建议：新模板统一显式写 `1`，这样更清楚。

### 4.2 `name`

- 模板名
- 必填
- 用于 `exec <name>`、`show templates` 展示、歧义场景的显式选择

建议命名方式：

- 用能表达业务意图的名字
- 不要只叫 `prod`、`workflow1` 这种泛名

推荐风格：

```text
create_database_with_user
readonly_user
drop_database_guarded
prod_app_database
```

### 4.3 `category`

- 可选
- 不写时默认是 `custom`

它主要用于：

- `show templates` 输出整理
- 人类阅读和筛选

常见写法：

```text
database
user
inspection
grant
custom
```

### 4.4 `tags`

- 可选
- 用于检索与筛选
- `show templates --tag value` 会用到它

例如：

```json
"tags": ["tenant", "grant", "readonly"]
```

### 4.5 `description`

- 可选
- 用于模板目录展示和歧义提示

当同一命令在同一层命中多个模板时，CLI 错误提示里会把 `description` 一起带出来，所以写得清楚很有价值。

### 4.6 `transaction`

- 可选
- `true` 表示让 `dbx` 以事务方式执行整组动作

适合：

- 多条授权 SQL
- 需要“要么都做，要么都不做”的工作流

但要记住：

- 这只是 `dbx` 层面的事务请求
- 具体 DDL 在 MySQL 里的事务语义，仍然受 MySQL 自身规则约束

建议：只有在你明确需要时才打开。

### 4.7 `match`

结构：

```json
"match": {
  "command": "create database",
  "driver": "mysql"
}
```

说明：

- `command` 必填
- `driver` 可选，但当前项目实际只支持 `mysql`

这是模板最关键的路由条件。模板会不会被自动选中，首先看这里。

## 5. `match.command` 怎么写才对

这是最容易踩坑的地方。

并不是所有“用户看到的命令文本”都直接等于模板匹配命令。当前代码里的模板挂载点如下：

| 用户命令 | 模板 `match.command` |
| --- | --- |
| `create database` | `create database` |
| `show databases` | `show databases` |
| `drop database` | `drop database` |
| `create user` | `create user` |
| `show users` | `show users` |
| `drop user` | `drop user` |
| `show tables` | `show tables` |
| `show columns <table>` | `show columns` |
| `show table <name>` | `show create table` |
| `show rows <table>` | `peek rows` |

注意最后两项：

- 想覆盖 `show table`，你要匹配的是 `show create table`
- 想覆盖 `show rows`，你要匹配的是 `peek rows`

如果你把模板写成：

```json
"match": { "command": "show rows", "driver": "mysql" }
```

当前实现里它不会自动接管 `show rows <table>`。

## 6. 输入系统怎么设计

模板输入定义在 `inputs` 里，每个输入至少有一个名字：

```json
{
  "name": "password",
  "type": "secret",
  "prompt": "New user password"
}
```

当前支持 6 种输入类型：

- `string`
- `secret`
- `select`
- `confirm`
- `identifier`
- `int`

### 6.1 通用字段

单个 input 支持这些字段：

- `name`
- `type`
- `prompt`
- `description`
- `required`
- `secret`
- `default`
- `options`
- `choices`
- `identifier`

其中几个行为规则非常重要：

- `required` 不写时：如果没有 `default`，默认必填；如果有 `default`，默认可缺省
- `prompt` 不写时，会退回到 `description`，再退回到 `name`
- `options` 和 `choices` 都可用于 `select`
- `secret: true` 可以隐式把输入当成 `secret`
- `identifier: true` 可以隐式把输入当成 `identifier`

### 6.2 `string`

最普通的字符串输入：

```json
{
  "name": "user_host",
  "type": "string",
  "prompt": "User host",
  "default": "%"
}
```

适合：

- host
- charset 名
- comment
- 业务标签

### 6.3 `secret`

秘密值输入：

```json
{
  "name": "password",
  "type": "secret",
  "prompt": "Password"
}
```

特点：

- REPL 下会用隐藏输入方式读取
- preview / verbose / JSON 输出中会被脱敏成 `***`
- 审计日志不会记录明文

最佳实践：

- CLI 下不要直接写 `--input password=super-secret`
- 尽量写成 `--input password-env=APP_PASSWORD`

这样可以避免 shell history 泄露。

### 6.4 `select`

枚举输入：

```json
{
  "name": "charset",
  "type": "select",
  "prompt": "Charset",
  "default": "utf8mb4",
  "options": ["utf8mb4", "utf8"]
}
```

特点：

- REPL 下会给出选项
- CLI 下如果你传了不在集合里的值，会直接校验失败

适合：

- charset
- collation
- privilege mode
- tenant type

### 6.5 `confirm`

布尔确认输入：

```json
{
  "name": "really_drop",
  "type": "confirm",
  "prompt": "Drop this database?",
  "default": false
}
```

渲染后会得到字符串：

- `true`
- `false`

适合做“操作前再确认一次”的模板输入，但要注意：模板系统本身没有条件执行，所以它更适合渲染说明文字，而不是在模板里做分支逻辑。

### 6.6 `identifier`

更严格的标识符输入：

```json
{
  "name": "role_name",
  "type": "identifier",
  "prompt": "Role name"
}
```

当前规则是：

```text
[a-zA-Z_][a-zA-Z0-9_]*
```

这比数据库名和 MySQL 用户名更严格。

非常重要的现实建议：

- 如果你的值需要允许 `-`，不要用 `identifier`
- 例如数据库名当前允许 `greenhn-prod` 这种带横线的名字
- 但 `identifier` 会拒绝这种值

所以：

- 内部变量名、角色名、严格标识符，用 `identifier`
- 数据库名、用户名这类有自己业务规则的值，不要轻易套 `identifier`

### 6.7 `int`

整数输入：

```json
{
  "name": "limit",
  "type": "int",
  "prompt": "Limit",
  "default": 20
}
```

REPL 下会走整数输入逻辑，CLI 下也会做整数校验。

## 7. 动作系统怎么写

`actions` 是模板真正执行的内容：

```json
{
  "type": "sql",
  "description": "Grant SELECT on `{{database}}`.*",
  "sql": "GRANT SELECT ON `{{database}}`.* TO '{{username}}'@'{{user_host}}'"
}
```

当前规则：

- `type` 只能是 `sql`
- `sql` 不能为空
- `description` 会显示在 preview / 结果输出里

建议：

- `description` 写成人类可读的动作说明
- `sql` 写成最终要执行的 SQL 模板
- 如果一个工作流有 3 步，就拆成 3 个 action，不要把所有 SQL 塞一条里

## 8. 模板变量如何渲染

模板使用 Mustache 风格占位符：

```text
{{database}}
{{username}}
{{connection.name}}
{{connection.host}}
```

当前内置 connection 变量包括：

- `{{connection.name}}`
- `{{connection.driver}}`
- `{{connection.mode}}`
- `{{connection.host}}`
- `{{connection.port}}`
- `{{connection.user}}`

### 8.1 一个关键安全行为：SQL 字符串会自动转义

`dbx` 在渲染 SQL 时，会对普通字符串值做 MySQL 字符串转义。

例如：

- 输入 `pa'ss`
- 在 SQL 里会被渲染成 `pa''ss`

这也是为什么模板里应该写：

```sql
IDENTIFIED BY '{{password}}'
```

而不是自己再手动转义一层。

### 8.2 另一个关键行为：`description` 用原始值，`sql` 用转义值

例如：

- `description`: `Create user {{username}}`
- `sql`: `CREATE USER '{{username}}'`

这样用户看到的说明会更自然，而真正执行的 SQL 仍然是安全的。

### 8.3 `_sql` 后缀变量会原样插入

这是高级用法，尤其重要。

如果变量名以 `_sql` 结尾，当前实现会在 SQL 渲染阶段“跳过转义”，直接原样插入。

例如内置 `create user` 会使用：

- `grant_sql`

这样的变量适合放“由 `dbx` 自己拼出来的 SQL 片段”，例如：

```text
GRANT SELECT ON `app_prod`.* TO 'ro'@'%'
```

不要把用户随便输入的内容塞到 `_sql` 变量里。

最佳实践：

- `_sql` 变量只能装“程序生成的、可信的 SQL 子句”
- 绝不要把原始用户输入直接拼进 `_sql`

## 9. CLI 输入怎么喂给模板

CLI 里主要有两种方式。

### 9.1 直接传值

```bash
dbx --connection prod exec create_database_with_user \
  --input database=app_demo \
  --input user_host=% \
  --input password=super-secret \
  --preview
```

### 9.2 从环境变量读取

这是更推荐的方式：

```bash
export APP_PASSWORD='super-secret'

dbx --connection prod exec create_database_with_user \
  --input database=app_demo \
  --input user_host=% \
  --input password-env=APP_PASSWORD \
  --preview
```

规则是：

- 如果 key 以 `-env` 结尾
- `dbx` 会把它去掉后缀
- 然后把 value 当成环境变量名去读取

所以：

- `password-env=APP_PASSWORD`
- 最终会变成模板输入 `password`

这套规则同样适合别的 secret 输入。

## 10. REPL 与 CLI 的模板输入差异

REPL 模式下，如果模板输入没给全：

- `string` 会逐个提问
- `secret` 会隐藏输入
- `select` 会列出选项
- `confirm` 会走 y/n
- `int` 会按整数读取

CLI 模式下：

- 没给的值不会交互补齐
- 如果模板要求必填，就直接报错
- 有默认值的输入会自动补默认值

所以一般工作流建议是：

- REPL：适合探索式执行
- CLI：适合脚本化、显式传参、可审计

## 11. 先学会 4 个最常用命令

### 11.1 列出模板

```bash
dbx --connection prod show templates
dbx --connection prod show templates database
dbx --connection prod show templates --tag tenant
```

你会看到类似：

```text
Templates:
name                       scope       category   command
create_database_with_user  global      database   create database  [grant,tenant]
prod_app_database          connection  custom     create database
```

### 11.2 校验模板

```bash
dbx --connection prod exec create_database_with_user --validate
```

它会检查：

- 模板结构是否合法
- 输入类型是否支持
- action 类型是否合法

### 11.3 只预览，不执行

```bash
dbx --connection prod exec create_database_with_user \
  --input database=app_demo \
  --input user_host=% \
  --input password-env=APP_PASSWORD \
  --preview \
  --verbose
```

这一步最适合：

- 看动作列表
- 看脱敏后的 SQL
- 确认模板变量都被正确填上了

### 11.4 通过业务命令执行模板

如果某个业务命令支持 `--template`，你可以直接这样走：

```bash
dbx --connection prod create database app_demo \
  --template create_database_with_user \
  --input user_host=% \
  --input password-env=APP_PASSWORD \
  --yes
```

当前显式支持 `--template` 的主要命令有：

- `show databases`
- `create database`
- `drop database`
- `create user`
- `drop user`

而像这些命令当前没有 `--template` 入口：

- `show tables`
- `show columns`
- `show table`
- `show rows`
- `show users`

对于这些命令，如果你要用具名模板：

- 保持同一层里只命中一个模板
- 或者直接用 `exec <name>`

## 12. 实战一：从仓库示例模板起步

仓库自带了这个例子：

- [examples/templates/create_database_with_user.json](../examples/templates/create_database_with_user.json)

它做 3 件事：

1. 创建数据库
2. 创建同名 MySQL 用户
3. 给这个用户授予该库权限

先安装到全局模板目录：

```bash
mkdir -p ~/.config/dbx/templates
cp examples/templates/create_database_with_user.json ~/.config/dbx/templates/
```

然后先校验：

```bash
dbx --connection prod exec create_database_with_user --validate
```

再做预览：

```bash
export APP_PASSWORD='super-secret'

dbx --connection prod exec create_database_with_user \
  --input database=app_demo \
  --input user_host=% \
  --input password-env=APP_PASSWORD \
  --preview \
  --verbose
```

最后通过业务命令执行：

```bash
dbx --connection prod create database app_demo \
  --template create_database_with_user \
  --input user_host=% \
  --input password-env=APP_PASSWORD \
  --yes
```

这个流程就是最推荐的 Startkit 节奏：

1. 先 `show templates`
2. 再 `exec --validate`
3. 再 `exec --preview --verbose`
4. 最后再挂回正式命令执行

## 13. 实战二：给生产连接写一个连接级 drop 模板

需求：

- 生产库删除数据库前，统一走一套更明确的说明文案
- 只影响 `prod`
- 不影响别的连接

创建目录：

```bash
mkdir -p ~/.config/dbx/prod/templates
```

创建文件：

```json
{
  "version": 1,
  "name": "drop_database_guarded",
  "category": "database",
  "tags": ["danger", "prod"],
  "description": "Production database drop workflow with explicit labeling.",
  "match": {
    "command": "drop database",
    "driver": "mysql"
  },
  "actions": [
    {
      "type": "sql",
      "description": "Drop production database `{{database}}` on {{connection.name}}",
      "sql": "DROP DATABASE IF EXISTS `{{database}}`"
    }
  ]
}
```

保存到：

```text
~/.config/dbx/prod/templates/drop_database_guarded.json
```

执行：

```bash
dbx --connection prod show templates --tag prod
dbx --connection prod drop database app_demo --template drop_database_guarded --dry-run
```

这时你得到的是：

- 只在 `prod` 生效
- 同命令下优先于全局和内置模板
- 对其他连接完全无影响

这就是连接级模板最常见的落地方式。

## 14. 实战三：做一个只读用户模板

下面这个例子适合做租户库只读账号发放。

```json
{
  "version": 1,
  "name": "readonly_user",
  "category": "user",
  "tags": ["readonly", "grant"],
  "description": "Create a readonly MySQL user for the selected database.",
  "transaction": true,
  "match": {
    "command": "create user",
    "driver": "mysql"
  },
  "inputs": [
    {
      "name": "username",
      "type": "string",
      "prompt": "Username"
    },
    {
      "name": "user_host",
      "type": "string",
      "prompt": "User host",
      "default": "%"
    },
    {
      "name": "password",
      "type": "secret",
      "prompt": "Password"
    }
  ],
  "actions": [
    {
      "type": "sql",
      "description": "Create MySQL user '{{username}}'@'{{user_host}}'",
      "sql": "CREATE USER '{{username}}'@'{{user_host}}' IDENTIFIED BY '{{password}}'"
    },
    {
      "type": "sql",
      "description": "Grant SELECT on `{{database}}`.* to '{{username}}'@'{{user_host}}'",
      "sql": "GRANT SELECT ON `{{database}}`.* TO '{{username}}'@'{{user_host}}'"
    }
  ]
}
```

建议执行方式：

```bash
export RO_PASSWORD='replace-me'

dbx --connection prod --database app_prod exec readonly_user \
  --input username=analytics_ro \
  --input user_host=% \
  --input password-env=RO_PASSWORD \
  --preview \
  --verbose
```

确认 SQL 和输入摘要都对，再正式执行。

## 15. 什么时候选全局模板，什么时候选连接级模板

一个很好用的判断法：

用全局模板，当：

- 逻辑想在所有环境复用
- 差异只来自变量，不来自执行策略
- 你想沉淀成团队默认工作流

用连接级模板，当：

- 只想影响某个连接，例如 `prod`
- 生产和开发要走不同流程
- 需要替换掉全局模板的行为
- 需要更强的提示语或保护性说明

简单经验：

- “规则通用”用全局
- “环境特殊”用连接级

## 16. 歧义是怎么发生的

如果同一层里有多个模板同时匹配同一个命令，例如：

- 全局目录里有两个都匹配 `create database`

那么：

- REPL 会提示你选一个
- CLI 会报歧义错误

CLI 错误会提示你：

- 显式用 `exec <name>`
- 或在支持的命令上加 `--template <name>`

团队约定建议：

- 默认每个“命令 x 层级”只保留一个自动匹配模板
- 如果确实要做多套变体，就把它们当成“具名工作流”，主要靠 `exec <name>` 来跑

## 17. 脱敏与安全边界

模板系统当前已经帮你做了这些事情：

- `secret` 输入在 preview / JSON / verbose SQL 里会脱敏
- 审计日志不会记录 secret 明文
- 普通字符串会做 MySQL 字符串转义

但你仍然需要自己守住这些边界：

- 不要把秘密直接写进模板 JSON
- 不要把用户原始输入拼到 `_sql` 变量里
- CLI 下尽量用 `*-env` 方式传 secret
- 不要为了省事把生产专用逻辑写成全局模板

## 18. 当前实现里最值得记住的 7 个坑

### 坑 1：`identifier` 比数据库名规则更严格

如果你要支持 `greenhn-prod` 这种值，不要用 `identifier`。

### 坑 2：`show table` 的匹配命令不是 `show table`

要写成：

```json
"command": "show create table"
```

### 坑 3：`show rows` 的匹配命令不是 `show rows`

要写成：

```json
"command": "peek rows"
```

### 坑 4：不是所有命令都支持 `--template`

需要先看命令实现，不要默认所有 `show` 命令都能手选模板。

### 坑 5：CLI 不会替你交互补齐缺失输入

脚本里要把必填输入显式传全。

### 坑 6：`secret` 会脱敏，但 shell history 不会

所以生产场景优先用：

```text
--input password-env=ENV_NAME
```

### 坑 7：模板不是脚本语言

不要试图往里塞：

- if/else
- loop
- 远程 include
- shell

如果你开始需要这些东西，通常说明应该把流程上移到应用层，而不是继续堆模板。

## 19. 推荐的模板开发工作流

每次写新模板，建议固定走这套流程：

1. 先确定它应该挂在哪个 `match.command`
2. 先决定它应该放全局还是连接级
3. 先写最小版本，只保留 1 个 action
4. 跑 `show templates` 确认它被发现
5. 跑 `exec <name> --validate`
6. 跑 `exec <name> --preview --verbose`
7. 再补充额外 action、tags、description、transaction
8. 最后再挂回正式业务命令执行

## 20. 一个可直接复用的 Startkit 清单

写模板前，逐项确认：

- 这个流程真的适合模板，而不是适合应用层逻辑
- `match.command` 写的是当前实现真正使用的挂载点
- 模板该放全局还是连接级已经想清楚
- `name` 足够清晰
- `description` 足够让人一眼看懂
- 需要隐藏的值都用 `secret`
- 不该太严格的值没有误用 `identifier`
- 需要枚举约束的字段用了 `select`
- `_sql` 变量只装程序生成的 SQL 片段
- 先做 `--validate`
- 再做 `--preview --verbose`
- 正式执行时用 `--yes`

## 21. 结论

如果只记住一句话，那就是：

`dbx` 模板系统的最佳实践，不是“把 SQL 参数化”，而是“把常见数据库操作收敛成可校验、可预览、可覆盖、可脱敏的工作流”。

从今天开始写模板时，优先按下面这个顺序思考：

1. 我要挂到哪个命令
2. 我要放全局还是连接级
3. 哪些输入应该是 `secret` / `select` / `int`
4. 有没有歧义风险
5. 是否能先 preview 再执行

这样写出来的模板，通常会比“临时拼一段 SQL”稳得多，也更适合团队长期维护。
