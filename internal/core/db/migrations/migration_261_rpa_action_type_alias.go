package migrations

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"gorm.io/gorm"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// ============================================================================
// Migration 261 — W7.1 / 缺陷 A：RPA 任务脚本里的动作类型别名归一化（数据根治）
//
// 为什么需要数据迁移（alias 只是兜底）：
//   worker 的 ActionType 常量表只有 "navigate"（rpa-worker/internal/types/types.go），
//   但生产库存量脚本里出现过 "goto"（普查实证：19 个任务 / 12 种 type，
//   非规范写法仅 goto 共 6 步，全部是 2026-03-10 创建的测试数据）。
//   worker 侧已加别名归一化（goto/goTo/GOTO/url → navigate）作为兜底，
//   保证「即使数据不改也能跑」；但历史数据长期漂移会持续制造困惑
//   （前端编辑器显示 goto、AI 提示词把 goto 当候选类型等），所以必须根治。
//
// 迁移范围：
//   只改 script 数组里个别动作的 type 字段，**不动任何其它字段**
//   （selector / value / params / timeout / retry 原样保留）。
//   value 里放 URL 的老形状也照旧 —— worker 的 mapToAction 会把 navigate 的
//   value 提升到 params.url（缺陷 A 的第二层修复），因此迁移后可直接跑通。
//
// 幂等（双方言）：
//   · 逐任务解析 JSON，只在「确实命中了别名」时 UPDATE；
//   · 第二次执行时全部动作都已是 navigate → 零 UPDATE（不写库、不产生噪声）；
//   · PG 与 sqlite 的差异集中在类型转换（jsonb / uuid vs TEXT），用
//     isPostgreSQL(db) 分支处理，语句本身同构。
//
// 非阻断：单条任务写失败只记 WARN 并继续（一条脏数据不该让整个启动失败）；
// 但会汇总最终失败数，全部失败时返回错误（让 runAndRecord 不记录版本号，
// 下次启动重试）。
// ============================================================================

// rpaActionTypeAliasCanonical 迁移侧别名表：历史/前端惯用写法 → 规范写法。
//
// ⚠ 与 worker 侧的 actionTypeAliases（rpa-worker/internal/executor/engine.go）
// 必须保持同一集合。worker 是独立 Go module，无法被本仓库 import，因此这里是
// 一份刻意复制的清单；新增别名时**两处同时改**，否则会出现「数据迁移改良了、
// 运行时仍不认」或反过来的半修状态。
var rpaActionTypeAliasCanonical = map[string]string{
	"goto": "navigate", // 生产库实测 6 步（2026-03-10 的测试任务）
	"url":  "navigate", // 历史脚本把 URL 当类型名
}

// Migrate261RPActionTypeAliasNormalize 把 sys_rpa_tasks.script 里的别名 type 改成规范值。
func Migrate261RPActionTypeAliasNormalize(db *gorm.DB) error {
	log.Println("Running migration 261: normalize RPA action type aliases in sys_rpa_tasks.script")

	if !db.Migrator().HasTable("sys_rpa_tasks") {
		applogger.Warnf("[迁移] migration 261: sys_rpa_tasks 表不存在（全新库尚未建表），跳过")
		return nil
	}

	type row struct {
		ID     string
		Script string
	}
	var rows []row

	// id 在 PG 是 uuid、在 sqlite 是 TEXT；script 在 PG 是 jsonb、在 sqlite 是 TEXT。
	// SELECT 统一转成文本，避免驱动层类型不匹配。
	selectSQL := `SELECT CAST(id AS TEXT) AS id, CAST(script AS TEXT) AS script FROM sys_rpa_tasks ORDER BY CAST(id AS TEXT)`
	if isPostgreSQL(db) {
		selectSQL = `SELECT id::text AS id, script::text AS script FROM sys_rpa_tasks ORDER BY id::text`
	}
	if err := db.Raw(selectSQL).Scan(&rows).Error; err != nil {
		return fmt.Errorf("migration 261: 查询 sys_rpa_tasks 失败: %w", err)
	}

	updated, scanned, failed, unparsable := 0, 0, 0, 0
	for _, r := range rows {
		scanned++
		var actions []map[string]interface{}
		if err := json.Unmarshal([]byte(r.Script), &actions); err != nil {
			// 不是数组（或历史脏数据）：不碰它，只记一条 WARN，交给人工核查。
			unparsable++
			applogger.Warnf("[迁移] migration 261: 任务 %s 的 script 无法解析为动作数组，跳过: %v", r.ID, err)
			continue
		}

		changed := 0
		for _, action := range actions {
			raw, ok := action["type"].(string)
			if !ok || raw == "" {
				continue
			}
			canonical, hit := rpaActionTypeAliasCanonical[strings.ToLower(raw)]
			if !hit || canonical == raw {
				continue
			}
			action["type"] = canonical
			changed++
			applogger.Infof("[迁移] migration 261: 任务 %s 动作 type %q → %q", r.ID, raw, canonical)
		}
		if changed == 0 {
			continue
		}

		data, err := json.Marshal(actions)
		if err != nil {
			failed++
			applogger.Errorf("[迁移] migration 261: 任务 %s 重新序列化失败: %v", r.ID, err)
			continue
		}

		// 方言差异只在类型转换：PG 的 script 是 jsonb、id 是 uuid，需显式 cast；
		// sqlite 两者都是 TEXT，直接赋值即可。
		updateSQL := `UPDATE sys_rpa_tasks SET script = ? WHERE CAST(id AS TEXT) = ?`
		args := []interface{}{string(data), r.ID}
		if isPostgreSQL(db) {
			updateSQL = `UPDATE sys_rpa_tasks SET script = CAST(? AS jsonb) WHERE id = CAST(? AS uuid)`
		}
		if err := db.Exec(updateSQL, args...).Error; err != nil {
			failed++
			applogger.Errorf("[迁移] migration 261: 任务 %s 写回失败: %v", r.ID, err)
			continue
		}
		updated++
	}

	applogger.Infof("[迁移] migration 261 完成：扫描 %d 个任务，改写 %d 个，解析失败 %d 个，写失败 %d 个",
		scanned, updated, unparsable, failed)
	if failed > 0 {
		return fmt.Errorf("migration 261: %d 个任务写回失败（版本号不记录，下次启动重试）", failed)
	}
	return nil
}
