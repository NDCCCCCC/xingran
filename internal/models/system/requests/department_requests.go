package requests

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

type DepartmentListParams struct {
	DeptName string `json:"deptName"`
	Status   *int   `json:"status"`
}

type DepartmentCreateRequest struct {
	DeptName      string            `json:"deptName" binding:"required"`
	DeptCode      string            `json:"deptCode" binding:"required"`
	ParentID      *string           `json:"parentId"`
	OrderNum      int               `json:"orderNum"`
	Leader        *string           `json:"leader"`
	Phone         *string           `json:"phone"`
	Email         *string           `json:"email"`
	IsExternalOrg int               `json:"isExternalOrg"`
	Status        models.DeptStatus `json:"status"`
	Remark        *string           `json:"remark"`
}

func (r *DepartmentCreateRequest) ToModel(ancestors string) models.Department {
	return models.Department{
		DeptName:      r.DeptName,
		DeptCode:      r.DeptCode,
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

type DepartmentUpdateRequest struct {
	ID            string            `json:"id"` // ID 从 URL 参数获取，不在请求体验证
	DeptName      string            `json:"deptName" binding:"required"`
	DeptCode      string            `json:"deptCode" binding:"required"`
	ParentID      *string           `json:"parentId"`
	OrderNum      int               `json:"orderNum"`
	Leader        *string           `json:"leader"`
	Phone         *string           `json:"phone"`
	Email         *string           `json:"email"`
	IsExternalOrg int               `json:"isExternalOrg"`
	Status        models.DeptStatus `json:"status"`
	Remark        *string           `json:"remark"`
}
