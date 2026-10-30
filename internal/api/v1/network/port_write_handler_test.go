package network

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/xingran-next/xingran-go-backend/internal/services/portwrite"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ===== Mock service =====

// mockPortWriteService 实现 portwrite.PortWriteService，记录调用并返回预设结果。
type mockPortWriteService struct {
	mock.Mock

	shutdownResult     *portwrite.PortResult
	shutdownErr        error
	undoShutdownResult *portwrite.PortResult
	undoShutdownErr    error
	setDescriptionOut  *portwrite.PortResult
	setDescriptionErr  error
	enableDot1xOut     *portwrite.PortResult
	enableDot1xErr     error
	disableDot1xOut    *portwrite.PortResult
	disableDot1xErr    error
	batchResult        *portwrite.BatchResult
	batchErr           error
	// v1.20.1 set_access_vlan + port_binding mock outputs (Phase 56 W3)
	setAccessVlanOut *portwrite.PortResult
	setAccessVlanErr error
	portBindingOut   *portwrite.PortResult
	portBindingErr   error

	shutdownCalls     int
	undoCalls         int
	setDescCalls      int
	enableCalls       int
	disableCalls      int
	batchCalls        int
	lastShutdownInput struct {
		portID, operator string
	}
	lastBatchInput struct {
		req      portwrite.BatchWriteRequest
		operator string
	}
}

func (m *mockPortWriteService) Shutdown(ctx context.Context, portID, operator string) (*portwrite.PortResult, error) {
	m.shutdownCalls++
	m.lastShutdownInput.portID = portID
	m.lastShutdownInput.operator = operator
	if m.shutdownResult != nil || m.shutdownErr != nil {
		return m.shutdownResult, m.shutdownErr
	}
	return &portwrite.PortResult{PortID: portID, Action: portcollection.ActionShutdown, Status: "succeeded", CommandSent: "shutdown"}, nil
}
func (m *mockPortWriteService) UndoShutdown(ctx context.Context, portID, operator string) (*portwrite.PortResult, error) {
	m.undoCalls++
	if m.undoShutdownResult != nil || m.undoShutdownErr != nil {
		return m.undoShutdownResult, m.undoShutdownErr
	}
	return &portwrite.PortResult{PortID: portID, Action: portcollection.ActionUndoShutdown, Status: "succeeded"}, nil
}
func (m *mockPortWriteService) SetDescription(ctx context.Context, portID, desc, operator string) (*portwrite.PortResult, error) {
	m.setDescCalls++
	if m.setDescriptionOut != nil || m.setDescriptionErr != nil {
		return m.setDescriptionOut, m.setDescriptionErr
	}
	return &portwrite.PortResult{PortID: portID, Action: portcollection.ActionDescription, Status: "succeeded", CurrentState: desc}, nil
}
func (m *mockPortWriteService) EnableDot1x(ctx context.Context, portID, operator string) (*portwrite.PortResult, error) {
	m.enableCalls++
	if m.enableDot1xOut != nil || m.enableDot1xErr != nil {
		return m.enableDot1xOut, m.enableDot1xErr
	}
	return &portwrite.PortResult{PortID: portID, Action: portcollection.ActionDot1xEnable, Status: "succeeded"}, nil
}
func (m *mockPortWriteService) DisableDot1x(ctx context.Context, portID, operator string) (*portwrite.PortResult, error) {
	m.disableCalls++
	if m.disableDot1xOut != nil || m.disableDot1xErr != nil {
		return m.disableDot1xOut, m.disableDot1xErr
	}
	return &portwrite.PortResult{PortID: portID, Action: portcollection.ActionDot1xDisable, Status: "succeeded"}, nil
}
func (m *mockPortWriteService) BatchWritePorts(ctx context.Context, req portwrite.BatchWriteRequest, operator string) (*portwrite.BatchResult, error) {
	m.batchCalls++
	m.lastBatchInput.req = req
	m.lastBatchInput.operator = operator
	if m.batchResult != nil || m.batchErr != nil {
		return m.batchResult, m.batchErr
	}
	return &portwrite.BatchResult{Succeeded: []portwrite.PortResult{}, Failed: []portwrite.PortResult{}, Skipped: []portwrite.PortResult{}}, nil
}

// v1.20.1 新增:set_access_vlan + port_binding 2 个 mock 方法 (Phase 56 W2)。
//
// W3 扩展：增加可选 result/err 字段，便于 handler 集成测试覆盖
// 成功 / sentinel / failed 三条路径（不设置时返 zero-value succeeded PortResult）。
func (m *mockPortWriteService) SetAccessVlan(ctx context.Context, portID string, vlanId int, operator string) (*portwrite.PortResult, error) {
	if m.setAccessVlanOut != nil || m.setAccessVlanErr != nil {
		return m.setAccessVlanOut, m.setAccessVlanErr
	}
	return &portwrite.PortResult{PortID: portID, Action: portcollection.ActionSetAccessVLAN, Status: "succeeded"}, nil
}
func (m *mockPortWriteService) PortBinding(ctx context.Context, portID string, op string, ipAddress string, macAddress string, operator string) (*portwrite.PortResult, error) {
	if m.portBindingOut != nil || m.portBindingErr != nil {
		return m.portBindingOut, m.portBindingErr
	}
	return &portwrite.PortResult{PortID: portID, Action: portcollection.ActionPortBinding, Status: "succeeded"}, nil
}

// ===== Mock OperLogService =====

// mockOperLogService 实现 services.OperLogService，仅 RecordAsync 真实被调（其他方法 panic/不调）。
type mockOperLogService struct {
	recordAsyncCalls int
	lastTitle        string
	lastBusinessType int
	lastOperParam    *string
}

