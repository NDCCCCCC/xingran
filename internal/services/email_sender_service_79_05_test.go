package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// -------------------------------------------------------------------------
// 79-05 Task 6: email_sender_service.go(189 stmts,基线 14.3%)
//
// D-79-03 口径:
//   - plain fake SMTP(127.0.0.1:0 + bufio 会话)驱动 sendPlainSMTP 全链 + Send/SendNoticeEmail
//   - TLS 双路径(sendWithTLS / sendWithSTARTTLS)只测 dial-error 分支:
//     InsecureSkipVerify:false 硬编码(email_sender_service.go:203-204 / :271-272)
//     → 自签 fake 不可达,TLS 握手 happy path 不覆盖(SUMMARY 记录),禁改生产配置
//   - 纯 builder 全表驱动(buildEmailContent/notice 三函数/plainAuth)
//
// race 纪律:listener/goroutine 全部 t.Cleanup 关闭与等待,禁 t.Parallel。
// -------------------------------------------------------------------------

// newEml7905 sqlite(t.TempDir 文件库)+ EmailSenderService。
func newEml7905(t *testing.T) (*EmailSenderService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(strings.ReplaceAll(t.TempDir(), `\`, "/")+"/eml7905.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "open sqlite temp db")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.AutoMigrate(&models.EmailConfig{}), "auto migrate email config")
	return NewEmailSenderService(db), db
}

// eml7905SeedConfig 落一条 EmailConfig(password 经 EncryptPassword 默认 key 加密)。
func eml7905SeedConfig(t *testing.T, db *gorm.DB, host string, port int, useSSL, useSTARTTLS, isDefault bool, status int) *models.EmailConfig {
	t.Helper()
	cipher, err := EncryptPassword("testpass7905", "")
	require.NoError(t, err, "seed 加密口令")
	cfg := &models.EmailConfig{
		ID:          "eml7905-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		ConfigName:  "eml7905-config",
		Host:        host,
		Port:        port,
		Username:    "sender7905@test.local",
		Password:    cipher,
		FromName:    "XingRan 7905",
		FromEmail:   "noreply7905@test.local",
		UseSSL:      useSSL,
		UseSTARTTLS: useSTARTTLS,
		IsDefault:   isDefault,
		Status:      status,
		DelFlag:     0,
	}
	require.NoError(t, db.Create(cfg).Error)
	// QUIRK-79-04-D 同根:UseSSL/UseSTARTTLS 的 false 被列 default:true 的零值跳过吞掉,
	// 建后强制回写布尔列(测试侧规避,零生产改动)。
	require.NoError(t, db.Model(cfg).Updates(map[string]interface{}{
		"use_ssl":       useSSL,
		"use_start_tls": useSTARTTLS,
	}).Error)
	return cfg
}

// -------------------------------------------------------------------------
// plain fake SMTP(文件内 ~90 行,自包含;T-79-05-01: 一律 127.0.0.1:0)
// -------------------------------------------------------------------------

// smtpConversation7905 记录 fake SMTP 收到的会话内容(断言面)。
type smtpConversation7905 struct {
	mu       sync.Mutex
	MailFrom string
	RcptTo   []string
	Data     []byte
	Commands []string
}

func (c *smtpConversation7905) snapshot() smtpConversation7905 {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := smtpConversation7905{MailFrom: c.MailFrom, RcptTo: append([]string(nil), c.RcptTo...), Data: append([]byte(nil), c.Data...)}
	cp.Commands = append(cp.Commands, c.Commands...)
	return cp
}

// fakeSMTPServer7905 可编程响应的极简 SMTP 服务(单连接顺序处理)。
type fakeSMTPServer7905 struct {
	listener       net.Listener
	seen           *smtpConversation7905
	wg             sync.WaitGroup
	mu             sync.Mutex
	rejectMailFrom bool
	dropAfterCmd   int // 在第 N 条命令后硬断连接(0 = 不断)
}

func (s *fakeSMTPServer7905) addr() string { return s.listener.Addr().String() }

func (s *fakeSMTPServer7905) setReject(v bool) { s.mu.Lock(); s.rejectMailFrom = v; s.mu.Unlock() }
func (s *fakeSMTPServer7905) setDropAfter(n int) {
	s.mu.Lock()
	s.dropAfterCmd = n
	s.mu.Unlock()
}

// startFakeSMTP7905 启动 fake SMTP(t.Cleanup 关 listener 并等待 goroutine 退出)。
func startFakeSMTP7905(t *testing.T) *fakeSMTPServer7905 {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "listen 127.0.0.1:0")

	srv := &fakeSMTPServer7905{listener: ln, seen: &smtpConversation7905{}}
	srv.wg.Add(1)
	go srv.serve()
	t.Cleanup(func() {
		_ = ln.Close()
		srv.wg.Wait()
	})
	return srv
}

func (s *fakeSMTPServer7905) serve() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.handle(conn)
	}
}

func (s *fakeSMTPServer7905) handle(conn net.Conn) {
	defer func() {
		_ = conn.Close()
		s.wg.Done()
	}()
	s.wg.Add(1)

	s.mu.Lock()
	reject := s.rejectMailFrom
	dropAfter := s.dropAfterCmd
	s.mu.Unlock()

	cmdCount := 0
	write := func(line string) {
		_, _ = conn.Write([]byte(line + "\r\n"))
	}
	write("220 fake7905 ESMTP ready")

	reader := bufio.NewReader(conn)
	inData := false
	var data bytes.Buffer
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return // 客户端断开
		}
		trimmed := strings.TrimRight(line, "\r\n")

		if inData {
			if trimmed == "." {
				s.seen.mu.Lock()
				s.seen.Data = append([]byte(nil), data.Bytes()...)
				s.seen.mu.Unlock()
				inData = false
				write("250 OK: queued as 7905")
			} else {
				data.WriteString(line)
			}
			continue
		}

		upper := strings.ToUpper(trimmed)
		cmdCount++
		s.seen.mu.Lock()
		s.seen.Commands = append(s.seen.Commands, trimmed)
		s.seen.mu.Unlock()

		if dropAfter > 0 && cmdCount > dropAfter {
			return // 硬断连接 → 客户端 IO 错误
		}

		switch {
		case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
			if strings.HasPrefix(upper, "EHLO") {
				write("250-fake7905 greets you")
				write("250 AUTH PLAIN LOGIN")
			} else {
				write("250 fake7905")
			}
		case strings.HasPrefix(upper, "AUTH"):
			write("235 2.7.0 Authentication successful")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			if reject {
				write("550 5.7.1 rejected by fake7905")
				continue
			}
			s.seen.mu.Lock()
			s.seen.MailFrom = trimmed
			s.seen.mu.Unlock()
			write("250 2.1.0 OK")
		case strings.HasPrefix(upper, "RCPT TO:"):
			s.seen.mu.Lock()
			s.seen.RcptTo = append(s.seen.RcptTo, trimmed)
			s.seen.mu.Unlock()
			write("250 2.1.5 OK")
		case strings.HasPrefix(upper, "DATA"):
			write("354 End data with <CR><LF>.<CR><LF>")
			inData = true
		case strings.HasPrefix(upper, "QUIT"):
			write("221 2.0.0 bye")
			return
		case strings.HasPrefix(upper, "RSET"), strings.HasPrefix(upper, "NOOP"):
			write("250 OK")
		default:
			write("250 OK")
		}
	}
}

// -------------------------------------------------------------------------
// sendPlainSMTP 全链
// -------------------------------------------------------------------------

// TestEml7905_SendPlainSMTP_Happy fake SMTP 全链:认证/发件人/收件人/DATA 内容断言。
func TestEml7905_SendPlainSMTP_Happy(t *testing.T) {
	svc, _ := newEml7905(t)
	srv := startFakeSMTP7905(t)

	content, err := svc.buildEmailContent("XingRan 7905 <noreply7905@test.local>", &EmailMessage{
		To:       []string{"rcpt7905@test.local"},
		Subject:  "plain7905",
		TextBody: "你好 7905",
	})
	require.NoError(t, err)

	require.NoError(t, svc.sendPlainSMTP(srv.addr(), "sender7905", "testpass7905", "noreply7905@test.local",
		[]string{"rcpt7905@test.local"}, content))

	seen := srv.seen.snapshot()
	assert.Contains(t, seen.MailFrom, "noreply7905@test.local")
	require.Len(t, seen.RcptTo, 1)
	assert.Contains(t, seen.RcptTo[0], "rcpt7905@test.local")
	assert.Contains(t, string(seen.Data), "From: XingRan 7905 <noreply7905@test.local>")
	assert.Contains(t, string(seen.Data), "To: rcpt7905@test.local")
	assert.Contains(t, string(seen.Data), "Subject: plain7905")
	assert.Contains(t, string(seen.Data), "MIME-Version: 1.0")

	cmds := seen.Commands
	require.GreaterOrEqual(t, len(cmds), 5)
	upper := make([]string, 0, len(cmds))
	for _, c := range cmds {
		upper = append(upper, strings.ToUpper(c))
	}
	assert.Contains(t, upper[0], "EHLO", "首条命令应为 EHLO(hello 懒发)")
	assert.Contains(t, upper[1], "AUTH PLAIN", "第二条应为 AUTH PLAIN(自定义 plainAuth)")
	assert.Contains(t, upper[2], "MAIL FROM:")
	assert.Contains(t, upper[3], "RCPT TO:")
	assert.Equal(t, "DATA", upper[4], "最后一条命令应为 DATA")
}

// TestEml7905_SendPlainSMTP_ServerReject MAIL FROM 550 拒绝 + 中途断连两个错误分支。
func TestEml7905_SendPlainSMTP_ServerReject(t *testing.T) {
	svc, _ := newEml7905(t)
	content, err := svc.buildEmailContent("noreply7905@test.local", &EmailMessage{
		To: []string{"rcpt7905@test.local"}, Subject: "rej", TextBody: "x",
	})
	require.NoError(t, err)

	t.Run("mail_from_rejected_550", func(t *testing.T) {
		srv := startFakeSMTP7905(t)
		srv.setReject(true)
		err := svc.sendPlainSMTP(srv.addr(), "u", "p", "noreply7905@test.local", []string{"rcpt7905@test.local"}, content)
		require.Error(t, err, "550 拒绝应上抛")
		assert.Contains(t, err.Error(), "550")
	})

	t.Run("connection_dropped_mid_session", func(t *testing.T) {
		srv := startFakeSMTP7905(t)
		srv.setDropAfter(1) // EHLO 后即硬断
		err := svc.sendPlainSMTP(srv.addr(), "u", "p", "noreply7905@test.local", []string{"rcpt7905@test.local"}, content)
		require.Error(t, err, "中途断连应产生 IO 错误")
	})

	t.Run("dial_refused", func(t *testing.T) {
		// 指向已关闭端口(监听后立即关闭)
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := ln.Addr().String()
		require.NoError(t, ln.Close())
		err = svc.sendPlainSMTP(addr, "u", "p", "noreply7905@test.local", []string{"rcpt7905@test.local"}, content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "连接失败")
	})
}

// -------------------------------------------------------------------------
// Send / SendWithDefaultConfig / SendNoticeEmail / TestEmailConfig
// -------------------------------------------------------------------------

// TestEml7905_Send_HappyViaPlain 纯 SMTP 配置(UseSSL=false, UseSTARTTLS=false)全链成功。
func TestEml7905_Send_HappyViaPlain(t *testing.T) {
	svc, db := newEml7905(t)
	srv := startFakeSMTP7905(t)
	cfg := eml7905SeedConfig(t, db, "127.0.0.1", port7905(srv), false, false, true, int(models.NotificationConfigStatusNormal))

	result := svc.Send(context.Background(), cfg.ID, &EmailMessage{
		To:       []string{"rcpt7905@test.local"},
		Subject:  "send7905",
		HTMLBody: "<b>hello</b>",
		TextBody: "hello",
	})
	require.NotNil(t, result)
	assert.True(t, result.Success, "message=%v err=%v", result.Message, result.Error)
	assert.Equal(t, "邮件发送成功", result.Message)
	assert.NoError(t, result.Error)

	seen := srv.seen.snapshot()
	assert.NotEmpty(t, seen.Data, "fake SMTP 应收到完整 DATA")
	assert.Contains(t, string(seen.Data), "Subject: send7905")
}

// eml7905Decoded 把 DATA 内所有 base64 token 解码拼接(HTML/文本体均为 base64 传输)。
func eml7905Decoded(t *testing.T, data []byte) string {
	t.Helper()
	var out strings.Builder
	for _, tok := range strings.Fields(string(data)) {
		if len(tok) < 8 || strings.ContainsAny(tok, ":<>()/,") {
			continue
		}
		if dec, err := base64.StdEncoding.DecodeString(tok); err == nil {
			out.Write(dec)
			out.WriteString("\n")
		}
	}
	return out.String()
}

// port7905 从 fake 服务地址取端口。
func port7905(srv *fakeSMTPServer7905) int {
	addr := srv.addr()
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return 0
	}
	var port int
	if _, err := fmt.Sscanf(addr[idx+1:], "%d", &port); err != nil {
		return 0
	}
	return port
}

// TestEml7905_Send_ConfigMissingAndDisabled 配置不存在 / 停用 两分支。
func TestEml7905_Send_ConfigMissingAndDisabled(t *testing.T) {
	ctx := context.Background()
	svc, db := newEml7905(t)

	t.Run("config_missing", func(t *testing.T) {
		result := svc.Send(ctx, "no-such-config-7905", &EmailMessage{To: []string{"a@b.local"}})
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Equal(t, "获取邮件配置失败", result.Message)
		require.Error(t, result.Error)
	})

	t.Run("config_disabled", func(t *testing.T) {
		srv := startFakeSMTP7905(t)
		cfg := eml7905SeedConfig(t, db, "127.0.0.1", port7905(srv), false, false, false, int(models.NotificationConfigStatusStopped))
		result := svc.Send(ctx, cfg.ID, &EmailMessage{To: []string{"a@b.local"}})
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Equal(t, "邮件配置未启用", result.Message)
		assert.Empty(t, srv.seen.snapshot().Data, "停用配置不应触达 SMTP")
	})

	t.Run("empty_recipients", func(t *testing.T) {
		srv := startFakeSMTP7905(t)
		cfg := eml7905SeedConfig(t, db, "127.0.0.1", port7905(srv), false, false, false, int(models.NotificationConfigStatusNormal))
		result := svc.Send(ctx, cfg.ID, &EmailMessage{})
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Equal(t, "收件人不能为空", result.Message)
	})

	t.Run("invalid_recipient", func(t *testing.T) {
		srv := startFakeSMTP7905(t)
		cfg := eml7905SeedConfig(t, db, "127.0.0.1", port7905(srv), false, false, false, int(models.NotificationConfigStatusNormal))
		result := svc.Send(ctx, cfg.ID, &EmailMessage{To: []string{"not-an-address"}})
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Contains(t, result.Message, "无效的收件人地址")
	})

	t.Run("bad_cipher_password", func(t *testing.T) {
		srv := startFakeSMTP7905(t)
		cfg := eml7905SeedConfig(t, db, "127.0.0.1", port7905(srv), false, false, false, int(models.NotificationConfigStatusNormal))
		require.NoError(t, db.Model(&models.EmailConfig{}).Where("id = ?", cfg.ID).Update("password", "not-base64!!").Error)
		result := svc.Send(ctx, cfg.ID, &EmailMessage{To: []string{"a@b.local"}})
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Equal(t, "密码解密失败", result.Message)
	})
}

// TestEml7905_SendWithDefaultConfig 默认配置链(:149-161)。
func TestEml7905_SendWithDefaultConfig(t *testing.T) {
	ctx := context.Background()
	svc, db := newEml7905(t)

	t.Run("no_config_fails", func(t *testing.T) {
		result := svc.SendWithDefaultConfig(ctx, &EmailMessage{To: []string{"a@b.local"}})
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Equal(t, "未设置默认邮件配置", result.Message)
	})

	t.Run("with_default_config_sends", func(t *testing.T) {
		srv := startFakeSMTP7905(t)
		eml7905SeedConfig(t, db, "127.0.0.1", port7905(srv), false, false, true, int(models.NotificationConfigStatusNormal))
		result := svc.SendWithDefaultConfig(ctx, &EmailMessage{
			To: []string{"rcpt7905@test.local"}, Subject: "default7905", TextBody: "body",
		})
		require.NotNil(t, result)
		assert.True(t, result.Success, "message=%v err=%v", result.Message, result.Error)
		assert.Contains(t, string(srv.seen.snapshot().Data), "Subject: default7905")
	})
}

// TestEml7905_SendWithTLS_DialError TLS 双路径 dial-error(D-79-03)。
//
// InsecureSkipVerify:false 硬编码(email_sender_service.go:203-204 / :271-272)
// → 自签 fake 不可达,TLS 握手 happy path 不覆盖(SUMMARY 记录);禁改生产 TLS 配置。
func TestEml7905_SendWithTLS_DialError(t *testing.T) {
	svc, _ := newEml7905(t)
	content, err := svc.buildEmailContent("noreply7905@test.local", &EmailMessage{
		To: []string{"rcpt7905@test.local"}, Subject: "tls", TextBody: "x",
	})
	require.NoError(t, err)

	// 已关闭端口 → tls.Dial 失败 → dial-error 分支
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	deadAddr := ln.Addr().String()
	require.NoError(t, ln.Close())

	t.Run("sendWithTLS_dial_error", func(t *testing.T) {
		// InsecureSkipVerify:false 硬编码(:203-204)→ 自签 fake 不可达,TLS happy path 不覆盖
		err := svc.sendWithTLS(deadAddr, "u", "p", "noreply7905@test.local", []string{"rcpt7905@test.local"}, content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "TLS连接失败")
	})

	t.Run("sendWithSTARTTLS_dial_error", func(t *testing.T) {
		// InsecureSkipVerify:false 硬编码(:271-272)→ 自签 fake 不可达,STARTTLS happy path 不覆盖
		err := svc.sendWithSTARTTLS(deadAddr, "u", "p", "noreply7905@test.local", []string{"rcpt7905@test.local"}, content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "连接失败")
	})
}

// -------------------------------------------------------------------------
// 纯 builder
// -------------------------------------------------------------------------

// TestEml7905_BuildEmailContent buildEmailContent(:362-405)三形态。
func TestEml7905_BuildEmailContent(t *testing.T) {
	svc, _ := newEml7905(t)

	t.Run("text_only", func(t *testing.T) {
		content, err := svc.buildEmailContent("from7905@test.local", &EmailMessage{
			To: []string{"a@b.local", "c@d.local"}, Subject: "t7905", TextBody: "plain-body",
		})
		require.NoError(t, err)
		text := string(content)
		assert.Contains(t, text, "From: from7905@test.local\r\n")
		assert.Contains(t, text, "To: a@b.local,c@d.local\r\n")
		assert.Contains(t, text, "Subject: t7905\r\n")
		assert.Contains(t, text, "Content-Type: text/plain; charset=utf-8\r\n")
		assert.NotContains(t, text, "multipart/alternative")
		// 纯文本体经 base64 编码
		assert.NotContains(t, text, "plain-body")
		assert.Contains(t, text, "cGxhaW4tYm9keQ==\r\n")
	})

	t.Run("html_with_text_multipart", func(t *testing.T) {
		content, err := svc.buildEmailContent("from7905@test.local", &EmailMessage{
			To: []string{"a@b.local"}, Subject: "m7905", HTMLBody: "<p>hi</p>", TextBody: "hi",
		})
		require.NoError(t, err)
		text := string(content)
		assert.Contains(t, text, "multipart/alternative")
		assert.Contains(t, text, `boundary="----=_NextPart_1234567890"`)
		assert.Contains(t, text, "Content-Type: text/plain; charset=utf-8")
		assert.Contains(t, text, "Content-Type: text/html; charset=utf-8")
		assert.Contains(t, text, "------=_NextPart_1234567890--")
	})

	t.Run("html_only_skips_text_part", func(t *testing.T) {
		content, err := svc.buildEmailContent("from7905@test.local", &EmailMessage{
			To: []string{"a@b.local"}, Subject: "h7905", HTMLBody: "<p>hi</p>",
		})
		require.NoError(t, err)
		text := string(content)
		assert.Contains(t, text, "multipart/alternative")
		assert.Equal(t, 1, strings.Count(text, "Content-Type: text/html"), "HTML 段唯一")
	})

	t.Run("cc_header", func(t *testing.T) {
		content, err := svc.buildEmailContent("from7905@test.local", &EmailMessage{
			To: []string{"a@b.local"}, Cc: []string{"cc@b.local"}, Subject: "c7905", TextBody: "x",
		})
		require.NoError(t, err)
		assert.Contains(t, string(content), "Cc: cc@b.local\r\n")
	})
}

// TestEml7905_PlainAuth newPlainAuth/plainAuth.Start/Next(:19-37)。
func TestEml7905_PlainAuth(t *testing.T) {
	auth := newPlainAuth("", "user7905", "pass7905", "test.local")
	require.NotNil(t, auth)

	identity, resp, err := auth.Start(nil)
	require.NoError(t, err)
	assert.Equal(t, "PLAIN", identity)
	// \x00 + username + \x00 + password
	assert.Equal(t, []byte("\x00user7905\x00pass7905"), resp)

	// more=false → nil,nil
	next, err := auth.Next([]byte("ignored"), false)
	require.NoError(t, err)
	assert.Nil(t, next)

	// more=true → unexpected server challenge
	next, err = auth.Next([]byte("challenge"), true)
	require.Error(t, err)
	assert.Nil(t, next)
	assert.Contains(t, err.Error(), "unexpected server challenge")
}

// TestEml7905_NoticeBuilders 通知三函数 + HTML 构建器(:426-518)。
func TestEml7905_NoticeBuilders(t *testing.T) {
	svc, _ := newEml7905(t)

	t.Run("notice_type_label", func(t *testing.T) {
		assert.Equal(t, "公告", svc.getNoticeTypeLabel("1"))
		assert.Equal(t, "警告", svc.getNoticeTypeLabel("2"))
		assert.Equal(t, "通知", svc.getNoticeTypeLabel("3"))
		assert.Equal(t, "通知", svc.getNoticeTypeLabel(""))
	})

	t.Run("priority_class_and_label", func(t *testing.T) {
		cases := []struct {
			priority models.NoticePriority
			class    string
			label    string
		}{
			{models.PriorityImportant, "priority-important", "重要"},
			{models.PriorityUrgent, "priority-urgent", "紧急"},
			{models.PriorityNormal, "priority-normal", "普通"},
		}
		for _, tc := range cases {
			assert.Equal(t, tc.class, svc.getPriorityClass(tc.priority))
			assert.Equal(t, tc.label, svc.getPriorityLabel(tc.priority))
		}
	})

	t.Run("html_body_contains_title_type_priority_content", func(t *testing.T) {
		publishAt := mhq7905Time(10, 8, 30)
		notice := &models.Notice{
			NoticeTitle:   "标题7905",
			NoticeType:    "2",
			NoticeContent: "内容7905",
			Priority:      models.PriorityUrgent,
			PublishTime:   &publishAt,
		}
		html := svc.buildNoticeHTMLBody(notice)
		assert.Contains(t, html, "标题7905")
		assert.Contains(t, html, "警告")
		assert.Contains(t, html, "priority-urgent")
		assert.Contains(t, html, "紧急")
		assert.Contains(t, html, "内容7905")
		assert.Contains(t, html, publishAt.Format("2006-01-02 15:04:05"), "含发布时间")
		assert.Contains(t, html, "本邮件由系统自动发送")
	})

	t.Run("markdown_and_no_publish_time", func(t *testing.T) {
		notice := &models.Notice{
			NoticeTitle:   "md7905",
			NoticeType:    "1",
			NoticeContent: "# md 内容",
			Priority:      models.PriorityNormal,
			IsMarkdown:    true,
		}
		html := svc.buildNoticeHTMLBody(notice)
		assert.Contains(t, html, "<pre># md 内容</pre>", "markdown 走 pre 块")
		assert.NotContains(t, html, "发布时间", "PublishTime 为 nil 不输出发布时间")
	})
}

// TestEml7905_SendNoticeEmail_And_TestEmailConfig 通知邮件与配置测试链(:408-425 / :521-582)。
func TestEml7905_SendNoticeEmail_And_TestEmailConfig(t *testing.T) {
	ctx := context.Background()
	svc, db := newEml7905(t)
	srv := startFakeSMTP7905(t)
	cfg := eml7905SeedConfig(t, db, "127.0.0.1", port7905(srv), false, false, true, int(models.NotificationConfigStatusNormal))

	t.Run("send_notice_email", func(t *testing.T) {
		notice := &models.Notice{
			NoticeTitle:   "通知7905",
			NoticeType:    "1",
			NoticeContent: "通知内容7905",
			Priority:      models.PriorityImportant,
		}
		result := svc.SendNoticeEmail(ctx, cfg.ID, notice, []string{"rcpt7905@test.local"})
		require.NotNil(t, result)
		assert.True(t, result.Success, "message=%v err=%v", result.Message, result.Error)

		data := srv.seen.snapshot().Data
		assert.Contains(t, string(data), "Subject: [公告] 通知7905")
		// HTML 体经 base64 编码 → 解码后断言优先级样式
		assert.Contains(t, eml7905Decoded(t, data), "priority-important")
	})

	t.Run("test_email_config", func(t *testing.T) {
		result := svc.TestEmailConfig(ctx, cfg.ID, "rcpt7905@test.local")
		require.NotNil(t, result)
		assert.True(t, result.Success, "message=%v err=%v", result.Message, result.Error)
		assert.Equal(t, "测试邮件发送成功", result.Message)
		assert.Contains(t, string(srv.seen.snapshot().Data), "Subject: 测试邮件")
	})

	t.Run("test_email_config_missing", func(t *testing.T) {
		result := svc.TestEmailConfig(ctx, "nope-7905", "rcpt7905@test.local")
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Equal(t, "获取邮件配置失败", result.Message)
	})

	t.Run("send_notice_no_recipients", func(t *testing.T) {
		notice := &models.Notice{NoticeTitle: "x", NoticeType: "1", NoticeContent: "y"}
		result := svc.SendNoticeEmail(ctx, cfg.ID, notice, nil)
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.Equal(t, "收件人不能为空", result.Message)
	})
}

// TestEml7905_SendEmail_DispatchMatrix sendEmail(:164-198)三路派发矩阵。
//
// useSTARTTLS=true 且服务端未通告 STARTTLS 扩展时,sendWithSTARTTLS 的
// 扩展检查(:270)直接跳过 StartTLS,后续认证/发件人/收件人/DATA 全链在明文连接上
// 完成 → 这是 STARTTLS 路径可达的 happy 段(握手段仍受 D-79-03 限制)。
func TestEml7905_SendEmail_DispatchMatrix(t *testing.T) {
	svc, _ := newEml7905(t)

	t.Run("starttls_without_extension_plain_fallback", func(t *testing.T) {
		srv := startFakeSMTP7905(t)
		host, portStr, err := net.SplitHostPort(srv.addr())
		require.NoError(t, err)
		var port int
		_, err = fmt.Sscanf(portStr, "%d", &port)
		require.NoError(t, err)

		require.NoError(t, svc.sendEmail(host, port, "sender7905", "testpass7905",
			"noreply7905@test.local", "XingRan 7905", &EmailMessage{
				To: []string{"rcpt7905@test.local"}, Subject: "starttls7905", TextBody: "x",
			}, false, true))

		seen := srv.seen.snapshot()
		assert.Contains(t, string(seen.Data), "Subject: starttls7905", "未通告 STARTTLS → 明文会话完成全链")
		assert.NotContains(t, strings.Join(seen.Commands, "|"), "STARTTLS", "服务端未通告则不发 STARTTLS")
	})

	t.Run("port25_skips_starttls_goes_plain", func(t *testing.T) {
		// useSTARTTLS && port == 25 → 不走 STARTTLS,直接纯 SMTP(此处端口无监听 → 连接失败)
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		deadHost, deadPortStr, err := net.SplitHostPort(ln.Addr().String())
		require.NoError(t, err)
		require.NoError(t, ln.Close())

		err = svc.sendEmail(deadHost, 25, "sender7905", "testpass7905",
			"noreply7905@test.local", "", &EmailMessage{To: []string{"rcpt7905@test.local"}},
			false, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "连接失败")
		_ = deadPortStr
	})

	t.Run("ssl_dispatch_to_tls_path", func(t *testing.T) {
		// useSSL=true → sendWithTLS;对明文 fake 握手必然失败(:203-204 InsecureSkipVerify:false 锁定)
		srv := startFakeSMTP7905(t)
		host, portStr, err := net.SplitHostPort(srv.addr())
		require.NoError(t, err)
		var port int
		_, err = fmt.Sscanf(portStr, "%d", &port)
		require.NoError(t, err)

		err = svc.sendEmail(host, port, "sender7905", "testpass7905",
			"noreply7905@test.local", "", &EmailMessage{To: []string{"rcpt7905@test.local"}},
			true, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "TLS连接失败")
	})
}
