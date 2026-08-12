package operations

import (
	"context"
	"errors"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// aliasListJoinSelect alias 列表查询 select 子句:
// LEFT JOIN sys_dept 两次(原组织 / 物理位置),获取部门名方便前端展示。
const aliasListJoinSelect = `sys_dept_location_alias.*,
	origin.dept_name AS origin_dept_name,
	location.dept_name AS location_dept_name`

// aliasListJoinClause alias 列表查询 join 子句:
//
// alias.dept_id / location_id 都是 varchar(64),sys_dept.id 是 uuid。
// PG 拒绝 uuid = character varying 的列对列比较(SQLSTATE 42883),
// 必须显式转换一侧类型。用标准 SQL 的 CAST(... AS TEXT):
//   - PG: CAST(uuid AS TEXT) 把 uuid 转成 text,再与 varchar 比较(text 与 varchar 同族,有 = 操作符)
//   - SQLite: uuid 列实际存为 TEXT,CAST(x AS TEXT) 是 no-op,TEXT = TEXT 正常
// 故 CAST 写法在 PG/SQLite 双 DB 行为一致(不能用 PG 专有 `::text`,SQLite 语法错误)。
const aliasListJoinClause = `LEFT JOIN sys_dept origin ON CAST(origin.id AS TEXT) = sys_dept_location_alias.dept_id AND origin.deleted_at IS NULL
	LEFT JOIN sys_dept location ON CAST(location.id AS TEXT) = sys_dept_location_alias.location_id AND location.deleted_at IS NULL`

// aliasDefaultScope alias.scope 字段默认取值(D-04 决策:聚焦 workstation 场景)。
const aliasDefaultScope = "workstation"

// LocationAliasService 工位部门 ↔ 物理位置部门 映射(别名)服务接口
//
// 用途:管理 sys_dept_location_alias 表的 CRUD,工位导入/查询时把外部物理机构
// 合入到本系统的部门选择列表。
//
// 二级校验(REQ-39-02):所有 Create/Update 调用都会经过 validateAlias 二级校验:
//  1. 自映射拦截(dept_id == location_id)
//  2. location 必须是外部机构(is_external_org = 1)
//
// 校验失败返回中文 error,handler 直接透传给前端(400 + 明确错误信息)。
type LocationAliasService interface {
	// List 查询 alias 列表(分页 + 含 dept_name JOIN)。
	List(ctx context.Context, pageNum, pageSize int) (*PageResult, error)
	// GetByID 按 ID 查询单条 alias。
	GetByID(ctx context.Context, id string) (*models.SysDeptLocationAlias, error)
	// Create 新建 alias(必跑 validateAlias 三级校验,scope 空则兜底为 "workstation")。
	Create(ctx context.Context, req *LocationAliasCreateRequest) (*models.SysDeptLocationAlias, error)
	// Update 更新 alias(dept_id/location_id 变更时重新跑 validateAlias)。
	Update(ctx context.Context, id string, req *LocationAliasUpdateRequest) error
	// Delete 软删除 alias(GORM 默认 deleted_at)。
	Delete(ctx context.Context, id string) error
}

// LocationAliasCreateRequest 新建 alias 请求体
type LocationAliasCreateRequest struct {
	DeptID     string `json:"deptId" binding:"required"`     // 原组织部门ID(逻辑部门,sys_dept.id)
	LocationID string `json:"locationId" binding:"required"` // 物理位置部门ID(外部机构,sys_dept.id 且 is_external_org=1)
	Scope      string `json:"scope"`                         // 作用范围,默认 "workstation"
	Remark     string `json:"remark"`                        // 备注
}

// LocationAliasUpdateRequest 更新 alias 请求体(所有字段可选,仅修改传入的非空字段)
type LocationAliasUpdateRequest struct {
	DeptID     *string `json:"deptId"`
	LocationID *string `json:"locationId"`
	Scope      *string `json:"scope"`
	Remark     *string `json:"remark"`
}

// LocationAliasListItem alias 列表项(含 JOIN sys_dept 出来的部门名)。
//
// 这是查询 DTO,不是表模型 —— origin_dept_name / location_dept_name 是 List
// 查询通过 aliasListJoinSelect 动态 JOIN sys_dept 出来的列,不对应
// sys_dept_location_alias 的真实列(故不放进 SysDeptLocationAlias model,
// 避免 AutoMigrate 误建列)。嵌入 SysDeptLocationAlias 复用其全部字段。
type LocationAliasListItem struct {
	models.SysDeptLocationAlias
	OriginDeptName   string `gorm:"column:origin_dept_name" json:"originDeptName,omitempty"`
	LocationDeptName string `gorm:"column:location_dept_name" json:"locationDeptName,omitempty"`
}

// locationAliasServiceImpl LocationAliasService 私有实现
type locationAliasServiceImpl struct {
	db *gorm.DB
}

// NewLocationAliasService 构造 LocationAliasService 实例
func NewLocationAliasService(db *gorm.DB) LocationAliasService {
	return &locationAliasServiceImpl{db: db}
}

// List 查询 alias 列表(分页 + JOIN sys_dept 取 dept_name)
func (s *locationAliasServiceImpl) List(ctx context.Context, pageNum, pageSize int) (*PageResult, error) {
	if pageNum <= 0 {
		pageNum = constants.DefaultCurrent
	}
	if pageSize <= 0 {
		pageSize = constants.DefaultPageSize
	}
	pageSize = clampPageSize(pageSize)

	var total int64
	var list []LocationAliasListItem

	query := s.db.WithContext(ctx).Model(&models.SysDeptLocationAlias{}).
		Where("sys_dept_location_alias.deleted_at IS NULL")

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (pageNum - 1) * pageSize
	if err := query.
		Select(aliasListJoinSelect).
		Joins(aliasListJoinClause).
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

// GetByID 按 ID 查询 alias
func (s *locationAliasServiceImpl) GetByID(ctx context.Context, id string) (*models.SysDeptLocationAlias, error) {
	var alias models.SysDeptLocationAlias
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&alias).Error
	if err != nil {
		return nil, err
	}
	return &alias, nil
}

// Create 新建 alias
func (s *locationAliasServiceImpl) Create(ctx context.Context, req *LocationAliasCreateRequest) (*models.SysDeptLocationAlias, error) {
	if req == nil {
		return nil, errors.New("请求体不能为空")
	}

	// scope 兜底:空值/缺省一律设为 "workstation"(D-04 决策)
	scope := req.Scope
	if scope == "" {
		scope = aliasDefaultScope
	}

	alias := &models.SysDeptLocationAlias{
		DeptID:     req.DeptID,
		LocationID: req.LocationID,
		Scope:      scope,
		Remark:     req.Remark,
	}

	if err := s.validateAlias(ctx, alias); err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Create(alias).Error; err != nil {
		// partial unique idx_sys_dept_location_alias_dept_scope 冲突 → 友好中文
		// (CR-03a:避免把 PG 索引名 idx_... 透传给前端)
		// isDuplicateKeyError 同包 floor_service.go 提供,跨 PG/SQLite 识别
		// (pgconn 23505 + 字符串兜底)。
		if isDuplicateKeyError(err) {
			return nil, fmt.Errorf("该 dept_id + scope 组合已存在,不可重复创建: dept_id=%s scope=%s", alias.DeptID, alias.Scope)
		}
		return nil, fmt.Errorf("创建别名映射失败: %w", err)
	}
	return alias, nil
}

// Update 更新 alias
//
// 实现说明:handler 反序列化的请求体只含表单字段,不含 id/createdAt/updatedAt,
// 必须先用 id 查出原记录,再用 req 中的非空字段覆写,最后 Save。
// dept_id / location_id / scope 任一字段变更时都重新跑 validateAlias(保持校验对称性,
// 修复 CR-03b:之前仅 dept/location 变更触发,scope 单独变更漏校验)。
func (s *locationAliasServiceImpl) Update(ctx context.Context, id string, req *LocationAliasUpdateRequest) error {
	if req == nil {
		return errors.New("请求体不能为空")
	}

	var existing models.SysDeptLocationAlias
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&existing).Error; err != nil {
		return fmt.Errorf("别名映射不存在: %w", err)
	}

	deptChanged := false
	locationChanged := false
	scopeChanged := false

	if req.DeptID != nil && *req.DeptID != existing.DeptID {
		existing.DeptID = *req.DeptID
		deptChanged = true
	}
	if req.LocationID != nil && *req.LocationID != existing.LocationID {
		existing.LocationID = *req.LocationID
		locationChanged = true
	}
	if req.Scope != nil && *req.Scope != existing.Scope {
		existing.Scope = *req.Scope
		scopeChanged = true
	}
	if req.Remark != nil {
		existing.Remark = *req.Remark
	}

	// dept_id / location_id / scope 任一变更时重跑二级校验(CR-03b 修复)
	if deptChanged || locationChanged || scopeChanged {
		if err := s.validateAlias(ctx, &existing); err != nil {
			return err
		}
	}

	// 保留创建时间(避免 Save 零值覆盖)
	if err := s.db.WithContext(ctx).Save(&existing).Error; err != nil {
		// partial unique 冲突 → 友好中文(CR-03a)
		if isDuplicateKeyError(err) {
			return fmt.Errorf("该 dept_id + scope 组合已存在,不可重复: dept_id=%s scope=%s", existing.DeptID, existing.Scope)
		}
		return fmt.Errorf("更新别名映射失败: %w", err)
	}
	return nil
}

