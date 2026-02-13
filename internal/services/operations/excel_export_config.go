package operations

type ExcelExportField struct {
	Field       string                    // 导出字段名（对应数据库或计算字段）
	Header      string                    // Excel列标题
	DBField     string                    // 数据库字段名（可选，如果与Field不同）
	Options     map[interface{}]string    // 枚举值映射
	Join        *JoinConfig               // 关联查询配置（可选）
	Formatter   string                    // 格式化器名称（可选）
}

type JoinConfig struct {
	Table       string  // 关联表名
	LeftField   string  // 左表字段（通常是ID）
	RightField  string  // 右表字段
	SelectField string  // 要查询的字段
	As          string  // 别名
}

type ExcelExportConfig struct {
	SheetName      string
	TableName      string
	EntityName     string
	Columns        []ExcelExportField
	HasHeader      bool
	StartRow       int
	QueryBuilder   string                    // 查询构建器名称（可选）
	FilterMapping  map[string]string         // 参数名 -> 数据库字段名
}

func GetExportConfig(entityType string) (ExcelExportConfig, bool) {
	configs := map[string]ExcelExportConfig{
		"department": {
			SheetName:  "部门列表",
			TableName:  "sys_dept",
			EntityName: "部门",
			HasHeader:  true,
			StartRow:   2,
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
						SelectField: "nick_name",
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
			SheetName:  "楼宇列表",
			TableName:  "ops_buildings",
			EntityName: "楼宇",
			HasHeader:  true,
			StartRow:   2,
			QueryBuilder: "default",
			FilterMapping: map[string]string{
				"name":   "name",
				"code":   "code",
				"status": "status",
			},
			Columns: []ExcelExportField{
				{Field: "name", Header: "楼宇名称", DBField: "name"},
				{Field: "address", Header: "地址", DBField: "address"},
				{Field: "longitude", Header: "经度", DBField: "longitude"},
				{Field: "latitude", Header: "纬度", DBField: "latitude"},
				{Field: "level", Header: "层级", DBField: "level",
					Options: map[interface{}]string{1: "城市级汇总", 2: "具体楼宇"}},
				{Field: "status", Header: "状态", DBField: "status",
					Options: map[interface{}]string{int(0): "正常", int(1): "停用"}},
				{Field: "remark", Header: "备注", DBField: "remark"},
			},
		},
		"floor": {
			SheetName:  "楼层列表",
			TableName:  "ops_floors",
			EntityName: "楼层",
			HasHeader:  true,
			StartRow:   2,
			QueryBuilder: "default",
			FilterMapping: map[string]string{
				"name":   "name",
				"status": "status",
			},
			Columns: []ExcelExportField{
				{Field: "name", Header: "楼层名称", DBField: "name"},
				{Field: "buildingName", Header: "所属楼宇",
					Join: &JoinConfig{Table: "ops_buildings", LeftField: "building_id",
						RightField: "id", SelectField: "name", As: "buildingName"}},
				{Field: "floorNo", Header: "楼层号", DBField: "floor_no"}, // 使用 floor_no 与 GORM 模型字段名一致
				{Field: "area", Header: "面积(㎡)", DBField: "area"},
				{Field: "status", Header: "状态", DBField: "status",
					Options: map[interface{}]string{int(0): "正常", int(1): "停用"}},
				{Field: "remark", Header: "备注", DBField: "remark"},
			},
		},
		"workstation": {
			SheetName:  "工位列表",
			TableName:  "ops_workstations",
			EntityName: "工位",
			HasHeader:  true,
			StartRow:   2,
			QueryBuilder: "default",
			FilterMapping: map[string]string{
				"name":   "name",
				"status": "status",
			},
			Columns: []ExcelExportField{
				{Field: "name", Header: "工位名称", DBField: "name"},
				{Field: "floorName", Header: "所属楼层",
					Join: &JoinConfig{Table: "ops_floors", LeftField: "floor_id",
						RightField: "id", SelectField: "name", As: "floorName"}},
				{Field: "area", Header: "面积(㎡)", DBField: "area"},
				{Field: "capacity", Header: "容纳人数", DBField: "capacity"},
				{Field: "status", Header: "状态", DBField: "status",
					Options: map[interface{}]string{int(0): "启用", int(1): "禁用"}},
				{Field: "remark", Header: "备注", DBField: "remark"},
			},
		},
		"serverRoom": {
			SheetName:  "机房列表",
			TableName:  "ops_server_rooms",
			EntityName: "机房",
			HasHeader:  true,
			StartRow:   2,
			QueryBuilder: "default",
			FilterMapping: map[string]string{
				"name":   "name",
				"status": "status",
			},
			Columns: []ExcelExportField{
				{Field: "name", Header: "机房名称", DBField: "name"},
				{Field: "buildingName", Header: "所属楼宇",
					Join: &JoinConfig{Table: "ops_buildings", LeftField: "building_id",
						RightField: "id", SelectField: "name", As: "buildingName"}},
				{Field: "floorName", Header: "所属楼层",
					Join: &JoinConfig{Table: "ops_floors", LeftField: "floor_id",
						RightField: "id", SelectField: "name", As: "floorName"}},
				{Field: "status", Header: "状态", DBField: "status",
					Options: map[interface{}]string{int(0): "正常", int(1): "停用"}},
				{Field: "remark", Header: "备注", DBField: "remark"},
			},
		},
		"roomDevice": {
			SheetName:  "机房设备列表",
			TableName:  "ops_room_devices",
			EntityName: "机房设备",
			HasHeader:  true,
			StartRow:   2,
			QueryBuilder: "default",
			FilterMapping: map[string]string{
				"name":   "name",
				"status": "status",
			},
			Columns: []ExcelExportField{
				{Field: "name", Header: "设备名称", DBField: "name"},
				{Field: "roomName", Header: "所属机房",
					Join: &JoinConfig{Table: "ops_server_rooms", LeftField: "room_id",
						RightField: "id", SelectField: "name", As: "roomName"}},
				{Field: "deviceType", Header: "设备类型", DBField: "device_type"},
				{Field: "model", Header: "规格型号", DBField: "model"},
				{Field: "vendor", Header: "厂商", DBField: "vendor"},
				{Field: "status", Header: "状态", DBField: "status",
					Options: map[interface{}]string{int(0): "正常", int(1): "故障", int(2): "报废"}},
				{Field: "remark", Header: "备注", DBField: "remark"},
			},
		},
		"dedicatedLine": {
			SheetName:  "专线列表",
			TableName:  "ops_dedicated_lines",
			EntityName: "专线",
			HasHeader:  true,
			StartRow:   2,
			QueryBuilder: "default",
			FilterMapping: map[string]string{
				"name":   "name",
				"status": "status",
			},
			Columns: []ExcelExportField{
				{Field: "name", Header: "专线名称", DBField: "name"},
				{Field: "lineType", Header: "专线类型", DBField: "line_type"},
				{Field: "bandwidth", Header: "带宽", DBField: "bandwidth"},
				{Field: "isp", Header: "运营商", DBField: "isp"},
				{Field: "sourceRoomName", Header: "源机房",
					Join: &JoinConfig{Table: "ops_server_rooms", LeftField: "source_room_id",
						RightField: "id", SelectField: "name", As: "sourceRoomName"}},
				{Field: "destRoomName", Header: "目的机房",
					Join: &JoinConfig{Table: "ops_server_rooms", LeftField: "dest_room_id",
						RightField: "id", SelectField: "name", As: "destRoomName"}},
				{Field: "status", Header: "状态", DBField: "status",
					Options: map[interface{}]string{int(0): "正常", int(1): "故障", int(2): "停用"}},
				{Field: "remark", Header: "备注", DBField: "remark"},
			},
		},
		"infoPoint": {
			SheetName:  "信息点列表",
			TableName:  "ops_info_points",
			EntityName: "信息点",
			HasHeader:  true,
			StartRow:   2,
			QueryBuilder: "default",
			FilterMapping: map[string]string{
				"name":   "name",
				"status": "status",
			},
			Columns: []ExcelExportField{
				{Field: "name", Header: "信息点名称", DBField: "name"},
				{Field: "infoPointType", Header: "信息点类型", DBField: "info_point_type",
					Options: map[interface{}]string{"network": "网络信息点", "power": "电源信息点", "other": "其他"}},
				{Field: "workstationName", Header: "所属工位",
					Join: &JoinConfig{Table: "ops_workstations", LeftField: "workstation_id",
						RightField: "id", SelectField: "name", As: "workstationName"}},
				{Field: "deviceName", Header: "关联设备",
					Join: &JoinConfig{Table: "sys_network_device", LeftField: "device_id",
						RightField: "id", SelectField: "name", As: "deviceName"}},
				{Field: "portName", Header: "端口名称",
					Join: &JoinConfig{Table: "sys_device_port_status", LeftField: "port_id",
						RightField: "id", SelectField: "port_name", As: "portName"}},
				{Field: "status", Header: "状态", DBField: "status",
					Options: map[interface{}]string{int(0): "正常", int(1): "故障", int(2): "停用"}},
				{Field: "remark", Header: "备注", DBField: "remark"},
			},
		},
	}

	config, exists := configs[entityType]
	return config, exists
}
