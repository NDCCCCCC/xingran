package services

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// DeviceInfoCollectionService 设备信息采集服务
// 统一负责设备详细信息的采集和更新（纯异步架构）
type DeviceInfoCollectionService struct {
	db             *gorm.DB
	deviceExecutor *device.DeviceExecutor
	taskQueue      chan string     // 统一任务队列
	workerCount    int             // 工作协程数
	stopChan       chan struct{}   // 停止信号
	wg             sync.WaitGroup  // 等待组
	isRunning      bool            // 是否正在运行
	mu             sync.RWMutex    // 读写锁
}

// NewDeviceInfoCollectionService 创建设备信息采集服务
func NewDeviceInfoCollectionService(db *gorm.DB, deviceExecutor *device.DeviceExecutor) *DeviceInfoCollectionService {
	svc := &DeviceInfoCollectionService{
		db:             db,
		deviceExecutor: deviceExecutor,
		taskQueue:      make(chan string, 1000), // 缓冲队列，最多1000个任务
		workerCount:    5,                        // 默认5个工作协程
		stopChan:       make(chan struct{}),
	}
	// 从数据库加载配置
	svc.loadConfigFromDB()
	return svc
}

// loadConfigFromDB 从数据库加载配置
func (s *DeviceInfoCollectionService) loadConfigFromDB() {
	// 读取设备监控并发数配置
	var config models.Config
	err := s.db.Where("config_key = ?", "network.device.monitor.concurrent").First(&config).Error
	if err == nil && config.ConfigValue != "" {
		if concurrent, parseErr := strconv.Atoi(config.ConfigValue); parseErr == nil && concurrent > 0 {
			s.workerCount = concurrent
			log.Printf("[设备信息采集] 从数据库读取并发数配置: %d", s.workerCount)
		}
	}
}

// Start 启动后台采集服务
func (s *DeviceInfoCollectionService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("服务已经在运行")
	}

	s.isRunning = true
	log.Printf("启动设备信息采集服务，工作协程数: %d", s.workerCount)

	// 启动工作协程
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i)
	}

	// 启动待处理任务恢复协程
	s.wg.Add(1)
	go s.recoverPendingTasks(ctx)

	return nil
}

// Stop 停止服务
func (s *DeviceInfoCollectionService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	log.Println("正在停止设备信息采集服务...")
	close(s.stopChan)
	s.wg.Wait()
	s.isRunning = false
	log.Println("设备信息采集服务已停止")
}

// Enqueue 将设备ID加入采集队列（统一入队接口）
// 设备创建和定时任务都调用这个方法
func (s *DeviceInfoCollectionService) Enqueue(deviceID string) error {
	// 检查是否已有待执行或执行中的任务
	var existingTask models.DeviceEnrichmentTask
	err := s.db.Where("device_id = ? AND status IN (?)", deviceID, []models.EnrichmentStatus{
		models.EnrichmentStatusPending,
		models.EnrichmentStatusRunning,
	}).First(&existingTask).Error

	if err == nil {
		// 已存在任务，不再重复创建
		return nil
	}

	// 创建新的采集任务
	task := &models.DeviceEnrichmentTask{
		DeviceID: deviceID,
		Status:   models.EnrichmentStatusPending,
	}

	if err := s.db.Create(task).Error; err != nil {
		return fmt.Errorf("创建采集任务失败: %w", err)
	}

	// 加入队列
	select {
	case s.taskQueue <- deviceID:
		return nil
	default:
		return fmt.Errorf("任务队列已满，请稍后再试")
	}
}

// EnqueueAllOnlineDevices 将所有在线设备加入队列（定时任务调用）
func (s *DeviceInfoCollectionService) EnqueueAllOnlineDevices(ctx context.Context) error {
	// 获取所有在线设备
	var devices []models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("status = ?", models.DeviceStatusOnline).Find(&devices).Error; err != nil {
		return fmt.Errorf("查询在线设备失败: %w", err)
	}

	log.Printf("开始将 %d 个在线设备加入采集队列", len(devices))

	enqueueCount := 0
	for _, device := range devices {
		// 跳过无凭证的设备
		if device.CredentialID == nil || *device.CredentialID == "" {
			continue
		}

		if err := s.Enqueue(device.ID); err != nil {
			log.Printf("加入队列失败 [设备ID=%s]: %v", device.ID, err)
			continue
		}
		enqueueCount++
	}

	log.Printf("成功将 %d 个设备加入采集队列", enqueueCount)
	return nil
}

