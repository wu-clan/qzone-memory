package qzone

import (
	"net/url"
	"strconv"
	"time"
)

// AlbumItem 相册数据项
type AlbumItem struct {
	AlbumID     string
	Name        string
	Description string
	CoverURL    string
	PhotoCount  int
	CreateTime  time.Time
}

// PhotoItem 照片数据项
type PhotoItem struct {
	PhotoID     string
	AlbumID     string
	Name        string
	Description string
	URL         string
	ThumbURL    string
	Width       int
	Height      int
	PhotoTime   time.Time
}

// GetAlbums 获取相册列表
func (c *Client) GetAlbums() ([]AlbumItem, error) {
	params := url.Values{
		"uin":          {c.qq},
		"hostUin":      {c.qq},
		"appid":        {"4"},
		"mode":         {"2"},
		"idcNum":       {"4"},
		"g_tk":         {c.gtk},
		"format":       {"json"},
		"inCharset":    {"utf-8"},
		"outCharset":   {"utf-8"},
		"source":       {"qzone"},
		"plat":         {"qzone"},
		"notice":       {"0"},
		"filter":       {"1"},
		"handset":      {"4"},
		"needUserInfo": {"1"},
		"pageStart":    {"0"},
		"pageNum":      {"100"},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/photo.qzone.qq.com/fcgi-bin/fcg_list_album_v3"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return nil, err
	}

	result, err := parseCallback(data)
	if err != nil {
		return nil, err
	}

	if err := checkResponseCode(result, "获取相册失败"); err != nil {
		return nil, err
	}

	var albums []AlbumItem
	dataObj := getMapValue(result, "data")
	if dataObj == nil {
		return albums, nil
	}

	albumList := getSliceValue(dataObj, "albumListModeSort")
	if len(albumList) == 0 {
		albumList = getSliceValue(dataObj, "albumList")
	}
	for _, item := range albumList {
		albumMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		album := AlbumItem{
			AlbumID:     getStringValue(albumMap, "id"),
			Name:        cleanPlainText(getStringValue(albumMap, "name")),
			Description: cleanPlainText(getStringValue(albumMap, "desc")),
			CoverURL:    getStringValue(albumMap, "pre"),
			PhotoCount:  getIntValue(albumMap, "total"),
		}
		if looksLikeSystemAlbum(album.Name, getStringValue(albumMap, "desc")) {
			continue
		}

		if createTime := getInt64Value(albumMap, "createtime"); createTime > 0 {
			album.CreateTime = time.Unix(createTime, 0)
		}

		albums = append(albums, album)
	}

	return albums, nil
}

// GetPhotos 获取相册中的照片列表
func (c *Client) GetPhotos(albumID string, offset, limit int) ([]PhotoItem, error) {
	params := url.Values{
		"uin":        {c.qq},
		"hostUin":    {c.qq},
		"topicId":    {albumID},
		"appid":      {"4"},
		"pageStart":  {strconv.Itoa(offset)},
		"pageNum":    {strconv.Itoa(limit)},
		"mode":       {"0"},
		"idcNum":     {"4"},
		"g_tk":       {c.gtk},
		"format":     {"json"},
		"inCharset":  {"utf-8"},
		"outCharset": {"utf-8"},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/photo.qzone.qq.com/fcgi-bin/cgi_list_photo"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return nil, err
	}

	result, err := parseCallback(data)
	if err != nil {
		return nil, err
	}

	if err := checkResponseCode(result, "获取照片失败"); err != nil {
		return nil, err
	}

	var photos []PhotoItem
	dataObj := getMapValue(result, "data")
	if dataObj == nil {
		return photos, nil
	}

	photoList := getSliceValue(dataObj, "photoList")
	for _, item := range photoList {
		photoMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		photo := PhotoItem{
			PhotoID:     getStringValue(photoMap, "lloc"),
			AlbumID:     albumID,
			Name:        cleanPlainText(getStringValue(photoMap, "name")),
			Description: cleanPlainText(getStringValue(photoMap, "desc")),
			URL:         getStringValue(photoMap, "url"),
			ThumbURL:    getStringValue(photoMap, "pre"),
			Width:       getIntValue(photoMap, "width"),
			Height:      getIntValue(photoMap, "height"),
		}

		if uploadTime := getInt64Value(photoMap, "uploadtime"); uploadTime > 0 {
			photo.PhotoTime = time.Unix(uploadTime, 0)
		}

		photos = append(photos, photo)
	}

	return photos, nil
}
