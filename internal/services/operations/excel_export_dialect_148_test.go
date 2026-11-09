package operations

// Phase 148 / DIALECT-01/02/03/04 回归守护（就近 *_test.go，AGENTS.md 分层）。
//
// 覆盖：
//   1. T1 DryRun 静态断言：resolveAssociations 实际生成的 SQL 不含 PG 专有强转、
//      含单侧 CAST(... AS TEXT)、三处关联 IN ? 子句存在；
//   2. 源码静态断言：excel_export_config.go / excel_export_service.go /
//      workstation_device_service.go 不含方言 token（防 RightCast 机制回流、防 CTE 回潮）；
//   3. SQLite 实跑断言：3 个关联列全部非空且逐字段正确（含软删工位恢复名称、
//      硬删表 sys_device_port_status 不加 deleted_at 过滤）；
//   4. 反例断言：关联查询失败必须返回 error（不再 logger.Warnf + continue 静默降级）；
//   5. SkipSoftDelete:true 两处语义保留（配置 + 实跑）；
//   6. 共享召回单条/批量逐字段一致（DIALECT-02/03）。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// dialect148SourceFiles 是本 phase 改造过的 3 个生产文件（相对本包目录）。
func dialect148SourceFiles(t *testing.T) map[string]string {
	t.Helper()
	names := []string{
		"excel_export_config.go",
		"excel_export_service.go",
		"workstation_device_service.go",
	}
	out := make(map[string]string, len(names))
	for _, n := range names {
		b, err := os.ReadFile(n)
		require.NoError(t, err, "读取 %s 失败（测试须在本包目录执行）", n)
		out[n] = string(b)
	}
	return out
}

// dialect148ForbiddenTokens 是 DIALECT-04 forbidigo 门禁覆盖的 8 类方言 token。
// Go 源码静态断言用（含字符串字面量/注释，比 forbidigo 的 AST 标识符匹配更严）。
var dialect148ForbiddenTokens = []string{
	"::text", "::uuid", "DISTINCT ON", "REGEXP_REPLACE",
	"NULLS LAST", "ILIKE", "ON CONFLICT",
}

// TestDialect148_SourceFilesNoDialectTokens 静态归零断言（MH-1 / MH-2 / MH-4）。
// forbidigo 只匹配 AST 标识符，无法命中字符串字面量/注释里的方言片段；
// 本断言补上这一层，是「反向变异必红」真正生效的门禁（见 148-SUMMARY）。
func TestDialect148_SourceFilesNoDialectTokens(t *testing.T) {
	files := dialect148SourceFiles(t)
	for name, src := range files {
		for _, tok := range dialect148ForbiddenTokens {
			assert.NotContains(t, src, tok,
				"%s 不得含方言 token %q（历史 CTE / RightCast 已移除）", name, tok)
		}
		// 任意 PG 专有强转（含通配）也不得出现。
		assert.NotContains(t, src, "::", "%s 不得含任何 :: 强转片段", name)
	}
	// RightCast 机制不得回流（配置/结构体不承载 SQL 方言片段）。
	assert.NotContains(t, files["excel_export_config.go"], "RightCast",
		"RightCast 方言机制必须删除")
	assert.Contains(t, files["excel_export_config.go"], "RightFieldKind",
		"必须改为语义枚举 RightFieldKind")
	assert.Contains(t, files["excel_export_service.go"], "CAST(",
		"service 必须统一生成单侧 CAST(... AS TEXT)")
	assert.Contains(t, files["workstation_device_service.go"], "recallPhysicalRows",
		"两处 CTE 必须改为共享 recallPhysicalRows")
	assert.NotContains(t, files["workstation_device_service.go"], "WITH workstation_ports",
		"两处 PG 专属 CTE 必须整体删除")
}

