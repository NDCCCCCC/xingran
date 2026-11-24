package rpa

// Phase 153 / RUNTIME-03 / RUNTIME-04 handler 契约用例。
//
// 遵循 Phase 152 确立的样板（contract_p0_smoke_test.go）：httptest +
// 真实 sqlite 库（db.NewDatabase → AutoMigrate）+ 真实 service，不 mock
// 内部服务。Redis 不可达的场景下验证「降级可读」语义；真 Redis 全链路
// 用例见 runtime_p153_redis_test.go（环境可达时自动执行）。

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// fakeNoRedisCache 实现 cache.Cache（嵌入 nil 接口兜底），但**不提供**
// ExportClient / GetRedisClient —— publish 断言应返回可读错误而非 panic。
type fakeNoRedisCache struct {
	cache.Cache
}

// TestRuntime03_Cancel_Handler_SQLiteRealTables 取消入口语义加固：
// pending → cancel 200 + 终态留痕；已终态 → 状态机拒绝。
func TestRuntime03_Cancel_Handler_SQLiteRealTables(t *testing.T) {
	c, database := contractP0Core(t)
	const userID = "runtime-p153-cancel-user"
	seedContractP0Admin(t, c, userID)
	engine := contractP0Engine(t, c, userID)

	// 播种 pending 执行
	execA := seedRuntimeExecution(t, database, "pending")
	execB := seedRuntimeExecution(t, database, "success")

	// A: pending → cancelled（无 body 时也应成功）
	code, resp := doContractP0(t, engine, "/api/v1/rpa/executions/"+execA+"/cancel", nil)
	if code != 200 {
		t.Fatalf("pending 取消应 200，实际 %d: %v", code, respMessage(resp))
	}
	var status, logs string
	if err := database.DB.Raw(
		`SELECT status, COALESCE(logs,'') FROM sys_rpa_executions WHERE id = ?`, execA,
	).Row().Scan(&status, &logs); err != nil {
		t.Fatalf("查询执行记录失败: %v", err)
	}
	if status != "cancelled" {
		t.Fatalf("终态应为 cancelled，实际 %s", status)
	}
	if !containsAll(logs, userID, "runtime-cancel-reason") && !containsAll(logs, "cancelled by "+userID) {
		t.Fatalf("取消留痕应包含操作者（谁），实际 logs: %s", logs)
	}

	// A2: 携带 reason → 留痕包含原因
	execC := seedRuntimeExecution(t, database, "running")
	code, resp = doContractP0(t, engine, "/api/v1/rpa/executions/"+execC+"/cancel",
		map[string]interface{}{"reason": "运行参数有误"})
	if code != 200 {
		t.Fatalf("running 取消应 200，实际 %d: %v", code, respMessage(resp))
	}
	if err := database.DB.Raw(
		`SELECT COALESCE(logs,'') FROM sys_rpa_executions WHERE id = ?`, execC,
	).Row().Scan(&logs); err != nil {
		t.Fatalf("查询 logs 失败: %v", err)
	}
	if !containsAll(logs, userID, "运行参数有误") {
		t.Fatalf("留痕应包含谁/原因（审计要求），实际: %s", logs)
	}

	// B: success 终态 → 状态机拒绝（非 200 且错误明确）
	code, resp = doContractP0(t, engine, "/api/v1/rpa/executions/"+execB+"/cancel", nil)
	if code == 200 {
		t.Fatalf("终态执行取消应被拒绝，实际 200")
	}
	if msg := respMessage(resp); !containsAll(msg, "无法取消", "终态不可覆盖") {
		t.Fatalf("拒绝语义应可读，实际: %s", msg)
	}
}

// TestRuntime04_SubmitIntervention_PublishTarget RUNTIME-04：
// publish 落点为 SubmitHumanIntervention（h.core.Cache），cache 缺失/不支持
// 时返回可读错误（不静默、不经 excel_service）。
func TestRuntime04_SubmitIntervention_PublishTarget(t *testing.T) {
	c, database := contractP0Core(t)
	const userID = "runtime-p153-intervene-user"
	seedContractP0Admin(t, c, userID)
	engine := contractP0Engine(t, c, userID)
	execID := seedRuntimeExecution(t, database, "paused")

	body := map[string]interface{}{
		"executionId": execID,
		"action":      "resume",
		"input":       map[string]interface{}{"captcha": "1234"},
		"reason":      "人工确认验证码",
	}

	// 场景 1：core.Cache 未注入 → 明确 500（通道未配置）
	c.Cache = nil
	code, resp := doContractP0(t, engine, "/api/v1/rpa/executions/"+execID+"/human-intervention", body)
	if code == 200 {
		t.Fatalf("cache 未注入时应拒绝，实际 200")
	}
	if msg := respMessage(resp); !containsAll(msg, "人工干预通道未配置") {
		t.Fatalf("错误应可读，实际: %s", msg)
	}

	// 事件必须已落 DB（审计留痕：谁提交了干预）
	var count int64
	if err := database.DB.Raw(
		`SELECT COUNT(*) FROM sys_rpa_human_interventions WHERE execution_id = ? AND action = 'resume'`, execID,
	).Scan(&count).Error; err != nil {
		t.Fatalf("查询干预事件失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("干预事件应已落库，实际 %d 条", count)
	}

	// 场景 2：cache 不支持 Pub/Sub（无 Redis 后端）→ publish 失败可读
	c.Cache = &fakeNoRedisCache{}
	code, resp = doContractP0(t, engine, "/api/v1/rpa/executions/"+execID+"/human-intervention", body)
	if code == 200 {
		t.Fatalf("publish 失败时不应返回 200（干预未到达 Worker，pause 会超时而前端不知情）")
	}
	if msg := respMessage(resp); !containsAll(msg, "人工干预下发失败") {
		t.Fatalf("publish 失败语义应可读，实际: %s", msg)
	}
}

// TestRuntime03_Cancel_DegradedWhenCacheUnsupported publish 降级语义：
// DB 终态已落，Redis 不可用时 cancel 仍然 200（终态不被回滚）。
func TestRuntime03_Cancel_DegradedWhenCacheUnsupported(t *testing.T) {
	c, database := contractP0Core(t)
	const userID = "runtime-p153-degrade-user"
	seedContractP0Admin(t, c, userID)
	engine := contractP0Engine(t, c, userID)
	execID := seedRuntimeExecution(t, database, "running")

	c.Cache = &fakeNoRedisCache{}
	code, _ := doContractP0(t, engine, "/api/v1/rpa/executions/"+execID+"/cancel",
		map[string]interface{}{"reason": "降级用例"})
	if code != 200 {
		t.Fatalf("Redis 不可用时取消应降级成功（DB 终态优先），实际 %d", code)
	}
	var status string
	if err := database.DB.Raw(
		`SELECT status FROM sys_rpa_executions WHERE id = ?`, execID,
	).Row().Scan(&status); err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if status != "cancelled" {
		t.Fatalf("终态应为 cancelled，实际 %s", status)
	}
}

// ---- helpers ----

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

// seedRuntimeExecution 用真实 GORM 模型播种一条执行记录，返回 ID。
func seedRuntimeExecution(t *testing.T, database *db.Database, status string) string {
	t.Helper()
	exec := &rpamodels.Execution{
		TaskID:      "00000000-0000-0000-0000-000000000001",
		TaskName:    "runtime-p153-task",
		Status:      status,
		TriggeredBy: "runtime-p153-test",
		TriggerType: "manual",
	}
	require.NoError(t, database.DB.Create(exec).Error)
	return exec.ID
}
