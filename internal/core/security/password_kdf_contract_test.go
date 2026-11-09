// Phase 146 · CRYPTO KDF 契约回归测试（package security 同包，可访问未导出 pbkdf2SM3）
//
// 主防线说明：Hash↔Verify 自洽无法发现实现漂移（两边一起错仍全绿），
// 真正的漂移防护是 A1 的「独立于实现的黄金向量 hex」——期望值写死自改动前实现，
// 严禁在测试里「用当前实现现算当期望值」，更严禁向量不一致时更新期望值迁就实现。
package security

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

// golden 固定输入（planner 用改动前实现实跑采集，见 146-PLAN.md §3.4）
var (
	goldenPassword = []byte("GSD-Phase146-GoldenVector")
	goldenSalt     = [16]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}
)

// A1. TestPbkdf2SM3_GoldenVector_KeyLen32：重构前后逐字节锁。
// 4 行 hex 来自改动前实现（1 / 2 / 1000 / 600000 迭代），含默认 600k 与长循环路径。
func TestPbkdf2SM3_GoldenVector_KeyLen32(t *testing.T) {
	pm := NewPasswordManager(nil)

	cases := []struct {
		iterations int
		wantHex    string
	}{
		{1, "a044b33929ad0b9895724d0e0539ea6a1dd738034595316c4aab98037d0cbb8e"},
		{2, "9c96ab821f4556776dc8736a466a672b930f98155f3f0f98048f289d67f966b9"},
		{1000, "790998c08c743cf14f3a016b2d8e755c985548f581d01d28b70a916b582be71d"},
		{600000, "aa35b8efd5fcb94f0297ce15d31f9b89f52ef17cfbebe739aec9f4c0eb9a81aa"},
	}

	for _, tc := range cases {
		dk, err := pm.pbkdf2SM3(goldenPassword, goldenSalt[:], tc.iterations, 32)
		if err != nil {
			t.Fatalf("iter=%d: keyLen=32 意外返回 error: %v", tc.iterations, err)
		}
		if got := hex.EncodeToString(dk); got != tc.wantHex {
			t.Fatalf("iter=%d: keyLen=32 输出漂移 got=%s want=%s —— 禁止更新期望值，须回滚实现",
				tc.iterations, got, tc.wantHex)
		}
	}
}

// A2. TestPbkdf2SM3_RejectsIllegalInput：非法输入矩阵，断言必返 error 且绝不返回非 nil 弱结果。
// 关键回归：旧实现在 keyLen=64 时静默返回 32 真值 + 32 个 0 字节（result[32:] 全零），
// 本测试用「err!=nil && len(dk)==0」证明该失败模式已消灭。
func TestPbkdf2SM3_RejectsIllegalInput(t *testing.T) {
	pm := NewPasswordManager(nil)

	cases := []struct {
		name       string
		iterations int
		keyLen     int
	}{
		{"keyLen=0", 1, 0},
		{"keyLen=-1", 1, -1},
		{"keyLen=33", 1, 33},
		{"keyLen=64", 1, 64},
		{"keyLen=65", 2, 65},
		{"iterations=0", 0, 32},
		{"iterations=-1", -1, 32},
	}

	for _, tc := range cases {
		dk, err := pm.pbkdf2SM3(goldenPassword, goldenSalt[:], tc.iterations, tc.keyLen)
		if err == nil {
			t.Errorf("%s: 期望 error，实际 err=nil（旧实现此处静默返回「32 真值 + 后半段全零」弱密钥）", tc.name)
		}
		if len(dk) != 0 {
			t.Errorf("%s: 期望 nil/空结果，实际 len(dk)=%d（不得再出现 result[32:] 全零的非 nil 切片）", tc.name, len(dk))
		}
	}

	// 合法对照组：keyLen=32 + iterations=1 必须正常产出 32 字节，且 err=nil
	dk, err := pm.pbkdf2SM3(goldenPassword, goldenSalt[:], 1, 32)
	if err != nil || len(dk) != 32 {
		t.Fatalf("合法对照组失败: err=%v len(dk)=%d, want err=nil len=32", err, len(dk))
	}
}

