package v1

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// testCacheEndpoint 测试缓存端点
// @Summary 测试缓存功能
// @Description 用于测试缓存功能是否正常工作
// @Tags 测试
// @Accept json
// @Produce json
// @Router /test/cache [get]
func testCacheEndpoint(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("[TEST] 测试缓存端点被调用")

		if core.DeptService == nil {
			fmt.Println("[TEST] DeptService 为 nil")
			response.Error(c, response.ErrServerError, "DeptService 未初始化")
			return
		}

		fmt.Println("[TEST] DeptService 已初始化，开始调用 GetDeptTreeWithCache")

		// 调用带缓存的方法
		depts, err := core.DeptService.GetDeptTreeWithCache(context.Background(), false)
		if err != nil {
			fmt.Printf("[TEST] 获取部门树失败: %v\n", err)
			response.Error(c, response.ErrServerError, "获取部门树失败: "+err.Error())
			return
		}

		fmt.Printf("[TEST] 获取部门树成功，数量: %d\n", len(depts))

		// 等待异步写入
		// time.Sleep(1 * time.Second)

		// 检查缓存键
		keys, err := core.Cache.Keys(context.Background(), "dept:*")
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
}
