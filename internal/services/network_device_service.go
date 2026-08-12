package services

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// NetworkDeviceService 网络设备管理服务
type NetworkDeviceService struct {
	db                      *gorm.DB
	discoveryService        *DeviceDiscoveryService
	deviceInfoCollectionSvc *DeviceInfoCollectionService
}

// NewNetworkDeviceService 创建网络设备管理服务
func NewNetworkDeviceService(db *gorm.DB, discoveryService *DeviceDiscoveryService, deviceInfoCollectionSvc *DeviceInfoCollectionService) *NetworkDeviceService {
	return &NetworkDeviceService{
		db:                      db,
		discoveryService:        discoveryService,
		deviceInfoCollectionSvc: deviceInfoCollectionSvc,
	}
}

// ListRequest 设备列表请求
type ListDeviceRequest struct {
	base.BaseListRequest
	DeviceName *string
	DeviceType *models.DeviceType
	Vendor     *models.DeviceVendor
	IP         *string
	Status     *models.DeviceStatus
	DeptID     *string
}

// networkDeviceAllowedSortFields 网络设备可排序字段白名单(对应 sys_network_device 表列名)。
var networkDeviceAllowedSortFields = map[string]string{
	"deviceName": "device_name",
	"deviceType": "device_type",
	"vendor":     "vendor",
	"model":      "model",
	"ipAddress":  "ip_address",
	"port":       "port",
	"status":     "status",
	"lastSeenAt": "last_seen_at",
	"createdAt":  "created_at",
}

// List 获取设备列表
func (s *NetworkDeviceService) List(ctx context.Context, req *ListDeviceRequest) ([]models.NetworkDevice, int64, error) {
	var devices []models.NetworkDevice
	var total int64

	query := s.db.WithContext(ctx).Model(&models.NetworkDevice{})

	if req.DeviceName != nil && *req.DeviceName != "" {
		query = query.Where("device_name LIKE ?", "%"+*req.DeviceName+"%")
	}
	if req.DeviceType != nil {
		query = query.Where("device_type = ?", *req.DeviceType)
	}
	if req.Vendor != nil {
		query = query.Where("vendor = ?", *req.Vendor)
	}
	if req.IP != nil && *req.IP != "" {
		query = query.Where("ip_address LIKE ?", "%"+*req.IP+"%")
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}
	if req.DeptID != nil && *req.DeptID != "" {
		query = query.Where("dept_id = ?", *req.DeptID)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询设备总数失败: %w", err)
	}

	// 分页查询 - 用户排序(白名单)优先,无 OrderByColumn 时保留 created_at DESC 默认
	offset := (req.Current - 1) * req.PageSize
	query = base.ApplySort(query, req.BaseListRequest, networkDeviceAllowedSortFields)
	if req.OrderByColumn == "" {
		query = query.Order("created_at DESC")
	}
	if err := query.Offset(offset).Limit(req.PageSize).Find(&devices).Error; err != nil {
		return nil, 0, fmt.Errorf("查询设备列表失败: %w", err)
	}

	// 加载关联信息
	s.loadAssociations(ctx, &devices)

	return devices, total, nil
}

// loadAssociations 加载关联信息（部门名称、凭证名称）
func (s *NetworkDeviceService) loadAssociations(ctx context.Context, devices *[]models.NetworkDevice) {
	if len(*devices) == 0 {
		return
	}

	// 收集部门ID和凭证ID
	var deptIDs, credentialIDs []string
	for _, d := range *devices {
		if d.DeptID != nil && *d.DeptID != "" {
			deptIDs = append(deptIDs, *d.DeptID)
		}
		if d.CredentialID != nil && *d.CredentialID != "" {
			credentialIDs = append(credentialIDs, *d.CredentialID)
		}
	}

	// 查询部门名称
	deptMap := make(map[string]string)
	if len(deptIDs) > 0 {
		var depts []models.Department
		s.db.WithContext(ctx).Where("id IN ?", deptIDs).Find(&depts)
		for _, dept := range depts {
			deptMap[dept.ID] = dept.DeptName
		}
	}

	// 查询凭证名称
	credentialMap := make(map[string]string)
	if len(credentialIDs) > 0 {
		var credentials []models.AuthCredential
		s.db.WithContext(ctx).Where("id IN ?", credentialIDs).Find(&credentials)
		for _, cred := range credentials {
			credentialMap[cred.ID] = cred.CredentialName
		}
	}

	// 填充关联信息
	for i := range *devices {
		if (*devices)[i].DeptID != nil {
			if deptName, ok := deptMap[*(*devices)[i].DeptID]; ok {
				(*devices)[i].DeptName = &deptName
			}
		}
		if (*devices)[i].CredentialID != nil {
			if credName, ok := credentialMap[*(*devices)[i].CredentialID]; ok {
				(*devices)[i].CredentialName = &credName
			}
		}
	}
}

