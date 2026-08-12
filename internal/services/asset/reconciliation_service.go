package asset

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lib/pq"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// ============================================================================
// R1 边界说明 — 资产对账异常查询服务(Phase 42 R1 plan 02 / 任务 1)
//
// 本文件实现 R1 范围:仅"只读观察"。具体而言:
//
//   - ListExceptions:分页查询异常列表,支持多维度过滤 + 白名单排序
//   - GetByID:按主键查询单条异常
//
// R1 **不**提供的操作(后续 plan 接管):
//   - MarkResolved (D-18):由 R2 ReconciliationResolutionService 实现
//   - Create / Update / Delete 异常记录:由 Layer 3 检测引擎写入,
//     handler 不暴露写入口,运维只能"查看 + 解决"两类行为
//   - BulkResolve:由 R2 实现
//   - 例外规则 CRUD:由 R3 ReconciliationExceptionService 实现
//   - 统计/Count 聚合:由 R2 ReconciliationStatistics 实现
//
// 数据源:
//
//   - 主表 sys_data_reconciliation(R1 异常元数据)
//   - LEFT JOIN ops_asset 取 asset_code(devicesn) / machine_ip(显示用)
//   - LEFT JOIN reconciliation_normalized(MV,phase 42-01 已创建)取
//     physical_username (R1=NULL,R2 填充) / ad_username (R1 已有 AD 链)
//   - LEFT JOIN sys_user 取 responsible_username:此处按"asset 维度"理解
//     "责任人 username",即 ops_asset.user_id 关联 sys_user.username
//
// 关联约定 (来自 phase 42-01 物化视图定义):
//   - ops_asset.id (uuid) = sys_data_reconciliation.asset_id (uuid) —— 类型一致,可直接 JOIN
//   - reconciliation_normalized.asset_id = ops_asset.id (一对一) —— 同一资产
//   - sys_user.id (uuid) = ops_asset.user_id (varchar(64) per asset model) ——
//     严格说 sys_user.id 是 uuid,ops_asset.user_id 是 varchar(64),本查询
//     仅按 uuid = uuid 比较(假设 ops_asset.user_id 已规范化为 uuid 字符串;
//     历史上脏数据见 migration_145_fix_bound_user_id_uuid,本服务假设已完成)。
// ============================================================================

// ExceptionListParams 异常列表查询参数
//
// 嵌入 base.BaseListRequest 自动获得 current / pageSize / orderByColumn / isAsc 四个字段。
// 所有过滤字段均为可选;零值/空值跳过对应过滤。
type ExceptionListParams struct {
	base.BaseListRequest

	ConflictType string     `json:"conflictType"`           // 冲突类型(A/B/C/D/E/F,精确匹配)
	Severity     string     `json:"severity"`               // 严重度(low/medium/high/critical,精确匹配)
	AssetCode    string     `json:"assetCode"`              // 资产编号(devicesn),LIKE 模糊匹配
	DetectedFrom *time.Time `json:"detectedFrom,omitempty"` // 检测时间区间起点(包含)
	DetectedTo   *time.Time `json:"detectedTo,omitempty"`   // 检测时间区间终点(包含)

	// ShowSilenced R3 / D-R3-A1-01 — 是否在列表显示 silence 记录(默认 false 隐藏)
	//
	// silence action 命中仍写表(审计链不断),但异常列表默认过滤掉,仅运维
	// 审计场景需要时显式传 true 才可见。SQL 默认 WHERE NOT ('silence' = ANY(applied_actions))。
	ShowSilenced bool `json:"showSilenced,omitempty"`
}

// ExceptionListItem 异常列表展示项(查询 DTO,非表模型)
//
// 嵌入 SysDataReconciliation 复用其全部字段;额外字段是 JOIN 出来的展示列
// (asset_code / asset_ip_display / physical_username / responsible_username / ad_username),
// 不对应 sys_data_reconciliation 真实列(故不放进 model,避免 AutoMigrate 误建列)。
//
// GORM 列名映射说明:
//   - SysDataReconciliation.ID 等基础字段由嵌入体自身的 gorm tag 处理
//   - "asset_code" / "asset_ip" / "physical_username" / "ad_username" 由 gorm
//     column tag 显式映射;SELECT 子句使用同名 AS alias 让 GORM 顺利 Scan
//   - "responsible_username" 来自 sys_user.username,无 gorm column tag,
//     需在 SELECT 子句中显式 AS responsible_username
type ExceptionListItem struct {
	models.SysDataReconciliation

	// AssetCode 资产编号(来自 ops_asset.devicesn)
	AssetCode string `gorm:"column:asset_code" json:"assetCode,omitempty"`

	// AssetIPDisplay 资产 IP 显示值(来自 ops_asset.machine_ip,COALESCE 转 text)
	// 字段名加 Display 后缀,避免与 SysDataReconciliation.AssetIP(*net.IP)同名字段冲突。
	AssetIPDisplay string `gorm:"column:asset_ip" json:"assetIpDisplay,omitempty"`

	// PhysicalUsername 物理链路反推用户(R1 物化视图固定 NULL,R2 填充)
	PhysicalUsername *string `gorm:"column:physical_username" json:"physicalUsername,omitempty"`

	// ResponsibleUsername 责任人 username(来自 sys_user.username via ops_asset.user_id)
	// 命名遵循 CLAUDE.md json convention(驼峰);gorm column 与物理列同名(responsible_username)
	ResponsibleUsername *string `gorm:"column:responsible_username" json:"responsibleUsername,omitempty"`

	// AdUsername AD 域账号(来自 MV reconciliation_normalized.ad_username)
	AdUsername *string `gorm:"column:ad_username" json:"adUsername,omitempty"`
}

// reconAllowedSortFields 异常列表排序白名单
//
// key 为前端传入的 orderByColumn,value 为安全的物理列名(列名白名单是 SQL 注入防线:
// 即使恶意构造 orderByColumn,也只能命中本 map 内的值,无法拼接任意 SQL)。
//
// Phase 42-01 物化视图已建,Phase 42-02 R1 只读查询,排序字段仅限主表
// sys_data_reconciliation 的列;MV 列(physical_username / ad_username)暂不
// 开放给前端排序(R2 视情况补充)。
var reconAllowedSortFields = map[string]string{
	"detectedAt":      "sys_data_reconciliation.detected_at",
	"conflictType":    "sys_data_reconciliation.conflict_type",
	"severity":        "sys_data_reconciliation.severity",
	"confidenceScore": "sys_data_reconciliation.confidence_score",
}

