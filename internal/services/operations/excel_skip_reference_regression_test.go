package operations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sysmodels "github.com/xingran-next/xingran-go-backend/internal/models"
)

// TestExcelImport_SkipReferenceColumn_NotBreakOtherReferences 回归测试：
//
// 工位导入填「主设备序列号」后全部报「引用的 所属楼宇 不存在」的根因守护。
//
// 机制：deviceSerial 列的 Reference 是哨兵值 ReferenceSkip("skip")，
// 表示该列不参与引用解析（它由 post-import hook 写入 ops_workstation_device 子表）。
// 历史上该哨兵值被当作真实引用配置送进 resolver → parseReference("skip") 解析失败
// → ResolveBatch 直接 return nil, err → **整批**引用（楼宇/部门/用户）结果被丢弃
// → 楼宇 ID 为空 → 楼层依赖解析连带失败 → 报「所属楼宇 不存在」。
//
// 本测试锁定三件事：
//  1. extractReferenceRequests 不产出 skip 列的引用请求；
//  2. 混入 skip 引用请求时，ResolveBatch 仍能解析出其余合法引用（不中断整批）；
//  3. 端到端：填了主设备序列号的工位行能成功导入。
func TestExcelImport_SkipReferenceColumn_NotBreakOtherReferences(t *testing.T) {
	cfg, ok := GetExcelConfig("workstation")
	require.True(t, ok)

	// 1. 工位配置里的主设备序列号列必须使用哨兵值
	var serialCol *ExcelColumn
	for i := range cfg.Columns {
		if cfg.Columns[i].Field == "deviceSerial" {
			serialCol = &cfg.Columns[i]
		}
	}
	require.NotNil(t, serialCol, "工位配置应含 deviceSerial 列")
	assert.Equal(t, ReferenceSkip, serialCol.Reference, "deviceSerial 应使用 ReferenceSkip 哨兵值")
	assert.True(t, serialCol.isSkipReference())

	t.Run("extractReferenceRequests 不产出 skip 请求", func(t *testing.T) {
		svc := NewExcelService(nil, nil, nil, nil)
		data := map[string]any{
			"name":         "W-001",
			"buildingName": "真实大楼",
			"floorName":    "1楼",
			"deviceSerial": "SN-ABC-001",
		}
		reqs := svc.extractReferenceRequests(data, cfg)

		for _, r := range reqs {
			assert.NotEqual(t, ReferenceSkip, r.Reference, "skip 哨兵不应进入引用请求列表")
		}
		// 其余引用仍应正常收集
		var hasBuilding bool
		for _, r := range reqs {
			if r.Reference == "ops_buildings.name" && r.Value == "真实大楼" {
				hasBuilding = true
			}
		}
		assert.True(t, hasBuilding, "所属楼宇引用仍应被收集")
	})

	t.Run("混入 skip 引用时 ResolveBatch 不中断整批", func(t *testing.T) {
		db := setupImportRest77DB(t)
		require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name, org_id, level) VALUES ('b1', '真实大楼', 'D1', 2)`).Error)

		resolver := NewReferenceResolver(db)
		results, err := resolver.ResolveBatch(context.Background(), []ReferenceRequest{
			{Reference: ReferenceSkip, Value: "SN-ABC-001"},
			{Reference: "ops_buildings.name", Value: "真实大楼"},
		})
		// 无效分组被跳过，但合法引用必须解析成功
		require.NotNil(t, results, "跳过无效分组后仍应返回已解析的部分结果")
		assert.Equal(t, "b1", results["ops_buildings.name:真实大楼"],
			"合法引用不应被 skip 哨兵连累")
		if err != nil {
			assert.Contains(t, err.Error(), "skip", "错误信息应指出无效的引用配置")
		}
	})

	t.Run("端到端：填主设备序列号仍能解析楼宇与楼层", func(t *testing.T) {
		db := setupImportRest77DB(t)
		require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name, org_id, level) VALUES ('b1', '真实大楼', 'D1', 2)`).Error)
		require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, floor_no, building_id) VALUES ('f1', '1楼', '1', 'b1')`).Error)
		require.NoError(t, db.AutoMigrate(&sysmodels.Workstation{}))
		require.NoError(t, db.Exec(`CREATE UNIQUE INDEX ux_test_ws_skip_floor_name ON sys_workstation(floor_id, workstation_name)`).Error)

		svc := NewExcelService(db, nil, nil, nil)

		headers := []string{"工位名称", "所属楼宇", "所属楼层名称", "主设备序列号"}
		rows := [][]string{
			{"W-001", "真实大楼", "1楼", "SN-ABC-001"},
		}
		file := createWorkstationExcelFile(t, headers, rows)

		res, err := svc.ImportData(context.Background(), "workstation", file, "u1")
		require.NoError(t, err)
		assert.Equal(t, 0, res.Failed, "填了主设备序列号不应导致引用解析失败: %+v", res.Errors)
	})
}
