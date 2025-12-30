package main
import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)
func main() {
	db, _ := sql.Open("postgres", "host=10.62.10.34 port=5432 user=postgres password=Cpic1234 dbname=ruoyi sslmode=disable")
	defer db.Close()
	// 查询系统管理菜单及其子菜单
	rows, _ := db.Query(`
		SELECT m.menu_name, m.path, m.component, m.icon, m.menu_type, m.visible, m.status, p.menu_name as parent_name
		FROM sys_menu m
		LEFT JOIN sys_menu p ON m.parent_id = p.id
		WHERE p.menu_name = '系统管理' OR m.menu_name = '系统管理'
		ORDER BY m.order_num
	`)
	defer rows.Close()
	fmt.Println("菜单名称            路径                    组件                    图标            类型 可见 状态 父菜单")
	fmt.Println("---------------------------------------------------------------------------------------------------------------")
	for rows.Next() {
		var name, path, component, icon, menuType, parentName string
		var visible, status int
		rows.Scan(&name, &path, &component, &icon, &menuType, &visible, &status, &parentName)
		fmt.Printf("%-18s %-23s %-23s %-15s %s  %d   %d    %s\n", name, path, component, icon, menuType, visible, status, parentName)
	}
}
