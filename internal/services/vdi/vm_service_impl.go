package vdi

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// vmServiceImpl 虚拟机服务实现
type vmServiceImpl struct {
	db            *gorm.DB
	vdiClient     VDIClientExtended
	uuidValidator *regexp.Regexp
	clientMutex   sync.RWMutex
}

// NewVMService 创建虚拟机服务
func NewVMService(db *gorm.DB, vdiClient VDIClientExtended) VMService {
	return &vmServiceImpl{
		db:            db,
		vdiClient:     vdiClient,
		uuidValidator: constants.UuidPattern,
	}
}

// NewVMServiceWithDynamicClient 创建虚拟机服务（动态查找客户端）
func NewVMServiceWithDynamicClient(db *gorm.DB) VMService {
	return &vmServiceImpl{
		db:            db,
		vdiClient:     nil, // 将动态加载
		uuidValidator: constants.UuidPattern,
	}
}

// getClient 动态查找启用的VDI服务器并返回客户端
func (s *vmServiceImpl) getClient(ctx context.Context) (VDIClientExtended, error) {
	// Fast path: read lock
	s.clientMutex.RLock()
	if s.vdiClient != nil {
		s.clientMutex.RUnlock()
		return s.vdiClient, nil
	}
	s.clientMutex.RUnlock()

	// Slow path: write lock
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()

	// Double-check after acquiring write lock
	if s.vdiClient != nil {
		return s.vdiClient, nil
	}

	// 动态查找第一个启用的VDI服务器
	var server models.VDIServer
	if err := s.db.WithContext(ctx).
		Where("status = 0").
		Order("created_at ASC").
		First(&server).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no enabled VDI server found")
		}
		return nil, fmt.Errorf("failed to query VDI server: %w", err)
	}

	// 创建并缓存客户端
	s.vdiClient = NewVDIClientFromDB(s.db, server.ID)
	return s.vdiClient, nil
}

// syncVMsFromVDI 从VDI服务器同步虚拟机数据到本地数据库
func (s *vmServiceImpl) syncVMsFromVDI(ctx context.Context, client VDIClientExtended) error {
	// 用于收集所有从VDI获取的虚拟机ID
	vdiVMIDs := make(map[string]bool)
	vdiServerID, err := s.vdiServerID()
	if err != nil {
		return fmt.Errorf("failed to get VDI server ID: %w", err)
	}

	// 1. 获取所有资源组
	groups, err := client.ListResourceGroups(ctx)
	if err != nil {
		return fmt.Errorf("failed to list resource groups: %w", err)
	}

	// 1.1 同步资源组到本地数据库
	s.syncResourceGroups(ctx, groups, vdiServerID)

	// 2. 遍历每个资源组，获取其下的资源
	for _, group := range groups {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if group.Enable != "1" {
			continue // 跳过未启用的资源组
		}

		applogger.Debugf("[VDI] SYNC Processing resource group: %s (ID: %s)", group.Name, group.ID)

		// 2.1 获取该资源组下的所有资源
		resources, err := client.ListResources(ctx, group.ID)
		if err != nil {
			applogger.Warnf("[VDI] SYNC Failed to list resources for group %s: %v", group.Name, err)
			continue
		}

		applogger.Debugf("[VDI] SYNC Found %d resources in group %s", len(resources), group.Name)

		// 3. 遍历每个资源，获取虚拟机列表
		for _, resource := range resources {
			applogger.Debugf("[VDI] SYNC Processing resource: %s (ID: %d)", resource.Name, resource.ID)

			// 分页获取该资源下的所有虚拟机
			page := 1
			pageSize := 100
			for {
				vms, total, err := client.ListResourceServers(ctx, fmt.Sprintf("%d", resource.ID), page, pageSize)
				if err != nil {
					// 记录错误但继续处理其他资源
					applogger.Warnf("[VDI] SYNC Failed to list VMs for resource %s: %v", resource.Name, err)
					break
				}

				// 保存虚拟机数据到本地数据库并收集ID
				for _, vm := range vms {
					vdiVMIDs[vm.ID] = true // 标记该虚拟机存在于VDI服务器
					s.saveOrUpdateVM(ctx, vm, fmt.Sprintf("%d", resource.ID), vdiServerID)
				}

				// 检查是否还有更多数据
				if page*pageSize >= total {
					break
				}
				page++
			}
		}
	}

	// 4. 清理孤立的虚拟机记录（VDI服务器上不存在的虚拟机）
	s.cleanupOrphanedVMs(ctx, vdiVMIDs, vdiServerID)

	return nil
}

