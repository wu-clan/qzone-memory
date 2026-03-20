package model

import "time"

// User 登录用户信息
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	QQ        string    `gorm:"uniqueIndex;not null" json:"qq"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Cookie    string    `gorm:"type:text" json:"-"`
	GTK       string    `json:"-"`
	PSKey     string    `json:"-"`
	LoginAt   time.Time `json:"login_at"`
	ExpiredAt time.Time `json:"expired_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (User) TableName() string { return "users" }
