// 一次性诊断工具：核对 AD 配置/账号池/密钥是否自洽。
// 只输出解密成败、长度与密钥指纹，绝不打印明文密码或密钥本身。
package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
)

func main() {
	_ = godotenv.Load(".env", "configs/.env")

	cfg, err := config.Load(context.Background())
	if err != nil {
		fmt.Printf("!! 配置加载失败: %v\n", err)
		os.Exit(1)
	}

	sum := sha256.Sum256([]byte(cfg.Security.SM4Key))
	keyBytes, _ := base64.StdEncoding.DecodeString(cfg.Security.SM4Key)
	fmt.Printf("SM4_KEY 指纹: %x  解码字节数: %d\n", sum[:6], len(keyBytes))
	if string(keyBytes) == "test-secret16!!!" {
		fmt.Println("!! 当前使用仓库默认 key（危险）")
	}

	cipher, err := crypto.NewSM4Cipher(cfg.Security.SM4Key)
	if err != nil {
		fmt.Printf("!! SM4 cipher 构造失败: %v\n", err)
		os.Exit(1)
	}
	// 模拟 core.Init 的注入顺序
	addomain.SetADSM4Cipher(cipher)

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("!! 数据库连接失败: %v\n", err)
		os.Exit(1)
	}

	// sys_config 中的认证相关配置
	var cfgs []models.Config
	db.Where("config_key IN ?", []string{"sys.auth.ad.config_id", "sys.auth.default.mode", "sys.auth.ad.default_role_id"}).Find(&cfgs)
	fmt.Println("\n[sys_config]")
	for _, c := range cfgs {
		v := c.ConfigValue
		if c.ConfigKey == "sys.auth.ad.default_role_id" {
			v = "(略)"
		}
		fmt.Printf("  %s = %q\n", c.ConfigKey, v)
	}

	var adCfgs []models.ADConfig
	db.Order("created_at").Find(&adCfgs)
	fmt.Printf("\n[sys_ad_config] 共 %d 行\n", len(adCfgs))
	for _, a := range adCfgs {
		fmt.Printf("  id=%s name=%q status=%d server=%s:%d domain=%s created=%s\n",
			a.ID, a.ConfigName, a.Status, a.ServerAddress, a.ServerPort, a.DomainName,
			a.CreatedAt.Format("2006-01-02 15:04"))
	}

	for _, adCfg := range adCfgs {
		var accts []models.ADServiceAccount
		db.Where("config_id = ?", adCfg.ID).Unscoped().Order("created_at").Find(&accts)
		fmt.Printf("\n[账号池] config=%s (%s) 共 %d 行\n", adCfg.ConfigName, adCfg.ID, len(accts))
		if len(accts) == 0 {
			fmt.Println("  !! 该配置下没有服务账号 —— admin_bind 必然失败（Phase 38 起无单管理员回退）")
		}
		for _, a := range accts {
			deleted := a.DeletedAt.Valid
			// 真实业务路径：decryptPassword（走全局 cipher + legacy 回退）
			plain := addomain.DecryptPassword(a.PasswordCiphertext)
			// 直接 SM4 路径（用于区分「key 不对」还是「全局 cipher 未注入」）
			direct, directErr := cipher.Decrypt(a.PasswordCiphertext)
			_ = direct

			verdict := "OK"
			switch {
			case plain == "":
				verdict = "!! 解出空密码 → LDAP 206 Empty password"
			case plain == a.PasswordCiphertext:
				verdict = "!! 解密失败被回退为密文"
			}

			breaker := "-"
			if a.CircuitBreakerUntil != nil {
				breaker = a.CircuitBreakerUntil.Format("01-02 15:04:05")
			}
			fmt.Printf("  - %-16s status=%d available=%-5v fail=%d breaker=%s deleted=%v\n",
				a.Username, a.Status, a.IsAvailable(), a.FailureCount, breaker, deleted)
			fmt.Printf("      updatedAt=%s 密文len=%d\n",
				a.UpdatedAt.Format("2006-01-02 15:04:05"), len(a.PasswordCiphertext))
			fmt.Printf("      DecryptPassword明文len=%d  directSM4=%s  => %s\n",
				len(plain), okStr(directErr), verdict)
			if a.LastFailureReason != "" {
				fmt.Printf("      lastFailure=%s\n", truncate(a.LastFailureReason, 140))
			}
		}
	}
}

func okStr(err error) string {
	if err == nil {
		return "成功"
	}
	return "失败: " + truncate(err.Error(), 60)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
