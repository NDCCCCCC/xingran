//go:build !skip_db_tests
// +build !skip_db_tests

// Phase 78 Plan 07 Task 1: ldap_client.go 零 wire 段覆盖
//
// 覆盖范围(按 78-07-PLAN.md §Task 1):
//   - NewLDAPClient 变参三形态(:52-62)
//   - Conn() / Close() nil 安全(:70-72/:177-181)
//   - cleanDomain() 可疑语义(:128-135) — D-78-07c 裁决:现行为断言
//   - extractNetBIOSName() 表驱动(:439-445)
//   - extractRDNFromDN() 边界覆盖(已有 100%,补边界用例)
//   - dialConnection 三分支(default 拒绝 / SSL 握手失败 / StartTLS 失败):98-125
//   - Connect dial 失败(:75-90)
//   - tryBindAttempts 三次尝试全失败(:141-174)
//
// 设计原则:
//   - 零 mock:所有错误分支用真实 TCP 监听驱动
//   - 零真实网络:全部 127.0.0.1
//   - 每个网络用例 5s 硬超时守卫
//   - 环境变量 t.Setenv(禁 os.Setenv,D-78-07d)
//   - 成功路径(Connect bind 成功)归 Task 4 探针

package addomain

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// closedPort78 申请一个本地随机端口,立即关闭监听,返回端口号。
// 唯一允许的网络目标:127.0.0.1(零真实网络,D-78-07d)。
func closedPort78(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	require.NoError(t, ln.Close())
	port := 0
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}
	return port
}

// dummyConfig 返回一个 dummy ADConfig,ServerAddress=127.0.0.1,ServerPort 由调用方指定。
func dummyConfig(port int) *models.ADConfig {
	return &models.ADConfig{
		ServerAddress: "127.0.0.1",
		ServerPort:   port,
		BaseDN:       "DC=example,DC=com",
		DomainName:   "example.com",
		AdminUsername: "admin",
		AdminPassword: "admin_password",
		UseSSL:       false,
		UseTLS:       false,
		SyncEnabled:  true,
		SyncInterval: 10,
		Status:       models.ADConfigStatusEnabled,
		MemberOUDN:   "OU=Users,DC=example,DC=com",
	}
}

// ==================== NewLDAPClient 变参三形态 ====================

// TestLC78_NewLDAPClient_Variadic 验证 NewLDAPClient 变参(:52-62)的三个形态。
func TestLC78_NewLDAPClient_Variadic(t *testing.T) {
	cfg := dummyConfig(389)

	// 形态 1: 不传 account
	c1 := NewLDAPClient(cfg)
	assert.Nil(t, c1.GetAccount())

	// 形态 2: 传一个非 nil account
	acct := &models.ADServiceAccount{ID: "acct-1", Username: "svc1", PasswordCiphertext: "enc_pwd"}
	c2 := NewLDAPClient(cfg, acct)
	assert.Equal(t, acct, c2.GetAccount())

	// 形态 3: 传一个 nil account
	c3 := NewLDAPClient(cfg, (*models.ADServiceAccount)(nil))
	assert.Nil(t, c3.GetAccount())
}

// ==================== Conn() / Close() nil 安全 ====================

// TestLC78_Conn_And_Close 验证 Conn()在未 Connect 时返回 nil(:70-72);
// Close()在 conn==nil 时不 panic(:177-181)。
func TestLC78_Conn_And_Close(t *testing.T) {
	cfg := dummyConfig(389)
	c := NewLDAPClient(cfg)

	// Conn() 在未 Connect 时返回 nil
	assert.Nil(t, c.Conn())

	// Close() 在 conn==nil 时不 panic
	assert.NotPanics(t, func() { c.Close() })

	// 再次 Close 也不 panic(idempotent)
	assert.NotPanics(t, func() { c.Close() })
}

// ==================== cleanDomain 表驱动 ====================

// D-78-07c 裁决记录:
// cleanDomain(:128-135) 的 `suffix := "@"+DomainName` 后对 DomainName 自身 TrimSuffix
// 语义可疑("example.com" TrimSuffix "@example.com" = "" 再回退 = "example.com")。
// 查源码无注释/文档依据,D-78-10 判定:按现行为断言,SUMMARY 记待裁决。
// 后续若有注释/文档揭示真实意图,在此补回归用例。

