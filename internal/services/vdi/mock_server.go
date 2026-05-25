package vdi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// CommonResponse 通用响应结构
type CommonResponse struct {
	Code    int    `json:"error_code"`
	Message string `json:"error_message"`
}

// VDIAPIMock 模拟深信服 VDI API 服务器
type VDIAPIMock struct {
	server      *httptest.Server
	authToken   string
	authExpiry  time.Time
	callCount   map[string]int
	callCountMu sync.Mutex
	// 模拟数据库
	vms         map[string]*VMInfo
	vmsMu       sync.RWMutex
}

// AuthRequest 认证请求（mock 专用）
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// NewVDIAPIMock 创建新的 VDI API Mock 服务器
func NewVDIAPIMock() *VDIAPIMock {
	mock := &VDIAPIMock{
		authToken:  "mock-token-" + time.Now().Format("20060102150405"),
		authExpiry: time.Now().Add(24 * time.Hour),
		callCount:  make(map[string]int),
		vms:        make(map[string]*VMInfo),
	}

	// 启动测试服务器
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", mock.handleAuth)
	mux.HandleFunc("/api/v1/vm/create", mock.handleCreateVM)
	mux.HandleFunc("/api/v1/vm/delete", mock.handleDeleteVM)
	mux.HandleFunc("/api/v1/vm/start", mock.handleStartVM)
	mux.HandleFunc("/api/v1/vm/stop", mock.handleStopVM)
	mux.HandleFunc("/api/v1/vm/restart", mock.handleRestartVM)
	mux.HandleFunc("/api/v1/vm/config_ip", mock.handleConfigIP)
	mux.HandleFunc("/api/v1/vm/rename", mock.handleRename)
	mux.HandleFunc("/api/v1/vm/bind_user", mock.handleBindUser)
	mux.HandleFunc("/api/v1/vm/unbind_user", mock.handleUnbindUser)
	mux.HandleFunc("/api/v1/vm/list", mock.handleListVMs)
	mux.HandleFunc("/api/v1/vm/get_info", mock.handleGetVMInfo)

	mock.server = httptest.NewServer(mux)

	return mock
}

// GetServerURL 获取 Mock 服务器 URL
func (m *VDIAPIMock) GetServerURL() string {
	return m.server.URL
}

// Close 关闭 Mock 服务器
func (m *VDIAPIMock) Close() {
	if m.server != nil {
		m.server.Close()
	}
}

// GetCallCount 获取 API 调用次数
func (m *VDIAPIMock) GetCallCount(endpoint string) int {
	m.callCountMu.Lock()
	defer m.callCountMu.Unlock()
	return m.callCount[endpoint]
}

// ResetCallCount 重置调用计数
func (m *VDIAPIMock) ResetCallCount() {
	m.callCountMu.Lock()
	defer m.callCountMu.Unlock()
	m.callCount = make(map[string]int)
}

// AddVM 添加测试虚拟机
func (m *VDIAPIMock) AddVM(vm *VMInfo) {
	m.vmsMu.Lock()
	defer m.vmsMu.Unlock()
	if vm.CreatedAt.IsZero() {
		vm.CreatedAt = time.Now()
	}
	vm.UpdatedAt = time.Now()
	m.vms[vm.VMID] = vm
}

// GetVM 获取虚拟机信息
func (m *VDIAPIMock) GetVM(vmID string) *VMInfo {
	m.vmsMu.RLock()
	defer m.vmsMu.RUnlock()
	return m.vms[vmID]
}

// ClearVMs 清空所有虚拟机
func (m *VDIAPIMock) ClearVMs() {
	m.vmsMu.Lock()
	defer m.vmsMu.Unlock()
	m.vms = make(map[string]*VMInfo)
}

// SetTokenExpired 设置 Token 为已过期
func (m *VDIAPIMock) SetTokenExpired() {
	m.authExpiry = time.Now().Add(-1 * time.Hour)
}

// checkAuth 检查认证
func (m *VDIAPIMock) checkAuth(r *http.Request) bool {
	token := r.Header.Get("Authorization")
	if token == "" {
		return false
	}
	// 移除 "Bearer " 前缀
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	return token == m.authToken && time.Now().Before(m.authExpiry)
}

// recordCall 记录 API 调用
func (m *VDIAPIMock) recordCall(endpoint string) {
	m.callCountMu.Lock()
	defer m.callCountMu.Unlock()
	m.callCount[endpoint]++
}

// sendJSON 发送 JSON 响应
func (m *VDIAPIMock) sendJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

