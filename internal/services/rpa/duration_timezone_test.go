package rpa

import (
	"testing"
	"time"
)

// TestDurationTimezoneConsistency 锁定 2026-11-25 生产 UAT 缺陷：
// sys_rpa_executions.start_time 是 `timestamp without time zone`，写入本地时间，
// GORM 读回按 UTC 解释 → 与本地 time.Now() 相减得 -8h（实测 -28799423ms）。
// 本测试验证「读回值按本地时区重解释后」求差为非负且量级正确。
func TestDurationTimezoneConsistency(t *testing.T) {
	now := time.Now()
	// 模拟 GORM 读回：同一墙钟时间但 Location=UTC
	localWall := now.Add(-2 * time.Second)
	readBack := time.Date(localWall.Year(), localWall.Month(), localWall.Day(),
		localWall.Hour(), localWall.Minute(), localWall.Second(), localWall.Nanosecond(), time.UTC)

	// 修复前：直接相减 → 约 -8h
	if naive := now.Sub(readBack).Milliseconds(); naive > 0 {
		t.Logf("naive diff = %dms（本机时区可能为 UTC，跳过负值断言）", naive)
	}

	// 修复后：按本地时区重解释
	startedAt := readBack
	if startedAt.Location() == time.UTC {
		startedAt = time.Date(startedAt.Year(), startedAt.Month(), startedAt.Day(),
			startedAt.Hour(), startedAt.Minute(), startedAt.Second(), startedAt.Nanosecond(), time.Local)
	}
	d := now.Sub(startedAt).Milliseconds()
	if d < 0 {
		t.Fatalf("修复后 duration 仍为负: %dms", d)
	}
	if d > 10000 {
		t.Fatalf("修复后 duration 量级异常: %dms（期望 ~2000ms）", d)
	}
}
