package network

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"gorm.io/gorm"
)

// CacheService 网络设备缓存服务接口
type CacheService interface {
	// 基础服务方法
	List(ctx context.Context, req *services.ListDeviceRequest) ([]models.NetworkDevice, int64, error)
	GetByID(ctx context.Context, id string) (*models.NetworkDevice, error)
	Create(ctx context.Context, req *services.CreateDeviceRequest) (*models.NetworkDevice, error)
	QuickCreateDevice(ctx context.Context, req *services.QuickCreateRequest) (*models.NetworkDevice, error)
	Update(ctx context.Context, req *services.UpdateDeviceRequest) error
	Delete(ctx context.Context, id string) error
	BatchDelete(ctx context.Context, ids []string) error
	UpdateStatus(ctx context.Context, id string, status models.DeviceStatus) error
	UpdateStatusBatch(ctx context.Context, ids []string, status models.DeviceStatus) error

	// 高频查询方法（带缓存）
	GetDeviceStatistics(ctx context.Context) (map[string]interface{}, error)
	GetDevicesByDept(ctx context.Context, deptID string) ([]models.NetworkDevice, error)
	GetDevicesByCredential(ctx context.Context, credentialID string) ([]models.NetworkDevice, error)

	// 缓存失效方法
	InvalidateDeviceCache(ctx context.Context, deviceID string) error
	InvalidateStatisticsCache(ctx context.Context) error
	InvalidateDeptCache(ctx context.Context, deptID string) error
	InvalidateCredentialCache(ctx context.Context, credentialID string) error
	InvalidateAllDeviceCache(ctx context.Context) error
}

// cacheServiceImpl 网络设备缓存服务实现
type cacheServiceImpl struct {
	db     *gorm.DB
	base   *services.NetworkDeviceService
	cache  systemServices.CacheProvider
	config *services.CacheConfigService
}

// NewServiceWithCache 创建带缓存的网络设备服务
func NewServiceWithCache(
	db *gorm.DB,
	discoveryService *services.DeviceDiscoveryService,
	deviceInfoCollectionSvc *services.DeviceInfoCollectionService,
	cache systemServices.CacheProvider,
	config *services.CacheConfigService,
) CacheService {
	return &cacheServiceImpl{
		db:     db,
		base:   services.NewNetworkDeviceService(db, discoveryService, deviceInfoCollectionSvc),
		cache:  cache,
		config: config,
	}
}

// getExpiration 获取缓存过期时间
func (s *cacheServiceImpl) getExpiration(configKey string, defaultVal time.Duration) time.Duration {
	if s.config != nil {
		return s.config.GetDurationWithDefault(configKey, defaultVal)
	}
	return defaultVal
}

// ==================== 基础服务方法（带缓存失效） ====================

// List 获取设备列表（不缓存，参数多变）
func (s *cacheServiceImpl) List(ctx context.Context, req *services.ListDeviceRequest) ([]models.NetworkDevice, int64, error) {
	return s.base.List(ctx, req)
}

// GetByID 获取设备详情（不缓存，查询频率低）
func (s *cacheServiceImpl) GetByID(ctx context.Context, id string) (*models.NetworkDevice, error) {
	return s.base.GetByID(ctx, id)
}

// Create 创建设备（带缓存失效）
func (s *cacheServiceImpl) Create(ctx context.Context, req *services.CreateDeviceRequest) (*models.NetworkDevice, error) {
	device, err := s.base.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	// 清除统计缓存
	_ = s.InvalidateStatisticsCache(ctx)
	// 如果指定了部门，清除部门缓存
	if req.DeptID != nil && *req.DeptID != "" {
		_ = s.InvalidateDeptCache(ctx, *req.DeptID)
	}
	// 如果指定了凭证，清除凭证缓存
	if req.CredentialID != nil && *req.CredentialID != "" {
		_ = s.InvalidateCredentialCache(ctx, *req.CredentialID)
	}
	return device, nil
}

