package v1

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// getDeptTree 获取部门树
// @Summary 获取部门树
// @Description 获取完整的部门树形结构，用于上级部门选择等场景
// @Tags 部门管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]DeptResponse}
// @Failure 500 {object} response.Response
// @Router /departments/tree [get]
func getDeptTree(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用 DeptService 的缓存方法获取部门树
		deptsPtr, err := core.DeptService.GetDeptTreeWithCache(c.Request.Context(), true)
		if err != nil {
			response.Error(c, response.ErrServerError, "获取部门树失败")
			return
		}

		// 查询负责人信息
		leaderMap, usernameMap := getLeaderMapFromTree(deptsPtr, core)

		// 转换为响应结构
		deptTree := convertDeptTreeToResponse(deptsPtr, leaderMap, usernameMap)

		response.Success(c, deptTree)
	}
}

// getDeptTreeSelect 获取部门树选择数据
// @Summary 获取部门树选择数据
// @Description 获取用于下拉选择的部门树数据，通常用于上级部门选择器
// @Tags 部门管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]DeptResponse}
// @Failure 500 {object} response.Response
// @Router /departments/tree/select [get]
func getDeptTreeSelect(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用 DeptService 的缓存方法获取部门树
		deptsPtr, err := core.DeptService.GetDeptTreeWithCache(c.Request.Context(), true)
		if err != nil {
			response.Error(c, response.ErrServerError, "获取部门树选择数据失败")
			return
		}

		// 查询负责人信息
		leaderMap, usernameMap := getLeaderMapFromTree(deptsPtr, core)

		// 转换为响应结构
		deptTree := convertDeptTreeToResponse(deptsPtr, leaderMap, usernameMap)

		response.Success(c, deptTree)
	}
}

// getRoleDeptTreeSelect 获取角色部门树选择数据（排除本部门及子部门）
// @Summary 获取角色部门树选择数据
// @Description 获取角色数据权限配置用的部门树选择数据，需要排除指定部门及其子部门
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param deptId query string false "要排除的部门ID"
// @Success 200 {object} response.Response{data=[]DeptResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /departments/role/tree/select [get]
func getRoleDeptTreeSelect(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		deptID := c.Query("deptId")

		// 使用 DeptService 的缓存方法获取部门树
		deptsPtr, err := core.DeptService.GetDeptTreeWithCache(c.Request.Context(), true)
		if err != nil {
			response.Error(c, response.ErrServerError, "获取部门树选择数据失败")
			return
		}

		// 如果指定了要排除的部门ID，过滤该部门及其子部门
		if deptID != "" {
			deptsPtr = filterDeptTree(deptsPtr, deptID)
		}

		// 查询负责人信息
		leaderMap, usernameMap := getLeaderMapFromTree(deptsPtr, core)

		// 转换为响应结构
		deptTree := convertDeptTreeToResponse(deptsPtr, leaderMap, usernameMap)

		response.Success(c, deptTree)
	}
}

// buildDeptTreeResponse 构建部门树（响应结构）
func buildDeptTreeResponse(depts []DeptResponse, parentID *string) []*DeptResponse {
	var tree []*DeptResponse
	for i := range depts {
		if (parentID == nil && depts[i].ParentID == nil) || (parentID != nil && depts[i].ParentID != nil && *depts[i].ParentID == *parentID) {
			children := buildDeptTreeResponse(depts, &depts[i].ID)
			if len(children) > 0 {
				depts[i].Children = children
			}
			tree = append(tree, &depts[i])
		}
	}
	return tree
}

// buildDeptTree 构建部门树（模型结构）
func buildDeptTree(depts []models.Department, parentID *string) []*models.Department {
	var tree []*models.Department
	for _, dept := range depts {
		if (parentID == nil && dept.ParentID == nil) || (parentID != nil && dept.ParentID != nil && *dept.ParentID == *parentID) {
			children := buildDeptTree(depts, &dept.ID)
			if len(children) > 0 {
				dept.Children = children
			}
			// 复制部门对象到指针
			deptCopy := dept
			tree = append(tree, &deptCopy)
		}
	}
	return tree
}

