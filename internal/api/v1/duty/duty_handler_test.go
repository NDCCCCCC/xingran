package duty

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	dutyServices "github.com/xingran-next/xingran-go-backend/internal/services/duty"
)

// Compile-time assertion: mockDutyService implements dutyServices.DutyCacheService
var _ dutyServices.DutyCacheService = (*mockDutyService)(nil)

// mockDutyService implements dutyServices.DutyCacheService via function fields.
// Embedding the interface as a nil field allows unimplemented methods to panic
// if accidentally invoked — we override every method explicitly below so this
// is structural, not behavioral.
type mockDutyService struct {
	dutyServices.DutyCacheService

	// Pool methods
	CreateDutyPoolFunc         func(ctx context.Context, req *services.DutyPoolCreateRequest, creatorID string) (*models.DutyPool, error)
	GetDutyPoolListFunc        func(ctx context.Context, req *services.DutyPoolListRequest) ([]models.DutyPool, int64, error)
	GetDutyPoolStatisticsFunc  func(ctx context.Context) (*services.DutyPoolStatistics, error)
	GetDutyPoolByIDFunc        func(ctx context.Context, poolID string) (*models.DutyPool, error)
	UpdateDutyPoolFunc         func(ctx context.Context, req *services.DutyPoolUpdateRequest, updaterID string) error
	DeleteDutyPoolFunc         func(ctx context.Context, poolID string) error

	// Schedule methods
	GenerateScheduleFunc        func(ctx context.Context, req *services.GenerateScheduleRequest, creatorID string) (int, error)
	GetDutyScheduleListFunc     func(ctx context.Context, req *services.DutyScheduleListRequest) ([]models.DutySchedule, int64, error)
	GetTodayDutyFunc            func(ctx context.Context) ([]services.TodayDutyMember, error)
	GetMonthlyDutyScheduleFunc  func(ctx context.Context, year, month int) (map[string][]services.TodayDutyMember, error)
	SwapDutyFunc                func(ctx context.Context, req *services.SwapDutyRequest, operatorID string) error
	ManualDutyFunc              func(ctx context.Context, req *services.ManualDutyRequest, creatorID string) error
	DeleteDutyScheduleFunc      func(ctx context.Context, scheduleID string) error
	BatchDeleteDutySchedulesFunc func(ctx context.Context, scheduleIDs []string) error

	// My stats
	GetMyDutyStatsFunc func(ctx context.Context, userID string) (*services.MyDutyStats, error)

	// Holiday methods
	CreateHolidayFunc          func(ctx context.Context, holiday *models.Holiday, creatorID string) error
	GetHolidayListFunc         func(ctx context.Context, year int) ([]models.Holiday, error)
	UpdateHolidayFunc          func(ctx context.Context, holiday *models.Holiday, updaterID string) error
	DeleteHolidayFunc          func(ctx context.Context, holidayID string) error
	GetHolidayYearsFunc        func(ctx context.Context) ([]int, error)
	BatchCreateHolidaysFunc    func(ctx context.Context, holidays []models.Holiday, creatorID string) error

	// Config methods
	GetDutyConfigFunc    func(ctx context.Context) (*models.DutyConfig, error)
	UpdateDutyConfigFunc func(ctx context.Context, config *models.DutyConfig, updaterID string) error

	// Cache invalidation (kept for completeness; not exercised by handlers)
	InvalidateTodayDutyCacheFunc       func(ctx context.Context) error
	InvalidateMonthlyScheduleCacheFunc  func(ctx context.Context, year, month int) error
	InvalidateAllScheduleCacheFunc      func(ctx context.Context) error
	InvalidateHolidayCacheFunc          func(ctx context.Context, year int) error
	InvalidateAllHolidayCacheFunc       func(ctx context.Context) error
}

// ==================== Pool method overrides ====================

func (m *mockDutyService) CreateDutyPool(ctx context.Context, req *services.DutyPoolCreateRequest, creatorID string) (*models.DutyPool, error) {
	if m.CreateDutyPoolFunc != nil {
		return m.CreateDutyPoolFunc(ctx, req, creatorID)
	}
	return &models.DutyPool{}, nil
}

