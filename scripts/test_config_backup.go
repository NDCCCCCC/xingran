package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// 设备配置
type DeviceConfig struct {
	Name     string
	IP       string
	Port     int
	Vendor   string
	Username string
	Password string
}

func main() {
	// 设备列表配置 - 请修改为实际的用户名和密码
	devices := []DeviceConfig{
		{
			Name:     "CX_HB_JZ_R_RJ_30SPU1_1",
			IP:       "10.62.47.117",
			Port:     22,
			Vendor:   "ruijie",
			Username: "ninedrunk",
			Password: ")(PO09po", // 请修改为实际密码
		},
		{
			Name:     "CX_HB_XT_R_RJ_RSR10_1",
			IP:       "10.62.47.120",
			Port:     22,
			Vendor:   "ruijie",
			Username: "ninedrunk",
			Password: ")(PO09po", // 请修改为实际密码
		},
		{
			Name:     "CXHUB-SW-HW-S5735-XN-xianAn-01",
			IP:       "10.62.117.189",
			Port:     22,
			Vendor:   "huawei",
			Username: "ninedrunk",
			Password: ")(PO09po", // 请修改为实际密码
		},
		// 添加更多设备...
	}

	log.Printf("开始测试 %d 个设备的配置备份...\n", len(devices))

	// 逐个测试设备
	for i, devCfg := range devices {
		log.Printf("========================================")
		log.Printf("[%d/%d] 测试设备: %s", i+1, len(devices), devCfg.Name)
		log.Printf("IP: %s, 厂商: %s", devCfg.IP, devCfg.Vendor)
		log.Printf("========================================")

		testDevice(devCfg)
		log.Printf("")
	}

	log.Printf("测试完成!")
}

// testDevice 测试单个设备
func testDevice(devCfg DeviceConfig) {
	// 1. 创建设备模型
	networkDevice := &models.NetworkDevice{
		DeviceName: devCfg.Name,
		IPAddress:  devCfg.IP,
		Port:       devCfg.Port,
		Vendor:     models.DeviceVendor(devCfg.Vendor),
	}

	// 2. 创建 wrapper 并连接
	log.Printf("[1/5] 正在连接设备...")

	var wrapper *device.ScrapliWrapper
	var err error

	if devCfg.Port != 22 {
		// 使用自定义端口
		wrapper, err = device.NewScrapliWrapperWithPort(networkDevice, devCfg.Username, devCfg.Password, devCfg.Port, models.ProtocolTypeSSH)
	} else {
		// 使用默认端口 22
		wrapper, err = device.NewScrapliWrapper(networkDevice, devCfg.Username, devCfg.Password, models.ProtocolTypeSSH)
	}

	if err != nil {
		log.Printf("❌ 创建 wrapper 失败: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := wrapper.OpenContext(ctx); err != nil {
		log.Printf("❌ 连接设备失败: %v", err)
		return
	}
	defer wrapper.Close()
	log.Printf("✅ 设备连接成功")

	// 4. 获取配置命令
	command := getConfigCommand(devCfg.Vendor)
	log.Printf("[2/5] 使用命令: %s", command)

	// 5. 执行命令
	log.Printf("[3/5] 正在获取配置...")
	startTime := time.Now()

	response, err := wrapper.SendCommand(command, true)
	elapsed := time.Since(startTime)

	if err != nil {
		log.Printf("❌ 执行命令失败: %v", err)
		return
	}

	// 6. 显示结果
	configSize := len(response.Result)
	log.Printf("✅ 命令执行成功!")
	log.Printf("[4/5] 执行结果:")
	log.Printf("   - 耗时: %v", elapsed)
	log.Printf("   - 配置大小: %d 字节 (%.2f KB)", configSize, float64(configSize)/1024)

	// 7. 显示配置预览
	log.Printf("[5/5] 配置内容预览 (前500字符):")
	log.Printf("   ---")
	preview := response.Result
	if len(preview) > 500 {
		preview = preview[:500] + "...(省略)"
	}
	fmt.Printf("%s\n", preview)
	log.Printf("   ---")

	// 8. 检查配置是否合理
	if configSize < 100 {
		log.Printf("⚠️  警告: 配置内容过小 (%d 字节)，可能命令执行失败", configSize)
	} else {
		log.Printf("✅ 配置备份成功!")
	}
}

// getConfigCommand 根据厂商获取配置命令
func getConfigCommand(vendor string) string {
	switch vendor {
	case "huawei", "h3c":
		return "display current-configuration"
	case "ruijie", "maipu":
		return "show running-config"
	case "cisco":
		return "show running-config"
	default:
		return "show running-config"
	}
}
