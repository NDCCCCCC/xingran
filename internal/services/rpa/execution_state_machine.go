package rpa

import (
	"fmt"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
)

// ============================================================================
// Phase 153 / RUNTIME-05：执行状态机（唯一真相源）
//
// 此前终态判断散落在 execution_service.go（Cancel）与 worker_service.go
// （Progress）两处、各自手写字符串比较，导致终态可被静默覆盖。
// 本文件收敛全部合法迁移；任何状态写入必须经 CanTransition / CheckTransition，
// handler 与 service 禁止再各写一套判断（硬约束）。
//
// 合法迁移表：
//
//	pending  → running | cancelled            （未领取即被取消）
//	running  → success | failed | cancelled | timeout | paused
//	paused   → running | cancelled            （人工恢复 / 暂停中被取消）
//	终态（success/failed/cancelled/timeout）无出边，仅允许幂等重放（from==to）
// ============================================================================

// executionTransitions 合法迁移表。
var executionTransitions = map[rpamodels.RPAExecutionStatus][]rpamodels.RPAExecutionStatus{
	rpamodels.RPAExecutionStatusPending: {
		rpamodels.RPAExecutionStatusRunning,
		rpamodels.RPAExecutionStatusCancelled,
	},
	rpamodels.RPAExecutionStatusRunning: {
		rpamodels.RPAExecutionStatusSuccess,
		rpamodels.RPAExecutionStatusFailed,
		rpamodels.RPAExecutionStatusCancelled,
		rpamodels.RPAExecutionStatusTimeout,
		rpamodels.RPAExecutionStatusPaused,
	},
	rpamodels.RPAExecutionStatusPaused: {
		rpamodels.RPAExecutionStatusRunning,
		rpamodels.RPAExecutionStatusCancelled,
	},
}

// ExecutionStatusTerminal 判断是否终态（终态无出边，不可被覆盖）。
func ExecutionStatusTerminal(status rpamodels.RPAExecutionStatus) bool {
	switch status {
	case rpamodels.RPAExecutionStatusSuccess,
		rpamodels.RPAExecutionStatusFailed,
		rpamodels.RPAExecutionStatusCancelled,
		rpamodels.RPAExecutionStatusTimeout:
		return true
	}
	return false
}

// CanTransition 判断 from → to 是否为合法迁移。
// from == to 的终态幂等重放不算迁移，返回 false（调用方用 CheckTransition
// 区分「幂等接受」与「非法拒绝」）。
func CanTransition(from, to rpamodels.RPAExecutionStatus) bool {
	for _, next := range executionTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

// CheckTransition 校验迁移合法性。
//
// 返回：
//   - (true, nil)  ：from == to 且为终态 → 幂等重放，调用方应跳过写入；
//   - (false, nil) ：合法迁移，调用方继续执行写入；
//   - (false, err) ：非法迁移（终态覆盖、未知状态、无出边跃迁），err 可直接
//     透传给调用方，绝不静默覆盖终态。
func CheckTransition(from, to rpamodels.RPAExecutionStatus) (idempotent bool, err error) {
	if from == to {
		if ExecutionStatusTerminal(from) {
			return true, nil
		}
		// 非终态自环（pending→pending 等）：视为幂等（无信息量重复上报）。
		return true, nil
	}
	if !CanTransition(from, to) {
		return false, fmt.Errorf(
			"非法状态迁移 %s → %s：终态不可覆盖，合法出边 %v",
			from, to, executionTransitions[from])
	}
	return false, nil
}
