package model

import "time"

// Favorite 空间收藏
type Favorite struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserQQ     string    `gorm:"index;not null" json:"user_qq"`
	FavoriteID string    `gorm:"uniqueIndex;not null" json:"favorite_id"`
	Type       int       `json:"type"`
	Title      string    `json:"title"`
	Abstract   string    `gorm:"type:text" json:"abstract"`
	URL        string    `json:"url"`
	OwnerQQ    string    `gorm:"index" json:"owner_qq"`
	OwnerName  string    `json:"owner_name"`
	Images     string    `gorm:"type:text" json:"images"`
	CreateTime time.Time `gorm:"index" json:"create_time"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Favorite) TableName() string { return "favorites" }
