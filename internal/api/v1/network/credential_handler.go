package network

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	responseHelpers "github.com/xingran-next/xingran-go-backend/pkg/response"
)

// CredentialHandler 授权凭证处理器
type CredentialHandler struct {
	credentialService *services.AuthCredentialService
	core              *core.Core
}

// NewCredentialHandler 创建授权凭证处理器实例
func NewCredentialHandler(credentialService *services.AuthCredentialService) *CredentialHandler {
	return &CredentialHandler{credentialService: credentialService}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *CredentialHandler) WithCore(core *core.Core) *CredentialHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// List 查询授权凭证列表
// @Summary 查询授权凭证列表
// @Description 分页查询授权凭证列表
// @Tags 授权凭证管理
// @Accept json
// @Produce json
// @Param request body services.ListCredentialRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /network/credentials/list [post]
// Statistics 授权凭证统计(总数/SSH/Telnet)
// @Summary 授权凭证统计
// @Description 用 COUNT 聚合返回凭证统计,供统计卡片使用
// @Tags 授权凭证管理
// @Produce json
// @Success 200 {object} response.Response
// @Router /network/credentials/statistics [post]
func (h *CredentialHandler) Statistics(c *gin.Context) {
	result, err := h.credentialService.GetStatistics(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *CredentialHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	req := services.ListCredentialRequest{}
	req.Current = getIntField(rawReq, "current", 1)
	req.PageSize = getIntField(rawReq, "pageSize", 10)
	req.OrderByColumn = getStringField(rawReq, "orderByColumn")
	req.IsAsc = getBoolFieldPtr(rawReq, "isAsc")

	if val, ok := rawReq["credentialName"].(string); ok && val != "" {
		req.CredentialName = &val
	}
	if val, ok := rawReq["protocolType"].(string); ok && val != "" {
		pt := models.ProtocolType(val)
		req.ProtocolType = &pt
	}

	credentials, total, err := h.credentialService.List(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	pageResp := response.PageResponse{
		List:     credentials,
		Total:    total,
		Current:  req.Current,
		PageSize: req.PageSize,
	}
	response.Success(c, pageResp)
}

// GetByID 获取凭证详情
// @Summary 获取凭证详情
// @Description 根据ID获取凭证详情
// @Tags 授权凭证管理
// @Accept json
// @Produce json
// @Param id path string true "凭证ID"
// @Success 200 {object} response.Response
// @Router /network/credentials/:id [post]
func (h *CredentialHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("凭证ID"))
		return
	}

	credential, err := h.credentialService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Success(c, credential)
}

// Create 创建授权凭证
// @Summary 创建授权凭证
// @Description 创建新授权凭证
// @Tags 授权凭证管理
// @Accept json
// @Produce json
// @Param request body services.CreateCredentialRequest true "凭证信息"
// @Success 200 {object} response.Response
// @Router /network/credentials [post]
func (h *CredentialHandler) Create(c *gin.Context) {
	// 记录操作日志（含请求体敏感字段脱敏 — password/enablePassword/snmpCommunity/sshKey）
	// T-34-W4-01 缓解：必须使用 RecordWithBody 读+还原 body，再经 FilterSensitiveParams 脱敏。
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "网络设备凭据", operlog.OperTypeCreate)

	var req services.CreateCredentialRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")
	req.CreatedBy = userID.(string)

	credential, err := h.credentialService.Create(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "创建凭证") {
		return
	}

	response.Success(c, credential)
}

// Update 更新授权凭证
// @Summary 更新授权凭证
// @Description 更新凭证信息
// @Tags 授权凭证管理
// @Accept json
// @Produce json
// @Param id path string true "凭证ID"
// @Param request body services.UpdateCredentialRequest true "凭证信息"
// @Success 200 {object} response.Response
// @Router /network/credentials/:id/update [post]
func (h *CredentialHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("凭证ID"))
		return
	}

	// 记录操作日志（含请求体敏感字段脱敏 — T-34-W4-01 缓解）
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "网络设备凭据", operlog.OperTypeUpdate)

	var req services.UpdateCredentialRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	req.ID = id

	userID, _ := c.Get("user_id")
	req.UpdatedBy = userID.(string)

	err := h.credentialService.Update(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "更新凭证") {
		return
	}

	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除授权凭证
// @Summary 删除授权凭证
// @Description 删除指定凭证
// @Tags 授权凭证管理
// @Accept json
// @Produce json
// @Param id path string true "凭证ID"
// @Success 200 {object} response.Response
// @Router /network/credentials/:id/delete [post]
func (h *CredentialHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("凭证ID"))
		return
	}

	err := h.credentialService.Delete(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "删除凭证") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备凭据", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// BatchDelete 批量删除凭证
// @Summary 批量删除凭证
// @Description 批量删除多个凭证
// @Tags 授权凭证管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string} true "凭证ID列表"
// @Success 200 {object} response.Response
// @Router /network/credentials/batch-delete [post]
func (h *CredentialHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	err := h.credentialService.BatchDelete(c.Request.Context(), req.IDs)
	if !responseHelpers.HandleServiceError(c, err, "批量删除凭证") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备凭据", operlog.OperTypeBatch)
	response.Success(c, gin.H{
		"message": "批量删除成功",
		"count":   len(req.IDs),
	})
}

// SetDefault 设置默认凭证
// @Summary 设置默认凭证
// @Description 设置指定凭证为默认凭证
// @Tags 授权凭证管理
// @Accept json
// @Produce json
// @Param id path string true "凭证ID"
// @Success 200 {object} response.Response
// @Router /network/credentials/:id/set-default [post]
func (h *CredentialHandler) SetDefault(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("凭证ID"))
		return
	}

	userID, _ := c.Get("user_id")
	err := h.credentialService.SetDefaultCredential(c.Request.Context(), id, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "设置默认凭证") {
		return
	}

	// SetDefault 语义上是"授权"（指定哪个凭证为默认/主用），用 OperTypeGrant
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备凭据", operlog.OperTypeGrant)
	response.Success(c, gin.H{"message": "设置成功"})
}

// GetDevicesByCredential 获取使用凭证的设备列表
// @Summary 获取使用凭证的设备列表
// @Description 获取使用指定凭证的所有设备
// @Tags 授权凭证管理
// @Accept json
// @Produce json
// @Param id path string true "凭证ID"
// @Success 200 {object} response.Response
// @Router /network/credentials/:id/devices [post]
func (h *CredentialHandler) GetDevicesByCredential(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("凭证ID"))
		return
	}

	devices, err := h.credentialService.GetDevicesByCredential(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "获取使用凭证的设备") {
		return
	}

	response.Success(c, devices)
}
