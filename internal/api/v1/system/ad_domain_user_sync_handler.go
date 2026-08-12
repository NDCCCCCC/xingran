package system

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// ADUserSyncHandler AD域用户同步处理器
type ADUserSyncHandler struct {
	userSyncService *system.UserSyncService
	core            *core.Core
}

// NewADUserSyncHandler 创建AD用户同步处理器
func NewADUserSyncHandler(core *core.Core) *ADUserSyncHandler {
	mapper := addomain.NewDeptOUmapper(core.GetDB())

	return &ADUserSyncHandler{
		userSyncService: system.NewUserSyncService(core.GetDB(), core.PwdManager, mapper),
		core:            core,
	}
}

// BatchSyncUsersRequest 批量同步用户请求
type BatchSyncUsersRequest struct {
	ConfigID      string   `json:"configId" binding:"required"`        // AD配置ID
	UserDNs       []string `json:"userDns" binding:"required,min=1"`  // 用户DN列表
	DefaultRoleID string   `json:"defaultRoleId"`                     // 默认角色ID（可选）
}

// getADUsersByDNs 根据DN列表从数据库获取AD用户详细信息
// 从 sys_ad_user 表查询，而不是LDAP连接，提高性能并复用已同步的数据
func (h *ADUserSyncHandler) getADUsersByDNs(ctx context.Context, config *models.ADConfig, userDNs []string) ([]*system.ADUserInfoForSync, error) {
	// 从数据库查询AD用户信息
	var adUsers []models.ADUser
	err := h.core.GetDB().WithContext(ctx).
		Where("ad_config_id = ? AND user_dn IN ?", config.ID, userDNs).
		Find(&adUsers).Error
	if err != nil {
		return nil, err
	}

	// 转换为 ADUserInfoForSync
	users := make([]*system.ADUserInfoForSync, 0, len(adUsers))
	for _, adUser := range adUsers {
		users = append(users, &system.ADUserInfoForSync{
			UserDN:      adUser.UserDN,
			OuDn:        adUser.OUN,
			Username:    adUser.Username,
			DisplayName: adUser.DisplayName,
			Email:       adUser.Email,
			Phone:       adUser.Phone,
			Mobile:      adUser.Mobile,
			Title:       adUser.Title,
			Department:  adUser.Department,
		})
	}

	return users, nil
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

// BatchSyncADUsers 直接批量同步AD用户到系统用户表
// @Summary 批量同步AD用户
// @Description 将选中的AD用户批量同步到系统用户表，复用首次登录同步逻辑
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param request body BatchSyncUsersRequest true "同步请求"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ad-domain/users/batch-sync [post]
func (h *ADUserSyncHandler) BatchSyncADUsers(c *gin.Context) {
	var req BatchSyncUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "请求参数错误: "+err.Error())
		return
	}

	// 验证配置ID是否存在
	var config models.ADConfig
	err := h.core.GetDB().WithContext(c.Request.Context()).
		Where("id = ? AND deleted_at IS NULL", req.ConfigID).
		First(&config).Error
	if err != nil {
		response.Error(c, 404, "AD配置不存在")
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
		response.Error(c, 500, "获取AD用户信息失败: "+err.Error())
		return
	}

	// 执行批量同步
	result, err := h.userSyncService.BatchSyncADUsers(c.Request.Context(), users, defaultRoleID)
	if err != nil {
		response.Error(c, 500, "批量同步失败: "+err.Error())
		return
	}

	// BatchSyncADUsers synchronizes selected AD users into sys_user —
	// record as Sync. T-34-W7-01 mitigation: manual sync triggers must be
	// attributable for repudiation protection.
	recordOperLog(c, h.core, "AD用户同步", OperTypeSync)

	// 返回同步结果
	response.Success(c, map[string]interface{}{
		"total":   result.Total,
		"success": result.Success,
		"failed":  result.Failed,
		"skipped": result.Skipped,
		"errors":  result.Errors,
	})
}
