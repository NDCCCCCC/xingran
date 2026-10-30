package addomain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

type DeptToADSyncService struct {
	db     *gorm.DB
	pool   AccountPool // Phase 38 Wave 1 DI 脚手架（38-02 将用于 FailoverClient 闭包改造）
	mapper *DeptOUmapper
}

// NewDeptToADSyncService 创建部门→AD OU 同步服务。
// 域控审查 P2-3：移除从未使用的 ldapClient 参数（LDAP 连接统一由 FailoverClient 在
// ExecuteWithFailover 闭包内注入，服务实例不持有连接）。
func NewDeptToADSyncService(db *gorm.DB, pool AccountPool, mapper *DeptOUmapper) *DeptToADSyncService {
	return &DeptToADSyncService{
		db:     db,
		pool:   pool,
		mapper: mapper,
	}
}

// SyncDeptStructureToAD 同步部门结构到AD OU
func (s *DeptToADSyncService) SyncDeptStructureToAD(ctx context.Context, adConfigID string) (*DeptSyncResult, error) {
	result := &DeptSyncResult{
		StartTime:  time.Now(),
		ADConfigID: adConfigID,
		Status:     "running",
	}

	// 1. 获取AD配置
	var adConfig models.ADConfig
	if err := s.db.WithContext(ctx).Where("id = ?", adConfigID).First(&adConfig).Error; err != nil {
		result.Status = "failed"
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return nil, fmt.Errorf("获取AD配置失败: %w", err)
	}

	// 2. 连接LDAP（Phase 38 Wave 1: 改走 FailoverClient 账号池故障切换）
	//    operation 边界 = 整个同步流程（递归 syncDeptTree 整棵树一个 operation，SP-3）

	// 3. 获取根部门（parentId为空的部门）— DB 操作，不依赖 LDAP 连接，可放闭包外
	rootDepts, err := s.getRootDepartments(ctx)
	if err != nil {
		result.Status = "failed"
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return nil, fmt.Errorf("获取根部门失败: %w", err)
	}

	// 4. 跳过顶层部门，直接从二级部门开始同步
	// 收集所有二级部门（根部门的直接子部门）
	var secondLevelDepts []*models.Department
	for _, root := range rootDepts {
		for _, child := range root.Children {
			secondLevelDepts = append(secondLevelDepts, child)
		}
	}

	if len(secondLevelDepts) == 0 {
		applogger.Warnf("未找到二级部门，跳过同步")
		result.Status = "completed"
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, nil
	}

	result.TotalDepts = s.countTotalDepts(secondLevelDepts)

	fc := NewFailoverClient(s.pool, &adConfig)
	if err := fc.ExecuteWithFailover(ctx, func(ldapClient LDAPClientIface) error {
		// 5. 递归同步二级部门树到AD OU（所有 LDAP CreateOU 在闭包内，Pitfall 3）
		// 域控审查 P1-4：单树失败不再 break 中断其余树，也不再把整任务置为 failed ——
		// 单树失败计 result.FailedDepts，其余树照常同步；任务整体成败由 FailedDepts 体现。
		for _, dept := range secondLevelDepts {
			if err := s.syncDeptTree(ctx, ldapClient, &adConfig, dept, adConfig.BaseDN, result); err != nil {
				applogger.Errorf("同步部门树失败 [%s]，继续处理其余部门树: %v", dept.DeptName, err)
			}
		}
		return nil
	}); err != nil {
		result.Status = "failed"
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return nil, fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return nil, fmt.Errorf("连接AD失败: %w", err)
	}

	// 5. 更新结果
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	if result.Status == "running" {
		result.Status = "completed"
	}

	applogger.Infof("部门同步完成: 总数=%d, 成功=%d, 失败=%d, 跳过=%d, 耗时=%dms",
		result.TotalDepts, result.SuccessDepts, result.FailedDepts, result.SkippedDepts, result.Duration.Milliseconds())

	return result, nil
}

