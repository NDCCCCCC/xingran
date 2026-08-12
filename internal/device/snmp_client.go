package device

import (
	"context"
	"fmt"
	"net"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// SNMPClient SNMP客户端
type SNMPClient struct {
	client  *gosnmp.GoSNMP
	mu      sync.RWMutex  // 保护 gosnmp.Conn 的并发访问
	ready   bool          // 连接是否就绪
	readyMu sync.RWMutex  // 保护 ready 字段，支持并发读取
}

// SNMPClientConfig SNMP客户端配置
type SNMPClientConfig struct {
	Target    string
	Port      uint16
	Community string
	Version   SNMPVersion
	Timeout   time.Duration
	Retries   int
}

// SNMPVersion SNMP版本
type SNMPVersion gosnmp.SnmpVersion

const (
	// SNMPVersion1 SNMP v1
	SNMPVersion1 SNMPVersion = SNMPVersion(gosnmp.Version1)
	// SNMPVersion2c SNMP v2c
	SNMPVersion2c SNMPVersion = SNMPVersion(gosnmp.Version2c)
	// SNMPVersion3 SNMP v3
	SNMPVersion3 SNMPVersion = SNMPVersion(gosnmp.Version3)
)

// NewSNMPClient 创建SNMP客户端
func NewSNMPClient(config *SNMPClientConfig) *SNMPClient {
	if config == nil {
		config = &SNMPClientConfig{
			Port:      161,
			Community: "public",
			Version:   SNMPVersion2c,
			Timeout:   5 * time.Second,
			Retries:   3,
		}
	}

	return &SNMPClient{
		client: &gosnmp.GoSNMP{
			Target:    config.Target,
			Port:      config.Port,
			Community: config.Community,
			Version:   gosnmp.SnmpVersion(config.Version),
			Timeout:   config.Timeout,
			Retries:   config.Retries,
		},
	}
}

// connectLocked 内部连接方法，假设调用方已持有锁
func (c *SNMPClient) connectLocked() error {
	err := c.client.Connect()
	if err != nil {
		return fmt.Errorf("SNMP连接失败: %w", err)
	}
	return nil
}

// Connect 连接SNMP设备
func (c *SNMPClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.connectLocked()
	if err != nil {
		return err
	}

	// 标记为就绪
	c.setReady(true)
	return nil
}

// setReady 设置就绪状态（线程安全）
func (c *SNMPClient) setReady(ready bool) {
	c.readyMu.Lock()
	defer c.readyMu.Unlock()
	c.ready = ready
}

// isReady 检查是否就绪（线程安全）
func (c *SNMPClient) isReady() bool {
	c.readyMu.RLock()
	defer c.readyMu.RUnlock()
	return c.ready
}

// closeLocked 内部关闭方法，假设调用方已持有锁
func (c *SNMPClient) closeLocked() error {
	if c.client != nil {
		return c.client.Conn.Close()
	}
	return nil
}

// Close 关闭连接
func (c *SNMPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.closeLocked()
	c.setReady(false) // 标记为未就绪
	return err
}

// WaitForReady 等待连接就绪（带超时）
// timeout: 超时时间，建议 5-10 秒
// 返回 nil 表示连接已就绪，返回 error 表示超时或连接失败
func (c *SNMPClient) WaitForReady(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待连接就绪超时")

		case <-ticker.C:
			if c.isReady() {
				return nil
			}
		}
	}
}


// Get � 获取单个OID值
func (c *SNMPClient) Get(oid string) (result interface{}, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	defer func() {
		if r := recover(); r != nil {
			ip := "unknown"
			if c.client != nil {
				ip = c.client.Target
			}
			logger.Errorf("[SNMP] Panic: ip=%s, method=Get, error=%v, stack=%s",
				ip, r, string(debug.Stack()))
			err = fmt.Errorf("SNMP Get操作panic: %v", r)
			c.closeLocked()
		}
	}()

	if err := c.connectLocked(); err != nil {
		return nil, err
	}
	defer c.closeLocked()

	resp, err := c.client.Get([]string{oid})
	if err != nil {
		return nil, fmt.Errorf("SNMP GET失败: %w", err)
	}

	if len(resp.Variables) == 0 {
		return nil, fmt.Errorf("没有返回数据")
	}

	return parseSNMPValue(resp.Variables[0]), nil
}

