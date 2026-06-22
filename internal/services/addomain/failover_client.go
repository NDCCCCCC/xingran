package addomain

import (
	"context"
	"fmt"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
)

// FailoverClient 多账号故障切换客户端
//
// Phase 36: 包装 LDAPClient，提供"任一账号成功即返回"的语义
//
// 设计要点（Issue 3 + Issue 19）：
//   - ListAvailable 拿账号快照（事务 + SELECT FOR UPDATE 保证一致性）
//   - 顺序遍历快照，简化逻辑（去掉 triedIDs 防重 + random pick）
//   - maxAttempts = min(DefaultMaxHops, len(available))
//   - 失败时通过 pool.MarkFailure 上报（pool 内部加锁）
type FailoverClient struct {
	pool   AccountPool
	config *models.ADConfig
}

// NewFailoverClient 创建 FailoverClient
func NewFailoverClient(pool AccountPool, config *models.ADConfig) *FailoverClient {
	return &FailoverClient{pool: pool, config: config}
}

// ExecuteWithFailover 用池中可用账号依次尝试执行 operation，任一成功即返回
//
// operation 接收已 Bind 成功的 LDAPClient，可执行任意 AD 操作（搜索/修改/移动等）
func (f *FailoverClient) ExecuteWithFailover(
	ctx context.Context,
	operation func(client *LDAPClient) error,
) error {
	available, err := f.pool.ListAvailable(ctx, f.config.ID)
	if err != nil {
		return fmt.Errorf("查询账号池失败: %w", err)
	}
	if len(available) == 0 {
		return ErrAllAccountsUnavailable
	}

	maxAttempts := len(available)
	if maxAttempts > DefaultMaxHops {
		maxAttempts = DefaultMaxHops
	}

	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		acct := &available[i]

		client := NewLDAPClient(f.config, acct)
		if err := client.Connect(); err != nil {
			f.pool.MarkFailure(ctx, acct.ID, "dial:"+err.Error())
			lastErr = err
			applogger.Warnf("[Failover] 账号 %s 连接失败，尝试下一个 [ID=%s, err=%v]",
				acct.Username, acct.ID, err)
			continue
		}

		err := operation(client)
		client.Close()

		if err == nil {
			return nil
		}
		// 失败已在 client.Bind 内通过 pool 上报（同步）
		lastErr = err
		applogger.Warnf("[Failover] 账号 %s 操作失败，尝试下一个 [ID=%s, err=%v]",
			acct.Username, acct.ID, err)
	}
	return fmt.Errorf("账号池 %d 个账号均失败: %w", maxAttempts, lastErr)
}

// PickFirstConnect 与 ExecuteWithFailover 类似，但只建立连接不做后续操作
// （用于 ad_authenticator 的 "绑管理员" 场景）
//
// 返回：可用的 LDAPClient + 实际用到的账号（供调用方记录/审计）
func (f *FailoverClient) PickFirstConnect(ctx context.Context) (*LDAPClient, *models.ADServiceAccount, error) {
	available, err := f.pool.ListAvailable(ctx, f.config.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("查询账号池失败: %w", err)
	}
	if len(available) == 0 {
		return nil, nil, ErrAllAccountsUnavailable
	}

	maxAttempts := len(available)
	if maxAttempts > DefaultMaxHops {
		maxAttempts = DefaultMaxHops
	}

	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		acct := &available[i]

		client := NewLDAPClient(f.config, acct)
		if err := client.Connect(); err != nil {
			f.pool.MarkFailure(ctx, acct.ID, "dial:"+err.Error())
			lastErr = err
			continue
		}
		return client, acct, nil
	}
	return nil, nil, fmt.Errorf("账号池 %d 个账号均无法连接: %w", maxAttempts, lastErr)
}