package v1

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

// RoleListRequest 角色列表请求
type RoleListRequest struct {
	RoleName string `json:"roleName,omitempty"`
	RoleKey  string `json:"roleKey,omitempty"`
	Status   string `json:"status,omitempty"`
	Current  int    `json:"current,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
}

// RoleCreateRequest 角色创建请求
type RoleCreateRequest struct {
	RoleName        string            `json:"roleName" binding:"required,max=50"`
	RoleKey         string            `json:"roleKey" binding:"required,max=50"`
	RoleSort        int               `json:"roleSort" binding:"min=0"`
	DataScope       models.DataScope  `json:"dataScope"`
	MenuCheckStrictly *bool           `json:"menuCheckStrictly"`
	DeptCheckStrictly *bool           `json:"deptCheckStrictly"`
	Status          models.RoleStatus `json:"status"`
	Remark          *string           `json:"remark,omitempty"`
	MenuIds         []string          `json:"menuIds,omitempty"`
	DeptIds         []string          `json:"deptIds,omitempty"`
}

// RoleUpdateRequest 角色更新请求
type RoleUpdateRequest struct {
	ID              string            `json:"id" binding:"required"`
	RoleName        string            `json:"roleName" binding:"required,max=50"`
	RoleKey         string            `json:"roleKey" binding:"required,max=50"`
	RoleSort        int               `json:"roleSort" binding:"min=0"`
	DataScope       models.DataScope  `json:"dataScope"`
	MenuCheckStrictly *bool           `json:"menuCheckStrictly"`
	DeptCheckStrictly *bool           `json:"deptCheckStrictly"`
	Status          models.RoleStatus `json:"status"`
	Remark          *string           `json:"remark,omitempty"`
	MenuIds         []string          `json:"menuIds,omitempty"`
	DeptIds         []string          `json:"deptIds,omitempty"`
}

// RoleStatusRequest 角色状态请求
type RoleStatusRequest struct {
	Status models.RoleStatus `json:"status" binding:"required"`
}

// RoleMenusRequest 角色菜单权限请求
type RoleMenusRequest struct {
	MenuIds []string `json:"menuIds" binding:"required"`
}

// listRoles 查询角色列表
// @Summary 查询角色列表
// @Description 分页查询角色列表，支持按角色名称、权限字符、状态筛选
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param request body RoleListRequest false "查询参数"
// @Success 200 {object} response.Response{data=[]models.Role}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /roles/list [post]
func listRoles(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RoleListRequest
		// 允许空的请求体，所有字段都是可选的
		if err := c.ShouldBindJSON(&req); err != nil {
			// 如果是空请求体，初始化默认值
			req.Current = 1
			req.PageSize = 10
		}

		// 设置分页默认值
		current := req.Current
		pageSize := req.PageSize
		if current <= 0 {
			current = 1
		}
		if pageSize <= 0 {
			pageSize = 10
		}

		// 构建查询条件
		db := core.DB.GetDB().Model(&models.Role{})

		// 角色名称模糊查询
		if req.RoleName != "" {
			db = db.Where("role_name LIKE ?", "%"+req.RoleName+"%")
		}

		// 权限字符模糊查询
		if req.RoleKey != "" {
			db = db.Where("role_key LIKE ?", "%"+req.RoleKey+"%")
		}

		// 状态查询
		if req.Status != "" {
			if status, err := strconv.Atoi(req.Status); err == nil {
				db = db.Where("status = ?", status)
			}
		}

		// 统计总数
		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询角色总数失败")
			return
		}

		// 查询分页数据
		var roles []models.Role
		offset := (current - 1) * pageSize
		if err := db.Offset(offset).Limit(pageSize).Order("role_sort ASC, created_at DESC").Find(&roles).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询角色列表失败")
			return
		}

		response.Page(c, roles, total, current, pageSize)
	}
}

// getAllRoles 获取所有启用角色（用于下拉选择）
// @Summary 获取所有启用角色
// @Description 获取所有启用状态的角色，用于下拉选择
// @Tags 角色管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]models.Role}
// @Router /roles/all [post]
func getAllRoles(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用 RoleCacheService 获取所有启用的角色（带缓存）
		roles, err := core.RoleCacheService.GetAllRolesWithCache(c.Request.Context())
		if err != nil {
			response.Error(c, response.ErrServerError, "查询角色列表失败")
			return
		}

		// 只返回必要的字段
		var simplifiedRoles []map[string]interface{}
		for _, role := range roles {
			simplifiedRoles = append(simplifiedRoles, map[string]interface{}{
				"id":       role.ID,
				"roleName": role.RoleName,
				"roleKey":  role.RoleKey,
			})
		}

		response.Success(c, simplifiedRoles)
	}
}

// createRole 创建角色
// @Summary 创建角色
// @Description 创建新的角色，支持设置角色名称、权限字符、数据权限、菜单权限、部门权限等
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param request body RoleCreateRequest true "角色创建信息"
// @Success 200 {object} response.Response{data=models.Role}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /roles [post]
func createRole(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RoleName        string   `json:"roleName"`
			RoleKey         string   `json:"roleKey"`
			RoleSort        int      `json:"roleSort"`
			DataScope       int      `json:"dataScope"`
			MenuCheckStrictly *bool  `json:"menuCheckStrictly"`
			DeptCheckStrictly *bool  `json:"deptCheckStrictly"`
			Status          int      `json:"status"`
			Remark          *string  `json:"remark"`
			MenuIds         []string `json:"menuIds"`
			DeptIds         []string `json:"deptIds"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			// 尝试从原始请求中读取
			var rawReq map[string]interface{}
			if err2 := c.ShouldBindJSON(&rawReq); err2 != nil {
				response.Error(c, response.ErrBadRequest, "请求参数错误: "+err.Error())
				return
			}
			// 手动转换各个字段
			if name, ok := rawReq["roleName"]; ok {
				req.RoleName = fmt.Sprintf("%v", name)
			}
			if key, ok := rawReq["roleKey"]; ok {
				req.RoleKey = fmt.Sprintf("%v", key)
			}
			if sort, ok := rawReq["roleSort"]; ok {
				switch v := sort.(type) {
				case float64:
					req.RoleSort = int(v)
				case int:
					req.RoleSort = v
				case string:
					if s, err := strconv.Atoi(v); err == nil {
						req.RoleSort = s
					}
				}
			}
			if dataScope, ok := rawReq["dataScope"]; ok {
				switch v := dataScope.(type) {
				case float64:
					req.DataScope = int(v)
				case int:
					req.DataScope = v
				case string:
					if s, err := strconv.Atoi(v); err == nil {
						req.DataScope = s
					}
				}
			}
			if menuStrictly, ok := rawReq["menuCheckStrictly"]; ok {
				switch v := menuStrictly.(type) {
				case bool:
					req.MenuCheckStrictly = &v
				case string:
					if s, err := strconv.ParseBool(v); err == nil {
						req.MenuCheckStrictly = &s
					}
				}
			}
			if deptStrictly, ok := rawReq["deptCheckStrictly"]; ok {
				switch v := deptStrictly.(type) {
				case bool:
					req.DeptCheckStrictly = &v
				case string:
					if s, err := strconv.ParseBool(v); err == nil {
						req.DeptCheckStrictly = &s
					}
				}
			}
			if status, ok := rawReq["status"]; ok {
				switch v := status.(type) {
				case float64:
					req.Status = int(v)
				case string:
					if s, err := strconv.Atoi(v); err == nil {
						req.Status = s
					}
				case int:
					req.Status = v
				case bool:
					if v {
						req.Status = 0
					} else {
						req.Status = 1
					}
				}
			}
			if remark, ok := rawReq["remark"]; ok {
				s := fmt.Sprintf("%v", remark)
				req.Remark = &s
			}
			if menuIds, ok := rawReq["menuIds"]; ok {
				if ids, ok := menuIds.([]interface{}); ok {
					for _, id := range ids {
						req.MenuIds = append(req.MenuIds, fmt.Sprintf("%v", id))
					}
				}
			}
			if deptIds, ok := rawReq["deptIds"]; ok {
				if ids, ok := deptIds.([]interface{}); ok {
					for _, id := range ids {
						req.DeptIds = append(req.DeptIds, fmt.Sprintf("%v", id))
					}
				}
			}
		}

		// 验证必填字段
		if req.RoleName == "" {
			response.Error(c, response.ErrBadRequest, "角色名称不能为空")
			return
		}
		if req.RoleKey == "" {
			response.Error(c, response.ErrBadRequest, "权限字符不能为空")
			return
		}

		// 验证状态值
		if req.Status != 0 && req.Status != 1 {
			response.Error(c, response.ErrBadRequest, "状态值无效，必须是0(启用)或1(禁用)")
			return
		}

		// 检查角色名称是否已存在
		var count int64
		if err := core.DB.GetDB().Model(&models.Role{}).Where("role_name = ?", req.RoleName).Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查角色名称失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrRecordExists, "角色名称已存在")
			return
		}

		// 检查权限字符是否已存在
		if err := core.DB.GetDB().Model(&models.Role{}).Where("role_key = ?", req.RoleKey).Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查权限字符失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrRecordExists, "权限字符已存在")
			return
		}

		// 设置默认值（固定为关联模式）
		menuCheckStrictly := true
		deptCheckStrictly := true

		// 开始事务
		tx := core.DB.GetDB().Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// 创建角色
		role := models.Role{
			RoleName:         req.RoleName,
			RoleKey:          req.RoleKey,
			RoleSort:         req.RoleSort,
			DataScope:        models.DataScope(req.DataScope),
			MenuCheckStrictly: menuCheckStrictly,
			DeptCheckStrictly: deptCheckStrictly,
			Status:           models.RoleStatus(req.Status),
			Remark:           func() string { if req.Remark != nil { return *req.Remark }; return "" }(),
		}

		if err := tx.Create(&role).Error; err != nil {
			tx.Rollback()
			response.Error(c, response.ErrServerError, "创建角色失败")
			return
		}

		// 关联菜单权限
		if len(req.MenuIds) > 0 {
			for _, menuId := range req.MenuIds {
				if err := tx.Create(&models.RoleMenu{
					RoleID: role.ID,
					MenuID: menuId,
				}).Error; err != nil {
					tx.Rollback()
					response.Error(c, response.ErrServerError, "关联菜单权限失败")
					return
				}
			}
		}

		// 关联部门权限
		if len(req.DeptIds) > 0 {
			for _, deptId := range req.DeptIds {
				if err := tx.Create(&models.RoleDept{
					RoleID: role.ID,
					DeptID: deptId,
				}).Error; err != nil {
					tx.Rollback()
					response.Error(c, response.ErrServerError, "关联部门权限失败")
					return
				}
			}
		}

		// 提交事务
		if err := tx.Commit().Error; err != nil {
			response.Error(c, response.ErrServerError, "创建角色失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"角色管理",
			1, // 1=新增
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, role)
	}
}

