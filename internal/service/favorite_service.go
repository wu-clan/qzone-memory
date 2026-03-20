package service

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/model"
	"github.com/qzone-memory/pkg/response"
)

func GetFavoriteList(c *gin.Context) (*dto.PageResponse[*model.Favorite], *response.AppError) {
	var req dto.QueryByQQRequest
	if err := bindQuery(c, &req); err != nil {
		return nil, err
	}
	if req.QQ == "" {
		return nil, &response.AppError{Code: http.StatusBadRequest, Err: common.ErrInvalidParam}
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	items, total, err := dao.ListFavorites(req.QQ, offset, pageSize)
	if err != nil {
		return nil, &response.AppError{Code: http.StatusInternalServerError, Err: err}
	}
	return dto.NewPageResponse(items, total, page, pageSize), nil
}
