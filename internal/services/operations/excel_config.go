package operations

import (
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
)

// ExcelOption Excel选项值
type ExcelOption struct {
	Value interface{}
	Label string
}

// ExcelOptionsMap 选项映射类型
type ExcelOptionsMap map[interface{}]string

// ToOptionMap 将 ExcelOption 列表转换为映射
func ToOptionMap(opts []ExcelOption) ExcelOptionsMap {
	result := make(ExcelOptionsMap, len(opts))
	for _, opt := range opts {
		result[opt.Value] = opt.Label
	}
	return result
}

// ExcelColumn Excel列配置
type ExcelColumn struct {
	Field     string
	Header    string
	Required  bool
	Unique    bool
	MinLength int
	MaxLength int
	Pattern   string
	Options   ExcelOptionsMap
	Reference string
}

// ExcelConfig Excel配置
type ExcelConfig struct {
	SheetName  string
	TableName  string
	EntityName string
	Columns    []ExcelColumn
	HasHeader  bool
	StartRow   int
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
			{Field: "cityCode", Header: "城市代码", MaxLength: 20},
			{Field: "cityName", Header: "城市名称", MaxLength: 50},
			{Field: "address", Header: "地址", MaxLength: 200},
			{Field: "longitude", Header: "经度"},
			{Field: "latitude", Header: "纬度"},
			{Field: "parentId", Header: "父级楼宇ID", MaxLength: 64},
			{Field: "level", Header: "层级", Options: map[interface{}]string{1: "城市级汇总", 2: "具体楼宇"}},
			{Field: "orgId", Header: "所属机构ID", MaxLength: 64},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(operations.BuildingStatusNormal): "正常", int(operations.BuildingStatusStopped): "停用"}},
			{Field: "remark", Header: "备注", MaxLength: 500},
		},
	},
	"floor": {
		SheetName:  "楼层列表",
		TableName:  "ops_floors",
		EntityName: "楼层",
		HasHeader:  true,
		StartRow:   2,
		Columns: []ExcelColumn{
			{Field: "name", Header: "楼层名称", Required: true, MaxLength: 100},
			{Field: "floorNo", Header: "楼层号", Required: true, MaxLength: 50},
			{Field: "buildingId", Header: "所属楼宇ID", Required: true, Reference: "ops_buildings.id"},
			{Field: "buildingName", Header: "所属楼宇名称", MaxLength: 100},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(operations.FloorStatusNormal): "正常", int(operations.FloorStatusStopped): "停用"}},
			{Field: "remark", Header: "备注", MaxLength: 500},
		},
	},
	"workstation": {
		SheetName:  "工位列表",
		TableName:  "sys_workstation",
		EntityName: "工位",
		HasHeader:  true,
		StartRow:   2,
		Columns: []ExcelColumn{
			{Field: "name", Header: "工位名称", Required: true, MaxLength: 100},
			{Field: "floorId", Header: "所属楼层ID", Required: true, Reference: "ops_floors.id"},
			{Field: "floorName", Header: "所属楼层名称", MaxLength: 100},
			{Field: "workstationType", Header: "工位类型", Options: map[interface{}]string{int(models.WorkstationTypeFixed): "固定工位", int(models.WorkstationTypeHotDesk): "灵活工位", int(models.WorkstationTypeManager): "管理工位"}},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(models.WorkstationStatusAvailable): "空闲", int(models.WorkstationStatusOccupied): "占用", int(models.WorkstationStatusMaintain): "维护"}},
			{Field: "remark", Header: "备注", MaxLength: 500},
		},
	},
	"server-room": {
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
	"room-device": {
		SheetName:  "机房设备列表",
		TableName:  "ops_room_devices",
		EntityName: "机房设备",
		HasHeader:  true,
		StartRow:   2,
		Columns: []ExcelColumn{
			{Field: "name", Header: "设备名称", Required: true, MaxLength: 100},
			{Field: "deviceCode", Header: "设备编码", Required: true, Unique: true, MaxLength: 50},
			{Field: "deviceType", Header: "设备类型", Options: map[interface{}]string{string(operations.DeviceTypeServer): "服务器", string(operations.DeviceTypeStorage): "存储设备", string(operations.DeviceTypeUPS): "UPS", string(operations.DeviceTypePDU): "PDU", string(operations.DeviceTypeFirewall): "防火墙", string(operations.DeviceTypeKVM): "KVM", string(operations.DeviceTypeCabinet): "机柜", string(operations.DeviceTypeOther): "其他"}},
			{Field: "vendor", Header: "厂商", MaxLength: 100},
			{Field: "model", Header: "型号", MaxLength: 100},
			{Field: "serialNumber", Header: "序列号", MaxLength: 100},
			{Field: "roomId", Header: "所在机房ID", Required: true, Reference: "ops_server_rooms.id"},
			{Field: "roomName", Header: "所在机房名称", MaxLength: 100},
			{Field: "positionU", Header: "U位"},
			{Field: "positionDesc", Header: "位置描述", MaxLength: 200},
			{Field: "assetNumber", Header: "资产编号", MaxLength: 100},
			{Field: "purchaseDate", Header: "购买日期", MaxLength: 50},
			{Field: "warrantyDate", Header: "保修到期", MaxLength: 50},
			{Field: "powerConsumption", Header: "功耗(W)"},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(operations.RoomDeviceStatusNormal): "正常", int(operations.RoomDeviceStatusFault): "故障", int(operations.RoomDeviceStatusScrapped): "报废"}},
			{Field: "remark", Header: "备注", MaxLength: 500},
		},
	},
	"dedicated-line": {
		SheetName:  "专线列表",
		TableName:  "ops_dedicated_lines",
		EntityName: "专线",
		HasHeader:  true,
		StartRow:   2,
		Columns: []ExcelColumn{
			{Field: "name", Header: "专线名称", Required: true, MaxLength: 100},
			{Field: "lineType", Header: "专线类型", Required: true, Options: map[interface{}]string{"internet": "互联网专线", "intranet": "内网专线", "cloud_desktop": "云桌面专线", "mpls": "MPLS VPN", "fiber": "光纤专线", "leased_line": "租用专线"}},
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
			{Field: "remark", Header: "备注", MaxLength: 500},
		},
	},
	"info-point": {
		SheetName:  "信息点列表",
		TableName:  "ops_info_points",
		EntityName: "信息点",
		HasHeader:  true,
		StartRow:   2,
		Columns: []ExcelColumn{
			{Field: "name", Header: "信息点名称", Required: true, MaxLength: 100},
			{Field: "infoPointType", Header: "信息点类型", Options: map[interface{}]string{string(operations.InfoPointTypeNetwork): "网络信息点", string(operations.InfoPointTypePower): "电源信息点", string(operations.InfoPointTypeOther): "其他"}},
			{Field: "workstationId", Header: "关联工位ID", Required: true, Reference: "sys_workstation.id"},
			{Field: "workstationName", Header: "关联工位名称", MaxLength: 100},
			{Field: "deviceId", Header: "关联设备ID", MaxLength: 64},
			{Field: "deviceName", Header: "关联设备名称", MaxLength: 100},
			{Field: "portId", Header: "关联端口ID", MaxLength: 64},
			{Field: "portName", Header: "关联端口名称", MaxLength: 100},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(operations.InfoPointStatusNormal): "正常", int(operations.InfoPointStatusFault): "故障", int(operations.InfoPointStatusDisabled): "停用"}},
			{Field: "remark", Header: "备注", MaxLength: 500},
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
