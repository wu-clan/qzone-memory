package model

import "time"

// SyncState 同步状态账本：按 (user_qq, type) 持久化每类数据的同步进度，
// 用于断点续传与重建同步进度展示。type 取 talk / blog / album / … / media。
type SyncState struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserQQ       string    `gorm:"uniqueIndex:idx_sync_user_type;index;not null" json:"user_qq"`
	Type         string    `gorm:"uniqueIndex:idx_sync_user_type;not null" json:"type"`
	Status       string    `gorm:"not null" json:"status"` // pending / running / done / failed / paused
	LastOffset   int       `json:"last_offset"`
	Cursor       string    `json:"cursor"` // 部分接口用游标而非 offset
	ItemsSynced  int       `json:"items_synced"`
	LastSyncedAt time.Time `json:"last_synced_at"`
	Error        string    `gorm:"type:text" json:"error"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (SyncState) TableName() string { return "sync_state" }
