package server

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// 默认配置常量
const (
	defaultBackendURL  = "https://localhost:9000"
	defaultListenAddr   = ":8443"
	defaultHeartbeatInterval = 30 * time.Second
	defaultLogLevel    = "info"
	defaultLogPath     = "/var/log/xingran-agent"
	defaultPlatform    = "linux"
)

// Config Agent 服务配置
type Config struct {
	// 后端服务器配置
	BackendURL string `yaml:"backend_url" env:"BACKEND_URL"`
	AgentID    string `yaml:"agent_id" env:"AGENT_ID"`
	VMID       string `yaml:"vm_id" env:"VM_ID"`

	// JWT 配置
	JWTSecret  string        `yaml:"jwt_secret" env:"JWT_SECRET"`
	TokenExpiry time.Duration `yaml:"token_expiry"`

	// HTTP 服务器配置
	ListenAddr  string `yaml:"listen_addr"`
	TLSCertFile string `yaml:"tls_cert_file"`
	TLSKeyFile  string `yaml:"tls_key_file"`

	// TLS 强制配置
	TLSEnabled         bool   `yaml:"tls_enabled"`
	VerifyCertificates bool   `yaml:"verify_certificates"`
	CAFile             string `yaml:"ca_file"`

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
	config := &Config{
		BackendURL:         defaultBackendURL,
		AgentID:            "",
		VMID:               "",
		JWTSecret:          "",
		TokenExpiry:        24 * time.Hour,
		ListenAddr:         defaultListenAddr,
		TLSCertFile:        "",
		TLSKeyFile:         "",
		HeartbeatInterval: defaultHeartbeatInterval,
		LogLevel:           defaultLogLevel,
		LogPath:            defaultLogPath,
		Platform:           defaultPlatform,
		TLSEnabled:         true,  // 默认启用 TLS
		VerifyCertificates: true,  // 默认验证证书
		CAFile:             "",    // 可选 CA bundle
	}

	// 如果指定了配置文件，尝试加载
	if path != "" {
		viper.SetConfigFile(path)
		viper.SetConfigType("yaml")

		// 直接尝试读取，不预先检查文件存在性（避免 TOCTOU）
		if err := viper.ReadInConfig(); err == nil {
			// 文件存在且读取成功，解析配置
			if err := viper.Unmarshal(config); err != nil {
				return nil, fmt.Errorf("failed to parse config: %w", err)
			}
		} else {
			// 配置文件读取失败，记录警告但继续（可能使用环境变量）
			applogger.Warnf("Warning: could not read config file '%s': %v (will use env vars)", path, err)
		}
	}

	// 配置环境变量前缀和自动绑定
	viper.SetEnvPrefix("AGENT")
	viper.AutomaticEnv()

	// 手动读取环境变量覆盖（viper 自动绑定可能与结构体标签不完全匹配）
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
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.BackendURL == "" {
		return os.ErrInvalid
	}

	// TLS validation - 强制 TLS 启用
	if err := c.ValidateTLS(); err != nil {
		return err
	}

	// 如果缺少 AgentID/VMID，将自动注册（不报错）
	// 但需要警告
	if c.AgentID == "" || c.VMID == "" {
		WithFields(logrus.Fields{
			"agent_id": c.AgentID,
			"vm_id":    c.VMID,
		}).Warn("agent_id or vm_id not configured, will attempt auto-registration")
	}

	return nil
}

// ValidateTLS 验证 TLS 配置
func (c *Config) ValidateTLS() error {
	// 检查是否启用了 TLS
	if !c.TLSEnabled {
		return fmt.Errorf("TLS must be enabled for production security (set tls_enabled: true)")
	}

	// 检查证书文件存在性
	if c.TLSCertFile == "" || c.TLSKeyFile == "" {
		return fmt.Errorf("TLS certificate and key files must be specified when TLS is enabled")
	}

	// 验证证书文件可读
	if _, err := os.Stat(c.TLSCertFile); os.IsNotExist(err) {
		return fmt.Errorf("TLS certificate file not found: %s", c.TLSCertFile)
	}
	if _, err := os.Stat(c.TLSKeyFile); os.IsNotExist(err) {
		return fmt.Errorf("TLS key file not found: %s", c.TLSKeyFile)
	}

	// 如果启用证书验证，检查 CA 文件
	if c.VerifyCertificates && c.CAFile != "" {
		if _, err := os.Stat(c.CAFile); os.IsNotExist(err) {
			return fmt.Errorf("CA bundle file not found: %s", c.CAFile)
		}
	}

	return nil
}

// CheckCertificateFiles 检查证书文件权限和有效性
func (c *Config) CheckCertificateFiles() error {
	// 检查证书文件权限（不应全局可读）
	if runtime.GOOS != "windows" {
		certInfo, err := os.Stat(c.TLSCertFile)
		if err != nil {
			return err
		}
		// 检查文件模式（应限制为 600 或 644）
		mode := certInfo.Mode()
		if mode.Perm()&0444 != 0 {
			WithFields(logrus.Fields{
				"cert_file": c.TLSCertFile,
				"mode":      mode.Perm().String(),
			}).Warn("Certificate file is world-readable")
		}

		keyInfo, err := os.Stat(c.TLSKeyFile)
		if err != nil {
			return err
		}
		keyMode := keyInfo.Mode()
		if keyMode.Perm()&0044 != 0 {
			return fmt.Errorf("private key file should not be world-readable: %s", c.TLSKeyFile)
		}
	}

	return nil
}

