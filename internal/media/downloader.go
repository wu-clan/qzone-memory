package media

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/qzone-memory/config"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/model"
	"github.com/qzone-memory/pkg/logger"
	"github.com/qzone-memory/pkg/ratelimit"
	"go.uber.org/zap"
)

var (
	downloadClient = &http.Client{Timeout: 60 * time.Second}
	inflight       sync.Map // url_hash -> 正在下载，避免并发重复拉取同一资源
)

// fetchAndStore 经限速器拉取并原子落盘，返回相对路径/大小/MIME。不触碰数据库，便于单测。
func fetchAndStore(ctx context.Context, asset *model.MediaAsset) (rel string, size int64, mimeType string, err error) {
	lim := ratelimit.Shared()
	if err = lim.Acquire(ctx); err != nil {
		return "", 0, "", err
	}
	defer lim.Release()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.SourceURL, nil)
	if err != nil {
		return "", 0, "", err
	}
	req.Header.Set("Referer", "https://user.qzone.qq.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")

	resp, err := downloadClient.Do(req)
	if err != nil {
		return "", 0, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", 0, "", fmt.Errorf("http %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return "", 0, "", err
	}

	mimeType = resp.Header.Get("Content-Type")
	ext := extFromURL(asset.SourceURL)
	if ext == "" {
		ext = extFromContentType(mimeType)
	}
	rel = relPathFor(asset.UserQQ, asset.Category, asset.URLHash, ext)
	if err = writeFile(rel, data); err != nil {
		return "", 0, "", err
	}
	return rel, int64(len(data)), mimeType, nil
}

// Download 下载单个待处理资源，带重试与状态回写，并发去重。成功返回 nil。
func Download(ctx context.Context, asset *model.MediaAsset) error {
	if _, loaded := inflight.LoadOrStore(asset.URLHash, struct{}{}); loaded {
		return nil // 同一 URL 已在下载
	}
	defer inflight.Delete(asset.URLHash)

	maxRetries := 3
	if config.GlobalConfig != nil && config.GlobalConfig.Media.MaxRetries > 0 {
		maxRetries = config.GlobalConfig.Media.MaxRetries
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		rel, size, mimeType, err := fetchAndStore(ctx, asset)
		if err == nil {
			return dao.MarkMediaAsset(ctx, asset.ID, map[string]any{
				"status":        "done",
				"local_path":    rel,
				"mime_type":     mimeType,
				"bytes":         size,
				"attempts":      attempt,
				"downloaded_at": time.Now(),
				"error":         "",
			})
		}
		lastErr = err
		if ctx.Err() != nil {
			break
		}
	}

	_ = dao.MarkMediaAsset(ctx, asset.ID, map[string]any{
		"status":   "failed",
		"attempts": maxRetries,
		"error":    errString(lastErr),
	})
	logger.Warn("媒体下载失败", zap.String("url", asset.SourceURL), zap.Error(lastErr))
	return lastErr
}

// DrainPending 下载某 QQ 全部待处理媒体，按配置并发；返回成功/失败数。
func DrainPending(ctx context.Context, userQQ string) (done, failed int, err error) {
	pending, err := dao.ListMediaAssetsByStatus(ctx, userQQ, "pending", 0)
	if err != nil {
		return 0, 0, err
	}
	if len(pending) == 0 {
		return 0, 0, nil
	}

	concurrency := 3
	if config.GlobalConfig != nil && config.GlobalConfig.Media.DownloadConcurrency > 0 {
		concurrency = config.GlobalConfig.Media.DownloadConcurrency
	}

	var doneN, failN int64
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	for _, asset := range pending {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(a *model.MediaAsset) {
			defer wg.Done()
			defer func() { <-sem }()
			if Download(ctx, a) == nil {
				atomic.AddInt64(&doneN, 1)
			} else {
				atomic.AddInt64(&failN, 1)
			}
		}(asset)
	}
	wg.Wait()
	return int(doneN), int(failN), ctx.Err()
}

// EnqueueAndDownload 登记单个 URL 为待下载并在后台立即拉取，用于解析器未命中时。
func EnqueueAndDownload(userQQ, rawURL, category string) {
	if userQQ == "" || !shouldDownload(rawURL) {
		return
	}
	asset := &model.MediaAsset{
		UserQQ:    userQQ,
		URLHash:   hashURL(rawURL),
		SourceURL: normalizeURL(rawURL),
		Category:  category,
		Status:    "pending",
	}
	if err := dao.BatchUpsertMediaAssets([]*model.MediaAsset{asset}); err != nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		fresh, err := dao.GetMediaAsset(ctx, userQQ, asset.URLHash)
		if err != nil || fresh.Status == "done" {
			return
		}
		_ = Download(ctx, fresh)
	}()
}

// errString 安全地把错误转成可入库的字符串。
func errString(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	if len(s) > 500 {
		s = s[:500]
	}
	return s
}
