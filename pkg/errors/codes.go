package errors

import "net/http"

// ErrorCode 业务错误码类型
type ErrorCode int

// 错误码定义（按模块划分）
const (
	CodeSuccess ErrorCode = 0

	// 1000-1999 系统级通用错误
	CodeParamError       ErrorCode = 1001 // 参数错误
	CodeParamMissing     ErrorCode = 1002 // 参数缺失
	CodeParamInvalid     ErrorCode = 1003 // 参数无效
	CodeRecordNotFound   ErrorCode = 1010 // 记录不存在
	CodeRecordExists     ErrorCode = 1011 // 记录已存在
	CodeDatabaseError    ErrorCode = 1014 // 数据库错误
	CodeUnauthorized     ErrorCode = 1020 // 未授权
	CodeTokenExpired     ErrorCode = 1021 // 令牌过期
	CodeTokenInvalid     ErrorCode = 1022 // 令牌无效
	CodePermissionDenied ErrorCode = 1025 // 权限不足
	CodeServerError      ErrorCode = 1500 // 服务器内部错误
	CodeNotImplemented   ErrorCode = 1501 // 功能未实现

	// 2000-2999 用户权限模块
	// 用户相关 2010-2039
	CodeUserNotFound   ErrorCode = 2010 // 用户不存在
	CodeUserExists     ErrorCode = 2011 // 用户已存在
	CodeUserDisabled   ErrorCode = 2012 // 用户已禁用
	CodeUserPassword   ErrorCode = 2013 // 密码错误
	CodeUserHasRoles   ErrorCode = 2014 // 用户已分配角色
	CodeUserDeleteSelf ErrorCode = 2015 // 不能删除自己

	// 角色相关 2040-2069
	CodeRoleNotFound ErrorCode = 2040 // 角色不存在
	CodeRoleExists   ErrorCode = 2041 // 角色已存在
	CodeRoleHasUsers ErrorCode = 2042 // 角色已分配用户
	CodeRoleHasMenus ErrorCode = 2043 // 角色已分配菜单
	CodeRoleHasDepts ErrorCode = 2044 // 角色已分配部门
	CodeRoleIsAdmin  ErrorCode = 2045 // 不能修改管理员角色
	CodeRoleIsSuper  ErrorCode = 2046 // 不能修改超级管理员角色

	// 部门相关 2070-2099
	CodeDeptNotFound    ErrorCode = 2070 // 部门不存在
	CodeDeptExists      ErrorCode = 2071 // 部门已存在
	CodeDeptHasUsers    ErrorCode = 2072 // 部门存在用户
	CodeDeptHasChildren ErrorCode = 2073 // 部门存在子部门
	CodeDeptHasRoles    ErrorCode = 2074 // 部门已分配角色
	CodeDeptInvalid     ErrorCode = 2075 // 无效的部门
	CodeDeptIsParent    ErrorCode = 2076 // 不能选择父部门

	// 岗位相关 2100-2119
	CodePostNotFound ErrorCode = 2100 // 岗位不存在
	CodePostExists   ErrorCode = 2101 // 岗位已存在
	CodePostHasUsers ErrorCode = 2102 // 岗位已分配用户

	// 菜单相关 2120-2149
	CodeMenuNotFound    ErrorCode = 2120 // 菜单不存在
	CodeMenuExists      ErrorCode = 2121 // 菜单已存在
	CodeMenuHasChildren ErrorCode = 2122 // 菜单存在子菜单
	CodeMenuHasRoles    ErrorCode = 2123 // 菜单已分配角色
	CodeMenuInvalid     ErrorCode = 2124 // 无效的菜单
	CodeMenuIsParent    ErrorCode = 2125 // 不能选择父菜单
	CodeMenuTypeInvalid ErrorCode = 2126 // 菜单类型无效
	CodeMenuHasButtons  ErrorCode = 2127 // 菜单存在按钮

	// 字典相关 2150-2169
	CodeDictTypeNotFound ErrorCode = 2150 // 字典类型不存在
	CodeDictTypeExists   ErrorCode = 2151 // 字典类型已存在
	CodeDictDataNotFound ErrorCode = 2152 // 字典数据不存在
	CodeDictDataExists   ErrorCode = 2153 // 字典数据已存在
	CodeDictTypeInUse    ErrorCode = 2154 // 字典类型正在使用

	// 参数配置相关 2170-2189
	CodeConfigNotFound ErrorCode = 2170 // 参数配置不存在
	CodeConfigExists   ErrorCode = 2171 // 参数配置已存在
	CodeConfigReserved ErrorCode = 2172 // 系统保留配置

	// 通知公告相关 2190-2209
	CodeNoticeNotFound ErrorCode = 2190 // 通知公告不存在

	// AD域管理相关 2210-2229
	CodeADConfigNotFound   ErrorCode = 2210 // AD配置不存在
	CodeADConfigExists     ErrorCode = 2211 // AD配置已存在
	CodeADConnectionFailed ErrorCode = 2212 // AD连接失败
	CodeADSyncFailed       ErrorCode = 2213 // AD同步失败

	// 系统设置相关 2230-2249
	CodeSettingsNotFound  ErrorCode = 2230 // 系统设置不存在
	CodeDashboardNotFound ErrorCode = 2240 // 仪表盘不存在

	// 3000-3999 运维管理模块
	// 楼宇相关 3010-3039
	CodeBuildingNotFound   ErrorCode = 3010 // 楼宇不存在
	CodeBuildingExists     ErrorCode = 3011 // 楼宇已存在
	CodeBuildingHasFloors  ErrorCode = 3012 // 楼宇存在楼层
	CodeBuildingInvalid    ErrorCode = 3013 // 无效的楼宇
	CodeBuildingOrgInvalid ErrorCode = 3016 // 关联组织无效

	// 楼层相关 3040-3069
	CodeFloorNotFound        ErrorCode = 3040 // 楼层不存在
	CodeFloorExists          ErrorCode = 3041 // 楼层已存在
	CodeFloorHasWorkstations ErrorCode = 3042 // 楼层存在工位
	CodeFloorInvalid         ErrorCode = 3043 // 无效的楼层

	// 工位相关 3070-3099
	CodeWorkstationNotFound ErrorCode = 3070 // 工位不存在
	CodeWorkstationExists   ErrorCode = 3071 // 工位已存在
	CodeWorkstationInvalid  ErrorCode = 3072 // 无效的工位

	// 机房相关 3100-3129
	CodeServerRoomNotFound   ErrorCode = 3100 // 机房不存在
	CodeServerRoomExists     ErrorCode = 3101 // 机房已存在
	CodeServerRoomHasServers ErrorCode = 3102 // 机房存在服务器
	CodeRoomDeviceNotFound   ErrorCode = 3110 // 机房设备不存在
	CodeRoomDeviceCodeExists ErrorCode = 3111 // 机房设备编码已存在

	// 信息点相关 3130-3159
	CodeInfoPointNotFound ErrorCode = 3130 // 信息点不存在
	CodeInfoPointExists   ErrorCode = 3131 // 信息点已存在

	// 门相关 3160-3169
	CodeDoorNotFound ErrorCode = 3160 // 门不存在
	CodeDoorExists   ErrorCode = 3161 // 门已存在

	// 墙体相关 3170-3179
	CodeWallNotFound ErrorCode = 3170 // 墙体不存在
	CodeWallExists   ErrorCode = 3171 // 墙体已存在

	// 专线相关 3180-3189
	CodeDedicatedLineNotFound ErrorCode = 3180 // 专线不存在
	CodeDedicatedLineExists   ErrorCode = 3181 // 专线已存在

	// 楼层平面图文本相关 3190-3199
	CodeFloorPlanTextNotFound ErrorCode = 3190 // 楼层平面图文本不存在

	// 机房照片相关 3200-3209
	CodeRoomPhotoNotFound ErrorCode = 3200 // 机房照片不存在
	CodeRoomPhotoExists   ErrorCode = 3201 // 机房照片已存在

	// 4000-4999 调度任务模块
	CodeJobNotFound  ErrorCode = 4000 // 定时任务不存在
	CodeJobExists    ErrorCode = 4001 // 定时任务已存在
	CodeJobIsRunning ErrorCode = 4002 // 定时任务正在运行
	CodeJobHasCron   ErrorCode = 4003 // 定时任务已分配Cron
	CodeCronInvalid  ErrorCode = 4004 // Cron表达式无效

	// 5000-5999 工单模块
	CodeWorkorderNotFound ErrorCode = 5000 // 工单不存在
	CodeWorkorderExists   ErrorCode = 5001 // 工单已存在
	CodeWorkorderInvalid  ErrorCode = 5002 // 无效的工单
	CodeWorkorderStatus   ErrorCode = 5003 // 工单状态不允许该操作

	// 6000-6999 监控模块
	CodeMonitorDataNotFound  ErrorCode = 6000 // 监控数据不存在
	CodeCacheKeyNotFound     ErrorCode = 6010 // 缓存键不存在
	CodeCacheOperationFailed ErrorCode = 6011 // 缓存操作失败
	CodeServerInfoNotFound   ErrorCode = 6020 // 服务器信息不存在

	// 7000-7999 网络设备模块
	CodeNetworkDeviceNotFound ErrorCode = 7000 // 网络设备不存在
	CodeNetworkDeviceExists   ErrorCode = 7001 // 网络设备已存在
	CodeNetworkDeviceConnect  ErrorCode = 7002 // 网络设备连接失败
	CodeTemplateNotFound      ErrorCode = 7010 // 模板不存在
	CodeTemplateExists        ErrorCode = 7011 // 模板已存在
	CodeCredentialNotFound    ErrorCode = 7020 // 凭证不存在
	CodeCredentialExists      ErrorCode = 7021 // 凭证已存在
	CodeCommandFailed         ErrorCode = 7030 // 命令执行失败
	CodeBackupNotFound        ErrorCode = 7040 // 备份不存在
	CodePortCollectionFailed  ErrorCode = 7050 // 端口采集失败
	CodeDiscoveryFailed       ErrorCode = 7060 // 设备发现失败

	// 8000-8999 知识库模块
	CodeKnowledgeNotFound ErrorCode = 8000 // 知识不存在
	CodeKnowledgeExists   ErrorCode = 8001 // 知识已存在

	// 9000-9999 值班管理模块
	CodeDutyNotFound ErrorCode = 9000 // 值班不存在
	CodeDutyExists   ErrorCode = 9001 // 值班已存在

	// 54000-54999 VDI 桌面云模块
	CodeVDIServerNotFound  ErrorCode = 54001 // VDI 服务器不存在
	CodeVDIServerExists    ErrorCode = 54002 // VDI 服务器已存在
	CodeVDIApiFailed       ErrorCode = 54003 // VDI API 调用失败
	CodeVDIAuthFailed      ErrorCode = 54004 // VDI 认证失败
	CodeVDITokenExpired    ErrorCode = 54005 // VDI Token 过期
	CodeVMNotFound         ErrorCode = 54006 // 虚拟机不存在
	CodeVMExists           ErrorCode = 54007 // 虚拟机已存在
	CodeVMOperationFailed  ErrorCode = 54008 // 虚拟机操作失败
	CodeVMInconsistentState ErrorCode = 54009 // 虚拟机状态不一致
)

