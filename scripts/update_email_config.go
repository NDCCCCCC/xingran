package main
import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)
func main() {
	db, _ := sql.Open("postgres", "host=10.62.10.34 port=5432 user=postgres password=Cpic1234 dbname=ruoyi sslmode=disable")
	defer db.Close()
	// 添加 use_start_tls 列（如果不存在）
	_, err := db.Exec(`
		ALTER TABLE sys_email_config 
		ADD COLUMN IF NOT EXISTS use_start_tls BOOLEAN DEFAULT true
	`)
	if err != nil {
		fmt.Printf("添加列失败: %v\n", err)
	} else {
		fmt.Println("成功添加 use_start_tls 列")
	}
	// 更新现有配置，设置 use_start_tls = false（纯SMTP）
	result, err := db.Exec("UPDATE sys_email_config SET use_start_tls = false WHERE del_flag = 0")
	if err != nil {
		fmt.Printf("更新配置失败: %v\n", err)
		return
	}
	count, _ := result.RowsAffected()
	fmt.Printf("已更新 %d 条配置，设置为纯SMTP模式\n", count)
}
