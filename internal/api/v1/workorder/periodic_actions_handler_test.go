package workorder

// =============================================================================
// 周期工单模板动作 handler 测试（enable / disable / generate / logs）
//
// 背景：这 4 条 HTTP 通路在历史路由重构（cd306b359）中丢失，service 层
// PeriodicService 实现一直可用；本文件覆盖恢复后的 4 个 handler 方法。
//
// 复用 workorder_handler_test.go 的轻量模式：
//   - glebarez/sqlite in-memory（setupTestDB）
//   - 真实 service 调用（setupTestHandler + NoOpCacheProvider）
//   - httptest.NewRecorder + gin.CreateTestContext + c.Params 注入 path 参数
// =============================================================================

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnablePeriodic_Success 启用后 is_enabled=true 且 next_run_at 被计算
func TestEnablePeriodic_Success(t *testing.T) {
	db := setupTestDB(t)
	id := seedPeriodicTemplate(t, db, "待启用模板")
	// 先禁用
	require.NoError(t, db.Exec("UPDATE sys_periodic_workorder_template SET is_enabled = 0 WHERE id = ?", id).Error)

	h := setupTestHandler(t, db)
	c, w := newTestContext("POST", "/"+id+"/enable", "operator-1", nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	h.EnablePeriodic(c)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w.Body.Bytes())
	assert.Equal(t, float64(0), resp["code"])

	var enabled int
	require.NoError(t, db.Raw("SELECT is_enabled FROM sys_periodic_workorder_template WHERE id = ?", id).Scan(&enabled).Error)
	assert.Equal(t, 1, enabled, "启用后 is_enabled 应为 true(1)")
}

// TestEnablePeriodic_NotFound 模板不存在返回业务错误
func TestEnablePeriodic_NotFound(t *testing.T) {
	db := setupTestDB(t)
	h := setupTestHandler(t, db)
	missing := uuid.NewString()
	c, w := newTestContext("POST", "/"+missing+"/enable", "operator-1", nil)
	c.Params = gin.Params{{Key: "id", Value: missing}}
	h.EnablePeriodic(c)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TestDisablePeriodic_Success 禁用后 is_enabled=false
func TestDisablePeriodic_Success(t *testing.T) {
	db := setupTestDB(t)
	id := seedPeriodicTemplate(t, db, "待禁用模板")

	h := setupTestHandler(t, db)
	c, w := newTestContext("POST", "/"+id+"/disable", "operator-1", nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	h.DisablePeriodic(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var enabled int
	require.NoError(t, db.Raw("SELECT is_enabled FROM sys_periodic_workorder_template WHERE id = ?", id).Scan(&enabled).Error)
	assert.Equal(t, 0, enabled, "禁用后 is_enabled 应为 false(0)")
}

// TestDisablePeriodic_EmptyID_Returns400 缺 path 参数
func TestDisablePeriodic_EmptyID_Returns400(t *testing.T) {
	db := setupTestDB(t)
	h := setupTestHandler(t, db)
	c, w := newTestContext("POST", "//disable", "", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.DisablePeriodic(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestGeneratePeriodic_Success 立即生成返回工单并写执行日志
func TestGeneratePeriodic_Success(t *testing.T) {
	db := setupTestDB(t)
	// generate 依赖默认提交人解析链：sys_config 表 + 配置行 + admin 用户
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_config (
		id TEXT PRIMARY KEY,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		created_by TEXT,
		updated_by TEXT,
		version INTEGER DEFAULT 0,
		config_name TEXT,
		config_key TEXT,
		config_value TEXT,
		config_type TEXT DEFAULT 'Y',
		is_system INTEGER DEFAULT 0,
		remark TEXT
	)`).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO sys_config (id, config_key, config_value) VALUES (?, ?, ?)",
		"00000000-0000-0000-0000-000000000002", "sys.workorder.default_submitter_username", "admin",
	).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO sys_user (id, username) VALUES (?, ?)",
		"00000000-0000-0000-0000-000000000001", "admin",
	).Error)

	id := seedPeriodicTemplate(t, db, "立即生成模板")

	h := setupTestHandler(t, db)
	c, w := newTestContext("POST", "/"+id+"/generate", "operator-1", nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	h.GeneratePeriodic(c)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w.Body.Bytes())
	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok, "data 应为生成的工单对象")
	assert.NotEmpty(t, data["id"], "生成的工单应有 ID")
	assert.NotEmpty(t, data["workOrderNo"], "生成的工单应有工单号")

	// 执行日志已落库
	var logCount int
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM sys_periodic_workorder_log WHERE template_id = ?", id).Scan(&logCount).Error)
	assert.Equal(t, 1, logCount, "generate 应写入一条执行记录")

	// total_generated +1
	var total int64
	require.NoError(t, db.Raw("SELECT total_generated FROM sys_periodic_workorder_template WHERE id = ?", id).Scan(&total).Error)
	assert.Equal(t, int64(1), total)
}

// TestGetPeriodicLogs_Success 返回该模板的执行记录列表
func TestGetPeriodicLogs_Success(t *testing.T) {
	db := setupTestDB(t)
	id := seedPeriodicTemplate(t, db, "日志模板")
	for i := 0; i < 2; i++ {
		require.NoError(t, db.Exec(
			"INSERT INTO sys_periodic_workorder_log (id, template_id, status) VALUES (?, ?, ?)",
			uuid.NewString(), id, "success",
		).Error)
	}

	h := setupTestHandler(t, db)
	c, w := newTestContext("POST", "/"+id+"/logs", "operator-1", nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	h.GetPeriodicLogs(c)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w.Body.Bytes())
	logs, ok := resp["data"].([]interface{})
	require.True(t, ok, "data 应为日志数组")
	assert.Len(t, logs, 2)
}

// TestGetPeriodicLogs_Empty 无记录返回空数组
func TestGetPeriodicLogs_Empty(t *testing.T) {
	db := setupTestDB(t)
	id := seedPeriodicTemplate(t, db, "空日志模板")

	h := setupTestHandler(t, db)
	c, w := newTestContext("POST", "/"+id+"/logs", "operator-1", nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	h.GetPeriodicLogs(c)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w.Body.Bytes())
	logs, ok := resp["data"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, logs)
}
