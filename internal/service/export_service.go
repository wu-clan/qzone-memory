package service

import (
	"archive/zip"
	"context"
	"encoding/json"
	"html/template"
	"io"
	"os"
	"path"
	"time"

	"github.com/qzone-memory/internal/media"
)

type exportItem struct {
	TypeLabel string
	Author    string
	Time      string
	Title     string
	Content   string
	Images    []string
	IsDeleted bool
}

type exportYear struct {
	Year  int
	Items []exportItem
}

type exportData struct {
	QQ        string
	Nickname  string
	Total     int
	Generated string
	Years     []exportYear
}

var exportTypeLabels = map[string]string{
	"activity": "动态", "talk": "说说", "blog": "日志", "album": "相册",
	"message": "留言", "comment": "评论", "visitor": "访客", "video": "视频",
	"like": "点赞", "favorite": "收藏", "diary": "日记", "mention": "提及", "share": "转发",
}

func exportJSONURLs(raw string) []string {
	if raw == "" {
		return nil
	}
	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return nil
	}
	return arr
}

// ExportArchive 把某账号的全部回忆渲染成离线可开的静态 HTML，连同本地媒体一起打包为 zip 写入 w。
// 未本地化的图片回退为远程 URL（在线可见）；已本地化的图片随包带走，可彻底离线浏览。
func ExportArchive(ctx context.Context, qq, nickname string, w io.Writer) error {
	if err := validateQQ(qq); err != nil {
		return err
	}
	items, err := buildMemoryTimeline(ctx, qq, "all")
	if err != nil {
		return err
	}

	zw := zip.NewWriter(w)
	defer zw.Close()

	mediaMap := make(map[string]string) // 原始 URL -> zip 内相对路径（或回退的远程 URL）
	addImage := func(rawURL string) string {
		if rawURL == "" {
			return ""
		}
		if rel, ok := mediaMap[rawURL]; ok {
			return rel
		}
		abs, ok := media.Resolve(ctx, qq, rawURL)
		if !ok {
			mediaMap[rawURL] = rawURL // 未本地化，回退远程
			return rawURL
		}
		rel := "media/" + path.Base(abs)
		if err := addFileToZip(zw, abs, rel); err != nil {
			mediaMap[rawURL] = rawURL
			return rawURL
		}
		mediaMap[rawURL] = rel
		return rel
	}

	yearIndex := make(map[int]int)
	var years []exportYear
	for _, item := range items {
		urls := exportJSONURLs(item.Images)
		if (item.Type == "album" || item.Type == "video") && item.Cover != "" {
			urls = append(urls, item.Cover)
		}
		imgs := make([]string, 0, len(urls))
		for _, u := range urls {
			if r := addImage(u); r != "" {
				imgs = append(imgs, r)
			}
		}

		author := item.AuthorName
		if item.AuthorQQ == "" || item.AuthorQQ == qq {
			author = firstNonEmpty(item.AuthorName, nickname, "我")
		}
		author = firstNonEmpty(author, item.AuthorQQ, "好友")

		timeStr := ""
		year := 0
		if !item.PublishTime.IsZero() {
			timeStr = item.PublishTime.Format("2006-01-02 15:04")
			year = item.PublishTime.Year()
		}

		title := item.Title
		if title == author {
			title = ""
		}

		ei := exportItem{
			TypeLabel: firstNonEmpty(exportTypeLabels[item.Type], "动态"),
			Author:    author,
			Time:      timeStr,
			Title:     title,
			Content:   item.Content,
			Images:    imgs,
			IsDeleted: item.IsDeleted,
		}

		idx, ok := yearIndex[year]
		if !ok {
			years = append(years, exportYear{Year: year})
			idx = len(years) - 1
			yearIndex[year] = idx
		}
		years[idx].Items = append(years[idx].Items, ei)
	}

	data := exportData{
		QQ:        qq,
		Nickname:  nickname,
		Total:     len(items),
		Generated: time.Now().Format("2006-01-02 15:04"),
		Years:     years,
	}

	tmpl, err := template.New("archive").Parse(exportHTMLTemplate)
	if err != nil {
		return err
	}
	indexWriter, err := zw.Create("index.html")
	if err != nil {
		return err
	}
	if err := tmpl.Execute(indexWriter, data); err != nil {
		return err
	}

	if rw, err := zw.Create("README.txt"); err == nil {
		_, _ = io.WriteString(rw, exportReadme)
	}
	return nil
}

