package portwrite

import (
	"context"
	"errors"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// BatchResult 批量写操作结果（D-14 + BATCH-03 + DB-REFRESH）。
//
// 三个切片同时存在（即便为空也初始化为 []PortResult{} 而非 nil），
// 保证 JSON 序列化输出 [] 而非 null — 前端无需做 null 守卫。
type BatchResult struct {
	Succeeded []PortResult `json:"succeeded"`
	Failed    []PortResult `json:"failed"`
	Skipped   []PortResult `json:"skipped"`
}

// batchRequiresRefresh 判断批量结果是否需要触发端口状态刷新（2026-07-08 新增）。
//
// 只在有端口确实完成写操作（Succeeded 非空）时刷新 —— fail-fast 全部失败、pre-state
// 全部 NoOp 跳过、empty 路径都不消耗 SSH 流量，无需触碰设备。
// 失败/跳过路径不刷新可避免无意义的 SSH 往返（如果 SSH 已因前面成功连接池活跃）。
func batchRequiresRefresh(result *BatchResult) bool {
	return result != nil && len(result.Succeeded) > 0
}

// BatchWritePorts 批量写主入口（D-12 + D-17）。
//
// 第一行必须是 detached 30min context（context.Background 而非继承 ctx 参数），
// 这是 PROJECT.md Pitfall #5 的核心兜底：批量 SSH 写命令实测可能 5min/批次，
// 若继承 HTTP request ctx 会被 Core.Close() 30s 截止切断。
//
// 入口三连校验（顺序严格：empty → exceeds → deviceID）：
//   - len(PortIDs) == 0           → ErrEmptyBatch（无 SSH 流量）
//   - len(PortIDs) > maxBatchSize → ErrBatchTooLarge（无 SSH 流量）
//   - DeviceID == ""              → errors.New("portwrite: deviceId is required")
//
// D-17 fail-fast：任意端口 executeWrite 返回 transport/device_rejected 错误，立即 break。
// 剩余未执行端口不进任何数组，前端可二次发起。
//
// ★ 2026-07-08 移除 sendInterfaceExitCleanup（Bug #2 假成功真凶）：
// 此前每个非末尾端口后插入 `quit` cleanup cmd 让设备从 (config-if-<X>)# 退到 (config)#，
// 本意是解决"锐捷拒绝嵌套 interface view"假说。实测：quit 命令与设备内部状态机有 race，
// cleanup 期间致后续 2/3 端口的 SendConfigs 假成功（scrapli 看到 prompt 继续，但实际
// shutdown 命令在错 view 上下文里被忽略，设备 admin_status 仍 up）。
//
// 单端口测试对照（GE4/27, 19:48:50 admin=down 真成功）证明 scrapli 本身无问题 —
// 单独 `interface X | shutdown` 一条 SendConfigs 完美生效。batch 假成功 100% 来自
// cleanup 副作用，不是嵌套本身。
//
// 经用户确认 Ruijie RS8607E 实际支持连续 `interface X | action | interface Y | action`
// 嵌套（与 Cisco/H3C/华为一致）— 不再需要 cleanup。本轮直接删 sendInterfaceExitCleanup
// 全部代码，每 port 一次独立 ExecuteCustom + SendConfigs([interface X, action])，
// scrapli 自动处理 prompt 切换 + 跨 ExecuteCustom 复用 SSH 连接池的 view 状态。
func (s *portWriteServiceImpl) BatchWritePorts(_ context.Context, req BatchWriteRequest, operator string) (*BatchResult, error) {
	// D-12 detached 30min context (Pitfall #5 mitigation): 入参 ctx 被替换为
	// 后台 detached 上下文(避免上游 ctx 取消导致批量中断);命名上下文让 SA4009
	// 闭嘴的同时保留意图说明 — 不用 `_` 单纯遮蔽,因为此处业务意图明确。
	ctx, cancel := context.WithTimeout(context.Background(), batchDetachedTimeout)
	defer cancel()

	// D-17 入口校验
	if len(req.PortIDs) == 0 {
		return nil, ErrEmptyBatch
	}
	if len(req.PortIDs) > maxBatchSize {
		return nil, fmt.Errorf("%w: got %d", ErrBatchTooLarge, len(req.PortIDs))
	}
	if req.DeviceID == "" {
		return nil, errors.New("portwrite: deviceId is required")
	}

	// CR-01 + CR-04 共享入口校验（2026-07-09 修复）：
	// 批量路径必须与单端口方法走同一套 4 个 v1.20.1 validator,避免：
	//   - 渲染器 fmt.Sprintf("ip-address %s", p.IPAddress) 被 ";reboot" 等注入
	//   - bindOp="delete" typo 静默触发 "conservative default → remove" 分支
	//   - vlanId 越界落入 per-port Failed[] 而非 entry-rejection 4xx
	// 顺序与单端口 SetAccessVlan / PortBinding 保持一致：vlan → bindOp → IP → MAC。
	// action-specific：仅当 action 匹配时才校验对应字段（其他 action 的字段留空合法）。
	switch req.Action {
	case portcollection.ActionSetAccessVLAN:
		if err := validateVlanIdRange(req.VLANID); err != nil {
			return nil, err
		}
	case portcollection.ActionPortBinding:
		if err := validateBindOp(req.BindOp); err != nil {
			return nil, err
		}
		if err := validateIPAddress(req.IPAddress); err != nil {
			return nil, err
		}
		if err := validateMACAddress(req.MACAddress); err != nil {
			return nil, err
		}
	}

	applogger.Infof("[portwrite-batch] 收到批量请求 deviceID=%s action=%s portCount=%d portIDs=%v",
		req.DeviceID, req.Action, len(req.PortIDs), req.PortIDs)

	// D-13 批量 pre-state 查询（1 次 DB round-trip 拿全部）
	var ports []models.DevicePortStatus
	if err := s.db.WithContext(ctx).
		Where("device_id = ? AND id IN ?", req.DeviceID, req.PortIDs).
		Find(&ports).Error; err != nil {
		return nil, fmt.Errorf("query ports: %w", err)
	}
	preStateMap := make(map[string]models.DevicePortStatus, len(ports))
	for _, p := range ports {
		preStateMap[p.ID] = p
	}

	result := &BatchResult{
		Succeeded: []PortResult{},
		Failed:    []PortResult{},
		Skipped:   []PortResult{},
	}

	// D-17 serial fail-fast loop
	for idx, portID := range req.PortIDs {
		applogger.Infof("[portwrite-batch] (%d/%d) 处理 portID=%s", idx+1, len(req.PortIDs), portID)
		port, exists := preStateMap[portID]

		if !exists {
			// CR-02 跨层防御（Phase 55 leftover sweep）：
			// fallback 路径下前端 req.DeviceID 可能错位（CR-01 根因已修但兜底双保险），
			// 先查 port 真实 deviceID，与 req.DeviceID 不一致则归 Failed 不调 SSH。
			// 正常路径已有 preStateMap 的 WHERE device_id = ? AND id IN ? 隔离零开销。
			var actualPort models.DevicePortStatus
			if err := s.db.WithContext(ctx).
				Where("id = ?", portID).
				First(&actualPort).Error; err != nil {
				// DB 查不到该 port（可能已软删/物理删）— 归 Failed，继续校验下一个
				result.Failed = append(result.Failed, PortResult{
					PortID: portID,
					Error:  fmt.Sprintf("port not found: %v", err),
				})
				continue
			}
			if actualPort.DeviceID != req.DeviceID {
				// port 真实 deviceID 与请求 deviceID 不一致 — 拒绝 SSH 下发
				result.Failed = append(result.Failed, PortResult{
					PortID: portID,
					Error:  "port does not belong to device",
				})
				continue
			}

			// 校验通过 — 端口"消失"（D-13 fallback）但归属正确，直接下发
			writeResult, werr := s.executeWrite(ctx, portID, req.DeviceID, req.Action, req.Description, operator, "", req.VLANID, req.BindOp, req.IPAddress, req.MACAddress, nil)
			if werr != nil {
				if writeResult != nil {
					result.Failed = append(result.Failed, *writeResult)
				}
				break // fail-fast
			}
			if writeResult.NoOp {
				result.Skipped = append(result.Skipped, *writeResult)
				continue
			}
			result.Succeeded = append(result.Succeeded, *writeResult)
			continue
		}

		// pre-state 检测
		if noopResult := s.checkPreState(&port, req.Action, req.Description, req.VLANID, req.BindOp, req.IPAddress, req.MACAddress); noopResult != nil {
			result.Skipped = append(result.Skipped, *noopResult)
			continue
		}

		// 实际下发
		writeResult, werr := s.executeWrite(ctx, port.ID, port.DeviceID, req.Action, req.Description, operator, port.InterfaceName, req.VLANID, req.BindOp, req.IPAddress, req.MACAddress, nil)
		if werr != nil {
			if writeResult != nil {
				result.Failed = append(result.Failed, *writeResult)
			}
			if isTransportError(werr) || isDeviceRejected(werr) {
				break // fail-fast
			}
			break // 其他错误也立即停（D-17 fail-fast 语义一致）
		}
		if writeResult.NoOp {
			result.Skipped = append(result.Skipped, *writeResult)
			continue
		}
		result.Succeeded = append(result.Succeeded, *writeResult)
	}

	applogger.Infof("[portwrite-batch] 循环结束 succeeded=%d failed=%d skipped=%d",
		len(result.Succeeded), len(result.Failed), len(result.Skipped))

	// 2026-07-08 端口刷新：批次内所有写操作完成后，对 req.DeviceID 触发一次性
	// portcollection.CollectDevice —— 后台 goroutine fire-and-forget，HTTP response
	// 不被 5-15s 采集阻塞；详见 refreshPortStatus 注释。失败仅日志不影响返回。
	// 仅有 Succeeded > 0 才刷新（fail-fast 全部失败、pre-state 全部 NoOp 跳过无需触碰设备）。
	if batchRequiresRefresh(result) && req.DeviceID != "" {
		s.refreshPortStatus(ctx, req.DeviceID)
	}

	return result, nil
}