package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Cache    CacheConfig    `mapstructure:"cache"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Security SecurityConfig `mapstructure:"security"`
	Log      LogConfig      `mapstructure:"log"`
	Baidu    BaiduConfig    `mapstructure:"baidu"`
	RPA      RPAConfig      `mapstructure:"rpa"`
	VDI      VDIConfig      `mapstructure:"vdi"`
	AD       ADTLSConfig    `mapstructure:"ad"`
	OperLog  OperLogConfig  `mapstructure:"operlog"`
}

// VDIConfig VDI 客户端配置
type VDIConfig struct {
	// TLSSkipVerify 是否跳过 VDI 服务器 TLS 证书校验。
	// 默认 true 保持向后兼容（VDI 服务器通常使用自签名证书）。
	// 生产环境应在配置文件中设为 false,启用严格证书校验。
	TLSSkipVerify bool `mapstructure:"tls_skip_verify"`
}

// ADTLSConfig AD/LDAP TLS 配置
type ADTLSConfig struct {
	// TLSSkipVerify 是否跳过 AD/LDAP 服务器 TLS 证书校验。
	// 默认 true 保持向后兼容（AD 域控在内网常使用自签名证书）。
	// 生产环境应在配置文件中设为 false,启用严格证书校验。
	TLSSkipVerify bool `mapstructure:"tls_skip_verify"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Name string `mapstructure:"name"`
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`

	// SkipSetup 跳过一次性 setup 步骤(InitData / 默认角色菜单 / cmd seed)。
	//
	// 设计目的: 让"启动"与"初始化"分离。当前每次启动都跑 InitData(10+ 张表 count 查询)、
	// 默认角色菜单 permission、cmd 末尾的 MAC history retention + OUI 导入。
	// 这些是首次部署/恢复场景才需要的,在 DB 已稳定的环境每次跑是纯浪费。
	//
	// 用法:
	//   - 配置文件: server.skip_setup: true
	//   - 环境变量: SERVER_SKIP_SETUP=true (裸名,无需 XINGRAN_ 前缀)
	//   - 配套 viper AutomaticEnv 自动绑定: XINGRAN_SERVER_SKIP_SETUP=true
	//
	// 默认 false: dev 环境默认仍跑(零破坏性,符合现有 GSD 约定的"零破坏性变更")。
	// 生产/CI 部署: 显式置 true 跳过,只跑 setup 阶段需要的可单独 `xingran-backend setup` 命令(未来)。
	SkipSetup bool `mapstructure:"skip_setup"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	SSLMode      string `mapstructure:"sslmode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxLifetime  int    `mapstructure:"max_lifetime"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Type          string `mapstructure:"type"` // redis, memory
	Host          string `mapstructure:"host"`
	Port          int    `mapstructure:"port"`
	Password      string `mapstructure:"password"`
	DB            int    `mapstructure:"db"`
	PoolSize      int    `mapstructure:"pool_size"`
	MaxSize       int    `mapstructure:"max_size"`
	CleanupTime   int    `mapstructure:"cleanup_time"`
	WarmUpEnabled bool   `mapstructure:"warm_up_enabled"` // 是否启用缓存预热
	// L2写入Worker配置
	L2Writer *cache.L2WriterConfig `mapstructure:"l2_writer"` // L2写入Worker Pool配置
	// 重试配置
	RetryEnabled       bool    `mapstructure:"retry_enabled"`        // 是否启用L2缓存写入重试
	RetryMaxRetries    int     `mapstructure:"retry_max_retries"`    // 最大重试次数
	RetryInitialDelay  int     `mapstructure:"retry_initial_delay"`  // 初始延迟(毫秒)
	RetryMaxDelay      int     `mapstructure:"retry_max_delay"`      // 最大延迟(毫秒)
	RetryBackoffFactor float64 `mapstructure:"retry_backoff_factor"` // 退避因子
	RetryWorkerCount   int     `mapstructure:"retry_worker_count"`   // 重试工作协程数
}

