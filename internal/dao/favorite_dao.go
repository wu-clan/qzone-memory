package dao

import (
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm/clause"
)

func ListFavorites(userQQ string, offset, limit int) ([]*model.Favorite, int64, error) {
	var items []*model.Favorite
	var total int64
	query := database.DB.Model(&model.Favorite{}).Where("user_qq = ?", userQQ)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("create_time DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func BatchUpsertFavorites(items []*model.Favorite) error {
	if len(items) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "favorite_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_qq", "type", "title", "abstract", "url", "owner_qq", "owner_name", "images", "create_time", "updated_at"}),
	}).Create(&items).Error
}