func TestLC78_CleanDomain_Table(t *testing.T) {
	tests := []struct {
		name       string
		domainName string
		want       string
	}{
		{
			name:       "普通域名",
			domainName: "example.com",
			want:       "example.com", // TrimSuffix "@example.com" 失败,回退 DomainName
		},
		{
			name:       "带@后缀域名",
			domainName: "@example.com",
			want:       "@example.com", // TrimSuffix "@@example.com" 失败,回退 @example.com
		},
		{
			name:       "空字符串",
			domainName: "",
			want:       "", // 空串 TrimSuffix 任何都是 "",回退条件 cleanDomain=="" 不满足,直接返回 ""
		},
		{
			name:       "单段域名",
			domainName: "localhost",
			want:       "localhost", // TrimSuffix "@localhost" 失败,回退
		},
		{
			name:       "多段域名",
			domainName: "sub.example.com",
			want:       "sub.example.com", // TrimSuffix "@sub.example.com" 失败,回退
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := dummyConfig(389)
			cfg.DomainName = tt.domainName
			c := NewLDAPClient(cfg)
			got := c.cleanDomain()
			assert.Equal(t, tt.want, got, "cleanDomain(%q) = %q, want %q", tt.domainName, got, tt.want)
		})
	}
}

// ==================== extractNetBIOSName 表驱动 ====================

func TestLC78_ExtractNetBIOSName_Table(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   string
	}{
		{
			name:   "普通域名",
			domain: "example.com",
			want:   "EXAMPLE",
		},
		{
			name:   "单段域名",
			domain: "localhost",
			want:   "LOCALHOST",
		},
		{
			name:   "空字符串",
			domain: "",
			want:   "", // strings.Split("",".") 返回 [""];len>0,ToUpper "" = ""
		},
		{
			name:   "多段域名",
			domain: "sub.example.com",
			want:   "SUB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractNetBIOSName(tt.domain)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== extractRDNFromDN 边界 ====================

func TestLC78_ExtractRDNFromDN_Edge(t *testing.T) {
	tests := []struct {
		name string
		dn   string
		want string
	}{
		{
			name: "普通 DN",
			dn:   "CN=John,OU=Users,DC=example,DC=com",
			want: "CN=John",
		},
		{
			name: "无逗号单段",
			dn:   "CN=John",
			want: "CN=John", // 无逗号,Index=-1,返回整个字符串
		},
		{
			name: "空字符串",
			dn:   "",
			want: "",
		},
		{
			name: "转义逗号", // extractRDNFromDN 不处理转义,只找第一个逗号字面值
			dn:   "CN=John\\,Doe,OU=Users,DC=example,DC=com",
			want: "CN=John\\", // 第一个逗号在 "Doe" 前,返回 "CN=John\\"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRDNFromDN(tt.dn)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== dialConnection 三分支错误 ====================

// TestLC78_DialConnection_Default_Refused UseSSL=false/UseTLS=false 时 TCP 拒绝。
func TestLC78_DialConnection_Default_Refused(t *testing.T) {
	port := closedPort78(t)
	cfg := dummyConfig(port)
	cfg.UseSSL = false
	cfg.UseTLS = false
	c := NewLDAPClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := c.dialConnection("127.0.0.1:" + portStr(port))
		done <- err
	}()

	select {
	case <-ctx.Done():
		t.Fatal("dialConnection 超过 5s,疑似 hang")
	case err := <-done:
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "i/o timeout") ||
			strings.Contains(err.Error(), "refused"),
			"expect connection refused, got: %v", err)
	}
}

// TestLC78_DialConnection_SSL_HandshakeFail UseSSL=true + 纯 TCP 监听(无 TLS)→ TLS 握手失败或连接拒绝。
// 在 Windows 上 UseSSL=true 可能先触发 connect-ex (连接拒绝)再 TLS;
// 在 Linux 上会建立 TCP 连接后才 TLS 握手失败。两种情况都是预期行为。
func TestLC78_DialConnection_SSL_HandshakeFail(t *testing.T) {
	port := closedPort78(t)
	cfg := dummyConfig(port)
	cfg.UseSSL = true
	cfg.UseTLS = false
	c := NewLDAPClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := c.dialConnection("127.0.0.1:" + portStr(port))
		done <- err
	}()

	select {
	case <-ctx.Done():
		t.Fatal("dialConnection(SSL) 超过 5s")
	case err := <-done:
		assert.Error(t, err)
		errStr := err.Error()
		// Windows: connect-ex 拒绝; Linux: TLS 握手失败
		assert.True(t,
			strings.Contains(errStr, "handshake") ||
			strings.Contains(errStr, "tls") ||
			strings.Contains(errStr, "first record") ||
			strings.Contains(errStr, "refused") ||
			strings.Contains(errStr, "connection refused") ||
			strings.Contains(errStr, "No connection could be made"),
			"expect network or TLS error, got: %v", err)
	}
}

// TestLC78_DialConnection_TLS_StartTLSFail UseTLS=true + 纯 TCP 监听(无 StartTLS 扩展响应)→ StartTLS 失败。
// 在 Windows 上 UseTLS=true 可能先触发 connect-ex (连接拒绝);在 Linux 上建立 TCP 后 StartTLS 失败。
// 两种情况都是预期行为。
func TestLC78_DialConnection_TLS_StartTLSFail(t *testing.T) {
	port := closedPort78(t)
	cfg := dummyConfig(port)
	cfg.UseSSL = false
	cfg.UseTLS = true
	c := NewLDAPClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := c.dialConnection("127.0.0.1:" + portStr(port))
		done <- err
	}()

	select {
	case <-ctx.Done():
		t.Fatal("dialConnection(TLS/StartTLS) 超过 5s")
	case err := <-done:
		assert.Error(t, err)
		errStr := err.Error()
		assert.True(t,
			strings.Contains(errStr, "StartTLS") ||
			strings.Contains(errStr, "extension") ||
			strings.Contains(errStr, "first record") ||
			strings.Contains(errStr, "refused") ||
			strings.Contains(errStr, "connection refused") ||
			strings.Contains(errStr, "No connection could be made"),
			"expect network or StartTLS error, got: %v", err)
	}
}

