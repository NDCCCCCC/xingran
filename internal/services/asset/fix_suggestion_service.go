package asset

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	"gorm.io/gorm"
)

// ============================================================================
// Phase 46 R5 — 资产对账半自动修复建议 service
//
// 范围:6 状态状态机(pending/accepted/rejected/applied/rolled_back/failed)
//       + 7d 回滚窗口 + 完整 operlog 审计链
//
// 锁定的 18 项决策(详见 .planning/phases/46-r5/46-CONTEXT.md):
//   - D-A1: 修复字段仅 ops_asset.user_id
//   - D-A3: 置信度门槛可配置 sys_config
//   - D-A4: 触发条件 confidence>=threshold AND conflict_type='B' AND workorder_id IS NULL
//   - D-B2: 6 状态状态机
//   - D-C1: 回滚粒度仅恢复 user_id
//   - D-C2: 7d 回滚窗口(DB-side INTERVAL 避免时钟漂移)
//   - D-C5: 误修复率监控
//
// 关键 B-3 修复:Apply 步骤必须 UPDATE sys_data_reconciliation.resolved_at = NOW()
//                否则下次 DetectLayer3 cron 重新检出该 Type B,触发新一轮 fix_suggestion
//                操作员看到"应用 + 新待处理"重复对
// ============================================================================

// FixSuggestionListParams 修复建议列表查询参数
//
// 嵌入 base.BaseListRequest 自动获得 current / pageSize / orderByColumn / isAsc 四个字段
type FixSuggestionListParams struct {
	base.BaseListRequest

	// 过滤字段
	FixStatus         string     `json:"fixStatus"`         // pending/accepted/rejected/applied/rolled_back/failed
	ConflictType      string     `json:"conflictType"`      // 冲突类型(A/B/C/D/E/F)
	ResponsibleDeptID *string    `json:"responsibleDeptId"` // 责任人部门 ID(JOIN sys_asset.dept_id)
	CreatedFrom       *time.Time `json:"createdFrom,omitempty"`
	CreatedTo         *time.Time `json:"createdTo,omitempty"`
}

// FixSuggestionListItem 修复建议列表展示项
//
// 嵌入 SysReconciliationFixSuggestion 复用其全部字段;额外字段是 JOIN 出来的展示列
type FixSuggestionListItem struct {
	models.SysReconciliationFixSuggestion

	// AssetCode 资产编号(来自 ops_asset.devicesn)
	AssetCode string `gorm:"column:asset_code" json:"assetCode,omitempty"`

	// CurrentUserID 当前 ops_asset.user_id(应用前显示)
	CurrentUserID *string `gorm:"column:current_user_id" json:"currentUserId,omitempty"`

	// SuggestedUsername 建议 user_id 对应的 username(LEFT JOIN sys_user)
	SuggestedUsername *string `gorm:"column:suggested_username" json:"suggestedUsername,omitempty"`
}

// FixSuggestionDetail 修复建议详情(包含异常元数据 + 同 exception_id 历史)
type FixSuggestionDetail struct {
	Suggestion FixSuggestionListItem    `json:"suggestion"`
	Exception  *models.SysDataReconciliation `json:"exception,omitempty"`
	History    []FixSuggestionListItem  `json:"history"`
}

// TrendPoint 趋势点(Stats 用)
type FixSuggestionTrendPoint struct {
	Date     string `json:"date"`
	Pending  int64  `json:"pending"`
	Accepted int64  `json:"accepted"`
	Applied  int64  `json:"applied"`
	Rejected int64  `json:"rejected"`
}

