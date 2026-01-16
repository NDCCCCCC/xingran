# 运维管理模块设计文档

**创建日期**: 2026-01-14
**版本**: 1.1
**作者**: Claude
**状态**: 待评审

---

## 1. 概述

### 1.1 模块简介

运维管理模块实现企业物理资源和网络资源的数字化管理，包括楼宇、楼层、工位、机房、专线和机房设备的管理。该模块支持多视图展示（表格、卡片、平面图）和完整的 CRUD 操作，并提供批量导入导出、文件上传下载等功能。

### 1.2 功能范围

- **楼宇管理**: 楼宇信息的增删改查，支持表格和卡片视图切换
- **楼层管理**: 楼层信息管理，与楼宇关联，支持表格和卡片视图
- **工位管理**: 工位信息管理，与楼层、用户、部门关联，支持表格和平面图视图
- **机房管理**: 机房信息管理，支持照片上传、网络设备和机房设备管理
- **专线管理**: 专线信息管理，与网络设备关联，支持完整的生命周期管理
- **机房设备管理**: 机房内的非网络设备管理（服务器、存储、UPS等）

### 1.3 技术栈

- **后端**: Go 1.24, Gin, GORM, PostgreSQL 18
- **前端**: React 19, TypeScript 5, Vite 7, Ant Design 6
- **文件存储**: 本地文件系统 / 云存储（可扩展）

---

## 2. 数据模型设计

### 2.1 楼宇表 (ops_buildings)

```go
type OpsBuilding struct {
    models.BaseModel
    Name     string         // 楼宇名称
    Code     string         // 楼宇编码（唯一）
    Address  string         // 地址
    OrgID    string         // 所属机构ID（关联sys_dept）
    Status   BuildingStatus // 状态: 0=正常, 1=停用
    Remark   string         // 备注
    OrderNum int            // 排序号
}

type BuildingStatus int
const (
    BuildingStatusNormal  BuildingStatus = 0 // 正常
    BuildingStatusStopped BuildingStatus = 1 // 停用
)
```

### 2.2 楼层表 (ops_floors)

```go
type OpsFloor struct {
    models.BaseModel
    Name       string      // 楼层名称
    FloorNo    string      // 楼层编号（如1F, 2F, B1）
    BuildingID string      // 所属楼宇ID
    Status     FloorStatus // 状态: 0=正常, 1=停用
    Remark     string      // 备注
    OrderNum   int         // 排序号
}

type FloorStatus int
const (
    FloorStatusNormal  FloorStatus = 0 // 正常
    FloorStatusStopped FloorStatus = 1 // 停用
)
```

### 2.3 工位表 (ops_workstations)

```go
type OpsWorkstation struct {
    models.BaseModel
    Code        string              // 工位编号
    Name        string              // 工位名称
    FloorID     string              // 所属楼层ID
    UserID      string              // 关联用户ID
    DeptID      string              // 关联部门ID
    PhonePoint  string              // 电话信息点
    Status      WorkstationStatus   // 状态: 0=空闲, 1=占用
    Type        WorkstationType     // 类型: 0=固定, 1=共享
    Equipment   string              // 设备配置（JSON）
    NetworkInfo string              // 网络信息
    Remark      string              // 备注
    PositionX   *int                // 平面图X坐标（可选）
    PositionY   *int                // 平面图Y坐标（可选）
}

type WorkstationStatus int
const (
    WorkstationStatusFree   WorkstationStatus = 0 // 空闲
    WorkstationStatusOccupied WorkstationStatus = 1 // 占用
)

type WorkstationType int
const (
    WorkstationTypeFixed   WorkstationType = 0 // 固定工位
    WorkstationTypeShared  WorkstationType = 1 // 共享工位
)
```

### 2.4 机房表 (ops_server_rooms)

```go
type OpsServerRoom struct {
    models.BaseModel
    Name    string     // 机房名称
    Code    string     // 机房编号（唯一）
    FloorID string     // 所在楼层ID
    Status  RoomStatus // 状态: 0=正常, 1=停用
    Remark  string     // 备注
}

type RoomStatus int
const (
    RoomStatusNormal  RoomStatus = 0 // 正常
    RoomStatusStopped RoomStatus = 1 // 停用
)
```

**关联网络设备**：机房通过 `sys_network_device` 表的 `location` 字段关联，或创建 `ops_room_network_devices` 关联表：

```go
// 机房-网络设备关联表（可选，用于更精确的关联）
type OpsRoomNetworkDevice struct {
    models.BaseModel
    RoomID       string // 机房ID
    DeviceID     string // 网络设备ID（关联sys_network_device）
    PositionU    int    // 机架位置（U）
    PositionDesc string // 位置描述（如：第1列第3个机架）
}
```

### 2.5 专线表 (ops_dedicated_lines)

