package docs

import (
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// models.go
// 此文件包含Swagger文档使用的通用数据模型定义
// 这些模型用于API文档生成，提供准确的请求/响应示例

// ==================== 登录认证相关 ====================

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"admin"`          // 用户名
	Password string `json:"password" binding:"required" example:"admin123"`       // 密码
	Code     string `json:"code" binding:"required" example:"1234"`              // 验证码
	UUID     string `json:"uuid" binding:"required" example:"uuid-string"`       // 验证码UUID
}

// LoginResponse 登录响应
type LoginResponse struct {
	AccessToken  string    `json:"accessToken" example:"eyJhbGciOiJIUzI1NiIs..."`  // 访问令牌
	RefreshToken string    `json:"refreshToken" example:"eyJhbGciOiJIUzI1NiIs..."` // 刷新令牌
	ExpiresAt    int64     `json:"expiresAt" example:"1766380800"`               // 过期时间
	User         UserInfo  `json:"user"`                                         // 用户信息
}

// RefreshTokenRequest 刷新令牌请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required" example:"eyJhbGciOiJIUzI1NiIs..."` // 刷新令牌
}

// LogoutRequest 登出请求
type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" example:"eyJhbGciOiJIUzI1NiIs..."` // 刷新令牌（可选）
}

// ==================== 用户相关 ====================

// UserInfo 用户信息
type UserInfo struct {
	ID             string    `json:"id" example:"uuid-string"`                          // 用户ID
	Username       string    `json:"username" example:"admin"`                          // 用户名
	Nickname       *string   `json:"nickname" example:"管理员"`                          // 昵称
	EmployeeNo     *string   `json:"employeeNo" example:"E001"`                         // 工号
	Email          *string   `json:"email" example:"admin@example.com"`                // 邮箱
	Phone          *string   `json:"phone" example:"13800138000"`                       // 手机号
	Avatar         *string   `json:"avatar" example:"http://example.com/avatar.jpg"`   // 头像
	Gender         int       `json:"gender" example:"0"`                               // 性别：0=男，1=女，2=保密
	Status         int       `json:"status" example:"0"`                               // 状态：0=启用，1=禁用
	DeptID         *string   `json:"deptId" example:"dept-uuid"`                       // 部门ID
	DeptName       *string   `json:"deptName" example:"技术部"`                         // 部门名称
	Roles          []RoleInfo `json:"roles"`                                           // 角色列表
	Posts          []PostInfo `json:"posts"`                                           // 岗位列表
	LoginIP        *string   `json:"loginIp" example:"192.168.1.100"`                 // 最后登录IP
	LoginTime      *time.Time `json:"loginTime" example:"2024-01-01T12:00:00Z"`       // 最后登录时间
	PwdUpdateTime  *time.Time `json:"pwdUpdateTime" example:"2024-01-01T12:00:00Z"`   // 密码更新时间
	PwdExpireDays  int       `json:"pwdExpireDays" example:"90"`                      // 密码过期天数
	InitFlag       bool      `json:"initFlag" example:"false"`                        // 是否初始化
	Remark         string    `json:"remark" example:"系统管理员"`                       // 备注
	CreatedAt      time.Time `json:"createdAt" example:"2024-01-01T12:00:00Z"`       // 创建时间
	UpdatedAt      time.Time `json:"updatedAt" example:"2024-01-01T12:00:00Z"`       // 更新时间
}

// UserListRequest 用户列表查询请求
type UserListRequest struct {
	Username  *string `json:"username" example:"admin"`           // 用户名（模糊查询）
	Nickname  *string `json:"nickname" example:"管理员"`           // 昵称（模糊查询）
	Phone     *string `json:"phone" example:"138"`                // 手机号（模糊查询）
	Status    *int    `json:"status" example:"0"`                 // 状态：0=启用，1=禁用
	DeptID    *string `json:"deptId" example:"dept-uuid"`         // 部门ID
	BeginTime *string `json:"beginTime" example:"2024-01-01"`     // 开始时间
	EndTime   *string `json:"endTime" example:"2024-12-31"`       // 结束时间
	Current   int     `json:"current" example:"1"`                // 当前页码
	PageSize  int     `json:"pageSize" example:"10"`              // 每页大小
}

