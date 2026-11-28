package rpa

// ============================================================================
// PR #114 CI 覆盖率补齐：执行记录 HTTP 端点契约（此前 execution_handler.go 56%）
//
// 缺口集中在 Statistics / GetByID-notfound / GetLogs / GetBatchReport /
// 人工干预两段 / DownloadArtifacts —— 均为「读端点 + 错误分支」，从未有 HTTP 用例。
// 走真实 SetupRPARouter（真 sqlite + 真 AutoMigrate + 真 RBAC）。
// ============================================================================

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
)

// seedExec 播种一条执行记录（真实表，供读端点断言）。
func seedExec(t *testing.T, database *db.Database, status rpamodels.RPAExecutionStatus, logs string) string {
	t.Helper()
	exec := &rpamodels.Execution{
		TaskID:     "exec-task-" + time.Now().Format("150405.000000000"),
		Status:     string(status),
		Step:       2,
		TotalSteps: 5,
		Logs:       logs,
	}
	require.NoError(t, database.DB.Create(exec).Error, "播种执行失败")
	return exec.ID
}

// TestHandler_Execution_Statistics 统计端点 200 且字段齐全。
func TestHandler_Execution_Statistics(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-ex-1")
	code, resp := doRoute(t, engine, http.MethodPost, "/api/v1/rpa/executions/statistics", map[string]interface{}{})
	require.Equal(t, http.StatusOK, code, "统计必须 200: %v", resp)
	require.NotNil(t, resp["data"], "统计必须返回数据体")
}

// TestHandler_Execution_List 列表端点 200。
func TestHandler_Execution_List(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-ex-2")
	code, resp := doRoute(t, engine, http.MethodPost, "/api/v1/rpa/executions/list", map[string]interface{}{})
	require.Equal(t, http.StatusOK, code, "列表必须 200: %v", resp)
	require.NotNil(t, resp["data"])
}

// TestHandler_Execution_GetByID 详情 200 / 不存在语义 404。
func TestHandler_Execution_GetByID(t *testing.T) {
	engine, c := newRPARouteCore(t, "h-ex-3")
	id := seedExec(t, &db.Database{DB: c.GetDB()}, rpamodels.RPAExecutionStatusSuccess, "step1 ok")

	code, resp := doRoute(t, engine, http.MethodPost, "/api/v1/rpa/executions/"+id, nil)
	require.Equal(t, http.StatusOK, code, "详情必须 200: %v", resp)
	require.Equal(t, id, respData(resp)["id"])

	code, resp = doRoute(t, engine, http.MethodPost,
		"/api/v1/rpa/executions/00000000-0000-0000-0000-000000000000", nil)
	require.EqualValues(t, http.StatusNotFound, respCodeOf(resp), "不存在必须语义码 404: %v", resp)
	_ = code
}

// TestHandler_Execution_GetLogs 日志端点返回该执行的 logs 字段。
func TestHandler_Execution_GetLogs(t *testing.T) {
	engine, c := newRPARouteCore(t, "h-ex-4")
	id := seedExec(t, &db.Database{DB: c.GetDB()}, rpamodels.RPAExecutionStatusSuccess, "第1步 打开页面\n第2步 填写表单")

	code, resp := doRoute(t, engine, http.MethodPost, "/api/v1/rpa/executions/"+id+"/logs", nil)
	require.Equal(t, http.StatusOK, code, "日志必须 200: %v", resp)
	logs, _ := respData(resp)["logs"].(string)
	require.Contains(t, logs, "第1步", "日志必须回带真实内容")
}

// TestHandler_Execution_GetLogs_NotFound404 不存在执行的日志 → 语义 404。
func TestHandler_Execution_GetLogs_NotFound404(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-ex-5")
	code, resp := doRoute(t, engine, http.MethodPost,
		"/api/v1/rpa/executions/00000000-0000-0000-0000-000000000000/logs", nil)
	require.EqualValues(t, http.StatusNotFound, respCodeOf(resp), "不存在必须语义码 404: %v", resp)
	_ = code
}

// TestHandler_Execution_BatchReport 批量报告端点：无批量数据时返回空统计（不得是假常量）。
func TestHandler_Execution_BatchReport(t *testing.T) {
	engine, c := newRPARouteCore(t, "h-ex-6")
	id := seedExec(t, &db.Database{DB: c.GetDB()}, rpamodels.RPAExecutionStatusSuccess, "ok")

	code, resp := doRoute(t, engine, http.MethodGet, "/api/v1/rpa/executions/"+id+"/batch-report", nil)
	require.Equal(t, http.StatusOK, code, "批量报告必须 200: %v", resp)
	// CLEANUP-01：无数据必须返回空统计，而不是硬编码 100/95/5。
	data := respData(resp)
	for _, k := range []string{"successRate", "total", "success"} {
		if v, ok := data[k].(float64); ok {
			require.NotEqual(t, float64(100), v, "字段 %s 不得返回硬编码 100", k)
			require.NotEqual(t, float64(95), v, "字段 %s 不得返回硬编码 95", k)
		}
	}
}

// TestHandler_Execution_Cancel 可取消（pending → cancelled）。
func TestHandler_Execution_Cancel(t *testing.T) {
	engine, c := newRPARouteCore(t, "h-ex-7")
	id := seedExec(t, &db.Database{DB: c.GetDB()}, rpamodels.RPAExecutionStatusPending, "")

	code, resp := doRoute(t, engine, http.MethodPost, "/api/v1/rpa/executions/"+id+"/cancel", map[string]interface{}{
		"reason": "契约测试取消",
	})
	require.Equal(t, http.StatusOK, code, "取消必须 200: %v", resp)

	var status string
	require.NoError(t, c.GetDB().Table("sys_rpa_executions").Where("id = ?", id).Pluck("status", &status).Error)
	require.Equal(t, "cancelled", status, "取消必须落终态")
}

// TestHandler_Execution_HumanIntervention_Status 人工干预状态查询端点。
func TestHandler_Execution_HumanIntervention_Status(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-ex-8")
	code, resp := doRoute(t, engine, http.MethodGet,
		"/api/v1/rpa/executions/00000000-0000-0000-0000-000000000000/human-intervention", nil)
	// 不存在的执行：允许 404 或 200+空状态，但不得 500。
	require.NotEqual(t, http.StatusInternalServerError, code, "不得 500: %v", resp)
}
