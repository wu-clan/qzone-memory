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

func GetAlbumByAlbumID(ctx context.Context, albumID string) (*model.Album, error) {
	var album model.Album
	err := database.DB.WithContext(ctx).Where("album_id = ?", albumID).First(&album).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrNotFound
		}
		return nil, err
	}
	return &album, nil
}

func ListAlbums(ctx context.Context, userQQ string, offset, limit int) ([]*model.Album, int64, error) {
	var albums []*model.Album
	var total int64

	query := database.DB.WithContext(ctx).Model(&model.Album{}).Where("user_qq = ?", userQQ)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("create_time DESC").Offset(offset).Limit(limit).Find(&albums).Error
	return albums, total, err
}

func BatchUpsertAlbums(albums []*model.Album) error {
	if len(albums) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "album_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_qq", "name", "description", "cover_url", "photo_count", "privacy", "is_deleted", "create_time", "last_upload_time", "updated_at"}),
	}).Create(&albums).Error
}
