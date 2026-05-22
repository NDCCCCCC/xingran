package addomain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCountTotalDepts(t *testing.T) {
	service := &DeptToADSyncService{}

	// 测试空部门列表
	depts := []*models.Department{}
	count := service.countTotalDepts(depts)
	assert.Equal(t, 0, count)

	// 测试单层部门
	depts = []*models.Department{
		{DeptName: "Dept1"},
		{DeptName: "Dept2"},
	}
	count = service.countTotalDepts(depts)
	assert.Equal(t, 2, count)

	// 测试多层部门
	parent := &models.Department{DeptName: "Parent"}
	child1 := &models.Department{DeptName: "Child1"}
	child2 := &models.Department{DeptName: "Child2"}
	grandchild := &models.Department{DeptName: "Grandchild"}

	parent.Children = []*models.Department{child1, child2}
	child1.Children = []*models.Department{grandchild}

	depts = []*models.Department{parent}
	count = service.countTotalDepts(depts)
	assert.Equal(t, 4, count) // Parent + Child1 + Child2 + Grandchild
}

func TestGetRootDepartments(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 创建部门表
	db.Exec(`CREATE TABLE sys_dept (
		id TEXT PRIMARY KEY,
		dept_name TEXT NOT NULL,
		parent_id TEXT,
		status INTEGER DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	db.Exec(`CREATE INDEX idx_sys_dept_parent_id ON sys_dept(parent_id)`)

	service := &DeptToADSyncService{db: db}

	ctx := context.Background()

	// 测试无根部门
	depts, err := service.getRootDepartments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(depts))

	// 准备测试数据
	rootID := "root-1"
	childID := "child-1"
	db.Exec(`INSERT INTO sys_dept (id, dept_name, parent_id, status) VALUES (?, ?, NULL, 0)`, rootID, "RootDept")
	db.Exec(`INSERT INTO sys_dept (id, dept_name, parent_id, status) VALUES (?, ?, ?, 0)`, childID, "ChildDept", rootID)

	// 测试查询根部门
	depts, err = service.getRootDepartments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(depts))
	assert.Equal(t, "RootDept", depts[0].DeptName)
}

func TestGetRootDepartments_WithStatus(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 创建部门表
	db.Exec(`CREATE TABLE sys_dept (
		id TEXT PRIMARY KEY,
		dept_name TEXT NOT NULL,
		parent_id TEXT,
		status INTEGER DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	service := &DeptToADSyncService{db: db}

	ctx := context.Background()

	// 准备测试数据：包含正常和停用部门
	db.Exec(`INSERT INTO sys_dept (id, dept_name, parent_id, status) VALUES ('root-1', 'ActiveRoot', NULL, 0)`)
	db.Exec(`INSERT INTO sys_dept (id, dept_name, parent_id, status) VALUES ('root-2', 'InactiveRoot', NULL, 1)`)
	db.Exec(`INSERT INTO sys_dept (id, dept_name, parent_id, status) VALUES ('child-1', 'ActiveChild', 'root-1', 0)`)

	// 测试查询根部门（应只返回正常状态的根部门）
	depts, err := service.getRootDepartments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(depts)) // 只有 ActiveRoot，InactiveRoot 被过滤
	assert.Equal(t, "ActiveRoot", depts[0].DeptName)
}

func TestSyncDeptStructureToAD_EmptyTree(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 创建必要的表
	db.Exec(`CREATE TABLE sys_dept (
		id TEXT PRIMARY KEY,
		dept_name TEXT NOT NULL,
		parent_id TEXT,
		status INTEGER DEFAULT 0
	)`)
	db.Exec(`CREATE TABLE sys_dept_ou_mapping (
		id TEXT PRIMARY KEY,
		dept_id TEXT NOT NULL,
		ad_config_id TEXT NOT NULL,
		ou_dn TEXT NOT NULL,
		ou_name TEXT NOT NULL
	)`)
	db.Exec(`CREATE TABLE sys_ad_sync_log (
		id TEXT PRIMARY KEY,
		ad_config_id TEXT NOT NULL,
		sync_type TEXT NOT NULL,
		start_time DATETIME,
		end_time DATETIME,
		total_count INTEGER DEFAULT 0,
		success_count INTEGER DEFAULT 0,
		failed_count INTEGER DEFAULT 0,
		ou_count INTEGER DEFAULT 0
	)`)

	// Mock LDAP client
	mockLDAP := &LDAPClient{}

	service := &DeptToADSyncService{
		db:        db,
		ldap:      mockLDAP,
		mapper:    NewDeptOUmapper(db),
		config: &ADConfig{
			ID:     "config-1",
			Config: &ADConfigDetail{},
		},
	}

	ctx := context.Background()

	// 测试空树
	result, err := service.SyncDeptStructureToAD(ctx, "config-1")
	assert.NoError(t, err)
	assert.Equal(t, 0, result.TotalCount)
	assert.Equal(t, 0, result.OUCount)
}

func TestBuildSyncResult(t *testing.T) {
	service := &DeptToADSyncService{}

	// 测试构建同步结果
	result := service.buildSyncResult("config-1", 100, 95, 5, 20)
	assert.Equal(t, int64(100), result.TotalCount)
	assert.Equal(t, int64(95), result.SuccessCount)
	assert.Equal(t, int64(5), result.FailedCount)
	assert.Equal(t, int64(20), result.OUCount)
	assert.Equal(t, "config-1", result.ConfigID)
}
