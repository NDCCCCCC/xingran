# Phase 59: 登录回写守卫与部门对账视图 - Pattern Map

**Mapped:** 2026-10-19
**Files analyzed:** 19（新建 9 / 修改 8 / 仅引用 2）
**Analogs found:** 17 / 19
**Analog 追踪校验:** 所有被引用的 analog 均通过 `git ls-files` 确认是 git-tracked 源码（非 gitignore 的安装/运行时镜像）。

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/services/system/user_sync_service.go`（修改 :417-430） | service | event-driven（登录同步）+ request-response | 同文件 `SyncADUser` 现状 + `internal/services/addomain/ad_push_queue_service.go`（队列语义） | exact |
| `internal/services/system/user_sync_service_phase59_test.go`（新建） | test | unit / sqlite in-memory | `internal/services/system/user_sync_service_test.go`（`setupTestDBForSync`） | exact |
| `internal/models/ad_push_queue.go`（**不改**） | model | — | 自身 `ADPushQueueActivePredicate` / `ADPushEntityUser` / `ADPushActionUpdateUser` | 仅引用 |
| `internal/core/db/migrations/migration_217_dept_ou_mismatch_view.go`（新建） | migration | DDL（VIEW，双方言） | `migration_215_create_ad_push_queue.go`（方言分支）+ `migration_175_reconciliation_physical_link.go`（PG VIEW）+ `sqlite_reconciliation_views.go`（sqlite DROP+CREATE） | role-match（组合 3 个先例） |
| `internal/core/db/migrations/migration_217_dept_ou_mismatch_view_test.go`（新建） | test | migration / sqlite | `migration_216_adpush_drain_sysconfig_test.go` | exact |
| `internal/core/db/migrations/migration_218_ad_reconciliation_permission.go`（新建，可合并入 217） | migration | permission seed（`sys_menu` + `sys_role_menu`） | `migration_163_ad_account_pool_menu.go` | exact |
| `internal/core/db/database.go`（修改：DROP 钩子 + 3 条注册路径） | config / bootstrap | startup DDL | 同文件 `ensureADPushQueue` / `Migrate216` 注册块 | exact |
| `internal/core/db/sqlite_reconciliation_views.go`（修改：追加 DROP） | utility | startup DDL | 同文件 `dropSQLiteReconciliationViews`（:113-129） | exact |
| `internal/services/addomain/dept_ou_mismatch_service.go`（新建） | service | CRUD（只读分页） | `internal/services/addomain/login_history_query.go` | exact |
| `internal/services/addomain/dept_ou_mismatch_service_test.go`（新建） | test | unit / sqlite view | `internal/services/addomain/login_history_query_test.go` | role-match |
| `internal/api/v1/system/ad_reconciliation_handler.go`（新建） | controller（handler） | request-response | `internal/api/v1/system/ad_login_history_handler.go` | exact |
| `internal/api/v1/system/ad_reconciliation_handler_test.go`（新建） | test | handler（httptest） | 同目录既有 handler 测试（`httptest` 模式） | role-match |
| `internal/api/v1/system/ad_domain_router.go`（修改） | route | request-response | 同文件 `accounts.readGroup`（:51-59） | exact |
| `ruoyi-react-frontend/src/lib/adDomainApi.ts`（修改） | utility / API client | request-response | 同文件 `getADComputerLoginHistory`（:463-469）+ `withDefaultPagination`（:225-229） | exact |
| `ruoyi-react-frontend/src/lib/adDomainApi.test.ts`（修改） | test | unit | 同文件 `describe("adDomainApi — 用户与电脑")`（:206） | exact |
| `ruoyi-react-frontend/src/pages/ad-domain/configs/index.tsx`（修改） | component | request-response（Tabs 壳） | 同文件详情 Drawer 的 `Tabs`（:438-497） | exact |
| `ruoyi-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.tsx`（新建） | component | CRUD（只读列表） | `src/pages/ad-domain/configs/AccountPoolTab.tsx` | exact |
| `ruoyi-react-frontend/src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx`（新建） | test | component | `src/pages/ad-domain/__tests__/accountPoolTab.render.test.tsx` | exact |

---

## Pattern Assignments

### `internal/services/system/user_sync_service.go`（service, event-driven）

**Analog:** 同文件 `SyncADUser`（`user_sync_service.go:405-433`）+ 队列常量（`internal/models/ad_push_queue.go:53`）

**Imports pattern**（现状 `:1-16`；守卫只需新增标准库 `errors`）：
```go
import (
    "context"
    "fmt"
    "strings"
    "sync"
    "time"

    "github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
    "github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
    applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
    "golang.org/x/sync/errgroup"
    "gorm.io/gorm"
)
```
> 新增方法需要 `errors.Is(err, gorm.ErrRecordNotFound)`，补 `"errors"`。

**插入点现状**（`user_sync_service.go:417-430`，D-04 只包住 `Update`）：
```go
if category == catLocalSameName {
    applogger.Infof("[SyncADUser] 关联绑定用户保留原部门: userID=%s, deptID=%v", user.ID, user.DeptID)
} else if adUserInfo.OUDN != "" && s.ouMapper != nil {
    deptID, err := s.resolveDeptFromOU(ctx, adUserInfo.OUDN)
    if err != nil {
        applogger.Warnf("解析部门失败（不影响同步）: %v", err)
    } else if deptID == "" {
        applogger.Infof("[SyncADUser] 无OU映射，保留原部门: userID=%s, deptID=%v", user.ID, user.DeptID)
    } else if err := s.db.WithContext(ctx).Model(user).Update("dept_id", deptID).Error; err != nil {
        applogger.Warnf("更新用户部门失败: %v", err)
    } else {
        applogger.Infof("[SyncADUser] 成功设置用户部门: userID=%s, deptID=%s", user.ID, deptID)
    }
}
```

**守卫查询模式**（D-02/D-03/D-07/D-08；常量逐字来自 `ad_push_queue.go:53`）：
```go
const ADPushQueueActivePredicate = "status IN ('pending', 'failed')"
// 实体/动作枚举：ad_push_queue.go:17 ADPushEntityUser = "user"、:25 ADPushActionUpdateUser = "update_user"

