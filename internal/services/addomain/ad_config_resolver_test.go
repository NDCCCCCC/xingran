//go:build !skip_db_tests
// +build !skip_db_tests

package addomain

// Phase 58 Wave 1 —— D-02 最长后缀唯一匹配契约 + D-03 双入口配置 helper 测试。
//
// 本文件引用 ad_push_queue_drain_test.go 的 setupDrainTestDB / seedDrainADConfig
// （该文件带 `!skip_db_tests` tag），故本文件同样携带该 tag——否则
// `go test -tags skip_db_tests ./internal/services/addomain/` 会因 undefined 编译失败。
//
// 命名一律 `TestPhase58_*`。

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// D-02 最长后缀唯一匹配（纯函数，不连 DB）
// ============================================================================

// TestPhase58_MatchADConfigByDN_LongestSuffixWins 契约主用例：
// 父域与子域同时命中时，必须取 base_dn **最长**的子域（不是切片第一条）。
func TestPhase58_MatchADConfigByDN_LongestSuffixWins(t *testing.T) {
	configs := []models.ADConfig{
		{BaseModel: models.BaseModel{ID: "corp"}, Status: models.ADConfigStatusEnabled, BaseDN: "DC=corp,DC=com"},
		{BaseModel: models.BaseModel{ID: "na"}, Status: models.ADConfigStatusEnabled, BaseDN: "DC=na,DC=corp,DC=com"},
	}
	dn := "CN=zhangsan,OU=IT,DC=na,DC=corp,DC=com"

	cfg, err := MatchADConfigByDN(dn, configs)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "na", cfg.ID, "父子域共享后缀必须命中更长（子域）的 base_dn")
}

// TestPhase58_MatchADConfigByDN_ParentChildSharedSuffix CONTEXT 点名的反例：
// 字符串级 ends-with 会误命中父域（陷阱确实存在），本实现必须避开它。
// 两件事都锁进同一个测试。
func TestPhase58_MatchADConfigByDN_ParentChildSharedSuffix(t *testing.T) {
	normDn := NormalizeDN("CN=zhangsan,OU=IT,DC=na,DC=corp,DC=com")

	// ① 陷阱成立：裸后缀匹配确实会把子域 DN 判成"属于父域"
	assert.True(t, strings.HasSuffix(normDn, "dc=corp,dc=com"),
		"裸 HasSuffix 确实命中 —— 这就是 P6 必须带逗号边界的原因")

	// ② 被正确规避：MatchADConfigByDN 返回子域而不是父域
	configs := []models.ADConfig{
		{BaseModel: models.BaseModel{ID: "corp"}, Status: models.ADConfigStatusEnabled, BaseDN: "DC=corp,DC=com"},
		{BaseModel: models.BaseModel{ID: "na"}, Status: models.ADConfigStatusEnabled, BaseDN: "DC=na,DC=corp,DC=com"},
	}
	cfg, err := MatchADConfigByDN("CN=zhangsan,OU=IT,DC=na,DC=corp,DC=com", configs)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "na", cfg.ID, "最长后缀匹配必须避开父域的裸后缀误命中")
}

// TestPhase58_MatchADConfigByDN_CommaBoundaryRejectsXcorp P6 逗号边界反例：
// DC=xcorp,DC=com 不得被 DC=corp,DC=com 命中。
func TestPhase58_MatchADConfigByDN_CommaBoundaryRejectsXcorp(t *testing.T) {
	configs := []models.ADConfig{
		{BaseModel: models.BaseModel{ID: "corp"}, Status: models.ADConfigStatusEnabled, BaseDN: "DC=corp,DC=com"},
	}
	cfg, err := MatchADConfigByDN("CN=a,DC=xcorp,DC=com", configs)
	assert.Error(t, err, "xcorp 不得被 corp 命中")
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "无匹配 AD 配置")
}

// TestPhase58_MatchADConfigByDN_AmbiguousEqualsLength 等长并列 → 歧义 → error。
//
// ⚠ 触发歧义的真实形态：两条启用配置归一后 base_dn **相同**（同一域被重复配置）。
// 数学上，两个"等长但不同"的字符串不可能同时是同一字符串的逗号边界后缀
// （若 ",X" 与 ",Y" 长度相同且都是后缀，则 X == Y），故等长并列只有"重复配置"一种形态。
func TestPhase58_MatchADConfigByDN_AmbiguousEqualsLength(t *testing.T) {
	configs := []models.ADConfig{
		{BaseModel: models.BaseModel{ID: "a"}, Status: models.ADConfigStatusEnabled, BaseDN: "DC=corp,DC=com"},
		{BaseModel: models.BaseModel{ID: "b"}, Status: models.ADConfigStatusEnabled, BaseDN: "dc=CORP,dc=COM"},
	}
	cfg, err := MatchADConfigByDN("CN=u,OU=IT,DC=corp,DC=com", configs)
	assert.Error(t, err, "等长并列必须报歧义，禁止静默取第一条（T-58-02-05）")
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "AD 配置歧义")
}

