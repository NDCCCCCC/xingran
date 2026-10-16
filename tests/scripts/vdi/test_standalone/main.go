package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// VDI 连接信息从环境变量读取，避免明文凭证入库：
//
//	VDI_BASE_URL / VDI_USER / VDI_PASS
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var (
	VdiBaseURL = getenv("VDI_BASE_URL", "")
	VdiUser    = getenv("VDI_USER", "")
	VdiPass    = os.Getenv("VDI_PASS")
)

type VDIAPIClient struct {
	client  *http.Client
	token   string
	baseURL string
}

func NewVDIAPIClient() *VDIAPIClient {
	return &VDIAPIClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		baseURL: VdiBaseURL,
	}
}

func (c *VDIAPIClient) Authenticate() error {
	fmt.Println("【认证】")

	authPayload := map[string]interface{}{
		"auth": map[string]string{
			"name":     VdiUser,
			"password": VdiPass,
		},
	}

	jsonData, _ := json.Marshal(authPayload)
	req, _ := http.NewRequest("POST", c.baseURL+"/v1/auth/tokens", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("认证请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if token, ok := result["token"].(map[string]interface{}); ok {
		if authToken, ok := token["auth_token"].(string); ok {
			c.token = authToken
			fmt.Printf("✓ 认证成功\n\n")
			return nil
		}
	}

	return fmt.Errorf("认证失败")
}

func (c *VDIAPIClient) CallAPI(method, path string, body interface{}) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonData)
		fmt.Printf("[请求体] %s\n", string(jsonData))
	}

	req, _ := http.NewRequest(method, c.baseURL+path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Auth-Token", c.token)

	fmt.Printf("[API] %s %s\n", method, path)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("[响应] 状态=%d\n", resp.StatusCode)

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)
	return result, nil
}

func (c *VDIAPIClient) OperateVM(vmID, action string) error {
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("操作虚拟机 [%s]: %s\n", vmID, action)
	fmt.Println("═══════════════════════════════════════════════════════")

	actionMap := map[string]string{
		"start":   "startup",
		"stop":    "shutdown",
		"restart": "reboot",
		"suspend": "suspend",
		"resume":  "resume",
	}

	actionName, ok := actionMap[action]
	if !ok {
		return fmt.Errorf("不支持的操作: %s", action)
	}

	reqBody := map[string]interface{}{
		"servers": map[string]string{"ids": vmID},
		"action":  map[string]string{"name": actionName},
	}

	result, err := c.CallAPI("POST", "/v1/servers/action", reqBody)
	if err != nil {
		return err
	}

	errorCode, _ := result["error_code"].(float64)
	errorMsg, _ := result["error_message"].(string)

	if errorCode == 0 {
		fmt.Printf("✓ 操作成功: %s\n\n", errorMsg)
		return nil
	} else {
		fmt.Printf("✗ 操作失败: [%d] %s\n\n", int(errorCode), errorMsg)
		return fmt.Errorf("操作失败: %s", errorMsg)
	}
}

