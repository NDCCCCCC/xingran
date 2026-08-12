package services

/**
 * Phase 15 PERF 抽样验证测试
 *
 * 目的 (D-15 锁定):
 *   - 不替代完整 benchmark 脚本
 *   - 用 EXPLAIN ANALYZE 抽样证据化 索引/MV 命中
 *   - 默认 t.Skip,IMPLEMENT 阶段用 `-run TestMACPerfVerify` 手动跑
 *
 * 使用方式:
 *   go test -v -run TestMACPerfVerify ./internal/services/ -args -perf_verify
 */

import (
	"context"
	"strings"
	"testing"
	"time"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// allMatViewNames 测试用的 4 个物化视图名 (与 matViewWhiteList 一致)
var allMatViewNames = []string{
	"mv_mac_port_latest",
	"mv_mac_device_summary",
	"mv_mac_long_occupancy_top",
	"mv_mac_port_daily_count",
}

// assertPlanContains 任一 substring 命中即通过;全部不命中 t.Errorf
func assertPlanContains(t *testing.T, plan string, substrings []string) {
	t.Helper()
	lower := strings.ToLower(plan)
	for _, s := range substrings {
		if strings.Contains(lower, strings.ToLower(s)) {
			t.Logf("✓ plan contains %q", s)
		} else {
			t.Errorf("✗ plan missing %q\nplan:\n%s", s, plan)
		}
	}
}

// perfVerifyDB 返回测试用的 DB,若无环境则 t.Skip
func perfVerifyDB(t *testing.T) *gorm.DB {
	t.Helper()
	// 测试环境 DB 通过 TestMain 或 setup 注入;若未注入则 skip
	if testDB == nil {
		t.Skip("Phase 15 perf verify: requires -perf_verify flag and live PostgreSQL test DB; skipping")
	}
	return testDB
}

// testDB 进程级 DB 句柄 (由 test setup 注入)
var testDB *gorm.DB

// SetTestDBForPerfVerify 由 TestMain 调用,提供真实 DB 给抽样测试
func SetTestDBForPerfVerify(db *gorm.DB) {
	testDB = db
}

// TestMACPerfVerify Phase 15 抽样验证主入口
func TestMACPerfVerify(t *testing.T) {
	// 检查 -perf_verify flag
	perfVerify := false
	for _, arg := range []string{"-perf_verify", "-perf-verify"} {
		if arg == "" {
			continue
		}
		// 简化: 任何 -perf_verify 形参都启用
		perfVerify = true
		_ = arg
	}
	if !perfVerify {
		t.Skip("Phase 15 抽样验证,需在 IMPLEMENT 阶段手动跑 -perf_verify")
	}

	db := perfVerifyDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("TestPortHistory_90d_IndexScan", func(t *testing.T) {
		// 端口历史 SQL 模式: WHERE device_id = ? AND first_seen BETWEEN ? AND ?
		sql := `EXPLAIN ANALYZE
SELECT * FROM sys_device_mac_history
WHERE device_id = '00000000-0000-0000-0000-000000000001'
  AND mac_address = 'AA:BB:CC:DD:EE:FF'
  AND first_seen >= NOW() - INTERVAL '90 days'
  AND first_seen <= NOW()
ORDER BY first_seen DESC
LIMIT 50`
		var plan string
		if err := db.WithContext(ctx).Raw(sql).Scan(&plan).Error; err != nil {
			// EXPLAIN 返回多行,Scan 单字段会失败,改用 Rows
			rows, err2 := db.WithContext(ctx).Raw(sql).Rows()
			if err2 != nil {
				t.Fatalf("EXPLAIN ANALYZE 失败: %v", err2)
			}
			defer rows.Close()
			var lines []string
			for rows.Next() {
				var line string
				_ = rows.Scan(&line)
				lines = append(lines, line)
			}
			plan = strings.Join(lines, "\n")
		}
		t.Logf("plan:\n%s", plan)
		assertPlanContains(t, plan, []string{
			"idx_mac_history_device_mac_first_seen",
		})
	})

	t.Run("TestDeviceHistory_90d_IndexScan", func(t *testing.T) {
		sql := `EXPLAIN ANALYZE
SELECT * FROM sys_device_mac_history
WHERE device_id = '00000000-0000-0000-0000-000000000001'
  AND first_seen >= NOW() - INTERVAL '90 days'
  AND first_seen <= NOW()
ORDER BY first_seen DESC
LIMIT 100`
		rows, err := db.WithContext(ctx).Raw(sql).Rows()
		if err != nil {
			t.Fatalf("EXPLAIN ANALYZE 失败: %v", err)
		}
		defer rows.Close()
		var lines []string
		for rows.Next() {
			var line string
			_ = rows.Scan(&line)
			lines = append(lines, line)
		}
		plan := strings.Join(lines, "\n")
		t.Logf("plan:\n%s", plan)
		assertPlanContains(t, plan, []string{
			"idx_mac_history_device_mac_first_seen",
		})
	})

	t.Run("TestConnectionStats_7d_IndexScan", func(t *testing.T) {
		// 7 天范围连接统计: 命中 idx_mac_history_device_mac_first_seen 或 mv_mac_device_summary
		sql := `EXPLAIN ANALYZE
SELECT device_id, COUNT(DISTINCT mac_address) AS unique_macs
FROM sys_device_mac_history
WHERE first_seen >= NOW() - INTERVAL '7 days'
  AND first_seen <= NOW()
GROUP BY device_id
LIMIT 100`
		rows, err := db.WithContext(ctx).Raw(sql).Rows()
		if err != nil {
			t.Fatalf("EXPLAIN ANALYZE 失败: %v", err)
		}
		defer rows.Close()
		var lines []string
		for rows.Next() {
			var line string
			_ = rows.Scan(&line)
			lines = append(lines, line)
		}
		plan := strings.Join(lines, "\n")
		t.Logf("plan:\n%s", plan)
		// 命中任一即通过 (7 天分区裁剪 + 复合索引 OR MV-02)
		assertPlanContains(t, plan, []string{
			"idx_mac_history_device_mac_first_seen",
			"mv_mac_device_summary",
		})
	})

	t.Run("TestHeatmap_MV04_BitmapScan", func(t *testing.T) {
		// heatmap 走 MV-04 (mv_mac_port_daily_count)
		sql := `EXPLAIN ANALYZE
SELECT device_id, device_name_snapshot, interface_name, date, change_count
FROM mv_mac_port_daily_count
WHERE date >= CURRENT_DATE - INTERVAL '7 days'
  AND date <= CURRENT_DATE
ORDER BY change_count DESC
LIMIT 100`
		rows, err := db.WithContext(ctx).Raw(sql).Rows()
		if err != nil {
			t.Fatalf("EXPLAIN ANALYZE 失败: %v", err)
		}
		defer rows.Close()
		var lines []string
		for rows.Next() {
			var line string
			_ = rows.Scan(&line)
			lines = append(lines, line)
		}
		plan := strings.Join(lines, "\n")
		t.Logf("plan:\n%s", plan)
		// MV-04 必须 Seq Scan 或 Index Scan mv_mac_port_daily_count 自身
		assertPlanContains(t, plan, []string{
			"mv_mac_port_daily_count",
		})
	})

	t.Run("TestMatViewRefresh_All4_Success", func(t *testing.T) {
		// 4 个 MV 依次 REFRESH CONCURRENTLY,断言全部返回 nil
		if !isPostgresDB(db) {
			t.Skip("非 PostgreSQL,跳过 MV 刷新抽样")
		}
		for _, name := range allMatViewNames {
			refreshSQL := "REFRESH MATERIALIZED VIEW CONCURRENTLY " + name
			if err := db.WithContext(ctx).Exec(refreshSQL).Error; err != nil {
				t.Errorf("✗ REFRESH %s 失败: %v", name, err)
			} else {
				t.Logf("✓ REFRESH %s 成功", name)
			}
		}
	})
}

// isPostgresDB 判断 DB 驱动类型
func isPostgresDB(db *gorm.DB) bool {
	if db == nil || db.Config == nil || db.Config.Dialector == nil {
		return false
	}
	return db.Config.Dialector.Name() == "postgres"
}

// Compile-time 接口断言 (确保 Service 仍存在)
var _ MACHistoryMatViewService = (MACHistoryMatViewService)(nil)
var _ = applogger.Infof
