package rpa

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ============================================================================
// Phase 156 / NOCODE-03：AI 动作枚举校验测试
//
// 关键断言（防回归）：
//  1. 合法动作全通过；非法动作被拒且**逐条给出可读中文原因**。
//  2. 校验白名单与 rpa-worker 的 types.go **同源** —— 直接读 worker 源文件
//     抽取 ActionType 常量，与本文件的 workerActionTypes 对比。
//     这条测试是「156 不得作为动作全集第三写方」的机械保障：154 新增动作后
//     忘了同步，这里立刻红。
//  3. AI 未启用的哨兵错误可被 IsAIDisabled 识别（前端降级判定的依据）。
// ============================================================================

// TestValidateAIActions_AllValid 全部合法动作通过。
func TestValidateAIActions_AllValid(t *testing.T) {
	actions := []json.RawMessage{
		json.RawMessage(`{"type":"navigate","value":"https://x"}`),
		json.RawMessage(`{"type":"fill","selector":"#a","value":"1"}`),
		json.RawMessage(`{"type":"click","selector":"#b"}`),
	}
	valid, err := ValidateAIActions(actions)
	if err != nil {
		t.Fatalf("合法动作不应报错: %v", err)
	}
	if valid != 3 {
		t.Fatalf("合法动作数应为 3，实际 %d", valid)
	}
}

// TestValidateAIActions_RejectsUnknownType 未知动作类型被拒且原因可读。
func TestValidateAIActions_RejectsUnknownType(t *testing.T) {
	actions := []json.RawMessage{
		json.RawMessage(`{"type":"click","selector":"#a"}`),
		json.RawMessage(`{"type":"teleport","selector":"#b"}`),
		json.RawMessage(`{"type":"fill","selector":"#c","value":"v"}`),
	}
	valid, err := ValidateAIActions(actions)
	if err == nil {
		t.Fatal("含非法动作时必须报错")
	}
	if valid != 2 {
		t.Fatalf("合法动作数应为 2，实际 %d", valid)
	}

	var ve *ActionValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("错误类型应为 *ActionValidationError，实际 %T", err)
	}
	if len(ve.Invalid) != 1 {
		t.Fatalf("应恰有 1 条非法动作，实际 %d", len(ve.Invalid))
	}
	if ve.Invalid[0].Index != 2 {
		t.Errorf("非法动作下标应为 2（1-based），实际 %d", ve.Invalid[0].Index)
	}
	if ve.Invalid[0].Type != "teleport" {
		t.Errorf("应记录原始动作类型 teleport，实际 %q", ve.Invalid[0].Type)
	}
	if ve.Total != 3 {
		t.Errorf("总数应为 3，实际 %d", ve.Total)
	}

	msg := err.Error()
	// 可读性断言：面向一般员工，必须说清「第几步」「哪个类型」「怎么办」
	for _, want := range []string{"第 2 步", "teleport", "无法执行", "录制"} {
		if !strings.Contains(msg, want) {
			t.Errorf("错误信息应包含 %q，实际: %s", want, msg)
		}
	}
	// 零技术术语断言：不得出现枚举/字段/JSON 这类词
	for _, banned := range []string{"enum", "unmarshal", "ActionType", "unsupported action type", "nil"} {
		if strings.Contains(msg, banned) {
			t.Errorf("错误信息泄漏技术术语 %q: %s", banned, msg)
		}
	}
}

// TestValidateAIActions_RejectsMissingType 缺 type 字段被拒。
func TestValidateAIActions_RejectsMissingType(t *testing.T) {
	_, err := ValidateAIActions([]json.RawMessage{json.RawMessage(`{"selector":"#a"}`)})
	if err == nil {
		t.Fatal("缺 type 必须报错")
	}
	if !strings.Contains(err.Error(), "缺少动作类型") {
		t.Errorf("原因应说明缺少动作类型，实际: %v", err)
	}
}

// TestValidateAIActions_RejectsMalformed 非法 JSON 被拒且不 panic。
func TestValidateAIActions_RejectsMalformed(t *testing.T) {
	_, err := ValidateAIActions([]json.RawMessage{json.RawMessage(`{not json}`)})
	if err == nil {
		t.Fatal("非法 JSON 必须报错")
	}
	if !strings.Contains(err.Error(), "格式") {
		t.Errorf("原因应说明格式问题，实际: %v", err)
	}
}

