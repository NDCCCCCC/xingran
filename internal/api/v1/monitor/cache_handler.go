package monitor

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

// CacheHandler 缓存监控处理器
type CacheHandler struct {
	core                 *core.Core
	db                   *gorm.DB
	cacheConfigService   *services.CacheConfigService
}

// NewCacheHandler 创建缓存监控处理器实例
func NewCacheHandler(core *core.Core, cacheConfigService *services.CacheConfigService) *CacheHandler {
	return &CacheHandler{
		core:               core,
		db:                 core.DB.GetDB(),
		cacheConfigService: cacheConfigService,
	}
}

// ==================== 请求/响应类型 ====================

// CacheListRequest 缓存列表请求
type CacheListRequest struct {
	Key      string `json:"key,omitempty"`
	Type     string `json:"type,omitempty"`
	Level    string `json:"level,omitempty"` // "l1", "l2", "all"
	Current  int    `json:"current"`
	PageSize int    `json:"pageSize"`
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
// @Success 200 {object} response.Response{data=[]models.CacheInfo}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/cache/list [post]
func (h *CacheHandler) GetCacheList(c *gin.Context) {
	var req CacheListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Current = 1
		req.PageSize = 10
	}

	if req.Current <= 0 {
		req.Current = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.Level == "" {
		req.Level = "all"
	}

	if h.core.Cache != nil {
		caches, err := getCachesFromCacheWithLevel(h.core.Cache, req)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "从Redis获取缓存列表失败: "+err.Error())
			return
		}

		total := int64(len(caches))
		start := (req.Current - 1) * req.PageSize
		end := start + req.PageSize
		if end > len(caches) {
			end = len(caches)
		}

		var paginatedCaches []models.CacheInfo
		if start < len(caches) {
			if end <= len(caches) {
				paginatedCaches = caches[start:end]
			} else {
				paginatedCaches = caches[start:]
			}
		}

		response.Page(c, paginatedCaches, total, req.Current, req.PageSize)
		return
	}

	// 从数据库查询（备用方案）
	db := h.db.Model(&models.CacheInfo{})

	if req.Key != "" {
		db = db.Where("key LIKE ?", "%"+req.Key+"%")
	}
	if req.Type != "" {
		db = db.Where("type = ?", req.Type)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "查询缓存总数失败")
		return
	}

	var caches []models.CacheInfo
	offset := (req.Current - 1) * req.PageSize
	if err := db.Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&caches).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "查询缓存列表失败")
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
// @Success 200 {object} response.Response{data=models.CacheInfo}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/cache/{key} [get]
func (h *CacheHandler) GetCacheInfo(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.Error(c, http.StatusBadRequest, "缓存键不能为空")
		return
	}

	// 如果键名包含 ruoyi: 前缀，需要移除它
	// 因为 Cache.Get() 会自动添加前缀，所以这里传入的应该是不带前缀的键名
	if strings.HasPrefix(key, "ruoyi:") {
		key = key[6:] // 移除 "ruoyi:" 前缀
	}

	if h.core.Cache != nil {
		ctx := context.Background()
		value, err := h.core.Cache.Get(ctx, key)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "获取缓存失败")
			return
		}

		if value != "" {
			ttl, _ := h.core.Cache.TTL(ctx, key)
			ttlSeconds := int64(-1)
			if ttl > 0 {
				ttlSeconds = int64(ttl.Seconds())
			}

			cacheInfo := models.CacheInfo{
				Key:       key,
				Value:     value,
				Type:      "string",
				TTL:       ttlSeconds,
				Size:      int64(len(value)),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			response.Success(c, cacheInfo)
			return
		}
	}

	var cache models.CacheInfo
	if err := h.db.Where("key = ?", key).First(&cache).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "缓存不存在")
		} else {
			response.Error(c, http.StatusInternalServerError, "查询缓存失败")
		}
		return
	}

	response.Success(c, cache)
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
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	if h.core.Cache == nil {
		response.Error(c, http.StatusInternalServerError, "缓存服务未初始化")
		return
	}

	var result interface{}
	var err error
	ctx := context.Background()

	switch req.Operation {
	case "get":
		var value string
		value, err = h.core.Cache.Get(ctx, req.Key)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "获取缓存失败")
			return
		}
		result = value

	case "set":
		if req.Value == "" {
			response.Error(c, http.StatusBadRequest, "设置缓存值不能为空")
			return
		}

		ttl := time.Hour
		if req.TTL > 0 {
			ttl = time.Duration(req.TTL) * time.Second
		}

		err = h.core.Cache.Set(ctx, req.Key, req.Value, ttl)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "设置缓存失败")
			return
		}
		result = "设置成功"

	case "del":
		err = h.core.Cache.Delete(ctx, req.Key)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "删除缓存失败")
			return
		}
		result = "删除成功"

	case "exists":
		exists, err := h.core.Cache.Exists(ctx, req.Key)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "检查缓存存在性失败")
			return
		}
		result = map[string]interface{}{"exists": exists}

	case "expire":
		if req.TTL <= 0 {
			response.Error(c, http.StatusBadRequest, "过期时间必须大于0")
			return
		}
		err = h.core.Cache.Expire(ctx, req.Key, time.Duration(req.TTL)*time.Second)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "设置过期时间失败")
			return
		}
		result = "设置成功"

	case "ttl":
		result = map[string]interface{}{"ttl": -1}

	default:
		response.Error(c, http.StatusBadRequest, "不支持的操作")
		return
	}

	// 异步保存操作日志
	go h.saveCacheOperationLog(req, result)

	response.Success(c, result)
}

