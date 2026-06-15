package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"

	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/model"
)

// ReprocessResult 重解析结果
type ReprocessResult struct {
	QQ          string `json:"qq"`
	TalksBefore int64  `json:"talks_before"`
	TalksAfter  int64  `json:"talks_after"`
	Recovered   int    `json:"recovered"`
	Spam        int    `json:"spam"`
	Likes       int    `json:"likes"`
	Comments    int    `json:"comments"`
}

// stripInvisible 去掉零宽 / 不可见字符（QQ 空间文本里常见的 ⁢ 等）。
func stripInvisible(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 0x200b && r <= 0x200f:
			return -1
		case r >= 0x2060 && r <= 0x2064:
			return -1
		case r == 0xfeff:
			return -1
		}
		return r
	}, s)
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r)
}

// normalizeContent 规整正文：去零宽字符、合并空白，并去掉汉字之间被 HTML 逐字拆 span
// 带出来的多余空格（如"老婆 不 在 家" → "老婆不在家"），但保留 @提及 / 英文 的正常空格。
func normalizeContent(s string) string {
	s = stripInvisible(s)
	s = strings.Join(strings.Fields(s), " ") // 多空白 → 单空格
	runes := []rune(s)
	var b strings.Builder
	for i := 0; i < len(runes); i++ {
		if runes[i] == ' ' && i > 0 && i+1 < len(runes) && isCJK(runes[i-1]) && isCJK(runes[i+1]) {
			continue
		}
		b.WriteRune(runes[i])
	}
	return strings.TrimSpace(b.String())
}

// extractUserContentFromFeed 从"X赞了我的说说 ： <正文>"这类动态文本里剥离动作前缀，
// 取出真正属于用户的正文（已规整）；无冒号（如"发表说说"）则视为无可恢复正文。
func extractUserContentFromFeed(content string) string {
	c := normalizeContent(content)
	for _, sep := range []string{"：", ":"} {
		if idx := strings.Index(c, sep); idx >= 0 {
			if after := strings.TrimSpace(c[idx+len(sep):]); after != "" {
				return after
			}
		}
	}
	return ""
}

// contentTalkID 按规整后的正文生成稳定的说说 ID——同一条说说无论被多少人赞、
// 出现在多少条动态里，都会归并成同一条，赞也能对齐到它。
func contentTalkID(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "talk_c" + hex.EncodeToString(sum[:8])
}

// spamMarkers 游戏自动说说 / 广告 / 微信号等噪音特征。
var spamMarkers = []string{
	"王者荣耀", "和平精英", "qq炫舞", "穿越火线", "天天爱消除", "欢乐斗地主", "球球大作战",
	"wx:", "wx：", "vx:", "vx：", "微信:", "微信：", "加我微信", "贴膜", "代练", "刷钻",
}

func isSpamContent(content string) bool {
	c := strings.ToLower(normalizeContent(content))
	if c == "" {
		return false
	}
	for _, m := range spamMarkers {
		if strings.Contains(c, strings.ToLower(m)) {
			return true
		}
	}
	return false
}

func activityUserContent(qq string, item model.Activity) string {
	if item.AuthorQQ != "" && item.AuthorQQ != qq {
		return extractUserContentFromFeed(item.Content)
	}
	return normalizeContent(item.Content)
}

func canonicalActivityTargetID(qq, targetType string, item model.Activity) string {
	if targetType == "talk" {
		if content := activityUserContent(qq, item); content != "" {
			return contentTalkID(content)
		}
	}
	return activityTargetID(targetType, item)
}

