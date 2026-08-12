package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// CacheService 缓存监控服务接口
type CacheService interface {
	// GetCacheList 获取缓存列表
	GetCacheList(ctx context.Context, params CacheListParams) ([]models.CacheInfo, int64, error)

	// GetCacheInfo 获取缓存详情
	GetCacheInfo(ctx context.Context, key string) (*models.CacheInfo, error)

	// OperateCache 操作缓存
	OperateCache(ctx context.Context, params CacheOperateParams) (interface{}, error)

	// BatchOperateCache 批量操作缓存
	BatchOperateCache(ctx context.Context, params CacheBatchOperateParams) (map[string]interface{}, error)

	// ClearCache 清空缓存
	ClearCache(ctx context.Context) error

	// GetCacheStats 获取缓存统计
	GetCacheStats(ctx context.Context, params CacheStatsParams) ([]models.CacheStats, int64, error)

	// GetCacheMonitor 获取缓存监控数据
	GetCacheMonitor(ctx context.Context) (map[string]interface{}, error)

	// ExportCache 导出缓存数据
	ExportCache(ctx context.Context, params CacheExportParams) ([]models.CacheInfo, error)

	// GetCacheConfigs 获取缓存配置列表
	GetCacheConfigs(ctx context.Context) (map[string]CacheConfigInfo, map[string]int, error)

	// UpdateCacheConfig 更新缓存配置
	UpdateCacheConfig(ctx context.Context, key string, value int) error

	// ReloadCacheConfigs 重新加载缓存配置
	ReloadCacheConfigs(ctx context.Context) error
}

// CacheProvider 缓存提供者接口
type CacheProvider interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)
	Keys(ctx context.Context, pattern string) ([]string, error)
	FlushDB(ctx context.Context) error
}

// CacheConfigProvider 缓存配置提供者接口
type CacheConfigProvider interface {
	GetConfigInfo() map[string]CacheConfigInfo
	GetAllConfigs(ctx context.Context) map[string]int
	ReloadConfig(ctx context.Context) error
}

// MultiLevelCacheProvider 多级缓存提供者接口
type MultiLevelCacheProvider interface {
	KeysByLevel(ctx context.Context, pattern string, level string) ([]string, error)
}

// DirectRedisProvider 直接 Redis 访问接口
type DirectRedisProvider interface {
	DirectRedisKeys(ctx context.Context, pattern string) ([]string, error)
	DirectRedisGet(ctx context.Context, key string) (string, error)
	DirectRedisTTL(ctx context.Context, key string) (time.Duration, error)
}

// StatsProvider 统计信息提供者接口
type StatsProvider interface {
	GetStats(ctx context.Context) (map[string]interface{}, error)
}

// ==================== 请求参数类型 ====================

// CacheListParams 缓存列表查询参数
type CacheListParams struct {
	Key      string
	Type     string
	Level    string // "l1", "l2", "all"
	Current  int
	PageSize int
	OrderByColumn string
	IsAsc        bool
}

// CacheOperateParams 缓存操作参数
type CacheOperateParams struct {
	Key       string
	Value     string
	TTL       int64
	Operation string // get, set, del, exists, expire, ttl
}

// CacheBatchOperateParams 批量缓存操作参数
type CacheBatchOperateParams struct {
	Keys      []string
	Operation string // get, del
}

// CacheStatsParams 缓存统计查询参数
type CacheStatsParams struct {
	CacheType  string
	StartTime  *string
	EndTime    *string
	IsRealtime bool
	Current    int
	PageSize   int
}

// CacheExportParams 缓存导出参数
type CacheExportParams struct {
	Key  string
	Type string
}

// CacheConfigInfo 缓存配置信息
type CacheConfigInfo struct {
	Key         string
	Name        string
	Description string
	Category    string
	Min         int
	Max         int
	Default     int
}

// ==================== 服务实现 ====================
// cacheAllowedSortFields 服务端排序白名单
var cacheAllowedSortFields = map[string]string{
	"key":       "key",
	"type":      "type",
	"location":  "location",
	"ttl":       "ttl",
	"size":      "size",
	"createdAt": "created_at",
	"updatedAt": "updated_at",
}


// cacheServiceImpl 缓存服务实现

