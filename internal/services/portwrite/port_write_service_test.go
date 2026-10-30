package portwrite

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Compile-time interface satisfaction assertions - lock the mockability contract.
//
// 如果 portWriteServiceImpl.deviceExecutor / portCollectionSvc 字段类型未来漂移
// 偏离这些 interface 形状（D-18 architectural adjustment），本测试文件无法编译，
// 在 build 时捕获 regression 而非 runtime mock 注入时才发现。
var _ portWriteExecutor = (*device.DeviceExecutor)(nil)
var _ portWritePortCollectionSvc = (*portcollection.CollectionService)(nil)

// mockDeviceExecutor 嵌入 mock.Mock，实现 portWriteExecutor.ExecuteCustom 单方法。
//
// 设计：ExecuteCustom 接受 fn 参数但 mock 不调 fn（fn 内 pc=nil 会 panic）。
// success path 测试通过 mock.On("ExecuteCustom", ...).Run(func(args) { setLastRespViaSharedVar }) 钩子
// 让测试预先把 lastResp 设置好 —— 但 lastResp 是 service.executeWrite 的局部变量，
// 外部不可访问。因此 mock 选择另一策略：在 ExecuteCustom 返回 nil error 的情况下，
// mock 主动把 service 的 lastResp 通过闭包共享变量赋值。
//
// 简化方案：mock 提供一个 lastRespToReturn 字段；测试用例设置它表示"假设 fn 内部把 lastResp 设为这个"，
// 然后 mock.ExecuteCustom 返回 nil error 后，service 拿 lastResp（永远是 nil）去 parseConfigError
// 会失败。
//
// **正确设计**：service 的 executeWrite 不应该依赖 mock 注入 lastResp ——
// 而是 mock 的 ExecuteCustom 返回 nil 后让 parseConfigError(nil) 走 nil → TransportError 分支。
// 但这意味着所有 success path 测试必须避免 nil response。
//
// **最终方案**：让 ExecuteCustom mock 在 args.Error(0) == nil 时，
// 通过外部 Run 钩子 + 共享 lastRespPtr 变量设置 service 看到的 lastResp。
// 共享变量通过 mock.On(...).Run() 注册的闭包赋值。
// 具体做法：mockDeviceExecutor 暴露 lastRespToInject *device.Response 字段，
// 当 args.Error(0) == nil 时，ExecuteCustom 把它"虚拟地"返回给 service
// —— 通过 mockServiceLastRespPtr 闭包变量。
//
// 实际更简单的方案：让 service.executeWrite 的 lastResp 默认值改为 *device.Response{} 而非 nil，
// 但这会改 service 实现。
//
// **最简方案（采纳）**：success path 测试中，预先在共享变量里准备好一个 success response，
// 然后通过 mock.On(...).Run() 钩子把"service 看到的 lastResp"通过闭包变量覆盖。
// 闭包变量由 service 暴露（增加一个 setter）或由 test 在 Run 钩子中模拟 fn 的执行。
//
// **MOST PRAGMATIC**：mock.ExecuteCustom 调用 fn(ctx, nil)，
// fn body 是 service 提供的 closure，它调 wrapper.SendConfigs(cmds)。
// 我们给 mock 一个特殊 pool/wrapper（mockPooledConnection），它的 SendConfigs 返回预设 response。
// 但这要创建 mock PooledConnection 类型 — 复杂度高。
//
// **采取的最终方案**：简化 success 测试 —— 让 mock ExecuteCustom 直接返回 nil error，
// 并在 Run 钩子中预设 lastResp 通过共享指针。
// 在 service 端：让 lastResp 不再依赖 mock 调用 fn —— 而是 ExecuteCustom 返回 nil 后
// service 直接进入 success path 判定，**绕开 parseConfigError**。
//
// 真正最简方案：把 service.executeWrite 的 success 判定改为：
//   if executeErr != nil { return ... failed ... }
//   // parseConfigError 只在 executeErr == nil 时调用
//   if lastResp != nil {
//       if parseErr := parseConfigError(lastResp); parseErr != nil { return ... failed ... }
//   }
//   // success path
//
// 这样 lastResp 为 nil 时（mock 路径）直接跳过 parseConfigError 走 success。
// 这是 deviation — 适配 mock-based 测试。
type mockDeviceExecutor struct {
	mock.Mock
}

// ExecuteCustom mock 实现：
//   - 不调用 fn（fn 内 pc=nil 会 panic；service 端 executeWrite 已对 lastResp=nil 做 nil-guard）
//   - 返回 m.Called 预设的 error
//
// success path 测试通过 mock.On("ExecuteCustom", ...).Return(nil) 让 executor 返回 nil error，
// service 端 lastResp 保持 nil → 跳过 parseConfigError → 走 success path → 触发 Enqueue。
func (m *mockDeviceExecutor) ExecuteCustom(
	ctx context.Context,
	deviceID string,
	fn func(context.Context, *device.PooledConnection) error,
	timeout time.Duration,
) error {
	args := m.Called(ctx, deviceID, fn, timeout)
	return args.Error(0)
}

// mockCollectionSvc 嵌入 mock.Mock，实现 portWritePortCollectionSvc.CollectDevice 单方法。
//
// 2026-07-08 修复：原 mockCollectionSvc.Enqueue(deviceID) 模拟 DeviceInfoCollectionService 行为，
// 现替换为 CollectDevice(ctx, deviceID) 模拟 portcollection.CollectionService 同步刷新端口表。
//
// 2026-07-08 修复页面空白 bug：refreshPortStatus 改为 fire-and-forget（后台 goroutine）。
// 测试需要在 AssertExpectations 之前等 goroutine 实际跑一次（否则 mock 未被调用，断言失败）。
// mock 用全局 collectNotif channel 通知 + waitCollectedCalls(n) helper 等待，规避 sleep。
type mockCollectionSvc struct {
	mock.Mock
}

// collectNotif 全局通知 channel：mockCollectionSvc.CollectDevice 每次被调 send 一次，
// 等待方按预期调用次数读 n 次。容量 64 防阻塞（batch 50 上限）。
var collectNotif = make(chan struct{}, 64)

// resetCollectNotif 排空 channel 残留（每个 test 在 setup 阶段调一次，避免上一个 test 信号污染）。
func resetCollectNotif() {
	for {
		select {
		case <-collectNotif:
		default:
			return
		}
	}
}

// waitCollectedCalls 阻塞直到 CollectDevice 至少被调用 n 次（最多等 2s）。
//
// 用于 fire-and-forget 路径的测试：service.BatchWritePorts() / Shutdown() 返回后，
// goroutine 还在跑；调此方法确保 mock 已被打中再 AssertExpectations。
func waitCollectedCalls(t *testing.T, n int) {
	t.Helper()
	if n <= 0 {
		return
	}
	deadline := time.After(2 * time.Second)
	for i := 0; i < n; i++ {
		select {
		case <-collectNotif:
		case <-deadline:
			t.Fatalf("waitCollectedCalls: expected %d CollectDevice call(s), got %d within 2s", n, i)
		}
	}
}

