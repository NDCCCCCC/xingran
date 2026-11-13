package operations

// excel_export_dict_fallback_test.go —— 导出侧**数据字典兜底**回归测试。
//
// 背景：ExcelExportField.Options 是导出模板的静态快照，管理员在字典管理页
// 新增枚举项（如工位类型「虚拟工位」）后 Options 不含新值 → 导出列显示原始数字。
// 修复：ExcelExportField 增加 DictType 字段，Options 未命中时按字典反查 dictLabel。

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sysmodels "github.com/xingran-next/xingran-go-backend/internal/models"
)

// TestFormatExportValue_DictFallback 导出侧：Options 未命中时按字典反查 dictLabel。
func TestFormatExportValue_DictFallback(t *testing.T) {
	db := newCRUDTestDB(t)
	require.NoError(t, db.AutoMigrate(&sysmodels.DictData{}))
	require.NoError(t, db.Create(&sysmodels.DictData{
		DictType:  "ops_workstation_type",
		DictLabel: "虚拟工位",
		DictValue: "3",
		Status:    0,
	}).Error)

	svc := NewExcelExportService(db).(*excelExportServiceImpl)
	col := ExcelExportField{
		Field:    "workstationType",
		DBField:  "workstation_type",
		DictType: "ops_workstation_type",
		Options: map[interface{}]string{
			0: "固定工位",
			1: "灵活工位",
			2: "管理工位",
		},
	}

	// 字典新增项 → 导出可读标签
	got := svc.formatExportCellValue(col, map[string]any{"workstation_type": 3})
	assert.Equal(t, "虚拟工位", got)

	// 静态 Options 命中 → 不受影响
	got = svc.formatExportCellValue(col, map[string]any{"workstation_type": 1})
	assert.Equal(t, "灵活工位", got)

	// 字典无对应项 → 回退原始值
	got = svc.formatExportCellValue(col, map[string]any{"workstation_type": 99})
	assert.Equal(t, "99", got)
}

// TestFormatExportValue_Int64Fallback 导出侧 int64 值（PG 常见）也能命中字典。
func TestFormatExportValue_Int64Fallback(t *testing.T) {
	db := newCRUDTestDB(t)
	require.NoError(t, db.AutoMigrate(&sysmodels.DictData{}))
	require.NoError(t, db.Create(&sysmodels.DictData{
		DictType:  "ops_workstation_type",
		DictLabel: "虚拟工位",
		DictValue: "3",
		Status:    0,
	}).Error)

	svc := NewExcelExportService(db).(*excelExportServiceImpl)
	col := ExcelExportField{
		Field:    "workstationType",
		DBField:  "workstation_type",
		DictType: "ops_workstation_type",
		Options:  map[interface{}]string{0: "固定工位"},
	}
	got := svc.formatExportCellValue(col, map[string]any{"workstation_type": int64(3)})
	assert.Equal(t, "虚拟工位", got)
}
