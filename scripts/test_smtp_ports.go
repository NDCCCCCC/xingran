package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// SMTP配置
const (
	SMTPHost     = "21.0.120.22"
	SMTPUsername = "guanliyuan-014@cpic.com.cn"
	SMTPPassword = "Cpic@1234"
	TestToEmail  = "chenchao-076@cpic.com.cn"
)

// 要测试的端口列表
var testPorts = []int{25, 465, 587, 2525}

func main() {
	fmt.Println("========================================")
	fmt.Println("SMTP 多端口连接测试工具")
	fmt.Println("========================================")
	fmt.Printf("服务器: %s\n", SMTPHost)
	fmt.Printf("账号: %s\n", SMTPUsername)
	fmt.Println("========================================\n")

	successfulPort := 0

	for _, port := range testPorts {
		addr := fmt.Sprintf("%s:%d", SMTPHost, port)
		fmt.Printf("\n========================================\n")
		fmt.Printf("测试端口: %d\n", port)
		fmt.Printf("========================================\n")

		// TCP连接测试
		fmt.Printf("[TCP连接] ")
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			fmt.Printf("❌ 失败: %v\n", err)
			continue
		}
		conn.Close()
		fmt.Println("✅ 成功")

		// 测试SSL/TLS
		fmt.Printf("[SSL/TLS]  ")
		if testSSL(addr) == nil {
			fmt.Println("✅ 成功")
			successfulPort = port
			goto SendTest
		}

		// 测试STARTTLS
		fmt.Printf("[STARTTLS] ")
		if testSTARTTLS(addr) == nil {
			fmt.Println("✅ 成功")
			successfulPort = port
			goto SendTest
		}

		// 测试纯SMTP
		fmt.Printf("[纯SMTP]   ")
		if testPlainSMTP(addr) == nil {
			fmt.Println("✅ 成功")
			successfulPort = port
			goto SendTest
		}

		fmt.Println("所有连接方式均失败")
		continue

	SendTest:
		// 发送测试邮件
		fmt.Printf("[发送邮件] ")
		if sendTestEmail(addr) == nil {
			fmt.Println("✅ 成功！")
			fmt.Printf("\n========================================\n")
			fmt.Printf("找到可用配置！\n")
			fmt.Printf("服务器: %s:%d\n", SMTPHost, port)
			fmt.Printf("请更新系统配置\n")
			fmt.Printf("========================================\n")
			return
		} else {
			fmt.Println("❌ 失败")
		}
	}

	if successfulPort == 0 {
		fmt.Println("\n========================================")
		fmt.Println("❌ 所有端口测试均失败！")
		fmt.Println("\n可能的原因：")
		fmt.Println("1. 邮件服务器地址不正确")
		fmt.Println("2. 账号或密码错误")
		fmt.Println("3. 防火墙阻止了SMTP连接")
		fmt.Println("4. 服务器使用了非标准端口")
		fmt.Println("\n建议联系邮件服务器管理员确认：")
		fmt.Println("  - SMTP服务器地址和端口")
		fmt.Println("  - 是否需要SSL/TLS加密")
		fmt.Println("  - 账号权限是否正确")
		fmt.Println("========================================")
	}
}

func testSSL(addr string) error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         strings.Split(addr, ":")[0],
	}

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return err
	}
	defer client.Close()

	auth := smtp.PlainAuth("", SMTPUsername, SMTPPassword, strings.Split(addr, ":")[0])
	return client.Auth(auth)
}

func testSTARTTLS(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return err
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); !ok {
		return fmt.Errorf("不支持STARTTLS")
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // 测试时跳过证书验证
		ServerName:         strings.Split(addr, ":")[0],
	}

	if err := client.StartTLS(tlsConfig); err != nil {
		return err
	}

	auth := smtp.PlainAuth("", SMTPUsername, SMTPPassword, strings.Split(addr, ":")[0])
	return client.Auth(auth)
}

func testPlainSMTP(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return err
	}
	defer client.Close()

	auth := smtp.PlainAuth("", SMTPUsername, SMTPPassword, strings.Split(addr, ":")[0])
	return client.Auth(auth)
}

func sendTestEmail(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return err
	}
	defer client.Close()

	// 尝试STARTTLS
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         strings.Split(addr, ":")[0],
		}
		client.StartTLS(tlsConfig)
	}

	auth := smtp.PlainAuth("", SMTPUsername, SMTPPassword, strings.Split(addr, ":")[0])
	if err := client.Auth(auth); err != nil {
		return err
	}

	if err := client.Mail(SMTPUsername); err != nil {
		return err
	}

	if err := client.Rcpt(TestToEmail); err != nil {
		return err
	}

	from := SMTPUsername
	to := TestToEmail
	subject := "测试邮件"
	body := "这是一封测试邮件。\n\n如果收到此邮件，说明SMTP服务器配置正确。"

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		from, to, subject, body)

	writer, err := client.Data()
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = writer.Write([]byte(msg))
	return err
}
