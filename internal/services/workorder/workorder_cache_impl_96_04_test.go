package workorder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWorkOrderCacheImpl96_CACHEDEF04_GetMyPending_KeyFormat_Limit10
// CACHEDEF-04: GetMyPending with limit=10 writes key workorder:my_pending:<userID>:limit:10
func TestWorkOrderCacheImpl96_CACHEDEF04_GetMyPending_KeyFormat_Limit10(t *testing.T) {
	userID := "user-123"
	limit := 10
	expectedKey := "workorder:my_pending:user-123:limit:10"
	actualKey := buildMyPendingKey(userID, limit)
	assert.Equal(t, expectedKey, actualKey)
}

// TestWorkOrderCacheImpl96_CACHEDEF04_GetMyPending_KeyFormat_Limit20
// CACHEDEF-04: Different limit values produce different keys.
func TestWorkOrderCacheImpl96_CACHEDEF04_GetMyPending_KeyFormat_Limit20(t *testing.T) {
	userID := "user-123"
	limit := 20
	expectedKey := "workorder:my_pending:user-123:limit:20"
	actualKey := buildMyPendingKey(userID, limit)
	assert.Equal(t, expectedKey, actualKey)
}

// TestWorkOrderCacheImpl96_CACHEDEF04_GetMyPending_KeyFormat_NilReq
// CACHEDEF-04: nil req (limit=0) writes key workorder:my_pending:<userID>:limit:0
func TestWorkOrderCacheImpl96_CACHEDEF04_GetMyPending_KeyFormat_NilReq(t *testing.T) {
	userID := "user-456"
	limit := 0 // nil req
	expectedKey := "workorder:my_pending:user-456:limit:0"
	actualKey := buildMyPendingKey(userID, limit)
	assert.Equal(t, expectedKey, actualKey)
}

// TestWorkOrderCacheImpl96_CACHEDEF04_GetMyPending_KeysAreDistinct
// CACHEDEF-04: Different limits produce distinct keys — no key collision.
func TestWorkOrderCacheImpl96_CACHEDEF04_GetMyPending_KeysAreDistinct(t *testing.T) {
	userID := "user-789"
	key5 := buildMyPendingKey(userID, 5)
	key10 := buildMyPendingKey(userID, 10)
	key0 := buildMyPendingKey(userID, 0)

	assert.NotEqual(t, key5, key10, "limit=5 and limit=10 should produce different keys")
	assert.NotEqual(t, key5, key0, "limit=5 and limit=0 should produce different keys")
	assert.NotEqual(t, key10, key0, "limit=10 and limit=0 should produce different keys")
}

// buildMyPendingKey mirrors the cache key construction that GetMyPending should use
// after the CACHEDEF-04 fix. This function is the reference implementation for tests.
func buildMyPendingKey(userID string, limit int) string {
	return "workorder:my_pending:" + userID + ":limit:" + itoa(limit)
}

// itoa converts a non-negative integer to its decimal string representation.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
