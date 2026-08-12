package component_collector

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupWriterTestDB 建立内存 SQLite,包含 ops_asset + sys_data_reconciliation 最小列集。
// 复用 asset_listfilter_test 的 schema 思路,只列 ops_asset_writer / reconciliation_emitter 真正读写的列。
func setupWriterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "open sqlite")
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_asset (
			id TEXT PRIMARY KEY,
			devicesn TEXT,
			device_model_name TEXT,
			parent_asset_id TEXT,
			source_device_id TEXT,
			component_type TEXT,
			component_slot TEXT,
			deleted_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error, "create ops_asset")
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_data_reconciliation (
			id TEXT PRIMARY KEY,
			asset_id TEXT NOT NULL,
			conflict_type TEXT NOT NULL,
			recon_category TEXT,
			severity TEXT NOT NULL,
			raw_snapshot TEXT NOT NULL,
			detected_at DATETIME NOT NULL,
			resolved_at DATETIME,
			deleted_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error, "create sys_data_reconciliation")
	return db
}

// insertAsset inserts a row with given id + devicesn + (optional) component_type.
// parent_asset_id/source_device_id/component_slot default NULL — writer is UPDATE-only
// so the test verifies they were NULL before Write() and populated after.
func insertAsset(t *testing.T, db *gorm.DB, id, deviceSN, componentType string) {
	t.Helper()
	now := time.Now().Format("2006-01-02 15:04:05")
	var ct interface{}
	if componentType != "" {
		ct = componentType
	}
	require.NoError(t, db.Exec(
		`INSERT INTO ops_asset (id, devicesn, component_type, created_at) VALUES (?, ?, ?, ?)`,
		id, deviceSN, ct, now,
	).Error, "insert ops_asset")
}

// noopRecorder is a minimal spy implementing operlog.Recorder.
type noopRecorder struct {
	calls int
	last  recCallArgs
}

type recCallArgs struct {
	module        string
	businessType  int
	method        string
	requestMethod string
	operatorName  *string
}

func (r *noopRecorder) RecordAsync(db *gorm.DB, title string, businessType int, method, requestMethod, operUrl string,
	operatorName, operatorNickname, deptName *string, operIP *string, operParam, jsonResult, errorMsg *string, status int, costTime int64) {
	r.calls++
	r.last = recCallArgs{
		module:        title,
		businessType:  businessType,
		method:        method,
		requestMethod: requestMethod,
		operatorName:  operatorName,
	}
}

// TestOpsAssetWriterHitUpdatesFourColumns REQ-48-05 hit path:
// 组件 SN 在 ops_asset 命中 → UPDATE 4 列(parent/source/type/slot),hitCount=1,调一次 operlog。
func TestOpsAssetWriterHitUpdatesFourColumns(t *testing.T) {
	db := setupWriterTestDB(t)
	insertAsset(t, db, "asset-card-1", "CARD-SN-001", "card") // existing row, parent/source/slot 全 NULL

	svc := &fakeAssetLookup{db: db}
	rec := &noopRecorder{}
	w := NewOpsAssetWriter(db, svc, rec)

	res, err := w.Write(context.Background(), "switch-1", "parent-uuid-1", []Component{
		{ComponentType: "card", Slot: "Slot 1", SerialNumber: "CARD-SN-001", Model: "M8600E-X"},
	})
	require.NoError(t, err)
	require.Equal(t, 1, res.HitCount)
	require.Equal(t, 0, res.MissCount)

	// Verify 4 columns updated
	type row struct {
		ParentAssetID  *string
		SourceDeviceID *string
		ComponentType  *string
		ComponentSlot  *string
	}
	var r1 row
	require.NoError(t, db.Table("ops_asset").Where("id = ?", "asset-card-1").Scan(&r1).Error)
	require.NotNil(t, r1.ParentAssetID, "parent_asset_id written")
	require.Equal(t, "parent-uuid-1", *r1.ParentAssetID)
	require.NotNil(t, r1.SourceDeviceID, "source_device_id written")
	require.Equal(t, "switch-1", *r1.SourceDeviceID)
	require.NotNil(t, r1.ComponentType)
	require.Equal(t, "card", *r1.ComponentType)
	require.NotNil(t, r1.ComponentSlot)
	require.Equal(t, "Slot 1", *r1.ComponentSlot)

	// operlog called once
	require.Equal(t, 1, rec.calls, "operlog.RecordBackground called once per hit")
	require.Equal(t, "资产管理", rec.last.module)
	require.Equal(t, 14, rec.last.businessType, "OperTypeSync=14")
	require.Equal(t, "BACKGROUND", rec.last.method, "cron path method=BACKGROUND")
	if rec.last.operatorName == nil {
		t.Fatal("operatorName must be non-nil (system-cron)")
	}
	require.Equal(t, "system-cron", *rec.last.operatorName)
}

