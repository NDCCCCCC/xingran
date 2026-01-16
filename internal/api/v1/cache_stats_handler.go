package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ==================== 缓存统计和监控 ====================

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

		// 判断是否为实时统计查询（有 IsRealtime 标志或未提供时间范围）
		isRealtime := req.IsRealtime || (req.StartTime == nil && req.EndTime == nil)

		// 实时统计：从缓存获取
		if isRealtime {
			if core.Cache == nil {
				response.Error(c, response.ErrServerError, "缓存服务不可用")
				return
			}

			// 类型断言为支持 GetStats 的缓存
			cacheWithStats, ok := core.Cache.(interface {
				GetStats(ctx context.Context) (map[string]interface{}, error)
			})
			if !ok {
				response.Error(c, response.ErrServerError, "缓存不支持统计信息")
				return
			}

			// 获取多级缓存统计信息
			stats, err := cacheWithStats.GetStats(context.Background())
			if err != nil {
				response.Error(c, response.ErrServerError, "获取缓存统计失败")
				return
			}

			// 格式化为前端需要的格式
			result := make(map[string]interface{})

			if l1Stats, ok := stats["l1"].(map[string]interface{}); ok {
				result["l1"] = formatCacheStats(l1Stats, "L1(内存)")
			}

			if l2Stats, ok := stats["l2"].(map[string]interface{}); ok {
				result["l2"] = formatCacheStats(l2Stats, "L2(Redis)")
			}

			response.Success(c, result)
			return
		}

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
// @Description 获取缓存实时监控数据，包括L1和L2的统计信息
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

		// 类型断言为支持 GetStats 的缓存
		cacheWithStats, ok := core.Cache.(interface {
			GetStats(ctx context.Context) (map[string]interface{}, error)
		})
		if !ok {
			response.Error(c, response.ErrServerError, "缓存不支持统计信息")
			return
		}

		// 获取多级缓存统计信息
		stats, err := cacheWithStats.GetStats(context.Background())
		if err != nil {
			response.Error(c, response.ErrServerError, "获取监控数据失败")
			return
		}

		monitor := make(map[string]interface{})

		// L1 监控数据
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

		// L2 监控数据
		if l2Stats, ok := stats["l2"].(map[string]interface{}); ok {
			formattedL2Stats := formatCacheStatsForMonitor(l2Stats)

			// 获取Redis服务器信息
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
