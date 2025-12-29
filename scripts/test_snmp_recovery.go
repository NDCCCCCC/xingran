package main

import (
	"fmt"
	"time"

	"github.com/gosnmp/gosnmp"
)

// 测试：先用错误的community，然后等待不同时间后重试
func main() {
	target := "10.62.117.189"
	port := 161

	fmt.Println("========================================")
	fmt.Println("SNMP 设备恢复测试")
	fmt.Println("========================================")

	// 步骤1: 用错误的community探测一次
	fmt.Println("\n【步骤1】使用错误的community (cpic321)")
	snmpTest(target, "cpic321", port)

	// 步骤2: 立即用正确的community
	fmt.Println("\n【步骤2】立即使用正确的community (cpic1234)")
	snmpTest(target, "cpic1234", port)

	// 步骤3: 等待5秒后再试
	fmt.Println("\n【步骤3】等待5秒后重试")
	time.Sleep(5 * time.Second)
	snmpTest(target, "cpic1234", port)

	// 步骤4: 等待15秒后再试
	fmt.Println("\n【步骤4】等待15秒后重试")
	time.Sleep(15 * time.Second)
	snmpTest(target, "cpic1234", port)

	// 步骤5: 等待30秒后再试
	fmt.Println("\n【步骤5】等待30秒后重试")
	time.Sleep(30 * time.Second)
	snmpTest(target, "cpic1234", port)

	fmt.Println("\n========================================")
	fmt.Println("测试完成")
	fmt.Println("========================================")
}

func snmpTest(target, community string, port int) {
	snmpClient := &gosnmp.GoSNMP{
		Target:    target,
		Port:      uint16(port),
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(5) * time.Second,
		MaxOids:   1,
	}

	err := snmpClient.Connect()
	if err != nil {
		fmt.Printf("  连接失败: %v\n", err)
		return
	}
	defer snmpClient.Conn.Close()

	start := time.Now()
	result, err := snmpClient.Get([]string{".1.3.6.1.2.1.1.5.0"})
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("  结果: 失败 - %v (耗时: %v)\n", err, elapsed)
	} else {
		fmt.Printf("  结果: 成功! (耗时: %v)\n", elapsed)
		if len(result.Variables) > 0 {
			variable := result.Variables[0]
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
