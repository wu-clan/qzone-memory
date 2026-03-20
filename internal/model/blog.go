package model

import "time"

// Blog 日志
type Blog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserQQ       string    `gorm:"index;not null" json:"user_qq"`
	BlogID       string    `gorm:"uniqueIndex;not null" json:"blog_id"`
	Title        string    `json:"title"`
	Content      string    `gorm:"type:text" json:"content"`
	Summary      string    `json:"summary"`
	Category     string    `json:"category"`
	Tags         string    `gorm:"type:text" json:"tags"`
	IsDeleted    bool      `gorm:"default:false" json:"is_deleted"`
	LikeCount    int       `json:"like_count"`
	CommentCount int       `json:"comment_count"`
	ReadCount    int       `json:"read_count"`
	PublishTime  time.Time `json:"publish_time"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Blog) TableName() string { return "blogs" }
