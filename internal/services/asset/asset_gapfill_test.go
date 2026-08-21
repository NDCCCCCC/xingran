package asset

import (
	"bytes"
	"context"
	"database/sql"
	"mime/multipart"
	"strings"
	"sync"
	"testing"
	"time"

	pgconn "github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	pkgcache "github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// =====================================================================
// Phase 74-07: asset 包 0% 函数补测 —
// cache_keys / snapshot / reconciliation_service(GetByWorkstation 链) /
// fix_suggestion(GetByID/Apply/Rollback/generator/monitor) /
// exception(MatchException/ImportFromExcel) / workorder helpers /
// detection+statistics 构造器。
// =====================================================================

// newGapTestDB 在 setupTestDB 基础上补齐本文件所需的表:
// ops_asset 重建为模型全列(Apply 的 First(&models.Asset{}) 需要全部列)、
// 工位/工位设备/信息点/网络设备/配置/部门/操作日志/修复建议表。
func newGapTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupTestDB(t, t.Name())

	// ops_asset 全列重建(sqlite 视图按名解析,先 DROP 后建不影响 view)
	require.NoError(t, db.Exec(`DROP TABLE ops_asset`).Error)
	assetSchema, err := schema.Parse(&models.Asset{}, &sync.Map{}, db.NamingStrategy)
	require.NoError(t, err)
	var ddl strings.Builder
	ddl.WriteString("CREATE TABLE ops_asset (id TEXT PRIMARY KEY")
	timeCols := map[string]bool{"created_at": true, "updated_at": true, "deleted_at": true}
	for _, name := range assetSchema.DBNames {
		if name == "id" {
			continue
		}
		if timeCols[name] {
			ddl.WriteString(", " + name + " DATETIME")
		} else {
			ddl.WriteString(", " + name + " TEXT")
		}
	}
	ddl.WriteString(")")
	require.NoError(t, db.Exec(ddl.String()).Error)

	require.NoError(t, db.Exec(`CREATE TABLE sys_workstation (id TEXT PRIMARY KEY, workstation_name TEXT, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE ops_workstation_device (id TEXT PRIMARY KEY, workstation_id TEXT, asset_id TEXT, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE ops_info_points (id TEXT PRIMARY KEY, workstation_id TEXT, device_id TEXT, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_network_device (id TEXT PRIMARY KEY, ip_address TEXT, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_config (config_key TEXT, config_value TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_dept (id TEXT PRIMARY KEY, dept_name TEXT, dept_code TEXT, ancestors TEXT, status INTEGER, deleted_at DATETIME)`).Error)
	require.NoError(t, db.AutoMigrate(&models.OperLog{}))

	// 例外规则 upsert 目标唯一索引(UpsertKey=name)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX ux_gap_exc_name ON sys_reconciliation_exception(name)`).Error)

	// sys_reconciliation_fix_suggestion(与 setupFixSuggestionTestDB 同构)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_reconciliation_fix_suggestion (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			exception_id TEXT,
			suggested_user_id TEXT,
			pre_fix_user_id TEXT,
			confidence_score REAL,
			reason TEXT,
			fix_status TEXT,
			conflict_type TEXT,
			accepted_at DATETIME,
			rejected_at DATETIME,
			applied_at DATETIME,
			rolled_back_at DATETIME,
			accepted_by TEXT,
			rejected_by TEXT,
			applied_by TEXT,
			rolled_back_by TEXT,
			rejection_reason TEXT,
			rollback_reason TEXT,
			rollback_window_until DATETIME,
			superseded_at DATETIME,
			apply_client_ip TEXT,
			rollback_client_ip TEXT
		)
	`).Error)
	return db
}

// =====================================================================
// cache_keys.go
// =====================================================================

func TestCacheKeyBuildersAndStripPrefix(t *testing.T) {
	assert.Equal(t, "reconciliation:dashboard:global", GetReconciliationDashboardKey("global"))
	assert.Equal(t, "reconciliation:exception:list:dept-1", GetReconciliationExceptionListKey("dept-1"))
	assert.Equal(t, "reconciliation:exception:byID:rec-1", GetReconciliationExceptionByIDKey("rec-1"))
	assert.Equal(t, "reconciliation:exceptionRule:list", GetReconciliationExceptionRuleListKey())
	assert.Equal(t, "reconciliation:exceptionRule:byID:r-1", GetReconciliationExceptionRuleByIDKey("r-1"))
	assert.Equal(t, "reconciliation:view:lastRefresh", GetReconciliationViewLastRefreshKey())
	assert.Equal(t, "reconciliation:health:workstation:ws-1", GetReconciliationHealthByWorkstationKey("ws-1"))
	assert.Equal(t, "reconciliation:health:asset:a-1", GetReconciliationHealthByAssetKey("a-1"))

	assert.Equal(t, "health:ws-1", StripCachePrefix("xingran:health:ws-1"))
	assert.Equal(t, "no-prefix", StripCachePrefix("no-prefix"))
}

func TestInvalidateWorkstationHealthCacheKey(t *testing.T) {
	ctx := context.Background()

	// nil cache / 空 wsID → 直接 nil
	require.NoError(t, InvalidateWorkstationHealth(ctx, nil, "ws-1"))
	require.NoError(t, InvalidateWorkstationHealth(ctx, nil, ""))

	mem := pkgcache.NewMemoryCache(10, time.Minute)
	key := GetReconciliationHealthByWorkstationKey("ws-9")
	require.NoError(t, mem.Set(ctx, key, []byte("{}"), time.Minute))

	require.NoError(t, InvalidateWorkstationHealth(ctx, mem, "ws-9"))
	keys, _ := mem.Keys(ctx, "reconciliation:health:workstation:*")
	assert.Empty(t, keys, "失效后健康度缓存应被删除")
}

// =====================================================================
// reconciliation_snapshot.go
// =====================================================================

func TestSnapshotService_RefreshViewAndLastRefreshAt(t *testing.T) {
	db := newGapTestDB(t)
	svc := NewReconciliationSnapshotService(db)
	ctx := context.Background()

	// sqlite → PG 专属 REFRESH 直接跳过
	require.NoError(t, svc.RefreshView(ctx))

	// LastRefreshAt:无 key → nil,nil
	ts, err := svc.LastRefreshAt(ctx)
	require.NoError(t, err)
	assert.Nil(t, ts)

	// 非法值 → nil,nil
	require.NoError(t, db.Exec(`INSERT INTO sys_config (config_key, config_value) VALUES ('asset.reconciliation.view.last_refresh_at', 'not-a-time')`).Error)
	ts, err = svc.LastRefreshAt(ctx)
	require.NoError(t, err)
	assert.Nil(t, ts)

	// RFC3339 与常见 layout 均可解析
	require.NoError(t, db.Exec(`UPDATE sys_config SET config_value = '2026-08-01T10:00:00Z'`).Error)
	ts, err = svc.LastRefreshAt(ctx)
	require.NoError(t, err)
	require.NotNil(t, ts)
	assert.Equal(t, 2026, ts.Year())

	require.NoError(t, db.Exec(`UPDATE sys_config SET config_value = '2026-08-02 10:00:00'`).Error)
	ts, err = svc.LastRefreshAt(ctx)
	require.NoError(t, err)
	require.NotNil(t, ts)
	assert.Equal(t, 2, ts.Day())
}

func TestSnapshotService_JobStubFunctions(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, ExecuteRefreshViewTask(ctx))
	require.NoError(t, ExecuteDetectLayer3Task(ctx))
	require.NoError(t, ExecuteDetectExpiredSilenceTask(ctx))
	require.NoError(t, ExecuteCleanupExpiredExceptionsTask(ctx))

	// StartupRefreshView: nil db → error; sqlite db → RefreshView 短路 nil
	require.Error(t, StartupRefreshView(ctx, nil))
	require.NoError(t, StartupRefreshView(ctx, newGapTestDB(t)))
}

// =====================================================================
// reconciliation_service.go — SetMatcher / Refresh / GetByID / ResolveException
// =====================================================================

func TestReconciliationService_SetMatcherRefreshGetByID(t *testing.T) {
	db := newGapTestDB(t)
	ctx := context.Background()

	// SetMatcher:nil receiver 安全 + 真实注入
	var nilImpl *reconciliationServiceImpl
	nilImpl.SetMatcher(nil)

	svc := NewReconciliationService(db, nil, nil)
	impl := svc.(*reconciliationServiceImpl)
	require.Nil(t, impl.matcher)
	impl.SetMatcher(NewReconciliationExceptionService(db))
	require.NotNil(t, impl.matcher)

	// Refresh: nil db → error
	_, _, _, _, err := NewReconciliationService(nil, nil, nil).Refresh(ctx)
	require.Error(t, err)

	// Refresh(sqlite):Type B 资产 → DetectLayer3 立即检出 1 条
	uid := "00000000-0000-0000-0000-0000000000aa"
	require.NoError(t, db.Exec(`INSERT INTO ops_asset (id, devicesn, user_id, status, deleted_at) VALUES (?, 'SN-R1', NULL, 0, NULL)`, uid).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset_physical (asset_id, physical_user_id, physical_username) VALUES (?, ?, 'bob')`, uid, uid).Error)
	inserted, skipped, _, _, err := svc.Refresh(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, inserted, "sqlite Refresh 跳过 MV 但应执行 DetectLayer3")
	assert.Equal(t, 0, skipped, "无 Type A 资产")

	// GetByID:命中 / 未命中 (nil,nil)
	got, err := svc.GetByID(ctx, uid)
	_ = got
	require.NoError(t, err)
	// 检出的异常行按 asset 反查
	var recID string
	require.NoError(t, db.Raw(`SELECT id FROM sys_data_reconciliation WHERE asset_id = ?`, uid).Scan(&recID).Error)
	require.NotEmpty(t, recID)
	hit, err := svc.GetByID(ctx, recID)
	require.NoError(t, err)
	require.NotNil(t, hit)
	assert.Equal(t, "B", hit.ConflictType)
	miss, err := svc.GetByID(ctx, "ghost-id")
	require.NoError(t, err)
	assert.Nil(t, miss)
}

