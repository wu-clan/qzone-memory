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
	apiV1.POST("/sync/resume", v1.ResumeSync)       // 断点续传
	apiV1.POST("/sync/pause", v1.PauseSync)         // 暂停同步
	apiV1.POST("/sync/cancel", v1.CancelSync)       // 取消同步
	apiV1.GET("/sync/progress", v1.GetSyncProgress) // 获取同步进度

	// 历史动态归档
	apiV1.GET("/activities", v1.GetActivityList)                     // 获取历史动态归档
	apiV1.GET("/activities/detail", v1.GetActivityDetail)            // 获取历史动态详情
	apiV1.GET("/memory/timeline", v1.GetMemoryTimeline)              // 获取统一回忆时间线
	apiV1.GET("/memory/stats", v1.GetMemoryStats)                    // 获取回忆统计
	apiV1.GET("/memory/on-this-day", v1.GetOnThisDay)                // 那年今日（同月同日）
	apiV1.GET("/memory/search", v1.SearchMemory)                     // 全文搜索
	apiV1.GET("/memory/interactions", v1.SearchFriendInteractions)   // 按好友/QQ 查询互动
	apiV1.GET("/memory/item/interactions", v1.GetMemoryInteractions) // 获取单条回忆下的完整互动
	apiV1.GET("/memory/report", v1.GetMemoryReport)                  // 年度纪念卡
	apiV1.GET("/friends", v1.GetFriendList)                          // 获取好友与历史联系人
	apiV1.GET("/visitors", v1.GetVisitorList)                        // 获取访客记录
	apiV1.GET("/videos", v1.GetVideoList)                            // 获取视频列表
	apiV1.GET("/favorites", v1.GetFavoriteList)                      // 获取收藏列表
	apiV1.GET("/diaries", v1.GetDiaryList)                           // 获取私密日记

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

	// 图片代理（命中本地走本地，未命中回源并后台下载）
	apiV1.GET("/proxy/image", v1.ProxyImage)

	// 数据与隐私
	apiV1.GET("/storage/stats", v1.GetStorageStats)       // 数据位置与媒体本地化统计
	apiV1.POST("/media/backfill", v1.BackfillMedia)       // 媒体回填 / 重新下载失败项
	apiV1.POST("/data/delete", v1.DeleteData)             // 彻底删除我的数据
	apiV1.POST("/data/reprocess", v1.ReprocessActivities) // 重解析动态（剥离"别人赞我"、恢复真实说说）
	apiV1.GET("/export", v1.ExportArchive)                // 导出离线纪念册（zip）

	return router
}
