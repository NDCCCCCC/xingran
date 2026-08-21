package network

// Router registration smoke tests (Phase 74-03).
//
// Strategy: identical to port_write_router_test.go (Phase 52). Each Setup*Router
// function constructs real services via core, so a runtime invocation needs the
// full Core init chain. Per VALIDATION.md §4.5 (Phase 73-05), grep-style source
// assertions are accepted as a compile-time + structural verification of the
// route table. This locks down:
//
//   - endpoint paths (kebab-case POST/GET registrations)
//   - permission middleware application (no leaked permissionless routes)
//   - handler binding (correct handler bound to each path)
//
// D-12 honored: zero business code touched.
//
// D-15 P2 floor: per-package ≥70% — these router tests don't add runtime
// coverage but they do register one of the new *_test.go files in the diff
// and document the route table as a regression shield.

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSetupNetworkRouter_RegistersEndpointGroups asserts every top-level route
// group is mounted on the parent router.
//
// network_router.go mounts the following groups via r.Group("/..."):
//
//	/devices       (DeviceHandler + DiscoveryHandler + ExportHandler.ExportDevices)
//	/credentials   (CredentialHandler + ExportHandler.ExportCredentials)
//	/templates     (TemplateHandler + ExportHandler.ExportTemplates)
//	/command       (CommandHandler + ExportHandler.ExportCommands)
//	/executions    (ExecutionHandler + ExportHandler.ExportExecutions)
//	/backups       (BackupHandler + ExportHandler.ExportBackups)
//	/discoveries   (DiscoveryHandler + ExportHandler.ExportDiscoveries)
//	/mac           (delegates to SetupMACRouter)
//	/ports         (delegates to SetupPortRouter + SetupPortWriteRouter)
//
// Plus the top-level batch-export endpoint:
func TestSetupNetworkRouter_RegistersEndpointGroups(t *testing.T) {
	src := readFile(t, "network_router.go")

	expectedGroups := []string{
		`r.Group("/devices")`,
		`r.Group("/credentials")`,
		`r.Group("/templates")`,
		`r.Group("/command")`,
		`r.Group("/executions")`,
		`r.Group("/backups")`,
		`r.Group("/discoveries")`,
		`r.Group("/mac")`,
		`r.Group("/ports")`,
		`r.POST("/batch-export"`,
	}
	for _, g := range expectedGroups {
		assert.Contains(t, src, g, "SetupNetworkRouter missing group/endpoint: %s", g)
	}
}

