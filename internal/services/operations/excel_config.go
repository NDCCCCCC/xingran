package operations

import (
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
)

// ExcelColumn Excel列配置
type ExcelColumn struct {
	Field     string                 // 字段名
	Header    string                 // 表头名称
	Required  bool                   // 是否必填
	Unique    bool                   // 是否唯一
	MinLength int                    // 最小长度
	MaxLength int                    // 最大长度
	Pattern   string                 // 正则表达式
	Options   map[interface{}]string // 枚举值映射（值 -> 显示文本）
	Reference string                 // 关联表
}

// ExcelConfig Excel配置
type ExcelConfig struct {
	SheetName  string        // 工作表名称
	TableName  string        // 数据表名
	EntityName string        // 实体名称
	Columns    []ExcelColumn // 列配置
	HasHeader  bool          // 是否有表头
	StartRow   int           // 数据起始行（从1开始）
}

// ExcelConfigs 所有Excel配置映射
var ExcelConfigs = map[string]ExcelConfig{
	"user": {
		SheetName:  "用户列表",
		TableName:  "sys_user",
		EntityName: "用户",
		HasHeader:  true,
		StartRow:   2,
		Columns: []ExcelColumn{
			{Field: "username", Header: "用户名", Required: true, Unique: true, MaxLength: 64},
			{Field: "nickname", Header: "用户昵称", MaxLength: 64},
			{Field: "employeeNo", Header: "工号", MaxLength: 64},
			{Field: "email", Header: "邮箱", MaxLength: 128},
			{Field: "phone", Header: "手机号", MaxLength: 32},
			{Field: "gender", Header: "性别", Options: map[interface{}]string{int(0): "男", int(1): "女", int(2): "未知"}},
			{Field: "deptCode", Header: "部门编码", Reference: "sys_dept.dept_code"},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(0): "正常", int(1): "停用"}},
			{Field: "remark", Header: "备注", MaxLength: 500},
		},
	},
	"building": {
		SheetName:  "楼宇列表",
		TableName:  "ops_buildings",
		EntityName: "楼宇",
		HasHeader:  true,
		StartRow:   2, // 第一行是表头，从第二行开始是数据
		Columns: []ExcelColumn{
			{Field: "name", Header: "楼宇名称", Required: true, MaxLength: 100},
			{Field: "code", Header: "楼宇编码", Required: true, Unique: true, MaxLength: 50},
			{Field: "address", Header: "地址", MaxLength: 500},
			{Field: "totalFloors", Header: "总楼层数", MinLength: 1},
			{Field: "totalArea", Header: "总面积(㎡)"},
			{Field: "description", Header: "描述", MaxLength: 500},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(operations.BuildingStatusNormal): "正常", int(operations.BuildingStatusStopped): "停用"}},
		},
	},
	"floor": {
		SheetName:  "楼层列表",
		TableName:  "ops_floors",
		EntityName: "楼层",
		HasHeader:  true,
		StartRow:   2,
		Columns: []ExcelColumn{
			{Field: "buildingCode", Header: "楼宇编码", Required: true, Reference: "ops_buildings.code"},
			{Field: "floorNo", Header: "楼层号", Required: true, MaxLength: 10},
			{Field: "name", Header: "楼层名称", MaxLength: 100},
			{Field: "area", Header: "面积(㎡)"},
			{Field: "description", Header: "描述", MaxLength: 500},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(operations.FloorStatusNormal): "正常", int(operations.FloorStatusStopped): "停用"}},
		},
	},
	"workstation": {
		SheetName:  "工位列表",
		TableName:  "sys_workstation",
		EntityName: "工位",
		HasHeader:  true,
		StartRow:   2,
		Columns: []ExcelColumn{
			{Field: "floorCode", Header: "楼层编码", Required: true, Reference: "ops_floors.floor_no"},
			{Field: "workstationCode", Header: "工位编码", Required: true, MaxLength: 50},
			{Field: "name", Header: "工位名称", Required: true, MaxLength: 100},
			{Field: "workstationType", Header: "工位类型", Options: map[interface{}]string{int(models.WorkstationTypeFixed): "固定工位", int(models.WorkstationTypeHotDesk): "灵活工位", int(models.WorkstationTypeManager): "管理工位"}},
			{Field: "capacity", Header: "容纳人数"},
			{Field: "area", Header: "面积(㎡)"},
			{Field: "facilities", Header: "设施", MaxLength: 500},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(models.WorkstationStatusAvailable): "空闲", int(models.WorkstationStatusOccupied): "占用", int(models.WorkstationStatusMaintain): "维护"}},
		},
	},
	"server-room": {
		SheetName:  "机房列表",
		TableName:  "ops_server_rooms",
		EntityName: "机房",
		HasHeader:  true,
		StartRow:   2,
		Columns: []ExcelColumn{
			{Field: "buildingCode", Header: "楼宇编码", Required: true, Reference: "ops_buildings.code"},
			{Field: "floorNo", Header: "楼层号", Required: true, MaxLength: 10},
			{Field: "roomCode", Header: "机房编码", Required: true, MaxLength: 50},
			{Field: "name", Header: "机房名称", Required: true, MaxLength: 100},
			{Field: "area", Header: "面积(㎡)"},
			{Field: "rackCount", Header: "机柜数量"},
			{Field: "powerCapacity", Header: "电力容量(kW)"},
			{Field: "description", Header: "备注", MaxLength: 500},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(operations.RoomStatusNormal): "正常", int(operations.RoomStatusStopped): "停用"}},
		},
	},
	"room-device": {
		SheetName:  "机房设备列表",
		TableName:  "ops_room_devices",
		EntityName: "机房设备",
		HasHeader:  true,
		StartRow:   2,
		Columns: []ExcelColumn{
			{Field: "roomCode", Header: "机房编码", Required: true, Reference: "ops_server_rooms.code"},
			{Field: "deviceCode", Header: "设备编码", Required: true, MaxLength: 50},
			{Field: "deviceName", Header: "设备名称", Required: true, MaxLength: 100},
			{Field: "deviceType", Header: "设备类型", Options: map[interface{}]string{string(operations.DeviceTypeServer): "服务器", string(operations.DeviceTypeStorage): "存储设备", string(operations.DeviceTypeUPS): "UPS", string(operations.DeviceTypePDU): "PDU", string(operations.DeviceTypeFirewall): "防火墙", string(operations.DeviceTypeKVM): "KVM", string(operations.DeviceTypeCabinet): "机柜", string(operations.DeviceTypeOther): "其他"}},
			{Field: "vendor", Header: "厂商", MaxLength: 100},
			{Field: "model", Header: "型号", MaxLength: 100},
			{Field: "serialNumber", Header: "序列号", MaxLength: 100},
			{Field: "positionDesc", Header: "位置描述", MaxLength: 200},
			{Field: "positionU", Header: "U位"},
			{Field: "powerConsumption", Header: "功耗(W)"},
			{Field: "purchaseDate", Header: "购买日期", MaxLength: 50},
			{Field: "warrantyDate", Header: "保修到期", MaxLength: 50},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(operations.RoomDeviceStatusNormal): "正常", int(operations.RoomDeviceStatusFault): "故障", int(operations.RoomDeviceStatusScrapped): "报废"}},
		},
	},
	"dedicated-line": {
		SheetName:  "专线列表",
		TableName:  "ops_dedicated_lines",
		EntityName: "专线",
		HasHeader:  true,
		StartRow:   2,
		Columns: []ExcelColumn{
			{Field: "lineCode", Header: "专线编码", Required: true, MaxLength: 50},
			{Field: "lineType", Header: "专线类型", Options: map[interface{}]string{string(operations.LineTypeMPLS): "MPLS VPN", string(operations.LineTypeInternet): "互联网专线", string(operations.LineTypeLeasedLine): "租用专线", string(operations.LineTypeFiber): "光纤专线"}},
			{Field: "bandwidth", Header: "带宽", MaxLength: 50},
			{Field: "isp", Header: "运营商", Required: true, MaxLength: 100},
			{Field: "sourceDeviceName", Header: "源设备名称", MaxLength: 100},
			{Field: "sourcePort", Header: "源端口", MaxLength: 50},
			{Field: "destDeviceName", Header: "目的设备名称", MaxLength: 100},
			{Field: "destPort", Header: "目的端口", MaxLength: 50},
			{Field: "ipAddress", Header: "IP地址", MaxLength: 50},
			{Field: "subnetMask", Header: "子网掩码", MaxLength: 50},
			{Field: "vlan", Header: "VLAN", MaxLength: 50},
			{Field: "carrierContactName", Header: "运营商联系人", MaxLength: 100},
			{Field: "carrierContactPhone", Header: "运营商电话", MaxLength: 20},
			{Field: "contractNo", Header: "合同编号", MaxLength: 100},
			{Field: "monthlyFee", Header: "月租(元)"},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(operations.LineStatusNormal): "正常", int(operations.LineStatusFault): "故障", int(operations.LineStatusDisabled): "停用"}},
		},
	},
	"info-point": {
		SheetName:  "信息点列表",
		TableName:  "ops_info_points",
		EntityName: "信息点",
		HasHeader:  true,
		StartRow:   2,
		Columns: []ExcelColumn{
			{Field: "workstationCode", Header: "工位编码", Required: true, Reference: "sys_workstation.workstation_code"},
			{Field: "name", Header: "信息点名称", Required: true, MaxLength: 100},
			{Field: "code", Header: "信息点编码", Required: true, MaxLength: 50},
			{Field: "infoPointType", Header: "信息点类型", Options: map[interface{}]string{string(operations.InfoPointTypeNetwork): "网络信息点", string(operations.InfoPointTypePower): "电源信息点", string(operations.InfoPointTypeOther): "其他"}},
			{Field: "deviceName", Header: "设备名称", MaxLength: 100},
			{Field: "portName", Header: "端口名称", MaxLength: 100},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(operations.InfoPointStatusNormal): "正常", int(operations.InfoPointStatusFault): "故障", int(operations.InfoPointStatusDisabled): "停用"}},
		},
	},
}

// GetExcelConfig 获取指定实体的Excel配置
func GetExcelConfig(entityType string) (ExcelConfig, bool) {
	config, exists := ExcelConfigs[entityType]
	return config, exists
}

// GetAllEntityTypes 获取所有支持导入导出的实体类型
func GetAllEntityTypes() []string {
	types := make([]string, 0, len(ExcelConfigs))
	for t := range ExcelConfigs {
		types = append(types, t)
	}
	return types
}
