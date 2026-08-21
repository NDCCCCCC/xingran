package network

// DiscoveryHandler tests (Phase 74-03).
//
// DiscoveryHandler takes the CONCRETE *services.DeviceDiscoveryService (DB-backed).
// Network-touching flows (SNMP probe with live communities, ExecuteDiscovery scans)
// are avoided: Probe covers its validation-only branches, Execute covers the
// task-not-found branch — the live scan paths need real devices (out of scope, D-12).

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
)

func newDiscoveryTestEnv(t *testing.T) *netTestEnv {
	return newNetworkTestEnv(t, &models.DeviceDiscovery{}, &models.AuthCredential{}, &models.NetworkDevice{})
}

func newDiscoveryHandler(env *netTestEnv) *DiscoveryHandler {
	return NewDiscoveryHandler(services.NewDeviceDiscoveryService(env.db)).WithCore(env.core)
}

func seedDiscovery(t *testing.T, env *netTestEnv, id, name string, status models.DiscoveryStatus, discovered int) *models.DeviceDiscovery {
	t.Helper()
	task := &models.DeviceDiscovery{
		ID:              id,
		TaskName:        name,
		DiscoveryType:   models.DiscoveryTypeSNMP,
		IPRanges:        models.IPRangeList{{StartIP: "192.168.1.1", EndIP: "192.168.1.10"}},
		SNMPPort:        161,
		Status:          status,
		TotalIPs:        10,
		DiscoveredCount: discovered,
	}
	require.NoError(t, env.db.Create(task).Error)
	return task
}

