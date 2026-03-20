package service

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/model"
	"github.com/qzone-memory/pkg/response"
	"gorm.io/gorm"
)

func GetAlbumList(c *gin.Context) (*dto.PageResponse[*model.Album], *response.AppError) {
	var req dto.QueryByQQRequest
	if err := bindQuery(c, &req); err != nil {
		return nil, err
	}
	if req.QQ == "" {
		return nil, &response.AppError{Code: http.StatusBadRequest, Err: common.ErrInvalidParam}
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	albums, total, err := dao.ListAlbums(req.QQ, offset, pageSize)
	if err != nil {
		return nil, &response.AppError{Code: http.StatusInternalServerError, Err: err}
	}

	return dto.NewPageResponse(albums, total, page, pageSize), nil
}

func GetAlbumDetail(c *gin.Context) (*model.Album, *response.AppError) {
	var req dto.QueryByAlbumIDRequest
	if err := bindQuery(c, &req); err != nil {
		return nil, err
	}
	if req.AlbumID == "" {
		return nil, &response.AppError{Code: http.StatusBadRequest, Err: common.ErrInvalidParam}
	}

	album, err := dao.GetAlbumByAlbumID(req.AlbumID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &response.AppError{Code: http.StatusNotFound, Err: common.ErrNotFound}
		}
		return nil, &response.AppError{Code: http.StatusInternalServerError, Err: err}
	}

	return album, nil
}
