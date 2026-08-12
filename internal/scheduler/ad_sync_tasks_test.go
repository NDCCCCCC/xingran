package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/stretchr/testify/assert"
)

// TestADSyncScheduler_New 测试创建 AD 同步调度器
func TestADSyncScheduler_New(t *testing.T) {
	// 使用 nil DB 进行基本功能测试
	scheduler := NewADSyncScheduler(nil, 5)

	assert.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.cron)
	assert.NotNil(t, scheduler.sem)
	assert.Nil(t, scheduler.db) // db 可以为 nil 用于测试
	assert.False(t, scheduler.IsStarted())
}

// TestADSyncScheduler_StartStop 测试启动和停止调度器
func TestADSyncScheduler_StartStop(t *testing.T) {
	scheduler := NewADSyncScheduler(nil, 5)

	// 测试启动
	scheduler.Start()
	assert.True(t, scheduler.IsStarted())

	// 测试重复启动（应该幂等）
	scheduler.Start()
	assert.True(t, scheduler.IsStarted())

	// 测试停止
	scheduler.Stop()
	assert.False(t, scheduler.IsStarted())
}

// TestADSyncScheduler_ParallelStartStop 测试并发启动和停止
func TestADSyncScheduler_ParallelStartStop(t *testing.T) {
	scheduler := NewADSyncScheduler(nil, 5)

	// 并发启动
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			scheduler.Start()
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.True(t, scheduler.IsStarted())

	// 并发停止
	for i := 0; i < 10; i++ {
		go func() {
			scheduler.Stop()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	assert.False(t, scheduler.IsStarted())
}

// TestGlobalADSyncScheduler 测试全局调度器
func TestGlobalADSyncScheduler(t *testing.T) {
	// 保存原有状态
	oldScheduler := globalADSyncScheduler
	defer func() {
		globalADSyncScheduler = oldScheduler
	}()

	// 确保初始状态为 nil
	globalADSyncScheduler = nil
	assert.Nil(t, globalADSyncScheduler)

	// 设置全局调度器
	globalADSyncScheduler = NewADSyncScheduler(nil, constants.MaxConcurrentADSync)
	globalADSyncScheduler.Start()

	assert.NotNil(t, globalADSyncScheduler)
	assert.True(t, globalADSyncScheduler.IsStarted())

	// 重复启动应该幂等
	globalADSyncScheduler.Start()
	assert.NotNil(t, GetADSyncScheduler())

	// 停止全局调度器
	StopADSyncScheduler()
	if globalADSyncScheduler != nil {
		assert.False(t, globalADSyncScheduler.IsStarted())
	}
}

// TestADSyncScheduler_SemaphoreAcquire 测试信号量获取
func TestADSyncScheduler_SemaphoreAcquire(t *testing.T) {
	scheduler := NewADSyncScheduler(nil, 2) // 最大并发为 2

	ctx := context.Background()
	acquired := make(chan bool, 3)

	// 尝试获取 3 次信号量
	for i := 0; i < 3; i++ {
		go func(n int) {
			err := scheduler.sem.Acquire(ctx, 1)
			if err == nil {
				acquired <- true
				time.Sleep(100 * time.Millisecond)
				scheduler.sem.Release(1)
			} else {
				acquired <- false
			}
		}(i)
	}

	// 前两个应该成功获取
	timeout := time.After(1 * time.Second)
	successCount := 0
	for i := 0; i < 3; i++ {
		select {
		case ok := <-acquired:
			if ok {
				successCount++
			}
		case <-timeout:
			break
		}
	}

	assert.Equal(t, 3, successCount, "所有 goroutine 应该最终成功获取信号量")
}

// BenchmarkADSyncScheduler_IsStarted 基准测试
func BenchmarkADSyncScheduler_IsStarted(b *testing.B) {
	scheduler := NewADSyncScheduler(nil, constants.MaxConcurrentADSync)
	scheduler.Start()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scheduler.IsStarted()
	}
}
