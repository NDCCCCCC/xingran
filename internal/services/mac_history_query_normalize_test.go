package services

// mac_history_query_normalize_test.go — Phase 73 / #5（DBG-06）回归守护
//
// 根因（2026-10-27 PG 生产库实测，见 .planning/debug/mac-history-query-returns-empty.md）：
//   sys_device_mac_history 的 mac_address 存储格式 = **大写冒号**（B0:22:7A:2E:4A:4F），
//   而旧查询把入参 normalize 成**无分隔大写**（B0227A2E4A4F）后直接等值比较
//   → 等值匹配永远空集。实测对照：无分隔字面量 count=0 / 冒号字面量 count=53。
//
// 修复契约（本文件锁定的两条）：
//  1. 双侧归一化：SQL 对**存储值**也做去分隔符 + 大写后再与入参（无分隔）比较，
//     任意输入格式（点分 / 连字符 / 无分隔 / 大小写）都能命中任意存储格式。
//  2. 错误传播：查询失败必须返回 error（handler 层返回 500），不得吞成空数组。

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newMACNormalizeTestDB 建一个独立的内存 SQLite（每条测试独立，避免 file::memory:?cache=shared 串库）
// 并迁移 MAC 历史表。
func newMACNormalizeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.DeviceMACHistory{}))
	return db
}

// seedMACHistory 写入若干历史行。
//
// 注意：DeviceMACHistory.BeforeCreate 会把 MAC 归一化为**大写冒号** canonical 格式，
// 这正是生产库的存储格式（2026-10-27 实测），所以走 GORM Create 就等价于「存大写冒号」。
// storedRaw 用于模拟历史脏格式（点分 / 连字符 / 小写无分隔）——直接 UPDATE 绕过 hook。
func seedMACHistory(t *testing.T, db *gorm.DB, seat string, macs []string) []string {
	t.Helper()
	base := time.Date(2026, 10, 27, 16, 2, 0, 0, time.Local)
	ids := make([]string, 0, len(macs))
	for i, mac := range macs {
		rec := &models.DeviceMACHistory{
			DeviceID:           "96aab2e4-65c5-464f-9ee2-c4645d8ae868",
			DeviceNameSnapshot: "CX-WH-WH-04F-FL-RS8607E-01",
			MACAddress:         mac,
			InterfaceName:      "GE0/40",
			EventType:          models.EventAppeared,
			FirstSeen:          base.Add(time.Duration(i) * time.Minute),
			LastSeen:           base.Add(time.Duration(i) * time.Minute),
			CollectedAt:        base.Add(time.Duration(i) * time.Minute),
		}
		require.NoError(t, db.Create(rec).Error)
		ids = append(ids, rec.ID)
	}
	return ids
}

// TestQueryHistory_MACDoubleSidedNormalization 是 #5 的核心回归用例。
//
// 断言矩阵（存储大写冒号 × 四种输入格式全部命中）：
//
//	存储      : B0:22:7A:2E:4A:4F（GORM hook 产出，等价生产库实测格式）
//	查询输入  1) 无分隔 B0227A2E4A4F      → 命中（修复前：0 行，BUG）
//	         2) 小写无分隔 b0227a2e4a4f   → 命中
//	         3) 点分 b022.7a2e.4a4f       → 命中
//	         4) 连字符 B0-22-7A-2E-4A-4F  → 命中
//	         5) 冒号   B0:22:7A:2E:4A:4F  → 命中（走索引分支）
func TestQueryHistory_MACDoubleSidedNormalization(t *testing.T) {
	db := newMACNormalizeTestDB(t)
	seedMACHistory(t, db, "main", []string{"B0:22:7A:2E:4A:4F"})

	svc := &macHistoryQueryServiceImpl{db: db, cache: nil}
	ctx := context.Background()

	// 先断言存储侧确实是「大写冒号」——否则本用例失去意义（防止 hook 行为变更导致假绿）
	var stored struct{ MACAddress string }
	require.NoError(t, db.Raw("SELECT mac_address FROM sys_device_mac_history LIMIT 1").Scan(&stored).Error)
	require.Equal(t, "B0:22:7A:2E:4A:4F", stored.MACAddress,
		"前置条件：存储格式必须是大写冒号（等价 2026-10-27 PG 实测）")

	inputs := []struct {
		name string
		mac  string
	}{
		{"无分隔大写(修复前必空集)", "B0227A2E4A4F"},
		{"无分隔小写", "b0227a2e4a4f"},
		{"点分cisco风格", "b022.7a2e.4a4f"},
		{"连字符大写", "B0-22-7A-2E-4A-4F"},
		{"冒号大写(与存储同格式)", "B0:22:7A:2E:4A:4F"},
	}

	for _, in := range inputs {
		t.Run(in.name, func(t *testing.T) {
			result, err := svc.QueryHistory(ctx, &MACHistoryListQuery{
				Current:  1,
				PageSize: 20,
				MAC:      in.mac,
			})
			require.NoError(t, err, "查询不应报错")
			require.NotNil(t, result)
			assert.EqualValues(t, 1, result.Total, "双侧归一化后必须命中 1 行（输入 %q）", in.mac)
			require.Len(t, result.List, 1)
			assert.Equal(t, "B0:22:7A:2E:4A:4F", result.List[0].MACAddress,
				"返回的应是存储原值（大写冒号），便于前端展示与二次查询")
		})
	}
}

