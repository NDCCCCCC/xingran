package network

// CredentialHandler tests (Phase 74-03).
//
// CredentialHandler takes the CONCRETE *services.AuthCredentialService (no interface),
// so these tests exercise the real service against a glebarez sqlite in-memory DB
// (D-02), with SM4 replaced by netStubCipher and operlog captured by the package's
// mockOperLogService (D-03). Create/Update use RecordWithBody — the operlog mock
// asserts the sensitive-request path still records.

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
)

func newCredentialHandler(env *netTestEnv) *CredentialHandler {
	svc := services.NewAuthCredentialService(env.db, netStubCipher{})
	return NewCredentialHandler(svc).WithCore(env.core)
}

func newCredentialTestEnv(t *testing.T) *netTestEnv {
	return newNetworkTestEnv(t, &models.AuthCredential{}, &models.NetworkDevice{})
}

func seedCredential(t *testing.T, env *netTestEnv, id, name string, protocol models.ProtocolType, isDefault bool) *models.AuthCredential {
	t.Helper()
	c := &models.AuthCredential{
		BaseModel:     models.BaseModel{ID: id},
		CredentialName: name,
		ProtocolType:  protocol,
		Username:      "admin",
		Password:      "enc:secret",
		SNMPVersion:   "v2c",
		IsDefault:     isDefault,
	}
	require.NoError(t, env.db.Create(c).Error)
	return c
}

func TestCredentialHandler_Statistics(t *testing.T) {
	env := newCredentialTestEnv(t)
	h := newCredentialHandler(env)

	t.Run("success_counts_by_protocol", func(t *testing.T) {
		seedCredential(t, env, "c1", "ssh-1", models.ProtocolType("ssh"), false)
		seedCredential(t, env, "c2", "ssh-2", models.ProtocolType("ssh"), true)
		seedCredential(t, env, "c3", "tel-1", models.ProtocolType("telnet"), false)

		w := netPost(t, "/credentials/statistics", h.Statistics, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.JSONEq(t, `{"total":3,"ssh":2,"telnet":1}`, string(resp.Data))
	})

	t.Run("empty_table_zeros", func(t *testing.T) {
		freshEnv := newCredentialTestEnv(t)
		freshH := newCredentialHandler(freshEnv)
		w := netPost(t, "/credentials/statistics", freshH.Statistics, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), `"total":0`)
	})
}

func TestCredentialHandler_List(t *testing.T) {
	env := newCredentialTestEnv(t)
	h := newCredentialHandler(env)
	seedCredential(t, env, "c1", "core-ssh", models.ProtocolType("ssh"), false)
	seedCredential(t, env, "c2", "edge-ssh", models.ProtocolType("ssh"), false)
	seedCredential(t, env, "c3", "legacy-telnet", models.ProtocolType("telnet"), true)

	t.Run("filter_by_name_and_protocol", func(t *testing.T) {
		w := netPost(t, "/credentials/list", h.List,
			`{"current":1,"pageSize":10,"credentialName":"ssh","protocolType":"ssh"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), `"total":2`)
		assert.Contains(t, string(resp.Data), "core-ssh")
		assert.Contains(t, string(resp.Data), "edge-ssh")
		assert.NotContains(t, string(resp.Data), "legacy-telnet")
	})

	t.Run("bad_body_falls_back_to_defaults", func(t *testing.T) {
		w := netPost(t, "/credentials/list", h.List, `not-json`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), `"total":3`)
		assert.Contains(t, string(resp.Data), `"current":1`)
		assert.Contains(t, string(resp.Data), `"pageSize":10`)
	})

	t.Run("sorted_by_username_desc", func(t *testing.T) {
		w := netPost(t, "/credentials/list", h.List,
			`{"current":1,"pageSize":10,"orderByColumn":"credentialName","isAsc":false}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		// legacy-telnet sorts before edge-ssh before core-ssh ascending → desc = legacy first
		body := string(resp.Data)
		assert.Less(t, indexOf(body, "legacy-telnet"), indexOf(body, "core-ssh"))
	})
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestCredentialHandler_GetByID(t *testing.T) {
	env := newCredentialTestEnv(t)
	h := newCredentialHandler(env)
	seedCredential(t, env, "c9", "find-me", models.ProtocolType("ssh"), false)

	t.Run("found_password_hidden", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/credentials/:id", h.GetByID}},
			http.MethodPost, "/credentials/c9", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), "find-me")
		// Password/EnablePassword carry json:"-" so they never serialize
		assert.NotContains(t, string(resp.Data), "enc:secret")
	})

	t.Run("not_found_wrapped_500", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/credentials/:id", h.GetByID}},
			http.MethodPost, "/credentials/none", "")
		resp := decodeNetResp(t, w)
		// apperrors.InternalServerError → HTTP 500, body code 1500
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, 1500, resp.Code)
	})
}

