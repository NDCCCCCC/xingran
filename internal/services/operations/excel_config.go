package operations

import (
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
)

const (
	StatusNormal               = "正常"
	StatusStopped              = "停用"
	StatusFault                = "故障"
	StatusDisabled             = "停用"
	WorkstationTypeFixed       = "固定工位"
	WorkstationTypeHotDesk     = "灵活工位"
	WorkstationTypeManager     = "管理工位"
	WorkstationStatusAvailable = "空闲"
	WorkstationStatusOccupied  = "占用"
	WorkstationStatusMaintain  = "维护"
)

type ExcelColumn struct {
	Field     string
	Header    string
	Required  bool
	Unique    bool
	MinLength int
	MaxLength int
	Pattern   string
	Options   map[interface{}]string
	Reference string // 引用配置，格式："table.field"，用于名称到ID的转换
	DependsOn string // 依赖字段：该字段的引用解析依赖于另一个字段的解析结果（如 floorName 依赖 buildingName）
	UpsertKey bool   // 是否作为Upsert的唯一键（用于判断重复记录）
	DBField   string // 数据库字段名（可选，如果与Field不同，例如 workstationName -> workstation_name）
}

type ExcelConfig struct {
	SheetName     string
	TableName     string
	EntityName    string
	Columns       []ExcelColumn
	HasHeader     bool
	StartRow      int
	CachePatterns []string // 导入后需要清理的缓存模式
	UniqueKeys    []string // 用于判断重复的字段组合（备用，未配置UpsertKey时使用）
	PartialUpdate bool     // 是否启用部分更新（只更新有值的字段）
	Instructions  []string // Excel 模板顶部的说明文字，每行一个单元格
}

