package dbbackup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/query"
)

// 异步任务分段 context 预算（独立生命周期，绝不使用 HTTP ctx）。
// 参考 config_restore_task_service.go 的「两段独立 budget」模式。
const (
	stageTableEnumTimeout = 2 * time.Minute
	stageDumpTimeout      = 60 * time.Minute // GB 级全库导出
	stageManifestTimeout  = 30 * time.Minute // 流式打包 + sha256
	stageRemoteTimeout    = 30 * time.Minute
	stagePreBackupTimeout = 60 * time.Minute // 恢复前自动全量备份
	stageRestoreTimeout   = 60 * time.Minute
)

// 任务进度与阶段标识（写进 sys_db_backup_task.progress / stage）
const (
	stageInit       = "初始化"
	stageEnumTables = "枚举物理表"
	stageDump       = "转储数据库"
	stagePack       = "生成备份容器"
	stageUpload     = "上传远程存储"
	stageCleanup    = "保留份数清理"
	stageVerify     = "校验备份容器"
	stagePreRestore = "恢复前自动备份"
	stageExtract    = "解压并校验 dump"
	stageRestore    = "执行恢复"

	progressInit    = 0
	progressEnum    = 10
	progressDump    = 30
	progressPack    = 60
	progressUpload  = 80
	progressCleanup = 90
	progressVerify  = 10
	progressPreBak  = 30
	progressExtract = 50
	progressRestore = 70
	progressDone    = 100
)

// service 全库备份服务私有实现。
//
// 并发互斥：taskMu 保证同一进程内同一时刻只有一个 export/restore 任务推进
// （同库不允许两个备份/恢复并发）。多实例部署时以 sys_db_backup_task 的
// pending/running 活跃状态行作为跨实例事实源：任一实例在 Trigger* 前查活跃
// 任务，存在即拒绝。注意：本项目当前 deployment 无 DB 分布式锁（PLAN.md 实锤），
// 跨实例互斥只能做到「查活跃行」的弱一致（两实例同时查到无活跃再各自 Create
// 存在极小竞态窗口）；进程内互斥由 taskMu 严格保证，注释在此说明限制。
// 若未来需要强跨实例互斥，应在迁移里为「活跃任务唯一状态行」加部分唯一索引。
type service struct {
	db        *gorm.DB
	dbCfg     DatabaseConfig
	exec      DumpExecutor
	backupDir string
	taskMu    sync.Mutex
}

// newService 构造服务（契约 NewService 的私有实现）。
// exec 为 nil 时装配生产 DumpExecutor；backupDir 为空时用默认 data/backups。
func newService(db *gorm.DB, dbCfg DatabaseConfig, exec DumpExecutor, backupDir string) Service {
	if strings.TrimSpace(backupDir) == "" {
		backupDir = filepath.Join("data", "backups")
	}
	if exec == nil {
		exec = newProductionDumper(db)
	}
	return &service{
		db:        db,
		dbCfg:     dbCfg,
		exec:      exec,
		backupDir: backupDir,
	}
}

// backupLayout 生成落盘路径：{backupDir}/{YYYY}/{MM}/{ts}_{dbtype}.zip
// ts 含纳秒避免同秒并发文件名碰撞；文件名部件全部经清洗防路径逃逸。
func (s *service) backupLayout(now time.Time) (absPath, remoteName string, err error) {
	dbType := strings.ToLower(strings.TrimSpace(s.dbCfg.Type))
	if isSQLiteCfg(s.dbCfg) {
		dbType = dbTypeSQLite
	} else if dbType == "" {
		dbType = dbTypePostgres
	}
	ts := now.Format("20060102_150405.000000000")
	fileName := sanitizeEntryName(fmt.Sprintf("%s_%s.zip", ts, dbType))
	root, err := filepath.Abs(s.backupDir)
	if err != nil {
		return "", "", fmt.Errorf("解析备份根目录: %w", err)
	}
	dir := filepath.Join(root, now.Format("2006"), now.Format("01"))
	abs := filepath.Join(dir, fileName)
	// 逃逸校验：拼接后的路径必须仍在 root 之下
	if !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("备份路径越出根目录: %s", abs)
	}
	remote := now.Format("2006") + "/" + now.Format("01") + "/" + fileName
	return abs, remote, nil
}

// toRecord 把 model 转为对外视图 DTO
func toRecord(b *models.DBBackup) *BackupRecord {
	return &BackupRecord{
		ID:            b.ID,
		FileName:      b.FileName,
		FilePath:      b.FilePath,
		StorageType:   b.StorageType,
		RemotePath:    b.RemotePath,
		DBType:        b.DBType,
		FileSize:      b.FileSize,
		Sha256:        b.Sha256,
		SchemaVersion: b.SchemaVersion,
		TableCount:    b.TableCount,
		Source:        b.Source,
		Status:        b.Status,
		TriggeredBy:   b.TriggeredBy,
		Remark:        b.Remark,
		CreatedAt:     b.CreatedAt,
	}
}

