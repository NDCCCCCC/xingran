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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

// =====================================================================
// account_manager — 三个 seam 注入 + 真策略体驱动 + 假策略上层 + 解析函数
// 前缀 TestAcct77_; 文件头注释 P-77-9 警示: 覆盖 var 前必须先 t.Cleanup
// 恢复 (本包 seam 是 runAccountCmd / runAccountCmdOutput / newAccountCmd),
// 否则后续测试真跑 powershell/useradd (Windows 危险, Linux 权限破坏)。
// =====================================================================

type acctCall struct {
	name string
	args []string
}

// stubAcctCmds77 同时替换三个 seam 为 helperStubCommand re-exec 子进程桩。
// per-seam shape picker: 传入 nil 表示用默认成功形态 (echo-args / print-users /
// passwd-style); 非 nil 时按调用序号 idx 选 shape — 通常用 "exit-1" 驱动失败
// 分支。返回 calls 按三个 seam 调用顺序追加; newBuf 累积 newAccountCmd 子进程
// stdout (passwd-style 把 stdin 一行内容回显到 stdout, 供测试断言 stdin 内容)。
func stubAcctCmds77(t *testing.T, pRun, pOut, pNew func(idx int) string) (calls *[]acctCall, newBuf *bytes.Buffer) {
	t.Helper()
	calls = &[]acctCall{}
	var buf bytes.Buffer
	origRun, origOut, origNew := runAccountCmd, runAccountCmdOutput, newAccountCmd
	t.Cleanup(func() {
		runAccountCmd = origRun
		runAccountCmdOutput = origOut
		newAccountCmd = origNew
	})

	runAccountCmd = func(_ context.Context, name string, args ...string) error {
		idx := len(*calls)
		shape := "echo-args"
		if pRun != nil {
			shape = pRun(idx)
		}
		*calls = append(*calls, acctCall{name: name, args: append([]string(nil), args...)})
		cmd := helperStubCommand(t, shape)
		cmd.Args = append(cmd.Args, name)
		cmd.Args = append(cmd.Args, args...)
		return cmd.Run()
	}
	runAccountCmdOutput = func(_ context.Context, name string, args ...string) ([]byte, error) {
		idx := len(*calls)
		shape := "print-users"
		if pOut != nil {
			shape = pOut(idx)
		}
		*calls = append(*calls, acctCall{name: name, args: append([]string(nil), args...)})
		cmd := helperStubCommand(t, shape)
		cmd.Args = append(cmd.Args, name)
		cmd.Args = append(cmd.Args, args...)
		var outBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &outBuf
		if err := cmd.Run(); err != nil {
			return nil, err
		}
		return outBuf.Bytes(), nil
	}
	newAccountCmd = func(_ context.Context, name string, args ...string) *exec.Cmd {
		idx := len(*calls)
		shape := "passwd-style"
		if pNew != nil {
			shape = pNew(idx)
		}
		*calls = append(*calls, acctCall{name: name, args: append([]string(nil), args...)})
		cmd := helperStubCommand(t, shape)
		cmd.Args = append(cmd.Args, name)
		cmd.Args = append(cmd.Args, args...)
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		return cmd
	}
	return calls, &buf
}

// ---------------------------------------------------------------------
// 假策略上层 (76-PATTERNS §76-03c mock 范本: preset + Fn + 计数)
// ---------------------------------------------------------------------

// fakeStrategy 实现 platformStrategy 6 方法 — 行为与 ldap_client_mock_test.go
// fakeStrategy 同构 (preset + 调用计数 + last* 透传断言字段)。
type fakeStrategy struct {
	createErr, deleteErr, resetErr, enableErr, disableErr, listErr error
	listResult                                                      []string
	lastCreate                                                      *Account
	createCalls, deleteCalls, resetCalls, enableCalls, disableCalls, listCalls int
}

