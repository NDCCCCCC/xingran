package device

import (
	"context"
	_ "embed"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/scrapli/scrapligo/driver/network"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/platform"
	"github.com/scrapli/scrapligo/transport"
	"github.com/scrapli/scrapligo/util"
)

// ConnectionState 连接状态
type ConnectionState int

const (
	StateInitializing ConnectionState = iota // 正在初始化
	StateReady                               // 已就绪，可以使用
	StateClosing                             // 正在关闭
	StateClosed                              // 已关闭
)

// scrapliInitTimeout Scrapli 客户端初始化就绪等待超时
const scrapliInitTimeout = 30 * time.Second

// String 返回状态的字符串表示
func (s ConnectionState) String() string {
	switch s {
	case StateInitializing:
		return "Initializing"
	case StateReady:
		return "Ready"
	case StateClosing:
		return "Closing"
	case StateClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}

// ScrapliWrapper scrapligo驱动封装（简化版）
// 并发安全由上层 PooledConnection 的设备级锁保护
// 使用 opMu (sync.RWMutex) 保护所有 driver 操作与 Close() 之间的竞态：
//   - 操作方法获取 RLock（允许多个操作并发）
//   - Close() 获取 Lock（等待所有操作完成后才关闭）
type ScrapliWrapper struct {
	driver      *network.Driver
	device      *models.NetworkDevice
	state       ConnectionState // 连接状态
	stateMu     sync.RWMutex    // 保护状态读写
	opMu        sync.RWMutex    // 保护 driver 操作与 Close() 的竞态
	getPromptMu sync.Mutex      // 串行化 GetPrompt 调用（防御 scrapligo Queue 竞态）
	closing     chan struct{}   // 用于通知关闭信号（用于 OpenContext 初始化循环等非 opMu 路径）
	initDone    chan struct{}   // 初始化完成信号
	closeOnce   sync.Once       // 确保 initDone 只关闭一次
}

// PlatformName 平台名称映射
// 注意：平台名称必须对应 scrapligo assets/platforms/ 目录中的 YAML 文件名（不含 .yaml 扩展名）
func PlatformName(vendor models.DeviceVendor) string {
	switch vendor {
	case models.VendorHuawei:
		return "huawei_vrp" // huawei_vrp.yaml
	case models.VendorH3C:
		return "hp_comware" // hp_comware.yaml
	case models.VendorRuijie:
		return "ruijie_rjos" // ruijie_rjos.yaml（注意：文件名是 rjos 而不是 rgos）
	case models.VendorMaipu:
		return "cisco_iosxe" // Maipu 使用通用的 cisco_iosxe 驱动
	default:
		return "cisco_iosxe" // 使用通用的 cisco_iosxe 作为默认
	}
}

//go:embed assets/ruijie_rgos_patched.yaml
var ruijiePatchedYAML []byte

// platformIdentifier 返回 scrapli platform 标识（string 内置平台名 或 []byte 自定义 yaml）。
//
// 锐捷用 patched yaml：scrapli 内置 ruijie_rjos.yaml 的 configuration prompt pattern 字符类
// [\+\w.\-@/:+] 不含空格，而锐捷接口名带空格（如 GigabitEthernet 4/18），导致接口视图 prompt
// `Ruijie(config-if-GigabitEthernet 4/18)#` 不匹配 → SendConfig 等不到 prompt 超时。
// patched yaml 在 pattern 加空格 + 扩容长度。scrapli platform 是 embedded assets 无法文件覆盖，
// 故用 platform.NewPlatform([]byte, ...) 注入（loadPlatformDefinitionFromBytes）。
func platformIdentifier(vendor models.DeviceVendor) interface{} {
	if vendor == models.VendorRuijie {
		return ruijiePatchedYAML
	}
	return PlatformName(vendor)
}

// checkDeviceReachable 检查设备是否可达（TCP 连接测试）
func checkDeviceReachable(host string, port int, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return fmt.Errorf("设备不可达 [%s:%d]: %w", host, port, err)
	}
	conn.Close()
	return nil
}