type cacheServiceImpl struct {
	db              *gorm.DB
	cacheProvider   CacheProvider
	configProvider  CacheConfigProvider
	multiLevelCache MultiLevelCacheProvider
	directRedis     DirectRedisProvider
	statsProvider   StatsProvider
}

// NewCacheService 创建缓存服务实例
func NewCacheService(
	db *gorm.DB,
	cacheProvider CacheProvider,
	configProvider CacheConfigProvider,
) CacheService {
	svc := &cacheServiceImpl{
		db:             db,
		cacheProvider:  cacheProvider,
		configProvider: configProvider,
	}

	// 尝试将 cacheProvider 转换为其他接口
	if mlc, ok := cacheProvider.(MultiLevelCacheProvider); ok {
		svc.multiLevelCache = mlc
	}
	if dr, ok := cacheProvider.(DirectRedisProvider); ok {
		svc.directRedis = dr
	}
	if sp, ok := cacheProvider.(StatsProvider); ok {
		svc.statsProvider = sp
	}

	return svc
}

// GetCacheList 获取缓存列表
func (s *cacheServiceImpl) GetCacheList(ctx context.Context, params CacheListParams) ([]models.CacheInfo, int64, error) {
	if s.cacheProvider == nil {
		return s.getCacheListFromDB(ctx, params)
	}

	caches, err := s.getCachesFromCacheWithLevel(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(caches))
	start := (params.Current - 1) * params.PageSize
	end := start + params.PageSize
	if end > len(caches) {
		end = len(caches)
	}

	var paginatedCaches []models.CacheInfo
	if start < len(caches) {
		paginatedCaches = caches[start:end]
	}

	return paginatedCaches, total, nil
}