// syncResourceGroups 同步资源组到本地数据库
func (s *vmServiceImpl) syncResourceGroups(ctx context.Context, groups []VDIResourceGroup, vdiServerID string) {
	applogger.Debugf("[VDI] SYNC Syncing %d resource groups for server %s", len(groups), vdiServerID)

	for _, group := range groups {
		// 将VDI的enable字段映射到status: enable=1 -> status=0(正常), 其他 -> status=1(停用)
		status := 1
		if group.Enable == "1" {
			status = 0
		}

		s.saveOrUpdateResourceGroup(ctx, group, vdiServerID, status)
	}

	// 清理VDI服务器上已不存在的资源组（软删除）
	var existingGroupIDs []string
	for _, g := range groups {
		existingGroupIDs = append(existingGroupIDs, g.ID)
	}

	if len(existingGroupIDs) > 0 {
		result := s.db.WithContext(ctx).
			Where("vdi_server_id = ? AND resource_group_id NOT IN ? AND deleted_at IS NULL", vdiServerID, existingGroupIDs).
			Delete(&models.VDIResourceGroup{})
		if result.Error != nil {
			applogger.Warnf("[VDI] SYNC Failed to cleanup orphaned resource groups: %v", result.Error)
		} else if result.RowsAffected > 0 {
			applogger.Debugf("[VDI] SYNC Cleaned up %d orphaned resource groups", result.RowsAffected)
		}
	}
}

// saveOrUpdateResourceGroup 保存或更新资源组记录
func (s *vmServiceImpl) saveOrUpdateResourceGroup(ctx context.Context, group VDIResourceGroup, vdiServerID string, status int) {
	var existing models.VDIResourceGroup
	err := s.db.WithContext(ctx).
		Where("resource_group_id = ? AND vdi_server_id = ?", group.ID, vdiServerID).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		record := models.VDIResourceGroup{
			ResourceGroupID: group.ID,
			Name:            group.Name,
			VdiServerID:     vdiServerID,
			Type:            group.Note, // 使用Note字段作为Type描述
			Status:          status,
		}
		if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
			applogger.Warnf("[VDI] SYNC Failed to create resource group %s: %v", group.Name, err)
		}
	} else if err == nil {
		// 更新现有记录
		updates := map[string]interface{}{
			"name":   group.Name,
			"type":   group.Note,
			"status": status,
		}
		if err := s.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
			applogger.Warnf("[VDI] SYNC Failed to update resource group %s: %v", group.Name, err)
		}
	} else {
		applogger.Warnf("[VDI] SYNC Failed to query resource group %s: %v", group.Name, err)
	}
}

// cleanupOrphanedVMs 清理孤立的虚拟机记录
func (s *vmServiceImpl) cleanupOrphanedVMs(ctx context.Context, vdiVMIDs map[string]bool, vdiServerID string) {
	applogger.Debugf("[VDI] SYNC Starting orphaned VMs cleanup...")

	// 获取本地数据库中该VDI服务器的所有虚拟机
	var localVMs []models.VDIVirtualMachine
	err := s.db.WithContext(ctx).
		Where("vdi_server_id = ? AND deleted_at IS NULL", vdiServerID).
		Find(&localVMs).Error
	if err != nil {
		applogger.Warnf("[VDI] SYNC Failed to query local VMs: %v", err)
		return
	}

	// 找出孤立的虚拟机（本地存在但VDI服务器上不存在）
	var orphanedIDs []string
	for _, localVM := range localVMs {
		if !vdiVMIDs[localVM.VMID] {
			orphanedIDs = append(orphanedIDs, localVM.ID)
			applogger.Debugf("[VDI] SYNC Found orphaned VM: %s (VM ID: %s)", localVM.Name, localVM.VMID)
		}
	}

	// 删除孤立的虚拟机记录（软删除）
	if len(orphanedIDs) > 0 {
		result := s.db.WithContext(ctx).
			Where("id IN ?", orphanedIDs).
			Delete(&models.VDIVirtualMachine{})
		if result.Error != nil {
			applogger.Warnf("[VDI] SYNC Failed to delete orphaned VMs: %v", result.Error)
		} else {
			applogger.Debugf("[VDI] SYNC Successfully cleaned up %d orphaned VM records", result.RowsAffected)
		}
	} else {
		applogger.Debugf("[VDI] SYNC No orphaned VMs found, database is in sync")
	}
}

