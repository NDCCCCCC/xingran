---
phase: 22b-vm-agent-service
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - scripts/agent/install-windows.ps1
  - configs/agent-config.yaml
  - internal/agent/server/config.go
  - internal/agent/server/jwt_auth.go
  - internal/agent/server/handlers.go
autonomous: false
requirements:
  - UAT-Gap-1: Security Configuration Issues (tls_enabled, certificate_validation, error_sanitization, security_headers)
must_haves:
  truths:
    - "Agent 在生产环境使用加密通信（TLS 1.3）"
    - "Agent 验证后端证书防止中间人攻击"
    - "Windows Agent 使用 JEA 受限权限执行账号操作"
    - "错误消息不泄露内部系统信息"
    - "HTTP 响应包含安全头（CSP, X-Frame-Options, HSTS）"
  artifacts:
    - path: scripts/agent/install-windows.ps1
      provides: Windows JEA 配置和受限权限安装
      contains: "New-PSSessionConfigurationFile, Register-PSSessionConfiguration"
    - path: configs/agent-config.yaml
      provides: 安全配置模板（TLS 启用、证书验证）
      contains: "tls_enabled: true, verify_certificates: true"
    - path: internal/agent/server/config.go
      provides: 配置验证（TLS 证书检查）
      exports: "ValidateTLSConfig, CheckCertificateFiles"
    - path: internal/agent/server/jwt_auth.go
      provides: 安全的 HTTP 客户端（证书验证）
      contains: "InsecureSkipVerify: false"
    - path: internal/agent/server/handlers.go
      provides: 安全响应头和通用错误消息
      contains: "security headers, generic error messages"
  key_links:
    - from: scripts/agent/install-windows.ps1
      to: internal/agent/server/config.go
      via: JEA 配置与 Agent 权限限制
      pattern: "RestrictedRemoteServer, RoleDefinitions"
    - from: internal/agent/server/config.go
      to: configs/agent-config.yaml
      via: TLS 配置验证
      pattern: "tls_cert_file, tls_key_file validation"
    - from: internal/agent/server/handlers.go
      to: internal/agent/server/middleware.go
      via: 安全响应头中间件
      pattern: "Content-Security-Policy, X-Frame-Options"
---

# Phase 22B-01: 安全配置加固

## Objective

修复 VM Agent 的安全配置漏洞，实现生产级安全标准。解决 Windows 完全管理员权限、TLS 默认禁用、证书验证跳过和错误消息信息泄露问题。

**Purpose**: 确保 Agent 在生产环境中遵循最小权限原则和安全通信最佳实践

**Output**: 安全加固的 Agent 配置和部署脚本

## Context

