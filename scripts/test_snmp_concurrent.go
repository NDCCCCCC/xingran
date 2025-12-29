package main

import (
	"fmt"
	"time"

	"github.com/gosnmp/gosnmp"
)

// 测试并发探测是否能解决顺序探测的问题
func main() {
	target := "10.62.117.189"
	communities := []string{"cpic321", "cpic1234"}
	port := 161

	fmt.Println("========================================")
	fmt.Println("并发 SNMP 探测测试")
	fmt.Println("========================================")

	type probeResult struct {
		device *DiscoveredDevice
		err    error
		index  int
	}

	resultChan := make(chan probeResult, len(communities))

	// 启动多个goroutine并发探测
	for i, community := range communities {
		go func(idx int, comm string) {
			fmt.Printf("启动 goroutine #%d, community: %s\n", idx+1, comm)

			snmpClient := &gosnmp.GoSNMP{
				Target:    target,
				Port:      uint16(port),
				Community: comm,
				Version:   gosnmp.Version2c,
				Timeout:   time.Duration(5) * time.Second,
				MaxOids:   1,
			}

			err := snmpClient.Connect()
			if err != nil {
				fmt.Printf("goroutine #%d 连接失败: %v\n", idx+1, err)
				resultChan <- probeResult{err: err, index: idx}
				return
			}

			start := time.Now()
			sysName, err := snmpClient.Get([]string{".1.3.6.1.2.1.1.5.0"})
			elapsed := time.Since(start)

			if err != nil {
				snmpClient.Conn.Close()
				fmt.Printf("goroutine #%d Get失败: %v (耗时: %v)\n", idx+1, err, elapsed)
				resultChan <- probeResult{err: err, index: idx}
				return
			}

			sysDescr, _ := snmpClient.Get([]string{".1.3.6.1.2.1.1.1.0"})
			snmpClient.Conn.Close()

			device := &DiscoveredDevice{
				IPAddress: target,
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

			fmt.Printf("goroutine #%d 探测成功! 耗时: %v, SysName=%s\n", idx+1, elapsed, device.SysName)
			resultChan <- probeResult{device: device, index: idx}
		}(i, community)
	}

	// 等待第一个成功的结果
	var discoveredDevice *DiscoveredDevice
	for range communities {
		result := <-resultChan
		if result.device != nil && result.device.IsAlive {
			discoveredDevice = result.device
			fmt.Printf("\n✅ 收到成功结果! 使用 goroutine #%d\n", result.index+1)
			break
		}
	}

	if discoveredDevice != nil {
		fmt.Printf("\n最终结果: 成功!\n")
		fmt.Printf("  设备名: %s\n", discoveredDevice.SysName)
		fmt.Printf("  系统描述: %s\n", discoveredDevice.SysDescr[:50]+"...")
	} else {
		fmt.Printf("\n最终结果: 失败!\n")
	}
}

type DiscoveredDevice struct {
	IPAddress  string
	SysName    string
	SysDescr   string
	IsAlive    bool
}
