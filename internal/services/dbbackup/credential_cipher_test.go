package dbbackup

import (
	"context"
	"strings"
	"testing"

	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
)

// ---------------------------------------------------------------------------
// 备份计划凭据 SM4 字段加解密测试
// ---------------------------------------------------------------------------
//
// 覆盖需求：
//   - 加解密往返（SavePlan → sys_config 存密文 → GetPlan 解出真值）；
//   - 占位符（******）/ 空串不覆盖真值（用户未改凭据时的兼容）；
//   - sys_config 里不再是明文（去掉「按字面写入」）；
//   - 未注册 cipher 时拒绝明文落库（fail-closed）；
//   - 历史明文配置平滑兼容（改造前已落库的值仍能读出）。

// testSM4Cipher 构造一把测试用 SM4 cipher（与 core 注入的 core.SM4Cipher 同型）。
func testSM4Cipher(t *testing.T) *crypto.SM4Cipher {
	t.Helper()
	c, err := crypto.NewSM4Cipher("dGVzdC1zZWNyZXQxNiEhIQ==") // 仓库默认测试 key, base64(16B)
	if err != nil {
		t.Fatalf("构造测试 SM4 cipher 失败: %v", err)
	}
	return c
}

// TestCredentialCipher_RoundTrip 验证 SavePlan/GetPlan 加解密往返：
//  1. 注册 cipher，保存含真实凭据的计划；
//  2. 直读 sys_config 断言三个凭据字段都是 sm4: 密文、绝不含明文；
//  3. GetPlan 解出的真值与原值一致；
//  4. 换个 cipher（不同 key）读会报错（密钥不匹配不静默回吐密文）。
func TestCredentialCipher_RoundTrip(t *testing.T) {
	db := newTestDB(t)
	exec := &fakeDumpExecutor{tables: []string{}, dumpBytes: []byte("X")}
	svc := newTestService(t, db, exec)

	c := testSM4Cipher(t)
	RegisterCredentialCipher(c)
	t.Cleanup(func() { RegisterCredentialCipher(nil) })

	want := &Plan{
		Enabled: true, Cron: "0 0 3 * * *", Retention: 5, Storage: StorageOSS,
		OSS: OSSConfig{Endpoint: "s3.example.com", Bucket: "bk",
			AccessKey: "AK-PLAIN-123", SecretKey: "SK-PLAIN-456", UseSSL: true},
		FTP: FTPConfig{Host: "ftp.example.com", Port: 2121, User: "u",
			Password: "FTP-PWD-789", RemoteDir: "/bak"},
		SMB: SMBConfig{Path: "/mnt/share"},
	}
	if err := svc.SavePlan(context.Background(), want, "admin"); err != nil {
		t.Fatalf("SavePlan 失败: %v", err)
	}

	// 直读 sys_config：凭据必须是密文且不含明文
	for key, plain := range map[string]string{
		cfgKeyOSSAccessKey: "AK-PLAIN-123",
		cfgKeyOSSSecretKey: "SK-PLAIN-456",
		cfgKeyFTPPassword:  "FTP-PWD-789",
	} {
		raw, err := readConfigValue(context.Background(), db, key)
		if err != nil {
			t.Fatalf("读 sys_config[%s] 失败: %v", key, err)
		}
		if !strings.HasPrefix(raw, credentialCipherPrefix) {
			t.Fatalf("sys_config[%s] 应为 SM4 密文（前缀 %q），实得 %q",
				key, credentialCipherPrefix, raw)
		}
		if strings.Contains(raw, plain) {
			t.Fatalf("sys_config[%s] 仍含明文 %q", key, plain)
		}
	}

	got, err := svc.GetPlan(context.Background())
	if err != nil {
		t.Fatalf("GetPlan 失败: %v", err)
	}
	if got.OSS.AccessKey != want.OSS.AccessKey ||
		got.OSS.SecretKey != want.OSS.SecretKey ||
		got.FTP.Password != want.FTP.Password {
		t.Fatalf("加解密往返丢失凭据: got=%+v", got)
	}
}