var qrow models.ADPushQueue
qerr := s.db.WithContext(ctx).
    Where("entity_id = ? AND entity_type = ? AND action_type = ?",
        user.ID, models.ADPushEntityUser, models.ADPushActionUpdateUser).
    Where(models.ADPushQueueActivePredicate).   // ⚠ 不得改写、不得加表名限定
    Take(&qrow).Error

switch {
case qerr != nil && !errors.Is(qerr, gorm.ErrRecordNotFound):
    // D-07：查询失败 → 保守跳过覆盖
    applogger.Warnf("部门回写守卫查询失败，跳过 dept_id 更新: ... err=%v", qerr)
case qerr == nil:
    // D-05：全量诊断字段（queueID / actionType / attempts）
    applogger.Warnf("部门回写被活跃队列守卫拦截: ... queueID=%s, actionType=%s, attempts=%d",
        qrow.ID, qrow.ActionType, qrow.Attempts)
default:
    // D-08：无活跃行 → 与现状完全相同的 Update 调用
    if err := s.db.WithContext(ctx).Model(user).Update("dept_id", deptID).Error; err != nil {
        applogger.Warnf("更新用户部门失败: %v", err)
    } else {
        applogger.Infof("[SyncADUser] 成功设置用户部门: userID=%s, deptID=%s", user.ID, deptID)
    }
}
```

**指针 deref 防御**（`user.DeptID` 是 `*string`，见 `internal/models/user.go:20`；现有 :418/:425 `%v` 会打印地址）：
```go
func deptValue(p *string) string {
    if p == nil { return "<nil>" }
    return *p
}
```

**支持索引已存在**（`migration_215_create_ad_push_queue.go:91-96`）：
```go
// 实体维度查询 (Phase 59 登录守卫按 user_id 查有无 pending/failed 行)
CREATE INDEX IF NOT EXISTS idx_ad_push_queue_entity_id ON ad_push_queue (entity_id)
```

**反模式（D-08 红线）**：不要新建 `tryUpdateDept` 平行分支；不要把守卫提前到 `resolveDeptFromOU` 之前；不要比较时间戳或新旧值。

---

### `internal/services/system/user_sync_service_phase59_test.go`（test, sqlite in-memory）

**Analog:** `internal/services/system/user_sync_service_test.go:16-...`（`setupTestDBForSync`）

**测试基座模式**（`:16-24`）：
```go
func setupTestDBForSync(t *testing.T) *gorm.DB {
    t.Helper()
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
        DisableForeignKeyConstraintWhenMigrating: true,
    })
    if err != nil { t.Skipf("Failed to open test database: %v", err) }
    ...
}
```
- 现有 fixture 已含 `sys_user` / `sys_dept` / `sys_ad_config` / `ad_push_queue`（RESEARCH Validation Wave 0 已确认）。
- 本 phase 需补 `CREATE TABLE sys_dept_ou_mapping (...)`（`dept_id TEXT, ad_config_id TEXT, ou_dn TEXT, sync_enabled INTEGER, ...`）。
- WARN 断言使用 `applogger.SetTestBuffer`（`pkg/logger/logger.go:225-233`）：
```go
buf := &bytes.Buffer{}
restore := applogger.SetTestBuffer(buf)
defer restore()
```
- 必测用例：`pending` / `failed` 拦截；`pushed` / `dead` 放行；`disable` 动作放行；无行则 `dept_id` 等于 AD 解析值；`DROP TABLE ad_push_queue` 模拟查询失败 → 跳过 + WARN。

---

### `internal/core/db/migrations/migration_217_dept_ou_mismatch_view.go`（migration, 双方言 VIEW）

**Analogs:**
- 方言判定：`migration_helpers.go:7-9`
- 双方言 DDL + 幂等 + 错误日志：`migration_215_create_ad_push_queue.go:30-99`
- PG `CREATE OR REPLACE VIEW`：`migration_175_reconciliation_physical_link.go:72-110`
- sqlite `DROP VIEW IF EXISTS` + `CREATE VIEW`：`sqlite_reconciliation_views.go:30-33, 113-129, 299-323`

**方言分支骨架**（`migration_215:65-72` 同款）：
```go
func Migrate217DeptOUMismatchView(db *gorm.DB) error {
    log.Println("Running migration 217: create v_dept_ou_mismatch view")
    body := buildDeptOUMismatchViewSelect() // 共享 SQL body，无方言分叉
    if isPostgreSQL(db) {
        return db.Exec("CREATE OR REPLACE VIEW v_dept_ou_mismatch AS " + body).Error
    }
    // sqlite 无 CREATE OR REPLACE VIEW（sqlite_reconciliation_views.go:30）
    if err := db.Exec("DROP VIEW IF EXISTS v_dept_ou_mismatch").Error; err != nil { return err }
    return db.Exec("CREATE VIEW v_dept_ou_mismatch AS " + body).Error
}
```

**PG 先例**（`migration_175:72-73`）：
```go
createPhysicalChainSQL := `
CREATE OR REPLACE VIEW reconciliation_physical_chain AS
WITH latest_mac AS ( ... )
SELECT ...
`
```

**sqlite 先例**（`sqlite_reconciliation_views.go:300-323`）：
```go
drops := []string{"DROP VIEW IF EXISTS reconciliation_normalized", ...}
creates := []struct{ name, ddl string }{ ... }
for _, stmt := range drops { d.db.Exec(stmt) }
for _, c := range creates { d.db.Exec(c.ddl) }
```

**共享 body 的硬约束**（来自 RESEARCH Pattern 2 / Pitfalls）：
- 每个 UNION ALL 分支的 `dept_id` / `ad_config_id` / `user_id` 一律 `CAST(... AS TEXT)`；`dept_deleted_at` 同样 `CAST(... AS TEXT)`（PG uuid/varchar 不能 UNION）。
- 布尔比较写 `NOT mp.sync_enabled`，不要 `= 0`。
- 禁止 `ILIKE` / `REGEXP_REPLACE` / `::`；后缀匹配用 `SUBSTR + LENGTH`，避免 `LIKE '%,' || base` 的通配符污染。
- 归一化表达式用 Go helper 生成（折叠空格 + 分段 `TRIM` + `LOWER`），并与 `NormalizeDN`（`dn_util.go:36-56`）逐输入对齐。
- 最长后缀唯一匹配在 SQL 中必须复刻 `MatchADConfigByDN`（`ad_config_resolver.go:39-81`）：`MAX(nbl)` + `cnt = 1`（等长并列视为 unrouted）。
- 列顺序必须逐字对齐 D-11：`dept_id, dept_name, dept_deleted_at, ad_config_id, config_name, mapped_ou_dn, user_id, username, user_ad_ou_dn, mismatch_type, mismatch_detail`。
- 数据源字段：`sys_dept_ou_mapping.dept_id/ad_config_id/ou_dn/sync_enabled`（`dept_ou_mapping.go:12-28`）；`sys_user.dept_id/ad_ou_dn/username`（`user.go:20,34-35`）；`sys_ad_config.base_dn/status/config_name`，`ADConfigStatusEnabled = 0`（`ad_domain.go:16-19,22-41`）；`sys_dept.deleted_at`（`BaseModel`）。

---

### `internal/core/db/migrations/migration_217_dept_ou_mismatch_view_test.go`（test, sqlite migration）

**Analog:** `migration_216_adpush_drain_sysconfig_test.go:11-59`

**测试基座模式**：
```go
func freshSQLiteDBForMigrate217(t *testing.T) *gorm.DB {
    t.Helper()
    dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
    db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
    if err != nil { t.Fatalf("open sqlite: %v", err) }
    return db
}
```
- `db.AutoMigrate(&models.User{}, &models.Department{}, &models.DeptOUMapping{}, &models.ADConfig{})` 建表（sqlite 下 `sanitizeSQLiteModelDefaults` 不在测试路径，必要时用显式 `CREATE TABLE` 绕开 `gen_random_uuid()`）。
- 断言：迁移一次可建视图；重复执行幂等（sqlite 分支先 DROP）；四类口径各造一行/不含一行。
- 一致性用例见 `dept_ou_mismatch_service_test.go`（父子域、逗号边界、空 base、停用、软删、等长歧义）。

---

### `internal/core/db/migrations/migration_218_ad_reconciliation_permission.go`（migration, permission seed）

**Analog:** `migration_163_ad_account_pool_menu.go:16-89`（按钮权限、无独立菜单）

**权限行模式**（`:40-68`）：
```go
emptyPath := ""
icon := "#"
btnPerms := "ops:ad:config:reconciliation:list"

