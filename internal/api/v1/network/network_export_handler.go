package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/services/lldp"
	networkServices "github.com/xingran-next/xingran-go-backend/internal/services/network"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/services/topology"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"github.com/xuri/excelize/v2"
)

// ExportRequest 通用导出请求
type ExportRequest struct {
	ExportMode string                 `json:"exportMode"` // "filtered" | "currentPage" | "all"
	Current    int                    `json:"current"`
	PageSize   int                    `json:"pageSize"`
	Filters    map[string]interface{} `json:"filters"` // 页面当前筛选条件
}

const (
	ExportModeFiltered    = "filtered"
	ExportModeCurrentPage = "currentPage"
	ExportModeAll         = "all"

	maxExportRows = 100000 // 最大导出行数
)

// NetworkExportHandler 网络设备导出处理器
type NetworkExportHandler struct {
	core *core.Core
}

// NewNetworkExportHandler 创建导出处理器
func NewNetworkExportHandler(core *core.Core) *NetworkExportHandler {
	return &NetworkExportHandler{core: core}
}

// ExportDevices 导出网络设备
func (h *NetworkExportHandler) ExportDevices(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}

	db := h.core.DB.GetDB()
	var cacheProvider systemServices.CacheProvider
	if h.core.DataCacheService != nil {
		cacheProvider = systemServices.NewCacheProvider(h.core.DataCacheService)
	} else {
		cacheProvider = &systemServices.NoOpCacheProvider{}
	}
	deviceService := networkServices.NewServiceWithCache(db, h.core.DeviceDiscoveryService, h.core.DeviceInfoCollectionService, cacheProvider, h.core.CacheConfigService)

	current, pageSize := h.getPaginationParams(&req)
	deviceReq := h.buildDeviceListRequest(req.Filters, current, pageSize)

	devices, _, err := deviceService.List(c.Request.Context(), deviceReq)
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	file := excelize.NewFile()
	sheet := "网络设备"
	file.SetSheetName("Sheet1", sheet)

	headers := []string{"设备名称", "设备类型", "厂商", "型号", "IP地址", "SSH端口", "SNMP端口", "授权凭证", "部门", "位置", "状态", "序列号", "软件版本", "运行时间", "最后连接", "备注", "创建时间"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheet, cell, header)
	}

	statusMap := map[int]string{0: "在线", 1: "离线", 2: "未知"}
	deviceTypeMap := map[string]string{"router": "路由器", "switch": "交换机", "firewall": "防火墙", "ap": "AP", "loadbalancer": "负载均衡"}
	vendorMap := map[string]string{"huawei": "华为", "h3c": "H3C", "ruijie": "锐捷", "maipu": "迈普"}

	for i, device := range devices {
		row := i + 2
		status := statusMap[int(device.Status)]
		if status == "" {
			status = fmt.Sprintf("%d", device.Status)
		}
		dt := deviceTypeMap[string(device.DeviceType)]
		if dt == "" {
			dt = string(device.DeviceType)
		}
		v := vendorMap[string(device.Vendor)]
		if v == "" {
			v = string(device.Vendor)
		}

		writeRow(file, sheet, row, []interface{}{
			device.DeviceName, dt, v, device.Model, device.IPAddress,
			device.Port, device.SNMPPort, device.CredentialName, device.DeptName,
			device.Location, status, device.SerialNumber, device.SoftwareVersion,
			device.Uptime, formatTimePtr((*time.Time)(device.LastSeenAt)), device.Description,
			formatTime(time.Time(device.CreatedAt)),
		})
	}

	if paramBytes, err := json.Marshal(req); err == nil {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备", operlog.OperTypeExport,
			operlog.WithOperParam(operlog.FilterSensitiveParams(string(paramBytes))))
	} else {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备", operlog.OperTypeExport)
	}
	h.setExportHeader(c, file, "网络设备")
}