// ReconciliationService 资产对账异常查询服务接口
//
// R1 范围:ListExceptions(分页过滤 + 排序) + GetByID(单条详情)。
// 写操作 / 标记已解决 / 例外规则 CRUD 见 R2/R3 的独立 service。
//
// R2 扩展(Phase 43 / D-A4-04):
//   - ResolveException:标记异常为已解决,SET resolved_at/resolved_by/resolution_note 三字段;
//     用于前端异常列表"标记已解决"按钮。
type ReconciliationService interface {
	// ListExceptions 列出对账异常(分页 + 过滤 + 白名单排序 + 多表 JOIN)
	ListExceptions(ctx context.Context, params *ExceptionListParams) (*base.PageResult, error)

	// GetByID 按主键查询单条异常(不带 JOIN,纯主表元数据)
	GetByID(ctx context.Context, id string) (*models.SysDataReconciliation, error)

	// ResolveException 标记异常为已解决(Phase 43 / D-A4-04)
	//
	// 行为:
	//   1. 按 id 查主表(deleted_at IS NULL 兜底)
	//   2. 防御:若 resolved_at 已存在 → 返回 error "该异常已标记为已解决",不允许重复 resolve
	//   3. SET resolved_at = NOW(), resolved_by = userID; resolution_note = note(可选)
	//   4. 不联动 workorder 关闭(workorder 独立在 workorder UI 关闭,D-A4-04 锁定)
	//
	// 入参:
	//   - ctx: 请求上下文
	//   - id: 异常主键(uuid)
	//   - userID: 当前操作用户 id(从 gin context 的 user_id 字段获取)
	//   - note: resolution_note,可为 nil(留空)或非空字符串
	//
	// 返回:
	//   - 成功 → nil
	//   - 失败 → error,handler 层映射为 500 / 400 / 404
	ResolveException(ctx context.Context, id string, userID string, note *string) error

	// Refresh 手动刷新物化视图并立即触发异常检测(Phase 45 R5 / D-R5-A3-01)
	//
	// 行为:
	//   1. REFRESH MATERIALIZED VIEW reconciliation_normalized(非 CONCURRENTLY,确保同步生效)
	//   2. 立即调一次 DetectLayer3,绕过 5min/6min cron 等待
	//   3. 返回 inserted/skipped/skippedSilence/skippedThrottle 计数
	//
	// 用于运维/UAT 调试,避免等待 R1 cron 周期。
	Refresh(ctx context.Context) (inserted, skipped, skippedSilence, skippedThrottle int, err error)

	// GetByWorkstation 拉取工位对账健康度聚合 (Phase 45 R4 / D-A4-01/02)
	//
	// 行为:
	//   1. SELECT ops_asset WHERE workstation_id=? AND deleted_at IS NULL → 资产 ID 列表
	//   2. SELECT sys_data_reconciliation WHERE asset_id IN (...) AND deleted_at IS NULL →
	//      按 conflict_type 分桶 (normal/drift/conflict/nodata) 用 COUNT(*) FILTER
	//      (无 list.length,严格遵守 stat-cards-from-list-length-capped-at-100)
	//   3. SELECT COUNT(*) WHERE exception_rule_id IS NOT NULL AND asset_id IN (...) → exceptionHit
	//   4. 复用 reconciliation_statistics.go:HealthTrend() 取最近 7 天 trend 点
	//   5. assets[] 用 LEFT JOIN reconciliation_normalized (asset_id, ip, conflict_type, severity,
	//      exception_rule_id, applied_actions, confidence_score) 一次拉取
	//   6. IP 解析链 inline:asset.ip → workstation.ip → "unknown" (D-A4-02)
	//   7. score = clamp(round((1 - 异常资产数/总资产数) × 100), 0, 100) (D-A2-03)
	//   8. 缓存 (CacheProvider.GetOrSet) TTL=5min 与 R1 MV 刷新一致 (D-A4-03)
	//   9. **不实现 Visible 字段**:service 单一职责,visible 由 handler 注入 (B3 安全 invariant)
	//
	// 入参:
	//   - ctx: 请求上下文
	//   - wsID: 工位 uuid
	//   - window: 时间窗口字符串(目前仅识别 "7d",其它值降级为 7d)
	//
	// 返回:
	//   - 成功 → *ByWorkstationResponse (含 Workstation/HealthScore/Assets;Visible 留 false)
	//   - 失败 → error,handler 层映射为 500
	GetByWorkstation(ctx context.Context, wsID string, window string) (*ByWorkstationResponse, error)
}

// reconciliationServiceImpl ReconciliationService 私有实现
type reconciliationServiceImpl struct {
	db *gorm.DB

	// cache R4 启用:GetByWorkstation 通过 CacheProvider.GetOrSet 做 5min TTL 缓存
	// 与 R1 MV 刷新节流对齐 (D-A4-03)。可为 nil(单元测试场景)→ 直查 DB。
	cache cache.Cache

	// matcher 例外规则匹配器(Phase 45 R4 / D-A4-02 注入)
	//
	// 用于 GetByWorkstation 内的 IP CIDR 命中测试:
	//   - assets[].IP 解析后调 matcher.MatchException 计算 exceptionHit
	//   - 为 nil 时跳过 MatchException 调用(单测场景,exceptionHit 仍由 DB COUNT
	//     提供基本准确度,但 per-asset actions 字段为空)
	matcher ReconciliationExceptionService

	// mvExists 缓存 reconciliation_normalized 物化视图是否存在(避免每次查询重复探测)。
	// 三态:1=MV 就位 / 0=MV 缺失 / -1=未探测。
	// 关键:必须显式初始化为 -1(默认零值 0 会被误判为"已探测且 MV 缺失")。
	mvExists    int32 // 0=false, 1=true, -1=未探测;atomic 操作
	mvExistsMux sync.Mutex
}

// NewReconciliationService 构造 ReconciliationService 实例
//
// mvExists 显式初始化为 -1(未探测),避免默认零值 0 导致首次查询跳过 probe。
//
// R4 (Phase 45) 改造:
//   - 第二个参数 c (cache.Cache) 注入以支持 GetByWorkstation 的 5min TTL 缓存
//   - 第三个参数 matcher 例外规则匹配器(Phase 45 R4 / D-A4-02),nil 时跳过 per-asset 匹配
//   - 现有调用方 (router.go, reconciliation_router.go) 必须同步更新
//   - 传 nil 时 GetByWorkstation 降级为直查 DB(单元测试友好)
func NewReconciliationService(db *gorm.DB, c cache.Cache, matcher ReconciliationExceptionService) ReconciliationService {
	return &reconciliationServiceImpl{db: db, cache: c, matcher: matcher, mvExists: -1}
}

// SetMatcher 注入例外规则匹配器(用于延迟注入 — 避免初始化循环依赖)
//
// 使用场景:router.go 构造 ReconciliationService 时 matcher 尚未就绪,可在
// reconciliation_router.go 内调 SetMatcher 注入。注:本方法为 nil-safe。
func (s *reconciliationServiceImpl) SetMatcher(matcher ReconciliationExceptionService) {
	if s != nil {
		s.matcher = matcher
	}
}

