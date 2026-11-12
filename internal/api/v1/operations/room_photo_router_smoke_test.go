package operations

// CI 覆盖率门禁（PR #87 coverage-diff）注册冒烟测试：
// task-4 权限接线新增的 SetupRoomPhotoRouter（机房照片子资源，借
// ops:serverroom 域码）此前无运行时触达。

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	coredb "github.com/xingran-next/xingran-go-backend/internal/core/db"
)

// TestSetupRoomPhotoRouter_RegistersPermissionWiredRoutes 机房照片 8 端点注册。
func TestSetupRoomPhotoRouter_RegistersPermissionWiredRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	c := &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &coredb.Database{DB: gdb}},
		CoreServices: &core.CoreServices{},
	}

	engine := gin.New()
	rg := engine.Group("/ops/serverrooms/photos")

	require.NotPanics(t, func() {
		SetupRoomPhotoRouter(rg, c)
	})

	set := map[string]bool{}
	for _, r := range engine.Routes() {
		set[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		http.MethodPost + " /ops/serverrooms/photos/upload",
		http.MethodGet + " /ops/serverrooms/photos/list/:roomId",
		http.MethodGet + " /ops/serverrooms/photos/:id",
		http.MethodPut + " /ops/serverrooms/photos/:id/primary",
		http.MethodPut + " /ops/serverrooms/photos/:id/description",
		http.MethodPut + " /ops/serverrooms/photos/sort",
		http.MethodDelete + " /ops/serverrooms/photos/:id",
		http.MethodPost + " /ops/serverrooms/photos/batch-delete",
	} {
		assert.True(t, set[want], "route not registered: %s", want)
	}
}
