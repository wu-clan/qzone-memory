package dao

import (
	"context"

	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
)

// CreateSyncLog 写入一条同步流水日志
func CreateSyncLog(ctx context.Context, item *model.SyncLog) error {
	return database.DB.WithContext(ctx).Create(item).Error
}

// BatchCreateSyncLogs 批量写入同步流水日志
func BatchCreateSyncLogs(ctx context.Context, items []*model.SyncLog) error {
	if len(items) == 0 {
		return nil
	}
	return database.DB.WithContext(ctx).Create(&items).Error
}

// ListSyncLogs 查询某次或某用户的同步流水，供后续分析和排障使用
func ListSyncLogs(ctx context.Context, userQQ, runID string, offset, limit int) ([]*model.SyncLog, int64, error) {
	var logs []*model.SyncLog
	var total int64

	query := database.DB.WithContext(ctx).Model(&model.SyncLog{})
	if userQQ != "" {
		query = query.Where("user_qq = ?", userQQ)
	}
	if runID != "" {
		query = query.Where("run_id = ?", runID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}
	err := query.Order("created_at ASC, id ASC").Find(&logs).Error
	return logs, total, err
}

// DeleteSyncLogsByQQ 删除某 QQ 的同步流水日志
func DeleteSyncLogsByQQ(ctx context.Context, userQQ string) error {
	return database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Delete(&model.SyncLog{}).Error
}
