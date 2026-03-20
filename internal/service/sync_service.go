package service

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qzone-memory/database"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
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
		{"动态归档", syncActivities},
		{"好友", syncFriends},
		{"访客", syncVisitors},
		{"视频", syncVideos},
		{"收藏", syncFavorites},
		{"私密日记", syncDiaries},
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

func syncFriends(client *qzone.Client, qq string) (int, error) {
	friendItems, groupItems, err := client.GetFriends()
	if err != nil {
		return 0, err
	}

	groups := make([]*model.FriendGroup, 0, len(groupItems)+1)
	groupIDs := make([]int, 0, len(groupItems)+1)
	for _, item := range groupItems {
		groups = append(groups, &model.FriendGroup{
			UserQQ:  qq,
			GroupID: item.GroupID,
			Name:    item.Name,
		})
		groupIDs = append(groupIDs, item.GroupID)
	}
	groups = append(groups, &model.FriendGroup{
		UserQQ:  qq,
		GroupID: -1,
		Name:    "历史互动",
	})
	groupIDs = append(groupIDs, -1)
	if err := dao.BatchUpsertFriendGroups(groups); err != nil {
		return 0, err
	}
	if err := dao.MarkMissingGroupsDeleted(qq, groupIDs); err != nil {
		return 0, err
	}

	specialCareList, err := client.GetSpecialCareFriends()
	if err != nil {
		logger.Warn("获取特别关心失败", zap.String("qq", qq), zap.Error(err))
	}
	specialCareMap := make(map[string]struct{}, len(specialCareList))
	for _, item := range specialCareList {
		specialCareMap[item.FriendQQ] = struct{}{}
	}

	friends := make([]*model.Friend, 0, len(friendItems))
	currentQQs := make([]string, 0, len(friendItems))
	for idx, item := range friendItems {
		friend := &model.Friend{
			UserQQ:     qq,
			FriendQQ:   item.FriendQQ,
			Name:       firstNonEmpty(item.Remark, item.Name),
			Remark:     item.Remark,
			Avatar:     qzone.GetPortrait(item.FriendQQ),
			GroupID:    item.GroupID,
			GroupName:  item.GroupName,
			IsCurrent:  true,
			IsDeleted:  false,
			SourceType: "friend_api",
			Online:     item.Online,
			Yellow:     item.Yellow,
			LastSeenAt: time.Now(),
		}
		if _, ok := specialCareMap[item.FriendQQ]; ok {
			friend.IsSpecialCare = true
		}
		if idx < 20 {
			addTime, commonGroup, friendshipErr := client.GetFriendship(item.FriendQQ)
			if friendshipErr == nil {
				friend.AddTime = addTime
				friend.CommonGroup = commonGroup
			}
		}
		friends = append(friends, friend)
		currentQQs = append(currentQQs, item.FriendQQ)
	}

	if err := dao.BatchUpsertFriends(friends); err != nil {
		return 0, err
	}
	if err := dao.MarkCurrentFriendsDeleted(qq, currentQQs); err != nil {
		return 0, err
	}

	if err := inferHistoricalFriends(qq); err != nil {
		logger.Warn("推断历史好友失败", zap.String("qq", qq), zap.Error(err))
	}

	return len(friendItems), nil
}

func syncVisitors(client *qzone.Client, qq string) (int, error) {
	items, err := client.GetVisitors()
	if err != nil {
		return 0, err
	}
	records := make([]*model.Visitor, 0, len(items))
	for _, item := range items {
		visitorID := item.VisitorQQ
		if visitorID == "" {
			visitorID = fmt.Sprintf("visitor_%d", item.VisitTime.Unix())
		}
		records = append(records, &model.Visitor{
			UserQQ:      qq,
			VisitorID:   visitorID + "_" + strconv.FormatInt(item.VisitTime.Unix(), 10),
			VisitorQQ:   item.VisitorQQ,
			VisitorName: item.VisitorName,
			Avatar:      qzone.GetPortrait(item.VisitorQQ),
			Source:      item.Source,
			IsHidden:    item.IsHidden,
			YellowLevel: item.YellowLevel,
			VisitTime:   item.VisitTime,
		})
	}
	if err := dao.BatchUpsertVisitors(records); err != nil {
		return 0, err
	}
	return len(records), nil
}

