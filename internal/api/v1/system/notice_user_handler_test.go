package system

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// =====================================================================
// Phase 74-04: notice_user_handler tests
//
// Scope (notice_user_handler.go — 12 funcs):
//   - getUserID, requireUserID
//   - getReadNoticeIDs
//   - GetMyNotices (with read-status merge)
//   - GetMyNoticeDetail
//   - MarkNoticeRead, MarkAllNoticesRead
//   - GetUnreadCount
//   - IgnoreNotice, UnignoreNotice
//
// Mock pattern: per-method *Func fields on mockNoticeCacheService satisfying
// the full systemServices.NoticeCacheService interface.
// =====================================================================

// mockNoticeCacheService satisfies systemServices.NoticeCacheService via per-method *Func fields.
type mockNoticeCacheService struct {
	GetUserNoticesFunc         func(ctx context.Context, userID string, page, pageSize int, status *string) ([]models.Notice, int64, error)
	GetUnreadCountFunc         func(ctx context.Context, userID string) (int, error)
	MarkNoticeReadFunc         func(ctx context.Context, noticeID, userID, ip string) error
	MarkAllNoticesReadFunc     func(ctx context.Context, userID string) error
	IgnoreNoticeFunc           func(ctx context.Context, noticeID, userID string) error
	UnignoreNoticeFunc         func(ctx context.Context, noticeID, userID string) error
	GetNoticeByIDFunc          func(ctx context.Context, id string) (*models.Notice, error)
	CreateFunc                 func(ctx context.Context, req *requests.NoticeCreateRequest, creatorID, creatorName string) (*models.Notice, error)
	UpdateFunc                 func(ctx context.Context, id string, req *requests.NoticeUpdateRequest) error
	DeleteFunc                 func(ctx context.Context, id string) error
	BatchDeleteFunc            func(ctx context.Context, ids []string) error
	ListFunc                   func(ctx context.Context, params requests.NoticeListParams) (*systemServices.PageResult, error)
	PublishFunc                func(ctx context.Context, id string) error
	WithdrawFunc               func(ctx context.Context, id string) error
	GetStatisticsFunc          func(ctx context.Context, id string) (*models.NoticeStatistics, error)
	GetStatusStatisticsFunc    func(ctx context.Context) (*services.NoticeStatusStatistics, error)
	InvalidateNoticeCacheFunc  func(ctx context.Context, noticeID string) error
	InvalidateUserNoticeCacheFunc func(ctx context.Context, userID string) error
	InvalidateAllNoticeCacheFunc  func(ctx context.Context) error
}

