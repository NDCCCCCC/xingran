package vdi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"gorm.io/gorm"
)

// VDIClientExtended VDI客户端扩展接口
type VDIClientExtended interface {
	// 认证
	Authenticate(ctx context.Context) (string, error)

	// 虚拟机查询
	GetVM(ctx context.Context, vmID string) (*VDIVMDetail, error)
	ListVMs(ctx context.Context, resourceID string) ([]VDIVMSummary, error)
	GetUserVMs(ctx context.Context, userID string) ([]VDIVMSummary, error)

	// 虚拟机操作
	OperateVM(ctx context.Context, vmIDs []string, action string) error
	DeleteVM(ctx context.Context, vmIDs []string) error
	ConfigIP(ctx context.Context, req []ConfigIPRequestExtended) error
	RenameVM(ctx context.Context, vmID, newName string) error

	// 用户关联
	BindUser(ctx context.Context, vmID, userID string) (*VDIVMDetail, error)
	GetAvailableUsers(ctx context.Context, vmID string) ([]VDIUser, error)
}

// vdiClientExtendedImpl VDI客户端扩展实现
type vdiClientExtendedImpl struct {
	authManager *VDIAuthManager
	db          *gorm.DB
	config      config.VDIServerConfig
	serverID    string
	httpClient  *http.Client
}