func syncVideos(client *qzone.Client, qq string) (int, error) {
	total := 0
	offset, limit := 0, 20
	for {
		items, err := client.GetVideos(offset, limit)
		if err != nil {
			if total > 0 {
				return total, nil
			}
			return total, err
		}
		if len(items) == 0 {
			return total, nil
		}
		records := make([]*model.Video, 0, len(items))
		for _, item := range items {
			records = append(records, &model.Video{
				UserQQ:       qq,
				VideoID:      item.VideoID,
				Title:        item.Title,
				Description:  item.Description,
				URL:          item.URL,
				PreviewURL:   item.PreviewURL,
				Width:        item.Width,
				Height:       item.Height,
				Duration:     item.Duration,
				CommentCount: item.CommentCount,
				UploadTime:   item.UploadTime,
			})
		}
		if err := dao.BatchUpsertVideos(records); err != nil {
			return total, err
		}
		total += len(records)
		if len(items) < limit {
			return total, nil
		}
		offset += limit
		time.Sleep(300 * time.Millisecond)
	}
}

func syncFavorites(client *qzone.Client, qq string) (int, error) {
	total := 0
	offset, limit := 0, 20
	for {
		items, err := client.GetFavorites(offset, limit)
		if err != nil {
			if total > 0 {
				return total, nil
			}
			return total, err
		}
		if len(items) == 0 {
			return total, nil
		}
		records := make([]*model.Favorite, 0, len(items))
		for _, item := range items {
			records = append(records, &model.Favorite{
				UserQQ:     qq,
				FavoriteID: item.FavoriteID,
				Type:       item.Type,
				Title:      item.Title,
				Abstract:   item.Abstract,
				URL:        item.URL,
				OwnerQQ:    item.OwnerQQ,
				OwnerName:  item.OwnerName,
				Images:     mustJSON(item.Images),
				CreateTime: item.CreateTime,
			})
		}
		if err := dao.BatchUpsertFavorites(records); err != nil {
			return total, err
		}
		total += len(records)
		if len(items) < limit {
			return total, nil
		}
		offset += limit
		time.Sleep(300 * time.Millisecond)
	}
}

func syncDiaries(client *qzone.Client, qq string) (int, error) {
	total := 0
	offset, limit := 0, 15
	for {
		items, err := client.GetDiaries(offset, limit)
		if err != nil {
			if total > 0 {
				return total, nil
			}
			return total, err
		}
		if len(items) == 0 {
			return total, nil
		}
		records := make([]*model.Diary, 0, len(items))
		for _, item := range items {
			content, _ := client.GetDiaryDetail(item.DiaryID)
			records = append(records, &model.Diary{
				UserQQ:     qq,
				DiaryID:    item.DiaryID,
				Title:      item.Title,
				Summary:    item.Summary,
				Content:    firstNonEmpty(content, item.Summary),
				CreateTime: item.CreateTime,
			})
		}
		if err := dao.BatchUpsertDiaries(records); err != nil {
			return total, err
		}
		total += len(records)
		if len(items) < limit {
			return total, nil
		}
		offset += limit
		time.Sleep(300 * time.Millisecond)
	}
}

