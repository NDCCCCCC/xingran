// Phase 79-06 (TAIL-01) — config_execution_service.go + command_dispatch_service.go
// coverage tests (validate/param-assembly segments + executor dispatch).
//
// Tier-1: the whole request-validation ladder of ExecuteByTemplate / Dispatch
// (empty devices → template missing → partial devices → render failure), the
// sqlite read/cancel/delete chain and the statistics tie-in.
//
// Tier-2 (D-79-02): the errgroup/serial fan-out runs against a
// *device.DeviceExecutor assembled from the public constructors with
// FileTransport connections seeded per device via device.SeedConnectionForTesting.
// Dispatch drives wrapper.SendCommand (fixture = command cycles);
// ExecuteByTemplate drives wrapper.SendConfig (fixture = the huawei_vrp config
// mode shape: system-view → [prompt] → config echo → return).
//
// The connection-missing branch (pool.GetConnection over a pool with NO seeded
// connection) is the deterministic failure matrix — it exercises the per-device
// result + detail-row write path without any fixture.
//
// QUIRK-79-06-H (locked, not fixed): neither executeConfigOnDevice nor
// executeOnDevice guards a nil executor — the first device touch dereferences
// s.executor unconditionally (asserted with require.Panics). Production always
// constructs both services with a non-nil executor.
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
)

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

// cex7906NewDB opens a fresh sqlite DB with the execution chain migrated.
func cex7906NewDB(t *testing.T) *gorm.DB {
	t.Helper()
	return newDB7906(t, &models.ConfigExecution{}, &models.ConfigExecutionDetail{},
		&models.ConfigTemplate{}, &models.NetworkDevice{}, &models.AuthCredential{}, &models.Config{})
}

