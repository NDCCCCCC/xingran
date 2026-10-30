package migrations

import (
	"fmt"
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate220ManagerRuleTables 建立「通用经理同步方案」所需的两张表。
//
// 表语义：
//   - sys_manager_org  机构定义（一个机构 = sys_dept 树上的某个节点）；
//   - sys_manager_rule 经理推导规则（主体 + 范围 + 经理取值来源 + 优先级）。
//
// 实现要点：
//   - 建表走 GORM AutoMigrate（与本仓库其它建表迁移一致，如 models.PortWriteAudit），
//     表/列已存在则跳过，天然幂等，重复执行不报错。
//   - 两模型的 ID 由 BaseModel.BeforeCreate 钩子用 uuid.New() 填充，
//     不含 gen_random_uuid() 函数式默认值 → SQLite / PostgreSQL 双方言均可建表
//     （无需经 sanitizeSQLiteModelDefaults 净化）。
//
// 非阻断档：建表失败只让经理同步端点 500（表不存在），不影响登录 / 入队 / 排水主链路
// （与 migration_216/217/218 同档）。
func Migrate220ManagerRuleTables(db *gorm.DB) error {
	log.Println("Running migration 220: manager org/rule tables")

	if err := db.AutoMigrate(&models.ManagerOrg{}, &models.ManagerRule{}); err != nil {
		return fmt.Errorf("经理同步规则表创建失败: %w", err)
	}

	log.Println("Migration 220 completed: sys_manager_org / sys_manager_rule ready")
	return nil
}