func syncActivities(client *qzone.Client, qq string) (int, error) {
	total := 0
	offset, limit := 0, 40
	repeatedPages := 0

	for page := 0; page < 500; page++ {
		items, err := client.GetFeeds(offset, limit)
		if err != nil {
			if total > 0 {
				logger.Warn("动态归档中断，保留已抓取数据", zap.String("qq", qq), zap.Int("saved", total), zap.Error(err))
				return total, nil
			}
			return total, err
		}
		if len(items) == 0 {
			return total, nil
		}

		activities := make([]*model.Activity, 0, len(items))
		for _, item := range items {
			imagesJSON, _ := json.Marshal(item.Images)
			activities = append(activities, &model.Activity{
				UserQQ:       qq,
				FeedID:       item.FeedID,
				FeedType:     item.FeedType,
				ObjectID:     item.ObjectID,
				Title:        item.Title,
				Content:      item.Content,
				HTMLContent:  item.HTMLContent,
				AuthorQQ:     item.AuthorQQ,
				AuthorName:   item.AuthorName,
				Images:       string(imagesJSON),
				LikeCount:    item.LikeCount,
				CommentCount: item.CommentCount,
				ShareCount:   item.ShareCount,
				IsDeleted:    item.IsDeleted,
				PublishTime:  item.PublishTime,
				StateText:    item.StateText,
			})
		}
		if err := dao.BatchUpsertActivities(activities); err != nil {
			return total, err
		}
		reconstructHistoricalObjects(qq, items)
		backfillHistoricalInteractions(qq)

		total += len(activities)

		if pageHasNoProgress(items) {
			repeatedPages++
			if repeatedPages >= 2 {
				return total, nil
			}
		} else {
			repeatedPages = 0
		}

		offset += limit
		qzoneSleepWithJitter(900*time.Millisecond, 700*time.Millisecond)
	}

	return total, nil
}

func inferHistoricalFriends(userQQ string) error {
	var currentFriends []model.Friend
	if err := database.DB.Where("user_qq = ? AND is_current = ?", userQQ, true).Find(&currentFriends).Error; err != nil {
		return err
	}
	currentMap := make(map[string]struct{}, len(currentFriends))
	for _, item := range currentFriends {
		currentMap[item.FriendQQ] = struct{}{}
	}

	type candidate struct {
		qq   string
		name string
		time time.Time
		src  string
	}

	candidates := make(map[string]*candidate)
	push := func(qq, name, src string, ts time.Time) {
		if strings.TrimSpace(qq) == "" || qq == userQQ {
			return
		}
		existing, ok := candidates[qq]
		if !ok {
			candidates[qq] = &candidate{qq: qq, name: name, time: ts, src: src}
			return
		}
		if ts.After(existing.time) {
			existing.time = ts
		}
		if existing.name == "" && name != "" {
			existing.name = name
		}
	}

	var activities []model.Activity
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&activities).Error; err != nil {
		return err
	}
	for _, item := range activities {
		push(item.AuthorQQ, item.AuthorName, "activity", item.PublishTime)
	}

	var messages []model.Message
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&messages).Error; err != nil {
		return err
	}
	for _, item := range messages {
		push(item.AuthorQQ, item.AuthorName, "message", item.MessageTime)
	}

	var comments []model.Comment
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&comments).Error; err != nil {
		return err
	}
	for _, item := range comments {
		push(item.AuthorQQ, item.AuthorName, "comment", item.CommentTime)
	}

	var likes []model.Like
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&likes).Error; err != nil {
		return err
	}
	for _, item := range likes {
		push(item.LikerQQ, item.LikerName, "like", item.LikeTime)
	}

	var shares []model.Share
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&shares).Error; err != nil {
		return err
	}
	for _, item := range shares {
		push(item.SharerQQ, item.SharerName, "share", item.ShareTime)
	}

	var mentions []model.Mention
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&mentions).Error; err != nil {
		return err
	}
	for _, item := range mentions {
		push(item.AuthorQQ, item.AuthorName, "mention", item.MentionTime)
	}

	var visitors []model.Visitor
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&visitors).Error; err != nil {
		return err
	}
	for _, item := range visitors {
		push(item.VisitorQQ, item.VisitorName, "visitor", item.VisitTime)
	}

	var favorites []model.Favorite
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&favorites).Error; err != nil {
		return err
	}
	for _, item := range favorites {
		push(item.OwnerQQ, item.OwnerName, "favorite", item.CreateTime)
	}

	historical := make([]*model.Friend, 0, len(candidates))
	for _, item := range candidates {
		if _, ok := currentMap[item.qq]; ok {
			continue
		}
		historical = append(historical, &model.Friend{
			UserQQ:        userQQ,
			FriendQQ:      item.qq,
			Name:          firstNonEmpty(item.name, item.qq),
			Avatar:        qzone.GetPortrait(item.qq),
			GroupID:       -1,
			GroupName:     "历史互动",
			IsCurrent:     false,
			IsDeleted:     true,
			SourceType:    item.src,
			InteractCount: 1,
			LastSeenAt:    item.time,
		})
	}
	return dao.BatchUpsertFriends(historical)
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

