// Package core 提供系统的核心功能模块
// 包括数据库管理、缓存管理、JWT认证、密码加密等核心服务的初始化和管理
package core

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/core/security"
	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/scheduler"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	assetSvc "github.com/xingran-next/xingran-go-backend/internal/services/asset"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/services/vdi"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
	"gorm.io/gorm"
)

// rpaDefaultScaleDownCooldown RPA 缩容冷却时间 fallback
// 在配置项缺失或解析失败时使用，避免散落的硬编码字面量
const rpaDefaultScaleDownCooldown = 5 * time.Minute

// fallbackMemoryCacheSize 核心初始化失败时使用的内存缓存容量 fallback
// 在Redis不可用或配置缺失时作为L1缓存的兜底容量
const fallbackMemoryCacheSize = 1000

// fallbackMemoryCacheCleanup 核心初始化失败时使用的内存缓存清理间隔 fallback
// 在Redis不可用或配置缺失时作为L1缓存的兜底清理周期
const fallbackMemoryCacheCleanup = 10 * time.Minute

// loadConnectionPoolConfig 从 sys_config 读取网络设备连接池配置 (web 可配)。
// 任一项缺失/解析失败均用默认值兜底, 保证启动不崩溃。
//
//   - network.connection_pool.max_connections (默认 50)
//   - network.connection_pool.max_idle_seconds (默认 300)
//
// 历史: core.go 曾硬编码 MaxConnections=20 (避 scrapligo panic); 设备数 >20 时池满导致
// 间歇性端口采集跳过, 已移入 sys_config 可配 (migration 203 seed)。修改后需重启后端生效。
func loadConnectionPoolConfig(db *gorm.DB) device.PoolConfig {
	const (
		defaultMaxConnections = 50
		defaultMaxIdleSeconds = 300
	)
	cfg := device.PoolConfig{
		MaxConnections: defaultMaxConnections,
		MaxIdle:        time.Duration(defaultMaxIdleSeconds) * time.Second,
	}

	readInt := func(key string, fallback int) int {
		var v string
		if err := db.Raw("SELECT config_value FROM sys_config WHERE config_key = ? AND deleted_at IS NULL", key).Scan(&v).Error; err != nil || v == "" {
			return fallback
		}
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return fallback
		}
		return n
	}

	cfg.MaxConnections = readInt("network.connection_pool.max_connections", defaultMaxConnections)
	cfg.MaxIdle = time.Duration(readInt("network.connection_pool.max_idle_seconds", defaultMaxIdleSeconds)) * time.Second

	applogger.Infof("[连接池] 配置加载: MaxConnections=%d, MaxIdle=%v", cfg.MaxConnections, cfg.MaxIdle)
	return cfg
}

// rpaScalingService RPA扩缩容服务最小接口
// 在 core 包内定义以避免循环依赖，rpa.ScalingService 自动满足此接口
type rpaScalingService interface {
	Stop()
}

// warmUpServices 缓存预热所需的服务集合（仅在初始化期间使用）
type warmUpServices struct {
	UserService system.UserService
	RoleService system.RoleService
	MenuService system.MenuService
	DeptService system.DepartmentService
	PostService system.PostService
}

// Core 核心模块管理器
// 负责管理系统的所有核心服务组件，包括数据库连接、缓存、JWT管理、密码加密等。
//
// 通过嵌入 *CoreInfra 和 *CoreServices 保持向后兼容的字段访问语法
// （core.DB、core.Cache、core.UserService、core.NoticeHub 等仍可直接访问 — 字段提升）。
//
// 字段分组：
//   - 基础设施层：见 core_infra.go（Config、DB、Cache、JWT、Pwd、SM4、Scheduler、验证码、指标缓存、认证工厂）
//   - 业务服务层：见 core_services.go（设备、通知、日志、缓存服务、RPA、API 元数据、分区）
type Core struct {
	*CoreInfra
	*CoreServices
}

// GetDB 获取数据库连接
// 返回底层的GORM数据库连接实例，供业务层使用
func (c *Core) GetDB() *gorm.DB {
	if c.DB != nil {
		return c.DB.GetDB()
	}
	return nil
}

// New 创建核心模块管理器
// 根据配置创建Core实例，初始化JWT管理器和密码管理器
func New(cfg *config.Config) (*Core, error) {
	sm4Cipher, err := initSM4Cipher(cfg.Security.SM4Key)
	if err != nil {
		return nil, fmt.Errorf("初始化 SM4 加密器失败: %w", err)
	}

	// 初始化JWT管理器
	jwtManager, err := security.NewJWTManager(&cfg.JWT)
	if err != nil {
		return nil, fmt.Errorf("初始化JWT管理器失败: %w", err)
	}

	// 初始化 CoreInfra 与 CoreServices，确保 struct embedding 字段提升能正常工作
	infra := &CoreInfra{
		Config:     cfg,
		JWTManager: jwtManager,
		PwdManager: security.NewPasswordManager(nil),
		SM4Cipher:  sm4Cipher,
	}
	svcs := &CoreServices{}

	// Phase 35 OPERLOG-01: wire the configured operlog exclude list before
	// any handler can call operlog.Record / RecordWithBody. This MUST happen
	// here (not in Init) because the call has no dependency on DB/Cache/JWT
	// and we want heartbeat endpoints excluded from the very first request.
	operlog.Configure(cfg.OperLog.ExcludePaths)

	return &Core{
		CoreInfra:    infra,
		CoreServices: svcs,
	}, nil
}

// initSM4Cipher 初始化SM4加密器
// 使用配置中的SM4密钥创建加密器，用于AD域密码加解密
//
// 启动校验策略（安全与可用性平衡）：
//   - SM4_KEY 未配置 → 启动中止（无 cipher 启动后 AD 同步会静默失败）
//   - SM4_KEY 是仓库默认值 dGVzdC1zZWNyZXQxNiEhIQ== → ⚠️ **仅 ERROR 日志,允许启动**
//     原因: 数据库里可能已用默认值加密大量数据,硬性中止会让所有历史数据无法读取
//     建议: 生成新 key (openssl rand -base64 16) → 用 migrate-sm4-key 命令迁移
//     数据 → 切换到新 key → 移除默认值
//   - 密钥错误无法创建 cipher → 启动中止
//
// ⚠️ SM4 加密的是持久化数据（设备密码/AD 凭据/RPA 凭证）,
//
//	启动后**不能**动态生成,否则重启后历史加密数据永久不可解密
func initSM4Cipher(sm4Key string) (addomain.PasswordCipher, error) {
	if sm4Key == "" {
		return nil, fmt.Errorf("SM4_KEY 未配置 — AD 域密码加解密将不可用,启动中止 (通过环境变量 SM4_KEY 注入)")
	}
	if sm4Key == "dGVzdC1zZWNyZXQxNiEhIQ==" {
		// ⚠️ 不强制中止: 数据库里可能已用默认值加密大量数据,中止会让所有历史数据无法读取
		// 仅打显眼 ERROR 日志,提醒部署者立即迁移
		applogger.Errorf("=================================================================")
		applogger.Errorf("= [SECURITY WARNING] SM4_KEY 是仓库默认值 test-secret16!!!")
		applogger.Errorf("= 任何能读仓库的人都能解密您所有 AD 凭据/设备密码/RPA 凭证")
		applogger.Errorf("= 强烈建议: 生成新 key (openssl rand -base64 16)")
		applogger.Errorf("=           → 用 migrate-sm4-key 命令迁移历史数据")
		applogger.Errorf("=           → 切换到新 key 后重新部署")
		applogger.Errorf("=================================================================")
	}
	cipher, err := crypto.NewSM4Cipher(sm4Key)
	if err != nil {
		return nil, fmt.Errorf("创建 SM4 cipher 失败: %w", err)
	}
	applogger.Infof("SM4 加密器初始化成功")
	return cipher, nil
}

