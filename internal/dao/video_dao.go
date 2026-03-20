package dao

import (
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm/clause"
)

func ListVideos(userQQ string, offset, limit int) ([]*model.Video, int64, error) {
	var items []*model.Video
	var total int64
	query := database.DB.Model(&model.Video{}).Where("user_qq = ?", userQQ)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("upload_time DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func BatchUpsertVideos(items []*model.Video) error {
	if len(items) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "video_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_qq", "title", "description", "url", "preview_url", "width", "height", "duration", "comment_count", "upload_time", "updated_at"}),
	}).Create(&items).Error
}