func (c *VDIAPIClient) ListVMs() error {
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("获取虚拟机列表")
	fmt.Println("═══════════════════════════════════════════════════════")

	// 尝试获取资源组
	fmt.Println("【步骤1】获取资源组...")
	result, err := c.CallAPI("GET", "/v1/resources_group", nil)
	if err != nil {
		return err
	}

	errorCode, _ := result["error_code"].(float64)
	if errorCode != 0 {
		errorMsg, _ := result["error_message"].(string)
		fmt.Printf("✗ 获取资源组失败: %s\n\n", errorMsg)
		return fmt.Errorf("获取资源组失败")
	}

	data, _ := result["data"].([]interface{})
	fmt.Printf("✓ 找到 %d 个资源组\n\n", len(data))

	if len(data) == 0 {
		return fmt.Errorf("没有资源组")
	}

	allVMs := make([]map[string]interface{}, 0)

	// 遍历所有资源组
	for groupIdx, groupItem := range data {
		group := groupItem.(map[string]interface{})
		groupID := fmt.Sprintf("%v", group["id"])
		groupName := fmt.Sprintf("%v", group["name"])
		fmt.Printf("【资源组 %d】%s (%s)\n", groupIdx+1, groupName, groupID)

		// 获取该资源组的资源列表
		fmt.Printf("  → 获取资源列表...\n")
		result, err = c.CallAPI("GET", "/v1/resources/list/"+groupID, nil)
		if err != nil {
			fmt.Printf("  ✗ 获取资源列表失败: %v\n", err)
			continue
		}

		errorCode, _ = result["error_code"].(float64)
		if errorCode != 0 {
			errorMsg, _ := result["error_message"].(string)
			fmt.Printf("  ✗ 获取资源列表失败: %s\n", errorMsg)
			continue
		}

		resourcesData, _ := result["data"].(map[string]interface{})
		resources, _ := resourcesData["resources"].([]interface{})
		fmt.Printf("  ✓ 找到 %d 个资源\n", len(resources))

		if len(resources) == 0 {
			fmt.Println("  (该资源组无资源)")
			fmt.Println()
			continue
		}

		// 遍历该资源组的所有资源
		for resIdx, resItem := range resources {
			res := resItem.(map[string]interface{})
			resID := fmt.Sprintf("%v", res["id"])
			resName := fmt.Sprintf("%v", res["name"])
			fmt.Printf("    【资源 %d】%s (%s)\n", resIdx+1, resName, resID)

			// 获取该资源的虚拟机列表
			fmt.Printf("      → 获取虚拟机列表...\n")
			path := fmt.Sprintf("/v1/resource/servers?rcid=%s&page=1&page_size=100", resID)
			result, err = c.CallAPI("GET", path, nil)
			if err != nil {
				fmt.Printf("      ✗ 获取虚拟机列表失败: %v\n", err)
				continue
			}

			errorCode, _ = result["error_code"].(float64)
			if errorCode != 0 {
				errorMsg, _ := result["error_message"].(string)
				fmt.Printf("      ✗ 获取虚拟机列表失败: %s\n", errorMsg)
				continue
			}

			vmsData, _ := result["data"].(map[string]interface{})
			vms, _ := vmsData["data"].([]interface{})
			totalCount := fmt.Sprintf("%v", vmsData["totalCount"])

			fmt.Printf("      ✓ 找到 %s 个虚拟机\n", totalCount)

			if len(vms) == 0 {
				fmt.Println("      (该资源无虚拟机)")
			}

			// 收集所有虚拟机
			for _, vm := range vms {
				if vmMap, ok := vm.(map[string]interface{}); ok {
					vmMap["_resource_group"] = groupName
					vmMap["_resource_name"] = resName
					allVMs = append(allVMs, vmMap)
				}
			}
		}
		fmt.Println()
	}

	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("虚拟机汇总 (共 %d 台)\n", len(allVMs))
	fmt.Println("═══════════════════════════════════════════════════════")

	if len(allVMs) == 0 {
		fmt.Println("没有虚拟机")
		return nil
	}

	for i, vmMap := range allVMs {
		vmID := fmt.Sprintf("%v", vmMap["_id"])
		vmName := fmt.Sprintf("%v", vmMap["vm_name"])
		status := fmt.Sprintf("%v", vmMap["status"])
		ip := fmt.Sprintf("%v", vmMap["ip"])
		groupName := fmt.Sprintf("%v", vmMap["_resource_group"])
		resName := fmt.Sprintf("%v", vmMap["_resource_name"])

		// 系统配置信息
		osType := fmt.Sprintf("%v", vmMap["osType"])
		mac := fmt.Sprintf("%v", vmMap["mac"])
		cpuNum := fmt.Sprintf("%v", vmMap["cpu_number"])
		cpuCore := fmt.Sprintf("%v", vmMap["cpu_core"])
		cpuPer := fmt.Sprintf("%v", vmMap["cpu_per"])
		memAll := fmt.Sprintf("%v", vmMap["mem_all"])
		memNow := fmt.Sprintf("%v", vmMap["mem_now"])
		memPer := fmt.Sprintf("%v", vmMap["mem_per"])
		discAll := fmt.Sprintf("%v", vmMap["disc_all"])
		discNow := fmt.Sprintf("%v", vmMap["disc_now"])
		discPer := fmt.Sprintf("%v", vmMap["disc_per"])

		// 显卡信息
		graphics := fmt.Sprintf("%v", vmMap["graphics"])
		graphicsMemAll := fmt.Sprintf("%v", vmMap["graphics_mem_all"])
		graphicsMemNow := fmt.Sprintf("%v", vmMap["graphics_mem_now"])
		graphicsCardType := fmt.Sprintf("%v", vmMap["graphics_card_type"])

		// 网络配置
		assignIP := fmt.Sprintf("%v", vmMap["assign_ip"])
		subnetmask := fmt.Sprintf("%v", vmMap["subnetmask"])
		defaultgateway := fmt.Sprintf("%v", vmMap["defaultgateway"])
		nameserver := fmt.Sprintf("%v", vmMap["nameserver"])

		// 用户关联信息
		applyUser := fmt.Sprintf("%v", vmMap["apply_user"])
		applyUserStatus := fmt.Sprintf("%v", vmMap["apply_user_status"])
		applyAppstack := fmt.Sprintf("%v", vmMap["apply_appstack"])

		// 策略组信息
		groupPolicyName := fmt.Sprintf("%v", vmMap["group_policy_name"])

		// 其他信息
		agentVersion := fmt.Sprintf("%v", vmMap["agent_version"])
		lastLogin := fmt.Sprintf("%v", vmMap["last_login"])
		vtpName := fmt.Sprintf("%v", vmMap["vtp_name"])

		fmt.Printf("\n【VM %d】%s\n", i+1, vmName)
		fmt.Printf("  ═══ 基本信息 ═══\n")
		fmt.Printf("  ID:           %s\n", vmID)
		fmt.Printf("  状态:         %s\n", status)
		fmt.Printf("  IP:           %s (分配: %s)\n", ip, assignIP)
		fmt.Printf("  MAC:          %s\n", mac)
		fmt.Printf("  所属:         %s / %s\n", groupName, resName)
		fmt.Printf("  VDI平台:      %s\n", vtpName)

		fmt.Printf("  ═══ 系统配置 ═══\n")
		fmt.Printf("  操作系统:     %s\n", osType)
		fmt.Printf("  CPU:          %s颗 × %s核 (使用率: %s%%)\n", cpuNum, cpuCore, cpuPer)
		fmt.Printf("  内存:         %sMB / %sMB (使用率: %s%%)\n", memNow, memAll, memPer)
		fmt.Printf("  磁盘:         %sMB / %sMB (使用率: %s%%)\n", discNow, discAll, discPer)

		fmt.Printf("  ═══ 显卡配置 ═══\n")
		if graphics == "2" || graphics == "4" || graphics == "5" {
			fmt.Printf("  显卡类型:     %s\n", graphicsCardType)
			fmt.Printf("  显存:         %sMB / %sMB\n", graphicsMemNow, graphicsMemAll)
		} else {
			fmt.Printf("  显卡:         2D虚拟机 (未配置显卡)\n")
		}

		fmt.Printf("  ═══ 网络配置 ═══\n")
		if assignIP != "" && assignIP != "<nil>" {
			fmt.Printf("  IP模式:       静态\n")
			fmt.Printf("  子网掩码:     %s\n", subnetmask)
			fmt.Printf("  网关:         %s\n", defaultgateway)
			fmt.Printf("  DNS:          %s\n", nameserver)
		} else {
			fmt.Printf("  IP模式:       DHCP\n")
		}

		fmt.Printf("  ═══ 用户关联 ═══\n")
		if applyUser != "" && applyUser != "<nil>" {
			loginStatus := "未登录"
			if applyUserStatus == "1" {
				loginStatus = "已登录"
			}
			fmt.Printf("  关联用户:     %s (%s)\n", applyUser, loginStatus)
			if applyAppstack != "" && applyAppstack != "<nil>" {
				fmt.Printf("  软件库:       %s\n", applyAppstack)
			}
		} else {
			fmt.Printf("  关联用户:     无\n")
		}

		if groupPolicyName != "" && groupPolicyName != "<nil>" {
			fmt.Printf("  ═══ 策略组 ═══\n")
			fmt.Printf("  策略组:       %s\n", groupPolicyName)
		}

		fmt.Printf("  ═══ 其他信息 ═══\n")
		if agentVersion != "" && agentVersion != "<nil>" {
			fmt.Printf("  Agent版本:    %s\n", agentVersion)
		}
		if lastLogin != "" && lastLogin != "0" && lastLogin != "<nil>" {
			fmt.Printf("  最近登录:     %s\n", lastLogin)
		}

		fmt.Println("  ═══ 操作命令 ═══")
		fmt.Printf("    go run scripts/vdi_test_standalone.go operate %s start\n", vmID)
	}

	fmt.Println()

	return nil
}

