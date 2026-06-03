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
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/rpa"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/vdi"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/websocket"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/cache"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/crypto"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/permission"
	"gorm.io/gorm"
)

// rpaScalingService RPA扩缩容服务最小接口
// 在 core 包内定义以避免循环依赖，rpa.ScalingService 自动满足此接口
type rpaScalingService interface {
	Stop()
}

// warmUpServices 缓存预热所需的服务集合（仅在初始化期间使用）
type warmUpServices struct {
	UserService     system.UserService
	RoleService     system.RoleService
	MenuService     system.MenuService
	DeptService     system.DepartmentService
	PostService     system.PostService
}

// Core 核心模块管理器
// 负责管理系统的所有核心服务组件，包括数据库连接、缓存、JWT管理、密码加密等
type Core struct {
	Config                      *config.Config                        // 系统配置
	DB                          *db.Database                          // 数据库连接管理器
	Cache                       cache.Cache                           // 缓存管理器
	JWTManager                  *security.JWTManager                  // JWT令牌管理器
	PwdManager                  *security.PasswordManager             // 密码管理器
	SM4Cipher                   addomain.PasswordCipher               // SM4加密器接口（用于AD域密码加密）
	MetricsCacheService         *MetricsCacheService                  // 系统指标缓存服务（被 monitor/adapters.go 使用）
	Scheduler                   *scheduler.Scheduler                  // 定时任务调度器
	DeviceExecutor              *device.DeviceExecutor                // 设备执行器（被 mac_handler.go, network_export_handler.go, port_handler.go 使用）
	DeviceDiscoveryService      *services.DeviceDiscoveryService      // 设备发现服务（被 network_export_handler.go 使用）
	DeviceInfoCollectionService *services.DeviceInfoCollectionService // 设备信息采集服务（被 network_export_handler.go 使用）
	DeviceMonitorService        *services.DeviceMonitorService        // 设备监控服务（被 scheduler.SetDeviceMonitorService() 使用）
	NoticeHub                   *websocket.NoticeHub                  // WebSocket通知中心（被 router.go 和 rpa_router.go 使用）
	CaptchaService              *CaptchaService                       // 验证码服务（被 auth.go 和 captcha_handler.go 使用）
	CaptchaBackgroundService    *CaptchaBackgroundService             // 验证码背景图服务（被 captcha_background_handler.go 使用）
	OperLogService              services.OperLogService               // 操作日志服务（被 middleware 和 helper.go 使用）
	TokenBlacklistService       services.TokenBlacklistService        // 令牌黑名单服务（被 middleware 和 auth.go 使用）
	AuthFactory                 *security.AuthStrategyFactory         // 认证策略工厂（被 auth.go 使用）

	// 缓存服务（用于 System 模块）
	DataCacheService   *services.DataCacheService   // 通用数据缓存服务
	CacheConfigService *services.CacheConfigService // 缓存配置服务
	CacheManager       *system.CacheManager         // 缓存管理器（增强功能）

	// RPA 服务
	RPAScalingService rpaScalingService // RPA 扩缩容服务（内部使用，仅在 Close() 中停止）

	// API端点元数据服务
	APIEndpointService *services.APIEndpointService // API端点服务（被 dashboard_router.go 使用）

	// MAC历史分区服务
	PartitionService services.PartitionService // MAC历史表分区管理服务
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
// 使用配置中的SM4密钥创建加密器，用于AD域密码加解密
func initSM4Cipher(sm4Key string) addomain.PasswordCipher {
	if sm4Key == "" {
		applogger.Warnf("SM4 密钥未配置，AD域密码加解密将不可用")
		return nil
	}
	cipher, err := crypto.NewSM4Cipher(sm4Key)
	if err != nil {
		applogger.Errorf("初始化SM4加密器失败: %v，AD域密码加解密将不可用", err)
		return nil
	}
	applogger.Infof("SM4加密器初始化成功")
	return cipher
}


// Init 初始化核心模块
// 按顺序初始化数据库、基础数据、权限服务和缓存系统
// 如果某个模块初始化失败，会记录警告但不会中断服务启动
func (c *Core) Init() error {
	// 0. 设置 AD 域全局 SM4 cipher（FIX-01: 支持 SM4 密码加密）
	if c.SM4Cipher != nil {
		addomain.SetADSM4Cipher(c.SM4Cipher)
		applogger.Infof("AD 域 SM4 加密器已设置")
	} else {
		applogger.Warnf("AD 域 SM4 加密器未设置，密码将使用 AES-legacy 回退")
	}

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

		// 初始化缓存预热服务
		warmUpSvcs := c.initSystemServicesForWarmUp()

		// 执行缓存预热（异步执行，不阻塞启动）
		if cacheManagerEnabled {
			go c.performCacheWarmUp(context.Background(), warmUpSvcs)
		}
	}

	// 7. 初始化系统指标缓存服务
	c.MetricsCacheService = NewMetricsCacheService(c)
	applogger.Infof("系统指标缓存服务初始化完成")

	// 8. 初始化网络设备管理器（新架构）
	// 8.1 创建连接池
	poolConfig := &device.PoolConfig{
		MaxIdle:        5 * time.Minute,
		MaxConnections: 20, // 降低并发连接数，减少 scrapligo panic 问题
	}
	connectionPool := device.NewDeviceConnectionPool(c.GetDB(), c.SM4Cipher, poolConfig)
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

	// 9. 初始化设备信息采集服务（后台异步，统一处理设备创建和定时任务）
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

	// 10. 初始化定时任务调度器
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

		// 注册MAC历史清理任务
		if c.PartitionService != nil {
			scheduler.SetPartitionService(c.PartitionService)
			scheduler.RegisterMACHistoryTasks(c.Scheduler, c.GetDB())
			applogger.Infof("MAC历史清理任务注册完成")
		}

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
	// 16.5 初始化认证策略工厂
	c.initAuthFactory()
	applogger.Infof("认证策略工厂初始化完成")

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
	// 停止系统指标缓存服务
	if c.MetricsCacheService != nil {
		c.MetricsCacheService.Stop()
		applogger.Infof("系统指标缓存服务已停止")
	}
	// 停止 RPA 扩缩容服务
	if c.RPAScalingService != nil {
		c.RPAScalingService.Stop()
		applogger.Infof("RPA 扩缩容服务已停止")
	}
	// 关闭缓存（必须在数据库之前关闭，因为缓存可能有异步写入操作）
	if c.Cache != nil {
		c.Cache.Close()
		applogger.Infof("缓存已关闭")
	}
	// 等待一小段时间，确保所有异步操作完成
	// 这包括：操作日志异步写入、缓存异步写入等
	time.Sleep(100 * time.Millisecond)
	// 最后关闭数据库连接
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
func (c *Core) initAPIEndpointService() error {
	// 加载API元数据配置
	metadata, err := config.LoadAPIMetadata("./configs/api_metadata.yaml")
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
		userSyncService := system.NewUserSyncService(c.GetDB(), c.PwdManager, mapper)
	c.AuthFactory.SetUserSyncer(userSyncService)

	applogger.Infof("认证策略工厂初始化完成（支持 local/ad/hybrid 模式）")
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
	time.Sleep(2 * time.Second)

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
		ScaleDownCooldown: parseDuration(rpaConfig.Scaling.ScaleDownCooldown, 5*time.Minute),
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
