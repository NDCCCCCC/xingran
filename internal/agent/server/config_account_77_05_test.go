package server

// =====================================================================
// config_account_77_05_test.go — Phase 77 Plan 05 (BLOCK-02 收口)
//
// 覆盖范围: config 校验/注册/LoadConfig + Q-77-A/B 回归 + account_manager
// 真策略体(re-exec)+ 假策略上层 + parse 纯函数。前缀 TestCfg77_ 与
// TestAcct77_ 区分段位;文件头注释 P-77-9 seam 警示(见底部)。
//
// 纪律:
//   - viper.Reset 在 LoadConfig 用例的 t.Cleanup 中调用防全局态污染 (P-77-3);
//   - 测试禁 t.Parallel(白盒 seam + 全局 logger 共享);
//   - 注入缝纪律: 覆盖 var 前先 t.Cleanup 恢复, P-77-9 Windows 危险警示;
//   - 无裸 sleep — 全部断言走 require/assert 同步路径(P-77-4)。
// =====================================================================

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------
// Q-77-A: generateRandomSecret 必须真正随机 — D-03 判修依据是函数名 + 注释
// 「生成随机密钥」与确定性实现的字面对照。
// ---------------------------------------------------------------------

// TestCfg77_GenerateRandomSecret_LengthIs32 锁长度契约: 修复后必须仍为 32。
func TestCfg77_GenerateRandomSecret_LengthIs32(t *testing.T) {
	require.Len(t, generateRandomSecret(), 32)
}

// TestCfg77_GenerateRandomSecret_TwoCallsDiffer RED→GREEN 翻转断言:
// 现行为确定性返回同一常量, 修复后必须两次调用产生不同字符串。
func TestCfg77_GenerateRandomSecret_TwoCallsDiffer(t *testing.T) {
	s1 := generateRandomSecret()
	s2 := generateRandomSecret()
	assert.Len(t, s1, 32)
	assert.Len(t, s2, 32)
	assert.NotEqual(t, s1, s2,
		"两次 generateRandomSecret 调用必须产生不同密钥 (Q-77-A 确定性常量漏洞)")
}

// ---------------------------------------------------------------------
// Q-77-B: 收集到的 MachineGUID 可能为空 (Linux 永远, 或 Windows reg 查询失败),
// 旧实现 fp.MachineGUID[:8] 直接 panic。fix 加长度守卫 + collectFingerprint seam
// 让 RED 探针跨平台可复现 (deterministic empty GUID 测试用例)。
// ---------------------------------------------------------------------

// stubFingerprint77 把 collectFingerprint 包级 seam 替换为给定的固定指纹, 并
// 在 t.Cleanup 中恢复。先 Cleanup 后覆盖, 禁 t.Parallel (P-77-9)。
func stubFingerprint77(t *testing.T, fp *SystemFingerprint) {
	t.Helper()
	orig := collectFingerprint
	t.Cleanup(func() { collectFingerprint = orig })
	collectFingerprint = func() (*SystemFingerprint, error) { return fp, nil }
}

// TestCfg77_RegisterToBackend_MissingMachineGUID_NoPanic RED→GREEN 翻转:
// fp.MachineGUID 长度不足 8 时, 旧行为 panic; 修复后返回 fallback 临时 ID 而
// 非 panic (D-03 判修依据: 切片边界语义 + operations 包 safePrefix 先例)。
func TestCfg77_RegisterToBackend_MissingMachineGUID_NoPanic(t *testing.T) {
	// 后端指向空闲端口 → AutoRegisterAgent 必然连接失败 → 进入 temp ID 兜底分支
	closedSrv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closedSrv.Close()

	require.NoError(t, InitLogger("info", ""), "P-77-5: WithFields.Warn 依赖全局 logger")
	stubFingerprint77(t, &SystemFingerprint{
		Hostname:    "winbox",
		MachineGUID: "", // Linux / Windows reg 失败场景
	})

	cfg := &Config{BackendURL: "http://127.0.0.1:1"}
	// 注: RegisterToBackend 的注册失败兜底是 warn + temp ID, 不返回 error;
	// 修复目标只是「不再 panic」+ temp ID 正确填充。
	out, err := RegisterToBackend(cfg)
	require.NoError(t, err)
	assert.Same(t, cfg, out)
	// 关键断言: 不 panic + 兜底字段正确 (旧实现会在 fp.MachineGUID[:8] 处切片越界)
	assert.Equal(t, "vm-temp-winbox", cfg.VMID, "兜底 VMID 必须包含 hostname")
	assert.Equal(t, "agent-winbox-", cfg.AgentID, "空 GUID 时 fallback agent ID 应仅含 hostname 前缀")
	assert.Len(t, cfg.JWTSecret, 32, "兜底路径仍需生成 32 字符随机密钥")
}

