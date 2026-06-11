package service

import (
	"context"

	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/model"
)

func GetAlbumList(ctx context.Context, req dto.QueryByQQRequest) (*dto.PageResponse[*model.Album], error) {
	if req.QQ == "" {
		return nil, common.ErrInvalidParam
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	albums, total, err := dao.ListAlbums(ctx, req.QQ, offset, pageSize)
	if err != nil {
		return nil, err
	}

	return dto.NewPageResponse(albums, total, page, pageSize), nil
}

func GetAlbumDetail(ctx context.Context, req dto.QueryByAlbumIDRequest) (*model.Album, error) {
	if req.AlbumID == "" {
		return nil, common.ErrInvalidParam
	}
	return dao.GetAlbumByAlbumID(ctx, req.AlbumID)
}
