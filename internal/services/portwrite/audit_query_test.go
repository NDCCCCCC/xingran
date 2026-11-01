package portwrite

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ====================== Phase 127 前置（PWX-04）审计查询测试 ======================

// TestListAudit_FilterAndPaginate 分页 + 各过滤维度。
func TestListAudit_FilterAndPaginate(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(&countingExecutor{}, nil, db)

	rows := []*models.PortWriteAudit{
		{DeviceID: "dev-1", PortID: "p1", Action: "shutdown", Status: "succeeded", Operator: "alice", CreatedAt: time.Now().Add(-2 * time.Hour)},
		{DeviceID: "dev-1", PortID: "p2", Action: "set_access_vlan", Status: "failed", Operator: "bob", CreatedAt: time.Now().Add(-1 * time.Hour)},
		{DeviceID: "dev-2", PortID: "p3", Action: "shutdown", Status: "succeeded", Operator: "alice", CreatedAt: time.Now()},
	}
	for _, r := range rows {
		require.NoError(t, db.Create(r).Error)
	}
	ctx := context.Background()

	// 全量分页：created_at 倒序
	res, err := svc.ListAudit(ctx, AuditListRequest{Current: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(3), res.Total)
	assert.Len(t, res.List, 2)
	assert.Equal(t, "dev-2", res.List[0].DeviceID, "created_at 倒序")

	// status 过滤
	res, err = svc.ListAudit(ctx, AuditListRequest{Status: "failed"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	assert.Equal(t, "bob", res.List[0].Operator)

	// action + deviceId 组合
	res, err = svc.ListAudit(ctx, AuditListRequest{DeviceID: "dev-1", Action: "shutdown"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)

	// operator 过滤
	res, err = svc.ListAudit(ctx, AuditListRequest{Operator: "alice"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total)

	// 时间范围：只取最近 90 分钟
	start := time.Now().Add(-90 * time.Minute)
	res, err = svc.ListAudit(ctx, AuditListRequest{StartTime: &start})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total)

	// portId 过滤
	res, err = svc.ListAudit(ctx, AuditListRequest{PortID: "p3"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)

	// 默认分页兜底（零值 → current=1, pageSize=20）
	res, err = svc.ListAudit(ctx, AuditListRequest{})
	require.NoError(t, err)
	assert.Equal(t, 1, res.Current)
	assert.Equal(t, 20, res.PageSize)
}

// TestListAudit_BatchIdFilter Phase 122 收尾：batchId 精确对账过滤。
func TestListAudit_BatchIdFilter(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(&countingExecutor{}, nil, db)

	rows := []*models.PortWriteAudit{
		{DeviceID: "dev-1", PortID: "p1", Action: "shutdown", Status: "succeeded", Operator: "a", BatchID: "batch-A"},
		{DeviceID: "dev-1", PortID: "p2", Action: "shutdown", Status: "succeeded", Operator: "a", BatchID: "batch-A"},
		{DeviceID: "dev-1", PortID: "p3", Action: "shutdown", Status: "succeeded", Operator: "a"}, // 历史行 NULL
	}
	for _, r := range rows {
		require.NoError(t, db.Create(r).Error)
	}
	ctx := context.Background()

	res, err := svc.ListAudit(ctx, AuditListRequest{BatchID: "batch-A"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total, "batchId 等值过滤")

	res, err = svc.ListAudit(ctx, AuditListRequest{BatchID: "batch-NULL-not-exist"})
	require.NoError(t, err)
	assert.Equal(t, int64(0), res.Total)

	// 无 batchId → 全量（含 NULL 历史行）
	res, err = svc.ListAudit(ctx, AuditListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), res.Total)
}