// TestCredentialCipher_PlaceholderDoesNotOverwrite 验证占位符/空串回填：
// 先存真值，再用「脱敏占位符 + 空串」保存（用户未改凭据），真值必须保持，
// 且占位符本身绝不能被当成真值写进 sys_config。
func TestCredentialCipher_PlaceholderDoesNotOverwrite(t *testing.T) {
	db := newTestDB(t)
	exec := &fakeDumpExecutor{tables: []string{}, dumpBytes: []byte("X")}
	svc := newTestService(t, db, exec)

	c := testSM4Cipher(t)
	RegisterCredentialCipher(c)
	t.Cleanup(func() { RegisterCredentialCipher(nil) })

	first := &Plan{
		Enabled: true, Cron: "0 0 2 * * *", Retention: 7, Storage: StorageOSS,
		OSS: OSSConfig{AccessKey: "REAL-AK", SecretKey: "REAL-SK", UseSSL: true},
		FTP: FTPConfig{Password: "REAL-PWD", RemoteDir: "/"},
	}
	if err := svc.SavePlan(context.Background(), first, "admin"); err != nil {
		t.Fatalf("首次 SavePlan 失败: %v", err)
	}

	// 用户打开表单、未改凭据：敏感字段回传 "******" 或空串（前端两种形态都兼容）
	unchanged := &Plan{
		Enabled: false, Cron: "0 0 4 * * *", Retention: 3, Storage: StorageOSS,
		OSS: OSSConfig{Endpoint: "new-endpoint", Bucket: "new-bucket",
			AccessKey: PlanMaskPlaceholder, SecretKey: "" /* 空串 */, UseSSL: false},
		FTP: FTPConfig{Host: "new-host", User: "new-user", Password: PlanMaskPlaceholder, RemoteDir: "/new"},
	}
	if err := svc.SavePlan(context.Background(), unchanged, "admin"); err != nil {
		t.Fatalf("占位符回填 SavePlan 失败: %v", err)
	}

	got, err := svc.GetPlan(context.Background())
	if err != nil {
		t.Fatalf("GetPlan 失败: %v", err)
	}
	if got.OSS.AccessKey != "REAL-AK" || got.OSS.SecretKey != "REAL-SK" || got.FTP.Password != "REAL-PWD" {
		t.Fatalf("占位符/空串覆盖了真实凭据: got=%+v", got)
	}
	// 非敏感字段应正常更新
	if got.OSS.Endpoint != "new-endpoint" || got.FTP.Host != "new-host" || got.Retention != 3 {
		t.Fatalf("非敏感字段未更新: got=%+v", got)
	}
	// sys_config 里不能出现占位符字面
	for _, key := range []string{cfgKeyOSSAccessKey, cfgKeyOSSSecretKey, cfgKeyFTPPassword} {
		raw, _ := readConfigValue(context.Background(), db, key)
		if strings.Contains(raw, PlanMaskPlaceholder) {
			t.Fatalf("sys_config[%s] 含脱敏占位符字面: %q", key, raw)
		}
	}
}

// TestCredentialCipher_NoCipherRejectsPlaintext 写路径 fail-closed：
// 未注册 cipher 且要保存真实凭据时必须报错，绝不把明文落库。
func TestCredentialCipher_NoCipherRejectsPlaintext(t *testing.T) {
	db := newTestDB(t)
	exec := &fakeDumpExecutor{tables: []string{}, dumpBytes: []byte("X")}
	svc := newTestService(t, db, exec)
	RegisterCredentialCipher(nil)

	p := &Plan{
		Enabled: false, Cron: "0 0 2 * * *", Retention: 7, Storage: StorageOSS,
		OSS: OSSConfig{AccessKey: "SHOULD-NOT-PERSIST", SecretKey: "SK", UseSSL: true},
		FTP: FTPConfig{Password: "PWD", RemoteDir: "/"},
	}
	if err := svc.SavePlan(context.Background(), p, "admin"); err == nil {
		t.Fatal("未注册 cipher 时应拒绝明文凭据落库")
	}
	raw, _ := readConfigValue(context.Background(), db, cfgKeyOSSAccessKey)
	if strings.Contains(raw, "SHOULD-NOT-PERSIST") {
		t.Fatalf("fail-closed 被绕过：明文已落库 %q", raw)
	}
}