func TestDiscoveryHandler_Statistics(t *testing.T) {
	env := newDiscoveryTestEnv(t)
	h := newDiscoveryHandler(env)

	t.Run("counts_by_status", func(t *testing.T) {
		seedDiscovery(t, env, "d1", "pending-task", models.DiscoveryStatusPending, 0)
		seedDiscovery(t, env, "d2", "done-task", models.DiscoveryStatusSuccess, 7)
		seedDiscovery(t, env, "d3", "failed-task", models.DiscoveryStatusFailed, 3)

		w := netPost(t, "/discoveries/statistics", h.Statistics, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.JSONEq(t, `{"total":3,"pending":1,"running":0,"completed":1,"failed":1,"totalDevices":10}`, string(resp.Data))
	})

	t.Run("empty_zeros", func(t *testing.T) {
		fresh := newDiscoveryTestEnv(t)
		w := netPost(t, "/discoveries/statistics", newDiscoveryHandler(fresh).Statistics, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"total":0`)
	})
}

func TestDiscoveryHandler_List(t *testing.T) {
	env := newDiscoveryTestEnv(t)
	h := newDiscoveryHandler(env)
	seedDiscovery(t, env, "d1", "task-one", models.DiscoveryStatusPending, 0)
	seedDiscovery(t, env, "d2", "task-two", models.DiscoveryStatusSuccess, 5)

	t.Run("lists_tasks", func(t *testing.T) {
		w := netPost(t, "/discoveries/list", h.List, `{"current":1,"pageSize":10}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"total":2`)
		assert.Contains(t, string(resp.Data), "task-one")
	})

	t.Run("bad_body_defaults", func(t *testing.T) {
		w := netPost(t, "/discoveries/list", h.List, `not-json`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"current":1`)
		assert.Contains(t, string(resp.Data), `"pageSize":10`)
	})
}

func TestDiscoveryHandler_Create(t *testing.T) {
	env := newDiscoveryTestEnv(t)
	h := newDiscoveryHandler(env)

	t.Run("success_computes_total_ips_and_defaults", func(t *testing.T) {
		w := netPost(t, "/discoveries/create", h.Create,
			`{"TaskName":"weekly-scan","DiscoveryType":"snmp","IPRanges":[{"startIp":"192.168.1.1","endIp":"192.168.1.10"}],"SNMPCommunities":["public"]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), `"discoveryId":`)

		var task models.DeviceDiscovery
		require.NoError(t, env.db.Where("task_name = ?", "weekly-scan").First(&task).Error)
		assert.Equal(t, 10, task.TotalIPs, "IP count derived from range")
		assert.Equal(t, 161, task.SNMPPort, "SNMP port defaulted")
		assert.Equal(t, models.DiscoveryStatusPending, task.Status)
		assert.Equal(t, "user-0001", task.CreatedBy)
		assert.Equal(t, 1, env.oper.lastBusinessType) // OperTypeCreate
		assert.Equal(t, "网络设备发现", env.oper.lastTitle)
	})

	t.Run("malformed_json_400", func(t *testing.T) {
		w := netPost(t, "/discoveries/create", h.Create, `{bad`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestDiscoveryHandler_GetByID(t *testing.T) {
	env := newDiscoveryTestEnv(t)
	h := newDiscoveryHandler(env)
	seedDiscovery(t, env, "d9", "find-me", models.DiscoveryStatusPending, 0)

	t.Run("found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/discoveries/:id", h.GetByID}},
			http.MethodPost, "/discoveries/d9", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), "find-me")
	})

	t.Run("not_found_quirk_1010", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/discoveries/:id", h.GetByID}},
			http.MethodPost, "/discoveries/none", "")
		resp := decodeNetResp(t, w)
		// apperrors.NotFound → code 1010 served over HTTP 400
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 1010, resp.Code)
	})
}

func TestDiscoveryHandler_GetResults(t *testing.T) {
	env := newDiscoveryTestEnv(t)
	h := newDiscoveryHandler(env)
	seedDiscovery(t, env, "dr1", "results-task", models.DiscoveryStatusSuccess, 3)

	t.Run("found_returns_empty_devices_stub", func(t *testing.T) {
		// GetDiscoveryResults is a documented TODO stub — returns an empty list
		w := netServe(t, []netRoute{{http.MethodPost, "/discoveries/:id/results", h.GetResults}},
			http.MethodPost, "/discoveries/dr1/results", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.JSONEq(t, `{"devices":[]}`, string(resp.Data))
	})

	t.Run("not_found_1500", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/discoveries/:id/results", h.GetResults}},
			http.MethodPost, "/discoveries/none/results", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, 1500, resp.Code)
	})
}

func TestDiscoveryHandler_Execute(t *testing.T) {
	env := newDiscoveryTestEnv(t)
	h := newDiscoveryHandler(env)

	t.Run("task_not_found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/discoveries/:id/execute", h.Execute}},
			http.MethodPost, "/discoveries/none/execute", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "发现任务不存在")
	})
}

func TestDiscoveryHandler_Cancel(t *testing.T) {
	env := newDiscoveryTestEnv(t)
	h := newDiscoveryHandler(env)

	t.Run("pending_cancelled", func(t *testing.T) {
		seedDiscovery(t, env, "c1", "cancel-me", models.DiscoveryStatusPending, 0)
		w := netServe(t, []netRoute{{http.MethodPost, "/discoveries/:id/cancel", h.Cancel}},
			http.MethodPost, "/discoveries/c1/cancel", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), "取消成功")
		assert.Equal(t, 0, env.oper.lastBusinessType) // OperTypeOther

		var stored models.DeviceDiscovery
		require.NoError(t, env.db.Where("id = ?", "c1").First(&stored).Error)
		assert.Equal(t, models.DiscoveryStatusCancelled, stored.Status)
	})

	t.Run("completed_not_cancellable", func(t *testing.T) {
		seedDiscovery(t, env, "c2", "done", models.DiscoveryStatusSuccess, 1)
		w := netServe(t, []netRoute{{http.MethodPost, "/discoveries/:id/cancel", h.Cancel}},
			http.MethodPost, "/discoveries/c2/cancel", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "只能取消待执行或执行中的任务")
	})

	t.Run("not_found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/discoveries/:id/cancel", h.Cancel}},
			http.MethodPost, "/discoveries/none/cancel", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})
}

func TestDiscoveryHandler_Delete(t *testing.T) {
	env := newDiscoveryTestEnv(t)
	h := newDiscoveryHandler(env)

	t.Run("success_operType_delete", func(t *testing.T) {
		seedDiscovery(t, env, "del1", "delete-me", models.DiscoveryStatusSuccess, 0)
		w := netServe(t, []netRoute{{http.MethodPost, "/discoveries/:id/delete", h.Delete}},
			http.MethodPost, "/discoveries/del1/delete", "")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 3, env.oper.lastBusinessType)

		var count int64
		env.db.Model(&models.DeviceDiscovery{}).Where("id = ?", "del1").Count(&count)
		assert.Zero(t, count)
	})

	t.Run("running_rejected", func(t *testing.T) {
		seedDiscovery(t, env, "del2", "running", models.DiscoveryStatusRunning, 0)
		w := netServe(t, []netRoute{{http.MethodPost, "/discoveries/:id/delete", h.Delete}},
			http.MethodPost, "/discoveries/del2/delete", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "无法删除执行中的任务")
	})
}

func TestDiscoveryHandler_BatchDelete(t *testing.T) {
	env := newDiscoveryTestEnv(t)
	h := newDiscoveryHandler(env)
	seedDiscovery(t, env, "bd1", "batch-1", models.DiscoveryStatusSuccess, 0)
	seedDiscovery(t, env, "bd2", "batch-2", models.DiscoveryStatusSuccess, 0)

	t.Run("success_operType_batch", func(t *testing.T) {
		w := netPost(t, "/discoveries/batch-delete", h.BatchDelete, `{"discoveryIds":["bd1","bd2"]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"count":2`)
		assert.Equal(t, 16, env.oper.lastBusinessType)
	})

	t.Run("empty_ids_binding_400", func(t *testing.T) {
		w := netPost(t, "/discoveries/batch-delete", h.BatchDelete, `{"discoveryIds":[]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestDiscoveryHandler_ImportDevices(t *testing.T) {
	env := newDiscoveryTestEnv(t)
	h := newDiscoveryHandler(env)
	seedDiscovery(t, env, "imp1", "import-task", models.DiscoveryStatusSuccess, 2)

	t.Run("success_returns_zero_count_stub", func(t *testing.T) {
		// Handler passes deviceIDs=nil → ImportDiscoveredDevices stub returns len(nil)=0
		w := netServe(t, []netRoute{{http.MethodPost, "/discoveries/:id/import", h.ImportDevices}},
			http.MethodPost, "/discoveries/imp1/import", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.JSONEq(t, `{"count":0}`, string(resp.Data))
		assert.Equal(t, 6, env.oper.lastBusinessType) // OperTypeImport
	})

	t.Run("not_found_1500", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/discoveries/:id/import", h.ImportDevices}},
			http.MethodPost, "/discoveries/none/import", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, 1500, resp.Code)
	})
}

func TestDiscoveryHandler_Probe(t *testing.T) {
	env := newDiscoveryTestEnv(t)
	h := newDiscoveryHandler(env)
	credUUID := "11111111-2222-3333-4444-555555555555"

	t.Run("binding_requires_valid_ip_and_uuid", func(t *testing.T) {
		w := netPost(t, "/devices/discover", h.Probe, `{"ipAddress":"not-an-ip","credentialId":"nope"}`)
		resp := decodeNetResp(t, w)
		// responseHelpers.Error(c, 400, ...) → HTTP 400, body code 400
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("credential_not_found_returns_success_false", func(t *testing.T) {
		// Probe treats validation misses as SoftFail: HTTP 200 + Success=false (no error)
		w := netPost(t, "/devices/discover", h.Probe,
			`{"ipAddress":"10.0.0.1","credentialId":"`+credUUID+`"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), `"success":false`)
		assert.Contains(t, string(resp.Data), "授权凭证不存在")
		assert.Equal(t, 0, env.oper.lastBusinessType) // OperTypeOther — probe is audited
	})

	t.Run("credential_without_communities_rejected", func(t *testing.T) {
		cred := &models.AuthCredential{
			BaseModel:     models.BaseModel{ID: credUUID},
			CredentialName: "no-snmp",
			ProtocolType:  models.ProtocolType("ssh"),
		}
		require.NoError(t, env.db.Create(cred).Error)

		w := netPost(t, "/devices/discover", h.Probe,
			`{"ipAddress":"10.0.0.1","credentialId":"`+credUUID+`"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), "未配置 SNMP community")
	})

	t.Run("malformed_json_400", func(t *testing.T) {
		w := netPost(t, "/devices/discover", h.Probe, `{bad`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}
