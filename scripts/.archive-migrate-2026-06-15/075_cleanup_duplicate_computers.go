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

	fmt.Println("=== 查找重复的电脑设备 ===")

	query := `
		SELECT ad_config_id, computer_name, COUNT(*) as count
		FROM sys_ad_computer
		WHERE deleted_at IS NULL
		GROUP BY ad_config_id, computer_name
		HAVING COUNT(*) > 1
		ORDER BY count DESC;
	`

	rows, err := database.Query(query)
	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}
	defer rows.Close()

	duplicateCount := 0
	for rows.Next() {
		var configID, computerName string
		var count int
		if err := rows.Scan(&configID, &computerName, &count); err != nil {
			log.Printf("扫描行失败: %v", err)
			continue
		}
		duplicateCount++
		fmt.Printf("发现重复: AD配置=%s, 计算机名=%s, 数量=%d\n", configID, computerName, count)
	}

	if duplicateCount == 0 {
		fmt.Println("没有发现重复数据")
		return
	}

	fmt.Printf("\n共发现 %d 组重复数据\n", duplicateCount)
	fmt.Println("开始清理重复数据（保留最新的一条）...")

	deleteQuery := `
		DELETE FROM sys_ad_computer
		WHERE id IN (
			SELECT id FROM (
				SELECT id,
					ROW_NUMBER() OVER (PARTITION BY ad_config_id, computer_name ORDER BY created_at DESC) as rn
				FROM sys_ad_computer
				WHERE deleted_at IS NULL
			) t
			WHERE rn > 1
		);
	`

	result, err := database.Exec(deleteQuery)
	if err != nil {
		log.Fatalf("清理失败: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("清理完成！删除了 %d 条重复记录\n", rowsAffected)
	fmt.Println("现在可以重新同步 AD 域电脑设备了")
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
