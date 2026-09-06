package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/pkg/constants"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/query"
	"gorm.io/gorm"
)

// isDuplicateActiveRestoreErr 识别同设备活跃任务唯一索引冲突（migration 212，
// v129-recheck C-1）。双方言：PG unique_violation (SQLSTATE 23505 / duplicate
// key)；sqlite (glebarez/modernc) "UNIQUE constraint failed"；GORM 侧启用
// TranslateError 时为 gorm.ErrDuplicatedKey。
func isDuplicateActiveRestoreErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "UNIQUE constraint failed")
}

// ConfigRestoreTaskService 配置恢复任务服务（Phase 93 D-15..D-22/D-34）：
// 异步恢复的生命周期编排——发起（同设备校验 D-04 + 互斥 D-08 + 建任务）→
// detached 执行（恢复前备份 D-10/14 → 读备份 D-23 解压路径 → RestoreConfig
// 下发 D-02/12 → 回读 hash 警告 D-11 → 版本链记录 D-09）→ 任务终态查询。
//
// 状态机（D-34 四态，不支持取消）：pending → running → success | failed。
type ConfigRestoreTaskService struct {
	db        *gorm.DB
	backupSvc *ConfigBackupService
	executor  *device.DeviceExecutor
}

// NewConfigRestoreTaskService 创建配置恢复任务服务
func NewConfigRestoreTaskService(db *gorm.DB, backupSvc *ConfigBackupService, executor *device.DeviceExecutor) *ConfigRestoreTaskService {
	return &ConfigRestoreTaskService{
		db:        db,
		backupSvc: backupSvc,
		executor:  executor,
	}
}

// restoreTaskAllowedSortFields 恢复任务列表可排序字段白名单（status 列在本表真实存在）
var restoreTaskAllowedSortFields = map[string]string{
	"deviceId":  "device_id",
	"status":    "status",
	"createdAt": "created_at",
}

// StartRestore 发起恢复：校验 → 互斥 → 建任务 → 异步执行。返回任务记录
// （handler 取其 ID 返回 taskId，D-17）。
func (s *ConfigRestoreTaskService) StartRestore(ctx context.Context, backupID, deviceID, createdBy string) (*models.ConfigRestoreTask, error) {
	// ① 同设备校验（D-04）：跨设备误推配置是高危事故源
	backup, err := s.backupSvc.GetBackupByID(ctx, backupID)
	if err != nil {
		return nil, err
	}
	if backup.DeviceID != deviceID {
		return nil, fmt.Errorf("备份不属于目标设备，仅限恢复到备份源设备")
	}

	// ② 同设备互斥（D-08）：并发恢复交叉下发比慢更有害（Enqueue 去重先例）
	var existing models.ConfigRestoreTask
	err = s.db.WithContext(ctx).
		Where("device_id = ? AND status IN (?)", deviceID, []models.RestoreTaskStatus{
			models.RestoreTaskStatusPending,
			models.RestoreTaskStatusRunning,
		}).
		First(&existing).Error
	if err == nil {
		return nil, fmt.Errorf("该设备存在进行中的恢复任务")
	}

	// ③ 创建任务（pending）。同设备活跃唯一索引（migration 212，v129-recheck C-1）
	// 使插入即夺锁：上方 ② 的预查只是快速路径，真正的互斥原子性由索引保证——
	// 并发窗口内第二个 Create 命中唯一冲突，此处识别并归一为互斥错误。
	task := &models.ConfigRestoreTask{
		DeviceID:  deviceID,
		BackupID:  backupID,
		Status:    string(models.RestoreTaskStatusPending),
		CreatedBy: createdBy,
	}
	if err := s.db.WithContext(ctx).Create(task).Error; err != nil {
		if isDuplicateActiveRestoreErr(err) {
			return nil, fmt.Errorf("该设备存在进行中的恢复任务")
		}
		return nil, fmt.Errorf("创建恢复任务失败: %w", err)
	}

	// ④ 异步执行——detached context（P1：HTTP ctx 随响应取消，任务必须独立生命周期）
	runCtx, cancel := context.WithTimeout(context.Background(), constants.RestoreConfigTimeout)
	go func() {
		defer cancel()
		s.runRestore(runCtx, task.ID)
	}()

	return task, nil
}

