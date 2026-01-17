package v1

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/utils"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

// ==================== 辅助函数 ====================

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

// ==================== 请求/响应类型 ====================

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
	Username   string   `json:"username" binding:"required,min=2,max=20"`
	Password   string   `json:"password" binding:"required,min=6,max=20"`
	Nickname   *string  `json:"nickname,omitempty"`
	EmployeeNo *string  `json:"employeeNo,omitempty"` // 工号（企业微信代码）
	Email      *string  `json:"email,omitempty" binding:"omitempty,email"`
	Phone      *string  `json:"phone,omitempty"`
	Gender     int      `json:"gender" binding:"min=0,max=2"`
	Status     int      `json:"status" binding:"min=0,max=1,oneof=0 1"`
	DeptID     *string  `json:"deptId,omitempty"`
	RoleIds    []string `json:"roleIds,omitempty"`
	PostIds    []string `json:"postIds,omitempty"`
	Remark     *string  `json:"remark,omitempty"`
}

// UserUpdateRequest 更新用户请求
type UserUpdateRequest struct {
	ID         string   `json:"id,omitempty"`
	Nickname   *string  `json:"nickname,omitempty"`
	EmployeeNo *string  `json:"employeeNo,omitempty"` // 工号（企业微信代码）
	Email      *string  `json:"email,omitempty" binding:"omitempty,email"`
	Phone      *string  `json:"phone,omitempty"`
	Gender     int      `json:"gender" binding:"min=0,max=2"`
	Status     int      `json:"status" binding:"min=0,max=1,oneof=0 1"`
	DeptID     *string  `json:"deptId,omitempty"`
	RoleIds    []string `json:"roleIds,omitempty"`
	PostIds    []string `json:"postIds,omitempty"`
	Remark     *string  `json:"remark,omitempty"`
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

// ==================== 基本 CRUD 操作 ====================

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
		if val, ok := rawReq["status"]; ok {
			req.Status = new(int)
			switch v := val.(type) {
			case float64:
				*req.Status = int(v)
			case int:
				*req.Status = v
			case string:
				*req.Status = parseInt(v)
			}
		}

		// 构建查询条件
		db := core.DB.GetDB().Model(&models.User{})

		// 应用数据权限过滤
		db = middleware.ApplyDataScopeFromContext(c, db, core, "id")

		// 添加筛选条件
		if req.Username != nil && *req.Username != "" {
			db = db.Where("username LIKE ?", "%"+*req.Username+"%")
		}
		if req.Nickname != nil && *req.Nickname != "" {
			db = db.Where("nickname LIKE ?", "%"+*req.Nickname+"%")
		}
		if req.Phone != nil && *req.Phone != "" {
			db = db.Where("phone LIKE ?", "%"+*req.Phone+"%")
		}
		if req.Status != nil {
			db = db.Where("status = ?", *req.Status)
		}
		if req.DeptID != nil && *req.DeptID != "" {
			db = db.Where("dept_id = ?", *req.DeptID)
		}
		if req.BeginTime != nil && *req.BeginTime != "" {
			if beginTime, err := time.Parse("2006-01-02 15:04:05", *req.BeginTime); err == nil {
				db = db.Where("created_at >= ?", beginTime)
			}
		}
		if req.EndTime != nil && *req.EndTime != "" {
			if endTime, err := time.Parse("2006-01-02 15:04:05", *req.EndTime); err == nil {
				db = db.Where("created_at <= ?", endTime)
			}
		}

		// 统计总数
		var total int64
		if err := db.Count(&total).Error; err != nil {
			response.Error(c, response.ErrServerError, "统计用户数量失败")
			return
		}

		// 分页查询
		offset := (req.Current - 1) * req.PageSize
		var users []models.User
		if err := db.Preload("Dept").
			Order("created_at DESC").
			Offset(offset).Limit(req.PageSize).
			Find(&users).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询用户列表失败")
			return
		}

		// 填充用户角色信息
		userIDs := make([]string, len(users))
		for i, u := range users {
			userIDs[i] = u.ID
		}

		// 查询用户角色关系
		var userRoles []models.UserRole
		if len(userIDs) > 0 {
			core.DB.GetDB().Where("user_id IN ?", userIDs).Find(&userRoles)
		}

		// 构建用户ID到角色ID的映射
		userRoleMap := make(map[string][]string)
		for _, ur := range userRoles {
			userRoleMap[ur.UserID] = append(userRoleMap[ur.UserID], ur.RoleID)
		}

		// 查询所有相关角色
		var allRoles []models.Role
		if len(userRoles) > 0 {
			roleIDs := make([]string, 0)
			for _, ur := range userRoles {
				roleIDs = append(roleIDs, ur.RoleID)
			}
			core.DB.GetDB().Where("id IN ?", roleIDs).Find(&allRoles)
		}

		// 构建角色ID到角色名称的映射
		roleMap := make(map[string]string)
		for _, r := range allRoles {
			roleMap[r.ID] = r.RoleName
		}

		// 填充每个用户的角色名称数组
		for i := range users {
			if roleIDs, ok := userRoleMap[users[i].ID]; ok {
				roleNames := make([]string, 0, len(roleIDs))
				for _, rid := range roleIDs {
					if roleName, exists := roleMap[rid]; exists {
						roleNames = append(roleNames, roleName)
					}
				}
				users[i].Roles = roleNames
			}
		}

		response.Page(c, users, total, req.Current, req.PageSize)
	}
}