// QuickCreateDevice 快速创建设备（带缓存失效）
func (s *cacheServiceImpl) QuickCreateDevice(ctx context.Context, req *services.QuickCreateRequest) (*models.NetworkDevice, error) {
	device, err := s.base.QuickCreateDevice(ctx, req)
	if err != nil {
		return nil, err
	}
	// 清除统计缓存
	_ = s.InvalidateStatisticsCache(ctx)
	// 快速创建会指定部门和凭证，清除相应缓存
	if req.DeptID != nil && *req.DeptID != "" {
		_ = s.InvalidateDeptCache(ctx, *req.DeptID)
	}
	_ = s.InvalidateCredentialCache(ctx, req.CredentialID)
	return device, nil
}

// Update 更新设备（带缓存失效）
func (s *cacheServiceImpl) Update(ctx context.Context, req *services.UpdateDeviceRequest) error {
	// 先获取设备以便获取部门和凭证信息
	var device models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", req.ID).First(&device).Error; err != nil {
		return err
	}

	oldDeptID := device.DeptID
	oldCredentialID := device.CredentialID

	if err := s.base.Update(ctx, req); err != nil {
		return err
	}

	// 清除设备缓存
	_ = s.InvalidateDeviceCache(ctx, req.ID)

	// 如果部门变更，清除旧部门缓存
	if (oldDeptID != nil && req.DeptID != nil && *oldDeptID != *req.DeptID) ||
		(oldDeptID != nil && req.DeptID == nil) {
		if oldDeptID != nil && *oldDeptID != "" {
			_ = s.InvalidateDeptCache(ctx, *oldDeptID)
		}
	}
	// 清除新部门缓存
	if req.DeptID != nil && *req.DeptID != "" {
		_ = s.InvalidateDeptCache(ctx, *req.DeptID)
	}

	// 如果凭证变更，清除旧凭证缓存
	if (oldCredentialID != nil && req.CredentialID != nil && *oldCredentialID != *req.CredentialID) ||
		(oldCredentialID != nil && req.CredentialID == nil) {
		if oldCredentialID != nil && *oldCredentialID != "" {
			_ = s.InvalidateCredentialCache(ctx, *oldCredentialID)
		}
	}
	// 清除新凭证缓存
	if req.CredentialID != nil && *req.CredentialID != "" {
		_ = s.InvalidateCredentialCache(ctx, *req.CredentialID)
	}

	// 如果状态变更，清除统计缓存
	if req.Status != device.Status {
		_ = s.InvalidateStatisticsCache(ctx)
	}

	return nil
}

// Delete 删除设备（带缓存失效）
func (s *cacheServiceImpl) Delete(ctx context.Context, id string) error {
	// 先获取设备以便获取部门和凭证信息
	var device models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&device).Error; err != nil {
		return err
	}

	deptID := device.DeptID
	credentialID := device.CredentialID

	if err := s.base.Delete(ctx, id); err != nil {
		return err
	}

	// 清除缓存
	_ = s.InvalidateDeviceCache(ctx, id)
	if deptID != nil && *deptID != "" {
		_ = s.InvalidateDeptCache(ctx, *deptID)
	}
	if credentialID != nil && *credentialID != "" {
		_ = s.InvalidateCredentialCache(ctx, *credentialID)
	}
	_ = s.InvalidateStatisticsCache(ctx)

	return nil
}

// BatchDelete 批量删除设备（带缓存失效）
func (s *cacheServiceImpl) BatchDelete(ctx context.Context, ids []string) error {
	// 先获取所有设备以便获取部门和凭证信息
	var devices []models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&devices).Error; err != nil {
		return err
	}

	if err := s.base.BatchDelete(ctx, ids); err != nil {
		return err
	}

	// 清除所有设备缓存
	for _, id := range ids {
		_ = s.InvalidateDeviceCache(ctx, id)
	}

	// 清除所有涉及的部门和凭证缓存
	deptMap := make(map[string]bool)
	credentialMap := make(map[string]bool)
	for _, device := range devices {
		if device.DeptID != nil && *device.DeptID != "" {
			deptMap[*device.DeptID] = true
		}
		if device.CredentialID != nil && *device.CredentialID != "" {
			credentialMap[*device.CredentialID] = true
		}
	}

	for deptID := range deptMap {
		_ = s.InvalidateDeptCache(ctx, deptID)
	}
	for credentialID := range credentialMap {
		_ = s.InvalidateCredentialCache(ctx, credentialID)
	}

	// 清除统计缓存
	_ = s.InvalidateStatisticsCache(ctx)

	return nil
}

