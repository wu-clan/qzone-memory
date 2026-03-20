package qzone

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/qzone-memory/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// Client QQ 空间客户端
type Client struct {
	httpClient *http.Client
	cookie     string
	qq         string
	gtk        string
}

// NewClient 创建 QQ 空间客户端
func NewClient(cookie, qq string) *Client {
	normalizedCookie := normalizeCookie(cookie)
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cookie: normalizedCookie,
		qq:     qq,
		gtk:    calculateGTK(normalizedCookie),
	}
}

func normalizeCookie(cookie string) string {
	if cookie == "" {
		return ""
	}

	order := make([]string, 0)
	values := make(map[string]string)

	for _, part := range strings.Split(cookie, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		pieces := strings.SplitN(part, "=", 2)
		if len(pieces) != 2 {
			continue
		}

		name := strings.TrimSpace(pieces[0])
		value := strings.TrimSpace(pieces[1])
		if name == "" || value == "" {
			continue
		}

		if _, exists := values[name]; !exists {
			order = append(order, name)
		}
		values[name] = value
	}

	normalized := make([]string, 0, len(order))
	for _, name := range order {
		if value := values[name]; value != "" {
			normalized = append(normalized, name+"="+value)
		}
	}

	return strings.Join(normalized, "; ")
}

// calculateGTK 计算 gtk 参数
func calculateGTK(cookie string) string {
	// 优先使用 p_skey，缺失时回退到 skey
	token := ""
	for _, part := range strings.Split(cookie, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "p_skey=") {
			token = strings.TrimPrefix(part, "p_skey=")
			break
		}
	}

	if token == "" {
		for _, part := range strings.Split(cookie, ";") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "skey=") {
				token = strings.TrimPrefix(part, "skey=")
				break
			}
		}
	}

	if token == "" {
		return ""
	}

	hash := 5381
	for _, c := range token {
		hash += (hash << 5) + int(c)
	}
	return fmt.Sprintf("%d", hash&0x7fffffff)
}

// request 发送 HTTP 请求
func (c *Client) request(method, urlStr string, params url.Values) ([]byte, error) {
	var req *http.Request
	var err error

	if method == "GET" {
		if params != nil {
			urlStr = urlStr + "?" + params.Encode()
		}
		req, err = http.NewRequest(method, urlStr, nil)
	} else {
		req, err = http.NewRequest(method, urlStr, strings.NewReader(params.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Set("Cookie", c.cookie)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("Referer", c.refererFor(urlStr))
	req.Header.Set("Origin", "https://user.qzone.qq.com")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	body = decodeResponseBody(resp.Header.Get("Content-Type"), body)

	// 诊断日志：记录每个 API 请求的原始响应片段
	snippet := string(body)
	if len(snippet) > 500 {
		snippet = snippet[:500] + "..."
	}
	// 提取 API 路径用于日志
	apiPath := urlStr
	if idx := strings.Index(urlStr, "?"); idx > 0 {
		apiPath = urlStr[:idx]
	}
	if idx := strings.LastIndex(apiPath, "/"); idx > 0 {
		apiPath = apiPath[idx+1:]
	}
	logger.Warn("API 响应诊断", zap.String("api", apiPath), zap.Int("status", resp.StatusCode), zap.Int("len", len(body)), zap.String("body", snippet))

	return body, nil
}

func (c *Client) refererFor(urlStr string) string {
	if strings.Contains(urlStr, "/blognew/") {
		return "https://user.qzone.qq.com/" + c.qq + "/2"
	}
	return "https://user.qzone.qq.com/" + c.qq
}

func decodeResponseBody(contentType string, body []byte) []byte {
	if len(body) == 0 {
		return body
	}

	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return body
	}

	charset := strings.ToLower(strings.TrimSpace(params["charset"]))
	var decoder *transform.Reader
	switch charset {
	case "gbk", "gb2312":
		decoder = transform.NewReader(strings.NewReader(string(body)), simplifiedchinese.GBK.NewDecoder())
	case "gb18030":
		decoder = transform.NewReader(strings.NewReader(string(body)), simplifiedchinese.GB18030.NewDecoder())
	default:
		return body
	}

	decoded, err := io.ReadAll(decoder)
	if err != nil || len(decoded) == 0 {
		return body
	}
	return decoded
}

// parseCallback 解析 JSONP 回调
func parseCallback(data []byte) (map[string]interface{}, error) {
	str := strings.TrimSpace(string(data))

	// 兼容直接返回 JSON 的情况
	if strings.HasPrefix(str, "{") && strings.HasSuffix(str, "}") {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(str), &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	// 常规 JSONP：callback({...});
	start := strings.Index(str, "(")
	end := strings.LastIndex(str, ")")
	if start == -1 || end == -1 || end <= start {
		// 兜底从首个 { 到最后一个 } 提取 JSON 片段
		jsonStart := strings.Index(str, "{")
		jsonEnd := strings.LastIndex(str, "}")
		if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
			return nil, fmt.Errorf("无效的 JSONP 响应")
		}
		start = jsonStart - 1
		end = jsonEnd + 1
	}

	jsonStr := str[start+1 : end]
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		normalized := normalizeJSObjectToJSON(jsonStr)
		if normalized != "" {
			if err := json.Unmarshal([]byte(normalized), &result); err == nil {
				return result, nil
			}
		}
		if jsResult, jsErr := parseJavaScriptObject(jsonStr); jsErr == nil {
			return jsResult, nil
		}
		return nil, err
	}

	return result, nil
}

var objectKeyPattern = regexp.MustCompile(`([{\[,]\s*)([A-Za-z_][A-Za-z0-9_]*)(\s*:)`)

func normalizeJSObjectToJSON(input string) string {
	if input == "" {
		return ""
	}

	quotedKeys := objectKeyPattern.ReplaceAllString(input, `$1"$2"$3`)
	quotedKeys = strings.ReplaceAll(quotedKeys, ":undefined", ":null")
	quotedKeys = strings.ReplaceAll(quotedKeys, ":NaN", ":null")
	return convertSingleQuotedJSStrings(quotedKeys)
}

func convertSingleQuotedJSStrings(input string) string {
	var builder strings.Builder
	builder.Grow(len(input))

	inSingle := false
	escaped := false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		if inSingle {
			if escaped {
				switch ch {
				case '\\', '\'':
					builder.WriteByte(ch)
				case 'n':
					builder.WriteString(`\n`)
				case 'r':
					builder.WriteString(`\r`)
				case 't':
					builder.WriteString(`\t`)
				case 'x':
					if i+2 >= len(input) {
						return ""
					}
					hex := input[i+1 : i+3]
					value, err := strconv.ParseUint(hex, 16, 8)
					if err != nil {
						return ""
					}
					builder.WriteString(fmt.Sprintf(`\u%04x`, value))
					i += 2
				case 'u':
					if i+4 >= len(input) {
						return ""
					}
					hex := input[i+1 : i+5]
					if _, err := strconv.ParseUint(hex, 16, 16); err != nil {
						return ""
					}
					builder.WriteString(`\u` + hex)
					i += 4
				default:
					if ch == '"' {
						builder.WriteString(`\"`)
					} else {
						builder.WriteByte(ch)
					}
				}
				escaped = false
				continue
			}

			switch ch {
			case '\\':
				escaped = true
			case '"':
				builder.WriteString(`\"`)
			case '\'':
				builder.WriteByte('"')
				inSingle = false
			case '\n':
				builder.WriteString(`\n`)
			case '\r':
				builder.WriteString(`\r`)
			case '\t':
				builder.WriteString(`\t`)
			default:
				builder.WriteByte(ch)
			}
			continue
		}

		if ch == '\'' {
			builder.WriteByte('"')
			inSingle = true
			continue
		}

		builder.WriteByte(ch)
	}

	if inSingle {
		return ""
	}

	return builder.String()
}

