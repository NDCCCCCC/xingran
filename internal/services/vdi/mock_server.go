package vdi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

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
	m.server.Close()
}

// GetCallCount 获取 API 调用次数
func (m *VDIAPIMock) GetCallCount(endpoint string) int {
	m.callCountMu.Lock()
	defer m.callCountMu.Unlock()
	return m.callCount[endpoint]
}

// AddVM 添加模拟虚拟机
func (m *VDIAPIMock) AddVM(vm *VMInfo) {
	m.vmsMu.Lock()
	defer m.vmsMu.Unlock()
	m.vms[vm.VMID] = vm
}

// handleAuth 处理认证请求
func (m *VDIAPIMock) handleAuth(w http.ResponseWriter, r *http.Request) {
	m.incrementCallCount("auth")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp := AuthResponse{
		Code:    0,
		Message: "success",
		Token:   m.authToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleCreateVM 处理创建虚拟机请求
func (m *VDIAPIMock) handleCreateVM(w http.ResponseWriter, r *http.Request) {
	m.incrementCallCount("create_vm")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req CreateVMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	vm := &VMInfo{
		VMID:        fmt.Sprintf("vm-%d", time.Now().UnixNano()),
		VMName:      req.VMName,
		IP:          req.IP,
		MAC:         req.MAC,
		Status:      "stopped",
		VolumeID:    req.VolumeID,
		ImageID:     req.ImageID,
		DesktopGroup: req.DesktopGroup,
		Comments:    req.Comments,
	}

	m.AddVM(vm)

	resp := CreateVMResponse{
		Code:    0,
		Message: "success",
		VMID:    vm.VMID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDeleteVM 处理删除虚拟机请求
func (m *VDIAPIMock) handleDeleteVM(w http.ResponseWriter, r *http.Request) {
	m.incrementCallCount("delete_vm")
	m.handleVMOperation(w, r, "delete")
}

// handleStartVM 处理启动虚拟机请求
func (m *VDIAPIMock) handleStartVM(w http.ResponseWriter, r *http.Request) {
	m.incrementCallCount("start_vm")
	m.handleVMOperation(w, r, "start")
}

// handleStopVM 处理停止虚拟机请求
func (m *VDIAPIMock) handleStopVM(w http.ResponseWriter, r *http.Request) {
	m.incrementCallCount("stop_vm")
	m.handleVMOperation(w, r, "stop")
}

// handleRestartVM 处理重启虚拟机请求
func (m *VDIAPIMock) handleRestartVM(w http.ResponseWriter, r *http.Request) {
	m.incrementCallCount("restart_vm")
	m.handleVMOperation(w, r, "restart")
}

// handleConfigIP 处理配置IP请求
func (m *VDIAPIMock) handleConfigIP(w http.ResponseWriter, r *http.Request) {
	m.incrementCallCount("config_ip")
	m.handleVMOperation(w, r, "config_ip")
}

// handleRename 处理重命名请求
func (m *VDIAPIMock) handleRename(w http.ResponseWriter, r *http.Request) {
	m.incrementCallCount("rename")
	m.handleVMOperation(w, r, "rename")
}

// handleBindUser 处理绑定用户请求
func (m *VDIAPIMock) handleBindUser(w http.ResponseWriter, r *http.Request) {
	m.incrementCallCount("bind_user")
	m.handleVMOperation(w, r, "bind_user")
}

// handleUnbindUser 处理解绑用户请求
func (m *VDIAPIMock) handleUnbindUser(w http.ResponseWriter, r *http.Request) {
	m.incrementCallCount("unbind_user")
	m.handleVMOperation(w, r, "unbind_user")
}

// handleListVMs 处理获取虚拟机列表请求
func (m *VDIAPIMock) handleListVMs(w http.ResponseWriter, r *http.Request) {
	m.incrementCallCount("list_vms")

	m.vmsMu.RLock()
	defer m.vmsMu.RUnlock()

	vms := make([]VMInfo, 0, len(m.vms))
	for _, vm := range m.vms {
		vms = append(vms, *vm)
	}

	resp := VMListResponse{
		Code:    0,
		Message: "success",
		Data:    vms,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleGetVMInfo 处理获取虚拟机详情请求
func (m *VDIAPIMock) handleGetVMInfo(w http.ResponseWriter, r *http.Request) {
	m.incrementCallCount("get_vm_info")

	var req VMOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	m.vmsMu.RLock()
	defer m.vmsMu.RUnlock()

	vm, exists := m.vms[req.VMID]
	if !exists {
		resp := VMInfoResponse{
			Code:    1,
			Message: "VM not found",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := VMInfoResponse{
		Code:    0,
		Message: "success",
		Data:    *vm,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleVMOperation 通用虚拟机操作处理
func (m *VDIAPIMock) handleVMOperation(w http.ResponseWriter, r *http.Request, operation string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req VMOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	m.vmsMu.Lock()
	defer m.vmsMu.Unlock()

	vm, exists := m.vms[req.VMID]
	if !exists {
		resp := VDIResponse{
			Code:    1,
			Message: "VM not found",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// 根据操作类型更新虚拟机状态
	switch operation {
	case "start":
		vm.Status = "running"
	case "stop":
		vm.Status = "stopped"
	case "restart":
		vm.Status = "running"
	case "delete":
		delete(m.vms, req.VMID)
	}

	resp := VDIResponse{
		Code:    0,
		Message: "success",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// incrementCallCount 增加API调用计数
func (m *VDIAPIMock) incrementCallCount(endpoint string) {
	m.callCountMu.Lock()
	defer m.callCountMu.Unlock()
	m.callCount[endpoint]++
}
