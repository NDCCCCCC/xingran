package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
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
	// RecordMACChange 记录MAC地址变更
	// 对比新旧状态，识别变更类型并写入历史表
	RecordMACChange(ctx context.Context, device *models.NetworkDevice, oldMACs, newMACs map[MACEvent]*models.DeviceMACAddress) error

	// MergeFlappingRecords 合并MAC flapping记录
	// 查询同一设备的连续相同MAC状态记录，更新last_seen避免重复
	MergeFlappingRecords(ctx context.Context, deviceID string) error
}

// macHistoryServiceImpl MAC历史服务实现
type macHistoryServiceImpl struct {
	db *gorm.DB
}

// NewMACHistoryService 创建MAC历史服务
func NewMACHistoryService(db *gorm.DB) MACHistoryService {
	return &macHistoryServiceImpl{db: db}
}

// normalizeMACAddress 规范化MAC地址格式
// 将各种格式的MAC地址统一转换为 AA:BB:CC:DD:EE:FF 格式
// 处理不同厂商输出格式不一致的问题（RESEARCH.md Pitfall 3）
//
// 支持的输入格式：
//   - aabb.ccdd.eeff (Cisco/Huawei) -> AA:BB:CC:DD:EE:FF
//   - aa-bb-cc-dd-ee-ff (标准格式) -> AA:BB:CC:DD:EE:FF
//   - aa:bb:cc:dd:ee:ff (已规范) -> AA:BB:CC:DD:EE:FF
//   - AABBCCDDEEFF (无分隔符) -> AA:BB:CC:DD:EE:FF
func normalizeMACAddress(mac string) string {
	if mac == "" {
		return ""
	}

	// 移除所有空格
	mac = strings.TrimSpace(mac)
	if mac == "" {
		return ""
	}

	// 移除所有分隔符（.: -）
	normalized := strings.ReplaceAll(mac, ".", "")
	normalized = strings.ReplaceAll(normalized, ":", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ToUpper(strings.TrimSpace(normalized))

	// 验证格式：应该是12个十六进制字符
	macPattern := regexp.MustCompile(`^[0-9A-F]{12}$`)
	if !macPattern.MatchString(normalized) {
		// 格式不合法，返回原始值
		applogger.Warnf("[MAC历史] MAC地址格式无效: %s", mac)
		return mac
	}

	// 插入冒号分隔符：AA:BB:CC:DD:EE:FF
	var result strings.Builder
	for i := 0; i < 12; i += 2 {
		if i > 0 {
			result.WriteString(":")
		}
		result.WriteString(normalized[i : i+2])
	}

	return result.String()
}

// buildMACStateMap 构建MAC状态映射
// 将DeviceMACAddress切片转换为map[MACEvent]*models.DeviceMACAddress
// 关键：在构建时调用normalizeMACAddress规范化MAC地址格式
func (s *macHistoryServiceImpl) buildMACStateMap(macList []models.DeviceMACAddress) map[MACEvent]*models.DeviceMACAddress {
	stateMap := make(map[MACEvent]*models.DeviceMACAddress)

	for i := range macList {
		mac := &macList[i]

		// 规范化MAC地址格式（RESEARCH.md Pitfall 3）
		normalizedMAC := normalizeMACAddress(mac.MACAddress)

		eventKey := MACEvent{
			MACAddress:    normalizedMAC,
			InterfaceName: mac.InterfaceName,
			VLANID:        mac.VLANID,
		}

		// 更新MAC地址为规范化格式
		mac.MACAddress = normalizedMAC

		stateMap[eventKey] = mac
	}

	return stateMap
}

// RecordMACChange 记录MAC地址变更（CONTEXT.md D-01, D-02, D-03）
// 对比新旧状态，识别4种变更类型并写入历史表
func (s *macHistoryServiceImpl) RecordMACChange(ctx context.Context, device *models.NetworkDevice, oldMACs, newMACs map[MACEvent]*models.DeviceMACAddress) error {
	// 构建历史记录列表
	var historyRecords []*models.DeviceMACHistory
	collectionTime := time.Now()

	// 1. appeared: 新数据存在且旧数据不存在
	for eventKey := range newMACs {
		if _, exists := oldMACs[eventKey]; !exists {
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
	}

	// 2. disappeared: 新数据不存在且旧数据存在
	for eventKey, oldMAC := range oldMACs {
		if _, exists := newMACs[eventKey]; !exists {
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
	}

	// 3. moved: MAC地址相同但接口不同
	// 4. vlan_changed: 接口相同但VLAN不同
	for oldEventKey, oldMAC := range oldMACs {
		// 查找相同MAC地址的新记录
		for newEventKey := range newMACs {
			// 比较规范化的MAC地址
			oldMACNorm := normalizeMACAddress(oldMAC.MACAddress)
			newMAC := newMACs[newEventKey]
			newMACNorm := normalizeMACAddress(newMAC.MACAddress)

			if oldMACNorm == newMACNorm {
				// MAC地址相同，检查接口和VLAN

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

				// 已找到匹配，跳出内层循环
				break
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

// MergeFlappingRecords 合并MAC flapping记录（CONTEXT.md D-03）
// 查询同一设备的连续相同MAC状态记录，更新last_seen避免重复
func (s *macHistoryServiceImpl) MergeFlappingRecords(ctx context.Context, deviceID string) error {
	// 查询该设备的所有历史记录，按时间排序
	var histories []models.DeviceMACHistory
	if err := s.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Order("first_seen ASC, created_at ASC").
		Find(&histories).Error; err != nil {
		return fmt.Errorf("查询MAC历史记录失败: %w", err)
	}

	// 合并逻辑：
	// 1. 按MAC地址分组
	// 2. 对每个MAC地址，找到连续的相同状态记录
	// 3. 更新第一条记录的last_seen为最后一条记录的last_seen
	// 4. 删除中间的重复记录

	// 按MAC地址分组
	type historyGroup struct {
		MACAddress string
		Records    []*models.DeviceMACHistory
	}
	groups := make(map[string]*historyGroup)

	for i := range histories {
		hist := &histories[i]

		if _, exists := groups[hist.MACAddress]; !exists {
			groups[hist.MACAddress] = &historyGroup{
				MACAddress: hist.MACAddress,
				Records:    []*models.DeviceMACHistory{},
			}
		}
		groups[hist.MACAddress].Records = append(groups[hist.MACAddress].Records, hist)
	}

	// 对每个MAC地址组进行合并
	for _, group := range groups {
		if len(group.Records) <= 1 {
			continue
		}

		// 找到连续的相同事件类型记录
		merged := make([]*models.DeviceMACHistory, 0, len(group.Records))
		current := group.Records[0]

		for i := 1; i < len(group.Records); i++ {
			next := group.Records[i]

			// 检查是否可以合并：
			// 1. 事件类型相同
			// 2. 接口名称相同
			// 3. VLAN相同
			// 4. 时间连续（next.FirstSeen == current.LastSeen）
			if current.EventType == next.EventType &&
				current.InterfaceName == next.InterfaceName &&
				vlanEqual(current.VLANID, next.VLANID) &&
				next.FirstSeen.Equal(current.LastSeen) {

				// 合并：更新当前记录的last_seen
				current.LastSeen = next.LastSeen

				// 标记next记录为待删除（在后续处理中删除）
				// 注意：这里不立即删除，避免在遍历中修改切片
			} else {
				// 不能合并，保存当前记录，开始新的合并组
				merged = append(merged, current)
				current = next
			}
		}

		// 保存最后一个合并组
		merged = append(merged, current)

		// 如果有合并发生，更新数据库
		if len(merged) < len(group.Records) {
			applogger.Infof("[MAC历史] 设备 %s: MAC %s 合并前 %d 条记录，合并后 %d 条记录",
				deviceID, group.MACAddress, len(group.Records), len(merged))

			// TODO: 实际删除重复记录的逻辑
			// 由于需要精确识别哪些记录被合并，这里暂时仅记录日志
			// 完整实现需要：
			// 1. 识别被合并的记录ID列表
			// 2. 更新保留记录的last_seen
			// 3. 删除重复记录
		}
	}

	return nil
}