// ExportCredentials 导出授权凭证
func (h *NetworkExportHandler) ExportCredentials(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}

	db := h.core.DB.GetDB()
	svc := services.NewAuthCredentialService(db, h.core.SM4Cipher)
	current, pageSize := h.getPaginationParams(&req)
	credReq := h.buildCredentialListRequest(req.Filters, current, pageSize)

	creds, _, err := svc.List(c.Request.Context(), credReq)
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	file := excelize.NewFile()
	sheet := "授权凭证"
	file.SetSheetName("Sheet1", sheet)

	headers := []string{"凭证名称", "协议类型", "用户名", "SNMP版本", "是否默认", "描述", "创建时间"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheet, cell, header)
	}

	for i, cred := range creds {
		row := i + 2
		isDefault := "否"
		if cred.IsDefault {
			isDefault = "是"
		}
		writeRow(file, sheet, row, []interface{}{
			cred.CredentialName, string(cred.ProtocolType), cred.Username,
			string(cred.SNMPVersion), isDefault, cred.Description,
			formatTime(time.Time(cred.CreatedAt)),
		})
	}

	if paramBytes, err := json.Marshal(req); err == nil {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "授权凭证", operlog.OperTypeExport,
			operlog.WithOperParam(operlog.FilterSensitiveParams(string(paramBytes))))
	} else {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "授权凭证", operlog.OperTypeExport)
	}
	h.setExportHeader(c, file, "授权凭证")
}

// ExportTemplates 导出配置模板
func (h *NetworkExportHandler) ExportTemplates(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}

	db := h.core.DB.GetDB()
	svc := services.NewTemplateService(db)
	current, pageSize := h.getPaginationParams(&req)
	tmplReq := h.buildTemplateListRequest(req.Filters, current, pageSize)

	templates, _, err := svc.List(c.Request.Context(), tmplReq)
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	file := excelize.NewFile()
	sheet := "配置模板"
	file.SetSheetName("Sheet1", sheet)

	headers := []string{"模板名称", "模板编码", "模板类型", "厂商", "设备类型", "描述", "是否系统模板", "创建时间"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheet, cell, header)
	}

	templateTypeMap := map[string]string{"init": "初始化", "config": "配置", "backup": "备份"}
	vendorMap := map[string]string{"huawei": "华为", "h3c": "H3C", "ruijie": "锐捷", "maipu": "迈普"}
	deviceTypeMap := map[string]string{"router": "路由器", "switch": "交换机", "firewall": "防火墙", "ap": "AP", "loadbalancer": "负载均衡"}

	for i, tmpl := range templates {
		row := i + 2
		tt := templateTypeMap[string(tmpl.TemplateType)]
		if tt == "" {
			tt = string(tmpl.TemplateType)
		}
		v := vendorMap[string(tmpl.Vendor)]
		if v == "" {
			v = string(tmpl.Vendor)
		}
		dt := deviceTypeMap[string(tmpl.DeviceType)]
		if dt == "" {
			dt = string(tmpl.DeviceType)
		}
		isSys := "否"
		if tmpl.IsSystem {
			isSys = "是"
		}
		writeRow(file, sheet, row, []interface{}{
			tmpl.TemplateName, tmpl.TemplateCode, tt, v, dt,
			tmpl.Description, isSys, formatTime(time.Time(tmpl.CreatedAt)),
		})
	}

	if paramBytes, err := json.Marshal(req); err == nil {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "配置模板", operlog.OperTypeExport,
			operlog.WithOperParam(operlog.FilterSensitiveParams(string(paramBytes))))
	} else {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "配置模板", operlog.OperTypeExport)
	}
	h.setExportHeader(c, file, "配置模板")
}

