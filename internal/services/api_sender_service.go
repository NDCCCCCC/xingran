package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

const (
	// apiSenderTimeout API 发送 HTTP 客户端超时
	apiSenderTimeout = 30 * time.Second
	// apiSenderRetryBaseMs API 发送重试退避基数（毫秒）
	apiSenderRetryBaseMs = 200
)

// APISenderService API发送服务
type APISenderService struct {
	db     *gorm.DB
	client *http.Client
}

// NewAPISenderService 创建API发送服务
func NewAPISenderService(db *gorm.DB) *APISenderService {
	return &APISenderService{
		db: db,
		client: &http.Client{
			Timeout: apiSenderTimeout,
		},
	}
}

// APIMessage API消息
type APIMessage struct {
	Recipients []string               `json:"recipients"` // 接收者列表（手机号、用户ID等）
	Title      string                 `json:"title"`      // 标题
	Content    string                 `json:"content"`    // 内容
	Data       map[string]interface{} `json:"data"`       // 额外数据
}

// APISendResult API发送结果
type APISendResult struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	HTTPCode     int    `json:"httpCode,omitempty"`
	ResponseBody string `json:"responseBody,omitempty"`
	Error        error  `json:"error,omitempty"`
	RetryCount   int    `json:"retryCount,omitempty"`
}

// Send 发送API通知
func (s *APISenderService) Send(ctx context.Context, configID string, msg *APIMessage) *APISendResult {
	// 获取API配置
	configService := NewNotificationConfigService(s.db)
	config, err := configService.GetAPINotificationConfigByID(ctx, configID)
	if err != nil {
		return &APISendResult{
			Success: false,
			Message: "获取API配置失败",
			Error:   err,
		}
	}

	// 检查配置状态
	if config.Status != int(models.NotificationConfigStatusNormal) {
		return &APISendResult{
			Success: false,
			Message: "API配置未启用",
		}
	}

	// 构建请求
	var lastErr error
	var result *APISendResult

	// 重试机制
	for i := 0; i <= config.RetryCount; i++ {
		result = s.sendRequest(ctx, config, msg)
		result.RetryCount = i

		if result.Success {
			return result
		}

		lastErr = result.Error

		// 如果不是最后一次重试，等待一段时间
		if i < config.RetryCount {
			time.Sleep(time.Duration(apiSenderRetryBaseMs*(i+1)) * time.Millisecond)
		}
	}

	return &APISendResult{
		Success:    false,
		Message:    fmt.Sprintf("API发送失败，已重试%d次", config.RetryCount),
		Error:      lastErr,
		RetryCount: config.RetryCount,
	}
}

// SendWithDefaultConfig 使用默认配置发送API通知
func (s *APISenderService) SendWithDefaultConfig(ctx context.Context, configType models.APIConfigType, msg *APIMessage) *APISendResult {
	configService := NewNotificationConfigService(s.db)
	configs, _, err := configService.ListAPINotificationConfigs(ctx, 1, 1, &configType, nil)
	if err != nil || len(configs) == 0 {
		return &APISendResult{
			Success: false,
			Message: "未找到默认API配置",
			Error:   err,
		}

	}

	// 查找默认配置
	var defaultConfig *models.APINotificationConfig
	for _, config := range configs {
		if config.IsDefault {
			defaultConfig = &config
			break
		}
	}

	if defaultConfig == nil {
		return &APISendResult{
			Success: false,
			Message: "未设置默认API配置",
		}
	}

	return s.Send(ctx, defaultConfig.ID, msg)
}

// sendRequest 发送HTTP请求
func (s *APISenderService) sendRequest(ctx context.Context, config *models.APINotificationConfig, msg *APIMessage) *APISendResult {
	// 构建请求体
	body, err := s.buildRequestBody(config, msg)
	if err != nil {
		return &APISendResult{
			Success: false,
			Message: "构建请求体失败",
			Error:   err,
		}
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, config.APIMethod, config.APIURL, body)
	if err != nil {
		return &APISendResult{
			Success: false,
			Message: "创建HTTP请求失败",
			Error:   err,
		}
	}

	// 设置请求头
	s.setRequestHeaders(req, config)

	// 设置认证
	if authErr := s.setAuthentication(req, config); authErr != nil {
		return &APISendResult{
			Success: false,
			Message: "设置认证失败",
			Error:   authErr,
		}
	}

	// 设置超时
	if config.Timeout > 0 {
		s.client.Timeout = time.Duration(config.Timeout) * time.Second
	}

	// 发送请求
	startTime := time.Now()
	resp, err := s.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		return &APISendResult{
			Success: false,
			Message: "HTTP请求失败",
			Error:   err,
		}
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APISendResult{
			Success:      false,
			Message:      "读取响应失败",
			HTTPCode:     resp.StatusCode,
			ResponseBody: string(respBody),
			Error:        err,
		}
	}

	// 检查HTTP状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APISendResult{
			Success:      false,
			Message:      fmt.Sprintf("HTTP请求失败，耗时: %v", duration),
			HTTPCode:     resp.StatusCode,
			ResponseBody: string(respBody),
			Error:        fmt.Errorf("HTTP status code: %d", resp.StatusCode),
		}
	}

	return &APISendResult{
		Success:      true,
		Message:      fmt.Sprintf("API发送成功，耗时: %v", duration),
		HTTPCode:     resp.StatusCode,
		ResponseBody: string(respBody),
	}
}

