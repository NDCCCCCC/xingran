package dbbackup

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// 备份计划敏感凭据的字段级加解密（复用 pkg/crypto.SM4Cipher）
// ---------------------------------------------------------------------------
//
// 背景：config.go 早期把 OSS accessKey/secretKey、FTP password 按字面写入
// sys_config，任何能读库/看参数管理页的人都能拿到明文凭据。本文件把它们改为
// **字段级 SM4-GCM 加密后落库、读取时解密**：
//
//   - 落库格式：credentialCipherPrefix + pkg/crypto.SM4Cipher.Encrypt 的输出
//     （base64(nonce+ciphertext)）。带前缀是为了让 loadPlan 能区分「密文」与
//     「历史明文/空值」，平滑兼容改造前已存在的明文配置，不需要停机迁移。
//   - 空串不加密、不落前缀：保持「未配置」语义（前端靠空串区分未配置/已配置）。
//   - cipher 由装配层通过 RegisterCredentialCipher 注入（core.SM4Cipher，与
//     AD 域密码共用同一把 SM4_KEY）。这是**包级注册器**：dbbackup 的
//     loadPlan/savePlan 都是包级函数（被 core 的定时任务 svc 与 HTTP svc 共享），
//     用包级动态读取而非 NewService 时快照，可保证「路由装配完成即对所有实例生效」，
//     也避免改动 NewService 的 4 参签名（core.go / router.go 归他人所有权）。
//   - 写路径 fail-closed：需要加密新凭据但 cipher 未注册时，返回明确错误，
//     **绝不把明文凭据落库**。读路径遇到密文但 cipher 未注册/密钥不匹配时同样
//     返回错误，不静默把密文串当凭据交给存储适配器。
//
// 注意：GetPlan 的 ****** 脱敏、SavePlan 的占位符回填仍在 API 层
// （internal/api/v1/dbbackup/handler.go）保持；服务层 savePlan 另加一道
// 占位符/空串防御，双保险确保「用户未改凭据时不会把占位符写成真值」。

const (
	// credentialCipherPrefix 密文前缀，标识该 sys_config 值已是 SM4 密文。
	credentialCipherPrefix = "sm4:"
)

// PlanMaskPlaceholder 敏感字段脱敏占位符（导出常量：API 层 handler 与本包共用，
// 不再各自定义以免漂移）。GetPlan 出参把已存凭据替换成它；SavePlan 收到它代表
// 「用户未改动，保留原值」。
const PlanMaskPlaceholder = "******"

// CredentialClearMarker 显式清空凭据标记：SavePlan 收到它表示「管理员要求删除
// 该凭据」。空串仍表示「未改动」（兼容前端 261110-tbq 约定），因此清空必须用
// 显式标记表达；handler 与服务层 savePlan 都识别该标记，最终落库值为空串。
const CredentialClearMarker = "__CLEAR__"

// ErrCredentialDecrypt 凭据解密失败 sentinel：GetPlan 的降级路径用它区分
// 「凭据解密失败（可降级展示）」与其它数据库错误。
var ErrCredentialDecrypt = errors.New("备份计划凭据解密失败")

// CredentialCipher 凭据字段加解密接口。
// 方法集与 pkg/crypto.SM4Cipher、addomain.PasswordCipher 完全一致，
// 装配层可直接把 core.SM4Cipher 传进来（Go 接口隐式实现）。
type CredentialCipher interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

var (
	credentialCipherMu sync.RWMutex
	credentialCipher   CredentialCipher
)

// RegisterCredentialCipher 注册进程级凭据加解密器。
//
// 由 internal/api/v1/dbbackup 在装配 core 时调用一次（注入 core.SM4Cipher）。
// 线程安全，可重复调用（后者覆盖前者，便于切换密钥/单测注入）。
// 传 nil 表示注销（仅测试或降级场景使用）。
func RegisterCredentialCipher(c CredentialCipher) {
	credentialCipherMu.Lock()
	defer credentialCipherMu.Unlock()
	credentialCipher = c
}