// toTaskView 把任务 model 转为对外视图 DTO
func toTaskView(t *models.DBBackupTask) *TaskView {
	return &TaskView{
		ID:           t.ID,
		TaskType:     t.TaskType,
		BackupID:     t.BackupID,
		Status:       t.Status,
		Progress:     t.Progress,
		Stage:        t.Stage,
		ErrorMessage: t.ErrorMessage,
		StartedAt:    t.StartedAt,
		CompletedAt:  t.CompletedAt,
		CreatedAt:    t.CreatedAt,
	}
}

// List 备份记录分页（current 从 1 开始，pageSize 经统一归一化）。
// 默认不展示已删除记录，按创建时间倒序。
func (s *service) List(ctx context.Context, current, pageSize int) (*ListResult, error) {
	current, pageSize = query.NormalizePagination(current, pageSize)
	var records []models.DBBackup
	var total int64
	q := s.db.WithContext(ctx).Model(&models.DBBackup{}).Where("status <> ?", StatusDeleted)
	if err := q.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计备份记录总数: %w", err)
	}
	if err := q.Order("created_at DESC").
		Offset((current - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("查询备份记录列表: %w", err)
	}
	list := make([]BackupRecord, 0, len(records))
	for i := range records {
		list = append(list, *toRecord(&records[i]))
	}
	return &ListResult{List: list, Total: total}, nil
}

// Get 查询单条备份记录
func (s *service) Get(ctx context.Context, id string) (*BackupRecord, error) {
	var b models.DBBackup
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&b).Error; err != nil {
		return nil, fmt.Errorf("备份记录不存在: %w", err)
	}
	return toRecord(&b), nil
}

// GetTask 查询异步任务状态（供前端轮询）
func (s *service) GetTask(ctx context.Context, id string) (*TaskView, error) {
	var t models.DBBackupTask
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&t).Error; err != nil {
		return nil, fmt.Errorf("备份任务不存在: %w", err)
	}
	return toTaskView(&t), nil
}

// execToolCfg 返回带已解析 PostgreSQL 工具路径的 DatabaseConfig 副本。
// SQLite 模式或注入的 fake DumpExecutor 不需要外部工具，直接原样返回。
// 缺工具时返回带安装/配置指引的错误（不再让 exec 抛 "not found in $PATH"）。
func (s *service) execToolCfg(ctx context.Context, which string) (DatabaseConfig, error) {
	cfg := s.dbCfg
	if isSQLiteCfg(cfg) {
		return cfg, nil
	}
	path, ok, hint := resolvePGToolPath(ctx, s.db, cfg, which)
	if !ok {
		return cfg, fmt.Errorf("转储/恢复工具不可用: %s", hint)
	}
	if which == "pg_dump" {
		cfg.PgDumpPath = path
	} else {
		cfg.PgRestorePath = path
	}
	return cfg, nil
}

// Capabilities 返回当前部署环境的能力预检，供页面在触发导出前提示工具状态。
func (s *service) Capabilities(ctx context.Context) (*Capabilities, error) {
	cap := &Capabilities{DBType: dbTypeOf(s.dbCfg)}
	if isSQLiteCfg(s.dbCfg) {
		cap.PgDumpAvailable = true
		cap.PgRestoreAvailable = true
		cap.Hint = "SQLite 模式：导出走纯 Go VACUUM INTO，无需外部工具；整库恢复需停服后离线执行。"
		return cap, nil
	}
	pDump, okDump, hintDump := resolvePGToolPath(ctx, s.db, s.dbCfg, "pg_dump")
	pRestore, okRestore, hintRestore := resolvePGToolPath(ctx, s.db, s.dbCfg, "pg_restore")
	cap.PgDumpPath = pDump
	cap.PgRestorePath = pRestore
	cap.PgDumpAvailable = okDump
	cap.PgRestoreAvailable = okRestore
	switch {
	case !okDump:
		cap.Hint = hintDump
	case !okRestore:
		cap.Hint = hintRestore
	default:
		cap.Hint = "pg_dump / pg_restore 均可用。"
	}
	return cap, nil
}

