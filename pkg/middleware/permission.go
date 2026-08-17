package middleware

import (
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"gorm.io/gorm"
)

// ==================== 中间件权限结果短 TTL 缓存 (login-menu-timeout-20260817 H7 修复) ====================
//
// 背景: isSuperAdmin / checkUserPermission 此前对每个受保护请求都执行无缓存 SQL
// (sys_user_role × sys_role / sys_role_menu × sys_menu JOIN)。在远程高延迟 DB 上,
// 登录 + 首批页面加载会串行触发 10+ 条此类慢查询,是登录链路超时的核心放大器。
//
// 设计:
//   - 进程内 sync.Map + 过期时间戳,TTL 30s,key 按 userID(及 permission)隔离。
//   - 不用 core.Cache 的原因: MultiLevelCache.Set 的 L1 写入硬编码 5min TTL,
//     会把 30s 语义静默放大到 5min;且权限判断是超热路径,进程内读零网络开销。
//   - 多实例部署各实例独立缓存 30s,可接受。
//
// 陈旧性(已接受的 tradeoff, documented):
//   - 角色/菜单权限变更后,最长 30s 内中间件可能返回旧结果(提权/降权窗口)。
//   - 该窗口远小于 JWT access_token TTL(7200s),也小于 menuCacheService 的 30min
//     菜单缓存,不引入新的语义。需要立即生效时调用 InvalidateMiddlewarePermCache。
const mwPermCacheTTL = 30 * time.Second

type mwPermCacheEntry struct {
	allow     bool
	expiresAt time.Time
}

var (
	mwSuperAdminCache sync.Map // key: userID → mwPermCacheEntry
	mwPermCheckCache  sync.Map // key: userID + "|" + permission → mwPermCacheEntry
)

// mwCacheLookup 查询缓存;命中且未过期返回 (allow, true)。过期条目惰性删除。
func mwCacheLookup(c *sync.Map, key string) (bool, bool) {
	if v, ok := c.Load(key); ok {
		if entry, ok := v.(mwPermCacheEntry); ok {
			if time.Now().Before(entry.expiresAt) {
				return entry.allow, true
			}
			c.Delete(key) // 过期清理(惰性)
		}
	}
	return false, false
}

func mwCacheStore(c *sync.Map, key string, allow bool) {
	c.Store(key, mwPermCacheEntry{allow: allow, expiresAt: time.Now().Add(mwPermCacheTTL)})
}

// InvalidateMiddlewarePermCache 使指定用户的中间件权限缓存立即失效。
// 供角色/用户角色变更路径调用;当前未接线 — 30s 短 TTL 陈旧已被接受(见上方注释)。
func InvalidateMiddlewarePermCache(userID string) {
	mwSuperAdminCache.Delete(userID)
	prefix := userID + "|"
	mwPermCheckCache.Range(func(k, _ interface{}) bool {
		if key, ok := k.(string); ok && strings.HasPrefix(key, prefix) {
			mwPermCheckCache.Delete(k)
		}
		return true
	})
}

// 允许的数据权限字段白名单
var allowedDataScopeFields = map[string]bool{
	"dept_id": true,
	"id":      true,
}

// isValidDataScopeField 验证字段名是否在白名单中
func isValidDataScopeField(field string) bool {
	return allowedDataScopeFields[field]
}

// getUserIDAsString 安全地从context获取user_id
func getUserIDAsString(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	userIDStr, ok := userID.(string)
	if !ok {
		return "", false
	}
	return userIDStr, true
}

// Permission 权限中间件
func Permission(permission string, core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		userID, ok := getUserIDAsString(c)
		if !ok {
			response.Error(c, response.ErrUnauthorized, "用户未认证")
			c.Abort()
			return
		}

		// 检查用户权限
		if !checkUserPermission(core, userID, permission) {
			response.Error(c, response.ErrForbidden, "没有访问权限")
			c.Abort()
			return
		}

		c.Next()
	}
}

