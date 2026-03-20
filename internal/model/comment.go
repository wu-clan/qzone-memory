package model

import "time"

// Comment 评论（多态，可属于说说/日志/照片/留言）
type Comment struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserQQ       string    `gorm:"index;not null" json:"user_qq"`
	CommentID    string    `gorm:"uniqueIndex;not null" json:"comment_id"`
	TargetType   string    `gorm:"index:idx_comment_target;not null" json:"target_type"`
	TargetID     string    `gorm:"index:idx_comment_target;not null" json:"target_id"`
	AuthorQQ     string    `gorm:"index" json:"author_qq"`
	AuthorName   string    `json:"author_name"`
	AuthorAvatar string    `json:"author_avatar"`
	Content      string    `gorm:"type:text" json:"content"`
	ReplyToQQ    string    `json:"reply_to_qq"`
	ReplyToName  string    `json:"reply_to_name"`
	IsDeleted    bool      `gorm:"default:false" json:"is_deleted"`
	CommentTime  time.Time `json:"comment_time"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Comment) TableName() string { return "comments" }