// restoreRunResult 任务 result JSON 的载体（RestoreResult + 回读校验，D-07/D-11）
type restoreRunResult struct {
	TotalLines   int    `json:"totalLines"`
	SentLines    int    `json:"sentLines"`
	FailedLine   string `json:"failedLine,omitempty"`
	HashMatched  bool   `json:"hashMatched"`
	RestoredHash string `json:"restoredHash,omitempty"`
}

// runRestore 异步执行恢复全流程（D-10/14 → D-23 → D-02/12 → D-11 → D-09 → 终态）。
// 入口 defer recover 防 goroutine panic 把任务悬挂在 running（T-93-08）。
func (s *ConfigRestoreTaskService) runRestore(ctx context.Context, taskID string) {
	defer func() {
		if r := recover(); r != nil {
			applogger.Errorf("[配置恢复] 任务 panic (taskID=%s): %v", taskID, r)
			s.failTask(taskID, fmt.Sprintf("恢复执行异常: %v", r))
		}
	}()

	task, err := s.GetRestoreTask(context.Background(), taskID)
	if err != nil {
		applogger.Errorf("[配置恢复] 任务不存在 (taskID=%s): %v", taskID, err)
		return
	}

	// nil executor 防护：handler 测试/未装配环境安全降级为 failed 而非 panic
	if s.executor == nil {
		s.failTask(taskID, "设备执行器未初始化")
		return
	}

	// running
	now := time.Now()
	if err := s.db.WithContext(ctx).Model(task).Updates(map[string]interface{}{
		"status":     string(models.RestoreTaskStatusRunning),
		"started_at": &now,
	}).Error; err != nil {
		applogger.Errorf("[配置恢复] 任务置 running 失败 (taskID=%s): %v", taskID, err)
	}

	// 源备份记录：DeviceName（⑤ 备份文件名 / ⑨ 版本链记录）与 ⑧ hash 比对都依赖。
	// StartRestore 已校验过存在性与同设备，这里重取防御备份在排队间隙被删除。
	backup, err := s.backupSvc.GetBackupByID(ctx, task.BackupID)
	if err != nil {
		s.failTask(taskID, fmt.Sprintf("备份记录不存在: %v", err))
		return
	}

	// ⑤ 恢复前自动备份（D-10）：CreateBackup 回读设备当前配置落库，失败即中止
	// （D-14）——无法备份 = 无回退退路，不下发
	preResult, err := s.backupSvc.CreateBackup(ctx, &BackupRequest{
		DeviceID:      task.DeviceID,
		DeviceName:    backup.DeviceName, // 设备真实名（文件名 sanitize 即用此值）
		BackupType:    models.BackupTypeAuto,
		ChangeReason:  "恢复前自动备份",
		CreatedBy:     "restore",
		CompressLarge: true,
	})
	if err != nil {
		s.failTask(taskID, fmt.Sprintf("恢复前自动备份失败，已中止恢复: %v", err))
		return
	}
	applogger.Infof("[配置恢复] 恢复前自动备份完成 (taskID=%s, backupID=%s)", taskID, preResult.BackupID)

	// ⑥ 读备份内容（走 93-01 的解压双检查路径）
	config, err := s.backupSvc.GetBackupContent(ctx, task.BackupID)
	if err != nil {
		s.failTask(taskID, fmt.Sprintf("读取备份内容失败: %v", err))
		return
	}

	// ⑦ 下发（D-02 RestoreConfig 唯一入口；D-12 fail-fast + 进度留痕）
	result, restoreErr := s.executor.RestoreConfig(ctx, task.DeviceID, config)
	if restoreErr != nil {
		if result != nil {
			// v129-recheck WR-01: 保留设备返回的真实错误文本（如 "Invalid input
			// detected at '^' marker"），否则 error_message 恒空、进度留痕半残
			s.failTaskWithResult(task, result, fmt.Sprintf("配置下发失败: %v", restoreErr))
			return
		}
		s.failTask(taskID, fmt.Sprintf("配置下发失败: %v", restoreErr))
		return
	}

	// ⑧ 回读一致性校验（D-11）：真实设备回读与备份文本完全一致几乎不可能
	// （版本差异/自动补全/顺序重排），不一致仅警告不算失败——下发已不可逆。
	// 回读失败同样只警告：版本链记录以下发的备份原文兜底（hash 与备份一致）。
	restored := config
	restoredHash := backup.ConfigHash
	hashMatched := false
	if readback, rerr := s.executor.GetConfig(ctx, task.DeviceID); rerr != nil {
		applogger.Warnf("[配置恢复] 回读配置失败 (taskID=%s): %v", taskID, rerr)
	} else {
		restored = readback
		restoredHash = calculateHash(readback)
		hashMatched = restoredHash == backup.ConfigHash
		if !hashMatched {
			applogger.Warnf("[配置恢复] 回读 hash 与备份不一致 (taskID=%s, restored=%s)", taskID, restoredHash)
		}
	}

	// ⑨ 版本链恢复记录（D-09）：复用 BackupTypeManual + ChangeReason 区分（统计链路零改动）
	nextVersion := 1
	{
		var latest models.ConfigBackup
		s.db.WithContext(ctx).Where("device_id = ?", task.DeviceID).Order("version DESC").First(&latest)
		if latest.ID != "" {
			nextVersion = latest.Version + 1
		}
	}
	changeReason := fmt.Sprintf("恢复自版本 %d", backup.Version)
	record := &models.ConfigBackup{
		DeviceID:      task.DeviceID,
		DeviceName:    backup.DeviceName,
		BackupType:    models.BackupTypeManual,
		StorageType:   models.StorageTypeDatabase,
		ConfigContent: restored,
		ConfigHash:    restoredHash,
		BackupSize:    len(restored),
		Version:       nextVersion,
		ChangeReason:  changeReason,
		CreatedBy:     task.CreatedBy,
	}
	if err := s.db.WithContext(ctx).Create(record).Error; err != nil {
		applogger.Warnf("[配置恢复] 版本链恢复记录写入失败 (taskID=%s): %v", taskID, err)
	}

	// ⑩ 任务 success
	runResult := restoreRunResult{
		TotalLines:   result.TotalLines,
		SentLines:    result.SentLines,
		HashMatched:  hashMatched,
		RestoredHash: restoredHash,
	}
	payload, _ := json.Marshal(runResult)
	completed := time.Now()
	if err := s.db.WithContext(ctx).Model(task).Updates(map[string]interface{}{
		"status":       string(models.RestoreTaskStatusSuccess),
		"total_lines":  result.TotalLines,
		"sent_lines":   result.SentLines,
		"result_json":  string(payload),
		"completed_at": &completed,
	}).Error; err != nil {
		applogger.Errorf("[配置恢复] 任务置 success 失败 (taskID=%s): %v", taskID, err)
	}
	applogger.Infof("[配置恢复] 任务成功 (taskID=%s, %d/%d 行, hashMatched=%v)", taskID, result.SentLines, result.TotalLines, hashMatched)
}

