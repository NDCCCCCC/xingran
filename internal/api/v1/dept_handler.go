package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

// DeptListRequest 部门列表请求
type DeptListRequest struct {
	DeptName string `json:"deptName,omitempty"`
	Status   string `json:"status,omitempty"`
	Current  int    `json:"current,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
}

// DeptCreateRequest 部门创建请求
type DeptCreateRequest struct {
	DeptName string          `json:"deptName" binding:"required,max=50"`
	ParentID *string         `json:"parentId,omitempty"`
	OrderNum int             `json:"orderNum"`
	Leader   *string         `json:"leader,omitempty"`
	Phone    *string         `json:"phone,omitempty"`
	Email    *string         `json:"email,omitempty"`
	Status   models.DeptStatus `json:"status"`
	Remark   *string         `json:"remark,omitempty"`
}

// DeptUpdateRequest 部门更新请求
type DeptUpdateRequest struct {
	ID       string          `json:"id" binding:"required"`
	DeptName string          `json:"deptName" binding:"required,max=50"`
	ParentID *string         `json:"parentId,omitempty"`
	OrderNum int             `json:"orderNum"`
	Leader   *string         `json:"leader,omitempty"`
	Phone    *string         `json:"phone,omitempty"`
	Email    *string         `json:"email,omitempty"`
	Status   models.DeptStatus `json:"status"`
	Remark   *string         `json:"remark,omitempty"`
}

// listDepts 查询部门列表
// @Summary 查询部门列表
// @Description 查询部门列表，返回树形结构的部门数据，支持按部门名称、状态筛选
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param request body DeptListRequest false "查询参数"
// @Success 200 {object} response.Response{data=[]models.Department}
// @Failure 500 {object} response.Response
// @Router /departments/list [post]
func listDepts(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DeptListRequest
		// 允许空的请求体，所有字段都是可选的
		c.ShouldBindJSON(&req)

		// 构建查询条件
		db := core.DB.GetDB().Model(&models.Department{})

		// 部门名称模糊查询
		if req.DeptName != "" {
			db = db.Where("dept_name LIKE ?", "%"+req.DeptName+"%")
		}

		// 状态查询
		if req.Status != "" {
			if status, err := strconv.Atoi(req.Status); err == nil {
				db = db.Where("status = ?", status)
			}
		}

		// 查询数据并按order_num排序
		// 注意：部门是树形结构，不需要分页，所以忽略current和pageSize参数
		var depts []models.Department
		if err := db.Order("order_num ASC").Find(&depts).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询部门列表失败")
			return
		}

		// 返回树形结构，部门管理页面需要树形展示
		deptTree := buildDeptTree(depts, nil)

		response.Success(c, deptTree)
	}
}

// buildDeptTree 构建部门树
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

// createDept 创建部门
// @Summary 创建部门
// @Description 创建新的部门，支持设置部门名称、父部门、负责人、联系电话、邮箱等
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param request body DeptCreateRequest true "部门创建信息"
// @Success 200 {object} response.Response{data=models.Department}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /departments [post]
func createDept(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DeptCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 检查部门名称是否已存在
		var count int64
		db := core.DB.GetDB().Model(&models.Department{})
		if req.ParentID != nil {
			db = db.Where("parent_id = ? AND dept_name = ?", req.ParentID, req.DeptName)
		} else {
			db = db.Where("parent_id IS NULL AND dept_name = ?", req.DeptName)
		}
		if err := db.Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查部门名称失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrRecordExists, "部门名称已存在")
			return
		}

		// 构建ancestors字符串
		ancestors := ""
		if req.ParentID != nil {
			var parentDept models.Department
			if err := core.DB.GetDB().First(&parentDept, "id = ?", *req.ParentID).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					response.Error(c, response.ErrRecordNotFound, "父部门不存在")
				} else {
					response.Error(c, response.ErrServerError, "查询父部门失败")
				}
				return
			}
			if parentDept.Ancestors != "" {
				ancestors = parentDept.Ancestors + "," + *req.ParentID
			} else {
				ancestors = *req.ParentID
			}
		}

		// 创建部门
		dept := models.Department{
			DeptName:  req.DeptName,
			ParentID:  req.ParentID,
			Ancestors: ancestors,
			OrderNum:  req.OrderNum,
			Leader:    req.Leader,
			Phone:     req.Phone,
			Email:     req.Email,
			Status:    req.Status,
			Remark:    func() string { if req.Remark != nil { return *req.Remark }; return "" }(),
		}

		if err := core.DB.GetDB().Create(&dept).Error; err != nil {
			response.Error(c, response.ErrServerError, "创建部门失败")
			return
		}

		response.Success(c, dept)
	}
}

// getDept 获取部门详情
// @Summary 获取部门详情
// @Description 根据部门ID获取部门详细信息
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param id path string true "部门ID"
// @Success 200 {object} response.Response{data=models.Department}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /departments/{id} [get]
func getDept(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "部门ID不能为空")
			return
		}

		var dept models.Department
		if err := core.DB.GetDB().First(&dept, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "部门不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询部门失败")
			}
			return
		}

		response.Success(c, dept)
	}
}

// updateDept 更新部门
// @Summary 更新部门
// @Description 更新部门信息，包括部门名称、父部门、负责人、联系电话、邮箱等，同时更新所有子部门的层级关系
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param id path string true "部门ID"
// @Param request body DeptUpdateRequest true "部门更新信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /departments/{id} [put]
func updateDept(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "部门ID不能为空")
			return
		}

		var req DeptUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 检查部门是否存在
		var dept models.Department
		if err := core.DB.GetDB().First(&dept, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "部门不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询部门失败")
			}
			return
		}

		// 检查是否将自己设置为父部门
		if req.ParentID != nil && *req.ParentID == id {
			response.Error(c, response.ErrBadRequest, "不能将自己设置为父部门")
			return
		}

		// 检查是否将自己的子部门设置为父部门（防止循环引用）
		if req.ParentID != nil {
			childIds := getChildDeptIds(core.DB.GetDB(), id)
			for _, childId := range childIds {
				if childId == *req.ParentID {
					response.Error(c, response.ErrBadRequest, "不能将子部门设置为父部门")
					return
				}
			}
		}

		// 检查部门名称是否与其他部门重复
		var count int64
		db := core.DB.GetDB().Model(&models.Department{}).Where("id != ?", id)
		if req.ParentID != nil {
			db = db.Where("parent_id = ? AND dept_name = ?", req.ParentID, req.DeptName)
		} else {
			db = db.Where("parent_id IS NULL AND dept_name = ?", req.DeptName)
		}
		if err := db.Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查部门名称失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrRecordExists, "部门名称已存在")
			return
		}

		// 重新构建ancestors字符串
		ancestors := ""
		if req.ParentID != nil {
			var parentDept models.Department
			if err := core.DB.GetDB().First(&parentDept, "id = ?", *req.ParentID).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					response.Error(c, response.ErrRecordNotFound, "父部门不存在")
				} else {
					response.Error(c, response.ErrServerError, "查询父部门失败")
				}
				return
			}
			if parentDept.Ancestors != "" {
				ancestors = parentDept.Ancestors + "," + *req.ParentID
			} else {
				ancestors = *req.ParentID
			}
		}

		// 开始事务
		tx := core.DB.GetDB().Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// 更新部门信息
		updates := map[string]interface{}{
			"dept_name": req.DeptName,
			"parent_id": req.ParentID,
			"ancestors": ancestors,
			"order_num": req.OrderNum,
			"leader":    req.Leader,
			"phone":     req.Phone,
			"email":     req.Email,
			"status":    req.Status,
			"remark":    func() string { if req.Remark != nil { return *req.Remark }; return "" }(),
		}

		if err := tx.Model(&dept).Updates(updates).Error; err != nil {
			tx.Rollback()
			response.Error(c, response.ErrServerError, "更新部门失败")
			return
		}

		// 更新所有子部门的ancestors
		if len(ancestors) > 0 {
			childIds := getChildDeptIds(tx, id)
			for _, childId := range childIds {
				if err := tx.Model(&models.Department{}).Where("id = ?", childId).
					Update("ancestors", gorm.Expr("REPLACE(ancestors, ?, ?)", dept.Ancestors, ancestors)).Error; err != nil {
					tx.Rollback()
					response.Error(c, response.ErrServerError, "更新子部门失败")
					return
				}
			}
		}

		// 提交事务
		if err := tx.Commit().Error; err != nil {
			response.Error(c, response.ErrServerError, "更新部门失败")
			return
		}

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteDept 删除部门
// @Summary 删除部门
// @Description 删除指定部门（需确保没有子部门、用户且没有被角色关联）
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param id path string true "部门ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /departments/{id} [delete]
func deleteDept(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "部门ID不能为空")
			return
		}

		// 检查部门是否存在
		var dept models.Department
		if err := core.DB.GetDB().First(&dept, "id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, response.ErrRecordNotFound, "部门不存在")
			} else {
				response.Error(c, response.ErrServerError, "查询部门失败")
			}
			return
		}

		// 检查是否有子部门
		var count int64
		if err := core.DB.GetDB().Model(&models.Department{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查子部门失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrBadRequest, "存在子部门，无法删除")
			return
		}

		// 检查是否存在用户
		if err := core.DB.GetDB().Model(&models.User{}).Where("dept_id = ?", id).Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查部门用户失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrBadRequest, "部门存在用户，无法删除")
			return
		}

		// 检查是否被角色关联
		if err := core.DB.GetDB().Table("role_depts").Where("dept_id = ?", id).Count(&count).Error; err != nil {
			response.Error(c, response.ErrServerError, "检查部门关联情况失败")
			return
		}
		if count > 0 {
			response.Error(c, response.ErrBadRequest, "部门已分配给角色，无法删除")
			return
		}

		// 删除部门
		if err := core.DB.GetDB().Delete(&dept).Error; err != nil {
			response.Error(c, response.ErrServerError, "删除部门失败")
			return
		}

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// getChildDeptIds 获取所有子部门ID
func getChildDeptIds(db *gorm.DB, parentId string) []string {
	var childIds []string
	var depts []models.Department

	if err := db.Where("parent_id = ?", parentId).Find(&depts).Error; err != nil {
		return childIds
	}

	for _, dept := range depts {
		childIds = append(childIds, dept.ID)
		// 递归获取子部门的子部门
		grandChildIds := getChildDeptIds(db, dept.ID)
		childIds = append(childIds, grandChildIds...)
	}

	return childIds
}

// getDeptTreeSelect 获取部门树选择器
// @Summary 获取部门树选择器
// @Description 获取所有正常状态的部门树，用于部门选择器组件
// @Tags 部门管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]models.Department}
// @Failure 500 {object} response.Response
// @Router /departments/tree-select [get]
func getDeptTreeSelect(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var depts []models.Department
		if err := core.DB.GetDB().Where("status = ?", models.DeptStatusNormal).
			Order("order_num ASC").Find(&depts).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询部门列表失败")
			return
		}

		// 构建树形结构
		deptTree := buildDeptTree(depts, nil)

		response.Success(c, deptTree)
	}
}

// getRoleDeptTreeSelect 获取角色部门树选择器
// @Summary 获取角色部门树选择器
// @Description 获取指定角色的部门树和已选中的部门ID，用于角色部门权限分配
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param roleId path string true "角色ID"
// @Success 200 {object} response.Response{data=object}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /departments/role-dept-tree-select/{roleId} [get]
func getRoleDeptTreeSelect(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleId := c.Param("roleId")
		if roleId == "" {
			response.Error(c, response.ErrBadRequest, "角色ID不能为空")
			return
		}

		// 查询所有部门
		var depts []models.Department
		if err := core.DB.GetDB().Where("status = ?", models.DeptStatusNormal).
			Order("order_num ASC").Find(&depts).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询部门列表失败")
			return
		}

		// 查询角色已关联的部门ID
		var deptIds []string
		if err := core.DB.GetDB().Table("role_depts").
			Where("role_id = ?", roleId).Pluck("dept_id", &deptIds).Error; err != nil {
			response.Error(c, response.ErrServerError, "查询角色部门关联失败")
			return
		}

		// 构建树形结构
		deptTree := buildDeptTree(depts, nil)

		response.Success(c, gin.H{
			"depts":    deptTree,
			"checkedKeys": deptIds,
		})
	}
}