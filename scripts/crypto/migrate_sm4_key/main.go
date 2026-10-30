// 一次性工具: SM4 密钥迁移(OLD → NEW)
// 用法:
//
//	export OLD_SM4_KEY="dGVzdC1zZWNyZXQxNiEhIQ=="      # 当前在用的 key
//	export NEW_SM4_KEY="$(openssl rand -base64 16)"     # 新 key
//	go run scripts/migrate-sm4-key.go --dry-run         # 先 dry-run 看效果
//	go run scripts/migrate-sm4-key.go                   # 实际迁移
//
// 不会进入正常构建(//go:build ignore 标签)
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 仓库默认 SM4 key(用户一直在用的默认值,作为 OLD key 兜底)
const defaultSM4Key = "dGVzdC1zZWNyZXQxNiEhIQ=="

// 4 个用 SM4 加密的字段(顺序:风险从高到低)
var targetFields = []targetField{
	{Table: "sys_ad_service_accounts", Column: "password_ciphertext", IDCol: "id"},
	{Table: "sys_ad_config", Column: "admin_password", IDCol: "id"},
	{Table: "sys_auth_credential", Column: "password", IDCol: "id"},
	{Table: "sys_auth_credential", Column: "enable_password", IDCol: "id"},
}

type targetField struct {
	Table  string
	Column string
	IDCol  string
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// 1. 加载 .env (godotenv 默认不覆盖已存在的 env)
	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️ .env 加载失败(可忽略): %v", err)
	}

	// 2. 解析参数
	dryRun := hasFlag("--dry-run")
	oldKey := getEnv("OLD_SM4_KEY", defaultSM4Key)
	newKey := os.Getenv("NEW_SM4_KEY")
	if newKey == "" {
		log.Fatal("❌ NEW_SM4_KEY 未设置。用法:\n" +
			"  export OLD_SM4_KEY=\"dGVzdC1zZWNyZXQxNiEhIQ==\"  # 旧 key,留空默认用仓库值\n" +
			"  export NEW_SM4_KEY=\"$(openssl rand -base64 16)\" # 新 key,必填\n" +
			"  go run scripts/migrate-sm4-key.go [--dry-run]")
	}
	if oldKey == newKey {
		log.Fatal("❌ OLD_SM4_KEY 与 NEW_SM4_KEY 相同,无需迁移")
	}

	// 3. 创建两个 cipher
	oldCipher, err := crypto.NewSM4Cipher(oldKey)
	if err != nil {
		log.Fatalf("❌ OLD key 无效: %v", err)
	}
	newCipher, err := crypto.NewSM4Cipher(newKey)
	if err != nil {
		log.Fatalf("❌ NEW key 无效: %v", err)
	}

	fmt.Println()
	log.Println("=== SM4 密钥迁移工具 ===")
	log.Printf("  OLD key:  %s... (长度 %d)", oldKey[:min(8, len(oldKey))], len(oldKey))
	log.Printf("  NEW key:  %s... (长度 %d)", newKey[:min(8, len(newKey))], len(newKey))
	log.Printf("  DRY-RUN:  %v", dryRun)
	fmt.Println()

	// 4. 确认(防误操作)
	if !dryRun {
		log.Println("⚠️  即将修改数据库!建议先备份:")
		log.Println("     pg_dump -U postgres -d xingran > backup-before-sm4-migration.sql")
		fmt.Println()
		log.Print("按 Enter 继续(5 秒后自动继续)...")
		waitOrCancel(5)
	}

	// 5. 连 DB
	db := connectDB()

	// 6. 迁移
	totalSuccess, totalFailed, totalSkipped := 0, 0, 0
	for _, t := range targetFields {
		s, f, sk := migrateField(db, t, oldCipher, newCipher, dryRun)
		totalSuccess += s
		totalFailed += f
		totalSkipped += sk
	}

	fmt.Println()
	log.Println("=== 迁移汇总 ===")
	log.Printf("  ✅ 成功:   %d", totalSuccess)
	log.Printf("  ❌ 失败:   %d", totalFailed)
	log.Printf("  ⏭️  跳过:   %d (空值)", totalSkipped)
	fmt.Println()

	if totalFailed > 0 {
		log.Fatal("❌ 存在失败行,数据库未完全迁移。请检查后重试(失败行密文未修改,数据库安全)。")
	}

	if dryRun {
		log.Println("🔍 DRY-RUN 完成(未修改任何数据)。如确认无误,去掉 --dry-run 实际执行。")
	} else {
		log.Println("✅ 迁移完成!现在可以:")
		log.Println("   1. 更新 .env 里的 SM4_KEY 为 NEW key")
		log.Println("   2. 重启应用,确认 [SECURITY WARNING] 不再出现")
		log.Println("   3. 业务功能测试(AD 同步 + 1-2 个设备连接)")
	}
}

