package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// TestPersistDiscoveryResults_AtomicRollback 锁住 v1.55 收档登记的
// 「persistDiscoveryResults 未开事务」缺陷（quick 261109-wdu 清偿）。
//
// 缺陷形态：Delete(旧结果) 与 Create(新结果) 原先是两条独立语句，Create 失败时
// 旧结果已经被删 → 出现「旧结果已清、新结果未写」的空档。修复 = 包进单个事务，
// 事务内全走 tx，任一步失败整体回滚。
//
// 红绿判据：先持久化 2 条旧结果，再注入 sys_device_discovery_result 的 Create
// 失败，调用 persistDiscoveryResults 写新结果。修复前查询得到 0 条（旧结果丢失）；
// 修复后事务回滚，旧结果 2 条仍在。
//
// 复用 newDdv7906（文件型 sqlite + 实例级 callback manager），零额外基建；
// callback 注入手法沿用 TestImportDiscoveredDevices_AtomicRollback（Phase 142）。
func TestPersistDiscoveryResults_AtomicRollback(t *testing.T) {
	ctx := context.Background()
	svc, db := newDdv7906(t)

	// 先落 2 条旧结果（Delete+Create 均成功，事务应正常提交）。
	require.NoError(t, svc.persistDiscoveryResults(ctx, "task-atomic", []*DiscoveredDevice{
		{IPAddress: "10.155.0.1", IsAlive: true, SysName: "old-1"},
		{IPAddress: "10.155.0.2", IsAlive: true, SysName: "old-2"},
	}))
	require.Len(t, loadDiscoveryResults(t, db, "task-atomic"), 2, "前置：旧结果已落库")

	// 注入仅影响本 db 句柄的 Create 失败（表名=gorm 自动推导的 sys_device_discovery_result）。
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register(
		"test-wdu:force_discovery_result_failure",
		func(tx *gorm.DB) {
			if tx != nil && tx.Statement != nil && tx.Statement.Table == "sys_device_discovery_result" {
				_ = tx.AddError(errors.New("injected create failure"))
			}
		}))

	err := svc.persistDiscoveryResults(ctx, "task-atomic", []*DiscoveredDevice{
		{IPAddress: "10.155.0.9", IsAlive: true, SysName: "new-1"},
	})
	require.Error(t, err, "注入的 Create 失败必须冒泡")
	assert.Contains(t, err.Error(), "保存发现结果失败")

	// 事务回滚：Delete 未提交，旧结果 2 条完好，新结果 0 条。
	rows := loadDiscoveryResults(t, db, "task-atomic")
	require.Len(t, rows, 2, "事务回滚后旧结果必须完好（修复前会变成 0 条）")
	assert.Equal(t, "10.155.0.1", rows[0].IPAddress)
	assert.Equal(t, "10.155.0.2", rows[1].IPAddress)
}

// loadDiscoveryResults 按 seq 升序读回某任务的持久化结果明细。
func loadDiscoveryResults(t *testing.T, db *gorm.DB, discoveryID string) []models.DeviceDiscoveryResult {
	t.Helper()
	var rows []models.DeviceDiscoveryResult
	require.NoError(t, db.Where("discovery_id = ?", discoveryID).Order("seq ASC").Find(&rows).Error)
	return rows
}