// getCacheListFromDB 从数据库获取缓存列表
func (s *cacheServiceImpl) getCacheListFromDB(ctx context.Context, params CacheListParams) ([]models.CacheInfo, int64, error) {
	db := s.db.WithContext(ctx).Model(&models.CacheInfo{})

	if params.Key != "" {
		db = db.Where("key LIKE ?", "%"+params.Key+"%")
	}
	if params.Type != "" {
		db = db.Where("type = ?", params.Type)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var caches []models.CacheInfo
	offset := (params.Current - 1) * params.PageSize
	// Apply server-side sort
	orderClause := "created_at DESC"
	if params.OrderByColumn != "" {
		if col, ok := cacheAllowedSortFields[params.OrderByColumn]; ok {
			direction := "DESC"
			if params.IsAsc {
				direction = "ASC"
			}
			orderClause = fmt.Sprintf("%s %s", col, direction)
		}
	}
	if err := db.Offset(offset).Limit(params.PageSize).Order(orderClause).Find(&caches).Error; err != nil {
		return nil, 0, err
	}

	return caches, total, nil
}

// getCachesFromCacheWithLevel 从缓存获取列表（支持层级）
func (s *cacheServiceImpl) getCachesFromCacheWithLevel(ctx context.Context, params CacheListParams) ([]models.CacheInfo, error) {
	if s.multiLevelCache == nil {
		return s.getCachesFromSimpleCache(ctx, params)
	}

	pattern := "*"
	if params.Key != "" {
		pattern = "*" + params.Key + "*"
	}

	var keys []string
	var err error
	var useDirectRedis bool

	// 尝试使用直接 Redis 访问（仅用于查询 L2）
	// 注意：查询 all 或 l1 时，不使用直接 Redis 访问，以确保获取所有层级的数据
	if params.Level == "l2" || params.Level == "L2" {
		if s.directRedis != nil {
			allRedisKeys, redisErr := s.directRedis.DirectRedisKeys(ctx, pattern)
			if redisErr == nil && len(allRedisKeys) > 0 {
				keys = allRedisKeys
				useDirectRedis = true
			}
		}
	}

	if !useDirectRedis {
		switch params.Level {
		case "l1":
			keys, err = s.multiLevelCache.KeysByLevel(ctx, pattern, "l1")
		case "l2":
			keys, err = s.multiLevelCache.KeysByLevel(ctx, pattern, "l2")
		default: // "all" 或其他值
			keys, err = s.multiLevelCache.KeysByLevel(ctx, pattern, "all")
		}
		if err != nil {
			return []models.CacheInfo{}, nil
		}
	}

	var caches []models.CacheInfo
	for _, key := range keys {
		displayKey := normalizeCacheKeyForService(key)

		if isSystemKeyForService(displayKey) {
			continue
		}

		var value string
		var ttl time.Duration
		var getErr error

		if useDirectRedis && s.directRedis != nil {
			value, getErr = s.directRedis.DirectRedisGet(ctx, key)
			if getErr == nil {
				ttl, _ = s.directRedis.DirectRedisTTL(ctx, key)
			}
		} else {
			value, getErr = s.cacheProvider.Get(ctx, displayKey)
			if getErr == nil {
				ttl, _ = s.cacheProvider.TTL(ctx, displayKey)
			}
		}

		if getErr != nil || value == "" {
			continue
		}

		ttlSeconds := int64(-1)
		if ttl > 0 {
			ttlSeconds = int64(ttl.Seconds())
		}

		location := "l2"
		if params.Level == "l1" {
			location = "l1"
		} else if params.Level == "all" && s.multiLevelCache != nil {
			l1Keys, _ := s.multiLevelCache.KeysByLevel(ctx, displayKey, "l1")
			if len(l1Keys) > 0 {
				location = "both"
			}
		}

		caches = append(caches, models.CacheInfo{
			Key:       displayKey,
			Value:     value,
			Type:      "string",
			Size:      int64(len(value)),
			TTL:       ttlSeconds,
			Location:  location,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}

	return caches, nil
}

// getCachesFromSimpleCache 从简单缓存获取列表
func (s *cacheServiceImpl) getCachesFromSimpleCache(ctx context.Context, params CacheListParams) ([]models.CacheInfo, error) {
	pattern := "*"
	if params.Key != "" {
		pattern = "*" + params.Key + "*"
	}

	keys, err := s.cacheProvider.Keys(ctx, pattern)
	if err != nil {
		return []models.CacheInfo{}, nil
	}

	var caches []models.CacheInfo
	for _, key := range keys {
		value, err := s.cacheProvider.Get(ctx, key)
		if err != nil {
			continue
		}

		ttl, _ := s.cacheProvider.TTL(ctx, key)
		ttlSeconds := int64(-1)
		if ttl > 0 {
			ttlSeconds = int64(ttl.Seconds())
		}

		caches = append(caches, models.CacheInfo{
			Key:       key,
			Value:     value,
			Type:      "string",
			Size:      int64(len(value)),
			TTL:       ttlSeconds,
			Location:  "l2", // 简单缓存默认为 Redis (L2)
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}

	return caches, nil
}

// GetCacheInfo 获取缓存详情
func (s *cacheServiceImpl) GetCacheInfo(ctx context.Context, key string) (*models.CacheInfo, error) {
	if key == "" {
		return nil, ErrCacheKeyRequired
	}

	key = normalizeCacheKeyForService(key)

	if s.cacheProvider != nil {
		value, err := s.cacheProvider.Get(ctx, key)
		if err != nil {
			return nil, err
		}

		if value != "" {
			ttl, _ := s.cacheProvider.TTL(ctx, key)
			ttlSeconds := int64(-1)
			if ttl > 0 {
				ttlSeconds = int64(ttl.Seconds())
			}

			return &models.CacheInfo{
				Key:       key,
				Value:     value,
				Type:      "string",
				TTL:       ttlSeconds,
				Size:      int64(len(value)),
				Location:  "l2", // 默认为 Redis (L2)
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		}
	}

	var cache models.CacheInfo
	err := s.db.WithContext(ctx).Where("key = ?", key).First(&cache).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrCacheNotFound
		}
		return nil, err
	}

	return &cache, nil
}

// OperateCache 操作缓存
func (s *cacheServiceImpl) OperateCache(ctx context.Context, params CacheOperateParams) (interface{}, error) {
	if s.cacheProvider == nil {
		return nil, ErrCacheServiceUnavailable
	}

	var result interface{}
	var err error

	switch params.Operation {
	case "get":
		result, err = s.cacheProvider.Get(ctx, params.Key)
		if err != nil {
			return nil, err
		}

	case "set":
		if params.Value == "" {
			return nil, ErrCacheValueRequired
		}

		ttl := time.Hour
		if params.TTL > 0 {
			ttl = time.Duration(params.TTL) * time.Second
		}

		err = s.cacheProvider.Set(ctx, params.Key, params.Value, ttl)
		if err != nil {
			return nil, err
		}
		result = "设置成功"

	case "del":
		deleteKey := normalizeCacheKeyForService(params.Key)
		err = s.cacheProvider.Delete(ctx, deleteKey)
		if err != nil {
			return nil, err
		}
		result = "删除成功"

	case "exists":
		checkKey := normalizeCacheKeyForService(params.Key)
		exists, err := s.cacheProvider.Exists(ctx, checkKey)
		if err != nil {
			return nil, err
		}
		result = map[string]interface{}{"exists": exists}

	case "expire":
		if params.TTL <= 0 {
			return nil, ErrCacheTTLRequired
		}
		expireKey := normalizeCacheKeyForService(params.Key)
		err = s.cacheProvider.Expire(ctx, expireKey, time.Duration(params.TTL)*time.Second)
		if err != nil {
			return nil, err
		}
		result = "设置成功"

	case "ttl":
		result = map[string]interface{}{"ttl": -1}

	default:
		return nil, ErrOperationUnsupported
	}

	return result, nil
}

// BatchOperateCache 批量操作缓存
func (s *cacheServiceImpl) BatchOperateCache(ctx context.Context, params CacheBatchOperateParams) (map[string]interface{}, error) {
	if s.cacheProvider == nil {
		return nil, ErrCacheServiceUnavailable
	}

	if len(params.Keys) == 0 {
		return nil, ErrCacheKeysRequired
	}

	var results map[string]interface{}

	switch params.Operation {
	case "get":
		values := make(map[string]interface{})
		for _, key := range params.Keys {
			value, err := s.cacheProvider.Get(ctx, key)
			if err != nil {
				values[key] = "获取失败: " + err.Error()
			} else {
				values[key] = value
			}
		}
		results = values

	case "del":
		deletedKeys := make([]string, 0)
		failedKeys := make(map[string]string)

		for _, key := range params.Keys {
			deleteKey := normalizeCacheKeyForService(key)
			err := s.cacheProvider.Delete(ctx, deleteKey)
			if err != nil {
				failedKeys[key] = err.Error()
			} else {
				deletedKeys = append(deletedKeys, key)
			}
		}

		results = map[string]interface{}{
			"deleted": deletedKeys,
			"failed":  failedKeys,
		}

	default:
		return nil, ErrOperationUnsupported
	}

	return results, nil
}

// ClearCache 清空缓存
func (s *cacheServiceImpl) ClearCache(ctx context.Context) error {
	if s.cacheProvider == nil {
		return ErrCacheServiceUnavailable
	}

	if err := s.cacheProvider.FlushDB(ctx); err != nil {
		return err
	}

	_ = s.db.WithContext(ctx).Exec("DELETE FROM sys_cache_info")
	return nil
}

// GetCacheStats 获取缓存统计
func (s *cacheServiceImpl) GetCacheStats(ctx context.Context, params CacheStatsParams) ([]models.CacheStats, int64, error) {
	isRealtime := params.IsRealtime || (params.StartTime == nil && params.EndTime == nil)

	if isRealtime {
		return s.getRealtimeCacheStats(ctx)
	}

	return s.getHistoryCacheStats(ctx, params)
}

// getRealtimeCacheStats 获取实时缓存统计
func (s *cacheServiceImpl) getRealtimeCacheStats(ctx context.Context) ([]models.CacheStats, int64, error) {
	if s.statsProvider == nil {
		return nil, 0, ErrCacheStatsUnsupported
	}

	stats, err := s.statsProvider.GetStats(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 转换为 CacheStats 格式
	var cacheStatsList []models.CacheStats

	if l1Stats, ok := stats["l1"].(map[string]interface{}); ok {
		cacheStatsList = append(cacheStatsList, convertToCacheStats(l1Stats, "L1(内存)"))
	}

	if l2Stats, ok := stats["l2"].(map[string]interface{}); ok {
		cacheStatsList = append(cacheStatsList, convertToCacheStats(l2Stats, "L2(Redis)"))
	}

	return cacheStatsList, int64(len(cacheStatsList)), nil
}

// getHistoryCacheStats 获取历史缓存统计
func (s *cacheServiceImpl) getHistoryCacheStats(ctx context.Context, params CacheStatsParams) ([]models.CacheStats, int64, error) {
	db := s.db.WithContext(ctx).Model(&models.CacheStats{})

	if params.CacheType != "" {
		db = db.Where("cache_type = ?", params.CacheType)
	}
	if params.StartTime != nil && *params.StartTime != "" {
		db = db.Where("collect_time >= ?", *params.StartTime)
	}
	if params.EndTime != nil && *params.EndTime != "" {
		db = db.Where("collect_time <= ?", *params.EndTime)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var cacheStatsList []models.CacheStats
	offset := (params.Current - 1) * params.PageSize
	if err := db.Offset(offset).Limit(params.PageSize).Order("collect_time DESC").Find(&cacheStatsList).Error; err != nil {
		return nil, 0, err
	}

	return cacheStatsList, total, nil
}

// GetCacheMonitor 获取缓存监控数据
func (s *cacheServiceImpl) GetCacheMonitor(ctx context.Context) (map[string]interface{}, error) {
	if s.statsProvider == nil {
		return nil, ErrCacheStatsUnsupported
	}

	stats, err := s.statsProvider.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	monitor := make(map[string]interface{})

	if l1Stats, ok := stats["l1"].(map[string]interface{}); ok {
		formattedL1Stats := formatCacheStatsForService(l1Stats)
		monitor["l1"] = map[string]interface{}{
			"status": map[string]interface{}{
				"connected": true,
				"type":      "memory",
			},
			"stats": formattedL1Stats,
		}
	}

	if l2Stats, ok := stats["l2"].(map[string]interface{}); ok {
		formattedL2Stats := formatCacheStatsForService(l2Stats)
		serverInfo := map[string]interface{}{
			"connected": true,
			"type":      "redis",
		}
		if version, ok := l2Stats["redis_version"].(string); ok {
			serverInfo["version"] = version
		}
		if uptime, ok := l2Stats["uptime_in_seconds"].(int64); ok {
			serverInfo["uptime"] = fmt.Sprintf("%ds", uptime)
		}

		monitor["l2"] = map[string]interface{}{
			"status": serverInfo,
			"stats":  formattedL2Stats,
		}
	}

	return monitor, nil
}

// ExportCache 导出缓存数据
func (s *cacheServiceImpl) ExportCache(ctx context.Context, params CacheExportParams) ([]models.CacheInfo, error) {
	db := s.db.WithContext(ctx).Model(&models.CacheInfo{})

	if params.Key != "" {
		db = db.Where("key LIKE ?", "%"+params.Key+"%")
	}
	if params.Type != "" {
		db = db.Where("type = ?", params.Type)
	}

	var caches []models.CacheInfo
	if err := db.Order("created_at DESC").Find(&caches).Error; err != nil {
		return nil, err
	}

	return caches, nil
}

// GetCacheConfigs 获取缓存配置列表
func (s *cacheServiceImpl) GetCacheConfigs(ctx context.Context) (map[string]CacheConfigInfo, map[string]int, error) {
	if s.configProvider == nil {
		return nil, nil, ErrCacheConfigUnavailable
	}

	configInfoMap := s.configProvider.GetConfigInfo()
	currentValues := s.configProvider.GetAllConfigs(ctx)

	return configInfoMap, currentValues, nil
}

// UpdateCacheConfig 更新缓存配置
func (s *cacheServiceImpl) UpdateCacheConfig(ctx context.Context, key string, value int) error {
	if s.configProvider == nil {
		return ErrCacheConfigUnavailable
	}

	configInfoMap := s.configProvider.GetConfigInfo()
	configInfo, exists := configInfoMap[key]
	if !exists {
		return ErrInvalidConfigKey
	}

	if value < configInfo.Min || value > configInfo.Max {
		return fmt.Errorf("配置值必须在 %d 到 %d 分钟之间", configInfo.Min, configInfo.Max)
	}

	// 更新数据库中的配置
	var config models.Config
	err := s.db.WithContext(ctx).Where("config_key = ?", key).First(&config).Error

	if err == gorm.ErrRecordNotFound {
		config = models.Config{
			ConfigKey:   key,
			ConfigName:  configInfo.Name,
			ConfigValue: fmt.Sprintf("%d", value),
			ConfigType:  models.ConfigTypeYes,
			Remark:      configInfo.Description,
		}
		err = s.db.WithContext(ctx).Create(&config).Error
	} else if err == nil {
		config.ConfigValue = fmt.Sprintf("%d", value)
		err = s.db.WithContext(ctx).Save(&config).Error
	}

	if err != nil {
		return err
	}

	// 重新加载配置
	return s.configProvider.ReloadConfig(ctx)
}

// ReloadCacheConfigs 重新加载缓存配置
func (s *cacheServiceImpl) ReloadCacheConfigs(ctx context.Context) error {
	if s.configProvider == nil {
		return ErrCacheConfigUnavailable
	}

	return s.configProvider.ReloadConfig(ctx)
}

// ==================== 辅助函数 ====================

// normalizeCacheKeyForService 规范化缓存键
func normalizeCacheKeyForService(key string) string {
	if len(key) > 6 && key[:6] == "xingran:" {
		return key[6:]
	}
	return key
}

// isSystemKeyForService 判断是否为系统键
func isSystemKeyForService(key string) bool {
	systemKeyPrefixes := []string{
		"__:", "redis:", "monitor:", "perf:", "stats:",
		"cluster:", "node:", "replication:", "sentinel:",
	}

	lowerKey := string(key)
	for i := 0; i < len(systemKeyPrefixes); i++ {
		prefix := systemKeyPrefixes[i]
		if len(lowerKey) >= len(prefix) && lowerKey[:len(prefix)] == prefix {
			return true
		}
	}

	if len(key) > 0 && key[0] >= '0' && key[0] <= '9' {
		return true
	}

	return false
}

// formatCacheStatsForService 格式化缓存统计信息
func formatCacheStatsForService(stats map[string]interface{}) map[string]interface{} {
	hitCount, _ := stats["keyspace_hits"].(int64)
	missCount, _ := stats["keyspace_misses"].(int64)
	hitRate, _ := stats["hit_rate"].(float64)
	usedMemory, _ := stats["used_memory"].(int64)
	keyCount, _ := stats["key_count"].(int64)

	var totalMemory int64
	if totalMem, ok := stats["total_system_memory"].(int64); ok && totalMem > 0 {
		totalMemory = totalMem
	} else if maxMem, ok := stats["maxmemory"].(int64); ok && maxMem > 0 {
		totalMemory = maxMem
	} else {
		totalMemory = usedMemory * 2
	}

	return map[string]interface{}{
		"hitCount":    hitCount,
		"missCount":   missCount,
		"hitRate":     hitRate,
		"totalMemory": totalMemory,
		"usedMemory":  usedMemory,
		"keyCount":    keyCount,
	}
}

// convertToCacheStats 转换为 CacheStats
func convertToCacheStats(stats map[string]interface{}, cacheType string) models.CacheStats {
	hitCount, _ := stats["keyspace_hits"].(int64)
	missCount, _ := stats["keyspace_misses"].(int64)
	hitRate, _ := stats["hit_rate"].(float64)
	usedMemory, _ := stats["used_memory"].(int64)
	keyCount, _ := stats["key_count"].(int64)

	return models.CacheStats{
		CacheType:   cacheType,
		HitCount:    hitCount,
		MissCount:   missCount,
		HitRate:     hitRate,
		UsedMemory:  usedMemory,
		TotalMemory: usedMemory * 2,
		KeyCount:    keyCount,
		CollectTime: time.Now(),
	}
}

// ==================== 错误定义 ====================

var (
	ErrCacheKeyRequired        = fmt.Errorf("缓存键不能为空")
	ErrCacheServiceUnavailable = fmt.Errorf("缓存服务不可用")
	ErrCacheNotFound           = fmt.Errorf("缓存不存在")
	ErrCacheValueRequired      = fmt.Errorf("设置缓存值不能为空")
	ErrCacheTTLRequired        = fmt.Errorf("过期时间必须大于0")
	ErrOperationUnsupported    = fmt.Errorf("不支持的操作")
	ErrCacheKeysRequired       = fmt.Errorf("缓存键列表不能为空")
	ErrCacheStatsUnsupported   = fmt.Errorf("缓存不支持统计信息")
	ErrCacheConfigUnavailable  = fmt.Errorf("缓存配置服务不可用")
	ErrInvalidConfigKey        = fmt.Errorf("无效的配置键")
)
