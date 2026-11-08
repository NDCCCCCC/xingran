package system

// Phase 144 · 漂移告警路由注册（权限三方一致 + 严格写组兄弟分离）。
//
// 读组：RequirePermissions（ANY-of 读码）→ status / logs / config
// 写组：RequireStrictPermissions（单一权限点、跳过 F 型继承与模块降级）
//       → config/update（edit）/ trigger（trigger）
// 写组是读组的**兄弟组**（不是子组），避免 gin Group 继承读组 ANY-of
// 导致只持 list 权限的只读角色被放行（v1.24 Phase 78 #16 教训）。

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
)

// SetupAdDriftAlertRouter 注册 5 个 POST 端点。
func SetupAdDriftAlertRouter(r *gin.RouterGroup, core *core.Core) {
	handler := NewAdDriftAlertHandler(core)

	// 读组：status / logs / config
	read := r.Group("/ad-drift-alert")
	read.Use(middleware.RequirePermissions([]string{
		string(permission.ADDriftAlertList),
		string(permission.ADDriftAlertConfig),
	}, core))
	{
		read.POST("/status", handler.GetStatus)
		read.POST("/logs", handler.ListLogs)
		read.POST("/config", handler.GetConfig)
	}

	// 写组 1：config/update（严格单一 edit 权限点，与读组兄弟分离）
	writeConfig := r.Group("/ad-drift-alert")
	writeConfig.Use(middleware.RequireStrictPermissions([]string{
		string(permission.ADDriftAlertEdit),
	}, core))
	{
		writeConfig.POST("/config/update", handler.UpdateConfig)
	}

	// 写组 2：trigger（严格单一 trigger 权限点）
	writeTrigger := r.Group("/ad-drift-alert")
	writeTrigger.Use(middleware.RequireStrictPermissions([]string{
		string(permission.ADDriftAlertTrigger),
	}, core))
	{
		writeTrigger.POST("/trigger", handler.Trigger)
	}
}