// ExportCommands 导出命令分发记录
func (h *NetworkExportHandler) ExportCommands(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}

	db := h.core.DB.GetDB()
	svc := services.NewCommandDispatchService(db, h.core.DeviceExecutor)
	current, pageSize := h.getPaginationParams(&req)

	records, _, err := svc.GetExecutionList(c.Request.Context(), current, pageSize, "", nil)
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	file := excelize.NewFile()
	sheet := "命令分发"
	file.SetSheetName("Sheet1", sheet)

	headers := []string{"执行名称", "状态", "设备总数", "成功数", "失败数", "执行策略", "并发数", "超时(秒)", "开始时间", "完成时间", "错误信息", "创建人", "创建时间"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheet, cell, header)
	}

	statusMap := map[int]string{0: "待执行", 1: "执行中", 2: "成功", 3: "失败", 4: "已取消"}

	for i, r := range records {
		row := i + 2
		status := statusMap[int(r.Status)]
		if status == "" {
			status = fmt.Sprintf("%d", r.Status)
		}
		writeRow(file, sheet, row, []interface{}{
			r.ExecutionName, status, r.TotalDevices, r.SuccessCount, r.FailureCount,
			string(r.ExecutionStrategy), r.Concurrency, r.Timeout,
			formatTimePtr((*time.Time)(r.StartedAt)), formatTimePtr((*time.Time)(r.CompletedAt)),
			r.ErrorMessage, r.CreatedBy, formatTime(time.Time(r.CreatedAt)),
		})
	}

	if paramBytes, err := json.Marshal(req); err == nil {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "命令分发", operlog.OperTypeExport,
			operlog.WithOperParam(operlog.FilterSensitiveParams(string(paramBytes))))
	} else {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "命令分发", operlog.OperTypeExport)
	}
	h.setExportHeader(c, file, "命令分发")
}

// ExportExecutions 导出配置执行记录
func (h *NetworkExportHandler) ExportExecutions(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}

	db := h.core.DB.GetDB()
	svc := services.NewConfigExecutionService(db, h.core.DeviceExecutor)
	current, pageSize := h.getPaginationParams(&req)

	records, _, err := svc.GetExecutionList(c.Request.Context(), current, pageSize, "", nil)
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	file := excelize.NewFile()
	sheet := "配置执行"
	file.SetSheetName("Sheet1", sheet)

	headers := []string{"执行名称", "状态", "设备总数", "成功数", "失败数", "执行策略", "并发数", "超时(秒)", "开始时间", "完成时间", "错误信息", "创建人", "创建时间"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheet, cell, header)
	}

	statusMap := map[int]string{0: "待执行", 1: "执行中", 2: "成功", 3: "失败", 4: "已取消"}

	for i, r := range records {
		row := i + 2
		status := statusMap[int(r.Status)]
		if status == "" {
			status = fmt.Sprintf("%d", r.Status)
		}
		writeRow(file, sheet, row, []interface{}{
			r.ExecutionName, status, r.TotalDevices, r.SuccessCount, r.FailureCount,
			string(r.ExecutionStrategy), r.Concurrency, r.Timeout,
			formatTimePtr((*time.Time)(r.StartedAt)), formatTimePtr((*time.Time)(r.CompletedAt)),
			r.ErrorMessage, r.CreatedBy, formatTime(time.Time(r.CreatedAt)),
		})
	}

	if paramBytes, err := json.Marshal(req); err == nil {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "配置执行", operlog.OperTypeExport,
			operlog.WithOperParam(operlog.FilterSensitiveParams(string(paramBytes))))
	} else {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "配置执行", operlog.OperTypeExport)
	}
	h.setExportHeader(c, file, "配置执行")
}

// ExportBackups 导出配置备份记录
func (h *NetworkExportHandler) ExportBackups(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}

	db := h.core.DB.GetDB()
	svc := services.NewConfigBackupService(db, h.core.DeviceExecutor)
	current, pageSize := h.getPaginationParams(&req)

	deviceID := ""
	if req.Filters != nil {
		if v, ok := req.Filters["deviceId"].(string); ok {
			deviceID = v
		}
	}

	records, _, err := svc.GetBackupList(c.Request.Context(), current, pageSize, deviceID, "", nil)
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	file := excelize.NewFile()
	sheet := "配置备份"
	file.SetSheetName("Sheet1", sheet)

	headers := []string{"设备名称", "IP地址", "备份类型", "存储方式", "版本号", "配置大小(B)", "是否压缩", "变更原因", "创建人", "创建时间"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheet, cell, header)
	}

	backupTypeMap := map[string]string{"auto": "自动", "manual": "手动"}
	storageTypeMap := map[string]string{"database": "数据库", "file": "文件"}

	for i, r := range records {
		row := i + 2
		bt := backupTypeMap[string(r.BackupType)]
		if bt == "" {
			bt = string(r.BackupType)
		}
		st := storageTypeMap[string(r.StorageType)]
		if st == "" {
			st = string(r.StorageType)
		}
		compressed := "否"
		if r.Compressed {
			compressed = "是"
		}
		writeRow(file, sheet, row, []interface{}{
			r.DeviceName, r.IPAddress, bt, st, r.Version,
			r.BackupSize, compressed, r.ChangeReason,
			r.CreatedBy, formatTime(time.Time(r.CreatedAt)),
		})
	}

	if paramBytes, err := json.Marshal(req); err == nil {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "配置备份", operlog.OperTypeExport,
			operlog.WithOperParam(operlog.FilterSensitiveParams(string(paramBytes))))
	} else {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "配置备份", operlog.OperTypeExport)
	}
	h.setExportHeader(c, file, "配置备份")
}

