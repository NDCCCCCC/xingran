package network

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/services/lldp"
	networkServices "github.com/xingran-next/xingran-go-backend/internal/services/network"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/services/topology"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"github.com/xuri/excelize/v2"
)

// BatchExportRequest 批量导出请求
type BatchExportRequest struct {
	EntityTypes []string                 `json:"entityTypes" binding:"required,min=1,max=9"`
	Filters     map[string]interface{}  `json:"filters"`
}

// entityExportConfig 实体导出配置
type entityExportConfig struct {
	key      string
	name     string
	filename string
}

// BatchExport 批量导出多个实体类型为 ZIP 文件
func (h *NetworkExportHandler) BatchExport(c *gin.Context) {
	var req BatchExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}

	// 定义所有支持的实体类型
	supportedEntities := []entityExportConfig{
		{"devices", "网络设备", "网络设备.xlsx"},
		{"credentials", "授权凭证", "授权凭证.xlsx"},
		{"templates", "配置模板", "配置模板.xlsx"},
		{"commands", "命令分发", "命令分发.xlsx"},
		{"executions", "配置执行", "配置执行.xlsx"},
		{"backups", "配置备份", "配置备份.xlsx"},
		{"discoveries", "设备发现", "设备发现.xlsx"},
		{"mac", "MAC地址", "MAC地址.xlsx"},
		{"ports", "端口状态", "端口状态.xlsx"},
	}

	// 验证请求的实体类型
	entityMap := make(map[string]entityExportConfig)
	for _, e := range supportedEntities {
		entityMap[e.key] = e
	}

	// 检查是否有无效的实体类型
	for _, et := range req.EntityTypes {
		if _, ok := entityMap[et]; !ok {
			response.Error(c, response.ErrBadRequest, fmt.Sprintf("不支持的实体类型: %s", et))
			return
		}
	}

	// 创建 ZIP 缓冲区
	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)

	// 为每个实体类型生成 Excel 文件并添加到 ZIP
	for _, entityType := range req.EntityTypes {
		config := entityMap[entityType]

		// 创建导出请求
		exportReq := ExportRequest{
			ExportMode: ExportModeFiltered,
			Current:    1,
			PageSize:   maxExportRows,
			Filters:    req.Filters,
		}

		// 生成 Excel 数据
		data, err := h.generateEntityExcel(entityType, exportReq)
		if err != nil {
			response.Error(c, response.ErrServerError, fmt.Sprintf("生成 %s Excel 失败: %v", config.name, err))
			zipWriter.Close()
			return
		}

		// 创建 ZIP 文件条目
		writer, err := zipWriter.Create(config.filename)
		if err != nil {
			response.Error(c, response.ErrServerError, fmt.Sprintf("创建 ZIP 条目失败: %v", err))
			zipWriter.Close()
			return
		}

		// 写入数据
		if _, err := writer.Write(data); err != nil {
			response.Error(c, response.ErrServerError, fmt.Sprintf("写入 ZIP 数据失败: %v", err))
			zipWriter.Close()
			return
		}
	}

	// 关闭 ZIP writer
	if err := zipWriter.Close(); err != nil {
		response.Error(c, response.ErrServerError, fmt.Sprintf("关闭 ZIP 失败: %v", err))
		return
	}

	// 检查 ZIP 大小（限制 100MB）
	const maxZipSize = 100 * 1024 * 1024
	if zipBuf.Len() > maxZipSize {
		response.Error(c, response.ErrBadRequest, "数据量过大，请缩小筛选范围后重试")
		return
	}

	// 设置响应头
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("网络管理_批量导出_%s.zip", timestamp)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络管理批量导出", operlog.OperTypeExport,
		operlog.WithOperParam("entities="+fmt.Sprintf("%v", req.EntityTypes)))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(http.StatusOK, "application/zip", zipBuf.Bytes())
}