// Delete 软删除 alias(GORM 默认 deleted_at)
func (s *locationAliasServiceImpl) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&models.SysDeptLocationAlias{}).Error
}

// validateAlias 二级校验(REQ-39-02 后端防线)
//
//  1. 防自映射:dept_id == location_id 直接拒绝
//  2. location 必须是外部机构:is_external_org = 1
//
// 校验失败返回中文 error,handler 透传给前端,HTTP 400。
//
// 历史偏差说明:原 SPEC REQ-39-02 还含第 3 条"dept 必须是 location 的后代
// (dept.ancestors 含 location_id)"。该规则与 Phase 39 功能本质直接矛盾——
// alias 存在的意义正是"组织编制与物理办公地分离"(dept 编制上属于另一分支,
// 物理上在 location 办公),dept 本就不是 location 的后代。SPEC 自身的验收用例
// (dept=运营服务部/子部门A, location=中心支公司B)在该规则下也无法创建。
// UAT 确认后移除规则③,保留 ①防自映射 + ②location 必须是外部机构。详见
// 39-VERIFICATION.md 偏差记录。
func (s *locationAliasServiceImpl) validateAlias(ctx context.Context, alias *models.SysDeptLocationAlias) error {
	// 1. 防自映射
	if alias.DeptID == alias.LocationID {
		return errors.New("dept_id 与 location_id 不能相同(自映射)")
	}

	// 2. location 必须是外部机构(is_external_org = 1)
	//
	// sys_dept.id 是 uuid,alias.location_id 是 varchar(64)。
	// 生产 PG 用 `id::text = ?` 强制把 uuid 转 text 再比较;SQLite 端 GORM
	// 会按字符串直接比对(SQLite 的 uuid 列实际就是 TEXT),也 OK。
	// 为保证 PG/SQLite 双 DB 行为一致,这里走 db.Where("id = ?", ...) 让
	// GORM 按驱动类型自动选择比较方式(PG 用 uuid 比较;SQLite 用 TEXT 比较)。
	var location models.Department
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", alias.LocationID).
		First(&location).Error; err != nil {
		return fmt.Errorf("物理位置部门不存在: %w", err)
	}
	if location.IsExternalOrg != 1 {
		return errors.New("物理位置必须是外部机构(is_external_org=1)")
	}

	return nil
}