func parseJavaScriptObject(input string) (map[string]interface{}, error) {
	vm := goja.New()
	value, err := vm.RunString("(" + input + ")")
	if err != nil {
		return nil, err
	}

	result, ok := value.Export().(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("JS 对象解析失败")
	}
	return result, nil
}

// getStringValue 安全获取字符串值
func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// getIntValue 安全获取整数值
func getIntValue(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	if v, ok := m[key].(string); ok {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return 0
}

// getInt64Value 安全获取 int64 值
func getInt64Value(m map[string]interface{}, key string) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	if v, ok := m[key].(string); ok {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			return parsed
		}
	}
	return 0
}

// getMapValue 安全获取 map 值
func getMapValue(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

// getSliceValue 安全获取 slice 值
func getSliceValue(m map[string]interface{}, key string) []interface{} {
	if v, ok := m[key].([]interface{}); ok {
		return v
	}
	return nil
}

// checkResponseCode 检查响应码，识别 Cookie 过期等认证错误
func checkResponseCode(result map[string]interface{}, action string) error {
	code, ok := result["code"].(float64)
	if !ok {
		// 尝试从字符串解析
		if codeStr, ok := result["code"].(string); ok {
			if parsed, err := strconv.ParseFloat(codeStr, 64); err == nil {
				code = parsed
			} else {
				return nil
			}
		} else {
			return nil
		}
	}
	if code == 0 {
		return nil
	}
	intCode := int(code)
	if intCode == -4001 || intCode == -3000 || intCode == -10000 {
		return ErrCookieExpired
	}
	return fmt.Errorf("%s: code=%d", action, intCode)
}

// LogEmptyResponse 当 API 返回空数据时记录响应片段用于诊断
func LogEmptyResponse(apiName string, data []byte) {
	snippet := string(data)
	if len(snippet) > 500 {
		snippet = snippet[:500] + "..."
	}
	logger.Warn("API 返回空数据", zap.String("api", apiName), zap.String("body", snippet))
}

// GetPortrait 获取用户头像 URL
func GetPortrait(qq string) string {
	return fmt.Sprintf("https://q.qlogo.cn/headimg_dl?dst_uin=%s&spec=100", qq)
}

// CheckCookie 通过获取第一条说说来检测 Cookie 是否有效
func (c *Client) CheckCookie() error {
	params := url.Values{
		"uin":     {c.qq},
		"hostUin": {c.qq},
		"ftype":   {"0"},
		"sort":    {"0"},
		"pos":     {"0"},
		"num":     {"1"},
		"g_tk":    {c.gtk},
		"format":  {"json"},
	}

	apiURL := "https://user.qzone.qq.com/proxy/domain/taotao.qq.com/cgi-bin/emotion_cgi_msglist_v6"
	data, err := c.request("GET", apiURL, params)
	if err != nil {
		return fmt.Errorf("网络请求失败: %w", err)
	}

	result, err := parseCallback(data)
	if err != nil {
		return fmt.Errorf("Cookie 无效或已过期（响应解析失败）")
	}

	return checkResponseCode(result, "Cookie 验证失败")
}