// Statistics 统计卡数据：总数、总字节、按来源/状态分布、最近备份时间。
func (s *service) Statistics(ctx context.Context) (map[string]interface{}, error) {
	var total int64
	var sizeSum int64
	if err := s.db.WithContext(ctx).Model(&models.DBBackup{}).
		Where("status <> ?", StatusDeleted).
		Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计备份总数: %w", err)
	}
	// 用 COALESCE 兼容空表；避免把 NULL 扫进 int64
	if err := s.db.WithContext(ctx).Model(&models.DBBackup{}).
		Where("status <> ?", StatusDeleted).
		Select("COALESCE(SUM(file_size), 0)").Scan(&sizeSum).Error; err != nil {
		return nil, fmt.Errorf("统计备份总大小: %w", err)
	}

	bySource := map[string]int64{}
	var sourceRows []struct {
		Source string
		Cnt    int64
	}
	if err := s.db.WithContext(ctx).Model(&models.DBBackup{}).
		Select("source, COUNT(*) AS cnt").
		Where("status <> ?", StatusDeleted).
		Group("source").Scan(&sourceRows).Error; err != nil {
		return nil, fmt.Errorf("统计备份来源分布: %w", err)
	}
	for _, r := range sourceRows {
		bySource[r.Source] = r.Cnt
	}

	byStatus := map[string]int64{}
	var statusRows []struct {
		Status string
		Cnt    int64
	}
	if err := s.db.WithContext(ctx).Model(&models.DBBackup{}).
		Select("status, COUNT(*) AS cnt").Group("status").Scan(&statusRows).Error; err != nil {
		return nil, fmt.Errorf("统计备份状态分布: %w", err)
	}
	for _, r := range statusRows {
		byStatus[r.Status] = r.Cnt
	}

	var latest models.DBBackup
	lastAt := interface{}(nil)
	if err := s.db.WithContext(ctx).Where("status <> ?", StatusDeleted).
		Order("created_at DESC").First(&latest).Error; err == nil {
		lastAt = latest.CreatedAt
	}

	return map[string]interface{}{
		"total":      total,
		"totalSize":  sizeSum,
		"bySource":   bySource,
		"byStatus":   byStatus,
		"lastBackup": lastAt,
		// 前端统计卡直读别名（quick 261110-tbq 集成对齐）：success=status=ready 的份数，
		// usedSpace=总字节数。保留 totalSize/byStatus 供其它消费方，别名不破坏既有键。
		"success":   byStatus[StatusReady],
		"usedSpace": sizeSum,
	}, nil
}

// GetPlan 读取备份计划（每调用重读 sys_config，改配置无需重启）。
// 用降级读取：凭据解密失败（如 SM4_KEY 被更换）时相应字段置空返回计划主体，
// 保证备份配置页不因解密失败整页不可用；密文绝不回吐，空串=「未配置」。
func (s *service) GetPlan(ctx context.Context) (*Plan, error) {
	return loadPlanTolerant(ctx, s.db)
}

// SavePlan 保存备份计划
func (s *service) SavePlan(ctx context.Context, p *Plan, operator string) error {
	return savePlan(ctx, s.db, p, operator)
}

// hasActiveTask 查跨实例事实源：是否存在 pending/running 的 export/restore 任务。
func (s *service) hasActiveTask(ctx context.Context) (bool, error) {
	var cnt int64
	if err := s.db.WithContext(ctx).Model(&models.DBBackupTask{}).
		Where("status IN ?", []string{models.DBBackupTaskPending, models.DBBackupTaskRunning}).
		Count(&cnt).Error; err != nil {
		return false, fmt.Errorf("查询活跃备份任务: %w", err)
	}
	return cnt > 0, nil
}

// TriggerExport 建 pending 任务并异步执行，立即返回 taskID。
// 互斥：进程内 taskMu.TryLock + 任务表活跃状态行双重门禁。
func (s *service) TriggerExport(ctx context.Context, source, operator string) (string, error) {
	if strings.TrimSpace(source) == "" {
		source = SourceManual
	}
	active, err := s.hasActiveTask(ctx)
	if err != nil {
		return "", err
	}
	if active {
		return "", fmt.Errorf("已有进行中的备份/恢复任务，请稍后再试")
	}
	// 进程内快速互斥（与 runTask 内的持锁形成闭环）
	if !s.taskMu.TryLock() {
		return "", fmt.Errorf("已有进行中的备份/恢复任务，请稍后再试")
	}
	s.taskMu.Unlock() // 只用它探测，不在此长期持锁；真正的串行在 runTask

	task := &models.DBBackupTask{
		TaskType:  models.DBBackupTaskTypeExport,
		Status:    models.DBBackupTaskPending,
		Progress:  progressInit,
		Stage:     stageInit,
		CreatedBy: operator,
		// source/operator 通过 runTask 闭包参数传递（任务表无 source 列），
		// 不塞进 Stage，避免污染阶段语义。
	}
	// Omit("backup_id")：PG 迁移 240 把 backup_id 建为 uuid 列，GORM 对空
	// string 会写 '' 导致 22P02；省略该列让它落 NULL（sqlite 同样安全）。
	if err := s.db.WithContext(ctx).Omit("backup_id").Create(task).Error; err != nil {
		return "", fmt.Errorf("创建导出任务: %w", err)
	}
	// detached：HTTP ctx 随响应取消，任务必须独立生命周期
	go s.runTask(context.Background(), task.ID, models.DBBackupTaskTypeExport, source, operator, "")
	return task.ID, nil
}

// TriggerRestore 上传文件已落临时路径，建任务并异步恢复。
func (s *service) TriggerRestore(ctx context.Context, uploadPath, operator string) (string, error) {
	if strings.TrimSpace(uploadPath) == "" {
		return "", fmt.Errorf("上传文件路径为空")
	}
	if _, err := os.Stat(uploadPath); err != nil {
		return "", fmt.Errorf("上传文件不存在: %w", err)
	}
	active, err := s.hasActiveTask(ctx)
	if err != nil {
		return "", err
	}
	if active {
		return "", fmt.Errorf("已有进行中的备份/恢复任务，请稍后再试")
	}
	if !s.taskMu.TryLock() {
		return "", fmt.Errorf("已有进行中的备份/恢复任务，请稍后再试")
	}
	s.taskMu.Unlock()

	task := &models.DBBackupTask{
		TaskType:  models.DBBackupTaskTypeRestore,
		Status:    models.DBBackupTaskPending,
		Progress:  progressInit,
		Stage:     stageInit,
		CreatedBy: operator,
	}
	// 同上：backup_id 为 PG uuid 列，省略以落 NULL
	if err := s.db.WithContext(ctx).Omit("backup_id").Create(task).Error; err != nil {
		return "", fmt.Errorf("创建恢复任务: %w", err)
	}
	go s.runTask(context.Background(), task.ID, models.DBBackupTaskTypeRestore, "", operator, uploadPath)
	return task.ID, nil
}

