package qzone

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/qzone-memory/pkg/logger"
	"go.uber.org/zap"
)

// TalkItem 说说数据项
type TalkItem struct {
	TalkID       string
	Content      string
	Images       []string
	Videos       []string
	Location     string
	Device       string
	LikeCount    int
	CommentCount int
	ShareCount   int
	PublishTime  time.Time
}

// GetTalks 获取说说列表
func (c *Client) GetTalks(offset, limit int) ([]TalkItem, error) {
	params := url.Values{
		"uin":                  {c.qq},
		"hostUin":              {c.qq},
		"ftype":                {"0"},
		"sort":                 {"0"},
		"pos":                  {strconv.Itoa(offset)},
		"num":                  {strconv.Itoa(limit)},
		"replynum":             {"100"},
		"g_tk":                 {c.gtk},
		"callback":             {"_preloadCallback"},
		"code_version":         {"1"},
		"format":               {"json"},
		"need_comment":         {"1"},
		"need_private_comment": {"1"},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/taotao.qq.com/cgi-bin/emotion_cgi_msglist_v6"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return nil, err
	}

	result, err := parseCallback(data)
	if err != nil {
		return nil, err
	}

	// 诊断：打印解析后 JSON 顶层键和 msglist 类型
	keys := make([]string, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}
	msglistRaw := result["msglist"]
	logger.Warn("GetTalks 解析结果", zap.Strings("keys", keys), zap.String("msglist_type", fmt.Sprintf("%T", msglistRaw)), zap.Bool("msglist_nil", msglistRaw == nil))
	if msglistRaw != nil {
		if arr, ok := msglistRaw.([]interface{}); ok {
			logger.Warn("GetTalks msglist", zap.Int("len", len(arr)))
			if len(arr) > 0 {
				if first, ok := arr[0].(map[string]interface{}); ok {
					firstKeys := make([]string, 0, len(first))
					for k := range first {
						firstKeys = append(firstKeys, k)
					}
					logger.Warn("GetTalks 第一条说说", zap.Strings("keys", firstKeys), zap.Any("tid", first["tid"]))
				}
			}
		} else {
			logger.Warn("GetTalks msglist 类型异常", zap.Any("value", msglistRaw))
		}
	}

	// 检查返回码
	if err := checkResponseCode(result, "获取说说失败"); err != nil {
		return nil, err
	}

	// 解析说说列表
	talks := []TalkItem{}
	msglist, ok := result["msglist"].([]interface{})
	if !ok {
		LogEmptyResponse("GetTalks", data)
		return talks, nil
	}

	for _, item := range msglist {
		msg, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		talk := TalkItem{
			TalkID:  getStringValue(msg, "tid"),
			Content: cleanPlainText(getStringValue(msg, "content")),
		}

		// 解析发布时间（优先 created_time，兼容 createTime）
		if createTime := getInt64Value(msg, "created_time"); createTime > 0 {
			talk.PublishTime = time.Unix(createTime, 0)
		} else if createTime, ok := msg["createTime"].(float64); ok && createTime > 0 {
			talk.PublishTime = time.Unix(int64(createTime), 0)
		}

		// 解析图片（优先 url2/url3 高清图，回退 url1）
		if pics, ok := msg["pic"].([]interface{}); ok {
			for _, pic := range pics {
				if picMap, ok := pic.(map[string]interface{}); ok {
					picURL := getStringValue(picMap, "url2")
					if picURL == "" {
						picURL = getStringValue(picMap, "url3")
					}
					if picURL == "" {
						picURL = getStringValue(picMap, "url1")
					}
					if picURL != "" {
						talk.Images = append(talk.Images, picURL)
					}
				}
			}
		}

		// 解析视频
		if vids, ok := msg["video"].([]interface{}); ok {
			for _, vid := range vids {
				if vidMap, ok := vid.(map[string]interface{}); ok {
					if videoURL := getStringValue(vidMap, "url3"); videoURL != "" {
						talk.Videos = append(talk.Videos, videoURL)
					}
				}
			}
		}

		// 解析位置
		if lbs := getMapValue(msg, "lbs"); lbs != nil {
			talk.Location = getStringValue(lbs, "idname")
		}

		// 解析来源设备
		talk.Device = getStringValue(msg, "source_name")

		// 解析统计数据
		talk.LikeCount = getIntValue(msg, "likeCount")
		talk.CommentCount = getIntValue(msg, "cmtnum")
		talk.ShareCount = getIntValue(msg, "fwdnum")

		if talk.TalkID != "" {
			talks = append(talks, talk)
		}
	}

	return talks, nil
}
