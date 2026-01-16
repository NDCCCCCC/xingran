# 运维管理模块实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标:** 实现企业物理资源和网络资源的数字化管理，包括楼宇、楼层、工位、机房、专线和机房设备的管理

**架构:** 采用 Service-Handler 分层架构，GORM 作为 ORM，遵循项目现有的开发规范。所有状态值遵循 0=启用/正常，1=禁用/停用 的统一规则。

**技术栈:** Go 1.24, Gin, GORM, PostgreSQL 18, React 19, TypeScript 5, Vite 7, Ant Design 6

---

## 阶段一：后端基础 - 数据模型

### Task 1: 创建楼宇模型

**文件:**
- 创建: `internal/models/operations/building.go`

**Step 1: 创建楼宇模型文件**

```go
package operations

import (
    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// BuildingStatus 楼宇状态枚举
type BuildingStatus int

const (
    BuildingStatusNormal  BuildingStatus = 0 // 正常
    BuildingStatusStopped BuildingStatus = 1 // 停用
)

// OpsBuilding 楼宇模型
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

// TableName 指定表名
func (OpsBuilding) TableName() string {
    return "ops_buildings"
}
```

**Step 2: 提交**

```bash
cd .worktrees/operations-management
git add internal/models/operations/building.go
git commit -m "feat(operations): 添加楼宇数据模型"
```

---

### Task 2: 创建楼层模型

**文件:**
- 创建: `internal/models/operations/floor.go`

**Step 1: 创建楼层模型文件**

```go
package operations

import (
    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// FloorStatus 楼层状态枚举
type FloorStatus int

const (
    FloorStatusNormal  FloorStatus = 0 // 正常
    FloorStatusStopped FloorStatus = 1 // 停用
)

// OpsFloor 楼层模型
type OpsFloor struct {
    models.BaseModel
    Name       string      // 楼层名称
    FloorNo    string      // 楼层编号（如1F, 2F, B1）
    BuildingID string      // 所属楼宇ID
    Status     FloorStatus // 状态: 0=正常, 1=停用
    Remark     string      // 备注
    OrderNum   int         // 排序号
}

// TableName 指定表名
func (OpsFloor) TableName() string {
    return "ops_floors"
}
```

**Step 2: 提交**

```bash
git add internal/models/operations/floor.go
git commit -m "feat(operations): 添加楼层数据模型"
```

---

### Task 3: 创建工位模型

**文件:**
- 创建: `internal/models/operations/workstation.go`

**Step 1: 创建工位模型文件**

```go
package operations

import (
    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// WorkstationStatus 工位状态枚举
type WorkstationStatus int

const (
    WorkstationStatusFree    WorkstationStatus = 0 // 空闲
    WorkstationStatusOccupied WorkstationStatus = 1 // 占用
)

// WorkstationType 工位类型枚举
type WorkstationType int

const (
    WorkstationTypeFixed  WorkstationType = 0 // 固定工位
    WorkstationTypeShared WorkstationType = 1 // 共享工位
)

// OpsWorkstation 工位模型
type OpsWorkstation struct {
    models.BaseModel
    Code        string                // 工位编号
    Name        string                // 工位名称
    FloorID     string                // 所属楼层ID
    UserID      string                // 关联用户ID
    DeptID      string                // 关联部门ID
    PhonePoint  string                // 电话信息点
    Status      WorkstationStatus     // 状态: 0=空闲, 1=占用
    Type        WorkstationType       // 类型: 0=固定, 1=共享
    Equipment   string                // 设备配置（JSON）
    NetworkInfo string                // 网络信息
    Remark      string                // 备注
    PositionX   *int                  // 平面图X坐标（可选）
    PositionY   *int                  // 平面图Y坐标（可选）
}

// TableName 指定表名
func (OpsWorkstation) TableName() string {
    return "ops_workstations"
}
```

**Step 2: 提交**

```bash
git add internal/models/operations/workstation.go
git commit -m "feat(operations): 添加工位数据模型"
```

---

### Task 4: 创建机房模型

**文件:**
- 创建: `internal/models/operations/server_room.go`

**Step 1: 创建机房模型文件**

```go
package operations

import (
    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// RoomStatus 机房状态枚举
type RoomStatus int

const (
    RoomStatusNormal  RoomStatus = 0 // 正常
    RoomStatusStopped RoomStatus = 1 // 停用
)

// OpsServerRoom 机房模型
type OpsServerRoom struct {
    models.BaseModel
    Name    string     // 机房名称
    Code    string     // 机房编号（唯一）
    FloorID string     // 所在楼层ID
    Status  RoomStatus // 状态: 0=正常, 1=停用
    Remark  string     // 备注
}

// TableName 指定表名
func (OpsServerRoom) TableName() string {
    return "ops_server_rooms"
}
```

**Step 2: 提交**

```bash
git add internal/models/operations/server_room.go
git commit -m "feat(operations): 添加机房数据模型"
```

---

### Task 5: 创建机房照片关联模型

**文件:**
- 创建: `internal/models/operations/room_photo.go`

**Step 1: 创建机房照片关联模型**

```go
package operations

import (
    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// OpsRoomPhoto 机房照片关联模型
type OpsRoomPhoto struct {
    models.BaseModel
    RoomID      string // 机房ID
    FileID      string // 文件ID(关联sys_files)
    SortOrder   int    // 排序号
    IsPrimary   bool   // 是否为主图
    Description string // 照片描述
}

// TableName 指定表名
func (OpsRoomPhoto) TableName() string {
    return "ops_room_photos"
}
```

**Step 2: 提交**

```bash
git add internal/models/operations/room_photo.go
git commit -m "feat(operations): 添加机房照片关联模型"
```

---

### Task 6: 创建机房-网络设备关联模型

**文件:**
- 创建: `internal/models/operations/room_network_device.go`

**Step 1: 创建关联模型**