func (m *mockOperLogService) RecordOperLog(ctx context.Context, db *gorm.DB, operLog *models.OperLog) error {
	return nil
}
func (m *mockOperLogService) RecordFromGinContext(c *gin.Context, db *gorm.DB, title string, businessType int, method string) {
}
func (m *mockOperLogService) RecordAsync(db *gorm.DB, title string, businessType int, method, requestMethod, operUrl string,
	operatorName, operatorNickname, deptName *string, operIP *string, operParam, jsonResult, errorMsg *string, status int, costTime int64) {
	m.recordAsyncCalls++
	m.lastTitle = title
	m.lastBusinessType = businessType
	m.lastOperParam = operParam
}

// ===== Test helpers =====

func newTestHandler(t *testing.T) (*PortWriteHandler, *mockPortWriteService, *mockOperLogService, *gorm.DB) {
	t.Helper()
	sqlDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Wave 1 测试手动 AutoMigrate PortWriteAudit + DevicePortStatus + NetworkDevice
	if err := sqlDB.AutoMigrate(&models.PortWriteAudit{}, &models.DevicePortStatus{}, &models.NetworkDevice{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	mockSvc := &mockPortWriteService{}
	mockOperLog := &mockOperLogService{}
	c := &core.Core{
		CoreInfra: &core.CoreInfra{
			DB: &db.Database{DB: sqlDB, Type: "sqlite"},
		},
		CoreServices: &core.CoreServices{
			OperLogService: mockOperLog,
		},
	}
	h := NewPortWriteHandler(mockSvc).WithCore(c)
	return h, mockSvc, mockOperLog, sqlDB
}

// invokeHandler 调用 handler 的 method（按 name dispatch），返回 *httptest.ResponseRecorder。
func invokeHandler(method string, h func(*gin.Context), body interface{}) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/test", h)

	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	// 设置 username 用于 utils.GetUsername
	req.Header.Set("X-Test-User", "tester")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("username", "tester")
	// 直接调 handler（绕过 gin 路由），以注入 c.Set
	h(c)
	return w
}

// invokeWithCtx 接受 preset context 设置，调用指定 handler
func invokeWithCtx(method string, h func(*gin.Context), body interface{}, setup func(c *gin.Context)) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("username", "tester")
	if setup != nil {
		setup(c)
	}
	h(c)
	return w
}

// ===== Tests =====

// TestPortWriteHandler_Shutdown_Success 调 Shutdown → service 返 succeeded → 响应 200 + audit 行落库 + operlog.Record 调用一次
func TestPortWriteHandler_Shutdown_Success(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)

	// 预置 port 行用于 D-02 预 SELECT
	port := &models.DevicePortStatus{
		ID: "port-001", DeviceID: "dev-001", InterfaceName: "GE0/0/1",
		AdminStatus: "up", Dot1xEnabled: false, Description: "old desc",
	}
	if err := sqlDB.Create(port).Error; err != nil {
		t.Fatalf("seed port: %v", err)
	}
	// 预置 device 行用于 oper_param device_name 可读性
	if err := sqlDB.Create(&models.NetworkDevice{BaseModel: models.BaseModel{ID: "dev-001"}, DeviceName: "测试设备-001"}).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}

	mockSvc.shutdownResult = &portwrite.PortResult{
		PortID: "port-001", Action: portcollection.ActionShutdown,
		Status: "succeeded", CommandSent: "system-view | interface GE0/0/1 | shutdown",
	}

	w := invokeWithCtx("Shutdown", h.Shutdown, PortWriteRequest{PortID: "port-001"}, nil)

	assert.Equal(t, http.StatusOK, w.Code)

	// audit 行落库 status=succeeded
	var audit models.PortWriteAudit
	if err := sqlDB.First(&audit, "port_id = ?", "port-001").Error; err != nil {
		t.Fatalf("audit row not found: %v", err)
	}
	assert.Equal(t, "succeeded", audit.Status)
	assert.Equal(t, "system-view | interface GE0/0/1 | shutdown", audit.CommandSent)
	assert.Equal(t, "OK", audit.DeviceResponse)
	assert.Nil(t, audit.FailureReason)
	assert.Nil(t, audit.OperLogID) // Path C: NULL
	assert.Equal(t, "dev-001", audit.DeviceID)
	assert.Equal(t, "tester", audit.Operator)

	// before_value 含 admin_status/dot1x_enabled/description/interface_name
	var before map[string]interface{}
	_ = json.Unmarshal(audit.BeforeValue, &before)
	assert.Equal(t, "up", before["admin_status"])
	assert.Equal(t, "old desc", before["description"])

	// after_value = {"admin_status":"down"} per D-03
	var after map[string]interface{}
	_ = json.Unmarshal(audit.AfterValue, &after)
	assert.Equal(t, "down", after["admin_status"])

	// operlog.Record 被调用一次，module="端口管理"，OperType=OperTypeDisable(13)
	// (Shutdown 关闭端口语义上是 Disable, 比 OperTypeStatus(10) 通用状态变更更精确)
	assert.Equal(t, 1, mockOperLog.recordAsyncCalls)
	assert.Equal(t, "端口管理", mockOperLog.lastTitle)
	assert.Equal(t, 13, mockOperLog.lastBusinessType)
	// oper_param 含 audit_ids + 可读名称（device_name + interface_name）
	assert.NotNil(t, mockOperLog.lastOperParam)
	assert.Contains(t, *mockOperLog.lastOperParam, audit.ID)
	assert.Contains(t, *mockOperLog.lastOperParam, "测试设备-001") // device_name
	assert.Contains(t, *mockOperLog.lastOperParam, "GE0/0/1")    // interface_name
}

