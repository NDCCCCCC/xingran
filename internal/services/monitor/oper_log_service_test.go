package monitor

// oper_log_service_test.go — Phase 73-04 Task 3 (D-02 ad_account 范本: glebarez sqlite + 真实 service).
//
// D-03 LOCK: oper_log_service 不豁免(strict-balance 原则)。
// Phase 72 已覆盖 api/v1/monitor/oper_log_handler(71.2%),但 handler 层覆盖不替代
// service 层覆盖 —— 本文件直接测 operLogService 的 List/GetByID/Delete/BatchDelete/Clean,
// 与 handler 测试相互独立。任何人不得以"handler 已测"为由删除本文件。
//
// 表名: models.OperLog.TableName() = sys_oper_log(DDL 复用 Phase 72 oper_log_handler_test.go)。
//
// 已锁定的业务 quirk(见 73-04-SUMMARY.md,NOT fixed per D-12):
//   Q4: List 的 OperName 过滤用 `operator_name LIKE`,但真实列名是 oper_name ——
//       sqlite/PG 上都是"列不存在"错误,该过滤器在生产环境必然报错。测试锁定 error 行为。
//   Q5: OrderByColumn 非空但不在白名单 → ApplySort 忽略且 service 不追加默认
//       oper_time DESC → 无显式排序(与 login_log 同款行为)。

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

