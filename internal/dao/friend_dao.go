package dao

import (
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ListFriends(userQQ string, includeDeleted bool, offset, limit int) ([]*model.Friend, int64, error) {
	var friends []*model.Friend
	var total int64

	query := database.DB.Model(&model.Friend{}).Where("user_qq = ?", userQQ)
	if !includeDeleted {
		query = query.Where("is_deleted = ?", false)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("is_current DESC, interact_count DESC, last_seen_at DESC, updated_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&friends).Error
	return friends, total, err
}

func CountFriendsByStatus(userQQ string) (currentTotal, historicalTotal int64, err error) {
	if err = database.DB.Model(&model.Friend{}).
		Where("user_qq = ? AND is_current = ?", userQQ, true).
		Count(&currentTotal).Error; err != nil {
		return 0, 0, err
	}

	if err = database.DB.Model(&model.Friend{}).
		Where("user_qq = ? AND is_deleted = ?", userQQ, true).
		Count(&historicalTotal).Error; err != nil {
		return 0, 0, err
	}

	return currentTotal, historicalTotal, nil
}

func BatchUpsertFriends(friends []*model.Friend) error {
	if len(friends) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_qq"}, {Name: "friend_qq"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name",
			"remark",
			"avatar",
			"group_id",
			"group_name",
			"is_current",
			"is_deleted",
			"is_special_care",
			"source_type",
			"online",
			"yellow",
			"common_group",
			"interact_count",
			"add_time",
			"last_seen_at",
			"updated_at",
		}),
	}).Create(&friends).Error
}

func MarkCurrentFriendsDeleted(userQQ string, currentQQs []string) error {
	query := database.DB.Model(&model.Friend{}).Where("user_qq = ? AND is_current = ?", userQQ, true)
	if len(currentQQs) > 0 {
		query = query.Where("friend_qq NOT IN ?", currentQQs)
	}
	return query.Updates(map[string]interface{}{
		"is_current": false,
		"is_deleted": true,
	}).Error
}

func IncreaseFriendInteract(userQQ, friendQQ string, delta int, seenAt interface{}) error {
	updates := map[string]interface{}{
		"interact_count": gorm.Expr("interact_count + ?", delta),
	}
	if seenAt != nil {
		updates["last_seen_at"] = seenAt
	}
	return database.DB.Model(&model.Friend{}).
		Where("user_qq = ? AND friend_qq = ?", userQQ, friendQQ).
		Updates(updates).Error
}
