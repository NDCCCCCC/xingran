package v1

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ==================== 缓存操作 ====================

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
			// 静默处理错误，不影响响应
			_ = err
		}

		response.Success(c, gin.H{"message": "清空成功"})
	}
}
