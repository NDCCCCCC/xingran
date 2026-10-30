package services

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/component_collector"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

const (
	defaultDeviceInfoQueueSize = 1000
	defaultDeviceInfoWorkers   = 5

	// deviceInfoStopTimeout DeviceInfoCollectionService.Stop() 等待 worker 退出的兜底超时。
	//
	// 设计目的(2026-07-06 修复 — shutdown-hang-after-port-close):
	//  worker.processTask 调 wrapper.SendCommand (SSH) 阻塞,无 per-task timeout;
	//  若 SSH 设备 hang,worker 永远不退出 → wg.Wait 永不返回 → core.Close 卡死。
	//  8s 足够健康 worker 完成收尾(dequeue 当前 task + markTaskSuccess/Failed),
	//  超时则丢弃等待继续返回,不让 Close 流程被 SSH 卡死拖累。
	//  资源由 SSH 连接池 / DB pool 在 OS 退出时回收。
	deviceInfoStopTimeout = 8 * time.Second
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
//
// 2026-07-06 修复(shutdown-hang-after-port-close):wg.Wait 改为 select + time.After
// 兜底 8s,防止 SSH 长连接 / DB 查询卡死的 worker 拖累 core.Close 整个流程。
// 超时仅 log 警告继续返回,资源由 OS 退出时回收。
func (s *DeviceInfoCollectionService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	applogger.Infof("正在停止设备信息采集服务...")
	close(s.stopChan)

	waitDone := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		applogger.Infof("设备信息采集 worker 已全部退出")
	case <-time.After(deviceInfoStopTimeout):
		applogger.Warnf("设备信息采集 worker 未在 %v 内全部退出,强制返回(资源由 OS 回收)", deviceInfoStopTimeout)
	}
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

	// Ruijie Gap 2 自动同步: 把 ops_asset.chassis.devicesn 与 sys_network_device.serial_number
	// 保持 lock-step。仅当 serial_number 实际变更(M1 SN → 真 chassis SN)时触发。
	s.syncOpsAssetChassisSN(&device, info)

	// 同步 device.SerialNumber 本地变量到 DB 新值,确保后续 collectComponentInfo 用
	// 正确的 chassis SN 查找 parent_asset_id(否则会用陈旧的 M1 SN 找 chassis,找不到 →
	// parent_asset_id 为空 → 板卡变孤儿 → 组件 Tab 空)。
	if info.SerialNumber != "" && device.SerialNumber != info.SerialNumber {
		device.SerialNumber = info.SerialNumber
	}

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

	// F-14 Phase 31 修复 (2026-07-06 复查):GetConnection 内部已完成 refCount +1,
	// 不能再 Acquire() (会双 +1) + Release() (仅 -1) 导致 refCount 永远停在 1,
	// IsIdle() 永不为 true,清理 goroutine 永远不删该连接 → 24/20 池满。
	// 正确配对: defer conn.ReleaseRef()
	defer conn.ReleaseRef()

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
		// Phase 49-01 Gap 3: legacy 字符串解析路径(负责 Model/SoftwareVersion/Uptime)。
		s.parseDeviceInfo(response.Result, device.Vendor, info)
		// Phase 49-01 Gap 3: 叠加 chassis SN 解析器(已验证稳定),回填 info.SerialNumber。
		// 仅在 chassis SN 命令上触发,避免对其它输出重复跑模板解析。
		s.enrichChassisSerial(cmd, device.Vendor, response.Result, info)
	}

	return info, nil
}

