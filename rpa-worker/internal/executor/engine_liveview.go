package executor

import (
	"context"
	"fmt"

	"github.com/xingran-next/rpa-worker/internal/browser"
	"github.com/xingran-next/rpa-worker/internal/liveview"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 155 / LIVEVIEW-01~05：引擎侧的实时画面接线点。
//
// 五个接线点（全在这里，engine.go 只留一行调用，尽量少碰 154 的写面）：
//
//	attachLiveView         Acquire 成功后登记可推流 + 初始叠加层情境
//	detachLiveView         Release 前注销并停流（不关 page/browser）
//	refreshLiveContext     动作边界刷新叠加层（步骤/动作/脱敏变量/最近日志）
//	liveActionGate         动作执行前的闸门（接管暂停 / copilot 确认 / manual 全停）
//	liveFailureGate        动作失败时的闸门（copilot/manual 的「失败自动暂停」）
//
// 全部方法 nil 安全：e.liveView == nil（未启用或单测）时短路，行为与 154 一致。
// ============================================================================

// attachLiveView 登记一个可推流的执行（Acquire 之后、导航之前）。
//
// 叠加层初始情境只带已知信息：任务名、Worker、总步数、模式（由 hub 填默认）。
// 变量快照脱敏后写入（LIVEVIEW-04 要求「脱敏变量」，这里的第一道关口）。
func (e *Engine) attachLiveView(task *types.TaskMessage, workerID string, page browser.Page, totalSteps int) {
	if e == nil || e.liveView == nil || task == nil || page == nil {
		return
	}
	meta := &types.LiveViewContext{
		ExecutionID: task.ExecutionID,
		TaskName:    task.TaskName,
		WorkerID:    workerID,
		Step:        0,
		TotalSteps:  totalSteps,
		Status:      string(types.StatusRunning),
		StartedAtMS: types.NowMS(),
	}
	// 变量脱敏：只保留一层键值，敏感键的值替换为 ***（不递归，避免泄露嵌套）。
	if len(task.Variables) > 0 {
		meta.Variables = maskedVariables(task.Variables)
	}
	e.liveView.Attach(task.ExecutionID, page, meta)
}

// detachLiveView 注销并停流（Release 之前）。只停 screencast，不关 page/browser。
func (e *Engine) detachLiveView(executionID string) {
	if e == nil || e.liveView == nil {
		return
	}
	e.liveView.Detach(executionID)
}

// maskedVariables LIVEVIEW-04：脱敏后的字符串快照（值统一格式化为字符串）。
//
// 敏感键（password/token/secret/...）由 executor.MaskSensitive 判定并替换为 ***，
// **原始值绝不进入叠加层**。
func maskedVariables(vars map[string]interface{}) map[string]string {
	if len(vars) == 0 {
		return nil
	}
	out := make(map[string]string, len(vars))
	for k, v := range vars {
		masked := MaskSensitive(k, v)
		// 敏感值统一为 ***（MaskSensitive 已返回该串）。
		out[k] = fmt.Sprintf("%v", masked)
	}
	return out
}

// refreshLiveContext 动作边界刷新叠加层情境（LIVEVIEW-04）。
func (e *Engine) refreshLiveContext(executionID string, action *types.Action, step, total int, status types.ExecutionStatus, message string, variables *VariableScope) {
	if e == nil || e.liveView == nil {
		return
	}
	var vars map[string]string
	if variables != nil {
		vars = maskedVariables(variables.Flatten())
	}
	e.liveView.UpdateContext(executionID, func(ctx *types.LiveViewContext) {
		ctx.Step = step
		ctx.TotalSteps = total
		ctx.Status = string(status)
		if action != nil {
			ctx.ActionType = string(action.Type)
			ctx.ActionDesc = action.Description
		}
		if message != "" {
			ctx.LastLog = message
		}
		if len(vars) > 0 {
			ctx.Variables = vars
		}
	})
}

// liveActionGate 动作执行前的闸门（LIVEVIEW-02/03 的核心接线点）。
//
// 返回 error 时 executor 应中止该动作并走既有错误路径（ctx 取消 → cancelled）。
// 阻塞规则见 liveview.Hub.BeforeAction。
func (e *Engine) liveActionGate(ctx context.Context, executionID string, action *types.Action, failed bool) error {
	if e == nil || e.liveView == nil {
		return nil
	}
	if err := e.liveView.BeforeAction(ctx, executionID, action, failed); err != nil {
		// ctx 取消（用户 Cancel/abort/超时）直接透传，让 RUNTIME-03 的既有
		// 取消路径接管（cancelled 终态）；其它错误（闸门自身故障）包装后上抛。
		if ctx != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("实时画面闸门失败: %w", err)
	}
	return nil
}

// liveStatusUpdate 把执行状态同步到叠加层（暂停/恢复时前端角标变化）。
func (e *Engine) liveStatusUpdate(executionID, status, message string) {
	if e == nil || e.liveView == nil {
		return
	}
	e.liveView.UpdateContext(executionID, func(ctx *types.LiveViewContext) {
		ctx.Status = status
		if message != "" {
			ctx.LastLog = message
		}
	})
}

// logLiveView 便捷日志（减少 engine.go 里的 logger 样板）。
func (e *Engine) logLiveView(msg string, fields ...logger.Field) {
	if e == nil || e.logger == nil {
		return
	}
	e.logger.Info(msg, fields...)
}

// 编译期断言：engine 使用的 hub 类型即 liveview.Hub（避免包间类型漂移）。
var _ = func(h *liveview.Hub) *liveview.Hub { return h }
