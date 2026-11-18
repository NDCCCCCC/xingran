package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// DeptManagedOrgHandler 条线部门 → 管理归属机构 映射 handler（scope='permission'）
//
// 3 个端点：
//   - POST /ops/dept-managed-org/list          分页查询(含 sys_dept JOIN)
//   - POST /ops/dept-managed-org               新建
//   - POST /ops/dept-managed-org/:id/delete    删除
//
// 与 LocationAliasHandler 的关键差异：**不注入** WithDeptCacheInvalidator。
// 原因：permission 映射不参与工位部门下拉 union（那是 scope='workstation' 的
// 物理位置映射才影响的缓存面），因此无 dept 缓存失效诉求。
//
// 写方法模块名用 "部门管理"（归属映射挂在部门管理菜单下，与工位域无关）。
type DeptManagedOrgHandler struct {
	service opsServices.DeptManagedOrgService
	core    *core.Core
}

// NewDeptManagedOrgHandler 构造 DeptManagedOrgHandler 实例
func NewDeptManagedOrgHandler(service opsServices.DeptManagedOrgService) *DeptManagedOrgHandler {
	return &DeptManagedOrgHandler{service: service}
}

// WithCore 注入 core 依赖(操作日志记录所需)。链式注入而非改写构造器签名,
// 与 LocationAliasHandler 保持调用点兼容(router.go)。
func (h *DeptManagedOrgHandler) WithCore(core *core.Core) *DeptManagedOrgHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// managedOrgListQuery 归属映射列表查询请求体(分页参数)
// 前端 managedOrgApi.list 在 POST body 发送 { pageNum, pageSize }，
// 从 body 绑定(与本包 location_alias_handler.List 的 body 绑定约定一致)。
type managedOrgListQuery struct {
	PageNum  int `json:"pageNum"`
	PageSize int `json:"pageSize"`
}

// List 查询归属映射列表(读操作,不写 operlog)
func (h *DeptManagedOrgHandler) List(c *gin.Context) {
	// 从 POST body 读取分页参数;缺失或非法时回退默认值( pageNum=1, pageSize=10)
	var q managedOrgListQuery
	_ = c.ShouldBindJSON(&q)
	pageNum := 1
	pageSize := 10
	if q.PageNum > 0 {
		pageNum = q.PageNum
	}
	if q.PageSize > 0 {
		pageSize = q.PageSize
	}

	result, err := h.service.List(c.Request.Context(), pageNum, pageSize)
	if err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeServerError, "查询归属映射失败"))
		return
	}
	response.Success(c, result)
}

// Create 新建归属映射(写操作,触发 validateManagedOrg 校验)
func (h *DeptManagedOrgHandler) Create(c *gin.Context) {
	var req opsServices.DeptManagedOrgCreateRequest
	if !response.HandleJSONBinding(c, &req) {
		return
	}

	alias, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		// 越界（ErrScopeDenied）由 helper 显式透传 403；其余 validateManagedOrg 校验失败
		// 沿用存量 HTTP 400 + 中文错误信息（自映射/部门不存在/机构不存在/重复映射）。
		if opsServices.IsScopeDenied(err) {
			writeScopeAwareServiceError(c, err)
			return
		}
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 注：与 LocationAliasHandler 不同，此处不触发 dept 缓存失效 ——
	// permission 映射不参与工位部门下拉 union（见 struct 注释）。

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "部门管理", operlog.OperTypeCreate)
	response.Success(c, alias)
}

// Delete 删除归属映射(软删除,仅限 scope='permission' 行)
func (h *DeptManagedOrgHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if opsServices.IsScopeDenied(err) {
			writeScopeAwareServiceError(c, err)
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 注：同 Create，不触发 dept 缓存失效（permission 映射不影响工位下拉 union）。

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "部门管理", operlog.OperTypeDelete)
	response.Success(c, gin.H{"id": id})
}