// newNetworkDriver 是 platform→network.Driver 的默认工厂（76-02 INFRA-02）。
// 生产路径行为与原先两处内联调用完全一致：NewPlatform 失败包装"创建平台实例失败"，
// GetNetworkDriver 失败包装"获取网络驱动失败"，错误字符串 byte 不变。
// 测试通过临时替换该 var 注入 FileTransport/自定义 transport（见
// driver_factory_76_02_test.go），替换必须 t.Cleanup 恢复且所在测试禁止 t.Parallel()。
var newNetworkDriver = func(platformName interface{}, host string, opts ...util.Option) (*network.Driver, error) {
	p, err := platform.NewPlatform(
		platformName,
		host,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("创建平台实例失败: %w", err)
	}

	d, err := p.GetNetworkDriver()
	if err != nil {
		return nil, fmt.Errorf("获取网络驱动失败: %w", err)
	}

	return d, nil
}

// NewScrapliWrapper 创建scrapligo封装实例
func NewScrapliWrapper(device *models.NetworkDevice, username, password string, protocolType models.ProtocolType) (*ScrapliWrapper, error) {
	if device == nil {
		return nil, fmt.Errorf("设备信息不能为空")
	}

	// 获取平台名称
	platformName := platformIdentifier(device.Vendor)

	// 创建基础选项
	opts := []util.Option{
		options.WithAuthUsername(username),
		options.WithAuthPassword(password),
		options.WithAuthNoStrictKey(),
	}

	// 根据协议类型选择传输类型
	switch protocolType {
	case models.ProtocolTypeTelnet:
		opts = append(opts, options.WithTransportType(transport.TelnetTransport))
	default:
		// SSH 使用 standard 传输 + 兼容新旧设备的加密算法
		// 支持 CBC 模式（老设备）和 CTR 模式（华为等新设备）
		opts = append(opts, options.WithTransportType(transport.StandardTransport))
		opts = append(opts, options.WithStandardTransportExtraCiphers([]string{
			// CTR 模式 - 华为等新设备需要
			"aes256-ctr",
			"aes192-ctr",
			"aes128-ctr",
			// CBC 模式 - 老设备支持
			"aes256-cbc",
			"aes192-cbc",
			"aes128-cbc",
			"3des-cbc",
			"blowfish-cbc",
		}))
	}

	// 经工厂构造平台实例与网络驱动（错误在工厂内部包装，文案不变）
	d, err := newNetworkDriver(platformName, device.IPAddress, opts...)
	if err != nil {
		return nil, err
	}

	return &ScrapliWrapper{
		driver:   d,
		device:   device,
		state:    StateInitializing,
		closing:  make(chan struct{}),
		initDone: make(chan struct{}),
	}, nil
}

// NewScrapliWrapperWithPort 创建带端口的scrapligo封装实例
func NewScrapliWrapperWithPort(device *models.NetworkDevice, username, password string, port int, protocolType models.ProtocolType) (*ScrapliWrapper, error) {
	if device == nil {
		return nil, fmt.Errorf("设备信息不能为空")
	}

	// 获取平台名称
	platformName := platformIdentifier(device.Vendor)

	// 创建基础选项
	opts := []util.Option{
		options.WithAuthUsername(username),
		options.WithAuthPassword(password),
		options.WithAuthNoStrictKey(),
		options.WithPort(port),
	}

	// 根据协议类型选择传输类型
	switch protocolType {
	case models.ProtocolTypeTelnet:
		opts = append(opts, options.WithTransportType(transport.TelnetTransport))
	default:
		// SSH 使用 standard 传输 + 兼容新旧设备的加密算法
		// 支持 CBC 模式（老设备）和 CTR 模式（华为等新设备）
		opts = append(opts, options.WithTransportType(transport.StandardTransport))
		opts = append(opts, options.WithStandardTransportExtraCiphers([]string{
			// CTR 模式 - 华为等新设备需要
			"aes256-ctr",
			"aes192-ctr",
			"aes128-ctr",
			// CBC 模式 - 老设备支持
			"aes256-cbc",
			"aes192-cbc",
			"aes128-cbc",
			"3des-cbc",
			"blowfish-cbc",
		}))
	}

	// 经工厂构造平台实例与网络驱动（错误在工厂内部包装，文案不变）
	d, err := newNetworkDriver(platformName, device.IPAddress, opts...)
	if err != nil {
		return nil, err
	}

	return &ScrapliWrapper{
		driver:   d,
		device:   device,
		state:    StateInitializing,
		closing:  make(chan struct{}),
		initDone: make(chan struct{}),
	}, nil
}

