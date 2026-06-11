package service

import (
	"context"

	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/model"
)

func GetMessageList(ctx context.Context, req dto.QueryByQQRequest) (*dto.PageResponse[*model.Message], error) {
	if req.QQ == "" {
		return nil, common.ErrInvalidParam
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	messages, total, err := dao.ListMessages(ctx, req.QQ, offset, pageSize)
	if err != nil {
		return nil, err
	}

	return dto.NewPageResponse(messages, total, page, pageSize), nil
}
