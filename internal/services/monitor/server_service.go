package monitor

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/pkg/system"
	"gorm.io/gorm"
)

// ServerService 服务器监控服务接口
type ServerService interface {
	GetServerInfo(ctx context.Context, params ServerInfoParams) ([]*models.ServerInfo, int64, error)
	GetCurrentServerMetrics(ctx context.Context) (*SystemMetricsData, error)
	SaveSystemMetrics(ctx context.Context, metrics *models.SystemMetrics) error
	GetSystemMetricsHistory(ctx context.Context, params MetricsHistoryParams) ([]*models.SystemMetrics, int64, error)
}

// SystemMetricsData 系统指标数据
type SystemMetricsData struct {
	CPUUsage    float64
	MemoryUsage float64
	DiskUsage   float64
	NetworkRx   uint64
	NetworkTx   uint64
	ProcessNum  int
	TotalMemory uint64
	UsedMemory  uint64
	Timestamp   time.Time
}

// ServerInfoParams 服务器信息查询参数
type ServerInfoParams struct {
	Current  int
	PageSize int
	OrderByColumn string
	IsAsc        bool
}

// MetricsHistoryParams 指标历史查询参数
type MetricsHistoryParams struct {
	ServerID  string
	StartTime *string
	EndTime   *string
	Current   int
	PageSize  int
	OrderByColumn string
	IsAsc        bool
}

// MetricsProvider 系统指标提供者接口（支持缓存降级）
type MetricsProvider interface {
	GetServerInfo(ctx context.Context) (map[string]interface{}, error)
	GetCurrentMetrics(ctx context.Context) (*SystemMetricsData, error)
}


// serverAllowedSortFields 服务端排序白名单
var serverAllowedSortFields = map[string]string{
	"timestamp":   "timestamp",
	"cpuUsage":    "cpu_usage",
	"memoryUsage": "memory_usage",
	"diskUsage":   "disk_usage",
	"serverId":    "server_id",
}
// serverServiceImpl 服务器监控服务实现
type serverServiceImpl struct {
	db              *gorm.DB
	metricsProvider MetricsProvider
}

// NewServerService 创建服务器监控服务实例
func NewServerService(db *gorm.DB, provider MetricsProvider) ServerService {
	return &serverServiceImpl{
		db:              db,
		metricsProvider: provider,
	}
}

// GetServerInfo 获取服务器信息（优先从缓存，降级到实时获取）
func (s *serverServiceImpl) GetServerInfo(ctx context.Context, params ServerInfoParams) ([]*models.ServerInfo, int64, error) {
	// 优先从缓存获取
	info, err := s.metricsProvider.GetServerInfo(ctx)
	if err == nil && info != nil {
		server := s.convertToServerInfo(info)
		return []*models.ServerInfo{server}, 1, nil
	}

	// 降级到实时获取
	server, err := s.getCurrentServerInfo()
	if err != nil {
		return nil, 0, fmt.Errorf("获取服务器信息失败: %w", err)
	}

	return []*models.ServerInfo{server}, 1, nil
}

// GetCurrentServerMetrics 获取当前服务器指标（优先从缓存）
func (s *serverServiceImpl) GetCurrentServerMetrics(ctx context.Context) (*SystemMetricsData, error) {
	// 优先从缓存获取
	metrics, err := s.metricsProvider.GetCurrentMetrics(ctx)
	if err == nil {
		return metrics, nil
	}

	// 降级到实时获取
	return s.getCurrentMetricsRealtime()
}

// SaveSystemMetrics 保存系统指标
func (s *serverServiceImpl) SaveSystemMetrics(ctx context.Context, metrics *models.SystemMetrics) error {
	return s.db.Create(metrics).Error
}

