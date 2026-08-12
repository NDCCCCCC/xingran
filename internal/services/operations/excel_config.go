package operations

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
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
	UpsertKey bool        // 是否作为Upsert的唯一键（用于判断重复记录）
	DBField   string      // 数据库字段名（可选，如果与Field不同，例如 workstationName -> workstation_name）
	Default   interface{} // 当Excel值为空时的默认值
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
			{Field: "remark", Header: "备注", MaxLength: 500, DBField: "remark"},
		},
	},
	"user": {
		SheetName:     "用户列表",
		TableName:     "sys_user",
		EntityName:    "用户",
		HasHeader:     true,
		StartRow:      2,
		CachePatterns: []string{"user:*", "users:*"},
		UniqueKeys:    []string{"username"}, // 按用户名判断重复
		PartialUpdate: true,                 // 启用部分更新（只更新有值的字段）
		// ⚠ 列顺序必须与"人员详情.xlsx"一致：validateAndParseRow 按位置 row[i] 匹配，
		// 不按表头名（Header 仅用于错误提示）。Excel 实际列序：昵称/用户名/工号/邮箱/
		// 手机号/科室代码/科室名称（共 7 列）。
		Columns: []ExcelColumn{
			{Field: "nickname", Header: "昵称", MaxLength: 64, DBField: "nickname"},
			{Field: "username", Header: "用户名", Required: true, Unique: true, MaxLength: 64, UpsertKey: true, DBField: "username"},
			{Field: "employeeNo", Header: "工号", MaxLength: 64, DBField: "employee_no"},
			{Field: "email", Header: "邮箱", MaxLength: 128, DBField: "email"},
			{Field: "phone", Header: "手机号", MaxLength: 32, DBField: "phone"},
			{Field: "deptCode", Header: "科室代码", Reference: "sys_dept.dept_code", DBField: "dept_id"},
			{Field: "deptNameText", Header: "科室名称", MaxLength: 100}, // 临时字段，无DBField不写库（prepareRecordsForUpsert 跳过）
			// gender/status：Excel 无对应列（仅 7 列，index 7/8 越界），走 Default 填充
			{Field: "gender", Header: "性别", Options: map[interface{}]string{int(0): "男", int(1): "女", int(2): "保密"}, Default: int(2)},
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(0): "启用", int(1): "禁用"}, Default: int(0)},
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
			{Field: "remark", Header: "备注", MaxLength: 500, DBField: "remark"},
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
			// 部门代码：按 sys_dept.dept_code 解析为 dept_id；与既有 deptName 共存时以本字段为准；
			// 本字段为空且 deptName 非空时走名称匹配回退，向后兼容老模板。
			{Field: "deptCode", Header: "部门代码", MaxLength: 100, Reference: "sys_dept.dept_code", DBField: "dept_id"},
			{Field: "deptName", Header: "所属部门", Reference: "sys_dept.dept_name", DBField: "dept_id"},
			{Field: "userName", Header: "所属用户", Reference: "sys_user.username", DBField: "user_id"},
			// 主设备序列号：跨表特殊处理，无 DBField → prepareRecordsForUpsert 跳过不写 sys_workstation；
			// 由 ExcelService.ImportData post-import hook 调用 WorkstationDeviceService.SetPrimaryAndSaveBySerial
			// 写入 ops_workstation_device 表（IsPrimary=true，字段值由 AD/Asset 合并填充）。
			// 加 Reference:"skip" 使 shouldSkipFromUpdate 跳过，避免 device_serial 进入 ON CONFLICT DO UPDATE SET。
			{Field: "deviceSerial", Header: "主设备序列号", MaxLength: 200, Reference: "skip"},
			{Field: "remark", Header: "备注", MaxLength: 500, DBField: "description"},
		},
	},
	"asset": {
		SheetName:     "资产列表",
		TableName:     "ops_asset",
		EntityName:    "资产",
		HasHeader:     true,
		StartRow:      2,
		CachePatterns: []string{"asset:*"},
		UniqueKeys:    []string{"devicesn"},
		PartialUpdate: true,
		Columns: []ExcelColumn{
			// 按用户 Excel 文件顺序（43列）
			{Field: "deviceSN", Header: "设备序列号", Required: true, MaxLength: 200, UpsertKey: true, DBField: "devicesn"},
			{Field: "deviceCategorySecondName", Header: "设备中类", MaxLength: 100, DBField: "device_category_second_name"},
			{Field: "deviceTypeName", Header: "设备类型", MaxLength: 100, DBField: "device_type_name"},
			{Field: "deviceModelName", Header: "设备型号", MaxLength: 200, DBField: "device_model_name"},
			{Field: "qudaoName", Header: "设备渠道", MaxLength: 100, DBField: "qudao_name"},
			{Field: "attributeValue", Header: "设备属性", MaxLength: 500, DBField: "attribute_value"},
			{Field: "deviceBasicTypeName", Header: "是否固定资产", MaxLength: 50, DBField: "device_basic_type_name"},
			{Field: "fixAssetNo", Header: "固定资产编号", MaxLength: 100, DBField: "fixassetno"},
			{Field: "storageDatetime", Header: "入库日期", MaxLength: 50, DBField: "storage_datetime"},
			{Field: "useDate", Header: "发放日期", MaxLength: 50, DBField: "use_date"},
			{Field: "drawingDate", Header: "接收日期", MaxLength: 50, DBField: "drawing_date"},
			{Field: "useStatusLabel", Header: "状态", MaxLength: 50, DBField: "usestatus_label"},
			{Field: "nbfStatus", Header: "是否拟报废", DBField: "nbf_status", Options: map[interface{}]string{int(0): "否", int(1): "是"}},
			{Field: "signOrgnoName", Header: "归属机构", MaxLength: 100, DBField: "sign_orgno_name"},
			{Field: "orgnoName", Header: "使用机构", MaxLength: 100, DBField: "orgno_name"},
			{Field: "usefulDeptName", Header: "部门", MaxLength: 100, DBField: "useful_dept_name"},
			{Field: "deptName", Header: "受益部门", MaxLength: 100, DBField: "deptname"},
			{Field: "deviceUserName", Header: "领取人", MaxLength: 100, DBField: "deviceuser_name"},
			{Field: "nowUserName", Header: "责任人", MaxLength: 100, DBField: "nowuser_name"},
			{Field: "outerUser", Header: "使用人", MaxLength: 100, DBField: "outer_user"},
			{Field: "nowUserJobName", Header: "责任人岗位", MaxLength: 100, DBField: "nowuser_job_name"},
			{Field: "usingTypeName", Header: "用途", MaxLength: 100, DBField: "using_type_name"},
			{Field: "subUsingTypeName", Header: "子用途", MaxLength: 100, DBField: "sub_using_type_name"},
			{Field: "storeroomName", Header: "库房", MaxLength: 100, DBField: "storeroom_name"},
			{Field: "storageAddress", Header: "存放地址", MaxLength: 200, DBField: "storage_address"},
			{Field: "contractNo", Header: "合同号", MaxLength: 100, DBField: "contractno"},
			{Field: "printFlagName", Header: "资产标签打印状态", MaxLength: 50, DBField: "print_flag_name"},
			{Field: "errorFlagName", Header: "异常标识", MaxLength: 50, DBField: "error_flag_name"},
			{Field: "newFlagLabel", Header: "新设备标识", MaxLength: 50, DBField: "new_flag_label"},
			{Field: "remark", Header: "备注", MaxLength: 1000, DBField: "remark"},
			{Field: "isNoStandardName", Header: "申请标准", MaxLength: 100, DBField: "is_no_standard_name"},
			{Field: "machineIP", Header: "IP地址", MaxLength: 50, DBField: "machine_ip"},
			{Field: "machineBS", Header: "加域标识", MaxLength: 50, DBField: "machine_bs"},
			{Field: "domainIP", Header: "加域ip地址", MaxLength: 50, DBField: "machine_ip"},
			{Field: "mac1", Header: "有线MAC地址", MaxLength: 100, DBField: "mac1"},
			{Field: "mac2", Header: "无线MAC地址", MaxLength: 100, DBField: "mac2"},
			{Field: "machineUptime", Header: "最后上线时间", MaxLength: 50, DBField: "machine_uptime"},
			{Field: "machineUserID", Header: "最后上线账号", MaxLength: 100, DBField: "machine_user_id"},
			{Field: "userName", Header: "APP扫码账号", MaxLength: 100, Reference: "sys_user.username", DBField: "user_id"},
			{Field: "lastUpdateDate", Header: "APP扫码时间", MaxLength: 50, DBField: "last_update_date"},
			{Field: "scanSite", Header: "AAP扫码地理位置", MaxLength: 200, DBField: "scan_site"},
			{Field: "lastInventoryDate", Header: "最近一次盘点时间", MaxLength: 50, DBField: "last_inventory_date"},
			{Field: "inventoryResult", Header: "盘点结果", MaxLength: 50, DBField: "inventory_result"},
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
			{Field: "remark", Header: "备注", MaxLength: 500, DBField: "remark"},
		},
	},
	"roomDevice": {
		SheetName:  "机房设备列表",
		TableName:  "ops_room_devices",
		EntityName: "机房设备",
		HasHeader:  true,
		StartRow:   2,
		Columns: []ExcelColumn{
			{Field: "name", Header: "设备名称", Required: true, MaxLength: 100, DBField: "name"},
			{Field: "deviceCode", Header: "设备编码", Required: true, Unique: true, UpsertKey: true, MaxLength: 200, DBField: "device_code"},
			{Field: "deviceType", Header: "设备类型", DBField: "device_type", Default: string(operations.DeviceTypeOther), Options: map[interface{}]string{string(operations.DeviceTypeServer): "服务器", string(operations.DeviceTypeStorage): "存储设备", string(operations.DeviceTypeUPS): "UPS", string(operations.DeviceTypePDU): "PDU", string(operations.DeviceTypeFirewall): "防火墙", string(operations.DeviceTypeKVM): "KVM", string(operations.DeviceTypeCabinet): "机柜", string(operations.DeviceTypeOther): "其他"}},
			{Field: "vendor", Header: "厂商", MaxLength: 100, DBField: "vendor"},
			{Field: "model", Header: "型号", MaxLength: 100, DBField: "model"},
			{Field: "serialNumber", Header: "序列号", MaxLength: 100, DBField: "serial_number"},
			{Field: "roomName", Header: "所在机房", Required: true, Reference: "ops_server_rooms.name", DBField: "room_id"},
			{Field: "positionU", Header: "U位", DBField: "position_u"},
			{Field: "positionDesc", Header: "位置描述", MaxLength: 200, DBField: "position_desc"},
			{Field: "assetNumber", Header: "资产编号", MaxLength: 100, DBField: "asset_number"},
			{Field: "purchaseDate", Header: "购买日期", MaxLength: 50, DBField: "purchase_date"},
			{Field: "warrantyDate", Header: "保修到期", MaxLength: 50, DBField: "warranty_date"},
			{Field: "powerConsumption", Header: "功耗(W)", DBField: "power_consumption"},
			{Field: "status", Header: "状态", DBField: "status", Options: map[interface{}]string{0: "正常", 1: "故障", 2: "报废"}},
			{Field: "remark", Header: "备注", MaxLength: 500, DBField: "remark"},
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
			{Field: "infoPointType", Header: "信息点类型", Options: map[interface{}]string{string(operations.InfoPointTypeNetwork): "网络信息点", string(operations.InfoPointTypePower): "电源信息点", string(operations.InfoPointTypeOther): "其他"}, DBField: "info_point_type", Default: string(operations.InfoPointTypeNetwork)},
			{Field: "workstationName", Header: "关联工位名称", Required: true, MaxLength: 100, Reference: "sys_workstation.workstation_name", DBField: "workstation_id"}, // 用户提供工位名称，自动转换为workstationId
			{Field: "deviceName", Header: "所属设备", MaxLength: 100, Reference: "sys_network_device.device_name", DBField: "device_id"},
			{Field: "portName", Header: "所属端口", MaxLength: 100, Reference: "sys_device_port_status.interface_name", DBField: "port_id", DependsOn: "deviceName"}, // 依赖 deviceName 解析结果(避免同名接口跨设备误选)
			{Field: "status", Header: "状态", Options: map[interface{}]string{int(operations.InfoPointStatusNormal): "正常", int(operations.InfoPointStatusFault): "故障", int(operations.InfoPointStatusDisabled): "停用"}, DBField: "status"},
			{Field: "description", Header: "描述", MaxLength: 500, DBField: "remark"},
		},
	},
	// Phase 44 R3 / Plan 44-02 Task 4 — 资产对账例外规则 Excel 导入导出
	//
	// 方案 B (WARN-7 锁定):
	//   - 9 列顺序严格 = name/ip_range/conflict_types/exception_actions/severity_override/scope_type/scope_name/expires_at/reason
	//     (项目记忆 xingran-excel-import-column-position-matching: validateAndParseRow 按 row[i] 位置匹配)
	//   - name 列 UpsertKey=true + DBField="name"(项目记忆 xingran-excel-import-upsertkey-needs-dbfield)
	//   - scopeName 是临时字段无 DBField, 由 ImportFromExcel 后处理按 scope_type 解析为 scope_id
	//     (dept→sys_dept.dept_name / user→sys_user.username / global→NULL)
	//   - conflict_types / exception_actions 用 TEXT 存(后处理转 TEXT[])
	//   - 不在 router.go 预注册 /asset/reconciliation/import (项目记忆 xingran-excel-import-route-conflict)
	"reconciliationExceptionRule": {
		SheetName:     "对账例外规则",
		TableName:     "sys_reconciliation_exception",
		EntityName:    "例外规则",
		HasHeader:     true,
		StartRow:      2,
		CachePatterns: []string{"reconciliation:*"},
		UniqueKeys:    []string{"name"},
		PartialUpdate: true,
		Columns: []ExcelColumn{
			{Field: "name", Header: "规则名称", Required: true, MaxLength: 128, UpsertKey: true, DBField: "name"},
			{Field: "ipRange", Header: "IP段(CIDR)", Required: true, DBField: "ip_range"},
			{Field: "conflictTypes", Header: "冲突类型(逗号分隔B/C/D/E/F,空=全部)", DBField: "conflict_types"},
			{Field: "exceptionActions", Header: "动作(逗号分隔no_alert/no_notice/no_workorder/skip_severity/silence)", Required: true, DBField: "exception_actions"},
			{Field: "severityOverride", Header: "严重度覆盖(low/medium/high,可空)", DBField: "severity_override"},
			{Field: "scopeType", Header: "范围(global/dept/user)", DBField: "scope_type"},
			{Field: "scopeName", Header: "范围名称(部门名/用户名,global留空)"}, // 无 DBField, 临时字段 (后处理 UPDATE scope_id)
			{Field: "expiresAt", Header: "过期时间(可空,YYYY-MM-DD)", DBField: "expires_at"},
			{Field: "reason", Header: "原因(≥10字符)", Required: true, DBField: "reason"},
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
