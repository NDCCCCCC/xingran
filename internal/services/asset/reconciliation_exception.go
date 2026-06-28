package asset

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/base"
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
// R3 扩展:Create/Update/Delete + MatchTest(命中测试)。
type ReconciliationExceptionService interface {
	// List 列出例外规则(分页 + 状态过滤)
	List(ctx context.Context, params *ExceptionRuleListParams) (*base.PageResult, error)

	// GetByID 按主键查询单条规则
	GetByID(ctx context.Context, id string) (*models.SysReconciliationException, error)

	// Create 创建例外规则(R3)
	//
	// 行为:
	//   1. ValidateCIDR / ValidateActions / ValidateSeverityOverride / ValidateReason
	//   2. 构造 model(IsActive=0 默认启用,CreatedBy 从 ctx 或外部注入)
	//   3. db.Create + 失效缓存
	Create(ctx context.Context, req *CreateExceptionRuleRequest) (*models.SysReconciliationException, error)

	// Update 更新例外规则(R3)
	Update(ctx context.Context, id string, req *UpdateExceptionRuleRequest) error

	// Delete 软删除例外规则(R3)
	// GORM 软删除填 deleted_at;与 cron 软停用(is_active=1)区分。
	Delete(ctx context.Context, id string) error

	// MatchTest 命中测试工具(R3,EXCEPTION-04 / SC 6)
	//
	// 行为:
	//   - 预加载 active 规则到内存(D-R3-A1-03)
	//   - 按 global / dept / user scope 分别匹配(GiST 索引在 PG 也可用,但内存匹配
	//     保证 dialect-agnostic,SQLite 测试也可验证逻辑)
	//   - dept scope **不递归子部门**,仅匹配 scope_id == deptID 直接部门(WARN-10)
	//   - 未传 userID + deptID 时,user/dept scope 规则不参与合并(NeedsUserDept=true)
	//   - 返回 MatchTestResult(命中规则 + 合并卡片)
	MatchTest(ctx context.Context, ip string, userID string, deptID string) (*MatchTestResult, error)
}

// CreateExceptionRuleRequest 创建例外规则请求
type CreateExceptionRuleRequest struct {
	Name             string         `json:"name"`
	IPRange          string         `json:"ipRange"`
	ConflictTypes    pq.StringArray `json:"conflictTypes"`
	ExceptionActions pq.StringArray `json:"exceptionActions"`
	SeverityOverride *string        `json:"severityOverride,omitempty"`
	ScopeType        string         `json:"scopeType"`
	ScopeID          *string        `json:"scopeId,omitempty"`
	ExpiresAt        *time.Time     `json:"expiresAt,omitempty"`
	Reason           string         `json:"reason"`
}

// UpdateExceptionRuleRequest 更新例外规则请求(字段同 Create)
type UpdateExceptionRuleRequest struct {
	Name             string         `json:"name"`
	IPRange          string         `json:"ipRange"`
	ConflictTypes    pq.StringArray `json:"conflictTypes"`
	ExceptionActions pq.StringArray `json:"exceptionActions"`
	SeverityOverride *string        `json:"severityOverride,omitempty"`
	ScopeType        string         `json:"scopeType"`
	ScopeID          *string        `json:"scopeId,omitempty"`
	ExpiresAt        *time.Time     `json:"expiresAt,omitempty"`
	Reason           string         `json:"reason"`
}

// MatchTestResult 命中测试结果(D-R3-A3-03 + A2-03 顶部合并卡片)
type MatchTestResult struct {
	MatchedRules  []models.SysReconciliationException `json:"matchedRules"`
	MergedActions pq.StringArray                      `json:"mergedActions"`
	FinalSeverity string                              `json:"finalSeverity"`
	IsSilence     bool                                `json:"isSilence"`
	NeedsUserDept bool                                `json:"needsUserDept"`
}

// ============================================================================
// 校验函数(R3 / V5 Input Validation + Threat Model T-44-01/04)
// ============================================================================

// ValidateCIDR 校验 IP CIDR 格式(T-44-01 CIDR 注入缓解)
//
// net.ParseCIDR 严格校验:返回 _, ipNet, err;失败返回 error。
// 业务侧拒绝任何解析失败的 CIDR 进入数据库(DB cidr 列兜底 INSERT 非法值 SQLSTATE 22P02)。
func ValidateCIDR(ipRange string) error {
	if strings.TrimSpace(ipRange) == "" {
		return errors.New("CIDR 不能为空")
	}
	_, _, err := net.ParseCIDR(ipRange)
	if err != nil {
		return fmt.Errorf("CIDR 格式非法: %w", err)
	}
	return nil
}