// setState 设置连接状态
func (w *ScrapliWrapper) setState(state ConnectionState) {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	w.state = state
}

// getState 获取连接状态
func (w *ScrapliWrapper) getState() ConnectionState {
	w.stateMu.RLock()
	defer w.stateMu.RUnlock()
	return w.state
}

// isClosing 检查是否正在关闭（非阻塞）
func (w *ScrapliWrapper) isClosing() bool {
	select {
	case <-w.closing:
		return true
	default:
		return false
	}
}

// acquireOp 获取操作锁（共享读锁），用于所有 driver 操作方法
// 确保 Close() 必须等待所有操作完成后才能关闭连接
func (w *ScrapliWrapper) acquireOp() error {
	state := w.getState()
	if state != StateReady {
		return fmt.Errorf("连接不可用 (当前状态: %s)", state)
	}

	w.opMu.RLock()

	// 获取锁后二次检查状态（防止 TOCTOU 竞态）
	if w.getState() != StateReady {
		w.opMu.RUnlock()
		return fmt.Errorf("连接不可用")
	}
	if w.driver == nil {
		w.opMu.RUnlock()
		return fmt.Errorf("设备未连接")
	}

	return nil
}

// releaseOp 释放操作锁
func (w *ScrapliWrapper) releaseOp() {
	w.opMu.RUnlock()
}

// GetPrompt 串行化调用 driver.GetPrompt，防御 scrapligo Queue.Dequeue 竞态。
// scrapligo v1.3.3 的内部 goroutine 在 Dequeue 时存在 TOCTOU 窗口，
// 并发 GetPrompt 会触发 panic（参见 .planning/debug/scrapligo-queue-panic-windows.md）。
// v1.4.0 已加双重检查缓解 panic，此处串行化作为稳健的防御纵深保留。
// 注意：v1.4.0 中 driver.GetPrompt 返回 (string, error)，而非 ([]byte, error)。
func (w *ScrapliWrapper) GetPrompt() (string, error) {
	w.getPromptMu.Lock()
	defer w.getPromptMu.Unlock()
	return w.driver.GetPrompt()
}

// Open 打开设备连接（已废弃，使用 OpenContext）
func (w *ScrapliWrapper) Open() error {
	// 捕获 scrapligo 库可能的 panic
	defer func() {
		if r := recover(); r != nil {
			w.setState(StateClosed)
		}
	}()

	if err := w.driver.Open(); err != nil {
		w.setState(StateClosed)
		return fmt.Errorf("连接设备失败 [%s]: %w", w.device.IPAddress, err)
	}

	// 等待初始化完成
	w.setState(StateReady)
	return nil
}