var configMenu models.Menu
if err := db.Where("menu_name = ? AND menu_type = ?", "AD域控配置", "C").First(&configMenu).Error; err != nil {
    log.Printf("Warning: Could not find AD域控配置 menu, skip %s: %v", btnPerms, err)
} else {
    buttonMenu := &models.Menu{
        MenuName: "对账查询",
        ParentID: &configMenu.ID,
        Path:     &emptyPath,
        MenuType: "F",
        Visible:  0,
        Status:   0,
        Perms:    &btnPerms,
        Icon:     &icon,
        OrderNum: 30,
        Remark:   "查看部门-OU 对账不一致条目",
    }
    db.Create(buttonMenu)
}
```
**授权模式**（`:71-89`）：按父菜单已授权角色精确插入（推荐，避免把权限撒给所有角色）：
```sql
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT rm.role_id, ? FROM sys_role_menu rm WHERE rm.menu_id = ?
ON CONFLICT DO NOTHING
```
> 依据 memory `migration-grant-new-menu-precision-helper`：只插 `sys_menu` 不插 `sys_role_menu` 会导致角色勾选视觉联动但实际无权。若沿用 migration_163 的「全 status=0 角色」写法，需在 plan 中显式说明这是更宽的授权面。
> RESEARCH Open Question 1 已指出：D-17「不新增菜单 seed」应解释为「不新增可导航菜单/路由」，`menu_type='F'` 按钮权限行是权限载体，必须新增，否则前后端权限数组都取不到。

---

### `internal/core/db/database.go`（config/bootstrap, 修改）

**Analog:** 同文件 `ensureADPushQueue`（:1315-1329）与 Migrate216 的三处注册（PG :1061-1066 / sqlite :1100-1104 / Bootstrap :1288-1292）

**AutoMigrate 前置 DROP（PG）**（`dropDependentMaterializedViews` 当前 noop，`:471-488`）：
```go
func (d *Database) dropDependentMaterializedViews() {
    // Phase 59: v_dept_ou_mismatch 引用 sys_user/sys_dept/sys_dept_ou_mapping/sys_ad_config
    // 普通 VIEW 同样会阻塞 GORM ALTER TYPE（0A000，见 :445-451）
    if err := d.DB.Exec("DROP VIEW IF EXISTS v_dept_ou_mismatch").Error; err != nil {
        applogger.Warnf("[dropDependent] DROP v_dept_ou_mismatch 失败: %v", err)
    }
    ...
}
```
> 该视图无 MV 依赖，无需 CASCADE。

**三条注册路径**（缺一不可，参照 Phase 57 F-03 盲区）：
1. PG 分支：`database.go` advisory-lock 块内，紧接 `Migrate216`（:1064-1066）之后、非阻断。
2. sqlite 分支：`Migrate216`（:1102-1104）之后、`ensureSQLiteReconciliationViews` 之前/之后均可，非阻断。
3. `BootstrapMissingTables`：`Migrate216`（:1290-1292）之后补调。

**调用风格**（非阻断，与 Migrate216 同档）：
```go
if err := migrations.Migrate217DeptOUMismatchView(d.DB); err != nil {
    applogger.Errorf("v_dept_ou_mismatch 视图创建失败 (非阻断,留待下次启动): %v", err)
}
```
> 不采用 `ensureADPushQueue` 的「失败阻断启动」档：视图缺失只会让对账端点 500，不影响登录/入队主链路。

---

### `internal/core/db/sqlite_reconciliation_views.go`（utility, 修改）

**Analog:** 同文件 `dropSQLiteReconciliationViews`（:113-129）

```go
drops := []string{
    "DROP VIEW IF EXISTS reconciliation_normalized",
    "DROP VIEW IF EXISTS reconciliation_physical_chain",
    "DROP VIEW IF EXISTS reconciliation_user_lookup",
    "DROP VIEW IF EXISTS v_dept_ou_mismatch", // Phase 59
}
```
> 位置：追加到现有 DROP 列表即可（AutoMigrate 的 sqlite 分支 :830 已调用）；重建由 migration_217 负责。

---

### `internal/services/addomain/dept_ou_mismatch_service.go`（service, 只读分页）

**Analog:** `internal/services/addomain/login_history_query.go:22-91`

**接口 + 私有实现 + 构造函数**（`:22-36` 同款）：
```go
type DeptOUMismatchService interface {
    ListDeptOUMismatch(ctx context.Context, req *DeptOUMismatchQuery) ([]DeptOUMismatchRow, int64, error)
}

