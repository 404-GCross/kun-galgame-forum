![kun-galgame-forum](https://kungal.com/kungalgame.webp)

### **[English](/README.md)** | **[日本語](/docs/readme/jp.md)** | **[简体中文](/docs/readme/chs.md)** | **[繁體中文](/docs/readme/cht.md)**

**お問い合わせ：[Telegram](https://telegram.me/kungalgame) | [Discord](https://discord.com/invite/5F4FS2cXhX)**

この画像は、ゲーム [Ark Order](https://apps.qoo-app.com/en/app/9593) から提供されており、キャラクターは 'こん'（Kun）です。

> **AI 支援開発について：** 本プロジェクトはバージョン **5.1.0** 以降、Claude Code を含む（ただしこれに限らない）LLM 支援ツールを用いた Vibe Coding を採用しています。**5.0.70** までのすべてのコードは完全に手書きで記述されました。最後の完全手書きコードベースはこちら：[v5.0.70 (commit b4ad59e)](https://github.com/KunMoe/kun-galgame-forum/tree/b4ad59eb77d3eaf36d082aa528651039816e1dfa)

# 鯤 ギャルゲームフォーラム

## ウェブサイト紹介

鯤 ギャルゲームはギャルゲーのジャンルに情熱を注ぐ個人のコレクティブです。現在、以下のサブウェブサイトが含まれています：

- [鯤 ギャルゲームフォーラム](https://kungal.com) (このプロジェクト)
- [鯤 ギャルゲームステッカーパック](https://sticker.kungal.com) (ギャルゲーステッカーパックの収集と作成に特化したウェブサイト)
- [鯤 ギャルゲーム開発ドキュメント](https://soft.moe/kun-visualnovel-docs/kun-forum.html) (このフォーラムは完全にオープンソースであり、開発ドキュメントはここで公開されます)
- [鯤 ギャルゲーム ナビゲーションページ](https://nav.kungal.org/) （完全オープンソースのナビゲーションサイトです！ 鯤 ギャルゲーム のすべてのサブサイトにアクセスできます！）
- [鯤 ギャルゲーム パッチサイト](https://www.moyu.moe) (現時点で世界最先端のビジュアルノベルパッチ資源サイト！永久無料！)
- [鯤 ギャルゲーム ダウンページ](https://down.kungal.com/) （やむを得ずダウンする場合、フォーラムは強制的にこのページにリダイレクトされます）

詳しい情報は、ウェブサイトの「私たちについて」ページをご覧ください

https://www.kungal.com/ja-jp/kungalgame

## 特徴

- **Galgame データベース** — コミュニティ主導の Galgame カタログ。VNDB 連携、多言語メタデータ (EN / JA / ZH-CN / ZH-TW)、評価、タグ、エンジン情報、開発者プロフィールに対応
- **リソース共有** — ゲームパッチ、翻訳、ボイスパックなどのリソースをアップロード・共有。提供者の追跡、プラットフォーム/言語フィルターを搭載
- **ディスカッションフォーラム** — リッチな Markdown 編集 (Milkdown + CodeMirror)、返信、ネストコメント、投票、いいね、お気に入りを備えた本格的なトピックシステム
- **共同編集** — Git 風の PR (Pull Request) ワークフローで Galgame 情報を編集。編集履歴の記録と貢献者クレジット付き
- **ダイレクトメッセージ＆チャット** — Go バックエンドが提供するダイレクトメッセージと連絡先リスト
- **萌萌点（Moemoepoint）システム** — 投稿・リソース共有・Galgame 情報編集などの貢献で得られるコミュニティ評価ポイント。共有 OAuth サービスを通じてエコシステム全体で統一
- **リッチコンテンツ編集** — KaTeX 数式、コードハイライト、ドラッグ＆ドロップ画像アップロードに対応した Milkdown Markdown エディター
- **ダーク / ライトテーマ** — システム連動のカラーモード。ページ透過度・フォント・背景画像のカスタマイズに対応
- **SEO 最適化** — サーバーサイドレンダリング、構造化データ (Schema.org)、サイトマップ生成、Galgame・トピックの RSS フィード

## アーキテクチャ

本プロジェクトは **pnpm workspace モノレポ** で、Go バックエンドと Nuxt フロントエンドで構成されています。共有の PostgreSQL / Redis / Meilisearch および OAuth・画像・Galgame Wiki サービスを所有する **`kun-galgame-infra`** エコシステムの下流アプリです。

| パッケージ | 役割 |
|------|------|
| `apps/api` | **Go (Fiber + GORM) REST API** — 認証、フォーラム、Galgame DB、リソース、検索、メッセージ、定期ジョブ |
| `apps/web` | **Nuxt 4 SSR フロントエンド** — Vue 3、Go API を呼び出す。Nitro サーバーは RSS フィードのみを提供 |

## 技術スタック

| レイヤー | 技術 |
|-------|-----------|
| フロントエンド | [Nuxt 4](https://nuxt.com/) (Vue 3 SSR + Nitro node-server) |
| UI レイヤー | `@kungal/ui-nuxt` — 共有 Nuxt レイヤー |
| スタイル | [Tailwind CSS 4](https://tailwindcss.com/) |
| 状態管理 | [Pinia](https://pinia.vuejs.org/)（永続化付き） |
| エディター | [Milkdown](https://milkdown.dev/) + [CodeMirror](https://codemirror.net/) |
| バックエンド API | [Go 1.26](https://go.dev/) + [Fiber v2](https://gofiber.io/) |
| データベース | PostgreSQL + [GORM](https://gorm.io/)、生 SQL マイグレーション（Prisma は不使用） |
| キャッシュ | Redis |
| 検索 | [Meilisearch](https://www.meilisearch.com/) |
| 認証 | JWT（デュアルトークン — access + refresh）+ OAuth (`kun-galgame-infra`) |
| オブジェクトストレージ | S3 互換（画像は Cloudflare R2、ツールセットアップロードは Backblaze B2） |
| スケジューラ | [robfig/cron](https://github.com/robfig/cron)（日次リセット、統計など） |
| バリデーション | [Zod](https://zod.dev/)（フロントエンド） |
| デプロイ | Docker → GHCR → [Dokploy](https://dokploy.com/)（または `scripts/` 経由で PM2） |
| アクセス解析 | [Umami](https://umami.is/) |

## プロジェクト構成

```text
├── apps/
│   ├── api/                 # Go Fiber バックエンド (REST API)
│   │   ├── cmd/             # server、migrate および各種ワンオフ backfill/sync ツール
│   │   ├── internal/        # ドメインモジュール (user、topic、galgame、moemoepoint、message、search…)
│   │   ├── migrations/      # 生 SQL マイグレーション (.up.sql / .down.sql)
│   │   └── pkg/             # 横断的関心事 (config、logger、health…)
│   └── web/                 # Nuxt 4 SSR フロントエンド
│       ├── app/             # ページ、コンポーネント、composable、ストア (Pinia)、バリデーション
│       ├── server/          # Nitro ルート（RSS フィードのみ）
│       └── shared/          # 共有 TypeScript 型・ユーティリティ
├── docker/                  # Dockerfile + 環境変数サンプル + Docker README
├── docker-compose*.yml      # base (joins infra) + prod
├── scripts/                 # PM2 デプロイスクリプト (deploy / start / stop / restart)
└── docs/                    # ドキュメント
```

## はじめに

**前提条件：** Node.js 22+（Corepack/pnpm 同梱）、Go 1.26+、PostgreSQL、Redis、および（任意で）Meilisearch。完全な機能には `kun-galgame-infra` のサービス（OAuth、画像、Galgame Wiki）も必要です。

```bash
# ワークスペースの依存関係をインストール
pnpm install

# 環境変数を設定（アプリごと）
cp apps/api/.env.example apps/api/.env   # Go API：DB、Redis、OAuth、S3、メール、検索…
cp apps/web/.env.example apps/web/.env   # Nuxt：API ベース URL、OAuth クライアント、画像/Wiki URL

# データベースマイグレーションを実行（リポジトリ横断の順序は docs/ を参照）
pnpm migrate

# フロント・バックを同時に起動 — API は :2334、Web は :2333
pnpm dev
#   pnpm dev:api   # Go API のみ（air ホットリロード）→ http://127.0.0.1:2334
#   pnpm dev:web   # Nuxt のみ                       → http://127.0.0.1:2333
```

または、コンテナでスタック全体を実行します（[`docker/README.md`](/docker/README.md) 参照）：

```bash
docker compose up -d api web        # kun-galgame-infra must be running first
```

## スクリプト

| コマンド | 説明 |
|---------|-------------|
| `pnpm dev` | API + Web を同時に起動（並列） |
| `pnpm dev:web` / `pnpm dev:api` | 単一アプリを起動 |
| `pnpm build` | 本番ビルド — Go API のあと Nuxt web |
| `pnpm lint` / `pnpm lint:fix` | ESLint（フロントエンド） |
| `pnpm typecheck` | `vue-tsc` 型チェック（フロントエンド） |
| `pnpm format` | アプリ横断で Prettier / gofmt |
| `pnpm vet` | `go vet`（バックエンド） |
| `pnpm test:api` | `go test`（バックエンド） |
| `pnpm migrate` / `pnpm migrate:down` | DB マイグレーションの実行 / ロールバック（バックエンド） |
| `pnpm sitemap` | サイトマップを生成 |
| `pnpm prod:deploy` / `prod:start` / `prod:stop` / `prod:restart` | PM2 デプロイスクリプト |

## 参加 / お問い合わせ

- [Telegram グループ](https://telegram.me/kungalgame)
- [Twitter / X](https://twitter.com/kungalgame)
- [GitHub リポジトリ](https://github.com/KunMoe/kun-galgame-forum)
- [Discord グループ](https://discord.com/invite/5F4FS2cXhX)
- [YouTube チャンネル](https://youtube.com/@kungalgame)
- [Bilibili](https://space.bilibili.com/1748455574)

## ライセンス

このプロジェクトは `AGPL-3.0` オープンソースライセンスに従います。
