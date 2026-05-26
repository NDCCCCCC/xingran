package addomain

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// GroupSyncHandler AD组同步API处理器
type GroupSyncHandler struct {
	adService *addomain.ADDomainService
}

// NewGroupSyncHandler 创建处理器
func NewGroupSyncHandler(adService *addomain.ADDomainService) *GroupSyncHandler {
	return &GroupSyncHandler{
		adService: adService,
	}
}

// SyncAllMembersRequest 同步所有成员请求
type SyncAllMembersRequest struct {
	ConfigID string `json:"configId" binding:"required"`
}

// CreateMappingRequest 创建映射请求
type CreateMappingRequest struct {
	DeptID      string `json:"deptId" binding:"required"`
	ADGroupID   string `json:"adGroupId" binding:"required"`
	MappingType string `json:"mappingType"`
	SyncEnabled bool   `json:"syncEnabled"`
}

// UpdateMappingRequest 更新映射请求
type UpdateMappingRequest struct {
	MappingStatus *string `json:"mappingStatus"`
	SyncEnabled   *bool   `json:"syncEnabled"`
}

// AutoMapRequest 自动映射请求
type AutoMapRequest struct {
	ConfigID string   `json:"configId" binding:"required"`
	DeptIDs  []string `json:"deptIds"`
}