// TestOpsAssetWriterMissEmitsAnomaly REQ-48-06 + REQ-48-05 miss path:
// 组件 SN 在 ops_asset 没找到 → emit sys_data_reconciliation 一行,
// conflict_type=F, recon_category=component_serial, missCount=1。
func TestOpsAssetWriterMissEmitsAnomaly(t *testing.T) {
	db := setupWriterTestDB(t)
	// 父交换机行存在(让 emit 走"用父交换机 id 作为 asset_id" 路径)
	insertAsset(t, db, "asset-switch-1", "SWITCH-SN-001", "")

	svc := &fakeAssetLookup{db: db}
	rec := &noopRecorder{}
	w := NewOpsAssetWriter(db, svc, rec)
	// 注意:parentAssetID 必须传非空才能让 emitter 拿到父交换机 id;
	// 真实 pipeline 里 Pipeline.Run 会查到父交换机 id 后传入。
	res, err := w.Write(context.Background(), "switch-1", "asset-switch-1", []Component{
		{ComponentType: "card", Slot: "Slot 2", SerialNumber: "ORPHAN-SN-999"},
	})
	require.NoError(t, err)
	require.Equal(t, 0, res.HitCount)
	require.Equal(t, 1, res.MissCount)

	// 验证 sys_data_reconciliation 有一行
	var reconCount int64
	require.NoError(t, db.Table("sys_data_reconciliation").Where("asset_id = ?", "asset-switch-1").Count(&reconCount).Error)
	require.Equal(t, int64(1), reconCount, "1 anomaly row for the parent switch")

	type reconRow struct {
		ConflictType  string
		ReconCategory *string
		Severity      string
	}
	var r reconRow
	require.NoError(t, db.Table("sys_data_reconciliation").Where("asset_id = ?", "asset-switch-1").Scan(&r).Error)
	require.Equal(t, "F", r.ConflictType, "conflict_type=F (data missing)")
	require.NotNil(t, r.ReconCategory, "recon_category populated")
	require.Equal(t, "component_serial", *r.ReconCategory)
	require.Equal(t, "medium", r.Severity, "severity=medium per RESEARCH Open Question 2")

	// Miss path 不调 operlog(D-13 仅 UPDATE 路径)
	require.Equal(t, 0, rec.calls, "miss path does not call operlog")
}

// TestOpsAssetWriterParentMissingDoesNotEmitSwitchAnomaly REQ-48-08 (D-04):
// parentAssetID="" 时仍 UPDATE 组件行,parent_asset_id 列=NULL,不产生交换机侧异常。
func TestOpsAssetWriterParentMissingDoesNotEmitSwitchAnomaly(t *testing.T) {
	db := setupWriterTestDB(t)
	insertAsset(t, db, "asset-card-1", "CARD-SN-001", "card")

	svc := &fakeAssetLookup{db: db}
	rec := &noopRecorder{}
	w := NewOpsAssetWriter(db, svc, rec)

	// parentAssetID="" 模拟 D-04 场景(父交换机不在 ops_asset)
	res, err := w.Write(context.Background(), "switch-1", "", []Component{
		{ComponentType: "card", Slot: "Slot 1", SerialNumber: "CARD-SN-001"},
	})
	require.NoError(t, err)
	require.Equal(t, 1, res.HitCount, "组件 UPDATE 仍命中")
	require.Equal(t, 0, res.MissCount)

	// parent_asset_id 列应为 NULL,其余 3 列写入
	type row struct {
		ParentAssetID  *string
		SourceDeviceID *string
		ComponentType  *string
		ComponentSlot  *string
	}
	var r1 row
	require.NoError(t, db.Table("ops_asset").Where("id = ?", "asset-card-1").Scan(&r1).Error)
	require.Nil(t, r1.ParentAssetID, "D-04: parent_asset_id 留 NULL")
	require.NotNil(t, r1.SourceDeviceID, "source_device_id 仍写")
	require.NotNil(t, r1.ComponentType)
	require.NotNil(t, r1.ComponentSlot)

	// 不应产生交换机侧异常(没有 anomaly 行)
	var reconCount int64
	require.NoError(t, db.Table("sys_data_reconciliation").Count(&reconCount).Error)
	require.Equal(t, int64(0), reconCount, "D-04: parent 缺失不报异常")
}

// TestReconciliationEmitterIdempotent REQ-48-06: 重复 emit 同 (assetID, F, component_serial)
// 第二次 Insert 会触发 unique 冲突;emitter 应 swallow 后日志跳过(sqlite 没有部分 unique index,
// 但 emitter 用 (asset_id, conflict_type, recon_category, resolved_at IS NULL) 显式查询去重兜底)。
func TestReconciliationEmitterIdempotent(t *testing.T) {
	db := setupWriterTestDB(t)
	em := NewReconciliationEmitter(db)

	snapshot, _ := json.Marshal(map[string]string{"sn": "X"})
	require.NoError(t, em.Emit(context.Background(), "asset-1", "F", "component_serial", snapshot))
	// 第二次 — 必须 NOT error 且表里仍只有 1 行
	require.NoError(t, em.Emit(context.Background(), "asset-1", "F", "component_serial", snapshot))

	var cnt int64
	require.NoError(t, db.Table("sys_data_reconciliation").Where("asset_id = ? AND conflict_type = ? AND recon_category = ?",
		"asset-1", "F", "component_serial").Count(&cnt).Error)
	require.Equal(t, int64(1), cnt, "idempotent: duplicate emit keeps only 1 row")
}