// Refresh 手动刷新物化视图并立即触发异常检测(Phase 45 R5 / D-R5-A3-01)
//
// 行为:
//   1. REFRESH MATERIALIZED VIEW reconciliation_normalized(非 CONCURRENTLY,确保同步生效)
//   2. 立即调一次 DetectLayer3,绕过 5min/6min cron 等待
//   3. 返回 inserted/skipped/skippedSilence/skippedThrottle 计数
//
// 用于运维/UAT 调试,避免等待 R1 cron 周期。
//
// SQLite 测试路径返回 (0, 0, 0, 0, nil),跳过 PostgreSQL-only 操作。
func (s *reconciliationServiceImpl) Refresh(ctx context.Context) (int, int, int, int, error) {
	if s.db == nil {
		return 0, 0, 0, 0, errors.New("db 未初始化")
	}

	// 1. REFRESH MATERIALIZED VIEW CONCURRENTLY(保持 PG 9.4+ 无锁刷新语义,与 R1 一致)
	if s.db.Dialector.Name() == "postgres" {
		// 用 CONCURRENTLY 避免锁表;但首次创建后未填充时 CONCURRENTLY 会失败,
		// 因此按是否存在行做兜底:首次或空时用非 CONCURRENTLY。
		var rowCount int64
		if err := s.db.WithContext(ctx).
			Raw("SELECT COUNT(*) FROM reconciliation_normalized").
			Scan(&rowCount).Error; err != nil {
			return 0, 0, 0, 0, fmt.Errorf("探测 reconciliation_normalized 行数失败: %w", err)
		}

		if rowCount == 0 {
			if err := s.db.WithContext(ctx).
				Exec("REFRESH MATERIALIZED VIEW reconciliation_normalized").
				Error; err != nil {
				return 0, 0, 0, 0, fmt.Errorf("REFRESH MATERIALIZED VIEW reconciliation_normalized 失败: %w", err)
			}
		} else {
			if err := s.db.WithContext(ctx).
				Exec("REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized").
				Error; err != nil {
				// CONCURRENTLY 失败时降级为非并发刷新(锁表瞬时)
				if err2 := s.db.WithContext(ctx).
					Exec("REFRESH MATERIALIZED VIEW reconciliation_normalized").
					Error; err2 != nil {
					return 0, 0, 0, 0, fmt.Errorf("REFRESH MATERIALIZED VIEW 失败(concurrent+fallback): %w / %v", err, err2)
				}
			}
		}
	}

	// 2. 立即跑 DetectLayer3 — 跳过 cron 6min 周期
	det := NewReconciliationDetection(s.db)
	inserted, skipped, skippedSilence, skippedThrottle, err := det.DetectLayer3(ctx)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("DetectLayer3 失败: %w", err)
	}

	return inserted, skipped, skippedSilence, skippedThrottle, nil
}

// probeMaterializedView 检查 reconciliation_normalized 物化视图是否存在。
//
// 仅 PostgreSQL 需要探测(SQLite 测试用 view 模拟,MySQL 由 migration 168 直接跳过);
// 使用 PG `to_regclass('schema.name')` 内置函数 — 返回 NULL 表示不存在,返回 oid 表示存在。
// 安全:SELECT to_regclass 不写表,且非破坏性。
func (s *reconciliationServiceImpl) probeMaterializedView() bool {
	if s.db == nil {
		return false
	}
	if s.db.Dialector.Name() != "postgres" {
		// SQLite 测试用 view 模拟,unit test 路径下 MV 等价物总存在(setupTestDB 建 view)
		return true
	}
	var exists sql.NullString
	if err := s.db.Raw("SELECT to_regclass('public.reconciliation_normalized')::text").Scan(&exists).Error; err != nil {
		applogger.Warnf("[reconciliation] MV probe SQL error: %v", err)
		return false
	}
	applogger.Infof("[reconciliation] MV probe result: valid=%v string=%q", exists.Valid, exists.String)
	return exists.Valid && exists.String != ""
}

// mvAvailable 返回并缓存 MV 可用性。
// 三态 mvExists 字段:-1 未探测 / 0 不可用 / 1 可用。
// sync.Mutex + atomic store 保证并发安全。
func (s *reconciliationServiceImpl) mvAvailable() bool {
	if atomic.LoadInt32(&s.mvExists) >= 0 {
		return atomic.LoadInt32(&s.mvExists) == 1
	}
	s.mvExistsMux.Lock()
	defer s.mvExistsMux.Unlock()
	if atomic.LoadInt32(&s.mvExists) >= 0 {
		return atomic.LoadInt32(&s.mvExists) == 1
	}
	avail := s.probeMaterializedView()
	if avail {
		atomic.StoreInt32(&s.mvExists, 1)
	} else {
		atomic.StoreInt32(&s.mvExists, 0)
	}
	return avail
}

// exceptionListJoinSelect ListExceptions 的 SELECT 子句
//
// 关键设计:
//   - "sys_data_reconciliation.*" 显式限定表名前缀,避免与 JOIN 表的列名冲突
//   - a.devicesn AS asset_code —— 把 devicesn 别名为业务字段 asset_code,
//     配合 ExceptionListItem.AssetCode 的 gorm column tag 让 GORM 自动 Scan
//   - COALESCE(a.machine_ip::text, '') AS asset_ip —— machine_ip 是 inet 类型,
//     PG 强类型 inet 不能直接走 json Marshal,转 text 安全(SQLite 端 inet 即 TEXT,
//     COALESCE(text, '') 在两库行为一致)
//   - rn.physical_username / rn.ad_username —— 来自 reconciliation_normalized 物化视图
//   - ru.username AS responsible_username —— sys_user.username 别名为 responsible_username
//     匹配 ExceptionListItem.ResponsibleUsername 的 column tag
const exceptionListJoinSelect = `sys_data_reconciliation.*,
	a.devicesn AS asset_code,
	COALESCE(a.machine_ip::text, '') AS asset_ip,
	rn.physical_username,
	ru.username AS responsible_username,
	rn.ad_username`

// exceptionListJoinClause ListExceptions 的 JOIN 子句
//
// 关联关系:
//   - sys_data_reconciliation.asset_id (uuid) → ops_asset.id (uuid) —— 类型一致,直接 =
//   - ops_asset.id (uuid) → reconciliation_normalized.asset_id (uuid) ——
//     MV 与 ops_asset 一对一(见 migration_168 定义:一个 asset 一行 MV)
//   - ops_asset.user_id (varchar(64) in asset model) → sys_user.id (uuid) ——
//     类型不一致,必须 ::uuid 显式转换(PG 强类型不会自动 cast,SQLSTATE 42883)
//
// 软删除条件:各表都加 deleted_at IS NULL 兜底,避免已删除的资产/用户污染对账展示。
const exceptionListJoinClause = `
LEFT JOIN ops_asset a ON a.id = sys_data_reconciliation.asset_id AND a.deleted_at IS NULL
LEFT JOIN reconciliation_normalized rn ON rn.asset_id = a.id
LEFT JOIN sys_user ru ON ru.id = a.user_id::uuid AND ru.deleted_at IS NULL`

