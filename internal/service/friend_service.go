package service

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/model"
	"github.com/qzone-memory/pkg/response"
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

func GetFriendList(c *gin.Context) (*FriendPageResponse, *response.AppError) {
	var req dto.QueryFriendsRequest
	if err := bindQuery(c, &req); err != nil {
		return nil, err
	}
	if req.QQ == "" {
		return nil, &response.AppError{Code: http.StatusBadRequest, Err: common.ErrInvalidParam}
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	friends, total, err := dao.ListFriends(req.QQ, req.IncludeDeleted, offset, pageSize)
	if err != nil {
		return nil, &response.AppError{Code: http.StatusInternalServerError, Err: err}
	}
	groups, err := dao.ListFriendGroups(req.QQ)
	if err != nil {
		return nil, &response.AppError{Code: http.StatusInternalServerError, Err: err}
	}
	currentTotal, historicalTotal, err := dao.CountFriendsByStatus(req.QQ)
	if err != nil {
		return nil, &response.AppError{Code: http.StatusInternalServerError, Err: err}
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
