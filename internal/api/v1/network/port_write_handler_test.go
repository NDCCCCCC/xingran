package network

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/db"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/portcollection"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/portwrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
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

	// operlog.Record 被调用一次，module="端口管理"，OperType=OperTypeStatus(10)
	assert.Equal(t, 1, mockOperLog.recordAsyncCalls)
	assert.Equal(t, "端口管理", mockOperLog.lastTitle)
	assert.Equal(t, 10, mockOperLog.lastBusinessType)
	// oper_param 含 audit_ids
	assert.NotNil(t, mockOperLog.lastOperParam)
	assert.Contains(t, *mockOperLog.lastOperParam, audit.ID)
	assert.Contains(t, *mockOperLog.lastOperParam, "port-001")
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

	// 预置 3 个端口
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
