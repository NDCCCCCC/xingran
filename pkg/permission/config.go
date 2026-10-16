package permission

import (
	"strings"
)

// PermissionCode 权限代码定义
type PermissionCode string

const (
	// 用户管理权限
	UserList     PermissionCode = "system:user:list"
	UserView     PermissionCode = "system:user:view"
	UserAdd      PermissionCode = "system:user:add"
	UserEdit     PermissionCode = "system:user:edit"
	UserRemove   PermissionCode = "system:user:remove"
	UserExport   PermissionCode = "system:user:export"
	UserImport   PermissionCode = "system:user:import"
	UserResetPwd PermissionCode = "system:user:resetPwd"

	// 角色管理权限
	RoleList   PermissionCode = "system:role:list"
	RoleView   PermissionCode = "system:role:view"
	RoleAdd    PermissionCode = "system:role:add"
	RoleEdit   PermissionCode = "system:role:edit"
	RoleRemove PermissionCode = "system:role:remove"
	RoleExport PermissionCode = "system:role:export"

	// 菜单管理权限
	MenuList   PermissionCode = "system:menu:list"
	MenuView   PermissionCode = "system:menu:view"
	MenuAdd    PermissionCode = "system:menu:add"
	MenuEdit   PermissionCode = "system:menu:edit"
	MenuRemove PermissionCode = "system:menu:remove"

	// 部门管理权限
	DeptList   PermissionCode = "system:dept:list"
	DeptView   PermissionCode = "system:dept:view"
	DeptAdd    PermissionCode = "system:dept:add"
	DeptEdit   PermissionCode = "system:dept:edit"
	DeptRemove PermissionCode = "system:dept:remove"

	// 岗位管理权限
	PostList   PermissionCode = "system:post:list"
	PostView   PermissionCode = "system:post:view"
	PostAdd    PermissionCode = "system:post:add"
	PostEdit   PermissionCode = "system:post:edit"
	PostRemove PermissionCode = "system:post:remove"

	// 工位管理权限
	WorkstationList   PermissionCode = "system:workstation:list"
	WorkstationView   PermissionCode = "system:workstation:view"
	WorkstationAdd    PermissionCode = "system:workstation:add"
	WorkstationEdit   PermissionCode = "system:workstation:edit"
	WorkstationRemove PermissionCode = "system:workstation:remove"

	// 字典管理权限
	DictTypeList   PermissionCode = "system:dict:list"
	DictTypeView   PermissionCode = "system:dict:view"
	DictTypeAdd    PermissionCode = "system:dict:add"
	DictTypeEdit   PermissionCode = "system:dict:edit"
	DictTypeRemove PermissionCode = "system:dict:remove"

	// 参数配置权限
	ConfigList   PermissionCode = "system:config:list"
	ConfigView   PermissionCode = "system:config:view"
	ConfigAdd    PermissionCode = "system:config:add"
	ConfigEdit   PermissionCode = "system:config:edit"
	ConfigRemove PermissionCode = "system:config:remove"

	// 验证码背景图权限
	CaptchaBackgroundList   PermissionCode = "system:captchaBackground:list"
	CaptchaBackgroundView   PermissionCode = "system:captchaBackground:view"
	CaptchaBackgroundAdd    PermissionCode = "system:captchaBackground:add"
	CaptchaBackgroundEdit   PermissionCode = "system:captchaBackground:edit"
	CaptchaBackgroundRemove PermissionCode = "system:captchaBackground:remove"

	// 系统模块权限
	System       PermissionCode = "system"
	SystemUser   PermissionCode = "system:user"
	SystemRole   PermissionCode = "system:role"
	SystemMenu   PermissionCode = "system:menu"
	SystemDept   PermissionCode = "system:dept"
	SystemPost   PermissionCode = "system:post"
	SystemDict   PermissionCode = "system:dict"
	SystemConfig PermissionCode = "system:config"
	SystemNotice PermissionCode = "system:notice"

	// API Key 管理权限(Phase 61 / AUTH-04 — 由 resource_action_map.go 引用)
	// 词汇约定(WR-01): 删除权限统一为 remove(system:* 模块约定, D-04)。
	// router.go apikeys 组级检查已从历史字面量 "system:apikey:delete" 对齐到
	// "system:apikey:remove"(v1.6 旧约定无菜单授权来源,永不命中)。
	// 注意: 菜单种子仅播种 system:apikey:list/logs, add/edit/remove 细粒度授权
	// 需由后续 migration 或 admin 手动授权补充(见 61-REVIEW.md WR-01)。
	APIKeyList   PermissionCode = "system:apikey:list"
	APIKeyView   PermissionCode = "system:apikey:view"
	APIKeyAdd    PermissionCode = "system:apikey:add"
	APIKeyEdit   PermissionCode = "system:apikey:edit"
	APIKeyRemove PermissionCode = "system:apikey:remove"

	// 通知公告权限
	NoticeList   PermissionCode = "system:notice:list"
	NoticeView   PermissionCode = "system:notice:view"
	NoticeAdd    PermissionCode = "system:notice:add"
	NoticeEdit   PermissionCode = "system:notice:edit"
	NoticeRemove PermissionCode = "system:notice:remove"

	// 日志管理权限
	OperLogList   PermissionCode = "monitor:operlog:list"
	OperLogView   PermissionCode = "monitor:operlog:view"
	OperLogExport PermissionCode = "monitor:operlog:export"
	OperLogRemove PermissionCode = "monitor:operlog:remove"

	LoginLogList   PermissionCode = "monitor:loginlog:list"
	LoginLogView   PermissionCode = "monitor:loginlog:view"
	LoginLogExport PermissionCode = "monitor:loginlog:export"
	LoginLogRemove PermissionCode = "monitor:loginlog:remove"

	// 在线用户权限
	OnlineList        PermissionCode = "monitor:online:list"
	OnlineForceLogout PermissionCode = "monitor:online:forceLogout"

	// 定时任务权限
	JobList   PermissionCode = "monitor:job:list"
	JobView   PermissionCode = "monitor:job:view"
	JobAdd    PermissionCode = "monitor:job:add"
	JobEdit   PermissionCode = "monitor:job:edit"
	JobRemove PermissionCode = "monitor:job:remove"
	JobExport PermissionCode = "monitor:job:export"
	JobStart  PermissionCode = "monitor:job:start"
	JobStop   PermissionCode = "monitor:job:stop"
	JobRun    PermissionCode = "monitor:job:run"

	// 服务监控权限
	ServerList PermissionCode = "monitor:server:list"
	ServerView PermissionCode = "monitor:server:view"

	// 缓存监控权限
	CacheList  PermissionCode = "monitor:cache:list"
	CacheView  PermissionCode = "monitor:cache:view"
	CacheClean PermissionCode = "monitor:cache:clean"

	// 监控模块权限
	Monitor       PermissionCode = "monitor"
	MonitorServer PermissionCode = "monitor:server"
	MonitorLog    PermissionCode = "monitor:log"
	MonitorJob    PermissionCode = "monitor:job"
	MonitorCache  PermissionCode = "monitor:cache"

	// 在线构建权限
	BuildList   PermissionCode = "tool:build:list"
	BuildAdd    PermissionCode = "tool:build:add"
	BuildEdit   PermissionCode = "tool:build:edit"
	BuildRemove PermissionCode = "tool:build:remove"
	BuildExport PermissionCode = "tool:build:export"

	// ==================== 网络设备管理权限 ====================
	// 设备管理权限
	NetworkDeviceList   PermissionCode = "network:device:list"
	NetworkDeviceView   PermissionCode = "network:device:view"
	NetworkDeviceAdd    PermissionCode = "network:device:add"
	NetworkDeviceEdit   PermissionCode = "network:device:edit"
	NetworkDeviceDelete PermissionCode = "network:device:delete"
	NetworkDeviceExport PermissionCode = "network:device:export"

	// 授权凭证管理权限
	NetworkCredentialList   PermissionCode = "network:credential:list"
	NetworkCredentialView   PermissionCode = "network:credential:view"
	NetworkCredentialAdd    PermissionCode = "network:credential:add"
	NetworkCredentialEdit   PermissionCode = "network:credential:edit"
	NetworkCredentialDelete PermissionCode = "network:credential:delete"

	// 配置模板管理权限
	NetworkTemplateList   PermissionCode = "network:template:list"
	NetworkTemplateView   PermissionCode = "network:template:view"
	NetworkTemplateAdd    PermissionCode = "network:template:add"
	NetworkTemplateEdit   PermissionCode = "network:template:edit"
	NetworkTemplateDelete PermissionCode = "network:template:delete"

	// 命令分发权限
	NetworkCommandExecute PermissionCode = "network:command:execute"
	NetworkCommandView    PermissionCode = "network:command:view"

	// 配置备份权限
	NetworkBackupList    PermissionCode = "network:backup:list"
	NetworkBackupAdd     PermissionCode = "network:backup:add"
	NetworkBackupRestore PermissionCode = "network:backup:restore"
	NetworkBackupDiff    PermissionCode = "network:backup:diff"

	// 设备发现权限
	NetworkDiscoveryAdd  PermissionCode = "network:discovery:add"
	NetworkDiscoveryView PermissionCode = "network:discovery:view"

	// MAC地址查询权限
	NetworkMacQuery PermissionCode = "network:mac:query"

	// 端口状态查询权限
	NetworkPortQuery PermissionCode = "network:port:query"

	// 端口写操作权限（Phase 52: shutdown/undo_shutdown/description/dot1x_enable/dot1x_disable/batch）
	NetworkPortWrite PermissionCode = "network:port:write"

	// 网络设备模块权限
	Network           PermissionCode = "network"
	NetworkDevice     PermissionCode = "network:device"
	NetworkCredential PermissionCode = "network:credential"
	NetworkTemplate   PermissionCode = "network:template"
	NetworkCommand    PermissionCode = "network:command"
	NetworkBackup     PermissionCode = "network:backup"
	NetworkDiscovery  PermissionCode = "network:discovery"
	NetworkMac        PermissionCode = "network:mac"
	NetworkPort       PermissionCode = "network:port"
)

