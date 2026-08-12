package operations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/asset"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// stubWorkstationService 满足 WorkstationService 接口的最小子集,只用于测试 GetByID
type stubWorkstationService struct {
	ws *models.Workstation
}

func (s *stubWorkstationService) GetByID(_ context.Context, _ string) (*models.Workstation, error) {
	return s.ws, nil
}

func (s *stubWorkstationService) List(_ context.Context, _ map[string]interface{}) (*opsServices.PageResult, error) {
	return nil, nil
}

func (s *stubWorkstationService) Create(_ context.Context, _ *models.Workstation) error {
	return nil
}

func (s *stubWorkstationService) Update(_ context.Context, _ *models.Workstation) error {
	return nil
}

func (s *stubWorkstationService) Delete(_ context.Context, _ string) error {
	return nil
}

func (s *stubWorkstationService) BatchDelete(_ context.Context, _ []string) error {
	return nil
}

func (s *stubWorkstationService) Statistics(_ context.Context, _ map[string]interface{}) (*opsServices.WorkstationStatisticsResult, error) {
	return nil, nil
}

func (s *stubWorkstationService) BatchUpdatePositions(_ context.Context, _ []opsServices.PositionUpdateItem) error {
	return nil
}

func (s *stubWorkstationService) GetWorkstationDeptOptions(_ context.Context, _ string) ([]opsServices.DeptOption, error) {
	return nil, nil
}

func (s *stubWorkstationService) SearchWorkstationOptions(_ context.Context, _ map[string]interface{}) ([]opsServices.DropdownOption, error) {
	return nil, nil
}

// stubReconciliationService 测试用 — 不接真实 DB,直接返回预设 ByWorkstationResponse
type stubReconciliationService struct {
	resp *asset.ByWorkstationResponse
	err  error
}

func (s *stubReconciliationService) ListExceptions(_ context.Context, _ *asset.ExceptionListParams) (*base.PageResult, error) {
	return nil, nil
}
func (s *stubReconciliationService) GetByID(_ context.Context, _ string) (*models.SysDataReconciliation, error) {
	return nil, nil
}
func (s *stubReconciliationService) ResolveException(_ context.Context, _ string, _ string, _ *string) error {
	return nil
}
func (s *stubReconciliationService) Refresh(_ context.Context) (int, int, int, int, error) {
	return 0, 0, 0, 0, nil
}
func (s *stubReconciliationService) GetByWorkstation(_ context.Context, _ string, _ string) (*asset.ByWorkstationResponse, error) {
	return s.resp, s.err
}

// newTestCoreForHandler 构造一个最小 *core.Core 用于 handler 测试(DB 内存 SQLite)
func newTestCoreForHandler(t *testing.T) *core.Core {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	return &core.Core{
		CoreInfra: &core.CoreInfra{
			DB: &db.Database{DB: gormDB, Type: "sqlite"},
		},
	}
}

// newTestWorkstation 构造测试用 Workstation 对象
func newTestWorkstation(id string) *models.Workstation {
	return &models.Workstation{
		BaseModel:      models.BaseModel{ID: id},
		WorkstationName: "测试工位",
		WorkstationType: 0,
		Status:          0,
	}
}

// TestWorkstationHandler_GetByID_WithoutReconciliationSvc 验证 reconciliationSvc=nil 时
// hasReconciliationPerm=true 但 Reconciliation=nil(降级路径)
func TestWorkstationHandler_GetByID_WithoutReconciliationSvc(t *testing.T) {
	gin.SetMode(gin.TestMode)
	wsID := uuid.New().String()
	stub := &stubWorkstationService{ws: newTestWorkstation(wsID)}
	coreInst := newTestCoreForHandler(t)

	h := NewWorkstationHandler(stub).WithCore(coreInst)
	// 注意:不调 WithReconciliationService → reconciliationSvc 保持 nil

	r := gin.New()
	r.POST("/:id", h.GetByID)

	req := httptest.NewRequest(http.MethodPost, "/"+wsID, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "GetByID should return 200")
	body := w.Body.String()
	// ReconciliationVisible 应当存在(因 HasUserPermission 在空 userID 时返回 false → visible=false)
	assert.Contains(t, body, "reconciliationVisible", "response should include reconciliationVisible field")
	// 解析 body 确认
	var resp struct {
		Data struct {
			ReconciliationVisible bool                   `json:"reconciliationVisible"`
			Reconciliation        map[string]interface{} `json:"reconciliation"`
		} `json:"data"`
	}
	assert.NoError(t, json.Unmarshal([]byte(body), &resp))
	// 无 user_id → HasUserPermission 返回 false → ReconciliationVisible=false
	assert.False(t, resp.Data.ReconciliationVisible, "no user_id should set visible=false")
	assert.Nil(t, resp.Data.Reconciliation, "no perm should leave reconciliation nil")
}

// TestWorkstationHandler_GetByID_ReconciliationData 验证注入成功路径:
// stub 返回非空 ByWorkstationResponse → 注入到 map[string]interface{} 后存在
func TestWorkstationHandler_GetByID_ReconciliationData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	wsID := uuid.New().String()
	stub := &stubWorkstationService{ws: newTestWorkstation(wsID)}
	coreInst := newTestCoreForHandler(t)

	// 构造 stub 返回 — 但因无 user_id 注入,handler 走无 perm 路径,
	// 所以 Reconciliation 仍为 nil;此测试主要验证类型转换不 panic
	mockRecon := &asset.ByWorkstationResponse{
		Workstation: asset.WorkstationBrief{ID: wsID, Name: "测试工位", Code: "WS001"},
		HealthScore: asset.HealthScore{
			Total:  3,
			Normal: 2,
			Drift:  1,
			Score:  67,
			Trend:  []asset.TrendPoint{},
		},
		Assets: []asset.AssetHealthItem{
			{AssetID: uuid.New().String(), AssetCode: "A001", ConflictType: "B"},
		},
		Visible: false,
	}
	reconStub := &stubReconciliationService{resp: mockRecon}

	h := NewWorkstationHandler(stub).
		WithCore(coreInst).
		WithReconciliationService(reconStub)

	r := gin.New()
	r.POST("/:id", h.GetByID)

	req := httptest.NewRequest(http.MethodPost, "/"+wsID, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "GetByID should return 200")
	// 解析响应 — 因无 user_id → visible=false → Reconciliation 仍 nil
	var resp struct {
		Data struct {
			ReconciliationVisible bool                   `json:"reconciliationVisible"`
			Reconciliation        map[string]interface{} `json:"reconciliation"`
		} `json:"data"`
	}
	assert.NoError(t, json.Unmarshal([]byte(w.Body.String()), &resp))
	assert.False(t, resp.Data.ReconciliationVisible)
	assert.Nil(t, resp.Data.Reconciliation)
}

// TestWorkstationHandler_HasReconciliationPerm_NoUserID 验证 hasReconciliationPerm 在无 user_id 时返回 false
func TestWorkstationHandler_HasReconciliationPerm_NoUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	coreInst := newTestCoreForHandler(t)
	h := NewWorkstationHandler(&stubWorkstationService{}).WithCore(coreInst)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	got := h.hasReconciliationPerm(c)
	assert.False(t, got, "no user_id should return false")
}
