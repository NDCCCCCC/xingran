package agent

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"gorm.io/gorm"
)

// AgentHandler Agent 注册处理器
type AgentHandler struct {
	db   *gorm.DB
	core *core.Core
}

// NewAgentHandler 创建 Agent 处理器
func NewAgentHandler(db *gorm.DB) *AgentHandler {
	return &AgentHandler{db: db}
}

// WithCore 注入 core 依赖（用于操作日志埋点），链式调用
func (h *AgentHandler) WithCore(core *core.Core) *AgentHandler {
	h.core = core
	return h
}

// AgentRegisterRequest Agent 注册请求
type AgentRegisterRequest struct {
	Hostname    string `json:"hostname" binding:"required"`
	IPAddress   string `json:"ip_address"`
	MACAddress  string `json:"mac_address"`
	OSType      string `json:"os_type"`
	Platform    string `json:"platform"`
	MachineGUID string `json:"machine_guid,omitempty"`
}

// RegisterAgentResponse Agent 注册响应
type AgentRegisterResponse struct {
	VMID    string `json:"vm_id"`
	AgentID string `json:"agent_id"`
	Matched bool   `json:"matched"`
	Message string `json:"message,omitempty"`
}

// RegisterAgent Agent 自动注册
// POST /api/agent/register
func (h *AgentHandler) RegisterAgent(c *gin.Context) {
	var req AgentRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	// 通过指纹匹配虚拟机
	vmID, matched, err := h.matchVMByFingerprint(&req)
	if err != nil {
		// 匹配失败，返回错误但不阻止继续（使用临时 ID）
		response.Error(c, http.StatusNotFound, fmt.Sprintf("无法匹配虚拟机: %v", err))
		return
	}

	// 生成 Agent ID
	agentID := fmt.Sprintf("agent-%s-%s", req.Hostname, generateRandomString(8))

	// RecordWithBody 屏蔽请求体中可能包含的 agent_key / token / secret 等敏感字段
	// （T-34-W6-03 缓解）。当前 AgentRegisterRequest 未含 agent_key，但保持
	// 屏蔽调用以便未来扩展字段时不漏掩。
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "Agent注册", operlog.OperTypeRegister)

	response.Success(c, AgentRegisterResponse{
		VMID:    vmID,
		AgentID: agentID,
		Matched: matched,
		Message: "自动注册成功",
	})
}

// matchVMByFingerprint 通过指纹匹配虚拟机
// 优先级: IP > MAC > 主机名
func (h *AgentHandler) matchVMByFingerprint(req *AgentRegisterRequest) (vmID string, matched bool, err error) {
	var vm models.VDIVirtualMachine

	// 优先级 1: 通过 IP 地址精确匹配
	if req.IPAddress != "" {
		if err := h.db.Where("ip_address = ?", req.IPAddress).First(&vm).Error; err == nil {
			return vm.VMID, true, nil
		}
	}

	// 优先级 2: 通过 MAC 地址匹配
	if req.MACAddress != "" {
		if err := h.db.Where("mac_address = ?", req.MACAddress).First(&vm).Error; err == nil {
			return vm.VMID, true, nil
		}
	}

	// 优先级 3: 通过主机名模糊匹配
	if req.Hostname != "" {
		if err := h.db.Where("name LIKE ?", "%"+req.Hostname+"%").First(&vm).Error; err == nil {
			return vm.VMID, true, nil
		}
	}

	// 优先级 4: 通过机器 GUID 匹配（Windows）
	if req.MachineGUID != "" {
		if err := h.db.Where("resource_id = ?", req.MachineGUID).First(&vm).Error; err == nil {
			return vm.VMID, true, nil
		}
	}

	// 都没匹配到，生成临时 VM ID
	tempVMID := fmt.Sprintf("vm-auto-%s-%s", req.OSType, req.Hostname)
	return tempVMID, false, nil
}

// generateRandomString 生成随机字符串（简化版）
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[i%len(charset)]
	}
	return string(result)
}