func TestReconciliationService_ResolveException(t *testing.T) {
	db := newGapTestDB(t)
	svc := NewReconciliationService(db, nil, nil)
	ctx := context.Background()

	// 参数守卫
	require.ErrorContains(t, svc.ResolveException(ctx, "", "u1", nil), "异常ID不能为空")
	require.ErrorContains(t, svc.ResolveException(ctx, "r-1", "", nil), "当前用户ID不能为空")
	// 不存在
	require.ErrorContains(t, svc.ResolveException(ctx, "ghost", "u1", nil), "异常不存在")

	now := time.Now()
	require.NoError(t, db.Exec(`INSERT INTO sys_data_reconciliation (id, asset_id, conflict_type, severity, raw_snapshot, detected_at) VALUES ('rec-1', 'a-1', 'B', 'high', CAST('{}' AS BLOB), ?)`, now).Error)

	// 成功(无备注)
	require.NoError(t, svc.ResolveException(ctx, "rec-1", "u1", nil))
	var resolvedAt *time.Time
	var note sql.NullString
	require.NoError(t, db.Raw(`SELECT resolved_at, resolution_note FROM sys_data_reconciliation WHERE id = 'rec-1'`).Row().Scan(&resolvedAt, &note))
	require.NotNil(t, resolvedAt)
	assert.False(t, note.Valid, "无备注时不应写 resolution_note")

	// 重复 resolve → 拦截
	require.ErrorContains(t, svc.ResolveException(ctx, "rec-1", "u1", nil), "已标记为已解决")

	// 成功(带备注)
	require.NoError(t, db.Exec(`INSERT INTO sys_data_reconciliation (id, asset_id, conflict_type, severity, raw_snapshot, detected_at) VALUES ('rec-2', 'a-1', 'C', 'high', CAST('{}' AS BLOB), ?)`, now).Error)
	withNote := "运维已人工核对并处理"
	require.NoError(t, svc.ResolveException(ctx, "rec-2", "u2", &withNote))
	require.NoError(t, db.Raw(`SELECT resolution_note FROM sys_data_reconciliation WHERE id = 'rec-2'`).Scan(&note).Error)
	assert.Equal(t, withNote, note.String)
}

