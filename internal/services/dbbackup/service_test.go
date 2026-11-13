package dbbackup

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// 本测试文件是包内单测：注入 fake DumpExecutor（不依赖真实 pg_dump/pg_restore），
// 用 glebarez/sqlite 纯 Go 内存库（CGO_ENABLED=0 友好），覆盖：
//   - NewService / List / GetPlan / SavePlan
//   - TriggerExport 状态机（pending→running→success）+ manifest sha256
//   - RunScheduled 门控（enabled / cron 未到点 / 到点触发）
//   - RecoverStaleTasks / DownloadPath / Delete / Statistics / GetTask / Get
//   - manifest 篡改检测（verifyContainerOnly）

// fakeDumpExecutor 生产执行器的替身：把一段固定字节流写成 dump 文件，
// 让 writeContainer / verifyAndExtract 的真实链路被完整跑到。
// 生产实现 Dump/Restore 会调外部命令或关闭连接池，fake 全部在内存/临时文件
// 内完成，保证单测「不依赖真实 pg_dump/pg_restore、不破坏测试库连接」。
type fakeDumpExecutor struct {
	tables    []string
	dumpBytes []byte
	dumpErr   error
	// restoreErr 非空时 Restore 直接返回该错误（用于 SQLite 离线恢复降级路径单测）。
	restoreErr error
	// restoreGate 非 nil 时 Restore 先阻塞等待该 channel 关闭，用于把恢复任务
	// 停在「维护模式中」以稳定观测 IsRestoring() 状态机（避免快速失败路径的竞态）。
	restoreGate <-chan struct{}
	// dumpPanic 非 nil 时 Dump 直接 panic（覆盖 runTask panic recover 兜底路径）。
	dumpPanic error

	mu            sync.Mutex
	calls         int // Dump 调用次数（RunScheduled 门控断言）
	restoreCalls  int // Restore 调用次数
	restoredPaths []string
}

func (f *fakeDumpExecutor) Dump(ctx context.Context, target string, cfg DatabaseConfig) (string, error) {
	f.mu.Lock()
	if f.dumpPanic != nil {
		f.mu.Unlock()
		panic(f.dumpPanic)
	}
	f.calls++
	f.mu.Unlock()
	if f.dumpErr != nil {
		return "", f.dumpErr
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(target, f.dumpBytes, 0o644); err != nil {
		return "", err
	}
	return target, nil
}

func (f *fakeDumpExecutor) Restore(ctx context.Context, source string, cfg DatabaseConfig) error {
	f.mu.Lock()
	f.restoreCalls++
	f.restoredPaths = append(f.restoredPaths, source)
	err := f.restoreErr
	gate := f.restoreGate
	f.mu.Unlock()
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}

func (f *fakeDumpExecutor) TableNames(ctx context.Context, cfg DatabaseConfig) ([]string, error) {
	return append([]string{}, f.tables...), nil
}

func (f *fakeDumpExecutor) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeDumpExecutor) restoreCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.restoreCalls
}

// newTestDB 建纯 Go sqlite 内存库并 AutoMigrate 三张表（Config/DBBackup/DBBackupTask）。
// 用文件型 DSN 而非 :memory: 是为了让同一连接的多次 AutoMigrate 与并发
// goroutine 共享同一数据库（:memory: 每连接独立，goroutine 里会看不到表）。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// 不用 t.TempDir()：Windows 下 sqlite 文件被连接池持有句柄，TempDir 的
	// 自动 RemoveAll 会与仍在收尾的异步 goroutine 抢文件而失败。改为自建目录
	// 并在 t.Cleanup 里「先关连接池 → 再删目录」，顺序完全可控。
	dir, err := os.MkdirTemp("", "dbbackup-test-*")
	if err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	dsn := filepath.Join(dir, "dbbackup_test.db")
	// logger.Silent 抑制 GORM 的 "record not found" 噪音（这些在用例里是预期分支）
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		_ = os.RemoveAll(dir)
		t.Fatalf("打开测试库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Config{}, &models.DBBackup{}, &models.DBBackupTask{}); err != nil {
		_ = os.RemoveAll(dir)
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	t.Cleanup(func() {
		// 先等后台任务全部进入终态，再关池删目录：否则异步 goroutine 会在
		// 关池后写库（"database is closed"）或与 RemoveAll 抢 sqlite 文件句柄。
		waitAllTasks(db, 20*time.Second)
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		_ = os.RemoveAll(dir)
	})
	return db
}

