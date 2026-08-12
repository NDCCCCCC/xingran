package services

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/normalize"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// MACEventType MAC地址历史事件类型（已移至 models.DeviceMACHistory，此处保留别名）
type MACEventType = models.MACEventType

// 事件类型常量（从 models 包引用）
const (
	EventAppeared    MACEventType = models.EventAppeared
	EventDisappeared MACEventType = models.EventDisappeared
	EventMoved       MACEventType = models.EventMoved
	EventVLANChanged MACEventType = models.EventVLANChanged
)

// MACEvent MAC事件键（用于状态对比）
// 使用复合键：MAC地址 + 接口名 + VLAN ID
type MACEvent struct {
	MACAddress    string
	InterfaceName string
	VLANID        *int
}

// MACHistoryService MAC地址历史记录服务接口
type MACHistoryService interface {
	// BuildMACStateMap 构建MAC状态映射
	// 将DeviceMACAddress切片转换为map[MACEvent]*models.DeviceMACAddress
	BuildMACStateMap(macList []models.DeviceMACAddress) map[MACEvent]*models.DeviceMACAddress

	// RecordMACChange 记录MAC地址变更
	// 对比新旧状态，识别变更类型并写入历史表
	RecordMACChange(ctx context.Context, device *models.NetworkDevice, oldMACs, newMACs map[MACEvent]*models.DeviceMACAddress) error

	// MergeFlappingRecords 合并MAC flapping记录
	// 查询同一设备的连续相同MAC状态记录，更新last_seen避免重复
	MergeFlappingRecords(ctx context.Context, deviceID string) error

	// CleanupAllDevicesFlapping 全量清理所有设备的 MAC flapping 记录(2026-06-30 新增)
	// 用途:Merge 算法放宽后,Retroactive 清理已有 flapping 数据。
	// 建议由 cron / 一次性任务调用(如每周一次),避免在线查询时合并数据造成延迟。
	CleanupAllDevicesFlapping(ctx context.Context) (int, error)

	// MergeByTransitions 按状态转换点合并 flapping 记录(2026-07-01 新增)。
	// 用户原话:"仅保留设备或接口有变化的记录,删除其余的所有记录"。
	// 按 (device_id, mac_address) 分组,仅保留位置签名 (interface_name, vlan_id)
	// 与上一保留记录不同的转换点。同位置签名下的反复出现/消失/迁移全部删除。
	// 与 CleanupAllDevicesFlapping (2h 时间窗口) 的区别:本工具按位置签名判定。
	MergeByTransitions(ctx context.Context) (int64, error)

	// PurgeMeaninglessRecords 全量清理 sys_device_mac_history 中的"无意义记录"
	// (2026-06-30 新增)。保留策略:
	//   - KEEP: 所有 moved 事件
	//   - KEEP: 所有 vlan_changed 事件
	//   - KEEP: 每个 (device_id, mac_address) 第一条 appeared(first_seen ASC)
	//   - DELETE: 其他 appeared / 所有 disappeared
	//
	// 实现细节:
	//   - 执行前自动创建备份表 sys_device_mac_history_purge_backup_YYYYMMDD_HHMMSS,
	//     包含所有当前行,可用于回滚
	//   - 事务内执行 DELETE + 自校验受影响行数
	//   - 返回 (deletedCount, backupTableName, error)
	//
	// 建议由 cron 任务每月调用一次(2026-06-30 用户决策),防新数据再次堆积。
	// 用户原始决策:"只有 mac 出现到其他交换机端口了,终端移动了位置,才记录"。
	PurgeMeaninglessRecords(ctx context.Context, dryRun bool) (deletedCount int64, backupTable string, err error)
}

// macHistoryServiceImpl MAC历史服务实现
type macHistoryServiceImpl struct {
	db *gorm.DB
}

// NewMACHistoryService 创建MAC历史服务
func NewMACHistoryService(db *gorm.DB) MACHistoryService {
	return &macHistoryServiceImpl{db: db}
}

// NormalizeMACAddress 委托共享实现（见 internal/services/mac_normalize.go）
//
// 历史背景：2026-07-01 port-mac-format-unify 之前此文件有本地实现（已删除）。
// 统一入口见 services.NormalizeMACAddress（mac_normalize.go），与 mac_collector 共享。

