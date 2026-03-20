package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qzone-memory/config"
	"github.com/qzone-memory/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init 初始化数据库
func Init() error {
	cfg := config.GlobalConfig.Database

	// 确保数据目录存在
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}

	// 打开数据库连接
	db, err := gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{
		Logger: logger.Default.LogMode(parseGormLogLevel(cfg.LogLevel)),
	})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// 设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConn)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConn)

	DB = db

	// 自动迁移表结构
	if err := autoMigrate(); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	return nil
}

func parseGormLogLevel(level string) logger.LogLevel {
	switch level {
	case "info":
		return logger.Info
	case "warn":
		return logger.Warn
	case "error":
		return logger.Error
	case "silent":
		fallthrough
	default:
		return logger.Silent
	}
}

// autoMigrate 自动迁移所有表
func autoMigrate() error {
	return DB.AutoMigrate(
		&model.User{},
		&model.Activity{},
		&model.FriendGroup{},
		&model.Friend{},
		&model.Visitor{},
		&model.Video{},
		&model.Favorite{},
		&model.Diary{},
		&model.Talk{},
		&model.Blog{},
		&model.Album{},
		&model.Photo{},
		&model.Message{},
		&model.Comment{},
		&model.Like{},
		&model.Share{},
		&model.Mention{},
	)
}

// Close 关闭数据库连接
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
