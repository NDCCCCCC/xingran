package scheduler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGlobalDeviceMonitorService_ConcurrentAccess 测试并发访问
func TestGlobalDeviceMonitorService_ConcurrentAccess(t *testing.T) {
	// 保存原有状态
	oldService := GlobalDeviceMonitorService
	defer func() {
		SetDeviceMonitorService(oldService)
	}()

	// 创建 mock service
	mockService := &mockDeviceMonitorService{}
	SetDeviceMonitorService(mockService)

	// 并发读写
	done := make(chan bool, 100)
	for i := 0; i < 50; i++ {
		go func() {
			svc := GetDeviceMonitorService()
			assert.NotNil(t, svc)
			done <- true
		}()
	}

	for i := 0; i < 50; i++ {
		go func() {
			SetDeviceMonitorService(mockService)
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 100; i++ {
		<-done
	}

	// 验证最终状态
	finalService := GetDeviceMonitorService()
	assert.NotNil(t, finalService)
}

// mockDeviceMonitorService mock 实现
type mockDeviceMonitorService struct{}

func (m *mockDeviceMonitorService) CheckAllDevicesStatus(ctx context.Context) (int, int, error) {
	return 0, 0, nil
}

func (m *mockDeviceMonitorService) CollectAllPortStatus(ctx context.Context) error {
	return nil
}

func (m *mockDeviceMonitorService) CollectAllMACAddresses(ctx context.Context) error {
	return nil
}

func (m *mockDeviceMonitorService) BackupAllConfigurations(ctx context.Context) error {
	return nil
}

// TestGlobalDeviceMonitorService_NilSafe 测试 nil 安全
func TestGlobalDeviceMonitorService_NilSafe(t *testing.T) {
	// 保存原有状态
	oldService := GlobalDeviceMonitorService
	defer func() {
		SetDeviceMonitorService(oldService)
	}()

	// 设置为 nil
	SetDeviceMonitorService(nil)

	// 获取应该返回 nil 而非 panic
	svc := GetDeviceMonitorService()
	assert.Nil(t, svc)
}
