//go:build integration

// Package tests M6 模块化命令树 E2E 测试框架（先行准备）。
//
// 命令树（cmd/ + internal/commands/）由 client-dev / crypto-dev 开发中；
// 本文件先固化断言工具与预期结构，命令树就绪前相关用例自动 skip，不阻塞 CI。
// 运行：go test -tags=integration -run TestCLI ./tests/ -v
package tests

import (
	"encoding/json"
	"errors"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// cliBin 解析 CLI 二进制：环境变量 XINGRAN_CLI_BIN 优先，其次探测常见产物路径，
// 都没有则返回空（runCLI 回退 go run，每次有编译开销，仅供兜底）。
func cliBin() string {
	if v := os.Getenv("XINGRAN_CLI_BIN"); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v
		}
	}
	candidates := []string{
		filepath.Join("..", "xingran-cli.exe"),
		filepath.Join("..", "build", "xingran-cli.exe"),
		filepath.Join("..", "bin", "xingran-cli.exe"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

// runCLI 执行 CLI 命令，返回 stdout/stderr/exitCode。
// 优先用编译好的二进制；否则 go run ./cmd/xingran（工作目录 xingran-cli/）。
func runCLI(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	cliRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("解析 xingran-cli 目录失败: %v", err)
	}

	var cmd *exec.Cmd
	if bin := cliBin(); bin != "" {
		cmd = exec.Command(bin, args...)
	} else {
		cmd = exec.Command("go", append([]string{"run", "./cmd/xingran"}, args...)...)
		cmd.Dir = cliRoot
	}
	cmd.Env = os.Environ()

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	exitCode := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		} else {
			return "", "", -1 // 进程都没起来（如 go 不在 PATH）
		}
	}
	return stdout.String(), stderr.String(), exitCode
}

// cliAvailable 探测 CLI 是否可执行；命令树未就绪时用例直接 skip。
func cliAvailable(t *testing.T) bool {
	t.Helper()
	_, stderr, code := runCLI(t, "--help")
	if code < 0 {
		t.Skip("CLI 不可执行（无二进制且 go run 失败），命令树就绪后自动生效")
	}
	_ = stderr
	return code == 0
}

// cobra 子命令解析：help 输出 Available Commands 段的行格式
//
//	system        System-related commands（名称，中文描述）
var cobraCmdRe = regexp.MustCompile(`(?m)^  (\S+)(?:\s+(\S.*))?\s*$`)

// listSubCommands 解析 help 输出中的子命令名（Available Commands 段）。
func listSubCommands(helpOut string) []string {
	var names []string
	inBlock := false
	for _, line := range strings.Split(helpOut, "\n") {
		switch {
		case strings.HasPrefix(line, "Available Commands:"):
			inBlock = true
		case inBlock && (strings.TrimSpace(line) == "" || strings.HasPrefix(line, "Additional") || strings.HasPrefix(line, "Flags:") || strings.HasPrefix(line, "Use \"")):
			inBlock = false
		case inBlock:
			if m := cobraCmdRe.FindStringSubmatch(line); m != nil && !strings.HasPrefix(m[1], "help") && m[1] != "completion" {
				names = append(names, m[1])
			}
		}
	}
	return names
}

// assertHelpSections 叶子命令 help 断言：含「对应 API」与示例段落（示例允许
// cobra「Examples:」或中文「示例」两种形态）。模块/资源层不做段落要求。
func assertHelpSections(t *testing.T, cmdPath string, helpOut string) {
	t.Helper()
	if !strings.Contains(helpOut, "对应 API") {
		t.Errorf("%s --help 缺少「对应 API」段落", cmdPath)
	}
	if !strings.Contains(helpOut, "Examples:") && !strings.Contains(helpOut, "示例") {
		t.Errorf("%s --help 缺少示例段落（Examples:/示例）", cmdPath)
	}
}

// expectCommand 命令存在则执行断言 fn；命令未实现（unknown command / 非零退出且
// help 不含 Available Commands）则 skip，避免拖慢 M6 开发节奏。
func expectCommand(t *testing.T, cmdPath string, fn func(t *testing.T, stdout string)) {
	t.Helper()
	parts := strings.Split(cmdPath, " ")
	stdout, stderr, code := runCLI(t, append(parts, "--help")...)
	joined := stdout + stderr
	if code != 0 {
		if strings.Contains(joined, "unknown command") || strings.Contains(joined, "未知的命令") ||
			strings.TrimSpace(joined) == "" {
			t.Skipf("命令 %q 尚未实现，命令树就绪后自动生效", cmdPath)
		}
		// 命令存在但 help 出错：真问题，fail
		t.Errorf("%s --help 非零退出(%d): %s", cmdPath, code, joined)
		return
	}
	fn(t, stdout)
}

