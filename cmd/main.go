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
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// 时区常量
const (
	// DefaultTimeZone 默认时区：东八区（中国标准时间）
	DefaultTimeZone = "Asia/Shanghai"
)

func init() {
	// 设置全局时区
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
	// 加载配置
	cfg := config.Load()

	// 初始化日志系统
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

	applogger.Info("========== 应用启动 ==========")
	applogger.Infof("日志级别: %s", cfg.Log.Level)
	applogger.Infof("日志目录: %s", cfg.Log.LogDir)

	// 初始化核心模块
	coreModule := core.New(cfg)
	if err := coreModule.Init(); err != nil {
		applogger.Fatalf("初始化核心模块失败: %v", err)
	}

	// 设置 Gin 模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// 创建 Gin 引擎
	r := gin.New()

	// 添加中间件
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.Cors())
	r.Use(middleware.Gzip())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time": time.Now().Format(time.RFC3339),
		})
	})

	// Swagger 文档
	if cfg.Server.Mode != "release" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// 静态文件服务 - 验证码背景图访问
	r.Static("/uploads/captcha/backgrounds", "./uploads/captcha/backgrounds")

	// API 路由
	apiRouter := r.Group("/api/v1")
	{
		api.SetupRouter(apiRouter, coreModule)
	}

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	// 启动服务器
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

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	applogger.Info("正在关闭服务器...")

	// 设置关闭超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		applogger.Errorf("服务器强制关闭: %v", err)
	}

	// 清理资源
	coreModule.Close()
	applogger.Close()

	applogger.Info("服务器已退出")
}