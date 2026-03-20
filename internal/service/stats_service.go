package service

import (
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/pkg/response"
)

type MemoryStats struct {
	Total  int64            `json:"total"`
	ByType map[string]int64 `json:"by_type"`
	ByYear []YearCount      `json:"by_year"`
}

type YearCount struct {
	Year  int   `json:"year"`
	Count int64 `json:"count"`
}

func GetMemoryStats(c *gin.Context) (*MemoryStats, *response.AppError) {
	var req dto.QueryByQQRequest
	if err := bindQuery(c, &req); err != nil {
		return nil, err
	}
	items, err := buildMemoryTimeline(req.QQ, "all")
	if err != nil {
		return nil, &response.AppError{Code: 500, Err: err}
	}

	stats := &MemoryStats{
		ByType: make(map[string]int64),
	}
	yearMap := make(map[int]int64)
	for _, item := range items {
		stats.Total++
		stats.ByType[item.Type]++
		if item.PublishTime.IsZero() {
			continue
		}
		yearMap[item.PublishTime.Year()]++
	}
	for year, count := range yearMap {
		stats.ByYear = append(stats.ByYear, YearCount{
			Year:  year,
			Count: count,
		})
	}
	sort.Slice(stats.ByYear, func(i, j int) bool {
		return stats.ByYear[i].Year > stats.ByYear[j].Year
	})
	return stats, nil
}
