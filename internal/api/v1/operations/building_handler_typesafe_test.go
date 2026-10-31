package operations

// building_handler_typesafe_test.go — BuildingHandlerTypeSafe 全函数覆盖（D-08 mock 模式）。
// 背景：该文件自引入起零测试（40 个 0% 函数之一），是 check-coverage.sh 中
// internal/api/v1/operations 单包 floor 例外（68.0）的直接成因；补齐后应删除例外。

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	opsModels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
)

// mockBuildingServiceTypeSafe implements opsServices.BuildingServiceTypeSafe (D-08 mock pattern).
type mockBuildingServiceTypeSafe struct {
	createErr error
	updateErr error
	deleteErr error
	getErr    error
	listErr   error
	batchErr  error

	capturedCreate *opsModels.OpsBuilding
	capturedUpdate *opsModels.OpsBuilding
	capturedDelete string
	capturedList   requests.BuildingListRequest
	capturedBatch  []string
}

func (m *mockBuildingServiceTypeSafe) Create(ctx context.Context, b *opsModels.OpsBuilding) error {
	m.capturedCreate = b
	return m.createErr
}

func (m *mockBuildingServiceTypeSafe) Update(ctx context.Context, b *opsModels.OpsBuilding) error {
	m.capturedUpdate = b
	return m.updateErr
}

func (m *mockBuildingServiceTypeSafe) Delete(ctx context.Context, id string) error {
	m.capturedDelete = id
	return m.deleteErr
}

func (m *mockBuildingServiceTypeSafe) GetByID(ctx context.Context, id string) (*opsModels.OpsBuilding, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	b := &opsModels.OpsBuilding{}
	b.ID = id
	return b, nil
}

func (m *mockBuildingServiceTypeSafe) List(ctx context.Context, req requests.BuildingListRequest) (*opsServices.PageResult, error) {
	m.capturedList = req
	if m.listErr != nil {
		return nil, m.listErr
	}
	return &opsServices.PageResult{List: []opsModels.OpsBuilding{}, Total: 2, Current: 1, PageSize: 20}, nil
}

func (m *mockBuildingServiceTypeSafe) BatchDelete(ctx context.Context, ids []string) error {
	m.capturedBatch = ids
	return m.batchErr
}

// doTypeSafeRequest routes an httptest request against the type-safe handler.
func doTypeSafeRequest(h *BuildingHandlerTypeSafe, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	switch {
	case strings.HasSuffix(path, "/list"):
		r.POST(path, h.List)
	case strings.HasSuffix(path, "/batch"):
		r.POST(path, h.BatchOperation)
	case strings.HasSuffix(path, "/update"):
		r.POST(strings.Replace(path, extractID(path), ":id", 1), h.Update)
	case strings.HasSuffix(path, "/delete"):
		r.POST(strings.Replace(path, extractID(path), ":id", 1), h.Delete)
	case strings.Contains(path, "/geocode"):
		r.POST(path, h.Geocode)
	default:
		if strings.Count(path, "/") >= 2 {
			r.POST(strings.Replace(path, extractID(path), ":id", 1), h.GetByID)
		} else {
			r.POST(path, h.Create)
		}
	}
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- Create -----------------------------------------------------------------

func TestBuildingHandlerTypeSafe_Create_Success(t *testing.T) {
	svc := &mockBuildingServiceTypeSafe{}
	h := NewBuildingHandlerTypeSafe(svc, nil)

	w := doTypeSafeRequest(h, http.MethodPost, "/buildings",
		`{"name":"测试楼宇","orgId":"00000000-0000-0000-0000-000000000001"}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":0`)
	require.NotNil(t, svc.capturedCreate)
	assert.Equal(t, "测试楼宇", svc.capturedCreate.Name)
}

func TestBuildingHandlerTypeSafe_Create_InvalidJSON(t *testing.T) {
	h := NewBuildingHandlerTypeSafe(&mockBuildingServiceTypeSafe{}, nil)

	w := doTypeSafeRequest(h, http.MethodPost, "/buildings", `{invalid`)

	// appErr.HTTPStatus 透传：参数绑定错误 → HTTP 400 + 业务包络 code!=0
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "参数错误")
}