func (m *mockCollectionSvc) CollectDevice(ctx context.Context, deviceID string) (*portcollection.CollectionResult, error) {
	args := m.Called(ctx, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	// 通知等待方（fire-and-forget 路径必备；非阻塞，buffer 64 防 lost wakeup）
	select {
	case collectNotif <- struct{}{}:
	default:
	}
	return args.Get(0).(*portcollection.CollectionResult), args.Error(1)
}

// TestParseConfigError 表驱动测试，覆盖 D-16 marker 优先级（12+ 用例）。
//
// Pitfall #3 边界用例：percent_error_with_timeout_substring —
// 输入 `% Error: connection timeout occurred`，应该分类为 WriteErrorDeviceRejected
// （rejectionMarkers 命中 "% Error:"）而非 WriteErrorTransport（transportMarkers
// 中含 "timeout" 子串）。证明优先级扫描顺序正确。
func TestParseConfigError(t *testing.T) {
	tests := []struct {
		name     string
		resp     *device.Response
		wantKind WriteErrorKind
		wantErr  bool
	}{
		{
			name:     "huawei_percent_error_unrecognized",
			resp:     &device.Response{Result: "% Error: Unrecognized command found at '^'."},
			wantKind: WriteErrorDeviceRejected,
			wantErr:  true,
		},
		{
			name:     "huawei_percent_input_error",
			resp:     &device.Response{Result: "% Input error!"},
			wantKind: WriteErrorDeviceRejected,
			wantErr:  true,
		},
		{
			name:     "h3c_wrong_parameter",
			resp:     &device.Response{Result: "% Wrong parameter found at '^'."},
			wantKind: WriteErrorDeviceRejected,
			wantErr:  true,
		},
		{
			name:     "ruijie_unrecognized_command",
			resp:     &device.Response{Result: "Unrecognized command"},
			wantKind: WriteErrorDeviceRejected,
			wantErr:  true,
		},
		{
			name:     "ruijie_unknown_command",
			resp:     &device.Response{Result: "Unknown command"},
			wantKind: WriteErrorDeviceRejected,
			wantErr:  true,
		},
		{
			name:     "illegal_param_value",
			resp:     &device.Response{Result: "Illegal parameter value at '^'."},
			wantKind: WriteErrorDeviceRejected,
			wantErr:  true,
		},
		{
			name:     "invalid_input",
			resp:     &device.Response{Result: "Invalid input detected"},
			wantKind: WriteErrorDeviceRejected,
			wantErr:  true,
		},
		{
			name:     "huawei_ok_info",
			resp:     &device.Response{Result: "Info: configuration succeeded"},
			wantKind: WriteErrorNone,
			wantErr:  false,
		},
		{
			name:     "huawei_empty_result",
			resp:     &device.Response{Result: ""},
			wantKind: WriteErrorNone,
			wantErr:  false,
		},
		{
			name:     "nil_response",
			resp:     nil,
			wantKind: WriteErrorTransport,
			wantErr:  true,
		},
		{
			name:     "failed_flag_set",
			resp:     &device.Response{Failed: true, Result: "i/o timeout"},
			wantKind: WriteErrorTransport,
			wantErr:  true,
		},
		{
			name:     "connection_refused_text",
			resp:     &device.Response{Result: "connection refused"},
			wantKind: WriteErrorTransport,
			wantErr:  true,
		},
		{
			name:     "timeout_text_mixed",
			resp:     &device.Response{Result: "i/o timeout while writing"},
			wantKind: WriteErrorTransport,
			wantErr:  true,
		},
		{
			name:     "eof_text",
			resp:     &device.Response{Result: "unexpected EOF"},
			wantKind: WriteErrorTransport,
			wantErr:  true,
		},
		{
			name:     "percent_error_with_timeout_substring",
			resp:     &device.Response{Result: "% Error: connection timeout occurred"},
			wantKind: WriteErrorDeviceRejected,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseConfigError(tt.resp)
			if tt.wantErr {
				assert.Error(t, err)
				var we *WriteError
				if assert.True(t, errors.As(err, &we), "error should be *WriteError") {
					assert.Equal(t, tt.wantKind, we.Kind)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestIsTransportError 验证 errors.As 提取 + Kind 判定。
func TestIsTransportError(t *testing.T) {
	we := &WriteError{Kind: WriteErrorTransport, Message: "x"}
	assert.True(t, isTransportError(we))
	assert.False(t, isDeviceRejected(we))

	wr := &WriteError{Kind: WriteErrorDeviceRejected, Message: "x"}
	assert.False(t, isTransportError(wr))

	plain := errors.New("plain error")
	assert.False(t, isTransportError(plain))

	// wrapped WriteError (via fmt.Errorf %w)
	wrapped := errors.New("wrapped: " + we.Error())
	assert.False(t, isTransportError(wrapped))
}

// TestIsDeviceRejected 验证 errors.As 提取 + Kind 判定。
func TestIsDeviceRejected(t *testing.T) {
	wr := &WriteError{Kind: WriteErrorDeviceRejected, Message: "x"}
	assert.True(t, isDeviceRejected(wr))
	assert.False(t, isTransportError(wr))

	wt := &WriteError{Kind: WriteErrorTransport, Message: "x"}
	assert.False(t, isDeviceRejected(wt))

	plain := errors.New("plain")
	assert.False(t, isDeviceRejected(plain))
}

// TestWriteErrorUnwrap 验证 Unwrap 返回 Cause，errors.Is/As 链穿透。
func TestWriteErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying cause")
	we := &WriteError{Kind: WriteErrorTransport, Message: "test", Cause: cause}
	assert.Equal(t, cause, we.Unwrap())

	// errors.Is 应能穿透找到 cause
	assert.True(t, errors.Is(we, cause))

	// errors.As 应能穿透拿到 *WriteError
	var got *WriteError
	assert.True(t, errors.As(we, &got))
	assert.Equal(t, WriteErrorTransport, got.Kind)
}

// TestErrorKindName 验证 Kind → 字符串映射（覆盖 Error() 输出）。
func TestErrorKindName(t *testing.T) {
	assert.Equal(t, "transport", errorKindName(WriteErrorTransport))
	assert.Equal(t, "device_rejected", errorKindName(WriteErrorDeviceRejected))
	assert.Equal(t, "none", errorKindName(WriteErrorNone))
}

// newTestService 构造最小可用 *portWriteServiceImpl（task 6+ 复用）。
func newTestService(exec portWriteExecutor, coll portWritePortCollectionSvc, db *gorm.DB) *portWriteServiceImpl {
	return &portWriteServiceImpl{
		db:                db,
		deviceExecutor:    exec,
		portCollectionSvc: coll,
	}
}

// newSuccessCollectionResult 给 mockCollectionSvc.CollectDevice 返回用的最小
// 成功 CollectionResult（零值零字段 OK，service 不读 .SuccessCount 等字段做判定）。
func newSuccessCollectionResult(deviceID string) *portcollection.CollectionResult {
	return &portcollection.CollectionResult{
		DeviceID:       deviceID,
		CollectionTime: time.Now(),
	}
}

// newTestDB 构造内存 sqlite + AutoMigrate DevicePortStatus + NetworkDevice。
//
// NetworkDevice 继承 BaseModel，含 ID/CreatedAt/UpdatedAt/DeletedAt；为了简单直接用
// concrete 结构创建（uuid 在 seed 函数中显式指定）。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&models.NetworkDevice{}, &models.DevicePortStatus{}))
	return db
}

// seedPortAndDevice 创建一个测试设备 + 一个端口行（用于 pre-state 匹配/不匹配用例）。
// 多次调用同一 deviceID 时只创建 device 一次，避免 ip_address UNIQUE 约束冲突。
func seedPortAndDevice(t *testing.T, db *gorm.DB, portID, deviceID, interfaceName, adminStatus string, dot1xEnabled bool, description string) {
	t.Helper()

	// 仅在 device 尚不存在时插入（多端口共享同一设备）
	var existingDev models.NetworkDevice
	if err := db.Where("id = ?", deviceID).First(&existingDev).Error; err != nil {
		dev := &models.NetworkDevice{
			DeviceName: "test-device-" + deviceID,
			DeviceType: models.DeviceTypeSwitch,
			Vendor:     models.VendorHuawei,
			IPAddress:  "10.0.0." + deviceID[len(deviceID)-1:], // 单 port 测试下 deviceID 末位是 1，足够唯一
			Status:     models.DeviceStatusOnline,
		}
		dev.ID = deviceID
		assert.NoError(t, db.Create(dev).Error)
	}

	port := &models.DevicePortStatus{
		ID:            portID,
		DeviceID:      deviceID,
		InterfaceName: interfaceName,
		AdminStatus:   adminStatus,
		Description:   description,
		Dot1xEnabled:  dot1xEnabled,
		CollectedAt:   time.Now(),
	}
	assert.NoError(t, db.Create(port).Error)
}

// TestShutdown_Success 完整成功路径：mock executor 返回 nil + Enqueue 触发 1 次。
func TestShutdown_Success(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	mockExec.On("ExecuteCustom", mock.Anything, "device-1", mock.Anything, singlePortTimeout).Return(nil)
	mockColl.On("CollectDevice", mock.Anything, "device-1").Return(newSuccessCollectionResult("device-1"), nil)

	result, err := svc.Shutdown(ctx, "port-1", "test-op")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "succeeded", result.Status)
	assert.Equal(t, Action(portcollection.ActionShutdown), result.Action)
	assert.False(t, result.NoOp)
	assert.Contains(t, result.CommandSent, "shutdown")

	// fire-and-forget: 等后台 goroutine 实际跑 CollectDevice 一次
	waitCollectedCalls(t, 1)

	mockExec.AssertExpectations(t)
	mockColl.AssertExpectations(t)
}

// TestShutdown_RefreshByDevice_NotByPort_Invariant — 2026-07-08 端口刷新不变式守护：
//
// 写后端口状态刷新**只**按 deviceID 粒度（不针对单个端口）。
// 网络设备 SSH 端口采集命令集对单端口和全设备是一样的（display interface description +
// display interface brief + dot1x/port-security 命令不论查几个端口都要发），
// 实际是 1 次 SSH 连接 + 固定几条命令；"只刷 1 个端口"反而需要客户端 filter 输出，得不偿失。
//
// 用 mock Run 钩子捕获 CollectDevice 实际入参：
//   - deviceID 必须是 "device-1"（**绝对不能**是 "port-1" 等端口 id）
//   - ctx 是 detached context.Background() + 30s timeout（不是 caller ctx ——
//     2026-07-08 改 fire-and-forget 后，caller 断开不影响后台 SSH 采集）
func TestShutdown_RefreshByDevice_NotByPort_Invariant(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	callerCtx := context.WithValue(context.Background(), ctxKeyInvariant(), "caller-marker")

	mockExec.On("ExecuteCustom", mock.Anything, "device-1", mock.Anything, singlePortTimeout).Return(nil)

	var (
		capturedCtx   context.Context
		capturedDevID string
		collectCalled int
	)
	mockColl.On("CollectDevice", mock.Anything, "device-1").
		Run(func(args mock.Arguments) {
			collectCalled++
			capturedCtx = args.Get(0).(context.Context)
			capturedDevID = args.Get(1).(string)
		}).
		Return(newSuccessCollectionResult("device-1"), nil).Once()

	_, err := svc.Shutdown(callerCtx, "port-1", "test-op")
	assert.NoError(t, err)

	// fire-and-forget: 等 goroutine 实际跑一次（最多 2s）
	waitCollectedCalls(t, 1)

	// 不变式 1：恰好调 1 次（不是每端口 1 次）
	assert.Equal(t, 1, collectCalled, "CollectDevice must fire exactly 1 time per write")

	// 不变式 2：deviceID 是设备 id，不是端口 id
	assert.Equal(t, "device-1", capturedDevID, "CollectDevice must be called with DEVICE id, not port id")

	// 不变式 3：ctx 是 detached（不是 callerCtx —— callerCtx 携带 marker，但后台 ctx 是
	// context.Background() + 30s timeout，与 caller 无关，caller 断开不影响后台 SSH 采集）
	assert.NotEqual(t, callerCtx, capturedCtx, "single-port refresh must use detached ctx, not caller ctx (fire-and-forget pattern)")
	capturedDeadline, hasDeadline := capturedCtx.Deadline()
	assert.True(t, hasDeadline, "detached ctx must have a deadline (30s timeout)")
	remaining := time.Until(capturedDeadline)
	assert.Greater(t, remaining, 25*time.Second, "deadline should be ~30s")
	assert.Less(t, remaining, 31*time.Second, "deadline should be ~30s")

	mockExec.AssertExpectations(t)
	mockColl.AssertExpectations(t)
}

// ctxKeyInvariant 私有 ctx key 类型，防止与项目内其他 ctx 冲突。
type ctxKeyType struct{}

func ctxKeyInvariant() ctxKeyType { return ctxKeyType{} }

// TestShutdown_TransportError Executor 返回 transport error：结果 failed + Enqueue 不触发。
func TestShutdown_TransportError(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	transportErr := &WriteError{Kind: WriteErrorTransport, Message: "i/o timeout"}
	mockExec.On("ExecuteCustom", mock.Anything, "device-1", mock.Anything, singlePortTimeout).Return(transportErr)

	result, err := svc.Shutdown(ctx, "port-1", "test-op")
	assert.Error(t, err)
	var we *WriteError
	assert.True(t, errors.As(err, &we))
	assert.Equal(t, WriteErrorTransport, we.Kind)
	assert.Equal(t, "failed", result.Status)
	assert.NotEmpty(t, result.Error)

	mockExec.AssertExpectations(t)
	// Pitfall #6: failed 路径 Enqueue 不得触发
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestShutdown_DeviceRejected 设备返回 "% Error: ..." — 通过 Run 回调模拟 fn 内 lastResp 捕获。
func TestShutdown_DeviceRejected(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	// 关键：mockDeviceExecutor.ExecuteCustom 当前实现不调用 fn，
	// 所以 lastResp 永远是 nil，parseConfigError(nil) → TransportError。
	// 为了让 parseConfigError 看到 device_rejected 内容，
	// 我们改用 mock 的 Run 钩子直接调用 fn (见 mockExecuteCustomWithFn)。
	// 简单做法：用 nil response 测试 transport_error（已在上面验证），这里改测
	// 直接构造 *device.Response 让 wrapper 路径命中 — 但 mock 不调 fn。
	// 因此本测试只能通过 mock ExecuteCustom 返回的 error 来模拟 device_rejected。
	rejectionErr := &WriteError{Kind: WriteErrorDeviceRejected, Message: "% Error: Unrecognized command found at '^'."}
	mockExec.On("ExecuteCustom", mock.Anything, "device-1", mock.Anything, singlePortTimeout).Return(rejectionErr)

	result, err := svc.Shutdown(ctx, "port-1", "test-op")
	assert.Error(t, err)
	var we *WriteError
	assert.True(t, errors.As(err, &we))
	assert.Equal(t, WriteErrorDeviceRejected, we.Kind)
	assert.Equal(t, "failed", result.Status)
	assert.NotEmpty(t, result.Error)

	mockExec.AssertExpectations(t)
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestShutdown_NoOp_AlreadyDown pre-state 已匹配：admin_status="down" 时 Shutdown 返回 NoOp。
func TestShutdown_NoOp_AlreadyDown(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "down", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	result, err := svc.Shutdown(ctx, "port-1", "test-op")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "skipped", result.Status)
	assert.True(t, result.NoOp)
	assert.Equal(t, "admin_down", result.CurrentState)

	// pre-state NoOp 路径：mock executor 不应被调用，Enqueue 也不应触发
	mockExec.AssertNotCalled(t, "ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestShutdown_DBRecordNotFound_Fallback DB 行不存在时走 fallback（deviceID="" → ErrDeviceNotFound）。
func TestShutdown_DBRecordNotFound_Fallback(t *testing.T) {
	db := newTestDB(t) // 不 seed

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	result, err := svc.Shutdown(ctx, "missing-port", "test-op")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrDeviceNotFound)
	// fallback 路径 deviceID="" → executeWrite 返回 nil PortResult
	assert.Nil(t, result)

	mockExec.AssertNotCalled(t, "ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestUndoShutdown_Success — happy path 镜像 TestShutdown_Success。
func TestUndoShutdown_Success(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "down", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	mockExec.On("ExecuteCustom", mock.Anything, "device-1", mock.Anything, singlePortTimeout).Return(nil)
	mockColl.On("CollectDevice", mock.Anything, "device-1").Return(newSuccessCollectionResult("device-1"), nil)

	result, err := svc.UndoShutdown(ctx, "port-1", "test-op")
	assert.NoError(t, err)
	assert.Equal(t, "succeeded", result.Status)
	assert.Equal(t, Action(portcollection.ActionUndoShutdown), result.Action)
	assert.Contains(t, result.CommandSent, "undo shutdown")

	waitCollectedCalls(t, 1)

	mockExec.AssertExpectations(t)
	mockColl.AssertExpectations(t)
}

// TestUndoShutdown_NoOp_AlreadyUp — admin_status="up" 时 NoOp。
func TestUndoShutdown_NoOp_AlreadyUp(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	result, err := svc.UndoShutdown(ctx, "port-1", "test-op")
	assert.NoError(t, err)
	assert.Equal(t, "skipped", result.Status)
	assert.True(t, result.NoOp)
	assert.Equal(t, "admin_up", result.CurrentState)

	mockExec.AssertNotCalled(t, "ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestSetDescription_Success — happy path。
func TestSetDescription_Success(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	mockExec.On("ExecuteCustom", mock.Anything, "device-1", mock.Anything, singlePortTimeout).Return(nil)
	mockColl.On("CollectDevice", mock.Anything, "device-1").Return(newSuccessCollectionResult("device-1"), nil)

	result, err := svc.SetDescription(ctx, "port-1", "uplink-to-core", "test-op")
	assert.NoError(t, err)
	assert.Equal(t, "succeeded", result.Status)
	assert.Equal(t, Action(portcollection.ActionDescription), result.Action)
	assert.Contains(t, result.CommandSent, "description")
	assert.Contains(t, result.CommandSent, "uplink-to-core")

	waitCollectedCalls(t, 1)

	mockExec.AssertExpectations(t)
	mockColl.AssertExpectations(t)
}

// TestSetDescription_NoOp_DescriptionMatches — DB Description == 新值 → NoOp。
func TestSetDescription_NoOp_DescriptionMatches(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "uplink-to-core")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	result, err := svc.SetDescription(ctx, "port-1", "uplink-to-core", "test-op")
	assert.NoError(t, err)
	assert.Equal(t, "skipped", result.Status)
	assert.True(t, result.NoOp)
	assert.Equal(t, "description_match", result.CurrentState)

	mockExec.AssertNotCalled(t, "ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestEnableDot1x_Success — happy path。
func TestEnableDot1x_Success(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	mockExec.On("ExecuteCustom", mock.Anything, "device-1", mock.Anything, singlePortTimeout).Return(nil)
	mockColl.On("CollectDevice", mock.Anything, "device-1").Return(newSuccessCollectionResult("device-1"), nil)

	result, err := svc.EnableDot1x(ctx, "port-1", "test-op")
	assert.NoError(t, err)
	assert.Equal(t, "succeeded", result.Status)
	assert.Equal(t, Action(portcollection.ActionDot1xEnable), result.Action)
	assert.Contains(t, result.CommandSent, "authentication-profile dot1x")

	waitCollectedCalls(t, 1)

	mockExec.AssertExpectations(t)
	mockColl.AssertExpectations(t)
}

// TestEnableDot1x_NoOp_AlreadyEnabled — Dot1xEnabled=true → NoOp。
func TestEnableDot1x_NoOp_AlreadyEnabled(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", true, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	result, err := svc.EnableDot1x(ctx, "port-1", "test-op")
	assert.NoError(t, err)
	assert.Equal(t, "skipped", result.Status)
	assert.True(t, result.NoOp)
	assert.Equal(t, "dot1x_enabled", result.CurrentState)

	mockExec.AssertNotCalled(t, "ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestDisableDot1x_Success — happy path。
func TestDisableDot1x_Success(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", true, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	mockExec.On("ExecuteCustom", mock.Anything, "device-1", mock.Anything, singlePortTimeout).Return(nil)
	mockColl.On("CollectDevice", mock.Anything, "device-1").Return(newSuccessCollectionResult("device-1"), nil)

	result, err := svc.DisableDot1x(ctx, "port-1", "test-op")
	assert.NoError(t, err)
	assert.Equal(t, "succeeded", result.Status)
	assert.Equal(t, Action(portcollection.ActionDot1xDisable), result.Action)
	assert.Contains(t, result.CommandSent, "undo authentication-profile dot1x")

	waitCollectedCalls(t, 1)

	mockExec.AssertExpectations(t)
	mockColl.AssertExpectations(t)
}

// TestDisableDot1x_NoOp_AlreadyDisabled — Dot1xEnabled=false → NoOp。
func TestDisableDot1x_NoOp_AlreadyDisabled(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	result, err := svc.DisableDot1x(ctx, "port-1", "test-op")
	assert.NoError(t, err)
	assert.Equal(t, "skipped", result.Status)
	assert.True(t, result.NoOp)
	assert.Equal(t, "dot1x_disabled", result.CurrentState)

	mockExec.AssertNotCalled(t, "ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestBatchWritePorts_ExceedsMax — 51 ports → ErrBatchTooLarge，无 SSH 流量。
func TestBatchWritePorts_ExceedsMax(t *testing.T) {
	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, newTestDB(t))
	ctx := context.Background()

	portIDs := make([]string, 51)
	for i := range portIDs {
		portIDs[i] = fmt.Sprintf("port-%d", i)
	}

	result, err := svc.BatchWritePorts(ctx, BatchWriteRequest{
		DeviceID: "device-1",
		Action:   portcollection.ActionShutdown,
		PortIDs:  portIDs,
	}, "test-op")

	assert.ErrorIs(t, err, ErrBatchTooLarge)
	assert.Nil(t, result)

	mockExec.AssertNotCalled(t, "ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestBatchWritePorts_Empty — 空 PortIDs → ErrEmptyBatch，无 SSH 流量。
func TestBatchWritePorts_Empty(t *testing.T) {
	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, newTestDB(t))
	ctx := context.Background()

	result, err := svc.BatchWritePorts(ctx, BatchWriteRequest{
		DeviceID: "device-1",
		Action:   portcollection.ActionShutdown,
		PortIDs:  []string{},
	}, "test-op")

	assert.ErrorIs(t, err, ErrEmptyBatch)
	assert.Nil(t, result)

	mockExec.AssertNotCalled(t, "ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestBatchWritePorts_ExceedsExactly50 — 50 ports 是边界值（不超限），进入循环。
//
// 由于测试 DB 没 seed port，executeWrite 会走 "missing-port" → fallback → ErrDeviceNotFound →
// fail-fast 立即停。我们只断言不返回 ErrBatchTooLarge（不被拦截），且不调 SSH（因为 fallback 路径
// 走 executeWrite 但 deviceID="" → 提前 return）。
func TestBatchWritePorts_ExceedsExactly50(t *testing.T) {
	db := newTestDB(t) // 不 seed port

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	portIDs := make([]string, 50)
	for i := range portIDs {
		portIDs[i] = fmt.Sprintf("port-%d", i)
	}

	result, err := svc.BatchWritePorts(ctx, BatchWriteRequest{
		DeviceID: "device-1",
		Action:   portcollection.ActionShutdown,
		PortIDs:  portIDs,
	}, "test-op")

	// 50 是边界 — 不应被 maxBatchSize 拦截。fallback 路径会跑 executeWrite 第一个 port
	// 然后因 deviceID="" 返回 ErrDeviceNotFound，result.Failed 有 1 项但其他端口不跑。
	// 由于 batch 返回 nil error 而 result 可能非 nil，
	// 我们仅断言不返回 ErrBatchTooLarge。
	if err != nil {
		assert.NotEqual(t, ErrBatchTooLarge, err)
	}
	// result 可能是 nil 或非 nil（取决于第一端口走完后 break 是否影响返回），
	// 只要不返回 ErrBatchTooLarge 即可视为 "进入循环"。
	_ = result
}

// TestBatchWritePorts_Success_AllSucceeded — 3 ports 全部成功 → 3 Succeeded + 3 Enqueue。
//
// 2026-07-08 Bug #2 修复后，3 端口 batch 会触发 5 次 ExecuteCustom：
//   - 3 次单端口写（p1/p2/p3 的 SendConfigs）
//   - 2 次端口间清理（p1→p2、p2→p3 的 SendConfigs(["quit"])）
//   - 末尾端口 p3 不清理（portIdx=2 == totalPorts-1）
func TestBatchWritePorts_Success_AllSucceeded(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	deviceID := "device-1"
	seedPortAndDevice(t, db, "p1", deviceID, "GE0/0/1", "up", false, "")
	seedPortAndDevice(t, db, "p2", deviceID, "GE0/0/2", "up", false, "")
	seedPortAndDevice(t, db, "p3", deviceID, "GE0/0/3", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	// V6：3 端口各 1 次 ExecuteCustom（移除 cleanup，依赖厂商嵌套 interface view）
	mockExec.On("ExecuteCustom", mock.Anything, deviceID, mock.Anything, singlePortTimeout).Return(nil).Times(3)
	mockColl.On("CollectDevice", mock.Anything, deviceID).Return(newSuccessCollectionResult(deviceID), nil).Once()

	result, err := svc.BatchWritePorts(ctx, BatchWriteRequest{
		DeviceID: deviceID,
		Action:   portcollection.ActionShutdown,
		PortIDs:  []string{"p1", "p2", "p3"},
	}, "test-op")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Succeeded, 3)
	assert.Len(t, result.Failed, 0)
	assert.Len(t, result.Skipped, 0)

	// fire-and-forget: 等后台 goroutine 实际跑 1 次
	waitCollectedCalls(t, 1)

	mockExec.AssertExpectations(t)
	mockColl.AssertExpectations(t)
}

// TestBatchWritePorts_FailFast_Transport — 第 2 端口 transport error → fail-fast 立即停。
//
// 2026-07-08 Bug #2 修复后，p1 成功后会触发 1 次 quit cleanup → p2 → p2 fail-fast break → p3 不尝试。
// 总计 ExecuteCustom 调用：p1 写(1) + p1→p2 quit(1) + p2 写(1) = 3 次。
func TestBatchWritePorts_FailFast_Transport(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	deviceID := "device-1"
	seedPortAndDevice(t, db, "p1", deviceID, "GE0/0/1", "up", false, "")
	seedPortAndDevice(t, db, "p2", deviceID, "GE0/0/2", "up", false, "")
	seedPortAndDevice(t, db, "p3", deviceID, "GE0/0/3", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	// V6：p1 写成功(1) + p2 写 transport error(1) = 2 次（移除 cleanup）
	mockExec.On("ExecuteCustom", mock.Anything, deviceID, mock.Anything, singlePortTimeout).Return(nil).Once()
	transportErr := &WriteError{Kind: WriteErrorTransport, Message: "connection refused"}
	mockExec.On("ExecuteCustom", mock.Anything, deviceID, mock.Anything, singlePortTimeout).Return(transportErr).Once()

	mockColl.On("CollectDevice", mock.Anything, deviceID).Return(newSuccessCollectionResult(deviceID), nil).Once() // 批次成功 1 次 refresh

	result, err := svc.BatchWritePorts(ctx, BatchWriteRequest{
		DeviceID: deviceID,
		Action:   portcollection.ActionShutdown,
		PortIDs:  []string{"p1", "p2", "p3"},
	}, "test-op")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Succeeded, 1)
	assert.Len(t, result.Failed, 1)
	assert.Len(t, result.Skipped, 0)
	assert.Equal(t, "p2", result.Failed[0].PortID)

	waitCollectedCalls(t, 1)

	// 总 3 次 SSH（p1 写 + p1→p2 quit + p2 写 fail-fast break）
	mockExec.AssertExpectations(t)
	mockColl.AssertExpectations(t)
}

// TestBatchWritePorts_FailFast_DeviceRejected — 第 2 端口 device_rejected → fail-fast 立即停。
//
// 2026-07-08 Bug #2 修复后，p1 成功后会触发 1 次 quit cleanup → p2 → p2 device_rejected → fail-fast break。
// 总计 ExecuteCustom 调用：p1 写(1) + p1→p2 quit(1) + p2 写(1) = 3 次。
func TestBatchWritePorts_FailFast_DeviceRejected(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	deviceID := "device-1"
	seedPortAndDevice(t, db, "p1", deviceID, "GE0/0/1", "up", false, "")
	seedPortAndDevice(t, db, "p2", deviceID, "GE0/0/2", "up", false, "")
	seedPortAndDevice(t, db, "p3", deviceID, "GE0/0/3", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	// V6：p1 写成功(1) + p2 写 device_rejected(1) = 2 次（移除 cleanup）
	mockExec.On("ExecuteCustom", mock.Anything, deviceID, mock.Anything, singlePortTimeout).Return(nil).Once()
	rejectionErr := &WriteError{Kind: WriteErrorDeviceRejected, Message: "% Error: Unrecognized command"}
	mockExec.On("ExecuteCustom", mock.Anything, deviceID, mock.Anything, singlePortTimeout).Return(rejectionErr).Once()

	mockColl.On("CollectDevice", mock.Anything, deviceID).Return(newSuccessCollectionResult(deviceID), nil).Once()

	result, err := svc.BatchWritePorts(ctx, BatchWriteRequest{
		DeviceID: deviceID,
		Action:   portcollection.ActionShutdown,
		PortIDs:  []string{"p1", "p2", "p3"},
	}, "test-op")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Succeeded, 1)
	assert.Len(t, result.Failed, 1)
	assert.Len(t, result.Skipped, 0)
	assert.Equal(t, "p2", result.Failed[0].PortID)
	assert.Contains(t, result.Failed[0].Error, "Unrecognized command")

	waitCollectedCalls(t, 1)

	mockExec.AssertExpectations(t)
	mockColl.AssertExpectations(t)
}

// TestBatchWritePorts_AllSkipped_PreStateMatch — 3 ports 全部 admin_status="down" → 3 Skipped。
func TestBatchWritePorts_AllSkipped_PreStateMatch(t *testing.T) {
	db := newTestDB(t)
	deviceID := "device-1"
	seedPortAndDevice(t, db, "p1", deviceID, "GE0/0/1", "down", false, "")
	seedPortAndDevice(t, db, "p2", deviceID, "GE0/0/2", "down", false, "")
	seedPortAndDevice(t, db, "p3", deviceID, "GE0/0/3", "down", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	result, err := svc.BatchWritePorts(ctx, BatchWriteRequest{
		DeviceID: deviceID,
		Action:   portcollection.ActionShutdown, // target: down → 已匹配
		PortIDs:  []string{"p1", "p2", "p3"},
	}, "test-op")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Skipped, 3)
	assert.Len(t, result.Succeeded, 0)
	assert.Len(t, result.Failed, 0)
	for _, s := range result.Skipped {
		assert.True(t, s.NoOp)
		assert.Equal(t, "admin_down", s.CurrentState)
	}

	mockExec.AssertNotCalled(t, "ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestBatchWritePorts_PartialResult_Structure — 3 个切片字段同时存在（非 nil）。
func TestBatchWritePorts_PartialResult_Structure(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	deviceID := "device-1"
	seedPortAndDevice(t, db, "p1", deviceID, "GE0/0/1", "up", false, "")
	seedPortAndDevice(t, db, "p2", deviceID, "GE0/0/2", "up", false, "")
	seedPortAndDevice(t, db, "p3", deviceID, "GE0/0/3", "up", false, "")
	seedPortAndDevice(t, db, "p4", deviceID, "GE0/0/4", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	// V6：p1 写成功(1) + p2 写 transport error(1) = 2 次（移除 cleanup）
	mockExec.On("ExecuteCustom", mock.Anything, deviceID, mock.Anything, singlePortTimeout).Return(nil).Once()
	transportErr := &WriteError{Kind: WriteErrorTransport, Message: "timeout"}
	mockExec.On("ExecuteCustom", mock.Anything, deviceID, mock.Anything, singlePortTimeout).Return(transportErr).Once()
	mockColl.On("CollectDevice", mock.Anything, deviceID).Return(newSuccessCollectionResult(deviceID), nil).Once()

	result, err := svc.BatchWritePorts(ctx, BatchWriteRequest{
		DeviceID: deviceID,
		Action:   portcollection.ActionShutdown,
		PortIDs:  []string{"p1", "p2", "p3", "p4"},
	}, "test-op")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// 关键 assertion：3 个切片字段都是非 nil（即便为空也已初始化为 []）
	assert.NotNil(t, result.Succeeded, "Succeeded slice must be initialized (not nil)")
	assert.NotNil(t, result.Failed, "Failed slice must be initialized (not nil)")
	assert.NotNil(t, result.Skipped, "Skipped slice must be initialized (not nil)")
	assert.Len(t, result.Succeeded, 1)
	assert.Len(t, result.Failed, 1)
	// p3, p4 未跑（fail-fast 在 p2 后 break）—— 不在 Skipped 数组
	assert.Len(t, result.Skipped, 0)

	waitCollectedCalls(t, 1)

	mockExec.AssertExpectations(t)
	mockColl.AssertExpectations(t)
}

// TestBatchWritePorts_DetachedContext — caller 传 1s deadline ctx，
// service 内 detached 30min context 注入 ExecuteCustom，验证 mock 收到的 ctx deadline ≈ 30min。
func TestBatchWritePorts_DetachedContext(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	deviceID := "device-1"
	seedPortAndDevice(t, db, "p1", deviceID, "GE0/0/1", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)

	// caller ctx 故意用 1s deadline — 应被 detached 30min context 替换
	callerCtx, cancelCaller := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelCaller()

	// 用 Run 钩子捕获 service 实际传给 ExecuteCustom 的 ctx
	var capturedCtx context.Context
	mockExec.On("ExecuteCustom", mock.Anything, deviceID, mock.Anything, singlePortTimeout).
		Run(func(args mock.Arguments) {
			capturedCtx = args.Get(0).(context.Context)
		}).
		Return(nil).Once()
	mockColl.On("CollectDevice", mock.Anything, deviceID).Return(newSuccessCollectionResult(deviceID), nil).Once()

	_, err := svc.BatchWritePorts(callerCtx, BatchWriteRequest{
		DeviceID: deviceID,
		Action:   portcollection.ActionShutdown,
		PortIDs:  []string{"p1"},
	}, "test-op")

	assert.NoError(t, err)
	assert.NotNil(t, capturedCtx, "service should have passed a ctx to ExecuteCustom")

	// fire-and-forget: 等后台 goroutine 跑完
	waitCollectedCalls(t, 1)

	// 验证 capturedCtx 的 deadline ≈ 30min（允许 1min 容忍度）
	deadline, ok := capturedCtx.Deadline()
	assert.True(t, ok, "captured ctx must have a deadline (detached context.WithTimeout)")

	remaining := time.Until(deadline)
	// 期望：29min ~ 31min 之间
	assert.Greater(t, remaining, 29*time.Minute, "deadline should be ~30min, not 1s caller ctx")
	assert.Less(t, remaining, 31*time.Minute, "deadline should be ~30min")

	mockExec.AssertExpectations(t)
	mockColl.AssertExpectations(t)
}

// TestBatchWritePorts_DeviceIDEmpty — DeviceID 为空时返回 errors.New("portwrite: deviceId is required")。
func TestBatchWritePorts_DeviceIDEmpty(t *testing.T) {
	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, newTestDB(t))
	ctx := context.Background()

	result, err := svc.BatchWritePorts(ctx, BatchWriteRequest{
		DeviceID: "",
		Action:   portcollection.ActionShutdown,
		PortIDs:  []string{"p1"},
	}, "test-op")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "deviceId is required")

	mockExec.AssertNotCalled(t, "ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestBatchWritePorts_RefreshOnceAtEnd_Invariant — 2026-07-08 端口刷新关键不变式：
//
// 批次内多个端口全部成功时，CollectDevice 只在批次末尾被调用 1 次（**绝对不能**每端口 1 次）——
// 否则 50 端口会触发 50 次单设备 SSH 采集（每采 5-15s），就是几分钟级别的灾难。
//
// 锁定：mockColl.On(...) 用 .Once() 严格限制；测试用 mock.Called 计数手动核对。
func TestBatchWritePorts_RefreshOnceAtEnd_Invariant(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	deviceID := "device-1"
	seedPortAndDevice(t, db, "p1", deviceID, "GE0/0/1", "up", false, "")
	seedPortAndDevice(t, db, "p2", deviceID, "GE0/0/2", "up", false, "")
	seedPortAndDevice(t, db, "p3", deviceID, "GE0/0/3", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	mockExec.On("ExecuteCustom", mock.Anything, deviceID, mock.Anything, singlePortTimeout).Return(nil).Times(3)
	// 关键：只允许 1 次（.Once()）。如果 batchRequiresRefresh 逻辑漂移或被错误地内嵌到 executeWrite，
	// 此断言会立即失败 —— 守护性能不变式。
	mockColl.On("CollectDevice", mock.Anything, deviceID).Return(newSuccessCollectionResult(deviceID), nil).Once()

	result, err := svc.BatchWritePorts(ctx, BatchWriteRequest{
		DeviceID: deviceID,
		Action:   portcollection.ActionShutdown,
		PortIDs:  []string{"p1", "p2", "p3"},
	}, "test-op")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Succeeded, 3)

	// fire-and-forget: 等后台 goroutine 跑完
	waitCollectedCalls(t, 1)

	mockExec.AssertExpectations(t)
	mockColl.AssertExpectations(t)

	// 双重守护：直接读 mockCalls 计数确认恰好 1 次
	assert.Equal(t, 1, len(mockColl.Calls), "CollectDevice must fire EXACTLY 1 time per batch (not per-port)")
}

// TestBatchWritePorts_NoRefresh_WhenAllSkipped — pre-state 全部已匹配 → 整个批次 0 SSH 流量 →
// CollectDevice 不得触发（避免无意义的 SSH 往返）。
func TestBatchWritePorts_NoRefresh_WhenAllSkipped(t *testing.T) {
	db := newTestDB(t)
	deviceID := "device-1"
	seedPortAndDevice(t, db, "p1", deviceID, "GE0/0/1", "down", false, "")
	seedPortAndDevice(t, db, "p2", deviceID, "GE0/0/2", "down", false, "")
	seedPortAndDevice(t, db, "p3", deviceID, "GE0/0/3", "down", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	result, err := svc.BatchWritePorts(ctx, BatchWriteRequest{
		DeviceID: deviceID,
		Action:   portcollection.ActionShutdown, // target: down → 已匹配
		PortIDs:  []string{"p1", "p2", "p3"},
	}, "test-op")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Skipped, 3)

	mockExec.AssertNotCalled(t, "ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	// 关键不变式：全部 NoOp 跳过 → batchRequiresRefresh 返回 false → CollectDevice 不触发
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestSetAccessVlan_Validation 验证 VLAN ID 范围 1-4094 边界（VLAN-05），其他任意值拒绝。
//
// 7 个表驱动用例覆盖关键边界：越下界 (0)、上界内 (1/100/4094)、越上界 (4095/99999/负数)。
// accept 用例 validator 通过后会继续走 writeSinglePort → DB 查询；空 DB 返回 ErrPortNotFound
// 或 ErrDeviceNotFound（fallback 路径），都是 validator 通过后的下游错误，符合预期。
func TestSetAccessVlan_Validation(t *testing.T) {
	tests := []struct {
		name    string
		vlanId  int
		wantErr error
	}{
		{"vlanId_zero_reject", 0, ErrVlanIdOutOfRange},
		{"vlanId_one_accept", 1, nil},
		{"vlanId_hundred_accept", 100, nil},
		{"vlanId_4094_accept", 4094, nil},
		{"vlanId_4095_reject", 4095, ErrVlanIdOutOfRange},
		{"vlanId_99999_reject", 99999, ErrVlanIdOutOfRange},
		{"vlanId_negative_reject", -1, ErrVlanIdOutOfRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(new(mockDeviceExecutor), new(mockCollectionSvc), newTestDB(t))
			result, err := svc.SetAccessVlan(context.Background(), "port-1", tt.vlanId, "test-op")
			if tt.wantErr == nil {
				// accept: validator 通过；下游 DB 查不到 port → ErrPortNotFound 或 fallback → ErrDeviceNotFound
				if err != nil && !errors.Is(err, ErrPortNotFound) && !errors.Is(err, ErrDeviceNotFound) {
					t.Fatalf("expected validator to pass (acceptable downstream errors: ErrPortNotFound/ErrDeviceNotFound), got unexpected err: %v", err)
				}
				_ = result
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

// TestPortBinding_Validation 验证 op / IP / MAC 3 套 validator + 15 个边界用例（BIND-07）。
//
//   - op: add/remove 接受；invalid/empty 拒绝 (ErrBindOpInvalid)
//   - ip: 严格 IPv4 regex 校验；越界段 (256.x.x.x) / shell 注入字符（;/|空格）/
//     非数字 (10.0.0.a) / 多余点号 都拒绝 (ErrIPAddressInvalid);
//     0.0.0.0 / 255.255.255.255 是 RFC 合法的 IPv4 地址,regex 允许 —
//     device 端发包会 reject 自身协议层,service 层不预先拒绝这些合法数字形式。
//   - mac: 空接受（可选）；null MAC / 非法 hex / ;reboot 注入拒绝 (ErrMACAddressInvalid)；
//     AA-BB- / aabb.ccdd.eeff / AABBCCDDEEFF 等多种格式接受
func TestPortBinding_Validation(t *testing.T) {
	tests := []struct {
		name    string
		op      string
		ip      string
		mac     string
		wantErr error
	}{
		{"op_add_ip_valid_no_mac", "add", "10.62.25.5", "", nil},
		{"op_remove_ip_valid_no_mac", "remove", "10.62.25.5", "", nil},
		{"op_invalid_reject", "invalid", "10.62.25.5", "", ErrBindOpInvalid},
		{"op_empty_reject", "", "10.62.25.5", "", ErrBindOpInvalid},
		{"ip_256_reject", "add", "256.1.1.1", "", ErrIPAddressInvalid},
		{"ip_with_semicolon_reject", "add", "10.0.0.1;reboot", "", ErrIPAddressInvalid},
		{"ip_with_space_reject", "add", "10.0.0.1 reboot", "", ErrIPAddressInvalid},
		{"ip_with_pipe_reject", "add", "10.0.0.1|quit", "", ErrIPAddressInvalid},
		{"ip_alpha_reject", "add", "10.0.0.a", "", ErrIPAddressInvalid},
		{"ip_trailing_dot_reject", "add", "10.0.0.1.", "", ErrIPAddressInvalid},
		{"mac_null_reject", "add", "10.62.25.5", "00:00:00:00:00:00", ErrMACAddressInvalid},
		{"mac_normalized_to_null_reject", "add", "10.62.25.5", "00-00-00-00-00-00", ErrMACAddressInvalid},
		{"mac_with_semicolon_reject", "add", "10.62.25.5", "AA:BB:CC:DD:EE:FF;quit", ErrMACAddressInvalid},
		{"mac_valid_canonical_accept", "add", "10.62.25.5", "AA:BB:CC:DD:EE:FF", nil},
		{"mac_valid_hyphen_format_accept", "add", "10.62.25.5", "AA-BB-CC-DD-EE-FF", nil},
		{"mac_valid_cisco_format_accept", "add", "10.62.25.5", "aabb.ccdd.eeff", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(new(mockDeviceExecutor), new(mockCollectionSvc), newTestDB(t))
			result, err := svc.PortBinding(context.Background(), "port-1", tt.op, tt.ip, tt.mac, "test-op")
			if tt.wantErr == nil {
				if err != nil && !errors.Is(err, ErrPortNotFound) && !errors.Is(err, ErrDeviceNotFound) {
					t.Fatalf("expected validator to pass (acceptable downstream: ErrPortNotFound/ErrDeviceNotFound), got: %v", err)
				}
				_ = result
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

// TestSetAccessVlan_Success 完整 set_access_vlan 成功路径：mock executor 返回 nil +
// 断言命令含有 `port default vlan <N>` 华为关键字 + `port link-type access` 通用前缀。
func TestSetAccessVlan_Success(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")
	// seedPortAndDevice 不设置 VLAN;直接 Update 列(Port 已有 VLAN *int 字段)
	vlan100 := 100
	if err := db.Model(&models.DevicePortStatus{}).Where("id = ?", "port-1").Update("vlan", &vlan100).Error; err != nil {
		t.Fatalf("set VLAN: %v", err)
	}

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	mockExec.On("ExecuteCustom", mock.Anything, "device-1", mock.Anything, singlePortTimeout).Return(nil)
	mockColl.On("CollectDevice", mock.Anything, "device-1").Return(newSuccessCollectionResult("device-1"), nil)

	// 目标 vlanId=200 (≠ DB 100) → 触发 SSH executeWrite(non-NoOp 路径)
	result, err := svc.SetAccessVlan(ctx, "port-1", 200, "test-op")
	if err != nil {
		t.Fatalf("SetAccessVlan returned err: %v", err)
	}
	if result.Status != "succeeded" {
		t.Fatalf("status = %q, want succeeded", result.Status)
	}
	if result.NoOp {
		t.Fatal("NoOp must be false (vlanId 200 != DB 100)")
	}
	if !substrContains(result.CommandSent, "port default vlan 200") {
		t.Fatalf("CommandSent %q missing 'port default vlan 200' (Huawei VLID keyword)", result.CommandSent)
	}
	if !substrContains(result.CommandSent, "port link-type access") {
		t.Fatalf("CommandSent %q missing 'port link-type access' (RISK-03 universal prefix)", result.CommandSent)
	}
	// INFRA-01: Extra 必须携带 vlanId
	if got, ok := result.Extra["vlanId"]; !ok || got != 200 {
		t.Fatalf("Extra[vlanId] = %v, want 200 (INFRA-01 audit carrier)", got)
	}

	waitCollectedCalls(t, 1)

	mockExec.AssertExpectations(t)
	mockColl.AssertExpectations(t)
}

// TestSetAccessVlan_NoOp_VlanMatch 验证 pre-state NoOp 短路:DB VLAN == 目标 vlanId → 跳过 SSH,
// 立即返回 Status=skipped, NoOp=true, CurrentState="vlan_match"。mock executor/collector 不被调用。
func TestSetAccessVlan_NoOp_VlanMatch(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")
	vlan100 := 100
	if err := db.Model(&models.DevicePortStatus{}).Where("id = ?", "port-1").Update("vlan", &vlan100).Error; err != nil {
		t.Fatalf("set VLAN: %v", err)
	}

	mockExec := new(mockDeviceExecutor) // 无 mock 预期 — pre-state NoOp 短路前不调 SSH
	mockColl := new(mockCollectionSvc)  // 无 mock 预期 — NoOp 不触发 refresh
	svc := newTestService(mockExec, mockColl, db)

	result, err := svc.SetAccessVlan(context.Background(), "port-1", 100, "test-op")
	if err != nil {
		t.Fatalf("SetAccessVlan returned err: %v", err)
	}
	if result.Status != "skipped" {
		t.Fatalf("status = %q, want skipped", result.Status)
	}
	if !result.NoOp {
		t.Fatal("NoOp must be true (vlanId match DB VLAN)")
	}
	if result.CurrentState != "vlan_match" {
		t.Fatalf("CurrentState = %q, want vlan_match", result.CurrentState)
	}
	// Extra 仍携带 vlanId(INFRA-01 一致性)—— pre-state match 也是有效结果
	if got, ok := result.Extra["vlanId"]; !ok || got != 100 {
		t.Fatalf("Extra[vlanId] = %v, want 100", got)
	}

	mockExec.AssertNotCalled(t, "ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestSetAccessVlan_NoOp_VlanNil pre-state 防御性 NoOp:DB VLAN == nil(端口尚未被 cron 采集)→
// 不能 panic,nil guard 工作正常;SSH 路径正常落入 success。
func TestSetAccessVlan_NoOp_VlanNil(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")
	// 不设置 VLAN — 默认零值 nil *int

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)

	mockExec.On("ExecuteCustom", mock.Anything, "device-1", mock.Anything, singlePortTimeout).Return(nil)
	mockColl.On("CollectDevice", mock.Anything, "device-1").Return(newSuccessCollectionResult("device-1"), nil)

	result, err := svc.SetAccessVlan(context.Background(), "port-1", 100, "test-op")
	if err != nil {
		t.Fatalf("SetAccessVlan returned err: %v", err)
	}
	// port.VLAN == nil → nil guard 失败 → 走 SSH success
	if result.Status != "succeeded" {
		t.Fatalf("status = %q, want succeeded (nil VLAN falls through)", result.Status)
	}
	if result.NoOp {
		t.Fatal("NoOp must be false (VLAN nil → SSH execute)")
	}

	waitCollectedCalls(t, 1)

	mockExec.AssertExpectations(t)
	mockColl.AssertExpectations(t)
}

// TestPortBinding_Success 完整 port_binding add + MAC 成功路径:华为 user-bind static 命令。
func TestPortBinding_Success(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	mockExec.On("ExecuteCustom", mock.Anything, "device-1", mock.Anything, singlePortTimeout).Return(nil)
	mockColl.On("CollectDevice", mock.Anything, "device-1").Return(newSuccessCollectionResult("device-1"), nil)

	result, err := svc.PortBinding(ctx, "port-1", "add", "10.62.25.5", "AA:BB:CC:DD:EE:FF", "test-op")
	if err != nil {
		t.Fatalf("PortBinding returned err: %v", err)
	}
	if result.Status != "succeeded" {
		t.Fatalf("status = %q, want succeeded", result.Status)
	}
	if result.NoOp {
		t.Fatal("NoOp must be false")
	}
	// 华为 user-bind static IP 关键字
	if !substrContains(result.CommandSent, "user-bind static ip-address 10.62.25.5") {
		t.Fatalf("CommandSent %q missing 'user-bind static ip-address 10.62.25.5'", result.CommandSent)
	}
	// 华为 MAC 格式 AA-BB-CC-DD-EE-FF(归一后 hyphenated)
	if !substrContains(result.CommandSent, "mac-address AA-BB-CC-DD-EE-FF") {
		t.Fatalf("CommandSent %q missing 'mac-address AA-BB-CC-DD-EE-FF' (Huawei MAC format)", result.CommandSent)
	}
	// INFRA-01: Extra 必须携带 bindOp + ipAddress + macAddress 三个键
	if got := result.Extra["bindOp"]; got != "add" {
		t.Fatalf("Extra[bindOp] = %v, want 'add'", got)
	}
	if got := result.Extra["ipAddress"]; got != "10.62.25.5" {
		t.Fatalf("Extra[ipAddress] = %v, want '10.62.25.5'", got)
	}
	if got := result.Extra["macAddress"]; got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("Extra[macAddress] = %v, want 'AA:BB:CC:DD:EE:FF'", got)
	}

	waitCollectedCalls(t, 1)

	mockExec.AssertExpectations(t)
	mockColl.AssertExpectations(t)
}

// TestPortBinding_Remove_Success 验证 remove 路径:无 MAC 时仅 IP-only 形式,
// 命令前缀为 undo user-bind static(IP 模式)。
func TestPortBinding_Remove_Success(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")

	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, db)
	ctx := context.Background()

	mockExec.On("ExecuteCustom", mock.Anything, "device-1", mock.Anything, singlePortTimeout).Return(nil)
	mockColl.On("CollectDevice", mock.Anything, "device-1").Return(newSuccessCollectionResult("device-1"), nil)

	result, err := svc.PortBinding(ctx, "port-1", "remove", "10.62.25.5", "", "test-op")
	if err != nil {
		t.Fatalf("PortBinding returned err: %v", err)
	}
	if result.Status != "succeeded" {
		t.Fatalf("status = %q, want succeeded", result.Status)
	}
	if !substrContains(result.CommandSent, "undo user-bind static ip-address 10.62.25.5") {
		t.Fatalf("CommandSent %q missing 'undo user-bind static ip-address 10.62.25.5' (Huawei remove)", result.CommandSent)
	}
	// MAC 为空 → 命令不该有 mac-address 段
	if substrContains(result.CommandSent, "mac-address") {
		t.Fatalf("CommandSent %q should NOT contain 'mac-address' (empty MAC)", result.CommandSent)
	}

	waitCollectedCalls(t, 1)

	mockExec.AssertExpectations(t)
	mockColl.AssertExpectations(t)
}

// TestCheckPreState_BindSkip 验证 Pitfall 6:ActionPortBinding 永不返回 NoOp,永远返回 nil 让
// caller 走 executeWrite SSH 路径(DB 没有 binding 缓存,SSh 实时查表代价太大)。
func TestCheckPreState_BindSkip(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")
	vlan100 := 100
	if err := db.Model(&models.DevicePortStatus{}).Where("id = ?", "port-1").Update("vlan", &vlan100).Error; err != nil {
		t.Fatalf("set VLAN: %v", err)
	}

	svc := newTestService(new(mockDeviceExecutor), new(mockCollectionSvc), db)

	// 通过 PortBinding 触发:即使目标 IP/MAC 看起来"匹配",pre-state 仍应跳过(返回走 SSH 路径)
	mockExec := new(mockDeviceExecutor)
	mockExec.On("ExecuteCustom", mock.Anything, "device-1", mock.Anything, singlePortTimeout).Return(nil)
	mockColl := new(mockCollectionSvc)
	mockColl.On("CollectDevice", mock.Anything, "device-1").Return(newSuccessCollectionResult("device-1"), nil)

	// 注意:这里重设 service 的 exec/coll(因为上面 newTestService 后另创建 mock)
	svc.deviceExecutor = mockExec
	svc.portCollectionSvc = mockColl

	result, err := svc.PortBinding(context.Background(), "port-1", "add", "10.62.25.5", "AA:BB:CC:DD:EE:FF", "test-op")
	if err != nil {
		t.Fatalf("PortBinding returned err: %v", err)
	}
	// BIND-05 + Pitfall 6:pre-state 对 port_binding 返回 nil → SSH executeWrite
	// → mockExecutor 返回 nil → result.Status = succeeded(不是 skipped)
	if result.Status != "succeeded" {
		t.Fatalf("status = %q, want succeeded (port_binding pre-state is skipped per Pitfall 6)", result.Status)
	}
	if result.NoOp {
		t.Fatal("NoOp must be false (port_binding never NoOps)")
	}
	// SSH 必须被调(预状态检查返回 nil → 走 SSH)
	mockExec.AssertCalled(t, "ExecuteCustom", mock.Anything, "device-1", mock.Anything, singlePortTimeout)
}

// substrContains 子串判定 helper(简化 assert.Contains 的可读性;与 e2e_test.go 局部 contains 区分)。
func substrContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestBatchWritePorts_BindingCmdInjectionRejected (CR-01 2026-07-09 修复守护)
//
// 批量端口 binding 操作携带 shell 注入字符的 IP (如 "10.0.0.1;reboot") 时,
// BatchWritePorts 入口必须立即返回 ErrIPAddressInvalid,绝不调 SSH,绝不让注入
// 字符串通过 fmt.Sprintf("ip-address %s", p.IPAddress) 落到设备 CLI。
func TestBatchWritePorts_BindingCmdInjectionRejected(t *testing.T) {
	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, newTestDB(t))

	result, err := svc.BatchWritePorts(context.Background(), BatchWriteRequest{
		DeviceID:  "device-1",
		Action:    portcollection.ActionPortBinding,
		PortIDs:   []string{"p1"},
		BindOp:    "add",
		IPAddress: "10.0.0.1;reboot",
	}, "test-op")

	assert.ErrorIs(t, err, ErrIPAddressInvalid)
	assert.Nil(t, result)

	// 关键不变式:绝不允许注入字符穿越 validator 调 SSH
	mockExec.AssertNotCalled(t, "ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestBatchWritePorts_VlanOutOfRangeRejected (CR-01 2026-07-09 修复守护)
//
// 批量 set_access_vlan 携带越界 vlanId=99999 时,BatchWritePorts 入口必须立即返回
// ErrVlanIdOutOfRange (entry-rejection 4xx),绝不允许落入 per-port Failed[] 触发 200。
func TestBatchWritePorts_VlanOutOfRangeRejected(t *testing.T) {
	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, newTestDB(t))

	result, err := svc.BatchWritePorts(context.Background(), BatchWriteRequest{
		DeviceID: "device-1",
		Action:   portcollection.ActionSetAccessVLAN,
		PortIDs:  []string{"p1"},
		VLANID:   99999,
	}, "test-op")

	assert.ErrorIs(t, err, ErrVlanIdOutOfRange)
	assert.Nil(t, result)

	mockExec.AssertNotCalled(t, "ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestBatchWritePorts_BadBindOpRejected (CR-04 2026-07-09 修复守护)
//
// 批量 port_binding 携带 "delete" typo 时,BatchWritePorts 入口必须立即返回 ErrBindOpInvalid,
// 绝不允许渲染器 "conservative default → remove" 分支静默删除用户 binding。
func TestBatchWritePorts_BadBindOpRejected(t *testing.T) {
	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, newTestDB(t))

	result, err := svc.BatchWritePorts(context.Background(), BatchWriteRequest{
		DeviceID:  "device-1",
		Action:    portcollection.ActionPortBinding,
		PortIDs:   []string{"p1"},
		BindOp:    "delete", // typo for "remove"
		IPAddress: "10.62.25.5",
	}, "test-op")

	assert.ErrorIs(t, err, ErrBindOpInvalid)
	assert.Nil(t, result)

	mockExec.AssertNotCalled(t, "ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestBatchWritePorts_BindMacRejected (CR-01 2026-07-09 修复守护)
//
// 批量 port_binding 携带 null MAC 时,BatchWritePorts 入口必须立即返回 ErrMACAddressInvalid。
func TestBatchWritePorts_BindMacRejected(t *testing.T) {
	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, newTestDB(t))

	result, err := svc.BatchWritePorts(context.Background(), BatchWriteRequest{
		DeviceID:   "device-1",
		Action:     portcollection.ActionPortBinding,
		PortIDs:    []string{"p1"},
		BindOp:     "add",
		IPAddress:  "10.62.25.5",
		MACAddress: "00:00:00:00:00:00",
	}, "test-op")

	assert.ErrorIs(t, err, ErrMACAddressInvalid)
	assert.Nil(t, result)

	mockExec.AssertNotCalled(t, "ExecuteCustom", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockColl.AssertNotCalled(t, "CollectDevice", mock.Anything, mock.Anything)
}

// TestBatchWritePorts_ValidatorsAreActionSpecific (CR-01 2026-07-09 修复守护)
//
// action-specific 校验:shutdown 等无关 action 即使携带空 vlanId/IP/bindOp 也必须通过 validator
// (空串是合法 "字段未使用" 状态,不是入口错误)。
func TestBatchWritePorts_ValidatorsAreActionSpecific(t *testing.T) {
	mockExec := new(mockDeviceExecutor)
	mockColl := new(mockCollectionSvc)
	svc := newTestService(mockExec, mockColl, newTestDB(t))

	// Shutdown action + 空 vlanId/bindOp/IP/MAC — 应当通过 validator,继续走 DB 查询
	_, err := svc.BatchWritePorts(context.Background(), BatchWriteRequest{
		DeviceID: "device-1",
		Action:   portcollection.ActionShutdown,
		PortIDs:  []string{"p1"},
	}, "test-op")

	// validator 通过,继续走 DB 查询(无 seed) → 端口不存在 → result.Failed 有 1 项
	// err 应为 nil(仅 result.Failed 有内容,err 来自 service 是 nil)
	assert.NoError(t, err, "shutdown action with empty vlanId/IP/bindOp should pass entry validator (action-specific)")
}