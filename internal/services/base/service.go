package base

import (
	"context"
	"fmt"

	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// Repository 基础仓储接口，定义通用数据访问操作
type Repository[T any] interface {
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*T, error)
	List(ctx context.Context, query *Query) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
}

// Query 查询参数
type Query struct {
	Where    []WhereCondition `json:"where"`
	OrderBy  string           `json:"orderBy"`
	Offset   int              `json:"offset"`
	Limit    int              `json:"limit"`
	Preloads []string         `json:"preloads"`
}

// WhereCondition 查询条件
type WhereCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // =, !=, >, <, >=, <=, LIKE, IN
	Value    interface{} `json:"value"`
}

// PageResult 分页结果
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Current  int         `json:"current"`
	PageSize int         `json:"pageSize"`
}

// GORMRepository GORM实现的通用仓储
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

// GetByID 根据ID获取实体
func (r *GORMRepository[T]) GetByID(ctx context.Context, id string) (*T, error) {
	var entity T
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// List 查询列表
func (r *GORMRepository[T]) List(ctx context.Context, query *Query) (*PageResult, error) {
	var total int64
	var list []T

	db := r.db.WithContext(ctx).Model(new(T))

	// 应用查询条件
	for _, cond := range query.Where {
		switch cond.Operator {
		case "=":
			db = db.Where(cond.Field+" = ?", cond.Value)
		case "!=":
			db = db.Where(cond.Field+" != ?", cond.Value)
		case ">":
			db = db.Where(cond.Field+" > ?", cond.Value)
		case "<":
			db = db.Where(cond.Field+" < ?", cond.Value)
		case ">=":
			db = db.Where(cond.Field+" >= ?", cond.Value)
		case "<=":
			db = db.Where(cond.Field+" <= ?", cond.Value)
		case "LIKE":
			db = db.Where(cond.Field+" LIKE ?", cond.Value)
		case "IN":
			db = db.Where(cond.Field+" IN ?", cond.Value)
		}
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 预加载关联
	for _, preload := range query.Preloads {
		db = db.Preload(preload)
	}

	// 排序
	if query.OrderBy != "" {
		db = db.Order(query.OrderBy)
	}

	// 分页
	if err := db.Offset(query.Offset).Limit(query.Limit).Find(&list).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  query.Offset/query.Limit + 1,
		PageSize: query.Limit,
	}, nil
}

// BatchDelete 批量删除
func (r *GORMRepository[T]) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return apperrors.BadRequest("ids不能为空")
	}
	return r.db.WithContext(ctx).Delete(new(T), "id IN ?", ids).Error
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