// Init 初始化核心模块
// 按顺序初始化数据库、基础数据、权限服务和缓存系统
// 如果某个模块初始化失败，会记录警告但不会中断服务启动
//
// 本函数仅为编排层:每个步骤的具体实现下沉到 initXxx 私有方法,保留原本的
// 执行顺序与 fail-fast / warn-continue 策略。详见各 initXxx 方法的 doc 注释。
func (c *Core) Init() error {
	// 0. 设置 AD 域全局 SM4 cipher（FIX-01: 支持 SM4 密码加密）
	if c.SM4Cipher != nil {
		addomain.SetADSM4Cipher(c.SM4Cipher)
		applogger.Infof("AD 域 SM4 加密器已设置")
	} else {
		applogger.Warnf("AD 域 SM4 加密器未设置，密码将使用 AES-legacy 回退")
	}

	// 1-4. 数据库连接 / 迁移 / 基础数据 / 默认角色与菜单
	if err := c.initDBAndData(); err != nil {
		return err
	}

	// 5-6. 缓存系统 + 缓存服务 + 异步预热
	if err := c.initCacheAndWarmUp(); err != nil {
		return err
	}

	// 7. 系统指标缓存服务
	c.initMetrics()

	// 8-9.5. 设备连接池 / 执行器 / 发现 / 信息采集 / MAC 分区
	c.initDeviceServices()

	// 10-12. 调度器 + 所有定时任务注册 + 设备监控服务
	if err := c.initSchedulerAndTasks(); err != nil {
		return err
	}

	// 13-14.1. 验证码服务 + 背景图服务
	c.initCaptchaServices()

	// 15-16.5. 操作日志 + 令牌黑名单 + 认证策略工厂
	c.initLogsAndAuth()

	// 17-19. RPA + API 端点元数据 + 子进程 reaper
	c.initRPAAndAPIAndReaper()

	// 20. Phase 42 R1: 启动时刷新 reconciliation_normalized 物化视图 (D-02 冷启兜底)
	// 避免 0-5min 数据为空(用户首次访问 dashboard 不会看到 0 资产)。goroutine + 30s 超时,不阻断启动;
	// 失败仅 log 警告 — 与 cron 调度一致(D-02 设计)。
	go func() {
		refreshCtx, refreshCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer refreshCancel()
		if err := assetSvc.StartupRefreshView(refreshCtx, c.GetDB()); err != nil {
			applogger.Errorf("Phase 42 R1 startup RefreshView failed (D-02 仅 log): %v", err)
			return
		}
		applogger.Infof("Phase 42 R1 startup RefreshView succeeded (D-02 冷启兜底)")
	}()

	return nil
}

// initDBAndData 初始化数据库连接、表结构迁移、基础数据、默认角色与菜单权限。
//
// 对应原 Init() 步骤 1-4。fail-fast / warn-continue 策略与原实现完全一致:
//   - 步骤 1 (DB 连接): 失败必须终止启动 (F-15) — 否则后续所有 DB 操作 panic
//   - 步骤 2 (AutoMigrate): 失败必须终止启动 (P0 #16) — 表结构是所有 GORM 操作的基础
//   - 步骤 3 (InitData): 失败仅警告(不影响启动)
//   - 步骤 4 (默认角色/菜单): 失败仅警告
func (c *Core) initDBAndData() error {
	// 1. 初始化数据库连接
	var err error
	c.DB, err = db.NewDatabase(&c.Config.Database)
	if err != nil {
		// F-15: 数据库初始化失败必须终止启动 —
		// 原代码 return nil 会让应用以为初始化成功继续启动,
		// 后续所有 DB 操作都会 panic,且错误已被吞掉无法定位。
		applogger.Errorf("数据库初始化失败: %v", err)
		return fmt.Errorf("数据库初始化失败: %w", err)
	}

	// 2. 自动迁移数据库表结构
	// P0 #16: 表结构是所有 GORM 操作的基础，迁移失败会导致后续业务在缺失的
	// 表/列上 panic。fail-fast 比静默继续更安全（审查报告 #16）。
	//
	// SKIP_AUTOMIGRATE=true 旁路开关:Supabase pooler 上 GORM AutoMigrate(80+ DDL)
	// 会卡死在 dropDependent 之后;此时经 gorm.Migrator().CreateTable 从 model 派生
	// 补建缺失表(sys_api_keys + sys_api_key_usage_logs)。
	// 生产模式(mode=release)下此开关 fatal(CDX-H2):BootstrapMissingTables 仅保证
	// api key 两表,不跑 175/176/202-205 迁移,新库会得到半初始化系统 —— 属 dev 应急,
	// 生产误设必须 fail-fast。
	//
	// 2026-08-17 (quick-260817-hfl): 旁路仅对 postgres 生效。sqlite 无 Supabase pooler
	// 卡死问题,且本地新文件库必须全量 AutoMigrate 建表 —— .env 中残留的
	// SKIP_AUTOMIGRATE=true 切到 sqlite 后不应让库保持空壳。
	if os.Getenv("SKIP_AUTOMIGRATE") == "true" && c.DB.Type == "postgres" {
		// CDX-H2 生产守卫:release 模式下旁路补建仅覆盖部分表,直接终止启动
		if c.Config.Server.Mode == "release" {
			return fmt.Errorf("SKIP_AUTOMIGRATE=true 禁止在生产模式(server.mode=release)使用:" +
				"旁路补建仅覆盖部分表,会产生半初始化系统;请移除该环境变量后重启")
		}
		applogger.Warnf("[SKIP_AUTOMIGRATE=true] 跳过 AutoMigrate,改用 model 派生 DDL 补建(dev 旁路)")
		if err := c.DB.BootstrapMissingTables(); err != nil {
			return fmt.Errorf("BootstrapMissingTables 失败: %w", err)
		}
	} else if err := c.DB.AutoMigrate(); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}
	applogger.Infof("数据库表结构迁移完成")

	// 3. 初始化系统基础数据
	// 包括创建默认管理员账户、基础角色等
	//
	// SkipSetup 开关: dev 默认开启(每次跑 count 检查是浪费但安全);
	// 生产/CI 部署可设 SERVER_SKIP_SETUP=true 跳过,只跑首次 setup 阶段。
	// 注意: AutoMigrate 必须保留(表结构可能变化),只跳 InitData seed。
	if c.Config.Server.SkipSetup {
		applogger.Infof("[SkipSetup=true] 跳过系统基础数据初始化(InitData)")
	} else if err := c.DB.InitData(); err != nil {
		applogger.Warnf("初始化基础数据失败: %v", err)
		// 数据初始化失败不影响服务启动
	} else {
		applogger.Infof("基础数据初始化成功")
	}

	// 4. 初始化权限服务
	// 创建默认角色和菜单权限
	if c.DB != nil {
		if c.Config.Server.SkipSetup {
			applogger.Infof("[SkipSetup=true] 跳过默认角色和菜单初始化")
		} else {
			permissionSvc := permission.NewService()
			if err := permissionSvc.InitDefaultRolesAndMenus(c.GetDB()); err != nil {
				applogger.Warnf("初始化默认角色和菜单失败: %v", err)
			} else {
				applogger.Infof("默认角色和菜单初始化成功")
			}
		}
	}

	return nil
}