// OpenContext 打开设备连接（支持超时控制）
func (w *ScrapliWrapper) OpenContext(ctx context.Context) error {
	// 使用 channel 来传递结果
	type result struct {
		err error
	}
	resultCh := make(chan result, 1)

	// 在 goroutine 中执行连接
	go func() {
		// 捕获 panic
		defer func() {
			if r := recover(); r != nil {
				w.setState(StateClosed)
				w.closeOnce.Do(func() {
					close(w.initDone)
				})
				resultCh <- result{err: fmt.Errorf("连接 panic: %v", r)}
			}
		}()

		// 调用 Open()
		err := w.driver.Open()
		if err != nil {
			w.setState(StateClosed)
			w.closeOnce.Do(func() {
				close(w.initDone)
			})
			resultCh <- result{err: err}
			return
		}

		// Open 成功后，进入初始化阶段
		// 等待 scrapligo 内部的 goroutine（如 GetPrompt）完成初始化
		// 使用轮询方式验证连接真正就绪
		initCtx, cancelInit := context.WithTimeout(context.Background(), scrapliInitTimeout)
		defer cancelInit()

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		ready := false
		for !ready {
			select {
			case <-initCtx.Done():
				w.setState(StateClosed)
				w.closeOnce.Do(func() {
					close(w.initDone)
				})
				resultCh <- result{err: fmt.Errorf("连接初始化超时")}
				return
			case <-ticker.C:
				// 检查是否正在关闭（此时尚未进入 Ready 状态，不走 acquireOp 路径）
				if w.isClosing() {
					w.setState(StateClosed)
					w.closeOnce.Do(func() {
						close(w.initDone)
					})
					resultCh <- result{err: fmt.Errorf("连接正在关闭")}
					return
				}

				// 尝试调用 GetPrompt 来验证连接是否真正就绪
				func() {
					defer func() {
						if r := recover(); r != nil {
							_ = r // GetPrompt panic, connection not ready
						}
					}()
					_, err := w.GetPrompt()
					if err == nil {
						ready = true
					}
				}()
			}
		}

		// 连接已就绪，更新状态
		w.setState(StateReady)
		w.closeOnce.Do(func() {
			close(w.initDone)
		})
		resultCh <- result{err: nil}
	}()

	// 等待连接完成或超时
	select {
	case <-ctx.Done():
		// 超时或取消，尝试关闭可能正在进行的连接
		go func() {
			_ = w.driver.Close()
			w.setState(StateClosed)
			w.closeOnce.Do(func() {
				close(w.initDone)
			})
		}()
		return fmt.Errorf("连接设备超时 [%s]: %w", w.device.IPAddress, ctx.Err())
	case res := <-resultCh:
		if res.err != nil {
			w.setState(StateClosed)
			return fmt.Errorf("连接设备失败 [%s]: %w", w.device.IPAddress, res.err)
		}
		return nil
	}
}

// Close 关闭设备连接
// 使用 opMu.Lock() 确保所有正在执行的 driver 操作完成后才关闭
func (w *ScrapliWrapper) Close() error {
	// 获取当前状态
	currentState := w.getState()
	if currentState == StateClosed || currentState == StateClosing {
		return nil // 已经关闭或正在关闭，直接返回
	}

	// 设置状态为正在关闭（阻止新操作通过 acquireOp 的状态检查）
	w.setState(StateClosing)

	// 通知 closing channel（用于 OpenContext 初始化循环等非 opMu 路径）
	select {
	case <-w.closing:
	default:
		close(w.closing)
	}

	// 获取排他锁 — 阻塞直到所有 acquireOp() 持有的 RLock 释放
	// 这保证了：所有正在执行的 driver 操作完成后，Close 才会继续
	w.opMu.Lock()

	// 安全网：给 scrapligo 内部子 goroutine（如 GetPrompt 内部的 Write goroutine）
	// 少量时间退出。正常情况下这些 goroutine 在 opMu 排他锁获取前已完成，
	// 但 GetPrompt() 的子 goroutine 可能在父方法返回后仍在运行
	time.Sleep(100 * time.Millisecond)

	if w.driver != nil {
		// 捕获 Close 可能的 panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					_ = r // Close panic, suppress to prevent crash
				}
			}()
			_ = w.driver.Close()
		}()
	}

	// 设置状态为已关闭
	w.setState(StateClosed)

	w.opMu.Unlock()

	return nil
}

// IsConnected 检查是否已连接
func (w *ScrapliWrapper) IsConnected() bool {
	state := w.getState()
	return w.driver != nil && state == StateReady
}

