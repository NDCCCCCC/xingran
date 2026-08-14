package system

import (
	"fmt"
	"strings"
	"time"
)

// CacheKeyManager 缓存键管理器
// 提供统一的缓存键构建和管理功能
type CacheKeyManager struct {
	prefix string // 缓存键前缀
}

// NewCacheKeyManager 创建缓存键管理器
func NewCacheKeyManager(prefix string) *CacheKeyManager {
	return &CacheKeyManager{
		prefix: prefix,
	}
}

// Build 构建缓存键
// 支持可变参数拼接，自动添加前缀
func (m *CacheKeyManager) Build(parts ...string) string {
	if m.prefix == "" {
		return strings.Join(parts, ":")
	}
	allParts := append([]string{m.prefix}, parts...)
	return strings.Join(allParts, ":")
}

// BuildPattern 构建缓存键模式（用于模糊匹配）
// 例如：BuildPattern("user", "*") -> "xingran:user:*"
func (m *CacheKeyManager) BuildPattern(parts ...string) string {
	result := m.Build(parts...)
	// 确保模式以 * 结尾
	if !strings.HasSuffix(result, "*") {
		result += "*"
	}
	return result
}

// Parse 解析缓存键，移除前缀
func (m *CacheKeyManager) Parse(key string) string {
	if m.prefix == "" {
		return key
	}
	if strings.HasPrefix(key, m.prefix+":") {
		return key[len(m.prefix)+1:]
	}
	return key
}

// ==================== 缓存键常量定义 ====================
// 这些常量定义了系统中所有缓存键的格式
// 使用 CacheKeyManager.Build() 方法构建完整的键

// 前缀定义
const (
	CachePrefixPrefix = "cache" // 缓存键前缀（用于避免与其他键冲突）
)

// 模块定义
const (
	ModuleSystem     = "system"     // 系统模块
	ModuleUser       = "user"       // 用户模块
	ModuleRole       = "role"       // 角色模块
	ModuleMenu       = "menu"       // 菜单模块
	ModuleDept       = "dept"       // 部门模块
	ModulePost       = "post"       // 岗位模块
	ModuleDict       = "dict"       // 字典模块
	ModuleConfig     = "config"     // 配置模块
	ModuleNotice     = "notice"     // 通知模块
	ModuleOperations = "operations" // 运维模块
	ModuleMonitor    = "monitor"    // 监控模块
)

