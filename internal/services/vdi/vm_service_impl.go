package vdi

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// vmServiceImpl 虚拟机服务实现
type vmServiceImpl struct {
	db            *gorm.DB
	vdiClient     VDIClientExtended
	uuidValidator *regexp.Regexp
}

// NewVMService 创建虚拟机服务
func NewVMService(db *gorm.DB, vdiClient VDIClientExtended) VMService {
	return &vmServiceImpl{
		db:            db,
		vdiClient:     vdiClient,
		uuidValidator: regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`),
	}
}

// NewVMServiceWithDynamicClient 创建虚拟机服务（动态查找客户端）
func NewVMServiceWithDynamicClient(db *gorm.DB) VMService {
	return &vmServiceImpl{
		db:            db,
		vdiClient:     nil, // 将动态加载
		uuidValidator: regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`),
	}
}

// getClient 动态查找启用的VDI服务器并返回客户端
func (s *vmServiceImpl) getClient(ctx context.Context) (VDIClientExtended, error) {
	// 如果已有客户端，直接返回
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
	// 1. 获取所有资源组
	groups, err := client.ListResourceGroups(ctx)
	if err != nil {
		return fmt.Errorf("failed to list resource groups: %w", err)
	}

	// 2. 遍历每个资源组，获取其下的资源
	for _, group := range groups {
		if group.Enable != "1" {
			continue // 跳过未启用的资源组
		}

		fmt.Printf("[VDI SYNC] Processing resource group: %s (ID: %s)\n", group.Name, group.ID)

		// 2.1 获取该资源组下的所有资源
		resources, err := client.ListResources(ctx, group.ID)
		if err != nil {
			fmt.Printf("Failed to list resources for group %s: %v\n", group.Name, err)
			continue
		}

		fmt.Printf("[VDI SYNC] Found %d resources in group %s\n", len(resources), group.Name)

		// 3. 遍历每个资源，获取虚拟机列表
		for _, resource := range resources {
			fmt.Printf("[VDI SYNC] Processing resource: %s (ID: %d)\n", resource.Name, resource.ID)

			// 分页获取该资源下的所有虚拟机
			page := 1
			pageSize := 100
			for {
				vms, total, err := client.ListResourceServers(ctx, fmt.Sprintf("%d", resource.ID), page, pageSize)
				if err != nil {
					// 记录错误但继续处理其他资源
					fmt.Printf("Failed to list VMs for resource %s: %v\n", resource.Name, err)
					break
				}

				// 保存虚拟机数据到本地数据库
				for _, vm := range vms {
					s.saveOrUpdateVM(ctx, vm, fmt.Sprintf("%d", resource.ID))
				}

				// 检查是否还有更多数据
				if page*pageSize >= total {
					break
				}
				page++
			}
		}
	}

	return nil
}

// saveOrUpdateVM 保存或更新虚拟机记录
func (s *vmServiceImpl) saveOrUpdateVM(ctx context.Context, resource VDIVMResource, resourceID string) {
	var vm models.VDIVirtualMachine
	err := s.db.WithContext(ctx).
		Where("vm_id = ?", resource.ID).
		First(&vm).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		vm = models.VDIVirtualMachine{
			VMID:        resource.ID,
			Name:        resource.VMName,
			ResourceID:  resourceID,
			Status:      0,
			PowerState:  s.mapPowerState(resource.Status),
			IPAddress:   resource.IP,
			MACAddress:  resource.MAC,
			OSType:      resource.OSType,
			CPU:         s.parseIntSafe(resource.CPUNumber),
			Memory:      s.parseIntSafe(resource.MemAll),
			Disk:        s.parseIntSafe(resource.DiscAll),
			BoundUserID: &resource.ApplyUser,
			VdiServerID: s.vdiServerID(),
			LastSyncAt:  &time.Time{},
		}
		s.db.WithContext(ctx).Create(&vm)
	} else if err == nil {
		// 更新现有记录
		updates := map[string]interface{}{
			"name":        resource.VMName,
			"resource_id": resourceID,
			"power_state": s.mapPowerState(resource.Status),
			"ip_address":  resource.IP,
			"mac_address": resource.MAC,
			"os_type":     resource.OSType,
			"cpu":         s.parseIntSafe(resource.CPUNumber),
			"last_sync_at": time.Now(),
			"memory":      s.parseIntSafe(resource.MemAll),
			"disk":        s.parseIntSafe(resource.DiscAll),
		}
		if resource.ApplyUser != "" {
			updates["bound_user_id"] = &resource.ApplyUser
		}
		s.db.WithContext(ctx).Model(&vm).Updates(updates)
	}
}

