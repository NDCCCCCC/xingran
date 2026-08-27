package services

// =====================================================================
// Phase 79-04 Task 4: notification_config CRUD + EncryptPassword/DecryptPassword
//                     AES-GCM round-trip
// Phase 79-04 Task 5(同文件追加): notification_sender 派发链(nil-sender 错误分支
//                     + 收件人解析)
//
// 覆盖目标: notification_config_service.go 0% → ≥70%(基线 127 stmts 全 unc)+
// notification_sender_service.go 0% → ≥70%(基线 120 stmts 全 unc,79-RESEARCH §2)。
//
// 纪律(79-01 SUMMARY 手注沿用):helper 名带 7904 后缀、sqlite t.TempDir 文件库、
// 禁 t.Parallel、状态断言引用 models.NotificationConfigStatus* / models.APIConfigType*
// / models.ChannelType* / models.PublishStatus* 具名常量。
//
// 真实发送路径不在本 plan(79-05 email/api_sender wire 级):本文件只测派发与
// 错误分支,所有发送断言都停在"配置缺失/配置停用/密码解密失败/收件人无效"这类
// 不出网的分支上(EmailSenderService.Send / APISenderService.Send 的前置校验)。
// =====================================================================

import (
	"context"
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ncf7904Key 显式 32 字节测试密钥(威胁模型 T-79-04-03:仅测试,禁作生产密钥复用;
// 实现内部截断到前 16 字节,AES-128-GCM)。
const ncf7904Key = "7904-TEST-ONLY-KEY-DO-NOT-USE-IN-PROD"

// newNcf7904 装配 NotificationConfigService + sqlite(t.TempDir 文件库)+
// AutoMigrate EmailConfig / APINotificationConfig(两 model 无 PG 专属 DDL)。
func newNcf7904(t *testing.T) (*NotificationConfigService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ncf7904.db")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "open sqlite temp db")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.AutoMigrate(
		&models.EmailConfig{},
		&models.APINotificationConfig{},
	), "auto migrate notification config models")
	return NewNotificationConfigService(db), db
}

// ncf7904EmailConfig 构造一条合法邮箱配置(未落库)。
func ncf7904EmailConfig(name, host string, status int) *models.EmailConfig {
	return &models.EmailConfig{
		ID:         uuid.New().String(),
		ConfigName: name,
		Host:       host,
		Port:       587,
		Username:   "noreply@example.com",
		Password:   "encrypted-password",
		FromName:   "XingRan",
		FromEmail:  "noreply@example.com",
		UseSSL:     true,
		Status:     status,
		DelFlag:    0,
	}
}

// -------------------------------------------------------------------------
// Task 4: notification_config
// -------------------------------------------------------------------------

// TestNcf7904_EncryptDecrypt_RoundTrip AES-GCM 往返:密文非明文、可还原、随机 nonce
// (同明文两次加密密文不同)、空明文分支、默认 key / 短 key 补齐 / 长 key 截断分支。
func TestNcf7904_EncryptDecrypt_RoundTrip(t *testing.T) {
	// 基本往返(显式测试 key)
	cipher, err := EncryptPassword("s3cret-密码", ncf7904Key)
	require.NoError(t, err)
	assert.NotEqual(t, "s3cret-密码", cipher, "密文必须非明文")
	assert.NotEmpty(t, cipher)
	plain, err := DecryptPassword(cipher, ncf7904Key)
	require.NoError(t, err)
	assert.Equal(t, "s3cret-密码", plain)

	// 随机 nonce:同明文两次加密得到不同密文,但都能还原
	cipher2, err := EncryptPassword("s3cret-密码", ncf7904Key)
	require.NoError(t, err)
	assert.NotEqual(t, cipher, cipher2, "GCM 随机 nonce 必须使同明文产出不同密文")
	plain2, err := DecryptPassword(cipher2, ncf7904Key)
	require.NoError(t, err)
	assert.Equal(t, "s3cret-密码", plain2)

	// 不同明文不同密文
	other, err := EncryptPassword("another", ncf7904Key)
	require.NoError(t, err)
	assert.NotEqual(t, cipher, other)

	// 空明文分支(Seal 空数据,往返还原空串)
	emptyCipher, err := EncryptPassword("", ncf7904Key)
	require.NoError(t, err)
	emptyPlain, err := DecryptPassword(emptyCipher, ncf7904Key)
	require.NoError(t, err)
	assert.Equal(t, "", emptyPlain)

	// key="" → 默认 key 分支(与显式 "xingran-default-key" 等价)
	defCipher, err := EncryptPassword("default-key-pw", "")
	require.NoError(t, err)
	defPlain, err := DecryptPassword(defCipher, "")
	require.NoError(t, err)
	assert.Equal(t, "default-key-pw", defPlain)

	// 短 key 补齐分支:len(key)<16 用 "xingran-notificaion" 尾部补齐
	shortCipher, err := EncryptPassword("short-key-pw", "abc")
	require.NoError(t, err)
	shortPlain, err := DecryptPassword(shortCipher, "abc")
	require.NoError(t, err)
	assert.Equal(t, "short-key-pw", shortPlain)

	// 长 key 截断分支:前 16 字节相同的不同 key 可互解(实现 key=key[:16])
	longCipher, err := EncryptPassword("trunc-pw", ncf7904Key+"EXTRA")
	require.NoError(t, err)
	longPlain, err := DecryptPassword(longCipher, ncf7904Key)
	require.NoError(t, err)
	assert.Equal(t, "trunc-pw", longPlain)
}

