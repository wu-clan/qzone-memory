package service

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/pkg/response"
)

var qqRegexp = regexp.MustCompile(`^\d{5,20}$`)

// normalizePage 标准化分页参数
func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = common.DefaultPage
	}
	if pageSize <= 0 {
		pageSize = common.DefaultPageSize
	}
	if pageSize > common.MaxPageSize {
		pageSize = common.MaxPageSize
	}
	return page, pageSize
}

func bindQuery(c *gin.Context, req interface{}) *response.AppError {
	if err := c.ShouldBindQuery(req); err != nil {
		return &response.AppError{Code: http.StatusBadRequest, Err: common.ErrInvalidParam}
	}
	return nil
}

func bindJSON(c *gin.Context, req interface{}) *response.AppError {
	if err := c.ShouldBindJSON(req); err != nil {
		return &response.AppError{Code: http.StatusBadRequest, Err: common.ErrInvalidParam}
	}
	return nil
}

func validateQQ(qq string) *response.AppError {
	if !qqRegexp.MatchString(qq) {
		return &response.AppError{Code: http.StatusBadRequest, Err: errors.New("QQ号格式错误")}
	}
	return nil
}
