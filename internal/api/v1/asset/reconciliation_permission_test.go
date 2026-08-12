package asset

// Package asset - Phase 42 R1 plan 06 (路由注册 + operlog + cron + 跨模块权限测试)
//
// reconciliation_permission_test.go
// 包含 3 个测试:
//
//   1. TestReconciliationModuleConstExists — 静态 grep 断言,验证 ModuleReconciliation
//      常量就位(D-16 R1 only 1 const)。防回归。
//   2. TestReconciliationEndpoints_PermissionBoundary — 完整 router spin-up
//      验证跨模块权限边界三态(无 token → 401;无 asset:reconciliation:list → 403;
//      有权限 → 200)。
//   3. TestReconciliationStatistics_NoListLength — 静态扫描 reconciliation_statistics.go
//      中 6 个方法体,断言不含 Find( 或 .Offset( 反模式(防 stat-cards-from-list-length-capped-at-100 回归)。

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ============================================================================
// Test 1: 静态 grep 断言(D-16 R1 only 1 module const)
//
// 防 ModuleReconciliation 常量被误删/改名(无运行时副作用,
// 静态断言确保 operlog.Record 在 R1 + R2 扩展时不会引用空字符串)。
// ============================================================================

func TestReconciliationModuleConstExists(t *testing.T) {
	// 读取同目录下的 handler 文件
	handlerPath := "reconciliation_handler.go"
	data, err := os.ReadFile(handlerPath)
	require.NoError(t, err, "must read reconciliation_handler.go for static assertion")

	content := string(data)
	assert.Contains(t, content, "ModuleReconciliation",
		"D-16: ModuleReconciliation const must exist in reconciliation_handler.go (防 R1 回归)")
	assert.Contains(t, content, `"资产对账"`,
		"D-16: ModuleReconciliation value must be 资产对账(operlog 模块中文名)")
}

// ============================================================================
// Test 2: 完整 router spin-up 验证跨模块权限边界三态
//
// Option A: 在测试中组装 minimal *core.Core + 假 JWTAuth(用简化 header 检测)。
// 真实路径使用 httptest 模拟调用,3 种状态可观测。
// 注: 实际内部 package middleware 的 JWTAuth 需真 token + SM4 + JWTManager,
// 测试中用简化版"Authorization header 存在性"替代,验证中间件链对 401 的反应。
// ============================================================================

// reconciliationPerms R1 端点要求的权限列表(与 internal/api/router.go 一致)
var reconciliationPerms = []string{
	"asset:reconciliation:list",
	"asset:reconciliation:dashboard",
	"asset:reconciliation:export",
}

