package system

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ADUserSyncHandler AD域用户同步处理器
type ADUserSyncHandler struct {
	userSyncService *system.UserSyncService
	core            *core.Core
}

// NewADUserSyncHandler 创建AD用户同步处理器
func NewADUserSyncHandler(core *core.Core) *ADUserSyncHandler {
	return &ADUserSyncHandler{
		userSyncService: system.NewUserSyncService(core.GetDB(), core.PwdManager),
		core:            core,
	}
}

// BatchSyncUsersRequest 批量同步用户请求
type BatchSyncUsersRequest struct {
	ConfigID      string   `json:"configId" binding:"required"`        // AD配置ID
	UserDNs       []string `json:"userDns" binding:"required,min=1"`  // 用户DN列表
	DefaultRoleID string   `json:"defaultRoleId"`                     // 默认角色ID（可选）
}

// BatchSyncGroupMembersResponse 批量同步结果响应
type BatchSyncGroupMembersResponse struct {
	Total   int                `json:"total"`
	Success int                `json:"success"`
	Failed  int                `json:"failed"`
	Skipped int                `json:"skipped"`
	Errors  []system.SyncError `json:"errors,omitempty"`
}

// BatchSyncGroupMembers 批量同步AD用户组成员到系统用户表
// POST /ad-domain/groups/:id/members/sync
func (h *ADUserSyncHandler) BatchSyncGroupMembers(c *gin.Context) {
	var req BatchSyncUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	groupID := c.Param("id")
	if groupID == "" {
		response.Error(c, http.StatusBadRequest, "组ID不能为空")
		return
	}

	// 验证配置ID是否存在
	var config models.ADConfig
	err := h.core.GetDB().WithContext(c.Request.Context()).
		Where("id = ? AND deleted_at IS NULL", req.ConfigID).
		First(&config).Error
	if err != nil {
		response.Error(c, http.StatusNotFound, "AD配置不存在")
		return
	}

	// 获取默认角色ID（如果未指定）
	defaultRoleID := req.DefaultRoleID
	if defaultRoleID == "" {
		defaultRoleID = h.getDefaultRoleID(c.Request.Context())
	}

	// 根据用户DN列表查询用户详细信息
	users, err := h.getADUsersByDNs(c.Request.Context(), &config, req.UserDNs)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取AD用户信息失败: "+err.Error())
		return
	}

	// 执行批量同步
	result, err := h.userSyncService.BatchSyncADUsers(c.Request.Context(), users, defaultRoleID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "批量同步失败: "+err.Error())
		return
	}

	// 返回同步结果
	response.Success(c, BatchSyncGroupMembersResponse{
		Total:   result.Total,
		Success: result.Success,
		Failed:  result.Failed,
		Skipped: result.Skipped,
		Errors:  result.Errors,
	})
}

// getADUsersByDNs 根据DN列表获取AD用户详细信息
func (h *ADUserSyncHandler) getADUsersByDNs(ctx context.Context, config *models.ADConfig, userDNs []string) ([]*system.ADUserInfoForSync, error) {
	// 这里需要调用AD服务查询用户详细信息
	// 复用现有的AD域连接和查询逻辑

	// 暂时实现：假设DN格式为 CN=username,OU=...,DC=...
	// 实际应该调用AD域服务获取用户信息
	users := make([]*system.ADUserInfoForSync, 0, len(userDNs))

	for _, dn := range userDNs {
		// TODO: 实际应该调用AD域服务查询用户详细信息
		// 这里先解析DN提取用户名（简化实现）
		username := extractUsernameFromDN(dn)

		users = append(users, &system.ADUserInfoForSync{
			UserDN:   dn,
			Username: username,
			// 其他字段需要从AD查询获取
		})
	}

	return users, nil
}

// extractUsernameFromDN 从DN中提取用户名
// 简化实现：CN=username,OU=...,DC=... -> username
func extractUsernameFromDN(dn string) string {
	// 简单解析CN=后面的用户名
	// 实际应该使用ldap包解析DN
	parts := strings.Split(dn, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "CN=") {
			return strings.TrimPrefix(part, "CN=")
		}
	}
	return ""
}

// getDefaultRoleID 获取默认角色ID
func (h *ADUserSyncHandler) getDefaultRoleID(ctx context.Context) string {
	var config struct {
		ConfigValue string `gorm:"column:config_value"`
	}
	err := h.core.GetDB().WithContext(ctx).
		Table("sys_config").
		Select("config_value").
		Where("config_key = ?", "sys.auth.ad.default_role_id").
		First(&config).Error

	if err != nil {
		return ""
	}
	return config.ConfigValue
}