// ExportDiscoveries 导出设备发现记录
func (h *NetworkExportHandler) ExportDiscoveries(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}

	svc := h.core.DeviceDiscoveryService
	current, pageSize := h.getPaginationParams(&req)

	records, _, err := svc.GetDiscoveryList(c.Request.Context(), current, pageSize, "", nil)
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	file := excelize.NewFile()
	sheet := "设备发现"
	file.SetSheetName("Sheet1", sheet)

	headers := []string{"任务名称", "发现类型", "状态", "IP总数", "已发现数", "自动导入", "SNMP团体名", "SNMP端口", "开始时间", "完成时间", "错误信息", "创建人", "创建时间"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheet, cell, header)
	}

	statusMap := map[int]string{0: "待执行", 1: "执行中", 2: "成功", 3: "失败", 4: "已取消"}
	discoveryTypeMap := map[string]string{"snmp": "SNMP发现", "scan": "扫描发现"}

	for i, r := range records {
		row := i + 2
		status := statusMap[int(r.Status)]
		if status == "" {
			status = fmt.Sprintf("%d", r.Status)
		}
		dt := discoveryTypeMap[string(r.DiscoveryType)]
		if dt == "" {
			dt = string(r.DiscoveryType)
		}
		autoImport := "否"
		if r.AutoImport {
			autoImport = "是"
		}
		writeRow(file, sheet, row, []interface{}{
			r.TaskName, dt, status, r.TotalIPs, r.DiscoveredCount,
			autoImport, r.SNMPCommunity, r.SNMPPort,
			formatTimePtr(r.StartedAt), formatTimePtr(r.CompletedAt),
			r.ErrorMessage, r.CreatedBy, formatTime(r.CreatedAt),
		})
	}

	if paramBytes, err := json.Marshal(req); err == nil {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "设备发现", operlog.OperTypeExport,
			operlog.WithOperParam(operlog.FilterSensitiveParams(string(paramBytes))))
	} else {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "设备发现", operlog.OperTypeExport)
	}
	h.setExportHeader(c, file, "设备发现")
}

// ExportMACAddresses 导出MAC地址记录
func (h *NetworkExportHandler) ExportMACAddresses(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}

	lldpSvc := lldp.NewLLDPService(h.core.DeviceExecutor)
	filterRuleSvc := topology.NewFilterRuleService(h.core.DB.GetDB())
	svc := services.NewMACCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor, lldpSvc, filterRuleSvc)
	current, pageSize := h.getPaginationParams(&req)

	var deviceID, deptID, macAddress, interfaceName string
	if req.Filters != nil {
		if v, ok := req.Filters["deviceId"].(string); ok {
			deviceID = v
		}
		if v, ok := req.Filters["deptId"].(string); ok {
			deptID = v
		}
		if v, ok := req.Filters["macAddress"].(string); ok {
			macAddress = v
		}
		if v, ok := req.Filters["interfaceName"].(string); ok {
			interfaceName = v
		}
	}

	records, _, err := svc.GetMACAddressList(c.Request.Context(), current, pageSize, deviceID, deptID, macAddress, interfaceName, "", nil)
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	file := excelize.NewFile()
	sheet := "MAC地址"
	file.SetSheetName("Sheet1", sheet)

	headers := []string{"MAC地址", "接口名称", "VLAN ID", "MAC类型", "采集时间"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheet, cell, header)
	}

	for i, r := range records {
		row := i + 2
		writeRow(file, sheet, row, []interface{}{
			r.MACAddress, r.InterfaceName, r.VLANID,
			string(r.MACType), formatTime(r.CollectedAt),
		})
	}

	if paramBytes, err := json.Marshal(req); err == nil {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "MAC地址", operlog.OperTypeExport,
			operlog.WithOperParam(operlog.FilterSensitiveParams(string(paramBytes))))
	} else {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "MAC地址", operlog.OperTypeExport)
	}
	h.setExportHeader(c, file, "MAC地址")
}

