package main
import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)
func main() {
	db, _ := sql.Open("postgres", "host=10.62.10.34 port=5432 user=postgres password=Cpic1234 dbname=ruoyi sslmode=disable")
	defer db.Close()
	// 更新图标和路径
	_, err := db.Exec("UPDATE sys_menu SET icon = 'SettingOutlined', path = '/system/settings-page' WHERE menu_name = '系统设置'")
	if err != nil {
		fmt.Println("更新失败:", err)
		return
	}
	fmt.Println("修复成功！")
	// 验证
	var icon, path string
	db.QueryRow("SELECT icon, path FROM sys_menu WHERE menu_name = '系统设置'").Scan(&icon, &path)
	fmt.Printf("图标: %s, 路径: %s\n", icon, path)
}
