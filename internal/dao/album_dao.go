package dao

import (
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/model"
	"gorm.io/gorm/clause"
)

func GetAlbumByAlbumID(albumID string) (*model.Album, error) {
	var album model.Album
	err := database.DB.Where("album_id = ?", albumID).First(&album).Error
	return &album, err
}

func ListAlbums(userQQ string, offset, limit int) ([]*model.Album, int64, error) {
	var albums []*model.Album
	var total int64

	query := database.DB.Model(&model.Album{}).Where("user_qq = ?", userQQ)
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
		DoUpdates: clause.AssignmentColumns([]string{"user_qq", "name", "description", "cover_url", "photo_count", "is_deleted", "create_time", "updated_at"}),
	}).Create(&albums).Error
}