// worker 工作协程
func (s *DeviceInfoCollectionService) worker(ctx context.Context, workerIndex int) {
	defer s.wg.Done()

	log.Printf("设备信息采集工作协程 #%d 启动", workerIndex)

	for {
		select {
		case deviceID := <-s.taskQueue:
			s.processTask(ctx, deviceID)
		case <-s.stopChan:
			log.Printf("设备信息采集工作协程 #%d 停止", workerIndex)
			return
		case <-ctx.Done():
			log.Printf("设备信息采集工作协程 #%d 上下文取消", workerIndex)
			return
		}
	}
}

// recoverPendingTasks 恢复待处理任务
func (s *DeviceInfoCollectionService) recoverPendingTasks(ctx context.Context) {
	defer s.wg.Done()

	log.Println("开始恢复待处理的设备信息采集任务...")

	var tasks []models.DeviceEnrichmentTask
	if err := s.db.Where("status = ?", models.EnrichmentStatusPending).Find(&tasks).Error; err != nil {
		log.Printf("查询待处理任务失败: %v", err)
		return
	}

	log.Printf("找到 %d 个待处理任务", len(tasks))

	for _, task := range tasks {
		select {
		case s.taskQueue <- task.DeviceID:
			log.Printf("恢复任务: 设备ID=%s", task.DeviceID)
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// processTask 处理单个采集任务
func (s *DeviceInfoCollectionService) processTask(ctx context.Context, deviceID string) {
	log.Printf("开始处理设备信息采集任务: 设备ID=%s", deviceID)

	// 获取任务
	var task models.DeviceEnrichmentTask
	if err := s.db.Where("device_id = ? AND status = ?", deviceID, models.EnrichmentStatusPending).First(&task).Error; err != nil {
		log.Printf("获取采集任务失败: 设备ID=%s, 错误=%v", deviceID, err)
		return
	}

	// 获取设备信息
	var device models.NetworkDevice
	if err := s.db.Where("id = ?", deviceID).First(&device).Error; err != nil {
		s.markTaskFailed(&task, fmt.Sprintf("获取设备信息失败: %v", err))
		return
	}

	// 更新任务状态为执行中
	now := time.Now()
	task.Status = models.EnrichmentStatusRunning
	task.StartedAt = &now
	s.db.Save(&task)

	// 采集设备信息
	info, err := s.CollectDeviceInfo(ctx, &device)
	if err != nil {
		s.markTaskFailed(&task, fmt.Sprintf("采集设备信息失败: %v", err))
		return
	}

	// 更新设备信息（4个字段）
	s.updateDeviceInfo(&device, info)

	// 标记任务成功
	s.markTaskSuccess(&task, info)

	log.Printf("设备信息采集任务完成: 设备ID=%s, 型号=%s", deviceID, info.Model)
}

// CollectDeviceInfo 采集设备详细信息（统一的采集逻辑）
func (s *DeviceInfoCollectionService) CollectDeviceInfo(ctx context.Context, device *models.NetworkDevice) (*DeviceInfo, error) {
	// 获取凭证
	if device.CredentialID == nil || *device.CredentialID == "" {
		return nil, fmt.Errorf("设备未配置授权凭证")
	}

	// 使用连接池获取连接
	pool := s.deviceExecutor.GetScheduler().GetConnectionPool()
	conn, err := pool.GetConnection(ctx, device.ID)
	if err != nil {
		return nil, fmt.Errorf("获取设备连接失败: %w", err)
	}

	if !conn.Acquire() {
		return nil, fmt.Errorf("无法获取连接锁")
	}
	defer conn.Release()

	wrapper := conn.GetWrapper()

	// 确定命令列表
	commands := s.getCommandsByVendor(device.Vendor)

	info := &DeviceInfo{}

	// 执行命令并解析
	for _, cmd := range commands {
		response, err := wrapper.SendCommand(cmd, true)
		if err != nil {
			log.Printf("执行命令失败 [设备=%s, 命令=%s]: %v", device.DeviceName, cmd, err)
			continue
		}
		s.parseDeviceInfo(response.Result, device.Vendor, info)
	}

	return info, nil
}

// getCommandsByVendor 根据厂商获取命令列表
func (s *DeviceInfoCollectionService) getCommandsByVendor(vendor models.DeviceVendor) []string {
	switch vendor {
	case models.VendorHuawei, models.VendorH3C:
		return []string{"display version", "display device"}
	case models.VendorRuijie, models.VendorMaipu:
		return []string{"show version"}
	default:
		return []string{"show version"}
	}
}

// updateDeviceInfo 更新设备信息
func (s *DeviceInfoCollectionService) updateDeviceInfo(device *models.NetworkDevice, info *DeviceInfo) {
	updates := map[string]interface{}{}

	if info.Model != "" {
		updates["model"] = info.Model
	}
	if info.SerialNumber != "" {
		updates["serial_number"] = info.SerialNumber
	}
	if info.SoftwareVersion != "" {
		updates["software_version"] = info.SoftwareVersion
	}
	if info.Uptime != "" {
		updates["uptime"] = info.Uptime
	}

	if len(updates) > 0 {
		now := time.Now()
		updates["updated_at"] = now
		if err := s.db.Model(device).Updates(updates).Error; err != nil {
			log.Printf("更新设备信息失败: %v", err)
		}
	}
}

// markTaskSuccess 标记任务成功
func (s *DeviceInfoCollectionService) markTaskSuccess(task *models.DeviceEnrichmentTask, info *DeviceInfo) {
	completedAt := time.Now()
	task.Status = models.EnrichmentStatusSuccess
	task.CompletedAt = &completedAt

	// 保存采集到的信息
	if info.Model != "" {
		task.EnrichedModel = &info.Model
	}
	if info.SerialNumber != "" {
		task.EnrichedSerialNumber = &info.SerialNumber
	}
	if info.SoftwareVersion != "" {
		task.EnrichedSoftwareVer = &info.SoftwareVersion
	}
	if info.Uptime != "" {
		task.EnrichedUptime = &info.Uptime
	}

	s.db.Save(task)
}

// markTaskFailed 标记任务失败
func (s *DeviceInfoCollectionService) markTaskFailed(task *models.DeviceEnrichmentTask, errorMsg string) {
	completedAt := time.Now()
	task.Status = models.EnrichmentStatusFailed
	task.CompletedAt = &completedAt
	task.ErrorMessage = errorMsg
	s.db.Save(task)
	log.Printf("设备信息采集任务失败: 设备ID=%s, 错误=%s", task.DeviceID, errorMsg)
}

// GetTaskStatus 获取设备采集任务状态
func (s *DeviceInfoCollectionService) GetTaskStatus(ctx context.Context, deviceID string) (*models.DeviceEnrichmentTask, error) {
	var task models.DeviceEnrichmentTask
	if err := s.db.WithContext(ctx).Where("device_id = ?", deviceID).Order("created_at DESC").First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 没有任务
		}
		return nil, fmt.Errorf("查询任务状态失败: %w", err)
	}
	return &task, nil
}

// ================================
// 解析逻辑（从 DeviceMonitorService 迁移）
// ================================

// DeviceInfo 设备详细信息
type DeviceInfo struct {
	Model           string
	SerialNumber    string
	SoftwareVersion string
	Uptime          string
}

// parseDeviceInfo 解析设备信息
func (s *DeviceInfoCollectionService) parseDeviceInfo(output string, vendor models.DeviceVendor, info *DeviceInfo) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		switch vendor {
		case models.VendorHuawei, models.VendorH3C:
			s.parseHuaweiDeviceInfo(line, info)
		case models.VendorRuijie, models.VendorMaipu:
			s.parseRuijieDeviceInfo(line, info)
		default:
			s.parseGenericDeviceInfo(line, info)
		}
	}
}