func (m *mockDutyService) GetDutyPoolList(ctx context.Context, req *services.DutyPoolListRequest) ([]models.DutyPool, int64, error) {
	if m.GetDutyPoolListFunc != nil {
		return m.GetDutyPoolListFunc(ctx, req)
	}
	return nil, 0, nil
}

func (m *mockDutyService) GetDutyPoolStatistics(ctx context.Context) (*services.DutyPoolStatistics, error) {
	if m.GetDutyPoolStatisticsFunc != nil {
		return m.GetDutyPoolStatisticsFunc(ctx)
	}
	return &services.DutyPoolStatistics{}, nil
}

func (m *mockDutyService) GetDutyPoolByID(ctx context.Context, poolID string) (*models.DutyPool, error) {
	if m.GetDutyPoolByIDFunc != nil {
		return m.GetDutyPoolByIDFunc(ctx, poolID)
	}
	return &models.DutyPool{}, nil
}

func (m *mockDutyService) UpdateDutyPool(ctx context.Context, req *services.DutyPoolUpdateRequest, updaterID string) error {
	if m.UpdateDutyPoolFunc != nil {
		return m.UpdateDutyPoolFunc(ctx, req, updaterID)
	}
	return nil
}

func (m *mockDutyService) DeleteDutyPool(ctx context.Context, poolID string) error {
	if m.DeleteDutyPoolFunc != nil {
		return m.DeleteDutyPoolFunc(ctx, poolID)
	}
	return nil
}

// ==================== Schedule method overrides ====================

func (m *mockDutyService) GenerateSchedule(ctx context.Context, req *services.GenerateScheduleRequest, creatorID string) (int, error) {
	if m.GenerateScheduleFunc != nil {
		return m.GenerateScheduleFunc(ctx, req, creatorID)
	}
	return 0, nil
}

func (m *mockDutyService) GetDutyScheduleList(ctx context.Context, req *services.DutyScheduleListRequest) ([]models.DutySchedule, int64, error) {
	if m.GetDutyScheduleListFunc != nil {
		return m.GetDutyScheduleListFunc(ctx, req)
	}
	return nil, 0, nil
}

func (m *mockDutyService) GetTodayDuty(ctx context.Context) ([]services.TodayDutyMember, error) {
	if m.GetTodayDutyFunc != nil {
		return m.GetTodayDutyFunc(ctx)
	}
	return nil, nil
}

func (m *mockDutyService) GetMonthlyDutySchedule(ctx context.Context, year, month int) (map[string][]services.TodayDutyMember, error) {
	if m.GetMonthlyDutyScheduleFunc != nil {
		return m.GetMonthlyDutyScheduleFunc(ctx, year, month)
	}
	return nil, nil
}

func (m *mockDutyService) SwapDuty(ctx context.Context, req *services.SwapDutyRequest, operatorID string) error {
	if m.SwapDutyFunc != nil {
		return m.SwapDutyFunc(ctx, req, operatorID)
	}
	return nil
}

func (m *mockDutyService) ManualDuty(ctx context.Context, req *services.ManualDutyRequest, creatorID string) error {
	if m.ManualDutyFunc != nil {
		return m.ManualDutyFunc(ctx, req, creatorID)
	}
	return nil
}

func (m *mockDutyService) DeleteDutySchedule(ctx context.Context, scheduleID string) error {
	if m.DeleteDutyScheduleFunc != nil {
		return m.DeleteDutyScheduleFunc(ctx, scheduleID)
	}
	return nil
}

func (m *mockDutyService) BatchDeleteDutySchedules(ctx context.Context, scheduleIDs []string) error {
	if m.BatchDeleteDutySchedulesFunc != nil {
		return m.BatchDeleteDutySchedulesFunc(ctx, scheduleIDs)
	}
	return nil
}

// ==================== My stats ====================

func (m *mockDutyService) GetMyDutyStats(ctx context.Context, userID string) (*services.MyDutyStats, error) {
	if m.GetMyDutyStatsFunc != nil {
		return m.GetMyDutyStatsFunc(ctx, userID)
	}
	return &services.MyDutyStats{}, nil
}

