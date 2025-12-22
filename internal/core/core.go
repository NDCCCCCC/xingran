// Package core 提供系统的核心功能模块
// 包括数据库管理、缓存管理、JWT认证、密码加密等核心服务的初始化和管理
package core

import (
	"log"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/db"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/cache"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/permission"
	"gorm.io/gorm"
)

// Core 核心模块管理器
// 负责管理系统的所有核心服务组件，包括数据库连接、缓存、JWT管理、密码加密等
type Core struct {
	Config     *config.Config          // 系统配置
	DB         *db.Database           // 数据库连接管理器
	Cache      cache.Cache            // 缓存管理器
	JWTManager *security.JWTManager   // JWT令牌管理器
	PwdManager *security.PasswordManager // 密码管理器
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
	return &Core{
		Config:     cfg,
		JWTManager: security.NewJWTManager(&cfg.JWT),
		PwdManager: security.NewPasswordManager(nil),
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

	return nil
}

// Close 关闭核心模块
// 优雅关闭所有核心服务，释放资源
func (c *Core) Close() {
	if c.DB != nil {
		c.DB.Close()
	}
	if c.Cache != nil {
		c.Cache.Close()
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