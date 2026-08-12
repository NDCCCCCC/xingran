package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	monitorServices "github.com/xingran-next/xingran-go-backend/internal/services/monitor"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// 常量定义（用于向后兼容）
const (
	ErrMsgCacheKeyRequired     = "缓存键不能为空"
	ErrMsgCacheServiceUnavail  = "缓存服务不可用"
	ErrMsgCacheConfigUnavail   = "缓存配置服务不可用"
	ErrMsgInvalidParams        = "请求参数错误"
	ErrMsgOperationUnsupported = "不支持的操作"
	ErrMsgInvalidConfigKey     = "无效的配置键"
)

// CacheHandler 缓存监控处理器
type CacheHandler struct {
	cacheService monitorServices.CacheService
	core         *core.Core
}

// NewCacheHandler 创建缓存监控处理器实例
func NewCacheHandler(cacheService monitorServices.CacheService, core *core.Core) *CacheHandler {
	return &CacheHandler{
		cacheService: cacheService,
		core:         core,
	}
}

// WithCore 覆盖 core 依赖（链式调用，供 router 在构造后再次注入用于操作日志埋点）
func (h *CacheHandler) WithCore(core *core.Core) *CacheHandler {
	h.core = core
	return h
}

// ==================== 辅助方法 ====================

// setPaginationDefaults 设置分页默认值
func (h *CacheHandler) setPaginationDefaults(current, pageSize int) (int, int) {
	if current <= 0 {
		current = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return current, pageSize
}

// ==================== 请求/响应类型 ====================

// CacheListRequest 缓存列表请求
type CacheListRequest struct {
	Key      string `json:"key,omitempty"`
	Type     string `json:"type,omitempty"`
	Level    string `json:"level,omitempty"` // "l1", "l2", "all"
	Current      int    `json:"current"`
	PageSize      int    `json:"pageSize"`
	OrderByColumn string `json:"orderByColumn,omitempty"`
	IsAsc         bool   `json:"isAsc"`
}

// CacheStatsRequest 缓存统计请求
type CacheStatsRequest struct {
	CacheType  string  `json:"cacheType,omitempty"`
	StartTime  *string `json:"startTime,omitempty"`
	EndTime    *string `json:"endTime,omitempty"`
	IsRealtime bool    `json:"isRealtime,omitempty"` // true=实时统计, false=历史统计
	Current    int     `json:"current"`
	PageSize   int     `json:"pageSize"`
}

// CacheOperateRequest 缓存操作请求
type CacheOperateRequest struct {
	Key       string `json:"key" binding:"required"`
	Value     string `json:"value,omitempty"`
	TTL       int64  `json:"ttl,omitempty"`
	Operation string `json:"operation" binding:"required,oneof=get set del exists expire ttl"`
}

// CacheBatchRequest 批量缓存操作请求
type CacheBatchRequest struct {
	Keys      []string `json:"keys" binding:"required"`
	Operation string   `json:"operation" binding:"required,oneof=get del"`
}

// UpdateCacheConfigRequest 更新缓存配置请求
type UpdateCacheConfigRequest struct {
	Key   string `json:"key" binding:"required"`                 // 配置键
	Value int    `json:"value" binding:"required,min=1,max=180"` // 配置值（分钟）
}

// CacheConfigInfo 缓存配置信息
type CacheConfigInfo struct {
	Key         string `json:"key"`         // 配置键
	Name        string `json:"name"`        // 配置名称
	Description string `json:"description"` // 配置说明
	Category    string `json:"category"`    // 配置分类
	Min         int    `json:"min"`         // 最小值（分钟）
	Max         int    `json:"max"`         // 最大值（分钟）
	Default     int    `json:"default"`     // 默认值（分钟）
	Value       int    `json:"value"`       // 当前值（分钟）
}

// CacheConfigCategory 缓存配置分类
type CacheConfigCategory struct {
	Name    string            `json:"name"`    // 分类名称
	Configs []CacheConfigInfo `json:"configs"` // 配置列表
}

// ==================== 缓存查询操作 ====================

// GetCacheList 获取缓存列表
// @Summary 获取缓存列表
// @Description 分页查询缓存键值列表
// @Tags 缓存管理
// @Accept json
// @Produce json
// @Param request body CacheListRequest false "查询参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/cache/list [post]
func (h *CacheHandler) GetCacheList(c *gin.Context) {
	var req CacheListRequest
	_ = c.ShouldBindJSON(&req)

	req.Current, req.PageSize = h.setPaginationDefaults(req.Current, req.PageSize)
	if req.Level == "" {
		req.Level = "all"
	}

	params := monitorServices.CacheListParams{
		Key:           req.Key,
		Type:          req.Type,
		Level:         req.Level,
		Current:       req.Current,
		PageSize:      req.PageSize,
		OrderByColumn: req.OrderByColumn,
		IsAsc:         req.IsAsc,
	}
	caches, total, err := h.cacheService.GetCacheList(c.Request.Context(), params)
	if err != nil {
		response.Error(c, apperrors.CacheOperationFailed(err))
		return
	}

	response.Page(c, caches, total, req.Current, req.PageSize)
}

// GetCacheInfo 获取缓存详情
// @Summary 获取缓存详情
// @Description 根据缓存键获取缓存详细信息
// @Tags 缓存管理
// @Accept json
// @Produce json
// @Param key path string true "缓存键"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/cache/{key} [get]
func (h *CacheHandler) GetCacheInfo(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.Error(c, apperrors.ParamMissing("key"))
		return
	}

	cacheInfo, err := h.cacheService.GetCacheInfo(c.Request.Context(), key)
	if err != nil {
		if err == monitorServices.ErrCacheNotFound {
			response.Error(c, apperrors.CacheKeyNotFound())
		} else {
			response.Error(c, apperrors.CacheOperationFailed(err))
		}
		return
	}

	response.Success(c, cacheInfo)
}