// IsReady 检查连接是否真正可用（通过尝试调用 GetPrompt 验证）
// 使用 acquireOp 确保与 Close() 的正确同步
func (w *ScrapliWrapper) IsReady() bool {
	if err := w.acquireOp(); err != nil {
		return false
	}
	defer w.releaseOp()

	// 捕获 GetPrompt 可能的 panic
	ready := func() bool {
		defer func() {
			if r := recover(); r != nil {
				// panic 发生，说明 transport 未就绪
				w.setState(StateClosed)
			}
		}()
		_, err := w.GetPrompt()
		return err == nil
	}()

	return ready && w.getState() == StateReady
}

// WaitForReady 等待连接就绪（最多等待指定时间）
func (w *ScrapliWrapper) WaitForReady(timeout time.Duration) error {
	state := w.getState()
	if state == StateReady {
		return nil
	}

	if state == StateClosed || state == StateClosing {
		return fmt.Errorf("连接已关闭或正在关闭")
	}

	// 等待初始化完成
	select {
	case <-w.initDone:
		// 初始化完成，再次检查状态
		if w.getState() == StateReady {
			return nil
		}
		return fmt.Errorf("连接初始化失败")
	case <-time.After(timeout):
		return fmt.Errorf("等待连接就绪超时")
	}
}

// SendCommand 发送单个命令
func (w *ScrapliWrapper) SendCommand(command string, stripPrompt bool) (*Response, error) {
	if err := w.acquireOp(); err != nil {
		return nil, err
	}
	defer w.releaseOp()

	// 捕获 scrapligo 库可能的 panic
	defer func() {
		if r := recover(); r != nil {
			// panic 被捕获，连接可能已损坏
			w.setState(StateClosed)
		}
	}()

	r, err := w.driver.SendCommand(command)
	if err != nil {
		// 如果是 EOF 或连接相关的错误，标记连接为已关闭
		errStr := err.Error()
		if containsEOF(errStr) || containsConnectionError(errStr) {
			w.setState(StateClosed)
		}
		return nil, fmt.Errorf("发送命令失败: %w", err)
	}

	return &Response{
		Result:   r.Result,
		Started:  r.StartTime,
		Finished: r.EndTime,
		Failed:   r.Failed != nil,
	}, nil
}

// SendCommands 发送多个命令
func (w *ScrapliWrapper) SendCommands(commands []string, stripPrompt bool) ([]*Response, error) {
	if err := w.acquireOp(); err != nil {
		return nil, err
	}
	defer w.releaseOp()

	responses := make([]*Response, 0, len(commands))

	for _, cmd := range commands {
		r, err := w.driver.SendCommand(cmd)
		if err != nil {
			return responses, fmt.Errorf("发送命令 '%s' 失败: %w", cmd, err)
		}
		responses = append(responses, &Response{
			Result:   r.Result,
			Started:  r.StartTime,
			Finished: r.EndTime,
			Failed:   r.Failed != nil,
		})
	}

	return responses, nil
}

// SendConfig 发送配置命令
func (w *ScrapliWrapper) SendConfig(config string) (*Response, error) {
	if err := w.acquireOp(); err != nil {
		return nil, err
	}
	defer w.releaseOp()

	// 捕获 scrapligo 库可能的 panic
	defer func() {
		if r := recover(); r != nil {
			w.setState(StateClosed)
		}
	}()

	r, err := w.driver.SendConfig(config)
	if err != nil {
		return nil, fmt.Errorf("发送配置失败: %w", err)
	}

	return &Response{
		Result:   r.Result,
		Started:  r.StartTime,
		Finished: r.EndTime,
		Failed:   r.Failed != nil,
	}, nil
}

// SendConfigs 发送多个配置命令
func (w *ScrapliWrapper) SendConfigs(configs []string) ([]*Response, error) {
	if err := w.acquireOp(); err != nil {
		return nil, err
	}
	defer w.releaseOp()

	responses := make([]*Response, 0, len(configs))

	for _, cfg := range configs {
		r, err := w.driver.SendConfig(cfg)
		if err != nil {
			return responses, fmt.Errorf("发送配置 '%s' 失败: %w", cfg, err)
		}
		responses = append(responses, &Response{
			Result:   r.Result,
			Started:  r.StartTime,
			Finished: r.EndTime,
			Failed:   r.Failed != nil,
		})
	}

	return responses, nil
}

