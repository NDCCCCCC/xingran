package main

import (
	"fmt"
	"log"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/permission"
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

	// 初始化权限服务
	permissionService := permission.NewService()

	// 重新初始化默认角色和菜单
	fmt.Println("开始初始化系统菜单...")
	if err := permissionService.InitDefaultRolesAndMenus(db); err != nil {
		log.Fatalf("初始化菜单失败: %v", err)
	}

	fmt.Println("系统菜单初始化完成！")
	fmt.Println("新增菜单：")
	fmt.Println("  - 岗位管理")
	fmt.Println("  - 字典管理")
	fmt.Println("  - 参数配置")
	fmt.Println("  - 通知公告")
}
