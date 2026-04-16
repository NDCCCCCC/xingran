package addomain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	batchCreateSize     = 100
	batchUpdateSize     = 200 // 批量更新大小（PostgreSQL 参数限制 65535 / 28 字段 ≈ 2340，保守取 200）
	onlineThresholdDays = 7
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
	OUN          string `json:"ouDn,omitempty"`         // 所属OU
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

// normalizePagination 设置默认分页参数
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
		// 选择父OU时包含所有子OU: oudn = '选中的OU' OR oudn LIKE '%,选中的OU'
		query = query.Where("oudn = ? OR oudn LIKE ?", req.OUN, "%,"+req.OUN)
	}

	if req.ComputerName != "" {
		query = query.Where("computer_name LIKE ?", "%"+req.ComputerName+"%")
	}

	return query
}

// countComputers 统计电脑设备数量
func (s *ComputerService) countComputers(query *gorm.DB) (int64, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, fmt.Errorf("统计总数失败: %w", err)
	}
	return total, nil
}

// fetchComputers 获取电脑数据列表
func (s *ComputerService) fetchComputers(ctx context.Context, req *ComputerListRequest, total int64) ([]models.ADComputer, error) {
	query := s.buildComputerQuery(ctx, req)

	var computers []models.ADComputer
	offset := (req.Current - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).
		Order("created_at DESC").
		Find(&computers).Error; err != nil {
		return nil, fmt.Errorf("查询列表失败: %w", err)
	}

	return computers, nil
}

