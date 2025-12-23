package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/cache"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

// CacheListRequest 缓存列表请求
type CacheListRequest struct {
	Key      string `json:"key,omitempty"`
	Type     string `json:"type,omitempty"`
	Current  int    `json:"current"`
	PageSize int    `json:"pageSize"`
}

// CacheStatsRequest 缓存统计请求
type CacheStatsRequest struct {
	CacheType string     `json:"cacheType,omitempty"`
	StartTime *string    `json:"startTime,omitempty"`
	EndTime   *string    `json:"endTime,omitempty"`
	Current   int        `json:"current"`
	PageSize  int        `json:"pageSize"`
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

// getCacheList 获取缓存列表
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
func getCacheList(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CacheListRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			req.Current = 1
			req.PageSize = 10
		}

		// 设置默认值
		if req.Current <= 0 {
			req.Current = 1
		}
		if req.PageSize <= 0 {
			req.PageSize = 10
		}

		// 如果有Redis实例，从Redis获取实时数据
		if core.Cache != nil {
			caches, err := getCachesFromCache(core.Cache, req)
			if err != nil {
				response.Error(c, response.ErrServerError, "从Redis获取缓存列表失败: "+err.Error())
				return
			}

			// 计算总数
			total := int64(len(caches))

			// 实现简单的分页逻辑
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
		db := core.DB.GetDB().Model(&models.CacheInfo{})

		if req.Key != "" {
			db = db.Where("key LIKE ?", "%"+req.Key+"%")
		}
		if req.Type != "" {
			db = db.Where("type = ?", req.Type)
		}

		// 统计总数
		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询缓存总数失败")
			return
		}

		// 查询分页数据
		var caches []models.CacheInfo
		offset := (req.Current - 1) * req.PageSize
		if err := db.Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&caches).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询缓存列表失败")
			return
		}

		response.Page(c, caches, total, req.Current, req.PageSize)
	}
}

// getCacheInfo 获取缓存详情
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
func getCacheInfo(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		if key == "" {
			response.Error(c, response.ErrBadRequest, "缓存键不能为空")
			return
		}

		// 从缓存中获取
		if core.Cache != nil {
			ctx := context.Background()
			value, err := core.Cache.Get(ctx, key)
			if err != nil {
				response.Error(c, response.ErrServerError, "获取缓存失败")
				return
			}

			if value != "" {
				// 获取键的TTL
				ttl, _ := core.Cache.TTL(ctx, key)
				ttlSeconds := int64(-1)
				if ttl > 0 {
					ttlSeconds = int64(ttl.Seconds())
				}

				cacheInfo := models.CacheInfo{
					Key:       key,
					Value:     value,
					Type:      "string", // 简化处理
					TTL:       ttlSeconds,
					Size:      int64(len(value)),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				response.Success(c, cacheInfo)
				return
			}
		}

		// 从数据库中获取
		var cache models.CacheInfo
		if err := core.DB.GetDB().Where("key = ?", key).First(&cache).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "缓存不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询缓存失败")
			}
			return
		}

		response.Success(c, cache)
	}
}