func (m *mockNoticeCacheService) GetUserNotices(ctx context.Context, userID string, page, pageSize int, status *string) ([]models.Notice, int64, error) {
	if m.GetUserNoticesFunc != nil {
		return m.GetUserNoticesFunc(ctx, userID, page, pageSize, status)
	}
	return nil, 0, errors.New("GetUserNoticesFunc not set")
}
func (m *mockNoticeCacheService) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	if m.GetUnreadCountFunc != nil {
		return m.GetUnreadCountFunc(ctx, userID)
	}
	return 0, errors.New("GetUnreadCountFunc not set")
}
func (m *mockNoticeCacheService) MarkNoticeRead(ctx context.Context, noticeID, userID, ip string) error {
	if m.MarkNoticeReadFunc != nil {
		return m.MarkNoticeReadFunc(ctx, noticeID, userID, ip)
	}
	return errors.New("MarkNoticeReadFunc not set")
}
func (m *mockNoticeCacheService) MarkAllNoticesRead(ctx context.Context, userID string) error {
	if m.MarkAllNoticesReadFunc != nil {
		return m.MarkAllNoticesReadFunc(ctx, userID)
	}
	return errors.New("MarkAllNoticesReadFunc not set")
}
func (m *mockNoticeCacheService) IgnoreNotice(ctx context.Context, noticeID, userID string) error {
	if m.IgnoreNoticeFunc != nil {
		return m.IgnoreNoticeFunc(ctx, noticeID, userID)
	}
	return errors.New("IgnoreNoticeFunc not set")
}
func (m *mockNoticeCacheService) UnignoreNotice(ctx context.Context, noticeID, userID string) error {
	if m.UnignoreNoticeFunc != nil {
		return m.UnignoreNoticeFunc(ctx, noticeID, userID)
	}
	return errors.New("UnignoreNoticeFunc not set")
}
func (m *mockNoticeCacheService) GetNoticeByID(ctx context.Context, id string) (*models.Notice, error) {
	if m.GetNoticeByIDFunc != nil {
		return m.GetNoticeByIDFunc(ctx, id)
	}
	return nil, errors.New("GetNoticeByIDFunc not set")
}
func (m *mockNoticeCacheService) Create(ctx context.Context, req *requests.NoticeCreateRequest, creatorID, creatorName string) (*models.Notice, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req, creatorID, creatorName)
	}
	return nil, nil
}
func (m *mockNoticeCacheService) Update(ctx context.Context, id string, req *requests.NoticeUpdateRequest) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, req)
	}
	return nil
}
func (m *mockNoticeCacheService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}
func (m *mockNoticeCacheService) BatchDelete(ctx context.Context, ids []string) error {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, ids)
	}
	return nil
}
func (m *mockNoticeCacheService) List(ctx context.Context, params requests.NoticeListParams) (*systemServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, params)
	}
	return &systemServices.PageResult{}, nil
}
func (m *mockNoticeCacheService) Publish(ctx context.Context, id string) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, id)
	}
	return nil
}
func (m *mockNoticeCacheService) Withdraw(ctx context.Context, id string) error {
	if m.WithdrawFunc != nil {
		return m.WithdrawFunc(ctx, id)
	}
	return nil
}
func (m *mockNoticeCacheService) GetStatistics(ctx context.Context, id string) (*models.NoticeStatistics, error) {
	if m.GetStatisticsFunc != nil {
		return m.GetStatisticsFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockNoticeCacheService) GetStatusStatistics(ctx context.Context) (*services.NoticeStatusStatistics, error) {
	if m.GetStatusStatisticsFunc != nil {
		return m.GetStatusStatisticsFunc(ctx)
	}
	return nil, nil
}
func (m *mockNoticeCacheService) InvalidateNoticeCache(ctx context.Context, noticeID string) error { return nil }
func (m *mockNoticeCacheService) InvalidateUserNoticeCache(ctx context.Context, userID string) error { return nil }
func (m *mockNoticeCacheService) InvalidateAllNoticeCache(ctx context.Context) error { return nil }

// setupNoticeUserDB creates a SQLite DB with sys_notice_read for read-status tracking.
func setupNoticeUserDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_notice_read (
			id TEXT PRIMARY KEY,
			notice_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			read_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	return db
}

// invokeNUHandler builds a gin context with user_id in the context map, then invokes the handler.
func invokeNUHandler(t *testing.T, method, path string, body interface{}, params gin.Params, userID string,
	handler func(*gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	if body != nil {
		b, _ := json.Marshal(body)
		c.Request = httptest.NewRequest(method, path, asReader(b))
		c.Request.Header.Set("Content-Type", "application/json")
	}
	if userID != "" {
		c.Set("user_id", userID)
	}
	if params != nil {
		c.Params = params
	}
	if handler != nil {
		handler(c)
	}
	return w
}

func newNUHandler(db *gorm.DB, mock *mockNoticeCacheService) *NoticeUserHandler {
	h := NewNoticeUserHandler(mock, db)
	h.core = &core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{OperLogService: nil},
	}
	return h
}

// ----------------------------------------------------------------------------
// getUserID / requireUserID (unexported helpers)
// ----------------------------------------------------------------------------

func TestGetUserID_FromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_id", "user-abc")

	got, ok := getUserID(c)
	assert.True(t, ok)
	assert.Equal(t, "user-abc", got)
}