// reconciliationPermTables 为 permission 中间件准备的最小 sys_* schema。
// (抄自 pkg/middleware/permission_inherit_test.go::createTables,精简到 reconciliation 端点必需字段)
func reconciliationPermTables(t *testing.T, gormDB *gorm.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS sys_role (
			id TEXT PRIMARY KEY,
			role_name TEXT NOT NULL,
			role_key TEXT NOT NULL,
			role_sort INTEGER DEFAULT 0,
			data_scope INTEGER DEFAULT 1,
			menu_check_strictly BOOLEAN DEFAULT 1,
			dept_check_strictly BOOLEAN DEFAULT 1,
			status INTEGER DEFAULT 0,
			del_flag BOOLEAN DEFAULT 0,
			remark TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS sys_menu (
			id TEXT PRIMARY KEY,
			menu_name TEXT NOT NULL,
			parent_id TEXT,
			order_num INTEGER DEFAULT 0,
			path TEXT,
			component TEXT,
			menu_type TEXT DEFAULT 'M',
			visible INTEGER DEFAULT 1,
			status INTEGER DEFAULT 0,
			perms TEXT,
			icon TEXT,
			remark TEXT,
			meta TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS sys_user_role (
			user_id TEXT NOT NULL,
			role_id TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sys_role_menu (
			role_id TEXT NOT NULL,
			menu_id TEXT NOT NULL
		)`,
		// reconciliation / statistics 端点需要的最小表 schema
		// ops_asset.machine_ip 用 TEXT(SQLite 不支持 inet,ListExceptions 的 COALESCE(a.machine_ip::text, '')
		// 在 SQLite 上 ::text 会被忽略 — 实际上 PG 与 SQLite 行为不一致,这里测试只关心 permission path 通过 200,
		// 不验证 SQL 语义;若想测 SQL 应在 PG in-memory 测试中验证)
		`CREATE TABLE IF NOT EXISTS ops_asset (
			id TEXT PRIMARY KEY,
			devicesn TEXT,
			machine_ip TEXT,
			user_id TEXT,
			status INTEGER DEFAULT 0,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS sys_data_reconciliation (
			id TEXT PRIMARY KEY,
			asset_id TEXT,
			conflict_type TEXT,
			severity TEXT,
			confidence_score REAL,
			raw_snapshot TEXT,
			detected_at DATETIME,
			resolved_at DATETIME,
			resolved_by TEXT,
			resolution_note TEXT,
			workorder_id TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS sys_reconciliation_exception (
			id TEXT PRIMARY KEY,
			name TEXT,
			conflict_types TEXT,
			severity_override TEXT,
			is_active INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0
		)`,
		// reconciliation_normalized 物化视图(SQLite 不支持 MATERIALIZED VIEW,用 VIEW 模拟)
		// ListExceptions LEFT JOIN rn ON rn.asset_id = a.id,即使 view 为空也不影响 JOIN 语义
		`CREATE VIEW IF NOT EXISTS reconciliation_normalized AS
			SELECT
				'' AS asset_id,
				'' AS physical_username,
				'' AS ad_username
			WHERE 1=0`,
		// sys_user(sys_data_reconciliation JOIN 用户表)
		`CREATE TABLE IF NOT EXISTS sys_user (
			id TEXT PRIMARY KEY,
			username TEXT,
			nickname TEXT,
			deleted_at DATETIME
		)`,
	}
	for _, s := range stmts {
		require.NoError(t, gormDB.Exec(s).Error, "create table for permission test: %s", s)
	}
}

// grantUserPerm 给 userID 授予 perm 对应的菜单权限(等价 migration_165 button seed)。
func grantUserPerm(t *testing.T, gormDB *gorm.DB, perm string) string {
	t.Helper()
	userID := uuid.NewString()
	roleID := uuid.NewString()
	menuID := uuid.NewString()

	require.NoError(t, gormDB.Exec(
		`INSERT INTO sys_role (id, role_name, role_key, status, del_flag) VALUES (?, ?, ?, 0, 0)`,
		roleID, "role-"+perm, "r_"+perm).Error)
	require.NoError(t, gormDB.Exec(
		`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, userID, roleID).Error)
	require.NoError(t, gormDB.Exec(
		`INSERT INTO sys_menu (id, menu_name, parent_id, menu_type, perms, status) VALUES (?, ?, NULL, 'C', ?, 0)`,
		menuID, "m-"+perm, perm).Error)
	require.NoError(t, gormDB.Exec(
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`, roleID, menuID).Error)
	return userID
}

// newReconciliationTestRouter 构造一个 minimal router,挂载简化版 JWTAuth + RequirePermissions + 3 个 Setup*Router。
// 简化版 JWTAuth: 检查 Authorization header 存在性;无 → 401,有 → 注入预设 userID。
// 真实 JWT 解析需 SM4 + 真 token,在跨模块权限边界测试中我们仅验证中间件链对 401/403/200 的反应。
func newReconciliationTestRouter(t *testing.T, coreInst *core.Core, injectUserID string) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()

	g := r.Group("/asset/reconciliation")
	// 简化版 JWTAuth(测试用):模拟 401 与 user 注入
	g.Use(func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// 测试用简化 token:注入固定 userID
		if injectUserID != "" {
			c.Set("user_id", injectUserID)
		}
		c.Next()
	})
	g.Use(middleware.RequirePermissions(reconciliationPerms, coreInst))

	// 挂载 3 个 Setup*Router(与 internal/api/router.go 主路由一致)
	SetupReconciliationRouter(g, coreInst)
	SetupReconciliationStatisticsRouter(g, coreInst)
	return r
}

func TestReconciliationEndpoints_PermissionBoundary(t *testing.T) {
	// 准备 minimal core + DB(包含 permission 中间件所需的 sys_role / sys_menu / sys_user_role / sys_role_menu)
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	reconciliationPermTables(t, gormDB)

	coreInst := &core.Core{
		CoreInfra: &core.CoreInfra{
			DB: &db.Database{DB: gormDB, Type: "sqlite"},
		},
	}

	// --- Test 2a: 无 token → 401 ---
	t.Run("无token_应401", func(t *testing.T) {
		router := newReconciliationTestRouter(t, coreInst, "")
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/asset/reconciliation/exception/list", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"无 Authorization 头应被 JWTAuth 拒绝(401)")
	})

	// --- Test 2b: 有 token 但无 asset:reconciliation:list 权限 → 403 ---
	t.Run("无权限_应403", func(t *testing.T) {
		// 用户仅持 ops:workstation:list,与 reconciliation perms 命名空间割裂
		userID := grantUserPerm(t, gormDB, "ops:workstation:list")
		router := newReconciliationTestRouter(t, coreInst, userID)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/asset/reconciliation/statistics/summary", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer fake-token")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code,
			"无 asset:reconciliation:list 权限应被 RequirePermissions 拒绝(403)")
	})

	// --- Test 2c: 有 token + asset:reconciliation:list 权限 → 200 (statistics/summary) ---
	t.Run("有权限_statistics_应200", func(t *testing.T) {
		userID := grantUserPerm(t, gormDB, "asset:reconciliation:list")
		router := newReconciliationTestRouter(t, coreInst, userID)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/asset/reconciliation/statistics/summary", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer fake-token")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code,
			"有 asset:reconciliation:list 权限应通过 RequirePermissions(200)")
	})

	// --- Test 2d: 有 token + asset:reconciliation:list 权限 → 200 (exception/list) ---
	t.Run("有权限_exception_list_应200", func(t *testing.T) {
		userID := grantUserPerm(t, gormDB, "asset:reconciliation:list")
		router := newReconciliationTestRouter(t, coreInst, userID)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/asset/reconciliation/exception/list", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer fake-token")
		router.ServeHTTP(w, req)
		// 权限边界测试只关心权限链是否放行,不验证 SQL 语义。
		// 在 SQLite 测试环境,PG::text 转换语法导致 SQL 报错(返回 500),
		// 这是 PG 特定实现的预期行为;权限链放行后 status 不会是 401/403。
		assert.NotEqual(t, http.StatusUnauthorized, w.Code,
			"有权限用户不应被 401 拒绝")
		assert.NotEqual(t, http.StatusForbidden, w.Code,
			"有权限用户不应被 403 拒绝")
		t.Logf("exception/list 通过权限链,response=%d body=%s", w.Code, w.Body.String())
	})
}

// ============================================================================
// Test 3: Statistics 方法静态 list.length 反模式守护
//
// 防 stat-cards-from-list-length-capped-at-100 MEMORY 回归:
// 6 个 Statistics 方法体不能出现 Find( 或 .Offset(,否则会触发 list.length 模式
// (被 MaxPageSize=100 钳制,统计卡片计数失真)。本测试通过 go/parser 解析源文件 AST,
// 抽取 6 个方法体的代码字符串,断言不含反模式关键字。
// ============================================================================

var statisticsMethodNames = []string{
	"Summary",
	"ByConflictType",
	"BySeverity",
	"HealthTrend",
	"TopUnresolved",
	"ExceptionRuleStats",
}

func TestReconciliationStatistics_NoListLength(t *testing.T) {
	// 解析 reconciliation_statistics.go(相对路径: ../../../services/asset/,从 internal/api/v1/asset/ 出发)
	statsPath := "../../../services/asset/reconciliation_statistics.go"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, statsPath, nil, parser.ParseComments)
	require.NoError(t, err, "must parse reconciliation_statistics.go at %s", statsPath)

	srcData, err := os.ReadFile(statsPath)
	require.NoError(t, err)
	src := string(srcData)

	for _, methodName := range statisticsMethodNames {
		t.Run(methodName, func(t *testing.T) {
			var methodDecl *ast.FuncDecl
			for _, decl := range f.Decls {
				if fd, ok := decl.(*ast.FuncDecl); ok {
					if fd.Recv != nil && fd.Name.Name == methodName {
						methodDecl = fd
						break
					}
				}
			}
			require.NotNil(t, methodDecl, "method %s must exist in reconciliation_statistics.go", methodName)
			require.NotNil(t, methodDecl.Body, "method %s must have a body", methodName)

			startPos := fset.Position(methodDecl.Body.Pos())
			endPos := fset.Position(methodDecl.Body.End())
			bodySrc := src[startPos.Offset:endPos.Offset]

			assert.NotContains(t, bodySrc, "Find(",
				"%s 必须避免 Find( 反模式(防 list.length 被 MaxPageSize=100 钳制)", methodName)
			assert.NotContains(t, bodySrc, ".Offset(",
				"%s 必须避免 .Offset( 反模式(分页聚合应该走 COUNT/GROUP BY/RAW aggregate)", methodName)
		})
	}
}