// GetSystemMetricsHistory 获取系统指标历史
func (s *serverServiceImpl) GetSystemMetricsHistory(ctx context.Context, params MetricsHistoryParams) ([]*models.SystemMetrics, int64, error) {
	db := s.db.Model(&models.SystemMetrics{})

	if params.ServerID != "" {
		db = db.Where("server_id = ?", params.ServerID)
	}
	if params.StartTime != nil && *params.StartTime != "" {
		db = db.Where("timestamp >= ?", *params.StartTime)
	}
	if params.EndTime != nil && *params.EndTime != "" {
		db = db.Where("timestamp <= ?", *params.EndTime)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var metrics []*models.SystemMetrics
	offset := (params.Current - 1) * params.PageSize
	// Apply server-side sort
	orderClause := "timestamp DESC"
	if params.OrderByColumn != "" {
		if col, ok := serverAllowedSortFields[params.OrderByColumn]; ok {
			direction := "DESC"
			if params.IsAsc {
				direction = "ASC"
			}
			orderClause = fmt.Sprintf("%s %s", col, direction)
		}
	}
	if err := db.Offset(offset).Limit(params.PageSize).Order(orderClause).Find(&metrics).Error; err != nil {
		return nil, 0, err
	}

	return metrics, total, nil
}

// convertToServerInfo 将缓存数据转换为ServerInfo
func (s *serverServiceImpl) convertToServerInfo(info map[string]interface{}) *models.ServerInfo {
	now := time.Now()
	server := &models.ServerInfo{
		Status:       0,
		LastActiveAt: &now,
	}

	if v, ok := info["hostname"].(string); ok {
		server.HostName = v
	}
	if v, ok := info["os"].(string); ok {
		server.OS = v
	}
	if v, ok := info["arch"].(string); ok {
		server.Arch = v
	}
	if v, ok := info["cpu_count"].(float64); ok {
		server.CPUCount = int(v)
	}
	if v, ok := info["total_memory"].(float64); ok {
		server.TotalMemory = uint64(v)
	}
	if v, ok := info["disk_total"].(float64); ok {
		server.DiskTotal = uint64(v)
	}
	if v, ok := info["disk_available"].(float64); ok {
		server.DiskAvailable = uint64(v)
	}
	if v, ok := info["available_memory"].(float64); ok {
		server.AvailableMemory = uint64(v)
	}

	return server
}

// getCurrentServerInfo 获取当前服务器信息（实时）
func (s *serverServiceImpl) getCurrentServerInfo() (*models.ServerInfo, error) {
	metrics, err := system.GetSystemMetrics()
	if err != nil {
		return nil, err
	}

	disks, err := system.GetAllDiskInfo()
	if err != nil {
		return nil, fmt.Errorf("获取磁盘信息失败: %w", err)
	}

	if len(disks) == 0 {
		return nil, fmt.Errorf("未找到任何磁盘信息")
	}

	var totalDiskSize, availableDiskSize uint64
	for _, disk := range disks {
		if disk.Total > 0 && disk.Available <= disk.Total {
			totalDiskSize += disk.Total
			availableDiskSize += disk.Available
		}
	}

	if totalDiskSize == 0 {
		return nil, fmt.Errorf("获取到的磁盘总容量为0")
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "localhost"
	}

	availableMemory := metrics.TotalMemory - metrics.UsedMemory

	return &models.ServerInfo{
		HostName:        hostname,
		OS:              runtime.GOOS,
		Arch:            runtime.GOARCH,
		CPUCount:        runtime.NumCPU(),
		TotalMemory:     metrics.TotalMemory,
		AvailableMemory: availableMemory,
		DiskTotal:       totalDiskSize,
		DiskAvailable:   availableDiskSize,
		Status:          0,
		LastActiveAt:    &metrics.Timestamp,
	}, nil
}

// getCurrentMetricsRealtime 获取当前指标（实时）
func (s *serverServiceImpl) getCurrentMetricsRealtime() (*SystemMetricsData, error) {
	metrics, err := system.GetSystemMetrics()
	if err != nil {
		return nil, err
	}

	return &SystemMetricsData{
		CPUUsage:    metrics.CPUUsage,
		MemoryUsage: metrics.MemoryUsage,
		DiskUsage:   metrics.DiskUsage,
		NetworkRx:   metrics.NetworkRx,
		NetworkTx:   metrics.NetworkTx,
		ProcessNum:  metrics.ProcessNum,
		TotalMemory: metrics.TotalMemory,
		UsedMemory:  metrics.UsedMemory,
		Timestamp:   metrics.Timestamp,
	}, nil
}
