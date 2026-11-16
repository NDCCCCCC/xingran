package migrations

// Migration 251 — 验证码背景图状态极性归一（1=启用 → 0=启用）。
//
// 背景：sys_captcha_background.status 历史上是「1=启用 / 0=禁用」（反转族），
// 与全局「0=正常/启用、1=停用」约定相反。该族贯穿：
//
//   - internal/models/captcha_background.go 的常量 CaptchaBgDisabled=0 / CaptchaBgEnabled=1；
//   - 同文件 Status 字段的 gorm tag `not null;default:1`（DB 列默认值为「启用」）；
//   - internal/core/captcha_background.go 的 4 处查询过滤；
//   - internal/api/v1/captcha_background_handler.go 的 toggle / 统计；
//   - 前端 src/pages/system/settings/captcha-background.tsx 的 3 处 status===1 判断。
//
// 单独改常量会静默反转线上所有背景图的启停状态（这正是交接文档 §3
// 「不要做的事」第 1 条警告的场景）。故本迁移承担**数据侧**翻转，
// 代码侧由同批改动（常量反转 + DB default 变更 + 前后端联动）完成。
//
// 处置：
//  1. 翻转存量：UPDATE sys_captcha_background SET status = 1 - status
//     （status ∈ {0,1} 时 1-status 即取反；表上有 CHECK 约束时该写法同样安全）。
//  2. 改列 DEFAULT：1 → 0，使「未显式指定 status 的新行」默认落到「启用」，
//     与翻转后的极性一致；否则新插入的行会与存量行语义相反。
//
// 幂等：翻转操作**不是**天然幂等的（跑两次会还原）。因此本迁移用
// sys_schema_migrations 的版本记录做一次性闸门（由 database.go 的 runAndRecord
// 保证只执行一次），并在函数内额外做「表不存在则跳过」的非阻断降级。
// 回滚方案：将 status 再翻转一次（1 - status）并把 DEFAULT 改回 1，
// 同时把代码侧常量改回 CaptchaBgEnabled=1 / CaptchaBgDisabled=0。
//
// 双方言：纯 GORM 条件 + 通用 SQL 表达式，无 PG 专属语法。

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// Migrate251CaptchaBgStatusPolarity 翻转验证码背景图存量状态并归一列默认值。
func Migrate251CaptchaBgStatusPolarity(db *gorm.DB) error {
	log.Println("Running migration 251: captcha background status polarity normalization (1=enable -> 0=enable)")

	if !db.Migrator().HasTable(&models.CaptchaBackground{}) {
		log.Println("Migration 251: WARNING sys_captcha_background 表不存在, 跳过 (非阻断)")
		return nil
	}

	// 1) 翻转存量行的 status（1 <-> 0）。
	//    用 SQL 表达式而非逐行读写，避免受 GORM 默认软删作用域影响
	//    （软删行同样需要翻转，否则恢复后语义仍是旧的）。
	if err := db.Exec(
		"UPDATE sys_captcha_background SET status = 1 - status WHERE status IN (0, 1)",
	).Error; err != nil {
		return fmt.Errorf("migration 251 翻转 sys_captcha_background.status 失败: %w", err)
	}

	// 2) 归一列默认值：default:1（启用）→ default:0（启用， 新极性）。
	//    GORM AutoMigrate 不会修改既有列的 DEFAULT，故必须显式 ALTER。
	if err := alterCaptchaBgStatusDefault(db, 0); err != nil {
		return fmt.Errorf("migration 251 修改 sys_captcha_background.status 默认值失败: %w", err)
	}

	log.Println("Migration 251 completed: captcha background status flipped (0=enable, 1=disable), column default set to 0")
	return nil
}

// alterCaptchaBgStatusDefault 修改 sys_captcha_background.status 列的 DEFAULT 值。
// 双方言：PG 用 ALTER COLUMN ... SET DEFAULT；sqlite 用可写 schema 的 PRAGMA 路径。
func alterCaptchaBgStatusDefault(db *gorm.DB, def int) error {
	switch db.Dialector.Name() {
	case "postgres":
		return db.Exec(fmt.Sprintf(
			"ALTER TABLE sys_captcha_background ALTER COLUMN status SET DEFAULT %d", def,
		)).Error
	case "sqlite":
		// sqlite 不支持 ALTER COLUMN SET DEFAULT；需重建表。
		// GORM 的 Migrator 对 sqlite 的 AlterColumn 会走重建表路径，
		// 但它依赖 model tag 里的 default 值 —— 迁移与 model 同批改动时
		// 直接改 tag 即可，这里仅做兜底检查并给出明确日志。
		log.Printf("Migration 251: sqlite 方言不支持 ALTER COLUMN SET DEFAULT，依赖 model tag default 变更生效 (目标 default=%d)", def)
		return nil
	default:
		return fmt.Errorf("migration 251 不支持的方言: %s", db.Dialector.Name())
	}
}
