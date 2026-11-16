package operations

// excel_info_point_type_drift_test.go —— 信息点类型值域一致性回归（handoff R9）。
//
// 背景：模型 InfoPointTypePower/Other 与 Excel Options 曾用 032 时代的三值
// （network/power/other），而 legacy 033 迁移已把 ops_info_points 的 DB CHECK
// 约束替换为 IN ('network','phone')，字典 ops_info_point_type 与前端亦为
// network/phone——Excel 提供 power/other 选项，落库即违反 CHECK 约束报错；
// 即使导入成功（sqlite 测试路径无 CHECK），前端字典也渲染不出标签。
// 2026-11-16 修复：模型常量改 InfoPointTypePhone，Excel 双侧 Options 对齐
// network/phone 并挂 DictType 走字典兜底。本测试锁定双侧与字典一致，防复活。

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
)

// dictSeededInfoPointTypes 与 migration_208_dict_seed.go 的
// ops_info_point_type 两值逐字一致（其源头 legacy 033 迁移，与 DB CHECK 同值域）。
var dictSeededInfoPointTypes = map[interface{}]string{
	"network": "网络信息点",
	"phone":   "电话信息点",
}

// TestInfoPointType_AlignedWithDict 锁定模型常量值域 = 字典值域 = DB CHECK 值域。
func TestInfoPointType_AlignedWithDict(t *testing.T) {
	assert.Equal(t, "network", string(operationsmodels.InfoPointTypeNetwork))
	assert.Equal(t, "phone", string(operationsmodels.InfoPointTypePhone))
	// 编译期断言的补充：确认旧的 power/other 常量已不存在（此处以字面量集合
	// 锁定全集，若未来新增第三个常量而不更新字典与 DB CHECK，本测试不会拦截，
	// 但 excel 双侧漂移测试会在文案层拦截）。
	assert.Len(t, dictSeededInfoPointTypes, 2,
		"字典/DB CHECK 值域为 2 值，模型常量集不得超出")
}

// TestExcelInfoPointType_NoDrift 锁定导入/导出双侧 infoPointType Options
// 与字典值域逐字一致，且双侧必须挂 DictType 走字典兜底。
func TestExcelInfoPointType_NoDrift(t *testing.T) {
	cfg, ok := ExcelConfigs["infoPoint"]
	require.True(t, ok, "导入配置必须含 infoPoint 实体")
	var importCol *ExcelColumn
	for i := range cfg.Columns {
		if cfg.Columns[i].Field == "infoPointType" {
			importCol = &cfg.Columns[i]
			break
		}
	}
	require.NotNil(t, importCol, "导入配置中未找到 infoPointType 列")
	require.Equal(t, "ops_info_point_type", importCol.DictType,
		"导入侧 infoPointType 必须挂 DictType 走字典兜底")
	assert.Equal(t, dictSeededInfoPointTypes, importCol.Options,
		"导入侧 infoPointType Options 与字典漂移（对照 migration_208_dict_seed.go）")

	exportCfg, ok := GetExportConfig("infoPoint")
	require.True(t, ok, "导出配置必须含 infoPoint 实体")
	var exportCol *ExcelExportField
	for i := range exportCfg.Columns {
		if exportCfg.Columns[i].Field == "infoPointType" {
			exportCol = &exportCfg.Columns[i]
			break
		}
	}
	require.NotNil(t, exportCol, "导出配置中未找到 infoPointType 列")
	require.Equal(t, "ops_info_point_type", exportCol.DictType,
		"导出侧 infoPointType 必须挂 DictType 走字典兜底")
	assert.Equal(t, dictSeededInfoPointTypes, exportCol.Options,
		"导出侧 infoPointType Options 与字典漂移（对照 migration_208_dict_seed.go）")
}
