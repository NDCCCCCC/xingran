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
	"net/http"
	"net/http/httptest"
	"testing"

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
