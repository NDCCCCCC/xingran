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

	// 首先删除重复数据
	deleteDuplicatesSQL := `
	WITH duplicate_floors AS (
		SELECT
			id,
			building_id,
			floor_no,
			ROW_NUMBER() OVER (PARTITION BY building_id, floor_no ORDER BY created_at ASC) as row_num
		FROM ops_floors
		WHERE deleted_at IS NULL
	)
	DELETE FROM ops_floors
	WHERE id IN (
		SELECT id FROM duplicate_floors WHERE row_num > 1
	)`
	result := db.Exec(deleteDuplicatesSQL)
	if result.Error != nil {
		log.Printf("警告: 删除重复数据失败: %v", result.Error)
	} else {
		log.Printf("删除重复数据: %d 行受影响", result.RowsAffected)
	}

	// 检查索引是否已存在
	var indexCount int64
	db.Raw(`
		SELECT COUNT(*) FROM pg_indexes
			WHERE indexname = 'idx_ops_floors_building_floor_unique'
		`).Scan(&indexCount)

	if indexCount > 0 {
		log.Println("唯一索引已存在，跳过创建")
		return
	}

	// 创建唯一索引
	createIndexSQL := `
	CREATE UNIQUE INDEX idx_ops_floors_building_floor_unique
	ON ops_floors (building_id, floor_no)
	WHERE deleted_at IS NULL`
	result = db.Exec(createIndexSQL)
	if result.Error != nil {
		log.Fatalf("创建唯一索引失败: %v", result.Error)
	}

	// 记录迁移
	db.Exec(`
		INSERT INTO schema_migrations (version, description)
		VALUES (98, 'add_floor_unique_constraint')
		ON CONFLICT (version) DO NOTHING
		`)

	fmt.Println("楼层唯一约束迁移完成!")
}