// exceptionListJoinSelectFallback MV 缺失时的降级 SELECT 子句
//
// 不引用 rn.physical_username / rn.ad_username,以 NULL 字面量占位,让 GORM Scan 时
// ExceptionListItem.PhysicalUsername / AdUsername 字段为空字符串(omitempty tag 会隐藏)。
// 其余字段(资产维度 + 责任人维度)由 ops_asset / sys_user JOIN 提供,无依赖 MV。
const exceptionListJoinSelectFallback = `sys_data_reconciliation.*,
	a.devicesn AS asset_code,
	COALESCE(a.machine_ip::text, '') AS asset_ip,
	''::text AS physical_username,
	ru.username AS responsible_username,
	''::text AS ad_username`

// exceptionListJoinClauseFallback MV 缺失时的降级 JOIN 子句
//
// 不 JOIN reconciliation_normalized,避免 SQLSTATE 42P01。
// 仍保留 ops_asset / sys_user JOIN 以提供 asset_code / asset_ip / responsible_username。
// a.user_id 是 varchar(64) 而 sys_user.id 是 uuid,必须 ::uuid 显式转换(同 MV 内 JOIN)。
const exceptionListJoinClauseFallback = `
LEFT JOIN ops_asset a ON a.id = sys_data_reconciliation.asset_id AND a.deleted_at IS NULL
LEFT JOIN sys_user ru ON ru.id = a.user_id::uuid AND ru.deleted_at IS NULL`

// ListExceptions 列出对账异常
//
// 行为:
//  1. 校验 params 非 nil
//  2. 默认 current=1, pageSize=10
//  3. 基础查询:Model(SysDataReconciliation{}) + deleted_at IS NULL
//  4. 过滤:conflictType / severity (精确) + assetCode (LIKE) + detectedFrom/To (区间)
//  5. Count + Find,Find 携带 JOIN + SELECT 子句
//  6. 排序:base.ApplySort 白名单;无显式排序则默认 detected_at DESC
//  7. 分页:Offset + Limit
//  8. 拼装 base.PageResult 返回
func (s *reconciliationServiceImpl) ListExceptions(ctx context.Context, params *ExceptionListParams) (*base.PageResult, error) {
	if params == nil {
		return nil, errors.New("查询参数不能为空")
	}

	// 默认分页
	current := params.Current
	if current <= 0 {
		current = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	// 基础查询
	// 关键修复:assetCode LIKE 过滤在 WHERE 子句里直接引用 `a.devicesn`,
	// 因此基础 query 必须先 Joins(ops_asset AS a),否则 Count 阶段 SQL 不含
	// `a` 别名会报 SQLSTATE 42P01 "missing FROM-clause entry for table \"a\""。
	// mvAvailable() 探测保证 MV 存在,所以这里只 Joins(ops_asset);MV 缺失时
	// fallback 路径使用全新 session(见下),再 Joins(fallback_clause),GORM
	// Joins() 链式累加特性由 fallback 路径独立 session 规避。
	query := s.db.WithContext(ctx).
		Model(&models.SysDataReconciliation{}).
		Where("sys_data_reconciliation.deleted_at IS NULL").
		Joins("LEFT JOIN ops_asset a ON a.id = sys_data_reconciliation.asset_id AND a.deleted_at IS NULL")

	// 过滤:精确匹配
	if params.ConflictType != "" {
		query = query.Where("sys_data_reconciliation.conflict_type = ?", params.ConflictType)
	}
	if params.Severity != "" {
		query = query.Where("sys_data_reconciliation.severity = ?", params.Severity)
	}
	// 过滤:资产编号 LIKE 模糊匹配
	// 注意:LIKE 操作符未走白名单过滤,值通过 ? 占位符 + 参数化,无 SQL 注入风险
	if params.AssetCode != "" {
		query = query.Where("a.devicesn LIKE ?", "%"+params.AssetCode+"%")
	}
	// 过滤:检测时间区间
	if params.DetectedFrom != nil {
		query = query.Where("sys_data_reconciliation.detected_at >= ?", *params.DetectedFrom)
	}
	if params.DetectedTo != nil {
		query = query.Where("sys_data_reconciliation.detected_at <= ?", *params.DetectedTo)
	}

	// R3 / D-R3-A1-01 — silence 默认过滤(异常列表不显示 silence 记录)
	//
	// 用全限定列名 sys_data_reconciliation.applied_actions 避免 JOIN 时歧义;
	// PG 原生 ANY() 数组操作符 + 'silence' = ANY(array) 在 PG 与 SQLite(text 列)
	// 行为一致(SQLite 把 applied_actions 当 text,ANY 退化为子串匹配 — 实际测试
	// 通过,因为 silence 值在数组里包含 'silence' 字符串)。
	if !params.ShowSilenced {
		query = query.Where("NOT ('silence' = ANY(sys_data_reconciliation.applied_actions))")
	}

	// Count(COUNT 不需要 JOIN 出来的列;基础查询 + 过滤已足够)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Find 携带 SELECT + JOIN
	var list []ExceptionListItem
	// MV 降级路径:R1 部署场景下 reconciliation_normalized 可能尚未创建(migration_168 未跑或被 drop)。
	// 若 MV 缺失,LEFT JOIN 会触发 SQLSTATE 42P01 "relation does not exist",前端 400。
	// 探测结果通过 s.mvAvailable() 缓存,后续查询跳过探测;MV 缺失时 SELECT/JOIN 子句都不含 rn.*。
	//
	// GORM Joins() 链式调用会 **累积追加** 到同一查询对象(query 已含 dev_assets LIKE filter 等),
	// 因此 fallback 路径必须使用全新 DB session(s.db.WithContext(ctx))避免 JOIN 重复堆叠。
	mvAvailable := s.mvAvailable()
	var findQuery *gorm.DB
	if mvAvailable {
		// 基础 query 已 Joins(ops_asset) 保证 a.devicesn WHERE 引用可解析,
		// 这里只追加 MV JOIN + SELECT。注意:不能再次 .Joins("LEFT JOIN ops_asset ..."),
		// GORM Joins() 链式累加会导致重复 JOIN(不影响结果但污染 SQL)。
		findQuery = query.
			Select(exceptionListJoinSelect).
			Joins("LEFT JOIN reconciliation_normalized rn ON rn.asset_id = a.id").
			Joins("LEFT JOIN sys_user ru ON ru.id = a.user_id::uuid AND ru.deleted_at IS NULL")
	} else {
		applogger.Warnf("[reconciliation] reconciliation_normalized 物化视图缺失,异常列表查询降级 (无 physical_username / ad_username 字段)。请执行 migration_168 创建 MV 后重启服务")
		// 全新 session,只含 filter WHERE,不累加前次 Joins()
		filterQuery := s.db.WithContext(ctx).
			Model(&models.SysDataReconciliation{}).
			Where("sys_data_reconciliation.deleted_at IS NULL")
		if params.ConflictType != "" {
			filterQuery = filterQuery.Where("sys_data_reconciliation.conflict_type = ?", params.ConflictType)
		}
		if params.Severity != "" {
			filterQuery = filterQuery.Where("sys_data_reconciliation.severity = ?", params.Severity)
		}
		if params.AssetCode != "" {
			filterQuery = filterQuery.Where("a.devicesn LIKE ?", "%"+params.AssetCode+"%")
		}
		if params.DetectedFrom != nil {
			filterQuery = filterQuery.Where("sys_data_reconciliation.detected_at >= ?", *params.DetectedFrom)
		}
		if params.DetectedTo != nil {
			filterQuery = filterQuery.Where("sys_data_reconciliation.detected_at <= ?", *params.DetectedTo)
		}
		// R3 / D-R3-A1-01 — silence 默认过滤(fallback 路径同步)
		if !params.ShowSilenced {
			filterQuery = filterQuery.Where("NOT ('silence' = ANY(sys_data_reconciliation.applied_actions))")
		}
		findQuery = filterQuery.
			Select(exceptionListJoinSelectFallback).
			Joins(exceptionListJoinClauseFallback)
	}

	// 排序:白名单过滤;无显式排序则默认 detected_at DESC(操作员最关心"最新告警")
	findQuery = base.ApplySort(findQuery, params.BaseListRequest, reconAllowedSortFields)
	if findQuery.Statement.SQL.String() != "" {
		// base.ApplySort 在 OrderByColumn 为空时不会追加 Order,这里手动判断
		// 简化做法:始终先调用 ApplySort,然后无条件追加默认排序作为 fallback
		// 但 GORM 多个 Order() 调用后者会覆盖前者,所以仅在未指定排序时追加
		if params.OrderByColumn == "" {
			findQuery = findQuery.Order("sys_data_reconciliation.detected_at DESC")
		}
	} else {
		findQuery = findQuery.Order("sys_data_reconciliation.detected_at DESC")
	}

	offset := (current - 1) * pageSize
	if err := findQuery.Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}

	return &base.PageResult{
		List:     list,
		Total:    total,
		Current:  current,
		PageSize: pageSize,
	}, nil
}

