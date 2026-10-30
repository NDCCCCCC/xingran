package asset

// ============================================================================
// Phase 44 R3 / Plan 44-02 Task 2 — 资产对账降噪基线 service (D-R3-A4-01 + WARN-8)
//
// ⚠ 运维责任文档(BLOCKER-3 / SC 8 ≥60% 降噪量化验证前置):
// 运维必须在 R3 部署前 + R2 数据保留期内调用 Snapshot 记录 R2 末期基线,
// 否则 SC 8 ≥60% 降噪不可量化验证(Compare 端点会返回 400 "未找到基线快照")。
//
// 设计要点:
//   - 独立 .Count(&...) 3 个 COUNT 查询(total/totalWorkorders/critical),
//     严禁 ListExceptions(Pitfall 5: MaxPageSize=100 钳制,
//     项目记忆 stat-cards-from-list-length-capped-at-100)
//   - Snapshot COUNT 含 silence 记录(WARN-8):WHERE 仅 deleted_at IS NULL,
//     不加 silence 过滤(silence 是降噪手段应计入"当前告警量"基准,
//     与 ListExceptions UI 默认隐藏 silence 的运维视图偏好解耦)
//   - 基线快照写 sys_config(config_key=asset.reconciliation.baseline, JSON 序列化),
//     GetByKey 存在则 Update,不存在则 Create(幂等覆盖)
//   - Compare 读 baseline JSON + 当前独立 COUNT → 算下降%, 无 baseline 返回 error
//
// 接口与 handler 解耦:本 service 只做 sys_config 读写 + COUNT,不调 operlog(由 handler 负责)。
// ============================================================================

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// BaselineConfigKey sys_config 中存 baseline 的 config_key(全局唯一)
const BaselineConfigKey = "asset.reconciliation.baseline"

// BaselineSnapshot 基线快照(单个时刻的告警量快照)
type BaselineSnapshot struct {
	SnapshotAt         time.Time `json:"snapshot_at"`
	TotalExceptions    int64     `json:"total_exceptions"`
	TotalWorkorders    int64     `json:"total_workorders"`
	CriticalExceptions int64     `json:"critical_exceptions"`
}

// BaselineCompareResult 对比结果(baseline vs current + 3 个下降%)
type BaselineCompareResult struct {
	Baseline               BaselineSnapshot `json:"baseline"`
	Current                BaselineSnapshot `json:"current"`
	ExceptionsReductionPct float64          `json:"exceptions_reduction_pct"`
	WorkordersReductionPct float64          `json:"workorders_reduction_pct"`
	CriticalReductionPct   float64          `json:"critical_reduction_pct"`
}

// ReconciliationBaselineService 资产对账降噪基线服务接口
type ReconciliationBaselineService interface {
	// Snapshot 记录当前为基线,写 sys_config(幂等覆盖)
	//
	// 运维责任:R3 部署前 + R2 数据保留期内必须调用本方法记录 R2 末期基线,
	// 否则后续 Compare 无法量化降噪效果(SC 8 不可验证)。
	Snapshot(ctx context.Context) (*BaselineSnapshot, error)

	// Compare 对比当前与 baseline,返回下降百分比
	//
	// 无 baseline 时返回 error,引导运维先调用 Snapshot。
	Compare(ctx context.Context) (*BaselineCompareResult, error)
}

// reconciliationBaselineServiceImpl 私有实现
//
// 不依赖 system.ConfigService(避免 import 循环 + 保持 service 层 ctx-agnostic),
// 直接 SQL 读写 sys_config(GORM Table("sys_config"))。
type reconciliationBaselineServiceImpl struct {
	db *gorm.DB
}

// NewReconciliationBaselineService 构造 baseline service 实例
func NewReconciliationBaselineService(db *gorm.DB) ReconciliationBaselineService {
	return &reconciliationBaselineServiceImpl{db: db}
}