// CreateRequest 创建设备请求
type CreateDeviceRequest struct {
	DeviceName   string
	DeviceType   models.DeviceType
	Vendor       models.DeviceVendor
	Model        string
	IPAddress    string
	Port         int
	SNMPPort     int
	CredentialID *string
	DeptID       *string
	Location     string
	Status       models.DeviceStatus
	Description  string
	CreatedBy    string
}

// QuickCreateRequest 快速创建设备请求
type QuickCreateRequest struct {
	IPAddress    string `json:"ipAddress" binding:"required,ip"`
	CredentialID string `json:"credentialId" binding:"required,uuid"`
	SNMPPort     int    `json:"snmpPort,omitempty"`
	// Communities 用户输入的多个 SNMP community 进行尝试
	Communities []string `json:"communities,omitempty"`
	DeptID      *string  `json:"deptId,omitempty"`
	Location    string   `json:"location,omitempty"`
	Description string   `json:"description,omitempty"`
	CreatedBy   string   `json:"-"`
}

// Create 创建设备
func (s *NetworkDeviceService) Create(ctx context.Context, req *CreateDeviceRequest) (*models.NetworkDevice, error) {
	// 检查IP地址是否已存在
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.NetworkDevice{}).Where("ip_address = ?", req.IPAddress).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("检查IP地址失败: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("IP地址已存在")
	}

	// 验证凭证是否存在
	if req.CredentialID != nil && *req.CredentialID != "" {
		var credentialCount int64
		if err := s.db.WithContext(ctx).Model(&models.AuthCredential{}).Where("id = ?", *req.CredentialID).Count(&credentialCount).Error; err != nil {
			return nil, fmt.Errorf("验证凭证失败: %w", err)
		}
		if credentialCount == 0 {
			return nil, fmt.Errorf("授权凭证不存在")
		}
	}

	// 验证部门是否存在
	if req.DeptID != nil && *req.DeptID != "" {
		var deptCount int64
		if err := s.db.WithContext(ctx).Model(&models.Department{}).Where("id = ?", *req.DeptID).Count(&deptCount).Error; err != nil {
			return nil, fmt.Errorf("验证部门失败: %w", err)
		}
		if deptCount == 0 {
			return nil, fmt.Errorf("部门不存在")
		}
	}

	device := &models.NetworkDevice{
		DeviceName:   req.DeviceName,
		DeviceType:   req.DeviceType,
		Vendor:       req.Vendor,
		Model:        req.Model,
		IPAddress:    req.IPAddress,
		Port:         req.Port,
		SNMPPort:     req.SNMPPort,
		CredentialID: req.CredentialID,
		DeptID:       req.DeptID,
		Location:     req.Location,
		Status:       req.Status,
		Description:  req.Description,
	}

	if err := s.db.WithContext(ctx).Create(device).Error; err != nil {
		return nil, fmt.Errorf("创建设备失败: %w", err)
	}

	return device, nil
}

