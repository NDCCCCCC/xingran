package main

import (
	"fmt"
	"time"

	"github.com/gosnmp/gosnmp"
)

func main() {
	// 配置参数
	target := "10.62.117.189"
	community := "cpic1234"
	port := 161

	fmt.Println("========================================")
	fmt.Println("SNMP 探测测试工具")
	fmt.Println("========================================")
	fmt.Printf("目标: %s\n", target)
	fmt.Printf("Community: %s\n", community)
	fmt.Printf("端口: %d\n", port)
	fmt.Println("========================================\n")

	// 测试1: SNMPv1
	fmt.Println("【测试1】 SNMPv1")
	testSNMP(target, community, port, gosnmp.Version1, "SNMPv1")
	fmt.Println()

	// 测试2: SNMPv2c
	fmt.Println("【测试2】 SNMPv2c")
	testSNMP(target, community, port, gosnmp.Version2c, "SNMPv2c")
	fmt.Println()

	// 测试3: SNMPv2c with TCP
	fmt.Println("【测试3】 SNMPv2c over TCP")
	testSNMPTCP(target, community, port, gosnmp.Version2c, "SNMPv2c+TCP")
	fmt.Println()

	// 测试4: 多次请求测试
	fmt.Println("【测试4】 连续10次请求测试")
	testMultipleRequests(target, community, port, gosnmp.Version2c)
	fmt.Println()

	fmt.Println("========================================")
	fmt.Println("测试完成！")
	fmt.Println("========================================")
}

func testSNMP(target, community string, port int, version gosnmp.SnmpVersion, versionName string) {
	// 创建SNMP客户端
	client := &gosnmp.GoSNMP{
		Target:    target,
		Port:      uint16(port),
		Community: community,
		Version:   version,
		Timeout:   time.Duration(5) * time.Second,
		MaxOids:   1,
	}

	fmt.Printf("  配置: Version=%d, Transport=%s, Timeout=%v\n", version, client.Transport, client.Timeout)

	// 连接
	fmt.Print("  连接中... ")
	err := client.Connect()
	if err != nil {
		fmt.Printf("失败: %v\n", err)
		return
	}
	defer client.Conn.Close()
	fmt.Println("成功!")

	// 测试获取 sysName
	fmt.Print("  获取 sysName.1.3.6.1.2.1.1.5.0... ")
	start := time.Now()
	result, err := client.Get([]string{".1.3.6.1.2.1.1.5.0"})
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("失败: %v (耗时: %v)\n", err, elapsed)
		return
	}

	if len(result.Variables) > 0 {
		variable := result.Variables[0]
		var value string
		switch v := variable.Value.(type) {
		case string:
			value = v
		case []byte:
			value = string(v)
		}
		fmt.Printf("成功! 值=%s (耗时: %v)\n", value, elapsed)
	} else {
		fmt.Printf("成功但没有数据 (耗时: %v)\n", elapsed)
	}

	// 测试获取 sysDescr
	fmt.Print("  获取 sysDescr.1.3.6.1.2.1.1.1.0... ")
	start = time.Now()
	result, err = client.Get([]string{".1.3.6.1.2.1.1.1.0"})
	elapsed = time.Since(start)

	if err != nil {
		fmt.Printf("失败: %v (耗时: %v)\n", err, elapsed)
		return
	}

	if len(result.Variables) > 0 {
		variable := result.Variables[0]
		var value string
		switch v := variable.Value.(type) {
		case string:
			value = v
		case []byte:
			value = string(v)
		}
		// 只显示前100个字符
		if len(value) > 100 {
			fmt.Printf("成功! 值=%s... (耗时: %v)\n", value[:100], elapsed)
		} else {
			fmt.Printf("成功! 值=%s (耗时: %v)\n", value, elapsed)
		}
	}
}

func testSNMPTCP(target, community string, port int, version gosnmp.SnmpVersion, versionName string) {
	// 创建SNMP客户端（使用TCP）
	client := &gosnmp.GoSNMP{
		Target:    target,
		Port:      uint16(port),
		Community: community,
		Version:   version,
		Timeout:   time.Duration(5) * time.Second,
		Transport: "tcp", // 使用TCP
		MaxOids:   1,
	}

	fmt.Printf("  配置: Version=%d, Transport=%s, Timeout=%v\n", version, client.Transport, client.Timeout)

	// 连接
	fmt.Print("  连接中... ")
	err := client.Connect()
	if err != nil {
		fmt.Printf("失败: %v\n", err)
		return
	}
	defer client.Conn.Close()
	fmt.Println("成功!")

	// 测试获取 sysName
	fmt.Print("  获取 sysName... ")
	start := time.Now()
	result, err := client.Get([]string{".1.3.6.1.2.1.1.5.0"})
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("失败: %v (耗时: %v)\n", err, elapsed)
		return
	}

	if len(result.Variables) > 0 {
		variable := result.Variables[0]
		var value string
		switch v := variable.Value.(type) {
		case string:
			value = v
		case []byte:
			value = string(v)
		}
		fmt.Printf("成功! 值=%s (耗时: %v)\n", value, elapsed)
	}
}

func testMultipleRequests(target, community string, port int, version gosnmp.SnmpVersion) {
	client := &gosnmp.GoSNMP{
		Target:    target,
		Port:      uint16(port),
		Community: community,
		Version:   version,
		Timeout:   time.Duration(5) * time.Second,
		MaxOids:   1,
	}

	err := client.Connect()
	if err != nil {
		fmt.Printf("  连接失败: %v\n", err)
		return
	}
	defer client.Conn.Close()

	successCount := 0
	failCount := 0
	var totalTime time.Duration

	for i := 1; i <= 10; i++ {
		fmt.Printf("  请求 #%2d: ", i)
		start := time.Now()
		result, err := client.Get([]string{".1.3.6.1.2.1.1.5.0"})
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("失败 - %v (耗时: %v)\n", err, elapsed)
			failCount++
		} else {
			successCount++
			totalTime += elapsed
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
			} else {
				fmt.Printf("成功但无数据 (耗时: %v)\n", elapsed)
			}
		}

		// 间隔100ms
		if i < 10 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	fmt.Printf("\n  统计: 成功=%d/10, 失败=%d/10, 平均耗时=%v\n",
		successCount, failCount, totalTime/time.Duration(successCount))
}
