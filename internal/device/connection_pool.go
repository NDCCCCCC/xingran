package device

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// PooledConnection 池化连接，带有引用计数和设备级锁
type PooledConnection struct {
	wrapper  *ScrapliWrapper
	refCount int32       // 引用计数（原子操作）
	lastUsed time.Time   // 最后使用时间
	deviceID string      // 设备ID
	mu       *sync.Mutex // 设备级互斥锁（指针类型，用于共享）
	pool     *DeviceConnectionPool
}

// Acquire 获取连接使用权（增加引用计数并获取设备锁）
// 返回 true 表示成功获取，false 表示连接不可用
func (pc *PooledConnection) Acquire() bool {
	// 先获取设备级锁
	pc.mu.Lock()

	// 检查连接是否仍然有效
	if pc.wrapper == nil || !pc.wrapper.IsConnected() {
		pc.mu.Unlock()
		return false
	}

	// 等待连接完全就绪（防止竞态条件）
	// 这确保只有在连接完全初始化后才允许使用
	if err := pc.wrapper.WaitForReady(5 * time.Second); err != nil {
		pc.mu.Unlock()
		return false
	}

	// 再次验证连接状态（可能在等待期间变化）
	if !pc.wrapper.IsConnected() || !pc.wrapper.IsReady() {
		pc.mu.Unlock()
		return false
	}

	// 增加引用计数
	newCount := atomic.AddInt32(&pc.refCount, 1)
	if newCount == 1 {
		// 连接被重新激活，更新最后使用时间
		pc.lastUsed = time.Now()
	}
	return true
}

// Release 释放连接使用权（减少引用计数并释放设备锁）
func (pc *PooledConnection) Release() {
	// 减少引用计数
	newCount := atomic.AddInt32(&pc.refCount, -1)
	if newCount < 0 {
		// 不应该发生，说明引用计数管理有问题
		panic(fmt.Sprintf("负引用计数: deviceID=%s, refCount=%d", pc.deviceID, newCount))
	}

	// 更新最后使用时间
	pc.lastUsed = time.Now()

	// 释放设备级锁
	pc.mu.Unlock()
}

// Execute 在连接上执行命令（自动管理锁和引用计数）
func (pc *PooledConnection) Execute(ctx context.Context, fn func(*ScrapliWrapper) error) (err error) {
	if !pc.Acquire() {
		return fmt.Errorf("连接不可用: deviceID=%s", pc.deviceID)
	}
	defer pc.Release()

	// 检查上下文是否已取消
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// 捕获可能的 panic（特别是 scrapligo 库的空指针问题）
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("执行命令时发生 panic: %v, deviceID=%s", r, pc.deviceID)
			// 标记连接为已关闭，防止后续使用
			if pc.wrapper != nil {
				pc.wrapper.setState(StateClosed)
			}
		}
	}()

	// 执行命令
	return fn(pc.wrapper)
}

// IsIdle 检查连接是否空闲（无引用）
func (pc *PooledConnection) IsIdle() bool {
	return atomic.LoadInt32(&pc.refCount) <= 0
}

// Close 关闭连接
func (pc *PooledConnection) Close() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.wrapper != nil {
		return pc.wrapper.Close()
	}
	return nil
}

// GetWrapper 获取底层 wrapper
func (pc *PooledConnection) GetWrapper() *ScrapliWrapper {
	return pc.wrapper
}

// DeviceConnectionPool 设备连接池
type DeviceConnectionPool struct {
	connections    map[string]*PooledConnection // deviceID -> PooledConnection
	deviceLocks    map[string]*sync.Mutex       // deviceID -> 设备锁
	poolLock       sync.RWMutex                 // 保护 connections 和 deviceLocks
	db             *gorm.DB                     // 数据库（用于查询设备和凭证）
	sm4Cipher      *security.SM4Cipher          // SM4 加密器
	maxIdle        time.Duration                // 空闲连接超时时间
	maxConnections int                          // 最大连接数
	cleanupTicker  *time.Ticker                 // 清理定时器
	done           chan struct{}                // 停止信号
	enabled        bool                         // 是否启用连接池（用于平滑迁移）
}

