package services

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// plainAuth 自定义认证实现，不检查TLS（兼容某些要求纯文本认证的SMTP服务器）
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

// newPlainAuth 创建自定义PlainAuth（不检查TLS）
func newPlainAuth(identity, username, password, host string) smtp.Auth {
	return &plainAuth{identity, username, password, host}
}

// EmailSenderService 邮件发送服务
type EmailSenderService struct {
	db *gorm.DB
}

// NewEmailSenderService 创建邮件发送服务
func NewEmailSenderService(db *gorm.DB) *EmailSenderService {
	return &EmailSenderService{db: db}
}

// EmailMessage 邮件消息
type EmailMessage struct {
	To       []string // 收件人列表
	Cc       []string // 抄送列表
	Bcc      []string // 密送列表
	Subject  string   // 邮件主题
	HTMLBody string   // HTML内容
	TextBody string   // 纯文本内容
}

// SendResult 发送结果
type SendResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   error  `json:"error,omitempty"`
}

// Send 发送邮件
func (s *EmailSenderService) Send(ctx context.Context, configID string, msg *EmailMessage) *SendResult {
	// 获取邮件配置
	configService := NewNotificationConfigService(s.db)
	config, err := configService.GetEmailConfigByID(ctx, configID)
	if err != nil {
		return &SendResult{
			Success: false,
			Message: "获取邮件配置失败",
			Error:   err,
		}
	}

	// 检查配置状态
	if config.Status != int(models.NotificationConfigStatusNormal) {
		return &SendResult{
			Success: false,
			Message: "邮件配置未启用",
		}
	}

	// 解密密码
	password, err := DecryptPassword(config.Password, "")
	if err != nil {
		return &SendResult{
			Success: false,
			Message: "密码解密失败",
			Error:   err,
		}
	}

	// 构建发件人地址
	fromAddress := config.Username
	if config.FromEmail != "" {
		fromAddress = config.FromEmail
	}

	// 验证收件人地址
	if len(msg.To) == 0 {
		return &SendResult{
			Success: false,
			Message: "收件人不能为空",
		}
	}

	for _, to := range msg.To {
		if _, parseErr := mail.ParseAddress(to); parseErr != nil {
			return &SendResult{
				Success: false,
				Message: fmt.Sprintf("无效的收件人地址: %s", to),
				Error:   parseErr,
			}
		}
	}

	// 发送邮件
	err = s.sendEmail(
		config.Host,
		config.Port,
		config.Username,
		password,
		fromAddress,
		config.FromName,
		msg,
		config.UseSSL,
		config.UseSTARTTLS,
	)

	if err != nil {
		return &SendResult{
			Success: false,
			Message: "邮件发送失败",
			Error:   err,
		}
	}

	return &SendResult{
		Success: true,
		Message: "邮件发送成功",
	}
}

// SendWithDefaultConfig 使用默认配置发送邮件
func (s *EmailSenderService) SendWithDefaultConfig(ctx context.Context, msg *EmailMessage) *SendResult {
	configService := NewNotificationConfigService(s.db)
	config, err := configService.GetDefaultEmailConfig(ctx)
	if err != nil {
		return &SendResult{
			Success: false,
			Message: "未设置默认邮件配置",
			Error:   err,
		}
	}

	return s.Send(ctx, config.ID, msg)
}

// sendEmail 实际发送邮件的方法
func (s *EmailSenderService) sendEmail(host string, port int, username, password, fromAddress, fromName string, msg *EmailMessage, useSSL, useSTARTTLS bool) error {
	// 构建服务器地址
	addr := fmt.Sprintf("%s:%d", host, port)

	// 构建发件人
	from := fromAddress
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, fromAddress)
	}

	// 构建邮件内容
	emailContent, err := s.buildEmailContent(from, msg)
	if err != nil {
		return fmt.Errorf("构建邮件内容失败: %w", err)
	}

	// 收件人列表
	toAddresses := append(msg.To, msg.Cc...)
	toAddresses = append(toAddresses, msg.Bcc...)

	if useSSL {
		// 使用SSL/TLS连接
		return s.sendWithTLS(addr, username, password, fromAddress, toAddresses, emailContent)
	}

	// 端口25通常用于纯SMTP，不使用STARTTLS
	// 端口587通常用于STARTTLS
	if useSTARTTLS && port != 25 {
		// 使用STARTTLS
		return s.sendWithSTARTTLS(addr, username, password, fromAddress, toAddresses, emailContent)
	}

	// 使用纯SMTP（不加密）
	return s.sendPlainSMTP(addr, username, password, fromAddress, toAddresses, emailContent)
}

