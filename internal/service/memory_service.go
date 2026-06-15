package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/model"
)

type MemoryItem struct {
	Type           string                 `json:"type"`
	Subtype        string                 `json:"subtype"`
	ID             string                 `json:"id"`
	Title          string                 `json:"title"`
	Content        string                 `json:"content"`
	Cover          string                 `json:"cover"`
	Images         string                 `json:"images"`
	OwnerQQ        string                 `json:"owner_qq"`
	OwnerName      string                 `json:"owner_name"`
	ActorQQ        string                 `json:"actor_qq"`
	ActorName      string                 `json:"actor_name"`
	ActorAvatar    string                 `json:"actor_avatar"`
	TargetType     string                 `json:"target_type"`
	TargetID       string                 `json:"target_id"`
	TargetTitle    string                 `json:"target_title"`
	TargetText     string                 `json:"target_text"`
	Relation       string                 `json:"relation"`
	CanExpand      bool                   `json:"can_expand"`
	AuthorQQ       string                 `json:"author_qq"`
	AuthorName     string                 `json:"author_name"`
	IsDeleted      bool                   `json:"is_deleted"`
	PublishTime    time.Time              `json:"publish_time"`
	Source         string                 `json:"source"`
	LikeCount      int                    `json:"like_count"`
	CommentCount   int                    `json:"comment_count"`
	ShareCount     int                    `json:"share_count"`
	LikePreview    []MemoryLikePreview    `json:"like_preview,omitempty"`
	CommentPreview []MemoryCommentPreview `json:"comment_preview,omitempty"`
	SharePreview   []MemorySharePreview   `json:"share_preview,omitempty"`
}

type MemoryLikePreview struct {
	QQ     string    `json:"qq"`
	Name   string    `json:"name"`
	Avatar string    `json:"avatar"`
	Time   time.Time `json:"time"`
}

type MemoryCommentPreview struct {
	QQ          string    `json:"qq"`
	Name        string    `json:"name"`
	Avatar      string    `json:"avatar"`
	Content     string    `json:"content"`
	ReplyToQQ   string    `json:"reply_to_qq"`
	ReplyToName string    `json:"reply_to_name"`
	Time        time.Time `json:"time"`
	IsDeleted   bool      `json:"is_deleted"`
}

type MemorySharePreview struct {
	QQ      string    `json:"qq"`
	Name    string    `json:"name"`
	Avatar  string    `json:"avatar"`
	Comment string    `json:"comment"`
	Time    time.Time `json:"time"`
}

const (
	relationOwnContent         = "own_content"
	relationBoardMessage       = "board_message"
	relationInboundInteraction = "inbound_interaction"
)

func GetMemoryTimeline(ctx context.Context, req dto.QueryMemoryRequest) (*dto.PageResponse[*MemoryItem], error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	filter := req.Type
	if filter == "" || filter == "all" {
		filter = "content" // 默认时间线只展示你自己的内容，互动单独看
	}
	all, err := buildMemoryTimeline(ctx, req.QQ, filter)
	if err != nil {
		return nil, err
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

// GetOnThisDay 返回历史上「今天」（同月同日）各年留下的回忆。month/day 省略时用服务器当天。
func GetOnThisDay(ctx context.Context, req dto.QueryOnThisDayRequest) (*dto.PageResponse[*MemoryItem], error) {
	month := time.Month(req.Month)
	day := req.Day
	if req.Month <= 0 || req.Day <= 0 {
		now := time.Now()
		month, day = now.Month(), now.Day()
	}

	all, err := buildMemoryTimeline(ctx, req.QQ, "all")
	if err != nil {
		return nil, err
	}

	matched := make([]*MemoryItem, 0)
	for _, item := range all {
		if item.PublishTime.IsZero() {
			continue
		}
		if item.PublishTime.Month() == month && item.PublishTime.Day() == day {
			matched = append(matched, item)
		}
	}
	// buildMemoryTimeline 已按时间倒序，这里保持
	pageSize := len(matched)
	if pageSize == 0 {
		pageSize = 1
	}
	return dto.NewPageResponse(matched, int64(len(matched)), 1, pageSize), nil
}

// SearchMemory 在合并时间线里按关键词全文检索（标题 / 正文 / 作者），分页返回。
func SearchMemory(ctx context.Context, req dto.QueryMemorySearchRequest) (*dto.PageResponse[*MemoryItem], error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))

	all, err := buildMemoryTimeline(ctx, req.QQ, "all")
	if err != nil {
		return nil, err
	}

	matched := make([]*MemoryItem, 0)
	if keyword != "" {
		for _, item := range all {
			if memoryItemMatches(item, keyword) {
				matched = append(matched, item)
			}
		}
	}

	total := int64(len(matched))
	start := (page - 1) * pageSize
	if start >= len(matched) {
		return dto.NewPageResponse([]*MemoryItem{}, total, page, pageSize), nil
	}
	end := start + pageSize
	if end > len(matched) {
		end = len(matched)
	}
	return dto.NewPageResponse(matched[start:end], total, page, pageSize), nil
}

