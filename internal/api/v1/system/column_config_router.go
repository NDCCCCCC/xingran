package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

func SetupColumnConfigRouter(r *gin.RouterGroup, core *core.Core) {
	columnConfigService := systemServices.NewColumnConfigService(core.GetDB())
	columnConfigHandler := NewColumnConfigHandler(columnConfigService).WithCore(core)

	r.GET("/:page_key", columnConfigHandler.GetByPageKey)
	r.POST("", columnConfigHandler.Save)
	r.DELETE("/:page_key", columnConfigHandler.Reset)
}
