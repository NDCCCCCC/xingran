package system

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

type departmentCacheService struct {
	*departmentService
	cache CacheProvider
	CacheServiceBase
}

func NewDepartmentServiceWithCache(
	db *gorm.DB,
	cache CacheProvider,
	config *services.CacheConfigService,
) DepartmentService {
	base := &departmentService{db: db}
	return &departmentCacheService{
		departmentService: base,
		cache:             cache,
		CacheServiceBase:  CacheServiceBase{Config: config},
	}
}

func (s *departmentCacheService) GetTree(ctx context.Context, includeDisabled bool) ([]*models.Department, error) {
	return s.GetTreeWithFilter(ctx, includeDisabled, requests.DepartmentListParams{})
}

func (s *departmentCacheService) GetTreeWithFilter(ctx context.Context, includeDisabled bool, params requests.DepartmentListParams) ([]*models.Department, error) {
	// 诊断日志（v1.54）：确认缓存装饰器收到的 scoped 值与 ctx 注入状态。
	r, ctxOK := datascope.FromContext(ctx)
	applogger.WithFields(map[string]interface{}{
		"scoped": params.Scoped, "ctx_injected": ctxOK,
		"dept_ids_len": len(r.DeptIDs), "scope_all": r.All,
	}).Info("[部门树缓存层] GetTreeWithFilter 诊断")

	// v1.54 关键修复：scoped 请求**必须绕过缓存**。
	//
	// 原实现里，普通树请求（仅 scoped=true，无 DeptName/Status）会命中
	// cacheKey=dept:tree[:all]（**不含 scoped、不含 userID**）——第一个请求把
	// 全量树写进缓存，后续任何用户（含机构管理员）都读到这份全量树；且
	// queryTree 自己查全量、从不调用 filterDeptTreeByScope。这正是生产
	// "scoped:true 却返回全树"的真因（handler 日志有 scoped:true，但没有
	// 过滤器日志 = 装饰器根本没走到收口代码）。
	//
	// 收口结果依赖"当前登录人的数据范围"，按人实时变化，本就不该共享缓存；
	// 直接委托底层 departmentService.GetTreeWithFilter（内含 filterDeptTreeByScope）。
	// 非 scoped 的全量树继续走缓存，零回归。
	if params.Scoped {
		return s.departmentService.GetTreeWithFilter(ctx, includeDisabled, params)
	}

	if params.DeptName != "" || params.Status != nil {
		return s.departmentService.GetTreeWithFilter(ctx, includeDisabled, params)
	}

	cacheKey := BuildDeptCacheKey(CacheKeyDeptTree)
	if includeDisabled {
		cacheKey += ":all"
	}

	return base.GetOrSetJSON(ctx, s.cache,
		cacheKey,
		s.GetExpiration(services.CacheConfigDeptTree, 30*time.Minute),
		func() ([]*models.Department, error) { return s.queryTree(ctx, includeDisabled) })
}

func (s *departmentCacheService) GetTreeWithCache(ctx context.Context, includeDisabled bool) ([]*models.Department, error) {
	return s.GetTree(ctx, includeDisabled)
}

func (s *departmentCacheService) queryTree(ctx context.Context, includeDisabled bool) ([]*models.Department, error) {
	var depts []models.Department
	query := s.db.WithContext(ctx).Model(&models.Department{})

	if !includeDisabled {
		query = query.Where("status = ?", models.DeptStatusNormal)
	}

	if err := query.Order("order_num ASC").Find(&depts).Error; err != nil {
		return nil, err
	}

	s.departmentService.fillLeaderInfo(ctx, depts)

	return s.departmentService.buildDeptTree(depts, nil), nil
}

func (s *departmentCacheService) GetSelectDataWithCache(ctx context.Context) ([]*models.Department, error) {
	return base.GetOrSetJSON(ctx, s.cache,
		BuildDeptCacheKey("tree:select"),
		s.GetExpiration(services.CacheConfigDeptSelect, 30*time.Minute),
		func() ([]*models.Department, error) { return s.querySelectData(ctx) })
}

func (s *departmentCacheService) querySelectData(ctx context.Context) ([]*models.Department, error) {
	var depts []models.Department
	if err := s.db.WithContext(ctx).
		Where("status = ?", models.DeptStatusNormal).
		Order("order_num ASC").
		Find(&depts).Error; err != nil {
		return nil, err
	}

	s.departmentService.fillLeaderInfo(ctx, depts)

	return s.departmentService.buildDeptTree(depts, nil), nil
}

func (s *departmentCacheService) InvalidateDeptCache(ctx context.Context) error {
	base.InvalidatePattern(ctx, s.cache, []string{
		BuildDeptCacheKey(CacheKeyDeptTree) + "*",
		BuildDeptCacheKey(CacheKeyDeptList) + "*",
		BuildDeptCacheKey("tree:select") + "*",
	}, "DEPT")
	return nil
}

func (s *departmentCacheService) Create(ctx context.Context, req *requests.DepartmentCreateRequest) error {
	if err := s.departmentService.Create(ctx, req); err != nil {
		return err
	}
	return s.InvalidateDeptCache(ctx)
}

func (s *departmentCacheService) Update(ctx context.Context, req *requests.DepartmentUpdateRequest) error {
	if err := s.departmentService.Update(ctx, req); err != nil {
		return err
	}
	return s.InvalidateDeptCache(ctx)
}

func (s *departmentCacheService) Delete(ctx context.Context, id string) error {
	if err := s.departmentService.Delete(ctx, id); err != nil {
		return err
	}
	return s.InvalidateDeptCache(ctx)
}

func (s *departmentCacheService) BatchDelete(ctx context.Context, ids []string) error {
	if err := s.departmentService.BatchDelete(ctx, ids); err != nil {
		return err
	}
	return s.InvalidateDeptCache(ctx)
}

func (s *departmentCacheService) UpdateStatus(ctx context.Context, id string, status models.DeptStatus) error {
	if err := s.departmentService.UpdateStatus(ctx, id, status); err != nil {
		return err
	}
	return s.InvalidateDeptCache(ctx)
}

func (s *departmentCacheService) GetDB() *gorm.DB {
	return s.departmentService.GetDB()
}
