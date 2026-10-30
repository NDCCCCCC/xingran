package portwrite

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// checkPreState PORT-06 pre-state 检测（D-13 + D-14）。
//
// 返回非 nil NoOp PortResult 时表示：DB 状态已匹配 action 目标状态，跳过 SSH 下发。
// 返回 nil 时表示：需继续走 executeWrite SSH 路径。
//
// 7 个 action 全覆盖（5 v1.19 + 2 v1.20.1）：
//   - ActionShutdown:         port.AdminStatus == "down" → 已是关闭态
//   - ActionUndoShutdown:     port.AdminStatus == "up"   → 已是开启态
//   - ActionDot1xEnable:      port.Dot1xEnabled == true  → 802.1X 已启用
//   - ActionDot1xDisable:     port.Dot1xEnabled == false → 802.1X 未启用
//   - ActionDescription:      port.Description == desc   → 描述文本一致
//   - ActionSetAccessVLAN:    port.VLAN == vlanId         → PVID 已匹配（VLAN-03）
//   - ActionPortBinding:      返回 nil（跳过 — Pitfall 6，见下方说明）
//
// receiver (s *portWriteServiceImpl) 当前未在函数体使用 — Go 方法必须挂 struct；
// 保留 receiver 便于未来注入缓存（D-18 deferred）。
//
// vlanId / bindOp / ipAddr / macAddr 4 个新增参数为 v1.20.1 set_access_vlan + port_binding 准备：
//   - ActionSetAccessVLAN 使用 vlanId 比对 port.VLAN
//   - ActionPortBinding 因无 DB 模型存 binding tuple（sys_device_port_status 不存），
//     走 SSH 实时查 `display user-bind static all` / `show port-security binding` 需 3-5s/port，
//     加在每写路径上不可接受（千条 binding 时更长）；故本函数返回 nil 让 caller 走 executeWrite。
//     device 端会通过 duplicate detection 防重复 binding，未来 v1.20.x 引入 device_bindings
//     缓存表 (30s TTL) 后可恢复 pre-state NoOp 短路。
//   - 5 个 v1.19 action 忽略这 4 个新参数（Go 允许未使用的方法参数）。
func (s *portWriteServiceImpl) checkPreState(
	port *models.DevicePortStatus,
	action Action,
	desc string,
	vlanId int,
	bindOp, ipAddr, macAddr string,
) *PortResult {
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
	case portcollection.ActionSetAccessVLAN:
		// VLAN-03: 端口当前 PVID == 目标 vlanId → NoOp 短路。
		// port.VLAN 由 port-collection cron 周期采集写入 sys_device_port_status，
		// 是 DB 缓存；无需 SSH 实时查询。
		// 注意: port.VLAN 是 *int (nullable),需先 nil-check 防 panic。
		if port.VLAN != nil && *port.VLAN == vlanId {
			return &PortResult{
				PortID:       port.ID,
				Action:       action,
				Status:       "skipped",
				NoOp:         true,
				CurrentState: "vlan_match",
			}
		}
	case portcollection.ActionPortBinding:
		// BIND-05 + Pitfall 6: 端口绑定表无 DB model (sys_device_port_status 不存 binding tuple),
		// 需 SSH 实时查 `display user-bind static all` / `show port-security binding` 拿当前 binding 列表。
		// 实测 SSH 查表 + parse 3-5s/port (千条 binding 时更长),加在每 write 路径上不可接受。
		// 优化策略: 跳过 pre-state,直接 executeWrite;靠 device 端 duplicate detection 防重复 binding。
		// 未来 v1.20.x: 引入 device_bindings 缓存表 (30s TTL) 后可恢复 pre-state NoOp 短路。
		return nil
	}
	return nil
}