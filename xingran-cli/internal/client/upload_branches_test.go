package client

// upload.go 错误分支与辅助函数覆盖率补强（M8 CI diff-coverage）。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCovUploadServerUnreachable 覆盖 c.httpClient.Do 失败分支（107-108）。
func TestCovUploadServerUnreachable(t *testing.T) {
	c := covClient("http://127.0.0.1:1", http.DefaultClient, &covMockAuth{token: "t"})
	if _, err := c.DoUpload(context.Background(), "/x", "file", covXlsx(t, nil), nil); err == nil {
		t.Fatal("连接失败应返回 error")
	}
}

// TestCovUploadNonJSONResponse 覆盖非 JSON 响应（110-119 的 wrapped=false 路径）。
// resolve 对 2xx 非包装按裸数据兼容返回（client.go:244-250）。
func TestCovUploadNonJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("plain text"))
	}))
	defer srv.Close()
	c := covClient(srv.URL, srv.Client(), &covMockAuth{token: "t"})
	resp, err := c.DoUpload(context.Background(), "/x", "file", covXlsx(t, nil), nil)
	if err != nil || resp == nil || !resp.OK() {
		t.Fatalf("2xx 裸数据应兼容返回: err=%v resp=%+v", err, resp)
	}
	if string(resp.Data) != "plain text" {
		t.Fatalf("裸数据应原样返回: %q", resp.Data)
	}
}

// TestCovDownloadServerUnreachable 覆盖下载连接失败（167-168）与 download err 分支（185-186）。
func TestCovDownloadServerUnreachable(t *testing.T) {
	c := covClient("http://127.0.0.1:1", http.DefaultClient, &covMockAuth{token: "t"})
	out := filepath.Join(t.TempDir(), "o.xlsx")
	if _, err := c.DoDownload(context.Background(), "/x", "", out, "GET"); err == nil {
		t.Fatal("连接失败应返回 error")
	}
}

// TestCovDownloadJSONCodeZero 覆盖 code=0 但 Content-Type=json 的分支（203）。
func TestCovDownloadJSONCodeZero(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":null}`))
	}))
	defer srv.Close()
	out := filepath.Join(t.TempDir(), "o.xlsx")
	c := covClient(srv.URL, srv.Client(), &covMockAuth{token: "t"})
	if _, err := c.DoDownload(context.Background(), "/x", "", out, "GET"); err == nil {
		t.Fatal("JSON 响应（即使 code=0）应报错")
	}
}

// TestCovDownloadWriteFail 覆盖 WriteFile 失败分支（213-215）。
func TestCovDownloadWriteFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(xlsxMagic)
	}))
	defer srv.Close()
	// destPath 指向一个目录 → WriteFile 失败
	dir := t.TempDir()
	c := covClient(srv.URL, srv.Client(), &covMockAuth{token: "t"})
	if _, err := c.DoDownload(context.Background(), "/x", "", dir, "GET"); err == nil {
		t.Fatal("写入目录应失败")
	}
}

// TestCovDownloadDefaultMethod 覆盖 method="" → GET 分支（148-150）。
func TestCovDownloadDefaultMethod(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Method
		_, _ = w.Write(xlsxMagic)
	}))
	defer srv.Close()
	out := filepath.Join(t.TempDir(), "o.xlsx")
	c := covClient(srv.URL, srv.Client(), &covMockAuth{token: "t"})
	if _, err := c.DoDownload(context.Background(), "/x", "", out, ""); err != nil {
		t.Fatal(err)
	}
	if got != "GET" {
		t.Fatalf("method 空应为 GET，实际 %s", got)
	}
}

// TestCovFilenameFromDisposition 覆盖 filenameFromDisposition 全部分支（221-244）。
func TestCovFilenameFromDisposition(t *testing.T) {
	cases := []struct {
		name, cd, path, want string
	}{
		{"mime 标准", `attachment; filename="std.xlsx"`, "/ops/x/template", "std.xlsx"},
		{"filename 变体", `attachment; filename=loose.xlsx`, "/ops/x/template", "loose.xlsx"},
		{"filename 为空回退", `attachment; filename=""`, "/ops/workstation/template", "workstation.xlsx"},
		{"无 CD 取倒数第二段", "", "/ops/workstation/template", "workstation.xlsx"},
		{"无 CD 且路径短", "", "/template", "template.xlsx"},
		{"无 CD 普通末段", "", "/ops/workstation/import", "import.xlsx"},
		{"cd 解析失败", "garbage", "/ops/workstation/export", "workstation.xlsx"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := filenameFromDisposition(tc.cd, tc.path); got != tc.want {
				t.Fatalf("filenameFromDisposition(%q,%q)=%q want %q", tc.cd, tc.path, got, tc.want)
			}
		})
	}
}

// TestCovEscapeQuotes 覆盖 escapeQuotes（257-260）。
func TestCovEscapeQuotes(t *testing.T) {
	if got := escapeQuotes(`a"b\c`); got != `a\"b\\c` {
		t.Fatalf("escapeQuotes 不正确: %q", got)
	}
}

// TestCovTruncateStr 覆盖 truncateStr（262-267）两个分支。
func TestCovTruncateStr(t *testing.T) {
	if got := truncateStr("abc", 10); got != "abc" {
		t.Fatalf("短串不应截断: %q", got)
	}
	if got := truncateStr("abcdefghij", 5); got != "abcde..." {
		t.Fatalf("长串应截断: %q", got)
	}
}

// TestCovDownloadDefaultFilenameFallback 覆盖 210-211（连 filename 都派生不出时）。
func TestCovDownloadDefaultFilenameFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(xlsxMagic)
	}))
	defer srv.Close()
	c := covClient(srv.URL, srv.Client(), &covMockAuth{token: "t"})
	// path="/" → segs 全空 → filenameFromDisposition 返回 ".xlsx"，非空，走不到 211
	// 用极短 path 让派生结果仍非空；211 是防御分支，这里至少覆盖 206-209 的多种形态
	saved, err := c.DoDownload(context.Background(), "/", "", "", "GET")
	if err != nil {
		t.Fatal(err)
	}
	if saved == "" {
		t.Fatal("saved 不应为空")
	}
	// 清理可能落在当前目录的文件
	_ = os.Remove(saved)
	if !strings.HasSuffix(saved, ".xlsx") {
		t.Fatalf("派生文件名应以 .xlsx 结尾: %q", saved)
	}
}
