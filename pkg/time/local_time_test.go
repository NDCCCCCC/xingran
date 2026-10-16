package timeutil

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalTime_Constructors(t *testing.T) {
	now := time.Now()
	lt := NewLocalTime(now)
	assert.True(t, lt.Time.Equal(now))

	now2 := NewLocalTimeNow()
	assert.WithinDuration(t, time.Now(), now2.Time, time.Second)
}

func TestLocalTime_Value(t *testing.T) {
	now := time.Date(2024, 6, 15, 10, 30, 45, 0, time.Local)
	lt := NewLocalTime(now)
	v, err := lt.Value()
	require.NoError(t, err)
	assert.Equal(t, "2024-06-15 10:30:45", v.(string))
}

func TestLocalTime_Scan(t *testing.T) {
	var lt LocalTime

	// nil
	require.NoError(t, lt.Scan(nil))
	assert.True(t, lt.Time.IsZero())

	// time.Time
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
	require.NoError(t, lt.Scan(now))
	assert.True(t, lt.Time.Equal(now))

	// []byte
	lt = LocalTime{}
	require.NoError(t, lt.Scan([]byte("2024-12-31 23:59:59")))
	assert.Equal(t, 2024, lt.Time.Year())

	// []byte 微秒
	lt = LocalTime{}
	require.NoError(t, lt.Scan([]byte("2024-12-31 23:59:59.123456")))
	assert.Equal(t, 2024, lt.Time.Year())

	// string
	lt = LocalTime{}
	require.NoError(t, lt.Scan("2024-06-01 12:00:00"))
	assert.Equal(t, 2024, lt.Time.Year())

	// 默认 → err
	err := lt.Scan(123)
	require.Error(t, err)
}

func TestLocalTime_ParseLocalTime(t *testing.T) {
	lt, err := ParseLocalTime("2024-06-15 10:30:45")
	require.NoError(t, err)
	assert.Equal(t, 2024, lt.Time.Year())

	// 带前后空格
	lt, err = ParseLocalTime("  2024-06-15 10:30:45  ")
	require.NoError(t, err)
	assert.Equal(t, 10, lt.Time.Hour())

	// 微秒格式
	lt, err = ParseLocalTime("2024-06-15 10:30:45.123456")
	require.NoError(t, err)
	assert.Equal(t, 10, lt.Time.Hour())

	// 无效
	_, err = ParseLocalTime("not a date")
	require.Error(t, err)
}

func TestLocalTime_JSON(t *testing.T) {
	// Zero → null
	lt := LocalTime{}
	b, err := json.Marshal(lt)
	require.NoError(t, err)
	assert.Equal(t, "null", string(b))

	// 非零 → RFC3339 with offset
	now := time.Date(2024, 6, 15, 10, 30, 45, 0, time.Local)
	lt = NewLocalTime(now)
	b, err = json.Marshal(lt)
	require.NoError(t, err)
	s := string(b)
	assert.True(t, strings.HasPrefix(s, "\"2024-06-15T10:30:45"), "应包含 ISO 时间前缀")

	// Unmarshal null
	var lt2 LocalTime
	require.NoError(t, json.Unmarshal([]byte("null"), &lt2))
	assert.True(t, lt2.Time.IsZero())

	// Unmarshal RFC3339
	require.NoError(t, json.Unmarshal([]byte(s), &lt2))
	assert.Equal(t, 2024, lt2.Time.Year())

	// Unmarshal 失败
	var lt3 LocalTime
	require.Error(t, json.Unmarshal([]byte(`""`), &lt3))
}

func TestLocalTime_StringIsZeroBeforeAfter(t *testing.T) {
	t1 := NewLocalTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local))
	t2 := NewLocalTime(time.Date(2024, 12, 31, 0, 0, 0, 0, time.Local))

	assert.True(t, t1.Before(t2))
	assert.True(t, t2.After(t1))
	assert.False(t, t1.IsZero())

	zero := LocalTime{}
	assert.True(t, zero.IsZero())
	assert.Equal(t, "0001-01-01 00:00:00", zero.String())
}

func TestLocalTime_AddSubFormatToTime(t *testing.T) {
	t1 := NewLocalTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local))

	added := t1.Add(24 * time.Hour)
	assert.Equal(t, 2024, added.Time.Year())
	assert.Equal(t, time.Hour*24, added.Time.Sub(t1.Time))

	d := t1.Sub(added)
	assert.Equal(t, -time.Hour*24, d)

	assert.Equal(t, "2024", t1.Format("2006"))
	assert.True(t, t1.ToTime().Equal(t1.Time))
}