// FixSuggestionStatsResponse 7d 统计响应(D-C5)
type FixSuggestionStatsResponse struct {
	WindowDays        int                       `json:"windowDays"`
	Pending           int64                     `json:"pending"`    // 7d 窗口内 created pending
	PendingAll        int64                     `json:"pendingAll"` // 全量 pending(无 7d 窗口,W-I3 修订)
	Accepted          int64                     `json:"accepted"`   // 7d 窗口内 created accepted
	Rejected          int64                     `json:"rejected"`
	Applied           int64                     `json:"applied"`    // 按 applied_at 过滤(W-2 修订)
	RolledBack        int64                     `json:"rolledBack"` // 按 rolled_back_at 过滤
	Failed            int64                     `json:"failed"`
	MisFixRate        float64                   `json:"misFixRate"`
	Threshold         float64                   `json:"threshold"`
	ThresholdBreached bool                      `json:"thresholdBreached"`
	TrendSeries       []FixSuggestionTrendPoint `json:"trendSeries"`
}

// fixAllowedSortFields 修复建议排序白名单
//
// key 为前端传入的 orderByColumn,value 为安全的物理列名(列名白名单是 SQL 注入防线)
var fixAllowedSortFields = map[string]string{
	"createdAt":       "sys_reconciliation_fix_suggestion.created_at",
	"confidenceScore": "sys_reconciliation_fix_suggestion.confidence_score",
	"fixStatus":       "sys_reconciliation_fix_suggestion.fix_status",
	"appliedAt":       "sys_reconciliation_fix_suggestion.applied_at",
}

// FixSuggestionService 修复建议服务接口
type FixSuggestionService interface {
	// 读端点
	ListFixSuggestions(ctx context.Context, params *FixSuggestionListParams) (*base.PageResult, error)
	GetByID(ctx context.Context, id string) (*FixSuggestionDetail, error)
	Stats(ctx context.Context, windowDays int) (*FixSuggestionStatsResponse, error)

	// 写端点(D-D3 单条)
	Accept(ctx context.Context, id, userID string) error
	Reject(ctx context.Context, id, userID, reason string) error
	Apply(ctx context.Context, id, userID string) error
	Rollback(ctx context.Context, id, userID, reason string) error

	// Cron trigger
	GenerateFixSuggestions(ctx context.Context) (int, error)
}

// fixSuggestionServiceImpl 私有实现
type fixSuggestionServiceImpl struct {
	db            *gorm.DB
	cache         cache.Cache
	configService system.ConfigService
	noticeHub     *websocket.NoticeHub
}

// NewFixSuggestionService 构造函数
func NewFixSuggestionService(db *gorm.DB, c cache.Cache, configSvc system.ConfigService, noticeHub *websocket.NoticeHub) FixSuggestionService {
	return &fixSuggestionServiceImpl{
		db:            db,
		cache:         c,
		configService: configSvc,
		noticeHub:     noticeHub,
	}
}

// ====================== 读端点 ======================

