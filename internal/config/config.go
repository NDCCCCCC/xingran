package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

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
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Name string `mapstructure:"name"`
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
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
	Type        string `mapstructure:"type"` // redis, memory
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	Password    string `mapstructure:"password"`
	DB          int    `mapstructure:"db"`
	PoolSize    int    `mapstructure:"pool_size"`
	MaxSize     int    `mapstructure:"max_size"`
	CleanupTime int    `mapstructure:"cleanup_time"`
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
	SM4Key            string                  `mapstructure:"sm4_key"`            // SM4 加密密钥（16字节，Base64编码）
	RequestEncryption RequestEncryptionConfig `mapstructure:"request_encryption"` // 请求体加密配置
}

// RequestEncryptionConfig 请求体加密配置
type RequestEncryptionConfig struct {
	Enabled           bool     `mapstructure:"enabled"`            // 是否启用
	ExcludePaths      []string `mapstructure:"exclude_paths"`      // 排除路径
	RequireEncryption bool     `mapstructure:"require_encryption"` // 是否强制加密
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
			log.Println("配置文件未找到，使用默认配置")
		} else {
			log.Fatalf("读取配置文件失败: %v", err)
		}
	}

	// 设置环境变量前缀
	viper.SetEnvPrefix("RUOYI")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 环境变量覆盖
	viper.AutomaticEnv()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("解析配置失败: %v", err)
	}

	// 从环境变量直接读取敏感配置
	if host := os.Getenv("DB_HOST"); host != "" {
		config.Database.Host = host
	}
	if user := os.Getenv("DB_USER"); user != "" {
		config.Database.User = user
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		config.Database.Password = password
	}
	if dbname := os.Getenv("DB_NAME"); dbname != "" {
		config.Database.DBName = dbname
	}
	if port := os.Getenv("DB_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Database.Port = p
		}
	}
	if sslmode := os.Getenv("DB_SSLMODE"); sslmode != "" {
		config.Database.SSLMode = sslmode
	}
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		config.JWT.SecretKey = jwtSecret
	}
	if serverPort := os.Getenv("SERVER_PORT"); serverPort != "" {
		if p, err := strconv.Atoi(serverPort); err == nil {
			config.Server.Port = p
		}
	}
	// SM4 密钥从环境变量读取
	if sm4Key := os.Getenv("SM4_KEY"); sm4Key != "" {
		config.Security.SM4Key = sm4Key
	}

	return &config
}

// setDefaults 设置默认配置
func setDefaults() {
	// 服务器默认配置
	viper.SetDefault("server.name", "RuoYi-Next")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")

	// 数据库默认配置
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.dbname", "ruoyi_next")
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
	viper.SetDefault("jwt.secret_key", "ruoyi-next-secret-key")
	viper.SetDefault("jwt.access_key_expire", 7200)
	viper.SetDefault("jwt.refresh_key_expire", 604800)
	viper.SetDefault("jwt.issuer", "RuoYi-Next")

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
