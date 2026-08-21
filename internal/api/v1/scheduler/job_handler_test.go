package scheduler

// =============================================================================
// JobHandler 测试 (Phase 72 CORE-03)
// =============================================================================
//
// 目标: 覆盖 internal/api/v1/scheduler/job_handler.go 的所有方法,
//       将包覆盖率从 0% 提升到 >= 70%。
//
// 范本 (D-01 lightweight handler pattern):
//   - function-field mock (mockJobService + mockJobLogService)
//   - httptest.NewRecorder + gin.CreateTestContext
//   - 表驱动 TC1/TC2/... 命名
// =============================================================================

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/common"
	schedulerServices "github.com/xingran-next/xingran-go-backend/internal/services/scheduler"
)

// ==================== Mock JobService ====================

type mockJobService struct {
	schedulerServices.JobService

	CreateFunc       func(ctx context.Context, req *schedulerServices.JobCreateRequest) (*models.Job, error)
	UpdateFunc       func(ctx context.Context, req *schedulerServices.JobUpdateRequest) error
	DeleteFunc       func(ctx context.Context, id string) error
	GetByIDFunc      func(ctx context.Context, id string) (*models.Job, error)
	ListFunc         func(ctx context.Context, params *schedulerServices.JobListParams) (*common.PageResult, error)
	UpdateStatusFunc func(ctx context.Context, id string, status int) error
	ExecuteFunc      func(ctx context.Context, id string) error
}

func (m *mockJobService) Create(ctx context.Context, req *schedulerServices.JobCreateRequest) (*models.Job, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return nil, nil
}
func (m *mockJobService) Update(ctx context.Context, req *schedulerServices.JobUpdateRequest) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, req)
	}
	return nil
}
func (m *mockJobService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}
func (m *mockJobService) GetByID(ctx context.Context, id string) (*models.Job, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockJobService) List(ctx context.Context, params *schedulerServices.JobListParams) (*common.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, params)
	}
	return nil, nil
}
func (m *mockJobService) UpdateStatus(ctx context.Context, id string, status int) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, status)
	}
	return nil
}
func (m *mockJobService) Execute(ctx context.Context, id string) error {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, id)
	}
	return nil
}

var _ schedulerServices.JobService = (*mockJobService)(nil)

// ==================== Mock JobLogService ====================

type mockJobLogService struct {
	schedulerServices.JobLogService

	CreateFunc       func(ctx context.Context, log *models.JobLog) error
	ListFunc         func(ctx context.Context, params *schedulerServices.JobLogListParams) (*common.PageResult, error)
	StatisticsFunc   func(ctx context.Context, params *schedulerServices.JobLogListParams) (*schedulerServices.JobLogStatistics, error)
	CleanOldLogsFunc func(ctx context.Context, days int) error
}

func (m *mockJobLogService) Create(ctx context.Context, log *models.JobLog) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, log)
	}
	return nil
}
func (m *mockJobLogService) List(ctx context.Context, params *schedulerServices.JobLogListParams) (*common.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, params)
	}
	return nil, nil
}
func (m *mockJobLogService) Statistics(ctx context.Context, params *schedulerServices.JobLogListParams) (*schedulerServices.JobLogStatistics, error) {
	if m.StatisticsFunc != nil {
		return m.StatisticsFunc(ctx, params)
	}
	return nil, nil
}
func (m *mockJobLogService) CleanOldLogs(ctx context.Context, days int) error {
	if m.CleanOldLogsFunc != nil {
		return m.CleanOldLogsFunc(ctx, days)
	}
	return nil
}

var _ schedulerServices.JobLogService = (*mockJobLogService)(nil)

// ==================== Test Infrastructure ====================

func newTestCtx(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

func parseResp(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

func setupHandler(jobSvc *mockJobService, jobLogSvc *mockJobLogService) *JobHandler {
	return NewJobHandler(jobSvc, jobLogSvc).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{},
	})
}

// ==================== Test Cases ====================

