![kun-galgame-forum](https://kungal.com/kungalgame.webp)

### **[English](/README.md)** | **[日本語](/docs/readme/jp.md)** | **[简体中文](/docs/readme/chs.md)** | **[繁體中文](/docs/readme/cht.md)**

**联系我们：[Telegram](https://telegram.me/kungalgame) | [Discord](https://discord.com/invite/5F4FS2cXhX)**

图片来源为游戏 [方舟指令](https://apps.qoo-app.com/en/app/9593) 中的角色鲲（KUN）

> **AI辅助开发说明：** 本项目自 **5.1.0** 版本起使用包含但不限于 Claude Code 等 LLM 辅助工具进行 Vibe Coding。**5.0.70** 及之前版本的所有代码均完全由手工编写。最后一个完全手写的代码库版本见：[v5.0.70 (commit b4ad59e)](https://github.com/KunMoe/kun-galgame-forum/tree/b4ad59eb77d3eaf36d082aa528651039816e1dfa)

# 鲲 Galgame 论坛

## 项目简介

鲲 Galgame 是一个 Galgame 集体，它由无数热爱 Galgame 游戏体裁的人们组成。它目前有以下几个子网站：

- [鲲 Galgame 论坛](https://kungal.com) (本项目)
- [鲲 Galgame 表情包](https://sticker.kungal.com) (一个专注于收集制作 Galgame 表情包的网站)
- [鲲 Galgame 开发文档](https://soft.moe/kun-visualnovel-docs/kun-forum.html) (本论坛是完全开源的，开发文档将会全部公开在这里)
- [鲲 Galgame 导航页面](https://nav.kungal.org/) （完全开源的导航站！ 可以前往鲲 Galgame 所有子网站！）
- [鲲 Galgame 补丁站](https://www.moyu.moe) (目前全球最先进的视觉小说补丁资源网站！永久免费！)
- [鲲 Galgame 论坛下线页面](https://down.kungal.com/) （碰到不得不下线的时候，我们会将论坛强制重定向到此页面）

更多信息请直接访问网站的关于我们页面

https://www.kungal.com/kungalgame

## 特性

- **Galgame 数据库** — 社区驱动的 Galgame 目录，集成 VNDB，支持多语言元数据 (EN / JA / ZH-CN / ZH-TW)、评分、标签、引擎信息和开发者资料
- **资源分享** — 上传和分享游戏补丁、汉化、语音包和其他资源，具备提供者追踪及平台和语言筛选功能
- **讨论论坛** — 功能齐全的话题系统，支持富文本 Markdown 编辑 (Milkdown + CodeMirror)、回复、子评论、投票、点赞和收藏
- **协作编辑** — 采用 Git 风格的 PR (Pull Request) 工作流进行 Galgame 信息编辑，包含编辑历史记录及贡献者致谢
- **私信与聊天** — 由 Go 后端提供的私信与联系人列表
- **萌萌点系统** — 通过贡献（发帖、分享资源、编辑 Galgame 信息）获得的社区声望点数，并通过共享的 OAuth 服务在整个生态中统一
- **多媒体内容编辑** — 支持 KaTeX 数学公式、代码高亮和拖拽上传图片的 Milkdown Markdown 编辑器
- **深色 / 浅色主题** — 系统自带的色彩模式切换，支持自定义页面透明度、字体和背景图片
- **SEO 优化** — 服务端渲染，结构化数据 (Schema.org)、站点地图生成和 Galgame 及话题的 RSS 订阅

## 架构

本项目是一个 **pnpm workspace monorepo**，由 Go 后端与 Nuxt 前端组成。它是 **`kun-galgame-infra`** 生态中的下游应用——该生态拥有共享的 PostgreSQL / Redis / Meilisearch，以及 OAuth、图床、Galgame Wiki 等服务。

| 包 | 职责 |
|------|------|
| `apps/api` | **Go (Fiber + GORM) REST API** — 鉴权、论坛、Galgame 库、资源、搜索、消息、定时任务 |
| `apps/web` | **Nuxt 4 SSR 前端** — Vue 3，调用 Go API；Nitro 服务端只负责 RSS 订阅 |

## 技术栈

| 层 | 技术 |
|-------|-----------|
| 前端 | [Nuxt 4](https://nuxt.com/) (Vue 3 SSR + Nitro node-server) |
| UI 层 | `@kungal/ui-nuxt` — 共享 Nuxt layer |
| 样式 | [Tailwind CSS 4](https://tailwindcss.com/) |
| 状态管理 | [Pinia](https://pinia.vuejs.org/) (带持久化) |
| 文本编辑器 | [Milkdown](https://milkdown.dev/) + [CodeMirror](https://codemirror.net/) |
| 后端 API | [Go 1.26](https://go.dev/) + [Fiber v2](https://gofiber.io/) |
| 数据库 | PostgreSQL + [GORM](https://gorm.io/)，原生 SQL 迁移（不再使用 Prisma） |
| 缓存 | Redis |
| 搜索 | [Meilisearch](https://www.meilisearch.com/) |
| 身份验证 | JWT (双 token — access + refresh) + OAuth (`kun-galgame-infra`) |
| 对象存储 | 兼容 S3（图床用 Cloudflare R2，工具集上传用 Backblaze B2） |
| 定时任务 | [robfig/cron](https://github.com/robfig/cron)（每日重置、统计等） |
| 数据验证 | [Zod](https://zod.dev/)（前端） |
| 部署 | Docker → GHCR → [Dokploy](https://dokploy.com/)（或通过 `scripts/` 使用 PM2） |
| 流量分析 | [Umami](https://umami.is/) |

## 项目结构

```text
├── apps/
│   ├── api/                 # Go Fiber 后端 (REST API)
│   │   ├── cmd/             # server、migrate 及若干一次性 backfill/sync 工具
│   │   ├── internal/        # 领域模块 (user、topic、galgame、moemoepoint、message、search…)
│   │   ├── migrations/      # 原生 SQL 迁移 (.up.sql / .down.sql)
│   │   └── pkg/             # 横切关注点 (config、logger、health…)
│   └── web/                 # Nuxt 4 SSR 前端
│       ├── app/             # 页面、组件、组合式函数、状态 (Pinia)、校验
│       ├── server/          # Nitro 路由（仅 RSS 订阅）
│       └── shared/          # 共享 TypeScript 类型与工具
├── docker/                  # Dockerfile + 环境变量示例 + Docker 说明
├── docker-compose*.yml      # base (joins infra) + prod
├── scripts/                 # PM2 部署脚本 (deploy / start / stop / restart)
└── docs/                    # 开发文档
```

## 快速开始

**前置依赖：** Node.js 22+（含 Corepack/pnpm）、Go 1.26+、PostgreSQL、Redis，以及（可选）Meilisearch。完整功能还需要 `kun-galgame-infra` 的服务（OAuth、图床、Galgame Wiki）。

```bash
# 安装 workspace 依赖
pnpm install

# 配置环境变量（按 app 分别配置）
cp apps/api/.env.example apps/api/.env   # Go API：数据库、Redis、OAuth、S3、邮件、搜索…
cp apps/web/.env.example apps/web/.env   # Nuxt：API 地址、OAuth 客户端、图床/Wiki 地址

# 执行数据库迁移（跨仓库迁移顺序详见 docs/）
pnpm migrate

# 同时启动前后端 — API 在 :2334，Web 在 :2333
pnpm dev
#   pnpm dev:api   # 仅 Go API（air 热重载）→ http://127.0.0.1:2334
#   pnpm dev:web   # 仅 Nuxt              → http://127.0.0.1:2333
```

或使用容器运行整套服务（详见 [`docker/README.md`](/docker/README.md)）：

```bash
docker compose up -d api web        # kun-galgame-infra must be running first
```

## 脚本命令

| 命令 | 描述 |
|---------|-------------|
| `pnpm dev` | 同时启动 API + Web（并行） |
| `pnpm dev:web` / `pnpm dev:api` | 单独启动某个 app |
| `pnpm build` | 生产构建 — 先 Go API 后 Nuxt web |
| `pnpm lint` / `pnpm lint:fix` | ESLint（前端） |
| `pnpm typecheck` | `vue-tsc` 类型检查（前端） |
| `pnpm format` | 跨 app 运行 Prettier / gofmt |
| `pnpm vet` | `go vet`（后端） |
| `pnpm test:api` | `go test`（后端） |
| `pnpm migrate` / `pnpm migrate:down` | 执行 / 回滚数据库迁移（后端） |
| `pnpm sitemap` | 生成站点地图 |
| `pnpm prod:deploy` / `prod:start` / `prod:stop` / `prod:restart` | PM2 部署脚本 |

## 加入 / 联系我们

- [Telegram 群组](https://telegram.me/kungalgame)
- [Twitter / X](https://twitter.com/kungalgame)
- [GitHub 仓库](https://github.com/KunMoe/kun-galgame-forum)
- [Discord 群组](https://discord.com/invite/5F4FS2cXhX)
- [YouTube 频道](https://youtube.com/@kungalgame)
- [Bilibili](https://space.bilibili.com/1748455574)

## License

本项目遵循 `AGPL-3.0` 开源协议
