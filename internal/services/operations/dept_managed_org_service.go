package operations

import (
	"context"
	"errors"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/constants"
	pkgquery "github.com/xingran-next/xingran-go-backend/pkg/query"
	"gorm.io/gorm"
)

// AliasScopePermission sys_dept_location_alias.scope 的「权限数据范围」取值。
//
// 语义：条线部门(dept_id，如「潜江团客共享作业服务中心」挂在运营服务本部条线下，
// ancestors 链上无任何中支) → 管理归属机构(location_id，org_level∈{1,2,3} 的
// sys_dept 节点，如潜江中支)。
//
// 与 scope='workstation'(工位物理位置映射)共用一张表(partial unique (dept_id, scope)
// 天然隔离，migration_165)，但业务语义、校验规则、管理端点、权限点完全独立：
//   - 本服务读写一律显式 WHERE scope = 'permission'，绝不触碰工位数据；
//   - 机构校验按 org_level(管理归属语义)，而非 is_external_org(有无职场)。
//
// 消费方：pkg/datascope/anchor.go 锚点解析
//   - 方向①(条线人员当管理员)：ancestors 链 miss 后按 dept_id 反查本表兜底锚点；
//   - 方向②(中支管理员管辖条线人员)：锚点机构子树展开后并入映射到该机构的条线部门。
const AliasScopePermission = "permission"

// managedOrgListJoinSelect 归属映射列表 select 子句(与工位 aliasListJoinSelect 同构，
// 列别名沿用 origin/location 以复用同一 JOIN 常量；本场景语义为 条线部门/归属机构)。
const managedOrgListJoinSelect = aliasListJoinSelect

// managedOrgListJoinClause 复用工位 alias 的 JOIN 子句(纯 sys_dept JOIN，与 scope 无关)。
const managedOrgListJoinClause = aliasListJoinClause

// DeptManagedOrgService 条线部门 → 管理归属机构 映射服务接口
//
// 只覆盖 sys_dept_location_alias 中 scope='permission' 的行。
// 不提供 Update：归属映射是简单二元关系，改错了删除重建即可(与工位 alias 的
// Update 场景不同，本场景无渐进编辑诉求)。
type DeptManagedOrgService interface {
	// List 查询归属映射列表(分页 + JOIN sys_dept 取部门名，仅 scope='permission')。
	List(ctx context.Context, pageNum, pageSize int) (*PageResult, error)
	// Create 新建归属映射(必跑 validateManagedOrg 校验)。
	Create(ctx context.Context, req *DeptManagedOrgCreateRequest) (*models.SysDeptLocationAlias, error)
	// Delete 软删除归属映射(仅限 scope='permission' 行，防误删工位映射)。
	Delete(ctx context.Context, id string) error
}

// DeptManagedOrgCreateRequest 新建归属映射请求体
type DeptManagedOrgCreateRequest struct {
	DeptID string `json:"deptId" binding:"required"` // 条线部门ID(sys_dept.id，ancestors 链上无目标机构)
	OrgID  string `json:"orgId" binding:"required"`  // 管理归属机构ID(sys_dept.id 且 org_level∈{1,2,3})
	Remark string `json:"remark"`                    // 备注
}

// DeptManagedOrgListItem 归属映射列表项(JOIN sys_dept 出的部门名，查询 DTO 不落库)。
type DeptManagedOrgListItem struct {
	models.SysDeptLocationAlias
	OriginDeptName   string `gorm:"column:origin_dept_name" json:"originDeptName,omitempty"`  // 条线部门名
	LocationDeptName string `gorm:"column:location_dept_name" json:"locationDeptName,omitempty"` // 归属机构名
}

// deptManagedOrgServiceImpl DeptManagedOrgService 私有实现
type deptManagedOrgServiceImpl struct {
	db *gorm.DB
}

// NewDeptManagedOrgService 构造 DeptManagedOrgService 实例
func NewDeptManagedOrgService(db *gorm.DB) DeptManagedOrgService {
	return &deptManagedOrgServiceImpl{db: db}
}

// List 查询归属映射列表(仅 scope='permission'，分页 + JOIN sys_dept 取部门名)
func (s *deptManagedOrgServiceImpl) List(ctx context.Context, pageNum, pageSize int) (*PageResult, error) {
	pageNum, pageSize = pkgquery.NormalizePaginationWithMax(pageNum, pageSize, constants.MaxOptionsPageSize)

	var total int64
	var list []DeptManagedOrgListItem

	// 与工位 alias 不同：List 显式限定 scope='permission'，
	// 工位映射管理列表不会混入本场景数据(反之亦然)。
	// Phase 151 / ARCH-01：读链统一 ScopedDB 起链（scopeEntityLocationAlias
	// 已登记 dept_id 直挂谓词，由 ScopedDB 自动应用，不再手动 applyEntityReadScope；
	// 挂在 Count 之前保证 Count 与 Find 同源）。
	query := pkgquery.ScopedDB(ctx, s.db).Model(&models.SysDeptLocationAlias{}).
		Where("sys_dept_location_alias.deleted_at IS NULL").
		Where("sys_dept_location_alias.scope = ?", AliasScopePermission)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (pageNum - 1) * pageSize
	if err := query.
		Select(managedOrgListJoinSelect).
		Joins(managedOrgListJoinClause).
		Order("sys_dept_location_alias.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  pageNum,
		PageSize: pageSize,
	}, nil
}