// failTask 将任务置为 failed（无结果载荷）
func (s *ConfigRestoreTaskService) failTask(taskID, message string) {
	updates := map[string]interface{}{
		"status":        string(models.RestoreTaskStatusFailed),
		"error_message": message,
		"completed_at":  time.Now(),
	}
	if err := s.db.Model(&models.ConfigRestoreTask{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
		applogger.Errorf("[配置恢复] 任务置 failed 失败 (taskID=%s): %v", taskID, err)
	}
	applogger.Warnf("[配置恢复] 任务失败 (taskID=%s): %s", taskID, message)
}

// failTaskWithResult 将任务置为 failed 并保留部分下发进度（D-12 留痕）
func (s *ConfigRestoreTaskService) failTaskWithResult(task *models.ConfigRestoreTask, result *device.RestoreResult, message string) {
	payload, _ := json.Marshal(restoreRunResult{
		TotalLines: result.TotalLines,
		SentLines:  result.SentLines,
		FailedLine: result.FailedLine,
	})
	updates := map[string]interface{}{
		"status":        string(models.RestoreTaskStatusFailed),
		"total_lines":   result.TotalLines,
		"sent_lines":    result.SentLines,
		"failed_line":   result.FailedLine,
		"result_json":   string(payload),
		"error_message": message,
		"completed_at":  time.Now(),
	}
	if err := s.db.Model(task).Updates(updates).Error; err != nil {
		applogger.Errorf("[配置恢复] 任务置 failed 失败 (taskID=%s): %v", task.ID, err)
	}
	applogger.Warnf("[配置恢复] 任务失败 (taskID=%s, 已发 %d/%d 行, 失败行: %q)", task.ID, result.SentLines, result.TotalLines, result.FailedLine)
}

// GetRestoreTask 查询恢复任务详情
func (s *ConfigRestoreTaskService) GetRestoreTask(ctx context.Context, id string) (*models.ConfigRestoreTask, error) {
	var task models.ConfigRestoreTask
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&task).Error; err != nil {
		return nil, fmt.Errorf("恢复任务不存在: %w", err)
	}
	return &task, nil
}

