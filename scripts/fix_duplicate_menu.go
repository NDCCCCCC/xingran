package main

import (
	"fmt"
	"log"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化核心组件
	c := core.New(cfg)
	if err := c.Init(); err != nil {
		log.Fatalf("初始化核心组件失败: %v", err)
	}
	defer c.Close()

	db := c.GetDB()

	// 首先获取"参数设置"菜单的ID
	var menuID string
	db.Raw("SELECT id FROM sys_menu WHERE menu_name = '参数设置' AND path = '/system/config' LIMIT 1").Scan(&menuID)

	if menuID == "" {
		fmt.Println("没有找到重复的菜单")
		return
	}

	fmt.Printf("找到重复菜单 ID: %s\n", menuID)

	// 1. 先删除子菜单（按钮权限）
	db.Exec("DELETE FROM sys_menu WHERE parent_id = ?", menuID)
	fmt.Println("已删除相关按钮权限")

	// 2. 删除角色菜单关联
	db.Exec("DELETE FROM sys_role_menu WHERE menu_id = ?", menuID)
	fmt.Println("已删除角色菜单关联")

	// 3. 删除菜单本身
	result := db.Exec("DELETE FROM sys_menu WHERE id = ?", menuID)
	if result.Error != nil {
		log.Fatalf("删除重复菜单失败: %v", result.Error)
	}

	rowsAffected := result.RowsAffected
	fmt.Printf("成功删除 %d 个重复菜单\n", rowsAffected)

	// 4. 更新"参数配置"的order_num为7
	db.Exec("UPDATE sys_menu SET order_num = 7 WHERE menu_name = '参数配置' AND path = '/system/config'")

	// 5. 重新设置所有系统管理子菜单的order_num
	db.Exec(`
		UPDATE sys_menu SET order_num = t.new_order
		FROM (VALUES
			('用户管理', 1),
			('角色管理', 2),
			('菜单管理', 3),
			('部门管理', 4),
			('岗位管理', 5),
			('字典管理', 6),
			('参数配置', 7),
			('通知公告', 8),
			('工位管理', 9)
		) AS t(name, new_order)
		WHERE sys_menu.menu_name = t.name
		AND sys_menu.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '系统管理')
	`)

	fmt.Println("菜单排序已更新")
}
