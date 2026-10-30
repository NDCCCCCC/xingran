package communication

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// APIClient HTTP API 客户端
type APIClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	logger     logger.Logger
}

// NewAPIClient 创建 API 客户端
func NewAPIClient(cfg *config.BackendConfig, log logger.Logger) *APIClient {
	return &APIClient{
		baseURL: cfg.BaseURL,
		token:   cfg.APIToken,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: log,
	}
}

// Register 注册 Worker
func (c *APIClient) Register(ctx context.Context, req *types.WorkerRegisterRequest) (*types.WorkerRegisterResponse, error) {
	c.logger.Info("注册 Worker", logger.String("worker_id", req.WorkerID))

	var resp types.APIResponse
	err := c.post(ctx, "/workers/register", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("注册失败: %w", err)
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("注册失败: %s", resp.Message)
	}

	// 解析响应数据
	dataBytes, _ := json.Marshal(resp.Data)
	var registerResp types.WorkerRegisterResponse
	if err := json.Unmarshal(dataBytes, &registerResp); err != nil {
		return &registerResp, nil
	}

	return &registerResp, nil
}

// Heartbeat 发送心跳
func (c *APIClient) Heartbeat(ctx context.Context, req *types.WorkerHeartbeatRequest) error {
	return c.post(ctx, fmt.Sprintf("/workers/%s/heartbeat", req.WorkerID), req, nil)
}

// ReportProgress 上报进度
func (c *APIClient) ReportProgress(ctx context.Context, req *types.ProgressReport) error {
	return c.post(ctx, fmt.Sprintf("/workers/%s/progress", req.WorkerID), req, nil)
}

// post 发送 POST 请求
func (c *APIClient) post(ctx context.Context, path string, body, result interface{}) error {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bodyReader)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("解析响应失败: %w", err)
		}
	}

	return nil
}
