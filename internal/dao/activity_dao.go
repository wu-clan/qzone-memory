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

func GetActivityByFeedID(ctx context.Context, feedID string) (*model.Activity, error) {
	var activity model.Activity
	err := database.DB.WithContext(ctx).Where("feed_id = ?", feedID).First(&activity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrNotFound
		}
		return nil, err
	}
	return &activity, nil
}

func ListActivities(ctx context.Context, userQQ, feedType string, offset, limit int) ([]*model.Activity, int64, error) {
	var activities []*model.Activity
	var total int64

	query := database.DB.WithContext(ctx).Model(&model.Activity{}).Where("user_qq = ?", userQQ)
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
			"app_id",
			"raw_type",
			"object_id",
			"title",
			"content",
			"html_content",
			"author_qq",
			"author_name",
			"images",
			"source_name",
			"device",
			"location",
			"url",
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
