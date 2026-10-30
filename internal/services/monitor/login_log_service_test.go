package monitor

// login_log_service_test.go — Phase 73-04 Task 2 (D-02 ad_account 范本: glebarez sqlite + 真实 service).
//
// 范本要点(镜像 internal/api/v1/system/ad_account_handler_test.go):
//   - glebarez sqlite :memory: + 手写 CREATE TABLE(列对齐 internal/models/log.go LoginLog)
//   - 真实 NewLoginLogService(db) + 真实 List/GetByID/Delete/BatchDelete/Clean 调用
//   - 无 mock service —— 直接验证 SQL 行为与 DB 状态变迁
//
// 表名说明: models.LoginLog.TableName() = sys_logininfor(Phase 72 login_log_handler_test.go 同款 DDL)。
//
// 已锁定的业务 quirk(见 73-04-SUMMARY.md,NOT fixed per D-12):
//   Q3: loginLogService.Clean() 的原生 SQL 是 `DELETE FROM sys_login_log`,但模型表名是
//       sys_logininfor —— 生产库中 sys_login_log 不存在(全库无该表迁移),Clean 在生产
//       Postgres 上必然失败。测试用 stub 表覆盖 happy 分支 + 无 stub 表锁定 error 分支。

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// baseListReq 构造通用分页参数的便捷 helper。
func baseListReq(current, pageSize int) base.BaseListRequest {
	return base.BaseListRequest{Current: current, PageSize: pageSize}
}

// setupTestLoginLogDB 创建 sys_logininfor 表(DDL 复用 Phase 72 login_log_handler_test.go)。
func setupTestLoginLogDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_logininfor (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			user_name TEXT,
			nickname TEXT,
			ipaddr TEXT,
			login_location TEXT,
			browser TEXT,
			os TEXT,
			status INTEGER,
			msg TEXT,
			login_time DATETIME
		)
	`).Error)
	return db
}

// createLoginLogCleanStub 创建 Clean() 原生 SQL 指向的 sys_login_log stub 表(quirk Q3)。
func createLoginLogCleanStub(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_login_log (
			id TEXT PRIMARY KEY
		)
	`).Error)
}

// newLoginLogServiceFixture 构造真实 service。
func newLoginLogServiceFixture(db *gorm.DB) *loginLogService {
	return NewLoginLogService(db).(*loginLogService)
}

// seedLoginLog 插入一行登录日志(显式 id 便于断言)。
func seedLoginLog(t *testing.T, db *gorm.DB, id, username, ipaddr string, status int, loginTime time.Time) {
	t.Helper()
	require.NoError(t, db.Create(&models.LoginLog{
		BaseTimeLine: models.BaseTimeLine{ID: id},
		Username:     username,
		IPAddr:       ipaddr,
		Status:       status,
		LoginTime:    loginTime,
	}).Error)
}

func TestLoginLogService_CompileOnly(t *testing.T) {
	svc := newLoginLogServiceFixture(setupTestLoginLogDB(t))
	assert.NotNil(t, svc)
}

func TestLoginLogService_DefaultListParams(t *testing.T) {
	params := DefaultLoginLogListParams()
	assert.Equal(t, 1, params.Current)
	assert.Equal(t, 10, params.PageSize)
}

// ==================== List ====================

func TestLoginLogService_List_Empty(t *testing.T) {
	svc := newLoginLogServiceFixture(setupTestLoginLogDB(t))
	result, err := svc.List(context.Background(), DefaultLoginLogListParams())
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
	assert.Empty(t, result.List)
	assert.Equal(t, 1, result.Current)
	assert.Equal(t, 10, result.PageSize)
}

func TestLoginLogService_List_AllSeeded_DefaultOrderDesc(t *testing.T) {
	db := setupTestLoginLogDB(t)
	base := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	seedLoginLog(t, db, "log-1", "admin", "10.0.0.1", 0, base)
	seedLoginLog(t, db, "log-2", "zhangsan", "10.0.0.2", 1, base.Add(2*time.Hour))
	seedLoginLog(t, db, "log-3", "lisi", "10.0.0.3", 0, base.Add(1*time.Hour))

	svc := newLoginLogServiceFixture(db)
	result, err := svc.List(context.Background(), DefaultLoginLogListParams())
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.Total)
	require.Len(t, result.List, 3)

	list := result.List.([]models.LoginLog)
	// 无 OrderByColumn → login_time DESC
	assert.Equal(t, "log-2", list[0].ID)
	assert.Equal(t, "log-3", list[1].ID)
	assert.Equal(t, "log-1", list[2].ID)
}

