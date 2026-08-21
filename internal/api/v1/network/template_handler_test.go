package network

// TemplateHandler tests (Phase 74-03).
//
// TemplateHandler takes the CONCRETE *services.TemplateService → real service +
// glebarez sqlite (D-02). Preview exercises the real utils.TemplateEngine render
// path; operlog assertions follow D-03 (module title "命令模板").

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
)

func newTemplateHandler(env *netTestEnv) *TemplateHandler {
	return NewTemplateHandler(services.NewTemplateService(env.db)).WithCore(env.core)
}

func newTemplateTestEnv(t *testing.T) *netTestEnv {
	return newNetworkTestEnv(t, &models.ConfigTemplate{}, &models.ConfigExecution{})
}

func seedTemplate(t *testing.T, env *netTestEnv, id, name, code string, isSystem bool, content string) *models.ConfigTemplate {
	t.Helper()
	tpl := &models.ConfigTemplate{
		BaseModel:      models.BaseModel{ID: id},
		TemplateName:   name,
		TemplateCode:   code,
		TemplateType:   models.TemplateTypeConfig,
		Vendor:         models.DeviceVendor("huawei"),
		DeviceType:     models.DeviceType("switch"),
		TemplateContent: content,
		IsSystem:       isSystem,
	}
	require.NoError(t, env.db.Create(tpl).Error)
	return tpl
}

func TestTemplateHandler_Statistics(t *testing.T) {
	env := newTemplateTestEnv(t)
	h := newTemplateHandler(env)

	t.Run("success_counts", func(t *testing.T) {
		seedTemplate(t, env, "t1", "tpl-1", "CODE1", false, "vlan {{.Vlan}}")
		seedTemplate(t, env, "t2", "tpl-2", "CODE2", false, "hostname {{.Name}}")
		sys := seedTemplate(t, env, "t3", "tpl-3", "CODE3", true, "init")
		// make t3 an init template for the init counter
		require.NoError(t, env.db.Model(sys).Update("template_type", models.TemplateTypeInit).Error)

		w := netPost(t, "/templates/statistics", h.Statistics, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.JSONEq(t, `{"total":3,"system":1,"custom":2,"init":1}`, string(resp.Data))
	})

	t.Run("empty_table_zeros", func(t *testing.T) {
		freshEnv := newTemplateTestEnv(t)
		w := netPost(t, "/templates/statistics", newTemplateHandler(freshEnv).Statistics, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"total":0`)
	})
}

func TestTemplateHandler_List(t *testing.T) {
	env := newTemplateTestEnv(t)
	h := newTemplateHandler(env)
	seedTemplate(t, env, "t1", "huawei-vlan", "HW_VLAN", false, "vlan {{.Vlan}}")
	seedTemplate(t, env, "t2", "cisco-vlan", "CS_VLAN", false, "vlan {{.Vlan}}")
	seedTemplate(t, env, "t3", "system-init", "SYS_INIT", true, "init")

	t.Run("filter_by_name", func(t *testing.T) {
		w := netPost(t, "/templates/list", h.List, `{"current":1,"pageSize":10,"templateName":"vlan"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"total":2`)
	})

	t.Run("filter_by_vendor_and_deviceType", func(t *testing.T) {
		w := netPost(t, "/templates/list", h.List,
			`{"current":1,"pageSize":10,"vendor":"huawei","deviceType":"switch"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"total":3`)
	})

	t.Run("filter_isSystem_false_excludes_system", func(t *testing.T) {
		w := netPost(t, "/templates/list", h.List, `{"current":1,"pageSize":10,"isSystem":false}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"total":2`)
		assert.NotContains(t, string(resp.Data), "system-init")
	})

	t.Run("bad_body_defaults", func(t *testing.T) {
		w := netPost(t, "/templates/list", h.List, `not-json`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"total":3`)
		assert.Contains(t, string(resp.Data), `"current":1`)
	})
}

func TestTemplateHandler_GetByID(t *testing.T) {
	env := newTemplateTestEnv(t)
	h := newTemplateHandler(env)
	seedTemplate(t, env, "t9", "find-me", "FIND_ME", false, "content")

	t.Run("found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/templates/:id", h.GetByID}},
			http.MethodPost, "/templates/t9", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), "find-me")
	})

	t.Run("not_found_1500", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/templates/:id", h.GetByID}},
			http.MethodPost, "/templates/none", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, 1500, resp.Code)
	})
}