// UserCreateRequest 创建用户请求
type UserCreateRequest struct {
	Username   string       `json:"username" binding:"required" example:"newuser"`      // 用户名
	Password   string       `json:"password" binding:"required" example:"password123"`  // 密码
	Nickname   *string      `json:"nickname" example:"新用户"`                           // 昵称
	EmployeeNo *string      `json:"employeeNo" example:"E002"`                          // 工号
	Email      *string      `json:"email" example:"newuser@example.com"`               // 邮箱
	Phone      *string      `json:"phone" example:"13900139000"`                        // 手机号
	Gender     models.Gender `json:"gender" example:"0"`                                // 性别：0=男，1=女，2=保密
	Status     models.UserStatus `json:"status" example:"0"`                            // 状态：0=启用，1=禁用
	DeptID     *string      `json:"deptId" example:"dept-uuid"`                         // 部门ID
	RoleIds    []string     `json:"roleIds" example:"[\"role-uuid1\",\"role-uuid2\"]"`  // 角色ID列表
	PostIds    []string     `json:"postIds" example:"[\"post-uuid1\"]"`                 // 岗位ID列表
	Remark     *string      `json:"remark" example:"新员工"`                              // 备注
}

// UserUpdateRequest 更新用户请求
type UserUpdateRequest struct {
	ID         string           `json:"id" binding:"required" example:"user-uuid"`        // 用户ID
	Nickname   *string          `json:"nickname" example:"更新后的昵称"`                     // 昵称
	EmployeeNo *string          `json:"employeeNo" example:"E002"`                         // 工号
	Email      *string          `json:"email" example:"user@example.com"`                 // 邮箱
	Phone      *string          `json:"phone" example:"13900139000"`                      // 手机号
	Gender     models.Gender    `json:"gender" example:"0"`                               // 性别
	Status     models.UserStatus `json:"status" example:"0"`                              // 状态
	DeptID     *string          `json:"deptId" example:"dept-uuid"`                       // 部门ID
	RoleIds    []string         `json:"roleIds" example:"[\"role-uuid1\"]"`               // 角色ID列表
	PostIds    []string         `json:"postIds" example:"[\"post-uuid1\"]"`               // 岗位ID列表
	Remark     *string          `json:"remark" example:"更新后的备注"`                       // 备注
}

// UserUpdatePasswordRequest 修改密码请求
type UserUpdatePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required" example:"oldPassword123"`  // 旧密码
	NewPassword string `json:"newPassword" binding:"required" example:"newPassword123"`  // 新密码
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=6,max=20" example:"123456"`  // 新密码
}

// ==================== 角色相关 ====================

// RoleInfo 角色信息
type RoleInfo struct {
	ID                string            `json:"id" example:"role-uuid"`                      // 角色ID
	RoleName          string            `json:"roleName" example:"管理员"`                    // 角色名称
	RoleKey           string            `json:"roleKey" example:"admin"`                     // 角色权限字符串
	RoleSort          int               `json:"roleSort" example:"1"`                        // 显示顺序
	DataScope         models.DataScope  `json:"dataScope" example:"1"`                       // 数据范围：1=全部，2=自定义，3=本部门，4=本部门及以下，5=仅本人
	MenuCheckStrictly bool              `json:"menuCheckStrictly" example:"true"`           // 菜单树选择项是否关联显示
	DeptCheckStrictly bool              `json:"deptCheckStrictly" example:"true"`           // 部门树选择项是否关联显示
	Status            models.RoleStatus `json:"status" example:"0"`                          // 角色状态：0=正常，1=停用
	Remark            string            `json:"remark" example:"超级管理员"`                  // 备注
	CreatedAt         time.Time         `json:"createdAt" example:"2024-01-01T12:00:00Z"`  // 创建时间
	UpdatedAt         time.Time         `json:"updatedAt" example:"2024-01-01T12:00:00Z"`  // 更新时间
}

// RoleListRequest 角色列表查询请求
type RoleListRequest struct {
	RoleName string `json:"roleName" example:"管理员"`            // 角色名称（模糊查询）
	RoleKey  string `json:"roleKey" example:"admin"`             // 角色权限字符串（模糊查询）
	Status   string `json:"status" example:"0"`                  // 角色状态
	Current  int    `json:"current" example:"1"`                 // 当前页码
	PageSize int    `json:"pageSize" example:"10"`               // 每页大小
}