```go
type OpsDedicatedLine struct {
    models.BaseModel
    Name           string             // 专线名称
    LineCode       string             // 专线编码（唯一）
    LineType       DedicatedLineType  // 专线类型
    Bandwidth      string             // 带宽（如：100M, 1G）
    ISP            string             // 运营商（电信/移动/联通等）
    SourceDeviceID string             // 源设备ID（关联sys_network_device）
    SourcePort     string             // 源端口
    DestDeviceID   string             // 目的设备ID（关联sys_network_device）
    DestPort       string             // 目的端口
    IPAddress      string             // IP地址（可选）
    SubnetMask     string             // 子网掩码（可选）
    Gateway        string             // 网关（可选）
    VLAN           string             // VLAN（可选）
    Status         LineStatus         // 状态: 0=正常, 1=故障, 2=停用
    ActivationDate *time.Time         // 启用日期
    ExpiryDate     *time.Time         // 到期日期
    ResponsibleID  string             // 负责人ID（关联sys_user）
    ContractNo     string             // 合同编号
    MonthlyFee     float64            // 月租费用
    Remark         string             // 备注
}

type DedicatedLineType string
const (
    LineTypeInternet    DedicatedLineType = "internet"    // 互联网专线
    LineTypeMPLS        DedicatedLineType = "mpls"        // MPLS VPN
    LineTypeLeasedLine  DedicatedLineType = "leased_line" // 租用专线
    LineTypeFiber       DedicatedLineType = "fiber"       // 光纤专线
)

type LineStatus int
const (
    LineStatusNormal   LineStatus = 0 // 正常
    LineStatusFault    LineStatus = 1 // 故障
    LineStatusDisabled LineStatus = 2 // 停用
)
```

### 2.6 机房设备表 (ops_room_devices)

```go
type OpsRoomDevice struct {
    models.BaseModel
    Name            string            // 设备名称
    DeviceCode      string            // 设备编码（唯一）
    DeviceType      RoomDeviceType    // 设备类型
    Model           string            // 设备型号
    Vendor          string            // 厂商
    SerialNumber    string            // 序列号
    RoomID          string            // 所在机房ID（关联ops_server_rooms）
    PositionU       int               // 机架位置（U）
    PositionDesc    string            // 位置描述（如：第1列第3个机架）
    AssetNumber     string            // 资产编号
    PurchaseDate    *time.Time        // 购买日期
    WarrantyDate    *time.Time        // 保修到期日期
    Status          RoomDeviceStatus  // 状态: 0=正常, 1=故障, 2=报废
    ResponsibleID   string            // 负责人ID（关联sys_user）
    PowerConsumption int              // 功耗（瓦）
    Remark          string            // 备注
}

type RoomDeviceType string
const (
    DeviceTypeServer   RoomDeviceType = "server"   // 服务器
    DeviceTypeStorage  RoomDeviceType = "storage"  // 存储设备
    DeviceTypeUPS      RoomDeviceType = "ups"      // UPS不间断电源
    DeviceTypePDU      RoomDeviceType = "pdu"      // PDU电源分配
    DeviceTypeFirewall RoomDeviceType = "firewall" // 硬件防火墙
    DeviceTypeKVM      RoomDeviceType = "kvm"      // KVM切换器
    DeviceTypeCabinet  RoomDeviceType = "cabinet"  // 机柜
    DeviceTypeOther    RoomDeviceType = "other"    // 其他设备
)

type RoomDeviceStatus int
const (
    RoomDeviceStatusNormal   RoomDeviceStatus = 0 // 正常
    RoomDeviceStatusFault    RoomDeviceStatus = 1 // 故障
    RoomDeviceStatusScrapped RoomDeviceStatus = 2 // 报废
)
```

### 2.7 文件管理表

#### 通用文件表 (sys_files)

```go
type SysFile struct {
    models.BaseModel
    FileName    string     // 原始文件名
    FileSize    int64      // 文件大小(字节)
    FileType    string     // 文件类型(MIME)
    Extension   string     // 文件扩展名
    StoragePath string     // 存储路径
    FileHash    string     // 文件哈希(SHA256)用于去重
    UploaderID  string     // 上传者ID
    BusinessType string    // 业务类型(room_photo/import/export等)
    IsDeleted   bool       // 是否已删除(软删除)
    DeleteTime  *time.Time // 删除时间
}
```

#### 机房照片关联表 (ops_room_photos)

```go
type OpsRoomPhoto struct {
    models.BaseModel
    RoomID      string // 机房ID
    FileID      string // 文件ID(关联sys_files)
    SortOrder   int    // 排序号
    IsPrimary   bool   // 是否为主图
    Description string // 照片描述
}
```

#### 文件访问日志表 (sys_file_access_logs)

```go
type SysFileAccessLog struct {
    models.BaseTimeLine
    FileID     string // 文件ID
    ActionType string // 操作类型(upload/download/delete/view)
    UserID     string // 操作者ID
    UserName   string // 操作者姓名
    IPAddress  string // IP地址
    UserAgent  string // 浏览器信息
}
```

---

## 3. API 接口设计

### 3.1 楼宇管理 API

```
POST /api/v1/ops/buildings              # 创建楼宇
POST /api/v1/ops/buildings/list         # 查询楼宇列表（分页、搜索）
POST /api/v1/ops/buildings/:id          # 获取楼宇详情
POST /api/v1/ops/buildings/:id/update   # 更新楼宇
POST /api/v1/ops/buildings/:id/delete   # 删除楼宇
POST /api/v1/ops/buildings/batch        # 批量操作
POST /api/v1/ops/buildings/import       # 批量导入
POST /api/v1/ops/buildings/export       # 导出
POST /api/v1/ops/buildings/template     # 下载导入模板
```

### 3.2 楼层管理 API

```
POST /api/v1/ops/flools                # 创建楼层
POST /api/v1/ops/floors/list           # 查询楼层列表（支持按楼宇筛选）
POST /api/v1/ops/floors/tree           # 获取楼层树形结构（楼宇→楼层）
POST /api/v1/ops/floors/:id            # 详情
POST /api/v1/ops/floors/:id/update     # 更新
POST /api/v1/ops/floors/:id/delete     # 删除
POST /api/v1/ops/floors/batch          # 批量操作
POST /api/v1/ops/floors/import         # 批量导入
POST /api/v1/ops/floors/export         # 导出
POST /api/v1/ops/floors/template       # 下载导入模板
```

### 3.3 工位管理 API

