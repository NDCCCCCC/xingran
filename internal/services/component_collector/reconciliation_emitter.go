package component_collector

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ReconciliationEmitter inserts component-serial anomaly rows into
// sys_data_reconciliation per D-06.
//
// Each emit writes one row with:
//   - asset_id       = supplied UUID (typically the parent switch's
//                      ops_asset.id — the row the UI locates the anomaly
//                      under)
//   - conflict_type  = supplied (typically "F" for missing data)
//   - recon_category = supplied (typically "component_serial")
//   - severity       = "medium" (RESEARCH Open Question 2)
//   - detected_at    = now()
//   - resolved_at    = NULL (open)
//
// Idempotency: the production PG schema has a partial unique index
// uniq_recon_asset_type_cat_open (asset_id, conflict_type, recon_category)
// WHERE resolved_at IS NULL AND deleted_at IS NULL (migration_201). On PG,
// a duplicate emit raises SQLSTATE 23505 which we swallow.
//
// On SQLite (tests), no partial unique index exists, so the emitter adds
// a pre-INSERT dedup query: COUNT(*) WHERE (asset_id, conflict_type,
// recon_category, resolved_at IS NULL) → skip if >= 1. This is the same
// logical predicate as the PG index, so behaviour matches.
type ReconciliationEmitter struct {
	db *gorm.DB
}

// NewReconciliationEmitter constructs an emitter bound to db.
func NewReconciliationEmitter(db *gorm.DB) *ReconciliationEmitter {
	return &ReconciliationEmitter{db: db}
}

// Emit writes one anomaly row. Returns nil on success, nil on idempotent
// duplicate (already-open anomaly for the same triple), or the underlying
// DB error otherwise.
func (e *ReconciliationEmitter) Emit(ctx context.Context, assetID, conflictType, reconCategory string, rawSnapshot json.RawMessage) error {
	if e.db == nil {
		return nil
	}
	if assetID == "" || conflictType == "" {
		return errors.New("reconciliation_emitter: assetID and conflictType are required")
	}

	// Pre-INSERT dedup — SQLite has no partial unique index; on PG this
	// is a cheap defensive query that also handles the race window before
	// the unique violation surfaces.
	//
	// NOTE: 用 `recon_category = ?` 单子句。原写法 `(recon_category IS ? OR
	// recon_category = ?)` 在 SQLite 下工作(SQLite 把 IS 当 nullable 等价),
	// 但 PostgreSQL 拒绝 `IS <string>` —— SQLSTATE 42601 "syntax error at or
	// near $3",导致每次组件 MISS 的 anomaly emit 失败(Phase 49-02 E2E 暴露)。
	// partial unique index uniq_recon_asset_type_cat_open 本身按精确值匹配,
	// NULL 在 PG unique index 里互异,因此 dedup 也只需精确值匹配。
	var existing int64
	if err := e.db.WithContext(ctx).
		Table("sys_data_reconciliation").
		Where("asset_id = ? AND conflict_type = ? AND recon_category = ?",
			assetID, conflictType, reconCategory).
		Where("resolved_at IS NULL AND deleted_at IS NULL").
		Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		// Idempotent — already an open anomaly for this triple.
		return nil
	}

	rc := reconCategory
	severity := "medium"
	now := time.Now()
	row := map[string]interface{}{
		"id":             newUUID(),
		"asset_id":       assetID,
		"conflict_type":  conflictType,
		"recon_category": &rc,
		"severity":       severity,
		"raw_snapshot":   string(rawSnapshot),
		"detected_at":    now,
		"created_at":     now,
		"updated_at":     now,
	}
	if err := e.db.WithContext(ctx).
		Table("sys_data_reconciliation").
		Create(row).Error; err != nil {
		// Swallow PG unique-violation (23505) — race-condition duplicate.
		if isUniqueViolation(err) {
			return nil
		}
		return err
	}
	return nil
}

// isUniqueViolation returns true when err is a PG unique-violation
// (SQLSTATE 23505). Recognises both gorm's wrapped error string and the
// raw pq error message.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "unique constraint")
}