func memoryItemMatches(item *MemoryItem, lowerKeyword string) bool {
	return strings.Contains(strings.ToLower(item.Content), lowerKeyword) ||
		strings.Contains(strings.ToLower(item.Title), lowerKeyword) ||
		strings.Contains(strings.ToLower(item.AuthorName), lowerKeyword) ||
		strings.Contains(strings.ToLower(item.ActorName), lowerKeyword) ||
		strings.Contains(strings.ToLower(item.ActorQQ), lowerKeyword) ||
		strings.Contains(strings.ToLower(item.TargetTitle), lowerKeyword) ||
		strings.Contains(strings.ToLower(item.TargetText), lowerKeyword)
}

type FriendInteractionResult struct {
	FriendCandidates []FriendCandidate              `json:"friend_candidates"`
	Keyword          string                         `json:"keyword"`
	Stats            map[string]int64               `json:"stats"`
	Items            *dto.PageResponse[*MemoryItem] `json:"items"`
}

type FriendCandidate struct {
	QQ            string    `json:"qq"`
	Name          string    `json:"name"`
	Remark        string    `json:"remark"`
	GroupName     string    `json:"group_name"`
	IsCurrent     bool      `json:"is_current"`
	IsDeleted     bool      `json:"is_deleted"`
	SourceType    string    `json:"source_type"`
	InteractCount int       `json:"interact_count"`
	LastSeenAt    time.Time `json:"last_seen_at"`
}

type MemoryInteractionDetail struct {
	Target   *MemoryItem            `json:"target,omitempty"`
	Likes    []MemoryLikePreview    `json:"likes"`
	Comments []MemoryCommentPreview `json:"comments"`
	Shares   []MemorySharePreview   `json:"shares"`
	Stats    map[string]int         `json:"stats"`
}