// ListFixSuggestions 列出修复建议
func (s *fixSuggestionServiceImpl) ListFixSuggestions(ctx context.Context, params *FixSuggestionListParams) (*base.PageResult, error) {
	if params == nil {
		return nil, errors.New("查询参数不能为空")
	}

	current := params.Current
	if current <= 0 {
		current = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	// MaxPageSize = 100 防 DoS(参考 stat-cards-from-list-length-capped-at-100 记忆)
	if pageSize > 100 {
		pageSize = 100
	}

	// 基础查询
	query := s.db.WithContext(ctx).
		Model(&models.SysReconciliationFixSuggestion{}).
		Where("sys_reconciliation_fix_suggestion.deleted_at IS NULL").
		Joins("LEFT JOIN sys_data_reconciliation r ON r.id = sys_reconciliation_fix_suggestion.exception_id AND r.deleted_at IS NULL").
		Joins("LEFT JOIN ops_asset a ON a.id = r.asset_id AND a.deleted_at IS NULL").
		Joins("LEFT JOIN sys_user su ON su.id::text = sys_reconciliation_fix_suggestion.suggested_user_id AND su.deleted_at IS NULL").
		Select(`sys_reconciliation_fix_suggestion.*,
			a.devicesn AS asset_code,
			a.user_id AS current_user_id,
			su.username AS suggested_username`)

	if params.FixStatus != "" {
		query = query.Where("sys_reconciliation_fix_suggestion.fix_status = ?", params.FixStatus)
	}
	if params.ConflictType != "" {
		query = query.Where("sys_reconciliation_fix_suggestion.conflict_type = ?", params.ConflictType)
	}
	if params.ResponsibleDeptID != nil && *params.ResponsibleDeptID != "" {
		query = query.Where("a.dept_id = ?", *params.ResponsibleDeptID)
	}
	if params.CreatedFrom != nil {
		query = query.Where("sys_reconciliation_fix_suggestion.created_at >= ?", *params.CreatedFrom)
	}
	if params.CreatedTo != nil {
		query = query.Where("sys_reconciliation_fix_suggestion.created_at <= ?", *params.CreatedTo)
	}

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询修复建议总数失败: %w", err)
	}

	// 排序(白名单)
	query = base.ApplySort(query, params.BaseListRequest, fixAllowedSortFields)

	// 默认排序:created_at DESC
	if params.OrderByColumn == "" {
		query = query.Order("sys_reconciliation_fix_suggestion.created_at DESC")
	}

	// 分页
	var list []FixSuggestionListItem
	if err := query.Offset((current - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询修复建议列表失败: %w", err)
	}

	return &base.PageResult{
		List:     list,
		Total:    total,
		Current:  current,
		PageSize: pageSize,
	}, nil
}

// GetByID 按主键查询单条建议
func (s *fixSuggestionServiceImpl) GetByID(ctx context.Context, id string) (*FixSuggestionDetail, error) {
	if id == "" {
		return nil, errors.New("建议ID不能为空")
	}

	var sugg FixSuggestionListItem
	if err := s.db.WithContext(ctx).
		Table("sys_reconciliation_fix_suggestion").
		Select(`sys_reconciliation_fix_suggestion.*,
			a.devicesn AS asset_code,
			a.user_id AS current_user_id,
			su.username AS suggested_username`).
		Joins("LEFT JOIN sys_data_reconciliation r ON r.id = sys_reconciliation_fix_suggestion.exception_id AND r.deleted_at IS NULL").
		Joins("LEFT JOIN ops_asset a ON a.id = r.asset_id AND a.deleted_at IS NULL").
		Joins("LEFT JOIN sys_user su ON su.id::text = sys_reconciliation_fix_suggestion.suggested_user_id AND su.deleted_at IS NULL").
		Where("sys_reconciliation_fix_suggestion.id = ? AND sys_reconciliation_fix_suggestion.deleted_at IS NULL", id).
		First(&sugg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询修复建议失败: %w", err)
	}

	// 关联异常
	var exception models.SysDataReconciliation
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", sugg.ExceptionID).
		First(&exception).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("查询异常失败: %w", err)
	}

	// 历史:同 exception_id 的所有 fix_suggestion 记录(按 created_at DESC)
	var history []FixSuggestionListItem
	if err := s.db.WithContext(ctx).
		Table("sys_reconciliation_fix_suggestion").
		Select(`sys_reconciliation_fix_suggestion.*,
			a.devicesn AS asset_code,
			a.user_id AS current_user_id,
			su.username AS suggested_username`).
		Joins("LEFT JOIN sys_data_reconciliation r ON r.id = sys_reconciliation_fix_suggestion.exception_id AND r.deleted_at IS NULL").
		Joins("LEFT JOIN ops_asset a ON a.id = r.asset_id AND a.deleted_at IS NULL").
		Joins("LEFT JOIN sys_user su ON su.id::text = sys_reconciliation_fix_suggestion.suggested_user_id AND su.deleted_at IS NULL").
		Where("sys_reconciliation_fix_suggestion.exception_id = ? AND sys_reconciliation_fix_suggestion.deleted_at IS NULL", sugg.ExceptionID).
		Order("sys_reconciliation_fix_suggestion.created_at DESC").
		Find(&history).Error; err != nil {
		return nil, fmt.Errorf("查询修复建议历史失败: %w", err)
	}

	detail := &FixSuggestionDetail{
		Suggestion: sugg,
		History:    history,
	}
	if exception.ID != "" {
		detail.Exception = &exception
	}
	return detail, nil
}