```
POST /api/v1/ops/workstations          # 创建工位
POST /api/v1/ops/workstations/list     # 查询工位列表（支持按楼层/部门/用户筛选）
POST /api/v1/ops/workstations/:id      # 详情
POST /api/v1/ops/workstations/:id/update  # 更新
POST /api/v1/ops/workstations/:id/delete  # 删除
POST /api/v1/ops/workstations/batch       # 批量操作
POST /api/v1/ops/workstations/import      # 批量导入
POST /api/v1/ops/workstations/export      # 导出
POST /api/v1/ops/workstations/template    # 下载导入模板
POST /api/v1/ops/workstations/position    # 更新平面图位置
```

### 3.4 机房管理 API

```
POST /api/v1/ops/server-rooms                # 创建机房
POST /api/v1/ops/server-rooms/list           # 查询机房列表（支持按楼层筛选）
POST /api/v1/ops/server-rooms/:id            # 详情
POST /api/v1/ops/server-rooms/:id/update     # 更新
POST /api/v1/ops/server-rooms/:id/delete     # 删除
POST /api/v1/ops/server-rooms/batch          # 批量操作
POST /api/v1/ops/server-rooms/import         # 批量导入
POST /api/v1/ops/server-rooms/export         # 导出
POST /api/v1/ops/server-rooms/template       # 下载导入模板
POST /api/v1/ops/server-rooms/:id/photos/upload       # 上传机房照片
POST /api/v1/ops/server-rooms/:id/photos/:photoId    # 删除照片
POST /api/v1/ops/server-rooms/:id/photos/:photoId/primary # 设置主图
POST /api/v1/ops/server-rooms/:id/network-devices     # 关联网络设备
POST /api/v1/ops/server-rooms/:id/network-devices/:deviceId # 取消关联网络设备
```

### 3.5 专线管理 API

```
POST /api/v1/ops/dedicated-lines          # 创建专线
POST /api/v1/ops/dedicated-lines/list     # 查询专线列表（支持按设备/状态/类型筛选）
POST /api/v1/ops/dedicated-lines/:id      # 详情
POST /api/v1/ops/dedicated-lines/:id/update   # 更新
POST /api/v1/ops/dedicated-lines/:id/delete   # 删除
POST /api/v1/ops/dedicated-lines/batch        # 批量操作
POST /api/v1/ops/dedicated-lines/import       # 批量导入
POST /api/v1/ops/dedicated-lines/export       # 导出
POST /api/v1/ops/dedicated-lines/template     # 下载导入模板
POST /api/v1/ops/dedicated-lines/:id/status   # 更新状态
POST /api/v1/ops/dedicated-lines/expiring     # 查询即将到期的专线
```

### 3.6 机房设备管理 API

```
POST /api/v1/ops/room-devices           # 创建机房设备
POST /api/v1/ops/room-devices/list      # 查询机房设备列表（支持按机房/类型/状态筛选）
POST /api/v1/ops/room-devices/:id       # 详情
POST /api/v1/ops/room-devices/:id/update    # 更新
POST /api/v1/ops/room-devices/:id/delete    # 删除
POST /api/v1/ops/room-devices/batch         # 批量操作
POST /api/v1/ops/room-devices/import        # 批量导入
POST /api/v1/ops/room-devices/export        # 导出
POST /api/v1/ops/room-devices/template      # 下载导入模板
POST /api/v1/ops/room-devices/:id/status    # 更新状态
POST /api/v1/ops/room-devices/warning       # 查询即将过保设备
POST /api/v1/ops/room-devices/statistics    # 设备统计（按机房/类型）
```

### 3.7 文件管理 API

```
POST /api/v1/ops/common/upload         # 通用文件上传
POST /api/v1/ops/common/upload/photo   # 图片上传专用（压缩、格式转换）
POST /api/v1/ops/common/download/:id   # 文件下载
POST /api/v1/ops/server-rooms/:id/photos/upload    # 上传机房照片
POST /api/v1/ops/server-rooms/:id/photos/:photoId # 删除照片
POST /api/v1/ops/server-rooms/:id/photos/:photoId/primary # 设置主图
```

---

## 4. 前端设计

### 4.1 目录结构

