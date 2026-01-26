package addomain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// ComputerService 电脑设备服务
type ComputerService struct {
	db *gorm.DB
}

// NewComputerService 创建电脑设备服务
func NewComputerService(db *gorm.DB) *ComputerService {
	return &ComputerService{db: db}
}

// ComputerListRequest 电脑设备列表请求
type ComputerListRequest struct {
	ConfigID     string `json:"configId"`
	OUN          string `json:"ouDn,omitempty"`       // 所属OU
	ComputerName string `json:"computerName,omitempty"` // 计算机名称模糊搜索
	Current      int    `json:"current"`
	PageSize     int    `json:"pageSize"`
}

// ComputerDetail 电脑设备详情（包含解析后的信息）
type ComputerDetail struct {
	models.ADComputer
	LastLogonUser string `json:"lastLogonUser,omitempty"` // 最后登录用户（从描述中解析）
}

// List 获取电脑设备列表
func (s *ComputerService) List(ctx context.Context, req *ComputerListRequest) ([]ComputerDetail, int64, error) {
	s.normalizePagination(req)

	query := s.buildComputerQuery(ctx, req)
	total, err := s.countComputers(query)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []ComputerDetail{}, 0, nil
	}

	computers, err := s.fetchComputers(ctx, req, total)
	if err != nil {
		return nil, 0, err
	}

	return s.convertToDetails(computers), total, nil
}

// normalizePagination 标准化分页参数
func (s *ComputerService) normalizePagination(req *ComputerListRequest) {
	if req.Current <= 0 {
		req.Current = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
}

// buildComputerQuery 构建查询条件
func (s *ComputerService) buildComputerQuery(ctx context.Context, req *ComputerListRequest) *gorm.DB {
	query := s.db.WithContext(ctx).Model(&models.ADComputer{}).
		Where("ad_config_id = ? AND deleted_at IS NULL", req.ConfigID)

	if req.OUN != "" {
		query = query.Where("ou_dn = ?", req.OUN)
	}

	if req.ComputerName != "" {
		query = query.Where("computer_name LIKE ?", "%"+req.ComputerName+"%")
	}

	return query
}

// countComputers 统计电脑数量
func (s *ComputerService) countComputers(query *gorm.DB) (int64, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, fmt.Errorf("查询电脑设备总数失败: %w", err)
	}
	return total, nil
}

// fetchComputers 获取电脑数据
func (s *ComputerService) fetchComputers(ctx context.Context, req *ComputerListRequest, total int64) ([]models.ADComputer, error) {
	query := s.buildComputerQuery(ctx, req)

	var computers []models.ADComputer
	offset := (req.Current - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).
		Order("created_at DESC").
		Find(&computers).Error; err != nil {
		return nil, fmt.Errorf("查询电脑设备列表失败: %w", err)
	}

	return computers, nil
}

// convertToDetails 转换为详情格式
func (s *ComputerService) convertToDetails(computers []models.ADComputer) []ComputerDetail {
	details := make([]ComputerDetail, len(computers))
	for i, comp := range computers {
		details[i] = ComputerDetail{
			ADComputer:     comp,
			LastLogonUser: parseComputerDescriptionForUser(comp.OriginalDescription),
		}
	}
	return details
}

// GetByDN 根据DN获取电脑设备详情
func (s *ComputerService) GetByDN(ctx context.Context, configID, computerDN string) (*ComputerDetail, error) {
	var computer models.ADComputer
	if err := s.db.WithContext(ctx).
		Where("ad_config_id = ? AND distinguished_name = ? AND deleted_at IS NULL", configID, computerDN).
		First(&computer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("电脑设备不存在")
		}
		return nil, fmt.Errorf("查询电脑设备失败: %w", err)
	}

	detail := &ComputerDetail{
		ADComputer:     computer,
		LastLogonUser: parseComputerDescriptionForUser(computer.OriginalDescription),
	}

	return detail, nil
}

// parseComputerDescriptionForUser 从描述中解析最后登录用户
// 格式: |username|ip|mac|serial|os|cpu|arch|memory|disk|datetime
func parseComputerDescriptionForUser(desc string) string {
	if desc == "" {
		return ""
	}

	// 按竖线分割
	parts := strings.Split(desc, "|")
	if len(parts) > 1 && parts[1] != "" {
		return strings.TrimSpace(parts[1])
	}

	return ""
}