// GetByID 按主键查询单条异常
//
// 行为:
//   - 软删除兜底:deleted_at IS NULL
//   - 记录不存在返回 nil(nil + nil 是该方法的正常"未找到"语义,
//     区别于"数据库出错"返回的 nil + err)
func (s *reconciliationServiceImpl) GetByID(ctx context.Context, id string) (*models.SysDataReconciliation, error) {
	var recon models.SysDataReconciliation
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&recon).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &recon, nil
}

// ResolveException 标记异常为已解决(Phase 43 R2 / D-A4-04)
//
// 行为(详见接口注释):
//   step 1: SELECT ... WHERE id=? AND deleted_at IS NULL → First
//   step 2: 防御检查 — 若 resolved_at != nil → 返回 error(不允许重复 resolve)
//   step 3: 构造 updates map(resolved_at=NOW, resolved_by=userID, resolution_note=note)
//   step 4: db.Model(&rec).Updates(updates) — GORM 只 SET 包含的字段,避免覆盖其他字段
//
// 并发安全:
//   - 防御 1:layer 2 SELECT resolved_at 检查 → 普通并发可拦截
//   - 防御 2:GORM Update 本身在 PG 上是单行 UPDATE,原子操作
//   - 防御 3:即使并发两个 resolve 请求同时进入 step 4,两次 UPDATE 都设置相同的
//     resolved_at=NOW() + resolved_by=userID,但第二次 UPDATE 不会失败 — 业务上
//     可接受(同状态被不同用户重写)。若需严格互斥,可在 handler 层加分布式锁;
//     当前 R2 范围不做。
//
// 入参:
//   - id: 异常主键(uuid)
//   - userID: 当前操作用户(从 gin context 的 user_id 字段获取)
//   - note: resolution_note,可为 nil(留空)或非空字符串(运维填的备注)
//
// 返回:
//   - nil: 成功
//   - errors.New("该异常已标记为已解决"): 重复 resolve 防御
//   - 其他 err: DB 层错误(handler 层映射 500)
func (s *reconciliationServiceImpl) ResolveException(ctx context.Context, id string, userID string, note *string) error {
	if id == "" {
		return errors.New("异常ID不能为空")
	}
	if userID == "" {
		return errors.New("当前用户ID不能为空")
	}

	// step 1: 查询异常
	var rec models.SysDataReconciliation
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("异常不存在")
		}
		return fmt.Errorf("查询异常失败: %w", err)
	}

	// step 2: 已 resolved 不允许重复 resolve(D-A4-04 锁定)
	if rec.ResolvedAt != nil {
		return errors.New("该异常已标记为已解决")
	}

	// step 3: 构造 updates map(GORM Updates 只 SET 包含字段,避免覆盖其他字段)
	now := time.Now()
	updates := map[string]interface{}{
		"resolved_at": now,
		"resolved_by": userID,
	}
	if note != nil && *note != "" {
		updates["resolution_note"] = *note
	}

	// step 4: UPDATE
	if err := s.db.WithContext(ctx).Model(&rec).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新异常已解决状态失败: %w", err)
	}
	applogger.Infof("[reconciliation] 异常已标记为已解决 id=%s userID=%s hasNote=%v", id, userID, note != nil && *note != "")
	return nil
}

// ============================================================================
// R4 (Phase 45) — 工位对账健康度聚合 GetByWorkstation
//
// D-A4-01/02/03 锁定:
//   - POST /asset/reconciliation/by-workstation 入参 {workstationId, window}
//   - 出参 ByWorkstationResponse{Workstation, HealthScore, Assets, Visible}
//   - TTL=5min 缓存 (D-A4-03 与 R1 MV 刷新一致)
//   - Visible 字段由 handler 注入,service 不负责 (B3 安全 invariant)
//
// score 公式 (D-A2-03): clamp(round((1 - 异常资产数/总资产数) × 100), 0, 100)
// ============================================================================

// WorkstationBrief 工位基础信息(显示用,不携带敏感字段)
type WorkstationBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// HealthScore 5 KPI + 趋势数据 (D-A2-01/03 锁定)
//
// 字段命名 camelCase 与前端 useQuery 解构对齐;
// 所有数字字段(总/正常/漂移/冲突/无数据/例外命中)均由 COUNT(*) 聚合得出,
// 严禁 list.length 路径(MEMORY `stat-cards-from-list-length-capped-at-100`)。
type HealthScore struct {
	Total        int          `json:"total"`
	Normal       int          `json:"normal"`
	Drift        int          `json:"drift"`
	Conflict     int          `json:"conflict"`
	NoData       int          `json:"noData"`
	ExceptionHit int          `json:"exceptionHit"`
	Score        int          `json:"score"`
	Trend        []TrendPoint `json:"trend"`
}

// AssetHealthItem 资产健康度条目(行内徽标数据源,UI-SPEC D-A4-02)
//
// IP 解析链 inline 实现 (D-A4-02 + strategy §4.4):
//   asset.ip → workstation.ip → "unknown"
type AssetHealthItem struct {
	AssetID         string   `json:"assetId"`
	AssetCode       string   `json:"assetCode"`
	ConflictType    string   `json:"conflictType"`
	Severity        string   `json:"severity"`
	IP              string   `json:"ip,omitempty"`
	ExceptionRuleID *string  `json:"exceptionRuleId,omitempty"`
	AppliedActions  []string `json:"appliedActions,omitempty"`
	ConfidenceScore float64  `json:"confidenceScore"`
}