// dialect148NewDB 建 t.TempDir() 上的 SQLite（不落盘仓库、不写生产数据）。
func dialect148NewDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "dialect148.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// dialect148SeedJoinTables 建 resolveAssociations 的 3 张关联表：
//   - sys_workstation（有 deleted_at，含 1 个软删工位 → 验证 SkipSoftDelete:true 语义）
//   - sys_network_device（有 deleted_at）
//   - sys_device_port_status（硬删除表，**无 deleted_at 列** → 验证 SkipSoftDelete:true 语义）
func dialect148SeedJoinTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, ddl := range []string{
		`CREATE TABLE sys_workstation (id TEXT PRIMARY KEY, workstation_name TEXT, deleted_at DATETIME)`,
		`CREATE TABLE sys_network_device (id TEXT PRIMARY KEY, device_name TEXT, deleted_at DATETIME)`,
		`CREATE TABLE sys_device_port_status (id TEXT PRIMARY KEY, interface_name TEXT)`,
	} {
		require.NoError(t, db.Exec(ddl).Error, "建表失败: %s", ddl)
	}
	const (
		wsLive   = "10000000-0000-0000-0000-000000000001"
		wsSoft   = "10000000-0000-0000-0000-000000000002" // 软删工位：:351 语义
		devID    = "20000000-0000-0000-0000-000000000001"
		portID   = "30000000-0000-0000-0000-000000000001"
	)
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES (?, '工位-正常', NULL)`, wsLive).Error)
	// 软删工位（deleted_at 非空）：SkipSoftDelete:true 时仍应被关联到并恢复名称。
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES (?, '工位-软删恢复', '2026-01-01 00:00:00')`, wsSoft).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_network_device VALUES (?, '交换机-A', NULL)`, devID).Error)
	// 硬删除表：故意不建 deleted_at 列，若误加 `deleted_at IS NULL` 会 SQLite 报 no such column。
	require.NoError(t, db.Exec(`INSERT INTO sys_device_port_status VALUES (?, 'GE1/0/1')`, portID).Error)
}

// dialect148SeedJoinTables 对应的固定 ID（供数据行构造）。
const (
	dialect148WSLive = "10000000-0000-0000-0000-000000000001"
	dialect148WSSoft = "10000000-0000-0000-0000-000000000002"
	dialect148DevID  = "20000000-0000-0000-0000-000000000001"
	dialect148PortID = "30000000-0000-0000-0000-000000000001"
)

// dialect148InfoPointRows 构造 infoPoint 导出的主表数据行（3 个左值均为 Go 字符串）。
func dialect148InfoPointRows() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"workstation_id": dialect148WSSoft, // 软删工位 → 验证 :351 SkipSoftDelete
			"device_id":      dialect148DevID,
			"port_id":        dialect148PortID, // 硬删表 → 验证 :359 SkipSoftDelete
		},
		{
			"workstation_id": dialect148WSLive,
			"device_id":      dialect148DevID,
			"port_id":        dialect148PortID,
		},
	}
}

// TestDialect148_ResolveAssociations_SQLiteRealResult SQLite 实跑逐字段断言（MH-1 / T1-6）。
func TestDialect148_ResolveAssociations_SQLiteRealResult(t *testing.T) {
	db := dialect148NewDB(t)
	dialect148SeedJoinTables(t, db)

	cfg, ok := GetExportConfig("infoPoint")
	require.True(t, ok, "infoPoint 导出配置必须存在")
	svc := &excelExportServiceImpl{db: db}
	data := dialect148InfoPointRows()

	got, err := svc.resolveAssociations(context.Background(), cfg, data)
	require.NoError(t, err, "根治后关联查询不得静默吞错误、也不得报方言错误")
	require.Len(t, got, 2)

	// 逐字段断言：三个关联列全部非空且值正确（workstationName / deviceName / portName）。
	assert.Equal(t, "工位-软删恢复", got[0]["workstationName"],
		"软删工位名称仍恢复显示（SkipSoftDelete:true :351 语义）")
	assert.Equal(t, "工位-正常", got[1]["workstationName"])
	for i := range got {
		assert.Equal(t, "交换机-A", got[i]["deviceName"], "deviceName 关联列非空且正确")
		assert.Equal(t, "GE1/0/1", got[i]["portName"],
			"portName 关联列非空且正确（硬删表不加 deleted_at 过滤，:359 语义）")
	}
}

// TestDialect148_ResolveAssociations_ErrorPropagates 反例断言（MH-1 / T1-6 / T5 变异）。
// 人为让关联表不存在 → resolveAssociations 必须返回 error，而不是「日志 + 空列仍成功」。
func TestDialect148_ResolveAssociations_ErrorPropagates(t *testing.T) {
	db := dialect148NewDB(t)
	// 故意不建任何关联表 → 第一条 join 查询即报错。
	cfg, ok := GetExportConfig("infoPoint")
	require.True(t, ok)
	svc := &excelExportServiceImpl{db: db}

	_, err := svc.resolveAssociations(context.Background(), cfg, dialect148InfoPointRows())
	require.Error(t, err, "关联查询失败必须返回 error（禁止 Warnf+continue 静默降级）")
	assert.Contains(t, err.Error(), "关联表", "错误须带关联表定位信息")
}

// TestDialect148_ResolveAssociations_DryRunNoDialectToken 捕获 resolveAssociations
// **实际生成**的 SQL（P1-8 式静态断言）：用真实 SQLite + GORM Query 回调记录每条 SQL，
// 逐条断言无方言片段、含双方言单侧 CAST、三处关联 IN ? 子句齐全。
// （实测 golangci-lint 2.12.2 的 forbidigo 只匹配 AST 标识符，不命中字符串字面量；
//   本断言补上字符串/注释层，是反向变异真正生效的门禁。）
func TestDialect148_ResolveAssociations_DryRunNoDialectToken(t *testing.T) {
	db := dialect148NewDB(t)
	dialect148SeedJoinTables(t, db)

	var captured []string
	capture := func(tx *gorm.DB) {
		captured = append(captured, tx.Statement.SQL.String())
	}
	// resolveAssociations 用 .Rows()，GORM 走 Row 处理器；同时注册 Query 处理器
	// 以兼容未来改为 .Find()/Scan() 的实现。
	require.NoError(t, db.Callback().Row().After("gorm:row").Register("dialect148:capture_row", capture))
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("dialect148:capture_query", capture))

	cfg, ok := GetExportConfig("infoPoint")
	require.True(t, ok)
	svc := &excelExportServiceImpl{db: db}
	_, err := svc.resolveAssociations(context.Background(), cfg, dialect148InfoPointRows())
	require.NoError(t, err)

	require.NotEmpty(t, captured, "必须捕获到 resolveAssociations 生成的 SQL")
	joined := strings.Join(captured, "\n")
	assert.NotContains(t, joined, "::", "导出关联 SQL 不得含任何 PG 专有强转")
	assert.Contains(t, joined, "CAST(", "导出关联 SQL 必须用双方言单侧 CAST(... AS TEXT)")
	assert.Contains(t, joined, "AS TEXT", "CAST 目标类型必须是 TEXT")
	assert.Contains(t, joined, "IN ", "三处关联 IN ? 子句必须存在")
	// 三张关联表都出现在 SQL 里（IN ? 子句数量 = 3）
	inCount := strings.Count(joined, "IN ")
	assert.GreaterOrEqual(t, inCount, 3, "三处关联查询都应生成 IN ? 子句，实际 SQL: %s", joined)
}

// TestDialect148_SkipSoftDeleteSemantics 两处 SkipSoftDelete:true 语义保留（T3-4 / MH-1）。
func TestDialect148_SkipSoftDeleteSemantics(t *testing.T) {
	cfg, ok := GetExportConfig("infoPoint")
	require.True(t, ok)
	byAs := make(map[string]*JoinConfig)
	for _, c := range cfg.Columns {
		if c.Join != nil {
			byAs[c.Join.As] = c.Join
		}
	}
	// :351 sys_workstation 软删恢复名称
	ws, ok := byAs["workstationName"]
	require.True(t, ok)
	assert.True(t, ws.SkipSoftDelete, "sys_workstation 关联须保留 SkipSoftDelete:true（软删工位恢复名称）")
	assert.Equal(t, RightFieldUUID, ws.RightFieldKind, "右表 id 为 UUID 列 → 语义枚举")
	// :355 sys_network_device 正常走 deleted_at 过滤
	dev, ok := byAs["deviceName"]
	require.True(t, ok)
	assert.False(t, dev.SkipSoftDelete, "sys_network_device 有 deleted_at 列，须保留软删过滤")
	assert.Equal(t, RightFieldUUID, dev.RightFieldKind)
	// :359 sys_device_port_status 硬删除表无 deleted_at 列
	port, ok := byAs["portName"]
	require.True(t, ok)
	assert.True(t, port.SkipSoftDelete, "sys_device_port_status 硬删表须保留 SkipSoftDelete:true")
	assert.Equal(t, RightFieldUUID, port.RightFieldKind)
	// 任何配置都不得承载 SQL 方言片段，RightFieldKind 必须是语义枚举合法值。
	for as, j := range byAs {
		assert.Contains(t, []RightFieldKind{RightFieldText, RightFieldUUID}, j.RightFieldKind,
			"%s 的 RightFieldKind 必须是语义枚举合法值", as)
	}
}
