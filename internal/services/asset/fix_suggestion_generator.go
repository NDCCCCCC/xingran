package asset

import (
	"context"
	"errors"
	"fmt"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// GenerateFixSuggestions 扫描 Type B 高置信度异常 → 写 sys_reconciliation_fix_suggestion
//
// D-A4 触发条件(全部 AND):
//   - r.conflict_type = 'B'
//   - r.confidence_score >= threshold (默认 0.9,sys_config 可配 D-A3)
//   - r.workorder_id IS NULL
//   - r.resolved_at IS NULL
//   - r.deleted_at IS NULL
//   - NOT EXISTS(pending suggestion for this exception) — 避免重复
//
// D-A1:仅当 reconciliation_normalized.physical_user_id 非空时生成(物理链路无责场景)
//
// 紧急熔断(D-C5 / 紧急开关):sys_config asset.reconciliation.fix.enabled = 0 → 跳过
//
// 返回 (inserted, nil) 表示本轮生成条数
func (s *fixSuggestionServiceImpl) GenerateFixSuggestions(ctx context.Context) (int, error) {
	if s.db == nil {
		return 0, errors.New("db 未初始化")
	}

	// 1. 紧急熔断:读 sys_config enabled(0 → 返回 0, nil)
	if !s.isFixFeatureEnabled(ctx) {
		applogger.Infof("[fix-suggestion:Generator] 功能已禁用(enabled=0),跳过本轮")
		return 0, nil
	}

	// 2. 读 confidence_threshold(默认 0.9)
	threshold := s.getConfidenceThreshold(ctx)

	// 3. 查找候选 Type B 异常(D-A4 触发条件 + NOT EXISTS 重复保护)
	type candidate struct {
		ID              string
		AssetID         string
		ConfidenceScore float64
	}
	var candidates []candidate

	candidateSQL := `
SELECT r.id, r.asset_id, r.confidence_score
FROM sys_data_reconciliation r
WHERE r.conflict_type = 'B'
  AND r.confidence_score >= ?
  AND r.workorder_id IS NULL
  AND r.resolved_at IS NULL
  AND r.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM sys_reconciliation_fix_suggestion s
    WHERE s.exception_id = r.id
      AND s.fix_status = 'pending'
      AND s.superseded_at IS NULL
      AND s.deleted_at IS NULL
  )
`
	if err := s.db.WithContext(ctx).Raw(candidateSQL, threshold).Scan(&candidates).Error; err != nil {
		return 0, fmt.Errorf("查询候选异常失败: %w", err)
	}

	// 4. 逐条 INSERT(物理链路 user_id 通过 reconciliation_normalized JOIN 取 — D-A1 锁定)
	inserted := 0
	for _, c := range candidates {
		var physUserID *string
		row := s.db.WithContext(ctx).
			Table("reconciliation_normalized").
			Select("physical_user_id").
			Where("asset_id = ?", c.AssetID).
			Row()
		scanErr := row.Scan(&physUserID)
		if scanErr != nil && !errors.Is(scanErr, gorm.ErrRecordNotFound) {
			applogger.Warnf("[fix-suggestion:Generator] 查 physical_user_id 失败 asset_id=%s: %v", c.AssetID, scanErr)
			continue
		}

		if physUserID == nil || *physUserID == "" {
			// D-A1:仅当 physical_user_id 非空时生成建议
			// Type B 在 physical_user_id 为空时已归 D/E 类,不会进入 R5 触达
			continue
		}

		sugg := &models.SysReconciliationFixSuggestion{
			ExceptionID:     c.ID,
			SuggestedUserID: physUserID,
			ConfidenceScore: c.ConfidenceScore,
			Reason:          fmt.Sprintf("物理链路 user_id=%s, ops_asset 当前无责(Type B 触发)", *physUserID),
			FixStatus:       "pending",
			ConflictType:    "B",
		}
		if err := s.db.WithContext(ctx).Create(sugg).Error; err != nil {
			applogger.Warnf("[fix-suggestion:Generator] 生成建议失败 exception_id=%s: %v", c.ID, err)
			continue
		}
		inserted++
	}

	applogger.Infof("[fix-suggestion:Generator] 本轮生成 %d 条建议(threshold=%.2f, candidates=%d)", inserted, threshold, len(candidates))
	return inserted, nil
}

// isFixFeatureEnabled 读 sys_config asset.reconciliation.fix.enabled
//
// 默认 1(启用);0 = 紧急熔断
func (s *fixSuggestionServiceImpl) isFixFeatureEnabled(ctx context.Context) bool {
	if s.configService == nil {
		return true // 默认启用
	}
	cfg, err := s.configService.GetByKey(ctx, "asset.reconciliation.fix.enabled")
	if err != nil || cfg == nil {
		return true // 默认启用
	}
	return cfg.ConfigValue == "1" || cfg.ConfigValue == "true" || cfg.ConfigValue == "yes"
}

// getConfidenceThreshold 读 sys_config asset.reconciliation.fix.confidence_threshold
//
// 默认 0.9(D-A3)
func (s *fixSuggestionServiceImpl) getConfidenceThreshold(ctx context.Context) float64 {
	if s.configService == nil {
		return 0.9
	}
	cfg, err := s.configService.GetByKey(ctx, "asset.reconciliation.fix.confidence_threshold")
	if err != nil || cfg == nil {
		return 0.9
	}
	var v float64
	if _, err := fmt.Sscanf(cfg.ConfigValue, "%f", &v); err != nil {
		return 0.9
	}
	return v
}