// checkUserPermission 检查用户权限 (带 30s 短 TTL 缓存,见文件头 H7 修复说明)
// 注意：只检查 status（停用），不检查 visible（隐藏）
// - 隐藏的菜单（visible=0）：不显示在导航栏，但权限仍然有效，可通过直接URL访问
// - 停用的菜单（status=1）：完全不可用，权限无效，即使知道URL也无法访问
func checkUserPermission(core *core.Core, userID, permission string) bool {
	cacheKey := userID + "|" + permission
	if allow, hit := mwCacheLookup(&mwPermCheckCache, cacheKey); hit {
		return allow
	}
	allow := checkUserPermissionUncached(core, userID, permission)
	mwCacheStore(&mwPermCheckCache, cacheKey, allow)
	return allow
}

// checkUserPermissionUncached 权限检查的原始实现(每次执行 DB 查询)。
func checkUserPermissionUncached(core *core.Core, userID, permission string) bool {
	// 超级管理员直接通过
	if isSuperAdmin(core, userID) {
		return true
	}

	// 1. 首先检查精确匹配的权限
	var count int64
	applogger.Infof("[Permission Check] userID=%s, permission=%s", userID, permission)
	err := core.DB.GetDB().Raw(`
		SELECT COUNT(DISTINCT rm.menu_id)
		FROM sys_user_role ur
		INNER JOIN sys_role_menu rm ON ur.role_id = rm.role_id
		INNER JOIN sys_menu m ON rm.menu_id = m.id
		WHERE ur.user_id = ? AND m.perms = ? AND m.status = ?
	`, userID, permission, models.MenuStatusNormal).Scan(&count).Error

	applogger.Infof("[Permission Check] exact match: count=%d, err=%v", count, err)
	if err == nil && count > 0 {
		applogger.Infof("[Permission Check] exact match PASS")
		return true
	}

	// 2. 如果精确匹配失败，尝试通过子菜单权限检查父菜单权限
	// 支持子菜单权限自动包含父菜单权限：勾选子菜单时，自动拥有父菜单的页面访问权限
	// 例如：用户有 vdi:vm:query 权限，检查 vdi:vm:list 时应该通过
	//
	// P1 fix (权限提升): 仅允许"按钮类(F)子权限 → 父菜单页面访问权"继承,
	// 不允许"菜单类(C)子权限 → 父菜单",否则用户拥有任一子页面就自动获得父页面
	// 的全部 :list/:view 权限,在精细化数据权限场景构成越权读取。
	var menu models.Menu
	err = core.DB.GetDB().Where("perms = ? AND status = ?", permission, models.MenuStatusNormal).First(&menu).Error
	if err == nil && menu.ID != "" {
		// 查找该菜单的所有"按钮"子菜单 (menu_type='F')
		var childMenus []models.Menu
		err = core.DB.GetDB().Where("parent_id = ? AND menu_type = ? AND status = ?",
			menu.ID, models.MenuTypeButton, models.MenuStatusNormal).Find(&childMenus).Error
		if err == nil {
			// 提取子菜单的权限标识
			childPerms := make([]interface{}, 0)
			for _, child := range childMenus {
				if child.Perms != nil && *child.Perms != "" {
					childPerms = append(childPerms, *child.Perms)
				}
			}
			// 检查用户是否有任意子菜单的权限
			if len(childPerms) > 0 {
				err = core.DB.GetDB().Raw(`
					SELECT COUNT(DISTINCT rm.menu_id)
					FROM sys_user_role ur
					INNER JOIN sys_role_menu rm ON ur.role_id = rm.role_id
					INNER JOIN sys_menu m ON rm.menu_id = m.id
					WHERE ur.user_id = ? AND m.perms IN ? AND m.status = ?
				`, userID, childPerms, models.MenuStatusNormal).Scan(&count).Error

				applogger.Infof("[Permission Check] child menu match: count=%d, childPerms=%v, err=%v", count, childPerms, err)
				if err == nil && count > 0 {
					applogger.Infof("[Permission Check] child menu match PASS (子菜单权限自动包含父菜单权限)")
					return true
				}
			}
		}
	}

	// 3. 如果精确匹配失败，检查模块级权限
	// 例如：system:menu:list -> system:menu
	modulePermission := extractModulePermission(permission)
	applogger.Infof("[Permission Check] modulePermission=%s", modulePermission)
	if modulePermission != "" {
		err = core.DB.GetDB().Raw(`
			SELECT COUNT(DISTINCT rm.menu_id)
			FROM sys_user_role ur
			INNER JOIN sys_role_menu rm ON ur.role_id = rm.role_id
			INNER JOIN sys_menu m ON rm.menu_id = m.id
			WHERE ur.user_id = ? AND m.perms = ? AND m.status = ?
		`, userID, modulePermission, models.MenuStatusNormal).Scan(&count).Error

		applogger.Infof("[Permission Check] module match: count=%d, err=%v", count, err)
		if err == nil && count > 0 {
			applogger.Infof("[Permission Check] module match PASS")
			return true
		}
	}

	applogger.Infof("[Permission Check] ALL CHECKS FAILED, returning false")
	return false
}

