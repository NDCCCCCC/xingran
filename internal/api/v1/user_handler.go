package v1

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// parseInt 安全地将字符串转换为整数
func parseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}

// getIntField 从map中获取整数字段，提供默认值
func getIntField(m map[string]interface{}, key string, defaultValue int) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case string:
			return parseInt(v)
		}
	}
	return defaultValue
}

// UserListRequest 用户列表请求
type UserListRequest struct {
	Username  *string `json:"username,omitempty"`
	Nickname  *string `json:"nickname,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	Status    *int    `json:"status,omitempty"`
	DeptID    *string `json:"deptId,omitempty"`
	BeginTime *string `json:"beginTime,omitempty"`
	EndTime   *string `json:"endTime,omitempty"`
	Current   int     `json:"current" binding:"min=1"`
	PageSize  int     `json:"pageSize" binding:"min=1,max=100"`
}

// UserCreateRequest 创建用户请求
type UserCreateRequest struct {
	Username     string  `json:"username" binding:"required,min=2,max=20"`
	Password     string  `json:"password" binding:"required,min=6,max=20"`
	Nickname     *string `json:"nickname,omitempty"`
	EmployeeNo   *string `json:"employeeNo,omitempty"` // 工号（企业微信代码）
	Email        *string `json:"email,omitempty" binding:"omitempty,email"`
	Phone        *string `json:"phone,omitempty"`
	Gender       int     `json:"gender" binding:"min=0,max=2"`
	Status       int     `json:"status" binding:"min=0,max=1,oneof=0 1"`
	DeptID       *string `json:"deptId,omitempty"`
	RoleIds      []string `json:"roleIds,omitempty"`
	PostIds      []string `json:"postIds,omitempty"`
	Remark       *string `json:"remark,omitempty"`
}

// UserUpdateRequest 更新用户请求
type UserUpdateRequest struct {
	ID           string  `json:"id,omitempty"`
	Nickname     *string `json:"nickname,omitempty"`
	EmployeeNo   *string `json:"employeeNo,omitempty"` // 工号（企业微信代码）
	Email        *string `json:"email,omitempty" binding:"omitempty,email"`
	Phone        *string `json:"phone,omitempty"`
	Gender       int     `json:"gender" binding:"min=0,max=2"`
	Status       int     `json:"status" binding:"min=0,max=1,oneof=0 1"`
	DeptID       *string `json:"deptId,omitempty"`
	RoleIds      []string `json:"roleIds,omitempty"`
	PostIds      []string `json:"postIds,omitempty"`
	Remark       *string `json:"remark,omitempty"`
}

// UserStatusRequest 用户状态请求
type UserStatusRequest struct {
	ID     string `json:"id" binding:"required"`
	Status int    `json:"status" binding:"min=0,max=1"`
}

// UserDeleteRequest 删除用户请求
type UserDeleteRequest struct {
	Ids []string `json:"ids" binding:"required,min=1"`
}


// listUsers 查询用户列表
// @Summary 查询用户列表
// @Description 分页查询用户列表，支持多条件过滤
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body UserListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /users/list [post]
func listUsers(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawReq map[string]interface{}
		// 允许空的请求体，设置默认值
		if err := c.ShouldBindJSON(&rawReq); err != nil {
			// 如果是空请求体，初始化默认值
			rawReq = make(map[string]interface{})
		}

		// 转换为结构体并处理类型转换
		req := UserListRequest{
			Current:  getIntField(rawReq, "current", 1),
			PageSize: getIntField(rawReq, "pageSize", 10),
		}

		// 处理字符串字段
		if val, ok := rawReq["username"].(string); ok && val != "" {
			req.Username = &val
		}
		if val, ok := rawReq["nickname"].(string); ok && val != "" {
			req.Nickname = &val
		}
		if val, ok := rawReq["phone"].(string); ok && val != "" {
			req.Phone = &val
		}
		if val, ok := rawReq["deptId"].(string); ok && val != "" {
			req.DeptID = &val
		}
		if val, ok := rawReq["beginTime"].(string); ok && val != "" {
			req.BeginTime = &val
		}
		if val, ok := rawReq["endTime"].(string); ok && val != "" {
			req.EndTime = &val
		}

		// 处理状态字段 - 支持字符串和数字类型
		if statusVal, ok := rawReq["status"]; ok && statusVal != nil {
			var status int
			switch v := statusVal.(type) {
			case string:
				if v == "0" || v == "1" {
					status = parseInt(v)
				}
			case float64:
				status = int(v)
			case int:
				status = v
			}
			req.Status = &status
		}

		// 构建查询条件
		db := core.DB.GetDB().Model(&models.User{})

		// 应用数据权限过滤
		// 对于用户表，使用 "dept_id" 字段进行过滤
		db = middleware.ApplyDataScopeFromContext(c, db, core, "dept_id")

		// 用户名过滤
		if req.Username != nil && *req.Username != "" {
			db = db.Where("username LIKE ?", "%"+*req.Username+"%")
		}

		// 昵称过滤
		if req.Nickname != nil && *req.Nickname != "" {
			db = db.Where("nickname LIKE ?", "%"+*req.Nickname+"%")
		}

		// 手机号过滤
		if req.Phone != nil && *req.Phone != "" {
			db = db.Where("phone = ?", *req.Phone)
		}

		// 状态过滤
		if req.Status != nil {
			db = db.Where("status = ?", *req.Status)
		}

		// 部门过滤
		if req.DeptID != nil && *req.DeptID != "" {
			db = db.Where("dept_id = ?", *req.DeptID)
		}

		// 时间范围过滤
		if req.BeginTime != nil && *req.BeginTime != "" {
			db = db.Where("created_at >= ?", *req.BeginTime)
		}
		if req.EndTime != nil && *req.EndTime != "" {
			db = db.Where("created_at <= ?", *req.EndTime)
		}

		// 计算总数
		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询用户总数失败")
			return
		}

		// 分页查询
		var users []models.User
		offset := (req.Current - 1) * req.PageSize
		if err := db.Offset(offset).Limit(req.PageSize).Find(&users).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询用户列表失败")
			return
		}

		// 查询所有角色（用于显示角色名称）
		var allRoles []models.Role
		core.DB.GetDB().Find(&allRoles)
		roleMap := make(map[string]string)
		for _, role := range allRoles {
			roleMap[role.ID] = role.RoleName
		}

		// 加载每个用户的角色和部门信息并构建响应
		var userWithDept []map[string]interface{}
		for _, user := range users {
			// 查询用户角色
			var userRoles []models.UserRole
			var roleNames []string
			var roleIds []string
			if err := core.DB.GetDB().Where("user_id = ?", user.ID).Find(&userRoles).Error; err == nil {
				for _, ur := range userRoles {
					if roleName, ok := roleMap[ur.RoleID]; ok {
						roleNames = append(roleNames, roleName)
						roleIds = append(roleIds, ur.RoleID)
					}
				}
			}

			userMap := map[string]interface{}{
				"id":         user.ID,
				"username":   user.Username,
				"nickname":   user.Nickname,
				"employeeNo": user.EmployeeNo, // 工号（企业微信代码）
				"email":      user.Email,
				"phone":      user.Phone,
				"gender":     user.Gender,
				"status":     user.Status,
				"deptId":     user.DeptID,
				"deptName":   "", // 默认值
				"roles":      roleNames, // 角色名称数组
				"roleIds":    roleIds,  // 角色ID数组
				"remark":     user.Remark,
				"createdAt":  user.CreatedAt,
				"updatedAt":  user.UpdatedAt,
			}

			// 如果有部门ID，查询部门名称
			if user.DeptID != nil && *user.DeptID != "" {
				var dept models.Department
				if err := core.DB.GetDB().First(&dept, "id = ?", *user.DeptID).Error; err == nil {
					userMap["deptName"] = dept.DeptName
				}
			}

			userWithDept = append(userWithDept, userMap)
		}

		// 构建响应
		pageResp := response.PageResponse{
			List:     userWithDept,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}

		response.Success(c, pageResp)
	}
}

// createUser 创建用户
// @Summary 创建用户
// @Description 创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body UserCreateRequest true "用户信息"
// @Success 200 {object} response.Response
// @Router /users [post]
func createUser(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UserCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 检查用户名是否已存在
		var existUser models.User
		if err := core.DB.GetDB().Where("username = ?", req.Username).First(&existUser).Error; err == nil {
			response.Error(c, response.ErrBadRequest, "用户名已存在")
			return
		}

		// 验证部门ID是否存在
		if req.DeptID != nil && *req.DeptID != "" {
			var dept models.Department
			if err := core.DB.GetDB().Where("id = ?", *req.DeptID).First(&dept).Error; err != nil {
				response.Error(c, response.ErrBadRequest, "部门不存在")
				return
			}
		}

		// 密码加密
		pwdManager := core.PwdManager
		hashedPassword, err := pwdManager.HashPassword(req.Password)
		if err != nil {
			response.Error(c, response.ErrServerError, "密码加密失败")
			return
		}

		// 性别字段：前后端枚举值已统一，0=男,1=女,2=保密

		// 创建用户
		user := models.User{
			BaseModel: models.BaseModel{},
			Username:  req.Username,
			Password:  hashedPassword,
			Nickname:  req.Nickname,
			EmployeeNo: req.EmployeeNo, // 工号（企业微信代码）
			Email:     req.Email,
			Phone:     req.Phone,
			Gender:    models.Gender(req.Gender),
			Status:    models.UserStatus(req.Status),
			DeptID:    req.DeptID,
			DeptName: func() *string {
				if req.DeptID != nil && *req.DeptID != "" {
					var dept models.Department
					if err := core.DB.GetDB().Where("id = ?", *req.DeptID).First(&dept).Error; err == nil {
						return &dept.DeptName
					}
				}
				return nil
			}(),
			Remark: func() string { if req.Remark != nil { return *req.Remark }; return "" }(),
		}

		if err := core.DB.GetDB().Create(&user).Error; err != nil {
			response.Error(c, response.ErrServerError, "创建用户失败")
			return
		}

		// 处理用户角色关联
		if len(req.RoleIds) > 0 {
			// 创建用户角色关联
			for _, roleId := range req.RoleIds {
				userRole := models.UserRole{
					UserID: user.ID,
					RoleID: roleId,
				}
				if err := core.DB.GetDB().Create(&userRole).Error; err != nil {
					// 记录错误但不回滚用户创建
					continue
				}
			}
		}

		// 清除用户缓存
		if err := core.UserCacheService.InvalidateUserCache(c.Request.Context(), user.ID); err != nil {
			// 记录错误但不影响响应
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"用户管理",
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

		response.Success(c, gin.H{
			"id": user.ID,
		})
	}
}

// getUser 获取用户详情
// @Summary 获取用户详情
// @Description 根据用户ID获取用户详细信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response{data=models.User}
// @Router /users/{id} [post]
func getUser(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")
		if userID == "" {
			response.Error(c, response.ErrBadRequest, "用户ID不能为空")
			return
		}

		// 使用 UserCacheService 获取用户（带缓存）
		user, err := core.UserCacheService.GetUserByIDWithCache(c.Request.Context(), userID)
		if err != nil {
			response.Error(c, response.ErrNotFound, "用户不存在")
			return
		}

		response.Success(c, user)
	}
}

// updateUser 更新用户
// @Summary 更新用户
// @Description 更新用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param request body UserUpdateRequest true "用户信息"
// @Success 200 {object} response.Response
// @Router /users/{id}/update [post]
func updateUser(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")
		if userID == "" {
			response.Error(c, response.ErrBadRequest, "用户ID不能为空")
			return
		}

		var req struct {
			Nickname   *string  `json:"nickname,omitempty"`
			EmployeeNo *string  `json:"employeeNo,omitempty"` // 工号（企业微信代码）
			Email      *string  `json:"email,omitempty" binding:"omitempty,email"`
			Phone      *string  `json:"phone,omitempty"`
			Gender     int      `json:"gender"`
			Status     int      `json:"status"`
			DeptID     *string  `json:"deptId,omitempty"`
			RoleIds    []string `json:"roleIds,omitempty"`
			Remark     *string  `json:"remark,omitempty"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误: "+err.Error())
			return
		}

		// 验证 gender 和 status 的值
		if req.Gender < 0 || req.Gender > 2 {
			response.Error(c, response.ErrBadRequest, "性别值无效，必须是0(男)、1(女)或2(保密)")
			return
		}
		if req.Status != 0 && req.Status != 1 {
			response.Error(c, response.ErrBadRequest, "状态值无效，必须是0(启用)或1(禁用)")
			return
		}

		// 检查用户是否存在
		var user models.User
		if err := core.DB.GetDB().Where("id = ?", userID).First(&user).Error; err != nil {
			response.Error(c, response.ErrNotFound, "用户不存在")
			return
		}

		// 验证部门ID是否存在
		if req.DeptID != nil && *req.DeptID != "" {
			var dept models.Department
			if err := core.DB.GetDB().Where("id = ?", *req.DeptID).First(&dept).Error; err != nil {
				response.Error(c, response.ErrBadRequest, "部门不存在")
				return
			}
		}

		// 更新用户信息
		updates := map[string]interface{}{}

		// 只更新非空字段
		if req.Nickname != nil {
			updates["nickname"] = req.Nickname
		}
		if req.EmployeeNo != nil {
			updates["employee_no"] = req.EmployeeNo
		}
		if req.Email != nil {
			updates["email"] = req.Email
		}
		if req.Phone != nil {
			updates["phone"] = req.Phone
		}
		// 性别字段：前后端枚举值已统一，0=男,1=女,2=保密
		updates["gender"] = models.Gender(req.Gender)

		// 状态字段保持不变：0=启用,1=禁用
		updates["status"] = models.UserStatus(req.Status)

		// 处理部门更新（包括清空部门的情况）
		if req.DeptID != nil {
			if *req.DeptID == "" {
				// 清空部门
				emptyStr := ""
				updates["dept_id"] = nil
				updates["dept_name"] = &emptyStr
			} else {
				// 设置新部门
				updates["dept_id"] = req.DeptID
				// 同时更新部门名称
				var dept models.Department
				if err := core.DB.GetDB().Where("id = ?", *req.DeptID).First(&dept).Error; err == nil {
					updates["dept_name"] = dept.DeptName
				}
			}
		}
		if req.Remark != nil {
			updates["remark"] = req.Remark
		}

		if err := core.DB.GetDB().Model(&user).Updates(updates).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新用户失败: "+err.Error())
			return
		}

		// 更新用户角色关联
		if len(req.RoleIds) > 0 {
			// 先删除原有的角色关联
			if err := core.DB.GetDB().Where("user_id = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
				// 记录错误但不中断流程
			}

			// 创建新的角色关联
			for _, roleId := range req.RoleIds {
				userRole := models.UserRole{
					UserID: userID,
					RoleID: roleId,
				}
				if err := core.DB.GetDB().Create(&userRole).Error; err != nil {
					// 记录错误但不中断流程
					continue
				}
			}
		} else {
			// 如果没有传递角色ID，删除所有角色关联
			core.DB.GetDB().Where("user_id = ?", userID).Delete(&models.UserRole{})
		}

		// 清除用户缓存
		if err := core.UserCacheService.InvalidateUserCache(c.Request.Context(), userID); err != nil {
			// 记录错误但不影响响应
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"用户管理",
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

		response.Success(c, nil)
	}
}

