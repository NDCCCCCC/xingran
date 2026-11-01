package addomain

// DEBT-01（v1.51）第五批：部门-AD OU 级联服务（DeptADCascadeService）行为测试。
//
// 覆盖目标：
//  1. pool=nil 的 no-op 守卫（单元测试/未启用 AD 场景契约）
//  2. deptOUMappings：无映射短路 / 只联启用的 AD 配置
//  3. configByID / derefParent 纯函数
//  4. replaceOUDNStrings：本部门 + 子孙映射 DN 替换（ou_dn/ou_name/parent_ou_dn），
//     其它配置的映射行不受影响
//  5. cleanupDeptOURows：部门 OU 映射与 OU 组映射引用清理
//
// sys_dept_ou_mapping / sys_ou_group_mapping 带 gen_random_uuid 默认值，
// SQLite 无法 AutoMigrate，复用 test_fixtures_test.go 的手写建表模式。

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

func setupCascadeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(createDeptOUMappingTable).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_ou_group_mapping (
		id TEXT PRIMARY KEY,
		ad_config_id TEXT NOT NULL,
		ou_dn TEXT NOT NULL UNIQUE,
		ou_name TEXT NOT NULL,
		ad_group_id TEXT NOT NULL,
		mapping_status TEXT NOT NULL DEFAULT 'active',
		sync_enabled INTEGER DEFAULT 1,
		created_by TEXT DEFAULT '', updated_by TEXT DEFAULT '',
		deleted_at DATETIME,
		created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.AutoMigrate(&models.ADConfig{}))
	return db
}

func TestDeptADCascade_NilPoolNoOp(t *testing.T) {
	db := setupCascadeTestDB(t)
	ctx := context.Background()
	svc := NewDeptADCascadeService(db, nil)

	// pool=nil：单元测试/未启用 AD 场景 → 所有方法 no-op
	require.NoError(t, svc.RenameDeptOU(ctx, "d-1", "新名字"))
	require.NoError(t, svc.DeleteDeptOU(ctx, "d-1"))

	// nil 接收者双重空检
	var nilSvc *DeptADCascadeService
	require.NoError(t, nilSvc.RenameDeptOU(ctx, "d-1", "新名字"))
	require.NoError(t, nilSvc.DeleteDeptOU(ctx, "d-1"))
}