func reconstructHistoricalObjects(qq string, feeds []qzone.FeedItem) {
	talks := make([]*model.Talk, 0)
	blogs := make([]*model.Blog, 0)
	albums := make([]*model.Album, 0)
	photos := make([]*model.Photo, 0)
	likes := make([]*model.Like, 0)
	mentions := make([]*model.Mention, 0)
	messages := make([]*model.Message, 0)

	for _, item := range feeds {
		switch item.FeedType {
		case "talk":
			talkID := stableObjectID("talk", item)
			talks = append(talks, &model.Talk{
				UserQQ:       qq,
				TalkID:       talkID,
				Content:      item.Content,
				Images:       mustJSON(item.Images),
				Videos:       "[]",
				IsDeleted:    true,
				LikeCount:    item.LikeCount,
				CommentCount: item.CommentCount,
				ShareCount:   item.ShareCount,
				PublishTime:  item.PublishTime,
			})
			if strings.Contains(item.StateText, "赞了我的说说") {
				likes = append(likes, &model.Like{
					UserQQ:      qq,
					LikeID:      stableEventID("like", item),
					TargetType:  "talk",
					TargetID:    talkID,
					LikerQQ:     item.AuthorQQ,
					LikerName:   firstNonEmpty(item.AuthorName, item.AuthorQQ),
					LikerAvatar: qzone.GetPortrait(item.AuthorQQ),
					LikeTime:    item.PublishTime,
				})
			}
		case "blog":
			content := item.Content
			if content == "" {
				content = item.Title
			}
			blogs = append(blogs, &model.Blog{
				UserQQ:       qq,
				BlogID:       stableObjectID("blog", item),
				Title:        firstNonEmpty(item.Title, truncateString(content, 60)),
				Content:      content,
				Summary:      truncateString(content, 160),
				IsDeleted:    true,
				LikeCount:    item.LikeCount,
				CommentCount: item.CommentCount,
				PublishTime:  item.PublishTime,
			})
		case "photo":
			albumID := stableObjectID("album", item)
			albums = append(albums, &model.Album{
				UserQQ:      qq,
				AlbumID:     albumID,
				Name:        firstNonEmpty(item.Title, truncateString(item.Content, 60), "历史相册"),
				Description: truncateString(item.Content, 200),
				CoverURL:    firstImage(item.Images),
				PhotoCount:  maxInt(len(item.Images), 1),
				IsDeleted:   true,
				CreateTime:  item.PublishTime,
			})
			if len(item.Images) > 0 {
				photos = append(photos, &model.Photo{
					UserQQ:      qq,
					PhotoID:     stableObjectID("photo", item),
					AlbumID:     albumID,
					Name:        firstNonEmpty(item.Title, "历史照片"),
					Description: truncateString(item.Content, 200),
					URL:         item.Images[0],
					ThumbURL:    item.Images[0],
					IsDeleted:   true,
					PhotoTime:   item.PublishTime,
				})
			}
		case "message":
			messageID := stableObjectID("message", item)
			messages = append(messages, &model.Message{
				UserQQ:       qq,
				MessageID:    messageID,
				AuthorQQ:     item.AuthorQQ,
				AuthorName:   firstNonEmpty(item.AuthorName, item.AuthorQQ),
				AuthorAvatar: qzone.GetPortrait(item.AuthorQQ),
				Content:      item.Content,
				IsDeleted:    true,
				MessageTime:  item.PublishTime,
			})
			if strings.Contains(item.StateText, "提到我") {
				mentions = append(mentions, &model.Mention{
					UserQQ:      qq,
					MentionID:   stableEventID("mention", item),
					SourceType:  "message",
					SourceID:    messageID,
					AuthorQQ:    item.AuthorQQ,
					AuthorName:  firstNonEmpty(item.AuthorName, item.AuthorQQ),
					Content:     item.Content,
					MentionTime: item.PublishTime,
				})
			}
		}
	}

	if err := dao.BatchUpsertTalks(talks); err != nil {
		logger.Warn("归档重建说说失败", zap.String("qq", qq), zap.Error(err))
	}
	if err := dao.BatchUpsertBlogs(blogs); err != nil {
		logger.Warn("归档重建日志失败", zap.String("qq", qq), zap.Error(err))
	}
	if err := dao.BatchUpsertAlbums(albums); err != nil {
		logger.Warn("归档重建相册失败", zap.String("qq", qq), zap.Error(err))
	}
	if err := dao.BatchUpsertPhotos(photos); err != nil {
		logger.Warn("归档重建照片失败", zap.String("qq", qq), zap.Error(err))
	}
	if err := dao.BatchUpsertLikes(likes); err != nil {
		logger.Warn("归档重建点赞失败", zap.String("qq", qq), zap.Error(err))
	}
	if err := dao.BatchUpsertMentions(mentions); err != nil {
		logger.Warn("归档重建提及失败", zap.String("qq", qq), zap.Error(err))
	}
	if err := dao.BatchUpsertMessages(messages); err != nil {
		logger.Warn("归档重建留言失败", zap.String("qq", qq), zap.Error(err))
	}
}

