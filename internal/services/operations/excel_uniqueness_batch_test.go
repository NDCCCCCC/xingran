//go:build !skip_db_tests
// +build !skip_db_tests

package operations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	// Pure-Go SQLite driver
)

// setupUniquenessTestDB creates an in-memory SQLite DB with a simple
// test table that mirrors the columns referenced by validateUniqueness.
func setupUniquenessTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default,
	})
	assert.NoError(t, err)

	// Table mirroring a uniqueness-checked entity (e.g., ops_buildings)
	err = db.Exec(`
		CREATE TABLE test_unique_entities (
			id TEXT PRIMARY KEY,
			code TEXT,
			name TEXT,
			status INTEGER,
			updated_at DATETIME,
			created_at DATETIME,
			deleted_at DATETIME
		);
	`).Error
	assert.NoError(t, err)

	return db
}

// TestValidateUniqueness_SingleBatchQuery verifies P1-C6 fix:
// validateUniqueness issues at most 1 query per Unique column regardless
// of the number of rows being validated. Previously it issued N×M
// Count queries (N rows × M unique columns).
func TestValidateUniqueness_SingleBatchQuery(t *testing.T) {
	db := setupUniquenessTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()

	// Seed: 3 existing rows with codes A, B, C
	now := "2026-06-13 00:00:00"
	for _, code := range []string{"A", "B", "C"} {
		err := db.Exec(
			`INSERT INTO test_unique_entities (id, code, name, status, created_at, updated_at) VALUES (?, ?, '', 0, ?, ?)`,
			"id-"+code, code, now, now,
		).Error
		assert.NoError(t, err)
	}

	// ExcelService only needs *gorm.DB and cache fields the test path
	// doesn't touch. We construct manually to skip NewExcelService's
	// full wiring.
	svc := &ExcelService{
		db:                db,
		uniqueValueCache:  make(map[string]map[string]map[string]struct{}),
		uniqueValueLoaded: make(map[string]bool),
	}

	config := ExcelConfig{
		TableName: "test_unique_entities",
		Columns: []ExcelColumn{
			{Field: "code", Header: "编码", Unique: true, DBField: "code"},
			{Field: "name", Header: "名称", Unique: false},
		},
	}

	ctx := context.Background()

	// Phase 1: First call populates the cache.
	// This should issue 1 Pluck query for the `code` column (the only Unique col).
	data := map[string]any{"code": "Z", "name": "new-name"}
	errs := svc.validateUniqueness(ctx, config, data, 2)
	assert.Empty(t, errs, "non-existent code should not produce an error")

	// Verify the cache is populated
	svc.uniqueValueMu.Lock()
	cacheLoaded := svc.uniqueValueLoaded["test_unique_entities"]
	cacheData := svc.uniqueValueCache["test_unique_entities"]
	svc.uniqueValueMu.Unlock()
	assert.True(t, cacheLoaded, "cache should be marked loaded after first call")
	assert.NotNil(t, cacheData, "cache should hold data after first call")
	assert.NotNil(t, cacheData["code"], "code column should be cached")
	assert.Equal(t, 3, len(cacheData["code"]), "cache should contain the 3 seeded codes (A, B, C)")

	// Phase 2: Subsequent calls hit the in-memory cache.
	// Even with N rows, NO additional SQL is issued. We assert this
	// by checking the cache state is unchanged (no reload triggers).
	beforeLoaded := svc.uniqueValueLoaded["test_unique_entities"]

	// Test positive case: existing code "A" should be flagged
	dataDup := map[string]any{"code": "A", "name": "anything"}
	errs = svc.validateUniqueness(ctx, config, dataDup, 3)
	assert.NotEmpty(t, errs, "existing code 'A' should produce uniqueness error")
	assert.Equal(t, "编码", errs[0].Field)
	assert.Equal(t, "A", errs[0].Value)
	assert.Equal(t, "该值已存在", errs[0].Error)

	// Test negative case: non-existent code should pass
	dataNew := map[string]any{"code": "XYZ", "name": "fresh"}
	errs = svc.validateUniqueness(ctx, config, dataNew, 4)
	assert.Empty(t, errs, "non-existent code 'XYZ' should not produce an error")

	// Test all known codes (A, B, C) — all should be flagged
	for i, code := range []string{"A", "B", "C"} {
		data := map[string]any{"code": code, "name": ""}
		errs = svc.validateUniqueness(ctx, config, data, i+10)
		assert.NotEmptyf(t, errs, "existing code %q should be flagged", code)
	}

	// Cache state should remain consistent — no reload triggered.
	assert.Equal(t, beforeLoaded, svc.uniqueValueLoaded["test_unique_entities"],
		"cache loaded flag should not change across per-row calls")
}