// TestPhase58_MatchADConfigByDN_CaseAndSpaceInsensitive
// base_dn 侧的大小写与空格差异必须被吃掉（跨文件证据：NormalizeDN 值侧 ToLower）。
func TestPhase58_MatchADConfigByDN_CaseAndSpaceInsensitive(t *testing.T) {
	configs := []models.ADConfig{
		{BaseModel: models.BaseModel{ID: "corp"}, Status: models.ADConfigStatusEnabled, BaseDN: "  dc=CORP , dc=com "},
	}
	cfg, err := MatchADConfigByDN("cn=Zhang San, ou=IT, dc=corp, dc=com", configs)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "corp", cfg.ID,
		"base_dn 侧值与空格的差异必须归一；DN 侧多出的 ou=IT 前缀不影响逗号边界后缀判定")
}

// TestPhase58_MatchADConfigByDN_NoMatchIsError 三种"无匹配"都要报错且 cfg 为 nil。
func TestPhase58_MatchADConfigByDN_NoMatchIsError(t *testing.T) {
	dn := "CN=u,OU=IT,DC=corp,DC=com"

	// ① 空 configs
	cfg, err := MatchADConfigByDN(dn, nil)
	assert.Error(t, err)
	assert.Nil(t, cfg)

	// ② 全部停用
	cfg, err = MatchADConfigByDN(dn, []models.ADConfig{
		{BaseModel: models.BaseModel{ID: "off"}, Status: models.ADConfigStatusDisabled, BaseDN: "DC=corp,DC=com"},
	})
	assert.Error(t, err, "停用配置不得参与匹配")
	assert.Nil(t, cfg)

	// ③ 启用但 base_dn 全为空串（空串后缀恒命中，必须跳过 → 落到无匹配）
	cfg, err = MatchADConfigByDN(dn, []models.ADConfig{
		{BaseModel: models.BaseModel{ID: "empty"}, Status: models.ADConfigStatusEnabled, BaseDN: ""},
		{BaseModel: models.BaseModel{ID: "blank"}, Status: models.ADConfigStatusEnabled, BaseDN: "   "},
	})
	assert.Error(t, err, "空 base_dn 必须被跳过而不是恒命中（T-58-02-06）")
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "无匹配 AD 配置")
}

// TestPhase58_MatchADConfigByDN_ReadsBaseDnFromDB 走 GORM 读回（证明生产查询路径能用上
// F-F 补的 base_dn 列，而不是只能退化成裸 SQL 断言）。
func TestPhase58_MatchADConfigByDN_ReadsBaseDnFromDB(t *testing.T) {
	db := setupDrainTestDB(t)
	seedDrainADConfig(t, db, "cfg-1", "总部域", "DC=corp,DC=com", true, 0)

	var configs []models.ADConfig
	require.NoError(t, db.Where("status = ?", models.ADConfigStatusEnabled).Find(&configs).Error)
	require.Len(t, configs, 1)

	cfg, err := MatchADConfigByDN("CN=zhangsan,OU=IT,DC=corp,DC=com", configs)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "cfg-1", cfg.ID, "base_dn 必须能经 GORM 读回并参与最长后缀匹配")
}

// ============================================================================
// D-03 双入口（F-D 行为锁）
// ============================================================================

// TestPhase58_FirstEnabledADConfigID_SkipsSyncDisabled
// FirstEnabledADConfigID 只判 status（刻意不判 sync_enabled），
// FirstSyncEnabledADConfigID 判双条件 —— 两者在"配置启用但同步关闭"时取到不同行。
func TestPhase58_FirstEnabledADConfigID_SkipsSyncDisabled(t *testing.T) {
	db := setupDrainTestDB(t)

	seedDrainADConfig(t, db, "cfg-A", "A-同步关闭", "DC=a,DC=com", false, 0) // sync_enabled=0, status=0
	seedDrainADConfig(t, db, "cfg-B", "B-同步开启", "DC=b,DC=com", true, 0)  // sync_enabled=1, status=0
	seedDrainADConfig(t, db, "cfg-C", "C-停用", "DC=c,DC=com", true, 1)     // sync_enabled=1, status=1

	// seedDrainADConfig 用同一秒的时间戳，显式错开 created_at 让 ORDER BY 具备确定性
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, id := range []string{"cfg-A", "cfg-B", "cfg-C"} {
		require.NoError(t, db.Exec(`UPDATE sys_ad_config SET created_at = ? WHERE id = ?`,
			base.Add(time.Duration(i)*time.Minute).Format("2006-01-02 15:04:05"), id).Error)
	}

	ctx := t.Context()

	// FirstEnabledADConfigID 只判 status → 最早的启用配置是 A（尽管它 sync_enabled=0）
	enabledID, err := FirstEnabledADConfigID(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, "cfg-A", enabledID,
		"F-D：只判 status 的入口必须选中 sync_enabled=0 的第一条启用配置（否则手动同步行为变更）")

	// FirstSyncEnabledADConfigID 判双条件 → B（C 被 status 排除）
	syncID, err := FirstSyncEnabledADConfigID(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, "cfg-B", syncID, "双条件入口必须跳过 sync_enabled=0 的 A 与停用的 C")
}