// extractModulePermission 从具体权限中提取模块权限
// 例如：system:menu:list -> system:menu
// 例如：system:user:add -> system:user
func extractModulePermission(permission string) string {
	// 找到最后一个冒号的位置
	lastColon := -1
	for i := len(permission) - 1; i >= 0; i-- {
		if permission[i] == ':' {
			lastColon = i
			break
		}
	}

	// 如果没有冒号或者只有一个部分，返回空
	if lastColon <= 0 {
		return ""
	}

	// 返回冒号之前的部分（模块权限）
	return permission[:lastColon]
}

// isSuperAdmin 检查是否是超级管理员 (带 30s 短 TTL 缓存,见文件头 H7 修复说明)
func isSuperAdmin(core *core.Core, userID string) bool {
	if allow, hit := mwCacheLookup(&mwSuperAdminCache, userID); hit {
		return allow
	}

	// 检查用户是否有超级管理员角色
	var count int64
	err := core.DB.GetDB().Raw(`
		SELECT COUNT(*)
		FROM sys_user_role ur
		INNER JOIN sys_role r ON ur.role_id = r.id
		WHERE ur.user_id = ? AND r.role_key = ?
	`, userID, "admin").Scan(&count).Error

	if err != nil {
		return false // DB 错误不缓存,下次请求重试
	}

	allow := count > 0
	mwCacheStore(&mwSuperAdminCache, userID, allow)
	return allow
}

// GetCurrentUserPermissions 获取当前用户所有权限
func GetCurrentUserPermissions(core *core.Core, userID string) ([]string, error) {
	permissionSvc := permission.NewService()
	return permissionSvc.GetUserPermissions(core.GetDB(), userID)
}

// RequirePermissions 需要多个权限中的任意一个
func RequirePermissions(permissions []string, core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := getUserIDAsString(c)
		if !ok {
			response.Error(c, response.ErrUnauthorized, "用户未认证")
			c.Abort()
			return
		}

		// 超级管理员直接通过
		if isSuperAdmin(core, userID) {
			c.Next()
			return
		}

		// 检查是否有任意一个权限
		for _, permission := range permissions {
			if checkUserPermission(core, userID, permission) {
				c.Next()
				return
			}
		}

		response.Error(c, response.ErrForbidden, "没有访问权限")
		c.Abort()
	}
}