// ==================== Holiday method overrides ====================

func (m *mockDutyService) CreateHoliday(ctx context.Context, holiday *models.Holiday, creatorID string) error {
	if m.CreateHolidayFunc != nil {
		return m.CreateHolidayFunc(ctx, holiday, creatorID)
	}
	return nil
}

func (m *mockDutyService) GetHolidayList(ctx context.Context, year int) ([]models.Holiday, error) {
	if m.GetHolidayListFunc != nil {
		return m.GetHolidayListFunc(ctx, year)
	}
	return nil, nil
}

func (m *mockDutyService) UpdateHoliday(ctx context.Context, holiday *models.Holiday, updaterID string) error {
	if m.UpdateHolidayFunc != nil {
		return m.UpdateHolidayFunc(ctx, holiday, updaterID)
	}
	return nil
}

func (m *mockDutyService) DeleteHoliday(ctx context.Context, holidayID string) error {
	if m.DeleteHolidayFunc != nil {
		return m.DeleteHolidayFunc(ctx, holidayID)
	}
	return nil
}

func (m *mockDutyService) GetHolidayYears(ctx context.Context) ([]int, error) {
	if m.GetHolidayYearsFunc != nil {
		return m.GetHolidayYearsFunc(ctx)
	}
	return nil, nil
}

func (m *mockDutyService) BatchCreateHolidays(ctx context.Context, holidays []models.Holiday, creatorID string) error {
	if m.BatchCreateHolidaysFunc != nil {
		return m.BatchCreateHolidaysFunc(ctx, holidays, creatorID)
	}
	return nil
}

// ==================== Config method overrides ====================

func (m *mockDutyService) GetDutyConfig(ctx context.Context) (*models.DutyConfig, error) {
	if m.GetDutyConfigFunc != nil {
		return m.GetDutyConfigFunc(ctx)
	}
	return &models.DutyConfig{}, nil
}

func (m *mockDutyService) UpdateDutyConfig(ctx context.Context, config *models.DutyConfig, updaterID string) error {
	if m.UpdateDutyConfigFunc != nil {
		return m.UpdateDutyConfigFunc(ctx, config, updaterID)
	}
	return nil
}

// ==================== Cache invalidation overrides ====================

func (m *mockDutyService) InvalidateTodayDutyCache(ctx context.Context) error {
	if m.InvalidateTodayDutyCacheFunc != nil {
		return m.InvalidateTodayDutyCacheFunc(ctx)
	}
	return nil
}

func (m *mockDutyService) InvalidateMonthlyScheduleCache(ctx context.Context, year, month int) error {
	if m.InvalidateMonthlyScheduleCacheFunc != nil {
		return m.InvalidateMonthlyScheduleCacheFunc(ctx, year, month)
	}
	return nil
}

func (m *mockDutyService) InvalidateAllScheduleCache(ctx context.Context) error {
	if m.InvalidateAllScheduleCacheFunc != nil {
		return m.InvalidateAllScheduleCacheFunc(ctx)
	}
	return nil
}

func (m *mockDutyService) InvalidateHolidayCache(ctx context.Context, year int) error {
	if m.InvalidateHolidayCacheFunc != nil {
		return m.InvalidateHolidayCacheFunc(ctx, year)
	}
	return nil
}

func (m *mockDutyService) InvalidateAllHolidayCache(ctx context.Context) error {
	if m.InvalidateAllHolidayCacheFunc != nil {
		return m.InvalidateAllHolidayCacheFunc(ctx)
	}
	return nil
}

// ==================== Test helpers ====================

// newTestCtxDuty creates a gin.Context with optional JSON body for handler tests.
func newTestCtxDuty(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c, w
}

// setupDutyHandler creates a handler wired to a mock service and minimal core.
// core.OperLogService is nil and core.DB is empty so operlog.Record early-returns.
func setupDutyHandler(mock *mockDutyService) *DutyHandler {
	return NewDutyHandler(mock).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	})
}

// ==================== Compile-only smoke ====================

func TestDutyHandler_CompileOnly(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	assert.NotNil(t, h)
}

// ==================== Pool tests ====================