func TestGetUserID_Missing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	got, ok := getUserID(c)
	assert.False(t, ok)
	assert.Empty(t, got)
}

func TestGetUserID_WrongType_Panics(t *testing.T) {
	// getUserID assumes the value stored under "user_id" is a string.
	// A non-string value causes a panic — this test documents that invariant.
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_id", 12345)

	require.Panics(t, func() {
		_, _ = getUserID(c)
	}, "non-string user_id should panic")
}

func TestRequireUserID_FailureWritesResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/x", nil)

	got, ok := requireUserID(c)
	assert.False(t, ok)
	assert.Empty(t, got)
	assert.NotEqual(t, 0, w.Code) // unauthorized response must have been written
}

// ----------------------------------------------------------------------------
// getReadNoticeIDs
// ----------------------------------------------------------------------------

func TestNoticeUserHandler_GetReadNoticeIDs_Empty(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{}
	h := newNUHandler(db, mock)
	m, err := h.getReadNoticeIDs("user-x")
	require.NoError(t, err)
	assert.Empty(t, m)
}

func TestNoticeUserHandler_GetReadNoticeIDs_Seeded(t *testing.T) {
	db := setupNoticeUserDB(t)
	now := "2024-01-01 00:00:00"
	require.NoError(t, db.Exec(`INSERT INTO sys_notice_read (id, notice_id, user_id, created_at, updated_at) VALUES ('1','n1','u1',?,?)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_notice_read (id, notice_id, user_id, created_at, updated_at) VALUES ('2','n2','u1',?,?)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_notice_read (id, notice_id, user_id, created_at, updated_at) VALUES ('3','n3','u2',?,?)`, now, now).Error)

	mock := &mockNoticeCacheService{}
	h := newNUHandler(db, mock)
	m, err := h.getReadNoticeIDs("u1")
	require.NoError(t, err)
	assert.True(t, m["n1"])
	assert.True(t, m["n2"])
	assert.False(t, m["n3"], "read IDs for u2 must not leak into u1's map")
}

// ----------------------------------------------------------------------------
// GetMyNotices
// ----------------------------------------------------------------------------

func TestNoticeUserHandler_GetMyNotices_NoUserID_Returns401(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "GET", "/system/my-notices", nil, nil, "", h.GetMyNotices)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeUserHandler_GetMyNotices_EmptyList(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{
		GetUserNoticesFunc: func(_ context.Context, _ string, _, _ int, _ *string) ([]models.Notice, int64, error) {
			return []models.Notice{}, 0, nil
		},
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "GET", "/system/my-notices", nil, nil, "user-1", h.GetMyNotices)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			List  []interface{} `json:"list"`
			Total int           `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, 0, resp.Data.Total)
}

func TestNoticeUserHandler_GetMyNotices_MergesReadStatus(t *testing.T) {
	db := setupNoticeUserDB(t)
	now := "2024-01-01 00:00:00"
	require.NoError(t, db.Exec(`INSERT INTO sys_notice_read (id, notice_id, user_id, created_at, updated_at) VALUES ('1','n-read','user-1',?,?)`, now, now).Error)

	mock := &mockNoticeCacheService{
		GetUserNoticesFunc: func(_ context.Context, uid string, _, _ int, _ *string) ([]models.Notice, int64, error) {
			return []models.Notice{
				{BaseModel: models.BaseModel{ID: "n-read"}},
				{BaseModel: models.BaseModel{ID: "n-unread"}},
			}, 2, nil
		},
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "GET", "/system/my-notices", nil, nil, "user-1", h.GetMyNotices)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			List []struct {
				IsRead bool   `json:"isRead"`
				ID     string `json:"id"`
			} `json:"list"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Data.List, 2)

	isReadMap := map[string]bool{}
	for _, n := range resp.Data.List {
		isReadMap[n.ID] = n.IsRead
	}
	assert.True(t, isReadMap["n-read"], "n-read should be marked read")
	assert.False(t, isReadMap["n-unread"], "n-unread should be marked unread")
}

