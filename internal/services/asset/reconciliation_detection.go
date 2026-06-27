package asset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
)

// NormalizedRow reconciliation_normalized 物化视图单行(R1 + R2 检测引擎输入)
//
// R2 新增 3 字段(Phase 43 / D-A3-01):LastResolvedAt / LastResolvedBy / LastConflictType
// — 来自 LEFT JOIN LATERAL,语义为"该资产最近一条已解决异常"的快照。
// DetectLayer3 据此实现 7d 静默期(同 asset + type 在 7 天内不重复触发)。
type NormalizedRow struct {
	AssetID          string     `gorm:"column:asset_id" json:"assetId"`
	AssetCode        string     `gorm:"column:asset_code" json:"assetCode"`
	AssetIP          *string    `gorm:"column:asset_ip" json:"assetIp,omitempty"`
	PhysicalUserID   *string    `gorm:"column:physical_user_id" json:"physicalUserId,omitempty"`
	PhysicalUsername *string    `gorm:"column:physical_username" json:"physicalUsername,omitempty"`
	AssetUserID      *string    `gorm:"column:asset_user_id" json:"assetUserId,omitempty"`
	AssetUsername    *string    `gorm:"column:asset_username" json:"assetUsername,omitempty"`
	AdUserID         *string    `gorm:"column:ad_id" json:"adUserId,omitempty"`
	AdUsername       *string    `gorm:"column:ad_username" json:"adUsername,omitempty"`
	AdIsEnabled      *bool      `gorm:"column:ad_is_enabled" json:"adIsEnabled,omitempty"`
	MVRefreshedAt    time.Time  `gorm:"column:mv_refreshed_at" json:"mvRefreshedAt"`
	MAC1             *string    `gorm:"column:mac1" json:"mac1,omitempty"`
	MAC2             *string    `gorm:"column:mac2" json:"mac2,omitempty"`
	MACJoin          *string    `gorm:"column:mac_join" json:"macJoin,omitempty"`
	// === Phase 43 R2 / D-A3-01 新增字段 ===
	LastResolvedAt   *time.Time `gorm:"column:last_resolved_at" json:"lastResolvedAt,omitempty"`
	LastResolvedBy   *string    `gorm:"column:last_resolved_by" json:"lastResolvedBy,omitempty"`
	LastConflictType *string    `gorm:"column:last_conflict_type" json:"lastConflictType,omitempty"`
}

// ConflictSignals Layer 3 分类信号(三路存在 + 两路匹配)
type ConflictSignals struct {
	HasPhysical           bool
	HasDeclared           bool
	HasAD                 bool
	PhysicalMatchDeclared bool
	PhysicalMatchAD       bool
}

// ClassifiedException 分类结果
type ClassifiedException struct {
	AssetID      string
	ConflictType string
	Severity     string
	Confidence   float64
	RawSnapshot  map[string]interface{}
}

// ReconciliationDetection Layer 3 检测引擎接口
type ReconciliationDetection interface {
	// DetectLayer3 遍历 reconciliation_normalized 物化视图,按 Type A-F 分类,
	// 写 sys_data_reconciliation 主表。
	//
	// R1 返回值: (inserted int, skipped int, err error)
	// R2 扩展返回值(Phase 43 / D-A3-03):(inserted int, skipped int,
	// skippedSilence int, skippedThrottle int, err error)
	//
	// 计数语义:
	//   - inserted       — 本轮 INSERT 成功笔数(Type B-F,D-09 健康跳过不算)
	//   - skipped        — Type A 健康跳过 + partial unique index 冲突跳过(D-09/D-11)
	//   - skippedSilence — 7d 静默期命中跳过(D-A3-01)
	//   - skippedThrottle — 24h 节流命中跳过(D-A3-02)
	DetectLayer3(ctx context.Context) (inserted int, skipped int, skippedSilence int, skippedThrottle int, err error)
	// ClassifySignals (纯函数,可独立测试)从行数据提取三路存在 + 两路匹配信号
	ClassifySignals(row NormalizedRow) ConflictSignals
	// ClassifyType (纯函数)从信号判定 Type A-F
	ClassifyType(sig ConflictSignals) string
	// ComputeConfidence (纯函数)按 physical*0.5 + declared*0.3 + ad*0.2 计算
	ComputeConfidence(sig ConflictSignals) float64
	// ComputeSeverity (纯函数)按 type 映射 severity
	ComputeSeverity(t string) string
}

type reconciliationDetectionImpl struct {
	db *gorm.DB
}

// NewReconciliationDetection 构造 Layer 3 检测引擎
func NewReconciliationDetection(db *gorm.DB) ReconciliationDetection {
	return &reconciliationDetectionImpl{db: db}
}

