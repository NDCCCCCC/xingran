// Package benchmark 提供登录请求加密链路的性能基准测试。
package benchmark

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/tjfoc/gmsm/sm2"

	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
)

// 本文件 5 个 benchmark 已补真实测量(FIX-09 / Phase 139),不再是空桩:
//   - 全部复用项目既有国密与防重放设施(pkg/crypto),零新依赖。
//   - 实测命令: go test -bench=. -benchmem ./tests/benchmark/
//   - 实测结果与 P99<350ms / overhead<50ms 结论见
//     .planning/notes/261108-login-encryption-benchmark.md (本 Phase 产物)。
//   - 边界: encrypted_login / plaintext_login 为「同等加密/序列化计算路径」近似基线,
//     不含完整 HTTP 链路(路由/中间件/DB/网络),不伪造端到端延迟;详见对应注释。

// loginReqBody 模拟一次登录请求的明文 body,作为加/解密与序列化基准负载。
var loginReqBody = []byte(`{"username":"admin","password":"XingRan@2026","captcha":"a1b2c3","uuid":"6f1e0a20-8c2d-4b7e-9f3a-1d2c3b4a5e6f"}`)

// sm4KeyHex 为 16 字节 SM4 密钥的十六进制字符串(32 字符),与前端登录加密
// 用 SM2 公钥加密 SM4 key 的格式一致(参考 pkg/middleware/encryption_80_05_test.go)。
const sm4KeyHex = "0123456789abcdeffedcba9876543210"

// newBenchEncryptor 构造基准用的真实 SM2 密钥对 + SM4 key/IV。
// 密钥生成、cipher 构建等一次性成本由调用方在 b.ResetTimer() 之前完成,不计入测量。
func newBenchEncryptor(b *testing.B) (*sm2.PublicKey, *crypto.RequestEncryptor, []byte, []byte) {
	b.Helper()
	_, pub, err := crypto.GenerateKeyPair()
	if err != nil {
		b.Fatalf("生成 SM2 密钥对失败: %v", err)
	}
	keyBytes, err := hex.DecodeString(sm4KeyHex)
	if err != nil {
		b.Fatalf("解码 SM4 key 失败: %v", err)
	}
	iv := make([]byte, 16)
	if _, err := rand.Read(iv); err != nil {
		b.Fatalf("生成 IV 失败: %v", err)
	}
	// RequestEncryptor 仅用于复用其 SM4-CBC 加密实现(EncryptResponseWithKey),
	// 与服务端登录解密链路是同一套国密算法。
	enc := crypto.NewRequestEncryptor(nil, pub)
	return pub, enc, keyBytes, iv
}

