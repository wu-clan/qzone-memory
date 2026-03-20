package common

import "errors"

var (
	ErrInvalidParam = errors.New("请求参数错误")
	ErrNotFound     = errors.New("资源不存在")
)