func (c *VDIAPIClient) GetRunPositions(vtpID int) ([]map[string]interface{}, error) {
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("获取运行位置 (vtp_id=%d)\n", vtpID)
	fmt.Println("═══════════════════════════════════════════════════════")

	path := fmt.Sprintf("/v1/run_position?vtp_id=%d", vtpID)
	result, err := c.CallAPI("GET", path, nil)
	if err != nil {
		return nil, err
	}

	errorCode, _ := result["error_code"].(float64)
	if errorCode != 0 {
		errorMsg, _ := result["error_message"].(string)
		fmt.Printf("✗ 获取运行位置失败: %s\n\n", errorMsg)
		return nil, fmt.Errorf("获取运行位置失败: %s", errorMsg)
	}

	// 从 data.run 中获取数据
	var runData []interface{}
	if data, ok := result["data"].(map[string]interface{}); ok {
		if run, ok := data["run"].([]interface{}); ok {
			runData = run
		}
	}

	if len(runData) == 0 {
		fmt.Println("✗ 响应格式错误：未找到 run 数组")
		return nil, fmt.Errorf("响应格式错误")
	}

	fmt.Printf("✓ 找到 %d 个运行位置\n\n", len(runData))

	positions := make([]map[string]interface{}, 0)
	for i, item := range runData {
		if pos, ok := item.(map[string]interface{}); ok {
			positions = append(positions, pos)
			id := fmt.Sprintf("%v", pos["id"])
			name := fmt.Sprintf("%v", pos["name"])
			fatherID := fmt.Sprintf("%v", pos["father_id"])
			fmt.Printf("  [%d] %s (id=%s, father_id=%s)\n", i+1, name, id, fatherID)
		}
	}
	fmt.Println()

	return positions, nil
}

func (c *VDIAPIClient) GetStorages(vtpID int) ([]map[string]interface{}, error) {
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("获取存储位置 (vtp_id=%d)\n", vtpID)
	fmt.Println("═══════════════════════════════════════════════════════")

	path := fmt.Sprintf("/v1/storages?vtp_id=%d", vtpID)
	result, err := c.CallAPI("GET", path, nil)
	if err != nil {
		return nil, err
	}

	errorCode, _ := result["error_code"].(float64)
	if errorCode != 0 {
		errorMsg, _ := result["error_message"].(string)
		fmt.Printf("✗ 获取存储位置失败: %s\n\n", errorMsg)
		return nil, fmt.Errorf("获取存储位置失败: %s", errorMsg)
	}

	// 从 data.storages 中获取数据
	var storagesData []interface{}
	if data, ok := result["data"].(map[string]interface{}); ok {
		if storages, ok := data["storages"].([]interface{}); ok {
			storagesData = storages
		}
	}

	if len(storagesData) == 0 {
		fmt.Println("✗ 响应格式错误：未找到 storages 数组")
		return nil, fmt.Errorf("响应格式错误")
	}

	fmt.Printf("✓ 找到 %d 个存储位置\n\n", len(storagesData))

	storages := make([]map[string]interface{}, 0)
	for i, item := range storagesData {
		if storage, ok := item.(map[string]interface{}); ok {
			storages = append(storages, storage)
			id := fmt.Sprintf("%v", storage["id"])
			name := fmt.Sprintf("%v", storage["name"])
			storageType := fmt.Sprintf("%v", storage["type"])
			total := fmt.Sprintf("%v", storage["total"])
			avail := fmt.Sprintf("%v", storage["avail"])
			fmt.Printf("  [%d] %s (id=%s, type=%s, total=%sMB, avail=%sMB)\n", i+1, name, id, storageType, total, avail)
		}
	}
	fmt.Println()

	return storages, nil
}

func (c *VDIAPIClient) GetNetworks(vtpID int) ([]map[string]interface{}, error) {
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("获取网络接口 (vtp_id=%d)\n", vtpID)
	fmt.Println("═══════════════════════════════════════════════════════")

	path := fmt.Sprintf("/v1/networks?vtp_id=%d", vtpID)
	result, err := c.CallAPI("GET", path, nil)
	if err != nil {
		return nil, err
	}

	errorCode, _ := result["error_code"].(float64)
	if errorCode != 0 {
		errorMsg, _ := result["error_message"].(string)
		fmt.Printf("✗ 获取网络接口失败: %s\n\n", errorMsg)
		return nil, fmt.Errorf("获取网络接口失败: %s", errorMsg)
	}

	// 从 data.networks 中获取数据
	var networksData []interface{}
	if data, ok := result["data"].(map[string]interface{}); ok {
		if networks, ok := data["networks"].([]interface{}); ok {
			networksData = networks
		}
	}

	if len(networksData) == 0 {
		fmt.Println("✗ 响应格式错误：未找到 networks 数组")
		return nil, fmt.Errorf("响应格式错误")
	}

	fmt.Printf("✓ 找到 %d 个网络接口\n\n", len(networksData))

	networks := make([]map[string]interface{}, 0)
	for i, item := range networksData {
		if network, ok := item.(map[string]interface{}); ok {
			networks = append(networks, network)
			id := fmt.Sprintf("%v", network["id"])
			name := fmt.Sprintf("%v", network["name"])
			mode := fmt.Sprintf("%v", network["mode"])
			fmt.Printf("  [%d] %s (id=%s, mode=%s)\n", i+1, name, id, mode)
		}
	}
	fmt.Println()

	return networks, nil
}

func (c *VDIAPIClient) CreateServer(resourceID, vtpID int, runPositionID, diskID, storageID, networkID string, count int) (*CreateServerResponse, error) {
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("创建虚拟机")
	fmt.Println("═══════════════════════════════════════════════════════")

	// 获取运行位置以确定 host.id 和 run_position.id
	positions, err := c.GetRunPositions(vtpID)
	if err != nil {
		return nil, fmt.Errorf("获取运行位置失败: %w", err)
	}

	var selectedPosition map[string]interface{}
	for _, pos := range positions {
		posID := fmt.Sprintf("%v", pos["id"])
		if posID == runPositionID {
			selectedPosition = pos
			break
		}
	}

	if selectedPosition == nil {
		return nil, fmt.Errorf("未找到指定的运行位置: %s", runPositionID)
	}

	id := fmt.Sprintf("%v", selectedPosition["id"])
	fatherID := fmt.Sprintf("%v", selectedPosition["father_id"])

	// host.id 取 father_id
	hostID := fatherID

	// run_position.id 根据规则确定
	var finalRunPositionID string
	if id == fatherID {
		finalRunPositionID = "" // id == father_id 时为空
	} else {
		finalRunPositionID = id // id != father_id 时取 id
	}

	fmt.Printf("【参数解析】运行位置: id=%s, father_id=%s\n", id, fatherID)
	fmt.Printf("  → host.id: %s\n", hostID)
	fmt.Printf("  → run_position.id: %s\n", finalRunPositionID)

	// 构建请求
	reqBody := map[string]interface{}{
		"resource":     map[string]int{"id": resourceID},
		"host":         map[string]string{"id": hostID},
		"run_position": map[string]string{"id": finalRunPositionID},
		"disk":         map[string]string{"id": diskID},
		"storage":      map[string]string{"id": storageID},
		"network":      map[string]string{"id": networkID},
		"servers":      map[string]int{"count": count},
	}

	fmt.Println("\n【创建参数】")
	fmt.Printf("  资源ID: %d\n", resourceID)
	fmt.Printf("  主机位置: %s\n", hostID)
	fmt.Printf("  运行位置: %s\n", finalRunPositionID)
	fmt.Printf("  个人盘: %s\n", diskID)
	fmt.Printf("  存储: %s\n", storageID)
	fmt.Printf("  网络: %s\n", networkID)
	fmt.Printf("  数量: %d\n", count)

	result, err := c.CallAPI("POST", "/v1/servers", reqBody)
	if err != nil {
		return nil, err
	}

	errorCode, _ := result["error_code"].(float64)
	errorMsg, _ := result["error_message"].(string)

	if errorCode != 0 {
		fmt.Printf("✗ 创建虚拟机失败: [%d] %s\n\n", int(errorCode), errorMsg)
		return nil, fmt.Errorf("创建虚拟机失败: %s", errorMsg)
	}

	// 解析响应
	var response CreateServerResponse
	if data, ok := result["data"].(map[string]interface{}); ok {
		if taskID, ok := data["task_id"].(float64); ok {
			response.Data.TaskID = int(taskID)
		}
		if serverIDs, ok := data["server_id"].([]interface{}); ok {
			for _, id := range serverIDs {
				response.Data.ServerID = append(response.Data.ServerID, fmt.Sprintf("%v", id))
			}
		}
	}

	response.ErrorCode = int(errorCode)
	response.ErrorMessage = errorMsg

	fmt.Printf("✓ 创建任务已提交!\n")
	fmt.Printf("  任务ID: %d\n", response.Data.TaskID)
	fmt.Printf("  虚拟机IDs: %v\n", response.Data.ServerID)
	fmt.Println("\n【提示】虚拟机创建是异步操作，正在查询虚拟机状态...")

	// 查询虚拟机状态（使用虚拟机ID）
	for _, serverID := range response.Data.ServerID {
		c.GetServerStatus(serverID)
	}

	return &response, nil
}

