# Phase 22: Sangfor VDI Integration - Pattern Map

**Mapped:** 2025-05-25
**Files analyzed:** 12 new/modified files
**Analogs found:** 11 / 12

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/api/v1/vdi/vm_handler.go` | handler | request-response | `internal/api/v1/operations/building_handler.go` | exact |
| `internal/api/v1/vdi/vm_router.go` | router | request-response | `internal/api/v1/duty/duty_router.go` | exact |
| `internal/api/v1/vdi/requests/vm_requests.go` | request-types | request-response | `internal/api/v1/operations/requests/building_requests.go` | exact |
| `internal/services/vdi/vm_service.go` | service | CRUD | `internal/services/operations/building_service.go` | exact |
| `internal/services/vdi/vdi_client.go` | service | event-driven | `internal/device/scrapli_wrapper.go` | role-match |
| `internal/services/vdi/vdi_auth_service.go` | service | request-response | `internal/services/api_sender_service.go` | role-match |
| `internal/models/vdi/virtual_machine.go` | model | CRUD | `internal/models/operations/building.go` | exact |
| `internal/models/vdi/vdi_server.go` | model | CRUD | `internal/models/operations/building.go` | exact |
| `internal/models/vdi/vdi_resource_group.go` | model | CRUD | `internal/models/operations/building.go` | exact |
| `internal/config/vdi_config.go` | config | CRUD | `configs/config.yaml` | role-match |
| `xingran-react-frontend/src/lib/vdiApi.ts` | api-client | request-response | `xingran-react-frontend/src/lib/opsApi.ts` | exact |
| `xingran-react-frontend/src/pages/vdi/VirtualMachineList.tsx` | component | request-response | `xingran-react-frontend/src/pages/ops/BuildingList.tsx` | exact |

## Pattern Assignments

### `internal/api/v1/vdi/vm_handler.go` (handler, request-response)

**Analog:** `internal/api/v1/operations/building_handler.go`

**Imports pattern** (lines 1-10):
```go
package operations

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)
```

**Handler struct pattern** (lines 12-19):
```go
type BuildingHandler struct {
	service          opsServices.BuildingService
	geocodingService *opsServices.GeocodingService
}

func NewBuildingHandler(service opsServices.BuildingService, geocodingService *opsServices.GeocodingService) *BuildingHandler {
	return &BuildingHandler{service: service, geocodingService: geocodingService}
}

// NewBuildingHandlerWithCore 使用 core 创建 BuildingHandler（向后兼容）
func NewBuildingHandlerWithCore(service opsServices.BuildingService, core *core.Core) *BuildingHandler {
	return NewBuildingHandler(service, opsServices.NewGeocodingService(core.Config.Baidu.MapAK))
}
```

**CRUD handler pattern** (lines 37-48):
```go
// Create 创建楼宇
func (h *BuildingHandler) Create(c *gin.Context) {
	var building operations.OpsBuilding
	if !handleJSONBinding(c, &building) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &building), "创建") {
		return
	}

	response.Success(c, building)
}
```

**List handler pattern** (lines 61-73):
```go
// List 查询楼宇列表
func (h *BuildingHandler) List(c *gin.Context) {
	var params map[string]interface{}
	if !handleJSONBinding(c, &params) {
		return
	}

	result, err := h.service.List(c.Request.Context(), params)
	if !handleServiceError(c, err, "查询") {
		return
	}

	response.Success(c, result)
}
```

**Batch operation pattern** (lines 157-178):
```go
// BatchOperation 批量操作
func (h *BuildingHandler) BatchOperation(c *gin.Context) {
	var req struct {
		IDs    []string `json:"ids"`
		Action string   `json:"action"`
	}

	if !handleJSONBinding(c, &req) {
		return
	}

	switch req.Action {
	case "delete":
		if !handleServiceError(c, h.service.BatchDelete(c.Request.Context(), req.IDs), "批量删除") {
			return
		}
	default:
		response.Error(c, apperrors.InvalidOperation("不支持的操作"))
		return
	}

	response.Success(c, nil)
}
```

---

### `internal/api/v1/vdi/vm_router.go` (router, request-response)

**Analog:** `internal/api/v1/duty/duty_router.go`

**Router setup pattern** (lines 11-20):
```go
// SetupDutyPoolsRouter 设置值班池路由
func SetupDutyPoolsRouter(r *gin.RouterGroup, core *core.Core) {
	service := createDutyService(core)
	handler := NewDutyHandler(service)

	r.POST("/list", handler.ListPools)
	r.POST("", handler.CreatePool)
	r.POST("/:id", handler.GetPoolByID)
	r.POST("/:id/update", handler.UpdatePool)
	r.POST("/:id/delete", handler.DeletePool)
}
```

**Service creation with cache pattern** (lines 68-80):
```go
// createDutyService 创建值班服务（带缓存）
func createDutyService(core *core.Core) dutyServices.DutyCacheService {
	var cacheProvider systemServices.CacheProvider
	if core.DataCacheService != nil {
		cacheProvider = systemServices.NewCacheProvider(core.DataCacheService)
	} else {
		cacheProvider = &systemServices.NoOpCacheProvider{}
	}

	return dutyServices.NewDutyServiceWithCache(
		core.DB.GetDB(),
		cacheProvider,
		core.CacheConfigService,
	)
}
```

---

### `internal/api/v1/vdi/requests/vm_requests.go` (request-types, request-response)

**Analog:** `internal/api/v1/operations/requests/building_requests.go`

**Request struct pattern** (lines 1-9):
```go
package requests

