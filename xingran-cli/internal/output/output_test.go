package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/xingran-next/ruoyi-go-backend/xingran-cli/internal/client"
)

func resp(code int, msg, reqID string) client.Response {
	return client.Response{Code: code, Message: msg, RequestID: reqID}
}

func TestExitFromCode(t *testing.T) {
	cases := []struct {
		code, httpStatus, want int
	}{
		{0, 200, ExitOK},
		{401, 401, ExitAuth},
		{1007, 200, ExitAuth},
		{400, 400, ExitParam},
		{1001, 200, ExitParam},
		{403, 403, ExitForbidden},
		{500, 500, ExitOther},
		{42, 200, ExitOther},
		{0, 0, ExitOK},
		// 裸 HTTP 状态（非标准包装）
		{401, 0, ExitAuth},
		{403, 0, ExitForbidden},
		{422, 0, ExitParam},
	}
	for _, c := range cases {
		if got := ExitFromCode(c.code, c.httpStatus); got != c.want {
			t.Errorf("ExitFromCode(%d,%d) = %d, want %d", c.code, c.httpStatus, got, c.want)
		}
	}
}

func TestEmitSuccessJSONStdout(t *testing.T) {
	var out, errOut bytes.Buffer
	result := map[string]any{"list": []int{1, 2}, "total": 2}
	code := Emit(result, nil, &out, &errOut, false)
	if code != ExitOK {
		t.Fatalf("code = %d", code)
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr 应为空: %q", errOut.String())
	}
	var m map[string]any
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatalf("stdout 非 JSON: %q", out.String())
	}
	if m["total"].(float64) != 2 {
		t.Fatalf("m = %v", m)
	}
}

func TestEmitAPIErrorJSONStderr(t *testing.T) {
	var out, errOut bytes.Buffer
	err := &client.APIError{Resp: resp(1007, "token invalid", "req-9"), HTTPStatus: 200}
	code := Emit(nil, err, &out, &errOut, false)
	if code != ExitAuth {
		t.Fatalf("code = %d, want 2", code)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout 应为空: %q", out.String())
	}
	var ej ErrorJSON
	if err := json.Unmarshal(errOut.Bytes(), &ej); err != nil {
		t.Fatalf("stderr 非 JSON: %q", errOut.String())
	}
	if ej.Code != 1007 || ej.ExitCode != ExitAuth || ej.RequestID != "req-9" {
		t.Fatalf("ej = %+v", ej)
	}
}

func TestEmitTransportError(t *testing.T) {
	var out, errOut bytes.Buffer
	err := &client.TransportError{Method: "POST", URL: "http://x", Err: errors.New("dial refused")}
	code := Emit(nil, err, &out, &errOut, false)
	if code != ExitNetwork {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(errOut.String(), `"exit_code":1`) {
		t.Fatalf("errOut = %q", errOut.String())
	}
}

func TestEmitLocalExitError(t *testing.T) {
	var out, errOut bytes.Buffer
	err := NewExitError(ExitParam, "JSON body 解析失败", errors.New("bad json"))
	code := Emit(nil, err, &out, &errOut, false)
	if code != ExitParam {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(errOut.String(), "bad json") {
		t.Fatalf("errOut = %q", errOut.String())
	}
}

func TestEmitUnknownErrorIsOther(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Emit(nil, errors.New("boom"), &out, &errOut, false)
	if code != ExitOther {
		t.Fatalf("code = %d", code)
	}
}

func TestEmitTTYHumanReadable(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Emit(map[string]any{"ok": true}, nil, &out, &errOut, true)
	if code != ExitOK {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(out.String(), `"ok": true`) {
		t.Fatalf("TTY 输出应缩进: %q", out.String())
	}
	// TTY 错误也是人类可读（不强制 JSON）
	errOut.Reset()
	_ = Emit(nil, errors.New("boom"), &out, &errOut, true)
	if strings.Contains(errOut.String(), `"exit_code"`) {
		t.Fatalf("TTY 错误不应输出 JSON: %q", errOut.String())
	}
}

func TestEmitNilResult(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := Emit(nil, nil, &out, &errOut, false); code != ExitOK {
		t.Fatalf("code = %d", code)
	}
	if strings.TrimSpace(out.String()) != "{}" {
		t.Fatalf("out = %q", out.String())
	}
}
