package base

import (
	"context"
	"fmt"

	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// Scope 与 gorm 官方 scope 函数签名一致（gorm.Scopes 一等公民形态）。
//
// service 层把业务过滤/排序/JOIN 构造为 Scope 传给 GORMRepository.List/GetByID，
// repo 只消费 scope，不做任何业务解释（filter 语义是业务知识，repo 只承接管道）。
type Scope = func(*gorm.DB) *gorm.DB

// PageParams 已归一化的分页参数。
//
// repo 直信入参（有意设计）：项目现存三种 clamp 语义（10..100 / 10..10000 / 无 clamp）
// 并存于各 service 层，repo 侧统一 clamp 会破坏零行为变更底线。
// service 层负责归一化后再传入。
type PageParams struct {
	Current  int
	PageSize int
}

// PageResult 分页结果
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Current  int         `json:"current"`
	PageSize int         `json:"pageSize"`
}

// GORMRepository GORM实现的通用仓储（scope 函数式，纯 struct 组合——不定义 interface，
// service 层直接组合本 struct，测试打 sqlite 真库而非 mock）。
type GORMRepository[T any] struct {
	db *gorm.DB
}

// NewGORMRepository 创建GORM仓储实例
func NewGORMRepository[T any](db *gorm.DB) *GORMRepository[T] {
	return &GORMRepository[T]{db: db}
}

// Create 创建实体
func (r *GORMRepository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// Update 更新实体
func (r *GORMRepository[T]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

// Delete 删除实体
func (r *GORMRepository[T]) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(new(T), "id = ?", id).Error
}

// GetByID 根据ID获取实体，支持变长 scopes。
//
// id 过滤规则：
//   - 无（非 nil）scope → WHERE id = ?（与直查形态逐字等价）
//   - 有 scope（典型为 Joins/Select 场景）→ 先对 new(T) 做 Tabler 断言，
//     命中则用 <TableName()>.id = ? 表限定（避免 join 场景 id 列 ambiguous）；
//     断言失败退回 WHERE id = ?
//
// 经 Model(new(T)) 起链，软删除过滤对 Count/Find 一致生效。
func (r *GORMRepository[T]) GetByID(ctx context.Context, id string, scopes ...Scope) (*T, error) {
	var entity T

	db := r.db.WithContext(ctx).Model(new(T))
	hasScope := false
	for _, s := range scopes {
		if s != nil {
			hasScope = true
			db = s(db)
		}
	}

	if hasScope {
		if tabler, ok := any(new(T)).(interface{ TableName() string }); ok {
			// tableName 是模型编译期常量（TableName()），非外部输入
			db = db.Where(fmt.Sprintf("%s.id = ?", tabler.TableName()), id)
		} else {
			db = db.Where("id = ?", id)
		}
	} else {
		db = db.Where("id = ?", id)
	}

	if err := db.First(&entity).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

// List scope 函数式分页查询。
//
// 内部管道：Model(new(T)) 起链 → 依序应用非 nil scopes → Count → Offset/Limit/Find。
// GORM 链语义（已由契约测试锁定）：Count 时非 count Select 被临时替换为 count(*)
// 并在返回后恢复；ORDER BY 在 Count 时被剥离并在 Find 恢复；Count→Find 共享链无污染。
//
// 设计红线：
//   - repo 内不 clamp 分页（PageParams 直信入参，clamp 语义归 service 层）
//   - repo 内不加默认排序（11 个消费者默认序各不相同，排序必须由 scope 提供）
//   - 不用 fmt.Sprintf 拼用户输入进条件；ORDER BY 只能来自白名单 map value
//     （见 SortScope/SortScopeWithTail，底座为 ApplySort）
//
// PageResult.List 为值切片 []T。
func (r *GORMRepository[T]) List(ctx context.Context, page PageParams, scopes ...Scope) (*PageResult, error) {
	query := r.db.WithContext(ctx).Model(new(T))
	for _, s := range scopes {
		if s != nil {
			query = s(query)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var list []T
	offset := (page.Current - 1) * page.PageSize
	if err := query.Offset(offset).Limit(page.PageSize).Find(&list).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  page.Current,
		PageSize: page.PageSize,
	}, nil
}

// BatchDelete 批量删除。
//
// 空 ids 返回 nil（与 11 个 operations service 现状语义一致：空选择是合法 no-op，
// 而非 400 错误——Phase 91-01 P1 语义反转，workstation_floor_code_77_03_test 有锁定）。
func (r *GORMRepository[T]) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Delete(new(T), "id IN ?", ids).Error
}

// SortScope B 型排序 scope（排他默认）：9 个 operations 服务的现状排序语义。
//
// 三态：
//   - req.OrderByColumn 白名单命中 → ORDER BY <col> <dir>（用户排序，无默认序）
//   - req.OrderByColumn 为空 → ORDER BY defaultOrder（默认序）
//   - req.OrderByColumn 非法（不在白名单）→ ApplySort 打 warn 日志 + 无显式排序
//
// 与 operations 侧现有 "base.ApplySort(query, ...) + if OrderByColumn == \"\" { Order(default) }"
// 组合语义逐字一致；ORDER BY 构造复用 ApplySort/ResolveSort 白名单机制，不重写。
func SortScope(req BaseListRequest, allowed map[string]string, defaultOrder string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		db = ApplySort(db, req, allowed)
		if req.OrderByColumn == "" {
			db = db.Order(defaultOrder)
		}
		return db
	}
}

// SortScopeWithTail A 型排序 scope（复合尾随）：door/wall 的现状排序语义。
//
//   - req.OrderByColumn 白名单命中 → ORDER BY <col> <dir>, <tail>（复合排序）
//   - req.OrderByColumn 为空/非法 → 仅 ORDER BY <tail>
//
// 与 door_service.go 现状（ApplySort 后 fetchRecords 恒追加 created_at DESC）一致。
func SortScopeWithTail(req BaseListRequest, allowed map[string]string, tail string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		db = ApplySort(db, req, allowed)
		return db.Order(tail)
	}
}

// WrapError 包装错误信息
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// IsNotFound 判断是否为记录不存在错误
func IsNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound || apperrors.IsAppError(err) && apperrors.GetAppError(err).GetCode() == apperrors.CodeRecordNotFound
}

// IsDuplicate 判断是否为重复记录错误
func IsDuplicate(err error) bool {
	return apperrors.IsAppError(err) && apperrors.GetAppError(err).GetCode() == apperrors.CodeRecordExists
}
