package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// loadEnvWhitelist 加载 .env 白名单 key(避免密码含括号/特殊字符时 `source` 失败)
func loadEnvWhitelist(path string, keys []string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, k := range keys {
			prefix := k + "="
			if strings.HasPrefix(line, prefix) {
				val := strings.TrimPrefix(line, prefix)
				val = strings.Trim(val, `"'`)
				os.Setenv(k, val)
				break
			}
		}
	}
}

// 被 reconciliation_normalized / physical_chain / user_lookup 引用的列清单
// (从 migration_175 + migration_176 的 CREATE VIEW SQL 中提取)
var referencedColumns = map[string][]string{
	"ops_asset":               {"id", "devicesn", "machine_ip", "mac1", "mac2", "user_id", "nowuser_name", "deptname", "deleted_at"},
	"sys_ad_user":             {"id", "username", "deleted_at", "is_enabled"},
	"sys_data_reconciliation": {"asset_id", "resolved_at", "resolved_by", "conflict_type", "deleted_at"},
	"sys_device_mac_address":  {"mac_address", "device_id", "interface_name", "collected_at"},
	"sys_device_port_status":  {"id", "device_id", "interface_name"},
	"ops_info_points":         {"workstation_id", "port_id", "deleted_at", "status"},
	"sys_workstation":         {"id", "user_id", "deleted_at"},
	"sys_user":                {"id", "username", "nickname", "dept_id", "deleted_at"},
	"sys_dept":                {"id", "dept_name", "deleted_at"},
}

func main() {
	loadEnvWhitelist(".env", []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE"})
	cfg := config.Load()
	d, err := db.NewDatabase(&cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	// [DEBUG] 用 GORM 自己的 Migrator 读 column 类型,看 GORM 认为它是什么
	import_aduser := models.ADUser{}
	colTypes, err := d.DB.Migrator().ColumnTypes(&import_aduser)
	if err != nil {
		log.Println("ColumnTypes err:", err)
	} else {
		for _, ct := range colTypes {
			if strings.ToLower(ct.Name()) == "username" {
				colType, _ := ct.ColumnType()
				fmt.Printf("[DEBUG GORM Migrator] sys_ad_user.username DatabaseTypeName=%q ColumnType=%q Name=%q\n",
					ct.DatabaseTypeName(), colType, ct.Name())
			}
		}
	}
	_ = import_aduser

	fmt.Printf("%-25s %-20s %-25s %-15s\n", "table", "column", "db_type (length)", "needs_alter?")
	fmt.Println(strings.Repeat("-", 95))

	for table, cols := range referencedColumns {
		// 一次查该表所有相关列
		query := fmt.Sprintf(`
			SELECT column_name, data_type, character_maximum_length, numeric_precision, numeric_scale
			FROM information_schema.columns
			WHERE table_name = '%s'
			  AND column_name IN ('%s')
		`, table, strings.Join(cols, "','"))
		rows, err := d.DB.Raw(query).Rows()
		if err != nil {
			log.Println(err)
			continue
		}
		for rows.Next() {
			var col, dt string
			var charLen, numPrec, numScale *int
			rows.Scan(&col, &dt, &charLen, &numPrec, &numScale)
			t := dt
			if charLen != nil {
				t = fmt.Sprintf("%s(%d)", dt, *charLen)
			} else if numPrec != nil && numScale != nil {
				t = fmt.Sprintf("%s(%d,%d)", dt, *numPrec, *numScale)
			}
			fmt.Printf("%-25s %-20s %-20s %s\n", table, col, t, "查model tag")
		}
		rows.Close()
	}
}
