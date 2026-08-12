//go:build archive_skip


package migrations

import (
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate197ReconciliationWorkorderCategoriesAlign 同步对齐
// sys_workorder_category 字典 (对账-A/B/C/D/E/F 类) 描述与 detection 真值。
//
// 根因 (与 M196 同根, 2026-07-02 排查 .planning/debug/reconciliation-health-misjudgment):
//   - migration_169 行 222-227 R1 seed 时, 对账工单类别的 description 与
//     detection engine ClassifyType 真值完全相反 (B/C/D 文字顺序颠倒).
//   - 工单模板按 category_name 引用 description, 描述错位会让新工单通知/前端
//     工单详情显示的"原因/处理建议"完全偏离真实语义.
//
// 真值 (与 M196 同步, reconciliation_detection.go ClassifyType D-08/D-09 锁定):
//
//	A: physical + declared 匹配        → 不入主表 (success)
//	B: physical 有, declared 无         → high    (error)
//	C: physical + declared 都有但不匹配 → high    (error)
//	D: physical 无, declared 有         → medium  (warning)
//	E: physical + declared 都没有       → low     (default)
//	F: physical/declared 匹配但 AD 不一致 → medium  (warning)
//
// 改动: UPDATE sys_workorder_category SET description=? WHERE category_name='对账-?类'
// 幂等: 先 SELECT 当前 description, 一致则跳过.
//
// 不修改: 字典 dict_label (M196 已修), detection engine logic,
//         HealthBadge.tsx (已是权威文案), sys_dict_data 中相关 severity 字典.
func Migrate197ReconciliationWorkorderCategoriesAlign(db *gorm.DB) error {
	log.Println("Running migration 197: 对齐 sys_workorder_category 对账-A/B/C/D/E/F 描述与 detection 真值")

	type catUpdate struct {
		name        string // category_name (对账-A类 / 对账-B类 / ...)
		newDesc     string // 对齐 detection 后的中文描述
	}
	updates := []catUpdate{
		{"对账-A类", "物理有/责任人有且一致 健康无需动作(R1 仅作 dashboard 统计,不入 sys_data_reconciliation)"},
		{"对账-B类", "物理有/责任人无 — 资产已采集但未分配责任人 (高危)"},
		{"对账-C类", "物理有/责任人有但不一致 — 责任人变更未生效或工位调岗 (高危)"},
		{"对账-D类", "物理无(未采集)/责任人有 — 资产未上线或采集缺失 (中危)"},
		{"对账-E类", "双方都无用户关联 — 物理链路与责任人均未登记 (低危)"},
		{"对账-F类", "物理与责任人一致但 AD 不一致 — AD 账号错配 (中危,AD 已知不可靠)"},
	}

	for _, u := range updates {
		// 幂等: 读当前 description, 一致则跳过
		var currentDesc string
		row := db.Table("sys_workorder_category").
			Select("description").
			Where("category_name = ?", u.name).
			Row()
		if err := row.Scan(&currentDesc); err != nil {
			applogger.Errorf("Migration 197: 读取 description 失败 (name=%s): %v", u.name, err)
			return err
		}
		if currentDesc == u.newDesc {
			log.Printf("Migration 197: description 已对齐, 跳过 (name=%s)", u.name)
			continue
		}

		// UPDATE: 写新 description
		tx := db.Exec(
			"UPDATE sys_workorder_category SET description = ? WHERE category_name = ?",
			u.newDesc, u.name,
		)
		if tx.Error != nil {
			applogger.Errorf("Migration 197: UPDATE description 失败 (name=%s): %v", u.name, tx.Error)
			return tx.Error
		}
		applogger.Infof("[迁移] sys_workorder_category %s: description 已对齐 (旧 %q → 新 %q)",
			u.name, truncate(currentDesc, 30), truncate(u.newDesc, 30))
	}

	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}