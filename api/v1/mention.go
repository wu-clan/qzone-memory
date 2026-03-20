package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/service"
	"github.com/qzone-memory/pkg/response"
)

func GetMentionList(c *gin.Context) {
	data, err := service.GetMentionList(c)
	if err != nil {
		response.Error(c, err.Code, err.Error())
		return
	}
	response.Success(c, data)
}
