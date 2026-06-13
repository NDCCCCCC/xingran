// P1-S6 (Phase 32 Wave 2) regression test: GenerateRandomPassword must
// produce a uniform distribution over its charset (no modulo bias).
//
// Background: an earlier version used `randomByte[0] % len(charset)` to pick
// a character index, which is biased when the byte range (256) is not an
// exact multiple of the charset length. For charset length 70, the first
// 46 charset indices get a ~20% higher probability than the last 24.
//
// The fix in commit 07f210c switched to `crypto/rand.Int` rejection sampling,
// which produces a uniform distribution in [0, charsetLen).
//
// This test is a per-character chi-square goodness-of-fit check across the
// 72-character charset. It is intentionally loose (threshold 130) so it stays
// fast and stable on developer hardware while still catching a real 20%
// modulo-bias regression (which would push chi-square to several hundred).
package security

import "testing"

// TestGenerateRandomPassword_NoBiasDistribution verifies that
// GenerateRandomPassword samples uniformly from its 72-character charset.
//
// The most direct way to detect a modulo-bias regression is to count how
// often each of the 72 individual characters appears, then run a chi-square
// goodness-of-fit test against the uniform null hypothesis. A modulo bias
// would cause some characters to appear significantly more often than others
// (the first `256 mod 72 = 40` characters would skew high, for example).
//
// The test uses 5000 passwords of length 16 (80,000 characters total), giving
// an expected count of ~1111 per character. The chi-square statistic for
// 71 degrees of freedom at alpha=0.01 is ~99.4, so we use a generous
// threshold to keep the test stable on developer hardware.
func TestGenerateRandomPassword_NoBiasDistribution(t *testing.T) {
	const (
		passwords = 5000
		length    = 16
	)

	// Charset must mirror GenerateRandomPassword's literal exactly:
	//
	//	"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	charsetSize := len(charset)

	// Tally: index i = position in charset, value = how many times that
	// character appeared across all generated passwords.
	counts := make([]int, charsetSize)
	totalChars := 0

	for i := 0; i < passwords; i++ {
		pwd, err := GenerateRandomPassword(length)
		if err != nil {
			t.Fatalf("GenerateRandomPassword returned error: %v", err)
		}
		if len(pwd) != length {
			t.Fatalf("password length = %d, want %d", len(pwd), length)
		}
		for j := 0; j < len(pwd); j++ {
			idx := indexOf(charset, pwd[j])
			if idx < 0 {
				t.Fatalf("character %q at password %d position %d is not in charset", pwd[j], i, j)
			}
			counts[idx]++
			totalChars++
		}
	}

	if totalChars != passwords*length {
		t.Fatalf("totalChars = %d, want %d", totalChars, passwords*length)
	}

	// Chi-square goodness-of-fit test.
	expected := float64(totalChars) / float64(charsetSize)
	var chiSquare float64
	// Track min/max to surface extremes in the failure message.
	minCount := totalChars
	maxCount := 0
	var minChar, maxChar byte
	for i := 0; i < len(charset); i++ {
		c := charset[i]
		diff := float64(counts[i]) - expected
		chiSquare += (diff * diff) / expected
		if counts[i] < minCount {
			minCount = counts[i]
			minChar = c
		}
		if counts[i] > maxCount {
			maxCount = counts[i]
			maxChar = c
		}
	}

	// Critical value for 71 degrees of freedom at alpha=0.001 is ~115.
	// We use 130 to be safe and avoid flakes on slow / busy CI machines.
	// A modulo bias would push chi-square to several hundred or thousand.
	const chiSquareThreshold = 130.0

	if chiSquare > chiSquareThreshold {
		t.Errorf("chi-square = %.1f > %.1f threshold (modulo bias regression suspected). "+
			"min=%q(%d), max=%q(%d), expected=%.1f",
			chiSquare, chiSquareThreshold,
			minChar, minCount, maxChar, maxCount, expected)
	}
}

// indexOf returns the index of c in s, or -1 if not present. s is expected
// to be ASCII-only.
func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