func backfillHistoricalInteractions(qq string) {
	var activities []model.Activity
	if err := database.DB.Where("user_qq = ?", qq).Find(&activities).Error; err != nil {
		logger.Warn("读取动态归档失败", zap.String("qq", qq), zap.Error(err))
		return
	}

	likes := make([]*model.Like, 0)
	comments := make([]*model.Comment, 0)
	mentions := make([]*model.Mention, 0)
	messages := make([]*model.Message, 0)
	shares := make([]*model.Share, 0)

	for _, item := range activities {
		switch {
		case item.FeedType == "talk" && strings.Contains(item.StateText, "赞了我的说说"):
			likes = append(likes, &model.Like{
				UserQQ:      qq,
				LikeID:      "like_" + firstNonEmpty(item.FeedID, item.ObjectID),
				TargetType:  "talk",
				TargetID:    activityTargetID("talk", item),
				LikerQQ:     item.AuthorQQ,
				LikerName:   firstNonEmpty(item.AuthorName, item.AuthorQQ),
				LikerAvatar: qzone.GetPortrait(item.AuthorQQ),
				LikeTime:    item.PublishTime,
			})
		case isHistoricalCommentState(item):
			targetType := firstNonEmpty(item.FeedType, "talk")
			targetID := activityTargetID(targetType, item)
			commentDetail := extractHistoricalCommentDetail(item)
			authorQQ := firstNonEmpty(commentDetail.AuthorQQ, item.AuthorQQ)
			authorName := firstNonEmpty(commentDetail.AuthorName, item.AuthorName, authorQQ)
			content := firstNonEmpty(commentDetail.Content, item.Content, item.Title, "评论了你的历史内容")
			comments = append(comments, &model.Comment{
				UserQQ:       qq,
				CommentID:    "comment_" + firstNonEmpty(item.FeedID, item.ObjectID),
				TargetType:   targetType,
				TargetID:     targetID,
				AuthorQQ:     authorQQ,
				AuthorName:   authorName,
				AuthorAvatar: qzone.GetPortrait(authorQQ),
				Content:      content,
				ReplyToQQ:    commentDetail.ReplyToQQ,
				ReplyToName:  commentDetail.ReplyToName,
				IsDeleted:    true,
				CommentTime:  item.PublishTime,
			})
		case item.FeedType == "message" || strings.Contains(item.StateText, "留言提到我"):
			messageID := activityTargetID("message", item)
			messages = append(messages, &model.Message{
				UserQQ:       qq,
				MessageID:    messageID,
				AuthorQQ:     item.AuthorQQ,
				AuthorName:   firstNonEmpty(item.AuthorName, item.AuthorQQ),
				AuthorAvatar: qzone.GetPortrait(item.AuthorQQ),
				Content:      item.Content,
				IsDeleted:    true,
				MessageTime:  item.PublishTime,
			})
			if strings.Contains(item.StateText, "提到我") {
				mentions = append(mentions, &model.Mention{
					UserQQ:      qq,
					MentionID:   "mention_" + firstNonEmpty(item.FeedID, item.ObjectID),
					SourceType:  "message",
					SourceID:    messageID,
					AuthorQQ:    item.AuthorQQ,
					AuthorName:  firstNonEmpty(item.AuthorName, item.AuthorQQ),
					Content:     item.Content,
					MentionTime: item.PublishTime,
				})
			}
		case strings.Contains(item.StateText, "转发") || strings.Contains(item.Title, "转发"):
			targetType := firstNonEmpty(item.FeedType, "share")
			shares = append(shares, &model.Share{
				UserQQ:     qq,
				ShareID:    "share_" + firstNonEmpty(item.FeedID, item.ObjectID),
				TargetType: targetType,
				TargetID:   activityTargetID(targetType, item),
				SharerQQ:   item.AuthorQQ,
				SharerName: firstNonEmpty(item.AuthorName, item.AuthorQQ),
				Comment:    firstNonEmpty(item.Content, item.Title, "转发了你的历史内容"),
				ShareTime:  item.PublishTime,
			})
		}
	}

	if err := dao.BatchUpsertLikes(likes); err != nil {
		logger.Warn("回填历史点赞失败", zap.String("qq", qq), zap.Error(err))
	}
	if err := dao.BatchUpsertComments(comments); err != nil {
		logger.Warn("回填历史评论失败", zap.String("qq", qq), zap.Error(err))
	}
	if err := dao.BatchUpsertMentions(mentions); err != nil {
		logger.Warn("回填历史提及失败", zap.String("qq", qq), zap.Error(err))
	}
	if err := dao.BatchUpsertMessages(messages); err != nil {
		logger.Warn("回填历史留言失败", zap.String("qq", qq), zap.Error(err))
	}
	if err := dao.BatchUpsertShares(shares); err != nil {
		logger.Warn("回填历史转发失败", zap.String("qq", qq), zap.Error(err))
	}
	if err := dedupeHistoricalLikes(qq); err != nil {
		logger.Warn("清理历史点赞重复记录失败", zap.String("qq", qq), zap.Error(err))
	}
}