// RequirePermissionsWithQuery 需要多个权限中的任意一个；对只读查询路径（末段为 list/tree）
// 额外接受 queryExtraPermissions 中的任一权限。
//
// 用途：只读可视化页面（如「楼宇空间」「楼宇空间3D」）复用了 CRUD 模块的 list 接口拼装数据，
// 但页面菜单自身的权限标识（ops:building:spaces:list）与 CRUD 模块的权限标识（ops:building:list）
// 属于不同命名空间。此中间件让查询类路径额外放行可视化读权限，而 create/update/delete 等写操作
// 仍只受严格权限保护，避免越权。
//
// 例：空间管理角色持有 ops:building:spaces:list
//   - POST /ops/building/list → 命中 queryExtraPermissions → 通过
//   - POST /ops/building      → 写操作，严格权限 → 拒绝（安全）
func RequirePermissionsWithQuery(permissions []string, queryExtraPermissions []string, core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := getUserIDAsString(c)
		if !ok {
			response.Error(c, response.ErrUnauthorized, "用户未认证")
			c.Abort()
			return
		}

		// 超级管理员直接通过
		if isSuperAdmin(core, userID) {
			c.Next()
			return
		}

		// 查询类路径额外纳入可视化读权限；写操作保持严格权限
		checkPerms := permissions
		if len(queryExtraPermissions) > 0 && isQueryPath(c.Request.URL.Path) {
			merged := make([]string, 0, len(permissions)+len(queryExtraPermissions))
			merged = append(merged, permissions...)
			merged = append(merged, queryExtraPermissions...)
			checkPerms = merged
		}

		for _, permission := range checkPerms {
			if checkUserPermission(core, userID, permission) {
				c.Next()
				return
			}
		}

		response.Error(c, response.ErrForbidden, "没有访问权限")
		c.Abort()
	}
}

// isQueryPath 判断是否为只读查询路径（末段为 list 或 tree）。
// 写操作路径（create/update/delete/batch/geocode/import/export 等）返回 false。
func isQueryPath(path string) bool {
	p := strings.TrimSuffix(path, "/")
	idx := strings.LastIndex(p, "/")
	if idx < 0 {
		return false
	}
	last := p[idx+1:]
	return last == "list" || last == "tree"
}

