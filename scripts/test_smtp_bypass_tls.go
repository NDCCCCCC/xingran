package main

import (
	"fmt"
	"net/smtp"
	"strings"
)

// SMTP配置
const (
	mailHost = "21.0.120.22"
	mailUser = "guanliyuan-014@cpic.com.cn"
	mailPass = "Cpic@1234"
	sender   = "guanliyuan-014@cpic.com.cn"
)

// plainAuth 不检查TLS的认证（绕过Go的安全限制）
type plainAuth struct {
	identity, username, password, host string
}

func (a *plainAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "PLAIN", []byte("\x00" + a.username + "\x00" + a.password), nil
}

func (a *plainAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		return nil, fmt.Errorf("unexpected server challenge")
	}
	return nil, nil
}

func newPlainAuth(identity, username, password, host string) smtp.Auth {
	return &plainAuth{identity, username, password, host}
}

func main() {
	receiver := []string{"chenchao-076@cpic.com.cn"}

	fmt.Println("========================================")
	fmt.Println("SMTP 邮件发送测试 (绕过TLS检查)")
	fmt.Println("========================================")
	fmt.Printf("服务器: %s:25\n", mailHost)
	fmt.Printf("账号: %s\n", mailUser)
	fmt.Printf("收件人: %s\n", receiver[0])
	fmt.Println("========================================\n")

	title := "测试邮件"
	content := "这是一封测试邮件。\n\n如果收到此邮件，说明SMTP服务器配置正确。"

	result := sendmail(title, receiver, content)
	fmt.Printf("\n发送结果: %s\n", result)
	fmt.Println("========================================")
}

func sendmail(title string, receiver []string, content string) string {
	// 创建SMTP客户端
	smtpObj, err := smtp.Dial(mailHost + ":25")
	if err != nil {
		return fmt.Sprintf("ERROR: 连接失败 - %v", err)
	}
	defer smtpObj.Close()

	fmt.Println("✅ SMTP连接成功")

	// 使用自定义的PlainAuth（不检查TLS）
	auth := newPlainAuth("", mailUser, mailPass, strings.Split(mailHost+":25", ":")[0])
	if err := smtpObj.Auth(auth); err != nil {
		return fmt.Sprintf("ERROR: 认证失败 - %v", err)
	}

	fmt.Println("✅ 认证成功")

	// 设置发件人
	if err := smtpObj.Mail(sender); err != nil {
		return fmt.Sprintf("ERROR: 设置发件人失败 - %v", err)
	}

	// 设置收件人
	for _, recv := range receiver {
		if err := smtpObj.Rcpt(recv); err != nil {
			return fmt.Sprintf("ERROR: 设置收件人失败 - %v", err)
		}
	}

	// 获取数据写入器
	writer, err := smtpObj.Data()
	if err != nil {
		return fmt.Sprintf("ERROR: 获取数据写入器失败 - %v", err)
	}
	defer writer.Close()

	// 构建邮件内容
	message := fmt.Sprintf("Subject: %s\r\nFrom: %s\r\nTo: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		title, sender, receiver[0], content)

	// 写入邮件内容
	if _, err := writer.Write([]byte(message)); err != nil {
		return fmt.Sprintf("ERROR: 写入邮件内容失败 - %v", err)
	}

	fmt.Println("✅ 邮件发送成功")
	return "success"
}
