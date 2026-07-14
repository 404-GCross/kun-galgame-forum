![kun-galgame-forum](https://kungal.com/kungalgame.webp)

### **[English](/README.md)** | **[日本語](/docs/readme/jp.md)** | **[简体中文](/docs/readme/chs.md)** | **[繁體中文](/docs/readme/cht.md)**

**聯繫我們：[Telegram](https://telegram.me/kungalgame) | [Discord](https://discord.com/invite/5F4FS2cXhX)**

圖片來源為遊戲 [方舟指令](https://apps.qoo-app.com/en/app/9593) 中的角色鯤（KUN）

> **AI 輔助開發說明：** 本項目自 **5.1.0** 版本起使用包含但不限於 Claude Code 等 LLM 輔助工具進行 Vibe Coding。**5.0.70** 及之前版本的所有程式碼均完全由手工編寫。最後一個完全手寫的程式碼庫版本見：[v5.0.70 (commit b4ad59e)](https://github.com/KunMoe/kun-galgame-forum/tree/b4ad59eb77d3eaf36d082aa528651039816e1dfa)

# 鯤 Galgame 論壇

## 項目簡介

鯤 Galgame 是一個 Galgame 集體，它由無數熱愛 Galgame 遊戲體裁的人們組成。它目前有以下幾個子網站：

- [鯤 Galgame 論壇](https://kungal.com) (本項目)
- [鯤 Galgame 表情包](https://sticker.kungal.com) (一個專注於收集制作 Galgame 表情包的網站)
- [鯤 Galgame 開發文檔](https://soft.moe/kun-visualnovel-docs/kun-forum.html) (本論壇是完全開源的，開發文檔將會全部公開在這裡)
- [鯤 Galgame 導航頁面](https://nav.kungal.org/) （完全開源的導航站！ 可以前往鯤 Galgame 所有子網站！）
- [鯤 Galgame 補丁站](https://www.moyu.moe) (目前全球最先進的視覺小說補丁資源網站！永久免費！)
- [鯤 Galgame 論壇下線頁面](https://down.kungal.com/) （碰到不得不下線的時候，我們會將論壇強制重定向到此頁面）

更多資訊請直接訪問網站的關於我們頁面

https://www.kungal.com/zh-tw/kungalgame

## 特性

- **Galgame 資料庫** — 社區驅動的 Galgame 目錄，整合 VNDB，支援多語言元資料 (EN / JA / ZH-CN / ZH-TW)、評分、標籤、引擎資訊和開發者資料
- **資源分享** — 上傳和分享遊戲補丁、漢化、語音包和其他資源，具備提供者追蹤及平台和語言篩選功能
- **討論論壇** — 功能齊全的話題系統，支援富文本 Markdown 編輯 (Milkdown + CodeMirror)、回覆、子評論、投票、點讚和收藏
- **協作編輯** — 採用 Git 風格的 PR (Pull Request) 工作流進行 Galgame 資訊編輯，包含編輯歷史記錄及貢獻者致謝
- **私信與聊天** — 由 Go 後端提供的私信與聯絡人列表
- **萌萌點系統** — 透過貢獻（發帖、分享資源、編輯 Galgame 資訊）獲得的社區聲望點數，並透過共享的 OAuth 服務在整個生態中統一
- **多媒體內容編輯** — 支援 KaTeX 數學公式、程式碼高亮和拖曳上傳圖片的 Milkdown Markdown 編輯器
- **深色 / 淺色主題** — 系統自帶的色彩模式切換，支援自訂頁面透明度、字型和背景圖片
- **SEO 優化** — 伺服器端渲染，結構化資料 (Schema.org)、網站地圖生成和 Galgame 及話題的 RSS 訂閱

## 架構

本項目是一個 **pnpm workspace monorepo**，由 Go 後端與 Nuxt 前端組成。它是 **`kun-galgame-infra`** 生態中的下游應用——該生態擁有共享的 PostgreSQL / Redis / Meilisearch，以及 OAuth、圖床、Galgame Wiki 等服務。

| 套件 | 職責 |
|------|------|
| `apps/api` | **Go (Fiber + GORM) REST API** — 鑑權、論壇、Galgame 庫、資源、搜尋、訊息、定時任務 |
| `apps/web` | **Nuxt 4 SSR 前端** — Vue 3，呼叫 Go API；Nitro 伺服端只負責 RSS 訂閱 |

## 技術棧

| 層 | 技術 |
|-------|-----------|
| 前端 | [Nuxt 4](https://nuxt.com/) (Vue 3 SSR + Nitro node-server) |
| UI 層 | `@kungal/ui-nuxt` — 共享 Nuxt layer |
| 樣式 | [Tailwind CSS 4](https://tailwindcss.com/) |
| 狀態管理 | [Pinia](https://pinia.vuejs.org/) (帶持久化) |
| 文字編輯器 | [Milkdown](https://milkdown.dev/) + [CodeMirror](https://codemirror.net/) |
| 後端 API | [Go 1.26](https://go.dev/) + [Fiber v2](https://gofiber.io/) |
| 資料庫 | PostgreSQL + [GORM](https://gorm.io/)，原生 SQL 遷移（不再使用 Prisma） |
| 快取 | Redis |
| 搜尋 | [Meilisearch](https://www.meilisearch.com/) |
| 身分驗證 | JWT (雙 token — access + refresh) + OAuth (`kun-galgame-infra`) |
| 物件儲存 | 相容 S3（圖床用 Cloudflare R2，工具集上傳用 Backblaze B2） |
| 定時任務 | [robfig/cron](https://github.com/robfig/cron)（每日重置、統計等） |
| 資料驗證 | [Zod](https://zod.dev/)（前端） |
| 部署 | Docker → GHCR → [Dokploy](https://dokploy.com/)（或透過 `scripts/` 使用 PM2） |
| 流量分析 | [Umami](https://umami.is/) |

## 項目結構

```text
├── apps/
│   ├── api/                 # Go Fiber 後端 (REST API)
│   │   ├── cmd/             # server、migrate 及若干一次性 backfill/sync 工具
│   │   ├── internal/        # 領域模組 (user、topic、galgame、moemoepoint、message、search…)
│   │   ├── migrations/      # 原生 SQL 遷移 (.up.sql / .down.sql)
│   │   └── pkg/             # 橫切關注點 (config、logger、health…)
│   └── web/                 # Nuxt 4 SSR 前端
│       ├── app/             # 頁面、元件、組合式函式、狀態 (Pinia)、校驗
│       ├── server/          # Nitro 路由（僅 RSS 訂閱）
│       └── shared/          # 共享 TypeScript 型別與工具
├── docker/                  # Dockerfile + 環境變數範例 + Docker 說明
├── docker-compose*.yml      # base (joins infra) + prod
├── scripts/                 # PM2 部署腳本 (deploy / start / stop / restart)
└── docs/                    # 開發文檔
```

## 快速開始

**前置依賴：** Node.js 22+（含 Corepack/pnpm）、Go 1.26+、PostgreSQL、Redis，以及（可選）Meilisearch。完整功能還需要 `kun-galgame-infra` 的服務（OAuth、圖床、Galgame Wiki）。

```bash
# 安裝 workspace 依賴
pnpm install

# 設定環境變數（按 app 分別設定）
cp apps/api/.env.example apps/api/.env   # Go API：資料庫、Redis、OAuth、S3、郵件、搜尋…
cp apps/web/.env.example apps/web/.env   # Nuxt：API 位址、OAuth 用戶端、圖床/Wiki 位址

# 執行資料庫遷移（跨倉庫遷移順序詳見 docs/）
pnpm migrate

# 同時啟動前後端 — API 在 :2334，Web 在 :2333
pnpm dev
#   pnpm dev:api   # 僅 Go API（air 熱重載）→ http://127.0.0.1:2334
#   pnpm dev:web   # 僅 Nuxt              → http://127.0.0.1:2333
```

或使用容器執行整套服務（詳見 [`docker/README.md`](/docker/README.md)）：

```bash
docker compose up -d api web        # kun-galgame-infra must be running first
```

## 腳本命令

| 命令 | 描述 |
|---------|-------------|
| `pnpm dev` | 同時啟動 API + Web（並行） |
| `pnpm dev:web` / `pnpm dev:api` | 單獨啟動某個 app |
| `pnpm build` | 生產建置 — 先 Go API 後 Nuxt web |
| `pnpm lint` / `pnpm lint:fix` | ESLint（前端） |
| `pnpm typecheck` | `vue-tsc` 型別檢查（前端） |
| `pnpm format` | 跨 app 執行 Prettier / gofmt |
| `pnpm vet` | `go vet`（後端） |
| `pnpm test:api` | `go test`（後端） |
| `pnpm migrate` / `pnpm migrate:down` | 執行 / 回滾資料庫遷移（後端） |
| `pnpm sitemap` | 生成網站地圖 |
| `pnpm prod:deploy` / `prod:start` / `prod:stop` / `prod:restart` | PM2 部署腳本 |

## 加入 / 聯繫我們

- [Telegram 群](https://telegram.me/kungalgame)
- [Twitter / X](https://twitter.com/kungalgame)
- [GitHub Repository](https://github.com/KunMoe/kun-galgame-forum)
- [Discord 群](https://discord.com/invite/5F4FS2cXhX)
- [YouTube Channel](https://youtube.com/@kungalgame)
- [Bilibili](https://space.bilibili.com/1748455574)

## License

本項目遵從 `AGPL-3.0` 開源協議
