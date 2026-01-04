package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	connStr := "host=10.62.10.34 port=5432 user=postgres password=Cpic1234 dbname=ruoyi sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("数据库连接测试失败:", err)
	}

	fmt.Println("成功连接到数据库")
	fmt.Println("========================================")

	// 步骤1: 删除重复记录（保留最新的一条）
	fmt.Println("\n步骤1: 清理重复记录...")
	result, err := db.Exec(`
		DELETE FROM sys_device_port_status
		WHERE id IN (
			SELECT id FROM (
				SELECT id,
					ROW_NUMBER() OVER (
						PARTITION BY device_id, interface_name
						ORDER BY created_at DESC, id DESC
					) AS rn
				FROM sys_device_port_status
			) t
			WHERE rn > 1
		)
	`)
	if err != nil {
		log.Printf("清理重复记录失败: %v\n", err)
	} else {
		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("✅ 清理了 %d 条重复记录\n", rowsAffected)
	}

	// 步骤2: 添加唯一约束
	fmt.Println("\n步骤2: 添加唯一约束...")
	_, err = db.Exec(`
		ALTER TABLE sys_device_port_status
		ADD CONSTRAINT uniq_device_interface
		UNIQUE (device_id, interface_name)
	`)
	if err != nil {
		if err.Error() == "pq: constraint \"uniq_device_interface\" for relation \"sys_device_port_status\" already exists" {
			fmt.Println("⏭️ 唯一约束已存在，跳过")
		} else {
			log.Printf("❌ 添加唯一约束失败: %v\n", err)
			return
		}
	} else {
		fmt.Println("✅ 成功添加唯一约束 uniq_device_interface")
	}

	// 步骤3: 验证
	fmt.Println("\n验证约束...")
	var constraintName string
	err = db.QueryRow(`
		SELECT constraint_name
		FROM information_schema.table_constraints
		WHERE table_name = 'sys_device_port_status'
		AND constraint_type = 'UNIQUE'
		AND constraint_name = 'uniq_device_interface'
	`).Scan(&constraintName)

	if err == sql.ErrNoRows {
		fmt.Println("❌ 约束验证失败")
	} else if err != nil {
		log.Printf("❌ 验证查询失败: %v\n", err)
	} else {
		fmt.Printf("✅ 约束验证成功: %s\n", constraintName)
	}

	fmt.Println("\n========================================")
	fmt.Println("迁移完成！现在可以重新运行端口采集测试了。")
}
