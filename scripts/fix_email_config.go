package main
import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)
func main() {
	db, _ := sql.Open("postgres", "host=10.62.10.34 port=5432 user=postgres password=Cpic1234 dbname=ruoyi sslmode=disable")
	defer db.Close()
	// 更新 is_default 字段为 true (boolean)
	_, err := db.Exec("UPDATE sys_email_config SET is_default = true WHERE del_flag = 0")
	if err != nil {
		fmt.Printf("更新失败: %v\n", err)
		return
	}
	fmt.Println("已设置邮件配置为默认配置")
	// 验证
	var isDefault bool
	db.QueryRow("SELECT is_default FROM sys_email_config WHERE del_flag = 0").Scan(&isDefault)
	if isDefault {
		fmt.Println("验证成功: is_default = true")
	} else {
		fmt.Println("验证失败: is_default = false")
	}
}
