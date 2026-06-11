package service

import (
	"context"

	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/model"
)

func ListCommentsByTarget(ctx context.Context, req dto.QueryByTargetRequest) (*dto.PageResponse[*model.Comment], error) {
	if req.TargetType == "" || req.TargetID == "" {
		return nil, common.ErrInvalidParam
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	comments, total, err := dao.ListCommentsByTarget(ctx, req.TargetType, req.TargetID, offset, pageSize)
	if err != nil {
		return nil, err
	}

	return dto.NewPageResponse(comments, total, page, pageSize), nil
}