// BenchmarkLoginEncryption 对比登录请求「加密计算路径」与「明文序列化路径」的开销。
//
// 目标: P99 < 350ms,加密相对明文 overhead < 50ms。
//
// ⚠️ 边界声明:这两个子基准是「同等加密/序列化计算路径」的近似基线,不是端到端 HTTP
// 登录延迟。encrypted_login 组合了服务端登录请求解密所需的等价计算:SM2 加密 SM4 key
// + SM4-CBC 加密请求体 + nonce 防重放检查;plaintext_login 仅做 JSON 序列化/解析。
// 均不含路由、Gin 中间件、数据库校验、密码哈希(SM3)、网络往返。因此这两个数值只用于
// 衡量「加解密相对明文增加的计算开销」,不可当作真实 HTTP P99 上报。
func BenchmarkLoginEncryption(b *testing.B) {
	b.Run("encrypted_login", func(b *testing.B) {
		pub, enc, keyBytes, iv := newBenchEncryptor(b)
		nonceStore := crypto.NewDefaultNonceStorage()
		now := time.Now().Unix()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// 1) SM2 公钥加密 SM4 key(与前端登录加密同路径)
			encKey, err := crypto.EncryptWithSM2(sm4KeyHex, pub)
			if err != nil {
				b.Fatalf("SM2 加密 SM4 key 失败: %v", err)
			}
			// 2) SM4-CBC 加密登录请求体(PKCS#7 填充,与服务端解密同算法)
			encResp, err := enc.EncryptResponseWithKey(loginReqBody, keyBytes, iv)
			if err != nil {
				b.Fatalf("SM4-CBC 加密请求体失败: %v", err)
			}
			// 3) nonce 防重放检查(与 RequestDecryption 中间件同一设施)
			nonce := fmt.Sprintf("bench-encrypted-login-%d", i)
			if !nonceStore.CheckAndStore(nonce, now) {
				b.Fatalf("nonce 重复(不应发生): %s", nonce)
			}
			// 4) 组装并序列化完整加密请求体(与前端实际发送的 EncryptedRequest 结构一致)
			encReq := crypto.EncryptedRequest{
				Encrypted: true,
				Data:      encResp.Data,
				SM4Key:    encKey,
				IV:        encResp.IV,
				Timestamp: now,
				Nonce:     nonce,
			}
			if _, err := json.Marshal(encReq); err != nil {
				b.Fatalf("序列化加密请求失败: %v", err)
			}
		}
	})

	b.Run("plaintext_login", func(b *testing.B) {
		// 明文基线:等价的序列化/解析计算路径,不含任何加解密,用于量化 overhead。
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var req map[string]interface{}
			if err := json.Unmarshal(loginReqBody, &req); err != nil {
				b.Fatalf("解析登录请求失败: %v", err)
			}
			if _, err := json.Marshal(req); err != nil {
				b.Fatalf("序列化登录请求失败: %v", err)
			}
		}
	})
}

// BenchmarkNonceStorage 基准防重放 nonce 存储的 CheckAndStore(检查 + 写入)开销。
// 复用项目真实设施 crypto.NewDefaultNonceStorage()(分段锁 shardedNonceStorage,
// 256 分片),与 RequestDecryption 中间件在线使用的一致。每次用唯一 nonce 走
// "新增接受"分支,后台清理 goroutine 不影响单次测量。
func BenchmarkNonceStorage(b *testing.B) {
	b.Run("nonce_check_and_store", func(b *testing.B) {
		store := crypto.NewDefaultNonceStorage()
		now := time.Now().Unix()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			nonce := fmt.Sprintf("bench-nonce-%d", i)
			if !store.CheckAndStore(nonce, now) {
				b.Fatalf("nonce 重复(不应发生): %s", nonce)
			}
		}
	})
}

// BenchmarkSM2Encryption 基准 SM2 公钥加密 SM4 key(登录混合加密的密钥封装步)。
// 复用 crypto.EncryptWithSM2 + crypto.GenerateKeyPair,与前端登录加密同算法;
// 密钥对生成在计时外完成。
func BenchmarkSM2Encryption(b *testing.B) {
	b.Run("sm2_encrypt_sm4_key", func(b *testing.B) {
		pub, _, _, _ := newBenchEncryptor(b)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := crypto.EncryptWithSM2(sm4KeyHex, pub); err != nil {
				b.Fatalf("SM2 加密 SM4 key 失败: %v", err)
			}
		}
	})
}

// BenchmarkSM4Encryption 基准 SM4-CBC 加密登录请求体。
// 复用 crypto.RequestEncryptor.EncryptResponseWithKey(内部 sm4.NewCipher +
// CBCEncrypter + PKCS#7 填充),与服务端 DecryptRequest 使用的 SM4-CBC 为同一
// 实现(tjfoc/gmsm/sm4),即登录请求体加密的真实算法路径。
func BenchmarkSM4Encryption(b *testing.B) {
	b.Run("sm4_cbc_encrypt_request", func(b *testing.B) {
		_, enc, keyBytes, iv := newBenchEncryptor(b)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := enc.EncryptResponseWithKey(loginReqBody, keyBytes, iv); err != nil {
				b.Fatalf("SM4-CBC 加密请求体失败: %v", err)
			}
		}
	})
}
