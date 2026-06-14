package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "host=10.62.10.34 port=5432 user=postgres password=Cpic1234 dbname=ruoyi sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()

	fmt.Println("=== Checking ops_asset constraints ===")

	// 查询 ops_asset 表的所有约束
	query := `
		SELECT
			conname,
			contype
		FROM pg_constraint
		WHERE conrelid::regclass = 'ops_asset'::regclass
		ORDER BY conname
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal("Query failed:", err)
	}
	defer rows.Close()

	fmt.Println("\nExisting constraints on ops_asset table:")
	for rows.Next() {
		var conname, contype string
		if err := rows.Scan(&conname, &contype); err != nil {
			continue
		}
		fmt.Printf("  - %s (%s)\n", conname, contype)
	}

	// 检查特定约束
	specificCheck := `
		SELECT EXISTS(
			SELECT 1 FROM pg_constraint
			WHERE conname = 'uni_ops_assetsn'
			AND conrelid::regclass = 'ops_asset'::regclass
		);
	`

	var exists bool
	err = db.QueryRow(specificCheck).Scan(&exists)
	if err != nil {
		fmt.Printf("\nConstraint check error: %v\n", err)
	} else {
		fmt.Printf("\nConstraint 'uni_ops_assetsn' exists: %v\n", exists)
	}
}