// NewVDIClientExtended 创建VDI客户端扩展
func NewVDIClientExtended(db *gorm.DB, serverID string, cfg config.VDIServerConfig) VDIClientExtended {
	return &vdiClientExtendedImpl{
		authManager: NewVDIAuthManager(db, serverID, cfg),
		db:          db,
		config:      cfg,
		serverID:    serverID,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// VDIVMDetail 虚拟机详情
type VDIVMDetail struct {
	VMID          string  `json:"vm_id"`
	Name          string  `json:"name"`
	ResourceID    string  `json:"resource_id"`
	Status        int     `json:"status"`
	PowerState    string  `json:"power_state"`
	IPAddress     string  `json:"ip_address"`
	MACAddress    string  `json:"mac_address"`
	OSType        string  `json:"os_type"`
	CPU           int     `json:"cpu"`
	Memory        int     `json:"memory"`
	Disk          int     `json:"disk"`
	BoundUserID   *string `json:"bound_user_id"`
	BoundUserName *string `json:"bound_user_name"`
	PolicyGroupID *string `json:"policy_group_id"`
}

// VDIVMSummary 虚拟机摘要信息
type VDIVMSummary struct {
	VMID       string `json:"vm_id"`
	Name       string `json:"name"`
	PowerState string `json:"power_state"`
	Status     int    `json:"status"`
}

// ConfigIPRequestExtended 配置IP请求
type ConfigIPRequestExtended struct {
	VMInfos []VMIPInfo `json:"vm_infos"`
}

// VMIPInfo 虚拟机IP信息
type VMIPInfo struct {
	VMID      string `json:"vm_id"`
	IPAddress string `json:"ip_address"`
	Netmask   string `json:"netmask"`
	Gateway   string `json:"gateway"`
}

// VDIUser VDI用户信息
type VDIUser struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	DN       string `json:"dn"`
}

// Authenticate 认证
func (c *vdiClientExtendedImpl) Authenticate(ctx context.Context) (string, error) {
	return c.authManager.Authenticate(ctx)
}

// GetVM 获取虚拟机详情
func (c *vdiClientExtendedImpl) GetVM(ctx context.Context, vmID string) (*VDIVMDetail, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ErrorCode    int         `json:"error_code"`
		ErrorMessage string      `json:"error_message"`
		Data         VDIVMDetail `json:"data"`
	}

	path := fmt.Sprintf("/v1/vm/%s", vmID)
	if err := c.callAPI(ctx, token, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return &resp.Data, nil
}

// ListVMs 获取资源下的所有虚拟机
func (c *vdiClientExtendedImpl) ListVMs(ctx context.Context, resourceID string) ([]VDIVMSummary, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ErrorCode    int           `json:"error_code"`
		ErrorMessage string        `json:"error_message"`
		Data         []VDIVMSummary `json:"data"`
	}

	path := fmt.Sprintf("/v1/resource/%s/vms", resourceID)
	if err := c.callAPI(ctx, token, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return resp.Data, nil
}

// GetUserVMs 获取用户关联的虚拟机列表
func (c *vdiClientExtendedImpl) GetUserVMs(ctx context.Context, userID string) ([]VDIVMSummary, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ErrorCode    int           `json:"error_code"`
		ErrorMessage string        `json:"error_message"`
		Data         []VDIVMSummary `json:"data"`
	}

	path := fmt.Sprintf("/v1/user/%s/vms", userID)
	if err := c.callAPI(ctx, token, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return resp.Data, nil
}

// OperateVM 操作虚拟机（批量）
func (c *vdiClientExtendedImpl) OperateVM(ctx context.Context, vmIDs []string, action string) error {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return err
	}

	req := map[string]interface{}{
		"vm_ids": vmIDs,
		"action": action,
	}

	var resp struct {
		ErrorCode    int      `json:"error_code"`
		ErrorMessage string   `json:"error_message"`
		Data         []string `json:"data"`
	}

	if err := c.callAPI(ctx, token, "POST", "/v1/vm/operate", req, &resp); err != nil {
		return err
	}

	if resp.ErrorCode != 0 {
		return &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return nil
}

// DeleteVM 删除虚拟机（批量）
func (c *vdiClientExtendedImpl) DeleteVM(ctx context.Context, vmIDs []string) error {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return err
	}

	req := map[string]interface{}{
		"vm_ids": vmIDs,
	}

	var resp struct {
		ErrorCode    int      `json:"error_code"`
		ErrorMessage string   `json:"error_message"`
		Data         []string `json:"data"`
	}

	if err := c.callAPI(ctx, token, "DELETE", "/v1/vm", req, &resp); err != nil {
		return err
	}

	if resp.ErrorCode != 0 {
		return &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return nil
}

// ConfigIP 配置虚拟机IP（批量）
func (c *vdiClientExtendedImpl) ConfigIP(ctx context.Context, req []ConfigIPRequestExtended) error {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return err
	}

	if len(req) == 0 {
		return nil
	}

	var resp struct {
		ErrorCode    int      `json:"error_code"`
		ErrorMessage string   `json:"error_message"`
		Data         []string `json:"data"`
	}

	if err := c.callAPI(ctx, token, "POST", "/v1/vm/config_ip", req[0], &resp); err != nil {
		return err
	}

	if resp.ErrorCode != 0 {
		return &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return nil
}

// RenameVM 重命名虚拟机
func (c *vdiClientExtendedImpl) RenameVM(ctx context.Context, vmID, newName string) error {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return err
	}

	req := map[string]string{
		"name": newName,
	}

	var resp struct {
		ErrorCode    int    `json:"error_code"`
		ErrorMessage string `json:"error_message"`
		Data         string `json:"data"`
	}

	path := fmt.Sprintf("/v1/vm/%s/name", vmID)
	if err := c.callAPI(ctx, token, "PUT", path, req, &resp); err != nil {
		return err
	}

	if resp.ErrorCode != 0 {
		return &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return nil
}

// BindUser 绑定用户
func (c *vdiClientExtendedImpl) BindUser(ctx context.Context, vmID, userID string) (*VDIVMDetail, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	req := map[string]string{
		"user_id": userID,
	}

	var resp struct {
		ErrorCode    int         `json:"error_code"`
		ErrorMessage string      `json:"error_message"`
		Data         VDIVMDetail `json:"data"`
	}

	path := fmt.Sprintf("/v1/vm/%s/bind_user", vmID)
	if err := c.callAPI(ctx, token, "POST", path, req, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return &resp.Data, nil
}

// GetAvailableUsers 获取可关联用户列表
func (c *vdiClientExtendedImpl) GetAvailableUsers(ctx context.Context, vmID string) ([]VDIUser, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ErrorCode    int      `json:"error_code"`
		ErrorMessage string   `json:"error_message"`
		Data         []VDIUser `json:"data"`
	}

	path := fmt.Sprintf("/v1/vm/%s/available_users", vmID)
	if err := c.callAPI(ctx, token, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return resp.Data, nil
}

// callAPI 调用VDI API
func (c *vdiClientExtendedImpl) callAPI(ctx context.Context, token, method, path string, body, result interface{}) error {
	url := fmt.Sprintf("%s%s", c.config.Endpoint, path)

	var req *http.Request
	var err error

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return err
		}
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonBody))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}

	if err != nil {
		return err
	}

	req.Header.Set("Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")

	// 重试逻辑
	var lastErr error
	for i := 0; i < 3; i++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		return json.NewDecoder(resp.Body).Decode(result)
	}

	return fmt.Errorf("HTTP request failed after retries: %w", lastErr)
}
