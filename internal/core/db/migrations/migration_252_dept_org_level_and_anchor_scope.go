package migrations

// Migration 252 — 机构层级与数据权限锚点维度。
//
// 业务背景：系统三层机构架构「分公司 → 中支 → 支公司」。现有数据范围只有
// 5 档静态枚举（sys_role.data_scope），锚点被硬编码为"用户自己的 dept_id"，
// 导致「一个共享角色无法按人员所属机构动态授权」，只能每个机构建一个角色。
//
// 本迁移只做**结构**改动，不含任何数据猜测：
//   1. sys_dept.org_level          机构层级（0=非机构 1=分公司 2=中支 3=支公司 4=网点）
//   2. sys_role.anchor_mode        角色级锚点模式（0=不参与 2=指定部门 3=外部机构 5=机构层级）
//   3. sys_role.anchor_param       角色级锚点参数（多态：dept_id / 层级整数 / 空）
//   4. sys_post.anchor_mode        岗位级锚点模式
//   5. sys_post.anchor_param       岗位级锚点参数
//
// 重要设计约束（违反即静默降权事故）：
//   - anchor_mode **不参与** getUserMaxDataScope 的 MAX(data_scope) 排序。
//     DataScope 枚举数值越大越窄，若把锚点维度塞进同一枚举，一个同时持有
//     All(1) 角色的用户会被收窄成机构范围。故锚点是独立维度，独立解析后
//     与角色维度取并集（宽者胜）。
//   - 新增列一律 default 0 / ''，存量行零回归：不做任何"根据 ancestors 猜层级"
//     的自动推导 —— 中支含分公司本部等特例纯靠深度推导必然判错。
//
// 幂等：全部走 GORM Migrator 的 HasColumn/AddColumn，双方言安全，重复执行无副作用。
// 非阻断：由 database.go 的 runAndRecord 调用；失败只告警不中断启动。

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// Migrate252DeptOrgLevelAndAnchorScope 为部门/角色/岗位补机构层级与锚点列。
func Migrate252DeptOrgLevelAndAnchorScope(db *gorm.DB) error {
	log.Println("Running migration 252: dept org_level + role/post anchor scope")

	// 1) sys_dept.org_level
	if !db.Migrator().HasTable("sys_dept") {
		log.Println("Migration 252: WARNING sys_dept 表不存在, 跳过 org_level（非阻断）")
	} else if !db.Migrator().HasColumn("sys_dept", "org_level") {
		if err := db.Migrator().AddColumn(&orgLevelCarrier{}, "OrgLevel"); err != nil {
			return fmt.Errorf("migration 252 新增 sys_dept.org_level 失败: %w", err)
		}
		log.Println("Migration 252: sys_dept.org_level 已创建")
	}

	// 2/3) sys_role.anchor_mode / anchor_param
	if !db.Migrator().HasTable("sys_role") {
		log.Println("Migration 252: WARNING sys_role 表不存在, 跳过锚点列（非阻断）")
	} else {
		if !db.Migrator().HasColumn("sys_role", "anchor_mode") {
			if err := db.Migrator().AddColumn(&roleAnchorCarrier{}, "AnchorMode"); err != nil {
				return fmt.Errorf("migration 252 新增 sys_role.anchor_mode 失败: %w", err)
			}
			log.Println("Migration 252: sys_role.anchor_mode 已创建")
		}
		if !db.Migrator().HasColumn("sys_role", "anchor_param") {
			if err := db.Migrator().AddColumn(&roleAnchorCarrier{}, "AnchorParam"); err != nil {
				return fmt.Errorf("migration 252 新增 sys_role.anchor_param 失败: %w", err)
			}
			log.Println("Migration 252: sys_role.anchor_param 已创建")
		}
	}

	// 4/5) sys_post.anchor_mode / anchor_param
	if !db.Migrator().HasTable("sys_post") {
		log.Println("Migration 252: WARNING sys_post 表不存在, 跳过锚点列（非阻断）")
	} else {
		if !db.Migrator().HasColumn("sys_post", "anchor_mode") {
			if err := db.Migrator().AddColumn(&postAnchorCarrier{}, "AnchorMode"); err != nil {
				return fmt.Errorf("migration 252 新增 sys_post.anchor_mode 失败: %w", err)
			}
			log.Println("Migration 252: sys_post.anchor_mode 已创建")
		}
		if !db.Migrator().HasColumn("sys_post", "anchor_param") {
			if err := db.Migrator().AddColumn(&postAnchorCarrier{}, "AnchorParam"); err != nil {
				return fmt.Errorf("migration 252 新增 sys_post.anchor_param 失败: %w", err)
			}
			log.Println("Migration 252: sys_post.anchor_param 已创建")
		}
	}

	log.Println("Migration 252 completed: org_level + anchor scope columns ready")
	return nil
}

// 以下 carrier 结构只用于 Migrator().AddColumn 承载列定义（不带任何业务字段），
// 保证迁移的列定义与 models 中同名 tag 一致，避免手工写 DDL 在 PG/SQLite 间漂移。

// orgLevelCarrier 承载 sys_dept.org_level 的列定义。
type orgLevelCarrier struct {
	OrgLevel int `gorm:"column:org_level;default:0"`
}

func (orgLevelCarrier) TableName() string { return "sys_dept" }

// roleAnchorCarrier 承载 sys_role 的锚点列定义。
type roleAnchorCarrier struct {
	AnchorMode  int    `gorm:"column:anchor_mode;default:0"`
	AnchorParam string `gorm:"column:anchor_param;size:64;default:''"`
}

func (roleAnchorCarrier) TableName() string { return "sys_role" }

// postAnchorCarrier 承载 sys_post 的锚点列定义。
type postAnchorCarrier struct {
	AnchorMode  int    `gorm:"column:anchor_mode;default:0"`
	AnchorParam string `gorm:"column:anchor_param;size:64;default:''"`
}

func (postAnchorCarrier) TableName() string { return "sys_post" }
