package service

import (
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/model"
	"github.com/qzone-memory/pkg/response"
)

type MemoryItem struct {
	Type        string    `json:"type"`
	Subtype     string    `json:"subtype"`
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Cover       string    `json:"cover"`
	Images      string    `json:"images"`
	AuthorQQ    string    `json:"author_qq"`
	AuthorName  string    `json:"author_name"`
	IsDeleted   bool      `json:"is_deleted"`
	PublishTime time.Time `json:"publish_time"`
	Source      string    `json:"source"`
}

func GetMemoryTimeline(c *gin.Context) (*dto.PageResponse[*MemoryItem], *response.AppError) {
	var req dto.QueryMemoryRequest
	if err := bindQuery(c, &req); err != nil {
		return nil, err
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	all, err := buildMemoryTimeline(req.QQ, req.Type)
	if err != nil {
		return nil, &response.AppError{Code: 500, Err: err}
	}

	total := int64(len(all))
	start := (page - 1) * pageSize
	if start >= len(all) {
		return dto.NewPageResponse([]*MemoryItem{}, total, page, pageSize), nil
	}
	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}
	return dto.NewPageResponse(all[start:end], total, page, pageSize), nil
}

func buildMemoryTimeline(userQQ, filterType string) ([]*MemoryItem, error) {
	var activities []*model.Activity
	var talks []*model.Talk
	var blogs []*model.Blog
	var albums []*model.Album
	var messages []*model.Message
	var visitors []*model.Visitor
	var videos []*model.Video
	var favorites []*model.Favorite
	var diaries []*model.Diary
	var likes []*model.Like
	var shares []*model.Share
	var mentions []*model.Mention
	var comments []*model.Comment

	if err := database.DB.Where("user_qq = ?", userQQ).Find(&activities).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&talks).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&blogs).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&albums).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&messages).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&visitors).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&videos).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&favorites).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&diaries).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&likes).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&shares).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&mentions).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Where("user_qq = ?", userQQ).Find(&comments).Error; err != nil {
		return nil, err
	}

	talkMap := make(map[string]*model.Talk, len(talks))
	for _, item := range talks {
		if item == nil {
			continue
		}
		talkMap[item.TalkID] = item
	}

	messageMap := make(map[string]*model.Message, len(messages))
	for _, item := range messages {
		if item == nil {
			continue
		}
		messageMap[item.MessageID] = item
	}

	commentTargetText := make(map[string]string, len(talks)+len(blogs)+len(messages))
	for _, item := range talks {
		if item == nil {
			continue
		}
		commentTargetText["talk:"+item.TalkID] = firstNonEmpty(item.Content, "评论了你的历史说说")
	}
	for _, item := range blogs {
		if item == nil {
			continue
		}
		commentTargetText["blog:"+item.BlogID] = firstNonEmpty(item.Title, item.Summary, item.Content, "评论了你的历史日志")
	}
	for _, item := range messages {
		if item == nil {
			continue
		}
		commentTargetText["message:"+item.MessageID] = firstNonEmpty(item.Content, "评论了你的历史留言")
	}

	items := make([]*MemoryItem, 0, len(activities)+len(talks)+len(blogs)+len(albums)+len(messages)+len(visitors)+len(videos)+len(favorites)+len(diaries)+len(likes)+len(shares)+len(mentions)+len(comments))
	for _, item := range activities {
		items = append(items, &MemoryItem{
			Type:        "activity",
			Subtype:     item.FeedType,
			ID:          item.FeedID,
			Title:       item.Title,
			Content:     item.Content,
			Images:      item.Images,
			AuthorQQ:    item.AuthorQQ,
			AuthorName:  item.AuthorName,
			IsDeleted:   item.IsDeleted,
			PublishTime: item.PublishTime,
			Source:      "activity",
		})
	}
	for _, item := range talks {
		items = append(items, &MemoryItem{
			Type:        "talk",
			Subtype:     "talk",
			ID:          item.TalkID,
			Content:     item.Content,
			Images:      item.Images,
			IsDeleted:   item.IsDeleted,
			PublishTime: item.PublishTime,
			Source:      "talk",
		})
	}
	for _, item := range blogs {
		items = append(items, &MemoryItem{
			Type:        "blog",
			Subtype:     "blog",
			ID:          item.BlogID,
			Title:       item.Title,
			Content:     item.Content,
			IsDeleted:   item.IsDeleted,
			PublishTime: item.PublishTime,
			Source:      "blog",
		})
	}
	for _, item := range albums {
		items = append(items, &MemoryItem{
			Type:        "album",
			Subtype:     "photo",
			ID:          item.AlbumID,
			Title:       item.Name,
			Content:     item.Description,
			Cover:       item.CoverURL,
			IsDeleted:   item.IsDeleted,
			PublishTime: item.CreateTime,
			Source:      "album",
		})
	}
	for _, item := range messages {
		items = append(items, &MemoryItem{
			Type:        "message",
			Subtype:     "message",
			ID:          item.MessageID,
			Content:     item.Content,
			AuthorQQ:    item.AuthorQQ,
			AuthorName:  item.AuthorName,
			PublishTime: item.MessageTime,
			Source:      "message",
		})
	}
	for _, item := range visitors {
		items = append(items, &MemoryItem{
			Type:        "visitor",
			Subtype:     "visitor",
			ID:          item.VisitorID,
			Title:       firstNonEmpty(item.VisitorName, item.VisitorQQ),
			Content:     "访问了你的空间",
			AuthorQQ:    item.VisitorQQ,
			AuthorName:  item.VisitorName,
			Cover:       item.Avatar,
			PublishTime: item.VisitTime,
			Source:      "visitor",
		})
	}
	for _, item := range videos {
		items = append(items, &MemoryItem{
			Type:        "video",
			Subtype:     "video",
			ID:          item.VideoID,
			Title:       item.Title,
			Content:     item.Description,
			Cover:       item.PreviewURL,
			PublishTime: item.UploadTime,
			Source:      "video",
		})
	}
	for _, item := range favorites {
		items = append(items, &MemoryItem{
			Type:        "favorite",
			Subtype:     "favorite",
			ID:          item.FavoriteID,
			Title:       item.Title,
			Content:     item.Abstract,
			Images:      item.Images,
			AuthorQQ:    item.OwnerQQ,
			AuthorName:  item.OwnerName,
			PublishTime: item.CreateTime,
			Source:      "favorite",
		})
	}
	for _, item := range diaries {
		items = append(items, &MemoryItem{
			Type:        "diary",
			Subtype:     "diary",
			ID:          item.DiaryID,
			Title:       item.Title,
			Content:     firstNonEmpty(item.Content, item.Summary),
			PublishTime: item.CreateTime,
			Source:      "diary",
		})
	}
	for _, item := range likes {
		content := "赞过你的历史内容"
		images := ""
		isDeleted := false
		if target, ok := talkMap[item.TargetID]; ok && target != nil {
			content = firstNonEmpty(target.Content, content)
			images = target.Images
			isDeleted = target.IsDeleted
		}
		items = append(items, &MemoryItem{
			Type:        "like",
			Subtype:     item.TargetType,
			ID:          item.LikeID,
			Title:       firstNonEmpty(item.LikerName, item.LikerQQ),
			Content:     content,
			AuthorQQ:    item.LikerQQ,
			AuthorName:  item.LikerName,
			Cover:       item.LikerAvatar,
			Images:      images,
			IsDeleted:   isDeleted,
			PublishTime: item.LikeTime,
			Source:      "like",
		})
	}
	for _, item := range shares {
		items = append(items, &MemoryItem{
			Type:        "share",
			Subtype:     item.TargetType,
			ID:          item.ShareID,
			Title:       firstNonEmpty(item.SharerName, item.SharerQQ),
			Content:     firstNonEmpty(item.Comment, "转发了你的历史内容"),
			AuthorQQ:    item.SharerQQ,
			AuthorName:  item.SharerName,
			PublishTime: item.ShareTime,
			Source:      "share",
		})
	}
	for _, item := range comments {
		content := firstNonEmpty(item.Content, "评论了你的历史内容")
		if item.ReplyToName != "" {
			content = "回复 " + item.ReplyToName + "：" + content
		}
		if target := commentTargetText[item.TargetType+":"+item.TargetID]; target != "" && target != item.Content {
			content = content + " | 原动态：" + target
		}
		items = append(items, &MemoryItem{
			Type:        "comment",
			Subtype:     item.TargetType,
			ID:          item.CommentID,
			Title:       firstNonEmpty(item.AuthorName, item.AuthorQQ),
			Content:     content,
			AuthorQQ:    item.AuthorQQ,
			AuthorName:  item.AuthorName,
			Cover:       item.AuthorAvatar,
			IsDeleted:   item.IsDeleted,
			PublishTime: item.CommentTime,
			Source:      "comment",
		})
	}
	for _, item := range mentions {
		content := firstNonEmpty(item.Content, "提到了你")
		if target, ok := messageMap[item.SourceID]; ok && target != nil {
			content = firstNonEmpty(target.Content, content)
		}
		items = append(items, &MemoryItem{
			Type:        "mention",
			Subtype:     item.SourceType,
			ID:          item.MentionID,
			Title:       firstNonEmpty(item.AuthorName, item.AuthorQQ),
			Content:     content,
			AuthorQQ:    item.AuthorQQ,
			AuthorName:  item.AuthorName,
			PublishTime: item.MentionTime,
			Source:      "mention",
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].PublishTime.After(items[j].PublishTime)
	})
	result := dedupeMemoryItems(items)
	if filterType == "" || filterType == "all" {
		return result, nil
	}
	filtered := make([]*MemoryItem, 0, len(result))
	for _, item := range result {
		if item.Type == filterType || item.Subtype == filterType {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func dedupeMemoryItems(items []*MemoryItem) []*MemoryItem {
	seen := make(map[string]struct{}, len(items))
	result := make([]*MemoryItem, 0, len(items))
	for _, item := range items {
		key := item.Source + ":" + item.ID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	return result
}
