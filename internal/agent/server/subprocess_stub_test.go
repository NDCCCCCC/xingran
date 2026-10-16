package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

// TestHelperProcess is not a real test: it is the entry point for the re-exec
// subprocess stub (Go stdlib pattern, cf. GOROOT src/os/os_test.go). When this
// test binary runs with GO_WANT_HELPER_PROCESS=1 and
// -test.run=^TestHelperProcess$, the guard passes and the FIRST argv element
// after "--" (the shape argument) selects the stub behavior. Any further
// trailing argv elements are extras forwarded from the parent seam override
// (e.g. strategy 的 runCommand("powershell","-Command","<script>") 的真实 args),
// 供 echo-args 形态回显辅助断言。每分支必达 os.Exit。
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" { // guard must stay first
		return
	}
	// shape 扫描: 从 "--" 后取第一个元素作为 shape, 其余为 extras
	shape := ""
	var extras []string
	for i, a := range os.Args {
		if a == "--" && i+1 < len(os.Args) {
			shape = os.Args[i+1]
			extras = os.Args[i+2:]
			break
		}
	}
	if shape == "" {
		shape = os.Args[len(os.Args)-1] // 兼容旧 path 调用方式
	}
	switch {
	case shape == "sleep-until-stdin-close":
		// Long-lived shape: exit only after the parent closes our stdin.
		io.Copy(io.Discard, os.Stdin)
	case shape == "ignore-sigterm" && runtime.GOOS == "linux":
		// SIGTERM has no meaning on Windows; non-linux falls through to default.
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM)
		// Ready marker: lets the parent synchronize on handler installation
		// so its SIGTERM can never race the Notify call.
		fmt.Println("sigterm-armed")
		<-sig
		fmt.Println("still-alive")
	case shape == "stdout-flood":
		for i := 0; i < 1000; i++ {
			fmt.Println("line")
		}
	case shape == "echo-args":
		// 通用成功桩: 退出 0, 无 stdout 污染; 父侧 closure 直接捕获 args 无需回显
		_ = extras
	case shape == "exit-1":
		// 失败桩: 直接退出码 1, 驱动 Run/Output 返回 *exec.ExitError
		os.Exit(1)
	case shape == "passwd-style":
		// 读 stdin 一行 (用户:密码 或 sudoers 内容), 原样回显到 stdout 后退出 0
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			fmt.Print(scanner.Text())
		}
	case shape == "print-users":
		// 打印多行用户数据模拟 getent / Get-LocalUser 输出; 内容可经
		// STUB_USERS_77 环境变量覆盖 (windows parser 用纯姓名, linux parser 用
		// getent 格式)。默认: 两条 uid>=1000 用户 + 一条系统用户。
		content := os.Getenv("STUB_USERS_77")
		if content == "" {
			content = "root:x:0:0:root:/root:/bin/bash\n" +
				"sysuser:x:999:999::/:/sbin/nologin\n" +
				"alice:x:1000:1000::/home/alice:/bin/bash\n" +
				"bob:x:1001:1001::/home/bob:/bin/bash\n"
		}
		fmt.Print(content)
	default:
		// Fast-exit shape: print "hello" and exit 0. (默认 fallback, 兼容旧调用)
		fmt.Println("hello")
	}
	os.Exit(0)
}

// helperStubCommand builds a re-exec subprocess stub command: it executes this
// test binary (os.Args[0]) filtered down to TestHelperProcess only, passes the
// shape argument after "--" (positional args are never consumed by the testing
// flag machinery), and arms the GO_WANT_HELPER_PROCESS gate the helper guard
// requires.
//
// This is the Go stdlib subprocess-stub pattern (cf. GOROOT src/os/os_test.go).
// It replaces the former platform-dependent exec.Command("echo", ...) stubs:
// the stub is the test binary itself, so its behavior is identical on every
// platform ("echo" is a cmd.exe builtin on Windows and only worked by accident
// when a Git Bash echo.exe happened to be on PATH).
func helperStubCommand(t *testing.T, shape string) *exec.Cmd {
	t.Helper()
	ctx := context.Background()
	cmd := newCommand(ctx, os.Args[0], "-test.run=^TestHelperProcess$", "--", shape)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	return cmd
}

// TestSubprocessStub_Default verifies the fast-exit shape: the helper stub
// prints "hello" and exits 0.
func TestSubprocessStub_Default(t *testing.T) {
	cmd := helperStubCommand(t, "hello")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("helper stub exited with error: %v", err)
	}
	if !strings.Contains(string(output), "hello") {
		t.Fatalf("expected output to contain %q, got %q", "hello", string(output))
	}
}

// TestSubprocessStub_StdoutFlood verifies the stdout-flood shape: the helper
// stub prints 1000 lines and exits 0 (output capture must not deadlock).
func TestSubprocessStub_StdoutFlood(t *testing.T) {
	cmd := helperStubCommand(t, "stdout-flood")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("helper stub exited with error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 1000 {
		t.Fatalf("expected >= 1000 output lines, got %d", len(lines))
	}
	if !strings.Contains(string(output), "line") {
		t.Fatalf("expected flood output to contain %q", "line")
	}
}

// TestSubprocessStub_StdinClose verifies the long-lived shape: the helper stub
// keeps running until its stdin is closed, then exits cleanly (Wait == nil).
func TestSubprocessStub_StdinClose(t *testing.T) {
	cmd := helperStubCommand(t, "sleep-until-stdin-close")
	pipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	// Closing stdin is the shutdown signal for this shape.
	if err := pipe.Close(); err != nil {
		t.Fatalf("close stdin pipe: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("Wait after stdin close: %v", err)
	}
}

// TestSubprocessStub_IgnoreSigterm verifies the ignore-sigterm shape: the
// helper stub arms a SIGTERM handler, survives a SIGTERM, prints
// "still-alive" and only then exits. syscall.SIGTERM has no meaning on
// Windows, so the test is linux-only.
func TestSubprocessStub_IgnoreSigterm(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("syscall.SIGTERM is not meaningful on %s", runtime.GOOS)
	}

	cmd := helperStubCommand(t, "ignore-sigterm")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait until the child reports its signal handler is armed, so the
	// SIGTERM we send can never race the handler installation and kill the
	// process via the default disposition.
	scanner := bufio.NewScanner(stdout)
	armed := false
	for scanner.Scan() {
		if scanner.Text() == "sigterm-armed" {
			armed = true
			break
		}
	}
	if !armed {
		_ = cmd.Wait()
		t.Fatalf("helper stub never armed its SIGTERM handler (scan err: %v)", scanner.Err())
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM: %v", err)
	}

	stillAlive := false
	for scanner.Scan() {
		if scanner.Text() == "still-alive" {
			stillAlive = true
			break
		}
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("Wait after SIGTERM: %v", err)
	}
	if !stillAlive {
		t.Fatal("expected stub to print still-alive after ignoring SIGTERM")
	}
}
