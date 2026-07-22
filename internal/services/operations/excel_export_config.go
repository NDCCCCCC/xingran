package operations

type ExcelExportField struct {
	Field     string                 // 导出字段名（对应数据库或计算字段）
	Header    string                 // Excel列标题
	DBField   string                 // 数据库字段名（可选，如果与Field不同）
	Options   map[interface{}]string // 枚举值映射
	Join      *JoinConfig            // 关联查询配置（可选）
	Formatter string                 // 格式化器名称（可选）
}

type JoinConfig struct {
	Table          string // 关联表名
	LeftField      string // 左表字段（通常是ID）
	RightField     string // 右表字段
	SelectField    string // 要查询的字段
	As             string // 别名
	RightCast      string // 选填, RightField 类型转换(如 "text"),用于 IN 子句类型匹配
	SkipSoftDelete bool   // 选填: true 时跳过 `deleted_at IS NULL` 过滤(用于无 deleted_at 列的硬删除表,如 sys_device_port_status)
}

type ExcelExportConfig struct {
	SheetName     string
	TableName     string
	EntityName    string
	Columns       []ExcelExportField
	HasHeader     bool
	StartRow      int
	QueryBuilder  string            // 查询构建器名称（可选）
	FilterMapping map[string]string // 参数名 -> 数据库字段名
}

