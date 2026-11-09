// Phase 142 / DISC-01 — ImportDiscoveredDevices 真实导入单测。
//
// 同包 services，复用 newDdv7906 文件型 sqlite（每个用例独立 t.TempDir()），
// 迁移时通过本文件的 newImport7906 helper 追加 &models.NetworkDevice{}。
//
// 测试红线：newNetworkTestEnv 是 :memory: + SetMaxOpenConns(1)，事务回调内
// 再发 gorm 查询会死锁；本文件一律用文件型 sqlite，不碰网络模块 handler 基建。
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

// newImport7906 组装带 sys_network_device 迁移的 DeviceDiscoveryService。
func newImport7906(t *testing.T) (*DeviceDiscoveryService, *gorm.DB) {
	t.Helper()
	db := newDB7906(t,
		&models.DeviceDiscovery{},
		&models.DeviceDiscoveryResult{},
		&models.AuthCredential{},
		&models.NetworkDevice{}, // Phase 142：真实导入目标表
	)
	return &DeviceDiscoveryService{db: db}, db
}

// seedImportTask 直接插入一条发现任务（带合法 UUID GroupID 与自定义 SNMPPort）。
func seedImportTask(t *testing.T, db *gorm.DB, id string, groupID *string, snmpPort int) {
	t.Helper()
	task := &models.DeviceDiscovery{
		ID:              id,
		TaskName:        "import-" + id,
		DiscoveryType:   models.DiscoveryTypeSNMP,
		IPRanges:        models.IPRangeList{{StartIP: "10.142.0.1", EndIP: "10.142.0.10"}},
		SNMPPort:        snmpPort,
		Status:          models.DiscoveryStatusSuccess,
		DiscoveredCount: 0,
		GroupID:         groupID,
	}
	require.NoError(t, db.Create(task).Error)
}

// seedImportResult 插入一条发现结果明细。
// 走 map create：IsAlive 带 `default:true`，gorm 对结构体零值 false 会无条件
// 替换成默认值 true（Select("*") 也拦不住），map 显式写列才能造出「死设备」用例。
func seedImportResult(t *testing.T, db *gorm.DB, id, discoveryID, ip string, r models.DeviceDiscoveryResult, seq int) {
	t.Helper()
	m := map[string]interface{}{
		"id":           id,
		"discovery_id": discoveryID,
		"ip_address":   ip,
		"device_type":  r.DeviceType,
		"vendor":       r.Vendor,
		"model":        r.Model,
		"sys_name":     r.SysName,
		"sys_descr":    r.SysDescr,
		"is_alive":     r.IsAlive,
		"seq":          seq,
	}
	require.NoError(t, db.Model(&models.DeviceDiscoveryResult{}).Create(m).Error)
}

func countImportDevices(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&models.NetworkDevice{}).Count(&n).Error)
	return n
}

// -----------------------------------------------------------------------------
// 用例
// -----------------------------------------------------------------------------