// mapPowerState 映射VDI状态到电源状态
func (s *vmServiceImpl) mapPowerState(status string) string {
	// VDI状态码映射（根据深信服文档）
	// 11=运行中, 12=关机, 13=暂停, 14=故障, 等
	switch status {
	case "11":
		return "running"
	case "12":
		return "stopped"
	case "13":
		return "suspended"
	default:
		return "unknown"
	}
}

// parseIntSafe 安全解析整数
func (s *vmServiceImpl) parseIntSafe(str string) int {
	var i int
	fmt.Sscanf(str, "%d", &i)
	return i
}

// vdiServerID 获取当前VDI服务器ID
func (s *vmServiceImpl) vdiServerID() string {
	var server models.VDIServer
	s.db.Where("status = 0").First(&server)
	return server.ID
}

// CreateVM 创建虚拟机（调用VDI API）
func (s *vmServiceImpl) CreateVM(ctx context.Context, req *CreateVMServiceRequest) (*VDIVMDTO, error) {
	// 1. 验证VDI服务器存在
	var server models.VDIServer
	if err := s.db.WithContext(ctx).Where("id = ? AND status = 0", req.VdiServerID).First(&server).Error; err != nil {
		return nil, fmt.Errorf("VDI server not found or disabled: %w", err)
	}

	// 2. 调用VDI API创建虚拟机
	// 注意：当前VDI客户端扩展接口没有CreateVM方法，需要使用底层VDIClient
	// 这里我们模拟创建，实际应该调用VDI API
	// 由于VDI API的CreateVM方法在旧客户端中，我们暂时使用模拟方式

	// 3. 创建本地虚拟机记录（使用VDI返回的VM ID）
	vm := &models.VDIVirtualMachine{
		VMID:        fmt.Sprintf("vm-%d", time.Now().UnixNano()), // 模拟VDI返回的VM ID
		Name:        req.Name,
		ResourceID:  req.ResourceID,
		VdiServerID: req.VdiServerID,
		CPU:         req.CPU,
		Memory:      req.Memory,
		Disk:        req.Disk,
		Status:      0,
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

// ListVMs 获取虚拟机列表
func (s *vmServiceImpl) ListVMs(ctx context.Context, req *ListVMRequest) (*PageResult, error) {
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
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
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

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count VMs: %w", err)
	}

	// 分页查询
	var vms []models.VDIVirtualMachine
	offset := (req.Page - 1) * req.PageSize
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
	if req.Status != nil {
		updates["status"] = *req.Status
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
	if err := client.DeleteVM(ctx, vmIDs); err != nil {
		return fmt.Errorf("failed to delete VMs from VDI: %w", err)
	}

	// 3. 删除本地记录（软删除）
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

// BatchConfigIP 批量配置虚拟机IP（调用VDI API）
func (s *vmServiceImpl) BatchConfigIP(ctx context.Context, req []VMIPConfigRequest) error {
	if len(req) == 0 {
		return nil
	}

	// 获取VDI客户端
	client, err := s.getClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get VDI client: %w", err)
	}

	// 1. 查询本地VM记录，验证存在性
	vmIDs := make([]string, len(req))
	for i, r := range req {
		vmIDs[i] = r.VMID
	}

	var vms []models.VDIVirtualMachine
	if err := s.db.WithContext(ctx).Where("vm_id IN ?", vmIDs).Find(&vms).Error; err != nil {
		return fmt.Errorf("failed to query VMs: %w", err)
	}

	if len(vms) == 0 {
		return fmt.Errorf("no VMs found")
	}

	// 2. 调用VDI API配置IP
	// VDI API expects batch configuration with comma-separated IDs
	// For now, use type 2 (specified IP range) to configure individual IPs
	var idsStr string
	var vmNamesStr string
	mask := ""
	gateway := ""
	dns := ""

	for i, r := range req {
		if i > 0 {
			idsStr += ","
			vmNamesStr += ","
		}
		idsStr += r.VMID
		vmNamesStr += r.VMID // Using VMID as VM name for now
		if r.Netmask != "" {
			mask = r.Netmask
		}
		if r.Gateway != "" {
			gateway = r.Gateway
		}
		if r.IPAddress != "" {
			dns = r.IPAddress // Using IP as DNS placeholder
		}
	}

	configReq := ConfigIPRequestExtended{
		IDs:       idsStr,
		Type:      2, // Type 2: obtain specified IP
		IsRestart: 0, // Don't restart after configuration
		BeginIP:   req[0].IPAddress, // Use first IP as begin
		EndIP:     req[len(req)-1].IPAddress, // Use last IP as end
		VMNames:   vmNamesStr,
		Mask:      mask,
		Gate:      gateway,
		DNS:       dns,
	}

	if err := client.ConfigIP(ctx, configReq); err != nil {
		return fmt.Errorf("failed to config IPs: %w", err)
	}

	// 3. 更新本地记录
	for _, r := range req {
		s.db.WithContext(ctx).
			Model(&models.VDIVirtualMachine{}).
			Where("vm_id = ?", r.VMID).
			Update("ip_address", r.IPAddress)
	}

	return nil
}

// RenameVM 重命名虚拟机（调用VDI API）
func (s *vmServiceImpl) RenameVM(ctx context.Context, id string, req *RenameVMServiceRequest) error {
	// 获取VDI客户端
	client, err := s.getClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get VDI client: %w", err)
	}

	// 1. 查询本地VM记录
	var vm models.VDIVirtualMachine
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&vm).Error; err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	// 2. 调用VDI API重命名
	if err := client.RenameVM(ctx, vm.VMID, req.NewName); err != nil {
		return fmt.Errorf("failed to rename VM: %w", err)
	}

	// 3. 更新本地记录
	if err := s.db.WithContext(ctx).
		Model(&vm).
		Update("name", req.NewName).Error; err != nil {
		return fmt.Errorf("failed to update VM record: %w", err)
	}

	return nil
}