// getCommandsByVendor 根据厂商获取命令列表
//
// Phase 49-D-11: 锐捷 chassis SN 改走 "show manuinfo"(Device 1 / Location: Chassis)
// 作为权威来源 —— "show version" 的 "System serial number" 行实为活动主控板
// M1 的 SN,不是真机箱 SN。该命令顺序保证 manuinfo 先跑,enrichChassisSerial 的
// only-if-empty 守卫让真 chassis SN 先填入,后续 show version 跳过,避免 M1 SN 覆盖。
func (s *DeviceInfoCollectionService) getCommandsByVendor(vendor models.DeviceVendor) []string {
	switch vendor {
	case models.VendorHuawei, models.VendorH3C:
		// Phase 49-01 Gap 3: 加入 "display device esn" 用于 chassis SN 采集。
		// Phase 49-D-12: 加入 "display device elabel brief" 用于板卡/风扇/电源 SN
		// 采集(替代 D-08 deferred 的 ENTITY-MIB 板卡路径),同时作为 chassis SN
		// 的 fallback(老固件 V600R024C00 退役了 display device esn 时,
		// elabel brief 第一行 "Equipment SN(ESN):" 仍可拿到 chassis ESN)。
		return []string{"display version", "display device", "display device esn", "display device elabel brief"}
	case models.VendorRuijie, models.VendorMaipu:
		// Phase 49-D-11: manuinfo 必须放在 show version 之前(only-if-empty 守卫的语义)。
		return []string{"show manuinfo", "show version"}
	default:
		return []string{"show version"}
	}
}

// isChassisSNCommand 返回 cmd 是否是用于提取 chassis SN 的厂商命令。
// Phase 49-01 Gap 3: 仅 chassis SN 命令走解析器路径(避免重复解析无关输出)。
// Phase 49-D-11: 锐捷 chassis SN 命令从 "show version" 切到 "show manuinfo"。
// show version 的 "System serial number" 是活动 M1 主控板的 SN,不是真机箱 SN。
// show manuinfo Device 1 / Location: Chassis 才是物理机箱 SN 的权威来源。
// Phase 49-D-12: 华为 chassis SN 备选源加入 "display device elabel brief"
// (V600R024C00+ 退役 display device esn 时的 fallback)。
func isChassisSNCommand(vendor models.DeviceVendor, cmd string) bool {
	switch vendor {
	case models.VendorHuawei, models.VendorH3C:
		return cmd == "display device esn" || cmd == "display device elabel brief"
	case models.VendorRuijie, models.VendorMaipu:
		return cmd == "show manuinfo"
	default:
		return false
	}
}

