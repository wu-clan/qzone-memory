package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/service"
	"github.com/qzone-memory/pkg/response"
)

func GenerateLoginQRCode(c *gin.Context) {
	data, err := service.GenerateLoginQRCode()
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}

func PollLoginStatus(c *gin.Context) {
	c.Header("X-Login-Status-Version", "normalized-200-v2")

	data, err := service.PollLoginStatus(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}

func GetCurrentUser(c *gin.Context) {
	var req dto.QueryByQQRequest
	if !bindQuery(c, &req) {
		return
	}
	data, err := service.GetCurrentUser(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}