// RoleCreateRequest 创建角色请求
type RoleCreateRequest struct {
	RoleName          string            `json:"roleName" binding:"required" example:"编辑"`              // 角色名称
	RoleKey           string            `json:"roleKey" binding:"required" example:"editor"`             // 角色权限字符串
	RoleSort          int               `json:"roleSort" example:"2"`                                    // 显示顺序
	DataScope         models.DataScope  `json:"dataScope" example:"1"`                                   // 数据范围
	MenuCheckStrictly *bool             `json:"menuCheckStrictly" example:"true"`                        // 菜单树选择项是否关联显示
	DeptCheckStrictly *bool             `json:"deptCheckStrictly" example:"true"`                        // 部门树选择项是否关联显示
	Status            models.RoleStatus `json:"status" example:"0"`                                      // 角色状态
	Remark            *string           `json:"remark" example:"普通角色"`                                 // 备注
	MenuIds           []string          `json:"menuIds" example:"[\"menu-uuid1\",\"menu-uuid2\"]"`       // 菜单ID列表
	DeptIds           []string          `json:"deptIds" example:"[\"dept-uuid1\"]"`                      // 部门ID列表
}

// RoleUpdateRequest 更新角色请求
type RoleUpdateRequest struct {
	ID                string            `json:"id" binding:"required" example:"role-uuid"`               // 角色ID
	RoleName          string            `json:"roleName" binding:"required" example:"编辑"`               // 角色名称
	RoleKey           string            `json:"roleKey" binding:"required" example:"editor"`             // 角色权限字符串
	RoleSort          int               `json:"roleSort" example:"2"`                                    // 显示顺序
	DataScope         models.DataScope  `json:"dataScope" example:"1"`                                   // 数据范围
	MenuCheckStrictly *bool             `json:"menuCheckStrictly" example:"true"`                        // 菜单树选择项是否关联显示
	DeptCheckStrictly *bool             `json:"deptCheckStrictly" example:"true"`                        // 部门树选择项是否关联显示
	Status            models.RoleStatus `json:"status" example:"0"`                                      // 角色状态
	Remark            *string           `json:"remark" example:"更新后的备注"`                               // 备注
	MenuIds           []string          `json:"menuIds" example:"[\"menu-uuid1\"]"`                      // 菜单ID列表
	DeptIds           []string          `json:"deptIds" example:"[\"dept-uuid1\"]"`                      // 部门ID列表
}

// ==================== 部门相关 ====================

// DepartmentInfo 部门信息
type DepartmentInfo struct {
	ID             string            `json:"id" example:"dept-uuid"`                       // 部门ID
	DeptName       string            `json:"deptName" example:"技术部"`                     // 部门名称
	DeptCode       string            `json:"deptCode" example:"TECH"`                      // 部门编码
	ParentID       *string           `json:"parentId" example:"parent-dept-uuid"`          // 父部门ID
	Ancestors      string            `json:"ancestors" example:"0,uuid1,uuid2"`            // 祖级列表
	OrderNum       int               `json:"orderNum" example:"1"`                         // 显示顺序
	Leader         *string           `json:"leader" example:"leader-uuid"`                 // 负责人ID
	LeaderName     *string           `json:"leaderName" example:"张三"`                     // 负责人姓名
	LeaderUsername *string           `json:"leaderUsername" example:"zhangsan"`            // 负责人用户名
	Phone          *string           `json:"phone" example:"010-12345678"`                 // 联系电话
	Email          *string           `json:"email" example:"tech@example.com"`             // 邮箱
	IsExternalOrg  int               `json:"isExternalOrg" example:"0"`                    // 是否为外部机构：0=否，1=是
	Status         models.DeptStatus `json:"status" example:"0"`                           // 部门状态：0=正常，1=停用
	Remark         string            `json:"remark" example:"技术部门"`                     // 备注
	Children       []DepartmentInfo  `json:"children"`                                     // 子部门列表
	CreatedAt      time.Time         `json:"createdAt" example:"2024-01-01T12:00:00Z"`    // 创建时间
	UpdatedAt      time.Time         `json:"updatedAt" example:"2024-01-01T12:00:00Z"`    // 更新时间
}

// DepartmentListRequest 部门列表查询请求
type DepartmentListRequest struct {
	DeptName string `json:"deptName" example:"技术"`                // 部门名称（模糊查询）
	Status   *int   `json:"status" example:"0"`                    // 部门状态
	Current  int    `json:"current" example:"1"`                   // 当前页码
	PageSize int    `json:"pageSize" example:"10"`                 // 每页大小
}

