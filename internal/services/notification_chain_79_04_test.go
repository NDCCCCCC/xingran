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

// -------------------------------------------------------------------------
// Task 5: notification_sender(派发链,真实发送路径归 79-05)
// -------------------------------------------------------------------------

// newNsn7904 装配 NotificationSenderService + sqlite(t.TempDir 文件库)+ 通知链 model。
// sys_notification_channel 不能走 AutoMigrate(NotificationChannel.ID 带 PG 专属
// default:gen_random_uuid(),sqlite 缺该函数 → 79-03 同款缺表 family pattern)→ 裸表。
// sender 装配用 NewNotificationSenderService(真实 EmailSender/APISender,同库),
// 所有断言路径都停在不出网的前置校验分支;需要 nil-sender 的用例用 nsn7904NilSender。
func newNsn7904(t *testing.T) (*NotificationSenderService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "nsn7904.db")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "open sqlite temp db")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.AutoMigrate(
		&models.Notice{},
		&models.NoticeTarget{},
		&models.NoticeRead{},
		&models.User{},
		&models.EmailConfig{},
		&models.APINotificationConfig{},
	), "auto migrate notice chain models")
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_notification_channel (
		id TEXT PRIMARY KEY,
		notice_id TEXT NOT NULL,
		channel_type TEXT NOT NULL,
		email_config_id TEXT,
		api_config_id TEXT,
		custom_recipients TEXT,
		created_at DATETIME
	)`).Error, "create sanitized sys_notification_channel table")
	return NewNotificationSenderService(db), db
}

// nsn7904NilSender nil 依赖装配(emailSender/apiSender 均为 nil)。
func nsn7904NilSender(db *gorm.DB) *NotificationSenderService {
	return &NotificationSenderService{db: db}
}

// nsn7904User 预置用户(email/phone 可为 nil)。
func nsn7904User(t *testing.T, db *gorm.DB, username string, email, phone *string) models.User {
	t.Helper()
	u := models.User{
		Username: username,
		Password: "x",
		Salt:     "s",
		Email:    email,
		Phone:    phone,
	}
	require.NoError(t, db.Create(&u).Error, "seed user %s", username)
	return u
}

// nsn7904Notice 预置指定发布状态、TargetUser 型接收范围的通知(无渠道)。
func nsn7904Notice(t *testing.T, db *gorm.DB, title string, status models.PublishStatus,
	userIDs ...string) *models.Notice {
	t.Helper()
	notice := &models.Notice{
		NoticeTitle:   title,
		NoticeType:    "1",
		NoticeContent: "7904 通知内容",
		TargetType:    models.TargetUser,
		PublishStatus: status,
		CreatedByName: "creator-7904",
	}
	require.NoError(t, db.Create(notice).Error, "seed notice %s", title)
	for _, uid := range userIDs {
		require.NoError(t, db.Create(&models.NoticeTarget{
			NoticeID:   notice.ID,
			TargetType: "user",
			TargetID:   uid,
		}).Error, "seed notice target")
	}
	return notice
}

// nsn7904Channel 构造渠道行(未落库,SetNotificationChannels 负责落库)。
func nsn7904Channel(channelType models.NotificationChannelType,
	emailConfigID, apiConfigID *string, recipients ...string) models.NotificationChannel {
	ch := models.NotificationChannel{
		ChannelType:   channelType,
		EmailConfigID: emailConfigID,
		APIConfigID:   apiConfigID,
	}
	if recipients != nil {
		ch.CustomRecipients = &recipients
	}
	return ch
}

// TestNsn7904_SendNotification_NoChannels 无渠道 → 走默认站内信(不发送不报错);
// notice 不存在 / 非发布态(isValidPublishStatus=false)→ 错误分支。
func TestNsn7904_SendNotification_NoChannels(t *testing.T) {
	ctx := context.Background()
	svc, db := newNsn7904(t)

	u := nsn7904User(t, db, "nochan-user", nil, nil)
	notice := nsn7904Notice(t, db, "无渠道通知", models.PublishStatusPublished, u.ID)

	// 无渠道 → sendWebNotification 兜底(仅日志),整体 nil
	require.NoError(t, svc.SendNotification(ctx, notice.ID))

	// notice 不存在 → 错误
	err := svc.SendNotification(ctx, uuid.New().String())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "获取通知失败")

	// 草稿态 → isValidPublishStatus=false → 错误分支
	draft := nsn7904Notice(t, db, "草稿通知", models.PublishStatusDraft, u.ID)
	err = svc.SendNotification(ctx, draft.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "通知未发布或状态无效")

	// 已撤回态 → 同样拒绝
	withdrawn := nsn7904Notice(t, db, "撤回通知", models.PublishStatusWithdrawn, u.ID)
	err = svc.SendNotification(ctx, withdrawn.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "通知未发布或状态无效")
}

// TestNsn7904_IsValidPublishStatus 表驱动:仅已发布/定时发布中有效(具名常量)。
func TestNsn7904_IsValidPublishStatus(t *testing.T) {
	cases := []struct {
		status models.PublishStatus
		want   bool
	}{
		{models.PublishStatusDraft, false},
		{models.PublishStatusPublished, true},
		{models.PublishStatusScheduled, true},
		{models.PublishStatusWithdrawn, false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, isValidPublishStatus(tc.status), "status=%d", int(tc.status))
	}
}

// TestNsn7904_Send_EmailChannel_NilSender nil EmailSender 下 email 渠道的两条安全分支:
//   - 渠道未指定 config 且库中无默认配置 → getEmailConfigID 报错被捕获(不触达 sender);
//   - 有默认配置但所有目标用户无邮箱 → buildRecipientList 为空 → Warnf 早退(不触达 sender)。
//     两条路径整体都必须返回 nil 且不 panic(锁定派发层吞错语义)。
func TestNsn7904_Send_EmailChannel_NilSender(t *testing.T) {
	ctx := context.Background()
	svc, db := newNsn7904(t)
	nilSvc := nsn7904NilSender(db)

	u := nsn7904User(t, db, "nilmail-user", nil, nil)
	notice := nsn7904Notice(t, db, "nil-sender 邮件", models.PublishStatusPublished, u.ID)
	ch := nsn7904Channel(models.ChannelTypeEmail, nil, nil)
	require.NoError(t, svc.SetNotificationChannels(ctx, notice.ID, []models.NotificationChannel{ch}))

	// 分支 1:无默认邮件配置 → getEmailConfigID 错误分支
	require.NoError(t, nilSvc.SendNotification(ctx, notice.ID), "nil-sender 错误必须被捕获,不 panic")

	// 分支 2:有默认配置但无有效邮箱 → 收件人为空早退
	def := ncf7904EmailConfig("默认邮箱", "smtp.example.com", int(models.NotificationConfigStatusNormal))
	require.NoError(t, db.Create(def).Error)
	require.NoError(t, nilSvc.SendNotification(ctx, notice.ID), "空收件人早退同样不触达 nil sender")

	// 收件人非空 + nil sender 会触达 Send 的 s.db 解引用(必 panic),属"nil 不可达"边界:
	// 该组合改由真实 sender + 出网前终止的路径覆盖(见 TestNsn7904_Send_EmailChannel_ConfigFailure)。
}

// TestNsn7904_Send_EmailChannel_ConfigFailure 真实 sender + 出网前终止的两条路径:
//   - 渠道显式指向停用配置 → Send 返回 "邮件配置未启用";
//   - 渠道未指定 config → 默认配置(状态正常)但密码非合法密文 → Send 返回 "密码解密失败"。
//     两条都覆盖 sendEmailNotification 的 result.Success=false 日志分支且不出网。
func TestNsn7904_Send_EmailChannel_ConfigFailure(t *testing.T) {
	ctx := context.Background()
	svc, db := newNsn7904(t)

	mail := "user@example.com"
	u := nsn7904User(t, db, "mailfail-user", &mail, nil)
	notice := nsn7904Notice(t, db, "邮件失败派发", models.PublishStatusPublished, u.ID)

	// 路径 1:显式停用配置
	stopped := ncf7904EmailConfig("停用邮箱", "smtp.example.com", int(models.NotificationConfigStatusStopped))
	require.NoError(t, db.Create(stopped).Error)
	require.NoError(t, svc.SetNotificationChannels(ctx, notice.ID,
		[]models.NotificationChannel{nsn7904Channel(models.ChannelTypeEmail, &stopped.ID, nil, "ops@example.com")}))
	require.NoError(t, svc.SendNotification(ctx, notice.ID), "发送失败只记日志,不向上报错")

	// 路径 2:默认配置 + 密码不可解密(出网前终止)
	broken := ncf7904EmailConfig("坏密码邮箱", "smtp.example.com", int(models.NotificationConfigStatusNormal))
	broken.Password = "not-a-valid-ciphertext"
	require.NoError(t, db.Create(broken).Error)
	require.NoError(t, svc.SetNotificationChannels(ctx, notice.ID,
		[]models.NotificationChannel{nsn7904Channel(models.ChannelTypeEmail, nil, nil, "ops@example.com")}))
	require.NoError(t, svc.SendNotification(ctx, notice.ID))
}

// TestNsn7904_Send_ApiChannel_NilSender nil APISender 下 api 渠道的安全分支(APIConfigID
// 为 nil → 早退);带 config 的真实 sender 走"配置不存在"分支(不出网)。
func TestNsn7904_Send_ApiChannel_NilSender(t *testing.T) {
	ctx := context.Background()
	svc, db := newNsn7904(t)
	nilSvc := nsn7904NilSender(db)

	u := nsn7904User(t, db, "api-user", nil, nil)
	notice := nsn7904Notice(t, db, "api 渠道", models.PublishStatusPublished, u.ID)

	// 分支 1:APIConfigID == nil → sendAPINotification 早退(nil sender 不被触达)
	ch := nsn7904Channel(models.ChannelTypeAPI, nil, nil)
	require.NoError(t, svc.SetNotificationChannels(ctx, notice.ID, []models.NotificationChannel{ch}))
	require.NoError(t, nilSvc.SendNotification(ctx, notice.ID))

	// 分支 2:APIConfigID 指向不存在配置 + 真实 sender → "获取API配置失败" 日志分支
	ch2 := nsn7904Channel(models.ChannelTypeAPI, nil, ndv7904StrPtr(uuid.New().String()))
	require.NoError(t, svc.SetNotificationChannels(ctx, notice.ID, []models.NotificationChannel{ch2}))
	require.NoError(t, svc.SendNotification(ctx, notice.ID))
}

// TestNsn7904_Send_SmsAndWebChannels web 渠道直返 nil;sms 渠道 APIConfigID=nil 早退;
// sms 带 config 且配置停用 → SendSMS "API配置未启用"(不出网)。
func TestNsn7904_Send_SmsAndWebChannels(t *testing.T) {
	ctx := context.Background()
	svc, db := newNsn7904(t)

	phone := "13800000000"
	u := nsn7904User(t, db, "sms-user", nil, &phone)
	notice := nsn7904Notice(t, db, "多渠道派发", models.PublishStatusPublished, u.ID)

	// web + sms(无 config)组合 → 全部走非出网分支
	require.NoError(t, svc.SetNotificationChannels(ctx, notice.ID, []models.NotificationChannel{
		nsn7904Channel(models.ChannelTypeWeb, nil, nil),
		nsn7904Channel(models.ChannelTypeSMS, nil, nil),
	}))
	require.NoError(t, svc.SendNotification(ctx, notice.ID))

	// sms 指向停用配置 → SendSMS 返回失败结果,只记日志
	stopped := &models.APINotificationConfig{
		ID:         uuid.New().String(),
		ConfigName: "停用短信",
		ConfigType: models.APIConfigTypeSMS,
		APIURL:     "https://sms.example.com/send",
		APIMethod:  "POST",
		Status:     int(models.NotificationConfigStatusStopped),
	}
	require.NoError(t, db.Create(stopped).Error)
	require.NoError(t, svc.SetNotificationChannels(ctx, notice.ID, []models.NotificationChannel{
		nsn7904Channel(models.ChannelTypeSMS, nil, &stopped.ID),
	}))
	require.NoError(t, svc.SendNotification(ctx, notice.ID), "发送失败只记日志,不向上报错")
}

// TestNsn7904_BuildRecipientList customRecipients 优先/用户列表展开/合并去重/空入参。
func TestNsn7904_BuildRecipientList(t *testing.T) {
	svc := &NotificationSenderService{}

	// customRecipients == nil → 原样返回用户列表
	users := []string{"u1", "u2"}
	assert.Equal(t, users, svc.buildRecipientList(users, nil))
	// customRecipients 指向空切片 → 同样原样返回
	empty := []string{}
	assert.Equal(t, users, svc.buildRecipientList(users, &empty))
	// 合并去重(保序:先用户后自定义)
	custom := []string{"c1", "u1", "c1"}
	got := svc.buildRecipientList(users, &custom)
	assert.Equal(t, []string{"u1", "u2", "c1"}, got)
	// 用户列表为空 → 只剩自定义
	got = svc.buildRecipientList(nil, &custom)
	assert.Equal(t, []string{"c1", "u1"}, got)
	// 双空 → 空(实现原样返回空 userItems)
	assert.Empty(t, svc.buildRecipientList(nil, &empty))
}

// TestNsn7904_GetUserInfo_Emails_Phones 预置 3 用户(有邮箱无手机/有手机无邮箱/全无)
// → getUserInfo map 形态 + getUserEmails/getUserPhones 空值剔除。
func TestNsn7904_GetUserInfo_Emails_Phones(t *testing.T) {
	ctx := context.Background()
	svc, db := newNsn7904(t)

	mail := "mail@example.com"
	phoneValue := "13900000000"
	mailOnly := nsn7904User(t, db, "mail-only", &mail, nil)
	phoneOnly := nsn7904User(t, db, "phone-only", nil, &phoneValue)
	neither := nsn7904User(t, db, "neither", nil, nil)

	info, err := svc.getUserInfo(ctx, []string{mailOnly.ID, phoneOnly.ID, neither.ID})
	require.NoError(t, err)
	assert.Len(t, info, 3)
	assert.Equal(t, mail, info[mailOnly.ID].email)
	assert.Empty(t, info[mailOnly.ID].phone)
	assert.Equal(t, phoneValue, info[phoneOnly.ID].phone)
	assert.Empty(t, info[neither.ID].email)
	assert.Equal(t, mailOnly.ID, info[mailOnly.ID].userID)

	// 邮箱/手机号列表:空值剔除
	assert.Equal(t, []string{mail}, svc.getUserEmails(info))
	assert.Equal(t, []string{phoneValue}, svc.getUserPhones(info))

	// 空入参 → 空 map / 空列表(不报错)
	emptyInfo, err := svc.getUserInfo(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, emptyInfo)
	assert.Empty(t, svc.getUserEmails(nil))
	assert.Empty(t, svc.getUserPhones(nil))
}

// TestNsn7904_GetEmailConfigID 渠道指定 configID / 未指定走默认配置 / 默认缺失错误三分支。
func TestNsn7904_GetEmailConfigID(t *testing.T) {
	ctx := context.Background()
	svc, db := newNsn7904(t)

	// 分支 1:渠道指定 configID → 直接返回
	explicit := "explicit-config-id"
	got, err := svc.getEmailConfigID(ctx, &models.NotificationChannel{EmailConfigID: &explicit})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, explicit, *got)

	// 分支 2:未指定 → 走默认配置(status normal + del_flag=0)
	def := ncf7904EmailConfig("默认配置", "smtp.example.com", int(models.NotificationConfigStatusNormal))
	require.NoError(t, db.Create(def).Error)
	got, err = svc.getEmailConfigID(ctx, &models.NotificationChannel{})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, def.ID, *got)

	// 分支 3:无默认配置 → 错误
	require.NoError(t, db.Where("id = ?", def.ID).Delete(&models.EmailConfig{}).Error)
	_, err = svc.getEmailConfigID(ctx, &models.NotificationChannel{})
	require.Error(t, err)
}

// TestNsn7904_PublishAndSendNotice 发布+发送复合链(发布态断言 models.PublishStatusPublished);
// 重复发布按现行为报错(实现无幂等,锁定);notice 不存在 → 错误。
func TestNsn7904_PublishAndSendNotice(t *testing.T) {
	ctx := context.Background()
	svc, db := newNsn7904(t)

	u := nsn7904User(t, db, "publish-user", nil, nil)
	notice := nsn7904Notice(t, db, "发布并发送", models.PublishStatusDraft, u.ID)

	// 草稿 → 发布 + 发送(无渠道 → 站内信兜底)
	require.NoError(t, svc.PublishAndSendNotice(ctx, notice.ID))
	var got models.Notice
	require.NoError(t, db.Where("id = ?", notice.ID).First(&got).Error)
	assert.Equal(t, models.PublishStatusPublished, got.PublishStatus)
	assert.NotNil(t, got.PublishTime, "手动发布必须回填发布时间")

	// 重复调用:PublishNotice 拒绝已发布(锁定现行为:非幂等)
	err := svc.PublishAndSendNotice(ctx, notice.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "已经发布")

	// 不存在 → 错误
	err = svc.PublishAndSendNotice(ctx, uuid.New().String())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "发布通知失败")
}

// TestNsn7904_SetGetNotificationChannels Set 后 Get 读回一致(通道类型/配置 ID/自定义收件人);
// 空集合 → 仅清空旧配置(空集合覆盖语义)。
func TestNsn7904_SetGetNotificationChannels(t *testing.T) {
	ctx := context.Background()
	svc, db := newNsn7904(t)

	notice := nsn7904Notice(t, db, "渠道配置", models.PublishStatusPublished)
	emailID := "email-config-1"
	apiID := "api-config-1"

	channels := []models.NotificationChannel{
		nsn7904Channel(models.ChannelTypeEmail, &emailID, nil, "ops@example.com", "lead@example.com"),
		nsn7904Channel(models.ChannelTypeAPI, nil, &apiID),
	}
	require.NoError(t, svc.SetNotificationChannels(ctx, notice.ID, channels))

	got, err := svc.GetNotificationChannels(ctx, notice.ID)
	require.NoError(t, err)
	require.Len(t, got, 2)
	byType := map[string]models.NotificationChannel{}
	for _, ch := range got {
		byType[string(ch.ChannelType)] = ch
		assert.Equal(t, notice.ID, ch.NoticeID, "Set 必须回填 NoticeID")
	}
	require.Contains(t, byType, string(models.ChannelTypeEmail))
	assert.Equal(t, emailID, *byType[string(models.ChannelTypeEmail)].EmailConfigID)
	require.NotNil(t, byType[string(models.ChannelTypeEmail)].CustomRecipients)
	assert.Equal(t, []string{"ops@example.com", "lead@example.com"},
		*byType[string(models.ChannelTypeEmail)].CustomRecipients)
	require.Contains(t, byType, string(models.ChannelTypeAPI))
	assert.Equal(t, apiID, *byType[string(models.ChannelTypeAPI)].APIConfigID)

	// 覆盖写:Set 先删旧配置再写入
	replaced := []models.NotificationChannel{nsn7904Channel(models.ChannelTypeWeb, nil, nil)}
	require.NoError(t, svc.SetNotificationChannels(ctx, notice.ID, replaced))
	got, err = svc.GetNotificationChannels(ctx, notice.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, models.ChannelTypeWeb, got[0].ChannelType)

	// 空集合 → 只删不建
	require.NoError(t, svc.SetNotificationChannels(ctx, notice.ID, nil))
	got, err = svc.GetNotificationChannels(ctx, notice.ID)
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestNsn7904_UniqueStrings 表驱动去重保序(包级纯函数)。
func TestNsn7904_UniqueStrings(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "空切片", in: []string{}, want: []string{}},
		{name: "nil", in: nil, want: []string{}},
		{name: "无重复保序", in: []string{"a", "b", "c"}, want: []string{"a", "b", "c"}},
		{name: "重复去重保序", in: []string{"a", "b", "a", "c", "b"}, want: []string{"a", "b", "c"}},
		{name: "全重复", in: []string{"x", "x", "x"}, want: []string{"x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, uniqueStrings(tc.in))
		})
	}
}