// TestNcf7904_Decrypt_WrongKeyAndBadCipher 错 key → GCM 认证失败;坏 base64 → 解码错误;
// 合法 base64 但短于 nonce → "ciphertext too short"。
func TestNcf7904_Decrypt_WrongKeyAndBadCipher(t *testing.T) {
	cipher, err := EncryptPassword("topsecret", ncf7904Key)
	require.NoError(t, err)

	// 错 key(同为 16 字节长度,认证必失败)
	_, err = DecryptPassword(cipher, "WRONG-WRONG-KEY0")
	require.Error(t, err, "错 key 必须 GCM 认证失败")

	// 错 key(不同长度,经补齐/截断后仍不同)
	_, err = DecryptPassword(cipher, "nope")
	require.Error(t, err)

	// 坏密文形态 1:非 base64
	_, err = DecryptPassword("not-base64-!!", ncf7904Key)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "base64")

	// 坏密文形态 2:合法 base64 但短于 nonce size
	short := base64.StdEncoding.EncodeToString([]byte("abc"))
	_, err = DecryptPassword(short, ncf7904Key)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")

	// 坏密文形态 3:nonce 正确但密文体被篡改
	raw, err := base64.StdEncoding.DecodeString(cipher)
	require.NoError(t, err)
	tampered := append([]byte{}, raw...)
	tampered[len(tampered)-1] ^= 0xFF
	_, err = DecryptPassword(base64.StdEncoding.EncodeToString(tampered), ncf7904Key)
	require.Error(t, err, "篡改密文体必须 GCM 认证失败")

	// 坏密文形态 4:密文体为空(base64 的空串)
	_, err = DecryptPassword("", ncf7904Key)
	require.Error(t, err)
}

