package dao

import (
	"context"
	"errors"
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GetUserByQQ(ctx context.Context, qq string) (*model.User, error) {
	var user model.User
	err := database.DB.WithContext(ctx).Where("qq = ?", qq).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func UpsertUser(ctx context.Context, user *model.User) error {
	return database.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "qq"}},
		DoUpdates: clause.AssignmentColumns([]string{"nickname", "avatar", "cookie", "gtk", "ps_key", "login_at", "expired_at", "updated_at"}),
	}).Create(user).Error
}
