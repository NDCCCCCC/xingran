// 一次性工具: 加载 .env 并打印关键配置(不连接 DB/Redis)
// 用法: go run scripts/check-env.go
// 不会进入正常构建(//go:build ignore 标签)
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/xingran-next/xingran-go-backend/internal/config"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("❌ .env 加载失败: %v\n", err)
		return
	}
	fmt.Println("✅ .env 加载成功")

	cfg, err := config.Load(context.Background())
	if err != nil {
		fmt.Printf("❌ 配置加载失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("=== 配置加载结果 ===")
	fmt.Printf("数据库:     %s:%d user=%s db=%s ssl=%s\n",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.DBName, cfg.Database.SSLMode)
	fmt.Printf("数据库密码: %s (长度=%d)\n", mask(cfg.Database.Password), len(cfg.Database.Password))
	if cfg.Database.Password == "" {
		fmt.Println("  ❌ 数据库密码为空,启动会失败")
	}
	fmt.Printf("Redis:      %s:%d password=%s (长度=%d)\n",
		cfg.Cache.Host, cfg.Cache.Port, mask(cfg.Cache.Password), len(cfg.Cache.Password))
	fmt.Printf("JWT secret: %s (长度=%d)\n", mask(cfg.JWT.SecretKey), len(cfg.JWT.SecretKey))
	if cfg.JWT.SecretKey == "" {
		fmt.Println("  ❌ JWT secret 为空,启动会失败")
	}
	fmt.Printf("SM2 私钥:   %s (长度=%d)\n", mask(cfg.JWT.SM2PrivateKey), len(cfg.JWT.SM2PrivateKey))
	if cfg.JWT.SM2PrivateKey == "" {
		fmt.Println("  ⚠️ SM2 私钥为空,会走\"动态生成\"分支(每次重启密钥对都换,旧 token 失效)")
	}
	fmt.Printf("SM2 公钥:   %s (长度=%d)\n", mask(cfg.JWT.SM2PublicKey), len(cfg.JWT.SM2PublicKey))
	fmt.Printf("SM4 key:    %s (长度=%d)\n", mask(cfg.Security.SM4Key), len(cfg.Security.SM4Key))
	if cfg.Security.SM4Key == "" {
		fmt.Println("  ❌ SM4 key 为空,启动会中止")
	}
	if cfg.Security.SM4Key == "dGVzdC1zZWNyZXQxNiEhIQ==" {
		fmt.Println("  ⚠️ SM4 key 是仓库默认值 — 启动会打 [SECURITY WARNING] 警告")
	}
	fmt.Printf("Baidu AK:   %s (长度=%d)\n", mask(cfg.Baidu.MapAK), len(cfg.Baidu.MapAK))
	if cfg.Baidu.MapAK == "" {
		fmt.Println("  ⚠️ Baidu AK 为空,前端地理编码会失败(后端启动不受影响)")
	}
	fmt.Printf("服务器:     %s:%d mode=%s\n", cfg.Server.Host, cfg.Server.Port, cfg.Server.Mode)
}

func mask(s string) string {
	if len(s) == 0 {
		return "<empty>"
	}
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "***" + s[len(s)-4:]
}
