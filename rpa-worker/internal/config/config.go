package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 配置结构
type Config struct {
	Worker    WorkerConfig    `mapstructure:"worker"`
	Backend   BackendConfig   `mapstructure:"backend"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Browser   BrowserConfig   `mapstructure:"browser"`
	Logging   LoggingConfig   `mapstructure:"logging"`
	Monitor   MonitorConfig   `mapstructure:"monitor"`
	Executor  ExecutorConfig  `mapstructure:"executor"`
}

// WorkerConfig Worker 配置
type WorkerConfig struct {
	ID                  string        `mapstructure:"id"`
	Name                string        `mapstructure:"name"`
	Version             string        `mapstructure:"version"`
	MaxConcurrency      int           `mapstructure:"max_concurrency"`
	BrowserPoolSize     int           `mapstructure:"browser_pool_size"`
	BrowserInitSize     int           `mapstructure:"browser_pool_initial"`
	TaskTimeout         time.Duration `mapstructure:"task_timeout"`
	ActionTimeout       time.Duration `mapstructure:"action_timeout"`
	ScreenshotsDir      string        `mapstructure:"screenshots_dir"`
	DownloadsDir        string        `mapstructure:"downloads_dir"`
	StartTime           time.Time     `mapstructure:"-"` // Worker启动时间
	RetryCheckInterval  time.Duration `mapstructure:"retry_check_interval"`
	PauseTimeout        time.Duration `mapstructure:"pause_timeout"`          // 暂停操作默认超时
	AutoLoginWaitDelay  time.Duration `mapstructure:"autologin_wait_delay"`   // 自动登录等待延迟
	AutoLoginFillDelay  time.Duration `mapstructure:"autologin_fill_delay"`   // 自动登录填写延迟
	AutoLoginNavDelay   time.Duration `mapstructure:"autologin_nav_delay"`    // 自动登录导航延迟
	LoopConcurrency     int           `mapstructure:"loop_concurrency"`       // 循环内并发执行数量（0表示串行）
	ShutdownTimeout     time.Duration `mapstructure:"shutdown_timeout"`      // 优雅关闭超时
	MaxScaleUpConcurrency   int        `mapstructure:"max_scale_up_concurrency"`   // 最大扩容并发数
	MinScaleDownConcurrency int        `mapstructure:"min_scale_down_concurrency"` // 最小缩容并发数
	HybridMode          HybridModeConfig `mapstructure:"hybrid_mode"`         // 混合模式配置
}

// BackendConfig 后端 API 配置
type BackendConfig struct {
	BaseURL              string        `mapstructure:"base_url"`
	APIToken              string        `mapstructure:"token"`
	Timeout               time.Duration `mapstructure:"timeout"`
	HeartbeatInterval     time.Duration `mapstructure:"heartbeat_interval"`
	ProgressPublishTimeout time.Duration `mapstructure:"progress_publish_timeout"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr               string        `mapstructure:"addr"`
	Password           string        `mapstructure:"password"`
	DB                 int           `mapstructure:"db"`
	StreamGroup        string        `mapstructure:"stream_group"`
	StreamTasks        string        `mapstructure:"stream_tasks"`
	StreamProgress     string        `mapstructure:"stream_progress"`
	BlockTime          time.Duration `mapstructure:"block_time"`
	ConsumerCount      int           `mapstructure:"consumer_count"`
	ConnectionTimeout  time.Duration `mapstructure:"connection_timeout"` // 连接测试超时
}

