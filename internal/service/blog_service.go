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

func GetBlogList(c *gin.Context) (*dto.PageResponse[*model.Blog], *response.AppError) {
	var req dto.QueryByQQRequest
	if err := bindQuery(c, &req); err != nil {
		return nil, err
	}
	if req.QQ == "" {
		return nil, &response.AppError{Code: http.StatusBadRequest, Err: common.ErrInvalidParam}
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	blogs, total, err := dao.ListBlogs(req.QQ, offset, pageSize)
	if err != nil {
		return nil, &response.AppError{Code: http.StatusInternalServerError, Err: err}
	}

	return dto.NewPageResponse(blogs, total, page, pageSize), nil
}

func GetBlogDetail(c *gin.Context) (*model.Blog, *response.AppError) {
	var req dto.QueryByBlogIDRequest
	if err := bindQuery(c, &req); err != nil {
		return nil, err
	}
	if req.BlogID == "" {
		return nil, &response.AppError{Code: http.StatusBadRequest, Err: common.ErrInvalidParam}
	}

	blog, err := dao.GetBlogByBlogID(req.BlogID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &response.AppError{Code: http.StatusNotFound, Err: common.ErrNotFound}
		}
		return nil, &response.AppError{Code: http.StatusInternalServerError, Err: err}
	}

	return blog, nil
}
