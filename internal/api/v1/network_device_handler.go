package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// NetworkDeviceListRequest 网络设备列表请求
type NetworkDeviceListRequest struct {
	DeviceName *string                `json:"deviceName,omitempty"`
	DeviceType *models.DeviceType     `json:"deviceType,omitempty"`
	Vendor     *models.DeviceVendor  `json:"vendor,omitempty"`
	IP         *string                `json:"ip,omitempty"`
	Status     *models.DeviceStatus  `json:"status,omitempty"`
	DeptID     *string                `json:"deptId,omitempty"`
	Current    int                    `json:"current" binding:"min=1"`
	PageSize   int                    `json:"pageSize" binding:"min=1,max=100"`
}

// NetworkDeviceCreateRequest 创建网络设备请求
type NetworkDeviceCreateRequest struct {
	DeviceName   string                `json:"deviceName" binding:"required,min=2,max=100"`
	DeviceType   models.DeviceType     `json:"deviceType" binding:"required"`
	Vendor       models.DeviceVendor   `json:"vendor" binding:"required"`
	Model        string                `json:"model,omitempty" binding:"max=100"`
	IPAddress    string                `json:"ipAddress" binding:"required,ip"`
	Port         int                   `json:"port" binding:"min=1,max=65535"`
	SNMPPort     int                   `json:"snmpPort" binding:"min=1,max=65535"`
	CredentialID *string               `json:"credentialId,omitempty"`
	DeptID       *string               `json:"deptId,omitempty"`
	Location     string                `json:"location,omitempty" binding:"max=200"`
	Status       models.DeviceStatus   `json:"status" binding:"min=0,max=2"`
	Description  string                `json:"description,omitempty"`
}

// NetworkDeviceUpdateRequest 更新网络设备请求
type NetworkDeviceUpdateRequest struct {
	ID           string                `json:"id,omitempty"`
	DeviceName   string                `json:"deviceName" binding:"required,min=2,max=100"`
	DeviceType   models.DeviceType     `json:"deviceType" binding:"required"`
	Vendor       models.DeviceVendor   `json:"vendor" binding:"required"`
	Model        string                `json:"model,omitempty" binding:"max=100"`
	IPAddress    string                `json:"ipAddress" binding:"required,ip"`
	Port         int                   `json:"port" binding:"min=1,max=65535"`
	SNMPPort     int                   `json:"snmpPort" binding:"min=1,max=65535"`
	CredentialID *string               `json:"credentialId,omitempty"`
	DeptID       *string               `json:"deptId,omitempty"`
	Location     string                `json:"location,omitempty" binding:"max=200"`
	Status       models.DeviceStatus   `json:"status" binding:"min=0,max=2"`
	Description  string                `json:"description,omitempty"`
}

// listNetworkDevices 查询网络设备列表
// @Summary 查询网络设备列表
// @Description 分页查询网络设备列表
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body NetworkDeviceListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /network/devices/list [post]
func listNetworkDevices(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawReq map[string]interface{}
		if err := c.ShouldBindJSON(&rawReq); err != nil {
			rawReq = make(map[string]interface{})
		}

		req := NetworkDeviceListRequest{
			Current:  getIntField(rawReq, "current", 1),
			PageSize: getIntField(rawReq, "pageSize", 10),
		}

		if val, ok := rawReq["deviceName"].(string); ok && val != "" {
			req.DeviceName = &val
		}
		if val, ok := rawReq["deviceType"].(string); ok && val != "" {
			dt := models.DeviceType(val)
			req.DeviceType = &dt
		}
		if val, ok := rawReq["vendor"].(string); ok && val != "" {
			vendor := models.DeviceVendor(val)
			req.Vendor = &vendor
		}
		if val, ok := rawReq["ip"].(string); ok && val != "" {
			req.IP = &val
		}
		if val, ok := rawReq["status"]; ok && val != nil {
			switch v := val.(type) {
			case float64:
				s := models.DeviceStatus(int(v))
				req.Status = &s
			case int:
				s := models.DeviceStatus(v)
				req.Status = &s
			}
		}
		if val, ok := rawReq["deptId"].(string); ok && val != "" {
			req.DeptID = &val
		}

		db := core.DB.GetDB().Model(&models.NetworkDevice{})

		if req.DeviceName != nil && *req.DeviceName != "" {
			db = db.Where("device_name LIKE ?", "%"+*req.DeviceName+"%")
		}
		if req.DeviceType != nil {
			db = db.Where("device_type = ?", *req.DeviceType)
		}
		if req.Vendor != nil {
			db = db.Where("vendor = ?", *req.Vendor)
		}
		if req.IP != nil && *req.IP != "" {
			db = db.Where("ip_address LIKE ?", "%"+*req.IP+"%")
		}
		if req.Status != nil {
			db = db.Where("status = ?", *req.Status)
		}
		if req.DeptID != nil && *req.DeptID != "" {
			// 获取该部门及其所有子部门的ID
			deptIds := []string{*req.DeptID}
			getChildDepts(core, *req.DeptID, &deptIds)
			db = db.Where("dept_id IN ?", deptIds)
		}

		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询设备总数失败")
			return
		}

		var devices []models.NetworkDevice
		offset := (req.Current - 1) * req.PageSize
		if err := db.Offset(offset).Limit(req.PageSize).Find(&devices).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询设备列表失败")
			return
		}

		// 加载部门名称和凭证名称
		loadDeviceAssociations(core, &devices)

		pageResp := response.PageResponse{
			List:     devices,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}

		response.Success(c, pageResp)
	}
}

