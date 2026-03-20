package model

import "time"

// Diary 私密日记
type Diary struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserQQ     string    `gorm:"index;not null" json:"user_qq"`
	DiaryID    string    `gorm:"uniqueIndex;not null" json:"diary_id"`
	Title      string    `json:"title"`
	Summary    string    `gorm:"type:text" json:"summary"`
	Content    string    `gorm:"type:text" json:"content"`
	CreateTime time.Time `gorm:"index" json:"create_time"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Diary) TableName() string { return "diaries" }