type deptOUMismatchServiceImpl struct{ db *gorm.DB }

func NewDeptOUMismatchService(db *gorm.DB) DeptOUMismatchService {
    return &deptOUMismatchServiceImpl{db: db}
}
```

**DTO**（RESEARCH Code Example 3；ID/时间统一 `*string` 规避跨方言类型）：
```go
type DeptOUMismatchQuery struct {
    Current      int    `json:"current"`
    PageSize     int    `json:"pageSize"`
    MismatchType string `json:"mismatchType,omitempty"`
    DeptID       string `json:"deptId,omitempty"`
    Keyword      string `json:"keyword,omitempty"`
}

type DeptOUMismatchRow struct {
    DeptID         *string `json:"deptId"`
    DeptName       *string `json:"deptName"`
    DeptDeletedAt  *string `json:"deptDeletedAt"`
    ADConfigID     *string `json:"adConfigId"`
    ConfigName     *string `json:"configName"`
    MappedOUDN     *string `json:"mappedOuDn"`
    UserID         *string `json:"userId"`
    Username       *string `json:"username"`
    UserADOUDN     *string `json:"userAdOuDn"`
    MismatchType   string  `json:"mismatchType"`
    MismatchDetail string  `json:"mismatchDetail"`
}
```

**分页 + Count + Scan**（照 `login_history_query.go:46-90` 的「先 Count 后分页」结构）：
```go
rows := make([]DeptOUMismatchRow, 0) // 空结果必须序列化为 [] 而非 null
q := s.db.WithContext(ctx).Table("v_dept_ou_mismatch")
if req.MismatchType != "" { q = q.Where("mismatch_type = ?", req.MismatchType) } // 四值白名单
if req.DeptID != ""       { q = q.Where("dept_id = ?", req.DeptID) }
if kw := strings.TrimSpace(req.Keyword); kw != "" {
    like := "%" + escapeLike(strings.ToLower(kw)) + "%"
    q = q.Where(`LOWER(COALESCE(username,'')) LIKE ? ESCAPE '\'
             OR LOWER(COALESCE(dept_name,'')) LIKE ? ESCAPE '\' ...`, like, like, like, like)
}
if err := q.Count(&total).Error; err != nil { return nil, 0, fmt.Errorf("统计对账总数失败: %w", err) }
err := q.Order("mismatch_type, dept_name, username, dept_id, user_id"). // 必须有确定性 ORDER BY
    Offset((req.Current-1)*req.PageSize).Limit(req.PageSize).Scan(&rows).Error