// GetNext 获取下一个OID值（用于遍历）
func (c *SNMPClient) GetNext(oid string) (resultOid string, resultValue interface{}, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	defer func() {
		if r := recover(); r != nil {
			ip := "unknown"
			if c.client != nil {
				ip = c.client.Target
			}
			logger.Errorf("[SNMP] Panic: ip=%s, method=GetNext, error=%v, stack=%s",
				ip, r, string(debug.Stack()))
			err = fmt.Errorf("SNMP GetNext操作panic: %v", r)
			c.closeLocked()
		}
	}()

	if err := c.connectLocked(); err != nil {
		return "", nil, err
	}
	defer c.closeLocked()

	resp, err := c.client.GetNext([]string{oid})
	if err != nil {
		return "", nil, fmt.Errorf("SNMP GETNEXT失败: %w", err)
	}

	if len(resp.Variables) == 0 {
		return "", nil, fmt.Errorf("没有返回数据")
	}

	variable := resp.Variables[0]
	return variable.Name, parseSNMPValue(variable), nil
}

// Walk 遍历OID树
func (c *SNMPClient) Walk(oid string, callback func(oid string, value interface{}) bool) (err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	defer func() {
		if r := recover(); r != nil {
			ip := "unknown"
			if c.client != nil {
				ip = c.client.Target
			}
			logger.Errorf("[SNMP] Panic: ip=%s, method=Walk, error=%v, stack=%s",
				ip, r, string(debug.Stack()))
			err = fmt.Errorf("SNMP Walk操作panic: %v", r)
			c.closeLocked()
		}
	}()

	if err := c.connectLocked(); err != nil {
		return err
	}
	defer c.closeLocked()

	return c.client.Walk(oid, func(variable gosnmp.SnmpPDU) error {
		if !callback(variable.Name, parseSNMPValue(variable)) {
			return fmt.Errorf("walk stopped")
		}
		return nil
	})
}

// GetBulk 批量获取（SNMP v2c/v3）
func (c *SNMPClient) GetBulk(oid string, maxRepetitions uint8) (result []gosnmp.SnmpPDU, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	defer func() {
		if r := recover(); r != nil {
			ip := "unknown"
			if c.client != nil {
				ip = c.client.Target
			}
			logger.Errorf("[SNMP] Panic: ip=%s, method=GetBulk, error=%v, stack=%s",
				ip, r, string(debug.Stack()))
			err = fmt.Errorf("SNMP GetBulk操作panic: %v", r)
			c.closeLocked()
		}
	}()

	if c.client.Version < gosnmp.Version2c {
		return nil, fmt.Errorf("GETBULK仅支持SNMP v2c或更高版本")
	}

	if err := c.connectLocked(); err != nil {
		return nil, err
	}
	defer c.closeLocked()

	resp, err := c.client.GetBulk([]string{oid}, 0, uint32(maxRepetitions))
	if err != nil {
		return nil, fmt.Errorf("SNMP GETBULK失败: %w", err)
	}

	return resp.Variables, nil
}

// parseSNMPValue 解析SNMP值
func parseSNMPValue(variable gosnmp.SnmpPDU) interface{} {
	switch variable.Type {
	case gosnmp.Integer:
		if variable.Value != nil {
			return gosnmp.ToBigInt(variable.Value).Int64()
		}
	case gosnmp.OctetString:
		if variable.Value != nil {
			return string(variable.Value.([]byte))
		}
	case gosnmp.ObjectIdentifier:
		if variable.Value != nil {
			return variable.Value.(string)
		}
	case gosnmp.Counter32, gosnmp.Counter64, gosnmp.Gauge32, gosnmp.TimeTicks:
		if variable.Value != nil {
			return gosnmp.ToBigInt(variable.Value).Uint64()
		}
	case gosnmp.IPAddress:
		// IPAddress类型已在新版gosnmp中移除，作为OctetString处理
		if variable.Value != nil {
			return net.IP(variable.Value.([]byte)).String()
		}
	}
	return variable.Value
}

