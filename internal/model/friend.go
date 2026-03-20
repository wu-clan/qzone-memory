package model

import "time"

// Friend QQ 好友与历史联系人
type Friend struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserQQ        string    `gorm:"uniqueIndex:idx_user_friend;index;not null" json:"user_qq"`
	FriendQQ      string    `gorm:"uniqueIndex:idx_user_friend;index;not null" json:"friend_qq"`
	Name          string    `json:"name"`
	Remark        string    `json:"remark"`
	Avatar        string    `json:"avatar"`
	GroupID       int       `gorm:"index" json:"group_id"`
	GroupName     string    `json:"group_name"`
	IsCurrent     bool      `gorm:"default:false;index" json:"is_current"`
	IsDeleted     bool      `gorm:"default:false;index" json:"is_deleted"`
	IsSpecialCare bool      `gorm:"default:false" json:"is_special_care"`
	SourceType    string    `gorm:"index" json:"source_type"`
	Online        int       `json:"online"`
	Yellow        int       `json:"yellow"`
	CommonGroup   int       `json:"common_group"`
	InteractCount int       `json:"interact_count"`
	AddTime       time.Time `json:"add_time"`
	LastSeenAt    time.Time `json:"last_seen_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Friend) TableName() string { return "friends" }
