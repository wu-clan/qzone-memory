package qzone

import (
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type FavoriteItem struct {
	FavoriteID string
	Type       int
	Title      string
	Abstract   string
	URL        string
	OwnerQQ    string
	OwnerName  string
	Images     []string
	CreateTime time.Time
}

func (c *Client) GetFavorites(offset, limit int) ([]FavoriteItem, error) {
	if limit <= 0 {
		limit = 20
	}
	params := url.Values{
		"g_tk":        {c.gtk},
		"callback":    {"shine0_Callback"},
		"t":           {strconv.FormatFloat(rand.Float64(), 'f', 9, 64)},
		"uin":         {c.qq},
		"ra":          {"0.1"},
		"start":       {strconv.Itoa(offset)},
		"num":         {strconv.Itoa(limit)},
		"inCharset":   {"utf-8"},
		"outCharset":  {"utf-8"},
		"refer":       {"qzone"},
		"source":      {"qzone"},
		"callbackFun": {"shine0"},
		"_":           {strconv.FormatInt(time.Now().UnixMilli(), 10)},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/fav.qzone.qq.com/cgi-bin/get_fav_list"
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

	var items []FavoriteItem
	for _, raw := range getSliceValue(dataObj, "fav_list") {
		fMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		favoriteID := strings.TrimSpace(getStringValue(fMap, "id"))
		if favoriteID == "" {
			favoriteID = getNumericString(fMap, "id")
		}
		item := FavoriteItem{
			FavoriteID: favoriteID,
			Type:       getIntValue(fMap, "type"),
			Title:      cleanPlainText(getStringValue(fMap, "title")),
			Abstract:   cleanPlainText(pickFirstNonEmpty(getStringValue(fMap, "abstract"), getStringValue(fMap, "desp"))),
			URL:        getStringValue(fMap, "url"),
			OwnerQQ:    getNumericString(fMap, "owner_uin"),
			OwnerName:  cleanPlainText(getStringValue(fMap, "owner_name")),
		}
		if ts := getInt64Value(fMap, "create_time"); ts > 0 {
			item.CreateTime = time.Unix(ts, 0)
		} else if ts := getInt64Value(fMap, "createTime"); ts > 0 {
			item.CreateTime = time.Unix(ts, 0)
		}
		if pics, ok := fMap["origin_img_list"].([]interface{}); ok {
			for _, p := range pics {
				if s, ok := p.(string); ok && s != "" {
					item.Images = append(item.Images, s)
				}
			}
		}
		if len(item.Images) == 0 {
			if pics, ok := fMap["img_list"].([]interface{}); ok {
				for _, p := range pics {
					if s, ok := p.(string); ok && s != "" {
						item.Images = append(item.Images, s)
					}
				}
			}
		}
		if len(item.Images) == 0 {
			if pics, ok := fMap["images"].([]interface{}); ok {
				for _, p := range pics {
					if s, ok := p.(string); ok && s != "" {
						item.Images = append(item.Images, s)
					}
				}
			}
		}
		if photoItems := getSliceValue(fMap, "photo_list"); len(photoItems) > 0 {
			if photoMap, ok := photoItems[0].(map[string]interface{}); ok {
				if item.URL == "" {
					item.URL = getStringValue(photoMap, "url")
				}
				if item.Title == "" {
					item.Title = cleanPlainText(getStringValue(photoMap, "title"))
				}
				if item.Abstract == "" {
					item.Abstract = cleanPlainText(getStringValue(photoMap, "description"))
				}
				if item.OwnerQQ == "" {
					item.OwnerQQ = getNumericString(photoMap, "owner_uin")
				}
				if item.CreateTime.IsZero() {
					if ts := getInt64Value(photoMap, "create_time"); ts > 0 {
						item.CreateTime = time.Unix(ts, 0)
					}
				}
			}
		}
		if item.FavoriteID != "" {
			items = append(items, item)
		}
	}
	return items, nil
}
