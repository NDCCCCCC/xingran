package network

// TopologyHandler tests (Phase 74-03).
//
// TopologyHandler depends on topology.FilterRuleService (an interface) → D-08 mock
// with *Func fields. GetEffectiveRule additionally reads sys_network_device through
// the handler's own *gorm.DB, so that test uses the shared sqlite env.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/topology"
)

type mockFilterRuleService struct {
	CreateFunc         func(ctx context.Context, req *topology.CreateFilterRuleRequest) (*models.MACFilterRule, error)
	UpdateFunc         func(ctx context.Context, id string, req *topology.UpdateFilterRuleRequest) error
	DeleteFunc         func(ctx context.Context, id string) error
	GetByIDFunc        func(ctx context.Context, id string) (*models.MACFilterRule, error)
	ListFunc           func(ctx context.Context, params topology.ListFilterRulesParams) (*topology.PageResult, error)
	GetEffectiveFunc   func(ctx context.Context, device *models.NetworkDevice) (*models.MACFilterRule, error)

	lastCreateReq *topology.CreateFilterRuleRequest
	lastUpdateID  string
	lastUpdateReq *topology.UpdateFilterRuleRequest
	lastDeleteID  string
	lastGetID     string
	lastList      topology.ListFilterRulesParams
	lastEffective *models.NetworkDevice
}

var _ topology.FilterRuleService = (*mockFilterRuleService)(nil)

func (m *mockFilterRuleService) Create(ctx context.Context, req *topology.CreateFilterRuleRequest) (*models.MACFilterRule, error) {
	m.lastCreateReq = req
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return nil, errNetSvc
}

func (m *mockFilterRuleService) Update(ctx context.Context, id string, req *topology.UpdateFilterRuleRequest) error {
	m.lastUpdateID = id
	m.lastUpdateReq = req
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, req)
	}
	return errNetSvc
}

func (m *mockFilterRuleService) Delete(ctx context.Context, id string) error {
	m.lastDeleteID = id
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errNetSvc
}

func (m *mockFilterRuleService) GetByID(ctx context.Context, id string) (*models.MACFilterRule, error) {
	m.lastGetID = id
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errNetSvc
}

func (m *mockFilterRuleService) List(ctx context.Context, params topology.ListFilterRulesParams) (*topology.PageResult, error) {
	m.lastList = params
	if m.ListFunc != nil {
		return m.ListFunc(ctx, params)
	}
	return nil, errNetSvc
}

func (m *mockFilterRuleService) GetEffectiveRule(ctx context.Context, device *models.NetworkDevice) (*models.MACFilterRule, error) {
	m.lastEffective = device
	if m.GetEffectiveFunc != nil {
		return m.GetEffectiveFunc(ctx, device)
	}
	return nil, errNetSvc
}

func newTopologyHandlerUnderTest(svc *mockFilterRuleService, env *netTestEnv) *TopologyHandler {
	return NewTopologyHandler(svc, env.db).WithCore(env.core)
}

