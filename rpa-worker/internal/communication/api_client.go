package communication

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// APIClient HTTP API client
type APIClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	logger     logger.Logger
}

// NewAPIClient create API client
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

// Register register worker
func (c *APIClient) Register(ctx context.Context, req *types.WorkerRegisterRequest) (*types.WorkerRegisterResponse, error) {
	c.logger.Info("registering worker", logger.String("worker_id", req.WorkerID))

	var resp types.APIResponse
	err := c.post(ctx, "/rpa/workers/register", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("register failed: %w", err)
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("register failed: %s", resp.Message)
	}

	// Parse response data
	dataBytes, _ := json.Marshal(resp.Data)
	var registerResp types.WorkerRegisterResponse
	if err := json.Unmarshal(dataBytes, &registerResp); err != nil {
		return &registerResp, nil
	}

	return &registerResp, nil
}

// Heartbeat send heartbeat
func (c *APIClient) Heartbeat(ctx context.Context, req *types.WorkerHeartbeatRequest) error {
	return c.post(ctx, fmt.Sprintf("/rpa/workers/%s/heartbeat", req.WorkerID), req, nil)
}

// ReportProgress report progress
func (c *APIClient) ReportProgress(ctx context.Context, req *types.ProgressReport) error {
	// WorkerID is already in the request body
	return c.post(ctx, "/rpa/workers/progress", req, nil)
}

// post send POST request
func (c *APIClient) post(ctx context.Context, path string, body, result interface{}) error {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body failed: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("parse response failed: %w", err)
		}
	}

	return nil
}
