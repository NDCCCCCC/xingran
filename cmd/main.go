package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/xingran-next/xingran-go-backend/internal/api"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/server"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	macHistoryQueryService "github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
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

// Version 版本号，构建时经 ldflags 注入：
//
//	go build -ldflags="-X main.Version=v1.2.3" ...
//
// 缺省 "dev"（本地 go run / 未注入的构建）。/health 端点与启动日志都会输出，
// 部署流水线据此校验发布是否生效（curl /health | grep 版本号）。
var Version = "dev"

func init() {
	loadDotEnv()
	setTimeZone()
}

// loadDotEnv 在程序启动早期加载项目根目录的 .env 文件
// (Go 不像 Node.js 自动加载 .env, 需要显式调用)
// 已通过 os.Setenv 注入的真实环境变量优先级高于 .env (godotenv.Load 默认不覆盖)
// 文件不存在时静默忽略 — 生产环境应通过真实环境变量注入而非 .env
func loadDotEnv() {
	if err := godotenv.Load(); err != nil {
		// 没有 .env 文件是合法情况(生产环境用真实环境变量), 仅在 debug 输出提示
		if os.Getenv("DEBUG_ENV") != "" {
			fmt.Printf("[init] .env 文件未加载: %v\n", err)
		}
	}
}

// setTimeZone 设置全局时区
func setTimeZone() {
	tz, err := time.LoadLocation(DefaultTimeZone)
	if err != nil {
		panic(fmt.Sprintf("加载时区失败: %v", err))
	}
	time.Local = tz
}

// @title Xingran Next API
// @version 1.0
// @description 企业级权限管理系统 API
// @termsOfService https://github.com/xingran-next/xingran-next

// @contact.name API Support
// @contact.url https://github.com/xingran-next/xingran-next/issues
// @contact.email support@xingran-next.com

// @license.name MIT
// @license.url https://github.com/xingran-next/xingran-next/blob/main/LICENSE

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description Bearer token
func main() {
	cfg, err := config.Load(context.Background())
	if err != nil {
		fmt.Printf("配置加载失败: %v\n", err)
		os.Exit(1)
	}

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
	applogger.Infof("版本: %s", Version)
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

	// P1 fix: 注入加密配置变更回调,避免 services/system → pkg/middleware 直接依赖
	// 形成循环 (system → middleware → core → system)。
	// 让 config_service.Update 修改 sys.request.encryption.enabled 后能立即失效中间件缓存。
	system.OnEncryptionConfigChanged = middleware.RefreshEncryptionConfigCache

	// 添加MAC历史保留期默认配置
	// SkipSetup 开关: 与 InitData 一致,dev 默认跑(幂等 count 检查),
	// 生产/CI 设 SERVER_SKIP_SETUP=true 时跳过这俩 cmd-level seed。
	if !coreModule.Config.Server.SkipSetup {
		initMACHistoryRetentionConfig(coreModule)
		// 导入OUI厂商数据
		importOUIData(coreModule)
	} else {
		applogger.Infof("[SkipSetup=true] 跳过 cmd-level seed (MAC history retention / OUI)")
	}

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
//
// R5 调整(2026-06-29): GIN-debug 路由注册日志太多(每次启动打印 100+ 行),
// 统一改写到 io.Discard(完全静默)。需要路由调试时,改这里临时改回 os.Stdout。
func setGinMode(mode string) {
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	// 关闭 GIN 内置 debug 日志(路由注册表 + 请求级 [GIN] 日志)，
	// 走项目自己的 logrus + middleware.Logger 中间件记录请求
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard
}

// setupRoutes 设置路由
func setupRoutes(engine *gin.Engine, cfg *config.Config, coreModule *core.Core, allowedOrigins []string) {
	// 健康检查
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": Version,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// pprof 性能分析端点（仅调试/开发环境）
	// 遵循 Go 最佳实践：在生产环境禁用 pprof
	if cfg.Server.Mode != "release" {
		// 注册 pprof 端点到默认的 ServeMux
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

	// 前端静态文件服务（嵌入式）
	// 注意：这些路由必须在 API 路由之后注册，
	// 否则 /assets/*filepath 可能会匹配到 /assets/... 的 API 路径
	engine.GET("/assets/*filepath", server.ServeFrontend)
	engine.GET("/index.html", server.ServeFrontend)
	engine.GET("/", server.ServeFrontend)

	// SPA catch-all：所有未匹配的路由返回 index.html
	// 这确保 /login, /dashboard, /system/users 等客户端路由正常工作
	// 必须在所有 API 和静态资源路由之后注册
	engine.NoRoute(server.ServeSPA)
}

// startServer 启动 HTTP 服务器
func startServer(cfg *config.Config, engine *gin.Engine) *http.Server {
	// Phase 62-DBG-01: 加 WriteTimeout=75s（前端 axios=60s，后端稍长以让前端先超时，
	// 同时避免向已关闭 socket 长时间 block；为 cache-miss 长查询场景提供 fast-fail 路径）。
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      engine,
		WriteTimeout: 75 * time.Second,
		ReadTimeout:  75 * time.Second,
	}

	go func() {
		applogger.Infof("服务器启动在端口: %d", cfg.Server.Port)
		applogger.Infof("环境: %s", cfg.Server.Mode)
		if cfg.Server.Mode != "release" {
			applogger.Infof("Swagger 文档: http://localhost:%d/swagger/index.html", cfg.Server.Port)
		}

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			applogger.Fatalf("服务器启动失败: %v", err)
		}
	}()

	return srv
}

// waitForShutdown 等待关闭信号并优雅关闭
func waitForShutdown(srv *http.Server, coreModule *core.Core) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	applogger.Info("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
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

// importOUIData 导入OUI厂商数据
func importOUIData(coreModule *core.Core) {
	db := coreModule.GetDB()
	historyQueryService := macHistoryQueryService.NewMACHistoryQueryService(db)

	applogger.Infof("开始导入OUI厂商数据...")
	if err := historyQueryService.ImportOUIData(context.Background()); err != nil {
		applogger.Warnf("导入OUI厂商数据失败: %v", err)
	}
}
