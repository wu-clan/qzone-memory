package service

import (
	"context"

	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/model"
)

func GetTalkList(ctx context.Context, req dto.QueryByQQRequest) (*dto.PageResponse[*model.Talk], error) {
	if req.QQ == "" {
		return nil, common.ErrInvalidParam
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	talks, total, err := dao.ListTalks(ctx, req.QQ, offset, pageSize)
	if err != nil {
		return nil, err
	}

	return dto.NewPageResponse(talks, total, page, pageSize), nil
}

func GetTalkDetail(ctx context.Context, req dto.QueryByTalkIDRequest) (*model.Talk, error) {
	if req.TalkID == "" {
		return nil, common.ErrInvalidParam
	}
	return dao.GetTalkByTalkID(ctx, req.TalkID)
}
