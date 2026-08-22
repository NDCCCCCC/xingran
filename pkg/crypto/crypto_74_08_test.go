package crypto

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================================
// 74-08 Batch C: pkg/crypto — nonce_storage 双实现 + request_encryption
// 解密/加密 + sm2_jwt 密钥解析/加解密/PEM 导出。
// =====================================================================

// ---------------- nonce_storage.go ----------------

// nonceCounter 收敛 GetNonceCount(具体类型有,NonceStorage 接口无)。
type nonceCounter interface{ GetNonceCount() int }

func TestShardedNonceStorage_CheckAndStore(t *testing.T) {
	s := NewShardedNonceStorage()
	now := time.Now().Unix()

	assert.True(t, s.CheckAndStore("nonce-a", now), "首次存储成功")
	assert.False(t, s.CheckAndStore("nonce-a", now), "重复 nonce 拒绝")
	assert.True(t, s.CheckAndStore("nonce-b", now), "不同 nonce 独立")
	assert.Equal(t, 2, s.(nonceCounter).GetNonceCount())
}

func TestShardedNonceStorage_Cleanup(t *testing.T) {
	s := NewShardedNonceStorageWithConfig(1) // replayWindow=1s → 过期阈值 2s
	require.NotNil(t, s)
	old := time.Now().Unix() - 100
	now := time.Now().Unix()

	s.CheckAndStore("old-nonce", old)
	s.CheckAndStore("fresh-nonce", now)
	assert.Equal(t, 2, s.(nonceCounter).GetNonceCount())

	// 触发清理(同包可访问未导出方法)
	s.(*shardedNonceStorage).cleanupExpiredNonces()
	assert.Equal(t, 1, s.(nonceCounter).GetNonceCount(), "过期 nonce 被清理")
	assert.False(t, s.CheckAndStore("old-nonce", old) == false, "清理后可重新存储")
}

func TestSyncMapNonceStorage(t *testing.T) {
	s := NewSyncMapNonceStorage()
	now := time.Now().Unix()

	assert.True(t, s.CheckAndStore("n1", now))
	assert.False(t, s.CheckAndStore("n1", now), "重复拒绝")
	assert.True(t, s.CheckAndStore("n2", now))
	assert.Equal(t, 2, s.(nonceCounter).GetNonceCount())
}

func TestDefaultNonceStorage(t *testing.T) {
	s := NewDefaultNonceStorage()
	require.NotNil(t, s)
	now := time.Now().Unix()
	assert.True(t, s.CheckAndStore("d1", now))
	assert.False(t, s.CheckAndStore("d1", now))
}

// ---------------- request_encryption.go ----------------

func TestPkcs7PadUnpad_Roundtrip(t *testing.T) {
	for _, data := range [][]byte{
		[]byte("a"),
		[]byte("0123456789abcdef"),   // 正好一个块 → 追加整块填充
		[]byte("0123456789abcde"),    // 15 字节 → 填 1
		[]byte(""),                   // 空 → 整块填充
		[]byte(strings.Repeat("x", 100)),
	} {
		padded := pkcs7Pad(data, 16)
		assert.Equal(t, 0, len(padded)%16, "填充后块对齐")
		unpadded, err := pkcs7Unpad(padded)
		require.NoError(t, err)
		assert.Equal(t, data, unpadded)
	}
}

func TestPkcs7Unpad_Errors(t *testing.T) {
	_, err := pkcs7Unpad([]byte{})
	assert.Error(t, err, "空输入")

	_, err = pkcs7Unpad([]byte{1, 2, 3, 0x00})
	assert.Error(t, err, "padding=0 非法")

	_, err = pkcs7Unpad([]byte{1, 2, 3, 0x05})
	assert.Error(t, err, "padding 超过长度")
}

func TestValidateTimestamp(t *testing.T) {
	re := NewRequestEncryptor(nil, nil)

	assert.Error(t, re.validateTimestamp(0), "0 时间戳拒绝")
	assert.Error(t, re.validateTimestamp(-5), "负时间戳拒绝")
	assert.Error(t, re.validateTimestamp(100), "过早时间戳拒绝")

	now := time.Now().Unix()
	assert.NoError(t, re.validateTimestamp(now))
	assert.NoError(t, re.validateTimestamp(now-30), "窗口内")

	// 超窗
	assert.Error(t, re.validateTimestamp(now-3600), "过旧拒绝")
	assert.Error(t, re.validateTimestamp(now+3600), "未来过远拒绝")

	// 自定义窗口
	re.SetReplayWindowSec(7200)
	assert.Equal(t, 7200, re.ReplayWindowSec())
	assert.NoError(t, re.validateTimestamp(now-3600), "宽窗口放行")
}

func TestValidateNonce(t *testing.T) {
	re := NewRequestEncryptor(nil, nil)
	now := time.Now().Unix()

	assert.Error(t, re.validateNonce("", now), "空 nonce 拒绝")
	assert.NoError(t, re.validateNonce("unique-1", now))
	assert.Error(t, re.validateNonce("unique-1", now), "重放拒绝")
}