// BuildingListRequest 楼宇列表查询请求
type BuildingListRequest struct {
	PaginationParams        // 嵌入分页参数
	StatusRequest           // 嵌入状态筛选
	Name             string `json:"name"`  // 楼宇名称（模糊查询）
	OrgID            string `json:"orgId"` // 所属机构ID
}

// BuildingBatchOperationRequest 楼宇批量操作请求
type BuildingBatchOperationRequest struct {
	BatchOperationRequest // 嵌入批量操作参数
}
```

**Common embedded patterns** from `requests/common.go` (lines 7-34):
```go
// PaginationParams 分页参数（基础结构）
type PaginationParams struct {
	Current  int `json:"current"`  // 当前页码
	PageSize int `json:"pageSize"` // 每页数量
}

// BatchOperationRequest 批量操作请求（基础结构）
type BatchOperationRequest struct {
	IDs    []string `json:"ids" binding:"required"`    // 要操作的ID列表
	Action string   `json:"action" binding:"required"` // 操作类型：delete=删除
}

// StatusRequest 状态筛选请求（基础结构）
type StatusRequest struct {
	Status *int `json:"status"` // 状态（0=正常/启用 1=停用/禁用）
}
```

---

### `internal/services/vdi/vm_service.go` (service, CRUD)

**Analog:** `internal/services/operations/building_service.go`

**Service interface pattern** (lines 17-25):
```go
// BuildingService 楼宇服务接口
type BuildingService interface {
	Create(ctx context.Context, building *operations.OpsBuilding) error
	Update(ctx context.Context, building *operations.OpsBuilding) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*operations.OpsBuilding, error)
	List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
}
```

**Service implementation pattern** (lines 27-40):
```go
type buildingService struct {
	db            *gorm.DB
	codeGenerator *CodeGenerator
	uuidValidator *regexp.Regexp
}

