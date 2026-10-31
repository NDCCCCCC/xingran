package migrations

import (
	"log"

	"gorm.io/gorm"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
)

// Migrate230RPAWorkerToken —— v1.51 DEBT-03：RPA Worker API Token 认证字段。
//
// sys_rpa_workers 新增 token / token_expires_at / last_token_rotation 三列，
// 支撑 Provisioning Token 换发 Worker Token 的认证方案
// （docs/RPA-Worker-API认证方案-待办.md，Phase 35 起搁置、v1.51 DEBT-03 落地）。
//
// 实现方式：AutoMigrate(Worker)（加列安全，不动既有行）+ HasColumn 幂等兜底。
// 老 Worker 无 token（宽限期模式）→ 三列全空，不影响既有部署。
//
// 幂等：列已存在则跳过；失败非阻断（调用方 WARN）。
func Migrate230RPAWorkerToken(db *gorm.DB) error {
	log.Println("Running migration 230: DEBT-03 RPA Worker Token 认证字段")

	if db.Migrator().HasColumn(&rpamodels.Worker{}, "token") {
		log.Println("Migration 230: token 列已存在，跳过")
		return nil
	}
	if err := db.AutoMigrate(&rpamodels.Worker{}); err != nil {
		return err
	}
	// 二次确认（AutoMigrate 理论上必加列，防御方言差异）
	if !db.Migrator().HasColumn(&rpamodels.Worker{}, "token") {
		if err := db.Migrator().AddColumn(&rpamodels.Worker{}, "Token"); err != nil {
			return err
		}
	}
	log.Println("Migration 230: RPA Worker Token 字段就绪")
	return nil
}
