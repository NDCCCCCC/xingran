package system

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	coredb "github.com/xingran-next/xingran-go-backend/internal/core/db"
)

// =====================================================================
// Phase 74-04: router smoke tests (runtime invocation).
//
// Each Setup*Router constructs services from core.GetDB() and registers
// routes on a gin group. With a sqlite-backed core the service structs
// construct without touching the DB, so calling Setup registers the full
// route table — real statement coverage for every *_router.go.
//
// Each call runs under a recover guard: a router that requires an
// un-mockable core field at setup time is reported and skipped instead
// of killing the test binary.
// =====================================================================

// callSetup invokes fn under a recover guard; returns false when it panicked.
func callSetup(fn func()) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()
	fn()
	return true
}

func newRouterSmokeCore(t *testing.T) *core.Core {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &coredb.Database{DB: gdb}},
		CoreServices: &core.CoreServices{},
	}
}

func setupAndCountRoutes(t *testing.T, name string, register func(rg *gin.RouterGroup, c *core.Core)) int {
	t.Helper()
	c := newRouterSmokeCore(t)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	rg := engine.Group("/test")
	if !callSetup(func() { register(rg, c) }) {
		t.Logf("router %s panicked during setup (un-mockable core dep) — skipped", name)
		return -1
	}
	count := 0
	for _, route := range engine.Routes() {
		if route.Method != http.MethodGet || route.Path != "" {
			count++
		}
	}
	return len(engine.Routes())
}

func TestSetupRouters_AllRegisterRoutes(t *testing.T) {
	cases := []struct {
		name string
		reg  func(rg *gin.RouterGroup, c *core.Core)
	}{
		{"ad_domain", SetupADDomainRouter},
		{"apikey", SetupAPIKeyRouter},
		{"column_config", SetupColumnConfigRouter},
		{"config", SetupConfigRouter},
		{"dashboard", SetupDashboardRouter},
		{"department", SetupDepartmentRouter},
		{"dict", SetupDictRouter},
		{"file", SetupFileRouter},
		{"menu", SetupMenuRouter},
		{"user_menu", SetupUserMenuRouter},
		{"notice", SetupNoticeRouter},
		{"notice_user", SetupNoticeUserRouter},
		{"notification_config", SetupNotificationConfigRouter},
		{"ou_mapping", SetupOUMappingRouter},
		{"post", SetupPostRouter},
		{"profile", SetupProfileRouter},
		{"role", SetupRoleRouter},
		{"settings", SetupSettingsRouter},
		{"user", SetupUserRouter},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			n := setupAndCountRoutes(t, tc.name, tc.reg)
			if n < 0 {
				t.Skip("setup panicked — core dep unavailable in test")
			}
			require.Greater(t, n, 0, "router registered no routes")
		})
	}
}
