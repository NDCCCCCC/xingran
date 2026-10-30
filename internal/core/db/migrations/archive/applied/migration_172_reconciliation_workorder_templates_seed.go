//go:build archive_skip


package migrations

import (
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate172ReconciliationWorkorderTemplatesSeed Phase 43 R2: 补种 6 个对账 category 的
// description(D-A2-04)
//
// 业务背景:
//   - migration_169 已 seed 6 条 sys_workorder_category(对账-A/B/C/D/E/F 类),
//     但当时 description 是 R1 风格的简短一句话
//   - R2 需要把 B-F 5 类 description 升级为 5 句中文建议(与 workorder/reconciliation_template.go
//     的 DescriptionLines 内容保持一致),作为转单后工单的"【建议修复】"段内容
//   - A 类 description 改为"健康无需动作",语义明确化(之前是"R1 仅作 dashboard 统计")
//
// 6 个 category 的 description:
//   - 对账-A类: "物理+责任人有且一致 健康无需动作"
//   - 对账-B类: 5 句中文(B 类:补责任人+核查转岗)
//   - 对账-C类: 5 句中文(C 类:核对物理使用人+确认 ops_asset.user_id 过期)
//   - 对账-D类: 5 句中文(D 类:确认是否在用+标记废弃)
//   - 对账-E类: 5 句中文(E 类:反查 sys_workstation+补充 ops_asset 或标记下线)
//   - 对账-F类: 5 句中文(F 类:检查 AD is_enabled+同步 sys_user.status)
//
// 幂等: UPDATE 仅在原 description 为空或为 R1 默认值时执行,避免覆盖运维在 admin 页手动
// 修改过的 description。WHERE 条件: description IS NULL OR description = R1 默认值。
//
// 与 reconciliation_template.go 的关联:
//   - 本 migration seed 的 description 与 ReconciliationWorkorderTemplate.DescriptionLines
//     内容完全一致,确保 UI admin 页编辑 description 与代码生成 description 一致
//   - 后续若修改模板,需要同步修改本 migration 的 seed 值 + 重新跑迁移
func Migrate172ReconciliationWorkorderTemplatesSeed(db *gorm.DB) error {
	log.Println("Running migration 172: Phase 43 R2 reconciliation workorder category description backfill")

	// 6 个 category 的 description(B-F 5 句中文 + A 健康说明)
	templates := []struct {
		name        string
		description string
		// r1Default 是 R1 默认 description,作为幂等检查的"未修改"标记
		// 若运维已手动修改 description,跳过 UPDATE(避免覆盖)
		r1Default string
	}{
		{
			name:        "对账-A类",
			description: "物理+责任人有且一致 健康无需动作",
			r1Default:   "物理有/责任人有且一致 健康无需动作(R1 仅作 dashboard 统计,不进 sys_data_reconciliation)",
		},
		{
			name: "对账-B类",
			description: "1. 在 ops_asset 上补充责任人(检查 sys_user.deleted_at)\n" +
				"2. 检查 sys_user.status 是否被禁用(status=1)\n" +
				"3. 排查 ops_asset.user_id 是否指向已软删用户\n" +
				"4. 核查工位 sys_workstation.user_id 是否指向已离职账号\n" +
				"5. 修复后请把 ops_asset.user_id 更新为当前在职人员",
			r1Default: "物理无(未采集)/责任人有 — 资产未上线或采集缺失",
		},
		{
			name: "对账-C类",
			description: "1. 核对物理使用人(端口 MAC 反查真实持有人)\n" +
				"2. 确认 ops_asset.user_id 是否过期或未设置\n" +
				"3. 通过 sys_user.username 在人事系统核对人员状态\n" +
				"4. 若人员仍在职,在 ops_asset.user_id 上补充当前用户\n" +
				"5. 若人员已离职,标记 ops_asset.status=1(停用)并归档",
			r1Default: "物理有/责任人无 — 资产已采集但未分配责任人",
		},
		{
			name: "对账-D类",
			description: "1. 确认资产是否仍在用(查最近 30 天 sys_device_mac_history)\n" +
				"2. 若仍在用,在 ops_asset.user_id 上同步到物理使用人\n" +
				"3. 同步更新 sys_workstation.user_id(若资产绑定工位)\n" +
				"4. 通知原责任人变更结果,确认无需后续资产交割\n" +
				"5. 若资产已无物理使用,标记 ops_asset.status=1(停用)并归档",
			r1Default: "物理有/责任人有但不一致 — 责任人变更未生效或工位调岗",
		},
		{
			name: "对账-E类",
			description: "1. 反查 sys_workstation 是否有该资产绑定(端口表 sys_port_mac)\n" +
				"2. 若工位存在但未绑资产:补充 ops_asset.workstation_id 关联\n" +
				"3. 若工位已下线:在 ops_asset 上标记废弃状态\n" +
				"4. 同步 AD 域控信息(检查 sys_user_ad_attrs.is_enabled)\n" +
				"5. 三方一致后,在 sys_data_reconciliation 标记 resolved_at",
			r1Default: "三方(物理/责任人/AD)互不一致 — 重大异常需人工核查",
		},
		{
			name: "对账-F类",
			description: "1. 检查 AD is_enabled 是否为 false(sys_user_ad_attrs)\n" +
				"2. 同步 sys_user.status=0(启用)与 AD 一致\n" +
				"3. 若 AD 账号已停用:在 sys_user 上同步停用状态\n" +
				"4. 同步 ops_asset.user_id 关联到正确 sys_user\n" +
				"5. 修复后刷新 reconciliation_normalized MV 验证一致",
			r1Default: "缺数据 — 资产或工位任一端基础数据缺失",
		},
	}

	for _, t := range templates {
		// 幂等:仅在原 description 为 NULL 或 R1 默认值时 UPDATE
		// 运维在 admin 页手动修改过的 description 不覆盖
		result := db.Model(&models.WorkOrderCategory{}).
			Where("category_name = ? AND (description IS NULL OR description = '' OR description = ?)",
				t.name, t.r1Default).
			Update("description", t.description)
		if result.Error != nil {
			log.Printf("Migration 172: update %s description failed: %v", t.name, result.Error)
			continue
		}
		if result.RowsAffected == 0 {
			applogger.Debugf("[迁移] %s description 已是运维修改版,跳过", t.name)
			continue
		}
		applogger.Infof("[迁移] %s description 已补种 (rows=%d)", t.name, result.RowsAffected)
	}

	log.Println("Migration 172 completed: 6 reconciliation workorder category descriptions backfilled (or skipped if manually edited)")
	return nil
}