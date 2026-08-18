package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// 通用错误消息（避免信息泄露）
const (
	errMsgGeneric            = "请求处理失败，请稍后重试"
	errMsgInvalidRequest     = "请求参数无效"
	errMsgUnauthorized       = "未授权访问"
	errMsgForbidden          = "权限不足"
	errMsgNotFound           = "资源不存在"
	errMsgInternalError      = "服务器内部错误"
	errMsgServiceUnavailable = "服务暂时不可用"
)

// 内部错误码（用于日志，不暴露给客户端）
const (
	errCodeInternal    = 1001
	errCodeDatabase    = 1002
	errCodeExternalAPI = 1003
	errCodeConfig      = 1004
	errCodeAuth        = 1005
)

// sanitizeError 清理错误消息，移除敏感信息
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()

	// 检查是否包含敏感信息
	sensitivePatterns := []string{
		"password", "secret", "token", "key", "credential",
		"internal", "database", "sql", "query", "connection",
		"file://", "/etc/", "/var/", "C:\\",
	}

	errMsgLower := strings.ToLower(errMsg)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(errMsgLower, pattern) {
			WithFields(logrus.Fields{
				"pattern": pattern,
				"error":   errMsg,
			}).Warn("Sanitized error containing sensitive pattern")
			return errMsgGeneric
		}
	}

	return errMsg
}

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

	// 应用安全响应头中间件（全局）
	api.Use(SecurityHeaders())

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
	requestID := c.GetHeader("X-Request-ID")

	var req Account
	if err := c.ShouldBindJSON(&req); err != nil {
		WithRequestID(requestID).WithField("error", err.Error()).Warn("Invalid request for CreateAccount")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": errMsgInvalidRequest,
			"code":  errCodeInternal,
		})
		return
	}

	if err := h.accountManager.CreateAccount(c.Request.Context(), &req); err != nil {
		WithFields(logrus.Fields{
			"request_id": requestID,
			"error":      err.Error(),
			"username":   req.Username,
		}).Error("Failed to create account")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": sanitizeError(err),
			"code":  errCodeInternal,
		})
		return
	}

	WithFields(logrus.Fields{
		"request_id": requestID,
		"username":   req.Username,
	}).Info("Account created successfully")

	c.JSON(http.StatusOK, gin.H{"message": "Account created successfully"})
}

// DeleteAccount 删除账号
func (h *AgentHandler) DeleteAccount(c *gin.Context) {
	requestID := c.GetHeader("X-Request-ID")
	username := c.Param("username")

	if err := h.accountManager.DeleteAccount(c.Request.Context(), username); err != nil {
		WithFields(logrus.Fields{
			"request_id": requestID,
			"error":      err.Error(),
			"username":   username,
		}).Error("Failed to delete account")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": sanitizeError(err),
			"code":  errCodeInternal,
		})
		return
	}

	WithFields(logrus.Fields{
		"request_id": requestID,
		"username":   username,
	}).Info("Account deleted successfully")

	c.JSON(http.StatusOK, gin.H{"message": "Account deleted successfully"})
}

