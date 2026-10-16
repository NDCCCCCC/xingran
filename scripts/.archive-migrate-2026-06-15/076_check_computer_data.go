//go:build ignore
// +build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	database := getDB()
	defer database.Close()

	fmt.Println("=== AD 域配置 ===")
	var configID string
	err := database.QueryRow("SELECT id FROM sys_ad_config WHERE config_name = 'AD域控主机' LIMIT 1").Scan(&configID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("未找到 'AD域控主机' 配置")
		} else {
			log.Printf("查询配置失败: %v", err)
		}
		return
	}
	fmt.Printf("AD配置ID: %s\n", configID)

	fmt.Println("\n=== 电脑设备统计 ===")
	var totalCount int
	err = database.QueryRow("SELECT COUNT(*) FROM sys_ad_computer WHERE ad_config_id = $1 AND deleted_at IS NULL", configID).Scan(&totalCount)
	if err != nil {
		log.Printf("统计失败: %v", err)
		return
	}
	fmt.Printf("总设备数: %d\n", totalCount)

	fmt.Println("\n=== 检查 computer_name 重复 ===")
	query := `
		SELECT computer_name, COUNT(*) as count,
			STRING_AGG(distinguished_name, ', ') as dns
		FROM sys_ad_computer
		WHERE ad_config_id = $1 AND deleted_at IS NULL
		GROUP BY computer_name
		HAVING COUNT(*) > 1
		ORDER BY count DESC
		LIMIT 10;
	`

	rows, err := database.Query(query, configID)
	if err != nil {
		log.Printf("查询重复失败: %v", err)
		return
	}
	defer rows.Close()

	hasDuplicates := false
	for rows.Next() {
		hasDuplicates = true
		var computerName string
		var count int
		var dns string
		if err := rows.Scan(&computerName, &count, &dns); err != nil {
			log.Printf("扫描行失败: %v", err)
			continue
		}
		fmt.Printf("重复: %s (数量: %d)\nDNs: %s\n\n", computerName, count, dns)
	}

	if !hasDuplicates {
		fmt.Println("没有发现 computer_name 重复")
	}

	fmt.Println("\n=== 最近添加的设备 ===")
	rows2, err := database.Query(`
		SELECT computer_name, distinguished_name, created_at
		FROM sys_ad_computer
		WHERE ad_config_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 5
	`, configID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var computerName, dn, createdAt string
			if err := rows2.Scan(&computerName, &dn, &createdAt); err == nil {
				fmt.Printf("- %s: %s\n", computerName, dn)
			}
		}
	}
}

func getDB() *sql.DB {
	host := getEnv("DB_HOST", "localhost")
	port := getEnvInt("DB_PORT", 5432)
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "")
	dbname := getEnv("DB_NAME", "xingran")

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("打开数据库连接失败: %v", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		log.Fatalf("数据库连接测试失败: %v", err)
	}

	log.Printf("数据库连接成功: %s@%s:%d/%s", user, host, port, dbname)
	return db
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var num int
		if _, err := fmt.Sscanf(value, "%d", &num); err == nil {
			return num
		}
	}
	return defaultValue
}
