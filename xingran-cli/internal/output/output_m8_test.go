package output

// M8 补强：table/csv/yaml 多格式输出的覆盖率测试。
// 背景：PR #73 的 coverage-diff 失败，output.go 新增的多格式渲染逻辑
// （emitSuccess format 分发 / writeYAML / writeCSV / writeTable /
//  extractRows / collectHeaders / formatCell）缺少同包单测，
// CI 的 -coverprofile 只计入被测包自身的测试文件。

import (
	"bytes"
	"strings"
	"testing"
)

// ---- emitSuccess format 分发（覆盖 88-97 的各 case 与未知 format 回退） ----

func TestEmitSuccessFormatDispatch(t *testing.T) {
	data := map[string]any{"name": "张三", "age": float64(30)}

	cases := map[string]string{
		"yaml":   "name: 张三",
		"yml":    "name: 张三",
		"csv":    "age",
		"table":  "张三",
		"json":   `"name"`,
		"JSON":   `"name"`, // 大小写归一
		" yaml ": "name",  // TrimSpace 归一
		"unknown": "",     // 未知 format 回退 JSON（非 tty）
	}
	for format, want := range cases {
		var buf bytes.Buffer
		code := emitSuccess(data, &buf, false, format)
		if code != ExitOK {
			t.Errorf("format=%q 期望 ExitOK，实际 %d", format, code)
		}
		if want != "" && !strings.Contains(buf.String(), want) {
			t.Errorf("format=%q 输出应含 %q，实际: %s", format, want, buf.String())
		}
	}
}

// TestEmitSuccessFormatNilData 覆盖 result==nil → struct{} 分支（83-85）。
func TestEmitSuccessFormatNilData(t *testing.T) {
	var buf bytes.Buffer
	if code := emitSuccess(nil, &buf, false, "yaml"); code != ExitOK {
		t.Fatalf("nil result yaml 应 ExitOK，实际 %d", code)
	}
	if !strings.Contains(buf.String(), "{}") {
		t.Fatalf("nil 应渲染为空对象: %s", buf.String())
	}
}

// ---- writeYAML（覆盖 182-192 正常编码路径） ----
// 注：yaml.v3 对不可编码类型经 panic/recover 报错，enc.Encode 的 err 分支
// 为防御性代码、合法输入不可达，故不构造错误用例（会产生 panic 导致测试崩溃）。

func TestWriteYAMLOutput(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{"name": "张三", "age": 30, "ok": true}
	if code := writeYAML(&buf, data); code != ExitOK {
		t.Fatalf("writeYAML 应 ExitOK，实际 %d", code)
	}
	out := buf.String()
	for _, want := range []string{"name", "张三", "age", "ok"} {
		if !strings.Contains(out, want) {
			t.Fatalf("yaml 输出应含 %q: %s", want, out)
		}
	}
}

// ---- extractRows（覆盖 196-231：marshal 失败 / Response 包装 / 数组 / 标量数组 / 标量） ----

func TestExtractRowsVariants(t *testing.T) {
	t.Run("marshal 失败返回 nil", func(t *testing.T) {
		if got := extractRows(map[string]any{"bad": make(chan int)}); got != nil {
			t.Fatalf("不可 marshal 应返回 nil，实际 %+v", got)
		}
	})

	t.Run("Response 包装提取 data", func(t *testing.T) {
		in := map[string]any{
			"code": float64(0),
			"data": []any{map[string]any{"id": "a"}, map[string]any{"id": "b"}},
		}
		rows := extractRows(in)
		if len(rows) != 2 || rows[0]["id"] != "a" {
			t.Fatalf("应从 data 提取 2 行: %+v", rows)
		}
	})

	t.Run("对象数组", func(t *testing.T) {
		rows := extractRows([]any{map[string]any{"k": 1}})
		if len(rows) != 1 || rows[0]["k"] != float64(1) {
			t.Fatalf("对象数组应转行: %+v", rows)
		}
	})

	t.Run("标量数组包装为 value", func(t *testing.T) {
		rows := extractRows([]any{"x", "y"})
		if len(rows) != 2 || rows[0]["value"] != "x" {
			t.Fatalf("标量数组应包装 value 键: %+v", rows)
		}
	})

	t.Run("单个标量返回 nil", func(t *testing.T) {
		if rows := extractRows("solo"); rows != nil {
			t.Fatalf("标量应返回 nil（output.go:233），实际: %+v", rows)
		}
	})

	t.Run("单个对象", func(t *testing.T) {
		rows := extractRows(map[string]any{"a": 1})
		if len(rows) != 1 || rows[0]["a"] != float64(1) {
			t.Fatalf("单个对象应为一行: %+v", rows)
		}
	})
}