// BuildMACStateMap 构建MAC状态映射（导出方法供其他服务使用）
// 将DeviceMACAddress切片转换为map[MACEvent]*models.DeviceMACAddress
// 关键：在构建时归一化 MAC 地址 + 接口名格式
//
// 2026-07-01 修复: 此前仅归一 MAC,InterfaceName 原样使用。当 DB 历史数据是全称
// (GigabitEthernet0/0/1) 而新采集是短名(GE0/0/1)时,oldState/newState 的 InterfaceName
// 不等会被误判为 moved 事件,产生虚假 MAC 轨迹。且全称 InterfaceName 直接写入
// sys_device_mac_history,导致轨迹表接口名都是全称。归一化后两端一致,虚假 moved 消失。
func (s *macHistoryServiceImpl) BuildMACStateMap(macList []models.DeviceMACAddress) map[MACEvent]*models.DeviceMACAddress {
	stateMap := make(map[MACEvent]*models.DeviceMACAddress)

	for i := range macList {
		mac := &macList[i]

		// 归一化 MAC 地址(大写+冒号) + 接口名(大写短名)
		normalizedMAC := NormalizeMACAddress(mac.MACAddress)
		normalizedIface := normalize.InterfaceName(mac.InterfaceName)

		eventKey := MACEvent{
			MACAddress:    normalizedMAC,
			InterfaceName: normalizedIface,
			VLANID:        mac.VLANID,
		}

		// 更新为归一化格式(调用方 Step9 用 macRecordsSlice 构建 newState,保持一致)
		mac.MACAddress = normalizedMAC
		mac.InterfaceName = normalizedIface

		stateMap[eventKey] = mac
	}

	return stateMap
}