// getBestIPAddress 获取最佳IP地址
// 优先使用AssignIP（分配的IP地址），如果为空或"-"则使用IP（当前实际IP）
// 这样可以确保DHCP虚拟机在关机状态下也能显示配置的IP地址
func getBestIPAddress(resource VDIVMResource) string {
	ipAddress := resource.AssignIP
	if ipAddress == "" || ipAddress == "-" {
		ipAddress = resource.IP
	}
	return ipAddress
}

// saveOrUpdateVM 保存或更新虚拟机记录
func (s *vmServiceImpl) saveOrUpdateVM(ctx context.Context, resource VDIVMResource, resourceID string, vdiServerID string) {
	var vm models.VDIVirtualMachine
	// 使用 Unscoped() 包含软删除记录，避免重复插入导致唯一约束冲突
	err := s.db.WithContext(ctx).
		Unscoped().
		Where("vm_id = ?", resource.ID).
		First(&vm).Error

	// 获取最佳IP地址
	ipAddress := getBestIPAddress(resource)

	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		vm = models.VDIVirtualMachine{
			VMID:        resource.ID,
			Name:        resource.VMName,
			ResourceID:  resourceID,
			PowerState:  s.mapPowerState(resource.Status),
			IPAddress:   ipAddress,
			MACAddress:  resource.MAC,
			OSType:      resource.OSType,
			CPUNumber:   s.parseIntSafe(resource.CPUNumber),
			CPUCore:     s.parseIntSafe(resource.CPUCore),
			CPUPer:      s.parseIntSafe(resource.CPUPer),
			Memory:      s.parseIntSafe(resource.MemAll),
			MemoryPer:   s.parseIntSafe(resource.MemPer),
			Disk:        s.parseIntSafe(resource.DiscAll),
			DiskPer:     s.parseIntSafe(resource.DiscPer),
			// 网络配置信息
			IPType:        s.mapIPType(resource.IPState),
			SubnetMask:    resource.Subnetmask,
			DefaultGateway: resource.DefaultGateway,
			NameServer:    resource.NameServer,
			AssignIP:      resource.AssignIP,
			VdiServerID:   vdiServerID,
			LastSyncAt:    &time.Time{},
		}
		// 调试日志：打印CPU使用率原始值和解析后的值
		if resource.CPUPer != "" {
			applogger.Debugf("[VDI] SYNC DEBUG VM: %s, CPUPer原始值: '%s', 解析后: %d",
				resource.VMName, resource.CPUPer, s.parseIntSafe(resource.CPUPer))
		}
		s.db.WithContext(ctx).Create(&vm)
	} else if err == nil {
		// 检查记录是否已被软删除
		if vm.DeletedAt.Valid {
			// 永久删除旧的软删除记录，避免唯一约束冲突
			s.db.WithContext(ctx).Unscoped().Delete(&vm)
			// 创建新记录
			vm = models.VDIVirtualMachine{
				VMID:        resource.ID,
				Name:        resource.VMName,
				ResourceID:  resourceID,
				PowerState:  s.mapPowerState(resource.Status),
				IPAddress:   ipAddress,
				MACAddress:  resource.MAC,
				OSType:      resource.OSType,
				CPUNumber:   s.parseIntSafe(resource.CPUNumber),
				CPUCore:     s.parseIntSafe(resource.CPUCore),
				CPUPer:      s.parseIntSafe(resource.CPUPer),
				Memory:      s.parseIntSafe(resource.MemAll),
				MemoryPer:   s.parseIntSafe(resource.MemPer),
				Disk:        s.parseIntSafe(resource.DiscAll),
				DiskPer:     s.parseIntSafe(resource.DiscPer),
				IPType:        s.mapIPType(resource.IPState),
				SubnetMask:    resource.Subnetmask,
				DefaultGateway: resource.DefaultGateway,
				NameServer:    resource.NameServer,
				AssignIP:      resource.AssignIP,
				VdiServerID:   vdiServerID,
				LastSyncAt:    &time.Time{},
			}
			if resource.CPUPer != "" {
				applogger.Debugf("[VDI] SYNC DEBUG VM: %s, 清理软删除记录后重新创建, CPUPer: %d",
					resource.VMName, s.parseIntSafe(resource.CPUPer))
			}
			s.db.WithContext(ctx).Create(&vm)
		} else {
			// 更新现有记录
			updates := map[string]interface{}{
				"name":          resource.VMName,
				"resource_id":   resourceID,
				"power_state":   s.mapPowerState(resource.Status),
				"mac_address":   resource.MAC,
				"os_type":       resource.OSType,
				"cpu_number":    s.parseIntSafe(resource.CPUNumber),
				"cpu_core":      s.parseIntSafe(resource.CPUCore),
				"cpu_per":       s.parseIntSafe(resource.CPUPer),
				"last_sync_at":  time.Now(),
				"memory":        s.parseIntSafe(resource.MemAll),
				"memory_per":    s.parseIntSafe(resource.MemPer),
				"disk":          s.parseIntSafe(resource.DiscAll),
				"disk_per":      s.parseIntSafe(resource.DiscPer),
			}

			// Smart IP update strategy
			// For DHCP VMs, if new IP is empty, keep existing IP (last known DHCP address)
			// For static IP VMs, or when new IP is not empty, use new IP
			if ipAddress == "" || ipAddress == "-" {
				// New IP is empty, check if we should keep old IP
				if vm.IPType == "dhcp" && vm.IPAddress != "" && vm.IPAddress != "-" {
					// DHCP VM with history IP, keep old IP
					updates["ip_address"] = vm.IPAddress
					applogger.Debugf("[VDI] SYNC VM %s: DHCP mode new IP empty, keeping history IP %s", resource.VMName, vm.IPAddress)
				} else {
					// Static IP or no history IP, use new value (may be empty)
					updates["ip_address"] = ipAddress
				}
			} else {
				// New IP not empty, use new IP
				updates["ip_address"] = ipAddress
			}

			// 网络配置信息更新
			updates["ip_type"] = s.mapIPType(resource.IPState)
			updates["subnet_mask"] = resource.Subnetmask
			updates["default_gateway"] = resource.DefaultGateway
			updates["name_server"] = resource.NameServer
			updates["assign_ip"] = resource.AssignIP
			s.db.WithContext(ctx).Model(&vm).Updates(updates)
		}
	}
}

