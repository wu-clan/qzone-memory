package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/model"
	"github.com/qzone-memory/pkg/logger"
	"github.com/qzone-memory/pkg/response"
	"github.com/qzone-memory/qzone"
	"go.uber.org/zap"
)

type SyncProgress struct {
	Status      string    `json:"status"`
	CurrentType string    `json:"current_type"`
	TotalTypes  int       `json:"total_types"`
	DoneTypes   int       `json:"done_types"`
	Message     string    `json:"message"`
	Error       string    `json:"error,omitempty"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	FinishedAt  time.Time `json:"finished_at,omitempty"`
}

var (
	syncProgress = &SyncProgress{Status: "idle"}
	syncMu       sync.RWMutex
)

func StartSync(c *gin.Context) (map[string]string, *response.AppError) {
	var req dto.SyncRequest
	if err := bindJSON(c, &req); err != nil {
		return nil, err
	}
	if err := validateQQ(req.QQ); err != nil {
		return nil, err
	}

	syncMu.Lock()
	if syncProgress.Status == "running" {
		syncMu.Unlock()
		return nil, &response.AppError{Code: http.StatusConflict, Err: errors.New("同步任务正在进行中")}
	}
	user, err := dao.GetUserByQQ(req.QQ)
	if err != nil {
		syncMu.Unlock()
		return nil, &response.AppError{Code: http.StatusUnauthorized, Err: errors.New("授权失败，请重新登录")}
	}
	syncProgress = &SyncProgress{Status: "running", StartedAt: time.Now()}
	syncMu.Unlock()

	client := qzone.NewClient(user.Cookie, user.QQ)
	go runSync(client, req.QQ)

	return map[string]string{"message": "同步任务已启动"}, nil
}

func GetSyncProgress() (*SyncProgress, *response.AppError) {
	syncMu.RLock()
	defer syncMu.RUnlock()
	return syncProgress, nil
}

func updateSyncProgress(currentType, message string, doneTypes int) {
	syncMu.Lock()
	defer syncMu.Unlock()
	syncProgress.CurrentType = currentType
	syncProgress.Message = message
	syncProgress.DoneTypes = doneTypes
}

func runSync(client *qzone.Client, qq string) {
	defer func() {
		if r := recover(); r != nil {
			syncMu.Lock()
			syncProgress.Status = "error"
			syncProgress.Error = fmt.Sprintf("同步异常: %v", r)
			syncProgress.FinishedAt = time.Now()
			syncMu.Unlock()
		}
	}()

	// 同步前检测 Cookie 有效性
	if err := client.CheckCookie(); err != nil {
		syncMu.Lock()
		syncProgress.Status = "error"
		syncProgress.Error = fmt.Sprintf("Cookie 无效: %v", err)
		syncProgress.FinishedAt = time.Now()
		syncMu.Unlock()
		logger.Error("Cookie 验证失败，终止同步", zap.String("qq", qq), zap.Error(err))
		return
	}

	syncSteps := []struct {
		name string
		fn   func(*qzone.Client, string) (int, error)
	}{
		{"说说", syncTalks},
		{"日志", syncBlogs},
		{"相册", syncAlbums},
		{"留言", syncMessages},
		{"评论", syncComments},
		{"点赞", syncLikes},
		{"转发", syncShares},
	}
	syncMu.Lock()
	syncProgress.TotalTypes = len(syncSteps)
	syncMu.Unlock()
	failedSteps := make([]string, 0)
	for i, step := range syncSteps {
		updateSyncProgress(step.name, fmt.Sprintf("正在同步%s...", step.name), i)
		count, err := step.fn(client, qq)
		if err != nil {
			logger.Error("同步失败", zap.String("type", step.name), zap.Error(err))
			failedSteps = append(failedSteps, fmt.Sprintf("%s: %v", step.name, err))
			// Cookie 过期时立即终止，不再尝试后续步骤
			if errors.Is(err, qzone.ErrCookieExpired) {
				syncMu.Lock()
				syncProgress.Status = "error"
				syncProgress.Error = "Cookie 已过期，请重新登录后再同步"
				syncProgress.FinishedAt = time.Now()
				syncMu.Unlock()
				logger.Error("Cookie 过期，终止同步", zap.String("qq", qq))
				return
			}
		} else {
			logger.Info("同步步骤完成", zap.String("type", step.name), zap.Int("count", count))
			if count == 0 {
				logger.Warn("同步步骤返回空数据", zap.String("type", step.name), zap.String("qq", qq))
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	syncMu.Lock()
	syncProgress.CurrentType = ""
	syncProgress.DoneTypes = len(syncSteps)
	syncProgress.FinishedAt = time.Now()
	if len(failedSteps) > 0 {
		syncProgress.Status = "error"
		syncProgress.Message = "同步结束，但部分项目失败"
		syncProgress.Error = strings.Join(failedSteps, "；")
	} else {
		syncProgress.Status = "done"
		syncProgress.Message = "同步完成"
		syncProgress.Error = ""
	}
	syncMu.Unlock()
	if len(failedSteps) > 0 {
		logger.Warn("数据同步部分失败", zap.String("qq", qq), zap.Strings("failed_steps", failedSteps))
		return
	}
	logger.Info("数据同步完成", zap.String("qq", qq))
}

func syncTalks(client *qzone.Client, qq string) (int, error) {
	total := 0
	offset, limit := 0, 20
	for {
		items, err := client.GetTalks(offset, limit)
		if err != nil {
			return total, err
		}
		if len(items) == 0 {
			return total, nil
		}
		talks := make([]*model.Talk, 0, len(items))
		for _, item := range items {
			imagesJSON, _ := json.Marshal(item.Images)
			videosJSON, _ := json.Marshal(item.Videos)
			talks = append(talks, &model.Talk{
				UserQQ:       qq,
				TalkID:       item.TalkID,
				Content:      item.Content,
				Images:       string(imagesJSON),
				Videos:       string(videosJSON),
				Location:     item.Location,
				Device:       item.Device,
				LikeCount:    item.LikeCount,
				CommentCount: item.CommentCount,
				ShareCount:   item.ShareCount,
				PublishTime:  item.PublishTime,
			})
		}
		if err := dao.BatchUpsertTalks(talks); err != nil {
			logger.Error("保存说说失败", zap.Error(err))
		}
		total += len(items)
		offset += limit
		if len(items) < limit {
			return total, nil
		}
		time.Sleep(time.Second)
	}
}

func syncBlogs(client *qzone.Client, qq string) (int, error) {
	total := 0
	offset, limit := 0, 20
	for {
		items, err := client.GetBlogList(offset, limit)
		if err != nil {
			return total, err
		}
		if len(items) == 0 {
			return total, nil
		}
		blogs := make([]*model.Blog, 0, len(items))
		for _, item := range items {
			content, _ := client.GetBlogContent(item.BlogID)
			if content == "" {
				content = item.Summary
			}
			tagsJSON, _ := json.Marshal(item.Tags)
			blogs = append(blogs, &model.Blog{
				UserQQ:       qq,
				BlogID:       item.BlogID,
				Title:        item.Title,
				Content:      content,
				Summary:      item.Summary,
				Category:     item.Category,
				Tags:         string(tagsJSON),
				LikeCount:    item.LikeCount,
				CommentCount: item.CommentCount,
				ReadCount:    item.ReadCount,
				PublishTime:  item.PublishTime,
			})
		}
		if err := dao.BatchUpsertBlogs(blogs); err != nil {
			logger.Error("保存日志失败", zap.Error(err))
		}
		total += len(items)
		offset += limit
		if len(items) < limit {
			return total, nil
		}
		time.Sleep(time.Second)
	}
}

func syncAlbums(client *qzone.Client, qq string) (int, error) {
	items, err := client.GetAlbums()
	if err != nil {
		return 0, err
	}
	albums := make([]*model.Album, 0, len(items))
	for _, item := range items {
		albums = append(albums, &model.Album{
			UserQQ:      qq,
			AlbumID:     item.AlbumID,
			Name:        item.Name,
			Description: item.Description,
			CoverURL:    item.CoverURL,
			PhotoCount:  item.PhotoCount,
			CreateTime:  item.CreateTime,
		})
	}
	if err := dao.BatchUpsertAlbums(albums); err != nil {
		logger.Error("保存相册失败", zap.Error(err))
	}
	total := len(items)
	for _, item := range items {
		if err := syncPhotos(client, qq, item.AlbumID); err != nil {
			logger.Error("同步照片失败", zap.String("album", item.AlbumID), zap.Error(err))
		}
	}
	return total, nil
}

func syncPhotos(client *qzone.Client, qq, albumID string) error {
	offset, limit := 0, 50
	for {
		items, err := client.GetPhotos(albumID, offset, limit)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		photos := make([]*model.Photo, 0, len(items))
		for _, item := range items {
			photos = append(photos, &model.Photo{
				UserQQ:      qq,
				PhotoID:     item.PhotoID,
				AlbumID:     item.AlbumID,
				Name:        item.Name,
				Description: item.Description,
				URL:         item.URL,
				ThumbURL:    item.ThumbURL,
				Width:       item.Width,
				Height:      item.Height,
				PhotoTime:   item.PhotoTime,
			})
		}
		if err := dao.BatchUpsertPhotos(photos); err != nil {
			logger.Error("保存照片失败", zap.Error(err))
		}
		offset += limit
		if len(items) < limit {
			return nil
		}
	}
}

func syncMessages(client *qzone.Client, qq string) (int, error) {
	total := 0
	offset, limit := 0, 20
	for {
		items, err := client.GetMessages(offset, limit)
		if err != nil {
			return total, err
		}
		if len(items) == 0 {
			return total, nil
		}
		messages := make([]*model.Message, 0, len(items))
		for _, item := range items {
			messages = append(messages, &model.Message{
				UserQQ:       qq,
				MessageID:    item.MessageID,
				AuthorQQ:     item.AuthorQQ,
				AuthorName:   item.AuthorName,
				AuthorAvatar: item.AuthorAvatar,
				Content:      item.Content,
				ReplyContent: item.ReplyContent,
				MessageTime:  item.MessageTime,
			})
		}
		if err := dao.BatchUpsertMessages(messages); err != nil {
			logger.Error("保存留言失败", zap.Error(err))
		}
		total += len(items)
		offset += limit
		if len(items) < limit {
			return total, nil
		}
	}
}

func syncComments(client *qzone.Client, qq string) (int, error) {
	total := 0
	talks, err := dao.ListTalkSummaries(qq)
	if err != nil {
		return 0, fmt.Errorf("获取说说列表失败: %w", err)
	}

	for _, talk := range talks {
		if talk.CommentCount == 0 {
			continue
		}
		offset := 0
		for {
			items, err := client.GetTalkComments(talk.TalkID, offset, 20)
			if err != nil {
				logger.Warn("获取说说评论失败", zap.String("talk", talk.TalkID), zap.Error(err))
				break
			}
			if len(items) == 0 {
				break
			}
			comments := make([]*model.Comment, 0, len(items))
			for _, item := range items {
				comments = append(comments, &model.Comment{
					UserQQ:      qq,
					CommentID:   item.CommentID,
					TargetType:  "talk",
					TargetID:    talk.TalkID,
					AuthorQQ:    item.AuthorQQ,
					AuthorName:  item.AuthorName,
					Content:     item.Content,
					ReplyToQQ:   item.ReplyToQQ,
					ReplyToName: item.ReplyToName,
					CommentTime: item.CommentTime,
				})
			}
			if err := dao.BatchUpsertComments(comments); err != nil {
				logger.Error("保存评论失败", zap.Error(err))
			}
			offset += 20
			if len(items) < 20 {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		time.Sleep(200 * time.Millisecond)
	}

	blogs, err := dao.ListBlogSummaries(qq)
	if err != nil {
		return 0, fmt.Errorf("获取日志列表失败: %w", err)
	}
	for _, blog := range blogs {
		if blog.CommentCount == 0 {
			continue
		}
		offset := 0
		for {
			items, err := client.GetBlogComments(blog.BlogID, offset, 50)
			if err != nil {
				logger.Warn("获取日志评论失败", zap.String("blog", blog.BlogID), zap.Error(err))
				break
			}
			if len(items) == 0 {
				break
			}
			comments := make([]*model.Comment, 0, len(items))
			for _, item := range items {
				comments = append(comments, &model.Comment{
					UserQQ:       qq,
					CommentID:    item.CommentID,
					TargetType:   "blog",
					TargetID:     blog.BlogID,
					AuthorQQ:     item.AuthorQQ,
					AuthorName:   item.AuthorName,
					AuthorAvatar: item.AuthorAvatar,
					Content:      item.Content,
					ReplyToQQ:    item.ReplyToQQ,
					ReplyToName:  item.ReplyToName,
					CommentTime:  item.CommentTime,
				})
			}
			if err := dao.BatchUpsertComments(comments); err != nil {
				logger.Error("保存日志评论失败", zap.Error(err))
			}
			offset += 50
			if len(items) < 50 {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		time.Sleep(200 * time.Millisecond)
	}

	photos, err := dao.ListPhotoSummaries(qq)
	if err != nil {
		return 0, fmt.Errorf("获取照片列表失败: %w", err)
	}
	for _, photo := range photos {
		offset := 0
		for {
			items, err := client.GetPhotoComments(photo.AlbumID, photo.PhotoID, offset, 50)
			if err != nil {
				logger.Warn("获取照片评论失败", zap.String("album", photo.AlbumID), zap.String("photo", photo.PhotoID), zap.Error(err))
				break
			}
			if len(items) == 0 {
				break
			}
			comments := make([]*model.Comment, 0, len(items))
			for _, item := range items {
				comments = append(comments, &model.Comment{
					UserQQ:       qq,
					CommentID:    item.CommentID,
					TargetType:   "photo",
					TargetID:     photo.PhotoID,
					AuthorQQ:     item.AuthorQQ,
					AuthorName:   item.AuthorName,
					AuthorAvatar: item.AuthorAvatar,
					Content:      item.Content,
					ReplyToQQ:    item.ReplyToQQ,
					ReplyToName:  item.ReplyToName,
					CommentTime:  item.CommentTime,
				})
			}
			if err := dao.BatchUpsertComments(comments); err != nil {
				logger.Error("保存照片评论失败", zap.Error(err))
			}
			offset += 50
			if len(items) < 50 {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		time.Sleep(200 * time.Millisecond)
	}

	return total, nil
}

func syncLikes(client *qzone.Client, qq string) (int, error) {
	total := 0
	talks, err := dao.ListTalkSummaries(qq)
	if err != nil {
		return 0, fmt.Errorf("获取说说列表失败: %w", err)
	}

	for _, talk := range talks {
		if talk.LikeCount == 0 {
			continue
		}
		items, err := client.GetTalkLikes(talk.TalkID)
		if err != nil {
			logger.Warn("获取说说点赞失败", zap.String("talk", talk.TalkID), zap.Error(err))
			time.Sleep(200 * time.Millisecond)
			continue
		}
		likes := make([]*model.Like, 0, len(items))
		for _, item := range items {
			likeID := fmt.Sprintf("talk_%s_%s", talk.TalkID, item.LikerQQ)
			if likeID == fmt.Sprintf("talk_%s_", talk.TalkID) {
				likeID = fmt.Sprintf("talk_%s_%d", talk.TalkID, item.LikeTime.Unix())
			}
			likes = append(likes, &model.Like{
				UserQQ:      qq,
				LikeID:      likeID,
				TargetType:  "talk",
				TargetID:    talk.TalkID,
				LikerQQ:     item.LikerQQ,
				LikerName:   item.LikerName,
				LikerAvatar: item.LikerAvatar,
				LikeTime:    item.LikeTime,
			})
		}
		if err := dao.BatchUpsertLikes(likes); err != nil {
			logger.Error("保存点赞失败", zap.Error(err))
		}
		total += len(items)
		time.Sleep(200 * time.Millisecond)
	}

	return total, nil
}

func syncShares(client *qzone.Client, qq string) (int, error) {
	total := 0
	talks, err := dao.ListTalkSummaries(qq)
	if err != nil {
		return 0, fmt.Errorf("获取说说列表失败: %w", err)
	}

	for _, talk := range talks {
		if talk.ShareCount == 0 {
			continue
		}
		offset := 0
		for {
			items, err := client.GetShares("talk", talk.TalkID, offset, 20)
			if err != nil {
				logger.Warn("获取说说转发失败", zap.String("talk", talk.TalkID), zap.Error(err))
				break
			}
			if len(items) == 0 {
				break
			}
			shares := make([]*model.Share, 0, len(items))
			for _, item := range items {
				shareID := fmt.Sprintf("talk_%s_%s_%d", talk.TalkID, item.SharerQQ, item.ShareTime.Unix())
				if shareID == fmt.Sprintf("talk_%s__0", talk.TalkID) {
					shareID = "talk_" + item.ShareID
				}
				shares = append(shares, &model.Share{
					UserQQ:     qq,
					ShareID:    shareID,
					TargetType: "talk",
					TargetID:   talk.TalkID,
					SharerQQ:   item.SharerQQ,
					SharerName: item.SharerName,
					Comment:    item.Comment,
					ShareTime:  item.ShareTime,
				})
			}
			if err := dao.BatchUpsertShares(shares); err != nil {
				logger.Error("保存转发失败", zap.Error(err))
			}
			offset += 20
			if len(items) < 20 {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		time.Sleep(200 * time.Millisecond)
	}

	return total, nil
}
