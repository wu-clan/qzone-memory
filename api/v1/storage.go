package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/service"
	"github.com/qzone-memory/pkg/response"
)

// GetStorageStats 返回数据位置与媒体本地化统计（数据归我面板）。
func GetStorageStats(c *gin.Context) {
	var req dto.QueryByQQRequest
	if !bindQuery(c, &req) {
		return
	}
	data, err := service.GetStorageStats(c.Request.Context(), req.QQ)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}

// BackfillMedia 触发后台媒体回填 / 重新下载失败项。
func BackfillMedia(c *gin.Context) {
	var req dto.QueryByQQRequest
	if !bindQuery(c, &req) {
		return
	}
	if err := service.BackfillMedia(req.QQ); err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "媒体回填已在后台开始"})
}