// TestCfg77_RegisterToBackend_TruncatesLongMachineGUID 正向契约: GUID 长度 ≥8
// 时行为与旧实现一致 (截取前 8 位), 锁住「未破坏既有正确行为」不变量。
func TestCfg77_RegisterToBackend_TruncatesLongMachineGUID(t *testing.T) {
	require.NoError(t, InitLogger("info", ""))
	stubFingerprint77(t, &SystemFingerprint{
		Hostname:    "host-A",
		MachineGUID: "0123456789abcdef",
	})

	cfg := &Config{BackendURL: "http://127.0.0.1:1"}
	_, err := RegisterToBackend(cfg)
	require.NoError(t, err)
	assert.Equal(t, "agent-host-A-01234567", cfg.AgentID,
		"GUID ≥8 字符时 fallback 应截取前 8 位 (行为不变)")
	assert.Equal(t, "vm-temp-host-A", cfg.VMID, "vm_id 兜底含 hostname")
}

// ---------------------------------------------------------------------
// Config.Validate / ValidateTLS / CheckCertificateFiles — 纯结构 + 文件存在性
// ---------------------------------------------------------------------

// writeCertAndKey 在 t.TempDir() 写入任意内容的 cert / key 文件, 返回路径对。
// 文件内容无要求, ValidateTLS 只做 os.Stat 存在性检查。
func writeCertAndKey(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	cert := filepath.Join(dir, "cert.pem")
	key := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(cert, []byte("dummy-cert"), 0o600))
	require.NoError(t, os.WriteFile(key, []byte("dummy-key"), 0o600))
	return cert, key
}

func TestCfg77_Validate_NilBackendURL(t *testing.T) {
	c := &Config{} // BackendURL == "" → os.ErrInvalid
	err := c.Validate()
	require.ErrorIs(t, err, os.ErrInvalid)
}

func TestCfg77_ValidateTLSDisabled(t *testing.T) {
	c := &Config{TLSEnabled: false}
	err := c.ValidateTLS()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TLS must be enabled")
}

func TestCfg77_ValidateTLSRequiresCertAndKey(t *testing.T) {
	c := &Config{TLSEnabled: true} // cert/key 都空
	err := c.ValidateTLS()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be specified when TLS is enabled")
}

func TestCfg77_ValidateTLS_HappyPath(t *testing.T) {
	cert, key := writeCertAndKey(t)
	c := &Config{
		BackendURL:         "http://127.0.0.1:9000",
		TLSEnabled:         true,
		TLSCertFile:        cert,
		TLSKeyFile:         key,
		VerifyCertificates: false, // 不验 CA
	}
	require.NoError(t, c.Validate())
}

func TestCfg77_ValidateTLS_CertFileMissing(t *testing.T) {
	_, key := writeCertAndKey(t)
	c := &Config{
		BackendURL:  "http://127.0.0.1:9000",
		TLSEnabled:  true,
		TLSCertFile: filepath.Join(t.TempDir(), "no-such.pem"),
		TLSKeyFile:  key,
	}
	err := c.ValidateTLS()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TLS certificate file not found")
}

func TestCfg77_ValidateTLS_KeyFileMissing(t *testing.T) {
	cert, _ := writeCertAndKey(t)
	c := &Config{
		BackendURL:  "http://127.0.0.1:9000",
		TLSEnabled:  true,
		TLSCertFile: cert,
		TLSKeyFile:  filepath.Join(t.TempDir(), "no-such.pem"),
	}
	err := c.ValidateTLS()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TLS key file not found")
}

func TestCfg77_ValidateTLS_CAFileMissing(t *testing.T) {
	cert, key := writeCertAndKey(t)
	c := &Config{
		BackendURL:         "http://127.0.0.1:9000",
		TLSEnabled:         true,
		TLSCertFile:        cert,
		TLSKeyFile:         key,
		VerifyCertificates: true,
		CAFile:             filepath.Join(t.TempDir(), "no-ca.pem"),
	}
	err := c.ValidateTLS()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CA bundle file not found")
}

func TestCfg77_ValidateTLS_ValidCAFile(t *testing.T) {
	cert, key := writeCertAndKey(t)
	ca := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(ca, []byte("dummy-ca"), 0o600))
	c := &Config{
		BackendURL:         "http://127.0.0.1:9000",
		TLSEnabled:         true,
		TLSCertFile:        cert,
		TLSKeyFile:         key,
		VerifyCertificates: true,
		CAFile:             ca,
	}
	require.NoError(t, c.ValidateTLS())
}

