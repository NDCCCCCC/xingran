package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// 缓存配置键常量
const (
	// 部门缓存时间配置
	CacheConfigDeptTree   = "cache.dept.tree"   // 部门树缓存时间（分钟）
	CacheConfigDeptList   = "cache.dept.list"   // 部门列表缓存时间（分钟）
	CacheConfigDeptSelect = "cache.dept.select" // 部门选择器缓存时间（分钟）

	// 角色缓存时间配置
	CacheConfigRoleMenus = "cache.role.menus" // 角色菜单缓存时间（分钟）
	CacheConfigRoleDepts = "cache.role.depts" // 角色部门缓存时间（分钟）

	// 字典缓存时间配置
	CacheConfigDictType = "cache.dict.type" // 字典类型缓存时间（分钟）
	CacheConfigDictData = "cache.dict.data" // 字典数据缓存时间（分钟）

	// 用户缓存时间配置
	CacheConfigUserList       = "cache.user.list"       // 用户列表缓存时间（分钟）
	CacheConfigUserByID       = "cache.user.byid"       // 用户详情缓存时间（分钟）
	CacheConfigUserByUsername = "cache.user.byusername" // 用户名查询缓存时间（分钟）
	CacheConfigUserByDept     = "cache.user.bydept"     // 用户部门缓存时间（分钟）
	CacheConfigUserRoles      = "cache.user.roles"      // 用户角色缓存时间（分钟）
	CacheConfigUserPosts      = "cache.user.posts"      // 用户岗位缓存时间（分钟）

	// 菜单缓存时间配置
	CacheConfigMenuTree   = "cache.menu.tree"   // 菜单树缓存时间（分钟）
	CacheConfigMenuRouter = "cache.menu.router" // 菜单路由缓存时间（分钟）
	CacheConfigMenuAll    = "cache.menu.all"    // 所有菜单缓存时间（分钟）

	// 岗位缓存时间配置
	CacheConfigPostAll     = "cache.post.all"     // 所有岗位缓存时间（分钟）
	CacheConfigPostEnabled = "cache.post.enabled" // 启用岗位缓存时间（分钟）

	// 系统配置缓存时间配置
	CacheConfigConfigAll   = "cache.config.all"   // 所有配置缓存时间（分钟）
	CacheConfigConfigByID  = "cache.config.byid"  // 配置详情缓存时间（分钟）
	CacheConfigConfigByKey = "cache.config.bykey" // 配置键查询缓存时间（分钟）

	// 用户设置缓存时间配置
	CacheConfigSettingsUser = "cache.settings.user" // 用户设置缓存时间（分钟）

	// 运维管理缓存时间配置
	CacheConfigFloorTree     = "cache.floor.tree"     // 楼层树缓存时间（分钟）
	CacheConfigFloorBuilding = "cache.floor.building" // 楼宇楼层列表缓存时间（分钟）

	// 知识库缓存时间配置
	CacheConfigKBCategory = "cache.kb.category" // 知识库分类树缓存时间（分钟）
	CacheConfigKBTags     = "cache.kb.tags"     // 知识库标签列表缓存时间（分钟）
	CacheConfigKBArticle  = "cache.kb.article"  // 知识库文章详情缓存时间（分钟）

	// 值班管理缓存时间配置
	CacheConfigDutyToday    = "cache.duty.today"    // 今日值班缓存时间（分钟）
	CacheConfigDutyMonthly  = "cache.duty.monthly"  // 月度排班缓存时间（分钟）
	CacheConfigDutyHolidays = "cache.duty.holidays" // 节假日数据缓存时间（分钟）

	// 工单管理缓存时间配置
	CacheConfigWorkorderMyPending  = "cache.workorder.my_pending" // 待办工单缓存时间（分钟）
	CacheConfigWorkorderStatistics = "cache.workorder.statistics" // 统计数据缓存时间（分钟）

	// 通知公告缓存时间配置
	CacheConfigNoticeMyNotices   = "cache.notice.my_notices"   // 我的通知列表缓存时间（分钟）
	CacheConfigNoticeUnreadCount = "cache.notice.unread_count" // 未读数量缓存时间（分钟）

	// 网络设备缓存时间配置
	CacheConfigNetworkDeviceStatistics = "cache.network_device.statistics" // 设备统计数据缓存时间（分钟）
	CacheConfigNetworkDeviceDept       = "cache.network_device.dept"       // 部门设备列表缓存时间（分钟）
	CacheConfigNetworkDeviceCredential = "cache.network_device.credential" // 凭证设备列表缓存时间（分钟）
)

