package timeutil

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

const (
	// 时间格式常量
	datetimeFormat      = "2006-01-02 15:04:05"
	datetimeMicroFormat = "2006-01-02 15:04:05.999999"
	isoFormat           = "2006-01-02T15:04:05-07:00"
)

// LocalTime 本地时间类型，始终以配置的时区存储和读取
// 用于替代标准的 time.Time，避免 GORM 自动转换为 UTC
type LocalTime struct {
	time.Time
}

// NewLocalTime 创建本地时间
func NewLocalTime(t time.Time) LocalTime {
	return LocalTime{Time: t}
}

// NewLocalTimeNow 创建当前本地时间
func NewLocalTimeNow() LocalTime {
	return LocalTime{Time: time.Now()}
}

// Value 实现 driver.Valuer 接口，用于写入数据库
// 存储时转换为本地时区的时间字符串
func (lt LocalTime) Value() (driver.Value, error) {
	return lt.Time.Format(datetimeFormat), nil
}

// Scan 实现 sql.Scanner 接口，用于从数据库读取
// 读取时解析为本地时区的时间
func (lt *LocalTime) Scan(value interface{}) error {
	if value == nil {
		lt.Time = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		lt.Time = v
	case []byte:
		t, err := parseLocalTime(string(v))
		if err != nil {
			return err
		}
		lt.Time = t
	case string:
		t, err := parseLocalTime(v)
		if err != nil {
			return err
		}
		lt.Time = t
	default:
		return fmt.Errorf("不支持的时间类型: %T", value)
	}

	return nil
}

// parseLocalTime 解析本地时间字符串
func parseLocalTime(s string) (time.Time, error) {
	// 尝试带微秒的格式
	t, err := time.ParseInLocation(datetimeMicroFormat, s, time.Local)
	if err == nil {
		return t, nil
	}

	// 尝试不带微秒的格式
	t, err = time.ParseInLocation(datetimeFormat, s, time.Local)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("无法解析时间: %s", s)
}

// MarshalJSON 实现 json.Marshaler 接口
// 序列化为带本地时区偏移的 RFC3339 格式
func (lt LocalTime) MarshalJSON() ([]byte, error) {
	if lt.Time.IsZero() {
		return []byte("null"), nil
	}
	formatted := fmt.Sprintf("\"%s\"", lt.Time.Format(isoFormat))
	return []byte(formatted), nil
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (lt *LocalTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		lt.Time = time.Time{}
		return nil
	}

	str := unquoteString(data)
	if str == "" {
		return fmt.Errorf("无效的JSON时间格式")
	}

	t, err := parseJSONTime(str)
	if err != nil {
		return err
	}

	lt.Time = t
	return nil
}

// unquoteString 去除字符串的引号
func unquoteString(data []byte) string {
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		return str[1 : len(str)-1]
	}
	return str
}

// parseJSONTime 解析JSON时间字符串
func parseJSONTime(str string) (time.Time, error) {
	// 尝试 RFC3339 格式
	t, err := time.Parse(time.RFC3339, str)
	if err == nil {
		return t, nil
	}

	// 尝试自定义 ISO 格式
	t, err = time.Parse(isoFormat, str)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("无法解析JSON时间: %s", str)
}

// String 返回格式化的时间字符串
func (lt LocalTime) String() string {
	return lt.Time.Format(datetimeFormat)
}

// IsZero 检查时间是否为零值
func (lt LocalTime) IsZero() bool {
	return lt.Time.IsZero()
}

// Before 检查是否在另一个时间之前
func (lt LocalTime) Before(other LocalTime) bool {
	return lt.Time.Before(other.Time)
}

// After 检查是否在另一个时间之后
func (lt LocalTime) After(other LocalTime) bool {
	return lt.Time.After(other.Time)
}

// Add 返回增加指定持续时间后的时间
func (lt LocalTime) Add(d time.Duration) LocalTime {
	return LocalTime{Time: lt.Time.Add(d)}
}

// Sub 返回两个时间之间的持续时间
func (lt LocalTime) Sub(other LocalTime) time.Duration {
	return lt.Time.Sub(other.Time)
}

// Format 按指定格式格式化时间
func (lt LocalTime) Format(layout string) string {
	return lt.Time.Format(layout)
}

// ToTime 转换为标准 time.Time
func (lt LocalTime) ToTime() time.Time {
	return lt.Time
}

// ParseLocalTime 从字符串解析本地时间
func ParseLocalTime(s string) (LocalTime, error) {
	t, err := parseLocalTime(strings.TrimSpace(s))
	if err != nil {
		return LocalTime{}, err
	}
	return LocalTime{Time: t}, nil
}