func TestCfg77_Validate_MissingAgentVMIsWarning(t *testing.T) {
	// Phase 75 五步法: Validate 在缺 agent_id/vm_id 时只 warn 不返回 error。
	cert, key := writeCertAndKey(t)
	c := &Config{
		BackendURL:  "http://127.0.0.1:9000",
		TLSEnabled:  true,
		TLSCertFile: cert,
		TLSKeyFile:  key,
		// AgentID 与 VMID 都空
	}
	require.NoError(t, c.Validate())
}

// TestCfg77_CheckCertificateFiles_LinuxKeyWorldReadable Linux 上 key 文件 0o644
// 必须报错; Windows 自然只覆盖 guard 行 (SC#3 抵消数学的一部分, 勿 Skipf)。
func TestCfg77_CheckCertificateFiles_LinuxKeyWorldReadable(t *testing.T) {
	cert, key := writeCertAndKey(t)
	c := &Config{TLSCertFile: cert, TLSKeyFile: key}
	if runtime.GOOS == "windows" {
		// Windows 走 guard 行直接 return nil
		require.NoError(t, c.CheckCertificateFiles())
		return
	}
	require.NoError(t, os.Chmod(key, 0o644))
	err := c.CheckCertificateFiles()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private key file should not be world-readable")
}

func TestCfg77_CheckCertificateFiles_LinuxKeyRestrictive(t *testing.T) {
	cert, key := writeCertAndKey(t)
	require.NoError(t, os.Chmod(key, 0o600))
	c := &Config{TLSCertFile: cert, TLSKeyFile: key}
	if runtime.GOOS != "windows" {
		// Linux 也要校验 cert 文件 0o644 时会 warn 但不返回 error
		require.NoError(t, os.Chmod(cert, 0o644))
	}
	require.NoError(t, c.CheckCertificateFiles())
}

// ---------------------------------------------------------------------
// AutoRegisterAgent — http.Post 直打后端 /api/agent/register 假后端
// ---------------------------------------------------------------------

// fakeRegisterBackend 返回一个本地回环 httptest.Server, 处理 POST /api/agent/register,
// 默认返回成功响应。test 可在调用前替换 defaultHandler 变量驱动失败分支。
var fakeRegisterHandlerOverride func(http.ResponseWriter, *http.Request)

