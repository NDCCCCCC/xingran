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
	Worker   WorkerConfig   `mapstructure:"worker"`
	Backend  BackendConfig  `mapstructure:"backend"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Browser  BrowserConfig  `mapstructure:"browser"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Monitor  MonitorConfig  `mapstructure:"monitor"`
	Executor ExecutorConfig `mapstructure:"executor"`

	// Warnings Phase 154 / ENGINE-01：配置加载期的兼容性告警（如旧键
	// browser.max_pages 被合并）。mapstructure "-" 表示不从 yaml 读取，
	// 由 Load 填充，Worker 启动时以 WARN 级别输出。
	Warnings []string `mapstructure:"-"`

	// LiveView Phase 155 / LIVEVIEW-01/05：实时画面推流参数（顶层节 worker.liveview）。
	// 归属协议第 1 条：154 拥有 browser 侧的页并发 schema（max_pages→max_concurrency
	// 合并已完成）；155 只新增 screencast 相关键，**不动**任何既有判定。
	LiveView LiveViewConfig `mapstructure:"liveview"`

	// Recording Phase W6 / REC-01：录制会话参数（顶层节 worker.recording）。
	// 归属协议第 4 条：录制类型与配置**均放独立文件/独立节**，不进 types.go
	// 与既有 worker 配置节，避免与 154/155 同文件冲突。
	Recording RecordingConfig `mapstructure:"recording"`
}

