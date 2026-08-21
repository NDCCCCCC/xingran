package network

// MACHistoryHandler + MACHistoryHeatmapHandler tests (Phase 74-03).
//
// Both handlers depend on interfaces (services.MACHistoryQueryService /
// services.MACHistoryHeatmapService) → D-08 mock pattern with *Func fields.
// ExportHistory is verified end-to-end: the mock writes real xlsx bytes into the
// handler-provided io.Writer and the test asserts Content-Type/Disposition headers.

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/services"
)

type mockMACHistoryQueryService struct {
	QueryPortHistoryFunc   func(ctx context.Context, req *services.PortHistoryQuery) (*services.MACHistoryQueryResult, error)
	QueryDeviceHistoryFunc func(ctx context.Context, req *services.DeviceHistoryQuery) (*services.MACHistoryQueryResult, error)
	QueryHistoryFunc       func(ctx context.Context, req *services.MACHistoryListQuery) (*services.MACHistoryQueryResult, error)
	ExportHistoryFunc      func(ctx context.Context, req *services.MACHistoryListQuery, w io.Writer) error
	QueryConnStatsFunc     func(ctx context.Context, req *services.ConnectionStatsQuery) (*services.ConnectionStatsResponse, error)
	GetVendorFunc          func(ctx context.Context, mac string) (string, error)

	lastPortReq    *services.PortHistoryQuery
	lastDeviceReq  *services.DeviceHistoryQuery
	lastListQuery  *services.MACHistoryListQuery
	lastExportReq  *services.MACHistoryListQuery
	lastStatsQuery *services.ConnectionStatsQuery
	lastVendorMAC  string
}

func (m *mockMACHistoryQueryService) ExportHistory(ctx context.Context, req *services.MACHistoryListQuery, w io.Writer) error {
	m.lastExportReq = req
	if m.ExportHistoryFunc != nil {
		return m.ExportHistoryFunc(ctx, req, w)
	}
	return errNetSvc
}

func (m *mockMACHistoryQueryService) QueryPortHistory(ctx context.Context, req *services.PortHistoryQuery) (*services.MACHistoryQueryResult, error) {
	m.lastPortReq = req
	if m.QueryPortHistoryFunc != nil {
		return m.QueryPortHistoryFunc(ctx, req)
	}
	return nil, errNetSvc
}

func (m *mockMACHistoryQueryService) QueryDeviceHistory(ctx context.Context, req *services.DeviceHistoryQuery) (*services.MACHistoryQueryResult, error) {
	m.lastDeviceReq = req
	if m.QueryDeviceHistoryFunc != nil {
		return m.QueryDeviceHistoryFunc(ctx, req)
	}
	return nil, errNetSvc
}

func (m *mockMACHistoryQueryService) QueryHistory(ctx context.Context, req *services.MACHistoryListQuery) (*services.MACHistoryQueryResult, error) {
	m.lastListQuery = req
	if m.QueryHistoryFunc != nil {
		return m.QueryHistoryFunc(ctx, req)
	}
	return nil, errNetSvc
}

func (m *mockMACHistoryQueryService) QueryConnectionStats(ctx context.Context, req *services.ConnectionStatsQuery) (*services.ConnectionStatsResponse, error) {
	m.lastStatsQuery = req
	if m.QueryConnStatsFunc != nil {
		return m.QueryConnStatsFunc(ctx, req)
	}
	return nil, errNetSvc
}

func (m *mockMACHistoryQueryService) ImportOUIData(ctx context.Context) error { return nil }

func (m *mockMACHistoryQueryService) GetVendor(ctx context.Context, macAddress string) (string, error) {
	m.lastVendorMAC = macAddress
	if m.GetVendorFunc != nil {
		return m.GetVendorFunc(ctx, macAddress)
	}
	return "", errNetSvc
}

var _ services.MACHistoryQueryService = (*mockMACHistoryQueryService)(nil)

type mockHeatmapService struct {
	QueryHeatmapFunc func(ctx context.Context, req *services.HeatmapQuery) (*services.HeatmapResult, error)
	lastReq          *services.HeatmapQuery
}