// =====================================================================
// reconciliation_service.go — GetByWorkstation 聚合链
// =====================================================================

func seedWorkstationHealth(t *testing.T, db *gorm.DB) {
	t.Helper()
	now := time.Now()
	// 工位 + 2 资产(其一有 machine_ip,其二无)
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation (id, workstation_name) VALUES ('ws-1', '工位A')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset (id, devicesn, machine_ip, user_id, status, deleted_at) VALUES ('a-1', 'SN-A', '10.1.0.5', 'u1', 0, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset (id, devicesn, machine_ip, user_id, status, deleted_at) VALUES ('a-2', 'SN-B', NULL, NULL, 0, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_workstation_device (id, workstation_id, asset_id) VALUES ('wsd-1', 'ws-1', 'a-1'), ('wsd-2', 'ws-1', 'a-2')`).Error)

	// 异常:B(drift) 挂 a-1 且命中例外规则;E(noData) 挂 a-2
	require.NoError(t, db.Exec(`INSERT INTO sys_data_reconciliation (id, asset_id, conflict_type, severity, raw_snapshot, detected_at, exception_rule_id) VALUES ('rec-b', 'a-1', 'B', 'high', CAST('{}' AS BLOB), ?, 'rule-1')`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_data_reconciliation (id, asset_id, conflict_type, severity, raw_snapshot, detected_at) VALUES ('rec-e', 'a-2', 'E', 'low', CAST('{}' AS BLOB), ?)`, now).Error)

	// 信息点 → 网络设备 IP(IP 链第三级来源)
	require.NoError(t, db.Exec(`INSERT INTO ops_info_points (id, workstation_id, device_id) VALUES ('ip-1', 'ws-1', 'nd-1')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_network_device (id, ip_address) VALUES ('nd-1', '10.2.0.9')`).Error)
}