// operateCache 操作缓存
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
func operateCache(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CacheOperateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		if core.Cache == nil {
			response.Error(c, response.ErrServerError, "缓存服务未初始化")
			return
		}

		var result interface{}
		var err error
		ctx := context.Background()

		switch req.Operation {
		case "get":
			var value string
			value, err = core.Cache.Get(ctx, req.Key)
			if err != nil {
				response.Error(c, response.ErrServerError, "获取缓存失败")
				return
			}
			result = value

		case "set":
			if req.Value == "" {
				response.Error(c, response.ErrBadRequest, "设置缓存值不能为空")
				return
			}

			var ttl time.Duration
			if req.TTL > 0 {
				ttl = time.Duration(req.TTL) * time.Second
			} else {
				ttl = time.Hour // 默认1小时
			}

			err = core.Cache.Set(ctx, req.Key, req.Value, ttl)
			if err != nil {
				response.Error(c, response.ErrServerError, "设置缓存失败")
				return
			}
			result = "设置成功"

		case "del":
			err = core.Cache.Delete(ctx, req.Key)
			if err != nil {
				response.Error(c, response.ErrServerError, "删除缓存失败")
				return
			}
			result = "删除成功"

		case "exists":
			exists, err := core.Cache.Exists(ctx, req.Key)
			if err != nil {
				response.Error(c, response.ErrServerError, "检查缓存存在性失败")
				return
			}
			result = map[string]interface{}{
				"exists": exists,
			}

		case "expire":
			if req.TTL <= 0 {
				response.Error(c, response.ErrBadRequest, "过期时间必须大于0")
				return
			}
			err = core.Cache.Expire(ctx, req.Key, time.Duration(req.TTL)*time.Second)
			if err != nil {
				response.Error(c, response.ErrServerError, "设置过期时间失败")
				return
			}
			result = "设置成功"

		case "ttl":
			// 获取Redis键的TTL需要额外实现，这里返回默认值
			// TODO: 实现Redis TTL方法
			result = map[string]interface{}{
				"ttl": -1, // -1表示永不过期
			}

		default:
			response.Error(c, response.ErrBadRequest, "不支持的操作")
			return
		}

		// 记录操作日志
		cacheInfo := models.CacheInfo{
			Key:   req.Key,
			Type:  req.Operation,
			Size:  int64(len(fmt.Sprintf("%v", result))),
		}

		if req.Value != "" {
			cacheInfo.Value = req.Value
		}

		if req.TTL > 0 {
			cacheInfo.TTL = req.TTL
		}

		// 异步保存到数据库，不影响响应
		go func() {
			core.DB.GetDB().Save(&cacheInfo)
		}()

		response.Success(c, result)
	}
}

// batchOperateCache 批量操作缓存
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
func batchOperateCache(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CacheBatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		if core.Cache == nil {
			response.Error(c, response.ErrServerError, "缓存服务未初始化")
			return
		}

		if len(req.Keys) == 0 {
			response.Error(c, response.ErrBadRequest, "缓存键列表不能为空")
			return
		}

		results := make(map[string]interface{})
		var err error
		ctx := context.Background()

		switch req.Operation {
		case "get":
			values := make(map[string]interface{})
			for _, key := range req.Keys {
				value, err := core.Cache.Get(ctx, key)
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
				err = core.Cache.Delete(ctx, key)
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
			response.Error(c, response.ErrBadRequest, "不支持的操作")
			return
		}

		response.Success(c, results)
	}
}

// clearCache 清空缓存
// @Summary 清空缓存
// @Description 清空所有缓存数据
// @Tags 缓存管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/cache/clear [post]
func clearCache(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		if core.Cache == nil {
			response.Error(c, response.ErrServerError, "缓存服务未初始化")
			return
		}

		// 清空缓存
		ctx := context.Background()
		err := core.Cache.FlushDB(ctx)
		if err != nil {
			response.Error(c, response.ErrServerError, "清空缓存失败")
			return
		}

		// 清空数据库记录
		if err := core.DB.GetDB().Exec("DELETE FROM sys_cache_info").Error; err != nil {
			// 记录错误但不影响响应
			fmt.Printf("清空缓存数据库记录失败: %v\n", err)
		}

		response.Success(c, gin.H{"message": "清空成功"})
	}
}

