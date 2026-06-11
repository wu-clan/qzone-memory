package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/service"
	"github.com/qzone-memory/pkg/response"
)

func GetActivityList(c *gin.Context) {
	var req dto.QueryActivityRequest
	if !bindQuery(c, &req) {
		return
	}
	data, err := service.GetActivityList(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}

func GetActivityDetail(c *gin.Context) {
	var req dto.QueryByFeedIDRequest
	if !bindQuery(c, &req) {
		return
	}
	data, err := service.GetActivityDetail(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}
