package vdi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
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

	// 资源管理
	ListResources(ctx context.Context, groupID string) ([]VDIResource, error)
	ListResourceGroups(ctx context.Context) ([]VDIResourceGroup, error)
	ListResourceServers(ctx context.Context, resourceID string, page, pageSize int) ([]VDIVMResource, int, error)
}

// vdiClientExtendedImpl VDI客户端扩展实现
type vdiClientExtendedImpl struct {
	authManager *VDIAuthManager
	db          *gorm.DB
	server      models.VDIServer
	serverID    string
	httpClient  *http.Client
}

// newVDIHTTPClient 创建跳过TLS证书验证的HTTP客户端
// VDI服务器通常使用自签名证书，需要跳过验证
func newVDIHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // 跳过TLS证书验证（VDI服务器使用自签名证书）
			},
		},
	}
}

// NewVDIClientExtended 创建VDI客户端扩展（从配置文件 - 已废弃）
// 保留此函数以兼容旧代码，建议使用 NewVDIClientFromDB
func NewVDIClientExtended(db *gorm.DB, serverID string, cfg interface{}) VDIClientExtended {
	// 从数据库读取服务器配置
	var server models.VDIServer
	if err := db.First(&server, "id = ?", serverID).Error; err != nil {
		// 如果读取失败，返回空客户端
		return &vdiClientExtendedImpl{
			db:         db,
			serverID:   serverID,
			httpClient: newVDIHTTPClient(),
		}
	}

	return &vdiClientExtendedImpl{
		authManager: NewVDIAuthManager(db, serverID, server),
		db:          db,
		server:      server,
		serverID:    serverID,
		httpClient:  newVDIHTTPClient(),
	}
}

// NewVDIClientFromDB 从数据库创建VDI客户端扩展
func NewVDIClientFromDB(db *gorm.DB, serverID string) VDIClientExtended {
	var server models.VDIServer
	if err := db.First(&server, "id = ?", serverID).Error; err != nil {
		// 如果读取失败，返回空客户端
		return &vdiClientExtendedImpl{
			db:         db,
			serverID:   serverID,
			httpClient: newVDIHTTPClient(),
		}
	}

	return &vdiClientExtendedImpl{
		authManager: NewVDIAuthManager(db, serverID, server),
		db:          db,
		server:      server,
		serverID:    serverID,
		httpClient:  newVDIHTTPClient(),
	}
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

// ListVMs 获取虚拟机列表
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

	path := "/v1/vm/list"
	if resourceID != "" {
		path = fmt.Sprintf("/v1/vm/list?resource_id=%s", resourceID)
	}

	if err := c.callAPI(ctx, token, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return resp.Data, nil
}

// GetUserVMs 获取用户的虚拟机列表
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

	path := fmt.Sprintf("/v1/user/%s/vm", userID)
	if err := c.callAPI(ctx, token, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return resp.Data, nil
}

// OperateVM 操作虚拟机
func (c *vdiClientExtendedImpl) OperateVM(ctx context.Context, vmIDs []string, action string) error {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return err
	}

	req := struct {
		VMIDs  []string `json:"vm_ids"`
		Action string   `json:"action"`
	}{
		VMIDs:  vmIDs,
		Action: action,
	}

	return c.callAPI(ctx, token, "POST", "/v1/vm/operate", req, nil)
}

// DeleteVM 删除虚拟机
func (c *vdiClientExtendedImpl) DeleteVM(ctx context.Context, vmIDs []string) error {
	return c.OperateVM(ctx, vmIDs, "delete")
}

// ConfigIP 配置IP
func (c *vdiClientExtendedImpl) ConfigIP(ctx context.Context, req []ConfigIPRequestExtended) error {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return err
	}

	return c.callAPI(ctx, token, "POST", "/v1/vm/config_ip", req, nil)
}

// RenameVM 重命名虚拟机
func (c *vdiClientExtendedImpl) RenameVM(ctx context.Context, vmID, newName string) error {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return err
	}

	req := struct {
		VMID   string `json:"vm_id"`
		VMName string `json:"vm_name"`
	}{
		VMID:   vmID,
		VMName: newName,
	}

	return c.callAPI(ctx, token, "POST", "/v1/vm/rename", req, nil)
}

// BindUser 绑定用户
func (c *vdiClientExtendedImpl) BindUser(ctx context.Context, vmID, userID string) (*VDIVMDetail, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ErrorCode    int         `json:"error_code"`
		ErrorMessage string      `json:"error_message"`
		Data         VDIVMDetail `json:"data"`
	}

	req := struct {
		VMID   string `json:"vm_id"`
		UserID string `json:"user_id"`
	}{
		VMID:   vmID,
		UserID: userID,
	}

	if err := c.callAPI(ctx, token, "POST", "/v1/vm/bind_user", req, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return &resp.Data, nil
}

// GetAvailableUsers 获取可用用户列表
func (c *vdiClientExtendedImpl) GetAvailableUsers(ctx context.Context, vmID string) ([]VDIUser, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ErrorCode    int       `json:"error_code"`
		ErrorMessage string    `json:"error_message"`
		Data         []VDIUser `json:"data"`
	}

	path := fmt.Sprintf("/v1/vm/%s/users", vmID)
	if err := c.callAPI(ctx, token, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return resp.Data, nil
}

// ListResourceGroups 获取所有资源组
func (c *vdiClientExtendedImpl) ListResourceGroups(ctx context.Context) ([]VDIResourceGroup, error) {
	// 尝试使用缓存的 token（如果有）
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	// 调试日志：检查 token 值
	if token == "" {
		return nil, fmt.Errorf("token is empty after authentication")
	}

	var resp VDIResourceGroupsResponse
	apiErr := c.callAPI(ctx, token, "GET", "/v1/resources_group", nil, &resp)

	// 如果收到 AUTH_TOKEN_INVALID 错误，清除缓存并重试
	if apiErr != nil && strings.Contains(apiErr.Error(), "AUTH_TOKEN_INVALID") {
		fmt.Printf("[VDI API DEBUG] Received AUTH_TOKEN_INVALID, forcing re-authentication\n")

		// 清除内存中的 token 缓存，强制重新认证
		c.authManager.ClearTokenCache()

		// 重新认证
		token, err = c.authManager.Authenticate(ctx)
		if err != nil {
			return nil, fmt.Errorf("re-authentication failed: %w", err)
		}

		// 重试 API 调用
		apiErr = c.callAPI(ctx, token, "GET", "/v1/resources_group", nil, &resp)
		if apiErr != nil {
			return nil, fmt.Errorf("retry after re-auth failed: %w", apiErr)
		}
	} else if apiErr != nil {
		return nil, apiErr
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return resp.Data, nil
}

// ListResources 获取指定资源组下的资源列表
func (c *vdiClientExtendedImpl) ListResources(ctx context.Context, groupID string) ([]VDIResource, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ErrorCode    int          `json:"error_code"`
		ErrorMessage string       `json:"error_message"`
		Data         struct {
			Resources []VDIResource `json:"resources"`
		} `json:"data"`
	}

	path := fmt.Sprintf("/v1/resources/list/%s", groupID)
	if err := c.callAPI(ctx, token, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return resp.Data.Resources, nil
}

// ListResourceServers 获取指定资源下的虚拟机列表
func (c *vdiClientExtendedImpl) ListResourceServers(ctx context.Context, resourceID string, page, pageSize int) ([]VDIVMResource, int, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, 0, err
	}

	var resp VDIResourceServersResponse
	path := fmt.Sprintf("/v1/resource/servers?rcid=%s&page=%d&page_size=%d", resourceID, page, pageSize)

	if err := c.callAPI(ctx, token, "GET", path, nil, &resp); err != nil {
		return nil, 0, err
	}

	if resp.ErrorCode != 0 {
		return nil, 0, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	// 解析总数
	totalCount := 0
	if resp.Data.TotalCount != "" {
		fmt.Sscanf(resp.Data.TotalCount, "%d", &totalCount)
	}

	return resp.Data.Data, totalCount, nil
}

// callAPI 调用VDI API
func (c *vdiClientExtendedImpl) callAPI(ctx context.Context, token, method, path string, body, result interface{}) error {
	url := fmt.Sprintf("%s%s", c.server.Endpoint, path)

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body failed: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth-Token", token) // VDI 系统要求使用 Auth-Token header（不是 Authorization）

	// 调试日志：记录请求详情
	fmt.Printf("[VDI API DEBUG] Request URL: %s\n", url)
	fmt.Printf("[VDI API DEBUG] Auth-Token header: %s\n", req.Header.Get("Auth-Token"))
	fmt.Printf("[VDI API DEBUG] Token length: %d\n", len(token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 读取响应体内容以获取详细错误信息
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response failed: %w", err)
		}
	}

	return nil
}
