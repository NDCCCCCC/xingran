package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// PortHistoryQuery 端口历史查询请求
type PortHistoryQuery struct {
	DeviceID      string `json:"deviceId" binding:"required"`
	InterfaceName string `json:"interfaceName,omitempty"`
	StartTime     string `json:"startTime,omitempty"` // RFC3339 格式
	EndTime       string `json:"endTime,omitempty"`   // RFC3339 格式
	Current       int    `json:"current" binding:"min=1"`
	PageSize      int    `json:"pageSize" binding:"min=1,max=100"`
}

// DeviceHistoryQuery 设备历史查询请求
type DeviceHistoryQuery struct {
	DeviceID  string `json:"deviceId" binding:"required"`
	StartTime string `json:"startTime,omitempty"` // RFC3339 格式
	EndTime   string `json:"endTime,omitempty"`   // RFC3339 格式
	Current   int    `json:"current" binding:"min=1"`
	PageSize  int    `json:"pageSize" binding:"min=1,max=100"`
}

// MACHistoryRecord MAC历史记录响应
type MACHistoryRecord struct {
	ID                 string      `json:"id"`
	DeviceID           string      `json:"deviceId"`
	DeviceNameSnapshot string      `json:"deviceNameSnapshot"`
	MACAddress         string      `json:"macAddress"`
	InterfaceName      string      `json:"interfaceName"`
	VLANID             *int        `json:"vlanId,omitempty"`
	EventType          string      `json:"eventType"`
	FirstSeen          time.Time   `json:"firstSeen"`
	LastSeen           time.Time   `json:"lastSeen"`
	CollectedAt        time.Time   `json:"collectedAt"`
}

// MACHistoryQueryResult MAC历史查询结果
type MACHistoryQueryResult struct {
	List     []MACHistoryRecord `json:"list"`
	Total    int64              `json:"total"`
	Current  int                `json:"current"`
	PageSize int                `json:"pageSize"`
}

// MACTrajectoryQuery MAC轨迹查询请求
type MACTrajectoryQuery struct {
	MACAddress string `json:"macAddress" binding:"required"`
	StartTime  string `json:"startTime,omitempty"` // RFC3339 格式
	EndTime    string `json:"endTime,omitempty"`   // RFC3339 格式
}

// TrajectoryNode 轨迹节点
type TrajectoryNode struct {
	DeviceID    string     `json:"deviceId"`
	DeviceName  string     `json:"deviceName"`
	Interface   string     `json:"interface"`
	VLANID      *int       `json:"vlanId,omitempty"`
	EventType   string     `json:"eventType"`
	StartTime   time.Time  `json:"startTime"`
	EndTime     time.Time  `json:"endTime"`
	Duration    int64      `json:"duration"` // 秒
}

// MACTrajectoryResult MAC轨迹查询结果
type MACTrajectoryResult struct {
	MACAddress string          `json:"macAddress"`
	Nodes      []TrajectoryNode `json:"nodes"`
}

// rawEvent LAG窗口函数查询结果（原始事件）
type rawEvent struct {
	ID                 string
	DeviceID           string
	DeviceNameSnapshot string
	MACAddress         string
	InterfaceName      string
	VLANID             *int
	EventType          string
	FirstSeen          time.Time
	LastSeen           time.Time
	PrevInterface      *string
	PrevVLAN           *int
	PrevFirstSeen      *time.Time
	PrevDeviceID       *string
}

// ConnectionStatsQuery 连接时长统计查询请求
type ConnectionStatsQuery struct {
	MACAddress string `json:"macAddress,omitempty"` // 可选，留空统计所有MAC
	StartTime  string `json:"startTime" binding:"required"` // RFC3339，必填
	EndTime    string `json:"endTime" binding:"required"`   // RFC3339，必填
	TopN       int    `json:"topN,omitempty"`        // 默认10
}

// ConnectionStatsDetail 连接时长统计明细
type ConnectionStatsDetail struct {
	MACAddress      string    `json:"macAddress"`
	DeviceID        string    `json:"deviceId"`
	DeviceName      string    `json:"deviceName"`
	Interface       string    `json:"interface"`
	FirstSeen       time.Time `json:"firstSeen"`
	LastSeen        time.Time `json:"lastSeen"`
	Duration        int64     `json:"duration"`       // 秒
	EventCount      int       `json:"eventCount"`
	FlappingCount   int       `json:"flappingCount"`  // event_type='moved'计数
	IsLongOccupancy bool      `json:"isLongOccupancy"` // duration > threshold
}

