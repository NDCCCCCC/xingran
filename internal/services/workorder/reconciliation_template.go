package workorder

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ReconciliationWorkorderTemplate 资产对账工单模板(Phase 43 R2 / D-A2-01~D-A2-04)
//
// 业务背景:
//   - Phase 42 R1 已 seed 6 个 sys_workorder_category(对账-A/B/C/D/E/F 类),R2 在此基础上
//     为每类工单定义差异化模板(标题前缀 / 描述建议 / 默认优先级 / 默认 assignee role)
//   - A 类不入主表(物理+责任人有且一致),仅 B-F 5 类实际触发转单
//   - type→role 映射从 sys_config:asset.reconciliation.workorder.assignee_role_map 读取
//     (D-A2-02),运维可在 admin 页调整,无需改代码
//
// 与 Phase 42 R1 boundary:
//   - 本文件仅定义模板常量 + GetTemplate 查找函数,不涉及 cron 调度与转单执行;
//     cron 调度见 internal/scheduler/reconciliation_tasks.go 续写的 2 个 case;
//     转单执行见 internal/services/asset/reconciliation_workorder.go CreateWorkorderFromException。
//
// Status Convention: WorkOrderPriority 0=Low / 1=Medium / 2=High / 3=Urgent (models.WorkOrderPriority)。
// 0=启用 1=停用 不适用于 priority 字段。
type ReconciliationWorkorderTemplate struct {
	// ConflictType 取值字典 asset_reconciliation_conflict_type(A/B/C/D/E/F)
	ConflictType string
	// TypePrefix 标题前缀(如 "对账-B类"),用于 D-A2-01 标题模板
	TypePrefix string
	// AssigneeRoleKey 期望分派给的角色 key(B/C/D="asset_owner", E="ops_owner", F="responsible_owner")
	// 实际 role_id 从 sys_config JSONB 反查,运维可改
	AssigneeRoleKey string
	// DescriptionLines 5 句中文建议,写入工单 description 的"【建议修复】"段
	DescriptionLines []string
	// DefaultPriority 工单默认优先级(D-A2-03 SLA 关联)
	DefaultPriority models.WorkOrderPriority
}

