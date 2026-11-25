package rpa

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ============================================================================
// Phase 156 / REC-03：动作枚举三方一致性测试（Go ↔ 前端枚举 ↔ 前端动词表）
//
// 背景：
//
//	Go 侧唯一真相源是 rpa-worker/internal/types/types.go 的 ActionType 常量
//	（26 个）。历史上前端另有一份手写的 ActionType 联合类型（只有 13 个，
//	且混入了 Go 侧根本不存在的 press），与 worker 形成「双源漂移」：
//	  - worker 能执行 26 种动作，前端下拉只有 9 种；
//	  - 前端能表达 press，worker 永远执行不了。
//
// 本测试直接读三方源文件做**双向**比对，任何一方多/少都报错：
//	  1. Go 侧 ActionType 常量集合
//	  2. 前端共享枚举 RPA_ACTION_TYPES（xingran-react-frontend/src/types/rpa.ts）
//	  3. 前端动词表 HUMAN_VERBS 的键集（wizard/stepLabels.ts）
//
// 另加后端镜像 workerActionTypes 的键集合，凑成四方一致。
//
// 读文件失败一律 t.Skipf（不误红）：例如只跑前端仓库、或 CI 分片未检出
// rpa-worker 目录时，跳过即可，绝不能因为「文件不在」就判定枚举不一致。
// ============================================================================

// repoRoot 从 internal/services/rpa 上溯到仓库根。
func repoRoot() string {
	return filepath.Join("..", "..", "..")
}

// readSourceFile 读一个源文件，失败时跳过测试（而不是判红）。
func readSourceFile(t *testing.T, path string) string {
	t.Helper()
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("读不到 %s（%v），跳过跨语言一致性校验", path, err)
	}
	return string(buf)
}

// goActionTypes 抽取 rpa-worker/internal/types/types.go 里的 ActionType 常量。
//
// 匹配形如：ActionClick ActionType = "click"（与既有
// TestWorkerActionTypes_MirrorsWorkerTypesGo 同一手法）。
func goActionTypes(t *testing.T) []string {
	t.Helper()
	root := repoRoot()
	path := filepath.Join(root, "rpa-worker", "internal", "types", "types.go")
	src := readSourceFile(t, path)

	re := regexp.MustCompile(`Action\w+\s+ActionType\s*=\s*"([^"]+)"`)
	matches := re.FindAllStringSubmatch(src, -1)
	if len(matches) == 0 {
		t.Fatalf("未能从 %s 抽取到任何动作常量（正则失配？）", path)
	}

	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
}

// frontendActionTypes 抽取前端共享枚举 RPA_ACTION_TYPES 的字符串字面量。
func frontendActionTypes(t *testing.T) []string {
	t.Helper()
	root := repoRoot()
	path := filepath.Join(root, "xingran-react-frontend", "src", "types", "rpa.ts")
	src := readSourceFile(t, path)

	block := regexp.MustCompile(`RPA_ACTION_TYPES\s*=\s*\[([^\]]*)\]`).FindStringSubmatch(src)
	if len(block) < 2 {
		t.Fatalf("未能从 %s 抽取到 RPA_ACTION_TYPES 数组（结构变了？）", path)
	}

	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(block[1], -1)
	if len(matches) == 0 {
		t.Fatalf("RPA_ACTION_TYPES 数组里没有解析到字符串字面量：%s", path)
	}

	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
}

// frontendHumanVerbKeys 抽取前端动词表 HUMAN_VERBS 的键集。
func frontendHumanVerbKeys(t *testing.T) []string {
	t.Helper()
	root := repoRoot()
	path := filepath.Join(root, "xingran-react-frontend", "src", "pages", "operations", "rpa", "wizard", "stepLabels.ts")
	src := readSourceFile(t, path)

	block := regexp.MustCompile(`HUMAN_VERBS[^=]*=\s*\{([^}]*)\}`).FindStringSubmatch(src)
	if len(block) < 2 {
		t.Fatalf("未能从 %s 抽取到 HUMAN_VERBS 对象（结构变了？）", path)
	}

	re := regexp.MustCompile(`(?m)^\s*([A-Za-z_][A-Za-z0-9_]*)\s*:`)
	matches := re.FindAllStringSubmatch(block[1], -1)
	if len(matches) == 0 {
		t.Fatalf("HUMAN_VERBS 里没有解析到任何键：%s", path)
	}

	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
}

