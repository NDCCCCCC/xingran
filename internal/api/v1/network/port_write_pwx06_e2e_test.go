package network

// PWX-06（v1.51 Phase 127）：端口写 handler 层 e2e —— gin test engine 走真实路由。
//
// 全链路：preview → write → audit/list，覆盖：
//   - RBAC：未认证 401 / superadmin 旁路（走真实 RequirePermissions + DB 判定）
//   - preview：假设备 404、VLAN 越界 400、成功 200 且零网络调用（executor nil 不触）
//   - write：端口不存在 404（不进入连接层，nil executor 安全）
//   - audit/list：seed 数据的分页 / status 过滤 / 空结果序列化
//
// 不 mock gin 层：SetupPortWriteRouter 原样挂载（含 RequirePermissions 组级中间件），
// 认证态通过 user_id 上下文 + sqlite 库内 admin 角色关系构造（走 isSuperAdmin 真实
// DB 判定路径，非测试旁路）。

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	core "github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

type pwx06Fixture struct {
	engine *gin.Engine
	db     *gorm.DB
}

// setupPWX06 组装真实路由 e2e 环境。
func setupPWX06(t *testing.T) *pwx06Fixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&models.NetworkDevice{}, &models.DevicePortStatus{},
		&models.PortWriteAudit{},
		&models.User{}, &models.Role{}, &models.UserRole{},
	))

	testCore := &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{DB: database, Type: "sqlite"}},
		CoreServices: &core.CoreServices{},
	}
	engine := gin.New()
	// 认证态注入（排在业务路由前）：模拟 auth 中间件写入的 user_id
	engine.Use(func(c *gin.Context) {
		if c.GetHeader("X-Test-User") != "" {
			c.Set("user_id", c.GetHeader("X-Test-User"))
		}
		c.Next()
	})
	v1 := engine.Group("/api/v1")
	ports := v1.Group("/network/ports")
	SetupPortWriteRouter(ports, testCore)

	return &pwx06Fixture{engine: engine, db: database}
}

// loginAsPWX06 构造 superadmin 认证链路：sys_user + admin 角色（走
// RequirePermissions → isSuperAdmin 的真实 DB 判定）。
func (f *pwx06Fixture) loginAsPWX06(t *testing.T, userID string) {
	t.Helper()
	u := models.User{BaseModel: models.BaseModel{ID: userID}, Username: userID, Status: models.UserStatusEnabled}
	require.NoError(t, f.db.Create(&u).Error)
	role := models.Role{BaseModel: models.BaseModel{ID: "role-admin-" + userID}, RoleKey: "admin", RoleName: "超管", Status: models.RoleStatusEnabled}
	require.NoError(t, f.db.Create(&role).Error)
	require.NoError(t, f.db.Create(&models.UserRole{UserID: userID, RoleID: role.ID}).Error)
}

func (f *pwx06Fixture) do(method, path, userID string, body interface{}) (int, string) {
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set("X-Test-User", userID)
	}
	w := httptest.NewRecorder()
	f.engine.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

// seedPWX06Audit 写入审计测试行。
func (f *pwx06Fixture) seedPWX06Audit(t *testing.T, id, deviceID, status, action string) {
	t.Helper()
	row := models.PortWriteAudit{
		ID:          id,
		DeviceID:    deviceID,
		PortID:      "port-" + id[len(id)-4:],
		Action:      action,
		BeforeValue: json.RawMessage(`{"adminStatus":"up"}`),
		AfterValue:  json.RawMessage(`{"adminStatus":"down"}`),
		CommandSent: "interface GE1/0/1\nshutdown",
		Status:      status,
		Operator:    "tester",
		CreatedAt:   time.Now(),
	}
	require.NoError(t, f.db.Create(&row).Error)
}

// TestPWX06_RBAC 未认证请求必须 401（真实 RequirePermissions 中间件路径）。
func TestPWX06_RBAC(t *testing.T) {
	f := setupPWX06(t)
	code, body := f.do("POST", "/api/v1/network/ports/write/preview", "", map[string]interface{}{})
	require.Equal(t, http.StatusUnauthorized, code, "未认证应 401, got %s", body)
}