// Stats 7d 统计(D-C5)
func (s *fixSuggestionServiceImpl) Stats(ctx context.Context, windowDays int) (*FixSuggestionStatsResponse, error) {
	if windowDays <= 0 {
		windowDays = 7
	}
	if windowDays > 365 {
		windowDays = 365
	}
	// W-3 修订:统一用 DB-side INTERVAL 避免 app clock vs DB clock 漂移
	// (GORM time 参数会从 app 端转 string,DATE_TRUNC 算时区也以 DB 为准)

	result := &FixSuggestionStatsResponse{WindowDays: windowDays}

	// 6 个独立 Count:统一按"动作发生时间"过滤
	// pending/accepted/rejected/failed:created_at 过滤
	if err := s.db.WithContext(ctx).
		Model(&models.SysReconciliationFixSuggestion{}).
		Where("deleted_at IS NULL AND fix_status = ? AND created_at >= NOW() - ? * INTERVAL '1 day'", "pending", windowDays).
		Count(&result.Pending).Error; err != nil {
		return nil, fmt.Errorf("统计 pending 失败: %w", err)
	}
	if err := s.db.WithContext(ctx).
		Model(&models.SysReconciliationFixSuggestion{}).
		Where("deleted_at IS NULL AND fix_status = ? AND created_at >= NOW() - ? * INTERVAL '1 day'", "accepted", windowDays).
		Count(&result.Accepted).Error; err != nil {
		return nil, fmt.Errorf("统计 accepted 失败: %w", err)
	}
	if err := s.db.WithContext(ctx).
		Model(&models.SysReconciliationFixSuggestion{}).
		Where("deleted_at IS NULL AND fix_status = ? AND created_at >= NOW() - ? * INTERVAL '1 day'", "rejected", windowDays).
		Count(&result.Rejected).Error; err != nil {
		return nil, fmt.Errorf("统计 rejected 失败: %w", err)
	}
	if err := s.db.WithContext(ctx).
		Model(&models.SysReconciliationFixSuggestion{}).
		Where("deleted_at IS NULL AND fix_status = ? AND created_at >= NOW() - ? * INTERVAL '1 day'", "failed", windowDays).
		Count(&result.Failed).Error; err != nil {
		return nil, fmt.Errorf("统计 failed 失败: %w", err)
	}

	// applied(W-2 修订):applied_at 过滤
	if err := s.db.WithContext(ctx).
		Model(&models.SysReconciliationFixSuggestion{}).
		Where("deleted_at IS NULL AND fix_status = ? AND applied_at >= NOW() - ? * INTERVAL '1 day'", "applied", windowDays).
		Count(&result.Applied).Error; err != nil {
		return nil, fmt.Errorf("统计 applied 失败: %w", err)
	}

	// rolledBack:rolled_back_at 过滤
	if err := s.db.WithContext(ctx).
		Model(&models.SysReconciliationFixSuggestion{}).
		Where("deleted_at IS NULL AND fix_status = ? AND rolled_back_at >= NOW() - ? * INTERVAL '1 day'", "rolled_back", windowDays).
		Count(&result.RolledBack).Error; err != nil {
		return nil, fmt.Errorf("统计 rolledBack 失败: %w", err)
	}

	// PendingAll:全量 pending(无 7d 窗口,W-I3 修订)
	if err := s.db.WithContext(ctx).
		Model(&models.SysReconciliationFixSuggestion{}).
		Where("deleted_at IS NULL AND fix_status = ?", "pending").
		Count(&result.PendingAll).Error; err != nil {
		return nil, fmt.Errorf("统计 pendingAll 失败: %w", err)
	}

	// MisFixRate = rolledBack / applied
	if result.Applied > 0 {
		result.MisFixRate = float64(result.RolledBack) / float64(result.Applied)
	}

	// Threshold(从 sys_config 读)
	threshold := 0.01
	if s.configService != nil {
		cfg, _ := s.configService.GetByKey(ctx, "asset.reconciliation.fix.mis_fix_threshold")
		if cfg != nil && cfg.ConfigValue != "" {
			if v, err := strconv.ParseFloat(cfg.ConfigValue, 64); err == nil {
				threshold = v
			}
		}
	}
	result.Threshold = threshold

	// MinSampleSize:最小样本量门槛(W-2026-07-05 修订)
	//
	// 背景: 7d 内若仅有 1 条 applied + 1 条 rolledBack,misFixRate=100%,单点回滚
	// 触发阈值告警属于小样本假阳性(参考 incident 260705-fix-suggestion-flood)。
	// 任意 n=1 时只要发生回滚 → 误报 100%,违反"显著性"原则。
	//
	// 策略: applied < MinSampleSize 时直接返回 ThresholdBreached=false,前端
	// StatsResponse.Applied/RolledBack/MisFixRate 仍正常返回,只是不再触发告警。
	// MinSampleSize 固定 5(典型 A/B 测试最小样本);若产品后续要参数化可加
	// sys_config 'asset.reconciliation.fix.min_sample_size'。
	const minSampleSize int64 = 5
	if result.Applied < minSampleSize {
		result.ThresholdBreached = false
	} else {
		result.ThresholdBreached = result.MisFixRate > threshold
	}

	// TrendSeries(dialect-aware):date_trunc + FILTER(PG)/strftime + CASE(SQLite)
	result.TrendSeries = s.statsTrend(ctx, windowDays)

	return result, nil
}