// PoolConfig 连接池配置
type PoolConfig struct {
	MaxIdle        time.Duration // 空闲连接超时时间，默认 5 分钟
	MaxConnections int           // 最大连接数，默认 50
}

// DefaultPoolConfig 默认连接池配置
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxIdle:        5 * time.Minute,
		MaxConnections: 50,
	}
}

// NewDeviceConnectionPool 创建设备连接池
func NewDeviceConnectionPool(db *gorm.DB, sm4Cipher *security.SM4Cipher, config *PoolConfig) *DeviceConnectionPool {
	if config == nil {
		config = DefaultPoolConfig()
	}

	pool := &DeviceConnectionPool{
		connections:    make(map[string]*PooledConnection),
		deviceLocks:    make(map[string]*sync.Mutex),
		db:             db,
		sm4Cipher:      sm4Cipher,
		maxIdle:        config.MaxIdle,
		maxConnections: config.MaxConnections,
		done:           make(chan struct{}),
		enabled:        true, // 默认启用
	}

	// 启动清理协程
	pool.startCleanup()

	applogger.Infof("[连接池] 已创建连接池: 最大空闲=%v, 最大连接数=%d", config.MaxIdle, config.MaxConnections)

	return pool
}

// SetEnabled 设置连接池是否启用（用于平滑迁移）
func (p *DeviceConnectionPool) SetEnabled(enabled bool) {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()
	p.enabled = enabled
}

// IsEnabled 检查连接池是否启用
func (p *DeviceConnectionPool) IsEnabled() bool {
	p.poolLock.RLock()
	defer p.poolLock.RUnlock()
	return p.enabled
}

// GetConnection 获取设备连接（线程安全）
func (p *DeviceConnectionPool) GetConnection(ctx context.Context, deviceID string) (*PooledConnection, error) {
	// 如果连接池未启用，返回错误（让调用方使用旧模式）
	if !p.IsEnabled() {
		return nil, fmt.Errorf("连接池未启用")
	}

	// 获取设备级锁
	deviceLock := p.getDeviceLock(deviceID)
	deviceLock.Lock()

	// 检查池中是否已有连接
	p.poolLock.RLock()
	pc, exists := p.connections[deviceID]
	p.poolLock.RUnlock()

	if exists {
		// 连接存在，检查是否有效
		// 使用 IsReady() 而不是 IsConnected() 以确保 transport 完全初始化
		if pc.wrapper != nil && pc.wrapper.IsReady() {
			// CRITICAL: 必须在返回前释放设备锁，否则会导致死锁
			// 因为后续的 Acquire() 会尝试获取同一个锁（pc.mu 就是 deviceLock）
			deviceLock.Unlock()
			return pc, nil
		}

		// 连接失效，从池中移除
		// 释放 deviceLock 后再调用 removeConnection，避免持有 deviceLock 时获取 poolLock
		deviceLock.Unlock()
		if err := p.removeConnection(deviceID); err != nil {
			applogger.Warnf("[连接池] 移除失效连接失败: %s, error: %v", deviceID, err)
		}
		// 重新获取 deviceLock 用于后续操作
		deviceLock.Lock()
	}

	// 检查连接数限制
	p.poolLock.RLock()
	currentCount := len(p.connections)
	p.poolLock.RUnlock()

	if currentCount >= p.maxConnections {
		deviceLock.Unlock()
		return nil, fmt.Errorf("连接池已满: 当前=%d, 最大=%d", currentCount, p.maxConnections)
	}

	// 创建新连接
	wrapper, err := p.createConnection(ctx, deviceID)
	if err != nil {
		deviceLock.Unlock()
		return nil, fmt.Errorf("创建连接失败: %w", err)
	}

	// CRITICAL: 等待连接完全初始化后再返回
	// 这是防止 panic 的关键：确保连接完全就绪后才允许使用
	// 使用一个较长的超时时间（30秒），因为 OpenContext 内部已经有超时控制
	if err := wrapper.WaitForReady(30 * time.Second); err != nil {
		// 初始化失败，关闭连接
		_ = wrapper.Close()
		deviceLock.Unlock()
		return nil, fmt.Errorf("等待连接就绪失败: %w", err)
	}

	// 最终验证：确保连接真正可用
	if !wrapper.IsReady() {
		_ = wrapper.Close()
		deviceLock.Unlock()
		return nil, fmt.Errorf("连接未完全初始化")
	}

	// 创建池化连接
	pc = &PooledConnection{
		wrapper:  wrapper,
		refCount: 0,
		lastUsed: time.Now(),
		deviceID: deviceID,
		mu:       deviceLock,
		pool:     p,
	}

	// 添加到池中
	p.poolLock.Lock()
	p.connections[deviceID] = pc
	p.poolLock.Unlock()

	// CRITICAL: 必须在返回前释放设备锁！
	// Acquire() 方法会负责重新获取这个锁
	deviceLock.Unlock()

	return pc, nil
}

