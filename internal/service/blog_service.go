package service

import (
	"context"

	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/model"
)

func GetBlogList(ctx context.Context, req dto.QueryByQQRequest) (*dto.PageResponse[*model.Blog], error) {
	if req.QQ == "" {
		return nil, common.ErrInvalidParam
	}

	page, pageSize := normalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	blogs, total, err := dao.ListBlogs(ctx, req.QQ, offset, pageSize)
	if err != nil {
		return nil, err
	}

	return dto.NewPageResponse(blogs, total, page, pageSize), nil
}

func GetBlogDetail(ctx context.Context, req dto.QueryByBlogIDRequest) (*model.Blog, error) {
	if req.BlogID == "" {
		return nil, common.ErrInvalidParam
	}
	return dao.GetBlogByBlogID(ctx, req.BlogID)
}
