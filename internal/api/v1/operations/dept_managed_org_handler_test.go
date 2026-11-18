package operations

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
)

// mockDeptManagedOrgService 实现 opsServices.DeptManagedOrgService（D-08 mock 模式）。
type mockDeptManagedOrgService struct {
	ListFunc   func(ctx context.Context, pageNum, pageSize int) (*opsServices.PageResult, error)
	CreateFunc func(ctx context.Context, req *opsServices.DeptManagedOrgCreateRequest) (*models.SysDeptLocationAlias, error)
	DeleteFunc func(ctx context.Context, id string) error
}

func (m *mockDeptManagedOrgService) List(ctx context.Context, pageNum, pageSize int) (*opsServices.PageResult, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, pageNum, pageSize)
	}
	return nil, errors.New("List not implemented")
}

func (m *mockDeptManagedOrgService) Create(ctx context.Context, req *opsServices.DeptManagedOrgCreateRequest) (*models.SysDeptLocationAlias, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return nil, errors.New("Create not implemented")
}

func (m *mockDeptManagedOrgService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errors.New("Delete not implemented")
}

func newManagedOrgRouter(h *DeptManagedOrgHandler) *gin.Engine {
	r := gin.New()
	g := r.Group("/ops/dept-managed-org")
	g.POST("/list", h.List)
	g.POST("", h.Create)
	g.POST("/:id/delete", h.Delete)
	return r
}

func TestDeptManagedOrgHandler_List_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDeptManagedOrgService{
		ListFunc: func(_ context.Context, pageNum, pageSize int) (*opsServices.PageResult, error) {
			assert.Equal(t, 2, pageNum, "body 分页参数应透传")
			assert.Equal(t, 20, pageSize)
			return &opsServices.PageResult{List: []opsServices.DeptManagedOrgListItem{}}, nil
		},
	}
	h := NewDeptManagedOrgHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newManagedOrgRouter(h), http.MethodPost, "/ops/dept-managed-org/list", `{"pageNum":2,"pageSize":20}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeptManagedOrgHandler_List_DefaultPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDeptManagedOrgService{
		ListFunc: func(_ context.Context, pageNum, pageSize int) (*opsServices.PageResult, error) {
			assert.Equal(t, 1, pageNum, "缺省 pageNum 回退 1")
			assert.Equal(t, 10, pageSize, "缺省 pageSize 回退 10")
			return &opsServices.PageResult{List: []opsServices.DeptManagedOrgListItem{}}, nil
		},
	}
	h := NewDeptManagedOrgHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newManagedOrgRouter(h), http.MethodPost, "/ops/dept-managed-org/list", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeptManagedOrgHandler_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDeptManagedOrgService{
		ListFunc: func(_ context.Context, _, _ int) (*opsServices.PageResult, error) {
			return nil, errors.New("db down")
		},
	}
	h := NewDeptManagedOrgHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newManagedOrgRouter(h), http.MethodPost, "/ops/dept-managed-org/list", `{}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "查询归属映射失败")
}

func TestDeptManagedOrgHandler_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDeptManagedOrgService{
		CreateFunc: func(_ context.Context, req *opsServices.DeptManagedOrgCreateRequest) (*models.SysDeptLocationAlias, error) {
			require.NotNil(t, req)
			assert.Equal(t, "D-tx", req.DeptID)
			assert.Equal(t, "D-qj", req.OrgID)
			return &models.SysDeptLocationAlias{DeptID: req.DeptID, LocationID: req.OrgID, Scope: "permission"}, nil
		},
	}
	h := NewDeptManagedOrgHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newManagedOrgRouter(h), http.MethodPost, "/ops/dept-managed-org",
		`{"deptId":"D-tx","orgId":"D-qj","remark":"条线归属"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeptManagedOrgHandler_Create_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDeptManagedOrgService{}
	h := NewDeptManagedOrgHandler(svc).WithCore(newTestCore(t))
	// 缺 required 字段 deptId/orgId → 绑定失败
	w := httpDo(newManagedOrgRouter(h), http.MethodPost, "/ops/dept-managed-org", `{"remark":"x"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeptManagedOrgHandler_Create_ScopeDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDeptManagedOrgService{
		CreateFunc: func(_ context.Context, _ *opsServices.DeptManagedOrgCreateRequest) (*models.SysDeptLocationAlias, error) {
			return nil, opsServices.ErrScopeDenied
		},
	}
	h := NewDeptManagedOrgHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newManagedOrgRouter(h), http.MethodPost, "/ops/dept-managed-org",
		`{"deptId":"D-tx","orgId":"D-qj"}`)
	assert.Equal(t, http.StatusForbidden, w.Code, "越界由 helper 显式透传 403")
}

func TestDeptManagedOrgHandler_Create_ValidateError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDeptManagedOrgService{
		CreateFunc: func(_ context.Context, _ *opsServices.DeptManagedOrgCreateRequest) (*models.SysDeptLocationAlias, error) {
			return nil, errors.New("管理归属机构必须是机构节点(分公司/中支/支公司)，不能是普通部门")
		},
	}
	h := NewDeptManagedOrgHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newManagedOrgRouter(h), http.MethodPost, "/ops/dept-managed-org",
		`{"deptId":"D-tx","orgId":"D-plain"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "必须是机构节点")
}

func TestDeptManagedOrgHandler_Delete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	svc := &mockDeptManagedOrgService{
		DeleteFunc: func(_ context.Context, id string) error {
			called = true
			assert.Equal(t, "alias-1", id)
			return nil
		},
	}
	h := NewDeptManagedOrgHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newManagedOrgRouter(h), http.MethodPost, "/ops/dept-managed-org/alias-1/delete", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

func TestDeptManagedOrgHandler_Delete_ScopeGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockDeptManagedOrgService{
		DeleteFunc: func(_ context.Context, _ string) error {
			return errors.New("该记录不是条线部门归属映射(scope=workstation)，禁止通过本端点删除")
		},
	}
	h := NewDeptManagedOrgHandler(svc).WithCore(newTestCore(t))
	w := httpDo(newManagedOrgRouter(h), http.MethodPost, "/ops/dept-managed-org/alias-1/delete", `{}`)
	// response.Error 对非 ScopeDenied 的 service 错误统一回业务错误码（与 location_alias Delete 一致）
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "禁止通过本端点删除")
}
