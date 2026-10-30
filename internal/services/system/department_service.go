package system

import (
	"context"
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/pkg/constants"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

type DepartmentService interface {
	Create(ctx context.Context, req *requests.DepartmentCreateRequest) error
	Update(ctx context.Context, req *requests.DepartmentUpdateRequest) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.Department, error)
	GetTree(ctx context.Context, includeDisabled bool) ([]*models.Department, error)
	GetTreeWithFilter(ctx context.Context, includeDisabled bool, params requests.DepartmentListParams) ([]*models.Department, error)
	List(ctx context.Context, params requests.DepartmentListParams) ([]models.Department, error)
	BatchDelete(ctx context.Context, ids []string) error
	UpdateStatus(ctx context.Context, id string, status int) error
	GetRoleDeptIDs(ctx context.Context, roleID string, deptIDs *[]string) error
	GetDB() *gorm.DB

	GetTreeWithCache(ctx context.Context, includeDisabled bool) ([]*models.Department, error)
	GetSelectDataWithCache(ctx context.Context) ([]*models.Department, error)
	InvalidateDeptCache(ctx context.Context) error
}

type departmentService struct {
	db *gorm.DB
	// adCascade 部门删除/改名 → AD OU 级联（域控审查 P0-2）。nil 时跳过级联
	// （单测/未启用 AD 场景），生产由 router 装配处 WithADCascade 注入。
	adCascade *addomain.DeptADCascadeService
}

func NewDepartmentService(db *gorm.DB) DepartmentService {
	return &departmentService{db: db}
}

// SetADCascade 注入 AD OU 级联服务。
// 生产由 router 装配处通过匿名接口断言注入（缓存包装 departmentCacheService 内嵌
// *departmentService，方法集透传）；单测不注入则跳过级联。
func (s *departmentService) SetADCascade(cascade *addomain.DeptADCascadeService) {
	s.adCascade = cascade
}

func (s *departmentService) Create(ctx context.Context, req *requests.DepartmentCreateRequest) error {
	ancestors, err := s.buildAncestors(ctx, req.ParentID)
	if err != nil {
		return fmt.Errorf("构建祖先链失败: %w", err)
	}

	if exists, err := s.checkDeptNameExists(ctx, req.ParentID, req.DeptName, ""); err != nil {
		return fmt.Errorf("检查部门名称失败: %w", err)
	} else if exists {
		return fmt.Errorf("同级部门名称已存在")
	}

	// 检查部门编码是否已存在
	if req.DeptCode != "" {
		if exists, err := s.checkDeptCodeExists(ctx, req.DeptCode, ""); err != nil {
			return fmt.Errorf("检查部门编码失败: %w", err)
		} else if exists {
			return fmt.Errorf("部门编码已存在")
		}
	}

	dept := req.ToModel(ancestors)

	if err := s.db.WithContext(ctx).Create(&dept).Error; err != nil {
		return fmt.Errorf("创建部门失败: %w", err)
	}

	return nil
}

