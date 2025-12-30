package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// SMTP配置 - 请根据实际情况修改
const (
	SMTPHost     = "21.0.120.22"      // SMTP服务器地址
	SMTPPort     = 25                  // SMTP端口
	SMTPUsername = "guanliyuan-014@cpic.com.cn" // 邮箱账号
	SMTPPassword = "Cpic@1234"         // 邮箱密码
	TestToEmail  = "chenchao-076@cpic.com.cn" // 测试收件人
)

func main() {
	fmt.Println("========================================")
	fmt.Println("SMTP 邮件服务器连接测试工具")
	fmt.Println("========================================")
	fmt.Printf("服务器: %s:%d\n", SMTPHost, SMTPPort)
	fmt.Printf("账号: %s\n", SMTPUsername)
	fmt.Printf("测试收件人: %s\n", TestToEmail)
	fmt.Println("========================================\n")

	addr := fmt.Sprintf("%s:%d", SMTPHost, SMTPPort)

	// 测试1: TCP连接测试
	fmt.Println("[测试1] TCP连接测试...")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("  ❌ TCP连接失败: %v\n\n", err)
		return
	}
	conn.Close()
	fmt.Println("  ✅ TCP连接成功\n")

	// 测试2: 纯SMTP连接（不加密）
	fmt.Println("[测试2] 纯SMTP连接（不加密）...")
	err = testPlainSMTP(addr)
	if err != nil {
		fmt.Printf("  ❌ 纯SMTP连接失败: %v\n\n", err)
	} else {
		fmt.Println("  ✅ 纯SMTP连接成功\n")
	}

	// 测试3: SMTP + STARTTLS
	fmt.Println("[测试3] SMTP + STARTTLS...")
	err = testSTARTTLS(addr)
	if err != nil {
		fmt.Printf("  ❌ STARTTLS连接失败: %v\n\n", err)
	} else {
		fmt.Println("  ✅ STARTTLS连接成功\n")
	}

	// 测试4: SMTP over SSL/TLS (端口465)
	fmt.Println("[测试4] SMTP over SSL/TLS...")
	err = testSSL(addr)
	if err != nil {
		fmt.Printf("  ❌ SSL/TLS连接失败: %v\n\n", err)
	} else {
		fmt.Println("  ✅ SSL/TLS连接成功\n")
	}

	// 测试5: 发送测试邮件
	fmt.Println("[测试5] 发送测试邮件...")
	err = sendTestEmail(addr)
	if err != nil {
		fmt.Printf("  ❌ 发送邮件失败: %v\n\n", err)
	} else {
		fmt.Println("  ✅ 测试邮件发送成功，请检查收件箱\n")
	}

	fmt.Println("========================================")
	fmt.Println("测试完成！")
	fmt.Println("========================================")
}

// testPlainSMTP 测试纯SMTP连接
func testPlainSMTP(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return err
	}
	defer client.Close()

	// 检查服务器支持的扩展
	if ok, _ := client.Extension("STARTTLS"); ok {
		fmt.Println("    ℹ️  服务器声明支持STARTTLS")
	} else {
		fmt.Println("    ℹ️  服务器不支持STARTTLS")
	}

	// 尝试认证
	auth := smtp.PlainAuth("", SMTPUsername, SMTPPassword, strings.Split(addr, ":")[0])
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("认证失败: %w", err)
	}

	return nil
}

// testSTARTTLS 测试STARTTLS连接
func testSTARTTLS(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return err
	}
	defer client.Close()

	// 检查是否支持STARTTLS
	if ok, _ := client.Extension("STARTTLS"); !ok {
		return fmt.Errorf("服务器不支持STARTTLS")
	}

	// 启动TLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         strings.Split(addr, ":")[0],
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return err
	}

	// 认证
	auth := smtp.PlainAuth("", SMTPUsername, SMTPPassword, strings.Split(addr, ":")[0])
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("认证失败: %w", err)
	}

	return nil
}

// testSSL 测试SSL/TLS连接
func testSSL(addr string) error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         strings.Split(addr, ":")[0],
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return err
	}
	defer client.Close()

	// 认证
	auth := smtp.PlainAuth("", SMTPUsername, SMTPPassword, strings.Split(addr, ":")[0])
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("认证失败: %w", err)
	}

	return nil
}

// sendTestEmail 发送测试邮件
func sendTestEmail(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return err
	}
	defer client.Close()

	// 尝试STARTTLS（如果支持）
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         strings.Split(addr, ":")[0],
		}
		if err := client.StartTLS(tlsConfig); err == nil {
			fmt.Println("    ℹ️  使用STARTTLS加密连接")
		}
		// STARTTLS失败则继续使用纯SMTP
	}

	// 认证
	auth := smtp.PlainAuth("", SMTPUsername, SMTPPassword, strings.Split(addr, ":")[0])
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("认证失败: %w", err)
	}

	// 设置发件人和收件人
	if err := client.Mail(SMTPUsername); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}

	if err := client.Rcpt(TestToEmail); err != nil {
		return fmt.Errorf("设置收件人失败: %w", err)
	}

	// 构建邮件内容
	from := SMTPUsername
	to := TestToEmail
	subject := "测试邮件"
	body := "这是一封测试邮件。\n\n如果收到此邮件，说明SMTP服务器配置正确。"

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		from, to, subject, body)

	// 发送邮件
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("获取数据写入器失败: %w", err)
	}
	defer writer.Close()

	_, err = writer.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}

	return nil
}