// initCacheAndWarmUp 初始化缓存系统、缓存服务,并按配置启动异步缓存预热。
//
// 对应原 Init() 步骤 5-6。fail-fast / warn-continue 策略与原实现完全一致:
//   - 步骤 5 (Cache 系统): 失败必须终止启动 (P0 #16) — 大量服务通过 CacheProvider 依赖缓存,
//     无缓存状态下静默退化会导致性能崩溃
//   - 步骤 6 (CacheConfigService / DataCacheService / CacheManager / 预热 goroutine): 无 fail-fast
func (c *Core) initCacheAndWarmUp() error {
	// 5. 初始化缓存系统
	// P0 #16: 大量系统服务通过 CacheProvider 接口依赖缓存。审查报告明确要求
	// Cache 失败应终止启动，避免服务在无缓存状态下静默退化/性能崩溃。
	var err error
	c.Cache, err = c.initCache()
	if err != nil {
		return fmt.Errorf("初始化缓存系统失败: %w", err)
	}
	applogger.Infof("缓存系统初始化完成")

	// 6. 初始化缓存服务（用于 System 模块）
	// System 模块的服务通过 CacheProvider 接口使用缓存
	if c.Cache != nil {
		// 初始化缓存配置服务
		c.CacheConfigService = services.NewCacheConfigService(c.GetDB())

		// 初始化数据缓存服务（供 System 模块适配使用）
		c.DataCacheService = services.NewDataCacheService(c.Cache)
		c.DataCacheService.SetCacheConfig(c.CacheConfigService)

		// Migrate209 配套菜单缓存失效 (Phase 70 D-11): 迁移改写了 sys_menu.component
		// (系统设置页前端目录合并), 必须清掉 6 个 menu: key 前缀, 否则 Redis 30 分钟
		// TTL 内菜单接口持续返回旧 component 路径 → 前端懒加载拾取不到组件白屏。
		// 时序约束: 迁移执行于 db.NewDatabase (initDBAndData), 早于本处 cache 创建,
		// 迁移函数内拿不到 cache 实例 —— 故由 Database.SettingsMenuComponentChanged
		// 标志传递, 在 DataCacheService 就绪后、缓存预热前按标志失效。
		if c.DB != nil && c.DB.SettingsMenuComponentChanged {
			system.InvalidateCacheByPattern(context.Background(), system.NewCacheProvider(c.DataCacheService), []string{
				system.CacheKeyMenuTree + "*",
				system.CacheKeyMenuRouter + "*",
				system.CacheKeyMenuAll + "*",
				system.CacheKeyMenuUserMenus + ":*",
				system.CacheKeyMenuUserAllMenus + ":*",
				system.CacheKeyMenuUserPermissions + ":*",
			}, "MENU")
			applogger.Infof("[迁移] 209 菜单缓存已失效 (系统设置 component 路径变更)")
		}

		// 初始化缓存管理器（增强功能）
		// 缓存预热功能默认禁用，可通过配置启用
		cacheManagerEnabled := c.Config.Cache.WarmUpEnabled
		c.CacheManager = system.NewCacheManager(
			system.NewCacheAdapter(c.Cache),
			"xingran", // 缓存键前缀
			cacheManagerEnabled,
		)

		applogger.Infof("缓存服务初始化完成（预热功能: %v）", cacheManagerEnabled)

		// 初始化缓存预热服务
		warmUpSvcs := c.initSystemServicesForWarmUp()

		// 执行缓存预热（异步执行，不阻塞启动）
		if cacheManagerEnabled {
			go c.performCacheWarmUp(context.Background(), warmUpSvcs)
		}
	}

	return nil
}

// initMetrics 初始化系统指标缓存服务。
//
// 对应原 Init() 步骤 7。无 fail-fast。
func (c *Core) initMetrics() {
	// 7. 初始化系统指标缓存服务
	c.MetricsCacheService = NewMetricsCacheService(c)
	applogger.Infof("系统指标缓存服务初始化完成")
}

// initDeviceServices 初始化网络设备相关服务:连接池、任务调度器、执行器、设备发现、
// 设备信息采集、MAC 历史分区管理。
//
// 对应原 Init() 步骤 8-9.5。无 fail-fast:
//   - 步骤 9.1 (DeviceInfoCollectionService.Start): 失败仅警告
//   - 步骤 9.5 (PartitionService.EnsurePartitionsExist): 失败仅警告
//     (不阻断应用启动,分区创建可稍后手动执行)
func (c *Core) initDeviceServices() {
	// 8. 初始化网络设备管理器（新架构）
	// 8.1 创建连接池 (容量/空闲时长从 sys_config 读取, web 可配, 默认 50/300s)
	poolConfig := loadConnectionPoolConfig(c.GetDB())
	connectionPool := device.NewDeviceConnectionPool(c.GetDB(), c.SM4Cipher, &poolConfig)
	// 8.2 创建任务调度器
	schedulerConfig := &device.SchedulerConfig{
		TaskTimeout: 5 * time.Minute,
		QueueSize:   100,
	}
	taskScheduler := device.NewDeviceTaskScheduler(connectionPool, schedulerConfig)
	// 8.3 创建执行器
	executorConfig := &device.ExecutionConfig{
		MaxRetries:          2,
		RetryDelay:          time.Second,
		Timeout:             30 * time.Second,
		EnablePanicRecovery: true,
	}
	c.DeviceExecutor = device.NewDeviceExecutor(taskScheduler, executorConfig)
	applogger.Infof("网络设备管理器（新架构）初始化完成")

	// 9. 初始化设备发现服务
	c.DeviceDiscoveryService = services.NewDeviceDiscoveryService(c.GetDB())
	applogger.Infof("设备发现服务初始化完成")

	// 9.1. 初始化设备信息采集服务（后台异步，统一处理设备创建和定时任务）
	c.DeviceInfoCollectionService = services.NewDeviceInfoCollectionService(c.GetDB(), c.DeviceExecutor)
	if err := c.DeviceInfoCollectionService.Start(context.Background()); err != nil {
		applogger.Warnf("启动设备信息采集服务失败: %v", err)
	} else {
		applogger.Infof("设备信息采集服务启动成功")
	}

	// 9.5. 初始化MAC历史分区管理服务（必须在调度器初始化之前）
	c.PartitionService = services.NewPartitionService(c.GetDB())
	// 确保未来2个月的分区存在
	if err := c.PartitionService.EnsurePartitionsExist(context.Background(), 2); err != nil {
		applogger.Warnf("初始化MAC历史分区失败: %v", err)
		// 不阻断应用启动（分区创建可稍后手动执行）
	} else {
		applogger.Infof("MAC历史分区管理服务初始化完成")
	}
}