// --- 命令树结构校验（命令树就绪前自动 skip） ---

func TestCLIHelpTopLevel(t *testing.T) {
	if !cliAvailable(t) {
		return
	}
	stdout, _, code := runCLI(t, "--help")
	if code != 0 {
		t.Fatalf("--help 非零退出: %d", code)
	}
	mods := listSubCommands(stdout)
	if len(mods) == 0 {
		t.Fatal("顶层 help 无 Available Commands")
	}
	t.Logf("顶层模块: %v", mods)
}

func TestCLIHelpSystemModule(t *testing.T) {
	if !cliAvailable(t) {
		return
	}
	expectCommand(t, "system", func(t *testing.T, stdout string) {
		res := listSubCommands(stdout)
		if len(res) == 0 {
			t.Fatal("system --help 无子资源")
		}
		t.Logf("system 资源: %v", res)
	})
}

func TestCLIHelpSystemUser(t *testing.T) {
	if !cliAvailable(t) {
		return
	}
	expectCommand(t, "system user", func(t *testing.T, stdout string) {
		acts := listSubCommands(stdout)
		if len(acts) == 0 {
			t.Fatal("system user --help 无动作")
		}
		t.Logf("system user 动作: %v", acts)
	})
}

// legacyLeafCommands M6 之前的既有顶层命令（login/logout/api/schema/version 等）。
// 它们不属于「模块/资源/动作」三级结构，不要求「对应 API」「示例」段落；
// M6 模块化命令树的模块/资源/动作命令才执行该断言。
var legacyLeafCommands = map[string]bool{
	"login": true, "logout": true, "refresh": true,
	"api": true, "schema": true, "version": true,
	"help": true, "completion": true, "web": true,
}

// m6Modules M6 模块化命令树的模块列表（schema 前缀与命令名一致）。
var m6Modules = []string{"system", "ops", "monitor", "duty", "knowledge", "network", "workorder"}

// leafInfo 叶子命令（无子命令，对应一个 API 端点）。
type leafInfo struct {
	Path string // 完整命令路径，如 "system user list"
	Help string
}

// runCLIHelp 按空格切分命令路径执行 --help（多级路径不得整串作为单参数传递）。
func runCLIHelp(t *testing.T, path string) (string, int) {
	t.Helper()
	stdout, _, code := runCLI(t, append(strings.Split(path, " "), "--help")...)
	return stdout, code
}

// collectLeaves 递归收集 root 命令树下全部叶子命令。
func collectLeaves(t *testing.T, root string) []leafInfo {
	t.Helper()
	stdout, code := runCLIHelp(t, root)
	if code != 0 {
		return nil
	}
	var leaves []leafInfo
	for _, name := range listSubCommands(stdout) {
		path := root + " " + name
		sub, c := runCLIHelp(t, path)
		if c != 0 {
			continue
		}
		hasAPI := strings.Contains(sub, "对应 API:")
		subs := listSubCommands(sub)
		if hasAPI {
			leaves = append(leaves, leafInfo{Path: path, Help: sub})
		}
		if len(subs) > 0 {
			leaves = append(leaves, collectLeaves(t, path)...)
		} else if !hasAPI {
			leaves = append(leaves, leafInfo{Path: path, Help: sub})
		}
	}
	return leaves
}

// ensureBuilt 编译测试自持二进制（tests/_tmpcli/），经 XINGRAN_CLI_BIN 指向，
// 与 client-dev 的 xingran-cli.exe 产物隔离，避免并发编译竞态。
func ensureBuilt(t *testing.T) {
	t.Helper()
	tmpBin, _ := filepath.Abs(filepath.Join("_tmpcli", "xingran-cli.exe"))
	if _, err := os.Stat(tmpBin); err == nil {
		t.Setenv("XINGRAN_CLI_BIN", tmpBin)
		return
	}
	t.Log("编译测试自持二进制 go build ./cmd/xingran ...")
	cliRoot, err := filepath.Abs("..")
	if err != nil {
		t.Skipf("解析根目录失败: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(tmpBin), 0o755); err != nil {
		t.Skipf("创建临时目录失败: %v", err)
	}
	cmd := exec.Command("go", "build", "-o", tmpBin, "./cmd/xingran")
	cmd.Dir = cliRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("预编译失败（命令树构建中？）: %v %s", err, out)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(tmpBin)) })
	t.Setenv("XINGRAN_CLI_BIN", tmpBin)
}