// enrichChassisSerial 复用 component_collector 已经过单测验证的解析器
// (ParseShowManuinfo / ParseShowVersionModules / ParseDisplayDeviceEsn)
// 从 chassis SN 命令的原始输出中提取 chassis SN,回填到 info.SerialNumber。
//
// Phase 49-01 Gap 3: 修复 sys_network_device.serial_number 在 ruijie/huawei
// 在线设备上 100% 为空的问题。现有 parseRuijieDeviceInfo/parseHuaweiDeviceInfo
// 字符串关键字解析过于脆弱(无法匹配生产输出格式),本方法叠加(非替换)使用
// 已验证的解析器作为 chassis SN 的真实来源。
//
// Phase 49-D-11: 锐捷 chassis SN 解析源由 ParseShowVersionModules 切到
// ParseShowManuinfo,因 "show version" 的 "System serial number" 实为活动 M1
// 主控板的 SN,不是物理机箱 SN。getCommandsByVendor 把 "show manuinfo" 排在
// "show version" 之前,manuinfo 先跑 → 真 chassis SN 先填 → 后续 show version
// 触发 only-if-empty 守卫跳过,M1 SN 不再覆盖(若被 manuinfo 错过则从 M1 槽位
// 推断 — 见下方 fall-through)。
//
// 语义:
//   - 仅当 info.SerialNumber 当前为空时写入(only-if-empty,沿用 updateDeviceInfo 语义)
//   - huawei "Unrecognized command" → 解析器返回空切片+nil err(非错误)
//   - 其它解析错误: applogger.Infof 记录并返回(不阻塞,D-14 容错语义)
//   - 不修改 Model/SoftwareVersion/Uptime(继续由 legacy parseDeviceInfo 路径负责)
func (s *DeviceInfoCollectionService) enrichChassisSerial(cmd string, vendor models.DeviceVendor, raw string, info *DeviceInfo) {
	if info == nil {
		return
	}
	// only-if-empty: 不覆盖更早命令(或 SNMP 探测)已经填入的 SN。
	if info.SerialNumber != "" {
		return
	}
	if !isChassisSNCommand(vendor, cmd) {
		return
	}

	var components []component_collector.Component
	var err error
	switch vendor {
	case models.VendorRuijie, models.VendorMaipu:
		// Phase 49-D-11: ruijie chassis SN 权威来源是 show manuinfo(物理机箱 SN),
		// 而不是 show version(活动 M1 主控板 SN)。此处 cmd 已被 isChassisSNCommand
		// 过滤为 "show manuinfo",show version 走 legacy parseDeviceInfo 不经此处。
		components, err = component_collector.NewRuijieCliCollector().ParseShowManuinfo(raw)
	case models.VendorHuawei, models.VendorH3C:
		// Phase 49-D-12: 华为 chassis SN 双源。
		//   - "display device esn"        → ParseDisplayDeviceEsn (主)
		//   - "display device elabel brief" → regex 抓 "Equipment SN(ESN):" 行
		//     (V600R024C00+ 退役了 esn 命令时的 fallback,elabel brief 仍报 ESN)
		if cmd == "display device elabel brief" {
			esn := huaweiElabelChassisESN(raw)
			if esn != "" {
				info.SerialNumber = esn
			}
			return
		}
		components, err = component_collector.NewHuaweiCliCollector().ParseDisplayDeviceEsn(raw)
	default:
		return
	}
	if err != nil {
		// 沿用 D-14 容错语义:解析失败不阻塞,记录日志(不打印 SN 全量值)。
		applogger.Infof("chassis SN 解析失败 [vendor=%s, cmd=%s]: %v", vendor, cmd, err)
		return
	}

	for _, c := range components {
		if c.ComponentType == component_collector.ComponentTypeChassis && c.SerialNumber != "" {
			info.SerialNumber = c.SerialNumber
			return
		}
	}
	// 未找到 chassis 行(如 huawei Unrecognized command / 固定形态设备 slot-only 输出)
	// → 静默返回,info.SerialNumber 保持空,符合 Pitfall 3 语义。
}

