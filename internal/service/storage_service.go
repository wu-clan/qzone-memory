package service

import (
	"context"
	"path/filepath"

	"github.com/qzone-memory/config"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/media"
	"github.com/qzone-memory/pkg/logger"
	"go.uber.org/zap"
)

// StorageStats 「数据归我」面板所需的本地存储统计。
type StorageStats struct {
	DataDir       string           `json:"data_dir"`
	DBPath        string           `json:"db_path"`
	MediaDir      string           `json:"media_dir"`
	MediaByStatus map[string]int64 `json:"media_by_status"`
	MediaBytes    int64            `json:"media_bytes"`
}

// GetStorageStats 汇总某 QQ 的数据位置与媒体本地化情况。
func GetStorageStats(ctx context.Context, qq string) (*StorageStats, error) {
	if err := validateQQ(qq); err != nil {
		return nil, err
	}
	byStatus, err := dao.CountMediaAssetsByStatus(ctx, qq)
	if err != nil {
		return nil, err
	}
	bytes, _ := dao.SumMediaAssetBytes(ctx, qq)

	stats := &StorageStats{
		MediaByStatus: byStatus,
		MediaBytes:    bytes,
		DBPath:        config.GlobalConfig.Database.Path,
		MediaDir:      config.GlobalConfig.Media.Dir,
	}
	if abs, err := filepath.Abs(config.GlobalConfig.Database.Path); err == nil {
		stats.DBPath = abs
		stats.DataDir = filepath.Dir(abs)
	}
	if abs, err := filepath.Abs(config.GlobalConfig.Media.Dir); err == nil {
		stats.MediaDir = abs
	}
	return stats, nil
}

// BackfillMedia 为某 QQ 登记并下载尚未本地化的媒体（存量回填 / 重新下载失败项），后台执行。
func BackfillMedia(qq string) error {
	if err := validateQQ(qq); err != nil {
		return err
	}
	go func() {
		ctx := context.Background()
		if err := dao.ResetFailedMediaAssets(ctx, qq); err != nil {
			logger.Warn("重置失败媒体出错", zap.String("qq", qq), zap.Error(err))
		}
		if _, err := media.EnqueueAll(ctx, qq); err != nil {
			logger.Warn("回填登记媒体失败", zap.String("qq", qq), zap.Error(err))
		}
		done, failed, _ := media.DrainPending(ctx, qq)
		logger.Info("媒体回填完成", zap.String("qq", qq), zap.Int("done", done), zap.Int("failed", failed))
	}()
	return nil
}
