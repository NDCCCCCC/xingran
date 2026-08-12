package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

type BuildingHandler struct {
	service          opsServices.BuildingService
	geocodingService *opsServices.GeocodingService
	core             *core.Core
}

func NewBuildingHandler(service opsServices.BuildingService, geocodingService *opsServices.GeocodingService) *BuildingHandler {
	return &BuildingHandler{service: service, geocodingService: geocodingService}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
// 链式注入而非改写 NewBuildingHandler 签名以保持既有调用点（router.go）兼容。
func (h *BuildingHandler) WithCore(core *core.Core) *BuildingHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Statistics 楼宇统计(读操作,不记操作日志;支持筛选参数透传)
func (h *BuildingHandler) Statistics(c *gin.Context) {
	var params map[string]interface{}
	_ = c.ShouldBindJSON(&params)
	result, err := h.service.Statistics(c.Request.Context(), params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// SearchBuildingOptions 楼宇下拉数据源(name LIKE 模糊 + orgId/status 筛选,LIMIT 50,读操作不写操作日志)
// 设计给前端 Select/AutoComplete 远程搜索,避免分页 list 当全集源导致 >1000 楼宇后截断。
func (h *BuildingHandler) SearchBuildingOptions(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		params = map[string]interface{}{}
	}
	result, err := h.service.SearchBuildingOptions(c.Request.Context(), params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// NewBuildingHandlerWithCore 使用 core 创建 BuildingHandler（向后兼容）
func NewBuildingHandlerWithCore(service opsServices.BuildingService, core *core.Core) *BuildingHandler {
	return NewBuildingHandler(service, opsServices.NewGeocodingService(core.Config.Baidu.MapAK))
}

// Create 创建楼宇
// @Summary 创建楼宇
// @Description 创建新的楼宇信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body operations.OpsBuilding true "楼宇信息"
// @Success 200 {object} response.Response{data=operations.OpsBuilding}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/building [post]
func (h *BuildingHandler) Create(c *gin.Context) {
	var building operations.OpsBuilding
	if !handleJSONBinding(c, &building) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &building), "创建") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "楼宇管理", operlog.OperTypeCreate)
	response.Success(c, building)
}

// List 查询楼宇列表
// @Summary 查询楼宇列表
// @Description 分页查询楼宇列表，支持按条件筛选
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body object{current=int,pageSize=int,name=string,orgId=string,status=int} true "查询参数"
// @Success 200 {object} response.Response{data=object{list=[]operations.OpsBuilding,total=int}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/building/list [post]
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
// @Summary 获取楼宇详情
// @Description 根据ID获取楼宇详细信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "楼宇ID"
// @Success 200 {object} response.Response{data=operations.OpsBuilding}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /ops/building/{id} [post]
func (h *BuildingHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	building, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.BuildingNotFound())
		return
	}

	response.Success(c, building)
}

// Update 更新楼宇
// @Summary 更新楼宇
// @Description 更新楼宇信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "楼宇ID"
// @Param request body operations.OpsBuilding true "楼宇信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/building/{id}/update [post]
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "楼宇管理", operlog.OperTypeUpdate)
	response.Success(c, nil)
}

// Delete 删除楼宇
// @Summary 删除楼宇
// @Description 根据ID删除楼宇
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "楼宇ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/building/{id}/delete [post]
func (h *BuildingHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "楼宇管理", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// BatchOperation 批量操作
// @Summary 批量操作
// @Description 对楼宇进行批量操作，如批量删除
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string,action=string} true "批量操作参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/building/batch [post]
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "楼宇管理", operlog.OperTypeBatch)
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
		response.Error(c, apperrors.ParamMissing("地址"))
		return
	}

	// 调用地理编码服务
	lng, lat, err := h.geocodingService.Geocode(c.Request.Context(), req.Address)
	if !handleServiceError(c, err, "地址解析") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "楼宇管理", operlog.OperTypeOther,
		operlog.WithOperParam(`{"address":"`+req.Address+`"}`))
	response.Success(c, GeocodeResponse{
		Longitude: lng,
		Latitude:  lat,
	})
}
