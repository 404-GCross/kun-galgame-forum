package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"regexp"

	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
)

var contentImageTokenRe = regexp.MustCompile(`/image/[0-9a-f]{64}`)

func main() {
	_ = godotenv.Load()

	dryRun := flag.Bool("dry-run", true, "TRUE (default): report workload only, no DB writes. Pass -dry-run=false to apply.")
	maxCovers := flag.Int("max", 3, "Max covers to take per topic (the first N distinct token-images in body order; capped at 9)")
	limit := flag.Int("limit", 0, "Max topics to process (0 = all); for smoke-testing -dry-run=false on a small batch")
	flag.Parse()

	if *maxCovers < 1 || *maxCovers > 9 {
		slog.Error("max 必须在 1..9 之间", "max", *maxCovers)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)

	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)

	type row struct {
		ID      int64
		Content string
	}
	var rows []row
	q := db.Table("topic").
		Select("id, content").
		Where("cover_images = ''").
		Where("content LIKE ?", "%/image/%").
		Order("id ASC")
	if *limit > 0 {
		q = q.Limit(*limit)
	}
	if err := q.Scan(&rows).Error; err != nil {
		slog.Error("扫描话题失败", "error", err)
		os.Exit(1)
	}

	slog.Info("开始回填话题封面", "dry_run", *dryRun, "max", *maxCovers, "limit", *limit, "候选话题数", len(rows))

	var wouldFill, filled int
	countDist := map[int]int{}

	for _, r := range rows {
		covers := firstDistinctTokens(r.Content, *maxCovers)
		if len(covers) == 0 {
			continue
		}
		wouldFill++
		countDist[len(covers)]++

		if *dryRun {
			continue
		}

		payload, merr := json.Marshal(covers)
		if merr != nil {
			slog.Error("序列化封面失败, 跳过", "topic_id", r.ID, "error", merr)
			continue
		}
		if err := db.Exec(
			"UPDATE topic SET cover_images = ? WHERE id = ?",
			string(payload), r.ID,
		).Error; err != nil {
			slog.Error("更新话题封面失败", "topic_id", r.ID, "error", err)
			continue
		}
		filled++
		slog.Info("已回填封面", "topic_id", r.ID, "封面数", len(covers))
	}

	if *dryRun {
		fmt.Printf("dry-run 完成。可回填话题 %d / 候选 %d。各封面数分布: ", wouldFill, len(rows))
		for n := 1; n <= *maxCovers; n++ {
			if countDist[n] > 0 {
				fmt.Printf("%d张=%d ", n, countDist[n])
			}
		}
		fmt.Printf("\n加 -dry-run=false 执行。\n")
		return
	}

	slog.Info("回填完成", "已回填话题数", filled, "可回填", wouldFill, "候选", len(rows))
	fmt.Printf("回填完成: 已为 %d 个话题写入封面 (候选 %d)。\n", filled, len(rows))
}

func firstDistinctTokens(content string, max int) []string {
	matches := contentImageTokenRe.FindAllString(content, -1)
	seen := make(map[string]struct{}, len(matches))
	out := make([]string, 0, max)
	for _, tk := range matches {
		if _, dup := seen[tk]; dup {
			continue
		}
		seen[tk] = struct{}{}
		out = append(out, tk)
		if len(out) >= max {
			break
		}
	}
	return out
}
