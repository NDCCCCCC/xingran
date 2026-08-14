package db

import (
	"os"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/core/security"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// freshUserDB 创建内存 SQLite + 仅 sys_user 表,供 createDefaultUser 测试用。
//
// 注意:不用 file::memory:?cache=shared —— shared cache 在多个测试间共享,
// 会导致先运行的测试插入的 admin User 行污染后续测试(每次 freshUserDB
// 返回的 *gorm.DB 看到 count > 0 → createDefaultUser 走 idempotent 跳过
// 路径,返回的 User 行的密码哈希来自第一次创建时的密码)。
// 这里用每个测试独立的 in-memory DSN (file:TestName?mode=memory&cache=private)。
func freshUserDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file::memory:?cache=private&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

// TestCreateDefaultUser_EnvOverride SYS_ADMIN_BOOTSTRAP_PASSWORD 环境变量被识别为初始密码。
// C2 修复:env 覆盖路径必须正确生效,VerifyPassword 通过。
func TestCreateDefaultUser_EnvOverride(t *testing.T) {
	t.Setenv("SYS_ADMIN_BOOTSTRAP_PASSWORD", "Test@2026")
	db := freshUserDB(t)

	if err := createDefaultUser(db); err != nil {
		t.Fatalf("createDefaultUser: %v", err)
	}

	var user models.User
	if err := db.Where("username = ?", "admin").First(&user).Error; err != nil {
		t.Fatalf("admin user not found: %v", err)
	}

	pm := security.NewPasswordManager(nil)
	ok, err := pm.VerifyPassword("Test@2026", user.Password)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatalf("VerifyPassword(env password) failed: env override path broken")
	}
}

// TestCreateDefaultUser_FallbackDefault 未设置环境变量时回退 admin123(向后兼容)。
// C2 修复:回退路径必须保证 admin123 仍可登录,以避免破坏现有部署。
func TestCreateDefaultUser_FallbackDefault(t *testing.T) {
	// 通过 t.Setenv("KEY", "") 把 env var 设为空串 — 这是
	// createDefaultUser 中 "if bootstrapPassword == ''" 判定所看到的语义。
	// 与"未设置"在 os.Getenv 层面行为等价(os.Getenv 返回 ""),
	// 但显式可控、可还原。
	t.Setenv("SYS_ADMIN_BOOTSTRAP_PASSWORD", "")

	db := freshUserDB(t)
	if err := createDefaultUser(db); err != nil {
		t.Fatalf("createDefaultUser: %v", err)
	}

	var user models.User
	if err := db.Where("username = ?", "admin").First(&user).Error; err != nil {
		t.Fatalf("admin user not found: %v", err)
	}

	pm := security.NewPasswordManager(nil)
	ok, err := pm.VerifyPassword("admin123", user.Password)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatalf("VerifyPassword(admin123) failed: fallback path broken")
	}
}

// TestCreateDefaultUser_NoDefaultLiteralSalt 两种路径下 Salt 字段均不为字面量 "default"。
// C2 修复:Salt 是死字段(PasswordManager 盐已嵌入 $sm3$iterations$salt$hash),
// 不再携带误导字面量。
func TestCreateDefaultUser_NoDefaultLiteralSalt(t *testing.T) {
	cases := []struct {
		name string
		env  string
	}{
		{"env path", "Test@2026"},
		{"fallback path", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SYS_ADMIN_BOOTSTRAP_PASSWORD", tc.env)

			db := freshUserDB(t)
			if err := createDefaultUser(db); err != nil {
				t.Fatalf("createDefaultUser: %v", err)
			}

			var user models.User
			if err := db.Where("username = ?", "admin").First(&user).Error; err != nil {
				t.Fatalf("admin user not found: %v", err)
			}

			if user.Salt == "default" {
				t.Fatalf("Salt field must NOT be literal \"default\" (dead field — actual salt embedded in PasswordManager hash); got %q", user.Salt)
			}
		})
	}
}

