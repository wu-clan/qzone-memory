package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/pkg/response"
)

func bindQuery(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindQuery(req); err != nil {
		response.Error(c, http.StatusBadRequest, common.ErrInvalidParam.Error())
		return false
	}
	return true
}

func bindJSON(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		response.Error(c, http.StatusBadRequest, common.ErrInvalidParam.Error())
		return false
	}
	return true
}

func writeServiceError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, common.ErrInvalidParam), errors.Is(err, common.ErrInvalidQQ):
		status = http.StatusBadRequest
	case errors.Is(err, common.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, common.ErrUnauthorized):
		status = http.StatusUnauthorized
	case errors.Is(err, common.ErrSyncRunning):
		status = http.StatusConflict
	}
	response.Error(c, status, err.Error())
}