// parseComputerDescription 解析电脑描述字段
// 格式: |username|ip|mac|serial|os|cpu|arch|memory|disk|datetime
// 示例: |chengang-071|10.62.97.24|E0:BE:03:5E:07:85|M70PXBL9|Microsoft Windows 10 专业版 10.0.18363|Intel(R) Core(TM) i5-10500 CPU @ 3.10GHz|64位|Memory:8GB|Hard disk:119GB|2026/1/26 8:58:21
func parseComputerDescription(desc string) map[string]string {
	result := make(map[string]string)

	if desc == "" {
		return result
	}

	// 按竖线分割
	parts := strings.Split(desc, "|")
	if len(parts) < 10 {
		return result
	}

	// 索引映射 (跳过第一个空元素)
	// 0: 空, 1: username, 2: ip, 3: mac, 4: serial, 5: os, 6: cpu, 7: arch, 8: memory, 9: disk, 10: datetime
	if len(parts) > 1 && parts[1] != "" {
		result["lastLogonUser"] = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 && parts[2] != "" {
		result["ipAddress"] = strings.TrimSpace(parts[2])
	}
	if len(parts) > 3 && parts[3] != "" {
		result["macAddress"] = strings.TrimSpace(parts[3])
	}
	if len(parts) > 4 && parts[4] != "" {
		result["serialNumber"] = strings.TrimSpace(parts[4])
	}
	if len(parts) > 5 && parts[5] != "" {
		result["operatingSystem"] = strings.TrimSpace(parts[5])
	}
	if len(parts) > 6 && parts[6] != "" {
		result["cpuModel"] = strings.TrimSpace(parts[6])
	}
	if len(parts) > 7 && parts[7] != "" {
		result["architecture"] = strings.TrimSpace(parts[7])
	}
	if len(parts) > 8 && parts[8] != "" {
		// 提取内存数值: "Memory:8GB" -> "8GB"
		memParts := strings.Split(parts[8], ":")
		if len(memParts) > 1 {
			result["memoryCapacity"] = strings.TrimSpace(memParts[1])
		} else {
			result["memoryCapacity"] = strings.TrimSpace(parts[8])
		}
	}
	if len(parts) > 9 && parts[9] != "" {
		// 提取硬盘数值: "Hard disk:119GB" -> "119GB"
		diskParts := strings.Split(parts[9], ":")
		if len(diskParts) > 1 {
			result["hardDiskCapacity"] = strings.TrimSpace(diskParts[1])
		} else {
			result["hardDiskCapacity"] = strings.TrimSpace(parts[9])
		}
	}
	if len(parts) > 10 && parts[10] != "" {
		result["lastOnlineTime"] = strings.TrimSpace(parts[10])
	}

	return result
}

// parseDateTime 解析日期时间字符串
// 支持格式: "2026/1/26 8:58:21", "2026-01-26 08:58:21"
func parseDateTime(dtStr string) *time.Time {
	if dtStr == "" {
		return nil
	}

	// 尝试多种格式
	formats := []string{
		"2006/1/2 15:4:5",
		"2006/1/2 15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dtStr); err == nil {
			return &t
		}
	}

	return nil
}