// TestNcf7904_EmailConfigCRUD Create(系统仅一条)/ GetByID / List(分页+状态过滤)/
// Update / Delete(软删 del_flag)/ GetDefaultEmailConfig 命中与无默认两分支。
func TestNcf7904_EmailConfigCRUD(t *testing.T) {
	ctx := context.Background()
	svc, db := newNcf7904(t)

	// GetDefaultEmailConfig:空表 → "未设置邮件配置"
	_, err := svc.GetDefaultEmailConfig(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未设置邮件配置")

	// 创建首条(自动 IsDefault=true)
	cfg := ncf7904EmailConfig("主邮箱", "smtp.example.com", int(models.NotificationConfigStatusNormal))
	require.NoError(t, svc.CreateEmailConfig(ctx, cfg))
	assert.True(t, cfg.IsDefault, "实现规定:首条配置自动设为默认")

	// 第二条 → 拒绝(系统只允许一条)
	dup := ncf7904EmailConfig("第二邮箱", "smtp2.example.com", int(models.NotificationConfigStatusNormal))
	err = svc.CreateEmailConfig(ctx, dup)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "只允许一条")

	// GetByID 命中
	got, err := svc.GetEmailConfigByID(ctx, cfg.ID)
	require.NoError(t, err)
	assert.Equal(t, "主邮箱", got.ConfigName)
	assert.Equal(t, "smtp.example.com", got.Host)

	// GetByID 不存在 / 已软删 → 错误
	_, err = svc.GetEmailConfigByID(ctx, uuid.New().String())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "邮箱配置不存在")

	// List:分页 + 状态过滤(del_flag=0 基线)
	list, total, err := svc.ListEmailConfigs(ctx, 1, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)
	stopped := int(models.NotificationConfigStatusStopped)
	list, total, err = svc.ListEmailConfigs(ctx, 1, 10, &stopped)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total, "状态过滤 models.NotificationConfigStatusStopped 不命中")
	// 空页 → 空列表不报错
	list, total, err = svc.ListEmailConfigs(ctx, 5, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Empty(t, list)

	// GetDefaultEmailConfig:命中(only-one + status normal)
	def, err := svc.GetDefaultEmailConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, cfg.ID, def.ID)

	// Update:改名 + 保留密码字段(GORM struct Updates 跳过零值字段 → 空密码不覆盖)。
	// 注意:更新载荷不得携带 ID —— GORM struct Updates 会把非零主键一并写进 SET,
	// 带上 ID 会改写行主键(锁定现行为,调用侧以空 ID 规避)。
	rename := ncf7904EmailConfig("", "", 0)
	rename.ID = ""
	rename.ConfigName = "主邮箱-改"
	require.NoError(t, svc.UpdateEmailConfig(ctx, cfg.ID, rename))
	got, err = svc.GetEmailConfigByID(ctx, cfg.ID)
	require.NoError(t, err)
	assert.Equal(t, "主邮箱-改", got.ConfigName)
	assert.Equal(t, "encrypted-password", got.Password, "零值字段不得覆盖原密码")
	// Update 不存在 → 错误
	err = svc.UpdateEmailConfig(ctx, uuid.New().String(), rename)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "邮箱配置不存在")

	// Delete:软删(del_flag=1)后不可再查,GetDefault 亦转为"未设置"
	require.NoError(t, svc.DeleteEmailConfig(ctx, cfg.ID))
	_, err = svc.GetEmailConfigByID(ctx, cfg.ID)
	require.Error(t, err)
	var delFlag int
	require.NoError(t, db.Model(&models.EmailConfig{}).Where("id = ?", cfg.ID).
		Pluck("del_flag", &delFlag).Error)
	assert.Equal(t, 1, delFlag, "Delete 必须是 del_flag=1 软删")
	_, err = svc.GetDefaultEmailConfig(ctx)
	require.Error(t, err)

	// Delete 不存在 → RowsAffected==0 → 错误
	err = svc.DeleteEmailConfig(ctx, uuid.New().String())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "邮箱配置不存在")

	// 软删后允许重新创建(唯一性检查只看 del_flag=0)
	again := ncf7904EmailConfig("重建邮箱", "smtp3.example.com", int(models.NotificationConfigStatusNormal))
	require.NoError(t, svc.CreateEmailConfig(ctx, again))
	assert.True(t, again.IsDefault)
}

// TestNcf7904_EmailConfig_DefaultUniqueness Update(IsDefault=true)分支:先清其他默认
// 再更新自身。系统"仅一条"规则使服务层无法建出第二条 → 用直插行驱动该分支(注释锁定语义)。
func TestNcf7904_EmailConfig_DefaultUniqueness(t *testing.T) {
	ctx := context.Background()
	svc, db := newNcf7904(t)

	primary := ncf7904EmailConfig("主配置", "smtp.example.com", int(models.NotificationConfigStatusNormal))
	require.NoError(t, svc.CreateEmailConfig(ctx, primary))

	// 直插第二行(绕过服务层"仅一条"校验)→ 两行都是默认,制造待清理状态
	shadow := ncf7904EmailConfig("影子配置", "smtp9.example.com", int(models.NotificationConfigStatusNormal))
	shadow.IsDefault = true
	require.NoError(t, db.Create(shadow).Error)

	// Update primary 且 IsDefault=true → 其他默认被清(:115-117 分支)
	primary.ConfigName = "主配置-改"
	primary.IsDefault = true
	require.NoError(t, svc.UpdateEmailConfig(ctx, primary.ID, primary))

	var shadowDefault bool
	require.NoError(t, db.Model(&models.EmailConfig{}).Where("id = ?", shadow.ID).
		Pluck("is_default", &shadowDefault).Error)
	assert.False(t, shadowDefault, "Update(IsDefault=true) 必须清掉其他默认配置")
}

