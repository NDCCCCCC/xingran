package services

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// 缓存配置键常量
const (
	// 部门缓存时间配置
	CacheConfigDeptTree      = "cache.dept.tree"       // 部门树缓存时间（分钟）
	CacheConfigDeptList      = "cache.dept.list"       // 部门列表缓存时间（分钟）
	CacheConfigDeptSelect    = "cache.dept.select"     // 部门选择器缓存时间（分钟）

	// 角色缓存时间配置
	CacheConfigRoleMenus     = "cache.role.menus"      // 角色菜单缓存时间（分钟）
	CacheConfigRoleDepts     = "cache.role.depts"      // 角色部门缓存时间（分钟）

	// 字典缓存时间配置
	CacheConfigDictType      = "cache.dict.type"       // 字典类型缓存时间（分钟）
	CacheConfigDictData      = "cache.dict.data"       // 字典数据缓存时间（分钟）

	// 用户缓存时间配置
	CacheConfigUserList      = "cache.user.list"       // 用户列表缓存时间（分钟）
	CacheConfigUserByID      = "cache.user.byid"       // 用户详情缓存时间（分钟）

	// 菜单缓存时间配置
	CacheConfigMenuTree      = "cache.menu.tree"       // 菜单树缓存时间（分钟）
	CacheConfigMenuRouter    = "cache.menu.router"     // 菜单路由缓存时间（分钟）
)

// CacheConfigService 缓存配置服务
// 从数据库读取缓存时间配置，支持动态调整缓存时间
type CacheConfigService struct {
	db        *gorm.DB
	configs   map[string]time.Duration
	mu        sync.RWMutex
}

// NewCacheConfigService 创建缓存配置服务
func NewCacheConfigService(db *gorm.DB) *CacheConfigService {
	service := &CacheConfigService{
		db:      db,
		configs: make(map[string]time.Duration),
	}

	// 加载配置
	if err := service.LoadConfigs(context.Background()); err != nil {
		logger.Warnf("[CACHE_CONFIG] 初始化加载配置失败: %v", err)
	}

	return service
}

// LoadConfigs 从数据库加载缓存配置
func (s *CacheConfigService) LoadConfigs(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var configs []models.Config
	if err := s.db.Where("config_key LIKE ?", "cache.%").Find(&configs).Error; err != nil {
		return fmt.Errorf("查询缓存配置失败: %w", err)
	}

	// 清空旧配置
	s.configs = make(map[string]time.Duration)

	// 解析配置
	for _, config := range configs {
		minutes, err := strconv.Atoi(config.ConfigValue)
		if err != nil {
			logger.Warnf("[CACHE_CONFIG] 解析配置失败 [%s=%s]: %v", config.ConfigKey, config.ConfigValue, err)
			continue
		}

		s.configs[config.ConfigKey] = time.Duration(minutes) * time.Minute
		logger.Infof("[CACHE_CONFIG] 加载配置: %s = %d 分钟", config.ConfigKey, minutes)
	}

	// 如果配置为空，使用默认值
	s.setDefaultsIfNeeded()

	logger.Infof("[CACHE_CONFIG] 配置加载完成，共 %d 项", len(s.configs))
	return nil
}

// setDefaultsIfNeeded 设置默认配置
func (s *CacheConfigService) setDefaultsIfNeeded() {
	defaults := map[string]int{
		CacheConfigDeptTree:   30,
		CacheConfigDeptList:   30,
		CacheConfigDeptSelect: 30,
		CacheConfigRoleMenus:  30,
		CacheConfigRoleDepts:  30,
		CacheConfigDictType:   60,
		CacheConfigDictData:   30,
		CacheConfigUserList:   10,
		CacheConfigUserByID:   30,
		CacheConfigMenuTree:   30,
		CacheConfigMenuRouter: 30,
	}

	for key, minutes := range defaults {
		if _, exists := s.configs[key]; !exists {
			s.configs[key] = time.Duration(minutes) * time.Minute
		}
	}
}

// GetDuration 获取缓存时间配置
func (s *CacheConfigService) GetDuration(configKey string) time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if duration, ok := s.configs[configKey]; ok {
		return duration
	}

	// 默认返回 30 分钟
	return 30 * time.Minute
}