func TestReconciliationService_GetByWorkstation(t *testing.T) {
	db := newGapTestDB(t)
	seedWorkstationHealth(t, db)
	ctx := context.Background()

	// 守卫:空 ID / 工位不存在
	svc := NewReconciliationService(db, nil, nil)
	_, err := svc.GetByWorkstation(ctx, "", "7d")
	require.ErrorContains(t, err, "工位ID不能为空")
	_, err = svc.GetByWorkstation(ctx, "ghost-ws", "7d")
	require.ErrorContains(t, err, "工位不存在")

	// 无资产工位:score=100,assets 空 slice
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation (id, workstation_name) VALUES ('ws-empty', '空工位')`).Error)
	empty, err := svc.GetByWorkstation(ctx, "ws-empty", "7d")
	require.NoError(t, err)
	assert.Equal(t, 100, empty.HealthScore.Score)
	require.NotNil(t, empty.Assets)
	assert.Empty(t, empty.Assets)

	// 有资产聚合:total=2, drift=1(B), noData=1(E), exceptionHit=1
	// abnormal = 1+1+1 = 3 > total=2 → raw=-50 → clamp 0
	resp, err := svc.GetByWorkstation(ctx, "ws-1", "7d")
	require.NoError(t, err)
	assert.Equal(t, "工位A", resp.Workstation.Name)
	assert.False(t, resp.Visible, "service 层固定 Visible=false")
	assert.Equal(t, 2, resp.HealthScore.Total)
	assert.Equal(t, 1, resp.HealthScore.Drift)
	assert.Equal(t, 1, resp.HealthScore.NoData)
	assert.Equal(t, 0, resp.HealthScore.Conflict)
	assert.Equal(t, 1, resp.HealthScore.ExceptionHit)
	assert.Equal(t, 0, resp.HealthScore.Score, "异常数超总数应 clamp 到 0")

	require.Len(t, resp.Assets, 2)
	byCode := map[string]AssetHealthItem{}
	for _, it := range resp.Assets {
		byCode[it.AssetCode] = it
	}
	// IP 解析链:a-1 有 machine_ip(第一级);a-2 走信息点→设备 IP(第三级)
	assert.Equal(t, "10.1.0.5", byCode["SN-A"].IP)
	assert.Equal(t, "10.2.0.9", byCode["SN-B"].IP)
	assert.Equal(t, "B", byCode["SN-A"].ConflictType)
	require.NotNil(t, byCode["SN-A"].ExceptionRuleID)
	assert.Equal(t, "rule-1", *byCode["SN-A"].ExceptionRuleID)
}

func TestReconciliationService_GetByWorkstationCache(t *testing.T) {
	db := newGapTestDB(t)
	seedWorkstationHealth(t, db)
	ctx := context.Background()

	mem := pkgcache.NewMemoryCache(50, time.Minute)
	svc := NewReconciliationService(db, mem, nil)

	resp, err := svc.GetByWorkstation(ctx, "ws-1", "7d")
	require.NoError(t, err)
	keys, _ := mem.Keys(ctx, "reconciliation:health:workstation:*")
	require.Len(t, keys, 1, "首次查询应写健康度缓存")

	// 删掉工位行后再次查询 → 命中缓存仍返回原数据
	require.NoError(t, db.Exec(`DELETE FROM sys_workstation WHERE id = 'ws-1'`).Error)
	cached, err := svc.GetByWorkstation(ctx, "ws-1", "7d")
	require.NoError(t, err)
	assert.Equal(t, "工位A", cached.Workstation.Name, "缓存命中应短路 DB 查询")
	_ = resp
}

func TestResolveAssetIPChain(t *testing.T) {
	// 第一级:资产 IP
	assert.Equal(t, "10.0.0.1", resolveAssetIPChain("10.0.0.1", nil, nil))
	// 第二级:工位 IP
	ws := "10.0.0.2"
	assert.Equal(t, "10.0.0.2", resolveAssetIPChain("", &ws, nil))
	empty := ""
	assert.Equal(t, "10.0.0.3", resolveAssetIPChain("", &empty, []string{"10.0.0.3"}), "空工位 IP 应跳过")
	// 第三级:设备 IP 列表首个非空
	assert.Equal(t, "10.0.0.4", resolveAssetIPChain("", nil, []string{"", "10.0.0.4"}))
	// 兜底
	assert.Equal(t, "unknown", resolveAssetIPChain("", nil, nil))
	assert.Equal(t, "unknown", resolveAssetIPChain("", nil, []string{""}))
}

// =====================================================================
// fix_suggestion_service.go — GetByID / Apply / Rollback / 纯 helper
// =====================================================================

func seedFixSuggestionEnv(t *testing.T, db *gorm.DB) {
	t.Helper()
	now := time.Now()
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username) VALUES ('u-new', 'alice')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset (id, devicesn, user_id, status, deleted_at) VALUES ('a-1', 'SN-A', 'u-old', 0, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_data_reconciliation (id, asset_id, conflict_type, severity, raw_snapshot, detected_at) VALUES ('exc-1', 'a-1', 'B', 'high', CAST('{}' AS BLOB), ?)`, now).Error)

	insertSugg := func(id, status string) {
		require.NoError(t, db.Exec(`
			INSERT INTO sys_reconciliation_fix_suggestion
			(id, created_at, updated_at, version, exception_id, suggested_user_id, fix_status, conflict_type, confidence_score, reason)
			VALUES (?, ?, ?, 0, 'exc-1', 'u-new', ?, 'B', 0.95, 'test')`, id, now, now, status).Error)
	}
	insertSugg("s-1", "pending")
	insertSugg("s-2", "pending")
	insertSugg("s-app", "accepted")
}

func TestFixSuggestion_GetByID(t *testing.T) {
	db := newGapTestDB(t)
	seedFixSuggestionEnv(t, db)
	svc := NewFixSuggestionService(db, nil, nil, nil)
	ctx := context.Background()

	// 空 ID / 未命中 (nil,nil)
	_, err := svc.GetByID(ctx, "")
	require.ErrorContains(t, err, "建议ID不能为空")
	miss, err := svc.GetByID(ctx, "ghost")
	require.NoError(t, err)
	assert.Nil(t, miss)

	detail, err := svc.GetByID(ctx, "s-1")
	require.NoError(t, err)
	assert.Equal(t, "s-1", detail.Suggestion.ID)
	assert.Equal(t, "SN-A", detail.Suggestion.AssetCode, "JOIN ops_asset.devicesn")
	require.NotNil(t, detail.Suggestion.CurrentUserID)
	assert.Equal(t, "u-old", *detail.Suggestion.CurrentUserID)
	require.NotNil(t, detail.Suggestion.SuggestedUsername)
	assert.Equal(t, "alice", *detail.Suggestion.SuggestedUsername)
	// 历史:同 exception_id 的 3 条
	require.Len(t, detail.History, 3)
	// 关联异常
	require.NotNil(t, detail.Exception)
	assert.Equal(t, "exc-1", detail.Exception.ID)
}

