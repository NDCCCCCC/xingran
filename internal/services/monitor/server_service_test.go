package monitor

// server_service_test.go — Phase 73-04 Task 4 (D-02 混合范本: glebarez sqlite + MetricsProvider mock).
//
// 范本要点:
//   - MetricsProvider(testify/mock)mock 覆盖缓存命中/未命中降级路径
//   - glebarez sqlite(sys_server_info + sys_system_metrics, 列对齐 internal/models/monitor.go)
//     覆盖 SaveSystemMetrics / GetSystemMetricsHistory 的真实 SQL 路径
//   - 降级路径 getCurrentServerInfo / getCurrentMetricsRealtime 走 internal/pkg/system
//     真实系统指标(测试机上真实可执行,断言真实值)
//
// 已锁定的业务行为(见 73-04-SUMMARY.md):
//   - GetServerInfo 降级判定是 `err == nil && info != nil`:provider 返回 (nil, nil)
//     同样触发实时降级(锁定)。
//   - GetSystemMetricsHistory 的 OrderByColumn 非空但不在白名单 → 无显式排序(与
//     login_log/oper_log 的 Q5 同款行为)。

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// Compile-time interface assertion — 锁定 MetricsProvider 可 mock 契约。
var _ MetricsProvider = (*mockMetricsProvider)(nil)

// mockMetricsProvider 实现 MetricsProvider(GetServerInfo + GetCurrentMetrics)。
type mockMetricsProvider struct {
	mock.Mock
}

func (m *mockMetricsProvider) GetServerInfo(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockMetricsProvider) GetCurrentMetrics(ctx context.Context) (*SystemMetricsData, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SystemMetricsData), args.Error(1)
}