// createNetworkDevice 创建网络设备
// @Summary 创建网络设备
// @Description 创建新网络设备
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body NetworkDeviceCreateRequest true "设备信息"
// @Success 200 {object} response.Response
// @Router /network/devices [post]
func createNetworkDevice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req NetworkDeviceCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 检查IP地址是否已存在
		var existDevice models.NetworkDevice
		if err := core.DB.GetDB().Where("ip_address = ?", req.IPAddress).First(&existDevice).Error; err == nil {
			response.Error(c, response.ErrBadRequest, "IP地址已存在")
			return
		}

		// 验证凭证ID是否存在
		if req.CredentialID != nil && *req.CredentialID != "" {
			var credential models.AuthCredential
			if err := core.DB.GetDB().Where("id = ?", *req.CredentialID).First(&credential).Error; err != nil {
				response.Error(c, response.ErrBadRequest, "授权凭证不存在")
				return
			}
		}

		// 验证部门ID并加载部门名称
		var deptName *string
		if req.DeptID != nil && *req.DeptID != "" {
			var dept models.Department
			if err := core.DB.GetDB().Where("id = ?", *req.DeptID).First(&dept).Error; err != nil {
				response.Error(c, response.ErrBadRequest, "部门不存在")
				return
			}
			deptName = &dept.DeptName
		}

		device := models.NetworkDevice{
			DeviceName:     req.DeviceName,
			DeviceType:     req.DeviceType,
			Vendor:         req.Vendor,
			Model:          req.Model,
			IPAddress:      req.IPAddress,
			Port:           req.Port,
			SNMPPort:       req.SNMPPort,
			CredentialID:   req.CredentialID,
			DeptID:         req.DeptID,
			DeptName:       deptName,
			Location:       req.Location,
			Status:         req.Status,
			Description:    req.Description,
			BaseModel:      models.BaseModel{CreatedBy: c.GetString("userId")},
		}

		if err := core.DB.GetDB().Create(&device).Error; err != nil {
			response.Error(c, response.ErrServerError, "创建设备失败")
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "网络设备", OperTypeCreate)

		response.Success(c, gin.H{"id": device.ID})
	}
}

// getNetworkDevice 获取设备详情
// @Summary 获取设备详情
// @Description 根据ID获取设备详情
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} response.Response{data=models.NetworkDevice}
// @Router /network/devices/:id [post]
func getNetworkDevice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var device models.NetworkDevice
		if err := core.DB.GetDB().Where("id = ?", id).First(&device).Error; err != nil {
			response.Error(c, response.ErrNotFound, "设备不存在")
			return
		}

		// 加载关联信息
		devices := []models.NetworkDevice{device}
		loadDeviceAssociations(core, &devices)

		response.Success(c, devices[0])
	}
}