// syncDeptTree 递归同步部门树
//
// Phase 58 HYGIENE-02: OU DN 拼接走 BuildOUDN（全仓唯一拼接点），部门名特殊字符按 RFC4514 转义。
// 存量 sys_dept_ou_mapping.ou_dn 为未转义值，含特殊字符的部门需重跑部门同步后才会写入转义 DN
// （Q4 决策：不做存量数据迁移）。
func (s *DeptToADSyncService) syncDeptTree(ctx context.Context, ldapClient LDAPClientIface, config *models.ADConfig, dept *models.Department, parentOUDN string, result *DeptSyncResult) error {
	// 1. 构建当前部门的OU DN
	ouDN := BuildOUDN(dept.DeptName, parentOUDN)

	// 2. 在AD中创建OU（如果不存在）
	// 域控审查 P1-4 说明：CreateOU 失败时必须 return —— AD 的 DN 层级要求父 OU
	// 真实存在，父 OU 建不出来时子部门必然逐个失败（产生 N 个重复错误），此处提前
	// 止损只报一条错误；调用方（SyncDeptStructureToAD）已改为 continue 其余部门树。
	if err := ldapClient.CreateOU(ouDN, dept.DeptName); err != nil {
		result.FailedDepts++
		result.Errors = append(result.Errors, DeptSyncError{
			DeptID:   dept.ID,
			DeptName: dept.DeptName,
			Error:    err.Error(),
		})
		return fmt.Errorf("创建OU失败 [%s]: %w", dept.DeptName, err)
	}

	// 3. 更新映射关系
	mapping := &models.DeptOUMapping{
		DeptID:      dept.ID,
		ADConfigID:  config.ID,
		OUDN:        ouDN,
		OUName:      dept.DeptName,
		ParentOUDN:  &parentOUDN,
		SyncEnabled: true,
		SyncStatus:  "synced",
	}

	now := time.Now()
	mapping.LastSyncAt = &now

	if err := s.mapper.UpsertMapping(ctx, mapping); err != nil {
		var conflict *OUDNConflictError
		if errors.As(err, &conflict) {
			// HYGIENE-05:同名兄弟部门争用同一 OU DN —— 显式计失败 + Errorf 双部门名，不静默覆盖
			applogger.Errorf("部门 OU DN 冲突: 部门 [%s] 与部门 [%s] 争用同一 OU DN %s，映射保留给 [%s]，本次不同步该部门",
				conflict.IncomingDeptName, conflict.ExistingDeptName, conflict.OUDN, conflict.ExistingDeptName)
			result.FailedDepts++
			result.Errors = append(result.Errors, DeptSyncError{DeptID: dept.ID, DeptName: dept.DeptName, Error: conflict.Error()})
		} else {
			applogger.Warnf("更新映射关系失败 [%s]: %v", dept.DeptName, err)
		}
		// 不中断同步流程，继续处理子部门
	} else {
		result.SuccessDepts++
	}

	// 4. 递归处理子部门
	if len(dept.Children) > 0 {
		for _, child := range dept.Children {
			if err := s.syncDeptTree(ctx, ldapClient, config, child, ouDN, result); err != nil {
				applogger.Errorf("同步子部门失败 [%s]: %v", child.DeptName, err)
				// 继续处理其他子部门
			}
		}
	}

	return nil
}

// getRootDepartments 获取根部门（parentId为空的部门）及其完整子树。
//
// 域控审查 P1-7：原实现 `Preload("Children.Children.Children")` 只加载固定 3 层，
// 第 4 层及更深的部门被静默排除在同步之外（且子树计数也少算）。改为一次查询全部
// 正常部门后在内存中按 parent_id 建树：任何深度都能覆盖，查询也从 N 层 N 次
// Preload 降为 1 次全表查询。
func (s *DeptToADSyncService) getRootDepartments(ctx context.Context) ([]*models.Department, error) {
	var all []*models.Department
	if err := s.db.WithContext(ctx).
		Select("id", "dept_name", "parent_id", "ancestors").
		Where("status = ?", models.DeptStatusNormal).
		Find(&all).Error; err != nil {
		return nil, err
	}

	// 先为每个部门建一个独立节点（Children 清空，避免复用 Preload 残留）
	nodes := make(map[string]*models.Department, len(all))
	for _, d := range all {
		node := *d
		node.Children = nil
		nodes[d.ID] = &node
	}

	// 按 parent_id 挂接：父不在集合中（父被停用/删除）的部门成为孤儿，
	// 不挂在任何根下 —— 与原 Preload 语义一致（停用父的子部门不同步）。
	var roots []*models.Department
	for _, node := range nodes {
		if node.ParentID == nil || *node.ParentID == "" {
			roots = append(roots, node)
			continue
		}
		if parent, ok := nodes[*node.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		}
	}
	return roots, nil
}

// countTotalDepts 递归计算部门总数
func (s *DeptToADSyncService) countTotalDepts(depts []*models.Department) int {
	count := len(depts)
	for _, dept := range depts {
		if len(dept.Children) > 0 {
			count += s.countTotalDepts(dept.Children)
		}
	}
	return count
}
