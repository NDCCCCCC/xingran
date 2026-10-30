package operations

// workstation_physical_diagnosis_test.go — Phase 73 / #7（DBG-07）回归守护
//
// 根因（2026-10-27 实测，见 .planning/debug/workstation-3f130-physical-link-empty.md）：
//   工位 3f130 的 ws_user_id=NULL、active_info_points=1、ip_bound_to_valid_device=1、
//   mac_match_count=1 —— 数据链路全通，空因 = 工位未绑定 user_id。
//   旧实现只返回裸 `[]`，前端无法区分「未绑用户」与「真没设备」。
//
// 本文件锁定两条契约：
//  1. user_id 为空时，诊断返回 emptyReason = no_user_bound，而不是裸空数组；
//  2. 有数据时 emptyReason 为空（不得误报空因）。
//
// 测试在 SQLite 上真实调用 service（诊断实现刻意不用 PG 专属语法，
// 就是为了让这些用例能真跑，而不是只测字符串常量）。

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const testWSID = "00000000-0000-0000-0000-000000000130"

// newPhysicalTestDB 建独立内存库 + 物理链路 CTE 依赖的表（SQLite 方言）。
// 表结构按 PG 模型裁剪到诊断实际用到的列。
func newPhysicalTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	for _, ddl := range []string{
		`CREATE TABLE sys_workstation (
			id TEXT PRIMARY KEY, workstation_name TEXT, status INTEGER DEFAULT 0,
			user_id TEXT, deleted_at DATETIME)`,
		`CREATE TABLE ops_info_points (
			id TEXT PRIMARY KEY, name TEXT, workstation_id TEXT, device_id TEXT,
			port_id TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE sys_device_port_status (
			id TEXT PRIMARY KEY, device_id TEXT, interface_name TEXT)`,
		`CREATE TABLE sys_network_device (id TEXT PRIMARY KEY, device_name TEXT)`,
		`CREATE TABLE sys_device_mac_address (
			id TEXT PRIMARY KEY, device_id TEXT, mac_address TEXT, interface_name TEXT,
			collected_at DATETIME)`,
		`CREATE TABLE sys_device_mac_history (
			id TEXT PRIMARY KEY, device_id TEXT, device_name_snapshot TEXT, mac_address TEXT,
			interface_name TEXT, vlan_id INTEGER, event_type TEXT, first_seen DATETIME,
			last_seen DATETIME, collected_at DATETIME, created_at DATETIME)`,
		`CREATE TABLE ops_asset (
			id TEXT PRIMARY KEY, devicesn TEXT, device_model_name TEXT, device_type_name TEXT,
			mac1 TEXT, mac2 TEXT, machine_ip TEXT, nowuser_name TEXT, deleted_at DATETIME)`,
		`CREATE TABLE sys_ad_computer (
			id TEXT PRIMARY KEY, mac_address TEXT, serial_number TEXT, computer_name TEXT,
			ip_address TEXT, operating_system TEXT, updated_at DATETIME, deleted_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(ddl).Error, "建表失败: %s", ddl)
	}
	return db
}

// newDiagnosisTestService 构造一个「诊断专用」service：
//   - db 是真实 SQLite（诊断 SQL 全部用 GORM chainable，可双端跑）；
//   - physicalDevicesFetcher 用 SQLite 版链路召回替换 PG 专属 CTE
//     （GetPhysicalDevices 的 CTE 含 ::text / REGEXP_REPLACE，SQLite 无法执行）。
//
// 这样测的是**真实的诊断实现 + 真实的空因分类**，只有「CTE 方言」这一层被替换。
func newDiagnosisTestService(db *gorm.DB) *workstationDeviceService {
	svc := NewWorkstationDeviceService(db).(*workstationDeviceService)
	svc.physicalDevicesFetcher = func(ctx context.Context, wsID string) ([]*models.WorkstationDevice, error) {
		return sqlitePhysicalRecall(ctx, db, wsID)
	}
	return svc
}

// sqlitePhysicalRecall 是 GetPhysicalDevices CTE 的 SQLite 等价实现（仅测试用）：
// 按 信息点(active) → 端口存在 → 设备存在 → 端口上有 MAC 的顺序召回。
// 召回规则与生产 CTE 同构，只是方言换成 SQLite 支持的写法。
func sqlitePhysicalRecall(_ context.Context, db *gorm.DB, wsID string) ([]*models.WorkstationDevice, error) {
	type row struct {
		InterfaceName string
		InfoName      string
		DeviceID      string
		PortID        string
	}
	var rows []row
	if err := db.Raw(`
		SELECT port.interface_name AS interface_name,
		       ip.name            AS info_name,
		       ip.device_id       AS device_id,
		       port.id            AS port_id
		  FROM ops_info_points ip
		  JOIN sys_device_port_status port ON port.id = ip.port_id
		 WHERE ip.workstation_id = ? AND ip.deleted_at IS NULL AND ip.status = 0
		   AND EXISTS (SELECT 1 FROM sys_network_device d WHERE d.id = ip.device_id)
	`, wsID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]*models.WorkstationDevice, 0, len(rows))
	for i, r := range rows {
		// 端口上的 MAC（大小写/分隔符不敏感比对，对齐生产 CTE 的 norm_iface 归一）
		var macs []string
		if err := db.Raw(`
			SELECT mac_address FROM sys_device_mac_address
			 WHERE device_id = ? AND REPLACE(REPLACE(LOWER(interface_name),' ',''),'gigabitethernet','ge') =
			       REPLACE(REPLACE(LOWER(?),' ',''),'gigabitethernet','ge')
		`, r.DeviceID, r.InterfaceName).Scan(&macs).Error; err != nil {
			return nil, err
		}

		var mac *string
		var assetID *string
		if len(macs) > 0 {
			m := macs[0]
			mac = &m
			var ids []string
			norm := strings.ToLower(strings.NewReplacer(":", "", "-", "", ".", "").Replace(m))
			if err := db.Raw(`
				SELECT id FROM ops_asset
				 WHERE deleted_at IS NULL
				   AND (LOWER(REPLACE(REPLACE(REPLACE(COALESCE(mac1,''),':',''),'-',''),'.','')) = ?
				     OR LOWER(REPLACE(REPLACE(REPLACE(COALESCE(mac2,''),':',''),'-',''),'.','')) = ?)
			`, norm, norm).Scan(&ids).Error; err != nil {
				return nil, err
			}
			if len(ids) > 0 {
				assetID = &ids[0]
			}
		}

		infoName := r.InfoName
		dev := &models.WorkstationDevice{
			BaseModel:     models.BaseModel{ID: fmt.Sprintf("physical-%d", i)},
			WorkstationID: wsID,
			DeviceSource:  models.DeviceSourcePhysical,
			AssetID:       assetID,
			MACAddress:    mac,
		}
		if infoName != "" {
			dev.Description = &infoName
		}
		out = append(out, dev)
	}
	return out, nil
}

// seedWorkstation 插入工位。userID 传空串表示未绑定用户（模拟 3f130）。
func seedWorkstation(t *testing.T, db *gorm.DB, id, name string, userID *string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO sys_workstation (id, workstation_name, status, user_id, deleted_at)
		 VALUES (?, ?, 0, ?, NULL)`, id, name, userID,
	).Error)
}

