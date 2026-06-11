package qzone

import (
	"net/url"
	"strconv"
	"time"
)

type FriendItem struct {
	FriendQQ  string
	Name      string
	Remark    string
	GroupID   int
	GroupName string
	Online    int
	Yellow    int
}

type FriendGroupItem struct {
	GroupID int
	Name    string
}

type SpecialCareItem struct {
	FriendQQ string
	Name     string
}

func (c *Client) GetFriends() ([]FriendItem, []FriendGroupItem, error) {
	params := url.Values{
		"uin":            {c.qq},
		"follow_flag":    {"0"},
		"groupface_flag": {"0"},
		"fupdate":        {"1"},
		"g_tk":           {c.gtk},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/r.qzone.qq.com/cgi-bin/tfriend/friend_show_qqfriends.cgi"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return nil, nil, err
	}

	result, err := parseCallback(data)
	if err != nil {
		return nil, nil, err
	}
	if err := checkResponseCode(result, "获取好友失败"); err != nil {
		return nil, nil, err
	}

	dataObj := getMapValue(result, "data")
	if dataObj == nil {
		dataObj = result
	}

	groupMap := make(map[int]string)
	groups := make([]FriendGroupItem, 0)
	for _, item := range getSliceValue(dataObj, "gpnames") {
		groupData, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		groupID := getIntValue(groupData, "gpid")
		if groupID == 0 {
			groupID = getIntValue(groupData, "groupid")
		}
		groupName := cleanPlainText(getStringValue(groupData, "gpname"))
		groupMap[groupID] = groupName
		groups = append(groups, FriendGroupItem{
			GroupID: groupID,
			Name:    groupName,
		})
	}

	friends := make([]FriendItem, 0)
	for _, item := range getSliceValue(dataObj, "items") {
		friendData, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		friendQQ := getNumericString(friendData, "uin")
		if friendQQ == "" {
			continue
		}
		groupID := getIntValue(friendData, "groupid")
		friends = append(friends, FriendItem{
			FriendQQ:  friendQQ,
			Name:      cleanPlainText(getStringValue(friendData, "name")),
			Remark:    cleanPlainText(getStringValue(friendData, "remark")),
			GroupID:   groupID,
			GroupName: groupMap[groupID],
			Online:    getIntValue(friendData, "online"),
			Yellow:    getIntValue(friendData, "yellow"),
		})
	}
	return friends, groups, nil
}

func (c *Client) GetFriendship(friendQQ string) (time.Time, int, error) {
	params := url.Values{
		"activeuin":  {c.qq},
		"passiveuin": {friendQQ},
		"situation":  {"1"},
		"isCalendar": {"1"},
		"g_tk":       {c.gtk},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/r.qzone.qq.com/cgi-bin/friendship/cgi_friendship"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return time.Time{}, 0, err
	}

	result, err := parseCallback(data)
	if err != nil {
		return time.Time{}, 0, err
	}
	if err := checkResponseCode(result, "获取好友关系失败"); err != nil {
		return time.Time{}, 0, err
	}

	dataObj := getMapValue(result, "data")
	if dataObj == nil {
		return time.Time{}, 0, nil
	}

	addTime := time.Time{}
	if ts := getInt64Value(dataObj, "addFriendTime"); ts > 0 {
		addTime = time.Unix(ts, 0)
	}
	return addTime, getIntValue(dataObj, "common_group_num"), nil
}

func (c *Client) GetSpecialCareFriends() ([]SpecialCareItem, error) {
	params := url.Values{
		"uin":     {c.qq},
		"do":      {"3"},
		"fupdate": {"1"},
		"rd":      {strconv.FormatFloat(float64(time.Now().UnixNano()%1000000)/1000000, 'f', 6, 64)},
		"g_tk":    {c.gtk},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/r.qzone.qq.com/cgi-bin/tfriend/specialcare_get.cgi"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return nil, err
	}

	result, err := parseCallback(data)
	if err != nil {
		return nil, err
	}
	if err := checkResponseCode(result, "获取特别关心失败"); err != nil {
		return nil, err
	}

	dataObj := getMapValue(result, "data")
	if dataObj == nil {
		dataObj = result
	}

	items := make([]SpecialCareItem, 0)
	for _, item := range getSliceValue(dataObj, "items_special") {
		friendData, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		friendQQ := getNumericString(friendData, "uin")
		if friendQQ == "" {
			continue
		}
		items = append(items, SpecialCareItem{
			FriendQQ: friendQQ,
			Name:     cleanPlainText(getStringValue(friendData, "name")),
		})
	}
	if len(items) > 0 {
		return items, nil
	}

	for _, item := range getSliceValue(dataObj, "items") {
		friendData, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		friendQQ := getNumericString(friendData, "uin")
		if friendQQ == "" {
			continue
		}
		items = append(items, SpecialCareItem{
			FriendQQ: friendQQ,
			Name:     cleanPlainText(getStringValue(friendData, "name")),
		})
	}
	return items, nil
}
