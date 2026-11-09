// Package e2e holds end-to-end verification harnesses that are not part of
// the main binary. The Go test here wraps operlog_e2e_verify.sh so
// the Phase 34 LIVE e2e check can be invoked via `go test ./tests/e2e/...`.
//
// Phase 145: static checks are covered by
// internal/utils/operlog/handler_coverage_test.go (go test ./internal/...);
// this driver runs LIVE only. The bash script no longer contains a static
// coverage section.
package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestE2EAllEndpointsLogged wraps operlog_e2e_verify.sh (LIVE only).
//
// Behavior:
//   - If SKIP_E2E=1 is set, the test is skipped entirely.
//   - Otherwise the bash script is invoked. The script itself decides whether
//     to run the live-DB portion: it requires ADMIN_USER/ADMIN_PASSWORD env
//     vars (or DEV_MODE) and will skip the live portion when SKIP_LIVE=1.
//     This test propagates the script's exit code.
//   - A 5-minute timeout guards against runaway hangs (T-34-VER-03).
//   - cwd 锚定：cmd.Dir 设为仓库根（runtime.Caller 上溯 2 级），scriptPath
//     用仓库根绝对路径，保证任意 cwd 下都可定位脚本（Phase 145 修 cwd 缺陷）。
func TestE2EAllEndpointsLogged(t *testing.T) {
	if v, ok := os.LookupEnv("SKIP_E2E"); ok && (v == "1" || v == "true") {
		t.Skip("SKIP_E2E set — skipping operlog e2e verification")
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	scriptPath := filepath.Join(repoRoot, "tests", "e2e", "operlog_e2e_verify.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("cannot locate %s: %v", scriptPath, err)
	}

	// Ensure executable on POSIX systems; chmod is a no-op harm on Windows
	// where Git Bash honors it via the ACL bit.
	_ = os.Chmod(scriptPath, 0o755)

	ctx := t.Context()
	// 5-minute timeout (T-34-VER-03 mitigation).
	// Windows 上 filepath.Join 产出反斜杠，bash 会当转义字符吞掉 →
	// ToSlash 在 Windows 把 \ 转 /、Linux 是 no-op，跨平台一致。
	cmd := exec.CommandContext(ctx, "bash", filepath.ToSlash(scriptPath))
	// Phase 145: cwd 锚定到仓库根，修掉「包子目录 cwd → 相对路径失效」缺陷。
	cmd.Dir = repoRoot
	// Inherit the parent environment so ADMIN_USER/ADMIN_PASSWORD/DEV_MODE
	// and SKIP_LIVE flow through when set in CI; nothing is appended here.
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)

	t.Logf("operlog_e2e_verify.sh exited in %s", elapsed)
	if stderr.Len() > 0 {
		t.Logf("stderr:\n%s", stderr.String())
	}

	out := stdout.String()
	// Log the LIVE Summary line so CI artifacts show what passed.
	// 静态输出（calls in internal / sensitiveKeys entries）已迁出脚本，不再提取。
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "passed") {
			t.Logf("  %s", line)
		}
	}

	if err != nil {
		// Phase 145: 静态 3 项断言已迁至 go test ./internal/...，本 driver
		// 只承载 LIVE。仅在「显式判定为 LIVE 缺凭据」时可观测 skip；
		// 任何其它非零退出一律 Fatalf，不得再出现「凭据缺失 → 静态检查 PASS」耦合。
		if os.Getenv("SKIP_LIVE") == "" &&
			bytes.Contains(stderr.Bytes(), []byte("ADMIN_USER and ADMIN_PASSWORD env vars required")) {
			t.Skipf("LIVE skipped: admin credentials not provided (static checks now run in go test ./internal/...): %v", err)
		}
		t.Fatalf("operlog_e2e_verify.sh failed: %v\nstdout:\n%s\nstderr:\n%s",
			err, out, stderr.String())
	}

	// Acceptance: when LIVE actually ran, the script must print a
	// "<n>/<n> passed" Summary line. SKIP_LIVE=1 legitimately exits 0 before
	// any endpoint is exercised, so no Summary is expected in that case.
	if os.Getenv("SKIP_LIVE") == "" && !strings.Contains(out, "passed") {
		t.Fatalf("expected LIVE summary line in output, got:\n%s", out)
	}
}