// SNMP OID 常量
const (
	// 系统信息 OIDs
	oidSysDescr    = "1.3.6.1.2.1.1.1.0" // 系统描述
	oidSysObjectID = "1.3.6.1.2.1.1.2.0" // 系统对象ID
	oidSysUpTime   = "1.3.6.1.2.1.1.3.0" // 系统运行时间
	oidSysContact  = "1.3.6.1.2.1.1.4.0" // 系统联系人
	oidSysName     = "1.3.6.1.2.1.1.5.0" // 系统名称
	oidSysLocation = "1.3.6.1.2.1.1.6.0" // 系统位置

)

// SystemInfo 系统信息
type SystemInfo struct {
	Description string
	ObjectID    string
	UpTime      time.Duration
	Contact     string
	Name        string
	Location    string
}

// GetSystemInfo 获取系统信息
func (c *SNMPClient) GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}

	if descr, err := c.Get(oidSysDescr); err == nil && descr != nil {
		info.Description = descr.(string)
	}
	if objectID, err := c.Get(oidSysObjectID); err == nil && objectID != nil {
		info.ObjectID = objectID.(string)
	}
	if name, err := c.Get(oidSysName); err == nil && name != nil {
		info.Name = name.(string)
	}
	if location, err := c.Get(oidSysLocation); err == nil && location != nil {
		info.Location = location.(string)
	}

	return info, nil
}

// DetectVendor 通过系统描述检测设备厂商
func DetectVendor(sysDescr string) models.DeviceVendor {
	descr := string(sysDescr)

	// 厂商特征字符串检测
	switch {
	case containsIgnoreCase(descr, "Huawei"), containsIgnoreCase(descr, "HUAWEI"):
		return models.VendorHuawei
	case containsIgnoreCase(descr, "H3C"), containsIgnoreCase(descr, "HP"), containsIgnoreCase(descr, "3Com"):
		return models.VendorH3C
	case containsIgnoreCase(descr, "Ruijie"), containsIgnoreCase(descr, "RUIJIE"):
		return models.VendorRuijie
	case containsIgnoreCase(descr, "Maipu"), containsIgnoreCase(descr, "MAIPU"):
		return models.VendorMaipu
	case containsIgnoreCase(descr, "Cisco"):
		return "" // 不支持的厂商
	default:
		return "" // 未知厂商
	}
}

// DetectDeviceType 检测设备类型
func DetectDeviceType(sysDescr string) models.DeviceType {
	descr := string(sysDescr)

	switch {
	case containsIgnoreCase(descr, "Router"), containsIgnoreCase(descr, "MSR"):
		return models.DeviceTypeRouter
	case containsIgnoreCase(descr, "Switch"), containsIgnoreCase(descr, "S5"), containsIgnoreCase(descr, "S6"):
		return models.DeviceTypeSwitch
	case containsIgnoreCase(descr, "Firewall"), containsIgnoreCase(descr, "USG"):
		return models.DeviceTypeFirewall
	case containsIgnoreCase(descr, "AP"), containsIgnoreCase(descr, "Fat AP"), containsIgnoreCase(descr, "Fit AP"):
		return models.DeviceTypeAP
	default:
		return models.DeviceTypeSwitch // 默认为交换机
	}
}

// ExtractModelFromSysDescr 从系统描述提取设备型号
// 支持华为、H3C、锐捷、迈普等厂商的型号提取
func ExtractModelFromSysDescr(sysDescr string, vendor models.DeviceVendor) string {
	descr := string(sysDescr)
	if descr == "" {
		return ""
	}

	// 根据厂商使用不同的提取策略
	switch vendor {
	case models.VendorHuawei:
		return extractHuaweiModel(descr)
	case models.VendorH3C:
		return extractH3CModel(descr)
	case models.VendorRuijie:
		return extractRuijieModel(descr)
	case models.VendorMaipu:
		return extractMaipuModel(descr)
	default:
		// 未知厂商，尝试通用提取
		return extractGenericModel(descr)
	}
}