// getRole 获取角色详情
// @Summary 获取角色详情
// @Description 根据角色ID获取角色详细信息
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path string true "角色ID"
// @Success 200 {object} response.Response{data=models.Role}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /roles/{id} [get]
func getRole(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "角色ID不能为空")
			return
		}

		var role models.Role
		if err := core.DB.GetDB().First(&role, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "角色不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询角色失败")
			}
			return
		}

		response.Success(c, role)
	}
}

// updateRole 更新角色
// @Summary 更新角色
// @Description 更新角色信息，包括角色名称、权限字符、数据权限、菜单权限、部门权限等
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path string true "角色ID"
// @Param request body RoleUpdateRequest true "角色更新信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /roles/{id} [put]
func updateRole(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "角色ID不能为空")
			return
		}

		var req struct {
			RoleName          string  `json:"roleName"`
			RoleKey           string  `json:"roleKey"`
			RoleSort          int     `json:"roleSort"`
			DataScope         int     `json:"dataScope"`
			MenuCheckStrictly *bool   `json:"menuCheckStrictly"`
			DeptCheckStrictly *bool   `json:"deptCheckStrictly"`
			Status            int     `json:"status"`
			Remark            *string `json:"remark"`
			MenuIds           []string `json:"menuIds"`
			DeptIds           []string `json:"deptIds"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			// 尝试从原始请求中读取
			var rawReq map[string]interface{}
			if err2 := c.ShouldBindJSON(&rawReq); err2 != nil {
				response.Error(c, response.ErrBadRequest, "请求参数错误: "+err.Error())
				return
			}
			// 手动转换各个字段
			if name, ok := rawReq["roleName"]; ok {
				req.RoleName = fmt.Sprintf("%v", name)
			}
			if key, ok := rawReq["roleKey"]; ok {
				req.RoleKey = fmt.Sprintf("%v", key)
			}
			if sort, ok := rawReq["roleSort"]; ok {
				switch v := sort.(type) {
				case float64:
					req.RoleSort = int(v)
				case int:
					req.RoleSort = v
				case string:
					if s, err := strconv.Atoi(v); err == nil {
						req.RoleSort = s
					}
				}
			}
			if dataScope, ok := rawReq["dataScope"]; ok {
				switch v := dataScope.(type) {
				case float64:
					req.DataScope = int(v)
				case int:
					req.DataScope = v
				case string:
					if s, err := strconv.Atoi(v); err == nil {
						req.DataScope = s
					}
				}
			}
			if menuStrictly, ok := rawReq["menuCheckStrictly"]; ok {
				switch v := menuStrictly.(type) {
				case bool:
					req.MenuCheckStrictly = &v
				case string:
					if s, err := strconv.ParseBool(v); err == nil {
						req.MenuCheckStrictly = &s
					}
				}
			}
			if deptStrictly, ok := rawReq["deptCheckStrictly"]; ok {
				switch v := deptStrictly.(type) {
				case bool:
					req.DeptCheckStrictly = &v
				case string:
					if s, err := strconv.ParseBool(v); err == nil {
						req.DeptCheckStrictly = &s
					}
				}
			}
			if status, ok := rawReq["status"]; ok {
				switch v := status.(type) {
				case float64:
					req.Status = int(v)
				case string:
					if s, err := strconv.Atoi(v); err == nil {
						req.Status = s
					}
				case int:
					req.Status = v
				case bool:
					if v {
						req.Status = 0
					} else {
						req.Status = 1
					}
				}
			}
			if remark, ok := rawReq["remark"]; ok {
				s := fmt.Sprintf("%v", remark)
				req.Remark = &s
			}
			if menuIds, ok := rawReq["menuIds"]; ok {
				if ids, ok := menuIds.([]interface{}); ok {
					for _, id := range ids {
						req.MenuIds = append(req.MenuIds, fmt.Sprintf("%v", id))
					}
				}
			}
			if deptIds, ok := rawReq["deptIds"]; ok {
				if ids, ok := deptIds.([]interface{}); ok {
					for _, id := range ids {
						req.DeptIds = append(req.DeptIds, fmt.Sprintf("%v", id))
					}
				}
			}
		}

		// 验证必填字段
		if req.RoleName == "" {
			response.Error(c, response.ErrBadRequest, "角色名称不能为空")
			return
		}
		if req.RoleKey == "" {
			response.Error(c, response.ErrBadRequest, "权限字符不能为空")
			return
		}

		// 验证状态值
		if req.Status != 0 && req.Status != 1 {
			response.Error(c, response.ErrBadRequest, "状态值无效，必须是0(启用)或1(禁用)")
			return
		}

		// 检查角色是否存在
		var role models.Role
		if err := core.DB.GetDB().First(&role, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "角色不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询角色失败")
			}
			return
		}

		// 检查角色名称是否与其他角色重复
		var count int64
		if err := core.DB.GetDB().Model(&models.Role{}).Where("role_name = ? AND id != ?", req.RoleName, id).Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查角色名称失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrRecordExists, "角色名称已存在")
			return
		}

		// 检查权限字符是否与其他角色重复
		if err := core.DB.GetDB().Model(&models.Role{}).Where("role_key = ? AND id != ?", req.RoleKey, id).Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查权限字符失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrRecordExists, "权限字符已存在")
			return
		}

		// 设置默认值（固定为关联模式）
		menuCheckStrictly := true
		deptCheckStrictly := true

		// 开始事务
		tx := core.DB.GetDB().Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// 更新角色信息
		updates := map[string]interface{}{
			"role_name":          req.RoleName,
			"role_key":           req.RoleKey,
			"role_sort":          req.RoleSort,
			"data_scope":         models.DataScope(req.DataScope),
			"menu_check_strictly": menuCheckStrictly,
			"dept_check_strictly": deptCheckStrictly,
			"status":             models.RoleStatus(req.Status),
			"remark":             func() string { if req.Remark != nil { return *req.Remark }; return "" }(),
		}

		if err := tx.Model(&role).Updates(updates).Error; err != nil {
			tx.Rollback()
			response.Error(c, response.ErrServerError, "更新角色失败")
			return
		}

		// 删除原有的菜单权限关联
		if err := tx.Where("role_id = ?", id).Delete(&models.RoleMenu{}).Error; err != nil {
			tx.Rollback()
			response.Error(c, response.ErrServerError, "删除原菜单权限关联失败")
			return
		}

		// 重新关联菜单权限
		if len(req.MenuIds) > 0 {
			for _, menuId := range req.MenuIds {
				if err := tx.Create(&models.RoleMenu{
					RoleID: id,
					MenuID: menuId,
				}).Error; err != nil {
					tx.Rollback()
					response.Error(c, response.ErrServerError, "关联菜单权限失败")
					return
				}
			}
		}

		// 删除原有的部门权限关联
		if err := tx.Where("role_id = ?", id).Delete(&models.RoleDept{}).Error; err != nil {
			tx.Rollback()
			response.Error(c, response.ErrServerError, "删除原部门权限关联失败")
			return
		}

		// 重新关联部门权限
		if len(req.DeptIds) > 0 {
			for _, deptId := range req.DeptIds {
				if err := tx.Create(&models.RoleDept{
					RoleID: id,
					DeptID: deptId,
				}).Error; err != nil {
					tx.Rollback()
					response.Error(c, response.ErrServerError, "关联部门权限失败")
					return
				}
			}
		}

		// 提交事务
		if err := tx.Commit().Error; err != nil {
			response.Error(c, response.ErrServerError, "更新角色失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"角色管理",
			2, // 2=修改
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteRole 删除角色
// @Summary 删除角色
// @Description 删除指定角色（需确保没有用户使用该角色）
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path string true "角色ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /roles/{id} [delete]
func deleteRole(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "角色ID不能为空")
			return
		}

		// 检查角色是否存在
		var role models.Role
		if err := core.DB.GetDB().First(&role, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "角色不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询角色失败")
			}
			return
		}

		// 检查是否有用户使用该角色
		var count int64
		if err := core.DB.GetDB().Table("sys_user_roles").Where("role_id = ?", id).Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查角色使用情况失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrBadRequest, "该角色已分配给用户，无法删除")
			return
		}

		// 开始事务
		tx := core.DB.GetDB().Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// 删除角色菜单关联
		if err := tx.Where("role_id = ?", id).Delete(&models.RoleMenu{}).Error; err != nil {
			tx.Rollback()
			response.Error(c, response.ErrServerError, "删除角色菜单关联失败")
			return
		}

		// 删除角色部门关联
		if err := tx.Where("role_id = ?", id).Delete(&models.RoleDept{}).Error; err != nil {
			tx.Rollback()
			response.Error(c, response.ErrServerError, "删除角色部门关联失败")
			return
		}

		// 删除角色
		if err := tx.Delete(&role).Error; err != nil {
			tx.Rollback()
			response.Error(c, response.ErrServerError, "删除角色失败")
			return
		}

		// 提交事务
		if err := tx.Commit().Error; err != nil {
			response.Error(c, response.ErrServerError, "删除角色失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"角色管理",
			3, // 3=删除
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// batchDeleteRoles 批量删除角色
// @Summary 批量删除角色
// @Description 批量删除多个角色（需确保没有用户使用这些角色）
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string} true "角色ID列表"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /roles/batch [delete]
func batchDeleteRoles(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			IDs []string `json:"ids" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		if len(req.IDs) == 0 {
			response.Error(c, response.ErrBadRequest, "请选择要删除的角色")
			return
		}

		// 检查是否有用户使用这些角色
		var count int64
		if err := core.DB.GetDB().Table("sys_user_roles").Where("role_id IN ?", req.IDs).Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查角色使用情况失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrBadRequest, "存在已分配给用户的角色，无法删除")
			return
		}

		// 开始事务
		tx := core.DB.GetDB().Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// 删除角色菜单关联
		if err := tx.Where("role_id IN ?", req.IDs).Delete(&models.RoleMenu{}).Error; err != nil {
			tx.Rollback()
			response.Error(c, response.ErrServerError, "删除角色菜单关联失败")
			return
		}

		// 删除角色部门关联
		if err := tx.Where("role_id IN ?", req.IDs).Delete(&models.RoleDept{}).Error; err != nil {
			tx.Rollback()
			response.Error(c, response.ErrServerError, "删除角色部门关联失败")
			return
		}

		// 删除角色
		if err := tx.Where("id IN ?", req.IDs).Delete(&models.Role{}).Error; err != nil {
			tx.Rollback()
			response.Error(c, response.ErrServerError, "删除角色失败")
			return
		}

		// 提交事务
		if err := tx.Commit().Error; err != nil {
			response.Error(c, response.ErrServerError, "批量删除角色失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"角色管理",
			3, // 3=删除
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, gin.H{"message": "批量删除成功"})
	}
}