// getCacheStats 获取缓存统计
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
func getCacheStats(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CacheStatsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			req.Current = 1
			req.PageSize = 10
		}

		// 设置默认值
		if req.Current <= 0 {
			req.Current = 1
		}
		if req.PageSize <= 0 {
			req.PageSize = 10
		}

		// 尝试获取缓存统计信息
		var stats map[string]interface{}
		var err error

		if core.Cache != nil {
			// 获取缓存统计信息
			stats, err = getCacheStatistics(core.Cache)
		} else {
			// 如果缓存实例为空，使用默认统计信息
			stats = map[string]interface{}{
				"cacheType":    "redis",
				"keyspace_hits": 0,
				"keyspace_misses": 0,
				"hit_rate":     0.0,
				"used_memory":  0,
				"total_memory": 1024 * 1024 * 100, // 100MB默认总内存
				"key_count":    0,
			}
			err = nil
		}

		if err != nil {
			// 如果获取统计信息失败，仍然返回默认数据而不是错误
			stats = map[string]interface{}{
				"cacheType":    "redis",
				"keyspace_hits": 0,
				"keyspace_misses": 0,
				"hit_rate":     0.0,
				"used_memory":  0,
				"total_memory": 1024 * 1024 * 100,
				"key_count":    0,
			}
		}

		// 安全地构建统计对象，避免类型断言panic
		hitCount, _ := stats["keyspace_hits"].(int64)
		missCount, _ := stats["keyspace_misses"].(int64)
		hitRate, _ := stats["hit_rate"].(float64)
		usedMemory, _ := stats["used_memory"].(int64)
		keyCount, _ := stats["key_count"].(int64)

		currentStats := models.CacheStats{
			CacheType:   "redis",
			HitCount:    hitCount,
			MissCount:   missCount,
			HitRate:     hitRate,
			TotalMemory: usedMemory,
			UsedMemory:  usedMemory,
			KeyCount:    keyCount,
			CollectTime: time.Now(),
		}

		// 保存统计信息到数据库
		go func() {
			core.DB.GetDB().Create(&currentStats)
		}()

		// 转换为字符串以便JSON序列化
		response.Success(c, map[string]interface{}{
			"cacheType":   currentStats.CacheType,
			"hitCount":    currentStats.HitCount,
			"missCount":   currentStats.MissCount,
			"hitRate":     currentStats.HitRate,
			"totalMemory": currentStats.TotalMemory,
			"usedMemory":  currentStats.UsedMemory,
			"keyCount":    currentStats.KeyCount,
			"collectTime": currentStats.CollectTime,
		})
		return

		// 从数据库获取历史统计
	db := core.DB.GetDB().Model(&models.CacheStats{})

		if req.CacheType != "" {
			db = db.Where("cache_type = ?", req.CacheType)
		}
		if req.StartTime != nil && *req.StartTime != "" {
			db = db.Where("collect_time >= ?", *req.StartTime)
		}
		if req.EndTime != nil && *req.EndTime != "" {
			db = db.Where("collect_time <= ?", *req.EndTime)
		}

		// 统计总数
		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询缓存统计总数失败")
			return
		}

		// 查询分页数据
		var cacheStatsList []models.CacheStats
		offset := (req.Current - 1) * req.PageSize
		if err := db.Offset(offset).Limit(req.PageSize).Order("collect_time DESC").Find(&cacheStatsList).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询缓存统计失败")
			return
		}

		response.Page(c, cacheStatsList, total, req.Current, req.PageSize)
	}
}

