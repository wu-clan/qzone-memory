package dao

import (
	"context"
	"errors"
	"time"

	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UpsertSyncState 写入/更新某 QQ 某类型的同步状态，按 (user_qq, type) 唯一。
func UpsertSyncState(ctx context.Context, state *model.SyncState) error {
	return database.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_qq"}, {Name: "type"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "last_offset", "cursor", "items_synced", "last_synced_at", "error", "updated_at"}),
	}).Create(state).Error
}

// GetSyncState 取某 QQ 某类型的同步状态。
func GetSyncState(ctx context.Context, userQQ, syncType string) (*model.SyncState, error) {
	var item model.SyncState
	err := database.DB.WithContext(ctx).
		Where("user_qq = ? AND type = ?", userQQ, syncType).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

// ListSyncStates 列出某 QQ 的全部同步状态，用于重建同步进度。
func ListSyncStates(ctx context.Context, userQQ string) ([]*model.SyncState, error) {
	var items []*model.SyncState
	err := database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Find(&items).Error
	return items, err
}

// SetSyncStateStatus 仅更新某 QQ 某类型的状态与错误信息。
func SetSyncStateStatus(ctx context.Context, userQQ, syncType, status, errMsg string) error {
	return database.DB.WithContext(ctx).Model(&model.SyncState{}).
		Where("user_qq = ? AND type = ?", userQQ, syncType).
		Updates(map[string]any{"status": status, "error": errMsg, "last_synced_at": time.Now()}).Error
}

// DeleteSyncStatesByQQ 删除某 QQ 的全部同步状态。
func DeleteSyncStatesByQQ(ctx context.Context, userQQ string) error {
	return database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Delete(&model.SyncState{}).Error
}
