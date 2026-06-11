package qzone

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// BlogItem 日志数据项
type BlogItem struct {
	BlogID       string
	Title        string
	Content      string
	Summary      string
	Category     string
	Tags         []string
	LikeCount    int
	CommentCount int
	ReadCount    int
	PublishTime  time.Time
}

// GetBlogList 获取日志列表
func (c *Client) GetBlogList(offset, limit int) ([]BlogItem, error) {
	params := url.Values{
		"uin":      {c.qq},
		"hostUin":  {c.qq},
		"blogType": {"0"},
		"cateName": {""},
		"cateHex":  {""},
		"statYear": {strconv.Itoa(time.Now().Year())},
		"verbose":  {"1"},
		"reqInfo":  {"7"},
		"pos":      {strconv.Itoa(offset)},
		"num":      {strconv.Itoa(limit)},
		"sortType": {"0"},
		"source":   {"0"},
		"rand":     {fmt.Sprintf("%.16f", float64(time.Now().UnixNano())/1e18)},
		"ref":      {"qzone"},
		"g_tk":     {c.gtk},
		"format":   {"json"},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/b.qzone.qq.com/cgi-bin/blognew/get_abs"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return nil, err
	}

	result, err := parseCallback(data)
	if err != nil {
		return nil, err
	}

	if err := checkResponseCode(result, "获取日志失败"); err != nil {
		// -4003 通常表示无权限访问日志（非 Cookie 过期），跳过而非终止
		if err.Error() == "获取日志失败: code=-4003" {
			return nil, nil
		}
		return nil, err
	}

	var blogs []BlogItem
	dataObj := getMapValue(result, "data")
	if dataObj == nil {
		return blogs, nil
	}

	blogList := getSliceValue(dataObj, "list")
	for _, item := range blogList {
		blogMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		blog := BlogItem{
			BlogID:       getNumericString(blogMap, "blogId"),
			Title:        cleanPlainText(getStringValue(blogMap, "title")),
			Summary:      cleanPlainText(getStringValue(blogMap, "abstract")),
			Category:     cleanPlainText(pickFirstBlogValue(getStringValue(blogMap, "cate"), getStringValue(blogMap, "category"))),
			ReadCount:    getIntValue(blogMap, "readCount"),
			LikeCount:    getIntValue(blogMap, "likeCount"),
			CommentCount: getIntValue(blogMap, "commentNum"),
		}

		if pubTime, err := time.ParseInLocation("2006-01-02 15:04", getStringValue(blogMap, "pubTime"), time.Local); err == nil {
			blog.PublishTime = pubTime
		} else if pubTime := getInt64Value(blogMap, "pubtime"); pubTime > 0 {
			blog.PublishTime = time.Unix(pubTime, 0)
		} else if pubTime := getInt64Value(blogMap, "lastModifyTime"); pubTime > 0 {
			blog.PublishTime = time.Unix(pubTime, 0)
		}

		blogs = append(blogs, blog)
	}

	return blogs, nil
}

// GetBlogContent 获取日志完整内容
func (c *Client) GetBlogContent(blogID string) (string, error) {
	params := url.Values{
		"uin":        {c.qq},
		"blogid":     {blogID},
		"styledm":    {"qzonestyle.gtimg.cn"},
		"imgdm":      {"qzs.qq.com"},
		"bdm":        {"b.qzone.qq.com"},
		"mode":       {"2"},
		"numperpage": {"50"},
		"timestamp":  {strconv.FormatInt(time.Now().Unix(), 10)},
		"ref":        {"qzone"},
		"page":       {"1"},
		"g_tk":       {c.gtk},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/b.qzone.qq.com/cgi-bin/blognew/blog_output_data"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return "", err
	}

	content := extractBlogHTMLContent(string(data))
	if content != "" {
		return content, nil
	}

	return "", fmt.Errorf("获取日志内容失败")
}

func extractBlogHTMLContent(page string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(page))
	if err != nil {
		return ""
	}

	content := doc.Find("#blogDetailDiv").Text()
	return cleanPlainText(content)
}

func pickFirstBlogValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func getNumericString(data map[string]interface{}, key string) string {
	if value := getStringValue(data, key); value != "" {
		return value
	}
	if value := getInt64Value(data, key); value > 0 {
		return strconv.FormatInt(value, 10)
	}
	return ""
}
