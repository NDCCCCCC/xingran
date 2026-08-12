package portwrite

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
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
	// v1.20.1 新增 sentinel（set_access_vlan + port_binding 4 个 validator）
	ErrVlanIdOutOfRange = errors.New("portwrite: vlanId out of range 1-4094")
	ErrBindOpInvalid    = errors.New("portwrite: bind op must be add or remove")
	ErrIPAddressInvalid = errors.New("portwrite: invalid ipv4 address")
	ErrMACAddressInvalid = errors.New("portwrite: invalid mac address")
)

// ipv4Pattern 严格 IPv4 regex（Pitfall 5: 拒绝 0.x.x.x / 255.x.x.x / 非法段长 / 命令分隔符）。
//
// 校验收紧：
//   - 各段值域 [0-255]，且首段不允许 0（避免 0.0.0.0/255.255.255.255 等边界值穿透到设备）
//   - 整段必须是数字 0-255 的合法表示（无前导 0、无超出）
//   - 不含 ; | & 等 shell 命令分隔符（input 自正则字符类已排除）
var ipv4Pattern = regexp.MustCompile(`^(([1-9]?\d|1\d\d|2[0-4]\d|25[0-5])\.){3}([1-9]?\d|1\d\d|2[0-4]\d|25[0-5])$`)

// Action 是 Phase 50 PortAction 的别名，避免双 import 路径冲突。
type Action = portcollection.PortAction

