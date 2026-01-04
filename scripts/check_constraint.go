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
		log.Fatal(err)
	}
	defer db.Close()

	// 检查约束是否存在
	rows, err := db.Query(`
		SELECT constraint_name
		FROM information_schema.table_constraints
		WHERE table_name = 'sys_device_port_status'
		AND constraint_type = 'UNIQUE'
	`)
	if err != nil {
		log.Printf("查询失败: %v\n", err)
		return
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		var constraintName string
		if err := rows.Scan(&constraintName); err != nil {
			log.Printf("扫描失败: %v\n", err)
			return
		}
		fmt.Printf("✅ 找到唯一约束: %s\n", constraintName)
		found = true
	}

	if !found {
		fmt.Println("❌ 未找到唯一约束，需要手动添加")
		fmt.Println("\n请执行以下 SQL:")
		fmt.Println("ALTER TABLE sys_device_port_status")
		fmt.Println("ADD CONSTRAINT uniq_device_interface")
		fmt.Println("UNIQUE (device_id, interface_name);")
	}
}