// handleAuth 处理认证请求
func (m *VDIAPIMock) handleAuth(w http.ResponseWriter, r *http.Request) {
	m.recordCall("/api/v1/auth/login")

	if r.Method != http.MethodPost {
		m.sendJSON(w, http.StatusMethodNotAllowed, CommonResponse{
			Code:    405,
			Message: "Method Not Allowed",
		})
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.sendJSON(w, http.StatusBadRequest, CommonResponse{
			Code:    400,
			Message: "Invalid request body",
		})
		return
	}

	// 模拟认证逻辑
	if req.Username == "error" {
		m.sendJSON(w, http.StatusUnauthorized, AuthResponse{
			Code:    401,
			Message: "Authentication failed",
		})
		return
	}

	m.sendJSON(w, http.StatusOK, AuthResponse{
		Code:    0,
		Message: "Success",
		Token:   m.authToken,
	})
}

// handleCreateVM 处理创建虚拟机请求
func (m *VDIAPIMock) handleCreateVM(w http.ResponseWriter, r *http.Request) {
	m.recordCall("/api/v1/vm/create")

	if !m.checkAuth(r) {
		m.sendJSON(w, http.StatusUnauthorized, CommonResponse{
			Code:    401,
			Message: "Unauthorized",
		})
		return
	}

	if r.Method != http.MethodPost {
		m.sendJSON(w, http.StatusMethodNotAllowed, CommonResponse{
			Code:    405,
			Message: "Method Not Allowed",
		})
		return
	}

	var req CreateVMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.sendJSON(w, http.StatusBadRequest, CommonResponse{
			Code:    400,
			Message: "Invalid request body",
		})
		return
	}

	// 检查 VM 名称是否已存在
	m.vmsMu.RLock()
	for _, vm := range m.vms {
		if vm.VMName == req.VMName {
			m.vmsMu.RUnlock()
			m.sendJSON(w, http.StatusConflict, CreateVMResponse{
				Code:    409,
				Message: "VM name already exists",
			})
			return
		}
	}
	m.vmsMu.RUnlock()

	// 创建新虚拟机
	vmID := fmt.Sprintf("vm-%d", time.Now().UnixNano())
	vm := &VMInfo{
		VMID:       vmID,
		VMName:     req.VMName,
		IP:         req.IP,
		MAC:        req.MAC,
		Username:   req.Username,
		Password:   req.Password,
		VolumeID:   req.VolumeID,
		ImageID:    req.ImageID,
		DesktopGroup: req.DesktopGroup,
		Comments:   req.Comments,
		Status:     "STOPPED",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	m.vmsMu.Lock()
	m.vms[vmID] = vm
	m.vmsMu.Unlock()

	m.sendJSON(w, http.StatusOK, CreateVMResponse{
		Code:    0,
		Message: "Success",
		VMID:    vmID,
	})
}

// handleDeleteVM 处理删除虚拟机请求
func (m *VDIAPIMock) handleDeleteVM(w http.ResponseWriter, r *http.Request) {
	m.recordCall("/api/v1/vm/delete")

	if !m.checkAuth(r) {
		m.sendJSON(w, http.StatusUnauthorized, CommonResponse{
			Code:    401,
			Message: "Unauthorized",
		})
		return
	}

	var req struct {
		VMID string `json:"vM_ID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.sendJSON(w, http.StatusBadRequest, CommonResponse{
			Code:    400,
			Message: "Invalid request body",
		})
		return
	}

	m.vmsMu.Lock()
	defer m.vmsMu.Unlock()

	if _, exists := m.vms[req.VMID]; !exists {
		m.sendJSON(w, http.StatusNotFound, CommonResponse{
			Code:    404,
			Message: "VM not found",
		})
		return
	}

	delete(m.vms, req.VMID)
	m.sendJSON(w, http.StatusOK, CommonResponse{
		Code:    0,
		Message: "Success",
	})
}

// handleStartVM 处理启动虚拟机请求
func (m *VDIAPIMock) handleStartVM(w http.ResponseWriter, r *http.Request) {
	m.recordCall("/api/v1/vm/start")

	if !m.checkAuth(r) {
		m.sendJSON(w, http.StatusUnauthorized, CommonResponse{
			Code:    401,
			Message: "Unauthorized",
		})
		return
	}

	var req struct {
		VMID string `json:"vM_ID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.sendJSON(w, http.StatusBadRequest, CommonResponse{
			Code:    400,
			Message: "Invalid request body",
		})
		return
	}

	m.vmsMu.Lock()
	defer m.vmsMu.Unlock()

	vm, exists := m.vms[req.VMID]
	if !exists {
		m.sendJSON(w, http.StatusNotFound, CommonResponse{
			Code:    404,
			Message: "VM not found",
		})
		return
	}

	if vm.Status == "RUNNING" {
		m.sendJSON(w, http.StatusConflict, CommonResponse{
			Code:    409,
			Message: "VM is already running",
		})
		return
	}

	vm.Status = "RUNNING"
	vm.UpdatedAt = time.Now()

	m.sendJSON(w, http.StatusOK, CommonResponse{
		Code:    0,
		Message: "Success",
	})
}

