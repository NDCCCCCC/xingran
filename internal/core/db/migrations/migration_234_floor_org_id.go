package migrations

import (
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
)

// Migrate234FloorOrgID —— 为 ops_floors 增加可空 org_id（楼层所属机构覆盖值）。
//
// 背景（quick-261106-wso）：一栋楼跨机构办公（如浙商大厦 4/5/6/47 楼属分公司
// 本部、1/3 楼属武汉中支），而 ops_buildings.org_id 是单值，工位/楼层/楼宇的
// 机构筛选全部读楼宇 org_id → 1/3 楼无法显示武汉中支的部门，左侧选武汉中支
// 也筛不出 1/3 楼工位。归属真实粒度在楼层这一层。
//
// 语义：org_id 为 NULL 或空串 = 继承 ops_buildings.org_id（COALESCE 口径），
// 因此**不做任何历史数据回填**，存量楼层行为完全不变，向后兼容。
//
// 与 AutoMigrate 的关系：
//   - OpsFloor 已在 MigrateModelList 注册（internal/core/db/database.go:759），
//     正常启动路径由 GORM AutoMigrate 自动 ADD COLUMN。
//   - 本迁移覆盖 SKIP_AUTOMIGRATE=true 旁路（BootstrapMissingTables 之后仍需补列）
//     以及 AutoMigrate 未覆盖该模型的极端分支。
//   - HasColumn 守卫保证幂等；ADD COLUMN 无 NOT NULL/DEFAULT，PG/SQLite 双方言安全。
func Migrate234FloorOrgID(db *gorm.DB) error {
	log.Println("Running migration 234: ops_floors.org_id 楼层机构覆盖列")

	if !db.Migrator().HasTable(&operations.OpsFloor{}) {
		log.Println("Migration 234: ops_floors 表不存在，跳过")
		return nil
	}
	if db.Migrator().HasColumn(&operations.OpsFloor{}, "org_id") {
		log.Println("Migration 234: ops_floors.org_id 已存在，跳过")
		return nil
	}
	if err := db.Migrator().AddColumn(&operations.OpsFloor{}, "org_id"); err != nil {
		return err
	}

	log.Println("Migration 234: ops_floors.org_id 添加完成（NULL = 继承楼宇 org_id）")
	return nil
}