// RecordMACChange 记录MAC地址变更（CONTEXT.md D-01, D-02, D-03）
// 对比新旧状态，识别4种变更类型并写入历史表
//
// 设计原则(2026-07-01 修正):
//   - L1:用 normalized MAC 集合判定"MAC 是否在设备上",取代 (mac, interface, vlan)
//     三元组比对,避免漏采周期产生冗余 disappeared/appeared 配对
//   - 不再做 L2 history re-check 同端口跳过 — 该优化过激,会把真实同端口
//     反复出现/消失/迁移也吞掉(2026-07-01 用户反馈的 bug 现象)
//   - flapping 合并由 MergeFlappingRecords (2h 窗口) 兜底
//   - 长尾清理由 PurgeMeaninglessRecords 兜底(只保留每 MAC 首条 appeared)
//
// 关联: [[mac-address-normalize-returns-colon-format]]
func (s *macHistoryServiceImpl) RecordMACChange(ctx context.Context, device *models.NetworkDevice, oldMACs, newMACs map[MACEvent]*models.DeviceMACAddress) error {
	// 构建历史记录列表
	var historyRecords []*models.DeviceMACHistory
	collectionTime := time.Now()

	// L1:normalized MAC 集合判定 — 取代旧 (mac, iface, vlan) 三元组比对
	oldMACNormSet := make(map[string]bool, len(oldMACs))
	for _, oldMAC := range oldMACs {
		oldMACNormSet[NormalizeMACAddress(oldMAC.MACAddress)] = true
	}
	newMACNormSet := make(map[string]bool, len(newMACs))
	for _, newMAC := range newMACs {
		newMACNormSet[NormalizeMACAddress(newMAC.MACAddress)] = true
	}

	// 1. appeared: MAC 在设备上完全新增(不在 oldMACs 任何端口)
	for eventKey := range newMACs {
		normMAC := NormalizeMACAddress(eventKey.MACAddress)
		if oldMACNormSet[normMAC] {
			continue
		}
		history := &models.DeviceMACHistory{
			DeviceID:           device.ID,
			DeviceNameSnapshot: device.DeviceName,
			MACAddress:         eventKey.MACAddress,
			InterfaceName:      eventKey.InterfaceName,
			VLANID:             eventKey.VLANID,
			EventType:          EventAppeared,
			FirstSeen:          collectionTime,
			LastSeen:           collectionTime,
			CollectedAt:        collectionTime,
		}
		historyRecords = append(historyRecords, history)

		applogger.Debugf("[MAC历史] %s: appeared MAC=%s interface=%s vlan=%v",
			device.DeviceName, eventKey.MACAddress, eventKey.InterfaceName, eventKey.VLANID)
	}

	// 2. disappeared: MAC 在设备上完全消失(不在 newMACs 任何端口)
	for eventKey, oldMAC := range oldMACs {
		normMAC := NormalizeMACAddress(oldMAC.MACAddress)
		if newMACNormSet[normMAC] {
			continue
		}
		history := &models.DeviceMACHistory{
			DeviceID:           device.ID,
			DeviceNameSnapshot: device.DeviceName,
			MACAddress:         eventKey.MACAddress,
			InterfaceName:      eventKey.InterfaceName,
			VLANID:             eventKey.VLANID,
			EventType:          EventDisappeared,
			FirstSeen:          oldMAC.CollectedAt,
			LastSeen:           collectionTime,
			CollectedAt:        collectionTime,
		}
		historyRecords = append(historyRecords, history)

		applogger.Debugf("[MAC历史] %s: disappeared MAC=%s interface=%s vlan=%v",
			device.DeviceName, eventKey.MACAddress, eventKey.InterfaceName, eventKey.VLANID)
	}

	// 3. moved: MAC地址相同但接口不同
	// 4. vlan_changed: 接口相同但VLAN不同
	// CR-02 fix: 优化O(n²)复杂度为O(n) - 构建MAC地址到事件的映射
	// 构建新MAC的映射: MAC地址 -> 事件列表
	newMACByAddress := make(map[string][]MACEvent)
	for eventKey, mac := range newMACs {
		normMAC := NormalizeMACAddress(mac.MACAddress)
		newMACByAddress[normMAC] = append(newMACByAddress[normMAC], eventKey)
	}

	// 遍历旧MAC，通过映射查找匹配的新MAC（O(1)查找）
	for oldEventKey, oldMAC := range oldMACs {
		oldMACNorm := NormalizeMACAddress(oldMAC.MACAddress)

		if matchingEvents, exists := newMACByAddress[oldMACNorm]; exists {
			// 遍历所有匹配的事件
			for _, newEventKey := range matchingEvents {
				// vlan_changed: 接口相同但VLAN不同
				if oldEventKey.InterfaceName == newEventKey.InterfaceName {
					if !vlanEqual(oldEventKey.VLANID, newEventKey.VLANID) {
						history := &models.DeviceMACHistory{
							DeviceID:           device.ID,
							DeviceNameSnapshot: device.DeviceName,
							MACAddress:         oldMACNorm,
							InterfaceName:      oldEventKey.InterfaceName,
							VLANID:             newEventKey.VLANID,
							EventType:          EventVLANChanged,
							FirstSeen:          oldMAC.CollectedAt,
							LastSeen:           collectionTime,
							CollectedAt:        collectionTime,
						}
						historyRecords = append(historyRecords, history)

						applogger.Debugf("[MAC历史] %s: vlan_changed MAC=%s interface=%s oldVlan=%v newVlan=%v",
							device.DeviceName, oldMACNorm, oldEventKey.InterfaceName,
							oldEventKey.VLANID, newEventKey.VLANID)
					}
				} else {
					// moved: MAC地址相同但接口不同
					history := &models.DeviceMACHistory{
						DeviceID:           device.ID,
						DeviceNameSnapshot: device.DeviceName,
						MACAddress:         oldMACNorm,
						InterfaceName:      newEventKey.InterfaceName,
						VLANID:             newEventKey.VLANID,
						EventType:          EventMoved,
						FirstSeen:          oldMAC.CollectedAt,
						LastSeen:           collectionTime,
						CollectedAt:        collectionTime,
					}
					historyRecords = append(historyRecords, history)

					applogger.Debugf("[MAC历史] %s: moved MAC=%s oldInterface=%s newInterface=%s",
						device.DeviceName, oldMACNorm, oldEventKey.InterfaceName, newEventKey.InterfaceName)
				}
			}
		}
	}

	// 批量插入历史记录
	if len(historyRecords) > 0 {
		if err := s.db.WithContext(ctx).Create(&historyRecords).Error; err != nil {
			return fmt.Errorf("批量插入MAC历史记录失败: %w", err)
		}

		applogger.Infof("[MAC历史] %s: 记录 %d 条变更事件", device.DeviceName, len(historyRecords))
	}

	return nil
}

// vlanEqual 比较两个VLAN ID是否相等（处理nil情况）
func vlanEqual(v1, v2 *int) bool {
	if v1 == nil && v2 == nil {
		return true
	}
	if v1 == nil || v2 == nil {
		return false
	}
	return *v1 == *v2
}

