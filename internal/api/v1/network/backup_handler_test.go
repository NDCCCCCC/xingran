package network

// BackupHandler tests (Phase 74-03).
//
// ConfigBackupService wraps a device.DeviceExecutor for live fetches; only
// CreateBackup's success path reaches it (after the handler-side device lookup),
// so tests inject a nil executor and cover: device-not-found, all DB-only flows
// (list/content/diff/statistics/delete/history/version), binding errors, and the
// documented RestoreBackup stub ("配置恢复功能待实现").

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
)

func newBackupTestEnv(t *testing.T) *netTestEnv {
	return newNetworkTestEnv(t, &models.ConfigBackup{}, &models.NetworkDevice{})
}

func newBackupHandler(env *netTestEnv) *BackupHandler {
	return NewBackupHandler(services.NewConfigBackupService(env.db, nil), env.db).WithCore(env.core)
}

func seedBackup(t *testing.T, env *netTestEnv, id, deviceID, deviceName string, version int, content string) *models.ConfigBackup {
	t.Helper()
	b := &models.ConfigBackup{
		ID:            id,
		DeviceID:      deviceID,
		DeviceName:    deviceName,
		BackupType:    models.BackupTypeManual,
		StorageType:   models.StorageTypeDatabase,
		Version:       version,
		ConfigContent: content,
		BackupSize:    len(content),
	}
	require.NoError(t, env.db.Create(b).Error)
	return b
}