// 系统模块缓存键
const (
	// 用户相关
	CacheKeyUserByID       = "user:id"   // 用户详情: user:id:{uuid}
	CacheKeyUserByUserName = "user:name" // 用户名查询: user:name:{username}
	CacheKeyUserList       = "user:list" // 用户列表: user:list
	CacheKeyUserListByDept = "user:dept" // 部门用户: user:dept:{deptId}

	// 角色相关
	CacheKeyRoleByID      = "role:id"      // 角色详情: role:id:{uuid}
	CacheKeyRoleList      = "role:list"    // 角色列表: role:list
	CacheKeyRoleByUserID  = "role:user"    // 用户角色: role:user:{userId}
	CacheKeyRoleAll       = "role:all"     // 所有角色: role:all
	CacheKeyRoleEnabled   = "role:enabled" // 启用的角色: role:enabled
	CacheKeyRoleMenus     = "role:menus"   // 角色菜单: role:menus:{roleId}
	CacheKeyRoleDepts     = "role:depts"   // 角色部门: role:depts:{roleId}

	// 菜单相关
	CacheKeyMenuByID     = "menu:id"      // 菜单详情: menu:id:{uuid}
	CacheKeyMenuList     = "menu:list"    // 菜单列表: menu:list
	CacheKeyMenuByRoleID = "menu:role"    // 角色菜单: menu:role:{roleId}
	CacheKeyMenuTree     = "menu:tree"    // 菜单树: menu:tree
	CacheKeyMenuRouter   = "menu:router"  // 菜单路由: menu:router
	CacheKeyMenuAll      = "menu:all"     // 所有菜单: menu:all
	// user-scoped 菜单缓存（按 userID 隔离，所有者和失效方是 menu 服务，故置于 menu: 命名空间）
	CacheKeyMenuUserMenus       = "menu:user:menus"  // 用户菜单树: menu:user:menus:{userID}
	CacheKeyMenuUserAllMenus    = "menu:user:all"    // 用户全部菜单(含隐藏): menu:user:all:{userID}
	CacheKeyMenuUserPermissions = "menu:user:perms"  // 用户权限标识: menu:user:perms:{userID}

	// 部门相关
	CacheKeyDeptByID     = "dept:id"       // 部门详情: dept:id:{uuid}
	CacheKeyDeptList     = "dept:list"     // 部门列表: dept:list
	CacheKeyDeptTree     = "dept:tree"     // 部门树: dept:tree
	CacheKeyDeptChildren = "dept:children" // 子部门: dept:children:{parentId}

	// 岗位相关
	CacheKeyPostByID     = "post:id"      // 岗位详情: post:id:{uuid}
	CacheKeyPostList     = "post:list"    // 岗位列表: post:list
	CacheKeyPostAll      = "post:all"     // 所有岗位: post:all
	CacheKeyPostEnabled  = "post:enabled" // 启用的岗位: post:enabled

	// 字典相关
	CacheKeyDictType       = "dict:type"      // 字典类型: dict:type
	CacheKeyDictData       = "dict:data"      // 字典数据: dict:data:{dictType}
	CacheKeyDictDataByType = "dict:data:type" // 字典数据列表: dict:data:type:{dictType}

	// 配置相关
	CacheKeyConfigByID  = "config:id"   // 配置详情: config:id:{uuid}
	CacheKeyConfigByKey = "config:key"  // 配置键: config:key:{configKey}
	CacheKeyConfigList  = "config:list" // 配置列表: config:list
)

// 运维模块缓存键
const (
	// 楼宇相关
	CacheKeyBuildingByID = "building:id"   // 楼宇详情: building:id:{uuid}
	CacheKeyBuildingList = "building:list" // 楼宇列表: building:list

	// 楼层相关
	CacheKeyFloorByID       = "floor:id"       // 楼层详情: floor:id:{uuid}
	CacheKeyFloorList       = "floor:list"     // 楼层列表: floor:list
	CacheKeyFloorByBuilding = "floor:building" // 楼宇楼层: floor:building:{buildingId}

	// 工位相关
	CacheKeyWorkstationByID    = "workstation:id"    // 工位详情: workstation:id:{uuid}
	CacheKeyWorkstationList    = "workstation:list"  // 工位列表: workstation:list
	CacheKeyWorkstationByFloor = "workstation:floor" // 楼层工位: workstation:floor:{floorId}

	// 信息点相关
	CacheKeyInfoPointByID = "infopoint:id"   // 信息点详情: infopoint:id:{uuid}
	CacheKeyInfoPointList = "infopoint:list" // 信息点列表: infopoint:list
)

// 监控模块缓存键
const (
	CacheKeyServerInfo  = "monitor:server"      // 服务器信息: monitor:server:{serverKey}
	CacheKeyCacheStats  = "monitor:cache:stats" // 缓存统计: monitor:cache:stats
	CacheKeyOnlineUsers = "monitor:online"      // 在线用户: monitor:online
)

// ==================== 缓存键构建辅助函数 ====================

// BuildUserCacheKey 构建用户缓存键
func BuildUserCacheKey(keyType string, params ...string) string {
	parts := append([]string{CachePrefixPrefix, ModuleUser, keyType}, params...)
	return strings.Join(parts, ":")
}

// BuildRoleCacheKey 构建角色缓存键
func BuildRoleCacheKey(keyType string, params ...string) string {
	parts := append([]string{CachePrefixPrefix, ModuleRole, keyType}, params...)
	return strings.Join(parts, ":")
}

