package service

import (
	"context"
	"sort"
	"time"

	"github.com/qzone-memory/internal/dto"
)

// MemoryReport 「青春纪念卡」——对一个账号全部回忆的聚合概览。
type MemoryReport struct {
	QQ                  string           `json:"qq"`
	Total               int64            `json:"total"`
	FirstTime           time.Time        `json:"first_time"`
	LastTime            time.Time        `json:"last_time"`
	SpanYears           int              `json:"span_years"`
	ByType              map[string]int64 `json:"by_type"`
	ByYear              []YearCount      `json:"by_year"`
	MostActiveYear      int              `json:"most_active_year"`
	MostActiveYearCount int64            `json:"most_active_year_count"`
	TopPeople           []PersonCount    `json:"top_people"`
	FirstTalk           *MemoryItem      `json:"first_talk,omitempty"`
}

// PersonCount 互动最多的人。
type PersonCount struct {
	QQ    string `json:"qq"`
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// interactionTypes 计入「互动最多的人」的条目类型。
var interactionTypes = map[string]bool{
	"comment": true, "like": true, "message": true,
	"visitor": true, "share": true, "mention": true,
}

// GetMemoryReport 聚合某账号的回忆概览，用于生成纪念卡。
func GetMemoryReport(ctx context.Context, req dto.QueryByQQRequest) (*MemoryReport, error) {
	items, err := buildMemoryTimeline(ctx, req.QQ, "all")
	if err != nil {
		return nil, err
	}

	report := &MemoryReport{QQ: req.QQ, ByType: make(map[string]int64)}
	yearMap := make(map[int]int64)
	peopleCount := make(map[string]int64)
	peopleName := make(map[string]string)
	var oldestTalk *MemoryItem

	for _, item := range items {
		report.Total++
		report.ByType[item.Type]++

		if !item.PublishTime.IsZero() {
			yearMap[item.PublishTime.Year()]++
			if report.FirstTime.IsZero() || item.PublishTime.Before(report.FirstTime) {
				report.FirstTime = item.PublishTime
			}
			if item.PublishTime.After(report.LastTime) {
				report.LastTime = item.PublishTime
			}
		}

		if interactionTypes[item.Type] && item.AuthorQQ != "" && item.AuthorQQ != req.QQ {
			peopleCount[item.AuthorQQ]++
			if item.AuthorName != "" {
				peopleName[item.AuthorQQ] = item.AuthorName
			}
		}

		if item.Type == "talk" && !item.PublishTime.IsZero() {
			if oldestTalk == nil || item.PublishTime.Before(oldestTalk.PublishTime) {
				oldestTalk = item
			}
		}
	}

	for year, count := range yearMap {
		report.ByYear = append(report.ByYear, YearCount{Year: year, Count: count})
		if count > report.MostActiveYearCount {
			report.MostActiveYearCount = count
			report.MostActiveYear = year
		}
	}
	sort.Slice(report.ByYear, func(i, j int) bool { return report.ByYear[i].Year > report.ByYear[j].Year })

	if !report.FirstTime.IsZero() {
		report.SpanYears = report.LastTime.Year() - report.FirstTime.Year() + 1
	}

	for qq, count := range peopleCount {
		report.TopPeople = append(report.TopPeople, PersonCount{
			QQ:    qq,
			Name:  firstNonEmpty(peopleName[qq], qq),
			Count: count,
		})
	}
	sort.Slice(report.TopPeople, func(i, j int) bool { return report.TopPeople[i].Count > report.TopPeople[j].Count })
	if len(report.TopPeople) > 5 {
		report.TopPeople = report.TopPeople[:5]
	}

	report.FirstTalk = oldestTalk
	return report, nil
}