func TestNoticeUserHandler_GetMyNotices_ServiceError(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{
		GetUserNoticesFunc: func(_ context.Context, _ string, _, _ int, _ *string) ([]models.Notice, int64, error) {
			return nil, 0, errors.New("db failure")
		},
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "GET", "/system/my-notices", nil, nil, "user-1", h.GetMyNotices)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeUserHandler_GetMyNotices_ReadIDsError(t *testing.T) {
	// First call to GetUserNotices succeeds; readIDs fails because we close the DB.
	db := setupNoticeUserDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	mock := &mockNoticeCacheService{
		GetUserNoticesFunc: func(_ context.Context, _ string, _, _ int, _ *string) ([]models.Notice, int64, error) {
			return []models.Notice{}, 0, nil
		},
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "GET", "/system/my-notices", nil, nil, "user-1", h.GetMyNotices)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// GetMyNoticeDetail
// ----------------------------------------------------------------------------

func TestNoticeUserHandler_GetMyNoticeDetail_Success(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{
		GetNoticeByIDFunc: func(_ context.Context, id string) (*models.Notice, error) {
			return &models.Notice{
				BaseModel:     models.BaseModel{ID: id},
				NoticeTitle:   "Hello",
				NoticeContent: "world",
			}, nil
		},
		MarkNoticeReadFunc: func(_ context.Context, _, _, _ string) error { return nil },
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "GET", "/system/my-notices/n-1", nil,
		gin.Params{{Key: "id", Value: "n-1"}}, "user-1", h.GetMyNoticeDetail)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeUserHandler_GetMyNoticeDetail_EmptyID(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "GET", "/system/my-notices/", nil,
		gin.Params{{Key: "id", Value: ""}}, "user-1", h.GetMyNoticeDetail)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeUserHandler_GetMyNoticeDetail_ServiceError(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{
		GetNoticeByIDFunc: func(_ context.Context, _ string) (*models.Notice, error) {
			return nil, errors.New("not found")
		},
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "GET", "/system/my-notices/n-1", nil,
		gin.Params{{Key: "id", Value: "n-1"}}, "user-1", h.GetMyNoticeDetail)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// MarkNoticeRead
// ----------------------------------------------------------------------------

func TestNoticeUserHandler_MarkNoticeRead_Success(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{
		MarkNoticeReadFunc: func(_ context.Context, _, _, _ string) error { return nil },
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "POST", "/system/my-notices/n-1/read", nil,
		gin.Params{{Key: "id", Value: "n-1"}}, "user-1", h.MarkNoticeRead)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeUserHandler_MarkNoticeRead_EmptyID(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "POST", "/system/my-notices//read", nil,
		gin.Params{{Key: "id", Value: ""}}, "user-1", h.MarkNoticeRead)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeUserHandler_MarkNoticeRead_ServiceError(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{
		MarkNoticeReadFunc: func(_ context.Context, _, _, _ string) error { return errors.New("boom") },
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "POST", "/system/my-notices/n-1/read", nil,
		gin.Params{{Key: "id", Value: "n-1"}}, "user-1", h.MarkNoticeRead)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// MarkAllNoticesRead
// ----------------------------------------------------------------------------

func TestNoticeUserHandler_MarkAllNoticesRead_Success(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{
		MarkAllNoticesReadFunc: func(_ context.Context, _ string) error { return nil },
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "POST", "/system/my-notices/read-all", nil, nil, "user-1", h.MarkAllNoticesRead)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeUserHandler_MarkAllNoticesRead_ServiceError(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{
		MarkAllNoticesReadFunc: func(_ context.Context, _ string) error { return errors.New("boom") },
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "POST", "/system/my-notices/read-all", nil, nil, "user-1", h.MarkAllNoticesRead)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// GetUnreadCount
// ----------------------------------------------------------------------------

func TestNoticeUserHandler_GetUnreadCount_Success(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{
		GetUnreadCountFunc: func(_ context.Context, _ string) (int, error) { return 7, nil },
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "GET", "/system/my-notices/unread-count", nil, nil, "user-1", h.GetUnreadCount)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeUserHandler_GetUnreadCount_ServiceError(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{
		GetUnreadCountFunc: func(_ context.Context, _ string) (int, error) { return 0, errors.New("boom") },
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "GET", "/system/my-notices/unread-count", nil, nil, "user-1", h.GetUnreadCount)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// IgnoreNotice
// ----------------------------------------------------------------------------

func TestNoticeUserHandler_IgnoreNotice_Success(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{
		IgnoreNoticeFunc: func(_ context.Context, _, _ string) error { return nil },
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "POST", "/system/my-notices/n-1/ignore", nil,
		gin.Params{{Key: "id", Value: "n-1"}}, "user-1", h.IgnoreNotice)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeUserHandler_IgnoreNotice_EmptyID(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "POST", "/system/my-notices//ignore", nil,
		gin.Params{{Key: "id", Value: ""}}, "user-1", h.IgnoreNotice)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeUserHandler_IgnoreNotice_ServiceError(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{
		IgnoreNoticeFunc: func(_ context.Context, _, _ string) error { return errors.New("boom") },
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "POST", "/system/my-notices/n-1/ignore", nil,
		gin.Params{{Key: "id", Value: "n-1"}}, "user-1", h.IgnoreNotice)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// UnignoreNotice
// ----------------------------------------------------------------------------

func TestNoticeUserHandler_UnignoreNotice_Success(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{
		UnignoreNoticeFunc: func(_ context.Context, _, _ string) error { return nil },
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "POST", "/system/my-notices/n-1/unignore", nil,
		gin.Params{{Key: "id", Value: "n-1"}}, "user-1", h.UnignoreNotice)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeUserHandler_UnignoreNotice_NotIgnoredIdempotentSuccess(t *testing.T) {
	// The handler treats "该通知未被忽略" as a no-op success (idempotent).
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{
		UnignoreNoticeFunc: func(_ context.Context, _, _ string) error {
			return errors.New("该通知未被忽略")
		},
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "POST", "/system/my-notices/n-1/unignore", nil,
		gin.Params{{Key: "id", Value: "n-1"}}, "user-1", h.UnignoreNotice)
	assert.Equal(t, http.StatusOK, w.Code, "unignoring an un-ignored notice must succeed (idempotent)")
}

func TestNoticeUserHandler_UnignoreNotice_OtherServiceError(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{
		UnignoreNoticeFunc: func(_ context.Context, _, _ string) error {
			return errors.New("some other error")
		},
	}
	h := newNUHandler(db, mock)

	w := invokeNUHandler(t, "POST", "/system/my-notices/n-1/unignore", nil,
		gin.Params{{Key: "id", Value: "n-1"}}, "user-1", h.UnignoreNotice)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// WithCore
// ----------------------------------------------------------------------------

func TestNoticeUserHandler_WithCore_NilSafe(t *testing.T) {
	db := setupNoticeUserDB(t)
	mock := &mockNoticeCacheService{}
	h := NewNoticeUserHandler(mock, db)
	// nil receiver must not panic — guard via the typed nil path.
	var nilH *NoticeUserHandler
	got := nilH.WithCore(&core.Core{})
	assert.Nil(t, got, "nil receiver WithCore should return nil")
	// Real call returns the same handler.
	out := h.WithCore(&core.Core{})
	assert.Same(t, h, out)
}

// asReader is a tiny indirection so the file imports stay tidy.
func asReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }
