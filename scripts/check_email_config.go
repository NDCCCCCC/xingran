package main
import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)
func main() {
	db, _ := sql.Open("postgres", "host=10.62.10.34 port=5432 user=postgres password=Cpic1234 dbname=ruoyi sslmode=disable")
	defer db.Close()
	// 查询现有的邮件配置
	rows, _ := db.Query(`
		SELECT id, config_name, host, port, username, is_default, status, del_flag 
		FROM sys_email_config
		ORDER BY created_at DESC
	`)
	defer rows.Close()
	fmt.Println("ID                                   ConfigName        Host            Port IsDefault Status DelFlag")
	fmt.Println("------------------------------------------------------------------------------------------")
	count := 0
	for rows.Next() {
		var id, configName, host string
		var port int
		var isDefault, status, delFlag int
		rows.Scan(&id, &configName, &host, &port, &isDefault, &status, &delFlag)
		defaultMark := ""
		if isDefault == 1 {
			defaultMark = "*"
		}
		delMark := ""
		if delFlag == 1 {
			delMark = " (已删除)"
		}
		fmt.Printf("%s %.20s %-.15s %d %-9s %d     %d     %s\n", id[:8]+"...", configName, host, port, defaultMark, status, delFlag, delMark)
		count++
	}
	fmt.Printf("\n共 %d 条记录\n", count)
	
	// 统计未删除的配置
	var activeCount int
	db.QueryRow("SELECT COUNT(*) FROM sys_email_config WHERE del_flag = 0").Scan(&activeCount)
	fmt.Printf("有效配置数量: %d\n", activeCount)
}