// statsTrend 7 天趋势聚合
func (s *fixSuggestionServiceImpl) statsTrend(ctx context.Context, windowDays int) []FixSuggestionTrendPoint {
	// 默认空数组,JSON 序列化一致性
	result := []FixSuggestionTrendPoint{}

	dialect := s.db.Dialector.Name()
	var sql string
	switch dialect {
	case "postgres":
		sql = `
SELECT date_trunc('day', created_at)::date AS date,
       COUNT(*) FILTER (WHERE fix_status = 'pending')   AS pending,
       COUNT(*) FILTER (WHERE fix_status = 'accepted')  AS accepted,
       COUNT(*) FILTER (WHERE fix_status = 'applied')   AS applied,
       COUNT(*) FILTER (WHERE fix_status = 'rejected')  AS rejected
FROM sys_reconciliation_fix_suggestion
WHERE deleted_at IS NULL
  AND created_at >= NOW() - ? * INTERVAL '1 day'
GROUP BY date_trunc('day', created_at)
ORDER BY date ASC
`
	default:
		sql = `
SELECT strftime('%Y-%m-%d', created_at) AS date,
       SUM(CASE WHEN fix_status = 'pending'  THEN 1 ELSE 0 END) AS pending,
       SUM(CASE WHEN fix_status = 'accepted' THEN 1 ELSE 0 END) AS accepted,
       SUM(CASE WHEN fix_status = 'applied'  THEN 1 ELSE 0 END) AS applied,
       SUM(CASE WHEN fix_status = 'rejected' THEN 1 ELSE 0 END) AS rejected
FROM sys_reconciliation_fix_suggestion
WHERE deleted_at IS NULL
  AND created_at >= datetime('now', '-' || ? || ' day')
GROUP BY strftime('%Y-%m-%d', created_at)
ORDER BY date ASC
`
	}

	type trendRow struct {
		Date     string
		Pending  int64
		Accepted int64
		Applied  int64
		Rejected int64
	}
	var rows []trendRow
	if err := s.db.WithContext(ctx).Raw(sql, windowDays).Scan(&rows).Error; err != nil {
		applogger.Warnf("[fix-suggestion:Stats] trend query failed: %v", err)
		return result
	}
	for _, r := range rows {
		result = append(result, FixSuggestionTrendPoint{
			Date:     r.Date,
			Pending:  r.Pending,
			Accepted: r.Accepted,
			Applied:  r.Applied,
			Rejected: r.Rejected,
		})
	}
	return result
}

// ====================== 写端点 ======================

