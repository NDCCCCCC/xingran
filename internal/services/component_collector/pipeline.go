// Package component_collector already defines the Wave 2 parsing types
// (ComponentSet / Component / EntityRow / OwnedComponent) and collectors
// (EntityCollector / HuaweiCliCollector / RuijieCliCollector). This file
// adds the Wave 3 runtime glue: an AssetLookup abstraction (so the package
// avoids importing internal/services/operations and stays cycle-free),
// a small DeviceRef adapter, the OpsAssetWriter (UPDATE-only per D-02/D-03/
// D-04/D-13), the ReconciliationEmitter (D-06 sibling-column anomaly), and
// the Pipeline orchestrator.
package component_collector

import (
	"context"

	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"gorm.io/gorm"
)

// AssetLookup is the read-side slice of operations.AssetService that the
// component pipeline consumes. Defined here (rather than imported from
// internal/services/operations) so the component_collector package stays
// free of the operations dependency — the wiring layer (cron hook)
// supplies a thin adapter implementing this interface.
//
// Contract mirrors operations.AssetService.GetByDeviceSN exactly:
//   - found            → (*AssetRef, nil)
//   - not found        → (nil, nil)            // NOT an error
//   - DB / query error → (nil, error)          // propagates, halts pipeline
type AssetLookup interface {
	GetByDeviceSN(ctx context.Context, deviceSN string) (*AssetRef, error)
}

// AssetRef is the minimal projection of models.Asset the writer needs
// (just the primary key, since UPDATE targets ops_asset.id).
type AssetRef struct {
	// ID is the ops_asset.id UUID of the matched row.
	ID string
}

// DeviceRef is the minimal projection of models.NetworkDevice that
// Pipeline.Run consumes. The wiring layer adapts a *models.NetworkDevice to
// this struct; tests construct one directly.
type DeviceRef struct {
	// ID is sys_network_device.id — written to ops_asset.source_device_id.
	ID string
	// SerialNumber is the chassis SN — used to locate the parent ops_asset
	// row via AssetLookup.GetByDeviceSN.
	SerialNumber string
	// Vendor identifies the device vendor (Huawei / Ruijie / H3C / Maipu) —
	// consulted by the cron hook's D-10 two-step transceiver pipeline
	// dispatcher; not used directly by Pipeline.Run today but exposed so
	// future hooks can read it without re-querying.
	Vendor string
}

// Pipeline orchestrates one device's component collection cycle:
//
//  1. Resolve the parent (chassis) ops_asset row by looking up
//     device.SerialNumber via AssetLookup. If absent (D-04), parentAssetID
//     is "" — components still UPDATE with parent_asset_id=NULL.
//  2. Hand the component list + parentAssetID + device.ID to OpsAssetWriter
//     which performs the per-SN lookup/UPDATE/anomaly/operlog sequence.
//
// Pipeline.Run does NOT collect from the device — that is the
// DeviceInfoCollectionService.collectComponentInfo hook's job. Pipeline.Run
// accepts an already-built ComponentSet and writes its effect.
type Pipeline struct {
	db        *gorm.DB
	assetSvc  AssetLookup
	operLog   operlog.Recorder
	emitter   *ReconciliationEmitter
}

// NewPipeline constructs a Pipeline bound to the supplied dependencies.
// operLogSvc may be nil in tests (the writer nil-checks before calling).
func NewPipeline(db *gorm.DB, assetSvc AssetLookup, operLogSvc operlog.Recorder) *Pipeline {
	return &Pipeline{
		db:       db,
		assetSvc: assetSvc,
		operLog:  operLogSvc,
		emitter:  NewReconciliationEmitter(db),
	}
}

// Run executes the pipeline for one device + component set. Returns nil
// unless a DB error halts propagation. Per D-14, callers (cron hook) treat
// any error as a non-fatal warning — chassis updates are unaffected.
func (p *Pipeline) Run(ctx context.Context, device DeviceRef, set *ComponentSet) error {
	if set == nil || p.assetSvc == nil {
		return nil
	}

	// Step 1: resolve parent (chassis) ops_asset row by device SN (D-04).
	parentAssetID := ""
	if device.SerialNumber != "" {
		parent, err := p.assetSvc.GetByDeviceSN(ctx, device.SerialNumber)
		if err != nil {
			return err
		}
		if parent != nil {
			parentAssetID = parent.ID
		}
		// D-04: parent == nil → parentAssetID stays "" — no anomaly emitted
		// for the chassis row; component UPDATE still proceeds.
	}

	// Step 2: feed (device.ID, parentAssetID, components) to the writer.
	writer := NewOpsAssetWriterWithEmitter(p.db, p.assetSvc, p.operLog, p.emitter)
	_, err := writer.Write(ctx, device.ID, parentAssetID, set.Components)
	return err
}
