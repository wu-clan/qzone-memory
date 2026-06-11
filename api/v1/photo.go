package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/service"
	"github.com/qzone-memory/pkg/response"
)

func GetPhotoList(c *gin.Context) {
	var req dto.QueryByQQRequest
	if !bindQuery(c, &req) {
		return
	}
	data, err := service.GetPhotoList(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}

func ListPhotosByAlbum(c *gin.Context) {
	var req dto.QueryByAlbumRequest
	if !bindQuery(c, &req) {
		return
	}
	data, err := service.ListPhotosByAlbum(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}
