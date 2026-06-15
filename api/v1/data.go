package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/service"
	"github.com/qzone-memory/pkg/response"
)

// DeleteData 彻底删除某 QQ 的全部本地数据（库 + 媒体 + 登录态）。
func DeleteData(c *gin.Context) {
	var req dto.DataDeleteRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := service.DeleteAllData(c.Request.Context(), req.QQ, req.Confirm); err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "数据已彻底删除"})
}

// ReprocessActivities 用修正后的逻辑重新解析已存动态（剥离"别人赞我"、恢复真实说说）。
func ReprocessActivities(c *gin.Context) {
	var req dto.QueryByQQRequest
	if !bindQuery(c, &req) {
		return
	}
	data, err := service.ReprocessActivities(c.Request.Context(), req.QQ)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}
