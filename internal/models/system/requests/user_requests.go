package requests

import (
	"github.com/ruoyi-next/ruoyi-go-backend/internal/constants"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// UserListParams 用户列表查询参数
type UserListParams struct {
	Username  *string `json:"username"`
	Nickname  *string `json:"nickname"`
	Phone     *string `json:"phone"`
	Status    *int    `json:"status"`
	DeptID    *string `json:"deptId"`
	BeginTime *string `json:"beginTime"`
	EndTime   *string `json:"endTime"`
	Current   int     `json:"current"`
	PageSize  int     `json:"pageSize"`
}

// DefaultUserListParams 默认列表参数
func DefaultUserListParams() UserListParams {
	return UserListParams{
		Current:  constants.DefaultCurrent,
		PageSize: constants.DefaultPageSize,
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
	Username   string    `json:"username" binding:"required"`
	Password   string    `json:"password" binding:"required"`
	Nickname   *string   `json:"nickname"`
	EmployeeNo *string   `json:"employeeNo"`
	Email      *string   `json:"email"`
	Phone      *string   `json:"phone"`
	Gender     models.Gender `json:"gender"`
	Status     models.UserStatus `json:"status"`
	DeptID     *string   `json:"deptId"`
	RoleIds    []string  `json:"roleIds"`
	PostIds    []string  `json:"postIds"`
	Remark     *string   `json:"remark"`
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
	ID         string          `json:"id" binding:"required"`
	Nickname   *string        `json:"nickname"`
	EmployeeNo *string        `json:"employeeNo"`
	Email      *string        `json:"email"`
	Phone      *string        `json:"phone"`
	Gender     models.Gender  `json:"gender"`
	Status     models.UserStatus `json:"status"`
	DeptID     *string        `json:"deptId"`
	RoleIds    []string       `json:"roleIds"`
	PostIds    []string       `json:"postIds"`
	Remark     *string        `json:"remark"`
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