// parseHuaweiDeviceInfo 解析华为/H3C设备信息
func (s *DeviceInfoCollectionService) parseHuaweiDeviceInfo(line string, info *DeviceInfo) {
	// 解析型号
	// 示例: H3C S5120-SI52P-EI
	//       Huawei Versatile Routing Platform Software
	if strings.Contains(line, "H3C") || strings.Contains(line, "Huawei") {
		if strings.Contains(line, "S5") || strings.Contains(line, "S6") || strings.Contains(line, "S31") || strings.Contains(line, "S51") || strings.Contains(line, "S65") {
			fields := strings.Fields(line)
			for i, f := range fields {
				if strings.HasPrefix(f, "S") && len(f) > 3 {
					info.Model = strings.Join(fields[i:], " ")
					break
				}
			}
		}
	}

	// 解析版本号
	// 示例: Version 5.20, Release 2432P02
	if strings.HasPrefix(line, "Version") || strings.Contains(line, "Software Version") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if f == "Version" && i+1 < len(fields) {
				info.SoftwareVersion = fields[i+1]
				break
			}
		}
	}

	// 解析序列号
	// 示例: DEVICE_SERIAL_NUMBER : xxxxx
	if strings.Contains(line, "Serial") || strings.Contains(line, "DEVICE_SERIAL_NUMBER") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if (f == "NUMBER" || f == "Serial") && i+1 < len(fields) {
				if fields[i+1] != ":" {
					info.SerialNumber = fields[i+1]
				} else if i+2 < len(fields) {
					info.SerialNumber = fields[i+2]
				}
				break
			}
		}
	}

	// 解析运行时间
	// 示例: Uptime is 10 days, 2 hours, 30 minutes
	if strings.Contains(line, "Uptime") || strings.Contains(line, "uptime") {
		if idx := strings.Index(line, "is"); idx > 0 {
			info.Uptime = strings.TrimSpace(line[idx+2:])
		}
	}
}

