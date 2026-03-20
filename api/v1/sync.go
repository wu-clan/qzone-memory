package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/service"
	"github.com/qzone-memory/pkg/response"
)

func StartSync(c *gin.Context) {
	data, err := service.StartSync(c)
	if err != nil {
		response.Error(c, err.Code, err.Error())
		return
	}
	response.Success(c, data)
}

func GetSyncProgress(c *gin.Context) {
	data, err := service.GetSyncProgress()
	if err != nil {
		response.Error(c, err.Code, err.Error())
		return
	}
	response.Success(c, data)
}
