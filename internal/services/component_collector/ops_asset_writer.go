package component_collector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"gorm.io/gorm"
)

// OpsAssetWriter implements the UPDATE-only ops_asset write path
// (D-02 / D-03 / D-04 / D-13).
//
// For each Component in the input list:
//
//   - HIT  (SN found in ops_asset)  → UPDATE the 4 component columns
//     (parent_asset_id, source_device_id, component_type, component_slot)
//     and call operlog.RecordBackground once (D-13).
//   - MISS (SN not found, parentAssetID != "") → emit one anomaly row in
//     sys_data_reconciliation (conflict_type=F, recon_category=
//     component_serial, severity=medium). D-06.
//   - MISS when parentAssetID == "" → anomaly cannot anchor to a parent
//     ops_asset row; emit is skipped silently (D-04 isolation — the
//     switch-side absense is a separate concern).
//
// The writer never INSERTs new ops_asset rows (D-02) and never updates
// devicesn or any non-component column (avoids corrupting Excel-imported
// asset data).
type OpsAssetWriter struct {
	db       *gorm.DB
	assetSvc AssetLookup
	operLog  operlog.Recorder
	emitter  *ReconciliationEmitter
}

// WriteResult summarises one Write call. Counts feed pipeline metrics /
// logs without exposing internal rows.
type WriteResult struct {
	// HitCount = number of components matched to an ops_asset row.
	HitCount int
	// MissCount = number of components with no ops_asset match. Anomaly
	// emission may be 0 when parentAssetID is "" (D-04).
	MissCount int
	// EmittedAnomalies lists asset_id values for which an anomaly row was
	// emitted (length <= MissCount; deduped by asset_id).
	EmittedAnomalies []string
}

// NewOpsAssetWriter constructs a writer. operLog may be nil (tests skip
// operlog assertion paths). The ReconciliationEmitter is constructed
// internally bound to db; tests inject a fake via NewOpsAssetWriterWithEmitter.
func NewOpsAssetWriter(db *gorm.DB, assetSvc AssetLookup, operLog operlog.Recorder) *OpsAssetWriter {
	w := &OpsAssetWriter{db: db, assetSvc: assetSvc, operLog: operLog}
	w.emitter = NewReconciliationEmitter(db)
	return w
}

// NewOpsAssetWriterWithEmitter is the test-friendly constructor that
// accepts an explicit ReconciliationEmitter (e.g. for spy assertions).
func NewOpsAssetWriterWithEmitter(db *gorm.DB, assetSvc AssetLookup, operLog operlog.Recorder, emitter *ReconciliationEmitter) *OpsAssetWriter {
	if emitter == nil {
		emitter = NewReconciliationEmitter(db)
	}
	return &OpsAssetWriter{db: db, assetSvc: assetSvc, operLog: operLog, emitter: emitter}
}

// Write iterates components and applies the UPDATE / emit / operlog
// sequence described in the struct doc. Returns (WriteResult, error);
// error is non-nil only on DB failure, which halts the pipeline (per
// D-14 the cron caller logs warn and continues the chassis update path).
func (w *OpsAssetWriter) Write(ctx context.Context, deviceID, parentAssetID string, components []Component) (WriteResult, error) {
	res := WriteResult{}
	if w.assetSvc == nil || w.db == nil {
		return res, nil
	}

	// parentAssetIDPtr is the value written to parent_asset_id. nil when
	// D-04 (parent absent) → NULL preserved on the column. Otherwise a
	// pointer to the UUID string.
	var parentAssetIDPtr *string
	if parentAssetID != "" {
		s := parentAssetID
		parentAssetIDPtr = &s
	}

	// emitted dedup — multiple orphan components on the same parent only
	// emit one anomaly (the partial unique index would dedup anyway, but
	// avoiding the INSERT attempt saves DB roundtrips and log noise).
	emitted := make(map[string]bool)

	for _, comp := range components {
		if comp.SerialNumber == "" {
			// Placeholder component (e.g. chassis row projected via SNMP
			// without SN) — nothing to look up.
			continue
		}

		asset, err := w.assetSvc.GetByDeviceSN(ctx, comp.SerialNumber)
		if err != nil {
			return res, fmt.Errorf("ops_asset_writer: lookup %s: %w", comp.SerialNumber, err)
		}
		if asset == nil {
			// MISS — emit anomaly only if we have a parent asset_id to
			// anchor it on (D-04: skip silently otherwise).
			res.MissCount++
			if parentAssetID != "" && !emitted[parentAssetID] {
				if emitErr := w.emitComponentAnomaly(ctx, parentAssetID, comp); emitErr != nil {
					// Non-fatal — log via return chain. Pipeline.Run does
					// not treat emit failure as fatal (idempotent retries
					// on next run via partial unique index).
					return res, fmt.Errorf("ops_asset_writer: emit anomaly: %w", emitErr)
				}
				emitted[parentAssetID] = true
				res.EmittedAnomalies = append(res.EmittedAnomalies, parentAssetID)
			}
			continue
		}

		// HIT — UPDATE 4 component columns. Use Updates(map) so *string
		// NULL semantics (parentAssetIDPtr=nil → NULL) are honoured.
		ct := comp.ComponentType
		slot := comp.Slot
		updates := map[string]interface{}{
			"parent_asset_id":  parentAssetIDPtr,
			"source_device_id": deviceID,
			"component_type":   &ct,
			"component_slot":   &slot,
			"updated_at":       time.Now(),
		}
		if err := w.db.WithContext(ctx).
			Table("ops_asset").
			Where("id = ?", asset.ID).
			Updates(updates).Error; err != nil {
			return res, fmt.Errorf("ops_asset_writer: update %s: %w", asset.ID, err)
		}
		res.HitCount++

		// operlog.RecordBackground per D-13. nil-check lets tests inject a
		// noop recorder or skip entirely.
		if w.operLog != nil {
			operlog.RecordBackground(
				w.operLog,
				w.db,
				"资产管理",
				operlog.OperTypeSync,
				"system-cron",
				map[string]interface{}{
					"assetId":       asset.ID,
					"deviceSN":      comp.SerialNumber,
					"componentType": comp.ComponentType,
					"componentSlot": comp.Slot,
					"sourceDevice":  deviceID,
					"parentAssetId": parentAssetID,
				},
			)
		}
	}
	return res, nil
}

// emitComponentAnomaly emits one sys_data_reconciliation row anchored on
// the parent switch's ops_asset.id. severity=medium per RESEARCH Open
// Question 2. ReconCategory=component_serial per D-06. ConflictType=F
// (data missing). The emitter's idempotency (partial unique index +
// pre-emit dedup query) keeps repeat emits to 1 row per
// (asset_id, conflict_type, recon_category).
func (w *OpsAssetWriter) emitComponentAnomaly(ctx context.Context, parentAssetID string, comp Component) error {
	snapshot, _ := json.Marshal(map[string]interface{}{
		"componentSN":    comp.SerialNumber,
		"componentType":  comp.ComponentType,
		"componentSlot":  comp.Slot,
		"model":          comp.Model,
		"source":         comp.Source,
		"detectedAt":     time.Now().UTC().Format(time.RFC3339),
	})
	return w.emitter.Emit(ctx, parentAssetID, "F", "component_serial", snapshot)
}
