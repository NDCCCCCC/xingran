package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestLocalWallClock 锁定 LocalWallClock 的契约:
// wall clock(Y/M/D/H/M/S/Nano)与输入完全一致,loc 被重塑为 time.Local。
//
// 为什么重要: pgx 读 timestamp without time zone 强制赋予 UTC loc,
// 但 DB 实际存的是北京时间裸值(采集器 time.Now() in Asia/Shanghai)。
// 若 wall clock 与 loc 不一致,JSON 输出 "Z" → 前端 dayjs 当 UTC 再 +8 → 错 8h。
// 此测试守住"重塑 loc 但不偏移时间"的契约,防止后续重构破坏。
func TestLocalWallClock(t *testing.T) {
	// 模拟 pgx 读出的形态:wall=2026-06-29 13:01:30.123456789,loc=UTC
	utcWallBeijing := time.Date(2026, 6, 29, 13, 1, 30, 123456789, time.UTC)

	got := LocalWallClock(utcWallBeijing)

	// 用 Format() 比较:无论 loc 怎么变,wall clock 字符串必须完全一致
	const layout = "2006-01-02T15:04:05.000000000"
	wantStr := utcWallBeijing.Format(layout)
	gotStr := got.Format(layout)
	if gotStr != wantStr {
		t.Errorf("LocalWallClock 改变了 wall clock:\n  want %s\n  got  %s", wantStr, gotStr)
	}
}

// TestLocalWallClock_JSONSerialization 验证修复效果:
// 修复后 JSON 必须输出 wall clock 字符串(后跟本地 offset 如 "+08:00"),
// 不能输出 "Z"(会让前端 dayjs 当 UTC 再 +8 → 错 8h)。
//
// 生产部署时区是 Asia/Shanghai(+08:00),但 CI runner 是 UTC —
// 在 UTC 机器上 time.Local==UTC,marshal 合法地输出 "Z",断言必挂。
// 因此测试内把 time.Local 钉到固定 +08:00 区域,保证在任何 runner 上
// 都验证"offset 被附加、wall clock 不偏移"这一真实契约。
func TestLocalWallClock_JSONSerialization(t *testing.T) {
	origLocal := time.Local
	time.Local = time.FixedZone("Asia/Shanghai(+8 test)", 8*60*60)
	defer func() { time.Local = origLocal }()

	// 模拟 pgx 读出的形态:wall=13:01:30,loc=UTC
	utcBeijingWall := time.Date(2026, 6, 29, 13, 1, 30, 0, time.UTC)

	// 修复前(直接 marshal)会输出 "...Z" — 这是 bug 的源头
	beforeBytes, _ := json.Marshal(utcBeijingWall)
	beforeJSON := string(beforeBytes)
	if !strings.HasSuffix(beforeJSON, "Z\"") {
		t.Logf("注意: 直接 marshal UTC 时间 = %s(本测试机 time.Local 可能不是 UTC,bug 不一定复现)", beforeJSON)
	}

	// 修复后(经 LocalWallClock 矫正 loc)
	fixed := LocalWallClock(utcBeijingWall)
	afterBytes, _ := json.Marshal(fixed)
	afterJSON := string(afterBytes)

	// 关键不变量:wall clock 部分必须是 "2026-06-29T13:01:30"(不偏移)
	const wantPrefix = `"2026-06-29T13:01:30`
	if !strings.HasPrefix(afterJSON, wantPrefix) {
		t.Errorf("LocalWallClock 改变了 wall clock 字符串:\n  want 前缀 %s\n  got  %s", wantPrefix, afterJSON)
	}

	// 关键不变量:不能输出 "...Z"(否则前端会按 UTC 解析再 +8 错位)
	if strings.HasSuffix(afterJSON, "Z\"") {
		t.Errorf("LocalWallClock 修复后 JSON 仍以 Z 结尾,前端会按 UTC +8 错位:\n  got %s", afterJSON)
	}
}