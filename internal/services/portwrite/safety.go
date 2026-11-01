package portwrite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// Phase 123 安全三件套（PORTW-04/05/06）：
//
//   - PORTW-04 快照 + 自动回滚：critical 动作下发前抓快照 + 预生成 reverse 命令；
//     下发失败自动逆序回滚；回滚失败 → rollback_failed 显式告警（不静默）
//   - PORTW-05 并发写互斥：内存锁（单实例并发）+ DB 标记（跨实例），TTL 防残留
//   - PORTW-06 不可达预检：下发前 1s 连通性探测，可配置跳过（审计留痕）

// 哨兵错误（handler 翻译：locked→409，unreachable→503/400 语义见 handler）。
var (
	// ErrPortWriteLocked 端口正被其他写会话持有（%w 内嵌 holder 描述）。
	ErrPortWriteLocked = errors.New("portwrite: port is locked by another write session")
	// ErrDeviceUnreachable 预检探测设备不可达。
	ErrDeviceUnreachable = errors.New("portwrite: device unreachable (preflight failed)")
)

// 锁 TTL：单端口 SSH 操作超时 30s，预留 3 倍余量 + batch 场景逐端口短持锁。
const portWriteLockTTL = 5 * time.Minute

// preflightTimeout PORTW-06 探测超时（1s 级要求）。
const preflightTimeout = 1 * time.Second

// WriteOption 单端口写方法的可选修饰（variadic，向后兼容既有签名与 mock）。
type WriteOption func(*writeOptions)

type writeOptions struct {
	skipPreflight bool
}

// WithSkipPreflight PORTW-06：管理员强制跳过不可达预检（跳过事件由 caller 留痕审计）。
func WithSkipPreflight() WriteOption {
	return func(o *writeOptions) { o.skipPreflight = true }
}

// applyWriteOptions 归并 variadic options。
func applyWriteOptions(opts []WriteOption) writeOptions {
	var o writeOptions
	for _, f := range opts {
		f(&o)
	}
	return o
}

// isCriticalAction PORTW-04 快照覆盖的 critical 动作集合。
func isCriticalAction(action Action) bool {
	return action == portcollection.ActionShutdown || action == portcollection.ActionSetAccessVLAN
}

// ====================== PORTW-05 并发写互斥 ======================

// portLockRegistry 进程内端口写锁（单实例并发层）。
//
// entries[portID] = holder + expiresAt；过期条目惰性清理（Acquire 时顺带删），
// 进程重启 map 清零 —— "重启不残留"由内存层天然满足，DB 层靠 expires_at。
type portLockRegistry struct {
	mu      sync.Mutex
	entries map[string]portLockEntry
}

type portLockEntry struct {
	holder    string
	expiresAt time.Time
}

// globalPortLocks 进程级单例。portWriteServiceImpl 各实例共享（router 每次
// Setup 重建 service，锁必须进程级而非 service 级，否则互斥失效）。
var globalPortLocks = &portLockRegistry{entries: make(map[string]portLockEntry)}

// tryLockMemory 尝试内存加锁；已持锁（未过期）返回 false。
func (r *portLockRegistry) tryLockMemory(portID, holder string, ttl time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.entries[portID]; ok && time.Now().Before(e.expiresAt) {
		return false
	}
	r.entries[portID] = portLockEntry{holder: holder, expiresAt: time.Now().Add(ttl)}
	return true
}

// unlockMemory 释放内存锁（仅当 holder 匹配，防误删他人锁）。
func (r *portLockRegistry) unlockMemory(portID, holder string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.entries[portID]; ok && e.holder == holder {
		delete(r.entries, portID)
	}
}

// currentMemoryHolder 返回当前持锁者（未过期），无锁返回 ""。
func (r *portLockRegistry) currentMemoryHolder(portID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.entries[portID]; ok && time.Now().Before(e.expiresAt) {
		return e.holder
	}
	return ""
}

// acquirePortWriteLock 双层加锁：内存 → DB。
//
// DB 层：先清过期行，再插入；唯一键冲突 → 查现持有者报冲突。
// 任一层冲突即返回 ErrPortWriteLocked（handler 409 + holder 信息）。
func (s *portWriteServiceImpl) acquirePortWriteLock(ctx context.Context, portID, holder string) error {
	if !globalPortLocks.tryLockMemory(portID, holder, portWriteLockTTL) {
		return fmt.Errorf("%w: held by %q (in-process)", ErrPortWriteLocked, globalPortLocks.currentMemoryHolder(portID))
	}
	// 清过期行（TTL 兜底：crash 残留行到期自然可抢占）
	if err := s.db.WithContext(ctx).
		Where("port_id = ? AND expires_at < ?", portID, time.Now()).
		Delete(&models.PortWriteLock{}).Error; err != nil {
		globalPortLocks.unlockMemory(portID, holder)
		return fmt.Errorf("clean expired port locks: %w", err)
	}
	lock := &models.PortWriteLock{PortID: portID, Holder: holder, ExpiresAt: time.Now().Add(portWriteLockTTL)}
	if err := s.db.WithContext(ctx).Create(lock).Error; err != nil {
		globalPortLocks.unlockMemory(portID, holder)
		var existing models.PortWriteLock
		if qerr := s.db.WithContext(ctx).Where("port_id = ?", portID).First(&existing).Error; qerr == nil {
			return fmt.Errorf("%w: held by %q (cross-instance)", ErrPortWriteLocked, existing.Holder)
		}
		return fmt.Errorf("insert port lock: %w", err)
	}
	return nil
}