func TestTemplateHandler_Create(t *testing.T) {
	env := newTemplateTestEnv(t)
	h := newTemplateHandler(env)

	t.Run("success_records_operlog", func(t *testing.T) {
		w := netPost(t, "/templates", h.Create,
			`{"TemplateName":"new-tpl","TemplateCode":"NEW_TPL","TemplateType":"config","Vendor":"huawei","DeviceType":"switch","TemplateContent":"vlan {{.Vlan}}"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), "new-tpl")

		var stored models.ConfigTemplate
		require.NoError(t, env.db.Where("template_code = ?", "NEW_TPL").First(&stored).Error)
		assert.Equal(t, "user-0001", stored.CreatedBy)
		assert.Equal(t, 1, env.oper.lastBusinessType) // OperTypeCreate
		assert.Equal(t, "命令模板", env.oper.lastTitle)
	})

	t.Run("duplicate_code_rejected", func(t *testing.T) {
		seedTemplate(t, env, "dup", "dup-tpl", "DUP", false, "x")
		w := netPost(t, "/templates", h.Create,
			`{"TemplateName":"other","TemplateCode":"DUP","TemplateContent":"x"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "模板编码已存在")
	})

	t.Run("syntax_error_rejected", func(t *testing.T) {
		w := netPost(t, "/templates", h.Create,
			`{"TemplateName":"bad","TemplateCode":"BAD_TPL","TemplateContent":"{{.Vlan"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "模板语法错误")
	})

	t.Run("malformed_json_400", func(t *testing.T) {
		w := netPost(t, "/templates", h.Create, `{bad`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestTemplateHandler_Update(t *testing.T) {
	env := newTemplateTestEnv(t)
	h := newTemplateHandler(env)
	seedTemplate(t, env, "u1", "update-me", "UPD", false, "old")
	seedTemplate(t, env, "u2", "system-tpl", "SYS", true, "locked")

	t.Run("success", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/templates/:id/update", h.Update}},
			http.MethodPost, "/templates/u1/update",
			`{"TemplateName":"renamed","TemplateType":"config","TemplateContent":"new-content"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), "更新成功")

		var stored models.ConfigTemplate
		require.NoError(t, env.db.Where("id = ?", "u1").First(&stored).Error)
		assert.Equal(t, "renamed", stored.TemplateName)
		assert.Equal(t, "new-content", stored.TemplateContent)
		assert.Equal(t, 2, env.oper.lastBusinessType) // OperTypeUpdate
	})

	t.Run("system_template_immutable", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/templates/:id/update", h.Update}},
			http.MethodPost, "/templates/u2/update", `{"TemplateName":"x","TemplateContent":"y"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "系统内置模板不允许修改")
	})

	t.Run("not_found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/templates/:id/update", h.Update}},
			http.MethodPost, "/templates/ghost/update", `{"TemplateName":"x","TemplateContent":"y"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "模板不存在")
	})
}

func TestTemplateHandler_Delete(t *testing.T) {
	env := newTemplateTestEnv(t)
	h := newTemplateHandler(env)

	t.Run("success_operType_delete", func(t *testing.T) {
		seedTemplate(t, env, "d1", "del-me", "DEL", false, "x")
		w := netServe(t, []netRoute{{http.MethodPost, "/templates/:id/delete", h.Delete}},
			http.MethodPost, "/templates/d1/delete", "")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 3, env.oper.lastBusinessType)
		var count int64
		env.db.Model(&models.ConfigTemplate{}).Where("id = ?", "d1").Count(&count)
		assert.Zero(t, count)
	})

	t.Run("system_template_undeletable", func(t *testing.T) {
		seedTemplate(t, env, "d2", "sys-del", "SYS_DEL", true, "x")
		w := netServe(t, []netRoute{{http.MethodPost, "/templates/:id/delete", h.Delete}},
			http.MethodPost, "/templates/d2/delete", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "系统内置模板不允许删除")
	})
}

