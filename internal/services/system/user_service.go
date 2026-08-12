package system

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/core/security"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// PasswordManager 密码管理器接口
type PasswordManager interface {
	HashPassword(password string) (string, error)
	VerifyPassword(password, hash string) (bool, error)
}

// UserService 用户服务接口
type UserService interface {
	Create(ctx context.Context, user *requests.UserCreateRequest) error
	Update(ctx context.Context, user *requests.UserUpdateRequest) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	List(ctx context.Context, params requests.UserListParams) (*PageResult, error)
	Statistics(ctx context.Context) (*UserStatisticsResult, error)
	BatchDelete(ctx context.Context, ids []string) error
	UpdateStatus(ctx context.Context, id string, status int) error
	ResetPassword(ctx context.Context, id string, newPassword string) error
}

// userService 用户服务实现
type userService struct {
	db         *gorm.DB
	pwdManager PasswordManager
}

// NewUserService 创建用户服务实例
func NewUserService(db *gorm.DB, pwdManager PasswordManager) UserService {
	return &userService{
		db:         db,
		pwdManager: pwdManager,
	}
}

// PageResult 分页结果
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Current  int         `json:"current"`
	PageSize int         `json:"pageSize"`
}

// UserStatisticsResult 用户统计结果。
// status 约定: 0=正常(active), 1=停用(inactive)。
type UserStatisticsResult struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`
	Inactive int64 `json:"inactive"`
}

// userAllowedSortFields 用户列表可排序字段白名单。
// 值为 DB 列名(可能含 schema/表别名),必须与查询 Select/Joins 后的列匹配。
// sys_user 因 List 末尾会 LEFT JOIN sys_dept,需用表别名限定避免歧义。
var userAllowedSortFields = map[string]string{
	"username":   "sys_user.username",
	"nickname":   "sys_user.nickname",
	"employeeNo": "sys_user.employee_no",
	"email":      "sys_user.email",
	"phone":      "sys_user.phone",
	"gender":     "sys_user.gender",
	"status":     "sys_user.status",
	"createdAt":  "sys_user.created_at",
	"updatedAt":  "sys_user.updated_at",
	"deptName":   "sys_dept.dept_name",
}

// ==================== 服务方法实现 ====================

// Create 创建用户
func (s *userService) Create(ctx context.Context, req *requests.UserCreateRequest) error {
	// 检查用户名是否已存在
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		return apperrors.DatabaseError(err)
	}
	if count > 0 {
		return apperrors.UserExistsWithUsername(req.Username)
	}

	// 加密密码
	hashedPassword, err := s.pwdManager.HashPassword(req.Password)
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeServerError, "密码加密失败")
	}

	user := req.ToModel(hashedPassword)

	// 使用事务创建用户及其关联
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建用户
		if err := tx.Create(&user).Error; err != nil {
			return apperrors.DatabaseError(err)
		}

		// 分配角色
		// 遵循 Go 最佳实践：使用批量插入避免 N+1 问题
		if len(req.RoleIds) > 0 {
			userRoles := make([]models.UserRole, len(req.RoleIds))
			for i, roleID := range req.RoleIds {
				userRoles[i] = models.UserRole{
					UserID: user.ID,
					RoleID: roleID,
				}
			}
			if err := tx.Create(&userRoles).Error; err != nil {
				return apperrors.DatabaseError(err)
			}
		}

		// 分配岗位
		// 遵循 Go 最佳实践：使用批量插入避免 N+1 问题
		if len(req.PostIds) > 0 {
			userPosts := make([]models.UserPost, len(req.PostIds))
			for i, postID := range req.PostIds {
				userPosts[i] = models.UserPost{
					UserID: user.ID,
					PostID: postID,
				}
			}
			if err := tx.Create(&userPosts).Error; err != nil {
				return apperrors.DatabaseError(err)
			}
		}

		return nil
	})

	return err
}

// Update 更新用户
func (s *userService) Update(ctx context.Context, req *requests.UserUpdateRequest) error {
	// 检查用户是否存在
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.UserNotFoundWithID(req.ID)
		}
		return apperrors.DatabaseError(err)
	}

	// 使用事务更新用户及其关联
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新用户基本信息
		user.Nickname = req.Nickname
		user.EmployeeNo = req.EmployeeNo
		user.Email = req.Email
		user.Phone = req.Phone
		user.Gender = req.Gender
		user.Status = req.Status
		user.DeptID = req.DeptID
		user.Remark = toStringPtr(req.Remark)

		// 如果部门发生变化，同步更新部门名称
		if req.DeptID != nil {
			var dept models.Department
			if err := tx.Select("dept_name").First(&dept, "id = ?", *req.DeptID).Error; err == nil {
				user.DeptName = &dept.DeptName
			}
		} else {
			user.DeptName = nil
		}

		if err := tx.Save(&user).Error; err != nil {
			return apperrors.DatabaseError(err)
		}

		// 更新角色关联：先删除旧关联，再添加新关联
		// 遵循 Go 最佳实践：使用批量插入避免 N+1 问题
		if err := tx.Table("sys_user_role").Where("user_id = ?", req.ID).Delete(&models.UserRole{}).Error; err != nil {
			return apperrors.DatabaseError(err)
		}
		if len(req.RoleIds) > 0 {
			userRoles := make([]models.UserRole, len(req.RoleIds))
			for i, roleID := range req.RoleIds {
				userRoles[i] = models.UserRole{
					UserID: req.ID,
					RoleID: roleID,
				}
			}
			if err := tx.Create(&userRoles).Error; err != nil {
				return apperrors.DatabaseError(err)
			}
		}

		// 更新岗位关联：先删除旧关联，再添加新关联
		// 遵循 Go 最佳实践：使用批量插入避免 N+1 问题
		if err := tx.Table("sys_user_post").Where("user_id = ?", req.ID).Delete(&models.UserPost{}).Error; err != nil {
			return apperrors.DatabaseError(err)
		}
		if len(req.PostIds) > 0 {
			userPosts := make([]models.UserPost, len(req.PostIds))
			for i, postID := range req.PostIds {
				userPosts[i] = models.UserPost{
					UserID: req.ID,
					PostID: postID,
				}
			}
			if err := tx.Create(&userPosts).Error; err != nil {
				return apperrors.DatabaseError(err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// Delete 删除用户
func (s *userService) Delete(ctx context.Context, id string) error {
	// 检查用户是否存在
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.UserNotFoundWithID(id)
		}
		return apperrors.DatabaseError(err)
	}

	// 使用事务删除用户及其关联数据
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除用户角色关联
		if err := tx.Table("sys_user_role").Where("user_id = ?", id).Delete(&models.UserRole{}).Error; err != nil {
			return apperrors.DatabaseError(err)
		}

		// 删除用户岗位关联
		if err := tx.Table("sys_user_post").Where("user_id = ?", id).Delete(&models.UserPost{}).Error; err != nil {
			return apperrors.DatabaseError(err)
		}

		// 软删除用户
		if err := tx.Delete(&user).Error; err != nil {
			return apperrors.DatabaseError(err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// GetByID 获取用户详情
func (s *userService) GetByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).Preload("Dept").First(&user, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.UserNotFoundWithID(id)
		}
		return nil, apperrors.DatabaseError(err)
	}

	// 填充角色信息
	s.fillUserRoles(ctx, &user)

	return &user, nil
}

// fillUserRoles 填充用户的角色信息
//
// P1 fix: 原实现两次顺序查询 (UserRole → Role IN),改为单次 JOIN,
// 减少 50% DB 往返,与已有 user_service_optimized.go 的方案保持一致。
func (s *userService) fillUserRoles(ctx context.Context, user *models.User) {
	type roleRow struct {
		ID       string
		RoleName string
	}
	var rows []roleRow
	err := s.db.WithContext(ctx).
		Table("sys_role AS r").
		Select("r.id AS id, r.role_name AS role_name").
		Joins("INNER JOIN sys_user_role ur ON ur.role_id = r.id").
		Where("ur.user_id = ? AND r.deleted_at IS NULL", user.ID).
		Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		user.Roles = []string{}
		user.RoleIds = []string{}
		return
	}

	roleIDs := make([]string, len(rows))
	roleNames := make([]string, len(rows))
	for i, row := range rows {
		roleIDs[i] = row.ID
		roleNames[i] = row.RoleName
	}
	user.Roles = roleNames
	user.RoleIds = roleIDs
}

// List 查询用户列表
func (s *userService) List(ctx context.Context, params requests.UserListParams) (*PageResult, error) {
	var total int64
	var list []models.User

	query := s.db.WithContext(ctx).Model(&models.User{})

	// 添加筛选条件
	if params.Username != nil && *params.Username != "" {
		query = query.Where("username LIKE ?", "%"+*params.Username+"%")
	}
	if params.Nickname != nil && *params.Nickname != "" {
		query = query.Where("nickname LIKE ?", "%"+*params.Nickname+"%")
	}
	if params.Phone != nil && *params.Phone != "" {
		query = query.Where("phone LIKE ?", "%"+*params.Phone+"%")
	}
	if params.Status != nil {
		// F-NEW: 必须使用 sys_user.status 限定符,因为 List() 后续会 LEFT JOIN sys_dept,
		// 而 sys_dept 也有 status 列,未限定会导致 PostgreSQL 抛出 SQLSTATE 42702
		// (column reference "status" is ambiguous)。
		query = query.Where("sys_user.status = ?", *params.Status)
	}
	if params.DeptID != nil && *params.DeptID != "" {
		query = query.Where("dept_id = ?", *params.DeptID)
	}
	if params.RecursiveDeptID != nil && *params.RecursiveDeptID != "" {
		// 递归查询该部门及所有子部门下的用户。
		// sys_dept.ancestors 格式为 "a,b,c"(无首尾逗号,不含自身),descendant 的 ancestors
		// 包含自身到根的完整祖先链(由 department_service.buildAncestors 维护)。
		// 匹配 5 种情形:
		//   1. id = rid                                  — 部门自身
		//   2. ancestors = rid                           — 直接子部门(rid 是其唯一祖先)
		//   3. ancestors LIKE 'rid,%'                    — 深层 descendant,rid 是其链首
		//   4. ancestors LIKE '%,rid,%'                  — 深层 descendant,rid 在链中
		//   5. ancestors LIKE '%,rid'                    — 深层 descendant,rid 在链尾
		// 项目中其他 service(building/asset/infopoint/server_room)使用的是 4 条件变体,
		// 漏了第 3 种情形,只对 2 层部门(A→B)正确,3 层及以上(A→B→C)会漏掉深层 descendant。
		// 这里新写一个 5 条件变体以正确处理工位运维中常见的深层组织树。
		rid := *params.RecursiveDeptID
		query = query.Where(
			"dept_id IN (?)",
			s.db.WithContext(ctx).Model(&models.Department{}).
				Select("id").
				Where(
					"id = ? OR ancestors = ? OR ancestors LIKE ? OR ancestors LIKE ? OR ancestors LIKE ?",
					rid, rid, rid+",%", "%,"+rid+",%", "%,"+rid,
				),
		)
	}
	if params.BeginTime != nil && *params.BeginTime != "" {
		if beginTime, err := time.Parse("2006-01-02 15:04:05", *params.BeginTime); err == nil {
			query = query.Where("created_at >= ?", beginTime)
		}
	}
	if params.EndTime != nil && *params.EndTime != "" {
		if endTime, err := time.Parse("2006-01-02 15:04:05", *params.EndTime); err == nil {
			query = query.Where("created_at <= ?", endTime)
		}
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 分页查询 - 使用 SQL JOIN 获取部门信息
	offset := (params.Current - 1) * params.PageSize

	// 定义 SQL JOIN 查询
	// P4 fix: sys_user.dept_id 和 sys_dept.id 都是 PostgreSQL 原生 uuid 类型列,无需任何类型转换。
	// 之前的 NULLIF/CASE WHEN 防御性写法是错误的:基于"dept_id 是 VARCHAR"的错误假设,
	// 实际查询中触发的是 PostgreSQL planner 在 uuid 列上做 != '' / ~ regex 比较时的类型推断错误。
	// 简化为直接等值比较,数据库直接处理类型,LEFT JOIN 对 NULL 自动跳过。
	userJoinSelect := "sys_user.*, sys_dept.dept_name, sys_dept.ancestors"
	userJoinClause := "LEFT JOIN sys_dept ON sys_dept.id = sys_user.dept_id"

	// 修复 GORM v1 链式调用顺序:ApplySort(.Order) 必须在 Select/Joins 之前调用,
	// 否则 GORM 在 Select/Joins 改变 statement state 时会丢弃 OrderBuilds,
	// 导致最终 SQL 没有 ORDER BY(这是 user 排序无效的根因之一)。
	// role list 无 Joins 所以未触发;user list 有 LEFT JOIN sys_dept 所以 username/nickname/email/phone 都失效。
	// 另一个根因:user_cache_impl/role_cache_impl 的 buildListCacheKey 之前遗漏了
	// orderByColumn/isAsc 等 BaseListRequest 字段,导致不同排序请求命中同一缓存。
	query = base.ApplySort(query, params.BaseListRequest, userAllowedSortFields)

	// 再应用 Select + Joins(决定可见列)
	query = query.Select(userJoinSelect).Joins(userJoinClause)

	if err := query.Offset(offset).Limit(params.PageSize).Find(&list).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 注: 部门路径在 line 404 角色填充后统一构建,删除此处早期调用(P1 fix)。

	// 填充用户角色信息
	userIDs := make([]string, len(list))
	for i, u := range list {
		userIDs[i] = u.ID
	}

	// 查询用户角色关系
	var userRoles []models.UserRole
	if len(userIDs) > 0 {
		s.db.WithContext(ctx).Where("user_id IN ?", userIDs).Find(&userRoles)
	}

	// 构建用户ID到角色ID的映射
	userRoleMap := make(map[string][]string)
	for _, ur := range userRoles {
		userRoleMap[ur.UserID] = append(userRoleMap[ur.UserID], ur.RoleID)
	}

	// 查询所有相关角色
	var allRoles []models.Role
	if len(userRoles) > 0 {
		roleIDs := make([]string, 0)
		for _, ur := range userRoles {
			roleIDs = append(roleIDs, ur.RoleID)
		}
		s.db.WithContext(ctx).Where("id IN ?", roleIDs).Find(&allRoles)
	}

	// 构建角色ID到角色名称的映射
	roleMap := make(map[string]string)
	for _, r := range allRoles {
		roleMap[r.ID] = r.RoleName
	}

	// 填充每个用户的角色名称数组和角色ID数组
	for i := range list {
		if roleIDs, ok := userRoleMap[list[i].ID]; ok {
			roleNames := make([]string, 0, len(roleIDs))
			roleIDsCopy := make([]string, 0, len(roleIDs))
			for _, rid := range roleIDs {
				if roleName, exists := roleMap[rid]; exists {
					roleNames = append(roleNames, roleName)
				}
				roleIDsCopy = append(roleIDsCopy, rid)
			}
			list[i].Roles = roleNames
			list[i].RoleIds = roleIDsCopy
		}
	}

	// 构建部门路径
	s.buildDepartmentPaths(ctx, list)

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// Statistics 统计用户总数及启停状态计数(status: 0=正常, 1=停用)。
// 用单条带条件聚合的查询(SUM(CASE WHEN ...)),避免「加载全量行进内存再 filter」的反模式;
// 用 CASE 而非 COUNT FILTER 是为了同时兼容 PostgreSQL 与 SQLite 两种后端。
// base query 与 List 一致(s.db.Model(&models.User{})),GORM 自动排除软删除记录,
// 因此 total 与 List 的 Count 口径完全对齐。
func (s *userService) Statistics(ctx context.Context) (*UserStatisticsResult, error) {
	var result UserStatisticsResult
	err := s.db.WithContext(ctx).
		Model(&models.User{}).
		Select(
			"COUNT(*) AS total",
			"SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END) AS active",
			"SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) AS inactive",
		).
		Scan(&result).Error
	if err != nil {
		return nil, apperrors.DatabaseError(err)
	}
	return &result, nil
}

// buildDepartmentPaths 为用户列表构建完整部门路径（从二级部门开始）
func (s *userService) buildDepartmentPaths(ctx context.Context, list []models.User) {
	// 收集所有唯一的祖先部门ID
	allAncestorIDs := make(map[string]bool)

	for _, user := range list {
		// 使用 DeptAncestors 字段（由 JOIN 查询填充）
		if user.DeptID != nil && user.DeptAncestors != nil && *user.DeptAncestors != "" {
			ancestors := splitAncestors(*user.DeptAncestors)
			for _, ancestorID := range ancestors {
				allAncestorIDs[ancestorID] = true
			}
		}
	}

	if len(allAncestorIDs) == 0 {
		return
	}

	// 批量查询所有祖先部门
	ancestorIDList := make([]string, 0, len(allAncestorIDs))
	for id := range allAncestorIDs {
		ancestorIDList = append(ancestorIDList, id)
	}

	var ancestorDepts []models.Department
	s.db.WithContext(ctx).Select("id", "dept_name").Where("id IN ?", ancestorIDList).Find(&ancestorDepts)

	// 构建祖先部门ID到名称的映射
	ancestorDeptMap := make(map[string]string)
	for i := range ancestorDepts {
		ancestorDeptMap[ancestorDepts[i].ID] = ancestorDepts[i].DeptName
	}

	// 为每个用户构建部门路径
	for i := range list {
		user := &list[i]

		// 检查是否有部门路径信息
		if user.DeptID == nil || user.DeptAncestors == nil || *user.DeptAncestors == "" {
			// 如果没有 ancestors，至少显示当前部门名称
			if user.DeptName != nil {
				user.DeptFullName = user.DeptName
			}
			continue
		}

		ancestors := splitAncestors(*user.DeptAncestors)
		pathParts := make([]string, 0, len(ancestors)+1)

		// 添加所有祖先部门名称（不跳过任何层级）
		for _, ancestorID := range ancestors {
			if name, exists := ancestorDeptMap[ancestorID]; exists {
				pathParts = append(pathParts, name)
			}
		}

		// 添加当前部门名称
		if user.DeptName != nil {
			pathParts = append(pathParts, *user.DeptName)
		}

		// 构建路径字符串
		if len(pathParts) > 0 {
			path := ""
			for k, part := range pathParts {
				if k > 0 {
					path += "/"
				}
				path += part
			}
			user.DeptFullName = &path
		}
	}
}


// BatchDelete 批量删除用户
func (s *userService) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return apperrors.ParamMissing("ids")
	}

	// 使用事务批量删除用户及其关联数据
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 批量删除用户角色关联
		if err := tx.Table("sys_user_role").Where("user_id IN ?", ids).Delete(&models.UserRole{}).Error; err != nil {
			return apperrors.DatabaseError(err)
		}

		// 批量删除用户岗位关联
		if err := tx.Table("sys_user_post").Where("user_id IN ?", ids).Delete(&models.UserPost{}).Error; err != nil {
			return apperrors.DatabaseError(err)
		}

		// 批量软删除用户
		if err := tx.Where("id IN ?", ids).Delete(&models.User{}).Error; err != nil {
			return apperrors.DatabaseError(err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// UpdateStatus 更新用户状态
func (s *userService) UpdateStatus(ctx context.Context, id string, status int) error {
	// 检查用户是否存在
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.UserNotFoundWithID(id)
		}
		return apperrors.DatabaseError(err)
	}

	// 更新状态
	if err := s.db.WithContext(ctx).Model(&user).Update("status", status).Error; err != nil {
		return apperrors.DatabaseError(err)
	}

	return nil
}

// ResetPassword 重置用户密码
func (s *userService) ResetPassword(ctx context.Context, id string, newPassword string) error {
	// 检查用户是否存在
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.UserNotFoundWithID(id)
		}
		return apperrors.DatabaseError(err)
	}

	// 加密新密码
	hashedPassword, err := s.pwdManager.HashPassword(newPassword)
	if err != nil {
		return apperrors.Wrap(err, apperrors.CodeServerError, "密码加密失败")
	}

	// 更新密码
	if err := s.db.WithContext(ctx).Model(&user).Update("password", hashedPassword).Error; err != nil {
		return apperrors.DatabaseError(err)
	}

	return nil
}

// ==================== 辅助函数 ====================

// toStringPtr 将字符串转换为指针
func toStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// passwordManagerAdapter 密码管理器适配器
type passwordManagerAdapter struct {
	*security.PasswordManager
}

// 确保passwordManagerAdapter实现了PasswordManager接口
var _ PasswordManager = (*passwordManagerAdapter)(nil)

// NewPasswordManagerAdapter 创建密码管理器适配器
func NewPasswordManagerAdapter(pm *security.PasswordManager) PasswordManager {
	return &passwordManagerAdapter{PasswordManager: pm}
}
