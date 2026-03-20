package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/service"
	"github.com/qzone-memory/pkg/response"
)

func GetTalkList(c *gin.Context) {
	data, err := service.GetTalkList(c)
	if err != nil {
		response.Error(c, err.Code, err.Error())
		return
	}
	response.Success(c, data)
}

func GetTalkDetail(c *gin.Context) {
	data, err := service.GetTalkDetail(c)
	if err != nil {
		response.Error(c, err.Code, err.Error())
		return
	}
	response.Success(c, data)
}