```
ruoyi-react-frontend/src/
├── pages/
│   └── operations/              # 运维管理模块
│       ├── buildings/           # 楼宇管理
│       │   ├── index.tsx       # 列表页
│       │   ├── components/     # 组件
│       │   │   ├── BuildingForm.tsx      # 新增/编辑表单
│       │   │   ├── BuildingImport.tsx    # 导入弹窗
│       │   │   ├── BuildingCard.tsx      # 卡片视图组件
│       │   │   └── BuildingDetail.tsx    # 详情抽屉
│       │   └── types.ts        # 类型定义
│       ├── floors/              # 楼层管理
│       │   ├── index.tsx
│       │   ├── components/
│       │   │   ├── FloorForm.tsx
│       │   │   ├── FloorImport.tsx
│       │   │   ├── FloorCard.tsx         # 卡片视图组件
│       │   │   └── FloorTree.tsx         # 楼宇-楼层树形组件
│       │   └── types.ts
│       ├── workstations/        # 工位管理
│       │   ├── index.tsx
│       │   ├── components/
│       │   │   ├── WorkstationForm.tsx
│       │   │   ├── WorkstationImport.tsx
│       │   │   ├── FloorPlanView.tsx     # 平面图视图组件
│       │   │   ├── WorkstationMarker.tsx # 工位标记组件
│       │   │   └── RoomMarker.tsx        # 机房标记组件
│       │   └── types.ts
│       ├── server-rooms/        # 机房管理
│       │   ├── index.tsx
│       │   ├── components/
│       │   │   ├── RoomForm.tsx
│       │   │   ├── RoomImport.tsx
│       │   │   ├── PhotoUpload.tsx       # 照片上传组件
│       │   │   ├── PhotoPreview.tsx      # 照片预览组件
│       │   │   ├── NetworkDeviceSelector.tsx  # 网络设备选择器
│       │   │   └── RoomDeviceList.tsx    # 机房设备列表
│       │   └── types.ts
│       ├── dedicated-lines/     # 专线管理
│       │   ├── index.tsx
│       │   ├── components/
│       │   │   ├── LineForm.tsx
│       │   │   ├── LineImport.tsx
│       │   │   ├── DeviceSelector.tsx    # 设备选择器（支持源设备和目的设备）
│       │   │   └── LineTopology.tsx     # 专线拓扑图组件
│       │   └── types.ts
│       └── room-devices/        # 机房设备管理
│           ├── index.tsx
│           ├── components/
│           │   ├── DeviceForm.tsx
│           │   ├── DeviceImport.tsx
│           │   ├── DeviceRackView.tsx    # 机架视图组件
│           │   └── WarrantyWarning.tsx   # 保修预警组件
│           └── types.ts
├── services/
│   └── operations/              # 运维管理API
│       ├── buildings.ts
│       ├── floors.ts
│       ├── workstations.ts
│       ├── server-rooms.ts
│       ├── dedicated-lines.ts
│       ├── room-devices.ts
│       └── files.ts            # 文件上传下载API
└── types/
     └── operations.ts           # 运维管理通用类型
```

### 4.2 楼宇管理页面

**视图模式切换**：
- 表格视图：标准 Ant Design Table
- 卡片视图：Grid 布局，每个卡片显示楼宇信息

**页面组件**：
```
├── 统计卡片（总楼宇数、正常、停用）
├── 搜索表单（楼宇名称、编码、机构、状态）
├── 操作按钮（新增、批量删除、导入、导出）
├── 视图切换（表格/卡片）
└── 内容区
```

### 4.3 楼层管理页面

**视图模式切换**：
- 表格视图：标准 Table
- 卡片视图：按楼宇分组展示楼层列表

**页面组件**：
```
├── 统计卡片（总楼层、楼层分布）
├── 搜索表单（楼层名称、楼宇筛选、状态）
├── 操作按钮（新增、批量删除、导入、导出）
├── 视图切换（表格/卡片）
└── 内容区
```

### 4.4 工位管理页面

**视图模式切换**：
- 表格视图：标准 Table
- 平面图视图：可视化平面图（Canvas/SVG）

**平面图功能**：
```
┌─────────────────────────────────────────────┐
│  楼层：3F                                   │
│  ┌──────┐  ┌──────┐  ┌──────────────┐     │
│  │ 工位 │  │ 工位 │  │   机房 A     │     │
│  │ 001  │  │ 002  │  │  [🖼️照片预览] │     │
│  │ 🟢空闲│  │ 🔴占用│  │              │     │
│  └──────┘  └──────┘  └──────────────┘     │
│  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐  │
│  │ 工位 │  │ 工位 │  │ 工位 │  │ 工位 │  │
│  │ 003  │  │ 004  │  │ 005  │  │ 006  │  │
│  │ 🟢空闲│  │ 🟢空闲│  │ 🔴占用│  │ 🟢空闲│  │
│  └──────┘  └──────┘  └──────┘  └──────┘  │
└─────────────────────────────────────────────┘
```

**平面图元素**：
- 工位：方块显示，颜色区分状态（绿色=空闲，红色=占用，灰色=共享）
- 机房：矩形区域，显示照片预览
- 交互：点击查看详情，拖拽调整位置

**图例**：
```
🟢 空闲工位
🔴 占用工位
⚪ 共享工位
🟦 机房区域
```

### 4.5 机房管理页面

**页面组件**：
```
├── 统计卡片（总机房数、正常、停用）
├── 搜索表单（机房名称、楼层、状态）
├── 操作按钮（新增、批量删除、导入、导出）
└── 表格（包含照片预览列）
```

**照片管理**：
- 多张照片上传
- 照片预览（悬停显示大图）
- 设置主图
- 照片排序
- 照片删除

**网络设备关联**：
- 在机房详情页中关联现有网络设备
- 支持设置机架位置（U）
- 查看机房内的所有网络设备列表

### 4.6 专线管理页面

**页面组件**：
```
├── 统计卡片（总专线数、正常、故障、停用）
├── 搜索表单（专线名称、编码、设备、状态、类型）
├── 操作按钮（新增、批量删除、导入、导出）
├── 到期预警（即将到期30天的专线）
└── 表格
```

**表格列**：
- 专线名称、编码
- 专线类型、运营商、带宽
- 源设备/端口、目的设备/端口
- IP地址、VLAN
- 状态、启用日期、到期日期
- 负责人、合同编号、月租费用
- 操作（编辑、删除、查看拓扑）

**拓扑视图**：
- 可视化展示专线连接关系
- 显示源设备 → 目的设备的连接
- 支持点击设备跳转到设备详情

### 4.7 机房设备管理页面

**页面组件**：
```
├── 统计卡片（总设备数、正常、故障、报废）
├── 搜索表单（设备名称、编码、机房、类型、状态）
├── 操作按钮（新增、批量删除、导入、导出）
├── 保修预警（即将过保30天的设备）
└── 表格/机架视图切换
```

**表格列**：
- 设备名称、编码、类型
- 型号、厂商、序列号
- 所在机房、机架位置
- 资产编号、购买日期、保修到期日期
- 状态、负责人、功耗
- 操作（编辑、删除）