// getCacheMonitor 获取缓存监控数据
// @Summary 获取缓存监控数据
// @Description 获取缓存实时监控数据，包括内存使用、键分布等
// @Tags 缓存管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 500 {object} response.Response
// @Router /monitor/cache/monitor [post]
func getCacheMonitor(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		if core.Cache == nil {
			response.Error(c, response.ErrServerError, "缓存服务未初始化")
			return
		}

		ctx := context.Background()

		// 获取真实的Redis统计信息
		stats, err := getRealCacheStats(core.Cache, ctx)
		if err != nil {
			// 如果获取失败，返回零值而不是错误
			stats = map[string]interface{}{
				"hitCount":    int64(0),
				"missCount":   int64(0),
				"hitRate":     0.0,
				"usedMemory":  int64(0),
				"totalMemory": int64(0),
				"keyCount":    int64(0),
			}
		}

		// 获取真实的键分布信息
		keyDistribution, err := getRealKeyDistribution(core.Cache, ctx)
		if err != nil {
			keyDistribution = map[string]int64{}
		}

		// 获取真实的内存使用趋势
		memoryTrend, err := getRealMemoryTrend(core.Cache, ctx)
		if err != nil {
			memoryTrend = []map[string]interface{}{}
		}

		// 获取真实的命中率趋势
		hitRateTrend, err := getRealHitRateTrend(core.Cache, ctx)
		if err != nil {
			hitRateTrend = []map[string]interface{}{}
		}

		// 获取Redis服务器信息
		serverInfo, err := getRedisServerInfo(core.Cache, ctx)
		if err != nil {
			serverInfo = map[string]interface{}{
				"connected": false,
				"type":      "redis",
				"version":   "unknown",
				"uptime":    "unknown",
			}
		}

		monitor := map[string]interface{}{
			"status":        serverInfo,
			"stats":         stats,
			"distribution":  keyDistribution,
			"memoryTrend":   memoryTrend,
			"hitRateTrend":  hitRateTrend,
		}

		response.Success(c, monitor)
	}
}

// exportCache 导出缓存数据
// @Summary 导出缓存数据
// @Description 导出缓存数据到JSON文件
// @Tags 缓存管理
// @Accept json
// @Produce json
// @Param request body CacheListRequest false "导出条件"
// @Success 200 {object} response.Response{data=string}
// @Failure 500 {object} response.Response
// @Router /monitor/cache/export [post]
func exportCache(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CacheListRequest
		c.ShouldBindJSON(&req)

		// 构建查询条件
		db := core.DB.GetDB().Model(&models.CacheInfo{})

		if req.Key != "" {
			db = db.Where("key LIKE ?", "%"+req.Key+"%")
		}
		if req.Type != "" {
			db = db.Where("type = ?", req.Type)
		}

		// 查询所有数据
		var caches []models.CacheInfo
		if err := db.Order("created_at DESC").Find(&caches).Error; err != nil {
			response.Error(c, response.ErrServerError, "导出缓存数据失败")
			return
		}

		// 转换为JSON
		data, err := json.MarshalIndent(caches, "", "  ")
		if err != nil {
			response.Error(c, response.ErrServerError, "序列化数据失败")
			return
		}

		// 生成文件名
		filename := fmt.Sprintf("cache_export_%s.json", time.Now().Format("20060102_150405"))

		response.Success(c, map[string]interface{}{
			"filename": filename,
			"data":     string(data),
			"count":    len(caches),
		})
	}
}

