package asset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
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
//  6. 构造 sys_data_reconciliation 行 + UPSERT(2026-07-03 Phase 47 R3 改造)
//     — ON CONFLICT 复用 partial unique index uniq_recon_asset_type_open,
//       命中即更新 9 字段 EXCLUDED.* + detected_at = CURRENT_TIMESTAMP,UPSERT 命中计入 inserted
//  7. (D-01 死代码删除) 不再有 unique violation catch 静默 skip 路径 —
//     24h 内同 (asset_id, conflict_type) 已存在 open 行由 PG 走 UPDATE,不再抛错
//  8. 计数 inserted / skipped / skippedSilence / skippedThrottle 返回
//
// 2026-07-03 Phase 47 R3 改造:
//   - D-01: INSERT → UPSERT (ON CONFLICT 复用 partial unique index 列序匹配)
//   - D-02: 9 字段 EXCLUDED.* + detected_at=CURRENT_TIMESTAMP (resolved_at/resolved_by 不更新)
//   - D-03: 返回签名保留,UPSERT 命中计入 inserted 而非 skipped
//   - 移除 isReconciliationDuplicate(err) catch 路径,函数定义保留
func (s *reconciliationDetectionImpl) DetectLayer3(ctx context.Context) (int, int, int, int, error) {
	var rows []NormalizedRow
	if err := s.db.WithContext(ctx).
		Table("reconciliation_normalized").
		Find(&rows).Error; err != nil {
		return 0, 0, 0, 0, err
	}

	// === R3 / D-R3-A1-03 Layer 3.5 前置:预加载 active 例外规则 ===
	// 一次性 SELECT * FROM sys_reconciliation_exception WHERE is_active=0 AND deleted_at IS NULL,
	// 逐条 net.ParseCIDR 预编译。循环内零 DB 查询(性能架构 D-R3-A1-03)。
	activeRules := preloadActiveRules(s.db.WithContext(ctx))
	exceptionHit := 0 // 统计命中例外数(仅日志,不影响返回签名)

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
			// === D-09-fix-01 (2026-07-02 排查 .planning/debug/reconciliation-health-misjudgment):
			// 资产变健康时, 把该资产所有 open 冲突 (resolved_at IS NULL) 标记为
			// auto-resolved. 否则:
			//   1. HealthScore = (1 - open_count/total)*100 仍计入老 open → 红点误报
			//      (典型: 4F001 三路径设备全一致但因 C 记录 open 导致 score=0 红)
			//   2. GetByWorkstation LEFT JOIN resolved_at IS NULL → assets[].conflictType=C
			//      → HealthBadge 红点 + tooltip "物理有/责任人不一致(高危)"
			//   3. 异常列表长期显示已不存在的冲突
			//
			// 边界与权衡:
			//   - 仅 UPDATE resolved_at IS NULL 的 open 记录, 已 resolved 的不动 (审计)
			//   - resolved_by 标 "system:auto-resolve-on-healthy" 区分人工 vs 系统
			//   - 不限定 grace period: open 记录存在即代表上次检测时冲突, 现在 A 即
			//     表示已自愈, 即时 resolve 是正确行为
			//   - 静默执行, 失败不阻塞 skipped 计数 (UI 降级在 GetByWorkstation 已处理)
			//   - 性能: 每次 cron (5min) 跑一次, open 记录数 << 总资产, 可接受
			if err := s.db.WithContext(ctx).
				Model(&models.SysDataReconciliation{}).
				Where("asset_id = ? AND deleted_at IS NULL AND resolved_at IS NULL", row.AssetID).
				Updates(map[string]interface{}{
					"resolved_at": time.Now(),
					"resolved_by": "system:auto-resolve-on-healthy",
				}).Error; err != nil {
				applogger.Warnf("[reconciliation] auto-resolve-on-healthy 失败 asset_id=%s: %v",
					row.AssetID, err)
				// 不 return: 降级处理, 不影响 skipped 统计; 下次 cron 会重试
			}
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

		// === R3 / D-R3-A1-02 Layer 3.5: 例外规则匹配 ===
		//
		// 例外过滤集中在 DetectLayer3 循环内,一次性匹配后写 exception_rule_id +
		// applied_actions 到 sys_data_reconciliation(单一真相源 D-R3-A1-02)。
		// 下游通路(R2 WS / SysNotice / 转单 cron)读 applied_actions 决定行为,
		// 不各自重查例外表。
		//
		// matchException 纯函数(Task 2):返回首条命中规则 ID + 合并 actions +
		// 最终 severity + isSilence。命中时填入 rec 的 ExceptionRuleID +
		// AppliedActions + 调整 Severity(D-R3-A2-02 skip_severity 降级 + severity_override 取最低)。
		// silence 命中仍写表(D-R3-A1-01),仅 ListExceptions 默认过滤。
		confidence := s.ComputeConfidence(signals)
		severity := s.ComputeSeverity(conflictType)

		var exceptionRuleID *string
		var appliedActions pq.StringArray

		// 资产 IP 非空时参与例外匹配(IP 是 CIDR 匹配的必要条件)
		// 资产责任人 user_id 用于 dept/user scope 双条件匹配(D-R3-A3-01)
		assetUserIDForMatch := ""
		if row.AssetUserID != nil {
			assetUserIDForMatch = *row.AssetUserID
		}
		if row.AssetIP != nil && *row.AssetIP != "" {
			ruleID, actions, finalSev, _ := matchExceptionWithSeverity(activeRules, *row.AssetIP, assetUserIDForMatch, conflictType, severity)
			if ruleID != "" {
				exceptionRuleID = &ruleID
				appliedActions = actions
				severity = finalSev // R3/D-R3-A2-02: skip_severity 降级 + severity_override 取最低
				exceptionHit++
			}
		}

		// 脏数据防御:ops_asset.machine_ip 是 inet 列,历史数据可能含非法值(如 '192.168.x.x*10.x.x.x')。
		// MV 直接 SELECT machine_ip,脏值会原样穿透到 row.AssetIP,后续 INSERT inet 列触发
		// SQLSTATE 22P02 "invalid input syntax for type inet"。这里做轻量校验:含 * 或空格的视为脏数据,
		// AssetIP 置 nil(不写入),其余字段继续保存,异常检测结论不受影响。
		if row.AssetIP != nil {
			if ip := *row.AssetIP; ip == "" || strings.ContainsAny(ip, "* ") {
				row.AssetIP = nil
			}
		}

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
			ExceptionRuleID: exceptionRuleID, // R3 / D-R3-A1-02 Layer 3.5 命中例外规则 ID
			AppliedActions:  appliedActions,  // R3 / D-R3-A1-02 合并后的 actions(可能含 silence)
			DetectedAt:      time.Now(),
		}

		// R3 / D-01/D-02/D-03: UPSERT 语义(2026-07-03 Phase 47 改造)
		// 原实现 INSERT-only, 同 (asset_id, conflict_type) 24h 内已存在 open 行 →
		//   抛 SQLSTATE 23505 → isReconciliationDuplicate catch → skipped++。
		// 现改为 ON CONFLICT 复用 partial unique index uniq_recon_asset_type_open
		//   (resolved_at IS NULL AND deleted_at IS NULL),命中即更新 9 字段
		//   severity/raw_snapshot/physical_value/declared_value/ad_value/
		//   asset_ip/exception_rule_id/applied_actions/confidence_score +
		//   detected_at = NOW()。
		// 返回 inserted 累计 INSERT 或 UPDATE 命中数(语义统一 D-03)。
		rec.ID = ""                  // 显式重置,确保 GORM 走 INSERT-or-UPDATE 而非 full UPDATE
		rec.CreatedAt = time.Time{}  // 同上
		// 修复 SQLSTATE 42P10 (debug: recon-upsert-partial-idx):
		// migration_201 (Phase 48-01) 切换了 partial unique index:
		//   DROP uniq_recon_asset_type_open (asset_id, conflict_type)
		//   CREATE uniq_recon_asset_type_cat_open (asset_id, conflict_type, recon_category)
		//     WHERE resolved_at IS NULL AND deleted_at IS NULL (三列, 谓词不变)
		// PG 与 SQLite(3.35+) 对 partial unique index 的 ON CONFLICT inference 都强制要求
		// conflict_target 显式带 index_predicate, 且列集精确匹配 index 列(此处三列);
		// 否则报 "there is no unique or exclusion constraint matching the ON CONFLICT
		// specification"(PG, 42P10) / "ON CONFLICT clause does not match any PRIMARY KEY
		// or UNIQUE constraint"(SQLite)。故 TargetWhere 跨 dialect 统一:
		//   ON CONFLICT (asset_id, conflict_type, recon_category) WHERE resolved_at IS NULL AND deleted_at IS NULL DO UPDATE
		// 注: DetectLayer3 不设 ReconCategory (NULL); 三列 index 下 NULL 语义使同 (asset,type,NULL)
		// 不冲突, 靠 24h 节流 guard (D-A3-02) 防重复 INSERT。UPSERT UPDATE 路径对 recon_category
		// 非 NULL 场景 (如 component_serial) 生效。
		onConflict := clause.OnConflict{
			Columns: []clause.Column{
				{Name: "asset_id"},
				{Name: "conflict_type"},
				{Name: "recon_category"},
			},
			TargetWhere: clause.Where{Exprs: []clause.Expression{
				clause.Expr{SQL: "resolved_at IS NULL"},
				clause.Expr{SQL: "deleted_at IS NULL"},
			}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"severity":          gorm.Expr("EXCLUDED.severity"),
				"raw_snapshot":      gorm.Expr("EXCLUDED.raw_snapshot"),
				"physical_value":    gorm.Expr("EXCLUDED.physical_value"),
				"declared_value":    gorm.Expr("EXCLUDED.declared_value"),
				"ad_value":          gorm.Expr("EXCLUDED.ad_value"),
				"asset_ip":          gorm.Expr("EXCLUDED.asset_ip"),
				"exception_rule_id": gorm.Expr("EXCLUDED.exception_rule_id"),
				"applied_actions":   gorm.Expr("EXCLUDED.applied_actions"),
				"confidence_score":  gorm.Expr("EXCLUDED.confidence_score"),
				"detected_at":       gorm.Expr("CURRENT_TIMESTAMP"),
			}),
		}
		err := s.db.WithContext(ctx).Clauses(onConflict).Create(rec).Error
		if err != nil {
			return inserted, skipped, skippedSilence, skippedThrottle, err
		}
		inserted++ // D-03: UPSERT 命中(无论 INSERT 或 UPDATE)都计入 inserted
	}

	// R3 / D-R3-A1-02 — Layer 3.5 命中例外规则数日志(运维可观测降噪效果)
	if exceptionHit > 0 {
		logrus.Infof("[reconciliation:DetectLayer3] Layer 3.5 命中例外规则 %d 条(inserted=%d)", exceptionHit, inserted)
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

// 防止 isReconciliationDuplicate unused 警告 — D-01 死代码,函数定义保留供单元测试 / 外部兼容路径引用
var _ = isReconciliationDuplicate