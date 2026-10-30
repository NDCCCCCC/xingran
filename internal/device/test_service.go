package device

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// TestService 新架构测试服务
type TestService struct {
	db        *gorm.DB
	sm4Cipher *crypto.SM4Cipher
	pool      *DeviceConnectionPool
	scheduler *DeviceTaskScheduler
	executor  *DeviceExecutor
}

// NewTestService 创建测试服务
func NewTestService(db *gorm.DB, sm4Cipher *crypto.SM4Cipher) *TestService {
	// 创建连接池
	poolConfig := &PoolConfig{
		MaxIdle:        5 * time.Minute,
		MaxConnections: 50,
	}
	pool := NewDeviceConnectionPool(db, sm4Cipher, poolConfig)

	// 创建任务调度器
	schedulerConfig := &SchedulerConfig{
		TaskTimeout: 5 * time.Minute,
		QueueSize:   100,
	}
	scheduler := NewDeviceTaskScheduler(pool, schedulerConfig)

	// 创建执行器
	executorConfig := &ExecutionConfig{
		MaxRetries:          2,
		RetryDelay:          time.Second,
		Timeout:             30 * time.Second,
		EnablePanicRecovery: true,
	}
	executor := NewDeviceExecutor(scheduler, executorConfig)

	return &TestService{
		db:        db,
		sm4Cipher: sm4Cipher,
		pool:      pool,
		scheduler: scheduler,
		executor:  executor,
	}
}

// TestConnection 测试单个设备连接
func (s *TestService) TestConnection(ctx context.Context, deviceID string) error {
	applogger.Infof("[测试服务] 测试设备连接: deviceID=%s", deviceID)

	// 查询设备信息
	var device models.NetworkDevice
	if err := s.db.Where("id = ?", deviceID).First(&device).Error; err != nil {
		return fmt.Errorf("查询设备失败: %w", err)
	}

	// 执行简单命令测试连接
	command := "display version"
	if device.Vendor == models.VendorRuijie || device.Vendor == models.VendorMaipu {
		command = "show version"
	}

	output, err := s.executor.ExecuteOnDevice(ctx, deviceID, command, true)
	if err != nil {
		return fmt.Errorf("执行命令失败: %w", err)
	}

	applogger.Infof("[测试服务] 设备 %s 连接测试成功，输出长度=%d", device.DeviceName, len(output))
	return nil
}

// TestAllDevices 测试所有设备连接
func (s *TestService) TestAllDevices(ctx context.Context) ([]*TestResult, error) {
	applogger.Infof("[测试服务] 开始测试所有设备连接...")

	// 查询所有在线设备
	var devices []models.NetworkDevice
	if err := s.db.Where("status = ?", "online").Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("查询设备列表失败: %w", err)
	}

	applogger.Infof("[测试服务] 找到 %d 个在线设备", len(devices))

	results := make([]*TestResult, 0, len(devices))

	for _, device := range devices {
		// 跳过没有凭证的设备
		if device.CredentialID == nil || *device.CredentialID == "" {
			applogger.Infof("[测试服务] 跳过设备 %s: 没有关联凭证", device.DeviceName)
			results = append(results, &TestResult{
				DeviceID:   device.ID,
				DeviceName: device.DeviceName,
				IPAddress:  device.IPAddress,
				Success:    false,
				Message:    "没有关联凭证",
			})
			continue
		}

		result := &TestResult{
			DeviceID:   device.ID,
			DeviceName: device.DeviceName,
			IPAddress:  device.IPAddress,
		}

		// 测试连接
		err := s.TestConnection(ctx, device.ID)
		if err != nil {
			result.Success = false
			result.Message = err.Error()
			applogger.Infof("[测试服务] 设备 %s 测试失败: %v", device.DeviceName, err)
		} else {
			result.Success = true
			result.Message = "连接成功"
			applogger.Infof("[测试服务] 设备 %s 测试成功", device.DeviceName)
		}

		results = append(results, result)

		// 避免过快连续连接
		time.Sleep(500 * time.Millisecond)
	}

	// 打印统计
	successCount := 0
	failCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failCount++
		}
	}

	applogger.Infof("[测试服务] 测试完成: 成功=%d, 失败=%d, 总计=%d", successCount, failCount, len(results))

	return results, nil
}

// TestResult 测试结果
type TestResult struct {
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	IPAddress  string `json:"ip_address"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
}

// GetStats 获取统计信息
func (s *TestService) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// 连接池统计
	poolStats := s.pool.GetStats()
	for k, v := range poolStats {
		stats["pool_"+k] = v
	}

	// 调度器统计
	schedulerStats := s.scheduler.GetStats()
	for k, v := range schedulerStats {
		stats["scheduler_"+k] = v
	}

	return stats
}

// Close 关闭测试服务
func (s *TestService) Close() error {
	applogger.Infof("[测试服务] 正在关闭测试服务...")

	// 停止调度器
	s.scheduler.Stop()

	// 关闭连接池
	if err := s.pool.Close(); err != nil {
		applogger.Infof("[测试服务] 关闭连接池失败: %v", err)
		return err
	}

	applogger.Infof("[测试服务] 测试服务已关闭")
	return nil
}
