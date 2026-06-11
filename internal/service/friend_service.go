package service

import (
	"context"

	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/model"
)

type FriendPageResponse struct {
	List            []*model.Friend      `json:"list"`
	Groups          []*model.FriendGroup `json:"groups"`
	Total           int64                `json:"total"`
	CurrentTotal    int64                `json:"current_total"`
	HistoricalTotal int64                `json:"historical_total"`
	GroupTotal      int                  `json:"group_total"`
	Page            int                  `json:"page"`
	PageSize        int                  `json:"page_size"`
}

func GetFriendList(ctx context.Context, req dto.QueryFriendsRequest) (*FriendPageResponse, error) {
	if req.QQ == "" {
		return nil, common.ErrInvalidParam
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	friends, total, err := dao.ListFriends(ctx, req.QQ, req.IncludeDeleted, offset, pageSize)
	if err != nil {
		return nil, err
	}
	groups, err := dao.ListFriendGroups(ctx, req.QQ)
	if err != nil {
		return nil, err
	}
	currentTotal, historicalTotal, err := dao.CountFriendsByStatus(ctx, req.QQ)
	if err != nil {
		return nil, err
	}

	groupTotal := 0
	for _, group := range groups {
		if group != nil && !group.IsDeleted {
			groupTotal++
		}
	}

	return &FriendPageResponse{
		List:            friends,
		Groups:          groups,
		Total:           total,
		CurrentTotal:    currentTotal,
		HistoricalTotal: historicalTotal,
		GroupTotal:      groupTotal,
		Page:            page,
		PageSize:        pageSize,
	}, nil
}
