package v1

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/media"
)

var proxyClient = &http.Client{
	Timeout: 15 * time.Second,
}

// ProxyImage 媒体解析器：命中本地则直接返回本地文件，否则实时回源并登记后台下载，
// 使图片在首次浏览后逐步本地化，最终达到「浏览零外部请求」。
func ProxyImage(c *gin.Context) {
	imageURL := c.Query("url")
	if imageURL == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	// 仅允许 QQ 相关域名
	if !strings.Contains(imageURL, ".qq.com") && !strings.Contains(imageURL, ".qlogo.cn") && !strings.Contains(imageURL, ".qpic.cn") {
		c.Status(http.StatusForbidden)
		return
	}

	// 命中本地：直接返回本地文件（http.ServeFile 自动处理 Content-Type 与 Range）
	qq := c.Query("qq")
	if qq != "" {
		if local, ok := media.Resolve(c.Request.Context(), qq, imageURL); ok {
			c.Header("Cache-Control", "public, max-age=31536000")
			c.File(local)
			return
		}
		// 未命中：登记并后台下载，下次浏览即走本地
		media.EnqueueAndDownload(qq, imageURL, "ondemand")
	}

	// 回源（降级路径，确保任何阶段都不比从前更糟）
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}

	req.Header.Set("Referer", "https://user.qzone.qq.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")

	resp, err := proxyClient.Do(req)
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=86400")
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}
