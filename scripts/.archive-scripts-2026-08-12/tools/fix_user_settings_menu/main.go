package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type UpdateMenuRequest struct {
	MenuName  string  `json:"menuName"`
	ParentID  *string `json:"parentId"`
	OrderNum  int     `json:"orderNum"`
	Path      *string `json:"path"`
	Component *string `json:"component"`
	MenuType  string  `json:"menuType"`
	Visible   int     `json:"visible"`
	Status    int     `json:"status"`
	Perms     *string `json:"perms"`
	Icon      *string `json:"icon"`
	Remark    *string `json:"remark"`
}

type MenuListResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    []Menu `json:"data"`
}

type Menu struct {
	ID        string  `json:"id"`
	MenuName  string  `json:"menuName"`
	ParentID  *string `json:"parentId"`
	Path      string  `json:"path"`
	Component *string `json:"component"`
	MenuType  string  `json:"menuType"`
	Visible   int     `json:"visible"`
	Status    int     `json:"status"`
}

const baseURL = "http://localhost:8080/api/v1"

func main() {
	token := os.Getenv("XINGRAN_TOKEN")
	if token == "" {
		fmt.Println("错误: 请设置环境变量 XINGRAN_TOKEN")
		fmt.Println("使用方式: XINGRAN_TOKEN=your_token_here go run scripts/fix_user_settings_menu.go")
		os.Exit(1)
	}

	client := &http.Client{}

	fmt.Println("步骤1: 查询菜单列表...")
	menus, err := getMenuList(client, token)
	if err != nil {
		fmt.Printf("查询菜单失败: %v\n", err)
		os.Exit(1)
	}

	var userCenterID *string
	var systemSettingsUnderUserCenter *Menu

	for _, menu := range menus {
		if menu.MenuName == "用户中心" && menu.ParentID == nil {
			id := menu.ID
			userCenterID = &id
		}
		if menu.MenuName == "系统设置" && userCenterID != nil && menu.ParentID != nil && *menu.ParentID == *userCenterID {
			systemSettingsUnderUserCenter = &menu
		}
	}

	if userCenterID == nil {
		fmt.Println("错误: 找不到用户中心父菜单")
		os.Exit(1)
	}

	if systemSettingsUnderUserCenter == nil {
		fmt.Println("警告: 用户中心下没有找到'系统设置'菜单")
		fmt.Println("可能已经被修复，检查是否存在'用户设置'菜单...")

		for _, menu := range menus {
			if menu.MenuName == "用户设置" && menu.ParentID != nil && *menu.ParentID == *userCenterID {
				fmt.Println("✓ 用户设置菜单已存在，无需修复")
				return
			}
		}
		fmt.Println("错误: 用户中心下既没有'系统设置'也没有'用户设置'菜单")
		os.Exit(1)
	}

	fmt.Printf("找到用户中心下的'系统设置'菜单: ID=%s\n", systemSettingsUnderUserCenter.ID)

	fmt.Println("\n步骤2: 更新菜单名称为'用户设置'...")
	component := "settings/index"
	remark := "用户设置页面（个人偏好设置：主题、布局等）"

	updateReq := UpdateMenuRequest{
		MenuName:  "用户设置",
		ParentID:  systemSettingsUnderUserCenter.ParentID,
		OrderNum:  2,
		Path:      &systemSettingsUnderUserCenter.Path,
		Component: &component,
		MenuType:  systemSettingsUnderUserCenter.MenuType,
		Visible:   systemSettingsUnderUserCenter.Visible,
		Status:    systemSettingsUnderUserCenter.Status,
		Remark:    &remark,
	}

	err = updateMenu(client, token, systemSettingsUnderUserCenter.ID, updateReq)
	if err != nil {
		fmt.Printf("更新菜单失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ 成功将用户中心下的'系统设置'改名为'用户设置'")
	fmt.Println("✓ 组件路径已修复为: settings/index")
	fmt.Println("\n提示: 请刷新前端页面查看效果")
}

func getMenuList(client *http.Client, token string) ([]Menu, error) {
	req, err := http.NewRequest("POST", baseURL+"/system/menus/list", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result MenuListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API返回错误: %s", result.Message)
	}

	return result.Data, nil
}

func updateMenu(client *http.Client, token, menuID string, req UpdateMenuRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/system/menus/%s/update", baseURL, menuID)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if code, ok := result["code"].(float64); !ok || code != 0 {
		return fmt.Errorf("API返回错误: %v", result)
	}

	return nil
}