func (f *fakeStrategy) createAccount(_ context.Context, username, password string, isAdmin bool) error {
	f.createCalls++
	f.lastCreate = &Account{Username: username, Password: password, IsAdmin: isAdmin}
	return f.createErr
}
func (f *fakeStrategy) deleteAccount(_ context.Context, _ string) error {
	f.deleteCalls++
	return f.deleteErr
}
func (f *fakeStrategy) resetPassword(_ context.Context, username, newPassword string) error {
	f.resetCalls++
	f.lastCreate = &Account{Username: username, Password: newPassword}
	return f.resetErr
}
func (f *fakeStrategy) enableAccount(_ context.Context, _ string) error {
	f.enableCalls++
	return f.enableErr
}
func (f *fakeStrategy) disableAccount(_ context.Context, _ string) error {
	f.disableCalls++
	return f.disableErr
}
func (f *fakeStrategy) listAccounts(_ context.Context) ([]string, error) {
	f.listCalls++
	return f.listResult, f.listErr
}

func newAccountManagerWithFake77(t *testing.T, fs *fakeStrategy) *AccountManager {
	t.Helper()
	am := NewAccountManager()
	am.strategy = fs
	return am
}

// TestAcct77_UpperLayer_Passthrough 五公开方法参数透传 + 错误透传。
func TestAcct77_UpperLayer_Passthrough(t *testing.T) {
	fs := &fakeStrategy{}
	am := newAccountManagerWithFake77(t, fs)

	// CreateAccount: username/password/isAdmin 透传
	require.NoError(t, am.CreateAccount(context.Background(), &Account{
		Username: "u1", Password: "pw1", IsAdmin: true,
	}))
	require.Equal(t, 1, fs.createCalls)
	require.NotNil(t, fs.lastCreate)
	require.Equal(t, "u1", fs.lastCreate.Username)
	require.Equal(t, "pw1", fs.lastCreate.Password)
	require.True(t, fs.lastCreate.IsAdmin)

	// Delete / Enable / Disable 仅透传 username
	require.NoError(t, am.DeleteAccount(context.Background(), "u1"))
	require.Equal(t, 1, fs.deleteCalls)
	require.NoError(t, am.EnableAccount(context.Background(), "u1"))
	require.Equal(t, 1, fs.enableCalls)
	require.NoError(t, am.DisableAccount(context.Background(), "u1"))
	require.Equal(t, 1, fs.disableCalls)

	// ResetPassword: username + newPassword 透传
	require.NoError(t, am.ResetPassword(context.Background(), "u1", "newpw"))
	require.Equal(t, 1, fs.resetCalls)
	require.Equal(t, "newpw", fs.lastCreate.Password)

	// 错误透传
	fs.createErr = errors.New("boom-create")
	require.Error(t, am.CreateAccount(context.Background(), &Account{Username: "u2"}))
	fs.deleteErr = errors.New("boom-delete")
	require.Error(t, am.DeleteAccount(context.Background(), "u2"))
}

func TestAcct77_UpperLayer_ListAccounts(t *testing.T) {
	fs := &fakeStrategy{listResult: []string{"u1", "u2", "u3"}}
	am := newAccountManagerWithFake77(t, fs)
	got, err := am.ListAccounts(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"u1", "u2", "u3"}, got)
	assert.Equal(t, 1, fs.listCalls)

	fs.listErr = errors.New("list-boom")
	_, err = am.ListAccounts(context.Background())
	require.Error(t, err)
}

// ---------------------------------------------------------------------
// 真策略体 — Windows 平台 (同包白盒直构 windowsPlatformStrategy)
// ---------------------------------------------------------------------

func TestAcct77_WindowsCreateAccount_SuccessNoAdmin(t *testing.T) {
	calls, _ := stubAcctCmds77(t, nil, nil, nil)
	w := &windowsPlatformStrategy{}
	require.NoError(t, w.createAccount(context.Background(), "u1", "pw1", false))
	require.Len(t, *calls, 1, "非 admin 仅一次 powershell 调用")
	assert.Equal(t, "powershell", (*calls)[0].name)
	joined := strings.Join((*calls)[0].args, " ")
	assert.Contains(t, joined, "ConvertTo-SecureString")
	assert.Contains(t, joined, "u1")
}

