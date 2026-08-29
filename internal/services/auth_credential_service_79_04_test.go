package services

// =====================================================================
// Phase 79-04 Task 6: auth_credential 全方法测试(PasswordCipher stub)+ plan 收口
//
// 覆盖目标: auth_credential_service.go 0% → ≥70%(基线 142 stmts 全 unc,
// 79-RESEARCH §2)。
//
// 纪律(79-01 SUMMARY 手注沿用):helper/stub 名带 7904 后缀、sqlite t.TempDir 文件库、
// 禁 t.Parallel、协议断言引用 models.ProtocolType* 具名常量。
//
// stub 范式(Phase 73 D-02 "portwrite 纯 mock 范本",STATE.md 决策沿用):
// per-interface *Func 字段 + 未注册即 panic,不引入 testify/mock。
// race 安全:每个用例独立赋值 *Func,不跨用例共享(禁 t.Parallel 双保险)。
// =====================================================================

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

const acv7904Operator = "operator-7904"

// stubCipher7904 addomain.PasswordCipher 的 per-interface *Func stub(Phase 73 D-02 范本)。
// 契约:某方法未注册(*Func == nil)即 panic —— 防止测试静默走到非预期路径。
type stubCipher7904 struct {
	EncryptFunc *func(plaintext string) (string, error)
	DecryptFunc *func(ciphertext string) (string, error)

	// 观测面:记录入参,供测试断言 service 是否把明文交给 cipher。
	encryptCalls []string
	decryptCalls []string
}

func (s *stubCipher7904) Encrypt(plaintext string) (string, error) {
	if s.EncryptFunc == nil {
		panic("stubCipher7904: EncryptFunc 未注册(stub 契约:未注册即 panic)")
	}
	s.encryptCalls = append(s.encryptCalls, plaintext)
	return (*s.EncryptFunc)(plaintext)
}

func (s *stubCipher7904) Decrypt(ciphertext string) (string, error) {
	if s.DecryptFunc == nil {
		panic("stubCipher7904: DecryptFunc 未注册(stub 契约:未注册即 panic)")
	}
	s.decryptCalls = append(s.decryptCalls, ciphertext)
	return (*s.DecryptFunc)(ciphertext)
}

var _ addomain.PasswordCipher = (*stubCipher7904)(nil)

// acv7904ReversibleCipher 可逆 stub 行为:加密加 "enc:" 前缀,解密剥离。
func acv7904ReversibleCipher(s *stubCipher7904) {
	enc := func(plaintext string) (string, error) { return "enc:" + plaintext, nil }
	dec := func(ciphertext string) (string, error) {
		return strings.TrimPrefix(ciphertext, "enc:"), nil
	}
	s.EncryptFunc, s.DecryptFunc = &enc, &dec
}

// newAcv7904 装配 AuthCredentialService + sqlite(t.TempDir 文件库)+
// AutoMigrate AuthCredential 与 NetworkDevice(Delete 的"设备占用"检查引用)。
func newAcv7904(t *testing.T) (*AuthCredentialService, *gorm.DB, *stubCipher7904) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "acv7904.db")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "open sqlite temp db")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.AutoMigrate(
		&models.AuthCredential{},
		&models.NetworkDevice{},
	), "auto migrate auth credential chain models")
	stub := &stubCipher7904{}
	acv7904ReversibleCipher(stub)
	return NewAuthCredentialService(db, stub), db, stub
}

// acv7904SeedCredential 直插凭证行(cipherText 直接落库,绕过加密)。
func acv7904SeedCredential(t *testing.T, db *gorm.DB, name, password, enablePassword string,
	protocol models.ProtocolType, isDefault bool) *models.AuthCredential {
	t.Helper()
	cred := &models.AuthCredential{
		CredentialName:  name,
		ProtocolType:    protocol,
		Username:        "admin",
		Password:        password,
		EnablePassword:  enablePassword,
		SNMPCommunities: pq.StringArray{"public"},
		SNMPVersion:     models.SNMPVersionV2c,
		IsDefault:       isDefault,
	}
	require.NoError(t, db.Create(cred).Error, "seed credential %s", name)
	return cred
}

