package dao

import (
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm/clause"
)

func GetMessageByMessageID(messageID string) (*model.Message, error) {
	var message model.Message
	err := database.DB.Where("message_id = ?", messageID).First(&message).Error
	return &message, err
}

func ListMessages(userQQ string, offset, limit int) ([]*model.Message, int64, error) {
	var messages []*model.Message
	var total int64

	query := database.DB.Model(&model.Message{}).Where("user_qq = ?", userQQ)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("message_time DESC").Offset(offset).Limit(limit).Find(&messages).Error
	return messages, total, err
}

func BatchUpsertMessages(messages []*model.Message) error {
	if len(messages) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "message_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"author_qq", "author_name", "author_avatar", "content", "reply_content", "message_time", "is_deleted", "updated_at"}),
	}).Create(&messages).Error
}
