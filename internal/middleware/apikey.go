package middleware

import (
	"encoding/json"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

const (
	apiKeyHeader = "X-API-Key"
	keyPrefix    = "rec_"
)

// MultiAuth 多重认证中间件（JWT + API Key）
// 如果提供 X-API-Key 请求头，使用 API Key 认证
// 否则跳过，允许 JWT 认证中间件处理
func MultiAuth(apiKeyService system.APIKeyService, usageLogger services.UsageLogger) gin.HandlerFunc {
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

		// 验证 IP 白名单
		if apiKey.IPWhitelist != "" && apiKey.IPWhitelist != "[]" {
			var whitelist []string
			if err := json.Unmarshal([]byte(apiKey.IPWhitelist), &whitelist); err == nil {
				clientIP := c.ClientIP()
				if !isIPAllowed(clientIP, whitelist) {
					response.Error(c, response.ErrForbidden, "客户端IP不在白名单中")
					c.Abort()
					return
				}
			}
		}

		// 解析作用域
		var scopes []string
		if apiKey.Scopes != "" && apiKey.Scopes != "[]" {
			if err := json.Unmarshal([]byte(apiKey.Scopes), &scopes); err != nil {
				scopes = []string{}
			}
		}

		// 设置用户上下文
		setUserContextForAPIKey(c, apiKey, scopes)

		// 异步记录使用日志
		go func() {
			usageLogger.LogUsage(c.Request.Context(), &services.LogUsageRequest{
				APIKeyID: apiKey.ID,
				UserID:   *apiKey.UserID,
				Method:   c.Request.Method,
				Path:     c.Request.URL.Path,
				ClientIP: c.ClientIP(),
			})
		}()

		c.Next()
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
// apiKey 参数的类型是 *models.APIKey，使用 interface{} 避免循环导入
func setUserContextForAPIKey(c *gin.Context, apiKey interface{}, scopes []string) {
	// 通过类型断言访问 apiKey 的字段
	// models.APIKey 结构: ID, Name, UserID, User, InheritPerms
	type apiKeyType struct {
		ID           string
		Name         string
		UserID       *string
		InheritPerms bool
		User         *interface{}
	}

	if ak, ok := apiKey.(apiKeyType); ok {
		userID := ""
		username := ak.Name

		if ak.UserID != nil {
			userID = *ak.UserID
		}

		// 设置上下文
		c.Set("user_id", userID)
		c.Set("username", username)
		c.Set("api_key_id", ak.ID)
		c.Set("scopes", scopes)
		c.Set("auth_type", "api_key")

		// 如果继承权限且有关联用户，加载用户角色
		if ak.InheritPerms && ak.User != nil {
			// User 角色会在需要时从数据库加载
			c.Set("inherit_perms", true)
		}
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
func RequireAPIKeyResourcePermission(resource string, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 确定所需作用域
		requiredScope := getRequiredScope(action)

		// 调用作用域验证
		RequireScope(requiredScope)(c)
	}
}

// getRequiredScope 根据操作获取所需作用域（私有函数）
// 映射规则：view/create/edit/delete -> read/write
func getRequiredScope(action string) string {
	// 操作到作用域的映射
	scopeMap := map[string]string{
		"view":   "read",
		"create": "write",
		"edit":   "write",
		"delete": "write",
	}

	// 如果操作不在映射中，默认为 read
	if scope, ok := scopeMap[action]; ok {
		return scope
	}
	return "read"
}
