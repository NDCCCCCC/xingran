package rpa

// DEBT-03（v1.51）：WorkerTokenService 行为测试。
//
// 覆盖：Provisioning 白名单 constant-time 校验（命中/未命中/空）、Token 换发
// （已有 worker 行轮换 / 无行时创建）、ValidateWorkerToken（无效前缀 / 不存在 /
// 过期 / 有效）、RevokeWorkerToken。SQLite 内存库，不 mock gorm 层。

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
)

func setupWorkerTokenTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&rpamodels.Worker{}))
	return db
}

func tokenAuthCfg(tokens ...string) config.RPAWorkerAuthConfig {
	return config.RPAWorkerAuthConfig{
		ProvisioningTokens: tokens,
		AuthRequired:       true,
		TokenExpireDays:    90,
	}
}

func TestWorkerTokenService_ProvisioningWhitelist(t *testing.T) {
	db := setupWorkerTokenTestDB(t)
	svc := NewWorkerTokenService(db, tokenAuthCfg("rpa-prod-main-2026", "rpa-staging-2026"))

	if !svc.ValidateProvisioningToken("rpa-prod-main-2026") {
		t.Fatal("白名单内令牌应命中")
	}
	if !svc.ValidateProvisioningToken("rpa-staging-2026") {
		t.Fatal("第二个白名单令牌应命中")
	}
	for _, bad := range []string{"", "wrong-token", "rpa-prod-main-2026 ", "RPA-PROD-MAIN-2026"} {
		if svc.ValidateProvisioningToken(bad) {
			t.Errorf("令牌 %q 不应命中", bad)
		}
	}
	// 未配置白名单 → 全部拒绝
	empty := NewWorkerTokenService(db, config.RPAWorkerAuthConfig{})
	if empty.ValidateProvisioningToken("rpa-prod-main-2026") {
		t.Fatal("白名单为空应全部拒绝")
	}
}

func TestWorkerTokenService_IssueAndValidate(t *testing.T) {
	db := setupWorkerTokenTestDB(t)
	ctx := context.Background()
	svc := NewWorkerTokenService(db, tokenAuthCfg("prov-1"))

	// 预置已注册 worker（无 token）
	worker := rpamodels.Worker{WorkerID: "w-1", WorkerName: "节点一", Status: rpamodels.WorkerStatusOnline}
	require.NoError(t, db.Create(&worker).Error)

	pair, err := svc.IssueWorkerToken(ctx, "w-1")
	require.NoError(t, err)
	if pair == nil || !strings.HasPrefix(pair.Token, "wk_") {
		t.Fatalf("Token 应带 wk_ 前缀, got %v", pair)
	}
	if pair.ExpiresAt.Before(time.Now().AddDate(0, 0, 89)) || pair.ExpiresAt.After(time.Now().AddDate(0, 0, 91)) {
		t.Fatalf("有效期应约 90 天, got %s", pair.ExpiresAt)
	}

	// 校验通过且 claims 归属正确
	claims, err := svc.ValidateWorkerToken(ctx, pair.Token)
	require.NoError(t, err)
	if claims.WorkerID != "w-1" || claims.WorkerName != "节点一" {
		t.Fatalf("claims 归属不符: %+v", claims)
	}

	// 再次换发 → 轮换，旧 token 失效
	pair2, err := svc.IssueWorkerToken(ctx, "w-1")
	require.NoError(t, err)
	if pair2.Token == pair.Token {
		t.Fatal("轮换应产生新 token")
	}
	if _, err := svc.ValidateWorkerToken(ctx, pair.Token); err == nil {
		t.Fatal("旧 token 轮换后应失效")
	}
	if _, err := svc.ValidateWorkerToken(ctx, pair2.Token); err != nil {
		t.Fatalf("新 token 应有效: %v", err)
	}

	// worker 行不存在 → 创建最小行并发 token（宽限期兼容路径）
	pair3, err := svc.IssueWorkerToken(ctx, "w-new")
	require.NoError(t, err)
	if pair3 == nil || !strings.HasPrefix(pair3.Token, "wk_") {
		t.Fatalf("新 worker 应创建并发 token, got %v", pair3)
	}
	var created rpamodels.Worker
	require.NoError(t, db.Where("worker_id = ?", "w-new").First(&created).Error)
	if created.Token != pair3.Token {
		t.Fatal("创建行的 token 应落库")
	}
}

func TestWorkerTokenService_ValidateFailures(t *testing.T) {
	db := setupWorkerTokenTestDB(t)
	ctx := context.Background()
	svc := NewWorkerTokenService(db, tokenAuthCfg("prov-1"))

	worker := rpamodels.Worker{WorkerID: "w-1", WorkerName: "n", Token: "wk_valid", TokenExpiresAt: ptrTime(time.Now().AddDate(0, 0, 1))}
	require.NoError(t, db.Create(&worker).Error)

	// 无效形态
	for _, bad := range []string{"", "not-wk-format", "wk_nope"} {
		if _, err := svc.ValidateWorkerToken(ctx, bad); err == nil {
			t.Errorf("token %q 应校验失败", bad)
		}
	}
	// 过期
	expired := rpamodels.Worker{WorkerID: "w-2", WorkerName: "n2", Token: "wk_expired", TokenExpiresAt: ptrTime(time.Now().AddDate(0, 0, -1))}
	require.NoError(t, db.Create(&expired).Error)
	if _, err := svc.ValidateWorkerToken(ctx, "wk_expired"); err == nil || !strings.Contains(err.Error(), "过期") {
		t.Fatalf("过期 token 应报过期, got %v", err)
	}
}

func TestWorkerTokenService_Revoke(t *testing.T) {
	db := setupWorkerTokenTestDB(t)
	ctx := context.Background()
	svc := NewWorkerTokenService(db, tokenAuthCfg("prov-1"))

	worker := rpamodels.Worker{WorkerID: "w-1", WorkerName: "n", Token: "wk_revoke-me", TokenExpiresAt: ptrTime(time.Now().AddDate(0, 0, 1))}
	require.NoError(t, db.Create(&worker).Error)

	require.NoError(t, svc.RevokeWorkerToken(ctx, "w-1"))
	if _, err := svc.ValidateWorkerToken(ctx, "wk_revoke-me"); err == nil {
		t.Fatal("撤销后 token 应失效")
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
