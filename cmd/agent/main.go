package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/agent/server"
)

func main() {
	// 加载配置
	configPath := os.Getenv("AGENT_CONFIG_PATH")
	if configPath == "" {
		configPath = "/etc/ruoyi-agent/config.yaml"
	}

	config, err := server.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化组件
	authenticator := server.NewJWTAuthenticator(
		config.JWTSecret,
		config.BackendURL,
		config.AgentID,
		config.VMID,
	)
	accountManager := server.NewAccountManager()
	handler := server.NewAgentHandler(accountManager, authenticator)

	// 注册 Agent
	log.Printf("Registering agent %s for VM %s...", config.AgentID, config.VMID)
	if err := authenticator.Register(context.Background()); err != nil {
		log.Fatalf("Failed to register agent: %v", err)
	}
	log.Println("Agent registered successfully")

	// 启动 HTTP 服务器
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// 使用中间件
	r.Use(server.RecoveryMiddleware())
	r.Use(server.LoggingMiddleware())
	r.Use(server.CORSMiddleware())

	// 注册路由
	handler.RegisterRoutes(r)

	// 启动心跳服务
	heartbeatCtx, cancelHeartbeat := context.WithCancel(context.Background())
	defer cancelHeartbeat()
	go startHeartbeat(heartbeatCtx, authenticator, config.HeartbeatInterval)

	// 启动 HTTP 服务器
	go func() {
		addr := config.ListenAddr
		if config.TLSCertFile != "" && config.TLSKeyFile != "" {
			log.Printf("Starting HTTPS server on %s", addr)
			if err := r.RunTLS(addr, config.TLSCertFile, config.TLSKeyFile); err != nil {
				log.Fatalf("Failed to start HTTPS server: %v", err)
			}
		} else {
			log.Printf("Starting HTTP server on %s", addr)
			if err := r.Run(addr); err != nil {
				log.Fatalf("Failed to start HTTP server: %v", err)
			}
		}
	}()

	// 等待信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Agent shutting down...")
}

// startHeartbeat 启动心跳服务
func startHeartbeat(ctx context.Context, authenticator *server.JWTAuthenticator, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Heartbeat service stopped")
			return
		case <-ticker.C:
			if err := authenticator.SendHeartbeat(context.Background()); err != nil {
				log.Printf("Heartbeat failed: %v", err)
			} else {
				log.Println("Heartbeat sent successfully")
			}
		}
	}
}
