package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/agent/server"
	"github.com/sirupsen/logrus"
)

func main() {
	// 加载配置
	configPath := os.Getenv("AGENT_CONFIG_PATH")
	if configPath == "" {
		configPath = "/etc/xingran-agent/config.yaml"
	}

	config, err := server.LoadConfig(configPath)
	if err != nil {
		// 配置加载失败，尝试自动注册
		log.Println("配置加载失败，尝试自动注册...")
		backendURL := os.Getenv("BACKEND_URL")
		if backendURL == "" {
			backendURL = "https://localhost:9000"
		}

		// 创建临时配置用于注册
		tempConfig := &server.Config{
			BackendURL: backendURL,
		}

		// 调用自动注册
		config, err = server.RegisterToBackend(tempConfig)
		if err != nil {
			log.Fatalf("自动注册失败: %v", err)
		}

		log.Printf("自动注册成功: VMID=%s, AgentID=%s", config.VMID, config.AgentID)
	}

	// 初始化结构化日志
	if err := server.InitLogger(config.LogLevel, config.LogPath); err != nil {
		log.Printf("Failed to initialize logger: %v", err)
	}

	// 创建 TLS 配置
	tlsConfig, err := server.NewTLSConfigFromConfig(
		config.TLSCertFile,
		config.TLSKeyFile,
		config.CAFile,
		config.VerifyCertificates,
	)
	if err != nil {
		server.Fatal("Failed to create TLS config:", err)
	}

	// 初始化组件
	authenticator := server.NewJWTAuthenticator(
		config.JWTSecret,
		config.BackendURL,
		config.AgentID,
		config.VMID,
		tlsConfig,
	)
	accountManager := server.NewAccountManager()
	handler := server.NewAgentHandler(accountManager, authenticator)

	// 创建连接管理器
	connManager := server.NewConnectionManager(authenticator)

	// 设置状态变更回调
	connManager.SetStateChangeCallback(func(state server.ConnectionState) {
		server.WithFields(logrus.Fields{
			"state": state.String(),
		}).Info("Connection state changed")
	})

	// 初始连接
	server.Info("Attempting initial connection...")
	if err := connManager.Connect(context.Background()); err != nil {
		server.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Warn("Initial connection failed, will continue with auto-reconnect")
	}

	// 注册 Agent（启动时注册）
	log.Printf("Registering agent %s for VM %s...", config.AgentID, config.VMID)
	if err := authenticator.RegisterToBackend(context.Background(), map[string]interface{}{}); err != nil {
		log.Printf("Warning: Initial registration failed: %v. Will retry with heartbeat...", err)
		// 不致命，继续启动，心跳时会重试
	} else {
		log.Println("Agent registered successfully")
	}

	// 启动 HTTP 服务器
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// 使用中间件
	r.Use(server.RecoveryMiddleware())
	r.Use(server.LoggingMiddleware())
	r.Use(server.CORSMiddleware())

	// 注册路由
	handler.RegisterRoutes(r)

	// 启动健康监控（替代原始心跳）
	healthCtx, cancelHealth := context.WithCancel(context.Background())
	defer cancelHealth()
	go connManager.StartHealthMonitor(healthCtx, config.HeartbeatInterval)

	// 启动 HTTP 服务器
	go func() {
		addr := config.ListenAddr
		// 检查 TLS 证书是否都存在
		hasCert := config.TLSCertFile != ""
		hasKey := config.TLSKeyFile != ""

		if hasCert && hasKey {
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

	server.Info("Shutdown signal received")
	server.Info("Agent shutting down...")
	connManager.Disconnect()
	server.Info("Agent stopped")
}
