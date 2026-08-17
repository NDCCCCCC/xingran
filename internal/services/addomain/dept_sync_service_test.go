package addomain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/glebarez/sqlite"
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

	// 创建部门表（包含deleted_at用于GORM软删除）
	db.Exec(`CREATE TABLE sys_dept (
		id TEXT PRIMARY KEY,
		dept_name TEXT NOT NULL,
		dept_code TEXT NOT NULL DEFAULT '',
		parent_id TEXT,
		status INTEGER DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
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

	// 创建部门表（包含deleted_at用于GORM软删除）
	db.Exec(`CREATE TABLE sys_dept (
		id TEXT PRIMARY KEY,
		dept_name TEXT NOT NULL,
		dept_code TEXT NOT NULL DEFAULT '',
		parent_id TEXT,
		status INTEGER DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
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

func TestGetRootDepartments_SkipsTopLevel(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 创建部门表（包含deleted_at用于GORM软删除）
	db.Exec(`CREATE TABLE sys_dept (
		id TEXT PRIMARY KEY,
		dept_name TEXT NOT NULL,
		dept_code TEXT NOT NULL DEFAULT '',
		parent_id TEXT,
		status INTEGER DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	db.Exec(`CREATE INDEX idx_sys_dept_parent_id ON sys_dept(parent_id)`)

	service := &DeptToADSyncService{db: db}

	ctx := context.Background()

	// 准备测试数据：模拟实际场景
	// 顶级部门：中国太平洋财产保险股份有限公司湖北分公司
	// 二级部门：分公司本部、武汉中心支公司
	rootID := "root-1"
	secondLevelID1 := "second-1"
	secondLevelID2 := "second-2"
	thirdLevelID := "third-1"

	db.Exec(`INSERT INTO sys_dept (id, dept_name, parent_id, status) VALUES (?, ?, NULL, 0)`, rootID, "中国太平洋财产保险股份有限公司湖北分公司")
	db.Exec(`INSERT INTO sys_dept (id, dept_name, parent_id, status) VALUES (?, ?, ?, 0)`, secondLevelID1, "分公司本部", rootID)
	db.Exec(`INSERT INTO sys_dept (id, dept_name, parent_id, status) VALUES (?, ?, ?, 0)`, secondLevelID2, "武汉中心支公司", rootID)
	db.Exec(`INSERT INTO sys_dept (id, dept_name, parent_id, status) VALUES (?, ?, ?, 0)`, thirdLevelID, "综合管理部", secondLevelID1)

	// 获取根部门
	rootDepts, err := service.getRootDepartments(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(rootDepts))
	assert.Equal(t, "中国太平洋财产保险股份有限公司湖北分公司", rootDepts[0].DeptName)

	// 验证跳过顶层部门的逻辑：收集二级部门
	var secondLevelDepts []*models.Department
	for _, root := range rootDepts {
		for _, child := range root.Children {
			secondLevelDepts = append(secondLevelDepts, child)
		}
	}
	assert.Equal(t, 2, len(secondLevelDepts))

	// 验证二级部门名称
	secondLevelNames := make(map[string]bool)
	for _, dept := range secondLevelDepts {
		secondLevelNames[dept.DeptName] = true
	}
	assert.True(t, secondLevelNames["分公司本部"])
	assert.True(t, secondLevelNames["武汉中心支公司"])

	// 验证三级部门被预加载
	for _, dept := range secondLevelDepts {
		if dept.DeptName == "分公司本部" {
			assert.Equal(t, 1, len(dept.Children))
			assert.Equal(t, "综合管理部", dept.Children[0].DeptName)
		}
	}
}