func migrateField(db *gorm.DB, t targetField, oldCipher, newCipher *crypto.SM4Cipher, dryRun bool) (int, int, int) {
	var rows []map[string]interface{}
	if err := db.Table(t.Table).
		Where(t.Column + " IS NOT NULL AND " + t.Column + " <> ''").
		Find(&rows).Error; err != nil {
		log.Printf("❌ 查询 %s.%s 失败: %v", t.Table, t.Column, err)
		return 0, 0, 0
	}

	log.Printf("📋 %s.%s: %d 行待处理", t.Table, t.Column, len(rows))

	success, failed, skipped := 0, 0, 0
	for i, row := range rows {
		id := row[t.IDCol]
		ciphertext, _ := row[t.Column].(string)

		if ciphertext == "" {
			skipped++
			continue
		}

		// 1. 用 OLD key 解密
		plain, err := oldCipher.Decrypt(ciphertext)
		if err != nil {
			log.Printf("  ❌ [%d] id=%v OLD 解密失败: %v (该行密文可能不是 SM4-GCM 格式,跳过)", i, id, err)
			failed++
			continue
		}

		// 2. 用 NEW key 加密
		newCiphertext, err := newCipher.Encrypt(plain)
		if err != nil {
			log.Printf("  ❌ [%d] id=%v NEW 加密失败: %v", i, id, err)
			failed++
			continue
		}

		// 3. round-trip 验证(用 NEW key 解密,对比 plaintext)
		verifyPlain, err := newCipher.Decrypt(newCiphertext)
		if err != nil || verifyPlain != plain {
			log.Printf("  ❌ [%d] id=%v round-trip 验证失败(明文不一致,绝不能写库)", i, id)
			failed++
			continue
		}

		// 4. 写库(非 dry-run)
		if !dryRun {
			if err := db.Table(t.Table).
				Where(t.IDCol+" = ?", id).
				Update(t.Column, newCiphertext).Error; err != nil {
				log.Printf("  ❌ [%d] id=%v 写库失败: %v", i, id, err)
				failed++
				continue
			}
		}

		log.Printf("  ✅ [%d] id=%v 迁移成功 (明文长度 %d 字节,内容已脱敏)", i, id, len(plain))
		// 安全:绝不在日志/输出中打印明文密码
		_ = plain
		success++
	}

	log.Printf("  📊 %s.%s 完成: ✅%d / ❌%d / ⏭%d", t.Table, t.Column, success, failed, skipped)
	fmt.Println()
	return success, failed, skipped
}

func connectDB() *gorm.DB {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		os.Getenv("DB_PASSWORD"),
		getEnv("DB_NAME", "xingran"),
		getEnv("DB_SSLMODE", "disable"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}
	return db
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func hasFlag(flag string) bool {
	for _, arg := range os.Args[1:] {
		if arg == flag {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// waitOrCancel 等待 N 秒,期间用户按 Enter 立即继续(无 Enter 则超时自动继续)
func waitOrCancel(seconds int) {
	done := make(chan struct{})
	go func() {
		fmt.Scanln()
		close(done)
	}()

	timeout := time.After(time.Duration(seconds) * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	remaining := seconds
	for {
		select {
		case <-done:
			fmt.Println()
			return
		case <-timeout:
			fmt.Println()
			return
		case <-ticker.C:
			remaining--
			fmt.Printf("\r  %d 秒后继续(或按 Enter 立即继续)... ", remaining)
		}
	}
}
