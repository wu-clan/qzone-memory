package model

import "time"

// Like 点赞（多态）
type Like struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserQQ      string    `gorm:"index;not null" json:"user_qq"`
	LikeID      string    `gorm:"uniqueIndex;not null" json:"like_id"`
	TargetType  string    `gorm:"index:idx_like_target;not null" json:"target_type"`
	TargetID    string    `gorm:"index:idx_like_target;not null" json:"target_id"`
	LikerQQ     string    `gorm:"index" json:"liker_qq"`
	LikerName   string    `json:"liker_name"`
	LikerAvatar string    `json:"liker_avatar"`
	LikeTime    time.Time `json:"like_time"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Like) TableName() string { return "likes" }
