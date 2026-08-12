package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// APIKeyHandler API密钥处理器
type APIKeyHandler struct {
	apiKeyService systemServices.APIKeyService
	core          *core.Core
}

// NewAPIKeyHandler 创建API密钥处理器实例
func NewAPIKeyHandler(apiKeyService systemServices.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyService: apiKeyService,
	}
}

// WithCore 注入 core 依赖（操作日志记录所需）。返回 receiver 自身以支持链式调用。
// Phase 34 操作日志全模块覆盖新增，用于 operlog.Record 访问 core.OperLogService 与
// core.GetDB()。不改写 NewAPIKeyHandler 单参构造器签名，避免破坏既有调用点。
func (h *APIKeyHandler) WithCore(core *core.Core) *APIKeyHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Create 创建API密钥
// @Summary 创建API密钥
// @Description 创建新的API密钥，完整密钥仅在此返回一次
// @Tags API密钥管理
// @Accept json
// @Produce json
// @Param request body requests.CreateAPIKeyRequest true "API密钥信息"
// @Success 200 {object} response.Response{data=map[string]string}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/apikeys [post]
func (h *APIKeyHandler) Create(c *gin.Context) {
	var req requests.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	// 从context获取用户ID
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, apperrors.Wrap(nil, apperrors.CodeUnauthorized, "用户未认证"))
		return
	}

	// 调用服务层创建密钥
	key, err := h.apiKeyService.CreateAPIKey(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	// T-34-W2-01: API 密钥创建属于敏感路径，用 RecordWithBody 读+还原+遮蔽，
	// 确保请求体中的 name/scopes 等正常记录，但任何含 secret/key 的字段被遮蔽。
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "API密钥管理", operlog.OperTypeCreate)
	// 返回完整密钥（仅此一次）
	response.Success(c, gin.H{"key": *key})
}

// List 查询API密钥列表
// @Summary 查询API密钥列表
// @Description 分页查询API密钥列表，支持关键词搜索和状态筛选
// @Tags API密钥管理
// @Accept json
// @Produce json
// @Param request body requests.ListAPIKeysParams true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /system/apikeys/list [post]
func (h *APIKeyHandler) List(c *gin.Context) {
	var params requests.ListAPIKeysParams
	if err := c.ShouldBindJSON(&params); err != nil {
		// 如果是空请求体或解析失败，使用默认值
		params = requests.DefaultListAPIKeysParams()
	}

	// 应用分页限制
	current, pageSize := params.GetPagination()
	params.Current = current
	params.PageSize = pageSize

	// 从context获取用户ID
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, apperrors.Wrap(nil, apperrors.CodeUnauthorized, "用户未认证"))
		return
	}

	result, err := h.apiKeyService.ListAPIKeys(c.Request.Context(), userID, params)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 脱敏处理：key字段仅保留前12位
	maskedList := maskAPIKeys(result.List.([]models.APIKey))

	response.Page(c, maskedList, result.Total, result.Current, result.PageSize)
}

// GetByID 获取API密钥详情
// @Summary 获取API密钥详情
// @Description 根据ID获取API密钥详细信息
// @Tags API密钥管理
// @Accept json
// @Produce json
// @Param id path string true "密钥ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /system/apikeys/:id [post]
func (h *APIKeyHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("密钥ID"))
		return
	}

	apiKey, err := h.apiKeyService.GetAPIKey(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 脱敏处理：key字段仅保留前12位
	maskedKey := maskKey(apiKey.Key)
	apiKey.Key = maskedKey

	response.Success(c, apiKey)
}

// Update 更新API密钥
// @Summary 更新API密钥
// @Description 更新API密钥信息（不能更新密钥值本身）
// @Tags API密钥管理
// @Accept json
// @Produce json
// @Param id path string true "密钥ID"
// @Param request body requests.UpdateAPIKeyRequest true "更新信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /system/apikeys/:id/update [post]
func (h *APIKeyHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("密钥ID"))
		return
	}

	var req requests.UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	// 设置ID
	req.ID = id

	if err := h.apiKeyService.UpdateAPIKey(c.Request.Context(), id, &req); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "API密钥管理", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除API密钥