// TestCredentialCipher_LegacyPlaintextCompatible 历史明文配置平滑兼容：
// 改造前已按字面写入的明文值，loadPlan 仍能原样读出（不报错、不误解密）。
func TestCredentialCipher_LegacyPlaintextCompatible(t *testing.T) {
	db := newTestDB(t)
	exec := &fakeDumpExecutor{tables: []string{}, dumpBytes: []byte("X")}
	svc := newTestService(t, db, exec)
	RegisterCredentialCipher(nil)

	seedConfig(t, db, cfgKeyOSSAccessKey, "LEGACY-AK")
	seedConfig(t, db, cfgKeyOSSSecretKey, "LEGACY-SK")
	seedConfig(t, db, cfgKeyFTPPassword, "LEGACY-PWD")

	got, err := svc.GetPlan(context.Background())
	if err != nil {
		t.Fatalf("历史明文配置读取失败: %v", err)
	}
	if got.OSS.AccessKey != "LEGACY-AK" || got.OSS.SecretKey != "LEGACY-SK" || got.FTP.Password != "LEGACY-PWD" {
		t.Fatalf("历史明文凭据未兼容读出: got=%+v", got)
	}
}

// TestCredentialCipher_ClearMarkerDeletes 显式清空凭据：SavePlan 收到
// __CLEAR__ 标记时该字段置空落库（删除凭据），其余未改动字段保留原值。
func TestCredentialCipher_ClearMarkerDeletes(t *testing.T) {
	db := newTestDB(t)
	exec := &fakeDumpExecutor{tables: []string{}, dumpBytes: []byte("X")}
	svc := newTestService(t, db, exec)

	RegisterCredentialCipher(testSM4Cipher(t))
	t.Cleanup(func() { RegisterCredentialCipher(nil) })

	first := &Plan{
		Enabled: true, Cron: "0 0 2 * * *", Retention: 7, Storage: StorageOSS,
		OSS: OSSConfig{AccessKey: "REAL-AK", SecretKey: "REAL-SK", UseSSL: true},
		FTP: FTPConfig{Password: "REAL-PWD", RemoteDir: "/"},
	}
	if err := svc.SavePlan(context.Background(), first, "admin"); err != nil {
		t.Fatalf("首次 SavePlan 失败: %v", err)
	}

	// 管理员清空 OSS secretKey：传 __CLEAR__ 标记；其余敏感字段未改动。
	clearing := &Plan{
		Enabled: true, Cron: "0 0 2 * * *", Retention: 7, Storage: StorageOSS,
		OSS: OSSConfig{AccessKey: "REAL-AK", SecretKey: CredentialClearMarker, UseSSL: true},
		FTP: FTPConfig{Password: PlanMaskPlaceholder, RemoteDir: "/"},
	}
	if err := svc.SavePlan(context.Background(), clearing, "admin"); err != nil {
		t.Fatalf("清空凭据 SavePlan 失败: %v", err)
	}

	got, err := svc.GetPlan(context.Background())
	if err != nil {
		t.Fatalf("GetPlan 失败: %v", err)
	}
	if got.OSS.SecretKey != "" {
		t.Fatalf("__CLEAR__ 应删除凭据（落库空串），实得 %q", got.OSS.SecretKey)
	}
	if got.OSS.AccessKey != "REAL-AK" || got.FTP.Password != "REAL-PWD" {
		t.Fatalf("未改动凭据应保留原值: got=%+v", got)
	}
}
