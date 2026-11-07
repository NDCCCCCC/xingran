package operations

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/operations"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// writeAssetServiceError 统一资产写端点错误响应（Phase 133 / DS-05）。
//
// 类型分派保证「越界 403」与「普通错误存量语义」同时成立：
//   - *apperrors.AppError（典型为 ErrScopeDenied，NewWithHTTPStatus 显式 403）
//     → 原样透传，response.Error 取 GetHTTPStatus() = 403；
//   - 普通 error → 恢复既有 `response.Error(c, 500, err.Error())` 形态
//     （int-first-arg quirk：存量测试断言 400，零回归硬约束）；
//   - nil 不应进入本函数（调用方已判 err != nil）。
func writeAssetServiceError(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		response.Error(c, err)
		return
	}
	response.Error(c, http.StatusInternalServerError, err.Error())
}

// AssetHandler 资产处理器
type AssetHandler struct {
	assetService operations.AssetService
	core         *core.Core
}

// NewAssetHandler 创建资产处理器实例
func NewAssetHandler(assetService operations.AssetService) *AssetHandler {
	return &AssetHandler{
		assetService: assetService,
	}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *AssetHandler) WithCore(core *core.Core) *AssetHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Create 创建资产
func (h *AssetHandler) Create(c *gin.Context) {
	var asset models.Asset
	if err := c.ShouldBindJSON(&asset); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	if err := h.assetService.Create(c.Request.Context(), &asset); err != nil {
		// Phase 133 / DS-05：越界 403 透传，普通错误保持存量 500/400 语义。
		writeAssetServiceError(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "资产管理", operlog.OperTypeCreate)
	response.Success(c, asset)
}

// Update 更新资产
func (h *AssetHandler) Update(c *gin.Context) {
	var asset models.Asset
	if err := c.ShouldBindJSON(&asset); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	if err := h.assetService.Update(c.Request.Context(), &asset); err != nil {
		writeAssetServiceError(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "资产管理", operlog.OperTypeUpdate)
	response.Success(c, asset)
}

// Delete 删除资产
func (h *AssetHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "资产ID不能为空")
		return
	}

	if err := h.assetService.Delete(c.Request.Context(), id); err != nil {
		writeAssetServiceError(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "资产管理", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// GetByID 根据ID获取资产
func (h *AssetHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "资产ID不能为空")
		return
	}

	asset, err := h.assetService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, asset)
}

// List 查询资产列表
func (h *AssetHandler) List(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.assetService.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// BatchOperation 批量操作资产
func (h *AssetHandler) BatchOperation(c *gin.Context) {
	var req struct {
		Action string   `json:"action" binding:"required"`
		IDs    []string `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	switch req.Action {
	case "delete":
		if err := h.assetService.BatchDelete(c.Request.Context(), req.IDs); err != nil {
			writeAssetServiceError(c, err)
			return
		}
	default:
		response.Error(c, http.StatusBadRequest, "不支持的操作类型")
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "资产管理", operlog.OperTypeBatch)
	response.Success(c, nil)
}

// GetDeviceTypes 获取设备类型列表
func (h *AssetHandler) GetDeviceTypes(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.assetService.GetDeviceTypes(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// GetDeviceCategories 获取设备种类列表
func (h *AssetHandler) GetDeviceCategories(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.assetService.GetDeviceCategories(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// GetStatusValues 获取状态列表
func (h *AssetHandler) GetStatusValues(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.assetService.GetStatusValues(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// Statistics 资产统计(读操作,不记操作日志)
func (h *AssetHandler) Statistics(c *gin.Context) {
	result, err := h.assetService.Statistics(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// SearchBySerial 根据序列号查询资产
func (h *AssetHandler) SearchBySerial(c *gin.Context) {
	serial := c.Param("serial")
	if serial == "" {
		response.Error(c, http.StatusBadRequest, "序列号不能为空")
		return
	}

	asset, err := h.assetService.GetByDeviceSN(c.Request.Context(), serial)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	if asset == nil {
		response.Error(c, http.StatusNotFound, "资产不存在")
		return
	}

	response.Success(c, asset)
}