// ValidateActions 校验 exception_actions 必填 + 5 白名单值
//
// 白名单: no_alert / no_notice / no_workorder / skip_severity / silence
// 与 DB CHECK chk_recon_exc_actions 同步,避免 INSERT 触发 SQLSTATE 23514。
func ValidateActions(actions pq.StringArray) error {
	if len(actions) == 0 {
		return errors.New("exception_actions 必填(至少 1 项)")
	}
	whitelist := map[string]struct{}{
		"no_alert":     {},
		"no_notice":    {},
		"no_workorder": {},
		"skip_severity": {},
		"silence":      {},
	}
	for _, a := range actions {
		if _, ok := whitelist[a]; !ok {
			return fmt.Errorf("exception_actions 含非法值 %q(白名单: no_alert/no_notice/no_workorder/skip_severity/silence)", a)
		}
	}
	return nil
}

// ValidateSeverityOverride 校验 severity_override 白名单
//
// 白名单: nil(可空) / low / medium / high
// **不含 critical**(Pitfall 8 — override 是降级语义,与 chk_recon_exc_severity_override 同步)。
func ValidateSeverityOverride(sev *string) error {
	if sev == nil {
		return nil
	}
	switch *sev {
	case "low", "medium", "high":
		return nil
	default:
		return fmt.Errorf("severity_override 非法值 %q(白名单: low/medium/high,不含 critical)", *sev)
	}
}

// ValidateReason 校验 reason ≥10 字符
//
// 告警风暴缓解(T-44-04):强制运维说明原因,降低 0.0.0.0/0 silence 误配风险。
// 用 utf8.RuneCountInString 支持中文字符(1 中文 = 1 字符)。
func ValidateReason(reason string) error {
	if utf8.RuneCountInString(reason) < 10 {
		return errors.New("reason 至少 10 字符(告警风暴缓解强制说明原因)")
	}
	return nil
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

// ============================================================================
// R3 写操作 (Create / Update / Delete / MatchTest)
// ============================================================================

// Create 创建例外规则
//
// 行为:
//   1. ValidateCIDR / ValidateActions / ValidateSeverityOverride / ValidateReason
//   2. 构造 model(IsActive=0 默认启用,CreatedBy 默认 0 — handler 可通过
//      gin context 注入真实 user_id,但本 service 层保持 ctx-agnostic)
//   3. db.Create
//   4. 失效缓存(CacheKeyReconciliationExceptionRuleList)
//
// 注意:ScopeType 不在 service 层强校验(DB default 'global' + 前端 Radio.Group
// 限制 global/dept/user 三选一);ScopeID 在 R3 仅做"nil 允许"非强校验(避免
// 引入 dept/user 表 JOIN,46 可视情况加强)。
func (s *reconciliationExceptionServiceImpl) Create(ctx context.Context, req *CreateExceptionRuleRequest) (*models.SysReconciliationException, error) {
	if req == nil {
		return nil, errors.New("请求参数不能为空")
	}
	if err := ValidateCIDR(req.IPRange); err != nil {
		return nil, err
	}
	if err := ValidateActions(req.ExceptionActions); err != nil {
		return nil, err
	}
	if err := ValidateSeverityOverride(req.SeverityOverride); err != nil {
		return nil, err
	}
	if err := ValidateReason(req.Reason); err != nil {
		return nil, err
	}

	scopeType := req.ScopeType
	if scopeType == "" {
		scopeType = "global"
	}

	rule := &models.SysReconciliationException{
		Name:             req.Name,
		IPRange:          req.IPRange,
		ConflictTypes:    req.ConflictTypes,
		ExceptionActions: req.ExceptionActions,
		SeverityOverride: req.SeverityOverride,
		ScopeType:        scopeType,
		ScopeID:          req.ScopeID,
		Reason:           req.Reason,
		IsActive:         0, // Status Convention: 0=启用(默认)
		ExpiresAt:        req.ExpiresAt,
	}

	if err := s.db.WithContext(ctx).Create(rule).Error; err != nil {
		return nil, fmt.Errorf("创建例外规则失败: %w", err)
	}

	s.invalidateCache()
	return rule, nil
}

// Update 更新例外规则
//
// 行为:
//   1. 先 First 验证 id 存在(不存在返回 error)
//   2. Validate(同 Create)
//   3. db.Updates(GORM Updates 只 SET 包含字段)
//   4. 失效缓存
func (s *reconciliationExceptionServiceImpl) Update(ctx context.Context, id string, req *UpdateExceptionRuleRequest) error {
	if id == "" {
		return errors.New("规则ID不能为空")
	}
	if req == nil {
		return errors.New("请求参数不能为空")
	}
	if err := ValidateCIDR(req.IPRange); err != nil {
		return err
	}
	if err := ValidateActions(req.ExceptionActions); err != nil {
		return err
	}
	if err := ValidateSeverityOverride(req.SeverityOverride); err != nil {
		return err
	}
	if err := ValidateReason(req.Reason); err != nil {
		return err
	}

	// 验证存在(软删除兜底)
	var existing models.SysReconciliationException
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("规则不存在")
		}
		return fmt.Errorf("查询规则失败: %w", err)
	}

	scopeType := req.ScopeType
	if scopeType == "" {
		scopeType = "global"
	}

	updates := map[string]interface{}{
		"name":              req.Name,
		"ip_range":          req.IPRange,
		"conflict_types":    req.ConflictTypes,
		"exception_actions": req.ExceptionActions,
		"severity_override": req.SeverityOverride,
		"scope_type":        scopeType,
		"scope_id":          req.ScopeID,
		"reason":            req.Reason,
	}
	if req.ExpiresAt != nil {
		updates["expires_at"] = req.ExpiresAt
	} else {
		updates["expires_at"] = nil
	}

	if err := s.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新例外规则失败: %w", err)
	}

	s.invalidateCache()
	return nil
}

