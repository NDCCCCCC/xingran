package network

// MACHandler + PortHandler tests (Phase 74-03).
//
// Both handlers build their services inline from core (core.DB + core.DeviceExecutor).
// The executor is nil here — every test sticks to DB-only flows (list/stats/clean/
// batch-delete) or pre-executor validation errors (ghost device IDs, no online
// devices). Live collection requires real device connections (out of scope, D-12).

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

func newMACPortTestEnv(t *testing.T) *netTestEnv {
	return newNetworkTestEnv(t, &models.DeviceMACAddress{}, &models.DevicePortStatus{}, &models.NetworkDevice{})
}

func newMACHandler(env *netTestEnv) *MACHandler {
	return NewMACHandler(env.core)
}

func newPortHandler(env *netTestEnv) *PortHandler {
	return NewPortHandler(env.core)
}

func seedMAC(t *testing.T, env *netTestEnv, id, deviceID, mac, iface string, collectedAt time.Time) *models.DeviceMACAddress {
	t.Helper()
	m := &models.DeviceMACAddress{
		ID:            id,
		DeviceID:      deviceID,
		MACAddress:    mac,
		InterfaceName: iface,
		CollectedAt:   collectedAt,
	}
	require.NoError(t, env.db.Create(m).Error)
	return m
}

func seedPortStatus(t *testing.T, env *netTestEnv, id, deviceID, iface, admin, oper string, collectedAt time.Time) *models.DevicePortStatus {
	t.Helper()
	p := &models.DevicePortStatus{
		ID:            id,
		DeviceID:      deviceID,
		InterfaceName: iface,
		AdminStatus:   admin,
		OperStatus:    oper,
		CollectedAt:   collectedAt,
	}
	require.NoError(t, env.db.Create(p).Error)
	return p
}

// --- MACHandler ---