// handleStopVM 处理停止虚拟机请求
func (m *VDIAPIMock) handleStopVM(w http.ResponseWriter, r *http.Request) {
	m.recordCall("/api/v1/vm/stop")

	if !m.checkAuth(r) {
		m.sendJSON(w, http.StatusUnauthorized, CommonResponse{
			Code:    401,
			Message: "Unauthorized",
		})
		return
	}

	var req struct {
		VMID string `json:"vM_ID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.sendJSON(w, http.StatusBadRequest, CommonResponse{
			Code:    400,
			Message: "Invalid request body",
		})
		return
	}

	m.vmsMu.Lock()
	defer m.vmsMu.Unlock()

	vm, exists := m.vms[req.VMID]
	if !exists {
		m.sendJSON(w, http.StatusNotFound, CommonResponse{
			Code:    404,
			Message: "VM not found",
		})
		return
	}

	if vm.Status == "STOPPED" {
		m.sendJSON(w, http.StatusConflict, CommonResponse{
			Code:    409,
			Message: "VM is already stopped",
		})
		return
	}

	vm.Status = "STOPPED"
	vm.UpdatedAt = time.Now()

	m.sendJSON(w, http.StatusOK, CommonResponse{
		Code:    0,
		Message: "Success",
	})
}

// handleRestartVM 处理重启虚拟机请求
func (m *VDIAPIMock) handleRestartVM(w http.ResponseWriter, r *http.Request) {
	m.recordCall("/api/v1/vm/restart")

	if !m.checkAuth(r) {
		m.sendJSON(w, http.StatusUnauthorized, CommonResponse{
			Code:    401,
			Message: "Unauthorized",
		})
		return
	}

	var req struct {
		VMID string `json:"vM_ID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.sendJSON(w, http.StatusBadRequest, CommonResponse{
			Code:    400,
			Message: "Invalid request body",
		})
		return
	}

	m.vmsMu.Lock()
	defer m.vmsMu.Unlock()

	vm, exists := m.vms[req.VMID]
	if !exists {
		m.sendJSON(w, http.StatusNotFound, CommonResponse{
			Code:    404,
			Message: "VM not found",
		})
		return
	}

	vm.Status = "RUNNING"
	vm.UpdatedAt = time.Now()

	m.sendJSON(w, http.StatusOK, CommonResponse{
		Code:    0,
		Message: "Success",
	})
}

// handleConfigIP 处理配置 IP 请求
func (m *VDIAPIMock) handleConfigIP(w http.ResponseWriter, r *http.Request) {
	m.recordCall("/api/v1/vm/config_ip")

	if !m.checkAuth(r) {
		m.sendJSON(w, http.StatusUnauthorized, CommonResponse{
			Code:    401,
			Message: "Unauthorized",
		})
		return
	}

	var req struct {
		VMID string `json:"vM_ID"`
		IP   string `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.sendJSON(w, http.StatusBadRequest, CommonResponse{
			Code:    400,
			Message: "Invalid request body",
		})
		return
	}

	m.vmsMu.Lock()
	defer m.vmsMu.Unlock()

	vm, exists := m.vms[req.VMID]
	if !exists {
		m.sendJSON(w, http.StatusNotFound, CommonResponse{
			Code:    404,
			Message: "VM not found",
		})
		return
	}

	vm.IP = req.IP
	vm.UpdatedAt = time.Now()

	m.sendJSON(w, http.StatusOK, CommonResponse{
		Code:    0,
		Message: "Success",
	})
}

// handleRename 处理重命名虚拟机请求
func (m *VDIAPIMock) handleRename(w http.ResponseWriter, r *http.Request) {
	m.recordCall("/api/v1/vm/rename")

	if !m.checkAuth(r) {
		m.sendJSON(w, http.StatusUnauthorized, CommonResponse{
			Code:    401,
			Message: "Unauthorized",
		})
		return
	}

	var req struct {
		VMID    string `json:"vM_ID"`
		VMName  string `json:"vM_NAME"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.sendJSON(w, http.StatusBadRequest, CommonResponse{
			Code:    400,
			Message: "Invalid request body",
		})
		return
	}

	m.vmsMu.Lock()
	defer m.vmsMu.Unlock()

	vm, exists := m.vms[req.VMID]
	if !exists {
		m.sendJSON(w, http.StatusNotFound, CommonResponse{
			Code:    404,
			Message: "VM not found",
		})
		return
	}

	vm.VMName = req.VMName
	vm.UpdatedAt = time.Now()

	m.sendJSON(w, http.StatusOK, CommonResponse{
		Code:    0,
		Message: "Success",
	})
}