// initSchedulerAndTasks 初始化定时任务调度器、注册所有 cron 任务处理器,
// 并初始化设备监控服务。
//
// 对应原 Init() 步骤 10-12。fail-fast / warn-continue 策略与原实现完全一致:
//   - 步骤 10 (Scheduler.Start): 失败必须终止启动 (F-16 部分) — 所有 cron 任务依赖它,
//     调度器静默失败会导致后台业务全部停摆而管理员无任何感知
//   - 步骤 11 (Register*Tasks / VDI / ADSync / SyncPeriodicWorkOrderJobs 等): 失败仅警告
//   - 步骤 12 (DeviceMonitorService 装配): 无 fail-fast
func (c *Core) initSchedulerAndTasks() error {
	// 10. 初始化定时任务调度器
	// F-16 (部分): Scheduler.Start 失败必须终止启动 —
	// 所有 cron 任务(AD 同步/工单/MAC清理/VDI 等)都依赖它,
	// 调度器静默失败会导致这些后台业务全部停摆而管理员无任何感知。
	c.Scheduler = scheduler.NewScheduler(c.GetDB())
	if err := c.Scheduler.Start(context.Background()); err != nil {
		return fmt.Errorf("启动调度器失败: %w", err)
	}
	applogger.Infof("定时任务调度器启动成功")

	{

		// 设置全局调度器引用（供定时任务使用）
		scheduler.SetGlobalScheduler(c.Scheduler)

		// 注册网络设备相关定时任务
		scheduler.RegisterNetworkDeviceTasks(c.Scheduler, c.GetDB())
		applogger.Infof("网络设备定时任务注册完成")

		// 注册通知相关定时任务
		scheduler.RegisterNoticeTasks(c.Scheduler)
		applogger.Infof("通知定时任务注册完成")

		// 注册运维工单相关定时任务
		scheduler.RegisterWorkOrderTasks(c.Scheduler)
		applogger.Infof("运维工单定时任务注册完成")

		// 注册AD域同步定时任务
		scheduler.RegisterADSyncTasks(c.Scheduler)
		applogger.Infof("AD域同步定时任务注册完成")

		// Phase 42 R1: 注册对账相关 4 个 cron 任务处理器(refreshView / detectLayer3 /
		// detectExpiredSilence / cleanupExpiredExceptions)。原 42-02 plan 创建了 sys_job
		// 记录和 Execute*Task 全局函数,但漏了 RegisterTask 把 InvokeTarget 映射到
		// 实际 handler,导致调度器报 "未找到任务处理器: reconciliation"。
		// Phase 43 R2: 注入 NoticeHub + NoticeService,critical 转单成功后触发 WS 推送 +
		// SysNotice 写入(D-A4-01/03 双通道)。nil 时跳过,保持向后兼容。
		//
		// Phase 45 R4: 注入 c.Cache,R2 createWorkorderBySeverity 完成后通过
		// woSvc.WorkstationIDForException + woSvc.InvalidateWorkstationHealth 主动失效
		// 工位健康度缓存(D-A4-04)。nil 时跳过(单测 / 非 production)。
		scheduler.RegisterReconciliationTasks(c.Scheduler, c.GetDB(), c.Cache, c.NoticeHub, services.NewNoticeService(c.GetDB()))
		applogger.Infof("对账定时任务注册完成")

		// 注册MAC历史清理任务
		if c.PartitionService != nil {
			scheduler.SetPartitionService(c.PartitionService)

			// 2026-06-30 新增:注册 MAC历史 purge 服务(cron 月度无意义记录清理)。
			// 单独构造,不挂在 Core 字段上(服务无状态、每次调用新建 *gorm.DB 即用即弃)。
			macPurgeSvc := services.NewMACHistoryService(c.GetDB())
			scheduler.SetMACHistoryPurgeService(macPurgeSvc)

			scheduler.RegisterMACHistoryTasks(c.Scheduler, c.GetDB())
			applogger.Infof("MAC历史清理任务注册完成")
		}

		// 注意:Phase 45 R5 数据治理 (2026-06-30) 的"对账-port_status漂移检测"已整合到
		// 上方 scheduler.RegisterReconciliationTasks(...) 调用中,作为 reconciliation 框架
		// 的子任务 reconciliation:checkPortStatusDrift,与现有对账任务共用 sys_job 表
		// + cron 调度,避免与对账任务重复。

		// Phase 15: 性能优化 — 物化视图刷新任务 + 性能配置 seed
		if err := services.SeedMACPerfConfigs(c.GetDB()); err != nil {
			applogger.Errorf("MAC 性能配置 seed 失败: %v", err)
		}
		matViewSvc := services.NewMACHistoryMatViewService(c.GetDB())
		scheduler.RegisterMACHistoryMatViewTasks(c.Scheduler, c.GetDB(), matViewSvc)
		applogger.Infof("MAC历史物化视图刷新任务注册完成")

		// 启动AD域同步调度器（每5分钟检查需要同步的配置）
		if c.SM4Cipher != nil {
			scheduler.SetADSM4Cipher(c.SM4Cipher)
		}
		scheduler.StartADSyncScheduler(c.GetDB())

		// 初始化VDI虚拟机服务并注入到调度器
		vdiVMService := vdi.NewVMServiceWithDynamicClient(c.GetDB())
		scheduler.SetVDIVMService(vdiVMService)
		applogger.Infof("VDI虚拟机服务初始化完成")

		scheduler.RegisterVDISyncTasks(c.Scheduler)
		applogger.Infof("VDI同步定时任务注册完成")

		// 注册部门到AD同步定时任务
		scheduler.RegisterDeptSyncTasks(c.Scheduler)
		applogger.Infof("部门到AD同步定时任务注册完成")

		// 设置全局数据库访问器，供定时任务使用（必须在同步周期性工单任务之前）
		scheduler.SetDB(c)

		// 同步周期性工单模板到调度器
		if err := scheduler.SyncPeriodicWorkOrderJobs(c.Scheduler); err != nil {
			applogger.Warnf("同步周期性工单任务失败: %v", err)
		} else {
			applogger.Infof("周期性工单任务同步完成")
		}
	}

	// 12. 初始化设备监控服务
	c.DeviceMonitorService = services.NewDeviceMonitorService(c.GetDB(), c.SM4Cipher, services.DefaultDeviceMonitorConfig())
	// 设置执行器并初始化子服务
	c.DeviceMonitorService.SetExecutor(c.DeviceExecutor)
	// 设置设备监控服务到调度器，供定时任务调用
	scheduler.SetDeviceMonitorService(c.DeviceMonitorService)
	// 设置设备信息采集服务到调度器，供定时任务调用
	scheduler.SetDeviceInfoCollectionService(c.DeviceInfoCollectionService)
	applogger.Infof("设备监控服务初始化完成")

	return nil
}

