/**
 * 状态值常量
 *
 * 定义项目中通用的状态值
 * 注意：0 = 启用/正常/可见, 1 = 禁用/停止/隐藏
 * 特殊例外：visible 字段 (1 = 可见, 0 = 隐藏)
 */
package constants

// 用户状态
const (
	// UserStatusEnabled 用户启用状态
	UserStatusEnabled = 0
	// UserStatusDisabled 用户禁用状态
	UserStatusDisabled = 1
)

// 角色状态
const (
	// RoleStatusEnabled 角色正常状态
	RoleStatusEnabled = 0
	// RoleStatusDisabled 角色停用状态
	RoleStatusDisabled = 1
)

// 菜单状态
const (
	// MenuStatusNormal 菜单正常状态
	MenuStatusNormal = 0
	// MenuStatusStop 菜单停用状态
	MenuStatusStop = 1
)

// 菜单可见性（布尔语义，与其他状态相反）
const (
	// VisibleHidden 菜单隐藏
	VisibleHidden = 0
	// VisibleShow 菜单显示
	VisibleShow = 1
)

// 部门状态
const (
	// DeptStatusNormal 部门正常状态
	DeptStatusNormal = 0
	// DeptStatusStop 部门停用状态
	DeptStatusStop = 1
)

// 岗位状态
const (
	// PostStatusNormal 岗位正常状态
	PostStatusNormal = 0
	// PostStatusStop 岗位停用状态
	PostStatusStop = 1
)

// 工位状态
const (
	// WorkstationStatusAvailable 工位空闲状态
	WorkstationStatusAvailable = 0
	// WorkstationStatusOccupied 工位占用状态
	WorkstationStatusOccupied = 1
)

// 通知发布状态
const (
	// NoticeStatusPublished 通知已发布
	NoticeStatusPublished = 0
	// NoticeStatusDraft 通知草稿
	NoticeStatusDraft = 1
)

// 定时任务状态
const (
	// JobStatusNormal 定时任务正常
	JobStatusNormal = 0
	// JobStatusPaused 定时任务暂停
	JobStatusPaused = 1
)

// 定时任务执行状态
const (
	// JobLogStatusSuccess 任务执行成功
	JobLogStatusSuccess = 0
	// JobLogStatusFailure 任务执行失败
	JobLogStatusFailure = 1
)

// 设备发现状态
const (
	// DiscoveryStatusPending 设备发现待执行
	DiscoveryStatusPending = 0
	// DiscoveryStatusRunning 设备发现执行中
	DiscoveryStatusRunning = 1
)

// 验证码背景状态
const (
	// CaptchaBgDisabled 验证码背景禁用
	CaptchaBgDisabled = 0
	// CaptchaBgEnabled 验证码背景启用
	CaptchaBgEnabled = 1
)

// 仪表板状态
const (
	// DashboardStatusNormal 仪表板正常
	DashboardStatusNormal = 0
	// DashboardStatusDisabled 仪表板禁用
	DashboardStatusDisabled = 1
)

// 值班状态
const (
	// DutyStatusNormal 值班正常
	DutyStatusNormal = 0
	// DutyStatusException 值班异常
	DutyStatusException = 1
)

// API 配置状态
const (
	// APIConfigEnabled API 配置启用
	APIConfigEnabled = 0
	// APIConfigDisabled API 配置禁用
	APIConfigDisabled = 1
)

// 操作日志状态
const (
	// OperLogStatusSuccess 操作成功
	OperLogStatusSuccess = 0
	// OperLogStatusFailure 操作失败
	OperLogStatusFailure = 1
)

// HTTP 状态码相关
const (
	// HTTPStatusSuccess HTTP 成功状态码范围起始
	HTTPStatusSuccess = 200
	// HTTPStatusSuccessEnd HTTP 成功状态码范围结束
	HTTPStatusSuccessEnd = 299
	// HTTPStatusClientError HTTP 客户端错误状态码范围起始
	HTTPStatusClientError = 400
	// HTTPStatusServerError HTTP 服务端错误状态码范围起始
	HTTPStatusServerError = 500
)