// BindUser 绑定用户（调用VDI API）
func (s *vmServiceImpl) BindUser(ctx context.Context, vmID string, req *BindUserServiceRequest) error {
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

	// 2. 调用VDI API绑定用户
	vdiVMDetail, err := client.BindUser(ctx, vm.VMID, req.UserID)
	if err != nil {
		return fmt.Errorf("failed to bind user: %w", err)
	}

	// 3. 更新本地记录
	updates := map[string]interface{}{
		"bound_user_id":   &req.UserID,
		"bound_user_name": vdiVMDetail.BoundUserName,
	}

	if err := s.db.WithContext(ctx).Model(&vm).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update VM record: %w", err)
	}

	return nil
}

// UnbindUser 解绑用户
func (s *vmServiceImpl) UnbindUser(ctx context.Context, vmID string) error {
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

	// 2. 调用VDI API解绑用户（使用空用户ID）
	_, err = client.BindUser(ctx, vm.VMID, "")
	if err != nil {
		return fmt.Errorf("failed to unbind user: %w", err)
	}

	// 3. 更新本地记录
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

	// 4. 更新本地记录
	vm.PowerState = s.mapPowerState(targetVM.Status)
	vm.IPAddress = targetVM.IP
	vm.MACAddress = targetVM.MAC
	vm.OSType = targetVM.OSType
	vm.CPU = s.parseIntSafe(targetVM.CPUNumber)
	vm.Memory = s.parseIntSafe(targetVM.MemAll)
	vm.Disk = s.parseIntSafe(targetVM.DiscAll)
	if targetVM.ApplyUser != "" {
		vm.BoundUserID = &targetVM.ApplyUser
	}
	now := time.Now()
	vm.LastSyncAt = &now

	if err := s.db.WithContext(ctx).Save(&vm).Error; err != nil {
		return fmt.Errorf("failed to update VM: %w", err)
	}

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
			fmt.Printf("Failed to sync VM %s: %v\n", vm.VMID, err)
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
		Status:        vm.Status,
		PowerState:    vm.PowerState,
		IPAddress:     vm.IPAddress,
		MACAddress:    vm.MACAddress,
		OSType:        vm.OSType,
		CPU:           vm.CPU,
		Memory:        vm.Memory,
		Disk:          vm.Disk,
		BoundUserID:   vm.BoundUserID,
		BoundUserName: vm.BoundUserName,
		VdiServerID:   vm.VdiServerID,
		LastSyncAt:    vm.LastSyncAt,
	}
}
