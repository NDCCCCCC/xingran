package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToInt(t *testing.T) {
	assert.Equal(t, 5, ToInt(5))
	assert.Equal(t, 5, ToInt(5.9))
	assert.Equal(t, 42, ToInt("42"))
	assert.Equal(t, 0, ToInt("not-a-number"))
	assert.Equal(t, 0, ToInt(nil))
	assert.Equal(t, 0, ToInt(true))
}

func TestToInt64(t *testing.T) {
	assert.Equal(t, int64(5), ToInt64(5))
	assert.Equal(t, int64(5), ToInt64(5.9))
	assert.Equal(t, int64(5), ToInt64(int64(5)))
	assert.Equal(t, int64(42), ToInt64("42"))
	assert.Equal(t, int64(0), ToInt64(nil))
	assert.Equal(t, int64(0), ToInt64("bad"))
}

func TestToBoolPtr(t *testing.T) {
	b := ToBoolPtr(true)
	require.NotNil(t, b)
	assert.True(t, *b)

	b = ToBoolPtr("true")
	require.NotNil(t, b)
	assert.True(t, *b)

	b = ToBoolPtr("false")
	require.NotNil(t, b)
	assert.False(t, *b)

	assert.Nil(t, ToBoolPtr("garbage"))
	assert.Nil(t, ToBoolPtr(nil))
}

func TestToIntPtr(t *testing.T) {
	p := ToIntPtr(42)
	require.NotNil(t, p)
	assert.Equal(t, 42, *p)
	assert.Nil(t, ToIntPtr(0))
	assert.Nil(t, ToIntPtr(""))
}

func TestToStringPtr(t *testing.T) {
	assert.Nil(t, ToStringPtr(nil))
	p := ToStringPtr(42)
	require.NotNil(t, p)
	assert.Equal(t, "42", *p)
	p = ToStringPtr("hi")
	require.NotNil(t, p)
	assert.Equal(t, "hi", *p)
	assert.Nil(t, ToStringPtr(""))
}

func TestToStringSlice(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, ToStringSlice([]interface{}{"a", "b"}))
	assert.Equal(t, []string{"a", "b"}, ToStringSlice([]string{"a", "b"}))
	assert.Equal(t, []string{"a", "b"}, ToStringSlice("a,b"))
	assert.Equal(t, []string{"a", "b"}, ToStringSlice(" a , b "))
	assert.Empty(t, ToStringSlice(""))
	assert.Empty(t, ToStringSlice(nil))
	assert.Empty(t, ToStringSlice(123))
}

func TestParseStatusToInt(t *testing.T) {
	assert.Equal(t, 1, ParseStatusToInt(1))
	assert.Equal(t, 1, ParseStatusToInt(1.0))
	assert.Equal(t, 1, ParseStatusToInt("1"))
	assert.Equal(t, 0, ParseStatusToInt(true))
	assert.Equal(t, 1, ParseStatusToInt(false))
	assert.Equal(t, 0, ParseStatusToInt(nil))
	assert.Equal(t, 0, ParseStatusToInt("bad"))
}

func TestDerefHelpers(t *testing.T) {
	s := "x"
	assert.Equal(t, "x", DerefString(&s))
	assert.Equal(t, "", DerefString(nil))

	i := 5
	assert.Equal(t, 5, DerefInt(&i))
	assert.Equal(t, 0, DerefInt(nil))

	b := true
	assert.True(t, DerefBool(&b))
	assert.False(t, DerefBool(nil))
}

func TestToSlicePtrAndBack(t *testing.T) {
	sl := []string{"a", "b"}
	ptrs := ToSlicePtr(sl)
	require.Len(t, ptrs, 2)
	assert.Equal(t, "a", *ptrs[0])

	vals := ToSlice(ptrs)
	assert.Equal(t, []string{"a", "b"}, vals)

	// 带 nil
	vals = ToSlice([]*string{ptrs[0], nil, ptrs[1]})
	assert.Equal(t, []string{"a", "b"}, vals)
}