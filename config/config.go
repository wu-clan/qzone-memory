package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config 全局配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	QZone    QZoneConfig    `mapstructure:"qzone"`
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
	LoginTimeout int `mapstructure:"login_timeout"`
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

// Load 加载配置文件
func Load() error {
	v := viper.New()
	v.SetConfigFile(filepath.Join("config", "config.yaml"))
	v.SetConfigType("yaml")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置
	if err := v.Unmarshal(&GlobalConfig); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	return nil
}