// seedFullChain 建一条完整链路：信息点→端口→设备→MAC→资产。
// 返回使用的 MAC（大写冒号 canonical 格式，与生产库一致）。
func seedFullChain(t *testing.T, db *gorm.DB, wsID, mac string) {
	t.Helper()
	const (
		devID  = "dev-aaa"
		portID = "port-aaa"
		iface  = "GE2/6"
	)
	require.NoError(t, db.Exec(`INSERT INTO sys_network_device (id, device_name)
		VALUES (?, 'CX-WH-WH-04F-FL-RS8607E-01')`, devID).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_device_port_status (id, device_id, interface_name)
		VALUES (?, ?, ?)`, portID, devID, iface).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_info_points
		(id, name, workstation_id, device_id, port_id, status, deleted_at)
		VALUES ('ip-aaa', '3f130-A', ?, ?, ?, 0, NULL)`, wsID, devID, portID).Error)
	// MAC 走 canonical 大写冒号（生产库实测格式）
	require.NoError(t, db.Exec(`INSERT INTO sys_device_mac_address
		(id, device_id, mac_address, interface_name, collected_at)
		VALUES ('mac-aaa', ?, ?, ?, '2026-10-27 16:02:00')`, devID, mac, "GigabitEthernet "+iface[2:]).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset
		(id, devicesn, device_model_name, device_type_name, mac1, mac2, machine_ip,
		 nowuser_name, deleted_at)
		VALUES ('asset-aaa', 'SN-3F130', 'ThinkCentre M90q', 'PC', ?, NULL,
		 '10.62.1.130', '张三', NULL)`, mac).Error)
}

// TestGetPhysicalDevicesWithDiagnosis_NoUserBound 是 #7 的核心回归用例（对齐 3f130 实测）。
//
// 场景构造刻意复刻 3f130 的四项读数：
//   - ws_user_id = NULL      （H3 断点）
//   - active info_point = 1  （H2 排除）
//   - info_point→device 有效 （H1/H4 排除）
//   - 端口上有 MAC            （MAC 链路通）
//
// 断言：emptyReason = no_user_bound，而不是裸空数组 + 无解释。
func TestGetPhysicalDevicesWithDiagnosis_NoUserBound(t *testing.T) {
	db := newPhysicalTestDB(t)
	seedWorkstation(t, db, testWSID, "3f130", nil) // user_id = NULL
	seedFullChain(t, db, testWSID, "B0:22:7A:2E:4A:4F")

	svc := newDiagnosisTestService(db)
	result, err := svc.GetPhysicalDevicesWithDiagnosis(context.Background(), testWSID)
	require.NoError(t, err)
	require.NotNil(t, result)

	// 核心断言：未绑定用户必须可诊断，而不是静默空数组
	assert.Equal(t, PhysicalEmptyReasonNoUserBound, result.EmptyReason,
		"user_id 为空时必须返回 emptyReason=no_user_bound")
	assert.NotEmpty(t, result.EmptyMessage, "空因必须带中文可读结论")
	assert.Contains(t, result.EmptyMessage, "未绑定用户")

	// 关键：3f130 的实况是「MAC 链路通（mac_match_count=1）但未绑用户」
	// → 未绑定用户必须压过「有数据」，否则前端按用户维度展示时仍是无解释的空白
	assert.NotEmpty(t, result.Devices, "数据链路本身是通的（对齐 3f130 实测 mac_match_count=1）")
	hasUserWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "未绑定用户") {
			hasUserWarning = true
		}
	}
	assert.True(t, hasUserWarning, "未绑用户即使有数据也必须在 warnings 中可见")

	// 诊断计数必须与 3f130 实测一致
	assert.False(t, result.Diagnosis.UserBound, "UserBound 必须为 false")
	assert.Equal(t, 1, result.Diagnosis.InfoPointTotal, "active_info_points 口径应看到 1")
	assert.Equal(t, 1, result.Diagnosis.InfoPointActive)
	assert.Equal(t, 1, result.Diagnosis.InfoPointWithPort)
	assert.Equal(t, 1, result.Diagnosis.PortResolved)
	assert.Equal(t, 1, result.Diagnosis.DeviceResolved, "ip_bound_to_valid_device=1")

	// devices 字段必须与链路召回结果完全一致（不因加诊断而分叉）
	recall, err := sqlitePhysicalRecall(context.Background(), db, testWSID)
	require.NoError(t, err)
	assert.Equal(t, len(recall), result.Total, "Total 必须等于召回条数")
	assert.Equal(t, len(recall), len(result.Devices), "Devices 必须与召回结果逐条一致")
}

// TestGetPhysicalDevicesWithDiagnosis_HasDataNoReason 验证「有数据时不得误报空因」。
//
// 用户已绑定 + 链路通 → emptyReason 必须为空（JSON 中该字段省略）。
// 这一条是防止修复过度：不能把「未绑用户」当唯一结论硬塞给所有工位。
func TestGetPhysicalDevicesWithDiagnosis_HasDataNoReason(t *testing.T) {
	db := newPhysicalTestDB(t)
	uid := "user-zhangsan"
	seedWorkstation(t, db, testWSID, "3f130", &uid)
	seedFullChain(t, db, testWSID, "B0:22:7A:2E:4A:4F")

	svc := newDiagnosisTestService(db)
	result, err := svc.GetPhysicalDevicesWithDiagnosis(context.Background(), testWSID)
	require.NoError(t, err)
	require.NotNil(t, result)

	require.NotEmpty(t, result.Devices, "完整链路（用户+信息点+端口+MAC+资产）必须召回设备")
	assert.Equal(t, PhysicalEmptyReasonNone, result.EmptyReason,
		"有数据时 emptyReason 必须为空，不得误报空因")
	assert.Empty(t, result.EmptyMessage, "有数据时不输出空因文案")
	assert.Equal(t, len(result.Devices), result.Total)
	assert.True(t, result.Diagnosis.UserBound)
	assert.GreaterOrEqual(t, result.Diagnosis.MACMatched, 1, "MAC 段必须命中")
	for _, w := range result.Warnings {
		assert.NotContains(t, w, "未绑定用户", "已绑定用户的工位不应出现未绑用户告警")
	}
}

// TestGetPhysicalDevicesWithDiagnosis_EmptyReasonNeverEmptyWhenNoDevices 是不变量守护：
// 任何「0 台设备」的结果都必须带非空 emptyReason —— 这正是 #7 要根治的「静默空白」。
func TestGetPhysicalDevicesWithDiagnosis_EmptyReasonNeverEmptyWhenNoDevices(t *testing.T) {
	cases := []struct {
		name     string
		userID   *string
		wantCode string
	}{
		{"未绑定用户", nil, PhysicalEmptyReasonNoUserBound},
		{"已绑定用户但无信息点", diagStrPtr("user-x"), PhysicalEmptyReasonInfoPointMisbound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newPhysicalTestDB(t)
			seedWorkstation(t, db, testWSID, "3f130", tc.userID)
			// 不建任何信息点 → 0 台设备

			svc := newDiagnosisTestService(db)
			result, err := svc.GetPhysicalDevicesWithDiagnosis(context.Background(), testWSID)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, 0, result.Total)
			assert.Equal(t, tc.wantCode, result.EmptyReason)
			assert.NotEmpty(t, result.EmptyReason,
				"0 台设备时必须给出可诊断的空因，绝不返回「无解释的空白」")
			assert.NotEmpty(t, result.EmptyMessage)
		})
	}
}

// TestClassifyPhysicalEmptyReason_CoversAllFour 穷举四类空因的判定顺序与文案，
// 保证任何链路断点都有确定结论（不出现「落不到任何分支」的空白）。
// diagStrPtr 本地 helper(避免与其他测试文件的 ptr 冲突)
func diagStrPtr(s string) *string { return &s }

func TestClassifyPhysicalEmptyReason_CoversAllFour(t *testing.T) {
	const userBound = "user-1"

	cases := []struct {
		name       string
		diag       PhysicalChainDiagnosis
		deviceCnt  int
		wantReason string
	}{
		{
			name:       "①未绑定用户",
			diag:       PhysicalChainDiagnosis{WorkstationExists: true, UserBound: false},
			deviceCnt:  0,
			wantReason: PhysicalEmptyReasonNoUserBound,
		},
		{
			name:       "②无任何信息点",
			diag:       PhysicalChainDiagnosis{WorkstationExists: true, UserBound: true, InfoPointTotal: 0},
			deviceCnt:  0,
			wantReason: PhysicalEmptyReasonInfoPointMisbound,
		},
		{
			name: "②信息点均不可用(软删/停用)",
			diag: PhysicalChainDiagnosis{WorkstationExists: true, UserBound: true,
				InfoPointTotal: 2, InfoPointActive: 0},
			deviceCnt:  0,
			wantReason: PhysicalEmptyReasonInfoPointMisbound,
		},
		{
			name: "②信息点未绑端口",
			diag: PhysicalChainDiagnosis{WorkstationExists: true, UserBound: true,
				InfoPointTotal: 1, InfoPointActive: 1, InfoPointWithPort: 0},
			deviceCnt:  0,
			wantReason: PhysicalEmptyReasonInfoPointMisbound,
		},
		{
			name: "③端口不存在",
			diag: PhysicalChainDiagnosis{WorkstationExists: true, UserBound: true,
				InfoPointTotal: 1, InfoPointActive: 1, InfoPointWithPort: 1, PortResolved: 0},
			deviceCnt:  0,
			wantReason: PhysicalEmptyReasonDeviceOrMACMissing,
		},
		{
			name: "③设备 orphan",
			diag: PhysicalChainDiagnosis{WorkstationExists: true, UserBound: true,
				InfoPointTotal: 1, InfoPointActive: 1, InfoPointWithPort: 1,
				PortResolved: 1, DeviceResolved: 0},
			deviceCnt:  0,
			wantReason: PhysicalEmptyReasonDeviceOrMACMissing,
		},
		{
			name: "③端口与设备都在但无 MAC",
			diag: PhysicalChainDiagnosis{WorkstationExists: true, UserBound: true,
				InfoPointTotal: 1, InfoPointActive: 1, InfoPointWithPort: 1,
				PortResolved: 1, DeviceResolved: 1, MACMatched: 0},
			deviceCnt:  0,
			wantReason: PhysicalEmptyReasonDeviceOrMACMissing,
		},
		{
			name: "④正常有数据",
			diag: PhysicalChainDiagnosis{WorkstationExists: true, UserBound: true,
				InfoPointTotal: 1, InfoPointActive: 1, InfoPointWithPort: 1,
				PortResolved: 1, DeviceResolved: 1, MACMatched: 1},
			deviceCnt:  1,
			wantReason: PhysicalEmptyReasonHasDevices,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reason, message := classifyPhysicalEmptyReason(tc.diag, tc.deviceCnt)
			assert.Equal(t, tc.wantReason, reason)
			if reason == PhysicalEmptyReasonNone {
				assert.Empty(t, message, "正常态不应有文案")
			} else {
				assert.NotEmpty(t, message, "空因必须带可读结论")
			}
		})
	}

	// 契约：未绑定用户优先级最高 —— 即使能召回设备也报 no_user_bound。
	// 依据：3f130 实测 mac_match_count=1，前端仍空白；根因就是面板按用户维度呈现。
	// 若此处返回 ""，前端依旧无法解释空白，DBG-07 的需求（可区分空因）就没解决。
	reason, _ := classifyPhysicalEmptyReason(
		PhysicalChainDiagnosis{UserBound: false, MACMatched: 1}, 1)
	assert.Equal(t, PhysicalEmptyReasonNoUserBound, reason,
		"未绑定用户是最高优先级空因，即使有数据也必须报出")
	_ = userBound
}

// TestGetPhysicalDevices_LegacySignatureUnchanged 是向后兼容守护：
// 旧方法**签名与委托关系**不变 —— 仍返回裸切片、仍由同一实现提供数据。
//
// 说明：旧方法内部是 PG 专属 CTE（::text / REGEXP_REPLACE / DISTINCT ON），
// SQLite 无法执行，因此这里只做「编译期签名 + 入口参数校验 + 委托关系」三重守护；
// 链路真实性由既有 workstation_device_physical_test.go（physicalJoin 等价实现）覆盖。
func TestGetPhysicalDevices_LegacySignatureUnchanged(t *testing.T) {
	db := newPhysicalTestDB(t)
	seedWorkstation(t, db, testWSID, "3f130", nil)

	svc := NewWorkstationDeviceService(db)

	// ① 编译期签名：返回类型必须是 []*models.WorkstationDevice（不是诊断结构）
	var _ func(context.Context, string) ([]*models.WorkstationDevice, error) = svc.GetPhysicalDevices

	// ② 入口参数校验不变（空 ID → 参数错误，不触达 SQL）
	devices, err := svc.GetPhysicalDevices(context.Background(), "")
	require.Error(t, err, "空工位ID必须报参数错误")
	assert.Empty(t, devices)

	// ③ 诊断入口复用同一实现：注入 fetcher 后 Total 必须等于 fetcher 返回条数
	diagSvc := newDiagnosisTestService(db)
	result, err := diagSvc.GetPhysicalDevicesWithDiagnosis(context.Background(), testWSID)
	require.NoError(t, err)
	require.NotNil(t, result)
	recall, err := sqlitePhysicalRecall(context.Background(), db, testWSID)
	require.NoError(t, err)
	assert.Equal(t, len(recall), result.Total, "诊断的 Total 必须与召回一致（同一份数据源）")
}
