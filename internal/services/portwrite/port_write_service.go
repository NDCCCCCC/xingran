package portwrite

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/portcollection"
	"gorm.io/gorm"
)

// 哨兵错误（D-17 入口校验 + D-13 fallback 路径）。
//
// 使用 portwrite: 前缀与 Phase 50 portcollection: 前缀风格一致，便于日志聚合过滤。
var (
	ErrBatchTooLarge = errors.New("portwrite: batch exceeds max size of 50")
	ErrEmptyBatch    = errors.New("portwrite: batch is empty")
	ErrMixedDevices  = errors.New("portwrite: batch contains ports from different devices")
	ErrPortNotFound  = errors.New("portwrite: port not found")
	ErrDeviceNotFound = errors.New("portwrite: device not found")
)

// Action 是 Phase 50 PortAction 的别名，避免双 import 路径冲突。
type Action = portcollection.PortAction

// PortResult 单端口写操作结果（D-14 + D-15）。
//
// CommandSent 不脱敏（audit 真相源），Phase 52 handler 负责在 HTTP 响应中按字段过滤。
type PortResult struct {
	PortID       string `json:"portId"`
	Action       Action `json:"action"`
	Status       string `json:"status"` // "succeeded" | "failed" | "skipped"
	NoOp         bool   `json:"noOp"`
	CurrentState string `json:"currentState,omitempty"`
	Error        string `json:"error,omitempty"`
	CommandSent  string `json:"commandSent,omitempty"`
}

// BatchWriteRequest 批量写请求（D-12 + D-17）。
type BatchWriteRequest struct {
	DeviceID    string   `json:"deviceId"`
	Action      Action   `json:"action"`
	PortIDs     []string `json:"portIds"`
	Description string   `json:"description,omitempty"` // 仅 ActionDescription 使用
}

// 超时常量（D-11 + D-12）。
const (
	singlePortTimeout    = 30 * time.Second // 单端口 SSH 操作超时
	batchPortTimeout     = 60 * time.Second // 批量内每端口 SSH 操作超时（预留，本 phase 不直接使用）
	batchDetachedTimeout = 30 * time.Minute // 批量 detached context 总超时（核心 Pitfall #5 兜底）
	maxBatchSize         = 50               // 批量入口上限（D-17）
)

// 内部接口（D-18 + RESEARCH Open Questions #1+#2 — 为 testify/mock 提供注入点）。
//
// 工厂函数 NewPortWriteService 接受 *device.DeviceExecutor / *services.DeviceInfoCollectionService
// concrete 指针（Phase 52 router 兼容），内部赋值给 interface 字段。

// portWriteExecutor 暴露 DeviceExecutor.ExecuteCustom 单方法（其余 ExecuteOnDevice/ExecuteMultipleOnDevice 不被本 service 使用）。
type portWriteExecutor interface {
	ExecuteCustom(ctx context.Context, deviceID string, fn func(context.Context, *device.PooledConnection) error, timeout time.Duration) error
}

// portWriteCollectionSvc 暴露 DeviceInfoCollectionService.Enqueue 单方法。
type portWriteCollectionSvc interface {
	Enqueue(deviceID string) error
}

// PortWriteService 端口写 service 接口（6 方法 — 5 单端口 + 1 批量）。
type PortWriteService interface {
	Shutdown(ctx context.Context, portID string, operator string) (*PortResult, error)
	UndoShutdown(ctx context.Context, portID string, operator string) (*PortResult, error)
	SetDescription(ctx context.Context, portID string, desc string, operator string) (*PortResult, error)
	EnableDot1x(ctx context.Context, portID string, operator string) (*PortResult, error)
	DisableDot1x(ctx context.Context, portID string, operator string) (*PortResult, error)
	BatchWritePorts(ctx context.Context, req BatchWriteRequest, operator string) (*BatchResult, error)
}

// portWriteServiceImpl 私有实现。字段类型为 interface 以启用 testify/mock（D-18）。
type portWriteServiceImpl struct {
	db             *gorm.DB
	deviceExecutor portWriteExecutor
	collectionSvc  portWriteCollectionSvc
}

// NewPortWriteService 工厂函数：Phase 52 router 通过 *device.DeviceExecutor / *services.DeviceInfoCollectionService
// concrete 类型注入；内部赋值给 interface 字段。
func NewPortWriteService(
	db *gorm.DB,
	deviceExecutor *device.DeviceExecutor,
	collectionSvc *services.DeviceInfoCollectionService,
) PortWriteService {
	return &portWriteServiceImpl{
		db:             db,
		deviceExecutor: deviceExecutor,
		collectionSvc:  collectionSvc,
	}
}

// Shutdown 关闭单端口。
func (s *portWriteServiceImpl) Shutdown(ctx context.Context, portID string, operator string) (*PortResult, error) {
	return s.writeSinglePort(ctx, portID, portcollection.ActionShutdown, "", operator)
}

