package rpa

// ============================================================================
// PR #114 CI 覆盖率补齐：定时调度 HTTP 端点契约（此前 schedule_handler.go 15.8%）
//
// 走真实 SetupRPARouter（真 sqlite + 真 AutoMigrate + 真 RBAC），因此同时锁住
// 路由挂载、权限码接线、参数校验与错误码分流。
// ============================================================================

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
)

// seedScheduleTask 播种一条 RPA 任务（调度外键需要真实 taskId）。
func seedScheduleTask(t *testing.T, database *db.Database) string {
	t.Helper()
	task := &rpamodels.Task{
		TaskName: "调度契约用任务",
		Script:   []byte("[]"),
		Status:   rpamodels.TaskStatusEnabled,
	}
	require.NoError(t, database.DB.Create(task).Error, "播种任务失败")
	return task.ID
}

// TestHandler_Schedule_CRUD 创建 → 详情 → 列表 → 更新 → 启停 → 删除 全链路。
func TestHandler_Schedule_CRUD(t *testing.T) {
	engine, c := newRPARouteCore(t, "h-sch-1")
	database := c.GetDB()
	taskID := seedScheduleTask(t, &db.Database{DB: database})

	// 创建（每分钟）
	code, resp := doRoute(t, engine, http.MethodPost, "/api/v1/rpa/schedules", map[string]interface{}{
		"name":     "契约-每分钟",
		"taskId":   taskID,
		"cronExpr": "* * * * *",
		"timezone": "Asia/Shanghai",
	})
	require.Equal(t, http.StatusOK, code, "创建调度必须 200: %v", resp)
	created := respData(resp)
	id, _ := created["id"].(string)
	require.NotEmpty(t, id, "创建必须返回 id")
	require.Equal(t, "契约-每分钟", created["name"])
	require.EqualValues(t, 0, created["status"], "默认启用")
	require.NotNil(t, created["nextRunTime"], "创建即算下次执行时间")

	// 详情
	code, resp = doRoute(t, engine, http.MethodPost, "/api/v1/rpa/schedules/"+id, nil)
	require.Equal(t, http.StatusOK, code, "详情必须 200: %v", resp)
	require.Equal(t, id, respData(resp)["id"])

	// 列表
	code, resp = doRoute(t, engine, http.MethodPost, "/api/v1/rpa/schedules/list", map[string]interface{}{})
	require.Equal(t, http.StatusOK, code, "列表必须 200: %v", resp)
	require.GreaterOrEqual(t, respTotal(t, resp), int64(1), "列表应含刚创建的一条")

	// 更新（改名 + 改 cron）
	code, resp = doRoute(t, engine, http.MethodPost, "/api/v1/rpa/schedules/"+id+"/update", map[string]interface{}{
		"name":     "契约-每小时",
		"cronExpr": "0 * * * *",
	})
	require.Equal(t, http.StatusOK, code, "更新必须 200: %v", resp)
	var name string
	require.NoError(t, database.Table("sys_rpa_schedules").Where("id = ?", id).Pluck("name", &name).Error)
	require.Equal(t, "契约-每小时", name, "更新必须落库")

	// 启停
	code, resp = doRoute(t, engine, http.MethodPost, "/api/v1/rpa/schedules/"+id+"/status", map[string]interface{}{
		"status": 1,
	})
	require.Equal(t, http.StatusOK, code, "停用必须 200: %v", resp)
	var status int
	require.NoError(t, database.Table("sys_rpa_schedules").Where("id = ?", id).Pluck("status", &status).Error)
	require.Equal(t, 1, status, "停用必须落库")

	// 删除
	code, resp = doRoute(t, engine, http.MethodPost, "/api/v1/rpa/schedules/"+id+"/delete", nil)
	require.Equal(t, http.StatusOK, code, "删除必须 200: %v", resp)
}

// TestHandler_Schedule_Create_Invalid400 三类必填缺失/非法 cron 一律 400 且文案可读。
func TestHandler_Schedule_Create_Invalid400(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-sch-2")

	// 缺 name
	code, resp := doRoute(t, engine, http.MethodPost, "/api/v1/rpa/schedules", map[string]interface{}{
		"taskId":   "t-not-exist",
		"cronExpr": "* * * * *",
	})
	require.Equal(t, http.StatusBadRequest, code, "缺 name 必须 400: %v", resp)

	// 非法 cron
	code, resp = doRoute(t, engine, http.MethodPost, "/api/v1/rpa/schedules", map[string]interface{}{
		"name":     "坏表达式",
		"taskId":   "t-not-exist",
		"cronExpr": "not-a-cron",
	})
	require.Equal(t, http.StatusBadRequest, code, "非法 cron 必须 400: %v", resp)

	// 空 cron
	code, resp = doRoute(t, engine, http.MethodPost, "/api/v1/rpa/schedules", map[string]interface{}{
		"name":   "空表达式",
		"taskId": "t-not-exist",
	})
	require.Equal(t, http.StatusBadRequest, code, "空 cron 必须 400: %v", resp)
}

// TestHandler_Schedule_List_FilterByStatus 列表 status 筛选（0/1 合法，2 非法）。
func TestHandler_Schedule_List_FilterByStatus(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-sch-3")

	code, resp := doRoute(t, engine, http.MethodPost, "/api/v1/rpa/schedules/list", map[string]interface{}{
		"status": 0,
	})
	require.Equal(t, http.StatusOK, code, "status=0 筛选必须 200: %v", resp)

	code, resp = doRoute(t, engine, http.MethodPost, "/api/v1/rpa/schedules/list", map[string]interface{}{
		"status": 2,
	})
	require.Equal(t, http.StatusBadRequest, code, "status=2 必须 400（取值域只接受 0/1）: %v", resp)
}

// TestHandler_Schedule_GetByID_NotFound404 不存在的 id。
func TestHandler_Schedule_GetByID_NotFound404(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-sch-4")
	code, resp := doRoute(t, engine, http.MethodPost,
		"/api/v1/rpa/schedules/00000000-0000-0000-0000-000000000000", nil)
	require.EqualValues(t, http.StatusNotFound, respCodeOf(resp), "不存在的调度必须语义码 404: %v", resp)
	_ = code
}

// TestHandler_Schedule_Update_NotFound400 更新不存在的 id。
func TestHandler_Schedule_Update_NotFound400(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-sch-5")
	code, resp := doRoute(t, engine, http.MethodPost,
		"/api/v1/rpa/schedules/00000000-0000-0000-0000-000000000000/update",
		map[string]interface{}{"name": "改不存在"})
	require.Equal(t, http.StatusBadRequest, code, "更新不存在必须 400: %v", resp)
}