// setDiff 返回「在 a 中但不在 b 中」的元素（已排序）。
func setDiff(a, b []string) []string {
	inB := map[string]bool{}
	for _, v := range b {
		inB[v] = true
	}
	var out []string
	for _, v := range a {
		if !inB[v] {
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

// assertSameSet 双向比对两个集合，任何一方多/少都报错。
func assertSameSet(t *testing.T, name string, a, b []string) {
	t.Helper()
	onlyA := setDiff(a, b)
	onlyB := setDiff(b, a)
	if len(onlyA) == 0 && len(onlyB) == 0 {
		return
	}
	if len(onlyA) > 0 {
		t.Errorf("%s：只在左集合多出 %v", name, onlyA)
	}
	if len(onlyB) > 0 {
		t.Errorf("%s：只在右集合多出 %v", name, onlyB)
	}
}

// TestActionTypeConsistency_GoFrontendStepLabels 三方枚举一致性（缺一即红）。
//
// 这是 REC-03 的核心机械保障：
//   - Go 侧 worker types.go（唯一真相源）
//   - 前端共享枚举 RPA_ACTION_TYPES（src/types/rpa.ts）
//   - 前端动词表 HUMAN_VERBS 键集（wizard/stepLabels.ts）
//
// 任一方新增/删除动作而忘记同步其它两方，这里立刻红。
func TestActionTypeConsistency_GoFrontendStepLabels(t *testing.T) {
	goSet := goActionTypes(t)
	feSet := frontendActionTypes(t)
	verbSet := frontendHumanVerbKeys(t)

	assertSameSet(t, "Go 动作常量 vs 前端共享枚举 RPA_ACTION_TYPES", goSet, feSet)
	assertSameSet(t, "Go 动作常量 vs 前端动词表 HUMAN_VERBS 键集", goSet, verbSet)

	t.Logf("动作枚举三方对齐通过：Go %d 个 / 前端枚举 %d 个 / 前端动词表 %d 个",
		len(goSet), len(feSet), len(verbSet))
}

// TestActionTypeConsistency_FrontendHasNoExtraAction 前端不得引入 Go 侧没有的动作。
//
// 单独拎出来是因为它对应历史上最典型的漂移：前端有 press、Go 侧没有。
func TestActionTypeConsistency_FrontendHasNoExtraAction(t *testing.T) {
	goSet := goActionTypes(t)
	feSet := frontendActionTypes(t)
	verbSet := frontendHumanVerbKeys(t)

	for _, extra := range setDiff(feSet, goSet) {
		t.Errorf("前端共享枚举引入了 Go 侧不存在的动作 %q（worker 永远执行不了）", extra)
	}
	for _, extra := range setDiff(verbSet, goSet) {
		t.Errorf("前端动词表出现了 Go 侧不存在的动作 %q", extra)
	}
	for _, missing := range setDiff(goSet, feSet) {
		t.Errorf("前端共享枚举缺少 Go 侧动作 %q（worker 能执行但前端表达不了）", missing)
	}
	for _, missing := range setDiff(goSet, verbSet) {
		t.Errorf("前端动词表缺少 Go 侧动作 %q（会显示英文）", missing)
	}
}

// TestActionTypeConsistency_WorkerActionTypesMirror 后端镜像与 Go 侧一致。
//
// workerActionTypes 是后端对 worker 枚举的镜像（ai_action_validator.go），
// 它必须与 Go 真相源、与前端两侧都同源 —— 三方 + 镜像 = 四方一致。
func TestActionTypeConsistency_WorkerActionTypesMirror(t *testing.T) {
	goSet := goActionTypes(t)
	feSet := frontendActionTypes(t)
	verbSet := frontendHumanVerbKeys(t)

	mirror := make([]string, 0, len(workerActionTypes))
	for k := range workerActionTypes {
		mirror = append(mirror, k)
	}

	assertSameSet(t, "Go 动作常量 vs 后端镜像 workerActionTypes", goSet, mirror)
	assertSameSet(t, "Go 动作常量 vs 前端共享枚举 RPA_ACTION_TYPES", goSet, feSet)
	assertSameSet(t, "Go 动作常量 vs 前端动词表 HUMAN_VERBS 键集", goSet, verbSet)
}

// TestActionTypeConsistency_GoSetIsExactly26 锁定动作全集的规模。
//
// 26 是当前 worker 的动作全集规模；worker 增删动作时这个数字应当跟着变，
// 届时同步更新本断言与前端枚举即可（断言的作用是让「规模意外变化」显性化，
// 而不是阻止 worker 演进）。
func TestActionTypeConsistency_GoSetIsExactly26(t *testing.T) {
	goSet := goActionTypes(t)
	unique := map[string]bool{}
	for _, v := range goSet {
		unique[v] = true
	}
	if len(unique) != 26 {
		t.Errorf("Go 侧动作常量应为 26 个（去重后），实际 %d 个", len(unique))
	}
	if len(goSet) != len(unique) {
		t.Errorf("Go 侧动作常量存在重复项：共 %d 条、去重后 %d 条", len(goSet), len(unique))
	}
}

// TestActionTypeConsistency_FrontendEnumHasNoDuplicates 前端枚举不得有重复项。
func TestActionTypeConsistency_FrontendEnumHasNoDuplicates(t *testing.T) {
	feSet := frontendActionTypes(t)
	seen := map[string]bool{}
	for _, v := range feSet {
		if seen[v] {
			t.Errorf("前端共享枚举出现重复动作 %q", v)
		}
		seen[v] = true
	}
	if len(feSet) != len(seen) {
		t.Errorf("前端共享枚举共 %d 项、去重后 %d 项", len(feSet), len(seen))
	}
	if len(feSet) != 26 {
		t.Errorf("前端共享枚举应为 26 项，实际 %d 项", len(feSet))
	}
}

// TestActionTypeConsistency_FrontendLabelsAreChinese 前端两张映射表的中文名不含英文。
//
// 中文名里出现 ASCII 字母会直接暴露给普通员工（错误信息/下拉项），
// 违反 NOCODE 的「零技术术语」约束。
func TestActionTypeConsistency_FrontendLabelsAreChinese(t *testing.T) {
	latin := regexp.MustCompile(`[A-Za-z]`)

	root := repoRoot()
	textMapPath := filepath.Join(root, "xingran-react-frontend", "src", "pages", "operations", "rpa", "constants.tsx")
	src := readSourceFile(t, textMapPath)

	block := regexp.MustCompile(`ACTION_TYPE_TEXT_MAP[^=]*=\s*\{([^}]*)\}`).FindStringSubmatch(src)
	if len(block) < 2 {
		t.Fatalf("未能从 %s 抽取到 ACTION_TYPE_TEXT_MAP", textMapPath)
	}
	entryRe := regexp.MustCompile(`(?m)^\s*([A-Za-z_][A-Za-z0-9_]*)\s*:\s*"([^"]*)"`)
	for _, m := range entryRe.FindAllStringSubmatch(block[1], -1) {
		key, label := m[1], m[2]
		if strings.TrimSpace(label) == "" {
			t.Errorf("动作 %q 缺少中文名（会显示空白）", key)
		}
		if latin.MatchString(label) {
			t.Errorf("动作 %q 的中文名含 ASCII 字母: %q", key, label)
		}
	}
}
