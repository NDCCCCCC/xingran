package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/component_collector"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

const (
	defaultDeviceInfoQueueSize = 1000
	defaultDeviceInfoWorkers   = 5
)

// DeviceInfoCollectionService 设备信息采集服务
// 统一负责设备详细信息的采集和更新（纯异步架构）
type DeviceInfoCollectionService struct {
	db             *gorm.DB
	deviceExecutor *device.DeviceExecutor
	taskQueue      chan string    // 统一任务队列
	workerCount    int            // 工作协程数
	stopChan       chan struct{}  // 停止信号
	wg             sync.WaitGroup // 等待组
	isRunning      bool           // 是否正在运行
	mu             sync.RWMutex   // 读写锁
}

// NewDeviceInfoCollectionService 创建设备信息采集服务
func NewDeviceInfoCollectionService(db *gorm.DB, deviceExecutor *device.DeviceExecutor) *DeviceInfoCollectionService {
	svc := &DeviceInfoCollectionService{
		db:             db,
		deviceExecutor: deviceExecutor,
		taskQueue:      make(chan string, defaultDeviceInfoQueueSize), // 缓冲队列，最多1000个任务
		workerCount:    defaultDeviceInfoWorkers,                      // 默认5个工作协程
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
			applogger.Infof("[设备信息采集] 从数据库读取并发数配置: %d", s.workerCount)
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
	applogger.Infof("启动设备信息采集服务，工作协程数: %d", s.workerCount)

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

	applogger.Infof("正在停止设备信息采集服务...")
	close(s.stopChan)
	s.wg.Wait()
	s.isRunning = false
	applogger.Infof("设备信息采集服务已停止")
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

	applogger.Infof("开始将 %d 个在线设备加入采集队列", len(devices))

	enqueueCount := 0
	for _, device := range devices {
		// 跳过无凭证的设备
		if device.CredentialID == nil || *device.CredentialID == "" {
			continue
		}

		if err := s.Enqueue(device.ID); err != nil {
			applogger.Infof("加入队列失败 [设备ID=%s]: %v", device.ID, err)
			continue
		}
		enqueueCount++
	}

	applogger.Infof("成功将 %d 个设备加入采集队列", enqueueCount)
	return nil
}

// worker 工作协程
func (s *DeviceInfoCollectionService) worker(ctx context.Context, workerIndex int) {
	defer s.wg.Done()

	// 启动日志从 Info 降级为 Debug:Start() 已经在第 73 行打印过"工作协程数: N"
	// 汇总,worker 内部每个 goroutine 各自再打一行属于噪音(N=10 时重复 10 行)。
	// Debug 模式可保留逐个 worker 的可追溯性。
	applogger.Debugf("设备信息采集工作协程 #%d 启动", workerIndex)

	for {
		select {
		case deviceID := <-s.taskQueue:
			s.processTask(ctx, deviceID)
		case <-s.stopChan:
			applogger.Infof("设备信息采集工作协程 #%d 停止", workerIndex)
			return
		case <-ctx.Done():
			applogger.Infof("设备信息采集工作协程 #%d 上下文取消", workerIndex)
			return
		}
	}
}

// recoverPendingTasks 恢复待处理任务
func (s *DeviceInfoCollectionService) recoverPendingTasks(ctx context.Context) {
	defer s.wg.Done()

	applogger.Infof("开始恢复待处理的设备信息采集任务...")

	var tasks []models.DeviceEnrichmentTask
	if err := s.db.Where("status = ?", models.EnrichmentStatusPending).Find(&tasks).Error; err != nil {
		applogger.Infof("查询待处理任务失败: %v", err)
		return
	}

	applogger.Infof("找到 %d 个待处理任务", len(tasks))

	for _, task := range tasks {
		select {
		case s.taskQueue <- task.DeviceID:
			applogger.Infof("恢复任务: 设备ID=%s", task.DeviceID)
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// processTask 处理单个采集任务
func (s *DeviceInfoCollectionService) processTask(ctx context.Context, deviceID string) {
	applogger.Infof("开始处理设备信息采集任务: 设备ID=%s", deviceID)

	// 获取任务
	var task models.DeviceEnrichmentTask
	if err := s.db.Where("device_id = ? AND status = ?", deviceID, models.EnrichmentStatusPending).First(&task).Error; err != nil {
		applogger.Infof("获取采集任务失败: 设备ID=%s, 错误=%v", deviceID, err)
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

	// Phase 48 Wave 3 D-12/D-14: 组件采集 hook。
	// 故障不记 operlog: D-13 范围仅限 UPDATE 路径(由 OpsAssetWriter 内部 RecordBackground),
	// 失败用 applogger.Warnf 信令(应用日志层,非审计层)。
	// 不引入对称 operlog 失败记录,避免审计噪声 + operlog 表语义混淆(WARNING 6)。
	// 不阻塞 chassis 更新:即便失败也 markTaskSuccess(D-14)。
	if err := s.collectComponentInfo(ctx, &device); err != nil {
		applogger.Warnf("组件采集失败 [设备=%s]: %v", device.DeviceName, err)
	}

	// 标记任务成功
	s.markTaskSuccess(&task, info)

	applogger.Infof("设备信息采集任务完成: 设备ID=%s, 型号=%s", deviceID, info.Model)
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
			applogger.Infof("执行命令失败 [设备=%s, 命令=%s]: %v", device.DeviceName, cmd, err)
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

// updateDeviceInfo 更新设备信息（仅更新空字段，保留现有数据）
func (s *DeviceInfoCollectionService) updateDeviceInfo(device *models.NetworkDevice, info *DeviceInfo) {
	updates := map[string]interface{}{}

	// 只更新当前为空的字段，保留SNMP探测获取的数据
	if info.Model != "" && device.Model == "" {
		updates["model"] = info.Model
	}
	if info.SerialNumber != "" && device.SerialNumber == "" {
		updates["serial_number"] = info.SerialNumber
	}
	if info.SoftwareVersion != "" && device.SoftwareVersion == "" {
		updates["software_version"] = info.SoftwareVersion
	}
	if info.Uptime != "" && device.Uptime == "" {
		updates["uptime"] = info.Uptime
	}

	if len(updates) > 0 {
		now := time.Now()
		updates["updated_at"] = now
		if err := s.db.Model(device).Updates(updates).Error; err != nil {
			applogger.Infof("更新设备信息失败: %v", err)
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
	applogger.Infof("设备信息采集任务失败: 设备ID=%s, 错误=%s", task.DeviceID, errorMsg)
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

// ================================
// Phase 48 Wave 3: 组件序列号采集 hook
// ================================

// commandRunner is the SSH-style command executor signature used by the
// D-10 two-step transceiver pipeline. The real implementation wraps
// conn.GetWrapper().SendCommand(cmd, true).Result; tests inject a spy.
type commandRunner func(cmd string) (string, error)

// transceiverCommandPair returns the D-10 two-step command pair
// (status → transceiver) for the given vendor, or nil for out-of-scope
// vendors (H3C / Maipu / unknown). Per D-10, the status command MUST run
// first so the transceiver parse can filter to up interfaces only.
func transceiverCommandPair(vendor models.DeviceVendor) []string {
	switch vendor {
	case models.VendorHuawei:
		return []string{"display interface status", "display interface transceiver"}
	case models.VendorRuijie:
		return []string{"show interfaces status", "show interface transceiver"}
	default:
		// H3C / Maipu / unknown — out of scope per RESEARCH.
		return nil
	}
}

// runTwoStepTransceiverPipeline implements the D-10 two-step pipeline
// (status → transceiver) using an injectable commandRunner. It returns a
// ComponentSet containing the parsed transceiver Components (already
// filtered to up interfaces by the collector's Parse* method).
//
// The function is package-internal so the cron hook and unit tests share
// one code path — production callers pass a real SSH-backed runner;
// tests pass a spy that records command order.
func runTwoStepTransceiverPipeline(ctx context.Context, vendor models.DeviceVendor, runner commandRunner) (*component_collector.ComponentSet, error) {
	set := &component_collector.ComponentSet{Components: []component_collector.Component{}}
	if runner == nil {
		return set, nil
	}
	cmds := transceiverCommandPair(vendor)
	if cmds == nil {
		return set, nil
	}

	// Step 1: status — extract up-interface list (D-10 pre-filter).
	statusRaw, err := runner(cmds[0])
	if err != nil {
		return set, fmt.Errorf("two-step pipeline status %q: %w", cmds[0], err)
	}
	var upInterfaces []string
	switch vendor {
	case models.VendorHuawei:
		upInterfaces, err = component_collector.NewHuaweiCliCollector().ParseInterfaceStatus(statusRaw)
		if err != nil {
			return set, fmt.Errorf("huawei status parse: %w", err)
		}
	case models.VendorRuijie:
		upInterfaces, err = component_collector.NewRuijieCliCollector().ParseInterfacesStatus(statusRaw)
		if err != nil {
			return set, fmt.Errorf("ruijie status parse: %w", err)
		}
	}

	// Step 2: transceiver — gated by the up-interface list from step 1.
	transceiverRaw, err := runner(cmds[1])
	if err != nil {
		return set, fmt.Errorf("two-step pipeline transceiver %q: %w", cmds[1], err)
	}
	var transceivers []component_collector.Component
	switch vendor {
	case models.VendorHuawei:
		transceivers, err = component_collector.NewHuaweiCliCollector().ParseInterfaceTransceiver(transceiverRaw, upInterfaces)
		if err != nil {
			return set, fmt.Errorf("huawei transceiver parse: %w", err)
		}
	case models.VendorRuijie:
		transceivers, err = component_collector.NewRuijieCliCollector().ParseTransceiverDDM(transceiverRaw, upInterfaces)
		if err != nil {
			return set, fmt.Errorf("ruijie transceiver parse: %w", err)
		}
	}
	set.Components = append(set.Components, transceivers...)
	return set, nil
}

// collectComponentInfo is the Phase 48 cron hook (D-12 / D-14). It runs
// the D-10 two-step transceiver pipeline for the device and feeds the
// resulting ComponentSet into the component_collector.Pipeline which
// UPDATEs ops_asset and emits reconciliation anomalies as needed.
//
// SNMP-based ENTITY-MIB collection (D-08) is intentionally omitted in v1
// cron path — the transceiver CLI path is the higher-priority surface
// (Ruijie business-board SNs come from show version, already collected by
// the chassis collector; Huawei S8700 chassis/board/fan SNs come from
// ENTITY-MIB which requires a real SNMP probe — deferred to真机 UAT).
// Future revisions can extend this hook to instantiate an SNMP client
// and call EntityCollector.Collect before the CLI path; the ComponentSet
// merge API is already in place.
//
// Returns error to the caller (processTask) which logs warn per D-14
// but does NOT mark the chassis-update task as failed.
func (s *DeviceInfoCollectionService) collectComponentInfo(ctx context.Context, device *models.NetworkDevice) error {
	if s == nil || device == nil {
		return nil
	}
	if device.Vendor != models.VendorHuawei && device.Vendor != models.VendorRuijie {
		// Out-of-scope vendor (H3C/Maipu/unknown) — no-op, not an error.
		return nil
	}

	// Build a real SSH-backed runner bound to the device connection pool.
	runner := func(cmd string) (string, error) {
		return s.runDeviceCommand(ctx, device, cmd)
	}

	set, err := runTwoStepTransceiverPipeline(ctx, device.Vendor, runner)
	if err != nil {
		return err
	}
	if set == nil || (len(set.Components) == 0 && set.Chassis == nil) {
		// Nothing to write — common for switches with no up SFP ports.
		return nil
	}

	// Wire the pipeline. cronAssetLookup adapts the DB to
	// component_collector.AssetLookup (avoids cyclic import on
	// operations). operLog is left nil: per-UPDATE audit row is a future
	// hardening; D-13 scope remains documented in the SUMMARY.
	pipeline := component_collector.NewPipeline(
		s.db,
		&cronAssetLookup{db: s.db},
		nil,
	)
	devRef := component_collector.DeviceRef{
		ID:           device.ID,
		SerialNumber: device.SerialNumber,
		Vendor:       string(device.Vendor),
	}
	return pipeline.Run(ctx, devRef, set)
}

// runDeviceCommand wraps the existing connection pool to issue a single
// SSH command on the device. Returns the raw response text on success.
func (s *DeviceInfoCollectionService) runDeviceCommand(ctx context.Context, device *models.NetworkDevice, cmd string) (string, error) {
	if s.deviceExecutor == nil {
		return "", fmt.Errorf("deviceExecutor not configured")
	}
	pool := s.deviceExecutor.GetScheduler().GetConnectionPool()
	conn, err := pool.GetConnection(ctx, device.ID)
	if err != nil {
		return "", fmt.Errorf("获取设备连接失败: %w", err)
	}
	if !conn.Acquire() {
		return "", fmt.Errorf("无法获取连接锁")
	}
	defer conn.Release()

	wrapper := conn.GetWrapper()
	resp, err := wrapper.SendCommand(cmd, true)
	if err != nil {
		return "", err
	}
	return resp.Result, nil
}

// cronAssetLookup adapts the cron-path DB to component_collector.AssetLookup.
// It is a minimal read-side implementation of GetByDeviceSN mirroring
// operations.assetService.GetByDeviceSN's (nil, nil) on not-found contract.
type cronAssetLookup struct {
	db *gorm.DB
}

// GetByDeviceSN implements component_collector.AssetLookup. Returns the
// ops_asset.id for the given device SN, or (nil, nil) when not found.
func (a *cronAssetLookup) GetByDeviceSN(ctx context.Context, deviceSN string) (*component_collector.AssetRef, error) {
	if deviceSN == "" {
		return nil, nil
	}
	var id string
	row := a.db.WithContext(ctx).
		Table("ops_asset").
		Where("devicesn = ? AND deleted_at IS NULL", deviceSN).
		Select("id").Limit(1).Row()
	if err := row.Scan(&id); err != nil {
		// not-found → (nil, nil); surface other errors.
		if strings.Contains(err.Error(), "no rows") || strings.Contains(err.Error(), "NotFound") {
			return nil, nil
		}
		return nil, err
	}
	if id == "" {
		return nil, nil
	}
	return &component_collector.AssetRef{ID: id}, nil
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