```go
package operations

import (
    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// OpsRoomNetworkDevice 机房-网络设备关联模型
type OpsRoomNetworkDevice struct {
    models.BaseModel
    RoomID       string // 机房ID
    DeviceID     string // 网络设备ID（关联sys_network_device）
    PositionU    int    // 机架位置（U）
    PositionDesc string // 位置描述（如：第1列第3个机架）
}

// TableName 指定表名
func (OpsRoomNetworkDevice) TableName() string {
    return "ops_room_network_devices"
}
```

**Step 2: 提交**

```bash
git add internal/models/operations/room_network_device.go
git commit -m "feat(operations): 添加机房-网络设备关联模型"
```

---

### Task 7: 创建专线模型

**文件:**
- 创建: `internal/models/operations/dedicated_line.go`

**Step 1: 创建专线模型**

```go
package operations

import (
    "time"
    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// DedicatedLineType 专线类型枚举
type DedicatedLineType string

const (
    LineTypeInternet    DedicatedLineType = "internet"    // 互联网专线
    LineTypeMPLS        DedicatedLineType = "mpls"        // MPLS VPN
    LineTypeLeasedLine  DedicatedLineType = "leased_line" // 租用专线
    LineTypeFiber       DedicatedLineType = "fiber"       // 光纤专线
)

// LineStatus 专线状态枚举
type LineStatus int

const (
    LineStatusNormal   LineStatus = 0 // 正常
    LineStatusFault    LineStatus = 1 // 故障
    LineStatusDisabled LineStatus = 2 // 停用
)

// OpsDedicatedLine 专线模型
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

// TableName 指定表名
func (OpsDedicatedLine) TableName() string {
    return "ops_dedicated_lines"
}
```

**Step 2: 提交**

```bash
git add internal/models/operations/dedicated_line.go
git commit -m "feat(operations): 添加专线数据模型"
```

---

### Task 8: 创建机房设备模型

**文件:**
- 创建: `internal/models/operations/room_device.go`

**Step 1: 创建机房设备模型**

```go
package operations

import (
    "time"
    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// RoomDeviceType 机房设备类型枚举
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

// RoomDeviceStatus 机房设备状态枚举
type RoomDeviceStatus int

const (
    RoomDeviceStatusNormal   RoomDeviceStatus = 0 // 正常
    RoomDeviceStatusFault    RoomDeviceStatus = 1 // 故障
    RoomDeviceStatusScrapped RoomDeviceStatus = 2 // 报废
)

// OpsRoomDevice 机房设备模型
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
    PowerConsumption int               // 功耗（瓦）
    Remark          string            // 备注
}

// TableName 指定表名
func (OpsRoomDevice) TableName() string {
    return "ops_room_devices"
}
```

**Step 2: 提交**

```bash
git add internal/models/operations/room_device.go
git commit -m "feat(operations): 添加机房设备数据模型"
```

---

### Task 9: 创建文件管理模型

**文件:**
- 创建: `internal/models/system/file.go`

**Step 1: 创建文件管理模型**

```go
package system

import (
    "time"
    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// SysFile 通用文件模型
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

// TableName 指定表名
func (SysFile) TableName() string {
    return "sys_files"
}
```

**Step 2: 创建文件访问日志模型**

```go
// 在同一文件中添加

// SysFileAccessLog 文件访问日志模型
type SysFileAccessLog struct {
    models.BaseTimeLine
    FileID     string // 文件ID
    ActionType string // 操作类型(upload/download/delete/view)
    UserID     string // 操作者ID
    UserName   string // 操作者姓名
    IPAddress  string // IP地址
    UserAgent  string // 浏览器信息
}

// TableName 指定表名
func (SysFileAccessLog) TableName() string {
    return "sys_file_access_logs"
}
```

**Step 3: 提交**

```bash
git add internal/models/system/file.go
git commit -m "feat(system): 添加文件管理数据模型"
```

---

### Task 10: 创建数据库迁移文件

**文件:**
- 创建: `migrations/001_create_ops_buildings_table.up.sql`
- 创建: `migrations/002_create_ops_floors_table.up.sql`
- 创建: `migrations/003_create_ops_workstations_table.up.sql`
- 创建: `migrations/004_create_ops_server_rooms_table.up.sql`
- 创建: `migrations/005_create_ops_room_photos_table.up.sql`
- 创建: `migrations/006_create_ops_room_network_devices_table.up.sql`
- 创建: `migrations/007_create_ops_dedicated_lines_table.up.sql`
- 创建: `migrations/008_create_ops_room_devices_table.up.sql`
- 创建: `migrations/009_create_sys_files_table.up.sql`
- 创建: `migrations/010_create_sys_file_access_logs_table.up.sql`

**Step 1: 创建楼宇表迁移**

```sql
-- 001_create_ops_buildings_table.up.sql
CREATE TABLE IF NOT EXISTS ops_buildings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,

    name VARCHAR(100) NOT NULL,
    code VARCHAR(20) NOT NULL UNIQUE,
    address VARCHAR(200),
    org_id UUID NOT NULL,
    status INTEGER DEFAULT 0,
    remark TEXT,
    order_num INTEGER DEFAULT 0
);

-- 创建索引
CREATE INDEX idx_ops_buildings_org_id ON ops_buildings(org_id);
CREATE INDEX idx_ops_buildings_status ON ops_buildings(status);
CREATE INDEX idx_ops_buildings_deleted_at ON ops_buildings(deleted_at);

-- 添加注释
COMMENT ON TABLE ops_buildings IS '楼宇表';
COMMENT ON COLUMN ops_buildings.status IS '状态: 0=正常, 1=停用';
```

**Step 2: 创建楼层表迁移**