// releasePortWriteLock 双层释放（DB 行删除失败仅日志——TTL 兜底）。
func (s *portWriteServiceImpl) releasePortWriteLock(ctx context.Context, portID, holder string) {
	globalPortLocks.unlockMemory(portID, holder)
	if err := s.db.WithContext(ctx).
		Where("port_id = ? AND holder = ?", portID, holder).
		Delete(&models.PortWriteLock{}).Error; err != nil {
		applogger.Warnf("[portwrite] 释放 DB 端口锁失败 portID=%s holder=%s: %v", portID, holder, err)
	}
}

// ====================== PORTW-06 不可达预检 ======================

// preflightReachability 下发前 1s 级连通性探测（PORTW-06）。
//
// 复用 device.PingCheck（既有连接池同款 ICMP 探测），不可达返回 ErrDeviceUnreachable。
// skip=true 时跳过（管理员强制），由 caller 负责在审计 Extra 留痕。
func (s *portWriteServiceImpl) preflightReachability(ctx context.Context, dev *models.NetworkDevice, skip bool) error {
	if skip {
		applogger.Warnf("[portwrite] 预检被管理员跳过 deviceID=%s ip=%s", dev.ID, dev.IPAddress)
		return nil
	}
	if dev.IPAddress == "" {
		return fmt.Errorf("%w: device %s has no ip address", ErrDeviceUnreachable, dev.ID)
	}
	// PingCheck 是同步阻塞探测；1s 超时满足"1s 级"要求
	ping := s.pingFunc
	if ping == nil {
		ping = device.PingCheck
	}
	if !ping(dev.IPAddress, preflightTimeout) {
		return fmt.Errorf("%w: %s (%s) icmp probe timeout %s", ErrDeviceUnreachable, dev.IPAddress, dev.DeviceName, preflightTimeout)
	}
	return nil
}

// ====================== PORTW-04 快照 + 自动回滚 ======================

// portBeforeState 快照 JSON 载荷（critical 动作可回滚字段子集）。
type portBeforeState struct {
	AdminStatus  string `json:"adminStatus"`
	Description  string `json:"description"`
	VLAN         *int   `json:"vlan,omitempty"`
	Dot1xEnabled bool   `json:"dot1xEnabled"`
}

// buildPortBeforeState 从端口行抽取快照载荷。
func buildPortBeforeState(port *models.DevicePortStatus) portBeforeState {
	return portBeforeState{
		AdminStatus:  port.AdminStatus,
		Description:  port.Description,
		VLAN:         port.VLAN,
		Dot1xEnabled: port.Dot1xEnabled,
	}
}

// renderRollbackCommands 按快照预生成逆序回滚命令（与 dry-run/写路径共享渲染层）。
//
//   - shutdown → undo_shutdown（复用 RenderCommand，零第二套模板）
//   - set_access_vlan → 恢复快照原 VLAN；原 VLAN 为 nil 时用厂商"清除 PVID"命令
//
// 未覆盖动作返回 nil + false（快照仍可抓，仅不生成回滚命令——保守不误恢复）。
func renderRollbackCommands(vendor models.DeviceVendor, action Action, before portBeforeState, interfaceName string) ([]string, bool) {
	switch action {
	case portcollection.ActionShutdown:
		cmds, err := portcollection.RenderCommand(vendor, portcollection.ActionUndoShutdown, portcollection.PortTemplateParams{
			InterfaceName: expandInterfaceName(interfaceName, vendor),
		})
		if err != nil {
			return nil, false
		}
		return cmds, true
	case portcollection.ActionSetAccessVLAN:
		iface := expandInterfaceName(interfaceName, vendor)
		if before.VLAN != nil {
			cmds, err := portcollection.RenderCommand(vendor, portcollection.ActionSetAccessVLAN, portcollection.PortTemplateParams{
				InterfaceName: iface,
				VLANID:        *before.VLAN,
			})
			if err != nil {
				return nil, false
			}
			return cmds, true
		}
		// 原 PVID 为空 → 厂商清除 PVID 默认值命令（接口视图单条）
		var clear string
		switch vendor {
		case models.VendorHuawei, models.VendorH3C:
			clear = "undo port default vlan"
		case models.VendorRuijie:
			clear = "no switchport access vlan"
		default:
			return nil, false
		}
		return []string{"interface " + iface, clear}, true
	default:
		return nil, false
	}
}

