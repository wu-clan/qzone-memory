package model

import "time"

// Video 空间视频
type Video struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserQQ       string    `gorm:"index;not null" json:"user_qq"`
	VideoID      string    `gorm:"uniqueIndex;not null" json:"video_id"`
	Title        string    `json:"title"`
	Description  string    `gorm:"type:text" json:"description"`
	URL          string    `json:"url"`
	PreviewURL   string    `json:"preview_url"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	Duration     int       `json:"duration"`
	CommentCount int       `json:"comment_count"`
	UploadTime   time.Time `gorm:"index" json:"upload_time"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Video) TableName() string { return "videos" }