// LongOccupancyByMAC 长期占用TOP（按MAC聚合）
type LongOccupancyByMAC struct {
	MACAddress    string `json:"macAddress"`
	Vendor        string `json:"vendor"`        // Phase 13留空字符串，由前端OUI调用补齐
	TotalDuration int64  `json:"totalDuration"` // 秒
	PortCount     int    `json:"portCount"`
}

// HotspotByPort 端口热点TOP（按端口聚合）
type HotspotByPort struct {
	DeviceID       string `json:"deviceId"`
	DeviceName     string `json:"deviceName"`
	Interface      string `json:"interface"`
	UniqueMACCount int    `json:"uniqueMacCount"`
	TotalDuration  int64  `json:"totalDuration"` // 秒
}

// ConnectionStatsResponse 连接时长统计响应
type ConnectionStatsResponse struct {
	Details           []ConnectionStatsDetail `json:"details"`
	TopByMAC          []LongOccupancyByMAC    `json:"topByMac"`
	TopByPort         []HotspotByPort         `json:"topByPort"`
	LongOccupancyDays int                     `json:"longOccupancyDays"` // 当前阈值
}

// MACHistoryQueryService MAC历史查询服务接口
type MACHistoryQueryService interface {
	QueryPortHistory(ctx context.Context, req *PortHistoryQuery) (*MACHistoryQueryResult, error)
	QueryDeviceHistory(ctx context.Context, req *DeviceHistoryQuery) (*MACHistoryQueryResult, error)
	QueryMACTrajectory(ctx context.Context, req *MACTrajectoryQuery) (*MACTrajectoryResult, error)
	QueryConnectionStats(ctx context.Context, req *ConnectionStatsQuery) (*ConnectionStatsResponse, error)
}

// macHistoryQueryServiceImpl MAC历史查询服务实现
type macHistoryQueryServiceImpl struct {
	db *gorm.DB
}

// NewMACHistoryQueryService 创建MAC历史查询服务
func NewMACHistoryQueryService(db *gorm.DB) MACHistoryQueryService {
	return &macHistoryQueryServiceImpl{db: db}
}

