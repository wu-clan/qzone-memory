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

func GetActivityList(c *gin.Context) (*dto.PageResponse[*model.Activity], *response.AppError) {
	var req dto.QueryActivityRequest
	if err := bindQuery(c, &req); err != nil {
		return nil, err
	}
	if req.QQ == "" {
		return nil, &response.AppError{Code: http.StatusBadRequest, Err: common.ErrInvalidParam}
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	activities, total, err := dao.ListActivities(req.QQ, req.Type, offset, pageSize)
	if err != nil {
		return nil, &response.AppError{Code: http.StatusInternalServerError, Err: err}
	}

	return dto.NewPageResponse(activities, total, page, pageSize), nil
}

func GetActivityDetail(c *gin.Context) (*model.Activity, *response.AppError) {
	var req dto.QueryByFeedIDRequest
	if err := bindQuery(c, &req); err != nil {
		return nil, err
	}
	if req.FeedID == "" {
		return nil, &response.AppError{Code: http.StatusBadRequest, Err: common.ErrInvalidParam}
	}

	activity, err := dao.GetActivityByFeedID(req.FeedID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &response.AppError{Code: http.StatusNotFound, Err: common.ErrNotFound}
		}
		return nil, &response.AppError{Code: http.StatusInternalServerError, Err: err}
	}

	return activity, nil
}
