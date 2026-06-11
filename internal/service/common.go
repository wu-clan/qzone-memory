package service

import (
	"regexp"

	"github.com/qzone-memory/internal/common"
)

var qqRegexp = regexp.MustCompile(`^\d{5,20}$`)

// normalizePage 标准化分页参数
func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = common.DefaultPage
	}
	if pageSize <= 0 {
		pageSize = common.DefaultPageSize
	}
	if pageSize > common.MaxPageSize {
		pageSize = common.MaxPageSize
	}
	return page, pageSize
}

func validateQQ(qq string) error {
	if !qqRegexp.MatchString(qq) {
		return common.ErrInvalidQQ
	}
	return nil
}
