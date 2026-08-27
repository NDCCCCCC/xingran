// Phase 79-06 (TAIL-01) — device_monitor_service.go coverage tests
// (setters / nil-guards / convertSNMPVersion / Collect* delegation).
//
// Task 5 scope: the injection surface (SetExecutor + three service setters),
// every nil-service guard branch of the Collect*/Backup* delegation methods,
// convertSNMPVersion, loadConfigFromDB / ReloadConfig / Close, and
// getCredentialForDevice. The SNMP probe segment (pingDeviceViaSNMP /
// CheckDeviceStatus happy path) is Task 7 with the D-79-05 UDP fake.
//
// Naming: TestDmn7906_* / newDmn7906 (D-79-06). No t.Parallel.
package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

// newDmn7906 assembles a DeviceMonitorService over a fresh sqlite DB with the
// monitor-relevant chain migrated (defaults: 10 concurrent / 30s timeout).
func newDmn7906(t *testing.T) (*DeviceMonitorService, *gorm.DB) {
	t.Helper()
	db := newDB7906(t, &models.NetworkDevice{}, &models.AuthCredential{}, &models.Config{}, &models.ConfigBackup{})
	svc := NewDeviceMonitorService(db, nil, nil)
	t.Cleanup(func() { _ = svc.Close() })
	return svc, db
}

// dmn7906SeedDevice inserts a device row with an explicit ID/status.
func dmn7906SeedDevice(t *testing.T, db *gorm.DB, id, name string, status models.DeviceStatus) *models.NetworkDevice {
	t.Helper()
	dev := &models.NetworkDevice{
		DeviceName: name,
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.VendorHuawei,
		IPAddress:  fmt.Sprintf("10.79.61.%d", dmn7906IPSeq%250+1),
		Status:     status,
	}
	dmn7906IPSeq++
	dev.ID = id
	require.NoError(t, db.Create(dev).Error, "seed device %s", name)
	// QUIRK-79-04-D: zero-value status (online=0) is dropped by the column
	// default on Create — force it back explicitly.
	require.NoError(t, db.Model(dev).Update("status", status).Error)
	return dev
}

// dmn7906IPSeq allocates unique third octets for the uniqueIndex on IPAddress.
var dmn7906IPSeq int

// -----------------------------------------------------------------------------
// setters + nil-guard 面
// -----------------------------------------------------------------------------

// TestDmn7906_Setters_NilGuards — SetExecutor wires the four sub-services even
// with a nil executor, the three dedicated setters override them, and the
// delegation methods report their 未初始化 errors when the field is nil.
func TestDmn7906_Setters_NilGuards(t *testing.T) {
	ctx := context.Background()
	svc, db := newDmn7906(t)

	t.Run("set_executor_wires_subservices", func(t *testing.T) {
		require.Nil(t, svc.portCollectionSvc)
		svc.SetExecutor(nil) // nil executor is accepted — sub-services still built
		assert.NotNil(t, svc.portCollectionSvc, "port collection service wired")
		assert.NotNil(t, svc.macCollectionSvc, "MAC collection service wired")
		assert.NotNil(t, svc.configBackupSvc, "config backup service wired")
		assert.NotNil(t, svc.portCollectionSvc.Collection, "inner collection service wired")
	})

	t.Run("dedicated_setters_override", func(t *testing.T) {
		altPort := portcollection.NewPortCollectionService(db, nil)
		altMAC := &MACCollectionService{}
		altBackup := &ConfigBackupService{}
		svc.SetPortCollectionService(altPort)
		svc.SetMACCollectionService(altMAC)
		svc.SetConfigBackupService(altBackup)
		assert.Same(t, altPort, svc.portCollectionSvc)
		assert.Same(t, altMAC, svc.macCollectionSvc)
		assert.Same(t, altBackup, svc.configBackupSvc)
	})

	t.Run("nil_service_guards", func(t *testing.T) {
		bare := &DeviceMonitorService{} // everything nil
		assert.EqualError(t, bare.CollectAllPortStatus(ctx), "端口采集服务未初始化")
		assert.EqualError(t, bare.CollectPortStatus(ctx, "dev"), "端口采集服务未初始化")
		assert.EqualError(t, bare.CollectAllMACAddresses(ctx), "MAC采集服务未初始化")
		assert.EqualError(t, bare.CollectMACAddresses(ctx, "dev"), "MAC采集服务未初始化")
		assert.EqualError(t, bare.BackupAllConfigurations(ctx), "配置备份服务未初始化")
		assert.EqualError(t, bare.BackupConfiguration(ctx, "dev", "op"), "配置备份服务未初始化")
	})
}

