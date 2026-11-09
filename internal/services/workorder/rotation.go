package workorder

// FIX-05（v1.53 Phase 138）：周期性工单 Rotation 轮询分配语义的唯一真相源。
//
// 背景：cron 路径（internal/scheduler/workorder_tasks.go 的 assignWorkOrderHandler）
// 原先直接 `return nil, nil` 不分配，而 service 路径（本包 periodic.go 的
// GenerateWorkOrder）已实现按 member_order 取模轮询 —— 两条路径不一致。
//
// 本文件把「成员集合来源 + 取模选人」抽成两条路径共用的纯逻辑，避免复制粘贴
// 两份各自漂移的实现。两条路径仍各自管理 total_generated 游标 +1（service
// 用 UPDATE...RETURNING 原子 +1；scheduler 在事务末尾统一 +1），但选人所用的
// 「分配前游标」语义必须一致，见 RotationSelectedIndex 注释。

import (
	"context"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// LoadRotationMembers 读取值班池成员，按 member_order ASC 排序。
//
// 成员集合来自 sys_duty_pool_member，WHERE pool_id = 模板的 AssignTargetID。
// service 路径（GenerateWorkOrder）与 scheduler 路径（assignWorkOrderHandler）
// 共用本函数，保证「成员集合来源 + 排序」两路径完全一致。
//
// db 形参接受任意 *gorm.DB：service 传 s.db，scheduler 传事务 tx；二者
// 都通过 WithContext(ctx) 传播上下文。
func LoadRotationMembers(ctx context.Context, db *gorm.DB, poolID string) ([]models.DutyPoolMember, error) {
	var members []models.DutyPoolMember
	err := db.WithContext(ctx).
		Where("pool_id = ?", poolID).
		Order("member_order ASC").
		Find(&members).Error
	return members, err
}

// RotationSelectedIndex 由「分配前已生成工单数」与成员数计算轮询索引。
//
// 三处等价、勿改（这是双路径一致性的核心不变量）：
//   - service 原子路径：UPDATE...RETURNING 得 newCount = old+1，
//     原实现 selectedIndex = (newCount-1) % len(members)
//   - service 降级路径：selectedIndex = template.TotalGenerated % len(members)
//   - scheduler 路径：selectedIndex = template.TotalGenerated % len(members)
//
// 三者都化简为 cursorBefore % memberCount，其中 cursorBefore 是「本次分配前」
// 的 total_generated。memberCount <= 0 返回 -1（调用方据此判定无可用成员）；
// cursorBefore 为负时做一次归一，避免异常数据下索引越界。
func RotationSelectedIndex(cursorBefore, memberCount int) int {
	if memberCount <= 0 {
		return -1
	}
	idx := cursorBefore % memberCount
	if idx < 0 {
		idx += memberCount
	}
	return idx
}
