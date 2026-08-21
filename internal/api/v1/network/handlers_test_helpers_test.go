package network

// Shared test helpers for internal/api/v1/network handler tests (Phase 74-03).
//
// Two test strategies coexist in this package, mirroring how the handlers are wired:
//  1. Interface-backed handlers (DeviceHandler/TopologyHandler/MACHistoryHandler/
//     MACHistoryHeatmapHandler) use mock services with *Func fields (D-08 pattern,
//     identical to Phase 74-01 operations handlers).
//  2. Handlers that receive CONCRETE service structs (CredentialHandler/
//     TemplateHandler/CommandHandler/ExecutionHandler/BackupHandler/DiscoveryHandler)
//     or build services from core (MACHandler/PortHandler/NetworkExportHandler) are
//     exercised against a real glebarez sqlite in-memory DB (D-02), with the operlog
//     dependency stubbed via the package's existing mockOperLogService (D-03).
//
// Zero business code is modified (D-12).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
)

// netTestEnv bundles the sqlite-backed core used by DB-driven handler tests.
type netTestEnv struct {
	core *core.Core
	db   *gorm.DB
	oper *mockOperLogService // reused from port_write_handler_test.go (same package)
}

// newNetworkTestEnv opens an in-memory sqlite DB, optionally AutoMigrates the given
// models, and returns a core.Core whose OperLogService is the recording mock and
// whose DeviceDiscoveryService is a real (DB-backed) instance — several export
// endpoints dereference core.DeviceDiscoveryService directly, so it must be non-nil.
func newNetworkTestEnv(t *testing.T, migrate ...interface{}) *netTestEnv {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	if len(migrate) > 0 {
		require.NoError(t, gormDB.AutoMigrate(migrate...))
	}
	// sys_user is always migrated (even when caller passes no models): operlog.Record
	// resolves nickname/dept_name from sys_user on every write — without the table it
	// spews SQL errors into every test's log output.
	require.NoError(t, gormDB.AutoMigrate(&models.User{}))
	oper := &mockOperLogService{}
	c := &core.Core{
		CoreInfra: &core.CoreInfra{
			DB: &db.Database{DB: gormDB, Type: "sqlite"},
		},
		CoreServices: &core.CoreServices{
			OperLogService:         oper,
			DeviceDiscoveryService: services.NewDeviceDiscoveryService(gormDB),
		},
	}
	return &netTestEnv{core: c, db: gormDB, oper: oper}
}

// netMigrateAll migrates every model touched by the network module in one shot —
// used by tests that traverse several handlers (export/router smoke).
func netMigrateAll(t *testing.T, env *netTestEnv) {
	t.Helper()
	require.NoError(t, env.db.AutoMigrate(
		&models.NetworkDevice{},
		&models.AuthCredential{},
		&models.ConfigTemplate{},
		&models.ConfigExecution{},
		&models.ConfigExecutionDetail{},
		&models.ConfigBackup{},
		&models.DeviceDiscovery{},
		&models.DeviceMACAddress{},
		&models.DevicePortStatus{},
		&models.MACFilterRule{},
	))
}

// netStubCipher satisfies addomain.PasswordCipher for AuthCredentialService tests.
type netStubCipher struct{}

func (netStubCipher) Encrypt(plaintext string) (string, error) { return "enc:" + plaintext, nil }
func (netStubCipher) Decrypt(ciphertext string) (string, error) {
	return strings.TrimPrefix(ciphertext, "enc:"), nil
}

// netRoute describes one mounted route for netServe.
type netRoute struct {
	method  string
	path    string
	handler gin.HandlerFunc
}

// netServe mounts routes on a fresh engine (with a middleware that populates the
// auth context keys handlers blindly type-assert on: user_id / username) and
// performs the request. Path may contain :params — they are resolved by gin.
func netServe(t *testing.T, routes []netRoute, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user-0001")
		c.Set("username", "tester")
		c.Next()
	})
	for _, rt := range routes {
		switch rt.method {
		case http.MethodGet:
			r.GET(rt.path, rt.handler)
		default:
			r.POST(rt.path, rt.handler)
		}
	}
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// netPost is a convenience wrapper for the common single-route POST case.
func netPost(t *testing.T, path string, handler gin.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	return netServe(t, []netRoute{{method: http.MethodPost, path: path, handler: handler}}, http.MethodPost, path, body)
}

// netGet is a convenience wrapper for the common single-route GET case.
func netGet(t *testing.T, path string, handler gin.HandlerFunc, query string) *httptest.ResponseRecorder {
	t.Helper()
	return netServe(t, []netRoute{{method: http.MethodGet, path: path, handler: handler}}, http.MethodGet, path+query, "")
}

// netResp is the decoded response envelope used by every assertion below.
type netResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// decodeNetResp decodes the recorder body as the unified response envelope.
func decodeNetResp(t *testing.T, w *httptest.ResponseRecorder) netResp {
	t.Helper()
	var resp netResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "body=%s", w.Body.String())
	return resp
}

// netSeedDevice inserts a minimal NetworkDevice row and returns it.
func netSeedDevice(t *testing.T, gormDB *gorm.DB, id, name, ip string) *models.NetworkDevice {
	t.Helper()
	d := &models.NetworkDevice{
		BaseModel:  models.BaseModel{ID: id},
		DeviceName: name,
		DeviceType: models.DeviceType("switch"),
		Vendor:     models.DeviceVendor("huawei"),
		IPAddress:  ip,
		Port:       22,
		SNMPPort:   161,
		Status:     models.DeviceStatus(0),
	}
	require.NoError(t, gormDB.Create(d).Error)
	return d
}
