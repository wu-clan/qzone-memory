package model

import "time"

// Visitor 空间访客记录
type Visitor struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserQQ      string    `gorm:"index;not null" json:"user_qq"`
	VisitorID   string    `gorm:"uniqueIndex;not null" json:"visitor_id"`
	VisitorQQ   string    `gorm:"index" json:"visitor_qq"`
	VisitorName string    `json:"visitor_name"`
	Avatar      string    `json:"avatar"`
	Source      int       `json:"source"`
	IsHidden    bool      `json:"is_hidden"`
	YellowLevel int       `json:"yellow_level"`
	VisitTime   time.Time `gorm:"index" json:"visit_time"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Visitor) TableName() string { return "visitors" }
