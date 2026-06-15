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

type PhotoSummary struct {
	PhotoID string
	AlbumID string
}

func GetPhotoByPhotoID(ctx context.Context, photoID string) (*model.Photo, error) {
	var photo model.Photo
	err := database.DB.WithContext(ctx).Where("photo_id = ?", photoID).First(&photo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ErrNotFound
		}
		return nil, err
	}
	return &photo, nil
}

func ListPhotos(ctx context.Context, userQQ string, offset, limit int) ([]*model.Photo, int64, error) {
	var photos []*model.Photo
	var total int64

	query := database.DB.WithContext(ctx).Model(&model.Photo{}).Where("user_qq = ?", userQQ)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("photo_time DESC").Offset(offset).Limit(limit).Find(&photos).Error
	return photos, total, err
}

func ListPhotosByAlbum(ctx context.Context, albumID string, offset, limit int) ([]*model.Photo, int64, error) {
	var photos []*model.Photo
	var total int64

	query := database.DB.WithContext(ctx).Model(&model.Photo{}).Where("album_id = ?", albumID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("photo_time DESC").Offset(offset).Limit(limit).Find(&photos).Error
	return photos, total, err
}

func BatchUpsertPhotos(photos []*model.Photo) error {
	if len(photos) == 0 {
		return nil
	}
	return database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "photo_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_qq", "album_id", "name", "description", "url", "thumb_url", "owner_qq", "owner_name", "width", "height", "is_deleted", "photo_time", "updated_at"}),
	}).Create(&photos).Error
}

func ListPhotoSummaries(userQQ string) ([]PhotoSummary, error) {
	var summaries []PhotoSummary
	err := database.DB.Model(&model.Photo{}).
		Select("photo_id, album_id").
		Where("user_qq = ?", userQQ).
		Find(&summaries).Error
	return summaries, err
}