// runTask 异步任务主循环：defer recover + 进程内串行 + 分段 context timeout +
// 状态机 pending→running→success|failed，全程写 Progress/Stage/ErrorMessage。
//
// taskMu 在整个任务生命周期持有，保证同库不并发 export/restore。
//
// 上传临时文件清理（agent-7 跨层约定）：handler 用 os.CreateTemp 把 zip 写到系统
// 临时目录且不 defer 删除（避免异步竞态），本 goroutine 在**收尾**（success /
// failed / panic recover 全部路径）负责 os.Remove(uploadPath)。路径由闭包参数
// 捕获，保证 recover 路径也能删到，不依赖任务表字段。
func (s *service) runTask(baseCtx context.Context, taskID, taskType, source, operator, uploadPath string) {
	isRestore := taskType == models.DBBackupTaskTypeRestore
	// 上传临时文件清理只执行一次（正常收尾或 panic 兜底都可能触发）
	var cleanupOnce sync.Once
	cleanupUpload := func() {
		cleanupOnce.Do(func() {
			if uploadPath == "" {
				return
			}
			if err := os.Remove(uploadPath); err != nil && !os.IsNotExist(err) {
				applogger.Warnf("[全库备份] 清理上传临时文件失败 (taskID=%s, path=%s): %v",
					taskID, uploadPath, err)
			}
		})
	}
	// 恢复一致性收口只执行一次：正常路径显式同步调用，panic 路径由 defer 兜底，
	// 两条路径互斥但都经过本闭包，用 Once 保证「清缓存 + 记录会话失效」不重不漏。
	// 详见 restore_consistency.go（本项目无全局会话失效入口，采用「需重新登录」最小实现）。
	var consistencyOnce sync.Once
	finalizeConsistency := func() {
		if !isRestore {
			return
		}
		consistencyOnce.Do(func() {
			s.finalizeRestoreConsistency(baseCtx, taskID)
		})
	}
	// defer 只做 panic 兜底：cleanup + 恢复一致性收口 + recover + failTask。
	// 正常路径的清理由下方终态写入前显式同步完成，不依赖 defer 调度时机。
	defer func() {
		cleanupUpload()
		if r := recover(); r != nil {
			applogger.Errorf("[全库备份] 任务 panic (taskID=%s): %v", taskID, r)
			finalizeConsistency()
			msg := fmt.Sprintf("任务执行异常: %v", r)
			if isRestore {
				msg = withReloginNotice(msg)
			}
			s.failTask(taskID, msg)
		}
	}()

	s.taskMu.Lock()
	defer s.taskMu.Unlock()

	// running
	now := time.Now()
	if err := s.db.WithContext(baseCtx).Model(&models.DBBackupTask{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":     models.DBBackupTaskRunning,
			"started_at": &now,
			"stage":      stageInit,
			"progress":   progressInit,
		}).Error; err != nil {
		applogger.Errorf("[全库备份] 任务置 running 失败 (taskID=%s): %v", taskID, err)
		return
	}

	var runErr error
	if isRestore {
		runErr = s.runRestore(baseCtx, taskID, uploadPath, operator)
	} else {
		runErr = s.runExport(baseCtx, taskID, source, operator)
	}
	// 先清理上传临时文件、再写任务终态：
	// waitTask 一旦从 DB 读到 success/failed 就会返回并立刻 os.Stat 校验文件已删，
	// 若把 Remove 放在终态写入之后会留下竞态窗口（全量并发跑偶发 FAIL）。
	// 此处保证「查到终态」时文件必已删除；panic recover 的 defer 仍兜底。
	cleanupUpload()

	// 恢复任务收尾一致性（success / failed 两条路径都必须经过，panic 由 defer 兜底）：
	// 整库恢复替换了 sys_user/sys_menu/sys_role 等鉴权表，必须清进程内菜单/权限
	// 缓存，并在任务结果里写明确「需重新登录」。详见 restore_consistency.go。
	finalizeConsistency()

	if runErr != nil {
		msg := runErr.Error()
		if isRestore {
			msg = withReloginNotice(msg)
		}
		s.failTask(taskID, msg)
	} else {
		s.succeedTask(taskID, isRestore)
	}
}

