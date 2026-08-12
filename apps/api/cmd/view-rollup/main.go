package main

import (
	"log/slog"
	"os"

	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/internal/infrastructure/viewstats"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)

	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)

	if err := viewstats.RunRollup(db); err != nil {
		slog.Error("浏览量滚动统计失败", "error", err)
		os.Exit(1)
	}
	slog.Info("浏览量滚动统计完成 (view_7d / view_30d 已刷新, 旧桶已清理)")
}
