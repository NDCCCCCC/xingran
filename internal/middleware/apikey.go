package middleware

import (
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"gorm.io/gorm"
)

const (
	apiKeyHeader = "X-API-Key"
	keyPrefix    = "rec_"
)

// MultiAuth 多重认证中间件（JWT + API Key）
// 如果提供 X-API-Key 请求头，使用 API Key 认证
// 否则跳过，允许 JWT 认证中间件处理
//
// Phase 61 / D-06 扩展: 新增 permSvc + db 参数用于 InheritPerms=true 时实时加载
// User 权限代码并与 API Key scopes 取并集。permission.Service 是 interface 注入,
// db 透传给 GetUserPermissions (service.go:272 接收 *gorm.DB)。InheritPerms=false
// 时 permSvc/db 不被调用,传 nil 也无副作用(测试 stub 场景)。
func MultiAuth(apiKeyService system.APIKeyService, usageLogger services.UsageLogger, permSvc permission.Service, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 提取 API Key
		apiKeyStr := extractAPIKey(c)
		if apiKeyStr == "" {
			// 没有 API Key，跳过（允许 JWT 认证）
			c.Next()
			return
		}

		// 验证密钥格式
		if !isValidKeyFormat(apiKeyStr) {
			response.Error(c, response.ErrUnauthorized, "无效的密钥格式")
			c.Abort()
			return
		}

		// 验证密钥
		apiKey, err := apiKeyService.ValidateAPIKey(c.Request.Context(), apiKeyStr)
		if err != nil {
			response.Error(c, response.ErrUnauthorized, "密钥验证失败: "+err.Error())
			c.Abort()
			return
		}

		// 验证 IP 白名单（GORM已自动反序列化为[]string）
		if len(apiKey.IPWhitelist) > 0 {
			clientIP := c.ClientIP()
			if !isIPAllowed(clientIP, apiKey.IPWhitelist) {
				response.Error(c, response.ErrForbidden, "客户端IP不在白名单中")
				c.Abort()
				return
			}
		}

		// 设置用户上下文（GORM已自动反序列化Scopes为[]string）
		// Phase 61 / D-06/D-07/D-09/D-10: InheritPerms 加载路径 + username/nickname 修正
		setUserContextForAPIKey(c, apiKey, apiKey.Scopes, permSvc, db)

		// OBSERV-01: 计时前置，c.Next() 后捕获真实响应结果。
		// 先例: pkg/middleware/logger.go:19-28 + :47-49（gin 标准记录模式）。
		start := time.Now()
		c.Next() // 下游 handler / RequireScope / RateLimitByScope 执行完毕

		// c.Next() 返回后: 真实状态码 / 耗时此刻才可用 (OBSERV-01)
		statusCode := c.Writer.Status()
		duration := time.Since(start).Milliseconds()

		userID := ""
		if apiKey.UserID != nil {
			userID = *apiKey.UserID
		}

		// D-02a: middleware 不再包外层 go func; LogUsage 内部已 go logUsageAsync()。
		usageLogger.LogUsage(c.Request.Context(), &services.LogUsageRequest{
			APIKeyID:   apiKey.ID,
			UserID:     userID,
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			ClientIP:   c.ClientIP(),
			StatusCode: statusCode,                            // 新填 (OBSERV-01)
			Duration:   int(duration),                         // 新填 (OBSERV-01)
			Success:    statusCode >= 200 && statusCode < 300, // 新填 (D-01)
		})
	}
}

// extractAPIKey 从请求头提取 API Key（私有函数）
func extractAPIKey(c *gin.Context) string {
	return c.GetHeader(apiKeyHeader)
}

// isValidKeyFormat 验证密钥格式（私有函数）
// 格式: rec_ + 64位十六进制字符 = 68字符
func isValidKeyFormat(key string) bool {
	// 检查长度: 4（前缀）+ 64（hex）= 68
	if len(key) != 68 {
		return false
	}

	// 检查前缀
	if !strings.HasPrefix(key, keyPrefix) {
		return false
	}

	// 检查后64位是否为有效十六进制
	hexPart := key[4:]
	for _, c := range hexPart {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}

	return true
}

