package cache

import (
	"context"
	"testing"
	"time"
)

// TestMemoryCache_Close_Idempotent verifies that Close() can be called multiple times
// without panicking (QUIRK-P1: close of closed channel).
func TestMemoryCache_Close_Idempotent(t *testing.T) {
	cache := NewMemoryCache(100, time.Hour)

	// Verify cache works before close
	ctx := context.Background()
	cache.Set(ctx, "key", "value", time.Minute)
	val, err := cache.Get(ctx, "key")
	if err != nil {
		t.Fatalf("expected no error before close, got %v", err)
	}
	if val != "value" {
		t.Fatalf("expected 'value', got %q", val)
	}

	// First close — must not panic
	cache.Close()

	// Second close — must also not panic (sync.Once guards against double-close)
	cache.Close()

	// Verify cache is still accessible after close (reads from existing items)
	// Note: cleanup goroutine has stopped, but in-memory items remain
	_, err = cache.Get(ctx, "key")
	if err != nil && err != ErrExpired {
		t.Fatalf("Get after Close should not return unexpected error: %v", err)
	}
}
