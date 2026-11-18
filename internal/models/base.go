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

// OrgLevel 机构层级枚举（sys_dept.org_level）
//
// 业务背景：系统三层机构架构「分公司 → 中支 → 支公司」。部门树用 ancestors
// 物化路径表达上下级，但节点自身**没有**层级语义；`is_external_org` 只回答
// 「有没有职场」，`sys_manager_org.org_type` 属于经理同步域。数据权限需要
// 「这是第几级机构」这个判据来定位锚点，故在 sys_dept 上显式标注。
//
// 0=非机构（普通部门/科室，默认值，保证存量数据零回归）。
//
// 注意：本字段**只**用于数据权限锚点定位，不参与任何其它业务分支判定。
type OrgLevel int

const (
	OrgLevelNone    OrgLevel = 0 // 非机构（普通部门/科室）
	OrgLevelBranch  OrgLevel = 1 // 分公司（如 湖北分公司）
	OrgLevelSubBang OrgLevel = 2 // 中支（如 武汉中支）
	OrgLevelOffice  OrgLevel = 3 // 支公司（如 硚口支公司）
	OrgLevelSite    OrgLevel = 4 // 网点/班组（预留）
)

// AnchorMode 数据权限锚点模式（sys_role.anchor_mode / sys_post.anchor_mode）
//
// 定位：与 DataScope 正交的**独立维度**，决定"可见集的锚点从哪来"。
// 绝不可并入 DataScope 枚举 —— getUserMaxDataScope 用 MAX(data_scope) 取
// 数值最大（最窄）档，新增更高数值会把 All(1) 用户静默收窄（静默降权事故）。
// 故锚点维度独立解析后与角色维度**取并集（宽者胜）**。
//
// 值 1（相对层深）与 4（sys_manager_org 经理机构）已论证弃用：
//   - 1 被 5 取代：固定深度无法让一个岗位自适应三层，且依赖树深度规整；
//   - 4 依赖未治理的机构表（无种子数据，空表时全员未命中）。
//
// 编号保留原始设计值以便追溯；保存时收到 1/4 必须拒绝。
type AnchorMode int

const (
	AnchorModeNone        AnchorMode = 0 // 不参与机构授权（默认）
	AnchorModeDept        AnchorMode = 2 // 显式锚点：anchor_param = dept_id，展开其子树
	AnchorModeExternalOrg AnchorMode = 3 // 沿 ancestors 由近到远找第一个 is_external_org=1 的节点
	AnchorModeLevel       AnchorMode = 5 // 沿 ancestors 由近到远找第一个 org_level == anchor_param 的节点
)

// IsValidAnchorMode 判断锚点模式是否在已实现集合内（0/2/3/5）。
// 1 与 4 为弃用编号，恒返回 false。
func IsValidAnchorMode(m AnchorMode) bool {
	switch m {
	case AnchorModeNone, AnchorModeDept, AnchorModeExternalOrg, AnchorModeLevel:
		return true
	default:
		return false
	}
}