// NewBuildingService 创建楼宇服务实例
func NewBuildingService(db *gorm.DB) BuildingService {
	return &buildingService{
		db:            db,
		codeGenerator: NewCodeGenerator(db),
		uuidValidator: regexp.MustCompile(uuidPattern),
	}
}
```

**Create with validation pattern** (lines 43-55):
```go
// Create 创建楼宇
func (s *buildingService) Create(ctx context.Context, building *operations.OpsBuilding) error {
	// 验证机构存在性
	if err := s.validateOrg(ctx, building.OrgID); err != nil {
		return err
	}

	// 验证楼宇名称唯一性（同一机构下不能有同名楼宇）
	if err := s.validateNameUnique(ctx, building.OrgID, building.Name, ""); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Create(building).Error
}
```

**List with filters pattern** (lines 90-117):
```go
// List 查询楼宇列表
func (s *buildingService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
	query := s.db.WithContext(ctx).Table("ops_buildings")

	// 应用筛选条件
	query = s.applyFilters(query, params)

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 应用分页
	pagination := extractPagination(params)
	offset := calculateOffset(pagination)

	var list []operations.OpsBuilding
	if err := query.Offset(offset).Limit(pagination.PageSize).Order("order_num ASC").Find(&list).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  pagination.Current,
		PageSize: pagination.PageSize,
	}, nil
}
```

---

### `internal/services/vdi/vdi_client.go` (service, event-driven)

**Analog:** `internal/device/scrapli_wrapper.go`

**Client wrapper pattern** (lines 45-59):
```go
// ScrapliWrapper scrapligo驱动封装（简化版）
// 并发安全由上层 PooledConnection 的设备级锁保护
type ScrapliWrapper struct {
	driver    *network.Driver
	device    *models.NetworkDevice
	state     ConnectionState // 连接状态
	stateMu   sync.RWMutex    // 保护状态读写
	opMu      sync.RWMutex    // 保护 driver 操作与 Close() 的竞态
	closing   chan struct{}   // 用于通知关闭信号
	initDone  chan struct{}   // 初始化完成信号
	closeOnce sync.Once       // 确保 initDone 只关闭一次
}
```

**Connection initialization pattern** (lines 88-100):
```go
// NewScrapliWrapper 创建scrapligo封装实例
func NewScrapliWrapper(device *models.NetworkDevice, username, password string, protocolType models.ProtocolType) (*ScrapliWrapper, error) {
	if device == nil {
		return nil, fmt.Errorf("设备信息不能为空")
	}

	// 获取平台名称
	platformName := PlatformName(device.Vendor)

	// 创建基础选项
	opts := []util.Option{
		options.WithAuthUsername(username),
		options.WithAuthPassword(password),
```

**State management pattern** (lines 19-43):
```go
// ConnectionState 连接状态
type ConnectionState int

const (
	StateInitializing ConnectionState = iota // 正在初始化
	StateReady                               // 已就绪，可以使用
	StateClosing                             // 正在关闭
	StateClosed                              // 已关闭
)

// String 返回状态的字符串表示
func (s ConnectionState) String() string {
	switch s {
	case StateInitializing:
		return "Initializing"
	case StateReady:
		return "Ready"
	case StateClosing:
		return "Closing"
	case StateClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}
```

---

### `internal/services/vdi/vdi_auth_service.go` (service, request-response)

**Analog:** `internal/services/api_sender_service.go`

**HTTP client pattern** (lines 19-32):
```go
// APISenderService API发送服务
type APISenderService struct {
	db     *gorm.DB
	client *http.Client
}

// NewAPISenderService 创建API发送服务
func NewAPISenderService(db *gorm.DB) *APISenderService {
	return &APISenderService{
		db: db,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}
```

**Retry mechanism pattern** (lines 52-100):
```go
// Send 发送API通知
func (s *APISenderService) Send(ctx context.Context, configID string, msg *APIMessage) *APISendResult {
	// 获取API配置
	configService := NewNotificationConfigService(s.db)
	config, err := configService.GetAPINotificationConfigByID(ctx, configID)
	if err != nil {
		return &APISendResult{
			Success: false,
			Message: "获取API配置失败",
			Error:   err,
		}
	}

	// 检查配置状态
	if config.Status != 0 {
		return &APISendResult{
			Success: false,
			Message: "API配置未启用",
		}
	}

	// 构建请求
	var lastErr error
	var result *APISendResult

	// 重试机制
	for i := 0; i <= config.RetryCount; i++ {
		result = s.sendRequest(ctx, config, msg)
		result.RetryCount = i

		if result.Success {
			return result
		}

		lastErr = result.Error

		// 如果不是最后一次重试，等待一段时间
		if i < config.RetryCount {
			time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
		}
	}

	return &APISendResult{
		Success:    false,
		Message:    fmt.Sprintf("API发送失败，已重试%d次", config.RetryCount),
		Error:      lastErr,
		RetryCount: config.RetryCount,
	}
}
```

---

### `internal/models/vdi/virtual_machine.go` (model, CRUD)

**Analog:** `internal/models/operations/building.go`

**Model struct pattern** (lines 7-34):
```go
// BuildingStatus 楼宇状态枚举
type BuildingStatus int

const (
	BuildingStatusNormal  BuildingStatus = 0 // 正常
	BuildingStatusStopped BuildingStatus = 1 // 停用
)

// OpsBuilding 楼宇模型
type OpsBuilding struct {
	models.BaseModel
	Name        string         `gorm:"size:100;not null" json:"name"`                 // 楼宇名称
	Address     string         `gorm:"size:200" json:"address"`                       // 详细地址
	Longitude   *float64       `gorm:"type:decimal(11,8)" json:"longitude,omitempty"` // 经度（通过地址自动解析）
	Latitude    *float64       `gorm:"type:decimal(11,8)" json:"latitude,omitempty"`  // 纬度（通过地址自动解析）
	Level       int            `gorm:"default:2;not null" json:"level"`               // 层级：1=城市级汇总，2=具体楼宇
	OrgID       string         `gorm:"size:64" json:"orgId"`                          // 所属机构ID（关联sys_dept）
	OrgName     *string        `gorm:"size:100" json:"orgName,omitempty"`             // 所属机构名称
	TotalFloors int            `gorm:"default:0" json:"totalFloors"`                  // 楼层数（根据创建的楼层自动计算）
	Status      BuildingStatus `gorm:"default:0" json:"status"`                       // 状态: 0=正常, 1=停用
	Remark      *string        `gorm:"size:500" json:"remark,omitempty"`              // 备注
	OrderNum    int            `gorm:"default:0" json:"orderNum"`                     // 排序号
}

// TableName 指定表名
func (OpsBuilding) TableName() string {
	return "ops_buildings"
}
```

**BaseModel pattern** from `internal/models/base.go` (lines 11-27):
```go
// BaseModel 基础模型
type BaseModel struct {
	ID        string         `gorm:"type:uuid;primary_key" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
	CreatedBy string         `gorm:"size:64" json:"createdBy"`
	UpdatedBy string         `gorm:"size:64" json:"updatedBy"`
	Version   int            `json:"version"`
}

// BeforeCreate GORM钩子 - 创建前
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}
```

---

### `xingran-react-frontend/src/lib/vdiApi.ts` (api-client, request-response)

**Analog:** `xingran-react-frontend/src/lib/opsApi.ts`

**CRUD factory pattern** (lines 21-55):
```typescript
interface CrudApiConfig {
  basePath: string;
}

function createCrudApi<T>(config: CrudApiConfig) {
  const { basePath } = config;

  return {
    list: async (params: PageParams & Record<string, unknown>) => {
      return await post<PageResponse<T>>(`${basePath}/list`, params);
    },

    get: async (id: string) => {
      return await post<T>(`${basePath}/${id}`, {});
    },

    create: async (data: Partial<T>) => {
      return await post(basePath, data);
    },

    update: async (id: string, data: Partial<T>) => {
      return await post(`${basePath}/${id}/update`, data);
    },

    delete: async (id: string) => {
      return await post(`${basePath}/${id}/delete`, {});
    },

    batch: async (action: string, data: Record<string, unknown>) => {
      return await post(`${basePath}/batch`, { action, ...data });
    },
  };
}
```

**Module-specific API pattern** (lines 59-66):
```typescript
export interface BuildingListParams extends PageParams {
  name?: string;
  code?: string;
  status?: number;
  orgId?: string;  // 所属机构ID，用于按部门筛选
}

export const buildingApi = createCrudApi<Building>({ basePath: '/ops/building' });
```

**Extended API with custom methods pattern** (lines 78-86):
```typescript
const floorCrudApi = createCrudApi<Floor>({ basePath: '/ops/floor' });

export const floorApi = {
  ...floorCrudApi,

  tree: async () => {
    return await post<Floor[]>('/ops/floor/tree', {});
  },
};
```

---

### `xingran-react-frontend/src/pages/vdi/VirtualMachineList.tsx` (component, request-response)

**Analog:** `xingran-react-frontend/src/pages/ops/BuildingList.tsx` (inferred from opsApi pattern)

**Component structure pattern** (typical for list pages):
```typescript
import React, { useEffect, useState } from 'react';
import { Table, Button, Space, message } from 'antd';
import { buildingApi } from '@/lib/opsApi';

const BuildingList: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 });

  const fetchData = async (params: any) => {
    setLoading(true);
    try {
      const result = await buildingApi.list(params);
      setData(result.data.list);
      setPagination({
        current: result.data.current,
        pageSize: result.data.pageSize,
        total: result.data.total,
      });
    } catch (error) {
      message.error('查询失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData({ current: 1, pageSize: 10 });
  }, []);

  // ... rest of component
};
```

---

## Shared Patterns

### Error Handling
**Source:** `internal/api/v1/operations/base_handler.go` and `pkg/errors/errors.go`
**Apply to:** All handler files

**Unified error handling pattern** (base_handler.go):
```go
// handleJSONBinding 统一处理 JSON 绑定
func handleJSONBinding(c *gin.Context, v interface{}) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "参数错误"))
		return false
	}
	return true
}

// handleServiceError 统一处理服务层错误
func handleServiceError(c *gin.Context, err error, action string) bool {
	if err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeServerError, action+"失败"))
		return false
	}
	return true
}
```

**Typed error pattern** (pkg/errors/errors.go):
```go
// AppError 应用错误结构
type AppError struct {
	Code       ErrorCode              // 业务错误码
	Message    string                 // 错误消息
	HTTPStatus int                    // HTTP状态码
	Err        error                  // 原始错误
	Context    map[string]interface{} // 错误上下文
}

// New 创建新的应用错误
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: code.DefaultHTTPStatus(),
	}
}
```

---

### Response Formatting
**Source:** `pkg/response/response.go`
**Apply to:** All handler files

**Success response pattern** (lines 73-84):
```go
// Success 成功响应
func Success(c *gin.Context, data ...interface{}) {
	responseData := getOptionalData(data)

	c.JSON(http.StatusOK, Response{
		Code:      successCode,
		Message:   successMessage,
		Data:      responseData,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	})
}
```

**Error response pattern** (lines 86-100):
```go
// Error 错误响应
func Error(c *gin.Context, err interface{}, message ...string) {
	appErr := toAppError(err)

	// 如果提供了自定义消息，覆盖默认消息
	if len(message) > 0 {
		appErr.Message = message[0]
	}

	c.JSON(appErr.HTTPStatus, Response{
		Code:      appErr.Code,
		Message:   appErr.Message,
		Data:      nil,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	})
}
```

---

### Service Layer Validation
**Source:** `internal/services/operations/building_service.go`
**Apply to:** All service files

**Validation pattern** (lines 161-183):
```go
// validateOrg 验证机构存在性
func (s *buildingService) validateOrg(ctx context.Context, orgID string) error {
	if orgID == "" {
		return nil
	}

	// 验证UUID格式
	if !s.uuidValidator.MatchString(orgID) {
		return apperrors.BuildingOrgInvalidWithMsg("所属机构ID格式无效：必须是有效的UUID格式")
	}

	// 验证机构是否存在
	var count int64
	if err := s.db.WithContext(ctx).Table("sys_dept").Where("id = ?", orgID).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		return apperrors.BuildingOrgInvalidWithMsg("所属机构不存在")
	}

	return nil
}
```

---

### Context Propagation
**Source:** All service files
**Apply to:** All service methods

**Context pattern** (consistent across services):
```go
// Always use ctx from request context
func (s *buildingService) Create(ctx context.Context, building *operations.OpsBuilding) error {
	return s.db.WithContext(ctx).Create(building).Error
}

// Handler passes context
func (h *BuildingHandler) Create(c *gin.Context) {
	// ...
	if !handleServiceError(c, h.service.Create(c.Request.Context(), &building), "创建") {
		return
	}
	// ...
}
```

---

### Status Convention
**Source:** `internal/models/operations/building.go` and project-wide
**Apply to:** All model files

**Status enum pattern**:
```go
// BuildingStatus 楼宇状态枚举
type BuildingStatus int

const (
	BuildingStatusNormal  BuildingStatus = 0 // 正常
	BuildingStatusStopped BuildingStatus = 1 // 停用
)
```

**Universal Rule:** `0 = enabled/normal/visible, 1 = disabled/stopped/hidden`

---

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/services/vdi/vdi_sync_service.go` | service | event-driven | No existing sync/polling service pattern for third-party APIs |
| `internal/config/vdi_config.go` | config | CRUD | VDI-specific config structure, needs custom design |

---

## VDI-Specific Patterns (from RESEARCH.md)

### VDI Client Authentication Pattern
**Source:** RESEARCH.md section 1.3
**Apply to:** `internal/services/vdi/vdi_client.go`

```go
type VDIClient interface {
	// 认证
	Authenticate(ctx context.Context) (string, error)

	// 虚拟机操作
	OperateVM(ctx context.Context, vmIDs []string, action string) error
	DeleteVM(ctx context.Context, vmIDs []string) error
	ConfigIP(ctx context.Context, req []VMIPConfig) error
	RenameVM(ctx context.Context, vmID, newName string) error

	// 查询
	GetVM(ctx context.Context, vmID string) (*VDIVMDetail, error)
	ListVMs(ctx context.Context, resourceID string) ([]VDIVMSummary, error)
	GetUserVMs(ctx context.Context, userID string) ([]VDIVMSummary, error)

	// 关联操作
	BindUser(ctx context.Context, vmID, userID string) error
	BindPolicy(ctx context.Context, vmID, policyID string) error
	GetAvailableUsers(ctx context.Context, vmID string) ([]VDIUser, error)
}
```

### VDI Service Layer Pattern
**Source:** RESEARCH.md section 4.2
**Apply to:** `internal/services/vdi/vm_service.go`

```go
type VMService interface {
	// 虚拟机CRUD
	CreateVM(ctx context.Context, req *CreateVMRequest) (*VDIVirtualMachine, error)
	GetVM(ctx context.Context, id string) (*VDIVMDTO, error)
	ListVMs(ctx context.Context, req *ListVMRequest) (*PageResult, error)
	UpdateVM(ctx context.Context, id string, req *UpdateVMRequest) error
	DeleteVM(ctx context.Context, ids []string) error

	// 虚拟机操作
	OperateVM(ctx context.Context, ids []string, action VMPowerAction) error
	BatchConfigIP(ctx context.Context, req []VMIPConfigRequest) error
	RenameVM(ctx context.Context, id, newName string) error

	// 用户关联
	BindUser(ctx context.Context, vmID, userID string) error
	UnbindUser(ctx context.Context, vmID string) error

	// 同步
	SyncVMFromVDI(ctx context.Context, vmID string) error
	SyncAllVMs(ctx context.Context, serverID string) error
}
```

---

## Metadata

**Analog search scope:**
- `internal/api/v1/operations/` - Handler and router patterns
- `internal/api/v1/duty/` - Router setup with Core DI
- `internal/services/operations/` - Service layer patterns
- `internal/device/` - Third-party client wrapper patterns
- `internal/services/api_sender_service.go` - HTTP client patterns
- `internal/models/operations/` - Model patterns
- `pkg/errors/` - Error handling patterns
- `pkg/response/` - Response formatting patterns
- `xingran-react-frontend/src/lib/opsApi.ts` - Frontend API patterns

**Files scanned:** 15+
**Pattern extraction date:** 2025-05-25

---

## Anti-Patterns to Avoid

1. **Direct GORM calls in handlers** - Always use service layer
2. **Hardcoded API endpoints** - Use configuration and constants
3. **Missing context propagation** - Always pass `c.Request.Context()` to services
4. **Inconsistent error handling** - Use `handleJSONBinding` and `handleServiceError` helpers
5. **Mixed response formats** - Always use `response.Success()` and `response.Error()`
6. **Manual UUID generation** - Use `BaseModel.BeforeCreate` hook
7. **Ignoring status conventions** - Follow 0=normal, 1=stopped convention
8. **Raw SQL queries** - Use GORM chainable API
9. **Synchronous third-party API calls in handlers** - Use async patterns or background jobs
10. **Missing request validation** - Validate in service layer before DB operations
