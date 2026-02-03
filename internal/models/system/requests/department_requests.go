package requests

import (
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// DepartmentListParams 部门列表查询参数
type DepartmentListParams struct {
	DeptName string `json:"deptName"`
	Status   string `json:"status"`
}

// DepartmentCreateRequest 创建部门请求
type DepartmentCreateRequest struct {
	DeptName      string              `json:"deptName" binding:"required"`
	ParentID      *string             `json:"parentId"`
	OrderNum      int                 `json:"orderNum"`
	Leader        *string             `json:"leader"`
	Phone         *string             `json:"phone"`
	Email         *string             `json:"email"`
	IsExternalOrg int                 `json:"isExternalOrg"`
	Status        models.DeptStatus   `json:"status"`
	Remark        *string             `json:"remark"`
}

// ToModel 转换为部门模型
func (r *DepartmentCreateRequest) ToModel(ancestors string) models.Department {
	return models.Department{
		DeptName:      r.DeptName,
		ParentID:      r.ParentID,
		Ancestors:     ancestors,
		OrderNum:      r.OrderNum,
		Leader:        r.Leader,
		Phone:         r.Phone,
		Email:         r.Email,
		IsExternalOrg: r.IsExternalOrg,
		Status:        r.Status,
		Remark:        toStringPtr(r.Remark),
	}
}

// DepartmentUpdateRequest 更新部门请求
type DepartmentUpdateRequest struct {
	ID            string              `json:"id" binding:"required"`
	DeptName      string              `json:"deptName" binding:"required"`
	ParentID      *string             `json:"parentId"`
	OrderNum      int                 `json:"orderNum"`
	Leader        *string             `json:"leader"`
	Phone         *string             `json:"phone"`
	Email         *string             `json:"email"`
	IsExternalOrg int                 `json:"isExternalOrg"`
	Status        models.DeptStatus   `json:"status"`
	Remark        *string             `json:"remark"`
}
