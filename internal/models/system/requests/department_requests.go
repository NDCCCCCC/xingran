package requests

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

type DepartmentListParams struct {
	DeptName string `json:"deptName"`
	Status   *int   `json:"status"`
	// Scoped 请求按当前登录人的数据范围（机构锚点/数据范围档位）收口部门树。
	//
	// 显式 opt-in 设计（默认 false）：部门树被大量非管理场景复用——角色「自定义
	// 数据范围」勾选树、岗位锚点部门选择、ops 页面选择器等**必须看到全树**才能
	// 正确配置；只有「部门管理」「系统用户」这类管理视图传 scoped=true。
	// 未登录上下文/超管（All）不受影响；解析失败 fail-closed 空树。
	Scoped bool `json:"scoped"`
}

type DepartmentCreateRequest struct {
	DeptName      string  `json:"deptName" binding:"required"`
	DeptCode      string  `json:"deptCode" binding:"required"`
	ParentID      *string `json:"parentId"`
	OrderNum      int     `json:"orderNum"`
	Leader        *string `json:"leader"`
	Phone         *string `json:"phone"`
	Email         *string `json:"email"`
	IsExternalOrg int     `json:"isExternalOrg"`
	// OrgLevel 机构层级：0=非机构（普通部门/科室）1=分公司 2=中支 3=支公司 4=网点。
	// 存量表单不传该字段时零值 0 = 非机构，零回归。
	OrgLevel int               `json:"orgLevel"`
	Status   models.DeptStatus `json:"status"`
	Remark   *string           `json:"remark"`
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
		OrgLevel:      models.OrgLevel(r.OrgLevel),
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
	OrgLevel      int               `json:"orgLevel"`
	Status        models.DeptStatus `json:"status"`
	Remark        *string           `json:"remark"`
}
