package model

import "time"

// SyncLog 同步流水日志，按 run_id 串联一次同步中的阶段、分页、保存与异常
type SyncLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	RunID        string    `gorm:"index;not null" json:"run_id"`
	UserQQ       string    `gorm:"index;not null" json:"user_qq"`
	Level        string    `gorm:"index;not null" json:"level"`
	Stage        string    `gorm:"index" json:"stage"`
	Event        string    `gorm:"index;not null" json:"event"`
	Message      string    `gorm:"type:text" json:"message"`
	Page         int       `json:"page"`
	Offset       int       `json:"offset"`
	Limit        int       `json:"limit"`
	ItemsFetched int       `json:"items_fetched"`
	ItemsSaved   int       `json:"items_saved"`
	DurationMS   int64     `json:"duration_ms"`
	Error        string    `gorm:"type:text" json:"error"`
	ContextJSON  string    `gorm:"type:text" json:"context_json"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}

func (SyncLog) TableName() string { return "sync_logs" }
