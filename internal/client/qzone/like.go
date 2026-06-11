package qzone

import (
	"fmt"
	"net/url"
	"time"
)

// LikeItem 点赞数据项
type LikeItem struct {
	LikerQQ     string
	LikerName   string
	LikerAvatar string
	LikeTime    time.Time
}

// GetTalkLikes 获取说说点赞列表（真实 API: get_like_list_app）
func (c *Client) GetTalkLikes(talkID string) ([]LikeItem, error) {
	unikey := fmt.Sprintf("http://user.qzone.qq.com/%s/mood/%s", c.qq, talkID)

	params := url.Values{
		"uin":           {c.qq},
		"unikey":        {unikey},
		"begin_uin":     {"0"},
		"query_count":   {"100"},
		"if_first_page": {"1"},
		"g_tk":          {c.gtk},
		"format":        {"json"},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/users.qzone.qq.com/cgi-bin/likes/get_like_list_app"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return nil, err
	}

	result, err := parseCallback(data)
	if err != nil {
		return nil, err
	}

	var likes []LikeItem
	dataObj := getMapValue(result, "data")
	if dataObj == nil {
		return likes, nil
	}

	likeList := getSliceValue(dataObj, "like_uin_info")
	for _, item := range likeList {
		likeMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		like := LikeItem{
			LikerQQ:     fmt.Sprintf("%d", getInt64Value(likeMap, "fuin")),
			LikerName:   getStringValue(likeMap, "nick"),
			LikerAvatar: getStringValue(likeMap, "portrait"),
		}

		if likeTime := getInt64Value(likeMap, "if_gender_time"); likeTime > 0 {
			like.LikeTime = time.Unix(likeTime, 0)
		}

		likes = append(likes, like)
	}

	return likes, nil
}
