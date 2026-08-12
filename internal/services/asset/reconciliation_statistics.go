package asset

import (
	"context"
	"errors"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// ============================================================================
// R1 边界说明 — 资产对账统计服务(Phase 42 R1 plan 04)
//
// 本文件实现 D-06 锁定的 5 KPI + 3 图表 + 2 辅助指标所需的 6 个 Statistics 端点,
// 全部走真 SQL 聚合查询(SELECT COUNT / GROUP BY / SUM(CASE WHEN ...)),
// **严禁**走 list.length 路径(项目记忆 `stat-cards-from-list-length-capped-at-100`)。
//
// 6 个端点对应 ROADMAP success criteria 7 的 dashboard 数据源:
//
//  1. Summary            — 5 KPI 卡片(全量资产数/未解决异常数/critical 数/7d 新增/Top1 冲突类型)
//  2. ByConflictType     — 按冲突类型 A-F 聚合(柱状图数据源)
//  3. BySeverity         — 按严重级 low/medium/high/critical 聚合(柱状图数据源)
//  4. HealthTrend        — 按天聚合的 7d 健康趋势(折线图数据源,PG `date_trunc` + `FILTER`)
//  5. TopUnresolved      — Top N 长期未解决异常(列表数据源)
//  6. ExceptionRuleStats — 按 IP 段例外规则聚合的命中数(R3 接入后生效)
//
// 数据源:sys_data_reconciliation (主表) + ops_asset (资产维度聚合)。
// 不消费 reconciliation_normalized MV(本表已包含必要字段,R1 物理链路未填)。
//
// SQLite unit test 不覆盖 HealthTrend(SQLite 不支持 `FILTER (WHERE ...)`),
// 由 PG dev DB 集成测试覆盖(per CONTEXT.md D-13)。其余 5 端点 SQLite 单测完整。
// ============================================================================

// SummaryResult 5 KPI 卡片返回值(D-06)
//
// 字段命名 camelCase,与前端 useQuery 解构约定对齐;
// `TotalAssets / OpenExceptions / CriticalOpen / Last7dNew` 四个数字直接走 COUNT(*) 聚合,
// `TopConflictType / TopConflictCount` 走 GROUP BY + ORDER BY + LIMIT 1。
type SummaryResult struct {
	TotalAssets      int64  `json:"totalAssets"`      // SELECT COUNT(*) FROM ops_asset WHERE deleted_at IS NULL
	OpenExceptions   int64  `json:"openExceptions"`   // SELECT COUNT(*) FROM sys_data_reconciliation WHERE resolved_at IS NULL AND deleted_at IS NULL
	CriticalOpen     int64  `json:"criticalOpen"`     // 同上 + severity = 'critical'
	Last7dNew        int64  `json:"last7dNew"`        // detected_at >= NOW() - INTERVAL '7 days'
	TopConflictType  string `json:"topConflictType"`  // GROUP BY conflict_type ORDER BY cnt DESC LIMIT 1 (空表示无数据)
	TopConflictCount int64  `json:"topConflictCount"` // Top1 计数
}

// TrendPoint 按天聚合的健康度趋势点
//
// 字段命名与前端 ECharts line chart x/y 轴 dataIndex 对齐;
// `Date` 用 YYYY-MM-DD 字符串(避免前端 timezone 转换歧义);
// `OpenCount` = 当日 resolved_at IS NULL 累计;
//
//	`CriticalCount` = 当日 severity='critical' AND resolved_at IS NULL;
//	`NewCount` = 当日 detected_at 当天的所有记录(含已解决)。
type TrendPoint struct {
	Date          string `json:"date"`
	OpenCount     int64  `json:"openCount"`
	CriticalCount int64  `json:"criticalCount"`
	NewCount      int64  `json:"newCount"`
}

// ExceptionSummary 长期未解决异常摘要(TopUnresolved 端点)
//
// 字段命名与前端 TopN 列表对齐;
// `DaysUnresolved` 由数据库 NOW() - detected_at 实时计算(EXTRACT DAY / JULIANDAY 兼容)。
type ExceptionSummary struct {
	ID             string    `json:"id"`
	AssetCode      string    `json:"assetCode"`      // JOIN ops_asset.devicesn
	ConflictType   string    `json:"conflictType"`
	Severity       string    `json:"severity"`
	DetectedAt     time.Time `json:"detectedAt"`
	DaysUnresolved int       `json:"daysUnresolved"`
}

// RuleStats 例外规则命中统计(ExceptionRuleStats 端点)
//
// R3 接入例外规则 CRUD 后由 ops 前端展示命中分布;
// R1 exception_rule_id 全部为 NULL,返回空 slice 即可。
type RuleStats struct {
	RuleID      string `json:"ruleId"`
	RuleName    string `json:"ruleName"`
	MatchedCount int64 `json:"matchedCount"`
}

// StatsFilter Statistics 端点通用过滤参数(R1 仅 days,future R3+ 扩展)
//
// 字段全部 optional,零值采用端点内默认值;
// 与前端 useQuery 参数透传对齐。
type StatsFilter struct {
	Days int `json:"days"` // HealthTrend / Summary.Last7dNew 时间窗口,默认 7,上限 365
}

// ReconciliationStatistics 资产对账统计服务接口
//
// 实现约束:
//   - 全部方法走 GORM aggregate query(SELECT COUNT / GROUP BY / SUM)
//   - 严禁 Find/Offset/list.length(MEMORY `stat-cards-from-list-length-capped-at-100`)
//   - HealthTrend 使用 PG `date_trunc("day", detected_at)` + `COUNT(*) FILTER (WHERE ...)`
//     SQLite 不支持 FILTER 子句,该端点 SQLite 单测 SKIP
type ReconciliationStatistics interface {
	// Summary 5 KPI 卡片数据源
	Summary(ctx context.Context, filter StatsFilter) (*SummaryResult, error)

	// ByConflictType 按冲突类型聚合(默认覆盖 A-F,无数据键值 0)
	ByConflictType(ctx context.Context, filter StatsFilter) (map[string]int64, error)

	// BySeverity 按严重级聚合(默认覆盖 low/medium/high/critical,无数据键值 0)
	BySeverity(ctx context.Context, filter StatsFilter) (map[string]int64, error)

	// HealthTrend 按天聚合的健康度趋势(PG `date_trunc` + `FILTER`,SQLite 跳过)
	HealthTrend(ctx context.Context, filter StatsFilter) ([]TrendPoint, error)

	// TopUnresolved 长期未解决异常 Top N(默认 limit=10,MaxPageSize=10000 兜底)
	TopUnresolved(ctx context.Context, limit int) ([]ExceptionSummary, error)

	// ExceptionRuleStats 按例外规则聚合的命中数(R3 接入后有数据)
	ExceptionRuleStats(ctx context.Context) ([]RuleStats, error)
}

// reconciliationStatisticsImpl Statistics 服务私有实现
type reconciliationStatisticsImpl struct {
	db *gorm.DB
}

// NewReconciliationStatistics 构造 ReconciliationStatistics 实例
//
// 用法:
//
//	stats := asset.NewReconciliationStatistics(core.DB.GetDB())
//	result, err := stats.Summary(ctx, asset.StatsFilter{Days: 7})
func NewReconciliationStatistics(db *gorm.DB) ReconciliationStatistics {
	return &reconciliationStatisticsImpl{db: db}
}

// defaultStatsDays 默认统计时间窗口(7 天)
const defaultStatsDays = 7

// maxStatsDays 最大统计时间窗口(365 天,1 年,T-42-14 mitigates HealthTrend DoS)
const maxStatsDays = 365

// conflictTypesSeed A-F 6 个冲突类型种子(D-09 锁定的所有分类)
// 保证 ByConflictType 返回 map 始终覆盖这 6 个 key(无数据键值 0)
var conflictTypesSeed = []string{"A", "B", "C", "D", "E", "F"}

// severitySeed 4 个严重级种子
// 保证 BySeverity 返回 map 始终覆盖这 4 个 key(无数据键值 0)
var severitySeed = []string{"low", "medium", "high", "critical"}

// clampStatsDays 把 days 钳制在 [1, maxStatsDays] 范围内;0 走 defaultStatsDays。
func clampStatsDays(days int) int {
	if days <= 0 {
		return defaultStatsDays
	}
	if days > maxStatsDays {
		return maxStatsDays
	}
	return days
}

// Summary 5 KPI 聚合(D-06):
//
//   - TotalAssets:COUNT(ops_asset WHERE deleted_at IS NULL)
//   - OpenExceptions:COUNT(sys_data_reconciliation WHERE resolved_at IS NULL AND deleted_at IS NULL)
//   - CriticalOpen:同上 + severity='critical'
//   - Last7dNew:同上 + detected_at >= NOW() - 7d
//   - TopConflictType / TopConflictCount:GROUP BY conflict_type ORDER BY cnt DESC LIMIT 1
//
// 5 个独立查询(避免单条 SQL CASE WHEN 与 GROUP BY 互不兼容)。
// 严禁 list.length(MEMORY `stat-cards-from-list-length-capped-at-100`)。
func (s *reconciliationStatisticsImpl) Summary(ctx context.Context, filter StatsFilter) (*SummaryResult, error) {
	if s.db == nil {
		return nil, errors.New("db 不能为空")
	}
	days := clampStatsDays(filter.Days)
	sevenDaysAgo := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

	result := &SummaryResult{}

	// 1) TotalAssets
	if err := s.db.WithContext(ctx).
		Model(&models.Asset{}).
		Where("deleted_at IS NULL").
		Count(&result.TotalAssets).Error; err != nil {
		return nil, err
	}

	// 2) OpenExceptions
	if err := s.db.WithContext(ctx).
		Model(&models.SysDataReconciliation{}).
		Where("deleted_at IS NULL AND resolved_at IS NULL").
		Count(&result.OpenExceptions).Error; err != nil {
		return nil, err
	}

	// 3) CriticalOpen
	if err := s.db.WithContext(ctx).
		Model(&models.SysDataReconciliation{}).
		Where("deleted_at IS NULL AND resolved_at IS NULL AND severity = ?", "critical").
		Count(&result.CriticalOpen).Error; err != nil {
		return nil, err
	}

	// 4) Last7dNew(沿用 filter.Days,默认 7)
	if err := s.db.WithContext(ctx).
		Model(&models.SysDataReconciliation{}).
		Where("deleted_at IS NULL AND detected_at >= ?", sevenDaysAgo).
		Count(&result.Last7dNew).Error; err != nil {
		return nil, err
	}

	// 5) TopConflictType + TopConflictCount
	// GORM 链式 GROUP BY + ORDER BY + LIMIT 1,Row().Scan(&topType, &topCount)
	type topRow struct {
		ConflictType string
		Cnt          int64
	}
	var top topRow
	row := s.db.WithContext(ctx).
		Model(&models.SysDataReconciliation{}).
		Select("conflict_type, COUNT(*) AS cnt").
		Where("deleted_at IS NULL").
		Group("conflict_type").
		Order("cnt DESC, conflict_type ASC").
		Limit(1).
		Row()
	if row != nil {
		if err := row.Scan(&top.ConflictType, &top.Cnt); err == nil {
			result.TopConflictType = top.ConflictType
			result.TopConflictCount = top.Cnt
		}
		// sql.ErrNoRows 不报错:无数据时 TopConflictType="", TopConflictCount=0
	}

	return result, nil
}

// ByConflictType 按冲突类型聚合(A-F 6 值,无数据键值 0)
//
// SQL:SELECT conflict_type, COUNT(*) FROM sys_data_reconciliation
//
//	WHERE deleted_at IS NULL GROUP BY conflict_type
//
// 然后 merge 到 conflictTypesSeed 保证 6 个 key 都在(无数据 0)。
func (s *reconciliationStatisticsImpl) ByConflictType(ctx context.Context, filter StatsFilter) (map[string]int64, error) {
	if s.db == nil {
		return nil, errors.New("db 不能为空")
	}
	// 初始化 seed (覆盖 A-F 6 key)
	result := make(map[string]int64, len(conflictTypesSeed))
	for _, t := range conflictTypesSeed {
		result[t] = 0
	}

	type row struct {
		ConflictType string
		Cnt          int64
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Model(&models.SysDataReconciliation{}).
		Select("conflict_type, COUNT(*) AS cnt").
		Where("deleted_at IS NULL").
		Group("conflict_type").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, r := range rows {
		// 只接受 A-F 范围,脏数据(未来扩展类型时)不污染 map
		if _, ok := result[r.ConflictType]; ok {
			result[r.ConflictType] = r.Cnt
		}
	}

	return result, nil
}

// BySeverity 按严重级聚合(low/medium/high/critical 4 值)
//
// SQL:SELECT severity, COUNT(*) FROM sys_data_reconciliation
//
//	WHERE deleted_at IS NULL GROUP BY severity
//
// 然后 merge 到 severitySeed 保证 4 个 key 都在(无数据 0)。
func (s *reconciliationStatisticsImpl) BySeverity(ctx context.Context, filter StatsFilter) (map[string]int64, error) {
	if s.db == nil {
		return nil, errors.New("db 不能为空")
	}
	result := make(map[string]int64, len(severitySeed))
	for _, s := range severitySeed {
		result[s] = 0
	}

	type row struct {
		Severity string
		Cnt      int64
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Model(&models.SysDataReconciliation{}).
		Select("severity, COUNT(*) AS cnt").
		Where("deleted_at IS NULL").
		Group("severity").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, r := range rows {
		if _, ok := result[r.Severity]; ok {
			result[r.Severity] = r.Cnt
		}
	}

	return result, nil
}

// HealthTrend 按天聚合的健康度趋势(PG `date_trunc` + `FILTER`,SQLite 不支持)
//
// SQL(POSTGRES):
//
//	SELECT
//	    TO_CHAR(date_trunc('day', detected_at), 'YYYY-MM-DD') AS date,
//	    COUNT(*) FILTER (WHERE resolved_at IS NULL) AS open_count,
//	    COUNT(*) FILTER (WHERE severity = 'critical' AND resolved_at IS NULL) AS critical_count,
//	    COUNT(*) AS new_count
//	FROM sys_data_reconciliation
//	WHERE deleted_at IS NULL AND detected_at >= ?
//	GROUP BY date
//	ORDER BY date ASC
//
// days 默认 7,MaxPageSize=365 钳制(T-42-14 mitigates DoS)。
//
// 注意:此端点使用 PostgreSQL 特定语法(`FILTER (WHERE ...)` / `date_trunc`),
// SQLite 不支持 FILTER 子句,SQLite 单测 SKIP(per CONTEXT.md D-13)。
func (s *reconciliationStatisticsImpl) HealthTrend(ctx context.Context, filter StatsFilter) ([]TrendPoint, error) {
	if s.db == nil {
		return nil, errors.New("db 不能为空")
	}
	days := clampStatsDays(filter.Days)
	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

	var points []TrendPoint = make([]TrendPoint, 0)

	// GORM 自动判断 dialect:
	//   - PostgreSQL:走 date_trunc + FILTER (WHERE ...) 标准 PG 语法
	//   - SQLite:用 strftime 替代 date_trunc + 用 CASE WHEN 替代 FILTER
	//
	// 这里手动判断 dialect,因为 FILTER 子句 GORM chainable API 不直接支持,
	// 必须走 db.Raw(...).Scan(&points),SQL 由 GetDialect() 决定。
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
		// SQLite / 其他 dialect:用 strftime + CASE WHEN 兼容
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

	if err := s.db.WithContext(ctx).Raw(sql, args...).Scan(&points).Error; err != nil {
		return nil, err
	}

	return points, nil
}

// TopUnresolved 长期未解决异常 Top N
//
// SQL (dialect-aware):
//   - PostgreSQL: EXTRACT(DAY FROM (NOW() - r.detected_at)) — 标准 PG 语法
//   - SQLite:     julianday('now') - julianday(r.detected_at)     — SQLite 内置
//
// 背景:`julianday()` 是 SQLite 函数,PostgreSQL 不支持(SQLSTATE 42883)。
// 用 GORM dialector.Name() 在运行时分支,避免 PG 生产环境 400。
//
// limit 默认 10,MaxPageSize=10000 钳制(T-42-12 mitigates DoS)。
func (s *reconciliationStatisticsImpl) TopUnresolved(ctx context.Context, limit int) ([]ExceptionSummary, error) {
	if s.db == nil {
		return nil, errors.New("db 不能为空")
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}

	// D-13 dialect 分支:PG 用 EXTRACT,SQLite 用 julianday
	daysExpr := "CAST((julianday('now') - julianday(r.detected_at)) AS INTEGER)"
	if s.db.Dialector.Name() == "postgres" {
		daysExpr = "CAST(EXTRACT(DAY FROM (NOW() - r.detected_at)) AS INTEGER)"
	}

	sql := `
		SELECT
			r.id AS id,
			a.devicesn AS asset_code,
			r.conflict_type,
			r.severity,
			r.detected_at,
			` + daysExpr + ` AS days_unresolved
		FROM sys_data_reconciliation r
		LEFT JOIN ops_asset a ON a.id = r.asset_id
		WHERE r.deleted_at IS NULL AND r.resolved_at IS NULL
		ORDER BY r.detected_at ASC
		LIMIT ?
	`

	var result []ExceptionSummary = make([]ExceptionSummary, 0)
	if err := s.db.WithContext(ctx).Raw(sql, limit).Scan(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

// ExceptionRuleStats 按例外规则聚合的命中数(R3 接入后有数据)
//
// SQL:
//
//	SELECT r.exception_rule_id, e.name, COUNT(*) AS matched_count
//	FROM sys_data_reconciliation r
//	LEFT JOIN sys_reconciliation_exception e ON e.id = r.exception_rule_id
//	WHERE r.deleted_at IS NULL AND r.exception_rule_id IS NOT NULL
//	GROUP BY r.exception_rule_id, e.name
//	ORDER BY matched_count DESC
//
// R1 exception_rule_id 全为 NULL,返回空 slice 即可。
func (s *reconciliationStatisticsImpl) ExceptionRuleStats(ctx context.Context) ([]RuleStats, error) {
	if s.db == nil {
		return nil, errors.New("db 不能为空")
	}

	sql := `
		SELECT
			r.exception_rule_id AS rule_id,
			e.name AS rule_name,
			COUNT(*) AS matched_count
		FROM sys_data_reconciliation r
		LEFT JOIN sys_reconciliation_exception e ON e.id = r.exception_rule_id
		WHERE r.deleted_at IS NULL AND r.exception_rule_id IS NOT NULL
		GROUP BY r.exception_rule_id, e.name
		ORDER BY matched_count DESC
	`

	var result []RuleStats = make([]RuleStats, 0)
	if err := s.db.WithContext(ctx).Raw(sql).Scan(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

// MaxPageSize 与 operations/pagination_helper.go:MaxPageSize 同值;
// 这里独立 const 避免 import cycle(operations → asset 不允许反向依赖)。
const MaxPageSize = 10000