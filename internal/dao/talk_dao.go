package dao

import (
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm/clause"
)

func GetTalkByTalkID(talkID string) (*model.Talk, error) {
	var talk model.Talk
	err := database.DB.Where("talk_id = ?", talkID).First(&talk).Error
	return &talk, err
}

func ListTalks(userQQ string, offset, limit int) ([]*model.Talk, int64, error) {
	var talks []*model.Talk
	var total int64

	query := database.DB.Model(&model.Talk{}).Where("user_qq = ?", userQQ)

	// 查询总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	err := query.Order("publish_time DESC").
		Offset(offset).
		Limit(limit).
		Find(&talks).Error

	return talks, total, err
}

func BatchUpsertTalks(talks []*model.Talk) error {
	if len(talks) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "talk_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_qq", "content", "images", "videos", "location", "device", "is_deleted", "like_count", "comment_count", "share_count", "publish_time", "updated_at"}),
	}).Create(&talks).Error
}

// TalkSummary 说说摘要，仅包含同步所需字段
type TalkSummary struct {
	TalkID       string
	CommentCount int
	LikeCount    int
	ShareCount   int
}

// ListTalkSummaries 获取指定用户的所有说说摘要
func ListTalkSummaries(userQQ string) ([]TalkSummary, error) {
	var summaries []TalkSummary
	err := database.DB.Model(&model.Talk{}).
		Select("talk_id, comment_count, like_count, share_count").
		Where("user_qq = ?", userQQ).
		Find(&summaries).Error
	return summaries, err
}