// setupTestOperLogDB 创建 sys_oper_log 表(DDL 复用 Phase 72 oper_log_handler_test.go)。
func setupTestOperLogDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_oper_log (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			title TEXT,
			business_type INTEGER,
			method TEXT,
			request_method TEXT,
			operator_type INTEGER,
			oper_name TEXT,
			nickname TEXT,
			dept_name TEXT,
			oper_url TEXT,
			oper_ip TEXT,
			oper_location TEXT,
			oper_param TEXT,
			json_result TEXT,
			status INTEGER,
			error_msg TEXT,
			oper_time DATETIME,
			cost_time INTEGER
		)
	`).Error)
	return db
}

// newOperLogServiceFixture 构造真实 service。
func newOperLogServiceFixture(db *gorm.DB) *operLogService {
	return NewOperLogService(db).(*operLogService)
}

// seedOperLog 插入一行操作日志(显式 id 便于断言)。
func seedOperLog(t *testing.T, db *gorm.DB, id, title, operName string, businessType, status int, operTime time.Time) {
	t.Helper()
	require.NoError(t, db.Create(&models.OperLog{
		BaseTimeLine: models.BaseTimeLine{ID: id},
		Title:        title,
		BusinessType: businessType,
		OperatorName: &operName,
		Status:       status,
		OperTime:     operTime,
		CostTime:     int64(50),
	}).Error)
}

func TestOperLogService_CompileOnly(t *testing.T) {
	svc := newOperLogServiceFixture(setupTestOperLogDB(t))
	assert.NotNil(t, svc)
}

func TestOperLogService_DefaultListParams(t *testing.T) {
	params := DefaultOperLogListParams()
	assert.Equal(t, 1, params.Current)
	assert.Equal(t, 10, params.PageSize)
}

// ==================== List ====================

func TestOperLogService_List_Empty(t *testing.T) {
	svc := newOperLogServiceFixture(setupTestOperLogDB(t))
	result, err := svc.List(context.Background(), DefaultOperLogListParams())
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
	assert.Empty(t, result.List)
}

func TestOperLogService_List_AllSeeded_DefaultOrderDesc(t *testing.T) {
	db := setupTestOperLogDB(t)
	base := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	seedOperLog(t, db, "op-1", "用户管理", "admin", 1, 0, base)
	seedOperLog(t, db, "op-2", "角色管理", "zhangsan", 2, 1, base.Add(2*time.Hour))
	seedOperLog(t, db, "op-3", "部门管理", "lisi", 1, 0, base.Add(1*time.Hour))

	svc := newOperLogServiceFixture(db)
	result, err := svc.List(context.Background(), DefaultOperLogListParams())
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.Total)
	require.Len(t, result.List, 3)

	list := result.List.([]models.OperLog)
	// 无 OrderByColumn → oper_time DESC
	assert.Equal(t, "op-2", list[0].ID)
	assert.Equal(t, "op-3", list[1].ID)
	assert.Equal(t, "op-1", list[2].ID)
}

func TestOperLogService_List_FilterTitle(t *testing.T) {
	db := setupTestOperLogDB(t)
	base := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	seedOperLog(t, db, "op-1", "用户管理", "admin", 1, 0, base)
	seedOperLog(t, db, "op-2", "角色管理", "zhangsan", 2, 1, base)

	svc := newOperLogServiceFixture(db)
	title := "用户"
	result, err := svc.List(context.Background(), OperLogListParams{
		BaseListRequest: baseListReq(1, 10), Title: &title,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)

	// 空串过滤值不过滤
	empty := ""
	result2, err := svc.List(context.Background(), OperLogListParams{
		BaseListRequest: baseListReq(1, 10), Title: &empty,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), result2.Total)
}

func TestOperLogService_List_FilterBusinessType(t *testing.T) {
	db := setupTestOperLogDB(t)
	base := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	seedOperLog(t, db, "op-1", "用户管理", "admin", 1, 0, base)
	seedOperLog(t, db, "op-2", "角色管理", "zhangsan", 2, 1, base)
	seedOperLog(t, db, "op-3", "岗位管理", "lisi", 1, 0, base)

	svc := newOperLogServiceFixture(db)
	bt := 1
	result, err := svc.List(context.Background(), OperLogListParams{
		BaseListRequest: baseListReq(1, 10), BusinessType: &bt,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
}

func TestOperLogService_List_FilterStatus(t *testing.T) {
	db := setupTestOperLogDB(t)
	base := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	seedOperLog(t, db, "op-1", "用户管理", "admin", 1, 0, base)
	seedOperLog(t, db, "op-2", "角色管理", "zhangsan", 2, 1, base)

	svc := newOperLogServiceFixture(db)
	status := 0 // OperLogStatusSuccess
	result, err := svc.List(context.Background(), OperLogListParams{
		BaseListRequest: baseListReq(1, 10), Status: &status,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// quirk Q4 lock: OperName 过滤用错误列名 operator_name → 查询报错(生产同样报错)。
func TestOperLogService_List_FilterOperName_BrokenColumn_Error(t *testing.T) {
	db := setupTestOperLogDB(t)
	base := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	seedOperLog(t, db, "op-1", "用户管理", "admin", 1, 0, base)

	svc := newOperLogServiceFixture(db)
	operName := "admin"
	_, err := svc.List(context.Background(), OperLogListParams{
		BaseListRequest: baseListReq(1, 10), OperName: &operName,
	})
	require.Error(t, err, "quirk Q4: 过滤列名 operator_name 不存在(真实列是 oper_name)")
	assert.Contains(t, err.Error(), "查询操作日志总数失败")
}

func TestOperLogService_List_FilterTimeRange(t *testing.T) {
	db := setupTestOperLogDB(t)
	base := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	seedOperLog(t, db, "op-1", "用户管理", "admin", 1, 0, base)                    // 09:00
	seedOperLog(t, db, "op-2", "角色管理", "zhangsan", 2, 1, base.Add(3*time.Hour)) // 12:00

	svc := newOperLogServiceFixture(db)
	begin := "2026-08-20 10:00:00"
	end := "2026-08-20 13:00:00"
	result, err := svc.List(context.Background(), OperLogListParams{
		BaseListRequest: baseListReq(1, 10), BeginTime: &begin, EndTime: &end,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total, "只有 12:00 的行落在 [10:00, 13:00] 窗口内")
	list := result.List.([]models.OperLog)
	assert.Equal(t, "op-2", list[0].ID)
}

func TestOperLogService_List_Pagination(t *testing.T) {
	db := setupTestOperLogDB(t)
	base := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		seedOperLog(t, db, uuid.NewString(), "批量", "admin", 1, 0, base.Add(time.Duration(i)*time.Minute))
	}

	svc := newOperLogServiceFixture(db)
	p1, err := svc.List(context.Background(), OperLogListParams{BaseListRequest: baseListReq(1, 2)})
	require.NoError(t, err)
	assert.Equal(t, int64(5), p1.Total)
	assert.Len(t, p1.List, 2)

	p3, err := svc.List(context.Background(), OperLogListParams{BaseListRequest: baseListReq(3, 2)})
	require.NoError(t, err)
	assert.Equal(t, int64(5), p3.Total)
	assert.Len(t, p3.List, 1)
}

func TestOperLogService_List_SortByCostTimeDesc(t *testing.T) {
	db := setupTestOperLogDB(t)
	opTime := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	require.NoError(t, db.Create(&models.OperLog{
		BaseTimeLine: models.BaseTimeLine{ID: "op-fast"}, Title: "快", OperTime: opTime, CostTime: 10,
	}).Error)
	require.NoError(t, db.Create(&models.OperLog{
		BaseTimeLine: models.BaseTimeLine{ID: "op-slow"}, Title: "慢", OperTime: opTime, CostTime: 900,
	}).Error)

	svc := newOperLogServiceFixture(db)
	desc := false
	result, err := svc.List(context.Background(), OperLogListParams{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: "costTime", IsAsc: &desc},
	})
	require.NoError(t, err)
	list := result.List.([]models.OperLog)
	assert.Equal(t, "op-slow", list[0].ID, "costTime DESC → 900ms 在前")
	assert.Equal(t, "op-fast", list[1].ID)
}

func TestOperLogService_List_SortByOperNameAsc(t *testing.T) {
	db := setupTestOperLogDB(t)
	opTime := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	seedOperLog(t, db, "op-z", "z模块", "zhangsan", 1, 0, opTime)
	seedOperLog(t, db, "op-a", "a模块", "admin", 1, 0, opTime)

	svc := newOperLogServiceFixture(db)
	result, err := svc.List(context.Background(), OperLogListParams{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: "operName"},
	})
	require.NoError(t, err)
	list := result.List.([]models.OperLog)
	assert.Equal(t, "op-a", list[0].ID, "operName ASC → admin 在前")
	assert.Equal(t, "op-z", list[1].ID)
}

// quirk Q5 lock: 非白名单排序列 → 忽略且不追加默认排序;无注入。
func TestOperLogService_List_InvalidSortColumnIgnored_NoInjection(t *testing.T) {
	db := setupTestOperLogDB(t)
	opBase := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	seedOperLog(t, db, "op-1", "用户管理", "admin", 1, 0, opBase)
	seedOperLog(t, db, "op-2", "角色管理", "zhangsan", 2, 1, opBase.Add(time.Hour))

	svc := newOperLogServiceFixture(db)
	result, err := svc.List(context.Background(), OperLogListParams{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10, OrderByColumn: "evil; DROP TABLE sys_oper_log"},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
	assert.Len(t, result.List, 2)

	var count int64
	require.NoError(t, db.Model(&models.OperLog{}).Count(&count).Error)
	assert.Equal(t, int64(2), count, "表未被注入破坏")
}

func TestOperLogService_List_CountError(t *testing.T) {
	db := setupTestOperLogDB(t)
	require.NoError(t, db.Exec("DROP TABLE sys_oper_log").Error)
	svc := newOperLogServiceFixture(db)

	_, err := svc.List(context.Background(), DefaultOperLogListParams())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询操作日志总数失败")
}

// ==================== GetByID ====================

func TestOperLogService_GetByID_Found(t *testing.T) {
	db := setupTestOperLogDB(t)
	seedOperLog(t, db, "op-1", "用户管理", "admin", 1, 0, time.Now())

	svc := newOperLogServiceFixture(db)
	log, err := svc.GetByID(context.Background(), "op-1")
	require.NoError(t, err)
	assert.Equal(t, "用户管理", log.Title)
	require.NotNil(t, log.OperatorName)
	assert.Equal(t, "admin", *log.OperatorName)
	assert.Equal(t, 1, log.BusinessType)
	assert.Equal(t, int64(50), log.CostTime)
}

func TestOperLogService_GetByID_NotFound(t *testing.T) {
	svc := newOperLogServiceFixture(setupTestOperLogDB(t))
	_, err := svc.GetByID(context.Background(), "missing-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "操作日志不存在")
}

// ==================== Delete ====================

func TestOperLogService_Delete_Success(t *testing.T) {
	db := setupTestOperLogDB(t)
	seedOperLog(t, db, "op-1", "用户管理", "admin", 1, 0, time.Now())

	svc := newOperLogServiceFixture(db)
	require.NoError(t, svc.Delete(context.Background(), "op-1"))

	var count int64
	require.NoError(t, db.Model(&models.OperLog{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestOperLogService_Delete_NotFound(t *testing.T) {
	svc := newOperLogServiceFixture(setupTestOperLogDB(t))
	err := svc.Delete(context.Background(), "missing-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "操作日志不存在")
}

// ==================== BatchDelete ====================

func TestOperLogService_BatchDelete_Success(t *testing.T) {
	db := setupTestOperLogDB(t)
	opTime := time.Now()
	seedOperLog(t, db, "op-1", "用户管理", "admin", 1, 0, opTime)
	seedOperLog(t, db, "op-2", "角色管理", "zhangsan", 2, 1, opTime)
	seedOperLog(t, db, "op-3", "部门管理", "lisi", 1, 0, opTime)

	svc := newOperLogServiceFixture(db)
	require.NoError(t, svc.BatchDelete(context.Background(), []string{"op-1", "op-2", "op-3"}))

	var count int64
	require.NoError(t, db.Model(&models.OperLog{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestOperLogService_BatchDelete_EmptyIDs(t *testing.T) {
	svc := newOperLogServiceFixture(setupTestOperLogDB(t))
	err := svc.BatchDelete(context.Background(), []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ids不能为空")
}

func TestOperLogService_BatchDelete_NonexistentIDs_NoError(t *testing.T) {
	svc := newOperLogServiceFixture(setupTestOperLogDB(t))
	assert.NoError(t, svc.BatchDelete(context.Background(), []string{"nope-1"}))
}

func TestOperLogService_BatchDelete_DBError(t *testing.T) {
	db := setupTestOperLogDB(t)
	require.NoError(t, db.Exec("DROP TABLE sys_oper_log").Error)
	svc := newOperLogServiceFixture(db)

	err := svc.BatchDelete(context.Background(), []string{"op-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "批量删除操作日志失败")
}

// ==================== Clean ====================

func TestOperLogService_Clean_Success(t *testing.T) {
	db := setupTestOperLogDB(t)
	seedOperLog(t, db, "op-1", "用户管理", "admin", 1, 0, time.Now())
	seedOperLog(t, db, "op-2", "角色管理", "zhangsan", 2, 1, time.Now())

	svc := newOperLogServiceFixture(db)
	require.NoError(t, svc.Clean(context.Background()))

	var count int64
	require.NoError(t, db.Model(&models.OperLog{}).Count(&count).Error)
	assert.Equal(t, int64(0), count, "Clean 应清空 sys_oper_log(表名匹配,与 login_log 的 Q3 quirk 不同)")
}

func TestOperLogService_Clean_DBError(t *testing.T) {
	db := setupTestOperLogDB(t)
	require.NoError(t, db.Exec("DROP TABLE sys_oper_log").Error)
	svc := newOperLogServiceFixture(db)

	err := svc.Clean(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "清空操作日志失败")
}
