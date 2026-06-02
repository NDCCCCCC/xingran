package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AgentHandler Agent HTTP 处理器
type AgentHandler struct {
	accountManager *AccountManager
	authenticator  *JWTAuthenticator
}

// NewAgentHandler 创建 Agent 处理器
func NewAgentHandler(accountManager *AccountManager, authenticator *JWTAuthenticator) *AgentHandler {
	return &AgentHandler{
		accountManager: accountManager,
		authenticator:  authenticator,
	}
}

// RegisterRoutes 注册路由
func (h *AgentHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")

	// 公开端点
	api.POST("/health", h.HealthCheck)
	api.POST("/register", h.Register)

	// 需要 JWT 认证的端点
	auth := api.Group("")
	auth.Use(JWTAuth(h.authenticator))
	{
		auth.POST("/accounts", h.CreateAccount)
		auth.DELETE("/accounts/:username", h.DeleteAccount)
		auth.POST("/accounts/:username/reset", h.ResetPassword)
		auth.POST("/accounts/:username/enable", h.EnableAccount)
		auth.POST("/accounts/:username/disable", h.DisableAccount)
		auth.GET("/accounts", h.ListAccounts)
		auth.GET("/console/ws", h.WebSocketTerminal)
		auth.POST("/heartbeat", h.Heartbeat)
	}
}

// CreateAccount 创建账号
func (h *AgentHandler) CreateAccount(c *gin.Context) {
	var req Account
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.accountManager.CreateAccount(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account created successfully"})
}

// DeleteAccount 删除账号
func (h *AgentHandler) DeleteAccount(c *gin.Context) {
	username := c.Param("username")

	if err := h.accountManager.DeleteAccount(c.Request.Context(), username); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account deleted successfully"})
}

// ResetPassword 重置密码
func (h *AgentHandler) ResetPassword(c *gin.Context) {
	username := c.Param("username")
	var req struct {
		NewPassword string `json:"new_password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.accountManager.ResetPassword(c.Request.Context(), username, req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// EnableAccount 启用账号
func (h *AgentHandler) EnableAccount(c *gin.Context) {
	username := c.Param("username")

	if err := h.accountManager.EnableAccount(c.Request.Context(), username); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account enabled successfully"})
}

// DisableAccount 禁用账号
func (h *AgentHandler) DisableAccount(c *gin.Context) {
	username := c.Param("username")

	if err := h.accountManager.DisableAccount(c.Request.Context(), username); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account disabled successfully"})
}

// ListAccounts 列出所有账号
func (h *AgentHandler) ListAccounts(c *gin.Context) {
	accounts, err := h.accountManager.ListAccounts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts": accounts,
		"total":    len(accounts),
	})
}

// HealthCheck 健康检查
func (h *AgentHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "ruoyi-vm-agent",
	})
}

// Register Agent 注册
func (h *AgentHandler) Register(c *gin.Context) {
	var req struct {
		VMID   string `json:"vm_id"`
		AgentID string `json:"agent_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 调用后端注册 API
	if err := h.authenticator.Register(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent registered successfully"})
}

// Heartbeat 心跳上报
func (h *AgentHandler) Heartbeat(c *gin.Context) {
	if err := h.authenticator.SendHeartbeat(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Heartbeat received"})
}

// WebSocketTerminal WebSocket 终端（预留）
func (h *AgentHandler) WebSocketTerminal(c *gin.Context) {
	// Wave 9 完整实现
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "WebSocket terminal not yet implemented",
	})
}