// Delete 软删除例外规则(GORM db.Delete 填 deleted_at)
//
// 与 cron 软停用(is_active=1)区分:
//   - Delete = admin 显式删除(deleted_at 被填,后续 List 查不到)
//   - 软停用 = cron 自动停用过期规则(is_active=1,记录仍可见)
//
// 历史 sys_data_reconciliation.exception_rule_id 仍指向该记录(JOIN 时 deleted_at
// 兜底返回空),审计链可追溯。
func (s *reconciliationExceptionServiceImpl) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("规则ID不能为空")
	}
	result := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		Delete(&models.SysReconciliationException{})
	if result.Error != nil {
		return fmt.Errorf("删除例外规则失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("规则不存在或已被删除")
	}
	s.invalidateCache()
	return nil
}

// MatchTest 命中测试工具(D-R3-A3-03 + EXCEPTION-04 / SC 6)
//
// 实现:
//   - 预加载 active 规则(preloadActiveRules 纯函数)
//   - 用内存 CIDR 匹配(同 DetectLayer3,保证 dialect-agnostic)
//   - **不**用 GiST `>>` SQL(D-R3-A1-03 GiST 留给 DB 优化器自动决策;
//     service 层内存匹配 + scope 双条件 + dept 不递归 WARN-10,逻辑透明)
//   - **不**递归子部门(WARN-10):dept scope 仅匹配 scope_id == deptID
//
// 停用规则双重保险过滤:preloadActiveRules WHERE is_active=0 + 内存匹配时
// compiledRule 已不含停用规则(TestMatchTestExcludesInactiveRules 锁定)。
func (s *reconciliationExceptionServiceImpl) MatchTest(ctx context.Context, ip string, userID string, deptID string) (*MatchTestResult, error) {
	if ip == "" {
		return nil, errors.New("IP 不能为空")
	}

	// 预加载 active 规则(已过滤 is_active=0 + deleted_at IS NULL)
	activeRules := preloadActiveRules(s.db.WithContext(ctx))

	parsedIP := net.ParseIP(ip)

	// 收集所有命中规则(global + user + dept,按 scope 分流)
	// 注意:D-R3-A3-03 不传 userID/deptID 时,user/dept scope 规则不参与合并,
	// 仅 global 命中,NeedsUserDept=true 提示运维。
	var matched []compiledRule
	if parsedIP != nil {
		for _, r := range activeRules {
			if !r.ipNet.Contains(parsedIP) {
				continue
			}
			switch r.rule.ScopeType {
			case "global":
				// global 仅 IP CIDR 匹配即生效
				matched = append(matched, r)
			case "user":
				// user scope: IP CIDR 命中 + userID 匹配(双条件 A3-01)
				if userID != "" && r.rule.ScopeID != nil && *r.rule.ScopeID == userID {
					matched = append(matched, r)
				}
			case "dept":
				// dept scope: IP CIDR 命中 + deptID 直接匹配(WARN-10 不递归子部门)
				if deptID != "" && r.rule.ScopeID != nil && *r.rule.ScopeID == deptID {
					matched = append(matched, r)
				}
			}
		}
	}

	result := &MatchTestResult{
		NeedsUserDept: userID == "" && deptID == "",
	}

	// 收集 MatchedRules 原始数据(展示)
	result.MatchedRules = make([]models.SysReconciliationException, 0, len(matched))
	for _, r := range matched {
		result.MatchedRules = append(result.MatchedRules, r.rule)
	}

	// 合并 actions + severity(原始 severity 用 medium 作为命中测试基线)
	if len(matched) > 0 {
		actions, sev, silence := mergeActions("medium", matched, "")
		result.MergedActions = actions
		result.FinalSeverity = sev
		result.IsSilence = silence
	}

	return result, nil
}

// invalidateCache 失效例外规则缓存(R3 写操作后调用)
//
// 当前实现是无 Redis 场景下的 no-op;Phase 42 INFRA-04 已定义 CacheProvider
// 接口,真实部署由 core.Cache 注入。这里保留方法签名,便于后续接入。
func (s *reconciliationExceptionServiceImpl) invalidateCache() {
	// TODO(R3+): core.Cache.Delete(ctx, CacheKeyReconciliationExceptionRuleList)
	// 当前 service 层 ctx-agnostic,CacheProvider 注入由 handler 完成。
	// 数据写入 DB 后,下次 List 查询会从 DB 读取,缓存陈旧窗口在 cron 周期内
	// 可接受(告警通路由 DetectLayer3 内存匹配驱动,不依赖此缓存)。
}