// Update 更新部门
func (s *departmentService) Update(ctx context.Context, req *requests.DepartmentUpdateRequest) error {
	var dept models.Department
	if err := s.db.WithContext(ctx).First(&dept, "id = ?", req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("部门不存在")
		}
		return fmt.Errorf("查询部门失败: %w", err)
	}

	if (req.ParentID == nil && dept.ParentID != nil) || (req.ParentID != nil && dept.ParentID == nil) ||
		(req.ParentID != nil && dept.ParentID != nil && *req.ParentID != *dept.ParentID) {
		ancestors, err := s.buildAncestors(ctx, req.ParentID)
		if err != nil {
			return fmt.Errorf("构建祖先链失败: %w", err)
		}
		dept.Ancestors = ancestors
	}

	if exists, err := s.checkDeptNameExists(ctx, req.ParentID, req.DeptName, req.ID); err != nil {
		return fmt.Errorf("检查部门名称失败: %w", err)
	} else if exists {
		return fmt.Errorf("同级部门名称已存在")
	}

	// 检查部门编码是否已存在（如果修改了编码）
	if req.DeptCode != "" && req.DeptCode != dept.DeptCode {
		if exists, err := s.checkDeptCodeExists(ctx, req.DeptCode, req.ID); err != nil {
			return fmt.Errorf("检查部门编码失败: %w", err)
		} else if exists {
			return fmt.Errorf("部门编码已存在")
		}
	}

	dept.DeptName = req.DeptName
	dept.DeptCode = req.DeptCode
	dept.ParentID = req.ParentID
	dept.OrderNum = req.OrderNum
	dept.Leader = req.Leader
	dept.Phone = req.Phone
	dept.Email = req.Email
	dept.IsExternalOrg = req.IsExternalOrg
	dept.Status = req.Status
	dept.Remark = toStringPtr(req.Remark)

	// 域控审查 P0-2：改名前记录旧名，用于判断是否触发 AD OU 重命名
	oldDeptName := ""
	nameChanged := false
	{
		var before models.Department
		if err := s.db.WithContext(ctx).First(&before, "id = ?", req.ID).Error; err == nil {
			oldDeptName = before.DeptName
			nameChanged = oldDeptName != req.DeptName
		}
	}

	// 同事务落库 + 经理重算入队（QUEUE-01 同生共死）。
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&dept).Error; err != nil {
			return fmt.Errorf("更新部门失败: %w", err)
		}

		// 经理重算（经理同步规则引擎）：部门 leader 是 dept_leader 来源经理的解析输入，
		// leader 变更 → 本部门及其子树全部活跃用户重算 AD manager。
		// 子树匹配沿用 operations/dept_filter.go 的 ancestors 三段 LIKE 模式。
		if leaderChanged := deptLeaderChanged(dept.Leader, req.Leader); leaderChanged {
			likeMid := "%," + dept.ID + ",%"
			likeEnd := "%," + dept.ID
			likeStart := dept.ID + ",%"
			var userIDIDs []string
			if err := tx.Model(&models.User{}).
				Where("dept_id IN (?)", tx.Model(&models.Department{}).
					Select("id").
					Where("id = ? OR ancestors LIKE ? OR ancestors LIKE ? OR ancestors LIKE ?",
						dept.ID, likeMid, likeEnd, likeStart)).
				Pluck("id", &userIDIDs).Error; err != nil {
				return fmt.Errorf("查询部门子树用户失败: %w", err)
			}
			if len(userIDIDs) > 0 {
				if err := addomain.NewADPushEnqueuer().EnqueueUpdateManager(tx, userIDIDs); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// 域控审查 P0-2：部门改名 → AD OU 重命名（ModifyDN 子树随动 + DB DN 字符串替换）。
	// 级联失败只 WARN 不回滚本地改名（本地已生效，可手动执行部门同步纠正）。
	if nameChanged && s.adCascade != nil {
		if err := s.adCascade.RenameDeptOU(ctx, dept.ID, req.DeptName); err != nil {
			applogger.Warnf("部门 %s 改名后的 AD OU 重命名失败（不影响本地保存）: %v", dept.ID, err)
		}
	}

	return nil
}

// deptLeaderChanged 判断部门负责人是否发生变更（nil 与空串视为等价）。
func deptLeaderChanged(old, new *string) bool {
	oldV, newV := "", ""
	if old != nil {
		oldV = *old
	}
	if new != nil {
		newV = *new
	}
	return oldV != newV
}

// Delete 删除部门
func (s *departmentService) Delete(ctx context.Context, id string) error {
	var dept models.Department
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&dept).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("部门不存在")
		}
		return fmt.Errorf("查询部门失败: %w", err)
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Department{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查子部门失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("存在子部门，无法删除")
	}

	if err := s.db.WithContext(ctx).Model(&models.User{}).Where("dept_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查用户失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("部门下存在用户，无法删除")
	}

	if err := s.db.WithContext(ctx).Delete(&dept).Error; err != nil {
		return fmt.Errorf("删除部门失败: %w", err)
	}

	// 域控审查 P0-2：本地删除成功后级联删除 AD OU（前提：OU 下无用户，双空检在
	// DeleteDeptOU 内）。级联失败只 WARN —— 本地删除已生效，AD 侧残留可由管理员
	// 手动清理或下轮部门同步纠正。
	if s.adCascade != nil {
		if err := s.adCascade.DeleteDeptOU(ctx, id); err != nil {
			applogger.Warnf("部门 %s 删除后的 AD OU 级联删除失败（不影响本地删除）: %v", id, err)
		}
	}

	return nil
}

func (s *departmentService) GetByID(ctx context.Context, id string) (*models.Department, error) {
	var dept models.Department
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&dept).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("部门不存在")
		}
		return nil, fmt.Errorf("查询部门失败: %w", err)
	}
	return &dept, nil
}

func (s *departmentService) GetTree(ctx context.Context, includeDisabled bool) ([]*models.Department, error) {
	return s.GetTreeWithFilter(ctx, includeDisabled, requests.DepartmentListParams{})
}