// TestNcf7904_APIConfigCRUD APINotificationConfig 六方法 + configType 过滤(具名常量)+
// Update 的默认互斥与不存在分支 + 软删语义。
func TestNcf7904_APIConfigCRUD(t *testing.T) {
	ctx := context.Background()
	svc, db := newNcf7904(t)

	mk := func(name string, ctype models.APIConfigType, isDefault bool) *models.APINotificationConfig {
		return &models.APINotificationConfig{
			ID:         uuid.New().String(),
			ConfigName: name,
			ConfigType: ctype,
			APIURL:     "https://qyapi.weixin.qq.com/cgi-bin/webhook/send",
			APIMethod:  "POST",
			AuthType:   models.AuthTypeNone,
			RetryCount: 2,
			Timeout:    5,
			IsDefault:  isDefault,
			Status:     int(models.NotificationConfigStatusNormal),
			DelFlag:    0,
		}
	}

	// Create(同类型默认互斥:首条无同类 → 直接落库)
	hook := mk("企微机器人", models.APIConfigTypeWebhook, true)
	require.NoError(t, svc.CreateAPINotificationConfig(ctx, hook))

	// GetByID 命中 / 不存在
	got, err := svc.GetAPINotificationConfigByID(ctx, hook.ID)
	require.NoError(t, err)
	assert.Equal(t, models.APIConfigTypeWebhook, got.ConfigType)
	_, err = svc.GetAPINotificationConfigByID(ctx, uuid.New().String())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API通知配置不存在")

	// Create 第二条(同类型,IsDefault=true)→ 先清同类默认再插入(:191-195 分支)
	second := mk("企微机器人-2", models.APIConfigTypeWebhook, true)
	require.NoError(t, svc.CreateAPINotificationConfig(ctx, second))
	var hookDefaultCount int64
	require.NoError(t, db.Model(&models.APINotificationConfig{}).
		Where("config_type = ? AND is_default = ? AND del_flag = 0",
			models.APIConfigTypeWebhook, true).Count(&hookDefaultCount).Error)
	// 注:实现只在"更新"分支排除自身(:217),Create 分支(:192)不排除 → 两条都为默认
	// 是锁定现行为(第一条的默认会被第二条清掉,但第二条自身也是默认)。
	assert.Equal(t, int64(1), hookDefaultCount, "锁定现行为:Create 后同类默认唯一")

	// 第三条(不同类型,IsDefault=true)→ 不得影响 webhook 类默认
	sms := mk("短信通道", models.APIConfigTypeSMS, true)
	require.NoError(t, svc.CreateAPINotificationConfig(ctx, sms))
	require.NoError(t, db.Model(&models.APINotificationConfig{}).
		Where("config_type = ? AND is_default = ? AND del_flag = 0",
			models.APIConfigTypeWebhook, true).Count(&hookDefaultCount).Error)
	assert.Equal(t, int64(1), hookDefaultCount, "不同类型的默认互不干扰")

	// List:configType + status 过滤 + 分页
	list, total, err := svc.ListAPINotificationConfigs(ctx, 1, 10,
		func() *models.APIConfigType { v := models.APIConfigTypeWebhook; return &v }(), nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	smsType := models.APIConfigTypeSMS
	list, total, err = svc.ListAPINotificationConfigs(ctx, 1, 10, &smsType, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "短信通道", list[0].ConfigName)
	// status 过滤(停用)
	stopped := int(models.NotificationConfigStatusStopped)
	list, total, err = svc.ListAPINotificationConfigs(ctx, 1, 10, nil, &stopped)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	// 空页
	list, total, err = svc.ListAPINotificationConfigs(ctx, 9, 10, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Empty(t, list)

	// Update:改名 + IsDefault=true(同类排除自身 :217 分支)+ 不存在分支
	second.ConfigName = "企微机器人-2-改"
	second.IsDefault = true
	require.NoError(t, svc.UpdateAPINotificationConfig(ctx, second.ID, second))
	got, err = svc.GetAPINotificationConfigByID(ctx, second.ID)
	require.NoError(t, err)
	assert.Equal(t, "企微机器人-2-改", got.ConfigName)
	require.NoError(t, db.Model(&models.APINotificationConfig{}).
		Where("config_type = ? AND id != ? AND is_default = ? AND del_flag = 0",
			models.APIConfigTypeWebhook, second.ID, true).Count(&hookDefaultCount).Error)
	assert.Equal(t, int64(0), hookDefaultCount, "Update(IsDefault=true) 必须清同类其他默认")
	err = svc.UpdateAPINotificationConfig(ctx, uuid.New().String(), second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API通知配置不存在")

	// Delete:软删;不存在 → 错误
	require.NoError(t, svc.DeleteAPINotificationConfig(ctx, sms.ID))
	_, err = svc.GetAPINotificationConfigByID(ctx, sms.ID)
	require.Error(t, err)
	err = svc.DeleteAPINotificationConfig(ctx, uuid.New().String())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API通知配置不存在")
}