// FlappingMergeWindow MAC flapping 合并窗口(2026-06-30 调整)
//
// 设计原因: 旧算法要求 next.FirstSeen == current.LastSeen(纳秒级精确相等),
// 实际 collector 写入时间会因 schedule jitter 偏差几毫秒~十几毫秒(如你数据里
// 12:46:51.890833 vs 12:46:51.876795 差 14ms),导致旧 Merge 永远不触发。
//
// 改用 2 小时窗口的语义依据:
//   - MAC history collector 每小时跑一次,2h = 2 个采集周期
//   - 网络设备 mac-address-table aging-time 通常 5 分钟(短)
//   - collector 偶发读不到 → 1h 后下次能读到 → 出现 flapping
//   - 但 2h 内 5+ 次都没出现 → 真消失
//   - 故 2h 内的 appeared/disappeared 交替认为是 flapping(短抖动),合并成单条区间
const FlappingMergeWindow = 2 * time.Hour

// MergeFlappingRecords 合并MAC flapping记录(2026-06-30 优化:放宽时间窗口+允许跨事件类型)
//
// 算法:
//   - 按 (MAC, interface, VLAN) 分组(同设备内)
//   - 按 first_seen ASC 排序后,逐对判断 next 与 current 的间隔 ≤ FlappingMergeWindow
//   - 允许 next 与 current 是不同事件类型(appeared/disappeared 交替,真实 flapping 模式)
//   - 合并后保留第一条记录(其 EventType 通常是 appeared),更新其 last_seen = 序列末 last_seen
//   - 中间记录删除
//
// 实际效果: 你给的 20 行 flapping 数据会合并成 1 条 (first_seen=12:46:51, last_seen=21:01:03),
// 前端轨迹查询展示为"MAC 在 [12:46, 21:01] 期间持续在 GigabitEthernet 2/25"——更接近真实业务语义。
func (s *macHistoryServiceImpl) MergeFlappingRecords(ctx context.Context, deviceID string) error {
	// 查询该设备的所有历史记录，按时间排序
	var histories []models.DeviceMACHistory
	if err := s.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Order("first_seen ASC, created_at ASC").
		Find(&histories).Error; err != nil {
		return fmt.Errorf("查询设备 %s 的MAC历史记录失败: %w", deviceID, err)
	}

	// 合并逻辑:
	// 1. 按 MAC+interface+VLAN 复合键分组(同 MAC 在不同接口/VLAN 不能合并)
	// 2. 在每个组内按 first_seen ASC 顺序逐对判断合并条件
	// 3. 跨事件类型(appeared ↔ disappeared)只要时间间隔 ≤ FlappingMergeWindow 就合并
	// 4. 更新当前记录 last_seen = next 的 last_seen;EventType 保留第一条(通常是 appeared)
	// 5. 删除中间的"被吞掉"记录

	type historyGroup struct {
		Key     string // MAC|interface|VLAN
		Records []*models.DeviceMACHistory
	}
	groups := make(map[string]*historyGroup)

	for i := range histories {
		hist := &histories[i]
		// 复合键:MAC + interface + VLAN(只在这三个全一致时合并)
		key := hist.MACAddress + "|" + hist.InterfaceName + "|" + fmt.Sprintf("%v", hist.VLANID)

		if _, exists := groups[key]; !exists {
			groups[key] = &historyGroup{
				Key:     key,
				Records: []*models.DeviceMACHistory{},
			}
		}
		groups[key].Records = append(groups[key].Records, hist)
	}

	// 对每个分组进行合并
	for _, group := range groups {
		if len(group.Records) <= 1 {
			continue
		}

		merged := make([]*models.DeviceMACHistory, 0, len(group.Records))
		retainedIDs := make(map[string]bool) // 记录保留的记录ID
		current := group.Records[0]

		for i := 1; i < len(group.Records); i++ {
			next := group.Records[i]

			// 合并条件(2026-06-30 放宽):
			// 1. 接口名称相同(已在 group key 强制)
			// 2. VLAN 相同(已在 group key 强制)
			// 3. 时间间隔 ≤ FlappingMergeWindow(放宽自原纳秒相等)
			// 4. 不允许 next.FirstSeen < current.LastSeen(防历史回填乱序)
			//
			// 允许 next.EventType != current.EventType(原限制过严,
			// 真实 flapping 模式就是 appeared/disappeared 交替)
			gap := next.FirstSeen.Sub(current.LastSeen)
			canMerge := current.InterfaceName == next.InterfaceName &&
				vlanEqual(current.VLANID, next.VLANID) &&
				gap >= 0 && gap <= FlappingMergeWindow

			if canMerge {
				// 合并: 更新当前记录的 last_seen 为 next 的 last_seen
				// EventType 保留第一条(通常是 appeared,语义"该 MAC 持续存在")
				current.LastSeen = next.LastSeen
				// 标记 next 待删除(不加入 retainedIDs)
			} else {
				// 不能合并,保存当前记录,开始新的合并组
				merged = append(merged, current)
				retainedIDs[current.ID] = true
				current = next
			}
		}

		// 保存最后一个合并组
		merged = append(merged, current)
		retainedIDs[current.ID] = true

		// 如果有合并发生，更新数据库
		if len(merged) < len(group.Records) {
			applogger.Infof("[MAC历史] 设备 %s: key=%s 合并前 %d 条记录，合并后 %d 条记录",
				deviceID, group.Key, len(group.Records), len(merged))

			// CR-03 fix: 实现完整的合并逻辑
			// 1. 批量更新保留记录的last_seen
			for _, mergedRecord := range merged {
				if err := s.db.WithContext(ctx).
					Model(&models.DeviceMACHistory{}).
					Where("id = ?", mergedRecord.ID).
					Update("last_seen", mergedRecord.LastSeen).Error; err != nil {
					return fmt.Errorf("更新合并记录失败: %w", err)
				}
			}

			// 2. 收集要删除的记录ID
			deletedIDs := make([]string, 0)
			for _, record := range group.Records {
				if !retainedIDs[record.ID] {
					deletedIDs = append(deletedIDs, record.ID)
				}
			}

			// 3. 批量删除重复记录
			if len(deletedIDs) > 0 {
				if err := s.db.WithContext(ctx).
					Where("id IN ?", deletedIDs).
					Delete(&models.DeviceMACHistory{}).Error; err != nil {
					return fmt.Errorf("删除重复记录失败: %w", err)
				}
				applogger.Infof("[MAC历史] 设备 %s: key=%s 删除了 %d 条重复记录",
					deviceID, group.Key, len(deletedIDs))
			}
		}
	}

	return nil
}

