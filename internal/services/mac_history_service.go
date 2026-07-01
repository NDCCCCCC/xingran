package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/normalize"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
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
// 2026-06-30 quick 优化(2 层去重):
//   - 第 1 层:用 normalized MAC 集合判定"MAC 是否在设备上",取代 (mac, interface, vlan)
//     三元组比对,避免漏采周期产生冗余 disappeared/appeared 配对
//   - 第 2 层:写入前查 history 表,如果最近一次同 MAC 记录是同端口 → 视为 re-detection
//     跳过(原本会写 disappeared 然后下一周期再写 appeared 配对,现在两次都不写)
//   - 结果:稳定 MAC 7 天 ≈ 1 条(首次出现)而不是 366 条冗余
//   - 真实变化(换端口/换 VLAN/新出现)仍正常写入
//   - 已知 trade-off:真实离开数天的 MAC 可能漏写 disappeared(同端口 re-check 命中
//     上次的 appeared),如需可加时间阈值扩展
//
// 关联: [[mac-address-normalize-returns-colon-format]]
func (s *macHistoryServiceImpl) RecordMACChange(ctx context.Context, device *models.NetworkDevice, oldMACs, newMACs map[MACEvent]*models.DeviceMACAddress) error {
	// 构建历史记录列表
	var historyRecords []*models.DeviceMACHistory
	collectionTime := time.Now()

	// 2026-06-30 quick L1:normalized MAC 集合判定
	oldMACNormSet := make(map[string]bool, len(oldMACs))
	for _, oldMAC := range oldMACs {
		oldMACNormSet[NormalizeMACAddress(oldMAC.MACAddress)] = true
	}
	newMACNormSet := make(map[string]bool, len(newMACs))
	for _, newMAC := range newMACs {
		newMACNormSet[NormalizeMACAddress(newMAC.MACAddress)] = true
	}

	// 2026-06-30 quick L2:查询候选 MAC 的最近一次历史记录接口
	// 单次 IN 查询,避免 N 次点查
	candidateMACs := make([]string, 0)
	for eventKey := range newMACs {
		normMAC := NormalizeMACAddress(eventKey.MACAddress)
		if !oldMACNormSet[normMAC] {
			candidateMACs = append(candidateMACs, eventKey.MACAddress)
		}
	}
	for eventKey := range oldMACs {
		normMAC := NormalizeMACAddress(eventKey.MACAddress)
		if !newMACNormSet[normMAC] {
			candidateMACs = append(candidateMACs, eventKey.MACAddress)
		}
	}

	// mac -> 最近一次记录的接口(去重:同 mac 多条记录只保留最新)
	recentIfaceByMAC := make(map[string]string)
	if len(candidateMACs) > 0 {
		var recentRecords []models.DeviceMACHistory
		if err := s.db.WithContext(ctx).
			Where("device_id = ? AND mac_address IN ?", device.ID, candidateMACs).
			Order("collected_at DESC").
			Find(&recentRecords).Error; err != nil {
			applogger.Warnf("[MAC历史] %s: 查询最近历史失败(非阻塞,继续): %v", device.DeviceName, err)
		}
		for _, r := range recentRecords {
			if _, exists := recentIfaceByMAC[r.MACAddress]; !exists {
				recentIfaceByMAC[r.MACAddress] = r.InterfaceName
			}
		}
	}

	// 1. appeared: MAC 在设备上完全新增(不在 oldMACs 任何端口)
	//    且最近一次记录不是同端口(re-detection 跳过)
	for eventKey := range newMACs {
		normMAC := NormalizeMACAddress(eventKey.MACAddress)
		if oldMACNormSet[normMAC] {
			continue
		}
		if recentIfaceByMAC[eventKey.MACAddress] == eventKey.InterfaceName {
			applogger.Debugf("[MAC历史] %s: 跳过 appeared(re-detection 同端口): MAC=%s interface=%s",
				device.DeviceName, eventKey.MACAddress, eventKey.InterfaceName)
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
	//    且最近一次记录不是同端口(漏采视为 re-detection 跳过)
	for eventKey, oldMAC := range oldMACs {
		normMAC := NormalizeMACAddress(eventKey.MACAddress)
		if newMACNormSet[normMAC] {
			continue
		}
		if recentIfaceByMAC[oldMAC.MACAddress] == eventKey.InterfaceName {
			applogger.Debugf("[MAC历史] %s: 跳过 disappeared(漏采 re-detection 同端口): MAC=%s interface=%s",
				device.DeviceName, oldMAC.MACAddress, eventKey.InterfaceName)
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
// 性能:基于现有 idx_device_first_seen + DISTINCT ON,<100 万行 ~12s(见 cmd/mac_purge_meaningless 实测)。
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
