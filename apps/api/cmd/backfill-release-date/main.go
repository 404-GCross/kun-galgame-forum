// Backfill galgame.release_date from wiki for every local galgame row.
//
// kungal's GET /galgame browse list filters + sorts on the LOCAL galgame
// table (migration 013 added the release_date column). This command
// mirrors wiki's canonical release_date into that column so the list can
// offer release year/month filtering (wiki §17).
//
// SOURCE: the detail endpoint GET /galgame/:gid, NOT /galgame/batch.
// The batch endpoint returns a *lite* brief that omits release_date
// (01-galgame.md §GET /galgame/batch lists no such field), so an earlier
// batch-based version silently wrote all-NULL. Single-detail isn't
// content-filtered (§16), so NSFW published rows return too. Fetches run
// through a worker pool — precise to kungal's own id set and fast against
// a local wiki.
//
// Idempotent: re-running overwrites release_date with wiki's current
// value (dated → set, null/missing-date → NULL). release_date is
// effectively immutable once a title ships, so an occasional re-run is
// also the refresh mechanism (kungal has no write path that mutates it
// locally). ids wiki returns 404 for (deleted upstream / pending stub)
// are left untouched.
//
// Usage:
//
//	go run ./cmd/backfill-release-date                 # do it
//	go run ./cmd/backfill-release-date --dry-run       # report-only
//	go run ./cmd/backfill-release-date --workers=32    # tune concurrency
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sync"

	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

// detailResult carries one galgame's resolved release_date back from a
// worker. `missing` = wiki returned an error/404 (leave the local row
// untouched); otherwise `date` is the wiki value (nil → write SQL NULL).
type detailResult struct {
	id      int
	date    *string
	missing bool
}

// fetchReleaseDate pulls GET /galgame/:gid and extracts just release_date
// from the `{ "galgame": { … } }` envelope. Everything else in the heavy
// detail payload is skipped by the minimal target struct.
func fetchReleaseDate(ctx context.Context, gc *galgameClient.GalgameClient, id int) detailResult {
	data, appErr := gc.Get(ctx, fmt.Sprintf("/galgame/%d", id), nil)
	if appErr != nil {
		// 404 (deleted / pending-not-owned) or transport error → skip.
		return detailResult{id: id, missing: true}
	}
	var env struct {
		Galgame struct {
			ReleaseDate *string `json:"release_date"`
		} `json:"galgame"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return detailResult{id: id, missing: true}
	}
	return detailResult{id: id, date: env.Galgame.ReleaseDate}
}

// writeBatch applies a chunk of resolved dates inside one transaction:
//
//   - dated ids → one per-row `SET release_date = ?::date WHERE id = ?`.
//     The date arrives as a string; `?::date` lets pgx encode it as text
//     (its natural path) and Postgres cast text→date, while the int id
//     sits in the int-typed `WHERE id = ?` slot. (A bulk VALUES (?,?)
//     update can't work here: pgx infers the VALUES id column as text and
//     fails to encode the int — column-side casts don't fix param typing.
//     The writes aren't the bottleneck anyway; the wiki fetch is.)
//   - null ids → one `UPDATE … WHERE id IN (…) SET NULL` (covers the
//     refresh case where wiki dropped a date; no-op on a fresh column).
//
// ids wiki 404'd aren't passed in, so they're left untouched. The whole
// chunk commits atomically.
func writeBatch(db *gorm.DB, datedIDs []int, datedVals []string, nullIDs []int) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for i, id := range datedIDs {
			if err := tx.Exec(
				"UPDATE galgame SET release_date = ?::date WHERE id = ?",
				datedVals[i], id,
			).Error; err != nil {
				return err
			}
		}
		if len(nullIDs) > 0 {
			if err := tx.Exec(
				"UPDATE galgame SET release_date = NULL WHERE id IN ?", nullIDs,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func main() {
	_ = godotenv.Load()

	dryRun := flag.Bool("dry-run", false, "Fetch from wiki but do not update rows")
	// 16 is the safe default: the wiki detail endpoint is heavy (relation
	// preloads), and ~48 concurrent saturated a local wiki into 10s client
	// timeouts. The keep-alive pool (MaxIdleConnsPerHost=64) keeps these
	// 16 reusing connections. Bump cautiously if the wiki has headroom.
	workers := flag.Int("workers", 16, "Concurrent detail fetches")
	writeChunk := flag.Int("chunk", 500, "Rows per bulk UPDATE")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)

	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)
	gc := galgameClient.New(
		cfg.NextMoeAPI.BaseURL,
		cfg.NextMoeAPI.APIKey,
		cfg.NextMoeAPI.ImageCDNBase,
	)

	ctx := context.Background()

	// Load every local galgame id up-front (the table is kungal's curated
	// subset — small enough to hold in memory).
	var allIDs []int
	if err := db.Table("galgame").Order("id ASC").Pluck("id", &allIDs).Error; err != nil {
		slog.Error("拉取本地 galgame id 失败", "error", err)
		os.Exit(1)
	}
	slog.Info("开始回填 release_date",
		"total", len(allIDs), "workers", *workers, "dry_run", *dryRun)

	// Fan out detail fetches across a worker pool.
	idCh := make(chan int)
	resCh := make(chan detailResult, *workers)
	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range idCh {
				resCh <- fetchReleaseDate(ctx, gc, id)
			}
		}()
	}
	go func() {
		for _, id := range allIDs {
			idCh <- id
		}
		close(idCh)
	}()
	go func() {
		wg.Wait()
		close(resCh)
	}()

	// Collect results, then flush in bulk chunks.
	withDate, nullDate, missing := 0, 0, 0
	datedIDs := make([]int, 0, *writeChunk)
	datedVals := make([]string, 0, *writeChunk)
	nullIDs := make([]int, 0, *writeChunk)

	flush := func() {
		if *dryRun || (len(datedIDs) == 0 && len(nullIDs) == 0) {
			datedIDs, datedVals, nullIDs = datedIDs[:0], datedVals[:0], nullIDs[:0]
			return
		}
		if err := writeBatch(db, datedIDs, datedVals, nullIDs); err != nil {
			slog.Error("批次写入失败", "error", err)
			os.Exit(1)
		}
		datedIDs, datedVals, nullIDs = datedIDs[:0], datedVals[:0], nullIDs[:0]
	}

	processed := 0
	for r := range resCh {
		processed++
		switch {
		case r.missing:
			missing++
		case r.date != nil && *r.date != "":
			withDate++
			datedIDs = append(datedIDs, r.id)
			datedVals = append(datedVals, *r.date)
		default:
			nullDate++
			nullIDs = append(nullIDs, r.id)
		}

		if len(datedIDs) >= *writeChunk || len(nullIDs) >= *writeChunk {
			flush()
		}
		if processed%1000 == 0 {
			slog.Info("进度", "processed", processed, "with_date", withDate,
				"null_date", nullDate, "missing", missing)
		}
	}
	flush()

	verb := "回填"
	if *dryRun {
		verb = "dry-run"
	}
	fmt.Printf(
		"%s 完成: 共 %d 行 (有日期 %d, 无日期/NULL %d, wiki 无此条 %d)\n",
		verb, processed, withDate, nullDate, missing,
	)
}
