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
	ConfigIP(ctx context.Context, req ConfigIPRequestExtended) error
	RenameVM(ctx context.Context, vmID, newName string) error

	// 用户关联
	BindUser(ctx context.Context, vmID, userID string) (*VDIVMDetail, error)
	GetAvailableUsers(ctx context.Context, vmID string) ([]VDIUser, error)

	// 资源管理
	ListResources(ctx context.Context, groupID string) ([]VDIResource, error)
	ListResourceGroups(ctx context.Context) ([]VDIResourceGroup, error)
	ListResourceServers(ctx context.Context, resourceID string, page, pageSize int) ([]VDIVMResource, int, error)

	// VDI 创建虚拟机相关方法
	GetVTPPlatforms(ctx context.Context) ([]VDIPlatform, error)
	GetRunPositions(ctx context.Context, vtpID int) ([]RunPosition, error)
	GetStorages(ctx context.Context, vtpID int) ([]Storage, error)
	GetNetworks(ctx context.Context, vtpID int) ([]Network, error)
	CreateServer(ctx context.Context, req CreateServerRequest) (*CreateServerResponse, error)
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
// VDI API endpoint: GET /v1/servers/detail/:vmid
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

	path := fmt.Sprintf("/v1/servers/detail/%s", vmID)
	if err := c.callAPIWithRetry(ctx, &token, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return &resp.Data, nil
}

// ListVMs 获取虚拟机列表
// VDI API endpoint: GET /v1/servers (支持按 rc_id 过滤)
func (c *vdiClientExtendedImpl) ListVMs(ctx context.Context, resourceID string) ([]VDIVMSummary, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ErrorCode    int            `json:"error_code"`
		ErrorMessage string         `json:"error_message"`
		Data         []VDIVMSummary `json:"data"`
	}

	path := "/v1/servers"
	if resourceID != "" {
		path = fmt.Sprintf("/v1/servers?rc_id=%s", resourceID)
	}

	if err := c.callAPIWithRetry(ctx, &token, "GET", path, nil, &resp); err != nil {
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
		ErrorCode    int            `json:"error_code"`
		ErrorMessage string         `json:"error_message"`
		Data         []VDIVMSummary `json:"data"`
	}

	path := fmt.Sprintf("/api/v1/user/%s/vm", userID)
	if err := c.callAPIWithRetry(ctx, &token, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return resp.Data, nil
}

// operateActionToName maps our internal action names to VDI API action names
// Based on VDI API documentation: startup, shutdown, reboot, suspend, resume
func operateActionToName(action string) (string, error) {
	switch action {
	case "start":
		return "startup", nil
	case "stop":
		return "shutdown", nil
	case "restart":
		return "reboot", nil
	case "suspend":
		return "suspend", nil
	default:
		return "", fmt.Errorf("unsupported VDI VM action: %s", action)
	}
}

// OperateVM 操作虚拟机
// VDI API endpoint: POST /v1/servers/action
// Request body format: {"action": {"name": "startup/shutdown/reboot/suspend"}, "servers": {"ids": "vm_id"}}
func (c *vdiClientExtendedImpl) OperateVM(ctx context.Context, vmIDs []string, action string) error {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return err
	}

	actionName, err := operateActionToName(action)
	if err != nil {
		return err
	}

	// VDI API supports batch operations, but IDs should be comma-separated
	var idsStr string
	for i, vmID := range vmIDs {
		if i > 0 {
			idsStr += ","
		}
		idsStr += vmID
	}

	// 使用 map 匹配工作脚本的请求格式和字段顺序
	req := map[string]interface{}{
		"action": map[string]string{
			"name": actionName,
		},
		"servers": map[string]string{
			"ids": idsStr,
		},
	}

	// Debug log the request body
	reqJSON, _ := json.Marshal(req)
	fmt.Printf("[VDI API DEBUG] Request body: %s\n", string(reqJSON))

	return c.callAPIWithRetry(ctx, &token, "POST", "/v1/servers/action", req, nil)
}

// DeleteVM 删除虚拟机
// VDI API endpoint: DELETE /v1/servers
// Request body format: {"ids": "1,2"} - VM IDs separated by commas
func (c *vdiClientExtendedImpl) DeleteVM(ctx context.Context, vmIDs []string) error {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return err
	}

	// Build comma-separated IDs string
	var idsStr string
	for i, vmID := range vmIDs {
		if i > 0 {
			idsStr += ","
		}
		idsStr += vmID
	}

	req := struct {
		IDs string `json:"ids"`
	}{
		IDs: idsStr,
	}

	return c.callAPIWithRetry(ctx, &token, "DELETE", "/v1/servers", req, nil)
}

// ConfigIP 配置IP
// VDI API endpoint: PUT /v1/servers/ip
// Request body format matches VDI API specification
func (c *vdiClientExtendedImpl) ConfigIP(ctx context.Context, req ConfigIPRequestExtended) error {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return err
	}

	return c.callAPIWithRetry(ctx, &token, "PUT", "/v1/servers/ip", req, nil)
}