func TestBuildingHandlerTypeSafe_Create_ServiceError(t *testing.T) {
	svc := &mockBuildingServiceTypeSafe{createErr: errors.New("db down")}
	h := NewBuildingHandlerTypeSafe(svc, nil)

	w := doTypeSafeRequest(h, http.MethodPost, "/buildings", `{"name":"x"}`)

	assert.Contains(t, w.Body.String(), "创建失败")
	assert.Contains(t, w.Body.String(), `"code":1500`) // CodeServerError
}

// --- List -------------------------------------------------------------------

func TestBuildingHandlerTypeSafe_List_Success(t *testing.T) {
	svc := &mockBuildingServiceTypeSafe{}
	h := NewBuildingHandlerTypeSafe(svc, nil)

	w := doTypeSafeRequest(h, http.MethodPost, "/buildings/list",
		`{"current":2,"pageSize":15,"name":"楼","status":0}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"total":2`)
	// 类型安全断言：请求结构体字段确实被绑定
	assert.Equal(t, 2, svc.capturedList.Current)
	assert.Equal(t, 15, svc.capturedList.PageSize)
	assert.Equal(t, "楼", svc.capturedList.Name)
}

func TestBuildingHandlerTypeSafe_List_ServiceError(t *testing.T) {
	h := NewBuildingHandlerTypeSafe(&mockBuildingServiceTypeSafe{listErr: errors.New("boom")}, nil)

	w := doTypeSafeRequest(h, http.MethodPost, "/buildings/list", `{}`)

	assert.Contains(t, w.Body.String(), "查询失败")
}

// --- GetByID ----------------------------------------------------------------

func TestBuildingHandlerTypeSafe_GetByID_Success(t *testing.T) {
	h := NewBuildingHandlerTypeSafe(&mockBuildingServiceTypeSafe{}, nil)

	w := doTypeSafeRequest(h, http.MethodPost, "/buildings/b-id-1", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":0`)
	assert.Contains(t, w.Body.String(), "b-id-1")
}

func TestBuildingHandlerTypeSafe_GetByID_NotFound(t *testing.T) {
	h := NewBuildingHandlerTypeSafe(&mockBuildingServiceTypeSafe{getErr: errors.New("not found")}, nil)

	w := doTypeSafeRequest(h, http.MethodPost, "/buildings/missing", "")

	// BuildingNotFound 分支：不经过 handleServiceError，直接 response.Error
	assert.Contains(t, w.Body.String(), "楼宇不存在")
}

// --- Update -----------------------------------------------------------------

func TestBuildingHandlerTypeSafe_Update_OverwritesPathID(t *testing.T) {
	svc := &mockBuildingServiceTypeSafe{}
	h := NewBuildingHandlerTypeSafe(svc, nil)

	// body 里带恶意/错误 ID，必须被路径参数覆写
	w := doTypeSafeRequest(h, http.MethodPost, "/buildings/real-id/update",
		`{"id":"spoofed-id","name":"改名"}`)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, svc.capturedUpdate)
	assert.Equal(t, "real-id", svc.capturedUpdate.ID, "路径参数必须覆写 body 中的 id")
}

func TestBuildingHandlerTypeSafe_Update_ServiceError(t *testing.T) {
	h := NewBuildingHandlerTypeSafe(&mockBuildingServiceTypeSafe{updateErr: errors.New("boom")}, nil)

	w := doTypeSafeRequest(h, http.MethodPost, "/buildings/x/update", `{"name":"y"}`)

	assert.Contains(t, w.Body.String(), "更新失败")
}

// --- Delete -----------------------------------------------------------------

func TestBuildingHandlerTypeSafe_Delete_Success(t *testing.T) {
	svc := &mockBuildingServiceTypeSafe{}
	h := NewBuildingHandlerTypeSafe(svc, nil)

	w := doTypeSafeRequest(h, http.MethodPost, "/buildings/del-1/delete", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "del-1", svc.capturedDelete)
}

