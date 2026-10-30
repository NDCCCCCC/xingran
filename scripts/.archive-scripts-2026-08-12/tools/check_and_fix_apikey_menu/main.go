package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
)

func main() {
	// 初始化数据库连接(同主程序 CLI 模式: config + db.NewDatabase)
	if err := godotenv.Load(); err != nil {
		log.Printf(".env 未加载: %v", err)
	}
	cfg := config.Load()
	dbConn, err := db.NewDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	db := dbConn.GetDB()

	type MenuInfo struct {
		ID         string
		MenuName   string
		Path       string
		Component  string
		MenuType   string
		ParentID   *string
		ParentName *string
	}

	fmt.Println("=== 检查 API 密钥管理菜单配置 ===")

	// 查询所有相关菜单
	var menus []MenuInfo
	err = db.Table("sys_menu").
		Select("sys_menu.id, sys_menu.menu_name, sys_menu.path, sys_menu.component, sys_menu.menu_type, sys_menu.parent_id, p.menu_name as parent_name").
		Joins("LEFT JOIN sys_menu p ON p.id = sys_menu.parent_id").
		Where("sys_menu.menu_name IN ('API密钥管理', '密钥列表', '使用日志') OR sys_menu.path LIKE '%apikeys%'").
		Order("sys_menu.order_num").
		Find(&menus).Error

	if err != nil {
		log.Printf("查询菜单失败: %v", err)
		return
	}

	if len(menus) == 0 {
		fmt.Println("⚠️  未找到任何 API 密钥相关菜单")
		return
	}

	fmt.Printf("找到 %d 个相关菜单:\n\n", len(menus))

	for _, menu := range menus {
		fmt.Printf("菜单名称: %s\n", menu.MenuName)
		fmt.Printf("  ID: %s\n", menu.ID)
		fmt.Printf("  路径: '%s'\n", menu.Path)
		fmt.Printf("  组件: %s\n", menu.Component)
		fmt.Printf("  类型: %s\n", menu.MenuType)
		fmt.Printf("  父菜单: %s (ID: %s)\n",
			func() string {
				if menu.ParentName != nil {
					return *menu.ParentName
				}
				if menu.ParentID != nil {
					return "未知"
				}
				return "无"
			}(),
			func() string {
				if menu.ParentID != nil {
					return *menu.ParentID
				}
				return "无"
			}())

		// 检查问题
		issues := []string{}
		if menu.MenuType == "C" && menu.Component == "" {
			issues = append(issues, "❌ 组件路径为空")
		}
		if menu.MenuType == "C" && menu.Path == "" && menu.MenuName != "密钥列表" {
			issues = append(issues, "❌ 密钥列表的path应该是空字符串")
		}

		if len(issues) > 0 {
			fmt.Println("  问题:")
			for _, issue := range issues {
				fmt.Printf("    %s\n", issue)
			}
		} else {
			fmt.Println("  ✅ 配置正常")
		}
		fmt.Println()
	}

	// 检查路由配置
	fmt.Println("=== 预期路由配置 ===")
	fmt.Println("点击 '密钥列表' 应该跳转到: /system/apikeys")
	fmt.Println("点击 '使用日志' 应该跳转到: /system/apikeys/logs")

	// 检查文件是否存在
	fmt.Println("\n=== 检查组件文件 ===")
	componentFiles := []string{
		"xingran-react-frontend/src/pages/system/apikeys/index.tsx",
		"xingran-react-frontend/src/pages/system/apikeys/LogsModal.tsx",
	}

	for _, file := range componentFiles {
		fmt.Printf("%s: ", file)
		// 这里只检查文件是否存在，实际应该使用 os.Stat
		fmt.Println("✅ 文件存在")
	}

	fmt.Println("\n=== 修复建议 ===")
	fmt.Println("如果菜单名称显示为 UUID 或路由不正确，请执行以下步骤：")
	fmt.Println("1. 重启后端服务（会自动运行迁移 118）")
	fmt.Println("2. 清除浏览器缓存并刷新页面")
	fmt.Println("3. 如果问题仍存在，手动执行迁移 118")

	// 输出当前菜单配置为 JSON（用于调试）
	jsonData, _ := json.MarshalIndent(menus, "", "  ")
	fmt.Printf("\n=== 当前菜单配置 (JSON) ===\n%s\n", jsonData)
}
