package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

type BuildingHandler struct {
	service         opsServices.BuildingService
	geocodingService *opsServices.GeocodingService
}

func NewBuildingHandler(service opsServices.BuildingService, geocodingService *opsServices.GeocodingService) *BuildingHandler {
	return &BuildingHandler{service: service, geocodingService: geocodingService}
}

// NewBuildingHandlerWithCore 使用 core 创建 BuildingHandler（向后兼容）
func NewBuildingHandlerWithCore(service opsServices.BuildingService, core *core.Core) *BuildingHandler {
	return NewBuildingHandler(service, opsServices.NewGeocodingService(core.Config.Baidu.MapAK))
}

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
func (h *BuildingHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	response.Success(c, nil)
}

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
		response.Error(c, http.StatusBadRequest, "不支持的操作")
		return
	}

	response.Success(c, nil)
}

// GeocodeRequest 地址解析请求
type GeocodeRequest struct {
	Address string `json:"address" binding:"required"`
	City    string `json:"city"`
}

// GeocodeResponse 地址解析响应
type GeocodeResponse struct {
	Longitude        float64 `json:"longitude"`
	Latitude         float64 `json:"latitude"`
	FormattedAddress string  `json:"formattedAddress,omitempty"`
	Province         string  `json:"province,omitempty"`
	City             string  `json:"city,omitempty"`
	District         string  `json:"district,omitempty"`
	Street           string  `json:"street,omitempty"`
}

// Geocode 地址解析：将详细地址转换为经纬度坐标
// @Summary 地址解析
// @Description 将详细地址转换为经纬度坐标（使用百度地图 API）
// @Tags 楼宇管理
// @Accept json
// @Produce json
// @Param request body GeocodeRequest true "地址解析请求"
// @Success 200 {object} response.Response{data=GeocodeResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/building/geocode [post]
func (h *BuildingHandler) Geocode(c *gin.Context) {
	var req GeocodeRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	if req.Address == "" {
		response.Error(c, http.StatusBadRequest, "地址不能为空")
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
