package model

import "time"

// MediaAsset 本地化媒体账本：记录每个远程媒体 URL 的下载状态与本地落点，
// 按 (user_qq, url_hash) 去重，同一 URL 只存一份。
type MediaAsset struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserQQ       string    `gorm:"uniqueIndex:idx_media_user_url;index;not null" json:"user_qq"`
	URLHash      string    `gorm:"uniqueIndex:idx_media_user_url;not null" json:"url_hash"` // sha256(source_url)
	SourceURL    string    `gorm:"type:text;not null" json:"source_url"`
	LocalPath    string    `json:"local_path"` // 相对 media.Dir 的路径
	Category     string    `gorm:"index" json:"category"`
	MimeType     string    `json:"mime_type"`
	Bytes        int64     `json:"bytes"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	Status       string    `gorm:"index;not null" json:"status"` // pending / done / failed / skipped
	Attempts     int       `json:"attempts"`
	Error        string    `gorm:"type:text" json:"error"`
	DownloadedAt time.Time `json:"downloaded_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (MediaAsset) TableName() string { return "media_assets" }
