package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/api"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	// 添加 pprof 支持，仅用于性能分析
	_ "net/http/pprof"
)

const (
	// DefaultTimeZone 默认时区：东八区（中国标准时间）
	DefaultTimeZone = "Asia/Shanghai"
	// ShutdownTimeout 服务器关闭超时时间
	ShutdownTimeout = 10 * time.Second
)

func init() {
	setTimeZone()
}

// setTimeZone 设置全局时区
func setTimeZone() {
	tz, err := time.LoadLocation(DefaultTimeZone)
	if err != nil {
		panic(fmt.Sprintf("加载时区失败: %v", err))
	}
	time.Local = tz
}

// @title RuoYi Next API
// @version 1.0
// @description 企业级权限管理系统 API
// @termsOfService https://github.com/ruoyi-next/ruoyi-next

// @contact.name API Support
// @contact.url https://github.com/ruoyi-next/ruoyi-next/issues
// @contact.email support@ruoyi-next.com

// @license.name MIT
// @license.url https://github.com/ruoyi-next/ruoyi-next/blob/main/LICENSE

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description Bearer token
func main() {
	cfg := config.Load()

	initializeLogger(cfg)
	logStartupInfo(cfg)

	coreModule := initializeCoreModule(cfg)
	engine := setupGinEngine(cfg)
	allowedOrigins := []string{"*"}
	setupRoutes(engine, cfg, coreModule, allowedOrigins)

	server := startServer(cfg, engine)
	waitForShutdown(server, coreModule)
}

// initializeLogger 初始化日志系统
func initializeLogger(cfg *config.Config) {
	logConfig := &applogger.Config{
		Level:         cfg.Log.Level,
		LogDir:        cfg.Log.LogDir,
		MaxSize:       cfg.Log.MaxSize,
		MaxBackups:    cfg.Log.MaxBackups,
		MaxAge:        cfg.Log.MaxAge,
		Compress:      cfg.Log.Compress,
		ConsoleOutput: true,
	}

	if err := applogger.Init(logConfig); err != nil {
		fmt.Printf("初始化日志系统失败: %v\n", err)
		os.Exit(1)
	}
}

// logStartupInfo 记录启动信息
func logStartupInfo(cfg *config.Config) {
	applogger.Info("========== 应用启动 ==========")
	applogger.Infof("日志级别: %s", cfg.Log.Level)
	applogger.Infof("日志目录: %s", cfg.Log.LogDir)
}

// initializeCoreModule 初始化核心模块
func initializeCoreModule(cfg *config.Config) *core.Core {
	coreModule, err := core.New(cfg)
	if err != nil {
		applogger.Fatalf("创建核心模块失败: %v", err)
	}
	if err := coreModule.Init(); err != nil {
		applogger.Fatalf("初始化核心模块失败: %v", err)
	}

	// 添加MAC历史保留期默认配置
	initMACHistoryRetentionConfig(coreModule)

	return coreModule
}

// setupGinEngine 设置 Gin 引擎
func setupGinEngine(cfg *config.Config) *gin.Engine {
	setGinMode(cfg.Server.Mode)

	engine := gin.New()
	engine.Use(middleware.Logger())
	engine.Use(middleware.Recovery())

	// CORS配置：生产环境应该配置具体的域名列表
	// 开发环境使用 "*" 允许所有来源
	engine.Use(middleware.Cors([]string{"*"}))

	engine.Use(middleware.Gzip())

	return engine
}

// setGinMode 设置 Gin 运行模式
func setGinMode(mode string) {
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
}

// setupRoutes 设置路由
func setupRoutes(engine *gin.Engine, cfg *config.Config, coreModule *core.Core, allowedOrigins []string) {
	// 健康检查
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// pprof 性能分析端点（仅调试/开发环境）
	// 遵循 Go 最佳实践：在生产环境禁用 pprof
	if cfg.Server.Mode != "release" {
		// 注册 pprof 端点到默认的 ServeMux
		// pprof 端点包括：
		//   /debug/pprof/         - pprof 首页
		//   /debug/pprof/goroutine - goroutine 堆栈
		//   /debug/pprof/heap      - 堆内存采样
		//   /debug/pprof/threadcreate - 线程创建堆栈
		//   /debug/pprof/block     - 阻塞堆栈
		//   /debug/pprof/mutex     - 互斥锁堆栈
		//   /debug/pprof/profile   - CPU profile
		//   /debug/pprof/trace     - 执行 trace
		//   /debug/pprof/cmdline   - 命令行
		//   /debug/pprof/symbol    - 符号表
		engine.GET("/debug/pprof/*any", gin.WrapH(http.DefaultServeMux))
		applogger.Infof("pprof 性能分析端点已启用: http://localhost:%d/debug/pprof/", cfg.Server.Port)
	}

	// Swagger 文档（仅非生产环境）
	if cfg.Server.Mode != "release" {
		engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// 静态文件服务
	engine.Static("/uploads", "./uploads")

	// API 路由
	apiRouter := engine.Group("/api/v1")
	api.SetupRouter(apiRouter, coreModule, allowedOrigins)
}

// startServer 启动 HTTP 服务器
func startServer(cfg *config.Config, engine *gin.Engine) *http.Server {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: engine,
	}

	go func() {
		applogger.Infof("服务器启动在端口: %d", cfg.Server.Port)
		applogger.Infof("环境: %s", cfg.Server.Mode)
		if cfg.Server.Mode != "release" {
			applogger.Infof("Swagger 文档: http://localhost:%d/swagger/index.html", cfg.Server.Port)
		}

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			applogger.Fatalf("服务器启动失败: %v", err)
		}
	}()

	return server
}

// waitForShutdown 等待关闭信号并优雅关闭
func waitForShutdown(server *http.Server, coreModule *core.Core) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	applogger.Info("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		applogger.Errorf("服务器强制关闭: %v", err)
	}

	coreModule.Close()
	applogger.Close()

	applogger.Info("服务器已退出")
}

// initMACHistoryRetentionConfig 初始化MAC历史保留期配置
func initMACHistoryRetentionConfig(coreModule *core.Core) {
	db := coreModule.GetDB()

	applogger.Infof("开始检查MAC历史保留期配置...")

	// 检查配置是否已存在
	var count int64
	err := db.Table("sys_config").
		Where("config_key = ?", "network.mac.history.retention_days").
		Count(&count).Error

	if err != nil {
		applogger.Warnf("查询MAC历史保留期配置失败: %v", err)
		return
	}

	if count > 0 {
		applogger.Infof("MAC历史保留期配置已存在，跳过初始化")
		return
	}

	// 插入默认配置：120天
	config := models.Config{
		ConfigName:  "MAC地址历史数据保留期",
		ConfigKey:   "network.mac.history.retention_days",
		ConfigValue: "120",
		ConfigType:  "Y",
		IsSystem:    1,
		Remark:      "MAC地址历史数据保留天数（90天数据+30天缓冲），默认120天，最小30天",
	}

	applogger.Infof("尝试插入MAC历史保留期配置...")
	if err := db.Create(&config).Error; err != nil {
		applogger.Warnf("创建MAC历史保留期配置失败: %v", err)
	} else {
		applogger.Infof("MAC历史保留期配置已创建（默认120天）")
	}
}