// getDeviceLock 获取设备级锁
func (p *DeviceConnectionPool) getDeviceLock(deviceID string) *sync.Mutex {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	if lock, exists := p.deviceLocks[deviceID]; exists {
		return lock
	}

	lock := &sync.Mutex{}
	p.deviceLocks[deviceID] = lock
	return lock
}

// createConnection 创建新连接
func (p *DeviceConnectionPool) createConnection(ctx context.Context, deviceID string) (*ScrapliWrapper, error) {
	// 查询设备信息
	var device models.NetworkDevice
	if err := p.db.Where("id = ?", deviceID).First(&device).Error; err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	// 查询凭证信息
	var credential models.AuthCredential
	if device.CredentialID != nil && *device.CredentialID != "" {
		if err := p.db.Where("id = ?", *device.CredentialID).First(&credential).Error; err != nil {
			return nil, fmt.Errorf("查询凭证失败: %w", err)
		}
		applogger.Infof("[连接池] 设备 %s 使用关联凭证: %s (id=%s, user=%s, protocol=%s)", device.DeviceName, credential.CredentialName, credential.ID, credential.Username, credential.ProtocolType)
	} else {
		if err := p.db.Where("is_default = ?", true).First(&credential).Error; err != nil {
			return nil, fmt.Errorf("未找到默认凭证: %w", err)
		}
		applogger.Infof("[连接池] 设备 %s 未关联凭证，使用默认凭证: %s (id=%s, user=%s, protocol=%s)", device.DeviceName, credential.CredentialName, credential.ID, credential.Username, credential.ProtocolType)
	}

	// 解密密码
	password := credential.Password
	if p.sm4Cipher == nil {
		return nil, fmt.Errorf("SM4 加密器未初始化，无法解密密码")
	}
	if password == "" {
		return nil, fmt.Errorf("凭证密码为空")
	}

	decrypted, err := p.sm4Cipher.Decrypt(password)
	if err != nil {
		return nil, fmt.Errorf("密码解密失败: %w", err)
	}
	password = decrypted
	applogger.Infof("[连接池] 设备 %s 凭证解密成功: 密文长度=%d, 明文长度=%d", device.DeviceName, len(credential.Password), len(password))

	// 创建连接
	var wrapper *ScrapliWrapper
	if device.Port > 0 && device.Port != 22 {
		wrapper, err = NewScrapliWrapperWithPort(&device, credential.Username, password, device.Port, credential.ProtocolType)
	} else {
		wrapper, err = NewScrapliWrapper(&device, credential.Username, password, credential.ProtocolType)
	}

	if err != nil {
		return nil, fmt.Errorf("创建 ScrapliWrapper 失败: %w", err)
	}

	// 先检查设备是否可达（避免 scrapligo 内部 panic）
	port := device.Port
	if port <= 0 {
		if credential.ProtocolType == models.ProtocolTypeTelnet {
			port = 23
		} else {
			port = 22
		}
	}

	checkCtx, cancelCheck := context.WithTimeout(ctx, 5*time.Second)
	defer cancelCheck()

	// 在单独的 goroutine 中执行检查
	reachableCh := make(chan error, 1)
	go func() {
		reachableCh <- checkDeviceReachable(device.IPAddress, port, 5*time.Second)
	}()

	select {
	case <-checkCtx.Done():
		return nil, fmt.Errorf("检查设备可达性超时 [%s:%d]", device.IPAddress, port)
	case err := <-reachableCh:
		if err != nil {
			return nil, fmt.Errorf("设备不可达，请检查网络和设备状态: %w", err)
		}
	}

	// 设置连接超时（增加到60秒，给设备更多响应时间）
	connectCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if err := wrapper.OpenContext(connectCtx); err != nil {
		return nil, fmt.Errorf("连接设备失败 [%s]: %w", device.IPAddress, err)
	}

	applogger.Infof("[连接池] 设备连接成功: %s (%s:%d)", device.DeviceName, device.IPAddress, port)

	return wrapper, nil
}