func (c *VDIAPIClient) GetServerStatus(serverID string) error {
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("查询虚拟机状态 (server_id=%s)\n", serverID)
	fmt.Println("═══════════════════════════════════════════════════════")

	path := fmt.Sprintf("/v1/servers/%s", serverID)
	result, err := c.CallAPI("GET", path, nil)
	if err != nil {
		fmt.Printf("✗ 查询虚拟机状态失败: %v\n\n", err)
		return err
	}

	errorCode, _ := result["error_code"].(float64)
	if errorCode != 0 {
		errorMsg, _ := result["error_message"].(string)
		fmt.Printf("✗ 查询虚拟机状态失败: [%d] %s\n\n", int(errorCode), errorMsg)
		return fmt.Errorf("查询虚拟机状态失败: %s", errorMsg)
	}

	// 打印完整响应以便调试
	if data, ok := result["data"].(map[string]interface{}); ok {
		jsonBytes, _ := json.MarshalIndent(data, "", "  ")
		fmt.Printf("【虚拟机状态】\n%s\n", string(jsonBytes))

		// 检查常见的状态字段
		if status, ok := data["status"].(string); ok {
			fmt.Printf("\n虚拟机状态: %s\n", status)
			if status == "failed" || status == "error" {
				if errMsg, ok := data["error_message"].(string); ok {
					fmt.Printf("错误信息: %s\n", errMsg)
				}
			}
		}
		if progress, ok := data["progress"].(float64); ok {
			fmt.Printf("进度: %.0f%%\n", progress)
		}
	} else {
		fmt.Println("✗ 响应格式错误")
	}

	fmt.Println()
	return nil
}

// GetCloneRunPosition 获取克隆虚拟机的运行位置
func (c *VDIAPIClient) GetCloneRunPosition(vtpID int, storageID string) ([]map[string]interface{}, error) {
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("获取克隆虚拟机运行位置 (vtp_id=%d, storage=%s)\n", vtpID, storageID)
	fmt.Println("═══════════════════════════════════════════════════════")

	path := fmt.Sprintf("/v1/vtp/clone_run_position?vtp_id=%d&storage=%s", vtpID, storageID)
	result, err := c.CallAPI("GET", path, nil)
	if err != nil {
		return nil, err
	}

	errorCode, _ := result["error_code"].(float64)
	if errorCode != 0 {
		errorMsg, _ := result["error_message"].(string)
		fmt.Printf("✗ 获取克隆运行位置失败: %s\n\n", errorMsg)
		return nil, fmt.Errorf("获取克隆运行位置失败: %s", errorMsg)
	}

	// 从 data 中获取运行位置列表
	var positionsData []interface{}
	if data, ok := result["data"].(map[string]interface{}); ok {
		if data2, ok := data["data"].([]interface{}); ok {
			positionsData = data2
		}
	}

	if len(positionsData) == 0 {
		fmt.Println("✗ 响应格式错误：未找到运行位置列表")
		return nil, fmt.Errorf("响应格式错误")
	}

	fmt.Printf("✓ 找到 %d 个克隆运行位置\n\n", len(positionsData))

	positions := make([]map[string]interface{}, 0)
	for i, item := range positionsData {
		if pos, ok := item.(map[string]interface{}); ok {
			positions = append(positions, pos)
			runPos := fmt.Sprintf("%v", pos["run_position"])
			name := fmt.Sprintf("%v", pos["name"])
			fmt.Printf("  [%d] %s (run_position=%s)\n", i+1, name, runPos)
		}
	}
	fmt.Println()

	return positions, nil
}

// GetHostInfo 获取指定主机信息
func (c *VDIAPIClient) GetHostInfo(vtpID int, hostID string) error {
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("获取主机信息 (vtp_id=%d, host_id=%s)\n", vtpID, hostID)
	fmt.Println("═══════════════════════════════════════════════════════")

	path := fmt.Sprintf("/v1/vtp/host?vtp_id=%d&host_id=%s", vtpID, hostID)
	result, err := c.CallAPI("GET", path, nil)
	if err != nil {
		return err
	}

	errorCode, _ := result["error_code"].(float64)
	if errorCode != 0 {
		errorMsg, _ := result["error_message"].(string)
		fmt.Printf("✗ 获取主机信息失败: %s\n\n", errorMsg)
		return fmt.Errorf("获取主机信息失败: %s", errorMsg)
	}

	// 解析主机信息
	if data, ok := result["data"].(map[string]interface{}); ok {
		hostIP := fmt.Sprintf("%v", data["host_ip"])
		runningVMs := fmt.Sprintf("%v", data["running_vms"])
		cpuRatio := fmt.Sprintf("%v", data["cpu_ratio"])
		memRatio := fmt.Sprintf("%v", data["mem_ratio"])

		fmt.Println("【主机信息】")
		fmt.Printf("  IP地址:       %s\n", hostIP)
		fmt.Printf("  运行虚拟机:   %s 台\n", runningVMs)
		fmt.Printf("  CPU使用率:    %s%%\n", cpuRatio)
		fmt.Printf("  内存使用率:   %s%%\n", memRatio)
		fmt.Println()
	} else {
		fmt.Println("✗ 响应格式错误")
		return fmt.Errorf("响应格式错误")
	}

	return nil
}