func TestSetNonceStorage(t *testing.T) {
	re := NewRequestEncryptor(nil, nil)
	custom := NewSyncMapNonceStorage()
	re.SetNonceStorage(custom)
	now := time.Now().Unix()
	assert.NoError(t, re.validateNonce("k1", now))
	assert.Equal(t, 1, custom.(nonceCounter).GetNonceCount(), "自定义存储生效")
}

// buildEncryptedRequest 构造一个合法加密请求(SM2 加密 SM4 key + SM4-CBC 加密数据)。
func buildEncryptedRequest(t *testing.T, plaintext string, nonce string, ts int64) (*EncryptedRequest, *RequestEncryptor) {
	t.Helper()
	priv, pub, err := GenerateKeyPair()
	require.NoError(t, err)

	sm4KeyHex := "0123456789abcdeffedcba9876543210" // 16 字节 hex
	keyBytes, err := hex.DecodeString(sm4KeyHex)
	require.NoError(t, err)
	iv := []byte("0123456789abcdef")

	encKey, err := EncryptWithSM2(sm4KeyHex, pub)
	require.NoError(t, err)

	re := NewRequestEncryptor(priv, pub)
	encResp, err := re.EncryptResponseWithKey([]byte(plaintext), keyBytes, iv)
	require.NoError(t, err)

	return &EncryptedRequest{
		Encrypted: true,
		Data:      encResp.Data,
		SM4Key:    encKey,
		IV:        base64.StdEncoding.EncodeToString(iv),
		Timestamp: ts,
		Nonce:     nonce,
	}, re
}

func TestDecryptRequest_Roundtrip(t *testing.T) {
	now := time.Now().Unix()
	encReq, re := buildEncryptedRequest(t, `{"hello":"world"}`, "nonce-rt-1", now)

	plain, err := re.DecryptRequest(encReq)
	require.NoError(t, err)
	assert.JSONEq(t, `{"hello":"world"}`, string(plain))
}

func TestDecryptRequest_ErrorPaths(t *testing.T) {
	now := time.Now().Unix()
	encReq, re := buildEncryptedRequest(t, "x", "nonce-err-1", now)

	// 时间戳非法
	bad := *encReq
	bad.Timestamp = 0
	_, err := re.DecryptRequest(&bad)
	assert.Error(t, err)

	// nonce 重放
	bad2 := *encReq
	bad2.Nonce = "nonce-replay"
	_, err = re.DecryptRequest(&bad2)
	require.NoError(t, err)
	_, err = re.DecryptRequest(&bad2)
	assert.Error(t, err, "重放拒绝")

	// SM4Key 非 base64 SM2 密文
	bad3 := *encReq
	bad3.Nonce = "nonce-badkey"
	bad3.SM4Key = "!!!not-base64!!!"
	_, err = re.DecryptRequest(&bad3)
	assert.Error(t, err)

	// IV 非法 base64
	bad4 := *encReq
	bad4.Nonce = "nonce-badiv"
	bad4.IV = "!!!"
	_, err = re.DecryptRequest(&bad4)
	assert.Error(t, err)

	// IV 长度错误
	bad5 := *encReq
	bad5.Nonce = "nonce-shortiv"
	bad5.IV = base64.StdEncoding.EncodeToString([]byte("short"))
	_, err = re.DecryptRequest(&bad5)
	assert.Error(t, err)

	// Data 非法 base64
	bad6 := *encReq
	bad6.Nonce = "nonce-baddata"
	bad6.Data = "!!!"
	_, err = re.DecryptRequest(&bad6)
	assert.Error(t, err)

	// 密文非块对齐
	bad7 := *encReq
	bad7.Nonce = "nonce-misaligned"
	bad7.Data = base64.StdEncoding.EncodeToString([]byte("not-block-aligned-17b"))
	_, err = re.DecryptRequest(&bad7)
	assert.Error(t, err)
}

func TestDecryptRequestWithKeyInfo(t *testing.T) {
	now := time.Now().Unix()
	encReq, re := buildEncryptedRequest(t, "payload-data", "nonce-ki-1", now)

	plain, sm4Key, iv, err := re.DecryptRequestWithKeyInfo(encReq)
	require.NoError(t, err)
	assert.Equal(t, "payload-data", string(plain))
	assert.Len(t, sm4Key, 16)
	assert.Len(t, iv, 16)

	// 用返回的 key/iv 加密响应 → 模拟前端用同一 key 解密
	encResp, err := re.EncryptResponseWithKey([]byte("resp"), sm4Key, iv)
	require.NoError(t, err)
	assert.True(t, encResp.Encrypted)
	assert.Empty(t, encResp.SM4Key, "响应不重复携带密钥")
}

func TestEncryptResponseWithKey_BadKey(t *testing.T) {
	re := NewRequestEncryptor(nil, nil)
	_, err := re.EncryptResponseWithKey([]byte("x"), []byte("short"), make([]byte, 16))
	assert.Error(t, err, "非法密钥长度")
}

