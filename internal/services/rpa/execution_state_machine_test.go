package rpa

import (
	"testing"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
)

// RUNTIME-05：状态迁移矩阵全量单测 —— 合法 + 非法两组，覆盖全部状态对。

var allStatuses = []rpamodels.RPAExecutionStatus{
	rpamodels.RPAExecutionStatusPending,
	rpamodels.RPAExecutionStatusRunning,
	rpamodels.RPAExecutionStatusPaused,
	rpamodels.RPAExecutionStatusSuccess,
	rpamodels.RPAExecutionStatusFailed,
	rpamodels.RPAExecutionStatusCancelled,
	rpamodels.RPAExecutionStatusTimeout,
}

// expectedLegal 显式列出全部合法迁移（与 execution_state_machine.go 注释一致）。
var expectedLegal = map[[2]rpamodels.RPAExecutionStatus]bool{
	{rpamodels.RPAExecutionStatusPending, rpamodels.RPAExecutionStatusRunning}:   true,
	{rpamodels.RPAExecutionStatusPending, rpamodels.RPAExecutionStatusCancelled}: true,
	{rpamodels.RPAExecutionStatusRunning, rpamodels.RPAExecutionStatusSuccess}:   true,
	{rpamodels.RPAExecutionStatusRunning, rpamodels.RPAExecutionStatusFailed}:    true,
	{rpamodels.RPAExecutionStatusRunning, rpamodels.RPAExecutionStatusCancelled}: true,
	{rpamodels.RPAExecutionStatusRunning, rpamodels.RPAExecutionStatusTimeout}:   true,
	{rpamodels.RPAExecutionStatusRunning, rpamodels.RPAExecutionStatusPaused}:    true,
	{rpamodels.RPAExecutionStatusPaused, rpamodels.RPAExecutionStatusRunning}:    true,
	{rpamodels.RPAExecutionStatusPaused, rpamodels.RPAExecutionStatusCancelled}:  true,
}

// TestStateMachine_LegalTransitions 全部合法迁移必须通过 CheckTransition。
func TestStateMachine_LegalTransitions(t *testing.T) {
	count := 0
	for _, from := range allStatuses {
		for _, to := range allStatuses {
			if !expectedLegal[[2]rpamodels.RPAExecutionStatus{from, to}] {
				continue
			}
			if !CanTransition(from, to) {
				t.Fatalf("合法迁移被拒: %s → %s", from, to)
			}
			idempotent, err := CheckTransition(from, to)
			if err != nil {
				t.Fatalf("CheckTransition(%s→%s) 返回错误: %v", from, to, err)
			}
			if idempotent {
				t.Fatalf("合法迁移 %s→%s 不应判为幂等", from, to)
			}
			count++
		}
	}
	if count != len(expectedLegal) {
		t.Fatalf("合法迁移用例数 %d 与迁移表 %d 不一致", count, len(expectedLegal))
	}
}

// TestStateMachine_IllegalTransitions 其余全部状态对（含终态覆盖、终态自环幂等）
// 必须按预期拒绝或幂等接受。
func TestStateMachine_IllegalTransitions(t *testing.T) {
	for _, from := range allStatuses {
		for _, to := range allStatuses {
			pair := [2]rpamodels.RPAExecutionStatus{from, to}
			if expectedLegal[pair] {
				continue
			}
			idempotent, err := CheckTransition(from, to)
			if from == to {
				// 自环：幂等接受（终态重放不覆盖、非终态重放不报错）
				if err != nil || !idempotent {
					t.Fatalf("自环 %s→%s 应幂等接受，got (idempotent=%v, err=%v)", from, to, idempotent, err)
				}
				continue
			}
			// 非法迁移：必须拒绝且错误明确
			if err == nil {
				t.Fatalf("非法迁移未被拒绝: %s → %s", from, to)
			}
			if idempotent {
				t.Fatalf("非法迁移不应判为幂等: %s → %s", from, to)
			}
		}
	}
}

// TestStateMachine_TerminalNotOverwritten 终态不可被任何其它状态覆盖（重点：
// cancelled 不得被后续 success 覆盖 —— RUNTIME-05 核心诉求）。
func TestStateMachine_TerminalNotOverwritten(t *testing.T) {
	terminals := []rpamodels.RPAExecutionStatus{
		rpamodels.RPAExecutionStatusSuccess,
		rpamodels.RPAExecutionStatusFailed,
		rpamodels.RPAExecutionStatusCancelled,
		rpamodels.RPAExecutionStatusTimeout,
	}
	for _, term := range terminals {
		if !ExecutionStatusTerminal(term) {
			t.Fatalf("%s 应判定为终态", term)
		}
		for _, to := range allStatuses {
			if to == term {
				continue
			}
			if _, err := CheckTransition(term, to); err == nil {
				t.Fatalf("终态覆盖未被拒绝: %s → %s", term, to)
			}
		}
	}
}

// TestStateMachine_IdempotentReplayCancelled 终态 cancelled 的幂等重放用例
// （RUNTIME-05「幂等重放」验收）：worker 在 Cancel 后仍上报 cancelled 终态，
// 应幂等接受而非报错。
func TestStateMachine_IdempotentReplayCancelled(t *testing.T) {
	cancelled := rpamodels.RPAExecutionStatusCancelled
	idempotent, err := CheckTransition(cancelled, cancelled)
	if err != nil || !idempotent {
		t.Fatalf("cancelled 幂等重放失败: idempotent=%v, err=%v", idempotent, err)
	}
}