// TC1: Create - success
func TestJobHandler_Create_Success(t *testing.T) {
	jobSvc := &mockJobService{
		CreateFunc: func(ctx context.Context, req *schedulerServices.JobCreateRequest) (*models.Job, error) {
			return &models.Job{JobName: req.JobName}, nil
		},
	}
	h := setupHandler(jobSvc, &mockJobLogService{})
	c, w := newTestCtx("POST", "/", map[string]interface{}{
		"jobName":        "test-job",
		"jobGroup":       "DEFAULT",
		"invokeTarget":   "test.func",
		"cronExpression": "0 0 0 * * *",
	})
	h.Create(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC2: Create - invalid body
func TestJobHandler_Create_InvalidBody(t *testing.T) {
	h := setupHandler(&mockJobService{}, &mockJobLogService{})
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	h.Create(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TC3: Create - service error
func TestJobHandler_Create_ServiceError(t *testing.T) {
	jobSvc := &mockJobService{
		CreateFunc: func(ctx context.Context, req *schedulerServices.JobCreateRequest) (*models.Job, error) {
			return nil, errors.New("create fail")
		},
	}
	h := setupHandler(jobSvc, &mockJobLogService{})
	c, w := newTestCtx("POST", "/", map[string]interface{}{
		"jobName":        "test-job",
		"jobGroup":       "DEFAULT",
		"invokeTarget":   "test.func",
		"cronExpression": "0 0 0 * * *",
	})
	h.Create(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
	resp := parseResp(t, w.Body.Bytes())
	assert.NotEqualValues(t, 0, resp["code"])
}

// TC4: List - empty
func TestJobHandler_List_Empty(t *testing.T) {
	jobSvc := &mockJobService{
		ListFunc: func(ctx context.Context, params *schedulerServices.JobListParams) (*common.PageResult, error) {
			return &common.PageResult{List: []models.Job{}, Total: 0, Current: 1, PageSize: 10}, nil
		},
	}
	h := setupHandler(jobSvc, &mockJobLogService{})
	c, w := newTestCtx("POST", "/", map[string]interface{}{})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC5: List - with filters
func TestJobHandler_List_WithFilters(t *testing.T) {
	jobSvc := &mockJobService{
		ListFunc: func(ctx context.Context, params *schedulerServices.JobListParams) (*common.PageResult, error) {
			return &common.PageResult{List: []models.Job{{JobName: "test"}}, Total: 1, Current: 2, PageSize: 20}, nil
		},
	}
	h := setupHandler(jobSvc, &mockJobLogService{})
	c, w := newTestCtx("POST", "/", map[string]interface{}{
		"current":   2.0,
		"pageSize":  20.0,
		"jobName":   "test",
		"jobGroup":  "DEFAULT",
		"status":    0.0,
	})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC6: List - invalid body
func TestJobHandler_List_InvalidBody(t *testing.T) {
	jobSvc := &mockJobService{
		ListFunc: func(ctx context.Context, params *schedulerServices.JobListParams) (*common.PageResult, error) {
			return &common.PageResult{List: []models.Job{}, Total: 0, Current: 1, PageSize: 10}, nil
		},
	}
	h := setupHandler(jobSvc, &mockJobLogService{})
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	h.List(c)
	// invalid body 走 default empty map
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC7: List - service error
func TestJobHandler_List_ServiceError(t *testing.T) {
	jobSvc := &mockJobService{
		ListFunc: func(ctx context.Context, params *schedulerServices.JobListParams) (*common.PageResult, error) {
			return nil, errors.New("list fail")
		},
	}
	h := setupHandler(jobSvc, &mockJobLogService{})
	c, w := newTestCtx("POST", "/", map[string]interface{}{})
	h.List(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC8: GetByID - empty id
func TestJobHandler_GetByID_EmptyID(t *testing.T) {
	h := setupHandler(&mockJobService{}, &mockJobLogService{})
	c, w := newTestCtx("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.GetByID(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TC9: GetByID - success
func TestJobHandler_GetByID_Success(t *testing.T) {
	id := uuid.NewString()
	jobSvc := &mockJobService{
		GetByIDFunc: func(ctx context.Context, gotID string) (*models.Job, error) {
			return &models.Job{JobName: "test", BaseModel: models.BaseModel{ID: gotID}}, nil
		},
	}
	h := setupHandler(jobSvc, &mockJobLogService{})
	c, w := newTestCtx("POST", "/"+id, nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	h.GetByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC10: GetByID - not found
func TestJobHandler_GetByID_NotFound(t *testing.T) {
	jobSvc := &mockJobService{
		GetByIDFunc: func(ctx context.Context, id string) (*models.Job, error) {
			return nil, errors.New("not found")
		},
	}
	h := setupHandler(jobSvc, &mockJobLogService{})
	c, w := newTestCtx("POST", "/missing", nil)
	c.Params = gin.Params{{Key: "id", Value: "missing"}}
	h.GetByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC11: Update - empty id
func TestJobHandler_Update_EmptyID(t *testing.T) {
	h := setupHandler(&mockJobService{}, &mockJobLogService{})
	c, w := newTestCtx("POST", "/", map[string]interface{}{"jobName": "x"})
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Update(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TC12: Update - success
func TestJobHandler_Update_Success(t *testing.T) {
	id := uuid.NewString()
	h := setupHandler(&mockJobService{}, &mockJobLogService{})
	c, w := newTestCtx("POST", "/"+id, map[string]interface{}{"jobName": "updated"})
	c.Params = gin.Params{{Key: "id", Value: id}}
	h.Update(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC13: Update - service error
func TestJobHandler_Update_ServiceError(t *testing.T) {
	id := uuid.NewString()
	jobSvc := &mockJobService{
		UpdateFunc: func(ctx context.Context, req *schedulerServices.JobUpdateRequest) error {
			return errors.New("update fail")
		},
	}
	h := setupHandler(jobSvc, &mockJobLogService{})
	c, w := newTestCtx("POST", "/"+id, map[string]interface{}{"jobName": "x"})
	c.Params = gin.Params{{Key: "id", Value: id}}
	h.Update(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC14: Delete - empty id
func TestJobHandler_Delete_EmptyID(t *testing.T) {
	h := setupHandler(&mockJobService{}, &mockJobLogService{})
	c, w := newTestCtx("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Delete(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TC15: Delete - success
func TestJobHandler_Delete_Success(t *testing.T) {
	id := uuid.NewString()
	h := setupHandler(&mockJobService{}, &mockJobLogService{})
	c, w := newTestCtx("POST", "/"+id, nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	h.Delete(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC16: Delete - service error
func TestJobHandler_Delete_ServiceError(t *testing.T) {
	jobSvc := &mockJobService{
		DeleteFunc: func(ctx context.Context, id string) error { return errors.New("del fail") },
	}
	h := setupHandler(jobSvc, &mockJobLogService{})
	c, w := newTestCtx("POST", "/"+uuid.NewString(), nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Delete(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC17: UpdateStatus - empty id
func TestJobHandler_UpdateStatus_EmptyID(t *testing.T) {
	h := setupHandler(&mockJobService{}, &mockJobLogService{})
	c, w := newTestCtx("POST", "/", map[string]interface{}{"status": 1})
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.UpdateStatus(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TC18: UpdateStatus - enable (status=0)
func TestJobHandler_UpdateStatus_Enable(t *testing.T) {
	h := setupHandler(&mockJobService{}, &mockJobLogService{})
	c, w := newTestCtx("POST", "/"+uuid.NewString(), map[string]interface{}{"status": 0})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.UpdateStatus(c)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResp(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "启用成功", data["message"])
}

// TC19: UpdateStatus - pause (status=1)
func TestJobHandler_UpdateStatus_Pause(t *testing.T) {
	h := setupHandler(&mockJobService{}, &mockJobLogService{})
	c, w := newTestCtx("POST", "/"+uuid.NewString(), map[string]interface{}{"status": 1})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.UpdateStatus(c)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResp(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "暂停成功", data["message"])
}

// TC20: UpdateStatus - service error
func TestJobHandler_UpdateStatus_ServiceError(t *testing.T) {
	jobSvc := &mockJobService{
		UpdateStatusFunc: func(ctx context.Context, id string, status int) error {
			return errors.New("status fail")
		},
	}
	h := setupHandler(jobSvc, &mockJobLogService{})
	c, w := newTestCtx("POST", "/"+uuid.NewString(), map[string]interface{}{"status": 1})
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.UpdateStatus(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC21: Execute - empty id
func TestJobHandler_Execute_EmptyID(t *testing.T) {
	h := setupHandler(&mockJobService{}, &mockJobLogService{})
	c, w := newTestCtx("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Execute(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TC22: Execute - success
func TestJobHandler_Execute_Success(t *testing.T) {
	h := setupHandler(&mockJobService{}, &mockJobLogService{})
	c, w := newTestCtx("POST", "/"+uuid.NewString(), nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Execute(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC23: Execute - service error
func TestJobHandler_Execute_ServiceError(t *testing.T) {
	jobSvc := &mockJobService{
		ExecuteFunc: func(ctx context.Context, id string) error { return errors.New("exec fail") },
	}
	h := setupHandler(jobSvc, &mockJobLogService{})
	c, w := newTestCtx("POST", "/"+uuid.NewString(), nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Execute(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC24: Statistics - success
func TestJobHandler_Statistics_Success(t *testing.T) {
	jobLogSvc := &mockJobLogService{
		StatisticsFunc: func(ctx context.Context, params *schedulerServices.JobLogListParams) (*schedulerServices.JobLogStatistics, error) {
			return &schedulerServices.JobLogStatistics{Total: 100, Success: 80, Fail: 20}, nil
		},
	}
	h := setupHandler(&mockJobService{}, jobLogSvc)
	c, w := newTestCtx("POST", "/", map[string]interface{}{"jobName": "test", "jobGroup": "DEFAULT"})
	h.Statistics(c)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResp(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	assert.EqualValues(t, 100, data["total"])
	assert.EqualValues(t, 80, data["success"])
}

// TC25: Statistics - service error
func TestJobHandler_Statistics_ServiceError(t *testing.T) {
	jobLogSvc := &mockJobLogService{
		StatisticsFunc: func(ctx context.Context, params *schedulerServices.JobLogListParams) (*schedulerServices.JobLogStatistics, error) {
			return nil, errors.New("stats fail")
		},
	}
	h := setupHandler(&mockJobService{}, jobLogSvc)
	c, w := newTestCtx("POST", "/", map[string]interface{}{})
	h.Statistics(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC26: ListLogs - empty
func TestJobHandler_ListLogs_Empty(t *testing.T) {
	jobLogSvc := &mockJobLogService{
		ListFunc: func(ctx context.Context, params *schedulerServices.JobLogListParams) (*common.PageResult, error) {
			return &common.PageResult{List: []models.JobLog{}, Total: 0, Current: 1, PageSize: 10}, nil
		},
	}
	h := setupHandler(&mockJobService{}, jobLogSvc)
	c, w := newTestCtx("POST", "/", map[string]interface{}{})
	h.ListLogs(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC27: ListLogs - with filters
func TestJobHandler_ListLogs_WithFilters(t *testing.T) {
	jobLogSvc := &mockJobLogService{
		ListFunc: func(ctx context.Context, params *schedulerServices.JobLogListParams) (*common.PageResult, error) {
			return &common.PageResult{List: []models.JobLog{{}}, Total: 1, Current: 1, PageSize: 10}, nil
		},
	}
	h := setupHandler(&mockJobService{}, jobLogSvc)
	c, w := newTestCtx("POST", "/", map[string]interface{}{
		"current":   1.0,
		"pageSize":  10.0,
		"jobName":   "test",
		"jobGroup":  "DEFAULT",
		"status":    0.0,
		"startTime": "2024-01-01",
		"endTime":   "2024-01-31",
	})
	h.ListLogs(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC28: ListLogs - service error
func TestJobHandler_ListLogs_ServiceError(t *testing.T) {
	jobLogSvc := &mockJobLogService{
		ListFunc: func(ctx context.Context, params *schedulerServices.JobLogListParams) (*common.PageResult, error) {
			return nil, errors.New("logs fail")
		},
	}
	h := setupHandler(&mockJobService{}, jobLogSvc)
	c, w := newTestCtx("POST", "/", map[string]interface{}{})
	h.ListLogs(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC29: CleanLogs - invalid body
func TestJobHandler_CleanLogs_InvalidBody(t *testing.T) {
	h := setupHandler(&mockJobService{}, &mockJobLogService{})
	c, w := newTestCtx("POST", "/", map[string]interface{}{})
	h.CleanLogs(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TC30: CleanLogs - success
func TestJobHandler_CleanLogs_Success(t *testing.T) {
	h := setupHandler(&mockJobService{}, &mockJobLogService{})
	c, w := newTestCtx("POST", "/", map[string]interface{}{"days": 30})
	h.CleanLogs(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC31: CleanLogs - service error
func TestJobHandler_CleanLogs_ServiceError(t *testing.T) {
	jobLogSvc := &mockJobLogService{
		CleanOldLogsFunc: func(ctx context.Context, days int) error { return errors.New("clean fail") },
	}
	h := setupHandler(&mockJobService{}, jobLogSvc)
	c, w := newTestCtx("POST", "/", map[string]interface{}{"days": 30})
	h.CleanLogs(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC32: WithCore
func TestJobHandler_WithCore(t *testing.T) {
	h := &JobHandler{}
	result := h.WithCore(&core.Core{CoreInfra: &core.CoreInfra{}})
	assert.Same(t, h, result)
}

// TC33: WithCore nil-safe
func TestJobHandler_WithCore_NilCore(t *testing.T) {
	h := &JobHandler{}
	result := h.WithCore(nil)
	assert.Same(t, h, result)
}
