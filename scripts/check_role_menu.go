package main
import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)
func main() {
	db, _ := sql.Open("postgres", "host=10.62.10.34 port=5432 user=postgres password=Cpic1234 dbname=ruoyi sslmode=disable")
	defer db.Close()
	// 查询admin角色的菜单
	rows, _ := db.Query(`
		SELECT r.role_name, m.menu_name, m.path, m.component
		FROM sys_role_menu rm
		JOIN sys_role r ON rm.role_id = r.id
		JOIN sys_menu m ON rm.menu_id = m.id
		WHERE r.role_key = 'admin' AND m.parent_id IN (
			SELECT id FROM sys_menu WHERE menu_name = '系统管理'
		)
		ORDER BY m.order_num
	`)
	defer rows.Close()
	fmt.Println("角色         菜单名称            路径                    组件")
	fmt.Println("-----------------------------------------------------------------------------")
	count := 0
	for rows.Next() {
		var roleName, menuName, path, component string
		rows.Scan(&roleName, &menuName, &path, &component)
		fmt.Printf("%-12s %-18s %-23s %s\n", roleName, menuName, path, component)
		count++
	}
	fmt.Printf("\n共 %d 个子菜单\n", count)
}