// PortStatusWithDeviceName 端口状态带设备名称
type PortStatusWithDeviceName struct {
	models.DevicePortStatus
	DeviceName string `json:"deviceName"`
}

// ExportPorts 导出端口状态记录
func (h *NetworkExportHandler) ExportPorts(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}

	db := h.core.DB.GetDB()
	current, pageSize := h.getPaginationParams(&req)

	// Build query with join to get device name
	query := db.Table("sys_device_port_status as p").
		Select("p.*, d.device_name").
		Joins("LEFT JOIN sys_network_device as d ON p.device_id = d.id")

	// Apply filters
	if req.Filters != nil {
		if v, ok := req.Filters["deviceId"].(string); ok && v != "" {
			query = query.Where("p.device_id = ?", v)
		}
		if v, ok := req.Filters["interfaceName"].(string); ok && v != "" {
			query = query.Where("p.interface_name LIKE ?", "%"+v+"%")
		}
		if v, ok := req.Filters["adminStatus"].(string); ok && v != "" {
			query = query.Where("p.admin_status = ?", v)
		}
		if v, ok := req.Filters["operStatus"].(string); ok && v != "" {
			query = query.Where("p.oper_status = ?", v)
		}
	}

	var records []PortStatusWithDeviceName
	offset := (current - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("p.collected_at DESC").Find(&records).Error; err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	file := excelize.NewFile()
	sheet := "端口状态"
	file.SetSheetName("Sheet1", sheet)

	headers := []string{"设备名称", "接口名称", "管理状态", "操作状态", "描述", "VLAN", "双工模式", "速率", "端口类型", "802.1X", "802.1X状态", "端口安全", "安全模式", "最大MAC数", "当前MAC数", "采集时间"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheet, cell, header)
	}

	for i, r := range records {
		row := i + 2
		dot1x := "否"
		if r.Dot1xEnabled {
			dot1x = "是"
		}
		sec := "否"
		if r.PortSecurityEnabled {
			sec = "是"
		}

		// Handle VLAN pointer - dereference safely
		var vlanValue interface{}
		if r.VLAN != nil {
			vlanValue = *r.VLAN
		} else {
			vlanValue = ""
		}

		writeRow(file, sheet, row, []interface{}{
			r.DeviceName, r.InterfaceName, r.AdminStatus, r.OperStatus, r.Description,
			vlanValue, r.Duplex, r.Speed, r.PortType,
			dot1x, r.Dot1xPortStatus, sec, r.PortSecurityMode,
			r.MaxMACCount, r.CurrentMACCount, formatTime(r.CollectedAt),
		})
	}

	if paramBytes, err := json.Marshal(req); err == nil {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "端口状态", operlog.OperTypeExport,
			operlog.WithOperParam(operlog.FilterSensitiveParams(string(paramBytes))))
	} else {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "端口状态", operlog.OperTypeExport)
	}
	h.setExportHeader(c, file, "端口状态")
}

// ==================== 辅助方法 ====================

