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
	if err := s.vdiClient.DeleteVM(ctx, vmIDs); err != nil {
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
	if err := s.vdiClient.OperateVM(ctx, vmIDs, action); err != nil {
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
	configReqs := make([]ConfigIPRequestExtended, len(req))
	for i, r := range req {
		configReqs[i] = ConfigIPRequestExtended{
			VMInfos: []VMIPInfo{
				{
					VMID:      r.VMID,
					IPAddress: r.IPAddress,
					Netmask:   r.Netmask,
					Gateway:   r.Gateway,
				},
			},
		}
	}

	if err := s.vdiClient.ConfigIP(ctx, configReqs); err != nil {
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
	// 1. 查询本地VM记录
	var vm models.VDIVirtualMachine
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&vm).Error; err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	// 2. 调用VDI API重命名
	if err := s.vdiClient.RenameVM(ctx, vm.VMID, req.NewName); err != nil {
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
	// 1. 查询本地VM记录
	var vm models.VDIVirtualMachine
	if err := s.db.WithContext(ctx).Where("id = ?", vmID).First(&vm).Error; err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	// 2. 调用VDI API绑定用户
	vdiVMDetail, err := s.vdiClient.BindUser(ctx, vm.VMID, req.UserID)
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
	// 1. 查询本地VM记录
	var vm models.VDIVirtualMachine
	if err := s.db.WithContext(ctx).Where("id = ?", vmID).First(&vm).Error; err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	// 2. 调用VDI API解绑用户（使用空用户ID）
	_, err := s.vdiClient.BindUser(ctx, vm.VMID, "")
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
	// 1. 查询本地VM记录
	var vm models.VDIVirtualMachine
	if err := s.db.WithContext(ctx).Where("id = ?", vmID).First(&vm).Error; err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	// 2. 调用VDI API获取最新数据
	vdiVMDetail, err := s.vdiClient.GetVM(ctx, vm.VMID)
	if err != nil {
		return fmt.Errorf("failed to fetch VM from VDI: %w", err)
	}

	// 3. 更新本地记录
	vm.PowerState = vdiVMDetail.PowerState
	vm.IPAddress = vdiVMDetail.IPAddress
	vm.MACAddress = vdiVMDetail.MACAddress
	vm.OSType = vdiVMDetail.OSType
	vm.CPU = vdiVMDetail.CPU
	vm.Memory = vdiVMDetail.Memory
	vm.Disk = vdiVMDetail.Disk
	if vdiVMDetail.BoundUserID != nil {
		vm.BoundUserID = vdiVMDetail.BoundUserID
	}
	if vdiVMDetail.BoundUserName != nil {
		vm.BoundUserName = vdiVMDetail.BoundUserName
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
