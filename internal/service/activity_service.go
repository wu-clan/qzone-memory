package service

import (
	"context"

	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/model"
)

func GetActivityList(ctx context.Context, req dto.QueryActivityRequest) (*dto.PageResponse[*model.Activity], error) {
	if req.QQ == "" {
		return nil, common.ErrInvalidParam
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	activities, total, err := dao.ListActivities(ctx, req.QQ, req.Type, offset, pageSize)
	if err != nil {
		return nil, err
	}

	return dto.NewPageResponse(activities, total, page, pageSize), nil
}

func GetActivityDetail(ctx context.Context, req dto.QueryByFeedIDRequest) (*model.Activity, error) {
	if req.FeedID == "" {
		return nil, common.ErrInvalidParam
	}
	return dao.GetActivityByFeedID(ctx, req.FeedID)
}
