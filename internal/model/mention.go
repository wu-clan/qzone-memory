package model

import "time"

// Mention @提及记录
type Mention struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserQQ      string    `gorm:"index;not null" json:"user_qq"`
	MentionID   string    `gorm:"uniqueIndex;not null" json:"mention_id"`
	SourceType  string    `gorm:"index:idx_mention_source" json:"source_type"`
	SourceID    string    `gorm:"index:idx_mention_source" json:"source_id"`
	AuthorQQ    string    `gorm:"index" json:"author_qq"`
	AuthorName  string    `json:"author_name"`
	Content     string    `gorm:"type:text" json:"content"`
	MentionTime time.Time `json:"mention_time"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Mention) TableName() string { return "mentions" }