// BuildMenuCacheKey 构建菜单缓存键
func BuildMenuCacheKey(keyType string, params ...string) string {
	parts := append([]string{CachePrefixPrefix, ModuleMenu, keyType}, params...)
	return strings.Join(parts, ":")
}

// BuildDeptCacheKey 构建部门缓存键
func BuildDeptCacheKey(keyType string, params ...string) string {
	parts := append([]string{CachePrefixPrefix, ModuleDept, keyType}, params...)
	return strings.Join(parts, ":")
}

// BuildPostCacheKey 构建岗位缓存键
func BuildPostCacheKey(keyType string, params ...string) string {
	parts := append([]string{CachePrefixPrefix, ModulePost, keyType}, params...)
	return strings.Join(parts, ":")
}

// BuildDictCacheKey 构建字典缓存键
func BuildDictCacheKey(keyType string, params ...string) string {
	parts := append([]string{CachePrefixPrefix, ModuleDict, keyType}, params...)
	return strings.Join(parts, ":")
}

// BuildConfigCacheKey 构建配置缓存键
func BuildConfigCacheKey(keyType string, params ...string) string {
	parts := append([]string{CachePrefixPrefix, ModuleConfig, keyType}, params...)
	return strings.Join(parts, ":")
}

// BuildBuildingCacheKey 构建楼宇缓存键
func BuildBuildingCacheKey(keyType string, params ...string) string {
	parts := append([]string{CachePrefixPrefix, ModuleOperations, "building", keyType}, params...)
	return strings.Join(parts, ":")
}

// BuildFloorCacheKey 构建楼层缓存键
func BuildFloorCacheKey(keyType string, params ...string) string {
	parts := append([]string{CachePrefixPrefix, ModuleOperations, "floor", keyType}, params...)
	return strings.Join(parts, ":")
}

// BuildWorkstationCacheKey 构建工位缓存键
func BuildWorkstationCacheKey(keyType string, params ...string) string {
	parts := append([]string{CachePrefixPrefix, ModuleOperations, "workstation", keyType}, params...)
	return strings.Join(parts, ":")
}

// ==================== 缓存过期时间常量 ====================
// 单位：秒

const (
	// 短期缓存（5分钟）
	CacheTTLShort = 5 * 60

	// 中期缓存（30分钟）
	CacheTTLMedium = 30 * 60

	// 长期缓存（2小时）
	CacheTTLLong = 2 * 60 * 60

	// 超长期缓存（24小时）
	CacheTTLVeryLong = 24 * 60 * 60
)

// GetCacheTTL 获取缓存过期时间
func GetCacheTTL(keyType string) time.Duration {
	switch keyType {
	// 用户数据使用中期缓存
	case CacheKeyUserByID, CacheKeyUserByUserName, CacheKeyUserList:
		return time.Duration(CacheTTLMedium) * time.Second

	// 角色数据使用长期缓存
	case CacheKeyRoleByID, CacheKeyRoleList:
		return time.Duration(CacheTTLLong) * time.Second

	// 菜单数据使用长期缓存
	case CacheKeyMenuByID, CacheKeyMenuList, CacheKeyMenuTree:
		return time.Duration(CacheTTLLong) * time.Second

	// 部门数据使用长期缓存
	case CacheKeyDeptByID, CacheKeyDeptList, CacheKeyDeptTree:
		return time.Duration(CacheTTLLong) * time.Second

	// 岗位数据使用超长期缓存
	case CacheKeyPostByID, CacheKeyPostList:
		return time.Duration(CacheTTLVeryLong) * time.Second

	// 字典数据使用超长期缓存
	case CacheKeyDictType, CacheKeyDictData:
		return time.Duration(CacheTTLVeryLong) * time.Second

	// 配置数据使用短期缓存
	case CacheKeyConfigByID, CacheKeyConfigByKey:
		return time.Duration(CacheTTLShort) * time.Second

	// 监控数据使用短期缓存
	case CacheKeyServerInfo, CacheKeyCacheStats:
		return time.Duration(CacheTTLShort) * time.Second

	default:
		return time.Duration(CacheTTLMedium) * time.Second
	}
}

