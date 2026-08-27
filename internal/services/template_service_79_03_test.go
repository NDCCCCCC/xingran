package services

// =====================================================================
// Phase 79-03 Task 3/4: template_service.go 全 16 方法测试。
//
// 覆盖目标: template_service.go 0% → ≥70%(基线 166 stmts / 166 unc,
// 79-RESEARCH §2 #12)。
// 变量形态锁定:utils.TemplateEngine 走 text/template(research §2 的
// 「纯字符串替换」描述与实装有出入),占位符 = `{{.name}}`;
// Preview/Render 经 BuildVariablesMap 做必填校验 + 默认值回填 + 类型转换。
// 纪律:7903 后缀 helper、sqlite t.TempDir 文件库、禁 t.Parallel、
// models.TemplateType*/DeviceVendor*/DeviceType* 具名常量。
// =====================================================================

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// newTsv7903 装配 TemplateService + sqlite(t.TempDir 文件库)。
func newTsv7903(t *testing.T) (*TemplateService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "tsv7903.db")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "open sqlite temp db")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.AutoMigrate(&models.ConfigTemplate{}), "auto migrate config template")
	// sys_config_execution 只被 Delete 的使用计数查询消费;ConfigExecution 模型
	// 带 custom Time / DeviceIDList 类型(sqlite 装配成本高),按 sqlite 缺表
	// family pattern 手动建仅含查询所需列的裸表。
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_config_execution (
		id TEXT PRIMARY KEY,
		template_id TEXT,
		deleted_at DATETIME
	)`).Error, "create bare sys_config_execution table")
	return NewTemplateService(db), db
}

// baseReq7903 / withOrder7903 / *Ptr7903 列表请求与指针构造(计划后缀防撞名)。
func baseReq7903(current, pageSize int) base.BaseListRequest {
	return base.BaseListRequest{Current: current, PageSize: pageSize}
}

func withOrder7903(current, pageSize int, col string, isAsc *bool) base.BaseListRequest {
	return base.BaseListRequest{Current: current, PageSize: pageSize, OrderByColumn: col, IsAsc: isAsc}
}

func typePtr7903(v models.TemplateType) *models.TemplateType   { return &v }
func vendorPtr7903(v models.DeviceVendor) *models.DeviceVendor { return &v }
func deviceTypePtr7903(v models.DeviceType) *models.DeviceType { return &v }

// tsv7903Vars 构造变量定义表(必填 iface + 带默认值的 vlan)。
func tsv7903Vars() models.TemplateVariables {
	return models.TemplateVariables{
		{Name: "iface", Description: "接口名", Required: true, Type: "string"},
		{Name: "vlan", Description: "VLAN", DefaultValue: "100", Required: false, Type: "int"},
	}
}

// tsv7903Content 含 `{{.iface}}` / `{{.vlan}}` 占位符的模板正文(text/template 形态)。
func tsv7903Content() string {
	return "interface {{.iface}}\nswitchport access vlan {{.vlan}}\n"
}

// tsv7903Create 以 service 路径建模板,失败即终止测试。
func tsv7903Create(t *testing.T, s *TemplateService, code string, isSystem bool) *models.ConfigTemplate {
	t.Helper()
	tpl, err := s.Create(context.Background(), &CreateTemplateRequest{
		TemplateName:    "模板-" + code,
		TemplateCode:    code,
		TemplateType:    models.TemplateTypeConfig,
		Vendor:          models.VendorHuawei,
		DeviceType:      models.DeviceTypeSwitch,
		TemplateContent: tsv7903Content(),
		Variables:       tsv7903Vars(),
		Description:     "desc-" + code,
		IsSystem:        isSystem,
		CreatedBy:       "creator-7903",
	})
	require.NoError(t, err, "create template %s", code)
	return tpl
}

// TestTsv7903_CreateAndGet 合法创建 → GetByID/GetByCode 双通道读回一致;
// 重复 code → 错误;坏语法 → 错误。
func TestTsv7903_CreateAndGet(t *testing.T) {
	svc, _ := newTsv7903(t)
	ctx := context.Background()

	created := tsv7903Create(t, svc, "tpl-create-01", false)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "模板-tpl-create-01", created.TemplateName)
	assert.Equal(t, models.TemplateTypeConfig, created.TemplateType)
	assert.Equal(t, models.VendorHuawei, created.Vendor)
	assert.Equal(t, models.DeviceTypeSwitch, created.DeviceType)
	assert.False(t, created.IsSystem)
	assert.Equal(t, "creator-7903", created.CreatedBy, "CreatedBy 透传")
	require.Len(t, created.Variables, 2, "变量随模板落库")

	// GetByID 通道
	byID, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, byID.ID)
	assert.Equal(t, created.TemplateCode, byID.TemplateCode)
	assert.Equal(t, created.TemplateContent, byID.TemplateContent)
	assert.Equal(t, created.Variables, byID.Variables, "变量 JSON round-trip 一致")

	// GetByCode 通道
	byCode, err := svc.GetByCode(ctx, created.TemplateCode)
	require.NoError(t, err)
	assert.Equal(t, byID.ID, byCode.ID, "双通道命中同一行")

	// 双通道未命中分支
	_, err = svc.GetByID(ctx, "ghost-id")
	require.ErrorContains(t, err, "查询模板失败")
	_, err = svc.GetByCode(ctx, "ghost-code")
	require.ErrorContains(t, err, "查询模板失败")

	// 重复 code → 明确报错
	_, err = svc.Create(ctx, &CreateTemplateRequest{
		TemplateName:    "重复编码模板",
		TemplateCode:    "tpl-create-01",
		TemplateType:    models.TemplateTypeConfig,
		TemplateContent: "plain",
	})
	require.ErrorContains(t, err, "模板编码已存在")

	// text/template 语法错误 → Create 拦截
	_, err = svc.Create(ctx, &CreateTemplateRequest{
		TemplateName:    "坏语法模板",
		TemplateCode:    "tpl-bad-syntax",
		TemplateType:    models.TemplateTypeConfig,
		TemplateContent: "broken {{.unclosed",
	})
	require.ErrorContains(t, err, "模板语法错误")
}

// TestTsv7903_List_PaginationFilter 预置多行 → 分页 + 五类过滤 + 白名单排序/非法列回退。
func TestTsv7903_List_PaginationFilter(t *testing.T) {
	svc, _ := newTsv7903(t)
	ctx := context.Background()

	type spec struct {
		name   string
		code   string
		ntype  models.TemplateType
		vendor models.DeviceVendor
		dtype  models.DeviceType
		isSys  bool
	}
	specs := []spec{
		{"华为初始化", "tpl-hw-init", models.TemplateTypeInit, models.VendorHuawei, models.DeviceTypeRouter, false},
		{"华为备份", "tpl-hw-backup", models.TemplateTypeBackup, models.VendorHuawei, models.DeviceTypeSwitch, false},
		{"华三配置", "tpl-h3c-config", models.TemplateTypeConfig, models.VendorH3C, models.DeviceTypeRouter, false},
		{"锐捷配置", "tpl-rj-config", models.TemplateTypeConfig, models.VendorRuijie, models.DeviceTypeFirewall, false},
		{"系统内置", "tpl-system", models.TemplateTypeConfig, models.VendorHuawei, models.DeviceTypeSwitch, true},
	}
	for _, sp := range specs {
		_, err := svc.Create(ctx, &CreateTemplateRequest{
			TemplateName:    sp.name,
			TemplateCode:    sp.code,
			TemplateType:    sp.ntype,
			Vendor:          sp.vendor,
			DeviceType:      sp.dtype,
			TemplateContent: tsv7903Content(),
			IsSystem:        sp.isSys,
			CreatedBy:       "creator-7903",
		})
		require.NoError(t, err, "create %s", sp.code)
	}

	// 分页
	list, total, err := svc.List(ctx, &ListTemplateRequest{BaseListRequest: baseReq7903(1, 2)})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	require.Len(t, list, 2)
	list, total, err = svc.List(ctx, &ListTemplateRequest{BaseListRequest: baseReq7903(3, 2)})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, list, 1, "尾页 1 行")

	// 模板名模糊
	_, total, err = svc.List(ctx, &ListTemplateRequest{
		BaseListRequest: baseReq7903(1, 10),
		TemplateName:    strPtr7903("华为"),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total, "template_name LIKE %华为%")

	// 类型过滤
	_, total, err = svc.List(ctx, &ListTemplateRequest{
		BaseListRequest: baseReq7903(1, 10),
		TemplateType:    typePtr7903(models.TemplateTypeConfig),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// 厂商过滤
	_, total, err = svc.List(ctx, &ListTemplateRequest{
		BaseListRequest: baseReq7903(1, 10),
		Vendor:          vendorPtr7903(models.VendorHuawei),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// 设备类型过滤
	_, total, err = svc.List(ctx, &ListTemplateRequest{
		BaseListRequest: baseReq7903(1, 10),
		DeviceType:      deviceTypePtr7903(models.DeviceTypeRouter),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 系统标志过滤
	_, total, err = svc.List(ctx, &ListTemplateRequest{
		BaseListRequest: baseReq7903(1, 10),
		IsSystem:        boolPtr7903(true),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 白名单排序:templateName 升序 → 首行 = 字典序最小模板名
	asc := true
	list, total, err = svc.List(ctx, &ListTemplateRequest{
		BaseListRequest: withOrder7903(1, 10, "templateName", &asc),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	require.Len(t, list, 5)
	assert.Equal(t, "华三配置", list[0].TemplateName, "templateName ASC 首行字典序最小")

	// QUIRK-79-03-G(锁定不修,与 QUIRK-79-02-A/QUIRK-79-03-B 同款):
	// 非法 orderByColumn 走 ApplySort 白名单回退(仅 warn 不报错),又因
	// List 仅在 OrderByColumn=="" 时补默认 created_at DESC → 非法列退化为
	// sqlite 自然序。断言无错误 + 总数正确,不锁具体顺序。
	list, total, err = svc.List(ctx, &ListTemplateRequest{
		BaseListRequest: withOrder7903(1, 10, "1; DROP TABLE sys_config_template", &asc),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, list, 5)

	// 空 orderByColumn → 默认 created_at DESC(最后创建的行在前)
	list, _, err = svc.List(ctx, &ListTemplateRequest{BaseListRequest: baseReq7903(1, 1)})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "系统内置", list[0].TemplateName, "默认排序首行 = 最后创建")
}

// TestTsv7903_Update_Delete_BatchDelete 更新读回 + 系统/不存在保护 + 单删/批删。
func TestTsv7903_Update_Delete_BatchDelete(t *testing.T) {
	svc, db := newTsv7903(t)
	ctx := context.Background()

	custom := tsv7903Create(t, svc, "tpl-update-01", false)
	system := tsv7903Create(t, svc, "tpl-update-sys", true)

	// 更新自定义模板
	newName := "更新后的模板"
	newType := models.TemplateTypeBackup
	newContent := "sysname {{.hostname}}\n"
	newVars := models.TemplateVariables{{Name: "hostname", Required: true, Type: "string"}}
	require.NoError(t, svc.Update(ctx, &UpdateTemplateRequest{
		ID:              custom.ID,
		TemplateName:    newName,
		TemplateType:    newType,
		Vendor:          models.VendorH3C,
		DeviceType:      models.DeviceTypeRouter,
		TemplateContent: newContent,
		Variables:       newVars,
		Description:     "更新后的描述",
		UpdatedBy:       "updater-7903",
	}))
	got, err := svc.GetByID(ctx, custom.ID)
	require.NoError(t, err)
	assert.Equal(t, newName, got.TemplateName)
	assert.Equal(t, newType, got.TemplateType)
	assert.Equal(t, models.VendorH3C, got.Vendor)
	assert.Equal(t, models.DeviceTypeRouter, got.DeviceType)
	assert.Equal(t, newContent, got.TemplateContent)
	assert.Equal(t, newVars, got.Variables)
	assert.Equal(t, "更新后的描述", got.Description)
	assert.Equal(t, "updater-7903", got.UpdatedBy, "updated_by 落库")

	// 系统模板不允许修改 / 删除
	require.ErrorContains(t, svc.Update(ctx, &UpdateTemplateRequest{
		ID:              system.ID,
		TemplateName:    "改系统模板",
		TemplateContent: tsv7903Content(),
	}), "系统内置模板不允许修改")
	require.ErrorContains(t, svc.Delete(ctx, system.ID), "系统内置模板不允许删除")

	// 坏语法更新 → 拦截
	require.ErrorContains(t, svc.Update(ctx, &UpdateTemplateRequest{
		ID:              custom.ID,
		TemplateName:    newName,
		TemplateContent: "bad {{.x",
	}), "模板语法错误")

	// 不存在 → 更新/删除均报"模板不存在"
	require.ErrorContains(t, svc.Update(ctx, &UpdateTemplateRequest{
		ID:              "ghost-id",
		TemplateName:    "x",
		TemplateContent: "plain",
	}), "模板不存在")
	require.ErrorContains(t, svc.Delete(ctx, "ghost-id"), "模板不存在")

	// QUIRK-79-03-H(锁定不修):Delete 先统计 sys_config_execution 使用计数,
	// 但结果未参与任何判断(死代码)—— 有执行记录引用也照删。
	require.NoError(t, db.Exec(
		`INSERT INTO sys_config_execution (id, template_id) VALUES ('exec-1', ?)`, custom.ID,
	).Error)
	require.NoError(t, svc.Delete(ctx, custom.ID))
	var live int64
	require.NoError(t, db.Model(&models.ConfigTemplate{}).Where("id = ?", custom.ID).Count(&live).Error)
	assert.Zero(t, live, "有执行记录引用仍被删除(锁定现行为)")

	// 批删:逐个走 Delete;含系统模板 → 中途报错
	b1 := tsv7903Create(t, svc, "tpl-batch-1", false)
	b2 := tsv7903Create(t, svc, "tpl-batch-2", false)
	require.NoError(t, svc.BatchDelete(ctx, []string{b1.ID, b2.ID}))
	require.NoError(t, db.Model(&models.ConfigTemplate{}).Where("template_code LIKE ?", "tpl-batch-%").Count(&live).Error)
	assert.Zero(t, live, "批删全部生效")
	require.ErrorContains(t, svc.BatchDelete(ctx, []string{system.ID}), "系统内置模板不允许删除")
	require.ErrorContains(t, svc.BatchDelete(ctx, []string{"ghost-id"}), "模板不存在")
}

// TestTsv7903_GetStatistics 多状态/多 vendor 行 → TemplateStatistics 计数与手算一致。
func TestTsv7903_GetStatistics(t *testing.T) {
	svc, _ := newTsv7903(t)
	ctx := context.Background()

	// 2 系统 + 3 自定义;其中 init 类型 2 条
	tsv7903Create(t, svc, "stat-sys-1", true)
	tsv7903Create(t, svc, "stat-sys-2", true)
	for _, code := range []string{"stat-cfg-1", "stat-cfg-2", "stat-cfg-3"} {
		tsv7903Create(t, svc, code, false)
	}
	for _, code := range []string{"stat-init-1", "stat-init-2"} {
		_, err := svc.Create(ctx, &CreateTemplateRequest{
			TemplateName:    "init-" + code,
			TemplateCode:    code,
			TemplateType:    models.TemplateTypeInit,
			TemplateContent: tsv7903Content(),
		})
		require.NoError(t, err, "create %s", code)
	}

	result, err := svc.GetStatistics(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(7), result.Total)
	assert.Equal(t, int64(2), result.System, "is_system = true 计数")
	assert.Equal(t, int64(5), result.Custom, "is_system = false 计数")
	assert.Equal(t, int64(2), result.Init, "template_type = init 计数")
}

// TestTsv7903_GetVariables 预置模板变量行 → 返回列表与序一致。
func TestTsv7903_GetVariables(t *testing.T) {
	svc, _ := newTsv7903(t)
	ctx := context.Background()

	tpl := tsv7903Create(t, svc, "tpl-vars-01", false)

	vars, err := svc.GetVariables(ctx, tpl.ID)
	require.NoError(t, err)
	require.Len(t, vars, 2)
	assert.Equal(t, "iface", vars[0].Name, "变量序与定义一致")
	assert.True(t, vars[0].Required)
	assert.Equal(t, "vlan", vars[1].Name)
	assert.Equal(t, "100", vars[1].DefaultValue, "默认值 round-trip")

	// 不存在模板 → 透传 GetByID 错误
	_, err = svc.GetVariables(ctx, "ghost-id")
	require.ErrorContains(t, err, "查询模板失败")
}

// TestTsv7903_ValidateVariables 必填缺失 → 错误;全给 → nil;
// 多余变量被接受(实装不做白名单外变量拒绝,锁定现行为)。
func TestTsv7903_ValidateVariables(t *testing.T) {
	svc, _ := newTsv7903(t)
	ctx := context.Background()

	tpl := tsv7903Create(t, svc, "tpl-validate-01", false)

	// 必填变量缺失 → RenderWithValidation 报"缺少必需变量"
	err := svc.ValidateVariables(ctx, tpl.ID, map[string]string{"vlan": "200"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "缺少必需变量: iface")

	// 必填 + 可选全给 → 通过
	require.NoError(t, svc.ValidateVariables(ctx, tpl.ID, map[string]string{
		"iface": "GigabitEthernet0/0",
		"vlan":  "200",
	}))

	// 仅给必填(可选走默认值)→ 通过
	require.NoError(t, svc.ValidateVariables(ctx, tpl.ID, map[string]string{
		"iface": "TenGigabitEthernet1/1",
	}))

	// 多余变量:实装不拒绝白名单外变量(锁定现行为)
	require.NoError(t, svc.ValidateVariables(ctx, tpl.ID, map[string]string{
		"iface": "Vlanif10",
		"extra": "not-defined",
	}))

	// 不存在模板 → 透传错误
	require.ErrorContains(t, svc.ValidateVariables(ctx, "ghost-id", nil), "查询模板失败")
}
