package main

import (
	"fmt"
	"time"

	"github.com/gosnmp/gosnmp"
)

// 直接测试 SNMP 探测，模拟服务代码的调用方式
func main() {
	target := "10.62.117.189"
	communities := []string{"cpic321", "cpic1234"}
	port := 161

	fmt.Println("========================================")
	fmt.Println("SNMP 探测测试（模拟服务代码逻辑）")
	fmt.Println("========================================")

	for i, community := range communities {
		fmt.Printf("\n【尝试 %d】 Community: %s\n", i+1, community)
		result := snmpProbe(target, community, port)
		if result != nil && result.IsAlive {
			fmt.Printf("✅ 成功！设备名: %s, 厂商: %s, 型号: %s\n",
				result.SysName, result.Vendor, result.SysDescr[:30])
			return
		}
	}

	fmt.Println("\n❌ 所有 community 都失败")
}

type DiscoveredDevice struct {
	IPAddress  string
	SysName    string
	SysDescr   string
	Vendor     string
	DeviceType string
	IsAlive    bool
}

// 完全复制服务代码的 snmpProbe 函数
func snmpProbe(ip, community string, port int) *DiscoveredDevice {
	fmt.Printf("[DEBUG] snmpProbe 开始: ip=%s, community=%s, port=%d\n", ip, community, port)

	// 添加调试：打印 gosnmp 版本
	fmt.Printf("[DEBUG] gosnmp.Version2c = %d\n", gosnmp.Version2c)

	// 创建SNMP客户端（配置与测试脚本完全一致）
	snmpClient := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      uint16(port),
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(5) * time.Second,
		MaxOids:   1,
	}

	fmt.Printf("[DEBUG] SNMP配置: Version=%d, Timeout=%v, MaxOids=%d, Transport=%q\n",
		snmpClient.Version, snmpClient.Timeout, snmpClient.MaxOids, snmpClient.Transport)

	// 连接
	fmt.Printf("[DEBUG] 尝试连接 SNMP 服务器...\n")
	err := snmpClient.Connect()
	if err != nil {
		fmt.Printf("[DEBUG] SNMP 连接失败: %v\n", err)
		return &DiscoveredDevice{
			IPAddress: ip,
			IsAlive:   false,
		}
	}
	fmt.Printf("[DEBUG] SNMP 连接成功, Conn类型=%T\n", snmpClient.Conn)

	// 使用 defer 确保连接在函数结束时关闭（与测试脚本一致）
	defer snmpClient.Conn.Close()
	fmt.Printf("[DEBUG] defer Close() 已设置\n")

	// 在同一个连接上获取 sysName（与测试脚本一致）
	fmt.Printf("[DEBUG] 获取 sysName (OID: .1.3.6.1.2.1.1.5.0)...\n")
	startTime := time.Now()
	sysName, err := snmpClient.Get([]string{".1.3.6.1.2.1.1.5.0"})
	elapsed := time.Since(startTime)

	if err != nil {
		fmt.Printf("[DEBUG] 获取 sysName 失败: %v (耗时: %v)\n", err, elapsed)
		return &DiscoveredDevice{
			IPAddress: ip,
			IsAlive:   false,
		}
	}
	fmt.Printf("[DEBUG] sysName 获取成功, 耗时: %v, 变量数量: %d\n", elapsed, len(sysName.Variables))

	// 在同一个连接上获取 sysDescr（与测试脚本一致）
	fmt.Printf("[DEBUG] 获取 sysDescr (OID: .1.3.6.1.2.1.1.1.0)...\n")
	startTime = time.Now()
	sysDescr, err := snmpClient.Get([]string{".1.3.6.1.2.1.1.1.0"})
	elapsed = time.Since(startTime)

	if err != nil {
		fmt.Printf("[DEBUG] sysDescr 获取失败: %v (耗时: %v)\n", err, elapsed)
		// sysDescr 失败不算致命错误，继续处理 sysName
	} else {
		fmt.Printf("[DEBUG] sysDescr 获取成功, 耗时: %v, 变量数量: %d\n", elapsed, len(sysDescr.Variables))
	}

	// 解析结果
	device := &DiscoveredDevice{
		IPAddress: ip,
		IsAlive:   true,
	}

	if len(sysName.Variables) > 0 {
		variable := sysName.Variables[0]
		switch v := variable.Value.(type) {
		case string:
			device.SysName = v
		case []byte:
			device.SysName = string(v)
		}
		fmt.Printf("[DEBUG] sysName=%s\n", device.SysName)
	}

	if len(sysDescr.Variables) > 0 {
		variable := sysDescr.Variables[0]
		switch v := variable.Value.(type) {
		case string:
			device.SysDescr = v
		case []byte:
			device.SysDescr = string(v)
		}
	}

	fmt.Printf("[DEBUG] snmpProbe 完成: IsAlive=%v\n", device.IsAlive)
	return device
}