func (s *departmentService) GetTreeWithFilter(ctx context.Context, includeDisabled bool, params requests.DepartmentListParams) ([]*models.Department, error) {
	var depts []models.Department
	query := s.db.WithContext(ctx).Model(&models.Department{})

	if params.DeptName != "" {
		applogger.WithField("dept_name", params.DeptName).Info("Searching departments by name")

		var matchedDepts []models.Department
		matchedQuery := s.db.WithContext(ctx).Model(&models.Department{})

		if !includeDisabled {
			matchedQuery = matchedQuery.Where("status = ?", models.DeptStatusNormal)
		}

		if err := matchedQuery.Where("dept_name LIKE ?", "%"+params.DeptName+"%").
			Order("order_num ASC").
			Find(&matchedDepts).Error; err != nil {
			return nil, fmt.Errorf("查询部门列表失败: %w", err)
		}

		if len(matchedDepts) == 0 {
			applogger.WithField("dept_name", params.DeptName).Info("No matching departments found")
			return []*models.Department{}, nil
		}

		applogger.WithField("matched_count", len(matchedDepts)).Info("Found matched departments")

		ancestorIDs := s.collectAncestorIDs(matchedDepts)

		var ancestorDepts []models.Department
		ancestorQuery := s.db.WithContext(ctx).Model(&models.Department{})

		if !includeDisabled {
			ancestorQuery = ancestorQuery.Where("status = ?", models.DeptStatusNormal)
		}

		if len(ancestorIDs) > 0 {
			if err := ancestorQuery.Where("id IN ?", ancestorIDs).
				Order("order_num ASC").
				Find(&ancestorDepts).Error; err != nil {
				return nil, fmt.Errorf("查询祖先部门失败: %w", err)
			}
		}

		deptMap := make(map[string]*models.Department)

		for i := range ancestorDepts {
			deptMap[ancestorDepts[i].ID] = &ancestorDepts[i]
		}

		for i := range matchedDepts {
			deptMap[matchedDepts[i].ID] = &matchedDepts[i]
		}

		depts = make([]models.Department, 0, len(deptMap))
		for _, dept := range deptMap {
			depts = append(depts, *dept)
		}

		applogger.WithFields(map[string]interface{}{
			"ancestor_count": len(ancestorDepts),
			"matched_count":  len(matchedDepts),
			"unique_count":   len(deptMap),
		}).Info("Combined departments for search (deduplicated)")
	} else {
		if !includeDisabled {
			query = query.Where("status = ?", models.DeptStatusNormal)
		}

		if params.Status != nil {
			query = query.Where("status = ?", *params.Status)
		}

		if err := query.Order("order_num ASC").Find(&depts).Error; err != nil {
			return nil, fmt.Errorf("查询部门列表失败: %w", err)
		}
	}

	s.fillLeaderInfo(ctx, depts)

	deptTree := s.buildDeptTree(depts, nil)
	return deptTree, nil
}

func (s *departmentService) List(ctx context.Context, params requests.DepartmentListParams) ([]models.Department, error) {
	var depts []models.Department
	query := s.db.WithContext(ctx).Model(&models.Department{})

	if params.DeptName != "" {
		query = query.Where("dept_name LIKE ?", "%"+params.DeptName+"%")
	}

	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	if err := query.Order("order_num ASC").Find(&depts).Error; err != nil {
		return nil, fmt.Errorf("查询部门列表失败: %w", err)
	}

	s.fillLeaderInfo(ctx, depts)

	return depts, nil
}

func (s *departmentService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("ids不能为空")
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Department{}).Where("parent_id IN ?", ids).Count(&count).Error; err != nil {
		return fmt.Errorf("检查子部门失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("存在子部门，无法删除")
	}

	if err := s.db.WithContext(ctx).Model(&models.User{}).Where("dept_id IN ?", ids).Count(&count).Error; err != nil {
		return fmt.Errorf("检查用户失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("部分部门下存在用户，无法删除")
	}

	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.Department{}).Error; err != nil {
		return fmt.Errorf("批量删除部门失败: %w", err)
	}

	// 域控审查 P0-2：批量删除同样级联删 AD OU（逐个走 DeleteDeptOU 的双空检；
	// 级联失败只 WARN，与单个删除一致）。
	if s.adCascade != nil {
		for _, id := range ids {
			if err := s.adCascade.DeleteDeptOU(ctx, id); err != nil {
				applogger.Warnf("部门 %s 批量删除后的 AD OU 级联删除失败（不影响本地删除）: %v", id, err)
			}
		}
	}

	return nil
}

func (s *departmentService) UpdateStatus(ctx context.Context, id string, status int) error {
	var dept models.Department
	if err := s.db.WithContext(ctx).First(&dept, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("部门不存在")
		}
		return fmt.Errorf("查询部门失败: %w", err)
	}

	// 更新状态
	if err := s.db.WithContext(ctx).Model(&dept).Update("status", status).Error; err != nil {
		return fmt.Errorf("更新部门状态失败: %w", err)
	}

	return nil
}