// CleanupAllDevicesFlapping 全量清理所有设备的 MAC flapping 记录(2026-06-30 新增)
//
// 流程:
//  1. 查 sys_device_mac_history.distinct device_id(只查有记录的设备,避免空扫)
//  2. 逐个 deviceID 调用 MergeFlappingRecords
//  3. 返回成功处理的设备数 + 首个错误(如有)
//
// 性能注意:每个 device 一次查询(已有索引 idx_device_first_seen 应足够),
// 全量清理在 1000 设备量级约 5-30s;建议低峰期手动触发或 cron 每周一次。
//
// 调用方:内部 CLI 工具 / 一次性运维脚本 / 未来的 scheduler 任务(暂未注册)。
// 不在常规请求路径调用,避免长时间持有 DB 连接。
func (s *macHistoryServiceImpl) CleanupAllDevicesFlapping(ctx context.Context) (int, error) {
	// 1. 查所有有 MAC 历史记录的设备 ID
	var deviceIDs []string
	if err := s.db.WithContext(ctx).
		Model(&models.DeviceMACHistory{}).
		Distinct("device_id").
		Pluck("device_id", &deviceIDs).Error; err != nil {
		return 0, fmt.Errorf("查询所有 deviceID 失败: %w", err)
	}

	applogger.Infof("[MAC历史] CleanupAllDevicesFlapping 开始: 共 %d 个设备", len(deviceIDs))

	// 2. 逐设备清理
	successCount := 0
	for i, deviceID := range deviceIDs {
		if err := s.MergeFlappingRecords(ctx, deviceID); err != nil {
			applogger.Errorf("[MAC历史] 设备 %s Merge 失败: %v (已成功 %d/%d)",
				deviceID, err, successCount, len(deviceIDs))
			return successCount, fmt.Errorf("设备 %s Merge 失败(已处理 %d/%d): %w",
				deviceID, successCount, len(deviceIDs), err)
		}
		successCount++

		// 每 50 个设备 log 一次进度(避免高频日志)
		if (i+1)%50 == 0 {
			applogger.Infof("[MAC历史] CleanupAllDevicesFlapping 进度: %d/%d",
				i+1, len(deviceIDs))
		}
	}

	applogger.Infof("[MAC历史] CleanupAllDevicesFlapping 完成: %d 个设备全量清理成功", successCount)
	return successCount, nil
}