// -------------------------------------------------------------------------
// 用例
// -------------------------------------------------------------------------

// TestAcv7904_CreateValidate Create 合法请求 → 密码经 cipher.Encrypt 落库(stub 入参断言);
// validateCredentialConfig 分支表驱动(缺用户名/新凭据缺密码/更新可留空密码)。
func TestAcv7904_CreateValidate(t *testing.T) {
	ctx := context.Background()
	svc, db, stub := newAcv7904(t)

	// 合法创建:密码与特权密码都交给 cipher
	cred, err := svc.Create(ctx, &CreateCredentialRequest{
		CredentialName:  "core-ssh",
		ProtocolType:    models.ProtocolTypeSSH,
		Username:        "admin",
		Password:        "plain-secret",
		EnablePassword:  "plain-enable",
		SNMPCommunities: []string{"public", "private"},
		SNMPVersion:     models.SNMPVersionV2c,
		Description:     "核心交换机凭证",
		IsDefault:       true,
		CreatedBy:       acv7904Operator,
	})
	require.NoError(t, err)
	require.NotNil(t, cred)
	assert.Equal(t, []string{"plain-secret", "plain-enable"}, stub.encryptCalls,
		"两个密码字段的明文必须都交给 cipher.Encrypt(stub 记录入参)")
	// 返回值必须清空密码(服务约定)
	assert.Empty(t, cred.Password)
	assert.Empty(t, cred.EnablePassword)

	// 落库行:密文存储 + 字段完整
	var row models.AuthCredential
	require.NoError(t, db.Where("credential_name = ?", "core-ssh").First(&row).Error)
	assert.Equal(t, "enc:plain-secret", row.Password)
	assert.Equal(t, "enc:plain-enable", row.EnablePassword)
	assert.Equal(t, models.ProtocolTypeSSH, row.ProtocolType)
	assert.True(t, row.IsDefault)
	assert.Equal(t, pq.StringArray{"public", "private"}, row.SNMPCommunities)

	// 重复名称 → 拒绝
	_, err = svc.Create(ctx, &CreateCredentialRequest{
		CredentialName: "core-ssh", ProtocolType: models.ProtocolTypeSSH,
		Username: "admin", Password: "x",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "凭证名称已存在")

	// validateCredentialConfig 分支表驱动(私有方法同包直调)
	cases := []struct {
		name     string
		username string
		password string
		isNew    bool
		wantErr  string
	}{
		{name: "新建缺用户名", username: "", password: "p", isNew: true, wantErr: "请输入用户名"},
		{name: "新建缺密码", username: "u", password: "", isNew: true, wantErr: "请输入密码"},
		{name: "新建齐全", username: "u", password: "p", isNew: true},
		{name: "更新缺用户名", username: "", password: "", isNew: false, wantErr: "请输入用户名"},
		{name: "更新密码可留空", username: "u", password: "", isNew: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc2, _, _ := newAcv7904(t)
			err := svc2.validateCredentialConfig(models.ProtocolTypeSSH, tc.username, tc.password, nil, tc.isNew)
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}

	// ValidateCredential 兼容入口( isNewCredential=true 语义)
	err = svc.ValidateCredential(&models.AuthCredential{Username: "", Password: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "请输入用户名")
}

// TestAcv7904_ListPagination 预置多行 → 分页/过滤(名称模糊 + 协议具名常量)+ 密码隐藏。
func TestAcv7904_ListPagination(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := newAcv7904(t)

	acv7904SeedCredential(t, db, "list-ssh-1", "enc:a", "enc:a2", models.ProtocolTypeSSH, false)
	acv7904SeedCredential(t, db, "list-ssh-2", "enc:b", "", models.ProtocolTypeSSH, false)
	acv7904SeedCredential(t, db, "list-telnet", "enc:c", "", models.ProtocolTypeTelnet, false)

	// 分页
	list, total, err := svc.List(ctx, &ListCredentialRequest{BaseListRequest: baseListReq7904(1, 2)})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, list, 2)
	// 密码必须隐藏(List 的脱敏循环)
	for _, c := range list {
		assert.Empty(t, c.Password, "List 必须隐藏密码")
		assert.Empty(t, c.EnablePassword)
	}

	// 名称模糊过滤
	list, _, err = svc.List(ctx, &ListCredentialRequest{
		BaseListRequest: baseListReq7904(1, 10),
		CredentialName:  func() *string { s := "telnet"; return &s }(),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)
	assert.Equal(t, "list-telnet", list[0].CredentialName)

	// 协议过滤(具名常量)
	telnet := models.ProtocolTypeTelnet
	list, total, err = svc.List(ctx, &ListCredentialRequest{
		BaseListRequest: baseListReq7904(1, 10), ProtocolType: &telnet,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, models.ProtocolTypeTelnet, list[0].ProtocolType)

	// 排序白名单分支(credentialName ASC)
	asc := true
	list, _, err = svc.List(ctx, &ListCredentialRequest{
		BaseListRequest: base.BaseListRequest{
			Current: 1, PageSize: 10, OrderByColumn: "credentialName", IsAsc: &asc,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "list-ssh-1", list[0].CredentialName)

	// 空页 → 空集不报错
	list, total, err = svc.List(ctx, &ListCredentialRequest{BaseListRequest: baseListReq7904(9, 10)})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Empty(t, list)
}

// TestAcv7904_GetByIDWithPassword 走 cipher 解密路径(stub 返回固定明文)→ 字段一致;
// 不存在 → 错误。
func TestAcv7904_GetByIDWithPassword(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := newAcv7904(t)

	seeded := acv7904SeedCredential(t, db, "withpw", "enc:raw-secret", "enc:raw-enable",
		models.ProtocolTypeSSH, false)

	got, err := svc.GetByIDWithPassword(ctx, seeded.ID)
	require.NoError(t, err)
	assert.Equal(t, "raw-secret", got.Password, "解密路径必须还原明文")
	assert.Equal(t, "raw-enable", got.EnablePassword)
	assert.Equal(t, seeded.CredentialName, got.CredentialName)

	// 不存在 → 错误
	_, err = svc.GetByIDWithPassword(ctx, uuid.New().String())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询凭证失败")
}

// TestAcv7904_GetDecryptedCredential_RoundTrip Create(stub 加密)→
// GetDecryptedCredential(stub 解密)→ 明文一致;cipher 报错 → 错误上抛;
// GetByID(非解密版)必须返回空密码。
func TestAcv7904_GetDecryptedCredential_RoundTrip(t *testing.T) {
	ctx := context.Background()
	svc, db, stub := newAcv7904(t)

	created, err := svc.Create(ctx, &CreateCredentialRequest{
		CredentialName: "roundtrip",
		ProtocolType:   models.ProtocolTypeSSH,
		Username:       "admin",
		Password:       "round-trip-secret",
		EnablePassword: "round-trip-enable",
	})
	require.NoError(t, err)

	decrypted, err := svc.GetDecryptedCredential(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "round-trip-secret", decrypted.Password)
	assert.Equal(t, "round-trip-enable", decrypted.EnablePassword)
	assert.NotEmpty(t, stub.decryptCalls, "解密必须走 cipher")

	// 对照:GetByID(隐藏密码版)不触发解密且返回空密码
	hidden, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Empty(t, hidden.Password)
	assert.Empty(t, hidden.EnablePassword)

	// cipher.Decrypt 报错 → 严格模式拒绝使用(错误上抛分支)
	failStub := &stubCipher7904{}
	dec := func(ciphertext string) (string, error) { return "", assert.AnError }
	failStub.DecryptFunc = &dec
	failSvc := NewAuthCredentialService(db, failStub)
	_, err = failSvc.GetDecryptedCredential(ctx, created.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "密码解密失败，请重新设置凭证密码")

	// EnablePassword 解密失败 → 特权密码分支
	enableFail := &stubCipher7904{}
	encOK := func(plaintext string) (string, error) { return "enc:" + plaintext, nil }
	decFail := func(ciphertext string) (string, error) {
		if strings.Contains(ciphertext, "enable") {
			return "", assert.AnError
		}
		return strings.TrimPrefix(ciphertext, "enc:"), nil
	}
	enableFail.EncryptFunc = &encOK
	enableFail.DecryptFunc = &decFail
	enableSvc := NewAuthCredentialService(db, enableFail)
	_, err = enableSvc.GetDecryptedCredential(ctx, created.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "特权密码解密失败")

	// 不存在 → 错误
	_, err = svc.GetDecryptedCredential(ctx, uuid.New().String())
	require.Error(t, err)
}

// TestAcv7904_Update_Delete_BatchDelete 更新读回(密码轮换/保留);名称冲突/不存在/占用删除
// 拒绝分支;单删/批删。
func TestAcv7904_Update_Delete_BatchDelete(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := newAcv7904(t)

	seeded := acv7904SeedCredential(t, db, "upd-cred", "enc:old", "enc:old-enable",
		models.ProtocolTypeSSH, false)
	other := acv7904SeedCredential(t, db, "other-cred", "enc:o", "", models.ProtocolTypeSSH, false)

	// 更新:轮换密码 + 改字段
	require.NoError(t, svc.Update(ctx, &UpdateCredentialRequest{
		ID:              seeded.ID,
		CredentialName:  "upd-cred-new",
		ProtocolType:    models.ProtocolTypeTelnet,
		Username:        "operator",
		Password:        "brand-new",
		EnablePassword:  "brand-new-enable",
		SNMPCommunities: []string{"community-x"},
		SNMPVersion:     models.SNMPVersionV3,
		Description:     "已轮换",
		UpdatedBy:       acv7904Operator,
	}))
	got, err := svc.GetDecryptedCredential(ctx, seeded.ID)
	require.NoError(t, err)
	assert.Equal(t, "brand-new", got.Password, "新密码必须经 cipher 轮换")
	assert.Equal(t, "brand-new-enable", got.EnablePassword)
	assert.Equal(t, models.ProtocolTypeTelnet, got.ProtocolType)
	assert.Equal(t, models.SNMPVersionV3, got.SNMPVersion)
	assert.Equal(t, acv7904Operator, got.UpdatedBy)

	// 更新:密码留空 → 保留原密文(不触发加密)
	require.NoError(t, svc.Update(ctx, &UpdateCredentialRequest{
		ID:             seeded.ID,
		CredentialName: "upd-cred-new",
		ProtocolType:   models.ProtocolTypeTelnet,
		Username:       "operator",
	}))
	kept, err := svc.GetDecryptedCredential(ctx, seeded.ID)
	require.NoError(t, err)
	assert.Equal(t, "brand-new", kept.Password, "留空密码必须保留原密文")

	// 更新:名称被其他凭证占用 → 拒绝
	err = svc.Update(ctx, &UpdateCredentialRequest{
		ID: seeded.ID, CredentialName: "other-cred", ProtocolType: models.ProtocolTypeSSH, Username: "u",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "凭证名称已存在")

	// 更新:不存在 → 拒绝
	err = svc.Update(ctx, &UpdateCredentialRequest{
		ID: uuid.New().String(), CredentialName: "ghost", ProtocolType: models.ProtocolTypeSSH, Username: "u",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "凭证不存在")

	// 删除被拒:有设备占用该凭证
	inUse := acv7904SeedCredential(t, db, "in-use", "enc:i", "", models.ProtocolTypeSSH, false)
	require.NoError(t, db.Create(&models.NetworkDevice{
		DeviceName: "sw-use", DeviceType: models.DeviceTypeSwitch, Vendor: models.VendorHuawei,
		IPAddress: "10.9.0.1", CredentialID: &inUse.ID,
	}).Error)
	err = svc.Delete(ctx, inUse.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "个设备正在使用此凭证")

	// 单删(无占用)→ 成功
	require.NoError(t, svc.Delete(ctx, other.ID))
	var gone int64
	require.NoError(t, db.Model(&models.AuthCredential{}).Where("id = ?", other.ID).Count(&gone).Error)
	assert.Equal(t, int64(0), gone)

	// 锁定现行为:Delete 不校验行存在性(只查设备占用),删除不存在的 ID 是静默 no-op
	// → BatchDelete 混入不存在 ID 也不报错(与 network_device_service 的先查后删不同)。
	err = svc.BatchDelete(ctx, []string{seeded.ID, uuid.New().String()})
	require.NoError(t, err, "锁定现行为:不存在 ID 的 Delete 是 no-op 不报错")
	require.NoError(t, db.Model(&models.AuthCredential{}).Count(&gone).Error)
	assert.Equal(t, int64(1), gone, "只剩被占用的 in-use")
}

// TestAcv7904_DefaultCredential_Uniqueness SetDefaultCredential 后旧默认被清 →
// GetDefaultCredential 唯一且密码隐藏;Set 不存在 ID 不报错(锁定 Update 语义);
// 无默认 → 错误。
func TestAcv7904_DefaultCredential_Uniqueness(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := newAcv7904(t)

	first := acv7904SeedCredential(t, db, "default-1", "enc:1", "", models.ProtocolTypeSSH, true)
	second := acv7904SeedCredential(t, db, "default-2", "enc:2", "", models.ProtocolTypeTelnet, false)

	// 当前默认 → first
	got, err := svc.GetDefaultCredential(ctx)
	require.NoError(t, err)
	assert.Equal(t, first.ID, got.ID)
	assert.Empty(t, got.Password, "默认凭证同样隐藏密码")

	// 切换默认 → 旧默认被清
	require.NoError(t, svc.SetDefaultCredential(ctx, second.ID, acv7904Operator))
	var firstDefault bool
	require.NoError(t, db.Model(&models.AuthCredential{}).Where("id = ?", first.ID).
		Pluck("is_default", &firstDefault).Error)
	assert.False(t, firstDefault, "旧默认必须被清")
	got, err = svc.GetDefaultCredential(ctx)
	require.NoError(t, err)
	assert.Equal(t, second.ID, got.ID)

	// Update 路径的默认切换(IsDefault && !credential.IsDefault 分支)
	third := acv7904SeedCredential(t, db, "default-3", "enc:3", "", models.ProtocolTypeSSH, false)
	require.NoError(t, svc.Update(ctx, &UpdateCredentialRequest{
		ID: third.ID, CredentialName: "default-3", ProtocolType: models.ProtocolTypeSSH,
		Username: "admin", IsDefault: true,
	}))
	got, err = svc.GetDefaultCredential(ctx)
	require.NoError(t, err)
	assert.Equal(t, third.ID, got.ID, "Update(IsDefault=true) 必须接管默认")

	// 无默认场景 → 错误
	require.NoError(t, db.Model(&models.AuthCredential{}).Where("1=1").Update("is_default", false).Error)
	_, err = svc.GetDefaultCredential(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未找到默认凭证")

	// Set 不存在 ID:UPDATE 影响 0 行,GORM 不视为错误(锁定 Update 语义)
	require.NoError(t, svc.SetDefaultCredential(ctx, uuid.New().String(), acv7904Operator))
}

// TestAcv7904_GetStatistics 预置多协议凭据 → CredentialStatistics 计数与手算一致。
func TestAcv7904_GetStatistics(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := newAcv7904(t)

	acv7904SeedCredential(t, db, "stat-ssh-1", "enc:a", "", models.ProtocolTypeSSH, false)
	acv7904SeedCredential(t, db, "stat-ssh-2", "enc:b", "", models.ProtocolTypeSSH, false)
	acv7904SeedCredential(t, db, "stat-telnet", "enc:c", "", models.ProtocolTypeTelnet, false)

	stats, err := svc.GetStatistics(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, int64(3), stats.Total)
	assert.Equal(t, int64(2), stats.SSH, "ssh 计数(models.ProtocolTypeSSH)")
	assert.Equal(t, int64(1), stats.Telnet, "telnet 计数(models.ProtocolTypeTelnet)")

	// 空表 → 全 0
	svc2, _, _ := newAcv7904(t)
	empty, err := svc2.GetStatistics(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), empty.Total)
}

// TestAcv7904_GetDevicesByCredential 预置关联设备 → 命中集;无关联 → 空集。
func TestAcv7904_GetDevicesByCredential(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := newAcv7904(t)

	cred := acv7904SeedCredential(t, db, "dev-cred", "enc:a", "", models.ProtocolTypeSSH, false)
	for _, ip := range []string{"10.10.1.1", "10.10.1.2"} {
		require.NoError(t, db.Create(&models.NetworkDevice{
			DeviceName: "sw-" + ip, DeviceType: models.DeviceTypeSwitch, Vendor: models.VendorHuawei,
			IPAddress: ip, CredentialID: &cred.ID,
		}).Error)
	}
	require.NoError(t, db.Create(&models.NetworkDevice{
		DeviceName: "sw-free", DeviceType: models.DeviceTypeSwitch, Vendor: models.VendorHuawei,
		IPAddress: "10.10.1.3",
	}).Error)

	devices, err := svc.GetDevicesByCredential(ctx, cred.ID)
	require.NoError(t, err)
	assert.Len(t, devices, 2)

	empty, err := svc.GetDevicesByCredential(ctx, uuid.New().String())
	require.NoError(t, err)
	assert.Empty(t, empty)
}

// TestAcv7904_CipherNilGuard cipher 为 nil 时 Create/Update 必须报"加密器未初始化"
// (服务层防御分支,不许明文落库)。
func TestAcv7904_CipherNilGuard(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := newAcv7904(t)
	nilSvc := NewAuthCredentialService(db, nil)

	_, err := nilSvc.Create(ctx, &CreateCredentialRequest{
		CredentialName: "nil-cipher", ProtocolType: models.ProtocolTypeSSH,
		Username: "u", Password: "p",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SM4 加密器未初始化")

	seeded := acv7904SeedCredential(t, db, "nil-cipher-update", "enc:x", "", models.ProtocolTypeSSH, false)
	err = nilSvc.Update(ctx, &UpdateCredentialRequest{
		ID: seeded.ID, CredentialName: "nil-cipher-update", ProtocolType: models.ProtocolTypeSSH,
		Username: "u", Password: "new",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SM4 加密器未初始化")

	_ = svc // 主装配仅用于种子行
}

// TestAcv7904_StubContract_UnregisteredPanics stub 契约:未注册的 *Func 被调用即 panic
// (防止测试静默走通非预期路径)。
func TestAcv7904_StubContract_UnregisteredPanics(t *testing.T) {
	encryptOnly := &stubCipher7904{}
	enc := func(plaintext string) (string, error) { return "enc:" + plaintext, nil }
	encryptOnly.EncryptFunc = &enc

	assert.NotPanics(t, func() { _, _ = encryptOnly.Encrypt("x") })
	assert.PanicsWithValue(t,
		"stubCipher7904: DecryptFunc 未注册(stub 契约:未注册即 panic)",
		func() { _, _ = encryptOnly.Decrypt("enc:x") })

	decryptOnly := &stubCipher7904{}
	assert.PanicsWithValue(t,
		"stubCipher7904: EncryptFunc 未注册(stub 契约:未注册即 panic)",
		func() { _, _ = decryptOnly.Encrypt("x") })
}