// TestQueryHistory_MACDirtyStorageVariants 验证反向场景：存储侧是历史脏格式时，
// 查询侧传入任意格式仍能命中（双侧归一化的另一半价值）。
func TestQueryHistory_MACDirtyStorageVariants(t *testing.T) {
	db := newMACNormalizeTestDB(t)
	ids := seedMACHistory(t, db, "main", []string{
		"A4:0C:66:2C:E9:44",
		"00:E0:FC:09:BC:F9",
		"11:22:33:44:55:66",
	})

	// 绕过 BeforeCreate hook，把三条改写成三种不同的历史脏格式
	for i, dirty := range []string{"a40c.662c.e944", "00-e0-fc-09-bc-f9", "112233445566"} {
		require.NoError(t, db.Exec(
			"UPDATE sys_device_mac_history SET mac_address = ? WHERE id = ?", dirty, ids[i],
		).Error)
	}

	svc := &macHistoryQueryServiceImpl{db: db, cache: nil}
	ctx := context.Background()

	for _, tc := range []struct{ dirty, query string }{
		{"点分存储", "A4:0C:66:2C:E9:44"},
		{"连字符存储", "00:E0:FC:09:BC:F9"},
		{"无分隔存储", "11-22-33-44-55-66"},
	} {
		t.Run(tc.dirty+"_查询"+tc.query, func(t *testing.T) {
			result, err := svc.QueryHistory(ctx, &MACHistoryListQuery{Current: 1, PageSize: 20, MAC: tc.query})
			require.NoError(t, err)
			assert.EqualValues(t, 1, result.Total, "存储 %q / 查询 %q 必须命中", tc.dirty, tc.query)
		})
	}
}

// TestQueryHistory_NoMatchStillEmpty 确认「归一化没有把无关 MAC 也拉进来」——
// 未命中时返回空列表 + nil error（这才是合法的空结果，与「查询失败」必须区分）。
func TestQueryHistory_NoMatchStillEmpty(t *testing.T) {
	db := newMACNormalizeTestDB(t)
	seedMACHistory(t, db, "main", []string{"B0:22:7A:2E:4A:4F"})

	svc := &macHistoryQueryServiceImpl{db: db, cache: nil}
	result, err := svc.QueryHistory(context.Background(), &MACHistoryListQuery{
		Current: 1, PageSize: 20, MAC: "DE:AD:BE:EF:00:01",
	})
	require.NoError(t, err)
	assert.EqualValues(t, 0, result.Total)
	assert.Empty(t, result.List)
}

// TestQueryHistory_QueryFailureReturnsError 是 #5 的第二条契约（REQ 原文：
// 「DB 错误不再被吞成空结果（查询失败时接口返回可见错误而不是空数组）」）。
//
// 制造失败：迁移后 DROP TABLE，让 SQL 报 no such table。
// 断言 err != nil —— 修复前若吞 error，此处会拿到 (空列表, nil) 的假绿。
func TestQueryHistory_QueryFailureReturnsError(t *testing.T) {
	db := newMACNormalizeTestDB(t)
	seedMACHistory(t, db, "main", []string{"B0:22:7A:2E:4A:4F"})
	require.NoError(t, db.Exec("DROP TABLE sys_device_mac_history").Error)

	svc := &macHistoryQueryServiceImpl{db: db, cache: nil}
	result, err := svc.QueryHistory(context.Background(), &MACHistoryListQuery{
		Current: 1, PageSize: 20, MAC: "B0227A2E4A4F",
	})
	require.Error(t, err, "DB 查询失败必须返回 error，不得吞成空数组")
	assert.Nil(t, result, "失败时不应返回半成品结果")
	assert.Contains(t, err.Error(), "查询历史记录", "错误应带可诊断上下文: %v", err)
}

// TestQueryConnectionStats_MACDoubleSidedNormalization 守住 Raw SQL 三条语句
// （明细 / TopByMAC / TopByPort）的归一化一致性——它们原先把无分隔字面量硬编码进 SQL。
//
// 说明：ConnectionStats 的 Raw SQL 使用 PG 专属语法（EXTRACT(EPOCH ...) / COUNT(*) FILTER /
// ::bigint），SQLite 无法执行，因此本用例只断言「拼进 SQL 的过滤片段与 args 的双侧归一化契约」，
// 不改用 PG 连接（测试不依赖真实 PG，见任务边界）。
func TestQueryConnectionStats_MACDoubleSidedNormalization(t *testing.T) {
	canonical, compact, ok := macMatchArgs("b022.7a2e.4a4f")
	require.True(t, ok)
	assert.Equal(t, "B0:22:7A:2E:4A:4F", canonical, "入参归一到 canonical 大写冒号")
	assert.Equal(t, "B0227A2E4A4F", compact, "入参归一到无分隔大写")

	// SQL 片段必须同时含「存储值原样等值」与「存储值归一化后等值」两个分支
	assert.Contains(t, macMatchSQL, "mac_address = ?", "分支一：等值命中索引")
	assert.Contains(t, macMatchSQL, "UPPER(mac_address)", "分支二：存储值大写")
	assert.Contains(t, macMatchSQL, "REPLACE", "分支二：存储值去分隔符")

	// 无效 MAC 必须 ok=false（调用方据此返回错误，而不是拼出永远空集的 SQL）
	_, _, bad := macMatchArgs("AA:BB:CC")
	assert.False(t, bad, "非法 MAC 必须被拒绝")
}