func TestLoginLogService_List_FilterUsername(t *testing.T) {
	db := setupTestLoginLogDB(t)
	base := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	seedLoginLog(t, db, "log-1", "admin", "10.0.0.1", 0, base)
	seedLoginLog(t, db, "log-2", "zhangsan", "10.0.0.2", 1, base)

	svc := newLoginLogServiceFixture(db)
	username := "zhang"
	result, err := svc.List(context.Background(), LoginLogListParams{
		BaseListRequest: baseListReq(1, 10), Username: &username,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	list := result.List.([]models.LoginLog)
	assert.Equal(t, "zhangsan", list[0].Username)

	// 空串过滤值不过滤
	empty := ""
	result2, err := svc.List(context.Background(), LoginLogListParams{
		BaseListRequest: baseListReq(1, 10), Username: &empty,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), result2.Total)
}

func TestLoginLogService_List_FilterIPAddr(t *testing.T) {
	db := setupTestLoginLogDB(t)
	base := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	seedLoginLog(t, db, "log-1", "admin", "10.62.1.1", 0, base)
	seedLoginLog(t, db, "log-2", "zhangsan", "192.168.0.9", 1, base)

	svc := newLoginLogServiceFixture(db)
	ip := "10.62"
	result, err := svc.List(context.Background(), LoginLogListParams{
		BaseListRequest: baseListReq(1, 10), IPAddr: &ip,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

func TestLoginLogService_List_FilterStatus(t *testing.T) {
	db := setupTestLoginLogDB(t)
	base := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	seedLoginLog(t, db, "log-1", "admin", "10.0.0.1", 0, base)
	seedLoginLog(t, db, "log-2", "zhangsan", "10.0.0.2", 1, base)
	seedLoginLog(t, db, "log-3", "lisi", "10.0.0.3", 0, base)

	svc := newLoginLogServiceFixture(db)
	status := 0 // 0 = 成功(LoginLogStatusSuccess)
	result, err := svc.List(context.Background(), LoginLogListParams{
		BaseListRequest: baseListReq(1, 10), Status: &status,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
}

func TestLoginLogService_List_FilterTimeRange(t *testing.T) {
	db := setupTestLoginLogDB(t)
	base := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	seedLoginLog(t, db, "log-1", "admin", "10.0.0.1", 0, base)               // 10:00
	seedLoginLog(t, db, "log-2", "zhangsan", "10.0.0.2", 1, base.Add(2*time.Hour)) // 12:00

	svc := newLoginLogServiceFixture(db)
	begin := "2026-08-20 11:00:00"
	end := "2026-08-20 13:00:00"
	result, err := svc.List(context.Background(), LoginLogListParams{
		BaseListRequest: baseListReq(1, 10), BeginTime: &begin, EndTime: &end,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total, "只有 12:00 的行落在 [11:00, 13:00] 窗口内")
	list := result.List.([]models.LoginLog)
	assert.Equal(t, "log-2", list[0].ID)

	// 空串时间不过滤
	empty := ""
	result2, err := svc.List(context.Background(), LoginLogListParams{
		BaseListRequest: baseListReq(1, 10), BeginTime: &empty, EndTime: &empty,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), result2.Total)
}

func TestLoginLogService_List_Pagination(t *testing.T) {
	db := setupTestLoginLogDB(t)
	base := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		seedLoginLog(t, db, uuid.NewString(), "u"+string(rune('a'+i)), "10.0.0.1", 0, base.Add(time.Duration(i)*time.Minute))
	}

	svc := newLoginLogServiceFixture(db)
	p1, err := svc.List(context.Background(), LoginLogListParams{BaseListRequest: baseListReq(1, 2)})
	require.NoError(t, err)
	assert.Equal(t, int64(5), p1.Total)
	assert.Len(t, p1.List, 2)

	p3, err := svc.List(context.Background(), LoginLogListParams{BaseListRequest: baseListReq(3, 2)})
	require.NoError(t, err)
	assert.Equal(t, int64(5), p3.Total)
	assert.Len(t, p3.List, 1)
}

func TestLoginLogService_List_SortByLoginTimeAsc(t *testing.T) {
	db := setupTestLoginLogDB(t)
	loginBase := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	seedLoginLog(t, db, "log-new", "admin", "10.0.0.1", 0, loginBase.Add(2*time.Hour))
	seedLoginLog(t, db, "log-old", "zhangsan", "10.0.0.2", 1, loginBase)

	svc := newLoginLogServiceFixture(db)
	result, err := svc.List(context.Background(), LoginLogListParams{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: "loginTime"},
	})
	require.NoError(t, err)
	list := result.List.([]models.LoginLog)
	assert.Equal(t, "log-old", list[0].ID, "loginTime ASC → 最早的在前")
	assert.Equal(t, "log-new", list[1].ID)
}

func TestLoginLogService_List_SortByNickname_Desc(t *testing.T) {
	db := setupTestLoginLogDB(t)
	loginBase := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	nickA, nickZ := "阿飞", "祖峰"
	require.NoError(t, db.Create(&models.LoginLog{
		BaseTimeLine: models.BaseTimeLine{ID: "log-a"}, Username: "a", Nickname: &nickA, LoginTime: loginBase,
	}).Error)
	require.NoError(t, db.Create(&models.LoginLog{
		BaseTimeLine: models.BaseTimeLine{ID: "log-z"}, Username: "z", Nickname: &nickZ, LoginTime: loginBase,
	}).Error)

	svc := newLoginLogServiceFixture(db)
	desc := false
	result, err := svc.List(context.Background(), LoginLogListParams{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: "nickname", IsAsc: &desc},
	})
	require.NoError(t, err)
	require.Len(t, result.List, 2)
}

func TestLoginLogService_List_InvalidSortColumnIgnored(t *testing.T) {
	db := setupTestLoginLogDB(t)
	loginBase := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	seedLoginLog(t, db, "log-1", "admin", "10.0.0.1", 0, loginBase)
	seedLoginLog(t, db, "log-2", "zhangsan", "10.0.0.2", 1, loginBase.Add(time.Hour))

	svc := newLoginLogServiceFixture(db)
	// 非白名单列 → ApplySort 忽略(不注入 SQL);且因 OrderByColumn 非空,
	// service 不再追加默认 login_time DESC → 无显式排序(natural order)。
	// 这里锁定: 无错 + 2 行全返回 + 不被注入破坏。
	result, err := svc.List(context.Background(), LoginLogListParams{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: "evil; DROP TABLE sys_logininfor"},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
	assert.Len(t, result.List, 2)

	// 表未被注入破坏,后续查询仍可用
	var count int64
	require.NoError(t, db.Model(&models.LoginLog{}).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

func TestLoginLogService_List_CountError(t *testing.T) {
	db := setupTestLoginLogDB(t)
	require.NoError(t, db.Exec("DROP TABLE sys_logininfor").Error)
	svc := newLoginLogServiceFixture(db)

	_, err := svc.List(context.Background(), DefaultLoginLogListParams())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询登录日志总数失败")
}

// ==================== GetByID ====================

func TestLoginLogService_GetByID_Found(t *testing.T) {
	db := setupTestLoginLogDB(t)
	seedLoginLog(t, db, "log-1", "admin", "10.0.0.1", 0, time.Now())

	svc := newLoginLogServiceFixture(db)
	log, err := svc.GetByID(context.Background(), "log-1")
	require.NoError(t, err)
	assert.Equal(t, "admin", log.Username)
	assert.Equal(t, "10.0.0.1", log.IPAddr)
}

func TestLoginLogService_GetByID_NotFound(t *testing.T) {
	svc := newLoginLogServiceFixture(setupTestLoginLogDB(t))
	_, err := svc.GetByID(context.Background(), "missing-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "登录日志不存在")
}

// ==================== Delete ====================

func TestLoginLogService_Delete_Success(t *testing.T) {
	db := setupTestLoginLogDB(t)
	seedLoginLog(t, db, "log-1", "admin", "10.0.0.1", 0, time.Now())

	svc := newLoginLogServiceFixture(db)
	require.NoError(t, svc.Delete(context.Background(), "log-1"))

	var count int64
	require.NoError(t, db.Model(&models.LoginLog{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestLoginLogService_Delete_NotFound(t *testing.T) {
	svc := newLoginLogServiceFixture(setupTestLoginLogDB(t))
	err := svc.Delete(context.Background(), "missing-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "登录日志不存在")
}

// ==================== BatchDelete ====================

func TestLoginLogService_BatchDelete_Success(t *testing.T) {
	db := setupTestLoginLogDB(t)
	base := time.Now()
	seedLoginLog(t, db, "log-1", "admin", "10.0.0.1", 0, base)
	seedLoginLog(t, db, "log-2", "zhangsan", "10.0.0.2", 1, base)
	seedLoginLog(t, db, "log-3", "lisi", "10.0.0.3", 0, base)

	svc := newLoginLogServiceFixture(db)
	require.NoError(t, svc.BatchDelete(context.Background(), []string{"log-1", "log-3"}))

	var count int64
	require.NoError(t, db.Model(&models.LoginLog{}).Count(&count).Error)
	assert.Equal(t, int64(1), count, "仅剩 log-2")
}

func TestLoginLogService_BatchDelete_EmptyIDs(t *testing.T) {
	svc := newLoginLogServiceFixture(setupTestLoginLogDB(t))
	err := svc.BatchDelete(context.Background(), []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ids不能为空")
}

func TestLoginLogService_BatchDelete_NonexistentIDs_NoError(t *testing.T) {
	svc := newLoginLogServiceFixture(setupTestLoginLogDB(t))
	// GORM IN 删除不命中时为成功 no-op
	assert.NoError(t, svc.BatchDelete(context.Background(), []string{"nope-1", "nope-2"}))
}

func TestLoginLogService_BatchDelete_DBError(t *testing.T) {
	db := setupTestLoginLogDB(t)
	require.NoError(t, db.Exec("DROP TABLE sys_logininfor").Error)
	svc := newLoginLogServiceFixture(db)

	err := svc.BatchDelete(context.Background(), []string{"log-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "批量删除登录日志失败")
}

// ==================== Clean (quirk Q3) ====================

// quirk Q3 lock: Clean 的原生 SQL 指向 sys_login_log(stub 表存在 → happy 分支),
// 且 sys_logininfor 业务表的数据【不会】被清掉(表名不匹配)。
func TestLoginLogService_Clean_StubTable_Success_DoesNotTouchRealTable(t *testing.T) {
	db := setupTestLoginLogDB(t)
	createLoginLogCleanStub(t, db)
	seedLoginLog(t, db, "log-1", "admin", "10.0.0.1", 0, time.Now())

	svc := newLoginLogServiceFixture(db)
	require.NoError(t, svc.Clean(context.Background()))

	var count int64
	require.NoError(t, db.Model(&models.LoginLog{}).Count(&count).Error)
	assert.Equal(t, int64(1), count, "quirk Q3: Clean 删的是 sys_login_log,业务表 sys_logininfor 数据仍在")
}

// quirk Q3 lock(生产行为): 无 sys_login_log 表(生产 PG 即此状态)→ Clean 报错。
func TestLoginLogService_Clean_NoStubTable_Error(t *testing.T) {
	db := setupTestLoginLogDB(t) // 无 stub 表
	seedLoginLog(t, db, "log-1", "admin", "10.0.0.1", 0, time.Now())

	svc := newLoginLogServiceFixture(db)
	err := svc.Clean(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "清空登录日志失败")

	var count int64
	require.NoError(t, db.Model(&models.LoginLog{}).Count(&count).Error)
	assert.Equal(t, int64(1), count, "失败路径下数据不动")
}
