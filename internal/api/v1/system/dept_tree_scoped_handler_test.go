package system

// dept_tree_scoped_handler_test.go —— 部门树 scoped 收口的 handler 级端到端断言。
//
// 复刻生产场景（刘洋）：请求体 {scoped:true} + ctx 注入了中支锚点 ScopeResult
// → GetTree handler → 响应树必须只包含武汉中支子树 + 祖先路径。
//
// 本测试把「JSON body 绑定 scoped」「ctx ScopeResult 读取」「树过滤」三段串起来，
// 任何一段断了都会红——这是比单测 filterDeptTreeByScope 更强的端到端保证。

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
)

func TestGetTree_Scoped_WithAnchorCtx_ReturnsScopedTree(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	if sqlDB, e := db.DB(); e == nil {
		defer sqlDB.Close()
	}

	// 建表（含 org_level 列）
	require.NoError(t, db.Exec(`CREATE TABLE sys_dept (
		id TEXT PRIMARY KEY, dept_name TEXT, dept_code TEXT, parent_id TEXT, ancestors TEXT DEFAULT '',
		org_level INTEGER DEFAULT 0, is_external_org INTEGER DEFAULT 0, status INTEGER DEFAULT 0,
		order_num INTEGER DEFAULT 0, leader TEXT, phone TEXT, email TEXT, remark TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, created_by TEXT, updated_by TEXT)`).Error)

	// 树：湖北分公司 → 武汉中支 → （综合管理部、硚口支公司）；长沙中支（兄弟）
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id,dept_name,dept_code,parent_id,ancestors,org_level,status) VALUES
		('D-hb','湖北分公司','HB',NULL,'',1,0),
		('D-wh','武汉中心支公司','WH','D-hb','D-hb',2,0),
		('D-zh','综合管理部','ZH','D-wh','D-hb,D-wh',0,0),
		('D-qk','硚口支公司','QK','D-wh','D-hb,D-wh',3,0),
		('D-cs','长沙中心支公司','CS','D-hb','D-hb',2,0)`).Error)

	// 构造 handler（复用 setupDeptHandler 的构造方式，但用我们的 db）
	svc := systemServices.NewDepartmentService(db)
	h := &DepartmentHandler{service: svc}

	// 构造 gin.Context：JSON body {scoped:true} + ctx 注入锚点 ScopeResult
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/system/departments/tree",
		bytes.NewBufferString(`{"scoped":true}`))
	c.Request.Header.Set("Content-Type", "application/json")
	ctx := datascope.WithContext(c.Request.Context(), datascope.ScopeResult{
		DeptIDs: []string{"D-wh", "D-zh", "D-qk"}, // 武汉中支子树（锚点展开结果）
		UserID:  "U-liu",
	})
	c.Request = c.Request.WithContext(ctx)

	h.GetTree(c)

	require.Equal(t, 200, w.Code, "response code")
	var resp struct {
		Code int                  `json:"code"`
		Data []*models.Department `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)

	ids := map[string]bool{}
	var walk func([]*models.Department)
	walk = func(nodes []*models.Department) {
		for _, n := range nodes {
			ids[n.ID] = true
			walk(n.Children)
		}
	}
	walk(resp.Data)

	assert.True(t, ids["D-hb"], "祖先路径保留")
	assert.True(t, ids["D-wh"], "武汉中支可见")
	assert.True(t, ids["D-qk"], "硚口支公司可见")
	assert.False(t, ids["D-cs"], "长沙中支必须不可见（收口核心）")
	t.Logf("scoped 树节点: %v", ids)
}
