package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/glebarez/sqlite"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/xingran-next/xingran-go-backend/internal/services/portwrite"
)

// ====================== Phase 129 真机修复：handler 层能力矩阵拒绝 ======================
//
// 能力矩阵拒绝（ErrActionNotSupported）必须翻译成 400 + 真机依据，
// 且不写审计行、不调 operlog（与其它 sentinel 路径同口径）。
//
// 另覆盖 Phase 123 的 409（写互斥）/ 503（设备不可达）翻译分支与
// Phase 123 PORTW-06 的 skipPreflight 选项透传。

// TestPortWriteHandler_ActionNotSupportedReturns400 能力矩阵拒绝 → 400 + 真机依据。
func TestPortWriteHandler_ActionNotSupportedReturns400(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)

	// 模拟 service 层能力矩阵拒绝（reason 含真机依据）
	mockSvc.portBindingErr = fmt.Errorf("%w: vendor=huawei model=\"S8700-10\" action=%s — %s",
		portcollection.ErrActionNotSupported, portcollection.ActionPortBinding,
		"2026-11-01 真机：user-bind static ip-address ? 仅接受 X.X.X.X")

	w := invokeWithCtx("PortBinding", h.PortBinding, PortBindingRequest{
		PortID: "p1", Op: "add", IPAddress: "10.62.25.5", MACAddress: "AA-BB-CC-DD-EE-FF",
	}, nil)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, http.StatusBadRequest, resp.Code, "能力矩阵拒绝必须翻译为 400")
	assert.Contains(t, resp.Message, "2026-11-01 真机", "400 文案必须携带真机依据")
	assert.Contains(t, resp.Message, "S8700-10")

	// sentinel 路径不写 audit / 不调 operlog
	var count int64
	sqlDB.Model(&models.PortWriteAudit{}).Count(&count)
	assert.Equal(t, int64(0), count, "sentinel 路径不得写审计行")
	assert.Equal(t, 0, mockOperLog.recordAsyncCalls, "sentinel 路径不得调 operlog")
}

// TestPortWriteHandler_PortWriteLockedReturns409 并发写互斥 → 409 + 持锁者信息。
func TestPortWriteHandler_PortWriteLockedReturns409(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)
	mockSvc.shutdownErr = fmt.Errorf("%w: held by %q (in-process)", portwrite.ErrPortWriteLocked, "other@tester")

	w := invokeWithCtx("Shutdown", h.Shutdown, PortWriteRequest{PortID: "p1"}, nil)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, http.StatusConflict, resp.Code, "写互斥冲突必须翻译为 409")
	assert.Contains(t, resp.Message, "端口正在被其他写操作处理")
	assert.Contains(t, resp.Message, "other@tester", "409 文案必须带持锁者信息")

	var count int64
	sqlDB.Model(&models.PortWriteAudit{}).Count(&count)
	assert.Equal(t, int64(0), count)
	assert.Equal(t, 0, mockOperLog.recordAsyncCalls)
}

// TestPortWriteHandler_DeviceUnreachableReturns503 不可达预检失败 → 503。
func TestPortWriteHandler_DeviceUnreachableReturns503(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)
	mockSvc.shutdownErr = fmt.Errorf("%w: device dev-1 ping timeout", portwrite.ErrDeviceUnreachable)

	w := invokeWithCtx("Shutdown", h.Shutdown, PortWriteRequest{PortID: "p1"}, nil)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, http.StatusServiceUnavailable, resp.Code, "不可达必须翻译为 503")
	assert.Contains(t, resp.Message, "设备不可达")
	assert.Contains(t, resp.Message, "skipPreflight", "文案应提示可由管理员强制下发")

	var count int64
	sqlDB.Model(&models.PortWriteAudit{}).Count(&count)
	assert.Equal(t, int64(0), count)
	assert.Equal(t, 0, mockOperLog.recordAsyncCalls)
}

// TestPortWriteHandler_SkipPreflightPassesOptionToService skipPreflight=true 时
// 必须把 WithSkipPreflight 选项透传给 service（false 时不透传）。
func TestPortWriteHandler_SkipPreflightPassesOptionToService(t *testing.T) {
	spy := &optsSpyService{mockPortWriteService: &mockPortWriteService{}}
	h, _, _ := newTestHandlerWithService(t, spy)

	// skipPreflight=true → 透传 1 个选项
	w := invokeWithCtx("Shutdown", h.Shutdown, PortWriteRequest{PortID: "p1", SkipPreflight: true}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, spy.gotOpts, 1, "skipPreflight=true 必须透传 WithSkipPreflight 选项")

	// skipPreflight 缺省 → 不透传任何选项
	spy.gotOpts = nil
	w = invokeWithCtx("Shutdown", h.Shutdown, PortWriteRequest{PortID: "p1"}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, spy.gotOpts, "未声明 skipPreflight 时不得透传选项")
}

