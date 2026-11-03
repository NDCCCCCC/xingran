// 一次性诊断工具：直接用账号池里的账号调 LDAPClient.Connect()，复现真实 bind 路径。
// 不经过 AccountPool（不写 fail/success 计数、不触发熔断），只观察 bind 结果。
package main

import (
	"context"
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
	cipher, err := crypto.NewSM4Cipher(cfg.Security.SM4Key)
	if err != nil {
		fmt.Printf("!! SM4 cipher 构造失败: %v\n", err)
		os.Exit(1)
	}
	addomain.SetADSM4Cipher(cipher) // 同 core.Init 顺序

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("!! 数据库连接失败: %v\n", err)
		os.Exit(1)
	}

	var adCfg models.ADConfig
	if err := db.First(&adCfg).Error; err != nil {
		fmt.Printf("!! 读取 AD 配置失败: %v\n", err)
		os.Exit(1)
	}

	var accts []models.ADServiceAccount
	db.Where("config_id = ?", adCfg.ID).Order("created_at").Find(&accts)

	for i := range accts {
		a := &accts[i]
		fmt.Printf("\n=== 测试账号 %s (status=%d) ===\n", a.Username, a.Status)

		// 先看解密结果（真实业务路径 decryptPassword）
		plain := addomain.DecryptPassword(a.PasswordCiphertext)
		fmt.Printf("decryptPassword 明文长度: %d\n", len(plain))
		if plain == "" {
			fmt.Println("!! 解密为空 → 必然 LDAP 206 Empty password，不再尝试 bind")
			continue
		}

		client := addomain.NewLDAPClient(&adCfg, a)
		err := client.Connect()
		if err != nil {
			fmt.Printf("bind 结果: 失败 → %v\n", err)
		} else {
			fmt.Println("bind 结果: 成功")
			client.Close()
		}
	}
}