**机架视图**：
- 以机柜形式展示设备
- 显示每个设备的U位置
- 拖拽调整设备位置
- 点击设备查看详情

---

## 5. 后端设计

### 5.1 代码结构

```
internal/
├── models/
│   └── operations/              # 运维管理模型
│       ├── building.go         # 楼宇模型
│       ├── floor.go            # 楼层模型
│       ├── workstation.go      # 工位模型
│       ├── server_room.go      # 机房模型
│       ├── room_photo.go       # 机房照片关联模型
│       ├── room_network_device.go  # 机房-网络设备关联模型
│       ├── dedicated_line.go   # 专线模型
│       └── room_device.go      # 机房设备模型
│
├── services/                    # 业务逻辑层
│   └── operations/
│       ├── building_service.go
│       ├── floor_service.go
│       ├── workstation_service.go
│       ├── server_room_service.go
│       ├── file_service.go     # 文件管理服务
│       ├── dedicated_line_service.go
│       └── room_device_service.go
│
├── api/v1/                      # HTTP 处理器层
│   └── operations/
│       ├── building_handler.go
│       ├── floor_handler.go
│       ├── workstation_handler.go
│       ├── server_room_handler.go
│       ├── file_handler.go
│       ├── dedicated_line_handler.go
│       └── room_device_handler.go
│
└── router.go                    # 路由注册
```

### 5.2 Service 层接口

```go
type BuildingService interface {
    Create(ctx context.Context, building *OpsBuilding) error
    Update(ctx context.Context, building *OpsBuilding) error
    Delete(ctx context.Context, id string) error
    GetByID(ctx context.Context, id string) (*OpsBuilding, error)
    List(ctx context.Context, params ListParams) (*PageResult, error)
    BatchDelete(ctx context.Context, ids []string) error
    Export(ctx context.Context, params ListParams) ([]byte, error)
    Import(ctx context.Context, file *multipart.FileHeader) (*ImportResult, error)
}

type FileService interface {
    Upload(ctx context.Context, file *multipart.FileHeader, businessType string) (*SysFile, error)
    Download(ctx context.Context, fileID string) (string, error)
    Delete(ctx context.Context, fileID string) error
    BatchUpload(ctx context.Context, files []*multipart.FileHeader) ([]*SysFile, error)
}
```

### 5.3 路由注册

```go
// 在 router.go 中添加
opsGroup := v1.Group("/ops")
{
    // 楼宇管理
    buildings := opsGroup.Group("/buildings")
    {
        buildings.POST("", buildingHandler.Create)
        buildings.POST("/list", buildingHandler.List)
        buildings.POST("/:id", buildingHandler.GetByID)
        buildings.POST("/:id/update", buildingHandler.Update)
        buildings.POST("/:id/delete", buildingHandler.Delete)
        buildings.POST("/batch", buildingHandler.BatchOperation)
        buildings.POST("/import", buildingHandler.Import)
        buildings.POST("/export", buildingHandler.Export)
        buildings.POST("/template", buildingHandler.DownloadTemplate)
    }

    // 楼层管理
    floors := opsGroup.Group("/floors")
    {
        floors.POST("", floorHandler.Create)
        floors.POST("/list", floorHandler.List)
        floors.POST("/tree", floorHandler.GetTree)
        floors.POST("/:id", floorHandler.GetByID)
        floors.POST("/:id/update", floorHandler.Update)
        floors.POST("/:id/delete", floorHandler.Delete)
        floors.POST("/batch", floorHandler.BatchOperation)
        floors.POST("/import", floorHandler.Import)
        floors.POST("/export", floorHandler.Export)
        floors.POST("/template", floorHandler.DownloadTemplate)
    }

    // 工位管理
    workstations := opsGroup.Group("/workstations")
    {
        workstations.POST("", workstationHandler.Create)
        workstations.POST("/list", workstationHandler.List)
        workstations.POST("/:id", workstationHandler.GetByID)
        workstations.POST("/:id/update", workstationHandler.Update)
        workstations.POST("/:id/delete", workstationHandler.Delete)
        workstations.POST("/batch", workstationHandler.BatchOperation)
        workstations.POST("/import", workstationHandler.Import)
        workstations.POST("/export", workstationHandler.Export)
        workstations.POST("/template", workstationHandler.DownloadTemplate)
        workstations.POST("/position", workstationHandler.UpdatePosition)
    }

    // 机房管理
    serverRooms := opsGroup.Group("/server-rooms")
    {
        serverRooms.POST("", serverRoomHandler.Create)
        serverRooms.POST("/list", serverRoomHandler.List)
        serverRooms.POST("/:id", serverRoomHandler.GetByID)
        serverRooms.POST("/:id/update", serverRoomHandler.Update)
        serverRooms.POST("/:id/delete", serverRoomHandler.Delete)
        serverRooms.POST("/batch", serverRoomHandler.BatchOperation)
        serverRooms.POST("/import", serverRoomHandler.Import)
        serverRooms.POST("/export", serverRoomHandler.Export)
        serverRooms.POST("/template", serverRoomHandler.DownloadTemplate)
        serverRooms.POST("/:id/photos/upload", serverRoomHandler.UploadPhoto)
        serverRooms.POST("/:id/photos/:photoId", serverRoomHandler.DeletePhoto)
        serverRooms.POST("/:id/photos/:photoId/primary", serverRoomHandler.SetPrimaryPhoto)
        serverRooms.POST("/:id/network-devices", serverRoomHandler.AddNetworkDevice)
        serverRooms.POST("/:id/network-devices/:deviceId", serverRoomHandler.RemoveNetworkDevice)
    }

    // 专线管理
    dedicatedLines := opsGroup.Group("/dedicated-lines")
    {
        dedicatedLines.POST("", dedicatedLineHandler.Create)
        dedicatedLines.POST("/list", dedicatedLineHandler.List)
        dedicatedLines.POST("/:id", dedicatedLineHandler.GetByID)
        dedicatedLines.POST("/:id/update", dedicatedLineHandler.Update)
        dedicatedLines.POST("/:id/delete", dedicatedLineHandler.Delete)
        dedicatedLines.POST("/batch", dedicatedLineHandler.BatchOperation)
        dedicatedLines.POST("/import", dedicatedLineHandler.Import)
        dedicatedLines.POST("/export", dedicatedLineHandler.Export)
        dedicatedLines.POST("/template", dedicatedLineHandler.DownloadTemplate)
        dedicatedLines.POST("/:id/status", dedicatedLineHandler.UpdateStatus)
        dedicatedLines.POST("/expiring", dedicatedLineHandler.GetExpiringLines)
    }

    // 机房设备管理
    roomDevices := opsGroup.Group("/room-devices")
    {
        roomDevices.POST("", roomDeviceHandler.Create)
        roomDevices.POST("/list", roomDeviceHandler.List)
        roomDevices.POST("/:id", roomDeviceHandler.GetByID)
        roomDevices.POST("/:id/update", roomDeviceHandler.Update)
        roomDevices.POST("/:id/delete", roomDeviceHandler.Delete)
        roomDevices.POST("/batch", roomDeviceHandler.BatchOperation)
        roomDevices.POST("/import", roomDeviceHandler.Import)
        roomDevices.POST("/export", roomDeviceHandler.Export)
        roomDevices.POST("/template", roomDeviceHandler.DownloadTemplate)
        roomDevices.POST("/:id/status", roomDeviceHandler.UpdateStatus)
        roomDevices.POST("/warning", roomDeviceHandler.GetWarrantyWarning)
        roomDevices.POST("/statistics", roomDeviceHandler.GetStatistics)
    }

    // 通用文件管理
    common := opsGroup.Group("/common")
    {
        common.POST("/upload", fileHandler.Upload)
        common.POST("/upload/photo", fileHandler.UploadPhoto)
        common.POST("/download/:id", fileHandler.Download)
    }
}
```

