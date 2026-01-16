package models

import (
	"time"
)

// ServerInfo 服务器信息
type ServerInfo struct {
	BaseModel
	HostName        string     `json:"hostName" gorm:"size:128;not null;comment:主机名"`
	OS              string     `json:"os" gorm:"size:64;comment:操作系统"`
	Arch            string     `json:"arch" gorm:"size:32;comment:系统架构"`
	CPUCount        int        `json:"cpuCount" gorm:"comment:CPU核心数"`
	TotalMemory     uint64     `json:"totalMemory" gorm:"comment:总内存(字节)"`
	AvailableMemory uint64     `json:"availableMemory" gorm:"comment:可用内存(字节)"`
	DiskTotal       uint64     `json:"diskTotal" gorm:"comment:磁盘总容量(字节)"`
	DiskAvailable   uint64     `json:"diskAvailable" gorm:"comment:磁盘可用容量(字节)"`
	Status          int        `json:"status" gorm:"default:0;comment:状态:0=正常,1=异常"`
	LastActiveAt    *time.Time `json:"lastActiveAt" gorm:"comment:最后活跃时间"`
}

func (ServerInfo) TableName() string {
	return "sys_server_info"
}

// SystemMetrics 系统性能指标
type SystemMetrics struct {
	BaseModel
	ServerID     string    `json:"serverId" gorm:"size:36;not null;index;comment:服务器ID"`
	CPUUsage     float64   `json:"cpuUsage" gorm:"comment:CPU使用率(%)"`
	MemoryUsage  float64   `json:"memoryUsage" gorm:"comment:内存使用率(%)"`
	DiskUsage    float64   `json:"diskUsage" gorm:"comment:磁盘使用率(%)"`
	NetworkRx    uint64    `json:"networkRx" gorm:"comment:网络接收字节数"`
	NetworkTx    uint64    `json:"networkTx" gorm:"comment:网络发送字节数"`
	ProcessCount int       `json:"processCount" gorm:"comment:进程数量"`
	LoadAverage  float64   `json:"loadAverage" gorm:"comment:系统负载"`
	Timestamp    time.Time `json:"timestamp" gorm:"index;comment:采集时间"`
}

func (SystemMetrics) TableName() string {
	return "sys_system_metrics"
}

// CacheInfo 缓存信息
type CacheInfo struct {
	Key       string    `json:"key" gorm:"primaryKey;size:255;comment:缓存键"`
	Value     string    `json:"value" gorm:"type:text;comment:缓存值"`
	TTL       int64     `json:"ttl" gorm:"comment:过期时间(秒)"`
	Size      int64     `json:"size" gorm:"comment:缓存大小(字节)"`
	Type      string    `json:"type" gorm:"size:32;comment:缓存类型"`
	Location  string    `json:"location" gorm:"size:16;comment:缓存位置:l1/l2/both"` // 新增
	CreatedAt time.Time `json:"createdAt" gorm:"comment:创建时间"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"comment:更新时间"`
}

func (CacheInfo) TableName() string {
	return "sys_cache_info"
}

// CacheStats 缓存统计
type CacheStats struct {
	BaseModel
	CacheType    string    `json:"cacheType" gorm:"size:32;not null;comment:缓存类型"`
	HitCount     int64     `json:"hitCount" gorm:"comment:命中次数"`
	MissCount    int64     `json:"missCount" gorm:"comment:未命中次数"`
	HitRate      float64   `json:"hitRate" gorm:"comment:命中率"`
	TotalMemory  int64     `json:"totalMemory" gorm:"comment:总内存占用"`
	UsedMemory   int64     `json:"usedMemory" gorm:"comment:已用内存"`
	KeyCount     int64     `json:"keyCount" gorm:"comment:键值对数量"`
	ExpiredCount int64     `json:"expiredCount" gorm:"comment:过期键数量"`
	CollectTime  time.Time `json:"collectTime" gorm:"comment:采集时间"`
}

func (CacheStats) TableName() string {
	return "sys_cache_stats"
}