// ClassifySignals 从 NormalizedRow 提取三路信号
//
// D-08 物理链路 R1 简化:physical_user_id 来自 ops_asset.user_id(MV 中已 join)。
// D-09 has_physical / has_declared / has_ad 三路独立判定。
func (s *reconciliationDetectionImpl) ClassifySignals(row NormalizedRow) ConflictSignals {
	hasPhysical := row.PhysicalUserID != nil && *row.PhysicalUserID != ""
	hasDeclared := row.AssetUserID != nil && *row.AssetUserID != ""
	hasAD := row.AdUserID != nil && *row.AdUserID != ""

	var physMatchDecl, physMatchAD bool
	if hasPhysical && hasDeclared && row.PhysicalUserID != nil && row.AssetUserID != nil {
		physMatchDecl = *row.PhysicalUserID == *row.AssetUserID
	}
	if hasPhysical && hasAD && row.PhysicalUserID != nil && row.AdUserID != nil {
		physMatchAD = *row.PhysicalUserID == *row.AdUserID
	}

	return ConflictSignals{
		HasPhysical:           hasPhysical,
		HasDeclared:           hasDeclared,
		HasAD:                 hasAD,
		PhysicalMatchDeclared: physMatchDecl,
		PhysicalMatchAD:       physMatchAD,
	}
}

// ClassifyType 把 ConflictSignals 映射为 Type A-F
//
// 规则(D-08/D-09 锁定):
//   A: physical + declared 匹配(健康,不入主表)
//   B: physical 有, declared 无(物理链路有用户,但 ops_asset 没标责任人)
//   C: physical + declared 都有但不匹配(物理使用人与责任人冲突)
//   D: physical 无, declared 有(只有责任人,无物理使用者)
//   E: physical + declared 都没有(异常:无用户关联)
//   F: physical/declared 匹配但 ad 不一致(AD 账号不一致)
func (s *reconciliationDetectionImpl) ClassifyType(sig ConflictSignals) string {
	if sig.HasPhysical && sig.HasDeclared && sig.PhysicalMatchDeclared {
		// 健康:三路检查 AD 是否一致
		if sig.HasAD && !sig.PhysicalMatchAD {
			return "F"
		}
		return "A"
	}
	if sig.HasPhysical && !sig.HasDeclared {
		return "B"
	}
	if sig.HasPhysical && sig.HasDeclared && !sig.PhysicalMatchDeclared {
		return "C"
	}
	if !sig.HasPhysical && sig.HasDeclared {
		return "D"
	}
	if !sig.HasPhysical && !sig.HasDeclared {
		return "E"
	}
	// 兜底:任何其他未覆盖组合(例如 hasPhysical==false && hasAD==true)归 F
	return "F"
}

// ComputeConfidence 按三路匹配计算置信度(0.00-1.00)
//
// 公式:physical*0.5 + declared*0.3 + ad*0.2
// R1 系数硬编码(sys_config 锁定在 R1 不可改);R2 可从 sys_config 读取
func (s *reconciliationDetectionImpl) ComputeConfidence(sig ConflictSignals) float64 {
	score := 0.0
	if sig.PhysicalMatchDeclared {
		score += 0.5
	}
	if sig.HasDeclared {
		score += 0.3
	}
	if sig.HasAD && sig.PhysicalMatchAD {
		score += 0.2
	}
	// 截断到 2 位小数
	return float64(int(score*100)) / 100.0
}

// ComputeSeverity 按 type 映射 severity
//
// 规则: B/C = high, D/F = medium, E = low, A 不入主表
func (s *reconciliationDetectionImpl) ComputeSeverity(t string) string {
	switch t {
	case "B", "C":
		return "high"
	case "D", "F":
		return "medium"
	case "E":
		return "low"
	default:
		return "low"
	}
}

