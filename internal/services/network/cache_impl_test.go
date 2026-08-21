// =====================================================================
// cache_impl_test.go — covers internal/services/network/cache_impl.go (127 stmts)
// Pattern: portwrite pure-mock (interface assertion + testify/mock, D-02).
// CacheProvider is fully mocked; base NetworkDeviceService runs on
// glebarez sqlite in-memory (unavoidable gorm path — plan allows minimal
// sqlite). Per Phase 73 Plan 03 — IMP-06 (services/network half)
//
// Fixture notes:
//   - NewServiceWithCache receives nil discovery/collection services:
//     only QuickCreateDevice's SUCCESS path needs the SNMP probe (network
//     I/O — untestable). QuickCreate is covered via its pre-probe error
//     branches (duplicate IP / missing credential).
//   - Status values use models.DeviceStatus* constants (Online/Offline/
//     Unknown) per the Status Value Convention — never raw 0/1.
//   - Seeds use raw db.Create so seeding never triggers invalidation
//     expectations (testify m.Called panics on unexpected calls).
// =====================================================================

package network

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	serviceBase "github.com/xingran-next/xingran-go-backend/internal/services/base"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// Compile-time interface assertion — locks mockability contract (D-02).
var _ systemServices.CacheProvider = (*mockCacheProvider)(nil)

// mockCacheProvider embeds mock.Mock and implements the CacheProvider
// interface. GetOrSet/Delete/DeleteByPattern go through m.Called; the
// remaining methods are deterministic no-ops (never invoked by the impl).
type mockCacheProvider struct {
	mock.Mock
}

// GetOrSet mock:
//   - error return → propagate WITHOUT invoking query (cache failure path)
//   - nil error → invoke query (cache miss → DB fallback), then populate
//     dest via reflection (same semantics as NoOpCacheProvider.setValue).
func (m *mockCacheProvider) GetOrSet(ctx context.Context, key string, dest interface{},
	expiration time.Duration, query func() (interface{}, error)) error {
	args := m.Called(ctx, key, dest, expiration, query)
	if err := args.Error(0); err != nil {
		return err
	}
	result, err := query()
	if err != nil {
		return err
	}
	setMockDest(dest, result)
	return nil
}

func (m *mockCacheProvider) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockCacheProvider) DeleteByPattern(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

// Untouched-by-impl methods: deterministic no-ops (never asserted).
func (m *mockCacheProvider) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (m *mockCacheProvider) MDelete(ctx context.Context, keys ...string) error { return nil }
func (m *mockCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}
func (m *mockCacheProvider) SetTTL(ctx context.Context, key string, expiration time.Duration) error {
	return nil
}
func (m *mockCacheProvider) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}
func (m *mockCacheProvider) GetStats(ctx context.Context) (*systemServices.CacheStats, error) {
	return &systemServices.CacheStats{}, nil
}

// setMockDest reflect-assigns query() result into the dest pointer passed
// by the service (same semantics as system.NoOpCacheProvider.setValue).
func setMockDest(dest interface{}, value interface{}) {
	if dest == nil || value == nil {
		return
	}
	dv := reflect.ValueOf(dest)
	if dv.Kind() != reflect.Ptr {
		return
	}
	elem := dv.Elem()
	vv := reflect.ValueOf(value)
	if vv.Kind() == reflect.Ptr {
		if vv.IsNil() {
			return
		}
		vv = vv.Elem()
	}
	if elem.IsValid() && vv.IsValid() && vv.Type().AssignableTo(elem.Type()) {
		elem.Set(vv)
	}
}

// newNetworkTestDB creates a sqlite in-memory DB with every table the
// base NetworkDeviceService touches. Unique named shared-cache DSN so
// concurrent pooled connections all see the same memory DB.
func newNetworkTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:nettest_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.NetworkDevice{},
		&models.Department{},
		&models.AuthCredential{},
	))
	return db
}

