package qzone

import (
	"net/url"
	"time"
)

type VisitorItem struct {
	VisitorQQ   string
	VisitorName string
	Source      int
	IsHidden    bool
	YellowLevel int
	VisitTime   time.Time
}

func (c *Client) GetVisitors() ([]VisitorItem, error) {
	params := url.Values{
		"uin":     {c.qq},
		"mask":    {"7"},
		"g_tk":    {c.gtk},
		"page":    {"1"},
		"fupdate": {"1"},
		"clear":   {"1"},
	}

	apiURL := "https://h5.qzone.qq.com/proxy/domain/g.qzone.qq.com/cgi-bin/friendshow/cgi_get_visitor_more"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return nil, err
	}
	result, err := parseCallback(data)
	if err != nil {
		return nil, err
	}
	if err := checkResponseCode(result, "获取访客失败"); err != nil {
		return nil, err
	}

	var items []VisitorItem
	dataObj := getMapValue(result, "data")
	if dataObj == nil {
		return items, nil
	}
	for _, raw := range getSliceValue(dataObj, "items") {
		vMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		item := VisitorItem{
			VisitorQQ:   getNumericString(vMap, "uin"),
			VisitorName: cleanPlainText(getStringValue(vMap, "name")),
			Source:      getIntValue(vMap, "src"),
			IsHidden:    getBoolValue(vMap, "is_hide_visit"),
			YellowLevel: getIntValue(vMap, "yellow"),
		}
		if ts := getInt64Value(vMap, "time"); ts > 0 {
			item.VisitTime = time.Unix(ts, 0)
		}
		if item.VisitorQQ != "" {
			items = append(items, item)
		}
	}
	return items, nil
}
