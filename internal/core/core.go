// Package core 提供系统的核心功能模块
// 包括数据库管理、缓存管理、JWT认证、密码加密等核心服务的初始化和管理
package core

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/db"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/scheduler"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/websocket"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/cache"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/permission"
	"gorm.io/gorm"
)

// Core 核心模块管理器
// 负责管理系统的所有核心服务组件，包括数据库连接、缓存、JWT管理、密码加密等
type Core struct {
	Config                      *config.Config                        // 系统配置
	DB                          *db.Database                          // 数据库连接管理器
	Cache                       cache.Cache                           // 缓存管理器
	JWTManager                  *security.JWTManager                  // JWT令牌管理器
	PwdManager                  *security.PasswordManager             // 密码管理器
	SM4Cipher                   *security.SM4Cipher                   // SM4加密器（用于凭证密码加密）
	MetricsCacheService         *MetricsCacheService                  // 系统指标缓存服务
	Scheduler                   *scheduler.Scheduler                  // 定时任务调度器
	DeviceManager               *device.Manager                       // 网络设备管理器（简化版，仅辅助方法）
	DeviceExecutor              *device.DeviceExecutor                // 设备执行器（新架构，统一入口）
	DeviceDiscoveryService      *services.DeviceDiscoveryService      // 设备发现服务
	DeviceInfoCollectionService *services.DeviceInfoCollectionService // 设备信息采集服务
	NetworkDeviceService        *services.NetworkDeviceService        // 网络设备服务
	DeviceMonitorService        *services.DeviceMonitorService        // 设备监控服务
	NoticeHub                   *websocket.NoticeHub                  // WebSocket通知中心
	CaptchaService              *CaptchaService                       // 验证码服务
	CaptchaBackgroundService    *CaptchaBackgroundService             // 验证码背景图服务
	OperLogService              services.OperLogService               // 操作日志服务
	TokenBlacklistService       services.TokenBlacklistService        // 令牌黑名单服务

	// 缓存服务（用于 System 模块）
	DataCacheService   *services.DataCacheService   // 通用数据缓存服务
	CacheConfigService *services.CacheConfigService // 缓存配置服务
	CacheManager       *system.CacheManager         // 缓存管理器（增强功能）

	// System 模块服务（用于缓存预热）
	UserService     system.UserService
	RoleService     system.RoleService
	MenuService     system.MenuService
	DeptService    system.DepartmentService
	PostService    system.PostService
	ConfigService  system.ConfigService
	SettingsService system.SettingsService

	// API端点元数据服务
	APIMetadata        *config.APIMetadataConfig    // API元数据配置
	APIEndpointService *services.APIEndpointService // API端点服务
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
	sm4Cipher := initSM4Cipher(cfg.Security.SM4Key)

	// 初始化JWT管理器
	jwtManager, err := security.NewJWTManager(&cfg.JWT)
	if err != nil {
		return nil, fmt.Errorf("初始化JWT管理器失败: %w", err)
	}

	return &Core{
		Config:     cfg,
		JWTManager: jwtManager,
		PwdManager: security.NewPasswordManager(nil),
		SM4Cipher:  sm4Cipher,
	}, nil
}

// initSM4Cipher 初始化SM4加密器
func initSM4Cipher(sm4Key string) *security.SM4Cipher {
	sm4Cipher, err := security.NewSM4Cipher(sm4Key)
	if err != nil {
		applogger.Errorf("SM4 加密器初始化失败: %v", err)
		applogger.Warnf("配置的密钥长度: %d", len(sm4Key))
		return nil
	}
	applogger.Infof("SM4 加密器初始化成功")
	return sm4Cipher
}

