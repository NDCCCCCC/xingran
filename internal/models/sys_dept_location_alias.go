package models

// SysDeptLocationAlias 工位部门 ↔ 物理位置部门 映射表
//
// 承载"逻辑部门（系统内 sys_dept）→ 物理位置部门（外部机构 / 资产归属方）"的
// 多对一映射，用于工位导入与查询时把外部物理机构下拉合入到本系统的部门选择。
//
// 数据规模：单 sys_dept 通常映射 0~5 个 location_id；scope 字段预留扩展
// （workstation/floor/building），当前实现聚焦 workstation 场景。
//
// 与 sys_workstation.dept_id（size:64）字段类型保持一致，UUID/雪花 ID 均可存入。
//
// 字段命名约定：使用 snake_case 列名 + camelCase JSON 标签，与项目其他表一致。
//
// partial unique constraint `(dept_id, scope) WHERE deleted_at IS NULL` 在
// migration 165 内手写 SQL 实现 —— GORM tag 不支持 partial index 语法。
//
// BaseModel 嵌入：提供 ID (UUID PK) / CreatedAt / UpdatedAt / DeletedAt (软删除)
// + CreatedBy / UpdatedBy / Version（与项目 sys_* 表一致）。
type SysDeptLocationAlias struct {
	BaseModel
	DeptID     string `gorm:"type:varchar(64);not null;index:idx_location_id,priority:2" json:"deptId"`     // 原组织部门ID(逻辑部门) — 系统内 sys_dept.id
	LocationID string `gorm:"type:varchar(64);not null;index:idx_location_id,priority:1" json:"locationId"` // 物理位置部门ID(外部机构) — 工位导入时的归属方
	Scope      string `gorm:"type:varchar(32);not null;default:'workstation'" json:"scope"`                // 作用范围: workstation / floor / building / ...
	Remark     string `gorm:"type:varchar(255)" json:"remark,omitempty"`                                    // 备注
}

// TableName 设置表名
func (SysDeptLocationAlias) TableName() string {
	return "sys_dept_location_alias"
}