// TestPortWriteHandler_BatchLockedReturns409 批量路径的写互斥 → 409。
func TestPortWriteHandler_BatchLockedReturns409(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)
	mockSvc.batchErr = fmt.Errorf("%w: held by %q (in-process)", portwrite.ErrPortWriteLocked, "other@tester")

	w := invokeWithCtx("BatchWrite", h.BatchWrite, portwrite.BatchWriteRequest{
		DeviceID: "dev-1", Action: portcollection.ActionShutdown, PortIDs: []string{"p1"},
	}, nil)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, http.StatusConflict, resp.Code)
	assert.Contains(t, resp.Message, "端口正在被其他写操作处理")

	var count int64
	sqlDB.Model(&models.PortWriteAudit{}).Count(&count)
	assert.Equal(t, int64(0), count)
	assert.Equal(t, 0, mockOperLog.recordAsyncCalls)
}

// TestPortWriteHandler_BatchDeviceUnreachableReturns503 批量路径的不可达 → 503。
func TestPortWriteHandler_BatchDeviceUnreachableReturns503(t *testing.T) {
	h, mockSvc, mockOperLog, sqlDB := newTestHandler(t)
	mockSvc.batchErr = fmt.Errorf("%w: device dev-1 ping timeout", portwrite.ErrDeviceUnreachable)

	w := invokeWithCtx("BatchWrite", h.BatchWrite, portwrite.BatchWriteRequest{
		DeviceID: "dev-1", Action: portcollection.ActionShutdown, PortIDs: []string{"p1"},
	}, nil)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)
	assert.Contains(t, resp.Message, "设备不可达")

	var count int64
	sqlDB.Model(&models.PortWriteAudit{}).Count(&count)
	assert.Equal(t, int64(0), count)
	assert.Equal(t, 0, mockOperLog.recordAsyncCalls)
}

// TestPortWriteHandler_BatchAuditTxFailureKeepsZeroAuditRows DEBT-02：批量审计
// 行单事务原子提交 —— 任一行 INSERT 失败则整体回滚（零审计行），但响应仍是 200
// （SSH 已生效，设备状态是真相源，审计缺失以 operlog + 日志兜底）。
func TestPortWriteHandler_BatchAuditTxFailureKeepsZeroAuditRows(t *testing.T) {
	mockSvc := &mockPortWriteService{}
	h, _, sqlDB := newTestHandlerWithService(t, mockSvc)

	mockSvc.batchResult = &portwrite.BatchResult{
		BatchID: "batch-1",
		Succeeded: []portwrite.PortResult{
			{PortID: "p1", Action: portcollection.ActionShutdown, Status: "succeeded"},
		},
		Failed:  []portwrite.PortResult{},
		Skipped: []portwrite.PortResult{},
	}
	// 用触发器让审计行 INSERT 必然失败（不删表 —— 删表后无法校验「零审计行」）
	require.NoError(t, sqlDB.Exec(`CREATE TRIGGER trg_block_audit_insert
		BEFORE INSERT ON sys_port_write_audit
		BEGIN SELECT RAISE(ABORT, 'audit insert blocked by test'); END;`).Error)

	w := invokeWithCtx("BatchWrite", h.BatchWrite, portwrite.BatchWriteRequest{
		DeviceID: "dev-1", Action: portcollection.ActionShutdown, PortIDs: []string{"p1"},
	}, nil)

	assert.Equal(t, http.StatusOK, w.Code, "审计事务失败不得把已生效的批量写变成失败响应")

	// 事务回滚 → 零审计行（不能出现部分写入的孤儿审计行）
	var count int64
	require.NoError(t, sqlDB.Model(&models.PortWriteAudit{}).Count(&count).Error)
	assert.Equal(t, int64(0), count, "事务失败必须零审计行落库")
}

// ===== 本文件私有 helper =====

// optsSpyService 记录 service 收到的 WriteOption（验证 skipPreflight 透传）。
type optsSpyService struct {
	*mockPortWriteService
	gotOpts []portwrite.WriteOption
}

func (s *optsSpyService) Shutdown(ctx context.Context, portID, operator string, opts ...portwrite.WriteOption) (*portwrite.PortResult, error) {
	s.gotOpts = opts
	return s.mockPortWriteService.Shutdown(ctx, portID, operator, opts...)
}

// newTestHandlerWithService 用指定 service 构造 handler（复用 newTestHandler 的
// core / sqlite 装配，区别只是 service 实例可注入 + 共享缓存内存库）。
//
// 为什么用共享缓存 DSN 而不是 :memory:：sqlite 的 :memory: 每个连接各自持一份
// 空库，连接池换连接就会「表不存在」（DROP TABLE 这类结构性操作不可靠）；
// 共享缓存让所有连接看到同一份库。
func newTestHandlerWithService(t *testing.T, svc portwrite.PortWriteService) (*PortWriteHandler, *mockOperLogService, *gorm.DB) {
	t.Helper()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared&_busy_timeout=5000", name, handlerDBSequence.Add(1))
	sqlDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, sqlDB.AutoMigrate(&models.PortWriteAudit{}, &models.DevicePortStatus{}, &models.NetworkDevice{}))

	mockOperLog := &mockOperLogService{}
	c := &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{DB: sqlDB, Type: "sqlite"}},
		CoreServices: &core.CoreServices{OperLogService: mockOperLog},
	}
	h := NewPortWriteHandler(svc).WithCore(c)
	return h, mockOperLog, sqlDB
}

// handlerDBSequence 保证每个用例的共享缓存内存库互不干扰。
var handlerDBSequence atomic.Int64
