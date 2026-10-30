package requests

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// MenuListParams 菜单列表查询参数
type MenuListParams struct {
	MenuName string `json:"menuName"`
	Status   string `json:"status"`
}

// MenuCreateRequest 创建菜单请求
type MenuCreateRequest struct {
	MenuName  string             `json:"menuName" binding:"required"`
	ParentID  *string            `json:"parentId"`
	OrderNum  int                `json:"orderNum"`
	Path      *string            `json:"path"`
	Component *string            `json:"component"`
	MenuType  models.MenuType    `json:"menuType"`
	Visible   models.VisibleType `json:"visible"`
	Status    models.MenuStatus  `json:"status"`
	Perms     *string            `json:"perms"`
	Icon      *string            `json:"icon"`
	Remark    *string            `json:"remark"`
}

// ToModel 转换为菜单模型
func (r *MenuCreateRequest) ToModel() models.Menu {
	return models.Menu{
		MenuName:  r.MenuName,
		ParentID:  normalizeParentID(r.ParentID),
		OrderNum:  r.OrderNum,
		Path:      r.Path,
		Component: r.Component,
		MenuType:  r.MenuType,
		Visible:   r.Visible,
		Status:    r.Status,
		Perms:     r.Perms,
		Icon:      r.Icon,
		Remark:    stringPtrValue(r.Remark),
	}
}

// MenuUpdateRequest 更新菜单请求
type MenuUpdateRequest struct {
	ID        string             `json:"id"` // ID 从 URL 参数获取，不在请求体验证
	MenuName  string             `json:"menuName" binding:"required"`
	ParentID  *string            `json:"parentId"`
	OrderNum  int                `json:"orderNum"`
	Path      *string            `json:"path"`
	Component *string            `json:"component"`
	MenuType  models.MenuType    `json:"menuType"`
	Visible   models.VisibleType `json:"visible"`
	Status    models.MenuStatus  `json:"status"`
	Perms     *string            `json:"perms"`
	Icon      *string            `json:"icon"`
	Remark    *string            `json:"remark"`
}

// normalizeParentID 处理空字符串的 ParentID
func normalizeParentID(parentID *string) *string {
	if parentID == nil || *parentID == "" || *parentID == "0" {
		return nil
	}
	return parentID
}

// stringPtrValue 获取字符串指针的值
func stringPtrValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
