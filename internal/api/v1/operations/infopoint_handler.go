package operations

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/operations/requests"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
)

type InfoPointHandler struct {
	service opsServices.InfoPointService
}

func NewInfoPointHandler(service opsServices.InfoPointService) *InfoPointHandler {
	return &InfoPointHandler{service: service}
}

// Create 创建信息点
// @Summary 创建信息点
// @Description 创建新的信息点
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body operations.OpsInfoPoint true "信息点"
// @Success 200 {object} response.Response{data=operations.OpsInfoPoint}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/infoPoint [post]
func (h *InfoPointHandler) Create(c *gin.Context) {
	var infoPoint operations.OpsInfoPoint
	if !handleJSONBinding(c, &infoPoint) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &infoPoint), "创建") {
		return
	}

	response.Success(c, infoPoint)
}

// List 查询信息点列表（类型安全版本）
// @Summary 查询信息点列表
// @Description 分页查询信息点列表，支持按条件筛选
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body requests.InfoPointListRequest true "查询参数"
// @Success 200 {object} response.Response{data=object{list=[]operations.OpsInfoPoint,total=int}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/infoPoint/list [post]
func (h *InfoPointHandler) List(c *gin.Context) {
	var req requests.InfoPointListRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	result, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("查询失败"))
		return
	}

	response.Success(c, result)
}

// GetByID 获取信息点详情
// @Summary 获取信息点详情
// @Description 根据ID获取信息点详细信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "信息点ID"
// @Success 200 {object} response.Response{data=operations.OpsInfoPoint}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /ops/infoPoint/{id} [post]
func (h *InfoPointHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	infoPoint, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.InfoPointNotFound())
		return
	}

	response.Success(c, infoPoint)
}

// Update 更新信息点
// @Summary 更新信息点
// @Description 更新信息点信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "信息点ID"
// @Param request body operations.OpsInfoPoint true "信息点"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/infoPoint/{id}/update [post]
func (h *InfoPointHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var infoPoint operations.OpsInfoPoint
	if !handleJSONBinding(c, &infoPoint) {
		return
	}

	infoPoint.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &infoPoint), "更新") {
		return
	}

	response.Success(c, nil)
}

// Delete 删除信息点
// @Summary 删除信息点
// @Description 根据ID删除信息点
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "信息点ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/infoPoint/{id}/delete [post]
func (h *InfoPointHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	response.Success(c, nil)
}

// BatchOperation 批量操作（类型安全版本）
// @Summary 批量操作
// @Description 对信息点进行批量操作，如批量删除
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body requests.InfoPointBatchOperationRequest true "批量操作参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/infoPoint/batch [post]
func (h *InfoPointHandler) BatchOperation(c *gin.Context) {
	var req requests.InfoPointBatchOperationRequest
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
