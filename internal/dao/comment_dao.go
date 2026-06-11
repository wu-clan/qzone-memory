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

func GetCommentByCommentID(ctx context.Context, commentID string) (*model.Comment, error) {
	var comment model.Comment
	err := database.DB.WithContext(ctx).Where("comment_id = ?", commentID).First(&comment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrNotFound
		}
		return nil, err
	}
	return &comment, nil
}

func ListCommentsByTarget(ctx context.Context, targetType, targetID string, offset, limit int) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	query := database.DB.WithContext(ctx).Model(&model.Comment{}).Where("target_type = ? AND target_id = ?", targetType, targetID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("comment_time DESC").Offset(offset).Limit(limit).Find(&comments).Error
	return comments, total, err
}

func BatchUpsertComments(comments []*model.Comment) error {
	if len(comments) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "comment_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"author_qq", "author_name", "author_avatar", "content", "reply_to_qq", "reply_to_name", "comment_time", "is_deleted", "updated_at"}),
	}).Create(&comments).Error
}
