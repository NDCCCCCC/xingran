package device

import (
	"fmt"
	"time"

	"github.com/scrapli/scrapligo/driver/network"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/platform"
	"github.com/scrapli/scrapligo/transport"
	"github.com/scrapli/scrapligo/util"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// ScrapliWrapper scrapligo驱动封装
type ScrapliWrapper struct {
	driver *network.Driver
	device *models.NetworkDevice
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
		// SSH 使用 standard 传输 + 旧加密算法支持
		// Windows/Linux 兼容，支持老设备的传统加密算法
		opts = append(opts, options.WithTransportType(transport.StandardTransport))
		opts = append(opts, options.WithStandardTransportExtraCiphers([]string{
			"aes128-cbc",
			"aes192-cbc",
			"aes256-cbc",
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
		driver: d,
		device: device,
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
		// SSH 使用 standard 传输 + 旧加密算法支持
		opts = append(opts, options.WithTransportType(transport.StandardTransport))
		opts = append(opts, options.WithStandardTransportExtraCiphers([]string{
			"aes128-cbc",
			"aes192-cbc",
			"aes256-cbc",
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
		driver: d,
		device: device,
	}, nil
}

// Open 打开设备连接
func (w *ScrapliWrapper) Open() error {
	if err := w.driver.Open(); err != nil {
		return fmt.Errorf("连接设备失败 [%s]: %w", w.device.IPAddress, err)
	}
	return nil
}

// Close 关闭设备连接
func (w *ScrapliWrapper) Close() error {
	if w.driver != nil {
		return w.driver.Close()
	}
	return nil
}

// IsConnected 检查是否已连接
func (w *ScrapliWrapper) IsConnected() bool {
	// v1.3.3 中没有 Connected() 方法，通过检查driver是否为nil来判断
	return w.driver != nil
}

// SendCommand 发送单个命令
func (w *ScrapliWrapper) SendCommand(command string, stripPrompt bool) (*Response, error) {
	if !w.IsConnected() {
		return nil, fmt.Errorf("设备未连接")
	}

	r, err := w.driver.SendCommand(command)
	if err != nil {
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
	if !w.IsConnected() {
		return nil, fmt.Errorf("设备未连接")
	}

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
	if !w.IsConnected() {
		return nil, fmt.Errorf("设备未连接")
	}

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
	if !w.IsConnected() {
		return nil, fmt.Errorf("设备未连接")
	}

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
	if !w.IsConnected() {
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
	if w.driver != nil {
		prompt, _ := w.driver.GetPrompt()
		return prompt
	}
	return ""
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