func pageHasNoProgress(items []qzone.FeedItem) bool {
	if len(items) == 0 {
		return true
	}
	same := 0
	for _, item := range items {
		if strings.TrimSpace(item.ObjectID) == "" && strings.TrimSpace(item.Content) == "" && strings.TrimSpace(item.Title) == "" {
			same++
		}
	}
	return same == len(items)
}

func stableObjectID(prefix string, item qzone.FeedItem) string {
	if item.ObjectID != "" {
		return prefix + "_" + item.ObjectID
	}
	if item.FeedID != "" {
		return prefix + "_" + item.FeedID
	}
	sum := md5.Sum([]byte(strings.Join([]string{
		prefix,
		item.AuthorQQ,
		item.Title,
		item.Content,
		item.PublishTime.Format(time.RFC3339),
	}, "|")))
	return prefix + "_" + hex.EncodeToString(sum[:])
}

func stableEventID(prefix string, item qzone.FeedItem) string {
	if item.FeedID != "" {
		return prefix + "_" + item.FeedID
	}
	sum := md5.Sum([]byte(strings.Join([]string{
		prefix,
		item.ObjectID,
		item.AuthorQQ,
		item.Title,
		item.Content,
		item.PublishTime.Format(time.RFC3339),
	}, "|")))
	return prefix + "_" + hex.EncodeToString(sum[:])
}

