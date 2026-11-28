package rpa

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// HumanInterventionEvent BeforeCreate 钩子（Phase 153 / RUNTIME-04）。
//
// 独立文件实现，**不触碰 excel_service.go**（Phase 158 正在重写其批量报告区）。
// 此前事件 ID 恒为空串：PG 由建表默认 gen_random_uuid() 兜底，但 GORM Create
// 显式插入空串在 sqlite 下 UNIQUE 冲突 —— 第二次提交干预必失败。
// 钩子补齐应用层 UUID 生成，与 rpamodels.Execution.BeforeCreate 同一先例。
func (e *HumanInterventionEvent) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	return nil
}
