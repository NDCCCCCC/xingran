package main

import (
	"fmt"
	"net"
	"time"

	"github.com/gosnmp/gosnmp"
)

// 测试：确保连接完全关闭并添加延迟
func main() {
	target := "10.62.117.189"
	communities := []string{"cpic321", "cpic1234"}
	port := 161

	fmt.Println("========================================")
	fmt.Println("多 Community 测试（确保完全关闭+延迟）")
	fmt.Println("========================================")

	for i, community := range communities {
		fmt.Printf("\n【尝试 %d】 Community: %s\n", i+1, community)

		var snmpClient *gosnmp.GoSNMP
		var conn net.Conn

		// 创建客户端
		snmpClient = &gosnmp.GoSNMP{
			Target:    target,
			Port:      uint16(port),
			Community: community,
			Version:   gosnmp.Version2c,
			Timeout:   time.Duration(5) * time.Second,
			MaxOids:   1,
		}

		// 连接
		err := snmpClient.Connect()
		if err != nil {
			fmt.Printf("  连接失败: %v\n", err)
			continue
		}

		// 保存连接引用
		conn = snmpClient.Conn

		// 获取数据
		startTime := time.Now()
		sysName, err := snmpClient.Get([]string{".1.3.6.1.2.1.1.5.0"})
		elapsed := time.Since(startTime)

		// 立即关闭连接
		conn.Close()

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

		// 添加延迟，让端口有时间释放
		if i < len(communities)-1 {
			fmt.Println("  等待 500ms 让端口释放...")
			time.Sleep(500 * time.Millisecond)
		}
	}
}
