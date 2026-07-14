![kun-galgame-forum](https://kungal.com/kungalgame.webp)

### **[English](/README.md)** | **[日本語](/docs/readme/jp.md)** | **[简体中文](/docs/readme/chs.md)** | **[繁體中文](/docs/readme/cht.md)**

**Contact us：[Telegram](https://telegram.me/kungalgame) | [Discord](https://discord.com/invite/5F4FS2cXhX)**

The image is sourced from the game [Ark Order](https://apps.qoo-app.com/en/app/9593), featuring the character 'KUN' (鲲).

> **Note on AI-assisted development:** Starting from version **5.1.0**, this project uses LLM-assisted tools including but not limited to Claude Code for Vibe Coding. All code up to and including version **5.0.70** was written entirely by hand. The last purely hand-written codebase can be found at: [v5.0.70 (commit b4ad59e)](https://github.com/KunMoe/kun-galgame-forum/tree/b4ad59eb77d3eaf36d082aa528651039816e1dfa)

# KUN Visual Novel Forum

## Website Introduction

KUN Visual Novel is a collective of individuals passionate about the Galgame genre. It currently consists of the following sub-websites:

- [KUN Visual Novel Forum](https://kungal.com) (this project)
- [KUN Visual Novel Sticker Pack](https://sticker.kungal.com) (a website dedicated to collecting and creating Galgame sticker packs)
- [KUN Visual Novel Development Documentation](https://soft.moe/kun-visualnovel-docs/kun-forum.html) (this forum is entirely open source, and the development documentation will be publicly available here)
- [Kun Visual Novel Navigation Page](https://nav.kungal.org/) （A completely open-source navigation site! You can visit all Kun Visual Novel subsites!）
- [Kun Visual Novel Patch](https://www.moyu.moe) (The most advanced visual novel patch resource website in the world at the moment! Free forever!)
- [Kun Visual Novel Forum Downtime Page](https://down.kungal.com/) （In the event of unavoidable downtime, we will forcibly redirect the forum to this page.）

For more information, please visit the website's About Us page directly

https://www.kungal.com/kungalgame

## Features

- **Galgame Database** — Community-driven Galgame catalog with VNDB integration, multi-language metadata (EN / JA / ZH-CN / ZH-TW), ratings, tags, engine info, and developer profiles
- **Resource Sharing** — Upload and share game patches, translations, voice packs, and other resources with provider tracking and platform/language filters
- **Discussion Forum** — Full-featured topic system with rich Markdown editing (Milkdown + CodeMirror), replies, nested comments, polls, upvotes, and favorites
- **Collaborative Editing** — Git-style PR (Pull Request) workflow for Galgame information edits, with edit history tracking and contributor credits
- **Private Messaging & Chat** — Direct messages and contact list served by the Go API
- **Moemoepoint System** — Community reputation points earned through contributions (posting, sharing resources, editing Galgame info), unified across the ecosystem via the shared OAuth service
- **Rich Content Editing** — Milkdown Markdown editor with KaTeX math formulas, code highlighting, and image upload via drag & drop
- **Dark / Light Theme** — System-aware color mode with customizable page transparency, fonts, and background images
- **SEO Optimized** — Server-side rendering, structured data (Schema.org), sitemap generation, and RSS feeds for Galgames and topics

## Architecture

This is a **pnpm workspace monorepo** with a Go backend and a Nuxt frontend. It is a downstream app in the **`kun-galgame-infra`** ecosystem, which owns the shared PostgreSQL / Redis / Meilisearch and the OAuth, image, and Galgame-wiki services.

| Package | Role |
|---------|------|
| `apps/api` | **Go (Fiber + GORM) REST API** — auth, forum, Galgame DB, resources, search, messaging, scheduled jobs |
| `apps/web` | **Nuxt 4 SSR frontend** — Vue 3, calls the Go API; the Nitro server only serves RSS feeds |

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Frontend | [Nuxt 4](https://nuxt.com/) (Vue 3 SSR, Nitro node-server) |
| UI Layer | `@kungal/ui-nuxt` — shared Nuxt layer |
| Styling | [Tailwind CSS 4](https://tailwindcss.com/) |
| State Management | [Pinia](https://pinia.vuejs.org/) with persisted state |
| Editor | [Milkdown](https://milkdown.dev/) + [CodeMirror](https://codemirror.net/) |
| Backend API | [Go 1.26](https://go.dev/) + [Fiber v2](https://gofiber.io/) |
| Database | PostgreSQL via [GORM](https://gorm.io/) — raw-SQL migrations (no Prisma) |
| Cache | Redis |
| Search | [Meilisearch](https://www.meilisearch.com/) |
| Authentication | JWT (dual token — access + refresh) + OAuth (`kun-galgame-infra`) |
| Object Storage | S3-compatible (Cloudflare R2 image bed + Backblaze B2 for toolset uploads) |
| Scheduler | [robfig/cron](https://github.com/robfig/cron) (daily resets, stats) |
| Validation | [Zod](https://zod.dev/) (web) |
| Deployment | Docker → GHCR → [Dokploy](https://dokploy.com/) (or PM2 via `scripts/`) |
| Analytics | [Umami](https://umami.is/) |

## Project Structure

```
├── apps/
│   ├── api/                 # Go Fiber backend (REST API)
│   │   ├── cmd/             # server, migrate, + one-off backfill/sync tools
│   │   ├── internal/        # domain modules (user, topic, galgame, moemoepoint, message, search, …)
│   │   ├── migrations/      # raw SQL migrations (.up.sql / .down.sql)
│   │   └── pkg/             # cross-cutting (config, logger, health, …)
│   └── web/                 # Nuxt 4 SSR frontend
│       ├── app/             # pages, components, composables, store (Pinia), validations
│       ├── server/          # Nitro routes (RSS feeds only)
│       └── shared/          # shared TypeScript types & utils
├── docker/                  # Dockerfiles + env examples + Docker README
├── docker-compose*.yml      # base (joins infra) + prod
├── scripts/                 # PM2 deploy scripts (deploy / start / stop / restart)
└── docs/                    # documentation
```

## Getting Started

**Prerequisites:** Node.js 22+ (with Corepack/pnpm), Go 1.26+, PostgreSQL, Redis, and (optionally) Meilisearch. Full functionality also needs the `kun-galgame-infra` services (OAuth, image, Galgame-wiki).

```bash
# Install workspace dependencies
pnpm install

# Configure environment (per app)
cp apps/api/.env.example apps/api/.env   # Go API: DB, Redis, OAuth, S3, mail, search, …
cp apps/web/.env.example apps/web/.env   # Nuxt: API base URL, OAuth client, image/wiki URLs

# Run database migrations (see docs/ for the cross-repo migration order)
pnpm migrate

# Start both apps in parallel — API on :2334, Web on :2333
pnpm dev
#   pnpm dev:api   # Go API only (air hot-reload) → http://127.0.0.1:2334
#   pnpm dev:web   # Nuxt only                     → http://127.0.0.1:2333
```

Or run the whole stack in containers (see [`docker/README.md`](/docker/README.md)):

```bash
docker compose up -d api web        # kun-galgame-infra must be running first
```

## Scripts

| Command | Description |
|---------|-------------|
| `pnpm dev` | Run API + Web together (parallel) |
| `pnpm dev:web` / `pnpm dev:api` | Run a single app |
| `pnpm build` | Production build — Go API then Nuxt web |
| `pnpm lint` / `pnpm lint:fix` | ESLint (web) |
| `pnpm typecheck` | `vue-tsc` type checking (web) |
| `pnpm format` | Prettier / gofmt across apps |
| `pnpm vet` | `go vet` (api) |
| `pnpm test:api` | `go test` (api) |
| `pnpm migrate` / `pnpm migrate:down` | Run / roll back DB migrations (api) |
| `pnpm sitemap` | Generate the sitemap |
| `pnpm prod:deploy` / `prod:start` / `prod:stop` / `prod:restart` | PM2 deployment scripts |

## Join / Contact Us

- [Telegram Group](https://telegram.me/kungalgame)
- [Twitter / X](https://twitter.com/kungalgame)
- [GitHub Repository](https://github.com/KunMoe/kun-galgame-forum)
- [Discord Group](https://discord.com/invite/5F4FS2cXhX)
- [YouTube Channel](https://youtube.com/@kungalgame)
- [Bilibili](https://space.bilibili.com/1748455574)

## License

This project follows the `AGPL-3.0` open-source license.