// 限流阈值配置 (Phase 61 QUAL-03 / D-16)
// 键形态: rate_limit.{read|write|admin|default}.{per_minute|per_hour|per_day}
// 值语义: 请求次数(整数),非分钟;默认值与既有 rate_limiter.go 硬编码一致(D-17)
//
// 12 键清单(D-16):
//   rate_limit.read.per_minute     rate_limit.read.per_hour     rate_limit.read.per_day
//   rate_limit.write.per_minute    rate_limit.write.per_hour    rate_limit.write.per_day
//   rate_limit.admin.per_minute    rate_limit.admin.per_hour    rate_limit.admin.per_day
//   rate_limit.default.per_minute  rate_limit.default.per_hour  rate_limit.default.per_day
const (
	RateLimitReadPerMinute    = "rate_limit.read.per_minute"    // 默认 30,   Min 1, Max 10000
	RateLimitReadPerHour      = "rate_limit.read.per_hour"      // 默认 500,  Min 1, Max 100000
	RateLimitReadPerDay       = "rate_limit.read.per_day"       // 默认 5000, Min 1, Max 1000000
	RateLimitWritePerMinute   = "rate_limit.write.per_minute"   // 默认 100,  Min 1, Max 10000
	RateLimitWritePerHour     = "rate_limit.write.per_hour"     // 默认 1500, Min 1, Max 100000
	RateLimitWritePerDay      = "rate_limit.write.per_day"      // 默认 15000,Min 1, Max 1000000
	RateLimitAdminPerMinute   = "rate_limit.admin.per_minute"   // 默认 200,  Min 1, Max 10000
	RateLimitAdminPerHour     = "rate_limit.admin.per_hour"     // 默认 5000, Min 1, Max 100000
	RateLimitAdminPerDay      = "rate_limit.admin.per_day"      // 默认 50000,Min 1, Max 1000000
	RateLimitDefaultPerMinute = "rate_limit.default.per_minute" // 默认 120,  Min 1, Max 10000
	RateLimitDefaultPerHour   = "rate_limit.default.per_hour"   // 默认 2000, Min 1, Max 100000
	RateLimitDefaultPerDay    = "rate_limit.default.per_day"    // 默认 20000,Min 1, Max 1000000
)

// RateLimitProvider 限流配置提供者接口 (Phase 61 QUAL-03 / D-18 解耦 RateLimiter)
// *CacheConfigService 自动实现该接口(Go duck typing)
type RateLimitProvider interface {
	GetRateLimit(key string, defaultValue int) int
}

// CacheConfigService 缓存配置服务
// 从数据库读取缓存时间配置，支持动态调整缓存时间
//
// Phase 61 QUAL-03: 新增 rate_limit.* 配置项,与 cache.* 共存于同一 service(D-15);
// 遵循既有 LoadConfigs → setDefaultsIfNeeded → ReloadConfig 启动加载 + 手动刷新模式。
type CacheConfigService struct {
	db         *gorm.DB
	configs    map[string]time.Duration // cache.* 缓存时间配置
	rateLimits map[string]int           // rate_limit.* 限流阈值配置(次数语义, D-16)
	mu         sync.RWMutex
}

