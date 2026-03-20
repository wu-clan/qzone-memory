package model

import "time"

// Share 转发
type Share struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserQQ    string    `gorm:"index;not null" json:"user_qq"`
	ShareID   string    `gorm:"uniqueIndex;not null" json:"share_id"`
	TargetType string   `gorm:"index:idx_share_target" json:"target_type"`
	TargetID  string    `gorm:"index:idx_share_target" json:"target_id"`
	SharerQQ  string    `gorm:"index" json:"sharer_qq"`
	SharerName string   `json:"sharer_name"`
	Comment   string    `gorm:"type:text" json:"comment"`
	ShareTime time.Time `json:"share_time"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Share) TableName() string { return "shares" }
