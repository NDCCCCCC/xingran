package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/api"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

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

	// 初始化核心模块
	coreModule := core.New(cfg)
	if err := coreModule.Init(); err != nil {
		log.Fatalf("初始化核心模块失败: %v", err)
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
		log.Printf("服务器启动在端口: %d", cfg.Server.Port)
		log.Printf("环境: %s", cfg.Server.Mode)
		if cfg.Server.Mode != "release" {
			log.Printf("Swagger 文档: http://localhost:%d/swagger/index.html", cfg.Server.Port)
		}
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务器...")

	// 设置关闭超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("服务器强制关闭:", err)
	}

	// 清理资源
	coreModule.Close()

	log.Println("服务器已退出")
}