// TestImportDiscoveredDevices_AllRows 3 条不同 IP 全量导入，字段逐项映射。
func TestImportDiscoveredDevices_AllRows(t *testing.T) {
	ctx := context.Background()
	svc, db := newImport7906(t)
	groupID := "11111111-2222-3333-4444-555555555555"
	seedImportTask(t, db, "task-all", &groupID, 1161)

	seedImportResult(t, db, "r1", "task-all", "10.142.0.1", models.DeviceDiscoveryResult{
		DeviceType: "router", Vendor: "Huawei", Model: "AR3200", SysName: "edge-rtr", SysDescr: "HW router", IsAlive: true,
	}, 1)
	seedImportResult(t, db, "r2", "task-all", "10.142.0.2", models.DeviceDiscoveryResult{
		DeviceType: "switch", Vendor: "Cisco", Model: "C9300", SysName: "core-sw", SysDescr: "CSCO switch", IsAlive: true,
	}, 2)
	seedImportResult(t, db, "r3", "task-all", "10.142.0.3", models.DeviceDiscoveryResult{
		DeviceType: "firewall", Vendor: "H3C", Model: "F1000", SysName: "fw-1", SysDescr: "H3C fw", IsAlive: true,
	}, 3)

	n, err := svc.ImportDiscoveredDevices(ctx, "task-all", []string{"ignored-id"}, "operator-142")
	require.NoError(t, err)
	assert.Equal(t, 3, n, "deviceIDs 被忽略，恒按结果行全量导入")
	assert.Equal(t, int64(3), countImportDevices(t, db))

	var got models.NetworkDevice
	require.NoError(t, db.Where("ip_address = ?", "10.142.0.1").First(&got).Error)
	assert.Equal(t, "edge-rtr", got.DeviceName)
	assert.Equal(t, models.DeviceTypeRouter, got.DeviceType)
	assert.Equal(t, models.DeviceVendor("Huawei"), got.Vendor)
	assert.Equal(t, "AR3200", got.Model)
	assert.Equal(t, "10.142.0.1", got.IPAddress)
	assert.Equal(t, 22, got.Port)
	assert.Equal(t, 1161, got.SNMPPort, "SNMPPort 取 task.SNMPPort")
	assert.Equal(t, models.DeviceStatusOnline, got.Status, "Status 按 DeviceStatus 0在线")
	assert.Equal(t, "HW router", got.Description, "SysDescr → Description")
	require.NotNil(t, got.DeptID)
	assert.Equal(t, groupID, *got.DeptID, "DeptID 取 task.GroupID")
	assert.Equal(t, "operator-142", got.CreatedBy)
	assert.Equal(t, "operator-142", got.UpdatedBy)
	assert.NotEmpty(t, got.ID, "ID 由 BaseModel.BeforeCreate 自动补")
	assert.Nil(t, got.CredentialID)
	assert.Empty(t, got.SerialNumber)
	assert.Empty(t, got.SoftwareVersion)
	assert.Empty(t, got.Uptime)

	// 非法 GroupID 时不写脏 UUID（PLAN：校验合法 UUID）。
	bad := "not-a-uuid"
	svc2, db2 := newImport7906(t)
	seedImportTask(t, db2, "task-bad-org", &bad, 0)
	seedImportResult(t, db2, "rb", "task-bad-org", "10.142.1.1", models.DeviceDiscoveryResult{
		DeviceType: "switch", SysName: "sw", IsAlive: true,
	}, 1)
	n2, err := svc2.ImportDiscoveredDevices(ctx, "task-bad-org", nil, "op")
	require.NoError(t, err)
	assert.Equal(t, 1, n2)
	var badOrgDev models.NetworkDevice
	require.NoError(t, db2.Where("ip_address = ?", "10.142.1.1").First(&badOrgDev).Error)
	assert.Nil(t, badOrgDev.DeptID, "非法 UUID GroupID 不落库")
	assert.Equal(t, 161, badOrgDev.SNMPPort, "SNMPPort=0 回落默认 161")
}