// updateNetworkDevice 更新网络设备
// @Summary 更新网络设备
// @Description 更新设备信息
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Param request body NetworkDeviceUpdateRequest true "设备信息"
// @Success 200 {object} response.Response
// @Router /network/devices/:id/update [post]
func updateNetworkDevice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var req NetworkDeviceUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		var device models.NetworkDevice
		if err := core.DB.GetDB().Where("id = ?", id).First(&device).Error; err != nil {
			response.Error(c, response.ErrNotFound, "设备不存在")
			return
		}

		// 检查IP地址是否被其他设备使用
		var existDevice models.NetworkDevice
		if err := core.DB.GetDB().Where("ip_address = ? AND id != ?", req.IPAddress, id).First(&existDevice).Error; err == nil {
			response.Error(c, response.ErrBadRequest, "IP地址已被其他设备使用")
			return
		}

		// 验证部门ID
		var deptName *string
		if req.DeptID != nil && *req.DeptID != "" {
			var dept models.Department
			if err := core.DB.GetDB().Where("id = ?", *req.DeptID).First(&dept).Error; err != nil {
				response.Error(c, response.ErrBadRequest, "部门不存在")
				return
			}
			deptName = &dept.DeptName
		}

		updates := map[string]interface{}{
			"device_name": req.DeviceName,
			"device_type": req.DeviceType,
			"vendor":      req.Vendor,
			"model":       req.Model,
			"ip_address":  req.IPAddress,
			"port":        req.Port,
			"snmp_port":   req.SNMPPort,
			"location":    req.Location,
			"status":      req.Status,
			"description": req.Description,
			"updated_by":  c.GetString("userId"),
		}

		if req.CredentialID != nil {
			updates["credential_id"] = req.CredentialID
		}
		if req.DeptID != nil {
			updates["dept_id"] = req.DeptID
			updates["dept_name"] = deptName
		}

		if err := core.DB.GetDB().Model(&device).Updates(updates).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新设备失败")
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "网络设备", OperTypeUpdate)

		response.Success(c, nil)
	}
}

// deleteNetworkDevice 删除网络设备
// @Summary 删除网络设备
// @Description 删除指定设备
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} response.Response
// @Router /network/devices/:id/delete [post]
func deleteNetworkDevice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		var device models.NetworkDevice
		if err := core.DB.GetDB().Where("id = ?", id).First(&device).Error; err != nil {
			response.Error(c, response.ErrNotFound, "设备不存在")
			return
		}

		if err := core.DB.GetDB().Delete(&device).Error; err != nil {
			response.Error(c, response.ErrServerError, "删除设备失败")
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "网络设备", OperTypeDelete)

		response.Success(c, nil)
	}
}

