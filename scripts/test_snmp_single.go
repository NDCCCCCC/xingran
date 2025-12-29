package main

import (
	"fmt"
	"time"

	"github.com/gosnmp/gosnmp"
)

// 只测试正确的 community
func main() {
	target := "10.62.117.189"
	community := "cpic1234"  // 只使用正确的 community
	port := 161

	fmt.Println("========================================")
	fmt.Println("单 Community 测试（只用 cpic1234）")
	fmt.Println("========================================")

	// 调用两次，看看是否成功
	for i := 1; i <= 2; i++ {
		fmt.Printf("\n【尝试 %d】\n", i)
		snmpProbe(target, community, port)
		time.Sleep(100 * time.Millisecond)
	}
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