// MergeByTransitions 合并同 (device_id, mac_address) 内 flapping 记录,
// 仅保留设备或接口(VLAN)有变化的转换点(2026-07-01 新增)。
//
// 设计目标:
//   - 用户原话:"仅保留设备或接口有变化的记录,删除其余的所有记录"
//   - 把"同一 MAC 在同设备同端口(同 VLAN)反复出现/消失/迁移"的抖动序列
//     折叠成单条首次事件
//   - 真实"跨设备/跨端口"的状态转换保留为多条事件,提供完整移动轨迹
//
// 算法:
//  1. 按 (device_id, mac_address) 分组
//  2. 每组内按 first_seen ASC, created_at ASC 排序
//  3. 维护"当前位置签名" = (interface_name, vlan_id)
//  4. 遍历:
//     - 首条记录 → 始终保留(初始位置)
//     - disappeared 事件 → 视为"在当前位置签名上离开",不更新当前位置签名
//     - 其他事件 → 若 (interface_name, vlan_id) 与当前位置签名不同,保留并更新;
//       若相同(纯 flapping),删除
//
// 与现有工具的对比:
//   - MergeFlappingRecords (2h 窗口):按时间窗口合并,本工具按位置签名
//   - PurgeMeaninglessRecords:保留 vlan_changed;本工具严格按用户原话,不保留 VLAN-only 变化
//
// 返回删除条数 + 首个错误。
func (s *macHistoryServiceImpl) MergeByTransitions(ctx context.Context) (int64, error) {
	// 1. 列出所有有 MAC 历史记录的 device_id
	var deviceIDs []string
	if err := s.db.WithContext(ctx).
		Model(&models.DeviceMACHistory{}).
		Distinct("device_id").
		Pluck("device_id", &deviceIDs).Error; err != nil {
		return 0, fmt.Errorf("查询所有 deviceID 失败: %w", err)
	}
	applogger.Infof("[MAC历史] MergeByTransitions 开始: 共 %d 个设备", len(deviceIDs))

	var totalDeleted int64
	for _, deviceID := range deviceIDs {
		deleted, err := s.mergeTransitionsForDevice(ctx, deviceID)
		if err != nil {
			return totalDeleted, fmt.Errorf("设备 %s MergeByTransitions 失败(已处理 %d): %w",
				deviceID, totalDeleted, err)
		}
		totalDeleted += deleted
	}
	applogger.Infof("[MAC历史] MergeByTransitions 完成: 删除 %d 条 flapping 记录", totalDeleted)
	return totalDeleted, nil
}

// mergeTransitionsForDevice 单设备内的转换点合并。
//
// 按 (device_id, mac_address) 分组遍历,每组保留状态转换点。
func (s *macHistoryServiceImpl) mergeTransitionsForDevice(ctx context.Context, deviceID string) (int64, error) {
	// 1. 查该设备所有 MAC 历史记录,按 first_seen ASC 排序
	var histories []models.DeviceMACHistory
	if err := s.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Order("first_seen ASC, created_at ASC").
		Find(&histories).Error; err != nil {
		return 0, fmt.Errorf("查询设备 %s 的 MAC 历史失败: %w", deviceID, err)
	}

	// 2. 按 mac_address 分组
	byMAC := make(map[string][]*models.DeviceMACHistory)
	for i := range histories {
		h := &histories[i]
		byMAC[h.MACAddress] = append(byMAC[h.MACAddress], h)
	}

	// 3. 每组内标记待删除的 ID
	toDelete := make([]string, 0)
	for mac, records := range byMAC {
		if len(records) <= 1 {
			continue
		}
		// 当前位置签名 = 第一条记录的位置(代表 MAC 当前所在)
		currentIface := records[0].InterfaceName
		currentVLAN := records[0].VLANID

		// 首条记录始终保留
		for i := 1; i < len(records); i++ {
			r := records[i]
			switch r.EventType {
			case models.EventDisappeared, models.EventVLANChanged:
				// disappeared:不更新位置签名(MAC 离开了当前位置但不视为去新位置)
				// vlan_changed:严格按用户原话"仅保留设备或接口有变化",VLAN 不算接口变化
				// 两者均删除
				toDelete = append(toDelete, r.ID)
			case models.EventAppeared, models.EventMoved:
				if r.InterfaceName == currentIface && vlanEqual(r.VLANID, currentVLAN) {
					// 同位置签名 = 纯 flapping,删除
					toDelete = append(toDelete, r.ID)
				} else {
					// 真实转换,保留并更新位置签名
					currentIface = r.InterfaceName
					currentVLAN = r.VLANID
				}
			default:
				// 未知类型保守保留
				applogger.Warnf("[MAC历史] 设备 %s MAC %s 未知事件类型 %s 保守保留",
					deviceID, mac, r.EventType)
			}
		}
	}

	// 4. 批量删除
	if len(toDelete) == 0 {
		return 0, nil
	}
	if err := s.db.WithContext(ctx).
		Where("id IN ?", toDelete).
		Delete(&models.DeviceMACHistory{}).Error; err != nil {
		return 0, fmt.Errorf("删除 flapping 记录失败: %w", err)
	}
	applogger.Debugf("[MAC历史] 设备 %s 合并: 删 %d 条 flapping 记录", deviceID, len(toDelete))
	return int64(len(toDelete)), nil
}

