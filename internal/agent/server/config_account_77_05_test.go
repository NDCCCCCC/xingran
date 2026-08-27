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
