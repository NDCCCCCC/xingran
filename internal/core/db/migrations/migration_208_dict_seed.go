package migrations

import (
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate208DictSeed 重建 11 组字典 seed (DICT-02)。
//
// 背景 (Phase 69 / 69-RESEARCH A4):
//   - dev 库 2026-08-17 从 Supabase PG 切到本地 SQLite 后, 全部历史字典 seed
//     只存在于 archive/ 迁移 (不执行) —— sys_dict_type / sys_dict_data 实测 0 行,
//     4 个既有 useDict 消费页 (dedicated-lines / info-points / reconciliation
//     exceptions / HealthBadge) 下拉全部空白。
//   - 本迁移是整条字典链路的数据恢复点: 8 组 archive 存量重建 + 3 组新增
//     (ops_workstation_type / sys_user_sex / duty_holiday_type, 按前端有对应
//     下拉圈定)。仪表盘三组与工单两组不重建 (前端零 useDict 消费)。
//
// 幂等:
//   - 组级快速路径: dict_type 已存在 → 整组跳过 (管理员增删的组不被覆盖,
//     删除的组不被复活 —— 尊重管理员意图)。查重走 Unscoped: 软删的 dict_type
//     同样视为「组已存在」, 否则软删行占位 uniqueIndex 会让每次启动 INSERT
//     撞唯一约束失败 (WARN 刷屏) 且实质构成复活。
//   - 组内不查重: 组级跳过已保证进入事务的组是全新组, 组内无需逐行查重。
//   - 单事务包裹: 组内任一写入失败整组回滚并返回 error, 不留半组数据;
//     由调用方 WARN 不阻断启动, 下次启动重试 (已成功的组被组级跳过)。
//   - 双方言: 纯 GORM 结构体 Create (无裸 SQL / 无 PG 专有语法),
//     database.go 的 PG advisory-lock 分支与 SQLite else 分支各挂载一次。
//
// 值来源 (抄值不改链路, RESEARCH A6/Q3):
//   - network_device_type: archive/legacy-2026-06-15/002 + models.DeviceType 注释
//   - ops_dedicated_line_type / ops_isp / ops_info_point_type:
//     archive/legacy-2026-06-15/{047,048,033}; 专线六值与 excel_config.go
//     lineType Options 逐字一致 (导入链路一致性硬约束)
//   - asset_reconciliation_*: archive/applied/migration_169, 其中 conflict_type
//     的 label/listClass 以 migration_196 对齐 detection 真值后的值为准
//   - ops_workstation_type / sys_user_sex / duty_holiday_type:
//     models.WorkstationType / models.Gender / models.HolidayType (int 常量
//     dictValue 写 "0"/"1"/"2" 字符串; DictValue 模型字段是 string)
//
// isDefault 语义: 每组恰好一条 true (消费端 isDefault 默认值逻辑的数据前提,
// dedicated-lines 三件套依赖)。有 archive/模型来源的照抄 (如 internet/telecom/
// network/low/no_alert/open, 以及模型 gorm default 注释: 工位 "0"、性别 "2"、
// 假日 custom); 无来源的组 (network_device_type / conflict_type, archive 全
// false) 取组内第一条为默认。
//
// Status 一律引用 models.DictStatusNormal (69-01 交付的 DictStatus 常量家族,
// 本迁移硬依赖 69-01, 不写裸 0)。
func Migrate208DictSeed(db *gorm.DB) error {
	for i := range dictSeedGroups {
		group := &dictSeedGroups[i]

		// 组级快速路径: dict_type 已存在 (含软删) → 整组跳过
		var n int64
		if err := db.Unscoped().Model(&models.DictType{}).
			Where("dict_type = ?", group.Type.DictType).
			Count(&n).Error; err != nil {
			return fmt.Errorf("migration 208: 查询字典组 %s 失败: %w", group.Type.DictType, err)
		}
		if n > 0 {
			continue
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&group.Type).Error; err != nil {
				return fmt.Errorf("migration 208: 写入 dict_type %s 失败: %w", group.Type.DictType, err)
			}
			if err := tx.Create(&group.Data).Error; err != nil {
				return fmt.Errorf("migration 208: 写入 dict_data %s (%d 条) 失败: %w",
					group.Type.DictType, len(group.Data), err)
			}
			return nil
		}); err != nil {
			return err
		}
		applogger.Infof("[迁移] migration 208: 字典组 %s seed 完成 (%d 值)",
			group.Type.DictType, len(group.Data))
	}
	return nil
}

