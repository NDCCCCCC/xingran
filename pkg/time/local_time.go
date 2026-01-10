package timeutil

import (
	"database/sql/driver"
	"fmt"
	"time"
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

// Value 实现 driver.Valuer 接口，用于写入数据库
// 存储时转换为本地时区的时间字符串
func (lt LocalTime) Value() (driver.Value, error) {
	return lt.Time.Format("2006-01-02 15:04:05"), nil
}

// Scan 实现 sql.Scanner 接口，用于从数据库读取
// 读取时解析为本地时区的时间
func (lt *LocalTime) Scan(value interface{}) error {
	if value == nil {
		lt.Time = time.Time{}
		return nil
	}

	// 处理不同类型的输入
	switch v := value.(type) {
	case time.Time:
		// 如果是 time.Time，直接使用（已经是本地时区）
		lt.Time = v
	case []byte:
		// 如果是字节数组，解析为本地时间
		t, err := time.ParseInLocation("2006-01-02 15:04:05.999999", string(v), time.Local)
		if err != nil {
			// 尝试不带微秒的格式
			t, err = time.ParseInLocation("2006-01-02 15:04:05", string(v), time.Local)
			if err != nil {
				return fmt.Errorf("无法解析时间: %w", err)
			}
		}
		lt.Time = t
	case string:
		// 如果是字符串，解析为本地时间
		t, err := time.ParseInLocation("2006-01-02 15:04:05.999999", v, time.Local)
		if err != nil {
			// 尝试不带微秒的格式
			t, err = time.ParseInLocation("2006-01-02 15:04:05", v, time.Local)
			if err != nil {
				return fmt.Errorf("无法解析时间: %w", err)
			}
		}
		lt.Time = t
	default:
		return fmt.Errorf("不支持的时间类型: %T", value)
	}

	return nil
}

// MarshalJSON 实现 json.Marshaler 接口
// 序列化为带本地时区偏移的 RFC3339 格式
func (lt LocalTime) MarshalJSON() ([]byte, error) {
	if lt.Time.IsZero() {
		return []byte("null"), nil
	}
	// 格式化为 RFC3339 并包含时区偏移
	formatted := fmt.Sprintf("\"%s\"", lt.Time.Format("2006-01-02T15:04:05-07:00"))
	return []byte(formatted), nil
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (lt *LocalTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		lt.Time = time.Time{}
		return nil
	}

	// 去掉引号
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	// 解析 RFC3339 格式
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		// 尝试其他格式
		t, err = time.Parse("2006-01-02T15:04:05-07:00", str)
		if err != nil {
			return fmt.Errorf("无法解析JSON时间: %w", err)
		}
	}

	lt.Time = t
	return nil
}