// setupTestServerDB 创建 sys_server_info + sys_system_metrics 表
// (列对齐 internal/models/monitor.go ServerInfo/SystemMetrics,含 BaseModel 软删列)。
func setupTestServerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_server_info (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			host_name TEXT,
			os TEXT,
			arch TEXT,
			cpu_count INTEGER,
			total_memory INTEGER,
			available_memory INTEGER,
			disk_total INTEGER,
			disk_available INTEGER,
			status INTEGER,
			last_active_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_system_metrics (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			server_id TEXT,
			cpu_usage REAL,
			memory_usage REAL,
			disk_usage REAL,
			network_rx INTEGER,
			network_tx INTEGER,
			process_count INTEGER,
			load_average REAL,
			timestamp DATETIME
		)
	`).Error)
	return db
}

// newServerServiceFixture 构造真实 serverServiceImpl。
func newServerServiceFixture(db *gorm.DB, provider MetricsProvider) *serverServiceImpl {
	return NewServerService(db, provider).(*serverServiceImpl)
}

// seedSystemMetrics 插入一行指标(显式 id)。
func seedSystemMetrics(t *testing.T, db *gorm.DB, id, serverID string, cpuUsage float64, ts time.Time) {
	t.Helper()
	require.NoError(t, db.Create(&models.SystemMetrics{
		BaseModel:  models.BaseModel{ID: id},
		ServerID:   serverID,
		CPUUsage:   cpuUsage,
		MemoryUsage: 40.5,
		DiskUsage:  60.0,
		Timestamp:  ts,
	}).Error)
}

func TestServerService_CompileOnly(t *testing.T) {
	svc := newServerServiceFixture(setupTestServerDB(t), &mockMetricsProvider{})
	assert.NotNil(t, svc)
}

// ==================== GetServerInfo ====================

func TestServerService_GetServerInfo_FromProvider_FullMap(t *testing.T) {
	provider := &mockMetricsProvider{}
	svc := newServerServiceFixture(setupTestServerDB(t), provider)
	provider.On("GetServerInfo", mock.Anything).Return(map[string]interface{}{
		"hostname":          "srv-01",
		"os":                "windows",
		"arch":              "amd64",
		"cpu_count":         float64(8),
		"total_memory":      float64(16 << 30),
		"available_memory":  float64(8 << 30),
		"disk_total":        float64(512 << 30),
		"disk_available":    float64(256 << 30),
	}, nil)

	servers, total, err := svc.GetServerInfo(context.Background(), ServerInfoParams{Current: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, servers, 1)

	s := servers[0]
	assert.Equal(t, "srv-01", s.HostName)
	assert.Equal(t, "windows", s.OS)
	assert.Equal(t, "amd64", s.Arch)
	assert.Equal(t, 8, s.CPUCount)
	assert.Equal(t, uint64(16<<30), s.TotalMemory)
	assert.Equal(t, uint64(8<<30), s.AvailableMemory)
	assert.Equal(t, uint64(512<<30), s.DiskTotal)
	assert.Equal(t, uint64(256<<30), s.DiskAvailable)
	assert.Equal(t, models.ServerStatusNormal, s.Status)
	require.NotNil(t, s.LastActiveAt)
	provider.AssertExpectations(t)
}

// convertToServerInfo 边界: 类型不匹配的字段被跳过(零值),不 panic。
func TestServerService_GetServerInfo_FromProvider_WrongTypesSkipped(t *testing.T) {
	provider := &mockMetricsProvider{}
	svc := newServerServiceFixture(setupTestServerDB(t), provider)
	provider.On("GetServerInfo", mock.Anything).Return(map[string]interface{}{
		"hostname":     123,           // int 而非 string → 跳过
		"cpu_count":    "eight",       // string 而非 float64 → 跳过
		"total_memory": float64(1024), // 正确类型 → 保留
	}, nil)

	servers, _, err := svc.GetServerInfo(context.Background(), ServerInfoParams{})
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Empty(t, servers[0].HostName, "错误类型 hostname 应被跳过(零值)")
	assert.Equal(t, 0, servers[0].CPUCount)
	assert.Equal(t, uint64(1024), servers[0].TotalMemory)
	provider.AssertExpectations(t)
}

func TestServerService_GetServerInfo_FromProvider_EmptyMap(t *testing.T) {
	provider := &mockMetricsProvider{}
	svc := newServerServiceFixture(setupTestServerDB(t), provider)
	provider.On("GetServerInfo", mock.Anything).Return(map[string]interface{}{}, nil)

	servers, total, err := svc.GetServerInfo(context.Background(), ServerInfoParams{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, servers, 1)
	assert.Equal(t, models.ServerStatusNormal, servers[0].Status)
	provider.AssertExpectations(t)
}

// 行为锁定: provider 返回 (nil, nil) → info != nil 判定不满足 → 实时降级。
func TestServerService_GetServerInfo_ProviderNilMap_FallsBackRealtime(t *testing.T) {
	provider := &mockMetricsProvider{}
	svc := newServerServiceFixture(setupTestServerDB(t), provider)
	provider.On("GetServerInfo", mock.Anything).Return(nil, nil)

	servers, total, err := svc.GetServerInfo(context.Background(), ServerInfoParams{})
	require.NoError(t, err, "实时降级在测试机上应成功")
	assert.Equal(t, int64(1), total)
	require.Len(t, servers, 1)

	// getCurrentServerInfo 真实值断言
	assert.NotEmpty(t, servers[0].HostName, "hostname 应取自 os.Hostname()")
	assert.Equal(t, runtime.GOOS, servers[0].OS)
	assert.Equal(t, runtime.GOARCH, servers[0].Arch)
	assert.GreaterOrEqual(t, servers[0].CPUCount, 1, "CPUCount >= 1")
	assert.Greater(t, servers[0].TotalMemory, uint64(0))
	assert.Greater(t, servers[0].DiskTotal, uint64(0))
	provider.AssertExpectations(t)
}

func TestServerService_GetServerInfo_ProviderError_FallsBackRealtime(t *testing.T) {
	provider := &mockMetricsProvider{}
	svc := newServerServiceFixture(setupTestServerDB(t), provider)
	provider.On("GetServerInfo", mock.Anything).Return(nil, assert.AnError)

	servers, total, err := svc.GetServerInfo(context.Background(), ServerInfoParams{})
	require.NoError(t, err, "provider 失败 → 实时降级成功")
	assert.Equal(t, int64(1), total)
	require.Len(t, servers, 1)
	assert.NotEmpty(t, servers[0].HostName)
	provider.AssertExpectations(t)
}

// ==================== GetCurrentServerMetrics ====================

func TestServerService_GetCurrentServerMetrics_FromProvider(t *testing.T) {
	provider := &mockMetricsProvider{}
	svc := newServerServiceFixture(setupTestServerDB(t), provider)
	want := &SystemMetricsData{
		CPUUsage: 55.5, MemoryUsage: 66.6, DiskUsage: 77.7,
		NetworkRx: 100, NetworkTx: 200, ProcessNum: 42,
		TotalMemory: 4096, UsedMemory: 2048, Timestamp: time.Now(),
	}
	provider.On("GetCurrentMetrics", mock.Anything).Return(want, nil)

	got, err := svc.GetCurrentServerMetrics(context.Background())
	require.NoError(t, err)
	assert.Equal(t, want, got)
	provider.AssertExpectations(t)
}

// getCurrentMetricsRealtime 路径: provider 错误 → 真实系统指标。
func TestServerService_GetCurrentServerMetrics_ProviderError_FallsBackRealtime(t *testing.T) {
	provider := &mockMetricsProvider{}
	svc := newServerServiceFixture(setupTestServerDB(t), provider)
	provider.On("GetCurrentMetrics", mock.Anything).Return(nil, assert.AnError)

	got, err := svc.GetCurrentServerMetrics(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, got.CPUUsage, 0.0)
	assert.GreaterOrEqual(t, got.MemoryUsage, 0.0)
	assert.Greater(t, got.TotalMemory, uint64(0), "真实 TotalMemory > 0")
	assert.False(t, got.Timestamp.IsZero())
	provider.AssertExpectations(t)
}

// ==================== SaveSystemMetrics ====================

func TestServerService_SaveSystemMetrics_Success(t *testing.T) {
	db := setupTestServerDB(t)
	svc := newServerServiceFixture(db, &mockMetricsProvider{})

	metrics := &models.SystemMetrics{
		ServerID:     "srv-01",
		CPUUsage:     12.5,
		MemoryUsage:  45.0,
		DiskUsage:    70.0,
		NetworkRx:    1024,
		NetworkTx:    2048,
		ProcessCount: 100,
		LoadAverage:  1.25,
		Timestamp:    time.Now(),
	}
	require.NoError(t, svc.SaveSystemMetrics(context.Background(), metrics))

	var saved models.SystemMetrics
	require.NoError(t, db.Where("server_id = ?", "srv-01").First(&saved).Error)
	assert.Equal(t, 12.5, saved.CPUUsage)
	assert.Equal(t, 100, saved.ProcessCount)
	assert.NotEmpty(t, saved.ID, "BaseModel.BeforeCreate 应自动生成 UUID")
}

func TestServerService_SaveSystemMetrics_DBError(t *testing.T) {
	db := setupTestServerDB(t)
	require.NoError(t, db.Exec("DROP TABLE sys_system_metrics").Error)
	svc := newServerServiceFixture(db, &mockMetricsProvider{})

	err := svc.SaveSystemMetrics(context.Background(), &models.SystemMetrics{ServerID: "srv-01"})
	assert.Error(t, err)
}

// ==================== GetSystemMetricsHistory ====================

func TestServerService_GetSystemMetricsHistory_Empty(t *testing.T) {
	svc := newServerServiceFixture(setupTestServerDB(t), &mockMetricsProvider{})
	list, total, err := svc.GetSystemMetricsHistory(context.Background(), MetricsHistoryParams{Current: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, list)
}

func TestServerService_GetSystemMetricsHistory_ServerIDFilter(t *testing.T) {
	db := setupTestServerDB(t)
	base := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	seedSystemMetrics(t, db, "m-1", "srv-a", 10.0, base)
	seedSystemMetrics(t, db, "m-2", "srv-a", 20.0, base.Add(time.Minute))
	seedSystemMetrics(t, db, "m-3", "srv-b", 30.0, base)

	svc := newServerServiceFixture(db, &mockMetricsProvider{})
	list, total, err := svc.GetSystemMetricsHistory(context.Background(), MetricsHistoryParams{
		ServerID: "srv-a", Current: 1, PageSize: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, list, 2)

	// 无 ServerID → 全量
	listAll, totalAll, err := svc.GetSystemMetricsHistory(context.Background(), MetricsHistoryParams{Current: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(3), totalAll)
	assert.Len(t, listAll, 3)
}

func TestServerService_GetSystemMetricsHistory_TimeRangeFilter(t *testing.T) {
	db := setupTestServerDB(t)
	base := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	seedSystemMetrics(t, db, "m-1", "srv-a", 10.0, base)                    // 10:00
	seedSystemMetrics(t, db, "m-2", "srv-a", 20.0, base.Add(2*time.Hour)) // 12:00

	svc := newServerServiceFixture(db, &mockMetricsProvider{})
	begin := "2026-08-20 11:00:00"
	end := "2026-08-20 13:00:00"
	list, total, err := svc.GetSystemMetricsHistory(context.Background(), MetricsHistoryParams{
		StartTime: &begin, EndTime: &end, Current: 1, PageSize: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total, "只有 12:00 的行落在窗口内")
	require.Len(t, list, 1)
	assert.Equal(t, "m-2", list[0].ID)
}

func TestServerService_GetSystemMetricsHistory_PaginationAndDefaultOrder(t *testing.T) {
	db := setupTestServerDB(t)
	base := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		seedSystemMetrics(t, db, "m-"+string(rune('0'+i)), "srv-a", float64(i), base.Add(time.Duration(i)*time.Minute))
	}

	svc := newServerServiceFixture(db, &mockMetricsProvider{})
	p1, total, err := svc.GetSystemMetricsHistory(context.Background(), MetricsHistoryParams{Current: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	require.Len(t, p1, 2)
	// 默认 timestamp DESC → 最新的 m-4 在前
	assert.Equal(t, "m-4", p1[0].ID)
	assert.Equal(t, "m-3", p1[1].ID)

	p3, _, err := svc.GetSystemMetricsHistory(context.Background(), MetricsHistoryParams{Current: 3, PageSize: 2})
	require.NoError(t, err)
	assert.Len(t, p3, 1)
}

func TestServerService_GetSystemMetricsHistory_SortCpuUsageAsc(t *testing.T) {
	db := setupTestServerDB(t)
	base := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	seedSystemMetrics(t, db, "m-high", "srv-a", 90.0, base)
	seedSystemMetrics(t, db, "m-low", "srv-a", 5.0, base)

	svc := newServerServiceFixture(db, &mockMetricsProvider{})
	list, _, err := svc.GetSystemMetricsHistory(context.Background(), MetricsHistoryParams{
		Current: 1, PageSize: 10, OrderByColumn: "cpuUsage", IsAsc: true,
	})
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "m-low", list[0].ID, "cpuUsage ASC → 5.0 在前")
	assert.Equal(t, "m-high", list[1].ID)
}

func TestServerService_GetSystemMetricsHistory_SortTimestampDesc_Explicit(t *testing.T) {
	db := setupTestServerDB(t)
	base := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	seedSystemMetrics(t, db, "m-old", "srv-a", 50.0, base)
	seedSystemMetrics(t, db, "m-new", "srv-a", 50.0, base.Add(time.Hour))

	svc := newServerServiceFixture(db, &mockMetricsProvider{})
	list, _, err := svc.GetSystemMetricsHistory(context.Background(), MetricsHistoryParams{
		Current: 1, PageSize: 10, OrderByColumn: "timestamp", IsAsc: false,
	})
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "m-new", list[0].ID)
	assert.Equal(t, "m-old", list[1].ID)
}

func TestServerService_GetSystemMetricsHistory_InvalidSortColumn_NoInjection(t *testing.T) {
	db := setupTestServerDB(t)
	base := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	seedSystemMetrics(t, db, "m-1", "srv-a", 10.0, base)
	seedSystemMetrics(t, db, "m-2", "srv-a", 20.0, base)

	svc := newServerServiceFixture(db, &mockMetricsProvider{})
	list, total, err := svc.GetSystemMetricsHistory(context.Background(), MetricsHistoryParams{
		Current: 1, PageSize: 10, OrderByColumn: "evil; DROP TABLE sys_system_metrics",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, list, 2)

	var count int64
	require.NoError(t, db.Model(&models.SystemMetrics{}).Count(&count).Error)
	assert.Equal(t, int64(2), count, "表未被注入破坏")
}

func TestServerService_GetSystemMetricsHistory_DBError(t *testing.T) {
	db := setupTestServerDB(t)
	require.NoError(t, db.Exec("DROP TABLE sys_system_metrics").Error)
	svc := newServerServiceFixture(db, &mockMetricsProvider{})

	_, _, err := svc.GetSystemMetricsHistory(context.Background(), MetricsHistoryParams{Current: 1, PageSize: 10})
	assert.Error(t, err)
}
