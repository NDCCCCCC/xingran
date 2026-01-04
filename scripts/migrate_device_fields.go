package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	// 数据库连接配置
	host := "10.62.10.34"
	port := 5432
	user := "postgres"
	password := "Cpic1234"
	dbname := "ruoyi"

	// 构造连接字符串
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// 连接数据库
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatal("数据库连接测试失败:", err)
	}

	fmt.Println("成功连接到数据库")

	// 检查字段是否已存在
	var columnExists int

	// 检查 serial_number 字段
	err = db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'sys_network_device'
		AND column_name = 'serial_number'
	`).Scan(&columnExists)
	if err != nil {
		log.Fatal("查询字段失败:", err)
	}

	sqlStatements := []string{}

	if columnExists == 0 {
		sqlStatements = append(sqlStatements,
			`ALTER TABLE "sys_network_device" ADD COLUMN "serial_number" VARCHAR(100);`,
		)
		fmt.Println("将添加 serial_number 字段")
	} else {
		fmt.Println("serial_number 字段已存在，跳过")
	}

	// 检查 software_version 字段
	err = db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'sys_network_device'
		AND column_name = 'software_version'
	`).Scan(&columnExists)
	if err != nil {
		log.Fatal("查询字段失败:", err)
	}

	if columnExists == 0 {
		sqlStatements = append(sqlStatements,
			`ALTER TABLE "sys_network_device" ADD COLUMN "software_version" VARCHAR(100);`,
		)
		fmt.Println("将添加 software_version 字段")
	} else {
		fmt.Println("software_version 字段已存在，跳过")
	}

	// 检查 uptime 字段
	err = db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'sys_network_device'
		AND column_name = 'uptime'
	`).Scan(&columnExists)
	if err != nil {
		log.Fatal("查询字段失败:", err)
	}

	if columnExists == 0 {
		sqlStatements = append(sqlStatements,
			`ALTER TABLE "sys_network_device" ADD COLUMN "uptime" VARCHAR(100);`,
		)
		fmt.Println("将添加 uptime 字段")
	} else {
		fmt.Println("uptime 字段已存在，跳过")
	}

	// 执行 SQL
	if len(sqlStatements) > 0 {
		fmt.Println("\n开始执行迁移...")
		for i, stmt := range sqlStatements {
			_, err := db.Exec(stmt)
			if err != nil {
				log.Printf("执行第 %d 条语句失败: %v\n", i+1, err)
				log.Printf("SQL: %s\n", stmt)
				continue
			}
			fmt.Printf("成功执行第 %d 条语句\n", i+1)
		}
		fmt.Println("\n迁移执行完成！")
	} else {
		fmt.Println("\n所有字段都已存在，无需迁移")
	}

	// 验证结果
	fmt.Println("\n当前 sys_network_device 表字段:")
	rows, err := db.Query(`
		SELECT column_name, data_type, character_maximum_length
		FROM information_schema.columns
		WHERE table_name = 'sys_network_device'
		AND column_name IN ('serial_number', 'software_version', 'uptime')
		ORDER BY column_name;
	`)
	if err != nil {
		log.Fatal("查询字段失败:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, dataType string
		var maxLength sql.NullInt64
		err = rows.Scan(&name, &dataType, &maxLength)
		if err != nil {
			log.Fatal("扫描行失败:", err)
		}
		fmt.Printf("  - %s: %s", name, dataType)
		if maxLength.Valid {
			fmt.Printf("(%d)", maxLength.Int64)
		}
		fmt.Println()
	}
}
