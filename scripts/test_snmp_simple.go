package main

import (
	"fmt"
	"time"

	"github.com/gosnmp/gosnmp"
)

// 完全复制 testMultipleRequests 的逻辑
func main() {
	target := "10.62.117.189"
	community := "cpic1234"
	port := 161

	fmt.Println("========================================")
	fmt.Println("SNMP 简单测试（复制 testMultipleRequests）")
	fmt.Println("========================================")

	client := &gosnmp.GoSNMP{
		Target:    target,
		Port:      uint16(port),
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(5) * time.Second,
		MaxOids:   1,
	}

	err := client.Connect()
	if err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}
	defer client.Conn.Close()

	// 测试10次
	for i := 1; i <= 10; i++ {
		fmt.Printf("请求 #%2d: ", i)
		start := time.Now()
		result, err := client.Get([]string{".1.3.6.1.2.1.1.5.0"})
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("失败 - %v (耗时: %v)\n", err, elapsed)
		} else {
			if len(result.Variables) > 0 {
				variable := result.Variables[0]
				var value string
				switch v := variable.Value.(type) {
				case string:
					value = v
				case []byte:
					value = string(v)
				}
				fmt.Printf("成功 - 值=%s (耗时: %v)\n", value, elapsed)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n========================================")
	fmt.Println("测试完成")
	fmt.Println("========================================")
}