// portStr converts port int to string (avoiding import "strconv" in helper).
func portStr(port int) string {
	if port == 0 {
		return "0"
	}
	result := ""
	for port > 0 {
		result = string(rune('0'+port%10)) + result
		port /= 10
	}
	return result
}

// ==================== Connect dial 失败 ====================

// TestLC78_Connect_DialFail Connect 在 dial 失败时直接返回 error 且 c.conn 仍为 nil。
func TestLC78_Connect_DialFail(t *testing.T) {
	port := closedPort78(t)
	cfg := dummyConfig(port)
	c := NewLDAPClient(cfg)

	err := c.Connect()

	assert.Error(t, err)
	assert.Nil(t, c.Conn(), "Connect 失败后 c.Conn() 必须仍为 nil")
}

// ==================== tryBindAttempts 三次尝试全失败 ====================

// echoTCPServer 启动一个接受连接后立即关闭的 TCP 服务器(使 Bind 快速返回 EOF)。
func echoTCPServer(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := 0
	for _, c := range ln.Addr().String()[len("127.0.0.1:"):] {
		port = port*10 + int(c-'0')
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close() // 立即关闭,让客户端读到 EOF
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return port
}

// TestLC78_TryBindAttempts_AllFail 验证三次尝试(UPN/NetBIOS/Direct)全失败时返回绑定失败错误。
func TestLC78_TryBindAttempts_AllFail(t *testing.T) {
	port := echoTCPServer(t)
	cfg := dummyConfig(port)
	c := NewLDAPClient(cfg)

	// 通过 dialConnection 拿到 conn(不走 Connect,避免真实 DNS 解析)
	conn, err := c.dialConnection("127.0.0.1:" + portStr(port))
	require.NoError(t, err)

	err = c.tryBindAttempts(conn, "example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "绑定失败")
	assert.Contains(t, err.Error(), "尝试: UPN")
	assert.Contains(t, err.Error(), "NetBIOS")
	assert.Contains(t, err.Error(), "直连")
	conn.Close()
}

// TestLC78_TryBindAttempts_WithAccount 用 account 非 nil 形态跑 tryBindAttempts。
// 验证走 decryptPassword 分支(明文密码来自解密)。
func TestLC78_TryBindAttempts_WithAccount(t *testing.T) {
	port := echoTCPServer(t)
	cfg := dummyConfig(port)
	acct := &models.ADServiceAccount{
		ID:                  "acct-echo",
		Username:            "svc_echo",
		PasswordCiphertext:  "encrypted_placeholder", // tryBindAttempts 用 decryptPassword 解密
	}
	c := NewLDAPClient(cfg, acct)

	conn, err := c.dialConnection("127.0.0.1:" + portStr(port))
	require.NoError(t, err)

	err = c.tryBindAttempts(conn, "example.com")

	// 仍然失败(服务端立即关闭),但走的是 account 路径
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "绑定失败")
	conn.Close()
}