// Accept 接受建议(pending → accepted)
func (s *fixSuggestionServiceImpl) Accept(ctx context.Context, id, userID string) error {
	if id == "" {
		return errors.New("建议ID不能为空")
	}
	if userID == "" {
		return errors.New("当前用户ID不能为空")
	}

	now := time.Now()
	res := s.db.WithContext(ctx).Model(&models.SysReconciliationFixSuggestion{}).
		Where("id = ? AND fix_status = ? AND superseded_at IS NULL AND deleted_at IS NULL", id, "pending").
		Updates(map[string]interface{}{
			"fix_status":  "accepted",
			"accepted_at": now,
			"accepted_by": userID,
		})
	if res.Error != nil {
		// partial unique index 23505 兜底 — 已在并发 Accept 后被处理
		if isUniqueViolation(res.Error) {
			return errors.New("该建议已被处理或不存在")
		}
		return fmt.Errorf("接受建议失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.New("该建议已被处理或不存在")
	}
	applogger.Infof("[fix-suggestion] 接受建议 id=%s userID=%s", id, userID)
	return nil
}

// Reject 拒绝建议(pending → rejected,reason ≥10 字符)
func (s *fixSuggestionServiceImpl) Reject(ctx context.Context, id, userID, reason string) error {
	if id == "" {
		return errors.New("建议ID不能为空")
	}
	if userID == "" {
		return errors.New("当前用户ID不能为空")
	}
	if len(reason) < 10 {
		return errors.New("拒绝原因至少 10 字符")
	}

	now := time.Now()
	res := s.db.WithContext(ctx).Model(&models.SysReconciliationFixSuggestion{}).
		Where("id = ? AND fix_status = ? AND superseded_at IS NULL AND deleted_at IS NULL", id, "pending").
		Updates(map[string]interface{}{
			"fix_status":       "rejected",
			"rejected_at":      now,
			"rejected_by":      userID,
			"rejection_reason": reason,
		})
	if res.Error != nil {
		if isUniqueViolation(res.Error) {
			return errors.New("该建议已被处理或不存在")
		}
		return fmt.Errorf("拒绝建议失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.New("该建议已被处理或不存在")
	}
	applogger.Infof("[fix-suggestion] 拒绝建议 id=%s userID=%s reason=%s", id, userID, reason)
	return nil
}

// Apply 应用建议(accepted → applied,事务内 5 步)
//
// B-3 关键修复:必须同步写 sys_data_reconciliation.resolved_at = NOW(),
//              否则下次 DetectLayer3 cron 重新检出该 Type B,触发新一轮 fix_suggestion,
//              操作员看到 "应用 + 新待处理" 重复对(RESEARCH.md §13.5 隐藏陷阱)
func (s *fixSuggestionServiceImpl) Apply(ctx context.Context, id, userID string) error {
	if id == "" {
		return errors.New("建议ID不能为空")
	}
	if userID == "" {
		return errors.New("当前用户ID不能为空")
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 读 accepted 建议
	var sugg models.SysReconciliationFixSuggestion
	if err := tx.Where("id = ? AND fix_status = ? AND deleted_at IS NULL", id, "accepted").First(&sugg).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("该建议不存在或未处于 accepted 状态")
		}
		return fmt.Errorf("查询建议失败: %w", err)
	}

	// 2. JOIN sys_data_reconciliation 取 asset_id
	var exception models.SysDataReconciliation
	if err := tx.Where("id = ? AND deleted_at IS NULL", sugg.ExceptionID).First(&exception).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("查询异常失败: %w", err)
	}

	// 3. 读 ops_asset 当前 user_id(用作 pre_fix_user_id 回填)
	var asset models.Asset
	if err := tx.Where("id = ? AND deleted_at IS NULL", exception.AssetID).First(&asset).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("查询资产失败: %w", err)
	}
	preFixUserID := asset.UserID

	// 4. UPDATE ops_asset.user_id(核心修复写 — D-A1)
	//    注意 ops_asset.user_id 是 varchar size:64,不做 ?::uuid 强转
	//    (参 xingran-info-point-port-id-varchar 记忆)
	if err := tx.Model(&models.Asset{}).
		Where("id = ?", exception.AssetID).
		Update("user_id", sugg.SuggestedUserID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新 ops_asset.user_id 失败: %w", err)
	}

	// 5. UPDATE 建议状态 + pre_fix_user_id + DB-side INTERVAL '7 day' 回滚窗口
	//    W-3 修订:用 DB-side INTERVAL 避免 app clock vs DB clock 漂移
	now := time.Now()
	if err := tx.Model(&sugg).Updates(map[string]interface{}{
		"fix_status":            "applied",
		"applied_at":            now,
		"applied_by":            userID,
		"pre_fix_user_id":       preFixUserID,
		"rollback_window_until": gorm.Expr("NOW() + INTERVAL '7 day'"),
	}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新建议 applied 状态失败: %w", err)
	}

	// 6. 【B-3 关键】UPDATE sys_data_reconciliation.resolved_at
	//    必须同步写,否则下次 DetectLayer3 cron 重新检出该 Type B
	//    注: 实际列名为 resolution_note (text), PLAN 误写为 resolution_method
	if err := tx.Model(&exception).Updates(map[string]interface{}{
		"resolved_at":     now,
		"resolution_note": "fix_suggestion_applied",
	}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新异常 resolved_at 失败[B-3 关键]: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	applogger.Infof("[fix-suggestion] 应用建议 id=%s userID=%s asset_id=%s pre_user=%v new_user=%v", id, userID, exception.AssetID, preFixUserID, sugg.SuggestedUserID)
	return nil
}

