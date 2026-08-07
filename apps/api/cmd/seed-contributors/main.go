// Seed galgame_contributor (migration 069) from the frozen wiki contributor
// ledger — the 17,966-row TSV cut before the wiki tables were dropped, which is
// the only surviving record of who edited what in the wiki era.
//
// Insert-only in spirit: an existing pair keeps its revision_count and its
// source, and only moves first_at EARLIER. The forward revision sync owns the
// counting, so a pair this seed meets again must not have its history rewritten
// by a re-run — which is why a second --apply reports zero inserts and is the
// acceptance check for the run.
//
//	seed-contributors --file wiki-contributors.tsv            # dry run (default)
//	seed-contributors --file wiki-contributors.tsv --apply     # write
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

// chunkSize bounds one statement's VALUES list. Large enough that 18k rows are
// a few dozen round-trips, small enough to stay well under the parameter cap.
const chunkSize = 500

func main() {
	_ = godotenv.Load()

	file := flag.String("file", "", "Path to the frozen wiki contributor TSV (galgame_id, catalog_work_id, user_id, created)")
	apply := flag.Bool("apply", false, "Write to the database (default: report only)")
	flag.Parse()

	if *file == "" {
		slog.Error("必须指定 --file")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)

	f, err := os.Open(*file)
	if err != nil {
		slog.Error("打开 TSV 失败", "file", *file, "error", err)
		os.Exit(1)
	}
	defer f.Close()

	rows, stats, err := parseContributorTSV(f)
	if err != nil {
		slog.Error("读取 TSV 失败", "file", *file, "error", err)
		os.Exit(1)
	}
	slog.Info("TSV 解析完成", "lines", stats.Lines, "pairs", len(rows),
		"skipped", stats.Skipped, "folded", stats.Folded)

	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)

	if !*apply {
		present, err := countPresent(db, rows)
		if err != nil {
			slog.Error("统计已存在的贡献者行失败", "error", err)
			os.Exit(1)
		}
		fmt.Printf("dry-run: 待写入 %d 对, 其中已存在 %d, 将新增 %d (加 --apply 执行)\n",
			len(rows), present, len(rows)-present)
		return
	}

	inserted, updated, err := seed(db, rows)
	if err != nil {
		slog.Error("写入贡献者失败", "inserted", inserted, "error", err)
		os.Exit(1)
	}
	gids, err := refreshCounts(db, rows)
	if err != nil {
		slog.Error("刷新 contributor_count 失败", "error", err)
		os.Exit(1)
	}
	fmt.Printf("seed 完成: 新增 %d, 已存在 %d, 刷新 contributor_count 的 galgame %d 个\n",
		inserted, updated, gids)
}

// countPresent reports how many of the parsed pairs the table already holds —
// the dry run's whole content, and the number a second run expects to equal the
// total.
func countPresent(db *gorm.DB, rows []seedRow) (int, error) {
	present := 0
	for _, chunk := range chunks(rows) {
		args := make([]any, 0, len(chunk)*2)
		values := make([]string, 0, len(chunk))
		for _, r := range chunk {
			values = append(values, "(?::bigint, ?::bigint)")
			args = append(args, r.GalgameID, r.UserID)
		}
		var n int64
		if err := db.Raw(`
			SELECT COUNT(*) FROM galgame_contributor c
			JOIN (VALUES `+strings.Join(values, ", ")+`) AS v(galgame_id, user_id)
			  ON c.galgame_id = v.galgame_id AND c.user_id = v.user_id`,
			args...).Scan(&n).Error; err != nil {
			return present, err
		}
		present += int(n)
	}
	return present, nil
}

// seed writes the ledger. `xmax = 0` on the RETURNING row distinguishes an
// INSERT from a conflict-update, which is what makes "a second run inserts
// nothing" checkable rather than assumed.
func seed(db *gorm.DB, rows []seedRow) (inserted, updated int, err error) {
	for _, chunk := range chunks(rows) {
		args := make([]any, 0, len(chunk)*3)
		values := make([]string, 0, len(chunk))
		for _, r := range chunk {
			values = append(values, "(?::bigint, ?::bigint, ?::timestamptz, ?::timestamptz, 0, 0)")
			args = append(args, r.GalgameID, r.UserID, r.Created, r.Created)
		}
		var flags []bool
		if err := db.Raw(`
			INSERT INTO galgame_contributor
				(galgame_id, user_id, first_at, last_at, revision_count, source)
			VALUES `+strings.Join(values, ", ")+`
			ON CONFLICT (galgame_id, user_id) DO UPDATE SET
				first_at = LEAST(galgame_contributor.first_at, excluded.first_at)
			RETURNING (xmax = 0) AS inserted`,
			args...).Scan(&flags).Error; err != nil {
			return inserted, updated, err
		}
		for _, isInsert := range flags {
			if isInsert {
				inserted++
				continue
			}
			updated++
		}
	}
	return inserted, updated, nil
}

// refreshCounts recomputes galgame.contributor_count for every gid the seed
// touched. Derived from the table, so it is the same statement the forward sync
// runs and the two can never disagree.
func refreshCounts(db *gorm.DB, rows []seedRow) (int, error) {
	seen := map[int64]bool{}
	gids := make([]int64, 0, len(rows))
	for _, r := range rows {
		if !seen[r.GalgameID] {
			seen[r.GalgameID] = true
			gids = append(gids, r.GalgameID)
		}
	}
	for i := 0; i < len(gids); i += chunkSize {
		end := min(i+chunkSize, len(gids))
		if err := db.Exec(`
			UPDATE galgame SET contributor_count = (
				SELECT COUNT(*) FROM galgame_contributor c WHERE c.galgame_id = galgame.id
			) WHERE id IN ?`, gids[i:end]).Error; err != nil {
			return 0, err
		}
	}
	return len(gids), nil
}

func chunks(rows []seedRow) [][]seedRow {
	out := make([][]seedRow, 0, len(rows)/chunkSize+1)
	for i := 0; i < len(rows); i += chunkSize {
		out = append(out, rows[i:min(i+chunkSize, len(rows))])
	}
	return out
}
