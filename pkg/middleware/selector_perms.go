package middleware

// OpsSelectorReadPerms 是跨模块只读选择器的放行权限集 (单一来源, DRY)。
//
// 背景: 运维/空间管理页面内嵌的选择器复用了 system / network 模块的 list 接口, 但运维
// 角色通常不持有 system:dept / system:user / system:dict / network:device 权限, 导致
// 这些选择器在每个运维页面都 403:
//   - 部门树 <DeptTree>        → /system/departments/tree   (system:dept)
//   - 工位用户选择器           → /system/users/list          (system:user)
//   - 字典 useDict             → /system/dicts/data/list     (system:dict)
//   - 信息点关联网络设备/端口  → /network/devices/list, /network/ports/list (network:device/port)
//
// 此列表作为 RequirePermissionsWithQuery 的 queryExtraPermissions, 让持有任一运维读权限
// 的角色也能读取上述选择器接口 (查询路径 /list,/tree); 写操作 (增删改) 保持严格权限。
//
// router.go (system/ops 组) 与 network_router.go (devices/ports 组) 共用此常量, 避免重复定义。
// 新增运维模块时, 在此追加其 :list 权限即可, 所有跨模块选择器自动对新模块角色放行。
var OpsSelectorReadPerms = []string{
	"ops:building:list",
	"ops:floor:list",
	"ops:workstation:list",
	"ops:serverroom:list",
	"ops:infopoint:list",
	"ops:dedicatedline:list",
	"ops:roomdevice:list",
	"ops:asset:list",
	"ops:building:spaces:list",
}