// ==================== 缓存操作 ====================

// OperateCache 操作缓存
// @Summary 操作缓存
// @Description 对缓存进行get/set/del/exists/expire/ttl等操作
// @Tags 缓存管理
// @Accept json
// @Produce json
// @Param request body CacheOperateRequest true "操作参数"
// @Success 200 {object} response.Response{data=interface{}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/cache/operate [post]
func (h *CacheHandler) OperateCache(c *gin.Context) {
	var req CacheOperateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.ParamError())
		return
	}

	params := monitorServices.CacheOperateParams{
		Key:       req.Key,
		Value:     req.Value,
		TTL:       req.TTL,
		Operation: req.Operation,
	}

	result, err := h.cacheService.OperateCache(c.Request.Context(), params)
	if err != nil {
		response.Error(c, apperrors.CacheOperationFailed(err))
		return
	}

	// 异步保存操作日志
	go h.saveCacheOperationLog(req, result)

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "缓存监控", operlog.OperTypeUpdate)

	response.Success(c, result)
}

// saveCacheOperationLog 保存缓存操作日志
func (h *CacheHandler) saveCacheOperationLog(req CacheOperateRequest, result interface{}) {
	if h.core == nil || h.core.DB == nil {
		return
	}

	cacheInfo := models.CacheInfo{
		Key:  req.Key,
		Type: req.Operation,
		Size: int64(len(fmt.Sprintf("%v", result))),
	}

	if req.Value != "" {
		cacheInfo.Value = req.Value
	}
	if req.TTL > 0 {
		cacheInfo.TTL = req.TTL
	}

	_ = h.core.DB.GetDB().Save(&cacheInfo)
}

// BatchOperateCache 批量操作缓存
// @Summary 批量操作缓存
// @Description 批量获取或删除缓存
// @Tags 缓存管理
// @Accept json
// @Produce json
// @Param request body CacheBatchRequest true "批量操作参数"
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/cache/batch [post]
func (h *CacheHandler) BatchOperateCache(c *gin.Context) {
	var req CacheBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.ParamError())
		return
	}

	if len(req.Keys) == 0 {
		response.Error(c, apperrors.ParamMissing("缓存键列表"))
		return
	}

	params := monitorServices.CacheBatchOperateParams{
		Keys:      req.Keys,
		Operation: req.Operation,
	}

	results, err := h.cacheService.BatchOperateCache(c.Request.Context(), params)
	if err != nil {
		response.Error(c, apperrors.CacheOperationFailed(err))
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "缓存监控", operlog.OperTypeBatch)

	response.Success(c, results)
}

