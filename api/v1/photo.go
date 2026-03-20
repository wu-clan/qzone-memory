package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/service"
	"github.com/qzone-memory/pkg/response"
)

func GetPhotoList(c *gin.Context) {
	data, err := service.GetPhotoList(c)
	if err != nil {
		response.Error(c, err.Code, err.Error())
		return
	}
	response.Success(c, data)
}

func ListPhotosByAlbum(c *gin.Context) {
	data, err := service.ListPhotosByAlbum(c)
	if err != nil {
		response.Error(c, err.Code, err.Error())
		return
	}
	response.Success(c, data)
}
