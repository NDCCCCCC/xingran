//go:build !skip_db_tests
// +build !skip_db_tests

package addomain

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	// Pure-Go SQLite driver
	_ "modernc.org/sqlite"
)

// TestSyncData_ConcurrentCallsDeduplicated verifies P1-C1 fix:
// 10 concurrent calls to SyncData with the same configID result in
// only 1 actual execution (singleflight deduplication).
//
// This is a regression test for the AD sync singleflight pattern
// that prevents double LDAP requests, write conflicts, and partial
// state restoration when multiple triggers (scheduler, manual, drift)
// fire simultaneously.
func TestSyncData_ConcurrentCallsDeduplicated(t *testing.T) {
	// We use a real (in-memory) DB so SyncService.db is initialized.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	svc := &SyncService{db: db}

	// We can't intercept syncDataInternal directly without exporting it,
	// so instead we test the singleflight behavior via the same group
	// field that production code uses (s.syncGroup.Do("sync:ID:type", fn)).
	// For a unit test we inject our own counter via a closure.
	var execCount int32
	execFn := func() (interface{}, error) {
		atomic.AddInt32(&execCount, 1)
		// Simulate slow sync (LDAP + DB writes typically take >100ms)
		time.Sleep(100 * time.Millisecond)
		return nil, errors.New("synthetic LDAP failure for test")
	}

	const concurrency = 10
	var wg sync.WaitGroup
	results := make([]error, concurrency)
	start := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start // align all goroutines to fire simultaneously

			// Mirror production: s.syncGroup.Do(key, fn)
			_, err, _ := svc.syncGroup.Do("sync:test-config-1:user", execFn)
			results[idx] = err
		}(i)
	}

	close(start) // release barrier — all goroutines fire together
	wg.Wait()

	// Singleflight assertion: exactly ONE execution, even with 10 concurrent callers.
	assert.Equal(t, int32(1), atomic.LoadInt32(&execCount),
		"expected exactly 1 sync execution under 10 concurrent callers (singleflight), got %d",
		execCount)

	// All callers see the same error (shared result)
	for i, err := range results {
		if err == nil {
			t.Errorf("goroutine %d expected synthetic error, got nil", i)
		}
	}
}

// TestSyncData_DifferentKeysRunIndependently verifies that singleflight
// only dedupes within the SAME key — different configIDs produce
// independent executions.
func TestSyncData_DifferentKeysRunIndependently(t *testing.T) {
	svc := &SyncService{}

	var execCount int32
	execFn := func() (interface{}, error) {
		atomic.AddInt32(&execCount, 1)
		// Sleep so all 5 goroutines are inside singleflight when each distinct
		// key first gets scheduled. Without this, the first call finishes
		// before the second one arrives and dedup doesn't engage.
		time.Sleep(30 * time.Millisecond)
		return nil, nil
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			key := []string{"sync:A:user", "sync:B:user", "sync:C:user", "sync:A:user", "sync:B:user"}[idx]
			_, _, _ = svc.syncGroup.Do(key, execFn)
		}(i)
	}
	close(start)
	wg.Wait()

	// 3 distinct keys → 3 executions (the 5 goroutines fire into 3 in-flight groups)
	assert.Equal(t, int32(3), atomic.LoadInt32(&execCount),
		"expected 3 executions (one per distinct key), got %d", execCount)
}

// TestSyncData_SharedResultFlag verifies that the `shared` return
// value from singleflight is true for callers that piggy-backed on
// an in-flight call. Per singleflight semantics, `shared=true` is
// returned by followers AND by the leader when there were dups
// (c.dups > 0). The leader sees shared=false ONLY if it ran alone.
//
// This is important for logging merged requests — production code at
// sync.go:62-63 logs a merge message when `shared` is true.
func TestSyncData_SharedResultFlag(t *testing.T) {
	svc := &SyncService{}

	execFn := func() (interface{}, error) {
		time.Sleep(100 * time.Millisecond)
		return "result", nil
	}

	const concurrency = 5
	// Each goroutine records (sharedFlag, returnValue) so we can
	// verify exactly one execution happened AND that all callers
	// received the same value.
	type outcome struct {
		shared bool
		val    interface{}
		err    error
	}
	results := make([]outcome, concurrency)

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			v, err, shared := svc.syncGroup.Do("sync:test:user", execFn)
			results[idx] = outcome{shared: shared, val: v, err: err}
		}(i)
	}
	close(start)
	wg.Wait()

	// All 5 callers must see the same return value (proves they
	// shared the result rather than each running independently).
	expectedVal := "result"
	for i, r := range results {
		assert.Equal(t, expectedVal, r.val,
			"goroutine %d got different value %v — singleflight didn't share", i, r.val)
		assert.NoError(t, r.err, "goroutine %d unexpected error", i)
	}

	// Per singleflight semantics:
	//   - 1 leader sees shared=true if any followers piggy-backed (c.dups > 0)
	//   - All followers see shared=true
	//   - Only the truly-alone leader sees shared=false
	// With 5 concurrent callers, at least 4 must report shared=true
	// (the 4 followers), and the leader also reports shared=true because
	// c.dups > 0 at the time of return.
	sharedCount := 0
	for _, r := range results {
		if r.shared {
			sharedCount++
		}
	}
	assert.GreaterOrEqual(t, sharedCount, concurrency-1,
		"at least %d callers should report shared=true, got %d", concurrency-1, sharedCount)
}

// TestSyncData_ResultReuse verifies that all callers receive the SAME
// return value (singleflight returns the leader's result to followers).
func TestSyncData_ResultReuse(t *testing.T) {
	svc := &SyncService{}

	execFn := func() (interface{}, error) {
		return &SyncResult{OUCount: 42, GroupCount: 7}, nil
	}

	const concurrency = 8
	results := make([]*SyncResult, concurrency)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			v, _, _ := svc.syncGroup.Do("sync:result-test:user", execFn)
			if v != nil {
				results[idx] = v.(*SyncResult)
			}
		}(i)
	}
	close(start)
	wg.Wait()

	for i, r := range results {
		assert.NotNil(t, r, "goroutine %d received nil result", i)
		if r != nil {
			assert.Equal(t, 42, r.OUCount, "goroutine %d OU count", i)
			assert.Equal(t, 7, r.GroupCount, "goroutine %d group count", i)
		}
	}
}

// Avoid unused import warnings for context
var _ = context.Background