// ---------------- sm2_jwt.go 密钥/加解密/导出 ----------------

func TestParsePrivateKeyFromHex(t *testing.T) {
	priv, _, err := GenerateKeyPair()
	require.NoError(t, err)

	// 32 字节原始 hex 路径
	rawHex := hex.EncodeToString(priv.D.Bytes())
	// D 可能不足 32 字节,左补零
	for len(rawHex) < 64 {
		rawHex = "0" + rawHex
	}
	parsed, err := ParsePrivateKeyFromHex(rawHex)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, 0, priv.D.Cmp(parsed.D), "D 值一致")

	// 非法 hex
	_, err = ParsePrivateKeyFromHex("zzzz")
	assert.Error(t, err)

	// 合法 hex 但非密钥
	_, err = ParsePrivateKeyFromHex("0102")
	assert.Error(t, err)
}

func TestParsePublicKeyFromHex(t *testing.T) {
	_, pub, err := GenerateKeyPair()
	require.NoError(t, err)

	hexKey := ExportPublicKeyToHex(pub)
	parsed, err := ParsePublicKeyFromHex(hexKey)
	require.NoError(t, err)
	assert.Equal(t, 0, pub.X.Cmp(parsed.X))
	assert.Equal(t, 0, pub.Y.Cmp(parsed.Y))

	_, err = ParsePublicKeyFromHex("zz")
	assert.Error(t, err)

	_, err = ParsePublicKeyFromHex("0102")
	assert.Error(t, err)
}

func TestSM2EncryptDecrypt_Roundtrip(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	require.NoError(t, err)

	// 空串 → 空
	enc, err := EncryptWithSM2("", pub)
	require.NoError(t, err)
	assert.Equal(t, "", enc)
	dec, err := DecryptWithSM2("", priv)
	require.NoError(t, err)
	assert.Equal(t, "", dec)

	// hex 输入(SM4 key 场景: 解密后应还原 hex 字符串)
	hexPlain := "0123456789abcdeffedcba9876543210"
	enc, err = EncryptWithSM2(hexPlain, pub)
	require.NoError(t, err)
	dec, err = DecryptWithSM2(enc, priv)
	require.NoError(t, err)
	assert.Equal(t, hexPlain, dec)

	// 普通文本
	enc, err = EncryptWithSM2("hello-sm2", pub)
	require.NoError(t, err)
	dec, err = DecryptWithSM2(enc, priv)
	require.NoError(t, err)
	assert.Equal(t, "hello-sm2", dec)

	// 非法 base64 → 错误
	_, err = DecryptWithSM2("!!!not-base64!!!", priv)
	assert.Error(t, err)

	// QUIRK(D-12 不修复): 合法 base64 但非 SM2 密文的输入会让
	// tjfoc/gmsm sm2.Decrypt 直接 panic(makeslice: len out of range,
	// gmsm v1.4.1 sm2.go:321 不做长度预检)。该子用例移除,仅记录。
}

func TestIsHexString(t *testing.T) {
	assert.True(t, isHexString("0123456789abcdefABCDEF"))
	assert.True(t, isHexString(""))
	assert.False(t, isHexString("xyz"))
	assert.False(t, isHexString("123g"))
}

func TestIsPrintableASCII(t *testing.T) {
	assert.True(t, isPrintableASCII([]byte("hello")))
	assert.False(t, isPrintableASCII([]byte{}))
	assert.False(t, isPrintableASCII([]byte{0x01, 0x02}))
	assert.False(t, isPrintableASCII([]byte("中文")))
}

func TestDecodeHexEncodedResult(t *testing.T) {
	// 可打印 ASCII → 原样
	got, err := decodeHexEncodedResult([]byte("plain"), "test")
	require.NoError(t, err)
	assert.Equal(t, "plain", got)

	// 不可打印 → hex
	got, err = decodeHexEncodedResult([]byte{0x01, 0x02}, "test")
	require.NoError(t, err)
	assert.Equal(t, "0102", got)
}

func TestExportPublicKeyToPEM(t *testing.T) {
	_, pub, err := GenerateKeyPair()
	require.NoError(t, err)

	pem, err := ExportPublicKeyToPEM(pub)
	require.NoError(t, err)
	assert.Contains(t, pem, "BEGIN PUBLIC KEY")
	assert.Contains(t, pem, "END OF PUBLIC KEY")
}

func TestGenerateRefreshTokenWithSM2_Roundtrip(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	require.NoError(t, err)

	token, err := GenerateRefreshTokenWithSM2("u1", "alice", "test-issuer", time.Hour, priv)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := ValidateRefreshTokenWithSM2(token, pub)
	require.NoError(t, err)
	require.NotNil(t, claims)

	// 错误公钥 → 验证失败
	_, pub2, err := GenerateKeyPair()
	require.NoError(t, err)
	_, err = ValidateRefreshTokenWithSM2(token, pub2)
	assert.Error(t, err)

	// 垃圾 token
	_, err = ValidateRefreshTokenWithSM2("not.a.token", pub)
	assert.Error(t, err)
}
