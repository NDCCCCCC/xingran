package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel 基础模型
type BaseModel struct {
	ID        string         `gorm:"type:uuid;primary_key" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
	CreatedBy string         `gorm:"size:64" json:"createdBy"`
	UpdatedBy string         `gorm:"size:64" json:"updatedBy"`
	Version   int            `json:"version"`
}

// BeforeCreate GORM钩子 - 创建前
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}

// BaseTimeLine 时间线基础模型
type BaseTimeLine struct {
	ID        string    `gorm:"type:uuid;primary_key" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// BeforeCreate GORM钩子 - 创建前
func (b *BaseTimeLine) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}

// Gender 性别枚举
type Gender int

const (
	GenderMale   Gender = 0 // 男
	GenderFemale Gender = 1 // 女
	GenderSecret Gender = 2 // 保密
)

// UserStatus 用户状态枚举
type UserStatus int

const (
	UserStatusEnabled  UserStatus = 0 // 启用
	UserStatusDisabled UserStatus = 1 // 禁用
)

// RoleStatus 角色状态枚举
type RoleStatus int

const (
	RoleStatusEnabled  RoleStatus = 0 // 正常
	RoleStatusDisabled RoleStatus = 1 // 停用
)

// DataScope 数据范围枚举
type DataScope int

const (
	DataScopeAll       DataScope = 1 // 全部数据
	DataScopeCustom    DataScope = 2 // 自定义数据
	DataScopeDept      DataScope = 3 // 本部门数据
	DataScopeDeptChild DataScope = 4 // 本部门及子部门数据
	DataScopeSelf      DataScope = 5 // 仅本人数据
)

// MenuType 菜单类型枚举
type MenuType string

const (
	MenuTypeDir    MenuType = "M" // 目录
	MenuTypeMenu   MenuType = "C" // 菜单
	MenuTypeButton MenuType = "F" // 按钮
)

// VisibleType 显示状态枚举
type VisibleType int

const (
	VisibleShow   VisibleType = 1 // 显示
	VisibleHidden VisibleType = 0 // 隐藏
)

// MenuStatus 菜单状态枚举
type MenuStatus int

const (
	MenuStatusNormal MenuStatus = 0 // 正常
	MenuStatusStop   MenuStatus = 1 // 停用
)

// DeptStatus 部门状态枚举
type DeptStatus int

const (
	DeptStatusNormal DeptStatus = 0 // 正常
	DeptStatusStop   DeptStatus = 1 // 停用
)

// PostStatus 岗位状态枚举
type PostStatus int

const (
	PostStatusEnabled  PostStatus = 0 // 正常
	PostStatusDisabled PostStatus = 1 // 停用
)

// ConfigType 配置类型枚举
type ConfigType string

const (
	ConfigTypeYes ConfigType = "Y" // 是
	ConfigTypeNo  ConfigType = "N" // 否
)

// ConfigIsSystem 系统内置枚举
type ConfigIsSystem int

const (
	ConfigIsSystemNo  ConfigIsSystem = 0 // 否
	ConfigIsSystemYes ConfigIsSystem = 1 // 是
)