// updateProgress 写任务进度与阶段
func (s *service) updateProgress(taskID string, progress int, stage string) {
	if err := s.db.Model(&models.DBBackupTask{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{"progress": progress, "stage": stage}).Error; err != nil {
		applogger.Warnf("[全库备份] 写任务进度失败 (taskID=%s): %v", taskID, err)
	}
}

// failTask 收敛终态 failed
func (s *service) failTask(taskID, message string) {
	now := time.Now()
	if err := s.db.Model(&models.DBBackupTask{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":        models.DBBackupTaskFailed,
			"error_message": message,
			"completed_at":  &now,
		}).Error; err != nil {
		applogger.Errorf("[全库备份] 任务置 failed 失败 (taskID=%s): %v", taskID, err)
	}
	applogger.Warnf("[全库备份] 任务失败 (taskID=%s): %s", taskID, message)
}

// succeedTask 收敛终态 success。
// isRestore=true 时把 stage 写成「完成（需重新登录）」：整库恢复替换了鉴权表，
// 但本项目无全局会话失效入口（TokenBlacklistService 只能按已知 token 逐个拉黑），
// 故在任务结果里明确要求前端/用户重新登录，以丢弃旧 token 并重建会话。
// 前端会据此常驻提示，禁止留空。
func (s *service) succeedTask(taskID string, isRestore bool) {
	stage := stageRestoreDone
	if isRestore {
		stage = stageRestoreDoneWithRelogin
	}
	now := time.Now()
	if err := s.db.Model(&models.DBBackupTask{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":        models.DBBackupTaskSuccess,
			"progress":      progressDone,
			"stage":         stage,
			"error_message": "",
			"completed_at":  &now,
		}).Error; err != nil {
		applogger.Errorf("[全库备份] 任务置 success 失败 (taskID=%s): %v", taskID, err)
	}
}

// stageCtx 创建分段独立 context（生命周期脱离 HTTP ctx）
func stageCtx(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// runExport 导出全流程：枚举表 → dump → 流式打包 → 落记录 → 上传远程 → 保留清理。
// Storage.Put 失败不阻断本地备份成功（记 remote 状态 + 返回错误供任务记录）。
func (s *service) runExport(baseCtx context.Context, taskID, source, operator string) error {
	// ① 枚举物理表（独立短预算）
	s.updateProgress(taskID, progressEnum, stageEnumTables)
	enumCtx, cancelEnum := stageCtx(stageTableEnumTimeout)
	tables, err := s.exec.TableNames(enumCtx, s.dbCfg)
	cancelEnum()
	if err != nil {
		return fmt.Errorf("枚举物理表: %w", err)
	}

	// ② 转储到临时 dump 文件
	s.updateProgress(taskID, progressDump, stageDump)
	now := time.Now()
	zipPath, remoteName, err := s.backupLayout(now)
	if err != nil {
		return err
	}
	dumpPath := zipPath + ".dump.tmp"
	defer func() { _ = os.Remove(dumpPath) }()

	dumpCtx, cancelDump := stageCtx(stageDumpTimeout)
	defer cancelDump()
	// 解析 pg_dump 可执行路径（sys_config → config.yaml → PATH）；缺工具时在此
	// 直接给出可执行的安装/配置诊断，避免把 exec: "pg_dump": not found 抛给用户。
	execCfg, err := s.execToolCfg(baseCtx, "pg_dump")
	if err != nil {
		return err
	}
	produced, err := s.exec.Dump(dumpCtx, dumpPath, execCfg)
	cancelDump()
	if err != nil {
		return fmt.Errorf("转储数据库: %w", err)
	}
	if produced != "" {
		dumpPath = produced
	}

	// ③ 流式打包（manifest + 双层 sha256）
	s.updateProgress(taskID, progressPack, stagePack)
	meta := &manifest{
		SchemaVersion:  SchemaVersionV1,
		CreatedAt:      now.UTC(),
		DBType:         dbTypeOf(s.dbCfg),
		Tables:         tables,
		ExcludedTables: append([]string{}, excludedPhysicalTables...),
	}
	packCtx, cancelPack := stageCtx(stageManifestTimeout)
	defer cancelPack()
	containerSha, err := writeContainer(packCtx, zipPath, dumpPath, meta)
	if err != nil {
		_ = os.Remove(zipPath)
		return fmt.Errorf("生成备份容器: %w", err)
	}

	info, err := os.Stat(zipPath)
	if err != nil {
		return fmt.Errorf("读取备份容器信息: %w", err)
	}

	// ④ 落备份记录（本地先成功）
	plan, err := loadPlan(baseCtx, s.db)
	if err != nil {
		return fmt.Errorf("读取备份计划: %w", err)
	}
	record := &models.DBBackup{
		FileName:      filepath.Base(zipPath),
		FilePath:      zipPath,
		StorageType:   string(plan.Storage),
		DBType:        dbTypeOf(s.dbCfg),
		FileSize:      info.Size(),
		Sha256:        containerSha,
		SchemaVersion: SchemaVersionV1,
		TableCount:    len(tables),
		Source:        source,
		Status:        StatusReady,
		TriggeredBy:   operator,
	}
	if err := s.db.WithContext(baseCtx).Create(record).Error; err != nil {
		_ = os.Remove(zipPath)
		return fmt.Errorf("保存备份记录: %w", err)
	}
	// 回填任务 BackupID
	_ = s.db.WithContext(baseCtx).Model(&models.DBBackupTask{}).
		Where("id = ?", taskID).Update("backup_id", record.ID).Error

	// ⑤ 上传远程：失败不阻断本地成功，记录 remote_only 状态并把错误返回供任务记录
	s.updateProgress(taskID, progressUpload, stageUpload)
	var remoteErr error
	store, err := newStorage(plan)
	if err == nil {
		remoteCtx, cancelRemote := stageCtx(stageRemoteTimeout)
		remotePath, putErr := store.Put(remoteCtx, zipPath, remoteName)
		cancelRemote()
		if putErr == nil {
			_ = s.db.WithContext(baseCtx).Model(&models.DBBackup{}).
				Where("id = ?", record.ID).
				Updates(map[string]interface{}{"remote_path": remotePath}).Error
		} else {
			remoteErr = putErr
		}
	} else {
		remoteErr = fmt.Errorf("构造存储适配器: %w", err)
	}
	if remoteErr != nil {
		_ = s.db.WithContext(baseCtx).Model(&models.DBBackup{}).
			Where("id = ?", record.ID).
			Updates(map[string]interface{}{
				"status": StatusRemoteOnly,
				"remark": "远程上传失败: " + remoteErr.Error(),
			}).Error
		applogger.Warnf("[全库备份] 远程上传失败 (backupID=%s): %v", record.ID, remoteErr)
	}

	// ⑥ 保留份数清理
	s.updateProgress(taskID, progressCleanup, stageCleanup)
	if err := s.enforceRetention(baseCtx, plan); err != nil {
		applogger.Warnf("[全库备份] 保留份数清理失败 (taskID=%s): %v", taskID, err)
	}

	if remoteErr != nil {
		return fmt.Errorf("本地备份成功但远程上传失败: %w", remoteErr)
	}
	return nil
}

// runRestore 恢复全流程：
// ① 校验容器 → ② 恢复前自动全量备份（失败即中止）→ ③ 解压校验 → ④ 灌库。
func (s *service) runRestore(baseCtx context.Context, taskID, uploadPath, operator string) error {
	// 维护模式状态机（A8 契约 dbbackup.IsRestoring 的写入端）：
	// 进入恢复即置 true，A8 的全局中间件据此拦截并发写入；用 defer 复位，
	// 覆盖 success / failed / panic 展开的全部退出路径（panic 会继续向上传播到
	// runTask 的 recover，defer 仍先执行复位，不会出现「卡在维护模式」）。
	// 这是 restoring 唯一的生产写入点，导出任务与其它模块不得改动。
	restoring.Store(true)
	defer restoring.Store(false)

	// ① 快速校验（不做昂贵动作前先拒绝非法上传）
	s.updateProgress(taskID, progressVerify, stageVerify)
	if _, err := verifyContainerOnly(uploadPath); err != nil {
		return fmt.Errorf("校验上传备份容器: %w", err)
	}

	// ② 恢复前自动全量备份：失败即中止，无回退退路不下发
	s.updateProgress(taskID, progressPreBak, stagePreRestore)
	preCtx, cancelPre := stageCtx(stagePreBackupTimeout)
	defer cancelPre()
	if err := s.preRestoreBackup(preCtx, operator); err != nil {
		return fmt.Errorf("恢复前自动全量备份失败，已中止恢复: %w", err)
	}

	// ③ 流式解压并复算 dump_sha256 / container_sha256
	s.updateProgress(taskID, progressExtract, stageExtract)
	extractDir := filepath.Join(filepath.Dir(uploadPath), "dbbackup_extract_"+taskID)
	defer func() { _ = os.RemoveAll(extractDir) }()
	extractCtx, cancelExtract := stageCtx(stageManifestTimeout)
	defer cancelExtract()
	_, dumpFile, err := verifyAndExtract(extractCtx, uploadPath, extractDir)
	if err != nil {
		return fmt.Errorf("解压并校验备份: %w", err)
	}

	// ④ 执行恢复：走可注入的 DumpExecutor.Restore（契约抽象）。
	//    生产实现 productionDumper.Restore 委托给 restoreDump（PG pg_restore /
	//    SQLite 原子替换）；单测注入 fake 时不会触发真实命令、也不会关闭
	//    GORM 连接池，保证「不依赖真实 pg_dump/pg_restore」的测试约束。
	s.updateProgress(taskID, progressRestore, stageRestore)
	restoreCtx, cancelRestore := stageCtx(stageRestoreTimeout)
	defer cancelRestore()
	execCfg, err := s.execToolCfg(baseCtx, "pg_restore")
	if err != nil {
		return err
	}
	if err := s.exec.Restore(restoreCtx, dumpFile, execCfg); err != nil {
		return fmt.Errorf("执行恢复: %w", err)
	}
	// 恢复任务的 backup_id 保持 NULL（PG uuid 列不接受空串 ''），不回填。
	return nil
}

// preRestoreBackup 恢复前全量备份：复用 runExport 的核心逻辑，但走独立
// context 且不写任务表（它是恢复任务的一个阶段，不是独立任务）。
func (s *service) preRestoreBackup(ctx context.Context, operator string) error {
	// 枚举表
	tables, err := s.exec.TableNames(ctx, s.dbCfg)
	if err != nil {
		return fmt.Errorf("枚举物理表: %w", err)
	}
	now := time.Now()
	zipPath, remoteName, err := s.backupLayout(now)
	if err != nil {
		return err
	}
	dumpPath := zipPath + ".dump.tmp"
	defer func() { _ = os.Remove(dumpPath) }()
	produced, err := s.exec.Dump(ctx, dumpPath, s.dbCfg)
	if err != nil {
		return fmt.Errorf("转储数据库: %w", err)
	}
	if produced != "" {
		dumpPath = produced
	}
	meta := &manifest{
		SchemaVersion:  SchemaVersionV1,
		CreatedAt:      now.UTC(),
		DBType:         dbTypeOf(s.dbCfg),
		Tables:         tables,
		ExcludedTables: append([]string{}, excludedPhysicalTables...),
	}
	containerSha, err := writeContainer(ctx, zipPath, dumpPath, meta)
	if err != nil {
		_ = os.Remove(zipPath)
		return fmt.Errorf("生成备份容器: %w", err)
	}
	info, err := os.Stat(zipPath)
	if err != nil {
		return fmt.Errorf("读取备份容器信息: %w", err)
	}
	plan, err := loadPlan(ctx, s.db)
	if err != nil {
		return fmt.Errorf("读取备份计划: %w", err)
	}
	record := &models.DBBackup{
		FileName:      filepath.Base(zipPath),
		FilePath:      zipPath,
		StorageType:   string(plan.Storage),
		DBType:        dbTypeOf(s.dbCfg),
		FileSize:      info.Size(),
		Sha256:        containerSha,
		SchemaVersion: SchemaVersionV1,
		TableCount:    len(tables),
		Source:        SourcePreRestore,
		Status:        StatusReady,
		TriggeredBy:   operator,
		Remark:        "恢复前自动备份",
	}
	if err := s.db.WithContext(ctx).Create(record).Error; err != nil {
		_ = os.Remove(zipPath)
		return fmt.Errorf("保存恢复前备份记录: %w", err)
	}
	// 远程上传失败不阻断恢复前备份成功
	store, err := newStorage(plan)
	if err == nil {
		if remotePath, putErr := store.Put(ctx, zipPath, remoteName); putErr == nil {
			_ = s.db.WithContext(ctx).Model(&models.DBBackup{}).
				Where("id = ?", record.ID).Update("remote_path", remotePath).Error
		} else {
			_ = s.db.WithContext(ctx).Model(&models.DBBackup{}).
				Where("id = ?", record.ID).
				Updates(map[string]interface{}{
					"status": StatusRemoteOnly,
					"remark": "远程上传失败: " + putErr.Error(),
				}).Error
		}
	}
	return nil
}

// enforceRetention 按 Plan.Retention 删除最旧备份：删记录 + 删本地文件 + 删远程文件。
// 只清理未标记 deleted 的记录；删除失败仅告警不阻断导出任务。
func (s *service) enforceRetention(ctx context.Context, plan *Plan) error {
	if plan == nil || plan.Retention <= 0 {
		return nil
	}
	var all []models.DBBackup
	if err := s.db.WithContext(ctx).Model(&models.DBBackup{}).
		Where("status <> ?", StatusDeleted).
		Order("created_at DESC").Find(&all).Error; err != nil {
		return fmt.Errorf("查询备份记录: %w", err)
	}
	if len(all) <= plan.Retention {
		return nil
	}
	// 按时间倒序稳定排序（DB 排序已给出，二次排序防同时间乱序）
	sort.SliceStable(all, func(i, j int) bool { return all[i].CreatedAt.After(all[j].CreatedAt) })
	toDelete := all[plan.Retention:]

	store, storeErr := newStorage(plan)
	for _, b := range toDelete {
		// 本地文件
		if b.FilePath != "" {
			if err := os.Remove(b.FilePath); err != nil && !os.IsNotExist(err) {
				applogger.Warnf("[全库备份] 删除本地备份文件失败 (id=%s): %v", b.ID, err)
			}
		}
		// 远程文件
		if b.RemotePath != "" && storeErr == nil {
			if err := store.Delete(ctx, b.RemotePath); err != nil {
				applogger.Warnf("[全库备份] 删除远程备份文件失败 (id=%s): %v", b.ID, err)
			}
		}
		// 记录：标记 deleted 后软删除（DBBackup 含 BaseModel.DeletedAt）
		if err := s.db.WithContext(ctx).Model(&models.DBBackup{}).
			Where("id = ?", b.ID).
			Updates(map[string]interface{}{"status": StatusDeleted}).Error; err != nil {
			applogger.Warnf("[全库备份] 标记备份记录 deleted 失败 (id=%s): %v", b.ID, err)
			continue
		}
		if err := s.db.WithContext(ctx).Delete(&models.DBBackup{}, "id = ?", b.ID).Error; err != nil {
			applogger.Warnf("[全库备份] 软删除备份记录失败 (id=%s): %v", b.ID, err)
		}
	}
	return nil
}

// Delete 删除单条备份记录：删记录 + 本地文件 + 远程文件。
func (s *service) Delete(ctx context.Context, id, operator string) error {
	var b models.DBBackup
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&b).Error; err != nil {
		return fmt.Errorf("备份记录不存在: %w", err)
	}
	if b.FilePath != "" {
		if err := os.Remove(b.FilePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("删除本地备份文件: %w", err)
		}
	}
	if b.RemotePath != "" {
		plan, err := loadPlan(ctx, s.db)
		if err == nil {
			store, sErr := newStorage(plan)
			if sErr == nil {
				if err := store.Delete(ctx, b.RemotePath); err != nil {
					applogger.Warnf("[全库备份] 删除远程备份文件失败 (id=%s): %v", id, err)
				}
			}
		}
	}
	_ = s.db.WithContext(ctx).Model(&models.DBBackup{}).
		Where("id = ?", id).Update("status", StatusDeleted).Error
	if err := s.db.WithContext(ctx).Delete(&models.DBBackup{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("删除备份记录: %w", err)
	}
	_ = operator // operlog 由 handler 层承担
	return nil
}

// DownloadPath 校验 backup 记录存在且文件存在，返回绝对路径。
func (s *service) DownloadPath(ctx context.Context, id string) (string, error) {
	var b models.DBBackup
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&b).Error; err != nil {
		return "", fmt.Errorf("备份记录不存在: %w", err)
	}
	if b.Status == StatusDeleted {
		return "", fmt.Errorf("备份记录已删除")
	}
	if b.FilePath == "" {
		return "", fmt.Errorf("备份记录无本地文件路径")
	}
	abs, err := filepath.Abs(b.FilePath)
	if err != nil {
		return "", fmt.Errorf("解析备份文件绝对路径: %w", err)
	}
	if info, err := os.Stat(abs); err != nil || info.IsDir() {
		if err != nil {
			return "", fmt.Errorf("备份文件不存在或不可访问: %w", err)
		}
		return "", fmt.Errorf("备份路径是目录而非文件")
	}
	return abs, nil
}

// RunScheduled 供 cron tick 调用（调度层固定 `0 * * * * *` 每分钟一次），
// 内部读 sys_config 门控决定是否真正执行。改配置无需重启。
//
// 门控：enabled=false 直接返回 nil；用「上次执行时间 + cron 解析的下一触发点」
// 判断是否到点，未到点直接返回 nil；到点才触发一次 source=cron 的导出并持久化
// 本次执行时间。周期精度有 ≤1 分钟抖动（tick 粒度），Phase 58 方案 A 既有取舍。
func (s *service) RunScheduled(ctx context.Context) error {
	plan, err := loadPlan(ctx, s.db)
	if err != nil {
		return fmt.Errorf("读取定时备份配置: %w", err)
	}
	if !plan.Enabled {
		return nil
	}
	due, err := shouldRunScheduled(ctx, s.db, time.Now(), plan.Cron)
	if err != nil {
		return fmt.Errorf("计算定时备份触发点: %w", err)
	}
	if !due {
		return nil
	}
	// 到点：持久化本次执行时间（先写再触发，防止触发失败后下一 tick 又立刻重试
	// 造成风暴）。写入失败不阻断，因为 sys_db_backup 的 source=cron 记录会兜底。
	if err := persistLastRunAt(ctx, s.db, time.Now()); err != nil {
		applogger.Warnf("[全库备份] 持久化定时执行时间失败: %v", err)
	}
	if _, err := s.TriggerExport(ctx, SourceCron, "scheduler"); err != nil {
		return fmt.Errorf("触发定时备份: %w", err)
	}
	return nil
}

// RecoverStaleTasks 启动时把残留 running/pending 任务收敛为 failed。
// 语义与 config_restore_task_service.RecoverStaleRunningTasks 一致：
// pending 无 goroutine 认领（真孤儿），running 加 2×stageDumpTimeout 宽限期。
func (s *service) RecoverStaleTasks(ctx context.Context) error {
	grace := time.Now().Add(-2 * stageDumpTimeout)
	res := s.db.WithContext(ctx).Model(&models.DBBackupTask{}).
		Where("(status = ? AND updated_at < ?) OR status = ?",
			models.DBBackupTaskRunning, grace, models.DBBackupTaskPending).
		Updates(map[string]interface{}{
			"status":        models.DBBackupTaskFailed,
			"error_message": "服务重启，任务中断",
			"completed_at":  time.Now(),
		})
	if res.Error != nil {
		return fmt.Errorf("收敛残留备份任务: %w", res.Error)
	}
	return nil
}

// dbTypeOf 归一数据库类型字符串（写进 manifest / 备份记录）
func dbTypeOf(cfg DatabaseConfig) string {
	if isSQLiteCfg(cfg) {
		return dbTypeSQLite
	}
	return dbTypePostgres
}

// 编译期确认：service 必须完整实现 Service 接口
var _ Service = (*service)(nil)

// 错误包别名（统一用 fmt.Errorf("context: %w", err) 包装，本别名仅备用）
var _ = errors.Is