// Rollback 回滚应用(applied → rolled_back,7d 窗口内)
//
// 校验顺序加固(46-02 / D-C1+D-C2+D-C3):
//   1) id / userID 非空
//   2) len(reason) >= 10(D-C3 审计:回滚必须填原因)
//   3) 读 applied 建议(WHERE fix_status='applied' AND deleted_at IS NULL)→ 防御性检查
//      3a) sugg.RollbackWindowUntil 非 nil
//      3b) time.Now().Before(*sugg.RollbackWindowUntil)(7d 内,Go-side 防御性 + DB-side 主判定)
//   4) sugg.PreFixUserID 非 nil(防止数据被外部修改为 NULL 致回滚静默写空值)
//   5) DB-side 主判定:SELECT (rollback_window_until > NOW()) — 避免 app clock 漂移
//   6) 反查 exception → asset_id → UPDATE ops_asset.user_id = pre_fix_user_id(D-C1)
//   7) UPDATE sys_reconciliation_fix_suggestion SET fix_status='rolled_back' + 时间戳
//   8) UPDATE sys_data_reconciliation SET resolved_at = NULL(让 DetectLayer3 重新检出)
func (s *fixSuggestionServiceImpl) Rollback(ctx context.Context, id, userID, reason string) error {
	if id == "" {
		return errors.New("建议ID不能为空")
	}
	if userID == "" {
		return errors.New("当前用户ID不能为空")
	}
	if len(reason) < 10 {
		return errors.New("回滚原因至少 10 字符")
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 读 applied 建议(带 fix_status='applied' 守卫,保证状态机严格)
	var sugg models.SysReconciliationFixSuggestion
	if err := tx.Where("id = ? AND fix_status = ? AND deleted_at IS NULL", id, "applied").First(&sugg).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("该建议不存在或未处于 applied 状态")
		}
		return fmt.Errorf("查询建议失败: %w", err)
	}

	// 2. 7d 回滚窗口硬约束(D-C2) — Go-side 防御性兜底
	//    真正权威的判定在 step 5 DB-side 表达式(app clock 可能与 DB clock 漂移)
	if sugg.RollbackWindowUntil == nil {
		tx.Rollback()
		return errors.New("回滚窗口未设置,无法回滚")
	}
	if time.Now().After(*sugg.RollbackWindowUntil) {
		tx.Rollback()
		return errors.New("回滚窗口已过(7d),不允许回滚")
	}

	// 3. (原 pre_fix_user_id != nil 防御性检查已删除:
	//    原 ops_asset.user_id 为 NULL 时 Apply 写 pre_fix_user_id=NULL,
	//    Rollback 需允许恢复 NULL 才能完整撤销到原状, 此检查过度防御, 删除.
	//    GORM Update("user_id", nil) 会正确写 NULL.)

	// 4. 反查 exception → asset_id
	var exception models.SysDataReconciliation
	if err := tx.Where("id = ? AND deleted_at IS NULL", sugg.ExceptionID).First(&exception).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("查询异常失败: %w", err)
	}

	// 5. DB-side 主判定(避免 app clock vs DB clock 漂移,W-3 修订通用原则)
	//    即使 step 2 的 Go-side 判定通过,DB-side 仍可能拒绝(时钟漂移场景)
	var stillInWindow bool
	row := tx.Raw("SELECT (rollback_window_until > NOW()) AS in_window FROM sys_reconciliation_fix_suggestion WHERE id = ?", id).Row()
	if err := row.Scan(&stillInWindow); err != nil {
		tx.Rollback()
		return fmt.Errorf("回滚窗口检查失败: %w", err)
	}
	if !stillInWindow {
		tx.Rollback()
		return errors.New("回滚窗口已过(7d),不允许回滚")
	}

	// 6. 恢复 ops_asset.user_id = pre_fix_user_id(D-C1 粒度仅 user_id)
	//    注意 ops_asset.user_id 是 varchar size:64,不做 ?::uuid 强转
	//    (参 xingran-info-point-port-id-varchar 记忆)
	if err := tx.Model(&models.Asset{}).
		Where("id = ?", exception.AssetID).
		Update("user_id", sugg.PreFixUserID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("回滚 ops_asset.user_id 失败: %w", err)
	}

	// 7. UPDATE 建议状态 → rolled_back(D-C2/D-C3 落库)
	now := time.Now()
	if err := tx.Model(&sugg).Updates(map[string]interface{}{
		"fix_status":      "rolled_back",
		"rolled_back_at":  now,
		"rolled_back_by":  userID,
		"rollback_reason": reason,
	}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新建议 rolled_back 状态失败: %w", err)
	}

	// 8. 反向操作 sys_data_reconciliation.resolved_at = NULL + workorder_id 不动
	//    让 DetectLayer3 下次 cron 重新检出该 Type B
	if err := tx.Model(&exception).Update("resolved_at", nil).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("解除异常 resolved_at 失败: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	applogger.Infof("[fix-suggestion] 回滚应用 id=%s userID=%s reason=%s", id, userID, reason)
	return nil
}