// PortResult 单端口写操作结果（D-14 + D-15 + DB-REFRESH）。
//
// DeviceID 字段用于写后端口状态刷新（refreshPortStatus）：把 writeSinglePort 拿到的 deviceID
// 透传到 caller,使单端口 5 个方法在执行成功后能精确知道要触发哪个设备的 portcollection.CollectDevice。
// 批次路径不需要这个字段（BatchWritePorts 自己持有 req.DeviceID）。
//
// CommandSent 不脱敏（audit 真相源），Phase 52 handler 负责在 HTTP 响应中按字段过滤。
//
// Extra 字段：v1.20.1 新增，用于携带 set_access_vlan 的 vlanId / port_binding 的 (op/ip/mac)
// 上下文供 W3 handler 写入 sys_port_write_audit.after_value JSON（INFRA-01）。
// 5 个 v1.19 方法不写 Extra；SetAccessVlan + PortBinding 写入。
type PortResult struct {
	PortID       string                 `json:"portId"`
	DeviceID     string                 `json:"deviceId,omitempty"`
	Action       Action                 `json:"action"`
	Status       string                 `json:"status"` // "succeeded" | "failed" | "skipped"
	NoOp         bool                   `json:"noOp"`
	CurrentState string                 `json:"currentState,omitempty"`
	Error        string                 `json:"error,omitempty"`
	CommandSent  string                 `json:"commandSent,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

// BatchWriteRequest 批量写请求（D-12 + D-17）。
//
// v1.20.1 新增 4 个可选字段（vlanId / bindOp / ipAddr / macAddr），仅 ActionSetAccessVLAN 和
// ActionPortBinding 路径下使用；其他 action 忽略这 4 个字段。
//   - VLANID：仅 set_access_vlan 使用（service 层 1-4094 范围校验）
//   - BindOp：仅 port_binding 使用（必须是 "add" 或 "remove"，service 层 400 拦截）
//   - IPAddress：仅 port_binding 使用（严格 IPv4 正则校验）
//   - MACAddress：仅 port_binding 使用（可选；空允许，非空时 null MAC 拒绝）
//
// 2026-07-09 防御性绑定标签（CR-01/CR-04 修复）：
//   - DeviceID/Action/PortIDs binding:"required" 在 gin 入口兜底非法请求
//   - VLANID binding:"omitempty,min=1,max=4094" 双层防护（service 仍为真相源）
//   - BindOp binding:"omitempty,oneof=add remove" 防止 "delete" 等 typo 静默触发
//     渲染器 "conservative default → remove" 分支（vendor_port_template.go:73-78）
//   - IPAddress / MACAddress 无法用 gin binding tag 做正则，仍依赖 service 层 validator
type BatchWriteRequest struct {
	DeviceID    string   `json:"deviceId" binding:"required"`
	Action      Action   `json:"action" binding:"required"`
	PortIDs     []string `json:"portIds" binding:"required,min=1,max=50"`
	Description string   `json:"description,omitempty"` // 仅 ActionDescription 使用
	// v1.20.1 set_access_vlan 字段
	VLANID int `json:"vlanId,omitempty" binding:"omitempty,min=1,max=4094"`
	// v1.20.1 port_binding 字段
	BindOp     string `json:"bindOp,omitempty" binding:"omitempty,oneof=add remove"`
	IPAddress  string `json:"ipAddress,omitempty"`
	MACAddress string `json:"macAddress,omitempty"`
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
// 工厂函数 NewPortWriteService 接受 *device.DeviceExecutor / *portcollection.CollectionService
// concrete 指针（Phase 52 router 兼容），内部赋值给 interface 字段。

// portWriteExecutor 暴露 DeviceExecutor.ExecuteCustom 单方法（其余 ExecuteOnDevice/ExecuteMultipleOnDevice 不被本 service 使用）。
type portWriteExecutor interface {
	ExecuteCustom(ctx context.Context, deviceID string, fn func(context.Context, *device.PooledConnection) error, timeout time.Duration) error
}

// portWritePortCollectionSvc 暴露 portcollection.CollectionService.CollectDevice 单方法。
//
// 设计变更（2026-07-08 portwrite-shutdown-db-refresh bug 修复）：
// 原先 PortWriteService 注入的是 *services.DeviceInfoCollectionService.Enqueue(deviceID)，
// 但该服务只采集设备级元信息（Model/SN/板卡/光模块），**不更新 sys_device_port_status**，
// 致 shutdown 成功后 DB 行 admin_status 一直保持 "up"，前端 loadPortStatus() 拉到陈旧数据。
//
// 正确服务是 portcollection.CollectionService.CollectDevice(ctx, deviceID) —— 同步阻塞
// 单设备端口采集（5-15s），完成后 sys_device_port_status 行已反映最新设备状态。
type portWritePortCollectionSvc interface {
	CollectDevice(ctx context.Context, deviceID string) (*portcollection.CollectionResult, error)
}

// PortWriteService 端口写 service 接口（8 方法 — 5 v1.19 单端口 + 2 v1.20.1 单端口 + 1 批量）。
type PortWriteService interface {
	Shutdown(ctx context.Context, portID string, operator string) (*PortResult, error)
	UndoShutdown(ctx context.Context, portID string, operator string) (*PortResult, error)
	SetDescription(ctx context.Context, portID string, desc string, operator string) (*PortResult, error)
	EnableDot1x(ctx context.Context, portID string, operator string) (*PortResult, error)
	DisableDot1x(ctx context.Context, portID string, operator string) (*PortResult, error)
	// v1.20.1 新增方法
	SetAccessVlan(ctx context.Context, portID string, vlanId int, operator string) (*PortResult, error)
	PortBinding(ctx context.Context, portID string, op string, ipAddress string, macAddress string, operator string) (*PortResult, error)
	BatchWritePorts(ctx context.Context, req BatchWriteRequest, operator string) (*BatchResult, error)
}

// portWriteServiceImpl 私有实现。字段类型为 interface 以启用 testify/mock（D-18）。
type portWriteServiceImpl struct {
	db                *gorm.DB
	deviceExecutor    portWriteExecutor
	portCollectionSvc portWritePortCollectionSvc
}

// NewPortWriteService 工厂函数：Phase 52 router 通过 *device.DeviceExecutor / *portcollection.CollectionService
// concrete 类型注入；内部赋值给 interface 字段。
//
// 第三个参数原先是 *services.DeviceInfoCollectionService（采集设备级元信息）——
// 但 4 层 bug 修复后仍无法刷新 sys_device_port_status。2026-07-08 修复：替换为
// portcollection.CollectionService，让 writeSinglePort 成功后调用 CollectDevice(ctx, deviceID)
// 同步刷新端口表，前端 loadPortStatus() 能拉到最新 admin_status/oper_status。
func NewPortWriteService(
	db *gorm.DB,
	deviceExecutor *device.DeviceExecutor,
	portCollectionSvc *portcollection.CollectionService,
) PortWriteService {
	return &portWriteServiceImpl{
		db:                db,
		deviceExecutor:    deviceExecutor,
		portCollectionSvc: portCollectionSvc,
	}
}

// Shutdown 关闭单端口。
func (s *portWriteServiceImpl) Shutdown(ctx context.Context, portID string, operator string) (*PortResult, error) {
	return s.writeAndRefresh(ctx, portID, portcollection.ActionShutdown, "", operator, 0, "", "", "", nil)
}

// UndoShutdown 取消关闭。
func (s *portWriteServiceImpl) UndoShutdown(ctx context.Context, portID string, operator string) (*PortResult, error) {
	return s.writeAndRefresh(ctx, portID, portcollection.ActionUndoShutdown, "", operator, 0, "", "", "", nil)
}

// SetDescription 设置端口描述（desc 由 RenderCommand 入口校验长度 ≤ 80）。
func (s *portWriteServiceImpl) SetDescription(ctx context.Context, portID string, desc string, operator string) (*PortResult, error) {
	return s.writeAndRefresh(ctx, portID, portcollection.ActionDescription, desc, operator, 0, "", "", "", nil)
}

// EnableDot1x 启用 802.1X。
//
// 往返不对称 (Disable/Enable 对 default-user-limit 的处理非对称): disable 自动清除 limit,
// enable 不会自动恢复 → 必须从采集缓存读 limit 值渲染命令,避免 limit 漂成设备默认。
//
// dot1x_user_limit 在本入口从 DB 单字段查询 (避免扩张其他 7 个 action 的方法签名),
// 然后透传到 RenderCommand.PortTemplateParams.Dot1xUserLimit; nil 兜底为 1。
func (s *portWriteServiceImpl) EnableDot1x(ctx context.Context, portID string, operator string) (*PortResult, error) {
	limit, err := s.loadPortDot1xUserLimit(ctx, portID)
	if err != nil {
		applogger.Warnf("[portwrite] EnableDot1x 读取 dot1x_user_limit 失败 (portID=%s) → 兜底 1: %v", portID, err)
		limit = nil
	}
	return s.writeAndRefresh(ctx, portID, portcollection.ActionDot1xEnable, "", operator, 0, "", "", "", limit)
}

// DisableDot1x 停用 802.1X。
func (s *portWriteServiceImpl) DisableDot1x(ctx context.Context, portID string, operator string) (*PortResult, error) {
	return s.writeAndRefresh(ctx, portID, portcollection.ActionDot1xDisable, "", operator, 0, "", "", "", nil)
}

// loadPortDot1xUserLimit 从 sys_device_port_status 单字段读取 dot1x_user_limit。
//
// 仅 EnableDot1x 调用:disable 不需要 limit(设备自动清);enable 必须显式恢复 limit。
// nil/ErrRecordNotFound 都返回 nil(兜底由 renderer fallback 到 1)。
//
// 用 GORM Model + Select + Where 保持与 codebase 一致(裸 SQL 仅在多表 JOIN 等
// 复杂场景使用),Scan 到 *int 自然处理 NULL → nil。migration_204 未跑时仍会
// SQLSTATE 42703,但 EnableDot1x 已有 err → limit=nil → fallback 1 兜底。
func (s *portWriteServiceImpl) loadPortDot1xUserLimit(ctx context.Context, portID string) (*int, error) {
	if portID == "" {
		return nil, fmt.Errorf("portID empty")
	}
	var limit *int
	err := s.db.WithContext(ctx).
		Model(&models.DevicePortStatus{}).
		Select("dot1x_user_limit").
		Where("id = ?", portID).
		Scan(&limit).Error
	if err != nil {
		return nil, err
	}
	return limit, nil
}

// validateVlanIdRange 校验 vlanId ∈ [1, 4094]（VLAN-05 + CR-01 共享入口）。
//
// 返回 nil 表示通过；返回 ErrVlanIdOutOfRange 包装具体值供 caller log 友好。
// 单独抽函数便于单端口 SetAccessVlan 与批量 BatchWritePorts 共用同一份校验语义。
func validateVlanIdRange(vlanId int) error {
	if vlanId < 1 || vlanId > 4094 {
		return fmt.Errorf("%w: %d (must be 1-4094)", ErrVlanIdOutOfRange, vlanId)
	}
	return nil
}

// validateBindOp 校验 op ∈ {"add", "remove"}（BIND-07 + CR-01 共享入口）。
//
// 防止 "delete" 等 typo 静默触发渲染器 "conservative default → remove" 分支
// （vendor_port_template.go:73-78）,避免误删用户 binding。
func validateBindOp(op string) error {
	if op != "add" && op != "remove" {
		return fmt.Errorf("%w: %q (must be add|remove)", ErrBindOpInvalid, op)
	}
	return nil
}

// validateIPAddress 严格 IPv4 regex 校验（BIND-07 + CR-01 共享入口）。
//
// 拒绝越界段 (256.x.x.x) / shell 注入字符（;|空格） / 非数字 / 多余点号。
// 注意 0.0.0.0 / 255.255.255.255 是 RFC 合法的 IPv4 地址,regex 允许通过 —
// device 端会 reject 自身协议层,service 层不预先拒绝这些合法数字形式。
func validateIPAddress(ip string) error {
	if !ipv4Pattern.MatchString(ip) {
		return fmt.Errorf("%w: %q", ErrIPAddressInvalid, ip)
	}
	return nil
}

// validateMACAddress 非空 MAC 归一 + 拒绝 null MAC / 12-hex-char 不匹配注入串。
//
// BIND-07 + CR-01 共享入口;空字符串通过（IP-only binding 允许）。
func validateMACAddress(mac string) error {
	if mac == "" {
		return nil
	}
	normalized := services.NormalizeMACAddress(mac)
	if normalized == "" || normalized == "00:00:00:00:00:00" {
		return fmt.Errorf("%w: %q", ErrMACAddressInvalid, mac)
	}
	return nil
}

// SetAccessVlan 设置 access 端口 PVID（v1.20.1 新增）。
//
// VLAN-05: service 层 validator（前端 InputNumber min/max 是 UX hint,后端为真相源）。
// 在做任何 DB 查询或 SSH 流量之前拒绝越界值,避免对设备下发非法 VLAN 命令。
//
// INFRA-01 / VLAN-06: 成功后把 vlanId 写入 PortResult.Extra,供 W3 handler 写入
// sys_port_write_audit.after_value JSON(vlan 真实值的唯一 carrier)。
func (s *portWriteServiceImpl) SetAccessVlan(ctx context.Context, portID string, vlanId int, operator string) (*PortResult, error) {
	// VLAN-05 validator: 防御性范围校验,任何越界值在入口处直接拒绝
	if err := validateVlanIdRange(vlanId); err != nil {
		return nil, err
	}
	result, err := s.writeAndRefresh(ctx, portID, portcollection.ActionSetAccessVLAN, "", operator, vlanId, "", "", "", nil)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = &PortResult{}
	}
	if result.Extra == nil {
		result.Extra = make(map[string]interface{})
	}
	result.Extra["vlanId"] = vlanId
	return result, nil
}

// PortBinding 端口绑定 (IP + 可选 MAC) 添加或解除（v1.20.1 新增）。
//
// BIND-07: service 层 3 个 validator(顺序严格: op → IP → MAC)。
//
//   - op 必须在 {"add", "remove"} 内,否则 ErrBindOpInvalid
//   - IP 必须通过严格 IPv4 regex,拒绝 0.x.x.x / 255.255.255.255 / 256.x.x.x / shell 分隔符注入
//   - MAC 可空(仅 IP-only binding),非空时先 NormalizeMACAddress 再拒绝 null MAC (`00:00:00:00:00:00`)
//     及 12-hex-char 不匹配的"伪 MAC"(`;reboot` 等命令注入字符串)
//
// INFRA-01: 成功后把 op / ipAddress / 可选 macAddress 写入 PortResult.Extra,供 W3 handler
// 写入 sys_port_write_audit.after_value JSON(binding tuple carrier)。
func (s *portWriteServiceImpl) PortBinding(ctx context.Context, portID string, op string, ipAddress string, macAddress string, operator string) (*PortResult, error) {
	// BIND-07 validator #1: op 必须 add|remove
	if err := validateBindOp(op); err != nil {
		return nil, err
	}
	// BIND-07 validator #2: 严格 IPv4 regex
	if err := validateIPAddress(ipAddress); err != nil {
		return nil, err
	}
	// BIND-07 validator #3: 非空 MAC 先归一再校验 null MAC / 12-hex-char
	if err := validateMACAddress(macAddress); err != nil {
		return nil, err
	}
	result, err := s.writeAndRefresh(ctx, portID, portcollection.ActionPortBinding, "", operator, 0, op, ipAddress, macAddress, nil)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = &PortResult{}
	}
	if result.Extra == nil {
		result.Extra = make(map[string]interface{})
	}
	result.Extra["bindOp"] = op
	result.Extra["ipAddress"] = ipAddress
	if macAddress != "" {
		result.Extra["macAddress"] = macAddress
	}
	return result, nil
}

// writeAndRefresh 单端口方法的公共外壳：writeSinglePort → 成功后调 refreshPortStatus。
//
// 2026-07-08 修复关键设计：原 executeWrite 内部调 s.collectionSvc.Enqueue(deviceID)，
// 触发的是 DeviceInfoCollectionService（采设备级，不碰 sys_device_port_status）——
// 写后 DB 不更新致前端看到 stale 数据。改为 writeAndRefresh 包一层，成功后调
// portcollection.CollectDevice 刷新端口表（后台 goroutine，fire-and-forget，HTTP
// response 不会被 5-15s 的采集阻塞；详见 refreshPortStatus 注释）。
//
// 批次路径走 BatchWritePorts 内部循环 + 末尾一次性 refresh，避免 50 次重复采集同设备。
func (s *portWriteServiceImpl) writeAndRefresh(
	ctx context.Context, portID string, action Action, desc, operator string, vlanId int, bindOp, ipAddr, macAddr string, dot1xUserLimit *int,
) (*PortResult, error) {
	result, err := s.writeSinglePort(ctx, portID, action, desc, operator, vlanId, bindOp, ipAddr, macAddr, dot1xUserLimit)
	if err == nil && result != nil && result.Status == "succeeded" && result.DeviceID != "" {
		s.refreshPortStatus(ctx, result.DeviceID)
	}
	return result, err
}

// writeSinglePort 公共单端口路径：DB 预查询 → PORT-06 pre-state 检测 → executeWrite。
//
// D-13：DB 行不存在（端口"消失"——尚未被 cron 采集或已软删除）fallback 直接下发，
// 避免误报；此路径下 deviceID 为空，会由 executeWrite 触发 ErrDeviceNotFound 返回。
//
// operator 参数：service 层暂不直接消费（Phase 52 handler 决定 audit/operlog 入参格式）；
// 保留参数保证未来扩展性 + 公共接口签名稳定。
//
// vlanId / bindOp / ipAddr / macAddr（v1.20.1 4 个新参数）：
//   - ActionSetAccessVLAN：vlanId 用于 checkPreState 的 VLAN 匹配 + 透传给 RenderCommand.VLANID
//   - ActionPortBinding：bindOp/ipAddr/macAddr 用于 RenderCommand.BindOp/IPAddress/MACAddress（pre-state 跳过）
//   - 5 v1.19 action 路径下这 4 个参数均忽略(由 caller 传 0/""/"")
func (s *portWriteServiceImpl) writeSinglePort(
	ctx context.Context, portID string, action Action, desc, operator string, vlanId int, bindOp, ipAddr, macAddr string, dot1xUserLimit *int,
) (*PortResult, error) {
	var port models.DevicePortStatus
	err := s.db.WithContext(ctx).Where("id = ?", portID).First(&port).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.executeWrite(ctx, portID, "", action, desc, operator, "", vlanId, bindOp, ipAddr, macAddr, dot1xUserLimit)
		}
		return nil, fmt.Errorf("query port: %w", err)
	}

	if noopResult := s.checkPreState(&port, action, desc, vlanId, bindOp, ipAddr, macAddr); noopResult != nil {
		return noopResult, nil
	}

	return s.executeWrite(ctx, port.ID, port.DeviceID, action, desc, operator, port.InterfaceName, vlanId, bindOp, ipAddr, macAddr, dot1xUserLimit)
}

// executeWrite 实际下发写命令：RenderCommand → ExecuteCustom → parseConfigError。
//
// 关键变更（2026-07-08 portwrite-shutdown-db-refresh bug 修复）：
// 原 success path 调 `s.collectionSvc.Enqueue(deviceID)` 触发 DeviceInfoCollectionService，
// 但该服务只采设备级信息（Model/SN/板卡/光模块），不碰 sys_device_port_status——
// 致前端 loadPortStatus() 拉到陈旧 admin_status。
//
// 新方案：executeWrite 不再调任何 refresh。刷新由调用者（writeAndRefresh 单端口方法 /
// BatchWritePorts 批次末尾）通过 refreshPortStatus 集中触发，共享同一设备只采一次。
//
// ★ 2026-07-08 V6 修正：移除 V5 错误的 `end` prefix。
//
//	V5 基于错误假设"设备拒绝嵌套 interface view，需 end 强制回 (config)# 顶层"，
//	且注释误称"华为/H3C 用 quit，Ruijie end 通杀"。经用户三次纠正，真相：
//	  - 锐捷 RGOS 退出 view 用 `exit`（退一级），华为 VRP / H3C Comware 用 `quit`；
//	  - `end` 在三家都会从 config 子视图直接退到 privileged EXEC（#），破坏 scrapli
//	    跨 ExecuteCustom 的 priv 状态跟踪 —— SendConfigs 假设目标 priv=configuration，
//	    发完 end 实际掉到 privileged EXEC，后续 cmd 在错 priv 下执行；
//	  - 锐捷 / 华为 / H3C / 思科 IOS 均支持连续 `interface X | action | interface Y`
//	    嵌套（在 (config-if-X)# 下发 interface Y 直接切换到新接口视图），无需任何
//	    cleanup / 退出前缀。
//
//	真实修复在 ruijie_rgos_patched.yaml：configuration priv 的 prompt pattern 加空格
//	+ 扩容长度（[\+\w.\-@/:+ ]{0,64}），匹配 `Ruijie(config-if-GigabitEthernet 4/18)#`
//	（接口名带空格）。prompt 识别正确后，scrapli 跨 ExecuteCustom 复用 SSH 连接时能
//	正确跟踪 view 切换，连续 interface 嵌套天然成立。
//
//	V5 测试 17/17 真实 shutdown 能通过，是因为 scrapli SendConfigs 前的 AcquirePriv
//	在 end 掉到 privileged EXEC 后会自动 escalate 回 configuration（多一次来回开销），
//	表面 work 但每 port 多 2 条 SSH 命令 + priv 状态抖动，非正确解法。
func (s *portWriteServiceImpl) executeWrite(
	ctx context.Context, portID, deviceID string, action Action, desc, operator, interfaceName string, vlanId int, bindOp, ipAddr, macAddr string, dot1xUserLimit *int,
) (*PortResult, error) {
	if deviceID == "" {
		return nil, ErrDeviceNotFound
	}

	var dev models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", deviceID).First(&dev).Error; err != nil {
		return nil, fmt.Errorf("query device: %w", err)
	}

	cmds, err := portcollection.RenderCommand(dev.Vendor, action, portcollection.PortTemplateParams{
		InterfaceName:  expandInterfaceName(interfaceName, dev.Vendor),
		Description:    desc,
		VLANID:         vlanId,
		BindOp:         bindOp,
		IPAddress:      ipAddr,
		MACAddress:     macAddr,
		Dot1xUserLimit: dot1xUserLimit,
	})
	if err != nil {
		return nil, err
	}

	commandsJoined := strings.Join(cmds, " | ")
	applogger.Infof("[portwrite] executeWrite 下发 portID=%s deviceID=%s vendor=%s interfaceName=%q cmds=%v",
		portID, deviceID, dev.Vendor, interfaceName, cmds)

	var allResponses []*device.Response
	executeErr := s.deviceExecutor.ExecuteCustom(ctx, deviceID, func(execCtx context.Context, pc *device.PooledConnection) error {
		wrapper := pc.GetWrapper()
		// V7 后置 exit（2026-07-08 批量假成功真因）：cmds 末尾追加厂商"退一级 view"
		// 命令（锐捷 exit / 华为 H3C quit），每端口执行后回 (config)# 顶层。实测锐捷
		// RGOS 不支持在 (config-if-X)# 下直接 interface Y（% Unknown command.），批量
		// 复用连接时设备停在上端口 config-if 视图，致本端口 interface 被拒、action 落
		// 到错接口。后置 exit 让下一端口复用时在 config 顶层直接 interface。
		// 选后置而非前置/GetPrompt 探针：首端口正常进 config-if 执行 action，末尾 exit
		// 退 config；对 e2e FileTransport 友好（消费 fixture 末尾 quit 段，不破坏读取顺序，
		// 不像 GetPrompt 探针会消费中间内容致错位）。exit/quit 只退一级（不像 end 退
		// privileged EXEC 破坏 scrapli priv 跟踪）。
		fullCmds := cmds
		if exitCmd := portcollection.VendorExitViewCmd(dev.Vendor); exitCmd != "" {
			fullCmds = append(append([]string{}, cmds...), exitCmd)
		}
		responses, sendErr := wrapper.SendConfigs(fullCmds)
		if sendErr != nil {
			return sendErr
		}
		allResponses = responses
		// 诊断：dump 每条 cmd 的设备实际回显（fullCmds 末尾含 exit cleanup）
		for i, r := range responses {
			cmd := ""
			if i < len(fullCmds) {
				cmd = fullCmds[i]
			}
			applogger.Infof("[portwrite] SendConfigs resp[%d] portID=%s cmd=%q failed=%v result=%q",
				i, portID, cmd, r.Failed, strings.TrimSpace(r.Result))
		}
		return nil
	}, singlePortTimeout)

	applogger.Infof("[portwrite] executeWrite ExecuteCustom 返回 portID=%s err=%v", portID, executeErr)

	if executeErr != nil {
		return &PortResult{
			PortID:      portID,
			DeviceID:    deviceID,
			Action:      action,
			Status:      "failed",
			Error:       executeErr.Error(),
			CommandSent: commandsJoined,
		}, executeErr
	}

	// V7 修复：检查所有业务 response（不只最后一条）。任一 failed=true 或命中错误
	// marker 即判失败 —— 防止 "interface 命令被设备拒绝（failed=true）但末条 action
	// 在错视图被静默接受（failed=false）" 的假成功（2026-07-08 dump 实测真因）。
	// allResponses 为空（mock 测试 ExecuteCustom 不调 fn）时跳过循环，直接 success。
	for i, r := range allResponses {
		if r == nil {
			continue
		}
		if i >= len(cmds) {
			break // 跳过末尾 exit cleanup（fullCmds 比 cmds 多一条 exit，非业务命令不参与失败判定）
		}
		cmd := cmds[i]
		if r.Failed {
			errMsg := fmt.Sprintf("命令 %q 被设备拒绝: %s", cmd, strings.TrimSpace(r.Result))
			return &PortResult{
				PortID:      portID,
				DeviceID:    deviceID,
				Action:      action,
				Status:      "failed",
				Error:       errMsg,
				CommandSent: commandsJoined,
			}, fmt.Errorf("%s", errMsg)
		}
		if parseErr := parseConfigError(r); parseErr != nil {
			return &PortResult{
				PortID:      portID,
				DeviceID:    deviceID,
				Action:      action,
				Status:      "failed",
				Error:       parseErr.Error(),
				CommandSent: commandsJoined,
			}, parseErr
		}
	}

	// 成功路径：仅返回状态，**不**触发采集（避免 50 端口批次重复采集同设备）。
	// 刷新责任由 caller 承担（writeAndRefresh 末尾或 BatchWritePorts 整体结束）。
	result := &PortResult{
		PortID:      portID,
		DeviceID:    deviceID,
		Action:      action,
		Status:      "succeeded",
		CommandSent: commandsJoined,
	}

	// W2 dot1x_enable: 把真正下发的 limit 注入 Extra,handler buildAfterValue
	// 写入 audit.after_value(JSON),供事后审计与对账。nil (设备unlimited)
	// 也写入,显式记录而非漏字段 —— 守 "禁默默吞错"。
	if action == portcollection.ActionDot1xEnable {
		limitForAudit := 1
		if dot1xUserLimit != nil && *dot1xUserLimit > 0 {
			limitForAudit = *dot1xUserLimit
		}
		if result.Extra == nil {
			result.Extra = make(map[string]interface{})
		}
		result.Extra["dot1xUserLimit"] = limitForAudit
	}

	return result, nil
}