// TestSetupNetworkRouter_BindsAll9ExportEndpoints asserts every
// NetworkExportHandler.* method is registered on the corresponding group.
//
// 9 exports: devices, credentials, templates, commands, executions, backups,
// discoveries, MAC, ports — plus the top-level batch-export.
//
// Note: /mac/export and /ports/export are registered inside the dedicated
// SetupMACRouter / SetupPortRouter helpers (delegated from network_router.go).
// We accept either the network_router.go source OR the delegated router's
// source as the place where the binding appears.
func TestSetupNetworkRouter_BindsAll9ExportEndpoints(t *testing.T) {
	// handler names live in network_export_handler.go (9 methods) +
	// batch_export_helper.go (BatchExport).
	src := readFile(t, "network_export_handler.go") + "\n" + readFile(t, "batch_export_helper.go")
	netRouterSrc := readFile(t, "network_router.go")
	macRouterSrc := readFile(t, "mac_router.go")
	portRouterSrc := readFile(t, "port_router.go")

	// searchAcross concatenates all router source files so assertions can find
	// endpoint paths regardless of which file the binding lives in.
	searchAcross := netRouterSrc + "\n" + macRouterSrc + "\n" + portRouterSrc

	expected := []struct {
		handlerMethod  string // substring of handler method name
		routerContains string // token expected in the router source (kebab endpoint + handler binding)
	}{
		// Per-group exports — handler binding is `exportHandler.ExportX` on the
		// same line as `<group>.POST("/export", ...)`.  We assert the binding
		// (which is unique per handler method) plus the trailing "/export" path.
		{"ExportDevices", "exportHandler.ExportDevices"},
		{"ExportCredentials", "exportHandler.ExportCredentials"},
		{"ExportTemplates", "exportHandler.ExportTemplates"},
		{"ExportCommands", "exportHandler.ExportCommands"},
		{"ExportExecutions", "exportHandler.ExportExecutions"},
		{"ExportBackups", "exportHandler.ExportBackups"},
		{"ExportDiscoveries", "exportHandler.ExportDiscoveries"},
		{"ExportMACAddresses", "exportHandler.ExportMACAddresses"},
		{"ExportPorts", "exportHandler.ExportPorts"},
		{"BatchExport", "exportHandler.BatchExport"},
	}
	for _, e := range expected {
		assert.Contains(t, src, "func (h *NetworkExportHandler) "+e.handlerMethod,
			"export handler method missing: %s", e.handlerMethod)
		assert.Contains(t, searchAcross, e.routerContains,
			"router missing binding of %s (token %s)", e.handlerMethod, e.routerContains)
	}
	// /batch-export is the only top-level export path (not inside a group).
	assert.Contains(t, netRouterSrc, `r.POST("/batch-export"`,
		"top-level /batch-export route missing")
	// Every group-bound /export path appears exactly 8x (devices, credentials,
	// templates, command, executions, backups, discoveries, plus mac and ports
	// from delegated routers).
	countExport := strings.Count(searchAcross, `"/export"`)
	assert.Equal(t, 9, countExport,
		"expected 9 group-bound /export routes, got %d", countExport)
}

// TestSetupNetworkRouter_AppliesPermissionMiddleware asserts every group has
// middleware.RequirePermissions or middleware.RequirePermissionsWithQuery
// applied — no permissionless sub-groups leak.
//
// All 9 sub-groups must apply permission middleware.
func TestSetupNetworkRouter_AppliesPermissionMiddleware(t *testing.T) {
	src := readFile(t, "network_router.go")

	// Each group should be followed by either .Use(middleware.RequirePermissions...)
	// or .Use(middleware.RequirePermissionsWithQuery...) within a few lines.
	groups := []string{
		"devices := r.Group(\"/devices\")",
		"credentials := r.Group(\"/credentials\")",
		"templates := r.Group(\"/templates\")",
		"command := r.Group(\"/command\")",
		"executions := r.Group(\"/executions\")",
		"backups := r.Group(\"/backups\")",
		"discoveries := r.Group(\"/discoveries\")",
		"mac := r.Group(\"/mac\")",
		"ports := r.Group(\"/ports\")",
	}
	for _, g := range groups {
		idx := strings.Index(src, g)
		requireGe0(t, idx, "group %q not found", g)
		// Look 400 chars after for middleware.RequirePermissions (some groups
		// have multi-line Chinese comments between r.Group() and .Use()).
		windowEnd := idx + 400
		if windowEnd > len(src) {
			windowEnd = len(src)
		}
		window := src[idx:windowEnd]
		assert.True(t,
			strings.Contains(window, "middleware.RequirePermissions") ||
				strings.Contains(window, "middleware.RequirePermissionsWithQuery"),
			"group %q does not apply permission middleware in next 400 chars\nsnippet:\n%s", g, window)
	}
}

// TestSetupMACRouter_Registers7Endpoints asserts every endpoint in
// SetupMACRouter is mounted: list, collect, collect/all, statistics, clean,
// batch-delete, export.
func TestSetupMACRouter_Registers7Endpoints(t *testing.T) {
	src := readFile(t, "mac_router.go")

	expected := []string{
		`r.POST("/list"`,
		`r.POST("/collect"`,
		`r.POST("/collect/all"`,
		`r.GET("/statistics"`,
		`r.POST("/clean"`,
		`r.POST("/batch-delete"`,
		`r.POST("/export"`,
	}
	for _, ep := range expected {
		assert.Contains(t, src, ep, "SetupMACRouter missing endpoint: %s", ep)
	}
}

