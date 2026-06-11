package service

import (
	"context"

	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/model"
)

func GetPhotoList(ctx context.Context, req dto.QueryByQQRequest) (*dto.PageResponse[*model.Photo], error) {
	if req.QQ == "" {
		return nil, common.ErrInvalidParam
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	photos, total, err := dao.ListPhotos(ctx, req.QQ, offset, pageSize)
	if err != nil {
		return nil, err
	}

	return dto.NewPageResponse(photos, total, page, pageSize), nil
}

func ListPhotosByAlbum(ctx context.Context, req dto.QueryByAlbumRequest) (*dto.PageResponse[*model.Photo], error) {
	if req.AlbumID == "" {
		return nil, common.ErrInvalidParam
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	photos, total, err := dao.ListPhotosByAlbum(ctx, req.AlbumID, offset, pageSize)
	if err != nil {
		return nil, err
	}

	return dto.NewPageResponse(photos, total, page, pageSize), nil
}