// refreshPortStatus 写后端口状态同步刷新（2026-07-08 新增，2026-07-08 改 fire-and-forget）。
//
// 按设备粒度刷新（端口写入的不变式，2026-07-08 用户确认）：
//
//	网络设备 SSH 端口采集的开销 = 1 次 SSH 连接 + 固定几条命令
//	（display interface description / display interface brief + dot1x / port-security），
//	不论只查 1 个端口还是查 N 个端口命令集完全一样；查"1 个端口"反而需要 filter 输出，
//	得不偿失。所以本函数**不**接受 portID 形参，只以 deviceID 为粒度整设备刷新，
//	DB 里 sys_device_port_status 该 device 全部端口行都会被同步更新。
//
// ★ 2026-07-08 修复页面空白 bug 时改为 fire-and-forget：调
// portcollection.CollectionService.CollectDevice 在后台 goroutine 中跑，
// HTTP response 立即返回，**不再被 5-15s 的 SSH 采集阻塞**。
//
//   - 用 context.Background() + 30s timeout 替代请求 ctx（detached 模式），
//     HTTP 断开不影响后台刷新继续完成，对齐 BatchWritePorts batchDetachedTimeout
//     模式（同文件 batch_orchestrator.go:45-46）。
//   - 失败仅日志，不影响 PortResult。失败兜底：下次 cron 'port_collection' (5min) 兜底。
//   - 调用方：writeAndRefresh 单端口方法 / BatchWritePorts 批次末尾。**调用方不要
//     await** —— 本函数立即返回；CollectDevice 在后台跑完即结束。
//
// ⚠ 反模式告警：本函数签名只收 deviceID，请勿扩展为 (ctx, deviceID, portIDs) 等"精准刷新"形态。
//
// ctx 参数保留仅为接口兼容性（caller 现状传请求 ctx）；fire-and-forget 后实际**不**用 ctx，
// 内部 detached 30s context 由 context.Background() 派生。
func (s *portWriteServiceImpl) refreshPortStatus(_ context.Context, deviceID string) {
	if s.portCollectionSvc == nil || deviceID == "" {
		return
	}
	// detached context：30s 超时（典型单设备端口采集 5-15s 留足余量），不继承 HTTP ctx
	// 防止 client disconnect 导致 SSH 采集被中途取消
	refreshCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	go func() {
		defer cancel()
		if _, err := s.portCollectionSvc.CollectDevice(refreshCtx, deviceID); err != nil {
			applogger.Warnf("[portwrite] 写后端口状态刷新失败 [deviceID=%s]: %v", deviceID, err)
		}
	}()
}

