package addomain

// ======== Phase 76-03: FailoverClient 接口驱动测试 ========
//
// 经 FailoverClient.clientFactory 注入 mockLDAPClient 工厂（同包白盒赋值），
// 零真实网络驱动顺序遍历 / maxHops 封顶 / mock 多批次（walk/分页）三个语义。
// 复用 account_pool_test.go 的 setupTestPool / insertAccount（勿重写 helper）。

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// dialErrForTest 统一的 Connect 失败错误（dummy 值，无真实 AD 信息）
var dialErrForTest = errors.New("dial tcp: connection refused")

// TestFailover_SequentialTraversal_StopsOnFirstSuccess 顺序遍历语义：
// 账号0（username-0）Connect 失败 → MarkFailure 落库（failure_count+1）；
// 账号1（username-1）Connect 成功 → operation 执行一次即止 → MarkSuccess 路径。
func TestFailover_SequentialTraversal_StopsOnFirstSuccess(t *testing.T) {
	pool, db, configID := setupTestPool(t)
	ctx := context.Background()

	// 插入顺序即 ListAvailable（GORM Find 无 ORDER BY，sqlite 按 rowid）返回顺序
	insertAccount(t, db, configID, "username-0", AccountStatusAvailable)
	insertAccount(t, db, configID, "username-1", AccountStatusAvailable)

	failMock := &mockLDAPClient{connectErr: dialErrForTest}
	okMock := &mockLDAPClient{}

	var factoryCalls int
	var operationCalls int
	cfg := &models.ADConfig{BaseModel: models.BaseModel{ID: configID}}
	fc := NewFailoverClient(pool, cfg)
	fc.clientFactory = func(_ *models.ADConfig, acct *models.ADServiceAccount) LDAPClientIface {
		factoryCalls++
		if acct.Username == "username-1" {
			return okMock
		}
		return failMock
	}

	err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		operationCalls++
		return client.AddGroupMember("CN=group,DC=example,DC=com", "CN=user,DC=example,DC=com")
	})
	require.NoError(t, err)

	// 成功即止：operation 恰执行 1 次；工厂恰构造 2 个客户端（账号0 失败 + 账号1 成功）
	assert.Equal(t, 1, operationCalls, "成功即止：operation 只应执行一次")
	assert.Equal(t, 2, factoryCalls, "应恰好尝试 2 个账号（0 失败 → 1 成功）")
	assert.Equal(t, 1, okMock.closeCalls, "成功账号的客户端应被 Close")

	// 账号0 走 MarkFailure 路径：failure_count 落库 +1（注入缝不破坏审计簿记）
	var failRow struct{ FailureCount int }
	require.NoError(t, db.Raw(
		`SELECT failure_count FROM sys_ad_service_accounts WHERE username = ?`, "username-0",
	).Scan(&failRow).Error)
	assert.Equal(t, 1, failRow.FailureCount, "username-0 应被 MarkFailure（failure_count=1）")

	// 账号1 走 MarkSuccess 路径：failure_count 归零 + last_success_at 落库
	var okRowCount int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM sys_ad_service_accounts
		 WHERE username = ? AND failure_count = 0 AND last_success_at IS NOT NULL`, "username-1",
	).Scan(&okRowCount).Error)
	assert.Equal(t, int64(1), okRowCount, "username-1 应走 MarkSuccess（failure_count=0 且 last_success_at 落库）")
}

// TestFailover_MaxHops_CapsAtDefaultMaxHops maxHops 封顶语义：
// 池含 12 个可用账号（全部 Connect 失败 mock）→ 工厂调用次数封顶
// min(DefaultMaxHops=10, len(available)=12) = DefaultMaxHops，operation 零执行。
func TestFailover_MaxHops_CapsAtDefaultMaxHops(t *testing.T) {
	pool, db, configID := setupTestPool(t)
	ctx := context.Background()

	for i := 0; i < 12; i++ {
		insertAccount(t, db, configID, fmt.Sprintf("username-%d", i), AccountStatusAvailable)
	}

	var factoryCalls int
	cfg := &models.ADConfig{BaseModel: models.BaseModel{ID: configID}}
	fc := NewFailoverClient(pool, cfg)
	fc.clientFactory = func(_ *models.ADConfig, _ *models.ADServiceAccount) LDAPClientIface {
		factoryCalls++
		return &mockLDAPClient{connectErr: dialErrForTest}
	}

	err := fc.ExecuteWithFailover(ctx, func(client LDAPClientIface) error {
		t.Error("全部 Connect 失败时 operation 不应被执行")
		return nil
	})

	require.Error(t, err)
	assert.Equal(t, DefaultMaxHops, factoryCalls,
		"12 个可用账号但尝试次数应封顶 DefaultMaxHops（常量断言，非魔法数）")
	// 真实语义（Rule 1 修正计划笔误）：封顶耗尽返回「账号池 N 个账号均失败」聚合错误，
	// ErrAllAccountsUnavailable 仅在池空（len(available)==0）时返回。
	assert.Contains(t, err.Error(), "均失败")
	assert.NotErrorIs(t, err, ErrAllAccountsUnavailable,
		"池非空时封顶路径不应返回 ErrAllAccountsUnavailable")
}

// TestFailover_MockSearchUsersFn_MultiBatch mock walk/分页驱动能力：
// searchUsersFn 函数字段非 nil 时优先于 searchUsersRes，支持多批次多次调用
// （RESEARCH Open Question 2 窄解读：驱动 service 层遍历，非 wire 级分页）。
func TestFailover_MockSearchUsersFn_MultiBatch(t *testing.T) {
	batch1 := []*ldap.Entry{
		ldap.NewEntry("CN=u1,DC=example,DC=com", map[string][]string{"sAMAccountName": {"u1"}}),
	}
	batch2 := []*ldap.Entry{
		ldap.NewEntry("CN=u2,DC=example,DC=com", map[string][]string{"sAMAccountName": {"u2"}}),
		ldap.NewEntry("CN=u3,DC=example,DC=com", map[string][]string{"sAMAccountName": {"u3"}}),
	}

	calls := 0
	mock := &mockLDAPClient{
		// 同时设 searchUsersRes，验证 fn 非 nil 时优先走 fn 分支
		searchUsersRes: []*ldap.Entry{},
		searchUsersFn: func() ([]*ldap.Entry, error) {
			calls++
			if calls == 1 {
				return batch1, nil
			}
			return batch2, nil
		},
	}

	r1, err := mock.SearchUsers("DC=example,DC=com")
	require.NoError(t, err)
	r2, err := mock.SearchUsers("DC=example,DC=com")
	require.NoError(t, err)

	assert.Len(t, r1, 1, "第一批返回 1 条")
	assert.Len(t, r2, 2, "第二批返回 2 条（连续两次调用返回不同批次）")
	assert.Equal(t, "CN=u1,DC=example,DC=com", r1[0].DN)
	assert.Equal(t, "CN=u2,DC=example,DC=com", r2[0].DN)
	assert.Equal(t, 2, mock.searchUsersCalls)
}
