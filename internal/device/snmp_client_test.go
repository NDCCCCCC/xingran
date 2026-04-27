package device

import (
	"sync"
	"testing"
	"time"
)

// TestSNMPClient_WaitForReady_ImmediateSuccess 测试连接已就绪时立即返回
func TestSNMPClient_WaitForReady_ImmediateSuccess(t *testing.T) {
	config := &SNMPClientConfig{
		Target:    "127.0.0.1", // 使用本地地址（不需要真实设备）
		Port:      161,
		Community: "public",
		Version:   SNMPVersion2c,
		Timeout:   1 * time.Second,
	}

	client := NewSNMPClient(config)

	// 模拟连接成功（手动设置 ready 状态）
	client.setReady(true)

	// 应该立即返回 nil
	err := client.WaitForReady(5 * time.Second)
	if err != nil {
		t.Errorf("WaitForReady 应该成功，返回错误: %v", err)
	}
}

// TestSNMPClient_WaitForReady_NotConnected 测试未连接时返回错误
func TestSNMPClient_WaitForReady_NotConnected(t *testing.T) {
	config := &SNMPClientConfig{
		Target:    "127.0.0.1",
		Port:      161,
		Community: "public",
		Version:   SNMPVersion2c,
		Timeout:   1 * time.Second,
	}

	client := NewSNMPClient(config)
	// 不调用 Connect()，保持未连接状态

	// 应该返回超时错误
	err := client.WaitForReady(500 * time.Millisecond)
	if err == nil {
		t.Error("WaitForReady 应该返回错误")
	}
}

// TestSNMPClient_WaitForReady_Timeout 测试超时机制
func TestSNMPClient_WaitForReady_Timeout(t *testing.T) {
	config := &SNMPClientConfig{
		Target:    "127.0.0.1",
		Port:      161,
		Community: "public",
		Version:   SNMPVersion2c,
		Timeout:   1 * time.Second,
	}

	client := NewSNMPClient(config)
	// 保持未就绪状态

	start := time.Now()
	err := client.WaitForReady(200 * time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("WaitForReady 应该返回超时错误")
	}
	if elapsed < 200*time.Millisecond {
		t.Errorf("WaitForReady 应该等待至少 200ms，实际等待: %v", elapsed)
	}
	if elapsed > 300*time.Millisecond {
		t.Errorf("WaitForReady 等待时间过长: %v", elapsed)
	}
}

// TestSNMPClient_WaitForReady_Concurrent 测试并发调用安全性
func TestSNMPClient_WaitForReady_Concurrent(t *testing.T) {
	config := &SNMPClientConfig{
		Target:    "127.0.0.1",
		Port:      161,
		Community: "public",
		Version:   SNMPVersion2c,
		Timeout:   1 * time.Second,
	}

	client := NewSNMPClient(config)
	client.setReady(true) // 标记为就绪

	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := client.WaitForReady(1 * time.Second); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// 检查是否有错误
	for err := range errors {
		t.Errorf("并发调用 WaitForReady 失败: %v", err)
	}
}

// TestSNMPClient_Get_WithWaitForReady 测试集成场景
func TestSNMPClient_Get_WithWaitForReady(t *testing.T) {
	config := &SNMPClientConfig{
		Target:    "127.0.0.1",
		Port:      161,
		Community: "public",
		Version:   SNMPVersion2c,
		Timeout:   1 * time.Second,
	}

	client := NewSNMPClient(config)

	// 先等待就绪（会超时，因为没有真实设备）
	err := client.WaitForReady(100 * time.Millisecond)
	if err == nil {
		t.Error("WaitForReady 应该超时（无真实设备）")
	}

	// 即使超时，也不应该 panic 或崩溃
	t.Log("WaitForReady 正确处理了超时场景")
}