func GetMemoryInteractions(ctx context.Context, req dto.QueryMemoryInteractionsRequest) (*MemoryInteractionDetail, error) {
	targetType := strings.TrimSpace(req.TargetType)
	targetID := strings.TrimSpace(req.TargetID)
	if targetType == "" || targetID == "" {
		return &MemoryInteractionDetail{Stats: map[string]int{}}, nil
	}

	aliases, err := loadTargetAliases(ctx, req.QQ)
	if err != nil {
		return nil, err
	}
	targetIDs := collectResolvedTargetIDs(targetType, targetID, aliases)

	detail := &MemoryInteractionDetail{
		Likes:    make([]MemoryLikePreview, 0),
		Comments: make([]MemoryCommentPreview, 0),
		Shares:   make([]MemorySharePreview, 0),
		Stats:    make(map[string]int),
	}

	var likes []*model.Like
	if err := database.DB.WithContext(ctx).
		Where("user_qq = ? AND target_type = ? AND target_id IN ?", req.QQ, targetType, targetIDs).
		Order("like_time DESC").
		Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, item := range likes {
		detail.Likes = append(detail.Likes, MemoryLikePreview{
			QQ:     item.LikerQQ,
			Name:   firstNonEmpty(item.LikerName, item.LikerQQ),
			Avatar: item.LikerAvatar,
			Time:   item.LikeTime,
		})
	}

	var comments []*model.Comment
	if err := database.DB.WithContext(ctx).
		Where("user_qq = ? AND target_type = ? AND target_id IN ?", req.QQ, targetType, targetIDs).
		Order("comment_time ASC, id ASC").
		Find(&comments).Error; err != nil {
		return nil, err
	}
	for _, item := range comments {
		detail.Comments = append(detail.Comments, MemoryCommentPreview{
			QQ:          item.AuthorQQ,
			Name:        firstNonEmpty(item.AuthorName, item.AuthorQQ),
			Avatar:      item.AuthorAvatar,
			Content:     item.Content,
			ReplyToQQ:   item.ReplyToQQ,
			ReplyToName: item.ReplyToName,
			Time:        item.CommentTime,
			IsDeleted:   item.IsDeleted,
		})
	}

	var shares []*model.Share
	if err := database.DB.WithContext(ctx).
		Where("user_qq = ? AND target_type = ? AND target_id IN ?", req.QQ, targetType, targetIDs).
		Order("share_time DESC, id DESC").
		Find(&shares).Error; err != nil {
		return nil, err
	}
	for _, item := range shares {
		detail.Shares = append(detail.Shares, MemorySharePreview{
			QQ:      item.SharerQQ,
			Name:    firstNonEmpty(item.SharerName, item.SharerQQ),
			Avatar:  item.SharerAvatar,
			Comment: item.Comment,
			Time:    item.ShareTime,
		})
	}

	detail.Stats["like"] = len(detail.Likes)
	detail.Stats["comment"] = len(detail.Comments)
	detail.Stats["share"] = len(detail.Shares)

	if target, err := findMemoryTarget(ctx, req.QQ, targetType, targetID); err == nil {
		detail.Target = target
	}
	return detail, nil
}

// SearchFriendInteractions 按好友昵称、备注或 QQ 聚合所有相关互动。
func SearchFriendInteractions(ctx context.Context, req dto.QueryFriendInteractionsRequest) (*FriendInteractionResult, error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	keyword := strings.TrimSpace(req.Keyword)
	if keyword == "" {
		return &FriendInteractionResult{
			Keyword: keyword,
			Stats:   map[string]int64{},
			Items:   dto.NewPageResponse([]*MemoryItem{}, 0, page, pageSize),
		}, nil
	}

	candidates, tokens, err := findFriendInteractionTokens(ctx, req.QQ, keyword)
	if err != nil {
		return nil, err
	}
	if req.CandidatesOnly {
		return &FriendInteractionResult{
			FriendCandidates: candidates,
			Keyword:          keyword,
			Stats:            map[string]int64{},
			Items:            dto.NewPageResponse([]*MemoryItem{}, 0, page, pageSize),
		}, nil
	}
	if len(tokens) == 0 {
		tokens = []string{strings.ToLower(keyword)}
	}

	all, err := buildMemoryTimeline(ctx, req.QQ, "all")
	if err != nil {
		return nil, err
	}

	matched := make([]*MemoryItem, 0)
	stats := make(map[string]int64)
	for _, item := range all {
		if friendMemoryItemMatches(item, tokens) {
			matched = append(matched, item)
			stats[item.Type]++
		}
	}

	total := int64(len(matched))
	start := (page - 1) * pageSize
	if start >= len(matched) {
		return &FriendInteractionResult{FriendCandidates: candidates, Keyword: keyword, Stats: stats, Items: dto.NewPageResponse([]*MemoryItem{}, total, page, pageSize)}, nil
	}
	end := start + pageSize
	if end > len(matched) {
		end = len(matched)
	}

	return &FriendInteractionResult{
		FriendCandidates: candidates,
		Keyword:          keyword,
		Stats:            stats,
		Items:            dto.NewPageResponse(matched[start:end], total, page, pageSize),
	}, nil
}

