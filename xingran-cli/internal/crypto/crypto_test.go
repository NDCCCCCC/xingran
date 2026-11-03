package crypto

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tjfoc/gmsm/sm2"

	pkgcrypto "github.com/xingran-next/xingran-go-backend/pkg/crypto"
)

// newTestKeyPair 生成测试 SM2 密钥对，返回私钥与导出的 hex 公钥
func newTestKeyPair(t *testing.T) (*sm2.PrivateKey, string) {
	t.Helper()
	priv, err := sm2.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("生成 SM2 密钥对失败: %v", err)
	}
	return priv, pkgcrypto.ExportPublicKeyToHex(&priv.PublicKey)
}

// newServerWithPubKey 启动返回公钥的 httptest 后端模拟
func newServerWithPubKey(t *testing.T, priv *sm2.PrivateKey) *httptest.Server {
	t.Helper()
	pubHex := pkgcrypto.ExportPublicKeyToHex(&priv.PublicKey)
	mux := http.NewServeMux()
	mux.HandleFunc(PublicKeyPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"code":0,"message":"success","data":{"publicKey":%q}}`, pubHex)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// TestParsePublicKeyHex 校验公钥格式校验逻辑
func TestParsePublicKeyHex(t *testing.T) {
	priv, pubHex := newTestKeyPair(t)

	pub, err := parsePublicKeyHex(pubHex)
	if err != nil {
		t.Fatalf("合法公钥解析失败: %v", err)
	}
	if pub.X.Cmp(priv.X) != 0 || pub.Y.Cmp(priv.Y) != 0 {
		t.Fatal("解析后的公钥坐标与原始密钥不一致")
	}

	if _, err := parsePublicKeyHex("0102"); err == nil {
		t.Fatal("长度非法的公钥应报错")
	}
	if _, err := parsePublicKeyHex(strings.Repeat("ab", 65)); err == nil {
		t.Fatal("非 04 开头的公钥应报错")
	}
	if _, err := parsePublicKeyHex("zz" + pubHex[2:]); err == nil {
		t.Fatal("非 hex 字符的公钥应报错")
	}
}

// TestFetchPublicKey 拉取公钥：成功 / HTTP 500 / code!=0 / 公钥格式非法
func TestFetchPublicKey(t *testing.T) {
	ctx := context.Background()
	priv, _ := newTestKeyPair(t)
	srv := newServerWithPubKey(t, priv)
	pubHex := pkgcrypto.ExportPublicKeyToHex(&priv.PublicKey)

	e := NewEncryptor().(*encryptor)
	if err := e.FetchPublicKey(ctx, srv.URL+"/"); err != nil {
		t.Fatalf("FetchPublicKey 失败: %v", err)
	}
	if e.publicKeyHex != pubHex {
		t.Fatalf("公钥缓存不匹配: got %s want %s", e.publicKeyHex, pubHex)
	}

	// HTTP 500
	srv500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv500.Close()
	if err := NewEncryptor().(*encryptor).FetchPublicKey(ctx, srv500.URL); err == nil {
		t.Fatal("HTTP 500 应报错")
	}

	// code != 0
	srvErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"code":500,"message":"SM2 未启用或公钥不可用"}`)
	}))
	defer srvErr.Close()
	if err := NewEncryptor().(*encryptor).FetchPublicKey(ctx, srvErr.URL); err == nil || !strings.Contains(err.Error(), "SM2 未启用") {
		t.Fatalf("code!=0 应报错并带 message, got: %v", err)
	}

	// 公钥格式非法
	srvBad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"code":0,"message":"success","data":{"publicKey":"1234"}}`)
	}))
	defer srvBad.Close()
	if err := NewEncryptor().(*encryptor).FetchPublicKey(ctx, srvBad.URL); err == nil {
		t.Fatal("公钥格式非法应报错")
	}
}

// TestEncryptRequestRoundTrip 全回路：客户端加密 → pkg/crypto 服务端 DecryptRequest 还原
func TestEncryptRequestRoundTrip(t *testing.T) {
	ctx := context.Background()
	priv, _ := newTestKeyPair(t)
	srv := newServerWithPubKey(t, priv)

	fixedNow := time.Now()
	e := newEncryptorForTest(func() time.Time { return fixedNow }, rand.Reader)
	if err := e.FetchPublicKey(ctx, srv.URL); err != nil {
		t.Fatalf("FetchPublicKey 失败: %v", err)
	}

	cases := [][]byte{
		[]byte(`{"username":"admin","password":"P@ssw0rd"}`),
		[]byte(`{"list":[],"total":0}`), // 短 body（触发一个完整填充块）
		[]byte(strings.Repeat("中文国密测试数据-", 100)), // 多块 + UTF-8
	}
	for i, body := range cases {
		payload, err := e.EncryptRequest(ctx, body)
		if err != nil {
			t.Fatalf("case %d EncryptRequest 失败: %v", i, err)
		}

		var encReq pkgcrypto.EncryptedRequest
		if err := json.Unmarshal(payload, &encReq); err != nil {
			t.Fatalf("case %d payload 解析失败: %v", i, err)
		}
		if !encReq.Encrypted {
			t.Fatalf("case %d: encrypted 标志应为 true", i)
		}
		if encReq.Timestamp != fixedNow.Unix() {
			t.Fatalf("case %d: timestamp 应为注入时钟值, got %d", i, encReq.Timestamp)
		}
		if len(encReq.Nonce) != 32 {
			t.Fatalf("case %d: nonce 应为 32 hex 字符, got %s", i, encReq.Nonce)
		}

		// 服务端解密（±60s 容差 + nonce 防重放校验与后端一致）
		re := pkgcrypto.NewRequestEncryptor(priv, nil)
		plaintext, err := re.DecryptRequest(&encReq)
		if err != nil {
			t.Fatalf("case %d: 服务端解密失败: %v", i, err)
		}
		if string(plaintext) != string(body) {
			t.Fatalf("case %d: 解密结果不匹配\ngot:  %q\nwant: %q", i, plaintext, body)
		}
	}
}

// TestEncryptRequestNoPublicKey 未拉取公钥时加密应报错
func TestEncryptRequestNoPublicKey(t *testing.T) {
	e := NewEncryptor()
	_, err := e.EncryptRequest(context.Background(), []byte(`{}`))
	if err == nil || !strings.Contains(err.Error(), "公钥未加载") {
		t.Fatalf("应提示公钥未加载, got: %v", err)
	}
}

// TestEncryptRequestDecryptRejectsReplay nonce 复用时服务端必须拒绝（防重放对齐验证）
func TestEncryptRequestDecryptRejectsReplay(t *testing.T) {
	ctx := context.Background()
	priv, _ := newTestKeyPair(t)
	srv := newServerWithPubKey(t, priv)

	e := newEncryptorForTest(time.Now, rand.Reader)
	if err := e.FetchPublicKey(ctx, srv.URL); err != nil {
		t.Fatalf("FetchPublicKey 失败: %v", err)
	}

	payload, err := e.EncryptRequest(ctx, []byte(`{"k":1}`))
	if err != nil {
		t.Fatalf("EncryptRequest 失败: %v", err)
	}
	var encReq pkgcrypto.EncryptedRequest
	if err := json.Unmarshal(payload, &encReq); err != nil {
		t.Fatal(err)
	}

	// 同一 nonce 重放：首次校验通过后，第二次必须被拒绝
	re := pkgcrypto.NewRequestEncryptor(priv, nil)
	if _, err := re.DecryptRequest(&encReq); err != nil {
		t.Fatalf("首次解密不应失败: %v", err)
	}
	if _, err := re.DecryptRequest(&encReq); err == nil || !strings.Contains(err.Error(), "重复") {
		t.Fatalf("重放同一 nonce 应被拒绝, got: %v", err)
	}
}

// TestTimestampOutsideWindow 超出 ±60s 容差的时间戳服务端必须拒绝
func TestTimestampOutsideWindow(t *testing.T) {
	ctx := context.Background()
	priv, _ := newTestKeyPair(t)
	srv := newServerWithPubKey(t, priv)

	stale := time.Now().Add(-61 * time.Second)
	e := newEncryptorForTest(func() time.Time { return stale }, rand.Reader)
	if err := e.FetchPublicKey(ctx, srv.URL); err != nil {
		t.Fatal(err)
	}
	payload, err := e.EncryptRequest(ctx, []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	var encReq pkgcrypto.EncryptedRequest
	if err := json.Unmarshal(payload, &encReq); err != nil {
		t.Fatal(err)
	}
	re := pkgcrypto.NewRequestEncryptor(priv, nil)
	if _, err := re.DecryptRequest(&encReq); err == nil {
		t.Fatal("超出 ±60s 容差的时间戳应被服务端拒绝")
	} else {
		t.Logf("预期拒绝: %v", err)
	}
}
