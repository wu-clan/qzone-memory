package model

import "time"

// Activity QQ 空间动态归档
type Activity struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserQQ       string    `gorm:"index;not null" json:"user_qq"`
	FeedID       string    `gorm:"uniqueIndex;not null" json:"feed_id"`
	FeedType     string    `gorm:"index;not null" json:"feed_type"`
	ObjectID     string    `gorm:"index" json:"object_id"`
	Title        string    `json:"title"`
	Content      string    `gorm:"type:text" json:"content"`
	HTMLContent  string    `gorm:"type:text" json:"html_content"`
	AuthorQQ     string    `gorm:"index" json:"author_qq"`
	AuthorName   string    `json:"author_name"`
	Images       string    `gorm:"type:text" json:"images"`
	LikeCount    int       `json:"like_count"`
	CommentCount int       `json:"comment_count"`
	ShareCount   int       `json:"share_count"`
	IsDeleted    bool      `gorm:"default:false" json:"is_deleted"`
	PublishTime  time.Time `gorm:"index" json:"publish_time"`
	StateText    string    `json:"state_text"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Activity) TableName() string { return "activities" }