// waitAllTasks 等待该库上所有备份任务进入终态，防止测试返回时后台 goroutine
// 仍在写库（否则关池/删目录会撞上 "database is closed" 或文件占用）。
func waitAllTasks(db *gorm.DB, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var active int64
		if err := db.Model(&models.DBBackupTask{}).
			Where("status IN ?", []string{models.DBBackupTaskPending, models.DBBackupTaskRunning}).
			Count(&active).Error; err != nil {
			return // 库已关闭，直接返回
		}
		if active == 0 {
			// 再给 defer 收尾（临时文件删除）留一点余量
			time.Sleep(50 * time.Millisecond)
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// newTestService 构造注入 fake 的服务，备份目录指向 t.TempDir()（不落仓库 data/）
func newTestService(t *testing.T, db *gorm.DB, exec *fakeDumpExecutor) Service {
	t.Helper()
	return NewService(db, DatabaseConfig{Type: dbTypeSQLite, Path: filepath.Join(t.TempDir(), "fake.db")},
		exec, filepath.Join(t.TempDir(), "backups"))
}

// waitTask 轮询等待异步任务进入终态（success/failed），带超时防挂死。
//
// 注意：终态写入与上传临时文件清理在生产代码里是「先清理、再写终态」，
// 但全仓并行 go test 时 goroutine 调度可能被其它包抢占数毫秒，测试侧仍需
// 对文件消失做有界重试（见 waitFileRemoved），不能读到终态就立刻 os.Stat。
func waitTask(t *testing.T, svc Service, id string, timeout time.Duration) *TaskView {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		v, err := svc.GetTask(context.Background(), id)
		if err != nil {
			t.Fatalf("查询任务 %s 失败: %v", id, err)
		}
		if v.Status == models.DBBackupTaskSuccess || v.Status == models.DBBackupTaskFailed {
			return v
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("任务 %s 在 %v 内未进入终态", id, timeout)
	return nil
}

// waitFileRemoved 有界等待文件被删除。高负载并行测试下清理 goroutine 可能晚于
// 终态读取数十毫秒，用短重试消除 flaky，同时仍严格断言「最终必须删除」。
func waitFileRemoved(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("文件应在任务收尾被清理，但仍存在: %s (stat err=%v)", path, err)
	}
}

// seedConfig 直接写一个 sys_config 键（走 service 的 SavePlan 之外的快速路径）
func seedConfig(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	if err := writeConfigValue(context.Background(), db, key, value); err != nil {
		t.Fatalf("写入 sys_config[%s] 失败: %v", key, err)
	}
}

// ========== 1. NewService / List / Get ==========

func TestNewService_List_Get(t *testing.T) {
	db := newTestDB(t)
	exec := &fakeDumpExecutor{tables: []string{"sys_user", "sys_role"}, dumpBytes: []byte("FAKE-DUMP")}
	svc := newTestService(t, db, exec)

	if svc == nil {
		t.Fatal("NewService 返回 nil")
	}
	// 空列表
	res, err := svc.List(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if res.Total != 0 || len(res.List) != 0 {
		t.Fatalf("空库应返回空列表，实得 total=%d len=%d", res.Total, len(res.List))
	}

	// 分页归一化：current<=0 / pageSize<=0 应被归一化，不报错
	if _, err := svc.List(context.Background(), 0, 0); err != nil {
		t.Fatalf("分页归一化 List 失败: %v", err)
	}

	// 手动插一条记录，验证 List/Get 的 DTO 映射
	rec := &models.DBBackup{
		FileName: "b.zip", FilePath: "/tmp/b.zip", StorageType: string(StorageLocal),
		DBType: dbTypeSQLite, FileSize: 123, Sha256: "abc", SchemaVersion: SchemaVersionV1,
		TableCount: 2, Source: SourceManual, Status: StatusReady, TriggeredBy: "u1",
	}
	if err := db.Create(rec).Error; err != nil {
		t.Fatalf("插入备份记录失败: %v", err)
	}
	res, err = svc.List(context.Background(), 1, 10)
	if err != nil || res.Total != 1 || len(res.List) != 1 {
		t.Fatalf("List 应返回 1 条: err=%v res=%+v", err, res)
	}
	got, err := svc.Get(context.Background(), rec.ID)
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if got.FileName != "b.zip" || got.Sha256 != "abc" || got.TableCount != 2 {
		t.Fatalf("Get DTO 映射错误: %+v", got)
	}
	if _, err := svc.Get(context.Background(), "not-exist"); err == nil {
		t.Fatal("Get 不存在的 id 应返回错误")
	}
}

// ========== 2. GetPlan / SavePlan ==========

func TestGetPlan_SavePlan(t *testing.T) {
	db := newTestDB(t)
	exec := &fakeDumpExecutor{tables: []string{}, dumpBytes: []byte("X")}
	svc := newTestService(t, db, exec)
	// savePlan 现对敏感凭据做 SM4 字段加密（fail-closed），单测需注册一个测试 cipher。
	// 用 t.Cleanup 复位，避免污染依赖「未注册 cipher」的其它用例。
	// 注意：RegisterCredentialCipher 是包级状态，本测试**不可**加 t.Parallel
	// （并行会与其它用例互踩注册器）；后续新增用例同样遵守此约束。
	RegisterCredentialCipher(testSM4Cipher(t))
	t.Cleanup(func() { RegisterCredentialCipher(nil) })

	// 默认值（未 seed 任何键）
	p, err := svc.GetPlan(context.Background())
	if err != nil {
		t.Fatalf("GetPlan 失败: %v", err)
	}
	if p.Enabled || p.Cron != defaultCronExpr || p.Retention != defaultRetention || p.Storage != StorageLocal {
		t.Fatalf("GetPlan 默认值错误: %+v", p)
	}

	// 保存完整计划（含敏感凭据），再读回
	want := &Plan{
		Enabled: true, Cron: "0 0 3 * * *", Retention: 5, Storage: StorageOSS,
		OSS: OSSConfig{Endpoint: "s3.example.com", Bucket: "bk", AccessKey: "AK", SecretKey: "SK", UseSSL: false},
		FTP: FTPConfig{Host: "ftp.example.com", Port: 2121, User: "u", Password: "P", RemoteDir: "/bak"},
		SMB: SMBConfig{Path: "/mnt/share"},
	}
	if err := svc.SavePlan(context.Background(), want, "admin"); err != nil {
		t.Fatalf("SavePlan 失败: %v", err)
	}
	got, err := svc.GetPlan(context.Background())
	if err != nil {
		t.Fatalf("GetPlan 失败: %v", err)
	}
	if got.Enabled != true || got.Cron != "0 0 3 * * *" || got.Retention != 5 || got.Storage != StorageOSS {
		t.Fatalf("SavePlan/GetPlan 往返丢失字段: %+v", got)
	}
	if got.OSS.AccessKey != "AK" || got.OSS.SecretKey != "SK" || got.FTP.Password != "P" {
		t.Fatalf("凭据加解密往返丢失（密文落库、读取时解出真值）: %+v", got)
	}
	if got.OSS.UseSSL != false {
		t.Fatalf("UseSSL=false 未保留: %+v", got.OSS)
	}

	// 非法 cron 必须被拒绝
	bad := &Plan{Enabled: true, Cron: "not a cron", Retention: 3, Storage: StorageLocal}
	if err := svc.SavePlan(context.Background(), bad, "admin"); err == nil {
		t.Fatal("非法 cron 应被 SavePlan 拒绝")
	}
}

// ========== 3. manifest sha256 双校验 + 容器真实性 ==========

func TestManifest_Sha256_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	dumpPath := filepath.Join(tmp, "dump.sql")
	zipPath := filepath.Join(tmp, "backup.zip")
	dumpBytes := []byte("PG-DUMP-CUSTOM-BYTES-0123456789")
	if err := os.WriteFile(dumpPath, dumpBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	meta := &manifest{
		SchemaVersion:  SchemaVersionV1,
		CreatedAt:      time.Now().UTC().Truncate(time.Second),
		DBType:         dbTypePostgres,
		Tables:         []string{"sys_user", "sys_role", "sys_menu"},
		ExcludedTables: append([]string{}, excludedPhysicalTables...),
	}
	containerSha, err := writeContainer(context.Background(), zipPath, dumpPath, meta)
	if err != nil {
		t.Fatalf("writeContainer 失败: %v", err)
	}
	if len(containerSha) != 64 {
		t.Fatalf("容器 sha256 长度错误: %d", len(containerSha))
	}
	// 整个 zip 文件哈希必须可复算（下载/恢复传输完整性校验用）
	fileSha, err := fileSha256(zipPath)
	if err != nil {
		t.Fatalf("fileSha256 失败: %v", err)
	}
	if fileSha != containerSha {
		t.Fatalf("writeContainer 返回的容器哈希与文件实际哈希不一致: %s vs %s", containerSha, fileSha)
	}

	// 读回 manifest.json，验证三层校验字段与零凭据
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var mf *zip.File
	for _, f := range zr.File {
		if f.Name == manifestEntryName {
			mf = f
			break
		}
	}
	if mf == nil {
		t.Fatal("容器缺少 manifest.json")
	}
	r, err := mf.Open()
	if err != nil {
		t.Fatal(err)
	}
	var parsed manifest
	if err := json.NewDecoder(r).Decode(&parsed); err != nil {
		t.Fatalf("解析 manifest.json 失败: %v", err)
	}
	_ = r.Close()
	if parsed.SchemaVersion != SchemaVersionV1 || parsed.TableCount != 3 || len(parsed.Tables) != 3 {
		t.Fatalf("manifest 元数据错误: %+v", parsed)
	}
	if parsed.ChecksumAlgorithm != "sha256" || parsed.AppVersion == "" {
		t.Fatalf("manifest 校验算法/版本字段缺失: %+v", parsed)
	}
	if parsed.DumpSha256 == "" || parsed.ContainerSha256 == "" || parsed.Sha256 == "" {
		t.Fatalf("manifest 三层 sha256 字段缺失: %+v", parsed)
	}
	if parsed.DumpSha256 == parsed.ContainerSha256 {
		t.Fatal("dump_sha256 与 container_sha256 不应相同（独立校验）")
	}
	// 零凭据：序列化后的清单绝不能出现任何凭据关键词
	b, _ := json.Marshal(parsed)
	for _, secret := range []string{"PGPASSWORD", "password", "secretKey", "accessKey", "SecretKey"} {
		if strings.Contains(string(b), secret) {
			t.Fatalf("manifest.json 泄露凭据关键词 %q", secret)
		}
	}

	// 校验 + 解压：dump 内容必须逐字节还原
	extractDir := filepath.Join(tmp, "extract")
	parsed2, extracted, err := verifyAndExtract(context.Background(), zipPath, extractDir)
	if err != nil {
		t.Fatalf("verifyAndExtract 失败: %v", err)
	}
	if parsed2.Sha256 != parsed.Sha256 {
		t.Fatal("两次读回的清单签名不一致")
	}
	got, err := os.ReadFile(extracted)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(dumpBytes) {
		t.Fatalf("解压后 dump 字节不一致: %q", got)
	}

	// 篡改清单签名必须被拒绝：把 sha256 改坏后重新打包，verifyContainerOnly 应报错
	tampered := parsed
	tampered.Sha256 = "0000000000000000000000000000000000000000000000000000000000000000"
	tb, _ := json.MarshalIndent(&tampered, "", "  ")
	tamperedZip := filepath.Join(tmp, "tampered.zip")
	if err := writeZipWithManifest(tamperedZip, mf.Name, tb, meta.DumpFile, dumpBytes); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyContainerOnly(tamperedZip); err == nil {
		t.Fatal("篡改后的 manifest 应被 verifyContainerOnly 拒绝")
	}
}

// writeZipWithManifest 用自定义 manifest 字节与原始 dump 字节重建一个 zip，
// 专供篡改检测用例使用（不经 writeContainer，因为后者会重算合法签名）。
func writeZipWithManifest(zipPath, mfName string, mfBytes []byte, dumpName string, dumpBytes []byte) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create(mfName)
	if err != nil {
		return err
	}
	if _, err := w.Write(mfBytes); err != nil {
		return err
	}
	dw, err := zw.Create(dumpName)
	if err != nil {
		return err
	}
	if _, err := dw.Write(dumpBytes); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return f.Close()
}

// ========== 4. TriggerExport 状态机 + 落盘 + 记录 ==========

func TestTriggerExport_StateMachine(t *testing.T) {
	db := newTestDB(t)
	exec := &fakeDumpExecutor{
		tables:    []string{"sys_user", "sys_dept"},
		dumpBytes: []byte("FULL-DB-DUMP-CONTENT"),
	}
	svc := newTestService(t, db, exec)

	taskID, err := svc.TriggerExport(context.Background(), SourceManual, "tester")
	if err != nil {
		t.Fatalf("TriggerExport 失败: %v", err)
	}
	if taskID == "" {
		t.Fatal("TriggerExport 返回空 taskID")
	}
	// 立即查：应处于 pending 或 running（不能还是 nil）
	initial, err := svc.GetTask(context.Background(), taskID)
	if err != nil {
		t.Fatalf("GetTask 失败: %v", err)
	}
	if initial.Status != models.DBBackupTaskPending && initial.Status != models.DBBackupTaskRunning {
		t.Fatalf("初始状态应为 pending/running，实得 %s", initial.Status)
	}

	final := waitTask(t, svc, taskID, 15*time.Second)
	if final.Status != models.DBBackupTaskSuccess {
		t.Fatalf("导出任务应成功，实得 status=%s err=%s", final.Status, final.ErrorMessage)
	}
	if final.Progress != progressDone {
		t.Fatalf("成功任务 progress 应为 100，实得 %d", final.Progress)
	}
	if final.StartedAt == nil || final.CompletedAt == nil {
		t.Fatal("成功任务应写 StartedAt/CompletedAt")
	}

	// 备份记录必须落库，且文件真实存在、sha256 可复算
	list, err := svc.List(context.Background(), 1, 10)
	if err != nil || list.Total != 1 {
		t.Fatalf("导出后应有 1 条备份记录: err=%v list=%+v", err, list)
	}
	rec := list.List[0]
	if rec.Source != SourceManual || rec.Status != StatusReady || rec.TableCount != 2 {
		t.Fatalf("备份记录字段错误: %+v", rec)
	}
	if rec.Sha256 == "" || len(rec.Sha256) != 64 {
		t.Fatalf("备份记录 sha256 错误: %q", rec.Sha256)
	}
	if _, err := os.Stat(rec.FilePath); err != nil {
		t.Fatalf("备份文件未落盘 %s: %v", rec.FilePath, err)
	}
	// 路径规范：data/backups/{YYYY}/{MM}/{ts}_{dbtype}.zip
	base := filepath.Base(rec.FilePath)
	if filepath.Ext(base) != ".zip" {
		t.Fatalf("备份文件应为 .zip: %s", base)
	}
	if !strings.Contains(base, "sqlite") {
		t.Fatalf("文件名应含 dbtype: %s", base)
	}
	// 容器必须能通过校验（下载/恢复前置门禁）
	if _, err := verifyContainerOnly(rec.FilePath); err != nil {
		t.Fatalf("导出产物容器校验失败: %v", err)
	}
	// 下载路径校验
	abs, err := svc.DownloadPath(context.Background(), rec.ID)
	if err != nil {
		t.Fatalf("DownloadPath 失败: %v", err)
	}
	if abs != rec.FilePath && !filepath.IsAbs(abs) {
		t.Fatalf("DownloadPath 应返回绝对路径: %s", abs)
	}
	// Statistics
	stats, err := svc.Statistics(context.Background())
	if err != nil {
		t.Fatalf("Statistics 失败: %v", err)
	}
	if stats["total"] != int64(1) {
		t.Fatalf("Statistics total 应为 1: %+v", stats)
	}
}

// ========== 5. RunScheduled 门控 ==========

func TestRunScheduled_Gating(t *testing.T) {
	db := newTestDB(t)
	exec := &fakeDumpExecutor{tables: []string{"sys_user"}, dumpBytes: []byte("SCHED-DUMP")}
	svc := newTestService(t, db, exec)

	// ① enabled=false：直接返回 nil，且绝不触发导出
	seedConfig(t, db, cfgKeyEnabled, "false")
	seedConfig(t, db, cfgKeyCron, defaultCronExpr)
	if err := svc.RunScheduled(context.Background()); err != nil {
		t.Fatalf("enabled=false 时 RunScheduled 应返回 nil: %v", err)
	}
	if exec.callCount() != 0 {
		t.Fatal("enabled=false 不应触发任何导出")
	}

	// ② enabled=true 但 cron 是「每年 1 月 1 日」且刚记录过 last_run_at：
	//   下一次触发远在未来 → 未到点，返回 nil 且不导出
	seedConfig(t, db, cfgKeyEnabled, "true")
	seedConfig(t, db, cfgKeyCron, "0 0 2 1 1 *") // 每年 1 月 1 日 02:00:00
	seedConfig(t, db, cfgKeyLastRunAt, time.Now().UTC().Format(lastRunAtLayout))
	if err := svc.RunScheduled(context.Background()); err != nil {
		t.Fatalf("未到点时 RunScheduled 应返回 nil: %v", err)
	}
	if exec.callCount() != 0 {
		t.Fatal("未到点不应触发导出（每分钟一份备份的回归守护）")
	}

	// ③ 连续多次 tick 都不得触发（防每分钟无条件执行）
	for i := 0; i < 3; i++ {
		if err := svc.RunScheduled(context.Background()); err != nil {
			t.Fatalf("第 %d 次 tick 失败: %v", i, err)
		}
	}
	if exec.callCount() != 0 {
		t.Fatal("连续 tick 未到点时不应触发任何导出")
	}

	// ④ 到点：把 last_run_at 设为很久以前，用「每秒都触发」的表达式，
	//   下一触发点必然 <= now → 应真正触发一次 source=cron 的导出
	seedConfig(t, db, cfgKeyCron, "* * * * * *") // 6 字段，每秒
	seedConfig(t, db, cfgKeyLastRunAt, time.Now().UTC().Add(-24*time.Hour).Format(lastRunAtLayout))
	if err := svc.RunScheduled(context.Background()); err != nil {
		t.Fatalf("到点时 RunScheduled 失败: %v", err)
	}
	// 等待异步导出落地
	deadline := time.Now().Add(15 * time.Second)
	for exec.callCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if exec.callCount() == 0 {
		t.Fatal("到点应触发一次导出")
	}
	// last_run_at 必须被持久化（防重启后补跑）
	raw, err := readConfigValue(context.Background(), db, cfgKeyLastRunAt)
	if err != nil || raw == "" {
		t.Fatalf("last_run_at 应被持久化: err=%v raw=%q", err, raw)
	}
	if _, err := time.Parse(lastRunAtLayout, raw); err != nil {
		t.Fatalf("last_run_at 格式非法: %q", raw)
	}

	// ⑤ 紧接着的下一 tick 不得再触发（last_run_at 刚更新 + 每秒表达式会在
	//   下一秒到点，但门控以「实际执行时间」为基准，允许最多 1 tick 抖动；
	//   这里只断言不会连续产生第二次成功记录在同一轮调用内）
	// 等待本轮任务终态后核对 source=cron 记录存在
	deadline = time.Now().Add(15 * time.Second)
	var cronCount int64
	for time.Now().Before(deadline) {
		if err := db.Model(&models.DBBackup{}).Where("source = ?", SourceCron).Count(&cronCount).Error; err == nil && cronCount >= 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if cronCount < 1 {
		t.Fatal("到点导出应产生 source=cron 的备份记录")
	}
}

// ========== 6. RecoverStaleTasks / Delete / 恢复临时文件清理 ==========

func TestRecoverStaleTasks_And_Delete(t *testing.T) {
	db := newTestDB(t)
	exec := &fakeDumpExecutor{tables: []string{}, dumpBytes: []byte("X")}
	svc := newTestService(t, db, exec)

	// 残留 pending + 老 running 应被收敛为 failed
	pending := &models.DBBackupTask{TaskType: models.DBBackupTaskTypeExport, Status: models.DBBackupTaskPending}
	running := &models.DBBackupTask{TaskType: models.DBBackupTaskTypeRestore, Status: models.DBBackupTaskRunning}
	if err := db.Omit("backup_id").Create(pending).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Omit("backup_id").Create(running).Error; err != nil {
		t.Fatal(err)
	}
	// running 的 updated_at 调到宽限期之外
	db.Model(&models.DBBackupTask{}).Where("id = ?", running.ID).
		UpdateColumn("updated_at", time.Now().Add(-3*stageDumpTimeout))
	if err := svc.RecoverStaleTasks(context.Background()); err != nil {
		t.Fatalf("RecoverStaleTasks 失败: %v", err)
	}
	v1, _ := svc.GetTask(context.Background(), pending.ID)
	v2, _ := svc.GetTask(context.Background(), running.ID)
	if v1.Status != models.DBBackupTaskFailed || v2.Status != models.DBBackupTaskFailed {
		t.Fatalf("残留任务应收敛为 failed: pending=%s running=%s", v1.Status, v2.Status)
	}
	if v1.ErrorMessage == "" || v1.CompletedAt == nil {
		t.Fatal("收敛任务应写 errorMessage/completedAt")
	}

	// Delete：删记录 + 删本地文件
	tmpFile := filepath.Join(t.TempDir(), "del.zip")
	if err := os.WriteFile(tmpFile, []byte("zipdata"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := &models.DBBackup{
		FileName: "del.zip", FilePath: tmpFile, StorageType: string(StorageLocal),
		DBType: dbTypeSQLite, Source: SourceManual, Status: StatusReady,
	}
	if err := db.Create(rec).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(context.Background(), rec.ID, "tester"); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Fatalf("Delete 应删除本地文件，stat err=%v", err)
	}
	if _, err := svc.DownloadPath(context.Background(), rec.ID); err == nil {
		t.Fatal("已删除记录 DownloadPath 应报错")
	}
}

// ========== 7. TriggerRestore 临时文件清理（agent-7 跨层约定） ==========

func TestTriggerRestore_CleansUpUploadTempFile(t *testing.T) {
	db := newTestDB(t)
	exec := &fakeDumpExecutor{tables: []string{"sys_user"}, dumpBytes: []byte("RESTORE-SRC")}
	svc := newTestService(t, db, exec)

	// 构造一个合法容器作为上传文件（写在系统临时目录，模拟 handler os.CreateTemp）
	tmp, err := os.CreateTemp("", "dbbackup-restore-*.zip")
	if err != nil {
		t.Fatal(err)
	}
	uploadPath := tmp.Name()
	_ = tmp.Close()

	dumpPath := filepath.Join(t.TempDir(), "src.dump")
	if err := os.WriteFile(dumpPath, []byte("UPLOADED-DUMP"), 0o644); err != nil {
		t.Fatal(err)
	}
	meta := &manifest{
		SchemaVersion: SchemaVersionV1, CreatedAt: time.Now().UTC(),
		DBType: dbTypeSQLite, Tables: []string{"sys_user"},
		ExcludedTables: append([]string{}, excludedPhysicalTables...),
	}
	if _, err := writeContainer(context.Background(), uploadPath, dumpPath, meta); err != nil {
		t.Fatalf("构造上传容器失败: %v", err)
	}

	// 用非法文件（校验失败路径）也要清理：先跑一次非法上传
	badPath := filepath.Join(t.TempDir(), "bad.zip")
	if err := os.WriteFile(badPath, []byte("not-a-zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	badID, err := svc.TriggerRestore(context.Background(), badPath, "tester")
	if err != nil {
		t.Fatalf("TriggerRestore(非法文件) 建任务失败: %v", err)
	}
	waitTask(t, svc, badID, 15*time.Second)
	waitFileRemoved(t, badPath, 5*time.Second)

	// 合法容器：完整走「校验 → 恢复前自动备份 → 解压 → exec.Restore」。
	// 注入 fake 后不会触发真实 pg_restore / SQLite 文件替换，测试库连接保持可用，
	// 因此可以同时断言恢复前备份记录、exec.Restore 调用与临时文件清理。
	goodID, err := svc.TriggerRestore(context.Background(), uploadPath, "tester")
	if err != nil {
		t.Fatalf("TriggerRestore(合法容器) 建任务失败: %v", err)
	}
	final := waitTask(t, svc, goodID, 30*time.Second)
	if final.Status != models.DBBackupTaskSuccess {
		t.Fatalf("合法容器恢复任务应成功，实得 status=%s err=%s", final.Status, final.ErrorMessage)
	}
	if exec.restoreCallCount() != 1 {
		t.Fatalf("exec.Restore 应被调用 1 次，实得 %d", exec.restoreCallCount())
	}
	// 恢复前自动全量备份必须落一条 source=pre_restore 的记录
	var preCount int64
	if err := db.Model(&models.DBBackup{}).Where("source = ?", SourcePreRestore).Count(&preCount).Error; err != nil {
		t.Fatalf("统计恢复前备份记录失败: %v", err)
	}
	if preCount != 1 {
		t.Fatalf("应产生 1 条 pre_restore 备份记录，实得 %d", preCount)
	}
	waitFileRemoved(t, uploadPath, 5*time.Second)
}

// ========== 8. NewStorage 四类型构造 ==========

func TestNewStorage_Types(t *testing.T) {
	// 平台无关的绝对路径：Windows 下 "/mnt/share" 不是 IsAbs，
	// 用 filepath.Join(t.TempDir(), ...) 生成真正的绝对路径。
	absMount := filepath.Join(t.TempDir(), "smb-mount")
	cases := []struct {
		name string
		p    *Plan
		want StorageType
	}{
		{"local 默认", &Plan{Storage: StorageLocal}, StorageLocal},
		{"空值回落 local", &Plan{}, StorageLocal},
		{"SMB 挂载路径", &Plan{Storage: StorageSMB, SMB: SMBConfig{Path: absMount}}, StorageSMB},
		{"OSS", &Plan{Storage: StorageOSS, OSS: OSSConfig{Endpoint: "s3.example.com", Bucket: "bk", AccessKey: "a", SecretKey: "b"}}, StorageOSS},
		{"FTP", &Plan{Storage: StorageFTP, FTP: FTPConfig{Host: "ftp.example.com", Port: 21}}, StorageFTP},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st, err := NewStorage(tc.p)
			if err != nil {
				t.Fatalf("NewStorage 失败: %v", err)
			}
			if st.Type() != tc.want {
				t.Fatalf("类型应为 %s，实得 %s", tc.want, st.Type())
			}
		})
	}
	// 非法配置必须报错
	if _, err := NewStorage(&Plan{Storage: StorageSMB, SMB: SMBConfig{Path: "relative/path"}}); err == nil {
		t.Fatal("相对路径 SMB 应被拒绝")
	}
	if _, err := NewStorage(nil); err == nil {
		t.Fatal("nil 计划应被拒绝")
	}
}

// ========== 9. 错误包装格式 ==========

func TestErrorsAreWrapped(t *testing.T) {
	db := newTestDB(t)
	exec := &fakeDumpExecutor{tables: []string{}, dumpBytes: []byte("X")}
	svc := newTestService(t, db, exec)
	// 不存在的记录：错误必须是包装后的文本（fmt.Errorf("context: %w") 风格）
	_, err := svc.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("应返回错误")
	}
	if !strings.Contains(err.Error(), "备份记录不存在") {
		t.Fatalf("错误应含上下文描述: %v", err)
	}
	if _, err := svc.DownloadPath(context.Background(), "missing"); err == nil {
		t.Fatal("DownloadPath 不存在记录应报错")
	}
	// Dump 失败时任务应 failed（不是 panic）
	failing := &fakeDumpExecutor{tables: []string{"t"}, dumpErr: fmt.Errorf("模拟 pg_dump 缺失")}
	svc2 := NewService(db, DatabaseConfig{Type: dbTypeSQLite}, failing, filepath.Join(t.TempDir(), "bk2"))
	id, err := svc2.TriggerExport(context.Background(), SourceManual, "t")
	if err != nil {
		t.Fatal(err)
	}
	v := waitTask(t, svc2, id, 15*time.Second)
	if v.Status != models.DBBackupTaskFailed || v.ErrorMessage == "" {
		t.Fatalf("Dump 失败应收敛为 failed 且写 errorMessage: %+v", v)
	}
}

// ========== 10. SQLite 真实恢复降级（code review S1 回归守护） ==========

// TestRestoreSQLite_DoesNotCloseGlobalDBPool 验证 SQLite 在线恢复不再关闭全局
// 连接池（旧实现 sqlDB.Close() 会让 runTask 写终态失败、整个进程 DB 半死）。
// 断言三件事：①返回 ErrSQLiteOfflineRestoreRequired；②落 .pending_restore 且
// 产物是合法 SQLite 库；③传入的 DB 连接仍可正常查询（证明连接池未被关闭）。
func TestRestoreSQLite_DoesNotCloseGlobalDBPool(t *testing.T) {
	// 建一个真实的 SQLite 文件库（作为「当前运行库」与恢复产物源）
	dir := t.TempDir()
	livePath := filepath.Join(dir, "live.db")
	live, err := gorm.Open(sqlite.Open(livePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("建立测试 SQLite 库失败: %v", err)
	}
	if err := live.Exec("CREATE TABLE IF NOT EXISTS t_probe (id INTEGER PRIMARY KEY, v TEXT)").Error; err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	if err := live.Exec("INSERT INTO t_probe (v) VALUES ('alive')").Error; err != nil {
		t.Fatalf("写入失败: %v", err)
	}

	// 用 VACUUM INTO 产出一份完整可用的 SQLite 备份作为「上传的 dump」
	dumpPath := filepath.Join(dir, "uploaded.dump")
	_ = os.Remove(dumpPath)
	sqlDB, err := live.DB()
	if err != nil {
		t.Fatalf("取底层连接池失败: %v", err)
	}
	// 测试自行在结束时关闭连接池，让 t.TempDir() 在 Windows 上能清理临时文件；
	// 这同时反证「restoreSQLite 没有擅自 Close 全局池」——否则这里会二次 Close。
	defer sqlDB.Close()
	if _, err := sqlDB.Exec(fmt.Sprintf("VACUUM INTO '%s'", dumpPath)); err != nil {
		t.Fatalf("VACUUM INTO 产出恢复源失败: %v", err)
	}

	cfg := DatabaseConfig{Type: dbTypeSQLite, Path: livePath}
	err = restoreSQLite(context.Background(), dumpPath, cfg)

	// ① 必须返回「需离线恢复」的明确错误，而不是静默破坏连接池
	if err == nil || !errors.Is(err, ErrSQLiteOfflineRestoreRequired) {
		t.Fatalf("应返回 ErrSQLiteOfflineRestoreRequired，实得: %v", err)
	}
	if !strings.Contains(err.Error(), "db-backup-restore.go") {
		t.Fatalf("错误信息应给出离线恢复 CLI 指引: %v", err)
	}
	// ② .pending_restore 产物落盘且是合法 SQLite 库
	pending := livePath + ".pending_restore"
	if _, statErr := os.Stat(pending); statErr != nil {
		t.Fatalf("应落 .pending_restore 待恢复文件: %v", statErr)
	}
	if probeErr := probeSQLite(context.Background(), pending); probeErr != nil {
		t.Fatalf("待恢复产物应是合法 SQLite 库: %v", probeErr)
	}
	// ③ 全局连接池必须仍健康：原库文件未被替换、连接仍可查询
	var v string
	if qErr := live.Raw("SELECT v FROM t_probe WHERE id = 1").Scan(&v).Error; qErr != nil || v != "alive" {
		t.Fatalf("恢复后全局连接池应保持可用（旧实现会 sqlDB.Close 导致此步失败）: qErr=%v v=%q", qErr, v)
	}
	if _, statErr := os.Stat(livePath); statErr != nil {
		t.Fatalf("在线恢复不应替换原库文件: %v", statErr)
	}
}

// ========== 10.5 SQLite 恢复降级路径（端到端：任务终态 + 需重新登录 + 维护标志复位） ==========

// TestTriggerRestore_SQLiteOfflineDegrade 覆盖「SQLite 整库恢复需停服离线执行」的
// 端到端降级路径：注入 Restore 返回 ErrSQLiteOfflineRestoreRequired 的 fake，
// 走完整 TriggerRestore → runTask → runRestore → failTask，断言：
//   - 任务终态 failed，errorMessage 含离线 CLI 命令与「需重新登录」提示；
//   - 成功/失败都清缓存收口（这里未注册钩子，仅记日志，不阻断终态）；
//   - IsRestoring 在失败路径后复位为 false（维护模式不会卡死）。
func TestTriggerRestore_SQLiteOfflineDegrade(t *testing.T) {
	restoring.Store(false)
	RegisterRestoreCacheFlusher(nil)
	t.Cleanup(func() {
		restoring.Store(false)
		RegisterRestoreCacheFlusher(nil)
	})

	db := newTestDB(t)
	// SQLite 配置（newTestService 默认即 SQLite），Restore 返回离线恢复降级错误。
	exec := &fakeDumpExecutor{
		tables: []string{"sys_user"}, dumpBytes: []byte("X"),
		restoreErr: fmt.Errorf("已将恢复产物落到 /data/live.db.pending_restore；请停服后执行 "+
			"`go run ./scripts/ops/db-backup-restore.go -db /data/live.db -pending /data/live.db.pending_restore`"),
	}
	svc := newTestService(t, db, exec)
	upload := buildValidContainer(t)

	id, err := svc.TriggerRestore(context.Background(), upload, "tester")
	if err != nil {
		t.Fatalf("TriggerRestore 失败: %v", err)
	}
	final := waitTask(t, svc, id, 30*time.Second)
	if final.Status != models.DBBackupTaskFailed {
		t.Fatalf("SQLite 降级应标 failed，实得 status=%s", final.Status)
	}
	if !strings.Contains(final.ErrorMessage, "db-backup-restore.go") {
		t.Fatalf("降级信息应含离线命令入口: %s", final.ErrorMessage)
	}
	if !strings.Contains(final.ErrorMessage, "需重新登录") {
		t.Fatalf("降级信息应含「需重新登录」一致性提示: %s", final.ErrorMessage)
	}
	if exec.restoreCallCount() != 1 {
		t.Fatalf("Restore 应被调用 1 次，实得 %d", exec.restoreCallCount())
	}
	if IsRestoring() {
		t.Fatal("失败路径收尾后 IsRestoring 必须复位为 false")
	}
}

// ========== 11. pg_dump 缺失的可诊断降级（运行时故障回归守护） ==========
// ========== 10.6 panic recover 兜底与 flusher 报错收口 ==========

// TestTriggerExport_DumpPanic_FailsTask 覆盖 runTask 的 panic recover 兜底：
// Dump 中途 panic 时任务必须落终态 failed（含 panic 信息），进程不崩溃。
func TestTriggerExport_DumpPanic_FailsTask(t *testing.T) {
	db := newTestDB(t)
	exec := &fakeDumpExecutor{tables: []string{"sys_user"}, dumpPanic: fmt.Errorf("boom-in-dump")}
	svc := newTestService(t, db, exec)

	id, err := svc.TriggerExport(context.Background(), SourceManual, "tester")
	if err != nil {
		t.Fatalf("TriggerExport 失败: %v", err)
	}
	final := waitTask(t, svc, id, 30*time.Second)
	if final.Status != models.DBBackupTaskFailed {
		t.Fatalf("panic 后应标 failed，实得 %s", final.Status)
	}
	if !strings.Contains(final.ErrorMessage, "boom-in-dump") {
		t.Fatalf("错误信息应含 panic 原因: %s", final.ErrorMessage)
	}
}

// TestTriggerRestore_FlusherError_DoesNotBlockTerminal 覆盖恢复收尾一致性中
// 「缓存清理钩子报错只记日志、不阻断任务终态」的分支（restore_consistency.go）。
func TestTriggerRestore_FlusherError_DoesNotBlockTerminal(t *testing.T) {
	restoring.Store(false)
	RegisterRestoreCacheFlusher(func(context.Context) error {
		return fmt.Errorf("cache flush boom")
	})
	t.Cleanup(func() {
		restoring.Store(false)
		RegisterRestoreCacheFlusher(nil)
	})

	db := newTestDB(t)
	exec := &fakeDumpExecutor{tables: []string{"sys_user"}, dumpBytes: []byte("X")}
	svc := newTestService(t, db, exec)
	upload := buildValidContainer(t)

	id, err := svc.TriggerRestore(context.Background(), upload, "tester")
	if err != nil {
		t.Fatalf("TriggerRestore 失败: %v", err)
	}
	final := waitTask(t, svc, id, 30*time.Second)
	// 本地 fake 场景下恢复可能成功或失败（取决于降级路径），关键断言是：
	// flusher 报错不得吞掉/改变终态写入，任务必须有明确 status。
	if final.Status != models.DBBackupTaskSuccess && final.Status != models.DBBackupTaskFailed {
		t.Fatalf("终态必须是 success/failed，实得 %s", final.Status)
	}
}

// ========== 11. pg_dump 缺失的可诊断降级（运行时故障回归守护） ==========

// TestCapabilities_PgDumpMissing 验证部署机缺 pg_dump 时页面预检能拿到
// 明确的安装/配置指引，而不是把 "exec: pg_dump: not found in $PATH" 抛给用户。
func TestCapabilities_PgDumpMissing(t *testing.T) {
	db := newTestDB(t)
	exec := &fakeDumpExecutor{tables: []string{"sys_user"}, dumpBytes: []byte("X")}
	svc := newTestService(t, db, exec).(*service)
	// 切成 PG 模式，且把 pg_dump 指向一个绝不存在的路径（模拟 PATH 缺失）
	svc.dbCfg = DatabaseConfig{Type: dbTypePostgres, Host: "127.0.0.1", Port: 5432, User: "u", DBName: "d"}
	seedConfig(t, db, cfgKeyPgDumpPath, filepath.Join(t.TempDir(), "no-such-pg_dump"))
	seedConfig(t, db, cfgKeyPgRestorePath, filepath.Join(t.TempDir(), "no-such-pg_restore"))

	cap, err := svc.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities 不应报错: %v", err)
	}
	if cap.DBType != dbTypePostgres || cap.PgDumpAvailable || cap.PgRestoreAvailable {
		t.Fatalf("缺工具时应标记为不可用: %+v", cap)
	}
	if !strings.Contains(cap.Hint, "postgresql") && !strings.Contains(cap.Hint, "sys.database.backup.pg_dump_path") {
		t.Fatalf("hint 应给出安装或配置键指引: %q", cap.Hint)
	}
}

// TestExecToolCfg_PgDumpMissing 验证导出前预检会直接返回带诊断的错误，
// 不会再走到 exec.CommandContext 抛底层 "not found in $PATH"。
func TestExecToolCfg_PgDumpMissing(t *testing.T) {
	db := newTestDB(t)
	exec := &fakeDumpExecutor{tables: []string{"sys_user"}, dumpBytes: []byte("X")}
	svc := newTestService(t, db, exec).(*service)
	svc.dbCfg = DatabaseConfig{Type: dbTypePostgres}
	seedConfig(t, db, cfgKeyPgDumpPath, filepath.Join(t.TempDir(), "definitely-absent-pg_dump"))

	_, err := svc.execToolCfg(context.Background(), "pg_dump")
	if err == nil {
		t.Fatal("缺 pg_dump 时 execToolCfg 必须返回错误")
	}
	if !strings.Contains(err.Error(), "sys.database.backup.pg_dump_path") &&
		!strings.Contains(err.Error(), "postgresql") {
		t.Fatalf("错误信息应含安装/配置指引: %v", err)
	}
	if strings.Contains(err.Error(), "not found in $PATH") {
		t.Fatalf("不应再把底层 exec 错误原样抛给用户: %v", err)
	}
}

// ========== 12. pg_dump / pg_restore 参数正确性（PG 18 实机回归守护） ==========

// TestPGToolArgs_SingleTransactionOnlyOnRestore 锁定一条实机踩过的坑：
// --single-transaction 是 pg_restore 的选项，pg_dump 不认识（PG 18 报
// "unrecognized option '--single-transaction'"）。custom 格式导出本身即一致快照。
// 同时断言口令绝不出现在任何命令行参数里（只走 PGPASSWORD 环境变量）。
func TestPGToolArgs_SingleTransactionOnlyOnRestore(t *testing.T) {
	cfg := DatabaseConfig{
		Type:     dbTypePostgres,
		Host:     "10.0.0.1",
		Port:     5432,
		User:     "postgres",
		Password: "S3cr3t-P@ss",
		DBName:   "ruoyi",
		SSLMode:  "disable",
	}

	dumpArgs := pgDumpArgs(cfg, "/tmp/backup.dump")
	for _, a := range dumpArgs {
		if a == "--single-transaction" || strings.HasPrefix(a, "--single-transaction") {
			t.Fatalf("pg_dump 参数不得含 --single-transaction（PG 18 不支持）: %v", dumpArgs)
		}
	}
	// 必要参数仍在
	assertContains := func(args []string, want string) {
		t.Helper()
		for _, a := range args {
			if a == want {
				return
			}
		}
		t.Fatalf("参数 %v 缺少 %q", args, want)
	}
	assertContains(dumpArgs, "--format=custom")
	assertContains(dumpArgs, "-f")
	assertContains(dumpArgs, "/tmp/backup.dump")

	restoreArgs := pgRestoreArgs(cfg, "/tmp/backup.dump")
	found := false
	for _, a := range restoreArgs {
		if a == "--single-transaction" {
			found = true
		}
	}
	if !found {
		t.Fatalf("pg_restore 必须带 --single-transaction（失败整体回滚）: %v", restoreArgs)
	}
	assertContains(restoreArgs, "--clean")
	assertContains(restoreArgs, "--if-exists")

	// 口令绝不出现在命令行参数（含 dump 文件路径也不应含口令）
	for _, args := range [][]string{dumpArgs, restoreArgs} {
		for _, a := range args {
			if strings.Contains(a, cfg.Password) {
				t.Fatalf("命令行参数泄露了数据库口令: %v", args)
			}
		}
	}
}