// QuickCreateDevice 快速创建设备（自动探测）
// 通过 SNMP 探测获取设备信息并创建，后台异步通过 SSH 补充详细信息
func (s *NetworkDeviceService) QuickCreateDevice(ctx context.Context, req *QuickCreateRequest) (*models.NetworkDevice, error) {
	// 检查IP地址是否已存在（包括已删除的记录）
	var existingDevice models.NetworkDevice
	err := s.db.WithContext(ctx).Unscoped().Where("ip_address = ?", req.IPAddress).First(&existingDevice).Error

	// 如果设备已存在且未被删除，返回错误
	if err == nil && existingDevice.DeletedAt.Time.IsZero() {
		return nil, fmt.Errorf("IP地址 %s 的设备已存在，请使用编辑功能更新设备信息", req.IPAddress)
	}

	// 验证凭证是否存在
	var credential models.AuthCredential
	if credErr := s.db.WithContext(ctx).Where("id = ?", req.CredentialID).First(&credential).Error; credErr != nil {
		return nil, fmt.Errorf("授权凭证不存在")
	}

	// 验证部门是否存在（如果提供）
	if req.DeptID != nil && *req.DeptID != "" {
		var deptCount int64
		if deptErr := s.db.WithContext(ctx).Model(&models.Department{}).Where("id = ?", *req.DeptID).Count(&deptCount).Error; deptErr != nil {
			return nil, fmt.Errorf("验证部门失败: %w", deptErr)
		}
		if deptCount == 0 {
			return nil, fmt.Errorf("部门不存在")
		}
	}

	// 调用设备发现服务进行 SNMP 探测
	probeReq := &DeviceProbeRequest{
		IPAddress:    req.IPAddress,
		CredentialID: req.CredentialID,
		SNMPPort:     req.SNMPPort,
		Communities:  req.Communities,
	}

	probeResult, err := s.discoveryService.ProbeSingleDevice(ctx, probeReq)
	if err != nil {
		return nil, fmt.Errorf("设备探测失败: %w", err)
	}

	// 如果探测失败，阻止创建
	if !probeResult.Success {
		return nil, fmt.Errorf("设备探测失败: %s", probeResult.Message)
	}

	// 确定设备名称：优先使用 sysName，否则使用 "Device-IP最后一段"
	deviceName := probeResult.DeviceName
	if deviceName == "" {
		deviceName = fmt.Sprintf("Device-%s", getLastIPOctet(req.IPAddress))
	}

	// 确定设备类型：默认为交换机
	deviceType := probeResult.DeviceType
	if deviceType == "" {
		deviceType = models.DeviceTypeSwitch
	}

	// 如果设备存在但已删除，恢复并更新
	if err == nil && existingDevice.DeletedAt.Valid {
		// 恢复已删除的设备（清空 deleted_at）
		// now := time.Now()
		existingDevice.DeletedAt = gorm.DeletedAt{}
		existingDevice.DeviceName = deviceName
		existingDevice.DeviceType = deviceType
		existingDevice.Vendor = probeResult.Vendor
		existingDevice.Model = probeResult.Model
		existingDevice.Port = 22
		existingDevice.SNMPPort = 161
		existingDevice.CredentialID = &req.CredentialID
		existingDevice.DeptID = req.DeptID
		existingDevice.Location = req.Location
		existingDevice.Status = models.DeviceStatusOnline
		existingDevice.Description = req.Description
		existingDevice.UpdatedBy = req.CreatedBy

		if err := s.db.WithContext(ctx).Save(&existingDevice).Error; err != nil {
			return nil, fmt.Errorf("恢复设备失败: %w", err)
		}

		return &existingDevice, nil
	}

	// 创建新设备
	device := &models.NetworkDevice{
		BaseModel: models.BaseModel{
			CreatedBy: req.CreatedBy,
			UpdatedBy: req.CreatedBy,
		},
		DeviceName:   deviceName,
		DeviceType:   deviceType,
		Vendor:       probeResult.Vendor,
		Model:        probeResult.Model,
		IPAddress:    req.IPAddress,
		Port:         22,  // 默认 SSH 端口
		SNMPPort:     161, // 默认 SNMP 端口
		CredentialID: &req.CredentialID,
		DeptID:       req.DeptID,
		Location:     req.Location,
		Status:       models.DeviceStatusOnline, // 初始状态为在线
		Description:  req.Description,
	}

	if err := s.db.WithContext(ctx).Create(device).Error; err != nil {
		return nil, fmt.Errorf("创建设备失败: %w", err)
	}

	// 异步采集设备详细信息（通过 SSH 获取型号、序列号、版本等）
	if s.deviceInfoCollectionSvc != nil {
		if err := s.deviceInfoCollectionSvc.Enqueue(device.ID); err != nil {
			// 记录错误但不影响设备创建
			logger.Warnf("加入信息采集队列失败: %v", err)
		}
	}

	return device, nil
}

