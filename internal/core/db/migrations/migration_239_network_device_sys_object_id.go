package migrations

import (
	"log"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// Migrate239NetworkDeviceSysObjectID —— 为 sys_network_device 与
// sys_device_discovery_result 增加可空 sys_object_id 列（B-3 / v1.56）。
//
// 语义：SNMP sysObjectID (1.3.6.1.2.1.1.2.0) 是设备身份属性，用做型号能力矩阵
// 的精确/前缀匹配键（Model 子串兜底）。NULL/空串 = 未采集，匹配时自动回退 Model，
// 因此**不做任何历史数据回填**，存量设备行为完全不变，向后兼容。
//
// 两张表都要加：发现→导入主链路经过 sys_device_discovery_result 中转，
// 若不给它加列，OID 在持久化中转时即丢失，导入时无值可写。
//
// 与 AutoMigrate 的关系（同 migration_234）：
//   - NetworkDevice / DeviceDiscoveryResult 已在 MigrateModelList 注册，
//     正常启动路径由 GORM AutoMigrate 自动 ADD COLUMN。
//   - 本迁移覆盖 SKIP_AUTOMIGRATE=true 旁路（BootstrapMissingTables 之后仍需补列）
//     以及 AutoMigrate 未覆盖该模型的极端分支。
//   - HasColumn 守卫保证幂等；ADD COLUMN 无 NOT NULL/DEFAULT，PG/SQLite 双方言安全。
func Migrate239NetworkDeviceSysObjectID(db *gorm.DB) error {
	log.Println("Running migration 239: sys_object_id (network device + discovery result)")

	type target struct {
		model  interface{}
		column string
		table  string
	}
	targets := []target{
		{&models.NetworkDevice{}, "sys_object_id", "sys_network_device"},
		{&models.DeviceDiscoveryResult{}, "sys_object_id", "sys_device_discovery_result"},
	}
	for _, t := range targets {
		if !db.Migrator().HasTable(t.model) {
			log.Printf("Migration 239: %s 表不存在，跳过", t.table)
			continue
		}
		if db.Migrator().HasColumn(t.model, t.column) {
			log.Printf("Migration 239: %s.%s 已存在，跳过", t.table, t.column)
			continue
		}
		if err := db.Migrator().AddColumn(t.model, t.column); err != nil {
			return err
		}
		log.Printf("Migration 239: %s.%s 添加完成（空 = 未采集，回退 Model 匹配）", t.table, t.column)
	}
	return nil
}