// ClearCache 清空缓存
// @Summary 清空缓存
// @Description 清空所有缓存数据
// @Tags 缓存管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/cache/clear [post]
func (h *CacheHandler) ClearCache(c *gin.Context) {
	if err := h.cacheService.ClearCache(c.Request.Context()); err != nil {
		response.Error(c, apperrors.CacheOperationFailed(err))
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "缓存监控", operlog.OperTypeClean)

	response.Success(c, gin.H{"message": "清空成功"})
}

// ==================== 缓存统计和监控 ====================

// GetCacheStats 获取缓存统计
// @Summary 获取缓存统计
// @Description 获取缓存性能统计信息
// @Tags 缓存管理
// @Accept json
// @Produce json
// @Param request body CacheStatsRequest false "查询参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/cache/stats/list [post]
func (h *CacheHandler) GetCacheStats(c *gin.Context) {
	var req CacheStatsRequest
	_ = c.ShouldBindJSON(&req)

	req.Current, req.PageSize = h.setPaginationDefaults(req.Current, req.PageSize)

	params := monitorServices.CacheStatsParams{
		CacheType:  req.CacheType,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		IsRealtime: req.IsRealtime,
		Current:    req.Current,
		PageSize:   req.PageSize,
	}

	statsList, total, err := h.cacheService.GetCacheStats(c.Request.Context(), params)
	if err != nil {
		response.Error(c, apperrors.CacheOperationFailed(err))
		return
	}

	response.Page(c, statsList, total, req.Current, req.PageSize)
}

// GetCacheMonitor 获取缓存监控数据
// @Summary 获取缓存监控数据
// @Description 获取缓存实时监控数据，包括L1和L2的统计信息
// @Tags 缓存管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 500 {object} response.Response
// @Router /monitor/cache/monitor [post]
func (h *CacheHandler) GetCacheMonitor(c *gin.Context) {
	monitor, err := h.cacheService.GetCacheMonitor(c.Request.Context())
	if err != nil {
		response.Error(c, apperrors.ServerError(err))
		return
	}

	response.Success(c, monitor)
}

// ExportCache 导出缓存数据
// @Summary 导出缓存数据
// @Description 导出缓存数据到JSON文件
// @Tags 缓存管理
// @Accept json
// @Produce json
// @Param request body CacheListRequest false "导出条件"
// @Success 200 {object} response.Response{data=string}
// @Failure 500 {object} response.Response
// @Router /monitor/cache/export [post]
func (h *CacheHandler) ExportCache(c *gin.Context) {
	var req CacheListRequest
	_ = c.ShouldBindJSON(&req)

	params := monitorServices.CacheExportParams{
		Key:  req.Key,
		Type: req.Type,
	}

	caches, err := h.cacheService.ExportCache(c.Request.Context(), params)
	if err != nil {
		response.Error(c, apperrors.CacheOperationFailed(err))
		return
	}

	filename := fmt.Sprintf("cache_export_%s.json", time.Now().Format("20060102_150405"))

	response.Success(c, map[string]interface{}{
		"filename": filename,
		"data":     caches,
		"count":    len(caches),
	})
}

// ==================== 缓存配置管理 ====================