// TestPipelineEndToEnd REQ-48-05/06/08 串起来:
// 父交换机在 ops_asset → 组件 1 命中 UPDATE / 组件 2 miss emit anomaly
func TestPipelineEndToEnd(t *testing.T) {
	db := setupWriterTestDB(t)
	insertAsset(t, db, "asset-switch-1", "SWITCH-SN-001", "")
	insertAsset(t, db, "asset-card-1", "CARD-SN-001", "card")

	svc := &fakeAssetLookup{db: db}
	rec := &noopRecorder{}
	p := NewPipeline(db, svc, rec)

	dev := &fakeNetworkDevice{id: "switch-1", serial: "SWITCH-SN-001"}
	set := &ComponentSet{
		Chassis: &Component{ComponentType: "chassis", SerialNumber: "SWITCH-SN-001"},
		Components: []Component{
			{ComponentType: "card", Slot: "Slot 1", SerialNumber: "CARD-SN-001"},
			{ComponentType: "fan", Slot: "Fan 1", SerialNumber: "ORPHAN-FAN-999"},
		},
	}
	require.NoError(t, p.Run(context.Background(), dev.toDeviceRef(), set))

	// 组件 1:4 列写入
	type row struct {
		ParentAssetID  *string
		SourceDeviceID *string
		ComponentType  *string
		ComponentSlot  *string
	}
	var r1 row
	require.NoError(t, db.Table("ops_asset").Where("id = ?", "asset-card-1").Scan(&r1).Error)
	require.NotNil(t, r1.ParentAssetID)
	require.Equal(t, "asset-switch-1", *r1.ParentAssetID)
	require.NotNil(t, r1.SourceDeviceID)

	// anomaly 1 行,asset_id=asset-switch-1
	var reconCount int64
	require.NoError(t, db.Table("sys_data_reconciliation").Where("asset_id = ?", "asset-switch-1").Count(&reconCount).Error)
	require.Equal(t, int64(1), reconCount)

	// operlog:1 次(只有 hit 触发)
	require.Equal(t, 1, rec.calls)
}

// TestPipelineParentMissingInOpsAsset REQ-48-08 端到端:
// 父交换机不在 ops_asset → parentAssetID="",组件 UPDATE 仍进行,parent_asset_id=NULL。
func TestPipelineParentMissingInOpsAsset(t *testing.T) {
	db := setupWriterTestDB(t)
	insertAsset(t, db, "asset-card-1", "CARD-SN-001", "card")

	svc := &fakeAssetLookup{db: db}
	rec := &noopRecorder{}
	p := NewPipeline(db, svc, rec)

	dev := &fakeNetworkDevice{id: "switch-1", serial: "SWITCH-SN-999-NO-MATCH"}
	set := &ComponentSet{
		Components: []Component{
			{ComponentType: "card", Slot: "Slot 1", SerialNumber: "CARD-SN-001"},
		},
	}
	require.NoError(t, p.Run(context.Background(), dev.toDeviceRef(), set))

	type row struct {
		ParentAssetID *string
		SourceDeviceID *string
	}
	var r1 row
	require.NoError(t, db.Table("ops_asset").Where("id = ?", "asset-card-1").Scan(&r1).Error)
	require.Nil(t, r1.ParentAssetID, "D-04: parent_asset_id NULL when switch absent")
	require.NotNil(t, r1.SourceDeviceID)
}

// ===== Test doubles =====

// fakeAssetLookup mimics operations.AssetService.GetByDeviceSN.
// Returns (*AssetRef, nil) on hit, (nil, nil) on miss, matching the production contract.
type fakeAssetLookup struct {
	db *gorm.DB
}

func (f *fakeAssetLookup) GetByDeviceSN(ctx context.Context, deviceSN string) (*AssetRef, error) {
	if deviceSN == "" {
		return nil, fmt.Errorf("empty sn")
	}
	type row struct {
		ID string
	}
	var r row
	err := f.db.WithContext(ctx).Table("ops_asset").
		Where("devicesn = ? AND deleted_at IS NULL", deviceSN).
		Select("id").Limit(1).Scan(&r).Error
	if err != nil || r.ID == "" {
		return nil, nil // not-found is (nil, nil)
	}
	return &AssetRef{ID: r.ID}, nil
}

// fakeNetworkDevice 携带 Pipeline.Run 实际读到的字段。
type fakeNetworkDevice struct {
	id     string
	serial string
}

// toDeviceRef adapts the fake to the production Pipeline.Run signature.
func (d *fakeNetworkDevice) toDeviceRef() DeviceRef {
	return DeviceRef{ID: d.id, SerialNumber: d.serial}
}
