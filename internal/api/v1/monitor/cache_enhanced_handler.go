package monitor

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// CacheEnhancedHandler 增强缓存处理器
type CacheEnhancedHandler struct {
	core *core.Core
}

// NewCacheEnhancedHandler 创建增强缓存处理器
func NewCacheEnhancedHandler(core *core.Core) *CacheEnhancedHandler {
	return &CacheEnhancedHandler{core: core}
}

// GetStatsRequest 获取缓存统计信息请求
type GetStatsRequest struct {
	Module string `json:"module"` // 可选，指定模块名称
}

// GetStatsResponse 获取缓存统计信息响应
type GetStatsResponse struct {
	*system.CacheManagerStats
}

// GetCacheStats 获取缓存统计信息
// @Summary 获取缓存统计信息
// @Description 获取缓存系统的统计信息，包括命中率、键数量等
// @Tags 监控
// @Accept json
// @Produce json
// @Param body body GetStatsRequest false "查询参数"
// @Success 200 {object} response.Response{data=GetStatsResponse}
// @Router /monitor/cache/stats [post]
func (h *CacheEnhancedHandler) GetCacheStats(c *gin.Context) {
	if h.core.CacheManager == nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("缓存管理器未初始化"))
		return
	}

	stats, err := h.core.CacheManager.GetStats(c.Request.Context())
	if err != nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("获取缓存统计失败"))
		return
	}

	response.Success(c, GetStatsResponse{
		CacheManagerStats: stats,
	})
}

// InvalidateByModuleRequest 按模块清除缓存请求
type InvalidateByModuleRequest struct {
	Module  string `json:"module" binding:"required"` // 模块名称（如 "user", "role" 等）
	KeyType string `json:"keyType"`                   // 可选，键类型
}

// InvalidateByModule 按模块清除缓存
// @Summary 按模块清除缓存
// @Description 根据模块名称清除相关缓存，可选择清除特定类型的缓存
// @Tags 监控
// @Accept json
// @Produce json
// @Param body body InvalidateByModuleRequest true "清除参数"
// @Success 200 {object} response.Response
// @Router /monitor/cache/invalidate [post]
func (h *CacheEnhancedHandler) InvalidateByModule(c *gin.Context) {
	if h.core.CacheManager == nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("缓存管理器未初始化"))
		return
	}

	var req InvalidateByModuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.BadRequest("请求参数错误"))
		return
	}

	err := h.core.CacheManager.InvalidateByModule(c.Request.Context(), req.Module, req.KeyType)
	if err != nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("清除缓存失败"))
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "缓存监控", operlog.OperTypeClean,
		operlog.WithOperParam("module="+req.Module))
	response.Success(c, nil)
}

// InvalidateByPatternRequest 按模式清除缓存请求
type InvalidateByPatternRequest struct {
	Pattern string `json:"pattern" binding:"required"` // 缓存键模式
}

// InvalidateByPattern 按模式清除缓存
// @Summary 按模式清除缓存
// @Description 根据缓存键模式清除匹配的所有缓存
// @Tags 监控
// @Accept json
// @Produce json
// @Param body body InvalidateByPatternRequest true "清除参数"
// @Success 200 {object} response.Response
// @Router /monitor/cache/invalidate-pattern [post]
func (h *CacheEnhancedHandler) InvalidateByPattern(c *gin.Context) {
	if h.core.CacheManager == nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("缓存管理器未初始化"))
		return
	}

	var req InvalidateByPatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.BadRequest("请求参数错误"))
		return
	}

	err := h.core.CacheManager.InvalidateByPattern(c.Request.Context(), req.Pattern)
	if err != nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("清除缓存失败"))
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "缓存监控", operlog.OperTypeClean,
		operlog.WithOperParam("pattern="+req.Pattern))
	response.Success(c, nil)
}

// WarmUpCacheRequest 缓存预热请求
type WarmUpCacheRequest struct {
	Modules []string `json:"modules"` // 可选，指定要预热的模块列表，为空则预热所有
}

// WarmUpCacheResponse 缓存预热响应
type WarmUpCacheResponse struct {
	SuccessCount int      `json:"successCount"` // 成功数量
	FailCount    int      `json:"failCount"`    // 失败数量
	Errors       []string `json:"errors"`       // 错误信息列表
}

// WarmUpCache 执行缓存预热
// @Summary 执行缓存预热
// @Description 执行缓存预热操作，可选择预热特定模块或全部模块
// @Tags 监控
// @Accept json
// @Produce json
// @Param body body WarmUpCacheRequest false "预热参数"
// @Success 200 {object} response.Response{data=WarmUpCacheResponse}
// @Router /monitor/cache/warmup [post]
func (h *CacheEnhancedHandler) WarmUpCache(c *gin.Context) {
	if h.core.CacheManager == nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("缓存管理器未初始化"))
		return
	}

	// 注意：预热功能需要在路由注册时传入相应的服务实例
	// 这里提供一个框架，实际使用时需要根据具体的服务实例构建预热函数

	// 记录操作日志（即使未实现，缓存预热属合规敏感操作，需审计调用尝试）。
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "缓存监控", operlog.OperTypeClean,
		operlog.WithErrorMsg("warmup not implemented"),
		operlog.WithStatus(1))
	response.Error(c, apperrors.NotImplemented())
}

// GetKeyInfoRequest 获取缓存键信息请求
type GetKeyInfoRequest struct {
	Key string `json:"key" binding:"required"` // 缓存键
}

// GetKeyInfoResponse 获取缓存键信息响应
type GetKeyInfoResponse struct {
	Key    string `json:"key"`    // 键
	Exists bool   `json:"exists"` // 是否存在
	TTL    int64  `json:"ttl"`    // 剩余存活时间（秒）
	Value  string `json:"value"`  // 值（可选，仅在请求时返回）
	Size   int64  `json:"size"`   // 大小（字节）
}

// GetKeyInfo 获取缓存键信息
// @Summary 获取缓存键信息
// @Description 获取指定缓存键的详细信息，包括是否存活、TTL等
// @Tags 监控
// @Accept json
// @Produce json
// @Param body body GetKeyInfoRequest true "查询参数"
// @Success 200 {object} response.Response{data=GetKeyInfoResponse}
// @Router /monitor/cache/key-info [post]
func (h *CacheEnhancedHandler) GetKeyInfo(c *gin.Context) {
	if h.core.CacheManager == nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("缓存管理器未初始化"))
		return
	}

	var req GetKeyInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.BadRequest("请求参数错误"))
		return
	}

	exists, err := h.core.CacheManager.Exists(c.Request.Context(), req.Key)
	if err != nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("查询缓存失败"))
		return
	}

	resp := GetKeyInfoResponse{
		Key:    req.Key,
		Exists: exists,
	}

	if exists {
		ttl, _ := h.core.CacheManager.GetTTL(c.Request.Context(), req.Key)
		resp.TTL = int64(ttl.Seconds())
	}

	response.Success(c, resp)
}
