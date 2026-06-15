package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"github.com/qzone-memory/api"
	"github.com/qzone-memory/config"
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/pkg/logger"
	"github.com/qzone-memory/pkg/ratelimit"
	"go.uber.org/zap"
)

func main() {
	// 让配置 / 数据 / 日志相对可执行文件定位，支持双击即用
	ensureWorkingDir()

	// 加载配置
	if err := config.Load(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志
	if err := logger.Init(); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Sync()

	// 初始化出站限速器（抓取与媒体下载共用同一闸门）
	qz := config.GlobalConfig.QZone
	ratelimit.Configure(
		time.Duration(qz.RequestIntervalMs)*time.Millisecond,
		time.Duration(qz.JitterMs)*time.Millisecond,
		qz.MaxConcurrency,
	)

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

	// 启动后自动打开浏览器（设置环境变量 QZONE_NO_BROWSER=1 可关闭）
	if os.Getenv("QZONE_NO_BROWSER") == "" {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openBrowser(fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port))
		}()
	}

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

// ensureWorkingDir 若当前目录不含配置/数据，则切换到可执行文件所在目录，
// 让双击运行（工作目录可能是 / 或用户主目录）时仍能在程序旁边读写数据。
func ensureWorkingDir() {
	if pathExists("config/config.yaml") || pathExists("data") {
		return
	}
	if exe, err := os.Executable(); err == nil {
		_ = os.Chdir(filepath.Dir(exe))
	}
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// openBrowser 用系统默认浏览器打开 URL（跨平台）。
func openBrowser(rawURL string) {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name, args = "open", []string{rawURL}
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler", rawURL}
	default:
		name, args = "xdg-open", []string{rawURL}
	}
	_ = exec.Command(name, args...).Start()
}