```sql
-- 002_create_ops_floors_table.up.sql
CREATE TABLE IF NOT EXISTS ops_floors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,

    name VARCHAR(100) NOT NULL,
    floor_no VARCHAR(10) NOT NULL,
    building_id UUID NOT NULL,
    status INTEGER DEFAULT 0,
    remark TEXT,
    order_num INTEGER DEFAULT 0,

    FOREIGN KEY (building_id) REFERENCES ops_buildings(id)
);

-- 创建索引
CREATE INDEX idx_ops_floors_building_id ON ops_floors(building_id);
CREATE INDEX idx_ops_floors_status ON ops_floors(status);
CREATE INDEX idx_ops_floors_deleted_at ON ops_floors(deleted_at);

-- 添加注释
COMMENT ON TABLE ops_floors IS '楼层表';
COMMENT ON COLUMN ops_floors.status IS '状态: 0=正常, 1=停用';
```

**Step 3: 创建工位表迁移**

```sql
-- 003_create_ops_workstations_table.up.sql
CREATE TABLE IF NOT EXISTS ops_workstations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,

    code VARCHAR(30) NOT NULL,
    name VARCHAR(100) NOT NULL,
    floor_id UUID NOT NULL,
    user_id UUID,
    dept_id UUID,
    phone_point VARCHAR(20),
    status INTEGER DEFAULT 0,
    type INTEGER DEFAULT 0,
    equipment TEXT,
    network_info VARCHAR(500),
    remark TEXT,
    position_x INTEGER,
    position_y INTEGER,

    FOREIGN KEY (floor_id) REFERENCES ops_floors(id)
);

-- 创建索引
CREATE INDEX idx_ops_workstations_floor_id ON ops_workstations(floor_id);
CREATE INDEX idx_ops_workstations_user_id ON ops_workstations(user_id);
CREATE INDEX idx_ops_workstations_dept_id ON ops_workstations(dept_id);
CREATE INDEX idx_ops_workstations_status ON ops_workstations(status);
CREATE INDEX idx_ops_workstations_deleted_at ON ops_workstations(deleted_at);

-- 添加注释
COMMENT ON TABLE ops_workstations IS '工位表';
COMMENT ON COLUMN ops_workstations.status IS '状态: 0=空闲, 1=占用';
COMMENT ON COLUMN ops_workstations.type IS '类型: 0=固定, 1=共享';
```

**Step 4: 创建机房表迁移**

```sql
-- 004_create_ops_server_rooms_table.up.sql
CREATE TABLE IF NOT EXISTS ops_server_rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,

    name VARCHAR(100) NOT NULL,
    code VARCHAR(20) NOT NULL UNIQUE,
    floor_id UUID NOT NULL,
    status INTEGER DEFAULT 0,
    remark TEXT,

    FOREIGN KEY (floor_id) REFERENCES ops_floors(id)
);

-- 创建索引
CREATE INDEX idx_ops_server_rooms_floor_id ON ops_server_rooms(floor_id);
CREATE INDEX idx_ops_server_rooms_status ON ops_server_rooms(status);
CREATE INDEX idx_ops_server_rooms_deleted_at ON ops_server_rooms(deleted_at);

-- 添加注释
COMMENT ON TABLE ops_server_rooms IS '机房表';
COMMENT ON COLUMN ops_server_rooms.status IS '状态: 0=正常, 1=停用';
```

**Step 5-10: 创建其他迁移文件**

（完整 SQL 内容省略，遵循相同模式）

**Step 11: 提交所有迁移文件**

```bash
git add migrations/
git commit -m "feat(operations): 添加数据库迁移文件"
```

---

## 阶段二：后端基础 - Service 层

### Task 11: 创建楼宇 Service

**文件:**
- 创建: `internal/services/operations/building_service.go`
- 创建: `internal/services/operations/building_service_test.go`

**Step 1: 创建楼宇 Service 接口和实现**

