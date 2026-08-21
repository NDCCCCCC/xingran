package system

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// =====================================================================
// Phase 74-04: notice_handler (admin CRUD) tests.
//
// Reuses mockNoticeCacheService declared in notice_user_handler_test.go
// (same package). This file adds the channel + scheduler mocks the admin
// handler needs. All success-path tests set *Func fields explicitly —
// the shared mock's zero-value defaults are not safe for Create (returns
// nil notice → nil deref on notice.ID).
// =====================================================================

type mockNotificationChannelService struct {
	GetNotificationChannelsFunc func(ctx context.Context, noticeID string) ([]models.NotificationChannel, error)
	SetNotificationChannelsFunc func(ctx context.Context, noticeID string, channels []models.NotificationChannel) error
	PublishAndSendNoticeFunc    func(ctx context.Context, noticeID string) error
}

func (m *mockNotificationChannelService) GetNotificationChannels(ctx context.Context, noticeID string) ([]models.NotificationChannel, error) {
	if m.GetNotificationChannelsFunc != nil {
		return m.GetNotificationChannelsFunc(ctx, noticeID)
	}
	return []models.NotificationChannel{}, nil
}
func (m *mockNotificationChannelService) SetNotificationChannels(ctx context.Context, noticeID string, channels []models.NotificationChannel) error {
	if m.SetNotificationChannelsFunc != nil {
		return m.SetNotificationChannelsFunc(ctx, noticeID, channels)
	}
	return nil
}
func (m *mockNotificationChannelService) PublishAndSendNotice(ctx context.Context, noticeID string) error {
	if m.PublishAndSendNoticeFunc != nil {
		return m.PublishAndSendNoticeFunc(ctx, noticeID)
	}
	return nil
}

type mockNoticeScheduler struct {
	AddJobFunc    func(job *models.Job) error
	RemoveJobFunc func(jobID string) error
}

func (m *mockNoticeScheduler) AddJob(job *models.Job) error {
	if m.AddJobFunc != nil {
		return m.AddJobFunc(job)
	}
	return nil
}
func (m *mockNoticeScheduler) RemoveJob(jobID string) error {
	if m.RemoveJobFunc != nil {
		return m.RemoveJobFunc(jobID)
	}
	return nil
}

// okNoticeCreate is the canonical Create Func returning a valid notice.
func okNoticeCreate(_ context.Context, _ *requests.NoticeCreateRequest, _, _ string) (*models.Notice, error) {
	return &models.Notice{BaseModel: models.BaseModel{ID: "notice-1"}}, nil
}

func newAdminNoticeHandler(svc systemServices.NoticeCacheService, ch systemServices.NotificationChannelService, sched SchedulerService) *NoticeHandler {
	h := NewNoticeHandler(svc, ch, sched)
	h.core = &core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{OperLogService: nil},
	}
	return h
}