// RenameVM 重命名虚拟机
// VDI API endpoint: PUT /v1/servers/update_name
// Request body: {"id": vm_id, "name_mode": 2, "name": "new_name", "is_reboot": 0}
func (c *vdiClientExtendedImpl) RenameVM(ctx context.Context, vmID, newName string) error {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return err
	}

	req := struct {
		ID       int    `json:"id"`
		NameMode int    `json:"name_mode"`
		Name     string `json:"name"`
		IsReboot int    `json:"is_reboot"`
	}{
		NameMode: 2, // 2 = 管理员指定名称
		Name:     newName,
		IsReboot: 0,
	}
	// 将 vmID 转换为整数
	fmt.Sscanf(vmID, "%d", &req.ID)

	return c.callAPIWithRetry(ctx, &token, "PUT", "/v1/servers/update_name", req, nil)
}

// BindUser 绑定用户
// VDI API endpoint: PUT /v1/servers/bind_users
// Request body: {"rcid": resource_id, "vmid": vm_id, "type": "1/2/3", "user_id": user_id or "user_name": name}
// type: 1=绑定, 2=解绑, 3=更换
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

	// 解析 vmID 获取 vmid
	var vmidInt int
	fmt.Sscanf(vmID, "%d", &vmidInt)

	// 如果需要 rcid，应该从 VM 信息中获取，这里暂时使用 vmID
	req := struct {
		RcID    int    `json:"rcid"`
		VmID    int    `json:"vmid"`
		Type    string `json:"type"`
		UserID  string `json:"user_id"`
	}{
		Type:   "1", // 1=绑定
		UserID: userID,
		VmID:   vmidInt,
	}

	if err := c.callAPIWithRetry(ctx, &token, "PUT", "/v1/servers/bind_users", req, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return &resp.Data, nil
}

// GetAvailableUsers 获取可用用户列表
// 注意：VDI API 中没有直接获取"可用用户"的端点
// GET /v1/bind_users/servers?user_name=xxx 返回指定用户绑定的虚拟机列表
// 这里返回空列表，实际使用中可能需要调用用户管理相关的 API
func (c *vdiClientExtendedImpl) GetAvailableUsers(ctx context.Context, vmID string) ([]VDIUser, error) {
	// VDI API 中没有直接获取"可用用户"的端点
	// 这里返回空列表，实际使用中可能需要调用用户管理相关的 API
	return []VDIUser{}, nil
}

// ListResourceGroups 获取所有资源组
// VDI API endpoint: GET /v1/resources_group
func (c *vdiClientExtendedImpl) ListResourceGroups(ctx context.Context) ([]VDIResourceGroup, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	if token == "" {
		return nil, fmt.Errorf("token is empty after authentication")
	}

	var resp VDIResourceGroupsResponse
	if err := c.callAPIWithRetry(ctx, &token, "GET", "/v1/resources_group", nil, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return resp.Data, nil
}

// ListResources 获取指定资源组下的资源列表
// VDI API endpoint: GET /v1/resources/list/:group_id
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
	if err := c.callAPIWithRetry(ctx, &token, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return resp.Data.Resources, nil
}

// ListResourceServers 获取指定资源下的虚拟机列表
// VDI API endpoint: GET /v1/resource/servers?rcid=xxx&page=x&page_size=x
func (c *vdiClientExtendedImpl) ListResourceServers(ctx context.Context, resourceID string, page, pageSize int) ([]VDIVMResource, int, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, 0, err
	}

	var resp VDIResourceServersResponse
	path := fmt.Sprintf("/v1/resource/servers?rcid=%s&page=%d&page_size=%d", resourceID, page, pageSize)

	if err := c.callAPIWithRetry(ctx, &token, "GET", path, nil, &resp); err != nil {
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

// callAPIWithRetry 调用VDI API，遇到AUTH_TOKEN_INVALID时自动清除缓存并重试一次
// token 参数为指针类型，以便重试时更新token值
func (c *vdiClientExtendedImpl) callAPIWithRetry(ctx context.Context, token *string, method, path string, body, result interface{}) error {
	err := c.callAPI(ctx, *token, method, path, body, result)
	if err != nil && strings.Contains(err.Error(), "AUTH_TOKEN_INVALID") {
		fmt.Printf("[VDI API DEBUG] Received AUTH_TOKEN_INVALID, forcing re-authentication\n")

		// 清除内存中的 token 缓存，强制重新认证（如果 authManager 存在）
		if c.authManager != nil {
			c.authManager.ClearTokenCache()
		}

		// 重新认证
		newToken, authErr := c.Authenticate(ctx)
		if authErr != nil {
			return fmt.Errorf("re-authentication failed after AUTH_TOKEN_INVALID: %w", authErr)
		}

		// 更新调用者的token引用
		*token = newToken

		// 重试 API 调用
		return c.callAPI(ctx, newToken, method, path, body, result)
	}
	return err
}

// callAPI 调用VDI API
func (c *vdiClientExtendedImpl) callAPI(ctx context.Context, token, method, path string, body, result interface{}) error {
	url := fmt.Sprintf("%s%s", c.server.Endpoint, path)
	fmt.Printf("[VDI API DEBUG] Calling: %s %s\n", method, url)

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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body failed: %w", err)
	}

	// 先尝试解析VDI错误响应格式（获取error_code）
	var vdiErr struct {
		ErrorCode    int    `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	json.Unmarshal(bodyBytes, &vdiErr)

	// 检查 AUTH_TOKEN_INVALID 错误
	if vdiErr.ErrorCode == 1101 {
		fmt.Printf("[VDI API DEBUG] AUTH_TOKEN_INVALID: %s\n", vdiErr.ErrorMessage)
		return fmt.Errorf("AUTH_TOKEN_INVALID: %s", vdiErr.ErrorMessage)
	}

	// VDI API 可能返回非200状态码但 error_code==0（表示成功）
	// 只有当 error_code != 0 时才返回错误
	if vdiErr.ErrorCode != 0 {
		fmt.Printf("[VDI API DEBUG] Error response: status=%d, error_code=%d, body=%s\n", resp.StatusCode, vdiErr.ErrorCode, string(bodyBytes))
		return fmt.Errorf("VDI API error %d: %s", vdiErr.ErrorCode, vdiErr.ErrorMessage)
	}

	// error_code == 0，解析成功响应
	if result != nil {
		if err := json.Unmarshal(bodyBytes, result); err != nil {
			return fmt.Errorf("decode response failed: %w", err)
		}
	}

	return nil
}

// GetVTPPlatforms 获取VTP平台列表
// 通过遍历所有资源组和资源，从虚拟机列表中提取唯一的 VTP 平台
func (c *vdiClientExtendedImpl) GetVTPPlatforms(ctx context.Context) ([]VDIPlatform, error) {
	// 使用 map 收集唯一的 VTP 平台
	vtpMap := make(map[int]VDIPlatform)

	// 获取所有资源组
	groups, err := c.ListResourceGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取资源组失败: %w", err)
	}

	// 遍历每个资源组
	for _, group := range groups {
		// 获取该资源组下的资源列表
		resources, err := c.ListResources(ctx, group.ID)
		if err != nil {
			continue // 跳过失败的资源组
		}

		// 遍历每个资源
		for _, resource := range resources {
			// 获取该资源下的虚拟机列表（获取第一页，最多100个虚拟机）
			resourceID := fmt.Sprintf("%d", resource.ID)
			vms, _, err := c.ListResourceServers(ctx, resourceID, 1, 100)
			if err != nil {
				continue // 跳过失败的资源
			}

			// 从虚拟机列表中提取 VTP 平台
			for _, vm := range vms {
				if vm.VTPID != "" && vm.VTPName != "" {
					var vtpID int
					fmt.Sscanf(vm.VTPID, "%d", &vtpID)
					if vtpID > 0 {
						vtpMap[vtpID] = VDIPlatform{
							ID:   vtpID,
							Name: vm.VTPName,
						}
					}
				}
			}
		}
	}

	// 转换为切片
	result := make([]VDIPlatform, 0, len(vtpMap))
	for _, vtp := range vtpMap {
		result = append(result, vtp)
	}

	return result, nil
}

// GetRunPositions 获取运行位置列表
// VDI API endpoint: GET /v1/run_position?vtp_id={vtpID}
func (c *vdiClientExtendedImpl) GetRunPositions(ctx context.Context, vtpID int) ([]RunPosition, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	var resp RunPositionResponse
	path := fmt.Sprintf("/v1/run_position?vtp_id=%d", vtpID)
	if err := c.callAPIWithRetry(ctx, &token, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return resp.Run, nil
}

// GetStorages 获取存储位置列表
// VDI API endpoint: GET /v1/storages?vtp_id={vtpID}
func (c *vdiClientExtendedImpl) GetStorages(ctx context.Context, vtpID int) ([]Storage, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	var resp StorageResponse
	path := fmt.Sprintf("/v1/storages?vtp_id=%d", vtpID)
	if err := c.callAPIWithRetry(ctx, &token, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return resp.Storages, nil
}

// GetNetworks 获取网络接口列表
// VDI API endpoint: GET /v1/networks?vtp_id={vtpID}
func (c *vdiClientExtendedImpl) GetNetworks(ctx context.Context, vtpID int) ([]Network, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	var resp NetworkResponse
	path := fmt.Sprintf("/v1/networks?vtp_id=%d", vtpID)
	if err := c.callAPIWithRetry(ctx, &token, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return resp.Networks, nil
}

// CreateServer 创建VDI虚拟机
// VDI API endpoint: POST /v1/servers
// Implements host/run position logic from vdi_test_standalone.go lines 543-554
func (c *vdiClientExtendedImpl) CreateServer(ctx context.Context, req CreateServerRequest) (*CreateServerResponse, error) {
	token, err := c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	var resp CreateServerResponse
	if err := c.callAPIWithRetry(ctx, &token, "POST", "/v1/servers", req, &resp); err != nil {
		return nil, err
	}

	if resp.ErrorCode != 0 {
		return nil, &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	return &resp, nil
}