// saveCacheOperationLog 保存缓存操作日志
func (h *CacheHandler) saveCacheOperationLog(req CacheOperateRequest, result interface{}) {
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

	_ = h.db.Save(&cacheInfo)
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
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	if h.core.Cache == nil {
		response.Error(c, http.StatusInternalServerError, "缓存服务未初始化")
		return
	}

	if len(req.Keys) == 0 {
		response.Error(c, http.StatusBadRequest, "缓存键列表不能为空")
		return
	}

	ctx := context.Background()
	results := make(map[string]interface{})

	switch req.Operation {
	case "get":
		values := make(map[string]interface{})
		for _, key := range req.Keys {
			value, err := h.core.Cache.Get(ctx, key)
			if err != nil {
				values[key] = "获取失败: " + err.Error()
			} else {
				values[key] = value
			}
		}
		results = values

	case "del":
		deletedKeys := make([]string, 0)
		failedKeys := make(map[string]string)

		for _, key := range req.Keys {
			err := h.core.Cache.Delete(ctx, key)
			if err != nil {
				failedKeys[key] = err.Error()
			} else {
				deletedKeys = append(deletedKeys, key)
			}
		}

		results = map[string]interface{}{
			"deleted": deletedKeys,
			"failed":  failedKeys,
		}

	default:
		response.Error(c, http.StatusBadRequest, "不支持的操作")
		return
	}

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
	if h.core.Cache == nil {
		response.Error(c, http.StatusInternalServerError, "缓存服务未初始化")
		return
	}

	ctx := context.Background()
	if err := h.core.Cache.FlushDB(ctx); err != nil {
		response.Error(c, http.StatusInternalServerError, "清空缓存失败")
		return
	}

	_ = h.db.Exec("DELETE FROM sys_cache_info")

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
// @Success 200 {object} response.Response{data=[]models.CacheStats}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/cache/stats/list [post]
func (h *CacheHandler) GetCacheStats(c *gin.Context) {
	var req CacheStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Current = 1
		req.PageSize = 10
	}

	if req.Current <= 0 {
		req.Current = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	isRealtime := req.IsRealtime || (req.StartTime == nil && req.EndTime == nil)

	if isRealtime {
		h.getRealtimeCacheStats(c)
		return
	}

	h.getHistoryCacheStats(c, req)
}

// getRealtimeCacheStats 获取实时缓存统计
func (h *CacheHandler) getRealtimeCacheStats(c *gin.Context) {
	if h.core.Cache == nil {
		response.Error(c, http.StatusInternalServerError, "缓存服务不可用")
		return
	}

	cacheWithStats, ok := h.core.Cache.(interface {
		GetStats(ctx context.Context) (map[string]interface{}, error)
	})
	if !ok {
		response.Error(c, http.StatusInternalServerError, "缓存不支持统计信息")
		return
	}

	stats, err := cacheWithStats.GetStats(context.Background())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取缓存统计失败")
		return
	}

	result := make(map[string]interface{})

	if l1Stats, ok := stats["l1"].(map[string]interface{}); ok {
		result["l1"] = formatCacheStats(l1Stats, "L1(内存)", true)
	}

	if l2Stats, ok := stats["l2"].(map[string]interface{}); ok {
		result["l2"] = formatCacheStats(l2Stats, "L2(Redis)", true)
	}

	response.Success(c, result)
}

