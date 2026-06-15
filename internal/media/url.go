package media

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"path"
	"strings"
)

// isTencentURL 判断是否为可本地化的腾讯自有资源（与 proxy 放行的域名一致）。
func isTencentURL(raw string) bool {
	v := strings.ToLower(raw)
	return strings.Contains(v, ".qq.com") ||
		strings.Contains(v, ".qlogo.cn") ||
		strings.Contains(v, ".qpic.cn")
}

// normalizeURL 仅做无害归一化（去首尾空白），保留可直接回源的完整 URL，
// 既保证去重为精确匹配，也保证降级回源仍可用。
func normalizeURL(raw string) string {
	return strings.TrimSpace(raw)
}

// hashURL 以归一化 URL 的 sha256 作为去重键与文件名。
func hashURL(raw string) string {
	sum := sha256.Sum256([]byte(normalizeURL(raw)))
	return hex.EncodeToString(sum[:])
}

// extFromURL 从 URL 路径推断扩展名，异常时返回空串。
func extFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	ext := strings.ToLower(path.Ext(u.Path))
	if len(ext) > 5 || strings.ContainsAny(ext, "/?&=") {
		return ""
	}
	return ext
}

// extFromContentType 在 URL 无扩展名时，按响应 Content-Type 推断扩展名。
func extFromContentType(contentType string) string {
	ct := strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])
	switch strings.ToLower(ct) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/bmp":
		return ".bmp"
	case "video/mp4":
		return ".mp4"
	default:
		return ".bin"
	}
}
