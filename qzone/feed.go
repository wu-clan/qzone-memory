package qzone

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// FeedItem 动态时间线数据项
type FeedItem struct {
	FeedID       string
	FeedType     string // "talk"/"blog"/"photo"/"share"/"other"
	ObjectID     string
	Title        string
	Content      string
	HTMLContent  string
	AuthorQQ     string
	AuthorName   string
	Images       []string
	LikeCount    int
	CommentCount int
	ShareCount   int
	IsDeleted    bool
	PublishTime  time.Time
	StateText    string
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
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			sleepWithJitter(time.Duration(attempt)*1200*time.Millisecond, 600*time.Millisecond)
		}

		data, err := c.request("GET", apiURL, params)
		if err != nil {
			lastErr = err
			continue
		}

		feeds, err := c.parseFeedResponse(data)
		if err == nil {
			return feeds, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return []FeedItem{}, nil
}

func (c *Client) parseFeedResponse(data []byte) ([]FeedItem, error) {
	legacyFeeds := parseLegacyFeedHTML(c.qq, data)

	result, err := parseCallback(data)
	if err != nil {
		if len(legacyFeeds) > 0 {
			return legacyFeeds, nil
		}
		return []FeedItem{}, nil
	}
	if err := checkResponseCode(result, "获取历史动态失败"); err != nil {
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
			ObjectID:    pickFirstNonEmpty(getStringValue(feedMap, "tid"), getStringValue(feedMap, "unikey"), getStringValue(feedMap, "curkey")),
			Title:       cleanText(pickFirstNonEmpty(getStringValue(feedMap, "title"), getStringValue(feedMap, "name"))),
			Content:     getStringValue(feedMap, "summary"),
			HTMLContent: getStringValue(feedMap, "html"),
			AuthorQQ:    pickFirstNonEmpty(getStringValue(feedMap, "uin"), fmt.Sprintf("%d", getInt64Value(feedMap, "uin"))),
			AuthorName:  getStringValue(feedMap, "nickname"),
		}

		appID := getIntValue(feedMap, "appid")
		switch appID {
		case 311:
			feed.FeedType = "talk"
		case 4:
			feed.FeedType = "blog"
		case 2:
			feed.FeedType = "photo"
		case 1:
			feed.FeedType = "share"
		default:
			feed.FeedType = "other"
		}

		if absTime := getInt64Value(feedMap, "abstime"); absTime > 0 {
			feed.PublishTime = time.Unix(absTime, 0)
		}

		if pics := getSliceValue(feedMap, "pic"); pics != nil {
			for _, pic := range pics {
				if picMap, ok := pic.(map[string]interface{}); ok {
					if picURL := getStringValue(picMap, "url"); picURL != "" {
						feed.Images = append(feed.Images, picURL)
					}
				}
			}
		}

		if cmtData := getMapValue(feedMap, "comment"); cmtData != nil {
			feed.CommentCount = getIntValue(cmtData, "num")
		}
		feed.LikeCount = getIntValue(feedMap, "likeTotal")
		feed.ShareCount = getIntValue(feedMap, "fwdnum")

		feed.enrichFromHTML()
		feed.inferType()
		feed.inferDeleted()
		feed.ensureID()

		feeds = append(feeds, feed)
	}

	return mergeFeedItems(feeds, legacyFeeds), nil
}

