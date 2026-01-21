package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// getAvailableEndpoints 获取可用的API端点列表
// @Summary 获取可用的API端点列表
// @Description 根据用户权限返回可访问的API端点，按模块分类
// @Tags Dashboard
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=services.CategoryEndpoints}
// @Router /dashboard/endpoints [get]
func getAvailableEndpoints(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户ID（由JWT中间件设置）
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		// 获取端点服务
		endpointService := core.APIEndpointService
		if endpointService == nil {
			response.Error(c, response.ErrServerError, "端点服务未初始化")
			return
		}

		endpoints, err := endpointService.GetUserAccessibleEndpoints(c.Request.Context(), userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"categories": endpoints,
			"total":      len(endpoints),
		})
	}
}

// validateEndpoint 验证API端点配置
// @Summary 验证API端点配置
// @Description 验证指定的API端点是否存在
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param route query string true "端点路径"
// @Param method query string true "请求方法"
// @Success 200 {object} response.Response{data=services.EndpointDetail}
// @Router /dashboard/endpoints/validate [get]
func validateEndpoint(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		route := c.Query("route")
		method := c.Query("method")

		if route == "" || method == "" {
			response.Error(c, response.ErrParamError)
			return
		}

		// 获取端点服务
		endpointService := core.APIEndpointService
		if endpointService == nil {
			response.Error(c, response.ErrServerError, "端点服务未初始化")
			return
		}

		endpoint, err := endpointService.ValidateEndpoint(route, method)
		if err != nil {
			response.Error(c, response.ErrNotFound)
			return
		}

		response.Success(c, endpoint)
	}
}

// getUserEndpointsWithFilter 获取过滤后的端点列表
// @Summary 获取过滤后的端点列表
// @Description 根据Widget类型过滤可用的API端点
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param widgetType query string false "Widget类型"
// @Success 200 {object} response.Response{data=[]services.CategoryEndpoints}
// @Router /dashboard/endpoints/filter [get]
func getUserEndpointsWithFilter(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		widgetType := c.Query("widgetType")

		// 获取端点服务
		endpointService := core.APIEndpointService
		if endpointService == nil {
			response.Error(c, response.ErrServerError, "端点服务未初始化")
			return
		}

		categories, err := endpointService.GetUserAccessibleEndpoints(c.Request.Context(), userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 如果指定了Widget类型，过滤端点
		if widgetType != "" {
			filteredCategories := make([]services.CategoryEndpoints, 0)
			for _, category := range categories {
				filteredEndpoints := make([]services.EndpointDetail, 0)
				for _, endpoint := range category.Endpoints {
					// 检查是否支持该Widget类型
					for _, supported := range endpoint.SupportedWidgets {
						if supported == widgetType {
							filteredEndpoints = append(filteredEndpoints, endpoint)
							break
						}
					}
				}
				if len(filteredEndpoints) > 0 {
					filteredCategories = append(filteredCategories, services.CategoryEndpoints{
						Module:    category.Module,
						Category:  category.Category,
						Icon:      category.Icon,
						Endpoints: filteredEndpoints,
					})
				}
			}
			categories = filteredCategories
		}

		response.Success(c, gin.H{
			"categories": categories,
			"total":      len(categories),
		})
	}
}

// invalidateEndpointCache 清除用户端点缓存
// @Summary 清除用户端点缓存
// @Description 清除当前用户的端点列表缓存
// @Tags Dashboard
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /dashboard/endpoints/cache/invalidate [post]
func invalidateEndpointCache(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		// 获取端点服务
		endpointService := core.APIEndpointService
		if endpointService == nil {
			response.Error(c, response.ErrServerError, "端点服务未初始化")
			return
		}

		endpointService.InvalidateUserCache(c.Request.Context(), userID.(string))

		response.Success(c, gin.H{
			"message": "缓存已清除",
		})
	}
}