// SyncAllMembers 手动触发同步所有成员
// @Summary 手动触发AD组成员同步
// @Description 同步所有有映射的部门的成员到对应的AD组
// @Tags AD Group Sync
// @Accept json
// @Produce json
// @Param request body SyncAllMembersRequest true "同步请求"
// @Success 200 {object} response.Response "同步成功"
// @Router /api/v1/ad/groups/sync [post]
func (h *GroupSyncHandler) SyncAllMembers(c *gin.Context) {
	var req SyncAllMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.adService.MemberSync.SyncAllMembers(c.Request.Context(), req.ConfigID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// GetSyncStatus 查询同步状态
// @Summary 查询AD组同步状态
// @Description 获取当前同步任务的状态和统计信息
// @Tags AD Group Sync
// @Accept json
// @Produce json
// @Success 200 {object} response.Response "查询成功"
// @Router /api/v1/ad/groups/sync/status [get]
func (h *GroupSyncHandler) GetSyncStatus(c *gin.Context) {
	// TODO: 实现从调度器获取状态的逻辑
	status := map[string]interface{}{
		"enabled":        true,
		"lastSyncTime":   nil,
		"nextSyncTime":   nil,
		"totalMappings":  0,
		"activeMappings": 0,
	}

	response.Success(c, status)
}

// ListMappings 查询部门-组映射列表
// @Summary 查询部门-组映射列表
// @Description 分页查询部门与AD组的映射关系
// @Tags AD Group Sync
// @Accept json
// @Produce json
// @Param current query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param configId query string false "AD配置ID"
// @Param deptId query string false "部门ID"
// @Param mappingType query string false "映射类型"
// @Success 200 {object} response.Response "查询成功"
// @Router /api/v1/ad/groups/mappings [get]
func (h *GroupSyncHandler) ListMappings(c *gin.Context) {
	current, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	var configID, deptID, mappingType *string
	if ci := c.Query("configId"); ci != "" {
		configID = &ci
	}
	if di := c.Query("deptId"); di != "" {
		deptID = &di
	}
	if mt := c.Query("mappingType"); mt != "" {
		mappingType = &mt
	}

	req := &addomain.ListMappingsRequest{
		ADConfigID:    configID,
		DeptID:        deptID,
		MappingType:   (*models.MappingType)(mappingType),
		MappingStatus: (*models.MappingStatus)(&[]string{"active"}[0]),
		Current:       current,
		PageSize:      pageSize,
	}

	result, err := h.adService.DeptGroupMapping.ListMappings(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// CreateMapping 创建部门-组映射
// @Summary 创建部门-组映射
// @Description 手动创建部门与AD组的映射关系
// @Tags AD Group Sync
// @Accept json
// @Produce json
// @Param request body CreateMappingRequest true "创建映射请求"
// @Success 200 {object} response.Response "创建成功"
// @Router /api/v1/ad/groups/mappings [post]
func (h *GroupSyncHandler) CreateMapping(c *gin.Context) {
	var req CreateMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	// 获取配置ID
	var configID string
	if req.MappingType == "" {
		req.MappingType = "manual"
	}

	creatorID := c.GetString("user_id")
	if creatorID == "" {
		creatorID = "system"
	}

	createReq := &addomain.CreateMappingRequest{
		DeptID:      req.DeptID,
		ADGroupID:   req.ADGroupID,
		ADConfigID:  configID, // TODO: 从上下文获取或验证
		MappingType: models.MappingType(req.MappingType),
		SyncEnabled: req.SyncEnabled,
		CreatedBy:   creatorID,
	}

	mapping, err := h.adService.DeptGroupMapping.CreateMapping(c.Request.Context(), createReq)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, mapping)
}

// GetMapping 获取单个映射详情
// @Summary 获取映射详情
// @Description 根据ID查询部门-组映射详情
// @Tags AD Group Sync
// @Accept json
// @Produce json
// @Param id path string true "映射ID"
// @Success 200 {object} response.Response "查询成功"
// @Router /api/v1/ad/groups/mappings/{id} [get]
func (h *GroupSyncHandler) GetMapping(c *gin.Context) {
	id := c.Param("id")

	mapping, err := h.adService.DeptGroupMapping.GetMapping(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "映射不存在")
		return
	}

	response.Success(c, mapping)
}

// UpdateMapping 更新部门-组映射
// @Summary 更新部门-组映射
// @Description 更新映射的状态和同步开关
// @Tags AD Group Sync
// @Accept json
// @Produce json
// @Param id path string true "映射ID"
// @Param request body UpdateMappingRequest true "更新请求"
// @Success 200 {object} response.Response "更新成功"
// @Router /api/v1/ad/groups/mappings/{id} [put]
func (h *GroupSyncHandler) UpdateMapping(c *gin.Context) {
	id := c.Param("id")
	var req UpdateMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	updaterID := c.GetString("user_id")
	if updaterID == "" {
		updaterID = "system"
	}

	updateReq := &addomain.UpdateMappingRequest{
		MappingStatus: (*models.MappingStatus)(req.MappingStatus),
		SyncEnabled:   req.SyncEnabled,
		UpdatedBy:     updaterID,
	}

	err := h.adService.DeptGroupMapping.UpdateMapping(c.Request.Context(), id, updateReq)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// DeleteMapping 删除部门-组映射
// @Summary 删除部门-组映射
// @Description 删除指定的部门-组映射关系
// @Tags AD Group Sync
// @Accept json
// @Produce json
// @Param id path string true "映射ID"
// @Success 200 {object} response.Response "删除成功"
// @Router /api/v1/ad/groups/mappings/{id} [delete]
func (h *GroupSyncHandler) DeleteMapping(c *gin.Context) {
	id := c.Param("id")

	err := h.adService.DeptGroupMapping.DeleteMapping(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// AutoMapDepartments 批量自动映射部门
// @Summary 批量自动映射部门到AD组
// @Description 根据部门名称自动匹配对应的AD组（cxhub-{dept}命名规则）
// @Tags AD Group Sync
// @Accept json
// @Produce json
// @Param request body AutoMapRequest true "自动映射请求"
// @Success 200 {object} response.Response "自动映射成功"
// @Router /api/v1/ad/groups/automap [post]
func (h *GroupSyncHandler) AutoMapDepartments(c *gin.Context) {
	var req AutoMapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	var result interface{}
	var err error

	if len(req.DeptIDs) == 0 {
		// 自动映射所有二级部门
		result, err = h.adService.DeptGroupMapping.AutoMapAllDepartments(c.Request.Context(), req.ConfigID)
	} else {
		// 映射指定部门
		// TODO: 实现批量指定部门映射
		response.Error(c, http.StatusNotImplemented, "暂未实现")
		return
	}

	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// SyncDeptMembers 同步指定部门的成员
// @Summary 同步指定部门的成员
// @Description 手动触发单个部门的成员同步
// @Tags AD Group Sync
// @Accept json
// @Produce json
// @Param deptId path string true "部门ID"
// @Success 200 {object} response.Response "同步成功"
// @Router /api/v1/ad/groups/sync/dept/{deptId} [post]
func (h *GroupSyncHandler) SyncDeptMembers(c *gin.Context) {
	deptID := c.Param("deptId")

	result, err := h.adService.MemberSync.SyncDeptMembers(c.Request.Context(), deptID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// EnsureExclusiveMembership 确保排他性成员关系
// @Summary 确保排他性成员关系
// @Description 检查并清理多组成员关系，确保每个成员只属于一个部门组
// @Tags AD Group Sync
// @Accept json
// @Produce json
// @Param configId query string true "AD配置ID"
// @Success 200 {object} response.Response "清理成功"
// @Router /api/v1/ad/groups/exclusive [post]
func (h *GroupSyncHandler) EnsureExclusiveMembership(c *gin.Context) {
	configID := c.Query("configId")
	if configID == "" {
		response.Error(c, http.StatusBadRequest, "缺少配置ID参数")
		return
	}

	result, err := h.adService.MemberSync.EnsureExclusiveMembership(c.Request.Context(), configID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}