---

## 6. 数据验证

### 6.1 前端表单验证

**楼宇验证规则**：
```typescript
{
  name: [
    { required: true, message: '请输入楼宇名称' },
    { max: 50, message: '楼宇名称不能超过50个字符' },
    { pattern: /^[\u4e00-\u9fa5a-zA-Z0-9]+$/, message: '只能输入中文、字母、数字' }
  ],
  code: [
    { required: true, message: '请输入楼宇编码' },
    { max: 20, message: '楼宇编码不能超过20个字符' },
    { pattern: /^[A-Z0-9_-]+$/, message: '只能输入大写字母、数字、下划线、横线' }
  ],
  address: [
    { max: 200, message: '地址不能超过200个字符' }
  ],
  orgId: [
    { required: true, message: '请选择所属机构' }
  ]
}
```

**楼层验证规则**：
```typescript
{
  name: [
    { required: true, message: '请输入楼层名称' },
    { max: 50, message: '楼层名称不能超过50个字符' }
  ],
  floorNo: [
    { required: true, message: '请输入楼层编号' },
    { pattern: /^(B\d+|[1-9]\d*F|1F)$/, message: '楼层编号格式错误，如：1F, 2F, B1' }
  ],
  buildingId: [
    { required: true, message: '请选择所属楼宇' }
  ]
}
```

**工位验证规则**：
```typescript
{
  code: [
    { required: true, message: '请输入工位编号' },
    { max: 30, message: '工位编号不能超过30个字符' }
  ],
  name: [
    { required: true, message: '请输入工位名称' },
    { max: 50, message: '工位名称不能超过50个字符' }
  ],
  floorId: [
    { required: true, message: '请选择所属楼层' }
  ],
  phonePoint: [
    { pattern: /^\d{4}$/, message: '电话信息点必须是4位数字' }
  ]
}
```

**专线验证规则**：
```typescript
{
  name: [
    { required: true, message: '请输入专线名称' },
    { max: 100, message: '专线名称不能超过100个字符' }
  ],
  lineCode: [
    { required: true, message: '请输入专线编码' },
    { max: 50, message: '专线编码不能超过50个字符' }
  ],
  lineType: [
    { required: true, message: '请选择专线类型' }
  ],
  bandwidth: [
    { required: true, message: '请输入带宽' },
    { pattern: /^\d+[MGMG]$/, message: '带宽格式错误，如：100M, 1G' }
  ],
  isp: [
    { required: true, message: '请输入运营商' }
  ],
  sourceDeviceId: [
    { required: true, message: '请选择源设备' }
  ],
  sourcePort: [
    { required: true, message: '请输入源端口' }
  ],
  destDeviceId: [
    { required: true, message: '请选择目的设备' }
  ],
  destPort: [
    { required: true, message: '请输入目的端口' }
  ],
  monthlyFee: [
    { type: 'number', min: 0, message: '月租费用不能为负数' }
  ]
}
```

**机房设备验证规则**：
```typescript
{
  name: [
    { required: true, message: '请输入设备名称' },
    { max: 100, message: '设备名称不能超过100个字符' }
  ],
  deviceCode: [
    { required: true, message: '请输入设备编码' },
    { max: 50, message: '设备编码不能超过50个字符' }
  ],
  deviceType: [
    { required: true, message: '请选择设备类型' }
  ],
  roomId: [
    { required: true, message: '请选择所在机房' }
  ],
  positionU: [
    { type: 'number', min: 1, max: 48, message: '机架位置必须在1-48U之间' }
  ],
  powerConsumption: [
    { type: 'number', min: 0, message: '功耗不能为负数' }
  ]
}
```

