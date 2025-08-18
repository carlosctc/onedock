package onedockclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client OneDock API 客户端
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	timeout    time.Duration
	debug      bool
}

// Option 客户端配置选项
type Option func(*Client)

// WithTimeout 设置请求超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
		c.httpClient.Timeout = timeout
	}
}

// WithHTTPClient 设置自定义 HTTP 客户端
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithDebug 启用调试模式
func WithDebug(debug bool) Option {
	return func(c *Client) {
		c.debug = debug
	}
}

// New 创建新的 OneDock API 客户端
func New(baseURL, token string, options ...Option) *Client {
	// 确保 baseURL 格式正确
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	client := &Client{
		baseURL: baseURL,
		token:   token,
		timeout: 30 * time.Second,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		debug: false,
	}

	// 应用选项
	for _, option := range options {
		option(client)
	}

	return client
}

// doRequest 执行 HTTP 请求
func (c *Client) doRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	url := c.baseURL + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	if c.debug {
		fmt.Printf("Request: %s %s\n", method, url)
		if body != nil {
			fmt.Printf("Body: %+v\n", body)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if c.debug {
		fmt.Printf("Response Status: %s\n", resp.Status)
	}

	return resp, nil
}

// parseResponse 解析响应
func (c *Client) parseResponse(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if c.debug {
		fmt.Printf("Response Body: %s\n", string(body))
	}

	// 检查 HTTP 状态码
	if resp.StatusCode >= 400 {
		var apiError APIError
		if err := json.Unmarshal(body, &apiError); err != nil {
			return NewAPIError(resp.StatusCode, string(body))
		}
		apiError.Code = resp.StatusCode
		return &apiError
	}

	// 如果 result 为 nil，说明不需要解析响应体
	if result == nil {
		return nil
	}
	data := new(Response)
	data.Data = result

	err = json.Unmarshal(body, data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// buildURL 构建带查询参数的 URL
func (c *Client) buildURL(endpoint string, params map[string]string) string {
	if len(params) == 0 {
		return endpoint
	}

	u, _ := url.Parse(endpoint)
	q := u.Query()
	for key, value := range params {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// GetBaseURL 获取基础 URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// SetToken 设置认证 token
func (c *Client) SetToken(token string) {
	c.token = token
}

// GetToken 获取当前 token
func (c *Client) GetToken() string {
	return c.token
}