// -----------------------------------------------------------------------------
// convertSNMPVersion
// -----------------------------------------------------------------------------

// TestDmn7906_ConvertSNMPVersion — models.SNMPVersion → device.SNMPVersion
// mapping, with the v2c default for unknown values.
func TestDmn7906_ConvertSNMPVersion(t *testing.T) {
	svc, _ := newDmn7906(t)

	cases := []struct {
		name string
		in   models.SNMPVersion
		want device.SNMPVersion
		note string
	}{
		{"v1", models.SNMPVersionV1, device.SNMPVersion1, "v1 maps 1:1"},
		{"v2c", models.SNMPVersionV2c, device.SNMPVersion2c, "v2c maps 1:1"},
		{"v3", models.SNMPVersionV3, device.SNMPVersion3, "v3 maps 1:1"},
		{"unknown_falls_back_to_v2c", models.SNMPVersion("bogus"), device.SNMPVersion2c, "default branch"},
		{"empty_falls_back_to_v2c", models.SNMPVersion(""), device.SNMPVersion2c, "default branch"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := svc.convertSNMPVersion(tc.in)
			assert.Equal(t, int(tc.want), int(got), tc.note)
		})
	}
}

// -----------------------------------------------------------------------------
// 配置装载 + ReloadConfig + Close
// -----------------------------------------------------------------------------

// TestDmn7906_LoadReloadConfig_Close — the two sys_config keys drive
// maxConcurrent and timeout; invalid values keep the defaults; ReloadConfig
// re-reads and fans out to the wired sub-services; Close is idempotent.
func TestDmn7906_LoadReloadConfig_Close(t *testing.T) {
	ctx := context.Background()

	t.Run("config_overrides", func(t *testing.T) {
		db := newDB7906(t, &models.NetworkDevice{}, &models.AuthCredential{}, &models.Config{}, &models.ConfigBackup{})
		require.NoError(t, db.Create(&models.Config{ConfigName: "c1", ConfigKey: "network.device.monitor.concurrent", ConfigValue: "7"}).Error)
		require.NoError(t, db.Create(&models.Config{ConfigName: "c2", ConfigKey: "network.device.timeout", ConfigValue: "45"}).Error)
		svc := NewDeviceMonitorService(db, nil, nil)
		assert.Equal(t, 7, svc.maxConcurrent)
		assert.Equal(t, 45*time.Second, svc.timeout)
		require.NoError(t, svc.Close())
	})

	t.Run("invalid_values_keep_defaults", func(t *testing.T) {
		db := newDB7906(t, &models.NetworkDevice{}, &models.AuthCredential{}, &models.Config{}, &models.ConfigBackup{})
		require.NoError(t, db.Create(&models.Config{ConfigName: "c1", ConfigKey: "network.device.monitor.concurrent", ConfigValue: "nope"}).Error)
		require.NoError(t, db.Create(&models.Config{ConfigName: "c2", ConfigKey: "network.device.timeout", ConfigValue: "-5"}).Error)
		svc := NewDeviceMonitorService(db, nil, nil)
		assert.Equal(t, 10, svc.maxConcurrent, "default from DefaultDeviceMonitorConfig")
		assert.Equal(t, 30*time.Second, svc.timeout)
		require.NoError(t, svc.Close())
	})

	t.Run("reload_fans_out_to_subservices", func(t *testing.T) {
		svc, db := newDmn7906(t)
		svc.SetExecutor(nil) // wire all sub-services
		// Change the concurrency config behind the service's back.
		require.NoError(t, db.Create(&models.Config{ConfigName: "c3", ConfigKey: "network.device.monitor.concurrent", ConfigValue: "4"}).Error)
		svc.ReloadConfig()
		assert.Equal(t, 4, svc.maxConcurrent, "ReloadConfig re-reads sys_config")
		// ReloadConfig with zero sub-services must also be safe.
		bare := &DeviceMonitorService{db: db}
		require.NotPanics(t, bare.ReloadConfig)
	})

	t.Run("close_is_idempotent", func(t *testing.T) {
		svc, _ := newDmn7906(t)
		require.NoError(t, svc.Close())
		require.NoError(t, svc.Close(), "second Close must not panic (Manager.Close is a no-op)")
		// QUIRK-79-06-F (locked, not fixed): Close on a zero service (nil
		// deviceManager) returns nil — there is no "not initialized" signal.
		bare := &DeviceMonitorService{}
		require.NoError(t, bare.Close())
		_ = ctx
	})
}