// getLeaderMapFromTree 从部门树中递归收集负责人ID并查询用户信息
func getLeaderMapFromTree(depts []*models.Department, core *core.Core) (map[string]*models.User, map[string]*models.User) {
	// 收集所有可能是用户ID或用户名的值
	possibleIDs := make([]string, 0)
	usernames := make([]string, 0)

	var collectLeaderIDs func([]*models.Department)
	collectLeaderIDs = func(deptList []*models.Department) {
		for _, dept := range deptList {
			if dept == nil {
				continue
			}
			if dept.Leader != nil && *dept.Leader != "" {
				leader := *dept.Leader
				// 判断是 UUID 还是用户名（UUID 包含连字符）
				if len(leader) == 36 && (strings.Count(leader, "-") == 4) {
					possibleIDs = append(possibleIDs, leader)
				} else {
					usernames = append(usernames, leader)
				}
			}
			// 递归处理子部门
			if len(dept.Children) > 0 {
				collectLeaderIDs(dept.Children)
			}
		}
	}

	collectLeaderIDs(depts)

	// 查询用户：分别按 ID 和用户名查询
	leaderMap := make(map[string]*models.User)
	usernameMap := make(map[string]*models.User)

	// 先按 ID 查询
	if len(possibleIDs) > 0 {
		var leaders []models.User
		if err := core.DB.GetDB().Where("id IN ?", possibleIDs).Find(&leaders).Error; err == nil {
			for i := range leaders {
				leaderMap[leaders[i].ID] = &leaders[i]
			}
		}
	}

	// 再按用户名查询（避免重复）
	if len(usernames) > 0 {
		var usersByName []models.User
		if err := core.DB.GetDB().Where("username IN ?", usernames).Find(&usersByName).Error; err == nil {
			for i := range usersByName {
				leaderMap[usersByName[i].ID] = &usersByName[i]
				usernameMap[usersByName[i].Username] = &usersByName[i]
			}
		}
	}

	return leaderMap, usernameMap
}

// convertDeptTreeToResponse 递归转换部门树为响应结构
func convertDeptTreeToResponse(depts []*models.Department, leaderMap map[string]*models.User, usernameMap map[string]*models.User) []*DeptResponse {
	result := make([]*DeptResponse, 0, len(depts))

	for _, dept := range depts {
		if dept == nil {
			continue
		}

		resp := &DeptResponse{
			ID:         dept.ID,
			DeptName:   dept.DeptName,
			ParentID:   dept.ParentID,
			Ancestors:  dept.Ancestors,
			OrderNum:   dept.OrderNum,
			Leader:     dept.Leader,
			Phone:      dept.Phone,
			Email:      dept.Email,
			Status:     dept.Status,
			CreatedAt:  dept.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:  dept.UpdatedAt.Format("2006-01-02 15:04:05"),
			Remark:     dept.Remark,
			Accessible: true, // 缓存数据默认可访问
		}

		// 填充负责人信息
		if dept.Leader != nil && *dept.Leader != "" {
			// 先尝试按用户ID查找
			if leader, exists := leaderMap[*dept.Leader]; exists {
				if leader.Nickname != nil && *leader.Nickname != "" {
					resp.LeaderName = leader.Nickname
				} else {
					resp.LeaderName = &leader.Username
				}
				resp.LeaderUsername = &leader.Username
			} else {
				// 尝试按用户名查找
				if leader, exists := usernameMap[*dept.Leader]; exists {
					if leader.Nickname != nil && *leader.Nickname != "" {
						resp.LeaderName = leader.Nickname
					} else {
						resp.LeaderName = &leader.Username
					}
					resp.LeaderUsername = &leader.Username
				}
			}
		}

		// 递归处理子部门
		if len(dept.Children) > 0 {
			resp.Children = convertDeptTreeToResponse(dept.Children, leaderMap, usernameMap)
		}

		result = append(result, resp)
	}

	return result
}

// filterDeptTree 过滤部门树，排除指定部门及其所有子部门
func filterDeptTree(depts []*models.Department, excludeDeptID string) []*models.Department {
	var filtered []*models.Department
	excludeSet := make(map[string]bool)

	// 收集所有需要排除的部门ID
	var collectExcluded func([]*models.Department)
	collectExcluded = func(deptList []*models.Department) {
		for _, dept := range deptList {
			if dept.ID == excludeDeptID {
				excludeSet[dept.ID] = true
				// 递归收集子部门
				if len(dept.Children) > 0 {
					collectExcluded(dept.Children)
				}
			} else if len(dept.Children) > 0 {
				// 检查子部门是否包含要排除的ID
				collectExcluded(dept.Children)
			}
		}
	}

	// 先遍历一次收集所有需要排除的ID
	for _, dept := range depts {
		if dept.ID == excludeDeptID {
			excludeSet[dept.ID] = true
			if len(dept.Children) > 0 {
				collectExcluded(dept.Children)
			}
		} else if len(dept.Children) > 0 {
			collectExcluded(dept.Children)
		}
	}

	// 过滤掉被排除的部门
	for _, dept := range depts {
		if !excludeSet[dept.ID] {
			// 递归处理子部门
			if len(dept.Children) > 0 {
				filteredChildren := filterDeptTree(dept.Children, excludeDeptID)
				if len(filteredChildren) > 0 {
					dept.Children = filteredChildren
					filtered = append(filtered, dept)
				}
			} else {
				filtered = append(filtered, dept)
			}
		}
	}

	return filtered
}