// RoutePermission 路由权限映射
type RoutePermission struct {
	Path        string
	Method      string
	Permission  PermissionCode
	Description string
}

// GetRoutePermissions 获取路由权限配置
func GetRoutePermissions() []RoutePermission {
	return []RoutePermission{
		// 用户管理
		{"/system/users/list", "POST", UserList, "查询用户列表"},
		{"/system/users/:id", "POST", UserView, "查看用户详情"},
		{"/system/users", "POST", UserAdd, "新增用户"},
		{"/system/users/:id/update", "POST", UserEdit, "编辑用户"},
		{"/system/users/:id/delete", "POST", UserRemove, "删除用户"},
		{"/system/users/batch-delete", "POST", UserRemove, "批量删除用户"},
		{"/system/users/:id/status", "POST", UserEdit, "更新用户状态"},
		{"/system/users/:id/reset-password", "POST", UserResetPwd, "重置用户密码"},
		{"/system/users/export", "POST", UserExport, "导出用户"},
		{"/system/users/import", "POST", UserImport, "导入用户"},

		// 角色管理
		{"/system/roles/list", "POST", RoleList, "查询角色列表"},
		{"/system/roles/all", "POST", RoleList, "获取所有角色"},
		{"/system/roles/:id", "POST", RoleView, "查看角色详情"},
		{"/system/roles", "POST", RoleAdd, "新增角色"},
		{"/system/roles/:id/update", "POST", RoleEdit, "编辑角色"},
		{"/system/roles/:id/delete", "POST", RoleRemove, "删除角色"},
		{"/system/roles/batch-delete", "POST", RoleRemove, "批量删除角色"},
		{"/system/roles/:id/status", "POST", RoleEdit, "更新角色状态"},
		{"/system/roles/:id/menus", "POST", RoleEdit, "更新角色菜单权限"},
		{"/system/roles/export", "POST", RoleExport, "导出角色"},

		// 菜单管理
		{"/system/menus/list", "POST", MenuList, "查询菜单列表"},
		{"/system/menus/:id", "POST", MenuView, "查看菜单详情"},
		{"/system/menus", "POST", MenuAdd, "新增菜单"},
		{"/system/menus/:id/update", "POST", MenuEdit, "编辑菜单"},
		{"/system/menus/:id/delete", "POST", MenuRemove, "删除菜单"},

		// 部门管理
		{"/system/depts/list", "POST", DeptList, "查询部门列表"},
		{"/system/depts/:id", "POST", DeptView, "查看部门详情"},
		{"/system/depts", "POST", DeptAdd, "新增部门"},
		{"/system/depts/:id/update", "POST", DeptEdit, "编辑部门"},
		{"/system/depts/:id/delete", "POST", DeptRemove, "删除部门"},

		// 岗位管理
		{"/system/posts/list", "POST", PostList, "查询岗位列表"},
		{"/system/posts/:id", "POST", PostView, "查看岗位详情"},
		{"/system/posts", "POST", PostAdd, "新增岗位"},
		{"/system/posts/:id/update", "POST", PostEdit, "编辑岗位"},
		{"/system/posts/:id/delete", "POST", PostRemove, "删除岗位"},

		// 工位管理
		{"/system/workstations/list", "POST", WorkstationList, "查询工位列表"},
		{"/system/workstations/:id", "POST", WorkstationView, "查看工位详情"},
		{"/system/workstations", "POST", WorkstationAdd, "新增工位"},
		{"/system/workstations/:id/update", "POST", WorkstationEdit, "编辑工位"},
		{"/system/workstations/:id/delete", "POST", WorkstationRemove, "删除工位"},
		{"/system/workstations/batch-delete", "POST", WorkstationRemove, "批量删除工位"},

		// 网络端口写操作（Phase 52: shutdown/undo-shutdown/description/dot1x-enable/dot1x-disable/batch）
		{"/network/ports/write/shutdown", "POST", NetworkPortWrite, "关闭端口"},
		{"/network/ports/write/undo-shutdown", "POST", NetworkPortWrite, "撤销关闭端口"},
		{"/network/ports/write/description", "POST", NetworkPortWrite, "修改端口描述"},
		{"/network/ports/write/dot1x-enable", "POST", NetworkPortWrite, "启用端口802.1X认证"},
		{"/network/ports/write/dot1x-disable", "POST", NetworkPortWrite, "关闭端口802.1X认证"},
		{"/network/ports/write/batch", "POST", NetworkPortWrite, "批量端口写操作"},
		// v1.20.1 端口写命令扩展（Phase 56 W3: set_access_vlan + port_binding，复用同一权限）
		{"/network/ports/write/set-access-vlan", "POST", NetworkPortWrite, "修改端口 access VLAN"},
		{"/network/ports/write/port-binding", "POST", NetworkPortWrite, "端口绑定（IP/MAC/Port 静态绑定）"},
	}
}

// GetPermissionByPath 根据路径和方法获取权限代码
func GetPermissionByPath(path, method string) PermissionCode {
	permissions := GetRoutePermissions()
	for _, p := range permissions {
		// 简单路径匹配，支持参数
		if matchPath(p.Path, path) && p.Method == method {
			return p.Permission
		}
	}
	return ""
}

// matchPath 路径匹配（支持参数）
func matchPath(pattern, path string) bool {
	// 简单实现，支持 :id 参数
	if pattern == path {
		return true
	}

	// 分割路径
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i := 0; i < len(patternParts); i++ {
		if strings.HasPrefix(patternParts[i], ":") {
			// 参数部分，跳过比较
			continue
		}
		if patternParts[i] != pathParts[i] {
			return false
		}
	}

	return true
}