// captureSnapshot critical 动作下发前抓快照 + 预生成回滚命令并落库。
func (s *portWriteServiceImpl) captureSnapshot(ctx context.Context, port *models.DevicePortStatus, dev *models.NetworkDevice, action Action) (*models.PortWriteSnapshot, error) {
	before := buildPortBeforeState(port)
	rbCmds, _ := renderRollbackCommands(dev.Vendor, action, before, port.InterfaceName)
	rbJSON, err := json.Marshal(rbCmds)
	if err != nil {
		return nil, fmt.Errorf("marshal rollback cmds: %w", err)
	}
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return nil, fmt.Errorf("marshal before state: %w", err)
	}
	snap := &models.PortWriteSnapshot{
		PortID:           port.ID,
		DeviceID:         dev.ID,
		InterfaceName:    port.InterfaceName,
		Action:           string(action),
		BeforeState:      string(beforeJSON),
		RollbackCommands: string(rbJSON),
		Status:           "captured",
	}
	if err := s.db.WithContext(ctx).Create(snap).Error; err != nil {
		return nil, fmt.Errorf("insert snapshot: %w", err)
	}
	return snap, nil
}

// rollbackSnapshot 下发失败后自动逆序回滚（PORTW-04）。
//
// 命令序列逆序逐条经 ExecuteCustom 下发；全部成功 → rolled_back；
// 任一失败 → rollback_failed + 显式 error 日志（告警语义：不允许静默）。
// 结果落快照行并返回给 caller 供审计 after_value 引用。
func (s *portWriteServiceImpl) rollbackSnapshot(ctx context.Context, snap *models.PortWriteSnapshot) (rolledBack bool, rollbackErr error) {
	var cmds []string
	if err := json.Unmarshal([]byte(snap.RollbackCommands), &cmds); err != nil || len(cmds) == 0 {
		// 无回滚命令（如 Phase 124 trunk 动作：DB 无原值可恢复）≠ 失败 ——
		// 快照保留供追溯，status=no_rollback 区别于 rollback_failed 告警。
		s.db.WithContext(ctx).Model(&models.PortWriteSnapshot{}).Where("id = ?", snap.ID).
			Update("status", "no_rollback")
		return false, nil
	}
	execErr := s.deviceExecutor.ExecuteCustom(ctx, snap.DeviceID, func(execCtx context.Context, pc *device.PooledConnection) error {
		wrapper := pc.GetWrapper()
		// 逆序回滚：后加的配置先撤销
		for i := len(cmds) - 1; i >= 0; i-- {
			responses, err := wrapper.SendConfigs([]string{cmds[i]})
			if err != nil {
				return fmt.Errorf("rollback cmd %q: %w", cmds[i], err)
			}
			for _, r := range responses {
				if r != nil && r.Failed {
					return fmt.Errorf("rollback cmd %q rejected: %s", cmds[i], r.Result)
				}
			}
		}
		return nil
	}, singlePortTimeout)
	if execErr != nil {
		s.markSnapshotFailed(ctx, snap, execErr.Error())
		applogger.Errorf("[portwrite] ⚠ 回滚失败（显式告警，不允许静默）portID=%s action=%s err=%v", snap.PortID, snap.Action, execErr)
		return false, execErr
	}
	s.db.WithContext(ctx).Model(&models.PortWriteSnapshot{}).Where("id = ?", snap.ID).Update("status", "rolled_back")
	return true, nil
}

// markSnapshotFailed 快照标记 rollback_failed（告警状态落库）。
func (s *portWriteServiceImpl) markSnapshotFailed(ctx context.Context, snap *models.PortWriteSnapshot, msg string) {
	if len(msg) > 500 {
		msg = msg[:500]
	}
	s.db.WithContext(ctx).Model(&models.PortWriteSnapshot{}).Where("id = ?", snap.ID).
		Updates(map[string]interface{}{"status": "rollback_failed", "rollback_error": msg})
}

// markSnapshotKept 下发成功后快照保留（供追溯）。
func (s *portWriteServiceImpl) markSnapshotKept(ctx context.Context, snapID string) {
	if snapID == "" {
		return
	}
	s.db.WithContext(ctx).Model(&models.PortWriteSnapshot{}).Where("id = ?", snapID).Update("status", "kept")
}