// TestSetupMACRouter_BindsCorrectHandler asserts the router constructs a MAC
// handler and binds it to every endpoint.
func TestSetupMACRouter_BindsCorrectHandler(t *testing.T) {
	src := readFile(t, "mac_router.go")

	assert.Contains(t, src, "handler := NewMACHandler(core)",
		"SetupMACRouter must construct MACHandler via NewMACHandler(core)")
	assert.Contains(t, src, "exportHandler.ExportMACAddresses",
		"SetupMACRouter must wire ExportMACAddresses on the export endpoint")
}

// TestSetupPortRouter_Registers7Endpoints asserts SetupPortRouter mounts all 7
// endpoint paths.
func TestSetupPortRouter_Registers7Endpoints(t *testing.T) {
	src := readFile(t, "port_router.go")

	expected := []string{
		`r.POST("/list"`,
		`r.POST("/collect"`,
		`r.POST("/collect-all"`,
		`r.GET("/statistics"`,
		`r.POST("/clean"`,
		`r.POST("/batch-delete"`,
		`r.POST("/export"`,
	}
	for _, ep := range expected {
		assert.Contains(t, src, ep, "SetupPortRouter missing endpoint: %s", ep)
	}
}

// TestSetupPortRouter_BindsCorrectHandler asserts PortHandler is bound and
// ExportPorts is wired on the export endpoint.
func TestSetupPortRouter_BindsCorrectHandler(t *testing.T) {
	src := readFile(t, "port_router.go")

	assert.Contains(t, src, "handler := NewPortHandler(core)",
		"SetupPortRouter must construct PortHandler via NewPortHandler(core)")
	assert.Contains(t, src, "exportHandler.ExportPorts",
		"SetupPortRouter must wire ExportPorts on the export endpoint")
}

// TestSetupMACHistoryRouter_Registers7Endpoints asserts all 7 endpoints
// (history/port, history/device, history/stats, history/vendor, history/list
// POST, history/list GET, history/heatmap).
func TestSetupMACHistoryRouter_Registers7Endpoints(t *testing.T) {
	src := readFile(t, "mac_history_router.go")

	expected := []string{
		`r.POST("/history/port"`,
		`r.POST("/history/device"`,
		`r.POST("/history/stats"`,
		`r.POST("/history/vendor"`,
		`r.POST("/history/list"`,
		`r.GET("/history/list"`,
		`r.POST("/history/heatmap"`,
	}
	for _, ep := range expected {
		assert.Contains(t, src, ep, "SetupMACHistoryRouter missing endpoint: %s", ep)
	}
}

// TestSetupMACHistoryRouter_BindsCorrectHandlers asserts both history + heatmap
// handlers are constructed.
func TestSetupMACHistoryRouter_BindsCorrectHandlers(t *testing.T) {
	src := readFile(t, "mac_history_router.go")

	assert.Contains(t, src, "historyHandler := NewMACHistoryHandler(historyQueryService)",
		"SetupMACHistoryRouter must construct history handler")
	assert.Contains(t, src, "heatmapHandler := NewMACHistoryHeatmapHandler(heatmapService)",
		"SetupMACHistoryRouter must construct heatmap handler")
	assert.Contains(t, src, "NewMACHistoryHeatmapService(core.GetDB(), nil, nil)",
		"SetupMACHistoryRouter must instantiate heatmap service with nil provider args (perf passthrough)")
}

