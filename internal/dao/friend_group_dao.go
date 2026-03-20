package dao

import (
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm/clause"
)

func ListFriendGroups(userQQ string) ([]*model.FriendGroup, error) {
	var groups []*model.FriendGroup
	err := database.DB.Where("user_qq = ?", userQQ).Order("group_id ASC").Find(&groups).Error
	return groups, err
}

func BatchUpsertFriendGroups(groups []*model.FriendGroup) error {
	if len(groups) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_qq"}, {Name: "group_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "is_deleted", "updated_at"}),
	}).Create(&groups).Error
}

func MarkMissingGroupsDeleted(userQQ string, groupIDs []int) error {
	query := database.DB.Model(&model.FriendGroup{}).Where("user_qq = ?", userQQ)
	if len(groupIDs) > 0 {
		query = query.Where("group_id NOT IN ?", groupIDs)
	}
	return query.Update("is_deleted", true).Error
}
