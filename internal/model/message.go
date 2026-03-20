package model

import "time"

// Message 留言板消息
type Message struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserQQ       string    `gorm:"index;not null" json:"user_qq"`
	MessageID    string    `gorm:"uniqueIndex;not null" json:"message_id"`
	AuthorQQ     string    `gorm:"index" json:"author_qq"`
	AuthorName   string    `json:"author_name"`
	AuthorAvatar string    `json:"author_avatar"`
	Content      string    `gorm:"type:text" json:"content"`
	ReplyContent string    `gorm:"type:text" json:"reply_content"`
	IsDeleted    bool      `gorm:"default:false" json:"is_deleted"`
	MessageTime  time.Time `json:"message_time"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Message) TableName() string { return "messages" }
