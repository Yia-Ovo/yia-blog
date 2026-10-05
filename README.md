# yia-blog

`yia-blog` 是一个基于 Go、Gin、GORM、MySQL 和 Redis 的博客后端项目，采用用户与管理员双 JWT 鉴权，并按 DDD 思路划分 API、应用、领域和基础设施层。

后端位于 `blog-server/`，React 前端位于 `blog-web/`。默认开发端口为 `8081`，当前 Router 固定使用 `/api/blog/v1`；`gin.router_prefix` 尚未接入路由，不会改变实际请求地址。前端接入范围与管理入口见 [前端接入清单](docs/frontend-readiness.md)，服务启动、只读 devcheck 与实际操作验收见 [本地联调说明](docs/local-integration.md)。

## 当前交付（2026-10-06）

公网博客：[sryia.cn](https://sryia.cn/)。已接入域名 HTTPS、四语言界面（中英日越）、每日主题、头像/文章图片、草稿恢复与 Markdown 文件导入。小说《未归之人》已发布20章，第一部仍在连载。

- [写作与 Markdown 导入](docs/editor-writing.md)
- [国际化说明](docs/frontend-i18n.md)
- [域名部署与续期](docs/domain-deployment.md)
- [Ubuntu 部署及每日备份](deploy/ubuntu/README.md)
- [小说公开章节](creative-writing/novel-01/publication.md)

最近前端验证为117项测试、类型检查与生产构建；Markdown 导入通过本地及公网桌面/手机浏览器验收。每日备份为同机快照，已做首次隔离恢复，不等于异机灾备。下方2026-09-11验收记录保留为首版历史，不能当作当前生产容量证明。

## 技术栈

| 类型 | 技术 |
| --- | --- |
| 语言 | Go 1.25.3 |
| Web | Gin |
| 前端 | React 19 + TypeScript + Vite + React Router |
| ORM | GORM |
| 数据库 | MySQL |
| 缓存 | Redis |
| 鉴权 | JWT + Redis 在线令牌/黑名单 + `token_version` |
| 配置 | YAML / Viper，可选 etcd 配置 |
| 日志 | Zap + Lumberjack |

## 核心功能

- 用户与管理员使用独立 JWT 密钥和请求头，避免身份体系混用。
- 管理员区分超级管理员与普通管理员，权限规则在领域层校验。
- `token_version` 支持改密码、改状态、改角色后的旧令牌强制失效。
- 文章支持用户作者和管理员作者两种来源，作者身份来自鉴权上下文。
- 评论写接口使用专用 DTO 和 App Command，客户端不能直接指定作者、审核状态、点赞数、IP 等服务端字段。
- 评论更新和删除同时经过领域归属校验及 `id + user_id` SQL 条件兜底。
- 已审核评论被用户编辑后会重新进入待审核状态；管理员审核通过预期正文和状态条件防止并发误放行。
- 作者可读取本人草稿；管理员可读取全部文章的管理列表和编辑详情，公开文章仍仅限已发布内容。
- 使用统一响应结构、结构化日志和手动依赖注入容器。

## 项目结构

```text
yia-blog/
├── blog-server/                # Go 后端服务
│   ├── api/
│   │   ├── dto/                # HTTP 请求/响应 DTO 与参数校验
│   │   ├── handler/            # 参数绑定、身份读取、超时和响应
│   │   └── router/             # 路由分组与鉴权中间件
│   ├── app/service/            # 应用服务、App Command、缓存编排
│   ├── common/
│   │   ├── consts/             # 状态、角色、Context Key 等常量
│   │   ├── gin/                # Gin 初始化与统一响应
│   │   └── utils/              # 日志、密码、JWT、转换工具
│   ├── config/                 # 配置结构与加载逻辑
│   ├── cmd/devcheck/            # 独立只读联调预检，不执行迁移
│   ├── domain/
│   │   ├── do/                 # 领域对象和领域查询条件
│   │   ├── repo/               # 仓储接口
│   │   └── service/            # 领域规则、资源归属与状态流转
│   ├── infra/
│   │   ├── model/              # GORM 数据库模型
│   │   └── repo/               # MySQL/Redis 仓储实现
│   ├── ioc/                    # 手动依赖注入容器
│   ├── middleware/             # 鉴权、CORS、访问日志、恢复等中间件
│   ├── go.mod
│   └── main.go
├── blog-web/                   # React 前端：阅读、写作与管理
├── docs/sql/                   # 手动数据库迁移 SQL
├── .gitignore
└── README.md
```

## DDD 请求流转

项目约定的依赖方向如下：

```text
HTTP Request
    ↓
API DTO
    ↓ Handler：绑定参数、读取鉴权身份、提取 IP/User-Agent
App Command / Query
    ↓ AppService：流程编排、调用 Domain、处理 Redis/缓存
Domain DO
    ↓ DomainService：业务规则、资源归属、状态流转
Repository Interface
    ↑
Infra Repository + Model
    ↓
MySQL / Redis
```

### 数据结构职责

| 类型 | 所属层 | 作用 |
| --- | --- | --- |
| DTO | `api/dto` | 描述 HTTP 输入输出，包含 `json/form/binding` tag |
| App Command | `app/service` | 描述应用用例需要的数据，不依赖 HTTP 或 API DTO |
| DO | `domain/do` | 表达领域状态和业务语义，不作为客户端直接输入 |
| Model | `infra/model` | 对应数据库表、字段、索引和 GORM tag |

例如创建评论时，客户端只能提交文章 ID、父评论 ID 和正文。用户 ID 来自鉴权上下文，IP 和 User-Agent 来自 HTTP 请求，审核状态和点赞数由领域规则维护。

## 分层职责

### API

- Router 负责路由分组和中间件绑定。
- Handler 负责 DTO 绑定、读取 Context 身份、设置请求超时和写响应。
- Handler 可以判断“当前 HTTP 请求是否具有有效登录身份”，但不能判断资源归属或角色业务规则。

### App

- 调用一个或多个领域服务完成用例编排。
- 负责 Redis、cache-aside、在线令牌清理等基础设施流程。
- 使用 App Command 隔离 HTTP DTO，不实现资源归属等领域不变量。

### Domain

- 负责资源归属、操作者与目标关系、角色限制、状态流转和内容规则。
- 业务规则在 HTTP、任务、测试或未来 RPC 入口调用时都必须成立。
- 不读取 Gin Context，不处理 HTTP 响应，不调用 Redis。

### Infra

- 实现 MySQL 和 Redis 仓储接口。
- GORM Model 与数据库表结构在这一层维护。
- 重要权限操作使用 SQL 条件做并发和越权兜底，但不能替代领域判断。

## 鉴权体系

| 身份 | 请求头 | 中间件 |
| --- | --- | --- |
| 用户 | `Authorization: Bearer <token>` | `DefaultUserAuth()` |
| 管理员 | `Admin-Authorization: Bearer <token>` | `DefaultAdminAuth()` |
| 超级管理员 | `Admin-Authorization: Bearer <token>` | `DefaultAdminAuth()` + `AdminSuperAuth()` |

令牌校验包含签名、Redis 在线状态、黑名单、账号状态和数据库 `token_version`。JWT 只用于证明身份和令牌版本，当前角色和账号状态以服务端数据为准。

## 配置说明

本地配置文件可能包含 JWT 密钥、MySQL 密码和 Redis 密码，因此已通过 `.gitignore` 忽略：

```text
blog-server/config/config.*.yaml
```

启动前需要在本地准备对应环境的配置文件，例如：

```text
blog-server/config/config.dev.yaml
```

环境由 `APP_ENV` 决定，未设置时默认为 `dev`。也可以通过 `-c` 显式指定配置路径，通过 `ETCD_ADDR` 或 `-r` 使用远程配置。

请勿提交包含真实密钥和密码的配置文件。已经进入 Git 历史的密钥应按泄露处理并立即轮换。

运行配置中以下值已经实际接入，时长使用 `5s`、`5m` 这样的字符串：

```yaml
server:
  shutdown_timeout: 5s
mysql:
  max_open: 20
  max_idle: 5
  conn_max_lifetime: 5m
  show_sql: false
```

以上只是配置片段，请合入本地完整配置，不要用它覆盖原文件。连接上限非正时默认 20，合法 max_idle=0 表示不保留空闲连接；连接寿命默认 5m。show_sql=false 保留慢 SQL/错误日志，SQL 参数始终不输出。Shutdown 超时后会显式关闭活动连接。

## 数据库迁移

项目不依赖 GORM `AutoMigrate`，数据库结构通过 `docs/sql/` 下的 SQL 手动维护。

当前增量脚本包括：

```text
docs/sql/20260707_add_token_version.sql
docs/sql/20260709_add_article_user_id.sql
docs/sql/20260716_add_comment_user_id.sql
docs/sql/20260908_add_comment_like.sql
```

部署对应代码前，应先备份数据库并按顺序执行尚未应用的迁移脚本。

当前点赞实现依赖 `comment_like` 表；新表及历史关系切换按 [点赞切换说明](docs/sql/20260908_comment_like_rollout.md) 执行。代码测试通过不代表目标数据库已完成迁移。

## 本地启动

### 1. 准备环境

- Go 1.25.3 或兼容版本
- MySQL
- Redis
- 本地 `config/config.dev.yaml`

### 2. 启动服务

```powershell
cd D:\develop\yia-blog\blog-server
$env:APP_ENV="dev"
go run .
```

也可以显式指定配置文件：

```powershell
go run . -c config/config.dev.yaml
```

开发环境默认访问地址：

```text
http://localhost:8081
```

健康检查：

```text
GET http://localhost:8081/ping
```

## 统一响应

接口使用统一 JSON 信封，当前 HTTP 状态码统一返回 `200`，业务结果通过 `code` 表达：

```json
{
  "code": 400,
  "msg": "Param Error",
  "err_msg": "",
  "data": null
}
```

| 字段 | 说明 |
| --- | --- |
| `code` | 业务状态码，`200` 表示成功 |
| `msg` | 标准错误或成功信息 |
| `err_msg` | 保留兼容字段，固定为空；内部诊断不返回客户端 |
| `data` | 业务数据，无数据时为 `null` |

## 前端开发

前端需要 Node.js 22.12+。在 Git Bash 执行：

```bash
cd /d/develop/yia-blog/blog-web
npm ci --ignore-scripts
npm test
npm run build
npm run dev
```

访问 `http://127.0.0.1:5173`，默认代理到后端 8081。完整页面、双登录态、CORS 来源配置及部署说明见 [前端 README](blog-web/README.md)；源码不包含样例数据或管理员账号，也不会自动执行迁移。

React 首版通过了 TypeScript 类型检查、19 项 Vitest 测试和 Vite 生产构建。2026-09-11 又完成本机真实 MySQL/Redis 的 9 组浏览器、18 组 API/数据库验收及桌面/移动端检查；既有评论迁移已应用，测试数据已清理。JWT 同秒重登录问题已用随机 jti 修复，补全量 Go 检查、两身份各 3 次真实同秒复验及旧令牌兼容验证；其他环境迁移和部署需独立验收，详见 [联调记录](docs/local-integration.md#当前记录)。

## 测试与检查

所有 Go 命令都应在 `blog-server/` 目录执行：

```powershell
cd D:\develop\yia-blog\blog-server
go test ./...
go vet ./...
go build -buildvcs=false ./...
```

运行单个包或测试：

```powershell
go test ./domain/service -run TestUpdateCommentByOwnerResetsStatus -v
```

## 开发约定

- 开始修改前先检查 `git status --short --branch`，保护现有工作区改动。
- Handler 只处理 HTTP；App 只负责编排；Domain 承载业务不变量；Repo 只负责持久化。
- 新增字段时同步检查 DTO、App Command、DO、Model、Convert 和 SQL migration。
- 新增安全相关修改时补充跨用户、状态流转、输入字段和持久化条件测试。
- 日志使用 `[API]`、`[APP]`、`[DOMAIN]`、`[MYSQL]`、`[REDIS]`、`[MIDDLEWARE]` 分层前缀。
- 不提交本地配置、密钥、日志、Go build cache、面试笔记或临时文件。