func findFriendInteractionTokens(ctx context.Context, userQQ, keyword string) ([]FriendCandidate, []string, error) {
	lowerKeyword := strings.ToLower(keyword)
	like := "%" + keyword + "%"
	var friends []*model.Friend
	if err := database.DB.WithContext(ctx).
		Where("user_qq = ? AND (friend_qq = ? OR name LIKE ? OR remark LIKE ? OR group_name LIKE ?)", userQQ, keyword, like, like, like).
		Order("interact_count DESC, is_current DESC").
		Limit(12).
		Find(&friends).Error; err != nil {
		return nil, nil, err
	}

	seen := make(map[string]struct{})
	addToken := func(v string) {
		v = strings.TrimSpace(strings.ToLower(v))
		if v == "" {
			return
		}
		seen[v] = struct{}{}
	}
	addToken(lowerKeyword)

	candidates := make([]FriendCandidate, 0, len(friends))
	for _, f := range friends {
		if f == nil {
			continue
		}
		candidates = append(candidates, FriendCandidate{
			QQ:            f.FriendQQ,
			Name:          f.Name,
			Remark:        f.Remark,
			GroupName:     f.GroupName,
			IsCurrent:     f.IsCurrent,
			IsDeleted:     f.IsDeleted,
			SourceType:    f.SourceType,
			InteractCount: f.InteractCount,
			LastSeenAt:    f.LastSeenAt,
		})
		addToken(f.FriendQQ)
		addToken(f.Name)
		addToken(f.Remark)
	}

	tokens := make([]string, 0, len(seen))
	for token := range seen {
		tokens = append(tokens, token)
	}
	return candidates, tokens, nil
}

func friendMemoryItemMatches(item *MemoryItem, tokens []string) bool {
	haystacks := []string{
		item.AuthorQQ,
		item.AuthorName,
		item.ActorQQ,
		item.ActorName,
		item.TargetTitle,
		item.TargetText,
		item.Title,
		item.Content,
	}
	for _, haystack := range haystacks {
		lower := strings.ToLower(strings.TrimSpace(haystack))
		if lower == "" {
			continue
		}
		for _, token := range tokens {
			if token == "" {
				continue
			}
			if lower == token || strings.Contains(lower, token) {
				return true
			}
		}
	}
	return false
}

