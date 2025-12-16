package core

import (
	"log"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/cache"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/db"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
)

// Core 核心模块管理器
type Core struct {
	Config     *config.Config
	DB         *db.Database
	Redis      *cache.Redis
	JWTManager *security.JWTManager
	PwdManager *security.PasswordManager
}

// New 创建核心模块管理器
func New(cfg *config.Config) *Core {
	return &Core{
		Config:     cfg,
		JWTManager: security.NewJWTManager(&cfg.JWT),
		PwdManager: security.NewPasswordManager(nil),
	}
}

// Init 初始化核心模块
func (c *Core) Init() error {
	// 初始化数据库
	var err error
	c.DB, err = db.NewDatabase(&c.Config.Database)
	if err != nil {
		log.Printf("警告：数据库初始化失败: %v", err)
		// 暂时跳过数据库初始化
		return nil
	}

	// 自动迁移数据库表
	if err := c.DB.AutoMigrate(); err != nil {
		log.Printf("警告：数据库迁移失败: %v", err)
	}

	// 初始化基础数据
	if err := c.DB.InitData(); err != nil {
		log.Printf("警告：初始化基础数据失败: %v", err)
		// 如果数据初始化失败，但不影响服务启动
	} else {
		log.Println("基础数据初始化成功")
	}

	// 初始化Redis
	c.Redis, err = cache.NewRedis(&c.Config.Redis)
	if err != nil {
		log.Printf("警告：初始化Redis失败: %v", err)
		// Redis失败不影响服务启动
	}

	return nil
}

// Close 关闭核心模块
func (c *Core) Close() {
	if c.DB != nil {
		c.DB.Close()
	}
	if c.Redis != nil {
		c.Redis.Close()
	}
}