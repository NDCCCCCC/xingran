package main
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)
func main() {
	// 测试创建邮件配置的API
	payload := map[string]interface{}{
		"configName": "测试邮件",
		"host":       "21.0.120.22",
		"port":       25,
		"username":   "test@example.com",
		"password":   "test123",
		"fromName":   "测试",
		"useSsl":     false,
		"isDefault":  false,
		"status":     0,
	}
	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(
		"http://10.62.10.33:9000/api/v1/system/settings/email-configs",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("状态码: %d\n", resp.StatusCode)
	fmt.Printf("响应: %s\n", string(body))
}