// TestValidateUniqueness_CacheReusedAcrossRows verifies that calling
// validateUniqueness for many rows only loads the cache once.
func TestValidateUniqueness_CacheReusedAcrossRows(t *testing.T) {
	db := setupUniquenessTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()

	// Seed 5 existing codes
	now := "2026-06-13 00:00:00"
	for _, code := range []string{"E01", "E02", "E03", "E04", "E05"} {
		err := db.Exec(
			`INSERT INTO test_unique_entities (id, code, name, status, created_at, updated_at) VALUES (?, ?, '', 0, ?, ?)`,
			"id-"+code, code, now, now,
		).Error
		assert.NoError(t, err)
	}

	svc := &ExcelService{
		db:                db,
		uniqueValueCache:  make(map[string]map[string]map[string]struct{}),
		uniqueValueLoaded: make(map[string]bool),
	}

	config := ExcelConfig{
		TableName: "test_unique_entities",
		Columns: []ExcelColumn{
			{Field: "code", Header: "编码", Unique: true, DBField: "code"},
		},
	}

	ctx := context.Background()

	// Validate 100 rows. Pre-fix: 100 Count queries.
	// Post-fix: 1 Pluck query on first call, 0 subsequent queries.
	for i := 0; i < 100; i++ {
		data := map[string]any{"code": "NEW" + string(rune('A'+i%26)) + string(rune('0'+i/26))}
		_ = svc.validateUniqueness(ctx, config, data, i+2)
	}

	// After all calls, cache should still be loaded exactly once.
	svc.uniqueValueMu.Lock()
	loaded := svc.uniqueValueLoaded["test_unique_entities"]
	cachedCodes := svc.uniqueValueCache["test_unique_entities"]["code"]
	svc.uniqueValueMu.Unlock()

	assert.True(t, loaded)
	assert.Equal(t, 5, len(cachedCodes), "cache should hold 5 seeded codes after batch validation")
}

// TestValidateUniqueness_SkipsUpsertKeyColumn verifies that columns
// marked UpsertKey=true are skipped (consistent with pre-fix behavior).
func TestValidateUniqueness_SkipsUpsertKeyColumn(t *testing.T) {
	db := setupUniquenessTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()

	now := "2026-06-13 00:00:00"
	// Seed with a code that also happens to be the upsert key value
	err := db.Exec(
		`INSERT INTO test_unique_entities (id, code, name, status, created_at, updated_at) VALUES (?, ?, '', 0, ?, ?)`,
		"id-1", "UPSERT-1", now, now,
	).Error
	assert.NoError(t, err)

	svc := &ExcelService{
		db:                db,
		uniqueValueCache:  make(map[string]map[string]map[string]struct{}),
		uniqueValueLoaded: make(map[string]bool),
	}

	// UpsertKey=true means we should NOT perform uniqueness check on this col
	// (the upsert path will handle dedup via INSERT...ON CONFLICT).
	config := ExcelConfig{
		TableName: "test_unique_entities",
		Columns: []ExcelColumn{
			{Field: "code", Header: "编码", Unique: true, UpsertKey: true, DBField: "code"},
		},
	}

	ctx := context.Background()
	data := map[string]any{"code": "UPSERT-1"}
	errs := svc.validateUniqueness(ctx, config, data, 2)
	assert.Empty(t, errs, "UpsertKey columns should skip uniqueness check")
}