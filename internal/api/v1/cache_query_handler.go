package v1

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

// ==================== 缓存查询操作 ====================

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
		if req.Level == "" {
			req.Level = "all" // 默认显示两级缓存
		}

		// 如果有Redis实例，从Redis获取实时数据
		if core.Cache != nil {
			caches, err := getCachesFromCacheWithLevel(core.Cache, req)
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
