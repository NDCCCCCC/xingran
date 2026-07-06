package portwrite

import (
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/portcollection"
)

// checkPreState PORT-06 pre-state 检测（D-13 + D-14）。
//
// 返回非 nil NoOp PortResult 时表示：DB 状态已匹配 action 目标状态，跳过 SSH 下发。
// 返回 nil 时表示：需继续走 executeWrite SSH 路径。
//
// 5 个 action 全覆盖：
//   - ActionShutdown:   port.AdminStatus == "down" → 已是关闭态
//   - ActionUndoShutdown: port.AdminStatus == "up"   → 已是开启态
//   - ActionDot1xEnable:  port.Dot1xEnabled == true  → 802.1X 已启用
//   - ActionDot1xDisable: port.Dot1xEnabled == false → 802.1X 未启用
//   - ActionDescription:  port.Description == desc   → 描述文本一致
//
// receiver (s *portWriteServiceImpl) 当前未在函数体使用 — Go 方法必须挂 struct；
// 保留 receiver 便于未来注入缓存（D-18 deferred）。
func (s *portWriteServiceImpl) checkPreState(port *models.DevicePortStatus, action Action, desc string) *PortResult {
	switch action {
	case portcollection.ActionShutdown:
		if port.AdminStatus == "down" {
			return &PortResult{
				PortID:       port.ID,
				Action:       action,
				Status:       "skipped",
				NoOp:         true,
				CurrentState: "admin_down",
			}
		}
	case portcollection.ActionUndoShutdown:
		if port.AdminStatus == "up" {
			return &PortResult{
				PortID:       port.ID,
				Action:       action,
				Status:       "skipped",
				NoOp:         true,
				CurrentState: "admin_up",
			}
		}
	case portcollection.ActionDot1xEnable:
		if port.Dot1xEnabled {
			return &PortResult{
				PortID:       port.ID,
				Action:       action,
				Status:       "skipped",
				NoOp:         true,
				CurrentState: "dot1x_enabled",
			}
		}
	case portcollection.ActionDot1xDisable:
		if !port.Dot1xEnabled {
			return &PortResult{
				PortID:       port.ID,
				Action:       action,
				Status:       "skipped",
				NoOp:         true,
				CurrentState: "dot1x_disabled",
			}
		}
	case portcollection.ActionDescription:
		if port.Description == desc {
			return &PortResult{
				PortID:       port.ID,
				Action:       action,
				Status:       "skipped",
				NoOp:         true,
				CurrentState: "description_match",
			}
		}
	}
	return nil
}