// CloneServer 克隆虚拟机
func (c *VDIAPIClient) CloneServer(serverID int, cloneName, storageID, runPosition, groupID, note string, startVM, errorMigrate bool, vtpID int) error {
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("克隆虚拟机 (server_id=%d)\n", serverID)
	fmt.Println("═══════════════════════════════════════════════════════")

	// 构建请求参数
	start := "0"
	if startVM {
		start = "1"
	}

	errorMigrateVal := 0
	if errorMigrate {
		errorMigrateVal = 1
	}

	reqBody := map[string]interface{}{
		"server": map[string]interface{}{
			"clonename":     cloneName,
			"storage":       storageID,
			"start":         start,
			"error_migrate": errorMigrateVal,
			"run_postion":   runPosition,
			"group":         groupID,
			"note":          note,
			"vtp_id":        vtpID,
		},
	}

	fmt.Println("\n【克隆参数】")
	fmt.Printf("  源虚拟机ID:   %d\n", serverID)
	fmt.Printf("  克隆名称:     %s\n", cloneName)
	fmt.Printf("  存储位置:     %s\n", storageID)
	fmt.Printf("  运行位置:     %s\n", runPosition)
	fmt.Printf("  虚拟机组:     %s\n", groupID)
	fmt.Printf("  克隆后启动:   %v\n", startVM)
	fmt.Printf("  故障迁移:     %v\n", errorMigrate)
	fmt.Printf("  描述:         %s\n", note)
	fmt.Printf("  VTP ID:       %d\n", vtpID)

	path := fmt.Sprintf("/v1/vtp/servers/clone/%d", serverID)
	result, err := c.CallAPI("PUT", path, reqBody)
	if err != nil {
		return err
	}

	success, _ := result["success"].(float64)
	if success != 1 {
		// 尝试从 message 或 error_message 字段获取错误信息
		errorMsg, _ := result["message"].(string)
		if errorMsg == "" {
			errorMsg, _ = result["error_message"].(string)
		}
		if errorMsg == "" {
			errorMsg = "未知错误"
		}
		fmt.Printf("✗ 克隆虚拟机失败: %s\n\n", errorMsg)
		return fmt.Errorf("克隆虚拟机失败: %s", errorMsg)
	}

	upid, _ := result["data"].(string)
	fmt.Printf("✓ 克隆任务已提交!\n")
	fmt.Printf("  UPID: %s\n", upid)
	fmt.Println("\n【提示】虚拟机克隆是异步操作，请使用 UPID 查询进度")

	return nil
}

// GetTaskStatus 查询任务状态（保留用于其他任务查询）
func (c *VDIAPIClient) GetTaskStatus(taskID int) error {
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("查询任务状态 (task_id=%d)\n", taskID)
	fmt.Println("═══════════════════════════════════════════════════════")

	path := fmt.Sprintf("/v1/tasks/%d", taskID)
	result, err := c.CallAPI("GET", path, nil)
	if err != nil {
		fmt.Printf("✗ 查询任务状态失败: %v\n\n", err)
		return err
	}

	errorCode, _ := result["error_code"].(float64)
	if errorCode != 0 {
		errorMsg, _ := result["error_message"].(string)
		fmt.Printf("✗ 查询任务状态失败: [%d] %s\n\n", int(errorCode), errorMsg)
		return fmt.Errorf("查询任务状态失败: %s", errorMsg)
	}

	// 打印完整响应以便调试
	if data, ok := result["data"].(map[string]interface{}); ok {
		jsonBytes, _ := json.MarshalIndent(data, "", "  ")
		fmt.Printf("【任务状态】\n%s\n", string(jsonBytes))

		// 检查常见的状态字段
		if status, ok := data["status"].(string); ok {
			fmt.Printf("\n任务状态: %s\n", status)
			if status == "failed" || status == "error" {
				if errMsg, ok := data["error_message"].(string); ok {
					fmt.Printf("错误信息: %s\n", errMsg)
				}
			}
		}
		if progress, ok := data["progress"].(float64); ok {
			fmt.Printf("进度: %.0f%%\n", progress)
		}
	}

	fmt.Println()
	return nil
}

func (c *VDIAPIClient) GetServerGroups(vtpID int) error {
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("获取虚拟机分组 (vtp_id=%d)\n", vtpID)
	fmt.Println("═══════════════════════════════════════════════════════")

	path := fmt.Sprintf("/v1/vtp/servers_group?vtp_id=%d", vtpID)
	result, err := c.CallAPI("GET", path, nil)
	if err != nil {
		fmt.Printf("✗ 获取虚拟机分组失败: %v\n\n", err)
		return err
	}

	errorCode, _ := result["error_code"].(float64)
	if errorCode != 0 {
		errorMsg, _ := result["error_message"].(string)
		fmt.Printf("✗ 获取虚拟机分组失败: [%d] %s\n\n", int(errorCode), errorMsg)
		return fmt.Errorf("获取虚拟机分组失败: %s", errorMsg)
	}

	// 从 data.list 中获取分组列表
	var groups []interface{}
	if data, ok := result["data"].(map[string]interface{}); ok {
		if list, ok := data["list"].([]interface{}); ok {
			groups = list
		}
	}

	if len(groups) == 0 {
		fmt.Println("✗ 响应格式错误：未找到分组列表")
		return fmt.Errorf("响应格式错误")
	}

	fmt.Printf("✓ 找到 %d 个虚拟机分组\n\n", len(groups))

	for i, item := range groups {
		if group, ok := item.(map[string]interface{}); ok {
			id := fmt.Sprintf("%v", group["id"])
			name := fmt.Sprintf("%v", group["name"])
			fmt.Printf("  [%d] %s (id=%s)\n", i+1, name, id)
		}
	}
	fmt.Println()

	return nil
}

