package dao

import (
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm/clause"
)

type BlogSummary struct {
	BlogID       string
	CommentCount int
}

func GetBlogByBlogID(blogID string) (*model.Blog, error) {
	var blog model.Blog
	err := database.DB.Where("blog_id = ?", blogID).First(&blog).Error
	return &blog, err
}

func ListBlogs(userQQ string, offset, limit int) ([]*model.Blog, int64, error) {
	var blogs []*model.Blog
	var total int64

	query := database.DB.Model(&model.Blog{}).Where("user_qq = ?", userQQ)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("publish_time DESC").Offset(offset).Limit(limit).Find(&blogs).Error
	return blogs, total, err
}

func BatchUpsertBlogs(blogs []*model.Blog) error {
	if len(blogs) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "blog_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_qq", "title", "content", "summary", "category", "tags", "is_deleted", "like_count", "comment_count", "read_count", "publish_time", "updated_at"}),
	}).Create(&blogs).Error
}

func ListBlogSummaries(userQQ string) ([]BlogSummary, error) {
	var summaries []BlogSummary
	err := database.DB.Model(&model.Blog{}).
		Select("blog_id, comment_count").
		Where("user_qq = ?", userQQ).
		Find(&summaries).Error
	return summaries, err
}