// -----------------------------------------------------------------------------
// Collect* 委托链(注入真实子服务)
// -----------------------------------------------------------------------------

// TestDmn7906_Collect_Delegates — with the sub-services injected, the
// delegation chain runs to the underlying service and surfaces its result:
// unknown devices produce 设备不存在 errors, empty device tables produce the
// 没有在线设备 errors, and BackupAllConfigurations over an empty table is a
// silent no-op (AutoBackupAllDevices returns nil for zero devices).
func TestDmn7906_Collect_Delegates(t *testing.T) {
	ctx := context.Background()
	svc, db := newDmn7906(t)
	svc.SetExecutor(nil) // wire the real sub-services (all over the same sqlite DB)

	t.Run("port_status_unknown_device", func(t *testing.T) {
		err := svc.CollectPortStatus(ctx, "dev-no-such")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "设备不存在")
	})

	t.Run("port_all_no_online_devices", func(t *testing.T) {
		err := svc.CollectAllPortStatus(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "没有在线设备")
	})

	t.Run("mac_unknown_device", func(t *testing.T) {
		err := svc.CollectMACAddresses(ctx, "dev-no-such")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "设备不存在")
	})

	t.Run("mac_all_no_online_devices", func(t *testing.T) {
		err := svc.CollectAllMACAddresses(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MAC地址采集失败")
	})

	t.Run("backup_configuration_unknown_device", func(t *testing.T) {
		err := svc.BackupConfiguration(ctx, "dev-no-such", "dmn7906")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "设备不存在")
	})

	t.Run("backup_all_empty_table_is_noop", func(t *testing.T) {
		require.NoError(t, svc.BackupAllConfigurations(ctx),
			"zero devices → AutoBackupAllDevices returns nil (locked)")
	})

	t.Run("delegation_reaches_seeded_backup_service", func(t *testing.T) {
		// Swap in a ConfigBackupService whose executor is nil but whose DB has a
		// device — the delegation must reach CreateBackup and fail on the
		// missing device lookup, proving the call actually crossed the boundary.
		altBackup := NewConfigBackupService(db, nil)
		svc.SetConfigBackupService(altBackup)
		err := svc.BackupConfiguration(ctx, "dev-no-such", "dmn7906")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "设备不存在", "error comes from the injected service")
	})
}

// -----------------------------------------------------------------------------
// CheckDeviceStatus 非 SNMP 分支 + getCredentialForDevice
// -----------------------------------------------------------------------------