// TestImportDiscoveredDevices_DedupByIP 批内同 IP 去重 / 库内存在跳过 / 软删恢复。
func TestImportDiscoveredDevices_DedupByIP(t *testing.T) {
	ctx := context.Background()

	t.Run("batch_dedup_keeps_last", func(t *testing.T) {
		svc, db := newImport7906(t)
		seedImportTask(t, db, "task-dd", nil, 161)
		// 同一 IP 两条，保留 seq 较大（后一条）的 SysName。
		seedImportResult(t, db, "dd1", "task-dd", "10.142.2.1", models.DeviceDiscoveryResult{
			SysName: "first", IsAlive: true,
		}, 1)
		seedImportResult(t, db, "dd2", "task-dd", "10.142.2.1", models.DeviceDiscoveryResult{
			SysName: "last", IsAlive: true,
		}, 2)

		n, err := svc.ImportDiscoveredDevices(ctx, "task-dd", nil, "op")
		require.NoError(t, err)
		assert.Equal(t, 1, n)
		assert.Equal(t, int64(1), countImportDevices(t, db))
		var got models.NetworkDevice
		require.NoError(t, db.Where("ip_address = ?", "10.142.2.1").First(&got).Error)
		assert.Equal(t, "last", got.DeviceName, "批内去重保留最后一条")
	})

	t.Run("existing_not_overwritten", func(t *testing.T) {
		svc, db := newImport7906(t)
		seedImportTask(t, db, "task-ex", nil, 161)
		seedImportResult(t, db, "ex1", "task-ex", "10.142.3.1", models.DeviceDiscoveryResult{
			SysName: "from-discovery", IsAlive: true,
		}, 1)
		// 库内预置同 IP 的未软删行，人工维护字段必须被保护、不被覆盖。
		require.NoError(t, db.Create(&models.NetworkDevice{
			DeviceName: "manual-maintained", DeviceType: models.DeviceTypeSwitch,
			Vendor: models.DeviceVendor("huawei"), IPAddress: "10.142.3.1",
			Port: 2222, Status: models.DeviceStatusOffline,
		}).Error)

		n, err := svc.ImportDiscoveredDevices(ctx, "task-ex", nil, "op")
		require.NoError(t, err)
		assert.Equal(t, 0, n, "库内未软删命中 → 跳过，不覆盖")
		assert.Equal(t, int64(1), countImportDevices(t, db))
		var got models.NetworkDevice
		require.NoError(t, db.Where("ip_address = ?", "10.142.3.1").First(&got).Error)
		assert.Equal(t, "manual-maintained", got.DeviceName)
		assert.Equal(t, 2222, got.Port)
		assert.Equal(t, models.DeviceStatusOffline, got.Status)
	})

	t.Run("soft_deleted_restored", func(t *testing.T) {
		svc, db := newImport7906(t)
		seedImportTask(t, db, "task-sd", nil, 161)
		seedImportResult(t, db, "sd1", "task-sd", "10.142.4.1", models.DeviceDiscoveryResult{
			SysName: "refreshed-name", DeviceType: "router", Vendor: "Huawei",
			Model: "NE40", SysDescr: "new descr", IsAlive: true,
		}, 1)
		// 预置一条已软删的同 IP 行，字段为旧值。
		require.NoError(t, db.Create(&models.NetworkDevice{
			DeviceName: "stale", DeviceType: models.DeviceTypeSwitch,
			Vendor: models.DeviceVendor("cisco"), IPAddress: "10.142.4.1",
			Port: 22, Status: models.DeviceStatusOffline, Description: "old",
		}).Error)
		require.NoError(t, db.Where("ip_address = ?", "10.142.4.1").Delete(&models.NetworkDevice{}).Error)
		assert.Equal(t, int64(0), countImportDevices(t, db), "删除后普通查询不可见")

		n, err := svc.ImportDiscoveredDevices(ctx, "task-sd", nil, "restorer")
		require.NoError(t, err)
		assert.Equal(t, 1, n, "软删恢复计入 count")
		assert.Equal(t, int64(1), countImportDevices(t, db), "恢复后可见但不新建重复行")
		var got models.NetworkDevice
		require.NoError(t, db.Where("ip_address = ?", "10.142.4.1").First(&got).Error)
		assert.False(t, got.DeletedAt.Valid)
		assert.Equal(t, "refreshed-name", got.DeviceName, "恢复时字段刷新")
		assert.Equal(t, models.DeviceTypeRouter, got.DeviceType)
		assert.Equal(t, models.DeviceVendor("Huawei"), got.Vendor)
		assert.Equal(t, "NE40", got.Model)
		assert.Equal(t, "new descr", got.Description)
		assert.Equal(t, models.DeviceStatusOnline, got.Status)
		assert.Equal(t, "restorer", got.CreatedBy)
		assert.Equal(t, "restorer", got.UpdatedBy)
	})
}