// buildRequestBody 构建请求体
func (s *APISenderService) buildRequestBody(config *models.APINotificationConfig, msg *APIMessage) (io.Reader, error) {
	// 如果有模板，使用模板构建请求体
	if config.TemplateBody != "" {
		return s.buildFromTemplate(config.TemplateBody, msg)
	}

	// 否则使用默认的JSON格式
	return s.buildDefaultBody(msg)
}

// buildFromTemplate 从模板构建请求体
func (s *APISenderService) buildFromTemplate(template string, msg *APIMessage) (io.Reader, error) {
	// 替换模板中的变量
	body := template
	body = strings.ReplaceAll(body, "{{title}}", msg.Title)
	body = strings.ReplaceAll(body, "{{content}}", msg.Content)

	// 替换接收者列表
	if len(msg.Recipients) > 0 {
		recipients := strings.Join(msg.Recipients, ",")
		body = strings.ReplaceAll(body, "{{recipients}}", recipients)
	}

	// 替换额外数据
	for key, value := range msg.Data {
		placeholder := fmt.Sprintf("{{%s}}", key)
		body = strings.ReplaceAll(body, placeholder, fmt.Sprintf("%v", value))
	}

	return strings.NewReader(body), nil
}

// buildDefaultBody 构建默认请求体
func (s *APISenderService) buildDefaultBody(msg *APIMessage) (io.Reader, error) {
	data := map[string]interface{}{
		"title":      msg.Title,
		"content":    msg.Content,
		"recipients": msg.Recipients,
	}

	// 合并额外数据
	for k, v := range msg.Data {
		data[k] = v
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("序列化JSON失败: %w", err)
	}

	return bytes.NewReader(jsonData), nil
}

// setRequestHeaders 设置请求头
func (s *APISenderService) setRequestHeaders(req *http.Request, config *models.APINotificationConfig) {
	// 设置默认Content-Type
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// 设置配置中的请求头
	if config.Headers != nil {
		for key, value := range config.Headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			} else {
				jsonValue, _ := json.Marshal(value)
				req.Header.Set(key, string(jsonValue))
			}
		}
	}

	// 设置User-Agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Xingran-Notification/1.0")
	}
}

// setAuthentication 设置认证
func (s *APISenderService) setAuthentication(req *http.Request, config *models.APINotificationConfig) error {
	switch config.AuthType {
	case models.AuthTypeNone:
		// 无需认证
		return nil

	case models.AuthTypeBasic:
		// Basic认证
		if config.AuthConfig == nil {
			return fmt.Errorf("Basic认证需要配置用户名和密码")
		}
		username, _ := config.AuthConfig["username"].(string)
		password, _ := config.AuthConfig["password"].(string)
		auth := username + ":" + password
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
		return nil

	case models.AuthTypeBearer:
		// Bearer Token认证
		if config.AuthConfig == nil {
			return fmt.Errorf("Bearer认证需要配置Token")
		}
		token, _ := config.AuthConfig["token"].(string)
		req.Header.Set("Authorization", "Bearer "+token)
		return nil

	case models.AuthTypeAPIKey:
		// API Key认证
		if config.AuthConfig == nil {
			return fmt.Errorf("API Key认证需要配置Key")
		}
		key, _ := config.AuthConfig["key"].(string)
		value, _ := config.AuthConfig["value"].(string)
		headerName := key
		if customHeader, ok := config.AuthConfig["header_name"].(string); ok && customHeader != "" {
			headerName = customHeader
		}
		req.Header.Set(headerName, value)
		return nil

	default:
		return fmt.Errorf("不支持的认证类型: %s", config.AuthType)
	}
}

// SendNoticeAPI 发送通知API
func (s *APISenderService) SendNoticeAPI(ctx context.Context, configID string, notice *models.Notice, recipients []string) *APISendResult {
	// 构建API消息
	msg := &APIMessage{
		Recipients: recipients,
		Title:      notice.NoticeTitle,
		Content:    notice.NoticeContent,
		Data: map[string]interface{}{
			"noticeId":    notice.ID,
			"noticeType":  notice.NoticeType,
			"priority":    int(notice.Priority),
			"publishTime": notice.PublishTime,
			"isMarkdown":  notice.IsMarkdown,
		},
	}

	return s.Send(ctx, configID, msg)
}

// SendSMS 发送短信
func (s *APISenderService) SendSMS(ctx context.Context, configID string, phones []string, content string) *APISendResult {
	msg := &APIMessage{
		Recipients: phones,
		Content:    content,
		Data: map[string]interface{}{
			"type": "sms",
		},
	}

	return s.Send(ctx, configID, msg)
}

// SendWebhook 发送Webhook
func (s *APISenderService) SendWebhook(ctx context.Context, configID string, data map[string]interface{}) *APISendResult {
	msg := &APIMessage{
		Title:   "Webhook通知",
		Content: "",
		Data:    data,
	}

	return s.Send(ctx, configID, msg)
}

// TestAPIConfig 测试API配置
func (s *APISenderService) TestAPIConfig(ctx context.Context, configID string) *APISendResult {
	configService := NewNotificationConfigService(s.db)
	config, err := configService.GetAPINotificationConfigByID(ctx, configID)
	if err != nil {
		return &APISendResult{
			Success: false,
			Message: "获取API配置失败",
			Error:   err,
		}
	}

	// 构建测试消息
	msg := &APIMessage{
		Title:      "测试消息",
		Content:    "这是一条测试消息",
		Recipients: []string{},
		Data: map[string]interface{}{
			"test":      true,
			"timestamp": time.Now().Unix(),
		},
	}

	return s.sendRequest(ctx, config, msg)
}
