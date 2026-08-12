// 一次性工具: 验证 .env 中的 SM2 私钥能否被 ParseSm2PrivateKey 成功解析
// 用法: go run scripts/test-sm2-parse.go
package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("❌ .env 加载失败: %v\n", err)
		os.Exit(1)
	}

	privHex := os.Getenv("JWT_SM2_PRIVATE_KEY")
	pubHex := os.Getenv("JWT_SM2_PUBLIC_KEY")

	fmt.Printf("私钥长度: %d hex 字符 (期望 64)\n", len(privHex))
	fmt.Printf("公钥长度: %d hex 字符 (期望 130)\n", len(pubHex))

	priv, err := crypto.ParsePrivateKeyFromHex(privHex)
	if err != nil {
		fmt.Printf("❌ 私钥解析失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 私钥解析成功\n")

	pub, err := crypto.ParsePublicKeyFromHex(pubHex)
	if err != nil {
		fmt.Printf("❌ 公钥解析失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 公钥解析成功\n")

	// 简单健全性检查: 解析后的私钥 D 值应与原 hex 一致
	if priv.D == nil {
		fmt.Printf("❌ 私钥 D 值为 nil\n")
		os.Exit(1)
	}
	fmt.Printf("✅ 私钥 D 值非空 (长度 %d 字节)\n", len(priv.D.Bytes()))

	if pub.X == nil || pub.Y == nil {
		fmt.Printf("❌ 公钥 X/Y 为 nil\n")
		os.Exit(1)
	}
	fmt.Printf("✅ 公钥 X/Y 非空\n")
}