func TestFixSuggestion_ApplyAndRollback(t *testing.T) {
	db := newGapTestDB(t)
	seedFixSuggestionEnv(t, db)
	svc := NewFixSuggestionService(db, nil, nil, nil)
	ctx := context.Background()

	// Apply 参数守卫
	require.ErrorContains(t, svc.Apply(ctx, "", "u1"), "建议ID不能为空")
	require.ErrorContains(t, svc.Apply(ctx, "s-app", ""), "当前用户ID不能为空")
	// 非 accepted 状态
	require.ErrorContains(t, svc.Apply(ctx, "s-1", "u1"), "不存在或未处于 accepted")

	// Apply happy path
	require.NoError(t, svc.Apply(ctx, "s-app", "u1"))

	var fixStatus, preFix string
	var windowUntil *time.Time
	require.NoError(t, db.Raw(`SELECT fix_status, pre_fix_user_id, rollback_window_until FROM sys_reconciliation_fix_suggestion WHERE id = 's-app'`).Row().Scan(&fixStatus, &preFix, &windowUntil))
	assert.Equal(t, "applied", fixStatus)
	assert.Equal(t, "u-old", preFix, "pre_fix_user_id 回填应用前责任人")
	require.NotNil(t, windowUntil, "7d 回滚窗口应写入")

	var assetUser string
	require.NoError(t, db.Raw(`SELECT user_id FROM ops_asset WHERE id = 'a-1'`).Scan(&assetUser).Error)
	assert.Equal(t, "u-new", assetUser, "核心修复:ops_asset.user_id 更新为建议用户")

	var resolvedAt *time.Time
	var note sql.NullString
	require.NoError(t, db.Raw(`SELECT resolved_at, resolution_note FROM sys_data_reconciliation WHERE id = 'exc-1'`).Row().Scan(&resolvedAt, &note))
	require.NotNil(t, resolvedAt, "B-3:Apply 必须同步 resolve 异常")
	assert.Equal(t, "fix_suggestion_applied", note.String)

	// Rollback 参数守卫
	require.ErrorContains(t, svc.Rollback(ctx, "", "u1", "原因够十个字了"), "建议ID不能为空")
	require.ErrorContains(t, svc.Rollback(ctx, "s-app", "", "原因够十个字了"), "当前用户ID不能为空")
	require.ErrorContains(t, svc.Rollback(ctx, "s-app", "u1", "short"), "至少 10 字符")
	// 非 applied 状态
	require.ErrorContains(t, svc.Rollback(ctx, "s-1", "u1", "原因够十个字了"), "不存在或未处于 applied")

	// Rollback happy path
	require.NoError(t, svc.Rollback(ctx, "s-app", "u1", "责任人分配错误需要回滚"))
	require.NoError(t, db.Raw(`SELECT fix_status FROM sys_reconciliation_fix_suggestion WHERE id = 's-app'`).Scan(&fixStatus).Error)
	assert.Equal(t, "rolled_back", fixStatus)
	require.NoError(t, db.Raw(`SELECT user_id FROM ops_asset WHERE id = 'a-1'`).Scan(&assetUser).Error)
	assert.Equal(t, "u-old", assetUser, "回滚应恢复 pre_fix_user_id")
	var afterRollback sql.NullTime
	require.NoError(t, db.Raw(`SELECT resolved_at FROM sys_data_reconciliation WHERE id = 'exc-1'`).Scan(&afterRollback).Error)
	assert.False(t, afterRollback.Valid, "回滚应解除 resolved_at 供重新检出")

	// 窗口守卫:窗口为 NULL / 已过期
	prepareApplied := func(id string, window interface{}) {
		now := time.Now()
		require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_fix_suggestion (id, created_at, updated_at, version, exception_id, fix_status, conflict_type, confidence_score, reason) VALUES (?, ?, ?, 0, 'exc-1', 'applied', 'B', 0.9, 't')`, id, now, now).Error)
		if window != nil {
			require.NoError(t, db.Exec(`UPDATE sys_reconciliation_fix_suggestion SET rollback_window_until = ? WHERE id = ?`, window, id).Error)
		}
	}
	prepareApplied("s-nil", nil)
	require.ErrorContains(t, svc.Rollback(ctx, "s-nil", "u1", "原因够十个字了"), "回滚窗口未设置")
	prepareApplied("s-exp", now2(-time.Hour))
	require.ErrorContains(t, svc.Rollback(ctx, "s-exp", "u1", "原因够十个字了"), "回滚窗口已过")
}

// now2 相对当前时间的偏移时间(测试内联 helper,避免与 time.Now 混淆命名)。
func now2(d time.Duration) time.Time { return time.Now().Add(d) }

func TestFixSuggestion_PureHelpers(t *testing.T) {
	db := newGapTestDB(t)
	impl := NewFixSuggestionService(db, nil, nil, nil).(*fixSuggestionServiceImpl)
	ctx := context.Background()

	// rollbackWindowValue:sqlite → Go 侧 now+7d
	now := time.Now()
	got := impl.rollbackWindowValue(now).(time.Time)
	assert.WithinDuration(t, now.Add(7*24*time.Hour), got, time.Second)

	// isUniqueViolation / contains / indexOf
	assert.False(t, isUniqueViolation(nil))
	assert.True(t, isUniqueViolation(errText("ERROR #23505 duplicate key")))
	assert.True(t, isUniqueViolation(errText("duplicate key value violates unique constraint")))
	assert.False(t, isUniqueViolation(errText("some other error")))

	assert.True(t, contains("abcdef", "abc"))
	assert.True(t, contains("abcdef", "def"))
	assert.True(t, contains("abc", "abc"))
	assert.True(t, contains("xxabcxx", "abc"))
	assert.False(t, contains("abc", "abcd"))
	assert.False(t, contains("", "a"))

	assert.Equal(t, 2, indexOf("xxabc", "abc"))
	assert.Equal(t, -1, indexOf("abc", "z"))

	// WorkstationIDForSuggestion:sqlite 测试视图无 workstation_id 列 → 错误路径
	_, err := impl.WorkstationIDForSuggestion(ctx, "s-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workstation_id")
}

// errText 包装字符串为 error(测试内联)。
func errText(s string) error { return &stubError{s} }

type stubError struct{ msg string }

func (e *stubError) Error() string { return e.msg }

// =====================================================================
// fix_suggestion_generator.go + fix_suggestion_monitor.go
// =====================================================================

// stubConfigSvc 嵌入接口补齐其余方法,仅实现 GetByKey。
type stubConfigSvc struct {
	system.ConfigService
	values map[string]string
}

func (s *stubConfigSvc) GetByKey(_ context.Context, key string) (*models.Config, error) {
	if v, ok := s.values[key]; ok {
		return &models.Config{ConfigValue: v}, nil
	}
	return nil, nil
}

func TestFixSuggestion_Generator(t *testing.T) {
	ctx := context.Background()

	// nil db → error
	nilImpl := &fixSuggestionServiceImpl{}
	_, err := nilImpl.GenerateFixSuggestions(ctx)
	require.Error(t, err)

	// sqlite → (0, nil) 方言早退(MV 为 PG-only)
	db := newGapTestDB(t)
	impl := NewFixSuggestionService(db, nil, nil, nil).(*fixSuggestionServiceImpl)
	n, err := impl.GenerateFixSuggestions(ctx)
	require.NoError(t, err)
	assert.Zero(t, n)

	// 熔断开关 / 置信度阈值读取
	disabled := &fixSuggestionServiceImpl{db: db, configService: &stubConfigSvc{values: map[string]string{
		"asset.reconciliation.fix.enabled": "0",
	}}}
	assert.False(t, disabled.isFixFeatureEnabled(ctx))
	enabled := &fixSuggestionServiceImpl{db: db, configService: &stubConfigSvc{values: map[string]string{
		"asset.reconciliation.fix.enabled": "1",
	}}}
	assert.True(t, enabled.isFixFeatureEnabled(ctx))
	noCfg := &fixSuggestionServiceImpl{db: db}
	assert.True(t, noCfg.isFixFeatureEnabled(ctx), "configService 为 nil 默认启用")

	assert.InDelta(t, 0.95, (&fixSuggestionServiceImpl{db: db, configService: &stubConfigSvc{values: map[string]string{
		"asset.reconciliation.fix.confidence_threshold": "0.95",
	}}}).getConfidenceThreshold(ctx), 1e-9)
	assert.InDelta(t, 0.9, (&fixSuggestionServiceImpl{db: db, configService: &stubConfigSvc{values: map[string]string{
		"asset.reconciliation.fix.confidence_threshold": "not-a-number",
	}}}).getConfidenceThreshold(ctx), 1e-9)
	assert.InDelta(t, 0.9, noCfg.getConfidenceThreshold(ctx), 1e-9)
}

// stubFixSvc 仅覆盖 Stats(monitor 唯一依赖的 service 方法)。
type stubFixSvc struct {
	FixSuggestionService
	stats *FixSuggestionStatsResponse
	err   error
}

func (s *stubFixSvc) Stats(_ context.Context, _ int) (*FixSuggestionStatsResponse, error) {
	return s.stats, s.err
}

func TestFixSuggestionMonitor_CheckAndNotify(t *testing.T) {
	db := newGapTestDB(t)
	ctx := context.Background()

	// nil service → nil
	mon := NewFixSuggestionMonitor(db, nil, nil, nil, nil)
	require.NoError(t, mon.CheckAndNotify(ctx))

	// Stats 出错 → 软失败 nil
	mon = NewFixSuggestionMonitor(db, nil, nil, nil, &stubFixSvc{err: assert.AnError})
	require.NoError(t, mon.CheckAndNotify(ctx))

	// 未超阈 → nil 且不写 operlog
	mon = NewFixSuggestionMonitor(db, nil, nil, nil, &stubFixSvc{stats: &FixSuggestionStatsResponse{ThresholdBreached: false}})
	require.NoError(t, mon.CheckAndNotify(ctx))
	var cnt int64
	require.NoError(t, db.Model(&models.OperLog{}).Count(&cnt).Error)
	assert.Zero(t, cnt)

	// 超阈 → 通知 + operlog 审计;1h 节流内第二次跳过
	breach := &FixSuggestionStatsResponse{
		MisFixRate: 0.5, Threshold: 0.01, Applied: 10, RolledBack: 5, ThresholdBreached: true,
	}
	mon = NewFixSuggestionMonitor(db, nil, nil, nil, &stubFixSvc{stats: breach})
	require.NoError(t, mon.CheckAndNotify(ctx))
	require.NoError(t, db.Model(&models.OperLog{}).Count(&cnt).Error)
	assert.Equal(t, int64(1), cnt, "超阈通知应写 1 条 operlog")

	require.NoError(t, mon.CheckAndNotify(ctx))
	require.NoError(t, db.Model(&models.OperLog{}).Count(&cnt).Error)
	assert.Equal(t, int64(1), cnt, "1h 节流窗口内不应重复通知")
}

// =====================================================================
// reconciliation_exception.go — MatchException / ImportFromExcel
// =====================================================================

func TestExceptionService_MatchException(t *testing.T) {
	db := newGapTestDB(t)
	svc := NewReconciliationExceptionService(db)
	ctx := context.Background()

	// 空 IP → nil,nil
	m, err := svc.MatchException(ctx, "", "", "")
	require.NoError(t, err)
	assert.Nil(t, m)

	// 命中:active 规则 CIDR 覆盖(is_active=0 为启用,actions 为 PG 数组字面量)
	require.NoError(t, db.Exec(`INSERT INTO sys_reconciliation_exception (id, name, ip_range, exception_actions, scope_type, reason, is_active, created_by, deleted_at) VALUES ('rule-1', '测试规则', '10.9.0.0/24', '{no_alert}', 'global', 'r', 0, 1, NULL)`).Error)
	m, err = svc.MatchException(ctx, "10.9.0.77", "", "B")
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, "rule-1", m.MatchedRuleID)
	assert.Contains(t, m.AppliedActions, "no_alert")

	// CIDR 外 → nil,nil
	m, err = svc.MatchException(ctx, "192.168.1.1", "", "B")
	require.NoError(t, err)
	assert.Nil(t, m)
}

// buildGapXLSX / gapXlsxFileHeader:内存 Excel → multipart.FileHeader。
func buildGapXLSX(t *testing.T, sheetName string, rows [][]string) []byte {
	t.Helper()
	f := excelize.NewFile()
	_, err := f.NewSheet(sheetName)
	require.NoError(t, err)
	require.NoError(t, f.DeleteSheet("Sheet1"))
	for r, row := range rows {
		for c, v := range row {
			cell, err := excelize.CoordinatesToCellName(c+1, r+1)
			require.NoError(t, err)
			require.NoError(t, f.SetCellValue(sheetName, cell, v))
		}
	}
	data, err := f.WriteToBuffer()
	require.NoError(t, err)
	return data.Bytes()
}

func gapXlsxFileHeader(t *testing.T, data []byte, name string) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", name)
	require.NoError(t, err)
	_, err = fw.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	form, err := multipart.NewReader(&buf, w.Boundary()).ReadForm(32 << 20)
	require.NoError(t, err)
	require.Len(t, form.File["file"], 1)
	return form.File["file"][0]
}

func TestExceptionService_ImportFromExcel(t *testing.T) {
	db := newGapTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('dept-9', '研发部', 'RD', 0)`).Error)
	svc := NewReconciliationExceptionService(db)
	ctx := context.Background()

	// nil file → error
	_, err := svc.ImportFromExcel(ctx, nil)
	require.ErrorContains(t, err, "文件不能为空")

	rows := [][]string{
		{"规则名称", "IP段(CIDR)", "冲突类型(逗号分隔B/C/D/E/F,空=全部)",
			"动作(逗号分隔no_alert/no_notice/no_workorder/skip_severity/silence)",
			"严重度覆盖(low/medium/high,可空)", "范围(global/dept/user)",
			"范围名称(部门名/用户名,global留空)", "过期时间(可空,YYYY-MM-DD)", "原因(≥10字符)"},
		{"规则X", "10.9.0.0/24", "B,C", "skip_severity", "", "dept", "研发部", "", "用于单元测试的导入原因数据"},
	}
	result, err := svc.ImportFromExcel(ctx, gapXlsxFileHeader(t, buildGapXLSX(t, "对账例外规则", rows), "rules.xlsx"))
	require.NoError(t, err)
	assert.Equal(t, 1, result.Inserted)
	assert.Equal(t, []string{"规则X"}, result.AffectedKeys)

	// 基础字段入库
	var ipRange, scopeType string
	require.NoError(t, db.Raw(`SELECT ip_range, scope_type FROM sys_reconciliation_exception WHERE name = '规则X'`).Row().Scan(&ipRange, &scopeType))
	assert.Equal(t, "10.9.0.0/24", ipRange)
	assert.Equal(t, "dept", scopeType)

	// QUIRK(疑似生产级 bug,D-12 记录不修):postProcessImportedRules 用英文
	// 字段名 row["scopeType"]/row["scopeName"]/row["conflictTypes"] 读二次 Excel,
	// 但 ReadRawRowsByName 按中文表头文本建键 → 查找恒 miss → ResolveReconScopeID
	// 收到空参提前返回 → scope_id 永不解析,且 ParseCSVToTextArray("")=nil 把
	// excel_service 已入库的原始 CSV 覆写为 NULL(基础字段 name/ip_range 等仍正确)。
	var scopeID, conflictTypes sql.NullString
	require.NoError(t, db.Raw(`SELECT scope_id, conflict_types FROM sys_reconciliation_exception WHERE name = '规则X'`).Row().Scan(&scopeID, &conflictTypes))
	assert.False(t, scopeID.Valid, "postProcess 键名错配 → scope_id 保持 NULL(quirk)")
	assert.False(t, conflictTypes.Valid, "CSV 未转换反而被空值覆写为 NULL(quirk)")

	// invalidateCache no-op(覆盖方法体)
	db.Exec(`UPDATE sys_reconciliation_exception SET is_active = 0`)
	impl := svc.(*reconciliationExceptionServiceImpl)
	impl.invalidateCache()
}