// extractOUIPrefix 提取MAC地址的OUI前缀（前3字节）
// 移除分隔符 (.:-) + ToUpper + 取前6位
func extractOUIPrefix(mac string) string {
	// 移除所有分隔符
	normalized := strings.ReplaceAll(mac, ".", "")
	normalized = strings.ReplaceAll(normalized, ":", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ToUpper(normalized)

	// 取前6位（OUI前缀）
	if len(normalized) >= 6 {
		return normalized[:6]
	}
	return normalized
}

// validateMACAddress 验证MAC地址格式
// 正则校验：12位十六进制字符
func validateMACAddress(mac string) error {
	// 先尝试标准化处理
	normalized := strings.ReplaceAll(mac, ".", "")
	normalized = strings.ReplaceAll(normalized, ":", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ToUpper(normalized)

	// 验证格式：应该是12个十六进制字符
	macPattern := regexp.MustCompile(`^[0-9A-F]{12}$`)
	if !macPattern.MatchString(normalized) {
		return fmt.Errorf("无效的MAC地址格式")
	}

	return nil
}

// QueryPortHistory 查询指定端口的历史记录
func (s *macHistoryQueryServiceImpl) QueryPortHistory(ctx context.Context, req *PortHistoryQuery) (*MACHistoryQueryResult, error) {
	// WR-01 fix: 验证设备ID格式
	if _, err := uuid.Parse(req.DeviceID); err != nil {
		return nil, fmt.Errorf("无效的设备ID格式: %w", err)
	}

	// 设置默认值
	if req.Current < 1 {
		req.Current = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	// 构建查询
	query := s.db.WithContext(ctx).Table("sys_device_mac_history")

	// 添加过滤条件
	query = query.Where("device_id = ?", req.DeviceID)

	// 可选：接口名过滤
	if req.InterfaceName != "" {
		query = query.Where("interface_name = ?", req.InterfaceName)
	}

	// 可选：时间范围过滤
	// WR-02 fix: 限制查询时间范围最大为1年，防止DoS攻击
	const maxQueryRange = 365 * 24 * time.Hour // 最多查询1年

	if req.StartTime != "" && req.EndTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			return nil, fmt.Errorf("无效的开始时间格式: %w", err)
		}
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			return nil, fmt.Errorf("无效的结束时间格式: %w", err)
		}

		// 验证时间跨度
		if endTime.Sub(startTime) > maxQueryRange {
			return nil, fmt.Errorf("查询时间跨度过大，最大允许 %d 天", int(maxQueryRange.Hours()/24))
		}

		query = query.Where("first_seen >= ?", startTime).
			Where("first_seen <= ?", endTime)
	} else if req.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			return nil, fmt.Errorf("无效的开始时间格式: %w", err)
		}
		query = query.Where("first_seen >= ?", startTime)
	} else if req.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			return nil, fmt.Errorf("无效的结束时间格式: %w", err)
		}
		query = query.Where("first_seen <= ?", endTime)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		applogger.Errorf("[MAC历史查询] 查询总数失败: %v", err)
		return nil, fmt.Errorf("查询历史记录总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Current - 1) * req.PageSize
	var records []models.DeviceMACHistory
	if err := query.
		Order("first_seen DESC").
		Limit(req.PageSize).
		Offset(offset).
		Find(&records).Error; err != nil {
		applogger.Errorf("[MAC历史查询] 查询记录失败: %v", err)
		return nil, fmt.Errorf("查询历史记录失败: %w", err)
	}

	// 映射结果
	result := &MACHistoryQueryResult{
		List:     make([]MACHistoryRecord, 0, len(records)),
		Total:    total,
		Current:  req.Current,
		PageSize: req.PageSize,
	}

	for _, record := range records {
		result.List = append(result.List, MACHistoryRecord{
			ID:                 record.ID,
			DeviceID:           record.DeviceID,
			DeviceNameSnapshot: record.DeviceNameSnapshot,
			MACAddress:         record.MACAddress,
			InterfaceName:      record.InterfaceName,
			VLANID:             record.VLANID,
			EventType:          string(record.EventType),
			FirstSeen:          record.FirstSeen,
			LastSeen:           record.LastSeen,
			CollectedAt:        record.CollectedAt,
		})
	}

	return result, nil
}

// QueryDeviceHistory 查询指定设备的历史记录
func (s *macHistoryQueryServiceImpl) QueryDeviceHistory(ctx context.Context, req *DeviceHistoryQuery) (*MACHistoryQueryResult, error) {
	// WR-01 fix: 验证设备ID格式
	if _, err := uuid.Parse(req.DeviceID); err != nil {
		return nil, fmt.Errorf("无效的设备ID格式: %w", err)
	}

	// 设置默认值
	if req.Current < 1 {
		req.Current = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	// 构建查询
	query := s.db.WithContext(ctx).Table("sys_device_mac_history")

	// 添加过滤条件
	query = query.Where("device_id = ?", req.DeviceID)

	// 可选：时间范围过滤
	// WR-02 fix: 限制查询时间范围最大为1年，防止DoS攻击
	const maxQueryRange = 365 * 24 * time.Hour // 最多查询1年

	if req.StartTime != "" && req.EndTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			return nil, fmt.Errorf("无效的开始时间格式: %w", err)
		}
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			return nil, fmt.Errorf("无效的结束时间格式: %w", err)
		}

		// 验证时间跨度
		if endTime.Sub(startTime) > maxQueryRange {
			return nil, fmt.Errorf("查询时间跨度过大，最大允许 %d 天", int(maxQueryRange.Hours()/24))
		}

		query = query.Where("first_seen >= ?", startTime).
			Where("first_seen <= ?", endTime)
	} else if req.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			return nil, fmt.Errorf("无效的开始时间格式: %w", err)
		}
		query = query.Where("first_seen >= ?", startTime)
	} else if req.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			return nil, fmt.Errorf("无效的结束时间格式: %w", err)
		}
		query = query.Where("first_seen <= ?", endTime)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		applogger.Errorf("[MAC历史查询] 查询总数失败: %v", err)
		return nil, fmt.Errorf("查询历史记录总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Current - 1) * req.PageSize
	var records []models.DeviceMACHistory
	if err := query.
		Order("first_seen DESC").
		Limit(req.PageSize).
		Offset(offset).
		Find(&records).Error; err != nil {
		applogger.Errorf("[MAC历史查询] 查询记录失败: %v", err)
		return nil, fmt.Errorf("查询历史记录失败: %w", err)
	}

	// 映射结果
	result := &MACHistoryQueryResult{
		List:     make([]MACHistoryRecord, 0, len(records)),
		Total:    total,
		Current:  req.Current,
		PageSize: req.PageSize,
	}

	for _, record := range records {
		result.List = append(result.List, MACHistoryRecord{
			ID:                 record.ID,
			DeviceID:           record.DeviceID,
			DeviceNameSnapshot: record.DeviceNameSnapshot,
			MACAddress:         record.MACAddress,
			InterfaceName:      record.InterfaceName,
			VLANID:             record.VLANID,
			EventType:          string(record.EventType),
			FirstSeen:          record.FirstSeen,
			LastSeen:           record.LastSeen,
			CollectedAt:        record.CollectedAt,
		})
	}

	return result, nil
}