// UndoShutdown 取消关闭。
func (s *portWriteServiceImpl) UndoShutdown(ctx context.Context, portID string, operator string) (*PortResult, error) {
	return s.writeSinglePort(ctx, portID, portcollection.ActionUndoShutdown, "", operator)
}

// SetDescription 设置端口描述（desc 由 RenderCommand 入口校验长度 ≤ 80）。
func (s *portWriteServiceImpl) SetDescription(ctx context.Context, portID string, desc string, operator string) (*PortResult, error) {
	return s.writeSinglePort(ctx, portID, portcollection.ActionDescription, desc, operator)
}

// EnableDot1x 启用 802.1X。
func (s *portWriteServiceImpl) EnableDot1x(ctx context.Context, portID string, operator string) (*PortResult, error) {
	return s.writeSinglePort(ctx, portID, portcollection.ActionDot1xEnable, "", operator)
}

// DisableDot1x 停用 802.1X。
func (s *portWriteServiceImpl) DisableDot1x(ctx context.Context, portID string, operator string) (*PortResult, error) {
	return s.writeSinglePort(ctx, portID, portcollection.ActionDot1xDisable, "", operator)
}

// writeSinglePort 公共单端口路径：DB 预查询 → PORT-06 pre-state 检测 → executeWrite。
//
// D-13：DB 行不存在（端口"消失"——尚未被 cron 采集或已软删除）fallback 直接下发，
// 避免误报；此路径下 deviceID 为空，会由 executeWrite 触发 ErrDeviceNotFound 返回。
//
// operator 参数：service 层暂不直接消费（Phase 52 handler 决定 audit/operlog 入参格式）；
// 保留参数保证未来扩展性 + 公共接口签名稳定。
func (s *portWriteServiceImpl) writeSinglePort(
	ctx context.Context, portID string, action Action, desc, operator string,
) (*PortResult, error) {
	var port models.DevicePortStatus
	err := s.db.WithContext(ctx).Where("id = ?", portID).First(&port).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.executeWrite(ctx, portID, "", action, desc, operator, "")
		}
		return nil, fmt.Errorf("query port: %w", err)
	}

	if noopResult := s.checkPreState(&port, action, desc); noopResult != nil {
		return noopResult, nil
	}

	return s.executeWrite(ctx, port.ID, port.DeviceID, action, desc, operator, port.InterfaceName)
}

// executeWrite 实际下发写命令：RenderCommand → ExecuteCustom → parseConfigError → success-only Enqueue。
//
// Pitfall #6：Enqueue 仅在成功路径触发；executeErr != nil 或 parseErr != nil 都直接返回，
// 不触发采集（避免 Enqueue 风暴 + 1-2s 后无意义的后台轮询）。
func (s *portWriteServiceImpl) executeWrite(
	ctx context.Context, portID, deviceID string, action Action, desc, operator, interfaceName string,
) (*PortResult, error) {
	if deviceID == "" {
		return nil, ErrDeviceNotFound
	}

	var dev models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", deviceID).First(&dev).Error; err != nil {
		return nil, fmt.Errorf("query device: %w", err)
	}

	cmds, err := portcollection.RenderCommand(dev.Vendor, action, portcollection.PortTemplateParams{
		InterfaceName: interfaceName,
		Description:   desc,
	})
	if err != nil {
		return nil, err
	}

	commandsJoined := strings.Join(cmds, " | ")

	var lastResp *device.Response
	executeErr := s.deviceExecutor.ExecuteCustom(ctx, deviceID, func(execCtx context.Context, pc *device.PooledConnection) error {
		wrapper := pc.GetWrapper()
		responses, sendErr := wrapper.SendConfigs(cmds)
		if sendErr != nil {
			return sendErr
		}
		if len(responses) > 0 {
			lastResp = responses[len(responses)-1]
		}
		return nil
	}, singlePortTimeout)

	if executeErr != nil {
		return &PortResult{
			PortID:      portID,
			Action:      action,
			Status:      "failed",
			Error:       executeErr.Error(),
			CommandSent: commandsJoined,
		}, executeErr
	}

	// parseConfigError 仅在 lastResp 非 nil 时调用。
	// lastResp == nil 表示 executor 未返回任何 response（如 mock 测试场景或 fn 内未跑 SendConfigs），
	// 这种情况下视为 success — executor 已返回 nil error 即代表操作完成。
	if lastResp != nil {
		if parseErr := parseConfigError(lastResp); parseErr != nil {
			return &PortResult{
				PortID:      portID,
				Action:      action,
				Status:      "failed",
				Error:       parseErr.Error(),
				CommandSent: commandsJoined,
			}, parseErr
		}
	}

	// 成功路径：D-15 + AUDIT-04，触发后台采集（fire-and-forget）。
	if s.collectionSvc != nil {
		_ = s.collectionSvc.Enqueue(deviceID)
	}

	return &PortResult{
		PortID:      portID,
		Action:      action,
		Status:      "succeeded",
		CommandSent: commandsJoined,
	}, nil
}