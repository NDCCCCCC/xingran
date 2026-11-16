package operations

// excel_line_type_drift_test.go —— 专线类型标签双侧一致性回归（handoff R8）。
//
// 背景：导出侧 excel_export_config.go 的 mpls/leased_line 文案曾漂移为
// "MPLS专线"/"租用线路"，与导入侧及 migration_208_dict_seed 的
// ops_dedicated_line_type 字典（"MPLS VPN"/"租用专线"）不一致——同一枚举
// 导入导出显示不同文案。2026-11-16 修复后双侧均对齐字典文案并挂 DictType
// 走字典兜底。本测试锁定双侧静态 Options 与字典种子逐字一致，防再漂移。

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dictSeededLineTypes 与 internal/core/db/migrations/migration_208_dict_seed.go
// 的 ops_dedicated_line_type 六值逐字一致（该种子文件注释明确「专线六值与
// excel_config lineType Options 逐字一致」为导入链路硬约束）。
var dictSeededLineTypes = map[interface{}]string{
	"internet":      "互联网专线",
	"intranet":      "内网专线",
	"cloud_desktop": "云桌面专线",
	"mpls":          "MPLS VPN",
	"fiber":         "光纤专线",
	"leased_line":   "租用专线",
}

func findExcelColumnOptionMap(t *testing.T) map[interface{}]string {
	t.Helper()
	cfg, ok := ExcelConfigs["dedicatedLine"]
	require.True(t, ok, "导入配置必须含 dedicatedLine 实体")
	for _, col := range cfg.Columns {
		if col.Field == "lineType" {
			require.NotNil(t, col.Options, "导入侧 lineType 必须提供 Options 快照")
			require.Equal(t, "ops_dedicated_line_type", col.DictType,
				"导入侧 lineType 必须挂 DictType 走字典兜底")
			return col.Options
		}
	}
	t.Fatal("导入配置中未找到 lineType 列")
	return nil
}

func findExcelExportField(t *testing.T) ExcelExportField {
	t.Helper()
	cfg, ok := GetExportConfig("dedicatedLine")
	require.True(t, ok, "导出配置必须含 dedicatedLine 实体")
	for _, col := range cfg.Columns {
		if col.Field == "lineType" {
			require.Equal(t, "ops_dedicated_line_type", col.DictType,
				"导出侧 lineType 必须挂 DictType 走字典兜底")
			return col
		}
	}
	t.Fatal("导出配置中未找到 lineType 列")
	return ExcelExportField{}
}

// TestExcelLineType_NoDrift 锁定导入/导出双侧 lineType Options 与字典种子逐字一致。
func TestExcelLineType_NoDrift(t *testing.T) {
	importOpts := findExcelColumnOptionMap(t)
	assert.Equal(t, dictSeededLineTypes, importOpts,
		"导入侧 lineType Options 与字典种子漂移（对照 migration_208_dict_seed.go）")

	exportField := findExcelExportField(t)
	assert.Equal(t, dictSeededLineTypes, exportField.Options,
		"导出侧 lineType Options 与字典种子漂移（对照 migration_208_dict_seed.go）")
}
