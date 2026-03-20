package dao

import (
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm/clause"
)

func GetUserByQQ(qq string) (*model.User, error) {
	var user model.User
	err := database.DB.Where("qq = ?", qq).First(&user).Error
	return &user, err
}

func UpsertUser(user *model.User) error {
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "qq"}},
		DoUpdates: clause.AssignmentColumns([]string{"nickname", "avatar", "cookie", "gtk", "ps_key", "login_at", "expired_at", "updated_at"}),
	}).Create(user).Error
}
