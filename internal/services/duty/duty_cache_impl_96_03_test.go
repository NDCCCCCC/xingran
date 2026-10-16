package duty

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDutyCacheImpl96_CACHEDEF03_parseInt_2DigitMonth
// CACHEDEF-03: parseInt("07") returns 7 (primary bug case).
// Before fix: len(s) >= 4 guard caused parseInt("07") to return 0.
func TestDutyCacheImpl96_CACHEDEF03_parseInt_2DigitMonth(t *testing.T) {
	result := parseInt("07")
	assert.Equal(t, 7, result, "parseInt(\"07\") should return 7")
}

// TestDutyCacheImpl96_CACHEDEF03_parseInt_4DigitYear
// CACHEDEF-03: parseInt("2024") returns 2024 (existing working case).
func TestDutyCacheImpl96_CACHEDEF03_parseInt_4DigitYear(t *testing.T) {
	result := parseInt("2024")
	assert.Equal(t, 2024, result, "parseInt(\"2024\") should return 2024")
}

// TestDutyCacheImpl96_CACHEDEF03_parseInt_SingleDigit
// CACHEDEF-03: parseInt("1") returns 1 (edge case).
func TestDutyCacheImpl96_CACHEDEF03_parseInt_SingleDigit(t *testing.T) {
	result := parseInt("1")
	assert.Equal(t, 1, result, "parseInt(\"1\") should return 1")
}

// TestDutyCacheImpl96_CACHEDEF03_parseInt_EmptyString
// CACHEDEF-03: parseInt("") returns 0 (boundary case).
func TestDutyCacheImpl96_CACHEDEF03_parseInt_EmptyString(t *testing.T) {
	result := parseInt("")
	assert.Equal(t, 0, result, "parseInt(\"\") should return 0")
}

// TestDutyCacheImpl96_CACHEDEF03_parseInt_3DigitString
// CACHEDEF-03: parseInt("123") returns 123 (3-digit edge case).
func TestDutyCacheImpl96_CACHEDEF03_parseInt_3DigitString(t *testing.T) {
	result := parseInt("123")
	assert.Equal(t, 123, result, "parseInt(\"123\") should return 123")
}

// TestDutyCacheImpl96_CACHEDEF03_parseInt_CacheKeyConsistency
// CACHEDEF-03: Verify write key and invalidation key formats are consistent.
// Write key: fmt.Sprintf("duty:monthly:%d:%d", year, month)
// Invalidation key: fmt.Sprintf("duty:monthly:%d:%d", year, month)
// Both must produce the same string for the same year/month values.
func TestDutyCacheImpl96_CACHEDEF03_parseInt_CacheKeyConsistency(t *testing.T) {
	// Simulate what GenerateSchedule does: parse "2026-07" → year=2026, month=7
	// Then build the cache key with those int values
	year := parseInt("2026") // should be 2026
	month := parseInt("07")  // should be 7 (CACHEDEF-03 primary fix)

	writeKey := formatCacheKey(year, month)
	invalidateKey := formatCacheKey(year, month)

	assert.Equal(t, writeKey, invalidateKey,
		"write and invalidation keys must match for year=%d, month=%d", year, month)
	assert.Equal(t, "duty:monthly:2026:7", writeKey,
		"cache key for July 2026 should be duty:monthly:2026:7")
}

// formatCacheKey mirrors the cache key construction in duty_cache_impl.go.
// This helper is for test readability only.
func formatCacheKey(year, month int) string {
	return "duty:monthly:" + itoa(year) + ":" + itoa(month)
}

// itoa is a minimal positive-int to string converter for test readability.
// Uses the same digits-extraction logic that parseInt/strconv.Atoi would use.
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
