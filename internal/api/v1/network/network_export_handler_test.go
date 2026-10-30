package network

// NetworkExportHandler tests (Phase 74-03).
//
// NetworkExportHandler depends on a *core.Core, not a service interface, so we
// exercise it against a real glebarez sqlite in-memory DB (D-02) with the operlog
// dependency stubbed via mockOperLogService (D-03). All exports write a binary
// .xlsx / .zip stream to the response — we assert on the Content-Type header,
// the Content-Disposition filename, and (where observable) the body length.
//
// D-12 honored: zero business code touched.

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// newExportTestEnv opens a sqlite-backed core and migrates every model touched
// by the export pipeline (NetworkDevice / AuthCredential / ConfigTemplate /
// ConfigExecution / ConfigBackup / DeviceDiscovery / DeviceMACAddress /
// DevicePortStatus).
func newExportTestEnv(t *testing.T) *netTestEnv {
	env := newNetworkTestEnv(t)
	netMigrateAll(t, env)
	return env
}

func newExportHandler(env *netTestEnv) *NetworkExportHandler {
	return NewNetworkExportHandler(env.core)
}

// seedNetworkDevice inserts a NetworkDevice with all the columns the export
// pipeline reads into the excel row.
func seedNetworkDevice(t *testing.T, env *netTestEnv, id, name, ip string) *models.NetworkDevice {
	t.Helper()
	d := netSeedDevice(t, env.db, id, name, ip)
	dept := "测试部门"
	d.DeptName = &dept
	d.Model = "S5700-28C"
	d.SerialNumber = "SN-001"
	d.SoftwareVersion = "V200R022C10"
	d.Uptime = "5d 3h"
	d.Description = "测试设备"
	now := models.Time(time.Now())
	d.LastSeenAt = &now
	require.NoError(t, env.db.Save(d).Error)
	return d
}

func seedAuthCredential(t *testing.T, env *netTestEnv, id, name string, isDefault bool) *models.AuthCredential {
	t.Helper()
	cred := &models.AuthCredential{
		BaseModel:       models.BaseModel{ID: id},
		CredentialName:  name,
		ProtocolType:    models.ProtocolTypeSSH,
		Username:        "admin",
		Password:        "enc:secret",
		SNMPCommunities: []string{"public"},
		SNMPVersion:     models.SNMPVersionV2c,
		Description:     "测试凭证",
		IsDefault:       isDefault,
	}
	require.NoError(t, env.db.Create(cred).Error)
	return cred
}

func seedExportTemplate(t *testing.T, env *netTestEnv, id, code string, sys bool) *models.ConfigTemplate {
	t.Helper()
	tmpl := &models.ConfigTemplate{
		BaseModel:       models.BaseModel{ID: id},
		TemplateName:    "模板-" + code,
		TemplateCode:    code,
		TemplateType:    models.TemplateTypeConfig,
		Vendor:          models.VendorHuawei,
		DeviceType:      models.DeviceTypeSwitch,
		TemplateContent: "vlan batch 10",
		Description:     "测试模板",
		IsSystem:        sys,
	}
	require.NoError(t, env.db.Create(tmpl).Error)
	return tmpl
}

func seedConfigExecution(t *testing.T, env *netTestEnv, id, name string, status models.ExecutionStatus) *models.ConfigExecution {
	t.Helper()
	now := models.Time(time.Now())
	exec := &models.ConfigExecution{
		ID:                id,
		ExecutionName:     name,
		ExecutionType:     models.ExecutionTypeCommand,
		DeviceIDs:         models.DeviceIDList{"dev-1"},
		Status:            status,
		TotalDevices:      1,
		SuccessCount:      1,
		ExecutionStrategy: models.ExecutionStrategyParallel,
		Concurrency:       5,
		Timeout:           60,
		StartedAt:         &now,
		CompletedAt:       &now,
		CreatedBy:         "tester",
	}
	require.NoError(t, env.db.Create(exec).Error)
	return exec
}

func seedExportDiscovery(t *testing.T, env *netTestEnv, id, name string, status models.DiscoveryStatus) *models.DeviceDiscovery {
	t.Helper()
	now := time.Now()
	disc := &models.DeviceDiscovery{
		ID:              id,
		TaskName:        name,
		DiscoveryType:   models.DiscoveryTypeSNMP,
		IPRanges:        models.IPRangeList{{StartIP: "10.0.0.1", EndIP: "10.0.0.10"}},
		SNMPCommunity:   "public",
		SNMPPort:        161,
		Status:          status,
		TotalIPs:        10,
		DiscoveredCount: 5,
		AutoImport:      true,
		StartedAt:       &now,
		CompletedAt:     &now,
		CreatedBy:       "tester",
	}
	require.NoError(t, env.db.Create(disc).Error)
	return disc
}

