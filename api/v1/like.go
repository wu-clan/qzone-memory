package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/service"
	"github.com/qzone-memory/pkg/response"
)

func ListLikesByTarget(c *gin.Context) {
	var req dto.QueryByTargetRequest
	if !bindQuery(c, &req) {
		return
	}
	data, err := service.ListLikesByTarget(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}
