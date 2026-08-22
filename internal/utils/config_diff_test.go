package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffConfig_Same(t *testing.T) {
	d := DiffConfig("same", "same")
	require.NotNil(t, d)
	assert.False(t, d.HasChanges)
}

func TestDiffConfig_Different(t *testing.T) {
	d := DiffConfig("a=1\nb=2", "a=1\nc=3")
	require.True(t, d.HasChanges)
	assert.NotEmpty(t, d.OldHash)
	assert.NotEmpty(t, d.NewHash)
	assert.Greater(t, d.LinesAdded, 0)
	assert.Greater(t, d.LinesRemoved, 0)
}

func TestCalculateHash(t *testing.T) {
	assert.Equal(t, "", CalculateHash(""))
	h := CalculateHash("abc")
	assert.Len(t, h, 32) // MD5 hex
	assert.Equal(t, h, CalculateHash("abc"))
	assert.NotEqual(t, h, CalculateHash("def"))
}

func TestGetUnifiedDiff_Same(t *testing.T) {
	// QUIRK: 配置相同时 GetUnifiedDiff 返回空字符串(不输出 headers)。
	d := GetUnifiedDiff("x=1", "x=1", 3)
	assert.Equal(t, "", d)
}

func TestGetUnifiedDiff_Different(t *testing.T) {
	d := GetUnifiedDiff("a=1\nb=2", "a=1\nc=3", 3)
	assert.NotEmpty(t, d)
	assert.Contains(t, d, "b=2")
	assert.Contains(t, d, "c=3")
}

func TestGetSideBySideDiff(t *testing.T) {
	d := GetSideBySideDiff("a=1\nb=2", "a=1\nc=3")
	assert.NotEmpty(t, d)
}

func TestNormalizeConfig(t *testing.T) {
	n := NormalizeConfig("  a=1  \n\n  b=2")
	assert.NotEmpty(t, n)
	// 内部至少不为空
	assert.NotEqual(t, "  a=1  \n\n  b=2", n)
}