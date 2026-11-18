package system

// dept_cache_scoped_test.go —— 缓存装饰器下 scoped 收口的回归（生产真因）。
//
// 生产真因：departmentCacheService.GetTreeWithFilter 对普通树请求（scoped=true 且
// 无 DeptName/Status）走缓存 key=dept:tree[:all] + queryTree（全量、无过滤），
// 从未调用 filterDeptTreeByScope —— 所以 handler 有 scoped:true 日志却没有
// 过滤器日志，响应全量。本测试锁死修复：scoped 请求必须委托底层走收口。

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
)

func TestDeptCacheService_ScopedRequest_UsesFilter(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	if sqlDB, e := db.DB(); e == nil {
		defer sqlDB.Close()
	}

	require.NoError(t, db.Exec(`CREATE TABLE sys_dept (
		id TEXT PRIMARY KEY, dept_name TEXT, dept_code TEXT, parent_id TEXT, ancestors TEXT DEFAULT '',
		org_level INTEGER DEFAULT 0, is_external_org INTEGER DEFAULT 0, status INTEGER DEFAULT 0,
		order_num INTEGER DEFAULT 0, leader TEXT, phone TEXT, email TEXT, remark TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, created_by TEXT, updated_by TEXT, version INTEGER DEFAULT 0)`).Error)
	// 用 setupDeptCacheDB 同款结构（created_at 等），直接插入
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id,dept_name,dept_code,parent_id,ancestors,org_level,status) VALUES
		('D-hb','湖北分公司','HB',NULL,'',1,0),
		('D-wh','武汉中支','WH','D-hb','D-hb',2,0),
		('D-zh','综合部','ZH','D-wh','D-hb,D-wh',0,0),
		('D-qk','硚口支公司','QK','D-wh','D-hb,D-wh',3,0),
		('D-cs','长沙中支','CS','D-hb','D-hb',2,0)`).Error)

	// 用缓存装饰器（生产 DI 形态）
	svc := NewDepartmentServiceWithCache(db, &NoOpCacheProvider{}, nil)

	// ctx 注入中支锚点可见集（模拟 DataScopePermission 的输出）
	ctx := datascope.WithContext(context.Background(), datascope.ScopeResult{
		DeptIDs: []string{"D-wh", "D-zh", "D-qk"},
		UserID:  "U-liu",
	})
	params := requests.DepartmentListParams{Scoped: true}

	tree, err := svc.GetTreeWithFilter(ctx, true, params)
	require.NoError(t, err)

	ids := map[string]bool{}
	var walk func([]*models.Department)
	walk = func(nodes []*models.Department) {
		for _, n := range nodes {
			ids[n.ID] = true
			walk(n.Children)
		}
	}
	walk(tree)

	// 修复前：装饰器走 queryTree 全量 → D-cs 可见 → 本断言失败。
	// 修复后：委托底层 filterDeptTreeByScope → D-cs 不可见。
	assert.True(t, ids["D-hb"], "祖先路径保留")
	assert.True(t, ids["D-wh"], "武汉中支可见")
	assert.True(t, ids["D-qk"], "硚口支公司可见")
	assert.False(t, ids["D-cs"], "★核心：长沙中支必须不可见（证明走了收口而非缓存全量树）")
	t.Logf("缓存装饰器 scoped 树节点: %v", ids)
}

func TestDeptCacheService_NoScoped_StillFullTree(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	if sqlDB, e := db.DB(); e == nil {
		defer sqlDB.Close()
	}
	require.NoError(t, db.Exec(`CREATE TABLE sys_dept (
		id TEXT PRIMARY KEY, dept_name TEXT, dept_code TEXT, parent_id TEXT, ancestors TEXT DEFAULT '',
		org_level INTEGER DEFAULT 0, is_external_org INTEGER DEFAULT 0, status INTEGER DEFAULT 0,
		order_num INTEGER DEFAULT 0, leader TEXT, phone TEXT, email TEXT, remark TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, created_by TEXT, updated_by TEXT, version INTEGER DEFAULT 0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id,dept_name,dept_code,status) VALUES
		('D-hb','湖北分公司','HB',0), ('D-cs','长沙中支','CS',0)`).Error)

	svc := NewDepartmentServiceWithCache(db, &NoOpCacheProvider{}, nil)
	// 不传 scoped，即使 ctx 有窄范围也应返回全量（tree-select 配置场景语义）
	ctx := datascope.WithContext(context.Background(), datascope.ScopeResult{
		DeptIDs: []string{"D-hb"}, UserID: "U-x",
	})
	tree, err := svc.GetTreeWithFilter(ctx, true, requests.DepartmentListParams{})
	require.NoError(t, err)
	ids := map[string]bool{}
	var walk func([]*models.Department)
	walk = func(nodes []*models.Department) {
		for _, n := range nodes {
			ids[n.ID] = true
			walk(n.Children)
		}
	}
	walk(tree)
	assert.True(t, ids["D-hb"])
	assert.True(t, ids["D-cs"], "不传 scoped 保持全量（配置场景零回归）")
}