var ExcelConfigs = map[string]ExcelConfig{
	"department": {
		SheetName:     "部门列表",
		TableName:     "sys_dept",
		EntityName:    "部门",
		HasHeader:     true,
		StartRow:      2,
		CachePatterns: []string{"dept:*", "departments:*"},
		UniqueKeys:    []string{"dept_code"}, // 按科室编码判断重复（使用数据库字段名）
		PartialUpdate: true,                  // 启用部分更新（只更新有值的字段）
		Columns: []ExcelColumn{
			{Field: "deptCode", Header: "科室编码(SECTION_OFFICE_CODE)", Required: true, Unique: true, MaxLength: 100, UpsertKey: true, DBField: "dept_code"},
			{Field: "deptName", Header: "科室名称(SECTION_OFFICE_NAME)", Required: true, MaxLength: 100, DBField: "dept_name"},
			{Field: "departmentCode", Header: "部门编码(DEPARTMENT_CODE)", Required: true, MaxLength: 100},             // 临时字段，无DBField，不会写入数据库
			{Field: "departmentName", Header: "部门名称(DEPARTMENT_NAME)", Required: true, MaxLength: 100},             // 临时字段，无DBField
			{Field: "departmentGroupCode", Header: "部门组编码(DEPARTMENT_GROUP_CODE)", Required: true, MaxLength: 100}, // 临时字段，无DBField
			{Field: "departmentGroupName", Header: "部门组名称(DEPARTMENT_GROUP_NAME)", Required: true, MaxLength: 100}, // 临时字段，无DBField
			{Field: "leader", Header: "负责人", MaxLength: 100},
			{Field: "phone", Header: "联系电话", MaxLength: 50},
			{Field: "email", Header: "邮箱", MaxLength: 100},
			{Field: "orderNum", Header: "显示顺序", DBField: "order_num"},
			{Field: "isExternalOrg", Header: "是否外部机构", Options: map[interface{}]string{int(0): "否", int(1): "是"}, DBField: "is_external_org"},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(0): "正常", int(1): "停用"}},
			{Field: "remark", Header: "备注", MaxLength: 500},
		},
	},
	"building": {
		SheetName:     "楼宇列表",
		TableName:     "ops_buildings",
		EntityName:    "楼宇",
		HasHeader:     true,
		StartRow:      2,
		CachePatterns: []string{"building:*", "buildings:*"},
		UniqueKeys:    []string{"name", "orgId"}, // 按名称+机构判断重复
		Columns: []ExcelColumn{
			{Field: "name", Header: "楼宇名称", Required: true, MaxLength: 100, UpsertKey: true},
			{Field: "address", Header: "地址", MaxLength: 200, DBField: "position_desc"},
			{Field: "orgName", Header: "所属机构名称/编码", Required: true, MaxLength: 100, Reference: "sys_dept.dept_code", DBField: "org_id"}, // 用户提供机构名称或编码，自动转换为orgId
			{Field: "level", Header: "层级", Options: map[interface{}]string{1: "城市级汇总", 2: "具体楼宇"}},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(operations.BuildingStatusNormal): "正常", int(operations.BuildingStatusStopped): "停用"}},
			{Field: "remark", Header: "备注", MaxLength: 500},
		},
	},
	"floor": {
		SheetName:     "楼层列表",
		TableName:     "ops_floors",
		EntityName:    "楼层",
		HasHeader:     true,
		StartRow:      2,
		CachePatterns: []string{"floor:*", "floors:*"},
		UniqueKeys:    []string{"building_id", "floor_no"}, // 按楼宇+楼层号判断重复（使用数据库字段名）
		Columns: []ExcelColumn{
			{Field: "name", Header: "楼层名称", Required: true, MaxLength: 100, DBField: "name"},
			{Field: "floorNo", Header: "楼层号", Required: true, MaxLength: 50, DBField: "floor_no"},
			{Field: "buildingName", Header: "所属楼宇名称", Required: true, MaxLength: 100, Reference: "ops_buildings.name", DBField: "building_id"},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(operations.FloorStatusNormal): "正常", int(operations.FloorStatusStopped): "停用"}, DBField: "status"},
			{Field: "remark", Header: "备注", MaxLength: 500, DBField: "remark"},
		},
	},
	"workstation": {
		SheetName:     "工位列表",
		TableName:     "sys_workstation",
		EntityName:    "工位",
		HasHeader:     true,
		StartRow:      2,
		CachePatterns: []string{"workstation:*", "workstations:*"},
		UniqueKeys:    []string{"floorName", "name"}, // 按楼层+工位名称判断重复
		Columns: []ExcelColumn{
			{Field: "name", Header: "工位名称", Required: true, MaxLength: 100, DBField: "workstation_name"},
			{Field: "buildingName", Header: "所属楼宇", MaxLength: 100, Reference: "ops_buildings.name", DBField: "building_id"},                                     // Excel列名：所属楼宇
			{Field: "floorName", Header: "所属楼层名称", Required: true, MaxLength: 100, Reference: "ops_floors.name", DBField: "floor_id", DependsOn: "buildingName"}, // 依赖 buildingName 的解析结果
			{Field: "workstationType", Header: "工位类型", DBField: "workstation_type", Options: map[interface{}]string{int(models.WorkstationTypeFixed): "固定工位", int(models.WorkstationTypeHotDesk): "灵活工位", int(models.WorkstationTypeManager): "管理工位"}},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(models.WorkstationStatusAvailable): "空闲", int(models.WorkstationStatusOccupied): "占用", int(models.WorkstationStatusMaintain): "维护"}},
			{Field: "deptName", Header: "所属部门", Reference: "sys_dept.dept_name", DBField: "dept_id"},
			{Field: "userName", Header: "所属用户", Reference: "sys_user.username", DBField: "user_id"},
			{Field: "remark", Header: "备注", MaxLength: 500, DBField: "description"},
		},
	},
	"serverRoom": {
		SheetName:  "机房列表",
		TableName:  "ops_server_rooms",
		EntityName: "机房",
		HasHeader:  true,
		StartRow:   2,
		Columns: []ExcelColumn{
			{Field: "name", Header: "机房名称", Required: true, MaxLength: 100},
			{Field: "buildingId", Header: "所属楼宇ID", Required: true, Reference: "ops_buildings.id"},
			{Field: "buildingName", Header: "所属楼宇名称", MaxLength: 100},
			{Field: "floorId", Header: "所在楼层ID", Required: true, Reference: "ops_floors.id"},
			{Field: "floorName", Header: "所在楼层名称", MaxLength: 100},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(operations.RoomStatusNormal): "正常", int(operations.RoomStatusStopped): "停用"}},
			{Field: "remark", Header: "备注", MaxLength: 500},
		},
	},
	"roomDevice": {
		SheetName:  "机房设备列表",
		TableName:  "ops_room_devices",
		EntityName: "机房设备",
		HasHeader:  true,
		StartRow:   2,
		Columns: []ExcelColumn{
			{Field: "name", Header: "设备名称", Required: true, MaxLength: 100},
			{Field: "deviceCode", Header: "设备编码", Required: true, Unique: true, MaxLength: 50, DBField: "device_code"},
			{Field: "deviceType", Header: "设备类型", DBField: "device_type", Options: map[interface{}]string{string(operations.DeviceTypeServer): "服务器", string(operations.DeviceTypeStorage): "存储设备", string(operations.DeviceTypeUPS): "UPS", string(operations.DeviceTypePDU): "PDU", string(operations.DeviceTypeFirewall): "防火墙", string(operations.DeviceTypeKVM): "KVM", string(operations.DeviceTypeCabinet): "机柜", string(operations.DeviceTypeOther): "其他"}},
			{Field: "vendor", Header: "厂商", MaxLength: 100},
			{Field: "model", Header: "型号", MaxLength: 100},
			{Field: "serialNumber", Header: "序列号", MaxLength: 100, DBField: "serial_number"},
			{Field: "roomName", Header: "所在机房", Required: true, Reference: "ops_server_rooms.name", DBField: "room_id"},
			{Field: "positionU", Header: "U位", DBField: "position_u"},
			{Field: "positionDesc", Header: "位置描述", MaxLength: 200},
			{Field: "assetNumber", Header: "资产编号", MaxLength: 100},
			{Field: "purchaseDate", Header: "购买日期", MaxLength: 50},
			{Field: "warrantyDate", Header: "保修到期", MaxLength: 50},
			{Field: "powerConsumption", Header: "功耗(W)", DBField: "power_consumption"},
			{Field: "status", Header: "状态", Options: map[interface{}]string{0: "正常", 1: "故障", 2: "报废"}},
			{Field: "remark", Header: "备注", MaxLength: 500},
		},
	},
	"dedicatedLine": {
		SheetName:  "专线列表",
		TableName:  "ops_dedicated_lines",
		EntityName: "专线",
		HasHeader:  true,
		StartRow:   2,
		Columns: []ExcelColumn{
			{Field: "name", Header: "专线名称", Required: true, MaxLength: 100},
			{Field: "lineType", Header: "专线类型", Required: true, Options: map[interface{}]string{"internet": "互联网专线", "intranet": "内网专线", "cloud_desktop": "云桌面专线", "mpls": "MPLS VPN", "fiber": "光纤专线", "leased_line": "租用专线"}, DBField: "line_type"},
			{Field: "bandwidth", Header: "带宽", MaxLength: 50, DBField: "bandwidth"},
			{Field: "isp", Header: "运营商", Required: true, MaxLength: 100, DBField: "isp"},
			{Field: "sourceRoomName", Header: "源机房", MaxLength: 100, DBField: "source_room_name"},
			{Field: "sourceDeviceName", Header: "源设备名称", MaxLength: 100, DBField: "source_device_name"},
			{Field: "sourcePort", Header: "源端口", MaxLength: 50, DBField: "source_port"},
			{Field: "sourceIpAddress", Header: "源IP地址", MaxLength: 50, DBField: "source_ip_address", Pattern: `^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`},
			{Field: "sourceSubnetMask", Header: "源子网掩码", MaxLength: 50, DBField: "source_subnet_mask", Pattern: `^((255|254|252|248|240|224|192|128|0)\.){3}(255|254|252|248|240|224|192|128|0)$`},
			{Field: "destRoomName", Header: "目的机房", MaxLength: 100, DBField: "dest_room_name"},
			{Field: "destDeviceName", Header: "目的设备名称", MaxLength: 100, DBField: "dest_device_name"},
			{Field: "destPort", Header: "目的端口", MaxLength: 50, DBField: "dest_port"},
			{Field: "destIpAddress", Header: "目的IP地址", MaxLength: 50, DBField: "dest_ip_address", Pattern: `^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`},
			{Field: "destSubnetMask", Header: "目的子网掩码", MaxLength: 50, DBField: "dest_subnet_mask", Pattern: `^((255|254|252|248|240|224|192|128|0)\.){3}(255|254|252|248|240|224|192|128|0)$`},
			{Field: "vlan", Header: "VLAN", MaxLength: 50, DBField: "vlan"},
			{Field: "carrierContactName", Header: "运营商联系人", MaxLength: 100, DBField: "carrier_contact_name"},
			{Field: "carrierContactPhone", Header: "运营商电话", MaxLength: 20, DBField: "carrier_contact_phone"},
			{Field: "monthlyFee", Header: "月租(元)", DBField: "monthly_fee"},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(operations.LineStatusNormal): "正常", int(operations.LineStatusFault): "故障", int(operations.LineStatusDisabled): "停用"}, DBField: "status"},
			{Field: "remark", Header: "备注", MaxLength: 500, DBField: "remark"},
		},
	},
	"infoPoint": {
		SheetName:     "信息点列表",
		TableName:     "ops_info_points",
		EntityName:    "信息点",
		HasHeader:     true,
		StartRow:      2,
		CachePatterns: []string{"infoPoint:*", "info_points:*"},
		UniqueKeys:    []string{"workstation_id", "name"}, // 按工位+名称判断重复（使用数据库字段名）
		PartialUpdate: true,                               // 启用部分更新以避免 GORM 无法识别部分索引的问题
		Columns: []ExcelColumn{
			{Field: "name", Header: "信息点名称", Required: true, MaxLength: 100, DBField: "name"},
			{Field: "infoPointType", Header: "信息点类型", Options: map[interface{}]string{string(operations.InfoPointTypeNetwork): "网络信息点", string(operations.InfoPointTypePower): "电源信息点", string(operations.InfoPointTypeOther): "其他"}, DBField: "info_point_type"},
			{Field: "workstationName", Header: "关联工位名称", Required: true, MaxLength: 100, Reference: "sys_workstation.workstation_name", DBField: "workstation_id"}, // 用户提供工位名称，自动转换为workstationId
			{Field: "deviceName", Header: "所属设备", MaxLength: 100, Reference: "sys_network_device.device_name", DBField: "device_id"},
			{Field: "portName", Header: "所属端口", MaxLength: 100, Reference: "sys_device_port_status.interface_name", DBField: "port_id"},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(operations.InfoPointStatusNormal): "正常", int(operations.InfoPointStatusFault): "故障", int(operations.InfoPointStatusDisabled): "停用"}, DBField: "status"},
			{Field: "remark", Header: "备注", MaxLength: 500, DBField: "remark"},
		},
	},
}

func GetExcelConfig(entityType string) (ExcelConfig, bool) {
	config, exists := ExcelConfigs[entityType]
	return config, exists
}

func GetAllEntityTypes() []string {
	types := make([]string, 0, len(ExcelConfigs))
	for t := range ExcelConfigs {
		types = append(types, t)
	}
	return types
}
