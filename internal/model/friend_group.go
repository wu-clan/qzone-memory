package model

import "time"

// FriendGroup 好友分组
type FriendGroup struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserQQ    string    `gorm:"uniqueIndex:idx_user_group;index;not null" json:"user_qq"`
	GroupID   int       `gorm:"uniqueIndex:idx_user_group;index;not null" json:"group_id"`
	Name      string    `json:"name"`
	IsDeleted bool      `gorm:"default:false" json:"is_deleted"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (FriendGroup) TableName() string { return "friend_groups" }