// BrowserConfig browser configuration
type BrowserConfig struct {
	Headless        bool          `mapstructure:"headless"`
	DevTools        bool          `mapstructure:"devtools"`
	SlowMo          time.Duration `mapstructure:"slow_mo"`
	Timeout         time.Duration `mapstructure:"timeout"`
	NavigationTimeout time.Duration `mapstructure:"navigation_timeout"`
	MaxPages        int           `mapstructure:"max_pages"`
	ViewportWidth   int           `mapstructure:"viewport_width"`
	ViewportHeight  int           `mapstructure:"viewport_height"`
	UserAgent       string        `mapstructure:"user_agent"`
	ChromePath      string        `mapstructure:"chrome_path"`
	ChromeFlags     []string      `mapstructure:"chrome_flags"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	HealthEnabled bool   `mapstructure:"health_enabled"`
	HealthPort    int    `mapstructure:"health_port"`
	MetricsEnabled bool   `mapstructure:"metrics_enabled"`
	PprofEnabled   bool   `mapstructure:"pprof_enabled"`
	PprofPort      int    `mapstructure:"pprof_port"`
}

// ExecutorConfig 执行器配置
type ExecutorConfig struct {
	MaxRetries    int           `mapstructure:"max_retries"`
	InitialDelay  time.Duration `mapstructure:"retry_delay"`
	MaxDelay      time.Duration `mapstructure:"max_retry_delay"`
	BackoffFactor float64       `mapstructure:"backoff_factor"`
}

// HybridModeConfig 混合模式配置
type HybridModeConfig struct {
	Enabled            bool          `mapstructure:"enabled"`              // 启用混合模式
	SubTaskTimeout     time.Duration `mapstructure:"subtask_timeout"`       // 子任务超时（默认5分钟）
	SubTaskRetryCount  int           `mapstructure:"subtask_retry_count"`  // 子任务重试次数（默认2）
	ProgressThrottle   time.Duration `mapstructure:"progress_throttle"`     // 进度上报节流（默认100ms）
	SubtaskPriority    int           `mapstructure:"subtask_priority"`      // 子任务优先级（默认5）
}

// Load load configuration
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file failed: %w", err)
	}

	// 环境变量前缀
	v.SetEnvPrefix("WORKER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 解析配置
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 设置默认值
	setDefaults(&cfg)

	// 环境变量覆盖（直接读取）
	overrideWithEnv(&cfg)

	return &cfg, nil
}

// setDefaults 设置默认值
func setDefaults(cfg *Config) {
	if cfg.Worker.Name == "" {
		cfg.Worker.Name = "RPA Worker"
	}
	if cfg.Worker.MaxConcurrency == 0 {
		cfg.Worker.MaxConcurrency = 3
	}
	if cfg.Worker.BrowserPoolSize == 0 {
		cfg.Worker.BrowserPoolSize = 5
	}
	if cfg.Worker.BrowserInitSize == 0 {
		cfg.Worker.BrowserInitSize = 2
	}
	if cfg.Worker.TaskTimeout == 0 {
		cfg.Worker.TaskTimeout = 5 * time.Minute
	}
	if cfg.Worker.ActionTimeout == 0 {
		cfg.Worker.ActionTimeout = 30 * time.Second
	}
	if cfg.Worker.ScreenshotsDir == "" {
		cfg.Worker.ScreenshotsDir = "./screenshots"
	}
	if cfg.Worker.DownloadsDir == "" {
		cfg.Worker.DownloadsDir = "./downloads"
	}
	if cfg.Worker.RetryCheckInterval == 0 {
		cfg.Worker.RetryCheckInterval = 100 * time.Millisecond
	}
	if cfg.Worker.PauseTimeout == 0 {
		cfg.Worker.PauseTimeout = 300 * time.Second // 5 分钟默认暂停超时
	}
	if cfg.Worker.AutoLoginWaitDelay == 0 {
		cfg.Worker.AutoLoginWaitDelay = 1 * time.Second
	}
	if cfg.Worker.AutoLoginFillDelay == 0 {
		cfg.Worker.AutoLoginFillDelay = 500 * time.Millisecond
	}
	if cfg.Worker.AutoLoginNavDelay == 0 {
		cfg.Worker.AutoLoginNavDelay = 2 * time.Second
	}
	// 默认循环并发数为3，表示循环内最多3个迭代同时执行
	// 设置为0则表示串行执行（原有行为）
	if cfg.Worker.LoopConcurrency == 0 {
		cfg.Worker.LoopConcurrency = 3
	}

	// 扩缩容配置默认值
	if cfg.Worker.ShutdownTimeout == 0 {
		cfg.Worker.ShutdownTimeout = 5 * time.Second
	}
	if cfg.Worker.MaxScaleUpConcurrency == 0 {
		cfg.Worker.MaxScaleUpConcurrency = 50
	}
	if cfg.Worker.MinScaleDownConcurrency == 0 {
		cfg.Worker.MinScaleDownConcurrency = 1
	}

	if cfg.Backend.BaseURL == "" {
		cfg.Backend.BaseURL = "http://localhost:9000/api/v1"
	}
	if cfg.Backend.Timeout == 0 {
		cfg.Backend.Timeout = 30 * time.Second
	}
	if cfg.Backend.HeartbeatInterval == 0 {
		cfg.Backend.HeartbeatInterval = 30 * time.Second
	}
	if cfg.Backend.ProgressPublishTimeout == 0 {
		cfg.Backend.ProgressPublishTimeout = 5 * time.Second
	}

	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "localhost:6379"
	}
	if cfg.Redis.StreamGroup == "" {
		cfg.Redis.StreamGroup = "rpa-workers"
	}
	if cfg.Redis.StreamTasks == "" {
		cfg.Redis.StreamTasks = "rpa:task:pending"
	}
	if cfg.Redis.BlockTime == 0 {
		cfg.Redis.BlockTime = 5 * time.Second
	}
	if cfg.Redis.ConnectionTimeout == 0 {
		cfg.Redis.ConnectionTimeout = 5 * time.Second
	}

	if cfg.Browser.Timeout == 0 {
		cfg.Browser.Timeout = 30 * time.Second
	}
	if cfg.Browser.NavigationTimeout == 0 {
		cfg.Browser.NavigationTimeout = 60 * time.Second
	}
	if cfg.Browser.ViewportWidth == 0 {
		cfg.Browser.ViewportWidth = 1920
	}
	if cfg.Browser.ViewportHeight == 0 {
		cfg.Browser.ViewportHeight = 1080
	}

	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "stdout"
	}
	if cfg.Logging.MaxSize == 0 {
		cfg.Logging.MaxSize = 100
	}
	if cfg.Logging.MaxBackups == 0 {
		cfg.Logging.MaxBackups = 10
	}
	if cfg.Logging.MaxAge == 0 {
		cfg.Logging.MaxAge = 30
	}

	if cfg.Monitor.HealthPort == 0 {
		cfg.Monitor.HealthPort = 8080
	}
	if cfg.Monitor.PprofPort == 0 {
		cfg.Monitor.PprofPort = 6060
	}

	if cfg.Executor.MaxRetries == 0 {
		cfg.Executor.MaxRetries = 3
	}
	if cfg.Executor.InitialDelay == 0 {
		cfg.Executor.InitialDelay = 1 * time.Second
	}
	if cfg.Executor.MaxDelay == 0 {
		cfg.Executor.MaxDelay = 10 * time.Second
	}
	if cfg.Executor.BackoffFactor == 0 {
		cfg.Executor.BackoffFactor = 2.0
	}

	// 混合模式默认配置
	if cfg.Worker.HybridMode.Enabled {
		if cfg.Worker.HybridMode.SubTaskTimeout == 0 {
			cfg.Worker.HybridMode.SubTaskTimeout = 5 * time.Minute
		}
		if cfg.Worker.HybridMode.SubTaskRetryCount == 0 {
			cfg.Worker.HybridMode.SubTaskRetryCount = 2
		}
		if cfg.Worker.HybridMode.ProgressThrottle == 0 {
			cfg.Worker.HybridMode.ProgressThrottle = 100 * time.Millisecond
		}
		if cfg.Worker.HybridMode.SubtaskPriority == 0 {
			cfg.Worker.HybridMode.SubtaskPriority = 5
		}
	}
}

// overrideWithEnv 环境变量覆盖
func overrideWithEnv(cfg *Config) {
	if id := os.Getenv("WORKER_ID"); id != "" {
		cfg.Worker.ID = id
	}
	if name := os.Getenv("WORKER_NAME"); name != "" {
		cfg.Worker.Name = name
	}
	if token := os.Getenv("WORKER_TOKEN"); token != "" {
		cfg.Backend.APIToken = token
	}
	if backendURL := os.Getenv("BACKEND_URL"); backendURL != "" {
		cfg.Backend.BaseURL = backendURL
	}
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		cfg.Redis.Addr = redisAddr
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		cfg.Redis.Password = redisPassword
	}
	if maxConcurrency := os.Getenv("MAX_CONCURRENCY"); maxConcurrency != "" {
		fmt.Sscanf(maxConcurrency, "%d", &cfg.Worker.MaxConcurrency)
	}
	if headless := os.Getenv("HEADLESS"); headless != "" {
		cfg.Browser.Headless = strings.ToLower(headless) == "true"
	}
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.Logging.Level = logLevel
	}
	if hybridEnabled := os.Getenv("HYBRID_MODE_ENABLED"); hybridEnabled != "" {
		cfg.Worker.HybridMode.Enabled = strings.ToLower(hybridEnabled) == "true"
	}
	if subtaskTimeout := os.Getenv("SUBTASK_TIMEOUT"); subtaskTimeout != "" {
		if duration, err := time.ParseDuration(subtaskTimeout); err == nil {
			cfg.Worker.HybridMode.SubTaskTimeout = duration
		}
	}
}