// isIPAllowed 检查客户端IP是否在白名单中（私有函数）
// 支持单个IP（192.168.1.1）和CIDR（192.168.1.0/24）
func isIPAllowed(clientIP string, whitelist []string) bool {
	// 如果白名单为空，允许所有
	if len(whitelist) == 0 {
		return true
	}

	// 解析客户端IP
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	// 遍历白名单
	for _, allowed := range whitelist {
		if strings.Contains(allowed, "/") {
			// CIDR格式
			_, ipNet, err := net.ParseCIDR(allowed)
			if err != nil {
				continue
			}
			if ipNet.Contains(ip) {
				return true
			}
		} else {
			// 单个IP
			if clientIP == allowed {
				return true
			}
		}
	}

	return false
}

// setUserContextForAPIKey 设置API Key认证的用户上下文（私有函数）
//
// Phase 61 / D-06/D-07/D-09/D-10 改造:
//   - 新增 permSvc 参数用于 InheritPerms=true 时实时加载 User 权限代码
//     (GetUserPermissions),与 API Key 自带 scopes 取并集写入 c.Set("scopes")
//   - D-07: 不引入缓存,每请求一次 DB 查询(InheritPerms=true 时)
//   - D-08: InheritPerms=false → 行为不变,scopes 仅含 API Key 自带
//   - D-09: 加载失败(DB error / UserID 为 nil)→ 401 fail-closed,abort 后续步骤
//   - D-10: username/nickname 从 apiKey.User 关联读取(apiKey.User 已在
//     ValidateAPIKey 中 Preload)。User 关联缺失时兜底 apiKey.Name + Warnf
func setUserContextForAPIKey(c *gin.Context, apiKey *models.APIKey, scopes []string, permSvc permission.Service, db *gorm.DB) {
	userID := ""
	if apiKey.UserID != nil {
		userID = *apiKey.UserID
	}

	// 设置上下文
	c.Set("user_id", userID)

	// D-10: username/nickname 取 apiKey.User 关联值。User 关联缺失时
	// 兜底 apiKey.Name + 记录 Warnf(不应发生,因为 ValidateAPIKey 已 Preload)
	if apiKey.User != nil {
		c.Set("username", apiKey.User.Username)
		nickname := ""
		if apiKey.User.Nickname != nil {
			nickname = *apiKey.User.Nickname
		}
		c.Set("nickname", nickname)
	} else {
		applogger.Warnf("[API_KEY] username/nickname 缺失 User 关联: apiKeyID=%s name=%s",
			apiKey.ID, apiKey.Name)
		c.Set("username", apiKey.Name) // 兜底: 与 D-04 既有行为一致
		c.Set("nickname", "")
	}

	c.Set("api_key_id", apiKey.ID)
	c.Set("scopes", scopes)
	c.Set("auth_type", "api_key")

	// D-06/D-07/D-09: InheritPerms=true 时实时加载 User 权限代码与 scopes 取并集。
	// 每请求一次 DB 查询(D-07),失败 → 401 fail-closed(D-09)。
	//
	// 注:顺序在 c.Set("auth_type", "api_key") 之后、c.Set("api_key_id") 之前
	// 不重要 — InheritPerms 分支在 abort 路径上不影响已设的 context 键。
	if apiKey.InheritPerms {
		// D-09 校验: UserID 为 nil 视为配置错误,fail-closed
		if apiKey.UserID == nil {
			applogger.Errorf("[API_KEY] InheritPerms=true 但 UserID 为 nil: apiKeyID=%s", apiKey.ID)
			response.Error(c, response.ErrUnauthorized, "用户权限加载失败")
			c.Abort()
			return
		}

		// 加载 User 权限代码 (每请求一次 DB, 无缓存 D-07)
		userPerms, err := permSvc.GetUserPermissions(db, *apiKey.UserID)
		if err != nil {
			// D-09: fail-closed — 加载失败意味着「无法判断是否真有权限」
			applogger.Errorf("[API_KEY] 加载 User 权限失败: apiKeyID=%s userID=%s err=%v",
				apiKey.ID, *apiKey.UserID, err)
			response.Error(c, response.ErrUnauthorized, "用户权限加载失败")
			c.Abort()
			return
		}

		// D-06: scopes 并集写入。粗粒度 read/write/admin(scope)与细粒度
		// system:user:list(permission code)同入 c.scopes 同一集合,
		// 供 RequireScope / getScopeFromContext / RequireAPIKeyResourcePermission
		// 检。
		mergedScopes := append([]string{}, scopes...)
		seen := make(map[string]struct{}, len(mergedScopes))
		for _, s := range mergedScopes {
			seen[s] = struct{}{}
		}
		for _, p := range userPerms {
			if _, ok := seen[p]; !ok {
				mergedScopes = append(mergedScopes, p)
				seen[p] = struct{}{}
			}
		}
		c.Set("scopes", mergedScopes)
		c.Set("inherit_perms", true)
	}
}