// sendWithTLS 使用SSL/TLS发送邮件
func (s *EmailSenderService) sendWithTLS(addr, username, password, fromAddress string, toAddresses []string, content []byte) error {
	// 创建TLS配置
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         strings.Split(addr, ":")[0],
	}

	// 建立连接
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS连接失败: %w", err)
	}

	// 创建SMTP客户端
	client, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return fmt.Errorf("创建SMTP客户端失败: %w", err)
	}
	defer client.Close()

	// 认证（使用自定义PlainAuth，兼容非加密连接）
	auth := newPlainAuth("", username, password, strings.Split(addr, ":")[0])
	if authErr := client.Auth(auth); authErr != nil {
		return fmt.Errorf("SMTP认证失败: %w", authErr)
	}

	// 设置发件人
	if mailErr := client.Mail(fromAddress); mailErr != nil {
		return fmt.Errorf("设置发件人失败: %w", mailErr)
	}

	// 设置收件人
	for _, to := range toAddresses {
		if rcptErr := client.Rcpt(to); rcptErr != nil {
			return fmt.Errorf("设置收件人失败: %w", rcptErr)
		}
	}

	// 发送邮件内容
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("获取数据写入器失败: %w", err)
	}
	defer writer.Close()

	_, writeErr := writer.Write(content)
	if writeErr != nil {
		return fmt.Errorf("写入邮件内容失败: %w", writeErr)
	}

	return nil
}

// sendWithSTARTTLS 使用STARTTLS发送邮件
func (s *EmailSenderService) sendWithSTARTTLS(addr, username, password, fromAddress string, toAddresses []string, content []byte) error {
	// 建立连接
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}

	// 创建SMTP客户端
	client, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return fmt.Errorf("创建SMTP客户端失败: %w", err)
	}
	defer client.Close()

	// 检查是否支持STARTTLS
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         strings.Split(addr, ":")[0],
		}
		if startTLSErr := client.StartTLS(tlsConfig); startTLSErr != nil {
			return fmt.Errorf("STARTTLS失败: %w", startTLSErr)
		}
	}

	// 认证（使用自定义PlainAuth，兼容非加密连接）
	auth := newPlainAuth("", username, password, strings.Split(addr, ":")[0])
	if authErr := client.Auth(auth); authErr != nil {
		return fmt.Errorf("SMTP认证失败: %w", authErr)
	}

	// 设置发件人
	if mailErr := client.Mail(fromAddress); mailErr != nil {
		return fmt.Errorf("设置发件人失败: %w", mailErr)
	}

	// 设置收件人
	for _, to := range toAddresses {
		if rcptErr := client.Rcpt(to); rcptErr != nil {
			return fmt.Errorf("设置收件人失败: %w", rcptErr)
		}
	}

	// 发送邮件内容
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("获取数据写入器失败: %w", err)
	}
	defer writer.Close()

	_, writeErr := writer.Write(content)
	if writeErr != nil {
		return fmt.Errorf("写入邮件内容失败: %w", writeErr)
	}

	return nil
}

// sendPlainSMTP 使用纯SMTP发送邮件（不加密，不使用STARTTLS）
func (s *EmailSenderService) sendPlainSMTP(addr, username, password, fromAddress string, toAddresses []string, content []byte) error {
	// 建立连接
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}

	// 创建SMTP客户端
	client, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return fmt.Errorf("创建SMTP客户端失败: %w", err)
	}
	defer client.Close()

	// 认证（使用自定义PlainAuth，兼容非加密连接）
	auth := newPlainAuth("", username, password, strings.Split(addr, ":")[0])
	if authErr := client.Auth(auth); authErr != nil {
		return fmt.Errorf("SMTP认证失败: %w", authErr)
	}

	// 设置发件人
	if mailErr := client.Mail(fromAddress); mailErr != nil {
		return fmt.Errorf("设置发件人失败: %w", mailErr)
	}

	// 设置收件人
	for _, to := range toAddresses {
		if rcptErr := client.Rcpt(to); rcptErr != nil {
			return fmt.Errorf("设置收件人失败: %w", rcptErr)
		}
	}

	// 发送邮件内容
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("获取数据写入器失败: %w", err)
	}
	defer writer.Close()

	_, writeErr := writer.Write(content)
	if writeErr != nil {
		return fmt.Errorf("写入邮件内容失败: %w", writeErr)
	}

	return nil
}

// buildEmailContent 构建邮件内容
func (s *EmailSenderService) buildEmailContent(from string, msg *EmailMessage) ([]byte, error) {
	var content strings.Builder

	// 邮件头
	content.WriteString(fmt.Sprintf("From: %s\r\n", from))
	content.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ",")))
	if len(msg.Cc) > 0 {
		content.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.Cc, ",")))
	}
	content.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	content.WriteString("MIME-Version: 1.0\r\n")

	// 如果有HTML内容
	if msg.HTMLBody != "" {
		boundary := fmt.Sprintf("----=_NextPart_%d", 1234567890)
		content.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		content.WriteString("\r\n")

		// 纯文本部分
		if msg.TextBody != "" {
			content.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			content.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
			content.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
			content.WriteString(base64.StdEncoding.EncodeToString([]byte(msg.TextBody)))
			content.WriteString("\r\n\r\n")
		}

		// HTML部分
		content.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		content.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		content.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		content.WriteString(base64.StdEncoding.EncodeToString([]byte(msg.HTMLBody)))
		content.WriteString("\r\n\r\n")
		content.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		// 只有纯文本
		content.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		content.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		content.WriteString(base64.StdEncoding.EncodeToString([]byte(msg.TextBody)))
		content.WriteString("\r\n")
	}

	return []byte(content.String()), nil
}

