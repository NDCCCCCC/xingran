package v1

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

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

// getCacheConfigs 获取缓存配置列表
// @Summary 获取缓存配置列表
// @Description 获取所有缓存配置信息，按分类组织
// @Tags 缓存配置
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]CacheConfigCategory}
// @Router /monitor/cache/config [get]
func getCacheConfigs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		if core.CacheConfigService == nil {
			response.Error(c, response.ErrServerError, "缓存配置服务未初始化")
			return
		}

		// 获取所有配置信息
		configInfoMap := core.CacheConfigService.GetConfigInfo()
		currentValues := core.CacheConfigService.GetAllConfigs(c.Request.Context())

		// 按分类组织配置
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

		// 转换为响应格式
		var categories []CacheConfigCategory
		for name, configs := range categoryMap {
			categories = append(categories, CacheConfigCategory{
				Name:    name,
				Configs: configs,
			})
		}

		response.Success(c, categories)
	}
}

// updateCacheConfig 更新缓存配置
// @Summary 更新缓存配置
// @Description 更新指定缓存项的时间配置
// @Tags 缓存配置
// @Accept json
// @Produce json
// @Param request body UpdateCacheConfigRequest true "更新缓存配置请求"
// @Success 200 {object} response.Response
// @Router /monitor/cache/config [put]
func updateCacheConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		if core.CacheConfigService == nil {
			response.Error(c, response.ErrServerError, "缓存配置服务未初始化")
			return
		}

		var req UpdateCacheConfigRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 验证配置键是否有效
		configInfoMap := core.CacheConfigService.GetConfigInfo()
		configInfo, exists := configInfoMap[req.Key]
		if !exists {
			response.Error(c, response.ErrBadRequest, "无效的配置键")
			return
		}

		// 验证值范围
		if req.Value < configInfo.Min || req.Value > configInfo.Max {
			response.Error(c, response.ErrBadRequest,
				fmt.Sprintf("配置值必须在 %d 到 %d 分钟之间", configInfo.Min, configInfo.Max))
			return
		}

		// 更新配置到数据库
		var config models.Config
		if err := core.DB.GetDB().Where("config_key = ?", req.Key).First(&config).Error; err != nil {
			// 如果配置不存在，创建新配置
			config = models.Config{
				ConfigKey:   req.Key,
				ConfigName:  configInfo.Name,
				ConfigValue: strconv.Itoa(req.Value),
				ConfigType:  models.ConfigTypeYes,
				Remark:      configInfo.Description,
			}
			if err := core.DB.GetDB().Create(&config).Error; err != nil {
				response.Error(c, response.ErrServerError, "创建配置失败")
				return
			}
		} else {
			// 更新现有配置
			config.ConfigValue = strconv.Itoa(req.Value)
			if err := core.DB.GetDB().Save(&config).Error; err != nil {
				response.Error(c, response.ErrServerError, "更新配置失败")
				return
			}
		}

		// 重新加载配置
		if err := core.CacheConfigService.ReloadConfig(c.Request.Context()); err != nil {
			response.Error(c, response.ErrServerError, "重新加载配置失败")
			return
		}

		response.Success(c, gin.H{"message": "配置更新成功"})
	}
}

// reloadCacheConfigs 重新加载缓存配置
// @Summary 重新加载缓存配置
// @Description 从数据库重新加载所有缓存配置
// @Tags 缓存配置
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /monitor/cache/config/reload [post]
func reloadCacheConfigs(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		if core.CacheConfigService == nil {
			response.Error(c, response.ErrServerError, "缓存配置服务未初始化")
			return
		}

		if err := core.CacheConfigService.ReloadConfig(c.Request.Context()); err != nil {
			response.Error(c, response.ErrServerError, "重新加载配置失败: "+err.Error())
			return
		}

		response.Success(c, gin.H{"message": "配置重新加载成功"})
	}
}

// UpdateCacheConfigRequest 更新缓存配置请求
type UpdateCacheConfigRequest struct {
	Key   string `json:"key" binding:"required"`                 // 配置键
	Value int    `json:"value" binding:"required,min=1,max=180"` // 配置值（分钟）
}