// DepartmentCreateRequest 创建部门请求
type DepartmentCreateRequest struct {
	DeptName      string  `json:"deptName" binding:"required" example:"开发部"`                // 部门名称
	DeptCode      string  `json:"deptCode" binding:"required" example:"DEV"`                  // 部门编码
	ParentID      *string `json:"parentId" example:"parent-dept-uuid"`                         // 父部门ID
	OrderNum      int     `json:"orderNum" example:"1"`                                        // 显示顺序
	Leader        *string `json:"leader" example:"leader-uuid"`                                // 负责人ID
	Phone         *string `json:"phone" example:"010-12345678"`                                // 联系电话
	Email         *string `json:"email" example:"dev@example.com"`                             // 邮箱
	IsExternalOrg int     `json:"isExternalOrg" example:"0"`                                   // 是否为外部机构：0=否，1=是
	Status        models.DeptStatus `json:"status" example:"0"`                                // 部门状态
	Remark        *string `json:"remark" example:"软件开发部门"`                                 // 备注
}

// DepartmentUpdateRequest 更新部门请求
type DepartmentUpdateRequest struct {
	ID            string            `json:"id" binding:"required" example:"dept-uuid"`         // 部门ID
	DeptName      string            `json:"deptName" binding:"required" example:"开发部"`       // 部门名称
	DeptCode      string            `json:"deptCode" binding:"required" example:"DEV"`         // 部门编码
	ParentID      *string           `json:"parentId" example:"parent-dept-uuid"`                // 父部门ID
	OrderNum      int               `json:"orderNum" example:"1"`                               // 显示顺序
	Leader        *string           `json:"leader" example:"leader-uuid"`                       // 负责人ID
	Phone         *string           `json:"phone" example:"010-12345678"`                       // 联系电话
	Email         *string           `json:"email" example:"dev@example.com"`                    // 邮箱
	IsExternalOrg int               `json:"isExternalOrg" example:"0"`                          // 是否为外部机构：0=否，1=是
	Status        models.DeptStatus `json:"status" example:"0"`                                 // 部门状态
	Remark        *string           `json:"remark" example:"更新后的备注"`                         // 备注
}

// ==================== 岗位相关 ====================

// PostInfo 岗位信息
type PostInfo struct {
	ID         string             `json:"id" example:"post-uuid"`                     // 岗位ID
	PostCode   string             `json:"postCode" example:"CTO"`                     // 岗位编码
	PostName   string             `json:"postName" example:"首席技术官"`                // 岗位名称
	PostSort   int                `json:"postSort" example:"1"`                       // 显示顺序
	Status     models.PostStatus  `json:"status" example:"0"`                         // 岗位状态：0=正常，1=停用
	Remark     string             `json:"remark" example:"技术负责人"`                 // 备注
	CreatedAt  time.Time          `json:"createdAt" example:"2024-01-01T12:00:00Z"`  // 创建时间
	UpdatedAt  time.Time          `json:"updatedAt" example:"2024-01-01T12:00:00Z"`  // 更新时间
}

// ==================== 菜单相关 ====================

// MenuInfo 菜单信息
type MenuInfo struct {
	ID                string           `json:"id" example:"menu-uuid"`                       // 菜单ID
	MenuName          string           `json:"menuName" example:"系统管理"`                   // 菜单名称
	ParentID          *string          `json:"parentId" example:"parent-menu-uuid"`          // 父菜单ID
	OrderNum          int              `json:"orderNum" example:"1"`                         // 显示顺序
	Path              *string          `json:"path" example:"/system"`                       // 路由地址
	Component         *string          `json:"component" example:"system/index"`             // 组件路径
	MenuType          models.MenuType  `json:"menuType" example:"M"`                        // 菜单类型：M=目录，C=菜单，F=按钮
	Visible           models.VisibleType `json:"visible" example:"1"`                       // 菜单状态：0=隐藏，1=显示
	Status            models.MenuStatus `json:"status" example:"0"`                         // 菜单状态：0=正常，1=停用
	Perms             *string          `json:"perms" example:"system:user:list"`            // 权限标识
	Icon              *string          `json:"icon" example:"setting"`                      // 菜单图标
	Remark            string           `json:"remark" example:"系统管理目录"`                 // 备注
	Meta              *models.MenuMeta `json:"meta"`                                        // 元数据
	Children          []MenuInfo       `json:"children"`                                     // 子菜单列表
	CreatedAt         time.Time        `json:"createdAt" example:"2024-01-01T12:00:00Z"`   // 创建时间
	UpdatedAt         time.Time        `json:"updatedAt" example:"2024-01-01T12:00:00Z"`   // 更新时间
}

