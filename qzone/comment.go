package qzone

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// CommentItem 评论数据项
type CommentItem struct {
	CommentID    string
	AuthorQQ     string
	AuthorName   string
	AuthorAvatar string
	Content      string
	ReplyToQQ    string
	ReplyToName  string
	CommentTime  time.Time
}

// GetTalkComments 获取说说评论列表（真实 API: emotion_cgi_getcmtreply_v6）
func (c *Client) GetTalkComments(talkID string, offset, limit int) ([]CommentItem, error) {
	params := url.Values{
		"uin":                  {c.qq},
		"hostUin":              {c.qq},
		"topicId":              {c.qq + "_" + talkID},
		"start":                {strconv.Itoa(offset)},
		"num":                  {strconv.Itoa(limit)},
		"order":                {"0"},
		"g_tk":                 {c.gtk},
		"format":               {"json"},
		"need_private_comment": {"1"},
		"inCharset":            {"utf-8"},
		"outCharset":           {"utf-8"},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/taotao.qzone.qq.com/cgi-bin/emotion_cgi_getcmtreply_v6"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return nil, err
	}

	result, err := parseCallback(data)
	if err != nil {
		return nil, err
	}

	dataObj := getMapValue(result, "data")
	if dataObj == nil {
		dataObj = result
	}
	return parseCommentItems(talkID, offset, getCommentList(dataObj))
}

// GetBlogComments 获取日志评论列表（真实 API: get_comment_list）
func (c *Client) GetBlogComments(blogID string, offset, limit int) ([]CommentItem, error) {
	params := url.Values{
		"uin":        {c.qq},
		"topicId":    {c.qq + "_" + blogID},
		"start":      {strconv.Itoa(offset)},
		"num":        {strconv.Itoa(limit)},
		"iNotice":    {"0"},
		"inCharset":  {"utf-8"},
		"outCharset": {"utf-8"},
		"format":     {"json"},
		"ref":        {"qzone"},
		"g_tk":       {c.gtk},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/b.qzone.qq.com/cgi-bin/blognew/get_comment_list"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return nil, err
	}

	result, err := parseCallback(data)
	if err != nil {
		return nil, err
	}

	dataObj := getMapValue(result, "data")
	if dataObj == nil {
		return nil, nil
	}

	return parseCommentItems("blog_"+blogID, offset, getCommentList(dataObj))
}

// GetPhotoComments 获取照片评论列表（真实 API: cgi_pcomment_xml_v2）
func (c *Client) GetPhotoComments(albumID, photoID string, offset, limit int) ([]CommentItem, error) {
	params := url.Values{
		"uin":        {c.qq},
		"hostUin":    {c.qq},
		"topicId":    {albumID + "_" + photoID},
		"start":      {strconv.Itoa(offset)},
		"num":        {strconv.Itoa(limit)},
		"order":      {"1"},
		"format":     {"json"},
		"g_tk":       {c.gtk},
		"inCharset":  {"utf-8"},
		"outCharset": {"utf-8"},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/app.photo.qzone.qq.com/cgi-bin/app/cgi_pcomment_xml_v2"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return nil, err
	}

	result, err := parseCallback(data)
	if err != nil {
		return nil, err
	}

	dataObj := getMapValue(result, "data")
	if dataObj == nil {
		dataObj = result
	}
	return parseCommentItems("photo_"+photoID, offset, getCommentList(dataObj))
}

func getCommentList(result map[string]interface{}) []interface{} {
	for _, key := range []string{"comments", "commentlist", "commentList"} {
		if items := getSliceValue(result, key); len(items) > 0 {
			return items
		}
	}
	return nil
}

func parseCommentItems(prefix string, offset int, commentList []interface{}) ([]CommentItem, error) {
	var comments []CommentItem
	for i, item := range commentList {
		cMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		poster := getMapValue(cMap, "poster")
		replyTo := getMapValue(cMap, "replyer")
		if replyTo == nil {
			replyTo = getMapValue(cMap, "replyUser")
		}
		if replyTo == nil {
			replyTo = getMapValue(cMap, "list_3")
		}

		commentID := pickCommentID(cMap)
		if commentID == "" {
			commentID = fmt.Sprintf("%s_%d_%d", prefix, offset, i)
		} else {
			commentID = prefix + "_" + commentID
		}

		comment := CommentItem{
			CommentID:    commentID,
			AuthorQQ:     pickCommentUserID(cMap, poster),
			AuthorName:   cleanPlainText(pickCommentUserName(cMap, poster)),
			AuthorAvatar: pickCommentUserAvatar(cMap, poster),
			Content:      cleanPlainText(getStringValue(cMap, "content")),
			ReplyToQQ:    pickCommentUserID(nil, replyTo),
			ReplyToName:  cleanPlainText(pickCommentUserName(nil, replyTo)),
		}

		if createTime := pickCommentTime(cMap); createTime > 0 {
			comment.CommentTime = time.Unix(createTime, 0)
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

func pickCommentID(comment map[string]interface{}) string {
	for _, key := range []string{"id", "cid", "commentid"} {
		if value := getStringValue(comment, key); value != "" {
			return value
		}
		if value := getInt64Value(comment, key); value > 0 {
			return strconv.FormatInt(value, 10)
		}
	}
	return ""
}

func pickCommentTime(comment map[string]interface{}) int64 {
	for _, key := range []string{"postTime", "createTime2", "create_time", "createTime"} {
		if value := getInt64Value(comment, key); value > 0 {
			return value
		}
	}
	return 0
}

func pickCommentUserID(comment map[string]interface{}, nested map[string]interface{}) string {
	for _, source := range []map[string]interface{}{comment, nested} {
		if source == nil {
			continue
		}
		for _, key := range []string{"uin", "id"} {
			if value := getStringValue(source, key); value != "" && value != "0" {
				return value
			}
			if value := getInt64Value(source, key); value > 0 {
				return strconv.FormatInt(value, 10)
			}
		}
	}
	return ""
}

func pickCommentUserName(comment map[string]interface{}, nested map[string]interface{}) string {
	for _, source := range []map[string]interface{}{comment, nested} {
		if source == nil {
			continue
		}
		for _, key := range []string{"name", "nick", "nickname"} {
			if value := getStringValue(source, key); value != "" {
				return value
			}
		}
	}
	return ""
}

func pickCommentUserAvatar(comment map[string]interface{}, nested map[string]interface{}) string {
	for _, source := range []map[string]interface{}{comment, nested} {
		if source == nil {
			continue
		}
		for _, key := range []string{"portrait", "avatar"} {
			if value := getStringValue(source, key); value != "" {
				return value
			}
		}
	}
	return ""
}
