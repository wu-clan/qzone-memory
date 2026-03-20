package dao

import (
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm/clause"
)

func ListLikesByTarget(targetType, targetID string, offset, limit int) ([]*model.Like, int64, error) {
	var likes []*model.Like
	var total int64

	query := database.DB.Model(&model.Like{}).Where("target_type = ? AND target_id = ?", targetType, targetID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("like_time DESC").Offset(offset).Limit(limit).Find(&likes).Error
	return likes, total, err
}

func BatchUpsertLikes(likes []*model.Like) error {
	if len(likes) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "like_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"liker_qq", "liker_name", "liker_avatar", "like_time", "updated_at"}),
	}).Create(&likes).Error
}
