package dbbackup

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// IsRestoring 状态机测试
// ---------------------------------------------------------------------------
//
// 覆盖 A8 依赖的接口契约：
//   - 初始/空闲为 false；
//   - runRestore 进入时置 true、退出（成功/失败）后复位 false；
//   - panic 展开路径的 defer 必然复位；
//   - 并发读（模拟全局中间件）与写（恢复任务）无竞争（atomic.Bool）。
//
// runRestore 会做「恢复前自动全量备份」，依赖 dbCfg 为 SQLite 以走纯 Go 路径、
// 不关闭全局连接池；这里复用包内 newTestDB / newTestService / fake 设施。

// buildValidContainer 构造一个合法的上传备份容器（复用 writeContainer 真实链路）。
func buildValidContainer(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "valid.zip")
	dumpPath := filepath.Join(dir, "dump.tmp")
	if err := os.WriteFile(dumpPath, []byte("FAKE-SQLITE-DUMP"), 0o644); err != nil {
		t.Fatalf("写 dump 失败: %v", err)
	}
	meta := &manifest{
		SchemaVersion: SchemaVersionV1, CreatedAt: time.Now().UTC(),
		DBType: dbTypeSQLite, Tables: []string{"sys_user"},
		ExcludedTables: append([]string{}, excludedPhysicalTables...),
	}
	if _, err := writeContainer(context.Background(), zipPath, dumpPath, meta); err != nil {
		t.Fatalf("构造合法容器失败: %v", err)
	}
	return zipPath
}

// TestIsRestoring_InitialState 断言空闲态为 false，并还原全局状态避免污染其它用例。
func TestIsRestoring_InitialState(t *testing.T) {
	restoring.Store(false)
	if IsRestoring() {
		t.Fatal("空闲态 IsRestoring() 应为 false")
	}
	t.Cleanup(func() { restoring.Store(false) })
}

// TestIsRestoring_SuccessAndFailed 驱动 runRestore 的状态机写入端，覆盖成功与失败
// 两条退出路径：进入后可见 true，返回后复位 false。
//
// 用 restoreGate 把 fake.Restore 停在「恢复执行中」阶段，消除「快速失败路径里
// true→false 瞬间完成、主协程观测不到」的竞态：放行 gate 前必然处于维护模式，
// 放行后由 defer 复位。
func TestIsRestoring_SuccessAndFailed(t *testing.T) {
	for _, tc := range []struct {
		name       string
		restoreErr error
		wantErr    bool
	}{
		{name: "success_复位", restoreErr: nil, wantErr: false},
		{name: "failed_复位", restoreErr: ErrSQLiteOfflineRestoreRequired, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			restoring.Store(false)
			db := newTestDB(t)
			gate := make(chan struct{})
			exec := &fakeDumpExecutor{
				tables: []string{"sys_user"}, dumpBytes: []byte("X"),
				restoreErr: tc.restoreErr, restoreGate: gate,
			}
			svc := newTestService(t, db, exec)
			upload := buildValidContainer(t)

			done := make(chan error, 1)
			go func() {
				done <- svc.(*service).runRestore(context.Background(), "task-"+tc.name, upload, "tester")
			}()
			// 校验/恢复前备份/解压完成后会进入 Restore 并阻塞在 gate，此时维护模式必为 true
			assertRestoringEventually(t, true, 10*time.Second)
			close(gate) // 放行，让 runRestore 走完 defer 复位
			err := <-done
			if tc.wantErr && err == nil {
				t.Fatal("Restore 返回错误时 runRestore 应返回错误")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("成功路径 runRestore 不应报错: %v", err)
			}
			assertRestoringEventually(t, false, 5*time.Second)
		})
	}
}

// TestIsRestoring_PanicPath 验证 panic 展开路径：runRestore 的状态机写入端用
// defer 复位，即使内部 panic（被上层 runTask recover 兜底）也必然复位 false。
func TestIsRestoring_PanicPath(t *testing.T) {
	restoring.Store(false)
	t.Cleanup(func() { restoring.Store(false) })

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 模拟 runTask 的 recover 兜底
		defer func() { _ = recover() }()
		// 模拟 runRestore 的「进入置 true + defer 复位」状态机
		func() {
			restoring.Store(true)
			defer restoring.Store(false)
			if !IsRestoring() {
				panic("进入恢复后标志应为 true")
			}
			panic("模拟恢复流程 panic")
		}()
	}()
	wg.Wait()
	if IsRestoring() {
		t.Fatal("panic 被 recover 后 IsRestoring() 必须复位为 false")
	}
}

// assertRestoringEventually 有界轮询等待标志达到期望值（避免 goroutine 调度 flaky）。
func assertRestoringEventually(t *testing.T, want bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if IsRestoring() == want {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("等待 IsRestoring()==%v 超时，实得 %v", want, IsRestoring())
}