// getLastIPOctet 获取 IP 地址的最后一段
func getLastIPOctet(ip string) string {
	parts := []byte(ip)
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == '.' {
			return string(parts[i+1:])
		}
	}
	return ip
}

// GetByID 根据ID获取设备
func (s *NetworkDeviceService) GetByID(ctx context.Context, id string) (*models.NetworkDevice, error) {
	var device models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&device).Error; err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	// 加载关联信息
	s.loadAssociations(ctx, &[]models.NetworkDevice{device})

	return &device, nil
}

// UpdateRequest 更新设备请求
type UpdateDeviceRequest struct {
	ID           string
	DeviceName   string
	DeviceType   models.DeviceType
	Vendor       models.DeviceVendor
	Model        string
	IPAddress    string
	Port         int
	SNMPPort     int
	CredentialID *string
	DeptID       *string
	Location     string
	Status       models.DeviceStatus
	Description  string
	UpdatedBy    string
}

// Update 更新设备
func (s *NetworkDeviceService) Update(ctx context.Context, req *UpdateDeviceRequest) error {
	var device models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", req.ID).First(&device).Error; err != nil {
		return fmt.Errorf("设备不存在: %w", err)
	}

	// 检查IP地址是否被其他设备占用
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.NetworkDevice{}).
		Where("ip_address = ? AND id != ?", req.IPAddress, req.ID).Count(&count).Error; err != nil {
		return fmt.Errorf("检查IP地址失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("IP地址已被其他设备使用")
	}

	// 验证凭证是否存在
	if req.CredentialID != nil && *req.CredentialID != "" {
		var credentialCount int64
		if err := s.db.WithContext(ctx).Model(&models.AuthCredential{}).Where("id = ?", *req.CredentialID).Count(&credentialCount).Error; err != nil {
			return fmt.Errorf("验证凭证失败: %w", err)
		}
		if credentialCount == 0 {
			return fmt.Errorf("授权凭证不存在")
		}
	}

	// 验证部门是否存在
	if req.DeptID != nil && *req.DeptID != "" {
		var deptCount int64
		if err := s.db.WithContext(ctx).Model(&models.Department{}).Where("id = ?", *req.DeptID).Count(&deptCount).Error; err != nil {
			return fmt.Errorf("验证部门失败: %w", err)
		}
		if deptCount == 0 {
			return fmt.Errorf("部门不存在")
		}
	}

	updates := map[string]interface{}{
		"device_name":   req.DeviceName,
		"device_type":   req.DeviceType,
		"vendor":        req.Vendor,
		"model":         req.Model,
		"ip_address":    req.IPAddress,
		"port":          req.Port,
		"snmp_port":     req.SNMPPort,
		"credential_id": req.CredentialID,
		"dept_id":       req.DeptID,
		"location":      req.Location,
		"status":        req.Status,
		"description":   req.Description,
		"updated_by":    req.UpdatedBy,
	}

	if err := s.db.WithContext(ctx).Model(&device).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新设备失败: %w", err)
	}

	return nil
}

// Delete 删除设备
func (s *NetworkDeviceService) Delete(ctx context.Context, id string) error {
	var device models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&device).Error; err != nil {
		return fmt.Errorf("设备不存在: %w", err)
	}

	if err := s.db.WithContext(ctx).Delete(&device).Error; err != nil {
		return fmt.Errorf("删除设备失败: %w", err)
	}

	return nil
}