// ResetPassword 重置密码
func (h *AgentHandler) ResetPassword(c *gin.Context) {
	requestID := c.GetHeader("X-Request-ID")
	username := c.Param("username")

	var req struct {
		NewPassword string `json:"new_password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		WithRequestID(requestID).WithField("error", err.Error()).Warn("Invalid request for ResetPassword")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": errMsgInvalidRequest,
			"code":  errCodeInternal,
		})
		return
	}

	if err := h.accountManager.ResetPassword(c.Request.Context(), username, req.NewPassword); err != nil {
		WithFields(logrus.Fields{
			"request_id": requestID,
			"error":      err.Error(),
			"username":   username,
		}).Error("Failed to reset password")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": sanitizeError(err),
			"code":  errCodeInternal,
		})
		return
	}

	WithFields(logrus.Fields{
		"request_id": requestID,
		"username":   username,
	}).Info("Password reset successfully")

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// EnableAccount 启用账号
func (h *AgentHandler) EnableAccount(c *gin.Context) {
	requestID := c.GetHeader("X-Request-ID")
	username := c.Param("username")

	if err := h.accountManager.EnableAccount(c.Request.Context(), username); err != nil {
		WithFields(logrus.Fields{
			"request_id": requestID,
			"error":      err.Error(),
			"username":   username,
		}).Error("Failed to enable account")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": sanitizeError(err),
			"code":  errCodeInternal,
		})
		return
	}

	WithFields(logrus.Fields{
		"request_id": requestID,
		"username":   username,
	}).Info("Account enabled successfully")

	c.JSON(http.StatusOK, gin.H{"message": "Account enabled successfully"})
}

// DisableAccount 禁用账号
func (h *AgentHandler) DisableAccount(c *gin.Context) {
	requestID := c.GetHeader("X-Request-ID")
	username := c.Param("username")

	if err := h.accountManager.DisableAccount(c.Request.Context(), username); err != nil {
		WithFields(logrus.Fields{
			"request_id": requestID,
			"error":      err.Error(),
			"username":   username,
		}).Error("Failed to disable account")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": sanitizeError(err),
			"code":  errCodeInternal,
		})
		return
	}

	WithFields(logrus.Fields{
		"request_id": requestID,
		"username":   username,
	}).Info("Account disabled successfully")

	c.JSON(http.StatusOK, gin.H{"message": "Account disabled successfully"})
}

// ListAccounts 列出所有账号
func (h *AgentHandler) ListAccounts(c *gin.Context) {
	requestID := c.GetHeader("X-Request-ID")

	accounts, err := h.accountManager.ListAccounts(c.Request.Context())
	if err != nil {
		WithFields(logrus.Fields{
			"request_id": requestID,
			"error":      err.Error(),
		}).Error("Failed to list accounts")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": sanitizeError(err),
			"code":  errCodeInternal,
		})
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
		"service": "xingran-vm-agent",
	})
}

// Register Agent 注册
func (h *AgentHandler) Register(c *gin.Context) {
	requestID := c.GetHeader("X-Request-ID")

	var req struct {
		VMID    string `json:"vm_id"`
		AgentID string `json:"agent_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		WithRequestID(requestID).WithField("error", err.Error()).Warn("Invalid request for Register")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": errMsgInvalidRequest,
			"code":  errCodeInternal,
		})
		return
	}

	// 如果请求中提供了 agent_id 和 vm_id，使用它们注册
	// 否则使用 authenticator 中已有的配置
	if req.AgentID != "" && req.VMID != "" {
		// TODO: 实现动态更新 agent_id 和 vm_id 的逻辑
		// 当前使用 authenticator 中的配置
		_ = req // 占位: 当前实现始终使用 authenticator 配置;SA9003 抑制
	}

	// 调用后端注册 API
	if err := h.authenticator.RegisterToBackend(c.Request.Context(), map[string]interface{}{}); err != nil {
		WithFields(logrus.Fields{
			"request_id": requestID,
			"error":      err.Error(),
		}).Error("Failed to register agent with backend")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": sanitizeError(err),
			"code":  errCodeInternal,
		})
		return
	}

	WithFields(logrus.Fields{
		"request_id": requestID,
	}).Info("Agent registered successfully")

	c.JSON(http.StatusOK, gin.H{"message": "Agent registered successfully"})
}

// Heartbeat 心跳上报
func (h *AgentHandler) Heartbeat(c *gin.Context) {
	requestID := c.GetHeader("X-Request-ID")

	if err := h.authenticator.SendHeartbeat(c.Request.Context()); err != nil {
		WithFields(logrus.Fields{
			"request_id": requestID,
			"error":      err.Error(),
		}).Error("Heartbeat failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": sanitizeError(err),
			"code":  errCodeInternal,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Heartbeat received"})
}

// WebSocketTerminal WebSocket 终端（预留）
func (h *AgentHandler) WebSocketTerminal(c *gin.Context) {
	requestID := c.GetHeader("X-Request-ID")
	WithRequestID(requestID).Warn("WebSocket terminal not yet implemented")
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": errMsgServiceUnavailable,
		"code":  errCodeInternal,
	})
}