// deleteUser 删除用户
// @Summary 删除用户
// @Description 删除指定用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response
// @Router /users/{id}/delete [post]
func deleteUser(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")
		if userID == "" {
			response.Error(c, response.ErrBadRequest, "用户ID不能为空")
			return
		}

		// 检查用户是否存在
		var user models.User
		if err := core.DB.GetDB().Where("id = ?", userID).First(&user).Error; err != nil {
			response.Error(c, response.ErrNotFound, "用户不存在")
			return
		}

		// 软删除用户
		if err := core.DB.GetDB().Delete(&user).Error; err != nil {
			response.Error(c, response.ErrServerError, "删除用户失败")
			return
		}

		// 清除用户缓存
		if err := core.UserCacheService.InvalidateUserCache(c.Request.Context(), userID); err != nil {
			// 记录错误但不影响响应
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"用户管理",
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

		// TODO: 删除用户角色和岗位关联

		response.Success(c, nil)
	}
}

// batchDeleteUsers 批量删除用户
// @Summary 批量删除用户
// @Description 批量删除多个用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body UserDeleteRequest true "用户ID列表"
// @Success 200 {object} response.Response
// @Router /users/batch-delete [post]
func batchDeleteUsers(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UserDeleteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 批量软删除用户
		if err := core.DB.GetDB().Where("id IN ?", req.Ids).Delete(&models.User{}).Error; err != nil {
			response.Error(c, response.ErrServerError, "批量删除用户失败")
			return
		}

		// 清除所有用户缓存
		if err := core.UserCacheService.InvalidateAllUserCache(c.Request.Context()); err != nil {
			// 记录错误但不影响响应
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"用户管理",
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

		// TODO: 删除用户角色和岗位关联

		response.Success(c, nil)
	}
}