// QueryMACTrajectory 查询MAC地址移动轨迹
// 使用PostgreSQL LAG窗口函数单次查询获取所有状态转换点，应用层做区间聚合
func (s *macHistoryQueryServiceImpl) QueryMACTrajectory(ctx context.Context, req *MACTrajectoryQuery) (*MACTrajectoryResult, error) {
	// 1. 验证MAC地址格式
	if err := validateMACAddress(req.MACAddress); err != nil {
		return nil, fmt.Errorf("MAC地址验证失败: %w", err)
	}

	// 2. 规范化MAC地址（移除分隔符，转大写）
	normalizedMAC := strings.ReplaceAll(req.MACAddress, ".", "")
	normalizedMAC = strings.ReplaceAll(normalizedMAC, ":", "")
	normalizedMAC = strings.ReplaceAll(normalizedMAC, "-", "")
	normalizedMAC = strings.ToUpper(normalizedMAC)

	// 3. 解析时间范围，设置默认值（now-30d ~ now）
	const maxQueryRange = 365 * 24 * time.Hour // 最多查询1年（复用现有常量）

	var startTime, endTime time.Time
	var err error

	if req.StartTime != "" {
		startTime, err = time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			return nil, fmt.Errorf("无效的开始时间格式: %w", err)
		}
	} else {
		// 默认30天前
		startTime = time.Now().Add(-30 * 24 * time.Hour)
	}

	if req.EndTime != "" {
		endTime, err = time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			return nil, fmt.Errorf("无效的结束时间格式: %w", err)
		}
	} else {
		// 默认当前时间
		endTime = time.Now()
	}

	// 4. 验证时间跨度（DoS保护）
	if endTime.Sub(startTime) > maxQueryRange {
		return nil, fmt.Errorf("查询时间跨度过大，最大允许 %d 天", int(maxQueryRange.Hours()/24))
	}

	// 5. 执行LAG窗口函数查询
	var rawEvents []rawEvent

	sql := `
		SELECT
			id, device_id, device_name_snapshot, mac_address,
			interface_name, vlan_id, event_type,
			first_seen, last_seen,
			LAG(interface_name) OVER w AS prev_interface,
			LAG(vlan_id)       OVER w AS prev_vlan,
			LAG(first_seen)    OVER w AS prev_first_seen,
			LAG(device_id)     OVER w AS prev_device_id
		FROM sys_device_mac_history
		WHERE mac_address = ? AND first_seen >= ? AND first_seen <= ?
		WINDOW w AS (PARTITION BY mac_address ORDER BY first_seen)
		ORDER BY first_seen ASC
	`

	if err := s.db.WithContext(ctx).Raw(sql, normalizedMAC, startTime, endTime).Scan(&rawEvents).Error; err != nil {
		applogger.Errorf("[MAC轨迹查询] 查询失败: %v", err)
		return nil, fmt.Errorf("查询MAC轨迹失败: %w", err)
	}

	// 6. 应用层区间聚合（按DeviceID, InterfaceName, VLANID复合键合并连续状态）
	nodes := s.aggregateTrajectory(rawEvents)

	return &MACTrajectoryResult{
		MACAddress: normalizedMAC,
		Nodes:      nodes,
	}, nil
}