```
> `escapeLike` 是本 phase 新增的小 helper（转义 `\` / `%` / `_`），仓库无现成函数。
> `mismatchType` 建议白名单校验（四值），非法值直接 400 或忽略——在 plan 中二选一并锁定。

---

### `internal/services/addomain/dept_ou_mismatch_service_test.go`（test）

**Analog:** `internal/services/addomain/login_history_query_test.go`（同包 sqlite 基座）

**Go/SQL 一致性测试骨架**（RESEARCH Code Example 4）：
```go
cfg, err := addomain.MatchADConfigByDN(tc.adOuDn, configs)
goRouted := err == nil
sqlRouted := viewHasConfigUnroutedRow(t, db, tc.userID) == false
assert.Equal(t, goRouted, sqlRouted, "case=%s", tc.name)
```
必测表（RESEARCH 已列）：最长后缀 / 逗号边界 / 无匹配 / 空 base / 停用配置 / 软删配置 / 等长歧义。

---

### `internal/api/v1/system/ad_reconciliation_handler.go`（controller, request-response）

**Analog:** `internal/api/v1/system/ad_login_history_handler.go:13-49`

**Handler 结构 + 内部构造 service + 只读不记 operlog**：
```go
type ADReconciliationHandler struct {
    svc addomainServices.DeptOUMismatchService
}

