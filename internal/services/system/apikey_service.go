package system

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// apiKeyAllowedSortFields API密钥列表可排序字段白名单。
// key=前端逻辑字段名(camelCase)，value=DB 列名(snake_case)。
var apiKeyAllowedSortFields = map[string]string{
	"name":        "name",
	"createdAt":   "created_at",
	"updatedAt":   "updated_at",
	"isActive":    "is_active",
	"expiresAt":   "expires_at",
	"lastUsedAt":  "last_used_at",
}

// APIKeyService API密钥服务接口
type APIKeyService interface {
	CreateAPIKey(ctx context.Context, userID string, req *requests.CreateAPIKeyRequest) (*string, error)
	ListAPIKeys(ctx context.Context, userID string, params requests.ListAPIKeysParams) (*PageResult, error)
	GetAPIKey(ctx context.Context, id string) (*models.APIKey, error)
	UpdateAPIKey(ctx context.Context, id string, req *requests.UpdateAPIKeyRequest) error
	DeleteAPIKey(ctx context.Context, id string) error
	ToggleAPIKeyStatus(ctx context.Context, id string) error
	ValidateAPIKey(ctx context.Context, keyStr string) (*models.APIKey, error)
	ListUsageLogs(ctx context.Context, params ListUsageLogsParams) (*UsageLogsPageResult, error)
	GetUsageLogSummary(ctx context.Context, apiKeyID string) (*UsageSummary, error)
}

// apiKeyServiceImpl API密钥服务实现
type apiKeyServiceImpl struct {
	db *gorm.DB
}

// NewAPIKeyService 创建API密钥服务实例
func NewAPIKeyService(db *gorm.DB) APIKeyService {
	return &apiKeyServiceImpl{
		db: db,
	}
}

// 常量定义
const (
	KeyPrefix      = "rec_"            // 密钥前缀
	KeyLength      = 64                // 密钥长度（十六进制字符数）
	MaxKeysPerUser = 100               // 单用户最大密钥数
	apiKeyByteLen  = 32                // API Key 字节长度（256-bit，对应 KeyLength=64 hex chars）
)

// API Key 作用域名称
const (
	APIKeyScopeRead  = "read"
	APIKeyScopeWrite = "write"
	APIKeyScopeAdmin = "admin"
)

// PageResult is declared in user_service.go to avoid redeclaration

// isKeyExpired 检查密钥是否过期
func isKeyExpired(expiresAt *time.Time) bool {
	if expiresAt == nil {
		return false
	}
	return expiresAt.Before(time.Now())
}

// isValidKeyFormat 验证密钥格式
func isValidKeyFormat(keyStr string) bool {
	// 检查长度：4（前缀）+ 64（hex）= 68
	if len(keyStr) != 68 {
		return false
	}

	// 检查前缀
	if len(keyStr) < 4 || keyStr[:4] != KeyPrefix {
		return false
	}

	// 检查后64位是否为有效十六进制
	hexPart := keyStr[4:]
	for _, c := range hexPart {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}

	return true
}

// validateScopes 验证作用域
func validateScopes(scopes []string) error {
	validScopes := map[string]bool{
		APIKeyScopeRead:  true,
		APIKeyScopeWrite: true,
		APIKeyScopeAdmin: true,
	}

	for _, scope := range scopes {
		if !validScopes[scope] {
			return apperrors.Wrap(nil, apperrors.CodeParamError, "无效的作用域: "+scope)
		}
	}

	return nil
}

// generateKey 生成随机密钥
func generateKey() (string, error) {
	bytes := make([]byte, apiKeyByteLen) // 32 bytes = 64 hex chars
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	hexStr := hex.EncodeToString(bytes)
	return fmt.Sprintf("%s%s", KeyPrefix, hexStr), nil
}

// ValidateAPIKey 验证API密钥
func (s *apiKeyServiceImpl) ValidateAPIKey(ctx context.Context, keyStr string) (*models.APIKey, error) {
	// 验证密钥格式
	if !isValidKeyFormat(keyStr) {
		return nil, apperrors.Wrap(nil, apperrors.CodeParamError, "无效的密钥格式")
	}

	// 查询数据库
	var apiKey models.APIKey
	err := s.db.WithContext(ctx).
		Preload("User").
		Where("key = ? AND is_active = ?", keyStr, true).
		First(&apiKey).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.Wrap(nil, apperrors.CodeUnauthorized, "密钥不存在或已禁用")
		}
		return nil, apperrors.DatabaseError(err)
	}

	// 验证过期时间
	if isKeyExpired(apiKey.ExpiresAt) {
		return nil, apperrors.Wrap(nil, apperrors.CodeUnauthorized, "密钥已过期")
	}

	// 异步更新最后使用时间
	go func() {
		now := time.Now()
		s.db.Model(&apiKey).Update("last_used_at", now)
	}()

	return &apiKey, nil
}