func GetExportConfig(entityType string) (ExcelExportConfig, bool) {
	configs := map[string]ExcelExportConfig{
		"department": {
			SheetName:    "部门列表",
			TableName:    "sys_dept",
			EntityName:   "部门",
			HasHeader:    true,
			StartRow:     2,
			QueryBuilder: "department",
			FilterMapping: map[string]string{
				"name":   "dept_name",
				"code":   "dept_code",
				"status": "status",
			},
			Columns: []ExcelExportField{
				{
					Field:   "deptCode",
					Header:  "科室编码(SECTION_OFFICE_CODE)",
					DBField: "dept_code",
				},
				{
					Field:   "deptName",
					Header:  "科室名称(SECTION_OFFICE_NAME)",
					DBField: "dept_name",
				},
				{
					Field:  "leaderName",
					Header: "负责人姓名",
					Join: &JoinConfig{
						Table:       "sys_user",
						LeftField:   "leader",
						RightField:  "id",
						SelectField: "nickname",
						As:          "leaderName",
					},
				},
				{
					Field:  "parentName",
					Header: "上级部门名称",
					Join: &JoinConfig{
						Table:       "sys_dept",
						LeftField:   "parent_id",
						RightField:  "id",
						SelectField: "dept_name",
						As:          "parentName",
					},
				},
				// parentPath: 完整父级链路("根 → … → 当前"),由 DepartmentQueryBuilder 的
				// PostgreSQL recursive CTE 一次性算好注入到结果集,不走 resolveAssociations Join。
				// 解决同名部门(例如多个"个人营销业务销售部")在不同市州中心支公司下无法区分的问题。
				// 必须在 parentName 之后追加,避免影响前端 column_config_service.go:206 的列顺序契约。
				{
					Field:   "parentPath",
					Header:  "上级部门链路(根 → 当前)",
					DBField: "parent_path",
				},
				{
					Field:   "phone",
					Header:  "联系电话",
					DBField: "phone",
				},
				{
					Field:   "email",
					Header:  "邮箱",
					DBField: "email",
				},
				{
					Field:   "orderNum",
					Header:  "显示顺序",
					DBField: "order_num",
				},
				{
					Field:   "isExternalOrg",
					Header:  "是否外部机构",
					DBField: "is_external_org",
					Options: map[interface{}]string{int(0): "否", int(1): "是"},
				},
				{
					Field:   "status",
					Header:  "状态",
					DBField: "status",
					Options: map[interface{}]string{int(0): "正常", int(1): "停用"},
				},
				{
					Field:   "remark",
					Header:  "备注",
					DBField: "remark",
				},
			},
		},
		"building": {
			SheetName:    "楼宇列表",
			TableName:    "ops_buildings",
			EntityName:   "楼宇",
			HasHeader:    true,
			StartRow:     2,
			QueryBuilder: "default",
			FilterMapping: map[string]string{
				"name":   "name",
				"code":   "code",
				"status": "status",
				"orgId":  "org_id",
				"level":  "level",
			},
			Columns: []ExcelExportField{
				{Field: "name", Header: "楼宇名称", DBField: "name"},
				{Field: "code", Header: "楼宇编码", DBField: "code"},
				{Field: "address", Header: "地址", DBField: "address"},
				{Field: "longitude", Header: "经度", DBField: "longitude"},
				{Field: "latitude", Header: "纬度", DBField: "latitude"},
				{Field: "level", Header: "层级", DBField: "level",
					Options: map[interface{}]string{int(1): "城市级汇总", int(2): "具体楼宇"}},
				{Field: "orgId", Header: "所属机构ID", DBField: "org_id"},
				{Field: "totalFloors", Header: "楼层数", DBField: "total_floors"},
				{Field: "status", Header: "状态", DBField: "status",
					Options: map[interface{}]string{int(0): "正常", int(1): "停用"}},
				{Field: "remark", Header: "备注", DBField: "remark"},
				{Field: "createdAt", Header: "创建时间", DBField: "created_at"},
				{Field: "updatedAt", Header: "更新时间", DBField: "updated_at"},
			},
		},
		"floor": {
			SheetName:    "楼层列表",
			TableName:    "ops_floors",
			EntityName:   "楼层",
			HasHeader:    true,
			StartRow:     2,
			QueryBuilder: "default",
			FilterMapping: map[string]string{
				"name":       "name",
				"status":     "status",
				"buildingId": "building_id",
			},
			Columns: []ExcelExportField{
				{Field: "name", Header: "楼层名称", DBField: "name"},
				{Field: "buildingId", Header: "所属楼宇ID", DBField: "building_id"},
				{Field: "buildingName", Header: "所属楼宇",
					Join: &JoinConfig{Table: "ops_buildings", LeftField: "building_id",
						RightField: "id", SelectField: "name", As: "buildingName"}},
				{Field: "floorNo", Header: "楼层号", DBField: "floor_no"},
				{Field: "area", Header: "面积(㎡)", DBField: "area"},
				{Field: "status", Header: "状态", DBField: "status",
					Options: map[interface{}]string{int(0): "正常", int(1): "停用"}},
				{Field: "remark", Header: "备注", DBField: "remark"},
				{Field: "createdAt", Header: "创建时间", DBField: "created_at"},
				{Field: "updatedAt", Header: "更新时间", DBField: "updated_at"},
			},
		},
		"workstation": {
			SheetName:    "工位列表",
			TableName:    "sys_workstation",
			EntityName:   "工位",
			HasHeader:    true,
			StartRow:     2,
			QueryBuilder: "workstation",
			FilterMapping: map[string]string{
				"name":    "workstation_name",
				"status":  "status",
				"floorId": "floor_id",
				"deptId":  "dept_id",
				"userId":  "user_id",
			},
			Columns: []ExcelExportField{
				{Field: "workstationName", Header: "工位名称", DBField: "workstation_name"},
				{Field: "floorId", Header: "所属楼层ID", DBField: "floor_id"},
				{Field: "floorName", Header: "所属楼层", DBField: "floor_name"},
				{Field: "buildingId", Header: "所属楼宇ID", DBField: "building_id"},
				{Field: "buildingName", Header: "所属楼宇", DBField: "building_name"},
				{Field: "deptId", Header: "所属部门ID", DBField: "dept_id"},
				{Field: "deptName", Header: "所属部门", DBField: "dept_name"},
				{Field: "userId", Header: "所属人员ID", DBField: "user_id"},
				{Field: "userName", Header: "所属人员", DBField: "user_name"},
				{Field: "deviceSerial", Header: "设备序列号", DBField: "device_serial"},
				{Field: "deviceName", Header: "设备名称", DBField: "device_name"},
				{Field: "deviceModel", Header: "设备型号", DBField: "device_model"},
				{Field: "workstationType", Header: "工位类型", DBField: "workstation_type",
					Options: map[interface{}]string{int(0): "固定工位", int(1): "灵活工位", int(2): "管理工位"}},
				{Field: "location", Header: "位置描述", DBField: "location"},
				{Field: "status", Header: "状态", DBField: "status",
					Options: map[interface{}]string{int(0): "空闲", int(1): "占用", int(2): "维护"}},
				{Field: "remark", Header: "备注", DBField: "description"},
				{Field: "createdAt", Header: "创建时间", DBField: "created_at"},
				{Field: "updatedAt", Header: "更新时间", DBField: "updated_at"},
			},
		},
		"serverRoom": {
			SheetName:    "机房列表",
			TableName:    "ops_server_rooms",
			EntityName:   "机房",
			HasHeader:    true,
			StartRow:     2,
			QueryBuilder: "default",
			FilterMapping: map[string]string{
				"name":       "name",
				"status":     "status",
				"buildingId": "building_id",
				"floorId":    "floor_id",
			},
			Columns: []ExcelExportField{
				{Field: "name", Header: "机房名称", DBField: "name"},
				{Field: "buildingId", Header: "所属楼宇ID", DBField: "building_id"},
				{Field: "buildingName", Header: "所属楼宇",
					Join: &JoinConfig{Table: "ops_buildings", LeftField: "building_id",
						RightField: "id", SelectField: "name", As: "buildingName"}},
				{Field: "floorId", Header: "所属楼层ID", DBField: "floor_id"},
				{Field: "floorName", Header: "所属楼层",
					Join: &JoinConfig{Table: "ops_floors", LeftField: "floor_id",
						RightField: "id", SelectField: "name", As: "floorName"}},
				{Field: "status", Header: "状态", DBField: "status",
					Options: map[interface{}]string{int(0): "正常", int(1): "停用"}},
				{Field: "remark", Header: "备注", DBField: "remark"},
				{Field: "createdAt", Header: "创建时间", DBField: "created_at"},
				{Field: "updatedAt", Header: "更新时间", DBField: "updated_at"},
			},
		},
		"roomDevice": {
			SheetName:    "机房设备列表",
			TableName:    "ops_room_devices",
			EntityName:   "机房设备",
			HasHeader:    true,
			StartRow:     2,
			QueryBuilder: "default",
			FilterMapping: map[string]string{
				"name":     "name",
				"status":   "status",
				"roomCode": "room_code",
			},
			Columns: []ExcelExportField{
				{Field: "name", Header: "设备名称", DBField: "name"},
				{Field: "deviceCode", Header: "设备编码", DBField: "device_code"},
				{Field: "roomId", Header: "所属机房ID", DBField: "room_id"},
				{Field: "roomName", Header: "所属机房",
					Join: &JoinConfig{Table: "ops_server_rooms", LeftField: "room_id",
						RightField: "id", SelectField: "name", As: "roomName"}},
				{Field: "deviceType", Header: "设备类型", DBField: "device_type",
					Options: map[interface{}]string{"server": "服务器", "storage": "存储设备", "ups": "UPS", "pdu": "PDU", "firewall": "防火墙", "kvm": "KVM", "cabinet": "机柜", "other": "其他"}},
				{Field: "model", Header: "规格型号", DBField: "model"},
				{Field: "vendor", Header: "厂商", DBField: "vendor"},
				{Field: "serialNumber", Header: "序列号", DBField: "serial_number"},
				{Field: "positionU", Header: "机架位置(U)", DBField: "position_u"},
				{Field: "positionDesc", Header: "位置描述", DBField: "position_desc"},
				{Field: "assetNumber", Header: "资产编号", DBField: "asset_number"},
				{Field: "status", Header: "状态", DBField: "status",
					Options: map[interface{}]string{int(0): "正常", int(1): "故障", int(2): "报废"}},
				{Field: "remark", Header: "备注", DBField: "remark"},
				{Field: "createdAt", Header: "创建时间", DBField: "created_at"},
				{Field: "updatedAt", Header: "更新时间", DBField: "updated_at"},
			},
		},
		"dedicatedLine": {
			SheetName:    "专线列表",
			TableName:    "ops_dedicated_lines",
			EntityName:   "专线",
			HasHeader:    true,
			StartRow:     2,
			QueryBuilder: "default",
			FilterMapping: map[string]string{
				"name":   "name",
				"status": "status",
				"isp":    "isp",
			},
			Columns: []ExcelExportField{
				{Field: "name", Header: "专线名称", DBField: "name"},
				{Field: "lineType", Header: "专线类型", DBField: "line_type",
					Options: map[interface{}]string{"internet": "互联网专线", "intranet": "内网专线", "cloud_desktop": "云桌面专线", "mpls": "MPLS专线", "fiber": "光纤专线", "leased_line": "租用线路"}},
				{Field: "bandwidth", Header: "带宽", DBField: "bandwidth"},
				{Field: "isp", Header: "运营商", DBField: "isp"},
				{Field: "sourceRoomId", Header: "源机房ID", DBField: "source_room_id"},
				{Field: "sourceRoomName", Header: "源机房",
					Join: &JoinConfig{Table: "ops_server_rooms", LeftField: "source_room_id",
						RightField: "id", SelectField: "name", As: "sourceRoomName"}},
				{Field: "destRoomId", Header: "目的机房ID", DBField: "dest_room_id"},
				{Field: "destRoomName", Header: "目的机房",
					Join: &JoinConfig{Table: "ops_server_rooms", LeftField: "dest_room_id",
						RightField: "id", SelectField: "name", As: "destRoomName"}},
				{Field: "sourceDeviceId", Header: "源设备ID", DBField: "source_device_id"},
				{Field: "sourceDeviceName", Header: "源设备", DBField: "source_device_name"},
				{Field: "sourcePort", Header: "源端口", DBField: "source_port"},
				{Field: "destDeviceId", Header: "目的设备ID", DBField: "dest_device_id"},
				{Field: "destDeviceName", Header: "目的设备", DBField: "dest_device_name"},
				{Field: "destPort", Header: "目的端口", DBField: "dest_port"},
				{Field: "sourceIPAddress", Header: "源IP地址", DBField: "source_ip_address"},
				{Field: "destIPAddress", Header: "目的IP地址", DBField: "dest_ip_address"},
				{Field: "vlan", Header: "VLAN", DBField: "vlan"},
				{Field: "carrierContactName", Header: "运营商联系人", DBField: "carrier_contact_name"},
				{Field: "carrierContactPhone", Header: "联系人电话", DBField: "carrier_contact_phone"},
				{Field: "monthlyFee", Header: "月租费用", DBField: "monthly_fee"},
				{Field: "status", Header: "状态", DBField: "status",
					Options: map[interface{}]string{int(0): "正常", int(1): "故障", int(2): "停用"}},
				{Field: "remark", Header: "备注", DBField: "remark"},
				{Field: "createdAt", Header: "创建时间", DBField: "created_at"},
				{Field: "updatedAt", Header: "更新时间", DBField: "updated_at"},
			},
		},
		"infoPoint": {
			SheetName:    "信息点列表",
			TableName:    "ops_info_points",
			EntityName:   "信息点",
			HasHeader:    true,
			StartRow:     2,
			QueryBuilder: "default",
			FilterMapping: map[string]string{
				"name":          "name",
				"status":        "status",
				"workstationId": "workstation_id",
				"infoPointType": "info_point_type",
			},
			Columns: []ExcelExportField{
				{Field: "name", Header: "信息点名称", DBField: "name"},
				{Field: "infoPointType", Header: "信息点类型", DBField: "info_point_type",
					Options: map[interface{}]string{"network": "网络信息点", "power": "电源信息点", "other": "其他"}},
				{Field: "workstationId", Header: "所属工位ID", DBField: "workstation_id"},
				{Field: "workstationName", Header: "所属工位",
					Join: &JoinConfig{Table: "sys_workstation", LeftField: "workstation_id",
						RightField: "id", RightCast: "text", SelectField: "workstation_name", As: "workstationName", SkipSoftDelete: true}}, // SkipSoftDelete: 109 个软删除工位恢复名称显示(诊断: sys_workstation ids=1168 results=1059 matched, 109 未匹配是软删)
				{Field: "deviceId", Header: "关联设备ID", DBField: "device_id"},
				{Field: "deviceName", Header: "关联设备",
					Join: &JoinConfig{Table: "sys_network_device", LeftField: "device_id",
						RightField: "id", RightCast: "text", SelectField: "device_name", As: "deviceName"}},
				{Field: "portId", Header: "端口ID", DBField: "port_id"},
				{Field: "portName", Header: "端口名称",
					Join: &JoinConfig{Table: "sys_device_port_status", LeftField: "port_id",
						RightField: "id", RightCast: "text", SelectField: "interface_name", As: "portName", SkipSoftDelete: true}}, // sys_device_port_status 是硬删除表(无 deleted_at 列)
				{Field: "status", Header: "状态", DBField: "status",
					Options: map[interface{}]string{int(0): "正常", int(1): "故障", int(2): "停用"}},
				{Field: "remark", Header: "备注", DBField: "remark"},
				{Field: "createdAt", Header: "创建时间", DBField: "created_at"},
				{Field: "updatedAt", Header: "更新时间", DBField: "updated_at"},
			},
		},
	}

	config, exists := configs[entityType]
	return config, exists
}
