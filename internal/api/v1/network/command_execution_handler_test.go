package network

// CommandHandler + ExecutionHandler tests (Phase 74-03).
//
// Both handlers take CONCRETE service structs that wrap a device.DeviceExecutor.
// The executor is only reached AFTER all pre-flight validation (device existence,
// template render), so these tests inject a nil executor and cover: every binding
// path, every early-validation error path, and all DB-only flows (statistics/list/
// result/cancel/delete). The happy-path Dispatch/QuickCommand/ExecuteByTemplate
// require a live device connection and are intentionally out of scope (D-12).

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
)

func newCommandTestEnv(t *testing.T) *netTestEnv {
	return newNetworkTestEnv(t, &models.NetworkDevice{}, &models.ConfigExecution{},
		&models.ConfigExecutionDetail{}, &models.ConfigTemplate{})
}

func newCommandHandler(env *netTestEnv) *CommandHandler {
	return NewCommandHandler(services.NewCommandDispatchService(env.db, nil), env.db).WithCore(env.core)
}

func newExecHandler(env *netTestEnv) *ExecutionHandler {
	return NewExecutionHandler(services.NewConfigExecutionService(env.db, nil)).WithCore(env.core)
}

func seedExecution(t *testing.T, env *netTestEnv, id, name string, execType models.ExecutionType, status models.ExecutionStatus) *models.ConfigExecution {
	t.Helper()
	e := &models.ConfigExecution{
		ID:                id,
		ExecutionName:     name,
		ExecutionType:     execType,
		DeviceIDs:         models.DeviceIDList{"dev-1"},
		Status:            status,
		TotalDevices:      1,
		SuccessCount:      1,
		ExecutionStrategy: models.ExecutionStrategyParallel,
		Concurrency:       10,
		Timeout:           300,
		CommandContent:    "display version",
		CreatedBy:         "tester",
	}
	require.NoError(t, env.db.Create(e).Error)
	return e
}

// --- CommandHandler ---