// A3. TestPasswordManager_HashVerifyRoundTrip：往返 + 锁 $sm3$ 恒 32 字节格式契约。
// 注意：Hash↔Verify 自洽无法发现实现漂移，真正的漂移防护是 A1 黄金向量；本测试不是唯一防线。
func TestPasswordManager_HashVerifyRoundTrip(t *testing.T) {
	managers := []struct {
		name   string
		config *PasswordConfig
	}{
		{"default-600k", nil},
		{"small-1000", &PasswordConfig{Iterations: 1000, SaltLength: 16}},
	}

	for _, mc := range managers {
		pm := NewPasswordManager(mc.config)
		const password = "RoundTrip-P@ss-146"

		hash, err := pm.HashPassword(password)
		if err != nil {
			t.Fatalf("[%s] HashPassword failed: %v", mc.name, err)
		}

		// $sm3$ 5 段：["", "sm3", iterations, salt, hash]
		parts := strings.Split(hash, "$")
		if len(parts) != 5 {
			t.Fatalf("[%s] $sm3$ 格式段数非法: got %d want 5, hash=%q", mc.name, len(parts), hash)
		}
		if parts[1] != "sm3" {
			t.Fatalf("[%s] parts[1]=%q want sm3", mc.name, parts[1])
		}
		if wantIter := fmt.Sprintf("%d", pm.config.Iterations); parts[2] != wantIter {
			t.Fatalf("[%s] parts[2]=%q want %q", mc.name, parts[2], wantIter)
		}
		salt, err := base64.RawStdEncoding.DecodeString(parts[3])
		if err != nil || len(salt) != pm.config.SaltLength {
			t.Fatalf("[%s] salt 解码/长度非法: err=%v len=%d want %d", mc.name, err, len(salt), pm.config.SaltLength)
		}
		hashBytes, err := base64.RawStdEncoding.DecodeString(parts[4])
		if err != nil || len(hashBytes) != 32 {
			t.Fatalf("[%s] hash 解码/长度非法: err=%v len=%d want 32（$sm3$ 恒 32 字节）", mc.name, err, len(hashBytes))
		}

		ok, err := pm.VerifyPassword(password, hash)
		if err != nil || !ok {
			t.Fatalf("[%s] 正确密码应验证通过: ok=%v err=%v", mc.name, ok, err)
		}
		ok, err = pm.VerifyPassword("wrong-password", hash)
		if err != nil || ok {
			t.Fatalf("[%s] 错误密码应被拒绝: ok=%v err=%v", mc.name, ok, err)
		}
	}
}

// A4. TestPasswordManager_Legacy100k_BackwardCompat：100k 旧哈希零迁移（补强 owasp 既有同名测试）。
func TestPasswordManager_Legacy100k_BackwardCompat(t *testing.T) {
	pm := NewPasswordManager(nil)
	const legacyIterations = 100000
	const plaintext = "legacy-100k-verify"

	salt := []byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f}
	hashBytes, err := pm.pbkdf2SM3([]byte(plaintext), salt, legacyIterations, 32)
	if err != nil {
		t.Fatalf("pbkdf2SM3(100k,32) failed: %v", err)
	}
	if len(hashBytes) != 32 {
		t.Fatalf("构造 100k 旧哈希长度非法: got %d want 32", len(hashBytes))
	}

	legacyHash := fmt.Sprintf("$sm3$%d$%s$%s",
		legacyIterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hashBytes),
	)
	if parts := strings.Split(legacyHash, "$"); len(parts) != 5 {
		t.Fatalf("legacyHash 段数非法: got %d want 5", len(parts))
	}

	ok, err := pm.VerifyPassword(plaintext, legacyHash)
	if err != nil || !ok {
		t.Fatalf("100k 旧哈希正确明文应通过: ok=%v err=%v", ok, err)
	}
	ok, err = pm.VerifyPassword("not-the-password", legacyHash)
	if err != nil || ok {
		t.Fatalf("100k 旧哈希错误明文应拒绝: ok=%v err=%v", ok, err)
	}
}

