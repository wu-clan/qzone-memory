package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Code int

const (
	CodeSuccess Code = 0
)

// Response 统一响应结构
type Response struct {
	Code    Code        `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type AppError struct {
	Code int
	Err  error
}

func (e *AppError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, statusCode int, message string) {
	JSON(c, statusCode, Response{
		Code:    Code(statusCode),
		Message: message,
	})
}

// JSON 写入 JSON 响应
func JSON(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}