// TestValidateAIActions_Empty 空列表不算错误（AI 可能合法地返回 0 条）。
func TestValidateAIActions_Empty(t *testing.T) {
	if _, err := ValidateAIActions(nil); err != nil {
		t.Fatalf("空动作列表不应报错: %v", err)
	}
}

// TestValidateAIActions_AllInvalid 全部非法时逐条列出。
func TestValidateAIActions_AllInvalid(t *testing.T) {
	actions := []json.RawMessage{
		json.RawMessage(`{"type":"a"}`),
		json.RawMessage(`{"type":"b"}`),
	}
	valid, err := ValidateAIActions(actions)
	if valid != 0 {
		t.Fatalf("全部非法时合法数应为 0，实际 %d", valid)
	}
	var ve *ActionValidationError
	if !errors.As(err, &ve) || len(ve.Invalid) != 2 {
		t.Fatalf("应列出 2 条非法动作，实际: %v", err)
	}
}

// TestWorkerActionTypes_MirrorsWorkerTypesGo 校验白名单与 worker types.go 同源。
//
// 这是「156 不得作为动作全集第三写方」的机械保障。做法：直接读
// rpa-worker/internal/types/types.go，抽取全部 ActionXxx ActionType = "yyy"
// 常量，与 workerActionTypes 的键集合双向比对。
func TestWorkerActionTypes_MirrorsWorkerTypesGo(t *testing.T) {
	// 从 internal/services/rpa 上溯到仓库根
	root := filepath.Join("..", "..", "..")
	path := filepath.Join(root, "rpa-worker", "internal", "types", "types.go")
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("读不到 worker types.go（%v），跳过同源校验", err)
	}

	// 匹配形如：ActionClick ActionType = "click"
	re := regexp.MustCompile(`Action\w+\s+ActionType\s*=\s*"([^"]+)"`)
	matches := re.FindAllStringSubmatch(string(buf), -1)
	if len(matches) == 0 {
		t.Fatal("未能从 worker types.go 抽取到任何动作常量（正则失配？）")
	}

	workerSet := map[string]bool{}
	for _, m := range matches {
		workerSet[m[1]] = true
	}

	// 双向比对：后端白名单不得有 worker 没有的动作（会生成不可执行脚本），
	// 也不得漏掉 worker 已有的动作（AI 生成了却报不支持）。
	for t2 := range workerSet {
		if _, ok := workerActionTypes[t2]; !ok {
			t.Errorf("worker 有动作 %q，后端校验白名单缺失（AI 生成的这类动作会被误拒）", t2)
		}
	}
	for b := range workerActionTypes {
		if !workerSet[b] {
			t.Errorf("后端白名单有 %q，但 worker types.go 不存在该动作（会产出不可执行脚本）", b)
		}
	}

	t.Logf("动作枚举同源校验通过：worker %d 个，后端镜像 %d 个", len(workerSet), len(workerActionTypes))
}

// TestErrAIDisabled_Detectable AI 未启用可被精确识别（前端降级依据）。
func TestErrAIDisabled_Detectable(t *testing.T) {
	if !IsAIDisabled(ErrAIDisabled) {
		t.Fatal("ErrAIDisabled 自身应被识别")
	}
	if IsAIDisabled(nil) {
		t.Fatal("nil 不应被识别为 AI 未启用")
	}
	if IsAIDisabled(errors.New("网络超时")) {
		t.Fatal("普通错误不应被误判为 AI 未启用")
	}
}

// TestWorkerActionTypes_HasChineseLabels 每个动作都有中文名（错误信息不暴露英文）。
func TestWorkerActionTypes_HasChineseLabels(t *testing.T) {
	for t2, label := range workerActionTypes {
		if strings.TrimSpace(label) == "" {
			t.Errorf("动作 %q 缺少中文名（错误信息会暴露英文枚举）", t2)
		}
		// 中文名不应含 ASCII 字母（"执行脚本" 这类纯中文）
		if regexp.MustCompile(`[A-Za-z]`).MatchString(label) {
			t.Errorf("动作 %q 的中文名含英文字符: %q", t2, label)
		}
	}
}