func NewADReconciliationHandler(db *gorm.DB) *ADReconciliationHandler {
    return &ADReconciliationHandler{svc: addomainServices.NewDeptOUMismatchService(db)}
}

func (h *ADReconciliationHandler) ListDeptOUMismatch(c *gin.Context) {
    var req addomainServices.DeptOUMismatchQuery
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
        return
    }
    req.Current, req.PageSize = query.NormalizePagination(req.Current, req.PageSize)
    list, total, err := h.svc.ListDeptOUMismatch(c.Request.Context(), &req)
    if err != nil {
        response.Error(c, err)
        return
    }
    response.Page(c, list, total, req.Current, req.PageSize)
}
```
> 只读端点**不调用** `operlog.Record`；不设置 `oper_log_title` 即 middleware 不落日志（`pkg/middleware/oper_log.go:86-89`）。

---

### `internal/api/v1/system/ad_domain_router.go`（route, 修改）

**Analog:** 同文件 `accounts.readGroup`（:51-59）

**路由注册模式**（挂在现有 `adDomain` 组下，最终 URL `/api/v1/ad-domain/reconciliation/mismatch`）：
```go
// Phase 59: 部门↔OU 对账（只读）
recon := r.Group("/reconciliation")
recon.Use(middleware.RequirePermissions([]string{"ops:ad:config:reconciliation:list"}, core))
{
    recon.POST("/mismatch", NewADReconciliationHandler(core.GetDB()).ListDeptOUMismatch)
}
```
> 路径契约对齐：D-14 字面 `/system/ad-config/...` 与真实挂载不一致（`internal/api/router.go:519` 为 `/ad-domain`），前端沿用 `/ad-domain/reconciliation/mismatch`，见 RESEARCH Open Question 2。
> `RequirePermissions` 为「任一权限」语义（`pkg/middleware/permission.go:292-298`），单权限点即可。

---

### `internal/api/v1/system/ad_reconciliation_handler_test.go`（test）

**Analog:** 同目录既有 handler 测试。若无直接先例，采用 `gin.CreateTestContext` + `httptest`；断言：
- 非法 JSON → `400` / `CodeParamError`
- 缺省 `current/pageSize` → `query.NormalizePagination` 后为 1/10（`pkg/constants/pagination.go`）
- service 返回错误 → `response.Error` 路径

---

### `ruoyi-react-frontend/src/lib/adDomainApi.ts`（API client, 修改）

**Analog:** 同文件 `getADComputerLoginHistory`（:463-469）+ `withDefaultPagination`（:225-229）

```ts
// ==================== 部门-OU 对账 API（Phase 59） ====================
export interface DeptOUMismatchRow {
  deptId?: string;
  deptName?: string;
  deptDeletedAt?: string | null;
  adConfigId?: string;
  configName?: string;
  mappedOuDn?: string;
  userId?: string;
  username?: string;
  userAdOuDn?: string;
  mismatchType: "user_ou_mismatch" | "dept_soft_deleted" | "mapping_disabled" | "config_unrouted";
  mismatchDetail: string;
}

export function getDeptOUMismatchList(params: {
  current?: number;
  pageSize?: number;
  mismatchType?: string;
  deptId?: string;
  keyword?: string;
}): Promise<BaseResponse<PageResponse<DeptOUMismatchRow>>> {
  return post("/ad-domain/reconciliation/mismatch", withDefaultPagination(params));
}
```
> 必须走 `import { post } from "./api"`（文件顶部 :1 已有），禁止原生 axios。

---

### `ruoyi-react-frontend/src/lib/adDomainApi.test.ts`（test, 修改）

**Analog:** 同文件既有 `describe` 块（:206、:358）

新增断言：请求路径为 `/ad-domain/reconciliation/mismatch`；未传分页时 `current=1, pageSize=10`（`withDefaultPagination` 契约）。

---

### `ruoyi-react-frontend/src/pages/ad-domain/configs/index.tsx`（component, 修改）

**Analog:** 同文件详情 Drawer 的 `Tabs`（:438-497）

**现状结构**（页面根为 `<Card>`，:323-500）：配置表格与 Modal/Drawer 全部直接挂在 Card 下。改造为：
```tsx
<Card>
  <Tabs
    activeKey={activeTab}
    onChange={setActiveTab}
    items={[
      { key: "configs", label: "配置列表", children: <现有表格+工具栏+Modal+Drawer> },
      { key: "reconciliation", label: "对账", children: <ReconciliationTab /> },
    ]}
  />