// PurgeMeaninglessRecords 全量清理 sys_device_mac_history 中的"无意义记录"(2026-06-30 新增)
//
// 保留策略(用户决策):
//   - KEEP: 所有 moved 事件
//   - KEEP: 所有 vlan_changed 事件
//   - KEEP: 每个 (device_id, mac_address) 第一条 appeared(first_seen ASC)
//   - DELETE: 其他 appeared / 所有 disappeared
//
// 流程:
//  1. dry-run 模式:仅统计将删除的行数,跳过备份与 DELETE
//  2. 真跑模式:创建备份表(普通表,时间戳命名) → 事务内 DELETE → 后置校验
//
// 备份表:sys_device_mac_history_purge_backup_YYYYMMDD_HHMMSS
//
// 回滚:
//   INSERT INTO sys_device_mac_history SELECT * FROM <backup_table>;
//   注意:备份表不含分区(普通表),需手工处理 first_seen 落对应分区。
//
// 性能:基于现有 idx_device_first_seen + DISTINCT ON,<100 万行 ~12s(见 scripts/mac/purge_meaningless 实测)。
//
// 建议调用频率:cron 每月 1 次(月初凌晨 3 点,见 internal/scheduler/mac_history_tasks.go)。
func (s *macHistoryServiceImpl) PurgeMeaninglessRecords(ctx context.Context, dryRun bool) (int64, string, error) {
	// 1. 总行数(用于校验)
	var totalCount int64
	if err := s.db.WithContext(ctx).
		Table("sys_device_mac_history").
		Count(&totalCount).Error; err != nil {
		return 0, "", fmt.Errorf("查总行数失败: %w", err)
	}

	// 2. 计算将被删除的行数(取每个 (device_id, mac_address) 第一条 appeared 的 id)
	//    - PostgreSQL:用 DISTINCT ON (PG 专属语法,生产用)
	//    - SQLite:用子查询 + MIN(first_seen) JOIN(单测用)
	var deleteCountSQL string
	if s.db.Dialector.Name() == "postgres" {
		deleteCountSQL = `
			WITH first_appeared AS (
			    SELECT DISTINCT ON (device_id, mac_address)
			           id
			      FROM sys_device_mac_history
			     WHERE event_type = 'appeared'
			     ORDER BY device_id, mac_address, first_seen ASC
			)
			SELECT COUNT(*)
			  FROM sys_device_mac_history
			 WHERE (event_type = 'disappeared')
			    OR (event_type = 'appeared' AND id NOT IN (SELECT id FROM first_appeared))
		`
	} else {
		// SQLite 兼容:用子查询取每组 MIN(first_seen) 对应的 id
		deleteCountSQL = `
			WITH first_appeared AS (
			    SELECT h.id
			      FROM sys_device_mac_history h
			      JOIN (
			          SELECT device_id, mac_address, MIN(first_seen) AS min_first_seen
			            FROM sys_device_mac_history
			           WHERE event_type = 'appeared'
			           GROUP BY device_id, mac_address
			      ) m
			        ON h.device_id = m.device_id
			       AND h.mac_address = m.mac_address
			       AND h.first_seen = m.min_first_seen
			     WHERE h.event_type = 'appeared'
			)
			SELECT COUNT(*)
			  FROM sys_device_mac_history
			 WHERE (event_type = 'disappeared')
			    OR (event_type = 'appeared' AND id NOT IN (SELECT id FROM first_appeared))
		`
	}
	var toDeleteCount int64
	if err := s.db.WithContext(ctx).Raw(deleteCountSQL).Scan(&toDeleteCount).Error; err != nil {
		return 0, "", fmt.Errorf("计算删除行数失败: %w", err)
	}
	toKeepCount := totalCount - toDeleteCount

	applogger.Infof("[MAC历史] PurgeMeaningless 预览: 总行=%d, 预计保留=%d, 预计删除=%d",
		totalCount, toKeepCount, toDeleteCount)

	if dryRun {
		return toDeleteCount, "", nil
	}

	if toDeleteCount == 0 {
		applogger.Infof("[MAC历史] PurgeMeaningless 无需删除,退出")
		return 0, "", nil
	}

	// 3. 创建备份表(普通表,非分区)
	backupTS := time.Now().Format("20060102_150405")
	backupTable := fmt.Sprintf("sys_device_mac_history_purge_backup_%s", backupTS)

	applogger.Infof("[MAC历史] PurgeMeaningless 创建备份表: %s", backupTable)
	createBackupSQL := fmt.Sprintf(`CREATE TABLE %s AS SELECT * FROM sys_device_mac_history`, backupTable)
	if err := s.db.WithContext(ctx).Exec(createBackupSQL).Error; err != nil {
		return 0, "", fmt.Errorf("创建备份表失败: %w", err)
	}

	// 4. 校验备份表行数
	var backupCount int64
	if err := s.db.WithContext(ctx).Table(backupTable).Count(&backupCount).Error; err != nil {
		applogger.Warnf("[MAC历史] 校验备份表行数失败(非阻断): %v", err)
	} else if backupCount != totalCount {
		applogger.Warnf("[MAC历史] 备份表行数 %d != 当前行数 %d,可能并发写入,中止清理",
			backupCount, totalCount)
		return 0, backupTable, fmt.Errorf("备份行数不匹配(backup=%d, total=%d)", backupCount, totalCount)
	}

	// 5. 事务内执行 DELETE
	applogger.Infof("[MAC历史] PurgeMeaningless 开始删除 %d 行...", toDeleteCount)
	start := time.Now()

	var deletedRows int64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var execSQL string
		if tx.Dialector.Name() == "postgres" {
			execSQL = `
				WITH first_appeared AS (
				    SELECT DISTINCT ON (device_id, mac_address)
				           id
				      FROM sys_device_mac_history
				     WHERE event_type = 'appeared'
				     ORDER BY device_id, mac_address, first_seen ASC
				)
				DELETE FROM sys_device_mac_history
				 WHERE (event_type = 'disappeared')
				    OR (event_type = 'appeared' AND id NOT IN (SELECT id FROM first_appeared))
			`
		} else {
			execSQL = `
				WITH first_appeared AS (
				    SELECT h.id
				      FROM sys_device_mac_history h
				      JOIN (
				          SELECT device_id, mac_address, MIN(first_seen) AS min_first_seen
				            FROM sys_device_mac_history
				           WHERE event_type = 'appeared'
				           GROUP BY device_id, mac_address
				      ) m
				        ON h.device_id = m.device_id
				       AND h.mac_address = m.mac_address
				       AND h.first_seen = m.min_first_seen
				     WHERE h.event_type = 'appeared'
				)
				DELETE FROM sys_device_mac_history
				 WHERE (event_type = 'disappeared')
				    OR (event_type = 'appeared' AND id NOT IN (SELECT id FROM first_appeared))
			`
		}
		result := tx.Exec(execSQL)
		if result.Error != nil {
			return fmt.Errorf("DELETE 失败: %w", result.Error)
		}
		deletedRows = result.RowsAffected
		applogger.Infof("[MAC历史] PurgeMeaningless DELETE 完成: %d 行受影响", deletedRows)
		return nil
	})
	if err != nil {
		applogger.Errorf("[MAC历史] PurgeMeaningless 删除失败: %v (回滚: INSERT INTO sys_device_mac_history SELECT * FROM %s)",
			err, backupTable)
		return 0, backupTable, err
	}

	// 6. 后置校验
	var finalCount int64
	if err := s.db.WithContext(ctx).Table("sys_device_mac_history").Count(&finalCount).Error; err != nil {
		return deletedRows, backupTable, fmt.Errorf("校验最终行数失败: %w", err)
	}
	elapsed := time.Since(start)

	applogger.Infof("[MAC历史] PurgeMeaningless 完成: 删除前=%d, 删除后=%d, 实际删除=%d (预期 %d), 备份=%s, 耗时=%s",
		totalCount, finalCount, totalCount-finalCount, toDeleteCount, backupTable, elapsed)

	if totalCount-finalCount != toDeleteCount {
		applogger.Warnf("[MAC历史] PurgeMeaningless 警告: 实际删除数 (%d) 与预期 (%d) 不一致,需人工核对",
			totalCount-finalCount, toDeleteCount)
	}

	return deletedRows, backupTable, nil
}