// currentCredentialCipher 取当前注册器（每次加解密动态读取，不快照）。
func currentCredentialCipher() CredentialCipher {
	credentialCipherMu.RLock()
	defer credentialCipherMu.RUnlock()
	return credentialCipher
}

// encryptCredential 加密单个凭据字段。
// 空串原样返回（不加密、不加前缀）；非空必须有 cipher，否则拒绝明文落库。
func encryptCredential(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	c := currentCredentialCipher()
	if c == nil {
		return "", fmt.Errorf("凭据加密器未初始化（SM4_KEY/装配未完成），拒绝明文落库")
	}
	cipherText, err := c.Encrypt(plain)
	if err != nil {
		return "", fmt.Errorf("加密备份计划凭据失败: %w", err)
	}
	return credentialCipherPrefix + cipherText, nil
}

// decryptCredential 解密单个凭据字段。
//   - 无前缀（历史明文 / 空值）：原样返回，兼容改造前已落库的明文配置；
//   - 有前缀：必须用当前 cipher 解密；失败返回错误，绝不回吐密文串。
func decryptCredential(stored string) (string, error) {
	if stored == "" || !strings.HasPrefix(stored, credentialCipherPrefix) {
		return stored, nil
	}
	c := currentCredentialCipher()
	if c == nil {
		return "", fmt.Errorf("%w: 凭据加密器未初始化，无法解密已加密的备份计划凭据（请检查 SM4_KEY 与装配顺序）", ErrCredentialDecrypt)
	}
	plain, err := c.Decrypt(strings.TrimPrefix(stored, credentialCipherPrefix))
	if err != nil {
		return "", fmt.Errorf("%w: 解密备份计划凭据失败（SM4_KEY 是否与加密时一致？）: %w", ErrCredentialDecrypt, err)
	}
	return plain, nil
}

// isCredentialUnchanged 判断 SavePlan 入参的凭据字段是否代表「用户未改动」。
// 与 API 层约定一致：脱敏占位符 "******" 与空串（含纯空白）都表示保留原值。
// 空串凭据本身无业务意义，若当作「清空」会导致管理员一保存计划就丢失凭据。
func isCredentialUnchanged(v string) bool {
	// 显式清空标记不是「未改动」：不等于占位符也非空串，自然返回 false。
	return v == PlanMaskPlaceholder || strings.TrimSpace(v) == ""
}

// isCredentialClear 判断入参是否为显式清空凭据标记（__CLEAR__）。
func isCredentialClear(v string) bool {
	return v == CredentialClearMarker
}

// ---------------------------------------------------------------------------
// 恢复收尾一致性：进程内菜单/权限缓存清理（包级注册器，同 CredentialCipher）
// ---------------------------------------------------------------------------

// RestoreCacheFlusher 清理进程内菜单/权限缓存的钩子。
// 由 API 装配层用 core.Cache 构造并注册；返回错误仅记日志，不阻断任务终态。
type RestoreCacheFlusher func(ctx context.Context) error

var (
	restoreCacheFlusherMu sync.RWMutex
	restoreCacheFlusher   RestoreCacheFlusher
)

// RegisterRestoreCacheFlusher 注册恢复收尾的缓存清理钩子（线程安全，可覆盖）。
func RegisterRestoreCacheFlusher(f RestoreCacheFlusher) {
	restoreCacheFlusherMu.Lock()
	defer restoreCacheFlusherMu.Unlock()
	restoreCacheFlusher = f
}

// currentRestoreCacheFlusher 取当前缓存清理钩子。
func currentRestoreCacheFlusher() RestoreCacheFlusher {
	restoreCacheFlusherMu.RLock()
	defer restoreCacheFlusherMu.RUnlock()
	return restoreCacheFlusher
}
