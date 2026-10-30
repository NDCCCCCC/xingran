package rpa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
)

// Compile-time assertion: mockTaskService implements rpa.TaskService.
var _ rpa.TaskService = (*mockTaskService)(nil)

// mockTaskService implements rpa.TaskService via function fields.
type mockTaskService struct {
	rpa.TaskService

	CreateFunc  func(ctx context.Context, req *rpa.CreateTaskRequest, userID string) (*rpamodels.Task, error)
	UpdateFunc  func(ctx context.Context, req *rpa.UpdateTaskRequest, userID string) error
	DeleteFunc  func(ctx context.Context, id string) error
	GetByIDFunc func(ctx context.Context, id string) (*rpamodels.Task, error)
	ListFunc    func(ctx context.Context, params *rpa.TaskListParams) (*rpa.PageResult, error)
	ExecuteFunc func(ctx context.Context, req *rpa.ExecuteTaskRequest, userID string) (*rpamodels.Execution, error)
}

func (m *mockTaskService) Create(ctx context.Context, req *rpa.CreateTaskRequest, userID string) (*rpamodels.Task, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req, userID)
	}
	return &rpamodels.Task{}, nil
}

func (m *mockTaskService) Update(ctx context.Context, req *rpa.UpdateTaskRequest, userID string) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, req, userID)
	}
	return nil
}

func (m *mockTaskService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *mockTaskService) GetByID(ctx context.Context, id string) (*rpamodels.Task, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockTaskService) List(ctx context.Context, params *rpa.TaskListParams) (*rpa.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, params)
	}
	return &rpa.PageResult{}, nil
}

func (m *mockTaskService) Execute(ctx context.Context, req *rpa.ExecuteTaskRequest, userID string) (*rpamodels.Execution, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, req, userID)
	}
	return &rpamodels.Execution{}, nil
}

// ==================== Test helpers ====================

// newTestCtxTask creates a gin.Context with optional JSON body.
func newTestCtxTask(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

// setupTaskHandler wires a TaskHandler with mock service and minimal core.
func setupTaskHandler(mock *mockTaskService) *TaskHandler {
	return NewTaskHandler(mock, nil).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	})
}

// ==================== Compile-only smoke ====================

func TestTaskHandler_CompileOnly(t *testing.T) {
	mock := &mockTaskService{}
	h := setupTaskHandler(mock)
	assert.NotNil(t, h)
}

// ==================== List ====================

