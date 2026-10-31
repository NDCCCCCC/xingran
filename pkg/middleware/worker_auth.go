package middleware

// DEBT-03（v1.51）：RPA Worker API Token 认证中间件。
//
// 双 Token 模型（docs/RPA-Worker-API认证方案-待办.md）：
//   - ProvisioningAuth：注册接口专用，校验预共享白名单 X-Provisioning-Token；
//   - WorkerAuth：运行期接口（心跳/进度等），校验 X-Worker-Token。
//
// 向后兼容宽限期（双读）：authRequired=false（默认）时无 Token 请求放行并
// 记一次 WARN（行为与 DEBT-03 之前一致）；=true 时无/错/过期 Token 一律 401。

import (
	"github.com/gin-gonic/gin"

	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

const (
	// HeaderProvisioningToken 注册换发令牌请求头。
	HeaderProvisioningToken = "X-Provisioning-Token"
	// HeaderWorkerToken 运行期 API 令牌请求头。
	HeaderWorkerToken = "X-Worker-Token"
)

// ProvisioningAuth 注册接口的预共享令牌认证。
//
// required=false（宽限期）：白名单为空或请求无头时放行（WARN），带错 token 仍 401
// （显式提供了凭证却错误，比缺失更可疑）。
func ProvisioningAuth(svc rpa.WorkerTokenService, required bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader(HeaderProvisioningToken)
		if token == "" {
			if !required {
				applogger.Warnf("[RPA-WORKER-AUTH] 宽限期：注册请求未携带 %s，已放行（auth_required=false）", HeaderProvisioningToken)
				c.Next()
				return
			}
			response.Error(c, response.ErrUnauthorized, "缺少 Provisioning 认证令牌")
			c.Abort()
			return
		}
		if !svc.ValidateProvisioningToken(token) {
			response.Error(c, response.ErrUnauthorized, "无效的 Provisioning 认证令牌")
			c.Abort()
			return
		}
		c.Next()
	}
}

// WorkerAuth 运行期接口的 Worker Token 认证。
//
// required=false（宽限期）：无头放行（WARN）；带头但校验失败仍 401。
// 校验通过后设置 worker_id / worker_name 上下文；若路由含 :id 参数且与
// claims.WorkerID 不一致 → 403（防止持 A 凭证伪造 B 的心跳）。
func WorkerAuth(svc rpa.WorkerTokenService, required bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader(HeaderWorkerToken)
		if token == "" {
			if !required {
				applogger.Warnf("[RPA-WORKER-AUTH] 宽限期：请求未携带 %s，已放行（auth_required=false）path=%s", HeaderWorkerToken, c.FullPath())
				c.Next()
				return
			}
			response.Error(c, response.ErrUnauthorized, "缺少 Worker 认证令牌")
			c.Abort()
			return
		}
		claims, err := svc.ValidateWorkerToken(c.Request.Context(), token)
		if err != nil {
			response.Error(c, response.ErrUnauthorized, "无效或过期的 Worker 认证令牌")
			c.Abort()
			return
		}
		// 凭证归属校验：URL :id 与 claims 不一致 → 403
		if pathID := c.Param("id"); pathID != "" && pathID != claims.WorkerID {
			response.Error(c, response.ErrForbidden, "Worker 令牌与目标节点不匹配")
			c.Abort()
			return
		}
		c.Set("worker_id", claims.WorkerID)
		c.Set("worker_name", claims.WorkerName)
		c.Next()
	}
}
