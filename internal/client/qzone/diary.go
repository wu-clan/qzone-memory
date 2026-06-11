package qzone

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

type DiaryItem struct {
	DiaryID    string
	Title      string
	Summary    string
	Content    string
	CreateTime time.Time
}

func (c *Client) GetDiaries(offset, limit int) ([]DiaryItem, error) {
	if limit <= 0 {
		limit = 15
	}
	params := url.Values{
		"uin":  {c.qq},
		"pos":  {strconv.Itoa(offset)},
		"num":  {strconv.Itoa(limit)},
		"g_tk": {c.gtk},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/b.qzone.qq.com/cgi-bin/privateblog/privateblog_get_titlelist"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return nil, err
	}
	result, err := parseCallback(data)
	if err != nil {
		return nil, err
	}
	if err := checkResponseCode(result, "获取私密日记失败"); err != nil {
		if strings.Contains(err.Error(), "code=-4003") {
			return []DiaryItem{}, nil
		}
		return nil, err
	}

	var items []DiaryItem
	for _, raw := range getSliceValue(result, "titlelist") {
		dMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		item := DiaryItem{
			DiaryID: getStringValue(dMap, "diaryid"),
			Title:   cleanPlainText(getStringValue(dMap, "title")),
			Summary: cleanPlainText(getStringValue(dMap, "summary")),
		}
		if ts := getInt64Value(dMap, "pubtime"); ts > 0 {
			item.CreateTime = time.Unix(ts, 0)
		}
		if item.DiaryID != "" {
			items = append(items, item)
		}
	}
	return items, nil
}

func (c *Client) GetDiaryDetail(diaryID string) (string, error) {
	params := url.Values{
		"uin":     {c.qq},
		"diaryid": {diaryID},
		"g_tk":    {c.gtk},
	}
	apiURL := "https://user.qzone.qq.com/proxy/domain/b.qzone.qq.com/cgi-bin/privateblog/privateblog_output_data"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return "", err
	}
	result, err := parseCallback(data)
	if err == nil {
		if text := getStringValue(result, "blogText"); text != "" {
			return cleanPlainText(text), nil
		}
	}
	return extractBlogHTMLContent(string(data)), nil
}
