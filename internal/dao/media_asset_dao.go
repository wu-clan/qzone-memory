package dao

import (
	"context"
	"errors"

	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BatchUpsertMediaAssets 批量登记媒体资源，按 (user_qq, url_hash) 去重。
// 冲突时只刷新来源信息，不覆盖已下载结果（local_path / status / bytes 等保持不变）。
func BatchUpsertMediaAssets(items []*model.MediaAsset) error {
	if len(items) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_qq"}, {Name: "url_hash"}},
		DoUpdates: clause.AssignmentColumns([]string{"source_url", "category", "updated_at"}),
	}).Create(&items).Error
}

// GetMediaAsset 按 (user_qq, url_hash) 取单条记录。
func GetMediaAsset(ctx context.Context, userQQ, urlHash string) (*model.MediaAsset, error) {
	var item model.MediaAsset
	err := database.DB.WithContext(ctx).
		Where("user_qq = ? AND url_hash = ?", userQQ, urlHash).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

// ListMediaAssetsByStatus 取某状态下的媒体资源，供下载器消费待处理队列。limit<=0 表示不限。
func ListMediaAssetsByStatus(ctx context.Context, userQQ, status string, limit int) ([]*model.MediaAsset, error) {
	var items []*model.MediaAsset
	query := database.DB.WithContext(ctx).Where("user_qq = ? AND status = ?", userQQ, status)
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Order("id ASC").Find(&items).Error
	return items, err
}

// MarkMediaAsset 更新一条媒体资源的下载结果（status / local_path / mime_type / bytes / error 等）。
func MarkMediaAsset(ctx context.Context, id uint, fields map[string]any) error {
	return database.DB.WithContext(ctx).Model(&model.MediaAsset{}).
		Where("id = ?", id).Updates(fields).Error
}

// CountMediaAssetsByStatus 统计某 QQ 各状态的媒体数量（done / failed / pending …）。
func CountMediaAssetsByStatus(ctx context.Context, userQQ string) (map[string]int64, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	err := database.DB.WithContext(ctx).Model(&model.MediaAsset{}).
		Select("status, count(*) as count").
		Where("user_qq = ?", userQQ).Group("status").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, r := range rows {
		result[r.Status] = r.Count
	}
	return result, nil
}

// SumMediaAssetBytes 统计某 QQ 已下载媒体占用的磁盘字节数。
func SumMediaAssetBytes(ctx context.Context, userQQ string) (int64, error) {
	var total int64
	err := database.DB.WithContext(ctx).Model(&model.MediaAsset{}).
		Where("user_qq = ? AND status = ?", userQQ, "done").
		Select("COALESCE(SUM(bytes), 0)").Scan(&total).Error
	return total, err
}

// DeleteMediaAssetsByQQ 删除某 QQ 的全部媒体登记，用于「彻底删除我的数据」。
func DeleteMediaAssetsByQQ(ctx context.Context, userQQ string) error {
	return database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Delete(&model.MediaAsset{}).Error
}

// ResetFailedMediaAssets 把失败的媒体重置为待处理，用于「重新下载失败项」。
func ResetFailedMediaAssets(ctx context.Context, userQQ string) error {
	return database.DB.WithContext(ctx).Model(&model.MediaAsset{}).
		Where("user_qq = ? AND status = ?", userQQ, "failed").
		Updates(map[string]any{"status": "pending", "error": ""}).Error
}
