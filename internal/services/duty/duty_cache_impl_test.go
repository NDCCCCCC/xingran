// =====================================================================
// duty_cache_impl_test.go — covers duty_cache_impl.go (114 stmts)
// Pattern: portwrite pure-mock (interface assertion + testify/mock, D-02).
// CacheProvider is fully mocked; base DutyService runs on glebarez
// sqlite in-memory (unavoidable gorm path — plan allows minimal sqlite).
// Per Phase 73 Plan 03 — IMP-05 (services/duty)
//
// Fixture notes (bugs fixed from prior partial run):
//   1. InvalidateCacheByKey loops cache.Delete(ctx, key); testify
//      m.Called PANICS on unexpected calls. Prior file registered Delete
//      expectations AFTER setup calls that already triggered invalidation
//      (e.g. CreateHoliday used as setup in UpdateHoliday/DeleteHoliday
//      tests) → panic at duty_cache_impl.go:227. Fix: seed rows via raw
//      db.Create instead of service calls, and register expectations
//      BEFORE any invalidation-triggering action (unbounded, no .Once()
//      when both setup and act invalidate the same key).
//   2. parseInt(req.StartDate[5:7]) returns 0 for 2-char months ("08" is
//      len < 4), so GenerateSchedule/ManualDuty invalidate
//      "duty:monthly:<year>:0", NOT "duty:monthly:<year>:8". Tests lock
//      ACTUAL behavior (read path uses real month; invalidation misses —
//      documented as a known business-code quirk, NOT fixed per D-12).
// =====================================================================

package duty

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// Compile-time interface assertion — locks mockability contract (D-02).
// If dutyCacheServiceImpl.cache field type drifts away from CacheProvider,
// this file fails to compile, catching regression at build time.
var _ systemServices.CacheProvider = (*mockCacheProvider)(nil)

// mockCacheProvider embeds mock.Mock and implements the CacheProvider
// interface. GetOrSet/Delete/DeleteByPattern go through m.Called (tests
// MUST register expectations — unexpected calls panic). The remaining
// methods are deterministic no-ops: no code path under test invokes them.
type mockCacheProvider struct {
	mock.Mock
}

// GetOrSet mock:
//   - error return → propagate WITHOUT invoking query (cache failure path)
//   - nil error → invoke query (cache miss → DB fallback), then populate
//     dest via reflection the same way NoOpCacheProvider.setValue does,
//     so service-level assertions observe real query data.
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
	if elem.IsValid() && vv.IsValid() && vv.Type().AssignableTo(elem.Type()) {
		elem.Set(vv)
	}
}

// newDutyTestDB creates a sqlite in-memory DB with every table the base
// DutyService sub-services touch (pool/member/schedule/exchange/config/
// holiday + sys_user for member validation & preloads).
func newDutyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.DutyPool{},
		&models.DutyPoolMember{},
		&models.DutySchedule{},
		&models.DutyExchange{},
		&models.DutyConfig{},
		&models.Holiday{},
		&models.User{},
	))
	return db
}

// newDutySvcOver wires a fresh cache impl + fresh mock cache over an
// existing DB. Multiple services can share one DB while keeping
// independent mock expectations per lifecycle phase (seed vs act).
func newDutySvcOver(db *gorm.DB) (*dutyCacheServiceImpl, *mockCacheProvider) {
	cache := new(mockCacheProvider)
	return &dutyCacheServiceImpl{
		base:   services.NewDutyService(db),
		cache:  cache,
		config: nil,
	}, cache
}

// newDutyTestService is the default one-shot fixture.
func newDutyTestService(t *testing.T) (*gorm.DB, *dutyCacheServiceImpl, *mockCacheProvider) {
	t.Helper()
	db := newDutyTestDB(t)
	svc, cache := newDutySvcOver(db)
	return db, svc, cache
}

// ---- raw seed helpers (db.Create — deliberately bypasses the cache impl
// so seeding never triggers cache invalidation expectations) ----

func seedDutyUser(t *testing.T, db *gorm.DB, id, username string) {
	t.Helper()
	u := &models.User{Username: username, Password: "test-pwd", Salt: "test-salt"}
	u.ID = id
	require.NoError(t, db.Create(u).Error)
}

func seedDutyPool(t *testing.T, db *gorm.DB, name string) *models.DutyPool {
	t.Helper()
	pool := &models.DutyPool{
		PoolName:   name,
		DailyCount: 1,
		Status:     models.DutyPoolStatusEnabled,
	}
	require.NoError(t, db.Create(pool).Error)
	return pool
}