// GetDurationWithDefault 获取缓存时间配置，支持自定义默认值
func (s *CacheConfigService) GetDurationWithDefault(configKey string, defaultDuration time.Duration) time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if duration, ok := s.configs[configKey]; ok {
		return duration
	}

	return defaultDuration
}

// ReloadConfig 重新加载配置
func (s *CacheConfigService) ReloadConfig(ctx context.Context) error {
	logger.Infof("[CACHE_CONFIG] 重新加载配置")
	return s.LoadConfigs(ctx)
}

// GetAllConfigs 获取所有缓存配置（用于前端展示）
func (s *CacheConfigService) GetAllConfigs(ctx context.Context) map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]int)
	for key, duration := range s.configs {
		result[key] = int(duration.Minutes())
	}

	return result
}

// GetConfigInfo 获取配置信息（包含名称和说明）
func (s *CacheConfigService) GetConfigInfo() map[string]ConfigInfo {
	return map[string]ConfigInfo{
		CacheConfigDeptTree: {
			Key:         CacheConfigDeptTree,
			Name:        "部门树缓存时间",
			Description: "部门树结构数据的缓存时间（分钟）",
			Category:    "部门管理",
			Min:         5,
			Max:         120,
			Default:     30,
		},
		CacheConfigDeptList: {
			Key:         CacheConfigDeptList,
			Name:        "部门列表缓存时间",
			Description: "部门列表数据的缓存时间（分钟）",
			Category:    "部门管理",
			Min:         5,
			Max:         120,
			Default:     30,
		},
		CacheConfigDeptSelect: {
			Key:         CacheConfigDeptSelect,
			Name:        "部门选择器缓存时间",
			Description: "部门选择器数据的缓存时间（分钟）",
			Category:    "部门管理",
			Min:         5,
			Max:         120,
			Default:     30,
		},
		CacheConfigRoleMenus: {
			Key:         CacheConfigRoleMenus,
			Name:        "角色菜单缓存时间",
			Description: "角色菜单权限数据的缓存时间（分钟）",
			Category:    "角色管理",
			Min:         5,
			Max:         120,
			Default:     30,
		},
		CacheConfigRoleDepts: {
			Key:         CacheConfigRoleDepts,
			Name:        "角色部门缓存时间",
			Description: "角色部门权限数据的缓存时间（分钟）",
			Category:    "角色管理",
			Min:         5,
			Max:         120,
			Default:     30,
		},
		CacheConfigDictType: {
			Key:         CacheConfigDictType,
			Name:        "字典类型缓存时间",
			Description: "字典类型数据的缓存时间（分钟）",
			Category:    "字典管理",
			Min:         10,
			Max:         180,
			Default:     60,
		},
		CacheConfigDictData: {
			Key:         CacheConfigDictData,
			Name:        "字典数据缓存时间",
			Description: "字典数据内容的缓存时间（分钟）",
			Category:    "字典管理",
			Min:         5,
			Max:         120,
			Default:     30,
		},
		CacheConfigUserList: {
			Key:         CacheConfigUserList,
			Name:        "用户列表缓存时间",
			Description: "用户列表数据的缓存时间（分钟）",
			Category:    "用户管理",
			Min:         5,
			Max:         60,
			Default:     10,
		},
		CacheConfigUserByID: {
			Key:         CacheConfigUserByID,
			Name:        "用户详情缓存时间",
			Description: "用户详情数据的缓存时间（分钟）",
			Category:    "用户管理",
			Min:         5,
			Max:         120,
			Default:     30,
		},
		CacheConfigMenuTree: {
			Key:         CacheConfigMenuTree,
			Name:        "菜单树缓存时间",
			Description: "菜单树结构数据的缓存时间（分钟）",
			Category:    "菜单管理",
			Min:         5,
			Max:         120,
			Default:     30,
		},
		CacheConfigMenuRouter: {
			Key:         CacheConfigMenuRouter,
			Name:        "菜单路由缓存时间",
			Description: "菜单路由数据的缓存时间（分钟）",
			Category:    "菜单管理",
			Min:         5,
			Max:         120,
			Default:     30,
		},
	}
}

// ConfigInfo 配置信息
type ConfigInfo struct {
	Key         string        // 配置键
	Name        string        // 配置名称
	Description string        // 配置说明
	Category    string        // 配置分类
	Min         int           // 最小值（分钟）
	Max         int           // 最大值（分钟）
	Default     int           // 默认值（分钟）
}