// expandInterfaceName 把归一化短名还原成设备 CLI 全名（NormalizeInterfaceName 的反向，仅写命令用）。
//
// 采集时 normalize.InterfaceName 把设备全名折叠成大写短名（GE4/18）入库；但写命令需设备 CLI
// 认的原始全名，否则设备返回 "% Unknown command"。
//   - 锐捷 RGOS：接口类型与编号间带空格（interface GigabitEthernet 1/0/1，display 用全称）
//   - 华为 VRP：用归一化短名（interface GE1/0/0，display 用缩写 GE，不展开全称）
//   - H3C Comware：连写无空格（interface GigabitEthernet1/0/1）
//   2026-07-09 真机验证：华为 S8700 拒绝全称 GigabitEthernet 1/0/0，接受缩写 GE1/0/0（与 display 一致）。
//
// 未识别前缀（Vlan/Loopback 等逻辑接口）原样返回。
// 与 normalize.InterfaceName（正向、采集用）分离，避免反向展开污染采集链路
// （memory normalize-iface-reverse-expand-trap 警告的陷阱）。
func expandInterfaceName(name string, vendor models.DeviceVendor) string {
	if name == "" {
		return name
	}
	type rep struct{ short, full string }
	// 长/易冲突前缀优先（XGE 在 GE 前），避免短前缀误匹配
	reps := []rep{
		{"XGE", "TenGigabitEthernet"},
		{"HGE", "HundredGigE"},
		{"TWE", "TwentyFiveGigE"},
		{"FOE", "FortyGigE"},
		{"GE", "GigabitEthernet"},
		{"FE", "FastEthernet"},
	}
	// 华为 VRP：直接用归一化短名（GE1/0/0），不展开全称。S8700 真机拒绝全称
	// GigabitEthernet 1/0/0（Wrong parameter），接受缩写 GE1/0/0（与 display 输出一致）。
	// 2026-07-09 真机验证。
	if vendor == models.VendorHuawei {
		return name
	}
	upper := strings.ToUpper(name)
	sep := ""
	// 锐捷 RGOS 接口类型与编号间带空格（display 用全称 GigabitEthernet 1/23）；H3C 连写。
	if vendor == models.VendorRuijie {
		sep = " "
	}
	for _, r := range reps {
		if strings.HasPrefix(upper, r.short) {
			rest := name[len(r.short):] // 数字部分，保留原值
			return r.full + sep + rest
		}
	}
	return name
}