// mapPowerState 映射VDI状态码到虚拟机状态
func (s *vmServiceImpl) mapPowerState(status string) string {
	// VDI虚拟机状态码映射（根据深信服文档）
	// 10=等待agent接入, 11=关机, 12=挂起, 13=待确认, 14=待确认, 15=正常使用
	switch status {
	case "10":
		return "pending"   // 等待agent接入
	case "11":
		return "stopped"   // 关机
	case "12":
		return "suspended" // 挂起
	case "13":
		return "unknown13" // 待确认
	case "14":
		return "unknown14" // 待确认
	case "15":
		return "in_use"    // 正常使用
	default:
		return "unknown"
	}
}

// mapIPType 映射VDI IP状态到IP类型
func (s *vmServiceImpl) mapIPType(ipState string) string {
	// VDI IP状态映射
	// 0=DHCP, 1=静态IP
	switch ipState {
	case "0":
		return "DHCP"
	case "1":
		return "STATIC"
	default:
		return "DHCP" // 默认DHCP
	}
}

// parseIntSafe 安全解析整数
func (s *vmServiceImpl) parseIntSafe(str string) int {
	var i int
	fmt.Sscanf(str, "%d", &i)
	return i
}

// vdiServerID 获取当前VDI服务器ID
func (s *vmServiceImpl) vdiServerID() (string, error) {
	var server models.VDIServer
	err := s.db.Where("status = 0").First(&server).Error
	if err != nil {
		return "", fmt.Errorf("failed to query VDI server: %w", err)
	}
	return server.ID, nil
}

// ListResourceGroups 查询资源组列表（从本地数据库查询，不调用VDI API）
func (s *vmServiceImpl) ListResourceGroups(ctx context.Context, vdiServerID string) ([]VDIResourceGroupDTO, error) {
	query := s.db.WithContext(ctx).Model(&models.VDIResourceGroup{}).Where("status = 0")

	if vdiServerID != "" {
		query = query.Where("vdi_server_id = ?", vdiServerID)
	}

	var groups []models.VDIResourceGroup
	if err := query.Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("failed to query resource groups: %w", err)
	}

	dtos := make([]VDIResourceGroupDTO, len(groups))
	for i, g := range groups {
		dtos[i] = VDIResourceGroupDTO{
			ResourceGroupID: g.ResourceGroupID,
			Name:            g.Name,
			VdiServerID:     g.VdiServerID,
			Type:            g.Type,
		}
	}

	return dtos, nil
}

