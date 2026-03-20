package dao

import (
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm/clause"
)

func GetActivityByFeedID(feedID string) (*model.Activity, error) {
	var activity model.Activity
	err := database.DB.Where("feed_id = ?", feedID).First(&activity).Error
	return &activity, err
}

func ListActivities(userQQ, feedType string, offset, limit int) ([]*model.Activity, int64, error) {
	var activities []*model.Activity
	var total int64

	query := database.DB.Model(&model.Activity{}).Where("user_qq = ?", userQQ)
	if feedType != "" {
		query = query.Where("feed_type = ?", feedType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("publish_time DESC, id DESC").Offset(offset).Limit(limit).Find(&activities).Error
	return activities, total, err
}

func BatchUpsertActivities(activities []*model.Activity) error {
	if len(activities) == 0 {
		return nil
	}

	return database.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "feed_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"user_qq",
			"feed_type",
			"object_id",
			"title",
			"content",
			"html_content",
			"author_qq",
			"author_name",
			"images",
			"like_count",
			"comment_count",
			"share_count",
			"is_deleted",
			"publish_time",
			"state_text",
			"updated_at",
		}),
	}).Create(&activities).Error
}