// syncComputers 同步电脑设备数据
func (s *ComputerService) syncComputers(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	// 批量获取所有 Computer DN
	computerDNs := make([]string, 0, len(entries))
	for _, entry := range entries {
		computerDNs = append(computerDNs, entry.DN)
	}

	// 批量查询已存在的 Computer
	var existingComputers []models.ADComputer
	s.db.WithContext(ctx).Where("ad_config_id = ? AND distinguished_name IN ?", config.ID, computerDNs).Find(&existingComputers)

	// 创建 DN -> Computer 的映射
	existingComputerMap := make(map[string]*models.ADComputer)
	for i := range existingComputers {
		existingComputerMap[existingComputers[i].DistinguishedName] = &existingComputers[i]
	}

	// 准备批量操作的数据
	var computersToCreate []models.ADComputer
	computersToUpdate := make(map[string]*models.ADComputer)
	now := time.Now()

	for _, entry := range entries {
		computerDN := entry.DN
		computerName := entry.GetAttributeValue("cn")
		if computerName == "" {
			continue
		}

		// 提取所属OU DN
		ouDN := extractParentDN(computerDN)

		// 获取原始描述
		description := entry.GetAttributeValue("description")
		parsedDesc := parseComputerDescription(description)

		// 解析时间字段
		var lastLogon, pwdLastSet *time.Time
		if lastLogonStr := entry.GetAttributeValue("lastLogon"); lastLogonStr != "" {
			lastLogon = parseFileTime(lastLogonStr)
		}
		if pwdLastSetStr := entry.GetAttributeValue("pwdLastSet"); pwdLastSetStr != "" {
			pwdLastSet = parseFileTime(pwdLastSetStr)
		}

		// 解析logonCount
		logonCount := parseIntOrDefault(entry.GetAttributeValue("logonCount"), 0)

		// 解析最后上线时间
		var lastOnlineTime *time.Time
		if onlineTimeStr := parsedDesc["lastOnlineTime"]; onlineTimeStr != "" {
			lastOnlineTime = parseDateTime(onlineTimeStr)
		}

		// 判断状态：如果最后上线时间在7天内则为在线
		status := models.ComputerStatusOffline
		if lastOnlineTime != nil && time.Since(*lastOnlineTime) < 7*24*time.Hour {
			status = models.ComputerStatusOnline
		}

		if existingComputer, exists := existingComputerMap[computerDN]; exists {
			// 已存在，更新
			existingComputer.ComputerName = computerName
			existingComputer.DistinguishedName = computerDN
			existingComputer.OUDN = ouDN
			existingComputer.OriginalDescription = description
			existingComputer.IPAddress = parsedDesc["ipAddress"]
			existingComputer.MacAddress = parsedDesc["macAddress"]
			existingComputer.ManagedBy = entry.GetAttributeValue("managedBy")
			existingComputer.OperatingSystem = entry.GetAttributeValue("operatingSystem")
			existingComputer.OSVersion = entry.GetAttributeValue("operatingSystemVersion")
			existingComputer.CPUModel = parsedDesc["cpuModel"]
			existingComputer.Architecture = parsedDesc["architecture"]
			existingComputer.MemoryCapacity = parsedDesc["memoryCapacity"]
			existingComputer.HardDiskCapacity = parsedDesc["hardDiskCapacity"]
			existingComputer.SerialNumber = parsedDesc["serialNumber"]
			existingComputer.SystemInfo = description
			existingComputer.LastLogon = lastLogon
			existingComputer.PasswordLastSet = pwdLastSet
			existingComputer.LogonCount = logonCount
			existingComputer.LastOnlineTime = lastOnlineTime
			existingComputer.Status = status
			existingComputer.UpdatedAt = now

			computersToUpdate[computerDN] = existingComputer
		} else {
			// 不存在，创建
			newComputer := models.ADComputer{
				ADConfigID:          config.ID,
				ComputerName:        computerName,
				DistinguishedName:    computerDN,
				OUDN:                ouDN,
				OriginalDescription: description,
				IPAddress:            parsedDesc["ipAddress"],
				MacAddress:           parsedDesc["macAddress"],
				ManagedBy:            entry.GetAttributeValue("managedBy"),
				OperatingSystem:      entry.GetAttributeValue("operatingSystem"),
				OSVersion:            entry.GetAttributeValue("operatingSystemVersion"),
				CPUModel:             parsedDesc["cpuModel"],
				Architecture:         parsedDesc["architecture"],
				MemoryCapacity:       parsedDesc["memoryCapacity"],
				HardDiskCapacity:     parsedDesc["hardDiskCapacity"],
				SerialNumber:         parsedDesc["serialNumber"],
				SystemInfo:           description,
				LastLogon:            lastLogon,
				PasswordLastSet:      pwdLastSet,
				LogonCount:           logonCount,
				LastOnlineTime:       lastOnlineTime,
				Status:               status,
			}
			computersToCreate = append(computersToCreate, newComputer)
		}
	}

	// 批量创建新的 Computer
	if len(computersToCreate) > 0 {
		batchSize := 100
		for i := 0; i < len(computersToCreate); i += batchSize {
			end := i + batchSize
			if end > len(computersToCreate) {
				end = len(computersToCreate)
			}
			if err := s.db.WithContext(ctx).Create(computersToCreate[i:end]).Error; err != nil {
				return fmt.Errorf("批量创建电脑设备失败: %w", err)
			}
		}
	}

	// 批量更新已存在的 Computer
	for _, computer := range computersToUpdate {
		if err := s.db.WithContext(ctx).Save(computer).Error; err != nil {
			return fmt.Errorf("更新电脑设备失败: %w", err)
		}
	}

	return nil
}
