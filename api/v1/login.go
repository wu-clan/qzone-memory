package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/service"
	"github.com/qzone-memory/pkg/response"
	"github.com/qzone-memory/qzone"
)

func GenerateLoginQRCode(c *gin.Context) {
	data, err := service.GenerateLoginQRCode()
	if err != nil {
		response.Error(c, err.Code, err.Error())
		return
	}
	response.Success(c, data)
}

func PollLoginStatus(c *gin.Context) {
	c.Header("X-Login-Status-Version", "normalized-200-v2")

	data, err := service.PollLoginStatus()
	if err != nil {
		if err.Code == http.StatusGone {
			response.Success(c, &qzone.LoginStatus{
				Status:  3,
				Message: err.Error(),
			})
			return
		}
		response.Error(c, err.Code, err.Error())
		return
	}
	response.Success(c, data)
}

func GetCurrentUser(c *gin.Context) {
	data, err := service.GetCurrentUser(c)
	if err != nil {
		response.Error(c, err.Code, err.Error())
		return
	}
	response.Success(c, data)
}