// ==================== 缓存模式构建辅助函数 ====================

// BuildModulePattern 构建模块级别的缓存键模式
// 例如：BuildModulePattern(ModuleUser) -> "cache:user:*"
func BuildModulePattern(module string) string {
	return fmt.Sprintf("%s:%s:*", CachePrefixPrefix, module)
}

// BuildInvalidatePattern 构建失效缓存的模式
func BuildInvalidatePattern(module string, keyType string) string {
	if keyType == "" {
		return BuildModulePattern(module)
	}
	return fmt.Sprintf("%s:%s:%s:*", CachePrefixPrefix, module, keyType)
}

// ==================== 特殊缓存键辅助函数 ====================

// GetDictDataByTypeKey 根据字典类型构建字典数据缓存键
// 参数：dictType 字典类型
// 返回：dict:data:{dictType}
func GetDictDataByTypeKey(dictType string) string {
	return CacheKeyDictData + ":" + dictType
}

// GetMenuTreeKey 根据是否包含隐藏菜单构建菜单树缓存键
// 参数：includeHidden 是否包含隐藏菜单
// 返回：menu:tree 或 menu:tree:all
func GetMenuTreeKey(includeHidden bool) string {
	if includeHidden {
		return CacheKeyMenuTree + ":all"
	}
	return CacheKeyMenuTree
}

// GetRoleMenusKey 根据角色ID构建角色菜单缓存键
// 参数：roleID 角色ID
// 返回：role:menus:{roleID}
func GetRoleMenusKey(roleID string) string {
	return CacheKeyRoleMenus + ":" + roleID
}

// GetUserByIDKey 根据用户ID构建用户缓存键
// 参数：id 用户ID
// 返回：user:id:{id}
func GetUserByIDKey(id string) string {
	return CacheKeyUserByID + ":" + id
}

// GetUserByUsernameKey 根据用户名构建用户缓存键
// 参数：username 用户名
// 返回：user:name:{username}
func GetUserByUsernameKey(username string) string {
	return CacheKeyUserByUserName + ":" + username
}

// GetUserRolesKey 根据用户ID构建用户角色缓存键
// 参数：userID 用户ID
// 返回：role:user:{userID}
func GetUserRolesKey(userID string) string {
	return CacheKeyRoleByUserID + ":" + userID
}

// GetUserPermissionsKey 根据用户ID构建用户权限缓存键
// 参数：userID 用户ID
// 返回：user:permissions:{userID}
func GetUserPermissionsKey(userID string) string {
	return "user:permissions:" + userID
}

// GetMenuUserMenusKey 根据 userID 构建用户菜单树缓存键
// 参数：userID 用户ID
// 返回：menu:user:menus:{userID}
func GetMenuUserMenusKey(userID string) string {
	return CacheKeyMenuUserMenus + ":" + userID
}

// GetMenuUserAllMenusKey 根据 userID 构建用户全部菜单(含隐藏)缓存键
// 参数：userID 用户ID
// 返回：menu:user:all:{userID}
func GetMenuUserAllMenusKey(userID string) string {
	return CacheKeyMenuUserAllMenus + ":" + userID
}

// GetMenuUserPermissionsKey 根据 userID 构建用户权限标识缓存键
// 命名加 Menu 前缀以与既有 GetUserPermissionsKey（user:permissions:{id}，user 模块命名空间，未被缓存层使用）区分
// 参数：userID 用户ID
// 返回：menu:user:perms:{userID}
func GetMenuUserPermissionsKey(userID string) string {
	return CacheKeyMenuUserPermissions + ":" + userID
}
