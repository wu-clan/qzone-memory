package service

import (
	"context"
	"sort"

	"github.com/qzone-memory/internal/dto"
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

func GetMemoryStats(ctx context.Context, req dto.QueryByQQRequest) (*MemoryStats, error) {
	items, err := buildMemoryTimeline(ctx, req.QQ, "content")
	if err != nil {
		return nil, err
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