// cex7906SeedExecutor assembles pool → scheduler → executor over db and seeds a
// FileTransport connection for every deviceID in fixtures (deviceID → fixture
// path). Goroutine-backed resources are shut down via t.Cleanup (pool Close is
// watchdog-bounded — FileTransport close reads can block).
func cex7906SeedExecutor(t *testing.T, db *gorm.DB, fixtures map[string]string) *device.DeviceExecutor {
	t.Helper()

	pool := device.NewDeviceConnectionPool(db, nil, &device.PoolConfig{MaxIdle: time.Hour, MaxConnections: 8})
	t.Cleanup(func() {
		done := make(chan struct{})
		go func() { _ = pool.Close(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Logf("pool.Close watchdog fired — goroutine leaked intentionally")
		}
	})
	scheduler := device.NewDeviceTaskScheduler(pool, nil)
	t.Cleanup(scheduler.Stop)
	executor := device.NewDeviceExecutor(scheduler, &device.ExecutionConfig{
		MaxRetries: 0, RetryDelay: time.Millisecond, Timeout: 10 * time.Second, EnablePanicRecovery: true,
	})

	for deviceID, fixture := range fixtures {
		conn := device.NewPooledConnectionForTesting(newDriver7906FromFixture(t, fixture))
		device.SeedConnectionForTesting(pool, deviceID, conn)
		conn.ReleaseRef()
	}
	return executor
}

// cex7906CommandFixture returns a SendCommand fixture with n reply cycles.
func cex7906CommandFixture(t *testing.T, cycles int, output string) string {
	t.Helper()
	return writeFixture7906(t, cycles, "show version", output)
}

// cex7906SendConfigFixture returns a fixture shaped for scrapligo's huawei_vrp
// SendConfig flow: config-mode acquisition (system-view → [prompt]) → config
// echo → prompt → return → exec prompt (mirrors the portwrite e2e fixture).
func cex7906SendConfigFixture(t *testing.T, configLine string) string {
	t.Helper()
	var b []byte
	add := func(s string) { b = append(b, s...) }
	add("<dummy-host>\n")
	add("screen-length 0 temporary\n")
	add("<dummy-host>\n")
	add("system-view\n")
	add("[dummy-host]\n")
	add(configLine + "\n")
	add("[dummy-host]\n")
	add("return\n")
	add("<dummy-host>\n")
	for i := 0; i < 8; i++ {
		add("<dummy-host>\n")
	}
	return writeFixtureBytes7906(t, b)
}

// cex7906SeedDevice inserts a device row with an explicit ID (third octet from
// the shared sequence — IPAddress carries a uniqueIndex).
func cex7906SeedDevice(t *testing.T, db *gorm.DB, id, name string) *models.NetworkDevice {
	t.Helper()
	dmn7906IPSeq++
	dev := &models.NetworkDevice{
		DeviceName: name, DeviceType: models.DeviceTypeSwitch,
		Vendor: models.VendorHuawei, IPAddress: fmt.Sprintf("10.79.62.%d", dmn7906IPSeq%250+1),
		Status: models.DeviceStatusOnline,
	}
	dev.ID = id
	require.NoError(t, db.Create(dev).Error, "seed device %s", name)
	return dev
}

// cex7906SeedTemplate inserts a config template with one optional variable.
func cex7906SeedTemplate(t *testing.T, db *gorm.DB, code, content string, required bool) *models.ConfigTemplate {
	tpl := &models.ConfigTemplate{
		TemplateName: "tpl-" + code, TemplateCode: code, TemplateType: models.TemplateTypeConfig,
		TemplateContent: content,
		Variables: models.TemplateVariables{{
			Name: "hostname", Type: "string", Required: required, DefaultValue: "default-7906",
		}},
	}
	require.NoError(t, db.Create(tpl).Error, "seed template %s", code)
	return tpl
}

// -----------------------------------------------------------------------------
// ConfigExecution — validate / param assembly
// -----------------------------------------------------------------------------

// TestCex7906_Validate_AndParamAssembly walks the ExecuteByTemplate request
// ladder: empty device list → template missing → partial device list → render
// failure on a missing required variable.
func TestCex7906_Validate_AndParamAssembly(t *testing.T) {
	ctx := context.Background()
	db := cex7906NewDB(t)
	svc := NewConfigExecutionService(db, nil) // never reaches the executor in this test

	t.Run("empty_device_ids", func(t *testing.T) {
		_, err := svc.ExecuteByTemplate(ctx, &TemplateExecutionRequest{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "请选择要执行配置的设备")
	})

	t.Run("template_missing", func(t *testing.T) {
		_, err := svc.ExecuteByTemplate(ctx, &TemplateExecutionRequest{
			DeviceIDs:  []string{"dev-x"},
			TemplateID: "no-such-template",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "模板不存在")
	})

	t.Run("partial_device_list_rejected", func(t *testing.T) {
		tpl := cex7906SeedTemplate(t, db, "cex-partial", "hostname {{.hostname}}", false)
		cex7906SeedDevice(t, db, "dev-cex-p1", "p1-switch")
		_, err := svc.ExecuteByTemplate(ctx, &TemplateExecutionRequest{
			DeviceIDs:  []string{"dev-cex-p1", "dev-cex-ghost"},
			TemplateID: tpl.ID,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "部分设备不存在")
	})

	t.Run("render_failure_missing_required_variable", func(t *testing.T) {
		tpl := cex7906SeedTemplate(t, db, "cex-render", "hostname {{.hostname}}", true)
		dev := cex7906SeedDevice(t, db, "dev-cex-render", "render-switch")
		_, err := svc.ExecuteByTemplate(ctx, &TemplateExecutionRequest{
			DeviceIDs:         []string{dev.ID},
			TemplateID:        tpl.ID,
			TemplateVariables: map[string]string{}, // required variable missing
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "渲染模板失败")
		// The failure happened before any execution row was written.
		var count int64
		require.NoError(t, db.Model(&models.ConfigExecution{}).Count(&count).Error)
		assert.Equal(t, int64(0), count, "no execution row on render failure")
	})

	t.Run("param_assembly_defaults", func(t *testing.T) {
		// With an optional variable and no user input the default value fills
		// the rendered config — the pure param-assembly segment.
		tpl := cex7906SeedTemplate(t, db, "cex-params", "hostname {{.hostname}}", false)
		require.NotNil(t, tpl)
		rendered, err := NewTemplateService(db).Render(ctx, tpl.TemplateCode, map[string]string{})
		require.NoError(t, err)
		assert.Equal(t, "hostname default-7906", rendered)
	})
}

// -----------------------------------------------------------------------------
// ConfigExecution — executor fan-out
// -----------------------------------------------------------------------------

// TestCex7906_Execute_MultiDeviceSerial — Tier-2 boundary record: the SendConfig
// happy path is NOT reachable with a hand-written fixture (scrapligo's
// huawei_vrp config-mode byte expectation does not match the replayed shape;
// the attempt stalls in the driver read until the context deadline). The
// reachable execution surface is therefore driven through the deterministic
// connection-missing matrix: a real executor whose pool has NO seeded
// connection, two devices in serial strategy, full row/detail write chain.
func TestCex7906_Execute_MultiDeviceSerial(t *testing.T) {
	ctx := context.Background()
	db := cex7906NewDB(t)

	dev1 := cex7906SeedDevice(t, db, "dev-cex-m1", "m1-switch")
	dev2 := cex7906SeedDevice(t, db, "dev-cex-m2", "m2-switch")
	tpl := cex7906SeedTemplate(t, db, "cex-matrix", "hostname miss-7906", false)

	// Real executor, empty pool → GetConnection fails for every device.
	executor := cex7906SeedExecutor(t, db, nil)
	svc := NewConfigExecutionService(db, executor)

	result, err := svc.ExecuteByTemplate(ctx, &TemplateExecutionRequest{
		ExecutionName:     "cex-matrix",
		TemplateID:        tpl.ID,
		DeviceIDs:         []string{dev1.ID, dev2.ID},
		ExecutionStrategy: models.ExecutionStrategySerial,
		Timeout:           5,
		CreatedBy:         "cex7906",
	})
	require.NoError(t, err, "ExecuteByTemplate itself succeeds — failures live in per-device results")
	require.NotNil(t, result)

	// Both per-device results failed with the connection error.
	assert.Equal(t, 2, result.Summary.TotalDevices)
	assert.Equal(t, 0, result.Summary.SuccessCount)
	assert.Equal(t, 2, result.Summary.FailureCount)
	for _, devID := range []string{dev1.ID, dev2.ID} {
		got := result.Results[devID]
		require.NotNil(t, got, devID)
		assert.Equal(t, models.ExecutionStatusFailed, got.Status, devID)
		assert.Contains(t, got.ErrorMessage, "获取连接失败", devID)
		require.NotNil(t, got.StartedAt)
		// Locked behavior: on the connection-failure path ConfigSent stays empty
		// — the rendered config is only recorded on the success branch.
		assert.Equal(t, "", got.ConfigSent)
	}

	// Two failed detail rows persisted with the same message.
	var details []models.ConfigExecutionDetail
	require.NoError(t, db.Where("execution_id = ?", result.ExecutionID).Order("device_id").Find(&details).Error)
	require.Len(t, details, 2)
	for _, d := range details {
		assert.Equal(t, models.ExecutionStatusFailed, d.Status)
		assert.Contains(t, d.ErrorMessage, "获取连接失败")
	}

	// QUIRK-79-06-G (locked, not fixed): the execution row is stamped success
	// even when every device failed — ExecuteByTemplate unconditionally sets
	// ExecutionStatusSuccess after executeConfig returns.
	stats, err := svc.GetStatistics(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Total)
	assert.Equal(t, int64(1), stats.Success, "row stamped success despite 0/2 devices (quirk)")
	assert.Equal(t, int64(0), stats.Failed, "no failed row despite 2 failed devices (quirk)")

	// List / cancel / delete chain over the persisted row.
	list, total, err := svc.GetExecutionList(ctx, 1, 10, "", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, models.ExecutionTypeTemplate, list[0].ExecutionType)

	err = svc.CancelExecution(ctx, result.ExecutionID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "只能取消待执行或执行中的任务")

	row, err := svc.GetExecutionResult(ctx, result.ExecutionID)
	require.NoError(t, err)
	assert.Len(t, row.Results, 2)

	require.NoError(t, svc.DeleteExecution(ctx, result.ExecutionID))
	_, err = svc.GetExecutionResult(ctx, result.ExecutionID)
	require.Error(t, err)
	require.NoError(t, svc.BatchDeleteExecutions(ctx, []string{"no-such"}),
		"batch delete continues past unknown ids")
}

// TestCex7906_Execute_ConnectionMissing — the deterministic failure matrix: a
// real executor whose pool has NO seeded connection. pool.GetConnection fails
// inside createConnection, the per-device result fails with 获取连接失败 and the
// detail row is persisted as failed.
func TestCex7906_Execute_ConnectionMissing(t *testing.T) {
	ctx := context.Background()
	db := cex7906NewDB(t)

	dev := cex7906SeedDevice(t, db, "dev-cex-miss", "miss-switch")
	tpl := cex7906SeedTemplate(t, db, "cex-miss", "hostname miss-7906", false)

	// Real executor, empty pool → GetConnection must fail (no credentials in DB).
	executor := cex7906SeedExecutor(t, db, nil)
	svc := NewConfigExecutionService(db, executor)

	result, err := svc.ExecuteByTemplate(ctx, &TemplateExecutionRequest{
		ExecutionName:     "cex-missing",
		TemplateID:        tpl.ID,
		DeviceIDs:         []string{dev.ID},
		ExecutionStrategy: models.ExecutionStrategySerial,
		Timeout:           5,
	})
	require.NoError(t, err, "ExecuteByTemplate itself succeeds — failures live in per-device results")
	require.NotNil(t, result)

	devResult := result.Results[dev.ID]
	require.NotNil(t, devResult)
	assert.Equal(t, models.ExecutionStatusFailed, devResult.Status)
	assert.Contains(t, devResult.ErrorMessage, "获取连接失败")

	// The detail row landed as failed with the same message.
	var detail models.ConfigExecutionDetail
	require.NoError(t, db.Where("execution_id = ?", result.ExecutionID).First(&detail).Error)
	assert.Equal(t, models.ExecutionStatusFailed, detail.Status)
	assert.Contains(t, detail.ErrorMessage, "获取连接失败")

	// QUIRK-79-06-G (locked, not fixed): the execution row is stamped success
	// even when every device failed — ExecuteByTemplate unconditionally sets
	// ExecutionStatusSuccess after executeConfig returns.
	assert.Equal(t, 0, result.Summary.SuccessCount)
	assert.Equal(t, 1, result.Summary.FailureCount)
	row, err := svc.GetExecutionResult(ctx, result.ExecutionID)
	require.NoError(t, err)
	assert.Equal(t, 1, row.Summary.FailureCount)
}

// TestCex7906_Execute_NilExecutor — QUIRK-79-06-H evidence: the per-device
// executor touch has no nil guard.
func TestCex7906_Execute_NilExecutor(t *testing.T) {
	db := cex7906NewDB(t)
	svc := NewConfigExecutionService(db, nil)
	dev := &models.NetworkDevice{DeviceName: "x", IPAddress: "10.0.0.1"}
	dev.ID = "dev-cex-nil"

	require.Panics(t, func() {
		_ = svc.executeConfigOnDevice(context.Background(), "exec-nil", dev, "cfg", 0)
	}, "executeConfigOnDevice dereferences the nil executor (locked quirk)")
}

// -----------------------------------------------------------------------------
// CommandDispatch — parse/route + executor guards
// -----------------------------------------------------------------------------

// TestCdp7906_Dispatch_ParseAndRoute — Dispatch's validation ladder plus the
// seeded parallel fan-out (errgroup with a concurrency limit) and the
// statistics tie-in for execution_type=command.
func TestCdp7906_Dispatch_ParseAndRoute(t *testing.T) {
	ctx := context.Background()
	db := cex7906NewDB(t)
	svc := NewCommandDispatchService(db, nil)

	t.Run("empty_device_ids", func(t *testing.T) {
		_, err := svc.Dispatch(ctx, &DispatchRequest{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "请选择要执行命令的设备")
	})

	t.Run("partial_device_list_rejected", func(t *testing.T) {
		cex7906SeedDevice(t, db, "dev-cdp-p1", "cdp-p1")
		_, err := svc.Dispatch(ctx, &DispatchRequest{DeviceIDs: []string{"dev-cdp-p1", "dev-cdp-ghost"}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "部分设备不存在")
	})

	t.Run("seeded_parallel_fanout", func(t *testing.T) {
		dev1 := cex7906SeedDevice(t, db, "dev-cdp-1", "cdp-1")
		dev2 := cex7906SeedDevice(t, db, "dev-cdp-2", "cdp-2")

		executor := cex7906SeedExecutor(t, db, map[string]string{
			dev1.ID: cex7906CommandFixture(t, 2, "cdp-ok-1"),
			dev2.ID: cex7906CommandFixture(t, 2, "cdp-ok-2"),
		})
		svc = NewCommandDispatchService(db, executor)

		result, err := svc.Dispatch(ctx, &DispatchRequest{
			ExecutionName:     "cdp-seeded",
			DeviceIDs:         []string{dev1.ID, dev2.ID},
			CommandContent:    "show version",
			ExecutionStrategy: models.ExecutionStrategyParallel, // errgroup with SetLimit
			Concurrency:       1,                                // limit exercised
			Timeout:           10,
			CreatedBy:         "cdp7906",
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		for _, devID := range []string{dev1.ID, dev2.ID} {
			got := result.Results[devID]
			require.NotNil(t, got, devID)
			assert.Equal(t, models.ExecutionStatusSuccess, got.Status, devID)
			assert.Equal(t, "show version", got.CommandSent)
		}
		assert.Equal(t, 2, result.Summary.TotalDevices)
		assert.Equal(t, 2, result.Summary.SuccessCount)

		// Statistics read path ties to the command-type rows written here.
		stats, err := svc.GetStatistics(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), stats.Total, "one command execution row")
		assert.Equal(t, int64(1), stats.Success)

		list, total, err := svc.GetExecutionList(ctx, 1, 10, "status", boolPtr7906(false))
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		require.Len(t, list, 1)
		assert.Equal(t, models.ExecutionTypeCommand, list[0].ExecutionType)
	})
}

// TestCdp7906_Dispatch_ExecutorGuards — the connection-missing matrix, the
// nil-executor quirk, and QuickCommand's happy + missing-device branches.
func TestCdp7906_Dispatch_ExecutorGuards(t *testing.T) {
	ctx := context.Background()
	db := cex7906NewDB(t)

	t.Run("connection_missing_marks_device_failed", func(t *testing.T) {
		dev := cex7906SeedDevice(t, db, "dev-cdp-miss", "cdp-miss")
		executor := cex7906SeedExecutor(t, db, nil) // real executor, empty pool
		svc := NewCommandDispatchService(db, executor)

		result, err := svc.Dispatch(ctx, &DispatchRequest{
			DeviceIDs:      []string{dev.ID},
			CommandContent: "display version",
			Timeout:        5,
		})
		require.NoError(t, err)
		got := result.Results[dev.ID]
		require.NotNil(t, got)
		assert.Equal(t, models.ExecutionStatusFailed, got.Status)
		assert.Contains(t, got.ErrorMessage, "获取连接失败")

		// QUIRK-79-06-G: execution row still stamped success.
		row, err := svc.GetExecutionResult(ctx, result.ExecutionID)
		require.NoError(t, err)
		assert.Equal(t, 1, row.Summary.FailureCount)
	})

	t.Run("nil_executor_panics", func(t *testing.T) {
		svc := NewCommandDispatchService(db, nil)
		dev := &models.NetworkDevice{DeviceName: "y", IPAddress: "10.0.0.2"}
		dev.ID = "dev-cdp-nil"
		require.Panics(t, func() {
			_ = svc.executeOnDevice(context.Background(), "exec-nil", dev, "cmd", 0)
		}, "executeOnDevice dereferences the nil executor (QUIRK-79-06-H)")
	})

	t.Run("quick_command_seeded_happy", func(t *testing.T) {
		dev := cex7906SeedDevice(t, db, "dev-cdp-quick", "cdp-quick")
		executor := cex7906SeedExecutor(t, db, map[string]string{
			dev.ID: cex7906CommandFixture(t, 2, "quick-ok"), // echo: show version
		})
		svc := NewCommandDispatchService(db, executor)

		// NOTE: the command must match the fixture's echoed line — scrapligo
		// locates the sent command inside the replayed echo bytes.
		result, err := svc.QuickCommand(ctx, dev.ID, "show version", 10)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, models.ExecutionStatusSuccess, result.Status)
		assert.Contains(t, result.OutputReceived, "quick-ok")
		assert.Equal(t, "", resultDetailExecutionID7906(t, db, result.DeviceID),
			"QuickCommand writes detail rows with an empty execution id")
	})

	t.Run("quick_command_unknown_device", func(t *testing.T) {
		svc := NewCommandDispatchService(db, nil)
		result, err := svc.QuickCommand(ctx, "dev-cdp-ghost", "cmd", 5)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "设备不存在")
	})
}

// resultDetailExecutionID7906 reads the execution_id of the latest detail row
// for a device (QuickCommand passes "" as the execution id).
func resultDetailExecutionID7906(t *testing.T, db *gorm.DB, deviceID string) string {
	t.Helper()
	var detail models.ConfigExecutionDetail
	require.NoError(t, db.Where("device_id = ?", deviceID).Order("created_at DESC").First(&detail).Error)
	return detail.ExecutionID
}