// generateEntityExcel 生成实体类型的 Excel 数据
func (h *NetworkExportHandler) generateEntityExcel(entityType string, req ExportRequest) ([]byte, error) {
	db := h.core.DB.GetDB()
	ctx := context.Background()
	current, pageSize := h.getPaginationParams(&req)

	file := excelize.NewFile()
	var sheet string
	var headers []string
	var data [][]interface{}

	switch entityType {
	case "devices":
		sheet = "网络设备"
		headers = []string{"设备名称", "设备类型", "厂商", "型号", "IP地址", "SSH端口", "SNMP端口", "授权凭证", "部门", "位置", "状态", "序列号", "软件版本", "运行时间", "最后连接", "备注", "创建时间"}

		var cacheProvider systemServices.CacheProvider
		if h.core.DataCacheService != nil {
			cacheProvider = systemServices.NewCacheProvider(h.core.DataCacheService)
		} else {
			cacheProvider = &systemServices.NoOpCacheProvider{}
		}
		deviceService := networkServices.NewServiceWithCache(db, h.core.DeviceDiscoveryService, h.core.DeviceInfoCollectionService, cacheProvider, h.core.CacheConfigService)

		deviceReq := h.buildDeviceListRequest(req.Filters, current, pageSize)
		devices, _, err := deviceService.List(ctx, deviceReq)
		if err != nil {
			return nil, err
		}

		statusMap := map[int]string{0: "在线", 1: "离线", 2: "未知"}
		deviceTypeMap := map[string]string{"router": "路由器", "switch": "交换机", "firewall": "防火墙", "ap": "AP", "loadbalancer": "负载均衡"}
		vendorMap := map[string]string{"huawei": "华为", "h3c": "H3C", "ruijie": "锐捷", "maipu": "迈普"}

		for _, device := range devices {
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
			data = append(data, []interface{}{
				device.DeviceName, dt, v, device.Model, device.IPAddress,
				device.Port, device.SNMPPort, device.CredentialName, device.DeptName,
				device.Location, status, device.SerialNumber, device.SoftwareVersion,
				device.Uptime, formatTimePtr((*time.Time)(device.LastSeenAt)), device.Description,
				formatTime(time.Time(device.CreatedAt)),
			})
		}

	case "credentials":
		sheet = "授权凭证"
		headers = []string{"凭证名称", "协议类型", "用户名", "SNMP版本", "是否默认", "描述", "创建时间"}

		svc := services.NewAuthCredentialService(db, h.core.SM4Cipher)
		credReq := h.buildCredentialListRequest(req.Filters, current, pageSize)
		creds, _, err := svc.List(ctx, credReq)
		if err != nil {
			return nil, err
		}

		for _, cred := range creds {
			isDefault := "否"
			if cred.IsDefault {
				isDefault = "是"
			}
			data = append(data, []interface{}{
				cred.CredentialName, string(cred.ProtocolType), cred.Username,
				string(cred.SNMPVersion), isDefault, cred.Description,
				formatTime(time.Time(cred.CreatedAt)),
			})
		}

	case "templates":
		sheet = "配置模板"
		headers = []string{"模板名称", "模板编码", "模板类型", "厂商", "设备类型", "描述", "是否系统模板", "创建时间"}

		svc := services.NewTemplateService(db)
		tmplReq := h.buildTemplateListRequest(req.Filters, current, pageSize)
		templates, _, err := svc.List(ctx, tmplReq)
		if err != nil {
			return nil, err
		}

		templateTypeMap := map[string]string{"init": "初始化", "config": "配置", "backup": "备份"}
		vendorMap := map[string]string{"huawei": "华为", "h3c": "H3C", "ruijie": "锐捷", "maipu": "迈普"}
		deviceTypeMap := map[string]string{"router": "路由器", "switch": "交换机", "firewall": "防火墙", "ap": "AP", "loadbalancer": "负载均衡"}

		for _, tmpl := range templates {
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
			data = append(data, []interface{}{
				tmpl.TemplateName, tmpl.TemplateCode, tt, v, dt,
				tmpl.Description, isSys, formatTime(time.Time(tmpl.CreatedAt)),
			})
		}

	case "commands":
		sheet = "命令分发"
		headers = []string{"执行名称", "状态", "设备总数", "成功数", "失败数", "执行策略", "并发数", "超时(秒)", "开始时间", "完成时间", "错误信息", "创建人", "创建时间"}

		svc := services.NewCommandDispatchService(db, h.core.DeviceExecutor)
		records, _, err := svc.GetExecutionList(ctx, current, pageSize, "", nil)
		if err != nil {
			return nil, err
		}

		statusMap := map[int]string{0: "待执行", 1: "执行中", 2: "成功", 3: "失败", 4: "已取消"}

		for _, r := range records {
			status := statusMap[int(r.Status)]
			if status == "" {
				status = fmt.Sprintf("%d", r.Status)
			}
			data = append(data, []interface{}{
				r.ExecutionName, status, r.TotalDevices, r.SuccessCount, r.FailureCount,
				string(r.ExecutionStrategy), r.Concurrency, r.Timeout,
				formatTimePtr((*time.Time)(r.StartedAt)), formatTimePtr((*time.Time)(r.CompletedAt)),
				r.ErrorMessage, r.CreatedBy, formatTime(time.Time(r.CreatedAt)),
			})
		}

	case "executions":
		sheet = "配置执行"
		headers = []string{"执行名称", "状态", "设备总数", "成功数", "失败数", "执行策略", "并发数", "超时(秒)", "开始时间", "完成时间", "错误信息", "创建人", "创建时间"}

		svc := services.NewConfigExecutionService(db, h.core.DeviceExecutor)
		records, _, err := svc.GetExecutionList(ctx, current, pageSize, "", nil)
		if err != nil {
			return nil, err
		}

		statusMap := map[int]string{0: "待执行", 1: "执行中", 2: "成功", 3: "失败", 4: "已取消"}

		for _, r := range records {
			status := statusMap[int(r.Status)]
			if status == "" {
				status = fmt.Sprintf("%d", r.Status)
			}
			data = append(data, []interface{}{
				r.ExecutionName, status, r.TotalDevices, r.SuccessCount, r.FailureCount,
				string(r.ExecutionStrategy), r.Concurrency, r.Timeout,
				formatTimePtr((*time.Time)(r.StartedAt)), formatTimePtr((*time.Time)(r.CompletedAt)),
				r.ErrorMessage, r.CreatedBy, formatTime(time.Time(r.CreatedAt)),
			})
		}

	case "backups":
		sheet = "配置备份"
		headers = []string{"设备名称", "IP地址", "备份类型", "存储方式", "版本号", "配置大小(B)", "是否压缩", "变更原因", "创建人", "创建时间"}

		svc := services.NewConfigBackupService(db, h.core.DeviceExecutor)

		deviceID := ""
		if req.Filters != nil {
			if v, ok := req.Filters["deviceId"].(string); ok {
				deviceID = v
			}
		}

		records, _, err := svc.GetBackupList(ctx, current, pageSize, deviceID, "", nil)
		if err != nil {
			return nil, err
		}

		backupTypeMap := map[string]string{"auto": "自动", "manual": "手动"}
		storageTypeMap := map[string]string{"database": "数据库", "file": "文件"}

		for _, r := range records {
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
			data = append(data, []interface{}{
				r.DeviceName, r.IPAddress, bt, st, r.Version,
				r.BackupSize, compressed, r.ChangeReason,
				r.CreatedBy, formatTime(time.Time(r.CreatedAt)),
			})
		}

	case "discoveries":
		sheet = "设备发现"
		headers = []string{"任务名称", "发现类型", "状态", "IP总数", "已发现数", "自动导入", "SNMP团体名", "SNMP端口", "开始时间", "完成时间", "错误信息", "创建人", "创建时间"}

		svc := h.core.DeviceDiscoveryService
		records, _, err := svc.GetDiscoveryList(ctx, current, pageSize, "", nil)
		if err != nil {
			return nil, err
		}

		statusMap := map[int]string{0: "待执行", 1: "执行中", 2: "成功", 3: "失败", 4: "已取消"}
		discoveryTypeMap := map[string]string{"snmp": "SNMP发现", "scan": "扫描发现"}

		for _, r := range records {
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
			data = append(data, []interface{}{
				r.TaskName, dt, status, r.TotalIPs, r.DiscoveredCount,
				autoImport, r.SNMPCommunity, r.SNMPPort,
				formatTimePtr(r.StartedAt), formatTimePtr(r.CompletedAt),
				r.ErrorMessage, r.CreatedBy, formatTime(r.CreatedAt),
			})
		}

	case "mac":
		sheet = "MAC地址"
		headers = []string{"MAC地址", "接口名称", "VLAN ID", "MAC类型", "采集时间"}

		lldpSvc := lldp.NewLLDPService(h.core.DeviceExecutor)
		filterRuleSvc := topology.NewFilterRuleService(h.core.DB.GetDB())
		svc := services.NewMACCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor, lldpSvc, filterRuleSvc)

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

		records, _, err := svc.GetMACAddressList(ctx, current, pageSize, deviceID, deptID, macAddress, interfaceName, "", nil)
		if err != nil {
			return nil, err
		}

		for _, r := range records {
			data = append(data, []interface{}{
				r.MACAddress, r.InterfaceName, r.VLANID,
				string(r.MACType), formatTime(r.CollectedAt),
			})
		}

	case "ports":
		sheet = "端口状态"
		headers = []string{"接口名称", "管理状态", "操作状态", "描述", "VLAN", "双工模式", "速率", "端口类型", "802.1X", "802.1X状态", "端口安全", "安全模式", "最大MAC数", "当前MAC数", "采集时间"}

		portSvc := portcollection.NewPortCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor)

		portReq := &portcollection.ListRequest{
			BaseListRequest: base.BaseListRequest{
				Current:  current,
				PageSize: pageSize,
			},
		}
		if req.Filters != nil {
			if v, ok := req.Filters["deviceId"].(string); ok {
				portReq.DeviceID = v
			}
			if v, ok := req.Filters["interfaceName"].(string); ok {
				portReq.InterfaceName = v
			}
			if v, ok := req.Filters["adminStatus"].(string); ok {
				portReq.AdminStatus = v
			}
			if v, ok := req.Filters["operStatus"].(string); ok {
				portReq.OperStatus = v
			}
		}

		records, _, err := portSvc.Query.GetList(ctx, portReq)
		if err != nil {
			return nil, err
		}

		for _, r := range records {
			dot1x := "否"
			if r.Dot1xEnabled {
				dot1x = "是"
			}
			sec := "否"
			if r.PortSecurityEnabled {
				sec = "是"
			}
			data = append(data, []interface{}{
				r.InterfaceName, r.AdminStatus, r.OperStatus, r.Description,
				r.VLAN, r.Duplex, r.Speed, r.PortType,
				dot1x, r.Dot1xPortStatus, sec, r.PortSecurityMode,
				r.MaxMACCount, r.CurrentMACCount, formatTime(r.CollectedAt),
			})
		}

	default:
		return nil, fmt.Errorf("不支持的实体类型: %s", entityType)
	}

	// 设置工作表名称
	file.SetSheetName("Sheet1", sheet)

	// 写入表头
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheet, cell, header)
	}

	// 写入数据
	for i, row := range data {
		writeRow(file, sheet, i+2, row)
	}

	// 生成字节流
	var buf bytes.Buffer
	if _, err := file.WriteTo(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