// newNetworkSvcOver wires a fresh cache impl + fresh mock cache over an
// existing DB. discovery/collection services are nil (see file header).
func newNetworkSvcOver(db *gorm.DB) (*cacheServiceImpl, *mockCacheProvider) {
	cache := new(mockCacheProvider)
	svc := &cacheServiceImpl{
		db:     db,
		base:   services.NewNetworkDeviceService(db, nil, nil),
		cache:  cache,
		config: nil,
	}
	return svc, cache
}

func newNetworkTestService(t *testing.T) (*gorm.DB, *cacheServiceImpl, *mockCacheProvider) {
	t.Helper()
	db := newNetworkTestDB(t)
	svc, cache := newNetworkSvcOver(db)
	return db, svc, cache
}

// ---- raw seed helpers (bypass the cache impl entirely) ----

// seedDevice inserts a device with an EXPLICIT id so cache keys
// (network_device:detail:<id> etc.) are assertable. Status is forced via
// explicit column update: DeviceStatusOnline (0) is a GORM zero value and
// would otherwise be skipped on Create (column default 2=Unknown applies).
func seedDevice(t *testing.T, db *gorm.DB, id, name, ip string, status models.DeviceStatus, deptID, credentialID *string) *models.NetworkDevice {
	t.Helper()
	device := &models.NetworkDevice{
		DeviceName:   name,
		DeviceType:   models.DeviceTypeSwitch,
		Vendor:       models.VendorHuawei,
		IPAddress:    ip,
		Port:         22,
		SNMPPort:     161,
		CredentialID: credentialID,
		DeptID:       deptID,
		Status:       status,
	}
	device.ID = id
	require.NoError(t, db.Create(device).Error)
	require.NoError(t, db.Model(&models.NetworkDevice{}).Where("id = ?", id).
		Update("status", int(status)).Error)
	return device
}

func seedDept(t *testing.T, db *gorm.DB, id, name string) {
	t.Helper()
	// dept_code has a UNIQUE index — derive it from the id to keep rows distinct.
	dept := &models.Department{DeptName: name, DeptCode: "code-" + id, Status: models.DeptStatusNormal}
	dept.ID = id
	require.NoError(t, db.Create(dept).Error)
}

func seedCredential(t *testing.T, db *gorm.DB, id, name string) {
	t.Helper()
	cred := &models.AuthCredential{
		CredentialName: name,
		ProtocolType:   "ssh",
		Username:       "admin",
	}
	cred.ID = id
	require.NoError(t, db.Create(cred).Error)
}

func strPtr(s string) *string { return &s }

