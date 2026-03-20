package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/qzone-memory/api"
	"github.com/qzone-memory/config"
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// 加载配置
	if err := config.Load(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志
	if err := logger.Init(); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Sync()

	// 初始化数据库
	if err := database.Init(); err != nil {
		logger.Fatal("初始化数据库失败")
	}
	defer database.Close()

	// 初始化路由
	cfg := config.GlobalConfig
	router := api.RegisterRoutes(cfg.Server.Mode)

	// 启动 HTTP 服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP 服务启动失败", zap.Error(err))
		}
	}()

	fmt.Println()
	fmt.Println("  💫 QQ 空间回忆")
	fmt.Println("  ─────────────────────────")
	fmt.Printf("  🌐 访问地址: http://%s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Println("  📝 扫码登录后即可查看回忆")
	fmt.Println("  ─────────────────────────")
	fmt.Println()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	defer signal.Stop(quit)
	<-quit
	log.Println("关闭 HTTP 服务 ...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Fatal("HTTP 服务关闭失败", zap.Error(err))
	}
	log.Println("HTTP 服务已关闭")
}
