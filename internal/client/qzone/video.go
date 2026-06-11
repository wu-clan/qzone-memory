package qzone

import (
	"math/rand"
	"net/url"
	"strconv"
	"time"
)

type VideoItem struct {
	VideoID      string
	Title        string
	Description  string
	URL          string
	PreviewURL   string
	Width        int
	Height       int
	Duration     int
	CommentCount int
	UploadTime   time.Time
}

func (c *Client) GetVideos(offset, limit int) ([]VideoItem, error) {
	if limit <= 0 {
		limit = 20
	}
	params := url.Values{
		"g_tk":        {c.gtk},
		"callback":    {"shine0_Callback"},
		"t":           {strconv.FormatFloat(rand.Float64(), 'f', 9, 64)},
		"uin":         {c.qq},
		"hostUin":     {c.qq},
		"appid":       {"4"},
		"getMethod":   {"2"},
		"start":       {strconv.Itoa(offset)},
		"count":       {strconv.Itoa(limit)},
		"need_old":    {"0"},
		"getUserInfo": {"0"},
		"inCharset":   {"utf-8"},
		"outCharset":  {"utf-8"},
		"refer":       {"qzone"},
		"source":      {"qzone"},
		"callbackFun": {"shine0"},
		"_":           {strconv.FormatInt(time.Now().UnixMilli(), 10)},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/taotao.qq.com/cgi-bin/video_get_data"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return nil, err
	}
	result, err := parseCallback(data)
	if err != nil {
		return nil, err
	}

	var items []VideoItem
	for _, raw := range getSliceValue(result, "videos") {
		vMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		item := VideoItem{
			VideoID:      getStringValue(vMap, "tid"),
			Title:        cleanPlainText(getStringValue(vMap, "title")),
			Description:  cleanPlainText(getStringValue(vMap, "desc")),
			URL:          getStringValue(vMap, "url"),
			PreviewURL:   getStringValue(vMap, "pre"),
			Width:        getIntValue(vMap, "width"),
			Height:       getIntValue(vMap, "height"),
			Duration:     getIntValue(vMap, "duration"),
			CommentCount: getIntValue(vMap, "cmtnum"),
		}
		if ts := getInt64Value(vMap, "uploadtime"); ts > 0 {
			item.UploadTime = time.Unix(ts, 0)
		}
		if item.VideoID != "" {
			items = append(items, item)
		}
	}
	return items, nil
}