func seedDutyMember(t *testing.T, db *gorm.DB, poolID, userID string, order int) {
	t.Helper()
	member := &models.DutyPoolMember{PoolID: poolID, UserID: userID, MemberOrder: order}
	require.NoError(t, db.Create(member).Error)
}

func seedDutySchedule(t *testing.T, db *gorm.DB, poolID, userID string, date time.Time) *models.DutySchedule {
	t.Helper()
	schedule := &models.DutySchedule{
		PoolID:       poolID,
		UserID:       userID,
		ScheduleDate: date,
		DutyType:     models.ScheduleModeWeekday,
		Status:       models.DutyStatusNormal,
		IsManual:     true,
	}
	require.NoError(t, db.Create(schedule).Error)
	return schedule
}

func seedHoliday(t *testing.T, db *gorm.DB, dateStr, name string, year int) *models.Holiday {
	t.Helper()
	date, err := time.Parse("2006-01-02", dateStr)
	require.NoError(t, err)
	holiday := &models.Holiday{
		HolidayDate: date,
		HolidayName: name,
		IsOffday:    true,
		HolidayType: models.HolidayTypeLegal,
		Year:        year,
	}
	require.NoError(t, db.Create(holiday).Error)
	return holiday
}

// assertNoCacheInteraction guards that uncached delegation paths never
// touch the cache provider.
func assertNoCacheInteraction(t *testing.T, cache *mockCacheProvider) {
	t.Helper()
	cache.AssertNotCalled(t, "GetOrSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	cache.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	cache.AssertNotCalled(t, "DeleteByPattern", mock.Anything, mock.Anything)
}

// ==================== Smoke / constructor ====================

// TestDutyService_CompileOnly — smoke test ensures file compiles + mocks match.
func TestDutyService_CompileOnly(t *testing.T) {
	svc, cache := newDutySvcOver(newDutyTestDB(t))
	assert.NotNil(t, svc)
	assert.NotNil(t, cache)
}

// TestDutyService_NewDutyServiceWithCache — constructor returns a
// DutyCacheService implementation.
func TestDutyService_NewDutyServiceWithCache(t *testing.T) {
	db := newDutyTestDB(t)
	var svc DutyCacheService = NewDutyServiceWithCache(db, new(mockCacheProvider), nil)
	assert.NotNil(t, svc)
}

// ==================== Duty pool management (uncached delegations) ====================

func TestDutyService_CreateDutyPool_Success(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	ctx := context.Background()
	seedDutyUser(t, db, "user-1", "alice")

	pool, err := svc.CreateDutyPool(ctx, &services.DutyPoolCreateRequest{
		PoolName:  "primary-oncall",
		DailyCount: 2,
		MemberIDs: []string{"user-1"},
	}, "creator-1")
	require.NoError(t, err)
	require.NotNil(t, pool)
	assert.Equal(t, "primary-oncall", pool.PoolName)
	assert.NotEmpty(t, pool.ID)

	var count int64
	require.NoError(t, db.Model(&models.DutyPool{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)

	assertNoCacheInteraction(t, cache)
}

func TestDutyService_CreateDutyPool_DuplicateName_Error(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	ctx := context.Background()
	seedDutyUser(t, db, "user-1", "alice")

	_, err := svc.CreateDutyPool(ctx, &services.DutyPoolCreateRequest{
		PoolName: "dup-pool", DailyCount: 1, MemberIDs: []string{"user-1"},
	}, "creator-1")
	require.NoError(t, err)

	_, err = svc.CreateDutyPool(ctx, &services.DutyPoolCreateRequest{
		PoolName: "dup-pool", DailyCount: 1, MemberIDs: []string{"user-1"},
	}, "creator-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "值班池名称已存在")
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_CreateDutyPool_MemberNotFound_Error(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	_ = db

	_, err := svc.CreateDutyPool(context.Background(), &services.DutyPoolCreateRequest{
		PoolName: "ghost-pool", DailyCount: 1, MemberIDs: []string{"missing-user"},
	}, "creator-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "用户不存在")
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_GetDutyPoolList_Success(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	seedDutyPool(t, db, "pool-a")
	seedDutyPool(t, db, "pool-b")

	list, total, err := svc.GetDutyPoolList(context.Background(), &services.DutyPoolListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, list, 2)
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_GetDutyPoolList_FilterByPoolName(t *testing.T) {
	db, svc, _ := newDutyTestService(t)
	seedDutyPool(t, db, "alpha-pool")
	seedDutyPool(t, db, "beta-pool")

	name := "alpha"
	list, total, err := svc.GetDutyPoolList(context.Background(), &services.DutyPoolListRequest{
		PoolName: &name,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, "alpha-pool", list[0].PoolName)
}

func TestDutyService_GetDutyPoolList_Empty(t *testing.T) {
	_, svc, _ := newDutyTestService(t)
	list, total, err := svc.GetDutyPoolList(context.Background(), &services.DutyPoolListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, list)
}

func TestDutyService_GetDutyPoolStatistics_Success(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "stats-pool")
	seedDutyUser(t, db, "user-1", "alice")
	seedDutyMember(t, db, pool.ID, "user-1", 0)

	stats, err := svc.GetDutyPoolStatistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, int64(1), stats.Total)
	assert.Equal(t, int64(1), stats.Enabled)
	assert.Equal(t, int64(0), stats.Disabled)
	assert.Equal(t, int64(1), stats.TotalMembers)
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_GetDutyPoolByID_Found(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "find-pool")

	found, err := svc.GetDutyPoolByID(context.Background(), pool.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, pool.ID, found.ID)
	assert.Equal(t, "find-pool", found.PoolName)
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_GetDutyPoolByID_NotFound_Error(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	found, err := svc.GetDutyPoolByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
	assert.Nil(t, found)
	assert.Contains(t, err.Error(), "值班池不存在")
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_UpdateDutyPool_Success(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "rename-pool")
	seedDutyUser(t, db, "user-1", "alice")

	err := svc.UpdateDutyPool(context.Background(), &services.DutyPoolUpdateRequest{
		ID:         pool.ID,
		PoolName:   "renamed-pool",
		DailyCount: 2,
		MemberIDs:  []string{"user-1"},
	}, "updater-1")
	require.NoError(t, err)

	var updated models.DutyPool
	require.NoError(t, db.Where("id = ?", pool.ID).First(&updated).Error)
	assert.Equal(t, "renamed-pool", updated.PoolName)
	assert.Equal(t, 2, updated.DailyCount)
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_UpdateDutyPool_NotFound_Error(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	err := svc.UpdateDutyPool(context.Background(), &services.DutyPoolUpdateRequest{
		ID:        uuid.NewString(),
		PoolName:  "ghost",
		DailyCount: 1,
		MemberIDs: []string{"any"},
	}, "updater-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "值班池不存在")
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_DeleteDutyPool_Success(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "delete-pool")
	seedDutyUser(t, db, "user-1", "alice")
	seedDutyMember(t, db, pool.ID, "user-1", 0)

	err := svc.DeleteDutyPool(context.Background(), pool.ID)
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.DutyPool{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_DeleteDutyPool_HasSchedules_Error(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "busy-pool")
	date, _ := time.Parse("2006-01-02", "2026-08-10")
	seedDutySchedule(t, db, pool.ID, "user-1", date)

	err := svc.DeleteDutyPool(context.Background(), pool.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "存在排班记录")
	assertNoCacheInteraction(t, cache)
}

// ==================== Schedule generation / listing ====================

func TestDutyService_GenerateSchedule_PoolNotFound_Error(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	count, err := svc.GenerateSchedule(context.Background(), &services.GenerateScheduleRequest{
		PoolID:    uuid.NewString(),
		StartDate: "2026-08-03",
		EndDate:   "2026-08-05",
		DutyType:  "weekday",
	}, "creator-1")
	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_GenerateSchedule_NoMembers_Error(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "empty-pool")

	count, err := svc.GenerateSchedule(context.Background(), &services.GenerateScheduleRequest{
		PoolID:    pool.ID,
		StartDate: "2026-08-03",
		EndDate:   "2026-08-05",
		DutyType:  "weekday",
	}, "creator-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "值班池没有成员")
	assert.Equal(t, 0, count)
	assertNoCacheInteraction(t, cache)
}

// TestDutyService_GenerateSchedule_Success_InvalidatesMonthlyCache —
// 2026-08-03..05 (Mon-Wed) weekday schedule for a 1-member pool → 3 rows.
//
// NOTE (quirk, locked as-is per D-12): the impl computes
// month = parseInt("08") = 0 (parseInt requires len >= 4), so it
// invalidates "duty:monthly:2026:0" — NOT "duty:monthly:2026:8" which is
// the key the read path (GetMonthlyDutySchedule) uses. The invalidation
// misses the real key; see SUMMARY deviations.
func TestDutyService_GenerateSchedule_Success_InvalidatesMonthlyCache(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	ctx := context.Background()
	pool := seedDutyPool(t, db, "gen-pool")
	seedDutyUser(t, db, "user-1", "alice")
	seedDutyMember(t, db, pool.ID, "user-1", 0)

	cache.On("Delete", mock.Anything, "duty:monthly:2026:0").Return(nil).Once()

	count, err := svc.GenerateSchedule(ctx, &services.GenerateScheduleRequest{
		PoolID:    pool.ID,
		StartDate: "2026-08-03",
		EndDate:   "2026-08-05",
		DutyType:  "weekday",
	}, "creator-1")
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	var schedules int64
	require.NoError(t, db.Model(&models.DutySchedule{}).Where("pool_id = ?", pool.ID).Count(&schedules).Error)
	assert.Equal(t, int64(3), schedules)
	cache.AssertExpectations(t)
}

func TestDutyService_GetDutyScheduleList_Empty(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	list, total, err := svc.GetDutyScheduleList(context.Background(), &services.DutyScheduleListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, list)
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_GetDutyScheduleList_Success(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "list-pool")
	date, _ := time.Parse("2006-01-02", "2026-08-10")
	seedDutySchedule(t, db, pool.ID, "user-1", date)
	seedDutySchedule(t, db, pool.ID, "user-2", date.AddDate(0, 0, 1))

	poolFilter := pool.ID
	list, total, err := svc.GetDutyScheduleList(context.Background(), &services.DutyScheduleListRequest{
		PoolID: &poolFilter,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, list, 2)
	assertNoCacheInteraction(t, cache)
}

// ==================== GetTodayDuty (cached) ====================

func TestDutyService_GetTodayDuty_CacheError_Propagates(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	wantErr := errors.New("redis connection refused")
	cache.On("GetOrSet", mock.Anything, "duty:today", mock.Anything, mock.Anything, mock.Anything).
		Return(wantErr).Once()

	members, err := svc.GetTodayDuty(context.Background())
	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, members)
	cache.AssertExpectations(t)
}

func TestDutyService_GetTodayDuty_NoDutyToday_Error(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	// GetOrSet success → query runs → empty DB → base error propagates.
	cache.On("GetOrSet", mock.Anything, "duty:today", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	members, err := svc.GetTodayDuty(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "今日无值班人员")
	assert.Nil(t, members)
	cache.AssertExpectations(t)
}

func TestDutyService_GetTodayDuty_Success(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "today-pool")
	seedDutyUser(t, db, "user-1", "alice")
	seedDutySchedule(t, db, pool.ID, "user-1", localNoonToday())

	cache.On("GetOrSet", mock.Anything, "duty:today", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	members, err := svc.GetTodayDuty(context.Background())
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, "user-1", members[0].UserID)
	assert.Equal(t, "today-pool", members[0].PoolName)
	assert.Equal(t, "weekday", members[0].DutyType)
	cache.AssertExpectations(t)
}

// ==================== GetMonthlyDutySchedule (cached) ====================

func TestDutyService_GetMonthlyDutySchedule_CacheError_Propagates(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	wantErr := errors.New("redis timeout")
	cache.On("GetOrSet", mock.Anything, "duty:monthly:2026:8", mock.Anything, mock.Anything, mock.Anything).
		Return(wantErr).Once()

	result, err := svc.GetMonthlyDutySchedule(context.Background(), 2026, 8)
	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, result)
	cache.AssertExpectations(t)
}

func TestDutyService_GetMonthlyDutySchedule_Success(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "monthly-pool")
	date, _ := time.Parse("2006-01-02", "2026-08-10")
	seedDutySchedule(t, db, pool.ID, "user-1", date)

	cache.On("GetOrSet", mock.Anything, "duty:monthly:2026:8", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	result, err := svc.GetMonthlyDutySchedule(context.Background(), 2026, 8)
	require.NoError(t, err)
	require.Contains(t, result, "2026-08-10")
	assert.Len(t, result["2026-08-10"], 1)
	cache.AssertExpectations(t)
}

func TestDutyService_GetMonthlyDutySchedule_EmptyMonth(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	cache.On("GetOrSet", mock.Anything, "duty:monthly:2026:9", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	result, err := svc.GetMonthlyDutySchedule(context.Background(), 2026, 9)
	require.NoError(t, err)
	assert.Empty(t, result)
	cache.AssertExpectations(t)
}

// ==================== SwapDuty ====================

func TestDutyService_SwapDuty_Success_InvalidatesTodayCache(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "swap-pool")
	date, _ := time.Parse("2006-01-02", "2026-08-10")
	s1 := seedDutySchedule(t, db, pool.ID, "user-1", date)
	s2 := seedDutySchedule(t, db, pool.ID, "user-2", date)

	cache.On("Delete", mock.Anything, "duty:today").Return(nil).Once()

	err := svc.SwapDuty(context.Background(), &services.SwapDutyRequest{
		FromScheduleID: s1.ID,
		ToScheduleID:   s2.ID,
		Reason:         "personal",
	}, "operator-1")
	require.NoError(t, err)

	// Verify users actually exchanged in DB
	var got1, got2 models.DutySchedule
	require.NoError(t, db.Where("id = ?", s1.ID).First(&got1).Error)
	require.NoError(t, db.Where("id = ?", s2.ID).First(&got2).Error)
	assert.Equal(t, "user-2", got1.UserID)
	assert.Equal(t, "user-1", got2.UserID)
	assert.Equal(t, models.DutyStatusExchanged, got1.Status)
	cache.AssertExpectations(t)
}

func TestDutyService_SwapDuty_FromScheduleMissing_Error(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	err := svc.SwapDuty(context.Background(), &services.SwapDutyRequest{
		FromScheduleID: "missing-1",
		ToScheduleID:   "missing-2",
	}, "operator-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "原排班记录不存在")
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_SwapDuty_ToScheduleMissing_Error(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "swap-pool-2")
	date, _ := time.Parse("2006-01-02", "2026-08-10")
	s1 := seedDutySchedule(t, db, pool.ID, "user-1", date)

	err := svc.SwapDuty(context.Background(), &services.SwapDutyRequest{
		FromScheduleID: s1.ID,
		ToScheduleID:   "missing-2",
	}, "operator-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "目标排班记录不存在")
	assertNoCacheInteraction(t, cache)
}

// ==================== ManualDuty ====================

// TestDutyService_ManualDuty_Success_InvalidatesMonthlyAndToday —
// month key is "duty:monthly:2026:0" due to the parseInt quirk (see file
// header note 2).
func TestDutyService_ManualDuty_Success_InvalidatesMonthlyAndToday(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "manual-pool")
	_ = db

	cache.On("Delete", mock.Anything, "duty:monthly:2026:0").Return(nil).Once()
	cache.On("Delete", mock.Anything, "duty:today").Return(nil).Once()

	err := svc.ManualDuty(context.Background(), &services.ManualDutyRequest{
		PoolID:   pool.ID,
		DutyDate: "2026-08-10",
		UserIDs:  []string{"user-1", "user-2"},
		DutyType: "weekday",
		Reason:   "coverage",
	}, "creator-1")
	require.NoError(t, err)
	cache.AssertExpectations(t)
}

func TestDutyService_ManualDuty_InvalidDate_Error_NoCacheCall(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "manual-pool-2")
	_ = db

	err := svc.ManualDuty(context.Background(), &services.ManualDutyRequest{
		PoolID:   pool.ID,
		DutyDate: "not-a-date",
		UserIDs:  []string{"user-1"},
		DutyType: "weekday",
	}, "creator-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "值班日期格式错误")
	assertNoCacheInteraction(t, cache)
}

// TestDutyService_ManualDuty_ShortDate_SkipsInvalidation — DutyDate len < 7
// still parses fine as date? No: "abc" fails parse (covered above). Use a
// short-but-parseable date scenario via len < 7 branch: dates always parse
// to len 10; use "2026-8-1" (len 8, parses) to hit the len>=7 branch with
// non-standard month slice, plus a len<7 string that still parses is
// impossible — instead lock the guard with an unparseable 6-char string.
func TestDutyService_ManualDuty_ShortDate_SkipsInvalidation(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "manual-pool-3")
	_ = db

	// "2026-8" (len 6 < 7): time.Parse fails first → error, and the
	// invalidation branch (len(req.DutyDate) >= 7) is never reached.
	err := svc.ManualDuty(context.Background(), &services.ManualDutyRequest{
		PoolID:   pool.ID,
		DutyDate: "2026-8",
		UserIDs:  []string{"user-1"},
		DutyType: "weekday",
	}, "creator-1")
	assert.Error(t, err)
	assertNoCacheInteraction(t, cache)
}

// ==================== DeleteDutySchedule / BatchDelete ====================

func TestDutyService_DeleteDutySchedule_Success_InvalidatesAllScheduleCache(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "del-sched-pool")
	date, _ := time.Parse("2006-01-02", "2026-08-10")
	schedule := seedDutySchedule(t, db, pool.ID, "user-1", date)

	cache.On("DeleteByPattern", mock.Anything, "duty:*").Return(nil).Once()

	err := svc.DeleteDutySchedule(context.Background(), schedule.ID)
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Unscoped().Model(&models.DutySchedule{}).Where("id = ?", schedule.ID).Count(&count).Error)
	assert.Equal(t, int64(1), count) // soft delete keeps the row
	cache.AssertExpectations(t)
}

// Soft-deleting a non-existent schedule is a no-op at DB level (no error),
// so invalidation still fires.
func TestDutyService_DeleteDutySchedule_MissingID_StillInvalidates(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	cache.On("DeleteByPattern", mock.Anything, "duty:*").Return(nil).Once()

	err := svc.DeleteDutySchedule(context.Background(), "missing-schedule")
	require.NoError(t, err)
	cache.AssertExpectations(t)
}

func TestDutyService_BatchDeleteDutySchedules_Success_InvalidatesAllScheduleCache(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "batch-del-pool")
	date, _ := time.Parse("2006-01-02", "2026-08-10")
	s1 := seedDutySchedule(t, db, pool.ID, "user-1", date)
	s2 := seedDutySchedule(t, db, pool.ID, "user-2", date.AddDate(0, 0, 1))

	cache.On("DeleteByPattern", mock.Anything, "duty:*").Return(nil).Once()

	err := svc.BatchDeleteDutySchedules(context.Background(), []string{s1.ID, s2.ID})
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.DutySchedule{}).Where("pool_id = ?", pool.ID).Count(&count).Error)
	assert.Equal(t, int64(0), count)
	cache.AssertExpectations(t)
}

// ==================== GetMyDutyStats (uncached delegation) ====================

func TestDutyService_GetMyDutyStats_Empty_ZeroCounts(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	stats, err := svc.GetMyDutyStats(context.Background(), "user-1")
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.False(t, stats.IsOnDutyToday)
	assert.Equal(t, 0, stats.ThisMonthCount)
	assert.Equal(t, 0, stats.TotalCount)
	assert.Nil(t, stats.NextDutyDate)
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_GetMyDutyStats_OnDutyToday(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	pool := seedDutyPool(t, db, "stats-pool")
	// GetMyDutyStats queries schedule_date = time.Now().Truncate(24h) —
	// seed the exact same instant so the equality matches.
	seedDutySchedule(t, db, pool.ID, "user-1", time.Now().Truncate(24*time.Hour))

	stats, err := svc.GetMyDutyStats(context.Background(), "user-1")
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.True(t, stats.IsOnDutyToday)
	assert.Equal(t, 1, stats.ThisMonthCount)
	assert.Equal(t, 1, stats.TotalCount)
	assert.NotNil(t, stats.TodayDutyRecords)
	assertNoCacheInteraction(t, cache)
}

// ==================== Holiday management (cached) ====================

func TestDutyService_CreateHoliday_Success_InvalidatesYearCache(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	cache.On("Delete", mock.Anything, "duty:holidays:2026").Return(nil).Once()

	err := svc.CreateHoliday(context.Background(), &models.Holiday{
		HolidayDate: mustDate(t, "2026-10-01"),
		HolidayName: "国庆节",
		IsOffday:    true,
		HolidayType: models.HolidayTypeLegal,
		Year:        2026,
	}, "creator-1")
	require.NoError(t, err)
	cache.AssertExpectations(t)
}

func TestDutyService_CreateHoliday_DuplicateDate_Error_NoCacheCall(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	seedHoliday(t, db, "2026-10-01", "国庆节", 2026)

	err := svc.CreateHoliday(context.Background(), &models.Holiday{
		HolidayDate: mustDate(t, "2026-10-01"),
		HolidayName: "国庆节-重复",
		IsOffday:    true,
		HolidayType: models.HolidayTypeCustom,
		Year:        2026,
	}, "creator-1")
	assert.Error(t, err) // unique index on holiday_date
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_GetHolidayList_CacheMiss_Success(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	seedHoliday(t, db, "2026-10-01", "国庆节", 2026)
	seedHoliday(t, db, "2026-10-02", "国庆节-第二天", 2026)

	cache.On("GetOrSet", mock.Anything, "duty:holidays:2026", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	list, err := svc.GetHolidayList(context.Background(), 2026)
	require.NoError(t, err)
	assert.Len(t, list, 2)
	cache.AssertExpectations(t)
}

func TestDutyService_GetHolidayList_CacheError_Propagates(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	wantErr := errors.New("cache write failed")
	cache.On("GetOrSet", mock.Anything, "duty:holidays:2026", mock.Anything, mock.Anything, mock.Anything).
		Return(wantErr).Once()

	list, err := svc.GetHolidayList(context.Background(), 2026)
	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, list)
	cache.AssertExpectations(t)
}

func TestDutyService_UpdateHoliday_Success_InvalidatesYearCache(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	holiday := seedHoliday(t, db, "2026-10-01", "国庆节", 2026)

	cache.On("Delete", mock.Anything, "duty:holidays:2026").Return(nil).Once()

	holiday.HolidayName = "国庆节-更新"
	err := svc.UpdateHoliday(context.Background(), holiday, "updater-1")
	require.NoError(t, err)
	cache.AssertExpectations(t)
}

func TestDutyService_DeleteHoliday_Success_InvalidatesAllHolidayCache(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	holiday := seedHoliday(t, db, "2026-10-01", "国庆节", 2026)

	cache.On("DeleteByPattern", mock.Anything, "duty:holidays:*").Return(nil).Once()

	err := svc.DeleteHoliday(context.Background(), holiday.ID)
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.Holiday{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
	cache.AssertExpectations(t)
}

func TestDutyService_GetHolidayYears_Success(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	seedHoliday(t, db, "2025-10-01", "国庆节-2025", 2025)
	seedHoliday(t, db, "2026-10-01", "国庆节", 2026)

	years, err := svc.GetHolidayYears(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []int{2026, 2025}, years) // descending
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_BatchCreateHolidays_Success_InvalidatesPerYear(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	// yearMap dedupes: 3 holidays across 2 years → exactly 2 invalidations.
	cache.On("Delete", mock.Anything, "duty:holidays:2025").Return(nil).Once()
	cache.On("Delete", mock.Anything, "duty:holidays:2026").Return(nil).Once()

	err := svc.BatchCreateHolidays(context.Background(), []models.Holiday{
		{HolidayDate: mustDate(t, "2025-10-01"), HolidayName: "h-2025-a", IsOffday: true, HolidayType: models.HolidayTypeCustom, Year: 2025},
		{HolidayDate: mustDate(t, "2026-10-01"), HolidayName: "h-2026", IsOffday: true, HolidayType: models.HolidayTypeCustom, Year: 2026},
		{HolidayDate: mustDate(t, "2025-10-02"), HolidayName: "h-2025-b", IsOffday: true, HolidayType: models.HolidayTypeCustom, Year: 2025},
	}, "creator-1")
	require.NoError(t, err)
	cache.AssertExpectations(t)
}

func TestDutyService_BatchCreateHolidays_DuplicateInBatch_Error_NoCacheCall(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	err := svc.BatchCreateHolidays(context.Background(), []models.Holiday{
		{HolidayDate: mustDate(t, "2026-10-01"), HolidayName: "dup-a", IsOffday: true, HolidayType: models.HolidayTypeCustom, Year: 2026},
		{HolidayDate: mustDate(t, "2026-10-01"), HolidayName: "dup-b", IsOffday: true, HolidayType: models.HolidayTypeCustom, Year: 2026},
	}, "creator-1")
	assert.Error(t, err) // unique holiday_date within one batch insert
	assertNoCacheInteraction(t, cache)
}

// ==================== DutyConfig (uncached delegation) ====================

func TestDutyService_GetDutyConfig_Default_WhenMissing(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	cfg, err := svc.GetDutyConfig(context.Background())
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.True(t, cfg.ReminderEnabled)
	assert.Equal(t, "08:00", cfg.ReminderTime)
	assert.Equal(t, "websocket", cfg.ReminderChannels)
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_GetDutyConfig_Existing(t *testing.T) {
	db, svc, _ := newDutyTestService(t)
	stored := &models.DutyConfig{ReminderEnabled: true, ReminderTime: "09:30", ReminderChannels: "email"}
	require.NoError(t, db.Create(stored).Error)
	// ReminderEnabled=false is a GORM zero value and would be skipped on
	// Create (column default true) — force it via explicit column update.
	require.NoError(t, db.Model(&models.DutyConfig{}).Where("id = ?", stored.ID).
		Update("reminder_enabled", false).Error)

	cfg, err := svc.GetDutyConfig(context.Background())
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, stored.ID, cfg.ID)
	assert.False(t, cfg.ReminderEnabled)
	assert.Equal(t, "09:30", cfg.ReminderTime)
}

func TestDutyService_UpdateDutyConfig_Creates_WhenMissing(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	err := svc.UpdateDutyConfig(context.Background(), &models.DutyConfig{
		ReminderEnabled:  true,
		ReminderTime:     "09:00",
		ReminderChannels: "websocket",
	}, "updater-1")
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.DutyConfig{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
	assertNoCacheInteraction(t, cache)
}

func TestDutyService_UpdateDutyConfig_Updates_WhenExists(t *testing.T) {
	db, svc, cache := newDutyTestService(t)
	stored := &models.DutyConfig{ReminderEnabled: true, ReminderTime: "08:00", ReminderChannels: "websocket"}
	require.NoError(t, db.Create(stored).Error)

	err := svc.UpdateDutyConfig(context.Background(), &models.DutyConfig{
		ReminderEnabled:  false,
		ReminderTime:     "10:15",
		ReminderChannels: "email,sms",
	}, "updater-1")
	require.NoError(t, err)

	var got models.DutyConfig
	require.NoError(t, db.First(&got).Error)
	assert.Equal(t, stored.ID, got.ID)
	assert.False(t, got.ReminderEnabled)
	assert.Equal(t, "10:15", got.ReminderTime)
	assert.Equal(t, "email,sms", got.ReminderChannels)
	assertNoCacheInteraction(t, cache)
}

// ==================== getExpiration helper ====================

func TestDutyService_GetExpiration_NilConfig_ReturnsDefault(t *testing.T) {
	_, svc, _ := newDutyTestService(t)
	got := svc.getExpiration("cache.duty.today", 5*time.Minute)
	assert.Equal(t, 5*time.Minute, got)
}

// ==================== Cache invalidation methods ====================

func TestDutyService_InvalidateTodayDutyCache_DeletesKey(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	cache.On("Delete", mock.Anything, "duty:today").Return(nil).Once()
	assert.NoError(t, svc.InvalidateTodayDutyCache(context.Background()))
	cache.AssertExpectations(t)
}

func TestDutyService_InvalidateMonthlyScheduleCache_DeletesKey(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	cache.On("Delete", mock.Anything, "duty:monthly:2026:8").Return(nil).Once()
	assert.NoError(t, svc.InvalidateMonthlyScheduleCache(context.Background(), 2026, 8))
	cache.AssertExpectations(t)
}

func TestDutyService_InvalidateAllScheduleCache_DeletesPattern(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	cache.On("DeleteByPattern", mock.Anything, "duty:*").Return(nil).Once()
	assert.NoError(t, svc.InvalidateAllScheduleCache(context.Background()))
	cache.AssertExpectations(t)
}

func TestDutyService_InvalidateHolidayCache_DeletesKey(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	cache.On("Delete", mock.Anything, "duty:holidays:2026").Return(nil).Once()
	assert.NoError(t, svc.InvalidateHolidayCache(context.Background(), 2026))
	cache.AssertExpectations(t)
}

func TestDutyService_InvalidateAllHolidayCache_DeletesPattern(t *testing.T) {
	_, svc, cache := newDutyTestService(t)
	cache.On("DeleteByPattern", mock.Anything, "duty:holidays:*").Return(nil).Once()
	assert.NoError(t, svc.InvalidateAllHolidayCache(context.Background()))
	cache.AssertExpectations(t)
}

// ==================== parseInt helper ====================

func TestDutyService_ParseInt(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty_string", "", 0},
		{"short_string_len2", "08", 0}, // len < 4 → 0 (quirk source)
		{"short_string_len3", "123", 0},
		{"valid_year", "2026", 2026},
		{"first4_of_longer", "20268", 2026},
		{"non_digit", "abcd", 0},
		{"mixed_digits", "2a4b", 24}, // '2'→2, 'a' skipped, '4'→24, 'b' skipped
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseInt(tt.input))
		})
	}
}

// mustDate parses YYYY-MM-DD or fails the test.
func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	require.NoError(t, err)
	return d
}

// localNoonToday returns today's date at 12:00 local. Chosen so that the
// UTC-converted date equals the local date for any |UTC offset| <= 12h —
// keeps DATE(schedule_date) = <local today> comparisons deterministic
// regardless of the machine timezone the suite runs on.
func localNoonToday() time.Time {
	y, m, d := time.Now().Local().Date()
	return time.Date(y, m, d, 12, 0, 0, 0, time.Local)
}