func (c *VDIAPIClient) MigrateServers(vtpID int, targetDir string, vmIDs []int) error {
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("移动虚拟机到其他分组")
	fmt.Println("═══════════════════════════════════════════════════════")

	// 构建请求
	reqBody := map[string]interface{}{
		"vtp_id":     vtpID,
		"target_dir": targetDir,
		"vms":        vmIDs,
		"action":     "migrate_vms",
	}

	fmt.Printf("【移动参数】\n")
	fmt.Printf("  VTP ID: %d\n", vtpID)
	fmt.Printf("  目标分组ID: %s\n", targetDir)
	fmt.Printf("  虚拟机IDs: %v\n", vmIDs)

	result, err := c.CallAPI("PUT", "/v1/vtp/servers", reqBody)
	if err != nil {
		fmt.Printf("✗ 移动虚拟机失败: %v\n\n", err)
		return err
	}

	success, _ := result["success"].(float64)
	if success != 1 {
		errorMsg, _ := result["error_message"].(string)
		fmt.Printf("✗ 移动虚拟机失败: %s\n\n", errorMsg)
		return fmt.Errorf("移动虚拟机失败: %s", errorMsg)
	}

	upid, _ := result["data"].(string)
	fmt.Printf("✓ 移动任务已提交!\n")
	fmt.Printf("  UPID: %s\n", upid)
	fmt.Println("\n【提示】虚拟机移动是异步操作，请使用 UPID 查询进度")

	return nil
}

// CreateServerResponse 创建服务器响应
type CreateServerResponse struct {
	ErrorCode    int
	ErrorMessage string
	Data         struct {
		TaskID   int
		ServerID []string
	}
}

