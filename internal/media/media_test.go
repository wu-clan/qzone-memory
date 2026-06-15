package media

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qzone-memory/config"
	"github.com/qzone-memory/internal/model"
)

func TestShouldDownload(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://q.qlogo.cn/headimg_dl?dst_uin=123&spec=100", true},  // 头像保留
		{"http://m.qpic.cn/psc?/abc/photo.jpg", true},                 // 真实照片
		{"https://qzonestyle.gtimg.cn/qzone/space_item/x.png", false}, // 装饰素材（非腾讯放行域名）
		{"https://user.qzone.qq.com/qzone/em/e123.gif", false},        // 表情
		{"https://example.com/a.jpg", false},                          // 非腾讯资源
		{"", false},
	}
	for _, c := range cases {
		if got := shouldDownload(c.url); got != c.want {
			t.Errorf("shouldDownload(%q)=%v, want %v", c.url, got, c.want)
		}
	}
}

func TestHashAndExt(t *testing.T) {
	a := hashURL("https://x/y.jpg")
	b := hashURL("  https://x/y.jpg  ") // 归一化去空白后应一致
	if a != b {
		t.Fatalf("归一化后哈希应一致：%s vs %s", a, b)
	}
	if len(a) != 64 {
		t.Fatalf("sha256 十六进制应为 64 位，得到 %d", len(a))
	}
	if ext := extFromURL("https://x/y/photo.JPG?a=1"); ext != ".jpg" {
		t.Fatalf("extFromURL=%q", ext)
	}
	if ext := extFromContentType("image/png; charset=binary"); ext != ".png" {
		t.Fatalf("extFromContentType=%q", ext)
	}
}

func TestAbsPathRejectsTraversal(t *testing.T) {
	config.GlobalConfig = &config.Config{Media: config.MediaConfig{Dir: t.TempDir()}}
	if _, ok := absPath(relPathFor("123", "photo", "abcd1234", ".jpg")); !ok {
		t.Fatal("正常相对路径应被接受")
	}
	if _, ok := absPath("../../etc/passwd"); ok {
		t.Fatal("路径穿越应被拒绝")
	}
}

func TestFetchAndStore(t *testing.T) {
	dir := t.TempDir()
	config.GlobalConfig = &config.Config{Media: config.MediaConfig{Dir: dir}}

	payload := []byte("\x89PNG\r\n\x1a\nfake-image-bytes")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(payload)
	}))
	defer ts.Close()

	asset := &model.MediaAsset{
		UserQQ:    "123",
		Category:  "photo",
		URLHash:   hashURL(ts.URL),
		SourceURL: ts.URL,
	}

	rel, size, mimeType, err := fetchAndStore(context.Background(), asset)
	if err != nil {
		t.Fatalf("fetchAndStore: %v", err)
	}
	if size != int64(len(payload)) {
		t.Fatalf("size=%d want %d", size, len(payload))
	}
	if mimeType != "image/png" {
		t.Fatalf("mime=%q", mimeType)
	}
	if !fileExists(rel) {
		t.Fatalf("文件未落盘：%s", rel)
	}
}
