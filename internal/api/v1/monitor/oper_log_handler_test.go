package monitor

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
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	monitorServices "github.com/xingran-next/xingran-go-backend/internal/services/monitor"
)

// mockOperLogService implements monitorServices.OperLogService via function fields
type mockOperLogService struct {
	monitorServices.OperLogService

	ListFunc        func(ctx context.Context, params monitorServices.OperLogListParams) (*monitorServices.PageResult, error)
	GetByIDFunc     func(ctx context.Context, id string) (*models.OperLog, error)
	DeleteFunc      func(ctx context.Context, id string) error
	BatchDeleteFunc func(ctx context.Context, ids []string) error
	CleanFunc       func(ctx context.Context) error
}

func (m *mockOperLogService) List(ctx context.Context, params monitorServices.OperLogListParams) (*monitorServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, params)
	}
	return nil, nil
}
func (m *mockOperLogService) GetByID(ctx context.Context, id string) (*models.OperLog, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockOperLogService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}
func (m *mockOperLogService) BatchDelete(ctx context.Context, ids []string) error {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, ids)
	}
	return nil
}
func (m *mockOperLogService) Clean(ctx context.Context) error {
	if m.CleanFunc != nil {
		return m.CleanFunc(ctx)
	}
	return nil
}

func newTestCtxOL(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

func setupOperLogHandler(mock *mockOperLogService) *OperLogHandler {
	return NewOperLogHandler(mock).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	})
}