// createUser 创建用户
// @Summary 创建用户
// @Description 创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body UserCreateRequest true "用户信息"
// @Success 200 {object} response.Response{data=models.User}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /users [post]
func createUser(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UserCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 检查用户名是否已存在
		var count int64
		if err := core.DB.GetDB().Model(&models.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查用户名失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrRecordExists, "用户名已存在")
			return
		}

		// 加密密码
		pwdManager := core.PwdManager
		hashedPassword, err := pwdManager.HashPassword(req.Password)
		if err != nil {
			response.Error(c, response.ErrServerError, "密码加密失败")
			return
		}

		// 创建用户
		user := models.User{
			Username:   req.Username,
			Password:   hashedPassword,
			Nickname:   req.Nickname,
			EmployeeNo: req.EmployeeNo,
			Email:      req.Email,
			Phone:      req.Phone,
			Gender:     models.Gender(req.Gender),
			Status:     models.UserStatus(req.Status),
			DeptID:     req.DeptID,
			Remark:     utils.DerefString(req.Remark),
		}

		// 使用事务创建用户及其关联
		if err := core.DB.GetDB().Transaction(func(tx *gorm.DB) error {
			// 创建用户
			if err := tx.Create(&user).Error; err != nil {
				return fmt.Errorf("创建用户失败: %w", err)
			}

			// 分配角色
			if len(req.RoleIds) > 0 {
				for _, roleID := range req.RoleIds {
					if err := tx.Table("sys_user_role").Create(&models.UserRole{
						UserID: user.ID,
						RoleID: roleID,
					}).Error; err != nil {
						return fmt.Errorf("分配角色失败: %w", err)
					}
				}
			}

			// 分配岗位
			if len(req.PostIds) > 0 {
				for _, postID := range req.PostIds {
					if err := tx.Table("sys_user_post").Create(&models.UserPost{
						UserID: user.ID,
						PostID: postID,
					}).Error; err != nil {
						return fmt.Errorf("分配岗位失败: %w", err)
					}
				}
			}

			return nil
		}); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 记录操作日志
		recordOperLog(c, core, "用户管理", OperTypeCreate)

		// 返回完整的用户信息
		core.DB.GetDB().Preload("Dept").First(&user, "id = ?", user.ID)
		response.Success(c, user)
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
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /users/{id} [get]
func getUser(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")
		if userID == "" {
			response.Error(c, response.ErrBadRequest, "用户ID不能为空")
			return
		}

		var user models.User
		if err := core.DB.GetDB().Preload("Dept").Preload("Roles").Preload("Posts").First(&user, "id = ?", userID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrNotFound, "用户不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询用户失败")
			}
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
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /users/{id} [put]
func updateUser(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")
		if userID == "" {
			response.Error(c, response.ErrBadRequest, "用户ID不能为空")
			return
		}

		var req UserUpdateRequest
		req.ID = userID
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 检查用户是否存在
		var user models.User
		if err := core.DB.GetDB().First(&user, "id = ?", userID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrNotFound, "用户不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询用户失败")
			}
			return
		}

		// 使用事务更新用户及其关联
		if err := core.DB.GetDB().Transaction(func(tx *gorm.DB) error {
			// 更新用户基本信息
			user.Nickname = req.Nickname
			user.EmployeeNo = req.EmployeeNo
			user.Email = req.Email
			user.Phone = req.Phone
			user.Gender = models.Gender(req.Gender)
			user.Status = models.UserStatus(req.Status)
			user.DeptID = req.DeptID
			user.Remark = utils.DerefString(req.Remark)

			if err := tx.Save(&user).Error; err != nil {
				return fmt.Errorf("更新用户失败: %w", err)
			}

			// 更新角色关联：先删除旧关联，再添加新关联
			if err := tx.Table("sys_user_role").Where("user_id = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
				return fmt.Errorf("删除用户角色关联失败: %w", err)
			}
			if len(req.RoleIds) > 0 {
				for _, roleID := range req.RoleIds {
					if err := tx.Table("sys_user_role").Create(&models.UserRole{
						UserID: userID,
						RoleID: roleID,
					}).Error; err != nil {
						return fmt.Errorf("分配角色失败: %w", err)
					}
				}
			}

			// 更新岗位关联：先删除旧关联，再添加新关联
			if err := tx.Table("sys_user_post").Where("user_id = ?", userID).Delete(&models.UserPost{}).Error; err != nil {
				return fmt.Errorf("删除用户岗位关联失败: %w", err)
			}
			if len(req.PostIds) > 0 {
				for _, postID := range req.PostIds {
					if err := tx.Table("sys_user_post").Create(&models.UserPost{
						UserID: userID,
						PostID: postID,
					}).Error; err != nil {
						return fmt.Errorf("分配岗位失败: %w", err)
					}
				}
			}

			return nil
		}); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 清除用户相关缓存
		if err := core.UserCacheService.InvalidateUserCache(c.Request.Context(), userID); err != nil {
			// 记录错误但不影响响应
		}
		if err := core.PostCacheService.InvalidateUserPostCache(c.Request.Context(), userID); err != nil {
			// 记录错误但不影响响应
		}

		// 记录操作日志
		recordOperLog(c, core, "用户管理", OperTypeUpdate)

		response.Success(c, gin.H{"message": "更新成功"})
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
// @Failure 400 {object} response.Response
// @Router /users/{id} [delete]
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

		// 使用事务删除用户及其关联数据
		if err := core.DB.GetDB().Transaction(func(tx *gorm.DB) error {
			// 删除用户角色关联
			if err := tx.Table("sys_user_role").Where("user_id = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
				return fmt.Errorf("删除用户角色关联失败: %w", err)
			}

			// 删除用户岗位关联
			if err := tx.Table("sys_user_post").Where("user_id = ?", userID).Delete(&models.UserPost{}).Error; err != nil {
				return fmt.Errorf("删除用户岗位关联失败: %w", err)
			}

			// 软删除用户
			if err := tx.Delete(&user).Error; err != nil {
				return fmt.Errorf("删除用户失败: %w", err)
			}

			return nil
		}); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 清除用户缓存
		if err := core.UserCacheService.InvalidateUserCache(c.Request.Context(), userID); err != nil {
			// 记录错误但不影响响应
		}

		// 清除用户岗位缓存
		if err := core.PostCacheService.InvalidateUserPostCache(c.Request.Context(), userID); err != nil {
			// 记录错误但不影响响应
		}

		// 记录操作日志
		recordOperLog(c, core, "用户管理", OperTypeDelete)

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

		// 使用事务批量删除用户及其关联数据
		if err := core.DB.GetDB().Transaction(func(tx *gorm.DB) error {
			// 批量删除用户角色关联
			if err := tx.Table("sys_user_role").Where("user_id IN ?", req.Ids).Delete(&models.UserRole{}).Error; err != nil {
				return fmt.Errorf("删除用户角色关联失败: %w", err)
			}

			// 批量删除用户岗位关联
			if err := tx.Table("sys_user_post").Where("user_id IN ?", req.Ids).Delete(&models.UserPost{}).Error; err != nil {
				return fmt.Errorf("删除用户岗位关联失败: %w", err)
			}

			// 批量软删除用户
			if err := tx.Where("id IN ?", req.Ids).Delete(&models.User{}).Error; err != nil {
				return fmt.Errorf("批量删除用户失败: %w", err)
			}

			return nil
		}); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 清除所有用户缓存
		if err := core.UserCacheService.InvalidateAllUserCache(c.Request.Context()); err != nil {
			// 记录错误但不影响响应
		}

		// 记录操作日志
		recordOperLog(c, core, "用户管理", OperTypeDelete)

		response.Success(c, nil)
	}
}
