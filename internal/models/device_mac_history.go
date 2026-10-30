package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/pkg/normalize"
	"gorm.io/gorm"
)

// MACEventType MAC地址历史事件类型枚举
type MACEventType string

const (
	EventAppeared    MACEventType = "appeared"    // MAC地址出现
	EventDisappeared MACEventType = "disappeared" // MAC地址消失
	EventMoved       MACEventType = "moved"       // MAC地址移动到其他接口
	EventVLANChanged MACEventType = "vlan_changed" // VLAN变更
)

// DeviceMACHistory 设备MAC地址历史记录模型
type DeviceMACHistory struct {
	ID                  string       `gorm:"type:uuid;primary_key" json:"id"`
	DeviceID            string       `gorm:"type:uuid;not null" json:"deviceId"`
	DeviceNameSnapshot  string       `gorm:"size:100" json:"deviceNameSnapshot"` // 设备名称快照，非外键
	MACAddress          string       `gorm:"size:30;not null" json:"macAddress"`
	InterfaceName       string       `gorm:"size:100;not null" json:"interfaceName"`
	VLANID              *int         `json:"vlanId,omitempty"`
	EventType           MACEventType `gorm:"size:20;not null" json:"eventType"`
	FirstSeen   time.Time `gorm:"not null" json:"firstSeen"`   // 记录首次出现时间
	LastSeen    time.Time `gorm:"not null" json:"lastSeen"`    // 记录最后看到时间
	CollectedAt time.Time `gorm:"not null" json:"collectedAt"` // 采集时间戳
	CreatedAt           time.Time    `json:"createdAt"`
}

// TableName 设置表名
func (DeviceMACHistory) TableName() string {
	return "sys_device_mac_history"
}

// BeforeCreate GORM 钩子：在创建记录前自动生成 UUID + 强制归一化 MAC/接口名
//
// 2026-07-01 根治(port-mac-format-unify): MAC 轨迹表此前接口名全是全称
// (BuildMACStateMap 未归一化 InterfaceName),写入前兜底归一化 MAC(大写+冒号)与
// 接口名(大写短名),保证轨迹表格式与 port_status/mac_address 一致。
func (d *DeviceMACHistory) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" || d.ID == "00000000-0000-0000-0000-000000000000" {
		d.ID = uuid.New().String()
	}
	d.MACAddress = normalize.MACAddress(d.MACAddress)
	d.InterfaceName = normalize.InterfaceName(d.InterfaceName)
	return nil
}

// AfterFind GORM 钩子：修正 pgx 读取 timestamp without time zone 强制赋予 UTC loc 的时区 bug。
//
// 背景: sys_device_mac_history 的 first_seen/last_seen/collected_at 是 timestamp without
// time zone,采集器 time.Now()(UTC+8) 写入,DB 存北京时间裸值(如 13:01 = 北京 13:01)。
// 但 pgx 驱动读取 timestamp without time zone 时无视 session timezone=Asia/Shanghai,
// 强制赋予 UTC loc → JSON "...Z" → 前端 dayjs 当 UTC 再 +8 → 显示成 21:01(实际 13:01)。
//
// 修复: 此处把 wall clock(年月日时分秒)不变、loc 重塑为 Local,让 JSON 输出 "+08:00",
// 前端 dayjs/formatDateTime/ECharts 全部正确显示北京时间。
//
// 为何在读取层而非 schema 层修: first_seen 是分区键(PARTITION BY RANGE first_seen),
// PG 禁止 ALTER 分区键列类型(SQLSTATE 42P16),无法改 timestamptz。
func (d *DeviceMACHistory) AfterFind(tx *gorm.DB) error {
	d.FirstSeen = LocalWallClock(d.FirstSeen)
	d.LastSeen = LocalWallClock(d.LastSeen)
	d.CollectedAt = LocalWallClock(d.CollectedAt)
	if !d.CreatedAt.IsZero() {
		d.CreatedAt = LocalWallClock(d.CreatedAt)
	}
	return nil
}

// LocalWallClock 把 time.Time 的 loc 重塑为 time.Local,保持 wall clock(年月日时分秒纳秒)不变。
// 用于修正 pgx 读 timestamp without time zone 强制 UTC loc 的时区偏移(见 DeviceMACHistory.AfterFind)。
func LocalWallClock(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
}
