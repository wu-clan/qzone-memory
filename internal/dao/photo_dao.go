package dao

import (
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm/clause"
)

type PhotoSummary struct {
	PhotoID string
	AlbumID string
}

func GetPhotoByPhotoID(photoID string) (*model.Photo, error) {
	var photo model.Photo
	err := database.DB.Where("photo_id = ?", photoID).First(&photo).Error
	return &photo, err
}

func ListPhotos(userQQ string, offset, limit int) ([]*model.Photo, int64, error) {
	var photos []*model.Photo
	var total int64

	query := database.DB.Model(&model.Photo{}).Where("user_qq = ?", userQQ)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("photo_time DESC").Offset(offset).Limit(limit).Find(&photos).Error
	return photos, total, err
}

func ListPhotosByAlbum(albumID string, offset, limit int) ([]*model.Photo, int64, error) {
	var photos []*model.Photo
	var total int64

	query := database.DB.Model(&model.Photo{}).Where("album_id = ?", albumID)
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
		DoUpdates: clause.AssignmentColumns([]string{"name", "description", "url", "thumb_url", "width", "height", "is_deleted", "updated_at"}),
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