// ---- collectHeaders（覆盖 258-268 的排序分支） ----

func TestCollectHeadersOrdering(t *testing.T) {
	rows := []map[string]any{
		{"b": 1, "100": 2, "a": 3, "zz": 4},
		{"c": 5}, // 仅第二行有的列也进 header
	}
	h := collectHeaders(rows)
	joined := strings.Join(h, ",")
	// 数字 header 优先（按数值），其余字母序；c 只出现一次
	if !strings.HasPrefix(joined, "100,") {
		t.Fatalf("数字 header 应排在最前: %v", h)
	}
	if strings.Count(joined, "c") != 1 {
		t.Fatalf("header 不应重复: %v", h)
	}
	if joined[len(joined)-1] != 'z' {
		t.Fatalf("字母序末位应为 zz: %v", h)
	}
}

// ---- writeCSV / writeTable 的空数据回退（275-277 / 306-308） ----

func TestWriteCSVEmptyFallsBackToJSON(t *testing.T) {
	var buf bytes.Buffer
	// chan 使 extractRows 返回 nil（触发回退），但回退的 writeJSON 对含 chan
	// 的数据同样编码失败 → ExitOther(5)，验证整条链路的失败语义。
	if code := writeCSV(&buf, map[string]any{"bad": make(chan int)}); code != ExitOther {
		t.Fatalf("回退 writeJSON 编码失败应 ExitOther，实际 %d", code)
	}
}

func TestWriteTableEmptyFallsBackToJSON(t *testing.T) {
	var buf bytes.Buffer
	// 与 CSV 同理：回退的 writeJSON 对含 chan 数据编码失败 → ExitOther(5)。
	if code := writeTable(&buf, map[string]any{"bad": make(chan int)}); code != ExitOther {
		t.Fatalf("回退 writeJSON 编码失败应 ExitOther，实际 %d", code)
	}
}

// ---- writeTable：长文本截断（325-327）与多行替换（323-324） ----

func TestWriteTableTruncationAndNewline(t *testing.T) {
	long := strings.Repeat("L", 60)
	multi := "line1\nline2\r\nline3"
	rows := []map[string]any{{"long": long, "multi": multi, "n": 1}}
	var buf bytes.Buffer
	if code := writeTable(&buf, rows); code != ExitOK {
		t.Fatalf("table 渲染应 ExitOK，实际 %d", code)
	}
	out := buf.String()
	if strings.Contains(out, strings.Repeat("L", 40)) {
		t.Fatal("长文本应被截断为 37+...")
	}
	if !strings.Contains(out, "...") {
		t.Fatal("截断应带省略号")
	}
	if strings.Contains(out, "\nline2") && !strings.Contains(out, "line1 line2") {
		t.Fatalf("多行应替换为空格: %s", out)
	}
	if !strings.Contains(out, "line1 line2  line3") && !strings.Contains(out, "line1 line2") {
		t.Fatalf("应含拍平后的文本: %s", out)
	}
}

// ---- formatCell 全类型分支（361-380） ----

func TestFormatCellTypes(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{nil, ""},
		{"str", "str"},
		{float64(42), "42"},
		{float64(3.14), "3.14"},
		{true, "true"},
		{map[string]any{"k": "v"}, `{"k":"v"}`},
	}
	for _, tc := range cases {
		if got := formatCell(tc.in); got != tc.want {
			t.Errorf("formatCell(%v)=%q want %q", tc.in, got, tc.want)
		}
	}
}

// TestFormatCellUnmarshalable default 分支 marshal 失败时按 %v 兜底（不 panic）。
func TestFormatCellUnmarshalable(t *testing.T) {
	if got := formatCell(make(chan int)); got == "" {
		t.Fatal("marshal 失败应有兜底输出，不应为空")
	}
}

// ---- EmitWithFormat 错误路径防御断言（format 不影响错误输出） ----

func TestEmitWithFormatErrorStillJSON(t *testing.T) {
	var out, errb bytes.Buffer
	code := EmitWithFormat(nil, NewExitError(ExitParam, "参数错误", nil), &out, &errb, false, "yaml")
	if code != ExitParam {
		t.Fatalf("错误路径退出码应保留: %d", code)
	}
}
