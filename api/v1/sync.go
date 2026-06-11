package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/service"
	"github.com/qzone-memory/pkg/response"
)

func StartSync(c *gin.Context) {
	var req dto.SyncRequest
	if !bindJSON(c, &req) {
		return
	}
	data, err := service.StartSync(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}

func GetSyncProgress(c *gin.Context) {
	data, err := service.GetSyncProgress()
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}
