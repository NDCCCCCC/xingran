package operations

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// newFakeCore wraps a sqlite DB in a minimal *core.Core for the handler
// (mirrors workstation_handler_test.go's newTestCoreForHandler pattern).
func newFakeCore(gormDB *gorm.DB) *core.Core {
	return &core.Core{
		CoreInfra: &core.CoreInfra{
			DB: &db.Database{DB: gormDB, Type: "sqlite"},
		},
	}
}

// setupComponentHandlerDB builds a minimal sqlite DB with ops_asset rows
// for the ListComponents endpoint.
func setupComponentHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "open sqlite")
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_asset (
			id TEXT PRIMARY KEY,
			devicesn TEXT,
			device_model_name TEXT,
			parent_asset_id TEXT,
			source_device_id TEXT,
			component_type TEXT,
			component_slot TEXT,
			deleted_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error, "create ops_asset")
	return db
}

// TestListComponentsHappyPath verifies GET /ops/asset/components returns
// the 3 component rows tied to parentAssetId and excludes the unrelated
// rows (main device + components under different parent).
func TestListComponentsHappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupComponentHandlerDB(t)
	now := time.Now().Format("2006-01-02 15:04:05")

	// parent-1 has 3 components: 1 card + 1 fan + 1 transceiver
	mustExec(t, db, `INSERT INTO ops_asset (id, devicesn, parent_asset_id, component_type, component_slot, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"asset-card-1", "CARD-SN-1", "parent-uuid-1", "card", "Slot 1", now)
	mustExec(t, db, `INSERT INTO ops_asset (id, devicesn, parent_asset_id, component_type, component_slot, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"asset-fan-1", "FAN-SN-1", "parent-uuid-1", "fan", "Fan 1", now)
	mustExec(t, db, `INSERT INTO ops_asset (id, devicesn, parent_asset_id, component_type, component_slot, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"asset-xp-1", "XP-SN-1", "parent-uuid-1", "transceiver", "10GE5/0/4", now)
	// parent-2 component (must NOT appear in parent-1 result)
	mustExec(t, db, `INSERT INTO ops_asset (id, devicesn, parent_asset_id, component_type, component_slot, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"asset-card-2", "CARD-SN-2", "parent-uuid-2", "card", "Slot 1", now)
	// main device row (no component_type) — must NOT appear
	mustExec(t, db, `INSERT INTO ops_asset (id, devicesn, component_type, created_at) VALUES (?, ?, NULL, ?)`,
		"asset-switch-1", "SWITCH-SN-1", now)

	// Build handler manually with a fake core wrapping our test DB.
	h := &AssetComponentHandler{core: newFakeCore(db)}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet,
		"/ops/asset/components?parentAssetId=11111111-1111-1111-1111-111111111111", nil)
	// We didn't insert parent-uuid-1 as a UUID (test uses string IDs
	// directly because sqlite doesn't enforce UUID type); set the query
	// to match the string we used in fixtures but in UUID form so the
	// UUIDPattern check passes.
	c.Request.URL.RawQuery = "parentAssetId=11111111-1111-1111-1111-111111111111"

	// Adjust the inserted parent_asset_id to be the valid UUID so the
	// query WHERE clause matches.
	mustExec(t, db, `UPDATE ops_asset SET parent_asset_id=? WHERE parent_asset_id=?`,
		"11111111-1111-1111-1111-111111111111", "parent-uuid-1")

	h.ListComponents(c)
	require.Equal(t, http.StatusOK, w.Code, "handler returns 200")

	var resp struct {
		Code int `json:"code"`
		Data struct {
			List  []map[string]interface{} `json:"list"`
			Total int                      `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code, "response code=0 success")
	require.Equal(t, 3, resp.Data.Total, "3 components under parent-uuid-1")
	require.Len(t, resp.Data.List, 3)
}

// TestListComponentsInvalidUUID verifies the UUID-format guard rejects
// malformed parentAssetId with 400.
func TestListComponentsInvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupComponentHandlerDB(t)
	h := &AssetComponentHandler{core: newFakeCore(db)}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet,
		"/ops/asset/components?parentAssetId=not-a-uuid", nil)

	h.ListComponents(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// TestListComponentsMissingParam verifies missing parentAssetId → 400.
func TestListComponentsMissingParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupComponentHandlerDB(t)
	h := &AssetComponentHandler{core: newFakeCore(db)}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ops/asset/components", nil)

	h.ListComponents(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// mustExec is a t.Helper-backed Exec that fails the test on error.
func mustExec(t *testing.T, db *gorm.DB, query string, args ...interface{}) {
	t.Helper()
	require.NoError(t, db.Exec(query, args...).Error, "exec: "+query)
}
