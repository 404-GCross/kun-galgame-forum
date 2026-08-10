package main

import (
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"kun-galgame-api/internal/app"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/health"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "2334"
	}
	port, _ := strconv.Atoi(serverPort)
	health.MaybeProbe(port, "/healthz")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	logger.Init(cfg.Server.Mode)

	application := app.New(cfg)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		slog.Info("正在关闭服务器...")
		_ = application.Fiber.Shutdown()
	}()

	addr := ":" + cfg.Server.Port
	slog.Info("服务器启动", "addr", addr)
	if err := application.Fiber.Listen(addr); err != nil {
		slog.Error("服务器启动失败", "error", err)
		os.Exit(1)
	}
}
