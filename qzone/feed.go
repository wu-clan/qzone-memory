package qzone

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// FeedItem 动态时间线数据项
type FeedItem struct {
	FeedID      string
	FeedType    string // "talk"/"blog"/"photo"/"share"/"other"
	Content     string
	HTMLContent string
	AuthorQQ    string
	AuthorName  string
	Images      []string
	LikeCount    int
	CommentCount int
	ShareCount  int
	IsDeleted   bool
	PublishTime time.Time
	StateText   string
}

// GetFeeds 获取所有动态（含已删除数据的统一时间线）
func (c *Client) GetFeeds(offset, limit int) ([]FeedItem, error) {
	params := url.Values{
		"uin":                {c.qq},
		"begin_time":         {"0"},
		"end_time":           {"0"},
		"getappnotification": {"1"},
		"getnotifi":          {"1"},
		"has_get_key":        {"0"},
		"offset":             {strconv.Itoa(offset)},
		"set":                {"0"},
		"count":              {strconv.Itoa(limit)},
		"useutf8":            {"1"},
		"outputhtmlfeed":     {"1"},
		"scope":              {"1"},
		"format":             {"json"},
		"g_tk":               {c.gtk},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/ic2.qzone.qq.com/cgi-bin/feeds/feeds2_html_pav_all"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return nil, err
	}

	result, err := parseCallback(data)
	if err != nil {
		return nil, err
	}

	var feeds []FeedItem
	dataObj := getMapValue(result, "data")
	if dataObj == nil {
		return feeds, nil
	}

	feedList := getSliceValue(dataObj, "data")
	for _, item := range feedList {
		feedMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		feed := FeedItem{
			FeedID:      getStringValue(feedMap, "key"),
			Content:     getStringValue(feedMap, "summary"),
			HTMLContent: getStringValue(feedMap, "html"),
			AuthorQQ:    pickFirstNonEmpty(getStringValue(feedMap, "uin"), fmt.Sprintf("%d", getInt64Value(feedMap, "uin"))),
			AuthorName:  getStringValue(feedMap, "nickname"),
		}

		// 解析类型
		appID := getIntValue(feedMap, "appid")
		switch appID {
		case 311: // 说说
			feed.FeedType = "talk"
		case 4: // 日志
			feed.FeedType = "blog"
		case 2: // 相册/照片
			feed.FeedType = "photo"
		case 1: // 转发
			feed.FeedType = "share"
		default:
			feed.FeedType = "other"
		}

		// 解析时间
		if absTime := getInt64Value(feedMap, "abstime"); absTime > 0 {
			feed.PublishTime = time.Unix(absTime, 0)
		}

		// 解析图片
		if pics := getSliceValue(feedMap, "pic"); pics != nil {
			for _, pic := range pics {
				if picMap, ok := pic.(map[string]interface{}); ok {
					if picURL := getStringValue(picMap, "url"); picURL != "" {
						feed.Images = append(feed.Images, picURL)
					}
				}
			}
		}

		// 统计数据
		if cmtData := getMapValue(feedMap, "comment"); cmtData != nil {
			feed.CommentCount = getIntValue(cmtData, "num")
		}
		feed.LikeCount = getIntValue(feedMap, "likeTotal")
		feed.ShareCount = getIntValue(feedMap, "fwdnum")

		feed.enrichFromHTML()
		feed.inferType()

		feeds = append(feeds, feed)
	}

	return feeds, nil
}

func (f *FeedItem) enrichFromHTML() {
	if strings.TrimSpace(f.HTMLContent) == "" {
		return
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<div>" + f.HTMLContent + "</div>"))
	if err != nil {
		return
	}

	if strings.TrimSpace(f.AuthorName) == "" || f.AuthorName == "null" {
		if name := cleanText(doc.Find("a.f-name.q_namecard").First().Text()); name != "" && name != "null" {
			f.AuthorName = name
		}
	}

	if f.AuthorQQ == "" {
		if link, exists := doc.Find("a.f-name.q_namecard").First().Attr("link"); exists {
			f.AuthorQQ = strings.TrimPrefix(link, "nameCard_")
		}
	}

	f.StateText = cleanText(doc.Find("span.state").First().Text())

	textSelectors := []string{
		"div.f-info",
		"div.f-info-content",
		"div.txt-prewrap",
		"p.txt-box-title",
		"div.f-single-content",
	}
	for _, selector := range textSelectors {
		text := cleanText(doc.Find(selector).First().Text())
		if text != "" {
			f.Content = text
			break
		}
	}

	doc.Find("a.img-item img, div.media-box img, img").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists || strings.TrimSpace(src) == "" {
			return
		}
		if !containsString(f.Images, src) {
			f.Images = append(f.Images, src)
		}
	})
}

func (f *FeedItem) inferType() {
	if f.FeedType != "other" {
		return
	}

	state := f.StateText
	switch {
	case strings.Contains(state, "日志"):
		f.FeedType = "blog"
	case strings.Contains(state, "相册"), strings.Contains(state, "照片"):
		f.FeedType = "photo"
	case strings.Contains(state, "留言"):
		f.FeedType = "message"
	case strings.Contains(state, "转发"):
		f.FeedType = "share"
	case strings.Contains(state, "说说"), strings.Contains(state, "发表"):
		f.FeedType = "talk"
	}
}

func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ")
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func pickFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" && value != "0" {
			return value
		}
	}
	return ""
}