### 6.2 后端请求验证

```go
type CreateBuildingRequest struct {
    Name   string `json:"name" binding:"required,max=50"`
    Code   string `json:"code" binding:"required,max=20,alphanum"`
    Address string `json:"address" binding:"max=200"`
    OrgID  string `json:"orgId" binding:"required,uuid"`
    Remark string `json:"remark" binding:"max=500"`
}

type CreateFloorRequest struct {
    Name       string `json:"name" binding:"required,max=50"`
    FloorNo    string `json:"floorNo" binding:"required,max=10"`
    BuildingID string `json:"buildingId" binding:"required,uuid"`
    Remark     string `json:"remark" binding:"max=500"`
}

type CreateWorkstationRequest struct {
    Code        string `json:"code" binding:"required,max=30"`
    Name        string `json:"name" binding:"required,max=50"`
    FloorID     string `json:"floorId" binding:"required,uuid"`
    UserID      string `json:"userId" binding:"omitempty,uuid"`
    DeptID      string `json:"deptId" binding:"omitempty,uuid"`
    PhonePoint  string `json:"phonePoint" binding:"omitempty,max=20"`
    Type        int    `json:"type" binding:"required,min=0,max=1"`
    Equipment   string `json:"equipment" binding:"max=1000"`
    NetworkInfo string `json:"networkInfo" binding:"max=500"`
    Remark      string `json:"remark" binding:"max=500"`
    PositionX   *int   `json:"positionX"`
    PositionY   *int   `json:"positionY"`
}

type CreateDedicatedLineRequest struct {
    Name           string             `json:"name" binding:"required,max=100"`
    LineCode       string             `json:"lineCode" binding:"required,max=50"`
    LineType       DedicatedLineType  `json:"lineType" binding:"required"`
    Bandwidth      string             `json:"bandwidth" binding:"required,max=20"`
    ISP            string             `json:"isp" binding:"required,max=50"`
    SourceDeviceID string             `json:"sourceDeviceId" binding:"required,uuid"`
    SourcePort     string             `json:"sourcePort" binding:"required,max=50"`
    DestDeviceID   string             `json:"destDeviceId" binding:"required,uuid"`
    DestPort       string             `json:"destPort" binding:"required,max=50"`
    IPAddress      string             `json:"ipAddress" binding:"omitempty,max=45"`
    SubnetMask     string             `json:"subnetMask" binding:"omitempty,max=45"`
    Gateway        string             `json:"gateway" binding:"omitempty,max=45"`
    VLAN           string             `json:"vlan" binding:"omitempty,max=10"`
    ActivationDate *time.Time         `json:"activationDate"`
    ExpiryDate     *time.Time         `json:"expiryDate"`
    ResponsibleID  string             `json:"responsibleId" binding:"omitempty,uuid"`
    ContractNo     string             `json:"contractNo" binding:"max=50"`
    MonthlyFee     float64            `json:"monthlyFee" binding:"min=0"`
    Remark         string             `json:"remark" binding:"max=500"`
}

type CreateRoomDeviceRequest struct {
    Name            string          `json:"name" binding:"required,max=100"`
    DeviceCode      string          `json:"deviceCode" binding:"required,max=50"`
    DeviceType      RoomDeviceType  `json:"deviceType" binding:"required"`
    Model           string          `json:"model" binding:"max=100"`
    Vendor          string          `json:"vendor" binding:"max=50"`
    SerialNumber    string          `json:"serialNumber" binding:"max=100"`
    RoomID          string          `json:"roomId" binding:"required,uuid"`
    PositionU       int             `json:"positionU" binding:"min=1,max=48"`
    PositionDesc    string          `json:"positionDesc" binding:"max=200"`
    AssetNumber     string          `json:"assetNumber" binding:"max=50"`
    PurchaseDate    *time.Time      `json:"purchaseDate"`
    WarrantyDate    *time.Time      `json:"warrantyDate"`
    ResponsibleID   string          `json:"responsibleId" binding:"omitempty,uuid"`
    PowerConsumption int            `json:"powerConsumption" binding:"min=0"`
    Remark          string          `json:"remark" binding:"max=500"`
}
```

---

## 7. 文件上传处理

### 7.1 文件类型支持

| 类型 | 支持格式 | 最大大小 | 用途 |
|------|---------|---------|------|
| 图片 | .jpg, .jpeg, .png | 5MB | 机房照片 |
| Excel | .xlsx, .xls | 10MB | 数据导入导出 |

### 7.2 图片处理

- 自动压缩：超过 2MB 的图片自动压缩
- 格式转换：统一转换为 JPEG 格式
- 缩略图生成：生成 200x200 缩略图用于列表展示
- 去重机制：使用 SHA256 哈希值去重

### 7.3 Excel 处理

- 模板下载：提供标准导入模板
- 数据验证：导入时验证字段类型和格式
- 错误提示：返回具体行列和错误原因
- 事务处理：全部成功或全部回滚

---

## 8. 测试计划

### 8.1 单元测试

**Service 层测试**：
- 创建/更新/删除功能
- 数据验证逻辑
- 关联关系验证
- 唯一性约束检查

### 8.2 集成测试

**API 测试**：
- 所有接口的请求响应
- 错误处理
- 权限验证

### 8.3 前端测试

**组件测试**：
- 表单验证
- 视图切换
- 交互功能