// parseRuijieDeviceInfo 解析锐捷/迈普设备信息
func (s *DeviceInfoCollectionService) parseRuijieDeviceInfo(line string, info *DeviceInfo) {
	// 解析型号
	// 示例: Ruijie Networks Switch S5750
	if strings.Contains(line, "S5750") || strings.Contains(line, "S6220") || strings.Contains(line, "N180") {
		fields := strings.Fields(line)
		for _, f := range fields {
			if strings.HasPrefix(f, "S") || strings.HasPrefix(f, "N") {
				info.Model = f
				break
			}
		}
	}

	// 解析版本号
	// 示例: Software Version 10.4(3b5)
	if strings.Contains(line, "Software Version") || strings.Contains(line, "SW Version") {
		if idx := strings.Index(line, "Version"); idx > 0 {
			info.SoftwareVersion = strings.TrimSpace(line[idx+7:])
		}
	}

	// 解析序列号
	if strings.Contains(line, "System Serial") || strings.Contains(line, "Serial Number") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if (f == "Number" || f == "Serial") && i+1 < len(fields) {
				if fields[i+1] != ":" {
					info.SerialNumber = fields[i+1]
				} else if i+2 < len(fields) {
					info.SerialNumber = fields[i+2]
				}
				break
			}
		}
	}

	// 解析运行时间
	if strings.Contains(line, "uptime") || strings.Contains(line, "Uptime") {
		if idx := strings.Index(line, "is"); idx > 0 {
			info.Uptime = strings.TrimSpace(line[idx+2:])
		}
	}
}

// parseGenericDeviceInfo 解析通用设备信息
func (s *DeviceInfoCollectionService) parseGenericDeviceInfo(line string, info *DeviceInfo) {
	// 通用解析逻辑
	if strings.Contains(line, "Version") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if f == "Version" && i+1 < len(fields) {
				info.SoftwareVersion = fields[i+1]
				break
			}
		}
	}

	if strings.Contains(line, "Serial") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if (f == "Serial" || f == "Number") && i+1 < len(fields) {
				if fields[i+1] != ":" {
					info.SerialNumber = fields[i+1]
				} else if i+2 < len(fields) {
					info.SerialNumber = strings.Join(fields[i+2:], " ")
				}
				break
			}
		}
	}

	if strings.Contains(line, "Uptime") || strings.Contains(line, "uptime") {
		if idx := strings.Index(line, "is"); idx > 0 {
			info.Uptime = strings.TrimSpace(line[idx+2:])
		}
	}
}
