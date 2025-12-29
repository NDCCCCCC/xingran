package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 检查密码是否是加密格式（Base64编码）
func isEncryptedPassword(password string) bool {
	if password == "" {
		return true // 空密码视为已加密
	}

	// 尝试Base64解码
	_, err := parseBase64Length(password)
	return err == nil
}

// parseBase64Length 尝试解析Base64并返回解码后的长度
func parseBase64Length(s string) (int, error) {
	// 简单检查：有效的Base64编码后长度应该是4的倍数
	// 解码后的长度大约是编码后的3/4
	if len(s) == 0 || len(s)%4 != 0 {
		return 0, fmt.Errorf("invalid base64 length")
	}

	// 检查字符集
	for _, c := range s {
		if !isValidBase64Char(c) {
			return 0, fmt.Errorf("invalid base64 character: %c", c)
		}
	}

	return len(s) * 3 / 4, nil
}

func isValidBase64Char(c rune) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '='
}

func main() {
	// 数据库连接配置
	dbHost := getEnv("DB_HOST", "postgres")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "ruoyi")
	dbPass := getEnv("DB_PASSWORD", "ruoyi123")
	dbName := getEnv("DB_NAME", "ruoyi_next")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)

	log.Printf("连接数据库: %s@%s:%s/%s", dbUser, dbHost, dbPort, dbName)

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 创建 SM4 加密器
	sm4Cipher, err := security.NewSM4Cipher("dGVzdC1zZWNyZXQxNiEhIQ==") // 使用默认密钥
	if err != nil {
		log.Fatalf("创建 SM4 加密器失败: %v", err)
	}

	log.Println("========================================")
	log.Println("检查凭证密码加密状态")
	log.Println("========================================")

	// 查询所有凭证
	var credentials []models.AuthCredential
	if err := db.Find(&credentials).Error; err != nil {
		log.Fatalf("查询凭证失败: %v", err)
	}

	log.Printf("找到 %d 个凭证\n", len(credentials))

	plaintextCount := 0
	encryptedCount := 0
	var toEncrypt []models.AuthCredential

	for i := range credentials {
		cred := credentials[i]
		isEncrypted := isEncryptedPassword(cred.Password)
		enableEncrypted := isEncryptedPassword(cred.EnablePassword)

		if !isEncrypted || !enableEncrypted {
			plaintextCount++
			toEncrypt = append(toEncrypt, cred)
			log.Printf("\n[%d] %s - 需要加密", i+1, cred.CredentialName)
			if !isEncrypted {
				log.Printf("    密码: 明文 (长度=%d)", len(cred.Password))
			}
			if !enableEncrypted && cred.EnablePassword != "" {
				log.Printf("    特权密码: 明文 (长度=%d)", len(cred.EnablePassword))
			}
		} else {
			encryptedCount++
			log.Printf("\n[%d] %s - 已加密", i+1, cred.CredentialName)
		}
	}

	log.Println("\n========================================")
	log.Printf("统计结果:")
	log.Printf("  已加密: %d", encryptedCount)
	log.Printf("  明文:   %d", plaintextCount)
	log.Println("========================================")

	// 如果有明文密码，询问是否加密
	if plaintextCount > 0 {
		log.Printf("\n发现 %d 个凭证使用明文密码", plaintextCount)
		log.Println("是否要加密这些明文密码？")
		log.Println("请手动运行以下SQL或使用加密工具更新密码")
		log.Println("")
		log.Println("加密命令示例（Go）:")
		log.Println("  encrypted, _ := sm4Cipher.Encrypt(\"your_password\")")
		log.Println("  UPDATE sys_auth_credential SET password = 'encrypted_value' WHERE id = 'credential_id';")
		log.Println("")

		// 生成加密SQL
		log.Println("生成的加密SQL:")
		log.Println("========================================")
		for _, cred := range toEncrypt {
			if !isEncryptedPassword(cred.Password) && cred.Password != "" {
				encrypted, err := sm4Cipher.Encrypt(cred.Password)
				if err != nil {
					log.Printf("-- 加密 %s 的密码失败: %v\n", cred.CredentialName, err)
					continue
				}
				log.Printf("-- %s (密码)\n", cred.CredentialName)
				log.Printf("UPDATE sys_auth_credential SET password = '%s' WHERE id = '%s';\n", encrypted, cred.ID)
			}
			if !isEncryptedPassword(cred.EnablePassword) && cred.EnablePassword != "" {
				encrypted, err := sm4Cipher.Encrypt(cred.EnablePassword)
				if err != nil {
					log.Printf("-- 加密 %s 的特权密码失败: %v\n", cred.CredentialName, err)
					continue
				}
				log.Printf("-- %s (特权密码)\n", cred.CredentialName)
				log.Printf("UPDATE sys_auth_credential SET enable_password = '%s' WHERE id = '%s';\n", encrypted, cred.ID)
			}
		}
		log.Println("========================================")
	} else {
		log.Println("\n✅ 所有凭证密码已加密，无需处理")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