func (s *departmentService) buildDeptTree(depts []models.Department, parentID *string) []*models.Department {
	var tree []*models.Department
	for _, dept := range depts {
		isParent := false
		if parentID == nil {
			isParent = dept.ParentID == nil || dept.Ancestors == ""
		} else {
			isParent = dept.ParentID != nil && *dept.ParentID == *parentID
		}

		if isParent {
			children := s.buildDeptTree(depts, &dept.ID)
			if len(children) > 0 {
				dept.Children = children
			}
			deptCopy := dept
			tree = append(tree, &deptCopy)
		}
	}
	return tree
}

func (s *departmentService) buildAncestors(ctx context.Context, parentID *string) (string, error) {
	if parentID == nil || *parentID == "" {
		return "", nil
	}

	var parentDept models.Department
	if err := s.db.WithContext(ctx).First(&parentDept, "id = ?", *parentID).Error; err != nil {
		return "", fmt.Errorf("查询父部门失败: %w", err)
	}

	if parentDept.Ancestors != "" {
		return parentDept.Ancestors + "," + *parentID, nil
	}

	return *parentID, nil
}

func (s *departmentService) checkDeptNameExists(ctx context.Context, parentID *string, deptName string, excludeID string) (bool, error) {
	var count int64
	query := s.db.WithContext(ctx).Model(&models.Department{}).Where("dept_name = ?", deptName)

	if parentID != nil {
		query = query.Where("parent_id = ?", *parentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *departmentService) GetRoleDeptIDs(ctx context.Context, roleID string, deptIDs *[]string) error {
	return s.db.WithContext(ctx).
		Table("sys_role_dept").
		Where("role_id = ?", roleID).
		Pluck("dept_id", deptIDs).Error
}

func (s *departmentService) GetTreeWithCache(ctx context.Context, includeDisabled bool) ([]*models.Department, error) {
	return s.GetTree(ctx, includeDisabled)
}

func (s *departmentService) GetSelectDataWithCache(ctx context.Context) ([]*models.Department, error) {
	return s.GetTree(ctx, false)
}

func (s *departmentService) InvalidateDeptCache(ctx context.Context) error {
	return nil
}

func (s *departmentService) GetDB() *gorm.DB {
	return s.db
}

func (s *departmentService) fillLeaderInfo(ctx context.Context, depts []models.Department) {
	var leaderIDs []string
	for _, dept := range depts {
		if dept.Leader != nil && *dept.Leader != "" {
			leaderIDs = append(leaderIDs, *dept.Leader)
		}
	}

	if len(leaderIDs) == 0 {
		return
	}

	// sys_dept.leader 字段目前无 UUID 约束，种子数据可能存用户名等非 UUID 值，
	// 过滤后再查 sys_user.id 防止 22P02 (invalid input syntax for type uuid)
	var validLeaderIDs []string
	for _, id := range leaderIDs {
		if constants.UUIDPattern.MatchString(id) {
			validLeaderIDs = append(validLeaderIDs, id)
		}
	}
	if len(validLeaderIDs) == 0 {
		return
	}

	var leaders []models.User
	if err := s.db.WithContext(ctx).
		Select("id, username, nickname").
		Where("id IN ?", validLeaderIDs).
		Find(&leaders).Error; err != nil {
		return
	}

	leaderMap := make(map[string]*models.User)
	for i := range leaders {
		leaderMap[leaders[i].ID] = &leaders[i]
	}

	for i := range depts {
		if depts[i].Leader != nil && *depts[i].Leader != "" {
			if leader, exists := leaderMap[*depts[i].Leader]; exists {
				depts[i].LeaderName = leader.Nickname
				depts[i].LeaderUsername = &leader.Username
			}
		}
	}
}

func (s *departmentService) collectAncestorIDs(depts []models.Department) []string {
	ancestorMap := make(map[string]bool)

	for _, dept := range depts {
		if dept.Ancestors != "" {
			ancestors := splitAncestors(dept.Ancestors)
			for _, ancestorID := range ancestors {
				ancestorMap[ancestorID] = true
			}
		}
	}

	result := make([]string, 0, len(ancestorMap))
	for id := range ancestorMap {
		result = append(result, id)
	}

	return result
}

func splitAncestors(ancestors string) []string {
	if ancestors == "" {
		return []string{}
	}

	parts := strings.Split(ancestors, ",")

	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}

func (s *departmentService) checkDeptCodeExists(ctx context.Context, deptCode string, excludeID string) (bool, error) {
	var count int64
	query := s.db.WithContext(ctx).Model(&models.Department{}).Where("dept_code = ?", deptCode)

	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