// ListResources 查询资源列表（调用VDI API获取资源组下的具体计算资源）
func (s *vmServiceImpl) ListResources(ctx context.Context, vdiServerID string, groupID string) ([]VDIResourceDTO, error) {
	if groupID == "" {
		return nil, fmt.Errorf("group_id is required")
	}

	// 获取VDI客户端
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get VDI client: %w", err)
	}

	// 调用VDI API获取资源列表
	resources, err := client.ListResources(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	// 转换为DTO
	dtos := make([]VDIResourceDTO, len(resources))
	for i, r := range resources {
		dtos[i] = VDIResourceDTO{
			ID:    r.ID,
			Name:  r.Name,
			Note:  r.Note,
			GrpID: r.GrpID,
		}
	}

	return dtos, nil
}

// CreateVM 创建虚拟机（调用VDI API）
func (s *vmServiceImpl) CreateVM(ctx context.Context, req *CreateVMServiceRequest) (*VDIVMDTO, error) {
	// 1. 验证VDI服务器存在
	var server models.VDIServer
	if err := s.db.WithContext(ctx).Where("id = ? AND status = 0", req.VdiServerID).First(&server).Error; err != nil {
		return nil, fmt.Errorf("VDI server not found or disabled: %w", err)
	}

	// 2. 获取VDI客户端
	client := NewVDIClientFromDB(s.db, req.VdiServerID)

	// 3. 解析资源ID为整数
	var resourceID int
	fmt.Sscanf(req.ResourceID, "%d", &resourceID)

	// 4. 构建VDI API创建请求
	createReq := CreateServerRequest{
		Resource: ResourceInfo{ID: resourceID},
		Host:     HostInfo{ID: req.HostID},
		RunPosition: PositionInfo{ID: req.RunPositionID},
		Disk:     DiskInfo{ID: req.DiskID},
		Storage:  StorageInfo{ID: req.StorageID},
		Network:  NetworkInfo{ID: req.NetworkID},
		Servers:  ServerCount{Count: req.Count},
	}

	// 5. 调用VDI API创建虚拟机
	resp, err := client.CreateServer(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM via VDI API: %w", err)
	}

	// 6. 创建本地虚拟机记录（使用VDI返回的VM ID）
	// 如果创建多个VM，只记录第一个
	vmID := ""
	if len(resp.Data.ServerID) > 0 {
		vmID = resp.Data.ServerID[0]
	} else {
		return nil, fmt.Errorf("VDI API returned no VM IDs")
	}

	vm := &models.VDIVirtualMachine{
		VMID:        vmID,
		Name:        "", // 名称将从VDI服务器同步获取，不使用用户输入
		ResourceID:  req.ResourceID,
		VdiServerID: req.VdiServerID,
		CPUNumber:   req.CPUNumber,
		CPUCore:     req.CPUCore,
		Memory:      req.Memory,
		Disk:        req.Disk,
		PowerState:  "stopped",
	}

	if err := s.db.WithContext(ctx).Create(vm).Error; err != nil {
		return nil, fmt.Errorf("failed to create VM record: %w", err)
	}

	return s.toDTO(vm), nil
}

// GetVM 获取虚拟机详情
func (s *vmServiceImpl) GetVM(ctx context.Context, id string) (*VDIVMDTO, error) {
	var vm models.VDIVirtualMachine
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&vm).Error; err != nil {
		return nil, fmt.Errorf("VM not found: %w", err)
	}

	return s.toDTO(&vm), nil
}

// vmAllowedSortFields 虚拟机列表可排序字段白名单（对应 vdi_virtual_machine 表列名）。
var vmAllowedSortFields = map[string]string{
	"name":        "name",
	"powerState":  "power_state",
	"vdiServerId": "vdi_server_id",
	"createdAt":   "created_at",
}