func sleepWithJitter(base, jitter time.Duration) {
	if base < 0 {
		base = 0
	}
	if jitter <= 0 {
		time.Sleep(base)
		return
	}
	extra := time.Duration(rand.Int63n(int64(jitter)))
	time.Sleep(base + extra)
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
		"h4.f-title",
		"div.f-single-head",
		"div.f-info",
		"div.f-info-content",
		"div.txt-prewrap",
		"p.txt-box-title",
		"div.f-single-content",
	}
	for _, selector := range textSelectors {
		text := cleanText(doc.Find(selector).First().Text())
		if text != "" {
			if f.Title == "" && (selector == "h4.f-title" || selector == "div.f-single-head") {
				f.Title = text
			}
			if f.Content == "" || selector != "h4.f-title" {
				f.Content = text
			}
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

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		if f.ObjectID == "" {
			f.ObjectID = extractObjectID(href)
		}
		if f.Title == "" {
			text := cleanText(s.Text())
			if text != "" {
				f.Title = text
			}
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

func (f *FeedItem) inferDeleted() {
	if f.IsDeleted {
		return
	}

	deletedTexts := []string{
		"已删除",
		"被删除",
		"已不存在",
		"已失效",
		"无权查看",
		"不可访问",
	}
	state := f.StateText + " " + f.Content + " " + f.Title
	for _, text := range deletedTexts {
		if strings.Contains(state, text) {
			f.IsDeleted = true
			return
		}
	}
}

func (f *FeedItem) ensureID() {
	if strings.TrimSpace(f.FeedID) != "" {
		return
	}

	sum := md5.Sum([]byte(strings.Join([]string{
		f.FeedType,
		f.ObjectID,
		f.AuthorQQ,
		f.Title,
		f.Content,
		f.PublishTime.Format(time.RFC3339),
	}, "|")))
	f.FeedID = hex.EncodeToString(sum[:])
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

var objectIDPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?:blogid|blogId)=([0-9]+)`),
	regexp.MustCompile(`/blog/[^/]+/([0-9]+)`),
	regexp.MustCompile(`(?:topicId|topicid|tid|curkey|unikey)=([A-Za-z0-9_\-]+)`),
	regexp.MustCompile(`/mood/([A-Za-z0-9_\-]+)`),
	regexp.MustCompile(`/photo/[^/]+/album/([A-Za-z0-9_\-]+)`),
	regexp.MustCompile(`/([A-Za-z0-9_\-]{16,})`),
}

func extractObjectID(raw string) string {
	for _, pattern := range objectIDPatterns {
		if matches := pattern.FindStringSubmatch(raw); len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

var legacyHexPattern = regexp.MustCompile(`\\x[0-9a-fA-F]{2}`)

func parseLegacyFeedHTML(receiverQQ string, raw []byte) []FeedItem {
	body := decodeLegacyHTML(raw)
	if strings.TrimSpace(body) == "" || !strings.Contains(body, "f-single") {
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil
	}

	feeds := make([]FeedItem, 0)
	doc.Find("li.f-single, li.f-single.f-s-s").Each(func(i int, s *goquery.Selection) {
		feed := FeedItem{}
		if id, ok := s.Attr("data-key"); ok {
			feed.FeedID = strings.TrimSpace(id)
		}
		if sender := s.Find("a.f-name.q_namecard").First(); sender.Length() > 0 {
			feed.AuthorName = cleanText(sender.Text())
			feed.AuthorQQ = strings.TrimPrefix(sender.AttrOr("link", ""), "nameCard_")
			if feed.ObjectID == "" {
				feed.ObjectID = extractObjectID(sender.AttrOr("href", ""))
			}
		}
		feed.StateText = cleanText(s.Find("span.state").First().Text())
		if t := cleanText(s.Find("div.info-detail").First().Text()); t != "" {
			feed.PublishTime = parseLooseFeedTime(t)
		}
		if title := cleanText(s.Find("p.txt-box-title, h4.f-title").First().Text()); title != "" {
			feed.Title = title
		}
		if content := cleanText(s.Find("div.f-info, div.f-info-content, div.txt-prewrap, div.f-single-content").First().Text()); content != "" {
			feed.Content = content
		}
		s.Find("a.img-item img, div.media-box img, img").Each(func(_ int, img *goquery.Selection) {
			src := strings.TrimSpace(img.AttrOr("src", ""))
			if src != "" && !containsString(feed.Images, src) {
				feed.Images = append(feed.Images, src)
			}
		})
		s.Find("a").Each(func(_ int, a *goquery.Selection) {
			if feed.ObjectID == "" {
				feed.ObjectID = extractObjectID(a.AttrOr("href", ""))
			}
		})
		feed.inferType()
		feed.inferDeleted()
		feed.ensureID()
		if feed.AuthorQQ == "" {
			feed.AuthorQQ = receiverQQ
		}
		if feed.FeedID != "" || feed.Content != "" || feed.Title != "" {
			feeds = append(feeds, feed)
		}
	})
	return feeds
}

func decodeLegacyHTML(raw []byte) string {
	text := string(raw)
	start := strings.Index(text, "html:'")
	if start == -1 {
		start = strings.Index(text, `html:"`)
	}
	if start == -1 {
		return ""
	}

	quote := text[start+5]
	content := text[start+6:]
	endToken := string([]byte{quote}) + ",opuin"
	end := strings.Index(content, endToken)
	if end == -1 {
		endToken = string([]byte{quote}) + ", opuin"
		end = strings.Index(content, endToken)
	}
	if end == -1 {
		return ""
	}

	content = content[:end]
	content = legacyHexPattern.ReplaceAllStringFunc(content, func(hexValue string) string {
		n, err := strconv.ParseUint(hexValue[2:], 16, 8)
		if err != nil {
			return hexValue
		}
		return string(rune(n))
	})
	content = strings.ReplaceAll(content, `\/`, `/`)
	content = strings.ReplaceAll(content, `\"`, `"`)
	content = strings.ReplaceAll(content, `\'`, `'`)
	content = strings.ReplaceAll(content, `\\`, `\`)
	content = strings.ReplaceAll(content, `\n`, " ")
	content = strings.ReplaceAll(content, `\r`, " ")
	content = strings.ReplaceAll(content, `\t`, " ")
	return content
}

func parseLooseFeedTime(text string) time.Time {
	now := time.Now()
	layouts := []string{
		"2006年1月2日 15:04",
		"2006年01月02日 15:04",
		"1月2日 15:04",
		"01月02日 15:04",
		"昨天 15:04",
		"15:04",
	}
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, text, time.Local)
		if err != nil {
			continue
		}
		switch layout {
		case "2006年1月2日 15:04", "2006年01月02日 15:04":
			return t
		case "1月2日 15:04", "01月02日 15:04":
			return time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
		case "昨天 15:04":
			y := now.AddDate(0, 0, -1)
			return time.Date(y.Year(), y.Month(), y.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
		case "15:04":
			return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
		}
	}
	return time.Time{}
}

func mergeFeedItems(primary, fallback []FeedItem) []FeedItem {
	if len(primary) == 0 {
		return fallback
	}
	index := make(map[string]int, len(primary))
	for i, item := range primary {
		key := firstNonEmptyForMerge(item.FeedID, item.ObjectID, item.Title+"|"+item.Content)
		if key != "" {
			index[key] = i
		}
	}
	for _, item := range fallback {
		key := firstNonEmptyForMerge(item.FeedID, item.ObjectID, item.Title+"|"+item.Content)
		if pos, ok := index[key]; ok {
			current := &primary[pos]
			if current.Title == "" {
				current.Title = item.Title
			}
			if current.Content == "" {
				current.Content = item.Content
			}
			if current.AuthorQQ == "" {
				current.AuthorQQ = item.AuthorQQ
			}
			if current.AuthorName == "" {
				current.AuthorName = item.AuthorName
			}
			if current.StateText == "" {
				current.StateText = item.StateText
			}
			if current.PublishTime.IsZero() {
				current.PublishTime = item.PublishTime
			}
			for _, img := range item.Images {
				if !containsString(current.Images, img) {
					current.Images = append(current.Images, img)
				}
			}
			if item.IsDeleted {
				current.IsDeleted = true
			}
			continue
		}
		primary = append(primary, item)
	}
	return primary
}

func firstNonEmptyForMerge(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && value != "|" {
			return value
		}
	}
	return ""
}