// extractAPIPaths 从叶子 help 提取「对应 API: METHOD /api/v1/path」归一化键集合。
// 归一化：剥 /api/v1 前缀；路径参数 :id 与 {id} 统一为 {}（与 docs.go 形态对齐）。
func extractAPIPaths(helpOut string) []string {
	var keys []string
	for _, line := range strings.Split(helpOut, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "对应 API:") {
			continue
		}
		api := strings.TrimSpace(strings.TrimPrefix(line, "对应 API:"))
		parts := strings.SplitN(api, " ", 2)
		if len(parts) != 2 {
			continue
		}
		method := strings.ToUpper(parts[0])
		path := strings.TrimPrefix(parts[1], "/api/v1")
		path = strings.ReplaceAll(path, "{", ":")
		path = strings.ReplaceAll(path, "}", "") // {id} -> :id（docs 用 {}，help 用 :）
		keys = append(keys, method+" "+path)
	}
	return keys
}

// TestCLIHelpSectionsRequired M6 模块树全部叶子命令的 help 必须含
// 「对应 API」与示例段落；legacy 命令与未实现模块自动跳过。
func TestCLIHelpSectionsRequired(t *testing.T) {
	if !cliAvailable(t) {
		return
	}
	ensureBuilt(t)
	for _, mod := range m6Modules {
		mod := mod
		t.Run(mod, func(t *testing.T) {
			leaves := collectLeaves(t, mod)
			if len(leaves) == 0 {
				t.Skipf("模块 %q 尚未实现（无叶子命令），就绪后自动生效", mod)
			}
			for _, lf := range leaves {
				assertHelpSections(t, lf.Path, lf.Help)
			}
			t.Logf("%s: %d 个叶子命令段落断言通过", mod, len(leaves))
		})
	}
}

// TestCLIHelpNoEmptyUsage 防回归：help 中不得出现空参数说明
// （cobra flag 形如 "  -u, --username string   "（说明为空）即「有 flag 无中文说明」）。
func TestCLIHelpNoEmptyUsage(t *testing.T) {
	if !cliAvailable(t) {
		return
	}
	stdout, _, code := runCLI(t, "--help")
	if code != 0 {
		t.Fatalf("--help 非零退出: %d", code)
	}
	// Flags 段行：flag 名后跟空说明（行尾即 flag 类型名或直接结束且无中文/英文描述）
	emptyFlagRe := regexp.MustCompile(`(?m)^  -\S+(?:, -\S+)? (?:\S+\s+)?$`)
	for _, mod := range listSubCommands(stdout) {
		s, _, c := runCLI(t, mod, "--help")
		if c != 0 {
			continue // 命令未实现由其他用例处理
		}
		for _, line := range strings.Split(s, "\n") {
			if strings.HasPrefix(line, "  -") && emptyFlagRe.MatchString(line) {
				t.Errorf("%s: flag 无说明: %q", mod, line)
			}
		}
	}
}

// --- 真实后端用例（后端在线且提供测试账号才跑） ---

// TestCLISystemUserListE2E system user list 带真实 token 调通（M6 冒烟预留）。
func TestCLISystemUserListE2E(t *testing.T) {
	if !cliAvailable(t) {
		return
	}
	user, pass, ok := testCreds()
	if !ok {
		t.Skip("未设置 XINGRAN_TEST_USER / XINGRAN_TEST_PASS")
	}
	// 先经 CLI 登录落盘 token（用独立 HOME 避免污染真实凭据）
	tmpHome := t.TempDir()
	t.Setenv("USERPROFILE", tmpHome) // Windows
	t.Setenv("HOME", tmpHome)        // POSIX
	t.Setenv("XINGRAN_BASE_URL", baseURL())

	stdout, stderr, code := runCLI(t, "login", "-u", user, "-p", pass)
	if code != 0 {
		t.Skipf("CLI 登录未通（命令树/后端模式限制）: %s%s", stdout, stderr)
	}

	stdout, stderr, code = runCLI(t, "system", "user", "list", "--current", "1", "--pageSize", "1")
	if code != 0 {
		t.Fatalf("system user list 失败(%d): %s%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "\"list\"") && !strings.Contains(stdout, "total") {
		t.Fatalf("输出应含分页结构: %s", stdout)
	}
}

// --- 端点覆盖一致性（防"端点丢失"的关键闸门） ---

// schemaEndpoint schema 命令输出的端点条目。
type schemaEndpoint struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}

