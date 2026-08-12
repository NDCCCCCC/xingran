package requests

import (
	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// UserListParams 用户列表查询参数
// 嵌入 base.BaseListRequest 使 json 顶层自动获得 current/pageSize/orderByColumn/isAsc 字段。
// promoted field 保证 p.Current / p.PageSize 访问仍能 work,GetPagination/GetOffset 方法无需改动。
type UserListParams struct {
	base.BaseListRequest
	Username        *string `json:"username"`
	Nickname        *string `json:"nickname"`
	Phone           *string `json:"phone"`
	Status          *int    `json:"status"`
	DeptID          *string `json:"deptId"`
	// RecursiveDeptID 非空时,把查询扩展为该部门及所有子部门的用户。
	// 与 DeptID 并列存在:DeptID 保持单部门等值语义,RecursiveDeptID 用于
	// 工位管理"所属部门"下拉需要"当前部门+子部门"递归查询的场景。
	// 复用 sys_dept.ancestors 的 4-条件模式(参见 building_service.go applyDeptFilter)。
	RecursiveDeptID *string `json:"recursiveDeptId"`
	BeginTime       *string `json:"beginTime"`
	EndTime         *string `json:"endTime"`
}

// DefaultUserListParams 默认列表参数
func DefaultUserListParams() UserListParams {
	return UserListParams{
		BaseListRequest: base.BaseListRequest{
			Current:  constants.DefaultCurrent,
			PageSize: constants.DefaultPageSize,
		},
	}
}

// GetPagination 获取分页参数
func (p *UserListParams) GetPagination() (current, pageSize int) {
	current = p.Current
	if current < 1 {
		current = constants.DefaultCurrent
	}
	pageSize = p.PageSize
	if pageSize < constants.MinPageSize {
		pageSize = constants.DefaultPageSize
	}
	if pageSize > constants.MaxPageSize {
		pageSize = constants.MaxPageSize
	}
	return current, pageSize
}

// GetOffset 计算偏移量
func (p *UserListParams) GetOffset() int {
	current, pageSize := p.GetPagination()
	return (current - 1) * pageSize
}

// UserCreateRequest 创建用户请求
type UserCreateRequest struct {
	Username   string            `json:"username" binding:"required"`
	Password   string            `json:"password" binding:"required"`
	Nickname   *string           `json:"nickname"`
	EmployeeNo *string           `json:"employeeNo"`
	Email      *string           `json:"email"`
	Phone      *string           `json:"phone"`
	Gender     models.Gender     `json:"gender"`
	Status     models.UserStatus `json:"status"`
	DeptID     *string           `json:"deptId"`
	RoleIds    []string          `json:"roleIds"`
	PostIds    []string          `json:"postIds"`
	Remark     *string           `json:"remark"`
}

// ToModel 转换为用户模型
func (r *UserCreateRequest) ToModel(hashedPassword string) models.User {
	return models.User{
		Username:   r.Username,
		Password:   hashedPassword,
		Nickname:   r.Nickname,
		EmployeeNo: r.EmployeeNo,
		Email:      r.Email,
		Phone:      r.Phone,
		Gender:     r.Gender,
		Status:     r.Status,
		DeptID:     r.DeptID,
		Remark:     toStringPtr(r.Remark),
	}
}

// UserUpdateRequest 更新用户请求
type UserUpdateRequest struct {
	ID         string            `json:"id"` // ID 从 URL 参数获取，不在请求体验证
	Nickname   *string           `json:"nickname"`
	EmployeeNo *string           `json:"employeeNo"`
	Email      *string           `json:"email"`
	Phone      *string           `json:"phone"`
	Gender     models.Gender     `json:"gender"`
	Status     models.UserStatus `json:"status"`
	DeptID     *string           `json:"deptId"`
	RoleIds    []string          `json:"roleIds"`
	PostIds    []string          `json:"postIds"`
	Remark     *string           `json:"remark"`
}

// ToModel 转换为用户模型
func (r *UserUpdateRequest) ToModel() models.User {
	return models.User{
		BaseModel:  models.BaseModel{ID: r.ID},
		Nickname:   r.Nickname,
		EmployeeNo: r.EmployeeNo,
		Email:      r.Email,
		Phone:      r.Phone,
		Gender:     r.Gender,
		Status:     r.Status,
		DeptID:     r.DeptID,
		Remark:     toStringPtr(r.Remark),
	}
}

// toStringPtr 将字符串转换为指针
func toStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