**E2E 测试**：
- 完整业务流程
- 导入导出功能

### 8.4 手动测试清单

**楼宇管理**：
- [ ] 创建楼宇（正常流程）
- [ ] 创建楼宇（重复编码）
- [ ] 编辑楼宇
- [ ] 删除楼宇（单个）
- [ ] 批量删除楼宇
- [ ] 导入楼宇数据
- [ ] 导出楼宇数据
- [ ] 切换表格/卡片视图
- [ ] 筛选和搜索功能

**楼层管理**：
- [ ] 创建楼层
- [ ] 关联到楼宇
- [ ] 查看楼宇下的楼层列表（卡片模式）
- [ ] 导入导出功能

**工位管理**：
- [ ] 创建工位
- [ ] 关联用户和部门
- [ ] 平面图模式显示
- [ ] 平面图中工位拖拽
- [ ] 平面图中机房显示
- [ ] 工位状态切换（空闲/占用）

**机房管理**：
- [ ] 创建机房
- [ ] 上传照片
- [ ] 照片预览
- [ ] 照片设为主图
- [ ] 照片排序
- [ ] 关联网络设备
- [ ] 查看机房内的网络设备列表

**专线管理**：
- [ ] 创建专线
- [ ] 关联源设备和目的设备
- [ ] 查看专线拓扑图
- [ ] 到期预警功能
- [ ] 导入导出功能

**机房设备管理**：
- [ ] 创建机房设备
- [ ] 设置机架位置
- [ ] 机架视图显示
- [ ] 保修预警功能
- [ ] 导入导出功能

---

## 9. 实施计划

### 9.1 开发阶段

1. **阶段一：后端基础**（预计 3 天）
   - 创建数据模型和数据库迁移
   - 实现 Service 层
   - 实现 API Handler
   - 单元测试

2. **阶段二：前端基础**（预计 4 天）
   - 创建页面组件
   - 实现 API 调用层
   - 实现表格视图
   - 表单验证

3. **阶段三：特色功能**（预计 4 天）
   - 卡片视图（楼宇、楼层）
   - 平面图视图（工位）
   - 专线拓扑图
   - 机房机架视图
   - 文件上传（机房照片）
   - 导入导出功能

4. **阶段四：测试与优化**（预计 1 天）
   - 集成测试
   - Bug 修复
   - 性能优化

### 9.2 里程碑

| 里程碑 | 交付物 | 完成标准 |
|--------|--------|---------|
| M1 | 后端 API | 所有 API 可用并通过单元测试 |
| M2 | 基础页面 | CRUD 功能完整 |
| M3 | 特色功能 | 多视图和文件管理完成 |
| M4 | 测试完成 | 通过所有测试用例 |

---

## 10. 附录

### 10.1 路由配置

运维管理作为一级菜单添加到系统菜单配置中：

```
运维管理 (operations)
├── 楼宇管理  (/operations/buildings)
├── 楼层管理  (/operations/floors)
├── 工位管理  (/operations/workstations)
├── 机房管理  (/operations/server-rooms)
├── 专线管理  (/operations/dedicated-lines)
└── 机房设备管理  (/operations/room-devices)
```

### 10.2 权限配置

为每个模块配置相应的权限标识：

```
operations:buildings:list     # 楼宇查询
operations:buildings:create   # 楼宇创建
operations:buildings:update   # 楼宇更新
operations:buildings:delete   # 楼宇删除
operations:buildings:export   # 楼宇导出
operations:buildings:import   # 楼宇导入

operations:floors:list        # 楼层查询
operations:floors:create      # 楼层创建
operations:floors:update      # 楼层更新
operations:floors:delete      # 楼层删除
operations:floors:export      # 楼层导出
operations:floors:import      # 楼层导入

operations:workstations:list  # 工位查询
operations:workstations:create # 工位创建
operations:workstations:update # 工位更新
operations:workstations:delete # 工位删除
operations:workstations:export # 工位导出
operations:workstations:import # 工位导入

operations:server-rooms:list        # 机房查询
operations:server-rooms:create      # 机房创建
operations:server-rooms:update      # 机房更新
operations:server-rooms:delete      # 机房删除
operations:server-rooms:export      # 机房导出
operations:server-rooms:import      # 机房导入
operations:server-rooms:link-device # 关联网络设备

operations:dedicated-lines:list     # 专线查询
operations:dedicated-lines:create   # 专线创建
operations:dedicated-lines:update   # 专线更新
operations:dedicated-lines:delete   # 专线删除
operations:dedicated-lines:export   # 专线导出
operations:dedicated-lines:import   # 专线导入

operations:room-devices:list        # 机房设备查询
operations:room-devices:create      # 机房设备创建
operations:room-devices:update      # 机房设备更新
operations:room-devices:delete      # 机房设备删除
operations:room-devices:export      # 机房设备导出
operations:room-devices:import      # 机房设备导入
```

### 10.3 数据库迁移文件

需要在 `migrations/` 目录下创建以下迁移文件：

- `001_create_ops_buildings_table.up.sql`
- `002_create_ops_floors_table.up.sql`
- `003_create_ops_workstations_table.up.sql`
- `004_create_ops_server_rooms_table.up.sql`
- `005_create_ops_room_photos_table.up.sql`
- `006_create_ops_room_network_devices_table.up.sql`
- `007_create_ops_dedicated_lines_table.up.sql`
- `008_create_ops_room_devices_table.up.sql`
- `009_create_sys_files_table.up.sql`
- `010_create_sys_file_access_logs_table.up.sql`

---

**文档状态**: 待评审
**下一步**: 评审通过后开始实施