// CreateAPIKey 创建API密钥
func (s *apiKeyServiceImpl) CreateAPIKey(ctx context.Context, userID string, req *requests.CreateAPIKeyRequest) (*string, error) {
	// 验证用户是否存在
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.Wrap(nil, apperrors.CodeParamError, "用户不存在")
		}
		return nil, apperrors.DatabaseError(err)
	}

	// 检查用户密钥数量限制
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.APIKey{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}
	if count >= MaxKeysPerUser {
		return nil, apperrors.Wrap(nil, apperrors.CodeParamError, fmt.Sprintf("已达到最大密钥数量限制（%d个）", MaxKeysPerUser))
	}

	// 生成密钥
	key, err := generateKey()
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.CodeServerError, "密钥生成失败")
	}

	// 验证作用域
	if err := validateScopes(req.Scopes); err != nil {
		return nil, err
	}

	// 解析过期时间
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		parsedTime, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.CodeParamError, "过期时间格式错误")
		}
		expiresAt = &parsedTime
	}

	// 创建APIKey记录
	apiKey := models.APIKey{
		Name:         req.Name,
		Key:          key,
		UserID:       &userID,
		Scopes:       req.Scopes,
		IPWhitelist:  req.IPWhitelist,
		Description:  req.Description,
		InheritPerms: req.InheritPerms,
		ExpiresAt:    expiresAt,
		IsActive:     true,
	}

	if err := s.db.WithContext(ctx).Create(&apiKey).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	return &key, nil
}

// ListAPIKeys 查询API密钥列表
func (s *apiKeyServiceImpl) ListAPIKeys(ctx context.Context, userID string, params requests.ListAPIKeysParams) (*PageResult, error) {
	var total int64
	var list []models.APIKey

	query := s.db.WithContext(ctx).Model(&models.APIKey{})

	// 筛选用户ID
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	// 添加筛选条件
	// 注:生产用 PostgreSQL(LEFT() / @> JSONB),测试用 SQLite 需要 dialect 兼容分支
	isSQLite := s.db.Dialector.Name() == "sqlite"
	if params.Keyword != nil && *params.Keyword != "" {
		keyword := "%" + *params.Keyword + "%"
		if isSQLite {
			// SQLite 用 substr() 替代 LEFT(),取 key 前 12 字符匹配
			query = query.Where("name LIKE ? OR substr(key, 1, 12) LIKE ?", keyword, keyword)
		} else {
			query = query.Where("name LIKE ? OR LEFT(key, 12) LIKE ?", keyword, keyword)
		}
	}
	if params.Status != nil {
		query = query.Where("is_active = ?", *params.Status)
	}
	if params.Scope != nil && *params.Scope != "" {
		if isSQLite {
			// SQLite 无 JSONB @>,用 json_each() 遍历 scopes 数组检查 value 匹配
			query = query.Where("EXISTS (SELECT 1 FROM json_each(sys_api_keys.scopes) WHERE json_each.value = ?)", *params.Scope)
		} else {
			query = query.Where("scopes @> ?", fmt.Sprintf("[\"%s\"]", *params.Scope))
		}
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 分页查询：服务端排序（白名单）优先，无 OrderByColumn 时保留 created_at DESC 默认
	offset := params.GetOffset()
	sortReq := base.BaseListRequest{
		Current:       params.Current,
		PageSize:      params.PageSize,
		OrderByColumn: params.OrderByColumn,
		IsAsc:         params.IsAsc,
	}
	query = base.ApplySort(query, sortReq, apiKeyAllowedSortFields)
	if params.OrderByColumn == "" {
		query = query.Order("created_at DESC")
	}
	if err := query.Preload("User").
		Offset(offset).Limit(params.PageSize).
		Find(&list).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// GetAPIKey 获取API密钥详情
func (s *apiKeyServiceImpl) GetAPIKey(ctx context.Context, id string) (*models.APIKey, error) {
	var apiKey models.APIKey
	err := s.db.WithContext(ctx).Preload("User").First(&apiKey, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.Wrap(nil, apperrors.CodeParamError, "密钥不存在")
		}
		return nil, apperrors.DatabaseError(err)
	}
	return &apiKey, nil
}

// UpdateAPIKey 更新API密钥
func (s *apiKeyServiceImpl) UpdateAPIKey(ctx context.Context, id string, req *requests.UpdateAPIKeyRequest) error {
	// 检查密钥是否存在
	var apiKey models.APIKey
	if err := s.db.WithContext(ctx).First(&apiKey, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.Wrap(nil, apperrors.CodeParamError, "密钥不存在")
		}
		return apperrors.DatabaseError(err)
	}

	// 更新可修改字段
	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.InheritPerms != nil {
		updates["inherit_perms"] = *req.InheritPerms
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.ExpiresAt != nil {
		if *req.ExpiresAt == "" {
			updates["expires_at"] = nil
		} else {
			parsedTime, err := time.Parse(time.RFC3339, *req.ExpiresAt)
			if err != nil {
				return apperrors.Wrap(err, apperrors.CodeParamError, "过期时间格式错误")
			}
			updates["expires_at"] = parsedTime
		}
	}
	if req.Scopes != nil {
		if err := validateScopes(req.Scopes); err != nil {
			return err
		}
		updates["scopes"] = req.Scopes
	}
	if req.IPWhitelist != nil {
		updates["ip_whitelist"] = req.IPWhitelist
	}

	if err := s.db.WithContext(ctx).Model(&apiKey).Updates(updates).Error; err != nil {
		return apperrors.DatabaseError(err)
	}

	return nil
}

// DeleteAPIKey 删除API密钥
func (s *apiKeyServiceImpl) DeleteAPIKey(ctx context.Context, id string) error {
	// 检查密钥是否存在
	var apiKey models.APIKey
	if err := s.db.WithContext(ctx).First(&apiKey, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.Wrap(nil, apperrors.CodeParamError, "密钥不存在")
		}
		return apperrors.DatabaseError(err)
	}

	// 软删除
	if err := s.db.WithContext(ctx).Delete(&apiKey).Error; err != nil {
		return apperrors.DatabaseError(err)
	}

	return nil
}

