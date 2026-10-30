package agent

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
)

// SetupAgentRouter 设置 Agent 路由
func SetupAgentRouter(r *gin.RouterGroup, core *core.Core) {
	handler := NewAgentHandler(core.GetDB()).WithCore(core)

	// Agent 注册 API（无需认证，供 Agent 自动注册使用）
	agentGroup := r.Group("/agent")
	{
		agentGroup.POST("/register", handler.RegisterAgent)
	}
}