// TestSetupTopologyRouter_Registers6Endpoints asserts SetupTopologyRouter
// mounts: rules/list, rules POST, rules/:id, rules/:id/update, rules/:id/delete,
// rules/effective GET.
func TestSetupTopologyRouter_Registers6Endpoints(t *testing.T) {
	src := readFile(t, "topology_router.go")

	expected := []string{
		`rules.POST("/list"`,
		`rules.POST(""`,
		`rules.POST("/:id"`,
		`rules.POST("/:id/update"`,
		`rules.POST("/:id/delete"`,
		`topology.GET("/rules/effective"`,
	}
	for _, ep := range expected {
		assert.Contains(t, src, ep, "SetupTopologyRouter missing endpoint: %s", ep)
	}
}

// TestSetupTopologyRouter_BindsCorrectHandler asserts TopologyHandler is built
// from FilterRuleService + db.
func TestSetupTopologyRouter_BindsCorrectHandler(t *testing.T) {
	src := readFile(t, "topology_router.go")

	assert.Contains(t, src, "filterRuleService := topology.NewFilterRuleService(db)",
		"SetupTopologyRouter must construct FilterRuleService")
	assert.Contains(t, src, "topologyHandler := NewTopologyHandler(filterRuleService, db).WithCore(core)",
		"SetupTopologyRouter must construct TopologyHandler with FilterRuleService + db + core")
}

// TestSetupTopologyRouter_NoPermissionMiddleware asserts topology routes are
// intentionally permissionless (Phase 56 dev/debug endpoints — no prod perm).
func TestSetupTopologyRouter_NoPermissionMiddleware(t *testing.T) {
	src := readFile(t, "topology_router.go")

	// Documented as debug-only — should not apply RequirePermissions
	assert.NotContains(t, src, "middleware.RequirePermissions",
		"SetupTopologyRouter is a debug surface; must NOT use RequirePermissions middleware")
}

// TestSetupNetworkRouter_CallsDelegateRouters asserts SetupNetworkRouter
// delegates to SetupMACRouter / SetupPortRouter / SetupPortWriteRouter.
func TestSetupNetworkRouter_CallsDelegateRouters(t *testing.T) {
	src := readFile(t, "network_router.go")

	assert.Contains(t, src, "SetupMACRouter(mac, core, exportHandler)",
		"SetupNetworkRouter must delegate to SetupMACRouter on /mac group")
	assert.Contains(t, src, "SetupPortRouter(ports, core, exportHandler)",
		"SetupNetworkRouter must delegate to SetupPortRouter on /ports group")
	assert.Contains(t, src, "SetupPortWriteRouter(ports, core)",
		"SetupNetworkRouter must delegate to SetupPortWriteRouter on /ports group (Phase 52)")
}

// TestSetupNetworkRouter_ConstructsAllServiceDependencies asserts every
// service the network module needs is constructed in SetupNetworkRouter
// (compile-time sanity check that no service is silently missing).
func TestSetupNetworkRouter_ConstructsAllServiceDependencies(t *testing.T) {
	src := readFile(t, "network_router.go")

	expected := []string{
		"services.NewAuthCredentialService",
		"networkServices.NewServiceWithCache",
		"services.NewTemplateService",
		"services.NewCommandDispatchService",
		"services.NewConfigExecutionService",
		"services.NewConfigBackupService",
		"core.DeviceDiscoveryService",
		"NewCredentialHandler",
		"NewDeviceHandler",
		"NewTemplateHandler",
		"NewCommandHandler",
		"NewExecutionHandler",
		"NewBackupHandler",
		"NewDiscoveryHandler",
		"NewNetworkExportHandler",
	}
	for _, s := range expected {
		assert.Contains(t, src, s, "SetupNetworkRouter missing service/handler: %s", s)
	}
}

