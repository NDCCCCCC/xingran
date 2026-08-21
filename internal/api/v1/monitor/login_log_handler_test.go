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

// mockLoginLogService implements monitorServices.LoginLogService via function fields
type mockLoginLogService struct {
	monitorServices.LoginLogService

	ListFunc        func(ctx context.Context, params monitorServices.LoginLogListParams) (*monitorServices.PageResult, error)
	GetByIDFunc     func(ctx context.Context, id string) (*models.LoginLog, error)
	DeleteFunc      func(ctx context.Context, id string) error
	BatchDeleteFunc func(ctx context.Context, ids []string) error
	CleanFunc       func(ctx context.Context) error
}

func (m *mockLoginLogService) List(ctx context.Context, params monitorServices.LoginLogListParams) (*monitorServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, params)
	}
	return nil, nil
}
func (m *mockLoginLogService) GetByID(ctx context.Context, id string) (*models.LoginLog, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockLoginLogService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}
func (m *mockLoginLogService) BatchDelete(ctx context.Context, ids []string) error {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, ids)
	}
	return nil
}
func (m *mockLoginLogService) Clean(ctx context.Context) error {
	if m.CleanFunc != nil {
		return m.CleanFunc(ctx)
	}
	return nil
}

func newTestCtxLL(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

func setupLoginLogHandler(mock *mockLoginLogService) *LoginLogHandler {
	return NewLoginLogHandler(mock).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	})
}

