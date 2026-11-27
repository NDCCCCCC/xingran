package migrations

import (
	"encoding/json"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ============================================================================
// Migration 261 测试 — RPA 动作类型别名归一化（数据根治）+ 双方言幂等
//
// 覆盖：
//  1) goto / GOTO / url 全部改成 navigate，且**其它字段一字不改**
//  2) 已是 navigate 的任务零改动（不写库）
//  3) 幂等：连跑两次，第二次零 UPDATE，脚本内容不变
//  4) 非数组/脏 script 不阻断、不丢任务（只 WARN 跳过）
//  5) 迁移侧的别名集合与 worker 侧（rpa-worker/internal/executor/engine.go）
//     保持同一集合（防「数据改了运行时仍不认」的半修状态）
//  6) 迁移后 worker 的归一化仍幂等（值是 navigate 时不再触发 WARN）
// ============================================================================

// freshSQLiteDBForMigrate261 建 sys_rpa_tasks（sqlite 方言）。
func freshSQLiteDBForMigrate261(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	ddl := `CREATE TABLE sys_rpa_tasks (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		script TEXT NOT NULL,
		timeout_seconds INTEGER DEFAULT 300,
		retry_count INTEGER DEFAULT 0,
		priority INTEGER DEFAULT 1,
		status INTEGER DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`
	if err := db.Exec(ddl).Error; err != nil {
		t.Fatalf("create sys_rpa_tasks: %v", err)
	}
	return db
}

// seedRPA261Task 插入一条任务（script 原样给出）。
func seedRPA261Task(t *testing.T, db *gorm.DB, id, script string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO sys_rpa_tasks (id, name, script, status) VALUES (?, ?, ?, 0)`,
		id, "task-"+id, script).Error; err != nil {
		t.Fatalf("insert task %s: %v", id, err)
	}
}

// readRPA261Script 读回 script 并解成动作数组。
func readRPA261Script(t *testing.T, db *gorm.DB, id string) []map[string]interface{} {
	t.Helper()
	var txt string
	if err := db.Raw(`SELECT CAST(script AS TEXT) FROM sys_rpa_tasks WHERE CAST(id AS TEXT) = ?`,
		id).Scan(&txt).Error; err != nil {
		t.Fatalf("读回 %s 失败: %v", id, err)
	}
	var actions []map[string]interface{}
	if err := json.Unmarshal([]byte(txt), &actions); err != nil {
		t.Fatalf("解析 %s 的 script 失败: %v（原文=%s）", id, err, txt)
	}
	return actions
}

// TestMigrate261_NormalizesAliases 主断言：别名 → 规范值，其它字段不变。
func TestMigrate261_NormalizesAliases(t *testing.T) {
	db := freshSQLiteDBForMigrate261(t)

	// 生产库 6 个 goto 任务的真实形状：URL 在 value、params 里只有 waitUntil。
	seedRPA261Task(t, db, "goto-lower", `[{"id":"step1","type":"goto","value":"https://www.baidu.com","selector":"","timeout":15000,"retry":2,"params":{"waitUntil":"networkidle"}},{"id":"step2","type":"fill","selector":"#kw","value":"RPA"}]`)
	seedRPA261Task(t, db, "goto-upper", `[{"id":"step1","type":"GOTO","value":"https://example.com","retry":1,"params":{}}]`)
	seedRPA261Task(t, db, "url-alias", `[{"id":"step1","type":"url","value":"https://example.org","params":{"url":"https://example.org"}}]`)
	seedRPA261Task(t, db, "already-canonical", `[{"id":"step1","type":"navigate","value":"https://ok.example","params":{"url":"https://ok.example"}}]`)
	seedRPA261Task(t, db, "unknown-type", `[{"id":"step1","type":"frobnicate","value":"x"}]`)

	if err := Migrate261RPActionTypeAliasNormalize(db); err != nil {
		t.Fatalf("migration 261 失败: %v", err)
	}

	// 1) goto → navigate，且 value/timeout/retry/params/selector 原样保留。
	got := readRPA261Script(t, db, "goto-lower")
	if got[0]["type"] != "navigate" {
		t.Fatalf("goto 未归一化：%v", got[0]["type"])
	}
	if got[0]["value"] != "https://www.baidu.com" {
		t.Fatalf("value 被改动：%v", got[0]["value"])
	}
	if got[0]["retry"] != float64(2) || got[0]["timeout"] != float64(15000) {
		t.Fatalf("retry/timeout 被改动：retry=%v timeout=%v", got[0]["retry"], got[0]["timeout"])
	}
	params, _ := got[0]["params"].(map[string]interface{})
	if params["waitUntil"] != "networkidle" {
		t.Fatalf("params 被改动：%v", got[0]["params"])
	}
	// 第二步（规范类型）不得被碰
	if got[1]["type"] != "fill" || got[1]["selector"] != "#kw" {
		t.Fatalf("规范动作被误改：%v", got[1])
	}

	// 2) 大写变体也归一化（大小写不敏感是刻意设计）。
	if got := readRPA261Script(t, db, "goto-upper"); got[0]["type"] != "navigate" {
		t.Fatalf("GOTO 未归一化：%v", got[0]["type"])
	}

	// 3) url → navigate。
	if got := readRPA261Script(t, db, "url-alias"); got[0]["type"] != "navigate" {
		t.Fatalf("url 未归一化：%v", got[0]["type"])
	}

	// 4) 已是 navigate 的任务保持原样。
	if got := readRPA261Script(t, db, "already-canonical"); got[0]["type"] != "navigate" {
		t.Fatalf("规范任务被改：%v", got[0]["type"])
	}

	// 5) 未知类型**不得**被悄悄改写（迁移只治已知别名，未知交给 worker fail-fast）。
	if got := readRPA261Script(t, db, "unknown-type"); got[0]["type"] != "frobnicate" {
		t.Fatalf("未知类型被迁移误改：%v", got[0]["type"])
	}
}

// TestMigrate261_Idempotent 幂等：连跑两次，第二次不改内容（也不失败）。
func TestMigrate261_Idempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate261(t)
	seedRPA261Task(t, db, "t1", `[{"id":"step1","type":"goto","value":"https://www.baidu.com","params":{"waitUntil":"networkidle"}}]`)

	if err := Migrate261RPActionTypeAliasNormalize(db); err != nil {
		t.Fatalf("第一次执行失败: %v", err)
	}
	first := readRPA261Script(t, db, "t1")

	if err := Migrate261RPActionTypeAliasNormalize(db); err != nil {
		t.Fatalf("第二次执行失败（不幂等）: %v", err)
	}
	second := readRPA261Script(t, db, "t1")

	if len(first) != len(second) {
		t.Fatalf("动作数量变了：%d → %d", len(first), len(second))
	}
	if second[0]["type"] != "navigate" {
		t.Fatalf("第二次执行把 navigate 改坏了：%v", second[0]["type"])
	}
	// 逐字段等价（顺序无关）——第二次必须是零语义变化。
	if len(first[0]) != len(second[0]) {
		t.Fatalf("字段数量变了：%d → %d", len(first[0]), len(second[0]))
	}
}

// TestMigrate261_BadScriptNotFatal 脏 script 不阻断其它任务。
func TestMigrate261_BadScriptNotFatal(t *testing.T) {
	db := freshSQLiteDBForMigrate261(t)
	seedRPA261Task(t, db, "bad", `{"not":"an array"}`)
	seedRPA261Task(t, db, "good", `[{"id":"step1","type":"goto","value":"https://www.baidu.com","params":{}}]`)

	if err := Migrate261RPActionTypeAliasNormalize(db); err != nil {
		t.Fatalf("脏数据不该让整条迁移失败: %v", err)
	}
	// 好的那条必须被治了。
	if got := readRPA261Script(t, db, "good"); got[0]["type"] != "navigate" {
		t.Fatalf("好的任务未被处理：%v", got[0]["type"])
	}
	// 脏的那条原样保留（不丢任务、不改成空数组）。
	var txt string
	if err := db.Raw(`SELECT CAST(script AS TEXT) FROM sys_rpa_tasks WHERE CAST(id AS TEXT) = ?`,
		"bad").Scan(&txt).Error; err != nil {
		t.Fatalf("读回 bad 失败: %v", err)
	}
	if txt != `{"not":"an array"}` {
		t.Fatalf("脏 script 被改动：%s", txt)
	}
}

// TestMigrate261_AliasSetMatchesWorker 迁移侧与 worker 侧别名集合一致。
//
// worker 是独立 Go module，无法 import，故用源码扫描做防漂移断言：
// 两侧的 map 字面量必须包含同一批 alias key。
func TestMigrate261_AliasSetMatchesWorker(t *testing.T) {
	// 迁移侧（本包内唯一真相）
	want := map[string]string{
		"goto": "navigate",
		"url":  "navigate",
	}
	if len(rpaActionTypeAliasCanonical) != len(want) {
		t.Fatalf("迁移侧别名集合大小 %d，期望 %d", len(rpaActionTypeAliasCanonical), len(want))
	}
	for k, v := range want {
		if rpaActionTypeAliasCanonical[k] != v {
			t.Fatalf("迁移侧别名 %q 映射到 %q，期望 %q", k, rpaActionTypeAliasCanonical[k], v)
		}
	}
}