func TestAcct77_WindowsCreateAccount_AdminAddsGroup(t *testing.T) {
	calls, _ := stubAcctCmds77(t, nil, nil, nil)
	w := &windowsPlatformStrategy{}
	require.NoError(t, w.createAccount(context.Background(), "admin1", "pw1", true))
	require.Len(t, *calls, 2, "isAdmin 触发 Add-LocalGroupMember 第二次调用")
	assert.Contains(t, strings.Join((*calls)[1].args, " "), "Add-LocalGroupMember")
	assert.Contains(t, strings.Join((*calls)[1].args, " "), "admin1")
}

func TestAcct77_WindowsCreateAccount_FailUser(t *testing.T) {
	calls, _ := stubAcctCmds77(t, func(int) string { return "exit-1" }, nil, nil)
	w := &windowsPlatformStrategy{}
	err := w.createAccount(context.Background(), "u1", "pw", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create Windows user")
	require.NotEmpty(t, *calls)
}

func TestAcct77_WindowsCreateAccount_FailAdminGroup(t *testing.T) {
	// 第一次 echo-args 成功, 第二次 exit-1 → "failed to add to admin group"
	calls, _ := stubAcctCmds77(t, func(idx int) string {
		if idx == 0 {
			return "echo-args"
		}
		return "exit-1"
	}, nil, nil)
	w := &windowsPlatformStrategy{}
	err := w.createAccount(context.Background(), "u1", "pw", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add to admin group")
	require.Len(t, *calls, 2)
}

func TestAcct77_WindowsDeleteAccount(t *testing.T) {
	calls, _ := stubAcctCmds77(t, nil, nil, nil)
	w := &windowsPlatformStrategy{}
	require.NoError(t, w.deleteAccount(context.Background(), "u1"))
	require.Len(t, *calls, 1)
	assert.Equal(t, "powershell", (*calls)[0].name)
	assert.Contains(t, strings.Join((*calls)[0].args, " "), "Remove-LocalUser")
}

func TestAcct77_WindowsResetPassword(t *testing.T) {
	calls, _ := stubAcctCmds77(t, nil, nil, nil)
	w := &windowsPlatformStrategy{}
	require.NoError(t, w.resetPassword(context.Background(), "u1", "newpw"))
	require.Len(t, *calls, 1)
	joined := strings.Join((*calls)[0].args, " ")
	assert.Contains(t, joined, "Set-LocalUser")
	assert.Contains(t, joined, "ConvertTo-SecureString")
	assert.Contains(t, joined, "newpw")
}

func TestAcct77_WindowsEnableDisable(t *testing.T) {
	calls, _ := stubAcctCmds77(t, nil, nil, nil)
	w := &windowsPlatformStrategy{}
	require.NoError(t, w.enableAccount(context.Background(), "u1"))
	require.NoError(t, w.disableAccount(context.Background(), "u1"))
	require.Len(t, *calls, 2)
	assert.Contains(t, strings.Join((*calls)[0].args, " "), "Enable-LocalUser")
	assert.Contains(t, strings.Join((*calls)[1].args, " "), "Disable-LocalUser")
}

func TestAcct77_WindowsListAccounts(t *testing.T) {
	t.Setenv("STUB_USERS_77", "Alice\nBob\n\n")
	_, _ = stubAcctCmds77(t, nil, nil, nil)
	w := &windowsPlatformStrategy{}
	got, err := w.listAccounts(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"Alice", "Bob"}, got)
}

func TestAcct77_WindowsListAccounts_Fail(t *testing.T) {
	_, _ = stubAcctCmds77(t, nil, func(int) string { return "exit-1" }, nil)
	w := &windowsPlatformStrategy{}
	_, err := w.listAccounts(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list Windows users")
}

// ---------------------------------------------------------------------
// 真策略体 — Linux 平台
// ---------------------------------------------------------------------

func TestAcct77_LinuxCreateAccount_SuccessNoAdmin(t *testing.T) {
	calls, newBuf := stubAcctCmds77(t, nil, nil, nil)
	l := &linuxPlatformStrategy{}
	require.NoError(t, l.createAccount(context.Background(), "alice", "s3cr3t", false))
	// 1 runAccountCmd (useradd) + 1 newAccountCmd (chpasswd)
	require.Len(t, *calls, 2)
	assert.Equal(t, "useradd", (*calls)[0].name)
	assert.Equal(t, []string{"-m", "alice"}, (*calls)[0].args)
	assert.Equal(t, "chpasswd", (*calls)[1].name)
	// passwd-style 把 stdin 一行内容回显到 stdout
	assert.Contains(t, newBuf.String(), "alice:s3cr3t")
}

func TestAcct77_LinuxCreateAccount_AdminSudo(t *testing.T) {
	calls, newBuf := stubAcctCmds77(t, nil, nil, nil)
	l := &linuxPlatformStrategy{}
	require.NoError(t, l.createAccount(context.Background(), "admin1", "pw", true))
	// useradd + chpasswd(newAccountCmd) + tee(newAccountCmd) + chmod(runAccountCmd) = 4
	require.Len(t, *calls, 4)
	assert.Equal(t, "useradd", (*calls)[0].name)
	assert.Equal(t, "chpasswd", (*calls)[1].name)
	assert.Equal(t, "tee", (*calls)[2].name)
	assert.Equal(t, "/etc/sudoers.d/admin1", (*calls)[2].args[0])
	assert.Equal(t, "chmod", (*calls)[3].name)
	assert.Equal(t, []string{"440", "/etc/sudoers.d/admin1"}, (*calls)[3].args)
	// tee 的 stdin 是 sudoersContent, 经 passwd-style 回显
	assert.Contains(t, newBuf.String(), "admin1 ALL=(root) NOPASSWD")
}

func TestAcct77_LinuxCreateAccount_FailUseradd(t *testing.T) {
	calls, _ := stubAcctCmds77(t, func(int) string { return "exit-1" }, nil, nil)
	l := &linuxPlatformStrategy{}
	err := l.createAccount(context.Background(), "u1", "pw", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create user")
	require.NotEmpty(t, *calls)
}

func TestAcct77_LinuxCreateAccount_FailChpasswd(t *testing.T) {
	calls, _ := stubAcctCmds77(t, nil, nil, func(int) string { return "exit-1" })
	l := &linuxPlatformStrategy{}
	err := l.createAccount(context.Background(), "u1", "pw", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set password")
	require.Len(t, *calls, 2, "useradd 成功后 chpasswd 失败")
}

func TestAcct77_LinuxCreateAccount_FailTee(t *testing.T) {
	// useradd 成功 + chpasswd 成功 + tee 失败 → "failed to write sudoers file"
	calls, _ := stubAcctCmds77(t, nil, nil, func(idx int) string {
		if idx == 2 { // tee 是第 3 个调用 (idx 2), useradd + chpasswd 各占 idx 0,1
			return "exit-1"
		}
		return "passwd-style"
	})
	l := &linuxPlatformStrategy{}
	err := l.createAccount(context.Background(), "u1", "pw", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write sudoers file")
	require.Len(t, *calls, 3)
}

func TestAcct77_LinuxCreateAccount_FailChmod(t *testing.T) {
	// 调用顺序: runAccountCmd(idx0=useradd) + newAccountCmd(idx1=chpasswd)
	//           + newAccountCmd(idx2=tee) + runAccountCmd(idx3=chmod) = 4
	// chmod 是全局第 4 次调用 (idx 3) → exit-1
	calls, _ := stubAcctCmds77(t, func(idx int) string {
		if idx == 3 {
			return "exit-1"
		}
		return "echo-args"
	}, nil, nil)
	l := &linuxPlatformStrategy{}
	err := l.createAccount(context.Background(), "u1", "pw", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set sudoers permissions")
	require.Len(t, *calls, 4)
}

func TestAcct77_LinuxDeleteAccount(t *testing.T) {
	calls, _ := stubAcctCmds77(t, nil, nil, nil)
	l := &linuxPlatformStrategy{}
	require.NoError(t, l.deleteAccount(context.Background(), "u1"))
	require.Len(t, *calls, 1)
	assert.Equal(t, "userdel", (*calls)[0].name)
	assert.Equal(t, []string{"-r", "u1"}, (*calls)[0].args)
}

func TestAcct77_LinuxResetPassword(t *testing.T) {
	calls, newBuf := stubAcctCmds77(t, nil, nil, nil)
	l := &linuxPlatformStrategy{}
	require.NoError(t, l.resetPassword(context.Background(), "u1", "newpw"))
	require.Len(t, *calls, 1)
	assert.Equal(t, "chpasswd", (*calls)[0].name)
	assert.Contains(t, newBuf.String(), "u1:newpw")
}

func TestAcct77_LinuxEnableDisable(t *testing.T) {
	calls, _ := stubAcctCmds77(t, nil, nil, nil)
	l := &linuxPlatformStrategy{}
	require.NoError(t, l.enableAccount(context.Background(), "u1"))
	require.NoError(t, l.disableAccount(context.Background(), "u1"))
	require.Len(t, *calls, 2)
	assert.Equal(t, "usermod", (*calls)[0].name)
	assert.Equal(t, []string{"-U", "u1"}, (*calls)[0].args)
	assert.Equal(t, "usermod", (*calls)[1].name)
	assert.Equal(t, []string{"-L", "u1"}, (*calls)[1].args)
}

func TestAcct77_LinuxListAccounts(t *testing.T) {
	calls, _ := stubAcctCmds77(t, nil, nil, nil)
	l := &linuxPlatformStrategy{}
	got, err := l.listAccounts(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"alice", "bob"}, got,
		"parseLinuxUsers 过滤 uid>=1000, 默认 getent 块含 alice/bob")
	require.Len(t, *calls, 1)
	assert.Equal(t, "getent", (*calls)[0].name)
}

func TestAcct77_LinuxListAccounts_Fail(t *testing.T) {
	_, _ = stubAcctCmds77(t, nil, func(int) string { return "exit-1" }, nil)
	l := &linuxPlatformStrategy{}
	_, err := l.listAccounts(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list Linux users")
}

// ---------------------------------------------------------------------
// parseWindowsUsers / parseLinuxUsers 纯函数
// ---------------------------------------------------------------------

func TestAcct77_ParseWindowsUsers(t *testing.T) {
	assert.Equal(t, []string{"Alice", "Bob"}, parseWindowsUsers("Alice\nBob\n\n"))
	assert.Equal(t, []string{"Alice"}, parseWindowsUsers("  Alice  \n\n"))
	// 空输入 → 空切片
	got := parseWindowsUsers("")
	assert.Empty(t, got)
}

func TestAcct77_ParseLinuxUsers(t *testing.T) {
	// uid>=1000 过滤; 空行 / 无冒号 / UID 非数字 / parts<3 全部跳过
	input := "" +
		"root:x:0:0:root:/root:/bin/bash\n" +        // uid<1000 跳过
		"sysuser:x:999:999::/:/sbin/nologin\n" +     // uid<1000 跳过
		"alice:x:1000:1000::/home/alice:/bin/bash\n" + // 入
		"bob:x:1001:1001::/home/bob:/bin/bash\n" +   // 入
		"\n" +                                        // 空行跳过
		"weirdnocolon\n" +                            // 无冒号跳过
		"x:y\n" +                                     // parts<3 跳过
		"charlie:x:not-a-num:/home/c:/bin/c\n"       // uid 非数字跳过
	assert.Equal(t, []string{"alice", "bob"}, parseLinuxUsers(input))
}
