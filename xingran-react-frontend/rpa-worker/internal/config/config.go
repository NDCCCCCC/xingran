package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 配置结构
type Config struct {
	Worker   WorkerConfig
	Backend  BackendConfig
	Redis    RedisConfig
	Browser  BrowserConfig
	Logging  LoggingConfig
	Monitor  MonitorConfig
	Executor ExecutorConfig
}

// WorkerConfig Worker 配置
type WorkerConfig struct {
	ID             string        `mapstructure:"id"`
	Name           string        `mapstructure:"name"`
	MaxConcurrency int           `mapstructure:"max_concurrency"`
	Version        string        `mapstructure:"version"`
	Heartbeat      time.Duration `mapstructure:"heartbeat"`
	TaskTimeout    time.Duration `mapstructure:"task_timeout"`
}

// BackendConfig 后端 API 配置
type BackendConfig struct {
	BaseURL            string        `mapstructure:"base_url"`
	APIToken           string        `mapstructure:"api_token"`
	Timeout            time.Duration `mapstructure:"timeout"`
	HeartbeatInterval  time.Duration `mapstructure:"heartbeat_interval"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr         string `mapstructure:"addr"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	StreamTasks  string `mapstructure:"stream_tasks"`
	StreamGroup  string `mapstructure:"stream_group"`
	PendingTasks string `mapstructure:"pending_tasks"`
}

// BrowserConfig 浏览器配置
type BrowserConfig struct {
	Headless       bool          `mapstructure:"headless"`
	Timeout        time.Duration `mapstructure:"timeout"`
	MaxPages       int           `mapstructure:"max_pages"`
	ViewportWidth  int           `mapstructure:"viewport_width"`
	ViewportHeight int           `mapstructure:"viewport_height"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	HealthEnabled bool   `mapstructure:"health_enabled"`
	HealthPort    int    `mapstructure:"health_port"`
	MetricsPath   string `mapstructure:"metrics_path"`
}

// ExecutorConfig 执行器配置
type ExecutorConfig struct {
	MaxRetries      int           `mapstructure:"max_retries"`
	InitialDelay    time.Duration `mapstructure:"initial_delay"`
	RetryMultiplier float64       `mapstructure:"retry_multiplier"`
	ScreenshotOnErr bool          `mapstructure:"screenshot_on_error"`
}

// Load 加载配置
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// 设置默认值
	setDefaults(v)

	// 环境变量支持
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return &cfg, nil
}

// setDefaults 设置默认值
func setDefaults(v *viper.Viper) {
	// Worker
	v.SetDefault("worker.max_concurrency", 5)
	v.SetDefault("worker.version", "1.0.0")
	v.SetDefault("worker.heartbeat", 30*time.Second)
	v.SetDefault("worker.task_timeout", 10*time.Minute)

	// Backend
	v.SetDefault("backend.base_url", "http://localhost:9000/api/v1")
	v.SetDefault("backend.timeout", 30*time.Second)
	v.SetDefault("backend.heartbeat_interval", 30*time.Second)

	// Redis
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.stream_tasks", "rpa:tasks")
	v.SetDefault("redis.stream_group", "rpa:workers")
	v.SetDefault("redis.pending_tasks", "rpa:pending")

	// Browser
	v.SetDefault("browser.headless", true)
	v.SetDefault("browser.timeout", 30*time.Second)
	v.SetDefault("browser.max_pages", 10)
	v.SetDefault("browser.viewport_width", 1920)
	v.SetDefault("browser.viewport_height", 1080)

	// Logging
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "stdout")

	// Monitor
	v.SetDefault("monitor.health_enabled", true)
	v.SetDefault("monitor.health_port", 8080)
	v.SetDefault("monitor.metrics_path", "/metrics")

	// Executor
	v.SetDefault("executor.max_retries", 3)
	v.SetDefault("executor.initial_delay", 1*time.Second)
	v.SetDefault("executor.retry_multiplier", 2.0)
	v.SetDefault("executor.screenshot_on_error", true)
}