// ListVMs 获取虚拟机列表
func (s *vmServiceImpl) ListVMs(ctx context.Context, req *ListVMRequest, userID string, dataScope models.DataScope) (*PageResult, error) {
	// 检查是否有启用的VDI服务器配置
	var serverCount int64
	if err := s.db.WithContext(ctx).Model(&models.VDIServer{}).Where("status = 0").Count(&serverCount).Error; err != nil {
		return nil, fmt.Errorf("failed to check VDI servers: %w", err)
	}

	// 如果没有启用的VDI服务器，返回空列表并提示
	if serverCount == 0 {
		return &PageResult{
			List:     []VDIVMDTO{},
			Total:    0,
			Page:     req.Page,
			PageSize: req.PageSize,
		}, nil
	}

	// 获取VDI客户端
	client, err := s.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get VDI client: %w", err)
	}

	// 检查本地数据库是否有虚拟机数据
	var localCount int64
	if err := s.db.WithContext(ctx).Model(&models.VDIVirtualMachine{}).Count(&localCount).Error; err != nil {
		return nil, fmt.Errorf("failed to check local VMs: %w", err)
	}

	// 如果本地数据为空，从VDI服务器同步数据
	if localCount == 0 {
		if err := s.syncVMsFromVDI(ctx, client); err != nil {
			return nil, fmt.Errorf("failed to sync VMs from VDI: %w", err)
		}
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 10
	}

	// 构建查询
	query := s.db.WithContext(ctx).Model(&models.VDIVirtualMachine{})

	// 添加过滤条件
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.VdiServerID != "" {
		query = query.Where("vdi_server_id = ?", req.VdiServerID)
	}
	if req.ResourceID != "" {
		query = query.Where("resource_id = ?", req.ResourceID)
	}
	if req.PowerState != "" {
		query = query.Where("power_state = ?", req.PowerState)
	}

	// Apply data scope filtering using explicit parameters from handler
	// (DataScopePermission middleware sets these in Gin context, handler extracts and passes them)
	if userID != "" && dataScope > 0 {
		query = ApplyVMDataScopeFilter(query, userID, dataScope, s.db)
		query = ApplyBoundUserFilter(query, dataScope)
	}
	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count VMs: %w", err)
	}

	// 分页查询：用户排序（白名单）优先，无 OrderByColumn 时保持 DB 默认顺序
	var vms []models.VDIVirtualMachine
	offset := (req.Page - 1) * req.PageSize
	sortReq := base.BaseListRequest{
		Current:       req.Page,
		PageSize:      req.PageSize,
		OrderByColumn: req.OrderByColumn,
		IsAsc:         req.IsAsc,
	}
	query = base.ApplySort(query, sortReq, vmAllowedSortFields)
	if err := query.Offset(offset).Limit(req.PageSize).Find(&vms).Error; err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	// 转换为DTO
	dtos := make([]VDIVMDTO, len(vms))
	for i, vm := range vms {
		dtos[i] = *s.toDTO(&vm)
	}

	return &PageResult{
		List:     dtos,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// UpdateVM 更新虚拟机信息
func (s *vmServiceImpl) UpdateVM(ctx context.Context, id string, req *UpdateVMRequest) error {
	var vm models.VDIVirtualMachine
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&vm).Error; err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.IPAddress != nil {
		updates["ip_address"] = *req.IPAddress
	}
	if req.MACAddress != nil {
		updates["mac_address"] = *req.MACAddress
	}

	if err := s.db.WithContext(ctx).Model(&vm).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update VM: %w", err)
	}

	return nil
}

// DeleteVM 删除虚拟机（调用VDI API）
func (s *vmServiceImpl) DeleteVM(ctx context.Context, ids []string) error {
	// 获取VDI客户端
	client, err := s.getClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get VDI client: %w", err)
	}

	// 1. 查询本地VM记录，获取VM ID
	var vms []models.VDIVirtualMachine
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&vms).Error; err != nil {
		return fmt.Errorf("failed to query VMs: %w", err)
	}

	if len(vms) == 0 {
		return fmt.Errorf("no VMs found")
	}

	vmIDs := make([]string, len(vms))
	for i, vm := range vms {
		vmIDs[i] = vm.VMID
	}

	// 2. 调用VDI API删除虚拟机
	vdiErr := client.DeleteVM(ctx, vmIDs)

	// 检查是否为 error_code 3001（虚拟机ID不存在）
	if vdiErr != nil {
		var vdiAPIErr *VDIError
		if errors.As(vdiErr, &vdiAPIErr) && vdiAPIErr.Code == 3001 {
			// 特殊处理：VDI服务器上不存在该虚拟机，记录警告但继续删除本地记录
			applogger.Warnf("[VDI] DELETE VM not found on VDI server (error_code 3001), cleaning up local orphaned records. VM IDs: %v", vmIDs)
		} else {
			// 其他错误：返回原错误，不删除本地记录
			return fmt.Errorf("failed to delete VMs from VDI: %w", vdiErr)
		}
	}

	// 3. 删除本地记录（软删除）
	// 无论VDI API是否成功（除了3001以外的错误），都删除本地记录
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.VDIVirtualMachine{}).Error; err != nil {
		return fmt.Errorf("failed to delete VM records: %w", err)
	}

	return nil
}