// RequireScope 作用域验证中间件
// 验证API Key是否具有所需的作用域
// 支持层级权限：admin > write > read
func RequireScope(requiredScope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取作用域
		scopes, exists := c.Get("scopes")
		if !exists {
			response.Error(c, response.ErrForbidden, "缺少权限作用域")
			c.Abort()
			return
		}

		// 类型断言
		userScopes, ok := scopes.([]string)
		if !ok {
			response.Error(c, response.ErrForbidden, "权限作用域格式错误")
			c.Abort()
			return
		}

		// 检查是否有所需作用域或admin权限（admin包含所有权限）
		hasScope := false
		for _, scope := range userScopes {
			if scope == "admin" || scope == requiredScope {
				hasScope = true
				break
			}
		}

		if !hasScope {
			response.Error(c, response.ErrForbidden, "权限不足，需要作用域: "+requiredScope)
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAPIKeyResourcePermission API Key资源权限验证中间件
// 根据资源和操作映射到相应的作用域
//
// Phase 61 / AUTH-04 改造:
//   - D-03 fail-closed: 查 pkg/permission/resource_action_map 命中 → 走 scope
//     check 路径;未命中 → 403 "资源权限未定义" + abort(不 c.Next())
//   - D-05: 本函数仍为公共 helper,本 phase 不挂载到 apikey_router.go(仅做
//     单元测试 + 文档化),最小爆炸半径
//   - 命中后 scopes 检查接受三种形式(union 语义):
//     ① admin 通配 — admin 含所有权限
//     ② 细粒度 PermissionCode — D-06 InheritPerms=true 路径下 User 权限代码
//        合并入 scopes,例如 "system:user:list"
//     ③ 粗粒度 scope(read/write) — D-13 非 InheritPerms 路径下 API Key 自带
//        scope 通过 getRequiredScope(action) 映射(view→read, edit→write)
//   - 不命中 → 403 "资源权限不足: <permCode>"
func RequireAPIKeyResourcePermission(resource string, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// D-03: 第一步 — 查静态 map。未命中 → fail-closed 403
		permCode, ok := permission.LookupResourceAction(resource, action)
		if !ok {
			response.Error(c, response.ErrForbidden, "资源权限未定义: "+resource+":"+action)
			c.Abort()
			return
		}

		// 第二步 — scopes 集合检查
		scopesVal, exists := c.Get("scopes")
		if !exists {
			response.Error(c, response.ErrForbidden, "权限作用域不足: "+string(permCode))
			c.Abort()
			return
		}
		userScopes, ok := scopesVal.([]string)
		if !ok {
			response.Error(c, response.ErrForbidden, "权限作用域格式错误")
			c.Abort()
			return
		}

		// 三种形式 union 匹配: admin 通配 / 细粒度 PermissionCode / 粗粒度 scope
		// BL-01 修复: 粗粒度支路仅在 action 属于已知词汇表(actionScopeMap)时参与
		// 匹配;未知 action 不再兜底 "read" — 粗粒度支路直接不匹配(fail-closed,
		// 与 D-03/D-12 一致),只剩 admin 通配 / 细粒度 PermissionCode 两支路可放行。
		requiredScope, actionKnown := actionScopeMap[action]
		hasPerm := false
		for _, scope := range userScopes {
			if scope == "admin" || scope == string(permCode) || (actionKnown && scope == requiredScope) {
				hasPerm = true
				break
			}
		}
		if !hasPerm {
			response.Error(c, response.ErrForbidden, "资源权限不足: "+string(permCode))
			c.Abort()
			return
		}

		c.Next()
	}
}

// actionScopeMap action → 所需 scope 词汇表(包级唯一真相源)。
//
// BL-01 修复(Phase 61 review): 与 pkg/permission/resource_action_map.go 的
// D-04 action 词汇表对齐 — list/view/add/edit/remove/export/import/resetPwd
// 全覆盖;create/delete 为历史别名保留(create≡add, delete≡remove,
// 见 resource_action_map.go 头注「remove 是 system:* 约定, delete 是 network:* 约定」)。
// export 归为 read(只读导出,不修改数据)。
var actionScopeMap = map[string]string{
	"list":     "read",
	"view":     "read",
	"add":      "write",
	"edit":     "write",
	"remove":   "write",
	"create":   "write", // 历史别名, 同 add
	"delete":   "write", // 历史别名, 同 remove
	"export":   "read",  // 只读导出
	"import":   "write",
	"resetPwd": "write",
}

// getRequiredScope 根据操作获取所需作用域（私有函数）
// 映射规则: 见 actionScopeMap(BL-01 修复后与 resource_action_map D-04 词汇表对齐)。
// 未知 action 兜底 "read" — 仅 RateLimitByScope/SelectScope 限流路径使用(D-11);
// RequireAPIKeyResourcePermission 鉴权路径对未知 action fail-closed(见该函数内
// actionKnown 判断),不经过本兜底。
func getRequiredScope(action string) string {
	if scope, ok := actionScopeMap[action]; ok {
		return scope
	}
	return "read"
}

// RateLimitByScope 基于作用域的速率限制中间件
// 根据API Key的作用域动态调整速率限制
// 符合 RFC 6585 规范（429 Too Many Requests）
//
// Phase 61 / D-11/D-12/D-14 改造:
//   - D-11: 新增 action 参数,注册期计算 requiredScope 闭包捕获(每请求零查表开销)
//   - D-12: 多 scope 选择改 action-aware — 委托 getScopeFromContext → SelectScope
//     纯函数;scopes 不含 required scope 且无 admin → 403 fail-closed(无 fallback),
//     不进入限流检查
//   - D-14: 与 RequireScope 职责分工 — RequireScope 做硬鉴权(不感知 action),
//     RateLimitByScope 做精细限流(感知 action 选择限额档位),两个中间件都保留
func RateLimitByScope(rateLimiter *services.RateLimiter, action string) gin.HandlerFunc {
	// D-11: 注册期计算 requiredScope,闭包捕获
	requiredScope := getRequiredScope(action)
	return func(c *gin.Context) {
		// 检查是否为API Key认证
		authType, exists := c.Get("auth_type")
		if !exists || authType != "api_key" {
			// 不是API Key认证，跳过速率限制
			c.Next()
			return
		}

		// D-12: action-aware 多 scope 选择,fail-closed
		scope, allowed := getScopeFromContext(c, action)
		if !allowed {
			applogger.Warnf("[API_KEY] 限流作用域不足: action=%s required_scope=%s path=%s",
				action, requiredScope, c.Request.URL.Path)
			response.Error(c, response.ErrForbidden, "权限作用域不足")
			c.Abort()
			return
		}

		// 获取唯一标识符
		identifier := getIdentifier(c)

		// 检查速率限制
		allowed, result := rateLimiter.Check(identifier, scope)

		// 设置速率限制响应头（RFC 6585）
		// QUAL-01 / D-11: 修复 P2-a —— 原 string(rune(int)) 把整数当 Unicode 码点转换
		// (Limit=100 → "d")，改用 strconv.Itoa 输出数字字面量 ("100")，前端 / 第三方
		// 工具可用标准 parseInt / strconv.Atoi 反解析。
		c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", result.ResetAt.Format(time.RFC3339))

		if !allowed {
			// 超过速率限制，返回429
			c.Header("Retry-After", "60") // 建议60秒后重试
			response.Error(c, 429, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

// getScopeFromContext 从上下文获取作用域（私有函数）
//
// Phase 61 / D-12/D-13 改造: 薄壳包装 SelectScope 纯函数(select_scope.go),
// 从既有 context 键 inherit_perms / scopes 读取(Phase 57 D-04 既有 7 键,
// 不新增内部中转键),返回 (scope, allowed) 供调用方 fail-closed 处理。
func getScopeFromContext(c *gin.Context, action string) (string, bool) {
	inheritRaw, exists := c.Get("inherit_perms")
	inheritPerms, _ := inheritRaw.(bool)
	scopesRaw, _ := c.Get("scopes")
	scopes, _ := scopesRaw.([]string)
	return SelectScope(scopes, inheritPerms && exists, action)
}

// getIdentifier 获取客户端唯一标识符（私有函数）
// 优先级：API Key ID > 用户 ID > 客户端 IP
func getIdentifier(c *gin.Context) string {
	// 优先使用 API Key ID
	if apiKeyID, exists := c.Get("api_key_id"); exists {
		if id, ok := apiKeyID.(string); ok && id != "" {
			return "apikey:" + id
		}
	}

	// 回退到用户 ID
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok && id != "" {
			return "user:" + id
		}
	}

	// 最后回退到客户端 IP
	return "ip:" + c.ClientIP()
}