func printUsage() {
	fmt.Println("╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║              VDI API 独立测试工具                         ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("用法: go run scripts/vdi_test_standalone.go <命令> [参数]")
	fmt.Println()
	fmt.Println("命令:")
	fmt.Println("  list                    - 获取虚拟机列表")
	fmt.Println("  operate <vmID> <action> - 操作虚拟机")
	fmt.Println("  run-pos <vtpID>         - 获取运行位置")
	fmt.Println("  storages <vtpID>        - 获取存储位置")
	fmt.Println("  networks <vtpID>        - 获取网络接口")
	fmt.Println("  create <resID> <vtpID> <runPosID> <diskID> <storageID> <networkID> [count]")
	fmt.Println("                          - 创建虚拟机")
	fmt.Println("  server-status <vmID>    - 查询虚拟机状态")
	fmt.Println("  server-groups <vtpID>   - 获取虚拟机分组")
	fmt.Println("  migrate <vtpID> <targetDir> <vmIDs...>")
	fmt.Println("                          - 移动虚拟机到分组")
	fmt.Println("  clone <vmID> <name> <storage> <runPos> <group> <note> <start> <errMig> <vtpID>")
	fmt.Println("                          - 克隆虚拟机")
	fmt.Println("  clone-run-pos <vtpID> <storageID>")
	fmt.Println("                          - 获取克隆虚拟机的运行位置")
	fmt.Println("  host-info <vtpID> <hostID>")
	fmt.Println("                          - 获取指定主机信息")
	fmt.Println()
	fmt.Println("操作类型: start, stop, restart, suspend, resume")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  go run scripts/vdi_test_standalone.go list")
	fmt.Println("  go run scripts/vdi_test_standalone.go operate 42 start")
	fmt.Println("  go run scripts/vdi_test_standalone.go run-pos 1")
	fmt.Println("  go run scripts/vdi_test_standalone.go storages 1")
	fmt.Println("  go run scripts/vdi_test_standalone.go networks 1")
	fmt.Println("  go run scripts/vdi_test_standalone.go create 1 1 169ee1724b651 vs_rep2 vs_rep2 br_eth0 2")
	fmt.Println("  go run scripts/vdi_test_standalone.go server-status 95")
	fmt.Println("  go run scripts/vdi_test_standalone.go server-groups 1")
	fmt.Println("  go run scripts/vdi_test_standalone.go migrate 1 fd3dd9b2ac3f 8 9")
	fmt.Println("  go run scripts/vdi_test_standalone.go clone-run-pos 1 af269268_vs_vol_rep2")
	fmt.Println("  go run scripts/vdi_test_standalone.go host-info 1 host-0894efbccf03")
	fmt.Println("  go run scripts/vdi_test_standalone.go clone 95 \"测试克隆\" af269268_vs_vol_rep2 host-0894efbccf03 d64c800964fd \"测试\" 1 0 1")
	fmt.Println()
	fmt.Printf("配置: VDI服务器 = %s, 用户 = %s\n", VdiBaseURL, VdiUser)
	os.Exit(0)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
	}

	command := os.Args[1]

	client := NewVDIAPIClient()
	if err := client.Authenticate(); err != nil {
		fmt.Printf("❌ 认证失败: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "list":
		if err := client.ListVMs(); err != nil {
			fmt.Printf("❌ 获取虚拟机列表失败: %v\n", err)
		}

	case "operate":
		if len(os.Args) < 4 {
			fmt.Println("请提供VM ID和操作类型")
			os.Exit(1)
		}
		vmID := os.Args[2]
		action := os.Args[3]
		if err := client.OperateVM(vmID, action); err != nil {
			fmt.Printf("❌ 操作失败: %v\n", err)
			os.Exit(1)
		}

	case "run-pos", "run-position":
		if len(os.Args) < 3 {
			fmt.Println("请提供VTP ID")
			os.Exit(1)
		}
		vtpID, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("无效的VTP ID: %v\n", err)
			os.Exit(1)
		}
		if _, err := client.GetRunPositions(vtpID); err != nil {
			fmt.Printf("❌ 获取运行位置失败: %v\n", err)
		}

	case "storages":
		if len(os.Args) < 3 {
			fmt.Println("请提供VTP ID")
			os.Exit(1)
		}
		vtpID, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("无效的VTP ID: %v\n", err)
			os.Exit(1)
		}
		if _, err := client.GetStorages(vtpID); err != nil {
			fmt.Printf("❌ 获取存储位置失败: %v\n", err)
		}

	case "networks":
		if len(os.Args) < 3 {
			fmt.Println("请提供VTP ID")
			os.Exit(1)
		}
		vtpID, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("无效的VTP ID: %v\n", err)
			os.Exit(1)
		}
		if _, err := client.GetNetworks(vtpID); err != nil {
			fmt.Printf("❌ 获取网络接口失败: %v\n", err)
		}

	case "create":
		if len(os.Args) < 8 {
			fmt.Println("请提供完整参数: <resID> <vtpID> <runPosID> <diskID> <storageID> <networkID> [count]")
			os.Exit(1)
		}
		resourceID, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("无效的资源ID: %v\n", err)
			os.Exit(1)
		}
		vtpID, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Printf("无效的VTP ID: %v\n", err)
			os.Exit(1)
		}
		runPositionID := os.Args[4]
		diskID := os.Args[5]
		storageID := os.Args[6]
		networkID := os.Args[7]
		count := 1
		if len(os.Args) >= 9 {
			count, err = strconv.Atoi(os.Args[8])
			if err != nil {
				fmt.Printf("无效的数量: %v\n", err)
				os.Exit(1)
			}
		}
		if _, err := client.CreateServer(resourceID, vtpID, runPositionID, diskID, storageID, networkID, count); err != nil {
			fmt.Printf("❌ 创建虚拟机失败: %v\n", err)
			os.Exit(1)
		}

	case "server-status", "task-status":
		if len(os.Args) < 3 {
			fmt.Println("请提供虚拟机ID")
			os.Exit(1)
		}
		serverID := os.Args[2]
		if err := client.GetServerStatus(serverID); err != nil {
			fmt.Printf("❌ 查询虚拟机状态失败: %v\n", err)
		}

	case "server-groups":
		if len(os.Args) < 3 {
			fmt.Println("请提供VTP ID")
			os.Exit(1)
		}
		vtpID, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("无效的VTP ID: %v\n", err)
			os.Exit(1)
		}
		if err := client.GetServerGroups(vtpID); err != nil {
			fmt.Printf("❌ 获取虚拟机分组失败: %v\n", err)
		}

	case "migrate":
		if len(os.Args) < 5 {
			fmt.Println("请提供完整参数: <vtpID> <targetDir> <vmIDs...>")
			fmt.Println("示例: go run scripts/vdi_test_standalone.go migrate 1 fd3dd9b2ac3f 8 9 10")
			os.Exit(1)
		}
		vtpID, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("无效的VTP ID: %v\n", err)
			os.Exit(1)
		}
		targetDir := os.Args[3]

		// 解析虚拟机ID列表
		var vmIDs []int
		for i := 4; i < len(os.Args); i++ {
			vmID, err := strconv.Atoi(os.Args[i])
			if err != nil {
				fmt.Printf("无效的虚拟机ID: %s\n", os.Args[i])
				os.Exit(1)
			}
			vmIDs = append(vmIDs, vmID)
		}

		if err := client.MigrateServers(vtpID, targetDir, vmIDs); err != nil {
			fmt.Printf("❌ 移动虚拟机失败: %v\n", err)
			os.Exit(1)
		}

	case "clone-run-pos":
		if len(os.Args) < 4 {
			fmt.Println("请提供完整参数: <vtpID> <storageID>")
			fmt.Println("示例: go run scripts/vdi_test_standalone.go clone-run-pos 1 af269268_vs_vol_rep2")
			os.Exit(1)
		}
		vtpID, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("无效的VTP ID: %v\n", err)
			os.Exit(1)
		}
		storageID := os.Args[3]
		if _, err := client.GetCloneRunPosition(vtpID, storageID); err != nil {
			fmt.Printf("❌ 获取克隆运行位置失败: %v\n", err)
		}

	case "host-info":
		if len(os.Args) < 4 {
			fmt.Println("请提供完整参数: <vtpID> <hostID>")
			fmt.Println("示例: go run scripts/vdi_test_standalone.go host-info 1 host-0894efbccf03")
			os.Exit(1)
		}
		vtpID, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("无效的VTP ID: %v\n", err)
			os.Exit(1)
		}
		hostID := os.Args[3]
		if err := client.GetHostInfo(vtpID, hostID); err != nil {
			fmt.Printf("❌ 获取主机信息失败: %v\n", err)
		}

	case "clone":
		if len(os.Args) < 11 {
			fmt.Println("请提供完整参数: <vmID> <cloneName> <storage> <runPos> <group> <note> <start> <errorMigrate> <vtpID>")
			fmt.Println("示例: go run scripts/vdi_test_standalone.go clone 95 \"测试克隆\" af269268_vs_vol_rep2 host-0894efbccf03 d64c800964fd \"测试\" 1 0 1")
			os.Exit(1)
		}
		serverID, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("无效的虚拟机ID: %v\n", err)
			os.Exit(1)
		}
		cloneName := os.Args[3]
		storageID := os.Args[4]
		runPosition := os.Args[5]
		groupID := os.Args[6]
		note := os.Args[7]

		startVM := false
		if os.Args[8] == "1" || os.Args[8] == "true" {
			startVM = true
		}

		errorMigrate := false
		if os.Args[9] == "1" || os.Args[9] == "true" {
			errorMigrate = true
		}

		vtpID, err := strconv.Atoi(os.Args[10])
		if err != nil {
			fmt.Printf("无效的VTP ID: %v\n", err)
			os.Exit(1)
		}

		if err := client.CloneServer(serverID, cloneName, storageID, runPosition, groupID, note, startVM, errorMigrate, vtpID); err != nil {
			fmt.Printf("❌ 克隆虚拟机失败: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Printf("未知命令: %s\n", command)
		printUsage()
	}
}