func TestCredentialHandler_Create(t *testing.T) {
	env := newCredentialTestEnv(t)
	h := newCredentialHandler(env)

	t.Run("success_encrypts_and_strips_password", func(t *testing.T) {
		w := netPost(t, "/credentials", h.Create,
			`{"credentialName":"new-cred","protocolType":"ssh","username":"op","password":"pw123","enablePassword":"en456","snmpCommunities":["public"],"description":"d"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), "new-cred")
		assert.NotContains(t, string(resp.Data), "pw123")

		var stored models.AuthCredential
		require.NoError(t, env.db.Where("credential_name = ?", "new-cred").First(&stored).Error)
		assert.Equal(t, "enc:pw123", stored.Password)
		assert.Equal(t, "enc:en456", stored.EnablePassword)
		assert.Equal(t, "user-0001", stored.CreatedBy, "CreatedBy injected from auth context user_id")

		assert.Equal(t, 1, env.oper.recordAsyncCalls)
		assert.Equal(t, "网络设备凭据", env.oper.lastTitle)
		assert.Equal(t, 1, env.oper.lastBusinessType) // OperTypeCreate
		// RecordWithBody must have sanitized the password before storing operParam
		require.NotNil(t, env.oper.lastOperParam)
		assert.Contains(t, *env.oper.lastOperParam, "******")
		assert.NotContains(t, *env.oper.lastOperParam, "pw123")
	})

	t.Run("duplicate_name_rejected", func(t *testing.T) {
		seedCredential(t, env, "dup", "dup-name", models.ProtocolType("ssh"), false)
		w := netPost(t, "/credentials", h.Create,
			`{"credentialName":"dup-name","protocolType":"ssh","username":"op","password":"x"}`)
		resp := decodeNetResp(t, w)
		// HandleServiceError → int 500 → HTTP 400, code 500 (quirk)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "凭证名称已存在")
	})

	t.Run("missing_username_rejected_by_service", func(t *testing.T) {
		// CreateCredentialRequest has NO binding tags → empty body reaches the service,
		// which enforces username/password for new credentials.
		w := netPost(t, "/credentials", h.Create, `{"credentialName":"no-user","password":"x"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "请输入用户名")
	})

	t.Run("missing_password_rejected_by_service", func(t *testing.T) {
		w := netPost(t, "/credentials", h.Create, `{"credentialName":"no-pass","username":"op"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "请输入密码")
	})

	t.Run("malformed_json_400", func(t *testing.T) {
		w := netPost(t, "/credentials", h.Create, `{bad`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestCredentialHandler_Update(t *testing.T) {
	env := newCredentialTestEnv(t)
	h := newCredentialHandler(env)
	seedCredential(t, env, "u1", "update-me", models.ProtocolType("ssh"), false)

	t.Run("success_keeps_old_password_when_blank", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/credentials/:id/update", h.Update}},
			http.MethodPost, "/credentials/u1/update",
			`{"credentialName":"updated","protocolType":"ssh","username":"op2"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), "更新成功")

		var stored models.AuthCredential
		require.NoError(t, env.db.Where("id = ?", "u1").First(&stored).Error)
		assert.Equal(t, "updated", stored.CredentialName)
		assert.Equal(t, "op2", stored.Username)
		assert.Equal(t, "enc:secret", stored.Password, "blank password must not overwrite")
		assert.Equal(t, "user-0001", stored.UpdatedBy)
		assert.Equal(t, 2, env.oper.lastBusinessType) // OperTypeUpdate
	})

	t.Run("new_password_encrypted", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/credentials/:id/update", h.Update}},
			http.MethodPost, "/credentials/u1/update",
			`{"credentialName":"updated","protocolType":"ssh","username":"op2","password":"fresh"}`)
		assert.Equal(t, http.StatusOK, w.Code)

		var stored models.AuthCredential
		require.NoError(t, env.db.Where("id = ?", "u1").First(&stored).Error)
		assert.Equal(t, "enc:fresh", stored.Password)
	})

	t.Run("not_found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/credentials/:id/update", h.Update}},
			http.MethodPost, "/credentials/ghost/update",
			`{"credentialName":"x","username":"op"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "凭证不存在")
	})
}

func TestCredentialHandler_Delete(t *testing.T) {
	env := newCredentialTestEnv(t)
	h := newCredentialHandler(env)

	t.Run("success_operType_delete", func(t *testing.T) {
		seedCredential(t, env, "d1", "delete-me", models.ProtocolType("ssh"), false)
		w := netServe(t, []netRoute{{http.MethodPost, "/credentials/:id/delete", h.Delete}},
			http.MethodPost, "/credentials/d1/delete", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, 3, env.oper.lastBusinessType) // OperTypeDelete

		var count int64
		env.db.Model(&models.AuthCredential{}).Where("id = ?", "d1").Count(&count)
		assert.Zero(t, count)
	})

	t.Run("in_use_rejected", func(t *testing.T) {
		seedCredential(t, env, "d2", "in-use", models.ProtocolType("ssh"), false)
		credID := "d2"
		dev := &models.NetworkDevice{
			BaseModel:    models.BaseModel{ID: "dev-d2"},
			DeviceName:   "uses-cred",
			DeviceType:   models.DeviceType("switch"),
			Vendor:       models.DeviceVendor("huawei"),
			IPAddress:    "10.3.3.3",
			CredentialID: &credID,
		}
		require.NoError(t, env.db.Create(dev).Error)

		w := netServe(t, []netRoute{{http.MethodPost, "/credentials/:id/delete", h.Delete}},
			http.MethodPost, "/credentials/d2/delete", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "正在使用此凭证")
	})
}

func TestCredentialHandler_BatchDelete(t *testing.T) {
	env := newCredentialTestEnv(t)
	h := newCredentialHandler(env)
	seedCredential(t, env, "b1", "batch-1", models.ProtocolType("ssh"), false)
	seedCredential(t, env, "b2", "batch-2", models.ProtocolType("ssh"), false)

	t.Run("success_operType_batch", func(t *testing.T) {
		w := netPost(t, "/credentials/batch-delete", h.BatchDelete, `{"ids":["b1","b2"]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), `"count":2`)
		assert.Equal(t, 16, env.oper.lastBusinessType) // OperTypeBatch
	})

	t.Run("empty_ids_binding_400", func(t *testing.T) {
		w := netPost(t, "/credentials/batch-delete", h.BatchDelete, `{"ids":[]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestCredentialHandler_SetDefault(t *testing.T) {
	env := newCredentialTestEnv(t)
	h := newCredentialHandler(env)
	seedCredential(t, env, "s1", "old-default", models.ProtocolType("ssh"), true)
	seedCredential(t, env, "s2", "new-default", models.ProtocolType("ssh"), false)

	t.Run("success_clears_previous_default_operType_grant", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/credentials/:id/set-default", h.SetDefault}},
			http.MethodPost, "/credentials/s2/set-default", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		// SetDefault 语义为授权 → OperTypeGrant(4)
		assert.Equal(t, 4, env.oper.lastBusinessType)

		var old, newCred models.AuthCredential
		require.NoError(t, env.db.Where("id = ?", "s1").First(&old).Error)
		require.NoError(t, env.db.Where("id = ?", "s2").First(&newCred).Error)
		assert.False(t, old.IsDefault)
		assert.True(t, newCred.IsDefault)
		assert.Equal(t, "user-0001", newCred.UpdatedBy)
	})
}

func TestCredentialHandler_GetDevicesByCredential(t *testing.T) {
	env := newCredentialTestEnv(t)
	h := newCredentialHandler(env)
	seedCredential(t, env, "g1", "shared", models.ProtocolType("ssh"), false)

	t.Run("success_empty_list", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/credentials/:id/devices", h.GetDevicesByCredential}},
			http.MethodPost, "/credentials/g1/devices", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		// response.Success wraps the raw slice — empty result serializes as []
		assert.Equal(t, "[]", string(resp.Data))
	})

	t.Run("success_with_devices", func(t *testing.T) {
		credID := "g1"
		dev := &models.NetworkDevice{
			BaseModel:   models.BaseModel{ID: "dev-g1"},
			DeviceName:  "bound-device",
			DeviceType:  models.DeviceType("switch"),
			Vendor:      models.DeviceVendor("huawei"),
			IPAddress:   "10.4.4.4",
			CredentialID: &credID,
		}
		require.NoError(t, env.db.Create(dev).Error)

		w := netServe(t, []netRoute{{http.MethodPost, "/credentials/:id/devices", h.GetDevicesByCredential}},
			http.MethodPost, "/credentials/g1/devices", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), "bound-device")
	})
}