// getHistoryCacheStats 获取历史缓存统计
func (h *CacheHandler) getHistoryCacheStats(c *gin.Context, req CacheStatsRequest) {
	db := h.db.Model(&models.CacheStats{})

	if req.CacheType != "" {
		db = db.Where("cache_type = ?", req.CacheType)
	}
	if req.StartTime != nil && *req.StartTime != "" {
		db = db.Where("collect_time >= ?", *req.StartTime)
	}
	if req.EndTime != nil && *req.EndTime != "" {
		db = db.Where("collect_time <= ?", *req.EndTime)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "查询缓存统计总数失败")
		return
	}

	var cacheStatsList []models.CacheStats
	offset := (req.Current - 1) * req.PageSize
	if err := db.Offset(offset).Limit(req.PageSize).Order("collect_time DESC").Find(&cacheStatsList).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "查询缓存统计失败")
		return
	}

	response.Page(c, cacheStatsList, total, req.Current, req.PageSize)
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
	if h.core.Cache == nil {
		response.Error(c, http.StatusInternalServerError, "缓存服务未初始化")
		return
	}

	cacheWithStats, ok := h.core.Cache.(interface {
		GetStats(ctx context.Context) (map[string]interface{}, error)
	})
	if !ok {
		response.Error(c, http.StatusInternalServerError, "缓存不支持统计信息")
		return
	}

	stats, err := cacheWithStats.GetStats(context.Background())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取监控数据失败")
		return
	}

	monitor := make(map[string]interface{})

	if l1Stats, ok := stats["l1"].(map[string]interface{}); ok {
		formattedL1Stats := formatCacheStatsForMonitor(l1Stats)
		monitor["l1"] = map[string]interface{}{
			"status": map[string]interface{}{
				"connected": true,
				"type":      "memory",
			},
			"stats": formattedL1Stats,
		}
	}

	if l2Stats, ok := stats["l2"].(map[string]interface{}); ok {
		formattedL2Stats := formatCacheStatsForMonitor(l2Stats)
		serverInfo := map[string]interface{}{
			"connected": true,
			"type":      "redis",
		}
		if version, ok := l2Stats["redis_version"].(string); ok {
			serverInfo["version"] = version
		}
		if uptime, ok := l2Stats["uptime_in_seconds"].(int64); ok {
			serverInfo["uptime"] = fmt.Sprintf("%ds", uptime)
		}

		monitor["l2"] = map[string]interface{}{
			"status": serverInfo,
			"stats":  formattedL2Stats,
		}
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
	c.ShouldBindJSON(&req)

	db := h.db.Model(&models.CacheInfo{})

	if req.Key != "" {
		db = db.Where("key LIKE ?", "%"+req.Key+"%")
	}
	if req.Type != "" {
		db = db.Where("type = ?", req.Type)
	}

	var caches []models.CacheInfo
	if err := db.Order("created_at DESC").Find(&caches).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "导出缓存数据失败")
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
	if h.cacheConfigService == nil {
		response.Error(c, http.StatusInternalServerError, "缓存配置服务未初始化")
		return
	}

	configInfoMap := h.cacheConfigService.GetConfigInfo()
	currentValues := h.cacheConfigService.GetAllConfigs(c.Request.Context())

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
	if h.cacheConfigService == nil {
		response.Error(c, http.StatusInternalServerError, "缓存配置服务未初始化")
		return
	}

	var req UpdateCacheConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	configInfoMap := h.cacheConfigService.GetConfigInfo()
	configInfo, exists := configInfoMap[req.Key]
	if !exists {
		response.Error(c, http.StatusBadRequest, "无效的配置键")
		return
	}

	if req.Value < configInfo.Min || req.Value > configInfo.Max {
		response.Error(c, http.StatusBadRequest,
			fmt.Sprintf("配置值必须在 %d 到 %d 分钟之间", configInfo.Min, configInfo.Max))
		return
	}

	if err := h.updateOrCreateConfig(req, configInfo); err != nil {
		return
	}

	if err := h.cacheConfigService.ReloadConfig(c.Request.Context()); err != nil {
		response.Error(c, http.StatusInternalServerError, "重新加载配置失败")
		return
	}

	response.Success(c, gin.H{"message": "配置更新成功"})
}