// TestPortWriteHandler_Shutdown_NoOp_AlreadyDown service 返 skipped → 响应 200 + audit 行 status=skipped
func TestPortWriteHandler_Shutdown_NoOp_AlreadyDown(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)

	port := &models.DevicePortStatus{ID: "port-002", DeviceID: "dev-001", InterfaceName: "GE0/0/2", AdminStatus: "down"}
	if err := sqlDB.Create(port).Error; err != nil {
		t.Fatalf("seed port: %v", err)
	}

	mockSvc.shutdownResult = &portwrite.PortResult{
		PortID: "port-002", Action: portcollection.ActionShutdown,
		Status: "skipped", NoOp: true, CommandSent: "",
	}

	w := invokeWithCtx("Shutdown", h.Shutdown, PortWriteRequest{PortID: "port-002"}, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var audit models.PortWriteAudit
	if err := sqlDB.First(&audit, "port_id = ?", "port-002").Error; err != nil {
		t.Fatalf("audit row not found: %v", err)
	}
	assert.Equal(t, "skipped", audit.Status)
	assert.Equal(t, "", audit.CommandSent)
	assert.Equal(t, "无需操作", audit.DeviceResponse)
	assert.Nil(t, audit.FailureReason)

	// after_value = before_value (NoOp)
	assert.JSONEq(t, string(audit.BeforeValue), string(audit.AfterValue))

	// operlog 仍被调用
	assert.Equal(t, 1, mockOperLog.recordAsyncCalls)
}

// TestPortWriteHandler_Shutdown_TransportFailed service 返 failed → 响应 200 + audit status=failed + device_response=error
func TestPortWriteHandler_Shutdown_TransportFailed(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)

	port := &models.DevicePortStatus{ID: "port-003", DeviceID: "dev-001", InterfaceName: "GE0/0/3", AdminStatus: "up"}
	if err := sqlDB.Create(port).Error; err != nil {
		t.Fatalf("seed port: %v", err)
	}

	mockSvc.shutdownResult = &portwrite.PortResult{
		PortID: "port-003", Action: portcollection.ActionShutdown,
		Status: "failed", Error: "transport_error: ssh timeout",
		CommandSent: "system-view | interface GE0/0/3 | shutdown",
	}

	w := invokeWithCtx("Shutdown", h.Shutdown, PortWriteRequest{PortID: "port-003"}, nil)
	assert.Equal(t, http.StatusOK, w.Code, "failed PortResult path returns 200 not 4xx")

	// audit 表行数 = 1（failed 路径写 audit）
	var audits []models.PortWriteAudit
	sqlDB.Find(&audits, "port_id = ?", "port-003")
	assert.Len(t, audits, 1)
	assert.Equal(t, "failed", audits[0].Status)
	assert.Equal(t, "transport_error: ssh timeout", audits[0].DeviceResponse)
	assert.NotNil(t, audits[0].FailureReason)
	assert.Equal(t, "transport_error: ssh timeout", *audits[0].FailureReason)

	// operlog 被调用
	assert.Equal(t, 1, mockOperLog.recordAsyncCalls)
}

// TestPortWriteHandler_Shutdown_PortNotFound service 返 ErrPortNotFound → 404 + 不写 audit + 不调 operlog
//
// 注：response.Error(c, int, msg) 项目惯例把 int 当 business code，HTTPStatus 固定 400。
// 测试断言走 body.code == 404 + body.message == "端口不存在"。
func TestPortWriteHandler_Shutdown_PortNotFound(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)

	mockSvc.shutdownErr = portwrite.ErrPortNotFound

	w := invokeWithCtx("Shutdown", h.Shutdown, PortWriteRequest{PortID: "no-such-port"}, nil)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, http.StatusNotFound, resp.Code, "body=%s", w.Body.String())
	assert.Equal(t, "端口不存在", resp.Message)

	// audit 表无新行
	var count int64
	sqlDB.Model(&models.PortWriteAudit{}).Count(&count)
	assert.Equal(t, int64(0), count, "sentinel path must NOT write audit")

	// operlog 未被调用
	assert.Equal(t, 0, mockOperLog.recordAsyncCalls, "sentinel path must NOT call operlog")
}

// TestPortWriteHandler_SetDescription OperType=OperTypeUpdate(=2); audit before/after 含 description
func TestPortWriteHandler_SetDescription(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)

	port := &models.DevicePortStatus{ID: "port-d1", DeviceID: "dev-002", InterfaceName: "GE0/0/5", AdminStatus: "up", Description: "old desc"}
	if err := sqlDB.Create(port).Error; err != nil {
		t.Fatalf("seed port: %v", err)
	}

	mockSvc.setDescriptionOut = &portwrite.PortResult{
		PortID: "port-d1", Action: portcollection.ActionDescription,
		Status: "succeeded", CurrentState: "new desc", CommandSent: "system-view | interface GE0/0/5 | description new desc",
	}

	w := invokeWithCtx("SetDescription", h.SetDescription, PortWriteRequest{PortID: "port-d1", Description: "new desc"}, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var audit models.PortWriteAudit
	if err := sqlDB.First(&audit, "port_id = ?", "port-d1").Error; err != nil {
		t.Fatalf("audit row not found: %v", err)
	}
	assert.Equal(t, "succeeded", audit.Status)

	// after_value 含 description 字段（来自 PortResult.CurrentState）
	var after map[string]interface{}
	_ = json.Unmarshal(audit.AfterValue, &after)
	assert.Equal(t, "new desc", after["description"])

	// OperType=OperTypeUpdate(=2)
	assert.Equal(t, 1, mockOperLog.recordAsyncCalls)
	assert.Equal(t, 2, mockOperLog.lastBusinessType)
}