func TestDutyHandler_ListPools_Success(t *testing.T) {
	mock := &mockDutyService{
		GetDutyPoolListFunc: func(ctx context.Context, req *services.DutyPoolListRequest) ([]models.DutyPool, int64, error) {
			return []models.DutyPool{{PoolName: "TestPool"}}, 1, nil
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/list", map[string]interface{}{})
	h.ListPools(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_ListPools_Error(t *testing.T) {
	mock := &mockDutyService{
		GetDutyPoolListFunc: func(ctx context.Context, req *services.DutyPoolListRequest) ([]models.DutyPool, int64, error) {
			return nil, 0, errors.New("db fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/list", map[string]interface{}{})
	h.ListPools(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_StatisticsPools_Success(t *testing.T) {
	mock := &mockDutyService{
		GetDutyPoolStatisticsFunc: func(ctx context.Context) (*services.DutyPoolStatistics, error) {
			return &services.DutyPoolStatistics{Total: 5, Enabled: 3, Disabled: 2, TotalMembers: 10}, nil
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/statistics", nil)
	h.StatisticsPools(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_StatisticsPools_Error(t *testing.T) {
	mock := &mockDutyService{
		GetDutyPoolStatisticsFunc: func(ctx context.Context) (*services.DutyPoolStatistics, error) {
			return nil, errors.New("stats fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/statistics", nil)
	h.StatisticsPools(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_CreatePool_Success(t *testing.T) {
	mock := &mockDutyService{
		CreateDutyPoolFunc: func(ctx context.Context, req *services.DutyPoolCreateRequest, creatorID string) (*models.DutyPool, error) {
			return &models.DutyPool{PoolName: req.PoolName}, nil
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/", services.DutyPoolCreateRequest{
		PoolName:   "MainPool",
		DailyCount: 1,
		MemberIDs:  []string{uuid.NewString()},
	})
	c.Set("user_id", "user-1")
	h.CreatePool(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_CreatePool_BindError(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/", map[string]interface{}{}) // missing required MemberIDs
	h.CreatePool(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_CreatePool_ServiceError(t *testing.T) {
	mock := &mockDutyService{
		CreateDutyPoolFunc: func(ctx context.Context, req *services.DutyPoolCreateRequest, creatorID string) (*models.DutyPool, error) {
			return nil, errors.New("create fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/", services.DutyPoolCreateRequest{
		PoolName:   "MainPool",
		DailyCount: 1,
		MemberIDs:  []string{uuid.NewString()},
	})
	c.Set("user_id", "user-1")
	h.CreatePool(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_GetPoolByID_Empty(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.GetPoolByID(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_GetPoolByID_Success(t *testing.T) {
	mock := &mockDutyService{
		GetDutyPoolByIDFunc: func(ctx context.Context, poolID string) (*models.DutyPool, error) {
			return &models.DutyPool{BaseModel: models.BaseModel{ID: poolID}, PoolName: "Fetched"}, nil
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.GetPoolByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_GetPoolByID_Error(t *testing.T) {
	mock := &mockDutyService{
		GetDutyPoolByIDFunc: func(ctx context.Context, poolID string) (*models.DutyPool, error) {
			return nil, errors.New("not found")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.GetPoolByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_UpdatePool_Empty(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/update", services.DutyPoolUpdateRequest{
		PoolName:   "X",
		DailyCount: 1,
		MemberIDs:  []string{uuid.NewString()},
	})
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.UpdatePool(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_UpdatePool_BindError(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/update", map[string]interface{}{})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.UpdatePool(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_UpdatePool_Success(t *testing.T) {
	mock := &mockDutyService{
		UpdateDutyPoolFunc: func(ctx context.Context, req *services.DutyPoolUpdateRequest, updaterID string) error {
			return nil
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/update", services.DutyPoolUpdateRequest{
		PoolName:   "Updated",
		DailyCount: 2,
		MemberIDs:  []string{uuid.NewString()},
	})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	c.Set("user_id", "user-1")
	h.UpdatePool(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_UpdatePool_ServiceError(t *testing.T) {
	mock := &mockDutyService{
		UpdateDutyPoolFunc: func(ctx context.Context, req *services.DutyPoolUpdateRequest, updaterID string) error {
			return errors.New("update fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/update", services.DutyPoolUpdateRequest{
		PoolName:   "Updated",
		DailyCount: 2,
		MemberIDs:  []string{uuid.NewString()},
	})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	c.Set("user_id", "user-1")
	h.UpdatePool(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_DeletePool_Empty(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.DeletePool(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_DeletePool_Success(t *testing.T) {
	mock := &mockDutyService{
		DeleteDutyPoolFunc: func(ctx context.Context, poolID string) error { return nil },
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.DeletePool(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_DeletePool_ServiceError(t *testing.T) {
	mock := &mockDutyService{
		DeleteDutyPoolFunc: func(ctx context.Context, poolID string) error {
			return errors.New("delete fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.DeletePool(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Schedule tests ====================

func TestDutyHandler_ListSchedules_Success(t *testing.T) {
	mock := &mockDutyService{
		GetDutyScheduleListFunc: func(ctx context.Context, req *services.DutyScheduleListRequest) ([]models.DutySchedule, int64, error) {
			return []models.DutySchedule{{}}, 1, nil
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/list", map[string]interface{}{})
	h.ListSchedules(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_ListSchedules_Error(t *testing.T) {
	mock := &mockDutyService{
		GetDutyScheduleListFunc: func(ctx context.Context, req *services.DutyScheduleListRequest) ([]models.DutySchedule, int64, error) {
			return nil, 0, errors.New("list fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/list", map[string]interface{}{})
	h.ListSchedules(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_GenerateSchedule_Success(t *testing.T) {
	mock := &mockDutyService{
		GenerateScheduleFunc: func(ctx context.Context, req *services.GenerateScheduleRequest, creatorID string) (int, error) {
			return 7, nil
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/generate", services.GenerateScheduleRequest{
		PoolID:    uuid.NewString(),
		StartDate: "2026-08-21",
		EndDate:   "2026-08-28",
		DutyType:  "weekday",
	})
	c.Set("user_id", "user-1")
	h.GenerateSchedule(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_GenerateSchedule_BindError(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/generate", map[string]interface{}{}) // missing required fields
	c.Set("user_id", "user-1")
	h.GenerateSchedule(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_GenerateSchedule_ServiceError(t *testing.T) {
	mock := &mockDutyService{
		GenerateScheduleFunc: func(ctx context.Context, req *services.GenerateScheduleRequest, creatorID string) (int, error) {
			return 0, errors.New("generate fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/generate", services.GenerateScheduleRequest{
		PoolID:    uuid.NewString(),
		StartDate: "2026-08-21",
		EndDate:   "2026-08-28",
		DutyType:  "weekday",
	})
	c.Set("user_id", "user-1")
	h.GenerateSchedule(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_GetTodayDuty_Success(t *testing.T) {
	mock := &mockDutyService{
		GetTodayDutyFunc: func(ctx context.Context) ([]services.TodayDutyMember, error) {
			return []services.TodayDutyMember{{UserID: "u1", Username: "alice"}}, nil
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/today", nil)
	h.GetTodayDuty(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_GetTodayDuty_Error(t *testing.T) {
	mock := &mockDutyService{
		GetTodayDutyFunc: func(ctx context.Context) ([]services.TodayDutyMember, error) {
			return nil, errors.New("today fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/today", nil)
	h.GetTodayDuty(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_GetMonthlySchedule_QueryParams_Success(t *testing.T) {
	mock := &mockDutyService{
		GetMonthlyDutyScheduleFunc: func(ctx context.Context, year, month int) (map[string][]services.TodayDutyMember, error) {
			return map[string][]services.TodayDutyMember{"2026-08-21": {{UserID: "u1"}}}, nil
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/monthly?year=2026&month=8", nil)
	h.GetMonthlySchedule(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_GetMonthlySchedule_JSONBody_Success(t *testing.T) {
	mock := &mockDutyService{
		GetMonthlyDutyScheduleFunc: func(ctx context.Context, year, month int) (map[string][]services.TodayDutyMember, error) {
			return map[string][]services.TodayDutyMember{"2026-08-21": {}}, nil
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/monthly", map[string]interface{}{"year": 2026, "month": 8})
	h.GetMonthlySchedule(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_GetMonthlySchedule_MissingParams(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/monthly", nil)
	h.GetMonthlySchedule(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_GetMonthlySchedule_InvalidYear(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/monthly?year=abc&month=8", nil)
	h.GetMonthlySchedule(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_GetMonthlySchedule_InvalidMonth(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/monthly?year=2026&month=13", nil)
	h.GetMonthlySchedule(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_GetMonthlySchedule_ServiceError(t *testing.T) {
	mock := &mockDutyService{
		GetMonthlyDutyScheduleFunc: func(ctx context.Context, year, month int) (map[string][]services.TodayDutyMember, error) {
			return nil, errors.New("monthly fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/monthly?year=2026&month=8", nil)
	h.GetMonthlySchedule(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_SwapDuty_Success(t *testing.T) {
	mock := &mockDutyService{
		SwapDutyFunc: func(ctx context.Context, req *services.SwapDutyRequest, operatorID string) error { return nil },
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/swap", services.SwapDutyRequest{
		FromScheduleID: uuid.NewString(),
		ToScheduleID:   uuid.NewString(),
	})
	c.Set("user_id", "user-1")
	h.SwapDuty(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_SwapDuty_BindError(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/swap", map[string]interface{}{}) // missing required IDs
	c.Set("user_id", "user-1")
	h.SwapDuty(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_SwapDuty_ServiceError(t *testing.T) {
	mock := &mockDutyService{
		SwapDutyFunc: func(ctx context.Context, req *services.SwapDutyRequest, operatorID string) error {
			return errors.New("swap fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/swap", services.SwapDutyRequest{
		FromScheduleID: uuid.NewString(),
		ToScheduleID:   uuid.NewString(),
	})
	c.Set("user_id", "user-1")
	h.SwapDuty(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_ManualDuty_Success(t *testing.T) {
	mock := &mockDutyService{
		ManualDutyFunc: func(ctx context.Context, req *services.ManualDutyRequest, creatorID string) error { return nil },
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/manual", services.ManualDutyRequest{
		PoolID:   uuid.NewString(),
		DutyDate: "2026-08-21",
		UserIDs:  []string{uuid.NewString()},
		DutyType: "weekday",
	})
	c.Set("user_id", "user-1")
	h.ManualDuty(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_ManualDuty_BindError(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/manual", map[string]interface{}{})
	c.Set("user_id", "user-1")
	h.ManualDuty(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_ManualDuty_ServiceError(t *testing.T) {
	mock := &mockDutyService{
		ManualDutyFunc: func(ctx context.Context, req *services.ManualDutyRequest, creatorID string) error {
			return errors.New("manual fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/manual", services.ManualDutyRequest{
		PoolID:   uuid.NewString(),
		DutyDate: "2026-08-21",
		UserIDs:  []string{uuid.NewString()},
		DutyType: "weekday",
	})
	c.Set("user_id", "user-1")
	h.ManualDuty(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_DeleteSchedule_Empty(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.DeleteSchedule(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_DeleteSchedule_Success(t *testing.T) {
	mock := &mockDutyService{
		DeleteDutyScheduleFunc: func(ctx context.Context, scheduleID string) error { return nil },
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.DeleteSchedule(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_DeleteSchedule_ServiceError(t *testing.T) {
	mock := &mockDutyService{
		DeleteDutyScheduleFunc: func(ctx context.Context, scheduleID string) error {
			return errors.New("del fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.DeleteSchedule(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_BatchDeleteSchedules_Success(t *testing.T) {
	mock := &mockDutyService{
		BatchDeleteDutySchedulesFunc: func(ctx context.Context, ids []string) error { return nil },
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/batch-delete", map[string]interface{}{
		"ids": []string{uuid.NewString(), uuid.NewString()},
	})
	h.BatchDeleteSchedules(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_BatchDeleteSchedules_BindError(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/batch-delete", map[string]interface{}{}) // missing ids
	h.BatchDeleteSchedules(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_BatchDeleteSchedules_ServiceError(t *testing.T) {
	mock := &mockDutyService{
		BatchDeleteDutySchedulesFunc: func(ctx context.Context, ids []string) error {
			return errors.New("batch fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/batch-delete", map[string]interface{}{
		"ids": []string{uuid.NewString()},
	})
	h.BatchDeleteSchedules(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Holiday tests ====================

func TestDutyHandler_ListHolidays_Success(t *testing.T) {
	mock := &mockDutyService{
		GetHolidayListFunc: func(ctx context.Context, year int) ([]models.Holiday, error) {
			return []models.Holiday{{HolidayName: "国庆"}}, nil
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/list", map[string]interface{}{"year": 2026})
	h.ListHolidays(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_ListHolidays_ServiceError(t *testing.T) {
	mock := &mockDutyService{
		GetHolidayListFunc: func(ctx context.Context, year int) ([]models.Holiday, error) {
			return nil, errors.New("holiday list fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/list", map[string]interface{}{"year": 2026})
	h.ListHolidays(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_CreateHoliday_Success(t *testing.T) {
	mock := &mockDutyService{
		CreateHolidayFunc: func(ctx context.Context, holiday *models.Holiday, creatorID string) error { return nil },
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/", map[string]interface{}{
		"holidayDate": "2026-10-01",
		"holidayName": "国庆节",
		"holidayType": "legal",
		"year":        2026,
	})
	c.Set("user_id", "user-1")
	h.CreateHoliday(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_CreateHoliday_BindError(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/", map[string]interface{}{}) // missing required fields
	c.Set("user_id", "user-1")
	h.CreateHoliday(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_CreateHoliday_BadDate(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/", map[string]interface{}{
		"holidayDate": "not-a-date",
		"holidayName": "国庆节",
		"holidayType": "legal",
		"year":        2026,
	})
	c.Set("user_id", "user-1")
	h.CreateHoliday(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_CreateHoliday_ServiceError(t *testing.T) {
	mock := &mockDutyService{
		CreateHolidayFunc: func(ctx context.Context, holiday *models.Holiday, creatorID string) error {
			return errors.New("create holiday fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/", map[string]interface{}{
		"holidayDate": "2026-10-01",
		"holidayName": "国庆节",
		"holidayType": "legal",
		"year":        2026,
	})
	c.Set("user_id", "user-1")
	h.CreateHoliday(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_UpdateHoliday_Empty(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/update", map[string]interface{}{})
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.UpdateHoliday(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_UpdateHoliday_BindError(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/update", "not-json")
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.UpdateHoliday(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_DeleteHoliday_Empty(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.DeleteHoliday(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_DeleteHoliday_Success(t *testing.T) {
	mock := &mockDutyService{
		DeleteHolidayFunc: func(ctx context.Context, holidayID string) error { return nil },
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.DeleteHoliday(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_DeleteHoliday_ServiceError(t *testing.T) {
	mock := &mockDutyService{
		DeleteHolidayFunc: func(ctx context.Context, holidayID string) error {
			return errors.New("del holiday fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.DeleteHoliday(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_BatchCreateHolidays_Success(t *testing.T) {
	mock := &mockDutyService{
		BatchCreateHolidaysFunc: func(ctx context.Context, holidays []models.Holiday, creatorID string) error { return nil },
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/batch", map[string]interface{}{
		"holidays": []map[string]interface{}{
			{"holidayDate": "2026-10-01", "holidayName": "国庆节", "holidayType": "legal", "year": 2026},
			{"holidayDate": "2026-10-02", "holidayName": "国庆节", "holidayType": "legal", "year": 2026},
		},
	})
	c.Set("user_id", "user-1")
	h.BatchCreateHolidays(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_BatchCreateHolidays_BindError(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/batch", map[string]interface{}{}) // missing required holidays
	c.Set("user_id", "user-1")
	h.BatchCreateHolidays(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_BatchCreateHolidays_BadDate(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/batch", map[string]interface{}{
		"holidays": []map[string]interface{}{
			{"holidayDate": "bad-date", "holidayName": "X", "holidayType": "legal", "year": 2026},
		},
	})
	c.Set("user_id", "user-1")
	h.BatchCreateHolidays(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDutyHandler_BatchCreateHolidays_ServiceError(t *testing.T) {
	mock := &mockDutyService{
		BatchCreateHolidaysFunc: func(ctx context.Context, holidays []models.Holiday, creatorID string) error {
			return errors.New("batch fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/batch", map[string]interface{}{
		"holidays": []map[string]interface{}{
			{"holidayDate": "2026-10-01", "holidayName": "国庆节", "holidayType": "legal", "year": 2026},
		},
	})
	c.Set("user_id", "user-1")
	h.BatchCreateHolidays(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_GetHolidayYears_Success(t *testing.T) {
	mock := &mockDutyService{
		GetHolidayYearsFunc: func(ctx context.Context) ([]int, error) {
			return []int{2025, 2026}, nil
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/years", nil)
	h.GetHolidayYears(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_GetHolidayYears_Error(t *testing.T) {
	mock := &mockDutyService{
		GetHolidayYearsFunc: func(ctx context.Context) ([]int, error) {
			return nil, errors.New("years fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/years", nil)
	h.GetHolidayYears(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Config tests ====================

func TestDutyHandler_GetConfig_Success(t *testing.T) {
	mock := &mockDutyService{
		GetDutyConfigFunc: func(ctx context.Context) (*models.DutyConfig, error) {
			return &models.DutyConfig{ReminderEnabled: true, ReminderTime: "08:00"}, nil
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/", nil)
	h.GetConfig(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_GetConfig_Error(t *testing.T) {
	mock := &mockDutyService{
		GetDutyConfigFunc: func(ctx context.Context) (*models.DutyConfig, error) {
			return nil, errors.New("config fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/", nil)
	h.GetConfig(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDutyHandler_UpdateConfig_Success(t *testing.T) {
	mock := &mockDutyService{
		UpdateDutyConfigFunc: func(ctx context.Context, config *models.DutyConfig, updaterID string) error { return nil },
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/update", models.DutyConfig{
		ReminderEnabled: true,
		ReminderTime:    "09:00",
	})
	c.Set("user_id", "user-1")
	h.UpdateConfig(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_UpdateConfig_ServiceError(t *testing.T) {
	mock := &mockDutyService{
		UpdateDutyConfigFunc: func(ctx context.Context, config *models.DutyConfig, updaterID string) error {
			return errors.New("update config fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/update", models.DutyConfig{
		ReminderEnabled: true,
	})
	c.Set("user_id", "user-1")
	h.UpdateConfig(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== My stats tests ====================

func TestDutyHandler_GetMyStats_NoUserID(t *testing.T) {
	mock := &mockDutyService{}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/stats", nil)
	h.GetMyStats(c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDutyHandler_GetMyStats_Success(t *testing.T) {
	mock := &mockDutyService{
		GetMyDutyStatsFunc: func(ctx context.Context, userID string) (*services.MyDutyStats, error) {
			return &services.MyDutyStats{IsOnDutyToday: true, ThisMonthCount: 3, TotalCount: 12}, nil
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/stats", nil)
	c.Set("user_id", "user-1")
	h.GetMyStats(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDutyHandler_GetMyStats_Error(t *testing.T) {
	mock := &mockDutyService{
		GetMyDutyStatsFunc: func(ctx context.Context, userID string) (*services.MyDutyStats, error) {
			return nil, errors.New("stats fail")
		},
	}
	h := setupDutyHandler(mock)
	c, w := newTestCtxDuty("POST", "/stats", nil)
	c.Set("user_id", "user-1")
	h.GetMyStats(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== WithCore nil-safety ====================

func TestDutyHandler_WithCore_NilSafety(t *testing.T) {
	h := &DutyHandler{}
	// With nil core should be no-op and return same handler.
	result := h.WithCore(nil)
	assert.Same(t, h, result)

	// WithCore with non-nil core wires it in.
	realCore := &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	}
	result2 := h.WithCore(realCore)
	assert.Same(t, h, result2)
	assert.Equal(t, realCore, h.core)
}

// Compile-time guard: ensure we never accidentally drop the unused imports.
var _ = time.Now