// extractHuaweiModel 提取华为设备型号
// 华为设备型号通常如: S5700-28P-LI-AC, AR2220, USG6000等
func extractHuaweiModel(descr string) string {
	// 华为常见型号模式
	patterns := []string{
		"S[0-9]{4}[A-Z0-9\\-]*", // S5700, S6720, S12700等
		"AR[0-9]{3,4}",          // AR2220, AR1220等路由器
		"USG[0-9]{4}",           // USG6000等防火墙
		"AirEngine[0-9]{4,6}",   // AirEngine系列AP
		"AP[0-9]{4}",            // AP4030等
		"NetEngine[0-9]{4}",     // NetEngine系列路由器
	}

	for _, pattern := range patterns {
		if model := extractByPattern(descr, pattern); model != "" {
			return model
		}
	}

	return ""
}

// extractH3CModel 提取H3C设备型号
// H3C设备型号通常如: S5120-28P-SI, S12508, MSR3640等
func extractH3CModel(descr string) string {
	patterns := []string{
		"S[0-9]{4,5}[A-Z0-9\\-]*", // S5120, S12508, S10500等
		"MSR[0-9]{4}",             // MSR3640, MSR3020等路由器
		"F[0-9]{4,5}",             // F1000, F5000等防火墙
		"WA[0-9]{4,5}",            // WA系列AP
		"H3C\\s+[A-Z0-9\\-]+",     // H3C空格后跟型号
	}

	for _, pattern := range patterns {
		if model := extractByPattern(descr, pattern); model != "" {
			return model
		}
	}

	return ""
}

// extractRuijieModel 提取锐捷设备型号
// 锐捷设备型号通常如: RG-S5750-28GT-P-S, RSR20-04, RG-AP640等
func extractRuijieModel(descr string) string {
	patterns := []string{
		"RG-S[0-9]{4,5}[A-Z0-9\\-]*", // RG-S5750, RG-S6220等交换机
		"RSR[0-9]{2,4}[A-Z0-9\\-]*",  // RSR20, RSR50等路由器
		"RG-WALL[0-9]{4}",            // RG-WALL防火墙
		"RG-AP[0-9]{3,4}",            // RG-AP640等AP
		"RG-EG[0-9]{4}",              // RG-EG网关
	}

	for _, pattern := range patterns {
		if model := extractByPattern(descr, pattern); model != "" {
			return model
		}
	}

	return ""
}

// extractMaipuModel 提取迈普设备型号
// 迈普设备型号通常如: Secure, MyPower, SM等系列
func extractMaipuModel(descr string) string {
	patterns := []string{
		"Secure[0-9]{4}",       // Secure系列防火墙
		"MyPower\\s*[A-Z0-9]+", // MyPower系列
		"SM[0-9]{4}",           // SM系列交换机
		"MP[0-9]{4}",           // MP系列路由器
	}

	for _, pattern := range patterns {
		if model := extractByPattern(descr, pattern); model != "" {
			return model
		}
	}

	return ""
}

// extractGenericModel 通用型号提取（适用于未知厂商）
// 尝试提取类似设备型号的字符串（大写字母开头，包含数字）
func extractGenericModel(descr string) string {
	// 常见通用模式
	patterns := []string{
		"[A-Z]{1,4}[0-9]{3,6}[A-Z0-9\\-]*", // 大写字母+数字组合
		"[A-Z]{2,}-[A-Z0-9\\-]+",           // XX-XXXX格式
	}

	for _, pattern := range patterns {
		if model := extractByPattern(descr, pattern); model != "" {
			return model
		}
	}

	return ""
}