// Create 新建归属映射
func (s *deptManagedOrgServiceImpl) Create(ctx context.Context, req *DeptManagedOrgCreateRequest) (*models.SysDeptLocationAlias, error) {
	if req == nil {
		return nil, errors.New("请求体不能为空")
	}

	// Phase 149 / SCOPE-03：提交侧 dept_id 越界 → 单条显式 403（本表 dept_id 直挂）。
	if err := checkEntitySubmittedScope(ctx, s.db, scopeEntityLocationAlias, req.DeptID); err != nil {
		return nil, err
	}

	alias := &models.SysDeptLocationAlias{
		DeptID:     req.DeptID,
		LocationID: req.OrgID,
		Scope:      AliasScopePermission, // 写入恒为 permission，不接受外部指定 scope
		Remark:     req.Remark,
	}

	if err := s.validateManagedOrg(ctx, alias); err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Create(alias).Error; err != nil {
		// partial unique idx_sys_dept_location_alias_dept_scope 冲突 → 友好中文
		if isDuplicateKeyError(err) {
			return nil, fmt.Errorf("该条线部门已配置管理归属机构，不可重复创建: dept_id=%s", alias.DeptID)
		}
		return nil, fmt.Errorf("创建归属映射失败: %w", err)
	}
	return alias, nil
}

// Delete 软删除归属映射
//
// 与工位 alias 的 Delete 的关键差异：删除前必须确认 scope='permission'，
// 防止本端点误删工位物理位置映射(两场景共用一张表，ID 是唯一防线)。
func (s *deptManagedOrgServiceImpl) Delete(ctx context.Context, id string) error {
	var alias models.SysDeptLocationAlias
	err := s.db.WithContext(ctx).
		Scopes(entityReadScope(ctx, scopeEntityLocationAlias, "sys_dept_location_alias")).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&alias).Error
	if err != nil {
		return err
	}
	if alias.Scope != AliasScopePermission {
		return fmt.Errorf("该记录不是条线部门归属映射(scope=%s)，禁止通过本端点删除", alias.Scope)
	}
	return s.db.WithContext(ctx).Delete(&alias).Error
}

// validateManagedOrg 归属映射二级校验(所有 Create 调用必跑)：
//  1. 自映射拦截(dept_id == org_id)；
//  2. dept_id 必须是存在的未软删部门；org_id 必须是存在的未软删机构(org_level∈{1,2,3})。
//
// 与工位 validateAlias(is_external_org=1) 的分叉点在第 2 条的机构判据：
// 管理归属语义看「这是哪级机构」(org_level)，不看「有没有职场」(is_external_org)。
func (s *deptManagedOrgServiceImpl) validateManagedOrg(ctx context.Context, alias *models.SysDeptLocationAlias) error {
	if alias.DeptID == alias.LocationID {
		return errors.New("条线部门与管理归属机构不能相同")
	}

	var deptRow struct {
		ID string `gorm:"column:id"`
	}
	if err := s.db.WithContext(ctx).
		Table("sys_dept").
		Select("CAST(id AS TEXT) AS id").
		Where("CAST(id AS TEXT) = ? AND deleted_at IS NULL", alias.DeptID).
		First(&deptRow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("条线部门不存在或已删除")
		}
		return fmt.Errorf("校验条线部门失败: %w", err)
	}

	var orgRow struct {
		ID       string         `gorm:"column:id"`
		OrgLevel models.OrgLevel `gorm:"column:org_level"`
	}
	if err := s.db.WithContext(ctx).
		Table("sys_dept").
		Select("CAST(id AS TEXT) AS id, COALESCE(org_level, 0) AS org_level").
		Where("CAST(id AS TEXT) = ? AND deleted_at IS NULL", alias.LocationID).
		First(&orgRow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("管理归属机构不存在或已删除")
		}
		return fmt.Errorf("校验管理归属机构失败: %w", err)
	}

	switch orgRow.OrgLevel {
	case models.OrgLevelBranch, models.OrgLevelSubBang, models.OrgLevelOffice:
		// 合法机构层级：分公司 / 中支 / 支公司
	default:
		return errors.New("管理归属机构必须是机构节点(分公司/中支/支公司)，不能是普通部门")
	}

	return nil
}