// @Summary 删除API密钥
// @Description 软删除API密钥
// @Tags API密钥管理
// @Accept json
// @Produce json
// @Param id path string true "密钥ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /system/apikeys/:id/delete [post]
func (h *APIKeyHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("密钥ID"))
		return
	}

	if err := h.apiKeyService.DeleteAPIKey(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "API密钥管理", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// ToggleStatus 切换API密钥状态
// @Summary 切换API密钥状态
// @Description 启用或禁用API密钥
// @Tags API密钥管理
// @Accept json
// @Produce json
// @Param id path string true "密钥ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /system/apikeys/:id/toggle [post]
func (h *APIKeyHandler) ToggleStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("密钥ID"))
		return
	}

	if err := h.apiKeyService.ToggleAPIKeyStatus(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "API密钥管理", operlog.OperTypeStatus)
	response.Success(c, gin.H{"message": "状态切换成功"})
}

// ListUsageLogs 查询API密钥使用日志
// @Summary 查询API密钥使用日志
// @Description 分页查询API密钥使用日志
// @Tags API密钥管理
// @Accept json
// @Produce json
// @Param id path string true "密钥ID"
// @Param request body requests.ListUsageLogsParams true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /system/apikeys/:id/logs [post]
func (h *APIKeyHandler) ListUsageLogs(c *gin.Context) {
	keyID := c.Param("id")
	if keyID == "" {
		response.Error(c, apperrors.ParamMissing("密钥ID"))
		return
	}

	var params systemServices.ListUsageLogsParams
	if err := c.ShouldBindJSON(&params); err != nil {
		// 使用默认值
		params = systemServices.ListUsageLogsParams{
			Current:  1,
			PageSize: 20,
		}
	}

	// 应用分页限制（最大100条每页）
	if params.Current < 1 {
		params.Current = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	// 设置API密钥ID
	params.APIKeyID = keyID

	result, err := h.apiKeyService.ListUsageLogs(c.Request.Context(), params)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetUsageSummary 获取API密钥使用统计
// @Summary 获取API密钥使用统计
// @Description 获取API密钥的使用统计汇总信息
// @Tags API密钥管理
// @Accept json
// @Produce json
// @Param id path string true "密钥ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /system/apikeys/:id/summary [get]
func (h *APIKeyHandler) GetUsageSummary(c *gin.Context) {
	keyID := c.Param("id")
	if keyID == "" {
		response.Error(c, apperrors.ParamMissing("密钥ID"))
		return
	}

	summary, err := h.apiKeyService.GetUsageLogSummary(c.Request.Context(), keyID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, summary)
}

// maskKey 脱敏处理密钥（仅保留前12位）
func maskKey(key string) string {
	if len(key) <= 12 {
		return key
	}
	return key[:12] + "..."
}

// maskAPIKeys 批量脱敏处理密钥列表
func maskAPIKeys(keys []models.APIKey) []map[string]interface{} {
	result := make([]map[string]interface{}, len(keys))
	for i, key := range keys {
		result[i] = map[string]interface{}{
			"id":            key.ID,
			"name":          key.Name,
			"key":           maskKey(key.Key),
			"userId":        key.UserID,
			"expiresAt":     key.ExpiresAt,
			"lastUsedAt":    key.LastUsedAt,
			"isActive":      key.IsActive,
			"scopes":        key.Scopes,
			"ipWhitelist":   key.IPWhitelist,
			"description":   key.Description,
			"inheritPerms":  key.InheritPerms,
			"createdAt":     key.CreatedAt,
			"updatedAt":     key.UpdatedAt,
		}
	}
	return result
}
