package vdi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// VDIClient VDI API 客户端
type VDIClient struct {
	ServerURL  string
	HTTPClient *http.Client
	authToken  string
	authExpiry time.Time
}

// NewVDIClient 创建 VDI 客户端
func NewVDIClient(serverURL string, timeoutMs int) *VDIClient {
	return &VDIClient{
		ServerURL: serverURL,
		HTTPClient: &http.Client{
			Timeout: time.Duration(timeoutMs) * time.Millisecond,
		},
	}
}

// Authenticate 认证并获取 Token
func (c *VDIClient) Authenticate(username, password string) error {
	url := fmt.Sprintf("%s/api/v1/auth/login", c.ServerURL)

	reqBody, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})

	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("VDI 认证请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return fmt.Errorf("解析认证响应失败: %w", err)
	}

	if authResp.Code != 0 {
		return fmt.Errorf("VDI 认证失败: %s", authResp.Message)
	}

	c.authToken = authResp.Token
	c.authExpiry = time.Now().Add(24 * time.Hour)
	return nil
}

// ensureAuth 确保 Token 有效
func (c *VDIClient) ensureAuth() error {
	if c.authToken == "" || time.Now().After(c.authExpiry) {
		return fmt.Errorf("VDI Token 无效或已过期")
	}
	return nil
}

// makeRequest 发送带认证的请求
func (c *VDIClient) makeRequest(method, endpoint string, body []byte) ([]byte, error) {
	if err := c.ensureAuth(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s%s", c.ServerURL, endpoint)
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// CreateVM 创建虚拟机
func (c *VDIClient) CreateVM(req *CreateVMRequest) (string, error) {
	body, _ := json.Marshal(req)
	respBody, err := c.makeRequest("POST", "/api/v1/vm/create", body)
	if err != nil {
		return "", err
	}

	var resp CreateVMResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}

	if resp.Code != 0 {
		return "", fmt.Errorf("创建虚拟机失败: %s", resp.Message)
	}

	return resp.VMID, nil
}

// DeleteVM 删除虚拟机
func (c *VDIClient) DeleteVM(vmID string) error {
	body, _ := json.Marshal(VMOperationRequest{VMID: vmID})
	respBody, err := c.makeRequest("POST", "/api/v1/vm/delete", body)
	if err != nil {
		return err
	}

	var resp VDIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("删除虚拟机失败: %s", resp.Message)
	}

	return nil
}

// StartVM 启动虚拟机
func (c *VDIClient) StartVM(vmID string) error {
	body, _ := json.Marshal(VMOperationRequest{VMID: vmID})
	respBody, err := c.makeRequest("POST", "/api/v1/vm/start", body)
	if err != nil {
		return err
	}

	var resp VDIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("启动虚拟机失败: %s", resp.Message)
	}

	return nil
}

// StopVM 停止虚拟机
func (c *VDIClient) StopVM(vmID string) error {
	body, _ := json.Marshal(VMOperationRequest{VMID: vmID})
	respBody, err := c.makeRequest("POST", "/api/v1/vm/stop", body)
	if err != nil {
		return err
	}

	var resp VDIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("停止虚拟机失败: %s", resp.Message)
	}

	return nil
}

// RestartVM 重启虚拟机
func (c *VDIClient) RestartVM(vmID string) error {
	body, _ := json.Marshal(VMOperationRequest{VMID: vmID})
	respBody, err := c.makeRequest("POST", "/api/v1/vm/restart", body)
	if err != nil {
		return err
	}

	var resp VDIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("重启虚拟机失败: %s", resp.Message)
	}

	return nil
}

// RenameVM 重命名虚拟机
func (c *VDIClient) RenameVM(vmID, newName string) error {
	body, _ := json.Marshal(RenameVMRequest{VMID: vmID, VMName: newName})
	respBody, err := c.makeRequest("POST", "/api/v1/vm/rename", body)
	if err != nil {
		return err
	}

	var resp VDIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("重命名虚拟机失败: %s", resp.Message)
	}

	return nil
}

// BindUser 绑定用户
func (c *VDIClient) BindUser(vmID, username, password string) error {
	body, _ := json.Marshal(BindUserRequest{VMID: vmID, Username: username, Password: password})
	respBody, err := c.makeRequest("POST", "/api/v1/vm/bind_user", body)
	if err != nil {
		return err
	}

	var resp VDIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("绑定用户失败: %s", resp.Message)
	}

	return nil
}

// UnbindUser 解绑用户
func (c *VDIClient) UnbindUser(vmID string) error {
	body, _ := json.Marshal(VMOperationRequest{VMID: vmID})
	respBody, err := c.makeRequest("POST", "/api/v1/vm/unbind_user", body)
	if err != nil {
		return err
	}

	var resp VDIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("解绑用户失败: %s", resp.Message)
	}

	return nil
}

// ListVMs 获取虚拟机列表
func (c *VDIClient) ListVMs() ([]VMInfo, error) {
	respBody, err := c.makeRequest("POST", "/api/v1/vm/list", nil)
	if err != nil {
		return nil, err
	}

	var resp VMListResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("获取虚拟机列表失败: %s", resp.Message)
	}

	return resp.Data, nil
}

// GetVMInfo 获取虚拟机详情
func (c *VDIClient) GetVMInfo(vmID string) (*VMInfo, error) {
	body, _ := json.Marshal(VMOperationRequest{VMID: vmID})
	respBody, err := c.makeRequest("POST", "/api/v1/vm/get_info", body)
	if err != nil {
		return nil, err
	}

	var resp VMInfoResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("获取虚拟机详情失败: %s", resp.Message)
	}

	return &resp.Data, nil
}
