package main
import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)
func main() {
	db, _ := sql.Open("postgres", "host=10.62.10.34 port=5432 user=postgres password=Cpic1234 dbname=ruoyi sslmode=disable")
	defer db.Close()
	rows, _ := db.Query("SELECT menu_name, icon, path, component, menu_type FROM sys_menu WHERE menu_name IN ('系统管理', '用户管理', '系统设置', '参数设置', '通知公告')")
	defer rows.Close()
	fmt.Println("菜单名称              图标         路径                    组件                  类型")
	fmt.Println("----------------------------------------------------------------------------------------------")
	for rows.Next() {
		var name, icon, path, component, menuType string
		rows.Scan(&name, &icon, &path, &component, &menuType)
		fmt.Printf("%-20s %-12s %-23s %-22s %s\n", name, icon, path, component, menuType)
	}
}