// 5 个 B-F 模板常量(Type A 不入主表,不需模板)
var (
	// TemplateBType 物理无(未采集)/责任人有 — 资产未上线或采集缺失(D-A2-04 描述)
	TemplateBType = ReconciliationWorkorderTemplate{
		ConflictType:    "B",
		TypePrefix:      "对账-B类",
		AssigneeRoleKey: "asset_owner",
		DescriptionLines: []string{
			"1. 在 ops_asset 上补充责任人(检查 sys_user.deleted_at)",
			"2. 检查 sys_user.status 是否被禁用(status=1)",
			"3. 排查 ops_asset.user_id 是否指向已软删用户",
			"4. 核查工位 sys_workstation.user_id 是否指向已离职账号",
			"5. 修复后请把 ops_asset.user_id 更新为当前在职人员",
		},
		DefaultPriority: models.WorkOrderPriorityHigh, // 2 = 高,B 类转单需要 owner 介入
	}

	// TemplateCType 物理有/责任人无 — 资产已采集但未分配责任人
	TemplateCType = ReconciliationWorkorderTemplate{
		ConflictType:    "C",
		TypePrefix:      "对账-C类",
		AssigneeRoleKey: "asset_owner",
		DescriptionLines: []string{
			"1. 核对物理使用人(端口 MAC 反查真实持有人)",
			"2. 确认 ops_asset.user_id 是否过期或未设置",
			"3. 通过 sys_user.username 在人事系统核对人员状态",
			"4. 若人员仍在职,在 ops_asset.user_id 上补充当前用户",
			"5. 若人员已离职,标记 ops_asset.status=1(停用)并归档",
		},
		DefaultPriority: models.WorkOrderPriorityHigh, // 2 = 高,C 类转单需明确责任人
	}

	// TemplateDType 物理有/责任人有但不一致 — 责任人变更未生效或工位调岗
	TemplateDType = ReconciliationWorkorderTemplate{
		ConflictType:    "D",
		TypePrefix:      "对账-D类",
		AssigneeRoleKey: "asset_owner",
		DescriptionLines: []string{
			"1. 确认资产是否仍在用(查最近 30 天 sys_device_mac_history)",
			"2. 若仍在用,在 ops_asset.user_id 上同步到物理使用人",
			"3. 同步更新 sys_workstation.user_id(若资产绑定工位)",
			"4. 通知原责任人变更结果,确认无需后续资产交割",
			"5. 若资产已无物理使用,标记 ops_asset.status=1(停用)并归档",
		},
		DefaultPriority: models.WorkOrderPriorityMedium, // 1 = 中,D 类转单通常可批量处理
	}

	// TemplateEType 三方(物理/责任人/AD)互不一致 — 重大异常需人工核查
	TemplateEType = ReconciliationWorkorderTemplate{
		ConflictType:    "E",
		TypePrefix:      "对账-E类",
		AssigneeRoleKey: "ops_owner",
		DescriptionLines: []string{
			"1. 反查 sys_workstation 是否有该资产绑定(端口表 sys_port_mac)",
			"2. 若工位存在但未绑资产:补充 ops_asset.workstation_id 关联",
			"3. 若工位已下线:在 ops_asset 上标记废弃状态",
			"4. 同步 AD 域控信息(检查 sys_user_ad_attrs.is_enabled)",
			"5. 三方一致后,在 sys_data_reconciliation 标记 resolved_at",
		},
		DefaultPriority: models.WorkOrderPriorityLow, // 0 = 低,E 类通常是历史遗留,优先级最低
	}

	// TemplateFType 缺数据 — 资产或工位任一端基础数据缺失
	TemplateFType = ReconciliationWorkorderTemplate{
		ConflictType:    "F",
		TypePrefix:      "对账-F类",
		AssigneeRoleKey: "responsible_owner",
		DescriptionLines: []string{
			"1. 检查 AD is_enabled 是否为 false(sys_user_ad_attrs)",
			"2. 同步 sys_user.status=0(启用)与 AD 一致",
			"3. 若 AD 账号已停用:在 sys_user 上同步停用状态",
			"4. 同步 ops_asset.user_id 关联到正确 sys_user",
			"5. 修复后刷新 reconciliation_normalized MV 验证一致",
		},
		DefaultPriority: models.WorkOrderPriorityMedium, // 1 = 中,F 类同步即可
	}
)

// reconciliationTemplatesByType 模板索引(用于 O(1) 查询)
var reconciliationTemplatesByType = map[string]*ReconciliationWorkorderTemplate{
	"B": &TemplateBType,
	"C": &TemplateCType,
	"D": &TemplateDType,
	"E": &TemplateEType,
	"F": &TemplateFType,
}

// GetTemplate 根据 ConflictType 查找模板。
//
// 返回值:
//   - 非 nil 指针: B/C/D/E/F 5 类模板
//   - nil: A 类(健康无需动作)或未知类型 / 空字符串
//
// 调用方在 nil 时应跳过转单并 log 警告(D-A1-03 失败仅 logrus)。
//
// 设计考量:
//   - 用 map 索引而非 switch,便于后续 phase(R3 例外规则、R5 半自动修复)动态扩展类型
//   - ConflictType 是字典 asset_reconciliation_conflict_type 的 value,大小写敏感
func GetTemplate(conflictType string) *ReconciliationWorkorderTemplate {
	if t, ok := reconciliationTemplatesByType[conflictType]; ok {
		return t
	}
	return nil
}

// AllTemplates 返回所有模板(供 admin UI 展示用)。
//
// 返回 B/C/D/E/F 5 个模板,顺序按 ConflictType 字典序(便于前端 grid 渲染)。
func AllTemplates() []*ReconciliationWorkorderTemplate {
	return []*ReconciliationWorkorderTemplate{
		&TemplateBType,
		&TemplateCType,
		&TemplateDType,
		&TemplateEType,
		&TemplateFType,
	}
}