// TestPWX06_PreviewContract preview 错误路径契约（404 / 400）与零网络调用。
func TestPWX06_PreviewContract(t *testing.T) {
	f := setupPWX06(t)
	f.loginAsPWX06(t, "u-admin")

	// 假设备 → 404
	code, body := f.do("POST", "/api/v1/network/ports/write/preview", "u-admin", map[string]interface{}{
		"deviceId": "de000000-0000-4000-8000-00000000dead",
		"action":   "shutdown",
		"portIds":  []string{"po000000-0000-4000-8000-00000000dead"},
	})
	// ⚠ 实际契约：HTTP 400 + 业务 code 404（response.Error 的 int 重载把
	// HTTPStatus 固定为 400，业务码落在 body.code）。前端按 body.code 分支。
	require.Equal(t, http.StatusBadRequest, code, "假设备应 HTTP 400, got %s", body)
	require.Contains(t, body, `"code":404`, "业务码应为 404")
	require.Contains(t, body, "设备不存在")
	for _, field := range []string{`"code"`, `"message"`, `"timestamp"`, `"request_id"`} {
		require.Contains(t, body, field, "响应契约缺 %s", field)
	}

	// VLAN 越界 → 400（先 seed 真设备让校验走到 vlan 检查）
	f.seedPWX06Device(t, "de-1", models.VendorHuawei)
	code, body = f.do("POST", "/api/v1/network/ports/write/preview", "u-admin", map[string]interface{}{
		"deviceId": "de-1", "action": "set_access_vlan", "portIds": []string{"p-1"}, "vlanId": 5000,
	})
	require.Equal(t, http.StatusBadRequest, code, "VLAN 越界应 400, got %s", body)

	// 成功路径：commands 渲染 + riskLevel，executor 为 nil 仍成功 = 零网络调用证明
	f.seedPWX06Port(t, "de-1", "p-1")
	code, body = f.do("POST", "/api/v1/network/ports/write/preview", "u-admin", map[string]interface{}{
		"deviceId": "de-1", "action": "shutdown", "portIds": []string{"p-1"},
	})
	require.Equal(t, http.StatusOK, code, "preview 成功路径应 200, got %s", body)
	require.Contains(t, body, `"dryRun":true`)
	require.Contains(t, body, `"riskLevel"`)
	require.Contains(t, body, "shutdown")
}

// TestPWX06_WriteAuditChain write 错误路径 + audit/list 全链路。
func TestPWX06_WriteAuditChain(t *testing.T) {
	f := setupPWX06(t)
	f.loginAsPWX06(t, "u-admin")
	f.seedPWX06Device(t, "de-1", models.VendorHuawei)

	// write：端口不存在 → 404（不进入连接层，nil executor 安全）
	code, body := f.do("POST", "/api/v1/network/ports/write/shutdown", "u-admin", map[string]interface{}{
		"deviceId": "de-1", "portId": "p-missing",
	})
		// 同上：HTTP 400 + 业务 code 404 契约
	require.Equal(t, http.StatusBadRequest, code, "write 端口不存在应 HTTP 400, got %s", body)
	require.Contains(t, body, `"code":404`)

	// audit/list：seed 3 行（2 succeeded + 1 failed）→ 分页 + status 过滤
	f.seedPWX06Audit(t, "a-0001", "de-1", "succeeded", "shutdown")
	f.seedPWX06Audit(t, "a-0002", "de-1", "succeeded", "description")
	f.seedPWX06Audit(t, "a-0003", "de-1", "failed", "shutdown")

	code, body = f.do("POST", "/api/v1/network/ports/write/audit/list", "u-admin", map[string]interface{}{
		"current": 1, "pageSize": 10,
	})
	require.Equal(t, http.StatusOK, code, "audit list 应 200, got %s", body)
	require.Contains(t, body, `"total":3`)
	require.Contains(t, body, "beforeValue", "审计行应带 beforeValue JSON（前端对比展示依赖）")

	// status 过滤
	code, body = f.do("POST", "/api/v1/network/ports/write/audit/list", "u-admin", map[string]interface{}{
		"current": 1, "pageSize": 10, "status": "failed",
	})
	require.Equal(t, http.StatusOK, code)
	require.Contains(t, body, `"total":1`)
	require.Contains(t, body, `"failed"`)

	// 空结果 → list 序列化为 []（非 null）
	code, body = f.do("POST", "/api/v1/network/ports/write/audit/list", "u-admin", map[string]interface{}{
		"current": 1, "pageSize": 10, "status": "rolled_back",
	})
	require.Equal(t, http.StatusOK, code)
	require.Contains(t, body, `"total":0`)
	require.Contains(t, body, `"list":[]`, "空结果必须序列化为 []")
}

// seedPWX06Device / seedPWX06Port 本地 seed（跨包不可复用 portwrite 包 helper）。
func (f *pwx06Fixture) seedPWX06Device(t *testing.T, id string, vendor models.DeviceVendor) {
	t.Helper()
	dev := models.NetworkDevice{
		DeviceName: "dev-" + id, DeviceType: models.DeviceTypeSwitch,
		Vendor: vendor, IPAddress: "10.0.0.9", Status: models.DeviceStatusOnline, Model: "m",
	}
	dev.ID = id
	require.NoError(t, f.db.Create(&dev).Error)
}

func (f *pwx06Fixture) seedPWX06Port(t *testing.T, deviceID, portID string) {
	t.Helper()
	port := models.DevicePortStatus{
		ID: portID, DeviceID: deviceID, InterfaceName: "GE1/0/1",
		AdminStatus: "up", Dot1xEnabled: false, CollectedAt: time.Now(),
	}
	require.NoError(t, f.db.Create(&port).Error)
}
