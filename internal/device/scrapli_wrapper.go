package device

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/scrapli/scrapligo/driver/network"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/platform"
	"github.com/scrapli/scrapligo/transport"
	"github.com/scrapli/scrapligo/util"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// ScrapliWrapper scrapligo驱动封装（简化版）
// 并发安全由上层 PooledConnection 的设备级锁保护
type ScrapliWrapper struct {
	driver  *network.Driver
	device  *models.NetworkDevice
	closed  bool          // 标记连接是否已关闭
	closing chan struct{} // 用于通知关闭信号
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

// checkDeviceReachable 检查设备是否可达（TCP 连接测试）
func checkDeviceReachable(host string, port int, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return fmt.Errorf("设备不可达 [%s:%d]: %w", host, port, err)
	}
	conn.Close()
	return nil
}

// NewScrapliWrapper 创建scrapligo封装实例
func NewScrapliWrapper(device *models.NetworkDevice, username, password string, protocolType models.ProtocolType) (*ScrapliWrapper, error) {
	if device == nil {
		return nil, fmt.Errorf("设备信息不能为空")
	}

	// 获取平台名称
	platformName := PlatformName(device.Vendor)

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

	// 创建平台实例
	p, err := platform.NewPlatform(
		platformName,
		device.IPAddress,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("创建平台实例失败: %w", err)
	}

	// 获取网络驱动
	d, err := p.GetNetworkDriver()
	if err != nil {
		return nil, fmt.Errorf("获取网络驱动失败: %w", err)
	}

	return &ScrapliWrapper{
		driver:  d,
		device:  device,
		closing: make(chan struct{}),
	}, nil
}

// NewScrapliWrapperWithPort 创建带端口的scrapligo封装实例
func NewScrapliWrapperWithPort(device *models.NetworkDevice, username, password string, port int, protocolType models.ProtocolType) (*ScrapliWrapper, error) {
	if device == nil {
		return nil, fmt.Errorf("设备信息不能为空")
	}

	// 获取平台名称
	platformName := PlatformName(device.Vendor)

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

	// 创建平台实例
	p, err := platform.NewPlatform(
		platformName,
		device.IPAddress,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("创建平台实例失败: %w", err)
	}

	// 获取网络驱动
	d, err := p.GetNetworkDriver()
	if err != nil {
		return nil, fmt.Errorf("获取网络驱动失败: %w", err)
	}

	return &ScrapliWrapper{
		driver:  d,
		device:  device,
		closing: make(chan struct{}),
	}, nil
}

// Open 打开设备连接
func (w *ScrapliWrapper) Open() error {
	// 捕获 scrapligo 库可能的 panic
	defer func() {
		if r := recover(); r != nil {
			w.closed = true
		}
	}()

	if err := w.driver.Open(); err != nil {
		return fmt.Errorf("连接设备失败 [%s]: %w", w.device.IPAddress, err)
	}
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
				resultCh <- result{err: fmt.Errorf("连接 panic: %v", r)}
			}
		}()

		// 调用 Open()
		err := w.driver.Open()
		if err != nil {
			resultCh <- result{err: err}
			return
		}

		// Open 成功后，给一点时间让 scrapligo 内部的 goroutine（如 GetPrompt）完成初始化
		// 这可以避免 transport.conn 尚未就绪时的空指针问题
		time.Sleep(100 * time.Millisecond)

		resultCh <- result{err: nil}
	}()

	// 等待连接完成或超时
	select {
	case <-ctx.Done():
		// 超时或取消，尝试关闭可能正在进行的连接
		_ = w.driver.Close()
		w.closed = true
		return fmt.Errorf("连接设备超时 [%s]: %w", w.device.IPAddress, ctx.Err())
	case res := <-resultCh:
		if res.err != nil {
			w.closed = true
			return fmt.Errorf("连接设备失败 [%s]: %w", w.device.IPAddress, res.err)
		}
		return nil
	}
}

// Close 关闭设备连接
func (w *ScrapliWrapper) Close() error {
	if w.closed {
		return nil // 已经关闭，直接返回
	}

	w.closed = true

	// 先通知所有等待的 goroutine（不关闭 closing channel，以防双重 close）
	select {
	case <-w.closing:
	default:
		close(w.closing)
	}

	// 短暂等待，让 scrapligo 内部的 goroutine 有机会退出
	// 这些 goroutine 可能在 GetPrompt 等方法中被 spawn
	time.Sleep(50 * time.Millisecond)

	if w.driver != nil {
		return w.driver.Close()
	}
	return nil
}

// IsConnected 检查是否已连接
func (w *ScrapliWrapper) IsConnected() bool {
	// v1.3.3 中没有 Connected() 方法，通过检查driver是否为nil来判断
	return w.driver != nil && !w.closed
}

// SendCommand 发送单个命令
func (w *ScrapliWrapper) SendCommand(command string, stripPrompt bool) (*Response, error) {
	// 检查是否已关闭
	if w.closed {
		return nil, fmt.Errorf("连接已关闭")
	}

	if w.driver == nil {
		return nil, fmt.Errorf("设备未连接")
	}

	// 捕获 scrapligo 库可能的 panic
	defer func() {
		if r := recover(); r != nil {
			// panic 被捕获，连接可能已损坏
			w.closed = true
		}
	}()

	r, err := w.driver.SendCommand(command)
	if err != nil {
		// 如果是 EOF 或连接相关的错误，标记连接为已关闭
		errStr := err.Error()
		if containsEOF(errStr) || containsConnectionError(errStr) {
			w.closed = true
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
	// 检查是否已关闭
	if w.closed {
		return nil, fmt.Errorf("连接已关闭")
	}

	if w.driver == nil {
		return nil, fmt.Errorf("设备未连接")
	}

	responses := make([]*Response, 0, len(commands))

	for _, cmd := range commands {
		// 检查是否在执行过程中被关闭
		if w.closed {
			return responses, fmt.Errorf("连接已关闭")
		}

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
	// 检查是否已关闭
	if w.closed {
		return nil, fmt.Errorf("连接已关闭")
	}

	if w.driver == nil {
		return nil, fmt.Errorf("设备未连接")
	}

	// 捕获 scrapligo 库可能的 panic
	defer func() {
		if r := recover(); r != nil {
			w.closed = true
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
	// 检查是否已关闭
	if w.closed {
		return nil, fmt.Errorf("连接已关闭")
	}

	if w.driver == nil {
		return nil, fmt.Errorf("设备未连接")
	}

	responses := make([]*Response, 0, len(configs))

	for _, cfg := range configs {
		// 检查是否在执行过程中被关闭
		if w.closed {
			return responses, fmt.Errorf("连接已关闭")
		}

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
	// 检查是否已关闭
	if w.closed {
		return "", fmt.Errorf("连接已关闭")
	}

	if w.driver == nil {
		return "", fmt.Errorf("设备未连接")
	}

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
	// 检查连接是否有效
	if w.driver == nil || w.closed {
		return ""
	}

	// 检查是否已连接
	if !w.IsConnected() {
		return ""
	}

	// 使用 recover 捕获 scrapligo 库可能的内部 panic
	defer func() {
		if r := recover(); r != nil {
			// panic 被捕获，记录日志但不让程序崩溃
			// 连接状态不好时，GetPrompt 内部的 goroutine 可能触发 nil pointer
		}
	}()

	// 安全地获取 prompt，忽略错误（GetPrompt 可能在连接状态不好时失败）
	prompt, err := w.driver.GetPrompt()
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