func TestTemplateHandler_BatchDelete(t *testing.T) {
	env := newTemplateTestEnv(t)
	h := newTemplateHandler(env)
	seedTemplate(t, env, "b1", "batch-1", "B1", false, "x")
	seedTemplate(t, env, "b2", "batch-2", "B2", false, "x")

	t.Run("success_operType_batch", func(t *testing.T) {
		w := netPost(t, "/templates/batch-delete", h.BatchDelete, `{"ids":["b1","b2"]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(resp.Data), `"count":2`)
		assert.Equal(t, 16, env.oper.lastBusinessType) // OperTypeBatch
	})

	t.Run("empty_ids_400", func(t *testing.T) {
		w := netPost(t, "/templates/batch-delete", h.BatchDelete, `{"ids":[]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

func TestTemplateHandler_Preview(t *testing.T) {
	env := newTemplateTestEnv(t)
	h := newTemplateHandler(env)
	// Seed a template with one required variable rendered via the real engine.
	tpl := &models.ConfigTemplate{
		BaseModel:       models.BaseModel{ID: "p1"},
		TemplateName:    "preview-tpl",
		TemplateCode:    "PREVIEW",
		TemplateType:    models.TemplateTypeConfig,
		TemplateContent: "vlan {{.Vlan}}",
		Variables: models.TemplateVariables{
			{Name: "Vlan", Required: true, Type: "string"},
		},
	}
	require.NoError(t, env.db.Create(tpl).Error)

	t.Run("success_renders_with_variables", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/templates/:id/preview", h.Preview}},
			http.MethodPost, "/templates/p1/preview", `{"Vlan":"100"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.JSONEq(t, `{"content":"vlan 100"}`, string(resp.Data))
	})

	t.Run("missing_required_variable", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/templates/:id/preview", h.Preview}},
			http.MethodPost, "/templates/p1/preview", `{}`)
		resp := decodeNetResp(t, w)
		// apperrors.InternalServerError masks the specific cause with a generic message
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, 1500, resp.Code)
		assert.Equal(t, "服务器内部错误", resp.Message)
	})

	t.Run("template_not_found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/templates/:id/preview", h.Preview}},
			http.MethodPost, "/templates/none/preview", `{"Vlan":"1"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, 1500, resp.Code)
	})
}

func TestTemplateHandler_Clone(t *testing.T) {
	env := newTemplateTestEnv(t)
	h := newTemplateHandler(env)
	seedTemplate(t, env, "c1", "origin", "ORIGIN", true, "vlan {{.Vlan}}")

	t.Run("success_creates_non_system_copy", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/templates/:id/clone", h.Clone}},
			http.MethodPost, "/templates/c1/clone", `{"newName":"origin-copy","newCode":"ORIGIN_COPY"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), "origin-copy")

		var cloned models.ConfigTemplate
		require.NoError(t, env.db.Where("template_code = ?", "ORIGIN_COPY").First(&cloned).Error)
		assert.False(t, cloned.IsSystem, "clone of a system template must be custom")
		assert.Equal(t, "vlan {{.Vlan}}", cloned.TemplateContent)
		assert.Equal(t, "user-0001", cloned.CreatedBy)
		assert.Equal(t, 1, env.oper.lastBusinessType) // Clone → OperTypeCreate
	})

	t.Run("binding_requires_newName_newCode", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/templates/:id/clone", h.Clone}},
			http.MethodPost, "/templates/c1/clone", `{"newName":"x"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("duplicate_new_code", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/templates/:id/clone", h.Clone}},
			http.MethodPost, "/templates/c1/clone", `{"newName":"copy2","newCode":"ORIGIN"}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 500, resp.Code)
		assert.Contains(t, resp.Message, "模板编码已存在")
	})
}

func TestTemplateHandler_GetVariables(t *testing.T) {
	env := newTemplateTestEnv(t)
	h := newTemplateHandler(env)
	tpl := &models.ConfigTemplate{
		BaseModel:       models.BaseModel{ID: "v1"},
		TemplateName:    "vars-tpl",
		TemplateCode:    "VARS",
		TemplateType:    models.TemplateTypeConfig,
		TemplateContent: "{{.A}}{{.B}}",
		Variables: models.TemplateVariables{
			{Name: "A", Required: true, Type: "string"},
			{Name: "B", DefaultValue: "b", Type: "string"},
		},
	}
	require.NoError(t, env.db.Create(tpl).Error)

	t.Run("found_returns_definitions", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/templates/:id/variables", h.GetVariables}},
			http.MethodPost, "/templates/v1/variables", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), `"name":"A"`)
		assert.Contains(t, string(resp.Data), `"defaultValue":"b"`)
	})

	t.Run("not_found", func(t *testing.T) {
		w := netServe(t, []netRoute{{http.MethodPost, "/templates/:id/variables", h.GetVariables}},
			http.MethodPost, "/templates/none/variables", "")
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, 1500, resp.Code)
	})
}
