package operations

// excel_dict_fallback_test.go —— Excel 导入/导出的**数据字典兜底**回归测试。
//
// 背景：工位类型等枚举是数据字典（ops_workstation_type），管理员可在字典管理页
// 新增枚举项（如「虚拟工位」）。但 excel_config.go 的 Options 是**静态快照**，
// 不含新增项，导致：
//   - 导入：parseFieldValue 未命中 → 原样透传字符串 → 写入整型列产生脏数据/报错；
//   - 导出：formatCellValue 未命中 → 输出原始数字，管理员看不懂。
//
// 修复：ExcelColumn 增加 DictType 字段，绑定字典的列在 Options 未命中时按字典
// 反查（导入查 dictValue、导出查 dictLabel），两侧共用同一字典 → 新增枚举项
// 无需改代码。
//
// 本测试锁定：
//  1. 导入侧：字典新增项（虚拟工位）能解析为整数 dictValue；
//  2. 导入侧：字典中不存在的标签仍按原行为透传（不引入新失败模式）；
//  3. 导出侧：字典新增项导出为可读标签；
//  4. 导出侧：字典无对应项时仍回退原始值；
//  5. 未绑定字典的列行为完全不变（不误伤其它实体）。

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sysmodels "github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// newDictFallbackDB 建带 sys_dict_data 表的测试库。
func newDictFallbackDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newCRUDTestDB(t)
	require.NoError(t, db.AutoMigrate(&sysmodels.DictData{}))
	return db
}

// TestParseFieldValue_DictFallback 导入侧：Options 未命中时按字典反查 dictValue。
func TestParseFieldValue_DictFallback(t *testing.T) {
	db := newDictFallbackDB(t)
	require.NoError(t, db.Create(&sysmodels.DictData{
		DictType:  "ops_workstation_type",
		DictLabel: "虚拟工位",
		DictValue: "3",
		Status:    0,
	}).Error)
	require.NoError(t, db.Create(&sysmodels.DictData{
		DictType:  "ops_workstation_type",
		DictLabel: "停用工位类型",
		DictValue: "9",
		Status:    1, // 停用项不参与匹配
	}).Error)

	svc := NewExcelService(db, nil, nil, nil)
	col := ExcelColumn{
		Field:    "workstationType",
		DBField:  "workstation_type",
		DictType: "ops_workstation_type",
		Options: map[interface{}]string{
			0: "固定工位",
			1: "灵活工位",
			2: "管理工位",
		},
	}

	// 字典新增项 → 解析为整数 3
	got, err := svc.parseFieldValue(col, "虚拟工位")
	require.NoError(t, err)
	assert.Equal(t, 3, got, "字典新增项应解析为 dictValue=3")

	// 静态 Options 命中 → 仍走 Options（不查库）
	got, err = svc.parseFieldValue(col, "固定工位")
	require.NoError(t, err)
	assert.Equal(t, 0, got)

	// 字典中不存在的标签 → 原行为透传字符串（不引入新失败模式）
	got, err = svc.parseFieldValue(col, "不存在的类型")
	require.NoError(t, err)
	assert.Equal(t, "不存在的类型", got)

	// 停用字典项不参与匹配 → 透传
	got, err = svc.parseFieldValue(col, "停用工位类型")
	require.NoError(t, err)
	assert.Equal(t, "停用工位类型", got)
}

// TestParseFieldValue_NoDictType_Unchanged 未绑定字典的列行为完全不变。
func TestParseFieldValue_NoDictType_Unchanged(t *testing.T) {
	db := newDictFallbackDB(t)
	svc := NewExcelService(db, nil, nil, nil)
	col := ExcelColumn{
		Field: "status",
		Options: map[interface{}]string{
			0: "正常",
			1: "停用",
		},
	}
	got, err := svc.parseFieldValue(col, "正常")
	require.NoError(t, err)
	assert.Equal(t, 0, got)

	got, err = svc.parseFieldValue(col, "未知状态")
	require.NoError(t, err)
	assert.Equal(t, "未知状态", got, "未绑定字典的列仍原样透传")
}

// TestFormatCellValue_DictFallback 导出侧：Options 未命中时按字典反查 dictLabel。
func TestFormatCellValue_DictFallback(t *testing.T) {
	db := newDictFallbackDB(t)
	require.NoError(t, db.Create(&sysmodels.DictData{
		DictType:  "ops_workstation_type",
		DictLabel: "虚拟工位",
		DictValue: "3",
		Status:    0,
	}).Error)

	svc := NewExcelService(db, nil, nil, nil)
	col := ExcelColumn{
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
	got := svc.formatCellValue(col, map[string]any{"workstation_type": 3})
	assert.Equal(t, "虚拟工位", got)

	// 静态 Options 命中 → 不受影响
	got = svc.formatCellValue(col, map[string]any{"workstation_type": 1})
	assert.Equal(t, "灵活工位", got)

	// 字典无对应项 → 回退原始值
	got = svc.formatCellValue(col, map[string]any{"workstation_type": 99})
	assert.Equal(t, "99", got)
}

// TestLookupDictValue_DBUnavailable 查库失败（如无表）时不阻断导入。
func TestLookupDictValue_DBUnavailable(t *testing.T) {
	// 构造一个没有 sys_dict_data 表的库
	db := newCRUDTestDB(t)
	svc := NewExcelService(db, nil, nil, nil)
	value, ok := svc.lookupDictValue("ops_workstation_type", "虚拟工位")
	assert.False(t, ok, "查库失败应返回 ok=false")
	assert.Equal(t, 0, value)
}