// MenuTreeSelectResponse 菜单树选择器响应
type MenuTreeSelectResponse struct {
	MenuTree    []MenuInfo `json:"menuTree"`                      // 菜单树
	CheckedKeys []string   `json:"checkedKeys"`                   // 已选中的菜单ID列表
}

// ==================== 字典相关 ====================

// DictTypeInfo 字典类型信息
type DictTypeInfo struct {
	ID         string    `json:"id" example:"dict-type-uuid"`            // 字典类型ID
	DictName   string    `json:"dictName" example:"用户状态"`             // 字典名称
	DictType   string    `json:"dictType" example:"sys_user_status"`    // 字典类型
	Status     int       `json:"status" example:"0"`                    // 状态：0=正常，1=停用
	Remark     string    `json:"remark" example:"用户状态列表"`           // 备注
	CreatedAt  time.Time `json:"createdAt" example:"2024-01-01T12:00:00Z"` // 创建时间
	UpdatedAt  time.Time `json:"updatedAt" example:"2024-01-01T12:00:00Z"` // 更新时间
}

// DictDataInfo 字典数据信息
type DictDataInfo struct {
	ID          string    `json:"id" example:"dict-data-uuid"`                       // 字典数据ID
	DictSort    int       `json:"dictSort" example:"0"`                              // 字典排序
	DictLabel   string    `json:"dictLabel" example:"正常"`                           // 字典标签
	DictValue   string    `json:"dictValue" example:"0"`                             // 字典键值
	DictType    string    `json:"dictType" example:"sys_user_status"`                // 字典类型
	CSSClass    *string   `json:"cssClass" example:""`                               // 样式属性
	ListClass   *string   `json:"listClass" example:"default"`                       // 表格回显样式
	IsDefault   bool      `json:"isDefault" example:"true"`                          // 是否默认
	Status      int       `json:"status" example:"0"`                                // 状态：0=正常，1=停用
	Remark      string    `json:"remark" example:"正常状态"`                          // 备注
	CreatedAt   time.Time `json:"createdAt" example:"2024-01-01T12:00:00Z"`         // 创建时间
	UpdatedAt   time.Time `json:"updatedAt" example:"2024-01-01T12:00:00Z"`         // 更新时间
}

// ==================== 通知公告相关 ====================

// NoticeInfo 通知公告信息
type NoticeInfo struct {
	ID          string           `json:"id" example:"notice-uuid"`                      // 公告ID
	NoticeTitle string           `json:"noticeTitle" example:"系统通知"`                  // 公告标题
	NoticeType  int              `json:"noticeType" example:"1"`                        // 公告类型：1=通知，2=公告
	Status      models.MenuStatus `json:"status" example:"0"`                            // 公告状态：0=正常，1=关闭
	Remark      *string          `json:"remark" example:"系统维护通知"`                   // 备注
	Content     *string          `json:"content" example:"系统将于今晚进行维护..."`        // 公告内容
	CreatedAt   time.Time        `json:"createdAt" example:"2024-01-01T12:00:00Z"`     // 创建时间
	UpdatedAt   time.Time        `json:"updatedAt" example:"2024-01-01T12:00:00Z"`     // 更新时间
}

// ==================== 配置相关 ====================

// ConfigInfo 配置信息
type ConfigInfo struct {
	ID          string             `json:"id" example:"config-uuid"`                       // 配置ID
	ConfigName  string             `json:"configName" example:"主框架页-默认皮肤样式名称"`     // 参数名称
	ConfigKey   string             `json:"configKey" example:"sys.index.skinName"`         // 参数键名
	ConfigValue string             `json:"configValue" example:"skin-blue"`                // 参数键值
	ConfigType  models.ConfigType  `json:"configType" example:"Y"`                         // 系统内置：Y=是，N=否
	IsSystem    models.ConfigIsSystem `json:"isSystem" example:"0"`                         // 是否系统内置：0=否，1=是
	Remark      string             `json:"remark" example:"主框架页默认皮肤样式"`             // 备注
	CreatedAt   time.Time          `json:"createdAt" example:"2024-01-01T12:00:00Z"`      // 创建时间
	UpdatedAt   time.Time          `json:"updatedAt" example:"2024-01-01T12:00:00Z"`      // 更新时间
}