// updateOrCreateConfig 更新或创建配置
func (h *CacheHandler) updateOrCreateConfig(req UpdateCacheConfigRequest, configInfo services.ConfigInfo) error {
	var config models.Config
	err := h.db.Where("config_key = ?", req.Key).First(&config).Error

	if err == gorm.ErrRecordNotFound {
		config = models.Config{
			ConfigKey:   req.Key,
			ConfigName:  configInfo.Name,
			ConfigValue: strconv.Itoa(req.Value),
			ConfigType:  models.ConfigTypeYes,
			Remark:      configInfo.Description,
		}
		return h.db.Create(&config).Error
	}

	if err != nil {
		return err
	}

	config.ConfigValue = strconv.Itoa(req.Value)
	return h.db.Save(&config).Error
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
	if h.cacheConfigService == nil {
		response.Error(c, http.StatusInternalServerError, "缓存配置服务未初始化")
		return
	}

	if err := h.cacheConfigService.ReloadConfig(c.Request.Context()); err != nil {
		response.Error(c, http.StatusInternalServerError, "重新加载配置失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "配置重新加载成功"})
}

// ==================== 测试端点 ====================

// TestCacheEndpoint 测试缓存端点
// @Summary 测试缓存功能
// @Description 用于测试缓存功能是否正常工作
// @Tags 测试
// @Accept json
// @Produce json
// @Router /monitor/cache/test [get]
func (h *CacheHandler) TestCacheEndpoint(c *gin.Context) {
	fmt.Println("[TEST] 测试缓存端点被调用")

	if h.core.DeptService == nil {
		fmt.Println("[TEST] DeptService 为 nil")
		response.Error(c, http.StatusInternalServerError, "DeptService 未初始化")
		return
	}

	fmt.Println("[TEST] DeptService 已初始化，开始调用 GetDeptTreeWithCache")

	depts, err := h.core.DeptService.GetDeptTreeWithCache(context.Background(), false)
	if err != nil {
		fmt.Printf("[TEST] 获取部门树失败: %v\n", err)
		response.Error(c, http.StatusInternalServerError, "获取部门树失败: "+err.Error())
		return
	}

	fmt.Printf("[TEST] 获取部门树成功，数量: %d\n", len(depts))

	keys, err := h.core.Cache.Keys(context.Background(), "dept:*")
	if err != nil {
		fmt.Printf("[TEST] 获取缓存键失败: %v\n", err)
	} else {
		fmt.Printf("[TEST] 找到 %d 个部门相关缓存键\n", len(keys))
		for _, key := range keys {
			fmt.Printf("[TEST]   - %s\n", key)
		}
	}

	response.Success(c, gin.H{
		"message":   "缓存测试成功",
		"count":     len(depts),
		"cacheKeys": keys,
	})
}

// DebugRawKeys 调试端点：直接查询Redis中的所有原始键
func (h *CacheHandler) DebugRawKeys(c *gin.Context) {
	fmt.Println("[DEBUG] 查询Redis中的所有原始键")

	if h.core.Cache == nil {
		response.Error(c, http.StatusInternalServerError, "缓存服务未初始化")
		return
	}

	ctx := context.Background()

	// 使用现有的 Cache 接口查询
	allKeys, _ := h.core.Cache.Keys(ctx, "*")
	fmt.Printf("[DEBUG] 通过 Cache.Keys(\"*\") 获取到 %d 个键\n", len(allKeys))

	ruoyiKeys, _ := h.core.Cache.Keys(ctx, "ruoyi:*")
	fmt.Printf("[DEBUG] 通过 Cache.Keys(\"ruoyi:*\") 获取到 %d 个键\n", len(ruoyiKeys))

	response.Success(c, gin.H{
		"totalKeys":     len(allKeys),
		"ruoyiKeys":     len(ruoyiKeys),
		"allKeys":       allKeys,
		"ruoyiKeysList": ruoyiKeys,
	})
}
