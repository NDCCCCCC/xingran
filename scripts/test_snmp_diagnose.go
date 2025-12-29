package main

import (
	"fmt"
	"net"
	"time"

	"github.com/gosnmp/gosnmp"
)

// 详细的UDP连接状态诊断
func main() {
	target := "10.62.117.189"
	communities := []string{"cpic321", "cpic1234"}
	port := 161

	fmt.Println("========================================")
	fmt.Println("UDP 连接状态详细诊断")
	fmt.Println("========================================")

	for i, community := range communities {
		fmt.Printf("\n【尝试 %d】 Community: %s\n", i+1, community)

		// 测试原始UDP连接
		fmt.Println("  测试原始UDP连接...")
		rawConn, err := net.DialTimeout("udp", fmt.Sprintf("%s:%d", target, port), 2*time.Second)
		if err != nil {
			fmt.Printf("    原始UDP连接失败: %v\n", err)
		} else {
			fmt.Printf("    原始UDP连接成功: 本地=%s, 远程=%s\n", rawConn.LocalAddr(), rawConn.RemoteAddr())

			// 设置超时并关闭
			rawConn.SetDeadline(time.Now().Add(1 * time.Second))
			rawConn.Close()
			fmt.Printf("    连接已关闭\n")
		}

		// 使用gosnmp
		fmt.Println("  测试GoSNMP...")
		snmpConn := &gosnmp.GoSNMP{
			Target:    target,
			Port:      uint16(port),
			Community: community,
			Version:   gosnmp.Version2c,
			Timeout:   time.Duration(5) * time.Second,
			MaxOids:   1,
		}

		err = snmpConn.Connect()
		if err != nil {
			fmt.Printf("    GoSNMP连接失败: %v\n", err)
		} else {
			if snmpConn.Conn != nil {
				fmt.Printf("    GoSNMP连接: 本地=%s\n", snmpConn.Conn.LocalAddr())
			}

			start := time.Now()
			result, err := snmpConn.Get([]string{".1.3.6.1.2.1.1.5.0"})
			elapsed := time.Since(start)

			snmpConn.Conn.Close()

			if err != nil {
				fmt.Printf("    Get失败: %v (耗时: %v)\n", err, elapsed)
			} else {
				fmt.Printf("    Get成功! (耗时: %v)\n", elapsed)
				if len(result.Variables) > 0 {
					variable := result.Variables[0]
					var value string
					switch v := variable.Value.(type) {
					case string:
						value = v
					case []byte:
						value = string(v)
					}
					fmt.Printf("    值: %s\n", value)
				}
			}
		}

		if i < len(communities)-1 {
			fmt.Println("\n  等待2秒...")
			time.Sleep(2 * time.Second)
		}
	}
}