// RecordingConfig Phase W6 / REC-01：录制与拾取会话的运行参数。
//
// 单位与语义：
//   - MaxSessions：单 Worker 同时进行的录制会话数上限（默认 2）。
//     与 155 的 max_streams 同款「并发推流上限」配置风格 —— **裁决在 Worker 侧**
//     （真正的名额在 Worker 裁决，后端只展示上限与可读错误文案）。
//     注意：每个录制会话占用一个 browser page 名额，故 MaxSessions 实际受
//     worker.max_concurrency 约束（配置加载期给出可读告警，不静默取小值）。
//   - IdleTimeout：无任何指令（含 keepalive）超过该时限即回收会话
//     （默认 5 分钟）。这是**孤儿 page 的兜底**：后端崩溃 / Redis 断连时
//     会话会悬挂，Worker 侧必须自己把 page 还回去。
type RecordingConfig struct {
	// Enabled 是否允许录制（默认开启）。
	Enabled bool `mapstructure:"enabled"`
	// MaxSessions 并发录制会话上限（默认 2）。
	MaxSessions int `mapstructure:"max_sessions"`
	// IdleTimeout 会话闲置回收时限（默认 5 分钟）。
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`
	// EnabledSet 内部标记（不从 yaml 读）：显式配置了 recording.enabled 时为 true。
	EnabledSet bool `mapstructure:"-"`
}

// Phase W6 / REC-01 默认值。
const (
	// DefaultRecordingMaxSessions 单 Worker 默认并发录制会话上限。
	DefaultRecordingMaxSessions = 2
	// DefaultRecordingIdleTimeout 默认会话闲置回收时限。
	DefaultRecordingIdleTimeout = 5 * time.Minute
)

// WorkerConfig Worker 配置
type WorkerConfig struct {
	ID                      string           `mapstructure:"id"`
	Name                    string           `mapstructure:"name"`
	Version                 string           `mapstructure:"version"`
	MaxConcurrency          int              `mapstructure:"max_concurrency"`
	BrowserPoolSize         int              `mapstructure:"browser_pool_size"`
	BrowserInitSize         int              `mapstructure:"browser_pool_initial"`
	TaskTimeout             time.Duration    `mapstructure:"task_timeout"`
	ActionTimeout           time.Duration    `mapstructure:"action_timeout"`
	ScreenshotsDir          string           `mapstructure:"screenshots_dir"`
	DownloadsDir            string           `mapstructure:"downloads_dir"`
	StartTime               time.Time        `mapstructure:"-"` // Worker启动时间
	RetryCheckInterval      time.Duration    `mapstructure:"retry_check_interval"`
	PauseTimeout            time.Duration    `mapstructure:"pause_timeout"`              // 暂停操作默认超时
	AutoLoginWaitDelay      time.Duration    `mapstructure:"autologin_wait_delay"`       // 自动登录等待延迟
	AutoLoginFillDelay      time.Duration    `mapstructure:"autologin_fill_delay"`       // 自动登录填写延迟
	AutoLoginNavDelay       time.Duration    `mapstructure:"autologin_nav_delay"`        // 自动登录导航延迟
	LoopConcurrency         int              `mapstructure:"loop_concurrency"`           // 循环内并发执行数量（0表示串行）
	ShutdownTimeout         time.Duration    `mapstructure:"shutdown_timeout"`           // 优雅关闭超时
	MaxScaleUpConcurrency   int              `mapstructure:"max_scale_up_concurrency"`   // 最大扩容并发数
	MinScaleDownConcurrency int              `mapstructure:"min_scale_down_concurrency"` // 最小缩容并发数
	HybridMode              HybridModeConfig `mapstructure:"hybrid_mode"`                // 混合模式配置
}

// LiveViewConfig Phase 155：screencast 帧流与接管的运行参数。
//
// 单位与语义：
//   - MaxStreams：单 Worker 同时推流路数上限（LIVEVIEW-01/05，默认 2）。
//     超限必须返回可读错误给前端，不得静默降级。
//   - Quality/MaxWidth/MaxHeight/EveryNthFrame：CDP startScreencast 参数，
//     默认 50/1280/720/2（调研 §1 的降档甜点）。
//   - AckTimeout：单帧等消费确认上限，超时强制 ACK 并计 drop（背压兜底）。
//   - IdleTimeout：无任何下行指令（含 keepalive）超过该时限即停流（默认 30s）。
//   - TakeoverTimeout：接管者无输入超过该时限自动释放（防僵尸占用，默认 120s）。
type LiveViewConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	MaxStreams      int           `mapstructure:"max_streams"`
	Quality         int           `mapstructure:"quality"`
	MaxWidth        int           `mapstructure:"max_width"`
	MaxHeight       int           `mapstructure:"max_height"`
	EveryNthFrame   int           `mapstructure:"every_nth_frame"`
	AckTimeout      time.Duration `mapstructure:"ack_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	TakeoverTimeout time.Duration `mapstructure:"takeover_timeout"`
	// DefaultMode 未显式指定时的协作模式（copilot 为默认，符合 LIVEVIEW-03
	// 的「一般员工信任曲线」）。
	DefaultMode string `mapstructure:"default_mode"`

	// EnabledSet 内部标记（不从 yaml 读）：显式配置了 liveview.enabled 时为 true。
	// 用于区分「未配置」与「显式 false」—— 未配置默认开启，显式 false 保持关闭。
	EnabledSet bool `mapstructure:"-"`
}

// Phase 155 默认值（与 browser/screencast.go 的常量保持一致，避免两处漂移）。
const (
	DefaultLiveViewMaxStreams      = 2
	DefaultLiveViewQuality         = 50
	DefaultLiveViewMaxWidth        = 1280
	DefaultLiveViewMaxHeight       = 720
	DefaultLiveViewEveryNthFrame   = 2
	DefaultLiveViewAckTimeout      = 800 * time.Millisecond
	DefaultLiveViewIdleTimeout     = 30 * time.Second
	DefaultLiveViewTakeoverTimeout = 120 * time.Second
	DefaultLiveViewMode            = "copilot"
)