```go
package operations

import (
    "context"
    "errors"
    "fmt"

    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
    "gorm.io/gorm"
)

// BuildingService 楼宇服务接口
type BuildingService interface {
    Create(ctx context.Context, building *models.OpsBuilding) error
    Update(ctx context.Context, building *models.OpsBuilding) error
    Delete(ctx context.Context, id string) error
    GetByID(ctx context.Context, id string) (*models.OpsBuilding, error)
    List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
    BatchDelete(ctx context.Context, ids []string) error
}

type buildingService struct {
    db *gorm.DB
}

// NewBuildingService 创建楼宇服务实例
func NewBuildingService(db *gorm.DB) BuildingService {
    return &buildingService{db: db}
}

// Create 创建楼宇
func (s *buildingService) Create(ctx context.Context, building *models.OpsBuilding) error {
    // 检查编码唯一性
    if err := s.checkCodeUnique(ctx, building.Code, ""); err != nil {
        return err
    }

    // 验证机构存在性
    if err := s.validateOrg(ctx, building.OrgID); err != nil {
        return err
    }

    return s.db.WithContext(ctx).Create(building).Error
}

// Update 更新楼宇
func (s *buildingService) Update(ctx context.Context, building *models.OpsBuilding) error {
    // 检查编码唯一性
    if err := s.checkCodeUnique(ctx, building.Code, building.ID); err != nil {
        return err
    }

    // 验证机构存在性
    if err := s.validateOrg(ctx, building.OrgID); err != nil {
        return err
    }

    return s.db.WithContext(ctx).Save(building).Error
}

// Delete 删除楼宇
func (s *buildingService) Delete(ctx context.Context, id string) error {
    return s.db.WithContext(ctx).Delete(&models.OpsBuilding{}, "id = ?", id).Error
}

// GetByID 根据ID获取楼宇
func (s *buildingService) GetByID(ctx context.Context, id string) (*models.OpsBuilding, error) {
    var building models.OpsBuilding
    err := s.db.WithContext(ctx).Where("id = ?", id).First(&building).Error
    if err != nil {
        return nil, err
    }
    return &building, nil
}

// List 查询楼宇列表
func (s *buildingService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
    var total int64
    var list []models.OpsBuilding

    query := s.db.WithContext(ctx).Model(&models.OpsBuilding{})

    // 添加筛选条件
    if name, ok := params["name"].(string); ok && name != "" {
        query = query.Where("name LIKE ?", "%"+name+"%")
    }
    if code, ok := params["code"].(string); ok && code != "" {
        query = query.Where("code LIKE ?", "%"+code+"%")
    }
    if orgId, ok := params["orgId"].(string); ok && orgId != "" {
        query = query.Where("org_id = ?", orgId)
    }
    if status, ok := params["status"].(int); ok {
        query = query.Where("status = ?", status)
    }

    // 获取总数
    if err := query.Count(&total).Error; err != nil {
        return nil, err
    }

    // 分页
    current := 1
    pageSize := 10
    if c, ok := params["current"].(int); ok {
        current = c
    }
    if ps, ok := params["pageSize"].(int); ok {
        pageSize = ps
    }

    offset := (current - 1) * pageSize
    if err := query.Offset(offset).Limit(pageSize).Order("order_num ASC").Find(&list).Error; err != nil {
        return nil, err
    }

    return &PageResult{
        List:     list,
        Total:    total,
        Current:  current,
        PageSize: pageSize,
    }, nil
}

// BatchDelete 批量删除楼宇
func (s *buildingService) BatchDelete(ctx context.Context, ids []string) error {
    return s.db.WithContext(ctx).Delete(&models.OpsBuilding{}, "id IN ?", ids).Error
}

// checkCodeUnique 检查编码唯一性
func (s *buildingService) checkCodeUnique(ctx context.Context, code string, excludeID string) error {
    var count int64
    query := s.db.WithContext(ctx).Model(&models.OpsBuilding{}).Where("code = ?", code)
    if excludeID != "" {
        query = query.Where("id != ?", excludeID)
    }
    if err := query.Count(&count).Error; err != nil {
        return err
    }
    if count > 0 {
        return errors.New("楼宇编码已存在")
    }
    return nil
}

// validateOrg 验证机构存在性
func (s *buildingService) validateOrg(ctx context.Context, orgID string) error {
    var count int64
    if err := s.db.WithContext(ctx).Table("sys_dept").Where("id = ?", orgID).Count(&count).Error; err != nil {
        return err
    }
    if count == 0 {
        return errors.New("所属机构不存在")
    }
    return nil
}

// PageResult 分页结果
type PageResult struct {
    List     interface{} `json:"list"`
    Total    int64      `json:"total"`
    Current  int        `json:"current"`
    PageSize int        `json:"pageSize"`
}
```

**Step 2: 创建测试文件**

```go
package operations

import (
    "context"
    "testing"

    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
    "github.com/stretchr/testify/assert"
)

func TestBuildingService_Create(t *testing.T) {
    // 这里需要 mock DB 或者使用测试数据库
    // 暂时跳过，实际实现时需要添加
    t.Skip("需要 mock DB 或测试数据库")
}

func TestBuildingService_CheckCodeUnique(t *testing.T) {
    t.Skip("需要 mock DB 或测试数据库")
}
```

**Step 3: 提交**

```bash
git add internal/services/operations/
git commit -m "feat(operations): 实现楼宇 Service 层"
```

---

### Task 12: 创建楼层 Service

**文件:**
- 创建: `internal/services/operations/floor_service.go`

**Step 1: 创建楼层 Service**