// listClass 返回字符串指针 (DictData.ListClass 为 *string; 无来源的组省略字段即 nil)。
func listClass(s string) *string { return &s }

// dictSeedGroups 11 组字典 seed: 8 组 archive 存量重建 + 3 组新增。
// DictSort 按组内顺序 1 起递增; DictData.DictType 回填组 key。
var dictSeedGroups = []struct {
	Type models.DictType
	Data []models.DictData
}{
	{
		Type: models.DictType{
			DictName: "网络设备类型",
			DictType: "network_device_type",
			Status:   int(models.DictStatusNormal),
			Remark:   "网络设备类型映射 (值对齐 models.DeviceType)",
		},
		Data: []models.DictData{
			{DictSort: 1, DictLabel: "路由器", DictValue: "router", DictType: "network_device_type", ListClass: listClass("default"), IsDefault: true, Status: int(models.DictStatusNormal)},
			{DictSort: 2, DictLabel: "交换机", DictValue: "switch", DictType: "network_device_type", ListClass: listClass("default"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 3, DictLabel: "防火墙", DictValue: "firewall", DictType: "network_device_type", ListClass: listClass("default"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 4, DictLabel: "无线接入点", DictValue: "ap", DictType: "network_device_type", ListClass: listClass("default"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 5, DictLabel: "负载均衡器", DictValue: "loadbalancer", DictType: "network_device_type", ListClass: listClass("default"), IsDefault: false, Status: int(models.DictStatusNormal)},
		},
	},
	{
		Type: models.DictType{
			DictName: "专线类型",
			DictType: "ops_dedicated_line_type",
			Status:   int(models.DictStatusNormal),
			Remark:   "专线类型字典：互联网专线、内网专线、云桌面专线等 (六值与 excel_config lineType 逐字一致)",
		},
		Data: []models.DictData{
			{DictSort: 1, DictLabel: "互联网专线", DictValue: "internet", DictType: "ops_dedicated_line_type", ListClass: listClass("primary"), IsDefault: true, Status: int(models.DictStatusNormal)},
			{DictSort: 2, DictLabel: "内网专线", DictValue: "intranet", DictType: "ops_dedicated_line_type", ListClass: listClass("success"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 3, DictLabel: "云桌面专线", DictValue: "cloud_desktop", DictType: "ops_dedicated_line_type", ListClass: listClass("warning"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 4, DictLabel: "MPLS VPN", DictValue: "mpls", DictType: "ops_dedicated_line_type", ListClass: listClass("processing"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 5, DictLabel: "光纤专线", DictValue: "fiber", DictType: "ops_dedicated_line_type", ListClass: listClass("default"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 6, DictLabel: "租用专线", DictValue: "leased_line", DictType: "ops_dedicated_line_type", ListClass: listClass("default"), IsDefault: false, Status: int(models.DictStatusNormal)},
		},
	},
	{
		Type: models.DictType{
			DictName: "运营商",
			DictType: "ops_isp",
			Status:   int(models.DictStatusNormal),
			Remark:   "运营商字典：电信、移动、联通等",
		},
		Data: []models.DictData{
			{DictSort: 1, DictLabel: "电信", DictValue: "telecom", DictType: "ops_isp", ListClass: listClass("primary"), IsDefault: true, Status: int(models.DictStatusNormal)},
			{DictSort: 2, DictLabel: "移动", DictValue: "mobile", DictType: "ops_isp", ListClass: listClass("success"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 3, DictLabel: "联通", DictValue: "unicom", DictType: "ops_isp", ListClass: listClass("warning"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 4, DictLabel: "广电", DictValue: "broadcast", DictType: "ops_isp", ListClass: listClass("processing"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 5, DictLabel: "其他", DictValue: "other", DictType: "ops_isp", ListClass: listClass("default"), IsDefault: false, Status: int(models.DictStatusNormal)},
		},
	},
	{
		Type: models.DictType{
			DictName: "信息点类型",
			DictType: "ops_info_point_type",
			Status:   int(models.DictStatusNormal),
			Remark:   "信息点类型字典：网络信息点、电话信息点",
		},
		Data: []models.DictData{
			{DictSort: 1, DictLabel: "网络信息点", DictValue: "network", DictType: "ops_info_point_type", ListClass: listClass("primary"), IsDefault: true, Status: int(models.DictStatusNormal)},
			{DictSort: 2, DictLabel: "电话信息点", DictValue: "phone", DictType: "ops_info_point_type", ListClass: listClass("success"), IsDefault: false, Status: int(models.DictStatusNormal)},
		},
	},
	{
		Type: models.DictType{
			DictName: "资产对账冲突类型",
			DictType: "asset_reconciliation_conflict_type",
			Status:   int(models.DictStatusNormal),
			Remark:   "资产对账冲突类型 (label/listClass 以 migration_196 对齐 detection 真值后为准)",
		},
		Data: []models.DictData{
			{DictSort: 1, DictLabel: "A类-完全一致", DictValue: "A", DictType: "asset_reconciliation_conflict_type", ListClass: listClass("success"), IsDefault: true, Status: int(models.DictStatusNormal)},
			{DictSort: 2, DictLabel: "B类-物理有/责任人无", DictValue: "B", DictType: "asset_reconciliation_conflict_type", ListClass: listClass("error"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 3, DictLabel: "C类-物理有/责任人不一致(高危)", DictValue: "C", DictType: "asset_reconciliation_conflict_type", ListClass: listClass("error"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 4, DictLabel: "D类-物理无/责任人有", DictValue: "D", DictType: "asset_reconciliation_conflict_type", ListClass: listClass("warning"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 5, DictLabel: "E类-双方都无用户关联", DictValue: "E", DictType: "asset_reconciliation_conflict_type", ListClass: listClass("default"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 6, DictLabel: "F类-物理与责任人一致但 AD 不一致", DictValue: "F", DictType: "asset_reconciliation_conflict_type", ListClass: listClass("warning"), IsDefault: false, Status: int(models.DictStatusNormal)},
		},
	},
	{
		Type: models.DictType{
			DictName: "资产对账例外动作",
			DictType: "asset_reconciliation_exception_action",
			Status:   int(models.DictStatusNormal),
			Remark:   "资产对账例外规则动作",
		},
		Data: []models.DictData{
			{DictSort: 1, DictLabel: "不告警", DictValue: "no_alert", DictType: "asset_reconciliation_exception_action", ListClass: listClass("default"), IsDefault: true, Status: int(models.DictStatusNormal)},
			{DictSort: 2, DictLabel: "不通知", DictValue: "no_notice", DictType: "asset_reconciliation_exception_action", ListClass: listClass("default"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 3, DictLabel: "不转工单", DictValue: "no_workorder", DictType: "asset_reconciliation_exception_action", ListClass: listClass("default"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 4, DictLabel: "跳过严重度", DictValue: "skip_severity", DictType: "asset_reconciliation_exception_action", ListClass: listClass("warning"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 5, DictLabel: "静默期", DictValue: "silence", DictType: "asset_reconciliation_exception_action", ListClass: listClass("default"), IsDefault: false, Status: int(models.DictStatusNormal)},
		},
	},
	{
		Type: models.DictType{
			DictName: "资产对账严重度",
			DictType: "asset_reconciliation_severity",
			Status:   int(models.DictStatusNormal),
			Remark:   "资产对账异常严重度",
		},
		Data: []models.DictData{
			{DictSort: 1, DictLabel: "低", DictValue: "low", DictType: "asset_reconciliation_severity", ListClass: listClass("default"), IsDefault: true, Status: int(models.DictStatusNormal)},
			{DictSort: 2, DictLabel: "中", DictValue: "medium", DictType: "asset_reconciliation_severity", ListClass: listClass("warning"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 3, DictLabel: "高", DictValue: "high", DictType: "asset_reconciliation_severity", ListClass: listClass("error"), IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 4, DictLabel: "紧急", DictValue: "critical", DictType: "asset_reconciliation_severity", ListClass: listClass("error"), IsDefault: false, Status: int(models.DictStatusNormal)},
		},
	},
	{
		Type: models.DictType{
			DictName: "资产对账状态",
			DictType: "asset_reconciliation_status",
			Status:   int(models.DictStatusNormal),
			Remark:   "资产对账异常解决状态",
		},
		Data: []models.DictData{
			{DictSort: 1, DictLabel: "未解决", DictValue: "open", DictType: "asset_reconciliation_status", ListClass: listClass("warning"), IsDefault: true, Status: int(models.DictStatusNormal)},
			{DictSort: 2, DictLabel: "已解决", DictValue: "resolved", DictType: "asset_reconciliation_status", ListClass: listClass("success"), IsDefault: false, Status: int(models.DictStatusNormal)},
		},
	},
	{
		Type: models.DictType{
			DictName: "工位类型",
			DictType: "ops_workstation_type",
			Status:   int(models.DictStatusNormal),
			Remark:   "工位类型 (值对齐 models.WorkstationType 与 excel_config workstationType)",
		},
		Data: []models.DictData{
			{DictSort: 1, DictLabel: "固定工位", DictValue: "0", DictType: "ops_workstation_type", IsDefault: true, Status: int(models.DictStatusNormal)},
			{DictSort: 2, DictLabel: "灵活工位", DictValue: "1", DictType: "ops_workstation_type", IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 3, DictLabel: "管理工位", DictValue: "2", DictType: "ops_workstation_type", IsDefault: false, Status: int(models.DictStatusNormal)},
		},
	},
	{
		Type: models.DictType{
			DictName: "用户性别",
			DictType: "sys_user_sex",
			Status:   int(models.DictStatusNormal),
			Remark:   "用户性别 (值对齐 models.Gender)",
		},
		Data: []models.DictData{
			{DictSort: 1, DictLabel: "男", DictValue: "0", DictType: "sys_user_sex", IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 2, DictLabel: "女", DictValue: "1", DictType: "sys_user_sex", IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 3, DictLabel: "保密", DictValue: "2", DictType: "sys_user_sex", IsDefault: true, Status: int(models.DictStatusNormal)},
		},
	},
	{
		Type: models.DictType{
			DictName: "节假日类型",
			DictType: "duty_holiday_type",
			Status:   int(models.DictStatusNormal),
			Remark:   "节假日类型 (值对齐 models.HolidayType)",
		},
		Data: []models.DictData{
			{DictSort: 1, DictLabel: "法定节假日", DictValue: "legal", DictType: "duty_holiday_type", IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 2, DictLabel: "调休工作日", DictValue: "workday", DictType: "duty_holiday_type", IsDefault: false, Status: int(models.DictStatusNormal)},
			{DictSort: 3, DictLabel: "自定义节假日", DictValue: "custom", DictType: "duty_holiday_type", IsDefault: true, Status: int(models.DictStatusNormal)},
		},
	},
}
