# Yia 博客前端

React 19 + TypeScript + Vite + React Router，使用当前 Go 后端的 `/api/blog/v1` 接口。首版包括文章阅读与分类筛选、注册登录、本人草稿编辑、管理员文章管理、超管分类维护和评论审核。

## 安装与运行

需要 Node.js 22.12+（当前开发环境为 24.18）。在 Git Bash 执行：

```bash
cd /d/develop/yia-blog/blog-web
npm ci --ignore-scripts --cache ../.review-commit-prep/2026-09-10-frontend-ready/npm-cache --no-audit --no-fund
npm run dev
```

开发地址为 [http://127.0.0.1:5173](http://127.0.0.1:5173)。默认代理 `/api` 到 `http://127.0.0.1:8081`；后端需要已有数据库结构、Redis 和完整本地配置。不会自动初始化数据、迁移数据库或创建管理员。

需要修改接口地址时，将 [.env.example](.env.example) 复制为 `.env.local` 再编辑；本地配置不会提交。`API_PROXY_TARGET` 只由开发/预览服务器读取；浏览器采用 `VITE_API_BASE_URL`，默认同源路径 `/api/blog/v1`。前端环境变量中不能放 JWT 密钥或数据库密码。

后端 CORS 会校验实际 Origin，即使使用 Vite 代理也需允许当前前端来源。将以下字段合入已有后端配置，保留其他字段，修改后重启后端：

```yaml
gin:
  cors:
    allow_origins:
      - http://127.0.0.1:5173
      - http://localhost:5173
      - http://127.0.0.1:4173
      - http://localhost:4173
    allow_methods: [GET, POST, OPTIONS]
    allow_headers: [Content-Type, Authorization, Admin-Authorization]
```

Origin 是精确匹配，不含尾斜杠；实际生产部署请配置生产来源。2026-09-11 已为本机开发配置增补以上四个来源和管理员头，保留原有来源、方法、其他头及凭据；修改后需要启动/重启 Go 服务。

真实服务启动、只读预检和逐步验收见 [本地联调说明](../docs/local-integration.md)。预检命令在 `blog-server/` 执行：`go run ./cmd/devcheck -frontend http://127.0.0.1:4173`，使用开发模式时将端口改为 5173。

## 页面与协议

| 页面 | 路径 | 接口权限 |
|---|---|---|
| 首页 / 搜索 / 分类 | `/` | 公开 |
| 阅读 / 评论 | `/articles/:id` | 阅读公开，评论操作需用户登录 |
| 用户登录 / 注册 | `/login`、`/register` | 公开 |
| 我的文章 / 写作 | `/me/articles`、`/write/new`、`/write/:id` | 用户 Authorization |
| 管理员登录 | `/admin/login` | 公开登录入口 |
| 管理文章 / 编辑 | `/admin/articles`、`/admin/articles/new`、`/admin/articles/:id/edit` | Admin-Authorization |
| 评论审核 | `/admin/comments` | 管理员 |
| 分类管理 | `/admin/categories` | 超级管理员 |

- 用户和管理员的令牌分别保存在当前标签页的 sessionStorage 中。公开请求不带令牌，权限失效只清理对应身份；旧失败响应不能清掉后续新登录。
- HTTP 200 仍可能是业务失败，必须检查信封 code。前端不使用 err_msg。
- 编辑从受保护的详情入口加载，提交完整可编辑字段；status、sort、is_top、comment_status 的 0 不能丢失。管理列表没有正文，不能从列表行直接拼发布请求。
- 路由通过 React Router 的 data router 管理，编辑器使用 useBlocker 和 beforeunload 保护未保存内容；保存中阻止离开，成功后再跳转列表。登出也会检查编辑器是否允许离开。保存请求使登录过期时，已打开的编辑器保留输入并禁用保存，提示先复制内容再重新登录；新的私有页面仍需要鉴权。
- React 状态通过 useSyncExternalStore 订阅不可变会话快照；useResource 取消旧请求并屏蔽陈旧结果，useMutation 防止重复提交及离开页面后的旧响应副作用。
- 评论列表实际返回 `{list,total}`，不同于文章/分类/管理员审核的 PageResult；查询使用 articleID/page/size。
- 审核提交原列表中的 expected_content/expected_status，发生冲突后刷新并重新查看，不能自动以新正文重试旧决定。
- Markdown 经 DOMPurify 和 URL 协议校验后渲染；评论按普通文本展示。
- 点赞与撤回点赞是两个明确动作，不凭本地标记声称知道持久化的“已点赞”状态。点赞需要后端 comment_like 迁移，见 [切换说明](../docs/sql/20260908_comment_like_rollout.md)。

## 检查与构建

```bash
npm test
npm run build
npm run preview
```

`build` 先运行 `tsc --noEmit` 再生成 `dist/`。`package-lock.json` 已生成，应与前端代码一起提交；后续用 `npm ci --ignore-scripts` 复现安装。测试环境在 jsdom 中保留 Node 的 AbortController/AbortSignal，与 Node 24 的原生 Fetch/Request 配套，不替换路由或取消逻辑。

生产部署需将 `dist/` 发布到静态服务器，为 `/api` 配置反向代理，并对前端页面路径使用 `try_files $uri $uri/ /index.html` 或等效回退；API 路径不能回退成 index.html。Vite preview 仅用于本地预览。

19 项测试覆盖双令牌隔离、业务错误、陈旧请求失效、零值查询、完整文章更新、审核快照、Markdown 注入防护，以及 React StrictMode 下的请求取消、登录切换、未保存提示、保存中阻止导航、保存失败/登录过期保留正文和最新管理员权限校验。

当前状态（2026-09-11）：React 类型检查、19 项测试、构建与原夹具验证通过；本轮另外完成 9 组真实浏览器、18 组 API/数据库检查及桌面/移动端视觉核对。MySQL/Redis/Go 和 4173 代理均可用，既有评论迁移已应用，临时测试记录与缓存已清理。JWT 同秒重登录缺陷已用随机 jti 修复，两身份各 3 次真实同秒复验与旧令牌兼容检查通过；具体范围见 [联调记录](../docs/local-integration.md#当前记录)。

## 版本与文件范围

React 首版及 lockfile 已提交为 `1453c50`，配套 JWT 同秒重登录修复为 `9a0ce12`，只读联调预检工具为 `608b78a`。

前端源码、测试、构建配置、`.env.example` 和 `package-lock.json` 纳入版本控制；依赖目录 `node_modules`、构建产物 `dist` 以及本地环境文件由前端 `.gitignore` 排除。安装依赖使用 `npm ci --ignore-scripts`，可复现 lockfile 中的版本。

## 最新交付（2026-10-06）

四语言、每日主题、头像与文章图片、草稿恢复及 Markdown 导入已上线。最新前端117项测试、类型检查和构建通过；导入另通过本地及公网桌面/手机浏览器验收。上文19项是首版历史记录。使用边界见[写作说明](../docs/editor-writing.md)和[国际化说明](../docs/frontend-i18n.md)。