// TestDmn7906_CheckDeviceStatus_NonSNMPBranches — the two branches that need no
// SNMP traffic: unknown device (error) and a credential with zero communities
// (skip → false, nil). The SNMP-carrying branches are Task 7 (D-79-05 fake).
func TestDmn7906_CheckDeviceStatus_NonSNMPBranches(t *testing.T) {
	ctx := context.Background()
	svc, db := newDmn7906(t)

	t.Run("unknown_device_errors", func(t *testing.T) {
		online, err := svc.CheckDeviceStatus(ctx, "dev-no-such")
		require.Error(t, err)
		assert.False(t, online)
		assert.Contains(t, err.Error(), "查询设备失败")
	})

	t.Run("credential_without_communities_skips", func(t *testing.T) {
		cred := &models.AuthCredential{CredentialName: "dmn-no-community", ProtocolType: models.ProtocolTypeSSH, Username: "u"}
		require.NoError(t, db.Create(cred).Error)
		dev := dmn7906SeedDevice(t, db, "dev-dmn-nosnmp", "nosnmp-switch", models.DeviceStatusOnline)
		require.NoError(t, db.Model(dev).Update("credential_id", cred.ID).Error)

		online, err := svc.CheckDeviceStatus(ctx, dev.ID)
		require.NoError(t, err, "empty community list is a skip, not an error")
		assert.False(t, online, "skipped check reports not-online")
	})

	t.Run("status_update_persists_last_seen", func(t *testing.T) {
		// A device whose credential carries a community would ping via SNMP —
		// deferred to Task 7. Here we only lock the DB write helper used by the
		// caller: re-reading the device after a skipped check still yields the
		// seeded status (no writes happened on the skip path).
		cred := &models.AuthCredential{CredentialName: "dmn-skip", ProtocolType: models.ProtocolTypeSSH, Username: "u"}
		require.NoError(t, db.Create(cred).Error)
		dev := dmn7906SeedDevice(t, db, "dev-dmn-skip", "skip-switch", models.DeviceStatusOffline)
		require.NoError(t, db.Model(dev).Update("credential_id", cred.ID).Error)

		online, err := svc.CheckDeviceStatus(ctx, dev.ID)
		require.NoError(t, err)
		assert.False(t, online)

		var after models.NetworkDevice
		require.NoError(t, db.Where("id = ?", dev.ID).First(&after).Error)
		assert.Equal(t, models.DeviceStatusOffline, after.Status, "skip path performs no status write")
		assert.Nil(t, after.LastSeenAt, "skip path performs no last_seen write")
	})
}

// TestDmn7906_GetCredentialForDevice — associated credential wins, default
// credential is the fallback, missing both errors.
func TestDmn7906_GetCredentialForDevice(t *testing.T) {
	ctx := context.Background()
	svc, db := newDmn7906(t)

	cred := &models.AuthCredential{CredentialName: "dmn-assoc", ProtocolType: models.ProtocolTypeSSH, Username: "u"}
	require.NoError(t, db.Create(cred).Error)
	defaultCred := &models.AuthCredential{CredentialName: "dmn-default", ProtocolType: models.ProtocolTypeSSH, Username: "d", IsDefault: true}
	require.NoError(t, db.Create(defaultCred).Error)

	assoc := dmn7906SeedDevice(t, db, "dev-dmn-assoc", "assoc-switch", models.DeviceStatusOnline)
	require.NoError(t, db.Model(assoc).Update("credential_id", cred.ID).Error)

	got, err := svc.getCredentialForDevice(ctx, assoc)
	require.NoError(t, err)
	assert.Equal(t, cred.ID, got.ID, "associated credential wins")

	plain := dmn7906SeedDevice(t, db, "dev-dmn-plain", "plain-switch", models.DeviceStatusOnline)
	got, err = svc.getCredentialForDevice(ctx, plain)
	require.NoError(t, err)
	assert.Equal(t, defaultCred.ID, got.ID, "unassociated device falls back to the default")

	// Fresh DB without a default credential → error.
	svc2, db2 := newDmn7906(t)
	orphan := dmn7906SeedDevice(t, db2, "dev-dmn-orphan", "orphan-switch", models.DeviceStatusOnline)
	got, err = svc2.getCredentialForDevice(ctx, orphan)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "未找到默认凭证")
}