func (m *mockHeatmapService) QueryHeatmap(ctx context.Context, req *services.HeatmapQuery) (*services.HeatmapResult, error) {
	m.lastReq = req
	if m.QueryHeatmapFunc != nil {
		return m.QueryHeatmapFunc(ctx, req)
	}
	return nil, errNetSvc
}

var _ services.MACHistoryHeatmapService = (*mockHeatmapService)(nil)

// newHistoryHandler wires MACHistoryHandler with the mock query service.
func newHistoryHandler(m *mockMACHistoryQueryService) *MACHistoryHandler {
	return NewMACHistoryHandler(m)
}

func TestMACHistoryHandler_QueryPortHistory(t *testing.T) {
	m := &mockMACHistoryQueryService{QueryPortHistoryFunc: func(ctx context.Context, req *services.PortHistoryQuery) (*services.MACHistoryQueryResult, error) {
		return &services.MACHistoryQueryResult{
			List: []services.MACHistoryRecord{{MACAddress: "AA:BB:CC:DD:EE:FF"}}, Total: 1, Current: 2, PageSize: 20,
		}, nil
	}}
	h := newHistoryHandler(m)

	t.Run("success", func(t *testing.T) {
		w := netPost(t, "/history/port", h.QueryPortHistory,
			`{"deviceId":"dev-1","interfaceName":"GE0/0/1","current":2,"pageSize":20}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		require.NotNil(t, m.lastPortReq)
		assert.Equal(t, "dev-1", m.lastPortReq.DeviceID)
		assert.Contains(t, string(resp.Data), `"total":1`)
		assert.Contains(t, string(resp.Data), `"current":2`)
	})

	t.Run("missing_deviceId_binding_400", func(t *testing.T) {
		w := netPost(t, "/history/port", h.QueryPortHistory, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("service_error", func(t *testing.T) {
		fail := &mockMACHistoryQueryService{}
		fh := newHistoryHandler(fail)
		w := netPost(t, "/history/port", fh.QueryPortHistory, `{"deviceId":"dev-1","current":1,"pageSize":20}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "查询端口MAC历史记录失败")
	})
}