// NewCacheConfigService 创建缓存配置服务
func NewCacheConfigService(db *gorm.DB) *CacheConfigService {
	service := &CacheConfigService{
		db:         db,
		configs:    make(map[string]time.Duration),
		rateLimits: make(map[string]int),
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

	var cacheConfigs []models.Config
	if err := s.db.Where("config_key LIKE ?", "cache.%").Find(&cacheConfigs).Error; err != nil {
		return fmt.Errorf("查询缓存配置失败: %w", err)
	}

	// Phase 61 QUAL-03 / D-15: 沿用单占位符 LIKE 模式单独查询 rate_limit.*,
	// 与项目其他 sys_config 查询保持一致,失败错误信息分别报告
	var rateLimitConfigs []models.Config
	if err := s.db.Where("config_key LIKE ?", "rate_limit.%").Find(&rateLimitConfigs).Error; err != nil {
		return fmt.Errorf("查询限流配置失败: %w", err)
	}
	configs := append(cacheConfigs, rateLimitConfigs...)

	// 清空旧配置
	s.configs = make(map[string]time.Duration)
	s.rateLimits = make(map[string]int)

	// 获取配置信息（包含最小值、最大值、默认值）
	configInfo := s.GetConfigInfo()

	// 解析配置
	for _, config := range configs {
		// Phase 61 QUAL-03 / D-16: rate_limit.* 值是「次数」整数,无需 * time.Minute 转换
		if strings.HasPrefix(config.ConfigKey, "rate_limit.") {
			count, err := strconv.Atoi(config.ConfigValue)
			if err != nil {
				logger.Warnf("[CACHE_CONFIG] 解析限流配置失败 [%s=%s]: %v，将使用默认值", config.ConfigKey, config.ConfigValue, err)
				if info, ok := configInfo[config.ConfigKey]; ok {
					s.rateLimits[config.ConfigKey] = info.Default
					// 修复数据库中的值
					s.db.Model(&models.Config{}).Where("config_key = ?", config.ConfigKey).
						Update("config_value", strconv.Itoa(info.Default))
					logger.Infof("[CACHE_CONFIG] 修复限流配置: %s = %d 次", config.ConfigKey, info.Default)
				}
				continue
			}

			// 验证配置值是否在有效范围内
			if info, ok := configInfo[config.ConfigKey]; ok {
				if count < info.Min || count > info.Max {
					logger.Warnf("[CACHE_CONFIG] 限流配置值超出范围 [%s=%d]，范围：%d-%d，将重置为默认值",
						config.ConfigKey, count, info.Min, info.Max)
					s.rateLimits[config.ConfigKey] = info.Default
					// 修复数据库中的值
					s.db.Model(&models.Config{}).Where("config_key = ?", config.ConfigKey).
						Update("config_value", strconv.Itoa(info.Default))
					continue
				}
			}

			s.rateLimits[config.ConfigKey] = count
			logger.Infof("[CACHE_CONFIG] 加载限流配置: %s = %d 次", config.ConfigKey, count)
			continue
		}

		// 尝试解析为整数
		minutes, err := strconv.Atoi(config.ConfigValue)
		if err != nil {
			logger.Warnf("[CACHE_CONFIG] 解析配置失败 [%s=%s]: %v，将使用默认值", config.ConfigKey, config.ConfigValue, err)
			// 如果解析失败，检查是否有默认值，如果有则修复数据库
			if info, ok := configInfo[config.ConfigKey]; ok {
				var duration time.Duration
				// 特殊处理：未读通知数量缓存，默认值为 0 表示 30 秒
				if config.ConfigKey == CacheConfigNoticeUnreadCount && info.Default == 0 {
					duration = 30 * time.Second
				} else {
					duration = time.Duration(info.Default) * time.Minute
				}
				s.configs[config.ConfigKey] = duration
				// 修复数据库中的值
				s.db.Model(&models.Config{}).Where("config_key = ?", config.ConfigKey).
					Update("config_value", strconv.Itoa(info.Default))
				logger.Infof("[CACHE_CONFIG] 修复配置: %s = %d 分钟", config.ConfigKey, info.Default)
			}
			continue
		}

		// 特殊处理：未读通知数量缓存，值为 0 表示 30 秒
		var duration time.Duration
		if config.ConfigKey == CacheConfigNoticeUnreadCount && minutes == 0 {
			duration = 30 * time.Second
		} else {
			duration = time.Duration(minutes) * time.Minute
		}
		s.configs[config.ConfigKey] = duration

		// 验证配置值是否在有效范围内
		if info, ok := configInfo[config.ConfigKey]; ok {
			// 特殊处理：未读通知数量缓存的 0 值表示 30 秒，是有效的
			if config.ConfigKey == CacheConfigNoticeUnreadCount && minutes == 0 {
				// 跳过验证，duration 已经设置为 30 秒
			} else if minutes < info.Min || minutes > info.Max {
				logger.Warnf("[CACHE_CONFIG] 配置值超出范围 [%s=%d]，范围：%d-%d，将重置为默认值",
					config.ConfigKey, minutes, info.Min, info.Max)
				// 重新计算 duration
				if config.ConfigKey == CacheConfigNoticeUnreadCount && info.Default == 0 {
					duration = 30 * time.Second
				} else {
					duration = time.Duration(info.Default) * time.Minute
				}
				s.configs[config.ConfigKey] = duration
				// 修复数据库中的值
				s.db.Model(&models.Config{}).Where("config_key = ?", config.ConfigKey).
					Update("config_value", strconv.Itoa(info.Default))
				continue
			}
		}

		logger.Infof("[CACHE_CONFIG] 加载配置: %s = %v", config.ConfigKey, duration)
	}

	// 如果配置为空，使用默认值并写入数据库
	s.setDefaultsIfNeeded(ctx)

	logger.Infof("[CACHE_CONFIG] 配置加载完成，共 %d 项 cache + %d 项 rate_limit", len(s.configs), len(s.rateLimits))
	return nil
}

// setDefaultsIfNeeded 设置默认配置
func (s *CacheConfigService) setDefaultsIfNeeded(_ context.Context) {
	defaults := map[string]int{
		CacheConfigDeptTree:                30,
		CacheConfigDeptList:                30,
		CacheConfigDeptSelect:              30,
		CacheConfigRoleMenus:               30,
		CacheConfigRoleDepts:               30,
		CacheConfigDictType:                60,
		CacheConfigDictData:                30,
		CacheConfigUserList:                10,
		CacheConfigUserByID:                30,
		CacheConfigUserByUsername:          30,
		CacheConfigUserByDept:              10,
		CacheConfigUserRoles:               30,
		CacheConfigUserPosts:               30,
		CacheConfigMenuTree:                30,
		CacheConfigMenuRouter:              30,
		CacheConfigMenuAll:                 30,
		CacheConfigPostAll:                 30,
		CacheConfigPostEnabled:             30,
		CacheConfigConfigAll:               30,
		CacheConfigConfigByID:              30,
		CacheConfigConfigByKey:             30,
		CacheConfigSettingsUser:            15,
		CacheConfigFloorTree:               30,
		CacheConfigFloorBuilding:           15,
		CacheConfigKBCategory:              30,
		CacheConfigKBTags:                  30,
		CacheConfigKBArticle:               10,
		CacheConfigDutyToday:               5,
		CacheConfigDutyMonthly:             30,
		CacheConfigDutyHolidays:            60,
		CacheConfigWorkorderMyPending:      2,
		CacheConfigWorkorderStatistics:     5,
		CacheConfigNoticeMyNotices:         1,
		CacheConfigNoticeUnreadCount:       0, // 30秒
		CacheConfigNetworkDeviceStatistics: 3,
		CacheConfigNetworkDeviceDept:       5,
		CacheConfigNetworkDeviceCredential: 5,
	}

	for key, minutes := range defaults {
		if _, exists := s.configs[key]; !exists {
			// 特殊处理：0表示30秒
			var duration time.Duration
			if minutes == 0 && (key == CacheConfigNoticeUnreadCount) {
				duration = 30 * time.Second
			} else {
				duration = time.Duration(minutes) * time.Minute
			}
			s.configs[key] = duration

			// 同时写入数据库
			var count int64
			s.db.Model(&models.Config{}).Where("config_key = ?", key).Count(&count)
			if count == 0 {
				// 如果配置不存在，创建新配置
				config := models.Config{
					ConfigName:  s.getConfigName(key),
					ConfigKey:   key,
					ConfigValue: strconv.Itoa(minutes),
					ConfigType:  "Y",
					IsSystem:    1,
					Remark:      s.getConfigRemark(key),
				}
				s.db.Create(&config)
				logger.Infof("[CACHE_CONFIG] 创建默认配置: %s = %d 分钟", key, minutes)
			}
		}
	}

	// Phase 61 QUAL-03 / D-16/D-17: rate_limit.* 默认值与既有 rate_limiter.go
	// 硬编码一致(read=30/500/5000, write=100/1500/15000, admin=200/5000/50000,
	// default=120/2000/20000);值语义为次数(非分钟),独立写入 s.rateLimits
	rateLimitDefaults := map[string]int{
		RateLimitReadPerMinute:    30,
		RateLimitReadPerHour:      500,
		RateLimitReadPerDay:       5000,
		RateLimitWritePerMinute:   100,
		RateLimitWritePerHour:     1500,
		RateLimitWritePerDay:      15000,
		RateLimitAdminPerMinute:   200,
		RateLimitAdminPerHour:     5000,
		RateLimitAdminPerDay:      50000,
		RateLimitDefaultPerMinute: 120,
		RateLimitDefaultPerHour:   2000,
		RateLimitDefaultPerDay:    20000,
	}

	for key, count := range rateLimitDefaults {
		if _, exists := s.rateLimits[key]; !exists {
			s.rateLimits[key] = count

			// 同时写入数据库
			var cnt int64
			s.db.Model(&models.Config{}).Where("config_key = ?", key).Count(&cnt)
			if cnt == 0 {
				config := models.Config{
					ConfigName:  s.getConfigName(key),
					ConfigKey:   key,
					ConfigValue: strconv.Itoa(count),
					ConfigType:  "Y",
					IsSystem:    1,
					Remark:      s.getConfigRemark(key),
				}
				s.db.Create(&config)
				logger.Infof("[CACHE_CONFIG] 创建默认限流配置: %s = %d 次", key, count)
			}
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

// GetRateLimit 获取限流阈值配置 (Phase 61 QUAL-03 / D-18)
// key 形态: rate_limit.{scope}.{per_minute|per_hour|per_day};缺省返回 defaultValue
// 实现 RateLimitProvider 接口,供 RateLimiter 运行时读取(不缓存到 RateLimiter 内部,
// reload 后新请求即读到新阈值,D-19)
func (s *CacheConfigService) GetRateLimit(key string, defaultValue int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if v, ok := s.rateLimits[key]; ok {
		return v
	}
	return defaultValue
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
		CacheConfigUserByUsername: {
			Key:         CacheConfigUserByUsername,
			Name:        "用户名查询缓存时间",
			Description: "用户名查询数据的缓存时间（分钟）",
			Category:    "用户管理",
			Min:         5,
			Max:         120,
			Default:     30,
		},
		CacheConfigUserByDept: {
			Key:         CacheConfigUserByDept,
			Name:        "用户部门缓存时间",
			Description: "用户部门列表数据的缓存时间（分钟）",
			Category:    "用户管理",
			Min:         5,
			Max:         60,
			Default:     10,
		},
		CacheConfigUserRoles: {
			Key:         CacheConfigUserRoles,
			Name:        "用户角色缓存时间",
			Description: "用户角色数据的缓存时间（分钟）",
			Category:    "用户管理",
			Min:         5,
			Max:         120,
			Default:     30,
		},
		CacheConfigUserPosts: {
			Key:         CacheConfigUserPosts,
			Name:        "用户岗位缓存时间",
			Description: "用户岗位数据的缓存时间（分钟）",
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
		CacheConfigMenuAll: {
			Key:         CacheConfigMenuAll,
			Name:        "所有菜单缓存时间",
			Description: "所有菜单数据的缓存时间（分钟）",
			Category:    "菜单管理",
			Min:         5,
			Max:         120,
			Default:     30,
		},
		CacheConfigPostAll: {
			Key:         CacheConfigPostAll,
			Name:        "所有岗位缓存时间",
			Description: "所有岗位数据的缓存时间（分钟）",
			Category:    "岗位管理",
			Min:         5,
			Max:         120,
			Default:     30,
		},
		CacheConfigPostEnabled: {
			Key:         CacheConfigPostEnabled,
			Name:        "启用岗位缓存时间",
			Description: "启用岗位数据的缓存时间（分钟）",
			Category:    "岗位管理",
			Min:         5,
			Max:         120,
			Default:     30,
		},
		CacheConfigConfigAll: {
			Key:         CacheConfigConfigAll,
			Name:        "所有配置缓存时间",
			Description: "所有系统配置数据的缓存时间（分钟）",
			Category:    "系统配置",
			Min:         10,
			Max:         120,
			Default:     30,
		},
		CacheConfigConfigByID: {
			Key:         CacheConfigConfigByID,
			Name:        "配置详情缓存时间",
			Description: "单个配置详情数据的缓存时间（分钟）",
			Category:    "系统配置",
			Min:         5,
			Max:         60,
			Default:     30,
		},
		CacheConfigConfigByKey: {
			Key:         CacheConfigConfigByKey,
			Name:        "配置键查询缓存时间",
			Description: "按配置键查询数据的缓存时间（分钟）",
			Category:    "系统配置",
			Min:         5,
			Max:         60,
			Default:     30,
		},
		CacheConfigSettingsUser: {
			Key:         CacheConfigSettingsUser,
			Name:        "用户设置缓存时间",
			Description: "用户个人设置数据的缓存时间（分钟）",
			Category:    "用户设置",
			Min:         5,
			Max:         60,
			Default:     15,
		},
		CacheConfigFloorTree: {
			Key:         CacheConfigFloorTree,
			Name:        "楼层树缓存时间",
			Description: "楼层树结构数据的缓存时间（分钟）",
			Category:    "运维管理",
			Min:         5,
			Max:         60,
			Default:     30,
		},
		CacheConfigFloorBuilding: {
			Key:         CacheConfigFloorBuilding,
			Name:        "楼宇楼层列表缓存时间",
			Description: "按楼宇分组的楼层列表数据缓存时间（分钟）",
			Category:    "运维管理",
			Min:         5,
			Max:         60,
			Default:     15,
		},
		CacheConfigKBCategory: {
			Key:         CacheConfigKBCategory,
			Name:        "知识库分类树缓存时间",
			Description: "知识库分类树结构数据的缓存时间（分钟）",
			Category:    "知识库",
			Min:         5,
			Max:         60,
			Default:     30,
		},
		CacheConfigKBTags: {
			Key:         CacheConfigKBTags,
			Name:        "知识库标签列表缓存时间",
			Description: "知识库标签列表数据的缓存时间（分钟）",
			Category:    "知识库",
			Min:         5,
			Max:         60,
			Default:     30,
		},
		CacheConfigKBArticle: {
			Key:         CacheConfigKBArticle,
			Name:        "知识库文章详情缓存时间",
			Description: "知识库文章详情数据的缓存时间（分钟）",
			Category:    "知识库",
			Min:         5,
			Max:         30,
			Default:     10,
		},
		CacheConfigDutyToday: {
			Key:         CacheConfigDutyToday,
			Name:        "今日值班缓存时间",
			Description: "今日值班人员数据的缓存时间（分钟）",
			Category:    "值班管理",
			Min:         1,
			Max:         15,
			Default:     5,
		},
		CacheConfigDutyMonthly: {
			Key:         CacheConfigDutyMonthly,
			Name:        "月度排班缓存时间",
			Description: "月度值班排班数据的缓存时间（分钟）",
			Category:    "值班管理",
			Min:         5,
			Max:         60,
			Default:     30,
		},
		CacheConfigDutyHolidays: {
			Key:         CacheConfigDutyHolidays,
			Name:        "节假日数据缓存时间",
			Description: "节假日数据列表的缓存时间（分钟）",
			Category:    "值班管理",
			Min:         10,
			Max:         180,
			Default:     60,
		},
		CacheConfigWorkorderMyPending: {
			Key:         CacheConfigWorkorderMyPending,
			Name:        "待办工单缓存时间",
			Description: "用户待办工单列表数据的缓存时间（分钟）",
			Category:    "工单管理",
			Min:         1,
			Max:         10,
			Default:     2,
		},
		CacheConfigWorkorderStatistics: {
			Key:         CacheConfigWorkorderStatistics,
			Name:        "工单统计缓存时间",
			Description: "工单统计数据（各状态数量、待处理数量等）的缓存时间（分钟）",
			Category:    "工单管理",
			Min:         1,
			Max:         30,
			Default:     5,
		},
		CacheConfigNoticeMyNotices: {
			Key:         CacheConfigNoticeMyNotices,
			Name:        "我的通知缓存时间",
			Description: "用户通知列表数据的缓存时间（分钟）",
			Category:    "通知公告",
			Min:         0,
			Max:         5,
			Default:     1,
		},
		CacheConfigNoticeUnreadCount: {
			Key:         CacheConfigNoticeUnreadCount,
			Name:        "未读通知数量缓存时间",
			Description: "用户未读通知数量数据的缓存时间（分钟）",
			Category:    "通知公告",
			Min:         0,
			Max:         2,
			Default:     0, // 特殊：0表示30秒
		},
		CacheConfigNetworkDeviceStatistics: {
			Key:         CacheConfigNetworkDeviceStatistics,
			Name:        "设备统计缓存时间",
			Description: "网络设备统计数据（在线/离线/未知数量等）的缓存时间（分钟）",
			Category:    "网络设备",
			Min:         1,
			Max:         10,
			Default:     3,
		},
		CacheConfigNetworkDeviceDept: {
			Key:         CacheConfigNetworkDeviceDept,
			Name:        "部门设备列表缓存时间",
			Description: "按部门筛选的网络设备列表数据的缓存时间（分钟）",
			Category:    "网络设备",
			Min:         1,
			Max:         15,
			Default:     5,
		},
		CacheConfigNetworkDeviceCredential: {
			Key:         CacheConfigNetworkDeviceCredential,
			Name:        "凭证设备列表缓存时间",
			Description: "按凭证筛选的网络设备列表数据的缓存时间（分钟）",
			Category:    "网络设备",
			Min:         1,
			Max:         15,
			Default:     5,
		},
		// Phase 61 QUAL-03 / D-16: 限流阈值配置(次数语义,非分钟)
		RateLimitReadPerMinute: {
			Key:         RateLimitReadPerMinute,
			Name:        "读作用域每分钟限流",
			Description: "API Key 读作用域每分钟请求数限制（次）",
			Category:    "限流配置",
			Min:         1,
			Max:         10000,
			Default:     30,
		},
		RateLimitReadPerHour: {
			Key:         RateLimitReadPerHour,
			Name:        "读作用域每小时限流",
			Description: "API Key 读作用域每小时请求数限制（次）",
			Category:    "限流配置",
			Min:         1,
			Max:         100000,
			Default:     500,
		},
		RateLimitReadPerDay: {
			Key:         RateLimitReadPerDay,
			Name:        "读作用域每天限流",
			Description: "API Key 读作用域每天请求数限制（次）",
			Category:    "限流配置",
			Min:         1,
			Max:         1000000,
			Default:     5000,
		},
		RateLimitWritePerMinute: {
			Key:         RateLimitWritePerMinute,
			Name:        "写作用域每分钟限流",
			Description: "API Key 写作用域每分钟请求数限制（次）",
			Category:    "限流配置",
			Min:         1,
			Max:         10000,
			Default:     100,
		},
		RateLimitWritePerHour: {
			Key:         RateLimitWritePerHour,
			Name:        "写作用域每小时限流",
			Description: "API Key 写作用域每小时请求数限制（次）",
			Category:    "限流配置",
			Min:         1,
			Max:         100000,
			Default:     1500,
		},
		RateLimitWritePerDay: {
			Key:         RateLimitWritePerDay,
			Name:        "写作用域每天限流",
			Description: "API Key 写作用域每天请求数限制（次）",
			Category:    "限流配置",
			Min:         1,
			Max:         1000000,
			Default:     15000,
		},
		RateLimitAdminPerMinute: {
			Key:         RateLimitAdminPerMinute,
			Name:        "管理员作用域每分钟限流",
			Description: "API Key 管理员作用域每分钟请求数限制（次）",
			Category:    "限流配置",
			Min:         1,
			Max:         10000,
			Default:     200,
		},
		RateLimitAdminPerHour: {
			Key:         RateLimitAdminPerHour,
			Name:        "管理员作用域每小时限流",
			Description: "API Key 管理员作用域每小时请求数限制（次）",
			Category:    "限流配置",
			Min:         1,
			Max:         100000,
			Default:     5000,
		},
		RateLimitAdminPerDay: {
			Key:         RateLimitAdminPerDay,
			Name:        "管理员作用域每天限流",
			Description: "API Key 管理员作用域每天请求数限制（次）",
			Category:    "限流配置",
			Min:         1,
			Max:         1000000,
			Default:     50000,
		},
		RateLimitDefaultPerMinute: {
			Key:         RateLimitDefaultPerMinute,
			Name:        "默认作用域每分钟限流",
			Description: "API Key 默认作用域（InheritPerms 或无 scope）每分钟请求数限制（次）",
			Category:    "限流配置",
			Min:         1,
			Max:         10000,
			Default:     120,
		},
		RateLimitDefaultPerHour: {
			Key:         RateLimitDefaultPerHour,
			Name:        "默认作用域每小时限流",
			Description: "API Key 默认作用域（InheritPerms 或无 scope）每小时请求数限制（次）",
			Category:    "限流配置",
			Min:         1,
			Max:         100000,
			Default:     2000,
		},
		RateLimitDefaultPerDay: {
			Key:         RateLimitDefaultPerDay,
			Name:        "默认作用域每天限流",
			Description: "API Key 默认作用域（InheritPerms 或无 scope）每天请求数限制（次）",
			Category:    "限流配置",
			Min:         1,
			Max:         1000000,
			Default:     20000,
		},
	}
}

// ConfigInfo 配置信息
type ConfigInfo struct {
	Key         string // 配置键
	Name        string // 配置名称
	Description string // 配置说明
	Category    string // 配置分类
	Min         int    // 最小值（分钟）
	Max         int    // 最大值（分钟）
	Default     int    // 默认值（分钟）
}

// getConfigName 根据配置键获取配置名称
func (s *CacheConfigService) getConfigName(key string) string {
	configInfo := s.GetConfigInfo()
	if info, ok := configInfo[key]; ok {
		return info.Name
	}
	return key
}

// getConfigRemark 根据配置键获取配置说明
func (s *CacheConfigService) getConfigRemark(key string) string {
	configInfo := s.GetConfigInfo()
	if info, ok := configInfo[key]; ok {
		// Phase 61 QUAL-03: rate_limit.* 值语义为次数,非分钟
		if strings.HasPrefix(key, "rate_limit.") {
			return fmt.Sprintf("%s，默认%d次，范围%d-%d次", info.Description, info.Default, info.Min, info.Max)
		}
		return fmt.Sprintf("%s，默认%d分钟，范围%d-%d分钟", info.Description, info.Default, info.Min, info.Max)
	}
	return "缓存配置"
}