// BackendConfig 后端 API 配置
type BackendConfig struct {
	BaseURL  string `mapstructure:"base_url"`
	APIToken string `mapstructure:"token"`
	// ProvisioningToken DEBT-03：预共享注册令牌（仅注册时经 X-Provisioning-Token 发送；
	// 注册成功后换发的 Worker Token 由 APIClient 内存保存，无需配置）。
	ProvisioningToken      string        `mapstructure:"provisioning_token"`
	Timeout                time.Duration `mapstructure:"timeout"`
	HeartbeatInterval      time.Duration `mapstructure:"heartbeat_interval"`
	ProgressPublishTimeout time.Duration `mapstructure:"progress_publish_timeout"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr              string        `mapstructure:"addr"`
	Password          string        `mapstructure:"password"`
	DB                int           `mapstructure:"db"`
	StreamGroup       string        `mapstructure:"stream_group"`
	StreamTasks       string        `mapstructure:"stream_tasks"`
	StreamProgress    string        `mapstructure:"stream_progress"`
	BlockTime         time.Duration `mapstructure:"block_time"`
	ConsumerCount     int           `mapstructure:"consumer_count"`
	ConnectionTimeout time.Duration `mapstructure:"connection_timeout"` // 连接测试超时
}

// BrowserConfig browser configuration
type BrowserConfig struct {
	Headless          bool          `mapstructure:"headless"`
	DevTools          bool          `mapstructure:"devtools"`
	SlowMo            time.Duration `mapstructure:"slow_mo"`
	Timeout           time.Duration `mapstructure:"timeout"`
	NavigationTimeout time.Duration `mapstructure:"navigation_timeout"`
	// MaxPages 已废弃（Phase 154 / ENGINE-01）。历史语义是「Chrome 进程数」，
	// 现语义与 worker.max_concurrency 合并为单一「并发页面数」。保留字段仅为
	// 旧键可读 + 旧代码可读，值由 Load 归一化为 EffectiveMaxPages。
	MaxPages int `mapstructure:"max_pages"`
	// MaxConcurrency 是合并后的「并发页面数」浏览器侧只读视图，
	// 永远等于 config.EffectiveMaxPages(cfg)。Phase 157 只读取它。
	MaxConcurrency int      `mapstructure:"max_concurrency"`
	ViewportWidth  int      `mapstructure:"viewport_width"`
	ViewportHeight int      `mapstructure:"viewport_height"`
	UserAgent      string   `mapstructure:"user_agent"`
	ChromePath     string   `mapstructure:"chrome_path"`
	ChromeFlags    []string `mapstructure:"chrome_flags"`
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
	HealthEnabled  bool `mapstructure:"health_enabled"`
	HealthPort     int  `mapstructure:"health_port"`
	MetricsEnabled bool `mapstructure:"metrics_enabled"`
	PprofEnabled   bool `mapstructure:"pprof_enabled"`
	PprofPort      int  `mapstructure:"pprof_port"`
}

// ExecutorConfig 执行器配置
type ExecutorConfig struct {
	MaxRetries    int           `mapstructure:"max_retries"`
	InitialDelay  time.Duration `mapstructure:"retry_delay"`
	MaxDelay      time.Duration `mapstructure:"max_retry_delay"`
	BackoffFactor float64       `mapstructure:"backoff_factor"`
	// UndefinedVariablePolicy Phase 154 / ENGINE-04：变量未定义策略。
	// "strict" = 严格失败并终止该动作；"empty" = 替换为空串（默认，向后兼容）。
	UndefinedVariablePolicy string `mapstructure:"undefined_variable_policy"`
}

// 变量未定义策略取值（ENGINE-04）。
const (
	UndefinedVariableStrict = "strict"
	UndefinedVariableEmpty  = "empty"
)

// EffectiveMaxPages 返回合并后的「并发页面数」（唯一真相）。
// Phase 154 / ENGINE-01：worker.max_concurrency 与 browser.max_pages 合并后
// 仅剩这一个语义。Phase 155/157 必须读取本函数而非直接读任一侧字段。
func EffectiveMaxPages(cfg *Config) int {
	if cfg == nil {
		return 0
	}
	if cfg.Worker.MaxConcurrency > 0 {
		return cfg.Worker.MaxConcurrency
	}
	return cfg.Browser.MaxConcurrency
}

// HybridModeConfig 混合模式配置
type HybridModeConfig struct {
	Enabled           bool          `mapstructure:"enabled"`             // 启用混合模式
	SubTaskTimeout    time.Duration `mapstructure:"subtask_timeout"`     // 子任务超时（默认5分钟）
	SubTaskRetryCount int           `mapstructure:"subtask_retry_count"` // 子任务重试次数（默认2）
	ProgressThrottle  time.Duration `mapstructure:"progress_throttle"`   // 进度上报节流（默认100ms）
	SubtaskPriority   int           `mapstructure:"subtask_priority"`    // 子任务优先级（默认5）
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

	// Phase 154 / ENGINE-01：旧键 browser.max_pages 与 worker.max_concurrency
	// 合并为单一「并发页面数」。必须在 setDefaults 之前判定，否则默认值会
	// 掩盖「旧键独有」的情形。
	mergePageConcurrency(v, &cfg)

	// 设置默认值
	setDefaults(&cfg)

	// 环境变量覆盖（直接读取）
	overrideWithEnv(&cfg)

	// 环境变量可能改写 worker.max_concurrency —— 归一化浏览器侧只读视图。
	syncPageConcurrency(&cfg)

	// Phase 155 / LIVEVIEW：screencast 默认值。需要 viper 判定 liveview.enabled
	// 是否显式配置（未配置=默认开启，显式 false=关闭），故在 Load 内单独调用。
	// 注意：setDefaults 已用 v==nil 路径填过一次默认（把 Enabled 置 true），
	// 因此这里必须先修正 EnabledSet 与 Enabled，否则显式 false 会被默认值覆盖。
	applyLiveViewDefaults(v, &cfg.LiveView)

	// Phase W6 / REC-01：录制会话默认值（口径同 liveview）。
	applyRecordingDefaults(v, &cfg.Recording)

	// Phase 154 / ENGINE-04：未定义变量策略校验（非法值在加载期即失败，
	// 而不是到执行期才静默走错分支）。
	if err := validateUndefinedVariablePolicy(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// setDefaults 设置默认值
func setDefaults(cfg *Config) {
	if cfg.Worker.Name == "" {
		cfg.Worker.Name = "RPA Worker"
	}
	if cfg.Worker.MaxConcurrency == 0 {
		// Phase 154 / ENGINE-01：默认并发页面数。历史默认 3（worker）与
		// 10（browser.max_pages）均不再各自生效；合并后的默认取 5，
		// 与 deployments/docker-compose.yml 的 MAX_CONCURRENCY 默认一致。
		cfg.Worker.MaxConcurrency = 5
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

	// Phase 154 / ENGINE-01：浏览器侧只读视图永远跟随合并后的单一值。
	syncPageConcurrency(cfg)

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
	// Phase 154 / ENGINE-04：默认「空串」策略，保持既有任务脚本不因新增
	// 严格模式而回归失败；需要强校验的任务显式配 strict。
	if cfg.Executor.UndefinedVariablePolicy == "" {
		cfg.Executor.UndefinedVariablePolicy = UndefinedVariableEmpty
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

	// Phase 155 / LIVEVIEW：纯结构体路径的兜底默认（v==nil → 默认开启）。
	applyLiveViewDefaults(nil, &cfg.LiveView)

	// Phase W6 / REC-01：录制配置兜底默认（v==nil → 默认开启）。
	applyRecordingDefaults(nil, &cfg.Recording)
}

// applyRecordingDefaults Phase W6 / REC-01：填录制会话默认值。
//
// 口径与 applyLiveViewDefaults 同款：「未配置」与「显式 false」的区分。
// 录制是本里程碑交付项，默认开启；显式写 enabled: false 才关闭。
func applyRecordingDefaults(v *viper.Viper, rc *RecordingConfig) {
	if rc == nil {
		return
	}
	switch {
	case v == nil:
		// 纯结构体路径（单测直接构造 Config）：默认开启。
		rc.Enabled = true
	case v.IsSet("recording.enabled"), v.IsSet("worker.recording.enabled"):
		// 显式给出 → 以 yaml 为准（含 false）。必须先重置 Enabled，
		// 否则 setDefaults 的 v==nil 兜底路径会把显式 false 覆盖掉。
		rc.EnabledSet = true
		if v.GetBool("recording.enabled") || v.GetBool("worker.recording.enabled") {
			rc.Enabled = true
		} else {
			rc.Enabled = false
		}
	default:
		rc.Enabled = true
	}
	if rc.MaxSessions <= 0 {
		rc.MaxSessions = DefaultRecordingMaxSessions
	}
	if rc.IdleTimeout <= 0 {
		rc.IdleTimeout = DefaultRecordingIdleTimeout
	}
}

// applyLiveViewDefaults Phase 155 / LIVEVIEW：填 screencast 默认值。
//
// 「未配置」与「显式 false」的区分：viper 的 IsSet 能判定键是否存在，因此
// worker.liveview.enabled / liveview.enabled 未出现在 yaml 时默认开启
// （LIVEVIEW 是本里程碑交付项，默认可用）；显式写 enabled: false 才关闭。
//
// v == nil 时（单测直接构造 Config 的路径）一律填默认值并默认开启。
func applyLiveViewDefaults(v *viper.Viper, lv *LiveViewConfig) {
	if lv == nil {
		return
	}
	switch {
	case v == nil:
		// 纯结构体路径（单测直接构造 Config）：默认开启。
		lv.Enabled = true
	case v.IsSet("liveview.enabled"), v.IsSet("worker.liveview.enabled"):
		// 用户显式给出 → 以 yaml 值为准（含 false）。这里必须**重置** Enabled：
		// setDefaults 的 v==nil 兜底路径已把它置 true，若不重置，显式 false 会被
		// 默认值静默覆盖（配置语义失效，已由 TestLiveView_ExplicitDisable 锁定）。
		lv.EnabledSet = true
		if yamlFlag := v.GetBool("liveview.enabled"); yamlFlag {
			lv.Enabled = true
		} else if yamlFlag2 := v.GetBool("worker.liveview.enabled"); yamlFlag2 {
			lv.Enabled = true
		} else {
			lv.Enabled = false
		}
	default:
		lv.Enabled = true
	}
	if lv.MaxStreams <= 0 {
		lv.MaxStreams = DefaultLiveViewMaxStreams
	}
	if lv.Quality <= 0 || lv.Quality > 100 {
		lv.Quality = DefaultLiveViewQuality
	}
	if lv.MaxWidth <= 0 {
		lv.MaxWidth = DefaultLiveViewMaxWidth
	}
	if lv.MaxHeight <= 0 {
		lv.MaxHeight = DefaultLiveViewMaxHeight
	}
	if lv.EveryNthFrame <= 0 {
		lv.EveryNthFrame = DefaultLiveViewEveryNthFrame
	}
	if lv.AckTimeout <= 0 {
		lv.AckTimeout = DefaultLiveViewAckTimeout
	}
	if lv.IdleTimeout <= 0 {
		lv.IdleTimeout = DefaultLiveViewIdleTimeout
	}
	if lv.TakeoverTimeout <= 0 {
		lv.TakeoverTimeout = DefaultLiveViewTakeoverTimeout
	}
	switch lv.DefaultMode {
	case "":
		lv.DefaultMode = DefaultLiveViewMode
	case "auto", "copilot", "manual":
		// 合法取值保持。
	default:
		// 非法模式回落默认（配置期不静默走错分支：Load 输出 WARN）。
		lv.DefaultMode = DefaultLiveViewMode
	}
}

// mergePageConcurrency Phase 154 / ENGINE-01：旧键 browser.max_pages 兼容。
//
// 语义裁定（供 Phase 157 CI 漂移校验读取）：
//  1. 若配置同时给出 worker.max_concurrency 与 browser.max_pages —— 以
//     worker.max_concurrency 为准，browser.max_pages 仅作告警（不覆盖）。
//  2. 若只给 browser.max_pages（旧部署形态）—— 取值迁移到
//     worker.max_concurrency，并告警「旧键已合并，请改配 max_concurrency」。
//  3. 两者都未给 —— 交给 setDefaults 的默认值（5）。
//  4. 合并后 browser.MaxPages 字段保留原值（可读），browser.MaxConcurrency
//     是唯一的派生只读视图（见 syncPageConcurrency）。
func mergePageConcurrency(v *viper.Viper, cfg *Config) {
	hasWorkerKey := v.IsSet("worker.max_concurrency")
	hasLegacyKey := v.IsSet("browser.max_pages")

	switch {
	case hasWorkerKey && hasLegacyKey:
		cfg.Warnings = append(cfg.Warnings, fmt.Sprintf(
			"browser.max_pages=%d 已废弃并合并进 worker.max_concurrency=%d；"+
				"本次以 max_concurrency 为准，请删除旧键",
			cfg.Browser.MaxPages, cfg.Worker.MaxConcurrency))
	case hasLegacyKey && !hasWorkerKey && cfg.Browser.MaxPages > 0:
		cfg.Worker.MaxConcurrency = cfg.Browser.MaxPages
		cfg.Warnings = append(cfg.Warnings, fmt.Sprintf(
			"检测到旧键 browser.max_pages=%d，已合并为 worker.max_concurrency=%d；"+
				"请迁移配置（旧键仍可读但不再有独立语义）",
			cfg.Browser.MaxPages, cfg.Worker.MaxConcurrency))
	}
}

// syncPageConcurrency 把合并值同步到浏览器侧只读视图。
func syncPageConcurrency(cfg *Config) {
	if cfg == nil {
		return
	}
	if cfg.Worker.MaxConcurrency > 0 {
		cfg.Browser.MaxConcurrency = cfg.Worker.MaxConcurrency
	} else if cfg.Browser.MaxConcurrency > 0 {
		cfg.Worker.MaxConcurrency = cfg.Browser.MaxConcurrency
	}
}

// validateUndefinedVariablePolicy Phase 154 / ENGINE-04：策略取值校验。
func validateUndefinedVariablePolicy(cfg *Config) error {
	switch cfg.Executor.UndefinedVariablePolicy {
	case "", UndefinedVariableStrict, UndefinedVariableEmpty:
		return nil
	default:
		return fmt.Errorf(
			"executor.undefined_variable_policy 取值非法: %q（允许 strict|empty）",
			cfg.Executor.UndefinedVariablePolicy)
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
	// DEBT-03：预共享注册令牌 env 注入（生产勿在 yaml 提交明文）。
	if pt := os.Getenv("WORKER_PROVISIONING_TOKEN"); pt != "" {
		cfg.Backend.ProvisioningToken = pt
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
	// Chromium 可执行路径的 env 回退（2026-11-25 生产 UAT 修复）。
	// 容器镜像只设 CHROMIUM_PATH / CHROME_BIN（Dockerfile ENV），从不写
	// config.yaml 的 browser.chrome_path；缺此回退时 cfg.ChromePath 为空，
	// rod launcher 会走默认行为「自动下载 chrome-linux.zip」——国内不可达，
	// 表现为任务永久卡在 Progress 00%。按优先级取第一个非空值。
	if cfg.Browser.ChromePath == "" {
		for _, key := range []string{"CHROMIUM_PATH", "CHROME_BIN", "CHROME_PATH"} {
			if v := strings.TrimSpace(os.Getenv(key)); v != "" {
				cfg.Browser.ChromePath = v
				break
			}
		}
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
