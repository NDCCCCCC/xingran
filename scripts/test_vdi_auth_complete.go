package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	// 正确的VDI服务器配置
	baseURL := "https://10.62.0.79:6060"
	username := "admin"
	password := "sangfor@2020"

	// 创建HTTP客户端（跳过TLS验证）
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	fmt.Println("=== VDI API 完整认证测试 ===\n")

	// 步骤1：获取认证令牌
	token := authenticate(client, baseURL, username, password)
	if token == "" {
		fmt.Println("❌ 认证失败，无法继续测试")
		return
	}

	// 步骤2：使用令牌测试其他接口
	testWithToken(client, baseURL, token)
}

// authenticate 执行认证并获取令牌
func authenticate(client *http.Client, baseURL, username, password string) string {
	fmt.Println("【步骤1】认证并获取令牌")
	fmt.Println("=====================================")

	authPayload := map[string]interface{}{
		"auth": map[string]string{
			"name":     username,
			"password": password,
		},
	}

	jsonData, _ := json.Marshal(authPayload)
	authURL := baseURL + "/v1/auth/tokens"

	fmt.Printf("请求端点: %s\n", authURL)
	fmt.Printf("请求体: %s\n\n", string(jsonData))

	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ 创建请求失败: %v\n", err)
		return ""
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("响应状态码: %d\n", resp.StatusCode)
	fmt.Printf("Content-Type: %s\n\n", resp.Header.Get("Content-Type"))

	// 尝试解析JSON（即使Content-Type不是application/json）
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("❌ JSON解析失败: %v\n", err)
		fmt.Printf("响应体: %s\n", string(body))
		return ""
	}

	fmt.Printf("✅ JSON解析成功！\n")
	pretty, _ := json.MarshalIndent(result, "  ", "  ")
	fmt.Printf("响应内容:\n%s\n\n", string(pretty))

	// 提取token
	if token, ok := result["token"].(map[string]interface{}); ok {
		if authToken, ok := token["auth_token"].(string); ok {
			tenantID := token["tenant_id"]
			fmt.Printf("🎉 认证成功！\n")
			fmt.Printf("   tenant_id: %v\n", tenantID)
			fmt.Printf("   auth_token: %s...\n\n", authToken[:50])
			return authToken
		}
	}

	fmt.Println("❌ 响应中没有找到 auth_token")
	return ""
}

// testWithToken 使用令牌测试其他接口
func testWithToken(client *http.Client, baseURL, token string) {
	fmt.Println("\n【步骤2】使用令牌测试其他接口")
	fmt.Println("=====================================")

	endpoints := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{"虚拟机列表", "GET", "/v1/vms", nil},
		{"资源组列表", "GET", "/v1/resource-groups", nil},
		{"所有虚拟机（按资源组）", "GET", "/v1/resource-groups/vms", nil},
		{"角色列表", "GET", "/v1/roles", nil},
		{"用户列表", "GET", "/v1/users", nil},
	}

	successCount := 0
	for _, ep := range endpoints {
		fmt.Printf("\n测试: %s\n", ep.name)
		fmt.Printf("请求: %s %s\n", ep.method, ep.path)

		var req *http.Request
		var err error

		if ep.method == "GET" {
			req, err = http.NewRequest("GET", baseURL+ep.path, nil)
		} else if ep.method == "POST" && ep.body != nil {
			jsonData, _ := json.Marshal(ep.body)
			req, err = http.NewRequest("POST", baseURL+ep.path, bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
		} else {
			continue
		}

		if err != nil {
			fmt.Printf("  ❌ 创建请求失败: %v\n", err)
			continue
		}

		// 设置认证令牌
		req.Header.Set("Auth-Token", token)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("  ❌ 请求失败: %v\n", err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		fmt.Printf("  状态码: %d\n", resp.StatusCode)

		// 解析响应
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err == nil {
			if errorCode, ok := result["error_code"].(float64); ok {
				if errorCode == 0 {
					fmt.Printf("  ✅ 成功！\n")
					successCount++
				} else {
					errorMsg := "未知错误"
					if msg, ok := result["error_message"].(string); ok {
						errorMsg = msg
					}
					fmt.Printf("  ❌ API错误: %s\n", errorMsg)
				}
			}
		} else {
			fmt.Printf("  ⚠️  无法解析JSON响应\n")
		}
	}

	fmt.Printf("\n=====================================\n")
	fmt.Printf("测试完成：%d/%d 成功\n", successCount, len(endpoints))
}