// extractByPattern 使用正则表达式提取匹配的型号
// 简化实现，不使用正则表达式库，避免额外依赖
func extractByPattern(descr, pattern string) string {
	// 简化版的模式匹配
	// 这里只实现基本的模式匹配，如果需要更复杂的匹配，可以考虑使用regexp包

	// 查找包含数字和字母的典型型号格式
	// 例如: S5700, AR2220, RG-S5750等

	// 转换为大写进行比较
	descrUpper := toUpper(descr)

	// 华为 S 系列交换机
	if contains(pattern, "S[0-9]{4}") || contains(pattern, "S[0-9]{5}") {
		// 查找 S+4-5位数字的模式
		for i := 0; i < len(descrUpper)-5; i++ {
			if descrUpper[i] == 'S' && isDigit(descrUpper[i+1]) && isDigit(descrUpper[i+2]) && isDigit(descrUpper[i+3]) && isDigit(descrUpper[i+4]) {
				// 找到 Sxxxx，提取完整型号
				end := i + 5
				// 继续提取后续的字母数字和横线
				for end < len(descrUpper) && (isDigit(descrUpper[end]) || isUpper(descrUpper[end]) || descrUpper[end] == '-') {
					end++
				}
				model := descr[i:end]
				// 限制型号长度，避免过长
				if len(model) <= 30 {
					return model
				}
			}
		}
	}

	// 华为 AR 系列路由器
	if contains(pattern, "AR[0-9") {
		for i := 0; i < len(descrUpper)-4; i++ {
			if descrUpper[i] == 'A' && descrUpper[i+1] == 'R' && isDigit(descrUpper[i+2]) && isDigit(descrUpper[i+3]) {
				end := i + 4
				for end < len(descrUpper) && isDigit(descrUpper[end]) {
					end++
				}
				return descr[i:end]
			}
		}
	}

	// 锐捷 RG-S 系列
	if contains(pattern, "RG-S") {
		idx := indexOfIgnoreCase(descr, "RG-S")
		if idx >= 0 && idx+4 < len(descr) {
			end := idx + 4
			for end < len(descr) && (isDigitRune(rune(descr[end])) || isUpperRune(rune(descr[end])) || descr[end] == '-') {
				end++
			}
			model := descr[idx:end]
			if len(model) <= 30 {
				return model
			}
		}
	}

	// H3C S 系列
	if contains(pattern, "S[0-9") && containsIgnoreCase(descr, "H3C") {
		for i := 0; i < len(descrUpper)-5; i++ {
			if descrUpper[i] == 'S' && isDigit(descrUpper[i+1]) && isDigit(descrUpper[i+2]) && isDigit(descrUpper[i+3]) && isDigit(descrUpper[i+4]) {
				end := i + 5
				for end < len(descrUpper) && (isDigit(descrUpper[end]) || isUpper(descrUpper[end]) || descrUpper[end] == '-') {
					end++
				}
				model := descr[i:end]
				if len(model) <= 30 && len(model) > 4 {
					return model
				}
			}
		}
	}

	// USG 防火墙
	if contains(pattern, "USG") {
		idx := indexOfIgnoreCase(descr, "USG")
		if idx >= 0 && idx+3 < len(descr) {
			end := idx + 3
			for end < len(descr) && isDigitRune(rune(descr[end])) {
				end++
			}
			return descr[idx:end]
		}
	}

	// MSR 路由器
	if contains(pattern, "MSR") {
		idx := indexOfIgnoreCase(descr, "MSR")
		if idx >= 0 && idx+3 < len(descr) {
			end := idx + 3
			for end < len(descr) && isDigitRune(rune(descr[end])) {
				end++
			}
			return descr[idx:end]
		}
	}

	// RSR 路由器（锐捷）
	if contains(pattern, "RSR") {
		idx := indexOfIgnoreCase(descr, "RSR")
		if idx >= 0 && idx+3 < len(descr) {
			end := idx + 3
			for end < len(descr) && (isDigitRune(rune(descr[end])) || descr[end] == '-') {
				end++
			}
			model := descr[idx:end]
			if len(model) <= 20 {
				return model
			}
		}
	}

	// RG-AP 系列（锐捷AP）
	if contains(pattern, "RG-AP") {
		idx := indexOfIgnoreCase(descr, "RG-AP")
		if idx >= 0 && idx+5 < len(descr) {
			end := idx + 5
			for end < len(descr) && isDigitRune(rune(descr[end])) {
				end++
			}
			return descr[idx:end]
		}
	}

	return ""
}

