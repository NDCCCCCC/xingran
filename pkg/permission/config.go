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
	// 85-02（RBAC-04）：部门 Excel 导入导出逐动作拆分新增（department_router.go
	// SetupExcelRouter 的 POST /import、POST /export）。菜单 F 型播种归 Phase 87。
	DeptImport PermissionCode = "system:dept:import"
	DeptExport PermissionCode = "system:dept:export"

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
	// 85-03 批次 B：导出路由（ExportCredentials handler）逐动作拆分新增
	NetworkCredentialExport PermissionCode = "network:credential:export"

	// 配置模板管理权限
	NetworkTemplateList   PermissionCode = "network:template:list"
	NetworkTemplateView   PermissionCode = "network:template:view"
	NetworkTemplateAdd    PermissionCode = "network:template:add"
	NetworkTemplateEdit   PermissionCode = "network:template:edit"
	NetworkTemplateDelete PermissionCode = "network:template:delete"
	// 85-03 批次 B：导出路由新增
	NetworkTemplateExport PermissionCode = "network:template:export"

	// 命令分发权限
	NetworkCommandExecute PermissionCode = "network:command:execute"
	NetworkCommandView    PermissionCode = "network:command:view"
	// 85-03 批次 B：导出路由新增
	NetworkCommandExport PermissionCode = "network:command:export"

	// 配置备份权限
	NetworkBackupList    PermissionCode = "network:backup:list"
	NetworkBackupAdd     PermissionCode = "network:backup:add"
	NetworkBackupRestore PermissionCode = "network:backup:restore"
	NetworkBackupDiff    PermissionCode = "network:backup:diff"
	// 85-03 批次 B：备份删除与导出路由新增（删除此前搭 ANY-of 组级车）
	NetworkBackupDelete PermissionCode = "network:backup:delete"
	NetworkBackupExport PermissionCode = "network:backup:export"

	// 设备发现权限
	NetworkDiscoveryAdd  PermissionCode = "network:discovery:add"
	NetworkDiscoveryView PermissionCode = "network:discovery:view"
	// 85-03 批次 B：发现记录删除与导出路由新增
	NetworkDiscoveryDelete PermissionCode = "network:discovery:delete"
	NetworkDiscoveryExport PermissionCode = "network:discovery:export"

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
	// ==================== 运维管理权限（ops，85-03 RBAC-01 批次 A） ====================
	// ops 域写码此前全部以字符串字面量散落在 router.go 组级 ANY-of 中，本批拆分
	// 为逐动作 RequireStrictPermissions 兄弟写组时统一升格为常量（b 类零豁免）。
	//
	// 词汇约定（WR-01）：本域既有组级码用 delete 字面量，且 ops:building:delete
	// 等 9 域 add/edit/delete 均有 sys_menu F 型授权来源（tmp_ftype 实证）——
	// **保持 delete 不改名**（remove 统一仅约束 system:* 与新增码）。
	// import/export 为本批新增码（此前 Excel 导入导出搭 ANY-of 组级车），
	// 无 F 型菜单 → 窗口期（87 播种前 fail-closed），c 类进 PENDING_SEED_WHITELIST。
	// walls/doors/floor-plan 域为既有字面量码但同样无 F 型授权来源，窗口期同上。

	// 楼宇管理
	OpsBuildingAdd    PermissionCode = "ops:building:add"
	OpsBuildingEdit   PermissionCode = "ops:building:edit"
	OpsBuildingDelete PermissionCode = "ops:building:delete"
	OpsBuildingImport PermissionCode = "ops:building:import"
	OpsBuildingExport PermissionCode = "ops:building:export"
	// 既有 F 型读码（TS 同批声明要求后端有对应常量；读组 ANY-of 仍用字面量）
	OpsBuildingQuery         PermissionCode = "ops:building:query"
	OpsBuildingSpacesQuery   PermissionCode = "ops:building:spaces:query"
	OpsBuildingSpaces3dQuery PermissionCode = "ops:building:spaces:3d:query"

	// 楼层管理
	OpsFloorAdd    PermissionCode = "ops:floor:add"
	OpsFloorEdit   PermissionCode = "ops:floor:edit"
	OpsFloorDelete PermissionCode = "ops:floor:delete"
	OpsFloorImport PermissionCode = "ops:floor:import"
	OpsFloorExport PermissionCode = "ops:floor:export"
	OpsFloorQuery  PermissionCode = "ops:floor:query"

	// 工位管理（ops:workstation:*；注意与 system:workstation:* 的 Workstation*
	// 常量是两个命名空间——工位设备(workstation-device)写路由借用本域码）
	OpsWorkstationAdd    PermissionCode = "ops:workstation:add"
	OpsWorkstationEdit   PermissionCode = "ops:workstation:edit"
	OpsWorkstationDelete PermissionCode = "ops:workstation:delete"
	OpsWorkstationImport PermissionCode = "ops:workstation:import"
	OpsWorkstationExport PermissionCode = "ops:workstation:export"
	OpsWorkstationQuery  PermissionCode = "ops:workstation:query"

	// 资产管理
	OpsAssetAdd    PermissionCode = "ops:asset:add"
	OpsAssetEdit   PermissionCode = "ops:asset:edit"
	OpsAssetDelete PermissionCode = "ops:asset:delete"
	OpsAssetImport PermissionCode = "ops:asset:import"
	OpsAssetExport PermissionCode = "ops:asset:export"
	OpsAssetList   PermissionCode = "ops:asset:list"

	// 信息点管理
	OpsInfoPointAdd    PermissionCode = "ops:infopoint:add"
	OpsInfoPointEdit   PermissionCode = "ops:infopoint:edit"
	OpsInfoPointDelete PermissionCode = "ops:infopoint:delete"
	OpsInfoPointImport PermissionCode = "ops:infopoint:import"
	OpsInfoPointExport PermissionCode = "ops:infopoint:export"
	OpsInfoPointQuery  PermissionCode = "ops:infopoint:query"

	// 机房管理
	OpsServerRoomAdd    PermissionCode = "ops:serverroom:add"
	OpsServerRoomEdit   PermissionCode = "ops:serverroom:edit"
	OpsServerRoomDelete PermissionCode = "ops:serverroom:delete"
	OpsServerRoomImport PermissionCode = "ops:serverroom:import"
	OpsServerRoomExport PermissionCode = "ops:serverroom:export"
	OpsServerRoomQuery  PermissionCode = "ops:serverroom:query"

	// 专线管理
	OpsDedicatedLineAdd    PermissionCode = "ops:dedicatedline:add"
	OpsDedicatedLineEdit   PermissionCode = "ops:dedicatedline:edit"
	OpsDedicatedLineDelete PermissionCode = "ops:dedicatedline:delete"
	OpsDedicatedLineImport PermissionCode = "ops:dedicatedline:import"
	OpsDedicatedLineExport PermissionCode = "ops:dedicatedline:export"
	OpsDedicatedLineQuery  PermissionCode = "ops:dedicatedline:query"

	// 机房设备管理（roomDevice）
	OpsRoomDeviceAdd    PermissionCode = "ops:roomdevice:add"
	OpsRoomDeviceEdit   PermissionCode = "ops:roomdevice:edit"
	OpsRoomDeviceDelete PermissionCode = "ops:roomdevice:delete"
	OpsRoomDeviceImport PermissionCode = "ops:roomdevice:import"
	OpsRoomDeviceExport PermissionCode = "ops:roomdevice:export"
	OpsRoomDeviceQuery  PermissionCode = "ops:roomdevice:query"

	// 工位部门位置映射（location-alias）
	OpsLocationAliasList   PermissionCode = "ops:location:alias:list"
	OpsLocationAliasAdd    PermissionCode = "ops:location:alias:add"
	OpsLocationAliasEdit   PermissionCode = "ops:location:alias:edit"
	OpsLocationAliasDelete PermissionCode = "ops:location:alias:delete"

	// CAD 平面图：墙体 / 门 / 平面图文本（texts 写路由统一归平面图编辑码）
	OpsWallAdd       PermissionCode = "ops:walls:add"
	OpsWallEdit      PermissionCode = "ops:walls:edit"
	OpsWallDelete    PermissionCode = "ops:walls:delete"
	OpsDoorAdd       PermissionCode = "ops:doors:add"
	OpsDoorEdit      PermissionCode = "ops:doors:edit"
	OpsDoorDelete    PermissionCode = "ops:doors:delete"
	OpsFloorPlanEdit PermissionCode = "ops:floor-plan:edit"

	// ==================== AD 域管理 / 经理同步（85-03 RBAC-01 批次 C） ====================
	// 均为既有字面量码升常量（此前散落在 ad_domain_router.go / manager_rule_router.go
	// 的组级 ANY-of 字符串中）；migration_221/163 菜单 seed 对应按钮权限。
	OpsADConfigList   PermissionCode = "ops:ad:config:list"
	OpsADConfigAdd    PermissionCode = "ops:ad:config:add"
	OpsADConfigEdit   PermissionCode = "ops:ad:config:edit"
	OpsADConfigDelete PermissionCode = "ops:ad:config:delete"
	OpsADConfigTest   PermissionCode = "ops:ad:config:test"
	OpsADConfigSync   PermissionCode = "ops:ad:config:sync"
	OpsADGroupEdit       PermissionCode = "ops:ad:group:edit"
	OpsADGroupMappingSync PermissionCode = "ops:ad:group:mapping:sync"
	OpsADUserEdit        PermissionCode = "ops:ad:user:edit"

	OpsManagerRuleList   PermissionCode = "ops:manager:rule:list"
	OpsManagerRuleEdit   PermissionCode = "ops:manager:rule:edit"
	OpsManagerRuleDelete PermissionCode = "ops:manager:rule:delete"
	OpsManagerOrgList    PermissionCode = "ops:manager:org:list"
	OpsManagerOrgEdit    PermissionCode = "ops:manager:org:edit"
	OpsManagerOrgDelete  PermissionCode = "ops:manager:org:delete"
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

		// 角色管理（85-02：batch 为 2 段具体路径，登记在 :id 之前防 matchPath
		// 参数段遮蔽——GetPermissionByPath 按数组顺序返回首个命中）
		{"/system/roles/batch", "POST", RoleRemove, "批量删除角色"},
		{"/system/roles/list", "POST", RoleList, "查询角色列表"},
		{"/system/roles/all", "POST", RoleList, "获取所有角色"},
		{"/system/roles/:id", "POST", RoleView, "查看角色详情"},
		{"/system/roles", "POST", RoleAdd, "新增角色"},
		{"/system/roles/:id/update", "POST", RoleEdit, "编辑角色"},
		{"/system/roles/:id/delete", "POST", RoleRemove, "删除角色"},
		{"/system/roles/:id/status", "POST", RoleEdit, "更新角色状态"},

		// 菜单管理（85-02 拆分同批补齐 batch-delete 与 :id/status；
		// batch-delete 为 2 段具体路径，登记在 :id 之前防参数段遮蔽）
		{"/system/menus/batch-delete", "POST", MenuRemove, "批量删除菜单"},
		{"/system/menus/list", "POST", MenuList, "查询菜单列表"},
		{"/system/menus/:id", "POST", MenuView, "查看菜单详情"},
		{"/system/menus", "POST", MenuAdd, "新增菜单"},
		{"/system/menus/:id/update", "POST", MenuEdit, "编辑菜单"},
		{"/system/menus/:id/status", "POST", MenuEdit, "更新菜单状态"},
		{"/system/menus/:id/delete", "POST", MenuRemove, "删除菜单"},

		// 部门管理（85-01 重建：组前缀实为 /system/departments，router.go:260，
		// 旧登记的 depts 旧前缀漂移已修正；85-02 拆分同批补齐 :id/status 与
		// Excel import/export（department_router.go SetupExcelRouter）。
		// import/export/batch 为 2 段具体路径，登记在 :id 之前防参数段遮蔽。）
		{"/system/departments/batch", "POST", DeptRemove, "批量删除部门"},
		{"/system/departments/import", "POST", DeptImport, "导入部门"},
		{"/system/departments/export", "POST", DeptExport, "导出部门"},
		{"/system/departments/list", "POST", DeptList, "查询部门列表"},
		{"/system/departments/:id", "POST", DeptView, "查看部门详情"},
		{"/system/departments", "POST", DeptAdd, "新增部门"},
		{"/system/departments/:id/update", "POST", DeptEdit, "编辑部门"},
		{"/system/departments/:id/status", "POST", DeptEdit, "更新部门状态"},
		{"/system/departments/:id/delete", "POST", DeptRemove, "删除部门"},

		// 岗位管理（85-02：batch-delete 为 2 段具体路径，登记在 :id 之前防参数段遮蔽）
		{"/system/posts/batch-delete", "POST", PostRemove, "批量删除岗位"}, // 85-01 补登记（post_router.go:35）
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

		// ==================== 网络设备管理（85-03 RBAC-01 批次 B 拆分同批登记） ====================
		// 8 组范围内写路由（拓扑/mac/ports 读组为 PLAN 覆盖清单外，维持 deferred 冻结）。
		// batch-delete/batch/export 等 2 段具体路径登记在 :id 类之前防参数段遮蔽。

		// 网络设备
		{"/network/devices/batch-delete", "POST", NetworkDeviceDelete, "批量删除网络设备"},
		{"/network/devices/export", "POST", NetworkDeviceExport, "导出网络设备"},
		{"/network/devices/quick-create", "POST", NetworkDeviceAdd, "快捷创建网络设备"},
		{"/network/devices", "POST", NetworkDeviceAdd, "新增网络设备"},
		{"/network/devices/:id/update", "POST", NetworkDeviceEdit, "编辑网络设备"},
		{"/network/devices/:id/delete", "POST", NetworkDeviceDelete, "删除网络设备"},

		// 授权凭证
		{"/network/credentials/batch-delete", "POST", NetworkCredentialDelete, "批量删除授权凭证"},
		{"/network/credentials/export", "POST", NetworkCredentialExport, "导出授权凭证"},
		{"/network/credentials", "POST", NetworkCredentialAdd, "新增授权凭证"},
		{"/network/credentials/:id/update", "POST", NetworkCredentialEdit, "编辑授权凭证"},
		{"/network/credentials/:id/delete", "POST", NetworkCredentialDelete, "删除授权凭证"},
		{"/network/credentials/:id/set-default", "POST", NetworkCredentialEdit, "设置默认凭证"},

		// 配置模板（preview 为读语义端点但按写路由口径挂 view 严格单码；clone 归 add）
		{"/network/templates/batch-delete", "POST", NetworkTemplateDelete, "批量删除配置模板"},
		{"/network/templates/export", "POST", NetworkTemplateExport, "导出配置模板"},
		{"/network/templates", "POST", NetworkTemplateAdd, "新增配置模板"},
		{"/network/templates/:id/update", "POST", NetworkTemplateEdit, "编辑配置模板"},
		{"/network/templates/:id/delete", "POST", NetworkTemplateDelete, "删除配置模板"},
		{"/network/templates/:id/clone", "POST", NetworkTemplateAdd, "克隆配置模板"},
		{"/network/templates/:id/preview", "POST", NetworkTemplateView, "预览配置模板"},

		// 命令分发
		{"/network/command/export", "POST", NetworkCommandExport, "导出命令执行记录"},
		{"/network/command/dispatch", "POST", NetworkCommandExecute, "分发命令"},
		{"/network/command/quick", "POST", NetworkCommandExecute, "快捷命令"},

		// 配置执行（执行记录删除/导出归 execute 域码——域内无独立 delete/export 码，
		// 85-03 裁定零新码，避免扩大窗口期）
		{"/network/executions/batch-delete", "POST", NetworkCommandExecute, "批量删除执行记录"},
		{"/network/executions/export", "POST", NetworkCommandExecute, "导出执行记录"},
		{"/network/executions/template/execute", "POST", NetworkCommandExecute, "按模板执行配置"},
		{"/network/executions/:id/delete", "POST", NetworkCommandExecute, "删除执行记录"},

		// 配置备份
		{"/network/backups/batch", "POST", NetworkBackupAdd, "批量创建备份"},
		{"/network/backups/batch-delete", "POST", NetworkBackupDelete, "批量删除备份"},
		{"/network/backups/export", "POST", NetworkBackupExport, "导出配置备份"},
		{"/network/backups/diff", "POST", NetworkBackupDiff, "备份差异对比"},
		{"/network/backups", "POST", NetworkBackupAdd, "创建配置备份"},
		{"/network/backups/:id/delete", "POST", NetworkBackupDelete, "删除配置备份"},
		{"/network/backups/:id/restore", "POST", NetworkBackupRestore, "恢复配置备份"},

		// 设备发现
		{"/network/discoveries/batch-delete", "POST", NetworkDiscoveryDelete, "批量删除发现记录"},
		{"/network/discoveries/export", "POST", NetworkDiscoveryExport, "导出发现记录"},
		{"/network/discoveries/create", "POST", NetworkDiscoveryAdd, "创建发现任务"},
		{"/network/discoveries/:id/execute", "POST", NetworkDiscoveryAdd, "执行发现任务"},
		{"/network/discoveries/:id/import", "POST", NetworkDiscoveryAdd, "导入发现设备"},
		{"/network/discoveries/:id/delete", "POST", NetworkDiscoveryDelete, "删除发现记录"},

		// ==================== system 其余组（85-03 RBAC-01 批次 C 拆分同批登记） ====================
		// batch-delete/refresh-cache 等 2 段具体路径登记在 :id 类之前防参数段遮蔽。

		// API密钥管理
		{"/system/apikeys/:id/toggle", "POST", APIKeyEdit, "切换API密钥状态"},
		{"/system/apikeys/:id/update", "POST", APIKeyEdit, "编辑API密钥"},
		{"/system/apikeys/:id/delete", "POST", APIKeyRemove, "删除API密钥"},
		{"/system/apikeys", "POST", APIKeyAdd, "新增API密钥"},

		// 参数配置（refresh-cache 为刷新系统配置缓存=编辑语义）
		{"/system/configs/batch-delete", "POST", ConfigRemove, "批量删除参数配置"},
		{"/system/configs/refresh-cache", "POST", ConfigEdit, "刷新参数缓存"},
		{"/system/configs", "POST", ConfigAdd, "新增参数配置"},
		{"/system/configs/:id/update", "POST", ConfigEdit, "编辑参数配置"},
		{"/system/configs/:id/delete", "POST", ConfigRemove, "删除参数配置"},

		// 字典管理（类型/数据共用 system:dict:* 码族，XingRan 惯例）
		{"/system/dicts/types", "POST", DictTypeAdd, "新增字典类型"},
		{"/system/dicts/types/:id/update", "POST", DictTypeEdit, "编辑字典类型"},
		{"/system/dicts/types/:id/delete", "POST", DictTypeRemove, "删除字典类型"},
		{"/system/dicts/data", "POST", DictTypeAdd, "新增字典数据"},
		{"/system/dicts/data/:id/update", "POST", DictTypeEdit, "编辑字典数据"},
		{"/system/dicts/data/:id/delete", "POST", DictTypeRemove, "删除字典数据"},

		// 通知公告（publish/withdraw 为状态变更=编辑语义）
		{"/system/notices/batch-delete", "POST", NoticeRemove, "批量删除通知公告"},
		{"/system/notices", "POST", NoticeAdd, "新增通知公告"},
		{"/system/notices/:id/update", "POST", NoticeEdit, "编辑通知公告"},
		{"/system/notices/:id/delete", "POST", NoticeRemove, "删除通知公告"},
		{"/system/notices/:id/publish", "POST", NoticeEdit, "发布通知公告"},
		{"/system/notices/:id/withdraw", "POST", NoticeEdit, "撤回通知公告"},
		{"/system/notices/:id/channels", "POST", NoticeEdit, "设置通知渠道"},

		// 经理同步规则/机构（migration_221 菜单 seed；generate-from-dept=批量创建映射）
		{"/system/manager-rules", "POST", OpsManagerRuleEdit, "新增经理同步规则"},
		{"/system/manager-rules/:id/update", "POST", OpsManagerRuleEdit, "编辑经理同步规则"},
		{"/system/manager-rules/:id/delete", "POST", OpsManagerRuleDelete, "删除经理同步规则"},
		{"/system/manager-orgs/generate-from-dept", "POST", OpsManagerOrgEdit, "从部门架构生成机构映射"},
		{"/system/manager-orgs", "POST", OpsManagerOrgEdit, "新增经理机构映射"},
		{"/system/manager-orgs/:id/update", "POST", OpsManagerOrgEdit, "编辑经理机构映射"},
		{"/system/manager-orgs/:id/delete", "POST", OpsManagerOrgDelete, "删除经理机构映射"},

		// AD 域管理（ad_domain_router.go 拆分；migration_163 菜单 seed）
		// 服务账号池 8 端点（85-03 同批补登死表；85 前仅挂载无登记）
		{"/ad-domain/accounts/list", "POST", PermissionCode("ops:ad:config:account:list"), "AD服务账号列表"},
		{"/ad-domain/accounts/stats", "POST", PermissionCode("ops:ad:config:account:list"), "AD服务账号统计"},
		{"/ad-domain/accounts/create", "POST", PermissionCode("ops:ad:config:account:edit"), "新增AD服务账号"},
		{"/ad-domain/accounts/update", "POST", PermissionCode("ops:ad:config:account:edit"), "编辑AD服务账号"},
		{"/ad-domain/accounts/enable", "POST", PermissionCode("ops:ad:config:account:edit"), "启用AD服务账号"},
		{"/ad-domain/accounts/disable", "POST", PermissionCode("ops:ad:config:account:edit"), "禁用AD服务账号"},
		{"/ad-domain/accounts/unlock", "POST", PermissionCode("ops:ad:config:account:edit"), "解锁AD服务账号"},
		{"/ad-domain/accounts/delete", "POST", PermissionCode("ops:ad:config:account:delete"), "删除AD服务账号"},
		{"/ad-domain/configs/:id/sync", "POST", OpsADConfigSync, "同步AD配置数据"},
		{"/ad-domain/configs/:id/test", "POST", OpsADConfigTest, "测试AD连接"},
		{"/ad-domain/configs/:id/delete", "POST", OpsADConfigEdit, "删除AD配置"},
		{"/ad-domain/configs/:id/update", "POST", OpsADConfigEdit, "编辑AD配置"},
		{"/ad-domain/configs", "POST", OpsADConfigAdd, "新增AD配置"},
		{"/ad-domain/groups/:id/update", "POST", OpsADGroupEdit, "编辑AD用户组"},
		{"/ad-domain/users/batch-sync", "POST", OpsADUserEdit, "批量同步AD用户"},
		{"/ad-domain/users/:id/update", "POST", OpsADUserEdit, "编辑AD用户"},
		{"/ad-domain/users/:id/move", "POST", OpsADUserEdit, "移动AD用户OU"},
		{"/ad-domain/users/:id/enable", "POST", OpsADUserEdit, "启用AD用户"},
		{"/ad-domain/users/:id/disable", "POST", OpsADUserEdit, "禁用AD用户"},

		// ==================== 运维管理 ops（85-03 RBAC-01 批次 A 拆分同批登记） ====================
		// 每域：batch/import/export 为 2 段具体路径，登记在 :id 类之前防 matchPath
		// 参数段遮蔽（GetPermissionByPath 按数组顺序返回首个命中）。
		// geocode 为读语义端点（POST 但仅地址解析查询，isWrite=false）不入表。

		// 楼宇管理
		{"/ops/building/batch", "POST", OpsBuildingDelete, "批量删除楼宇"},
		{"/ops/building/import", "POST", OpsBuildingImport, "导入楼宇"},
		{"/ops/building/export", "POST", OpsBuildingExport, "导出楼宇"},
		{"/ops/building", "POST", OpsBuildingAdd, "新增楼宇"},
		{"/ops/building/:id/update", "POST", OpsBuildingEdit, "编辑楼宇"},
		{"/ops/building/:id/delete", "POST", OpsBuildingDelete, "删除楼宇"},

		// 楼层管理
		{"/ops/floor/batch", "POST", OpsFloorDelete, "批量删除楼层"},
		{"/ops/floor/import", "POST", OpsFloorImport, "导入楼层"},
		{"/ops/floor/export", "POST", OpsFloorExport, "导出楼层"},
		{"/ops/floor", "POST", OpsFloorAdd, "新增楼层"},
		{"/ops/floor/:id/update", "POST", OpsFloorEdit, "编辑楼层"},
		{"/ops/floor/:id/delete", "POST", OpsFloorDelete, "删除楼层"},

		// 工位管理（positions 为批量更新工位物理位置=编辑）
		{"/ops/workstation/batch", "POST", OpsWorkstationDelete, "批量删除工位"},
		{"/ops/workstation/positions", "POST", OpsWorkstationEdit, "批量更新工位位置"},
		{"/ops/workstation/import", "POST", OpsWorkstationImport, "导入工位"},
		{"/ops/workstation/export", "POST", OpsWorkstationExport, "导出工位"},
		{"/ops/workstation", "POST", OpsWorkstationAdd, "新增工位"},
		{"/ops/workstation/:id/update", "POST", OpsWorkstationEdit, "编辑工位"},
		{"/ops/workstation/:id/delete", "POST", OpsWorkstationDelete, "删除工位"},

		// 工位设备关联（借用 ops:workstation:* 域码：manual=新增绑定、
		// sync-*/update/set-primary=编辑、delete=删除）
		{"/ops/workstation-device/manual", "POST", OpsWorkstationAdd, "手工添加工位设备"},
		{"/ops/workstation-device/sync-ad", "POST", OpsWorkstationEdit, "同步AD设备"},
		{"/ops/workstation-device/sync-asset", "POST", OpsWorkstationEdit, "同步资产设备"},
		{"/ops/workstation-device/:id/update", "POST", OpsWorkstationEdit, "更新工位设备"},
		{"/ops/workstation-device/:id/set-primary", "POST", OpsWorkstationEdit, "设置主设备"},
		{"/ops/workstation-device/:id/set-primary-and-save", "POST", OpsWorkstationEdit, "设置主设备并保存"},
		{"/ops/workstation-device/:id/delete", "POST", OpsWorkstationDelete, "删除工位设备"},

		// 资产管理
		{"/ops/asset/batch", "POST", OpsAssetDelete, "批量删除资产"},
		{"/ops/asset/import", "POST", OpsAssetImport, "导入资产"},
		{"/ops/asset/export", "POST", OpsAssetExport, "导出资产"},
		{"/ops/asset", "POST", OpsAssetAdd, "新增资产"},
		{"/ops/asset/:id/update", "POST", OpsAssetEdit, "编辑资产"},
		{"/ops/asset/:id/delete", "POST", OpsAssetDelete, "删除资产"},

		// 信息点管理
		{"/ops/infoPoint/batch", "POST", OpsInfoPointDelete, "批量删除信息点"},
		{"/ops/infoPoint/import", "POST", OpsInfoPointImport, "导入信息点"},
		{"/ops/infoPoint/export", "POST", OpsInfoPointExport, "导出信息点"},
		{"/ops/infoPoint", "POST", OpsInfoPointAdd, "新增信息点"},
		{"/ops/infoPoint/:id/update", "POST", OpsInfoPointEdit, "编辑信息点"},
		{"/ops/infoPoint/:id/delete", "POST", OpsInfoPointDelete, "删除信息点"},

		// 机房管理
		{"/ops/serverRoom/batch", "POST", OpsServerRoomDelete, "批量删除机房"},
		{"/ops/serverRoom/import", "POST", OpsServerRoomImport, "导入机房"},
		{"/ops/serverRoom/export", "POST", OpsServerRoomExport, "导出机房"},
		{"/ops/serverRoom", "POST", OpsServerRoomAdd, "新增机房"},
		{"/ops/serverRoom/:id/update", "POST", OpsServerRoomEdit, "编辑机房"},
		{"/ops/serverRoom/:id/delete", "POST", OpsServerRoomDelete, "删除机房"},

		// 专线管理
		{"/ops/dedicatedLine/batch", "POST", OpsDedicatedLineDelete, "批量删除专线"},
		{"/ops/dedicatedLine/import", "POST", OpsDedicatedLineImport, "导入专线"},
		{"/ops/dedicatedLine/export", "POST", OpsDedicatedLineExport, "导出专线"},
		{"/ops/dedicatedLine", "POST", OpsDedicatedLineAdd, "新增专线"},
		{"/ops/dedicatedLine/:id/update", "POST", OpsDedicatedLineEdit, "编辑专线"},
		{"/ops/dedicatedLine/:id/delete", "POST", OpsDedicatedLineDelete, "删除专线"},

		// 机房设备管理
		{"/ops/roomDevice/batch", "POST", OpsRoomDeviceDelete, "批量删除机房设备"},
		{"/ops/roomDevice/import", "POST", OpsRoomDeviceImport, "导入机房设备"},
		{"/ops/roomDevice/export", "POST", OpsRoomDeviceExport, "导出机房设备"},
		{"/ops/roomDevice", "POST", OpsRoomDeviceAdd, "新增机房设备"},
		{"/ops/roomDevice/:id/update", "POST", OpsRoomDeviceEdit, "编辑机房设备"},
		{"/ops/roomDevice/:id/delete", "POST", OpsRoomDeviceDelete, "删除机房设备"},

		// 工位部门位置映射
		{"/ops/location-alias", "POST", OpsLocationAliasAdd, "新增位置映射"},
		{"/ops/location-alias/:id/update", "POST", OpsLocationAliasEdit, "编辑位置映射"},
		{"/ops/location-alias/:id/delete", "POST", OpsLocationAliasDelete, "删除位置映射"},

		// CAD 平面图：墙体 / 门 / 平面图文本（batch 为 2 段具体路径，登记在 :id 类之前）
		{"/ops/walls/batch", "POST", OpsWallDelete, "批量删除墙体"},
		{"/ops/walls", "POST", OpsWallAdd, "新增墙体"},
		{"/ops/walls/:id/update", "POST", OpsWallEdit, "编辑墙体"},
		{"/ops/walls/:id/delete", "POST", OpsWallDelete, "删除墙体"},
		{"/ops/doors/batch", "POST", OpsDoorDelete, "批量删除门"},
		{"/ops/doors", "POST", OpsDoorAdd, "新增门"},
		{"/ops/doors/:id/update", "POST", OpsDoorEdit, "编辑门"},
		{"/ops/doors/:id/delete", "POST", OpsDoorDelete, "删除门"},
		{"/ops/floor-plan-texts/batch", "POST", OpsFloorPlanEdit, "批量操作平面图文本"},
		{"/ops/floor-plan-texts", "POST", OpsFloorPlanEdit, "新增平面图文本"},
		{"/ops/floor-plan-texts/:id/update", "POST", OpsFloorPlanEdit, "编辑平面图文本"},
		{"/ops/floor-plan-texts/:id/delete", "POST", OpsFloorPlanEdit, "删除平面图文本"},
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