// DetectLayer3 Layer 3 同步检测引擎
//
// 流程(R2 升级版):
//  1. 读 reconciliation_normalized 全部行
//  2. 逐行 ClassifySignals + ClassifyType
//  3. Type A 跳过(D-09,计入 skipped)
//  4. [R2 / D-A3-01] guard 1:7d 静默期拦截
//     — 若 last_resolved_at + last_conflict_type 与当前 conflictType 匹配,
//       且 NOW() - last_resolved_at < 7d → 计入 skippedSilence,continue
//  5. [R2 / D-A3-02] guard 2:24h 节流拦截
//     — 查 sys_data_reconciliation WHERE detected_at > NOW() - INTERVAL '24h'
//       AND asset_id=? AND conflict_type=? AND deleted_at IS NULL,
//       命中则计入 skippedThrottle,continue
//  6. 构造 sys_data_reconciliation 行 + INSERT
//  7. unique violation on uniq_recon_asset_type_open catch 跳过(D-11,计入 skipped)
//  8. 计数 inserted / skipped / skippedSilence / skippedThrottle 返回
func (s *reconciliationDetectionImpl) DetectLayer3(ctx context.Context) (int, int, int, int, error) {
	var rows []NormalizedRow
	if err := s.db.WithContext(ctx).
		Table("reconciliation_normalized").
		Find(&rows).Error; err != nil {
		return 0, 0, 0, 0, err
	}

	inserted := 0
	skipped := 0
	skippedSilence := 0
	skippedThrottle := 0

	// 7d 静默期时长(硬编码,D-A3-01 锁定 ROADMAP SC 7 默认值)
	const silencePeriod = 7 * 24 * time.Hour
	// 24h 节流时长(硬编码,D-A3-02 锁定 ROADMAP SC 8 默认值)
	const throttleWindow = 24 * time.Hour

	for _, row := range rows {
		signals := s.ClassifySignals(row)
		conflictType := s.ClassifyType(signals)
		if conflictType == "A" {
			skipped++
			continue // D-09 健康不写
		}

		// === R2 / D-A3-01 guard 1: 7d 静默期 ===
		// 条件:MV.last_resolved_at 非空 + last_conflict_type 等于当前 type +
		//       距 last_resolved_at < 7d → 跳过
		// 这是运维"标记已解决"后的 7 天静默窗口,避免同一异常被反复触发
		if row.LastResolvedAt != nil && row.LastConflictType != nil &&
			*row.LastConflictType == conflictType &&
			time.Since(*row.LastResolvedAt) < silencePeriod {
			skippedSilence++
			continue
		}

		// === R2 / D-A3-02 guard 2: 24h 节流 ===
		// 条件:sys_data_reconciliation 24h 内同 (asset_id, conflict_type) 已有记录 → 跳过
		// 注意:这条 guard 写在 guard 1 之后,语义上"即使没有 resolved 记录,
		// 24h 内已插过同 (asset, type) 也不再重复插"
		var recentCount int64
		throttleFrom := time.Now().Add(-throttleWindow)
		if err := s.db.WithContext(ctx).
			Model(&models.SysDataReconciliation{}).
			Where("asset_id = ? AND conflict_type = ? AND detected_at > ? AND deleted_at IS NULL",
				row.AssetID, conflictType, throttleFrom).
			Count(&recentCount).Error; err != nil {
			return inserted, skipped, skippedSilence, skippedThrottle,
				fmt.Errorf("24h 节流查询失败 asset_id=%s: %w", row.AssetID, err)
		}
		if recentCount > 0 {
			skippedThrottle++
			continue
		}

		// 脏数据防御:ops_asset.machine_ip 是 inet 列,历史数据可能含非法值(如 '192.168.x.x*10.x.x.x')。
		// MV 直接 SELECT machine_ip,脏值会原样穿透到 row.AssetIP,后续 INSERT inet 列触发
		// SQLSTATE 22P02 "invalid input syntax for type inet"。这里做轻量校验:含 * 或空格的视为脏数据,
		// AssetIP 置 nil(不写入),其余字段继续保存,异常检测结论不受影响。
		if row.AssetIP != nil {
			if s := *row.AssetIP; s == "" || strings.ContainsAny(s, "* ") {
				row.AssetIP = nil
			}
		}

		confidence := s.ComputeConfidence(signals)
		severity := s.ComputeSeverity(conflictType)

		// 构造 RawSnapshot(jsonb)
		rawSnapshot, _ := json.Marshal(map[string]interface{}{
			"asset_id":          row.AssetID,
			"asset_code":        row.AssetCode,
			"asset_ip":          row.AssetIP,
			"physical_user_id":  row.PhysicalUserID,
			"physical_username": row.PhysicalUsername,
			"asset_user_id":     row.AssetUserID,
			"asset_username":    row.AssetUsername,
			"ad_id":             row.AdUserID,
			"ad_username":       row.AdUsername,
			"signals":           signals,
		})

		// 构造三路 evidence(jsonb)
		physicalVal, _ := json.Marshal(map[string]interface{}{
			"user_id":  row.PhysicalUserID,
			"username": row.PhysicalUsername,
		})
		declaredVal, _ := json.Marshal(map[string]interface{}{
			"user_id":  row.AssetUserID,
			"username": row.AssetUsername,
		})
		adVal, _ := json.Marshal(map[string]interface{}{
			"ad_id":       row.AdUserID,
			"ad_username": row.AdUsername,
			"is_enabled":  row.AdIsEnabled,
		})

		rec := &models.SysDataReconciliation{
			AssetID:         row.AssetID,
			ConflictType:    conflictType,
			Severity:        severity,
			PhysicalValue:   physicalVal,
			DeclaredValue:   declaredVal,
			ADValue:         adVal,
			ConfidenceScore: confidence,
			RawSnapshot:     rawSnapshot,
			AssetIP:         row.AssetIP,
			DetectedAt:      time.Now(),
		}

		err := s.db.WithContext(ctx).Create(rec).Error
		if err != nil {
			if isReconciliationDuplicate(err) {
				logrus.WithError(err).WithField("asset_id", row.AssetID).
					Debug("Layer 3 detect: duplicate key skipped (D-11)")
				skipped++
				continue
			}
			return inserted, skipped, skippedSilence, skippedThrottle, err
		}
		inserted++
	}

	return inserted, skipped, skippedSilence, skippedThrottle, nil
}

// isReconciliationDuplicate 检测 PG unique_violation(23505)或 SQLite UNIQUE 约束失败
func isReconciliationDuplicate(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "23505") ||
		strings.Contains(msg, "unique_violation")
}

// 防止 unused import 警告
var _ = apperrors.BadRequest