// ToggleAPIKeyStatus 切换API密钥状态
func (s *apiKeyServiceImpl) ToggleAPIKeyStatus(ctx context.Context, id string) error {
	// 检查密钥是否存在
	var apiKey models.APIKey
	if err := s.db.WithContext(ctx).First(&apiKey, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.Wrap(nil, apperrors.CodeParamError, "密钥不存在")
		}
		return apperrors.DatabaseError(err)
	}

	// 切换状态
	apiKey.IsActive = !apiKey.IsActive
	if err := s.db.WithContext(ctx).Save(&apiKey).Error; err != nil {
		return apperrors.DatabaseError(err)
	}

	return nil
}

// ListUsageLogsParams 使用日志查询参数
type ListUsageLogsParams struct {
	APIKeyID  string   // 密钥ID
	Current   int      // 当前页
	PageSize  int      // 每页数量
	StartTime *string  // 开始时间
	EndTime   *string  // 结束时间
	Success   *bool    // 成功筛选
}

// UsageLogsPageResult 使用日志分页结果
type UsageLogsPageResult struct {
	List     []models.APIKeyUsageLog `json:"list"`
	Total    int64                    `json:"total"`
	Current  int                      `json:"current"`
	PageSize int                      `json:"pageSize"`
}

// GetOffset 计算分页偏移量
func (p ListUsageLogsParams) GetOffset() int {
	return (p.Current - 1) * p.PageSize
}

