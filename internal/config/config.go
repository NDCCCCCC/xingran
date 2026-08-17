package config

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/spf13/viper"
)

// Config 应用配置。
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

// VDIConfig VDI 客户端配置。
type VDIConfig struct {
	// TLSSkipVerify 是否跳过 VDI 服务器 TLS 证书校验。
	// 默认 true 保持向后兼容(VDI 服务器通常使用自签名证书)。
	// 生产环境应在配置文件中设为 false,启用严格证书校验。
	TLSSkipVerify bool `mapstructure:"tls_skip_verify"`
}

// ADTLSConfig AD/LDAP TLS 配置。
type ADTLSConfig struct {
	// TLSSkipVerify 是否跳过 AD/LDAP 服务器 TLS 证书校验。
	// 默认 true 保持向后兼容(AD 域控在内网常使用自签名证书)。
	// 生产环境应在配置文件中设为 false,启用严格证书校验。
	TLSSkipVerify bool `mapstructure:"tls_skip_verify"`
}

// ServerConfig 服务器配置。
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

// DatabaseConfig 数据库配置。
type DatabaseConfig struct {
	Type         string `mapstructure:"type"` // postgres | sqlite(缺省按 postgres 处理,行为与历史一致)
	Path         string `mapstructure:"path"` // sqlite 文件路径,仅 type=sqlite 时生效(默认 data/xingran.db)
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

// CacheConfig 缓存配置。
type CacheConfig struct {
	Type          string `mapstructure:"type"` // redis, memory
	Host          string `mapstructure:"host"`
	Port          int    `mapstructure:"port"`
	Password      string `mapstructure:"password"`
	DB            int    `mapstructure:"db"`
	PoolSize      int    `mapstructure:"pool_size"`
	TLS           bool   `mapstructure:"tls"` // 托管 Redis (Upstash 等) 需要 true (本地 Redis 留 false)
	MaxSize       int    `mapstructure:"max_size"`
	CleanupTime   int    `mapstructure:"cleanup_time"`
	WarmUpEnabled bool   `mapstructure:"warm_up_enabled"` // 是否启用缓存预热
	// L2写入Worker配置。
	L2Writer *cache.L2WriterConfig `mapstructure:"l2_writer"` // L2写入Worker Pool配置。
	// 重试配置。
	RetryEnabled       bool    `mapstructure:"retry_enabled"`        // 是否启用L2缓存写入重试
	RetryMaxRetries    int     `mapstructure:"retry_max_retries"`    // 最大重试次数
	RetryInitialDelay  int     `mapstructure:"retry_initial_delay"`  // 初始延迟(毫秒)
	RetryMaxDelay      int     `mapstructure:"retry_max_delay"`      // 最大延迟(毫秒)
	RetryBackoffFactor float64 `mapstructure:"retry_backoff_factor"` // 退避因子
	RetryWorkerCount   int     `mapstructure:"retry_worker_count"`   // 重试工作协程数
}

// JWTConfig JWT配置。
type JWTConfig struct {
	SecretKey        string `mapstructure:"secret_key"`
	AccessKeyExpire  int    `mapstructure:"access_key_expire"`
	RefreshKeyExpire int    `mapstructure:"refresh_key_expire"`
	Issuer           string `mapstructure:"issuer"`
	UseSM2           bool   `mapstructure:"use_sm2"`         // 是否使用SM2算法
	SM2PrivateKey    string `mapstructure:"sm2_private_key"` // SM2私钥(十六进制)
	SM2PublicKey     string `mapstructure:"sm2_public_key"`  // SM2公钥(十六进制)
}

// LogConfig 日志配置。
type LogConfig struct {
	Level      string `mapstructure:"level"`       // 日志级别: debug, info, warn, error
	LogDir     string `mapstructure:"log_dir"`     // 日志目录
	MaxSize    int    `mapstructure:"max_size"`    // 单个日志文件最大大小(MB)
	MaxBackups int    `mapstructure:"max_backups"` // 保留的旧日志文件最大数量
	MaxAge     int    `mapstructure:"max_age"`     // 保留旧日志文件的最大天数
	Compress   bool   `mapstructure:"compress"`    // 是否压缩旧日志文件
}

// SecurityConfig 安全配置。
type SecurityConfig struct {
	SM4Key            string                  `mapstructure:"sm4_key"`            // SM4 加密密钥(16字节,Base64编码),用于所有敏感数据加密
	RequestEncryption RequestEncryptionConfig `mapstructure:"request_encryption"` // 请求体加密配置
	ReplayWindowSec   int                     `mapstructure:"replay_window_sec"`  // 加密请求时间戳容差(秒),±N。0=使用默认60
}

// RequestEncryptionConfig 请求体加密配置。
type RequestEncryptionConfig struct {
	Enabled           bool     `mapstructure:"enabled"`            // 是否启用
	ExcludePaths      []string `mapstructure:"exclude_paths"`      // 排除路径
	RequireEncryption bool     `mapstructure:"require_encryption"` // 是否强制加密
}

// OperLogConfig 操作日志配置。
type OperLogConfig struct {
	// ExcludePaths 排除写入 sys_oper_log 的请求路径。
	// 匹配语义: filepath.Match + /* 后缀通配,与 security.request_encryption.exclude_paths 完全一致。
	// 默认包含 /api/v1/rpa/workers/*/heartbeat(30s/Worker 高频心跳写入会淹没日志表)。
	ExcludePaths []string `mapstructure:"exclude_paths"`
}

// BaiduConfig 百度地图配置。
type BaiduConfig struct {
	MapAK string `mapstructure:"map_ak"` // 百度地图API Key
}

// RPAConfig RPA配置。
type RPAConfig struct {
	Enabled bool             `mapstructure:"enabled"` // 是否启用RPA功能
	AI      RPAAIConfig      `mapstructure:"ai"`      // AI配置
	Worker  RPAWorkerConfig  `mapstructure:"worker"`  // Worker配置
	Scaling RPAScalingConfig `mapstructure:"scaling"` // 扩缩容配置
	Storage RPAStorageConfig `mapstructure:"storage"` // 存储配置
}

// RPAAIConfig RPA AI配置。
type RPAAIConfig struct {
	Generator RPAAIGeneratorConfig `mapstructure:"generator"` // 脚本生成模型配置
	Agent     RPAAIAgentConfig     `mapstructure:"agent"`     // AI Agent模型配置
	Fallback  RPAAIFallbackConfig  `mapstructure:"fallback"`  // 降级策略配置
}

// RPAAIGeneratorConfig 脚本生成模型配置。
type RPAAIGeneratorConfig struct {
	Enabled   bool   `mapstructure:"enabled"`    // 是否启用
	APIKey    string `mapstructure:"api_key"`    // API密钥
	BaseURL   string `mapstructure:"base_url"`   // API基础URL(OpenAI兼容)
	Model     string `mapstructure:"model"`      // 模型名称
	MaxTokens int    `mapstructure:"max_tokens"` // 最大token数
}

// RPAAIAgentConfig AI Agent模型配置。
type RPAAIAgentConfig struct {
	Enabled   bool   `mapstructure:"enabled"`    // 是否启用
	APIKey    string `mapstructure:"api_key"`    // API密钥
	BaseURL   string `mapstructure:"base_url"`   // API基础URL(OpenAI兼容)
	Model     string `mapstructure:"model"`      // 模型名称
	MaxTokens int    `mapstructure:"max_tokens"` // 最大token数
}

// RPAAIFallbackConfig AI降级策略配置。
type RPAAIFallbackConfig struct {
	MaxRetries  int    `mapstructure:"max_retries"`  // 最大重试次数
	EnableAgent bool   `mapstructure:"enable_agent"` // 是否启用AI Agent降级
	Timeout     string `mapstructure:"timeout"`      // 请求超时时间
}

// RPAWorkerConfig RPA Worker配置。
type RPAWorkerConfig struct {
	MinWorkers        int    `mapstructure:"min_workers"`        // 最小Worker数量
	MaxWorkers        int    `mapstructure:"max_workers"`        // 最大Worker数量
	HeartbeatInterval string `mapstructure:"heartbeat_interval"` // 心跳间隔
	TaskTimeout       string `mapstructure:"task_timeout"`       // 任务超时时间
}

// RPAStorageConfig RPA存储配置。
type RPAStorageConfig struct {
	DownloadsDir    string `mapstructure:"downloads_dir"`     // 下载文件目录
	ScreenshotsDir  string `mapstructure:"screenshots_dir"`   // 截图目录
	MaxRetainedDays int    `mapstructure:"max_retained_days"` // 最大保留天数
}

// RPAScalingConfig RPA扩缩容配置。
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

// DockerConfig Docker 配置。
type DockerConfig struct {
	DockerHost    string `mapstructure:"docker_host"`    // Docker 主机地址
	DockerPort    int    `mapstructure:"docker_port"`    // Docker API 端口
	ContainerName string `mapstructure:"container_name"` // 容器名称前缀
	ImageName     string `mapstructure:"image_name"`     // 镜像名称
	NetworkName   string `mapstructure:"network_name"`   // 网络名称
}

// envBinding 声明一个需要绑定到 viper key 的"裸名"环境变量。
//
// viper 规则: 后注册的 BindEnv 优先级更高,所以 bindEnvVars 中显式注册
// 的"裸名"会赢过 AutomaticEnv 自动绑定的 XINGRAN_<KEY>。
type envBinding struct {
	key string // viper key,如 "database.host"
	env string // 环境变量名,如 "DB_HOST"
}

// Load 加载配置。
//
// 返回错误的情形:
//   - 配置文件存在但格式错误(无法解析)
//   - Unmarshal 失败
//   - Validate() 校验不通过(如 SM4 密钥为空、Server.Mode 非法)
//
// 配置文件不存在(ConfigFileNotFoundError)被视为"使用默认配置",不返回错误,
// 以保留现有 dev 工作流(没有 config.yaml 也能跑起来,靠默认值 + 环境变量)。
//
// 环境变量优先级(高 → 低):
//  1. bindEnvVars() 显式绑定的"裸名"(如 DB_HOST,REDIS_HOST,SM4_KEY)
//  2. AutomaticEnv 自动绑定的 XINGRAN_<KEY>(如 XINGRAN_DATABASE_HOST)
//  3. config.yaml 中的显式配置
//  4. setDefaults() 的默认值
//
// 新增配置字段时,在 bindEnvVars() 中显式声明 env 变量名(裸名优先),
// 同时 AutomaticEnv 会自动提供 XINGRAN_<KEY> 作为兜底。
//
// 并发安全(WR-2 修复):每次调用都通过 viper.New() 创建独立实例,所有
// SetDefault / BindEnv / ReadInConfig / Unmarshal 都在私有实例上完成,
// 不再触碰进程级全局 viper。这样 VDI(sync.Once 懒加载)与 AD(另一个
// sync.Once)首次请求并发触发 Load() 时,各自操作独立的 viper 实例,
// 消除了旧实现"全局 viper + viper.Reset()"的 data race。
//
// ctx 当前为 placeholder(IN-1):viper.ReadInConfig 是阻塞磁盘读、不支持
// ctx 取消,传 context.WithTimeout 并不能限制 Load 的阻塞时间(磁盘读通常
// < 100ms)。参数仅为未来切换到可中断 reader 时保持 API 稳定而保留,当前
// 不影响任何行为。
func Load(ctx context.Context) (*Config, error) {
	_ = ctx // placeholder,详见上方 doc

	// WR-2 修复:独立 viper 实例,彻底隔离全局状态。旧实现依赖全局 viper 单例 +
	// viper.Reset() 清空,但 VDI / AD 两个独立 sync.Once 懒加载并发首次访问时,
	// 两次 Load() 会在无 mutex 保护的全局 viper map 上 data race(B 的 Reset 可能
	// 清掉 A 刚 SetDefault 的状态)。独立实例让每次 Load 互不干扰,Reset 也不再必要。
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	// 设置默认值
	setDefaults(v)

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			applogger.Info("配置文件未找到，使用默认配置")
		} else {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	// 设置环境变量前缀 + 自动绑定(提供 XINGRAN_<KEY> 兜底)
	v.SetEnvPrefix("XINGRAN")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 显式绑定"裸名"环境变量,优先级高于 AutomaticEnv 自动绑定
	// (后注册的 env 在 viper 中优先级更高)
	if err := bindEnvVars(v); err != nil {
		return nil, fmt.Errorf("绑定环境变量失败: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 不变量校验 + 生产模式安全告警
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置校验失败: %w", err)
	}
	warnSecurityRisks(&config)

	return &config, nil
}

// bindEnvVars 集中声明所有"裸名"环境变量覆盖关系。
//
// 命名约定:
//   - DB / Redis / JWT / SM4 / Baidu / RPA AI 等"运维常接触"的配置:
//     用裸名(如 DB_HOST/REDIS_HOST/SM4_KEY),免去 .env 写 XINGRAN_ 前缀,
//     与 Docker / k8s / docker-compose 部署惯例一致
//   - 其他配置不显式绑定,AutomaticEnv 自动提供 XINGRAN_<KEY> 兜底
//   - viper 规则: 后注册的 BindEnv 优先级更高,所以这里显式注册的"裸名"
//     会赢过 AutomaticEnv 自动绑定的 XINGRAN_<KEY>
//
// 新增配置字段需要从 env 注入时,在下表追加一行即可。
func bindEnvVars(v *viper.Viper) error {
	bindings := []envBinding{
		// 数据库
		{"database.host", "DB_HOST"},
		{"database.user", "DB_USER"},
		{"database.password", "DB_PASSWORD"},
		{"database.dbname", "DB_NAME"},
		{"database.sslmode", "DB_SSLMODE"},

		// 缓存(Redis)
		{"cache.host", "REDIS_HOST"},
		{"cache.password", "REDIS_PASSWORD"},
		{"cache.port", "REDIS_PORT"},
		{"cache.db", "REDIS_DB"},

		// 服务器(Server.* 用 SERVER_ 前缀,免去 .env 写 XINGRAN_ 前缀)
		{"server.host", "SERVER_HOST"},
		{"server.mode", "SERVER_MODE"},
		{"server.port", "SERVER_PORT"},
		{"server.skip_setup", "SERVER_SKIP_SETUP"},

		// JWT
		{"jwt.secret_key", "JWT_SECRET"},
		// SM2 密钥对 — 必须从 env 注入;留空会让 jwt.go 走"动态生成"分支,
		// 导致每次重启 SM2 密钥对都换,旧 token 全部失效,所有用户被强制登出
		{"jwt.sm2_private_key", "JWT_SM2_PRIVATE_KEY"},
		{"jwt.sm2_public_key", "JWT_SM2_PUBLIC_KEY"},

		// 安全
		{"security.sm4_key", "SM4_KEY"},

		// 百度地图
		{"baidu.map_ak", "BAIDU_MAP_AK"},

		// RPA AI
		{"rpa.ai.generator.api_key", "RPA_AI_GENERATOR_KEY"},
		{"rpa.ai.generator.base_url", "RPA_AI_GENERATOR_URL"},
		{"rpa.ai.agent.api_key", "RPA_AI_AGENT_KEY"},
		{"rpa.ai.agent.base_url", "RPA_AI_AGENT_URL"},
	}

	for _, b := range bindings {
		if err := v.BindEnv(b.key, b.env); err != nil {
			// WR-5 修复:BindEnv 在参数非法时会返回 error,这里聚合返回。
			return fmt.Errorf("绑定 %s→%s 失败: %w", b.env, b.key, err)
		}
	}
	return nil
}

// Validate 校验配置不变量,启动时阻断致命配置错误。
//
// 严格校验(返回 error,启动失败):
//   - Server.Mode 必须是 debug/release/test 之一
//   - Security.SM4Key 非空(否则 SM4 加密失效,所有凭据明文存储)
//   - Database.MaxOpenConns >= MaxIdleConns(否则连接池配置无意义且报错)
//
// 弱校验(由 warnSecurityRisks 处理,仅 WARN 不阻断):
//   - 任意模式下 DB 密码为空 / VDI-AD TLS 跳过校验 / JWT 密钥为空
func (c *Config) Validate() error {
	switch c.Server.Mode {
	case "debug", "release", "test":
	default:
		return fmt.Errorf("server.mode 必须是 debug/release/test,当前=%q", c.Server.Mode)
	}

	if c.Security.SM4Key == "" {
		return errors.New("security.sm4_key 必须配置(从环境变量 SM4_KEY 或 config.yaml 注入,生成命令: openssl rand -base64 16)")
	}

	if c.Database.MaxOpenConns > 0 && c.Database.MaxIdleConns > 0 && c.Database.MaxOpenConns < c.Database.MaxIdleConns {
		return fmt.Errorf("database.max_open_conns(%d) 不能小于 max_idle_conns(%d)", c.Database.MaxOpenConns, c.Database.MaxIdleConns)
	}

	return nil
}

// warnSecurityRisks 输出安全告警(WARN 不阻断启动)。
//
// 设计取舍(IN-1 修复):这些不是"硬错误",运维有时确实因为内网证书
// 不便而暂时跳过校验。但必须有显眼的日志提醒,避免遗忘。
//
// release 模式下用 ⚠️ [SECURITY] 前缀 + emoji 强调;其他模式降级为 Info,
// 提醒但不打扰开发体验。
func warnSecurityRisks(c *Config) {
	isProd := c.Server.Mode == "release"

	if c.Database.Password == "" {
		if isProd {
			applogger.Warn("⚠️ [SECURITY] release 模式下 database.password 为空,数据库连接将以无密码方式尝试,生产部署必须设置 DB_PASSWORD")
		} else {
			applogger.Info("[config] database.password 为空,如需密码认证请设置 DB_PASSWORD 环境变量")
		}
	}

	if c.VDI.TLSSkipVerify {
		if isProd {
			applogger.Warn("⚠️ [SECURITY] release 模式下 vdi.tls_skip_verify=true,VDI 服务器 TLS 证书校验已禁用,存在 MITM 风险")
		} else {
			applogger.Info("[config] vdi.tls_skip_verify=true(跳过 VDI TLS 校验),生产环境应在 yaml 中显式设为 false")
		}
	}

	if c.AD.TLSSkipVerify {
		if isProd {
			applogger.Warn("⚠️ [SECURITY] release 模式下 ad.tls_skip_verify=true,AD/LDAP 服务器 TLS 证书校验已禁用,存在 MITM 风险")
		} else {
			applogger.Info("[config] ad.tls_skip_verify=true(跳过 AD/LDAP TLS 校验),生产环境应在 yaml 中显式设为 false")
		}
	}

	if c.JWT.SecretKey == "" {
		if isProd {
			applogger.Warn("⚠️ [SECURITY] release 模式下 jwt.secret_key 为空,JWT 签名密钥将由 NewJWTManager 启动时拒绝生成,服务可能启动失败")
		} else {
			applogger.Info("[config] jwt.secret_key 为空,请通过 JWT_SECRET 环境变量或 config.yaml 注入")
		}
	}
}

// setDefaults 设置默认配置。
func setDefaults(v *viper.Viper) {
	// 服务器默认配置
	v.SetDefault("server.name", "Xingran-Next")
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")

	// 数据库默认配置
	// database.type: postgres(生产/默认) | sqlite(本地 dev 提速,纯 Go 驱动 glebarez/sqlite)。
	// 缺省 postgres 与历史行为完全一致 — 不会因新增配置项静默切换数据库。
	v.SetDefault("database.type", "postgres")
	v.SetDefault("database.path", "data/xingran.db") // 仅 type=sqlite 时生效
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	// database.password 默认值已删除(F-04 同源):
	// 历史上默认值 "postgres" 是 Postgres 镜像出厂密码,部署到生产忘了改会变成公开已知弱口令。
	// dev 环境靠 configs/config.dev.yaml 显式注入;生产环境必须从 env (DB_PASSWORD) 注入。
	// Validate() 在 release 模式下会强制校验非空。
	v.SetDefault("database.dbname", "xingran_next")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_lifetime", 3600)

	// 缓存默认配置
	v.SetDefault("cache.type", "memory")
	v.SetDefault("cache.host", "localhost")
	v.SetDefault("cache.port", 6379)
	v.SetDefault("cache.password", "")
	v.SetDefault("cache.db", 0)
	v.SetDefault("cache.pool_size", 10)
	v.SetDefault("cache.max_size", 1000)
	v.SetDefault("cache.cleanup_time", 600)

	// JWT默认配置
	// F-04: 不再设置硬编码默认密钥 "xingran-next-secret-key" —
	// 该默认值若被部署进生产,任何人都能用公开已知字符串伪造 JWT。
	// 启动时由 NewJWTManager 强制校验:必须从 env 或 config 注入非空且非弱默认值。
	v.SetDefault("jwt.secret_key", "")
	v.SetDefault("jwt.access_key_expire", 7200)
	v.SetDefault("jwt.refresh_key_expire", 604800)
	v.SetDefault("jwt.issuer", "Xingran-Next")

	// 日志默认配置
	v.SetDefault("log.level", "info")
	v.SetDefault("log.log_dir", "logs")
	v.SetDefault("log.max_size", 100)   // 100MB
	v.SetDefault("log.max_backups", 30) // 保留30个备份
	v.SetDefault("log.max_age", 90)     // 保留90天
	v.SetDefault("log.compress", true)  // 压缩旧日志

	// 安全默认配置(注意:生产环境必须从环境变量设置)
	// SM4 密钥必须是 16 字节(Base64 编码)。
	// 历史默认值 "dGVzdC1zZWNyZXQxNiEhIQ==" (解码后 "test-secret16!!!") 已被删除:
	// 该值是公开已知字符串,任何人都能解密用它加密的 SM4 数据(设备密码、AD 密码、RPA 凭据)。
	// dev 环境靠 configs/config.dev.yaml 显式注入;生产必须从 env (SM4_KEY) 注入。
	// Validate() 强制校验非空。
	v.SetDefault("security.sm4_key", "")

	// VDI/AD TLS 证书校验开关默认 true(跳过校验)
	// 原因:VDI/AD 服务器在内网部署时常使用自签名证书,跳过校验以保持向后兼容。
	// 生产环境部署时应在 configs/config.yaml 中显式设为 false,启用严格证书校验,避免 MITM 攻击。
	v.SetDefault("vdi.tls_skip_verify", true)
	v.SetDefault("ad.tls_skip_verify", true)
}

// GetDSN 获取数据库连接字符串(URL 格式)。
//
// 使用 net/url 构造的好处:
//   - url.UserPassword 自动 URL-escape 密码中的特殊字符
//     (如 @ : / ? # 空格 等),避免 DSN 解析错位
//   - 密码含特殊字符时也能正确连接(旧版 fmt.Sprintf 拼接会导致 lib/pq 解析错误)
//
// 例:
//
//	password="p@ss:word/123"
//	旧 DSN: host=... user=postgres password=p@ss:word/123 dbname=...
//	  ← 解析歧义,实际跑到 host=p, password=ss:word/123
//	新 DSN: postgres://postgres:p%40ss%3Aword%2F123@host:5432/db?sslmode=disable
//	  ← 正确转义,lib/pq / pgx 都能正确解析
//
// 时区:固定 Asia/Shanghai(Cron 计算使用本地时区 CST,不做时区转换)。
func (c *DatabaseConfig) GetDSN() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:   "/" + c.DBName,
	}
	q := u.Query()
	q.Set("sslmode", c.SSLMode)
	q.Set("timezone", "Asia/Shanghai")
	// 连接健壮性(debug backend-hang-on-automigrate):握手超时 + TCP keepalive。
	// 底层走 pgx(pgx.ParseConfig),解析器把这些 libpq 关键字映射到 net.Dialer。
	// Supabase 新加坡 pooler 链路存在随机 TCP 黑洞:server 端确认无锁无 active
	// query,查询在网络层被静默丢弃,无 keepalive 时被丢弃连接上的 Read 永久阻塞
	// => 启动挂死在 AutoMigrate 内省 / InitData seed。配置后 idle 10s 探测,每 5s
	// 一次,3 次无 ACK 判死(~25s)返回 error,把无限挂起转化为有界错误。
	q.Set("connect_timeout", "20")
	q.Set("keepalive_idle", "10")
	q.Set("keepalive_interval", "5")
	q.Set("keepalive_count", "3")
	u.RawQuery = q.Encode()
	return u.String()
}

// GetRedisAddr 获取Redis地址。
func (c *CacheConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}