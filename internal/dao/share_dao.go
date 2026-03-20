package dao

import (
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm/clause"
)

func ListSharesByTarget(targetType, targetID string, offset, limit int) ([]*model.Share, int64, error) {
	var shares []*model.Share
	var total int64

	query := database.DB.Model(&model.Share{}).Where("target_type = ? AND target_id = ?", targetType, targetID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("share_time DESC").Offset(offset).Limit(limit).Find(&shares).Error
	return shares, total, err
}

func BatchUpsertShares(shares []*model.Share) error {
	if len(shares) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "share_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"sharer_qq", "sharer_name", "comment", "share_time", "updated_at"}),
	}).Create(&shares).Error
}
