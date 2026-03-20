package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	v1 "github.com/qzone-memory/api/v1"
	"github.com/qzone-memory/web"
)

func RegisterRoutes(mode string) *gin.Engine {
	gin.SetMode(mode)

	router := gin.Default()
	router.Use(func(c *gin.Context) {
		if c.Request.Method == http.MethodGet &&
			(c.Request.URL.Path == "/" || strings.HasPrefix(c.Request.URL.Path, "/static/")) {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}
		c.Next()
	})

	apiV1 := router.Group("/api/v1")

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// 静态资源
	staticFS, _ := fs.Sub(web.Assets, "static")
	router.StaticFS("/static", http.FS(staticFS))

	// 根路径返回前端页面
	router.NoRoute(func(c *gin.Context) {
		if c.Request.URL.Path != "/" {
			c.Status(http.StatusNotFound)
			return
		}
		data, err := web.Assets.ReadFile("index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "页面加载失败")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	// 登录接口
	apiV1.GET("/login/qrcode", v1.GenerateLoginQRCode) // 获取登录二维码
	apiV1.GET("/login/status", v1.PollLoginStatus)     // 轮询登录状态
	apiV1.GET("/login/user", v1.GetCurrentUser)        // 获取当前登录用户

	// 同步接口
	apiV1.POST("/sync/start", v1.StartSync)         // 开始同步数据
	apiV1.GET("/sync/progress", v1.GetSyncProgress) // 获取同步进度

	// 说说和日志
	apiV1.GET("/talks", v1.GetTalkList)          // 获取说说列表
	apiV1.GET("/talks/detail", v1.GetTalkDetail) // 获取说说详情
	apiV1.GET("/blogs", v1.GetBlogList)          // 获取日志列表
	apiV1.GET("/blogs/detail", v1.GetBlogDetail) // 获取日志详情

	// 相册和照片
	apiV1.GET("/albums", v1.GetAlbumList)               // 获取相册列表
	apiV1.GET("/albums/detail", v1.GetAlbumDetail)      // 获取相册详情
	apiV1.GET("/photos", v1.GetPhotoList)               // 获取照片列表
	apiV1.GET("/photos/by-album", v1.ListPhotosByAlbum) // 按相册获取照片

	// 互动数据
	apiV1.GET("/messages", v1.GetMessageList)       // 获取留言列表
	apiV1.GET("/comments", v1.ListCommentsByTarget) // 获取评论列表
	apiV1.GET("/likes", v1.ListLikesByTarget)       // 获取点赞列表
	apiV1.GET("/shares", v1.ListSharesByTarget)     // 获取转发列表
	apiV1.GET("/mentions", v1.GetMentionList)       // 获取提及列表

	// 图片代理
	apiV1.GET("/proxy/image", v1.ProxyImage) // 代理 QQ 空间图片

	return router
}