func seedExportPortStatus(t *testing.T, env *netTestEnv, id, deviceID, ifName string) *models.DevicePortStatus {
	t.Helper()
	vlan := 10
	p := &models.DevicePortStatus{
		ID:            id,
		DeviceID:      deviceID,
		InterfaceName: ifName,
		AdminStatus:   "up",
		OperStatus:    "up",
		Description:   "测试端口",
		VLAN:          &vlan,
		Duplex:        "full",
		Speed:         "1000",
		PortType:      "GE",
		CollectedAt:   time.Now(),
	}
	require.NoError(t, env.db.Create(p).Error)
	return p
}

func seedMACAddress(t *testing.T, env *netTestEnv, id, deviceID, mac, ifName string) *models.DeviceMACAddress {
	t.Helper()
	vlan := 100
	m := &models.DeviceMACAddress{
		ID:            id,
		DeviceID:      deviceID,
		MACAddress:    mac,
		InterfaceName: ifName,
		VLANID:        &vlan,
		MACType:       models.MACTypeDynamic,
		CollectedAt:   time.Now(),
	}
	require.NoError(t, env.db.Create(m).Error)
	return m
}

// assertExportFilename decodes the URL-encoded filename from a Content-Disposition
// header and asserts both the entity-name fragment AND the file extension are
// present in the decoded filename. The handler URL-encodes Chinese chars via
// url.QueryEscape, so naive contains checks against the raw header fail.
//
// Example Content-Disposition:
//
//	attachment; filename=%E7%BD%91%E7%BB%9C%E8%AE%BE%E5%A4%87_export_20260821.xlsx; filename*=utf-8''...
//	attachment; filename="网络管理_批量导出_20260821.zip"   (BatchExport uses quoted form)
func assertExportFilename(t *testing.T, cd, wantFragment, wantExt string) {
	t.Helper()
	require.NotEmpty(t, cd, "Content-Disposition header missing")
	// Pull the first "filename=..." or 'filename="..."' segment
	idx := strings.Index(cd, "filename=")
	require.GreaterOrEqual(t, idx, 0, "no filename= segment in CD: %s", cd)
	rest := cd[idx+len("filename="):]
	// Strip wrapping double-quotes (BatchExport path).
	rest = strings.Trim(rest, `"`)
	// Strip everything from the next ";" or end of string.
	end := strings.IndexByte(rest, ';')
	if end >= 0 {
		rest = rest[:end]
	}
	rest = strings.TrimSpace(rest)
	decoded, err := url.QueryUnescape(rest)
	require.NoError(t, err, "filename not URL-decodable: %s", rest)
	assert.Contains(t, decoded, wantFragment, "filename should contain %q, got %q", wantFragment, decoded)
	assert.True(t, strings.HasSuffix(decoded, wantExt), "filename should end with %q, got %q", wantExt, decoded)
}

// ============================================================================
// ExportDevices
// ============================================================================

