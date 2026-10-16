package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AssetStatus 资产状态常量（Asset.Status 字段, int 型, 0=正常 1=停用 — Phase 69 DICT-01）。
// 保持无类型以便直接赋值给 int 字段。
const (
	AssetStatusNormal  = 0 // 正常
	AssetStatusStopped = 1 // 停用
)

// AssetNBFStatus 拟报废标识常量（Asset.NBFStatus 字段, 独立布尔维度, 0=否 1=拟报废）。
const (
	AssetNBFStatusNo  = 0 // 否
	AssetNBFStatusYes = 1 // 拟报废
)

// Asset 资产模型
type Asset struct {
	ID        string         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`

	// 核心标识 (3 fields)
	DeviceSN   string  `gorm:"type:varchar;not null;uniqueIndex;column:devicesn" json:"devicesn"` // 设备序列号（唯一标识）
	SequenceNo string  `gorm:"size:100;column:sequenceno" json:"sequenceNo,omitempty"`            // 序列号
	FixAssetNo *string `gorm:"size:100;column:fixassetno" json:"fixAssetNo,omitempty"`            // 固定资产编号

	// 设备信息 (4 fields)
	DeviceModelName     *string `gorm:"size:200;column:device_model_name" json:"deviceModelName,omitempty"`                    // 型号
	DeviceTypeName      *string `gorm:"size:100;column:device_type_name" json:"deviceTypeName,omitempty"`                      // 类型
	DeviceCategoryName  *string `gorm:"size:100;column:device_category_second_name" json:"deviceCategorySecondName,omitempty"` // 中类
	DeviceBasicTypeName *string `gorm:"size:50;column:device_basic_type_name" json:"deviceBasicTypeName,omitempty"`            // 是否固定资产

	// 用户关联 (4 fields)
	DeviceUserName *string `gorm:"size:100;column:deviceuser_name" json:"deviceUserName,omitempty"` // 领取人
	NowUserName    *string `gorm:"type:varchar;column:nowuser_name" json:"nowUserName,omitempty"`   // 责任人
	NowUserP13     *string `gorm:"size:100;column:nowuser_p13" json:"nowUserP13,omitempty"`         // 责任人p13
	DeviceUserP13  *string `gorm:"size:100;column:deviceuser_p13" json:"deviceUserP13,omitempty"`   // 领取人p13

	// 部门关联 (3 fields)
	DeptName        *string `gorm:"type:varchar;column:deptname" json:"deptName,omitempty"`             // 受益部门
	NowUserDeptCode *string `gorm:"size:100;column:nowuser_dept_code" json:"nowUserDeptCode,omitempty"` // 部门编码
	XNDeptCode      *string `gorm:"size:100;column:xndept_code" json:"xnDeptCode,omitempty"`            // 受益部门编码

	// 状态标识 (4 fields)
	UseStatusLabel *string `gorm:"size:50;column:usestatus_label" json:"useStatusLabel,omitempty"` // 状态
	NewFlagLabel   *string `gorm:"size:50;column:new_flag_label" json:"newFlagLabel,omitempty"`    // 新设备标识
	PrintFlagName  *string `gorm:"size:50;column:print_flag_name" json:"printFlagName,omitempty"`  // 打印状态
	NBFStatus      *int    `gorm:"default:0;column:nbf_status" json:"nbfStatus,omitempty"`         // 是否拟报废 (0=否, 1=是)

	// 时间字段 (7 fields)
	DrawingDate       *time.Time `gorm:"column:drawing_date" json:"drawingDate,omitempty"`              // 接收日期
	UseDate           *time.Time `gorm:"column:use_date" json:"useDate,omitempty"`                      // 发放日期
	StorageDatetime   *time.Time `gorm:"column:storage_datetime" json:"storageDatetime,omitempty"`      // 入库日期
	LastUpdateDate    *time.Time `gorm:"column:last_update_date" json:"lastUpdateDate,omitempty"`       // APP扫码时间
	Y07UpdateTime     *time.Time `gorm:"column:y07_update_time" json:"y07UpdateTime,omitempty"`         // Y07更新时间
	MachineUptime     *time.Time `gorm:"column:machine_uptime" json:"machineUptime,omitempty"`          // 最后上线时间
	LastInventoryDate *time.Time `gorm:"column:last_inventory_date" json:"lastInventoryDate,omitempty"` // 最近一次盘点时间

	// 网络信息 (4 fields)
	MAC1      *string `gorm:"type:varchar;column:mac1" json:"mac1,omitempty"`            // 有线MAC
	MAC2      *string `gorm:"type:varchar;column:mac2" json:"mac2,omitempty"`            // 无线MAC
	MachineIP *string `gorm:"type:varchar;column:machine_ip" json:"machineIp,omitempty"` // 加域IP
	MachineBS *string `gorm:"size:50;column:machine_bs" json:"machineBs,omitempty"`      // 加域标识

	// 合同与属性 (2 fields)
	ContractNo     *string `gorm:"size:100;column:contractno" json:"contractNo,omitempty"`          // 合同号
	AttributeValue *string `gorm:"size:500;column:attribute_value" json:"attributeValue,omitempty"` // 设备属性

	// 位置与归属 (8 fields)
	ScanSite         *string `gorm:"size:200;column:scan_site" json:"scanSite,omitempty"`                   // APP扫码地理位置
	Remark           *string `gorm:"size:1000;column:remark" json:"remark,omitempty"`                       // 备注
	QudaoName        *string `gorm:"size:100;column:qudao_name" json:"qudaoName,omitempty"`                 // 设备渠道
	UsingTypeName    *string `gorm:"size:100;column:using_type_name" json:"usingTypeName,omitempty"`        // 用途
	SubUsingTypeName *string `gorm:"size:100;column:sub_using_type_name" json:"subUsingTypeName,omitempty"` // 子用途
	OrgnoName        *string `gorm:"size:100;column:orgno_name" json:"orgnoName,omitempty"`                 // 使用机构
	StoreroomName    *string `gorm:"size:100;column:storeroom_name" json:"storeroomName,omitempty"`         // 库房
	StorageAddress   *string `gorm:"size:200;column:storage_address" json:"storageAddress,omitempty"`       // 存放地址

	// 机构与标准 (3 fields)
	SignOrgnoName    *string `gorm:"size:100;column:sign_orgno_name" json:"signOrgnoName,omitempty"`        // 归属机构
	IsNoStandardName *string `gorm:"size:100;column:is_no_standard_name" json:"isNoStandardName,omitempty"` // 申请标准名称
	ErrorFlagName    *string `gorm:"size:50;column:error_flag_name" json:"errorFlagName,omitempty"`         // 异常标识

	// 外部与部门用户 (5 fields)
	OuterUser       *string `gorm:"size:100;column:outer_user" json:"outerUser,omitempty"`            // 使用人
	UsefulDeptName  *string `gorm:"size:100;column:useful_dept_name" json:"usefulDeptName,omitempty"` // 部门名称
	NowUserJobName  *string `gorm:"size:100;column:nowuser_job_name" json:"nowUserJobName,omitempty"` // 责任人岗位
	UserName        *string `gorm:"size:100;column:user_name" json:"userName,omitempty"`              // APP扫码账号
	InventoryResult *string `gorm:"size:50;column:inventory_result" json:"inventoryResult,omitempty"` // 盘点结果

	// 系统关联字段 (2 fields) - 用于关联系统部门和用户
	DeptID        *string `gorm:"size:64;column:dept_id" json:"deptId,omitempty"`                 // 关联 sys_dept.id
	UserID        *string `gorm:"type:varchar;column:user_id" json:"userId,omitempty"`            // 关联 sys_user.id
	MachineUserID *string `gorm:"size:100;column:machine_user_id" json:"machineUserId,omitempty"` // 最后上线账号

	// 状态字段 (遵循 0=正常, 1=停用 惯例)
	Status int `gorm:"default:0;column:status" json:"status"` // 0=正常, 1=停用

	// Phase 48 组件序列号采集(D-01 / D-03 / D-05):板卡/引擎/电源/风扇/光模块作为 ops_asset 行,
	// 通过以下 4 列与父交换机/采集来源建立关联。主设备这些列保持 NULL(D-07 默认 component_type IS NULL 过滤)。
	ParentAssetID  *string `gorm:"size:64;index;column:parent_asset_id" json:"parentAssetId,omitempty"`   // 自引用 → ops_asset.id(父交换机/路由器行)
	SourceDeviceID *string `gorm:"size:64;index;column:source_device_id" json:"sourceDeviceId,omitempty"` // → sys_network_device.id(采集来源设备)
	ComponentType  *string `gorm:"size:32;index;column:component_type" json:"componentType,omitempty"`    // chassis / card / engine / power / fan / transceiver
	ComponentSlot  *string `gorm:"size:64;column:component_slot" json:"componentSlot,omitempty"`          // 槽位/接口位置(如 Slot 1 / GE1/0/24)
}

// BeforeCreate GORM钩子 - 创建前填充 UUID 主键(ID 为空时)。
// PG 下列自带 default:gen_random_uuid() 兜底原生 SQL 路径;该钩子让 GORM Create
// 应用层生成 ID,行为等价(非空 ID 直接 INSERT),同时兼容无函数式默认值的 SQLite
// (quick-260817-hfl)。
func (a *Asset) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// TableName 设置表名
func (Asset) TableName() string {
	return "ops_asset"
}