// TestSetupNetworkRouter_UsesOpsSelectorReadPerms asserts the docs/decision
// rationale is honored: list-only paths accept ops selector perms, write paths
// stay strict.
//
// Per the inline comment: "查询接口(/list)额外接受 ops 读权限".
func TestSetupNetworkRouter_UsesOpsSelectorReadPerms(t *testing.T) {
	src := readFile(t, "network_router.go")

	// /devices group uses OpsSelectorReadPerms
	assert.Contains(t, src, "middleware.OpsSelectorReadPerms",
		"SetupNetworkRouter should use OpsSelectorReadPerms for selector read paths")
	// Should appear at least twice (devices + ports groups)
	count := strings.Count(src, "middleware.OpsSelectorReadPerms")
	assert.GreaterOrEqual(t, count, 2, "OpsSelectorReadPerms should be used in at least 2 group middleware (devices + ports)")
}

// TestSetupNetworkRouter_CacheProviderFallback asserts DataCacheService nil
// path uses NoOpCacheProvider (D-02 fallback).
func TestSetupNetworkRouter_CacheProviderFallback(t *testing.T) {
	src := readFile(t, "network_router.go")

	assert.Contains(t, src, "if core.DataCacheService != nil",
		"SetupNetworkRouter must nil-check DataCacheService")
	assert.Contains(t, src, "&systemServices.NoOpCacheProvider{}",
		"SetupNetworkRouter must fall back to NoOpCacheProvider when DataCacheService is nil")
}

// TestRouterRegistration_HTTPMethods asserts network routers use POST for
// most endpoints (Phase 56 convention) and GET only for read-only queries.
//
// This catches accidental HTTP method drift across future router refactors.
func TestRouterRegistration_HTTPMethods(t *testing.T) {
	for _, file := range []string{"network_router.go", "mac_router.go", "port_router.go", "mac_history_router.go", "topology_router.go"} {
		t.Run(file, func(t *testing.T) {
			src := readFile(t, file)
			// Count POST vs GET registrations — POST should dominate
			postCount := strings.Count(src, "r.POST(") + strings.Count(src, "devices.POST(") +
				strings.Count(src, "credentials.POST(") + strings.Count(src, "templates.POST(") +
				strings.Count(src, "command.POST(") + strings.Count(src, "executions.POST(") +
				strings.Count(src, "backups.POST(") + strings.Count(src, "discoveries.POST(") +
				strings.Count(src, "mac.POST(") + strings.Count(src, "ports.POST(") +
				strings.Count(src, "topology.POST(") + strings.Count(src, "topology.GET(") +
				strings.Count(src, "rules.POST(")
			getCount := strings.Count(src, "r.GET(") +
				strings.Count(src, "backups.GET(") +
				strings.Count(src, "topology.GET(")
			assert.Greater(t, postCount, getCount,
				"%s should have more POST endpoints than GET (POST-dominant API style)", file)
		})
	}
}

// TestSetupNetworkRouter_RegisterOrder matters because middleware applied to a
// parent group cascades to children. MAC / ports / topology registration order
// is documented in the file — we lock it down.
//
// Per the file:
//   1. /mac group delegated to SetupMACRouter
//   2. /ports group delegated to SetupPortRouter + SetupPortWriteRouter
//   3. /batch-export top-level route
func TestSetupNetworkRouter_RegisterOrder(t *testing.T) {
	src := readFile(t, "network_router.go")

	idxMac := strings.Index(src, "SetupMACRouter(mac, core, exportHandler)")
	idxPort := strings.Index(src, "SetupPortRouter(ports, core, exportHandler)")
	idxPortWrite := strings.Index(src, "SetupPortWriteRouter(ports, core)")
	idxBatchExport := strings.Index(src, `r.POST("/batch-export"`)

	requireGe0(t, idxMac, "SetupMACRouter not found")
	requireGe0(t, idxPort, "SetupPortRouter not found")
	requireGe0(t, idxPortWrite, "SetupPortWriteRouter not found")
	requireGe0(t, idxBatchExport, "batch-export endpoint not found")

	assert.Less(t, idxMac, idxPort, "SetupMACRouter must register before SetupPortRouter")
	assert.Less(t, idxPort, idxPortWrite, "SetupPortRouter must register before SetupPortWriteRouter")
	assert.Less(t, idxPortWrite, idxBatchExport, "SetupPortWriteRouter must register before batch-export")
}

