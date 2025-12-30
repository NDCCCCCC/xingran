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

	// 检查表是否存在
	rows, err := db.Query(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name IN ('sys_email_config', 'sys_api_notification_config', 'sys_notification_channel')
		ORDER BY table_name
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("检查表是否存在:")
	foundCount := 0
	for rows.Next() {
		var name string
		rows.Scan(&name)
		fmt.Printf("  ✅ %s\n", name)
		foundCount++
	}

	if foundCount == 3 {
		fmt.Println("\n所有表都已创建成功！")
	} else {
		fmt.Printf("\n只找到 %d/3 个表\n", foundCount)
	}
}