// JWTConfig JWT配置
type JWTConfig struct {
	SecretKey        string `mapstructure:"secret_key"`
	AccessKeyExpire  int    `mapstructure:"access_key_expire"`
	RefreshKeyExpire int    `mapstructure:"refresh_key_expire"`
	Issuer           string `mapstructure:"issuer"`
	UseSM2           bool   `mapstructure:"use_sm2"`         // 是否使用SM2算法
	SM2PrivateKey    string `mapstructure:"sm2_private_key"` // SM2私钥(十六进制)
	SM2PublicKey     string `mapstructure:"sm2_public_key"`  // SM2公钥(十六进制)
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`       // 日志级别: debug, info, warn, error
	LogDir     string `mapstructure:"log_dir"`     // 日志目录
	MaxSize    int    `mapstructure:"max_size"`    // 单个日志文件最大大小(MB)
	MaxBackups int    `mapstructure:"max_backups"` // 保留的旧日志文件最大数量
	MaxAge     int    `mapstructure:"max_age"`     // 保留旧日志文件的最大天数
	Compress   bool   `mapstructure:"compress"`    // 是否压缩旧日志文件
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	SM4Key            string                  `mapstructure:"sm4_key"`            // SM4 加密密钥（16字节，Base64编码），用于所有敏感数据加密
	RequestEncryption RequestEncryptionConfig `mapstructure:"request_encryption"` // 请求体加密配置
	ReplayWindowSec   int                     `mapstructure:"replay_window_sec"`  // 加密请求时间戳容差(秒),±N。0=使用默认60
}

// RequestEncryptionConfig 请求体加密配置
type RequestEncryptionConfig struct {
	Enabled           bool     `mapstructure:"enabled"`            // 是否启用
	ExcludePaths      []string `mapstructure:"exclude_paths"`      // 排除路径
	RequireEncryption bool     `mapstructure:"require_encryption"` // 是否强制加密
}

// OperLogConfig 操作日志配置
type OperLogConfig struct {
	// ExcludePaths 排除写入 sys_oper_log 的请求路径。
	// 匹配语义: filepath.Match + /* 后缀通配,与 security.request_encryption.exclude_paths 完全一致。
	// 默认包含 /api/v1/rpa/workers/*/heartbeat（30s/Worker 高频心跳写入会淹没日志表）。
	ExcludePaths []string `mapstructure:"exclude_paths"`
}

// BaiduConfig 百度地图配置
type BaiduConfig struct {
	MapAK string `mapstructure:"map_ak"` // 百度地图API Key
}

// RPAConfig RPA配置
type RPAConfig struct {
	Enabled bool             `mapstructure:"enabled"` // 是否启用RPA功能
	AI      RPAAIConfig      `mapstructure:"ai"`      // AI配置
	Worker  RPAWorkerConfig  `mapstructure:"worker"`  // Worker配置
	Scaling RPAScalingConfig `mapstructure:"scaling"` // 扩缩容配置
	Storage RPAStorageConfig `mapstructure:"storage"` // 存储配置
}

// RPAAIConfig RPA AI配置
type RPAAIConfig struct {
	Generator RPAAIGeneratorConfig `mapstructure:"generator"` // 脚本生成模型配置
	Agent     RPAAIAgentConfig     `mapstructure:"agent"`     // AI Agent模型配置
	Fallback  RPAAIFallbackConfig  `mapstructure:"fallback"`  // 降级策略配置
}

// RPAAIGeneratorConfig 脚本生成模型配置
type RPAAIGeneratorConfig struct {
	Enabled   bool   `mapstructure:"enabled"`    // 是否启用
	APIKey    string `mapstructure:"api_key"`    // API密钥
	BaseURL   string `mapstructure:"base_url"`   // API基础URL（OpenAI兼容）
	Model     string `mapstructure:"model"`      // 模型名称
	MaxTokens int    `mapstructure:"max_tokens"` // 最大token数
}

// RPAAIAgentConfig AI Agent模型配置
type RPAAIAgentConfig struct {
	Enabled   bool   `mapstructure:"enabled"`    // 是否启用
	APIKey    string `mapstructure:"api_key"`    // API密钥
	BaseURL   string `mapstructure:"base_url"`   // API基础URL（OpenAI兼容）
	Model     string `mapstructure:"model"`      // 模型名称
	MaxTokens int    `mapstructure:"max_tokens"` // 最大token数
}

// RPAAIFallbackConfig AI降级策略配置
type RPAAIFallbackConfig struct {
	MaxRetries  int    `mapstructure:"max_retries"`  // 最大重试次数
	EnableAgent bool   `mapstructure:"enable_agent"` // 是否启用AI Agent降级
	Timeout     string `mapstructure:"timeout"`      // 请求超时时间
}

// RPAWorkerConfig RPA Worker配置
type RPAWorkerConfig struct {
	MinWorkers        int    `mapstructure:"min_workers"`        // 最小Worker数量
	MaxWorkers        int    `mapstructure:"max_workers"`        // 最大Worker数量
	HeartbeatInterval string `mapstructure:"heartbeat_interval"` // 心跳间隔
	TaskTimeout       string `mapstructure:"task_timeout"`       // 任务超时时间
}

// RPAStorageConfig RPA存储配置
type RPAStorageConfig struct {
	DownloadsDir    string `mapstructure:"downloads_dir"`     // 下载文件目录
	ScreenshotsDir  string `mapstructure:"screenshots_dir"`   // 截图目录
	MaxRetainedDays int    `mapstructure:"max_retained_days"` // 最大保留天数
}

// RPAScalingConfig RPA扩缩容配置
type RPAScalingConfig struct {
	Enabled           bool         `mapstructure:"enabled"`             // 是否启用自动扩缩容
	CheckInterval     string       `mapstructure:"check_interval"`      // 监控检查间隔
	MinWorkers        int          `mapstructure:"min_workers"`         // 最小 Worker 数量
	MaxWorkers        int          `mapstructure:"max_workers"`         // 最大 Worker 数量
	ScaleUpThreshold  float64      `mapstructure:"scale_up_threshold"`  // 扩容阈值
	ScaleDownCooldown string       `mapstructure:"scale_down_cooldown"` // 缩容冷却时间
	ScaleUpLimit      int          `mapstructure:"scale_up_limit"`      // 单次扩容上限
	EnableMockDocker  bool         `mapstructure:"enable_mock_docker"`  // 是否使用模拟 Docker 客户端
	Docker            DockerConfig `mapstructure:"docker"`              // Docker 配置
}

// DockerConfig Docker 配置
type DockerConfig struct {
	DockerHost    string `mapstructure:"docker_host"`    // Docker 主机地址
	DockerPort    int    `mapstructure:"docker_port"`    // Docker API 端口
	ContainerName string `mapstructure:"container_name"` // 容器名称前缀
	ImageName     string `mapstructure:"image_name"`     // 镜像名称
	NetworkName   string `mapstructure:"network_name"`   // 网络名称
}

// Load 加载配置
func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			applogger.Infof("配置文件未找到，使用默认配置")
		} else {
			applogger.Errorf("读取配置文件失败: %v", err)
		}
	}

	// 设置环境变量前缀
	viper.SetEnvPrefix("XINGRAN")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 环境变量覆盖
	viper.AutomaticEnv()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		applogger.Errorf("解析配置失败: %v", err)
	}

	// 从环境变量直接读取敏感配置(裸变量名,优先级高于 viper 自动绑定)
	overrideFromEnv(&config.Database.Host, "DB_HOST")
	overrideFromEnv(&config.Database.User, "DB_USER")
	overrideFromEnv(&config.Database.Password, "DB_PASSWORD")
	overrideFromEnv(&config.Database.DBName, "DB_NAME")
	overrideFromEnv(&config.Database.SSLMode, "DB_SSLMODE")
	overrideFromEnv(&config.Cache.Host, "REDIS_HOST")
	overrideFromEnv(&config.Cache.Password, "REDIS_PASSWORD")
	// Cache.Port / Cache.DB / Server.* 用裸名,免去 .env 写 XINGRAN_ 前缀
	overrideFromEnvInt(&config.Cache.Port, "REDIS_PORT")
	overrideFromEnvInt(&config.Cache.DB, "REDIS_DB")
	overrideFromEnv(&config.Server.Host, "SERVER_HOST")
	overrideFromEnv(&config.Server.Mode, "SERVER_MODE")
	overrideFromEnvInt(&config.Server.Port, "SERVER_PORT")
	overrideFromEnvBool(&config.Server.SkipSetup, "SERVER_SKIP_SETUP")
	overrideFromEnv(&config.JWT.SecretKey, "JWT_SECRET")
	// SM2 密钥对 — 必须通过 env 注入；留空会让 jwt.go 走"动态生成"分支,
	// 导致每次重启 SM2 密钥对都换,旧 token 全部失效,所有用户被强制登出
	overrideFromEnv(&config.JWT.SM2PrivateKey, "JWT_SM2_PRIVATE_KEY")
	overrideFromEnv(&config.JWT.SM2PublicKey, "JWT_SM2_PUBLIC_KEY")
	overrideFromEnv(&config.Security.SM4Key, "SM4_KEY")
	overrideFromEnv(&config.Baidu.MapAK, "BAIDU_MAP_AK")
	// RPA AI 配置
	overrideFromEnv(&config.RPA.AI.Generator.APIKey, "RPA_AI_GENERATOR_KEY")
	overrideFromEnv(&config.RPA.AI.Generator.BaseURL, "RPA_AI_GENERATOR_URL")
	overrideFromEnv(&config.RPA.AI.Agent.APIKey, "RPA_AI_AGENT_KEY")
	overrideFromEnv(&config.RPA.AI.Agent.BaseURL, "RPA_AI_AGENT_URL")

	return &config
}

// overrideFromEnv 从环境变量覆盖配置值（辅助函数）
func overrideFromEnv(target *string, envKey string) {
	if value := os.Getenv(envKey); value != "" {
		*target = value
	}
}

// overrideFromEnvInt 从环境变量覆盖整型配置值（辅助函数）
// env 未设置或非数字字符串时,保持 target 原值不变
func overrideFromEnvInt(target *int, envKey string) {
	if value := os.Getenv(envKey); value != "" {
		if p, err := strconv.Atoi(value); err == nil {
			*target = p
		}
	}
}

// overrideFromEnvBool 从环境变量覆盖布尔型配置值（辅助函数）
// env 未设置或非 true/false 字符串时,保持 target 原值不变（不覆盖）
// 支持: true/1/yes (true) / false/0/no (false),其余值忽略
func overrideFromEnvBool(target *bool, envKey string) {
	value := os.Getenv(envKey)
	if value == "" {
		return
	}
	switch strings.ToLower(value) {
	case "true", "1", "yes":
		*target = true
	case "false", "0", "no":
		*target = false
	default:
		// 非标准布尔值,保持 target 原值不变
	}
}

// setDefaults 设置默认配置
func setDefaults() {
	// 服务器默认配置
	viper.SetDefault("server.name", "Xingran-Next")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")

	// 数据库默认配置
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.dbname", "xingran_next")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_open_conns", 100)
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.max_lifetime", 3600)

	// 缓存默认配置
	viper.SetDefault("cache.type", "memory")
	viper.SetDefault("cache.host", "localhost")
	viper.SetDefault("cache.port", 6379)
	viper.SetDefault("cache.password", "")
	viper.SetDefault("cache.db", 0)
	viper.SetDefault("cache.pool_size", 10)
	viper.SetDefault("cache.max_size", 1000)
	viper.SetDefault("cache.cleanup_time", 600)

	// JWT默认配置
	// F-04: 不再设置硬编码默认密钥 "xingran-next-secret-key" —
	// 该默认值若被部署进生产,任何人都能用公开已知字符串伪造 JWT。
	// 启动时由 NewJWTManager 强制校验:必须从 env 或 config 注入非空且非弱默认值。
	viper.SetDefault("jwt.secret_key", "")
	viper.SetDefault("jwt.access_key_expire", 7200)
	viper.SetDefault("jwt.refresh_key_expire", 604800)
	viper.SetDefault("jwt.issuer", "Xingran-Next")

	// 日志默认配置
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.log_dir", "logs")
	viper.SetDefault("log.max_size", 100)   // 100MB
	viper.SetDefault("log.max_backups", 30) // 保留30个备份
	viper.SetDefault("log.max_age", 90)     // 保留90天
	viper.SetDefault("log.compress", true)  // 压缩旧日志

	// 安全默认配置（注意：生产环境必须从环境变量设置）
	// SM4 密钥必须是 16 字节（Base64 编码）
	// 默认值 "dGVzdC1zZWNyZXQxNiEhIQ==" 解码后是 "test-secret16!!!" （16字节）
	viper.SetDefault("security.sm4_key", "dGVzdC1zZWNyZXQxNiEhIQ==")

	// VDI/AD TLS 证书校验开关默认 true（跳过校验）
	// 原因：VDI/AD 服务器在内网部署时常使用自签名证书,跳过校验以保持向后兼容。
	// 生产环境部署时应在 configs/config.yaml 中显式设为 false,启用严格证书校验,避免 MITM 攻击。
	viper.SetDefault("vdi.tls_skip_verify", true)
	viper.SetDefault("ad.tls_skip_verify", true)
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	// 使用 Asia/Shanghai 时区
	// Cron 计算时使用本地时区（CST），直接存储，不做时区转换
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s timezone=Asia/Shanghai",
		c.Host, c.User, c.Password, c.DBName, c.Port, c.SSLMode)
}

// GetRedisAddr 获取Redis地址
func (c *CacheConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