// updateDeviceInfo 更新设备信息（仅更新空字段，保留现有数据）
//
// Ruijie 特殊处理(Gap 2 自动同步前奏):
//   - Ruijie 的 chassis SN 来源是 `show manuinfo`(设备机框 SN),不是 `show version` 的
//     System serial number(那是主控板 M1 的 SN)。如果导入 chassis 资产时使用了 M1 SN
//     作为 devicesn,sys_network_device.serial_number 与 ops_asset.chassis.devicesn 失配,
//     Gap 2 关联断裂 → 组件 Tab parent_asset_id 全空。
//   - 因此对 ruijie 设备:除了原 only-if-empty 守卫,允许 serial_number 从 M1 SN 覆写为
//     真 chassis SN(info.SerialNumber != "" && device.SerialNumber != info.SerialNumber)。
//     仅 ruijie 走这条路径;huawei 等其他厂商保持原 only-if-empty 行为。
//   - 实际的两表同步(ops_asset.chassis.devicesn)由 processTask 后续的
//     syncOpsAssetChassisSN 完成(基于本函数内的 device.SerialNumber 旧值做精确 UPDATE)。
func (s *DeviceInfoCollectionService) updateDeviceInfo(device *models.NetworkDevice, info *DeviceInfo) {
	updates := map[string]interface{}{}

	// 只更新当前为空的字段，保留SNMP探测获取的数据
	if info.Model != "" && device.Model == "" {
		updates["model"] = info.Model
	}
	if info.SerialNumber != "" && device.SerialNumber == "" {
		updates["serial_number"] = info.SerialNumber
	} else if device.Vendor == models.VendorRuijie && info.SerialNumber != "" && device.SerialNumber != info.SerialNumber {
		// Ruijie 特殊: 允许覆写(从 M1 SN → 真 chassis SN 的过渡)。
		// 仅 ruijie;huawei 等厂商仍走 only-if-empty 保持向后兼容。
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

// syncOpsAssetChassisSN 同步 ops_asset.chassis.devicesn 至 sys_network_device.serial_number。
//
// 仅对 ruijie 设备生效,且仅当 serial_number 实际变更时触发(从 M1 SN → 真 chassis SN 过渡)。
// 必须使用 device.SerialNumber 旧值作为 locator (1:1 唯一);不能用 source_device_id
// (该列在 chassis 行上未填充,见 debug session ruijie-chassis-sn-manuinfo)。
// 不修改板卡行(component_type IN ('engine','card','transceiver'))的 devicesn —— 板卡的
// devicesn 是板卡自己的 SN,不是 chassis SN,Gap 2 关联走 parent_asset_id 而非 devicesn。
func (s *DeviceInfoCollectionService) syncOpsAssetChassisSN(device *models.NetworkDevice, info *DeviceInfo) {
	if device == nil || info == nil {
		return
	}
	if device.Vendor != models.VendorRuijie {
		return
	}
	oldSN := device.SerialNumber // updateDeviceInfo 不回写本地变量,此处仍为旧值
	newSN := info.SerialNumber
	if oldSN == "" || newSN == "" || oldSN == newSN {
		return
	}
	res := s.db.Model(&models.Asset{}).
		Where("devicesn = ? AND component_type IS NULL AND deleted_at IS NULL", oldSN).
		UpdateColumn("devicesn", newSN)
	if res.Error != nil {
		applogger.Warnf("同步 ops_asset.chassis.devicesn 失败 [设备=%s, old=%s, new=%s]: %v",
			device.DeviceName, oldSN, newSN, res.Error)
		return
	}
	if res.RowsAffected > 0 {
		applogger.Infof("同步 ops_asset.chassis.devicesn [设备=%s, old=%s → new=%s, 受影响行=%d]",
			device.DeviceName, oldSN, newSN, res.RowsAffected)
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
	//
	// Phase 49-D-11 第二次修复:legacy chassis SN 抓取整体废弃,只保留 Model /
	// SoftwareVersion / Uptime 的解析。原因:show version "System serial number"
	// 实为活动 M1 主控板的 SN,不是物理机箱 SN(真机箱 SN 只在 show manuinfo
	// Device 1 / Location: Chassis 行)。Phase 49-D-11 第一次修复只把"Device
	// Serial Number"行(per-slot/per-chassis of manuinfo)从 legacy 排除,但
	// "System serial number"行(per-chassis of show version)仍被 legacy 命中
	// → show manuinfo 已经把真 chassis SN 写进 info.SerialNumber,接着
	// show version 又被 legacy 抓 M1 SN 覆盖 → enrichChassisSerial 的
	// only-if-empty 守卫反而成了"manuinfo 被污染后无法修复"的陷阱。
	//
	// 修法:legacy 完全不再写 info.SerialNumber,chassis SN 100% 走
	// enrichChassisSerial → textfsm 解析(show manuinfo 的 chassis 行,或
	// huawei "display device esn" 的 chassis esn 行)。老固件无法跑 manuinfo
	// 的情况:info.SerialNumber 留空(同 Phase 49-01 Gap 3 修复前的行为,优于
	// 落"Number:" 之类的脏值)。
	_ = line // legacy chassis SN 抓取已废弃,见上方 Phase 49-D-11 注释
	_ = info

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
// the board-collection branch (Gap 1 fix, 49-02) and the D-10 two-step
// transceiver pipeline, then feeds the merged ComponentSet into the
// component_collector.Pipeline which UPDATEs ops_asset and emits
// reconciliation anomalies as needed.
//
// Phase 49-02 Gap 1 fix: previously this hook ONLY ran the transceiver
// pipeline, leaving the already-implemented ParseShowVersionModules as
// dead code — boards (engine/card) were never collected. The misleading
// historical comment "already collected by the chassis collector"
// referenced a collector that never existed.
//
// SNMP-based ENTITY-MIB collection (D-08) is intentionally omitted in v1
// cron path (Huawei chassis/board/fan SNs come from ENTITY-MIB which
// requires a real SNMP probe — deferred to真机 UAT; Ruijie boards come
// from `show version` which IS collected here).
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

	// Initialize the merge set explicitly. set.Chassis MUST stay nil —
	// the chassis asset already exists (the parent device itself), and
	// Pipeline.Run (pipeline.go:108) only consumes set.Components via
	// writer.Write, never *set.Chassis.
	set := &component_collector.ComponentSet{Components: []component_collector.Component{}}

	// Step 1: board collection (Gap 1 fix, 49-02). D-14 fault tolerance:
	// command/parse errors are logged + swallowed so the transceiver
	// pipeline below is never blocked by a board-collection failure.
	if err := collectBoardsInto(device, runner, set); err != nil {
		applogger.Infof("板卡采集失败 [设备=%s]: %v", device.DeviceName, err)
	}

	// Step 2: D-10 two-step transceiver pipeline. Merge its Components
	// into set.Components; ignore its Chassis field (keep set.Chassis nil).
	xceiverSet, err := runTwoStepTransceiverPipeline(ctx, device.Vendor, runner)
	if err != nil {
		// Boards collected in step 1 are still valuable — fall through to
		// Pipeline.Run even if the transceiver pipeline errored, as long as
		// we have at least one board. If boards are also empty, return err.
		if len(set.Components) == 0 {
			return err
		}
		applogger.Infof("光模块采集失败 [设备=%s]: %v", device.DeviceName, err)
	} else if xceiverSet != nil {
		set.Components = append(set.Components, xceiverSet.Components...)
	}

	if len(set.Components) == 0 {
		// Nothing to write — common for switches with no up SFP ports
		// AND no boards (e.g. huawei with deferred ENTITY-MIB).
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

// collectBoardsInto is the Phase 49-02 Gap-1 board-collection branch.
// It parses per-slot board components (engine/card) from vendor CLI
// output and appends them to set.Components.
//
// Key design decisions (BLOCKER-2):
//   - Ruijie: parses `show version` via NewRuijieCliCollector().ParseShowVersionModules.
//     The chassis row (ComponentType==ComponentTypeChassis) is DROPPED — the
//     chassis asset already exists (parent device), and Pipeline.Run only
//     consumes set.Components (never *set.Chassis). engine/card rows
//     (M1/1-5/M2) are appended. Note: M1's SN on real hardware may equal
//     chassis SN (Ruijie main-control-board reuse), but M1's ComponentType
//     is ComponentTypeEngine — independent of the dropped chassis row.
//   - Huawei (Phase 49-D-12): parses `display device elabel brief` via
//     NewHuaweiCliCollector().ParseDisplayDeviceElabelBrief. Replaces the
//     D-08 deferred ENTITY-MIB board path for devices that have elabel
//     brief (V200R003+ S5700/S6700/S7700/S8700/S9700). chassis row
//     (BackPlane SLOTID="--") is dropped by the parser itself
//     (ParseDisplayDeviceElabelBrief). card/engine/fan/power rows are
//     appended.
//   - Out-of-scope vendor (H3C/Maipu/unknown): no-op.
//
// D-14 fault tolerance: command execution errors are logged + swallowed
// (return nil) so the caller (collectComponentInfo) can still run the
// transceiver pipeline. Parser errors are also tolerated.
//
// set.Chassis is NEVER set by this function (it stays nil per BLOCKER-2).
func collectBoardsInto(device *models.NetworkDevice, runner commandRunner, set *component_collector.ComponentSet) error {
	if device == nil || runner == nil || set == nil {
		return nil
	}
	switch device.Vendor {
	case models.VendorRuijie, models.VendorMaipu:
		return collectRuijieBoardsInto(device, runner, set)
	case models.VendorHuawei:
		// H3C shares the huawei VRP-style CLI on most models but Phase 49-D-12
		// has not yet verified elabel brief output for H3C. Stay opt-in for
		// vendor=Huawei only; H3C continues to no-op (same as pre-D-12).
		return collectHuaweiBoardsInto(device, runner, set)
	default:
		// H3C/Maipu/unknown out of scope. No-op, not an error.
		return nil
	}
}

// collectRuijieBoardsInto runs the Ruijie board path: `show version` →
// ParseShowVersionModules → drop chassis row → append engine/card.
func collectRuijieBoardsInto(device *models.NetworkDevice, runner commandRunner, set *component_collector.ComponentSet) error {
	raw, err := runner("show version")
	if err != nil {
		// D-14: don't block the transceiver pipeline on a command failure.
		applogger.Infof("板卡命令执行失败 [设备=%s, cmd=show version]: %v", device.DeviceName, err)
		return nil
	}
	components, err := component_collector.NewRuijieCliCollector().ParseShowVersionModules(raw)
	if err != nil {
		// D-14: parser/template error is tolerated.
		applogger.Infof("板卡解析失败 [设备=%s]: %v", device.DeviceName, err)
		return nil
	}

	for _, c := range components {
		// BLOCKER-2: drop chassis row — the chassis asset already exists
		// (parent device itself) and Pipeline.Run never consumes
		// *set.Chassis. Appending a chassis row here would cause the
		// writer to emit a component_type='chassis' row that pollutes the
		// 前端「从属组件清单」Tab filter.
		if c.ComponentType == component_collector.ComponentTypeChassis {
			continue
		}
		set.Components = append(set.Components, c)
	}
	return nil
}

// collectHuaweiBoardsInto runs the Huawei board path (Phase 49-D-12):
// `display device elabel brief` → ParseDisplayDeviceElabelBrief →
// append card/engine/fan/power (chassis row already dropped by parser).
//
// The Huawei collector's parser also tolerates the
// "Error: Unrecognized command" path (V100R005 and earlier / some
// fixed-form switches) by returning an empty slice — the transceiver
// pipeline still runs in that case per D-14.
func collectHuaweiBoardsInto(device *models.NetworkDevice, runner commandRunner, set *component_collector.ComponentSet) error {
	const elabelCmd = "display device elabel brief"
	raw, err := runner(elabelCmd)
	if err != nil {
		applogger.Infof("板卡命令执行失败 [设备=%s, cmd=%s]: %v", device.DeviceName, elabelCmd, err)
		return nil
	}
	components, err := component_collector.NewHuaweiCliCollector().ParseDisplayDeviceElabelBrief(raw)
	if err != nil {
		applogger.Infof("板卡解析失败 [设备=%s, cmd=%s]: %v", device.DeviceName, elabelCmd, err)
		return nil
	}

	for _, c := range components {
		// Belt-and-suspenders: even though ParseDisplayDeviceElabelBrief drops
		// the BackPlane row, also drop any chassis row that somehow leaked
		// (matches BLOCKER-2 invariant for the ruijie branch).
		if c.ComponentType == component_collector.ComponentTypeChassis {
			continue
		}
		set.Components = append(set.Components, c)
	}
	return nil
}

// huaweiElabelChassisESN extracts the chassis ESN from the
// `display device elabel brief` output via a simple regex on the
// "Equipment SN(ESN): <esn>" header line. Returns "" if the line
// is missing (e.g. very old firmware that uses a different header).
//
// Used by enrichChassisSerial as a chassis SN fallback when
// `display device esn` was retired by V600R024C00+ — elabel brief
// reports the same ESN in its first header line.
//
// Phase 49-D-12: keep this regex local (not in textfsm) because the
// ESN line is a single header row outside the tabular data, and a
// 1-line regex is clearer than a separate textfsm template.
func huaweiElabelChassisESN(raw string) string {
	// Multi-line match (^) on the first "Equipment SN(ESN):" row.
	re := regexp.MustCompile(`(?m)^Equipment\s+SN\(ESN\):\s*(\S+)`)
	m := re.FindStringSubmatch(raw)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
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
	// F-14 修复 (2026-07-06 复查):同 CollectDeviceInfo,改 ReleaseRef
	defer conn.ReleaseRef()

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
