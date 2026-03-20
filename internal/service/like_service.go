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

func ListLikesByTarget(c *gin.Context) (*dto.PageResponse[*model.Like], *response.AppError) {
	var req dto.QueryByTargetRequest
	if err := bindQuery(c, &req); err != nil {
		return nil, err
	}
	if req.TargetType == "" || req.TargetID == "" {
		return nil, &response.AppError{Code: http.StatusBadRequest, Err: common.ErrInvalidParam}
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	likes, total, err := dao.ListLikesByTarget(req.TargetType, req.TargetID, offset, pageSize)
	if err != nil {
		return nil, &response.AppError{Code: http.StatusInternalServerError, Err: err}
	}

	return dto.NewPageResponse(likes, total, page, pageSize), nil
}