// ByWorkstationResponse 工位对账健康度聚合响应 (D-A4-02 锁定)
type ByWorkstationResponse struct {
	Workstation WorkstationBrief  `json:"workstation"`
	HealthScore HealthScore       `json:"healthScore"`
	Assets      []AssetHealthItem `json:"assets"`
	Visible     bool              `json:"visible"` // 由 handler 注入,service 固定为 false
}

// reconciliationHealthCacheTTL GetByWorkstation 缓存 TTL (5min,与 R1 MV 刷新一致)
const reconciliationHealthCacheTTL = 5 * time.Minute

// GetByWorkstation 工位对账健康度聚合 (R4)
//
// 行为(详见接口注释):
//  1. 缓存命中 → 反序列化返回
//  2. 缓存未命中 → 执行 6 步查询(CPU/IO 密集,缓存防穿透)
//     a) 工位基础信息(SELECT FROM sys_workstation WHERE id=)
//     b) 关联资产(SELECT FROM ops_asset WHERE workstation_id=? AND deleted_at IS NULL)
//     c) 对账异常按 conflict_type 分桶(COUNT FILTER,严格走 DB 聚合)
//     d) exceptionHit 计数(COUNT WHERE exception_rule_id IS NOT NULL)
//     e) Trend 7 天(reconciliation_statistics.go:HealthTrend 同形态,本函数内联)
//     f) assets[] 行内徽标(LEFT JOIN reconciliation_normalized 一次拉取)
//
// 关键约束:
//   - 严禁 list.length 走分桶统计 (MEMORY `stat-cards-from-list-length-capped-at-100`)
//   - IP 解析链 inline:asset.ip → workstation.ip → "unknown" (B5 修复)
//   - assets[] 即使 total=0 也要返回空 slice 而非 nil(JSON 序列化一致性)
//   - Visible 字段固定为 false(handler 注入,与 service 单一职责保持一致)
func (s *reconciliationServiceImpl) GetByWorkstation(ctx context.Context, wsID string, window string) (*ByWorkstationResponse, error) {
	if wsID == "" {
		return nil, errors.New("工位ID不能为空")
	}

	// 缓存命中短路
	cacheKey := GetReconciliationHealthByWorkstationKey(wsID)
	if s.cache != nil {
		var cached ByWorkstationResponse
		if err := s.cache.GetJSON(ctx, cacheKey, &cached); err == nil && cached.Workstation.ID != "" {
			cached.Visible = false // service 单一职责,handler 重新注入
			return &cached, nil
		}
	}

	resp, err := s.computeByWorkstation(ctx, wsID)
	if err != nil {
		return nil, err
	}

	// 写缓存
	if s.cache != nil {
		if data, mErr := json.Marshal(resp); mErr == nil {
			if setErr := s.cache.Set(ctx, cacheKey, data, reconciliationHealthCacheTTL); setErr != nil {
				applogger.Warnf("[reconciliation] GetByWorkstation cache set failed wsID=%s: %v", wsID, setErr)
			}
		}
	}

	resp.Visible = false
	return resp, nil
}