// Snapshot 记录当前为基线
//
// 行为:
//   - 独立 COUNT 当前快照(3 个查询:total / totalWorkorders / critical)
//   - COUNT WHERE 仅 deleted_at IS NULL,**不加** silence 过滤(WARN-8: 含 silence 记录)
//   - 序列化 JSON 写 sys_config:先 GetByKey,存在则 Update config_value,不存在则 Create
//   - 幂等:二次 Snapshot 覆盖现有 baseline(key 唯一)
//
// 返回:BaselineSnapshot(写库成功后)
func (s *reconciliationBaselineServiceImpl) Snapshot(ctx context.Context) (*BaselineSnapshot, error) {
	snap, err := s.countCurrent(ctx)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(snap)
	if err != nil {
		return nil, fmt.Errorf("序列化 baseline 失败: %w", err)
	}

	// 读现有 baseline(GetByKey 模式)。
	//
	// 用 Limit(1) + Scan 到 []string(而非 Pluck 到 string),避免空结果时
	// GORM 返回 "sql: Scan error converting NULL to string"(Pluck 对 0 行结果
	// 在 PG/SQLite 都会触发 Scan NULL 错误)。
	var existingIDs []string
	if err := s.db.WithContext(ctx).
		Table("sys_config").
		Where("config_key = ? AND deleted_at IS NULL", BaselineConfigKey).
		Limit(1).
		Pluck("id", &existingIDs).Error; err != nil {
		return nil, fmt.Errorf("查询 baseline 失败: %w", err)
	}

	if len(existingIDs) > 0 && existingIDs[0] != "" {
		// 存在 → Update config_value(SnapshotAt 重新刷新)
		if err := s.db.WithContext(ctx).
			Table("sys_config").
			Where("id = ?", existingIDs[0]).
			Update("config_value", string(data)).Error; err != nil {
			return nil, fmt.Errorf("更新 baseline 失败: %w", err)
		}
	} else {
		// 不存在 → Create(config_type='Y' 表示是字符串配置;config_name 用运维可读名称)
		// id 显式生成 uuid.NewString()(SQLite/PG 一致:PG 生产用 gen_random_uuid() default,
		// 但 GORM Table("sys_config").Create(map) 不触发 model default tag,需手动赋值)
		newConfig := map[string]interface{}{
			"id":           uuid.NewString(),
			"config_name":  "资产对账降噪基线",
			"config_key":   BaselineConfigKey,
			"config_value": string(data),
			"config_type":  "Y",
			"is_system":    1, // 系统内置(防误删,与 sys.request.encryption.enabled 一致)
			"created_at":   time.Now(),
			"updated_at":   time.Now(),
		}
		if err := s.db.WithContext(ctx).Table("sys_config").Create(&newConfig).Error; err != nil {
			return nil, fmt.Errorf("创建 baseline 失败: %w", err)
		}
	}

	return snap, nil
}

// Compare 对比 baseline 与当前,返回下降%
//
// 行为:
//   - 读 baseline JSON(GetByKey),不存在返回 error "未找到基线快照,请先调用 Snapshot 记录基线"
//   - 反序列化 → BaselineSnapshot
//   - 独立 COUNT 当前(同 Snapshot 逻辑,含 silence)
//   - 算下降%:pct = (baseline - current) / baseline * 100 (baseline=0 时返回 0)
func (s *reconciliationBaselineServiceImpl) Compare(ctx context.Context) (*BaselineCompareResult, error) {
	// 读 baseline
	var configValue string
	err := s.db.WithContext(ctx).
		Table("sys_config").
		Where("config_key = ? AND deleted_at IS NULL", BaselineConfigKey).
		Pluck("config_value", &configValue).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("未找到基线快照,请先调用 Snapshot 记录基线")
		}
		return nil, fmt.Errorf("查询 baseline 失败: %w", err)
	}
	if configValue == "" {
		return nil, errors.New("未找到基线快照,请先调用 Snapshot 记录基线")
	}

	var baseline BaselineSnapshot
	if err := json.Unmarshal([]byte(configValue), &baseline); err != nil {
		return nil, fmt.Errorf("解析 baseline JSON 失败: %w", err)
	}

	// COUNT 当前
	current, err := s.countCurrent(ctx)
	if err != nil {
		return nil, err
	}

	// 算下降%
	pct := func(b, c int64) float64 {
		if b == 0 {
			return 0
		}
		return float64(b-c) / float64(b) * 100
	}

	return &BaselineCompareResult{
		Baseline:               baseline,
		Current:                *current,
		ExceptionsReductionPct: pct(baseline.TotalExceptions, current.TotalExceptions),
		WorkordersReductionPct: pct(baseline.TotalWorkorders, current.TotalWorkorders),
		CriticalReductionPct:   pct(baseline.CriticalExceptions, current.CriticalExceptions),
	}, nil
}

// countCurrent 独立 COUNT 当前告警量(3 个查询,WHERE 仅 deleted_at IS NULL,含 silence — WARN-8)
//
// 严禁:
//   - ListExceptions(Pitfall 5: MaxPageSize=100 钳制)
//   - WHERE 加 silence 过滤(WARN-8: silence 是降噪手段应计入基准)
func (s *reconciliationBaselineServiceImpl) countCurrent(ctx context.Context) (*BaselineSnapshot, error) {
	var totalExceptions int64
	if err := s.db.WithContext(ctx).
		Model(&models.SysDataReconciliation{}).
		Where("deleted_at IS NULL").
		Count(&totalExceptions).Error; err != nil {
		return nil, fmt.Errorf("统计总异常数失败: %w", err)
	}

	var totalWorkorders int64
	if err := s.db.WithContext(ctx).
		Model(&models.SysDataReconciliation{}).
		Where("deleted_at IS NULL AND workorder_id IS NOT NULL").
		Count(&totalWorkorders).Error; err != nil {
		return nil, fmt.Errorf("统计工单数失败: %w", err)
	}

	var criticalExceptions int64
	if err := s.db.WithContext(ctx).
		Model(&models.SysDataReconciliation{}).
		Where("deleted_at IS NULL AND severity = ?", "critical").
		Count(&criticalExceptions).Error; err != nil {
		return nil, fmt.Errorf("统计 critical 异常数失败: %w", err)
	}

	return &BaselineSnapshot{
		SnapshotAt:         time.Now(),
		TotalExceptions:    totalExceptions,
		TotalWorkorders:    totalWorkorders,
		CriticalExceptions: criticalExceptions,
	}, nil
}
