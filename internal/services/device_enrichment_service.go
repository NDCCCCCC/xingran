package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// DeviceEnrichmentService 设备信息补充服务
// 后台异步通过 SSH 获取设备详细信息（型号、版本等）
type DeviceEnrichmentService struct {
	db            *gorm.DB
	deviceManager *device.Manager
	taskQueue     chan string     // 设备ID队列
	workerCount   int              // 工作协程数
	stopChan      chan struct{}    // 停止信号
	wg            sync.WaitGroup   // 等待组
	isRunning     bool             // 是否正在运行
	mu            sync.RWMutex     // 读写锁
}

// NewDeviceEnrichmentService 创建设备补充服务
func NewDeviceEnrichmentService(db *gorm.DB, deviceManager *device.Manager, workerCount int) *DeviceEnrichmentService {
	if workerCount <= 0 {
		workerCount = 5 // 默认5个工作协程
	}

	return &DeviceEnrichmentService{
		db:            db,
		deviceManager: deviceManager,
		taskQueue:     make(chan string, 1000), // 缓冲队列，最多1000个任务
		workerCount:   workerCount,
		stopChan:      make(chan struct{}),
	}
}

// Start 启动后台补充服务
func (s *DeviceEnrichmentService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("服务已经在运行")
	}

	s.isRunning = true
	log.Printf("启动设备信息补充服务，工作协程数: %d", s.workerCount)

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
func (s *DeviceEnrichmentService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	log.Println("正在停止设备信息补充服务...")
	close(s.stopChan)
	s.wg.Wait()
	s.isRunning = false
	log.Println("设备信息补充服务已停止")
}

// Enqueue 将设备ID加入补充队列
func (s *DeviceEnrichmentService) Enqueue(deviceID string) error {
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

	// 创建新的补充任务
	task := &models.DeviceEnrichmentTask{
		DeviceID: deviceID,
		Status:   models.EnrichmentStatusPending,
	}

	if err := s.db.Create(task).Error; err != nil {
		return fmt.Errorf("创建补充任务失败: %w", err)
	}

	// 加入队列
	select {
	case s.taskQueue <- deviceID:
		return nil
	default:
		return fmt.Errorf("任务队列已满，请稍后再试")
	}
}

// worker 工作协程
func (s *DeviceEnrichmentService) worker(ctx context.Context, workerIndex int) {
	defer s.wg.Done()

	log.Printf("设备补充工作协程 #%d 启动", workerIndex)

	for {
		select {
		case deviceID := <-s.taskQueue:
			s.processTask(ctx, deviceID)
		case <-s.stopChan:
			log.Printf("设备补充工作协程 #%d 停止", workerIndex)
			return
		case <-ctx.Done():
			log.Printf("设备补充工作协程 #%d 上下文取消", workerIndex)
			return
		}
	}
}

// recoverPendingTasks 恢复待处理任务
func (s *DeviceEnrichmentService) recoverPendingTasks(ctx context.Context) {
	defer s.wg.Done()

	log.Println("开始恢复待处理的补充任务...")

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

// processTask 处理单个补充任务
func (s *DeviceEnrichmentService) processTask(ctx context.Context, deviceID string) {
	log.Printf("开始处理设备补充任务: 设备ID=%s", deviceID)

	// 获取任务
	var task models.DeviceEnrichmentTask
	if err := s.db.Where("device_id = ? AND status = ?", deviceID, models.EnrichmentStatusPending).First(&task).Error; err != nil {
		log.Printf("获取补充任务失败: 设备ID=%s, 错误=%v", deviceID, err)
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

	// 执行补充
	enrichmentData, err := s.EnrichDeviceDetails(ctx, &device)
	if err != nil {
		s.markTaskFailed(&task, fmt.Sprintf("补充设备信息失败: %v", err))
		return
	}

	// 更新设备信息（只更新Model字段，Version是BaseModel的乐观锁字段，Serial字段不存在）
	if enrichmentData.Model != "" {
		device.Model = enrichmentData.Model
	}

	s.db.Save(&device)

	// 标记任务成功
	completedAt := time.Now()
	task.Status = models.EnrichmentStatusSuccess
	task.CompletedAt = &completedAt
	task.EnrichedModel = &enrichmentData.Model
	s.db.Save(&task)

	log.Printf("设备补充任务完成: 设备ID=%s, 型号=%s", deviceID, enrichmentData.Model)
}

// markTaskFailed 标记任务失败
func (s *DeviceEnrichmentService) markTaskFailed(task *models.DeviceEnrichmentTask, errorMsg string) {
	completedAt := time.Now()
	task.Status = models.EnrichmentStatusFailed
	task.CompletedAt = &completedAt
	task.ErrorMessage = errorMsg
	s.db.Save(task)
	log.Printf("设备补充任务失败: 设备ID=%s, 错误=%s", task.DeviceID, errorMsg)
}

// EnrichmentData 补充数据
type EnrichmentData struct {
	Model string
}

// EnrichDeviceDetails 通过 SSH 获取设备详细信息
func (s *DeviceEnrichmentService) EnrichDeviceDetails(ctx context.Context, device *models.NetworkDevice) (*EnrichmentData, error) {
	// 获取设备凭证
	if device.CredentialID == nil || *device.CredentialID == "" {
		return nil, fmt.Errorf("设备未配置授权凭证")
	}

	var credential models.AuthCredential
	if err := s.db.Where("id = ?", *device.CredentialID).First(&credential).Error; err != nil {
		return nil, fmt.Errorf("获取授权凭证失败: %w", err)
	}

	// 获取设备厂商对应的版本命令
	versionCmd := getVersionCommand(device.Vendor)

	// 使用 DeviceManager 通过 SSH 连接设备并执行命令
	output, err := s.deviceManager.ExecuteOnDevice(ctx, device.ID, versionCmd, true)
	if err != nil {
		return nil, fmt.Errorf("执行命令失败: %w", err)
	}

	// 解析输出获取型号
	data := &EnrichmentData{
		Model: parseModelFromOutput(output, device.Vendor),
	}

	return data, nil
}

// getVersionCommand 根据厂商获取版本命令
func getVersionCommand(vendor models.DeviceVendor) string {
	switch vendor {
	case models.VendorHuawei:
		return "display version"
	case models.VendorH3C:
		return "display version"
	case models.VendorRuijie:
		return "show version"
	case models.VendorMaipu:
		return "show version"
	default:
		return "show version"
	}
}

// parseModelFromOutput 从输出解析型号
func parseModelFromOutput(output string, vendor models.DeviceVendor) string {
	// 简化实现，实际应该根据不同厂商的输出格式进行解析
	// 这里可以扩展更复杂的解析逻辑
	return ""
}

// GetTaskStatus 获取设备补充任务状态
func (s *DeviceEnrichmentService) GetTaskStatus(ctx context.Context, deviceID string) (*models.DeviceEnrichmentTask, error) {
	var task models.DeviceEnrichmentTask
	if err := s.db.Where("device_id = ?", deviceID).Order("created_at DESC").First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 没有任务
		}
		return nil, fmt.Errorf("查询任务状态失败: %w", err)
	}
	return &task, nil
}
