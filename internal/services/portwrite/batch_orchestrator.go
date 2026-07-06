package portwrite

import (
	"context"
	"errors"
	"fmt"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// BatchResult 批量写操作结果（D-14 + BATCH-03）。
//
// 三个切片同时存在（即便为空也初始化为 []PortResult{} 而非 nil），
// 保证 JSON 序列化输出 [] 而非 null — 前端无需做 null 守卫。
type BatchResult struct {
	Succeeded []PortResult `json:"succeeded"`
	Failed    []PortResult `json:"failed"`
	Skipped   []PortResult `json:"skipped"`
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
func (s *portWriteServiceImpl) BatchWritePorts(ctx context.Context, req BatchWriteRequest, operator string) (*BatchResult, error) {
	// D-12 detached 30min context (Pitfall #5 mitigation)
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
	for _, portID := range req.PortIDs {
		port, exists := preStateMap[portID]

		if !exists {
			// 端口"消失"（D-13 fallback）— 直接下发
			writeResult, werr := s.executeWrite(ctx, portID, req.DeviceID, req.Action, req.Description, operator, "")
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
		if noopResult := s.checkPreState(&port, req.Action, req.Description); noopResult != nil {
			result.Skipped = append(result.Skipped, *noopResult)
			continue
		}

		// 实际下发
		writeResult, werr := s.executeWrite(ctx, port.ID, port.DeviceID, req.Action, req.Description, operator, port.InterfaceName)
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

	return result, nil
}