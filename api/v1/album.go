package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/service"
	"github.com/qzone-memory/pkg/response"
)

func GetAlbumList(c *gin.Context) {
	data, err := service.GetAlbumList(c)
	if err != nil {
		response.Error(c, err.Code, err.Error())
		return
	}
	response.Success(c, data)
}

func GetAlbumDetail(c *gin.Context) {
	data, err := service.GetAlbumDetail(c)
	if err != nil {
		response.Error(c, err.Code, err.Error())
		return
	}
	response.Success(c, data)
}