</Card>
```
**Tab 持久化模式**（UI-SPEC 要求，与 `settings-page/index.tsx` 同款）：
```tsx
import { usePersistedStateController } from "@/hooks/usePersistedState";
const [activeTab, setActiveTab] = usePersistedStateController<string>({
  keyPrefix: location.pathname,
  keySuffix: "activeTab",
  defaultValue: "configs",
});
```
> 现有 `Drawer` 内的 `Tabs`（:439-496）是**嵌套先例**，不是页面级 Tab；页面级用外层 `Tabs` 包住现有内容即可，现有逻辑零改动。

---

### `ruoyi-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.tsx`（component, 只读列表）

**Analog:** `src/pages/ad-domain/configs/AccountPoolTab.tsx`

**状态与请求模式**（`AccountPoolTab.tsx:62-80` 同款简单 `useState`，不引入 React Query）：
```tsx
const [list, setList] = useState<DeptOUMismatchRow[]>([]);
const [total, setTotal] = useState(0);
const [loading, setLoading] = useState(false);
const [current, setCurrent] = useState(1);
const [pageSize, setPageSize] = useState(20);
const [mismatchType, setMismatchType] = useState<string>();
const [deptId, setDeptId] = useState<string>();
const [keyword, setKeyword] = useState("");
```

**权限门（D-17 / UI-SPEC）**：
```tsx
import { useMenuStore } from "@/store/menuStore";
const permissions = useMenuStore((s) => s.permissions);
const canView = permissions.includes("ops:ad:config:reconciliation:list");
if (!canView) {
  return <Result status="403" title="无对账查看权限"
    subTitle="请联系管理员为当前角色授予 ops:ad:config:reconciliation:list 权限后重新登录。" />;
}
```
> Tab 始终可见；无权限在 Tab 内渲染 403 且**不发请求**（与 `LocationAliasDrawer.tsx:59-63` 的 permission 读取写法一致）。

**加载模式**（CLAUDE.md useEffect 稳定性规则）：
```tsx
const loadList = useCallback(async () => {
  setLoading(true);
  try {
    const res = await getDeptOUMismatchList({ current, pageSize, mismatchType, deptId, keyword });
    setList(res.data?.list ?? []);
    setTotal(res.data?.total ?? 0);
  } catch (e) { setError(e as Error); }
  finally { setLoading(false); }
}, [current, pageSize, mismatchType, deptId, keyword]);
useEffect(() => { if (canView) loadList(); }, [canView, loadList]);
```
> 依赖只用原始值，禁把对象/数组放进依赖。

**状态组件**：
- 错误：`ErrorAlertWithRetry`（`description="对账数据加载失败，请稍后重试。"`, `onRetry={loadList}`），不要叠加多个 message。
- 空态：`EmptyStateWithAction`（标题/描述见 UI-SPEC Copywriting Contract）。
- 表格：7 列只读，`rowKey` 由组合键生成（`` `${mismatchType}-${deptId ?? ""}-${userId ?? ""}-${adConfigId ?? ""}` ``），`scroll={{ x: 1400 }}`。
- 颜色：`mismatch_type` 用 antd 预设 `Tag color`（orange/red/gold/blue），禁硬编码 hex。

**部门筛选**：`DepartmentTreeSelect`（`@/components/shared`）+ `useDeptTree()`。

---

### `ruoyi-react-frontend/src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx`（test, component）

**Analog:** `src/pages/ad-domain/__tests__/accountPoolTab.render.test.tsx:1-70`

**Mock 模式**（`:6-12`）：
```tsx
vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});
vi.mock("@/lib/adDomainApi", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/adDomainApi")>();
  return { ...actual, getDeptOUMismatchList: vi.fn().mockResolvedValue({ code: 0, data: { list: [], total: 0, current: 1, pageSize: 20 } }) };
});
import { renderWithProviders } from "@/test/utils/renderWithProviders";
```
断言：无权限 → 渲染 403 且 mock 未被调用；有权限 → 表格/空态；接口失败 → `ErrorAlertWithRetry`。

---

## Shared Patterns

### Response Wrapping
**Source:** `pkg/response/response.go:123-136`
**Apply to:** `ad_reconciliation_handler.go`
```go
response.Page(c, list, total, req.Current, req.PageSize)
// Data: PageResponse{List, Total, Current, PageSize}
```

### Pagination Normalization
**Source:** `pkg/query/pagination.go:43-62`
**Apply to:** 对账 handler
```go
req.Current, req.PageSize = query.NormalizePagination(req.Current, req.PageSize)
// current<=0 → DefaultCurrent; pageSize<=0 → DefaultPageSize; > MaxPageSize(200) → 钳制
```

### Permission Middleware
**Source:** `pkg/middleware/permission.go:277-303`；调用先例 `ad_domain_router.go:51-59`
**Apply to:** 新路由组
```go
middleware.RequirePermissions([]string{"ops:ad:config:reconciliation:list"}, core)
```

### operlog 只读豁免
**Source:** `pkg/middleware/oper_log.go:86-89`
**Apply to:** `ad_reconciliation_handler.go`
> 只读端点不调用 `operlog.Record`；不设置 `oper_log_title` 即不写日志。

### 双方言迁移注册
**Source:** `database.go`（PG :1053-1066 / sqlite :1095-1104 / Bootstrap :1275-1292）
**Apply to:** migration_217 / 218
> 三条启动路径必须同时注册；视图还需在 AutoMigrate 前 DROP（PG `dropDependentMaterializedViews`，sqlite `dropSQLiteReconciliationViews`）。

### 队列活跃谓词（单一事实源）
**Source:** `internal/models/ad_push_queue.go:42-53`
**Apply to:** GUARD-01 查询
```go
const ADPushQueueActivePredicate = "status IN ('pending', 'failed')"
```
> ⚠ 不得改写、不得加表名限定（D-02 one-way 决策）；migration_215 部分唯一索引与 `adPushOnConflictClause` 共用该文本。

### 前端权限读取
**Source:** `ruoyi-react-frontend/src/pages/operations/workstations/LocationAliasDrawer.tsx:59-63`
**Apply to:** `ReconciliationTab.tsx`
```tsx
const permissions = useMenuStore((s) => s.permissions);
const canView = permissions.includes("ops:ad:config:reconciliation:list");
```

### 前端 API 包装
**Source:** `ruoyi-react-frontend/src/lib/adDomainApi.ts:1,225-229,463-469`
**Apply to:** 新增 `getDeptOUMismatchList`
```ts
import { get, post } from "./api";
function withDefaultPagination<T extends { current?: number; pageSize?: number }>(params: T): T {
  return { ...DEFAULT_PAGINATION, ...params };
}
```

### 日志与测试断言
**Source:** `pkg/logger/logger.go:225-233,271-273`
**Apply to:** GUARD-01 WARN 与测试
```go
applogger.Warnf("...", user.ID, user.Username, localDept, deptID, qrow.ID, qrow.ActionType, qrow.Attempts)
buf := &bytes.Buffer{}
restore := applogger.SetTestBuffer(buf)
```

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| （无） | — | — | 所有新建文件均有同角色或同数据流先例；migration_217 的「单视图 UNION ALL 四类口径」是组合先例（PG VIEW + sqlite DROP/CREATE），无逐字同形的现成文件。 |

## 规划期必须显式记录的边界（非模式缺口）

1. **权限种子 vs D-17 字面冲突**：不新增可导航菜单/路由，但必须新增 `menu_type='F'` 权限行（RESEARCH Open Question 1）。否则前端 `permissions` 恒不含新权限点，Tab 永远 403。
2. **端点路径**：统一 `/api/v1/ad-domain/reconciliation/mismatch`，不额外挂 `/system/ad-config` 别名（RESEARCH Open Question 2）。
3. **视图口径缺口**：「部门无任何映射」的用户不属于 D-10 四类，plan 的 Known Gaps 需记录。
4. **`config_unrouted` 歧义语义**：等长并列按 unrouted 处理（`best_cnt.cnt = 1` 复刻 Go error 语义）。
5. **`dead` 行不纳入守卫**：D-02/D-03 锁定 pending/failed；58-07 的建议留待后续 phase。

## Metadata

**Analog search scope:** `internal/services/system/`、`internal/services/addomain/`、`internal/api/v1/system/`、`internal/core/db/`、`internal/models/`、`ruoyi-react-frontend/src/lib/`、`ruoyi-react-frontend/src/pages/ad-domain/`、`ruoyi-react-frontend/src/hooks/`、`ruoyi-react-frontend/src/components/shared/`
**Files scanned:** 约 30
**Pattern extraction date:** 2026-10-19