// =====================================================================
// reconciliation_workorder.go helpers + 构造器
// =====================================================================

func TestWorkorderService_HelpersAndCtors(t *testing.T) {
	db := newGapTestDB(t)
	ctx := context.Background()

	// 构造器(无 cache / 带 cache / SetCache)
	woSvc := NewReconciliationWorkorderService(db, nil, nil)
	require.NotNil(t, woSvc)
	mem := pkgcache.NewMemoryCache(10, time.Minute)
	woSvc2 := NewReconciliationWorkorderServiceWithCache(db, mem, nil, nil)
	require.NotNil(t, woSvc2)
	woSvc.SetCache(mem)

	// WorkstationIDForException:空 ID → ("", nil)
	wsID, err := woSvc.WorkstationIDForException(ctx, "")
	require.NoError(t, err)
	assert.Empty(t, wsID)

	// sqlite 测试视图无 workstation_id 列 → 错误路径
	_, err = woSvc.WorkstationIDForException(ctx, "rec-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workstation_id")

	// InvalidateWorkstationHealth:空 ID / 无 cache 注入 → nil
	require.NoError(t, woSvc2.InvalidateWorkstationHealth(ctx, ""))
	noCache := NewReconciliationWorkorderService(db, nil, nil)
	require.NoError(t, noCache.InvalidateWorkstationHealth(ctx, "ws-1"))

	// 带 cache:删除健康度键
	require.NoError(t, mem.Set(ctx, GetReconciliationHealthByWorkstationKey("ws-2"), []byte("{}"), time.Minute))
	require.NoError(t, woSvc2.InvalidateWorkstationHealth(ctx, "ws-2"))
	keys, _ := mem.Keys(ctx, "reconciliation:health:workstation:*")
	assert.Empty(t, keys)

	// severityToSLAMinutes 表驱动
	assert.Equal(t, 30, severityToSLAMinutes("critical"))
	assert.Equal(t, 240, severityToSLAMinutes("HIGH"))
	assert.Equal(t, 1440, severityToSLAMinutes("medium"))
	assert.Equal(t, 10080, severityToSLAMinutes("low"))
	assert.Equal(t, 1440, severityToSLAMinutes("unknown"))
}