func TestCommandHandler_Statistics(t *testing.T) {
	env := newCommandTestEnv(t)
	h := newCommandHandler(env)

	t.Run("counts_command_type_only", func(t *testing.T) {
		seedExecution(t, env, "cmd1", "c1", models.ExecutionTypeCommand, models.ExecutionStatusSuccess)
		seedExecution(t, env, "cmd2", "c2", models.ExecutionTypeCommand, models.ExecutionStatusPending)
		seedExecution(t, env, "tpl1", "t1", models.ExecutionTypeTemplate, models.ExecutionStatusSuccess)

		w := netPost(t, "/command/statistics", h.Statistics, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.JSONEq(t, `{"total":2,"pending":1,"running":0,"success":1,"failed":0}`, string(resp.Data))
	})

	t.Run("empty_zeros", func(t *testing.T) {
		fresh := newCommandTestEnv(t)
		w := netPost(t, "/command/statistics", newCommandHandler(fresh).Statistics, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"total":0`)
	})
}

func TestCommandHandler_List(t *testing.T) {
	env := newCommandTestEnv(t)
	h := newCommandHandler(env)
	seedExecution(t, env, "cmd1", "cmd-run", models.ExecutionTypeCommand, models.ExecutionStatusSuccess)
	seedExecution(t, env, "tpl1", "tpl-run", models.ExecutionTypeTemplate, models.ExecutionStatusSuccess)

	t.Run("command_type_only", func(t *testing.T) {
		w := netPost(t, "/command/list", h.List, `{"current":1,"pageSize":10}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"total":1`)
		assert.Contains(t, string(resp.Data), "cmd-run")
		assert.NotContains(t, string(resp.Data), "tpl-run")
	})

	t.Run("bad_body_defaults", func(t *testing.T) {
		w := netPost(t, "/command/list", h.List, `not-json`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"current":1`)
		assert.Contains(t, string(resp.Data), `"pageSize":10`)
	})
}

func TestCommandHandler_Dispatch(t *testing.T) {
	env := newCommandTestEnv(t)
	h := newCommandHandler(env)

	t.Run("empty_devices_rejected", func(t *testing.T) {
		w := netPost(t, "/command/dispatch", h.Dispatch,
			`{"ExecutionName":"batch","CommandContent":"display version"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "请选择要执行命令的设备")
	})

	t.Run("nonexistent_device_rejected", func(t *testing.T) {
		w := netPost(t, "/command/dispatch", h.Dispatch,
			`{"ExecutionName":"batch","DeviceIDs":["ghost"],"CommandContent":"display version"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "部分设备不存在")
	})

	t.Run("malformed_json_400", func(t *testing.T) {
		w := netPost(t, "/command/dispatch", h.Dispatch, `{bad`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestCommandHandler_QuickCommand(t *testing.T) {
	env := newCommandTestEnv(t)
	h := newCommandHandler(env)

	t.Run("device_not_found", func(t *testing.T) {
		w := netPost(t, "/command/quick", h.QuickCommand, `{"deviceId":"ghost","command":"display version"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "设备不存在")
	})

	t.Run("binding_requires_device_and_command", func(t *testing.T) {
		w := netPost(t, "/command/quick", h.QuickCommand, `{"deviceId":"x"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("timeout_out_of_range", func(t *testing.T) {
		// binding omitempty,min=10,max=300 → timeout=5 rejected before service call
		w := netPost(t, "/command/quick", h.QuickCommand,
			`{"deviceId":"x","command":"y","timeout":5}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestCommandHandler_GetExecutionResult(t *testing.T) {
	env := newCommandTestEnv(t)
	h := newCommandHandler(env)
	seedExecution(t, env, "cmd1", "with-detail", models.ExecutionTypeCommand, models.ExecutionStatusSuccess)
	detail := &models.ConfigExecutionDetail{
		ID: "det1", ExecutionID: "cmd1", DeviceID: "dev-1", DeviceName: "core-sw",
		IPAddress: "10.0.0.1", Status: models.ExecutionStatusSuccess,
		CommandSent: "display version", OutputReceived: "Version 8.180",
	}
	require.NoError(t, env.db.Create(detail).Error)

	t.Run("found_with_details", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/command/:id", h.GetExecutionResult}},
			http.MethodPost, "/command/cmd1", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), "core-sw")
	})

	t.Run("not_found_1500", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/command/:id", h.GetExecutionResult}},
			http.MethodPost, "/command/none", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, 1500, resp.Code)
	})
}

func TestCommandHandler_GetDeviceExecutionDetail(t *testing.T) {
	env := newCommandTestEnv(t)
	h := newCommandHandler(env)
	detail := &models.ConfigExecutionDetail{
		ID: "det9", ExecutionID: "cmd9", DeviceID: "dev-9", DeviceName: "edge-sw", Status: models.ExecutionStatusSuccess,
	}
	require.NoError(t, env.db.Create(detail).Error)

	t.Run("found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/command/:id/device/:deviceId", h.GetDeviceExecutionDetail}},
			http.MethodPost, "/command/cmd9/device/dev-9", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), "edge-sw")
	})

	t.Run("not_found_quirk_1010_over_400", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/command/:id/device/:deviceId", h.GetDeviceExecutionDetail}},
			http.MethodPost, "/command/cmd9/device/none", "")
		resp := decodeNetResp(t, w)
		// apperrors.NotFound(1010) maps to HTTP 400 (2xxx-8xxx range rule)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 1010, resp.Code)
	})
}

// --- ExecutionHandler ---

func TestExecutionHandler_Statistics(t *testing.T) {
	env := newCommandTestEnv(t)
	h := newExecHandler(env)
	seedExecution(t, env, "tpl1", "t1", models.ExecutionTypeTemplate, models.ExecutionStatusRunning)
	seedExecution(t, env, "tpl2", "t2", models.ExecutionTypeTemplate, models.ExecutionStatusFailed)

	w := netPost(t, "/executions/statistics", h.Statistics, `{}`)
	resp := decodeNetResp(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"total":2,"pending":0,"running":1,"success":0,"failed":1}`, string(resp.Data))
}

func TestExecutionHandler_List(t *testing.T) {
	env := newCommandTestEnv(t)
	h := newExecHandler(env)
	seedExecution(t, env, "tpl1", "tpl-run", models.ExecutionTypeTemplate, models.ExecutionStatusSuccess)
	seedExecution(t, env, "cmd1", "cmd-run", models.ExecutionTypeCommand, models.ExecutionStatusSuccess)

	w := netPost(t, "/executions/list", h.List, `{"current":1,"pageSize":10}`)
	resp := decodeNetResp(t, w)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, string(resp.Data), `"total":1`)
	assert.Contains(t, string(resp.Data), "tpl-run")
	assert.NotContains(t, string(resp.Data), "cmd-run")
}

func TestExecutionHandler_ExecuteByTemplate(t *testing.T) {
	env := newCommandTestEnv(t)
	h := newExecHandler(env)

	t.Run("empty_devices_rejected", func(t *testing.T) {
		w := netPost(t, "/executions/template/execute", h.ExecuteByTemplate, `{"TemplateID":"t","ExecutionName":"x"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "请选择要执行配置的设备")
	})

	t.Run("template_missing", func(t *testing.T) {
		w := netPost(t, "/executions/template/execute", h.ExecuteByTemplate,
			`{"TemplateID":"ghost","DeviceIDs":["dev-1"],"ExecutionName":"x"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "模板不存在")
	})

	t.Run("device_missing", func(t *testing.T) {
		tpl := &models.ConfigTemplate{
			BaseModel: models.BaseModel{ID: "tpl-x"}, TemplateName: "x", TemplateCode: "X",
			TemplateType: models.TemplateTypeConfig, TemplateContent: "static",
		}
		require.NoError(t, env.db.Create(tpl).Error)

		w := netPost(t, "/executions/template/execute", h.ExecuteByTemplate,
			`{"TemplateID":"tpl-x","DeviceIDs":["ghost"],"ExecutionName":"x"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "部分设备不存在")
	})

	t.Run("required_variable_missing_render_error", func(t *testing.T) {
		tpl := &models.ConfigTemplate{
			BaseModel: models.BaseModel{ID: "tpl-y"}, TemplateName: "y", TemplateCode: "Y",
			TemplateType: models.TemplateTypeConfig, TemplateContent: "vlan {{.Vlan}}",
			Variables: models.TemplateVariables{{Name: "Vlan", Required: true, Type: "string"}},
		}
		require.NoError(t, env.db.Create(tpl).Error)
		netSeedDevice(t, env.db, "dev-r", "render-dev", "10.0.0.9")

		w := netPost(t, "/executions/template/execute", h.ExecuteByTemplate,
			`{"TemplateID":"tpl-y","DeviceIDs":["dev-r"],"ExecutionName":"x"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "渲染模板失败")
	})
}

func TestExecutionHandler_GetByID(t *testing.T) {
	env := newCommandTestEnv(t)
	h := newExecHandler(env)
	seedExecution(t, env, "tpl1", "find-me", models.ExecutionTypeTemplate, models.ExecutionStatusSuccess)

	t.Run("found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/executions/:id", h.GetByID}},
			http.MethodPost, "/executions/tpl1", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		// GetExecutionResult projects to {ExecutionID, Results, Summary} — name not included
		assert.Contains(t, string(resp.Data), `"ExecutionID":"tpl1"`)
		assert.Contains(t, string(resp.Data), `"SuccessCount":1`)
	})

	t.Run("not_found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/executions/:id", h.GetByID}},
			http.MethodPost, "/executions/none", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, 1500, resp.Code)
	})
}

func TestExecutionHandler_Cancel(t *testing.T) {
	env := newCommandTestEnv(t)
	h := newExecHandler(env)

	t.Run("pending_cancelled_to_failed", func(t *testing.T) {
		seedExecution(t, env, "can1", "cancel-me", models.ExecutionTypeTemplate, models.ExecutionStatusPending)
		w := netServe(t, []netRoute{{http.MethodPost, "/executions/:id/cancel", h.Cancel}},
			http.MethodPost, "/executions/can1/cancel", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), "取消成功")
		assert.Equal(t, 0, env.oper.lastBusinessType) // OperTypeOther

		var stored models.ConfigExecution
		require.NoError(t, env.db.Where("id = ?", "can1").First(&stored).Error)
		assert.Equal(t, models.ExecutionStatusFailed, stored.Status)
		assert.Equal(t, "用户取消执行", stored.ErrorMessage)
	})

	t.Run("completed_not_cancellable", func(t *testing.T) {
		seedExecution(t, env, "can2", "done", models.ExecutionTypeTemplate, models.ExecutionStatusSuccess)
		w := netServe(t, []netRoute{{http.MethodPost, "/executions/:id/cancel", h.Cancel}},
			http.MethodPost, "/executions/can2/cancel", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "只能取消待执行或执行中的任务")
	})

	t.Run("not_found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/executions/:id/cancel", h.Cancel}},
			http.MethodPost, "/executions/none/cancel", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
	})
}

func TestExecutionHandler_Delete(t *testing.T) {
	env := newCommandTestEnv(t)
	h := newExecHandler(env)

	t.Run("success_cascades_details", func(t *testing.T) {
		seedExecution(t, env, "del1", "delete-me", models.ExecutionTypeTemplate, models.ExecutionStatusSuccess)
		require.NoError(t, env.db.Create(&models.ConfigExecutionDetail{
			ID: "dd1", ExecutionID: "del1", DeviceID: "dev-1", Status: models.ExecutionStatusSuccess,
		}).Error)

		w := netServe(t, []netRoute{{http.MethodPost, "/executions/:id/delete", h.Delete}},
			http.MethodPost, "/executions/del1/delete", "")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 3, env.oper.lastBusinessType) // OperTypeDelete

		var execCount, detailCount int64
		env.db.Model(&models.ConfigExecution{}).Where("id = ?", "del1").Count(&execCount)
		env.db.Model(&models.ConfigExecutionDetail{}).Where("execution_id = ?", "del1").Count(&detailCount)
		assert.Zero(t, execCount)
		assert.Zero(t, detailCount, "details must cascade")
	})

	t.Run("running_rejected", func(t *testing.T) {
		seedExecution(t, env, "del2", "running", models.ExecutionTypeTemplate, models.ExecutionStatusRunning)
		w := netServe(t, []netRoute{{http.MethodPost, "/executions/:id/delete", h.Delete}},
			http.MethodPost, "/executions/del2/delete", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "无法删除执行中的任务")
	})
}

func TestExecutionHandler_BatchDelete(t *testing.T) {
	env := newCommandTestEnv(t)
	h := newExecHandler(env)
	seedExecution(t, env, "bd1", "ok-to-delete", models.ExecutionTypeTemplate, models.ExecutionStatusSuccess)
	seedExecution(t, env, "bd2", "still-running", models.ExecutionTypeTemplate, models.ExecutionStatusRunning)

	t.Run("success_skips_running_operType_batch", func(t *testing.T) {
		w := netPost(t, "/executions/batch-delete", h.BatchDelete, `{"executionIds":["bd1","bd2"]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		// BatchDeleteExecutions swallows per-item failures (continue) — count = all IDs
		assert.Contains(t, string(resp.Data), `"count":2`)
		assert.Equal(t, 16, env.oper.lastBusinessType) // OperTypeBatch

		var remaining int64
		env.db.Model(&models.ConfigExecution{}).Where("id IN ?", []string{"bd1", "bd2"}).Count(&remaining)
		assert.Equal(t, int64(1), remaining, "only the non-running row is deleted")
	})

	t.Run("empty_ids_binding_400", func(t *testing.T) {
		w := netPost(t, "/executions/batch-delete", h.BatchDelete, `{"executionIds":[]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}