// getCachesFromCache 使用缓存接口获取缓存列表
func getCachesFromCache(cacheInterface interface {
	Get(ctx context.Context, key string) (string, error)
	Keys(ctx context.Context, pattern string) ([]string, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	Exists(ctx context.Context, key string) (bool, error)
}, req CacheListRequest) ([]models.CacheInfo, error) {
	ctx := context.Background()
	var caches []models.CacheInfo

	// 使用Keys方法获取所有匹配的键
	pattern := "*"
	if req.Key != "" {
		pattern = "*" + req.Key + "*"
	}

	keys, err := cacheInterface.Keys(ctx, pattern)
	if err != nil {
		// 如果获取键失败，返回空列表而不是错误
		return []models.CacheInfo{}, nil
	}

	for _, key := range keys {
		// 获取缓存值
		value, err := cacheInterface.Get(ctx, key)
		if err != nil {
			continue // 跳过获取失败的键
		}

		// 获取TTL
		ttl, _ := cacheInterface.TTL(ctx, key)
		ttlSeconds := int64(-1)
		if ttl > 0 {
			ttlSeconds = int64(ttl.Seconds())
		}

		// 检查是否存在
		exists, _ := cacheInterface.Exists(ctx, key)

		cacheInfo := models.CacheInfo{
			Key:       key,
			Value:     value,
			Type:      "string", // 简化处理，都作为字符串处理
			Size:      int64(len(value)),
			TTL:       ttlSeconds,
			CreatedAt: time.Now(), // 缓存接口不提供创建时间，使用当前时间
			UpdatedAt: time.Now(),
		}

		// 如果指定了类型过滤，只返回匹配的缓存（简化处理）
		if req.Type != "" && !strings.Contains(strings.ToLower(cacheInfo.Type), strings.ToLower(req.Type)) {
			continue
		}

		// 如果键不存在，跳过
		if !exists {
			continue
		}

		caches = append(caches, cacheInfo)
	}

	return caches, nil
}

// getRedisCaches 获取Redis缓存列表
func getRedisCaches(redisClient *cache.Redis, req CacheListRequest) ([]models.CacheInfo, error) {
	ctx := context.Background()
	var caches []models.CacheInfo

	// 使用SCAN命令遍历所有键（内存友好）
	cursor := uint64(0)
	pattern := "*"
	if req.Key != "" {
		pattern = "*" + req.Key + "*"
	}

	for {
		nextCursor, keys, err := redisClient.Scan(ctx, cursor, pattern, 100)
		if err != nil {
			return nil, fmt.Errorf("扫描Redis键失败: %w", err)
		}

		for _, key := range keys {
			// 获取键的类型
			keyType, err := redisClient.GetType(ctx, key)
			if err != nil {
				continue // 跳过获取失败的键
			}

			// 获取键的大小
			size, err := redisClient.GetSize(ctx, key)
			if err != nil {
				size = 0 // 获取失败时设为0
			}

			// 获取键的TTL
			ttl, err := redisClient.TTL(ctx, key)
			ttlSeconds := int64(-1)
			if err == nil && ttl > 0 {
				ttlSeconds = int64(ttl.Seconds())
			}

			// 获取键的值（仅对字符串类型获取完整值，其他类型获取摘要）
			var value string
			if keyType == "string" {
				if val, err := redisClient.GetRaw(ctx, key); err == nil {
					// 限制值长度，避免过大的值影响性能
					if len(val) > 1000 {
						value = val[:1000] + "..."
					} else {
						value = val
					}
				}
			} else {
				// 对于非字符串类型，显示类型信息
				if keyType == "list" {
					value = fmt.Sprintf("List[%d]", getRedisListLength(ctx, redisClient, key))
				} else if keyType == "hash" {
					value = fmt.Sprintf("Hash[%d]", getRedisHashLength(ctx, redisClient, key))
				} else if keyType == "set" {
					value = fmt.Sprintf("Set[%d]", getRedisSetLength(ctx, redisClient, key))
				} else if keyType == "zset" {
					value = fmt.Sprintf("ZSet[%d]", getRedisZSetLength(ctx, redisClient, key))
				} else {
					value = keyType
				}
			}

			// 如果指定了类型过滤，只返回匹配的缓存
			if req.Type != "" && !strings.Contains(strings.ToLower(keyType), strings.ToLower(req.Type)) {
				continue
			}

			cacheInfo := models.CacheInfo{
				Key:       key,
				Value:     value,
				Type:      keyType,
				Size:      size,
				TTL:       ttlSeconds,
				CreatedAt: time.Now(), // Redis不记录创建时间，使用当前时间
				UpdatedAt: time.Now(),
			}

			caches = append(caches, cacheInfo)
		}

		// 检查是否扫描完成
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return caches, nil
}

// getRedisListLength 获取Redis列表长度
func getRedisListLength(ctx context.Context, redisClient *cache.Redis, key string) int64 {
	length, err := redisClient.Client.LLen(ctx, key).Result()
	if err != nil {
		return 0
	}
	return length
}

// getRedisHashLength 获取Redis哈希长度
func getRedisHashLength(ctx context.Context, redisClient *cache.Redis, key string) int64 {
	length, err := redisClient.Client.HLen(ctx, key).Result()
	if err != nil {
		return 0
	}
	return length
}

// getRedisSetLength 获取Redis集合长度
func getRedisSetLength(ctx context.Context, redisClient *cache.Redis, key string) int64 {
	length, err := redisClient.Client.SCard(ctx, key).Result()
	if err != nil {
		return 0
	}
	return length
}

// getRedisZSetLength 获取Redis有序集合长度
func getRedisZSetLength(ctx context.Context, redisClient *cache.Redis, key string) int64 {
	length, err := redisClient.Client.ZCard(ctx, key).Result()
	if err != nil {
		return 0
	}
	return length
}

// getCacheStatistics 获取缓存统计信息
func getCacheStatistics(cacheInterface interface {
	Keys(ctx context.Context, pattern string) ([]string, error)
}) (map[string]interface{}, error) {
	ctx := context.Background()

	// 获取所有键的数量，如果出错则返回默认值
	keys, err := cacheInterface.Keys(ctx, "*")
	if err != nil {
		// 如果获取键失败，返回默认统计信息
		return map[string]interface{}{
			"cacheType":    "redis",
			"keyspace_hits": 0,
			"keyspace_misses": 0,
			"hit_rate":     0.0,
			"used_memory":  0,
			"total_memory": 1024 * 1024 * 100, // 100MB默认总内存
			"key_count":    0,
		}, nil
	}

	// 构建统计信息
	stats := map[string]interface{}{
		"cacheType":    "redis",
		"keyspace_hits": 0,    // 简化处理，使用默认值
		"keyspace_misses": 0,  // 简化处理，使用默认值
		"hit_rate":     0.0,   // 简化处理，使用默认值
		"used_memory":  int64(len(keys)) * 100, // 估算内存使用
		"total_memory": 1024 * 1024 * 100,     // 100MB默认总内存
		"key_count":    int64(len(keys)),
	}

	return stats, nil
}

// getRealCacheStats 获取真实的Redis缓存统计信息
func getRealCacheStats(cacheInterface interface {
	Keys(ctx context.Context, pattern string) ([]string, error)
}, ctx context.Context) (map[string]interface{}, error) {
	// 获取所有键的数量
	keys, err := cacheInterface.Keys(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("获取Redis键失败: %w", err)
	}

	// 尝试从缓存实例获取详细统计信息
	// 这里使用类型断言来检查是否支持统计信息获取
	stats := map[string]interface{}{
		"hitCount":    int64(0),
		"missCount":   int64(0),
		"hitRate":     0.0,
		"usedMemory":  int64(0),
		"totalMemory": int64(0),
		"keyCount":    int64(len(keys)),
	}

	// 尝试获取更详细的统计信息
	if cacheWithStats, ok := cacheInterface.(interface {
		GetStats(ctx context.Context) (map[string]interface{}, error)
	}); ok {
		detailedStats, err := cacheWithStats.GetStats(ctx)
		if err == nil {
			// 合并详细统计信息
			if hits, ok := detailedStats["keyspace_hits"].(int64); ok {
				stats["hitCount"] = hits
			}
			if misses, ok := detailedStats["keyspace_misses"].(int64); ok {
				stats["missCount"] = misses
			}
			if hitRate, ok := detailedStats["hit_rate"].(float64); ok {
				stats["hitRate"] = hitRate
			} else if hits, ok := stats["hitCount"].(int64); ok {
				if misses, ok := stats["missCount"].(int64); ok {
					if hits+misses > 0 {
						stats["hitRate"] = float64(hits) / float64(hits+misses) * 100
					}
				}
			}
			if usedMemory, ok := detailedStats["used_memory"].(int64); ok {
				stats["usedMemory"] = usedMemory
			}
			if totalMemory, ok := detailedStats["maxmemory"].(int64); ok {
				stats["totalMemory"] = totalMemory
			}
		}
	}

	return stats, nil
}

// getRealKeyDistribution 获取真实的键分布统计
func getRealKeyDistribution(cacheInterface interface {
	Keys(ctx context.Context, pattern string) ([]string, error)
}, ctx context.Context) (map[string]int64, error) {
	keys, err := cacheInterface.Keys(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("获取Redis键失败: %w", err)
	}

	distribution := make(map[string]int64)

	// 常见前缀分类
	prefixes := []string{"user:", "role:", "menu:", "config:", "session:", "token:", "cache:", "temp:", "sys:"}

	for _, key := range keys {
		keyLower := strings.ToLower(key)
		classified := false

		// 按前缀分类
		for _, prefix := range prefixes {
			if strings.HasPrefix(keyLower, prefix) {
				distribution[prefix]++
				classified = true
				break
			}
		}

		// 如果没有匹配前缀，归类为其他
		if !classified {
			distribution["other"]++
		}
	}

	return distribution, nil
}

// getRealMemoryTrend 获取真实的内存使用趋势
func getRealMemoryTrend(cacheInterface interface {
	Keys(ctx context.Context, pattern string) ([]string, error)
}, ctx context.Context) ([]map[string]interface{}, error) {
	// 获取当前内存使用情况
	keys, err := cacheInterface.Keys(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("获取Redis键失败: %w", err)
	}

	// 估算当前内存使用（基于键的数量和平均大小）
	estimatedMemory := int64(len(keys) * 100) // 假设每个键平均100字节

	// 由于Redis不保存历史内存使用数据，这里只能提供当前数据点
	trend := []map[string]interface{}{
		{"time": time.Now().Format("15:04"), "memory": estimatedMemory},
	}

	return trend, nil
}

// getRealHitRateTrend 获取真实的命中率趋势
func getRealHitRateTrend(cacheInterface interface {
	Keys(ctx context.Context, pattern string) ([]string, error)
}, ctx context.Context) ([]map[string]interface{}, error) {
	// 获取当前命中率统计
	stats, err := getRealCacheStats(cacheInterface, ctx)
	if err != nil {
		return nil, err
	}

	hitRate, ok := stats["hitRate"].(float64)
	if !ok {
		hitRate = 0.0
	}

	// 由于Redis不保存历史命中率数据，这里只能提供当前数据点
	trend := []map[string]interface{}{
		{"time": time.Now().Format("15:04"), "hitRate": hitRate},
	}

	return trend, nil
}

// getRedisServerInfo 获取Redis服务器信息
func getRedisServerInfo(cacheInterface interface {
	Keys(ctx context.Context, pattern string) ([]string, error)
}, ctx context.Context) (map[string]interface{}, error) {
	// 尝试获取详细的Redis服务器信息
	if cacheWithStats, ok := cacheInterface.(interface {
		GetStats(ctx context.Context) (map[string]interface{}, error)
	}); ok {
		stats, err := cacheWithStats.GetStats(ctx)
		if err == nil {
			serverInfo := map[string]interface{}{
				"connected": true,
				"type":      "redis",
			}

			// 尝试获取Redis版本信息
			if redisVer, ok := stats["redis_version"].(string); ok {
				serverInfo["version"] = redisVer
			} else {
				serverInfo["version"] = "unknown"
			}

			// 尝试获取运行时间信息
			if uptime, ok := stats["uptime_in_seconds"].(int64); ok {
				hours := uptime / 3600
				minutes := (uptime % 3600) / 60
				seconds := uptime % 60
				serverInfo["uptime"] = fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
			} else {
				serverInfo["uptime"] = "unknown"
			}

			return serverInfo, nil
		}
	}

	// 如果无法获取详细信息，返回基本状态
	return map[string]interface{}{
		"connected": true,
		"type":      "redis",
		"version":   "unknown",
		"uptime":    "unknown",
	}, nil
}