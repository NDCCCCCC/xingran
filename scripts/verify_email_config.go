package main
import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)
func main() {
	db, _ := sql.Open("postgres", "host=10.62.10.34 port=5432 user=postgres password=Cpic1234 dbname=ruoyi sslmode=disable")
	defer db.Close()
	rows, _ := db.Query(`
		SELECT id, config_name, host, port, username, from_name, use_ssl, is_default, status 
		FROM sys_email_config
		WHERE del_flag = 0
	`)
	defer rows.Close()
	fmt.Println("邮件配置列表:")
	fmt.Println("ID                                   ConfigName    Host            Port Username                           FromName      SSL  默认 状态")
	fmt.Println("--------------------------------------------------------------------------------------------------------------")
	for rows.Next() {
		var id, configName, host, username, fromName string
		var port int
		var useSSL, isDefault, status int
		rows.Scan(&id, &configName, &host, &port, &username, &fromName, &useSSL, &isDefault, &status)
		ssl := "否"
		if useSSL == 1 {
			ssl = "是"
		}
		def := "否"
		if isDefault == 1 {
			def = "是"
		}
		st := "正常"
		if status == 1 {
			st = "停用"
		}
		fmt.Printf("%s  %-14s %-15s %d  %-33s %-12s %s  %s   %s\n", 
			id[:8]+"...", configName, host, port, username, fromName, ssl, def, st)
	}
}