// SendNoticeEmail 发送通知邮件
func (s *EmailSenderService) SendNoticeEmail(ctx context.Context, configID string, notice *models.Notice, userAddresses []string) *SendResult {
	// 构建邮件主题
	subject := fmt.Sprintf("[%s] %s", s.getNoticeTypeLabel(notice.NoticeType), notice.NoticeTitle)

	// 构建邮件内容
	htmlBody := s.buildNoticeHTMLBody(notice)

	msg := &EmailMessage{
		To:       userAddresses,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: notice.NoticeContent,
	}

	return s.Send(ctx, configID, msg)
}

// getNoticeTypeLabel 获取通知类型标签
func (s *EmailSenderService) getNoticeTypeLabel(noticeType string) string {
	switch noticeType {
	case "1":
		return "公告"
	case "2":
		return "警告"
	default:
		return "通知"
	}
}

// buildNoticeHTMLBody 构建通知邮件HTML内容
func (s *EmailSenderService) buildNoticeHTMLBody(notice *models.Notice) string {
	var html strings.Builder

	html.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #1890ff; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f5f5f5; }
        .footer { text-align: center; padding: 10px; color: #999; font-size: 12px; }
        .priority { padding: 5px 10px; border-radius: 3px; display: inline-block; }
        .priority-normal { background: #52c41a; color: white; }
        .priority-important { background: #faad14; color: white; }
        .priority-urgent { background: #f5222d; color: white; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>` + notice.NoticeTitle + `</h2>
        </div>
        <div class="content">
            <p><strong>类型：</strong>` + s.getNoticeTypeLabel(notice.NoticeType) + `</p>
            <p><strong>优先级：</strong><span class="priority ` + s.getPriorityClass(notice.Priority) + `">` + s.getPriorityLabel(notice.Priority) + `</span></p>
`)

	if notice.PublishTime != nil {
		html.WriteString("            <p><strong>发布时间：</strong>" + notice.PublishTime.Format("2006-01-02 15:04:05") + "</p>\n")
	}

	html.WriteString(`            <div style="margin-top: 20px;">
                <strong>内容：</strong>
                <div style="background: white; padding: 15px; border-radius: 5px; margin-top: 10px;">
`)

	if notice.IsMarkdown {
		// 简单的Markdown处理（生产环境建议使用专门的Markdown库）
		html.WriteString("                    <pre>" + notice.NoticeContent + "</pre>\n")
	} else {
		html.WriteString("                    <p>" + notice.NoticeContent + "</p>\n")
	}

	html.WriteString(`                </div>
            </div>
        </div>
        <div class="footer">
            <p>本邮件由系统自动发送，请勿回复。</p>
        </div>
    </div>
</body>
</html>`)

	return html.String()
}

// getPriorityClass 获取优先级样式类
func (s *EmailSenderService) getPriorityClass(priority models.NoticePriority) string {
	switch priority {
	case models.PriorityImportant:
		return "priority-important"
	case models.PriorityUrgent:
		return "priority-urgent"
	default:
		return "priority-normal"
	}
}

// getPriorityLabel 获取优先级标签
func (s *EmailSenderService) getPriorityLabel(priority models.NoticePriority) string {
	switch priority {
	case models.PriorityImportant:
		return "重要"
	case models.PriorityUrgent:
		return "紧急"
	default:
		return "普通"
	}
}

// TestEmailConfig 测试邮件配置
func (s *EmailSenderService) TestEmailConfig(ctx context.Context, configID string, testTo string) *SendResult {
	configService := NewNotificationConfigService(s.db)
	config, err := configService.GetEmailConfigByID(ctx, configID)
	if err != nil {
		return &SendResult{
			Success: false,
			Message: "获取邮件配置失败",
			Error:   err,
		}
	}

	// 解密密码
	password, err := DecryptPassword(config.Password, "")
	if err != nil {
		return &SendResult{
			Success: false,
			Message: "密码解密失败",
			Error:   err,
		}
	}

	// 构建测试邮件
	fromAddress := config.Username
	if config.FromEmail != "" {
		fromAddress = config.FromEmail
	}

	msg := &EmailMessage{
		To:      []string{testTo},
		Subject: "测试邮件",
		TextBody: "这是一封测试邮件。\n\n" +
			"配置名称: " + config.ConfigName + "\n" +
			"SMTP服务器: " + config.Host + "\n" +
			"端口: " + fmt.Sprintf("%d", config.Port) + "\n" +
			"发送时间: " + time.Now().Format("2006-01-02 15:04:05"),
	}

	err = s.sendEmail(
		config.Host,
		config.Port,
		config.Username,
		password,
		fromAddress,
		config.FromName,
		msg,
		config.UseSSL,
		config.UseSTARTTLS,
	)

	if err != nil {
		return &SendResult{
			Success: false,
			Message: "测试邮件发送失败",
			Error:   err,
		}
	}

	return &SendResult{
		Success: true,
		Message: "测试邮件发送成功",
	}
}