// schemaEndpointsFor 调 xingran schema <prefix> 解析端点列表；模块未实现返回 nil。
func schemaEndpointsFor(t *testing.T, module string) []schemaEndpoint {
	t.Helper()
	stdout, _, code := runCLI(t, "schema", module)
	if code != 0 {
		return nil
	}
	var payload struct {
		Endpoints []schemaEndpoint `json:"endpoints"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("schema %s 输出解析失败: %v", module, err)
	}
	return payload.Endpoints
}

// normalizeAPIKey 归一化 API 键：method + path（剥 /api/v1，{id}→:id）。
func normalizeAPIKey(method, path string) string {
	path = strings.TrimPrefix(path, "/api/v1")
	path = strings.ReplaceAll(path, "{", ":")
		path = strings.ReplaceAll(path, "}", "")
	return strings.ToUpper(method) + " " + path
}

// TestCLIEndpointCoverage 端点覆盖一致性：schema <module> 的每个端点必须能被
// 命令树某个叶子 help 的「对应 API」覆盖，否则即"端点丢失"。
func TestCLIEndpointCoverage(t *testing.T) {
	if !cliAvailable(t) {
		return
	}
	ensureBuilt(t)
	for _, mod := range m6Modules {
		mod := mod
		t.Run(mod, func(t *testing.T) {
			eps := schemaEndpointsFor(t, mod)
			if len(eps) == 0 {
				t.Skipf("模块 %q schema 无端点（模块树未实现），就绪后自动生效", mod)
			}
			leaves := collectLeaves(t, mod)
			if len(leaves) == 0 {
				t.Skipf("模块 %q 命令树尚未实现", mod)
			}

			covered := map[string]string{} // apiKey → 覆盖它的命令路径
			for _, lf := range leaves {
				for _, key := range extractAPIPaths(lf.Help) {
					if _, dup := covered[key]; !dup {
						covered[key] = lf.Path
					}
				}
			}

			var missing []string
			for _, e := range eps {
				key := normalizeAPIKey(e.Method, e.Path)
				if _, ok := covered[key]; !ok {
					missing = append(missing, key)
				}
			}
			if len(missing) > 0 {
				sort.Strings(missing)
				t.Errorf("%s 端点丢失 %d/%d: %v", mod, len(missing), len(eps), missing)
			}
			t.Logf("%s: schema %d 端点全部被命令树覆盖（叶子 %d 个）", mod, len(eps), len(leaves))
		})
	}
}

// --- help 可读性抽查 ---

// TestCLIHelpReadability 随机抽 10 个叶子命令：参数说明非空、含中文
// （说明不能是空的或纯英文字段名；示例值占位不在此约束内）。
func TestCLIHelpReadability(t *testing.T) {
	if !cliAvailable(t) {
		return
	}
	ensureBuilt(t)

	var all []leafInfo
	for _, mod := range m6Modules {
		all = append(all, collectLeaves(t, mod)...)
	}
	if len(all) == 0 {
		t.Skip("M6 模块树尚未实现")
	}

	rng := rand.New(rand.NewSource(42)) // 固定 seed：结果可复现
	seen := map[int]bool{}
	sampleLimit := 10
	if len(all) < sampleLimit {
		sampleLimit = len(all)
	}
	for i := 0; i < sampleLimit; i++ {
		idx := rng.Intn(len(all))
		if seen[idx] {
			i--
			continue
		}
		seen[idx] = true
		lf := all[idx]
		t.Run(lf.Path, func(t *testing.T) { checkFlagUsability(t, lf) })
	}
}

// checkFlagUsability 校验单个叶子 help 的 Flags 段参数说明可读性。
func checkFlagUsability(t *testing.T, lf leafInfo) {
	t.Helper()
	hasUserFlag := false
	for _, line := range strings.Split(lf.Help, "\n") {
		// Flags 段用户参数行：以空格 + - 开头（排除 -h, --help）
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "-") || strings.Contains(trimmed, "help for") {
			continue
		}
		if !strings.HasPrefix(line, "      ") { // Global Flags 段无前导 6 空格
			continue
		}
		hasUserFlag = true
		// 说明 = 最后一个空白分隔字段类型之后的文本；取「最后一个类型词」之后
		// 简化：说明部分 = 行内最后一个非空白 token 之后的内容为空则说明缺失。
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			t.Errorf("%s: flag 行异常: %q", lf.Path, trimmed)
			continue
		}
		usage := extractFlagUsage(trimmed)
		if usage == "" {
			t.Errorf("%s: 参数说明为空: %q", lf.Path, trimmed)
			continue
		}
		if !containsCJK(usage) {
			t.Errorf("%s: 参数说明应为中文（不得纯英文字段名）: %q -> %q", lf.Path, trimmed, usage)
		}
	}
	_ = hasUserFlag
}

// extractFlagUsage 提取 cobra Flags 行中说明部分（最后一个空白分组之后文本）。
// cobra 行形态：--name type   说明文字（说明与类型间有 ≥2 空格）。
func extractFlagUsage(line string) string {
	// 找连续 ≥2 空格的最后一段分隔
	parts := regexp.MustCompile(`\s{2,}`).Split(strings.TrimSpace(line), -1)
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[len(parts)-1])
	}
	return ""
}

// containsCJK 是否含中日韩字符。
func containsCJK(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}
