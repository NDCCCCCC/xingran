package system

// Phase 144 · 漂移告警端点单测（response 包装 / operlog 位置与脱敏 / 分页）。
//
// 零真机、零生产库：sqlite 内存库 + 同步 stub recorder（捕获 RecordAsync 参数）。
// 权限分组（读组 RequirePermissions / 写组 RequireStrictPermissions）由
// ad_drift_alert_router.go 静态挂载 + pkg/permission/config_test.go 死表断言守护。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
)

// driftRecordedLog 捕获一条 RecordAsync 调用。
type driftRecordedLog struct {
	title        string
	businessType int
	operParam    *string
}

// stubDriftOperLog 同步捕获 recorder（满足 services.OperLogService）。
type stubDriftOperLog struct {
	mu   sync.Mutex
	logs []driftRecordedLog
}

func (s *stubDriftOperLog) RecordOperLog(_ context.Context, _ *gorm.DB, _ *models.OperLog) error {
	return nil
}
func (s *stubDriftOperLog) RecordFromGinContext(_ *gin.Context, _ *gorm.DB, _ string, _ int, _ string) {
}
func (s *stubDriftOperLog) RecordAsync(_ *gorm.DB, title string, businessType int, _, _, _ string,
	_, _, _ *string, _ *string, operParam, _, _ *string, _ int, _ int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, driftRecordedLog{title: title, businessType: businessType, operParam: operParam})
}

func (s *stubDriftOperLog) snapshot() []driftRecordedLog {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]driftRecordedLog(nil), s.logs...)
}

func setupDriftAlertHandler(t *testing.T) (*AdDriftAlertHandler, *stubDriftOperLog, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Config{}, &models.OperLog{}, &models.Notice{}, &models.ADGroupMember{}))

	rec := &stubDriftOperLog{}
	c := &core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{OperLogService: rec},
	}
	h := &AdDriftAlertHandler{
		core: c,
		db:   db,
		svc:  addomain.NewADSyncDriftAlertService(db),
	}
	return h, rec, db
}

func doDriftJSON(t *testing.T, h gin.HandlerFunc, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var req *http.Request
	if body != nil {
		raw, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c.Request = req
	h(c)
	return w
}

// driftResp 通用响应壳（response.Success → code=0）。
type driftResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func TestPhase144_Handler_ReadEndpointsUseResponseWrapper(t *testing.T) {
	h, rec, _ := setupDriftAlertHandler(t)

	for _, tc := range []struct {
		name string
		fn   gin.HandlerFunc
		path string
		body interface{}
	}{
		{"status", h.GetStatus, "/status", map[string]interface{}{}},
		{"logs", h.ListLogs, "/logs", map[string]interface{}{"current": 1, "pageSize": 10}},
		{"config", h.GetConfig, "/config", map[string]interface{}{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := doDriftJSON(t, tc.fn, http.MethodPost, "/system/ad-drift-alert"+tc.path, tc.body)
			require.Equal(t, http.StatusOK, w.Code)
			var resp driftResp
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, 0, resp.Code, "必须 response.Success 包装")
			assert.NotEmpty(t, resp.Data)
		})
	}
	// 读端点不得写 operlog
	assert.Empty(t, rec.snapshot())
}

func TestPhase144_Handler_LogsPaginationShape(t *testing.T) {
	h, _, db := setupDriftAlertHandler(t)
	for i := 0; i < 3; i++ {
		require.NoError(t, db.Create(&models.OperLog{Title: "AD 同步漂移告警", Method: "ADSyncDriftAlertService.CheckAndAlert"}).Error)
	}
	w := doDriftJSON(t, h.ListLogs, http.MethodPost, "/logs", map[string]interface{}{"current": 1, "pageSize": 2})
	var resp driftResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	var page struct {
		List     []map[string]interface{} `json:"list"`
		Total    int64                    `json:"total"`
		Current  int                      `json:"current"`
		PageSize int                      `json:"pageSize"`
	}
	require.NoError(t, json.Unmarshal(resp.Data, &page))
	assert.Equal(t, int64(3), page.Total)
	assert.Equal(t, 1, page.Current)
	assert.Equal(t, 2, page.PageSize)
	assert.Len(t, page.List, 2)
}

