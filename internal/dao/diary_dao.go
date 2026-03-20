package dao

import (
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm/clause"
)

func ListDiaries(userQQ string, offset, limit int) ([]*model.Diary, int64, error) {
	var items []*model.Diary
	var total int64
	query := database.DB.Model(&model.Diary{}).Where("user_qq = ?", userQQ)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("create_time DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func BatchUpsertDiaries(items []*model.Diary) error {
	if len(items) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "diary_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_qq", "title", "summary", "content", "create_time", "updated_at"}),
	}).Create(&items).Error
}