func TestLoginLog_List_Empty(t *testing.T) {
	mock := &mockLoginLogService{
		ListFunc: func(ctx context.Context, params monitorServices.LoginLogListParams) (*monitorServices.PageResult, error) {
			return &monitorServices.PageResult{List: []models.LoginLog{}, Total: 0, Current: 1, PageSize: 10}, nil
		},
	}
	h := setupLoginLogHandler(mock)
	c, w := newTestCtxLL("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginLog_List_Seeded(t *testing.T) {
	mock := &mockLoginLogService{
		ListFunc: func(ctx context.Context, params monitorServices.LoginLogListParams) (*monitorServices.PageResult, error) {
			return &monitorServices.PageResult{
				List:     []models.LoginLog{{Username: "alice"}, {Username: "bob"}},
				Total:    2, Current: 1, PageSize: 10,
			}, nil
		},
	}
	h := setupLoginLogHandler(mock)
	c, w := newTestCtxLL("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginLog_List_Error(t *testing.T) {
	mock := &mockLoginLogService{
		ListFunc: func(ctx context.Context, params monitorServices.LoginLogListParams) (*monitorServices.PageResult, error) {
			return nil, errors.New("list fail")
		},
	}
	h := setupLoginLogHandler(mock)
	c, w := newTestCtxLL("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestLoginLog_GetByID_Empty(t *testing.T) {
	mock := &mockLoginLogService{}
	h := setupLoginLogHandler(mock)
	c, w := newTestCtxLL("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.GetByID(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginLog_GetByID_Success(t *testing.T) {
	mock := &mockLoginLogService{
		GetByIDFunc: func(ctx context.Context, id string) (*models.LoginLog, error) {
			return &models.LoginLog{BaseTimeLine: models.BaseTimeLine{ID: id}, Username: "alice"}, nil
		},
	}
	h := setupLoginLogHandler(mock)
	c, w := newTestCtxLL("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.GetByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginLog_GetByID_NotFound(t *testing.T) {
	mock := &mockLoginLogService{
		GetByIDFunc: func(ctx context.Context, id string) (*models.LoginLog, error) {
			return nil, errors.New("not found")
		},
	}
	h := setupLoginLogHandler(mock)
	c, w := newTestCtxLL("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.GetByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestLoginLog_Delete_Empty(t *testing.T) {
	mock := &mockLoginLogService{}
	h := setupLoginLogHandler(mock)
	c, w := newTestCtxLL("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	h.Delete(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginLog_Delete_Success(t *testing.T) {
	mock := &mockLoginLogService{}
	h := setupLoginLogHandler(mock)
	c, w := newTestCtxLL("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Delete(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginLog_Delete_Error(t *testing.T) {
	mock := &mockLoginLogService{
		DeleteFunc: func(ctx context.Context, id string) error { return errors.New("del fail") },
	}
	h := setupLoginLogHandler(mock)
	c, w := newTestCtxLL("POST", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	h.Delete(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestLoginLog_BatchDelete_InvalidRequest(t *testing.T) {
	mock := &mockLoginLogService{}
	h := setupLoginLogHandler(mock)
	c, w := newTestCtxLL("POST", "/", map[string]interface{}{})
	h.BatchDelete(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestLoginLog_BatchDelete_Success(t *testing.T) {
	mock := &mockLoginLogService{}
	h := setupLoginLogHandler(mock)
	c, w := newTestCtxLL("POST", "/", map[string]interface{}{"ids": []string{uuid.NewString()}})
	h.BatchDelete(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginLog_BatchDelete_Error(t *testing.T) {
	mock := &mockLoginLogService{
		BatchDeleteFunc: func(ctx context.Context, ids []string) error { return errors.New("batch fail") },
	}
	h := setupLoginLogHandler(mock)
	c, w := newTestCtxLL("POST", "/", map[string]interface{}{"ids": []string{uuid.NewString()}})
	h.BatchDelete(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestLoginLog_Clean_Success(t *testing.T) {
	mock := &mockLoginLogService{}
	h := setupLoginLogHandler(mock)
	c, w := newTestCtxLL("POST", "/clean", nil)
	h.Clean(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginLog_Clean_Error(t *testing.T) {
	mock := &mockLoginLogService{
		CleanFunc: func(ctx context.Context) error { return errors.New("clean fail") },
	}
	h := setupLoginLogHandler(mock)
	c, w := newTestCtxLL("POST", "/clean", nil)
	h.Clean(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestLoginLog_UnlockUser_Empty(t *testing.T) {
	mock := &mockLoginLogService{}
	h := setupLoginLogHandler(mock)
	c, w := newTestCtxLL("POST", "/", nil)
	c.Params = gin.Params{{Key: "username", Value: ""}}
	h.UnlockUser(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoginLog_UnlockUser_Success(t *testing.T) {
	mock := &mockLoginLogService{}
	h := setupLoginLogHandler(mock)
	c, w := newTestCtxLL("POST", "/", nil)
	c.Params = gin.Params{{Key: "username", Value: "alice"}}
	h.UnlockUser(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginLog_WithCore(t *testing.T) {
	h := &LoginLogHandler{}
	result := h.WithCore(&core.Core{CoreInfra: &core.CoreInfra{}})
	assert.Same(t, h, result)
}

// setupLoginLogDB creates an in-memory DB with sys_logininfor table
func setupLoginLogDB(t *testing.T) *gorm.DB {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gormDB.Exec(`
		CREATE TABLE sys_logininfor (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			user_name TEXT,
			nickname TEXT,
			ipaddr TEXT,
			login_location TEXT,
			browser TEXT,
			os TEXT,
			status INTEGER,
			msg TEXT,
			login_time DATETIME
		)
	`).Error)
	return gormDB
}

func TestLoginLog_DB_List_Empty(t *testing.T) {
	gormDB := setupLoginLogDB(t)
	svc := monitorServices.NewLoginLogService(gormDB)
	h := NewLoginLogHandler(svc).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{DB: gormDB}},
		CoreServices: &core.CoreServices{},
	})
	c, w := newTestCtxLL("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginLog_DB_List_Seeded(t *testing.T) {
	gormDB := setupLoginLogDB(t)
	now := time.Now()
	for i := 0; i < 3; i++ {
		require.NoError(t, gormDB.Exec(
			"INSERT INTO sys_logininfor (id, user_name, ipaddr, status, login_time, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			uuid.NewString(), "alice", "127.0.0.1", 0, now, now, now,
		).Error)
	}
	svc := monitorServices.NewLoginLogService(gormDB)
	h := NewLoginLogHandler(svc).WithCore(&core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{DB: gormDB}},
		CoreServices: &core.CoreServices{},
	})
	c, w := newTestCtxLL("POST", "/list", map[string]interface{}{})
	h.List(c)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseRespLL(t, w.Body.Bytes())
	data := resp["data"].(map[string]interface{})
	assert.EqualValues(t, 3, data["total"])
}

func parseRespLL(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}