func TestTaskHandler_List_Success(t *testing.T) {
	mock := &mockTaskService{
		ListFunc: func(ctx context.Context, params *rpa.TaskListParams) (*rpa.PageResult, error) {
			assert.Equal(t, 1, params.Current)
			assert.Equal(t, 10, params.PageSize)
			return &rpa.PageResult{List: []rpamodels.Task{{TaskName: "task-1"}}, Total: 1, Current: 1, PageSize: 10}, nil
		},
	}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTaskHandler_List_BindError(t *testing.T) {
	mock := &mockTaskService{}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/list", "not-json")
	h.List(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTaskHandler_List_ServiceError(t *testing.T) {
	mock := &mockTaskService{
		ListFunc: func(ctx context.Context, params *rpa.TaskListParams) (*rpa.PageResult, error) {
			return nil, errors.New("list fail")
		},
	}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Create ====================

func TestTaskHandler_Create_Success(t *testing.T) {
	mock := &mockTaskService{
		CreateFunc: func(ctx context.Context, req *rpa.CreateTaskRequest, userID string) (*rpamodels.Task, error) {
			assert.Equal(t, "user-1", userID)
			return &rpamodels.Task{TaskName: req.Name}, nil
		},
	}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/", rpa.CreateTaskRequest{
		Name:   "new-task",
		Script: []interface{}{},
	})
	c.Set("userId", "user-1")
	h.Create(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTaskHandler_Create_BindError(t *testing.T) {
	// Script is required — missing → 400
	mock := &mockTaskService{}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/", map[string]interface{}{
		"name": "no-script",
	})
	h.Create(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTaskHandler_Create_ServiceError(t *testing.T) {
	mock := &mockTaskService{
		CreateFunc: func(ctx context.Context, req *rpa.CreateTaskRequest, userID string) (*rpamodels.Task, error) {
			return nil, errors.New("dup")
		},
	}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/", rpa.CreateTaskRequest{
		Name:   "x",
		Script: []interface{}{},
	})
	h.Create(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== GetByID ====================

func TestTaskHandler_GetByID_Success(t *testing.T) {
	mock := &mockTaskService{
		GetByIDFunc: func(ctx context.Context, id string) (*rpamodels.Task, error) {
			return &rpamodels.Task{TaskName: "task-1"}, nil
		},
	}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/t-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "t-1"}}
	h.GetByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTaskHandler_GetByID_EmptyID(t *testing.T) {
	mock := &mockTaskService{}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.GetByID(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTaskHandler_GetByID_NotFound(t *testing.T) {
	mock := &mockTaskService{
		GetByIDFunc: func(ctx context.Context, id string) (*rpamodels.Task, error) {
			return nil, errors.New("not found")
		},
	}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/t-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "t-1"}}
	h.GetByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Update ====================

func TestTaskHandler_Update_Success(t *testing.T) {
	mock := &mockTaskService{
		UpdateFunc: func(ctx context.Context, req *rpa.UpdateTaskRequest, userID string) error {
			assert.Equal(t, "t-1", req.ID)
			assert.Equal(t, "user-1", userID)
			return nil
		},
	}
	h := setupTaskHandler(mock)
	// UpdateTaskRequest.ID is `binding:"required"` so body must include id
	c, w := newTestCtxTask("POST", "/t-1/update", map[string]interface{}{
		"id":   "t-1",
		"name": "updated",
	})
	c.Params = gin.Params{{Key: "id", Value: "t-1"}}
	c.Set("userId", "user-1")
	h.Update(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTaskHandler_Update_EmptyID(t *testing.T) {
	mock := &mockTaskService{}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/", map[string]interface{}{})
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Update(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTaskHandler_Update_ServiceError(t *testing.T) {
	mock := &mockTaskService{
		UpdateFunc: func(ctx context.Context, req *rpa.UpdateTaskRequest, userID string) error {
			return errors.New("update fail")
		},
	}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/t-1/update", map[string]interface{}{})
	c.Params = gin.Params{{Key: "id", Value: "t-1"}}
	h.Update(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Delete ====================

func TestTaskHandler_Delete_Success(t *testing.T) {
	mock := &mockTaskService{
		DeleteFunc: func(ctx context.Context, id string) error {
			assert.Equal(t, "t-1", id)
			return nil
		},
	}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/t-1/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: "t-1"}}
	h.Delete(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTaskHandler_Delete_EmptyID(t *testing.T) {
	mock := &mockTaskService{}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Delete(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTaskHandler_Delete_ServiceError(t *testing.T) {
	mock := &mockTaskService{
		DeleteFunc: func(ctx context.Context, id string) error {
			return errors.New("del fail")
		},
	}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/t-1/delete", nil)
	c.Params = gin.Params{{Key: "id", Value: "t-1"}}
	h.Delete(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== Execute ====================

func TestTaskHandler_Execute_WithBody(t *testing.T) {
	mock := &mockTaskService{
		ExecuteFunc: func(ctx context.Context, req *rpa.ExecuteTaskRequest, userID string) (*rpamodels.Execution, error) {
			assert.Equal(t, "t-1", req.TaskID)
			return &rpamodels.Execution{ID: "exec-1"}, nil
		},
	}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/t-1/execute", rpa.ExecuteTaskRequest{
		TaskID: "t-1",
	})
	c.Params = gin.Params{{Key: "id", Value: "t-1"}}
	h.Execute(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTaskHandler_Execute_NoBody(t *testing.T) {
	// When JSON bind fails, handler constructs request from path param
	mock := &mockTaskService{
		ExecuteFunc: func(ctx context.Context, req *rpa.ExecuteTaskRequest, userID string) (*rpamodels.Execution, error) {
			assert.Equal(t, "t-1", req.TaskID)
			return &rpamodels.Execution{ID: "exec-1"}, nil
		},
	}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/t-1/execute", "not-json")
	c.Params = gin.Params{{Key: "id", Value: "t-1"}}
	h.Execute(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTaskHandler_Execute_EmptyID(t *testing.T) {
	mock := &mockTaskService{}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "//execute", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Execute(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTaskHandler_Execute_ServiceError(t *testing.T) {
	mock := &mockTaskService{
		ExecuteFunc: func(ctx context.Context, req *rpa.ExecuteTaskRequest, userID string) (*rpamodels.Execution, error) {
			return nil, errors.New("exec fail")
		},
	}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "/t-1/execute", rpa.ExecuteTaskRequest{TaskID: "t-1"})
	c.Params = gin.Params{{Key: "id", Value: "t-1"}}
	h.Execute(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ==================== UploadExcel ====================

// newTestCtxTaskMultipart creates a multipart upload context.
func newTestCtxTaskMultipart(path, fieldName, filename string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := &bytes.Buffer{}
	// boundary simulation not strictly needed for handler.FormFile
	// but Content-Type header must be multipart/form-data with boundary.
	body.WriteString("--X-BOUNDARY\r\nContent-Disposition: form-data; name=\"file\"; filename=\"" + filename + "\"\r\nContent-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet\r\n\r\nfakexlsx\r\n--X-BOUNDARY--\r\n")
	req := httptest.NewRequest("POST", path, body)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=X-BOUNDARY")
	c.Request = req
	return c, w
}

func TestTaskHandler_UploadExcel_NoFile(t *testing.T) {
	mock := &mockTaskService{}
	h := setupTaskHandler(mock)
	// Empty body — no "file" field
	c, w := newTestCtxTask("POST", "/upload-excel", nil)
	h.UploadExcel(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Note: UploadExcel and ExecuteWithExcel require excelService and real multipart parsing.
// We avoid them here since excelService=nil would panic; the handler nil-check fires on
// h.excelService.ParseExcelFile. The bind-error branch is sufficient for coverage.

// ==================== ExecuteWithExcel (partial — bind error path only) ====================

func TestTaskHandler_ExecuteWithExcel_EmptyID(t *testing.T) {
	mock := &mockTaskService{}
	h := setupTaskHandler(mock)
	c, w := newTestCtxTask("POST", "//execute-with-excel", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.ExecuteWithExcel(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