@.planning/phases/22b-vm-agent-service/22b-UAT.md (Gap #1: Security Configuration Issues)
@.planning/phases/22b-vm-agent-service/plans/22-06-SUMMARY.md
@internal/agent/server/config.go
@internal/agent/server/jwt_auth.go
@scripts/agent/install-windows.ps1

### Security Issues from UAT

1. **Windows Agent 完全管理员权限**: 应使用 JEA (Just Enough Administration) 受限权限
2. **TLS 默认未启用**: 配置应强制 TLS 并验证证书文件
3. **证书验证跳过**: `InsecureSkipVerify: true` 应改为 `false` 并配置 CA bundle
4. **错误消息泄露**: 应使用通用错误消息而非内部系统信息

## Tasks

### Task 1: 实现 Windows JEA 配置

**Files**: `scripts/agent/install-windows.ps1`

**Read First**:
- `scripts/agent/install-windows.ps1` (current implementation with full admin rights)
- `https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_session_configurations` (JEA reference)

**Action**:
1. **Add JEA Configuration Function** after line 52 (after log directory creation):

```powershell
# Create JEA Session Configuration
function Create-JEAConfiguration {
    param(
        [string]$ConfigPath = "C:\Program Files\WindowsPowerShell\Configuration\sessionConfigs\XingRanAgent.pssc"
    )

    Write-Host "Creating JEA session configuration..."

    # Define role capabilities
    $roleCapabilityPath = "C:\Program Files\WindowsPowerShell\Configuration\RoleCapabilities\XingRanAgentRole.ps1"

    # Create role capabilities file
    @"
@{
    # ID used to uniquely identify this role capability
    GUID = 'a8c7f7e3-6d4a-4f5b-9e8a-2d3c4b5a6e7f'

    # Author of this role capability
    Author = 'XingRan VDI System'

    # Company or vendor of this role capability
    CompanyName = 'XingRan'

    # Description of the functionality provided by these role capabilities
    Description = 'Restricted role for XingRan VDI Agent - user account management only'

    # Modules to import when applied to a session
    # Modules = @()

    # Cmdlets to make visible when applied to a session
    VisibleCmdlets = @(
        'Microsoft.PowerShell.LocalAccounts\New-LocalUser',
        'Microsoft.PowerShell.LocalAccounts\Remove-LocalUser',
        'Microsoft.PowerShell.LocalAccounts\Enable-LocalUser',
        'Microsoft.PowerShell.LocalAccounts\Disable-LocalUser',
        'Microsoft.PowerShell.LocalAccounts\Set-LocalUser',
        'Microsoft.PowerShell.LocalAccounts\Get-LocalUser',
        'Microsoft.PowerShell.LocalAccounts\Get-LocalGroup',
        'Microsoft.PowerShell.LocalAccounts\Add-LocalGroupMember',
        'Microsoft.PowerShell.LocalAccounts\Remove-LocalGroupMember'
    )

    # Functions to make visible when applied to a session
    VisibleFunctions = @()

    # External commands (scripts and applications) to make visible when applied to a session
    VisibleExternalCommands = @()

    # Providers to make visible when applied to a session
    VisibleProviders = 'Variable', 'Function'

    # All rights are reserved unless explicitly defined
    FunctionDefinitions = @()

    # Alias definitions
    AliasDefinitions = @()

    # Rules for which commands are allowed
    FunctionRules = @()

    # Scripts to run when applied to a session
    ScriptsToProcess = @()

    # Types of variables to define in this session
    VariableDefinitions = @()

    # Environment variables to define in this session
    EnvironmentVariables = @()

    # Type definitions (using .ps1xml) to load when applied to a session
    TypesToProcess = @()

    # Format specifications (using .ps1xml) to load when applied to a session
    FormatsToProcess = @()

    # Assemblies to load when applied to a session
    AssembliesToLoad = @()
}
"@ | Out-File -FilePath $roleCapabilityPath -Encoding UTF8

    # Create session configuration file
    $sessionConfigParams = @{
        Path      = $ConfigPath
        SessionType = 'RestrictedRemoteServer'
        RunAsVirtualAccount = $true
        VirtualAccountType = 'LocalAccount'
        RoleDefinitions = @{
            'XingRanAgentUser' = @{ RoleCapabilityFiles = $roleCapabilityPath }
        }
        TranscriptDirectory = 'C:\Program Files\XingRanAgent\transcripts'
    }

    New-PSSessionConfigurationFile @sessionConfigParams -Force

    Write-Host "JEA configuration created at: $ConfigPath"
    Write-Host "Role capabilities at: $roleCapabilityPath"
}

# Create JEA configuration
Create-JEAConfiguration
```

2. **Replace lines 58-78** (full admin rights) with JEA-restricted user:

```powershell
# Create JEA-restricted service account
$password = ConvertTo-SecureString "XingRanAgent123!" -AsPlainText -Force
try {
    New-LocalUser -Name "XingRanAgentUser" -Password $password -Description "XingRan VDI Agent User (JEA-Restricted)" -ErrorAction Stop | Out-Null
    Write-Host "Created JEA-restricted user: XingRanAgentUser"
} catch {
    if ($_.Exception.Message -like "*already exists*") {
        Write-Host "User XingRanAgentUser already exists" -ForegroundColor Yellow
    } else {
        Write-Host "WARNING: Failed to create user: $_" -ForegroundColor Yellow
    }
}

# NO LONGER ADD TO ADMINISTRATORS GROUP - JEA provides restricted elevated privileges
# User remains a standard user with JEA virtual account for admin tasks
Write-Host "User configured with JEA restricted privileges (NOT in Administrators group)"
```

3. **Add JEA session registration** before Windows service creation (around line 95):

```powershell
# Register JEA session configuration
Write-Host "Registering JEA session configuration..."
try {
    $psscPath = "C:\Program Files\WindowsPowerShell\Configuration\sessionConfigs\XingRanAgent.pssc"
    if (Test-Path $psscPath) {
        Register-PSSessionConfiguration -Name "XingRanAgentJEA" -Path $psscPath -Force -NoServiceRestart
        Write-Host "JEA session 'XingRanAgentJEA' registered successfully"
    } else {
        Write-Host "WARNING: JEA configuration file not found at $psscPath" -ForegroundColor Yellow
    }
} catch {
    Write-Host "WARNING: Failed to register JEA configuration: $_" -ForegroundColor Yellow
}

# Restart WinRM service to apply JEA configuration
Write-Host "Restarting WinRM service..."
Restart-Service -Name WinRM -Force
```

**Verify**:
```bash
# In Windows VM, verify JEA configuration
Test-Path "C:\Program Files\WindowsPowerShell\Configuration\sessionConfigs\XingRanAgent.pssc"
Get-PSSessionConfiguration -Name "XingRanAgentJEA" | Select-Object Name, Permission
```

**Done**:
- [ ] JEA session configuration file created
- [ ] Role capabilities restricted to user management commands only
- [ ] XingRanAgentUser NOT in Administrators group
- [ ] JEA session registered with WinRM
- [ ] Transcript directory configured for auditing

---

### Task 2: 实现强制 TLS 配置验证

**Files**: `internal/agent/server/config.go`, `configs/agent-config.yaml`

**Read First**:
- `internal/agent/server/config.go` (current Config struct and validation)
- `configs/agent-config.yaml` (if exists, else create template)

**Action**:
1. **Add TLS configuration fields** to Config struct (after line 44):

```go
	// TLS 强制配置
	TLSEnabled         bool   `yaml:"tls_enabled"`
	VerifyCertificates bool   `yaml:"verify_certificates"`
	CAFile             string `yaml:"ca_file"`
```

2. **Add validation methods** after `Validate()` method (line 136):

```go
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
			log.Printf("WARNING: Certificate file is world-readable: %s", c.TLSCertFile)
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
```

3. **Update `Validate()` method** (line 126-137) to call TLS validation:

```go
// Validate 验证配置
func (c *Config) Validate() error {
	if c.BackendURL == "" {
		return fmt.Errorf("backend_url is required")
	}

	// TLS validation - 强制 TLS 启用
	if err := c.ValidateTLS(); err != nil {
		return err
	}

	// 如果缺少 AgentID/VMID，将自动注册（不报错）
	// 但需要警告
	if c.AgentID == "" || c.VMID == "" {
		log.Printf("WARNING: agent_id or vm_id not configured, will attempt auto-registration")
	}

	return nil
}
```

4. **Update default values** in `LoadConfig()` (line 59-72):

```go
	config := &Config{
		BackendURL:         defaultBackendURL,
		AgentID:            "",
		VMID:               "",
		JWTSecret:          "",
		TokenExpiry:        24 * time.Hour,
		ListenAddr:         defaultListenAddr,
		TLSCertFile:        "",
		TLSKeyFile:         "",
		HeartbeatInterval:  defaultHeartbeatInterval,
		LogLevel:           defaultLogLevel,
		LogPath:            defaultLogPath,
		Platform:           defaultPlatform,
		TLSEnabled:         true,  // 默认启用 TLS
		VerifyCertificates: true,  // 默认验证证书
		CAFile:             "",    // 可选 CA bundle
	}
```

5. **Create `configs/agent-config.yaml` template**:

```yaml
# XingRan VDI Agent Configuration Template
# IMPORTANT: Enable TLS for production deployments

# Backend Configuration
backend_url: "https://xingran-backend.example.com"
agent_id: ""  # Leave empty for auto-registration
vm_id: ""      # Leave empty for auto-registration

# Security Configuration
jwt_secret: ""  # Leave empty for auto-generation
tls_enabled: true  # MUST be true for production
verify_certificates: true  # MUST be true for production
tls_cert_file: "/etc/xingran-agent/tls/server.crt"
tls_key_file: "/etc/xingran-agent/tls/server.key"
ca_file: "/etc/xingran-agent/tls/ca.crt"  # Optional: for self-signed certs

# HTTP Server
listen_addr: ":8443"

# Heartbeat
heartbeat_interval: 30s

# Logging
log_level: "info"
log_path: "/var/log/xingran-agent"

# Platform (auto-detected, usually don't set manually)
platform: "linux"  # or "windows"
```

**Verify**:
```bash
# Build check
go build ./...

# Test configuration validation with TLS disabled (should fail)
go run -c 'tls_enabled: false, backend_url: "https://test.com"'
# Expected: error about TLS must be enabled

# Test with valid TLS config
go run -c 'tls_enabled: true, tls_cert_file: "/tmp/test.crt", tls_key_file: "/tmp/test.key"'
# Expected: passes validation
```

**Done**:
- [ ] Config struct has TLS enforcement fields
- [ ] `ValidateTLS()` enforces TLS enabled for production
- [ ] Certificate file existence checked at startup
- [ ] Default configuration has TLS enabled
- [ ] Template config shows security best practices
- [ ] File permission checks for private keys (Linux)
- [ ] Code compiles without errors

**Cross-plan compatibility note**: This plan modifies `jwt_auth.go` (Task 3). Plan 22b-02 also modifies `jwt_auth.go` (Task 1). When executing both plans, preserve changes from the other plan to avoid conflicts.

---

### Task 3: 修复证书验证跳过漏洞

**Files**: `internal/agent/server/jwt_auth.go`

**Read First**:
- `internal/agent/server/jwt_auth.go` (line 63-70, current InsecureSkipVerify: true)
- `internal/agent/server/config.go` (new TLS configuration fields)

**Action**:
1. **Replace HTTP client initialization** in `NewJWTAuthenticator()` (line 63-71):

```go
// NewJWTAuthenticator 创建 JWT 认证管理器
func NewJWTAuthenticator(secret, backendURL, agentID, vmID string, tlsConfig *tls.Config) *JWTAuthenticator {
	// 如果未提供 TLS 配置，使用安全的默认值
	if tlsConfig == nil {
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS13,  // 强制 TLS 1.3
			// InsecureSkipVerify 默认为 false，不设置
		}
	}

	return &JWTAuthenticator{
		secret:       secret,
		backendURL:   strings.TrimSuffix(backendURL, "/"),
		tokenExpiry:  defaultTokenExpiry,
		agentID:      agentID,
		vmID:         vmID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
				// 启用 HTTP/2 提升性能
				ForceAttemptHTTP2: true,
			},
		},
	}
}
```

2. **Add TLS config constructor** after `NewJWTAuthenticator()` (line 72):

```go
// NewTLSConfigFromConfig 从配置创建 TLS 配置
func NewTLSConfigFromConfig(certFile, keyFile, caFile string, verifyCertificates bool) (*tls.Config, error) {
	config := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	// 如果提供了 CA 文件，加载 CA 证书
	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		config.RootCAs = caCertPool
	}

	// 如果提供了客户端证书（mTLS），加载证书和私钥
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		config.Certificates = []tls.Certificate{cert}
	}

	// 根据配置决定是否验证服务器证书
	config.InsecureSkipVerify = !verifyCertificates

	if !verifyCertificates {
		log.Printf("WARNING: Certificate verification is DISABLED (insecure)")
	}

	return config, nil
}
```

3. **Add imports** at the top of the file (line 3-15, add to existing imports):

```go
import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"os"
)
```

4. **Update `CallBackend()` method** (line 177-196) to handle TLS errors:

```go
// CallBackend 调用后端 API
func (a *JWTAuthenticator) CallBackend(ctx context.Context, method, path string, body interface{}) (*response.Response, error) {
	// 获取有效令牌
	token, err := a.GetCurrentToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	// 构建请求
	url := a.backendURL + path

	// 序列化请求体
	var bodyReader *bytes.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize request: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// 发送请求（使用配置的 TLS 客户端）
	resp, err := a.httpClient.Do(req)
	if err != nil {
		// 检查是否是 TLS 错误
		if tlsErr, ok := err.(tls.CertificateVerificationError); ok {
			return nil, fmt.Errorf("TLS certificate verification failed: %w", tlsErr)
		}
		return nil, fmt.Errorf("backend request failed: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result response.Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
```

5. **Add missing imports** for the new `CallBackend` implementation:

```go
import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)
```

**Verify**:
```bash
# Build check
go build ./...

# Test TLS verification is enabled by default
go test -v -run TestNewJWTAuthenticator ./internal/agent/server/

# Test TLS config creation
go test -v -run TestNewTLSConfigFromConfig ./internal/agent/server/

# Check InsecureSkipVerify is false
grep -n "InsecureSkipVerify" internal/agent/server/jwt_auth.go
# Should only show in NewTLSConfigFromConfig with proper conditional logic
```

**Done**:
- [ ] `NewJWTAuthenticator()` accepts TLS config parameter
- [ ] `NewTLSConfigFromConfig()` creates secure TLS configuration
- [ ] `InsecureSkipVerify: true` only set when verifyCertificates = false
- [ ] Default HTTP client uses TLS 1.3 minimum
- [ ] TLS errors properly handled with clear error messages
- [ ] mTLS support with client certificates
- [ ] All imports added and code compiles

**Cross-plan compatibility note**: This plan modifies `jwt_auth.go` (CallBackend method). Plan 22b-02 Task 1 also modifies `jwt_auth.go` (adds retry logic to CallBackend). When executing both plans, merge the CallBackend implementations to include both TLS error handling and retry mechanism.

---

### Task 4: 实现通用错误消息和响应安全头

**Files**: `internal/agent/server/handlers.go`, `internal/agent/server/middleware.go`

**Read First**:
- `internal/agent/server/handlers.go` (current error responses)
- `internal/agent/server/middleware.go` (current middleware)

**Action**:
1. **Add generic error messages** at the top of `handlers.go` (after imports):

```go
// 通用错误消息（避免信息泄露）
const (
	errMsgGeneric         = "请求处理失败，请稍后重试"
	errMsgInvalidRequest  = "请求参数无效"
	errMsgUnauthorized    = "未授权访问"
	errMsgForbidden       = "权限不足"
	errMsgNotFound        = "资源不存在"
	errMsgInternalError   = "服务器内部错误"
	errMsgServiceUnavailable = "服务暂时不可用"
)

// 内部错误码（用于日志，不暴露给客户端）
const (
	errCodeInternal       = 1001
	errCodeDatabase       = 1002
	errCodeExternalAPI    = 1003
	errCodeConfig         = 1004
	errCodeAuth           = 1005
)

// sanitizeError 清理错误消息，移除敏感信息
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()

	// 检查是否包含敏感信息
	sensitivePatterns := []string{
		"password", "secret", "token", "key", "credential",
		"internal", "database", "sql", "query", "connection",
		"file://", "/etc/", "/var/", "C:\\",
	}

	errMsgLower := strings.ToLower(errMsg)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(errMsgLower, pattern) {
			log.Printf("Sanitized error containing '%s': %v", pattern, err)
			return errMsgGeneric
		}
	}

	return errMsg
}
```

2. **Create security headers middleware** in `middleware.go` (after existing middleware):

```go
// SecurityHeaders 添加安全响应头中间件
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Content Security Policy - 限制资源来源
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; object-src 'none'")

		// X-Frame-Options - 防止点击劫持
		c.Header("X-Frame-Options", "DENY")

		// X-Content-Type-Options - 防止 MIME 类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")

		// X-XSS-Protection - 启用 XSS 过滤
		c.Header("X-XSS-Protection", "1; mode=block")

		// Strict-Transport-Security - 强制 HTTPS (如果启用了 TLS)
		if c.Request.TLS != nil || c.Request.URL.Scheme == "https" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Referrer-Policy - 控制 Referer 信息泄露
		c.Header("Referrer-Policy", "no-referrer")

		// Permissions-Policy - 限制浏览器功能
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}
```

3. **Update all error responses** in `handlers.go` to use generic messages:

Replace all instances of direct error message exposure with sanitized versions:

```go
// CreateAccount 创建账号
func (h *AgentHandler) CreateAccount(c *gin.Context) {
	var req Account
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": errMsgInvalidRequest,
			"code": errCodeInternal,
		})
		return
	}

	if err := h.accountManager.CreateAccount(c.Request.Context(), &req); err != nil {
		log.Printf("Failed to create account: %v", err)  // 内部日志记录详细错误
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": sanitizeError(err),  // 客户端收到清理后的错误
			"code": errCodeInternal,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account created successfully"})
}
```

4. **Apply the same pattern** to all other handler methods (DeleteAccount, ResetPassword, EnableAccount, DisableAccount, Register, etc.):

- Replace direct `err.Error()` with `sanitizeError(err)`
- Add internal logging with `log.Printf("Detailed error: %v", err)`
- Use generic error messages from constants

5. **Update middleware registration** in handlers.go or main.go:

```go
// RegisterRoutes 注册路由
func (h *AgentHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")

	// 应用安全响应头中间件（全局）
	api.Use(SecurityHeaders())

	// 公开端点
	public := api.Group("")
	public.POST("/health", h.HealthCheck)
	public.POST("/register", h.Register)

	// 需要 JWT 认证的端点
	auth := api.Group("")
	auth.Use(JWTAuth(h.authenticator))
	{
		auth.POST("/accounts", h.CreateAccount)
		// ... other routes
	}
}
```

**Verify**:
```bash
# Build check
go build ./...

# Test error message sanitization
curl -X POST http://localhost:8443/api/v1/accounts -d '{"invalid": "data"}'
# Expected: {"error": "请求参数无效", "code": 1001}
# NOT: {"error": "invalid email format at field user.email", ...}

# Test security headers
curl -I http://localhost:8443/api/v1/health
# Expected: X-Frame-Options: DENY, X-Content-Type-Options: nosniff, etc.

# Test sensitive errors are sanitized
curl -X POST http://localhost:8443/api/v1/accounts -d '{"username": "test"}'
# If internal error (e.g., database connection issue), should get generic message
```

**Done**:
- [ ] Generic error message constants defined
- [ ] `sanitizeError()` function filters sensitive information
- [ ] `SecurityHeaders()` middleware adds all security headers
- [ ] All handler methods use sanitized error messages
- [ ] Internal logging preserves detailed errors for debugging
- [ ] Client receives only generic, safe error messages
- [ ] Security headers verified with curl or similar tool
- [ ] Code compiles without errors

## Threat Model

### Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Agent → Backend | HTTPS/TLS 通信通道（双向验证）|
| Windows OS → Agent | JEA 受限权限边界 |
| Client → Agent API | HTTP 安全响应头保护 |

### STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-22b-01 | Elevation of Privilege | Windows Agent 权限 | mitigate | JEA 受限权限，虚拟账户，仅允许特定命令 |
| T-22b-02 | Tampering | TLS 通信 | mitigate | TLS 1.3 强制，证书验证，CA bundle |
| T-22b-03 | Information Disclosure | 错误消息 | mitigate | 通用错误消息，敏感模式过滤 |
| T-22b-04 | Spoofing | Agent 身份 | mitigate | JWT 令牌，mTLS 支持 |
| T-22b-05 | Tampering | Web 控制台 | mitigate | CSP, X-Frame-Options, HSTS 响应头 |

## Verification

1. **JEA Configuration Test** (Windows VM):
```powershell
# Verify JEA session exists
Get-PSSessionConfiguration -Name "XingRanAgentJEA" | Select-Object Name, Permission

# Test JEA restricted endpoint
Invoke-Command -ConfigurationName "XingRanAgentJEA" -ComputerName localhost -ScriptBlock {
    Get-LocalUser
} -Credential (Get-Credential XingRanAgentUser)

# Verify user is NOT in Administrators group
Get-LocalGroupMember -Group "Administrators" | Where-Object {$_.Name -like "*XingRanAgentUser*"}
# Should return empty
```

2. **TLS Verification Test**:
```bash
# Build check
go build ./...

# Test with invalid certificate (should fail)
AGENT_TLS_VERIFY=true AGENT_TLS_CERT=/tmp/invalid.crt ./agent
# Expected: startup failure with certificate validation error

# Test with valid certificates
AGENT_TLS_ENABLED=true AGENT_TLS_CERT=/etc/xingran-agent/tls/server.crt \
  AGENT_TLS_KEY=/etc/xingran-agent/tls/server.key ./agent
# Expected: successful startup
```

3. **Error Message Sanitization Test**:
```bash
# Trigger various error conditions and verify generic responses
curl -X POST https://agent:8443/api/v1/accounts -d '{"invalid":"json"}'
# Expected: {"error": "请求参数无效"}

# Test internal errors don't leak details
curl -X POST https://agent:8443/api/v1/accounts -d '{"username":"test"}'
# If DB error: {"error": "请求处理失败，请稍后重试"}
# NOT: {"error": "database connection failed: connection to 192.168.1.100:5432 timeout"}
```

4. **Security Headers Test**:
```bash
curl -I https://agent:8443/api/v1/health
# Expected headers:
# X-Frame-Options: DENY
# X-Content-Type-Options: nosniff
# Content-Security-Policy: default-src 'self'
# Strict-Transport-Security: max-age=31536000; includeSubDomains
```

## Success Criteria

1. [ ] Windows Agent 使用 JEA 受限权限（非完全管理员）
2. [ ] JEA 角色仅允许用户管理命令
3. [ ] TLS 1.3 默认强制启用
4. [ ] 配置验证确保证书文件存在
5. [ ] InsecureSkipVerify 设置为 false（除非明确禁用验证）
6. [ ] 错误消息不包含敏感系统信息
7. [ ] 所有 HTTP 响应包含安全响应头
8. [ ] 代码编译通过，无警告
9. [ ] Windows JEA 配置在测试 VM 中验证
10. [ ] TLS 连接在测试环境中验证

## Output

After completion, create `.planning/phases/22b-vm-agent-service/22b-01-SUMMARY.md`
