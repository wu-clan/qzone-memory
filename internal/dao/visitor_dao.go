package dao

import (
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm/clause"
)

func ListVisitors(userQQ string, offset, limit int) ([]*model.Visitor, int64, error) {
	var items []*model.Visitor
	var total int64
	query := database.DB.Model(&model.Visitor{}).Where("user_qq = ?", userQQ)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("visit_time DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func BatchUpsertVisitors(items []*model.Visitor) error {
	if len(items) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "visitor_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_qq", "visitor_qq", "visitor_name", "avatar", "source", "is_hidden", "yellow_level", "visit_time", "updated_at"}),
	}).Create(&items).Error
}