// 辅助函数
func toUpper(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		result[i] = c
	}
	return string(result)
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isUpper(c byte) bool {
	return c >= 'A' && c <= 'Z'
}

func isDigitRune(c rune) bool {
	return c >= '0' && c <= '9'
}

func isUpperRune(c rune) bool {
	return c >= 'A' && c <= 'Z'
}

func contains(s, substr string) bool {
	return indexOfIgnoreCase(s, substr) >= 0
}

// PingCheck ICMP ping检测设备是否在线
func PingCheck(ip string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("ip4:icmp", ip, timeout)
	if err != nil {
		return false
	}
	defer conn.Close()

	// 设置超时
	_ = conn.SetDeadline(time.Now().Add(timeout))

	return true
}

// ScanIPRange 扫描IP范围
func ScanIPRange(startIP, endIP string, timeout time.Duration) []string {
	var results []string

	start := net.ParseIP(startIP)
	end := net.ParseIP(endIP)

	if start == nil || end == nil {
		return results
	}

	// 简单的IP范围扫描
	for ip := start; ip != nil && ipToUint32(ip) <= ipToUint32(end); ip = nextIP(ip) {
		if PingCheck(ip.String(), timeout) {
			results = append(results, ip.String())
		}
	}

	return results
}

// ipToUint32 将IP转换为uint32
func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

// nextIP 获取下一个IP
func nextIP(ip net.IP) net.IP {
	next := make(net.IP, len(ip))
	copy(next, ip)

	for i := len(next) - 1; i >= 0; i-- {
		next[i]++
		if next[i] != 0 {
			break
		}
	}

	if next.Equal(net.IPv4zero) {
		return nil
	}

	return next
}

// containsIgnoreCase 忽略大小写的字符串包含检查
//
// P2 fix: 用 strings.Contains + ToLower 替代手写循环,等价但更简洁。
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// indexOfIgnoreCase 忽略大小写的字符串查找索引
//
// P2 fix: 原实现存在 bug — 函数名声称 "IgnoreCase" 但实际是大小写敏感的
// strings.Index。修正为真正的大小写无关,以匹配函数名预期。
// 注: snmp_client.go 多个 caller 用大写 RG-S/USG/MSR 等字面量,与 sysDescr
// 通常已是同大小写,因此原 bug 在常见路径下未被触发。
func indexOfIgnoreCase(s, substr string) int {
	return strings.Index(strings.ToLower(s), strings.ToLower(substr))
}

// ScanResult 扫描结果
type ScanResult struct {
	IPAddress  string
	Online     bool
	Vendor     models.DeviceVendor
	DeviceType models.DeviceType
	SysDescr   string
	SysName    string
}

// ScanDevice 扫描单个设备
func ScanDevice(target string, community string, port int, timeout time.Duration) *ScanResult {
	result := &ScanResult{
		IPAddress: target,
	}

	// 首先检查设备是否在线
	if !PingCheck(target, timeout) {
		return result
	}
	result.Online = true

	// 尝试SNMP获取设备信息
	config := &SNMPClientConfig{
		Target:    target,
		Port:      uint16(port),
		Community: community,
		Version:   SNMPVersion2c,
		Timeout:   timeout,
	}

	client := NewSNMPClient(config)
	sysInfo, err := client.GetSystemInfo()
	if err == nil {
		result.SysDescr = sysInfo.Description
		result.SysName = sysInfo.Name
		result.Vendor = DetectVendor(sysInfo.Description)
		result.DeviceType = DetectDeviceType(sysInfo.Description)
	}

	return result
}

// ConvertPortToInt 将端口字符串转换为整数
func ConvertPortToInt(portStr string) (int, error) {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("无效的端口号: %w", err)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("端口号超出范围: %d", port)
	}
	return port, nil
}