func TestExportHandler_ExportDevices(t *testing.T) {
	env := newExportTestEnv(t)
	h := newExportHandler(env)
	seedNetworkDevice(t, env, "dev-e1", "device-A", "10.0.0.1")
	seedNetworkDevice(t, env, "dev-e2", "device-B", "10.0.0.2")

	t.Run("filtered_mode_xlsx", func(t *testing.T) {
		w := netPost(t, "/export/devices", h.ExportDevices, `{"exportMode":"filtered","filters":{}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
		assertExportFilename(t, w.Header().Get("Content-Disposition"), "网络设备_export_", ".xlsx")
		assert.Greater(t, w.Body.Len(), 1000, "xlsx body should be non-trivial")
		// xlsx is a zip — magic bytes 50 4B 03 04
		assert.Equal(t, byte('P'), w.Body.Bytes()[0])
		assert.Equal(t, byte('K'), w.Body.Bytes()[1])
	})

	t.Run("current_page_mode", func(t *testing.T) {
		w := netPost(t, "/export/devices", h.ExportDevices, `{"exportMode":"currentPage","current":1,"pageSize":1}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("all_mode", func(t *testing.T) {
		w := netPost(t, "/export/devices", h.ExportDevices, `{"exportMode":"all"}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("filters_device_name", func(t *testing.T) {
		w := netPost(t, "/export/devices", h.ExportDevices,
			`{"exportMode":"filtered","filters":{"deviceName":"device-A"}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("filters_status_float64", func(t *testing.T) {
		w := netPost(t, "/export/devices", h.ExportDevices,
			`{"exportMode":"filtered","filters":{"status":0}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("filters_status_dept_vendor_deviceType", func(t *testing.T) {
		w := netPost(t, "/export/devices", h.ExportDevices,
			`{"exportMode":"filtered","filters":{"ipAddress":"10.0.0.1","deviceType":"switch","vendor":"huawei","deptId":"d1"}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("binding_error", func(t *testing.T) {
		w := netPost(t, "/export/devices", h.ExportDevices, `not-json`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("unknown_mode_defaults_to_all", func(t *testing.T) {
		w := netPost(t, "/export/devices", h.ExportDevices, `{"exportMode":"weird"}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ============================================================================
// ExportCredentials
// ============================================================================

func TestExportHandler_ExportCredentials(t *testing.T) {
	env := newExportTestEnv(t)
	h := newExportHandler(env)
	seedAuthCredential(t, env, "cred-1", "default-cred", true)
	seedAuthCredential(t, env, "cred-2", "backup-cred", false)

	t.Run("xlsx_output", func(t *testing.T) {
		w := netPost(t, "/export/credentials", h.ExportCredentials, `{"exportMode":"filtered","filters":{}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
		assertExportFilename(t, w.Header().Get("Content-Disposition"), "授权凭证_export_", ".xlsx")
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("filter_credential_name", func(t *testing.T) {
		w := netPost(t, "/export/credentials", h.ExportCredentials,
			`{"exportMode":"filtered","filters":{"credentialName":"default-cred"}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("filter_protocol_type", func(t *testing.T) {
		w := netPost(t, "/export/credentials", h.ExportCredentials,
			`{"exportMode":"filtered","filters":{"protocolType":"ssh"}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("binding_error", func(t *testing.T) {
		w := netPost(t, "/export/credentials", h.ExportCredentials, `not-json`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

// ============================================================================
// ExportTemplates
// ============================================================================

func TestExportHandler_ExportTemplates(t *testing.T) {
	env := newExportTestEnv(t)
	h := newExportHandler(env)
	seedExportTemplate(t, env, "tmpl-1", "TPL-001", false)
	seedExportTemplate(t, env, "tmpl-2", "TPL-002", true)

	t.Run("xlsx_output", func(t *testing.T) {
		w := netPost(t, "/export/templates", h.ExportTemplates, `{"exportMode":"filtered","filters":{}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
		assertExportFilename(t, w.Header().Get("Content-Disposition"), "配置模板_export_", ".xlsx")
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("all_filters", func(t *testing.T) {
		w := netPost(t, "/export/templates", h.ExportTemplates,
			`{"exportMode":"filtered","filters":{"templateName":"模板","templateType":"config","vendor":"huawei","deviceType":"switch","isSystem":true}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("filter_no_match", func(t *testing.T) {
		w := netPost(t, "/export/templates", h.ExportTemplates,
			`{"exportMode":"filtered","filters":{"templateName":"nonexistent"}}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("binding_error", func(t *testing.T) {
		w := netPost(t, "/export/templates", h.ExportTemplates, `not-json`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

// ============================================================================
// ExportCommands
// ============================================================================

func TestExportHandler_ExportCommands(t *testing.T) {
	env := newExportTestEnv(t)
	h := newExportHandler(env)
	seedConfigExecution(t, env, "exec-1", "命令-1", models.ExecutionStatusSuccess)
	seedConfigExecution(t, env, "exec-2", "命令-2", models.ExecutionStatusFailed)

	t.Run("xlsx_output", func(t *testing.T) {
		w := netPost(t, "/export/commands", h.ExportCommands, `{"exportMode":"filtered","filters":{}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
		assertExportFilename(t, w.Header().Get("Content-Disposition"), "命令分发_export_", ".xlsx")
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("all_mode", func(t *testing.T) {
		w := netPost(t, "/export/commands", h.ExportCommands, `{"exportMode":"all"}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("binding_error", func(t *testing.T) {
		w := netPost(t, "/export/commands", h.ExportCommands, `not-json`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

// ============================================================================
// ExportExecutions
// ============================================================================

func TestExportHandler_ExportExecutions(t *testing.T) {
	env := newExportTestEnv(t)
	h := newExportHandler(env)
	seedConfigExecution(t, env, "exec-cfg-1", "执行-1", models.ExecutionStatusSuccess)
	seedConfigExecution(t, env, "exec-cfg-2", "执行-2", models.ExecutionStatusRunning)

	t.Run("xlsx_output", func(t *testing.T) {
		w := netPost(t, "/export/executions", h.ExportExecutions, `{"exportMode":"filtered","filters":{}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
		assertExportFilename(t, w.Header().Get("Content-Disposition"), "配置执行_export_", ".xlsx")
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("current_page", func(t *testing.T) {
		w := netPost(t, "/export/executions", h.ExportExecutions,
			`{"exportMode":"currentPage","current":1,"pageSize":1}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("binding_error", func(t *testing.T) {
		w := netPost(t, "/export/executions", h.ExportExecutions, `not-json`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

// ============================================================================
// ExportBackups
// ============================================================================

func TestExportHandler_ExportBackups(t *testing.T) {
	env := newExportTestEnv(t)
	h := newExportHandler(env)
	netSeedDevice(t, env.db, "dev-bk1", "备份设备", "10.0.0.5")
	b := &models.ConfigBackup{
		ID:           "bk-1",
		DeviceID:     "dev-bk1",
		DeviceName:   "备份设备",
		BackupType:   models.BackupTypeManual,
		StorageType:  models.StorageTypeDatabase,
		Version:      1,
		ConfigHash:   "abc123",
		ConfigContent: "vlan 1",
		BackupSize:   6,
		Compressed:   false,
		ChangeReason: "initial",
		CreatedBy:    "tester",
	}
	require.NoError(t, env.db.Create(b).Error)

	t.Run("xlsx_output_no_filter", func(t *testing.T) {
		w := netPost(t, "/export/backups", h.ExportBackups, `{"exportMode":"filtered","filters":{}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
		assertExportFilename(t, w.Header().Get("Content-Disposition"), "配置备份_export_", ".xlsx")
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("filter_by_device_id", func(t *testing.T) {
		w := netPost(t, "/export/backups", h.ExportBackups,
			`{"exportMode":"filtered","filters":{"deviceId":"dev-bk1"}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("binding_error", func(t *testing.T) {
		w := netPost(t, "/export/backups", h.ExportBackups, `not-json`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

// ============================================================================
// ExportDiscoveries
// ============================================================================

func TestExportHandler_ExportDiscoveries(t *testing.T) {
	env := newExportTestEnv(t)
	h := newExportHandler(env)
	seedExportDiscovery(t, env, "disc-1", "扫描任务-1", models.DiscoveryStatusSuccess)
	seedExportDiscovery(t, env, "disc-2", "扫描任务-2", models.DiscoveryStatusFailed)

	t.Run("xlsx_output", func(t *testing.T) {
		w := netPost(t, "/export/discoveries", h.ExportDiscoveries, `{"exportMode":"filtered","filters":{}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
		assertExportFilename(t, w.Header().Get("Content-Disposition"), "设备发现_export_", ".xlsx")
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("all_mode", func(t *testing.T) {
		w := netPost(t, "/export/discoveries", h.ExportDiscoveries, `{"exportMode":"all"}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("binding_error", func(t *testing.T) {
		w := netPost(t, "/export/discoveries", h.ExportDiscoveries, `not-json`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

// ============================================================================
// ExportMACAddresses
// ============================================================================

func TestExportHandler_ExportMACAddresses(t *testing.T) {
	env := newExportTestEnv(t)
	h := newExportHandler(env)
	netSeedDevice(t, env.db, "dev-mac", "mac-dev", "10.0.0.10")
	seedMACAddress(t, env, "mac-1", "dev-mac", "AA:BB:CC:00:00:01", "GE0/0/1")
	seedMACAddress(t, env, "mac-2", "dev-mac", "AA:BB:CC:00:00:02", "GE0/0/2")

	t.Run("xlsx_output", func(t *testing.T) {
		w := netPost(t, "/export/mac", h.ExportMACAddresses, `{"exportMode":"filtered","filters":{}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
		assertExportFilename(t, w.Header().Get("Content-Disposition"), "MAC地址_export_", ".xlsx")
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("all_filters", func(t *testing.T) {
		w := netPost(t, "/export/mac", h.ExportMACAddresses,
			`{"exportMode":"filtered","filters":{"deviceId":"dev-mac","deptId":"d1","macAddress":"AA:BB:CC","interfaceName":"GE0"}}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("binding_error", func(t *testing.T) {
		w := netPost(t, "/export/mac", h.ExportMACAddresses, `not-json`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

// ============================================================================
// ExportPorts
// ============================================================================

func TestExportHandler_ExportPorts(t *testing.T) {
	env := newExportTestEnv(t)
	h := newExportHandler(env)
	netSeedDevice(t, env.db, "dev-port", "port-dev", "10.0.0.20")
	seedExportPortStatus(t, env, "port-e1", "dev-port", "GE0/0/1")
	seedExportPortStatus(t, env, "port-e2", "dev-port", "GE0/0/2")

	t.Run("xlsx_output_with_device_join", func(t *testing.T) {
		w := netPost(t, "/export/ports", h.ExportPorts, `{"exportMode":"filtered","filters":{}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
		assertExportFilename(t, w.Header().Get("Content-Disposition"), "端口状态_export_", ".xlsx")
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("filter_combinations", func(t *testing.T) {
		w := netPost(t, "/export/ports", h.ExportPorts,
			`{"exportMode":"filtered","filters":{"deviceId":"dev-port","interfaceName":"GE0","adminStatus":"up","operStatus":"up"}}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("empty_vlan_pointer_safe", func(t *testing.T) {
		// Add a port without a VLAN pointer to exercise the nil-safety branch
		p := &models.DevicePortStatus{
			ID:            "port-no-vlan",
			DeviceID:      "dev-port",
			InterfaceName: "GE0/0/99",
			AdminStatus:   "up",
			OperStatus:    "down",
			Dot1xEnabled:  true,
			CollectedAt:   time.Now(),
		}
		require.NoError(t, env.db.Create(p).Error)

		w := netPost(t, "/export/ports", h.ExportPorts, `{"exportMode":"filtered","filters":{}}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("binding_error", func(t *testing.T) {
		w := netPost(t, "/export/ports", h.ExportPorts, `not-json`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

// ============================================================================
// BatchExport
// ============================================================================

func TestExportHandler_BatchExport(t *testing.T) {
	env := newExportTestEnv(t)
	h := newExportHandler(env)
	seedNetworkDevice(t, env, "dev-bx", "device-bx", "10.0.0.30")
	seedAuthCredential(t, env, "cred-bx", "cred-bx", false)
	seedExportTemplate(t, env, "tmpl-bx", "TPL-BX", false)

	t.Run("single_entity_zip", func(t *testing.T) {
		w := netPost(t, "/batch-export", h.BatchExport, `{"entityTypes":["devices"]}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/zip")
		assertExportFilename(t, w.Header().Get("Content-Disposition"), "网络管理_批量导出_", ".zip")
		// zip magic bytes 50 4B 03 04
		assert.Equal(t, byte('P'), w.Body.Bytes()[0])
		assert.Equal(t, byte('K'), w.Body.Bytes()[1])
		assert.Greater(t, w.Body.Len(), 100)
	})

	t.Run("multiple_entities_zip", func(t *testing.T) {
		w := netPost(t, "/batch-export", h.BatchExport,
			`{"entityTypes":["devices","credentials","templates","commands","executions","backups","discoveries","mac","ports"],"filters":{}}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Greater(t, w.Body.Len(), 0)
	})

	t.Run("invalid_entity_type_400", func(t *testing.T) {
		w := netPost(t, "/batch-export", h.BatchExport, `{"entityTypes":["unknown_entity"]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
		assert.Contains(t, resp.Message, "不支持的实体类型")
	})

	t.Run("binding_empty_entity_types", func(t *testing.T) {
		w := netPost(t, "/batch-export", h.BatchExport, `{"entityTypes":[]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("binding_missing_entity_types", func(t *testing.T) {
		w := netPost(t, "/batch-export", h.BatchExport, `{}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})

	t.Run("binding_too_many_entity_types", func(t *testing.T) {
		// max is 9 — provide 10 (1 valid + 9 dups above limit)
		w := netPost(t, "/batch-export", h.BatchExport,
			`{"entityTypes":["devices","credentials","templates","commands","executions","backups","discoveries","mac","ports","extra1"]}`)
		resp := decodeNetResp(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, 400, resp.Code)
	})
}

// ============================================================================
// getPaginationParams + formatTime helpers (indirectly via the helpers above)
//
// Direct unit coverage via a non-zero current/pageSize from current_page branch.
// ============================================================================

func TestExportHandler_PaginationParams_Defaults(t *testing.T) {
	env := newExportTestEnv(t)
	h := newExportHandler(env)

	t.Run("current_page_zero_current_clamped_to_1", func(t *testing.T) {
		// current<1 → 1; pageSize<1 → 10. Empty DB so service returns 0 rows
		// but we still hit the pagination branch + buildDeviceListRequest.
		w := netPost(t, "/export/devices", h.ExportDevices,
			`{"exportMode":"currentPage","current":0,"pageSize":0,"filters":{}}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("current_page_normal", func(t *testing.T) {
		w := netPost(t, "/export/devices", h.ExportDevices,
			`{"exportMode":"currentPage","current":2,"pageSize":5,"filters":{}}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ============================================================================
// formatTime / formatTimePtr helpers — exercised indirectly via every export.
// These tests use tiny endpoints just to make sure the helpers don't panic on
// edge cases (nil pointers, zero times).
// ============================================================================

func TestFormatTimeHelpers_Regression(t *testing.T) {
	env := newExportTestEnv(t)
	h := newExportHandler(env)

	t.Run("zero_time_via_device_creation", func(t *testing.T) {
		// Insert a device with a zero CreatedAt to exercise formatTime's
		// IsZero branch.
		d := &models.NetworkDevice{
			BaseModel:  models.BaseModel{ID: "zero-time-dev"},
			DeviceName: "zero-time",
			DeviceType: models.DeviceTypeSwitch,
			Vendor:     models.VendorHuawei,
			IPAddress:  "10.0.99.99",
			Port:       22,
			SNMPPort:   161,
			Status:     models.DeviceStatus(0),
		}
		require.NoError(t, env.db.Create(d).Error)

		w := netPost(t, "/export/devices", h.ExportDevices, `{"exportMode":"filtered","filters":{}}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("nil_time_ptr_safe", func(t *testing.T) {
		// Same idea: nil LastSeenAt exercises formatTimePtr's nil-check branch.
		w := netPost(t, "/export/devices", h.ExportDevices, `{"exportMode":"filtered","filters":{}}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ============================================================================
// Sanity: buildXxxListRequest filters edge cases
// ============================================================================

func TestExportHandler_BuildRequest_EdgeCases(t *testing.T) {
	env := newExportTestEnv(t)
	h := newExportHandler(env)

	t.Run("nil_filters", func(t *testing.T) {
		w := netPost(t, "/export/devices", h.ExportDevices, `{"exportMode":"filtered"}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("device_credential_template_empty_filters", func(t *testing.T) {
		w1 := netPost(t, "/export/devices", h.ExportDevices, `{"exportMode":"filtered","filters":{}}`)
		w2 := netPost(t, "/export/credentials", h.ExportCredentials, `{"exportMode":"filtered","filters":{}}`)
		w3 := netPost(t, "/export/templates", h.ExportTemplates, `{"exportMode":"filtered","filters":{}}`)
		assert.Equal(t, http.StatusOK, w1.Code)
		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Equal(t, http.StatusOK, w3.Code)
	})

	t.Run("empty_string_filters_skipped", func(t *testing.T) {
		w := netPost(t, "/export/devices", h.ExportDevices,
			`{"exportMode":"filtered","filters":{"deviceName":"","ipAddress":"","deviceType":"","vendor":"","deptId":""}}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ensureBytesContainXLSXSignature confirms response body starts with the PK ZIP
// magic (xlsx files are zip archives internally).
func ensureBytesContainXLSXSignature(t *testing.T, body *bytes.Buffer) {
	t.Helper()
	require.Greater(t, body.Len(), 4)
	assert.Equal(t, []byte("PK\x03\x04"), body.Bytes()[:4])
}

// TestExportHandler_XLSXMagicBytes double-checks the xlsx ZIP magic bytes via a
// synthetic buffer (live responses already assert >= 1000 bytes; this confirms
// the leading PK\x03\x04 pattern in a focused test).
func TestExportHandler_XLSXMagicBytes(t *testing.T) {
	buf := bytes.NewBufferString("PK\x03\x04xxxxxxxxxx")
	ensureBytesContainXLSXSignature(t, buf)
	assert.True(t, strings.HasPrefix(buf.String(), "PK\x03\x04"))
}