// OperateVM 操作虚拟机（调用VDI API）
func (s *vmServiceImpl) OperateVM(ctx context.Context, req *VMOperateRequest) error {
	// 获取VDI客户端
	client, err := s.getClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get VDI client: %w", err)
	}

	// 1. 查询本地VM记录，获取VM ID
	var vms []models.VDIVirtualMachine
	if err := s.db.WithContext(ctx).Where("id IN ?", req.VMIDs).Find(&vms).Error; err != nil {
		return fmt.Errorf("failed to query VMs: %w", err)
	}

	if len(vms) == 0 {
		return fmt.Errorf("no VMs found")
	}

	vmIDs := make([]string, len(vms))
	for i, vm := range vms {
		vmIDs[i] = vm.VMID
	}

	// 2. 调用VDI API操作虚拟机
	action := string(req.Action)
	if err := client.OperateVM(ctx, vmIDs, action); err != nil {
		return fmt.Errorf("failed to operate VMs: %w", err)
	}

	// 3. 更新本地记录的电源状态
	// 注意：这里可以异步更新，通过SyncVMFromVDI来获取最新状态
	powerState := map[VMPowerAction]string{
		VMPowerOn:      "running",
		VMPowerOff:     "stopped",
		VMPowerRestart: "running",
		VMPowerSuspend: "suspended",
	}

	if state, ok := powerState[req.Action]; ok {
		s.db.WithContext(ctx).
			Model(&models.VDIVirtualMachine{}).
			Where("vm_id IN ?", vmIDs).
			Update("power_state", state)
	}

	return nil
}

// BindUser 绑定用户（仅更新本地权限标识，不调用VDI API）
func (s *vmServiceImpl) BindUser(ctx context.Context, vmID string, req *BindUserServiceRequest) error {
	// 1. 查询本地VM记录
	var vm models.VDIVirtualMachine
	if err := s.db.WithContext(ctx).Where("id = ?", vmID).First(&vm).Error; err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	// 2. Look up system user by username to build display name
	var systemUser models.User
	if err := s.db.WithContext(ctx).Where("username = ? AND deleted_at IS NULL", req.Username).First(&systemUser).Error; err != nil {
		return fmt.Errorf("system user not found: %s: %w", req.Username, err)
	}

	// 3. Build display name: nickname (username)
	displayName := req.Username
	if systemUser.Nickname != nil && *systemUser.Nickname != "" {
		displayName = fmt.Sprintf("%s (%s)", *systemUser.Nickname, req.Username)
	}

	// 4. 仅更新本地记录（不调用VDI API）
	updates := map[string]interface{}{
		"bound_user_id":   systemUser.ID, // Store UUID for data scope filtering
		"bound_user_name": displayName,
	}

	if err := s.db.WithContext(ctx).Model(&vm).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update VM record: %w", err)
	}

	return nil
}

// UnbindUser 解绑用户（仅更新本地权限标识，不调用VDI API）
func (s *vmServiceImpl) UnbindUser(ctx context.Context, vmID string) error {
	// 1. 查询本地VM记录
	var vm models.VDIVirtualMachine
	if err := s.db.WithContext(ctx).Where("id = ?", vmID).First(&vm).Error; err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	// 2. 仅更新本地记录（不调用VDI API）
	updates := map[string]interface{}{
		"bound_user_id":   nil,
		"bound_user_name": nil,
	}

	if err := s.db.WithContext(ctx).Model(&vm).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update VM record: %w", err)
	}

	return nil
}