func TestWorkorderService_CreateFromExceptionEarlyPaths(t *testing.T) {
	db := newGapTestDB(t)
	hub := websocket.NewNoticeHub()
	svc := NewReconciliationWorkorderService(db, hub, nil)
	ctx := context.Background()

	// 不存在/已转单 → (nil, nil) 静默跳过
	wo, err := svc.CreateWorkorderFromException(ctx, "ghost")
	require.NoError(t, err)
	assert.Nil(t, wo)

	// 记录存在(critical)→ WS 广播已执行,随后因缺 workorder category 报错
	now := time.Now()
	require.NoError(t, db.Exec(`INSERT INTO sys_data_reconciliation (id, asset_id, conflict_type, severity, raw_snapshot, detected_at) VALUES ('rec-w', 'a-1', 'B', 'critical', CAST('{"asset_code":"SN-W"}' AS BLOB), ?)`, now).Error)
	wo, err = svc.CreateWorkorderFromException(ctx, "rec-w")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查 category 失败")
	assert.Nil(t, wo)

	// 广播 helper 直调:真实 hub(缓冲通道不阻塞)+ raw_snapshot asset_code 提取
	rec := &models.SysDataReconciliation{AssetID: "a-1", ConflictType: "B", Severity: "critical", DetectedAt: now}
	rec.ID = "rec-w"
	svc.broadcastCriticalWorkorder(ctx, rec, "wo-1", "[资产对账·B类] 标题")
	// SysNotice 通道:noticeService 未注入 → 软跳过
	svc.publishCriticalSysNotice(ctx, rec, "wo-1", "标题", "SN-W")
	// nil hub 早退分支
	NewReconciliationWorkorderService(db, nil, nil).broadcastCriticalException(ctx, rec)
}

func TestDetectionAndStatisticsCtors(t *testing.T) {
	db := newGapTestDB(t)

	det := NewReconciliationDetection(db)
	require.NotNil(t, det)
	stat := NewReconciliationStatistics(db)
	require.NotNil(t, stat)

	// isReconciliationDuplicate:pgconn 23505 / 文本匹配 / nil / 无关错误
	assert.True(t, isReconciliationDuplicate(&pgconn.PgError{Code: "23505"}))
	assert.True(t, isReconciliationDuplicate(errText("UNIQUE constraint failed: x.y")))
	assert.True(t, isReconciliationDuplicate(errText("duplicate key value violates")))
	assert.True(t, isReconciliationDuplicate(errText("SQLSTATE 23505")))
	assert.False(t, isReconciliationDuplicate(nil))
	assert.False(t, isReconciliationDuplicate(errText("connection refused")))
}