// BatchDelete 批量删除设备
func (s *NetworkDeviceService) BatchDelete(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := s.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// UpdateStatus 更新设备状态
func (s *NetworkDeviceService) UpdateStatus(ctx context.Context, id string, status models.DeviceStatus) error {
	if err := s.db.WithContext(ctx).Model(&models.NetworkDevice{}).
		Where("id = ?", id).Update("status", status).Error; err != nil {
		return fmt.Errorf("更新设备状态失败: %w", err)
	}
	return nil
}

// UpdateStatusBatch 批量更新设备状态
func (s *NetworkDeviceService) UpdateStatusBatch(ctx context.Context, ids []string, status models.DeviceStatus) error {
	if err := s.db.WithContext(ctx).Model(&models.NetworkDevice{}).
		Where("id IN ?", ids).Update("status", status).Error; err != nil {
		return fmt.Errorf("批量更新设备状态失败: %w", err)
	}
	return nil
}

// GetDeviceStatistics 获取设备统计信息
func (s *NetworkDeviceService) GetDeviceStatistics(ctx context.Context) (map[string]interface{}, error) {
	var stats struct {
		TotalDevices   int64
		OnlineDevices  int64
		OfflineDevices int64
		UnknownDevices int64
		ByType         map[string]int64
		ByVendor       map[string]int64
		ByDept         map[string]int64
	}

	s.db.WithContext(ctx).Model(&models.NetworkDevice{}).Count(&stats.TotalDevices)
	s.db.WithContext(ctx).Model(&models.NetworkDevice{}).Where("status = ?", models.DeviceStatusOnline).Count(&stats.OnlineDevices)
	s.db.WithContext(ctx).Model(&models.NetworkDevice{}).Where("status = ?", models.DeviceStatusOffline).Count(&stats.OfflineDevices)
	s.db.WithContext(ctx).Model(&models.NetworkDevice{}).Where("status = ?", models.DeviceStatusUnknown).Count(&stats.UnknownDevices)

	// 按类型统计
	// GORM 不支持把双列结果 (device_type, count) Scan 进 map[string]int64,
	// 会报 "expected 2 destination arguments in Scan, not 1";改用匿名 struct 切片接收再转 map。
	var typeRows []struct {
		DeviceType string
		Count      int64
	}
	s.db.WithContext(ctx).Model(&models.NetworkDevice{}).
		Select("device_type, COUNT(*) as count").
		Group("device_type").
		Scan(&typeRows)
	stats.ByType = make(map[string]int64, len(typeRows))
	for _, r := range typeRows {
		stats.ByType[r.DeviceType] = r.Count
	}

	// 按厂商统计
	var vendorRows []struct {
		Vendor string
		Count  int64
	}
	s.db.WithContext(ctx).Model(&models.NetworkDevice{}).
		Select("vendor, COUNT(*) as count").
		Group("vendor").
		Scan(&vendorRows)
	stats.ByVendor = make(map[string]int64, len(vendorRows))
	for _, r := range vendorRows {
		stats.ByVendor[r.Vendor] = r.Count
	}

	// 按部门统计 (部门表是 sys_dept, 非 sys_department; 见 models.Department.TableName)
	var deptRows []struct {
		DeptName string
		Count    int64
	}
	s.db.WithContext(ctx).Table("sys_network_device").
		Select("dept.dept_name, COUNT(*) as count").
		Joins("LEFT JOIN sys_dept dept ON dept.id = sys_network_device.dept_id").
		Group("dept.dept_name").
		Scan(&deptRows)
	stats.ByDept = make(map[string]int64, len(deptRows))
	for _, r := range deptRows {
		stats.ByDept[r.DeptName] = r.Count
	}

	return map[string]interface{}{
		"totalDevices":   stats.TotalDevices,
		"onlineDevices":  stats.OnlineDevices,
		"offlineDevices": stats.OfflineDevices,
		"unknownDevices": stats.UnknownDevices,
		"byType":         stats.ByType,
		"byVendor":       stats.ByVendor,
		"byDept":         stats.ByDept,
	}, nil
}

// GetDevicesByDept 根据部门获取设备列表
func (s *NetworkDeviceService) GetDevicesByDept(ctx context.Context, deptID string) ([]models.NetworkDevice, error) {
	var devices []models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("dept_id = ?", deptID).Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("查询部门设备失败: %w", err)
	}
	return devices, nil
}

// GetDevicesByCredential 根据凭证获取设备列表
func (s *NetworkDeviceService) GetDevicesByCredential(ctx context.Context, credentialID string) ([]models.NetworkDevice, error) {
	var devices []models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("credential_id = ?", credentialID).Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("查询凭证设备失败: %w", err)
	}
	return devices, nil
}