func TestBuildingHandlerTypeSafe_Delete_ServiceError(t *testing.T) {
	h := NewBuildingHandlerTypeSafe(&mockBuildingServiceTypeSafe{deleteErr: errors.New("fk")}, nil)

	w := doTypeSafeRequest(h, http.MethodPost, "/buildings/x/delete", "")

	assert.Contains(t, w.Body.String(), "删除失败")
}

// --- BatchOperation ---------------------------------------------------------

func TestBuildingHandlerTypeSafe_BatchOperation_Delete(t *testing.T) {
	svc := &mockBuildingServiceTypeSafe{}
	h := NewBuildingHandlerTypeSafe(svc, nil)

	w := doTypeSafeRequest(h, http.MethodPost, "/buildings/batch",
		`{"ids":["a","b"],"action":"delete"}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"a", "b"}, svc.capturedBatch)
}

func TestBuildingHandlerTypeSafe_BatchOperation_InvalidAction(t *testing.T) {
	svc := &mockBuildingServiceTypeSafe{}
	h := NewBuildingHandlerTypeSafe(svc, nil)

	w := doTypeSafeRequest(h, http.MethodPost, "/buildings/batch",
		`{"ids":["a"],"action":"purge"}`)

	assert.Contains(t, w.Body.String(), "不支持的操作")
	assert.Nil(t, svc.capturedBatch, "非法 action 不得触达 service")
}

// --- Geocode ----------------------------------------------------------------

func TestBuildingHandlerTypeSafe_Geocode_MissingAddress(t *testing.T) {
	// geocodingService 非nil（避免 nil deref），address 为空走 ParamMissing 早退
	h := NewBuildingHandlerTypeSafe(&mockBuildingServiceTypeSafe{}, opsServices.NewGeocodingService(""))

	w := doTypeSafeRequest(h, http.MethodPost, "/buildings/geocode", `{"address":""}`)

	// binding:"required" 先行拦截空地址
	assert.Contains(t, w.Body.String(), "参数错误")
}

func TestBuildingHandlerTypeSafe_Geocode_ServiceError_NoAPIKey(t *testing.T) {
	// 空 API key：service 层 Geocode 必然返回错误 → handleServiceError false 分支
	h := NewBuildingHandlerTypeSafe(&mockBuildingServiceTypeSafe{}, opsServices.NewGeocodingService(""))

	w := doTypeSafeRequest(h, http.MethodPost, "/buildings/geocode", `{"address":"北京市海淀区"}`)

	assert.Contains(t, w.Body.String(), "地址解析失败")
}

// --- 构造函数 ----------------------------------------------------------------

func TestBuildingHandlerTypeSafe_NewBuildingHandlerTypeSafeWithCore(t *testing.T) {
	c := newBuildingHandlerCore(t)
	if c.Config == nil {
		t.Skip("skipping NewBuildingHandlerTypeSafeWithCore: requires non-nil core.Config")
	}
	h := NewBuildingHandlerTypeSafeWithCore(&mockBuildingServiceTypeSafe{}, c)
	assert.NotNil(t, h)
	assert.NotNil(t, h.geocodingService)
}

// 编译期接口符合性守卫：mock 必须始终实现 BuildingServiceTypeSafe。
var _ opsServices.BuildingServiceTypeSafe = (*mockBuildingServiceTypeSafe)(nil)

// 防呆守卫：core 依赖仅用于构造 GeocodingService，不允许出现 nil core 传入
// NewBuildingHandlerTypeSafeWithCore 的用法（会 panic 在 core.Config 解引用）。
func TestBuildingHandlerTypeSafe_WithCore_NilConfigGuard(t *testing.T) {
	c := &core.Core{} // Config 为 nil
	assert.Panics(t, func() {
		_ = NewBuildingHandlerTypeSafeWithCore(&mockBuildingServiceTypeSafe{}, c)
	}, "nil Config 应当显式 panic 暴露误用，而不是静默产出坏 handler")
}
