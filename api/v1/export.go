package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/service"
	"github.com/qzone-memory/pkg/logger"
	"go.uber.org/zap"
)

// ExportArchive 导出某账号的离线纪念册（自带媒体的静态 HTML 站点打包为 zip）。
func ExportArchive(c *gin.Context) {
	var req dto.QueryByQQRequest
	if !bindQuery(c, &req) {
		return
	}

	nickname := req.QQ
	if user, err := dao.GetUserByQQ(c.Request.Context(), req.QQ); err == nil && user.Nickname != "" {
		nickname = user.Nickname
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="qzone-archive-%s.zip"`, req.QQ))
	if err := service.ExportArchive(c.Request.Context(), req.QQ, nickname, c.Writer); err != nil {
		// 头部可能已发出，无法再改状态码，记录即可
		logger.Error("导出纪念册失败", zap.String("qq", req.QQ), zap.Error(err))
	}
}
