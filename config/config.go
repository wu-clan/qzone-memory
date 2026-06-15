package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config 全局配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	QZone    QZoneConfig    `mapstructure:"qzone"`
	Media    MediaConfig    `mapstructure:"media"`
	Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug, release, test
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path        string `mapstructure:"path"`
	MaxIdleConn int    `mapstructure:"max_idle_conn"`
	MaxOpenConn int    `mapstructure:"max_open_conn"`
	LogLevel    string `mapstructure:"log_level"` // silent, error, warn, info
}

// QZoneConfig QQ 空间配置
type QZoneConfig struct {
	LoginTimeout      int `mapstructure:"login_timeout"`
	RequestIntervalMs int `mapstructure:"request_interval_ms"` // 全局最小请求间隔（毫秒），抓取与下载共用
	MaxConcurrency    int `mapstructure:"max_concurrency"`     // 出站最大并发
	JitterMs          int `mapstructure:"jitter_ms"`           // 随机抖动上限（毫秒）
}

// MediaConfig 媒体本地化配置
type MediaConfig struct {
	Dir                 string `mapstructure:"dir"`                  // 媒体落盘目录
	DownloadConcurrency int    `mapstructure:"download_concurrency"` // 媒体下载并发
	MaxRetries          int    `mapstructure:"max_retries"`          // 单个媒体最大重试
	DownloadVideos      bool   `mapstructure:"download_videos"`      // 视频原片开关（默认关）
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`       // debug, info, warn, error
	Format     string `mapstructure:"format"`      // json, console
	OutputPath string `mapstructure:"output_path"` // 兼容旧配置：完整日志输出路径
	OutputDir  string `mapstructure:"output_dir"`  // 日志目录
	FileName   string `mapstructure:"file_name"`   // 日志文件名
	MaxSize    int    `mapstructure:"max_size"`    // 单个日志文件最大尺寸（MB）
	MaxAge     int    `mapstructure:"max_age"`     // 日志保留天数
	MaxBackups int    `mapstructure:"max_backups"` // 最多保留文件个数
}

var GlobalConfig *Config

// Load 加载配置：优先读取 config/config.yaml；文件缺失时使用内置默认值（便于
// 单个二进制双击即用）。无论是否存在，都会用默认值补齐缺省项。
func Load() error {
	cfg := &Config{}
	path := filepath.Join("config", "config.yaml")

	if _, statErr := os.Stat(path); statErr != nil {
		applyDefaults(cfg)
		GlobalConfig = cfg
		return nil
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}
	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	applyDefaults(cfg)
	GlobalConfig = cfg
	return nil
}

// applyDefaults 为缺省项补齐合理默认值。
func applyDefaults(c *Config) {
	if c.Server.Host == "" {
		c.Server.Host = "127.0.0.1"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8081
	}
	if c.Server.Mode == "" {
		c.Server.Mode = "release"
	}
	if c.Database.Path == "" {
		c.Database.Path = "./data/qzone.db"
	}
	if c.Database.MaxIdleConn == 0 {
		c.Database.MaxIdleConn = 10
	}
	if c.Database.MaxOpenConn == 0 {
		c.Database.MaxOpenConn = 100
	}
	if c.Database.LogLevel == "" {
		c.Database.LogLevel = "silent"
	}
	if c.QZone.LoginTimeout == 0 {
		c.QZone.LoginTimeout = 300
	}
	if c.QZone.RequestIntervalMs == 0 {
		c.QZone.RequestIntervalMs = 800
	}
	if c.QZone.MaxConcurrency == 0 {
		c.QZone.MaxConcurrency = 3
	}
	if c.QZone.JitterMs == 0 {
		c.QZone.JitterMs = 600
	}
	if c.Media.Dir == "" {
		c.Media.Dir = "./data/media"
	}
	if c.Media.DownloadConcurrency == 0 {
		c.Media.DownloadConcurrency = 3
	}
	if c.Media.MaxRetries == 0 {
		c.Media.MaxRetries = 3
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Format == "" {
		c.Log.Format = "console"
	}
	if c.Log.OutputDir == "" {
		c.Log.OutputDir = "./logs"
	}
	if c.Log.FileName == "" {
		c.Log.FileName = "app.log"
	}
	if c.Log.MaxSize == 0 {
		c.Log.MaxSize = 100
	}
	if c.Log.MaxAge == 0 {
		c.Log.MaxAge = 30
	}
	if c.Log.MaxBackups == 0 {
		c.Log.MaxBackups = 10
	}
}