// TestCreateDefaultUser_Idempotent admin 已存在时二次调用返回 nil 且不新增行。
// C2 修复:幂等性回归测试,防止后续改动破坏既有"exists 时跳过"行为。
func TestCreateDefaultUser_Idempotent(t *testing.T) {
	t.Setenv("SYS_ADMIN_BOOTSTRAP_PASSWORD", "FirstPass!")
	db := freshUserDB(t)

	if err := createDefaultUser(db); err != nil {
		t.Fatalf("first call: %v", err)
	}

	var countBefore int64
	db.Model(&models.User{}).Where("username = ?", "admin").Count(&countBefore)
	if countBefore != 1 {
		t.Fatalf("expected 1 admin row after first call, got %d", countBefore)
	}

	// 二次调用必须 nil 且行数不变
	if err := createDefaultUser(db); err != nil {
		t.Fatalf("second call (idempotent): %v", err)
	}
	var countAfter int64
	db.Model(&models.User{}).Where("username = ?", "admin").Count(&countAfter)
	if countAfter != 1 {
		t.Fatalf("expected 1 admin row after second call (idempotent), got %d", countAfter)
	}
}

// freshDeptDB 创建内存 SQLite + 仅 sys_dept 表,供 createDefaultDept 测试用。
//
// 同 freshUserDB:用 cache=private 避免跨测试共享内存数据库。
func freshDeptDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file::memory:?cache=private&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Department{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

// TestCreateDefaultDept_FullSeed 空库调用 createDefaultDept 期望所有 6 行部门创建成功。
// C5 修复:验证细粒度幂等不影响首装种子流程。
func TestCreateDefaultDept_FullSeed(t *testing.T) {
	db := freshDeptDB(t)

	if err := createDefaultDept(db); err != nil {
		t.Fatalf("createDefaultDept: %v", err)
	}

	var total int64
	db.Model(&models.Department{}).Count(&total)
	if total != 6 {
		t.Fatalf("expected 6 dept rows (1 top + 2 children + 3 shenzhen sub), got %d", total)
	}
}

// TestCreateDefaultDept_PartialRecovers 部分状态恢复:
// 预先只插入顶级部门,createDefaultDept 必须补齐剩余 5 行而不重复创建顶级。
// C5 修复:细粒度 check-and-create 的核心证明 — 首次启动中途失败可在下次启动自愈。
func TestCreateDefaultDept_PartialRecovers(t *testing.T) {
	db := freshDeptDB(t)

	// 预先插入顶级部门,模拟"首次启动中途失败只完成了顶级"
	topDept := models.Department{
		DeptName: "若依科技有限公司",
		OrderNum: 1,
		Status:   models.DeptStatusNormal,
	}
	if err := db.Create(&topDept).Error; err != nil {
		t.Fatalf("seed top dept: %v", err)
	}

	if err := createDefaultDept(db); err != nil {
		t.Fatalf("createDefaultDept: %v", err)
	}

	// 期望:补齐 5 个(深圳总公司/长沙分公司/研发部门/市场部门/测试部门)
	// 顶级部门不重复创建,总数 6,顶级仅 1 行
	var total int64
	db.Model(&models.Department{}).Count(&total)
	if total != 6 {
		t.Fatalf("expected 6 dept rows after partial recovery, got %d", total)
	}

	var topCount int64
	db.Model(&models.Department{}).Where("dept_name = ?", "若依科技有限公司").Count(&topCount)
	if topCount != 1 {
		t.Fatalf("expected exactly 1 top dept row, got %d (duplicate created — C5 regression)", topCount)
	}

	// 深圳总公司必须被补齐
	var shenzhenCount int64
	db.Model(&models.Department{}).Where("dept_name = ?", "深圳总公司").Count(&shenzhenCount)
	if shenzhenCount != 1 {
		t.Fatalf("expected 1 shenzhen parent row after recovery, got %d", shenzhenCount)
	}
}

// TestCreateDefaultDept_FullyIdempotent 完整 seed 后再次调用 createDefaultDept 必须 nil 且总数不变。
// C5 修复:验证细粒度幂等同样支持"已存在"的正常分支。
func TestCreateDefaultDept_FullyIdempotent(t *testing.T) {
	db := freshDeptDB(t)

	if err := createDefaultDept(db); err != nil {
		t.Fatalf("first call: %v", err)
	}
	var countBefore int64
	db.Model(&models.Department{}).Count(&countBefore)

	if err := createDefaultDept(db); err != nil {
		t.Fatalf("second call (idempotent): %v", err)
	}
	var countAfter int64
	db.Model(&models.Department{}).Count(&countAfter)

	if countBefore != countAfter {
		t.Fatalf("expected dept count unchanged (%d → %d), got delta=%d", countBefore, countAfter, countAfter-countBefore)
	}
	if countAfter != 6 {
		t.Fatalf("expected 6 dept rows after full seed, got %d", countAfter)
	}
}

// freshUserRoleDB 创建内存 SQLite + sys_user / sys_role / sys_user_role,供 createUserRoleRelations 测试用。
//
// 同 freshUserDB:用 cache=private 避免跨测试共享内存数据库。
func freshUserRoleDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file::memory:?cache=private&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Role{}, &models.UserRole{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

// TestCreateUserRoleRelations_CreatesAndIdempotent 首次创建 sys_user_role 行;
// 二次调用 nil 且行数不变(幂等)。CDX-M-USERROLE 修复:db.Create(&models.UserRole{...})
// 取代硬编码表名原生 INSERT,本测试同时验证修复后行为等价。
func TestCreateUserRoleRelations_CreatesAndIdempotent(t *testing.T) {
	db := freshUserRoleDB(t)

	// 直接 seed 一个 admin 用户和 admin 角色(createDefaultUser/createDefaultRole 不在本测试范围)
	pwdManager := security.NewPasswordManager(nil)
	pwHash, err := pwdManager.HashPassword("admin123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	adminUser := models.User{
		Username: "admin",
		Password: pwHash,
		Status:   models.UserStatusEnabled,
	}
	if err := db.Create(&adminUser).Error; err != nil {
		t.Fatalf("seed admin user: %v", err)
	}
	adminRole := models.Role{
		RoleName: "超级管理员",
		RoleKey:  "admin",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	}
	if err := db.Create(&adminRole).Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}

	if err := createUserRoleRelations(db); err != nil {
		t.Fatalf("createUserRoleRelations: %v", err)
	}

	var count1 int64
	db.Table("sys_user_role").Count(&count1)
	if count1 != 1 {
		t.Fatalf("expected 1 sys_user_role row after first call, got %d", count1)
	}

	// 二次调用必须幂等
	if err := createUserRoleRelations(db); err != nil {
		t.Fatalf("second call (idempotent): %v", err)
	}
	var count2 int64
	db.Table("sys_user_role").Count(&count2)
	if count2 != 1 {
		t.Fatalf("expected 1 sys_user_role row after second call, got %d (idempotent violation)", count2)
	}
}

// TestSourceAssertions_MenuSeedErrorPaths 源码断言:菜单种子两个循环必须使用
// errors.Is(err, gorm.ErrRecordNotFound) 显式区分错误类型;按钮循环含父菜单 ID 守卫;
// createUserRoleRelations 不得含硬编码 INSERT INTO sys_user_role。
// OC-M-MENUSEED + CDX-M-USERROLE 修复的源码契约回归守护。
func TestSourceAssertions_MenuSeedErrorPaths(t *testing.T) {
	src, err := os.ReadFile("init_data.go")
	if err != nil {
		t.Fatalf("read init_data.go: %v", err)
	}
	s := string(src)

	// OC-M-MENUSEED: 页面 + 按钮两个循环必须各用一次 errors.Is(err, gorm.ErrRecordNotFound)
	if got := countOccurrences(s, "errors.Is(err, gorm.ErrRecordNotFound)"); got < 2 {
		t.Fatalf("init_data.go must use errors.Is(err, gorm.ErrRecordNotFound) in BOTH page and button loops (>=2 occurrences), got %d", got)
	}

	// OC-M-MENUSEED: 按钮循环必须含父菜单 ID 空值守卫
	if !contains(s, `parentMenuID == ""`) {
		t.Fatalf("button loop must guard against empty parent menu ID: missing `parentMenuID == \"\"` check")
	}

	// CDX-M-USERROLE: createUserRoleRelations 必须用 db.Create(&models.UserRole{...})
	if !contains(s, "models.UserRole{") {
		t.Fatalf("createUserRoleRelations must use db.Create(&models.UserRole{...}) instead of hardcoded SQL")
	}

	// CDX-M-USERROLE: 反向断言 — 不得含硬编码表名 INSERT
	if contains(s, "INSERT INTO sys_user_role") {
		t.Fatalf("init_data.go must NOT contain hardcoded \"INSERT INTO sys_user_role\" (CDX-M-USERROLE)")
	}
}

// countOccurrences 统计 sub 在 s 中的非重叠出现次数。
func countOccurrences(s, sub string) int {
	if sub == "" {
		return 0
	}
	n := 0
	for i := 0; i+len(sub) <= len(s); {
		if s[i:i+len(sub)] == sub {
			n++
			i += len(sub)
			continue
		}
		i++
	}
	return n
}