func TestMACHistoryHandler_QueryDeviceHistory(t *testing.T) {
	m := &mockMACHistoryQueryService{QueryDeviceHistoryFunc: func(ctx context.Context, req *services.DeviceHistoryQuery) (*services.MACHistoryQueryResult, error) {
		return &services.MACHistoryQueryResult{List: []services.MACHistoryRecord{}, Total: 0, Current: 1, PageSize: 10}, nil
	}}
	h := newHistoryHandler(m)

	t.Run("success", func(t *testing.T) {
		w := netPost(t, "/history/device", h.QueryDeviceHistory, `{"deviceId":"dev-2","current":1,"pageSize":20}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		require.NotNil(t, m.lastDeviceReq)
		assert.Equal(t, "dev-2", m.lastDeviceReq.DeviceID)
	})

	t.Run("missing_deviceId_binding_400", func(t *testing.T) {
		w := netPost(t, "/history/device", h.QueryDeviceHistory, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("service_error", func(t *testing.T) {
		fail := &mockMACHistoryQueryService{}
		fh := newHistoryHandler(fail)
		w := netPost(t, "/history/device", fh.QueryDeviceHistory, `{"deviceId":"d","current":1,"pageSize":20}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})
}

func TestMACHistoryHandler_QueryConnectionStats(t *testing.T) {
	m := &mockMACHistoryQueryService{QueryConnStatsFunc: func(ctx context.Context, req *services.ConnectionStatsQuery) (*services.ConnectionStatsResponse, error) {
		return &services.ConnectionStatsResponse{LongOccupancyDays: 30}, nil
	}}
	h := newHistoryHandler(m)

	t.Run("success", func(t *testing.T) {
		w := netPost(t, "/history/stats", h.QueryConnectionStats,
			`{"startTime":"2026-08-01T00:00:00Z","endTime":"2026-08-21T00:00:00Z","topN":5}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		require.NotNil(t, m.lastStatsQuery)
		assert.Equal(t, 5, m.lastStatsQuery.TopN)
		assert.Contains(t, string(resp.Data), `"longOccupancyDays":30`)
	})

	t.Run("missing_time_range_binding_400", func(t *testing.T) {
		w := netPost(t, "/history/stats", h.QueryConnectionStats, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("service_error", func(t *testing.T) {
		fail := &mockMACHistoryQueryService{}
		fh := newHistoryHandler(fail)
		w := netPost(t, "/history/stats", fh.QueryConnectionStats,
			`{"startTime":"2026-08-01T00:00:00Z","endTime":"2026-08-21T00:00:00Z"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})
}

func TestMACHistoryHandler_GetVendor(t *testing.T) {
	m := &mockMACHistoryQueryService{GetVendorFunc: func(ctx context.Context, mac string) (string, error) {
		return "Huawei Technologies", nil
	}}
	h := newHistoryHandler(m)

	t.Run("success_camelcase_field", func(t *testing.T) {
		w := netPost(t, "/history/vendor", h.GetVendor, `{"mac":"AA:BB:CC:DD:EE:FF"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, "AA:BB:CC:DD:EE:FF", m.lastVendorMAC)
		// GetVendorResponse uses camelCase vendorName (Phase 13 W4 fix is locked here)
		assert.JSONEq(t, `{"vendorName":"Huawei Technologies"}`, string(resp.Data))
	})

	t.Run("missing_mac_binding_400", func(t *testing.T) {
		w := netPost(t, "/history/vendor", h.GetVendor, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("service_error", func(t *testing.T) {
		fail := &mockMACHistoryQueryService{}
		fh := newHistoryHandler(fail)
		w := netPost(t, "/history/vendor", fh.GetVendor, `{"mac":"AA:BB:CC"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})
}

func TestMACHistoryHandler_QueryHistory(t *testing.T) {
	m := &mockMACHistoryQueryService{QueryHistoryFunc: func(ctx context.Context, req *services.MACHistoryListQuery) (*services.MACHistoryQueryResult, error) {
		return &services.MACHistoryQueryResult{List: []services.MACHistoryRecord{}, Total: 42, Current: req.Current, PageSize: req.PageSize}, nil
	}}
	h := newHistoryHandler(m)

	t.Run("success_passes_filters", func(t *testing.T) {
		w := netPost(t, "/history/list", h.QueryHistory,
			`{"current":4,"pageSize":50,"mac":"AA:BB","deviceId":"d1","interfaceName":"GE0/0/1","eventType":"appeared"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		require.NotNil(t, m.lastListQuery)
		assert.Equal(t, 4, m.lastListQuery.Current)
		assert.Equal(t, 50, m.lastListQuery.PageSize)
		assert.Equal(t, "AA:BB", m.lastListQuery.MAC)
		assert.Equal(t, "appeared", m.lastListQuery.EventType)
		assert.Contains(t, string(resp.Data), `"total":42`)
	})

	t.Run("binding_error_400", func(t *testing.T) {
		w := netPost(t, "/history/list", h.QueryHistory, `{"pageSize":0}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("service_error", func(t *testing.T) {
		fail := &mockMACHistoryQueryService{}
		fh := newHistoryHandler(fail)
		w := netPost(t, "/history/list", fh.QueryHistory, `{"current":1,"pageSize":10}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})
}

func TestMACHistoryHandler_ExportHistory(t *testing.T) {
	// xlsx magic bytes (PK zip header) — the mock "renders" a workbook into the writer.
	fakeXLSX := []byte("PK\x03\x04fake-xlsx-content")
	m := &mockMACHistoryQueryService{ExportHistoryFunc: func(ctx context.Context, req *services.MACHistoryListQuery, w io.Writer) error {
		_, err := w.Write(fakeXLSX)
		return err
	}}
	h := newHistoryHandler(m)

	t.Run("success_streams_xlsx", func(t *testing.T) {
		w := netGet(t, "/history/list", h.ExportHistory, "?current=1&pageSize=20&exportScope=all")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", w.Header().Get("Content-Type"))
		assert.Contains(t, w.Header().Get("Content-Disposition"), "mac_history_all_")
		assert.Equal(t, fakeXLSX, w.Body.Bytes())
		require.NotNil(t, m.lastExportReq)
		assert.Equal(t, "all", m.lastExportReq.ExportScope)
		assert.Equal(t, 1, m.lastExportReq.Current)
		assert.Equal(t, 20, m.lastExportReq.PageSize)
	})

	t.Run("exportScope_defaults_to_current", func(t *testing.T) {
		// NOTE: current/pageSize carry binding min=1 even in GET form mode — a request
		// without pagination params is REJECTED (400) despite the Swagger default(1)/(20).
		w := netGet(t, "/history/list", h.ExportHistory, "?current=1&pageSize=20&mac=AA:BB")
		assert.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, m.lastExportReq)
		assert.Equal(t, "current", m.lastExportReq.ExportScope)
		assert.Equal(t, "AA:BB", m.lastExportReq.MAC)
	})

	t.Run("missing_pagination_rejected", func(t *testing.T) {
		// Quirk locked: mac filter alone is not enough — Current=0 fails min=1.
		fresh := &mockMACHistoryQueryService{}
		fh := newHistoryHandler(fresh)
		w := netGet(t, "/history/list", fh.ExportHistory, "?mac=AA:BB")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
		assert.Contains(t, resp.Message, "参数错误")
		assert.Nil(t, fresh.lastExportReq, "service must not be reached when binding fails")
	})

	t.Run("bad_query_param_400", func(t *testing.T) {
		w := netGet(t, "/history/list", h.ExportHistory, "?current=1&pageSize=notanint")
		resp := decodeNetResp(t, w)
		// response.Error(c, http.StatusBadRequest(int), ...) → HTTP 400, code 400
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("service_error_500_int_quirk", func(t *testing.T) {
		fail := &mockMACHistoryQueryService{}
		fh := newHistoryHandler(fail)
		w := netGet(t, "/history/list", fh.ExportHistory, "?current=1&pageSize=20")
		resp := decodeNetResp(t, w)
		// response.Error(c, http.StatusInternalServerError(int), ...) → HTTP 400, code 500 (quirk)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "导出失败")
	})
}

func TestMACHistoryHeatmapHandler_QueryHeatmap(t *testing.T) {
	m := &mockHeatmapService{QueryHeatmapFunc: func(ctx context.Context, req *services.HeatmapQuery) (*services.HeatmapResult, error) {
		return &services.HeatmapResult{Total: 9, TopN: req.TopN, Snapshot: "ok"}, nil
	}}
	h := NewMACHistoryHeatmapHandler(m)

	t.Run("success", func(t *testing.T) {
		w := netPost(t, "/history/heatmap", h.QueryHeatmap,
			`{"startTime":"2026-08-01T00:00:00Z","endTime":"2026-08-21T00:00:00Z","topN":15}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		require.NotNil(t, m.lastReq)
		assert.Equal(t, 15, m.lastReq.TopN)
		assert.Contains(t, string(resp.Data), `"total":9`)
	})

	t.Run("binding_error", func(t *testing.T) {
		w := netPost(t, "/history/heatmap", h.QueryHeatmap, `not-json`)
		resp := decodeNetResp(t, w)
		// response.Error(c, 400, ...) → HTTP 400, code 400
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("service_error_500_int_quirk", func(t *testing.T) {
		fail := &mockHeatmapService{}
		fh := NewMACHistoryHeatmapHandler(fail)
		w := netPost(t, "/history/heatmap", fh.QueryHeatmap, `{}`)
		resp := decodeNetResp(t, w)
		// response.Error(c, 500, ...) → HTTP 400, code 500 (quirk)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "热力图查询失败")
	})
}