func activityTargetID(prefix string, item model.Activity) string {
	if item.ObjectID != "" {
		return prefix + "_" + item.ObjectID
	}
	if item.FeedID != "" {
		return prefix + "_" + item.FeedID
	}
	sum := md5.Sum([]byte(strings.Join([]string{
		prefix,
		item.AuthorQQ,
		item.Title,
		item.Content,
		item.PublishTime.Format(time.RFC3339),
	}, "|")))
	return prefix + "_" + hex.EncodeToString(sum[:])
}

func dedupeHistoricalLikes(qq string) error {
	return database.DB.Exec(`
		DELETE FROM likes
		WHERE user_qq = ?
		  AND id NOT IN (
			SELECT MAX(id)
			FROM likes
			WHERE user_qq = ?
			GROUP BY target_id, liker_qq, like_time
		  )
	`, qq, qq).Error
}

type historicalCommentDetail struct {
	AuthorQQ    string
	AuthorName  string
	Content     string
	ReplyToQQ   string
	ReplyToName string
}

func isHistoricalCommentState(item model.Activity) bool {
	state := strings.TrimSpace(item.StateText)
	return strings.Contains(state, "评论") ||
		strings.Contains(state, "回复") ||
		strings.Contains(item.Title, "评论") ||
		strings.Contains(item.Title, "回复")
}

func extractHistoricalCommentDetail(item model.Activity) historicalCommentDetail {
	if strings.TrimSpace(item.HTMLContent) == "" {
		return historicalCommentDetail{}
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<div>" + item.HTMLContent + "</div>"))
	if err != nil {
		return historicalCommentDetail{}
	}

	bestScore := -1
	best := historicalCommentDetail{}
	doc.Find("li.comments-item").Each(func(_ int, s *goquery.Selection) {
		detail, score := buildHistoricalCommentDetail(s, item)
		if score > bestScore {
			bestScore = score
			best = detail
		}
	})

	if bestScore < 0 {
		return historicalCommentDetail{}
	}
	return best
}

func buildHistoricalCommentDetail(node *goquery.Selection, item model.Activity) (historicalCommentDetail, int) {
	score := 0

	timeText := cleanHistoricalText(node.Find("div.comments-op span.state").First().Text())
	if historicalCommentTimeMatches(item.PublishTime, timeText) {
		score += 4
	}

	contentNode := node.Find("div.comments-content").First()
	if contentNode.Length() == 0 {
		return historicalCommentDetail{}, -1
	}

	authorNode := contentNode.Find("a.nickname").First()
	authorName := cleanHistoricalText(authorNode.Text())
	authorQQ := extractQQFromSelection(authorNode)
	if item.AuthorQQ != "" && authorQQ == item.AuthorQQ {
		score += 4
	}
	if item.AuthorName != "" && authorName == item.AuthorName {
		score += 2
	}

	rawText := cleanHistoricalText(contentNode.Clone().Find("div.comments-op").Remove().End().Text())
	content, replyToName := trimHistoricalCommentText(rawText, authorName)
	if content == "" {
		return historicalCommentDetail{}, -1
	}

	replyToQQ := ""
	if replyToName != "" {
		contentNode.Find("a.nickname").Each(func(i int, s *goquery.Selection) {
			if i == 0 {
				return
			}
			if cleanHistoricalText(s.Text()) == replyToName && replyToQQ == "" {
				replyToQQ = extractQQFromSelection(s)
			}
		})
	}

	if score == 0 && authorQQ == "" && authorName == "" {
		return historicalCommentDetail{}, -1
	}

	return historicalCommentDetail{
		AuthorQQ:    authorQQ,
		AuthorName:  authorName,
		Content:     content,
		ReplyToQQ:   replyToQQ,
		ReplyToName: replyToName,
	}, score
}

func cleanHistoricalText(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ")
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func historicalCommentTimeMatches(publishTime time.Time, raw string) bool {
	if publishTime.IsZero() || strings.TrimSpace(raw) == "" {
		return false
	}
	parsed := parseHistoricalCommentTime(raw, publishTime.Location())
	if parsed.IsZero() {
		return false
	}
	return parsed.Format("2006-01-02 15:04") == publishTime.In(parsed.Location()).Format("2006-01-02 15:04")
}

func parseHistoricalCommentTime(raw string, loc *time.Location) time.Time {
	text := cleanHistoricalText(raw)
	if text == "" {
		return time.Time{}
	}
	if loc == nil {
		loc = time.Local
	}

	layouts := []string{
		"2006年1月2日 15:04",
		"2006年01月02日 15:04",
		"1月2日 15:04",
		"01月02日 15:04",
	}
	now := time.Now().In(loc)
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, text, loc)
		if err != nil {
			continue
		}
		switch layout {
		case "1月2日 15:04", "01月02日 15:04":
			return time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, loc)
		default:
			return t
		}
	}
	return time.Time{}
}

