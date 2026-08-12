# Research: Phase 26 资产管理模块

## 研究目标

研究 XingRan-Next 项目中类似的 CRUD 模块和 Excel 导入导出功能，为资产管理模块的设计提供参考。

## 现有模块分析

### 1. 类似的 CRUD 模块

#### Building 模块 (`internal/services/operations/building_service.go`)
- 实现了完整的 CRUD 操作
- 支持 Excel 导入导出
- 集成了地理编码服务
- 使用 `DataCacheService` 进行缓存管理

#### Workstation 模块 (`internal/services/operations/workstation_service.go`)
- 支持部门/用户关联
- Excel 导入时自动关联部门（通过部门名称匹配）
- 支持数据范围权限控制

### 2. Excel 导入导出架构

#### Excel 配置 (`internal/services/operations/excel_config.go`)
```go
// ExcelConfig 定义模块的 Excel 导入导出配置
type ExcelConfig struct {
    ModuleName    string
    TableName     string
    KeyField      string        // 唯一标识字段，用于判断更新/新增
    Columns       []ExcelColumn
    Required      []string      // 必填字段
    UniqueFields  []string      // 唯一字段
    SkipFields    []string      // 跳过的字段（不导入）
    ReferenceResolver          // 引用解析器（部门、用户等）
}
```

#### ExcelService (`internal/services/operations/excel_service.go`)
- 支持批量导入（BatchUpsert）
- 自动根据 KeyField 判断更新或新增
- 支持引用解析（部门、用户）
- 支持地理编码（Building 专用）
- 提供导入结果统计

### 3. Handler-Service 模式

#### 标准模式 (`internal/api/v1/operations/building_handler.go`)
```go
type BuildingHandler struct {
    service operations.BuildingService
}

func (h *BuildingHandler) List(c *gin.Context) {
    // 1. 绑定参数
    // 2. 调用 service
    // 3. 返回 response.Success 或 response.Error
}
```

#### Service 接口 (`internal/services/operations/building_service.go`)
```go
type BuildingService interface {
    List(ctx context.Context, params *ListParams) (*PageResult, error)
    Create(ctx context.Context, req *CreateRequest) (*Building, error)
    Update(ctx context.Context, id string, req *UpdateRequest) error
    Delete(ctx context.Context, id string) error
    BatchUpsert(ctx context.Context, items []*Building) (*BatchResult, error)
}
```

### 4. 前端架构

#### OpsApi 模式 (`xingran-react-frontend/src/lib/opsApi.ts`)
```typescript
// 为 operations 模块生成 CRUD API 工厂
export function createModuleApi<T>(moduleName: string) {
  return {
    list: (params: PageParams) => post(`/ops/${moduleName}/list`, params),
    create: (data: T) => post(`/ops/${moduleName}`, data),
    update: (id: string, data: T) => post(`/ops/${moduleName}/${id}/update`, data),
    delete: (id: string) => post(`/ops/${moduleName}/${id}/delete`),
  };
}
```

#### Excel 导入组件 (`xingran-react-frontend/src/pages/operations/building/index.tsx`)
- 使用 `Upload` 组件上传 Excel
- 使用 `excelApi.import()` 调用后端导入接口
- 显示导入结果统计

### 5. 路由模式 (`internal/api/v1/operations/operations_router.go`)

```go
func SetupOperationsRouter(r *gin.RouterGroup, core *core.Core) {
    // 为每个模块注册路由
    buildingGroup := r.Group("/building")
    SetupBuildingRouter(buildingGroup, core)

    workstationGroup := r.Group("/workstation")
    SetupWorkstationRouter(workstationGroup, core)
}
```

### 6. 权限配置

#### 菜单权限 (`sys_menu` 表)
- 父菜单：运维管理 (ID: 2)
- 子菜单：资产管理（新增一级菜单）
- 权限标识：`asset:list`, `asset:create`, `asset:update`, `asset:delete`

## 资产管理模块设计要点

### 1. 数据表设计

**表名**: `ops_asset`（使用 ops 前缀，属于运维管理模块）

**核心字段映射**:
- `id` - UUID (主键)
- `devicesn` - VARCHAR(200) (设备序列号，唯一索引，导入判断字段)
- `sequenceno` - VARCHAR(200)
- `fixassetno` - VARCHAR(100)
- `device_model_name` - VARCHAR(100)
- `device_type_name` - VARCHAR(100)
- `device_category_second_name` - VARCHAR(100)
- ... (其他字段)

**关联字段**:
- `dept_id` - UUID (关联 sys_dept.id，通过 DEPTNAME 匹配)
- `user_id` - UUID (关联 sys_user.id，通过 NOWUSER_NAME 匹配)

**标准字段**:
- `status` - INTEGER (0=正常, 1=停用)
- `created_at`, `updated_at`, `deleted_at`

### 2. 导入逻辑

基于 `DEVICESN` (设备序列号) 判断：
- 存在 → 更新记录
- 不存在 → 新增记录

### 3. 引用解析

- **部门关联**: 通过 `DEPTNAME` 匹配 `sys_dept.dept_name`
- **用户关联**: 通过 `NOWUSER_NAME` 匹配 `sys_user.nick_name`

### 4. 前端菜单

- 一级菜单：资产管理
- 菜单路径：`/asset`
- 组件路径：`pages/asset/index`

## 技术栈确认

- **后端**: Go 1.24, Gin, GORM, PostgreSQL, Excelize
- **前端**: React 19.2, TypeScript, Ant Design, Zustand
- **导入导出**: Excelize (后端), xlsx (前端)
- **架构模式**: Handler-Service, opsApi 工厂模式

## 参考文件

### 后端
- `internal/services/operations/building_service.go` - 建筑服务实现
- `internal/services/operations/excel_service.go` - Excel 导入导出服务
- `internal/services/operations/excel_config.go` - Excel 配置
- `internal/api/v1/operations/building_handler.go` - 建筑处理器
- `internal/api/v1/operations/building_router.go` - 建筑路由

### 前端
- `xingran-react-frontend/src/lib/opsApi.ts` - CRUD API 工厂
- `xingran-react-frontend/src/pages/operations/building/index.tsx` - 建筑管理页面
- `xingran-react-frontend/src/types/operations.ts` - 运维类型定义

### 数据库
- `internal/models/operations.go` - 运维模块模型定义

## 建议的实现顺序

1. **Wave 1 - 数据库与模型**: 数据库迁移、模型定义
2. **Wave 2 - 后端服务**: Service 层实现
3. **Wave 3 - 后端 API**: Handler 和 Router
4. **Wave 4 - Excel 导入导出**: Excel 配置和导入逻辑
5. **Wave 5 - 前端**: 页面组件和 API 集成
6. **Wave 6 - 菜单与权限**: 菜单配置和权限设置

## 总结

资产管理模块可以复用现有的 Operations 模块架构：
- 使用 Handler-Service 模式
- 使用 ExcelService 进行导入导出
- 使用 opsApi 工厂模式生成前端 API
- 参考 Workstation 模块的部门/用户关联逻辑

关键差异点：
- 资产表字段较多（40+ 字段）
- 需要配置完整的 Excel 导入导出字段映射
- 前端需要新建一级菜单"资产管理"