func TestTopologyHandler_CreateRule(t *testing.T) {
	env := newNetworkTestEnv(t)
	svc := &mockFilterRuleService{CreateFunc: func(ctx context.Context, req *topology.CreateFilterRuleRequest) (*models.MACFilterRule, error) {
		return &models.MACFilterRule{RuleName: req.RuleName}, nil
	}}
	h := newTopologyHandlerUnderTest(svc, env)

	t.Run("success", func(t *testing.T) {
		w := netPost(t, "/rules", h.CreateRule,
			`{"ruleName":"核心交换机","deviceType":"switch","vendor":"huawei","macThreshold":50,"enableLLDPFilter":true,"priority":10}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		// CreatedBy comes from the username auth key, falling back to "system"
		require.NotNil(t, svc.lastCreateReq)
		assert.Equal(t, "tester", svc.lastCreateReq.CreatedBy)
		assert.Equal(t, 1, env.oper.recordAsyncCalls)
		assert.Equal(t, "网络拓扑", env.oper.lastTitle)
		assert.Equal(t, 1, env.oper.lastBusinessType) // OperTypeCreate
	})

	t.Run("binding_error_400", func(t *testing.T) {
		// ruleName + deviceType + macThreshold + priority are required
		w := netPost(t, "/rules", h.CreateRule, `{"ruleName":""}`)
		resp := decodeNetResp(t, w)
		// response.Error(c, http.StatusBadRequest, msg) → int → HTTP 400, code 400
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("service_error", func(t *testing.T) {
		fail := &mockFilterRuleService{}
		fh := newTopologyHandlerUnderTest(fail, env)
		w := netPost(t, "/rules", fh.CreateRule,
			`{"ruleName":"r","deviceType":"switch","macThreshold":1,"priority":1}`)
		resp := decodeNetResp(t, w)
		// response.Error(c, http.StatusInternalServerError(int), err.Error()) → HTTP 400, code 500 (project quirk)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})
}

func TestTopologyHandler_UpdateRule(t *testing.T) {
	env := newNetworkTestEnv(t)
	svc := &mockFilterRuleService{UpdateFunc: func(ctx context.Context, id string, req *topology.UpdateFilterRuleRequest) error {
		return nil
	}}
	h := newTopologyHandlerUnderTest(svc, env)

	t.Run("success", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/rules/:id/update", h.UpdateRule}},
			http.MethodPost, "/rules/rule-1/update", `{"ruleName":"renamed"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		require.NotNil(t, svc.lastUpdateReq)
		assert.Equal(t, "rule-1", svc.lastUpdateReq.ID)
		assert.Equal(t, 2, env.oper.lastBusinessType) // OperTypeUpdate
	})

	t.Run("service_error", func(t *testing.T) {
		fail := &mockFilterRuleService{}
		fh := newTopologyHandlerUnderTest(fail, env)
		w := netServe(t, []netRoute{{http.MethodPost, "/rules/:id/update", fh.UpdateRule}},
			http.MethodPost, "/rules/rule-1/update", `{"ruleName":"x"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})

	t.Run("binding_error", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/rules/:id/update", h.UpdateRule}},
			http.MethodPost, "/rules/rule-1/update", `not-json`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestTopologyHandler_DeleteRule(t *testing.T) {
	env := newNetworkTestEnv(t)
	svc := &mockFilterRuleService{DeleteFunc: func(ctx context.Context, id string) error { return nil }}
	h := newTopologyHandlerUnderTest(svc, env)

	t.Run("success", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/rules/:id/delete", h.DeleteRule}},
			http.MethodPost, "/rules/rule-2/delete", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, "rule-2", svc.lastDeleteID)
		assert.Equal(t, 3, env.oper.lastBusinessType) // OperTypeDelete
	})

	t.Run("service_error", func(t *testing.T) {
		fail := &mockFilterRuleService{}
		fh := newTopologyHandlerUnderTest(fail, env)
		w := netServe(t, []netRoute{{http.MethodPost, "/rules/:id/delete", fh.DeleteRule}},
			http.MethodPost, "/rules/rule-2/delete", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})
}

func TestTopologyHandler_GetRule(t *testing.T) {
	env := newNetworkTestEnv(t)
	svc := &mockFilterRuleService{GetByIDFunc: func(ctx context.Context, id string) (*models.MACFilterRule, error) {
		return &models.MACFilterRule{RuleName: "found"}, nil
	}}
	h := newTopologyHandlerUnderTest(svc, env)

	t.Run("found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/rules/:id", h.GetRule}},
			http.MethodPost, "/rules/rule-3", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, "rule-3", svc.lastGetID)
	})

	t.Run("service_error_maps_to_404_int_quirk", func(t *testing.T) {
		fail := &mockFilterRuleService{}
		fh := newTopologyHandlerUnderTest(fail, env)
		w := netServe(t, []netRoute{{http.MethodPost, "/rules/:id", fh.GetRule}},
			http.MethodPost, "/rules/none", "")
		resp := decodeNetResp(t, w)
		// response.Error(c, http.StatusNotFound(404 int), ...) → HTTP 400, body code 404
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 404, resp.Code)
	})
}

func TestTopologyHandler_ListRules(t *testing.T) {
	env := newNetworkTestEnv(t)
	svc := &mockFilterRuleService{ListFunc: func(ctx context.Context, params topology.ListFilterRulesParams) (*topology.PageResult, error) {
		return &topology.PageResult{List: []*models.MACFilterRule{}, Total: 7}, nil
	}}
	h := newTopologyHandlerUnderTest(svc, env)

	t.Run("defaults_on_bad_body", func(t *testing.T) {
		w := netPost(t, "/rules/list", h.ListRules, `not-json`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		// bind failure resets to Current=1/PageSize=10 defaults
		assert.Equal(t, 1, svc.lastList.Current)
		assert.Equal(t, 10, svc.lastList.PageSize)
		assert.Contains(t, string(resp.Data), `"total":7`)
	})

	t.Run("clamps_invalid_pagination", func(t *testing.T) {
		w := netPost(t, "/rules/list", h.ListRules, `{"current":0,"pageSize":9999}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, 1, svc.lastList.Current)
		assert.Equal(t, 10, svc.lastList.PageSize) // >100 clamped back to 10
	})

	t.Run("service_error", func(t *testing.T) {
		fail := &mockFilterRuleService{}
		fh := newTopologyHandlerUnderTest(fail, env)
		w := netPost(t, "/rules/list", fh.ListRules, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})
}

func TestTopologyHandler_GetEffectiveRule(t *testing.T) {
	env := newNetworkTestEnv(t, &models.NetworkDevice{})
	svc := &mockFilterRuleService{GetEffectiveFunc: func(ctx context.Context, device *models.NetworkDevice) (*models.MACFilterRule, error) {
		return &models.MACFilterRule{RuleName: "effective"}, nil
	}}
	h := newTopologyHandlerUnderTest(svc, env)
	netSeedDevice(t, env.db, "dev-e1", "effective-dev", "10.9.9.9")

	t.Run("missing_deviceId_400", func(t *testing.T) {
		w := netGet(t, "/rules/effective", h.GetEffectiveRule, "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("device_not_found", func(t *testing.T) {
		w := netGet(t, "/rules/effective", h.GetEffectiveRule, "?deviceId=no-such")
		resp := decodeNetResp(t, w)
		// int http.StatusNotFound → HTTP 400, code 404 (quirk)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 404, resp.Code)
	})

	t.Run("success_passes_loaded_device", func(t *testing.T) {
		w := netGet(t, "/rules/effective", h.GetEffectiveRule, "?deviceId=dev-e1")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		require.NotNil(t, svc.lastEffective)
		assert.Equal(t, "dev-e1", svc.lastEffective.ID)
	})

	t.Run("service_error", func(t *testing.T) {
		fail := &mockFilterRuleService{}
		fh := newTopologyHandlerUnderTest(fail, env)
		w := netGet(t, "/rules/effective", fh.GetEffectiveRule, "?deviceId=dev-e1")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})
}

func TestGetCreatedByFromContext_Fallback(t *testing.T) {
	// When no username is present in the context the helper falls back to "system".
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	assert.Equal(t, "system", getCreatedByFromContext(c))
	// And a non-string username value also falls back.
	c.Set("username", 12345)
	assert.Equal(t, "system", getCreatedByFromContext(c))
	// A real string username wins.
	c.Set("username", "alice")
	assert.Equal(t, "alice", getCreatedByFromContext(c))
}