// convertToDetails 转换为详情格式
func (s *ComputerService) convertToDetails(computers []models.ADComputer) []ComputerDetail {
	details := make([]ComputerDetail, len(computers))
	for i, comp := range computers {
		details[i] = ComputerDetail{
			ADComputer:    comp,
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
		ADComputer:    computer,
		LastLogonUser: parseComputerDescriptionForUser(computer.OriginalDescription),
	}

	return detail, nil
}

// parseComputerDescriptionForUser 从描述中解析最后登录用户
func parseComputerDescriptionForUser(desc string) string {
	if desc == "" {
		return ""
	}

	parts := strings.Split(desc, "|")
	if len(parts) > 1 && parts[1] != "" {
		return strings.TrimSpace(parts[1])
	}

	return ""
}

// parseComputerDescription 解析电脑描述字段
func parseComputerDescription(desc string) map[string]string {
	result := make(map[string]string)

	if desc == "" {
		return result
	}

	parts := strings.Split(desc, "|")
	if len(parts) < 10 {
		return result
	}

	// 字段索引映射: 0: 空, 1: username, 2: ip, 3: mac, 4: serial, 5: os, 6: cpu, 7: arch, 8: memory, 9: disk, 10: datetime
	fieldMappings := map[int]string{
		1:  "lastLogonUser",
		2:  "ipAddress",
		3:  "macAddress",
		4:  "serialNumber",
		5:  "operatingSystem",
		6:  "cpuModel",
		7:  "architecture",
		10: "lastOnlineTime",
	}

	for idx, key := range fieldMappings {
		if len(parts) > idx && parts[idx] != "" {
			result[key] = strings.TrimSpace(parts[idx])
		}
	}

	// 特殊处理内存和硬盘容量
	if len(parts) > 8 && parts[8] != "" {
		result["memoryCapacity"] = extractCapacityValue(parts[8])
	}
	if len(parts) > 9 && parts[9] != "" {
		result["hardDiskCapacity"] = extractCapacityValue(parts[9])
	}

	return result
}

// extractCapacityValue 从容量字符串中提取数值
func extractCapacityValue(value string) string {
	parts := strings.Split(value, ":")
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(value)
}

// parseDateTime 解析日期时间字符串
func parseDateTime(dtStr string) *time.Time {
	if dtStr == "" {
		return nil
	}

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

// determineComputerStatus 确定电脑状态
func determineComputerStatus(lastOnlineTime *time.Time) models.ComputerStatus {
	if lastOnlineTime != nil && time.Since(*lastOnlineTime) < time.Duration(onlineThresholdDays*24)*time.Hour {
		return models.ComputerStatusOnline
	}
	return models.ComputerStatusOffline
}

// buildComputerFromEntry 从LDAP条目构建电脑对象
func buildComputerFromEntry(configID string, entry *ldap.Entry, parsedDesc map[string]string, status models.ComputerStatus) models.ADComputer {
	computerDN := entry.DN
	computerName := entry.GetAttributeValue("cn")

	var lastLogon, pwdLastSet *time.Time
	if lastLogonStr := entry.GetAttributeValue("lastLogon"); lastLogonStr != "" {
		lastLogon = parseFileTime(lastLogonStr)
	}
	if pwdLastSetStr := entry.GetAttributeValue("pwdLastSet"); pwdLastSetStr != "" {
		pwdLastSet = parseFileTime(pwdLastSetStr)
	}

	logonCount := parseIntOrDefault(entry.GetAttributeValue("logonCount"), 0)

	var lastOnlineTime *time.Time
	if onlineTimeStr := parsedDesc["lastOnlineTime"]; onlineTimeStr != "" {
		lastOnlineTime = parseDateTime(onlineTimeStr)
	}

	return models.ADComputer{
		ADConfigID:          configID,
		ComputerName:        computerName,
		DistinguishedName:   computerDN,
		OUDN:                extractParentDN(computerDN),
		OriginalDescription: entry.GetAttributeValue("description"),
		IPAddress:           parsedDesc["ipAddress"],
		MacAddress:          parsedDesc["macAddress"],
		ManagedBy:           entry.GetAttributeValue("managedBy"),
		OperatingSystem:     entry.GetAttributeValue("operatingSystem"),
		OSVersion:           entry.GetAttributeValue("operatingSystemVersion"),
		CPUModel:            parsedDesc["cpuModel"],
		Architecture:        parsedDesc["architecture"],
		MemoryCapacity:      parsedDesc["memoryCapacity"],
		HardDiskCapacity:    parsedDesc["hardDiskCapacity"],
		SerialNumber:        parsedDesc["serialNumber"],
		SystemInfo:          entry.GetAttributeValue("description"),
		LastLogon:           lastLogon,
		PasswordLastSet:     pwdLastSet,
		LogonCount:          logonCount,
		LastOnlineTime:      lastOnlineTime,
		Status:              status,
	}
}

// updateComputerFields 更新电脑对象字段
func updateComputerFields(computer *models.ADComputer, entry *ldap.Entry, parsedDesc map[string]string, status models.ComputerStatus, now time.Time) {
	computer.ComputerName = entry.GetAttributeValue("cn")
	computer.DistinguishedName = entry.DN
	computer.OUDN = extractParentDN(entry.DN)
	computer.OriginalDescription = entry.GetAttributeValue("description")
	computer.IPAddress = parsedDesc["ipAddress"]
	computer.MacAddress = parsedDesc["macAddress"]
	computer.ManagedBy = entry.GetAttributeValue("managedBy")
	computer.OperatingSystem = entry.GetAttributeValue("operatingSystem")
	computer.OSVersion = entry.GetAttributeValue("operatingSystemVersion")
	computer.CPUModel = parsedDesc["cpuModel"]
	computer.Architecture = parsedDesc["architecture"]
	computer.MemoryCapacity = parsedDesc["memoryCapacity"]
	computer.HardDiskCapacity = parsedDesc["hardDiskCapacity"]
	computer.SerialNumber = parsedDesc["serialNumber"]
	computer.SystemInfo = entry.GetAttributeValue("description")

	var lastLogon, pwdLastSet *time.Time
	if lastLogonStr := entry.GetAttributeValue("lastLogon"); lastLogonStr != "" {
		lastLogon = parseFileTime(lastLogonStr)
	}
	if pwdLastSetStr := entry.GetAttributeValue("pwdLastSet"); pwdLastSetStr != "" {
		pwdLastSet = parseFileTime(pwdLastSetStr)
	}

	computer.LastLogon = lastLogon
	computer.PasswordLastSet = pwdLastSet
	computer.LogonCount = parseIntOrDefault(entry.GetAttributeValue("logonCount"), 0)

	var lastOnlineTime *time.Time
	if onlineTimeStr := parsedDesc["lastOnlineTime"]; onlineTimeStr != "" {
		lastOnlineTime = parseDateTime(onlineTimeStr)
	}
	computer.LastOnlineTime = lastOnlineTime
	computer.Status = status
	computer.UpdatedAt = now
}

// batchCreate 批量创建电脑设备
func (s *ComputerService) batchCreate(ctx context.Context, computers []models.ADComputer) error {
	if len(computers) == 0 {
		return nil
	}

	start := time.Now()
	applogger.Infof("[电脑同步-batchCreate] 开始批量创建，总数: %d", len(computers))

	for i := 0; i < len(computers); i += batchCreateSize {
		batchStart := time.Now()
		end := i + batchCreateSize
		if end > len(computers) {
			end = len(computers)
		}
		if err := s.db.WithContext(ctx).Create(computers[i:end]).Error; err != nil {
			return fmt.Errorf("批量创建失败: %w", err)
		}
		applogger.Infof("[电脑同步-batchCreate] 批次 %d/%d (大小 %d): 耗时 %.2fs",
			(i/batchCreateSize)+1, (len(computers)+batchCreateSize-1)/batchCreateSize, end-i, time.Since(batchStart).Seconds())
	}

	applogger.Infof("[电脑同步-batchCreate] 完成，总耗时 %.2fs", time.Since(start).Seconds())
	return nil
}

// syncComputers 同步电脑设备数据
func (s *ComputerService) syncComputers(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	totalStart := time.Now()
	applogger.Infof("[电脑同步] 开始同步电脑设备，LDAP条目数: %d", len(entries))

	// 1. 提取DN列表
	step1Start := time.Now()
	computerDNs := extractDNs(entries)
	applogger.Infof("[电脑同步] 步骤1-提取DN: 耗时 %.2fs, DN数量: %d", time.Since(step1Start).Seconds(), len(computerDNs))

	// 2. 查询已存在的电脑（按DN）
	step2Start := time.Now()
	existingComputers := s.queryExistingComputers(ctx, config.ID, computerDNs)
	applogger.Infof("[电脑同步] 步骤2-查询已存在电脑(DN): 耗时 %.2fs, 找到 %d 条", time.Since(step2Start).Seconds(), len(existingComputers))

	// 3. 查询所有计算机名称（潜在瓶颈！）
	step3Start := time.Now()
	existingByName := s.queryAllComputerNames(ctx, config.ID)
	applogger.Infof("[电脑同步] 步骤3-查询所有计算机名称: 耗时 %.2fs, 找到 %d 条记录", time.Since(step3Start).Seconds(), len(existingByName))

	// 4. 构建映射
	step4Start := time.Now()
	existingComputerMap, existingComputerNameMap := s.buildComputerMaps(existingComputers, existingByName)
	applogger.Infof("[电脑同步] 步骤4-构建映射: 耗时 %.2fs", time.Since(step4Start).Seconds())

	// 5. 处理LDAP条目
	step5Start := time.Now()
	var computersToCreate []models.ADComputer
	computersToUpdate := make(map[string]*models.ADComputer)
	now := time.Now()

	for _, entry := range entries {
		computerName := entry.GetAttributeValue("cn")
		if computerName == "" {
			continue
		}

		description := entry.GetAttributeValue("description")
		parsedDesc := parseComputerDescription(description)

		var lastOnlineTime *time.Time
		if onlineTimeStr := parsedDesc["lastOnlineTime"]; onlineTimeStr != "" {
			lastOnlineTime = parseDateTime(onlineTimeStr)
		}

		status := determineComputerStatus(lastOnlineTime)

		s.processComputerEntry(entry, config.ID, parsedDesc, status, now, existingComputerMap, existingComputerNameMap, &computersToCreate, computersToUpdate)
	}
	applogger.Infof("[电脑同步] 步骤5-处理条目: 耗时 %.2fs, 待创建: %d, 待更新: %d", time.Since(step5Start).Seconds(), len(computersToCreate), len(computersToUpdate))

	// 6. 批量创建
	step6Start := time.Now()
	if err := s.batchCreate(ctx, computersToCreate); err != nil {
		return err
	}
	applogger.Infof("[电脑同步] 步骤6-批量创建: 耗时 %.2fs", time.Since(step6Start).Seconds())

	// 7. 批量更新
	step7Start := time.Now()
	if err := s.batchUpdate(ctx, computersToUpdate); err != nil {
		return err
	}
	applogger.Infof("[电脑同步] 步骤7-批量更新: 耗时 %.2fs", time.Since(step7Start).Seconds())

	applogger.Infof("[电脑同步] 完成同步，总耗时: %.2fs", time.Since(totalStart).Seconds())
	return nil
}

// queryExistingComputers 查询已存在的电脑（分批查询以优化性能）
const batchSize = 500 // 每批查询 500 个 DN，避免 IN 子句过大

func (s *ComputerService) queryExistingComputers(ctx context.Context, configID string, computerDNs []string) []models.ADComputer {
	if len(computerDNs) == 0 {
		return []models.ADComputer{}
	}

	// 使用 map 来自动去重
	dnMap := make(map[string]struct{})
	for _, dn := range computerDNs {
		dnMap[dn] = struct{}{}
	}

	// 将去重后的 DN 转回切片
	uniqueDNs := make([]string, 0, len(dnMap))
	for dn := range dnMap {
		uniqueDNs = append(uniqueDNs, dn)
	}

	// 分批查询
	var allComputers []models.ADComputer
	for i := 0; i < len(uniqueDNs); i += batchSize {
		end := i + batchSize
		if end > len(uniqueDNs) {
			end = len(uniqueDNs)
		}
		batch := uniqueDNs[i:end]

		var computers []models.ADComputer
		s.db.WithContext(ctx).
			Where("ad_config_id = ? AND distinguished_name IN ?", configID, batch).
			Find(&computers)
		allComputers = append(allComputers, computers...)
	}

	return allComputers
}

// queryAllComputerNames 查询所有计算机名称
func (s *ComputerService) queryAllComputerNames(ctx context.Context, configID string) []models.ADComputer {
	start := time.Now()
	var computers []models.ADComputer
	s.db.WithContext(ctx).
		Where("ad_config_id = ? AND deleted_at IS NULL", configID).
		Find(&computers)
	applogger.Infof("[电脑同步-queryAllComputerNames] 查询耗时 %.2fs，返回 %d 条记录", time.Since(start).Seconds(), len(computers))
	return computers
}

// buildComputerMaps 构建电脑映射
func (s *ComputerService) buildComputerMaps(existingByDN []models.ADComputer, allByName []models.ADComputer) (map[string]*models.ADComputer, map[string]*models.ADComputer) {
	dnMap := make(map[string]*models.ADComputer)
	nameMap := make(map[string]*models.ADComputer)

	for i := range existingByDN {
		dnMap[existingByDN[i].DistinguishedName] = &existingByDN[i]
	}

	for i := range allByName {
		nameMap[allByName[i].ComputerName] = &allByName[i]
	}

	return dnMap, nameMap
}

// processComputerEntry 处理单个电脑条目
func (s *ComputerService) processComputerEntry(
	entry *ldap.Entry,
	configID string,
	parsedDesc map[string]string,
	status models.ComputerStatus,
	now time.Time,
	existingDNMap map[string]*models.ADComputer,
	existingNameMap map[string]*models.ADComputer,
	toCreate *[]models.ADComputer,
	toUpdate map[string]*models.ADComputer,
) {
	computerDN := entry.DN
	computerName := entry.GetAttributeValue("cn")

	if existingComputer, exists := existingDNMap[computerDN]; exists {
		updateComputerFields(existingComputer, entry, parsedDesc, status, now)
		toUpdate[computerDN] = existingComputer
	} else if existingByName, nameExists := existingNameMap[computerName]; nameExists {
		updateComputerFields(existingByName, entry, parsedDesc, status, now)
		toUpdate[existingByName.DistinguishedName] = existingByName
	} else {
		newComputer := buildComputerFromEntry(configID, entry, parsedDesc, status)
		*toCreate = append(*toCreate, newComputer)
	}
}

// batchUpdate 批量更新电脑设备 - 使用 upsert（分批处理）
func (s *ComputerService) batchUpdate(ctx context.Context, computers map[string]*models.ADComputer) error {
	if len(computers) == 0 {
		return nil
	}

	start := time.Now()
	applogger.Infof("[电脑同步-batchUpdate] 开始批量更新，总数: %d", len(computers))

	computerSlice := make([]*models.ADComputer, 0, len(computers))
	for _, computer := range computers {
		computerSlice = append(computerSlice, computer)
	}

	// 分批处理，避免超过 PostgreSQL 参数限制 (65535)
	batchCount := 0
	for i := 0; i < len(computerSlice); i += batchUpdateSize {
		batchStart := time.Now()
		end := i + batchUpdateSize
		if end > len(computerSlice) {
			end = len(computerSlice)
		}
		batch := computerSlice[i:end]
		if len(batch) > 0 {
			// 使用数据库中实际的唯一约束 (ad_config_id, computer_name)
			if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "ad_config_id"}, {Name: "computer_name"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"distinguished_name", "oudn", "original_description", "ip_address",
					"mac_address", "managed_by", "operating_system", "os_version",
					"cpu_model", "architecture", "memory_capacity", "hard_disk_capacity",
					"serial_number", "system_info", "last_logon", "password_last_set",
					"logon_count", "last_online_time", "status", "updated_at",
				}),
			}).Create(batch).Error; err != nil {
				return fmt.Errorf("批量更新电脑设备失败: %w", err)
			}
			batchCount++
			applogger.Infof("[电脑同步-batchUpdate] 批次 %d (大小 %d): 耗时 %.2fs", batchCount, len(batch), time.Since(batchStart).Seconds())
		}
	}

	applogger.Infof("[电脑同步-batchUpdate] 完成，共 %d 批次，总耗时 %.2fs", batchCount, time.Since(start).Seconds())
	return nil
}
