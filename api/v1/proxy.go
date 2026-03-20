package v1

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var proxyClient = &http.Client{
	Timeout: 15 * time.Second,
}

// ProxyImage 代理 QQ 空间图片请求，解决浏览器无法直接访问 CDN 的问题
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