// initCaptchaServices 初始化验证码服务、验证码背景图服务,并确保存储目录存在。
//
// 对应原 Init() 步骤 13-14.1。无 fail-fast:
//   - 步骤 13 (CaptchaService.LoadConfig): 失败仅警告
//   - 步骤 14.1 (MkdirAll 存储目录): 失败仅警告
func (c *Core) initCaptchaServices() {
	// 13. 初始化验证码服务
	c.CaptchaService = NewCaptchaService(c.DB, c.Cache)
	if err := c.CaptchaService.LoadConfig(context.Background()); err != nil {
		applogger.Warnf("加载验证码配置失败: %v", err)
	} else {
		applogger.Infof("验证码服务初始化完成，类型: %s", c.CaptchaService.GetConfig().Enabled)
	}

	// 14. 初始化验证码背景图服务
	c.CaptchaBackgroundService = NewCaptchaBackgroundService(c.DB, c.Cache)
	// 将背景图服务设置到验证码服务
	c.CaptchaService.SetBackgroundService(c.CaptchaBackgroundService)

	// 14.1 确保存储目录存在
	storagePath := "./uploads/captcha/backgrounds"
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		applogger.Warnf("创建验证码背景图存储目录失败: %v", err)
	} else {
		applogger.Infof("验证码背景图存储目录已创建: %s", storagePath)
	}

	applogger.Infof("验证码背景图服务初始化完成")
}

// initLogsAndAuth 初始化操作日志服务、令牌黑名单服务、认证策略工厂。
//
// 对应原 Init() 步骤 15-16.5。无 fail-fast。
func (c *Core) initLogsAndAuth() {
	// 15. 初始化操作日志服务
	c.OperLogService = services.NewOperLogService()
	applogger.Infof("操作日志服务初始化完成")

	// 16. 初始化令牌黑名单服务
	c.TokenBlacklistService = services.NewTokenBlacklistService(c.Cache)
	applogger.Infof("令牌黑名单服务初始化完成")
	// 16.5 初始化认证策略工厂
	c.initAuthFactory()
	applogger.Infof("认证策略工厂初始化完成")
}

// initRPAAndAPIAndReaper 初始化 RPA 服务、API 端点元数据服务,并启动子进程 reaper。
//
// 对应原 Init() 步骤 17-19。无 fail-fast:
//   - 步骤 17 (RPA): 失败仅警告
//   - 步骤 18 (APIEndpointService): 失败仅警告
//   - 步骤 19 (子进程 reaper): 无错误路径
func (c *Core) initRPAAndAPIAndReaper() {
	// 17. 初始化RPA服务
	if c.Config.RPA.Enabled {
		if err := c.initRPAServices(&c.Config.RPA); err != nil {
			applogger.Warnf("初始化RPA服务失败: %v", err)
		} else {
			applogger.Infof("RPA服务初始化完成")
		}
	}

	// 18. 初始化API端点元数据服务
	if err := c.initAPIEndpointService(); err != nil {
		applogger.Warnf("初始化API端点元数据服务失败: %v", err)
	} else {
		applogger.Infof("API端点元数据服务初始化完成")
	}

	// 19. 启动子进程 reaper（P2-A7: 清理僵尸子进程，防止 FD 泄漏）
	c.reaperCtx, c.reaperCancel = context.WithCancel(context.Background())
	c.startSubprocessReaper(c.reaperCtx)
	applogger.Infof("子进程 reaper 已启动")
}

// coreShutdownTimeout 整个 Core.Close() 的硬截止时间。
//
// 设计目的:waitForShutdown 在 srv.Shutdown(ctx, 10s) 完成后调 coreModule.Close(),
// 但 Close 内部任何一步若阻塞(例如 DeviceInfoCollectionService.Stop() 等 SSH
// 长连接 worker,或 Scheduler.Stop() 等 in-flight cron) 都无 ctx 包裹 → 进程
// 永远不退。30s 兜底保证 close 流程在 30s 内必然返回,WaitGroup 卡死时强制
// 交还控制权(资源由 OS 回收,不会泄漏到下次启动)。
const coreShutdownTimeout = 30 * time.Second