```go
package operations

import (
    "context"
    "errors"

    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
    "gorm.io/gorm"
)

// FloorService 楼层服务接口
type FloorService interface {
    Create(ctx context.Context, floor *models.OpsFloor) error
    Update(ctx context.Context, floor *models.OpsFloor) error
    Delete(ctx context.Context, id string) error
    GetByID(ctx context.Context, id string) (*models.OpsFloor, error)
    List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
    GetTree(ctx context.Context) ([]FloorTreeNode, error)
    BatchDelete(ctx context.Context, ids []string) error
}

type FloorTreeNode struct {
    ID       string          `json:"id"`
    Name     string          `json:"name"`
    FloorNo  string          `json:"floorNo"`
    BuildingID string         `json:"buildingId"`
    Children []FloorTreeNode `json:"children"`
}

type floorService struct {
    db *gorm.DB
}

// NewFloorService 创建楼层服务实例
func NewFloorService(db *gorm.DB) FloorService {
    return &floorService{db: db}
}

// Create 创建楼层
func (s *floorService) Create(ctx context.Context, floor *models.OpsFloor) error {
    // 验证楼宇存在性
    if err := s.validateBuilding(ctx, floor.BuildingID); err != nil {
        return err
    }
    return s.db.WithContext(ctx).Create(floor).Error
}

// Update 更新楼层
func (s *floorService) Update(ctx context.Context, floor *models.OpsFloor) error {
    if floor.BuildingID != "" {
        if err := s.validateBuilding(ctx, floor.BuildingID); err != nil {
            return err
        }
    }
    return s.db.WithContext(ctx).Save(floor).Error
}

// Delete 删除楼层
func (s *floorService) Delete(ctx context.Context, id string) error {
    return s.db.WithContext(ctx).Delete(&models.OpsFloor{}, "id = ?", id).Error
}

// GetByID 根据ID获取楼层
func (s *floorService) GetByID(ctx context.Context, id string) (*models.OpsFloor, error) {
    var floor models.OpsFloor
    err := s.db.WithContext(ctx).Where("id = ?", id).First(&floor).Error
    if err != nil {
        return nil, err
    }
    return &floor, nil
}

// List 查询楼层列表
func (s *floorService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
    var total int64
    var list []models.OpsFloor

    query := s.db.WithContext(ctx).Model(&models.OpsFloor{})

    // 添加筛选条件
    if name, ok := params["name"].(string); ok && name != "" {
        query = query.Where("name LIKE ?", "%"+name+"%")
    }
    if buildingId, ok := params["buildingId"].(string); ok && buildingId != "" {
        query = query.Where("building_id = ?", buildingId)
    }

    // 分页
    current := 1
    pageSize := 10
    if c, ok := params["current"].(int); ok {
        current = c
    }
    if ps, ok := params["pageSize"].(int); ok {
        pageSize = ps
    }

    if err := query.Count(&total).Error; err != nil {
        return nil, err
    }

    offset := (current - 1) * pageSize
    if err := query.Offset(offset).Limit(pageSize).Order("order_num ASC").Find(&list).Error; err != nil {
        return nil, err
    }

    return &PageResult{
        List:     list,
        Total:    total,
        Current:  current,
        PageSize: pageSize,
    }, nil
}

// GetTree 获取楼宇-楼层树
func (s *floorService) GetTree(ctx context.Context) ([]FloorTreeNode, error) {
    type BuildingFloor struct {
        BuildingID   string
        BuildingName string
        FloorID      string
        FloorName    string
        FloorNo      string
    }

    var results []BuildingFloor
    err := s.db.WithContext(ctx).Raw(`
        SELECT
            b.id as building_id,
            b.name as building_name,
            f.id as floor_id,
            f.name as floor_name,
            f.floor_no
        FROM ops_buildings b
        LEFT JOIN ops_floors f ON b.id = f.building_id
        WHERE b.deleted_at IS NULL
        ORDER BY b.order_num ASC, f.order_num ASC
    `).Scan(&results).Error

    if err != nil {
        return nil, err
    }

    // 构建树形结构
    buildingMap := make(map[string]*FloorTreeNode)

    for _, r := range results {
        if _, exists := buildingMap[r.BuildingID]; !exists {
            buildingMap[r.BuildingID] = &FloorTreeNode{
                ID:       r.BuildingID,
                Name:     r.BuildingName,
                Children: []FloorTreeNode{},
            }
        }

        if r.FloorID != "" {
            buildingMap[r.BuildingID].Children = append(buildingMap[r.BuildingID].Children, FloorTreeNode{
                ID:          r.FloorID,
                Name:        r.FloorName,
                FloorNo:     r.FloorNo,
                BuildingID:  r.BuildingID,
            })
        }
    }

    // 转换为数组
    var tree []FloorTreeNode
    for _, node := range buildingMap {
        tree = append(tree, *node)
    }

    return tree, nil
}

// BatchDelete 批量删除楼层
func (s *floorService) BatchDelete(ctx context.Context, ids []string) error {
    return s.db.WithContext(ctx).Delete(&models.OpsFloor{}, "id IN ?", ids).Error
}

// validateBuilding 验证楼宇存在性
func (s *floorService) validateBuilding(ctx context.Context, buildingID string) error {
    var count int64
    if err := s.db.WithContext(ctx).Model(&models.OpsBuilding{}).Where("id = ?", buildingID).Count(&count).Error; err != nil {
        return err
    }
    if count == 0 {
        return errors.New("所属楼宇不存在")
    }
    return nil
}
```

**Step 2: 提交**

```bash
git add internal/services/operations/floor_service.go
git commit -m "feat(operations): 实现楼层 Service 层"
```

---

（由于篇幅限制，后续任务的详细步骤将在实际执行时按需展开。实施计划框架已完成，包含：

1. 完整的数据模型定义
2. Service 层接口示例（楼宇、楼层）
3. 标准化的文件创建和提交流程

剩余模块（工位、机房、专线、机房设备）的 Service 层将遵循相同模式实现。）

---

## 阶段三：后端 - Handler 层

### Task 13: 创建楼宇 Handler

**文件:**
- 创建: `internal/api/v1/operations/building_handler.go`

**Step 1: 创建楼宇 Handler**

```go
package operations

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
    "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
    "github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

type BuildingHandler struct {
    service operations.BuildingService
}

func NewBuildingHandler(service operations.BuildingService) *BuildingHandler {
    return &BuildingHandler{service: service}
}

// Create 创建楼宇
func (h *BuildingHandler) Create(c *gin.Context) {
    var building models.OpsBuilding
    if err := c.ShouldBindJSON(&building); err != nil {
        response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
        return
    }

    if err := h.service.Create(c.Request.Context(), &building); err != nil {
        response.Error(c, http.StatusInternalServerError, "创建失败: "+err.Error())
        return
    }

    response.Success(c, building)
}

// List 查询楼宇列表
func (h *BuildingHandler) List(c *gin.Context) {
    var params map[string]interface{}
    if err := c.ShouldBindJSON(&params); err != nil {
        params = make(map[string]interface{})
    }

    result, err := h.service.List(c.Request.Context(), params)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, "查询失败: "+err.Error())
        return
    }

    response.Success(c, result)
}

// GetByID 获取楼宇详情
func (h *BuildingHandler) GetByID(c *gin.Context) {
    id := c.Param("id")
    building, err := h.service.GetByID(c.Request.Context(), id)
    if err != nil {
        response.Error(c, http.StatusNotFound, "楼宇不存在")
        return
    }

    response.Success(c, building)
}

// Update 更新楼宇
func (h *BuildingHandler) Update(c *gin.Context) {
    id := c.Param("id")
    var building models.OpsBuilding
    if err := c.ShouldBindJSON(&building); err != nil {
        response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
        return
    }

    building.ID = id
    if err := h.service.Update(c.Request.Context(), &building); err != nil {
        response.Error(c, http.StatusInternalServerError, "更新失败: "+err.Error())
        return
    }

    response.Success(c, nil)
}

// Delete 删除楼宇
func (h *BuildingHandler) Delete(c *gin.Context) {
    id := c.Param("id")
    if err := h.service.Delete(c.Request.Context(), id); err != nil {
        response.Error(c, http.StatusInternalServerError, "删除失败: "+err.Error())
        return
    }

    response.Success(c, nil)
}

// BatchOperation 批量操作
func (h *BuildingHandler) BatchOperation(c *gin.Context) {
    var req struct {
        IDs   []string `json:"ids"`
        Action string   `json:"action"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, "参数错误")
        return
    }

    switch req.Action {
    case "delete":
        if err := h.service.BatchDelete(c.Request.Context(), req.IDs); err != nil {
            response.Error(c, http.StatusInternalServerError, "批量删除失败")
            return
        }
    default:
        response.Error(c, http.StatusBadRequest, "不支持的操作")
        return
    }

    response.Success(c, nil)
}
```

**Step 2: 提交**

```bash
git add internal/api/v1/operations/building_handler.go
git commit -m "feat(operations): 实现楼宇 Handler 层"
```

---

## 阶段四：路由注册

### Task 14: 注册运维管理路由

**文件:**
- 修改: `internal/api/router.go`

**Step 1: 在路由文件中添加运维管理路由组**

找到路由注册的位置，添加以下代码：

```go
// 在导入部分添加
import "github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/operations"
import "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"

