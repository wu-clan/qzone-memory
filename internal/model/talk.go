package model

import "time"

// Talk 说说
type Talk struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserQQ       string    `gorm:"index;not null" json:"user_qq"`
	TalkID       string    `gorm:"uniqueIndex;not null" json:"talk_id"`
	Content      string    `gorm:"type:text" json:"content"`
	Images       string    `gorm:"type:text" json:"images"` // JSON 数组
	Videos       string    `gorm:"type:text" json:"videos"` // JSON 数组
	Location     string    `json:"location"`
	Device       string    `json:"device"`
	IsDeleted    bool      `gorm:"default:false" json:"is_deleted"`
	LikeCount    int       `json:"like_count"`
	CommentCount int       `json:"comment_count"`
	ShareCount   int       `json:"share_count"`
	PublishTime  time.Time `json:"publish_time"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Talk) TableName() string {
	return "talks"
}