func TestDeptOUMappings_OnlyEnabledConfigs(t *testing.T) {
	db := setupCascadeTestDB(t)
	ctx := context.Background()
	svc := NewDeptADCascadeService(db, nil)

	// 无映射 → (nil, nil, nil)
	m, c, err := svc.deptOUMappings(ctx, "d-none")
	require.NoError(t, err)
	if m != nil || c != nil {
		t.Fatalf("无映射应返回 nil, got %v %v", m, c)
	}

	cfgs := []models.ADConfig{
		{BaseModel: models.BaseModel{ID: "cfg-on"}, ConfigName: "启用", ServerAddress: "1.1.1.1", ServerPort: 389, DomainName: "d", BaseDN: "dc=t", Status: models.ADConfigStatusEnabled},
		{BaseModel: models.BaseModel{ID: "cfg-off"}, ConfigName: "停用", ServerAddress: "1.1.1.2", ServerPort: 389, DomainName: "d", BaseDN: "dc=t", Status: 1},
	}
	require.NoError(t, db.Create(&cfgs).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept_ou_mapping (id, dept_id, ad_config_id, ou_dn, ou_name) VALUES
		('m-1','d-1','cfg-on','OU=财务,DC=t','财务'),
		('m-2','d-1','cfg-off','OU=停用,DC=t','停用')`).Error)

	m, c, err = svc.deptOUMappings(ctx, "d-1")
	require.NoError(t, err)
	if len(m) != 2 {
		t.Fatalf("映射应含全部 2 条, got %d", len(m))
	}
	if len(c) != 1 || c[0].ID != "cfg-on" {
		t.Fatalf("配置应只含启用配置, got %+v", c)
	}

	// configByID：命中 / 未命中
	if configByID(c, "cfg-on") == nil {
		t.Fatal("应命中 cfg-on")
	}
	if configByID(c, "cfg-off") != nil {
		t.Fatal("停用配置不应命中（不在启用列表）")
	}
	// derefParent
	if derefParent(nil) != "" || derefParent(strPtr("OU=p,DC=t")) != "OU=p,DC=t" {
		t.Fatal("derefParent 语义错误")
	}
}

func TestReplaceOUDNStrings(t *testing.T) {
	db := setupCascadeTestDB(t)
	ctx := context.Background()
	svc := NewDeptADCascadeService(db, nil)

	// 本部门行（精确命中）+ 子孙行（后缀命中）+ 其它配置行（不应受影响）
	require.NoError(t, db.Exec(`INSERT INTO sys_dept_ou_mapping (id, dept_id, ad_config_id, ou_dn, ou_name, parent_ou_dn) VALUES
		('m-self','d-1','cfg-1','OU=财务,DC=t','财务',NULL),
		('m-child','d-2','cfg-1','OU=薪酬,OU=财务,DC=t','薪酬','OU=财务,DC=t'),
		('m-other','d-3','cfg-2','OU=财务,DC=t','财务',NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_ou_group_mapping (id, ad_config_id, ou_dn, ou_name, ad_group_id) VALUES
		('g-1','cfg-1','OU=财务,DC=t','财务','grp-1'),
		('g-2','cfg-2','OU=hr,DC=t','人力','grp-2')`).Error)

	svc.replaceOUDNStrings(ctx, "cfg-1", "OU=财务,DC=t", "OU=财务中心,DC=t", "财务中心")

	var self, child, other struct {
		OUDN       string  `gorm:"column:ou_dn"`
		OUName     string  `gorm:"column:ou_name"`
		ParentOUDN *string `gorm:"column:parent_ou_dn"`
	}
	require.NoError(t, db.Table("sys_dept_ou_mapping").Select("ou_dn, ou_name, parent_ou_dn").Where("id = ?", "m-self").Scan(&self).Error)
	if self.OUDN != "OU=财务中心,DC=t" || self.OUName != "财务中心" {
		t.Fatalf("本部门行应替换 DN 与名称, got %+v", self)
	}
	require.NoError(t, db.Table("sys_dept_ou_mapping").Select("ou_dn, ou_name, parent_ou_dn").Where("id = ?", "m-child").Scan(&child).Error)
	if child.OUDN != "OU=薪酬,OU=财务中心,DC=t" {
		t.Fatalf("子孙行 ou_dn 应随动替换, got %q", child.OUDN)
	}
	if child.ParentOUDN == nil || *child.ParentOUDN != "OU=财务中心,DC=t" {
		t.Fatalf("子孙行 parent_ou_dn 应随动替换, got %v", child.ParentOUDN)
	}
	require.NoError(t, db.Table("sys_dept_ou_mapping").Select("ou_dn").Where("id = ?", "m-other").Scan(&other).Error)
	if other.OUDN != "OU=财务,DC=t" {
		t.Fatalf("其它配置的行不应受影响, got %q", other.OUDN)
	}

	var g1, g2 struct {
		OUDN   string `gorm:"column:ou_dn"`
		OUName string `gorm:"column:ou_name"`
	}
	require.NoError(t, db.Table("sys_ou_group_mapping").Select("ou_dn, ou_name").Where("id = ?", "g-1").Scan(&g1).Error)
	if g1.OUDN != "OU=财务中心,DC=t" || g1.OUName != "财务中心" {
		t.Fatalf("OU 组映射应同步替换, got %+v", g1)
	}
	require.NoError(t, db.Table("sys_ou_group_mapping").Select("ou_dn").Where("id = ?", "g-2").Scan(&g2).Error)
	if g2.OUDN != "OU=hr,DC=t" {
		t.Fatalf("其它配置的组映射不应受影响, got %q", g2.OUDN)
	}
}

func TestCleanupDeptOURows(t *testing.T) {
	db := setupCascadeTestDB(t)
	ctx := context.Background()
	svc := NewDeptADCascadeService(db, nil)

	require.NoError(t, db.Exec(`INSERT INTO sys_dept_ou_mapping (id, dept_id, ad_config_id, ou_dn, ou_name) VALUES
		('m-1','d-1','cfg-1','OU=财务,DC=t','财务')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_ou_group_mapping (id, ad_config_id, ou_dn, ou_name, ad_group_id) VALUES
		('g-1','cfg-1','OU=财务,DC=t','财务','grp-1')`).Error)

	svc.cleanupDeptOURows(ctx, models.DeptOUMapping{ID: "m-1", DeptID: "d-1", ADConfigID: "cfg-1", OUDN: "OU=财务,DC=t"})

	var cnt int64
	require.NoError(t, db.Table("sys_dept_ou_mapping").Where("id = ?", "m-1").Count(&cnt).Error)
	if cnt != 0 {
		t.Fatal("部门 OU 映射应被删除")
	}
	require.NoError(t, db.Table("sys_ou_group_mapping").Where("ou_dn = ?", "OU=财务,DC=t").Count(&cnt).Error)
	if cnt != 0 {
		t.Fatal("OU 组映射引用应被删除")
	}
	// 幂等：重复清理不报错
	svc.cleanupDeptOURows(ctx, models.DeptOUMapping{ID: "m-1", ADConfigID: "cfg-1", OUDN: "OU=财务,DC=t"})
}
