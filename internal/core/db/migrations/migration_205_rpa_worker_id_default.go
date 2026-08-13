package migrations

import (
	"fmt"
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate205RpaWorkerIdDefault 给 sys_rpa_workers.id 列补 DEFAULT gen_random_uuid()
//
// 背景:
//   - sys_rpa_workers 由 legacy SQL(102_add_rpa_tables.sql)建表,id 列当初未带 DEFAULT。
//   - worker_service.Register 用原生 SQL INSERT 走 ON CONFLICT DO UPDATE 路径,绕过 GORM
//     Create 钩子链(Worker.BeforeCreate / BaseModel.BeforeCreate 显式赋 UUID 均不触发)。
//   - 历史上 Register 的 INSERT 列清单未带 id → PG 收到 NULL → SQLSTATE 23502 NOT NULL 违规,
//     RPA 节点自动注册接口返回 400。
//
// 解决:
//   - A1(代码层): Register 的 INSERT 显式在 VALUES 首位写 gen_random_uuid(),服务端生成 id。
//   - A2(本迁移,根治层): 给 sys_rpa_workers.id 列补 DEFAULT gen_random_uuid(),与全库
//     UUID 主键惯例(其他 sys_* 表均由 GORM AutoMigrate 带此 DEFAULT)对齐。任何未来省略
//     id 的插入路径也能由 DEFAULT 兜底,不再触发 23502。
//
// 幂等性: ALTER COLUMN ... SET DEFAULT 本身幂等(PG 不报错,重复执行结果相同),
// 无 IF NOT EXISTS 语法,无需额外守护。
//
// 范围: 仅改 sys_rpa_workers.id 的 DEFAULT,不动其他列、不加索引、不动外键。
// rpamodels.Worker 不加入 MigrateModelList(legacy SQL 建表,GORM AutoMigrate 会触发未知 DDL 副作用)。
func Migrate205RpaWorkerIdDefault(db *gorm.DB) error {
	log.Println("Running migration 205: sys_rpa_workers.id SET DEFAULT gen_random_uuid()")

	if !isPostgreSQL(db) {
		applogger.Infof("[迁移] 205 跳过(非 PostgreSQL)")
		log.Println("Migration 205 skipped (non-PostgreSQL dialect)")
		return nil
	}

	// SET DEFAULT 幂等: 多次执行结果一致, 不报错。
	// gen_random_uuid() 是 PG 13+ 内置(PG 9.4-12 需 pgcrypto 扩展,本仓库 PG 18 已内置)。
	if err := db.Exec(`
ALTER TABLE sys_rpa_workers
ALTER COLUMN id SET DEFAULT gen_random_uuid()`).Error; err != nil {
		return fmt.Errorf("sys_rpa_workers.id SET DEFAULT gen_random_uuid() 失败: %w", err)
	}

	applogger.Infof("[迁移] 205 sys_rpa_workers.id DEFAULT gen_random_uuid() 已设置")
	log.Println("Migration 205 completed")
	return nil
}