// assertNoCacheInteraction guards that error paths never touch the cache.
func assertNoCacheInteraction(t *testing.T, cache *mockCacheProvider) {
	t.Helper()
	cache.AssertNotCalled(t, "GetOrSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	cache.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	cache.AssertNotCalled(t, "DeleteByPattern", mock.Anything, mock.Anything)
}

// ==================== Smoke / constructor ====================

// TestNetworkService_CompileOnly — smoke test ensures file compiles.
func TestNetworkService_CompileOnly(t *testing.T) {
	svc, cache := newNetworkSvcOver(newNetworkTestDB(t))
	assert.NotNil(t, svc)
	assert.NotNil(t, cache)
}

// TestNetworkService_NewServiceWithCache — constructor returns a
// CacheService implementation.
func TestNetworkService_NewServiceWithCache(t *testing.T) {
	db := newNetworkTestDB(t)
	var svc CacheService = NewServiceWithCache(db, nil, nil, new(mockCacheProvider), nil)
	assert.NotNil(t, svc)
}

// ==================== List / GetByID (uncached) ====================

func TestNetworkService_List_Empty(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	list, total, err := svc.List(context.Background(), &services.ListDeviceRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, list)
	assertNoCacheInteraction(t, cache)
}

func TestNetworkService_List_Success_FilterByType(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedDevice(t, db, "dev-1", "switch-1", "10.0.0.1", models.DeviceStatusOnline, nil, nil)
	seedDevice(t, db, "dev-2", "switch-2", "10.0.0.2", models.DeviceStatusOnline, nil, nil)
	require.NoError(t, db.Model(&models.NetworkDevice{}).Where("id = ?", "dev-2").
		Update("device_type", string(models.DeviceTypeRouter)).Error)

	// base List applies NO paging defaults — explicit Current/PageSize
	// required (PageSize 0 would compile to LIMIT 0).
	deviceType := models.DeviceTypeSwitch
	list, total, err := svc.List(context.Background(), &services.ListDeviceRequest{
		BaseListRequest: baseListRequest(),
		DeviceType:      &deviceType,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, "switch-1", list[0].DeviceName)
	assertNoCacheInteraction(t, cache)
}

func TestNetworkService_List_FilterByStatus(t *testing.T) {
	db, svc, _ := newNetworkTestService(t)
	seedDevice(t, db, "dev-1", "online-dev", "10.0.0.1", models.DeviceStatusOnline, nil, nil)
	seedDevice(t, db, "dev-2", "offline-dev", "10.0.0.2", models.DeviceStatusOffline, nil, nil)

	status := models.DeviceStatusOffline
	list, total, err := svc.List(context.Background(), &services.ListDeviceRequest{
		BaseListRequest: baseListRequest(),
		Status:          &status,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, "offline-dev", list[0].DeviceName)
}

// baseListRequest supplies explicit pagination (see FilterByType note).
func baseListRequest() serviceBase.BaseListRequest {
	return serviceBase.BaseListRequest{Current: 1, PageSize: 10}
}

func TestNetworkService_GetByID_Found(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedDevice(t, db, "dev-1", "found-dev", "10.0.0.1", models.DeviceStatusOnline, nil, nil)

	device, err := svc.GetByID(context.Background(), "dev-1")
	require.NoError(t, err)
	require.NotNil(t, device)
	assert.Equal(t, "dev-1", device.ID)
	assert.Equal(t, "found-dev", device.DeviceName)
	assertNoCacheInteraction(t, cache)
}

// TestNetworkService_GetByID_AssociationNames_NotPropagated — base
// GetByID calls loadAssociations on a THROWAWAY slice copy
// (&[]models.NetworkDevice{device}), so DeptName/CredentialName never
// reach the returned device. Latent base-service quirk (List IS correctly
// enriched) — locked as-is per D-12 (no business code changes).
func TestNetworkService_GetByID_AssociationNames_NotPropagated(t *testing.T) {
	db, svc, _ := newNetworkTestService(t)
	seedDept(t, db, "dept-1", "网络部")
	seedCredential(t, db, "cred-1", "核心交换凭证")
	seedDevice(t, db, "dev-1", "assoc-dev", "10.0.0.1", models.DeviceStatusOnline, strPtr("dept-1"), strPtr("cred-1"))

	device, err := svc.GetByID(context.Background(), "dev-1")
	require.NoError(t, err)
	require.NotNil(t, device)
	assert.Nil(t, device.DeptName, "GetByID does not propagate association names (base-service quirk)")
	assert.Nil(t, device.CredentialName)
}

// List DOES propagate association names (loadAssociations mutates the
// actual result slice) — verified here.
func TestNetworkService_List_AssociationNames(t *testing.T) {
	db, svc, _ := newNetworkTestService(t)
	seedDept(t, db, "dept-1", "网络部")
	seedCredential(t, db, "cred-1", "核心交换凭证")
	seedDevice(t, db, "dev-1", "assoc-dev", "10.0.0.1", models.DeviceStatusOnline, strPtr("dept-1"), strPtr("cred-1"))

	list, _, err := svc.List(context.Background(), &services.ListDeviceRequest{
		BaseListRequest: baseListRequest(),
	})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.NotNil(t, list[0].DeptName)
	assert.Equal(t, "网络部", *list[0].DeptName)
	require.NotNil(t, list[0].CredentialName)
	assert.Equal(t, "核心交换凭证", *list[0].CredentialName)
}

func TestNetworkService_GetByID_NotFound_Error(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	device, err := svc.GetByID(context.Background(), "missing")
	assert.Error(t, err)
	assert.Nil(t, device)
	assertNoCacheInteraction(t, cache)
}

// ==================== Create ====================

func TestNetworkService_Create_Success_NoDeps_InvalidatesStatisticsOnly(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	cache.On("Delete", mock.Anything, "network_device:statistics").Return(nil).Once()

	device, err := svc.Create(context.Background(), &services.CreateDeviceRequest{
		DeviceName: "bare-device",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.VendorHuawei,
		IPAddress:  "10.1.0.1",
		Port:       22,
		Status:     models.DeviceStatusOnline,
	})
	require.NoError(t, err)
	require.NotNil(t, device)
	assert.Equal(t, "bare-device", device.DeviceName)
	cache.AssertExpectations(t)
}

func TestNetworkService_Create_Success_WithDept_InvalidatesDeptAndStatistics(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedDept(t, db, "dept-1", "网络部")

	cache.On("Delete", mock.Anything, "network_device:statistics").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:dept:dept-1").Return(nil).Once()

	device, err := svc.Create(context.Background(), &services.CreateDeviceRequest{
		DeviceName: "dept-device",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.VendorH3C,
		IPAddress:  "10.1.0.2",
		Status:     models.DeviceStatusOnline,
		DeptID:     strPtr("dept-1"),
	})
	require.NoError(t, err)
	require.NotNil(t, device)
	cache.AssertExpectations(t)
}

func TestNetworkService_Create_Success_WithCredential_InvalidatesCredentialAndStatistics(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedCredential(t, db, "cred-1", "ssh-admin")

	cache.On("Delete", mock.Anything, "network_device:statistics").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:credential:cred-1").Return(nil).Once()

	device, err := svc.Create(context.Background(), &services.CreateDeviceRequest{
		DeviceName:   "cred-device",
		DeviceType:   models.DeviceTypeFirewall,
		Vendor:       models.VendorRuijie,
		IPAddress:    "10.1.0.3",
		Status:       models.DeviceStatusOnline,
		CredentialID: strPtr("cred-1"),
	})
	require.NoError(t, err)
	require.NotNil(t, device)
	cache.AssertExpectations(t)
}

func TestNetworkService_Create_DuplicateIP_Error_NoInvalidation(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedDevice(t, db, "dev-1", "ip-holder", "10.1.0.9", models.DeviceStatusOnline, nil, nil)

	device, err := svc.Create(context.Background(), &services.CreateDeviceRequest{
		DeviceName: "clone",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.VendorHuawei,
		IPAddress:  "10.1.0.9",
		Status:     models.DeviceStatusOnline,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "IP地址已存在")
	assert.Nil(t, device)
	assertNoCacheInteraction(t, cache)
}

func TestNetworkService_Create_DeptMissing_Error_NoInvalidation(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	device, err := svc.Create(context.Background(), &services.CreateDeviceRequest{
		DeviceName: "orphan",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.VendorHuawei,
		IPAddress:  "10.1.0.4",
		Status:     models.DeviceStatusOnline,
		DeptID:     strPtr("ghost-dept"),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "部门不存在")
	assert.Nil(t, device)
	assertNoCacheInteraction(t, cache)
}

func TestNetworkService_Create_CredentialMissing_Error_NoInvalidation(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	device, err := svc.Create(context.Background(), &services.CreateDeviceRequest{
		DeviceName:   "orphan-cred",
		DeviceType:   models.DeviceTypeSwitch,
		Vendor:       models.VendorHuawei,
		IPAddress:    "10.1.0.5",
		Status:       models.DeviceStatusOnline,
		CredentialID: strPtr("ghost-cred"),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "授权凭证不存在")
	assert.Nil(t, device)
	assertNoCacheInteraction(t, cache)
}

// ==================== QuickCreateDevice (pre-probe error branches) ====================

func TestNetworkService_QuickCreateDevice_DuplicateIP_Error_NoInvalidation(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedDevice(t, db, "dev-1", "existing", "10.2.0.1", models.DeviceStatusOnline, nil, nil)

	device, err := svc.QuickCreateDevice(context.Background(), &services.QuickCreateRequest{
		IPAddress:    "10.2.0.1",
		CredentialID: "00000000-0000-0000-0000-000000000001",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已存在")
	assert.Nil(t, device)
	assertNoCacheInteraction(t, cache)
}

func TestNetworkService_QuickCreateDevice_CredentialMissing_Error_NoInvalidation(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	device, err := svc.QuickCreateDevice(context.Background(), &services.QuickCreateRequest{
		IPAddress:    "10.2.0.2",
		CredentialID: "00000000-0000-0000-0000-000000000002",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "授权凭证不存在")
	assert.Nil(t, device)
	assertNoCacheInteraction(t, cache)
}

// ==================== Update (branch-heavy invalidation matrix) ====================

func TestNetworkService_Update_DeviceMissing_Error_NoInvalidation(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	// cache_impl.Update runs its own raw First before delegating, so a
	// missing device surfaces as bare gorm "record not found".
	err := svc.Update(context.Background(), &services.UpdateDeviceRequest{
		ID:     "missing",
		Status: models.DeviceStatusOnline,
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	assertNoCacheInteraction(t, cache)
}

func TestNetworkService_Update_IPConflict_Error_NoInvalidation(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedDevice(t, db, "dev-1", "one", "10.3.0.1", models.DeviceStatusOnline, nil, nil)
	seedDevice(t, db, "dev-2", "two", "10.3.0.2", models.DeviceStatusOnline, nil, nil)

	err := svc.Update(context.Background(), &services.UpdateDeviceRequest{
		ID:        "dev-1",
		IPAddress: "10.3.0.2", // held by dev-2
		Status:    models.DeviceStatusOnline,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "IP地址已被其他设备使用")
	assertNoCacheInteraction(t, cache)
}

func TestNetworkService_Update_SameDept_SameStatus_MinimalInvalidation(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedDept(t, db, "dept-1", "网络部")
	seedDevice(t, db, "dev-1", "stable", "10.3.0.1", models.DeviceStatusOnline, strPtr("dept-1"), nil)

	cache.On("Delete", mock.Anything, "network_device:detail:dev-1").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:dept:dept-1").Return(nil).Once()

	err := svc.Update(context.Background(), &services.UpdateDeviceRequest{
		ID:         "dev-1",
		DeviceName: "stable-renamed",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.VendorHuawei,
		IPAddress:  "10.3.0.1",
		Status:     models.DeviceStatusOnline, // unchanged → no stats invalidation
		DeptID:     strPtr("dept-1"),          // unchanged → old-dept branch NOT taken
	})
	require.NoError(t, err)
	cache.AssertExpectations(t)
}

func TestNetworkService_Update_DeptChanged_InvalidatesOldAndNewDept(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedDept(t, db, "dept-old", "旧部门")
	seedDept(t, db, "dept-new", "新部门")
	seedDevice(t, db, "dev-1", "mover", "10.3.0.2", models.DeviceStatusOnline, strPtr("dept-old"), nil)

	cache.On("Delete", mock.Anything, "network_device:detail:dev-1").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:dept:dept-old").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:dept:dept-new").Return(nil).Once()

	err := svc.Update(context.Background(), &services.UpdateDeviceRequest{
		ID:         "dev-1",
		DeviceName: "mover",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.VendorHuawei,
		IPAddress:  "10.3.0.2",
		Status:     models.DeviceStatusOnline,
		DeptID:     strPtr("dept-new"),
	})
	require.NoError(t, err)
	cache.AssertExpectations(t)
}

func TestNetworkService_Update_DeptCleared_InvalidatesOldDeptOnly(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedDept(t, db, "dept-old", "旧部门")
	seedDevice(t, db, "dev-1", "undepartmented", "10.3.0.3", models.DeviceStatusOnline, strPtr("dept-old"), nil)

	cache.On("Delete", mock.Anything, "network_device:detail:dev-1").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:dept:dept-old").Return(nil).Once()

	err := svc.Update(context.Background(), &services.UpdateDeviceRequest{
		ID:         "dev-1",
		DeviceName: "undepartmented",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.VendorHuawei,
		IPAddress:  "10.3.0.3",
		Status:     models.DeviceStatusOnline,
		DeptID:     nil, // cleared → old-dept branch taken, new-dept branch skipped
	})
	require.NoError(t, err)
	cache.AssertExpectations(t)
}

func TestNetworkService_Update_CredentialChanged_InvalidatesOldAndNewCredential(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedCredential(t, db, "cred-old", "旧凭证")
	seedCredential(t, db, "cred-new", "新凭证")
	seedDevice(t, db, "dev-1", "recred", "10.3.0.4", models.DeviceStatusOnline, nil, strPtr("cred-old"))

	cache.On("Delete", mock.Anything, "network_device:detail:dev-1").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:credential:cred-old").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:credential:cred-new").Return(nil).Once()

	err := svc.Update(context.Background(), &services.UpdateDeviceRequest{
		ID:           "dev-1",
		DeviceName:   "recred",
		DeviceType:   models.DeviceTypeSwitch,
		Vendor:       models.VendorHuawei,
		IPAddress:    "10.3.0.4",
		Status:       models.DeviceStatusOnline,
		CredentialID: strPtr("cred-new"),
	})
	require.NoError(t, err)
	cache.AssertExpectations(t)
}

func TestNetworkService_Update_StatusChanged_InvalidatesStatistics(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedDevice(t, db, "dev-1", "flapper", "10.3.0.5", models.DeviceStatusOnline, nil, nil)

	cache.On("Delete", mock.Anything, "network_device:detail:dev-1").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:statistics").Return(nil).Once()

	err := svc.Update(context.Background(), &services.UpdateDeviceRequest{
		ID:         "dev-1",
		DeviceName: "flapper",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.VendorHuawei,
		IPAddress:  "10.3.0.5",
		Status:     models.DeviceStatusOffline, // changed → stats invalidation
	})
	require.NoError(t, err)

	var got models.NetworkDevice
	require.NoError(t, db.Where("id = ?", "dev-1").First(&got).Error)
	assert.Equal(t, models.DeviceStatusOffline, got.Status)
	cache.AssertExpectations(t)
}

// ==================== Delete / BatchDelete ====================

func TestNetworkService_Delete_Success_InvalidatesDeviceDeptCredStats(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedDept(t, db, "dept-1", "网络部")
	seedCredential(t, db, "cred-1", "凭证")
	seedDevice(t, db, "dev-1", "doomed", "10.4.0.1", models.DeviceStatusOnline, strPtr("dept-1"), strPtr("cred-1"))

	cache.On("Delete", mock.Anything, "network_device:detail:dev-1").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:dept:dept-1").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:credential:cred-1").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:statistics").Return(nil).Once()

	err := svc.Delete(context.Background(), "dev-1")
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.NetworkDevice{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
	cache.AssertExpectations(t)
}

func TestNetworkService_Delete_DeviceMissing_Error_NoInvalidation(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	// Like Update, cache_impl.Delete runs its own raw First before
	// delegating — missing device surfaces as bare gorm "record not found".
	err := svc.Delete(context.Background(), "missing")
	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	assertNoCacheInteraction(t, cache)
}

func TestNetworkService_BatchDelete_Success_InvalidatesPerDeviceAndAggregates(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedDept(t, db, "dept-1", "一部")
	seedDept(t, db, "dept-2", "二部")
	seedCredential(t, db, "cred-1", "共享凭证")
	seedDevice(t, db, "dev-1", "b1", "10.4.0.1", models.DeviceStatusOnline, strPtr("dept-1"), strPtr("cred-1"))
	seedDevice(t, db, "dev-2", "b2", "10.4.0.2", models.DeviceStatusOnline, strPtr("dept-2"), strPtr("cred-1"))

	cache.On("Delete", mock.Anything, "network_device:detail:dev-1").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:detail:dev-2").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:dept:dept-1").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:dept:dept-2").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:credential:cred-1").Return(nil).Once() // deduped via credentialMap
	cache.On("Delete", mock.Anything, "network_device:statistics").Return(nil).Once()

	err := svc.BatchDelete(context.Background(), []string{"dev-1", "dev-2"})
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.NetworkDevice{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
	cache.AssertExpectations(t)
}

func TestNetworkService_BatchDelete_DeviceMissing_Error_NoInvalidation(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	err := svc.BatchDelete(context.Background(), []string{"ghost-1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "设备不存在")
	assertNoCacheInteraction(t, cache)
}

// ==================== UpdateStatus / UpdateStatusBatch ====================

func TestNetworkService_UpdateStatus_Success_InvalidatesDeviceAndStats(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedDevice(t, db, "dev-1", "status-flip", "10.5.0.1", models.DeviceStatusOnline, nil, nil)

	cache.On("Delete", mock.Anything, "network_device:detail:dev-1").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:statistics").Return(nil).Once()

	err := svc.UpdateStatus(context.Background(), "dev-1", models.DeviceStatusOffline)
	require.NoError(t, err)

	var got models.NetworkDevice
	require.NoError(t, db.Where("id = ?", "dev-1").First(&got).Error)
	assert.Equal(t, models.DeviceStatusOffline, got.Status)
	cache.AssertExpectations(t)
}

// base.UpdateStatus on a missing id updates 0 rows without error, so the
// invalidation still fires — lock that behavior.
func TestNetworkService_UpdateStatus_MissingID_StillInvalidates(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	cache.On("Delete", mock.Anything, "network_device:detail:missing").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:statistics").Return(nil).Once()

	err := svc.UpdateStatus(context.Background(), "missing", models.DeviceStatusOffline)
	require.NoError(t, err)
	cache.AssertExpectations(t)
}

func TestNetworkService_UpdateStatusBatch_Success_InvalidatesEachDeviceAndStats(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedDevice(t, db, "dev-1", "batch-a", "10.5.0.1", models.DeviceStatusOnline, nil, nil)
	seedDevice(t, db, "dev-2", "batch-b", "10.5.0.2", models.DeviceStatusOnline, nil, nil)

	cache.On("Delete", mock.Anything, "network_device:detail:dev-1").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:detail:dev-2").Return(nil).Once()
	cache.On("Delete", mock.Anything, "network_device:statistics").Return(nil).Once()

	err := svc.UpdateStatusBatch(context.Background(), []string{"dev-1", "dev-2"}, models.DeviceStatusUnknown)
	require.NoError(t, err)

	var updated int64
	require.NoError(t, db.Model(&models.NetworkDevice{}).Where("status = ?", models.DeviceStatusUnknown).Count(&updated).Error)
	assert.Equal(t, int64(2), updated)
	cache.AssertExpectations(t)
}

// ==================== GetDeviceStatistics (cached) ====================

func TestNetworkService_GetDeviceStatistics_CacheError_Propagates(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	wantErr := errors.New("redis unavailable")
	cache.On("GetOrSet", mock.Anything, "network_device:statistics", mock.Anything, mock.Anything, mock.Anything).
		Return(wantErr).Once()

	stats, err := svc.GetDeviceStatistics(context.Background())
	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, stats)
	cache.AssertExpectations(t)
}

func TestNetworkService_GetDeviceStatistics_CacheMiss_Success(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedDevice(t, db, "dev-1", "up-dev", "10.6.0.1", models.DeviceStatusOnline, nil, nil)
	seedDevice(t, db, "dev-2", "down-dev", "10.6.0.2", models.DeviceStatusOffline, nil, nil)

	cache.On("GetOrSet", mock.Anything, "network_device:statistics", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	stats, err := svc.GetDeviceStatistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, int64(2), toInt64(stats["totalDevices"]))
	assert.Equal(t, int64(1), toInt64(stats["onlineDevices"]))
	assert.Equal(t, int64(1), toInt64(stats["offlineDevices"]))
	assert.Equal(t, int64(0), toInt64(stats["unknownDevices"]))
	cache.AssertExpectations(t)
}

// ==================== GetDevicesByDept (cached) ====================

func TestNetworkService_GetDevicesByDept_CacheError_Propagates(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	wantErr := errors.New("redis down")
	cache.On("GetOrSet", mock.Anything, "network_device:dept:dept-9", mock.Anything, mock.Anything, mock.Anything).
		Return(wantErr).Once()

	devices, err := svc.GetDevicesByDept(context.Background(), "dept-9")
	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, devices)
	cache.AssertExpectations(t)
}

func TestNetworkService_GetDevicesByDept_CacheMiss_Success(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedDept(t, db, "dept-1", "网络部")
	seedDevice(t, db, "dev-1", "dept-dev-a", "10.7.0.1", models.DeviceStatusOnline, strPtr("dept-1"), nil)
	seedDevice(t, db, "dev-2", "dept-dev-b", "10.7.0.2", models.DeviceStatusOnline, strPtr("dept-1"), nil)
	seedDevice(t, db, "dev-3", "other-dev", "10.7.0.3", models.DeviceStatusOnline, nil, nil)

	cache.On("GetOrSet", mock.Anything, "network_device:dept:dept-1", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	devices, err := svc.GetDevicesByDept(context.Background(), "dept-1")
	require.NoError(t, err)
	require.Len(t, devices, 2)
	assert.Equal(t, "dept-dev-a", devices[0].DeviceName)
	cache.AssertExpectations(t)
}

// ==================== GetDevicesByCredential (cached) ====================

func TestNetworkService_GetDevicesByCredential_CacheError_Propagates(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	wantErr := errors.New("redis gone")
	cache.On("GetOrSet", mock.Anything, "network_device:credential:cred-9", mock.Anything, mock.Anything, mock.Anything).
		Return(wantErr).Once()

	devices, err := svc.GetDevicesByCredential(context.Background(), "cred-9")
	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, devices)
	cache.AssertExpectations(t)
}

func TestNetworkService_GetDevicesByCredential_CacheMiss_Success(t *testing.T) {
	db, svc, cache := newNetworkTestService(t)
	seedCredential(t, db, "cred-1", "shared-cred")
	seedDevice(t, db, "dev-1", "cred-dev", "10.8.0.1", models.DeviceStatusOnline, nil, strPtr("cred-1"))
	seedDevice(t, db, "dev-2", "nocred-dev", "10.8.0.2", models.DeviceStatusOnline, nil, nil)

	cache.On("GetOrSet", mock.Anything, "network_device:credential:cred-1", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	devices, err := svc.GetDevicesByCredential(context.Background(), "cred-1")
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, "cred-dev", devices[0].DeviceName)
	cache.AssertExpectations(t)
}

// ==================== Cache invalidation methods ====================

func TestNetworkService_InvalidateDeviceCache_DeletesKey(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	cache.On("Delete", mock.Anything, "network_device:detail:dev-42").Return(nil).Once()
	assert.NoError(t, svc.InvalidateDeviceCache(context.Background(), "dev-42"))
	cache.AssertExpectations(t)
}

func TestNetworkService_InvalidateStatisticsCache_DeletesKey(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	cache.On("Delete", mock.Anything, "network_device:statistics").Return(nil).Once()
	assert.NoError(t, svc.InvalidateStatisticsCache(context.Background()))
	cache.AssertExpectations(t)
}

func TestNetworkService_InvalidateDeptCache_DeletesKey(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	cache.On("Delete", mock.Anything, "network_device:dept:dept-42").Return(nil).Once()
	assert.NoError(t, svc.InvalidateDeptCache(context.Background(), "dept-42"))
	cache.AssertExpectations(t)
}

func TestNetworkService_InvalidateCredentialCache_DeletesKey(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	cache.On("Delete", mock.Anything, "network_device:credential:cred-42").Return(nil).Once()
	assert.NoError(t, svc.InvalidateCredentialCache(context.Background(), "cred-42"))
	cache.AssertExpectations(t)
}

func TestNetworkService_InvalidateAllDeviceCache_DeletesPattern(t *testing.T) {
	_, svc, cache := newNetworkTestService(t)
	cache.On("DeleteByPattern", mock.Anything, "network_device:*").Return(nil).Once()
	assert.NoError(t, svc.InvalidateAllDeviceCache(context.Background()))
	cache.AssertExpectations(t)
}

// ==================== getExpiration helper ====================

func TestNetworkService_GetExpiration_NilConfig_ReturnsDefault(t *testing.T) {
	_, svc, _ := newNetworkTestService(t)
	got := svc.getExpiration("cache.network_device.statistics", 3*time.Minute)
	assert.Equal(t, 3*time.Minute, got)
}

// toInt64 normalizes the map[string]interface{} statistics values.
func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return -1
	}
}