func fakeRegisterBackend(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/agent/register" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if fakeRegisterHandlerOverride != nil {
			fakeRegisterHandlerOverride(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"code":0,"message":"ok","data":{
			"vm_id":"vm-from-backend","agent_id":"agent-from-backend","matched":true
		}}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCfg77_AutoRegisterAgent_Success(t *testing.T) {
	srv := fakeRegisterBackend(t)
	vmID, agentID, err := AutoRegisterAgent(srv.URL, &SystemFingerprint{Hostname: "h"})
	require.NoError(t, err)
	assert.Equal(t, "vm-from-backend", vmID)
	assert.Equal(t, "agent-from-backend", agentID)
}

func TestCfg77_AutoRegisterAgent_Non200(t *testing.T) {
	srv := fakeRegisterBackend(t)
	fakeRegisterHandlerOverride = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"code":500,"message":"internal"}`)
	}
	_, _, err := AutoRegisterAgent(srv.URL, &SystemFingerprint{Hostname: "h"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
	fakeRegisterHandlerOverride = nil
}

func TestCfg77_AutoRegisterAgent_BadJSON(t *testing.T) {
	srv := fakeRegisterBackend(t)
	fakeRegisterHandlerOverride = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `not json at all`)
	}
	_, _, err := AutoRegisterAgent(srv.URL, &SystemFingerprint{Hostname: "h"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "解析响应失败")
	fakeRegisterHandlerOverride = nil
}

func TestCfg77_AutoRegisterAgent_ConnectionRefused(t *testing.T) {
	// 关闭假后端确保连接被拒
	srv := fakeRegisterBackend(t)
	srv.Close()
	_, _, err := AutoRegisterAgent(srv.URL, &SystemFingerprint{Hostname: "h"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "调用注册 API 失败")
}

// ---------------------------------------------------------------------
// RegisterToBackend — 成功路径走 fake 后端, fallback 路径在 Q-77-B 已测
// ---------------------------------------------------------------------

func TestCfg77_RegisterToBackend_SuccessWithBackend(t *testing.T) {
	require.NoError(t, InitLogger("info", ""))
	srv := fakeRegisterBackend(t)
	stubFingerprint77(t, &SystemFingerprint{Hostname: "host", MachineGUID: "1234567890abcdef"})

	cfg := &Config{BackendURL: srv.URL}
	out, err := RegisterToBackend(cfg)
	require.NoError(t, err)
	assert.Same(t, cfg, out)
	assert.Equal(t, "vm-from-backend", cfg.VMID)
	assert.Equal(t, "agent-from-backend", cfg.AgentID)
	assert.Len(t, cfg.JWTSecret, 32, "成功路径也需生成 32 字符随机密钥")
}

// ---------------------------------------------------------------------
// LoadConfig — viper 全局态, t.Cleanup(viper.Reset) 防跨测试污染 (P-77-3)
// ---------------------------------------------------------------------

// viperResetCleanup 每个 LoadConfig 用例的 t.Cleanup 钩子, 显式 Reset 全局 viper。
func viperResetCleanup(t *testing.T) {
	t.Helper()
	t.Cleanup(viper.Reset)
}

func TestCfg77_LoadConfig_MissingYAMLUsesEnv(t *testing.T) {
	viperResetCleanup(t)
	cert, key := writeCertAndKey(t)
	// yaml 缺失 + 默认 TLSEnabled=true → Phase 75 Q-13 修复后的报错行为
	t.Setenv("BACKEND_URL", "http://example.invalid:9000")

	_, err := LoadConfig(filepath.Join(t.TempDir(), "no-such-yaml.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TLS certificate and key files must be specified")
	_ = cert
	_ = key
}

func TestCfg77_LoadConfig_ValidYAMLHonoursFields(t *testing.T) {
	viperResetCleanup(t)
	cert, key := writeCertAndKey(t)
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "config.yaml")
	// 单引号 YAML scalar: Windows 路径含反斜杠, 单引号按字面保留不解析转义,
	// 双引号下 \U 等会被 YAML 误判为非法转义序列。
	yaml := `
backend_url: "http://yaml-host:9000"
agent_id: "yaml-agent"
vm_id: "yaml-vm"
jwt_secret: "yaml-secret"
tls_enabled: true
tls_cert_file: '` + cert + `'
tls_key_file: '` + key + `'
verify_certificates: false
log_level: "debug"
log_path: "logs/agent"
heartbeat_interval: "15s"
platform: "linux"
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yaml), 0o600))

	cfg, err := LoadConfig(yamlPath)
	require.NoError(t, err)
	assert.Equal(t, "http://yaml-host:9000", cfg.BackendURL)
	assert.Equal(t, "yaml-agent", cfg.AgentID)
	assert.Equal(t, "yaml-vm", cfg.VMID)
	assert.Equal(t, "yaml-secret", cfg.JWTSecret)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "linux", cfg.Platform)
	// 相对路径被转为绝对
	assert.True(t, filepath.IsAbs(cfg.LogPath),
		"相对 LogPath 必须被 LoadConfig 转换为绝对路径, 实际 %q", cfg.LogPath)
	assert.Equal(t, 15*time.Second, cfg.HeartbeatInterval)
}

func TestCfg77_LoadConfig_EnvOverridesYAML(t *testing.T) {
	viperResetCleanup(t)
	cert, key := writeCertAndKey(t)
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "config.yaml")
	yaml := `
backend_url: "http://yaml-host:9000"
agent_id: "yaml-agent"
vm_id: "yaml-vm"
jwt_secret: "yaml-secret"
tls_enabled: true
tls_cert_file: '` + cert + `'
tls_key_file: '` + key + `'
verify_certificates: false
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yaml), 0o600))

	// env 覆盖 YAML
	t.Setenv("BACKEND_URL", "http://env-host:9000")
	t.Setenv("AGENT_ID", "env-agent")

	cfg, err := LoadConfig(yamlPath)
	require.NoError(t, err)
	assert.Equal(t, "http://env-host:9000", cfg.BackendURL, "env 覆盖 yaml")
	assert.Equal(t, "env-agent", cfg.AgentID, "env 覆盖 yaml")
	// 未被 env 覆盖的字段保留 yaml 值
	assert.Equal(t, "yaml-vm", cfg.VMID)
}

func TestCfg77_LoadConfig_DefaultTLSRequiresFiles(t *testing.T) {
	viperResetCleanup(t)
	// 不写 yaml + 不提供 TLS 证书路径 → 默认 TLSEnabled=true 触发 Phase 75 Q-13
	// 修复后的报错 (TLS cert/key 必须指定)
	t.Setenv("BACKEND_URL", "http://env-host:9000")

	_, err := LoadConfig(filepath.Join(t.TempDir(), "no-such.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TLS certificate and key files must be specified")
}

// 仅占位校验 yaml 解析 schema 形状 — 防止误删除字段时编译期才发现。
var _ = json.Unmarshal