func TestBackupHandler_List(t *testing.T) {
	env := newBackupTestEnv(t)
	h := newBackupHandler(env)
	netSeedDevice(t, env.db, "dev-b1", "backup-dev-1", "10.0.0.1")
	seedBackup(t, env, "bk1", "dev-b1", "backup-dev-1", 1, "conf-a")
	seedBackup(t, env, "bk2", "dev-b2", "other-dev", 1, "conf-b")

	t.Run("all_backups_with_device_ip", func(t *testing.T) {
		w := netPost(t, "/backups/list", h.List, `{"current":1,"pageSize":10}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"total":2`)
		assert.Contains(t, string(resp.Data), `"ipAddress":"10.0.0.1"`)
	})

	t.Run("filter_by_deviceId", func(t *testing.T) {
		w := netPost(t, "/backups/list", h.List, `{"current":1,"pageSize":10,"deviceId":"dev-b2"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"total":1`)
		assert.Contains(t, string(resp.Data), "conf-b")
	})

	t.Run("bad_body_defaults", func(t *testing.T) {
		w := netPost(t, "/backups/list", h.List, `not-json`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"total":2`)
	})
}

func TestBackupHandler_Create(t *testing.T) {
	env := newBackupTestEnv(t)
	h := newBackupHandler(env)

	t.Run("device_not_found_7000", func(t *testing.T) {
		w := netPost(t, "/backups", h.Create, `{"deviceId":"ghost","backupType":"manual"}`)
		resp := decodeNetResp(t, w)
		// apperrors.NetworkDeviceNotFound → code 7000, HTTP 400
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 7000, resp.Code)
	})

	t.Run("binding_requires_deviceId", func(t *testing.T) {
		w := netPost(t, "/backups", h.Create, `{"backupType":"manual"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("binding_backupType_oneof", func(t *testing.T) {
		w := netPost(t, "/backups", h.Create, `{"deviceId":"x","backupType":"weekly"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestBackupHandler_GetContent(t *testing.T) {
	env := newBackupTestEnv(t)
	h := newBackupHandler(env)
	seedBackup(t, env, "bk9", "dev-b9", "dev-nine", 3, "vlan 100")

	t.Run("path_variant_found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/backups/:id", h.GetContent}},
			http.MethodPost, "/backups/bk9", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.JSONEq(t, `{"content":"vlan 100"}`, string(resp.Data))
	})

	t.Run("path_variant_not_found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/backups/:id", h.GetContent}},
			http.MethodPost, "/backups/none", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, 1500, resp.Code)
	})

	t.Run("body_variant_found", func(t *testing.T) {
		w := netPost(t, "/backups/content", h.GetContentFromBody, `{"id":"bk9"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"content":"vlan 100"}`, string(resp.Data))
	})

	t.Run("body_variant_binding_error", func(t *testing.T) {
		w := netPost(t, "/backups/content", h.GetContentFromBody, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestBackupHandler_Delete(t *testing.T) {
	env := newBackupTestEnv(t)
	h := newBackupHandler(env)
	seedBackup(t, env, "bk-del", "dev-d", "dev-d", 1, "x")

	t.Run("success_operType_delete", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/backups/:id/delete", h.Delete}},
			http.MethodPost, "/backups/bk-del/delete", "")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 3, env.oper.lastBusinessType)

		var count int64
		env.db.Model(&models.ConfigBackup{}).Where("id = ?", "bk-del").Count(&count)
		assert.Zero(t, count)
	})

	t.Run("not_found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/backups/:id/delete", h.Delete}},
			http.MethodPost, "/backups/none/delete", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "备份记录不存在")
	})
}

func TestBackupHandler_BatchDelete(t *testing.T) {
	env := newBackupTestEnv(t)
	h := newBackupHandler(env)
	seedBackup(t, env, "bkb1", "dev-1", "d1", 1, "x")
	seedBackup(t, env, "bkb2", "dev-2", "d2", 1, "y")

	t.Run("success_operType_batch", func(t *testing.T) {
		w := netPost(t, "/backups/batch-delete", h.BatchDelete, `{"backupIds":["bkb1","bkb2"]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"count":2`)
		assert.Equal(t, 16, env.oper.lastBusinessType)
	})

	t.Run("empty_ids_binding_400", func(t *testing.T) {
		w := netPost(t, "/backups/batch-delete", h.BatchDelete, `{"backupIds":[]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestBackupHandler_Diff(t *testing.T) {
	env := newBackupTestEnv(t)
	h := newBackupHandler(env)
	seedBackup(t, env, "bd1", "dev-diff", "diff-dev", 1, "line1\nsame\n")
	seedBackup(t, env, "bd2", "dev-diff", "diff-dev", 2, "line1\nchanged\n")

	t.Run("success_returns_names_and_diff", func(t *testing.T) {
		w := netPost(t, "/backups/diff", h.Diff, `{"backupId1":"bd1","backupId2":"bd2"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), `"name1":"diff-dev (版本1)"`)
		assert.Contains(t, string(resp.Data), `"name2":"diff-dev (版本2)"`)
		assert.Contains(t, string(resp.Data), "changed")
	})

	t.Run("missing_backup", func(t *testing.T) {
		w := netPost(t, "/backups/diff", h.Diff, `{"backupId1":"ghost","backupId2":"bd2"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})

	t.Run("binding_error", func(t *testing.T) {
		w := netPost(t, "/backups/diff", h.Diff, `{"backupId1":"bd1"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestBackupHandler_Restore(t *testing.T) {
	env := newBackupTestEnv(t)
	h := newBackupHandler(env)
	seedBackup(t, env, "bk-r", "dev-r", "restore-dev", 1, "config")

	t.Run("backup_not_found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/backups/:id/restore", h.Restore}},
			http.MethodPost, "/backups/none/restore", `{"deviceId":"dev-r"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})

	t.Run("restore_is_a_documented_stub", func(t *testing.T) {
		// RestoreBackup is a TODO stub — it always fails with 配置恢复功能待实现 (D-12: not fixed here)
		w := netServe(t, []netRoute{{http.MethodPost, "/backups/:id/restore", h.Restore}},
			http.MethodPost, "/backups/bk-r/restore", `{"deviceId":"dev-r"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "配置恢复功能待实现")
	})

	t.Run("binding_requires_deviceId", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/backups/:id/restore", h.Restore}},
			http.MethodPost, "/backups/bk-r/restore", `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestBackupHandler_GetStatistics(t *testing.T) {
	env := newBackupTestEnv(t)
	h := newBackupHandler(env)
	seedBackup(t, env, "st1", "dev-s1", "s1", 1, "aaaa")
	seedBackup(t, env, "st2", "dev-s2", "s2", 1, "bb")

	w := netGet(t, "/backups/statistics", h.GetStatistics, "")
	resp := decodeNetResp(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, resp.Code)
	assert.Contains(t, string(resp.Data), `"totalBackups":2`)
	assert.Contains(t, string(resp.Data), `"manualBackups":2`)
	assert.Contains(t, string(resp.Data), `"dbStorageCount":2`)
	assert.Contains(t, string(resp.Data), `"uniqueDevices":2`)
}

func TestBackupHandler_BatchBackup(t *testing.T) {
	env := newBackupTestEnv(t)
	h := newBackupHandler(env)

	t.Run("ghost_devices_return_null_results", func(t *testing.T) {
		// Nonexistent devices are skipped before the executor is touched → 200 + data:null
		// (nil slice serializes as null — quirk locked here)
		w := netPost(t, "/backups/batch", h.BatchBackup, `{"deviceIds":["ghost"],"backupType":"manual"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, "null", string(resp.Data))
	})

	t.Run("binding_requires_ids", func(t *testing.T) {
		w := netPost(t, "/backups/batch", h.BatchBackup, `{"backupType":"manual"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("binding_backupType_oneof", func(t *testing.T) {
		w := netPost(t, "/backups/batch", h.BatchBackup, `{"deviceIds":["x"],"backupType":"weekly"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestBackupHandler_GetByVersion(t *testing.T) {
	env := newBackupTestEnv(t)
	h := newBackupHandler(env)
	seedBackup(t, env, "bv2", "dev-v", "ver-dev", 2, "version-two")

	t.Run("found", func(t *testing.T) {
		w := netGet(t, "/backups/version", h.GetByVersion, "?deviceId=dev-v&version=2")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), "version-two")
	})

	t.Run("missing_params", func(t *testing.T) {
		w := netGet(t, "/backups/version", h.GetByVersion, "?deviceId=dev-v")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 1001, resp.Code)
	})

	t.Run("bad_version_format", func(t *testing.T) {
		w := netGet(t, "/backups/version", h.GetByVersion, "?deviceId=dev-v&version=x")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 1001, resp.Code)
	})

	t.Run("not_found_7040", func(t *testing.T) {
		w := netGet(t, "/backups/version", h.GetByVersion, "?deviceId=dev-v&version=99")
		resp := decodeNetResp(t, w)
		// apperrors.BackupNotFound → code 7040, HTTP 400
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 7040, resp.Code)
	})
}

func TestBackupHandler_GetHistory(t *testing.T) {
	env := newBackupTestEnv(t)
	h := newBackupHandler(env)
	seedBackup(t, env, "h1", "dev-h", "hist-dev", 1, "v1")
	seedBackup(t, env, "h2", "dev-h", "hist-dev", 2, "v2")

	t.Run("ordered_by_version_desc", func(t *testing.T) {
		w := netGet(t, "/backups/history", h.GetHistory, "?deviceId=dev-h")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		body := string(resp.Data)
		assert.Less(t, indexOf(body, `"version":2`), indexOf(body, `"version":1`))
	})

	t.Run("missing_deviceId_1002", func(t *testing.T) {
		w := netGet(t, "/backups/history", h.GetHistory, "")
		resp := decodeNetResp(t, w)
		// apperrors.ParamMissing → code 1002, HTTP 400
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 1002, resp.Code)
	})
}