// computeByWorkstation 实际执行 6 步聚合查询(GetByWorkstation 内部使用)
func (s *reconciliationServiceImpl) computeByWorkstation(ctx context.Context, wsID string) (*ByWorkstationResponse, error) {
	resp := &ByWorkstationResponse{
		Assets: []AssetHealthItem{}, // JSON 序列化一致性
	}

	// a) 工位基础信息
	//
	// 注意:sys_workstation 表无 workstation_no/code 列(legacy-2026-06-15/034 删除了 workstation_code)
	// — 只 select 真实存在的列,Code 字段在 Scan 后保留零值 "" (前端的 ByWorkstationResponse.workstation.code 类型兼容)
	var brief WorkstationBrief
	if err := s.db.WithContext(ctx).
		Table("sys_workstation").
		Select("id, workstation_name AS name").
		Where("id = ? AND deleted_at IS NULL", wsID).
		Scan(&brief).Error; err != nil {
		return nil, fmt.Errorf("查询工位基础信息失败: %w", err)
	}
	if brief.ID == "" {
		return nil, errors.New("工位不存在")
	}
	resp.Workstation = brief

	// b) 关联资产 ID 列表
	type assetRow struct {
		ID         string
		Devicesn   string
		IP         *string
	}
	// 关联资产通过 ops_workstation_device 中间表(已有 workstation_id + asset_id 外键,
	// 两列均为 VARCHAR(36) — 存 UUID 字符串,与 ops_asset.id (uuid) 需显式 cast 兼容 PG 强类型)。
	// 不能直接 WHERE ops_asset.workstation_id = ? — 该列在 ops_asset 不存在
	// (R1 已知约束:物理链路 sys_port_mac→sys_info_point→sys_workstation_info_point 未落地,
	//  R4 用 ops_workstation_device 作工位-资产关联的可靠路径)。
	// 沿用 workstation_service List 中 primary_device_serial 子查询的 cast 模式 (::text)。
	var assets []assetRow
	if err := s.db.WithContext(ctx).
		Table("ops_asset a").
		Select("a.id, a.devicesn, a.machine_ip::text AS ip").
		Joins("INNER JOIN ops_workstation_device wsd ON wsd.asset_id = a.id::text AND wsd.deleted_at IS NULL").
	Where("wsd.workstation_id = ? AND a.deleted_at IS NULL", wsID).
		Scan(&assets).Error; err != nil {
		return nil, fmt.Errorf("查询工位关联资产失败: %w", err)
	}
	resp.HealthScore.Total = len(assets)

	// WR-02: 单一流(原 early-return 拆成 trend + score 两段)。
	//   - Trend 始终先计算(廉价查询,无副作用)
	//   - Score 仅在 Total==0 时短路为 100
	//   - Total>0 时 Score 留给后续 bucket / exceptionHit 累加后由 (1 - abnormal/total)*100 公式算出
	// 两条路径收敛:无资产早退 / 有资产走完 bucket+exceptionHit+score。
	trend, _ := s.computeHealthTrend(ctx, 7)
	resp.HealthScore.Trend = trend
	if resp.HealthScore.Total == 0 {
		resp.HealthScore.Score = 100 // 无资产视为健康
		return resp, nil
	}

	assetIDs := make([]string, 0, len(assets))
	assetCodeByID := make(map[string]string, len(assets))
	assetIPByID := make(map[string]string, len(assets))
	for _, a := range assets {
		assetIDs = append(assetIDs, a.ID)
		assetCodeByID[a.ID] = a.Devicesn
		if a.IP != nil {
			assetIPByID[a.ID] = *a.IP
		}
	}

	// c) 按 conflict_type 分桶计数(严格 COUNT FILTER,无 list.length)
	type bucketRow struct {
		ConflictType string
		Cnt          int64
	}
	var buckets []bucketRow
	if err := s.db.WithContext(ctx).
		Table("sys_data_reconciliation").
		Select("conflict_type, COUNT(*) AS cnt").
		Where("asset_id IN ? AND deleted_at IS NULL AND resolved_at IS NULL", assetIDs).
		Group("conflict_type").
		Scan(&buckets).Error; err != nil {
		return nil, fmt.Errorf("查询对账异常分桶失败: %w", err)
	}

	// 初始化 6 桶(A 视为 normal,D-A2-04 + UI-SPEC 锁定:健康用 green,A/B-F 用 dict list_class)
	// 桶:normal(A)/drift(B,D)/conflict(C,F)/nodata(E)
	normal, drift, conflict, noData := 0, 0, 0, 0
	for _, b := range buckets {
		switch b.ConflictType {
		case "A":
			normal += int(b.Cnt)
		case "B", "D":
			drift += int(b.Cnt)
		case "C", "F":
			conflict += int(b.Cnt)
		case "E":
			noData += int(b.Cnt)
		default:
			// 未知类型归 conflict(防御性)
			conflict += int(b.Cnt)
		}
	}
	resp.HealthScore.Normal = normal
	resp.HealthScore.Drift = drift
	resp.HealthScore.Conflict = conflict
	resp.HealthScore.NoData = noData

	// d) exceptionHit 计数
	var exceptionHit int64
	if err := s.db.WithContext(ctx).
		Table("sys_data_reconciliation").
		Where("asset_id IN ? AND deleted_at IS NULL AND exception_rule_id IS NOT NULL", assetIDs).
		Count(&exceptionHit).Error; err != nil {
		return nil, fmt.Errorf("查询例外命中数失败: %w", err)
	}
	resp.HealthScore.ExceptionHit = int(exceptionHit)

	// f) score 公式 (D-A2-03): clamp(round((1 - 异常资产数/总资产数) × 100), 0, 100)
	// 异常资产数 = drift + conflict + nodata(以桶数计,而非 unique 资产数,简化为 5 维比)
	abnormalAssets := drift + conflict + noData + int(exceptionHit)
	if resp.HealthScore.Total > 0 {
		raw := (1 - float64(abnormalAssets)/float64(resp.HealthScore.Total)) * 100
		if raw < 0 {
			raw = 0
		}
		if raw > 100 {
			raw = 100
		}
		resp.HealthScore.Score = int(raw + 0.5)
	} else {
		resp.HealthScore.Score = 100
	}

	// g) assets[] 行内徽标数据(LEFT JOIN sys_data_reconciliation)
	//
	// 注意:reconciliation_normalized 物化视图(由 migration_168 创建)只包含 asset + AD 字段
	// (asset_id, asset_username, ad_username 等),冲突/严重程度/置信度等列在 sys_data_reconciliation 表。
	// 原代码 LEFT JOIN MV 取这些列会 SQLSTATE 42703 → graceful fallback 丢徽标数据。
	// 修复:改为 LEFT JOIN sys_data_reconciliation (resolved_at IS NULL 表示 open 冲突)。
	// healthRow 本地 Scan 结构。
	//
	// R5 修复 (2026-06-30):必须给 AppliedActions 加 gorm:"column:applied_actions" tag。
	//
	// 否则 GORM schema.Parse (gorm.io/gorm@v1.30.5/schema/schema.go:294-305) 会把
	// pq.StringArray (driver.Valuer) 误判为 relationship 字段:
	//   1. pq.StringArray.Value() 对零值返回 (nil, nil) → DataType 保持 ""
	//   2. DataType 为空 + Creatable/Updatable/Readable=true → 被分类为 relationship
	//   3. parseRelation → getOrParse → 解引用 *[] → [] → string,string 的
	//      PkgPath()=="" → 返回 "unsupported data type: &[]"
	//   4. 整行 Scan 失败 → rows=nil → LEFT JOIN 徽标数据全丢(200 仍返回,但 item 退化)
	//
	// 已有正解参考:internal/services/asset/reconciliation_detection_test.go:46-50, 72-76
	// 同样 SELECT applied_actions FROM ... 的测试用 gorm:"column:applied_actions" tag
	// 工作正常,正是缺这一行的差别。
	type healthRow struct {
		AssetID         string
		ConflictType    *string
		Severity        *string
		ExceptionRuleID *string
		AppliedActions  pq.StringArray `gorm:"column:applied_actions"`
		ConfidenceScore *float64
	}
	var rows []healthRow
	if err := s.db.WithContext(ctx).
		Table("ops_asset AS a").
		Select(`a.id AS asset_id,
			r.conflict_type,
			r.severity,
			r.exception_rule_id,
			r.applied_actions,
			r.confidence_score`).
		Joins(`LEFT JOIN sys_data_reconciliation r ON r.asset_id = a.id AND r.deleted_at IS NULL AND r.resolved_at IS NULL`).
		Where("a.id IN ? AND a.deleted_at IS NULL", assetIDs).
		Scan(&rows).Error; err != nil {
		// MV 缺失 → fallback 走空 (B5:不阻塞响应,徽标列无数据)
		applogger.Warnf("[reconciliation] GetByWorkstation MV JOIN 失败,徽标数据降级为空 wsID=%s: %v", wsID, err)
		rows = nil
	}

	// 工位 IP(R5 修复)
	// R4 代码尝试 SELECT machine_ip::text FROM sys_workstation,但 sys_workstation
	// 表没有 machine_ip 列(internal/models/workstation.go — Workstation 只有 UserID/UserName,
	// 没有 IP 列)。R4 用 `_ = ...` 静默吞错但留错误日志;R5 调用增多后噪声放大且会让
	// 潜在运维误以为是 R5 bug。改用 ops_info_points -> sys_network_device.ip_address
	// 的链作为工位 IP 来源(由 fetchWorkstationDeviceIPs 提供),不再依赖 sys_workstation
	// 不存在的列。下游 workstationIP 保持 nil,resolveAssetIPChain 会跳到第三级 IP 链。
	var workstationIP *string
	_ = workstationIP

	// 一次性预拉所有工位关联信息点的 device_ip(降低 N+1 风险)
	// 即使全部为空,这些工位信息点可作为 IP 解析链第三级的来源
	workstationDeviceIPs := s.fetchWorkstationDeviceIPs(ctx, wsID)

	assetItems := make([]AssetHealthItem, 0, len(assets))
	for _, a := range assets {
		item := AssetHealthItem{
			AssetID:   a.ID,
			AssetCode: a.Devicesn,
		}
		// 从 row 查
		for _, r := range rows {
			if r.AssetID == a.ID {
				if r.ConflictType != nil {
					item.ConflictType = *r.ConflictType
				}
				if r.Severity != nil {
					item.Severity = *r.Severity
				}
				item.ExceptionRuleID = r.ExceptionRuleID
				if len(r.AppliedActions) > 0 {
					item.AppliedActions = []string(r.AppliedActions)
				}
				if r.ConfidenceScore != nil {
					item.ConfidenceScore = *r.ConfidenceScore
				}
				break
			}
		}
		// IP 解析链(Phase 45 R4 / D-A4-02 inline):
		//   第一优先:ops_asset.machine_ip → 第二优先:sys_workstation.machine_ip
		//   第三优先:通过 ops_info_points 关联 sys_network_device.ip_address → "unknown"
		item.IP = resolveAssetIPChain(assetIPByID[a.ID], workstationIP, workstationDeviceIPs)
		assetItems = append(assetItems, item)

		// 🆕 Phase 45 R4 / D-A4-02:per-asset 例外规则命中(MatchException inline)
		//
		// 若 IP 解析成功且 matcher 已注入(单测场景可为 nil),调 MatchException 计算
		// applied_actions 合并结果。命中时:
		//   - item.ExceptionRuleID 已由 MV 填充(基于 reconciliation_normalized,
		//     数据可能在 R3 软停用后短暂不一致),MatchException 重新计算为最新 active 规则
		//   - item.AppliedActions 用 merge 后的并集覆盖 MV 值
		// 未命中时:保持 MV 填充值(不污染数据)
		// 失败仅 logrus.Warnf(不阻塞响应,异常统计兜底由 DB COUNT 提供)
		if s.matcher != nil && item.IP != "" && item.IP != "unknown" && item.ConflictType != "" {
			if match, mErr := s.matcher.MatchException(ctx, item.IP, "", item.ConflictType); mErr != nil {
				applogger.Warnf("[reconciliation] GetByWorkstation per-asset MatchException failed assetID=%s: %v", a.ID, mErr)
			} else if match != nil {
				item.ExceptionRuleID = &match.MatchedRuleID
				if len(match.AppliedActions) > 0 {
					item.AppliedActions = []string(match.AppliedActions)
				}
			}
		}
	}
	resp.Assets = assetItems

	return resp, nil
}

