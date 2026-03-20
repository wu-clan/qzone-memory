package model

import "time"

// Photo 照片
type Photo struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserQQ      string    `gorm:"index;not null" json:"user_qq"`
	PhotoID     string    `gorm:"uniqueIndex;not null" json:"photo_id"`
	AlbumID     string    `gorm:"index;not null" json:"album_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	ThumbURL    string    `json:"thumb_url"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	IsDeleted   bool      `gorm:"default:false" json:"is_deleted"`
	PhotoTime   time.Time `json:"photo_time"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Photo) TableName() string { return "photos" }