// batchDeleteNetworkDevices 批量删除设备
// @Summary 批量删除设备
// @Description 批量删除多个设备
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body map[string][]string true "ID列表"
// @Success 200 {object} response.Response
// @Router /network/devices/batch-delete [post]
func batchDeleteNetworkDevices(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Ids []string `json:"ids" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		if err := core.DB.GetDB().Where("id IN ?", req.Ids).Delete(&models.NetworkDevice{}).Error; err != nil {
			response.Error(c, response.ErrServerError, "批量删除设备失败")
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "网络设备", OperTypeDelete)

		response.Success(c, nil)
	}
}

// loadDeviceAssociations 加载设备关联信息
func loadDeviceAssociations(core *core.Core, devices *[]models.NetworkDevice) {
	var deptIDs []string
	var credentialIDs []string

	for _, d := range *devices {
		if d.DeptID != nil && *d.DeptID != "" {
			deptIDs = append(deptIDs, *d.DeptID)
		}
		if d.CredentialID != nil && *d.CredentialID != "" {
			credentialIDs = append(credentialIDs, *d.CredentialID)
		}
	}

	// 加载部门信息
	deptMap := make(map[string]string)
	if len(deptIDs) > 0 {
		var depts []models.Department
		core.DB.GetDB().Where("id IN ?", deptIDs).Find(&depts)
		for _, dept := range depts {
			deptMap[dept.ID] = dept.DeptName
		}
	}

	// 加载凭证信息
	credentialMap := make(map[string]string)
	if len(credentialIDs) > 0 {
		var credentials []models.AuthCredential
		core.DB.GetDB().Where("id IN ?", credentialIDs).Find(&credentials)
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

// getChildDepts 递归获取所有子部门ID
func getChildDepts(core *core.Core, parentId string, deptIds *[]string) {
	var childDepts []struct {
		ID string `gorm:"column:id"`
	}

	core.DB.GetDB().Table("sys_dept").Where("parent_id = ?", parentId).Select("id").Scan(&childDepts)

	for _, dept := range childDepts {
		*deptIds = append(*deptIds, dept.ID)
		getChildDepts(core, dept.ID, deptIds)
	}
}

// DeviceDiscoverRequest 设备探测请求
type DeviceDiscoverRequest struct {
	IPAddress    string   `json:"ipAddress" binding:"required,ip"`
	CredentialID string   `json:"credentialId" binding:"required,uuid"`
	SNMPPort     int      `json:"snmpPort,omitempty"`
	Communities  []string `json:"communities,omitempty"`
}

// DeviceQuickCreateRequest 快速创建设备请求
type DeviceQuickCreateRequest struct {
	IPAddress    string   `json:"ipAddress" binding:"required,ip"`
	CredentialID string   `json:"credentialId" binding:"required,uuid"`
	SNMPPort     int      `json:"snmpPort,omitempty"`
	Communities  []string `json:"communities,omitempty"`
	DeptID       *string  `json:"deptId,omitempty"`
	Location     string   `json:"location,omitempty"`
	Description  string   `json:"description,omitempty"`
}

// discoverNetworkDevice 探测网络设备
// @Summary 探测网络设备
// @Description 通过 SNMP 探测获取设备信息
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body DeviceDiscoverRequest true "探测请求"
// @Success 200 {object} response.Response
// @Router /network/devices/discover [post]
func discoverNetworkDevice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DeviceDiscoverRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 构造探测请求
		probeReq := &services.DeviceProbeRequest{
			IPAddress:    req.IPAddress,
			CredentialID: req.CredentialID,
			SNMPPort:     req.SNMPPort,
			Communities:  req.Communities,
		}

		// 调用设备发现服务
		result, err := core.DeviceDiscoveryService.ProbeSingleDevice(c.Request.Context(), probeReq)
		if err != nil {
			response.Error(c, response.ErrServerError, "探测失败: "+err.Error())
			return
		}

		// 返回探测结果
		if !result.Success {
			response.Error(c, response.ErrBadRequest, result.Message)
			return
		}

		response.Success(c, result)
	}
}

// quickCreateNetworkDevice 快速创建网络设备
// @Summary 快速创建网络设备
// @Description 通过 SNMP 探测自动获取设备信息并创建设备
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body DeviceQuickCreateRequest true "快速创建请求"
// @Success 200 {object} response.Response
// @Router /network/devices/quick-create [post]
func quickCreateNetworkDevice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DeviceQuickCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 构造快速创建请求
		quickCreateReq := &services.QuickCreateRequest{
			IPAddress:    req.IPAddress,
			CredentialID: req.CredentialID,
			SNMPPort:     req.SNMPPort,
			Communities:  req.Communities,
			DeptID:       req.DeptID,
			Location:     req.Location,
			Description:  req.Description,
			CreatedBy:    c.GetString("userId"),
		}

		// 调用设备服务快速创建
		device, err := core.NetworkDeviceService.QuickCreateDevice(c.Request.Context(), quickCreateReq)
		if err != nil {
			response.Error(c, response.ErrBadRequest, err.Error())
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "网络设备", OperTypeCreate)

		response.Success(c, gin.H{"id": device.ID})
	}
}

// getEnrichmentStatus 获取设备信息采集任务状态
// @Summary 获取设备信息采集任务状态
// @Description 查询设备信息采集任务的执行状态
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} response.Response
// @Router /network/devices/:id/enrichment-status [get]
func getEnrichmentStatus(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "设备ID不能为空")
			return
		}

		// 获取采集任务状态
		task, err := core.DeviceInfoCollectionService.GetTaskStatus(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, "查询任务状态失败: "+err.Error())
			return
		}

		// 如果没有任务，返回空
		if task == nil {
			response.Success(c, gin.H{
				"hasTask": false,
			})
			return
		}

		response.Success(c, gin.H{
			"hasTask":              true,
			"status":               task.Status,
			"startedAt":            task.StartedAt,
			"completedAt":          task.CompletedAt,
			"errorMessage":         task.ErrorMessage,
			"enrichedModel":        task.EnrichedModel,
			"enrichedSerialNumber": task.EnrichedSerialNumber,
			"enrichedSoftwareVer":  task.EnrichedSoftwareVer,
			"enrichedUptime":       task.EnrichedUptime,
		})
	}
}