// DefaultHTTPStatus 返回错误码对应的默认HTTP状态码
func (c ErrorCode) DefaultHTTPStatus() int {
	switch {
	case c == CodeSuccess:
		return http.StatusOK
	case c == CodeRoomDeviceNotFound:
		return http.StatusNotFound
	case c == CodeRoomDeviceCodeExists:
		return http.StatusConflict
	case c >= 1000 && c < 1020:
		return http.StatusBadRequest
	case c >= 1020 && c < 1030:
		return http.StatusUnauthorized
	case c >= 2000 && c < 9000:
		return http.StatusBadRequest
	case c >= 9000 && c < 10000:
		return http.StatusBadRequest
	case c >= 54000 && c < 55000:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// DefaultMessage 返回错误码对应的默认错误消息
func (c ErrorCode) DefaultMessage() string {
	switch c {
	case CodeSuccess:
		return "成功"
	case CodeParamError:
		return "参数错误"
	case CodeParamMissing:
		return "参数缺失"
	case CodeParamInvalid:
		return "参数无效"
	case CodeRecordNotFound:
		return "记录不存在"
	case CodeRecordExists:
		return "记录已存在"
	case CodeDatabaseError:
		return "数据库操作失败"
	case CodeUnauthorized:
		return "未授权"
	case CodeTokenExpired:
		return "令牌已过期"
	case CodeTokenInvalid:
		return "令牌无效"
	case CodePermissionDenied:
		return "权限不足"
	case CodeServerError:
		return "服务器内部错误"
	case CodeNotImplemented:
		return "功能未实现"
	case CodeUserNotFound:
		return "用户不存在"
	case CodeUserExists:
		return "用户已存在"
	case CodeUserDisabled:
		return "用户已禁用"
	case CodeUserPassword:
		return "密码错误"
	case CodeUserHasRoles:
		return "用户已分配角色"
	case CodeUserDeleteSelf:
		return "不能删除自己"
	case CodeRoleNotFound:
		return "角色不存在"
	case CodeRoleExists:
		return "角色已存在"
	case CodeRoleHasUsers:
		return "角色已分配用户"
	case CodeRoleHasMenus:
		return "角色已分配菜单"
	case CodeRoleHasDepts:
		return "角色已分配部门"
	case CodeRoleIsAdmin:
		return "不能修改管理员角色"
	case CodeRoleIsSuper:
		return "不能修改超级管理员角色"
	case CodeDeptNotFound:
		return "部门不存在"
	case CodeDeptExists:
		return "部门已存在"
	case CodeDeptHasUsers:
		return "部门存在用户"
	case CodeDeptHasChildren:
		return "部门存在子部门"
	case CodeDeptHasRoles:
		return "部门已分配角色"
	case CodeDeptInvalid:
		return "无效的部门"
	case CodeDeptIsParent:
		return "不能选择父部门"
	case CodePostNotFound:
		return "岗位不存在"
	case CodePostExists:
		return "岗位已存在"
	case CodePostHasUsers:
		return "岗位已分配用户"
	case CodeMenuNotFound:
		return "菜单不存在"
	case CodeMenuExists:
		return "菜单已存在"
	case CodeMenuHasChildren:
		return "菜单存在子菜单"
	case CodeMenuHasRoles:
		return "菜单已分配角色"
	case CodeMenuInvalid:
		return "无效的菜单"
	case CodeMenuIsParent:
		return "不能选择父菜单"
	case CodeMenuTypeInvalid:
		return "菜单类型无效"
	case CodeMenuHasButtons:
		return "菜单存在按钮"
	case CodeDictTypeNotFound:
		return "字典类型不存在"
	case CodeDictTypeExists:
		return "字典类型已存在"
	case CodeDictDataNotFound:
		return "字典数据不存在"
	case CodeDictDataExists:
		return "字典数据已存在"
	case CodeDictTypeInUse:
		return "字典类型正在使用"
	case CodeConfigNotFound:
		return "参数配置不存在"
	case CodeConfigExists:
		return "参数配置已存在"
	case CodeConfigReserved:
		return "系统保留配置"
	case CodeNoticeNotFound:
		return "通知公告不存在"
	case CodeADConfigNotFound:
		return "AD配置不存在"
	case CodeADConfigExists:
		return "AD配置已存在"
	case CodeADConnectionFailed:
		return "AD连接失败"
	case CodeADSyncFailed:
		return "AD同步失败"
	case CodeSettingsNotFound:
		return "系统设置不存在"
	case CodeDashboardNotFound:
		return "仪表盘不存在"
	case CodeBuildingNotFound:
		return "楼宇不存在"
	case CodeBuildingExists:
		return "楼宇已存在"
	case CodeBuildingHasFloors:
		return "楼宇存在楼层"
	case CodeBuildingInvalid:
		return "无效的楼宇"
	case CodeBuildingOrgInvalid:
		return "关联组织无效"
	case CodeFloorNotFound:
		return "楼层不存在"
	case CodeFloorExists:
		return "楼层已存在"
	case CodeFloorHasWorkstations:
		return "楼层存在工位"
	case CodeFloorInvalid:
		return "无效的楼层"
	case CodeWorkstationNotFound:
		return "工位不存在"
	case CodeWorkstationExists:
		return "工位已存在"
	case CodeWorkstationInvalid:
		return "无效的工位"
	case CodeServerRoomNotFound:
		return "机房不存在"
	case CodeServerRoomExists:
		return "机房已存在"
	case CodeServerRoomHasServers:
		return "机房存在服务器"
	case CodeRoomDeviceNotFound:
		return "机房设备不存在"
	case CodeRoomDeviceCodeExists:
		return "设备编码已存在"
	case CodeInfoPointNotFound:
		return "信息点不存在"
	case CodeInfoPointExists:
		return "信息点已存在"
	case CodeDoorNotFound:
		return "门不存在"
	case CodeDoorExists:
		return "门已存在"
	case CodeWallNotFound:
		return "墙体不存在"
	case CodeWallExists:
		return "墙体已存在"
	case CodeDedicatedLineNotFound:
		return "专线不存在"
	case CodeDedicatedLineExists:
		return "专线已存在"
	case CodeFloorPlanTextNotFound:
		return "楼层平面图文本不存在"
	case CodeRoomPhotoNotFound:
		return "机房照片不存在"
	case CodeRoomPhotoExists:
		return "机房照片已存在"
	case CodeJobNotFound:
		return "定时任务不存在"
	case CodeJobExists:
		return "定时任务已存在"
	case CodeJobIsRunning:
		return "定时任务正在运行"
	case CodeJobHasCron:
		return "定时任务已分配Cron"
	case CodeCronInvalid:
		return "Cron表达式无效"
	case CodeWorkorderNotFound:
		return "工单不存在"
	case CodeWorkorderExists:
		return "工单已存在"
	case CodeWorkorderInvalid:
		return "无效的工单"
	case CodeWorkorderStatus:
		return "工单状态不允许该操作"
	case CodeMonitorDataNotFound:
		return "监控数据不存在"
	case CodeCacheKeyNotFound:
		return "缓存键不存在"
	case CodeCacheOperationFailed:
		return "缓存操作失败"
	case CodeServerInfoNotFound:
		return "服务器信息不存在"
	case CodeNetworkDeviceNotFound:
		return "网络设备不存在"
	case CodeNetworkDeviceExists:
		return "网络设备已存在"
	case CodeNetworkDeviceConnect:
		return "网络设备连接失败"
	case CodeTemplateNotFound:
		return "模板不存在"
	case CodeTemplateExists:
		return "模板已存在"
	case CodeCredentialNotFound:
		return "凭证不存在"
	case CodeCredentialExists:
		return "凭证已存在"
	case CodeCommandFailed:
		return "命令执行失败"
	case CodeBackupNotFound:
		return "备份不存在"
	case CodePortCollectionFailed:
		return "端口采集失败"
	case CodeDiscoveryFailed:
		return "设备发现失败"
	case CodeKnowledgeNotFound:
		return "知识不存在"
	case CodeKnowledgeExists:
		return "知识已存在"
	case CodeDutyNotFound:
		return "值班不存在"
	case CodeDutyExists:
		return "值班已存在"
	case CodeVDIServerNotFound:
		return "VDI服务器不存在"
	case CodeVDIServerExists:
		return "VDI服务器已存在"
	case CodeVDIApiFailed:
		return "VDI API调用失败"
	case CodeVDIAuthFailed:
		return "VDI认证失败"
	case CodeVDITokenExpired:
		return "VDI Token已过期"
	case CodeVMNotFound:
		return "虚拟机不存在"
	case CodeVMExists:
		return "虚拟机已存在"
	case CodeVMOperationFailed:
		return "虚拟机操作失败"
	case CodeVMInconsistentState:
		return "虚拟机状态不一致"
	default:
		return "未知错误"
	}
}
