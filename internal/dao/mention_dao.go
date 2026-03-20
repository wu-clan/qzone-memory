package dao

import (
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm/clause"
)

func ListMentions(userQQ string, offset, limit int) ([]*model.Mention, int64, error) {
	var mentions []*model.Mention
	var total int64

	query := database.DB.Model(&model.Mention{}).Where("user_qq = ?", userQQ)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("mention_time DESC").Offset(offset).Limit(limit).Find(&mentions).Error
	return mentions, total, err
}

func BatchUpsertMentions(mentions []*model.Mention) error {
	if len(mentions) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "mention_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"content", "updated_at"}),
	}).Create(&mentions).Error
}
