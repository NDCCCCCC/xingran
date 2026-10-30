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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	addomainServices "github.com/xingran-next/xingran-go-backend/internal/services/addomain"
)

// ============================================================================
// Phase 59 / GUARD-02：对账只读 handler 测试
//
// 同包内用 &ADReconciliationHandler{svc: fake} 注入 fake service（记录收到的
// query 并返回预置行/错误），gin.CreateTestContext + httptest 构造请求。
// 覆盖：非法 JSON → 400；非法 mismatchType → 400；缺省分页归一 1/10；
// service 错误 → 非 2xx；成功 → code 0 且 data 含 list/total/current/pageSize。
// ============================================================================

// fakeDeptOUMismatchService 实现 DeptOUMismatchService，记录收到的请求。
type fakeDeptOUMismatchService struct {
	gotReq *addomainServices.DeptOUMismatchQuery
	rows   []addomainServices.DeptOUMismatchRow
	total  int64
	err    error
	calls  int
}

func (f *fakeDeptOUMismatchService) ListDeptOUMismatch(
	_ context.Context, req *addomainServices.DeptOUMismatchQuery,
) ([]addomainServices.DeptOUMismatchRow, int64, error) {
	f.calls++
	// 值拷贝快照：handler 归一化会原地改写 req 的 Current/PageSize，
	// 断言必须看「fake 被调用那一刻」的值。
	f.gotReq = &addomainServices.DeptOUMismatchQuery{
		Current:      req.Current,
		PageSize:     req.PageSize,
		MismatchType: req.MismatchType,
		DeptID:       req.DeptID,
		Keyword:      req.Keyword,
	}
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.rows, f.total, nil
}

// invokeReconHandler 以给定 body 调用 handler 并返回响应 recorder。
func invokeReconHandler(t *testing.T, body string, h *ADReconciliationHandler) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/ad-domain/reconciliation/mismatch", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.ListDeptOUMismatch(c)
	return w
}

// reconResponse 对账端点的响应 envelope（只解测试关心的字段）。
type reconResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *struct {
		List     []addomainServices.DeptOUMismatchRow `json:"list"`
		Total    int64                                `json:"total"`
		Current  int                                  `json:"current"`
		PageSize int                                  `json:"pageSize"`
	} `json:"data"`
}

func parseReconResponse(t *testing.T, w *httptest.ResponseRecorder) reconResponse {
	t.Helper()
	var resp reconResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "响应体应为 JSON: %s", w.Body.String())
	return resp
}

// TestPhase59_ReconHandler_InvalidJSON 非法 JSON → HTTP 400 / code 1001。
func TestPhase59_ReconHandler_InvalidJSON(t *testing.T) {
	h := &ADReconciliationHandler{svc: &fakeDeptOUMismatchService{}}
	w := invokeReconHandler(t, `{invalid`, h)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseReconResponse(t, w)
	assert.Equal(t, 1001, resp.Code, "CodeParamError")
	assert.Zero(t, h.svc.(*fakeDeptOUMismatchService).calls, "绑定失败不得触达 service")
}

// TestPhase59_ReconHandler_InvalidMismatchType 非法 mismatchType → HTTP 400，不触达 service。
func TestPhase59_ReconHandler_InvalidMismatchType(t *testing.T) {
	h := &ADReconciliationHandler{svc: &fakeDeptOUMismatchService{}}
	w := invokeReconHandler(t, `{"mismatchType":"bogus"}`, h)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseReconResponse(t, w)
	assert.Equal(t, 1003, resp.Code, "CodeParamInvalid")
	assert.Zero(t, h.svc.(*fakeDeptOUMismatchService).calls, "白名单拦截发生在 service 之前")
}

// TestPhase59_ReconHandler_DefaultPagination 缺省分页必须归一为 1/10 再传给 service。
func TestPhase59_ReconHandler_DefaultPagination(t *testing.T) {
	svc := &fakeDeptOUMismatchService{
		rows:  []addomainServices.DeptOUMismatchRow{},
		total: 0,
	}
	h := &ADReconciliationHandler{svc: svc}
	w := invokeReconHandler(t, `{}`, h)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, svc.gotReq)
	assert.Equal(t, 1, svc.gotReq.Current, "current 缺省归一为 1")
	assert.Equal(t, 10, svc.gotReq.PageSize, "pageSize 缺省归一为 10")
}

// TestPhase59_ReconHandler_PageSizeCap 上限 200（T-59-08 DoS 防线）。
func TestPhase59_ReconHandler_PageSizeCap(t *testing.T) {
	svc := &fakeDeptOUMismatchService{rows: []addomainServices.DeptOUMismatchRow{}}
	h := &ADReconciliationHandler{svc: svc}
	w := invokeReconHandler(t, `{"current":2,"pageSize":9999}`, h)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, svc.gotReq)
	assert.Equal(t, 2, svc.gotReq.Current)
	assert.Equal(t, 200, svc.gotReq.PageSize, "pageSize 超限必须钳制到 200")
}

// TestPhase59_ReconHandler_ServiceError service 失败 → 非 2xx。
func TestPhase59_ReconHandler_ServiceError(t *testing.T) {
	svc := &fakeDeptOUMismatchService{err: errors.New("查询部门-OU 对账条目失败: boom")}
	h := &ADReconciliationHandler{svc: svc}
	w := invokeReconHandler(t, `{}`, h)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotEqual(t, 0, parseReconResponse(t, w).Code, "错误响应不得返回 code 0")
}

// TestPhase59_ReconHandler_SuccessPageData 成功 → code 0 且 data 含 list/total/current/pageSize。
func TestPhase59_ReconHandler_SuccessPageData(t *testing.T) {
	deptID := "dept-1"
	username := "zhangsan"
	svc := &fakeDeptOUMismatchService{
		rows: []addomainServices.DeptOUMismatchRow{
			{
				DeptID:       &deptID,
				Username:     &username,
				MismatchType: "user_ou_mismatch",
				MismatchDetail: "用户 OU「OU=Sales,DC=corp,DC=com」与部门映射 OU「OU=IT,DC=corp,DC=com」不一致",
			},
		},
		total: 1,
	}
	h := &ADReconciliationHandler{svc: svc}
	w := invokeReconHandler(t, `{"current":1,"pageSize":10,"mismatchType":"user_ou_mismatch"}`, h)

	require.Equal(t, http.StatusOK, w.Code)
	resp := parseReconResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	require.NotNil(t, resp.Data, "成功响应必须携带 data")
	assert.Equal(t, int64(1), resp.Data.Total)
	assert.Equal(t, 1, resp.Data.Current)
	assert.Equal(t, 10, resp.Data.PageSize)
	require.Len(t, resp.Data.List, 1)
	assert.Equal(t, "user_ou_mismatch", resp.Data.List[0].MismatchType)
	// 透传断言：筛选条件必须原样到达 service
	require.NotNil(t, svc.gotReq)
	assert.Equal(t, "user_ou_mismatch", svc.gotReq.MismatchType)
}

// TestPhase59_ReconHandler_EmptyListIsArray 空结果序列化为 [] 而非 null。
func TestPhase59_ReconHandler_EmptyListIsArray(t *testing.T) {
	svc := &fakeDeptOUMismatchService{rows: []addomainServices.DeptOUMismatchRow{}}
	h := &ADReconciliationHandler{svc: svc}
	w := invokeReconHandler(t, `{}`, h)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"list":[]`,
		"空 list 必须序列化为 []，null 会破坏前端分页组件的默认行为")
}