func invokeAdminNotice(t *testing.T, method, path string, body interface{}, params gin.Params,
	handler func(*gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", "user-notice-1")
	c.Set("username", "tester")
	var reqBody io.Reader
	if body != nil {
		switch v := body.(type) {
		case string:
			reqBody = bytes.NewReader([]byte(v))
		default:
			b, _ := json.Marshal(body)
			reqBody = bytes.NewReader(b)
		}
	}
	c.Request = httptest.NewRequest(method, path, reqBody)
	c.Request.Header.Set("Content-Type", "application/json")
	if params != nil {
		c.Params = params
	}
	handler(c)
	return w
}

// ----------------------------------------------------------------------------
// WithCore
// ----------------------------------------------------------------------------

func TestNoticeHandler_WithCore(t *testing.T) {
	h := NewNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	out := h.WithCore(&core.Core{})
	assert.Same(t, h, out)
	assert.NotNil(t, h.core)

	var nilH *NoticeHandler
	assert.Nil(t, nilH.WithCore(&core.Core{}))
}

// ----------------------------------------------------------------------------
// Statistics
// ----------------------------------------------------------------------------

func TestNoticeHandler_Statistics_Success(t *testing.T) {
	svc := &mockNoticeCacheService{
		GetStatusStatisticsFunc: func(_ context.Context) (*services.NoticeStatusStatistics, error) {
			return &services.NoticeStatusStatistics{}, nil
		},
	}
	h := newAdminNoticeHandler(svc, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/statistics", nil, nil, h.Statistics)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Statistics_Error(t *testing.T) {
	svc := &mockNoticeCacheService{
		GetStatusStatisticsFunc: func(_ context.Context) (*services.NoticeStatusStatistics, error) {
			return nil, errors.New("boom")
		},
	}
	h := newAdminNoticeHandler(svc, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/statistics", nil, nil, h.Statistics)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Create
// ----------------------------------------------------------------------------

func TestNoticeHandler_Create_BindError(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices", "not-json", nil, h.Create)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Create_NoUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/system/notices", bytes.NewReader([]byte(`{"noticeTitle":"t"}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	h.Create(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Create_ServiceError(t *testing.T) {
	svc := &mockNoticeCacheService{
		CreateFunc: func(_ context.Context, _ *requests.NoticeCreateRequest, _, _ string) (*models.Notice, error) {
			return nil, errors.New("boom")
		},
	}
	h := newAdminNoticeHandler(svc, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices", map[string]interface{}{"noticeTitle": "t"}, nil, h.Create)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Create_ChannelSetError(t *testing.T) {
	ch := &mockNotificationChannelService{
		SetNotificationChannelsFunc: func(_ context.Context, _ string, _ []models.NotificationChannel) error {
			return errors.New("boom")
		},
	}
	body := map[string]interface{}{
		"noticeTitle": "t", "noticeType": "1", "noticeContent": "c",
		"channels":    []map[string]interface{}{{"channelType": "email"}},
	}
	h := newAdminNoticeHandler(&mockNoticeCacheService{CreateFunc: okNoticeCreate}, ch, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices", body, nil, h.Create)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Create_RecurringMissingCron(t *testing.T) {
	recurring := "recurring"
	body := map[string]interface{}{
		"noticeTitle":      "t", "noticeType": "1", "noticeContent": "c",
		"executionType":    recurring,
		"recurrenceConfig": map[string]interface{}{},
	}
	h := newAdminNoticeHandler(&mockNoticeCacheService{CreateFunc: okNoticeCreate}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices", body, nil, h.Create)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Create_RecurringWithCron_Success(t *testing.T) {
	recurring := "recurring"
	cron := "0 0 9 * * *"
	endDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	body := map[string]interface{}{
		"noticeTitle":   "t", "noticeType": "1", "noticeContent": "c",
		"executionType": recurring,
		"recurrenceConfig": map[string]interface{}{
			"cronExpression": cron,
			"endDate":        endDate,
		},
	}
	updateCalled := false
	svc := &mockNoticeCacheService{
		CreateFunc: okNoticeCreate,
		UpdateFunc: func(_ context.Context, _ string, _ *requests.NoticeUpdateRequest) error {
			updateCalled = true
			return nil
		},
	}
	h := newAdminNoticeHandler(svc, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices", body, nil, h.Create)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, updateCalled)
}

func TestNoticeHandler_Create_RecurringAddJobFail(t *testing.T) {
	recurring := "recurring"
	cron := "0 0 9 * * *"
	body := map[string]interface{}{
		"noticeTitle":   "t", "noticeType": "1", "noticeContent": "c",
		"executionType": recurring,
		"recurrenceConfig": map[string]interface{}{
			"cronExpression": cron,
		},
	}
	sched := &mockNoticeScheduler{
		AddJobFunc: func(_ *models.Job) error { return errors.New("boom") },
	}
	h := newAdminNoticeHandler(&mockNoticeCacheService{CreateFunc: okNoticeCreate}, &mockNotificationChannelService{}, sched)
	w := invokeAdminNotice(t, "POST", "/system/notices", body, nil, h.Create)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Create_ScheduledOnce_Success(t *testing.T) {
	body := map[string]interface{}{
		"noticeTitle": "t", "noticeType": "1", "noticeContent": "c",
		"publishTime": time.Now().Add(2 * time.Hour).Format(time.RFC3339),
	}
	h := newAdminNoticeHandler(&mockNoticeCacheService{CreateFunc: okNoticeCreate}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices", body, nil, h.Create)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Create_UpdateStatusFail(t *testing.T) {
	recurring := "recurring"
	cron := "0 0 9 * * *"
	body := map[string]interface{}{
		"noticeTitle":   "t", "noticeType": "1", "noticeContent": "c",
		"executionType": recurring,
		"recurrenceConfig": map[string]interface{}{
			"cronExpression": cron,
		},
	}
	svc := &mockNoticeCacheService{
		CreateFunc: okNoticeCreate,
		UpdateFunc: func(_ context.Context, _ string, _ *requests.NoticeUpdateRequest) error {
			return errors.New("boom")
		},
	}
	h := newAdminNoticeHandler(svc, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices", body, nil, h.Create)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Create_NilScheduler_PlainSuccess(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{CreateFunc: okNoticeCreate}, &mockNotificationChannelService{}, nil)
	w := invokeAdminNotice(t, "POST", "/system/notices", map[string]interface{}{"noticeTitle": "t", "noticeType": "1", "noticeContent": "c"}, nil, h.Create)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Create_NilScheduler_Recurring(t *testing.T) {
	recurring := "recurring"
	body := map[string]interface{}{
		"noticeTitle":   "t", "noticeType": "1", "noticeContent": "c",
		"executionType": recurring,
	}
	h := newAdminNoticeHandler(&mockNoticeCacheService{CreateFunc: okNoticeCreate}, &mockNotificationChannelService{}, nil)
	w := invokeAdminNotice(t, "POST", "/system/notices", body, nil, h.Create)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// List
// ----------------------------------------------------------------------------

func TestNoticeHandler_List_Success(t *testing.T) {
	svc := &mockNoticeCacheService{
		ListFunc: func(_ context.Context, _ requests.NoticeListParams) (*systemServices.PageResult, error) {
			return &systemServices.PageResult{List: []models.Notice{}, Total: 0, Current: 1, PageSize: 10}, nil
		},
	}
	h := newAdminNoticeHandler(svc, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/list", map[string]interface{}{
		"current": 1, "pageSize": 10, "noticeTitle": "x", "noticeType": "1",
		"createTime": "2026-01-01", "orderByColumn": "created_at", "isAsc": true,
	}, nil, h.List)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_List_BindErrorFallback(t *testing.T) {
	svc := &mockNoticeCacheService{
		ListFunc: func(_ context.Context, _ requests.NoticeListParams) (*systemServices.PageResult, error) {
			return &systemServices.PageResult{List: []models.Notice{}, Total: 0, Current: 1, PageSize: 10}, nil
		},
	}
	h := newAdminNoticeHandler(svc, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/list", "not-json", nil, h.List)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_List_ServiceError(t *testing.T) {
	svc := &mockNoticeCacheService{
		ListFunc: func(_ context.Context, _ requests.NoticeListParams) (*systemServices.PageResult, error) {
			return nil, errors.New("boom")
		},
	}
	h := newAdminNoticeHandler(svc, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/list", nil, nil, h.List)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// GetByID
// ----------------------------------------------------------------------------

func TestNoticeHandler_GetByID_EmptyID(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "GET", "/system/notices/", nil, gin.Params{{Key: "id", Value: ""}}, h.GetByID)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_GetByID_Success(t *testing.T) {
	svc := &mockNoticeCacheService{
		GetNoticeByIDFunc: func(_ context.Context, id string) (*models.Notice, error) {
			return &models.Notice{BaseModel: models.BaseModel{ID: id}}, nil
		},
	}
	h := newAdminNoticeHandler(svc, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "GET", "/system/notices/n1", nil, gin.Params{{Key: "id", Value: "n1"}}, h.GetByID)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_GetByID_Error(t *testing.T) {
	svc := &mockNoticeCacheService{
		GetNoticeByIDFunc: func(_ context.Context, _ string) (*models.Notice, error) {
			return nil, errors.New("boom")
		},
	}
	h := newAdminNoticeHandler(svc, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "GET", "/system/notices/n1", nil, gin.Params{{Key: "id", Value: "n1"}}, h.GetByID)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Update
// ----------------------------------------------------------------------------

func TestNoticeHandler_Update_EmptyID(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices//update", nil, gin.Params{{Key: "id", Value: ""}}, h.Update)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Update_BindError(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/n1/update", "not-json", gin.Params{{Key: "id", Value: "n1"}}, h.Update)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Update_ServiceError(t *testing.T) {
	svc := &mockNoticeCacheService{
		UpdateFunc: func(_ context.Context, _ string, _ *requests.NoticeUpdateRequest) error {
			return errors.New("boom")
		},
	}
	h := newAdminNoticeHandler(svc, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/n1/update", map[string]interface{}{"noticeTitle": "t"}, gin.Params{{Key: "id", Value: "n1"}}, h.Update)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Update_ChannelError(t *testing.T) {
	ch := &mockNotificationChannelService{
		SetNotificationChannelsFunc: func(_ context.Context, _ string, _ []models.NotificationChannel) error {
			return errors.New("boom")
		},
	}
	body := map[string]interface{}{
		"noticeTitle": "t", "noticeType": "1", "noticeContent": "c",
		"channels":    []map[string]interface{}{{"channelType": "email"}},
	}
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, ch, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/n1/update", body, gin.Params{{Key: "id", Value: "n1"}}, h.Update)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Update_ScheduledAndSuccess(t *testing.T) {
	title := "later"
	body := map[string]interface{}{
		"noticeTitle": title,
		"publishTime": time.Now().Add(2 * time.Hour).Format(time.RFC3339),
	}
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/n1/update", body, gin.Params{{Key: "id", Value: "n1"}}, h.Update)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Update_ScheduledAddJobFail(t *testing.T) {
	body := map[string]interface{}{
		"noticeTitle": "t", "noticeType": "1", "noticeContent": "c",
		"publishTime": time.Now().Add(2 * time.Hour).Format(time.RFC3339),
	}
	sched := &mockNoticeScheduler{AddJobFunc: func(_ *models.Job) error { return errors.New("boom") }}
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, sched)
	w := invokeAdminNotice(t, "POST", "/system/notices/n1/update", body, gin.Params{{Key: "id", Value: "n1"}}, h.Update)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// Delete / BatchDelete
// ----------------------------------------------------------------------------

func TestNoticeHandler_Delete_EmptyID(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices//delete", nil, gin.Params{{Key: "id", Value: ""}}, h.Delete)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Delete_Error(t *testing.T) {
	svc := &mockNoticeCacheService{DeleteFunc: func(_ context.Context, _ string) error { return errors.New("boom") }}
	h := newAdminNoticeHandler(svc, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/n1/delete", nil, gin.Params{{Key: "id", Value: "n1"}}, h.Delete)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Delete_Success(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/n1/delete", nil, gin.Params{{Key: "id", Value: "n1"}}, h.Delete)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_BatchDelete_BindError(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/batch", map[string]interface{}{}, nil, h.BatchDelete)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_BatchDelete_Error(t *testing.T) {
	svc := &mockNoticeCacheService{BatchDeleteFunc: func(_ context.Context, _ []string) error { return errors.New("boom") }}
	h := newAdminNoticeHandler(svc, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/batch", map[string]interface{}{"ids": []string{"a", "b"}}, nil, h.BatchDelete)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_BatchDelete_Success(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/batch", map[string]interface{}{"ids": []string{"a", "b"}}, nil, h.BatchDelete)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// GetStatistics / Publish / Withdraw
// ----------------------------------------------------------------------------

func TestNoticeHandler_GetStatistics_EmptyID(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "GET", "/system/notices//statistics", nil, gin.Params{{Key: "id", Value: ""}}, h.GetStatistics)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_GetStatistics_Error(t *testing.T) {
	svc := &mockNoticeCacheService{
		GetStatisticsFunc: func(_ context.Context, _ string) (*models.NoticeStatistics, error) {
			return nil, errors.New("boom")
		},
	}
	h := newAdminNoticeHandler(svc, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "GET", "/system/notices/n1/statistics", nil, gin.Params{{Key: "id", Value: "n1"}}, h.GetStatistics)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_GetStatistics_Success(t *testing.T) {
	svc := &mockNoticeCacheService{
		GetStatisticsFunc: func(_ context.Context, _ string) (*models.NoticeStatistics, error) {
			return &models.NoticeStatistics{}, nil
		},
	}
	h := newAdminNoticeHandler(svc, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "GET", "/system/notices/n1/statistics", nil, gin.Params{{Key: "id", Value: "n1"}}, h.GetStatistics)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Publish_EmptyID(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices//publish", nil, gin.Params{{Key: "id", Value: ""}}, h.Publish)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Publish_Error(t *testing.T) {
	ch := &mockNotificationChannelService{
		PublishAndSendNoticeFunc: func(_ context.Context, _ string) error { return errors.New("boom") },
	}
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, ch, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/n1/publish", nil, gin.Params{{Key: "id", Value: "n1"}}, h.Publish)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Publish_Success(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/n1/publish", nil, gin.Params{{Key: "id", Value: "n1"}}, h.Publish)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Withdraw_EmptyID(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices//withdraw", nil, gin.Params{{Key: "id", Value: ""}}, h.Withdraw)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Withdraw_Error(t *testing.T) {
	svc := &mockNoticeCacheService{WithdrawFunc: func(_ context.Context, _ string) error { return errors.New("boom") }}
	h := newAdminNoticeHandler(svc, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/n1/withdraw", nil, gin.Params{{Key: "id", Value: "n1"}}, h.Withdraw)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_Withdraw_Success(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/n1/withdraw", nil, gin.Params{{Key: "id", Value: "n1"}}, h.Withdraw)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ----------------------------------------------------------------------------
// GetChannels / SetChannels / GetCronExpressions
// ----------------------------------------------------------------------------

func TestNoticeHandler_GetChannels_EmptyID(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "GET", "/system/notices//channels", nil, gin.Params{{Key: "id", Value: ""}}, h.GetChannels)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_GetChannels_Error(t *testing.T) {
	ch := &mockNotificationChannelService{
		GetNotificationChannelsFunc: func(_ context.Context, _ string) ([]models.NotificationChannel, error) {
			return nil, errors.New("boom")
		},
	}
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, ch, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "GET", "/system/notices/n1/channels", nil, gin.Params{{Key: "id", Value: "n1"}}, h.GetChannels)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_GetChannels_Success(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "GET", "/system/notices/n1/channels", nil, gin.Params{{Key: "id", Value: "n1"}}, h.GetChannels)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_SetChannels_EmptyID(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices//channels", nil, gin.Params{{Key: "id", Value: ""}}, h.SetChannels)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_SetChannels_BindError(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/n1/channels", "not-json", gin.Params{{Key: "id", Value: "n1"}}, h.SetChannels)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_SetChannels_Error(t *testing.T) {
	ch := &mockNotificationChannelService{
		SetNotificationChannelsFunc: func(_ context.Context, _ string, _ []models.NotificationChannel) error {
			return errors.New("boom")
		},
	}
	body := []map[string]interface{}{{"channelType": "email"}}
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, ch, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/n1/channels", body, gin.Params{{Key: "id", Value: "n1"}}, h.SetChannels)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_SetChannels_Success(t *testing.T) {
	body := []map[string]interface{}{
		{"channelType": "email", "emailConfigId": "e1"},
		{"channelType": "api", "apiConfigId": "a1", "customRecipients": []string{"u1"}},
	}
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "POST", "/system/notices/n1/channels", body, gin.Params{{Key: "id", Value: "n1"}}, h.SetChannels)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoticeHandler_GetCronExpressions(t *testing.T) {
	h := newAdminNoticeHandler(&mockNoticeCacheService{}, &mockNotificationChannelService{}, &mockNoticeScheduler{})
	w := invokeAdminNotice(t, "GET", "/system/notices/cron-expressions", nil, nil, h.GetCronExpressions)
	assert.Equal(t, http.StatusOK, w.Code)
}