// handleBindUser 处理绑定用户请求
func (m *VDIAPIMock) handleBindUser(w http.ResponseWriter, r *http.Request) {
	m.recordCall("/api/v1/vm/bind_user")

	if !m.checkAuth(r) {
		m.sendJSON(w, http.StatusUnauthorized, CommonResponse{
			Code:    401,
			Message: "Unauthorized",
		})
		return
	}

	var req struct {
		VMID     string `json:"vM_ID"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.sendJSON(w, http.StatusBadRequest, CommonResponse{
			Code:    400,
			Message: "Invalid request body",
		})
		return
	}

	m.vmsMu.Lock()
	defer m.vmsMu.Unlock()

	vm, exists := m.vms[req.VMID]
	if !exists {
		m.sendJSON(w, http.StatusNotFound, CommonResponse{
			Code:    404,
			Message: "VM not found",
		})
		return
	}

	vm.Username = req.Username
	vm.Password = req.Password
	vm.UpdatedAt = time.Now()

	m.sendJSON(w, http.StatusOK, CommonResponse{
		Code:    0,
		Message: "Success",
	})
}

// handleUnbindUser 处理解绑用户请求
func (m *VDIAPIMock) handleUnbindUser(w http.ResponseWriter, r *http.Request) {
	m.recordCall("/api/v1/vm/unbind_user")

	if !m.checkAuth(r) {
		m.sendJSON(w, http.StatusUnauthorized, CommonResponse{
			Code:    401,
			Message: "Unauthorized",
		})
		return
	}

	var req struct {
		VMID string `json:"vM_ID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.sendJSON(w, http.StatusBadRequest, CommonResponse{
			Code:    400,
			Message: "Invalid request body",
		})
		return
	}

	m.vmsMu.Lock()
	defer m.vmsMu.Unlock()

	vm, exists := m.vms[req.VMID]
	if !exists {
		m.sendJSON(w, http.StatusNotFound, CommonResponse{
			Code:    404,
			Message: "VM not found",
		})
		return
	}

	vm.Username = ""
	vm.Password = ""
	vm.UpdatedAt = time.Now()

	m.sendJSON(w, http.StatusOK, CommonResponse{
		Code:    0,
		Message: "Success",
	})
}

// handleListVMs 处理获取虚拟机列表请求
func (m *VDIAPIMock) handleListVMs(w http.ResponseWriter, r *http.Request) {
	m.recordCall("/api/v1/vm/list")

	if !m.checkAuth(r) {
		m.sendJSON(w, http.StatusUnauthorized, CommonResponse{
			Code:    401,
			Message: "Unauthorized",
		})
		return
	}

	m.vmsMu.RLock()
	defer m.vmsMu.RUnlock()

	vms := make([]*VMInfo, 0, len(m.vms))
	for _, vm := range m.vms {
		vms = append(vms, vm)
	}

	response := struct {
		Code    int      `json:"code"`
		Message string   `json:"message"`
		Data    []*VMInfo `json:"data"`
	}{
		Code:    0,
		Message: "Success",
		Data:    vms,
	}

	m.sendJSON(w, http.StatusOK, response)
}

// handleGetVMInfo 处理获取虚拟机详情请求
func (m *VDIAPIMock) handleGetVMInfo(w http.ResponseWriter, r *http.Request) {
	m.recordCall("/api/v1/vm/get_info")

	if !m.checkAuth(r) {
		m.sendJSON(w, http.StatusUnauthorized, CommonResponse{
			Code:    401,
			Message: "Unauthorized",
		})
		return
	}

	var req struct {
		VMID string `json:"vM_ID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.sendJSON(w, http.StatusBadRequest, CommonResponse{
			Code:    400,
			Message: "Invalid request body",
		})
		return
	}

	m.vmsMu.RLock()
	defer m.vmsMu.RUnlock()

	vm, exists := m.vms[req.VMID]
	if !exists {
		m.sendJSON(w, http.StatusNotFound, CommonResponse{
			Code:    404,
			Message: "VM not found",
		})
		return
	}

	response := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    *VMInfo `json:"data"`
	}{
		Code:    0,
		Message: "Success",
		Data:    vm,
	}

	m.sendJSON(w, http.StatusOK, response)
}
