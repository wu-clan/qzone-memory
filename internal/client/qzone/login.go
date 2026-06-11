package qzone

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// LoginClient 登录客户端（独立 HTTP 会话，与业务 Client 分离）
type LoginClient struct {
	httpClient *http.Client
	qrSig      string
}

// LoginStatus 登录状态
type LoginStatus struct {
	Status   int    `json:"status"` // 0=等待扫描 1=已扫描待确认 2=登录成功 3=二维码过期 4=已取消
	Message  string `json:"message"`
	Nickname string `json:"nickname"`
	QQ       string `json:"qq,omitempty"` // 登录成功后返回 QQ 号
	LoginURL string `json:"-"`            // 仅内部使用，不暴露给前端
}

// LoginResult 登录成功结果
type LoginResult struct {
	QQ       string
	Nickname string
	Cookie   string
	PSKey    string
	GTK      string
}

// NewLoginClient 创建登录客户端
func NewLoginClient() *LoginClient {
	jar, _ := cookiejar.New(nil)
	return &LoginClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // 不自动跟随重定向
			},
		},
	}
}

// GetQRCode 获取登录二维码 PNG 图片
func (lc *LoginClient) GetQRCode() ([]byte, error) {
	params := url.Values{
		"appid": {"549000912"},
		"e":     {"2"},
		"l":     {"M"},
		"s":     {"3"},
		"d":     {"72"},
		"v":     {"4"},
		"t":     {fmt.Sprintf("%.17f", float64(time.Now().UnixMilli())/1000.0)},
		"daid":  {"5"},
	}

	qrcodeURL := "https://ssl.ptlogin2.qq.com/ptqrshow?" + params.Encode()

	req, err := http.NewRequest("GET", qrcodeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := lc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取二维码失败: %w", err)
	}
	defer resp.Body.Close()

	// 提取 qrsig cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "qrsig" {
			lc.qrSig = cookie.Value
			break
		}
	}

	png, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取二维码图片失败: %w", err)
	}

	return png, nil
}

// ptqrtoken 根据 qrsig 计算 ptqrtoken
func (lc *LoginClient) ptqrtoken() string {
	hash := 0
	for _, c := range lc.qrSig {
		hash += (hash << 5) + int(c)
	}
	return fmt.Sprintf("%d", hash&0x7fffffff)
}

