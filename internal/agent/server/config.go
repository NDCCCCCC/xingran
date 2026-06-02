package server

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config Agent 服务配置
type Config struct {
	// 后端服务器配置
	BackendURL string `yaml:"backend_url" env:"BACKEND_URL"`
	AgentID    string `yaml:"agent_id" env:"AGENT_ID"`
	VMID       string `yaml:"vm_id" env:"VM_ID"`

	// JWT 配置
	JWTSecret string        `yaml:"jwt_secret" env:"JWT_SECRET"`
	TokenExpiry time.Duration `yaml:"token_expiry"`

	// HTTP 服务器配置
	ListenAddr  string `yaml:"listen_addr"`
	TLSCertFile string `yaml:"tls_cert_file"`
	TLSKeyFile  string `yaml:"tls_key_file"`

	// 心跳配置
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`

	// 日志配置
	LogLevel string `yaml:"log_level"`
	LogPath  string `yaml:"log_path"`

	// 平台配置
	Platform string `yaml:"platform"` // windows or linux
}

// LoadConfig 从 YAML 文件或环境变量加载配置
func LoadConfig(path string) (*Config, error) {
	// 设置默认值
	config := &Config{
		BackendURL:         "https://localhost:9000",
		AgentID:            "",
		VMID:               "",
		JWTSecret:          "",
		TokenExpiry:        24 * time.Hour,
		ListenAddr:         ":8443",
		TLSCertFile:        "",
		TLSKeyFile:         "",
		HeartbeatInterval:  30 * time.Second,
		LogLevel:           "info",
		LogPath:            "/var/log/ruoyi-agent",
		Platform:           "linux",
	}

	// 如果指定了配置文件，尝试加载
	if path != "" {
		// 检查文件是否存在
		if _, err := os.Stat(path); err == nil {
			viper.SetConfigFile(path)
			viper.SetConfigType("yaml")

			if err := viper.ReadInConfig(); err != nil {
				return nil, err
			}

			if err := viper.Unmarshal(config); err != nil {
				return nil, err
			}
		}
	}

	// 从环境变量覆盖配置
	viper.SetEnvPrefix("AGENT")
	viper.AutomaticEnv()

	if backendURL := os.Getenv("BACKEND_URL"); backendURL != "" {
		config.BackendURL = backendURL
	}
	if agentID := os.Getenv("AGENT_ID"); agentID != "" {
		config.AgentID = agentID
	}
	if vmID := os.Getenv("VM_ID"); vmID != "" {
		config.VMID = vmID
	}
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		config.JWTSecret = jwtSecret
	}

	// 确保日志路径是绝对路径
	if config.LogPath != "" && !filepath.IsAbs(config.LogPath) {
		absPath, err := filepath.Abs(config.LogPath)
		if err == nil {
			config.LogPath = absPath
		}
	}

	// 验证必需的配置
	if config.BackendURL == "" {
		return nil, os.ErrInvalid
	}
	if config.AgentID == "" {
		return nil, os.ErrInvalid
	}
	if config.VMID == "" {
		return nil, os.ErrInvalid
	}

	return config, nil
}