func trimHistoricalCommentText(raw, authorName string) (string, string) {
	text := cleanHistoricalText(raw)
	if authorName != "" {
		text = strings.TrimSpace(strings.TrimPrefix(text, authorName))
	}
	text = strings.TrimSpace(strings.TrimLeft(text, ":："))
	if text == "" {
		return "", ""
	}

	replyPrefixes := []string{"回复", "也回复"}
	for _, prefix := range replyPrefixes {
		if !strings.HasPrefix(text, prefix) {
			continue
		}
		remainder := strings.TrimSpace(strings.TrimPrefix(text, prefix))
		for _, sep := range []string{":", "："} {
			parts := strings.SplitN(remainder, sep, 2)
			if len(parts) == 2 {
				replyToName := cleanHistoricalText(parts[0])
				content := cleanHistoricalText(parts[1])
				if content != "" {
					return content, replyToName
				}
			}
		}
	}

	for _, sep := range []string{":", "："} {
		parts := strings.SplitN(text, sep, 2)
		if len(parts) == 2 {
			content := cleanHistoricalText(parts[1])
			if content != "" {
				return content, ""
			}
		}
	}
	return text, ""
}

func extractQQFromSelection(node *goquery.Selection) string {
	if node == nil || node.Length() == 0 {
		return ""
	}
	if link, ok := node.Attr("link"); ok && strings.HasPrefix(link, "nameCard_") {
		return strings.TrimPrefix(link, "nameCard_")
	}
	href, ok := node.Attr("href")
	if !ok {
		return ""
	}
	href = strings.TrimSpace(href)
	idx := strings.LastIndex(href, "/")
	if idx == -1 || idx == len(href)-1 {
		return ""
	}
	return href[idx+1:]
}

func qzoneSleepWithJitter(base, jitter time.Duration) {
	if base < 0 {
		base = 0
	}
	if jitter <= 0 {
		time.Sleep(base)
		return
	}
	time.Sleep(base + time.Duration(rand.Int63n(int64(jitter))))
}

func mustJSON(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func firstImage(images []string) string {
	if len(images) == 0 {
		return ""
	}
	return images[0]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func truncateString(s string, n int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= n {
		return string(runes)
	}
	return string(runes[:n])
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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
