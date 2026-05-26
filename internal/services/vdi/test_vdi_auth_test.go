package vdi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"
	"time"
)

const (
	testVDIURL      = "https://10.62.0.79:4430"
	testVDIUsername = "admin"
	testVDIPassword = "sangfor@2020"
)

func TestVDIAPIAuthentication(t *testing.T) {
	fmt.Println("=== VDI API 认证测试 ===")

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// 测试1: 尝试 Sangfor VDI 标准认证格式
	fmt.Println("\n测试1: Sangfor VDI 标准认证格式")
	testSangforStandardAuth(t, client)

	// 测试2: 尝试代码中使用的格式
	fmt.Println("\n测试2: 代码中使用的认证格式")
	testCodeImplementationFormat(t, client)
}

func testSangforStandardAuth(t *testing.T, client *http.Client) {
	// Sangfor VCI 标准认证格式
	authURL := fmt.Sprintf("%s/API/V1.0/Auth/Login", testVDIURL)

	payload := map[string]string{
		"username": testVDIUsername,
		"password": testVDIPassword,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("JSON序列化失败: %v", err)
	}

	req, err := http.NewRequest("POST", authURL, bytes.NewReader(jsonData))
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	fmt.Printf("请求URL: %s\n", authURL)
	fmt.Printf("请求体: %s\n", string(jsonData))

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("响应状态码: %d\n", resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("响应体: %s\n", string(body))

	if resp.StatusCode == 200 {
		fmt.Println("✅ Sangfor标准认证成功")
	} else {
		fmt.Printf("⚠️  Sangfor标准认证返回状态码: %d\n", resp.StatusCode)
	}
}

func testCodeImplementationFormat(t *testing.T, client *http.Client) {
	// 代码中使用的认证格式
	authURL := fmt.Sprintf("%s/v1/auth/tokens", testVDIURL)

	payload := map[string]interface{}{
		"auth": map[string]string{
			"name":     testVDIUsername,
			"password": testVDIPassword,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("JSON序列化失败: %v", err)
	}

	req, err := http.NewRequest("POST", authURL, bytes.NewReader(jsonData))
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	fmt.Printf("请求URL: %s\n", authURL)
	fmt.Printf("请求体: %s\n", string(jsonData))

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("响应状态码: %d\n", resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("响应体: %s\n", string(body))

	if resp.StatusCode == 200 {
		fmt.Println("✅ 代码实现格式认证成功")

		// 解析响应
		var respData struct {
			ErrorCode    int    `json:"error_code"`
			ErrorMessage string `json:"error_message"`
			Data         struct {
				Token struct {
					TenantID  int    `json:"tenant_id"`
					AuthToken string `json:"auth_token"`
				} `json:"token"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &respData); err != nil {
			log.Printf("解析响应失败: %v\n", err)
			return
		}

		if respData.ErrorCode == 0 {
			fmt.Printf("✅ 认证成功，Token: %s\n", respData.Data.Token.AuthToken[:min(20, len(respData.Data.Token.AuthToken))]+"...")
		} else {
			fmt.Printf("❌ 认证失败，错误码: %d, 错误信息: %s\n", respData.ErrorCode, respData.ErrorMessage)
		}
	} else {
		fmt.Printf("⚠️  代码实现格式返回状态码: %d\n", resp.StatusCode)
	}
}

// TestVDIAPIWithEncryptedPassword 测试使用加密密码的认证
func TestVDIAPIWithEncryptedPassword(t *testing.T) {
	fmt.Println("=== VDI API 加密密码认证测试 ===")

	// 1. 加密密码
	encrypted := encryptVDIPassword(testVDIPassword)
	fmt.Printf("加密后的密码: %s...\n", encrypted[:min(30, len(encrypted))])

	// 2. 解密密码
	decrypted := decryptVDIPassword(encrypted)
	fmt.Printf("解密后的密码: %s\n", decrypted)

	if decrypted != testVDIPassword {
		t.Fatalf("解密失败: expected='%s', got='%s'", testVDIPassword, decrypted)
	}

	// 3. 使用解密后的密码进行认证
	fmt.Println("\n使用解密后的密码进行认证...")
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	ctx := context.Background()

	// 模拟真实的认证流程
	var server struct {
		Endpoint          string
		Username          string
		PasswordEncrypted string
	}

	server.Endpoint = testVDIURL
	server.Username = testVDIUsername
	server.PasswordEncrypted = encrypted

	// 检查解密是否成功
	password := decryptVDIPassword(server.PasswordEncrypted)
	if password == "" {
		t.Fatalf("解密失败，得到空字符串")
	}

	fmt.Printf("✅ 密码解密成功: %s\n", password)

	// 尝试认证
	authURL := fmt.Sprintf("%s/v1/auth/tokens", server.Endpoint)

	payload := map[string]interface{}{
		"auth": map[string]string{
			"name":     server.Username,
			"password": password, // 使用解密后的密码
		},
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", authURL, bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ 认证请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("响应状态码: %d\n", resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("响应体: %s\n", string(body))

	if resp.StatusCode == 200 {
		fmt.Println("✅ 使用加密密码的认证成功")
	} else {
		fmt.Printf("❌ 使用加密密码的认证失败，状态码: %d\n", resp.StatusCode)
	}
}