// removeConnection 移除连接（线程安全，内部获取 poolLock）
// 注意：调用此方法前必须释放设备锁（deviceLock），避免死锁
func (p *DeviceConnectionPool) removeConnection(deviceID string) error {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	if pc, exists := p.connections[deviceID]; exists {
		// 使用带超时的等待，避免永久阻塞
		// 遵循 Go 最佳实践：使用条件变量或 channel 而不是忙等待
		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for !pc.IsIdle() {
			select {
			case <-ticker.C:
				continue
			case <-timeout:
				return fmt.Errorf("等待连接空闲超时: %s", deviceID)
			}
		}

		// 关闭连接
		if pc.wrapper != nil {
			pc.wrapper.Close()
		}

		delete(p.connections, deviceID)
	}
	return nil
}

// startCleanup 启动清理协程
func (p *DeviceConnectionPool) startCleanup() {
	p.cleanupTicker = time.NewTicker(1 * time.Minute)

	go func() {
		for {
			select {
			case <-p.cleanupTicker.C:
				p.cleanupIdleConnections()
			case <-p.done:
				return
			}
		}
	}()
}

// cleanupIdleConnections 清理空闲连接
func (p *DeviceConnectionPool) cleanupIdleConnections() {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	now := time.Now()
	cleaned := 0

	for deviceID, pc := range p.connections {
		// 检查是否空闲超过阈值
		if now.Sub(pc.lastUsed) > p.maxIdle {
			// 检查引用计数
			if pc.IsIdle() {
				// 关闭连接
				if pc.wrapper != nil {
					pc.wrapper.Close()
				}
				delete(p.connections, deviceID)
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		applogger.Infof("[连接池] 清理了 %d 个空闲连接, 当前连接数=%d",
			cleaned, len(p.connections))
	}
}

// GetStats 获取连接池统计信息
func (p *DeviceConnectionPool) GetStats() map[string]interface{} {
	p.poolLock.RLock()
	defer p.poolLock.RUnlock()

	activeCount := 0
	idleCount := 0

	for _, pc := range p.connections {
		if pc.IsIdle() {
			idleCount++
		} else {
			activeCount++
		}
	}

	return map[string]interface{}{
		"total_connections":  len(p.connections),
		"active_connections": activeCount,
		"idle_connections":   idleCount,
		"max_connections":    p.maxConnections,
		"enabled":            p.enabled,
	}
}

// GetDevice 获取设备信息
func (p *DeviceConnectionPool) GetDevice(deviceID string) (*models.NetworkDevice, error) {
	var device models.NetworkDevice
	if err := p.db.Where("id = ?", deviceID).First(&device).Error; err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}
	return &device, nil
}

// Close 关闭连接池
func (p *DeviceConnectionPool) Close() error {
	applogger.Infof("[连接池] 正在关闭连接池...")

	// 停止清理协程
	close(p.done)
	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
	}

	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	// 关闭所有连接，使用超时控制避免永久阻塞
	closeTimeout := time.After(10 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for deviceID, pc := range p.connections {
		// 等待引用计数归零（带超时）
		idleCheck := time.After(2 * time.Second)
		for !pc.IsIdle() {
			select {
			case <-ticker.C:
				continue
			case <-idleCheck:
				applogger.Warnf("[连接池] 连接 %s 关闭超时，强制关闭", deviceID)
				goto closeConnection
			case <-closeTimeout:
				applogger.Warnf("[连接池] 整体关闭超时，退出清理")
				goto finalize
			}
		}

	closeConnection:
		if pc.wrapper != nil {
			pc.wrapper.Close()
		}
	}

finalize:
	p.connections = make(map[string]*PooledConnection)
	p.deviceLocks = make(map[string]*sync.Mutex)

	applogger.Infof("[连接池] 连接池已关闭")

	return nil
}
