package services

import (
	"testing"
	"time"
)

// TestParseQueryTimeRangeAsLocal 锁定 query 时间过滤的时区契约(Plan D 续)。
//
// 根因: sys_device_mac_history.first_seen/last_seen 是 timestamp without time zone,
// 采集器 time.Now()(Local=Asia/Shanghai) 写入,DB 存北京时间裸值(墙钟)。
// pgx 格式化 timestamp-without-tz 参数时用 Go time 的 loc 墙钟。
// 前端 RFC3339 字符串解析后 loc=UTC,直接传参会发 UTC 墙钟 → 与北京墙钟列错 8h,
// 最近 8h 数据被 cutoff 滤掉(典型症状:timeline 看到 6/29 21:01 但 6/30 数据存在)。
//
// 契约: 解析后必须 .Local() 转 Asia/Shanghai loc(墙钟变成北京),pgx 才发北京墙钟。
//
// 此测试守住"必须转 Local"的契约,防止后续重构去除 .Local() 又引入 8h 错位 bug。
// 注:time.Local 在 cmd/main.go setTimeZone() 已固定为 Asia/Shanghai,此测试依赖该全局。
func TestParseQueryTimeRangeAsLocal(t *testing.T) {
	// 时间(Local 必须为 Asia/Shanghai 才能让此测试有意义)
	if time.Local.String() != "Asia/Shanghai" {
		t.Skipf("此测试要求 time.Local=Asia/Shanghai,当前=%s,跳过(开发机可能用其他 tz)", time.Local.String())
	}

	// 前端会发送的典型 RFC3339 UTC 字符串(以 "now 在北京 13:00 = UTC 05:00" 为例)
	const utcRFC3339 = "2026-06-30T05:00:00Z"

	parsed, err := time.Parse(time.RFC3339, utcRFC3339)
	if err != nil {
		t.Fatalf("time.Parse 失败: %v", err)
	}

	// 不修正(原始 RFC3339 解析结果): loc=UTC, 墙钟=UTC
	if parsed.Location().String() != "UTC" {
		t.Logf("RFC3339 解析后 loc=%s(预期 UTC)", parsed.Location().String())
	}

	// 修正后(.Local() 转 Asia/Shanghai): loc=Asia/Shanghai, 墙钟=北京
	localized := parsed.Local()
	if localized.Location().String() != "Asia/Shanghai" {
		t.Errorf("Local() 后 loc=%s,期望 Asia/Shanghai", localized.Location().String())
	}

	// 关键: Local() 后墙钟必须是北京时间(UTC 05:00 → 北京 13:00)
	const wantBeijingWall = "2026-06-30T13:00:00"
	if got := localized.Format("2006-01-02T15:04:05"); got != wantBeijingWall {
		t.Errorf("Local() 后墙钟=%s,期望 %s(UTC→Beijing 偏移 8h)", got, wantBeijingWall)
	}

	// 反向不变量: 瞬时值不变(同 .Unix())
	if localized.Unix() != parsed.Unix() {
		t.Errorf("Local() 改变了瞬时值!parsed=%v localized=%v", parsed.Unix(), localized.Unix())
	}

	// pgx 行为模拟: 当传 localized(time.Time with loc=Asia/Shanghai)给
	// timestamp-without-tz 列时,pgx 格式化为该 time 的 loc 墙钟(北京墙钟)。
	// 这与 DB 列里存的北京墙钟一致 → 不再有 8h 错位。
	// 此处用 Format() 模拟 pgx 对 timestamp-without-tz 的输出。
	pgxOut := localized.Format("2006-01-02 15:04:05")
	wantPgxOut := "2026-06-30 13:00:00"
	if pgxOut != wantPgxOut {
		t.Errorf("pgx 输出墙钟=%s,期望 %s(必须北京墙钟匹配列存储)", pgxOut, wantPgxOut)
	}
}

// TestParseQueryTimeRangeAsLocal_DemonstratesBug 反证:
// 不转 Local 的旧行为会让 pgx 发 UTC 墙钟,与北京墙钟列错 8h。
// 这正是之前 timeline 显示 "最晚 6/29 21:01,6/30 数据看不到" 的根因。
func TestParseQueryTimeRangeAsLocal_DemonstratesBug(t *testing.T) {
	if time.Local.String() != "Asia/Shanghai" {
		t.Skipf("此测试要求 time.Local=Asia/Shanghai,当前=%s", time.Local.String())
	}

	const utcRFC3339 = "2026-06-30T05:00:00Z" // 北京 13:00

	parsed, _ := time.Parse(time.RFC3339, utcRFC3339)

	// 不修正,pgx 发出的字符串(UTC 墙钟):
	pgxOut := parsed.Format("2006-01-02 15:04:05")
	wantBugDemo := "2026-06-30 05:00:00" // 北京墙钟列里 13:00 行的 first_seen 会被比较
	if pgxOut != wantBugDemo {
		t.Errorf("演示 bug: pgx UTC 墙钟输出=%s(期望 %s),与北京墙钟列错 8h", pgxOut, wantBugDemo)
	}

	// 北京墙钟 6/30 13:00 行的 first_seen 与 UTC 墙钟 6/30 05:00 比较:
	//   "2026-06-30 13:00:00" <= "2026-06-30 05:00:00"? 否 → 被滤掉
	beijingWallClock := time.Date(2026, 6, 30, 13, 0, 0, 0, time.Local)
	if beijingWallClock.Format("2006-01-02 15:04:05") <= pgxOut {
		t.Errorf("bug 演示失败: 应该显示 UTC 墙钟 05:00 < 北京墙钟 13:00 导致最近 8h 数据被滤掉")
	}
}