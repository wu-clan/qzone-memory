package model

import "time"

// Album 相册
type Album struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserQQ      string    `gorm:"index;not null" json:"user_qq"`
	AlbumID     string    `gorm:"uniqueIndex;not null" json:"album_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CoverURL    string    `json:"cover_url"`
	PhotoCount  int       `json:"photo_count"`
	IsDeleted   bool      `gorm:"default:false" json:"is_deleted"`
	CreateTime  time.Time `json:"create_time"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Album) TableName() string { return "albums" }