// RequireAllPermissions 需要所有权限
func RequireAllPermissions(permissions []string, core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := getUserIDAsString(c)
		if !ok {
			response.Error(c, response.ErrUnauthorized, "用户未认证")
			c.Abort()
			return
		}

		// 超级管理员直接通过
		if isSuperAdmin(core, userID) {
			c.Next()
			return
		}

		// 检查是否拥有所有权限
		for _, permission := range permissions {
			if !checkUserPermission(core, userID, permission) {
				response.Error(c, response.ErrForbidden, "没有访问权限")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// DataScopePermission 数据权限中间件
func DataScopePermission(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := getUserIDAsString(c)
		if !ok {
			response.Error(c, response.ErrUnauthorized, "用户未认证")
			c.Abort()
			return
		}

		// 超级管理员直接通过
		if isSuperAdmin(core, userID) {
			c.Next()
			return
		}

		// 获取用户最大数据权限范围
		dataScope, err := getUserMaxDataScope(core, userID)
		if err != nil {
			response.Error(c, response.ErrServerError, "获取数据权限失败")
			c.Abort()
			return
		}

		// 将数据权限信息存储到上下文
		c.Set("data_scope", dataScope)
		c.Set("user_id", userID)

		c.Next()
	}
}

// getUserMaxDataScope 获取用户最大数据权限范围
func getUserMaxDataScope(core *core.Core, userID string) (models.DataScope, error) {
	var dataScope models.DataScope

	err := core.DB.GetDB().Raw(`
		SELECT COALESCE(MAX(r.data_scope), ?) as data_scope
		FROM sys_user_role ur
		INNER JOIN sys_role r ON ur.role_id = r.id
		WHERE ur.user_id = ? AND r.status = ?
	`, models.DataScopeSelf, userID, models.RoleStatusEnabled).Scan(&dataScope).Error

	return dataScope, err
}

// ApplyDataScope 应用数据权限过滤
func ApplyDataScope(db *gorm.DB, userID string, dataScope models.DataScope, core *core.Core, deptField string) *gorm.DB {
	// 验证字段名安全性，防止SQL注入
	if !isValidDataScopeField(deptField) {
		return db.Where("1=0") // 无效字段名，返回空结果
	}

	switch dataScope {
	case models.DataScopeAll:
		// 全部数据权限，不做过滤
		return db
	case models.DataScopeCustom:
		// 自定义数据权限，查询用户可访问的部门
		var deptIds []string
		core.DB.GetDB().Raw(`
			SELECT DISTINCT rd.dept_id
			FROM sys_user_role ur
			INNER JOIN sys_role_dept rd ON ur.role_id = rd.role_id
			WHERE ur.user_id = ?
		`, userID).Scan(&deptIds)

		if len(deptIds) > 0 {
			return db.Where(deptField+" IN ?", deptIds)
		}
		return db.Where("1=0") // 没有权限访问任何数据
	case models.DataScopeDept:
		// 本部门数据权限，查询用户所属部门
		var deptId string
		if err := core.DB.GetDB().Raw("SELECT dept_id FROM sys_user WHERE id = ?", userID).Scan(&deptId).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				// 非记录未找到错误，记录数据库错误日志
				applogger.Errorf("Failed to query user dept for data scope filtering (user_id=%s): %v", userID, err)
			}
			return db.Where("1=0")
		}
		if deptId != "" {
			return db.Where(deptField+" = ?", deptId)
		}
		return db.Where("1=0")
	case models.DataScopeDeptChild:
		// 本部门及子部门数据权限
		var deptId string
		if err := core.DB.GetDB().Raw("SELECT dept_id FROM sys_user WHERE id = ?", userID).Scan(&deptId).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				// 非记录未找到错误，记录数据库错误日志
				applogger.Errorf("Failed to query user dept for data scope filtering (user_id=%s): %v", userID, err)
			}
			return db.Where("1=0")
		}
		if deptId != "" {
			// 查询本部门及所有子部门
			var childDeptIds []string
			childDeptIds = append(childDeptIds, deptId)
			getChildDepts(core, deptId, &childDeptIds)

			return db.Where(deptField+" IN ?", childDeptIds)
		}
		return db.Where("1=0")
	case models.DataScopeSelf:
		// 仅本人数据权限
		return db.Where(deptField+" = ?", userID)
	default:
		return db.Where("1=0") // 默认无权限
	}
}

// getChildDepts 递归获取子部门
func getChildDepts(core *core.Core, parentId string, deptIds *[]string) {
	var childDepts []struct {
		ID string `gorm:"column:id"`
	}

	core.DB.GetDB().Table("sys_dept").Where("parent_id = ?", parentId).Select("id").Scan(&childDepts)

	for _, dept := range childDepts {
		*deptIds = append(*deptIds, dept.ID)
		getChildDepts(core, dept.ID, deptIds)
	}
}

// ApplyDataScopeFromContext 从 Gin 上下文中应用数据权限过滤
// 这是一个便捷函数，用于在 handler 中应用数据权限
// deptField 是要过滤的部门字段名，如 "dept_id" 或 "id"（对于部门表本身）
func ApplyDataScopeFromContext(c *gin.Context, db *gorm.DB, core *core.Core, deptField string) *gorm.DB {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		return db
	}

	// 检查是否是超级管理员
	if isSuperAdmin(core, userID.(string)) {
		return db // 超级管理员不过滤数据
	}

	// 从上下文中获取数据权限范围
	dataScope, exists := c.Get("data_scope")
	if !exists {
		return db // 没有数据权限信息，不做过滤
	}

	// 应用数据权限过滤
	return ApplyDataScope(db, userID.(string), dataScope.(models.DataScope), core, deptField)
}