// A5. TestVerifyPassword_RejectsNon32ByteHash：CRYPTO-02 验收。
// 非 32 字节存储哈希属损坏/伪造数据，必须 err!=nil && ok==false，绝不再被当作 keyLen 喂给 KDF。
func TestVerifyPassword_RejectsNon32ByteHash(t *testing.T) {
	pm := NewPasswordManager(nil)
	salt := make([]byte, 16)

	for _, n := range []int{0, 16, 31, 33, 64} {
		forgedHash := make([]byte, n)
		forged := fmt.Sprintf("$sm3$1000$%s$%s",
			base64.RawStdEncoding.EncodeToString(salt),
			base64.RawStdEncoding.EncodeToString(forgedHash),
		)
		ok, err := pm.VerifyPassword("any-password", forged)
		if err == nil {
			t.Errorf("N=%d: 非 32 字节哈希期望 error，实际 err=nil（不得把 len(hashBytes) 作 keyLen 喂给 KDF）", n)
		}
		if ok {
			t.Errorf("N=%d: 非 32 字节哈希 ok 必须为 false", n)
		}
		if err != nil && !strings.Contains(err.Error(), "32") && !strings.Contains(err.Error(), "长度") {
			t.Errorf("N=%d: 错误消息应含「32」或「长度」关键词, got %q", n, err.Error())
		}
	}

	// 非法 iterations 存储值：走 KDF error 兜底链路（Task 2 第 4 步）
	for _, iter := range []string{"0", "-1"} {
		h := make([]byte, 32)
		forged := fmt.Sprintf("$sm3$%s$%s$%s",
			iter,
			base64.RawStdEncoding.EncodeToString(salt),
			base64.RawStdEncoding.EncodeToString(h),
		)
		ok, err := pm.VerifyPassword("any-password", forged)
		if err == nil || ok {
			t.Errorf("iterations=%s 非法存储值应 err!=nil && ok==false, got ok=%v err=%v", iter, ok, err)
		}
	}
}

// A6. TestPbkdf2SM3_SliceAliasFixed：证明 make+append 修复了 :75 slice 别名污染，且输出不变。
func TestPbkdf2SM3_SliceAliasFixed(t *testing.T) {
	pm := NewPasswordManager(nil)

	// 构造 cap 富余的 password 底层数组，salt 故意用可辨识字节
	backing := make([]byte, 64)
	for i := range backing {
		backing[i] = 0xAA
	}
	password := backing[0:8] // len=8, cap=64：旧 append(password, salt...) 会原地改写 backing[8:]
	salt := []byte{0x01, 0x02, 0x03, 0x04}

	dk, err := pm.pbkdf2SM3(password, salt, 1, 32)
	if err != nil {
		t.Fatalf("pbkdf2SM3 failed: %v", err)
	}

	// 旧实现会把 salt 写进 backing[8:12]；修复后 backing 全段仍应保持 0xAA
	for i, b := range backing {
		if b != 0xAA {
			t.Fatalf("slice 别名污染未修复: backing[%d]=0x%02x want 0xAA", i, b)
		}
	}

	// 输出必须与 A1 的 iter=1 基线一致（用等价输入：password=8×0xAA, salt=01020304 不对应基线，
	// 故此处仅断言「同样输入两次结果稳定」+「len=32」，真正的逐字节锁在 A1）
	dk2, err := pm.pbkdf2SM3(password, salt, 1, 32)
	if err != nil || !bytes.Equal(dk, dk2) || len(dk) != 32 {
		t.Fatalf("slice 修复后输出应稳定且 len=32: err=%v len=%d equal=%v", err, len(dk), bytes.Equal(dk, dk2))
	}
}
