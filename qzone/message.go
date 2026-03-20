package qzone

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// MessageItem 留言数据项
type MessageItem struct {
	MessageID    string
	AuthorQQ     string
	AuthorName   string
	AuthorAvatar string
	Content      string
	ReplyContent string
	MessageTime  time.Time
}

// GetMessages 获取留言板消息
func (c *Client) GetMessages(offset, limit int) ([]MessageItem, error) {
	params := url.Values{
		"uin":        {c.qq},
		"hostUin":    {c.qq},
		"start":      {strconv.Itoa(offset)},
		"num":        {strconv.Itoa(limit)},
		"g_tk":       {c.gtk},
		"format":     {"json"},
		"inCharset":  {"utf-8"},
		"outCharset": {"utf-8"},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/m.qzone.qq.com/cgi-bin/new/get_msgb"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return nil, err
	}

	result, err := parseCallback(data)
	if err != nil {
		return nil, err
	}

	if err := checkResponseCode(result, "获取留言失败"); err != nil {
		return nil, err
	}

	var messages []MessageItem
	dataObj := getMapValue(result, "data")
	if dataObj == nil {
		return messages, nil
	}

	msgList := getSliceValue(dataObj, "commentList")
	for _, item := range msgList {
		msgMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		msg := MessageItem{
			MessageID:  fmt.Sprintf("%d", getInt64Value(msgMap, "id")),
			AuthorQQ:   fmt.Sprintf("%d", getInt64Value(msgMap, "uin")),
			AuthorName: cleanPlainText(getStringValue(msgMap, "nickname")),
		}

		// 解析头像
		if avatarURL := getStringValue(msgMap, "avatar"); avatarURL != "" {
			msg.AuthorAvatar = avatarURL
		}

		// 解析内容
		msg.Content = cleanPlainText(getStringValue(msgMap, "htmlContent"))
		if msg.Content == "" {
			msg.Content = cleanPlainText(getStringValue(msgMap, "ubbContent"))
		}

		// 解析回复
		if replyList := getSliceValue(msgMap, "replyList"); len(replyList) > 0 {
			if reply, ok := replyList[0].(map[string]interface{}); ok {
				msg.ReplyContent = cleanPlainText(getStringValue(reply, "content"))
			}
		}

		// 解析时间
		if pubTime := getInt64Value(msgMap, "pubtime"); pubTime > 0 {
			msg.MessageTime = time.Unix(pubTime, 0)
		}

		if msg.MessageID == "" || msg.MessageID == "0" {
			continue
		}
		if msg.Content == "" && msg.ReplyContent == "" {
			continue
		}

		messages = append(messages, msg)
	}

	return messages, nil
}