func addFileToZip(zw *zip.Writer, absPath, zipPath string) error {
	f, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer f.Close()
	dst, err := zw.Create(zipPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, f)
	return err
}

const exportReadme = `这是由「时光档案馆」导出的 QQ 空间回忆离线副本。

用浏览器打开 index.html 即可浏览，无需联网、无需原程序。
图片保存在 media/ 目录中，请将本文件夹完整保留。
`

const exportHTMLTemplate = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>时光档案馆 · {{.QQ}}</title>
<style>
*{box-sizing:border-box}
body{margin:0;font-family:-apple-system,"PingFang SC","Microsoft YaHei",sans-serif;background:#f4efe6;color:#2b2b2b;line-height:1.6}
header{padding:32px 20px;text-align:center;background:#2f5fae;color:#f4efe6}
header h1{margin:0 0 8px;font-size:26px;letter-spacing:4px}
header p{margin:3px 0;opacity:.9;font-size:13px}
.note{font-size:12px;opacity:.8}
section{max-width:720px;margin:24px auto;padding:0 16px}
.year h2{font-size:20px;color:#2f5fae;border-bottom:1px solid #d8cfbe;padding-bottom:6px}
.item{background:#fff;border:1px solid #e7ddc9;border-radius:10px;padding:14px 16px;margin:12px 0;box-shadow:0 1px 3px rgba(0,0,0,.04)}
.item.deleted{opacity:.7}
.meta{font-size:12px;color:#8a7f6a;display:flex;gap:10px;flex-wrap:wrap;align-items:center;margin-bottom:6px}
.tag{background:#eef2fb;color:#2f5fae;border-radius:4px;padding:1px 8px}
.title{font-weight:600;margin:4px 0}
.content{white-space:pre-wrap;word-break:break-word}
.imgs{display:flex;flex-wrap:wrap;gap:6px;margin-top:10px}
.imgs img{max-width:160px;max-height:160px;border-radius:6px;object-fit:cover}
footer{text-align:center;padding:30px;color:#8a7f6a;font-size:12px}
</style>
</head>
<body>
<header>
<h1>时光档案馆</h1>
<p>{{.Nickname}} · QQ {{.QQ}}</p>
<p>共 {{.Total}} 条回忆 · 导出于 {{.Generated}}</p>
<p class="note">本档案为离线副本，可脱离原程序与 QQ 永久保存与浏览。</p>
</header>
{{range .Years}}
<section class="year">
<h2>{{if .Year}}{{.Year}} 年{{else}}更早{{end}}</h2>
{{range .Items}}
<article class="item{{if .IsDeleted}} deleted{{end}}">
<div class="meta"><span class="tag">{{.TypeLabel}}</span><span class="author">{{.Author}}</span><span class="time">{{.Time}}</span>{{if .IsDeleted}}<span>（已删除）</span>{{end}}</div>
{{if .Title}}<div class="title">{{.Title}}</div>{{end}}
{{if .Content}}<div class="content">{{.Content}}</div>{{end}}
{{if .Images}}<div class="imgs">{{range .Images}}<img loading="lazy" src="{{.}}">{{end}}</div>{{end}}
</article>
{{end}}
</section>
{{end}}
<footer>由 时光档案馆 导出 · {{.Generated}}</footer>
</body>
</html>
`
