// Package media 负责把 QQ 空间的远程图片/媒体下载到本地、按需解析为本地路径，
// 让档案在腾讯 URL 失效后依然完好，并使浏览过程逐步走向零外部请求。所有出站下载
// 与抓取共用 pkg/ratelimit 的全局限速器，统一克制频率，避免触发风控。
package media

import "errors"

// maxBytes 单个媒体的下载上限，防止异常超大文件占满磁盘。
const maxBytes = 64 << 20 // 64 MB

var errInvalidPath = errors.New("media: 非法的本地路径")
