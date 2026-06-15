package media

import "strings"

// decorativeMarkers QQ 空间的装饰素材 / 表情 / 点赞图标等非用户内容的特征片段。
var decorativeMarkers = []string{
	"qzonestyle.gtimg.cn",
	"/space_item/",
	"custompraise",
	"/emotion/",
	"/qzone/em/",
	"/com_attr/",
}

// shouldDownload 判断一个媒体 URL 是否值得本地化。
//
// 注意：这与前端「正文配图」过滤规则不同——下载侧要保留头像（好友/访客/评论者头像
// 也是档案的一部分），只剔除纯装饰素材。两份规则各自维护，不要混用。
func shouldDownload(raw string) bool {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" || !isTencentURL(v) {
		return false
	}
	for _, marker := range decorativeMarkers {
		if strings.Contains(v, marker) {
			return false
		}
	}
	return true
}