// Close 关闭核心模块
// 优雅关闭所有核心服务，释放资源
//
// 关闭顺序(2026-07-06 修复 — shutdown-hang-after-port-close):
//  1. 子进程 reaper (P2-A7)
//  2. 通知中心 (NoticeHub)
//  3. 定时任务调度器 (Scheduler) — 先停 cron 引擎,切断新任务源
//  4. AD 域同步调度器 (StopADSyncScheduler) — 同上,robfig cron 无 wait
//  5. 设备信息采集服务 (DeviceInfoCollectionService.Stop)
//  6. 设备监控服务 (DeviceMonitorService.Close)
//  7. 系统指标缓存服务 (MetricsCacheService.Stop)
//  8. RPA 扩缩容服务 (RPAScalingService.Stop)
//  9. 缓存 (Cache.Close) — 在 DB 之前关,避免异步写丢
//  10. sleep 100ms — 等待操作日志异步写完成(详见函数内注释)
//  11. 数据库连接 (DB.Close)
//
// 总截止:coreShutdownTimeout(30s) — 任何子步骤阻塞 → 强制返回,绝不卡死
//
// C1 可取消的优雅关闭(谨慎两步法,本次落地第一步):
//   - 第一步(本次):建立 shutdownCtx (context.WithTimeout, 等于 coreShutdownTimeout) 作为
//     总 deadline,并把监视 goroutine 改为 select shutdownCtx.Done() — 单一 deadline 来源,
//     同时为 ctx-aware 子服务预留接入点。当前各子服务 Stop()/Close() 签名均不接受 ctx
//     (Scheduler.Stop / DeviceInfoCollectionService.Stop / DeviceMonitorService.Close /
//     MetricsCacheService.Stop / RPAScalingService.Stop / NoticeHub.Stop / Cache.Close /
//     Database.Close 均为无参或仅返回 error),不强行改签名以避免回归。
//   - 第二步(未来):当子服务升级为接受 context.Context 时,直接将 shutdownCtx 传入
//     对应 Stop(ctx) / Close(ctx) 调用,即可实现"deadline 触发即取消子调用"的真正语义。
func (c *Core) Close() {
	// shutdownCtx 是整个 Close() 流程的总 deadline。各子服务目前不接受 ctx,
	// 此 ctx 暂仅驱动下方 deadline 监视 goroutine;待子服务签名升级后即可注入。
	shutdownCtx, cancel := context.WithTimeout(context.Background(), coreShutdownTimeout)
	defer cancel()

	// deadline 监视 goroutine:到点仅 log 警告(Close 无法主动中断已无 ctx 包裹的子调用,
	// 资源由 OS 回收,不会泄漏到下次启动)。
	// closeDone 让 goroutine 在 Close() 正常返回时静默退出,避免 cancel() 触发的
	// shutdownCtx.Done() 产生误报 warn(Err() == DeadlineExceeded 才是真超时)。
	closeDone := make(chan struct{})
	defer close(closeDone)
	go func() {
		select {
		case <-shutdownCtx.Done():
			if shutdownCtx.Err() == context.DeadlineExceeded {
				applogger.Warnf("[Core.Close] 已超过 %v 强制结束(子步骤阻塞,资源将由 OS 回收)", coreShutdownTimeout)
			}
		case <-closeDone:
		}
	}()

	// 停止子进程 reaper（P2-A7）— reaperCancel 是 context.CancelFunc,立即返回,无需 ctx
	if c.reaperCancel != nil {
		c.reaperCancel()
		applogger.Infof("子进程 reaper 已停止")
	}
	// 停止通知中心 — NoticeHub.Stop() 当前不接受 ctx,内部为同步清理
	if c.NoticeHub != nil {
		c.NoticeHub.Stop()
		applogger.Infof("通知中心已停止")
	}
	// 关键修复:先停 cron 引擎(Scheduler + ADSync),切断新任务源。
	// 原顺序在 DeviceInfoCollectionService.Stop() 之后,前 5-15s 仍在 spawn
	// 对账/物化视图/采集任务,这些任务抢 DB 连接 + 触发 SSH 长连接 →
	// pool 24/20 + WaitGroup 死等 → 进程永远不退。
	// Scheduler.Stop() 当前不接受 ctx(robfig cron 无 ctx 集成),由总 deadline 兜底
	if c.Scheduler != nil {
		c.Scheduler.Stop()
		applogger.Infof("定时任务调度器已停止")
	}
	scheduler.StopADSyncScheduler()
	// 停止设备信息采集服务(内部自带 8s 兜底,不会阻塞 Close)。
	// DeviceInfoCollectionService.Stop() 当前不接受 ctx,内部 goroutine 通过 channel close 退出
	if c.DeviceInfoCollectionService != nil {
		c.DeviceInfoCollectionService.Stop()
		applogger.Infof("设备信息采集服务已停止")
	}
	// 关闭设备监控服务 — DeviceMonitorService.Close() 当前不接受 ctx,返回 error 沿用原行为忽略
	if c.DeviceMonitorService != nil {
		c.DeviceMonitorService.Close()
		applogger.Infof("设备监控服务已关闭")
	}
	// 停止系统指标缓存服务 — MetricsCacheService.Stop() 当前不接受 ctx
	if c.MetricsCacheService != nil {
		c.MetricsCacheService.Stop()
		applogger.Infof("系统指标缓存服务已停止")
	}
	// 停止 RPA 扩缩容服务 — RPAScalingService.Stop() 受 rpaScalingService 接口约束,当前不接受 ctx
	if c.RPAScalingService != nil {
		c.RPAScalingService.Stop()
		applogger.Infof("RPA 扩缩容服务已停止")
	}
	// 关闭缓存（必须在数据库之前关闭，因为缓存可能有异步写入操作）。
	// MultiLevelCache.Close() 内部已通过 L2WriteWriter.Stop() 的 wg.Wait() 确定性等待
	// 所有 L2 异步写完成(pkg/cache/l2_writer.go:181) — 这里无需额外同步原语。
	// Cache.Close() 当前不接受 ctx。
	if c.Cache != nil {
		c.Cache.Close()
		applogger.Infof("缓存已关闭")
	}
	// 等待操作日志异步写入完成(heuristic 兜底)。
	//
	// operlog.RecordAsync / operLogService.RecordAsync 均为 fire-and-forget goroutine
	// (internal/services/oper_log_service.go:67 `go func() { db.Create(operLog) }`),
	// **无暴露的 WaitGroup/Flush 原语**,因此无法用确定性 sync.WaitGroup 替代 sleep
	// (C1 spec: "if a given call has no clean WaitGroup target, keep current behavior
	// and add a brief comment noting why")。
	// 100ms 是经验值,覆盖 DB.Create 的往返延迟。如未来 operlog 服务暴露 Flush()/Wait(),
	// 应改为确定性等待(DB.Close 之前需确保异步审计写已落盘,避免丢失审计记录)。
	time.Sleep(100 * time.Millisecond)
	// 最后关闭数据库连接 — Database.Close() 不接受 ctx,内部 sqlDB.Close() 由 database/sql 自管超时
	if c.DB != nil {
		c.DB.Close()
		applogger.Infof("数据库连接已关闭")
	}
}