// updateUserStatus 更改用户状态
// @Summary 更改用户状态
// @Description 启用或禁用用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param request body UserStatusRequest true "状态信息"
// @Success 200 {object} response.Response
// @Router /users/{id}/status [post]
func updateUserStatus(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")
		if userID == "" {
			response.Error(c, response.ErrBadRequest, "用户ID不能为空")
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

		// 检查用户是否存在
		var user models.User
		if err := core.DB.GetDB().Where("id = ?", userID).First(&user).Error; err != nil {
			response.Error(c, response.ErrNotFound, "用户不存在")
			return
		}

		// 更新用户状态
		if err := core.DB.GetDB().Model(&user).Update("status", models.UserStatus(req.Status)).Error; err != nil {
			response.Error(c, response.ErrServerError, "更新用户状态失败")
			return
		}

		// 清除用户缓存
		if err := core.UserCacheService.InvalidateUserCache(c.Request.Context(), userID); err != nil {
			// 记录错误但不影响响应
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"用户管理",
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

// resetPassword 重置密码
// @Summary 重置密码
// @Description 重置用户密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response
// @Router /users/{id}/reset-password [post]
func resetPassword(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")
		if userID == "" {
			response.Error(c, response.ErrBadRequest, "用户ID不能为空")
			return
		}

		var req struct {
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 验证密码
		if req.Password == "" {
			response.Error(c, response.ErrBadRequest, "密码不能为空")
			return
		}
		if len(req.Password) < 6 {
			response.Error(c, response.ErrBadRequest, "密码长度不能少于6位")
			return
		}
		if len(req.Password) > 20 {
			response.Error(c, response.ErrBadRequest, "密码长度不能超过20位")
			return
		}

		// 检查用户是否存在
		var user models.User
		if err := core.DB.GetDB().Where("id = ?", userID).First(&user).Error; err != nil {
			response.Error(c, response.ErrNotFound, "用户不存在")
			return
		}

		// 密码加密
		pwdManager := core.PwdManager
		hashedPassword, err := pwdManager.HashPassword(req.Password)
		if err != nil {
			response.Error(c, response.ErrServerError, "密码加密失败")
			return
		}

		// 更新密码
		if err := core.DB.GetDB().Model(&user).Updates(map[string]interface{}{
			"password":        hashedPassword,
			"pwd_update_time": time.Now(),
		}).Error; err != nil {
			response.Error(c, response.ErrServerError, "重置密码失败")
			return
		}

		// 记录操作日志
		clientIP := c.ClientIP()
		operURL := c.Request.URL.String()
		core.OperLogService.RecordAsync(
			core.GetDB(),
			"用户管理",
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

		response.Success(c, gin.H{
			"message": "密码重置成功",
		})
	}
}

// exportUsers 导出用户
// @Summary 导出用户
// @Description 导出用户数据到Excel
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body UserListRequest true "查询条件"
// @Success 200 {file} file
// @Router /users/export [post]
func exportUsers(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现用户导出功能
		response.Error(c, response.ErrNotImplemented, "功能暂未实现")
	}
}

// importUsers 导入用户
// @Summary 导入用户
// @Description 从Excel导入用户数据
// @Tags 用户管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel文件"
// @Success 200 {object} response.Response
// @Router /users/import [post]
func importUsers(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现用户导入功能
		response.Error(c, response.ErrNotImplemented, "功能暂未实现")
	}
}

// downloadTemplate 下载导入模板
// @Summary 下载导入模板
// @Description 下载用户导入Excel模板
// @Tags 用户管理
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Success 200 {file} file
// @Router /users/template [get]
func downloadTemplate(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现模板下载功能
		response.Error(c, response.ErrNotImplemented, "功能暂未实现")
	}
}