// GetCacheConfigs 获取缓存配置列表
// @Summary 获取缓存配置列表
// @Description 获取所有缓存配置信息，按分类组织
// @Tags 缓存配置
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]CacheConfigCategory}
// @Router /monitor/cache/config [get]
func (h *CacheHandler) GetCacheConfigs(c *gin.Context) {
	configInfoMap, currentValues, err := h.cacheService.GetCacheConfigs(c.Request.Context())
	if err != nil {
		response.Error(c, apperrors.CacheOperationFailed(err))
		return
	}

	// 转换为响应格式
	categoryMap := make(map[string][]CacheConfigInfo)

	for key, info := range configInfoMap {
		configInfo := CacheConfigInfo{
			Key:         info.Key,
			Name:        info.Name,
			Description: info.Description,
			Category:    info.Category,
			Min:         info.Min,
			Max:         info.Max,
			Default:     info.Default,
			Value:       currentValues[key],
		}

		categoryMap[info.Category] = append(categoryMap[info.Category], configInfo)
	}

	var categories []CacheConfigCategory
	for name, configs := range categoryMap {
		categories = append(categories, CacheConfigCategory{
			Name:    name,
			Configs: configs,
		})
	}

	response.Success(c, categories)
}

// UpdateCacheConfig 更新缓存配置
// @Summary 更新缓存配置
// @Description 更新指定缓存项的时间配置
// @Tags 缓存配置
// @Accept json
// @Produce json
// @Param request body UpdateCacheConfigRequest true "更新缓存配置请求"
// @Success 200 {object} response.Response
// @Router /monitor/cache/config [put]
func (h *CacheHandler) UpdateCacheConfig(c *gin.Context) {
	var req UpdateCacheConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.ParamError())
		return
	}

	if err := h.cacheService.UpdateCacheConfig(c.Request.Context(), req.Key, req.Value); err != nil {
		if err == monitorServices.ErrInvalidConfigKey {
			response.Error(c, apperrors.ParamInvalid("配置键"))
		} else {
			response.Error(c, apperrors.CacheOperationFailed(err))
		}
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "缓存监控", operlog.OperTypeUpdate)

	response.Success(c, gin.H{"message": "配置更新成功"})
}

// ReloadCacheConfigs 重新加载缓存配置
// @Summary 重新加载缓存配置
// @Description 从数据库重新加载所有缓存配置
// @Tags 缓存配置
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /monitor/cache/config/reload [post]
func (h *CacheHandler) ReloadCacheConfigs(c *gin.Context) {
	if err := h.cacheService.ReloadCacheConfigs(c.Request.Context()); err != nil {
		response.Error(c, apperrors.CacheOperationFailed(err))
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "缓存监控", operlog.OperTypeOther)

	response.Success(c, gin.H{"message": "配置重新加载成功"})
}

// ==================== 测试端点（保留用于调试）====================

// TestCacheEndpoint 测试缓存端点
// @Summary 测试缓存功能
// @Description 用于测试缓存功能是否正常工作
// @Tags 测试
// @Accept json
// @Produce json
// @Router /monitor/cache/test [get]
func (h *CacheHandler) TestCacheEndpoint(c *gin.Context) {
	if h.core == nil || h.core.DataCacheService == nil {
		response.Error(c, apperrors.ServerError(fmt.Errorf("DataCacheService 未初始化")))
		return
	}

	// 测试缓存写入
	testKey := "test:cache:key"
	testValue := gin.H{"message": "test data", "timestamp": time.Now().Unix()}

	err := h.core.DataCacheService.Set(c.Request.Context(), testKey, testValue, 5*time.Minute)
	if err != nil {
		response.Error(c, apperrors.CacheOperationFailed(err))
		return
	}

	// 测试缓存读取
	var result gin.H
	err = h.core.DataCacheService.Get(c.Request.Context(), testKey, &result)
	if err != nil {
		response.Error(c, apperrors.CacheOperationFailed(err))
		return
	}

	// 查询缓存键
	keys, err := h.core.Cache.Keys(c.Request.Context(), "test:*")
	if err != nil {
		keys = []string{}
	}

	response.Success(c, gin.H{
		"message":   "缓存测试成功",
		"data":      result,
		"cacheKeys": keys,
	})
}

