package main

import (
	"fmt"
	"time"

	"github.com/gosnmp/gosnmp"
)

// 测试多个 community 的场景
func main() {
	target := "10.62.117.189"
	communities := []string{"cpic321", "cpic1234"}
	port := 161

	fmt.Println("========================================")
	fmt.Println("多 Community 测试（模拟服务代码）")
	fmt.Println("========================================")

	for i, community := range communities {
		fmt.Printf("\n【尝试 %d】 Community: %s\n", i+1, community)
		snmpProbe(target, community, port)

		// 添加间隔，模拟真实场景
		if i < len(communities)-1 {
			time.Sleep(100 * time.Millisecond)
		}
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

	fmt.Printf("  配置: Version=%d, Timeout=%v\n", snmpClient.Version, snmpClient.Timeout)

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
