package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
)

func init() { gin.SetMode(gin.TestMode) }

// newTestCtx 返回 httptest.ResponseRecorder + 已注入 request_id 的 gin.Context。
func newTestCtx(t *testing.T, requestID string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	if requestID != "" {
		c.Set("request_id", requestID)
	}
	return c, w
}

func decodeResp(t *testing.T, w *httptest.ResponseRecorder) Response {
	t.Helper()
	var got Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	return got
}

func TestSuccess_WithDataAndEmpty(t *testing.T) {
	c, w := newTestCtx(t, "req-success")
	Success(c, map[string]string{"k": "v"})
	assert.Equal(t, http.StatusOK, w.Code)
	got := decodeResp(t, w)
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "success", got.Message)
	assert.Equal(t, "req-success", got.RequestID)

	// 无 data → nil
	c2, w2 := newTestCtx(t, "")
	Success(c2)
	got2 := decodeResp(t, w2)
	assert.Nil(t, got2.Data)
}

func TestError_Path(t *testing.T) {
	c, w := newTestCtx(t, "req-err")
	Error(c, ErrBadRequest, "自定义提示")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	got := decodeResp(t, w)
	assert.Equal(t, 400, got.Code)
	assert.Equal(t, "自定义提示", got.Message)
	assert.Equal(t, "req-err", got.RequestID)
}

func TestErrorWithData(t *testing.T) {
	c, w := newTestCtx(t, "req-errdata")
	ErrorWithData(c, ErrForbidden, map[string]string{"hint": "no"}, "禁止")
	assert.Equal(t, http.StatusForbidden, w.Code)
	got := decodeResp(t, w)
	assert.Equal(t, 403, got.Code)
	assert.Equal(t, "禁止", got.Message)
	assert.NotNil(t, got.Data)
}

func TestPage(t *testing.T) {
	c, w := newTestCtx(t, "req-page")
	Page(c, []int{1, 2, 3}, 10, 1, 5)
	got := decodeResp(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, got.Code)
	page, ok := got.Data.(map[string]interface{})
	require.True(t, ok, "Data 应该是 JSON 对象")
	assert.Equal(t, float64(10), page["total"])
	assert.Equal(t, float64(1), page["current"])
	assert.Equal(t, float64(5), page["pageSize"])
}

func TestToAppError_Branches(t *testing.T) {
	// *AppError 直接返回
	e := toAppError(ErrUnauthorized)
	assert.Same(t, ErrUnauthorized, e)

	// int → 400
	e = toAppError(123)
	assert.Equal(t, 123, e.Code)
	assert.Equal(t, http.StatusBadRequest, e.HTTPStatus)

	// string → 500
	e = toAppError("boom")
	assert.Equal(t, defaultServerErrorCode, e.Code)
	assert.Equal(t, "boom", e.Message)

	// 普通 error → 500 with err msg
	e = toAppError(errors.New("plain"))
	assert.Equal(t, defaultServerErrorCode, e.Code)
	assert.Equal(t, "plain", e.Message)

	// 未知类型 → ErrServerError
	e = toAppError(struct{}{})
	assert.Same(t, ErrServerError, e)
}

func TestToAppError_PkgAppError(t *testing.T) {
	// pkg/errors.AppError
	pkgErr := &apperrors.AppError{Code: 1001, Message: "参数错了"}
	e := toAppError(pkgErr)
	assert.Equal(t, 1001, e.Code)
	assert.Equal(t, "参数错了", e.Message)
	assert.Equal(t, http.StatusBadRequest, e.HTTPStatus)
}

func TestAppError_Error(t *testing.T) {
	assert.Equal(t, "abc", (&AppError{Message: "abc"}).Error())
}

func TestGetOptionalData(t *testing.T) {
	assert.Nil(t, getOptionalData(nil))
	assert.Equal(t, 42, getOptionalData([]interface{}{42}))
}

func TestGetRequestID(t *testing.T) {
	// 无 request_id → ""
	c, _ := newTestCtx(t, "")
	assert.Equal(t, "", getRequestID(c))

	// 非 string 类型 → ""
	c2, _ := newTestCtx(t, "")
	c2.Set("request_id", 123)
	assert.Equal(t, "", getRequestID(c2))

	// 字符串 → 原样
	c3, _ := newTestCtx(t, "abc")
	assert.Equal(t, "abc", getRequestID(c3))
}