// getPaginationParams 根据导出模式返回分页参数
func (h *NetworkExportHandler) getPaginationParams(req *ExportRequest) (int, int) {
	switch req.ExportMode {
	case ExportModeFiltered:
		return 1, maxExportRows
	case ExportModeCurrentPage:
		current := req.Current
		if current < 1 {
			current = 1
		}
		pageSize := req.PageSize
		if pageSize < 1 {
			pageSize = 10
		}
		return current, pageSize
	case ExportModeAll:
		return 1, maxExportRows
	default:
		return 1, maxExportRows
	}
}

// buildDeviceListRequest 构建设备列表请求
func (h *NetworkExportHandler) buildDeviceListRequest(filters map[string]interface{}, current, pageSize int) *services.ListDeviceRequest {
	req := &services.ListDeviceRequest{
		BaseListRequest: base.BaseListRequest{
			Current:  current,
			PageSize: pageSize,
		},
	}
	if filters == nil {
		return req
	}
	if v, ok := filters["deviceName"].(string); ok && v != "" {
		req.DeviceName = &v
	}
	if v, ok := filters["ipAddress"].(string); ok && v != "" {
		req.IP = &v
	}
	if v, ok := filters["deviceType"].(string); ok && v != "" {
		dt := models.DeviceType(v)
		req.DeviceType = &dt
	}
	if v, ok := filters["vendor"].(string); ok && v != "" {
		vd := models.DeviceVendor(v)
		req.Vendor = &vd
	}
	if v, ok := filters["status"].(float64); ok {
		s := models.DeviceStatus(int(v))
		req.Status = &s
	}
	if v, ok := filters["deptId"].(string); ok && v != "" {
		req.DeptID = &v
	}
	return req
}

// buildCredentialListRequest 构建凭证列表请求
func (h *NetworkExportHandler) buildCredentialListRequest(filters map[string]interface{}, current, pageSize int) *services.ListCredentialRequest {
	req := &services.ListCredentialRequest{}
	req.Current = current
	req.PageSize = pageSize
	if filters == nil {
		return req
	}
	if v, ok := filters["credentialName"].(string); ok && v != "" {
		req.CredentialName = &v
	}
	if v, ok := filters["protocolType"].(string); ok && v != "" {
		pt := models.ProtocolType(v)
		req.ProtocolType = &pt
	}
	return req
}

// buildTemplateListRequest 构建模板列表请求
func (h *NetworkExportHandler) buildTemplateListRequest(filters map[string]interface{}, current, pageSize int) *services.ListTemplateRequest {
	req := &services.ListTemplateRequest{
		BaseListRequest: base.BaseListRequest{
			Current:  current,
			PageSize: pageSize,
		},
	}
	if filters == nil {
		return req
	}
	if v, ok := filters["templateName"].(string); ok && v != "" {
		req.TemplateName = &v
	}
	if v, ok := filters["templateType"].(string); ok && v != "" {
		tt := models.TemplateType(v)
		req.TemplateType = &tt
	}
	if v, ok := filters["vendor"].(string); ok && v != "" {
		vd := models.DeviceVendor(v)
		req.Vendor = &vd
	}
	if v, ok := filters["deviceType"].(string); ok && v != "" {
		dt := models.DeviceType(v)
		req.DeviceType = &dt
	}
	if v, ok := filters["isSystem"].(bool); ok {
		req.IsSystem = &v
	}
	return req
}

// setExportHeader 设置Excel响应头并写入文件
func (h *NetworkExportHandler) setExportHeader(c *gin.Context, file *excelize.File, entityName string) {
	var buf bytes.Buffer
	if _, err := file.WriteTo(&buf); err != nil {
		response.Error(c, response.ErrServerError, "生成文件失败")
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_export_%s.xlsx", entityName, timestamp)
	// URL-encode the filename to properly handle Chinese characters
	encodedFilename := url.QueryEscape(filename)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename="+encodedFilename+"; filename*=utf-8''"+encodedFilename)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// writeRow 写入一行数据到Excel
func writeRow(file *excelize.File, sheet string, row int, values []interface{}) {
	for col, val := range values {
		cell, _ := excelize.CoordinatesToCellName(col+1, row)
		file.SetCellValue(sheet, cell, val)
	}
}

// formatTime 格式化时间
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// formatTimePtr 格式化可空时间指针
func formatTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}