// initCache 初始化缓存系统
// 根据配置创建相应的缓存实例：
// - Redis缓存：创建L1(内存)+L2(Redis)多级缓存
// - 内存缓存：创建纯内存缓存
func (c *Core) initCache() (cache.Cache, error) {
	if c.Config.Cache.Type == "redis" {
		// 创建Redis缓存配置
		cacheConfig := &cache.CacheConfig{
			Type:     c.Config.Cache.Type,
			Host:     c.Config.Cache.Host,
			Port:     c.Config.Cache.Port,
			Password: c.Config.Cache.Password,
			DB:       c.Config.Cache.DB,
			PoolSize: c.Config.Cache.PoolSize,
			TLS:      c.Config.Cache.TLS,
		}

		// 创建Redis缓存实例作为L2缓存
		redisCache, err := cache.NewRedisCache(cacheConfig, "xingran")
		if err != nil {
			return nil, err
		}

		// 创建内存缓存作为L1缓存，提供快速访问
		// 容量与清理周期来自 package-level fallback 常量，便于集中调优
		memoryCache := cache.NewMemoryCache(fallbackMemoryCacheSize, fallbackMemoryCacheCleanup)

		// 创建多级缓存：L1(内存) -> L2(Redis)
		// 先查内存缓存，命中则返回；未命中则查Redis缓存
		if c.Config.Cache.RetryEnabled {
			// 创建带重试的多级缓存
			retryConfig := &cache.RetryConfig{
				MaxRetries:     c.Config.Cache.RetryMaxRetries,
				InitialDelay:   time.Duration(c.Config.Cache.RetryInitialDelay) * time.Millisecond,
				MaxDelay:       time.Duration(c.Config.Cache.RetryMaxDelay) * time.Millisecond,
				BackoffFactor:  c.Config.Cache.RetryBackoffFactor,
				RetryableCheck: cache.IsRetryableError,
			}
			workerCount := c.Config.Cache.RetryWorkerCount
			if workerCount <= 0 {
				workerCount = 3 // 默认3个工作协程
			}

			// 解析L2Writer配置
			l2WriterConfig := cache.DefaultL2WriterConfig()
			if c.Config.Cache.L2Writer != nil {
				l2WriterConfig = &cache.L2WriterConfig{
					WorkerCount:          c.Config.Cache.L2Writer.WorkerCount,
					QueueSize:            c.Config.Cache.L2Writer.QueueSize,
					EnqueueTimeout:       c.Config.Cache.L2Writer.EnqueueTimeout,
					WriteTimeout:         c.Config.Cache.L2Writer.WriteTimeout,
					FallbackWriteTimeout: c.Config.Cache.L2Writer.FallbackWriteTimeout,
				}
			}

			multiLevelCache := cache.NewMultiLevelCacheWithRetryAndWriter(memoryCache, redisCache, retryConfig, workerCount, l2WriterConfig)
			applogger.Infof("缓存重试功能已启用: 最大重试次数=%d, 工作协程数=%d", retryConfig.MaxRetries, workerCount)
			applogger.Infof("L2写入Worker已启用: worker数量=%d, 队列大小=%d, 降级超时=%v",
				l2WriterConfig.WorkerCount, l2WriterConfig.QueueSize, l2WriterConfig.FallbackWriteTimeout)
			return multiLevelCache, nil
		}

		// 不启用重试时，也启用L2Writer
		l2WriterConfig := cache.DefaultL2WriterConfig()
		if c.Config.Cache.L2Writer != nil {
			l2WriterConfig = &cache.L2WriterConfig{
				WorkerCount:          c.Config.Cache.L2Writer.WorkerCount,
				QueueSize:            c.Config.Cache.L2Writer.QueueSize,
				EnqueueTimeout:       c.Config.Cache.L2Writer.EnqueueTimeout,
				WriteTimeout:         c.Config.Cache.L2Writer.WriteTimeout,
				FallbackWriteTimeout: c.Config.Cache.L2Writer.FallbackWriteTimeout,
			}
		}
		multiLevelCache := cache.NewMultiLevelCacheWithWriter(memoryCache, redisCache, l2WriterConfig)
		applogger.Infof("L2写入Worker已启用: worker数量=%d, 队列大小=%d, 降级超时=%v",
			l2WriterConfig.WorkerCount, l2WriterConfig.QueueSize, l2WriterConfig.FallbackWriteTimeout)
		return multiLevelCache, nil
	} else {
		// 创建纯内存缓存
		// 容量：配置中指定的大小，过期时间：配置中指定的清理时间
		return cache.NewMemoryCache(c.Config.Cache.MaxSize, time.Duration(c.Config.Cache.CleanupTime)*time.Second), nil
	}
}

// initAPIEndpointService 初始化API端点元数据服务
//
// 启动期配置加载,使用 background ctx(API元数据 < 100KB,阻塞时间通常 < 10ms,
// 不需要可取消语义)。与 services/vdi/config.go / services/ad_ldap_client.go
// 的 config.Load 模式保持一致。
func (c *Core) initAPIEndpointService() error {
	// 加载API元数据配置
	metadata, err := config.LoadAPIMetadata(context.Background(), "./configs/api_metadata.yaml")
	if err != nil {
		return err
	}

	// 创建API端点服务
	c.APIEndpointService = services.NewAPIEndpointService(
		metadata,
		c.Cache,
		c.GetDB(),
	)

	return nil
}

// initAuthFactory 初始化认证策略工厂
func (c *Core) initAuthFactory() {
	if c.DB == nil {
		applogger.Warnf("数据库未初始化，跳过认证策略工厂初始化")
		return
	}
	c.AuthFactory = security.NewAuthStrategyFactory(c.GetDB(), c.PwdManager, c.SM4Cipher)

	// 初始化用户同步服务并注入到工厂
	mapper := addomain.NewDeptOUmapper(c.GetDB())
	userSyncService := system.NewUserSyncService(
		c.GetDB(),
		c.PwdManager,
		mapper,
		system.WithCacheProvider(system.NewCacheProvider(c.DataCacheService)), // L-02: 角色分配后失效 user-scoped 菜单缓存
	)
	c.AuthFactory.SetUserSyncer(userSyncService)

	// Phase 36: 注入 AD 账号池（多账号故障切换）
	// 单账号被 AD 锁定（data 775）不再阻断用户登录
	accountPool := addomain.NewAccountPool(c.GetDB(), nil) // 无 Redis pub/sub 跨进程广播（单机部署）
	c.AuthFactory.SetAccountPool(accountPool)

	// Phase 36: 启动 Redis pub/sub 跨进程缓存失效订阅
	// TODO: 当 core.Core 接入 Redis 后启用；当前传 nil 不影响主流程
	if err := accountPool.StartHotReload(context.Background()); err != nil {
		applogger.Warnf("启动 AD 账号池热加载失败（不影响主流程）: %v", err)
	}

	// Phase 38 (D-03): 启动空池校验（仅 WARN，不阻断启动）
	c.checkEmptyAccountPoolOnStartup(accountPool)

	applogger.Infof("认证策略工厂初始化完成（支持 local/ad/hybrid 模式 + AD 账号池）")
}

// checkEmptyAccountPoolOnStartup Phase 38 (D-03): 启动时校验启用的 AD 配置账号池非空。
// 仅记 WARN 日志，不返回 error，不阻断启动（Pitfall 6）——避免阻塞新环境首次部署。
func (c *Core) checkEmptyAccountPoolOnStartup(pool addomain.AccountPool) {
	type adCfgRow struct {
		ID         string
		ConfigName string
	}
	var configs []adCfgRow
	if err := c.GetDB().Table("sys_ad_config").
		Select("id, config_name").
		Where("status = 0 AND sync_enabled = true").
		Find(&configs).Error; err != nil {
		applogger.Warnf("[启动校验] 查询启用的 AD 配置失败: %v", err)
		return
	}
	ctx := context.Background()
	for _, cfg := range configs {
		total, available, _, _, err := pool.CountByStatus(ctx, cfg.ID)
		if err != nil {
			applogger.Warnf("[启动校验] AD 配置 %s 账号池查询失败: %v", cfg.ConfigName, err)
			continue
		}
		if total == 0 || available == 0 {
			applogger.Warnf("[启动校验] AD 配置 %s (ID=%s) 账号池为空（total=%d, available=%d），请在 AD 配置页详情 → 服务账号池 Tab 添加服务账号，否则登录/同步将失败",
				cfg.ConfigName, cfg.ID, total, available)
		}
	}
}

// GetAuthFactory 获取认证策略工厂
func (c *Core) GetAuthFactory() *security.AuthStrategyFactory {
	return c.AuthFactory
}

// initSystemServicesForWarmUp 初始化缓存预热所需的 System 服务
func (c *Core) initSystemServicesForWarmUp() *warmUpServices {
	db := c.GetDB()
	cacheProvider := system.NewCacheProvider(c.DataCacheService)

	return &warmUpServices{
		UserService: system.NewUserServiceWithCache(db, cacheProvider, c.CacheConfigService, c.PwdManager),
		RoleService: system.NewRoleServiceWithCache(db, cacheProvider, c.CacheConfigService),
		MenuService: system.NewMenuServiceWithCache(db, cacheProvider, c.CacheConfigService),
		DeptService: system.NewDepartmentServiceWithCache(db, cacheProvider, c.CacheConfigService),
		PostService: system.NewPostServiceWithCache(db, cacheProvider, c.CacheConfigService),
	}
}

