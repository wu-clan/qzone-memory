package common

import "errors"

var (
	ErrInvalidParam = errors.New("请求参数错误")
	ErrInvalidQQ    = errors.New("QQ号格式错误")
	ErrNotFound     = errors.New("资源不存在")
	ErrUnauthorized = errors.New("授权失败，请重新登录")
	ErrSyncRunning  = errors.New("同步任务正在进行中")
)