// GetConfig 获取设备配置
func (w *ScrapliWrapper) GetConfig() (string, error) {
	if err := w.acquireOp(); err != nil {
		return "", err
	}
	defer w.releaseOp()

	var command string
	switch w.device.Vendor {
	case models.VendorHuawei, models.VendorH3C:
		command = "display current-configuration"
	case models.VendorRuijie, models.VendorMaipu:
		command = "show running-config"
	default:
		command = "show running-config"
	}

	r, err := w.driver.SendCommand(command)
	if err != nil {
		return "", fmt.Errorf("获取配置失败: %w", err)
	}

	return r.Result, nil
}

// GetResponse 获取命令响应
func (w *ScrapliWrapper) GetResponse() string {
	if err := w.acquireOp(); err != nil {
		return ""
	}
	defer w.releaseOp()

	// 使用 recover 捕获 scrapligo 库可能的内部 panic
	defer func() {
		if r := recover(); r != nil {
			_ = r // panic 被捕获，记录日志但不让程序崩溃
		}
	}()

	// 安全地获取 prompt，忽略错误（GetPrompt 可能在连接状态不好时失败）
	prompt, err := w.GetPrompt()
	if err != nil {
		return ""
	}
	return prompt
}

// Response 命令响应
type Response struct {
	Result   string
	Started  time.Time
	Finished time.Time
	Failed   bool
}

// ElapsedTime 获取执行耗时（毫秒）
func (r *Response) ElapsedTime() int64 {
	if r.Finished.IsZero() || r.Started.IsZero() {
		return 0
	}
	return r.Finished.Sub(r.Started).Milliseconds()
}

// GetCommandForVendor 根据厂商获取对应命令
func GetCommandForVendor(vendor models.DeviceVendor, commandType string) string {
	commands := map[models.DeviceVendor]map[string]string{
		models.VendorHuawei: {
			"get_config":        "display current-configuration",
			"get_mac":           "display mac-address",
			"get_arp":           "display arp",
			"get_interface":     "display interface",
			"get_dot1x":         "display dot1x",
			"get_port_security": "display port-security",
			"get_version":       "display version",
		},
		models.VendorH3C: {
			"get_config":        "display current-configuration",
			"get_mac":           "display mac-address",
			"get_arp":           "display arp",
			"get_interface":     "display interface",
			"get_dot1x":         "display dot1x",
			"get_port_security": "display port-security",
			"get_version":       "display version",
		},
		models.VendorRuijie: {
			"get_config":        "show running-config",
			"get_mac":           "show mac-address-table",
			"get_arp":           "show arp",
			"get_interface":     "show interface",
			"get_dot1x":         "show dot1x",
			"get_port_security": "show port-security",
			"get_version":       "show version",
		},
		models.VendorMaipu: {
			"get_config":        "show running-config",
			"get_mac":           "show mac-address-table",
			"get_arp":           "show arp",
			"get_interface":     "show interface",
			"get_dot1x":         "show dot1x",
			"get_port_security": "show port-security",
			"get_version":       "show version",
		},
	}

	if vendorCmds, ok := commands[vendor]; ok {
		if cmd, ok := vendorCmds[commandType]; ok {
			return cmd
		}
	}
	// 默认返回通用命令
	return "show running-config"
}

// containsEOF 检查错误是否包含 EOF
func containsEOF(errStr string) bool {
	return strings.Contains(errStr, "EOF") ||
		strings.Contains(errStr, "unexpected EOF") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe")
}

// containsConnectionError 检查是否是连接相关错误
func containsConnectionError(errStr string) bool {
	return strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "closed")
}

// GetLLDPCommand 根据厂商获取LLDP命令
func GetLLDPCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display lldp neighbor brief",
		models.VendorH3C:    "display lldp neighbor brief",
		models.VendorRuijie: "show lldp neighbors",
		models.VendorMaipu:  "show lldp neighbors",
	}
	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "show lldp neighbors" // 默认
}
