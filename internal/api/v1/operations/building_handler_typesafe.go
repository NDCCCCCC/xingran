package operations

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// BuildingHandlerTypeSafe 楼宇处理器（类型安全版本）
// 这是新的类型安全 Handler 实现，使用专门的请求结构体替代 map[string]interface{}
type BuildingHandlerTypeSafe struct {
	service          opsServices.BuildingServiceTypeSafe
	geocodingService *opsServices.GeocodingService
}

func NewBuildingHandlerTypeSafe(service opsServices.BuildingServiceTypeSafe, geocodingService *opsServices.GeocodingService) *BuildingHandlerTypeSafe {
	return &BuildingHandlerTypeSafe{service: service, geocodingService: geocodingService}
}

// NewBuildingHandlerTypeSafeWithCore 使用 core 创建 BuildingHandler（向后兼容）
func NewBuildingHandlerTypeSafeWithCore(service opsServices.BuildingServiceTypeSafe, core *core.Core) *BuildingHandlerTypeSafe {
	return NewBuildingHandlerTypeSafe(service, opsServices.NewGeocodingService(core.Config.Baidu.MapAK))
}

// Create 创建楼宇
func (h *BuildingHandlerTypeSafe) Create(c *gin.Context) {
	var building operations.OpsBuilding
	if !handleJSONBinding(c, &building) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &building), "创建") {
		return
	}

	response.Success(c, building)
}

// List 查询楼宇列表（类型安全版本）
// 不再使用 map[string]interface{}，而是使用专门的请求结构体
func (h *BuildingHandlerTypeSafe) List(c *gin.Context) {
	var req requests.BuildingListRequest

	// 直接绑定到类型安全的请求结构体
	if !handleJSONBinding(c, &req) {
		return
	}

	// Service 接收类型安全的请求结构体
	result, err := h.service.List(c.Request.Context(), req)
	if !handleServiceError(c, err, "查询") {
		return
	}

	response.Success(c, result)
}

// GetByID 获取楼宇详情
func (h *BuildingHandlerTypeSafe) GetByID(c *gin.Context) {
	id := c.Param("id")
	building, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.BuildingNotFound())
		return
	}

	response.Success(c, building)
}

// Update 更新楼宇
func (h *BuildingHandlerTypeSafe) Update(c *gin.Context) {
	id := c.Param("id")
	var building operations.OpsBuilding
	if !handleJSONBinding(c, &building) {
		return
	}

	building.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &building), "更新") {
		return
	}

	response.Success(c, nil)
}

// Delete 删除楼宇
func (h *BuildingHandlerTypeSafe) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	response.Success(c, nil)
}

// BatchOperation 批量操作（类型安全版本）
func (h *BuildingHandlerTypeSafe) BatchOperation(c *gin.Context) {
	var req requests.BuildingBatchOperationRequest

	// 绑定到类型安全的请求结构体
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

// Geocode 地址解析：将详细地址转换为经纬度坐标
// GeocodeRequest 和 GeocodeResponse 已在 building_handler.go 中定义
func (h *BuildingHandlerTypeSafe) Geocode(c *gin.Context) {
	var req GeocodeRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	if req.Address == "" {
		response.Error(c, apperrors.ParamMissing("地址"))
		return
	}

	// 调用地理编码服务
	lng, lat, err := h.geocodingService.Geocode(c.Request.Context(), req.Address)
	if !handleServiceError(c, err, "地址解析") {
		return
	}

	response.Success(c, GeocodeResponse{
		Longitude: lng,
		Latitude:  lat,
	})
}

// ==============================================================================
// 对比：旧版本 vs 新版本
// ==============================================================================

// ❌ 旧版本：使用 map[string]interface{}
// func (h *BuildingHandler) List(c *gin.Context) {
//     var params map[string]interface{}
//     if !handleJSONBinding(c, &params) {
//         return
//     }
//     result, err := h.service.List(c.Request.Context(), params)
//     // ...
// }

// ✅ 新版本：使用类型安全的请求结构体
// func (h *BuildingHandlerTypeSafe) List(c *gin.Context) {
//     var req requests.BuildingListRequest
//     if !handleJSONBinding(c, &req) {
//         return
//     }
//     result, err := h.service.List(c.Request.Context(), req)
//     // ...
// }