// performCacheWarmUp 执行缓存预热
func (c *Core) performCacheWarmUp(ctx context.Context, svcs *warmUpServices) {
	// 等待 DB 就绪（替代硬编码 sleep）：短轮询 Ping，带总超时，慢机等够、快机不白等
	readyCtx, readyCancel := context.WithTimeout(ctx, 10*time.Second)
	defer readyCancel()
	for {
		if c.GetDB() != nil {
			if sqlDB, err := c.GetDB().DB(); err == nil {
				if err := sqlDB.PingContext(readyCtx); err == nil {
					break
				}
			}
		}
		select {
		case <-readyCtx.Done():
			applogger.Warnf("缓存预热：等待 DB 就绪超时，放弃本轮预热")
			return
		case <-time.After(200 * time.Millisecond):
		}
	}

	applogger.Infof("开始执行缓存预热...")

	warmUpFuncs := map[string]system.WarmUpFunc{
		"user": system.WarmUpUserCache(svcs.UserService),
		"role": system.WarmUpRoleCache(svcs.RoleService),
		"menu": system.WarmUpMenuCache(svcs.MenuService),
		"dept": system.WarmUpDeptCache(svcs.DeptService),
		"post": system.WarmUpPostCache(svcs.PostService),
	}

	if err := c.CacheManager.WarmUp(ctx, warmUpFuncs); err != nil {
		applogger.Warnf("缓存预热失败: %v", err)
	} else {
		applogger.Infof("缓存预热成功完成")
	}
}

// initRPAServices 初始化 RPA 服务
func (c *Core) initRPAServices(rpaConfig *config.RPAConfig) error {
	// 确保 RPA 存储目录存在
	if rpaConfig.Storage.DownloadsDir != "" {
		if err := os.MkdirAll(rpaConfig.Storage.DownloadsDir, 0755); err != nil {
			applogger.Warnf("创建RPA下载目录失败: %v", err)
		}
	}
	if rpaConfig.Storage.ScreenshotsDir != "" {
		if err := os.MkdirAll(rpaConfig.Storage.ScreenshotsDir, 0755); err != nil {
			applogger.Warnf("创建RPA截图目录失败: %v", err)
		}
	}

	// 检查 AI 配置
	if rpaConfig.AI.Agent.Enabled && rpaConfig.AI.Agent.APIKey == "" {
		applogger.Warnf("RPA AI Agent 已启用但未配置 API Key，请在环境变量中设置 RPA_AI_AGENT_KEY")
	}
	if rpaConfig.AI.Generator.Enabled && rpaConfig.AI.Generator.APIKey == "" {
		applogger.Warnf("RPA AI Generator 已启用但未配置 API Key，请在环境变量中设置 RPA_AI_GENERATOR_KEY")
	}

	// 初始化 RPA 扩缩容服务
	if rpaConfig.Scaling.Enabled {
		if err := c.initRPAScalingService(rpaConfig); err != nil {
			applogger.Warnf("启动RPA扩缩容服务失败: %v", err)
		}
	}

	// 注册 RPA 任务到调度器
	c.registerRPATasks()

	return nil
}

// registerRPATasks 注册 RPA 任务处理器到调度器
func (c *Core) registerRPATasks() {
	// 注册 RPA 任务执行处理器
	c.Scheduler.RegisterTask("rpa_task", func(ctx context.Context, params map[string]interface{}) error {
		// 从参数中获取任务ID
		taskIDStr, ok := params["task_id"].(string)
		if !ok || taskIDStr == "" {
			return fmt.Errorf("rpa_task 处理器缺少 task_id 参数")
		}

		// 获取 RPA 服务
		rpaServices := rpa.NewServiceGroup(c.GetDB(), c.Config, c.NoticeHub, c.Cache, c.SM4Cipher)

		// 执行 RPA 任务
		req := &rpa.ExecuteTaskRequest{
			TaskID: taskIDStr,
		}

		// 如果有额外参数，添加到输入参数中
		if len(params) > 1 {
			req.InputParams = make(map[string]interface{})
			for k, v := range params {
				if k != "task_id" {
					req.InputParams[k] = v
				}
			}
		}

		_, err := rpaServices.TaskService.Execute(ctx, req, "system")
		return err
	})

	applogger.Infof("RPA 任务处理器已注册到调度器")
}

// initRPAScalingService 初始化 RPA 扩缩容服务
func (c *Core) initRPAScalingService(rpaConfig *config.RPAConfig) error {
	// 动态导入以避免循环依赖
	scalingConfig := &rpa.ScalingConfig{
		Enabled:           rpaConfig.Scaling.Enabled,
		CheckInterval:     parseDuration(rpaConfig.Scaling.CheckInterval, 30*time.Second),
		MinWorkers:        rpaConfig.Scaling.MinWorkers,
		MaxWorkers:        rpaConfig.Scaling.MaxWorkers,
		ScaleUpThreshold:  rpaConfig.Scaling.ScaleUpThreshold,
		ScaleDownCooldown: parseDuration(rpaConfig.Scaling.ScaleDownCooldown, rpaDefaultScaleDownCooldown),
		ScaleUpLimit:      rpaConfig.Scaling.ScaleUpLimit,
		EnableMockDocker:  rpaConfig.Scaling.EnableMockDocker,
	}

	dockerConfig := &rpa.DockerConfig{
		DockerHost:    rpaConfig.Scaling.Docker.DockerHost,
		DockerPort:    rpaConfig.Scaling.Docker.DockerPort,
		ContainerName: rpaConfig.Scaling.Docker.ContainerName,
		ImageName:     rpaConfig.Scaling.Docker.ImageName,
		NetworkName:   rpaConfig.Scaling.Docker.NetworkName,
	}

	// 验证配置
	if err := rpa.ValidateScalingConfig(scalingConfig); err != nil {
		return fmt.Errorf("扩缩容配置无效: %w", err)
	}

	// 创建扩缩容服务
	scalingService := rpa.NewScalingService(c.GetDB(), scalingConfig, dockerConfig)

	// 启动扩缩容监控
	if err := scalingService.Start(context.Background()); err != nil {
		return err
	}

	c.RPAScalingService = scalingService
	applogger.Infof("RPA 扩缩容服务已启动 (检查间隔: %v, Worker范围: %d-%d)",
		scalingConfig.CheckInterval, scalingConfig.MinWorkers, scalingConfig.MaxWorkers)

	return nil
}

// parseDuration 解析时间间隔字符串
func parseDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		applogger.Warnf("解析时间间隔失败: %v, 使用默认值: %v", err, defaultVal)
		return defaultVal
	}
	return d
}