// ListUsageLogs 查询API密钥使用日志
func (s *apiKeyServiceImpl) ListUsageLogs(ctx context.Context, params ListUsageLogsParams) (*UsageLogsPageResult, error) {
	var total int64
	var list []models.APIKeyUsageLog

	query := s.db.WithContext(ctx).Model(&models.APIKeyUsageLog{})

	// 添加筛选条件
	if params.APIKeyID != "" {
		query = query.Where("api_key_id = ?", params.APIKeyID)
	}
	if params.StartTime != nil && *params.StartTime != "" {
		query = query.Where("created_at >= ?", *params.StartTime)
	}
	if params.EndTime != nil && *params.EndTime != "" {
		query = query.Where("created_at <= ?", *params.EndTime)
	}
	if params.Success != nil {
		query = query.Where("success = ?", *params.Success)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 分页查询
	offset := params.GetOffset()
	if err := query.
		Preload("APIKey").
		Preload("User").
		Order("created_at DESC").
		Offset(offset).Limit(params.PageSize).
		Find(&list).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	return &UsageLogsPageResult{
		List:     list,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// UsageSummary 使用统计汇总
type UsageSummary struct {
	TotalRequests    int64            `json:"totalRequests"`    // 总请求数
	SuccessRate      float64          `json:"successRate"`      // 成功率
	AvgDuration      float64          `json:"avgDuration"`      // 平均耗时
	RequestsByMethod map[string]int64 `json:"requestsByMethod"` // 按方法分组统计
	RequestsByPath   map[string]int64 `json:"requestsByPath"`   // 按路径分组统计（TOP 10）
	ErrorsByStatus   map[int]int      `json:"errorsByStatus"`   // 按状态码分组错误统计
}

// GetUsageLogSummary 获取API密钥使用统计汇总
func (s *apiKeyServiceImpl) GetUsageLogSummary(ctx context.Context, apiKeyID string) (*UsageSummary, error) {
	// 定义统计结果结构
	type MethodStat struct {
		Method string
		Count  int64
	}
	type PathStat struct {
		Path  string
		Count int64
	}
	type ErrorStat struct {
		StatusCode int
		Count      int64
	}

	// 1. 计算总请求数和成功率
	var totalRequests int64
	var successCount int64

	if err := s.db.WithContext(ctx).Model(&models.APIKeyUsageLog{}).
		Where("api_key_id = ?", apiKeyID).
		Count(&totalRequests).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 处理空结果
	if totalRequests == 0 {
		return &UsageSummary{
			TotalRequests:    0,
			SuccessRate:      0,
			AvgDuration:      0,
			RequestsByMethod: make(map[string]int64),
			RequestsByPath:   make(map[string]int64),
			ErrorsByStatus:   make(map[int]int),
		}, nil
	}

	if err := s.db.WithContext(ctx).Model(&models.APIKeyUsageLog{}).
		Where("api_key_id = ? AND success = ?", apiKeyID, true).
		Count(&successCount).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	successRate := (float64(successCount) / float64(totalRequests)) * 100

	// 2. 计算平均耗时
	var avgDuration float64
	if err := s.db.WithContext(ctx).Model(&models.APIKeyUsageLog{}).
		Select("AVG(duration)").
		Where("api_key_id = ?", apiKeyID).
		Scan(&avgDuration).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 3. 按方法分组统计
	var methodStats []MethodStat
	if err := s.db.WithContext(ctx).Model(&models.APIKeyUsageLog{}).
		Select("method, COUNT(*) as count").
		Where("api_key_id = ?", apiKeyID).
		Group("method").
		Scan(&methodStats).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	requestsByMethod := make(map[string]int64)
	for _, stat := range methodStats {
		requestsByMethod[stat.Method] = stat.Count
	}

	// 4. 按路径分组统计（TOP 10）
	var pathStats []PathStat
	if err := s.db.WithContext(ctx).Model(&models.APIKeyUsageLog{}).
		Select("path, COUNT(*) as count").
		Where("api_key_id = ?", apiKeyID).
		Group("path").
		Order("count DESC").
		Limit(10).
		Scan(&pathStats).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	requestsByPath := make(map[string]int64)
	for _, stat := range pathStats {
		requestsByPath[stat.Path] = stat.Count
	}

	// 5. 按状态码分组错误统计
	var errorStats []ErrorStat
	if err := s.db.WithContext(ctx).Model(&models.APIKeyUsageLog{}).
		Select("status_code, COUNT(*) as count").
		Where("api_key_id = ? AND success = ?", apiKeyID, false).
		Group("status_code").
		Scan(&errorStats).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	errorsByStatus := make(map[int]int)
	for _, stat := range errorStats {
		errorsByStatus[stat.StatusCode] = int(stat.Count)
	}

	return &UsageSummary{
		TotalRequests:    totalRequests,
		SuccessRate:      successRate,
		AvgDuration:      avgDuration,
		RequestsByMethod: requestsByMethod,
		RequestsByPath:   requestsByPath,
		ErrorsByStatus:   errorsByStatus,
	}, nil
}