func TestPhase144_Handler_UpdateConfig_RecordWithBodyMasksSecrets(t *testing.T) {
	h, rec, db := setupDriftAlertHandler(t)

	// 请求体含明文 webhook URL / token（模拟敏感配置写）。
	body := map[string]interface{}{
		"ad_sync.drift_alert.ws_enabled": "true",
		"webhookUrl":                     "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=PLAIN-KEY",
		"webhookKey":                     "PLAIN-WEBHOOK-KEY",
		"access_token":                   "PLAIN-TOKEN",
	}
	w := doDriftJSON(t, h.UpdateConfig, http.MethodPost, "/config/update", body)
	require.Equal(t, http.StatusOK, w.Code)
	var resp driftResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	logs := rec.snapshot()
	require.Len(t, logs, 1, "写端点必须在 success path 记录 operlog")
	assert.Equal(t, "同步漂移告警配置", logs[0].title)
	assert.Equal(t, 2, logs[0].businessType, "OperTypeUpdate=2")
	require.NotNil(t, logs[0].operParam, "RecordWithBody 必须落 oper_param")
	assert.Contains(t, *logs[0].operParam, "******")
	assert.NotContains(t, *logs[0].operParam, "PLAIN-KEY", "明文 webhook key 不得进 oper_param")
	assert.NotContains(t, *logs[0].operParam, "PLAIN-WEBHOOK-KEY")
	assert.NotContains(t, *logs[0].operParam, "PLAIN-TOKEN")

	// 白名单内旋钮确实落库；白名单外字段（webhookUrl 等）被忽略
	var v string
	require.NoError(t, db.Raw("SELECT config_value FROM sys_config WHERE config_key = ?", "ad_sync.drift_alert.ws_enabled").Scan(&v).Error)
	assert.Equal(t, "true", v)
	var n int64
	require.NoError(t, db.Model(&models.Config{}).Where("config_key = ?", "webhookUrl").Count(&n).Error)
	assert.Equal(t, int64(0), n)
}

func TestPhase144_Handler_Trigger_RecordsOperTypeSync(t *testing.T) {
	h, rec, _ := setupDriftAlertHandler(t)

	w := doDriftJSON(t, h.Trigger, http.MethodPost, "/trigger", map[string]interface{}{})
	require.Equal(t, http.StatusOK, w.Code)
	var resp driftResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	logs := rec.snapshot()
	require.Len(t, logs, 1, "trigger 必须在 success path 记录 operlog")
	assert.Equal(t, "同步漂移告警", logs[0].title)
	assert.Equal(t, 14, logs[0].businessType, "OperTypeSync=14")
}

// B-1 / quick 261110-161：MarkLogRead 按 logId 反查 noticeId 并复用
// NoticeService.MarkNoticeRead 写 sys_notice_read；success path 记 operlog。
func doDriftWithUser(t *testing.T, h gin.HandlerFunc, body interface{}, userID string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/system/ad-drift-alert/logs/read", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	if userID != "" {
		c.Set("user_id", userID)
	}
	h(c)
	return w
}

func TestB1_Handler_MarkLogRead(t *testing.T) {
	h, rec, db := setupDriftAlertHandler(t)
	// 用真实模型 AutoMigrate，列类型（uuid 文本主键）与生产一致；
	// 手建 INTEGER 主键会让 MarkNoticeRead 插入 id="" 时 datatype mismatch。
	require.NoError(t, db.AutoMigrate(&models.NoticeRead{}))
	// 一条触发记录带 noticeId="n-read-1"
	jr := `{"memberCount":10,"noticeId":"n-read-1"}`
	logRow := &models.OperLog{Title: "AD 同步漂移告警", Method: addomain.DriftAlertTriggerLogMethod, JsonResult: &jr}
	require.NoError(t, db.Create(logRow).Error)

	// 1) 无登录态 → 401（requireUserID）
	w := doDriftWithUser(t, h.MarkLogRead, map[string]string{"logId": logRow.ID}, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 2) 成功：写 sys_notice_read + operlog
	w = doDriftWithUser(t, h.MarkLogRead, map[string]string{"logId": logRow.ID}, "reader-1")
	require.Equal(t, http.StatusOK, w.Code)
	var resp driftResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	var readCount int64
	require.NoError(t, db.Table("sys_notice_read").
		Where("notice_id = ? AND user_id = ?", "n-read-1", "reader-1").Count(&readCount).Error)
	assert.Equal(t, int64(1), readCount, "复用既有已读设施写入一行")
	logs := rec.snapshot()
	require.Len(t, logs, 1, "标记已读属写操作，success path 必须记 operlog")

	// 3) 重复标记幂等（MarkNoticeRead 已读跳过）
	w = doDriftWithUser(t, h.MarkLogRead, map[string]string{"logId": logRow.ID}, "reader-1")
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, db.Table("sys_notice_read").
		Where("notice_id = ? AND user_id = ?", "n-read-1", "reader-1").Count(&readCount).Error)
	assert.Equal(t, int64(1), readCount, "重复标记不得插入第二行")

	// 4) 无关联通知（旧格式 JsonResult）→ 400，不静默成功
	jrOld := `{"memberCount":11}`
	oldRow := &models.OperLog{Title: "AD 同步漂移告警", Method: addomain.DriftAlertTriggerLogMethod, JsonResult: &jrOld}
	require.NoError(t, db.Create(oldRow).Error)
	w = doDriftWithUser(t, h.MarkLogRead, map[string]string{"logId": oldRow.ID}, "reader-1")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 5) 不存在的 logId → 400
	w = doDriftWithUser(t, h.MarkLogRead, map[string]string{"logId": "no-such-log"}, "reader-1")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