// aggregateTrajectory 聚合原始事件为轨迹节点
// 按复合键合并连续状态，计算停留时长
func (s *macHistoryQueryServiceImpl) aggregateTrajectory(rawEvents []rawEvent) []TrajectoryNode {
	if len(rawEvents) == 0 {
		return []TrajectoryNode{}
	}

	var nodes []TrajectoryNode
	var current *TrajectoryNode

	for _, evt := range rawEvents {
		if current == nil {
			// 第一个节点
			current = &TrajectoryNode{
				DeviceID:   evt.DeviceID,
				DeviceName: evt.DeviceNameSnapshot,
				Interface:  evt.InterfaceName,
				VLANID:     evt.VLANID,
				EventType:  evt.EventType,
				StartTime:  evt.FirstSeen,
				EndTime:    evt.LastSeen,
			}
		} else {
			// 检查是否与当前节点属于同一位置（复合键比较）
			if s.sameLocation(*current, evt) {
				// 同一位置，合并区间：更新EndTime和EventType
				current.EndTime = evt.LastSeen
				current.EventType = evt.EventType // 保留最新事件类型用于tooltip
			} else {
				// 不同位置，保存当前节点并开始新节点
				current.Duration = int64(current.EndTime.Sub(current.StartTime).Seconds())
				nodes = append(nodes, *current)

				current = &TrajectoryNode{
					DeviceID:   evt.DeviceID,
					DeviceName: evt.DeviceNameSnapshot,
					Interface:  evt.InterfaceName,
					VLANID:     evt.VLANID,
					EventType:  evt.EventType,
					StartTime:  evt.FirstSeen,
					EndTime:    evt.LastSeen,
				}
			}
		}
	}

	// 保存最后一个节点
	if current != nil {
		current.Duration = int64(current.EndTime.Sub(current.StartTime).Seconds())
		nodes = append(nodes, *current)
	}

	return nodes
}

// sameLocation 判断两个轨迹节点是否属于同一位置
// 复合键比较：DeviceID + InterfaceName + VLANID
func (s *macHistoryQueryServiceImpl) sameLocation(node TrajectoryNode, evt rawEvent) bool {
	if node.DeviceID != evt.DeviceID || node.Interface != evt.InterfaceName {
		return false
	}

	// VLANID比较（处理nil情况）
	if node.VLANID == nil && evt.VLANID == nil {
		return true
	}
	if node.VLANID == nil || evt.VLANID == nil {
		return false
	}
	return *node.VLANID == *evt.VLANID
}

// getLongOccupancyThreshold 获取长期占用阈值（天数）
// 从sys_config表查询，不存在或解析失败时返回默认值30
func (s *macHistoryQueryServiceImpl) getLongOccupancyThreshold(ctx context.Context) (int, error) {
	const defaultDays = 30
	const configKey = "network.mac.history.long_occupancy_threshold_days"

	var configValue string
	err := s.db.WithContext(ctx).
		Table("sys_config").
		Where("config_key = ?", configKey).
		Select("config_value").
		Row().
		Scan(&configValue)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 配置不存在，返回默认值
			return defaultDays, nil
		}
		return defaultDays, fmt.Errorf("查询长期占用阈值配置失败: %w", err)
	}

	// 解析配置值
	var days int
	if _, err := fmt.Sscanf(configValue, "%d", &days); err != nil || days <= 0 {
		// 解析失败或值无效，返回默认值
		applogger.Warnf("[MAC历史统计] 长期占用阈值配置值无效: %s，使用默认值 %d 天", configValue, defaultDays)
		return defaultDays, nil
	}

	return days, nil
}

