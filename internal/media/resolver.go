package media

import (
	"context"

	"github.com/qzone-memory/internal/dao"
)

// Resolve 若某 URL 已成功本地化且文件仍在，返回其本地绝对路径；否则返回 ok=false，
// 调用方应降级为实时回源。
func Resolve(ctx context.Context, userQQ, rawURL string) (string, bool) {
	asset, err := dao.GetMediaAsset(ctx, userQQ, hashURL(rawURL))
	if err != nil || asset.Status != "done" || asset.LocalPath == "" {
		return "", false
	}
	if !fileExists(asset.LocalPath) {
		return "", false
	}
	abs, ok := absPath(asset.LocalPath)
	if !ok {
		return "", false
	}
	return abs, true
}
