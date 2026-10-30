//go:build archive_skip


package migrations

import (
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate196ReconciliationDictLabelsAlign 修正 sys_dict_data 中
// asset_reconciliation_conflict_type 字典 B/C/D/E/F 五类的 dict_label 文案
// 与 list_class 颜色档次, 与检测引擎 (internal/services/asset/reconciliation_detection.go
// ClassifyType + ComputeSeverity) 保持一致。
//
// 根因 (2026-07-02 排查: reconciliation-health-misjudgment):
//   - migration_169 行 83-90 R1 seed 时, dict label 与 detection 真分类语义完全相反,
//     且 list_class 颜色档次与 ComputeSeverity 也存在 2 处偏差
//     (B 应该是 error 高危但 seed 是 warning; F 应该是 warning 中危但 seed 是 info)。
//   - 截图 9 (wangwenye-001 / yangfan-131): 实际为 Type C (使用人不一致, 高危),
//     但 seed label 写作 "C类-物理有责无", 异常列表渲染成 "物理有责任/无人"。
//   - 截图 10 (xiaoshan / xiaoshan): 实际应 Type A (健康) 不入主表, 但若走到主表,
//     seed 的 F label "缺数据" 完全偏离真实语义。
//   - 截图 11 (luowei-020 / 责任人空): 实际 Type B (物理有/责任人无),
//     seed label 写作 "B类-物理无责" 渲染成 "物理无责", 文字顺序颠倒。
//   - 用户反馈 "ABCDEF 是否按严重度排序" → 字母按 signals 组合分配 (非严重度),
//     但 seed 的 list_class 颜色档次与 ComputeSeverity 也存在 2 处偏差:
//       B (high) 应 error, seed 给 warning (轻一档)
//       F (medium) 应 warning, seed 给 info (轻一档)
//   - HealthBadge.tsx tooltip 文字才是权威 (与 detection 对齐), 本次让 sys_dict_data
//     也对齐同一真值, 消除"字典/检测"双真相源。
//
// 真值 (reconciliation_detection.go ClassifyType D-08/D-09 锁定, R1 不可改):
//
//	A: physical + declared 匹配        → 不入主表 (success)
//	B: physical 有, declared 无         → high    (error)
//	C: physical + declared 都有但不匹配 → high    (error)
//	D: physical 无, declared 有         → medium  (warning)
//	E: physical + declared 都没有       → low     (default)
//	F: physical/declared 匹配但 AD 不一致 → medium  (warning)
//
// 改动: UPDATE sys_dict_data SET dict_label=?, list_class=? WHERE dict_type='asset_reconciliation_conflict_type' AND dict_value=?
// 幂等: 先 SELECT 当前 dict_label + list_class, 与新值都相同则跳过。
// 不修改 detection engine 逻辑 (用户要求"不动检测逻辑")。
// 不修改 HealthBadge.tsx (已是权威文案, 与本迁移后端对齐)。
// 不修改 sys_workorder_category (同根因, 留待后续单独跟进, 避免越权)。
func Migrate196ReconciliationDictLabelsAlign(db *gorm.DB) error {
	log.Println("Running migration 196: 对齐 asset_reconciliation_conflict_type dict_label + list_class 与 detection 真值")

	type labelUpdate struct {
		value     string // dict_value (A/B/C/D/E/F)
		newLabel  string // 对齐 detection 后的中文文案
		newColor  string // 对齐 ComputeSeverity 后的颜色档次 (primary/success/warning/error/info/default)
	}
	updates := []labelUpdate{
		{"A", "A类-完全一致", "success"},                     // success 不变
		{"B", "B类-物理有/责任人无", "error"},                  // 原: warning (B 是 high)
		{"C", "C类-物理有/责任人不一致(高危)", "error"},           // error 不变 (C 是 high)
		{"D", "D类-物理无/责任人有", "warning"},                // warning 不变 (D 是 medium)
		{"E", "E类-双方都无用户关联", "default"},                // default 不变 (E 是 low)
		{"F", "F类-物理与责任人一致但 AD 不一致", "warning"},       // 原: info (F 是 medium)
	}

	dictType := "asset_reconciliation_conflict_type"

	for _, u := range updates {
		// 幂等: 读当前 label + list_class, 都一致则跳过
		var currentLabel, currentListClass string
		row := db.Table("sys_dict_data").
			Select("dict_label, COALESCE(list_class, '')").
			Where("dict_type = ? AND dict_value = ?", dictType, u.value).
			Row()
		if err := row.Scan(&currentLabel, &currentListClass); err != nil {
			applogger.Errorf("Migration 196: 读取 dict_label/list_class 失败 (value=%s): %v", u.value, err)
			return err
		}
		if currentLabel == u.newLabel && currentListClass == u.newColor {
			log.Printf("Migration 196: dict_label + list_class 已对齐, 跳过 (value=%s)", u.value)
			continue
		}

		// 构造 remark: 记录原值便于审计
		remarkSuffix := " | 2026-07-02 M196 对齐 detection 真值(原 label:" + currentLabel +
			" color:" + currentListClass + ")"

		// UPDATE: dict_label + list_class + remark
		tx := db.Exec(
			"UPDATE sys_dict_data SET dict_label = ?, list_class = ?, remark = COALESCE(remark,'') || ? "+
				"WHERE dict_type = ? AND dict_value = ?",
			u.newLabel, u.newColor, remarkSuffix,
			dictType, u.value,
		)
		if tx.Error != nil {
			applogger.Errorf("Migration 196: UPDATE dict_label/list_class 失败 (value=%s): %v", u.value, tx.Error)
			return tx.Error
		}
		applogger.Infof("[迁移] sys_dict_data %s: label %q→%q, color %q→%q",
			u.value, currentLabel, u.newLabel, currentListClass, u.newColor)
	}

	return nil
}