// isUniqueViolation 检查 GORM 错误是否为 PG unique violation (SQLSTATE 23505)
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "23505") || contains(msg, "duplicate key value")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// WorkstationIDForSuggestion 反查 fix_suggestion 关联工位(D-C4 缓存失效)
//
// 复用 reconciliation_handler.go:168-185 的 gorm.Row().Scan() + sql.ErrNoRows 模式
// (gorm.Row().Scan() 返回 sql.ErrNoRows 而非 gorm.ErrRecordNotFound)
func (s *fixSuggestionServiceImpl) WorkstationIDForSuggestion(ctx context.Context, suggestionID string) (string, error) {
	var wsID sql.NullString
	scanErr := s.db.WithContext(ctx).
		Table("reconciliation_normalized").
		Select("reconciliation_normalized.workstation_id").
		Joins("JOIN sys_data_reconciliation ON sys_data_reconciliation.asset_id = reconciliation_normalized.asset_id").
		Joins("JOIN sys_reconciliation_fix_suggestion ON sys_reconciliation_fix_suggestion.exception_id = sys_data_reconciliation.id").
		Where("sys_reconciliation_fix_suggestion.id = ? AND sys_reconciliation_fix_suggestion.deleted_at IS NULL", suggestionID).
		Limit(1).
		Row().
		Scan(&wsID)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return "", nil
		}
		return "", scanErr
	}
	if wsID.Valid {
		return wsID.String, nil
	}
	return "", nil
}
