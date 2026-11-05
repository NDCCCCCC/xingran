package client

// M8 覆盖率补强：upload.go 的 DoUpload/DoDownload 单测。
//
// 背景：tests/ 包下的同名用例属于外部包测试，go test 的 -coverprofile
// 只统计被测包自身的测试，因此 upload.go 在 CI 的 cli-coverage.out 里
// 全部是 count=0，触发 GOV-03 diff-coverage 门禁误伤。
// 本文件与 upload.go 同包，确保覆盖记录计入 internal/client。

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/auth"
)

var xlsxMagic = []byte{0x50, 0x4B, 0x03, 0x04}

type covMockAuth struct {
	token     string
	nextToken string
	refreshed int
}

func (m *covMockAuth) Login(context.Context, string, string, auth.CaptchaInput) (*auth.TokenPair, error) {
	return nil, errors.New("not implemented")
}
func (m *covMockAuth) Refresh(context.Context) (*auth.TokenPair, error) {
	if m.nextToken == "" {
		return nil, errors.New("refresh failed")
	}
	m.token = m.nextToken
	m.refreshed++
	return &auth.TokenPair{AccessToken: m.token, RefreshToken: "rt", ExpiresIn: 300}, nil
}
func (m *covMockAuth) GetAccessToken(context.Context) (string, error) { return m.token, nil }
func (m *covMockAuth) Logout(context.Context) error                   { return nil }

func covXlsx(t *testing.T, tail []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "data.xlsx")
	if err := os.WriteFile(p, append(append([]byte{}, xlsxMagic...), tail...), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func covClient(baseURL string, hc *http.Client, au Authenticator) Client {
	return New(Options{BaseURL: baseURL, HTTPClient: hc, Authenticator: au})
}

func TestCovUploadMultipart(t *testing.T) {
	var ct, au, field, filename string
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct, au = r.Header.Get("Content-Type"), r.Header.Get("Authorization")
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			http.Error(w, "bad", 400)
			return
		}
		f, h, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "no file", 400)
			return
		}
		field, filename = "file", h.Filename
		buf := make([]byte, 512)
		n, _ := f.Read(buf)
		body = buf[:n]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"inserted":1,"updated":0,"failed":0,"errors":[]}}`))
	}))
	defer srv.Close()

	c := covClient(srv.URL, srv.Client(), &covMockAuth{token: "tok-1"})
	resp, err := c.DoUpload(context.Background(), "/ops/workstation/import", "file", covXlsx(t, []byte("wb")), map[string]string{"note": "x"})
	if err != nil || !resp.OK() {
		t.Fatalf("upload failed: %v resp=%+v", err, resp)
	}
	if !strings.HasPrefix(ct, "multipart/form-data") || au != "Bearer tok-1" || field != "file" || filename != "data.xlsx" {
		t.Fatalf("shape wrong: ct=%q au=%q field=%q fn=%q", ct, au, field, filename)
	}
	if !bytes.HasPrefix(body, xlsxMagic) || string(body[4:]) != "wb" {
		t.Fatalf("body wrong: %q", body)
	}
}

func TestCovUploadBadExtension(t *testing.T) {
	c := covClient("http://unused", http.DefaultClient, &covMockAuth{token: "t"})
	p := filepath.Join(t.TempDir(), "bad.csv")
	_ = os.WriteFile(p, []byte("x"), 0o600)
	if _, err := c.DoUpload(context.Background(), "/ops/workstation/import", "", p, nil); err == nil {
		t.Fatal("应拒绝 .csv")
	}
}

func TestCovUploadMissingFile(t *testing.T) {
	c := covClient("http://unused", http.DefaultClient, &covMockAuth{token: "t"})
	if _, err := c.DoUpload(context.Background(), "/x", "file", filepath.Join(t.TempDir(), "nope.xlsx"), nil); err == nil {
		t.Fatal("不存在的文件应报错")
	}
}

func TestCovUpload401Refresh(t *testing.T) {
	au := &covMockAuth{token: "stale", nextToken: "fresh"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer stale" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"code":401,"message":"unauthorized"}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{}}`))
	}))
	defer srv.Close()
	c := covClient(srv.URL, srv.Client(), au)
	resp, err := c.DoUpload(context.Background(), "/ops/workstation/import", "file", covXlsx(t, nil), nil)
	if err != nil || !resp.OK() || au.refreshed != 1 {
		t.Fatalf("refresh retry failed: err=%v resp=%+v refreshed=%d", err, resp, au.refreshed)
	}
}

func TestCovUploadImportErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"data":{"inserted":0,"updated":0,"failed":2,"errors":[{"row":2,"message":"楼宇不存在"}]}}`))
	}))
	defer srv.Close()
	c := covClient(srv.URL, srv.Client(), &covMockAuth{token: "t"})
	resp, err := c.DoUpload(context.Background(), "/ops/workstation/import", "file", covXlsx(t, nil), nil)
	if err != nil || !resp.OK() {
		t.Fatalf("upload failed: %v", err)
	}
	var d struct {
		Failed int `json:"failed"`
		Errors []struct {
			Row int `json:"row"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(resp.Data, &d); err != nil || d.Failed != 2 || len(d.Errors) != 1 {
		t.Fatalf("import errors parse: %+v err=%v", d, err)
	}
}

func TestCovDownloadToFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="tpl.xlsx"`)
		_, _ = w.Write(append(append([]byte{}, xlsxMagic...), []byte("sheet")...))
	}))
	defer srv.Close()
	out := filepath.Join(t.TempDir(), "out.xlsx")
	c := covClient(srv.URL, srv.Client(), &covMockAuth{token: "t"})
	saved, err := c.DoDownload(context.Background(), "/ops/workstation/template", "", out, "GET")
	if err != nil || saved != out {
		t.Fatalf("download failed: %v saved=%q", err, saved)
	}
	data, _ := os.ReadFile(out)
	if !bytes.HasPrefix(data, xlsxMagic) || string(data[4:]) != "sheet" {
		t.Fatalf("content wrong: %q", data)
	}
}

func TestCovDownloadDefaultFilename(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="auto.xlsx"`)
		_, _ = w.Write(xlsxMagic)
	}))
	defer srv.Close()
	dir := t.TempDir()
	c := covClient(srv.URL, srv.Client(), &covMockAuth{token: "t"})
	saved, err := c.DoDownload(context.Background(), "/ops/workstation/template", "", "", "GET")
	if err != nil || !strings.HasSuffix(saved, "auto.xlsx") {
		t.Fatalf("default filename failed: %v saved=%q", err, saved)
	}
	_ = dir
}

func TestCovDownloadErrorNotWritten(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":500,"message":"内部错误","data":null}`))
	}))
	defer srv.Close()
	out := filepath.Join(t.TempDir(), "nope.xlsx")
	c := covClient(srv.URL, srv.Client(), &covMockAuth{token: "t"})
	if _, err := c.DoDownload(context.Background(), "/ops/workstation/template", "", out, "GET"); err == nil {
		t.Fatal("JSON 错误应返回 error")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("错误响应不得落盘: %s", out)
	}
}

func TestCovDownload401Refresh(t *testing.T) {
	au := &covMockAuth{token: "stale", nextToken: "fresh"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer stale" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"code":401,"message":"unauthorized"}`))
			return
		}
		_, _ = w.Write(xlsxMagic)
	}))
	defer srv.Close()
	out := filepath.Join(t.TempDir(), "t.xlsx")
	c := covClient(srv.URL, srv.Client(), au)
	if _, err := c.DoDownload(context.Background(), "/ops/workstation/template", "", out, "GET"); err != nil {
		t.Fatalf("refresh retry download failed: %v", err)
	}
	if au.refreshed != 1 {
		t.Fatalf("应刷新一次: %d", au.refreshed)
	}
}

func TestCovDownloadExportPost(t *testing.T) {
	var method, ct, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, ct = r.Method, r.Header.Get("Content-Type")
		buf := make([]byte, 256)
		n, _ := r.Body.Read(buf)
		body = string(buf[:n])
		_, _ = w.Write(xlsxMagic)
	}))
	defer srv.Close()
	out := filepath.Join(t.TempDir(), "e.xlsx")
	c := covClient(srv.URL, srv.Client(), &covMockAuth{token: "t"})
	if _, err := c.DoDownload(context.Background(), "/ops/workstation/export", "{}", out, "POST"); err != nil {
		t.Fatalf("export failed: %v", err)
	}
	if method != "POST" || ct != "application/json" || body != "{}" {
		t.Fatalf("export shape: %s %s %q", method, ct, body)
	}
}