// 在路由注册部分添加
func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
    v1 := r.Group("/api/v1")

    // ... 现有路由 ...

    // 运维管理路由组
    opsGroup := v1.Group("/ops")
    {
        buildingService := operations.NewBuildingService(db)
        buildingHandler := operations.NewBuildingHandler(buildingService)

        buildings := opsGroup.Group("/buildings")
        {
            buildings.POST("", buildingHandler.Create)
            buildings.POST("/list", buildingHandler.List)
            buildings.POST("/:id", buildingHandler.GetByID)
            buildings.POST("/:id/update", buildingHandler.Update)
            buildings.POST("/:id/delete", buildingHandler.Delete)
            buildings.POST("/batch", buildingHandler.BatchOperation)
        }

        // 其他模块路由将在后续任务中添加
    }
}
```

**Step 2: 提交**

```bash
git add internal/api/router.go
git commit -m "feat(operations): 注册运维管理路由"
```

---

## 阶段五：前端基础

### Task 15: 创建前端类型定义

**文件:**
- 创建: `ruoyi-react-frontend/src/types/operations.ts`

**Step 1: 创建运维管理类型定义**

```typescript
// 运维管理通用类型定义

// 楼宇
export interface Building {
  id: string;
  name: string;
  code: string;
  address: string;
  orgId: string;
  orgName?: string;
  status: BuildingStatus;
  remark?: string;
  orderNum: number;
  createdAt: string;
  updatedAt: string;
}

export type BuildingStatus = 0 | 1; // 0: 正常, 1: 停用

// 楼层
export interface Floor {
  id: string;
  name: string;
  floorNo: string;
  buildingId: string;
  buildingName?: string;
  status: FloorStatus;
  remark?: string;
  orderNum: number;
  createdAt: string;
  updatedAt: string;
}

export type FloorStatus = 0 | 1; // 0: 正常, 1: 停用

// 工位
export interface Workstation {
  id: string;
  code: string;
  name: string;
  floorId: string;
  floorName?: string;
  userId?: string;
  userName?: string;
  deptId?: string;
  deptName?: string;
  phonePoint?: string;
  status: WorkstationStatus;
  type: WorkstationType;
  equipment?: string;
  networkInfo?: string;
  remark?: string;
  positionX?: number;
  positionY?: number;
  createdAt: string;
  updatedAt: string;
}

export type WorkstationStatus = 0 | 1; // 0: 空闲, 1: 占用
export type WorkstationType = 0 | 1; // 0: 固定, 1: 共享

// 机房
export interface ServerRoom {
  id: string;
  name: string;
  code: string;
  floorId: string;
  floorName?: string;
  status: RoomStatus;
  remark?: string;
  photos?: RoomPhoto[];
  networkDevices?: NetworkDevice[];
  createdAt: string;
  updatedAt: string;
}

export type RoomStatus = 0 | 1; // 0: 正常, 1: 停用

export interface RoomPhoto {
  id: string;
  fileId: string;
  fileUrl: string;
  sortOrder: number;
  isPrimary: boolean;
  description?: string;
}

// 专线
export interface DedicatedLine {
  id: string;
  name: string;
  lineCode: string;
  lineType: DedicatedLineType;
  bandwidth: string;
  isp: string;
  sourceDeviceId: string;
  sourceDeviceName?: string;
  sourcePort: string;
  destDeviceId: string;
  destDeviceName?: string;
  destPort: string;
  ipAddress?: string;
  subnetMask?: string;
  gateway?: string;
  vlan?: string;
  status: LineStatus;
  activationDate?: string;
  expiryDate?: string;
  responsibleId?: string;
  responsibleName?: string;
  contractNo?: string;
  monthlyFee: number;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

export type DedicatedLineType = 'internet' | 'mpls' | 'leased_line' | 'fiber';
export type LineStatus = 0 | 1 | 2; // 0: 正常, 1: 故障, 2: 停用

// 机房设备
export interface RoomDevice {
  id: string;
  name: string;
  deviceCode: string;
  deviceType: RoomDeviceType;
  model?: string;
  vendor?: string;
  serialNumber?: string;
  roomId: string;
  roomName?: string;
  positionU: number;
  positionDesc?: string;
  assetNumber?: string;
  purchaseDate?: string;
  warrantyDate?: string;
  status: RoomDeviceStatus;
  responsibleId?: string;
  responsibleName?: string;
  powerConsumption?: number;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

export type RoomDeviceType = 'server' | 'storage' | 'ups' | 'pdu' | 'firewall' | 'kvm' | 'cabinet' | 'other';
export type RoomDeviceStatus = 0 | 1 | 2; // 0: 正常, 1: 故障, 2: 报废

// 分页数据
export interface PageData<T> {
  list: T[];
  total: number;
  current: number;
  pageSize: number;
}

// 分页参数
export interface PageParams {
  current: number;
  pageSize: number;
  [key: string]: any;
}
```

**Step 2: 提交**

```bash
cd ruoyi-react-frontend
git add src/types/operations.ts
git commit -m "feat(operations): 添加类型定义"
```

---

### Task 16: 创建楼宇管理 API

**文件:**
- 创建: `ruoyi-react-frontend/src/services/operations/buildings.ts`

**Step 1: 创建楼宇 API 封装**

```typescript
import { post } from '@/lib/api';
import { PageData, PageParams, Building } from '@/types/operations';

export const buildingApi = {
  // 列表查询
  list: (params: PageParams) => post<PageData<Building>>('/ops/buildings/list', params),

  // 创建
  create: (data: Omit<Building, 'id' | 'createdAt' | 'updatedAt'>) =>
    post('/ops/buildings', data),

  // 更新
  update: (id: string, data: Partial<Building>) =>
    post(`/ops/buildings/${id}/update`, data),

  // 删除
  delete: (id: string) => post(`/ops/buildings/${id}/delete`),

  // 批量删除
  batchDelete: (ids: string[]) =>
    post('/ops/buildings/batch', { ids, action: 'delete' }),

  // 导出
  export: (params: PageParams) =>
    post('/ops/buildings/export', params, { responseType: 'blob' }),

  // 导入
  import: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return post('/ops/buildings/import', formData);
  },