// QueryConnectionStats 查询连接时长统计
// 输出明细（每个MAC×端口的停留时长+flapping计数）+Top-N（按MAC长期占用Top+按端口热门连接Top）
func (s *macHistoryQueryServiceImpl) QueryConnectionStats(ctx context.Context, req *ConnectionStatsQuery) (*ConnectionStatsResponse, error) {
	// 1. 校验StartTime/EndTime必填
	if req.StartTime == "" || req.EndTime == "" {
		return nil, fmt.Errorf("开始时间和结束时间为必填项")
	}

	// 2. 解析时间范围
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return nil, fmt.Errorf("无效的开始时间格式: %w", err)
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("无效的结束时间格式: %w", err)
	}

	// 3. 验证时间跨度（复用maxQueryRange）
	const maxQueryRange = 365 * 24 * time.Hour
	if endTime.Sub(startTime) > maxQueryRange {
		return nil, fmt.Errorf("查询时间跨度过大，最大允许 %d 天", int(maxQueryRange.Hours()/24))
	}

	// 4. 调用getLongOccupancyThreshold拿到阈值
	thresholdDays, err := s.getLongOccupancyThreshold(ctx)
	if err != nil {
		applogger.Warnf("[MAC历史统计] 获取长期占用阈值失败，使用默认值30天: %v", err)
		thresholdDays = 30 // 降级使用默认值
	}
	thresholdSec := int64(thresholdDays * 86400)

	// 5. 设置TopN默认值
	topN := req.TopN
	if topN <= 0 {
		topN = 10
	}

	// 6. 查询明细SQL（按MAC×Device×Interface聚合）
	detailSQL := `
		SELECT
			mac_address, device_id, device_name_snapshot, interface_name,
			MIN(first_seen) AS first_seen,
			MAX(last_seen) AS last_seen,
			EXTRACT(EPOCH FROM (MAX(last_seen) - MIN(first_seen)))::bigint AS duration,
			COUNT(*) AS event_count,
			COUNT(*) FILTER (WHERE event_type = 'moved') AS flapping_count
		FROM sys_device_mac_history
		WHERE first_seen >= ? AND first_seen <= ?
	`
	// 可选：MAC地址过滤
	args := []interface{}{startTime, endTime}
	if req.MACAddress != "" {
		// 验证MAC格式
		if err := validateMACAddress(req.MACAddress); err != nil {
			return nil, fmt.Errorf("MAC地址验证失败: %w", err)
		}
		// 规范化MAC地址
		normalizedMAC := strings.ReplaceAll(req.MACAddress, ".", "")
		normalizedMAC = strings.ReplaceAll(normalizedMAC, ":", "")
		normalizedMAC = strings.ReplaceAll(normalizedMAC, "-", "")
		normalizedMAC = strings.ToUpper(normalizedMAC)
		detailSQL += " AND mac_address = ?"
		args = append(args, normalizedMAC)
	}

	detailSQL += `
		GROUP BY mac_address, device_id, device_name_snapshot, interface_name
		ORDER BY duration DESC
		LIMIT ? OFFSET ?
	`

	type rawDetail struct {
		MACAddress         string
		DeviceID           string
		DeviceNameSnapshot string
		InterfaceName      string
		FirstSeen          time.Time
		LastSeen           time.Time
		Duration           int64
		EventCount         int
		FlappingCount      int
	}

	var rawDetails []rawDetail
	if err := s.db.WithContext(ctx).Raw(detailSQL, append(args, 1000, 0)...).Scan(&rawDetails).Error; err != nil {
		applogger.Errorf("[MAC历史统计] 查询明细失败: %v", err)
		return nil, fmt.Errorf("查询连接时长明细失败: %w", err)
	}

	// 转换为响应格式，标记IsLongOccupancy
	details := make([]ConnectionStatsDetail, 0, len(rawDetails))
	for _, d := range rawDetails {
		details = append(details, ConnectionStatsDetail{
			MACAddress:      d.MACAddress,
			DeviceID:        d.DeviceID,
			DeviceName:      d.DeviceNameSnapshot,
			Interface:       d.InterfaceName,
			FirstSeen:       d.FirstSeen,
			LastSeen:        d.LastSeen,
			Duration:        d.Duration,
			EventCount:      d.EventCount,
			FlappingCount:   d.FlappingCount,
			IsLongOccupancy: d.Duration > thresholdSec,
		})
	}

	// 7. 查询TopByMAC SQL（按MAC聚合，过滤长期占用）
	topByMACSQL := `
		SELECT
			mac_address,
			SUM(EXTRACT(EPOCH FROM (MAX(last_seen) - MIN(first_seen))))::bigint AS total_duration,
			COUNT(DISTINCT device_id || ':' || interface_name) AS port_count
		FROM sys_device_mac_history
		WHERE first_seen >= ? AND first_seen <= ?
	`

	topByMACArgs := []interface{}{startTime, endTime}
	if req.MACAddress != "" {
		normalizedMAC := strings.ReplaceAll(req.MACAddress, ".", "")
		normalizedMAC = strings.ReplaceAll(normalizedMAC, ":", "")
		normalizedMAC = strings.ReplaceAll(normalizedMAC, "-", "")
		normalizedMAC = strings.ToUpper(normalizedMAC)
		topByMACSQL += " AND mac_address = ?"
		topByMACArgs = append(topByMACArgs, normalizedMAC)
	}

	topByMACSQL += `
		GROUP BY mac_address
		HAVING SUM(EXTRACT(EPOCH FROM (MAX(last_seen) - MIN(first_seen)))) > ?
		ORDER BY total_duration DESC
		LIMIT ?
	`

	type rawTopMAC struct {
		MACAddress    string
		TotalDuration int64
		PortCount     int
	}

	var rawTopMACs []rawTopMAC
	if err := s.db.WithContext(ctx).Raw(topByMACSQL, append(topByMACArgs, thresholdSec, topN)...).Scan(&rawTopMACs).Error; err != nil {
		applogger.Errorf("[MAC历史统计] 查询TopByMAC失败: %v", err)
		return nil, fmt.Errorf("查询长期占用TOP失败: %w", err)
	}

	// 转换为响应格式（Vendor字段留空，由前端OUI调用补齐）
	topByMAC := make([]LongOccupancyByMAC, 0, len(rawTopMACs))
	for _, m := range rawTopMACs {
		topByMAC = append(topByMAC, LongOccupancyByMAC{
			MACAddress:    m.MACAddress,
			Vendor:        "", // Phase 13留空，由前端补齐
			TotalDuration: m.TotalDuration,
			PortCount:     m.PortCount,
		})
	}

	// 8. 查询TopByPort SQL（按端口聚合）
	topByPortSQL := `
		SELECT
			device_id, device_name_snapshot, interface_name,
			COUNT(DISTINCT mac_address) AS unique_mac_count,
			SUM(EXTRACT(EPOCH FROM (MAX(last_seen) - MIN(first_seen))))::bigint AS total_duration
		FROM sys_device_mac_history
		WHERE first_seen >= ? AND first_seen <= ?
	`

	topByPortArgs := []interface{}{startTime, endTime}
	// 注意：TopByPort不过滤MAC，因为用户可能想看特定MAC的端口热点
	// 但如果请求指定了MAC，则只统计该MAC的端口连接
	if req.MACAddress != "" {
		normalizedMAC := strings.ReplaceAll(req.MACAddress, ".", "")
		normalizedMAC = strings.ReplaceAll(normalizedMAC, ":", "")
		normalizedMAC = strings.ReplaceAll(normalizedMAC, "-", "")
		normalizedMAC = strings.ToUpper(normalizedMAC)
		topByPortSQL += " AND mac_address = ?"
		topByPortArgs = append(topByPortArgs, normalizedMAC)
	}

	topByPortSQL += `
		GROUP BY device_id, device_name_snapshot, interface_name
		ORDER BY total_duration DESC
		LIMIT ?
	`

	type rawTopPort struct {
		DeviceID           string
		DeviceNameSnapshot string
		InterfaceName      string
		UniqueMACCount     int
		TotalDuration      int64
	}

	var rawTopPorts []rawTopPort
	if err := s.db.WithContext(ctx).Raw(topByPortSQL, append(topByPortArgs, topN)...).Scan(&rawTopPorts).Error; err != nil {
		applogger.Errorf("[MAC历史统计] 查询TopByPort失败: %v", err)
		return nil, fmt.Errorf("查询端口热点TOP失败: %w", err)
	}

	// 转换为响应格式
	topByPort := make([]HotspotByPort, 0, len(rawTopPorts))
	for _, p := range rawTopPorts {
		topByPort = append(topByPort, HotspotByPort{
			DeviceID:       p.DeviceID,
			DeviceName:     p.DeviceNameSnapshot,
			Interface:      p.InterfaceName,
			UniqueMACCount: p.UniqueMACCount,
			TotalDuration:  p.TotalDuration,
		})
	}

	// 9. 返回三段式响应
	return &ConnectionStatsResponse{
		Details:           details,
		TopByMAC:          topByMAC,
		TopByPort:         topByPort,
		LongOccupancyDays: thresholdDays,
	}, nil
}