// ReprocessActivities 用修正后的逻辑重新解析已存动态（v2）：
//  1. 剥离"别人赞我"前缀、规整正文（去字间空格/零宽）；
//  2. 按内容生成稳定 ID → 同一条说说去重；
//  3. 重建的赞对齐到内容 ID（修复孤儿赞）；
//  4. 游戏/广告/微信号等标记为 spam（默认在回忆里隐藏，不删除）。
//
// 仅影响重建数据(is_deleted=1)与其相关的赞，真实抓取的内容(is_deleted=0)及其赞受保护。
// 不需重新登录，对现有库即生效。
func ReprocessActivities(ctx context.Context, qq string) (*ReprocessResult, error) {
	if err := validateQQ(qq); err != nil {
		return nil, err
	}
	db := database.DB.WithContext(ctx)

	var before int64
	db.Model(&model.Talk{}).Where("user_qq = ?", qq).Count(&before)

	// 真实抓取的说说受保护，连同指向它们的赞一并保留
	var realIDs []string
	db.Model(&model.Talk{}).Where("user_qq = ? AND is_deleted = ?", qq, false).Pluck("talk_id", &realIDs)

	// 清掉旧的重建说说与重建/孤儿赞/历史评论，稍后按对齐后的 ID 重建
	if err := db.Where("user_qq = ? AND is_deleted = ?", qq, true).Delete(&model.Talk{}).Error; err != nil {
		return nil, err
	}
	likeDel := db.Where("user_qq = ? AND target_type = ?", qq, "talk")
	if len(realIDs) > 0 {
		likeDel = likeDel.Where("target_id NOT IN ?", realIDs)
	}
	likeDel.Delete(&model.Like{})
	db.Where("user_qq = ? AND target_type = ? AND is_deleted = ?", qq, "talk", true).Delete(&model.Comment{})

	var acts []model.Activity
	if err := db.Where("user_qq = ?", qq).Find(&acts).Error; err != nil {
		return nil, err
	}

	talksByID := make(map[string]*model.Talk)
	likes := make([]*model.Like, 0)
	comments := make([]*model.Comment, 0)
	for _, a := range acts {
		if a.FeedType != "talk" {
			continue
		}

		content := activityUserContent(qq, a)

		target := ""
		if content != "" {
			tid := contentTalkID(content)
			target = tid
			if t, ok := talksByID[tid]; ok {
				if !a.PublishTime.IsZero() && (t.PublishTime.IsZero() || a.PublishTime.Before(t.PublishTime)) {
					t.PublishTime = a.PublishTime // 保留最早时间
				}
			} else {
				talksByID[tid] = &model.Talk{
					UserQQ:      qq,
					TalkID:      tid,
					Content:     content,
					Images:      a.Images,
					Videos:      "[]",
					IsDeleted:   true,
					IsSpam:      isSpamContent(content),
					LikeCount:   a.LikeCount,
					PublishTime: a.PublishTime,
				}
			}
		}

		if strings.Contains(a.StateText, "赞了我的说说") && a.AuthorQQ != "" {
			likes = append(likes, &model.Like{
				UserQQ:      qq,
				LikeID:      "like_" + activityTargetID("talk", a),
				TargetType:  "talk",
				TargetID:    target,
				LikerQQ:     a.AuthorQQ,
				LikerName:   firstNonEmpty(a.AuthorName, a.AuthorQQ),
				LikerAvatar: fmt.Sprintf("https://q.qlogo.cn/headimg_dl?dst_uin=%s&spec=100", a.AuthorQQ),
				LikeTime:    a.PublishTime,
			})
		}
		if isHistoricalCommentState(a) {
			if target == "" {
				target = activityTargetID("talk", a)
			}
			commentDetail := extractHistoricalCommentDetail(a)
			authorQQ := firstNonEmpty(commentDetail.AuthorQQ, a.AuthorQQ)
			authorName := firstNonEmpty(commentDetail.AuthorName, a.AuthorName, authorQQ)
			commentContent := firstNonEmpty(commentDetail.Content, a.Title, a.Content, "评论了你的历史内容")
			comments = append(comments, &model.Comment{
				UserQQ:       qq,
				CommentID:    "comment_" + firstNonEmpty(a.FeedID, a.ObjectID),
				TargetType:   "talk",
				TargetID:     target,
				AuthorQQ:     authorQQ,
				AuthorName:   authorName,
				AuthorAvatar: fmt.Sprintf("https://q.qlogo.cn/headimg_dl?dst_uin=%s&spec=100", authorQQ),
				Content:      normalizeContent(commentContent),
				ReplyToQQ:    commentDetail.ReplyToQQ,
				ReplyToName:  commentDetail.ReplyToName,
				IsDeleted:    true,
				CommentTime:  a.PublishTime,
			})
		}
	}

	talks := make([]*model.Talk, 0, len(talksByID))
	spam := 0
	for _, t := range talksByID {
		if t.IsSpam {
			spam++
		}
		talks = append(talks, t)
	}
	if err := dao.BatchUpsertTalks(talks); err != nil {
		return nil, err
	}
	if len(likes) > 0 {
		if err := dao.BatchUpsertLikes(likes); err != nil {
			return nil, err
		}
	}
	if len(comments) > 0 {
		if err := dao.BatchUpsertComments(comments); err != nil {
			return nil, err
		}
	}
	_ = dedupeHistoricalLikes(qq)

	// 真实抓取的说说也按内容刷新 spam 标记
	var allTalks []model.Talk
	db.Where("user_qq = ?", qq).Find(&allTalks)
	for _, t := range allTalks {
		if s := isSpamContent(t.Content); s != t.IsSpam {
			db.Model(&model.Talk{}).Where("id = ?", t.ID).Update("is_spam", s)
		}
	}

	var after int64
	db.Model(&model.Talk{}).Where("user_qq = ?", qq).Count(&after)
	return &ReprocessResult{QQ: qq, TalksBefore: before, TalksAfter: after, Recovered: len(talks), Spam: spam, Likes: len(likes), Comments: len(comments)}, nil
}