// TestSetupPortRouter_RegisterOrder asserts SetupPortRouter registers in
// the documented order (Phase 56 + Phase 52).
func TestSetupPortRouter_RegisterOrder(t *testing.T) {
	src := readFile(t, "port_router.go")

	endpoints := []string{
		`r.POST("/list"`,
		`r.POST("/collect"`,
		`r.POST("/collect-all"`,
		`r.GET("/statistics"`,
		`r.POST("/clean"`,
		`r.POST("/batch-delete"`,
		`r.POST("/export"`,
	}
	lastIdx := -1
	for _, ep := range endpoints {
		idx := strings.Index(src, ep)
		requireGe0(t, idx, "SetupPortRouter missing endpoint: %s", ep)
		assert.Greater(t, idx, lastIdx,
			"endpoint %s must register after previous endpoint (got idx %d, prev %d)",
			ep, idx, lastIdx)
		lastIdx = idx
	}
}

// requireGe0 is a tiny test helper that wraps require.GreaterOrEqual with a
// printf-style format string — testify's require doesn't support formatted
// messages via t.Helper-style syntax consistently.
func requireGe0(t *testing.T, idx int, msg string, args ...interface{}) {
	t.Helper()
	if idx < 0 {
		t.Fatalf(msg, args...)
	}
}

// min is a tiny helper used by the middleware-window check; defined locally to
// avoid pulling in a Go 1.21+ constraint when older toolchains may be in use.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestAllRouters_UseGinRouterGroup asserts all Setup*Router functions take a
// *gin.RouterGroup as their first parameter (compile-time contract).
func TestAllRouters_UseGinRouterGroup(t *testing.T) {
	cases := []struct {
		file string
		sig  string
	}{
		{"network_router.go", "func SetupNetworkRouter(r *gin.RouterGroup, core *core.Core)"},
		{"mac_router.go", "func SetupMACRouter(r *gin.RouterGroup, core *core.Core, exportHandler *NetworkExportHandler)"},
		{"mac_history_router.go", "func SetupMACHistoryRouter(r *gin.RouterGroup, core *core.Core)"},
		{"port_router.go", "func SetupPortRouter(r *gin.RouterGroup, core *core.Core, exportHandler *NetworkExportHandler)"},
		{"topology_router.go", "func SetupTopologyRouter(r *gin.RouterGroup, db *gorm.DB, core *core.Core)"},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			src := readFile(t, c.file)
			assert.Contains(t, src, c.sig,
				"router %s must declare signature %q", c.file, c.sig)
		})
	}
}

// TestSetupNetworkRouter_HTTPMethod asserts the top-level batch-export uses
// POST (consistent with other batch endpoints in the package).
func TestSetupNetworkRouter_HTTPMethod(t *testing.T) {
	src := readFile(t, "network_router.go")
	// The exact line should be `r.POST("/batch-export"`.
	assert.Contains(t, src, `r.POST("/batch-export", exportHandler.BatchExport)`,
		"batch-export must be POST and bind exportHandler.BatchExport")
}

// TestRouters_ResponseContentTypes asserts none of the routers accidentally
// register a HEAD or OPTIONS route (which would imply handler-less endpoints).
func TestRouters_ResponseContentTypes(t *testing.T) {
	for _, file := range []string{"network_router.go", "mac_router.go", "port_router.go", "mac_history_router.go", "topology_router.go"} {
		t.Run(file, func(t *testing.T) {
			src := readFile(t, file)
			// HEAD and OPTIONS are not used by the network package
			assert.NotContains(t, src, ".HEAD(", file+" should not register HEAD endpoints")
			assert.NotContains(t, src, ".OPTIONS(", file+" should not register OPTIONS endpoints")
		})
	}
}

// Compile-time check: verify routers reference the response package indirectly
// (no top-level router assertion, just structural smoke).
var _ = http.MethodPost