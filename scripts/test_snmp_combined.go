package main

import (
	"fmt"
	"time"

	"github.com/gosnmp/gosnmp"
)

func main() {
	target := "10.62.117.189"
	community := "cpic1234"
	port := 161

	fmt.Println("========================================")
	fmt.Println("SNMP 组合测试")
	fmt.Println("========================================")

	// 方法1：testMultipleRequests 的方式（已知成功）
	fmt.Println("\n【方法1】 testMultipleRequests 方式:")
	client1 := &gosnmp.GoSNMP{
		Target:    target,
		Port:      uint16(port),
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(5) * time.Second,
		MaxOids:   1,
	}

	err := client1.Connect()
	if err != nil {
		fmt.Printf("  连接失败: %v\n", err)
	} else {
		defer client1.Conn.Close()
		start := time.Now()
		_, err := client1.Get([]string{".1.3.6.1.2.1.1.5.0"})
		elapsed := time.Since(start)
		if err != nil {
			fmt.Printf("  Get 失败: %v (耗时: %v)\n", err, elapsed)
		} else {
			fmt.Printf("  Get 成功! 耗时: %v\n", elapsed)
		}
	}

	// 等待一下
	time.Sleep(500 * time.Millisecond)

	// 方法2：snmpProbe 的方式（已知失败）
	fmt.Println("\n【方法2】 snmpProbe 方式:")
	snmpProbe(target, community, port)
}

func snmpProbe(ip, community string, port int) {
	snmpClient := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      uint16(port),
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(5) * time.Second,
		MaxOids:   1,
	}

	fmt.Printf("  配置: Version=%d, Timeout=%v, MaxOids=%d\n",
		snmpClient.Version, snmpClient.Timeout, snmpClient.MaxOids)

	err := snmpClient.Connect()
	if err != nil {
		fmt.Printf("  连接失败: %v\n", err)
		return
	}
	defer snmpClient.Conn.Close()
	fmt.Printf("  连接成功\n")

	startTime := time.Now()
	sysName, err := snmpClient.Get([]string{".1.3.6.1.2.1.1.5.0"})
	elapsed := time.Since(startTime)

	if err != nil {
		fmt.Printf("  Get 失败: %v (耗时: %v)\n", err, elapsed)
	} else {
		fmt.Printf("  Get 成功! 耗时: %v\n", elapsed)
		if len(sysName.Variables) > 0 {
			variable := sysName.Variables[0]
			var value string
			switch v := variable.Value.(type) {
			case string:
				value = v
			case []byte:
				value = string(v)
			}
			fmt.Printf("  值: %s\n", value)
		}
	}
}