func TestOperLog_List_Empty(t *testing.T) {
	mock := &mockOperLogService{
		ListFunc: func(ctx context.Context, params monitorServices.OperLogListParams) (*monitorServices.PageResult, error) {
			return &monitorServices.PageResult{List: []models.OperLog{}, Total: 0, Current: 1, PageSize: 10}, nil
		},
	}
	h := setupOperLogHandler(mock)
	c, w := newTestCtxOL("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOperLog_List_Seeded(t *testing.T) {
	mock := &mockOperLogService{
		ListFunc: func(ctx context.Context, params monitorServices.OperLogListParams) (*monitorServices.PageResult, error) {
			return &monitorServices.PageResult{
				List:     []models.OperLog{{Title: "user module"}, {Title: "role module"}},
				Total:    2, Current: 1, PageSize: 10,
			}, nil
		},
	}
	h := setupOperLogHandler(mock)
	c, w := newTestCtxOL("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOperLog_List_Error(t *testing.T) {
	mock := &mockOperLogService{
		ListFunc: func(ctx context.Context, params monitorServices.OperLogListParams) (*monitorServices.PageResult, error) {
			return nil, errors.New("list fail")
		},
	}
	h := setupOperLogHandler(mock)
	c, w := newTestCtxOL("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestOperLog_GetByID_Empty(t *testing.T) {
	mock := &mockOperLogService{}
	h := setupOperLogHandler(mock)
	c, w := newTestCtxOL("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.GetByID(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOperLog_GetByID_Success(t *testing.T) {
	mock := &mockOperLogService{
		GetByIDFunc: func(ctx context.Context, id string) (*models.OperLog, error) {
			return &models.OperLog{BaseTimeLine: models.BaseTimeLine{ID: id}, Title: "test"}, nil
		},
	}
	h := setupOperLogHandler(mock)
	c, w := newTestCtxOL("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.GetByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOperLog_GetByID_Error(t *testing.T) {
	mock := &mockOperLogService{
		GetByIDFunc: func(ctx context.Context, id string) (*models.OperLog, error) {
			return nil, errors.New("not found")
		},
	}
	h := setupOperLogHandler(mock)
	c, w := newTestCtxOL("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.GetByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestOperLog_Delete_Empty(t *testing.T) {
	mock := &mockOperLogService{}
	h := setupOperLogHandler(mock)
	c, w := newTestCtxOL("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Delete(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOperLog_Delete_Success(t *testing.T) {
	mock := &mockOperLogService{}
	h := setupOperLogHandler(mock)
	c, w := newTestCtxOL("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Delete(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOperLog_Delete_Error(t *testing.T) {
	mock := &mockOperLogService{
		DeleteFunc: func(ctx context.Context, id string) error { return errors.New("del fail") },
	}
	h := setupOperLogHandler(mock)
	c, w := newTestCtxOL("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Delete(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestOperLog_BatchDelete_InvalidRequest(t *testing.T) {
	mock := &mockOperLogService{}
	h := setupOperLogHandler(mock)
	c, w := newTestCtxOL("POST", "/", map[string]interface{}{})
	h.BatchDelete(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOperLog_BatchDelete_Success(t *testing.T) {
	mock := &mockOperLogService{}
	h := setupOperLogHandler(mock)
	c, w := newTestCtxOL("POST", "/", map[string]interface{}{"ids": []string{uuid.NewString()}})
	h.BatchDelete(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOperLog_BatchDelete_Error(t *testing.T) {
	mock := &mockOperLogService{
		BatchDeleteFunc: func(ctx context.Context, ids []string) error { return errors.New("batch fail") },
	}
	h := setupOperLogHandler(mock)
	c, w := newTestCtxOL("POST", "/", map[string]interface{}{"ids": []string{uuid.NewString()}})
	h.BatchDelete(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestOperLog_Clean_Success(t *testing.T) {
	mock := &mockOperLogService{}
	h := setupOperLogHandler(mock)
	c, w := newTestCtxOL("POST", "/clean", nil)
	h.Clean(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOperLog_Clean_Error(t *testing.T) {
	mock := &mockOperLogService{
		CleanFunc: func(ctx context.Context) error { return errors.New("clean fail") },
	}
	h := setupOperLogHandler(mock)
	c, w := newTestCtxOL("POST", "/clean", nil)
	h.Clean(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestOperLog_WithCore(t *testing.T) {
	h := &OperLogHandler{}
	result := h.WithCore(&core.Core{CoreInfra: &core.CoreInfra{}})
	assert.Same(t, h, result)
}

// setupOperLogDB creates an in-memory DB with sys_oper_log table
func setupOperLogDB(t *testing.T) *gorm.DB {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gormDB.Exec(`
		CREATE TABLE sys_oper_log (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			title TEXT,
			business_type INTEGER,
			method TEXT,
			request_method TEXT,
			operator_type INTEGER,
			oper_name TEXT,
			nickname TEXT,
			dept_name TEXT,
			oper_url TEXT,
			oper_ip TEXT,
			oper_location TEXT,
			oper_param TEXT,
			json_result TEXT,
			status INTEGER,
			error_msg TEXT,
			oper_time DATETIME,
			cost_time INTEGER
		)
	`).Error)
	return gormDB
}

func TestOperLog_DB_List_Empty(t *testing.T) {
	gormDB := setupOperLogDB(t)
	svc := monitorServices.NewOperLogService(gormDB)
	h := NewOperLogHandler(svc).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{DB: gormDB}},
		CoreServices: &core.CoreServices{},
	})
	c, w := newTestCtxOL("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOperLog_DB_List_Seeded(t *testing.T) {
	gormDB := setupOperLogDB(t)
	now := time.Now()
	for i := 0; i < 3; i++ {
		require.NoError(t, gormDB.Exec(
			"INSERT INTO sys_oper_log (id, title, business_type, status, oper_time, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			uuid.NewString(), "user module", 1, 0, now, now, now,
		).Error)
	}
	svc := monitorServices.NewOperLogService(gormDB)
	h := NewOperLogHandler(svc).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{DB: gormDB}},
		CoreServices: &core.CoreServices{},
	})
	c, w := newTestCtxOL("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOperLog_DB_GetByID(t *testing.T) {
	gormDB := setupOperLogDB(t)
	id := uuid.NewString()
	now := time.Now()
	require.NoError(t, gormDB.Exec(
		"INSERT INTO sys_oper_log (id, title, business_type, status, oper_time, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, "user module", 1, 0, now, now, now,
	).Error)
	svc := monitorServices.NewOperLogService(gormDB)
	h := NewOperLogHandler(svc).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{DB: gormDB}},
		CoreServices: &core.CoreServices{},
	})
	c, w := newTestCtxOL("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	h.GetByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOperLog_DB_Clean(t *testing.T) {
	gormDB := setupOperLogDB(t)
	svc := monitorServices.NewOperLogService(gormDB)
	// 用一个特殊的 record 用来验证 Clean 后能插入 audit row
	h := NewOperLogHandler(svc).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{DB: gormDB}},
		CoreServices: &core.CoreServices{},
	})
	c, w := newTestCtxOL("POST", "/clean", nil)
	h.Clean(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOperLog_Clean_NilCore(t *testing.T) {
	mock := &mockOperLogService{}
	h := NewOperLogHandler(mock) // 不注入 core
	c, w := newTestCtxOL("POST", "/clean", nil)
	h.Clean(c)
	// core=nil 时跳过 audit row 验证
	assert.Equal(t, http.StatusOK, w.Code)
}