// PollStatus 轮询二维码扫描状态
func (lc *LoginClient) PollStatus() (*LoginStatus, error) {
	if lc.qrSig == "" {
		return nil, fmt.Errorf("请先获取二维码")
	}

	params := url.Values{
		"u1":          {"https://qzs.qq.com/qzone/v5/loginsucc.html?para=izone"},
		"ptqrtoken":   {lc.ptqrtoken()},
		"ptredirect":  {"0"},
		"h":           {"1"},
		"t":           {"1"},
		"g":           {"1"},
		"from_ui":     {"1"},
		"ptlang":      {"2052"},
		"action":      {fmt.Sprintf("0-0-%d", time.Now().UnixMilli())},
		"js_ver":      {"22112817"},
		"js_type":     {"1"},
		"login_sig":   {""},
		"pt_uistyle":  {"40"},
		"aid":         {"549000912"},
		"daid":        {"5"},
		"has_resolve": {"1"},
	}

	pollURL := "https://ssl.ptlogin2.qq.com/ptqrlogin?" + params.Encode()

	req, err := http.NewRequest("GET", pollURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", "qrsig="+lc.qrSig)
	req.Header.Set("Referer", "https://xui.ptlogin2.qq.com/")

	resp, err := lc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	content := string(body)
	return lc.parseLoginResponse(content), nil
}

// parseLoginResponse 解析登录响应
func (lc *LoginClient) parseLoginResponse(content string) *LoginStatus {
	// QQ 返回的 ptuiCB 参数之间可能带空格，直接提取所有单引号字段更稳妥。
	re := regexp.MustCompile(`'([^']*)'`)
	matches := re.FindAllStringSubmatch(content, -1)
	if len(matches) < 6 {
		return &LoginStatus{Status: -1, Message: "解析响应失败"}
	}

	status := &LoginStatus{
		Message:  matches[4][1],
		Nickname: matches[5][1],
		LoginURL: matches[2][1],
	}

	switch matches[0][1] {
	case "0":
		status.Status = 2 // 登录成功
	case "65":
		status.Status = 3 // 二维码过期
	case "66":
		status.Status = 0 // 等待扫描
	case "67":
		status.Status = 1 // 已扫描待确认
	case "68":
		status.Status = 4 // 已取消
	default:
		status.Status = -1
	}

	return status
}

// DoLogin 完成登录流程，获取 cookie
func (lc *LoginClient) DoLogin(loginURL string) (*LoginResult, error) {
	if loginURL == "" {
		return nil, fmt.Errorf("登录 URL 为空")
	}

	if err := lc.followLoginRedirects(loginURL); err != nil {
		return nil, err
	}

	// 从 Cookie Jar 汇总登录后在多个 qzone 域名下可见的关键 cookie
	cookieMap := lc.collectLoginCookies()
	var cookies []string
	var pSkey string
	var qq string

	for name, value := range cookieMap {
		cookies = append(cookies, name+"="+value)
		if name == "p_skey" {
			pSkey = value
		}
	}

	// 从 loginURL 中提取 QQ 号
	re := regexp.MustCompile(`uin=(\d+)`)
	if matches := re.FindStringSubmatch(loginURL); len(matches) > 1 {
		qq = matches[1]
		// 移除前导零
		qq = strings.TrimLeft(qq, "0")
	}

	cookieStr := normalizeCookie(strings.Join(cookies, "; "))
	gtk := calculateGTK(cookieStr)

	return &LoginResult{
		QQ:     qq,
		Cookie: cookieStr,
		PSKey:  pSkey,
		GTK:    gtk,
	}, nil
}

func (lc *LoginClient) followLoginRedirects(loginURL string) error {
	currentURL := loginURL
	referer := "https://xui.ptlogin2.qq.com/"

	for i := 0; i < 10; i++ {
		req, err := http.NewRequest("GET", currentURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		if referer != "" {
			req.Header.Set("Referer", referer)
		}

		resp, err := lc.httpClient.Do(req)
		if err != nil {
			return err
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		location := resp.Header.Get("Location")
		if location == "" || (resp.StatusCode < 300 || resp.StatusCode >= 400) {
			return nil
		}

		nextURL, err := resolveRedirectURL(currentURL, location)
		if err != nil {
			return err
		}

		referer = currentURL
		currentURL = nextURL
	}

	return fmt.Errorf("登录跳转次数过多")
}

func resolveRedirectURL(baseURL, location string) (string, error) {
	locURL, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	if locURL.IsAbs() {
		return locURL.String(), nil
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(locURL).String(), nil
}

func (lc *LoginClient) collectLoginCookies() map[string]string {
	result := make(map[string]string)
	if lc.httpClient == nil || lc.httpClient.Jar == nil {
		return result
	}

	targets := []string{
		"https://qq.com",
		"https://qzone.qq.com",
		"https://user.qzone.qq.com",
		"https://h5.qzone.qq.com",
		"https://taotao.qq.com",
		"https://photo.qzone.qq.com",
		"https://m.qzone.qq.com",
		"https://b.qzone.qq.com",
	}

	for _, rawURL := range targets {
		u, err := url.Parse(rawURL)
		if err != nil {
			continue
		}
		for _, cookie := range lc.httpClient.Jar.Cookies(u) {
			if cookie.Name == "" || cookie.Value == "" {
				continue
			}
			if _, exists := result[cookie.Name]; !exists {
				result[cookie.Name] = cookie.Value
			}
		}
	}

	return result
}