// ListRestoreTasks 分页查询恢复任务列表（分页常量经 NormalizePagination，
// 排序走 base.ApplySort 白名单——与 GetBackupList 同款模式）
func (s *ConfigRestoreTaskService) ListRestoreTasks(ctx context.Context, current, pageSize int, deviceID, orderByColumn string, isAsc *bool) ([]models.ConfigRestoreTask, int64, error) {
	current, pageSize = query.NormalizePagination(current, pageSize)

	var tasks []models.ConfigRestoreTask
	var total int64

	q := s.db.WithContext(ctx).Model(&models.ConfigRestoreTask{})
	if deviceID != "" {
		q = q.Where("device_id = ?", deviceID)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询恢复任务总数失败: %w", err)
	}

	sortReq := base.BaseListRequest{
		Current:       current,
		PageSize:      pageSize,
		OrderByColumn: orderByColumn,
		IsAsc:         isAsc,
	}
	q = base.ApplySort(q, sortReq, restoreTaskAllowedSortFields)
	if orderByColumn == "" {
		q = q.Order("created_at DESC")
	}
	if err := q.Offset((current - 1) * pageSize).Limit(pageSize).Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("查询恢复任务列表失败: %w", err)
	}
	return tasks, total, nil
}

// RecoverStaleRunningTasks 启动收敛：进程崩溃/重启残留的 running 任务一次性
// 置 failed（A5 discretion 方案①——状态机自洽性收口，防互斥锁死，非清理 cron，
// 不违 D-20）。由装配点启动时调用一次。
//
// v129-recheck C-2：pending 一并收敛——StartRestore 的认领 goroutine 仅存在于
// 进程内存（:77 提交 pending 后才启动），重启后残留 pending 永远无人认领，
// 而互斥查询（:61-63）把 pending 计为进行中，遗留即永久锁死该设备恢复
// （D-34 无取消端点）。收敛为 failed 语义正确。
func (s *ConfigRestoreTaskService) RecoverStaleRunningTasks(ctx context.Context) {
	res := s.db.WithContext(ctx).
		Model(&models.ConfigRestoreTask{}).
		Where("status IN (?)", []models.RestoreTaskStatus{
			models.RestoreTaskStatusPending,
			models.RestoreTaskStatusRunning,
		}).
		Updates(map[string]interface{}{
			"status":        string(models.RestoreTaskStatusFailed),
			"error_message": "服务重启，任务中断",
			"completed_at":  time.Now(),
		})
	if res.Error != nil {
		applogger.Errorf("[配置恢复] 收敛残留任务失败: %v", res.Error)
		return
	}
	if res.RowsAffected > 0 {
		applogger.Infof("[配置恢复] 启动收敛: %d 个残留 pending/running 任务已置 failed", res.RowsAffected)
	}
}