// SyncVMFromVDI 从VDI同步虚拟机状态（调用VDI API）
func (s *vmServiceImpl) SyncVMFromVDI(ctx context.Context, vmID string) error {
	// 获取VDI客户端
	client, err := s.getClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get VDI client: %w", err)
	}

	// 1. 查询本地VM记录
	var vm models.VDIVirtualMachine
	if err := s.db.WithContext(ctx).Where("id = ?", vmID).First(&vm).Error; err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	// 2. 从VDI API获取该虚拟机所属资源的所有虚拟机，然后查找目标VM
	// 注意：VDI API 可能不支持直接通过 VM ID 获取单个虚拟机，
	// 所以我们需要从资源列表中获取所有虚拟机，然后查找目标VM
	vms, _, err := client.ListResourceServers(ctx, vm.ResourceID, 1, 1000)
	if err != nil {
		return fmt.Errorf("failed to fetch VMs from VDI: %w", err)
	}

	// 3. 从返回的虚拟机列表中查找目标虚拟机
	// VDI API返回的虚拟机ID在 _id 字段中
	var targetVM *VDIVMResource
	for i := range vms {
		if vms[i].ID == vm.VMID {
			targetVM = &vms[i]
			break
		}
	}

	if targetVM == nil {
		return fmt.Errorf("VM %s not found in VDI server (may have been deleted)", vm.VMID)
	}

	// 4. 更新本地记录（包含名称同步，使用最佳IP地址）
	vm.Name = targetVM.VMName
	vm.PowerState = s.mapPowerState(targetVM.Status)
	vm.IPAddress = getBestIPAddress(*targetVM)
	vm.MACAddress = targetVM.MAC
	vm.OSType = targetVM.OSType
	vm.CPUNumber = s.parseIntSafe(targetVM.CPUNumber)
	vm.CPUCore = s.parseIntSafe(targetVM.CPUCore)
	vm.CPUPer = s.parseIntSafe(targetVM.CPUPer)
	vm.Memory = s.parseIntSafe(targetVM.MemAll)
	vm.MemoryPer = s.parseIntSafe(targetVM.MemPer)
	vm.Disk = s.parseIntSafe(targetVM.DiscAll)
	vm.DiskPer = s.parseIntSafe(targetVM.DiscPer)

	return nil
}

// SyncAllVMs 同步所有虚拟机（调用VDI API）
func (s *vmServiceImpl) SyncAllVMs(ctx context.Context, serverID string) error {
	// 1. 验证服务器存在
	var server models.VDIServer
	if err := s.db.WithContext(ctx).Where("id = ?", serverID).First(&server).Error; err != nil {
		return fmt.Errorf("VDI server not found: %w", err)
	}

	// 2. 查询该服务器的所有虚拟机
	var vms []models.VDIVirtualMachine
	if err := s.db.WithContext(ctx).Where("vdi_server_id = ?", serverID).Find(&vms).Error; err != nil {
		return fmt.Errorf("failed to query VMs: %w", err)
	}

	// 3. 逐个同步虚拟机状态
	for _, vm := range vms {
		if err := s.SyncVMFromVDI(ctx, vm.ID); err != nil {
			// 记录错误但继续同步其他虚拟机
			applogger.Warnf("[VDI] SYNC Failed to sync VM %s: %v", vm.VMID, err)
		}
	}

	return nil
}

// SyncVMsFromVDIByServer 从指定VDI服务器同步所有虚拟机数据
func (s *vmServiceImpl) SyncVMsFromVDIByServer(ctx context.Context, server *models.VDIServer) error {
	// 为指定服务器创建VDI客户端
	client := NewVDIClientFromDB(s.db, server.ID)

	// 调用同步方法
	if err := s.syncVMsFromVDI(ctx, client); err != nil {
		return fmt.Errorf("failed to sync VMs from VDI server [%s]: %w", server.Name, err)
	}

	return nil
}

// toDTO 转换为DTO
func (s *vmServiceImpl) toDTO(vm *models.VDIVirtualMachine) *VDIVMDTO {
	return &VDIVMDTO{
		ID:            vm.ID,
		VMID:          vm.VMID,
		Name:          vm.Name,
		ResourceID:    vm.ResourceID,
		PowerState:    vm.PowerState,
		IPAddress:     vm.IPAddress,
		MACAddress:    vm.MACAddress,
		OSType:        vm.OSType,
		CPUNumber:     vm.CPUNumber,
		CPUCore:       vm.CPUCore,
		CPUPer:        vm.CPUPer,
		Memory:        vm.Memory,
		MemoryPer:     vm.MemoryPer,
		Disk:          vm.Disk,
		DiskPer:       vm.DiskPer,
		BoundUserID:    vm.BoundUserID,
		BoundUserName:  vm.BoundUserName,
		PolicyGroupID:  vm.PolicyGroupID,
		// 网络配置信息
		IPType:         vm.IPType,
		SubnetMask:     vm.SubnetMask,
		DefaultGateway: vm.DefaultGateway,
		NameServer:     vm.NameServer,
		AssignIP:       vm.AssignIP,
		VdiServerID:    vm.VdiServerID,
		LastSyncAt:     vm.LastSyncAt,
	}
}