// TestPortWriteHandler_Batch 调 BatchWritePorts → operlog(OperTypeBatch=16, WithOperParam 含 audit_ids) + N 条 audit + 200
func TestPortWriteHandler_Batch(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)

	// 预置 device + 3 个端口
	if err := sqlDB.Create(&models.NetworkDevice{BaseModel: models.BaseModel{ID: "bdev-1"}, DeviceName: "批量测试设备"}).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}
	for i, pid := range []string{"bp-1", "bp-2", "bp-3"} {
		port := &models.DevicePortStatus{
			ID: pid, DeviceID: "bdev-1", InterfaceName: "GE0/0/" + string(rune('1'+i)),
			AdminStatus: "up",
		}
		if err := sqlDB.Create(port).Error; err != nil {
			t.Fatalf("seed port %s: %v", pid, err)
		}
	}

	mockSvc.batchResult = &portwrite.BatchResult{
		Succeeded: []portwrite.PortResult{
			{PortID: "bp-1", Action: portcollection.ActionShutdown, Status: "succeeded", CommandSent: "shutdown"},
		},
		Failed: []portwrite.PortResult{
			{PortID: "bp-2", Action: portcollection.ActionShutdown, Status: "failed", Error: "transport_error", CommandSent: "shutdown"},
		},
		Skipped: []portwrite.PortResult{
			{PortID: "bp-3", Action: portcollection.ActionShutdown, Status: "skipped", NoOp: true, CommandSent: ""},
		},
	}

	w := invokeWithCtx("BatchWrite", h.BatchWrite, portwrite.BatchWriteRequest{
		DeviceID: "bdev-1", Action: portcollection.ActionShutdown, PortIDs: []string{"bp-1", "bp-2", "bp-3"},
	}, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// audit 行数 = 3（Succeeded 1 + Failed 1 + Skipped 1）
	var audits []models.PortWriteAudit
	sqlDB.Find(&audits, "device_id = ?", "bdev-1")
	assert.Len(t, audits, 3, "batch must write N audits = len(Succeeded)+len(Failed)+len(Skipped)")

	// operlog 调用一次（汇总）
	assert.Equal(t, 1, mockOperLog.recordAsyncCalls)
	assert.Equal(t, 16, mockOperLog.lastBusinessType, "OperTypeBatch=16")

	// oper_param 含 audit_ids 数组（3 条）
	assert.NotNil(t, mockOperLog.lastOperParam)
	var param map[string]interface{}
	_ = json.Unmarshal([]byte(*mockOperLog.lastOperParam), &param)
	auditIDs, ok := param["audit_ids"].([]interface{})
	assert.True(t, ok, "audit_ids must be array")
	assert.Len(t, auditIDs, 3)
	assert.Equal(t, float64(3), param["batch_size"])
	assert.Equal(t, float64(1), param["succeeded_count"])
	assert.Equal(t, float64(1), param["failed_count"])
	assert.Equal(t, float64(1), param["skipped_count"])
	// 可读性字段：device_name + 各分类接口名（2026-07-09 优化）
	assert.Equal(t, "批量测试设备", param["device_name"])
	succeededIfaces, _ := param["succeeded_interfaces"].([]interface{})
	assert.Len(t, succeededIfaces, 1)
	assert.Equal(t, "GE0/0/1", succeededIfaces[0]) // bp-1 的接口名
	failedIfaces, _ := param["failed_interfaces"].([]interface{})
	assert.Equal(t, "GE0/0/2", failedIfaces[0]) // bp-2 的接口名

	// response body 含 BatchResult 三切片
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Succeeded []portwrite.PortResult `json:"succeeded"`
			Failed    []portwrite.PortResult `json:"failed"`
			Skipped   []portwrite.PortResult `json:"skipped"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 0, resp.Code)
	assert.Len(t, resp.Data.Succeeded, 1)
	assert.Len(t, resp.Data.Failed, 1)
	assert.Len(t, resp.Data.Skipped, 1)
}

// TestPortWriteHandler_Batch_ExceedsMax service 返 ErrBatchTooLarge → 400 + 不写 audit + 不调 operlog
func TestPortWriteHandler_Batch_ExceedsMax(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)

	mockSvc.batchErr = portwrite.ErrBatchTooLarge

	w := invokeWithCtx("BatchWrite", h.BatchWrite, portwrite.BatchWriteRequest{
		DeviceID: "x", Action: portcollection.ActionShutdown,
		PortIDs: []string{"a", "b", "c"},
	}, nil)
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "批量端口数超过上限 50", resp.Message)

	var count int64
	sqlDB.Model(&models.PortWriteAudit{}).Count(&count)
	assert.Equal(t, int64(0), count, "sentinel path must NOT write audit")
	assert.Equal(t, 0, mockOperLog.recordAsyncCalls, "sentinel path must NOT call operlog")
}

// TestPortWriteHandler_WithOperID_NotAdded Path C 守卫：handler 文件不含 WithOperID / WithJsonResult
// (negative source-grep assertion). readFile helper lives in port_write_router_test.go (same package).
func TestPortWriteHandler_WithOperID_NotAdded(t *testing.T) {
	src := readFile(t, "port_write_handler.go")
	assert.NotContains(t, src, "operlog.WithOperID",
		"handler 文件禁止使用 operlog.WithOperID（Path C invariant）")
	assert.NotContains(t, src, "operlog.WithJsonResult",
		"handler 文件禁止使用 operlog.WithJsonResult（Path C invariant）")
}

// ===== v1.20.1 handler tests (Phase 56 W3) =====
//
// 覆盖 2 新 handler (SetAccessVlan / PortBinding) 的 4 关键路径:
// 1. binding 失败 (400) - vlanId 越界 / op 非 oneof
// 2. service sentinel 失败 (400) - ErrVlanIdOutOfRange / ErrBindOpInvalid
// 3. 成功路径 - audit after_value 从 PortResult.Extra 取，operlog 调用
// 4. 源码 grep - 4 新 sentinel 翻译存在 + buildAfterValue 接收 PortResult

// TestPortWriteHandler_SetAccessVlan_BindingRejectsOutOfRangeVlanId
// binding tag min=1,max=4094 在 handler 入口拦截 vlanId=0 / vlanId=4095 → 400，不调 service。
func TestPortWriteHandler_SetAccessVlan_BindingRejectsOutOfRangeVlanId(t *testing.T) {
	h, _, mockOperLog, sqlDB := newTestHandler(t)

	for _, vlan := range []int{0, 4095, 10000} {
		w := invokeWithCtx("SetAccessVlan", h.SetAccessVlan, SetAccessVlanRequest{PortID: "p1", VLANID: vlan}, nil)
		var resp struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, http.StatusBadRequest, resp.Code, "vlanId=%d should be rejected by binding tag", vlan)
	}

	// audit 表无新行 + operlog 未调用
	var count int64
	sqlDB.Model(&models.PortWriteAudit{}).Count(&count)
	assert.Equal(t, int64(0), count, "binding-fail path must NOT write audit")
	assert.Equal(t, 0, mockOperLog.recordAsyncCalls, "binding-fail path must NOT call operlog")
	// mock service 未被调用（mockSvc 仍是零值，若被调用会返默认 succeeded；这里靠 audit count 验证未走 service 成功路径）
}

// TestPortWriteHandler_SetAccessVlan_ServiceSentinelReturns400
// service 返 ErrVlanIdOutOfRange → handler 翻译为 400，不写 audit / 不调 operlog。
func TestPortWriteHandler_SetAccessVlan_ServiceSentinelReturns400(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)
	mockSvc.setAccessVlanErr = portwrite.ErrVlanIdOutOfRange

	w := invokeWithCtx("SetAccessVlan", h.SetAccessVlan, SetAccessVlanRequest{PortID: "p1", VLANID: 100}, nil)
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "VLAN ID 必须在 1-4094 之间", resp.Message)

	// audit 表无新行 + operlog 未调用
	var count int64
	sqlDB.Model(&models.PortWriteAudit{}).Count(&count)
	assert.Equal(t, int64(0), count, "sentinel path must NOT write audit")
	assert.Equal(t, 0, mockOperLog.recordAsyncCalls, "sentinel path must NOT call operlog")
}

// TestPortWriteHandler_SetAccessVlan_SuccessExtraPopulatesAfterValue
// service 返 succeeded + Extra={vlanId:100} → audit after_value 含 vlanId=100 + operlog(OperTypeUpdate=2)。
func TestPortWriteHandler_SetAccessVlan_SuccessExtraPopulatesAfterValue(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)
	port := &models.DevicePortStatus{ID: "vlan-p1", DeviceID: "dev-v", InterfaceName: "GE0/0/10", AdminStatus: "up"}
	if err := sqlDB.Create(port).Error; err != nil {
		t.Fatalf("seed port: %v", err)
	}

	mockSvc.setAccessVlanOut = &portwrite.PortResult{
		PortID:     "vlan-p1",
		Action:     portcollection.ActionSetAccessVLAN,
		Status:     "succeeded",
		CommandSent: "system-view | interface GE0/0/10 | port link-type access | port default vlan 100",
		Extra:      map[string]interface{}{"vlanId": 100},
	}

	w := invokeWithCtx("SetAccessVlan", h.SetAccessVlan, SetAccessVlanRequest{PortID: "vlan-p1", VLANID: 100}, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var audit models.PortWriteAudit
	if err := sqlDB.First(&audit, "port_id = ?", "vlan-p1").Error; err != nil {
		t.Fatalf("audit row not found: %v", err)
	}
	assert.Equal(t, "succeeded", audit.Status)
	// after_value 含 vlanId=100（来自 PortResult.Extra，由 buildAfterValue 读出）
	var after map[string]interface{}
	_ = json.Unmarshal(audit.AfterValue, &after)
	assert.Equal(t, float64(100), after["vlanId"], "after_value should contain vlanId=100 from PortResult.Extra; got=%s", string(audit.AfterValue))

	// operlog 调用一次，OperType=OperTypeUpdate(=2)
	assert.Equal(t, 1, mockOperLog.recordAsyncCalls)
	assert.Equal(t, "端口管理", mockOperLog.lastTitle)
	assert.Equal(t, 2, mockOperLog.lastBusinessType, "SetAccessVlan → OperTypeUpdate=2")
}

// TestPortWriteHandler_PortBinding_OperTypeBranching
// op=add → OperTypeCreate(1); op=remove → OperTypeDelete(3)。绑定 oneof 拒绝其他值。
func TestPortWriteHandler_PortBinding_OperTypeBranching(t *testing.T) {
	t.Run("op_add_OperTypeCreate", func(t *testing.T) {
		h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)
		sqlDB.Create(&models.DevicePortStatus{ID: "bnd-p1", DeviceID: "bnd-dev", InterfaceName: "GE0/0/20", AdminStatus: "up"})
		sqlDB.Create(&models.NetworkDevice{BaseModel: models.BaseModel{ID: "bnd-dev"}, DeviceName: "绑定测试设备"})

		mockSvc.portBindingOut = &portwrite.PortResult{
			PortID: "bnd-p1", Action: portcollection.ActionPortBinding, Status: "succeeded",
			Extra: map[string]interface{}{"bindOp": "add", "ipAddress": "10.62.25.5", "macAddress": "AA:BB:CC:DD:EE:FF"},
		}
		w := invokeWithCtx("PortBinding", h.PortBinding, PortBindingRequest{
			PortID: "bnd-p1", Op: "add", IPAddress: "10.62.25.5", MACAddress: "AA:BB:CC:DD:EE:FF",
		}, nil)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 1, mockOperLog.recordAsyncCalls)
		assert.Equal(t, "端口管理", mockOperLog.lastTitle)
		assert.Equal(t, 1, mockOperLog.lastBusinessType, "op=add → OperTypeCreate=1")

		// audit after_value 含 ipAddress / macAddress / bindOp（来自 PortResult.Extra）
		var audit models.PortWriteAudit
		if err := sqlDB.First(&audit, "port_id = ?", "bnd-p1").Error; err != nil {
			t.Fatalf("audit row not found: %v", err)
		}
		var after map[string]interface{}
		_ = json.Unmarshal(audit.AfterValue, &after)
		assert.Equal(t, "10.62.25.5", after["ipAddress"])
		assert.Equal(t, "AA:BB:CC:DD:EE:FF", after["macAddress"])
		assert.Equal(t, "add", after["bindOp"])
	})
	t.Run("op_remove_OperTypeDelete", func(t *testing.T) {
		h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)
		sqlDB.Create(&models.DevicePortStatus{ID: "bnd-p2", DeviceID: "bnd-dev2", InterfaceName: "GE0/0/21", AdminStatus: "up"})
		sqlDB.Create(&models.NetworkDevice{BaseModel: models.BaseModel{ID: "bnd-dev2"}, DeviceName: "绑定测试设备2"})

		mockSvc.portBindingOut = &portwrite.PortResult{
			PortID: "bnd-p2", Action: portcollection.ActionPortBinding, Status: "succeeded",
			Extra: map[string]interface{}{"bindOp": "remove", "ipAddress": "10.62.25.6"},
		}
		w := invokeWithCtx("PortBinding", h.PortBinding, PortBindingRequest{
			PortID: "bnd-p2", Op: "remove", IPAddress: "10.62.25.6",
		}, nil)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 1, mockOperLog.recordAsyncCalls)
		assert.Equal(t, 3, mockOperLog.lastBusinessType, "op=remove → OperTypeDelete=3")

		// audit after_value 含 bindOp=remove + ipAddress（无 macAddress，因请求未传）
		var audit models.PortWriteAudit
		if err := sqlDB.First(&audit, "port_id = ?", "bnd-p2").Error; err != nil {
			t.Fatalf("audit row not found: %v", err)
		}
		var after map[string]interface{}
		_ = json.Unmarshal(audit.AfterValue, &after)
		assert.Equal(t, "remove", after["bindOp"])
		assert.Equal(t, "10.62.25.6", after["ipAddress"])
		_, hasMac := after["macAddress"]
		assert.False(t, hasMac, "macAddress should be absent when Extra has no macAddress")
	})
	t.Run("op_invalid_rejected_by_binding", func(t *testing.T) {
		h, _, _, _ := newTestHandler(t)
		w := invokeWithCtx("PortBinding", h.PortBinding, PortBindingRequest{
			PortID: "bnd-x", Op: "DELETE", IPAddress: "10.62.25.5",
		}, nil)
		var resp struct{ Code int }
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, http.StatusBadRequest, resp.Code, "op=DELETE should be rejected by oneof binding tag")
	})
}

// TestPortWriteHandler_PortBinding_ServiceSentinelsReturn400
// service 返 ErrBindOpInvalid / ErrIPAddressInvalid / ErrMACAddressInvalid → 翻译为对应中文 400。
func TestPortWriteHandler_PortBinding_ServiceSentinelsReturn400(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		message string
	}{
		{"bind_op_invalid", portwrite.ErrBindOpInvalid, "绑定操作必须是 add 或 remove"},
		{"ip_invalid", portwrite.ErrIPAddressInvalid, "IP 地址格式不合法"},
		{"mac_invalid", portwrite.ErrMACAddressInvalid, "MAC 地址格式不合法"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)
			mockSvc.portBindingErr = tc.err
			w := invokeWithCtx("PortBinding", h.PortBinding, PortBindingRequest{
				PortID: "p1", Op: "add", IPAddress: "10.62.25.5",
			}, nil)
			var resp struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			assert.Equal(t, http.StatusBadRequest, resp.Code)
			assert.Equal(t, tc.message, resp.Message)
			var count int64
			sqlDB.Model(&models.PortWriteAudit{}).Count(&count)
			assert.Equal(t, int64(0), count, "sentinel path must NOT write audit")
			assert.Equal(t, 0, mockOperLog.recordAsyncCalls, "sentinel path must NOT call operlog")
		})
	}
}

// TestPortWriteHandler_V1201_SourceAssertions locks v1.20.1 wiring via source grep:
//   - 4 new sentinel→HTTP-400 translations present
//   - buildAfterValue signature takes *PortResult (so it can read Extra)
func TestPortWriteHandler_V1201_SourceAssertions(t *testing.T) {
	src := readFile(t, "port_write_handler.go")
	// 4 sentinel 翻译
	assert.Contains(t, src, `errors.Is(err, portwrite.ErrVlanIdOutOfRange)`)
	assert.Contains(t, src, `errors.Is(err, portwrite.ErrBindOpInvalid)`)
	assert.Contains(t, src, `errors.Is(err, portwrite.ErrIPAddressInvalid)`)
	assert.Contains(t, src, `errors.Is(err, portwrite.ErrMACAddressInvalid)`)
	// buildAfterValue 签名扩展（接收 *PortResult）
	assert.Contains(t, src, "func buildAfterValue(action portcollection.PortAction, pr *portwrite.PortResult) json.RawMessage")
	// 2 新 case
	assert.Contains(t, src, "case portcollection.ActionSetAccessVLAN:")
	assert.Contains(t, src, "case portcollection.ActionPortBinding:")
	// 2 新 request struct
	assert.Contains(t, src, "type SetAccessVlanRequest struct")
	assert.Contains(t, src, "type PortBindingRequest struct")
	// 2 新 handler methods
	assert.Contains(t, src, "func (h *PortWriteHandler) SetAccessVlan(c *gin.Context)")
	assert.Contains(t, src, "func (h *PortWriteHandler) PortBinding(c *gin.Context)")
	// OperType 分流（design.md §6）
	assert.Contains(t, src, "operType := operlog.OperTypeCreate")
	assert.Contains(t, src, "operType = operlog.OperTypeDelete")
	// 禁止引入敏感关键词处理（IP/MAC 非敏感）
	assert.NotContains(t, src, "RecordWithBody")
}

// TestBuildBeforeValue_IncludesVlan (CR-02 2026-07-09 修复守护)
//
// set_access_vlan 的 skipped 路径会让 after_value 覆盖为 before_value,
// buildBeforeValue 必须包含 vlan 字段,否则审计行丢失审计核心字段。
// 表驱动覆盖 3 种 VLAN 状态：已采集 (*int 100) / 未采集 (nil) / 零值 (*int 0)。
func TestBuildBeforeValue_IncludesVlan(t *testing.T) {
	tests := []struct {
		name        string
		vlan        *int
		wantVlanSet bool
		wantVlan    int
	}{
		{"vlan_collected", ptrInt(100), true, 100},
		{"vlan_nil_uncroned", nil, false, 0},
		{"vlan_zero_value", ptrInt(0), true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := &models.DevicePortStatus{
				InterfaceName: "GE0/0/1",
				AdminStatus:   "up",
				Dot1xEnabled:  false,
				Description:   "uplink",
				VLAN:         tt.vlan,
			}
			raw := buildBeforeValue(port)
			var got map[string]interface{}
			assert.NoError(t, json.Unmarshal(raw, &got))
			if tt.wantVlanSet {
				// JSON unmarshal 把 number 解为 float64
				assert.Equal(t, float64(tt.wantVlan), got["vlan"], "vlan field must be present in before_value")
			} else {
				_, has := got["vlan"]
				assert.False(t, has, "vlan field must be omitted when port.VLAN is nil")
			}
		})
	}
}

// TestBuildAuditRow_SkippedSetAccessVlan_ContainsVlan (CR-02 集成守护)
//
// buildAuditRow 对 skipped set_access_vlan 必须保留 vlan 字段(after_value=before_value
// 覆盖后,both sides 都应含 vlan)。这是审计员诊断 "为什么 vlan 没变化" 的唯一线索。
func TestBuildAuditRow_SkippedSetAccessVlan_ContainsVlan(t *testing.T) {
	vlan100 := 100
	port := &models.DevicePortStatus{
		InterfaceName: "GE0/0/1",
		AdminStatus:   "up",
		VLAN:         &vlan100,
	}
	beforeValue := buildBeforeValue(port)

	pr := &portwrite.PortResult{
		PortID:       "port-1",
		Action:       portcollection.ActionSetAccessVLAN,
		Status:       "skipped", // pre-state NoOp: vlan_id already matches
		NoOp:         true,
		CurrentState: "vlan_match",
	}

	row := buildAuditRow(pr, beforeValue, "device-1", "test-op")
	assert.Equal(t, "skipped", row.Status)

	// 关键：after_value == before_value(覆盖路径),但 vlan 字段必须在场
	var before, after map[string]interface{}
	assert.NoError(t, json.Unmarshal(beforeValue, &before))
	assert.NoError(t, json.Unmarshal(row.AfterValue, &after))
	assert.Equal(t, float64(100), before["vlan"], "before_value must contain vlan=100")
	assert.Equal(t, float64(100), after["vlan"], "after_value must contain vlan=100 (CR-02 regression)")
	assert.Equal(t, beforeValue, row.AfterValue, "skipped path: after_value == before_value")
}

// ptrInt 辅助构造 *int 字面量。
func ptrInt(i int) *int { return &i }

// ============================================================================
// Phase 74-03: UndoShutdown / EnableDot1x / DisableDot1x / SetDescription
// handler entry points (all were 0% / 66.7% in baseline).
// ============================================================================

// TestPortWriteHandler_UndoShutdown_Success 走通 UndoShutdown 完整成功路径 —
// 端口行存在 + service 返 succeeded → 200 + audit + operlog.Record。
func TestPortWriteHandler_UndoShutdown_Success(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)

	port := &models.DevicePortStatus{
		ID: "port-und-1", DeviceID: "dev-und-1", InterfaceName: "GE0/0/10",
		AdminStatus: "down",
	}
	require.NoError(t, sqlDB.Create(port).Error)
	require.NoError(t, sqlDB.Create(&models.NetworkDevice{BaseModel: models.BaseModel{ID: "dev-und-1"}, DeviceName: "undo-dev"}).Error)

	w := invokeWithCtx("UndoShutdown", h.UndoShutdown, PortWriteRequest{PortID: "port-und-1"}, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, mockOperLog.recordAsyncCalls)
	assert.Equal(t, "端口管理", mockOperLog.lastTitle)
	assert.Equal(t, operlog.OperTypeEnable, mockOperLog.lastBusinessType,
		"UndoShutdown maps to OperTypeEnable(12) per port_write_handler.go:115")
	assert.Equal(t, 1, mockSvc.undoCalls, "service.UndoShutdown must be called once")
}

// TestPortWriteHandler_UndoShutdown_PortNotFound 路径 — service 返
// portwrite.ErrPortNotFound sentinel → 不写 audit + 不调 operlog.
//
// 注：response.Error(c, http.StatusNotFound, ...) 的 int 首参硬编码为 400
// (toAppError case int → HTTPStatus=400 quirk — 见 pkg/response/response.go)，
// 这里断言 400 而非 404 是对 pre-existing 行为的如实记录 (D-12)。
func TestPortWriteHandler_UndoShutdown_PortNotFound(t *testing.T) {
	h, _, mockOperLog, sqlDB := newTestHandler(t)
	_ = sqlDB

	mockSvc := h.service.(*mockPortWriteService)
	mockSvc.undoShutdownErr = portwrite.ErrPortNotFound

	w := invokeWithCtx("UndoShutdown", h.UndoShutdown, PortWriteRequest{PortID: "ghost"}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code,
		"sentinel ErrPortNotFound maps to 400 due to response.Error int-first-arg quirk (D-12)")
	assert.Equal(t, 0, mockOperLog.recordAsyncCalls, "sentinel path must NOT call operlog")
}

// TestPortWriteHandler_UndoShutdown_BindingError 触发 binding 校验失败路径。
func TestPortWriteHandler_UndoShutdown_BindingError(t *testing.T) {
	h, _, mockOperLog, sqlDB := newTestHandler(t)
	_ = mockOperLog
	_ = sqlDB

	w := invokeWithCtx("UndoShutdown", h.UndoShutdown, map[string]string{}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestPortWriteHandler_EnableDot1x_Success 走通 EnableDot1x 完整成功路径 —
// OperType=Enable(12), 调用 service.EnableDot1x。
func TestPortWriteHandler_EnableDot1x_Success(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)

	port := &models.DevicePortStatus{
		ID: "port-dot1x-en", DeviceID: "dev-dot1x-en", InterfaceName: "GE0/0/20",
		AdminStatus: "up", Dot1xEnabled: false,
	}
	require.NoError(t, sqlDB.Create(port).Error)
	require.NoError(t, sqlDB.Create(&models.NetworkDevice{BaseModel: models.BaseModel{ID: "dev-dot1x-en"}, DeviceName: "dot1x-en-dev"}).Error)

	w := invokeWithCtx("EnableDot1x", h.EnableDot1x, PortWriteRequest{PortID: "port-dot1x-en"}, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, mockOperLog.recordAsyncCalls)
	assert.Equal(t, operlog.OperTypeEnable, mockOperLog.lastBusinessType,
		"EnableDot1x maps to OperTypeEnable(12) per port_write_handler.go:141")
	assert.Equal(t, 1, mockSvc.enableCalls, "service.EnableDot1x must be called once")

	// audit 行落库 + dot1x_enabled=true 在 after_value
	var audit models.PortWriteAudit
	require.NoError(t, sqlDB.First(&audit, "port_id = ?", "port-dot1x-en").Error)
	var after map[string]interface{}
	_ = json.Unmarshal(audit.AfterValue, &after)
	assert.Equal(t, true, after["dot1x_enabled"])
}

// TestPortWriteHandler_DisableDot1x_Success 走通 DisableDot1x 完整成功路径 —
// OperType=Disable(13), 调用 service.DisableDot1x。
func TestPortWriteHandler_DisableDot1x_Success(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)

	port := &models.DevicePortStatus{
		ID: "port-dot1x-dis", DeviceID: "dev-dot1x-dis", InterfaceName: "GE0/0/21",
		AdminStatus: "up", Dot1xEnabled: true,
	}
	require.NoError(t, sqlDB.Create(port).Error)
	require.NoError(t, sqlDB.Create(&models.NetworkDevice{BaseModel: models.BaseModel{ID: "dev-dot1x-dis"}, DeviceName: "dot1x-dis-dev"}).Error)

	w := invokeWithCtx("DisableDot1x", h.DisableDot1x, PortWriteRequest{PortID: "port-dot1x-dis"}, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, mockOperLog.recordAsyncCalls)
	assert.Equal(t, operlog.OperTypeDisable, mockOperLog.lastBusinessType,
		"DisableDot1x maps to OperTypeDisable(13) per port_write_handler.go:154")
	assert.Equal(t, 1, mockSvc.disableCalls, "service.DisableDot1x must be called once")

	var audit models.PortWriteAudit
	require.NoError(t, sqlDB.First(&audit, "port_id = ?", "port-dot1x-dis").Error)
	var after map[string]interface{}
	_ = json.Unmarshal(audit.AfterValue, &after)
	assert.Equal(t, false, after["dot1x_enabled"])
}

// TestPortWriteHandler_DisableDot1x_DeviceNotFound 路径 — service 返
// portwrite.ErrDeviceNotFound sentinel → 不写 audit + 不调 operlog.
//
// 注：同上 response.Error int-first-arg quirk — 实际 HTTP 400 而非 404。
func TestPortWriteHandler_DisableDot1x_DeviceNotFound(t *testing.T) {
	h, _, mockOperLog, sqlDB := newTestHandler(t)
	_ = sqlDB

	mockSvc := h.service.(*mockPortWriteService)
	mockSvc.disableDot1xErr = portwrite.ErrDeviceNotFound

	w := invokeWithCtx("DisableDot1x", h.DisableDot1x, PortWriteRequest{PortID: "port-ghost"}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code,
		"sentinel ErrDeviceNotFound maps to 400 due to response.Error int-first-arg quirk (D-12)")
	assert.Equal(t, 0, mockOperLog.recordAsyncCalls, "sentinel path must NOT call operlog")
}

// TestPortWriteHandler_EnableDot1x_BindingError 触发 binding 校验失败。
func TestPortWriteHandler_EnableDot1x_BindingError(t *testing.T) {
	h, _, _, _ := newTestHandler(t)

	w := invokeWithCtx("EnableDot1x", h.EnableDot1x, map[string]string{}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestPortWriteHandler_DisableDot1x_BindingError 触发 binding 校验失败。
func TestPortWriteHandler_DisableDot1x_BindingError(t *testing.T) {
	h, _, _, _ := newTestHandler(t)

	w := invokeWithCtx("DisableDot1x", h.DisableDot1x, map[string]string{}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestPortWriteHandler_SetDescription_FailedStatus walks the result.Status=="failed"
// path through execSinglePort — audit is still written + operlog.Record still called.
//
// (SetDescription was 66.7% in baseline; this exercises the after_value=different
// from before_value branch where description actually changes.)
func TestPortWriteHandler_SetDescription_FailedStatus(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)

	port := &models.DevicePortStatus{
		ID: "port-desc-fail", DeviceID: "dev-desc-fail", InterfaceName: "GE0/0/30",
		AdminStatus: "up", Description: "old desc",
	}
	require.NoError(t, sqlDB.Create(port).Error)
	require.NoError(t, sqlDB.Create(&models.NetworkDevice{BaseModel: models.BaseModel{ID: "dev-desc-fail"}, DeviceName: "desc-fail-dev"}).Error)

	// Force the failed status path: mock returns a result with Status="failed"
	// and a non-nil error → execSinglePort falls through to audit + operlog + 200.
	mockSvc.setDescriptionErr = errors.New("device rejected command")
	mockSvc.setDescriptionOut = &portwrite.PortResult{
		PortID: "port-desc-fail", Action: portcollection.ActionDescription,
		Status: "failed", Error: "device refused",
	}

	w := invokeWithCtx("SetDescription", h.SetDescription,
		PortWriteRequest{PortID: "port-desc-fail", Description: "new desc"}, nil)

	// result.Status==failed path returns 200 (per RESEATCH §3.3)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, mockOperLog.recordAsyncCalls, "failed path still writes operlog")
	assert.Equal(t, operlog.OperTypeUpdate, mockOperLog.lastBusinessType,
		"SetDescription maps to OperTypeUpdate(2)")
}