// TestImportDiscoveredDevices_SkipsDeadAndBadIP 死设备/空 IP/非法 IP 跳过 + 各类兜底。
func TestImportDiscoveredDevices_SkipsDeadAndBadIP(t *testing.T) {
	ctx := context.Background()
	svc, db := newImport7906(t)
	seedImportTask(t, db, "task-skip", nil, 161)

	seedImportResult(t, db, "sk1", "task-skip", "10.142.5.1", models.DeviceDiscoveryResult{
		SysName: "alive-bare", IsAlive: true, // DeviceType/Vendor 空 → 兜底
	}, 1)
	seedImportResult(t, db, "sk2", "task-skip", "10.142.5.2", models.DeviceDiscoveryResult{
		SysName: "bad-type", DeviceType: "unknown-type", Vendor: "", IsAlive: true,
	}, 2)
	seedImportResult(t, db, "sk3", "task-skip", "10.142.5.3", models.DeviceDiscoveryResult{
		SysName: "dead", IsAlive: false,
	}, 3)
	seedImportResult(t, db, "sk4", "task-skip", "", models.DeviceDiscoveryResult{
		SysName: "empty-ip", IsAlive: true,
	}, 4)
	seedImportResult(t, db, "sk5", "task-skip", "999.1.1.1", models.DeviceDiscoveryResult{
		SysName: "bad-ip", IsAlive: true,
	}, 5)
	seedImportResult(t, db, "sk6", "task-skip", "  10.142.5.6  ", models.DeviceDiscoveryResult{
		SysName: "trimmed", IsAlive: true,
	}, 6)

	n, err := svc.ImportDiscoveredDevices(ctx, "task-skip", nil, "op")
	require.NoError(t, err)
	assert.Equal(t, 3, n, "只有 3 条有效 IP 导入（死/空/非法 IP 跳过）")
	assert.Equal(t, int64(3), countImportDevices(t, db))

	var bare models.NetworkDevice
	require.NoError(t, db.Where("ip_address = ?", "10.142.5.1").First(&bare).Error)
	assert.Equal(t, "alive-bare", bare.DeviceName)
	assert.Equal(t, models.DeviceTypeSwitch, bare.DeviceType, "DeviceType 空兜底为 switch")
	assert.Empty(t, string(bare.Vendor), "Vendor 空 → 空串")

	var badType models.NetworkDevice
	require.NoError(t, db.Where("ip_address = ?", "10.142.5.2").First(&badType).Error)
	assert.Equal(t, models.DeviceTypeSwitch, badType.DeviceType, "DeviceType 越界兜底为 switch")

	var trimmed models.NetworkDevice
	require.NoError(t, db.Where("ip_address = ?", "10.142.5.6").First(&trimmed).Error)
	assert.Equal(t, "trimmed", trimmed.DeviceName, "IP trim 后归一入库")

	// SysName 空 → Device-<末段IP> 兜底，单独一个用例避免与上面混在一起。
	svc2, db2 := newImport7906(t)
	seedImportTask(t, db2, "task-fallback", nil, 161)
	seedImportResult(t, db2, "fb1", "task-fallback", "10.142.6.77", models.DeviceDiscoveryResult{
		IsAlive: true,
	}, 1)
	_, err = svc2.ImportDiscoveredDevices(ctx, "task-fallback", nil, "op")
	require.NoError(t, err)
	var fb models.NetworkDevice
	require.NoError(t, db2.Where("ip_address = ?", "10.142.6.77").First(&fb).Error)
	assert.Equal(t, "Device-77", fb.DeviceName)
}

// TestImportDiscoveredDevices_EmptyResults 无结果行 → (0, nil)。
func TestImportDiscoveredDevices_EmptyResults(t *testing.T) {
	ctx := context.Background()
	svc, db := newImport7906(t)
	seedImportTask(t, db, "task-empty", nil, 161)

	n, err := svc.ImportDiscoveredDevices(ctx, "task-empty", []string{"a", "b"}, "op")
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, int64(0), countImportDevices(t, db))
}

// TestImportDiscoveredDevices_MissingTask 任务不存在 → 保持既有文案。
func TestImportDiscoveredDevices_MissingTask(t *testing.T) {
	ctx := context.Background()
	svc, db := newImport7906(t)

	n, err := svc.ImportDiscoveredDevices(ctx, "no-such-task", nil, "op")
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.Contains(t, err.Error(), "发现任务不存在")
	assert.NotContains(t, err.Error(), "导入发现设备失败", "任务不存在是独立文案，不套导入失败前缀")
	assert.Equal(t, int64(0), countImportDevices(t, db))
}

