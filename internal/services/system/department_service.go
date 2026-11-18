package system

import (
	"context"
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/pkg/constants"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
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
	UpdateStatus(ctx context.Context, id string, status models.DeptStatus) error
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

	// 机构层级合法性（契约 §二 2.1）；0=非机构为默认值，存量请求零回归。
	if err := validateOrgLevel(req.OrgLevel); err != nil {
		return err
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

	// 机构层级合法性（契约 §二 2.1）：只允许 0=非机构 / 1=分公司 / 2=中支 / 3=支公司 / 4=网点。
	// 越界值直接拒绝，避免落库后锚点（anchor_mode=5）永远解析不到却无提示。
	if err := validateOrgLevel(req.OrgLevel); err != nil {
		return err
	}

	dept.DeptName = req.DeptName
	dept.DeptCode = req.DeptCode
	dept.ParentID = req.ParentID
	dept.OrderNum = req.OrderNum
	dept.Leader = req.Leader
	dept.Phone = req.Phone
	dept.Email = req.Email
	dept.IsExternalOrg = req.IsExternalOrg
	dept.OrgLevel = models.OrgLevel(req.OrgLevel)
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

	// Scoped opt-in：按登录人数据范围/机构锚点收口部门树（契约 v1.54 §五 前端可见性）。
	// 仅「部门管理」「系统用户」等管理视图传 scoped=true；tree-select（角色自定义
	// 数据范围勾选树、岗位锚点部门选择）等配置场景保持全量——管理员必须看到全树
	// 才能正确配置。未注入/All 原样返回（超管与既有调用方零回归）。
	if params.Scoped {
		deptTree = filterDeptTreeByScope(ctx, deptTree)
	}

	return deptTree, nil
}

// filterDeptTreeByScope 按登录人数据范围收口部门树（v1.54，Scoped opt-in）。
//
// 语义与前端 DeptTree/visibleDeptFilter.ts 的 filterTreeByVisibleDeptIds 完全对齐
// （Phase 133 / DS-06 预留的前端基建，后端现在补上收口为唯一真实源）：
//   - 未注入中间件（内部任务/单测）或 All → 原样返回（零回归）；
//   - FailClosed（异常档位）或 HookSelfUser（仅本人无部门集）→ 空树 fail-closed；
//   - DeptIDs 非空 → **精确匹配**：仅保留集合内节点 + 祖先路径（祖先做值拷贝）。
//     注意：DeptChild/锚点档的 DeptIDs 已由 Resolve 展开为完整子树，故精确匹配
//     天然覆盖它们的子树；Dept/Custom 档的精确清单也不会被放大到下级部门。
//
// 过滤时对"祖先路径保留"的节点做**值拷贝**（新建节点替换 Children），不改写原树
// ——防御性写法：GetTreeWithCache 目前是无缓存的直通转发，但未来若接入真缓存，
// 原地改写 Children 会把收口结果污染进全局缓存（跨用户可见性事故）。
func filterDeptTreeByScope(ctx context.Context, tree []*models.Department) []*models.Department {
	r, ok := datascope.FromContext(ctx)
	// 诊断日志（v1.54）：记录 scoped 请求的服务端真实状态，用于排查"请求体带
	// scoped:true 但响应全树"类问题。生产全树时看这条即可判断是
	// ①ctx 未注入(ok=false) ②All 档短路 ③DeptIDs 为空 ④过滤真生效。
	beforeCount := countDeptTreeNodes(tree)
	applogger.WithFields(map[string]interface{}{
		"ctx_injected": ok,
		"scope_all":    r.All,
		"fail_closed":  r.FailClosed,
		"dept_ids_len": len(r.DeptIDs),
		"hook":         int(r.ResourceHook),
		"before_nodes": beforeCount,
	}).Info("[部门树收口] scoped 请求诊断")

	if !ok || r.All {
		return tree
	}
	if r.FailClosed || r.ResourceHook == datascope.HookSelfUser || len(r.DeptIDs) == 0 {
		return []*models.Department{}
	}

	visible := make(map[string]struct{}, len(r.DeptIDs))
	for _, id := range r.DeptIDs {
		visible[id] = struct{}{}
	}

	// 精确匹配裁剪：只保留 DeptIDs 集合内的节点 + 通往这些节点的祖先路径。
	//
	// 关键修正（审计 P0）：DeptIDs 已是**最终可见集**——
	//   - DeptChild/机构锚点档：Resolve 已把子树展开进 DeptIDs，每个可见节点都在集合里；
	//   - Dept/Custom 档：DeptIDs 就是精确清单，本就不该含子部门。
	// 因此这里**不再**因"节点自身命中"就无条件保留其整棵子树——那会让 Dept(3)
	// 档用户看到下级部门树（数据范围放大）。改为逐节点精确判断：命中就保留该节点
	// 本身；未命中但后代有命中则作为祖先路径保留（值拷贝，Children 只含命中分支）。
	var walk func(nodes []*models.Department) []*models.Department
	walk = func(nodes []*models.Department) []*models.Department {
		out := make([]*models.Department, 0, len(nodes))
		for _, n := range nodes {
			_, selfHit := visible[n.ID]
			keptChildren := walk(n.Children)
			if selfHit {
				// 自身可见：保留自身。子节点由各自的命中/祖先逻辑决定，
				// 不再无条件整棵 append（避免 Dept 档放大到子部门）。
				node := *n
				if len(keptChildren) > 0 {
					node.Children = keptChildren
				} else {
					node.Children = nil
				}
				out = append(out, &node)
			} else if len(keptChildren) > 0 {
				pathNode := *n
				pathNode.Children = keptChildren
				out = append(out, &pathNode)
			}
		}
		return out
	}
	filtered := walk(tree)
	applogger.WithFields(map[string]interface{}{
		"after_nodes": countDeptTreeNodes(filtered),
		"user_id":     r.UserID,
	}).Info("[部门树收口] 收口完成")
	return filtered
}

// countDeptTreeNodes 统计树节点数（仅用于 filterDeptTreeByScope 的诊断日志）。
func countDeptTreeNodes(nodes []*models.Department) int {
	n := 0
	for _, node := range nodes {
		n++
		n += countDeptTreeNodes(node.Children)
	}
	return n
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

	// Scoped opt-in：扁平部门列表按登录人可见集过滤（v1.54）。
	// 与 GetTreeWithFilter 共用同一 ctx ScopeResult；列表是扁平结构，直接按 ID 精确
	// 过滤即可（无需树裁剪）。未注入/All 原样返回，Self/异常 fail-closed 空列表。
	if params.Scoped {
		filtered := filterDeptListByScope(ctx, depts)
		return filtered, nil
	}

	s.fillLeaderInfo(ctx, depts)

	return depts, nil
}

// filterDeptListByScope 按登录人可见集过滤扁平部门列表（v1.54，Scoped opt-in）。
// 语义与 filterDeptTreeByScope 一致：未注入/All 原样；FailClosed/Self/空集 → 空列表；
// DeptIDs 非空 → 仅保留集合内的部门（精确匹配，不放大到子部门）。
func filterDeptListByScope(ctx context.Context, depts []models.Department) []models.Department {
	r, ok := datascope.FromContext(ctx)
	if !ok || r.All {
		return depts
	}
	if r.FailClosed || r.ResourceHook == datascope.HookSelfUser || len(r.DeptIDs) == 0 {
		return []models.Department{}
	}
	visible := make(map[string]struct{}, len(r.DeptIDs))
	for _, id := range r.DeptIDs {
		visible[id] = struct{}{}
	}
	out := make([]models.Department, 0, len(depts))
	for _, d := range depts {
		if _, hit := visible[d.ID]; hit {
			out = append(out, d)
		}
	}
	return out
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

func (s *departmentService) UpdateStatus(ctx context.Context, id string, status models.DeptStatus) error {
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

// validateOrgLevel 校验机构层级取值（契约 §二 2.1）。
//
// 0=非机构（普通部门/科室，默认值，存量数据零回归）/ 1=分公司 / 2=中支 /
// 3=支公司 / 4=网点。越界值拒绝：落库后 anchor_mode=5 的锚点会永远解析不到，
// 用户在运行期只会看到 fail-closed 空集而不知道原因，故挡在保存期。
func validateOrgLevel(level int) error {
	if level < int(models.OrgLevelNone) || level > int(models.OrgLevelSite) {
		return apperrors.BadRequest(fmt.Sprintf(
			"机构层级非法：%d（仅支持 0=非机构 / 1=分公司 / 2=中支 / 3=支公司 / 4=网点）", level))
	}
	return nil
}