func TestMACHandler_List(t *testing.T) {
	env := newMACPortTestEnv(t)
	h := newMACHandler(env)
	netSeedDevice(t, env.db, "dev-m1", "mac-dev", "10.1.1.1")
	now := time.Now()
	seedMAC(t, env, "m1", "dev-m1", "AA:BB:CC:DD:EE:01", "GE0/0/1", now)
	seedMAC(t, env, "m2", "dev-m1", "AA:BB:CC:DD:EE:02", "GE0/0/2", now)

	t.Run("list_with_filters", func(t *testing.T) {
		w := netPost(t, "/mac/list", h.List,
			`{"current":1,"pageSize":10,"deviceId":"dev-m1","macAddress":"AA:BB:CC:DD:EE:01"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), `"total":1`)
		assert.Contains(t, string(resp.Data), `"deviceName":"mac-dev"`)
	})

	t.Run("interface_name_filter", func(t *testing.T) {
		w := netPost(t, "/mac/list", h.List,
			`{"current":1,"pageSize":10,"interfaceName":"GE0/0/2"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"total":1`)
	})

	t.Run("malformed_json_400", func(t *testing.T) {
		w := netPost(t, "/mac/list", h.List, `{bad`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestMACHandler_Collect(t *testing.T) {
	env := newMACPortTestEnv(t)
	h := newMACHandler(env)

	t.Run("binding_requires_deviceId", func(t *testing.T) {
		w := netPost(t, "/mac/collect", h.Collect, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("ghost_device_rejected_before_executor", func(t *testing.T) {
		w := netPost(t, "/mac/collect", h.Collect, `{"deviceId":"ghost"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "设备不存在")
	})
}

func TestMACHandler_CollectAll_NoOnlineDevices(t *testing.T) {
	// Empty device table → "没有在线设备" before any executor access
	env := newMACPortTestEnv(t)
	h := newMACHandler(env)

	w := netPost(t, "/mac/collect-all", h.CollectAll, "")
	resp := decodeNetResp(t, w)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 500, resp.Code)
	assert.Contains(t, resp.Message, "没有在线设备")
}

func TestMACHandler_GetStats(t *testing.T) {
	env := newMACPortTestEnv(t)
	h := newMACHandler(env)
	now := time.Now()
	seedMAC(t, env, "m1", "dev-1", "AA:BB:CC:DD:EE:01", "GE0/0/1", now)
	seedMAC(t, env, "m2", "dev-2", "AA:BB:CC:DD:EE:01", "GE0/0/1", now)
	seedMAC(t, env, "m3", "dev-1", "AA:BB:CC:DD:EE:02", "GE0/0/2", now)

	w := netPost(t, "/mac/stats", h.GetStats, "")
	resp := decodeNetResp(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, resp.Code)
	assert.Contains(t, string(resp.Data), `"totalRecords":3`)
	assert.Contains(t, string(resp.Data), `"uniqueDevices":2`)
	assert.Contains(t, string(resp.Data), `"uniqueMACs":2`)
}

func TestMACHandler_Clean(t *testing.T) {
	env := newMACPortTestEnv(t)
	h := newMACHandler(env)
	old := time.Now().AddDate(0, 0, -90)
	fresh := time.Now()
	seedMAC(t, env, "old1", "dev-1", "AA:BB:CC:DD:EE:01", "GE0/0/1", old)
	seedMAC(t, env, "new1", "dev-1", "AA:BB:CC:DD:EE:02", "GE0/0/2", fresh)

	t.Run("success_deletes_only_old_records", func(t *testing.T) {
		w := netPost(t, "/mac/clean", h.Clean, `{"days":30}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), `"deletedCount":1`)
		assert.Equal(t, 9, env.oper.lastBusinessType) // OperTypeClean

		var remaining int64
		env.db.Model(&models.DeviceMACAddress{}).Count(&remaining)
		assert.Equal(t, int64(1), remaining, "fresh record survives")
	})

	t.Run("binding_days_range", func(t *testing.T) {
		w := netPost(t, "/mac/clean", h.Clean, `{"days":0}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestMACHandler_BatchDelete(t *testing.T) {
	env := newMACPortTestEnv(t)
	h := newMACHandler(env)
	now := time.Now()
	seedMAC(t, env, "bd1", "dev-1", "AA:BB:CC:DD:EE:01", "GE0/0/1", now)
	seedMAC(t, env, "bd2", "dev-1", "AA:BB:CC:DD:EE:02", "GE0/0/2", now)

	t.Run("success_operType_batch", func(t *testing.T) {
		w := netPost(t, "/mac/batch-delete", h.BatchDelete, `{"ids":["bd1","bd2"]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"deletedCount":2`)
		assert.Equal(t, 16, env.oper.lastBusinessType)
	})

	t.Run("empty_ids_binding_400", func(t *testing.T) {
		w := netPost(t, "/mac/batch-delete", h.BatchDelete, `{"ids":[]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

// --- PortHandler ---

func TestPortHandler_List(t *testing.T) {
	env := newMACPortTestEnv(t)
	h := newPortHandler(env)
	now := time.Now()
	seedPortStatus(t, env, "p1", "dev-p1", "GE0/0/1", "up", "up", now)
	seedPortStatus(t, env, "p2", "dev-p1", "GE0/0/2", "down", "down", now)

	t.Run("list_all", func(t *testing.T) {
		w := netPost(t, "/port/list", h.List, `{"current":1,"pageSize":10}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), `"total":2`)
	})

	t.Run("filter_by_interface_and_status", func(t *testing.T) {
		w := netPost(t, "/port/list", h.List,
			`{"current":1,"pageSize":10,"interfaceName":"GE0/0/1","adminStatus":"up"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"total":1`)
	})

	t.Run("malformed_json_400", func(t *testing.T) {
		w := netPost(t, "/port/list", h.List, `{bad`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestPortHandler_Collect(t *testing.T) {
	env := newMACPortTestEnv(t)
	h := newPortHandler(env)

	t.Run("binding_requires_deviceId", func(t *testing.T) {
		w := netPost(t, "/port/collect", h.Collect, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("ghost_device_rejected", func(t *testing.T) {
		w := netPost(t, "/port/collect", h.Collect, `{"deviceId":"ghost"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})
}

func TestPortHandler_CollectAll_NoOnlineDevices(t *testing.T) {
	env := newMACPortTestEnv(t)
	h := newPortHandler(env)

	w := netPost(t, "/port/collect-all", h.CollectAll, "")
	resp := decodeNetResp(t, w)
	// No online devices → service error before executor access
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 500, resp.Code)
}

func TestPortHandler_GetStats(t *testing.T) {
	env := newMACPortTestEnv(t)
	h := newPortHandler(env)
	now := time.Now()
	seedPortStatus(t, env, "p1", "dev-1", "GE0/0/1", "up", "up", now)
	seedPortStatus(t, env, "p2", "dev-1", "GE0/0/2", "down", "down", now)

	w := netPost(t, "/port/stats", h.GetStats, "")
	resp := decodeNetResp(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, resp.Code)
	assert.Contains(t, string(resp.Data), `"totalRecords":2`)
}

func TestPortHandler_BatchDelete(t *testing.T) {
	env := newMACPortTestEnv(t)
	h := newPortHandler(env)
	now := time.Now()
	seedPortStatus(t, env, "pd1", "dev-1", "GE0/0/1", "up", "up", now)
	seedPortStatus(t, env, "pd2", "dev-1", "GE0/0/2", "up", "up", now)

	t.Run("success_direct_db_delete", func(t *testing.T) {
		w := netPost(t, "/port/batch-delete", h.BatchDelete, `{"ids":["pd1","pd2"]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		// Quirk locked: deletedCount mirrors len(ids), not rows affected
		assert.Contains(t, string(resp.Data), `"deletedCount":2`)
		assert.Equal(t, 16, env.oper.lastBusinessType)

		var remaining int64
		env.db.Model(&models.DevicePortStatus{}).Count(&remaining)
		assert.Zero(t, remaining)
	})

	t.Run("empty_ids_binding_400", func(t *testing.T) {
		w := netPost(t, "/port/batch-delete", h.BatchDelete, `{"ids":[]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestPortHandler_Clean(t *testing.T) {
	env := newMACPortTestEnv(t)
	h := newPortHandler(env)
	// CollectedAt 2 days ago → within the days=1 cutoff window
	seedPortStatus(t, env, "pc1", "dev-1", "GE0/0/1", "up", "up", time.Now().AddDate(0, 0, -2))

	t.Run("success_operType_clean", func(t *testing.T) {
		w := netPost(t, "/port/clean", h.Clean, `{"days":1}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"deletedCount":1`)
		assert.Equal(t, 9, env.oper.lastBusinessType) // OperTypeClean
	})

	t.Run("binding_days_range", func(t *testing.T) {
		w := netPost(t, "/port/clean", h.Clean, `{"days":400}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}