// UpdateStatus 更新设备状态（带缓存失效）
func (s *cacheServiceImpl) UpdateStatus(ctx context.Context, id string, status models.DeviceStatus) error {
	if err := s.base.UpdateStatus(ctx, id, status); err != nil {
		return err
	}
	// 清除设备缓存和统计缓存
	_ = s.InvalidateDeviceCache(ctx, id)
	_ = s.InvalidateStatisticsCache(ctx)
	return nil
}

// UpdateStatusBatch 批量更新设备状态（带缓存失效）
func (s *cacheServiceImpl) UpdateStatusBatch(ctx context.Context, ids []string, status models.DeviceStatus) error {
	if err := s.base.UpdateStatusBatch(ctx, ids, status); err != nil {
		return err
	}
	// 清除所有设备缓存和统计缓存
	for _, id := range ids {
		_ = s.InvalidateDeviceCache(ctx, id)
	}
	_ = s.InvalidateStatisticsCache(ctx)
	return nil
}

// ==================== 高频查询方法（带缓存） ====================

// GetDeviceStatistics 获取设备统计数据（带缓存）
func (s *cacheServiceImpl) GetDeviceStatistics(ctx context.Context) (map[string]interface{}, error) {
	cacheKey := "network_device:statistics"
	var result map[string]interface{}

	expiration := s.getExpiration("cache.network_device.statistics", 3*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.base.GetDeviceStatistics(ctx)
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetDevicesByDept 获取指定部门的设备列表（带缓存）
func (s *cacheServiceImpl) GetDevicesByDept(ctx context.Context, deptID string) ([]models.NetworkDevice, error) {
	cacheKey := fmt.Sprintf("network_device:dept:%s", deptID)
	var result []models.NetworkDevice

	expiration := s.getExpiration("cache.network_device.dept", 5*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.base.GetDevicesByDept(ctx, deptID)
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetDevicesByCredential 获取使用指定凭证的设备列表（带缓存）
func (s *cacheServiceImpl) GetDevicesByCredential(ctx context.Context, credentialID string) ([]models.NetworkDevice, error) {
	cacheKey := fmt.Sprintf("network_device:credential:%s", credentialID)
	var result []models.NetworkDevice

	expiration := s.getExpiration("cache.network_device.credential", 5*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.base.GetDevicesByCredential(ctx, credentialID)
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// ==================== 缓存失效方法 ====================

// InvalidateDeviceCache 失效指定设备的缓存
func (s *cacheServiceImpl) InvalidateDeviceCache(ctx context.Context, deviceID string) error {
	keys := []string{fmt.Sprintf("network_device:detail:%s", deviceID)}
	systemServices.InvalidateCacheByKey(ctx, s.cache, keys, "NETWORK_DEVICE")
	return nil
}

// InvalidateStatisticsCache 失效统计缓存
func (s *cacheServiceImpl) InvalidateStatisticsCache(ctx context.Context) error {
	keys := []string{"network_device:statistics"}
	systemServices.InvalidateCacheByKey(ctx, s.cache, keys, "NETWORK_DEVICE")
	return nil
}

// InvalidateDeptCache 失效部门设备缓存
func (s *cacheServiceImpl) InvalidateDeptCache(ctx context.Context, deptID string) error {
	keys := []string{fmt.Sprintf("network_device:dept:%s", deptID)}
	systemServices.InvalidateCacheByKey(ctx, s.cache, keys, "NETWORK_DEVICE")
	return nil
}

// InvalidateCredentialCache 失效凭证设备缓存
func (s *cacheServiceImpl) InvalidateCredentialCache(ctx context.Context, credentialID string) error {
	keys := []string{fmt.Sprintf("network_device:credential:%s", credentialID)}
	systemServices.InvalidateCacheByKey(ctx, s.cache, keys, "NETWORK_DEVICE")
	return nil
}

// InvalidateAllDeviceCache 失效所有设备缓存
func (s *cacheServiceImpl) InvalidateAllDeviceCache(ctx context.Context) error {
	systemServices.InvalidateCacheByPattern(ctx, s.cache, []string{"network_device:*"}, "NETWORK_DEVICE")
	return nil
}
