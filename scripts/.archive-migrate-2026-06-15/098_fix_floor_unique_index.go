//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	_ "github.com/lib/pq"
)

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getDB() (*gorm.DB, error) {
	host := getEnv("DB_HOST", "localhost")
	port := getEnvInt("DB_PORT", 5432)
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "")
	dbname := getEnv("DB_NAME", "xingran")

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("打开数据库连接失败: %w", err)
	}

	// 测试连接
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	log.Printf("数据库连接成功: %s@%s:%d/%s", user, host, port, dbname)
	return db, nil
}

func main() {
	db, err := getDB()
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("获取底层数据库连接失败: %v", err)
	}
	defer sqlDB.Close()

	// 删除部分索引
	fmt.Println("删除部分索引...")
	db.Exec(`DROP INDEX IF EXISTS idx_ops_floors_building_floor_unique`)

	// 创建完整的唯一索引（不带 WHERE 子句）
	fmt.Println("创建完整的唯一索引...")
	result := db.Exec(`
		CREATE UNIQUE INDEX idx_ops_floors_building_floor_unique
		ON ops_floors (building_id, floor_no)
		`)
	if result.Error != nil {
		log.Fatalf("创建唯一索引失败: %v", result.Error)
	}

	fmt.Println("楼层唯一索引迁移完成! (完整索引，不带 WHERE 子句)")
}