// Init 初始化核心模块
// 按顺序初始化数据库、基础数据、权限服务和缓存系统
// 如果某个模块初始化失败，会记录警告但不会中断服务启动
func (c *Core) Init() error {
	// 1. 初始化数据库连接
	var err error
	c.DB, err = db.NewDatabase(&c.Config.Database)
	if err != nil {
		applogger.Errorf("数据库初始化失败: %v", err)
		// 数据库失败时跳过后续数据库相关操作
		return nil
	}

	// 2. 自动迁移数据库表结构
	// 根据模型定义自动创建或更新数据库表
	if err := c.DB.AutoMigrate(); err != nil {
		applogger.Warnf("数据库迁移失败: %v", err)
	}

	// 3. 初始化系统基础数据
	// 包括创建默认管理员账户、基础角色等
	if err := c.DB.InitData(); err != nil {
		applogger.Warnf("初始化基础数据失败: %v", err)
		// 数据初始化失败不影响服务启动
	} else {
		applogger.Infof("基础数据初始化成功")
	}

	// 4. 初始化权限服务
	// 创建默认角色和菜单权限
	if c.DB != nil {
		permissionSvc := permission.NewService()
		if err := permissionSvc.InitDefaultRolesAndMenus(c.GetDB()); err != nil {
			applogger.Warnf("初始化默认角色和菜单失败: %v", err)
		} else {
			applogger.Infof("默认角色和菜单初始化成功")
		}
	}

	// 5. 初始化缓存系统
	// 支持Redis多级缓存或纯内存缓存
	c.Cache, err = c.initCache()
	if err != nil {
		applogger.Warnf("初始化缓存系统失败: %v", err)
		// 缓存失败不影响服务启动
	} else {
		applogger.Infof("缓存系统初始化完成")
	}

	// 6. 初始化缓存服务（用于 System 模块）
	// System 模块的服务通过 CacheProvider 接口使用缓存
	if c.Cache != nil {
		// 初始化缓存配置服务
		c.CacheConfigService = services.NewCacheConfigService(c.GetDB())

		// 初始化数据缓存服务（供 System 模块适配使用）
		c.DataCacheService = services.NewDataCacheService(c.Cache)
		c.DataCacheService.SetCacheConfig(c.CacheConfigService)

		// 初始化缓存管理器（增强功能）
		// 缓存预热功能默认禁用，可通过配置启用
		cacheManagerEnabled := c.Config.Cache.WarmUpEnabled
		c.CacheManager = system.NewCacheManager(
			system.NewCacheAdapter(c.Cache),
			"ruoyi", // 缓存键前缀
			cacheManagerEnabled,
		)

		applogger.Infof("缓存服务初始化完成（预热功能: %v）", cacheManagerEnabled)

		// 初始化 System 模块服务（用于缓存预热）
		c.initSystemServicesForWarmUp()

		// 执行缓存预热（异步执行，不阻塞启动）
		if cacheManagerEnabled {
			go c.performCacheWarmUp(context.Background())
		}
	}

	// 7. 初始化系统指标缓存服务
	c.MetricsCacheService = NewMetricsCacheService(c)
	applogger.Infof("系统指标缓存服务初始化完成")

	// 8. 初始化网络设备管理器（新架构）
	// 7.1 创建简化版管理器（仅辅助方法）
	c.DeviceManager = device.NewManager(c.GetDB())
	// 7.2 创建连接池
	poolConfig := &device.PoolConfig{
		MaxIdle:        5 * time.Minute,
		MaxConnections: 20, // 降低并发连接数，减少 scrapligo panic 问题
	}
	connectionPool := device.NewDeviceConnectionPool(c.GetDB(), c.SM4Cipher, poolConfig)
	// 7.3 创建任务调度器
	schedulerConfig := &device.SchedulerConfig{
		TaskTimeout: 5 * time.Minute,
		QueueSize:   100,
	}
	taskScheduler := device.NewDeviceTaskScheduler(connectionPool, schedulerConfig)
	// 7.4 创建执行器
	executorConfig := &device.ExecutionConfig{
		MaxRetries:          2,
		RetryDelay:          time.Second,
		Timeout:             30 * time.Second,
		EnablePanicRecovery: true,
	}
	c.DeviceExecutor = device.NewDeviceExecutor(taskScheduler, executorConfig)
	// 7.5 将执行器设置到管理器（兼容层）
	c.DeviceManager.SetExecutor(c.DeviceExecutor)
	applogger.Infof("网络设备管理器（新架构）初始化完成")

	// 8. 初始化设备发现服务
	c.DeviceDiscoveryService = services.NewDeviceDiscoveryService(c.GetDB())
	applogger.Infof("设备发现服务初始化完成")

	// 9. 初始化设备信息采集服务（后台异步，统一处理设备创建和定时任务）
	c.DeviceInfoCollectionService = services.NewDeviceInfoCollectionService(c.GetDB(), c.DeviceExecutor)
	if err := c.DeviceInfoCollectionService.Start(context.Background()); err != nil {
		applogger.Warnf("启动设备信息采集服务失败: %v", err)
	} else {
		applogger.Infof("设备信息采集服务启动成功")
	}

	// 10. 初始化网络设备服务
	c.NetworkDeviceService = services.NewNetworkDeviceService(c.GetDB(), c.DeviceDiscoveryService, c.DeviceInfoCollectionService)
	applogger.Infof("网络设备服务初始化完成")

	// 11. 初始化定时任务调度器
	c.Scheduler = scheduler.NewScheduler(c.GetDB())
	if err := c.Scheduler.Start(context.Background()); err != nil {
		applogger.Warnf("启动调度器失败: %v", err)
	} else {
		applogger.Infof("定时任务调度器启动成功")

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

		// 启动AD域同步调度器（每5分钟检查需要同步的配置）
		scheduler.StartADSyncScheduler(c.GetDB())

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

	// 15. 初始化操作日志服务
	c.OperLogService = services.NewOperLogService()
	applogger.Infof("操作日志服务初始化完成")

	// 16. 初始化令牌黑名单服务
	c.TokenBlacklistService = services.NewTokenBlacklistService(c.Cache)
	applogger.Infof("令牌黑名单服务初始化完成")

	// 17. 初始化API端点元数据服务
	if err := c.initAPIEndpointService(); err != nil {
		applogger.Warnf("初始化API端点元数据服务失败: %v", err)
	} else {
		applogger.Infof("API端点元数据服务初始化完成")
	}

	return nil
}

// Close 关闭核心模块
// 优雅关闭所有核心服务，释放资源
func (c *Core) Close() {
	// 停止通知中心
	if c.NoticeHub != nil {
		c.NoticeHub.Stop()
		applogger.Infof("通知中心已停止")
	}
	// 停止设备信息采集服务
	if c.DeviceInfoCollectionService != nil {
		c.DeviceInfoCollectionService.Stop()
		applogger.Infof("设备信息采集服务已停止")
	}
	// 停止AD域同步调度器
	scheduler.StopADSyncScheduler()
	// 关闭设备监控服务
	if c.DeviceMonitorService != nil {
		c.DeviceMonitorService.Close()
		applogger.Infof("设备监控服务已关闭")
	}
	// 停止定时任务调度器
	if c.Scheduler != nil {
		c.Scheduler.Stop()
		applogger.Infof("定时任务调度器已停止")
	}
	// 关闭网络设备连接管理器
	if c.DeviceManager != nil {
		c.DeviceManager.Close()
		applogger.Infof("网络设备连接管理器已关闭")
	}
	if c.DB != nil {
		c.DB.Close()
	}
	if c.Cache != nil {
		c.Cache.Close()
	}
	if c.MetricsCacheService != nil {
		c.MetricsCacheService.Stop()
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
		}

		// 创建Redis缓存实例作为L2缓存
		redisCache, err := cache.NewRedisCache(cacheConfig, "ruoyi")
		if err != nil {
			return nil, err
		}

		// 创建内存缓存作为L1缓存，提供快速访问
		// 容量：1000个键值对，过期时间：10分钟
		memoryCache := cache.NewMemoryCache(1000, 10*time.Minute)

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
					WorkerCount:         c.Config.Cache.L2Writer.WorkerCount,
					QueueSize:           c.Config.Cache.L2Writer.QueueSize,
					EnqueueTimeout:      c.Config.Cache.L2Writer.EnqueueTimeout,
					WriteTimeout:        c.Config.Cache.L2Writer.WriteTimeout,
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
				WorkerCount:         c.Config.Cache.L2Writer.WorkerCount,
				QueueSize:           c.Config.Cache.L2Writer.QueueSize,
				EnqueueTimeout:      c.Config.Cache.L2Writer.EnqueueTimeout,
				WriteTimeout:        c.Config.Cache.L2Writer.WriteTimeout,
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
func (c *Core) initAPIEndpointService() error {
	// 加载API元数据配置
	metadata, err := config.LoadAPIMetadata("./configs/api_metadata.yaml")
	if err != nil {
		return err
	}
	c.APIMetadata = metadata

	// 创建API端点服务
	c.APIEndpointService = services.NewAPIEndpointService(
		metadata,
		c.Cache,
		c.GetDB(),
	)

	return nil
}

// initSystemServicesForWarmUp 初始化 System 模块服务（用于缓存预热）
func (c *Core) initSystemServicesForWarmUp() {
	db := c.GetDB()

	// 创建缓存提供者
	cacheProvider := system.NewCacheProvider(c.DataCacheService)

	// 统一创建带缓存的服务（与 Router 中的逻辑保持一致）
	c.UserService = system.NewUserServiceWithCache(db, cacheProvider, c.CacheConfigService, c.PwdManager)
	c.RoleService = system.NewRoleServiceWithCache(db, cacheProvider, c.CacheConfigService)
	c.MenuService = system.NewMenuServiceWithCache(db, cacheProvider, c.CacheConfigService)
	c.DeptService = system.NewDepartmentServiceWithCache(db, cacheProvider, c.CacheConfigService)
	c.PostService = system.NewPostServiceWithCache(db, cacheProvider, c.CacheConfigService)
	c.ConfigService = system.NewConfigServiceWithCache(db, cacheProvider, c.CacheConfigService)
	c.SettingsService = system.NewSettingsServiceWithCache(db, cacheProvider, c.CacheConfigService)

	applogger.Infof("System 服务初始化完成（用于缓存预热）")
}

// performCacheWarmUp 执行缓存预热
func (c *Core) performCacheWarmUp(ctx context.Context) {
	// 等待一小段时间，确保服务已完全启动
	time.Sleep(2 * time.Second)

	applogger.Infof("开始执行缓存预热...")

	warmUpFuncs := map[string]system.WarmUpFunc{
		"user": system.WarmUpUserCache(c.UserService),
		"role": system.WarmUpRoleCache(c.RoleService),
		"menu": system.WarmUpMenuCache(c.MenuService),
		"dept": system.WarmUpDeptCache(c.DeptService),
		"post": system.WarmUpPostCache(c.PostService),
	}

	if err := c.CacheManager.WarmUp(ctx, warmUpFuncs); err != nil {
		applogger.Warnf("缓存预热失败: %v", err)
	} else {
		applogger.Infof("缓存预热成功完成")
	}
}
