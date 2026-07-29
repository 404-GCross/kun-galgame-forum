// Freeze the wiki-era creator onto every local galgame row (A2-3, migration 066).
//
// kungal renders an author chip on every galgame card. That user id used to
// ride along on the wiki's own brief; the catalog re-anchoring removes it as a
// source, because the catalog is a cross-source registry of WORKS and carries
// no product's submitter by design (doc 106 R2 — wiki state-machine fields
// never cross into the public catalog contract).
//
// The wiki-era attribution is real history, so it is FROZEN rather than dropped:
// this command copies it once into galgame.creator_user_id from the surviving
// `/internal/galgame/meta` ownership op, and nothing writes that column again.
// Newly created entries get their creator from the same op at write time.
//
// SOURCE: GET /internal/galgame/meta?ids= — status-blind and batched 100 at a
// time. Status-blind matters: an entry still in review has a creator too, and a
// published-only read would leave exactly those rows unattributed.
//
// Idempotent: re-running overwrites with the wiki's current value. Ids the op
// does not resolve (hard-deleted upstream) are left untouched rather than
// nulled — an unknown creator and a deleted upstream row are different facts,
// and only the first should render as "no author".
//
// Usage:
//
//	go run ./cmd/backfill-galgame-creator              # do it
//	go run ./cmd/backfill-galgame-creator --dry-run    # report-only
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

// metaBatch is the ownership op's own ceiling: over 100 ids it 400s rather than
// truncating (a truncated ownership answer reads as "not the owner", which is
// the worst failure this lane has).
const metaBatch = 100

// writeBatch applies one chunk of resolved creators in a single transaction.
// Rows are written one statement each: the ids are ints and the values are
// ints, but a bulk VALUES join would need pgx to infer both column types from
// an untyped literal list, and the fetch — not the write — is the bottleneck.
func writeBatch(db *gorm.DB, creators map[int]int) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for gid, userID := range creators {
			if err := tx.Exec(
				"UPDATE galgame SET creator_user_id = ? WHERE id = ?", userID, gid,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func main() {
	_ = godotenv.Load()

	dryRun := flag.Bool("dry-run", false, "Fetch from the wiki face but do not update rows")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)

	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)
	gc := client.New(
		cfg.NextMoeAPI.BaseURL,
		cfg.NextMoeAPI.APIKey,
		cfg.NextMoeAPI.ImageCDNBase,
	)
	ctx := context.Background()

	// kungal's local galgame table IS the id set worth attributing: a catalog
	// work kungal never ingested has no local row and no wiki-era creator.
	var ids []int
	if err := db.Table("galgame").Order("id").Pluck("id", &ids).Error; err != nil {
		slog.Error("读取本地 galgame id 失败", "error", err)
		os.Exit(1)
	}
	slog.Info("开始回填 galgame 创建者快照", "total", len(ids), "dry_run", *dryRun)

	var resolved, missing int
	for start := 0; start < len(ids); start += metaBatch {
		end := min(start+metaBatch, len(ids))
		chunk := ids[start:end]

		rows, appErr := gc.GalgameMeta(ctx, chunk)
		if appErr != nil {
			slog.Error("读取 galgame 元信息失败", "from", chunk[0], "error", appErr.Message)
			os.Exit(1)
		}

		creators := make(map[int]int, len(rows))
		for _, gid := range chunk {
			row, ok := rows[gid]
			if !ok || row.UserID <= 0 {
				missing++
				continue
			}
			creators[gid] = row.UserID
		}
		resolved += len(creators)

		if !*dryRun && len(creators) > 0 {
			if err := writeBatch(db, creators); err != nil {
				slog.Error("写入创建者快照失败", "from", chunk[0], "error", err)
				os.Exit(1)
			}
		}
		slog.Info("进度", "done", end, "total", len(ids), "resolved", resolved, "missing", missing)
	}

	slog.Info("回填完成", "resolved", resolved, "missing", missing, "dry_run", *dryRun)
}