func TestErrorCodes_Registration(t *testing.T) {
	// 触发所有 AppError 变量初始化覆盖(ErrSuccess Code=0 排除)
	all := []*AppError{
		ErrServerError, ErrBadRequest, ErrUnauthorized, ErrForbidden,
		ErrNotFound, ErrMethodNotAllowed, ErrTokenExpired, ErrTokenInvalid, ErrTokenNotValidYet,
		ErrUserNotFound, ErrPasswordError, ErrUserDisabled, ErrCaptchaError, ErrParamError,
		ErrRecordExists, ErrRecordNotFound, ErrNotImplemented, ErrPasswordHashFailed,
		ErrPasswordDecrypt, ErrCredentialInvalid, ErrDatabaseError,
	}
	assert.Len(t, all, 21)
	for _, e := range all {
		assert.NotZero(t, e.Code)
	}
	// ErrSuccess 显式校验
	assert.Equal(t, 0, ErrSuccess.Code)
}

// =====================================================================
// handler_helpers.go: HandleJSONBinding / HandleServiceError /
// HandleIDParam / HandleGetByID。
// =====================================================================

func TestHandleJSONBinding(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	// 成功
	c, _ := newTestCtx(t, "")
	c.Request = httptest.NewRequest(http.MethodPost, "/",
 strings.NewReader(`{"name":"alice"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	var okPayload payload
	require.True(t, HandleJSONBinding(c, &okPayload))
	assert.Equal(t, "alice", okPayload.Name)

	// 失败
	c2, w2 := newTestCtx(t, "")
	c2.Request = httptest.NewRequest(http.MethodPost, "/",
 strings.NewReader(`{`))
	c2.Request.Header.Set("Content-Type", "application/json")
	var bad payload
	require.False(t, HandleJSONBinding(c2, &bad))
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

// newRouterCtx 注册带 id 路径参数的路由,模拟真实 handler 调用。
func newRouterCtx(t *testing.T, requestID string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/items", nil)
	if requestID != "" {
		c.Set("request_id", requestID)
	}
	// gin.CreateTestContext 已绑定默认 engine,通过 params 手动注入路由参数
	c.Params = gin.Params{{Key: "id", Value: ""}}
	return c, w
}

func TestHandleServiceError(t *testing.T) {
	c, _ := newRouterCtx(t, "")
	require.True(t, HandleServiceError(c, nil, "测试"))

	// QUIRK: HandleServiceError 用 Error(c, http.StatusInternalServerError, ...)
	// 但 Error 接 int → toAppError 把 int 视作 Code 而非 HTTPStatus,
	// 实际 HTTPStatus 落到 400(D-12 不修)。
	c2, w2 := newRouterCtx(t, "")
	require.False(t, HandleServiceError(c2, errors.New("boom"), "写"))
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

func TestHandleIDParam(t *testing.T) {
	// id 为空 → 拦截
	c, w := newRouterCtx(t, "")
	_, ok := HandleIDParam(c)
	require.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// id 有值 → 放行
	c2, _ := newRouterCtx(t, "")
	c2.Params = gin.Params{{Key: "id", Value: "abc"}}
	id, ok := HandleIDParam(c2)
	require.True(t, ok)
	assert.Equal(t, "abc", id)
}

func TestHandleGetByID(t *testing.T) {
	// 缺 id → false
	c, _ := newRouterCtx(t, "")
	require.False(t, HandleGetByID(c, func(string) (interface{}, error) { return nil, nil }, "未找到"))

	// getter 报错 → false
	// QUIRK: HandleGetByID 用 Error(c, http.StatusNotFound, ...) 但 Error 接 int
	// → toAppError 把 int 视作 Code 而非 HTTPStatus,实际 HTTPStatus 落到 400。
	c2, w2 := newRouterCtx(t, "")
	c2.Params = gin.Params{{Key: "id", Value: "abc"}}
	require.False(t, HandleGetByID(c2, func(string) (interface{}, error) {
		return nil, errors.New("nf")
	}, "未找到"))
	assert.Equal(t, http.StatusBadRequest, w2.Code)

	// 成功 → true
	c3, w3 := newRouterCtx(t, "")
	c3.Params = gin.Params{{Key: "id", Value: "abc"}}
	require.True(t, HandleGetByID(c3, func(s string) (interface{}, error) {
		return map[string]string{"id": s}, nil
	}, "未找到"))
	assert.Equal(t, http.StatusOK, w3.Code)
}