// DebugRawKeys 调试端点：直接查询Redis中的所有原始键
// @Summary 调试原始缓存键
// @Description 查询Redis中所有原始键和xingran前缀的键，用于调试
// @Tags 监控管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 500 {object} response.Response
// @Router /monitor/cache/debug/raw-keys [post]
func (h *CacheHandler) DebugRawKeys(c *gin.Context) {
	if h.core == nil || h.core.Cache == nil {
		response.Error(c, apperrors.ServerError(fmt.Errorf("缓存服务未初始化")))
		return
	}

	ctx := c.Request.Context()

	allKeys, _ := h.core.Cache.Keys(ctx, "*")
	xingranKeys, _ := h.core.Cache.Keys(ctx, "xingran:*")

	response.Success(c, gin.H{
		"totalKeys":     len(allKeys),
		"xingranKeys":     len(xingranKeys),
		"allKeys":       allKeys,
		"xingranKeysList": xingranKeys,
	})
}

// DebugL1Cache 调试端点：检查L1缓存状态和命中率
// @Summary 调试L1缓存状态
// @Description 检查L1缓存的统计信息、命中率和键数量，用于诊断缓存问题
// @Tags 监控管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 500 {object} response.Response
// @Router /monitor/cache/debug/l1 [post]
func (h *CacheHandler) DebugL1Cache(c *gin.Context) {
	if h.core == nil || h.core.Cache == nil {
		response.Error(c, apperrors.ServerError(fmt.Errorf("缓存服务未初始化")))
		return
	}

	ctx := c.Request.Context()

	// 获取缓存统计
	type StatsProvider interface {
		GetStats(ctx context.Context) (map[string]interface{}, error)
	}

	cacheWithStats, ok := h.core.Cache.(StatsProvider)
	if !ok {
		response.Error(c, apperrors.ServerError(fmt.Errorf("缓存不支持统计信息")))
		return
	}

	stats, err := cacheWithStats.GetStats(ctx)
	if err != nil {
		response.Error(c, apperrors.CacheOperationFailed(fmt.Errorf("获取缓存统计失败")))
		return
	}

	// 提取L1统计
	l1StatsRaw, hasL1 := stats["l1"]
	var l1Stats map[string]interface{}
	if hasL1 {
		l1Stats, _ = l1StatsRaw.(map[string]interface{})
	}

	// 获取L1缓存键数量
	var l1Keys []string
	type KeysByLevelInterface interface {
		KeysByLevel(ctx context.Context, pattern string, level string) ([]string, error)
	}
	if cacheWithLevel, ok := h.core.Cache.(KeysByLevelInterface); ok {
		l1Keys, _ = cacheWithLevel.KeysByLevel(ctx, "*", "l1")
	}

	result := gin.H{
		"l1_stats": gin.H{
			"exists":      hasL1,
			"stats":       l1Stats,
			"key_count":   len(l1Keys),
			"sample_keys": l1Keys,
		},
	}

	// 分析命中率
	if l1Stats != nil {
		hits, _ := l1Stats["keyspace_hits"].(int64)
		misses, _ := l1Stats["keyspace_misses"].(int64)
		hitRate, _ := l1Stats["hit_rate"].(float64)

		result["analysis"] = gin.H{
			"hits":             hits,
			"misses":           misses,
			"total":            hits + misses,
			"hit_rate":         hitRate,
			"hit_rate_percent": fmt.Sprintf("%.2f%%", hitRate),
		}

		if hits == 0 && misses == 0 {
			result["diagnosis"] = "L1缓存没有发生任何读取操作（hits=0, misses=0）"
		} else if hits == 0 && misses > 0 {
			result["diagnosis"] = fmt.Sprintf("L1缓存全部未命中（%d次misses），可能原因：1.缓存冷启动 2.缓存过期(5分钟) 3.LRU淘汰", misses)
		} else if hitRate < 20 {
			result["diagnosis"] = fmt.Sprintf("L1缓存命中率过低(%.2f%%)，建议：1.增加L1容量 2.延长TTL 3.检查访问模式", hitRate)
		} else {
			result["diagnosis"] = "L1缓存工作正常"
		}
	}

	response.Success(c, result)
}
