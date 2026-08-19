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
	ldap   *LDAPClient
	mapper *DeptOUmapper
}

func NewDeptToADSyncService(db *gorm.DB, pool AccountPool, ldapClient *LDAPClient, mapper *DeptOUmapper) *DeptToADSyncService {
	return &DeptToADSyncService{
		db:     db,
		pool:   pool,
		ldap:   ldapClient,
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
	if err := fc.ExecuteWithFailover(ctx, func(ldapClient *LDAPClient) error {
		// 5. 递归同步二级部门树到AD OU（所有 LDAP CreateOU 在闭包内，Pitfall 3）
		for _, dept := range secondLevelDepts {
			if err := s.syncDeptTree(ctx, ldapClient, &adConfig, dept, adConfig.BaseDN, result); err != nil {
				applogger.Errorf("同步部门树失败: %v", err)
				result.Status = "failed"
				break
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
func (s *DeptToADSyncService) syncDeptTree(ctx context.Context, ldapClient *LDAPClient, config *models.ADConfig, dept *models.Department, parentOUDN string, result *DeptSyncResult) error {
	// 1. 构建当前部门的OU DN
	ouDN := fmt.Sprintf("OU=%s,%s", dept.DeptName, parentOUDN)

	// 2. 在AD中创建OU（如果不存在）
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
		applogger.Warnf("更新映射关系失败 [%s]: %v", dept.DeptName, err)
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

// getRootDepartments 获取根部门（parentId为空的部门）
func (s *DeptToADSyncService) getRootDepartments(ctx context.Context) ([]*models.Department, error) {
	var depts []*models.Department
	err := s.db.WithContext(ctx).
		Preload("Children.Children.Children"). // 预加载3层子部门
		Where("parent_id IS NULL").            // UUID字段不能使用空字符串比较
		Where("status = ?", models.DeptStatusNormal).
		Find(&depts).Error
	return depts, err
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