// TestImportDiscoveredDevices_AtomicRollback 写入中途失败 → 整批回滚、零残留。
//
// 通过 gorm create callback 在 sys_network_device 的某次 INSERT 上注入错误：
// 用 2 条有效 IP，batch size 1000 会把两条合并成一条 multi-values INSERT，
// 因此注册 gorm:create 的 before callback，仅对目标表 AddError，使该批 INSERT
// 失败 → 事务回滚；断言 count=0、err 包含「导入发现设备失败」。
func TestImportDiscoveredDevices_AtomicRollback(t *testing.T) {
	ctx := context.Background()
	svc, db := newImport7906(t)
	seedImportTask(t, db, "task-rb", nil, 161)
	seedImportResult(t, db, "rb1", "task-rb", "10.142.7.1", models.DeviceDiscoveryResult{
		SysName: "rb-1", IsAlive: true,
	}, 1)
	seedImportResult(t, db, "rb2", "task-rb", "10.142.7.2", models.DeviceDiscoveryResult{
		SysName: "rb-2", IsAlive: true,
	}, 2)

	// newDB7906 返回实例级 callback manager（每个 gorm.Open 独立），
	// 在此注入仅影响本 db 句柄，不污染其它用例。
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register(
		"test142:force_network_device_failure",
		func(tx *gorm.DB) {
			if tx != nil && tx.Statement != nil && tx.Statement.Table == "sys_network_device" {
				_ = tx.AddError(errors.New("injected constraint conflict"))
			}
		}))

	n, err := svc.ImportDiscoveredDevices(ctx, "task-rb", nil, "op")
	require.Error(t, err)
	assert.Equal(t, 0, n, "失败时返回 0")
	assert.Contains(t, err.Error(), "导入发现设备失败")
	assert.Equal(t, int64(0), countImportDevices(t, db), "事务回滚后 sys_network_device 零残留")
}

// TestImportDiscoveredDevices_Idempotent 连调两次，第二次 count=0、总行数不变。
func TestImportDiscoveredDevices_Idempotent(t *testing.T) {
	ctx := context.Background()
	svc, db := newImport7906(t)
	seedImportTask(t, db, "task-idem", nil, 161)
	seedImportResult(t, db, "id1", "task-idem", "10.142.8.1", models.DeviceDiscoveryResult{
		SysName: "idem-1", IsAlive: true,
	}, 1)
	seedImportResult(t, db, "id2", "task-idem", "10.142.8.2", models.DeviceDiscoveryResult{
		SysName: "idem-2", IsAlive: true,
	}, 2)

	n1, err := svc.ImportDiscoveredDevices(ctx, "task-idem", nil, "op")
	require.NoError(t, err)
	assert.Equal(t, 2, n1)

	n2, err := svc.ImportDiscoveredDevices(ctx, "task-idem", nil, "op")
	require.NoError(t, err)
	assert.Equal(t, 0, n2, "第二次全部命中库内已存在 → 跳过")
	assert.Equal(t, int64(2), countImportDevices(t, db), "总行数不变")
}

// TestImportDiscoveredDevices_DoesNotDeleteResults 导入后结果表仍在（GetDiscoveryResults 不改）。
func TestImportDiscoveredDevices_DoesNotDeleteResults(t *testing.T) {
	ctx := context.Background()
	svc, db := newImport7906(t)
	seedImportTask(t, db, "task-keep", nil, 161)
	seedImportResult(t, db, "kp1", "task-keep", "10.142.9.1", models.DeviceDiscoveryResult{
		SysName: "keep-1", Vendor: "Huawei", Model: "S5700", IsAlive: true,
	}, 1)
	seedImportResult(t, db, "kp2", "task-keep", "10.142.9.2", models.DeviceDiscoveryResult{
		SysName: "keep-2", Vendor: "Cisco", Model: "C9300", IsAlive: true,
	}, 2)

	_, err := svc.ImportDiscoveredDevices(ctx, "task-keep", nil, "op")
	require.NoError(t, err)

	devices, err := svc.GetDiscoveryResults(ctx, "task-keep")
	require.NoError(t, err)
	require.Len(t, devices, 2, "导入不删结果表，明细仍可复查")
	assert.Equal(t, "10.142.9.1", devices[0].IPAddress)
	assert.Equal(t, "Huawei", devices[0].Vendor)
	assert.Equal(t, "S5700", devices[0].Model)
	assert.Equal(t, "10.142.9.2", devices[1].IPAddress)

	var resultCount int64
	require.NoError(t, db.Model(&models.DeviceDiscoveryResult{}).
		Where("discovery_id = ?", "task-keep").Count(&resultCount).Error)
	assert.Equal(t, int64(2), resultCount)
}