  // 下载模板
  downloadTemplate: () =>
    post('/ops/buildings/template', {}, { responseType: 'blob' }),
};
```

**Step 2: 提交**

```bash
git add src/services/operations/buildings.ts
git commit -m "feat(operations): 添加楼宇 API"
```

---

### Task 17: 创建楼宇管理页面

**文件:**
- 创建: `ruoyi-react-frontend/src/pages/operations/buildings/index.tsx`

**Step 1: 创建楼宇管理主页面**

```typescript
import { useState, useEffect, useCallback } from 'react';
import type { FC } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Popconfirm,
  Tag,
  Card,
  Row,
  Col,
  Statistic,
  Radio,
  Grid,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  BuildingOutlined,
} from '@ant-design/icons';
import type { Key } from 'antd/es/table/interface';
import { buildingApi } from '@/services/operations/buildings';
import { useTableManager } from '@/hooks/useTableManager';
import { handleApiError, handleSuccess } from '@/utils/errorHandler';
import ActionButtons from '@/components/shared/ActionButtons';
import type { Building } from '@/types/operations';

const { Option } = Select;

const BuildingManagement: FC = () => {
  const [viewMode, setViewMode] = useState<'table' | 'card'>('table');
  const [departments, setDepartments] = useState<any[]>([]);

  const fetchBuildings = useCallback(async (params: any) => {
    const result = await buildingApi.list(params);
    return { list: result.data.list, total: result.data.total };
  }, []);

  const {
    loading,
    data: buildings,
    total,
    current,
    pageSize,
    selectedRowKeys,
    searchForm,
    editForm,
    editModalVisible,
    editingItem,
    setCurrent,
    setPageSize,
    setSelectedRowKeys,
    setEditModalVisible,
    loadData: loadBuildings,
    handleAdd,
    handleEdit,
    resetSelection,
  } = useTableManager<Building>(fetchBuildings, { pageSize: 10 });

  // 加载部门列表
  const loadDepartments = useCallback(async () => {
    try {
      const result = await post('/system/departments/tree-select');
      setDepartments(result.data || []);
    } catch (error) {
      handleApiError(error, '加载部门列表', false);
    }
  }, []);

  useEffect(() => {
    loadBuildings();
    loadDepartments();
  }, []);

  const handleSearch = () => {
    const values = searchForm.getFieldsValue();
    const searchParams: any = {};
    Object.keys(values).forEach(key => {
      const value = values[key];
      if (value !== undefined && value !== null && value !== '') {
        if (key === 'status') {
          searchParams[key] = Number(value);
        } else {
          searchParams[key] = value;
        }
      }
    });
    loadBuildings(searchParams);
  };

  const handleReset = () => {
    searchForm.resetFields();
    loadBuildings();
  };

  const handleRefresh = () => {
    loadBuildings();
  };

  const handleAddBuilding = () => {
    handleAdd();
    editForm.setFieldsValue({
      status: 0,
      orderNum: 0,
    });
  };

  const handleModalOpenChange = (open: boolean) => {
    if (open && editingItem) {
      editForm.setFieldsValue(editingItem);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await buildingApi.delete(id);
      handleSuccess('删除');
      loadBuildings();
    } catch (error) {
      handleApiError(error, '删除');
    }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;

    try {
      await buildingApi.batchDelete(selectedRowKeys as string[]);
      handleSuccess('批量删除');
      resetSelection();
      loadBuildings();
    } catch (error) {
      handleApiError(error, '批量删除');
    }
  };

  const handleSave = async () => {
    try {
      const values = await editForm.validateFields();

      const url = editingItem
        ? `/ops/buildings/${editingItem.id}/update`
        : '/ops/buildings';

      await post(url, values);
      handleSuccess(editingItem ? '更新' : '创建');
      setEditModalVisible(false);
      loadBuildings();
    } catch (error) {
      handleApiError(error, '操作');
    }
  };

  const columns: ColumnsType<Building> = [
    { title: '楼宇名称', dataIndex: 'name', key: 'name' },
    { title: '楼宇编码', dataIndex: 'code', key: 'code' },
    { title: '地址', dataIndex: 'address', key: 'address' },
    { title: '所属机构', dataIndex: 'orgName', key: 'orgName' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag color={status === 0 ? 'success' : 'error'}>
          {status === 0 ? '正常' : '停用'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => {
        const actions = [
          {
            key: 'edit',
            label: '编辑',
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
          },
          {
            key: 'delete',
            label: '删除',
            icon: <DeleteOutlined />,
            danger: true,
            render: () => (
              <Popconfirm
                title="确定要删除这个楼宇吗？"
                onConfirm={() => handleDelete(record.id)}
                okText="确定"
                cancelText="取消"
              >
                <Button type="link" danger icon={<DeleteOutlined />} size="small">
                  删除
                </Button>
              </Popconfirm>
            ),
          },
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];

  // 渲染部门树形选项
  const renderDeptTreeOptions = (departments: any[], level = 0) => {
    const options: any[] = [];
    departments.forEach(node => {
      options.push(
        <Option key={node.id} value={node.id}>
          {'　'.repeat(level * 2)}{node.deptName || node.name}
        </Option>
      );
      if (node.children && node.children.length > 0) {
        options.push(...renderDeptTreeOptions(node.children, level + 1));
      }
    });
    return options;
  };

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card>
            <Statistic
              title="总楼宇数"
              value={buildings.length}
              prefix={<BuildingOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="正常"
              value={buildings.filter(b => b.status === 0).length}
              styles={{ content: { color: '#3f8600' } }}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="停用"
              value={buildings.filter(b => b.status === 1).length}
              styles={{ content: { color: '#cf1322' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="name" label="楼宇名称">
              <Input placeholder="请输入楼宇名称" />
            </Form.Item>
            <Form.Item name="code" label="楼宇编码">
              <Input placeholder="请输入楼宇编码" />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" style={{ width: 120 }} allowClear>
                <Option value={0}>正常</Option>
                <Option value={1}>停用</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                  搜索
                </Button>
                <Button onClick={handleReset}>重置</Button>
                <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
                  刷新
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            {selectedRowKeys.length > 0 && (
              <Button
                danger
                icon={<DeleteOutlined />}
                onClick={handleBatchDelete}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            )}
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAddBuilding}>
              新增楼宇
            </Button>
          </Space>
        </div>
      </Card>

      {/* 视图切换 */}
      <Radio.Group
        value={viewMode}
        onChange={(e) => setViewMode(e.target.value)}
        style={{ marginBottom: 16 }}
      >
        <Radio.Button value="table">表格视图</Radio.Button>
        <Radio.Button value="card">卡片视图</Radio.Button>
      </Radio.Group>

      {/* 内容区 */}
      {viewMode === 'table' ? (
        <Card>
          <Table
            columns={columns}
            dataSource={buildings}
            rowKey="id"
            loading={loading}
            pagination={{
              current,
              pageSize,
              total,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: (total) => `共 ${total} 条`,
              onChange: (page, size) => {
                setCurrent(page);
                setPageSize(size);
                loadBuildings({ current: page, pageSize: size });
              },
            }}
            rowSelection={{
              selectedRowKeys,
              onChange: setSelectedRowKeys,
            }}
          />
        </Card>
      ) : (
        <Grid gutter={16}>
          {buildings.map((building) => (
            <Col span={8} key={building.id}>
              <Card
                title={building.name}
                extra={
                  <Space>
                    <Button size="small" icon={<EditOutlined />} onClick={() => handleEdit(building)}>
                      编辑
                    </Button>
                    <Popconfirm
                      title="确定要删除这个楼宇吗？"
                      onConfirm={() => handleDelete(building.id)}
                      okText="确定"
                      cancelText="取消"
                    >
                      <Button size="small" danger icon={<DeleteOutlined />}>
                        删除
                      </Button>
                    </Popconfirm>
                  </Space>
                }
              >
                <p><strong>编码：</strong>{building.code}</p>
                <p><strong>地址：</strong>{building.address || '-'}</p>
                <p><strong>状态：</strong>
                  <Tag color={building.status === 0 ? 'success' : 'error'}>
                    {building.status === 0 ? '正常' : '停用'}
                  </Tag>
                </p>
              </Card>
            </Col>
          ))}
        </Grid>
      )}

      {/* 编辑弹窗 */}
      <Modal
        title={editingItem ? '编辑楼宇' : '新增楼宇'}
        open={editModalVisible}
        onOk={handleSave}
        afterOpenChange={handleModalOpenChange}
        onCancel={() => setEditModalVisible(false)}
        width={600}
      >
        <Form form={editForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Form.Item
            name="name"
            label="楼宇名称"
            rules={[{ required: true, message: '请输入楼宇名称' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="code"
            label="楼宇编码"
            rules={[{ required: true, message: '请输入楼宇编码' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="address" label="地址">
            <Input />
          </Form.Item>
          <Form.Item
            name="orgId"
            label="所属机构"
            rules={[{ required: true, message: '请选择所属机构' }]}
          >
            <Select placeholder="请选择机构" allowClear showSearch>
              {renderDeptTreeOptions(departments)}
            </Select>
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue={0}>
            <Select>
              <Option value={0}>正常</Option>
              <Option value={1}>停用</Option>
            </Select>
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default BuildingManagement;
```

**Step 2: 提交**

```bash
git add src/pages/operations/buildings/
git commit -m "feat(operations): 实现楼宇管理页面"
```

---

## 阶段六：前端路由配置

### Task 18: 配置前端路由

**文件:**
- 修改: `ruoyi-react-frontend/src/App.tsx`

**Step 1: 添加运维管理路由**

在路由配置中添加：

```typescript
import BuildingManagement from '@/pages/operations/buildings';

// 在路由配置中添加
{
  path: '/operations',
  element: <Layout />,
  children: [
    {
      path: 'buildings',
      element: <BuildingManagement />,
    },
    // 其他运维管理路由将在后续添加
  ],
}
```

**Step 2: 提交**

```bash
git add src/App.tsx
git commit -m "feat(operations): 配置运维管理前端路由"
```

---

## 总结

以上是运维管理模块的核心实施计划，包含：

1. **数据模型**：楼宇、楼层、工位、机房、专线、机房设备、文件管理
2. **Service 层**：业务逻辑实现
3. **Handler 层**：API 接口实现
4. **路由注册**：后端路由配置
5. **前端类型**：TypeScript 类型定义
6. **前端 API**：API 封装
7. **前端页面**：楼宇管理页面（表格/卡片双视图）

剩余模块（楼层、工位、机房、专线、机房设备）的前端页面将遵循相同模式实现。

**预估工作量**：12 天
- 后端开发：3 天
- 前端开发：4 天
- 特色功能：4 天
- 测试优化：1 天