func buildMemoryTimeline(ctx context.Context, userQQ, filterType string) ([]*MemoryItem, error) {
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

	if err := database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Find(&activities).Error; err != nil {
		return nil, err
	}
	if err := database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Find(&talks).Error; err != nil {
		return nil, err
	}
	if err := database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Find(&blogs).Error; err != nil {
		return nil, err
	}
	if err := database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Find(&albums).Error; err != nil {
		return nil, err
	}
	if err := database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Find(&messages).Error; err != nil {
		return nil, err
	}
	if err := database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Find(&visitors).Error; err != nil {
		return nil, err
	}
	if err := database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Find(&videos).Error; err != nil {
		return nil, err
	}
	if err := database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Find(&favorites).Error; err != nil {
		return nil, err
	}
	if err := database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Find(&diaries).Error; err != nil {
		return nil, err
	}
	if err := database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Find(&likes).Error; err != nil {
		return nil, err
	}
	if err := database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Find(&shares).Error; err != nil {
		return nil, err
	}
	if err := database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Find(&mentions).Error; err != nil {
		return nil, err
	}
	if err := database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Find(&comments).Error; err != nil {
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

	targetAliases := buildTargetAliases(userQQ, activities, talkMap)
	targetPreviews := buildTargetPreviews(likes, comments, shares, targetAliases)

	commentTargetText := make(map[string]string, len(talks)+len(blogs)+len(messages))
	for _, item := range talks {
		if item == nil {
			continue
		}
		commentTargetText[targetKey("talk", item.TalkID)] = firstNonEmpty(item.Content, "评论了你的历史说说")
	}
	for _, item := range blogs {
		if item == nil {
			continue
		}
		commentTargetText[targetKey("blog", item.BlogID)] = firstNonEmpty(item.Title, item.Summary, item.Content, "评论了你的历史日志")
	}
	for _, item := range messages {
		if item == nil {
			continue
		}
		commentTargetText[targetKey("message", item.MessageID)] = firstNonEmpty(item.Content, "评论了你的历史留言")
	}

	items := make([]*MemoryItem, 0, len(activities)+len(talks)+len(blogs)+len(albums)+len(messages)+len(visitors)+len(videos)+len(favorites)+len(diaries)+len(likes)+len(shares)+len(mentions)+len(comments))
	// 原始动态(activity)不再进入时间线：它们多是"别人赞我/评论我"的好友动态，
	// 其有用内容（说说/赞/评论/留言）已被重建逻辑抽进各自的表，原始条目只是重复噪音。
	_ = activities
	for _, item := range talks {
		if item.IsSpam {
			continue // 游戏自动说说/广告等默认不进回忆（数据仍保留在库中）
		}
		preview := targetPreviews[targetKey("talk", item.TalkID)]
		items = append(items, &MemoryItem{
			Type:           "talk",
			Subtype:        "talk",
			ID:             item.TalkID,
			Content:        item.Content,
			Images:         item.Images,
			OwnerQQ:        userQQ,
			OwnerName:      "我",
			AuthorQQ:       userQQ,
			AuthorName:     "我",
			TargetType:     "talk",
			TargetID:       item.TalkID,
			TargetText:     item.Content,
			Relation:       relationOwnContent,
			CanExpand:      preview.HasAny(),
			IsDeleted:      item.IsDeleted,
			PublishTime:    item.PublishTime,
			Source:         "talk",
			LikeCount:      maxInt(item.LikeCount, preview.LikeCount),
			CommentCount:   maxInt(item.CommentCount, preview.CommentCount),
			ShareCount:     maxInt(item.ShareCount, preview.ShareCount),
			LikePreview:    preview.Likes,
			CommentPreview: preview.Comments,
			SharePreview:   preview.Shares,
		})
	}
	for _, item := range blogs {
		preview := targetPreviews[targetKey("blog", item.BlogID)]
		items = append(items, &MemoryItem{
			Type:           "blog",
			Subtype:        "blog",
			ID:             item.BlogID,
			Title:          item.Title,
			Content:        item.Content,
			OwnerQQ:        userQQ,
			OwnerName:      "我",
			AuthorQQ:       userQQ,
			AuthorName:     "我",
			TargetType:     "blog",
			TargetID:       item.BlogID,
			TargetTitle:    item.Title,
			TargetText:     firstNonEmpty(item.Summary, item.Content),
			Relation:       relationOwnContent,
			CanExpand:      preview.HasAny(),
			IsDeleted:      item.IsDeleted,
			PublishTime:    item.PublishTime,
			Source:         "blog",
			LikeCount:      maxInt(item.LikeCount, preview.LikeCount),
			CommentCount:   maxInt(item.CommentCount, preview.CommentCount),
			ShareCount:     preview.ShareCount,
			LikePreview:    preview.Likes,
			CommentPreview: preview.Comments,
			SharePreview:   preview.Shares,
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
			OwnerQQ:     userQQ,
			OwnerName:   "我",
			AuthorQQ:    userQQ,
			AuthorName:  "我",
			TargetType:  "album",
			TargetID:    item.AlbumID,
			TargetTitle: item.Name,
			TargetText:  item.Description,
			Relation:    relationOwnContent,
			IsDeleted:   item.IsDeleted,
			PublishTime: item.CreateTime,
			Source:      "album",
		})
	}
	for _, item := range messages {
		preview := targetPreviews[targetKey("message", item.MessageID)]
		items = append(items, &MemoryItem{
			Type:           "message",
			Subtype:        "message",
			ID:             item.MessageID,
			Content:        item.Content,
			OwnerQQ:        userQQ,
			OwnerName:      "我",
			ActorQQ:        item.AuthorQQ,
			ActorName:      item.AuthorName,
			ActorAvatar:    item.AuthorAvatar,
			TargetType:     "message",
			TargetID:       item.MessageID,
			TargetText:     item.Content,
			Relation:       relationBoardMessage,
			CanExpand:      preview.HasAny() || strings.TrimSpace(item.ReplyContent) != "",
			AuthorQQ:       item.AuthorQQ,
			AuthorName:     item.AuthorName,
			PublishTime:    item.MessageTime,
			Source:         "message",
			LikeCount:      preview.LikeCount,
			CommentCount:   preview.CommentCount,
			ShareCount:     preview.ShareCount,
			LikePreview:    preview.Likes,
			CommentPreview: preview.Comments,
			SharePreview:   preview.Shares,
		})
	}
	for _, item := range visitors {
		items = append(items, &MemoryItem{
			Type:        "visitor",
			Subtype:     "visitor",
			ID:          item.VisitorID,
			Title:       firstNonEmpty(item.VisitorName, item.VisitorQQ),
			Content:     "访问了你的空间",
			OwnerQQ:     userQQ,
			OwnerName:   "我",
			ActorQQ:     item.VisitorQQ,
			ActorName:   item.VisitorName,
			ActorAvatar: item.Avatar,
			Relation:    relationInboundInteraction,
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
			OwnerQQ:     userQQ,
			OwnerName:   "我",
			AuthorQQ:    userQQ,
			AuthorName:  "我",
			TargetType:  "video",
			TargetID:    item.VideoID,
			TargetTitle: item.Title,
			TargetText:  item.Description,
			Relation:    relationOwnContent,
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
			OwnerQQ:     userQQ,
			OwnerName:   "我",
			ActorQQ:     item.OwnerQQ,
			ActorName:   item.OwnerName,
			TargetType:  "favorite",
			TargetID:    item.FavoriteID,
			TargetTitle: item.Title,
			TargetText:  item.Abstract,
			Relation:    relationOwnContent,
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
			OwnerQQ:     userQQ,
			OwnerName:   "我",
			AuthorQQ:    userQQ,
			AuthorName:  "我",
			TargetType:  "diary",
			TargetID:    item.DiaryID,
			TargetTitle: item.Title,
			TargetText:  firstNonEmpty(item.Summary, item.Content),
			Relation:    relationOwnContent,
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
			OwnerQQ:     userQQ,
			OwnerName:   "我",
			ActorQQ:     item.LikerQQ,
			ActorName:   item.LikerName,
			ActorAvatar: item.LikerAvatar,
			TargetType:  item.TargetType,
			TargetID:    item.TargetID,
			TargetText:  content,
			Relation:    relationInboundInteraction,
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
			OwnerQQ:     userQQ,
			OwnerName:   "我",
			ActorQQ:     item.SharerQQ,
			ActorName:   item.SharerName,
			ActorAvatar: item.SharerAvatar,
			TargetType:  item.TargetType,
			TargetID:    item.TargetID,
			Relation:    relationInboundInteraction,
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
		targetID := resolveTargetID(item.TargetType, item.TargetID, targetAliases)
		if target := commentTargetText[targetKey(item.TargetType, targetID)]; target != "" && target != item.Content {
			content = content + " | 原动态：" + target
		}
		items = append(items, &MemoryItem{
			Type:        "comment",
			Subtype:     item.TargetType,
			ID:          item.CommentID,
			Title:       firstNonEmpty(item.AuthorName, item.AuthorQQ),
			Content:     content,
			OwnerQQ:     userQQ,
			OwnerName:   "我",
			ActorQQ:     item.AuthorQQ,
			ActorName:   item.AuthorName,
			ActorAvatar: item.AuthorAvatar,
			TargetType:  item.TargetType,
			TargetID:    item.TargetID,
			TargetText:  commentTargetText[targetKey(item.TargetType, targetID)],
			Relation:    relationInboundInteraction,
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
			OwnerQQ:     userQQ,
			OwnerName:   "我",
			ActorQQ:     item.AuthorQQ,
			ActorName:   item.AuthorName,
			TargetType:  item.SourceType,
			TargetID:    item.SourceID,
			TargetText:  content,
			Relation:    relationInboundInteraction,
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
	switch filterType {
	case "all":
		return result, nil
	case "", "content":
		return filterByMemoryTypes(result, contentMemoryTypes), nil
	case "interact", "interaction":
		return filterByMemoryTypes(result, interactionMemoryTypes), nil
	default:
		filtered := make([]*MemoryItem, 0, len(result))
		for _, item := range result {
			if item.Type == filterType || item.Subtype == filterType {
				filtered = append(filtered, item)
			}
		}
		return filtered, nil
	}
}

type memoryTargetPreview struct {
	LikeCount    int
	CommentCount int
	ShareCount   int
	Likes        []MemoryLikePreview
	Comments     []MemoryCommentPreview
	Shares       []MemorySharePreview
}

func (p memoryTargetPreview) HasAny() bool {
	return p.LikeCount > 0 || p.CommentCount > 0 || p.ShareCount > 0
}

func targetKey(targetType, targetID string) string {
	return strings.TrimSpace(targetType) + ":" + strings.TrimSpace(targetID)
}

func addTargetAlias(aliases map[string]string, targetType, from, to string) {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if targetType == "" || from == "" || to == "" {
		return
	}
	aliases[targetKey(targetType, from)] = to
}

func buildTargetAliases(userQQ string, activities []*model.Activity, talkMap map[string]*model.Talk) map[string]string {
	aliases := make(map[string]string)
	for talkID := range talkMap {
		addTargetAlias(aliases, "talk", talkID, talkID)
	}

	for _, item := range activities {
		if item == nil || item.FeedType != "talk" {
			continue
		}
		content := activityUserContent(userQQ, *item)
		if content == "" {
			continue
		}
		canonicalID := contentTalkID(content)
		if _, ok := talkMap[canonicalID]; !ok {
			continue
		}
		addTargetAlias(aliases, "talk", canonicalID, canonicalID)
		addTargetAlias(aliases, "talk", item.ObjectID, canonicalID)
		addTargetAlias(aliases, "talk", "talk_"+item.ObjectID, canonicalID)
		addTargetAlias(aliases, "talk", item.FeedID, canonicalID)
		addTargetAlias(aliases, "talk", "talk_"+item.FeedID, canonicalID)
		addTargetAlias(aliases, "talk", activityTargetID("talk", *item), canonicalID)
	}
	return aliases
}

func resolveTargetID(targetType, targetID string, aliases map[string]string) string {
	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		return ""
	}
	if canonical, ok := aliases[targetKey(targetType, targetID)]; ok && canonical != "" {
		return canonical
	}
	return targetID
}

func loadTargetAliases(ctx context.Context, userQQ string) (map[string]string, error) {
	var activities []*model.Activity
	var talks []*model.Talk
	if err := database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Find(&activities).Error; err != nil {
		return nil, err
	}
	if err := database.DB.WithContext(ctx).Where("user_qq = ?", userQQ).Find(&talks).Error; err != nil {
		return nil, err
	}
	talkMap := make(map[string]*model.Talk, len(talks))
	for _, item := range talks {
		if item != nil {
			talkMap[item.TalkID] = item
		}
	}
	return buildTargetAliases(userQQ, activities, talkMap), nil
}

func collectResolvedTargetIDs(targetType, targetID string, aliases map[string]string) []string {
	canonical := resolveTargetID(targetType, targetID, aliases)
	seen := map[string]struct{}{
		targetID:  {},
		canonical: {},
	}
	for aliasKey, aliasTargetID := range aliases {
		if !strings.HasPrefix(aliasKey, targetType+":") {
			continue
		}
		if aliasTargetID != canonical {
			continue
		}
		aliasID := strings.TrimPrefix(aliasKey, targetType+":")
		if aliasID != "" {
			seen[aliasID] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		if strings.TrimSpace(id) != "" {
			out = append(out, id)
		}
	}
	return out
}

func findMemoryTarget(ctx context.Context, userQQ, targetType, targetID string) (*MemoryItem, error) {
	items, err := buildMemoryTimeline(ctx, userQQ, "content")
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		if (item.Type == targetType || item.Subtype == targetType || item.TargetType == targetType) &&
			(item.ID == targetID || item.TargetID == targetID) {
			return item, nil
		}
	}
	return nil, nil
}

func buildTargetPreviews(likes []*model.Like, comments []*model.Comment, shares []*model.Share, aliases map[string]string) map[string]memoryTargetPreview {
	previews := make(map[string]memoryTargetPreview)

	for _, item := range likes {
		if item == nil {
			continue
		}
		targetID := resolveTargetID(item.TargetType, item.TargetID, aliases)
		if targetID == "" {
			continue
		}
		key := targetKey(item.TargetType, targetID)
		preview := previews[key]
		preview.LikeCount++
		preview.Likes = append(preview.Likes, MemoryLikePreview{
			QQ:     item.LikerQQ,
			Name:   firstNonEmpty(item.LikerName, item.LikerQQ),
			Avatar: item.LikerAvatar,
			Time:   item.LikeTime,
		})
		previews[key] = preview
	}

	for _, item := range comments {
		if item == nil {
			continue
		}
		targetID := resolveTargetID(item.TargetType, item.TargetID, aliases)
		if targetID == "" {
			continue
		}
		key := targetKey(item.TargetType, targetID)
		preview := previews[key]
		preview.CommentCount++
		preview.Comments = append(preview.Comments, MemoryCommentPreview{
			QQ:          item.AuthorQQ,
			Name:        firstNonEmpty(item.AuthorName, item.AuthorQQ),
			Avatar:      item.AuthorAvatar,
			Content:     item.Content,
			ReplyToQQ:   item.ReplyToQQ,
			ReplyToName: item.ReplyToName,
			Time:        item.CommentTime,
			IsDeleted:   item.IsDeleted,
		})
		previews[key] = preview
	}

	for _, item := range shares {
		if item == nil {
			continue
		}
		targetID := resolveTargetID(item.TargetType, item.TargetID, aliases)
		if targetID == "" {
			continue
		}
		key := targetKey(item.TargetType, targetID)
		preview := previews[key]
		preview.ShareCount++
		preview.Shares = append(preview.Shares, MemorySharePreview{
			QQ:      item.SharerQQ,
			Name:    firstNonEmpty(item.SharerName, item.SharerQQ),
			Avatar:  item.SharerAvatar,
			Comment: item.Comment,
			Time:    item.ShareTime,
		})
		previews[key] = preview
	}

	for key, preview := range previews {
		sort.SliceStable(preview.Likes, func(i, j int) bool {
			return preview.Likes[i].Time.After(preview.Likes[j].Time)
		})
		sort.SliceStable(preview.Comments, func(i, j int) bool {
			return preview.Comments[i].Time.After(preview.Comments[j].Time)
		})
		if len(preview.Likes) > 3 {
			preview.Likes = preview.Likes[:3]
		}
		if len(preview.Comments) > 2 {
			preview.Comments = preview.Comments[:2]
		}
		sort.SliceStable(preview.Shares, func(i, j int) bool {
			return preview.Shares[i].Time.After(preview.Shares[j].Time)
		})
		if len(preview.Shares) > 2 {
			preview.Shares = preview.Shares[:2]
		}
		previews[key] = preview
	}
	return previews
}

// contentMemoryTypes 「你自己的内容」——回忆时间线默认只展示这些。
var contentMemoryTypes = map[string]bool{
	"talk": true, "blog": true, "album": true, "photo": true,
	"message": true, "diary": true, "video": true, "favorite": true,
}

// interactionMemoryTypes 「别人对我的互动」——单独成一个「互动」视角。
var interactionMemoryTypes = map[string]bool{
	"like": true, "comment": true, "visitor": true, "mention": true, "share": true,
}

func filterByMemoryTypes(items []*MemoryItem, allow map[string]bool) []*MemoryItem {
	out := make([]*MemoryItem, 0, len(items))
	for _, item := range items {
		if allow[item.Type] {
			out = append(out, item)
		}
	}
	return out
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