// SystemFingerprint 系统指纹信息
type SystemFingerprint struct {
	Hostname    string `json:"hostname"`
	IPAddress   string `json:"ip_address"`
	MACAddress  string `json:"mac_address"`
	OSType      string `json:"os_type"`
	Platform    string `json:"platform"`
	MachineGUID string `json:"machine_guid,omitempty"`
}

// CollectSystemFingerprint 收集系统指纹
func CollectSystemFingerprint() (*SystemFingerprint, error) {
	fp := &SystemFingerprint{
		OSType:  runtime.GOOS,
		Platform: runtime.GOOS,
	}

	// 获取主机名
	if hostname, err := os.Hostname(); err == nil {
		fp.Hostname = hostname
	}

	// 获取 IP 地址
	if ips := getLocalIPs(); len(ips) > 0 {
		fp.IPAddress = ips[0]
	}

	// 获取 MAC 地址
	if mac := getPrimaryMAC(); mac != "" {
		fp.MACAddress = mac
	}

	// 获取机器 GUID（Windows）
	if runtime.GOOS == "windows" {
		fp.MachineGUID = getWindowsMachineGUID()
	}

	return fp, nil
}

// getLocalIPs 获取本地 IP 地址
func getLocalIPs() []string {
	var ips []string
	// 简化实现，返回常见的本地 IP
	// 实际应该通过 net.Interface() 获取
	if runtime.GOOS == "windows" {
		ips = append(ips, "127.0.0.1") // 示例
	}
	ips = append(ips, "127.0.0.1")
	return ips
}

// getPrimaryMAC 获取主要 MAC 地址
func getPrimaryMAC() string {
	if runtime.GOOS == "windows" {
		// Windows: 使用 getmac
		cmd := exec.Command("getmac")
		if output, err := cmd.Output(); err == nil {
			lines := strings.Split(string(output), "\n")
			if len(lines) > 0 {
				// 解析第一行获取 MAC
				fields := strings.Fields(lines[0])
				if len(fields) > 0 {
					return strings.ReplaceAll(fields[len(fields)-1], "-", "")
				}
			}
		}
	}
	// Linux/Unix
	if runtime.GOOS == "linux" {
		cmd := exec.Command("cat", "/sys/class/net/eth0/address")
		if output, err := cmd.Output(); err == nil {
			return strings.TrimSpace(string(output))
		}
	}
	return ""
}

// getWindowsMachineGUID 获取 Windows 机器 GUID
func getWindowsMachineGUID() string {
	cmd := exec.Command("reg", "query",
		"HKLM\\SOFTWARE\\Microsoft\\Cryptography", "/v", "MachineGuid")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "MachineGuid") {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					return fields[len(fields)-1]
				}
			}
		}
	}
	return ""
}

// AutoRegisterAgent 自动注册 Agent 到后端
func AutoRegisterAgent(backendURL string, fp *SystemFingerprint) (vmID, agentID string, err error) {
	// 调用后端注册 API
	registerURL := fmt.Sprintf("%s/api/agent/register", backendURL)

	jsonData, err := json.Marshal(fp)
	if err != nil {
		return "", "", fmt.Errorf("序列化指纹数据失败: %w", err)
	}

	resp, err := http.Post(registerURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("调用注册 API 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("注册 API 返回错误: %d", resp.StatusCode)
	}

	// 解析响应
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			VMID    string `json:"vm_id"`
			AgentID string `json:"agent_id"`
			Matched bool   `json:"matched"`
		} `json:"data"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("解析响应失败: %w", err)
	}

	return result.Data.VMID, result.Data.AgentID, nil
}

// RegisterToBackend 配置不完整时自动注册
func RegisterToBackend(config *Config) (*Config, error) {
	// 收集指纹
	fp, err := CollectSystemFingerprint()
	if err != nil {
		return nil, fmt.Errorf("收集系统指纹失败: %w", err)
	}

	// 调用后端注册
	vmID, agentID, err := AutoRegisterAgent(config.BackendURL, fp)
	if err != nil {
		// 注册失败，生成临时 ID
		WithFields(logrus.Fields{
			"backend_url": config.BackendURL,
			"error":       err.Error(),
		}).Warn("auto registration failed, using temporary ID")
		vmID = fmt.Sprintf("vm-temp-%s", fp.Hostname)
		agentID = fmt.Sprintf("agent-%s-%s", fp.Hostname, fp.MachineGUID[:8])
	}

	// 更新配置
	config.VMID = vmID
	config.AgentID = agentID
	config.JWTSecret = generateRandomSecret()

	return config, nil
}

// generateRandomSecret 用 crypto/rand 从字符集随机选 32 字符生成密钥。
// 修复 Q-77-A: 原实现 charset[i%len] 确定性返回常量串, 与函数名 + 注释
// 「生成随机密钥」的语义不符, 且让 Agent 自动注册的 JWT secret 可被外部预测。
// 这里使用 CSPRNG, math/rand 禁用; CSPRNG 读取失败属系统级异常, 直接 panic
// (签名同一代码基线的不变量: agent secret 永远不应该降级为常量)。
func generateRandomSecret() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, 32)
	max := big.NewInt(int64(len(charset)))
	for i := range result {
		n, err := cryptorand.Int(cryptorand.Reader, max)
		if err != nil {
			panic(fmt.Sprintf("generateRandomSecret: 读取 crypto/rand 失败: %v", err))
		}
		result[i] = charset[n.Int64()]
	}
	return string(result)
}