// fetchWorkstationDeviceIPs 一次性预拉工位关联信息点的设备 IP(避免 N+1)
//
// 业务说明(Phase 45 R4 / D-A4-02):
//   - IP 解析链第三级通过 ops_info_points JOIN sys_network_device.ip_address 实现
//   - 一个工位可能关联多条信息点,本函数 SELECT DISTINCT 去重后返回
//   - 失败返回 nil(IP 解析链第三级降级为 "unknown",不影响主路径)
//
// 表名校正(以 codebase 实际 schema 为准):
//   - 信息点:ops_info_points(非 sys_workstation_info_point,该表尚未在 R1 引入)
//   - 设备:sys_network_device
//   - 设备 IP 列:ip_address(非 ip)
//   - sys_network_device.id 是 UUID,ops_info_points.device_id 是 varchar(64),
//     JOIN 需显式 ::text 转换(R5 修复;之前 R4 0A000 类型不匹配)
func (s *reconciliationServiceImpl) fetchWorkstationDeviceIPs(ctx context.Context, wsID string) []string {
	if wsID == "" {
		return nil
	}
	var ips []string
	if err := s.db.WithContext(ctx).
		Table("ops_info_points").
		Distinct("sys_network_device.ip_address").
		Joins("JOIN sys_network_device ON sys_network_device.id::text = ops_info_points.device_id").
		Where("ops_info_points.workstation_id = ? AND ops_info_points.deleted_at IS NULL "+
			"AND sys_network_device.deleted_at IS NULL "+
			"AND sys_network_device.ip_address IS NOT NULL AND sys_network_device.ip_address != ''", wsID).
		Pluck("sys_network_device.ip_address", &ips).Error; err != nil {
		applogger.Warnf("[reconciliation] GetByWorkstation 拉取工位关联设备 IP 失败 wsID=%s: %v", wsID, err)
		return nil
	}
	return ips
}

// resolveAssetIPChain IP 解析链(Phase 45 R4 / D-A4-02)
//
// 优先级(三步降级):
//  1. ops_asset.machine_ip(assetIP) — 资产最直接的 IP
//  2. sys_workstation.machine_ip(workstationIP) — 工位 IP,适合 IP 集中分配场景
//  3. sys_network_device.ip_address(通过 ops_info_points JOIN 拉取的设备 IP 列表)
//     — 工位物理链路反查(最贴近用户实际接入设备)
//  4. "unknown" — 全部为空
//
// 入参均为已解析好的字符串(本函数不发起 DB 查询,纯计算路径)。
func resolveAssetIPChain(assetIP string, workstationIP *string, deviceIPs []string) string {
	if assetIP != "" {
		return assetIP
	}
	if workstationIP != nil && *workstationIP != "" {
		return *workstationIP
	}
	for _, ip := range deviceIPs {
		if ip != "" {
			return ip
		}
	}
	return "unknown"
}

// computeHealthTrend 内联 7 天趋势(dialect-aware,与 HealthTrend() 形态一致)
//
// 注:复用 reconciliation_statistics.go:HealthTrend() 会引入跨文件循环依赖风险;
// 这里内联精简版,行为对齐 (date / openCount / criticalCount / newCount)。
func (s *reconciliationServiceImpl) computeHealthTrend(ctx context.Context, days int) ([]TrendPoint, error) {
	if days <= 0 {
		days = 7
	}
	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	dialect := s.db.Dialector.Name()

	var sql string
	var args []interface{}
	switch dialect {
	case "postgres":
		sql = `
			SELECT
				TO_CHAR(date_trunc('day', detected_at), 'YYYY-MM-DD') AS date,
				COUNT(*) FILTER (WHERE resolved_at IS NULL) AS open_count,
				COUNT(*) FILTER (WHERE severity = ? AND resolved_at IS NULL) AS critical_count,
				COUNT(*) AS new_count
			FROM sys_data_reconciliation
			WHERE deleted_at IS NULL AND detected_at >= ?
			GROUP BY date_trunc('day', detected_at)
			ORDER BY date_trunc('day', detected_at) ASC
		`
		args = []interface{}{"critical", since}
	default:
		sql = `
			SELECT
				strftime('%Y-%m-%d', detected_at) AS date,
				SUM(CASE WHEN resolved_at IS NULL THEN 1 ELSE 0 END) AS open_count,
				SUM(CASE WHEN severity = ? AND resolved_at IS NULL THEN 1 ELSE 0 END) AS critical_count,
				COUNT(*) AS new_count
			FROM sys_data_reconciliation
			WHERE deleted_at IS NULL AND detected_at >= ?
			GROUP BY strftime('%Y-%m-%d', detected_at)
			ORDER BY strftime('%Y-%m-%d', detected_at) ASC
		`
		args = []interface{}{"critical", since}
	}

	var points []TrendPoint = make([]TrendPoint, 0)
	if err := s.db.WithContext(ctx).Raw(sql, args...).Scan(&points).Error; err != nil {
		applogger.Warnf("[reconciliation] computeHealthTrend failed: %v", err)
		return points, err
	}
	return points, nil
}
