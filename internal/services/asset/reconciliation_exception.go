package asset

import (
	"context"
	"errors"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// ============================================================================
// R1 边界说明 — 资产对账例外规则服务(Phase 42 R1 plan 02 / 任务 1)
//
// 本文件实现 R1 范围:**仅 skeleton (只读 List + GetByID)**,不提供写操作。
//
// 完整 CRUD 推迟到 R3,原因:
//   - 例外规则涉及 CIDR 段匹配,R3 才接 sys_reconciliation_exception 的
//     GiST 索引 + 检测引擎消费(写入即需立刻生效,索引与缓存联合验证)
//   - 状态语义:is_active 字段约定遵循 Status Convention(0=启用,1=停用),
//     R3 写操作时还需联动 Layer 3 检测缓存(命中变更后失效)
//   - 写入校验复杂:reason ≥10 字符 + 跨规则冲突检查(scope_type + scope_id +
//     ip_range 区间不能完全覆盖已有规则)+ AD/global/dept/user 多种 scope 语义
//     需 service 层多步校验,R1 时机尚不成熟
//
// R1 skeleton 的目的:
//   - 占位接口,让上游 handler / UI 可在 R1 时观察已 seed 的例外规则
//   - 验证 service ↔ model ↔ table 三层连线 OK
//   - 为 R3 写操作铺垫(List/GetByID 是 Create/Update/Delete 的依赖)
//
// 参考:
//   - .planning/notes/260627-reconciliation-reuse-audit.md §4.6
//     (R1 只读 skeleton → R3 完整 CRUD 的演进路径)
//   - .planning/notes/asset-reconciliation-strategy.md §4
//     (例外规则在检测引擎中的执行语义与时序)
// ============================================================================

// ExceptionRuleListParams 例外规则列表查询参数
//
// 嵌入 base.BaseListRequest 自动获得 current / pageSize / orderByColumn / isAsc。
// IsActive 可选:传 0 / 1 过滤启停,不传则查全部(包含停用)。
type ExceptionRuleListParams struct {
	base.BaseListRequest

	IsActive *int `json:"isActive,omitempty"` // 状态过滤(0=启用, 1=停用,遵循 Status Convention)
}

// ReconciliationExceptionService 资产对账例外规则服务接口
//
// R1 范围:List(分页查询) + GetByID(单条详情)。
// 写操作(Create/Update/Delete/Enable/Disable)由 R3 实现。
type ReconciliationExceptionService interface {
	// List 列出例外规则(分页 + 状态过滤)
	List(ctx context.Context, params *ExceptionRuleListParams) (*base.PageResult, error)

	// GetByID 按主键查询单条规则
	GetByID(ctx context.Context, id string) (*models.SysReconciliationException, error)
}

// reconciliationExceptionServiceImpl ReconciliationExceptionService 私有实现
type reconciliationExceptionServiceImpl struct {
	db *gorm.DB
}

// NewReconciliationExceptionService 构造 ReconciliationExceptionService 实例
func NewReconciliationExceptionService(db *gorm.DB) ReconciliationExceptionService {
	return &reconciliationExceptionServiceImpl{db: db}
}

// List 列出例外规则
//
// 行为:
//  1. 校验 params 非 nil
//  2. 默认 current=1, pageSize=10
//  3. Count + Find
//  4. 过滤:is_active(若提供)
//  5. 默认排序:created_at DESC(操作员最关心"最新创建的规则")
//  6. 返回 *base.PageResult
func (s *reconciliationExceptionServiceImpl) List(ctx context.Context, params *ExceptionRuleListParams) (*base.PageResult, error) {
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
	query := s.db.WithContext(ctx).
		Model(&models.SysReconciliationException{}).
		Where("sys_reconciliation_exception.deleted_at IS NULL")

	// 过滤:is_active
	if params.IsActive != nil {
		query = query.Where("sys_reconciliation_exception.is_active = ?", *params.IsActive)
	}

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Find:默认 created_at DESC
	var list []models.SysReconciliationException
	offset := (current - 1) * pageSize
	if err := query.
		Order("sys_reconciliation_exception.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, err
	}

	return &base.PageResult{
		List:     list,
		Total:    total,
		Current:  current,
		PageSize: pageSize,
	}, nil
}

// GetByID 按主键查询单条例外规则
//
// 行为:
//   - 软删除兜底:deleted_at IS NULL
//   - 记录不存在返回 nil,nil(同 ReconciliationService.GetByID 的"未找到"语义)
func (s *reconciliationExceptionServiceImpl) GetByID(ctx context.Context, id string) (*models.SysReconciliationException, error) {
	var rule models.SysReconciliationException
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&rule).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}
