package qzone

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrCookieExpired Cookie 过期错误
var ErrCookieExpired = fmt.Errorf("Cookie 已过期，请重新登录")

// ShareItem 转发数据项
type ShareItem struct {
	ShareID    string
	SharerQQ   string
	SharerName string
	Comment    string
	ShareTime  time.Time
}

// GetShares 获取转发列表
func (c *Client) GetShares(targetType, targetID string, offset, limit int) ([]ShareItem, error) {
	params := url.Values{
		"uin":    {c.qq},
		"tid":    {targetID},
		"t":      {"1"},
		"begin":  {strconv.Itoa(offset)},
		"count":  {strconv.Itoa(limit)},
		"g_tk":   {c.gtk},
		"format": {"json"},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/taotao.qq.com/cgi-bin/emotion_cgi_get_fwd_v6"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(data)) == "" {
		return nil, nil
	}

	result, err := parseCallback(data)
	if err != nil {
		return nil, err
	}

	var shares []ShareItem
	fwdList := getSliceValue(result, "fwdlist")
	for i, item := range fwdList {
		fwdMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		share := ShareItem{
			ShareID:    fmt.Sprintf("%s_%d_%d", targetID, offset, i),
			SharerQQ:   fmt.Sprintf("%d", getInt64Value(fwdMap, "uin")),
			SharerName: getStringValue(fwdMap, "name"),
			Comment:    getStringValue(fwdMap, "content"),
		}

		if shareTime := getInt64Value(fwdMap, "createTime"); shareTime > 0 {
			share.ShareTime = time.Unix(shareTime, 0)
		}

		shares = append(shares, share)
	}

	return shares, nil
}

// MentionItem @提及数据项
type MentionItem struct {
	MentionID   string
	SourceType  string
	SourceID    string
	AuthorQQ    string
	AuthorName  string
	Content     string
	MentionTime time.Time
}

// GetMentions QQ 空间未发现可验证的独立 @ 提及接口，返回明确错误
func (c *Client) GetMentions(offset, limit int) ([]MentionItem, error) {
	return nil, fmt.Errorf("@提及没有可验证的独立接口，无法获取真实数据")
}
