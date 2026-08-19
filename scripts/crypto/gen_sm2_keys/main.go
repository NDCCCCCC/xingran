// 一次性工具: 生成 SM2 密钥对并输出 hex 字符串
// 用法: go run scripts/crypto/gen_sm2_keys/main.go
// 不会进入正常构建（//go:build ignore 标签）
package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
)

func main() {
	priv, pub, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(err)
	}

	// 私钥:用 32 字节 raw D 格式(与 ParsePrivateKeyFromHex 的 fallback 分支兼容)
	// x509.MarshalSm2PrivateKey 输出的 DER 格式与 ParseSm2PrivateKey 期望的 ASN.1 结构不匹配
	// (asn1: structure error: tags don't match),改用 raw D format 避免此问题
	privBytes := priv.D.Bytes() // big-endian,可能 < 32 字节(前导零被截断)
	privHex := hex.EncodeToString(privBytes)
	// 补齐到 64 字符(32 字节)
	for len(privHex) < 64 {
		privHex = "0" + privHex
	}

	// 公钥:用项目提供的 ExportPublicKeyToHex(04 + X + Y 格式,65 字节)
	pubHex := crypto.ExportPublicKeyToHex(pub)

	fmt.Printf("JWT_SM2_PRIVATE_KEY=%s\n", privHex)
	fmt.Printf("JWT_SM2_PUBLIC_KEY=%s\n", pubHex)
	fmt.Fprintf(os.Stderr, "private key: %d hex chars (raw D, 32 bytes)\n", len(privHex))
	fmt.Fprintf(os.Stderr, "public key:  %d hex chars (04+X+Y)\n", len(pubHex))
}
