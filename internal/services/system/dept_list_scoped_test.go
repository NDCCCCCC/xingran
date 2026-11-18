package system

// dept_list_scoped_test.go —— 扁平部门列表 scoped 过滤的回归（网络发现页 dept 下拉）。
//
// /system/departments/list 曾被网络发现页以 pageSize:50 拉全量部门。v1.54 给该
// 端点加 Scoped opt-in（departmentService.List → filterDeptListByScope），前端
// useDiscoveryData 传 scoped:true。本测试锁死：scoped 请求只返回可见集内部门。

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
)

func newDeptListScopedDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, db.Exec(`CREATE TABLE sys_dept (
		id TEXT PRIMARY KEY, dept_name TEXT, dept_code TEXT, parent_id TEXT, ancestors TEXT DEFAULT '',
		org_level INTEGER DEFAULT 0, is_external_org INTEGER DEFAULT 0, status INTEGER DEFAULT 0,
		order_num INTEGER DEFAULT 0, leader TEXT, phone TEXT, email TEXT, remark TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, created_by TEXT, updated_by TEXT, version INTEGER DEFAULT 0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id,dept_name,dept_code,status) VALUES
		('D-wh','武汉中支','WH',0), ('D-zh','综合部','ZH',0), ('D-qk','硚口支公司','QK',0), ('D-cs','长沙中支','CS',0)`).Error)
	return db
}

func TestDeptList_Scoped_FiltersByVisibleSet(t *testing.T) {
	db := newDeptListScopedDB(t)
	svc := NewDepartmentService(db)

	ctx := datascope.WithContext(context.Background(), datascope.ScopeResult{
		DeptIDs: []string{"D-wh", "D-zh", "D-qk"}, UserID: "U-liu",
	})
	list, err := svc.List(ctx, requests.DepartmentListParams{Scoped: true})
	require.NoError(t, err)
	ids := make(map[string]bool)
	for _, d := range list {
		ids[d.ID] = true
	}
	assert.True(t, ids["D-wh"])
	assert.True(t, ids["D-qk"])
	assert.False(t, ids["D-cs"], "可见集外的长沙中支必须被过滤")
}

func TestDeptList_NoScoped_StillFull(t *testing.T) {
	db := newDeptListScopedDB(t)
	svc := NewDepartmentService(db)

	ctx := datascope.WithContext(context.Background(), datascope.ScopeResult{
		DeptIDs: []string{"D-wh"}, UserID: "U-x",
	})
	list, err := svc.List(ctx, requests.DepartmentListParams{})
	require.NoError(t, err)
	assert.Len(t, list, 4, "不传 scoped 保持全量（配置/既有场景零回归）")
}

func TestDeptList_Scoped_AllAndSelf(t *testing.T) {
	db := newDeptListScopedDB(t)
	svc := NewDepartmentService(db)

	// All 档 → 全量
	allCtx := datascope.WithContext(context.Background(), datascope.AllScope("U-admin"))
	allList, err := svc.List(allCtx, requests.DepartmentListParams{Scoped: true})
	require.NoError(t, err)
	assert.Len(t, allList, 4)

	// Self 档 → 空列表
	selfCtx := datascope.WithContext(context.Background(), datascope.ScopeResult{
		ResourceHook: datascope.HookSelfUser, UserID: "U-x",
	})
	selfList, err := svc.List(selfCtx, requests.DepartmentListParams{Scoped: true})
	require.NoError(t, err)
	assert.Empty(t, selfList)
}
