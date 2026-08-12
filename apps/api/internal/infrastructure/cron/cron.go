package cron

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/internal/infrastructure/viewstats"
	"kun-galgame-api/pkg/imageclient"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

const scheduleTZ = "Asia/Shanghai"

func Start(
	db *gorm.DB,
	rdb *redis.Client,
	imgCli *imageclient.Client,
	galgameClaimSync func(),
	galgameRevisionSync func(),
	galgameContributorSync func(),
) func() {
	loc, err := time.LoadLocation(scheduleTZ)
	if err != nil {
		slog.Warn("加载定时任务时区失败, 回退到进程本地时区", "tz", scheduleTZ, "error", err)
		loc = time.Local
	}
	c := cron.New(cron.WithLocation(loc))

	c.AddFunc("0 0 * * *", func() {
		resetDaily(db)
	})

	c.AddFunc("0 0 * * *", func() {
		if err := viewstats.RunRollup(db); err != nil {
			slog.Error("浏览量滚动统计失败", "error", err)
			return
		}
		slog.Info("浏览量滚动统计完成")
	})

	c.AddFunc("0 * * * *", func() {
		cleanupUploadCache(rdb)
	})

	if imgCli != nil {
		c.AddFunc("0 4 * * *", func() {
			distinct, updated, err := RunReferencePing(context.Background(), db, imgCli)
			if err != nil {
				slog.Error("内容图 reference-ping 失败", "distinct", distinct, "updated", updated, "error", err)
				return
			}
			slog.Info("内容图 reference-ping 完成", "distinct_hashes", distinct, "updated", updated)
		})
	} else {
		slog.Warn("image client 未配置, 跳过内容图 reference-ping —— 内容图存在被 image-gc 回收的风险")
	}

	if galgameClaimSync != nil {
		c.AddFunc("*/10 * * * *", galgameClaimSync)
	}

	if galgameRevisionSync != nil {
		c.AddFunc("*/10 * * * *", galgameRevisionSync)
	}

	if galgameContributorSync != nil {
		c.AddFunc("*/15 * * * *", galgameContributorSync)
	}

	c.Start()
	slog.Info("定时任务已启动")

	return func() {
		ctx := c.Stop()
		<-ctx.Done()
		slog.Info("定时任务已停止")
	}
}

// Targets `kungal_user_state`, NOT the old `"user"` table — migration 007 moved
// the daily_* columns. The original `UPDATE "user" SET daily_* = 0` silently
// errored every midnight after that migration, so users who hit their daily
// caps stayed capped indefinitely.
func resetDaily(db *gorm.DB) {
	result := db.Exec(`
		UPDATE kungal_user_state SET
			daily_check_in = 0,
			daily_image_count = 0,
			daily_toolset_upload_count = 0,
			daily_toolset_upload_bytes = 0
		WHERE daily_check_in != 0
		   OR daily_image_count != 0
		   OR daily_toolset_upload_count != 0
		   OR daily_toolset_upload_bytes != 0
	`)
	if result.Error != nil {
		slog.Error("每日重置失败", "error", result.Error)
		return
	}
	slog.Info("每日重置完成", "affected", result.RowsAffected)
}

func cleanupUploadCache(rdb *redis.Client) {
	ctx := context.Background()
	keys, err := rdb.Keys(ctx, "toolset:upload:*").Result()
	if err != nil {
		slog.Error("扫描上传缓存失败", "error", err)
		return
	}

	if len(keys) == 0 {
		return
	}

	deleted := 0
	for _, key := range keys {
		ttl, _ := rdb.TTL(ctx, key).Result()
		if ttl <= 0 {
			rdb.Del(ctx, key)
			deleted++
		}
	}

	if deleted > 0 {
		slog.Info("清理上传缓存完成", "deleted", deleted)
	}
}
