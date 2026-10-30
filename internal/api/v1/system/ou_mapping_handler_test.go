package system

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
)

// Phase 62 (HYGIENE-05) 手工 OU 重绑路径测试 —— 原 PLAN 记录"手工 API 路径无测试覆盖"，
// 本文件补齐 handler 层：预检（GetMappingByOU）不阻断 + UpsertMappingForce 真正覆盖。

func setupOUMappingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY,
			dept_name TEXT NOT NULL,
			dept_code TEXT,
			parent_id TEXT,
			ancestors TEXT DEFAULT '',
			order_num INTEGER DEFAULT 0,
			leader TEXT,
			phone TEXT,
			email TEXT,
			is_external_org INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			remark TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept_ou_mapping (
			id TEXT PRIMARY KEY,
			dept_id TEXT NOT NULL,
			ad_config_id TEXT NOT NULL,
			ou_dn TEXT NOT NULL,
			ou_name TEXT NOT NULL,
			parent_ou_dn TEXT,
			sync_enabled INTEGER DEFAULT 1,
			sync_status TEXT DEFAULT 'pending',
			last_sync_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			UNIQUE(dept_id, ad_config_id),
			UNIQUE(ad_config_id, ou_dn)
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_config (
			id TEXT PRIMARY KEY,
			config_name TEXT,
			sync_enabled INTEGER DEFAULT 1,
			status INTEGER DEFAULT 0,
			deleted_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error)

	return db
}

// seedTwoDepts 预置两个部门 + 一个启用中的 AD 配置 + 一条 OU=TestOU 映射到 dept-A。
func seedTwoDepts(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec("INSERT INTO sys_dept (id, dept_name, parent_id) VALUES (?,?,?), (?,?,?)",
		"dept-A", "财务部", nil, "dept-B", "人资部", nil).Error)
	require.NoError(t, db.Exec("INSERT INTO sys_ad_config (id, config_name, sync_enabled, status) VALUES (?,?,?,?)",
		"cfg-1", "AD域控主机", true, models.ADConfigStatusEnabled).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept_ou_mapping
		(id, dept_id, ad_config_id, ou_dn, ou_name, sync_enabled, sync_status)
		VALUES ('map-1', 'dept-A', 'cfg-1', 'OU=TestOU,DC=corp', 'TestOU', 1, 'synced')`).Error)
}

// TestUpdateOUDeptMapping_ForceRebindReplacesExisting 验证手工重绑走 Force：
// 原映射被替换为新部门（而不是返回冲突错误），且不产生第二行。
func TestUpdateOUDeptMapping_ForceRebindReplacesExisting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOUMappingTestDB(t)
	seedTwoDepts(t, db)

	mapper := addomain.NewDeptOUmapper(db)
	h := NewOUMappingHandler(db, mapper) // core 为 nil：recordOperLog 自带 nil 守卫

	body := `{"deptId":"dept-B"}`
	req := httptest.NewRequest(http.MethodPost, "/system/ou/OU=TestOU,DC=corp/dept-mapping", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ouDn", Value: "OU=TestOU,DC=corp"}}

	h.UpdateOUDeptMapping(c)

	assert.Equal(t, http.StatusOK, w.Code, "手工重绑应成功: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "关联成功")

	var rows []models.DeptOUMapping
	require.NoError(t, db.Where("ad_config_id = ? AND ou_dn = ?", "cfg-1", "OU=TestOU,DC=corp").Find(&rows).Error)
	require.Len(t, rows, 1, "Force 覆盖后同一 OU 只应有一行映射")
	assert.Equal(t, "dept-B", rows[0].DeptID, "映射应切换到新部门")
}

// TestUpdateOUDeptMapping_MissingDeptRejected 验证参数校验：部门不存在时 400，且不落映射。
func TestUpdateOUDeptMapping_MissingDeptRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOUMappingTestDB(t)
	seedTwoDepts(t, db)

	mapper := addomain.NewDeptOUmapper(db)
	h := NewOUMappingHandler(db, mapper)

	body := `{"deptId":"dept-does-not-exist"}`
	req := httptest.NewRequest(http.MethodPost, "/system/ou/OU=TestOU,DC=corp/dept-mapping", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "ouDn", Value: "OU=TestOU,DC=corp"}}

	h.UpdateOUDeptMapping(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var mapping models.DeptOUMapping
	err := db.Where("ad_config_id = ? AND ou_dn = ?", "cfg-1", "OU=TestOU,DC=corp").First(&mapping).Error
	require.NoError(t, err)
	assert.Equal(t, "dept-A", mapping.DeptID, "校验失败不得改动原映射")
}
