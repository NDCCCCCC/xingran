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

	// 检查 sys_notice_ignore 表是否存在
	rows, err := db.Query(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name = 'sys_notice_ignore'
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("检查 sys_notice_ignore 表是否存在:")
	found := false
	for rows.Next() {
		var name string
		rows.Scan(&name)
		fmt.Printf("  ✅ %s\n", name)
		found = true
	}

	if found {
		fmt.Println("\nsys_notice_ignore 表创建成功！")
	} else {
		fmt.Println("\n❌ sys_notice_ignore 表不存在")
	}
}
