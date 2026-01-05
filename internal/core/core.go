// Package core 提供系统的核心功能模块
// 包括数据库管理、缓存管理、JWT认证、密码加密等核心服务的初始化和管理
package core

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/db"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/scheduler"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/websocket"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/cache"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/permission"
	"gorm.io/gorm"
)

// Core 核心模块管理器
// 负责管理系统的所有核心服务组件，包括数据库连接、缓存、JWT管理、密码加密等
type Core struct {
	Config                  *config.Config            // 系统配置
	DB                      *db.Database             // 数据库连接管理器
	Cache                   cache.Cache              // 缓存管理器
	JWTManager              *security.JWTManager     // JWT令牌管理器
	PwdManager              *security.PasswordManager // 密码管理器
	SM4Cipher               *security.SM4Cipher      // SM4加密器（用于凭证密码加密）
	MetricsCacheService     *MetricsCacheService      // 系统指标缓存服务
	Scheduler               *scheduler.Scheduler      // 定时任务调度器
	DeviceManager           *device.Manager           // 网络设备管理器（简化版，仅辅助方法）
	DeviceExecutor          *device.DeviceExecutor    // 设备执行器（新架构，统一入口）
	DeviceDiscoveryService  *services.DeviceDiscoveryService  // 设备发现服务
	DeviceEnrichmentService *services.DeviceEnrichmentService // 设备信息补充服务
	NetworkDeviceService    *services.NetworkDeviceService    // 网络设备服务
	DeviceMonitorService    *services.DeviceMonitorService    // 设备监控服务
	NoticeHub               *websocket.NoticeHub     // WebSocket通知中心
	CaptchaService          *CaptchaService          // 验证码服务
	CaptchaBackgroundService *CaptchaBackgroundService // 验证码背景图服务
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
func New(cfg *config.Config) *Core {
	// 初始化 SM4 加密器
	sm4Cipher, err := security.NewSM4Cipher(cfg.Security.SM4Key)
	if err != nil {
		log.Printf("警告：SM4 加密器初始化失败: %v", err)
		log.Printf("配置的密钥长度: %d", len(cfg.Security.SM4Key))
		// 继续启动，但密码加密功能将不可用
	} else {
		log.Printf("SM4 加密器初始化成功")
	}

	return &Core{
		Config:     cfg,
		JWTManager: security.NewJWTManager(&cfg.JWT),
		PwdManager: security.NewPasswordManager(nil),
		SM4Cipher:  sm4Cipher,
	}
}

// Init 初始化核心模块
// 按顺序初始化数据库、基础数据、权限服务和缓存系统
// 如果某个模块初始化失败，会记录警告但不会中断服务启动
func (c *Core) Init() error {
	// 1. 初始化数据库连接
	var err error
	c.DB, err = db.NewDatabase(&c.Config.Database)
	if err != nil {
		log.Printf("警告：数据库初始化失败: %v", err)
		// 数据库失败时跳过后续数据库相关操作
		return nil
	}

	// 2. 自动迁移数据库表结构
	// 根据模型定义自动创建或更新数据库表
	if err := c.DB.AutoMigrate(); err != nil {
		log.Printf("警告：数据库迁移失败: %v", err)
	}

	// 3. 初始化系统基础数据
	// 包括创建默认管理员账户、基础角色等
	if err := c.DB.InitData(); err != nil {
		log.Printf("警告：初始化基础数据失败: %v", err)
		// 数据初始化失败不影响服务启动
	} else {
		log.Println("基础数据初始化成功")
	}

	// 4. 初始化权限服务
	// 创建默认角色和菜单权限
	if c.DB != nil {
		permissionSvc := permission.NewService()
		if err := permissionSvc.InitDefaultRolesAndMenus(c.GetDB()); err != nil {
			log.Printf("警告：初始化默认角色和菜单失败: %v", err)
		} else {
			log.Println("默认角色和菜单初始化成功")
		}
	}

	// 5. 初始化缓存系统
	// 支持Redis多级缓存或纯内存缓存
	c.Cache, err = c.initCache()
	if err != nil {
		log.Printf("警告：初始化缓存系统失败: %v", err)
		// 缓存失败不影响服务启动
	}

	// 6. 初始化系统指标缓存服务
	c.MetricsCacheService = NewMetricsCacheService(c)
	log.Println("系统指标缓存服务初始化完成")

	// 7. 初始化网络设备管理器（新架构）
	// 7.1 创建简化版管理器（仅辅助方法）
	c.DeviceManager = device.NewManager(c.GetDB())
	// 7.2 创建连接池
	poolConfig := &device.PoolConfig{
		MaxIdle:        5 * time.Minute,
		MaxConnections: 50,
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
	log.Println("网络设备管理器（新架构）初始化完成")

	// 8. 初始化设备发现服务
	c.DeviceDiscoveryService = services.NewDeviceDiscoveryService(c.GetDB())
	log.Println("设备发现服务初始化完成")

	// 9. 初始化设备信息补充服务（后台异步）
	c.DeviceEnrichmentService = services.NewDeviceEnrichmentService(c.GetDB(), c.DeviceManager, 5)
	if err := c.DeviceEnrichmentService.Start(context.Background()); err != nil {
		log.Printf("警告：启动设备信息补充服务失败: %v", err)
	} else {
		log.Println("设备信息补充服务启动成功")
	}

	// 10. 初始化网络设备服务
	c.NetworkDeviceService = services.NewNetworkDeviceService(c.GetDB(), c.DeviceDiscoveryService, c.DeviceEnrichmentService)
	log.Println("网络设备服务初始化完成")

	// 11. 初始化定时任务调度器
	c.Scheduler = scheduler.NewScheduler(c.GetDB())
	if err := c.Scheduler.Start(context.Background()); err != nil {
		log.Printf("警告：启动调度器失败: %v", err)
	} else {
		log.Println("定时任务调度器启动成功")

		// 设置全局调度器引用（供定时任务使用）
		scheduler.SetGlobalScheduler(c.Scheduler)

		// 注册网络设备相关定时任务
		scheduler.RegisterNetworkDeviceTasks(c.Scheduler, c.GetDB())
		log.Println("网络设备定时任务注册完成")

		// 注册通知相关定时任务
		scheduler.RegisterNoticeTasks(c.Scheduler)
		log.Println("通知定时任务注册完成")

		// 注册运维工单相关定时任务
		scheduler.RegisterWorkOrderTasks(c.Scheduler)
		log.Println("运维工单定时任务注册完成")

		// 同步周期性工单模板到调度器
		if err := scheduler.SyncPeriodicWorkOrderJobs(c.Scheduler); err != nil {
			log.Printf("警告：同步周期性工单任务失败: %v", err)
		} else {
			log.Println("周期性工单任务同步完成")
		}

		// 设置全局数据库访问器，供定时任务使用
		scheduler.SetDB(c)
	}

	// 12. 初始化设备监控服务
	c.DeviceMonitorService = services.NewDeviceMonitorService(c.GetDB(), c.SM4Cipher, services.DefaultDeviceMonitorConfig())
	// 设置执行器并初始化子服务
	c.DeviceMonitorService.SetExecutor(c.DeviceExecutor)
	// 设置设备监控服务到调度器，供定时任务调用
	scheduler.SetDeviceMonitorService(c.DeviceMonitorService)
	log.Println("设备监控服务初始化完成")

	// 13. 初始化验证码服务
	c.CaptchaService = NewCaptchaService(c.DB, c.Cache)
	if err := c.CaptchaService.LoadConfig(context.Background()); err != nil {
		log.Printf("警告：加载验证码配置失败: %v", err)
	} else {
		log.Printf("验证码服务初始化完成，类型: %s", c.CaptchaService.GetConfig().Enabled)
	}

	// 14. 初始化验证码背景图服务
	c.CaptchaBackgroundService = NewCaptchaBackgroundService(c.DB, c.Cache)
	// 将背景图服务设置到验证码服务
	c.CaptchaService.SetBackgroundService(c.CaptchaBackgroundService)

	// 14.1 确保存储目录存在
	storagePath := "./uploads/captcha/backgrounds"
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		log.Printf("警告：创建验证码背景图存储目录失败: %v", err)
	} else {
		log.Printf("验证码背景图存储目录已创建: %s", storagePath)
	}

	log.Println("验证码背景图服务初始化完成")

	return nil
}

// Close 关闭核心模块
// 优雅关闭所有核心服务，释放资源
func (c *Core) Close() {
	// 停止通知中心
	if c.NoticeHub != nil {
		c.NoticeHub.Stop()
		log.Println("通知中心已停止")
	}
	// 停止设备信息补充服务
	if c.DeviceEnrichmentService != nil {
		c.DeviceEnrichmentService.Stop()
		log.Println("设备信息补充服务已停止")
	}
	// 关闭设备监控服务
	if c.DeviceMonitorService != nil {
		c.DeviceMonitorService.Close()
		log.Println("设备监控服务已关闭")
	}
	// 停止定时任务调度器
	if c.Scheduler != nil {
		c.Scheduler.Stop()
		log.Println("定时任务调度器已停止")
	}
	// 关闭网络设备连接管理器
	if c.DeviceManager != nil {
		c.DeviceManager.Close()
		log.Println("网络设备连接管理器已关闭")
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
		return cache.NewMultiLevelCache(memoryCache, redisCache), nil
	} else {
		// 创建纯内存缓存
		// 容量：配置中指定的大小，过期时间：配置中指定的清理时间
		return cache.NewMemoryCache(c.Config.Cache.MaxSize, time.Duration(c.Config.Cache.CleanupTime)*time.Second), nil
	}
}