// updateRoleStatus 更改角色状态
// @Summary 更改角色状态
// @Description 启用或禁用指定角色
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path string true "角色ID"
// @Param request body RoleStatusRequest true "角色状态"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /roles/{id}/status [put]
func updateRoleStatus(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "角色ID不能为空")
			return
		}

		var req struct {
			Status int `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			// 尝试从原始请求中读取
			var rawReq map[string]interface{}
			if err2 := c.ShouldBindJSON(&rawReq); err2 != nil {
				response.Error(c, response.ErrBadRequest, "请求参数错误: "+err.Error())
				return
			}
			// 尝试转换status值
			if statusVal, ok := rawReq["status"]; ok {
				switch v := statusVal.(type) {
				case float64:
					req.Status = int(v)
				case string:
					if v == "0" || v == "1" {
						req.Status = 0
					} else {
						req.Status = 1
					}
				case int:
					req.Status = v
				case bool:
					if v {
						req.Status = 0
					} else {
						req.Status = 1
					}
				default:
					response.Error(c, response.ErrBadRequest, "status字段格式错误")
					return
				}
			} else {
				response.Error(c, response.ErrBadRequest, "缺少status字段")
				return
			}
		}

		// 验证status的值
		if req.Status != 0 && req.Status != 1 {
			response.Error(c, response.ErrBadRequest, "状态值无效，必须是0(启用)或1(禁用)")
			return
		}

		// 检查角色是否存在
		var role models.Role
		if err := core.DB.GetDB().First(&role, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "角色不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询角色失败")
			}
			return
		}

		// 更新状态
		if err := core.DB.GetDB().Model(&role).Update("status", models.RoleStatus(req.Status)).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新角色状态失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"角色管理",
			2, // 2=修改
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, gin.H{"message": "更新状态成功"})
	}
}

// updateRoleMenus 更新角色菜单权限
// @Summary 更新角色菜单权限
// @Description 更新角色的菜单权限，支持分配或取消分配菜单权限
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path string true "角色ID"
// @Param request body RoleMenusRequest true "菜单ID列表"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /roles/{id}/menus [put]
func updateRoleMenus(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "角色ID不能为空")
			return
		}

		var req struct {
			MenuIds []string `json:"menuIds"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			// 尝试从原始请求中读取
			var rawReq map[string]interface{}
			if err2 := c.ShouldBindJSON(&rawReq); err2 != nil {
				response.Error(c, response.ErrBadRequest, "请求参数错误: "+err.Error())
				return
			}
			// 尝试转换menuIds
			if menuIds, ok := rawReq["menuIds"]; ok {
				switch v := menuIds.(type) {
				case []interface{}:
					for _, id := range v {
						req.MenuIds = append(req.MenuIds, fmt.Sprintf("%v", id))
					}
				case []string:
					req.MenuIds = v
				case string:
					// 如果是字符串，可能是逗号分隔的ID
					if v != "" {
						ids := strings.Split(v, ",")
						for _, id := range ids {
							req.MenuIds = append(req.MenuIds, strings.TrimSpace(id))
						}
					}
				}
			}
		}

		// 检查角色是否存在
		var role models.Role
		if err := core.DB.GetDB().First(&role, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "角色不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询角色失败")
			}
			return
		}

		// 开始事务
		tx := core.DB.GetDB().Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// 删除原有的菜单权限关联
		if err := tx.Where("role_id = ?", id).Delete(&models.RoleMenu{}).Error; err != nil {
			tx.Rollback()
			response.Error(c, response.ErrServerError, "删除原菜单权限关联失败")
			return
		}

		// 重新关联菜单权限
		if len(req.MenuIds) > 0 {
			for _, menuId := range req.MenuIds {
				if err := tx.Create(&models.RoleMenu{
					RoleID: id,
					MenuID: menuId,
				}).Error; err != nil {
					tx.Rollback()
					response.Error(c, response.ErrServerError, "关联菜单权限失败")
					return
				}
			}
		}

		// 提交事务
		if err := tx.Commit().Error; err != nil {
			response.Error(c, response.ErrServerError, "更新菜单权限失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"角色管理",
			4, // 4=授权
			"",
			c.Request.Method,
			operURL,
			getUsernameFromContext(c),
			getDeptNameFromContext(c, core.GetDB()),
			&clientIP,
			nil,
			nil,
			nil,
			0,
			0,
		)

		response.Success(c, gin.H{"message": "菜单权限分配成功"})
	}
}